package link

import (
	"os"
	"path/filepath"
	"testing"
)

// copyFile là ĐƯỜNG LUI cuối cùng của LinkFile: khi cả symlink lẫn hardlink đều
// bị từ chối (máy không bật Developer Mode, hoặc hai đầu khác ổ đĩa), hồ sơ vẫn
// phải dùng được. Nên nó không được phép "gần đúng": nội dung phải khớp từng byte.
//
// Khác biệt phải nhớ khi đọc test này: bản chép KHÔNG phải link. Sửa đích thì
// bản chép không đổi theo — đó là cái giá của đường lui, và là lý do LinkFile
// chỉ dùng nó sau khi hai cách thật đã hỏng.
func TestCopyFileChepDungTungByteVaKhongDongBoNguoc(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "goc.txt")
	dst := filepath.Join(base, "ban-chep.txt")

	// Có byte 0 và ký tự nhiều byte: bắt lỗi kiểu "đọc theo dòng" hay cắt ở NUL.
	goc := []byte("dòng một\x00dòng hai\nkết thúc")
	if err := os.WriteFile(src, goc, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile = %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(goc) {
		t.Fatalf("bản chép sai nội dung:\n có: %q\nchờ: %q", got, goc)
	}

	// Ghi đè đích rồi chép lại: copyFile phải THAY nội dung cũ, không nối thêm.
	ngan := []byte("ngắn")
	if err := os.WriteFile(src, ngan, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile lần hai = %v", err)
	}
	got, _ = os.ReadFile(dst)
	if string(got) != string(ngan) {
		t.Fatalf("chép đè để lại rác của lần trước: %q", got)
	}
}

// Nguồn không tồn tại phải BÁO LỖI, không được lặng lẽ tạo file rỗng ở đích.
// Nếu nuốt lỗi, LinkFile sẽ trả nil và hồ sơ mới có một file rỗng thay cho
// cấu hình thật — hỏng kiểu im lặng, khó lần ra nhất.
func TestCopyFileNguonThieuThiBaoLoiVaKhongTaoFileRong(t *testing.T) {
	base := t.TempDir()
	dst := filepath.Join(base, "dich.txt")

	if err := copyFile(filepath.Join(base, "khong-ton-tai.txt"), dst); err == nil {
		t.Fatal("copyFile nuốt lỗi khi nguồn không tồn tại")
	}
	if _, err := os.Stat(dst); err == nil {
		t.Fatal("copyFile đã tạo file đích dù nguồn không đọc được")
	}
}
