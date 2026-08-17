package dash

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
)

func serverWithLogin(t *testing.T) *Server {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := SetPassword("Admin", "matkhau-thu-nghiem"); err != nil {
		t.Fatal(err)
	}
	a, err := api.New(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return New(a)
}

// Mật khẩu KHÔNG được lưu thô: file chỉ chứa salt + hash.
func TestMatKhauKhongLuuTho(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := SetPassword("Admin", "bi-mat-cua-toi"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(AuthPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "bi-mat-cua-toi") {
		t.Fatal("mật khẩu thô nằm trong file — phải băm")
	}
	a := LoadAuth()
	if !a.Check("Admin", "bi-mat-cua-toi") {
		t.Fatal("mật khẩu đúng mà không qua")
	}
	if a.Check("Admin", "sai") || a.Check("KhongPhaiAdmin", "bi-mat-cua-toi") {
		t.Fatal("mật khẩu/tên sai mà vẫn qua")
	}
}

func TestMatKhauQuaNganBiTuChoi(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := SetPassword("Admin", "abc"); err == nil {
		t.Fatal("mật khẩu 3 ký tự mà vẫn nhận")
	}
}

// Chưa đăng nhập, mở bằng trình duyệt → chuyển tới form thay vì 401 trần trụi.
func TestChuaDangNhapThiChuyenToiForm(t *testing.T) {
	s := serverWithLogin(t)
	r := httptest.NewRequest("GET", "/3d.html", nil)
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("phải chuyển hướng 303, được %d", w.Code)
	}
	if loc := w.Header().Get("Location"); !strings.HasPrefix(loc, "/login") {
		t.Fatalf("phải chuyển tới /login, được %s", loc)
	}
}

// Đăng nhập đúng thì nhận cookie và dùng được API mà KHÔNG cần token.
func TestDangNhapDungThiVaoDuocBangCookie(t *testing.T) {
	s := serverWithLogin(t)
	form := url.Values{"user": {"Admin"}, "password": {"matkhau-thu-nghiem"}}
	r := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://127.0.0.1:4600")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("đăng nhập đúng phải 303, được %d", w.Code)
	}
	var ck *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			ck = c
		}
	}
	if ck == nil || ck.Value == "" {
		t.Fatal("không nhận được cookie phiên")
	}
	if !ck.HttpOnly {
		t.Fatal("cookie phiên phải HttpOnly")
	}

	r2 := httptest.NewRequest("GET", "/api/state", nil)
	r2.Host = "127.0.0.1:4600"
	r2.AddCookie(ck)
	w2 := httptest.NewRecorder()
	s.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("có cookie phải vào được API, được %d", w2.Code)
	}
}

// Sai mật khẩu thì không cấp cookie.
func TestDangNhapSaiKhongCapCookie(t *testing.T) {
	s := serverWithLogin(t)
	form := url.Values{"user": {"Admin"}, "password": {"sai-bet"}}
	r := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://127.0.0.1:4600")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName && c.Value != "" {
			t.Fatal("sai mật khẩu mà vẫn cấp cookie")
		}
	}
}

// Đăng xuất thì cookie cũ hết tác dụng ngay.
func TestDangXuatHuyPhien(t *testing.T) {
	s := serverWithLogin(t)
	id := s.sess.create()
	if !s.sess.valid(id) {
		t.Fatal("phiên vừa tạo phải hợp lệ")
	}
	s.sess.drop(id)
	if s.sess.valid(id) {
		t.Fatal("phiên đã huỷ mà vẫn hợp lệ")
	}
}
