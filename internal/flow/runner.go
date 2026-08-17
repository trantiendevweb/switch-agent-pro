package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// AgentRunner là thứ bộ thực thi cần để chạy một bước `agent`.
//
// Cố ý là interface chứ không gọi thẳng gói fleet: nhờ vậy test chạy được flow
// thật mà không cần khởi động agent nào, và sau này đường API (model route) cắm
// vào cùng chỗ.
type AgentRunner interface {
	// RunAgents bật n agent với prompt, ĐỢI xong, trả về kết quả gộp (để bước
	// sau dùng qua {{steps.x.output}}) và lỗi nếu có.
	RunAgents(ctx context.Context, profile, prompt string, copies int, worktree bool) (string, error)
}

// Runner thực thi một flow.
type Runner struct {
	DB    *store.DB
	Bus   *events.Bus
	Agent AgentRunner

	// MaxParallel là trần số bước/agent chạy cùng lúc, lấy từ policy của dự án.
	MaxParallel int

	// Commands là các lệnh khai trong .sagent/project.toml (test, lint, build…),
	// để node `test`/`lint` không phải lặp lại lệnh trong từng flow.
	Commands map[string][]string
}

// Result tóm tắt một lần chạy.
type Result struct {
	RunID   int64
	State   string // store.RunDone | RunFailed | RunWaiting
	Waiting string // id bước đang chờ duyệt (nếu State = RunWaiting)
}

// Start mở một lần chạy mới rồi thực thi.
func (r *Runner) Start(ctx context.Context, f Flow, dir string, vars map[string]string) (Result, error) {
	merged := map[string]string{}
	for k, v := range f.Vars {
		merged[k] = v
	}
	for k, v := range vars { // tham số dòng lệnh đè giá trị mặc định
		merged[k] = v
	}
	raw, _ := json.Marshal(merged)

	runID, err := r.DB.CreateRun(f.Name, dir, string(raw))
	if err != nil {
		return Result{}, err
	}
	r.Bus.Publish(events.Event{
		Type: events.FlowStarted, Addr: f.Name, SessionID: runID,
		Msg: fmt.Sprintf("bắt đầu #%d — %d bước", runID, len(f.Steps)),
	})
	return r.execute(ctx, runID, f, merged)
}

// Resume chạy tiếp một lần chạy đang dở (sau khi duyệt, hoặc sau khi máy khởi
// động lại). Bước đã `done` được bỏ qua — đó là lý do trạng thái nằm ở SQLite.
func (r *Runner) Resume(ctx context.Context, runID int64, f Flow) (Result, error) {
	run, err := r.DB.GetRun(runID)
	if err != nil {
		return Result{}, fmt.Errorf("không có lần chạy #%d", runID)
	}
	if run.State == store.RunDone || run.State == store.RunCanceled {
		return Result{RunID: runID, State: run.State}, nil
	}
	vars := map[string]string{}
	if run.Vars != "" {
		_ = json.Unmarshal([]byte(run.Vars), &vars)
	}
	_ = r.DB.SetRunState(runID, store.RunRunning)
	return r.execute(ctx, runID, f, vars)
}

// execute là vòng chạy chính, chạy theo ĐỢT:
//
//	lặp { tìm mọi bước đã sẵn sàng → chạy CHÚNG SONG SONG → chờ hết đợt }
//
// Bước "sẵn sàng" = mọi bước nó phụ thuộc đã xong. Nhờ vậy các nhánh độc lập
// (chạy test + lint + build) diễn ra cùng lúc thay vì xếp hàng.
//
// Approval gate vẫn nguyên vẹn: bước approve không bao giờ được chạy trong đợt,
// nó chỉ chuyển sang `done` bằng hành động của con người.
func (r *Runner) execute(ctx context.Context, runID int64, f Flow, vars map[string]string) (Result, error) {
	if _, err := Order(f); err != nil { // vẫn kiểm chu trình trước khi chạy
		_ = r.DB.SetRunState(runID, store.RunFailed)
		return Result{RunID: runID, State: store.RunFailed}, err
	}

	saved, err := r.DB.Steps(runID)
	if err != nil {
		return Result{}, err
	}

	st := &runState{
		states:  map[string]string{},
		outputs: map[string]string{},
	}
	for id, s := range saved {
		st.states[id] = s.State
		if s.Output != "" {
			st.outputs[id] = s.Output
		}
	}

	byID := map[string]Step{}
	for _, s := range f.Steps {
		byID[s.ID] = s
	}

	for {
		if ctx.Err() != nil {
			_ = r.DB.SetRunState(runID, store.RunCanceled)
			return Result{RunID: runID, State: store.RunCanceled}, ctx.Err()
		}

		ready, waiting := st.readySteps(f.Steps)
		if len(ready) == 0 {
			// Không còn gì chạy được. Nếu vì đang chờ duyệt thì dừng ở đó.
			if waiting != "" {
				_ = r.DB.SetRunState(runID, store.RunWaiting)
				return Result{RunID: runID, State: store.RunWaiting, Waiting: waiting}, nil
			}
			break
		}

		// Bước approve không chạy — nó dựng rào rồi trả quyền cho con người.
		var work []Step
		for _, s := range ready {
			if s.Type == TypeApprove {
				continue
			}
			work = append(work, s)
		}

		if len(work) > 0 {
			if failed := r.runWave(ctx, runID, f, work, vars, st); failed != "" {
				_ = r.DB.SetRunState(runID, store.RunFailed)
				r.Bus.Publish(events.Event{Type: events.FlowFailed, Addr: f.Name, SessionID: runID,
					Msg: fmt.Sprintf("dừng ở bước %s", failed)})
				return Result{RunID: runID, State: store.RunFailed}, nil
			}
			continue // xong đợt, tính lại xem bước nào sẵn sàng
		}

		// Cả đợt chỉ còn approve: dựng rào ở cái đầu tiên rồi dừng.
		s := ready[0]
		_ = r.DB.SetStep(runID, s.ID, store.StepWaiting, s.Message, 0)
		st.set(s.ID, store.StepWaiting, "")
		_ = r.DB.SetRunState(runID, store.RunWaiting)
		r.Bus.Publish(events.Event{
			Type: events.FlowWaiting, Addr: f.Name + "." + s.ID, SessionID: runID,
			Msg:    "chờ duyệt: " + s.Message,
			Detail: map[string]string{"run": fmt.Sprint(runID), "step": s.ID},
		})
		return Result{RunID: runID, State: store.RunWaiting, Waiting: s.ID}, nil
	}

	_ = r.DB.SetRunState(runID, store.RunDone)
	r.Bus.Publish(events.Event{Type: events.FlowDone, Addr: f.Name, SessionID: runID,
		Msg: fmt.Sprintf("xong #%d", runID)})
	return Result{RunID: runID, State: store.RunDone}, nil
}

// runState giữ trạng thái/kết quả trong lúc chạy. Có khoá vì một đợt chạy nhiều
// bước song song, tất cả cùng ghi vào đây.
type runState struct {
	mu      sync.Mutex
	states  map[string]string
	outputs map[string]string
}

func (s *runState) set(id, state, out string) {
	s.mu.Lock()
	s.states[id] = state
	if out != "" {
		s.outputs[id] = out
	}
	s.mu.Unlock()
}

func (s *runState) snapshot() (map[string]string, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := make(map[string]string, len(s.states))
	b := make(map[string]string, len(s.outputs))
	for k, v := range s.states {
		a[k] = v
	}
	for k, v := range s.outputs {
		b[k] = v
	}
	return a, b
}

func (s *runState) finished(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.states[id]
	return v == store.StepDone || v == store.StepSkipped
}

func (s *runState) state(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.states[id]
}

// readySteps tìm các bước có thể chạy ngay. waiting là id bước approve đang chặn
// (nếu có) để lời gọi biết vì sao hết việc.
func (s *runState) readySteps(steps []Step) (ready []Step, waiting string) {
	for _, step := range steps {
		if s.state(step.ID) != "" && (s.finished(step.ID) || s.state(step.ID) == store.StepFailed) {
			continue
		}
		if s.state(step.ID) == store.StepWaiting {
			waiting = step.ID
			continue
		}
		ok := true
		for _, n := range step.Needs {
			if !s.finished(n) {
				ok = false
				if s.state(n) == store.StepWaiting {
					waiting = n
				}
				break
			}
		}
		if ok {
			ready = append(ready, step)
		}
	}
	return ready, waiting
}

// runWave chạy một đợt bước song song, có trần đồng thời. Trả về id bước làm cả
// flow phải dừng (rỗng nếu không có).
func (r *Runner) runWave(ctx context.Context, runID int64, f Flow, work []Step,
	vars map[string]string, st *runState) string {

	limit := r.MaxParallel
	if limit < 1 {
		limit = 4
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	stopAt := ""

	if len(work) > 1 {
		r.Bus.Infof("chạy song song %d bước: %s", len(work), stepIDs(work))
	}

	for _, s := range work {
		s := s
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// Điều kiện `when`: không thoả thì bỏ qua, bước sau vẫn chạy.
			states, outs := st.snapshot()
			if s.When != "" {
				ok, err := Eval(s.When, Ctx{Vars: vars, States: states, Outputs: outs})
				if err != nil {
					_ = r.DB.SetStep(runID, s.ID, store.StepFailed, "điều kiện sai: "+err.Error(), 0)
					st.set(s.ID, store.StepFailed, "")
					r.Bus.Failuref("%s.%s điều kiện sai: %v", f.Name, s.ID, err)
					mu.Lock()
					if stopAt == "" && s.OnFailure != OnFailContinue {
						stopAt = s.ID
					}
					mu.Unlock()
					return
				}
				if !ok {
					_ = r.DB.SetStep(runID, s.ID, store.StepSkipped, "điều kiện không thoả: "+s.When, 0)
					st.set(s.ID, store.StepSkipped, "")
					r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID,
						SessionID: runID, Msg: "bỏ qua — " + s.When})
					return
				}
			}

			state, msg, out := r.runStep(ctx, runID, f, s, vars, outs)
			st.set(s.ID, state, out)

			if state == store.StepFailed {
				switch s.OnFailure {
				case OnFailContinue:
					r.Bus.Warnf("%s.%s hỏng nhưng on_failure=continue — đi tiếp", f.Name, s.ID)
				case OnFailFallback:
					r.Bus.Warnf("%s.%s hỏng — bước %s sẽ chạy thay", f.Name, s.ID, s.Fallback)
				default:
					mu.Lock()
					if stopAt == "" {
						stopAt = s.ID + ": " + msg
					}
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	return stopAt
}

func stepIDs(ss []Step) string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.ID
	}
	return strings.Join(out, ", ")
}

// runStep chạy một bước, có timeout và retry.
func (r *Runner) runStep(ctx context.Context, runID int64, f Flow, s Step,
	vars map[string]string, outputs map[string]string) (state, msg, output string) {
	// Bước sau dùng được kết quả bước trước.
	env := WithOutputs(vars, outputs)
	tries := s.Retry + 1
	if tries < 1 {
		tries = 1
	}
	var lastErr error
	for attempt := 1; attempt <= tries; attempt++ {
		_ = r.DB.SetStep(runID, s.ID, store.StepRunning, "", attempt)
		r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID, SessionID: runID,
			Msg: fmt.Sprintf("chạy [%s] lần %d/%d", s.Type, attempt, tries)})

		stepCtx := ctx
		var cancel context.CancelFunc
		if s.TimeoutSec > 0 {
			stepCtx, cancel = context.WithTimeout(ctx, time.Duration(s.TimeoutSec)*time.Second)
		}
		out, err := r.do(stepCtx, s, env)
		if cancel != nil {
			cancel()
		}

		if err == nil {
			_ = r.DB.SetStep(runID, s.ID, store.StepDone, "", attempt)
			if out != "" {
				_ = r.DB.SetStepOutput(runID, s.ID, out)
			}
			r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID,
				SessionID: runID, Msg: "xong"})
			return store.StepDone, "", out
		}
		lastErr = err
		if attempt < tries {
			// lùi dần: 2s, 4s, 6s… đủ để thứ tạm thời tự khỏi
			wait := time.Duration(attempt*2) * time.Second
			r.Bus.Warnf("%s.%s hỏng (%v) — thử lại sau %s", f.Name, s.ID, err, wait)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				break
			}
		}
	}
	emsg := lastErr.Error()
	_ = r.DB.SetStep(runID, s.ID, store.StepFailed, emsg, tries)
	r.Bus.Failuref("%s.%s: %s", f.Name, s.ID, emsg)
	return store.StepFailed, emsg, ""
}

// do thực thi đúng một lần, theo loại node.
func (r *Runner) do(ctx context.Context, s Step, vars map[string]string) (string, error) {
	switch s.Type {
	case TypeAgent, TypeReview:
		if r.Agent == nil {
			return "", fmt.Errorf("không có bộ chạy agent")
		}
		n := s.Copies
		if n < 1 {
			n = 1
		}
		if r.MaxParallel > 0 && n > r.MaxParallel {
			r.Bus.Warnf("copies=%d vượt trần %d của dự án — hạ xuống", n, r.MaxParallel)
			n = r.MaxParallel
		}
		return r.Agent.RunAgents(ctx, s.Profile, Expand(s.Prompt, vars), n, s.Worktree)

	case TypeShell, TypeTest, TypeLint:
		argv := s.Run
		if len(argv) == 0 {
			// test/lint không cần khai báo lệnh: lấy từ .sagent/project.toml
			switch s.Type {
			case TypeTest:
				argv = r.Commands["test"]
			case TypeLint:
				argv = r.Commands["lint"]
			}
			if len(argv) == 0 {
				return "", fmt.Errorf("bước %s cần `run`, hoặc khai `commands.%s` trong .sagent/project.toml", s.Type, s.Type)
			}
		}
		s.Run = argv
		// argv, KHÔNG qua shell — flow là file người ta gửi cho nhau được.
		args := make([]string, len(s.Run))
		for i, a := range s.Run {
			args[i] = Expand(a, vars)
		}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		raw, err := cmd.CombinedOutput()
		if err != nil {
			line := strings.TrimSpace(lastLine(string(raw)))
			if line != "" {
				return "", fmt.Errorf("%v — %s", err, line)
			}
			return "", err
		}
		return strings.TrimRight(string(raw), "\r\n"), nil

	case TypeNotify:
		m := Expand(s.Message, vars)
		r.Bus.Infof("%s", m)
		return m, nil

	default:
		return "", fmt.Errorf("type %q chưa chạy được ở bản này", s.Type)
	}
}

// Approve đánh dấu một bước approve là ĐÃ DUYỆT.
//
// Đây là hàm DUY NHẤT chuyển một bước approve sang `done`. Bộ thực thi không có
// đường nào tự làm việc đó — nhờ vậy "approval không thể bị bỏ qua" là tính chất
// của kiến trúc chứ không phải một cái cờ ai cũng bật được.
func (r *Runner) Approve(runID int64, stepID string, by string) error {
	steps, err := r.DB.Steps(runID)
	if err != nil {
		return err
	}
	st, ok := steps[stepID]
	if !ok || st.State != store.StepWaiting {
		return fmt.Errorf("bước %q không ở trạng thái chờ duyệt", stepID)
	}
	if err := r.DB.SetStep(runID, stepID, store.StepDone, "duyệt bởi "+by, 0); err != nil {
		return err
	}
	r.Bus.Publish(events.Event{Type: events.FlowApproved, SessionID: runID,
		Addr: stepID, Msg: "đã duyệt bởi " + by})
	return nil
}

// Reject từ chối một bước approve — cả lần chạy dừng lại.
func (r *Runner) Reject(runID int64, stepID, by string) error {
	steps, err := r.DB.Steps(runID)
	if err != nil {
		return err
	}
	st, ok := steps[stepID]
	if !ok || st.State != store.StepWaiting {
		return fmt.Errorf("bước %q không ở trạng thái chờ duyệt", stepID)
	}
	if err := r.DB.SetStep(runID, stepID, store.StepFailed, "từ chối bởi "+by, 0); err != nil {
		return err
	}
	if err := r.DB.SetRunState(runID, store.RunCanceled); err != nil {
		return err
	}
	r.Bus.Publish(events.Event{Type: events.FlowRejected, SessionID: runID,
		Addr: stepID, Msg: "bị từ chối bởi " + by})
	return nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	return lines[len(lines)-1]
}
