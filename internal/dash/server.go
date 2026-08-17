// Package dash là mặt web: server localhost bọc internal/api, đẩy event realtime.
//
// Nó KHÔNG mở đường riêng vào store — mọi thứ đi qua api, đúng luật "một hợp
// đồng duy nhất" (MASTER-PLAN mục 2c). Nhờ vậy dashboard và CLI luôn ngang quyền.
//
// Bảo mật (mục 5b): chỉ bind loopback, đăng nhập bằng tên + mật khẩu (băm), chặn Host lạ
// (chống DNS-rebind) và Origin lạ (chống CSRF), và TUYỆT ĐỐI không gửi secret —
// mọi thứ trả về đều qua DTO liệt kê tường minh từng trường.
package dash

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
)

//go:embed web
var webFS embed.FS

// Server phục vụ dashboard. Tạo bằng New, chạy bằng Run.
type Server struct {
	api     *api.API
	mux     *http.ServeMux
	exposed bool // true = nghe ngoài loopback (phải chủ động bật)

	auth *Auth     // nil = chưa đặt mật khẩu → server từ chối chạy
	sess *sessions // phiên đăng nhập bằng cookie

	failMu sync.Mutex
	fails  int       // số lần sai mật khẩu liên tiếp
	failAt time.Time // lần sai gần nhất
}

// New dựng server. Cửa vào DUY NHẤT là form đăng nhập — tên + mật khẩu băm
// PBKDF2 ở ~/.ai-accounts/dash-auth.json. Không còn token trên URL: một secret
// nằm trong địa chỉ sẽ rơi vào log proxy, lịch sử trình duyệt và ảnh chụp màn hình.
func New(a *api.API) *Server {
	s := &Server{api: a, auth: LoadAuth(), sess: newSessions()}
	sub, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(sub))

	m := http.NewServeMux()
	m.HandleFunc("/api/state", s.guard(s.handleState))
	m.HandleFunc("/api/events", s.guard(s.handleEvents))
	m.HandleFunc("/api/fleet", s.guard(s.handleFleet))
	m.HandleFunc("/api/stop", s.guard(s.handleStop))

	m.HandleFunc("/api/flows", s.guard(s.handleFlows))
	m.HandleFunc("/api/run", s.guard(s.handleRun))
	m.HandleFunc("/api/flow/run", s.guard(s.handleFlowRun))
	m.HandleFunc("/api/flow/decide", s.guard(s.handleFlowDecide))
	m.HandleFunc("/api/flow/save", s.guard(s.handleFlowSave))
	m.HandleFunc("/api/flow/delete", s.guard(s.handleFlowDelete))
	m.HandleFunc("/api/flow/def", s.guard(s.handleFlowDef))

	m.HandleFunc("/login", s.handleLogin)
	m.HandleFunc("/logout", s.handleLogout)

	// /docs/ là vùng CÔNG KHAI: kế hoạch, thiết kế, master plan — chỉ để đọc.
	// Cố ý không đòi đăng nhập để chia sẻ link kế hoạch KHÔNG đồng nghĩa trao quyền
	// điều khiển agent. Đằng nào nội dung này cũng nằm công khai trên GitHub.
	m.Handle("/docs/", files)

	// Mọi thứ còn lại (dashboard 2D, 3D) cần đăng nhập. Riêng trang gốc thì đưa
	// tới form thay vì ném 401 trần trụi.
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && !s.authorized(r) {
			if s.auth != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			s.landing(w)
			return
		}
		s.guard(files.ServeHTTP)(w, r)
	})
	s.mux = m
	return s
}

// workDir là thư mục server được khởi động — cũng là gốc để tìm flows.toml.
func (s *Server) workDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// authorized: một cửa duy nhất — cookie phiên do form đăng nhập cấp.
func (s *Server) authorized(r *http.Request) bool {
	return s.sess.valid(cookieID(r))
}

// wantsHTML đoán request đến từ thanh địa chỉ trình duyệt (để chuyển hướng tới
// form) chứ không phải fetch/curl (để trả 401).
func wantsHTML(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

// ---------------------------- đăng nhập ----------------------------

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Error(w, "chưa đặt mật khẩu — chạy: sagent dash --set-password", http.StatusNotImplemented)
		return
	}
	next := r.URL.Query().Get("next")
	if next == "" || !strings.HasPrefix(next, "/") {
		next = "/" // chỉ nhận đường dẫn nội bộ, tránh open-redirect
	}

	if r.Method == http.MethodGet {
		if s.authorized(r) { // đã đăng nhập rồi thì vào thẳng
			http.Redirect(w, r, next, http.StatusSeeOther)
			return
		}
		s.loginPage(w, next, "")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "chỉ nhận GET/POST", http.StatusMethodNotAllowed)
		return
	}
	// Chống dò mật khẩu: dùng chung bộ đếm với lá chắn guard.
	if wait := s.throttle(); wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		s.loginPage(w, next, fmt.Sprintf("Sai nhiều lần — thử lại sau %d giây.", int(wait.Seconds())+1))
		return
	}
	if !sameOrigin(r) {
		http.Error(w, "origin không hợp lệ", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.loginPage(w, next, "Dữ liệu gửi lên không đọc được.")
		return
	}
	if !s.auth.Check(r.FormValue("user"), r.FormValue("password")) {
		s.noteFail()
		s.loginPage(w, next, "Sai tên đăng nhập hoặc mật khẩu.")
		return
	}
	s.noteOK()
	setCookie(w, s.sess.create())
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.sess.drop(cookieID(r))
	clearCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) loginPage(w http.ResponseWriter, next, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	banner := ""
	if errMsg != "" {
		banner = `<p class="err">` + html.EscapeString(errMsg) + `</p>`
	}
	warn := ""
	if s.exposed {
		warn = `<p class="warn">Đang phơi ra mạng qua HTTP — mật khẩu truyền đi <b>không mã hoá</b>.
		Chỉ dùng trên mạng bạn tin được, hoặc bọc qua SSH tunnel.</p>`
	}
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="vi"><head><meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>Đăng nhập — Switch-Agent-Pro</title><style>
body{margin:0;min-height:100vh;display:grid;place-items:center;background:#0F172A;color:#F8FAFC;
font-family:Inter,system-ui,-apple-system,"Segoe UI",sans-serif;padding:24px}
.c{width:100%%;max-width:340px}
h1{font-size:22px;margin:0 0 4px;display:flex;align-items:center;gap:9px}
p.s{color:#94A3B8;font-size:13px;margin:0 0 20px}
label{display:block;font-size:12px;color:#94A3B8;margin:12px 0 5px}
input{width:100%%;background:#0f172a;color:#F8FAFC;border:1px solid #475569;border-radius:9px;
padding:11px 12px;font:inherit;font-size:15px}
input:focus{outline:none;border-color:#22C55E;box-shadow:0 0 0 3px rgba(34,197,94,.18)}
button{width:100%%;margin-top:18px;background:#22C55E;color:#06240f;border:0;border-radius:9px;
padding:11px;font:inherit;font-size:15px;font-weight:700;cursor:pointer}
button:hover{opacity:.92}
.err{background:rgba(239,68,68,.12);border:1px solid rgba(239,68,68,.45);color:#fecaca;
border-radius:9px;padding:10px 12px;font-size:13px;margin:0 0 4px}
.warn{background:rgba(245,158,11,.1);border:1px solid rgba(245,158,11,.4);color:#fde68a;
border-radius:9px;padding:10px 12px;font-size:12px;line-height:1.5;margin:16px 0 0}
a{color:#7dd3fc;font-size:13px;display:inline-block;margin-top:16px;text-decoration:none}
</style></head><body><div class="c">
<h1><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#22C55E" stroke-width="1.8"><path d="M12 2l8.5 5v10L12 22 3.5 17V7z"/><circle cx="12" cy="12" r="3" fill="#22C55E" stroke="none"/></svg>Switch-Agent-Pro</h1>
<p class="s">Đăng nhập để vào dashboard điều khiển.</p>
%s
<form method="POST" action="/login?next=%s">
<label for="user">Tên đăng nhập</label>
<input id="user" name="user" autocomplete="username" autofocus required>
<label for="password">Mật khẩu</label>
<input id="password" name="password" type="password" autocomplete="current-password" required>
<button type="submit">Đăng nhập</button>
</form>
%s
<a href="/docs/">Xem kế hoạch (không cần đăng nhập) →</a>
</div></body></html>`, banner, url.QueryEscape(next), warn)
}

// landing chỉ hiện khi CHƯA đặt mật khẩu (server sẽ không chạy ở trạng thái đó,
// nên đây là lưới an toàn cho test): chỉ đường tới tài liệu công khai.
func (s *Server) landing(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `<!DOCTYPE html><html lang="vi"><head><meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Switch-Agent-Pro</title><style>
body{margin:0;min-height:100vh;display:grid;place-items:center;background:#0F172A;color:#F8FAFC;
font-family:Inter,system-ui,-apple-system,"Segoe UI",sans-serif;line-height:1.6;padding:24px}
.c{max-width:420px;text-align:center}h1{font-size:26px;margin:0 0 6px}
p{color:#94A3B8;font-size:14px}a{display:inline-block;margin-top:14px;text-decoration:none;color:#22C55E;
border:1px solid rgba(34,197,94,.45);background:rgba(34,197,94,.1);border-radius:9px;padding:9px 16px;font-weight:600}
code{background:#272F42;padding:2px 6px;border-radius:5px;font-size:.9em;color:#e2e8f0}
</style></head><body><div class="c">
<h1>Switch-Agent-Pro</h1>
<p>Control plane điều phối nhiều coding agent và nhiều AI API.</p>
<a href="/docs/">Xem kế hoạch &amp; thiết kế →</a>
<p style="margin-top:22px;font-size:13px">Dashboard chưa đặt mật khẩu. Trên máy chủ chạy:<br><code>sagent dash --set-password</code></p>
</div></body></html>`)
}

// ServeHTTP để test dùng httptest mà không cần mở cổng thật.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// Run bind vào host:port rồi phục vụ tới khi tiến trình dừng.
//
// host ngoài loopback = CHẾ ĐỘ PHƠI RA MẠNG. Lúc đó mật khẩu là hàng rào DUY
// NHẤT, nên phải in cảnh báo thật rõ chứ không được để người dùng tưởng nó kín.
func (s *Server) Run(host string, port int) error {
	// Token đã bị bỏ, nên chưa đặt mật khẩu nghĩa là KHÔNG còn cửa nào. Thà
	// không chạy còn hơn mở một dashboard điều khiển agent mà ai vào cũng được.
	if s.auth == nil {
		return errors.New("chưa đặt mật khẩu dashboard — chạy: sagent dash --set-password")
	}
	s.exposed = !isLoopbackAddr(host)

	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("không mở được cổng %d: %w (thử --port khác)", port, err)
	}

	fmt.Println()
	if s.exposed {
		fmt.Println("  ╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("  ║  ⚠  DASHBOARD ĐANG PHƠI RA MẠNG                              ║")
		fmt.Println("  ╚══════════════════════════════════════════════════════════════╝")
		fmt.Println("  Ai đăng nhập được đều BẬT/DỪNG được agent của bạn và tiêu hạn mức.")
		fmt.Printf("  Mật khẩu của %q là hàng rào DUY NHẤT — và HTTP KHÔNG mã hoá nó\n", s.auth.User)
		fmt.Println("  trên đường truyền. Xong việc thì Ctrl+C để đóng cổng.")
		fmt.Println()
		fmt.Printf("  Mở trên điện thoại/máy khác:\n    http://<IP-máy-này>:%d/\n", port)
	} else {
		fmt.Println("  Dashboard đang chạy — mở link này trên trình duyệt (cùng máy):")
		fmt.Printf("    http://%s/\n", ln.Addr().String())
		fmt.Println()
		fmt.Printf("  Chỉ nghe ở loopback; đăng nhập bằng tài khoản %q.\n", s.auth.User)
	}
	fmt.Println()
	fmt.Println("  Ctrl+C để dừng.")
	return http.Serve(ln, s)
}

func isLoopbackAddr(host string) bool {
	if host == "" {
		return false // rỗng = nghe mọi giao diện
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return host == "localhost"
}

// ---------------------------- lá chắn ----------------------------

func (s *Server) guard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Ở chế độ kín, Host phải là loopback (chống DNS-rebind: tên miền lạ trỏ
		// về 127.0.0.1). Ở chế độ phơi ra mạng thì Host là IP/tên miền thật nên
		// không kiểm được — lúc đó mật khẩu + cookie SameSite=Lax là hàng rào.
		if !s.exposed && !loopbackHost(r.Host) {
			http.Error(w, "chỉ phục vụ ở loopback", http.StatusForbidden)
			return
		}

		// Chống dò mật khẩu: sai nhiều lần liên tiếp thì bắt chờ.
		if wait := s.throttle(); wait > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
			http.Error(w, "thử lại sau", http.StatusTooManyRequests)
			return
		}

		// Một đường vào duy nhất: cookie phiên do form đăng nhập cấp.
		if !s.authorized(r) {
			s.noteFail()
			// Trình duyệt gõ đường dẫn thì đưa tới form đăng nhập cho tử tế;
			// còn gọi API thì trả 401 để client biết mà xử lý.
			if s.auth != nil && r.Method == http.MethodGet && wantsHTML(r) {
				http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
				return
			}
			http.Error(w, "chưa đăng nhập", http.StatusUnauthorized)
			return
		}
		s.noteOK()

		// Chống CSRF: mutation phải CÙNG GỐC với chính request (đúng cho cả hai
		// chế độ). Không có Origin = client dòng lệnh, đã qua cửa token là đủ.
		if r.Method == http.MethodPost && !sameOrigin(r) {
			http.Error(w, "origin không hợp lệ", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

// throttle trả về thời gian còn phải chờ, sau khi đã sai mật khẩu nhiều lần.
func (s *Server) throttle() time.Duration {
	s.failMu.Lock()
	defer s.failMu.Unlock()
	if s.fails < 5 {
		return 0
	}
	// sai 5 lần trở lên: khoá 2 giây, tăng dần tới tối đa 30 giây
	d := time.Duration(min(s.fails-4, 15)) * 2 * time.Second
	if left := d - time.Since(s.failAt); left > 0 {
		return left
	}
	return 0
}

func (s *Server) noteFail() {
	s.failMu.Lock()
	s.fails++
	s.failAt = time.Now()
	s.failMu.Unlock()
}

func (s *Server) noteOK() {
	s.failMu.Lock()
	s.fails = 0
	s.failMu.Unlock()
}

func loopbackHost(host string) bool {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	h = strings.TrimPrefix(strings.TrimSuffix(h, "]"), "[")
	switch h {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
}

// sameOrigin: Origin (nếu có) phải trùng đúng Host của chính request này.
// Đúng cho cả chế độ kín lẫn phơi ra mạng.
func sameOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return true // curl / client dòng lệnh: đã qua cửa đăng nhập là đủ
	}
	u, err := url.Parse(o)
	if err != nil {
		return false
	}
	return u.Host == r.Host
}

// ---------------------------- DTO (allowlist, không secret) ----------------------------

type profileDTO struct {
	Addr     string `json:"addr"`
	Provider string `json:"provider"`
	Account  string `json:"account"`
	Identity string `json:"identity"`
	HasToken bool   `json:"hasToken"`
	Active   bool   `json:"active"`
}

type sessionDTO struct {
	ID       int64  `json:"id"`
	Addr     string `json:"addr"`
	PID      int    `json:"pid"`
	Worktree string `json:"worktree,omitempty"`
	Log      string `json:"log,omitempty"`
	Started  int64  `json:"started"` // unix giây
}

// ---------------------------- endpoint ----------------------------

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.api.ProfileList()
	if err != nil {
		writeErr(w, err)
		return
	}
	sessions, err := s.api.SessionList()
	if err != nil {
		writeErr(w, err)
		return
	}
	ps := make([]profileDTO, 0, len(profiles))
	for _, p := range profiles {
		ps = append(ps, profileDTO{p.Addr(), p.Provider, p.Account, p.Identity, p.HasToken, p.Active})
	}
	ss := make([]sessionDTO, 0, len(sessions))
	for _, s := range sessions {
		ss = append(ss, sessionDTO{s.ID, s.Addr(), s.PID, s.Worktree, s.Log, s.Started.Unix()})
	}
	writeJSON(w, map[string]any{
		"apiVersion": api.Version,
		"profiles":   ps,
		"sessions":   ss,
		"now":        time.Now().Unix(),
	})
}

// handleEvents là luồng SSE. Client PHẢI đọc /api/state để có ảnh chụp đầy đủ khi
// kết nối, rồi dùng event để cập nhật — vì người nghe chậm có thể lỡ event.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "server không hỗ trợ streaming", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, cancel := s.api.Events().Subscribe(256)
	defer cancel()

	fmt.Fprint(w, ": connected\n\n")
	fl.Flush()

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			b, _ := json.Marshal(e)
			fmt.Fprintf(w, "data: %s\n\n", b)
			fl.Flush()
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n") // giữ kết nối, comment SSE
			fl.Flush()
		}
	}
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID  int64 `json:"id"`
		All bool  `json:"all"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	id := req.ID
	if req.All {
		id = -1
	}
	n, err := s.api.SessionStop(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]int{"stopped": n})
}

func (s *Server) handleFleet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Addr     string `json:"addr"`
		Copies   int    `json:"copies"`
		Worktree bool   `json:"worktree"`
		Command  string `json:"command"` // phần sau "--", dạng một chuỗi
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	res, err := s.api.FleetStart(api.FleetRequest{
		Addr:     api.ParseAddr(req.Addr),
		Copies:   req.Copies,
		Worktree: req.Worktree,
		Args:     splitArgs(req.Command),
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]any{"started": res.Started, "wanted": res.Wanted})
}

// ---------------------------- lặt vặt ----------------------------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// splitArgs tách chuỗi lệnh thành đối số, biết tôn trọng dấu ngoặc kép để
// -p "vài chữ" vẫn là một đối số.
func splitArgs(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case (r == ' ' || r == '\t') && !inQuote:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}
