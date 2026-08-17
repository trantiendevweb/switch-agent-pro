package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// fakeAdapter đủ để test link + clone + remove mà không cần Claude thật.
type fakeAdapter struct {
	base     string
	hasToken bool
}

func (f fakeAdapter) Name() string             { return "fake" }
func (f fakeAdapter) EnvVar() string           { return "FAKE_CONFIG_DIR" }
func (f fakeAdapter) Command() (string, error) { return "echo", nil }
func (f fakeAdapter) HeadlessArgs(p string) []string { return []string{"-p", p} }
func (f fakeAdapter) PrivateFiles() []string   { return []string{".credentials.json", ".claude.json"} }
func (f fakeAdapter) SharedKeys() []string     { return []string{"projects"} }
func (f fakeAdapter) BaseDir() string          { return f.base }
func (f fakeAdapter) IdentitySource() string   { return "" }
func (f fakeAdapter) Identity(string) string   { return "" }
func (f fakeAdapter) HasToken(string) bool     { return f.hasToken }
func (f fakeAdapter) TokenExpiry(string) (time.Time, bool) { return time.Time{}, false }
func (f fakeAdapter) Verify() []provider.Check { return nil }

// Phép đo an toàn quan trọng nhất (DoD Pha 1): thư mục hồ sơ toàn junction/symlink
// trỏ về base; Remove phải gỡ link trước rồi mới xoá, KHÔNG được xuyên link xoá
// dữ liệu thật ở base.
func TestRemoveDoesNotTouchBase(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "base")
	shared := filepath.Join(base, "shared")
	if err := os.MkdirAll(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	bait := filepath.Join(shared, "bait.txt")
	if err := os.WriteFile(bait, []byte("DỮ LIỆU THẬT"), 0o644); err != nil {
		t.Fatal(err)
	}

	prof := filepath.Join(tmp, "profile")
	if err := os.MkdirAll(prof, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := LinkShared(fakeAdapter{base: base}, prof); err != nil {
		t.Fatalf("LinkShared lỗi: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(prof, "shared")); err != nil {
		t.Fatalf("không nối được 'shared' vào hồ sơ: %v", err)
	}

	if err := Remove(prof); err != nil {
		t.Fatalf("Remove lỗi: %v", err)
	}
	if _, err := os.Stat(prof); !os.IsNotExist(err) {
		t.Fatal("hồ sơ chưa bị xoá")
	}
	if _, err := os.Stat(bait); err != nil {
		t.Fatalf("DỮ LIỆU GỐC BỊ MẤT qua link — bait.txt không còn: %v", err)
	}
}
