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
	"github.com/trantiendevweb/switch-agent-pro/internal/config"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

//go:embed web
var webFS embed.FS

// Server phục vụ dashboard. Tạo bằng New, chạy bằng Run.
type Server struct {
	api     *api.API
	mux     *http.ServeMux
	exposed bool // true = nghe ngoài loopback (phải chủ động bật)

	dungTLS  bool // phục vụ HTTPS bằng chứng chỉ tự ký
	chiuTran bool // người dùng CHỦ ĐỘNG chấp nhận HTTP trần khi phơi ra mạng

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
	// KHÔNG phục vụ thẳng bằng http.FileServer: embed.FS không có mốc thời gian
	// nên nó không phát ra van xác thực nào, và trình duyệt giữ bản cũ vô hạn.
	// Xem đầu cache.go — chuyện này đã ăn mất trọn một ngày sửa giao diện.
	files := tepNhung(sub, http.FileServer(http.FS(sub)))

	m := http.NewServeMux()
	m.HandleFunc("/api/state", s.guard(s.handleState))
	m.HandleFunc("/api/events", s.guard(s.handleEvents))
	m.HandleFunc("/api/fleet", s.guard(s.handleFleet))
	m.HandleFunc("/api/stop", s.guard(s.handleStop))
	m.HandleFunc("/api/quet", s.guard(s.handleQuet))
	m.HandleFunc("/api/db", s.guard(s.handleDB))
	m.HandleFunc("/api/ai", s.guard(s.handleAI))
	m.HandleFunc("/api/ai/lich-su", s.guard(s.handleAILichSu))
	m.HandleFunc("/api/so/ho-so", s.guard(s.handleSoHoSo))
	m.HandleFunc("/api/so/route", s.guard(s.handleSoRoute))
	m.HandleFunc("/api/route/kiem", s.guard(s.handleRouteKiem))
	m.HandleFunc("/api/tele", s.guard(s.handleTele))
	m.HandleFunc("/api/nang-luc", s.guard(s.handleNangLuc))

	m.HandleFunc("/api/flows", s.guard(s.handleFlows))
	m.HandleFunc("/api/run", s.guard(s.handleRun))
	m.HandleFunc("/api/flow/run", s.guard(s.handleFlowRun))
	m.HandleFunc("/api/flow/kho", s.guard(s.handleFlowKho))
	m.HandleFunc("/api/flow/decide", s.guard(s.handleFlowDecide))
	m.HandleFunc("/api/flow/detail", s.guard(s.handleFlowDetail))
	m.HandleFunc("/api/flow/tom-tat", s.guard(s.handleFlowTomTat))
	m.HandleFunc("/api/flow/cancel", s.guard(s.handleFlowCancel))
	m.HandleFunc("/api/flow/save", s.guard(s.handleFlowSave))
	m.HandleFunc("/api/flow/delete", s.guard(s.handleFlowDelete))
	m.HandleFunc("/api/flow/def", s.guard(s.handleFlowDef))

	m.HandleFunc("/login", s.handleLogin)
	m.HandleFunc("/logout", s.handleLogout)

	// /docs/ là vùng CÔNG KHAI: kế hoạch, thiết kế, master plan — chỉ để đọc.
	// Cố ý không đòi đăng nhập để chia sẻ link kế hoạch KHÔNG đồng nghĩa trao quyền
	// điều khiển agent. Đằng nào nội dung này cũng nằm công khai trên GitHub.
	m.Handle("/docs/", files)

	// /vendor/ cũng CÔNG KHAI: three.js + font nhúng sẵn trong binary. Trang /docs/
	// không đòi đăng nhập, nên font của nó cũng không được đòi — bắt đăng nhập ở
	// đây thì chữ trang kế hoạch rơi về font hệ điều hành. Đây là asset tĩnh,
	// không có bí mật nào trong đó.
	m.Handle("/vendor/", files)

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

// layNext đọc tham số ?next= và chỉ chấp nhận đường dẫn NỘI BỘ.
//
// Kiểm `strings.HasPrefix(next, "/")` là chưa đủ: "//evil.example" cũng bắt đầu
// bằng "/" nhưng trình duyệt hiểu nó là URL đổi tên miền (protocol-relative), và
// "/\evil.example" cũng vậy trên phần lớn trình duyệt. Người dùng bấm link đăng
// nhập rồi bị ném sang trang lừa đảo — mà URL trước đó đúng là dash của họ.
func layNext(r *http.Request) string {
	n := r.URL.Query().Get("next")
	if n == "" || !strings.HasPrefix(n, "/") {
		return "/"
	}
	if strings.HasPrefix(n, "//") || strings.HasPrefix(n, `/\`) {
		return "/"
	}
	return n
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
	next := layNext(r)

	// Ở chế độ kín, /login cũng phải kiểm Host loopback y như guard. Thiếu chỗ
	// này thì lớp chống DNS-rebind hở đúng cái cửa quan trọng nhất: tên miền của
	// kẻ tấn công trỏ về 127.0.0.1, trang của nó POST mật khẩu vào dash nội bộ,
	// và sameOrigin cho qua vì Origin lẫn Host đều là tên miền đó.
	if !s.exposed && !loopbackHost(r.Host) {
		http.Error(w, "chỉ phục vụ ở loopback", http.StatusForbidden)
		return
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
	setCookie(w, s.sess.create(), s.dungTLS)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Trang lạ điều hướng tới /logout thì cookie SameSite=Lax VẪN được gửi, nên
	// phiên bị xoá. Hại nhỏ — chỉ là bị đăng xuất — nhưng không có lý do để hở.
	//
	// Dùng Sec-Fetch-Site thay vì bắt POST: ba file HTML đang gọi bằng <a href>,
	// đổi hết sang form chỉ để chặn một trò chọc phá là đánh đổi tồi. Trình duyệt
	// hiện đại đều gắn header này; curl không gắn và vẫn dùng được như trước.
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		http.Error(w, "yêu cầu đến từ trang khác", http.StatusForbidden)
		return
	}
	s.sess.drop(cookieID(r))
	clearCookie(w, s.dungTLS)
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

// DungTLS bật/tắt HTTPS. ChiuHTTPTran là lối thoát CÓ Ý THỨC cho người chấp nhận
// gửi mật khẩu dạng trần (LAN kín, hoặc đã có đường hầm mã hoá ở ngoài).
func (s *Server) DungTLS(v bool) { s.dungTLS = v }
func (s *Server) ChiuHTTPTran()  { s.chiuTran = true }

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

	// Phơi ra mạng mà không mã hoá thì mọi thứ đã làm để bảo vệ mật khẩu — băm
	// PBKDF2 210k vòng, siết ACL, bỏ token khỏi URL — đều vô nghĩa: nó đi qua
	// dây dưới dạng chữ thường. Chặn ở ĐÂY chứ không chỉ cảnh báo ở CLI, vì đây
	// là chỗ không đi vòng được.
	if s.exposed && !s.dungTLS && !s.chiuTran {
		return errors.New("phơi ra mạng mà không có TLS — mật khẩu sẽ đi dạng trần.\n" +
			"     Bỏ cờ --http-tran để dùng HTTPS (mặc định), hoặc giữ nó nếu bạn đã có\n" +
			"     đường hầm mã hoá ở ngoài (SSH tunnel, VPN)")
	}

	var certFile, keyFile, vanTay string
	if s.dungTLS {
		var err error
		certFile, keyFile, vanTay, err = EnsureCert()
		if err != nil {
			return fmt.Errorf("không dựng được chứng chỉ TLS: %w", err)
		}
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		// Nói RÕ ai đang giữ cổng — xem đầu congban.go để biết vì sao chỗ này đáng
		// một file riêng.
		return loiCongBan(port, err)
	}

	giaoThuc := "http"
	if s.dungTLS {
		giaoThuc = "https"
	}

	fmt.Println()
	if s.exposed {
		fmt.Println("  ╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("  ║  ⚠  DASHBOARD ĐANG PHƠI RA MẠNG                              ║")
		fmt.Println("  ╚══════════════════════════════════════════════════════════════╝")
		fmt.Println("  Ai đăng nhập được đều BẬT/DỪNG được agent của bạn và tiêu hạn mức.")
		if s.dungTLS {
			fmt.Printf("  Mật khẩu của %q là hàng rào DUY NHẤT. Đường truyền đã mã hoá.\n", s.auth.User)
		} else {
			fmt.Printf("  Mật khẩu của %q là hàng rào DUY NHẤT — và HTTP KHÔNG mã hoá nó\n", s.auth.User)
			fmt.Println("  trên đường truyền (bạn đã chọn --http-tran).")
		}
		fmt.Println("  Xong việc thì Ctrl+C để đóng cổng.")
		fmt.Println()
		fmt.Printf("  Mở trên điện thoại/máy khác:\n    %s://<IP-máy-này>:%d/\n", giaoThuc, port)
	} else {
		fmt.Println("  Dashboard đang chạy — mở link này trên trình duyệt (cùng máy):")
		fmt.Printf("    %s://%s/\n", giaoThuc, ln.Addr().String())
		fmt.Println()
		fmt.Printf("  Chỉ nghe ở loopback; đăng nhập bằng tài khoản %q.\n", s.auth.User)
	}

	if s.dungTLS {
		fmt.Println()
		fmt.Println("  Chứng chỉ TỰ KÝ nên trình duyệt sẽ cảnh báo. Trước khi bấm \"vẫn tiếp tục\",")
		fmt.Println("  mở phần xem chứng chỉ và đối chiếu vân tay SHA-256 với dòng dưới đây.")
		fmt.Println("  Không đối chiếu thì TLS chỉ chống nghe lén, không chống kẻ đứng giữa.")
		fmt.Println()
		fmt.Printf("    %s\n", vanTay)
		fmt.Printf("    (chứng chỉ: %s)\n", certFile)
	}

	fmt.Println()
	fmt.Println("  Ctrl+C để dừng.")
	// http.Serve/ServeTLS dùng http.Server MẶC ĐỊNH: KHÔNG có hạn giờ nào cả.
	// Kẻ tấn công mở nhiều socket rồi nhỏ từng byte header là giữ mãi goroutine
	// và file descriptor — Slowloris. Nguy hiểm thật ở chế độ phơi ra mạng.
	//
	// ReadHeaderTimeout là cái chặn đúng trò đó. WriteTimeout KHÔNG đặt được:
	// /api/events là luồng SSE chạy dài, đặt vào là tự cắt tính năng của mình.
	srv := &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if s.dungTLS {
		return srv.ServeTLS(ln, certFile, keyFile)
	}
	return srv.Serve(ln)
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

		// KHÔNG throttle ở đây. Bộ đếm chỉ còn một nhiệm vụ: làm chậm việc DÒ MẬT
		// KHẨU, và chỗ dò mật khẩu là /login. Để nó ở guard thì kẻ tấn công gõ
		// sai mật khẩu vài lần là khoá luôn dashboard của người dùng hợp lệ —
		// biến lá chắn thành vũ khí chĩa vào đúng người nó bảo vệ. Đã đo bằng
		// TestDoMatKhauNhieuLanBiChan.

		// Một đường vào duy nhất: cookie phiên do form đăng nhập cấp.
		//
		// KHÔNG gọi noteFail ở đây. Bộ đếm này để chống DÒ MẬT KHẨU, mà request
		// không cookie thì chẳng dò gì cả — ID phiên là chuỗi ngẫu nhiên, không
		// đoán được. Trước đây có gọi, và hậu quả đo được: 8 dòng curl vô danh là
		// khoá luôn người đang đăng nhập hợp lệ (429). Một lá chắn quay ra đánh
		// đúng người nó bảo vệ.
		if !s.authorized(r) {
			// Trình duyệt gõ đường dẫn thì đưa tới form đăng nhập cho tử tế;
			// còn gọi API thì trả 401 để client biết mà xử lý.
			if s.auth != nil && r.Method == http.MethodGet && wantsHTML(r) {
				http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
				return
			}
			http.Error(w, "chưa đăng nhập", http.StatusUnauthorized)
			return
		}
		// Cũng KHÔNG gọi noteOK ở đây. Dashboard tự poll /api/state 5 giây một
		// lần, nên reset bộ đếm ở mọi request hợp lệ nghĩa là: chỉ cần có một tab
		// đang mở thì kẻ dò mật khẩu không bao giờ chạm ngưỡng. Chỉ ĐĂNG NHẬP
		// THÀNH CÔNG mới được xoá bộ đếm — xem handleLogin.

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
	HetHan   bool   `json:"hetHan"` // token còn đó nhưng đã quá hạn
	Active   bool   `json:"active"`
}

type sessionDTO struct {
	ID       int64  `json:"id"`
	Addr     string `json:"addr"`
	PID      int    `json:"pid"`
	Worktree string `json:"worktree,omitempty"`
	Log      string `json:"log,omitempty"`
	Started  int64  `json:"started"` // unix giây

	// State là trạng thái ĐÃ QUYẾT ở hợp đồng (api → store → provider). Mặt web
	// chỉ đọc lại, KHÔNG suy: trước đây index.html tự tính "running/pending" từ
	// pid, còn 3d.html tự đoán từ bước flow — hai mặt, hai luật, cùng một dữ
	// liệu. Nay chỉ có một luật, và nó nằm ngoài trình duyệt.
	State string `json:"state"`
	// LyDo/HanMucDenLai rỗng khi KHÔNG ĐO ĐƯỢC. Trang phải hiển thị đúng chỗ
	// trống đó chứ không được lấp bằng chữ suy ra.
	LyDo         string `json:"lyDo,omitempty"`
	HanMucDenLai int64  `json:"hanMucDenLai,omitempty"` // unix giây, 0 = không rõ
}

// uiDTO là phần `[ui]` của cấu hình, đưa ra cho mọi mặt web đọc chung.
//
// Vì sao đi kèm /api/state chứ không phải một endpoint riêng: mọi trang đã đọc
// ảnh chụp này ngay khi kết nối, nên thêm vào đây thì không trang nào phải nhớ
// fetch thêm một chỗ — trang nào quên sẽ lặng lẽ vẽ bằng bố cục mặc định mà
// không ai báo lỗi. Thêm endpoint mới còn kéo theo một action mới trong hợp
// đồng và một lệnh CLI tương ứng (test ngang quyền), trong khi `sagent config`
// đã trả lời đúng câu hỏi đó ở terminal rồi.
type uiDTO struct {
	DefaultSurface string   `json:"defaultSurface"`
	Theme          string   `json:"theme"`
	Columns        []string `json:"columns"`
	PinnedFlows    []string `json:"pinnedFlows"`
	Enable3D       bool     `json:"enable3d"`
}

// uiCuaCauHinh dựng uiDTO, và ĐÃ GIẢI SẴN giá trị mặc định.
//
// Cố ý giải ở server: nếu trả `columns: []` rồi để mỗi trang tự biết "rỗng thì
// dùng bốn cột kia", thì bộ mặc định bị chép làm nhiều bản trong JavaScript và
// sẽ trôi khỏi nhau. Một luật, một chỗ — trang chỉ việc vẽ đúng thứ nhận được.
func uiCuaCauHinh(c config.Config) uiDTO {
	cot := c.UI.Columns
	if len(cot) == 0 {
		cot = config.CotMacDinh
	}
	return uiDTO{
		DefaultSurface: c.UI.DefaultSurface,
		Theme:          c.UI.Theme,
		Columns:        cot,
		PinnedFlows:    c.UI.PinnedFlows,
		Enable3D:       c.UI.Enable3D,
	}
}

func phienDTO(s store.Session) sessionDTO {
	return sessionDTO{s.ID, s.Addr(), s.PID, s.Worktree, s.Log, s.Started.Unix(),
		s.State, s.StateLyDo, s.HanMucDenLai}
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
		ps = append(ps, profileDTO{p.Addr(), p.Provider, p.Account, p.Identity, p.HasToken, p.HetHan, p.Active})
	}
	// Phiên vừa chết BẤT THƯỜNG đi chung mảng `sessions` với phiên đang chạy,
	// phân biệt bằng trường `state`.
	//
	// Vì sao chung một mảng: nếu tách hai mảng thì mỗi trang phải nhớ đọc cả
	// hai, và trang nào quên sẽ mất hẳn cái card "hết hạn mức" mà không ai báo
	// lỗi — đúng kiểu lệch im lặng. Chung một mảng thì trang cũ vẫn vẽ được,
	// chỉ là vẽ thêm.
	hong, err := s.api.SessionHong(20)
	if err != nil {
		writeErr(w, err)
		return
	}
	ss := make([]sessionDTO, 0, len(sessions)+len(hong))
	for _, x := range sessions {
		ss = append(ss, phienDTO(x))
	}
	for _, x := range hong {
		ss = append(ss, phienDTO(x))
	}
	// runs: `flow.runs` khai đường vào từ web CHÍNH LÀ endpoint này, nên thiếu
	// nó thì mặt web không có cách nào biết lượt chạy nào đang chạy — mà test
	// ngang quyền vẫn xanh vì đường dẫn có tồn tại.
	runs, err := s.api.FlowRuns(20)
	if err != nil {
		writeErr(w, err)
		return
	}
	rs := make([]runDTO, 0, len(runs))
	for _, r := range runs {
		rs = append(rs, runDTO{r.ID, r.Flow, r.State, r.Started.Unix()})
	}
	writeJSON(w, map[string]any{
		"apiVersion": api.Version,
		"profiles":   ps,
		"sessions":   ss,
		"runs":       rs,
		"ui":         uiCuaCauHinh(s.api.Config()),
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

// handleQuet — action "session.sweep".
//
// Hai action `session.sweep` và `db.admin` đã nằm trong api.Actions từ lúc viết
// CLI, nhưng mặt web thì chưa có — tức luật "mọi mặt ngang quyền" (MASTER-PLAN
// mục 2c) đang bị vi phạm ngay trong repo có test canh nó. Test chỉ canh chiều
// CLI↔hợp đồng, không canh chiều dashboard.
//
// GIẾT phải là POST và phải xin tường minh. Windows dùng lại PID nên danh sách
// mồ côi có thể lẫn tiến trình vô can — mặc định chỉ liệt kê, y như CLI.
func (s *Server) handleQuet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Giet bool `json:"giet"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Giet && r.Method != http.MethodPost {
		writeErr(w, errors.New("dừng tiến trình phải dùng POST"))
		return
	}
	res, err := s.api.SessionSweep(req.Giet)
	if err != nil {
		writeErr(w, err)
		return
	}
	// DTO liệt kê tường minh từng trường — không ném thẳng struct nội bộ ra ngoài.
	type procDTO struct {
		PID    int    `json:"pid"`
		Ten    string `json:"ten"`
		BatDau string `json:"bat_dau"`
	}
	type mucDTO struct {
		SessionID int64     `json:"session_id"`
		Addr      string    `json:"addr"`
		PIDCu     int       `json:"pid_cu"`
		Procs     []procDTO `json:"procs"`
	}
	out := make([]mucDTO, 0, len(res))
	for _, m := range res {
		ps := make([]procDTO, 0, len(m.Procs))
		for _, p := range m.Procs {
			ps = append(ps, procDTO{PID: p.PID, Ten: p.Ten, BatDau: p.BatDau.Format("15:04:05 02/01")})
		}
		out = append(out, mucDTO{SessionID: m.Session.ID, Addr: m.Session.Addr(),
			PIDCu: m.Session.PID, Procs: ps})
	}
	writeJSON(w, map[string]any{"muc": out, "da_giet": req.Giet})
}

// handleTele — action "tele.notify" nhìn từ mặt web.
//
// GET trả trạng thái, POST gửi tin thử. Cố ý KHÔNG cho đặt token từ trình duyệt:
// token bot là bí mật, mà dash có chế độ phơi ra mạng (`--host 0.0.0.0`) — một
// secret gõ vào ô input sẽ nằm lại trong autofill, lịch sử và ảnh chụp màn hình.
// Đặt token là việc của terminal, đúng như tạo hồ sơ và đặt API key.
//
// Nhưng trạng thái và nút "gửi thử" thì phải có ở đây: người dùng cần biết đường
// báo tin còn thông TRƯỚC khi giao cho máy chạy flow qua đêm.
func (s *Server) handleTele(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := s.api.TeleThu(r.Context()); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"da_gui": true})
		return
	}
	da, chatID, duongDan := s.api.TeleTrangThai()
	// Không có trường nào cho token — DTO liệt kê tường minh chính là lá chắn.
	writeJSON(w, map[string]any{
		"da_cau_hinh": da,
		"chat_id":     chatID,
		"duong_dan":   duongDan,
		"bao_khi":     []string{"bước hỏng", "lượt chạy hỏng", "lượt chạy chờ duyệt", "lượt chạy xong"},
	})
}

// handleAI — action "api.call": gọi thẳng AI API.
//
// GET liệt kê route (không bao giờ trả key_id ra ngoài đã đủ, nhưng vẫn không trả
// key). POST mới gọi thật — vì lời gọi TIÊU TIỀN theo token, không phải thao tác
// đọc. Trả kèm usage để người bấm biết vừa tiêu gì.
func (s *Server) handleAI(w http.ResponseWriter, r *http.Request) {
	routes := s.api.AIRoutes()
	if r.Method != http.MethodPost {
		type rDTO struct {
			Ten     string `json:"ten"`
			BaseURL string `json:"base_url"`
			Model   string `json:"model"`
		}
		out := make([]rDTO, 0, len(routes))
		for _, x := range routes {
			// KHÔNG trả key_id: nó là tên file bí mật trong kho, không việc gì
			// phải để trình duyệt biết.
			out = append(out, rDTO{Ten: x.Ten, BaseURL: x.BaseURL, Model: x.Model})
		}
		writeJSON(w, map[string]any{"route": out})
		return
	}

	var req struct {
		Route  string `json:"route"`
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	kq, err := s.api.AICall(r.Context(), req.Route, req.Prompt)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]any{
		"noi_dung": kq.NoiDung,
		"model":    kq.Model,
		"usage": map[string]int{
			"vao": kq.Usage.Vao, "ra": kq.Usage.Ra, "tong": kq.Usage.Tong,
		},
		"giay": kq.Mat.Seconds(),
		// Route THẬT SỰ trả lời, và — nếu đã phải chuyển — route chính hỏng vì
		// gì, NGUYÊN VĂN kèm request id. Không trả về thì người bấm nút không
		// biết câu này đến từ nhà cung cấp nào và tiêu tiền của ai.
		"route":       kq.Route,
		"da_thu":      kq.DaThu,
		"route_chinh": kq.RouteChinh,
		"loi_chinh":   kq.LoiChinh,
	})
}

// handleAILichSu — action "api.history": sổ lời gọi API.
//
// CHỈ ĐỌC, và sổ không có prompt lẫn câu trả lời (xem migration v7). Nhờ vậy
// đây là endpoint duy nhất của mặt web đụng tới đường API mà không thể lỡ tay
// phơi nội dung hội thoại ra trình duyệt.
func (s *Server) handleAILichSu(w http.ResponseWriter, r *http.Request) {
	n := 20
	if v, err := strconv.Atoi(r.URL.Query().Get("n")); err == nil && v > 0 && v <= 200 {
		n = v
	}
	ds, err := s.api.AIHistory(n)
	if err != nil {
		writeErr(w, err)
		return
	}
	type dto struct {
		Luc       string  `json:"luc"`
		Route     string  `json:"route"`
		Model     string  `json:"model"`
		TokensIn  int     `json:"tokens_vao"`
		TokensOut int     `json:"tokens_ra"`
		CostUSD   float64 `json:"chi_phi_usd"`
		OK        bool    `json:"ok"`
		LyDo      string  `json:"ly_do,omitempty"`
		Giay      float64 `json:"giay"`
	}
	out := make([]dto, 0, len(ds))
	var tongVao, tongRa, hong int
	for _, g := range ds {
		if !g.OK {
			hong++
		}
		tongVao += g.TokensIn
		tongRa += g.TokensOut
		out = append(out, dto{
			Luc: g.Luc.Format(time.RFC3339), Route: g.Route, Model: g.Model,
			TokensIn: g.TokensIn, TokensOut: g.TokensOut, CostUSD: g.CostUSD,
			OK: g.OK, LyDo: g.LyDo, Giay: float64(g.Mili) / 1000,
		})
	}
	writeJSON(w, map[string]any{
		"muc":       out,
		"tong_vao":  tongVao,
		"tong_ra":   tongRa,
		"so_hong":   hong,
		"khong_luu": "sổ này không lưu câu hỏi và câu trả lời",
	})
}

// handleSoHoSo — action "profile.so". Đối chiếu SỔ ĐĂNG KÝ ↔ đĩa, chỉ đọc.
//
// Mặt web cố ý KHÔNG có nút xoá hồ sơ (xem danh sách miễn trừ trong
// lachan_test.go), nhưng nó vẫn phải NHÌN được cái sổ: khi terminal từ chối xoá
// vì "sổ không nhận sở hữu", đây là chỗ trả lời câu "vì sao". Một lời từ chối
// không giải thích được thì người dùng sẽ đi xoá tay — tức là mất luôn lá chắn.
func (s *Server) handleSoHoSo(w http.ResponseWriter, r *http.Request) {
	ds, err := s.api.ProfileSo()
	if err != nil {
		writeErr(w, err)
		return
	}
	type dto struct {
		Provider  string `json:"provider"`
		Account   string `json:"account"`
		DuongSo   string `json:"duong_so,omitempty"`
		DuongDia  string `json:"duong_dia,omitempty"`
		SoTaoRa   bool   `json:"sagent_tao_ra"`
		TrangThai string `json:"trang_thai"`
	}
	out := make([]dto, 0, len(ds))
	lech := 0
	for _, m := range ds {
		if m.Lech() {
			lech++
		}
		out = append(out, dto{m.Provider, m.Account, m.DuongSo, m.DuongDia, m.SoTaoRa, m.TrangThai})
	}
	writeJSON(w, map[string]any{"muc": out, "so_lech": lech})
}

// handleRouteKiem — action "route.kiem". Hỏi từng route CÓ DÙNG ĐƯỢC KHÔNG.
//
// Chạm mạng thật (GET /models ở nhà cung cấp) nhưng KHÔNG tốn token. Vì nó gọi
// ra ngoài nên đặt sau `guard` như mọi endpoint khác — một đường không cần đăng
// nhập mà gọi được ra Internet là một cái máy dò cổng miễn phí cho người lạ.
//
// DTO không mang key, chỉ key_id — y như `/api/so/route`.
func (s *Server) handleRouteKiem(w http.ResponseWriter, r *http.Request) {
	ds := s.api.RouteKiem(r.Context())
	type dto struct {
		Ten     string   `json:"ten"`
		Dung    bool     `json:"dung"`
		Song    bool     `json:"song"`
		Status  int      `json:"status,omitempty"`
		MatMs   int64    `json:"matMs"`
		Model   string   `json:"model,omitempty"`
		CoModel bool     `json:"coModel"`
		SoModel int      `json:"soModel,omitempty"`
		KhongRo bool     `json:"khongRo,omitempty"`
		Gan     []string `json:"gan,omitempty"`
		Loi     string   `json:"loi,omitempty"`
	}
	out := make([]dto, 0, len(ds))
	for _, k := range ds {
		out = append(out, dto{k.Ten, k.Dung(), k.Song, k.Status, k.Mat.Milliseconds(),
			k.Model, k.CoModel, k.SoModel, k.KhongRo, k.Gan, k.Loi})
	}
	writeJSON(w, map[string]any{"muc": out})
}


// handleSoRoute — action "route.list". Sổ route ↔ cấu hình, chỉ đọc.
//
// KHÔNG kèm key, chỉ key_id — sổ cũng chỉ giữ đúng chừng đó (migration v8).
func (s *Server) handleSoRoute(w http.ResponseWriter, r *http.Request) {
	ds, err := s.api.RouteList()
	if err != nil {
		writeErr(w, err)
		return
	}
	type dto struct {
		Ten            string `json:"ten"`
		BaseURLSo      string `json:"base_url_so,omitempty"`
		BaseURLCauHinh string `json:"base_url_cau_hinh,omitempty"`
		Model          string `json:"model,omitempty"`
		KeyID          string `json:"key_id,omitempty"`
		TrangThai      string `json:"trang_thai"`
	}
	out := make([]dto, 0, len(ds))
	for _, m := range ds {
		out = append(out, dto{m.Ten, m.BaseURLSo, m.BaseURLCauHinh, m.Model, m.KeyID, m.TrangThai})
	}
	writeJSON(w, map[string]any{"muc": out})
}

// handleNangLuc — action "provider.nang-luc". Bảng provider nào làm được gì.
//
// DTO liệt kê tường minh từng trường như mọi đường khác ở đây. Riêng bảng này
// thì không có gì phải giấu: nó chỉ mang KHOÁ, TRẠNG THÁI và BẰNG CHỨNG — toàn
// chữ viết sẵn trong mã nguồn, không đọc file hồ sơ nào, không chạm token nào.
// Nhờ vậy nó trả lời được cả khi chưa đăng nhập provider nào.
//
// `mo` (câu mô tả) đi kèm từ provider.MoiNangLuc chứ không chép sang JS: khoá
// `chon-model` mà mặt web tự đặt tên là "Chọn model" thì hai mặt nói hai kiểu,
// và sửa một bên sẽ quên bên kia.
func (s *Server) handleNangLuc(w http.ResponseWriter, r *http.Request) {
	ds, err := s.api.NangLuc(strings.TrimSpace(r.URL.Query().Get("provider")))
	if err != nil {
		writeErr(w, err)
		return
	}
	type mucDTO struct {
		Khoa      string `json:"khoa"`
		Mo        string `json:"mo"`
		TrangThai string `json:"trang_thai"`
		BangChung string `json:"bang_chung"`
	}
	type provDTO struct {
		Provider string   `json:"provider"`
		Muc      []mucDTO `json:"muc"`
		Lech     []string `json:"lech"`
	}
	mo := map[string]string{}
	for _, m := range provider.MoiNangLuc {
		mo[m.Khoa] = m.Mo
	}
	out := make([]provDTO, 0, len(ds))
	var soChuaDo int
	for _, p := range ds {
		d := provDTO{Provider: p.Provider, Muc: make([]mucDTO, 0, len(p.Muc)), Lech: p.Lech}
		if d.Lech == nil {
			d.Lech = []string{}
		}
		for _, m := range p.Muc {
			if m.TrangThai == provider.ChuaDo {
				soChuaDo++
			}
			d.Muc = append(d.Muc, mucDTO{m.Khoa, mo[m.Khoa], string(m.TrangThai), m.BangChung})
		}
		out = append(out, d)
	}
	// so_chua_do là con số người vận hành cần liếc: nó đếm những chỗ hệ thống
	// đang phải đoán. Để mặt web tự cộng thì mỗi mặt cộng một kiểu.
	writeJSON(w, map[string]any{"provider": out, "so_chua_do": soChuaDo})
}

// handleDB — action "db.admin", CHỈ phần đọc.
//
// `db backup` và `db restore` CỐ Ý không có ở đây: restore ghi đè chính file mà
// server đang mở, và backup thì nên đi cùng chỗ quyết định giữ bản sao ở đâu.
// Xem ghi chú ở api.Actions — mặt web trả lời được "đang ở đâu", còn "đổi cái gì"
// thì phải đứng ngoài mà làm.
func (s *Server) handleDB(w http.ResponseWriter, r *http.Request) {
	v, latest, path, err := s.api.DBInfo()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]any{
		"duong_dan":    path,
		"schema":       v,
		"schema_moi":   latest,
		"can_nang_cap": v < latest,
	})
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
