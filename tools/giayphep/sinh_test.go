// Test cho chính khâu sinh THONG-BAO-GIAY-PHEP.txt.
//
// Vì sao: công cụ này là thứ duy nhất canh cam kết giấy phép (MASTER-PLAN mục 0)
// trong CI, mà bản thân nó chưa có test nào. Điểm gãy nguy nhất không phải là
// sinh sai văn bản — mà là IM LẶNG bỏ qua một module không tìm được file giấy
// phép, vì lúc đó bản phát hành thiếu thông báo bản quyền mà không ai biết.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func moduleGia(t *testing.T, ten, noiDungGiayPhep string) modun {
	t.Helper()
	dir := t.TempDir()
	if noiDungGiayPhep != "" {
		if err := os.WriteFile(filepath.Join(dir, "LICENSE"), []byte(noiDungGiayPhep), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return modun{Path: ten, Version: "v1.2.3", Dir: dir}
}

// Nhận đủ các tên file giấy phép thường gặp — nhận thiếu một tên là báo "thiếu
// giấy phép" oan, và người ta sẽ học cách bỏ qua lời báo đó.
func TestDocGiayPhepNhanMoiTenThuongGap(t *testing.T) {
	for _, ten := range tenGiayPhep {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ten), []byte("bản quyền abc"), 0o644); err != nil {
			t.Fatal(err)
		}
		vb, thay := docGiayPhep(dir)
		if vb != "bản quyền abc" || thay != ten {
			t.Errorf("%s: đọc được (%q, %q), muốn (%q, %q)", ten, vb, thay, "bản quyền abc", ten)
		}
	}
}

func TestDocGiayPhepKhongDoanBuaKhiThieu(t *testing.T) {
	dir := t.TempDir()
	// README không phải giấy phép: đoán theo nội dung là cách nhanh nhất để dán
	// nhầm một giấy phép sai vào bản phát hành.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("MIT License"), 0o644); err != nil {
		t.Fatal(err)
	}
	if vb, ten := docGiayPhep(dir); vb != "" || ten != "" {
		t.Errorf("nhận nhầm %q làm giấy phép", ten)
	}
}

// Đây là hành vi quan trọng nhất của công cụ: module không có file giấy phép thì
// phải LỘ RA trong danh sách thiếu, chứ không bị bỏ qua lặng lẽ.
func TestDungVanBanBaoModuleThieuGiayPhep(t *testing.T) {
	mods := []modun{
		moduleGia(t, "example.com/co", "GIẤY PHÉP MIT ĐẦY ĐỦ"),
		moduleGia(t, "example.com/khong", ""),
	}
	vb, thieu := dungVanBan(mods)
	if len(thieu) != 1 || thieu[0] != "example.com/khong" {
		t.Fatalf("danh sách thiếu = %v, muốn [example.com/khong]", thieu)
	}
	if !strings.Contains(vb, "GIẤY PHÉP MIT ĐẦY ĐỦ") {
		t.Error("không chép toàn văn giấy phép của module có sẵn — đó là toàn bộ lý do file tồn tại")
	}
	// Cả hai vẫn phải có mặt trong danh sách đầu file: người đọc cần thấy binary
	// gồm những gì, kể cả module đang thiếu văn bản giấy phép.
	for _, m := range mods {
		if !strings.Contains(vb, m.Path+" "+m.Version) {
			t.Errorf("danh sách thiếu %s %s", m.Path, m.Version)
		}
	}
}

// Văn bản sinh ra phải ổn định: `-kiem` so bằng byte, nên hai lần sinh cùng đầu
// vào mà khác nhau là CI đỏ ngẫu nhiên.
func TestDungVanBanOnDinh(t *testing.T) {
	mods := []modun{moduleGia(t, "example.com/a", "A"), moduleGia(t, "example.com/b", "B")}
	if a, _ := dungVanBan(mods); a != mustVanBan(t, mods) {
		t.Error("hai lần sinh cùng đầu vào cho kết quả khác nhau")
	}
	vb, _ := dungVanBan(mods)
	for _, phai := range []string{"Đừng sửa tay", "go run ./tools/giayphep", "MIT"} {
		if !strings.Contains(vb, phai) {
			t.Errorf("phần đầu file thiếu %q — người đọc phải biết file này do máy sinh và sinh lại bằng lệnh nào", phai)
		}
	}
}

func mustVanBan(t *testing.T, mods []modun) string {
	t.Helper()
	vb, _ := dungVanBan(mods)
	return vb
}

// Giấy phép chép ra phải là LF thuần: kho này chạy trên Windows, và một file
// giấy phép có CRLF sẽ làm `-kiem` báo lệch trên máy khác dù nội dung y hệt.
func TestDungVanBanChuanHoaXuongDong(t *testing.T) {
	mods := []modun{moduleGia(t, "example.com/crlf", "dòng 1\r\ndòng 2\r\n")}
	vb, thieu := dungVanBan(mods)
	if len(thieu) != 0 {
		t.Fatalf("thiếu bất ngờ: %v", thieu)
	}
	if strings.Contains(vb, "\r") {
		t.Error("còn ký tự CR trong văn bản sinh ra — `-kiem` sẽ đỏ tuỳ máy")
	}
}

// lietKe chỉ được trả module CÓ phiên bản: module chính của repo không có phiên
// bản và cũng không phải phụ thuộc — kê nó vào là tự kê giấy phép của mình.
func TestLietKeKhongKeModuleChinh(t *testing.T) {
	t.Chdir(thuMucGoc)
	mods, err := lietKe()
	if err != nil {
		t.Skipf("bỏ qua: không chạy được go list (%v)", err)
	}
	if len(mods) == 0 {
		t.Fatal("không liệt kê được phụ thuộc nào")
	}
	for i, m := range mods {
		if m.Version == "" || m.Dir == "" {
			t.Errorf("%s: thiếu phiên bản/thư mục", m.Path)
		}
		if strings.HasPrefix(m.Path, "github.com/trantiendevweb/switch-agent-pro") {
			t.Errorf("kê chính module của dự án (%s) như một phụ thuộc", m.Path)
		}
		if i > 0 && mods[i-1].Path >= m.Path {
			t.Errorf("danh sách không sắp xếp tăng dần: %s đứng trước %s", mods[i-1].Path, m.Path)
		}
	}
}
