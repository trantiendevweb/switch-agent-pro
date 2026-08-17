package dash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
)

// newTestServer dựng một server thật (API + store trong HOME tạm) để test lá chắn.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	a, err := api.New(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return New(a)
}

func req(method, target string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	r.Host = "127.0.0.1:4600" // giả lập loopback
	return r
}

// Không có token thì mọi cửa đều đóng.
func TestThieuTokenBiChan(t *testing.T) {
	s := newTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req("GET", "/api/state"))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("không token phải 401, được %d", w.Code)
	}
}

func TestTokenSaiBiChan(t *testing.T) {
	s := newTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req("GET", "/api/state?t=saibet"))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("token sai phải 401, được %d", w.Code)
	}
}

// Đúng token thì /api/state trả JSON có mảng profiles + sessions.
func TestStateTraJSON(t *testing.T) {
	s := newTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req("GET", "/api/state?t="+s.Token()))
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
	r := httptest.NewRequest("GET", "/api/state?t="+s.Token(), nil)
	r.Host = "evil.example.com"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Host lạ phải 403, được %d", w.Code)
	}
}

// Chống CSRF: POST từ gốc khác (dù đúng token) bị chặn.
func TestOriginLaBiChanTrenPOST(t *testing.T) {
	s := newTestServer(t)
	r := httptest.NewRequest("POST", "/api/stop?t="+s.Token(), strings.NewReader(`{"all":true}`))
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Origin lạ trên POST phải 403, được %d", w.Code)
	}
}

// POST cùng gốc loopback thì qua được (stop all với 0 phiên trả stopped:0).
func TestPOSTcungGocQua(t *testing.T) {
	s := newTestServer(t)
	r := httptest.NewRequest("POST", "/api/stop?t="+s.Token(), strings.NewReader(`{"all":true}`))
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Origin", "http://127.0.0.1:4600")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("POST cùng gốc phải 200, được %d — %s", w.Code, w.Body.String())
	}
}

// Chế độ phơi ra mạng: Host là IP thật thì phải QUA, nhưng token vẫn bắt buộc.
func TestCheDoPhoiChoHostThatNhungVanDoiToken(t *testing.T) {
	s := newTestServer(t)
	s.exposed = true // như khi chạy --host 0.0.0.0

	r := httptest.NewRequest("GET", "/api/state?t="+s.Token(), nil)
	r.Host = "103.97.134.90:8788"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("phơi ra mạng + đúng token phải 200, được %d", w.Code)
	}

	r2 := httptest.NewRequest("GET", "/api/state", nil)
	r2.Host = "103.97.134.90:8788"
	w2 := httptest.NewRecorder()
	s.ServeHTTP(w2, r2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("phơi ra mạng mà thiếu token phải 401, được %d", w2.Code)
	}
}

// CSRF vẫn phải chặn khi phơi ra mạng: Origin lạ không được POST.
func TestCheDoPhoiVanChanOriginLa(t *testing.T) {
	s := newTestServer(t)
	s.exposed = true
	r := httptest.NewRequest("POST", "/api/stop?t="+s.Token(), strings.NewReader(`{"all":true}`))
	r.Host = "103.97.134.90:8788"
	r.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Origin lạ phải 403 kể cả khi phơi, được %d", w.Code)
	}
}

// Dò token nhiều lần thì bị bắt chờ (429) — token 128-bit vốn đã khó dò, đây là
// lớp phòng thủ thêm khi cổng nằm ngoài internet.
func TestDoTokenNhieuLanBiChan(t *testing.T) {
	s := newTestServer(t)
	s.exposed = true
	for i := 0; i < 5; i++ {
		r := httptest.NewRequest("GET", "/api/state?t=sai"+strconv.Itoa(i), nil)
		r.Host = "103.97.134.90:8788"
		s.ServeHTTP(httptest.NewRecorder(), r)
	}
	r := httptest.NewRequest("GET", "/api/state?t="+s.Token(), nil)
	r.Host = "103.97.134.90:8788"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("sau 5 lần sai phải bị 429, được %d", w.Code)
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
