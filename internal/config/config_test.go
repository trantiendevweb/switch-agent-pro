package config

import (
	"os"
	"path/filepath"
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
