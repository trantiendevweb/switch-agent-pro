package main

import (
	"context"
	"fmt"
	"time"

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
		case "kiem", "--kiem":
			routeKiem()
			return
		default:
			fail(fmt.Errorf("chưa có `route %s`. Hiện chỉ có: sagent route [ds|kiem]", args[0]))
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

// `sagent route kiem` — action "route.kiem". Hỏi từng route CÓ DÙNG ĐƯỢC KHÔNG.
//
// Khác `sagent route` ở chỗ nó chạm mạng thật: bảng kia đối chiếu giấy tờ (cấu
// hình khai gì, sổ ghi đã gọi qua đâu), còn bảng này đi hỏi nhà cung cấp. Đo
// được 20/08: route `deepseek` trả HTTP 503 ba lần lúc 16:54–16:56 rồi tự hồi
// phục — và cách duy nhất để biết, khi chưa có lệnh này, là gọi thật rồi hỏng.
//
// KHÔNG tốn token: đi bằng `GET /models`. Một phép kiểm có tính tiền thì người
// ta sẽ thôi chạy nó, và health check không ai chạy thì bằng không có.
func routeKiem() {
	a, done := open()
	ds := a.RouteKiem(context.Background())
	done()

	fmt.Println()
	if len(ds) == 0 {
		fmt.Println("  Chưa khai route nào trong .sagent/project.toml.")
		fmt.Println()
		return
	}
	fmt.Println("  Kiểm route (GET /models — không tốn token)")
	fmt.Println()
	for _, s := range ds {
		dau, nhan := "✗", "không dùng được"
		switch {
		case s.Dung():
			dau, nhan = "✓", "dùng được"
		case s.Song:
			// Nhà cung cấp còn đó nhưng model khai không có thật. Đây KHÁC hẳn
			// route chết, và cách sửa cũng khác: sửa cấu hình, không phải đợi.
			dau, nhan = "!", "sống, nhưng model khai không có"
		}
		fmt.Printf("  %s %-12s %-32s %s\n", dau, s.Ten, nhan, s.Mat.Round(time.Millisecond))

		if s.Song && s.SoModel > 0 {
			fmt.Printf("      model %q · nhà cung cấp liệt kê %d model\n", s.Model, s.SoModel)
		}
		if s.KhongRo {
			// Im lặng KHÁC phủ nhận: endpoint không cài /models thì không được
			// kết luận model khai sai.
			fmt.Println("      endpoint không liệt kê model — không kiểm được tên model, chỉ biết là route sống")
		}
		for _, g := range s.Gan {
			fmt.Printf("      có thể bạn muốn: %s\n", g)
		}
		if s.Loi != "" {
			// Nguyên văn, giữ cả request id — đó là thứ duy nhất dùng được khi
			// phải hỏi lại nhà cung cấp.
			ma := ""
			if s.Status > 0 {
				ma = fmt.Sprintf("HTTP %d: ", s.Status)
			}
			fmt.Printf("      %s%s\n", ma, s.Loi)
		}
	}
	fmt.Println()
	fmt.Println("  Lưu ý: phép kiểm này KHÔNG biết hạn mức còn hay hết — cái đó chỉ lộ ra khi gọi thật.")
	fmt.Println()
}
