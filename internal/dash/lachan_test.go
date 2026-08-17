package dash

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Năm test dưới đây sinh ra từ một lượt soát code bằng codex, và mỗi cáo buộc
// đều được kiểm lại bằng cách đọc code + dựng kịch bản, chứ không tin lời.

func postLogin(t *testing.T, s *Server, host, next, user, pass string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"user": {user}, "password": {pass}}
	u := "/login"
	if next != "" {
		u += "?next=" + url.QueryEscape(next)
	}
	r := httptest.NewRequest("POST", u, strings.NewReader(form.Encode()))
	r.Host = host
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://"+host)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

// 1. `next` bắt đầu bằng "//" là URL ĐỔI TÊN MIỀN, không phải đường dẫn nội bộ.
// Kiểm `strings.HasPrefix(next, "/")` cho nó lọt.
func TestNextKhongDuocDoiTenMien(t *testing.T) {
	for _, xau := range []string{"//evil.example", "//evil.example/x", `/\evil.example`, "///evil.example"} {
		s := newTestServer(t)
		w := postLogin(t, s, "127.0.0.1:4600", xau, "Admin", matKhauTest)
		loc := w.Header().Get("Location")
		if strings.HasPrefix(loc, "//") || strings.HasPrefix(loc, `/\`) {
			t.Errorf("next=%q -> chuyển hướng tới %q: ra khỏi máy người dùng", xau, loc)
		}
	}
}

// 2. /login nằm NGOÀI guard nên không có lớp chống DNS-rebind. Tên miền của kẻ
// tấn công trỏ về 127.0.0.1, trang của nó POST mật khẩu vào dash nội bộ; Origin
// và Host đều là tên miền đó nên sameOrigin cho qua.
func TestLoginCungPhaiKiemHostLoopback(t *testing.T) {
	s := newTestServer(t)
	w := postLogin(t, s, "evil.example", "", "Admin", matKhauTest)
	if w.Code == http.StatusSeeOther {
		t.Fatal("đăng nhập được với Host lạ ở chế độ kín — lớp chống DNS-rebind không bao /login")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("muốn 403, được %d", w.Code)
	}
}

// 3. guard gọi noteFail() cho MỌI request chưa đăng nhập, mà bộ đếm đó dùng
// chung. 5 request vô danh là khoá luôn người đang dùng hợp lệ.
func TestRequestVoDanhKhongKhoaNguoiDangDung(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	for i := 0; i < 8; i++ { // kẻ lạ gõ cửa, không cookie
		r := httptest.NewRequest("GET", "/api/state", nil)
		r.Host = "127.0.0.1:4600"
		s.ServeHTTP(httptest.NewRecorder(), r)
	}

	r := httptest.NewRequest("GET", "/api/state", nil)
	r.Host = "127.0.0.1:4600"
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code == http.StatusTooManyRequests {
		t.Fatal("người đã đăng nhập bị khoá vì request vô danh của kẻ khác — DoS bằng 8 dòng curl")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("muốn 200, được %d", w.Code)
	}
}

// 4. noteOK() reset bộ đếm ở MỌI request hợp lệ. Dashboard tự poll 5 giây/lần,
// nên chỉ cần có một tab đang mở là chống dò mật khẩu bị vô hiệu.
func TestPollKhongLamMatChongDoMatKhau(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	for i := 0; i < 6; i++ {
		postLogin(t, s, "127.0.0.1:4600", "", "Admin", "sai-mat-khau")
		// dashboard của người dùng thật poll xen vào giữa
		r := httptest.NewRequest("GET", "/api/state", nil)
		r.Host = "127.0.0.1:4600"
		r.AddCookie(ck)
		s.ServeHTTP(httptest.NewRecorder(), r)
	}

	w := postLogin(t, s, "127.0.0.1:4600", "", "Admin", "sai-mat-khau")
	if !strings.Contains(w.Body.String(), "thử lại sau") {
		t.Fatal("dò 7 lần mật khẩu sai mà không bị bắt chờ — poll của dashboard đã xoá bộ đếm")
	}
}

// 5. /logout nhận mọi method và không kiểm nguồn. Trang lạ điều hướng tới đó là
// phiên bị xoá. Hại nhỏ (chỉ bị đăng xuất) nhưng không có lý do gì để hở.
func TestLogoutKhongBiCheoTrang(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	r := httptest.NewRequest("GET", "/logout", nil)
	r.Host = "127.0.0.1:4600"
	r.AddCookie(ck)
	r.Header.Set("Sec-Fetch-Site", "cross-site") // trình duyệt gắn khi điều hướng chéo trang
	s.ServeHTTP(httptest.NewRecorder(), r)

	// Phiên phải CÒN SỐNG.
	r2 := httptest.NewRequest("GET", "/api/state", nil)
	r2.Host = "127.0.0.1:4600"
	r2.AddCookie(ck)
	w2 := httptest.NewRecorder()
	s.ServeHTTP(w2, r2)
	if w2.Code == http.StatusUnauthorized {
		t.Fatal("trang lạ điều hướng tới /logout đã xoá được phiên của người dùng")
	}
}
