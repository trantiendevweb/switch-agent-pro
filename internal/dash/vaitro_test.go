package dash

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// chiTiet gọi /api/flow/detail và trả về vai trò từng bước.
func chiTietVai(t *testing.T, s *Server, runID int64) map[string]string {
	t.Helper()
	r := req("GET", "/api/flow/detail?id="+strconv.FormatInt(runID, 10))
	r.AddCookie(dangNhap(t, s, "127.0.0.1:4600"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("phải 200, được %d — %s", w.Code, w.Body.String())
	}
	var body struct {
		Steps []struct {
			ID     string `json:"id"`
			VaiTro string `json:"vaiTro"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Steps) == 0 {
		t.Fatal("không có bước nào trong trả lời")
	}
	out := map[string]string{}
	for _, st := range body.Steps {
		out[st.ID] = st.VaiTro
	}
	return out
}

// (c) /api/flow/detail phải trả VAI TRÒ đúng của từng bước.
//
// Đây là endpoint mặt 2D và mặt 3D cùng đọc để dựng phòng agent. Thiếu vai thì
// hai mặt hoặc phải tự đoán từ tên bước (đoán sai là hiện sai), hoặc dồn tất cả
// vào một phòng chung — đúng thứ người dùng phàn nàn: "ai là leader, ai là nhân
// viên".
//
// Dùng bước `notify` để khỏi cần agent thật.
func TestFlowDetailTraVaiTroTungBuoc(t *testing.T) {
	s := newTestServer(t)
	dir := t.TempDir()
	t.Chdir(dir)

	f := flow.Flow{Name: "thu", Steps: []flow.Step{
		{ID: "chia", Type: flow.TypeNotify, Message: "chia việc", VaiTro: flow.VaiLeader},
		{ID: "kiem", Type: flow.TypeNotify, Message: "kiểm", VaiTro: flow.VaiTester, Needs: []string{"chia"}},
		// Cố ý KHÔNG khai vai — phải ra rỗng, không được đoán hộ.
		{ID: "bao", Type: flow.TypeNotify, Message: "xong", Needs: []string{"kiem"}},
	}}
	if _, err := s.api.FlowSave(dir, f); err != nil {
		t.Fatal(err)
	}
	res, err := s.api.FlowRun(context.Background(), dir, "thu", nil, api.Addr{})
	if err != nil {
		t.Fatal(err)
	}

	vai := chiTietVai(t, s, res.RunID)
	if vai["chia"] != flow.VaiLeader {
		t.Fatalf("bước chia phải là %q, mặt web nhận %q", flow.VaiLeader, vai["chia"])
	}
	if vai["kiem"] != flow.VaiTester {
		t.Fatalf("bước kiem phải là %q, mặt web nhận %q", flow.VaiTester, vai["kiem"])
	}
	if vai["bao"] != "" {
		t.Fatalf("bước không khai vai phải RỖNG (chưa phân vai), mặt web nhận %q", vai["bao"])
	}
}

// (d) Bước có trong SỔ mà flows.toml đã bỏ (flow sửa sau khi chạy) vẫn phải
// hiện, và vai của nó để RỖNG: không còn định nghĩa nào để lấy vai, mà bịa ra
// một vai thì mặt 3D sẽ xếp nó vào phòng sai và không ai biết là bịa.
func TestFlowDetailBuocBiXoaKhoiFlowThiVaiRong(t *testing.T) {
	s := newTestServer(t)
	dir := t.TempDir()
	t.Chdir(dir)

	f := flow.Flow{Name: "thu", Steps: []flow.Step{
		{ID: "chia", Type: flow.TypeNotify, Message: "chia việc", VaiTro: flow.VaiLeader},
		{ID: "cu", Type: flow.TypeNotify, Message: "bước sắp bị xoá", VaiTro: flow.VaiCoder,
			Needs: []string{"chia"}},
	}}
	if _, err := s.api.FlowSave(dir, f); err != nil {
		t.Fatal(err)
	}
	res, err := s.api.FlowRun(context.Background(), dir, "thu", nil, api.Addr{})
	if err != nil {
		t.Fatal(err)
	}
	// Sửa flow SAU khi chạy: bỏ hẳn bước `cu` khỏi flows.toml.
	f.Steps = f.Steps[:1]
	if _, err := s.api.FlowSave(dir, f); err != nil {
		t.Fatal(err)
	}

	vai := chiTietVai(t, s, res.RunID)
	if _, co := vai["cu"]; !co {
		t.Fatal("bước đã bị xoá khỏi flows.toml vẫn phải hiện trong lượt chạy cũ")
	}
	if vai["cu"] != "" {
		t.Fatalf("bước không còn định nghĩa thì vai phải RỖNG, mặt web nhận %q", vai["cu"])
	}
	if vai["chia"] != flow.VaiLeader {
		t.Fatalf("bước còn định nghĩa vẫn phải giữ vai: được %q", vai["chia"])
	}
}
