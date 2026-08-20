package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

// matWeb ánh xạ tên mặt trong `ui.default_surface` sang đường dẫn thật trên
// dashboard và một câu người đọc hiểu được.
//
// Bảng này là chỗ DUY NHẤT biết mặt nào nằm ở URL nào — thêm mặt thứ năm thì
// sửa đúng đây, không phải đi tìm chuỗi "/flow.html" rải trong mã.
var matWeb = map[string]struct{ Duong, Ten string }{
	"dashboard": {"/", "Dashboard 2D"},
	"workflow":  {"/flow.html", "Workflow board"},
	"3d":        {"/trung-tam.html", "Trung tâm (3D)"},
}

// moMatMacDinh chọn mặt để mở khi gõ `sagent` không tham số, theo
// `ui.default_surface` của cấu hình đã gộp. Đây là phần terminal của Pha 5d:
// hai project khác nhau gõ cùng một lệnh phải ra hai chỗ khác nhau.
//
// VÌ SAO ba mặt web chỉ CHỈ ĐƯỜNG chứ không tự bật server: `sagent dash` chiếm
// terminal cho tới khi Ctrl+C và đòi mật khẩu đã đặt trước — gõ `sagent` mà tự
// nhiên mọc ra một tiến trình đang nghe cổng là việc người dùng không hề xin.
// Cấu hình nói họ THÍCH mặt nào, không phải cho phép mở cổng thay họ.
func moMatMacDinh() {
	a, done := open()
	mat := a.Config().UI.DefaultSurface
	done()

	loi, laWeb := chiDuongMat(mat, dashPortMacDinh)
	if !laWeb {
		// "" hoặc "tui" — và bất cứ giá trị nào lọt qua được validate cũng chỉ
		// còn hai khả năng đó, nên nhánh này không cần báo lỗi.
		runTUI()
		return
	}
	fmt.Print(loi)
}

// chiDuongMat dựng lời chỉ đường tới một mặt web, hoặc báo `false` nếu tên mặt
// không phải mặt web (tức là mặt terminal).
//
// Tách khỏi moMatMacDinh để TEST ĐƯỢC: phần còn lại của moMatMacDinh phải mở
// API thật rồi chiếm terminal mới chạy tới đây, nên chôn ở trong thì không ai
// kiểm được — mà đúng chỗ này mới quyết định gõ `sagent` sẽ ra cái gì.
func chiDuongMat(mat string, port int) (string, bool) {
	w, laWeb := matWeb[mat]
	if !laWeb {
		return "", false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n  Mặt mặc định của dự án này là %s (ui.default_surface = %q).\n\n", w.Ten, mat)
	fmt.Fprintf(&b, "    Bật server:  sagent dash\n")
	fmt.Fprintf(&b, "    Rồi mở:      http://127.0.0.1:%d%s\n\n", port, w.Duong)
	fmt.Fprintf(&b, "  Muốn bảng chọn terminal thay vì mặt này: sagent ds\n\n")
	return b.String(), true
}

// runTUI là mặt terminal tương tác: gõ `sagent` không tham số là vào đây.
// Giống tinh thần tk v1 — gõ số để mở tài khoản, chữ cái cho việc khác.
func runTUI() {
	if !isInteractive() {
		// Không có bàn phím: in bảng + trợ giúp thay vì đứng chờ.
		cmdList(nil)
		fmt.Println("  (không có bàn phím — bỏ qua bảng chọn tương tác)")
		return
	}

	in := bufio.NewScanner(os.Stdin)
	for {
		list, running, hong := snapshot()
		printMenu(list, running, hong)

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
func snapshot() ([]api.Profile, []store.Session, []store.Session) {
	a, done := open()
	defer done()
	list, err := a.ProfileList()
	if err != nil {
		fail(err)
	}
	running, _ := a.SessionList()
	// Phiên vừa chết BẤT THƯỜNG: bảng chọn phải nói được "hạm đội trống vì hết
	// hạn mức" chứ không chỉ nói "hạm đội trống". Cùng nguồn với dashboard 2D
	// và màn 3D — không mặt nào tự suy trạng thái.
	hong, _ := a.SessionHong(5)
	return list, running, hong
}

func printMenu(list []api.Profile, running, hong []store.Session) {
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
	// Hạm đội trống có hai nghĩa hoàn toàn khác nhau: việc xong, hay agent chết
	// giữa chừng. Bảng chọn im lặng thì người dùng chỉ thấy nghĩa thứ nhất.
	if len(hong) > 0 {
		now := time.Now()
		fmt.Println()
		for _, s := range hong {
			fmt.Println(dongPhienHong(s, now))
		}
		fmt.Println("     (phiên đã chết — bấm s để xem đủ)")
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
