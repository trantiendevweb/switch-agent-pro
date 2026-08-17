//go:build windows

// Package console lo đúng một việc: chữ tiếng Việt hiện ra đọc được.
//
// Go luôn ghi ra byte UTF-8. Console Windows thì render byte theo **codepage
// đang bật**, và mặc định của máy KHÔNG phải UTF-8 — đo trên máy dev:
//
//	OEMCP = 437     (cmd.exe dùng cái này)
//	ACP   = 1252
//
// Nghĩa là mở `cmd.exe` sạch rồi chạy `sagent` thì mọi dòng tiếng Việt thành
// rác. Toàn bộ thông điệp của công cụ này là tiếng Việt, nên đó không phải lỗi
// nhỏ về thẩm mỹ — nó làm hỏng chính thứ người dùng cần đọc lúc có sự cố.
package console

import (
	"os"

	"golang.org/x/sys/windows"
)

const utf8CP = 65001

var cuKhoiPhuc uint32 // 0 = không đổi gì, không cần khôi phục

// Dat đặt console sang UTF-8 nếu cần. Trả về hàm khôi phục (gọi được nhiều lần).
//
// Hai điều kiện, cả hai đều có lý do:
//
//   - CHỈ đổi khi stdout là console thật. Bị chuyển hướng vào file hay ống dẫn
//     thì byte UTF-8 vốn đã đúng, đổi codepage chẳng giúp gì mà còn đụng vào
//     console của người khác.
//   - CHỈ đổi khi codepage hiện tại khác 65001, và KHÔI PHỤC lại lúc thoát.
//     Codepage là tài sản chung của cả cửa sổ console: đổi rồi bỏ đó thì những
//     lệnh chạy sau `sagent` — kể cả công cụ của người khác — sẽ hiển thị hoặc
//     phân tích sai. Mượn thì phải trả.
func Dat() func() {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return func() {} // không phải console (file/pipe) — không đụng vào
	}
	cu, err := windows.GetConsoleOutputCP()
	if err != nil {
		return func() {}
	}
	if cu == utf8CP {
		return func() {} // đã đúng sẵn
	}
	if err := windows.SetConsoleOutputCP(utf8CP); err != nil {
		return func() {} // không đặt được thì thôi, đừng làm hỏng thêm
	}
	cuKhoiPhuc = cu
	return KhoiPhuc
}

// KhoiPhuc trả codepage về như lúc đầu. Idempotent — gọi mấy lần cũng được.
//
// Phải gọi tường minh ở MỌI lối thoát: `fail()` và mấy chỗ `os.Exit` khác đều
// bỏ qua defer, nên chỉ dựa vào `defer` trong main là khôi phục hụt.
func KhoiPhuc() {
	if cuKhoiPhuc == 0 {
		return
	}
	_ = windows.SetConsoleOutputCP(cuKhoiPhuc)
	cuKhoiPhuc = 0
}
