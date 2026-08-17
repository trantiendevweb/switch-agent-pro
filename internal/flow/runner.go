package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
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
	// RunAgents bật n agent với prompt, ĐỢI xong, trả về lỗi nếu có.
	RunAgents(ctx context.Context, profile, prompt string, copies int, worktree bool) error
}

// Runner thực thi một flow.
type Runner struct {
	DB    *store.DB
	Bus   *events.Bus
	Agent AgentRunner

	// MaxParallel là trần số agent chạy cùng lúc, lấy từ policy của dự án.
	MaxParallel int
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

// execute là vòng chạy chính: đi theo thứ tự topo, mỗi bước chỉ chạy khi mọi
// bước nó phụ thuộc đã `done`.
//
// v1 chạy TUẦN TỰ theo thứ tự topo. Song song thật sự nằm bên trong bước agent
// (copies), đủ cho các flow mẫu; chạy nhiều bước cùng lúc để sau, khi có nhu
// cầu thật — đơn giản thì ít lỗi hơn.
func (r *Runner) execute(ctx context.Context, runID int64, f Flow, vars map[string]string) (Result, error) {
	order, err := Order(f)
	if err != nil {
		_ = r.DB.SetRunState(runID, store.RunFailed)
		return Result{RunID: runID, State: store.RunFailed}, err
	}

	done, err := r.DB.Steps(runID)
	if err != nil {
		return Result{}, err
	}
	isDone := func(id string) bool {
		s, ok := done[id]
		return ok && (s.State == store.StepDone || s.State == store.StepSkipped)
	}

	for _, s := range order {
		if ctx.Err() != nil {
			_ = r.DB.SetRunState(runID, store.RunCanceled)
			return Result{RunID: runID, State: store.RunCanceled}, ctx.Err()
		}
		if isDone(s.ID) {
			continue // đã chạy ở lần trước
		}

		// Phụ thuộc chưa xong thì KHÔNG được chạy. Đây cũng chính là chỗ khiến
		// approval gate không thể bị vượt mặt: bước approve chỉ chuyển sang
		// `done` bằng hành động của con người (Approve), không có nhánh code
		// nào trong bộ thực thi tự đánh dấu nó.
		blocked := ""
		for _, n := range s.Needs {
			if !isDone(n) {
				blocked = n
				break
			}
		}
		if blocked != "" {
			// Bị chặn bởi một bước đang chờ duyệt → dừng cả lần chạy ở đây.
			if st, ok := done[blocked]; ok && st.State == store.StepWaiting {
				_ = r.DB.SetRunState(runID, store.RunWaiting)
				return Result{RunID: runID, State: store.RunWaiting, Waiting: blocked}, nil
			}
			_ = r.DB.SetStep(runID, s.ID, store.StepSkipped, "bỏ qua vì "+blocked+" không xong", 0)
			done[s.ID] = store.StepRun{StepID: s.ID, State: store.StepSkipped}
			r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID,
				SessionID: runID, Msg: "bỏ qua — phụ thuộc " + blocked + " không xong"})
			continue
		}

		// Bước approve: ghi trạng thái chờ rồi dừng, trả quyền quyết cho con người.
		if s.Type == TypeApprove {
			_ = r.DB.SetStep(runID, s.ID, store.StepWaiting, s.Message, 0)
			_ = r.DB.SetRunState(runID, store.RunWaiting)
			r.Bus.Publish(events.Event{
				Type: events.FlowWaiting, Addr: f.Name + "." + s.ID, SessionID: runID,
				Msg:    "chờ duyệt: " + s.Message,
				Detail: map[string]string{"run": fmt.Sprint(runID), "step": s.ID},
			})
			return Result{RunID: runID, State: store.RunWaiting, Waiting: s.ID}, nil
		}

		state, msg := r.runStep(ctx, runID, f, s, vars)
		done[s.ID] = store.StepRun{StepID: s.ID, State: state, Msg: msg}

		if state == store.StepFailed {
			switch s.OnFailure {
			case OnFailContinue:
				r.Bus.Warnf("%s.%s hỏng nhưng on_failure=continue — đi tiếp", f.Name, s.ID)
			case OnFailFallback:
				r.Bus.Warnf("%s.%s hỏng — chuyển sang bước %s", f.Name, s.ID, s.Fallback)
				// bước fallback nằm trong cùng DAG, sẽ tới lượt theo thứ tự
			default: // OnFailStop
				_ = r.DB.SetRunState(runID, store.RunFailed)
				r.Bus.Publish(events.Event{Type: events.FlowFailed, Addr: f.Name, SessionID: runID,
					Msg: fmt.Sprintf("dừng ở bước %s: %s", s.ID, msg)})
				return Result{RunID: runID, State: store.RunFailed}, nil
			}
		}
	}

	_ = r.DB.SetRunState(runID, store.RunDone)
	r.Bus.Publish(events.Event{Type: events.FlowDone, Addr: f.Name, SessionID: runID,
		Msg: fmt.Sprintf("xong #%d", runID)})
	return Result{RunID: runID, State: store.RunDone}, nil
}

// runStep chạy một bước, có timeout và retry.
func (r *Runner) runStep(ctx context.Context, runID int64, f Flow, s Step, vars map[string]string) (string, string) {
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
		err := r.do(stepCtx, s, vars)
		if cancel != nil {
			cancel()
		}

		if err == nil {
			_ = r.DB.SetStep(runID, s.ID, store.StepDone, "", attempt)
			r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID,
				SessionID: runID, Msg: "xong"})
			return store.StepDone, ""
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
	msg := lastErr.Error()
	_ = r.DB.SetStep(runID, s.ID, store.StepFailed, msg, tries)
	r.Bus.Failuref("%s.%s: %s", f.Name, s.ID, msg)
	return store.StepFailed, msg
}

// do thực thi đúng một lần, theo loại node.
func (r *Runner) do(ctx context.Context, s Step, vars map[string]string) error {
	switch s.Type {
	case TypeAgent, TypeReview:
		if r.Agent == nil {
			return fmt.Errorf("không có bộ chạy agent")
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

	case TypeShell:
		if len(s.Run) == 0 {
			return fmt.Errorf("thiếu run")
		}
		// argv, KHÔNG qua shell — flow là file người ta gửi cho nhau được.
		args := make([]string, len(s.Run))
		for i, a := range s.Run {
			args[i] = Expand(a, vars)
		}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			line := strings.TrimSpace(lastLine(string(out)))
			if line != "" {
				return fmt.Errorf("%v — %s", err, line)
			}
			return err
		}
		return nil

	case TypeNotify:
		r.Bus.Infof("%s", Expand(s.Message, vars))
		return nil

	default:
		return fmt.Errorf("type %q chưa chạy được ở bản này", s.Type)
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
