package dash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
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
	State   string   `json:"state"` // pending|running|done|failed|skipped|waiting
	Msg     string   `json:"msg,omitempty"`
	Attempt int      `json:"attempt,omitempty"`
	Detail  string   `json:"detail,omitempty"` // prompt / lệnh / lời nhắn, đã rút gọn
	// Output là thứ agent THỰC SỰ trả về. Thiếu nó thì mặt web chỉ khoe đề bài
	// mà giấu bài làm: lần chạy #8 hiện "completed" đủ ba bước trong khi cả hai
	// agent chỉ trả về câu từ chối quyền. Không đọc được kết quả thì không cách
	// nào biết flow chạy thật hay chạy suông.
	Output string `json:"output,omitempty"`
	// Profile: TÀI KHOẢN làm bước này. Thiếu nó thì mặt 3D không biết gán bước
	// cho agent nào — cả cảnh chỉ còn là mấy hình hộp đứng im, đúng thứ người
	// dùng phàn nàn: "ai là leader, ai là nhân viên, nhiệm vụ ra sao".
	Profile string `json:"profile,omitempty"`
	// Chi phí đo được của bước, đọc từ dữ liệu CÓ CẤU TRÚC của agent (không đoán).
	// 0 nghĩa là provider không cho biết — bỏ khỏi JSON để mặt web khỏi hiện
	// "0,0000 USD" gây tưởng là miễn phí trong khi thật ra là chưa đo được.
	CostUSD   float64 `json:"cost_usd,omitempty"`
	TokensIn  int     `json:"tokens_in,omitempty"`
	TokensOut int     `json:"tokens_out,omitempty"`
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
	// Tổng cả lượt: chủ dự án hỏi "lượt này tốn bao nhiêu" — cộng dồn từ các bước
	// thật đã đo, không phải ước lượng.
	var tongChiPhi float64
	var tongVao, tongRa int
	for _, st := range order {
		d := stepDTO{ID: st.ID, Type: st.Type, Needs: st.Needs, State: "pending", Profile: st.Profile}
		if got, ok := steps[st.ID]; ok {
			d.State, d.Msg, d.Attempt = got.State, got.Msg, got.Attempt
			d.Output = got.Output
			d.CostUSD, d.TokensIn, d.TokensOut = got.CostUSD, got.TokensIn, got.TokensOut
			tongChiPhi += got.CostUSD
			tongVao += got.TokensIn
			tongRa += got.TokensOut
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
		"run":   runDTO{ID: run.ID, Flow: run.Flow, State: run.State, Started: run.Started.Unix()},
		"steps": out,
		"cost":  map[string]any{"usd": tongChiPhi, "tokens_in": tongVao, "tokens_out": tongRa},
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

// GET /api/flow/detail?id=N — một lượt chạy: trạng thái, CÂU HỎI và CÂU TRẢ LỜI
// của từng bước, kèm sơ đồ phụ thuộc.
//
// Vì sao cần: `sagent flow runs <N>` ở terminal đọc được hết những thứ này từ
// lâu, mặt web thì không. Luật ngang quyền (MASTER-PLAN mục 2c) nói một tính
// năng chưa xong nếu chưa làm ở cả bốn mặt — và test ngang quyền không bắt được
// vì `flow.runs` khai đường là /api/state, một endpoint KHÔNG hề trả runs.
// Đường dẫn tồn tại nên test xanh, còn dữ liệu thì không có.
func (s *Server) handleFlowDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		writeErr(w, fmt.Errorf("thiếu hoặc sai tham số id"))
		return
	}
	run, steps, def, err := s.api.FlowRunDetail(id)
	if err != nil {
		writeErr(w, err)
		return
	}

	// Trả về theo THỨ TỰ ĐỊNH NGHĨA để mặt web dựng được mạch trên xuống dưới
	// mà không phải tự sắp xếp lại đồ thị.
	type buocDTO struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Profile string `json:"profile"`
		// VaiTro lấy từ ĐỊNH NGHĨA flow (flows.toml), không phải từ sổ: sổ chỉ
		// ghi việc đã chạy, còn vai trò là thứ người dùng khai. Bước có trong sổ
		// mà flows.toml đã bỏ thì rỗng — không có chỗ nào để lấy, và đoán thì
		// sai.
		VaiTro    string   `json:"vaiTro"`
		Needs     []string `json:"needs"`
		State     string   `json:"state"`
		Msg       string   `json:"msg"`
		Prompt    string   `json:"prompt"`
		Output    string   `json:"output"`
		Attempt   int      `json:"attempt"`
		CostUSD   float64  `json:"costUsd"`
		TokensIn  int      `json:"tokensIn"`
		TokensOut int      `json:"tokensOut"`
		// Route: node `model` KHÔNG có `profile` — nó đi thẳng API, không qua CLI.
		// Thiếu trường này thì mặt web không biết ai đứng sau bước đó, và mặt
		// Trung tâm bỏ qua luôn: Phòng review trống trơn dù bước `soi` đã chạy
		// xong (đo 20/08, lượt #47).
		//
		// Server GIẢI SẴN: `route` rỗng trong flows.toml nghĩa là default_route
		// rồi tới dự phòng, mà luật đó nằm ở `api.ThuTuRoute`. Bắt mỗi mặt web tự
		// suy lại là cách để bốn mặt nói bốn kiểu về cùng một bước.
		Route []string `json:"route,omitempty"`
	}
	ds := make([]buocDTO, 0, len(def.Steps))
	daCo := map[string]bool{}
	themR := func(id, typ, prof, vai string, needs []string, st store.StepRun, route []string) {
		daCo[id] = true
		if needs == nil {
			needs = []string{}
		}
		ds = append(ds, buocDTO{
			ID: id, Type: typ, Profile: prof, VaiTro: vai, Needs: needs,
			State: st.State, Msg: st.Msg, Prompt: st.Prompt, Output: st.Output,
			Attempt: st.Attempt, CostUSD: st.CostUSD,
			TokensIn: st.TokensIn, TokensOut: st.TokensOut,
			Route: route,
		})
	}
	// them là lối gọi gọn cho bước KHÔNG phải node `model` — chúng không có route.
	them := func(id, typ, prof, vai string, needs []string, st store.StepRun) {
		themR(id, typ, prof, vai, needs, st, nil)
	}
	for i := range def.Steps {
		d := def.Steps[i]
		var route []string
		if d.Type == flow.TypeModel {
			route = s.api.ThuTuRoute(d.Route)
		}
		themR(d.ID, d.Type, d.Profile, d.VaiTro, d.Needs, steps[d.ID], route)
	}
	// Bước có trong sổ nhưng flows.toml đã bỏ (flow sửa sau khi chạy) vẫn phải
	// hiện — giấu đi thì người đọc tưởng lượt chạy ít bước hơn thực tế.
	for id, st := range steps {
		if !daCo[id] {
			them(id, "", "", "", nil, st)
		}
	}

	writeJSON(w, map[string]any{
		"id":      run.ID,
		"flow":    run.Flow,
		"state":   run.State,
		"dir":     run.Dir,
		"started": run.Started.Unix(),
		"steps":   ds,
	})
}

// GET /api/flow/tom-tat?id=N — BẢN TÓM TẮT một lượt chạy, kèm phần đối chiếu
// lời agent với git.
//
// Vì sao có mặt ở web chứ không chỉ ở terminal: /api/flow/detail đã đổ ra đủ
// mọi thứ agent nói, nhưng đọc hết một lượt bốn bước rồi tự kết luận là việc
// người dùng đang phải làm bằng tay — và bốn lần trong lịch sử dự án (lượt #21,
// #29, #31, #34) kết luận rút ra từ lời agent đã sai. Đây là endpoint duy nhất
// trả về CÂU TRẢ LỜI thay vì nguyên liệu.
//
// GET chứ không POST: nó chỉ đọc, không đổi gì trong sổ.
func (s *Server) handleFlowTomTat(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		writeErr(w, fmt.Errorf("thiếu hoặc sai tham số id"))
		return
	}
	tt, err := s.api.FlowTomTat(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, tt)
}

// POST /api/flow/cancel — đánh dấu một lượt chạy dở dang là ĐÃ HUỶ.
//
// Có mặt ở web chứ không chỉ ở terminal, vì đây đúng là chỗ người ta phát hiện
// ra vấn đề: bảng lịch sử hiện một lượt "đang chạy" từ đêm qua. Bắt họ mở
// terminal để dọn thứ họ vừa nhìn thấy là làm ngược luật ngang quyền.
//
// KHÔNG giết tiến trình nào — xem flow.Runner.Huy.
func (s *Server) handleFlowCancel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.api.FlowCancel(req.ID, s.who()); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]string{"cancelled": "ok"})
}

// GET /api/flow/def?name=x — định nghĩa đầy đủ của một flow, để bảng vẽ dựng lại.
func (s *Server) handleFlowDef(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	flows, _, err := s.api.FlowList(s.workDir())
	if err != nil {
		writeErr(w, err)
		return
	}
	f, ok := flows[name]
	if !ok {
		writeErr(w, errNotFound(name))
		return
	}
	writeJSON(w, map[string]any{
		"name": name, "desc": f.Desc, "vars": f.Vars, "steps": f.Steps,
		"builtin": flow.IsBuiltin(name),
	})
}

// POST /api/flow/save — ghi flow từ bảng vẽ xuống flows.toml.
//
// Bảng vẽ KHÔNG có kho riêng: nó đọc/ghi đúng file mà người dùng sửa tay được.
// Nhờ vậy flow dựng bằng giao diện và flow viết tay là một thứ.
func (s *Server) handleFlowSave(w http.ResponseWriter, r *http.Request) {
	var f flow.Flow
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeErr(w, err)
		return
	}
	path, err := s.api.FlowSave(s.workDir(), f)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]string{"saved": f.Name, "file": path})
}

// POST /api/flow/delete — xoá flow khỏi flows.toml.
func (s *Server) handleFlowDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	path, err := s.api.FlowDelete(s.workDir(), req.Name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]string{"deleted": req.Name, "file": path})
}

func errNotFound(name string) error { return fmt.Errorf("không có flow %q", name) }

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

// POST /api/flow/kho — CHẠY KHAN: trả về kế hoạch của một lượt chạy mà không
// chạy gì cả.
//
// Đây là endpoint DUY NHẤT trong nhóm flow không đẻ ra tác dụng phụ nào: không
// goroutine nền, không lượt chạy trong sổ, không agent. Nhờ vậy nó trả lời được
// NGAY trong request thay vì đẩy tiến độ qua luồng event như /api/flow/run —
// không có tiến độ nào để đẩy.
//
// Nút "Chạy khan" trên bảng flow gọi đúng đường này. Người dùng bấm "Chay
// workflow" để xem cổng kiểm nói gì là chuyện đã xảy ra thật ba lần trong ngày
// 19/08 (#30, #32, #33), mỗi lần đốt hạn mức và đẻ một lượt rác.
func (s *Server) handleFlowKho(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string            `json:"name"`
		Profile string            `json:"profile"`
		Vars    map[string]string `json:"vars"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	kh, err := s.api.FlowChayKho(s.workDir(), req.Name, req.Vars, api.ParseAddr(req.Profile))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, kh)
}
