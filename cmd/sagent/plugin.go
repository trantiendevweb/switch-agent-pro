package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/plugin"
)

// cmdPlugin in bảng plugin đã cài — action "plugin.list".
//
//	sagent plugin           — plugin nào đã cài, xin quyền gì
//	sagent plugin quyen     — bảng QUYỀN: host chặn được gì, chưa chặn được gì
//
// Bảng quyền có lệnh riêng vì nó trả lời một câu khác: không phải "tôi đã cài
// gì" mà "khai một quyền trong manifest thì THẬT SỰ được đảm bảo tới đâu". Câu
// thứ hai đúng cho mọi máy và trả lời được cả khi chưa cài plugin nào.
func cmdPlugin(args []string) {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "quyen", "quyền", "perm":
		inBangQuyen()
	case "", "list", "ds":
		inDanhSachPlugin()
	default:
		fail(fmt.Errorf("không có `sagent plugin %s`. Dùng: sagent plugin [list|quyen]", sub))
	}
}

func inDanhSachPlugin() {
	wd, _ := os.Getwd()
	a, done := open()
	defer done()
	b := a.Plugins(wd)
	done()

	if len(b.Muc) == 0 {
		fmt.Println()
		fmt.Println("  chưa cài plugin nào")
		if len(b.Nguon) == 0 {
			fmt.Println("  Đặt vào: <kho>/plugins/<ten>/plugin.toml")
			fmt.Println("       hoặc <dự án>/.sagent/plugins/<ten>/plugin.toml")
		}
	}
	for _, m := range b.Muc {
		fmt.Println()
		fmt.Printf("  %s  %s\n", m.Ten, m.MoTa)
		fmt.Printf("    phiên bản %s · giao thức %d · %s\n", m.PhienBan, m.GiaoThuc, m.Duong)
		if !m.CoExec {
			// Manifest đúng mà chưa có file chạy là ca RẤT hay gặp (quên build).
			// Nói thẳng ở đây, đừng để người dùng phát hiện lúc flow đang chạy.
			fmt.Printf("    ! CHƯA CÓ EXECUTABLE %q — manifest đúng nhưng chưa build/chép file chạy vào\n", m.Exec)
		}
		if len(m.Quyen) == 0 {
			fmt.Println("    quyền: KHÔNG XIN GÌ (chạy với capability tối thiểu)")
		}
		for _, q := range m.Quyen {
			fmt.Printf("    quyền %-22s %s  [%s]\n", q.Khoa, q.LyDo, dauChan(q.Chan))
		}
		for _, s := range m.Secret {
			fmt.Printf("    secret %s (THAM CHIẾU, không phải giá trị)\n", s)
		}
	}
	for _, l := range b.Loi {
		fmt.Println("  ! manifest hỏng: " + l)
	}
	fmt.Println()
	if len(b.Nguon) > 0 {
		fmt.Printf("  tìm ở: %s\n", strings.Join(b.Nguon, ", "))
	}
	fmt.Println("  Xem một khai báo quyền THẬT SỰ đảm bảo gì: sagent plugin quyen")
	fmt.Println()
	if len(b.Loi) > 0 {
		os.Exit(1)
	}
}

// inBangQuyen in đúng hai cột của internal/plugin: quyền là gì, và host chặn
// được tới đâu.
func inBangQuyen() {
	fmt.Println()
	fmt.Println("  QUYỀN CỦA PLUGIN — cột [chặn] nói host có hàng rào THẬT hay không.")
	fmt.Println("  Khai một quyền trong manifest KHÔNG tự nó là một hàng rào.")
	fmt.Println()
	var chuaDo int
	for _, q := range plugin.MoiQuyen {
		fmt.Printf("  %s %-24s %s\n", dauChan(string(q.Chan)), q.Khoa, q.Mo)
		fmt.Printf("      %s\n\n", xuongDong(q.BangChung, 74, "      "))
		if q.Chan == plugin.ChuaDo {
			chuaDo++
		}
	}
	fmt.Printf("  ✓ chặn thật   ✗ KHÔNG chặn được (đã đo)   ? CHƯA ĐO\n")
	if chuaDo > 0 {
		fmt.Printf("  ! %d quyền CHƯA ĐO: không khai quyền đó KHÔNG có nghĩa là plugin bị chặn.\n", chuaDo)
	}
	fmt.Println()
}

// dauChan: ba trạng thái, ba ký hiệu KHÁC NHAU — cùng luật với `sagent nang-luc`.
// Dùng chung một ký hiệu cho "không chặn được" và "chưa đo" là xoá mất đúng thứ
// bảng này sinh ra để nói.
func dauChan(tt string) string {
	switch tt {
	case string(plugin.ChanThat):
		return "✓"
	case string(plugin.KhongChanDuoc):
		return "✗"
	default:
		return "?"
	}
}

// xuongDong bẻ một đoạn dài thành nhiều dòng cho vừa terminal.
func xuongDong(s string, rong int, thut string) string {
	tu := strings.Fields(s)
	if len(tu) == 0 {
		return ""
	}
	var dong []string
	cur := ""
	for _, t := range tu {
		if cur == "" {
			cur = t
			continue
		}
		if len([]rune(cur))+1+len([]rune(t)) > rong {
			dong = append(dong, cur)
			cur = t
			continue
		}
		cur += " " + t
	}
	dong = append(dong, cur)
	return strings.Join(dong, "\n"+thut)
}
