package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/link"
)

// fakeHome trỏ HOME/USERPROFILE vào thư mục tạm để không đụng máy thật, rồi
// dựng sẵn một hồ sơ gốc có token.
func fakeHome(t *testing.T) (home, base string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)        // Linux
	t.Setenv("USERPROFILE", home) // Windows

	// base của adapter giả + nội dung dùng chung
	baseDir := filepath.Join(home, "fakebase")
	if err := os.MkdirAll(filepath.Join(baseDir, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "skills", "x.md"), []byte("chung"), 0o644); err != nil {
		t.Fatal(err)
	}

	// hồ sơ gốc fake:phu (đúng tên của adapter giả)
	base = filepath.Join(home, ".ai-accounts", "fake", "phu")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".credentials.json"), []byte(`{"t":"token"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".claude.json"), []byte(`{"projects":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, baseDir
}

// Điểm sống còn của fleet: mỗi bản clone phải có FILE RIÊNG cho token/danh tính
// (không phải link), nếu không N tiến trình sẽ đua nhau ghi .claude.json.
func TestClonePrivateFilesAreRealCopies(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := fakeAdapter{base: fakeBase, hasToken: true}

	dirs, err := Clone(a, "phu", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 3 {
		t.Fatalf("muốn 3 bản clone, được %d", len(dirs))
	}
	for _, d := range dirs {
		for _, name := range a.PrivateFiles() {
			p := filepath.Join(d, name)
			isLink, _ := link.IsLink(p)
			if isLink {
				t.Fatalf("%s phải là file thật, không được là link", p)
			}
			if _, err := os.Stat(p); err != nil {
				t.Fatalf("thiếu %s: %v", p, err)
			}
		}
		// phần dùng chung thì ngược lại: phải là link
		shared := filepath.Join(d, "skills")
		if isLink, _ := link.IsLink(shared); !isLink {
			t.Fatalf("%s phải là link tới base", shared)
		}
	}

	// Sửa file của bản 1 không được ảnh hưởng bản 2.
	if err := os.WriteFile(filepath.Join(dirs[0], ".claude.json"), []byte(`{"projects":{"a":1}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dirs[1], ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"projects":{}}` {
		t.Fatalf("bản clone 2 bị ảnh hưởng bởi bản 1: %s", b)
	}
}

// Chưa đăng nhập thì phải báo rõ chứ không tạo ra bản clone rỗng vô dụng.
func TestCloneRefusesWithoutToken(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := fakeAdapter{base: fakeBase, hasToken: false}
	if _, err := Clone(a, "phu", 2); err == nil {
		t.Fatal("phải báo lỗi khi hồ sơ chưa có token")
	}
}

// CleanClones dùng đường xoá an toàn: thư mục clone đầy link trỏ về base, xoá
// ẩu là mất dữ liệu thật.
func TestCleanClonesDoesNotTouchBase(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := fakeAdapter{base: fakeBase, hasToken: true}
	if _, err := Clone(a, "phu", 2); err != nil {
		t.Fatal(err)
	}
	bait := filepath.Join(fakeBase, "skills", "x.md")

	n, err := CleanClones("fake", "phu")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("muốn xoá 2 bản, được %d", n)
	}
	if _, err := os.Stat(bait); err != nil {
		t.Fatalf("DỮ LIỆU GỐC BỊ XOÁ QUA LINK: %v", err)
	}
	if _, err := os.Stat(CloneDir("fake", "phu", 1)); !os.IsNotExist(err) {
		t.Fatal("bản clone chưa bị xoá")
	}
}
