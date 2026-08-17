// Command sagent — CLI của Switch-Agent-Pro.
//
// CLI là MẶT ĐẦU TIÊN, không phải lõi: nó chỉ gọi internal/api và vẽ event ra
// màn hình (MASTER-PLAN mục 2c). TUI, dashboard 2D, workflow board và 3D sau
// này là client ngang hàng, dùng đúng các hành động trong `api.Actions`.
//
// Địa chỉ hoá hồ sơ: "provider:account" (mặc định provider "claude"), nên
//
//	sagent phu   ==   sagent claude:phu
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/config"
	"github.com/trantiendevweb/switch-agent-pro/internal/dash"
	"github.com/trantiendevweb/switch-agent-pro/internal/events"
)

// command là một lệnh CLI, gắn với đúng một hành động trong api.Actions.
//
// Trường `action` không phải trang trí: test ngang quyền (main_test.go) dùng nó
// để khẳng định MỌI hành động của hệ thống đều gọi được từ terminal.
type command struct {
	action  string
	summary string
	run     func(args []string)
}

var commands map[string]command

func init() {
	commands = map[string]command{
		"ds":      {"profile.list", "liệt kê tài khoản", func(a []string) { cmdList() }},
		"them":    {"profile.create", "tạo tài khoản mới", cmdAdd},
		"xoa":     {"profile.remove", "xoá tài khoản (an toàn)", cmdRemove},
		"dong-bo": {"profile.sync", "đồng bộ cấu hình dùng chung", cmdSync},
		"verify":  {"profile.verify", "chạy bộ \"đã đo\"", cmdVerify},
		"status":  {"session.list", "phiên nào đang chạy", func(a []string) { cmdStatus() }},
		"stop":    {"session.stop", "dừng phiên", cmdStop},
		"fleet":   {"fleet.start", "bật N phiên song song", cmdFleet},
		"clone":   {"clones.create", "tạo N thư mục cấu hình riêng", cmdClone},
		"clean":   {"clones.clean", "gỡ worktree + xoá clone", cmdClean},
		"config":  {"config.show", "xem cấu hình đã gộp", func(a []string) { cmdConfig() }},
		"init":    {"config.init", "tạo .sagent/project.toml", func(a []string) { cmdInit() }},
		"dash":    {"dash.serve", "mở dashboard 2D ở trình duyệt", cmdDash},
		// `run` không có tên lệnh riêng: gõ thẳng địa chỉ là chạy.
		"__run": {"profile.run", "chạy CLI bằng tài khoản đó", nil},
	}
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runTUI()
		return
	}
	switch args[0] {
	case "help", "-h", "--help", "giup", "?":
		cmdHelp()
		return
	case "list":
		cmdList()
		return
	case "add":
		cmdAdd(rest(args))
		return
	case "remove":
		cmdRemove(rest(args))
		return
	case "sync":
		cmdSync(rest(args))
		return
	case "goc":
		cmdRunRoot(rest(args))
		return
	}
	if c, ok := commands[args[0]]; ok && c.run != nil {
		c.run(rest(args))
		return
	}
	// Không khớp lệnh nào: coi như địa chỉ hồ sơ (profile.run).
	cmdRun(args[0], rest(args))
}

func rest(a []string) []string {
	if len(a) > 1 {
		return a[1:]
	}
	return nil
}

// open dựng API cho thư mục hiện tại và bắt đầu vẽ event ra màn hình.
//
// Hàm done trả về là IDEMPOTENT và **chờ vẽ xong mọi event còn trong hàng đợi**.
// Lệnh nào in tổng kết của riêng nó thì phải gọi done() TRƯỚC, nếu không dòng
// tổng kết sẽ chen lên trước những event cuối cùng (đã dính đúng lỗi này).
func open() (*api.API, func()) {
	wd, _ := os.Getwd()
	a, err := api.New(wd)
	if err != nil {
		fail(err)
	}
	stopPrinting := printEvents(a.Events())
	var once sync.Once
	return a, func() {
		once.Do(func() {
			stopPrinting() // đóng kênh rồi đợi goroutine vẽ hết phần còn lại
			a.Close()
		})
	}
}

// printEvents là chỗ DUY NHẤT biến event thành chữ trên terminal. Mặt khác
// (dashboard, 3D) sẽ có bộ vẽ riêng, cùng đọc một luồng.
func printEvents(bus *events.Bus) func() {
	ch, cancel := bus.Subscribe(128)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for e := range ch {
			switch e.Type {
			case events.Warning:
				fmt.Printf("  ⚠ %s\n", e.Msg)
			case events.Failure:
				fmt.Printf("  ✗ %s\n", e.Msg)
			case events.SessionStarted:
				fmt.Printf("  ✓ #%-3d %-18s %s\n", e.SessionID, e.Addr, e.Msg)
			case events.SessionStopped:
				fmt.Printf("  ✓ đã dừng #%d %s\n", e.SessionID, e.Addr)
			case events.WorktreeKept:
				fmt.Printf("  ! giữ lại worktree — %s\n", e.Detail["path"])
			case events.WorktreeAdded, events.WorktreeGone, events.ClonesCreated,
				events.ClonesCleaned, events.ProfileCreated, events.ProfileRemoved:
				fmt.Printf("  · %s %s\n", e.Addr, e.Msg)
			default:
				fmt.Printf("  · %s\n", e.Msg)
			}
		}
	}()
	return func() { cancel(); <-done }
}

// ---------------------------- hồ sơ ----------------------------

func cmdList() {
	a, done := open()
	defer done()
	list, err := a.ProfileList()
	if err != nil {
		fail(err)
	}
	fmt.Println()
	fmt.Println("  Tài khoản AI trên máy này")
	fmt.Println()
	for i, p := range list {
		id, tok, mark := p.Identity, "chưa đăng nhập", " "
		if id == "" {
			id = "(chưa đăng nhập)"
		}
		if p.HasToken {
			tok = "sẵn sàng"
		}
		if p.Active {
			mark = "*"
		}
		fmt.Printf("  %s %2d  %-7s %-12s %-34s %s\n", mark, i+1, p.Provider, p.Account, id, tok)
	}
	if len(list) == 0 {
		fmt.Println("  Chưa có tài khoản nào. Thêm: sagent them claude:phu1")
	}
	fmt.Println()
}

func cmdAdd(args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu tên. Ví dụ: sagent them claude:phu1"))
	}
	a, done := open()
	defer done()
	addr := api.ParseAddr(args[0])
	if addr.Account == "" {
		fail(fmt.Errorf("thiếu tên tài khoản"))
	}
	if _, _, err := a.ProfileCreate(addr); err != nil {
		fail(err)
	}
	fmt.Printf("  Đăng nhập: sagent %s   (xong gõ /exit)\n", addr)
}

func cmdRemove(args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu tên. Ví dụ: sagent xoa claude:phu1"))
	}
	a, done := open()
	defer done()
	if err := a.ProfileRemove(api.ParseAddr(args[0])); err != nil {
		fail(err)
	}
}

func cmdRun(addr string, args []string) {
	a, done := open()
	defer done()
	if err := a.ProfileRun(api.ParseAddr(addr), args); err != nil {
		fail(err)
	}
}

func cmdRunRoot(args []string) {
	a, done := open()
	defer done()
	if err := a.RunRoot(args); err != nil {
		os.Exit(1)
	}
}

func cmdSync(args []string) {
	dry, _ := boolFlag(args, "--dry-run")
	a, done := open()
	defer done()
	reports, err := a.ProfileSync(dry)
	if err != nil {
		fail(err)
	}
	if len(reports) == 0 {
		fmt.Println("  Chưa có tài khoản nào để đồng bộ.")
		return
	}
	for _, r := range reports {
		switch {
		case r.Err != nil:
			fmt.Printf("  %-18s LỖI: %v\n", r.Addr, r.Err)
		case r.Skipped != "":
			fmt.Printf("  %-18s %s\n", r.Addr, r.Skipped)
		case r.Changed > 0:
			fmt.Printf("  %-18s đổi %d khoá (đã sao lưu .bak)\n", r.Addr, r.Changed)
		default:
			fmt.Printf("  %-18s đã khớp\n", r.Addr)
		}
	}
}

func cmdVerify(args []string) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	a, done := open()
	defer done()
	res, err := a.ProfileVerify(name)
	if err != nil {
		fail(err)
	}
	code := 0
	for prov, checks := range res {
		fmt.Printf("\n  [%s]\n", prov)
		for _, c := range checks {
			s := "✓"
			if !c.OK {
				s, code = "✗", 1
			}
			fmt.Printf("    %s %-32s %s\n", s, c.Name, c.Detail)
		}
	}
	fmt.Println()
	done()
	os.Exit(code)
}

// ---------------------------- phiên ----------------------------

func cmdStatus() {
	a, done := open()
	defer done()
	list, err := a.SessionList()
	if err != nil {
		fail(err)
	}
	fmt.Println()
	if len(list) == 0 {
		fmt.Println("  Không có phiên nào đang chạy.")
		fmt.Println("  Bật thử: sagent fleet claude:<tên> --copies 4 -- -p \"...\"")
		fmt.Println()
		return
	}
	fmt.Println("  Phiên đang chạy")
	fmt.Println()
	for _, s := range list {
		where := s.Log
		if s.Worktree != "" {
			where = "worktree: " + s.Worktree
		}
		fmt.Printf("   #%-3d %-18s PID %-7d %6s  %s\n",
			s.ID, s.Addr(), s.PID, time.Since(s.Started).Truncate(time.Second), where)
	}
	fmt.Printf("\n  %d phiên. Dừng hết: sagent stop all\n\n", len(list))
}

func cmdStop(args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu mục tiêu. Ví dụ: sagent stop all  hoặc  sagent stop 3"))
	}
	a, done := open()
	defer done()
	var id int64 = -1
	if args[0] != "all" {
		n, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fail(fmt.Errorf("không hiểu '%s' — dùng số phiên hoặc 'all'", args[0]))
		}
		id = n
	}
	n, err := a.SessionStop(id)
	if err != nil {
		fail(err)
	}
	if n == 0 {
		fmt.Println("  Không có phiên nào khớp.")
	}
}

func cmdFleet(args []string) {
	mine, child := splitDashDash(args)
	worktree, mine := boolFlag(mine, "--worktree")
	copies, rest := intFlag(mine, "--copies", 2)
	if len(rest) == 0 {
		fail(fmt.Errorf(`thiếu tài khoản. Ví dụ:
  sagent fleet claude:phu --copies 4 --worktree -- -p "tóm tắt repo này"`))
	}
	a, done := open()
	defer done()
	res, err := a.FleetStart(api.FleetRequest{
		Addr: api.ParseAddr(rest[0]), Copies: copies, Worktree: worktree, Args: child,
	})
	if err != nil {
		fail(err)
	}
	done() // vẽ hết event rồi mới tới lượt dòng tổng kết
	fmt.Printf("\n  Đã khởi chạy %d/%d phiên. Xem: sagent status  ·  Dừng: sagent stop all\n",
		res.Started, res.Wanted)
}

func cmdClone(args []string) {
	copies, rest := intFlag(args, "--copies", 2)
	if len(rest) == 0 {
		fail(fmt.Errorf("thiếu tài khoản. Ví dụ: sagent clone claude:phu --copies 4"))
	}
	a, done := open()
	defer done()
	dirs, err := a.ClonesCreate(api.ParseAddr(rest[0]), copies)
	if err != nil {
		fail(err)
	}
	for _, d := range dirs {
		fmt.Println(d)
	}
}

func cmdClean(args []string) {
	force, args := boolFlag(args, "--force")
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu tài khoản. Ví dụ: sagent clean claude:phu"))
	}
	a, done := open()
	defer done()
	wd, _ := os.Getwd()
	addr := api.ParseAddr(args[0])
	res, err := a.ClonesClean(addr, wd, force)
	if err != nil {
		fail(err)
	}
	done()
	if len(res.WorktreesKept) > 0 {
		fmt.Printf("  → %d worktree được giữ. Commit/stash rồi chạy lại, hoặc `sagent clean %s --force` để bỏ luôn.\n",
			len(res.WorktreesKept), addr)
	}
}

// ---------------------------- dashboard ----------------------------

func cmdDash(args []string) {
	port, _ := intFlag(args, "--port", 4600)
	// Mở API TRỰC TIẾP (không gắn bộ vẽ terminal): dashboard tiêu thụ event qua
	// SSE, terminal chỉ cần in URL. Nếu dùng open() thì event sẽ đổ ra cả terminal.
	wd, _ := os.Getwd()
	a, err := api.New(wd)
	if err != nil {
		fail(err)
	}
	defer a.Close()
	if err := dash.New(a).Run("127.0.0.1", port); err != nil {
		fail(err)
	}
}

// ---------------------------- cấu hình ----------------------------

func cmdInit() {
	wd, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	dir := filepath.Join(wd, config.ProjectDirName)
	path := filepath.Join(dir, "project.toml")
	if _, err := os.Stat(path); err == nil {
		fail(fmt.Errorf("đã có %s — sửa tay hoặc xoá rồi chạy lại", path))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(path, []byte(fmt.Sprintf(config.Sample, filepath.Base(wd))), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("  ✓ đã tạo %s\n", path)
	fmt.Println("  Sửa rồi xem lại bằng: sagent config")
}

func cmdConfig() {
	a, done := open()
	defer done()
	c := a.Config()
	fmt.Println()
	if len(c.Sources) == 0 {
		fmt.Println("  Chưa có file cấu hình nào — đang dùng mặc định.")
		fmt.Println("  Tạo cho dự án này: sagent init")
	} else {
		fmt.Println("  Đọc theo thứ tự (dưới đè lên trên):")
		for _, s := range c.Sources {
			fmt.Println("    ·", s)
		}
	}
	fmt.Println()
	fmt.Printf("  name                    %s\n", c.Name)
	fmt.Printf("  project.workspace       %s\n", c.Project.Workspace)
	fmt.Printf("  project.default_branch  %s\n", c.Project.DefaultBranch)
	fmt.Printf("  policy.max_parallel     %d\n", c.Policy.MaxParallelSessions)
	if len(c.Policy.RequireApprovalFor) > 0 {
		fmt.Printf("  policy.require_approval %v\n", c.Policy.RequireApprovalFor)
	}
	fmt.Printf("  ui.default_surface      %s\n", c.UI.DefaultSurface)
	fmt.Printf("\n  api version             %d · event schema %d\n\n", api.Version, events.SchemaVersion)
}

// ---------------------------- cờ & lặt vặt ----------------------------

func splitDashDash(args []string) (mine, child []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

func boolFlag(args []string, name string) (bool, []string) {
	out := make([]string, 0, len(args))
	found := false
	for _, a := range args {
		if a == name {
			found = true
			continue
		}
		out = append(out, a)
	}
	return found, out
}

func intFlag(args []string, name string, def int) (int, []string) {
	out := make([]string, 0, len(args))
	val := def
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			if n, err := strconv.Atoi(args[i+1]); err == nil {
				val = n
				i++
				continue
			}
		}
		out = append(out, args[i])
	}
	return val, out
}

func cmdHelp() {
	fmt.Print(`
  sagent — điều phối nhiều tài khoản AI và nhiều agent

  Tài khoản:

    sagent                      bảng chọn tương tác (gõ số để mở tài khoản)
    sagent <provider:tên>       chạy CLI bằng tài khoản đó (mặc định claude)
    sagent goc                  chạy bằng tài khoản gốc
    sagent them <provider:tên>  tạo tài khoản mới
    sagent ds                   liệt kê
    sagent dong-bo [--dry-run]  đồng bộ cấu hình dùng chung
    sagent xoa <provider:tên>   xoá tài khoản (an toàn)
    sagent verify [provider]    chạy bộ "đã đo"

  Chạy song song (agent headless):

    sagent fleet <provider:tên> --copies N [--worktree] -- <lệnh>
                                bật N phiên song song trên MỘT tài khoản
                                --worktree: mỗi phiên một git worktree riêng
    sagent clone <provider:tên> --copies N
                                chỉ tạo thư mục cấu hình, không chạy
    sagent status               phiên nào đang chạy
    sagent stop <số|all>        dừng phiên
    sagent clean <provider:tên> [--force]
                                gỡ worktree + xoá clone (giữ lại worktree
                                còn thay đổi chưa commit)

  Cấu hình theo dự án:

    sagent init                 tạo .sagent/project.toml
    sagent config               xem cấu hình đã gộp + đọc từ file nào

  Dashboard:

    sagent dash [--port N]      mở dashboard 2D ở trình duyệt (chỉ loopback)

  Ví dụ:

    sagent fleet claude:phu --copies 4 --worktree -- -p "sửa lỗi trong repo"

`)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "  ✗ %v\n", err)
	os.Exit(1)
}

var _ = strings.TrimSpace
