package dash

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// Các endpoint cho workflow board (mặt thứ 3 trong bốn mặt điều khiển).
//
// Điểm quan trọng về thiết kế: chạy flow có thể mất hàng chục phút (bước agent
// đợi Claude làm xong). Không được để request HTTP treo suốt thời gian đó, nên
// các endpoint "chạy" chỉ KHỞI ĐỘNG rồi trả ngay; tiến độ đi qua luồng event
// mà cả ba mặt web đang nghe. Đúng luật "sự thật đến từ event".

type stepDTO struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Needs   []string `json:"needs,omitempty"`
	State   string   `json:"state"`   // pending|running|done|failed|skipped|waiting
	Msg     string   `json:"msg,omitempty"`
	Attempt int      `json:"attempt,omitempty"`
	Detail  string   `json:"detail,omitempty"` // prompt / lệnh / lời nhắn, đã rút gọn
}

type flowDTO struct {
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Steps int    `json:"steps"`
}

type runDTO struct {
	ID      int64  `json:"id"`
	Flow    string `json:"flow"`
	State   string `json:"state"`
	Started int64  `json:"started"`
}

// GET /api/flows — danh sách flow + lịch sử chạy.
func (s *Server) handleFlows(w http.ResponseWriter, r *http.Request) {
	dir := s.workDir()
	flows, _, err := s.api.FlowList(dir)
	if err != nil {
		writeErr(w, err)
		return
	}
	fs := make([]flowDTO, 0, len(flows))
	for _, n := range flow.Names(flows) {
		f := flows[n]
		fs = append(fs, flowDTO{Name: n, Desc: f.Desc, Steps: len(f.Steps)})
	}
	runs, err := s.api.FlowRuns(30)
	if err != nil {
		writeErr(w, err)
		return
	}
	rs := make([]runDTO, 0, len(runs))
	for _, r := range runs {
		rs = append(rs, runDTO{ID: r.ID, Flow: r.Flow, State: r.State, Started: r.Started.Unix()})
	}
	writeJSON(w, map[string]any{"flows": fs, "runs": rs})
}

// GET /api/run?id=N — chi tiết một lần chạy: từng bước và trạng thái.
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		writeErr(w, err)
		return
	}
	run, steps, def, err := s.api.FlowRunDetail(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	order, _ := flow.Order(def)
	out := make([]stepDTO, 0, len(order))
	for _, st := range order {
		d := stepDTO{ID: st.ID, Type: st.Type, Needs: st.Needs, State: "pending"}
		if got, ok := steps[st.ID]; ok {
			d.State, d.Msg, d.Attempt = got.State, got.Msg, got.Attempt
		}
		switch st.Type {
		case flow.TypeAgent, flow.TypeReview:
			d.Detail = st.Prompt
		case flow.TypeShell, flow.TypeTest, flow.TypeLint:
			d.Detail = join(st.Run)
		default:
			d.Detail = st.Message
		}
		out = append(out, d)
	}
	writeJSON(w, map[string]any{
		"run": runDTO{ID: run.ID, Flow: run.Flow, State: run.State, Started: run.Started.Unix()},
		"steps": out,
	})
}

// POST /api/flow/run — khởi động một flow rồi trả ngay số lần chạy.
func (s *Server) handleFlowRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string            `json:"name"`
		Profile string            `json:"profile"`
		Vars    map[string]string `json:"vars"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	dir := s.workDir()
	// Kiểm trước khi chạy nền, để lỗi rõ ràng (flow không tồn tại, DAG hỏng…)
	// còn báo được về cho người bấm nút.
	if _, _, err := s.api.FlowShow(dir, req.Name); err != nil {
		writeErr(w, err)
		return
	}
	go func() {
		_, _ = s.api.FlowRun(context.Background(), dir, req.Name, req.Vars, api.ParseAddr(req.Profile))
	}()
	writeJSON(w, map[string]string{"started": req.Name})
}

// POST /api/flow/decide — duyệt hoặc từ chối một bước đang chờ.
func (s *Server) handleFlowDecide(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      int64  `json:"id"`
		Step    string `json:"step"`
		Approve bool   `json:"approve"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	// Từ chối xong là hết, không có gì chạy tiếp → làm đồng bộ cho chắc.
	if !req.Approve {
		if _, err := s.api.FlowApprove(context.Background(), req.ID, req.Step, s.who(), false, api.Addr{}); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]string{"decided": "rejected"})
		return
	}
	// Duyệt thì có thể kéo theo cả chuỗi bước dài → chạy nền.
	if err := s.api.FlowApproveOnly(req.ID, req.Step, s.who()); err != nil {
		writeErr(w, err)
		return
	}
	go func() {
		_, _ = s.api.FlowResume(context.Background(), req.ID, api.Addr{})
	}()
	writeJSON(w, map[string]string{"decided": "approved"})
}

// who ghi lại ai đã duyệt. Có đăng nhập thì lấy tên tài khoản đó.
func (s *Server) who() string {
	if s.auth != nil && s.auth.User != "" {
		return s.auth.User + " (dashboard)"
	}
	return "dashboard"
}

func join(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}
