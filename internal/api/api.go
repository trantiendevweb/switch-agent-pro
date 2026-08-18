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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/acl"
	"github.com/trantiendevweb/switch-agent-pro/internal/config"
	"github.com/trantiendevweb/switch-agent-pro/internal/drift"
	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/fleet"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
	"github.com/trantiendevweb/switch-agent-pro/internal/jsonutil"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
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
	// Phiên bản binary. Đáng nằm trong hợp đồng chứ không phải một dòng in ở CLI:
	// khi người dùng báo lỗi, câu hỏi đầu tiên luôn là "bản nào" — và dashboard
	// nên trả lời được câu đó mà không bắt họ mở terminal.
	"config.version",
	// Quét tiến trình mồ côi: phiên tự chết thì `session.list` không còn thấy nó,
	// nhưng đám con nó đẻ ra có thể vẫn chạy và vẫn tiêu hạn mức. Không có hành
	// động này thì không mặt nào nhìn ra chúng.
	"session.sweep",
	"dash.serve",
	// db.admin cùng loại với dash.serve: có mặt trong hợp đồng, nhưng mặt web
	// KHÔNG tự làm được phần nặng của nó. `db restore` ghi đè chính file mà
	// server đang mở — muốn làm từ web thì server phải tự đóng DB dưới chân
	// mình rồi tin rằng mình mở lại được. Xem/sao lưu thì mặt khác làm được và
	// nên làm; khôi phục thì phải đứng ngoài mà làm.
	"db.admin",
	"flow.list",
	"flow.show",
	"flow.validate",
	"flow.run",
	"flow.runs",
	"flow.approve",
	"flow.save",
	"flow.delete",
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

// Validate chặn tên hồ sơ nguy hiểm ngay ở biên, TRƯỚC khi nó thành đường dẫn.
//
// Tên đến từ người dùng ở cả bốn mặt điều khiển, kể cả form trên dashboard đang
// mở ra internet. Đo được: `claude:../../.claude` cho ra đúng ~/.claude, tức là
// `sagent xoa` sẽ xoá dữ liệu Claude thật. Chi tiết ở internal/profile/name.go.
//
// Đây là lớp 1 — để báo lỗi dễ hiểu. Lớp 2 (profile.Remove từ chối đường dẫn
// ngoài kho) mới là lớp không thể quên.
func (ad Addr) Validate() error {
	if err := profile.ValidName(ad.Provider); err != nil {
		return fmt.Errorf("tên provider không hợp lệ: %w", err)
	}
	if err := profile.ValidName(ad.Account); err != nil {
		return fmt.Errorf("tên tài khoản không hợp lệ: %w", err)
	}
	return nil
}

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
	if err := addr.Validate(); err != nil {
		return 0, 0, err
	}
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
	if err := addr.Validate(); err != nil {
		return err
	}
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
	if err := addr.Validate(); err != nil {
		return err
	}
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
		// Provider không có file cấu hình gộp thì chẳng có gì để đồng bộ —
		// thói quen máy của nó nằm ở các file đã nối link dùng chung rồi.
		src := ad.IdentitySource()
		if src == "" || len(ad.SharedKeys()) == 0 {
			r.Skipped = "không cần (không có file cấu hình gộp)"
			out = append(out, r)
			continue
		}
		dst := filepath.Join(acc.Dir, filepath.Base(src))
		if !fileExists(dst) {
			r.Skipped = "chưa có " + filepath.Base(src)
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
func (a *API) ProfileVerify(providerName string, chapNhanDrift bool) (map[string][]provider.Check, error) {
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
		out[n] = append(ad.Verify(), driftCheck(ad, chapNhanDrift))
	}
	// Ô kiểm không thuộc provider nào: quyền truy cập của chính cái kho chứa
	// token. Đặt ở đây vì `verify` là chỗ người dùng đến để hỏi "có ổn không",
	// và vì siết quyền lúc tạo thư mục là việc BEST-EFFORT — ổ mạng hay FAT32
	// có thể từ chối. Không có ô kiểm này thì thất bại đó im lặng.
	out["kho hồ sơ"] = []provider.Check{khoHoSoCheck()}
	return out, nil
}

// driftCheck hỏi CLI phiên bản hiện tại và so với mốc đã ghi.
//
// Đặt trong `verify` chứ không chạy nền: nó SPAWN tiến trình thật (`claude
// --version`), và tự chạy tiến trình của provider sau lưng người dùng là đúng
// thứ đã gây sự cố trên máy dev — xem docs/DO-LUONG.md. `verify` là lệnh người
// ta chủ động gõ.
func driftCheck(ad provider.Adapter, chapNhan bool) provider.Check {
	c := provider.Check{Name: "phiên bản CLI (provider drift)"}
	v, err := ad.Version()
	if err != nil {
		// Không hỏi được phiên bản KHÔNG phải là lỗi của bộ đo drift — có thể
		// CLI chưa cài. Các ô kiểm khác đã nói điều đó rồi, đừng báo động hai lần.
		c.OK, c.Detail = true, "không hỏi được phiên bản ("+err.Error()+")"
		return c
	}
	duong, _ := ad.Command()
	kq := drift.Kiem(ad.Name(), v, duong, chapNhan)
	c.OK, c.Detail = kq.OK, kq.Chi
	return c
}

func khoHoSoCheck() provider.Check {
	root := paths.AccountsRoot()
	c := provider.Check{Name: "quyền truy cập " + root}
	if _, err := os.Stat(root); err != nil {
		c.OK, c.Detail = true, "chưa có kho — sẽ được siết khi tạo"
		return c
	}

	// Quét CẢ CÂY, không chỉ thư mục gốc.
	//
	// Vì sao: bản vá ACL đầu tiên nối `acl.Restrict` vào ba chỗ tạo thư mục và
	// BỎ SÓT `clone.go` — đúng chỗ token bị nhân ra N bản. Ô kiểm lúc đó chỉ soi
	// thư mục gốc nên nó vẫn xanh. Một phép kiểm chỉ nhìn một điểm thì không phát
	// hiện được "quên một chỗ", mà "quên một chỗ" mới là kiểu hỏng hay xảy ra.
	//
	// Quét cây thì mỗi thư mục mới thêm vào sau này cũng tự nằm trong tầm kiểm,
	// không cần ai nhớ cập nhật danh sách.
	var ho []string
	_ = filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() {
			return nil
		}
		if ok, _, err := acl.Check(p); err == nil && !ok {
			ho = append(ho, p)
		}
		return nil
	})
	if len(ho) > 0 {
		c.OK = false
		g := ho[0]
		if len(ho) > 1 {
			c.Detail = fmt.Sprintf("%d thư mục đang hở, ví dụ %s — chạy `sagent verify` lại sau khi sửa", len(ho), g)
		} else {
			c.Detail = "thư mục đang hở: " + g
		}
		return c
	}
	c.OK, c.Detail = true, "cả cây chỉ chủ sở hữu, SYSTEM và nhóm quản trị"
	return c
}

// ---------------------------- phiên ----------------------------

// SessionList — action "session.list". Luôn đối chiếu PID thật.
func (a *API) SessionList() ([]store.Session, error) {
	list, err := a.db.Running()
	// `Running` có thể trả về danh sách ĐÚNG kèm một lỗi: nó không đánh dấu nổi
	// mấy phiên đã chết thành `lost` (DB bị khoá chẳng hạn). Đó không phải lý do
	// để `sagent status` gãy — thao tác đó lặp lại được và lần sau tự sửa. Nhưng
	// cũng không được im: cảnh báo lên bus, còn danh sách vẫn trả về.
	if err != nil {
		a.bus.Warnf("không cập nhật được trạng thái phiên đã chết: %v", err)
	}
	return list, nil
}

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
		// KillTree chứ không Kill: agent sinh tiến trình con, và đã đo được là
		// `taskkill /T` bỏ sót đám con khi tiến trình cha thoát trước. Con sót
		// lại vẫn tiêu hạn mức của bạn — im lặng.
		if err := process.KillTree(s.PID); err != nil {
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

// MoCoi là một tiến trình còn sống của phiên đã chết.
type MoCoi struct {
	Session store.Session
	Procs   []process.Info
}

// SessionSweep — action "session.sweep". Tìm tiến trình còn sống thuộc các phiên
// đã tự chết; chỉ giết khi `giet` = true.
//
// MẶC ĐỊNH LÀ CHỈ BÁO, KHÔNG GIẾT. Windows dùng lại PID, nên danh sách này có
// thể lẫn tiến trình không liên quan — người dùng phải nhìn tên và thời điểm rồi
// tự quyết. Một lệnh tự động giết theo suy đoán thì sớm muộn cũng giết nhầm.
func (a *API) SessionSweep(giet bool) ([]MoCoi, error) {
	lost, err := a.db.Lost()
	if err != nil {
		return nil, err
	}
	var out []MoCoi
	for _, s := range lost {
		ps := process.MoCoi(s.PID, s.Started)
		if len(ps) == 0 {
			continue
		}
		out = append(out, MoCoi{Session: s, Procs: ps})
		if !giet {
			continue
		}
		for _, p := range ps {
			if err := process.KillTree(p.PID); err != nil {
				a.bus.Failuref("mồ côi PID %d (%s) không dừng được: %v", p.PID, p.Ten, err)
				continue
			}
			a.bus.Infof("đã dừng mồ côi PID %d (%s) của #%d %s", p.PID, p.Ten, s.ID, s.Addr())
		}
	}
	return out, nil
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
	if err := req.Addr.Validate(); err != nil {
		return fleet.Result{}, err
	}
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
	// Cảnh báo nếu token sắp hết hạn: hạm đội chạy dài sẽ vượt mốc refresh, mà
	// hành vi khi nhiều bản clone cùng refresh thì CHƯA ĐO (docs/DO-LUONG.md).
	if dir, ok := profile.ResolveDir(req.Addr.Provider, req.Addr.Account); ok {
		if exp, ok := ad.TokenExpiry(dir); ok {
			left := time.Until(exp)
			switch {
			case left <= 0:
				a.bus.Warnf("token của %s ĐÃ HẾT HẠN — chạy `sagent %s` một lần để làm mới trước.", req.Addr, req.Addr)
			case left < 2*time.Hour:
				a.bus.Warnf("token của %s còn %s. Hạm đội chạy lâu hơn thế sẽ phải refresh, "+
					"mà hành vi khi nhiều bản clone cùng refresh CHƯA ĐO.", req.Addr, left.Truncate(time.Minute))
			}
		}
	}

	return fleet.FanOut(a.db, a.bus, ad, req.Addr.Account, fleet.Opts{
		Copies: req.Copies, Worktree: req.Worktree,
	}, req.Args)
}

// ---------------------------- bản clone ----------------------------

// ClonesCreate — action "clones.create".
func (a *API) ClonesCreate(addr Addr, copies int) ([]string, error) {
	if err := addr.Validate(); err != nil {
		return nil, err
	}
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
	if err := addr.Validate(); err != nil {
		return res, err
	}

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

	// Mang token đã refresh trong clone về hồ sơ gốc TRƯỚC khi xoá clone đi —
	// nếu không thì công refresh mất trắng.
	if ad, err := adapterOf(addr.Provider); err == nil {
		if name, err := profile.SyncBackTokens(ad, addr.Account); err == nil && name != "" {
			a.bus.Infof("đã mang %s mới nhất từ bản clone về %s", name, addr)
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

// ---------------------------- flow ----------------------------

// FlowList — action "flow.list". Trả về flow đã gộp (mẫu + global + dự án).
func (a *API) FlowList(dir string) (map[string]flow.Flow, []string, error) {
	return flow.Load(dir)
}

// FlowShow — action "flow.show". Kèm thứ tự chạy đã sắp xếp.
func (a *API) FlowShow(dir, name string) (flow.Flow, []flow.Step, error) {
	flows, _, err := flow.Load(dir)
	if err != nil {
		return flow.Flow{}, nil, err
	}
	f, ok := flows[name]
	if !ok {
		return flow.Flow{}, nil, fmt.Errorf("không có flow %q (xem: sagent flow list)", name)
	}
	order, err := flow.Order(f)
	return f, order, err
}

// FlowValidate — action "flow.validate". Kiểm mọi flow, trả về lỗi + cảnh báo.
func (a *API) FlowValidate(dir string) ([]flow.Problem, error) {
	flows, _, err := flow.Load(dir)
	if err != nil {
		return nil, err
	}
	var ps []flow.Problem
	for _, n := range flow.Names(flows) {
		ps = append(ps, flow.Validate(flows[n])...)
	}
	return ps, nil
}

// agentBridge nối bước `agent` của flow vào fleet — và ĐỢI xong, vì flow cần
// biết bước trước kết thúc mới đi tiếp.
type agentBridge struct {
	a        *API
	fallback Addr
}

func (b agentBridge) RunAgents(ctx context.Context, profileStr, prompt string, copies int, worktree bool) (string, error) {
	addr := b.fallback
	if profileStr != "" {
		addr = ParseAddr(profileStr)
	}
	if addr.Account == "" {
		return "", fmt.Errorf("bước agent chưa biết chạy bằng tài khoản nào — đặt `profile` trong flow hoặc truyền --profile")
	}
	ad, err := adapterOf(addr.Provider)
	if err != nil {
		return "", err
	}
	res, err := b.a.FleetStart(FleetRequest{
		Addr: addr, Copies: copies, Worktree: worktree,
		Args: ad.HeadlessArgs(prompt), // mỗi provider một kiểu, hỏi adapter
	})
	if err != nil {
		return "", err
	}
	if res.Started == 0 {
		return "", fmt.Errorf("không bật được phiên nào")
	}
	// Ghi lại đường dẫn log TRƯỚC khi đợi: xong việc thì phiên đã rời sổ
	// "đang chạy", lúc đó hỏi lại là không còn.
	logs := b.a.sessionLogs(res.IDs)
	if err := b.a.waitSessions(ctx, res.IDs); err != nil {
		return "", err
	}
	return readLogs(logs), nil
}

// sessionLogs lấy đường dẫn log của các phiên vừa bật.
func (a *API) sessionLogs(ids []int64) []string {
	want := map[int64]bool{}
	for _, id := range ids {
		want[id] = true
	}
	running, err := a.db.Running()
	if err != nil {
		return nil
	}
	var out []string
	for _, s := range running {
		if want[s.ID] && s.Log != "" {
			out = append(out, s.Log)
		}
	}
	return out
}

// readLogs gộp log của các agent thành kết quả cho bước sau dùng.
func readLogs(paths []string) string {
	var sb strings.Builder
	for i, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if len(paths) > 1 {
			fmt.Fprintf(&sb, "===== agent %d =====\n", i+1)
		}
		sb.Write(data)
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

// waitSessions đợi các phiên kết thúc. Không có daemon nên hỏi sổ theo nhịp —
// đơn giản và đủ dùng cho một tiến trình CLI đang đứng chờ.
func (a *API) waitSessions(ctx context.Context, ids []int64) error {
	want := map[int64]bool{}
	for _, id := range ids {
		want[id] = true
	}
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		running, err := a.db.Running()
		if err != nil {
			return err
		}
		still := 0
		for _, s := range running {
			if want[s.ID] {
				still++
			}
		}
		if still == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

func (a *API) runner(defaultProfile Addr) *flow.Runner {
	return &flow.Runner{
		DB: a.db, Bus: a.bus,
		Agent:       agentBridge{a: a, fallback: defaultProfile},
		MaxParallel: a.cfg.Policy.MaxParallelSessions,
		// Node `test`/`lint` lấy lệnh từ .sagent/project.toml, khỏi lặp lại
		// trong từng flow.
		Commands: map[string][]string{
			"test":  a.cfg.Commands.Test,
			"lint":  a.cfg.Commands.Lint,
			"build": a.cfg.Commands.Build,
			"setup": a.cfg.Commands.Setup,
		},
	}
}

// FlowRun — action "flow.run". Chạy một flow; dừng lại nếu gặp bước chờ duyệt.
func (a *API) FlowRun(ctx context.Context, dir, name string, vars map[string]string, defaultProfile Addr) (flow.Result, error) {
	f, _, err := a.FlowShow(dir, name)
	if err != nil {
		return flow.Result{}, err
	}
	if ps := flow.Validate(f); len(ps) > 0 {
		for _, p := range ps {
			if !p.Warn {
				return flow.Result{}, fmt.Errorf("flow %q có lỗi: %s", name, p.Msg)
			}
		}
	}
	return a.runner(defaultProfile).Start(ctx, f, dir, vars)
}

// FlowResume — chạy tiếp một lần chạy đang dở.
func (a *API) FlowResume(ctx context.Context, runID int64, defaultProfile Addr) (flow.Result, error) {
	run, err := a.db.GetRun(runID)
	if err != nil {
		return flow.Result{}, fmt.Errorf("không có lần chạy #%d", runID)
	}
	f, _, err := a.FlowShow(run.Dir, run.Flow)
	if err != nil {
		return flow.Result{}, err
	}
	return a.runner(defaultProfile).Resume(ctx, runID, f)
}

// FlowRunDetail trả về lần chạy + trạng thái từng bước + định nghĩa flow.
func (a *API) FlowRunDetail(runID int64) (store.Run, map[string]store.StepRun, flow.Flow, error) {
	run, err := a.db.GetRun(runID)
	if err != nil {
		return store.Run{}, nil, flow.Flow{}, fmt.Errorf("không có lần chạy #%d", runID)
	}
	steps, err := a.db.Steps(runID)
	if err != nil {
		return run, nil, flow.Flow{}, err
	}
	def, _, err := a.FlowShow(run.Dir, run.Flow)
	return run, steps, def, err
}

// FlowApproveOnly chỉ đánh dấu đã duyệt, KHÔNG chạy tiếp — để mặt web trả lời
// người bấm nút ngay rồi mới chạy phần còn lại ở nền.
func (a *API) FlowApproveOnly(runID int64, stepID, by string) error {
	return a.runner(Addr{}).Approve(runID, stepID, by)
}

// FlowSave — action "flow.save". Ghi flow vào flows.toml (nguồn sự thật).
func (a *API) FlowSave(dir string, f flow.Flow) (string, error) { return flow.Save(dir, f) }

// FlowDelete — action "flow.delete". Xoá flow khỏi flows.toml.
func (a *API) FlowDelete(dir, name string) (string, error) { return flow.Delete(dir, name) }

// FlowImport nạp một file flows.toml rời vào dự án.
func (a *API) FlowImport(dir, src string) (string, []string, error) { return flow.Import(dir, src) }

// FlowRuns — action "flow.runs". Lịch sử các lần chạy.
func (a *API) FlowRuns(limit int) ([]store.Run, error) { return a.db.ListRuns(limit) }

// FlowSteps trả trạng thái từng bước của một lần chạy.
func (a *API) FlowSteps(runID int64) (map[string]store.StepRun, error) { return a.db.Steps(runID) }

// FlowApprove — action "flow.approve". Duyệt (hoặc từ chối) rồi chạy tiếp.
func (a *API) FlowApprove(ctx context.Context, runID int64, stepID, by string, ok bool, defaultProfile Addr) (flow.Result, error) {
	r := a.runner(defaultProfile)
	if !ok {
		return flow.Result{RunID: runID, State: store.RunCanceled}, r.Reject(runID, stepID, by)
	}
	if err := r.Approve(runID, stepID, by); err != nil {
		return flow.Result{}, err
	}
	return a.FlowResume(ctx, runID, defaultProfile)
}

// ---------------------------- lặt vặt ----------------------------

func currentConfigDir(ad provider.Adapter) string {
	return strings.TrimRight(os.Getenv(ad.EnvVar()), `\/`)
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

const pathSep = os.PathSeparator
