package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/nhatky"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Mặt TERMINAL của nhật ký phiên.
//
// Không có lệnh này thì tính năng coi như không tồn tại: nhật ký nằm dưới
// ~/.ai-accounts/.nhat-ky với tên mang mốc thời gian, và không ai đi dò tay để
// biết file nào là phiên #167. Ánh xạ "số phiên → đường dẫn" nằm ở cột `log`
// của sổ, và đây là chỗ hỏi nó.

// dongMacDinh là số dòng cuối in ra khi không nói gì khác.
//
// Có trần chứ không in cả file, và đây là một trong hai chỗ chặn dung lượng
// (chỗ kia là nhatky.Don). Nhật ký stream-json của một lượt dài có thể tới hàng
// chục MB — đổ hết ra terminal thì vừa không đọc được vừa treo màn hình. 200
// dòng cuối chứa dòng `{"type":"result"}` và lý do chết, tức đúng phần đáng đọc.
const dongMacDinh = 200

func cmdNhatKy(args []string) {
	het, args := boolFlag(args, "--het")
	dong, args := intFlag(args, "--dong", dongMacDinh)
	if het {
		dong = 0 // 0 = lấy hết
	}

	a, done := open()
	defer done()

	if len(args) == 0 {
		nhatKyBang(a)
		return
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fail(fmt.Errorf("không hiểu '%s' — cần SỐ phiên, xem bằng: sagent nhat-ky", args[0]))
	}
	m, noiDung, err := a.SessionNhatKyDoc(id, dong)
	if err != nil {
		fail(err)
	}
	done() // vẽ hết event trước khi in bảng của riêng lệnh này
	fmt.Println()
	fmt.Printf("  #%d  %s  —  %s\n", m.ID, m.Addr, nhanTrangThai(m.State))
	if m.LyDo != "" {
		fmt.Printf("  %s\n", m.LyDo)
	}
	fmt.Printf("  %s  (%s)\n", m.Duong, coChu(m.Co))
	if dong > 0 {
		fmt.Printf("  %d dòng cuối — cả file: sagent nhat-ky %d --het\n", dong, id)
	}
	fmt.Println()
	fmt.Println(noiDung)
	fmt.Println()
}

// nhatKyBang liệt kê nhật ký gần đây.
func nhatKyBang(a *api.API) {
	ds, err := a.SessionNhatKyDS(20)
	if err != nil {
		fail(err)
	}
	fmt.Println()
	if len(ds) == 0 {
		fmt.Println("  Sổ chưa có phiên nào.")
		fmt.Println("  Bật thử: sagent fleet claude:<tên> --copies 2 -- -p \"...\"")
		fmt.Println()
		return
	}
	fmt.Println("  Nhật ký phiên  (" + nhatky.Root() + ")")
	fmt.Println()
	for _, m := range ds {
		co := coChu(m.Co)
		if !m.ConFile {
			// Nói THẲNG là không còn, và nói vì sao có thể không còn. Im lặng ở
			// đây thì người đọc tưởng công cụ làm mất.
			co = "—"
		}
		fmt.Printf("   #%-3d %-18s %-20s %8s  %s\n",
			m.ID, m.Addr, nhanTrangThai(m.State), co, tenNgan(m.Duong, m.ConFile))
	}
	fmt.Println()
	fmt.Printf("  Đọc một phiên: sagent nhat-ky <số>   (giữ tối đa %d file / %d MB, cũ nhất bị dọn trước)\n",
		nhatky.SoFileToiDa, nhatky.TongByteToiDa>>20)
	fmt.Println()
}

// tenNgan chỉ in TÊN FILE trong bảng — cả đường dẫn thì bảng vỡ dòng trên mọi
// terminal hẹp. Đường dẫn đầy đủ in ở lệnh đọc một phiên.
func tenNgan(duong string, conFile bool) string {
	if duong == "" {
		return "(phiên không có nhật ký)"
	}
	ten := duong
	if i := strings.LastIndexAny(duong, `/\`); i >= 0 {
		ten = duong[i+1:]
	}
	if !conFile {
		return ten + "  (đã dọn hoặc không đọc được)"
	}
	return ten
}

// coChu đọc số byte thành chữ.
func coChu(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// dongNhatKyPhien dựng dòng chỉ đường tới nhật ký, dùng chung cho `sagent
// status` (cả bảng phiên sống lẫn bảng phiên chết).
//
// Một hàm chứ hai chỗ chép tay: hai bảng mà lệch cách gọi thì người đọc bảng
// này học được cách tra, sang bảng kia lại không tra được.
func dongNhatKyPhien(s store.Session) string {
	if s.Log == "" {
		return ""
	}
	return fmt.Sprintf("        nhật ký: sagent nhat-ky %d", s.ID)
}
