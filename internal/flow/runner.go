package flow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// AgentRunner là thứ bộ thực thi cần để chạy một bước `agent`.
//
// Cố ý là interface chứ không gọi thẳng gói fleet: nhờ vậy test chạy được flow
// thật mà không cần khởi động agent nào, và sau này đường API (model route) cắm
// vào cùng chỗ.
type AgentRunner interface {
	// RunAgents bật n agent với prompt, ĐỢI xong, trả về kết quả và lỗi nếu có.
	// Kết quả mang cả output (cho {{steps.x.output}}) lẫn chi phí đo được — chi
	// phí phải đi CÙNG output vì cả hai sinh ra trong cùng một lượt chạy; tách
	// ra hai đường thì bước nào tính tiền bước nấy sẽ lệch.
	RunAgents(ctx context.Context, profile, prompt string, copies int, worktree, tuDuyetQuyen bool) (KetQuaAgent, error)
}

// KetQuaAgent là những gì một lượt chạy agent trả về cho bộ thực thi flow.
type KetQuaAgent struct {
	Output    string  // kết quả cho bước sau dùng
	ChiPhiUSD float64 // 0 nếu provider không cho biết chi phí
	TokenVao  int
	TokenRa   int
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
