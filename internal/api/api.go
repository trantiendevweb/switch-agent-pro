// Package api là HỢP ĐỒNG DUY NHẤT của hệ thống.
//
// MASTER-PLAN mục 2c luật 1: mọi hành động đi qua đây. CLI không phải là lõi —
// CLI chỉ là *client đầu tiên*. TUI, dashboard 2D, workflow board và 3D là các
// client ngang hàng, không mặt nào được gọi thẳng vào `store` hay `profile`.
//
// Hiện tại API chạy TRONG TIẾN TRÌNH (không có daemon — xem mục 2b). Khi làm
// `sagent dash`, lớp HTTP sẽ bọc đúng các phương thức này chứ không mở đường
// riêng, nhờ vậy bốn mặt luôn ngang quyền.
package api

import (
	"fmt"
	"os"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/config"
	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/fleet"
	"github.com/trantiendevweb/switch-agent-pro/internal/jsonutil"
	"github.com/trantiendevweb/switch-agent-pro/internal/process"
	"github.com/trantiendevweb/switch-agent-pro/internal/profile"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
	"github.com/trantiendevweb/switch-agent-pro/internal/workspace"
)

// Version của hợp đồng. Mặt nào cũng phải kiểm trước khi nói chuyện.
const Version = 1

// Actions là danh sách CHÍNH THỨC mọi hành động hệ thống làm được.
//
// Đây là thứ giữ luật "ngang quyền" (mục 2c luật 2) không thành khẩu hiệu:
// có test khẳng định MỌI hành động ở đây đều có lệnh CLI tương đương. Thêm nút
// trên dashboard mà quên lệnh CLI thì test đỏ.
var Actions = []string{
	"profile.list",
	"profile.create",
	"profile.remove",
	"profile.run",
	"profile.sync",
	"profile.verify",
	"session.list",
	"session.stop",
	"fleet.start",
	"clones.create",
	"clones.clean",
	"config.show",
	"config.init",
}

// API gom mọi thứ một mặt cần. Tạo bằng New, nhớ Close.
type API struct {
	db  *store.DB
	bus *events.Bus
	cfg config.Config
}

// New mở store và nạp cấu hình cho thư mục làm việc dir.
func New(dir string) (*API, error) {
	cfg, err := config.Load(dir)
	if err != nil {
		return nil, err
	}
	db, err := store.Open()
	if err != nil {
		return nil, fmt.Errorf("không mở được sổ trạng thái (%s): %w", store.Path(), err)
	}
	return &API{db: db, bus: events.NewBus(), cfg: cfg}, nil
}

func (a *API) Close() error { a.bus.Close(); return a.db.Close() }

// Events là luồng sự thật để mọi mặt bám vào.
func (a *API) Events() *events.Bus { return a.bus }

// Config trả về cấu hình đã gộp các tầng.
func (a *API) Config() config.Config { return a.cfg }

// Addr là địa chỉ hồ sơ "provider:account"; thiếu provider thì mặc định claude.
type Addr struct{ Provider, Account string }

func ParseAddr(s string) Addr {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return Addr{s[:i], s[i+1:]}
	}
	return Addr{"claude", s}
}

func (ad Addr) String() string { return ad.Provider + ":" + ad.Account }

func adapterOf(name string) (provider.Adapter, error) {
	ad, ok := provider.Get(name)
	if !ok {
		return nil, fmt.Errorf("không có provider '%s' (có: %s)", name, strings.Join(provider.Names(), ", "))
	}
	return ad, nil
}

// ---------------------------- hồ sơ ----------------------------

// Profile là một dòng trong bảng tài khoản.
type Profile struct {
	Provider string
	Account  string
	Dir      string
	Identity string // email; rỗng = chưa đăng nhập
	HasToken bool
	Active   bool // đang là tài khoản của tiến trình hiện tại
}

func (p Profile) Addr() string { return p.Provider + ":" + p.Account }

// ProfileList — action "profile.list".
func (a *API) ProfileList() ([]Profile, error) {
	accs, err := profile.List()
	if err != nil {
		return nil, err
	}
	out := make([]Profile, 0, len(accs))
	for _, acc := range accs {
		p := Profile{Provider: acc.Provider, Account: acc.Name, Dir: acc.Dir}
		if ad, ok := provider.Get(acc.Provider); ok {
			p.Identity = ad.Identity(acc.Dir)
			p.HasToken = ad.HasToken(acc.Dir)
			p.Active = currentConfigDir(ad) == strings.TrimRight(acc.Dir, `\/`)
		}
		out = append(out, p)
	}
	return out, nil
}

// ProfileCreate — action "profile.create".
func (a *API) ProfileCreate(addr Addr) (linked, seeded int, err error) {
	ad, err := adapterOf(addr.Provider)
	if err != nil {
		return 0, 0, err
	}
	linked, seeded, err = profile.Create(ad, addr.Account)
	if err == nil {
		a.bus.Publish(events.Event{
			Type: events.ProfileCreated, Addr: addr.String(),
			Msg: fmt.Sprintf("nối %d mục dùng chung, gieo %d khoá", linked, seeded),
		})
	}
	return
}

// ProfileRemove — action "profile.remove". Xoá an toàn (gỡ link trước).
func (a *API) ProfileRemove(addr Addr) error {
	dir, ok := profile.ResolveDir(addr.Provider, addr.Account)
	if !ok {
		return fmt.Errorf("không có %s", addr)
	}
	if err := profile.Remove(dir); err != nil {
		return err
	}
	a.bus.Publish(events.Event{Type: events.ProfileRemoved, Addr: addr.String(), Msg: "đã xoá tài khoản và token"})
	return nil
}

// ProfileRun — action "profile.run". Chạy CLI tương tác, CHIẾM terminal cho tới
// khi người dùng thoát. Chỉ mặt terminal gọi được; các mặt khác dùng fleet.
func (a *API) ProfileRun(addr Addr, args []string) error {
	ad, err := adapterOf(addr.Provider)
	if err != nil {
		return err
	}
	dir, ok := profile.ResolveDir(addr.Provider, addr.Account)
	if !ok {
		return fmt.Errorf("không có %s. Tạo: sagent them %s", addr, addr)
	}
	return profile.Run(ad, dir, args)
}

// RunRoot chạy bằng tài khoản gốc: XOÁ biến môi trường thay vì trỏ vào ~/.claude.
func (a *API) RunRoot(args []string) error {
	ad, err := adapterOf("claude")
	if err != nil {
		return err
	}
	return profile.Run(ad, "", args)
}

// SyncReport là kết quả đồng bộ một hồ sơ.
type SyncReport struct {
	Addr    string
	Changed int
	Skipped string // lý do bỏ qua, rỗng nếu không bỏ
	Err     error
}

// ProfileSync — action "profile.sync". Đẩy whitelist "thói quen máy" sang mọi hồ sơ.
func (a *API) ProfileSync(dryRun bool) ([]SyncReport, error) {
	accs, err := profile.List()
	if err != nil {
		return nil, err
	}
	var out []SyncReport
	for _, acc := range accs {
		r := SyncReport{Addr: acc.Provider + ":" + acc.Name}
		ad, ok := provider.Get(acc.Provider)
		if !ok {
			r.Skipped = "không có adapter"
			out = append(out, r)
			continue
		}
		dst := acc.Dir + string(pathSep) + ".claude.json"
		if !fileExists(dst) {
			r.Skipped = "chưa có .claude.json"
			out = append(out, r)
			continue
		}
		if dryRun {
			r.Skipped = "xem trước — chưa ghi"
			out = append(out, r)
			continue
		}
		r.Changed, r.Err = jsonutil.SyncKeys(ad.IdentitySource(), dst, ad.SharedKeys())
		out = append(out, r)
	}
	return out, nil
}

// ProfileVerify — action "profile.verify". Chạy bộ "đã đo" của adapter.
func (a *API) ProfileVerify(providerName string) (map[string][]provider.Check, error) {
	names := provider.Names()
	if providerName != "" {
		if _, err := adapterOf(providerName); err != nil {
			return nil, err
		}
		names = []string{providerName}
	}
	out := map[string][]provider.Check{}
	for _, n := range names {
		ad, _ := provider.Get(n)
		out[n] = ad.Verify()
	}
	return out, nil
}

// ---------------------------- phiên ----------------------------

// SessionList — action "session.list". Luôn đối chiếu PID thật.
func (a *API) SessionList() ([]store.Session, error) { return a.db.Running() }

// SessionStop — action "session.stop". id < 0 nghĩa là dừng tất cả.
func (a *API) SessionStop(id int64) (int, error) {
	list, err := a.db.Running()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, s := range list {
		if id >= 0 && s.ID != id {
			continue
		}
		if err := process.Kill(s.PID); err != nil {
			a.bus.Failuref("#%d (PID %d) không dừng được: %v", s.ID, s.PID, err)
			continue
		}
		_ = a.db.SetState(s.ID, store.StateStopped)
		a.bus.Publish(events.Event{
			Type: events.SessionStopped, Addr: s.Addr(), SessionID: s.ID, Msg: "đã dừng",
		})
		n++
	}
	return n, nil
}

// FleetRequest là yêu cầu bật hạm đội.
type FleetRequest struct {
	Addr     Addr
	Copies   int
	Worktree bool
	Args     []string // lệnh headless cho CLI con
}

// FleetStart — action "fleet.start". Áp chính sách của dự án TRƯỚC khi chạy.
func (a *API) FleetStart(req FleetRequest) (fleet.Result, error) {
	ad, err := adapterOf(req.Addr.Provider)
	if err != nil {
		return fleet.Result{}, err
	}
	// Cấu hình dự án quyết định khi người dùng không nói gì.
	if !req.Worktree && a.cfg.Project.Workspace == "worktree" {
		req.Worktree = true
	}
	// Trần cứng: chặn cả --copies lẫn TỔNG số phiên đang chạy, tránh lỡ tay
	// bật fleet nhiều lần rồi đốt hạn mức.
	if m := a.cfg.Policy.MaxParallelSessions; m > 0 {
		running, _ := a.db.Running()
		room := m - len(running)
		if room < 0 {
			room = 0
		}
		if req.Copies > room {
			a.bus.Warnf("policy.max_parallel_sessions=%d, đang chạy %d — hạ %d xuống %d.",
				m, len(running), req.Copies, room)
			req.Copies = room
		}
		if req.Copies == 0 {
			return fleet.Result{}, fmt.Errorf("đã đạt trần %d phiên đang chạy — dừng bớt rồi thử lại (sagent stop all)", m)
		}
	}
	return fleet.FanOut(a.db, a.bus, ad, req.Addr.Account, fleet.Opts{
		Copies: req.Copies, Worktree: req.Worktree,
	}, req.Args)
}

// ---------------------------- bản clone ----------------------------

// ClonesCreate — action "clones.create".
func (a *API) ClonesCreate(addr Addr, copies int) ([]string, error) {
	ad, err := adapterOf(addr.Provider)
	if err != nil {
		return nil, err
	}
	dirs, err := profile.Clone(ad, addr.Account, copies)
	if err == nil {
		a.bus.Publish(events.Event{
			Type: events.ClonesCreated, Addr: addr.String(),
			Msg: fmt.Sprintf("đã tạo %d thư mục cấu hình riêng", len(dirs)),
		})
	}
	return dirs, err
}

// CleanResult tóm tắt một lần dọn.
type CleanResult struct {
	WorktreesRemoved int
	WorktreesKept    []string // còn thay đổi chưa commit
	ClonesRemoved    int
}

// ClonesClean — action "clones.clean". Gỡ worktree + xoá bản clone, AN TOÀN.
//
// force=false thì KHÔNG gỡ worktree còn thay đổi chưa commit — việc agent làm
// dở là dữ liệu thật (nguyên tắc #3).
func (a *API) ClonesClean(addr Addr, workDir string, force bool) (CleanResult, error) {
	var res CleanResult

	// Không dọn khi còn phiên đang chạy trên tài khoản đó.
	if list, err := a.db.Running(); err == nil {
		for _, s := range list {
			if s.Provider == addr.Provider && s.Account == addr.Account {
				return res, fmt.Errorf("còn phiên #%d đang chạy trên %s — dừng trước: sagent stop all", s.ID, addr)
			}
		}
	}

	if repoRoot, ok := workspace.RepoRoot(workDir); ok {
		for _, dir := range workspace.FindAll(repoRoot, addr.Account) {
			if workspace.IsDirty(dir) && !force {
				res.WorktreesKept = append(res.WorktreesKept, dir)
				a.bus.Publish(events.Event{
					Type: events.WorktreeKept, Addr: addr.String(),
					Msg: "giữ lại — còn thay đổi chưa commit", Detail: map[string]string{"path": dir},
				})
				continue
			}
			if err := workspace.Remove(repoRoot, dir); err == nil {
				res.WorktreesRemoved++
				a.bus.Publish(events.Event{Type: events.WorktreeGone, Addr: addr.String(),
					Msg: "đã gỡ worktree", Detail: map[string]string{"path": dir}})
			}
		}
	}

	n, err := profile.CleanClones(addr.Provider, addr.Account)
	res.ClonesRemoved = n
	if err == nil {
		a.bus.Publish(events.Event{Type: events.ClonesCleaned, Addr: addr.String(),
			Msg: fmt.Sprintf("đã xoá %d bản clone", n)})
	}
	return res, err
}

// ---------------------------- lặt vặt ----------------------------

func currentConfigDir(ad provider.Adapter) string {
	return strings.TrimRight(os.Getenv(ad.EnvVar()), `\/`)
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

const pathSep = os.PathSeparator
