package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultsWhenNoFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if c.Policy.MaxParallelSessions != 4 || c.UI.DefaultSurface != "tui" {
		t.Fatalf("mặc định sai: %+v", c.Policy)
	}
	if len(c.Sources) != 0 {
		t.Fatalf("không có file nào mà Sources = %v", c.Sources)
	}
}

// Tầng dưới (project) phải đè tầng trên (global), và khoá KHÔNG có trong file
// project thì giữ nguyên giá trị của global — đó mới là "đè", không phải "thay".
func TestProjectOverridesGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	write(t, GlobalPath(), "[policy]\nmax_parallel_sessions = 9\n[ui]\ntheme = \"light\"\n")

	proj := t.TempDir()
	write(t, filepath.Join(proj, ProjectDirName, "project.toml"),
		"name = \"app\"\n[policy]\nmax_parallel_sessions = 2\n")

	c, err := Load(proj)
	if err != nil {
		t.Fatal(err)
	}
	if c.Policy.MaxParallelSessions != 2 {
		t.Fatalf("project phải đè global: được %d", c.Policy.MaxParallelSessions)
	}
	if c.UI.Theme != "light" {
		t.Fatalf("khoá không có trong project phải giữ từ global: được %q", c.UI.Theme)
	}
	if c.Name != "app" {
		t.Fatalf("name = %q", c.Name)
	}
	if len(c.Sources) != 2 {
		t.Fatalf("muốn 2 nguồn, được %v", c.Sources)
	}
}

// Tìm ngược lên cây thư mục: chạy ở thư mục con vẫn phải thấy cấu hình ở gốc repo.
func TestFindProjectFileWalksUp(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ProjectDirName, "project.toml"), "name=\"x\"\n")
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindProjectFile(deep); got == "" {
		t.Fatal("không tìm thấy project.toml khi đứng ở thư mục con")
	}
	if got := FindProjectFile(t.TempDir()); got != "" {
		t.Fatalf("không nên tìm thấy gì, được %s", got)
	}
}

// Giá trị sai phải báo lỗi rõ ngay lúc đọc, không để tới lúc chạy mới hỏng.
func TestValidateRejectsBadValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	proj := t.TempDir()
	write(t, filepath.Join(proj, ProjectDirName, "project.toml"),
		"[project]\nworkspace = \"khong-ton-tai\"\n")
	if _, err := Load(proj); err == nil {
		t.Fatal("workspace sai mà không báo lỗi")
	}

	proj2 := t.TempDir()
	write(t, filepath.Join(proj2, ProjectDirName, "project.toml"),
		"[ui]\ndefault_surface = \"hologram\"\n")
	if _, err := Load(proj2); err == nil {
		t.Fatal("default_surface sai mà không báo lỗi")
	}
}

// ---- Hợp đồng [ui] của Pha 5d ----
//
// Bốn khoá mới (theme, columns, pinned_flows, enable_3d) là DỮ LIỆU cho mọi mặt
// đọc chung. Test ở đây canh đúng một việc: cấu hình sai phải kêu NGAY LÚC ĐỌC
// FILE, chứ không để mặt web tự đoán rồi vẽ ra trang trống.

func loadVoiUI(t *testing.T, than string) (Config, error) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()
	write(t, filepath.Join(proj, ProjectDirName, "project.toml"), than)
	return Load(proj)
}

func TestUIMacDinhBat3D(t *testing.T) {
	c, err := loadVoiUI(t, "name = \"x\"\n")
	if err != nil {
		t.Fatal(err)
	}
	// Không khai gì thì mặt ba chiều PHẢI còn đó. Mặc định mà là false thì mọi
	// project cũ lặng lẽ mất một mặt sau khi nâng cấp.
	if !c.UI.Enable3D {
		t.Error("không khai enable_3d thì phải mặc định true")
	}
	if len(c.UI.Columns) != 0 {
		t.Errorf("không khai columns thì phải rỗng (= dùng CotMacDinh), được %v", c.UI.Columns)
	}
}

func TestUINoiKhongVoi3D(t *testing.T) {
	c, err := loadVoiUI(t, "[ui]\nenable_3d = false\n")
	if err != nil {
		t.Fatal(err)
	}
	// Phân biệt "không nói gì" với "nói không": khai false thì phải ra false,
	// không bị mặc định true của tầng trước nuốt mất.
	if c.UI.Enable3D {
		t.Error("khai enable_3d = false mà vẫn ra true — tầng đè nuốt mất lựa chọn")
	}
}

func TestUIThemeLa(t *testing.T) {
	if _, err := loadVoiUI(t, "[ui]\ntheme = \"neon\"\n"); err == nil {
		t.Error("theme lạ phải báo lỗi lúc đọc file")
	}
}

func TestUICotLa(t *testing.T) {
	_, err := loadVoiUI(t, "[ui]\ncolumns = [\"provider\", \"gia_tien\"]\n")
	if err == nil {
		t.Fatal("tên cột lạ phải báo lỗi lúc đọc file")
	}
	// Thông điệp phải NÓI RA tên sai và các tên đúng — báo "cấu hình sai" trơn
	// thì người dùng vẫn phải đi đọc mã nguồn.
	if !strings.Contains(err.Error(), "gia_tien") || !strings.Contains(err.Error(), "danh_tinh") {
		t.Errorf("thông điệp phải nêu tên sai và bộ tên hợp lệ: %v", err)
	}
}

func TestUICotDungThiQua(t *testing.T) {
	c, err := loadVoiUI(t, "[ui]\ncolumns = [\"trang_thai\", \"provider\"]\n")
	if err != nil {
		t.Fatal(err)
	}
	// Giữ ĐÚNG THỨ TỰ người dùng khai, không tự sắp lại theo CotTaiKhoan.
	if len(c.UI.Columns) != 2 || c.UI.Columns[0] != "trang_thai" {
		t.Errorf("phải giữ nguyên thứ tự khai: %v", c.UI.Columns)
	}
}

func TestUIMatMacDinh3DMaTat3D(t *testing.T) {
	_, err := loadVoiUI(t, "[ui]\ndefault_surface = \"3d\"\nenable_3d = false\n")
	if err == nil {
		t.Error("mở mặc định vào mặt đã tắt là mâu thuẫn, phải báo lỗi")
	}
}

func TestCotMacDinhNamTrongCotTaiKhoan(t *testing.T) {
	// Hai danh sách phải không được trôi khỏi nhau: mặc định mà chứa tên không
	// vẽ được thì bảng phiên rỗng ngay cả khi người dùng không khai gì.
	for _, c := range CotMacDinh {
		if !CotHopLe(c) {
			t.Errorf("CotMacDinh có %q không nằm trong CotTaiKhoan", c)
		}
	}
}
