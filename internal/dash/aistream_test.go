package dash

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Mặt web hỏi AI API bằng STREAM. Đo được: một lượt `grok-4.5` mất 31 giây
// (docs/DO-LUONG.md). Không streaming thì người bấm nút nhìn ô trống nửa phút,
// không phân biệt được "đang nghĩ" với "đã treo" — và cách duy nhất họ có là
// bấm lại, tức trả tiền hai lần cho cùng một câu hỏi.

// Dùng CHUNG endpoint `/api/ai`, không mở `/api/ai/stream` riêng: đây vẫn là
// action `api.call`, chỉ khác cách gửi về. Thêm endpoint là thêm một hành động
// vào hợp đồng và một lệnh CLI phải có tương ứng (test ngang quyền) — trả giá
// đó cho một chi tiết truyền tải là sai chỗ.
func TestStreamDungChungEndpointKhongDeAction(t *testing.T) {
	b := docTep(t, "server.go")
	if strings.Contains(b, `"/api/ai/stream"`) {
		t.Error("mở endpoint riêng cho stream — nó vẫn là action api.call, đừng thêm hành động vào hợp đồng")
	}
	if !regexp.MustCompile(`Stream\s+bool\s+` + "`" + `json:"stream"` + "`").MatchString(b) {
		t.Error("không nhận cờ stream trong thân request của /api/ai")
	}
}

// Lỗi phải đi TRONG LUỒNG, không phải bằng mã HTTP: header 200 đã gửi từ trước
// khi biết kết quả, nên `writeErr` lúc đó không còn đổi được mã nữa. Đóng ngang
// mà không nói gì thì trình duyệt chỉ thấy kết nối đứt và sẽ tự thử lại — trả
// tiền lần nữa cho cùng một câu hỏi.
func TestLoiDiTrongLuongChuKhongPhaiMaHTTP(t *testing.T) {
	b := docTep(t, "server.go")
	i := strings.Index(b, "func (s *Server) aiStream")
	if i < 0 {
		t.Fatal("không có aiStream")
	}
	than := b[i:]
	if j := strings.Index(than, "\nfunc "); j > 0 {
		than = than[:j]
	}
	if !strings.Contains(than, `"loi": err.Error()`) {
		t.Error("aiStream không gửi lỗi vào luồng — trình duyệt chỉ thấy kết nối đứt rồi tự thử lại")
	}
	if !strings.Contains(than, "X-Accel-Buffering") {
		t.Error("không tắt đệm của proxy — streaming sẽ thành không-streaming mà không ai báo lỗi")
	}
	// Mẩu cuối phải mang usage VÀ câu cảnh báo khi nhà cung cấp không trả usage.
	for _, can := range []string{"thieu_usage", "canh_bao", "usage"} {
		if !strings.Contains(than, can) {
			t.Errorf("mẩu tổng kết thiếu %q — sổ chi phí của mặt web sẽ ghi 0 như thật", can)
		}
	}
}

// Trang phải ghép lại mẩu SSE bị cắt đôi giữa hai lần đọc. Bỏ phần dư là mất
// chữ giữa câu, mà mất im lặng — câu trả lời vẫn hiện ra, chỉ thiếu.
func TestTrangGhepLaiManhSSEBiCatDoi(t *testing.T) {
	s := boComment(doc2DThuong(t))
	if !strings.Contains(s, "hoiAIStream") {
		t.Fatal("index.html không gọi đường stream")
	}
	if !strings.Contains(s, "dem = phan.pop()") {
		t.Error("index.html không giữ phần dư giữa hai lần đọc — mẩu bị cắt đôi sẽ mất chữ")
	}
	// EventSource chỉ biết GET; prompt phải đi bằng POST vì URL rơi vào log
	// proxy và lịch sử trình duyệt.
	if strings.Contains(s, "new EventSource('/api/ai") {
		t.Error("dùng EventSource cho /api/ai — prompt sẽ nằm trong URL")
	}
}

// Cả hai mặt phải nói CÙNG một câu khi thiếu usage. Mỗi mặt tự chế một cách
// diễn đạt là cách để hai mặt nói hai nghĩa về cùng một chuyện tiền bạc.
func TestHaiMatCungNoiVeThieuUsage(t *testing.T) {
	web := boComment(doc2DThuong(t))
	if !strings.Contains(web, "canh_bao") {
		t.Error("index.html không hiện cảnh báo thiếu usage")
	}
	cli, err := os.ReadFile(filepath.Join("..", "..", "cmd", "sagent", "api.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cli), "CanhBaoThieuUsage") {
		t.Error("CLI không hiện cảnh báo thiếu usage — hai mặt sẽ nói khác nhau")
	}
}

// Endpoint stream vẫn phải nằm sau cửa đăng nhập. Một đường không cần đăng nhập
// mà gọi được ra Internet là máy dò cổng miễn phí cho người lạ — và ở đây còn
// tiêu tiền của chủ máy.
func TestStreamVanPhaiDangNhap(t *testing.T) {
	s := newTestServer(t)
	w := httptest.NewRecorder()
	r := req("POST", "/api/ai")
	r.Body = nil
	s.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Errorf("/api/ai trả 200 khi chưa đăng nhập")
	}
	_ = json.Valid(nil)
}
