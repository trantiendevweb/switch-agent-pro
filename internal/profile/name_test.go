package profile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// homeGia trỏ paths.Home() vào thư mục tạm để test không đụng máy thật.
func homeGia(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	} else {
		t.Setenv("HOME", tmp)
	}
	return tmp
}

// Đây là phép đo an toàn quan trọng thứ hai của dự án (sau "xoá không xuyên link").
//
// Công cụ này XOÁ THƯ MỤC, và tên hồ sơ đến thẳng từ người dùng — dòng lệnh,
// hoặc form trên dashboard đang mở ra internet. Nếu tên được ghép thẳng vào
// đường dẫn thì `claude:../../.claude` trỏ ra ngoài kho, và `sagent xoa` sẽ xoá
// đúng dữ liệu thật mà nó vốn phải bảo vệ.
//
// Test này chốt cả hai lớp: tên phải bị TỪ CHỐI, và kể cả khi lọt qua thì thư
// mục ngoài kho vẫn không được xoá.
func TestTenHoSoKhongDuocThoatRaNgoaiKho(t *testing.T) {
	doc := []string{
		"../../.claude",
		"..",
		".",
		"a/b",
		`a\b`,
		"C:/Windows",
		"/etc",
		"", // rỗng: filepath.Join nuốt mất, hồ sơ hoá thành cả thư mục provider
		strings.Repeat("x", 200),
	}
	for _, ten := range doc {
		if err := ValidName(ten); err == nil {
			t.Errorf("ValidName(%q) = nil — tên nguy hiểm này lẽ ra phải bị từ chối", ten)
		}
	}

	// Tên bình thường phải qua được, nếu không thì bản vá đã chặn nhầm người dùng thật.
	for _, ten := range []string{"phu", "cong-ty", "work_2", "acc.1", "Phu2"} {
		if err := ValidName(ten); err != nil {
			t.Errorf("ValidName(%q) = %v — đây là tên hợp lệ", ten, err)
		}
	}
}

// Lớp phòng thủ thứ hai: dù có chỗ nào quên gọi ValidName, Remove vẫn phải từ
// chối đường dẫn nằm ngoài kho. Lớp này quan trọng vì nó KHÔNG THỂ quên được —
// mọi lối xoá đều đi qua Remove.
func TestRemoveTuChoiDuongDanNgoaiKho(t *testing.T) {
	home := homeGia(t)

	// Dữ liệu thật của người dùng, nằm ngoài kho hồ sơ.
	that := filepath.Join(home, ".claude")
	if err := os.MkdirAll(that, 0o755); err != nil {
		t.Fatal(err)
	}
	moi := filepath.Join(that, "quan-trong.txt")
	if err := os.WriteFile(moi, []byte("DỮ LIỆU THẬT"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Chính xác đường dẫn mà `sagent xoa claude:../../.claude` sinh ra.
	if err := Remove(Dir("claude", filepath.Join("..", "..", ".claude"))); err == nil {
		t.Fatal("Remove chấp nhận đường dẫn ngoài kho — có thể xoá dữ liệu thật")
	}
	if _, err := os.Stat(moi); err != nil {
		t.Fatalf("DỮ LIỆU THẬT BỊ XOÁ: %v", err)
	}
}
