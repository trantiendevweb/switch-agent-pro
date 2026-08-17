package dash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
)

const matKhauTest = "matkhau-thu-nghiem"

// newTestServer dựng một server thật (API + store trong HOME tạm) đã đặt sẵn
// mật khẩu, để test lá chắn. Không còn token nên mọi test đều phải đăng nhập.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := SetPassword("Admin", matKhauTest); err != nil {
		t.Fatal(err)
	}
	a, err := api.New(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return New(a)
}

// dangNhap đi qua đúng form đăng nhập và trả về cookie phiên — cửa vào DUY NHẤT.
func dangNhap(t *testing.T, s *Server, host string) *http.Cookie {
	t.Helper()
	form := url.Values{"user": {"Admin"}, "password": {matKhauTest}}
	r := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	r.Host = host
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://"+host)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName && c.Value != "" {
			return c
		}
	}
	t.Fatalf("đăng nhập không cấp cookie (mã %d)", w.Code)
	return nil
}

func req(method, target string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	r.Host = "127.0.0.1:4600" // giả lập loopback
	return r
}

// Chưa đăng nhập thì mọi cửa đều đóng.
func TestChuaDangNhapBiChan(t *testing.T) {
	s := newTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req("GET", "/api/state"))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("chưa đăng nhập phải 401, được %d", w.Code)
	}
}

// Cookie bịa ra thì không qua.
func TestCookieGiaBiChan(t *testing.T) {
	s := newTestServer(t)
	r := req("GET", "/api/state")
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "bia-ra-thoi"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("cookie giả phải 401, được %d", w.Code)
	}
}

// Token trên URL KHÔNG còn là cửa vào — có `?t=` cũng vẫn bị chặn.
func TestTokenTrenURLKhongConTacDung(t *testing.T) {
	s := newTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req("GET", "/api/state?t=batkychuoinao"))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("token trên URL phải hết tác dụng (401), được %d", w.Code)
	}
}

// Đăng nhập rồi thì /api/state trả JSON có mảng profiles + sessions.
func TestStateTraJSON(t *testing.T) {
	s := newTestServer(t)
	r := req("GET", "/api/state")
	r.AddCookie(dangNhap(t, s, "127.0.0.1:4600"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("phải 200, được %d — %s", w.Code, w.Body.String())
	}
	var body struct {
		Profiles []any `json:"profiles"`
		Sessions []any `json:"sessions"`
		APIVer   int   `json:"apiVersion"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("không phải JSON hợp lệ: %v", err)
	}
	if body.APIVer != api.Version {
		t.Fatalf("apiVersion = %d, muốn %d", body.APIVer, api.Version)
	}
}

// Chống DNS-rebind: Host là tên miền lạ (dù trỏ về loopback) thì bị chặn.
func TestHostLaBiChan(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := httptest.NewRequest("GET", "/api/state", nil)
	r.Host = "evil.example.com"
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Host lạ phải 403, được %d", w.Code)
	}
}

// Chống CSRF: POST từ gốc khác (dù đã đăng nhập) bị chặn.
func TestOriginLaBiChanTrenPOST(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := httptest.NewRequest("POST", "/api/stop", strings.NewReader(`{"all":true}`))
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Origin", "http://evil.example.com")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Origin lạ trên POST phải 403, được %d", w.Code)
	}
}

// POST cùng gốc loopback thì qua được (stop all với 0 phiên trả stopped:0).
func TestPOSTcungGocQua(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := httptest.NewRequest("POST", "/api/stop", strings.NewReader(`{"all":true}`))
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Origin", "http://127.0.0.1:4600")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("POST cùng gốc phải 200, được %d — %s", w.Code, w.Body.String())
	}
}

// Chế độ phơi ra mạng: Host là IP thật thì phải QUA, nhưng vẫn đòi đăng nhập.
func TestCheDoPhoiChoHostThatNhungVanDoiDangNhap(t *testing.T) {
	s := newTestServer(t)
	s.exposed = true // như khi chạy --host 0.0.0.0

	r := httptest.NewRequest("GET", "/api/state", nil)
	r.Host = "103.97.134.90:8788"
	r.AddCookie(dangNhap(t, s, "103.97.134.90:8788"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("phơi ra mạng + đã đăng nhập phải 200, được %d", w.Code)
	}

	r2 := httptest.NewRequest("GET", "/api/state", nil)
	r2.Host = "103.97.134.90:8788"
	w2 := httptest.NewRecorder()
	s.ServeHTTP(w2, r2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("phơi ra mạng mà chưa đăng nhập phải 401, được %d", w2.Code)
	}
}

// CSRF vẫn phải chặn khi phơi ra mạng: Origin lạ không được POST.
func TestCheDoPhoiVanChanOriginLa(t *testing.T) {
	s := newTestServer(t)
	s.exposed = true
	ck := dangNhap(t, s, "103.97.134.90:8788")
	r := httptest.NewRequest("POST", "/api/stop", strings.NewReader(`{"all":true}`))
	r.Host = "103.97.134.90:8788"
	r.Header.Set("Origin", "http://evil.example.com")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Origin lạ phải 403 kể cả khi phơi, được %d", w.Code)
	}
}

// Dò nhiều lần thì bị bắt chờ (429). Quan trọng gấp bội so với thời còn token:
// mật khẩu do người đặt nên entropy thấp hơn hẳn 128 bit ngẫu nhiên.
// Bộ đếm chống dò tồn tại để chặn DÒ MẬT KHẨU — nên phải đo đúng việc đó.
//
// Bản trước của test này khẳng định: 5 request /api/state KHÔNG cookie thì người
// đã đăng nhập hợp lệ bị 429. Nó ghi thẳng một cái lỗi thành hợp đồng — bất kỳ ai
// chạm được cổng cũng khoá được người đang dùng bằng 5 dòng curl, mà chẳng dò gì
// cả (ID phiên là chuỗi ngẫu nhiên, không đoán được). Xem lachan_test.go.
func TestDoMatKhauNhieuLanBiChan(t *testing.T) {
	s := newTestServer(t)
	s.exposed = true
	const host = "103.97.134.90:8788"

	for i := 0; i < 6; i++ {
		postLogin(t, s, host, "", "Admin", "sai-mat-khau")
	}
	w := postLogin(t, s, host, "", "Admin", "sai-mat-khau")
	if !strings.Contains(w.Body.String(), "thử lại sau") {
		t.Fatal("dò mật khẩu sai 7 lần mà không bị bắt chờ")
	}

	// Và người có cookie hợp lệ KHÔNG bị vạ lây.
	s2 := newTestServer(t)
	s2.exposed = true
	ck := dangNhap(t, s2, host)
	for i := 0; i < 6; i++ {
		postLogin(t, s2, host, "", "Admin", "sai-mat-khau")
	}
	r := httptest.NewRequest("GET", "/api/state", nil)
	r.Host = host
	r.AddCookie(ck)
	w2 := httptest.NewRecorder()
	s2.ServeHTTP(w2, r)
	if w2.Code == http.StatusTooManyRequests {
		t.Fatal("người đã đăng nhập bị khoá vì kẻ khác dò mật khẩu")
	}
}

func TestSplitArgsGiuNgoacKep(t *testing.T) {
	got := splitArgs(`-p "tóm tắt repo này" --json`)
	want := []string{"-p", "tóm tắt repo này", "--json"}
	if len(got) != len(want) {
		t.Fatalf("số đối số = %d, muốn %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("đối số %d = %q, muốn %q", i, got[i], want[i])
		}
	}
}
