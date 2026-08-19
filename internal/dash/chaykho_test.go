package dash

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// postKho gọi /api/flow/kho qua đúng cửa đăng nhập, trả về mã và thân trả lời.
func postKho(t *testing.T, s *Server, than string) (int, map[string]any) {
	t.Helper()
	r := httptest.NewRequest("POST", "/api/flow/kho", strings.NewReader(than))
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(dangNhap(t, s, "127.0.0.1:4600"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

// Mặt web phải chạy khan được — không thì luật ngang quyền (MASTER-PLAN mục 2c)
// lại thành khẩu hiệu: người ngồi trước dashboard chỉ có mỗi nút chạy THẬT, và
// đó đúng là cách ba lượt #30/#32/#33 ngày 19/08 ra đời.
func TestWebChayKhoTraKeHoach(t *testing.T) {
	s := newTestServer(t)
	dir := t.TempDir()
	t.Chdir(dir)

	f := flow.Flow{Name: "kho", Desc: "thử", Vars: map[string]string{"viec": "sửa lỗi X"},
		Steps: []flow.Step{
			{ID: "code", Type: flow.TypeAgent, Profile: "claude:tns", Prompt: "làm: {{viec}}", Copies: 2},
			{ID: "kiem", Type: flow.TypeShell, Run: []string{"go", "version"}},
			{ID: "duyet", Type: flow.TypeApprove, Message: "duyệt đi", Needs: []string{"code", "kiem"}},
		}}
	if _, err := s.api.FlowSave(dir, f); err != nil {
		t.Fatal(err)
	}

	code, body := postKho(t, s, `{"name":"kho"}`)
	if code != 200 {
		t.Fatalf("phải 200, được %d — %v", code, body)
	}
	dots, _ := body["dot"].([]any)
	if len(dots) != 2 {
		t.Fatalf("phải trả 2 đợt, được %d: %v", len(dots), body["dot"])
	}
	d0, _ := dots[0].(map[string]any)
	buoc, _ := d0["buoc"].([]any)
	if len(buoc) != 2 {
		t.Fatalf("đợt 1 phải có 2 bước chạy song song, được %d", len(buoc))
	}
	b0, _ := buoc[0].(map[string]any)
	if b0["taiKhoan"] != "claude:tns" {
		t.Fatalf("phải nói tài khoản của bước, được %v", b0["taiKhoan"])
	}
	if p, _ := b0["prompt"].(string); !strings.Contains(p, "sửa lỗi X") {
		t.Fatalf("prompt phải đã thay biến, được %q", p)
	}
	d1, _ := dots[1].(map[string]any)
	if d1["choDuyet"] != true {
		t.Fatalf("đợt 2 phải được đánh dấu là rào duyệt, được %v", d1)
	}
	if n, _ := body["soAgent"].(float64); n != 2 {
		t.Fatalf("phải nói tổng 2 phiên agent, được %v", body["soAgent"])
	}
}

// Lời hứa của tính năng, kiểm từ mặt web: bấm "Chạy khan" KHÔNG được để lại một
// lượt chạy nào trong sổ, cũng không bật phiên agent nào.
//
// Nếu ai đó nối nhầm nút này vào /api/flow/run thì test đỏ ngay tại đây.
func TestWebChayKhoKhongGhiSoKhongBatAgent(t *testing.T) {
	s := newTestServer(t)
	dir := t.TempDir()
	t.Chdir(dir)

	f := flow.Flow{Name: "kho", Steps: []flow.Step{
		{ID: "bao", Type: flow.TypeNotify, Message: "một dòng"},
	}}
	if _, err := s.api.FlowSave(dir, f); err != nil {
		t.Fatal(err)
	}
	truoc, err := s.api.FlowRuns(100)
	if err != nil {
		t.Fatal(err)
	}

	if code, body := postKho(t, s, `{"name":"kho"}`); code != 200 {
		t.Fatalf("phải 200, được %d — %v", code, body)
	}

	sau, err := s.api.FlowRuns(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(sau) != len(truoc) {
		t.Fatalf("chạy khan từ web đã ghi %d lượt chạy vào sổ", len(sau)-len(truoc))
	}
	phien, err := s.api.SessionList()
	if err != nil {
		t.Fatal(err)
	}
	if len(phien) != 0 {
		t.Fatalf("chạy khan từ web đã bật %d phiên agent", len(phien))
	}
}

// Bảng flow phải có NÚT gọi đường này. Endpoint có mà không nút nào bấm được
// thì với người dùng nó không tồn tại — đúng thứ luật ngang quyền chống.
func TestBangFlowCoNutChayKhan(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("web", "flow.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `id="kho"`) {
		t.Error("flow.html không có nút nào mang id=\"kho\" — mặt web thiếu đường chạy khan")
	}
	if !strings.Contains(s, "/api/flow/kho") {
		t.Error("flow.html không gọi /api/flow/kho — nút chạy khan không nối vào đâu cả")
	}
	// Nút phải nối vào ĐÚNG chỗ: bắt luôn cả trường hợp nối nhầm sang chạy thật.
	i := strings.Index(s, `$('#kho').onclick`)
	if i < 0 {
		t.Fatal("không thấy chỗ gắn hành động cho nút #kho")
	}
	than := s[i:]
	if j := strings.Index(than, "/api/flow/"); j < 0 || !strings.HasPrefix(than[j:], "/api/flow/kho") {
		t.Error("nút Chạy khan gọi nhầm endpoint — nó phải gọi /api/flow/kho")
	}
}
