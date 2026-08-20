package main

import (
	"fmt"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Mặt TERMINAL của ba trạng thái phiên đo được (schema v9).
//
// Trước đây `sagent status` chỉ có một bảng "phiên đang chạy", và phiên tự chết
// biến mất hoàn toàn khỏi mọi mặt trừ `sagent quet`. Người vận hành thấy hạm
// đội trống trơn mà không biết nó trống vì việc xong, vì hết hạn mức, hay vì
// token chết — ba việc phải làm khác hẳn nhau.
//
// Hàm ở đây CHỈ ĐỊNH DẠNG. Trạng thái đã được quyết một lần ở
// api.phanLoaiPhienChet; terminal không suy thêm gì, đúng như dashboard 2D và
// màn 3D cũng không suy thêm gì.

// nhanTrangThai đổi mã trạng thái thành nhãn tiếng Việt kèm việc phải làm.
//
// `lost` giữ nguyên nghĩa "KHÔNG BIẾT vì sao" — nói thẳng ra thế, vì bịa một
// lý do nghe hợp lý sẽ khiến người ta đi sửa nhầm chỗ.
func nhanTrangThai(state string) string {
	switch state {
	case store.StateHanMuc:
		return "hết hạn mức"
	case store.StateChan:
		return "bị chặn quyền"
	case store.StateHong:
		return "lỗi nhà cung cấp"
	case store.StateLost:
		return "chết, chưa rõ vì sao"
	case store.StateXong:
		// KHÔNG phải một kiểu chết — lượt chạy kết thúc bình thường. Trước
		// 20/08 nó không có tên riêng nên đọc ra "chết, chưa rõ vì sao", lẫn
		// vào đám phiên chết thật.
		return "xong"
	case store.StateStopped:
		return "đã dừng"
	case store.StateRunning:
		return "đang chạy"
	}
	return state
}

// dongPhienHong dựng MỘT dòng bảng cho một phiên đã kết thúc bất thường.
//
// `now` truyền vào chứ không gọi time.Now() bên trong: mốc hạn mức cấp lại chỉ
// có nghĩa khi so với một thời điểm, và hàm nhận thời điểm đó thì test kiểm
// được chữ nó in ra thay vì phải chờ đồng hồ.
func dongPhienHong(s store.Session, now time.Time) string {
	d := fmt.Sprintf("   #%-3d %-18s %-20s", s.ID, s.Addr(), nhanTrangThai(s.State))
	if s.HanMucDenLai > 0 {
		lai := time.Unix(s.HanMucDenLai, 0)
		if con := lai.Sub(now); con > 0 {
			d += fmt.Sprintf("  cấp lại sau %s (%s)",
				con.Truncate(time.Minute), lai.Format("15:04 02/01"))
		} else {
			d += fmt.Sprintf("  đã cấp lại lúc %s — chạy lại được", lai.Format("15:04 02/01"))
		}
	}
	if s.StateLyDo != "" {
		d += "\n        " + s.StateLyDo
	}
	return d
}
