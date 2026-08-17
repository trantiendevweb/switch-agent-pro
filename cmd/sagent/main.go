// Command sagent — quản lý & chạy nhiều tài khoản AI (v2, Go).
//
// Địa chỉ hoá hồ sơ: "provider:account" (mặc định provider "claude"), nên
//
//	sagent phu   ==  sagent claude:phu
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/config"
	"github.com/trantiendevweb/switch-agent-pro/internal/fleet"
	"github.com/trantiendevweb/switch-agent-pro/internal/jsonutil"
	"github.com/trantiendevweb/switch-agent-pro/internal/process"
	"github.com/trantiendevweb/switch-agent-pro/internal/profile"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
	"github.com/trantiendevweb/switch-agent-pro/internal/workspace"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		cmdList()
		return
	}
	switch args[0] {
	case "help", "-h", "--help", "giup", "?":
		cmdHelp()
	case "ds", "list":
		cmdList()
	case "them", "add":
		cmdAdd(rest(args))
	case "dong-bo", "sync":
		cmdSync(rest(args))
	case "xoa", "remove":
		cmdRemove(rest(args))
	case "verify":
		cmdVerify(rest(args))
	case "clone":
		cmdClone(rest(args))
	case "fleet":
		cmdFleet(rest(args))
	case "status":
		cmdStatus()
	case "clean":
		cmdClean(rest(args))
	case "init":
		cmdInit()
	case "config":
		cmdConfig()
	case "stop":
		cmdStop(rest(args))
	case "goc":
		cmdRun("", "", rest(args))
	default:
		prov, acc := parseAddr(args[0])
		cmdRun(prov, acc, rest(args))
	}
}

func rest(a []string) []string {
	if len(a) > 1 {
		return a[1:]
	}
	return nil
}

func parseAddr(s string) (prov, acc string) {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return "claude", s
}

func adapterOf(name string) provider.Adapter {
	a, ok := provider.Get(name)
	if !ok {
		fail(fmt.Errorf("không có provider '%s' (có: %s)", name, strings.Join(provider.Names(), ", ")))
	}
	return a
}

func cmdList() {
	accs, err := profile.List()
	if err != nil {
		fail(err)
	}
	fmt.Println()
	fmt.Println("  Tài khoản AI trên máy này")
	fmt.Println()
	cur := os.Getenv("CLAUDE_CONFIG_DIR")
	for i, a := range accs {
		email, tok := "(chưa đăng nhập)", "chưa đăng nhập"
		if ad, ok := provider.Get(a.Provider); ok {
			if e := ad.Identity(a.Dir); e != "" {
				email = e
			}
			if ad.HasToken(a.Dir) {
				tok = "sẵn sàng"
			}
		}
		mark := " "
		if cur != "" && trimSlash(cur) == trimSlash(a.Dir) {
			mark = "*"
		}
		fmt.Printf("  %s %2d  %-7s %-12s %-34s %s\n", mark, i+1, a.Provider, a.Name, email, tok)
	}
	if len(accs) == 0 {
		fmt.Println("  Chưa có tài khoản nào. Thêm: sagent them claude:phu1")
	}
	fmt.Println()
}

func cmdAdd(args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu tên. Ví dụ: sagent them claude:phu1"))
	}
	prov, acc := parseAddr(args[0])
	if acc == "" {
		fail(fmt.Errorf("thiếu tên tài khoản"))
	}
	a := adapterOf(prov)
	linked, seeded, err := profile.Create(a, acc)
	if err != nil {
		fail(err)
	}
	fmt.Printf("  ✓ Đã tạo %s:%s (nối %d mục dùng chung, gieo %d khoá)\n", prov, acc, linked, seeded)
	fmt.Printf("  Đăng nhập: sagent %s:%s  (xong gõ /exit)\n", prov, acc)
}

func cmdRun(prov, acc string, args []string) {
	if prov == "" { // tài khoản gốc
		if err := profile.Run(adapterOf("claude"), "", args); err != nil {
			os.Exit(1)
		}
		return
	}
	a := adapterOf(prov)
	dir, ok := profile.ResolveDir(prov, acc)
	if !ok {
		fail(fmt.Errorf("không có %s:%s. Tạo: sagent them %s:%s", prov, acc, prov, acc))
	}
	if err := profile.Run(a, dir, args); err != nil {
		os.Exit(1)
	}
}

func cmdSync(args []string) {
	dry := false
	for _, x := range args {
		if x == "--dry-run" || x == "-XemTruoc" {
			dry = true
		}
	}
	accs, _ := profile.List()
	if len(accs) == 0 {
		fmt.Println("  Chưa có tài khoản nào để đồng bộ.")
		return
	}
	for _, a := range accs {
		ad, ok := provider.Get(a.Provider)
		if !ok {
			continue
		}
		dst := profile.Dir(a.Provider, a.Name) + string(os.PathSeparator) + ".claude.json"
		if _, err := os.Stat(dst); err != nil {
			fmt.Printf("  %s:%s  bỏ qua (chưa có .claude.json)\n", a.Provider, a.Name)
			continue
		}
		if dry {
			n, err := jsonutil.SyncKeys(ad.IdentitySource(), dst+".__dry", ad.SharedKeys())
			_ = n
			_ = err
			fmt.Printf("  %s:%s  (xem trước — chưa ghi)\n", a.Provider, a.Name)
			os.Remove(dst + ".__dry")
			continue
		}
		n, err := jsonutil.SyncKeys(ad.IdentitySource(), dst, ad.SharedKeys())
		switch {
		case err != nil:
			fmt.Printf("  %s:%s  LỖI: %v\n", a.Provider, a.Name, err)
		case n > 0:
			fmt.Printf("  %s:%s  đổi %d khoá (đã sao lưu .bak)\n", a.Provider, a.Name, n)
		default:
			fmt.Printf("  %s:%s  đã khớp\n", a.Provider, a.Name)
		}
	}
}

func cmdRemove(args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu tên. Ví dụ: sagent xoa claude:phu1"))
	}
	prov, acc := parseAddr(args[0])
	dir, ok := profile.ResolveDir(prov, acc)
	if !ok {
		fail(fmt.Errorf("không có %s:%s", prov, acc))
	}
	if err := profile.Remove(dir); err != nil {
		fail(err)
	}
	fmt.Printf("  ✓ Đã xoá %s:%s và token của nó.\n", prov, acc)
}

func cmdVerify(args []string) {
	names := provider.Names()
	if len(args) > 0 {
		names = []string{args[0]}
	}
	code := 0
	for _, n := range names {
		a, ok := provider.Get(n)
		if !ok {
			continue
		}
		fmt.Printf("\n  [%s]\n", n)
		for _, c := range a.Verify() {
			s := "✓"
			if !c.OK {
				s = "✗"
				code = 1
			}
			fmt.Printf("    %s %-32s %s\n", s, c.Name, c.Detail)
		}
	}
	fmt.Println()
	os.Exit(code)
}

// ---------------------- chạy song song (fleet) ----------------------

// splitDashDash tách phần đối số của ta và phần truyền thẳng cho CLI con.
func splitDashDash(args []string) (mine, child []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

// boolFlag rút cờ không tham số, trả về có/không + phần còn lại.
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

// intFlag lấy giá trị của cờ dạng `--ten N`, trả về phần còn lại.
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

func openStore() *store.DB {
	db, err := store.Open()
	if err != nil {
		fail(fmt.Errorf("không mở được sổ trạng thái (%s): %w", store.Path(), err))
	}
	return db
}

func cmdClone(args []string) {
	copies, rest := intFlag(args, "--copies", 2)
	if len(rest) == 0 {
		fail(fmt.Errorf("thiếu tài khoản. Ví dụ: sagent clone claude:phu --copies 4"))
	}
	prov, acc := parseAddr(rest[0])
	dirs, err := profile.Clone(adapterOf(prov), acc, copies)
	if err != nil {
		fail(err)
	}
	for _, d := range dirs {
		fmt.Println(d)
	}
}

func cmdFleet(args []string) {
	wd, _ := os.Getwd()
	cfg, err := config.Load(wd)
	if err != nil {
		fail(err)
	}

	mine, child := splitDashDash(args)
	worktree, mine := boolFlag(mine, "--worktree")
	// Không truyền cờ thì lấy theo cấu hình dự án.
	if !worktree && cfg.Project.Workspace == "worktree" {
		worktree = true
	}
	copies, rest := intFlag(mine, "--copies", 2)
	if len(rest) == 0 {
		fail(fmt.Errorf(`thiếu tài khoản. Ví dụ:
  sagent fleet claude:phu --copies 4 --worktree -- -p "tóm tắt repo này"`))
	}
	// Chính sách của dự án là trần cứng — tránh lỡ tay đốt hạn mức.
	if m := cfg.Policy.MaxParallelSessions; m > 0 && copies > m {
		fmt.Printf("  ! %d vượt policy.max_parallel_sessions=%d của dự án — hạ xuống %d.\n", copies, m, m)
		copies = m
	}
	prov, acc := parseAddr(rest[0])
	db := openStore()
	defer db.Close()
	opts := fleet.Opts{Copies: copies, Worktree: worktree}
	if err := fleet.FanOut(db, adapterOf(prov), acc, opts, child); err != nil {
		fail(err)
	}
}

// ---------------------- cấu hình theo dự án ----------------------

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
	body := fmt.Sprintf(config.Sample, filepath.Base(wd))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("  ✓ đã tạo %s\n", path)
	fmt.Println("  Sửa rồi xem lại bằng: sagent config")
}

func cmdConfig() {
	wd, _ := os.Getwd()
	c, err := config.Load(wd)
	if err != nil {
		fail(err)
	}
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
	fmt.Println()
}

func cmdClean(args []string) {
	force, args := boolFlag(args, "--force")
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu tài khoản. Ví dụ: sagent clean claude:phu"))
	}
	prov, acc := parseAddr(args[0])
	// Không xoá bản clone đang có phiên chạy — sẽ giết mất việc đang làm.
	db := openStore()
	defer db.Close()
	if list, err := db.Running(); err == nil {
		for _, s := range list {
			if s.Provider == prov && s.Account == acc {
				fail(fmt.Errorf("còn phiên #%d đang chạy trên %s:%s — dừng trước: sagent stop all",
					s.ID, prov, acc))
			}
		}
	}
	// Dọn worktree trước: chúng nằm ngoài thư mục clone nên xoá clone không
	// đụng tới, để lại là git giữ mục chết trong sổ worktree.
	if wd, err := os.Getwd(); err == nil {
		if repoRoot, ok := workspace.RepoRoot(wd); ok {
			gone, kept := 0, 0
			for _, dir := range workspace.FindAll(repoRoot, acc) {
				// Việc agent làm dở là DỮ LIỆU THẬT. Không xoá nếu chưa commit.
				if workspace.IsDirty(dir) && !force {
					fmt.Printf("  ! giữ lại worktree %s — còn thay đổi chưa commit\n", filepath.Base(dir))
					fmt.Printf("      xem: git -C %s status\n", dir)
					kept++
					continue
				}
				if err := workspace.Remove(repoRoot, dir); err == nil {
					gone++
				}
			}
			if gone > 0 {
				fmt.Printf("  ✓ đã gỡ %d worktree (nhánh sagent/%s-* giữ nguyên)\n", gone, acc)
			}
			if kept > 0 {
				fmt.Printf("  → %d worktree được giữ. Commit/stash rồi chạy lại, hoặc `sagent clean %s:%s --force` để bỏ luôn.\n",
					kept, prov, acc)
			}
		}
	}
	n, err := profile.CleanClones(prov, acc)
	if err != nil {
		fail(err)
	}
	fmt.Printf("  ✓ đã xoá %d bản clone của %s:%s\n", n, prov, acc)
}

func cmdStatus() {
	db := openStore()
	defer db.Close()
	list, err := db.Running()
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
			s.ID, s.Addr(), s.PID,
			time.Since(s.Started).Truncate(time.Second), where)
	}
	fmt.Printf("\n  %d phiên. Dừng hết: sagent stop all\n\n", len(list))
}

func cmdStop(args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu mục tiêu. Ví dụ: sagent stop all  hoặc  sagent stop 3"))
	}
	db := openStore()
	defer db.Close()
	list, err := db.Running()
	if err != nil {
		fail(err)
	}
	want := int64(-1)
	if args[0] != "all" {
		n, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fail(fmt.Errorf("không hiểu '%s' — dùng số phiên hoặc 'all'", args[0]))
		}
		want = n
	}
	n := 0
	for _, s := range list {
		if want >= 0 && s.ID != want {
			continue
		}
		if err := process.Kill(s.PID); err != nil {
			fmt.Printf("  ! #%d (PID %d) không dừng được: %v\n", s.ID, s.PID, err)
			continue
		}
		_ = db.SetState(s.ID, store.StateStopped)
		fmt.Printf("  ✓ đã dừng #%d %s\n", s.ID, s.Addr())
		n++
	}
	if n == 0 {
		fmt.Println("  Không có phiên nào khớp.")
	}
}

func cmdHelp() {
	fmt.Print(`
  sagent — quản lý & chạy nhiều tài khoản AI

    sagent                      bảng tài khoản
    sagent <provider:tên>       chạy CLI bằng tài khoản đó (mặc định provider claude)
    sagent goc                  chạy Claude bằng tài khoản gốc
    sagent them <provider:tên>  tạo tài khoản mới
    sagent ds                   liệt kê
    sagent dong-bo [--dry-run]  đồng bộ cấu hình dùng chung
    sagent xoa <provider:tên>   xoá tài khoản (an toàn)
    sagent verify [provider]    chạy bộ "đã đo"

  Chạy song song (agent headless):

    sagent fleet <provider:tên> --copies N [--worktree] -- <lệnh>
                                bật N phiên song song trên MỘT tài khoản
                                --worktree: mỗi phiên một git worktree riêng
                                (không có thì các phiên dùng chung thư mục
                                 hiện tại và có thể sửa đè file của nhau)
    sagent clone <provider:tên> --copies N
                                chỉ tạo N thư mục config, không chạy
    sagent status               phiên nào đang chạy
    sagent stop <số|all>        dừng phiên
    sagent clean <provider:tên> [--force]
                                gỡ worktree + xoá clone (an toàn; giữ lại
                                worktree còn thay đổi chưa commit)

  Cấu hình theo dự án:

    sagent init                 tạo .sagent/project.toml cho repo hiện tại
    sagent config               xem cấu hình đã gộp + đọc từ file nào

  Ví dụ:

    sagent fleet claude:phu --copies 4 --worktree -- -p "sửa lỗi trong repo"

`)
}

func trimSlash(s string) string { return strings.TrimRight(s, `\/`) }

func fail(err error) {
	fmt.Fprintf(os.Stderr, "  ✗ %v\n", err)
	os.Exit(1)
}
