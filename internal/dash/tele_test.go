package dash

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/tele"
)

// datTelegram viết cấu hình Telegram vào HOME giả của server test, và trỏ API
// vào một bưu điện giả. Không test nào ở đây chạm mạng thật.
func datTelegram(t *testing.T, apiURL string) string {
	t.Helper()
	const token = "8888888:BI-MAT-KHONG-DUOC-RA-KHOI-MAY"
	if err := os.MkdirAll(filepath.Dir(tele.ConfigPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	c := tele.Config{Token: token, ChatID: "-100999", API: apiURL}
	b, _ := json.Marshal(c)
	if err := os.WriteFile(tele.ConfigPath(), b, 0o600); err != nil {
		t.Fatal(err)
	}
	return token
}

// Luật của mặt web (đầu server.go): TUYỆT ĐỐI không gửi secret ra ngoài.
//
// Token bot là secret đúng nghĩa — ai có nó thì nhắn được vào nhóm chat của chủ
// máy. Mà dash có chế độ phơi ra mạng, nên một trường thừa trong JSON là đủ để
// token đi ra internet.
func TestApiTeleKhongBaoGioLoToken(t *testing.T) {
	s := newTestServer(t)
	token := datTelegram(t, "http://127.0.0.1:1") // không ai gọi tới trong test này
	ck := dangNhap(t, s, "127.0.0.1:4600")

	r := httptest.NewRequest("GET", "/api/tele", nil)
	r.Host = "127.0.0.1:4600"
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("muốn 200, được %d: %s", w.Code, w.Body.String())
	}
	than := w.Body.String()
	if strings.Contains(than, token) {
		t.Fatalf("token lọt ra mặt web:\n%s", than)
	}
	var d struct {
		DaCauHinh bool   `json:"da_cau_hinh"`
		ChatID    string `json:"chat_id"`
	}
	if err := json.Unmarshal([]byte(than), &d); err != nil {
		t.Fatal(err)
	}
	if !d.DaCauHinh || d.ChatID != "-100999" {
		t.Fatalf("trạng thái sai: %+v", d)
	}
}

// Nút "gửi thử" trên dashboard phải gửi THẬT, không phải nút giả cho đẹp.
func TestApiTeleGuiThuThatSuGuiDi(t *testing.T) {
	nhan := make(chan string, 4)
	buuDien := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var than struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&than)
		nhan <- than.Text
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer buuDien.Close()

	s := newTestServer(t)
	datTelegram(t, buuDien.URL)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	r := httptest.NewRequest("POST", "/api/tele", strings.NewReader("{}"))
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Origin", "http://127.0.0.1:4600")
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("muốn 200, được %d: %s", w.Code, w.Body.String())
	}
	select {
	case text := <-nhan:
		if !strings.Contains(text, "tin thử") {
			t.Fatalf("nội dung tin thử lạ:\n%s", text)
		}
	default:
		t.Fatal("bấm gửi thử mà không có tin nào bay đi")
	}
}

// Luật giao diện của dự án: trang nào có animation/transition thì BẮT BUỘC có
// @media (prefers-reduced-motion: reduce).
//
// Vì sao canh bằng test: quy tắc này chỉ sai khi có người thêm một hiệu ứng mới
// vào trang cũ, và đó đúng là lúc không ai nhớ mở lại tài liệu design system ra
// đọc. (Đã bắt được flow.html thiếu media query bằng chính test này.)
func TestTrangCoChuyenDongPhaiTonTrongReducedMotion(t *testing.T) {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		t.Fatal(err)
	}
	err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return err
		}
		b, err := fs.ReadFile(sub, p)
		if err != nil {
			return err
		}
		noiDung := string(b)
		coChuyenDong := strings.Contains(noiDung, "transition") ||
			strings.Contains(noiDung, "animation:") ||
			strings.Contains(noiDung, "@keyframes")
		if coChuyenDong && !strings.Contains(noiDung, "prefers-reduced-motion") {
			t.Errorf("%s có chuyển động nhưng thiếu @media (prefers-reduced-motion: reduce)", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
