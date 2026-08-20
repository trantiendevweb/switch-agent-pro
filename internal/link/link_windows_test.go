//go:build windows

package link

import (
	"os"
	"path/filepath"
	"testing"
)

// dungDich dựng một thư mục "dữ liệu thật" có nội dung nhận dạng được, để mọi
// test sau đó chứng minh được dữ liệu đó CÒN NGUYÊN.
func dungDich(t *testing.T, base, ten string) string {
	t.Helper()
	dich := filepath.Join(base, ten)
	if err := os.MkdirAll(filepath.Join(dich, "thu-muc-con"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dich, "du-lieu.txt"), []byte("dữ liệu thật"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dich
}

// dichConNguyen là phép đo kết luận của cả file này.
func dichConNguyen(t *testing.T, dich string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dich, "du-lieu.txt"))
	if err != nil {
		t.Fatalf("DỮ LIỆU ĐÍCH ĐÃ MẤT: %v", err)
	}
	if string(b) != "dữ liệu thật" {
		t.Fatalf("dữ liệu đích bị sửa: %q", b)
	}
	if _, err := os.Stat(filepath.Join(dich, "thu-muc-con")); err != nil {
		t.Fatalf("thư mục con của đích đã mất: %v", err)
	}
}

// Vòng đời đầy đủ của junction — tạo, đọc xuyên qua, gỡ — và điều kiện sống còn
// của cả dự án: GỠ LINK KHÔNG ĐƯỢC XOÁ DỮ LIỆU Ở ĐẦU BÊN KIA.
//
// Vì sao đo đúng chỗ này: junction đã xoá mất ~/.claude trên máy dev ngày
// 2026-08-17 (docs/DO-LUONG.md). internal/profile có test cho tầng trên, nhưng
// tầng dưới — bốn hàm trong package này — trước đó KHÔNG có test nào
// (go test ./internal/link in ra "no test files").
//
// Số đo trên máy này (Windows Server 2022, go1.25.13):
//
//	LinkDir            -> nil, dựng junction bằng `mklink /J`, KHÔNG cần quyền admin
//	IsLink(junction)   -> true  (cờ FILE_ATTRIBUTE_REPARSE_POINT)
//	os.ReadDir(link)   -> ĐI XUYÊN: liệt kê đúng 2 mục ruột của đích
//	Unlink(link, true) -> nil, link biến mất, đích còn nguyên 2 mục
func TestJunctionTaoDocXuyenVaGoKhongLamMatDuLieuDich(t *testing.T) {
	base := t.TempDir()
	dich := dungDich(t, base, "kho-that")
	noiChung := filepath.Join(base, "noi-chung")

	if err := LinkDir(dich, noiChung); err != nil {
		t.Fatalf("LinkDir = %v", err)
	}

	laLink, err := IsLink(noiChung)
	if err != nil {
		t.Fatalf("IsLink = %v", err)
	}
	if !laLink {
		t.Fatal("IsLink nói junction vừa tạo KHÔNG phải link — đây chính là lỗ hổng " +
			"làm code gọi os.RemoveAll và xoá xuyên qua đích")
	}

	// Junction phải đi xuyên khi ĐỌC: đó là cả công dụng của nó.
	es, err := os.ReadDir(noiChung)
	if err != nil {
		t.Fatalf("ReadDir qua junction = %v", err)
	}
	if len(es) != 2 {
		t.Fatalf("đọc qua junction thấy %d mục, chờ 2 (thu-muc-con/, du-lieu.txt)", len(es))
	}
	b, err := os.ReadFile(filepath.Join(noiChung, "du-lieu.txt"))
	if err != nil || string(b) != "dữ liệu thật" {
		t.Fatalf("đọc file qua junction = %q, %v", b, err)
	}

	if err := Unlink(noiChung, true); err != nil {
		t.Fatalf("Unlink = %v", err)
	}
	if _, err := os.Lstat(noiChung); err == nil {
		t.Fatal("Unlink trả nil nhưng junction vẫn còn — hỏng kiểu im lặng")
	}
	dichConNguyen(t, dich)
}

// Lưới an toàn thứ hai: gọi Unlink với CỜ SAI (isDir=false trên một junction).
// Chỗ gọi thật lấy cờ này từ os.DirEntry.IsDir(), mà junction thì IsDir()=false
// (Lstat trả ModeIrregular) — nên nhánh os.Remove là nhánh HAY CHẠY THẬT, không
// phải trường hợp hiếm.
//
// Số đo: os.Remove trên junction gọi RemoveDirectory, gỡ đúng điểm nối và
// KHÔNG đụng ruột đích. Chốt lại để bản sửa sau đừng đổi thành os.RemoveAll.
func TestUnlinkCoSaiVanKhongXoaXuyenQua(t *testing.T) {
	base := t.TempDir()
	dich := dungDich(t, base, "kho-that")
	noiChung := filepath.Join(base, "noi-chung")

	if err := LinkDir(dich, noiChung); err != nil {
		t.Fatalf("LinkDir = %v", err)
	}
	if err := Unlink(noiChung, false); err != nil {
		t.Fatalf("Unlink(isDir=false) trên junction = %v", err)
	}
	if _, err := os.Lstat(noiChung); err == nil {
		t.Fatal("cờ sai làm junction không được gỡ")
	}
	dichConNguyen(t, dich)
}

// IsLink phải PHÂN BIỆT được, chứ không phải lúc nào cũng nói một câu.
// Một hàm luôn trả true thì code gọi sẽ bỏ qua thư mục thật cần dọn; luôn trả
// false thì nó xoá xuyên link. Test chốt cả hai đầu, cộng với đường lỗi.
func TestIsLinkPhanBietThuMucThatFileThatVaDuongDanThieu(t *testing.T) {
	base := t.TempDir()
	dich := dungDich(t, base, "kho-that")

	if laLink, err := IsLink(dich); err != nil || laLink {
		t.Fatalf("IsLink(thư mục thật) = %v, %v — chờ false, nil", laLink, err)
	}
	if laLink, err := IsLink(filepath.Join(dich, "du-lieu.txt")); err != nil || laLink {
		t.Fatalf("IsLink(file thật) = %v, %v — chờ false, nil", laLink, err)
	}
	// Đường dẫn không tồn tại phải trả LỖI, không phải (false, nil): chỗ gọi
	// thật dùng dạng `isLink, _ := IsLink(p)`, nên "false" ở đây bị hiểu là
	// "thư mục thường" và mở đường cho một lần xoá lẽ ra phải dừng.
	if _, err := IsLink(filepath.Join(base, "khong-ton-tai")); err == nil {
		t.Fatal("IsLink nuốt lỗi cho đường dẫn không tồn tại")
	}
}

// Đường dẫn có DẤU CÁCH. LinkDir/Unlink chạy qua `cmd /c`, mà cmd.exe tách tham
// số theo luật riêng, khác luật Go dựng dòng lệnh. Đây là ca hỏng kinh điển, và
// nó KHÔNG hiếm: kho hồ sơ nằm dưới %USERPROFILE%, tức C:\Users\Nguyen Van A\
// trên rất nhiều máy thật.
//
// Số đo: cả hai lệnh chạy đúng với tên có dấu cách — bằng chứng để bản sửa sau
// đừng tự ý bỏ trích dẫn.
func TestJunctionChayDungKhiDuongDanCoDauCach(t *testing.T) {
	base := t.TempDir()
	dich := dungDich(t, base, "kho that cua toi")
	noiChung := filepath.Join(base, "noi chung cua toi")

	if err := LinkDir(dich, noiChung); err != nil {
		t.Fatalf("LinkDir với đường dẫn có dấu cách = %v", err)
	}
	if laLink, _ := IsLink(noiChung); !laLink {
		t.Fatal("không dựng được junction khi đường dẫn có dấu cách")
	}
	if _, err := os.ReadFile(filepath.Join(noiChung, "du-lieu.txt")); err != nil {
		t.Fatalf("đọc qua junction (đường dẫn có dấu cách) = %v", err)
	}
	if err := Unlink(noiChung, true); err != nil {
		t.Fatalf("Unlink với đường dẫn có dấu cách = %v", err)
	}
	if _, err := os.Lstat(noiChung); err == nil {
		t.Fatal("Unlink im lặng không gỡ gì khi đường dẫn có dấu cách")
	}
	dichConNguyen(t, dich)
}

// LinkDir phải BÁO LỖI khi chỗ đặt link đã bị chiếm, chứ không được đè.
// Nếu nó nuốt lỗi, hồ sơ sẽ tưởng mình đã nối phần dùng chung trong khi thực ra
// đang dùng một thư mục rỗng — cấu hình biến mất mà không ai báo.
func TestLinkDirTuChoiKhiChoDatLinkDaBiChiem(t *testing.T) {
	base := t.TempDir()
	dich := dungDich(t, base, "kho-that")
	choDat := filepath.Join(base, "da-co-san")
	if err := os.MkdirAll(choDat, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := LinkDir(dich, choDat); err == nil {
		t.Fatal("LinkDir trả nil dù chỗ đặt link đã tồn tại — sẽ tưởng đã nối xong")
	}
	if laLink, _ := IsLink(choDat); laLink {
		t.Fatal("thư mục có sẵn bị thay bằng junction")
	}
	dichConNguyen(t, dich)
}

// GHI NHẬN MỘT CÁI BẪY, không phải khen một tính năng.
//
// Số đo: `mklink /J` KHÔNG kiểm tra đích có tồn tại không — LinkDir trả nil và
// để lại một junction TREO (IsLink=true nhưng đọc vào thì lỗi). Nghĩa là
// "LinkDir == nil" KHÔNG chứng minh phần dùng chung đã nối được; chỗ gọi phải
// tự kiểm tra đích trước. Chốt lại để nếu sau này Windows/mklink đổi hành vi
// thì test này gãy và người sửa biết mà xem lại chỗ gọi.
func TestLinkDirVoiDichKhongTonTaiVanTraNilNhungTaoJunctionTreo(t *testing.T) {
	base := t.TempDir()
	treo := filepath.Join(base, "treo")

	if err := LinkDir(filepath.Join(base, "dich-khong-co"), treo); err != nil {
		t.Skipf("mklink /J đã đổi hành vi, nay từ chối đích không tồn tại: %v", err)
	}
	if laLink, _ := IsLink(treo); !laLink {
		t.Fatal("LinkDir trả nil mà không tạo ra gì cả")
	}
	if _, err := os.ReadDir(treo); err == nil {
		t.Fatal("đọc được qua junction treo — đích vốn không tồn tại")
	}
	// Junction treo vẫn phải gỡ được, nếu không hồ sơ sẽ kẹt không dọn nổi.
	if err := Unlink(treo, true); err != nil {
		t.Fatalf("không gỡ được junction treo: %v", err)
	}
}

// LinkFile có ba nhánh (symlink -> hardlink -> chép) và nhánh nào chạy là tuỳ
// MÁY: symlink cần Developer Mode hoặc quyền admin, hardlink cần cùng ổ đĩa.
// Nên test không đoán nhánh — nó đo hai thứ đúng với CẢ BA nhánh:
//
//  1. đọc qua link ra đúng nội dung nguồn
//  2. gỡ link xong, FILE NGUỒN CÒN NGUYÊN
//
// (Trên máy chạy test này nhánh symlink thắng: os.Lstat cho ModeSymlink.)
func TestLinkFileDocDuocVaGoKhongLamMatFileNguon(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "settings.json")
	noiDung := []byte("{\"khoa\":\"giá trị\"}")
	if err := os.WriteFile(src, noiDung, 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(base, "noi-toi.json")

	if err := LinkFile(src, dst); err != nil {
		t.Fatalf("LinkFile = %v", err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("đọc qua link = %v", err)
	}
	if string(b) != string(noiDung) {
		t.Fatalf("đọc qua link ra %q, chờ %q", b, noiDung)
	}
	if fi, err := os.Lstat(dst); err == nil {
		t.Logf("nhánh LinkFile trúng trên máy này: mode=%v", fi.Mode())
	}

	if err := Unlink(dst, false); err != nil {
		t.Fatalf("Unlink = %v", err)
	}
	if _, err := os.Lstat(dst); err == nil {
		t.Fatal("Unlink trả nil nhưng link vẫn còn")
	}
	b, err = os.ReadFile(src)
	if err != nil {
		t.Fatalf("FILE NGUỒN ĐÃ MẤT sau khi gỡ link: %v", err)
	}
	if string(b) != string(noiDung) {
		t.Fatalf("file nguồn bị sửa: %q", b)
	}
}
