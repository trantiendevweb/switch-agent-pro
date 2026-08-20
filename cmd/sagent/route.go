package main

import (
	"fmt"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// `sagent route` — sổ đăng ký route (action "route.list").
//
//	sagent route            đối chiếu: cấu hình khai gì, sổ ghi đã gọi thật qua đâu
//	sagent route ds|list    y hệt, cho ai quen gõ động từ
//
// Khác `sagent api ds` ở chỗ quyết định: `api ds` chỉ đọc file cấu hình, tức là
// nó chỉ biết những gì người ta KHAI. Bảng này ghép thêm sổ — thứ ghi lại route
// đã THẬT SỰ được gọi. Hai chiều lệch nói hai chuyện, và cả hai đều là chuyện
// tiền: route khai mà chưa gọi bao giờ thì chưa ai biết nó có chạy được không;
// route có trong sổ mà không còn trong cấu hình thì hoá đơn sẽ có dòng mà cấu
// hình không giải thích được.
//
// `route create` và `route test` (MASTER-PLAN Pha 1) CHƯA làm — cố ý không dựng
// khung rỗng cho chúng ở đây, vì một lệnh có tên mà không làm gì còn khó hiểu
// hơn một lệnh chưa có.
func cmdRoute(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "ds", "list":
		default:
			fail(fmt.Errorf("chưa có `route %s`. Hiện chỉ có: sagent route [ds]", args[0]))
		}
	}
	a, done := open()
	defer done()
	ds, err := a.RouteList()
	done()
	if err != nil {
		fail(err)
	}
	fmt.Println()
	if len(ds) == 0 {
		fmt.Println("  Chưa có route nào — chưa khai trong cấu hình, cũng chưa gọi bao giờ.")
		fmt.Println("  Khai route: sagent api ds  (in sẵn mẫu cần dán vào .sagent/project.toml)")
		fmt.Println()
		return
	}
	fmt.Println("  Sổ route ↔ cấu hình")
	fmt.Println()
	for _, m := range ds {
		dau := "·"
		if m.Lech() {
			dau = "!"
		}
		duong := m.BaseURLCauHinh
		if duong == "" {
			duong = m.BaseURLSo
		}
		// In key_id, KHÔNG in key — sổ cũng chỉ giữ đúng chừng đó.
		fmt.Printf("  %s %-12s %-20s %-32s %-14s key:%s\n",
			dau, m.Ten, nhan(m.TrangThai), duong, m.Model, m.KeyID)
		if m.TrangThai == store.SoLechDuong {
			fmt.Printf("      sổ ghi: %s\n", m.BaseURLSo)
		}
	}
	fmt.Println()
}

// nhan đổi trạng thái đối chiếu sang câu người đọc hiểu ngay ở NGỮ CẢNH ROUTE.
// Chuỗi gốc trong store nói "sổ/đĩa" vì nó dùng chung với bảng hồ sơ; ở đây
// "đĩa" thật ra là file cấu hình, gọi đúng tên thì đỡ phải đoán.
func nhan(t string) string {
	switch t {
	case store.SoThieuSo:
		return "khai, chưa gọi"
	case store.SoThieuDia:
		return "đã gọi, hết khai"
	case store.SoLechDuong:
		return "lệch base_url"
	default:
		return t
	}
}
