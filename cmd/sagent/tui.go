package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// isInteractive báo stdin có phải bàn phím thật không. Chạy trong CI, script,
// hay bị chuyển hướng đầu vào thì trả false — để TUI KHÔNG treo chờ gõ (đây là
// bài học từ tk v1: menu treo trong pipeline là cực kỳ khó chịu).
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// runTUI là mặt terminal tương tác: gõ `sagent` không tham số là vào đây.
// Giống tinh thần tk v1 — gõ số để mở tài khoản, chữ cái cho việc khác.
func runTUI() {
	if !isInteractive() {
		// Không có bàn phím: in bảng + trợ giúp thay vì đứng chờ.
		cmdList()
		fmt.Println("  (không có bàn phím — bỏ qua bảng chọn tương tác)")
		return
	}

	in := bufio.NewScanner(os.Stdin)
	for {
		list, running := snapshot()
		printMenu(list, running)

		fmt.Print("   Chọn (số=mở · t=thêm · d=đồng bộ · x=xoá · s=phiên · ?=trợ giúp · Enter=thoát): ")
		if !in.Scan() {
			return
		}
		choice := strings.TrimSpace(in.Text())

		switch {
		case choice == "":
			return
		case choice == "?":
			cmdHelp()
		case choice == "d":
			cmdSync(nil)
		case choice == "s":
			cmdStatus()
		case choice == "t":
			if name := ask(in, "   Tên tài khoản mới (vd claude:phu1): "); name != "" {
				tuiAdd(name)
			}
		case choice == "x":
			if name := ask(in, "   Xoá tài khoản nào (vd claude:phu1): "); name != "" {
				tuiRemove(name)
			}
		default:
			if n, err := strconv.Atoi(choice); err == nil {
				if n >= 1 && n <= len(list) {
					// Mở tài khoản: chiếm terminal cho tới khi người dùng thoát.
					cmdRun(list[n-1].Addr(), nil)
					return
				}
				fmt.Printf("   Không có số %d trong bảng.\n", n)
				continue
			}
			// Gõ thẳng địa chỉ cũng được.
			cmdRun(choice, nil)
			return
		}
	}
}

// snapshot đọc một lần trạng thái để vẽ bảng, rồi đóng ngay — không giữ store
// mở trong lúc chờ người dùng gõ.
func snapshot() ([]api.Profile, []store.Session) {
	a, done := open()
	defer done()
	list, err := a.ProfileList()
	if err != nil {
		fail(err)
	}
	running, _ := a.SessionList()
	return list, running
}

func printMenu(list []api.Profile, running []store.Session) {
	fmt.Println()
	fmt.Println("  Tài khoản AI trên máy này")
	fmt.Println()
	for i, p := range list {
		id := p.Identity
		if id == "" {
			id = "(chưa đăng nhập)"
		}
		tok := "chưa đăng nhập"
		if p.HasToken {
			tok = "sẵn sàng"
		}
		mark := " "
		if p.Active {
			mark = "*"
		}
		fmt.Printf("  %s %2d  %-7s %-12s %-30s %s\n", mark, i+1, p.Provider, p.Account, id, tok)
	}
	if len(list) == 0 {
		fmt.Println("     (chưa có tài khoản nào — bấm t để thêm)")
	}
	if len(running) > 0 {
		fmt.Printf("\n     %d phiên đang chạy — bấm s để xem\n", len(running))
	}
	fmt.Println()
}

func ask(in *bufio.Scanner, prompt string) string {
	fmt.Print(prompt)
	if !in.Scan() {
		return ""
	}
	return strings.TrimSpace(in.Text())
}

// tuiAdd/tuiRemove KHÔNG dùng cmdAdd/cmdRemove vì mấy hàm đó gọi fail() → os.Exit
// khi lỗi, sẽ giết luôn cả bảng chọn chỉ vì một cái gõ nhầm. Trong TUI ta bắt
// lỗi và vòng lại.
func tuiAdd(addrStr string) {
	a, done := open()
	defer done()
	addr := api.ParseAddr(addrStr)
	if addr.Account == "" {
		fmt.Println("   ✗ thiếu tên tài khoản")
		return
	}
	if _, _, err := a.ProfileCreate(addr); err != nil {
		fmt.Printf("   ✗ %v\n", err)
		return
	}
	done()
	fmt.Printf("   Đăng nhập: sagent %s   (xong gõ /exit)\n", addr)
}

func tuiRemove(addrStr string) {
	a, done := open()
	defer done()
	if err := a.ProfileRemove(api.ParseAddr(addrStr)); err != nil {
		fmt.Printf("   ✗ %v\n", err)
	}
}
