package dash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
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

// ---------------------------- token/chi phí THẬT của phiên ----------------------------

// Bản ghi THẬT của `claude -p --output-format stream-json --verbose` (đo 18/08,
// xem internal/provider/ketqua_claude_test.go). Trường đáng giá ở đây là
// `total_cost_usd` — Claude là provider DUY NHẤT khai nó.
const logClaudeCoGia = `{"type":"system","subtype":"init"}
{"is_error":false,"num_turns":1,"stop_reason":"end_turn","total_cost_usd":0.08446,` +
	`"usage":{"input_tokens":2,"output_tokens":4},"permission_denials":[],` +
	`"terminal_reason":"completed","subtype":"success","api_error_status":null,` +
	`"result":"OK","type":"result"}`

// Bản ghi THẬT của `cursor-agent -p --output-format stream-json` (đo 21/08). Có
// `usage` camelCase, KHÔNG có trường giá nào.
const logCursorKhongGia = `{"type":"system","subtype":"init","session_id":"a05639"}
{"type":"result","subtype":"success","duration_ms":3879,"is_error":false,"result":"OK",` +
	`"session_id":"a0563938","request_id":"9d4f016c",` +
	`"usage":{"inputTokens":8445,"outputTokens":31,"cacheReadTokens":5632,"cacheWriteTokens":0}}`

// phienDTOTest là hình dạng ô token/chi phí trên dây. `CostUSD` là CON TRỎ vì
// bài kiểm này hỏi đúng câu "trường có mặt hay không", chứ không phải "trường
// bằng mấy": nil = /api/state không gửi ô đó = chưa đo.
type phienDTOTest struct {
	Addr      string   `json:"addr"`
	State     string   `json:"state"`
	TokensIn  int      `json:"tokensIn"`
	TokensOut int      `json:"tokensOut"`
	CostUSD   *float64 `json:"costUsd"`
}

// themPhienCoLog ghi một nhật ký ra đĩa rồi ghi phiên trỏ vào đó thẳng vào sổ.
//
// Cố ý KHÔNG đi qua `fleet.start`: bật agent thật trong test là đốt hạn mức, mà
// thứ cần kiểm ở đây là ĐƯỜNG ĐỌC. PID quyết định phiên sống hay chết — PID của
// chính tiến trình test thì phiên ở lại `running`, còn một PID bịa thật to thì
// vòng quét đánh dấu nó đã kết thúc.
func themPhienCoLog(t *testing.T, taiKhoan, nhaCungCap, noiDung string, pid int) {
	t.Helper()
	duong := filepath.Join(t.TempDir(), "phien.ndjson")
	if err := os.WriteFile(duong, []byte(noiDung), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.AddSession(store.Session{
		Provider: nhaCungCap, Account: taiKhoan, Dir: "d", PID: pid, Log: duong,
	}); err != nil {
		t.Fatal(err)
	}
}

// dsPhien gọi /api/state qua đúng cửa đăng nhập và trả về map addr → ô đo được.
func dsPhien(t *testing.T, s *Server) map[string]phienDTOTest {
	t.Helper()
	r := req("GET", "/api/state")
	r.Host = "127.0.0.1:7788"
	r.AddCookie(dangNhap(t, s, "127.0.0.1:7788"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("/api/state: mã %d — %s", w.Code, w.Body.String())
	}
	var got struct {
		Sessions []phienDTOTest `json:"sessions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	m := map[string]phienDTOTest{}
	for _, p := range got.Sessions {
		m[p.Addr] = p
	}
	return m
}

// Token/chi phí của phiên CLI phải là SỐ THẬT đọc từ nhật ký, không phải ô trống
// vĩnh viễn.
//
// Trước ô này, sessionDTO chỉ mang id/addr/pid/worktree/log/started/state, nên
// mặt web KHÔNG CÓ CÁCH NÀO biết một phiên đã tiêu bao nhiêu — cả con số ấy nằm
// sẵn trong nhật ký, đọc được bằng đúng bộ đọc mà `state` đang dùng.
func TestPhienClaudeMangTokenVaChiPhiThat(t *testing.T) {
	s := newTestServer(t)
	themPhienCoLog(t, "cogia", "claude", logClaudeCoGia, 0x7FFFFFF0)

	p, co := dsPhien(t, s)["claude:cogia"]
	if !co {
		t.Fatal("/api/state không trả phiên đã kết thúc")
	}
	if p.TokensIn != 2 || p.TokensOut != 4 {
		t.Errorf("token = %d/%d, muốn 2/4 — số nằm sẵn trong nhật ký mà DTO không lấy",
			p.TokensIn, p.TokensOut)
	}
	if p.CostUSD == nil {
		t.Fatal("Claude khai total_cost_usd mà /api/state không gửi ô costUsd — " +
			"mặt web sẽ ghi \"chưa đo\" cho một con số đã đo được")
	}
	if *p.CostUSD != 0.08446 {
		t.Errorf("chi phí = %v, muốn 0.08446", *p.CostUSD)
	}
}

// Provider KHÔNG khai giá thì ô chi phí phải VẮNG MẶT hẳn trên dây.
//
// Đây là nửa còn lại của cùng một luật, và là nửa dễ mất nhất: gửi `costUsd: 0`
// thì JSON vẫn hợp lệ, trang vẫn vẽ, không ai báo lỗi — chỉ có điều mọi lượt
// Cursor/Codex hiện thành "tốn 0đ". Token thì ngược lại: Cursor CÓ khai, nên nó
// phải ra tới nơi. Hai ô của cùng một phiên, hai câu trả lời khác nhau.
func TestPhienCursorChuaDoChiPhiThiKhongGuiSoKhong(t *testing.T) {
	s := newTestServer(t)
	themPhienCoLog(t, "khonggia", "cursor", logCursorKhongGia, 0x7FFFFFF1)

	p, co := dsPhien(t, s)["cursor:khonggia"]
	if !co {
		t.Fatal("/api/state không trả phiên đã kết thúc")
	}
	if p.CostUSD != nil {
		t.Errorf("Cursor không có trường giá mà /api/state vẫn gửi costUsd = %v — "+
			"đó là \"chưa đo\" bị hoá trang thành một hoá đơn", *p.CostUSD)
	}
	if p.TokensIn != 8445 || p.TokensOut != 31 {
		t.Errorf("token = %d/%d, muốn 8445/31 — chưa đo được GIÁ không có nghĩa là "+
			"chưa đo được TOKEN", p.TokensIn, p.TokensOut)
	}
}

// Nhật ký của phiên ĐANG CHẠY thì KHÔNG được đọc.
//
// File đó agent còn đang ghi: dòng cuối có thể mới ra được nửa, mà với Claude thì
// chính dòng cuối (`{"type":"result"}`) mới mang usage. Đọc dở thì hoặc trả về số
// của một lượt chưa xong, hoặc chết ở bước phân tích JSON — cả hai đều tệ hơn là
// chưa nói gì. Nhật ký dựng ở đây CỐ Ý là một bản ghi đầy đủ, để bài kiểm hỏi
// đúng một câu: DTO có tự kiềm chế theo `state` hay chỉ thấy file là đọc.
func TestPhienDangChayKhongBiDocNhatKyDangGhiDo(t *testing.T) {
	s := newTestServer(t)
	themPhienCoLog(t, "dangchay", "claude", logClaudeCoGia, os.Getpid())

	p, co := dsPhien(t, s)["claude:dangchay"]
	if !co {
		t.Fatal("/api/state không trả phiên đang chạy")
	}
	if p.State != store.StateRunning {
		t.Fatalf("state = %q, muốn %q — bài kiểm này cần một phiên CÒN SỐNG",
			p.State, store.StateRunning)
	}
	if p.CostUSD != nil || p.TokensIn != 0 || p.TokensOut != 0 {
		t.Errorf("phiên đang chạy vẫn bị đọc nhật ký: token %d/%d, chi phí %v",
			p.TokensIn, p.TokensOut, p.CostUSD)
	}
}
