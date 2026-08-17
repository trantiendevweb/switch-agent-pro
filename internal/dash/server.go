// Package dash là mặt web: server localhost bọc internal/api, đẩy event realtime.
//
// Nó KHÔNG mở đường riêng vào store — mọi thứ đi qua api, đúng luật "một hợp
// đồng duy nhất" (MASTER-PLAN mục 2c). Nhờ vậy dashboard và CLI luôn ngang quyền.
//
// Bảo mật (mục 5b): chỉ bind loopback, token ngẫu nhiên trong URL, chặn Host lạ
// (chống DNS-rebind) và Origin lạ (chống CSRF), và TUYỆT ĐỐI không gửi secret —
// mọi thứ trả về đều qua DTO liệt kê tường minh từng trường.
package dash

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
)

//go:embed web
var webFS embed.FS

// Server phục vụ dashboard. Tạo bằng New, chạy bằng Run.
type Server struct {
	api   *api.API
	token string
	mux   *http.ServeMux
}

// New dựng server với token ngẫu nhiên mới.
func New(a *api.API) *Server {
	s := &Server{api: a, token: randToken()}
	sub, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(sub))

	m := http.NewServeMux()
	m.HandleFunc("/api/state", s.guard(s.handleState))
	m.HandleFunc("/api/events", s.guard(s.handleEvents))
	m.HandleFunc("/api/fleet", s.guard(s.handleFleet))
	m.HandleFunc("/api/stop", s.guard(s.handleStop))
	m.HandleFunc("/", s.guard(files.ServeHTTP))
	s.mux = m
	return s
}

// Token là mã ngẫu nhiên phải có trong mọi request.
func (s *Server) Token() string { return s.token }

// ServeHTTP để test dùng httptest mà không cần mở cổng thật.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// Run bind loopback, in URL kèm token, rồi phục vụ tới khi tiến trình dừng.
func (s *Server) Run(host string, port int) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("không mở được cổng %d: %w (thử --port khác)", port, err)
	}
	u := fmt.Sprintf("http://%s/?t=%s", ln.Addr().String(), s.token)
	fmt.Println()
	fmt.Println("  Dashboard đang chạy — mở link này trên trình duyệt (cùng máy):")
	fmt.Println("    " + u)
	fmt.Println()
	fmt.Println("  Chỉ nghe ở loopback; link có token ngẫu nhiên. Ctrl+C để dừng.")
	return http.Serve(ln, s)
}

// ---------------------------- lá chắn ----------------------------

func (s *Server) guard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Chống DNS-rebind: Host phải là loopback, không nhận tên miền lạ trỏ về 127.0.0.1.
		if !loopbackHost(r.Host) {
			http.Error(w, "chỉ phục vụ ở loopback", http.StatusForbidden)
			return
		}
		// Token: header trước (JS đặt), hoặc query (lần mở link đầu tiên).
		tok := r.Header.Get("X-Sagent-Token")
		if tok == "" {
			tok = r.URL.Query().Get("t")
		}
		if subtle.ConstantTimeCompare([]byte(tok), []byte(s.token)) != 1 {
			http.Error(w, "token sai hoặc thiếu", http.StatusUnauthorized)
			return
		}
		// Chống CSRF: mutation phải cùng gốc loopback (hoặc không có Origin = client dòng lệnh).
		if r.Method == http.MethodPost && !sameLoopbackOrigin(r) {
			http.Error(w, "origin không hợp lệ", http.StatusForbidden)
			return
		}
		h(w, r)
	}
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

func sameLoopbackOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return true // curl / client dòng lệnh: đã qua cửa token là đủ
	}
	u, err := url.Parse(o)
	if err != nil {
		return false
	}
	return loopbackHost(u.Host) && u.Host == r.Host
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

func randToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
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
