package main

import "fmt"

// `sagent quet` — tìm tiến trình còn sống của những phiên đã tự chết.
//
// Vì sao cần một lệnh riêng: `status` chỉ liệt kê phiên ĐANG CHẠY. Phiên tự chết
// bị đánh dấu `lost` và biến khỏi bảng — nhưng đám tiến trình con nó đẻ ra có
// thể vẫn chạy, vẫn gọi API, vẫn tiêu hạn mức của bạn. Không có lệnh này thì
// không mặt nào nhìn ra chúng.
func cmdQuet(args []string) {
	giet, _ := boolFlag(args, "--giet")

	a, done := open()
	defer done()
	res, err := a.SessionSweep(giet)
	if err != nil {
		done()
		fail(err)
	}
	done()

	fmt.Println()
	if len(res) == 0 {
		fmt.Println("  ✓ Không có tiến trình mồ côi nào.")
		fmt.Println()
		return
	}

	n := 0
	for _, m := range res {
		fmt.Printf("  Phiên #%d %s (chết, PID cũ %d, bật lúc %s)\n",
			m.Session.ID, m.Session.Addr(), m.Session.PID,
			m.Session.Started.Format("15:04 02/01"))
		for _, p := range m.Procs {
			fmt.Printf("    · PID %-7d %-24s bắt đầu %s\n",
				p.PID, p.Ten, p.BatDau.Format("15:04:05 02/01"))
			n++
		}
	}
	fmt.Println()
	if giet {
		fmt.Printf("  Đã dừng %d tiến trình.\n", n)
	} else {
		// Không tự giết. Windows dùng lại PID nên danh sách này có thể lẫn thứ
		// không liên quan — nhìn tên và thời điểm rồi hãy quyết.
		fmt.Printf("  Thấy %d tiến trình. Đây là DANH SÁCH, chưa dừng gì cả.\n", n)
		fmt.Println("  Xem kỹ tên và thời điểm — Windows dùng lại PID nên có thể lẫn")
		fmt.Println("  tiến trình không liên quan. Chắc chắn thì:  sagent quet --giet")
	}
	fmt.Println()
}
