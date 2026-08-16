// Command ccswitch — quản lý & chạy nhiều tài khoản AI (v2, Go).
//
// Địa chỉ hoá hồ sơ: "provider:account" (mặc định provider "claude"), nên
//
//	ccswitch phu   ==  ccswitch claude:phu
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/trantiendevweb/ccswitch/internal/jsonutil"
	"github.com/trantiendevweb/ccswitch/internal/profile"
	"github.com/trantiendevweb/ccswitch/internal/provider"
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
		fmt.Println("  Chưa có tài khoản nào. Thêm: ccswitch them claude:phu1")
	}
	fmt.Println()
}

func cmdAdd(args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu tên. Ví dụ: ccswitch them claude:phu1"))
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
	fmt.Printf("  Đăng nhập: ccswitch %s:%s  (xong gõ /exit)\n", prov, acc)
}

func cmdRun(prov, acc string, args []string) {
	if prov == "" { // tài khoản gốc
		if err := profile.Run(adapterOf("claude"), "", args); err != nil {
			os.Exit(1)
		}
		return
	}
	a := adapterOf(prov)
	dir := profile.Dir(prov, acc)
	if _, err := os.Stat(dir); err != nil {
		fail(fmt.Errorf("không có %s:%s. Tạo: ccswitch them %s:%s", prov, acc, prov, acc))
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
		fail(fmt.Errorf("thiếu tên. Ví dụ: ccswitch xoa claude:phu1"))
	}
	prov, acc := parseAddr(args[0])
	dir := profile.Dir(prov, acc)
	if _, err := os.Stat(dir); err != nil {
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

func cmdHelp() {
	fmt.Print(`
  ccswitch — quản lý & chạy nhiều tài khoản AI

    ccswitch                      bảng tài khoản
    ccswitch <provider:tên>       chạy CLI bằng tài khoản đó (mặc định provider claude)
    ccswitch goc                  chạy Claude bằng tài khoản gốc
    ccswitch them <provider:tên>  tạo tài khoản mới
    ccswitch ds                   liệt kê
    ccswitch dong-bo [--dry-run]  đồng bộ cấu hình dùng chung
    ccswitch xoa <provider:tên>   xoá tài khoản (an toàn)
    ccswitch verify [provider]    chạy bộ "đã đo"

`)
}

func trimSlash(s string) string { return strings.TrimRight(s, `\/`) }

func fail(err error) {
	fmt.Fprintf(os.Stderr, "  ✗ %v\n", err)
	os.Exit(1)
}
