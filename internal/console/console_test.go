package console

import "testing"

// `go test` luôn chạy với stdout bị chuyển hướng, nên đây là vế DUY NHẤT kiểm
// tự động được — và may thay nó cũng là vế nguy hiểm hơn: đụng vào codepage của
// một console dùng chung trong khi mình chỉ đang bị ghi vào file hay ống dẫn.
//
// Vế còn lại (console thật) không giả lập được trong `go test`. Đã đo bằng tay,
// chạy qua `start /wait` để có console thật và ghi kết quả ra FILE (nếu ghi ra
// stdout thì chính việc đo đã làm stdout thôi là console):
//
//	stdout là console? true
//	codepage trước     437
//	codepage trong khi 65001   <- có đổi
//	codepage sau       437     <- có trả lại
//
// Số đo đó nằm ở docs/DO-LUONG.md. Đừng thay bằng một test giả vờ.
func TestKhongDungConsoleThiKhongDungVao(t *testing.T) {
	khoiPhuc := Dat()
	if cuKhoiPhuc != 0 {
		t.Fatalf("stdout đang bị chuyển hướng mà vẫn đổi codepage (lưu %d để khôi phục) — "+
			"đó là đụng vào console của tiến trình khác chẳng vì lý do gì", cuKhoiPhuc)
	}
	khoiPhuc() // không được hoảng
	khoiPhuc() // gọi lại vẫn phải yên
}

// KhoiPhuc phải chịu được gọi nhiều lần và gọi khi chưa hề đổi gì — vì nó được
// gọi ở MỌI lối thoát, kể cả những lối chưa từng đi qua Dat().
func TestKhoiPhucGoiBuaKhongSao(t *testing.T) {
	KhoiPhuc()
	KhoiPhuc()
	if cuKhoiPhuc != 0 {
		t.Fatal("KhoiPhuc để lại trạng thái bẩn")
	}
}
