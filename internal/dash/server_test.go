package dash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
