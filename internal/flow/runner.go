package flow

import (
	"context"
	"encoding/json"
	"fmt"
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
	// RunAgents bật n agent với prompt, ĐỢI xong, trả về kết quả và lỗi nếu có.
	// Kết quả mang cả output (cho {{steps.x.output}}) lẫn chi phí đo được — chi
	// phí phải đi CÙNG output vì cả hai sinh ra trong cùng một lượt chạy; tách
	// ra hai đường thì bước nào tính tiền bước nấy sẽ lệch.
	RunAgents(ctx context.Context, profile, model, prompt string, copies int, worktree, tuDuyetQuyen bool) (KetQuaAgent, error)
}

// ModelRunner gọi THẲNG model API — đường thứ hai của dự án, không qua CLI agent.
//
// Tách khỏi AgentRunner vì hai đường khác nhau về mọi mặt đáng kể: đường agent
// tiêu hạn mức thuê bao và chạy được lệnh trên máy; đường API tiêu tiền theo
// token và chỉ trả về chữ. Gộp chung một interface thì bước gọi API sẽ mang theo
// những tham số vô nghĩa với nó (worktree, tự duyệt quyền, copies).
//
// route rỗng = dùng `default_route` rồi tới route dự phòng — cùng luật với
// `sagent api "câu hỏi"`.
type ModelRunner interface {
	GoiModel(ctx context.Context, route, prompt string) (KetQuaAgent, error)
}

// PluginRunner chạy node `plugin` — đường THỨ BA: không phải agent, không phải
// model API, mà một EXECUTABLE do người dùng cài, nói JSON-RPC trên stdio.
//
// Tách khỏi hai interface trên vì tham số của nó không giao nhau với chúng: một
// plugin không có tài khoản, không có model, không có worktree, không có quyền
// tự duyệt tool. Nhét chung một interface là bắt mỗi bên mang theo một nắm tham
// số vô nghĩa với mình.
//
// Cố ý trả về CHUỖI chứ không phải KetQuaAgent: plugin chạy trên máy người dùng
// và không tiêu token của ai, nên nó không có gì để nói trong các trường chi phí
// — trả về KetQuaAgent là mời gọi sổ chi phí ghi những con số 0 mà không ai
// phân biệt được với "chưa đo".
//
// Cài đặt thật: internal/plugin.BoChay. Gói flow KHÔNG import gói đó — khai
// interface ở đây thì test của flow chạy được mà không phải biên dịch plugin nào.
type PluginRunner interface {
	GoiPlugin(ctx context.Context, ten, vao string, thamSo map[string]string) (string, error)
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

	// Model chạy node `model`. nil = chưa cắm, và node `model` sẽ báo lỗi rõ
	// ràng thay vì im lặng bỏ qua.
	Model ModelRunner

	// Plugin chạy node `plugin`. nil = chưa cắm — cùng luật với Model: bước báo
	// lỗi nói rõ là chưa cắm, chứ không bỏ qua rồi trả về chuỗi rỗng.
	Plugin PluginRunner

	// Route chạy node `route` — chọn đường API còn sống rồi chuyền tên cho bước
	// sau. nil = chưa cắm, cùng luật với Model và Plugin.
	//
	// Tách khỏi Model dù cùng đi tới một gói: gọi model TIÊU TIỀN theo token và
	// trả về chữ, còn chọn đường chỉ hỏi thăm sức khoẻ (không tốn token) và trả
	// về một cái tên. Xem RouteChon trong route.go.
	Route RouteChon

	// MaxParallel là trần số bước/agent chạy cùng lúc, lấy từ policy của dự án.
	MaxParallel int

	// Commands là các lệnh khai trong .sagent/project.toml (test, lint, build…),
	// để node `test`/`lint` không phải lặp lại lệnh trong từng flow.
	Commands map[string][]string

	// DefaultProfile là tài khoản dùng cho bước agent không khai `profile`.
	//
	// Bộ thực thi KHÔNG dùng nó để chạy (việc đó là của AgentRunner) — nó chỉ để
	// event báo hỏng nói được "tài khoản nào". Tin báo "bước x hỏng" mà không
	// kèm tài khoản thì người đọc vẫn phải mở máy lên tra, tức là tin đó chưa
	// làm xong việc của nó.
	DefaultProfile string
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

	// Dọn artifact cũ ở đầu MỖI lượt chạy mới, không phải bằng một lệnh riêng.
	// Một nút dọn rác phải nhớ bấm là một nút không ai bấm. Chỉ đụng tới lượt
	// chạy ĐÃ KẾT THÚC và cũ hơn ArtifactGiuLai — xem DonArtifact.
	if n := DonArtifact(r.DB, ArtifactGiuLai, time.Now()); n > 0 {
		r.Bus.Infof("dọn artifact của %d lượt chạy cũ (quá %s)", n, ArtifactGiuLai)
	}

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

	// Lượt này có thể là lượt CHẠY LẠI của một lượt đã dừng ở rào duyệt sau khi
	// một bước chạy thay đã làm xong việc. Bảng biến trong bộ nhớ thì mất, nhưng
	// hai dòng sổ ("bước A failed", "bước chạy thay của A done") đủ để suy ra —
	// xem fallback.go. Gọi ở đây, ngay sau khi nạp sổ và TRƯỚC vòng chạy, để
	// bước sau tìm thấy kết quả ở đúng chỗ nó đang tìm.
	ganKetQuaThayThe(f, st)

	// Bước nào bị LOẠI khỏi lịch chạy thường — bước gỡ lại (`compensate`) và
	// bước chạy thay (`fallback`). Tính một lần cho cả lượt, ở MỘT chỗ duy nhất
	// dùng chung với `flow show` và bảng chạy khan.
	ngoaiLich := BuocNgoaiLichThuong(f)

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
		//
		// Bước GỠ LẠI và bước CHẠY THAY cũng không chạy ở đây, và lý do khác
		// hẳn: chúng thường không có `needs` nào, tức là một GỐC của DAG, nên để
		// yên thì chúng chạy ngay đợt đầu của MỌI lượt chạy — gỡ một việc chưa
		// ai làm, hoặc chạy thay cho một bước chưa kịp hỏng. Xem compensate.go
		// và fallback.go.
		var work, choDuyet []Step
		daDanhDauNgoaiLich := false
		for _, s := range ready {
			if s.Type == TypeApprove {
				choDuyet = append(choDuyet, s)
				continue
			}
			if ly, ngoai := ngoaiLich[s.ID]; ngoai {
				daDanhDauNgoaiLich = true
				// Ghi `skipped` kèm lời giải thích chứ không để trống: một ô
				// trống trên bảng đọc là "chưa tới lượt", còn đây là "sẽ không
				// chạy trừ khi có chuyện".
				_ = r.DB.SetStep(runID, s.ID, store.StepSkipped, ly, 0)
				st.set(s.ID, store.StepSkipped, "")
				r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID,
					SessionID: runID, Msg: ly})
				continue
			}
			work = append(work, s)
		}

		if len(work) > 0 {
			if failed, ly := r.runWave(ctx, runID, f, work, vars, st); failed != "" {
				_ = r.DB.SetRunState(runID, store.RunFailed)
				msg := fmt.Sprintf("dừng ở bước %s", failed)
				if ly != "" {
					msg += ": " + ly
				}
				r.Bus.Publish(events.Event{Type: events.FlowFailed, Addr: f.Name, SessionID: runID,
					Msg: msg,
					// Có cấu trúc chứ không chỉ một câu chữ: mặt nào muốn mở đúng
					// bước hỏng (workflow board) hay nhắn đúng tên bước (Telegram)
					// đều phải tự tách chuỗi nếu thiếu chỗ này — và tách chuỗi thì
					// sớm muộn cũng sai.
					Detail: map[string]string{
						"flow": f.Name, "run": fmt.Sprint(runID), "step": failed, "ly_do": ly,
					}})
				return Result{RunID: runID, State: store.RunFailed}, nil
			}
			continue // xong đợt, tính lại xem bước nào sẵn sàng
		}

		// Đợt này không có gì để chạy và cũng không có rào duyệt nào: chỉ vừa
		// đánh dấu vài bước NGOÀI LỊCH THƯỜNG là `skipped`. Tính lại đợt — chúng
		// sẽ không còn nổi lên nữa vì `skipped` tính là đã xong.
		if len(choDuyet) == 0 {
			if daDanhDauNgoaiLich {
				continue
			}
			break // không còn gì chạy được, và không phải vì chờ ai
		}

		// Cả đợt chỉ còn approve: dựng rào ở cái đầu tiên rồi dừng.
		s := choDuyet[0]
		_ = r.DB.SetStep(runID, s.ID, store.StepWaiting, s.Message, 0)
		st.set(s.ID, store.StepWaiting, "")
		_ = r.DB.SetRunState(runID, store.RunWaiting)
		r.Bus.Publish(events.Event{
			Type: events.FlowWaiting, Addr: f.Name + "." + s.ID, SessionID: runID,
			Msg: "chờ duyệt: " + s.Message,
			Detail: map[string]string{
				"flow": f.Name, "run": fmt.Sprint(runID), "step": s.ID, "ly_do": s.Message,
			},
		})
		return Result{RunID: runID, State: store.RunWaiting, Waiting: s.ID}, nil
	}

	_ = r.DB.SetRunState(runID, store.RunDone)
	r.Bus.Publish(events.Event{Type: events.FlowDone, Addr: f.Name, SessionID: runID,
		Msg:    fmt.Sprintf("xong #%d", runID),
		Detail: map[string]string{"flow": f.Name, "run": fmt.Sprint(runID)}})
	return Result{RunID: runID, State: store.RunDone}, nil
}
