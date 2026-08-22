package main

import (
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
)

// trangThai đổi chuỗi sang kiểu trạng thái, cho bảng ca ở dưới đọc gọn.
func trangThai(s string) aiapi.TrangThaiNangLuc { return aiapi.TrangThaiNangLuc(s) }

// Lệnh `sagent nang-luc-api` phải GÕ ĐƯỢC — không chỉ tồn tại trong hợp đồng.
//
// Bài này giữ một giả định TINH VI: hai dòng dispatch của lệnh nằm trong
// `init()` của nangluc_api.go chứ không trong main.go (main.go thuộc một agent
// khác), và nó chỉ chạy được vì init trong một gói chạy theo thứ tự TÊN FILE —
// "main.go" trước "nangluc_api.go", nên bảng `commands` đã dựng xong trước khi
// init này ghi vào.
//
// Nếu giả định đó sai thì main.go sẽ ghi đè bảng và lệnh biến mất khỏi
// `sagent help` MÀ KHÔNG BÁO GÌ: gõ `sagent nang-luc-api` sẽ được hiểu thành
// một địa chỉ hồ sơ. Đúng kiểu hỏng im lặng mà cả dự án này ghét nhất, nên nó
// phải có một bài kiểm nói ra, chứ không phải một bình luận mong người ta đọc.
//
// Cùng ý với TestCoLenhNangLucGoDuoc của nửa CLI: TestNgangQuyen chỉ đòi "có
// một mục nào đó nhận action này", mà mục đó có thể là chỗ giữ chỗ `__xxx` với
// run=nil. Bảng năng lực thì không phải cờ của ai cả.
func TestLenhNangLucAPIGoDuoc(t *testing.T) {
	c, ok := commands["nang-luc-api"]
	if !ok {
		t.Fatal("không có lệnh `sagent nang-luc-api` — bảng năng lực nửa API chỉ mặt web " +
			"xem được, đúng thứ luật ngang quyền cấm. Nhiều khả năng init của main.go " +
			"đã ghi đè bảng commands sau init của nangluc_api.go")
	}
	if c.run == nil {
		t.Fatal("lệnh `nang-luc-api` không có hàm chạy")
	}
	if c.action != "api.nang-luc" {
		t.Fatalf("lệnh `nang-luc-api` gắn nhầm action %q", c.action)
	}
	// Phép đo THẬT phải có mặt trong bảng như một hành động riêng: nó chạm mạng
	// và tiêu token, khác hẳn việc đọc bảng.
	d, ok := commands["__nlado"]
	if !ok {
		t.Fatal("action api.nang-luc-do không có chỗ trong bảng commands")
	}
	if d.action != "api.nang-luc-do" {
		t.Fatalf("chỗ giữ chỗ của phép đo gắn nhầm action %q", d.action)
	}
	// Hai hành động PHẢI tách nhau. Gộp lại thì không mặt nào cho người dùng
	// thấy được cái nào tốn tiền trước khi họ bấm.
	if c.action == d.action {
		t.Fatal("đọc bảng và đo thật đang dùng chung một action")
	}
}

// Ba trạng thái phải ra BA DẤU khác nhau ở terminal.
//
// Gộp "đã đo, KHÔNG" với "chưa ai đo" thành một dấu ✗ là xoá đúng thứ bảng
// sinh ra để nói — và ở terminal thì màu không cứu được, chỉ có dấu.
func TestBaDauKhacNhauOTerminal(t *testing.T) {
	thay := map[string]string{}
	for _, tt := range []string{"lam-duoc", "khong-lam-duoc", "chua-do"} {
		d := dauNangLuc(trangThai(tt))
		if truoc, trung := thay[d]; trung {
			t.Fatalf("trạng thái %q và %q in ra cùng một dấu %q", truoc, tt, d)
		}
		thay[d] = tt
	}
	// Khối `--dan` phải in TÊN HẰNG Go, không phải chuỗi thô: dán vào mà không
	// biên dịch được thì lần sau người ta sẽ chép tay, và chép tay là chỗ mọi
	// bảng năng lực bắt đầu mục ruỗng.
	for _, tt := range []string{"lam-duoc", "khong-lam-duoc", "chua-do"} {
		if h := tenHangGo(trangThai(tt)); h == "" || h == tt {
			t.Fatalf("trạng thái %q ra tên hằng %q — dán vào mã sẽ không build", tt, h)
		}
	}
}
