package profile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// Kho v1 (~/.claude-accounts) vẫn dùng được nguyên trạng — đây là lời hứa của
// bản v2 với người đang dùng tk v1, nên phải có test chứ không chỉ có câu nói.
func TestHoSoV1VanDungDuoc(t *testing.T) {
	homeGia(t)
	v1 := filepath.Join(paths.LegacyClaudeAccounts(), "phu")
	if err := os.MkdirAll(v1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v1, ".credentials.json"), []byte(`{"t":"TOKEN-V1"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	d, ok := ResolveDir("claude", "phu")
	if !ok {
		t.Fatal("không tìm ra hồ sơ v1 — người dùng tk v1 sẽ mất hết tài khoản khi nâng cấp")
	}
	if d != v1 {
		t.Fatalf("ResolveDir trỏ sai chỗ: %s, muốn %s", d, v1)
	}
}

// `them` một cái tên ĐÃ CÓ ở kho v1 phải bị TỪ CHỐI.
//
// Đo trước khi vá: Create chỉ kiểm ~/.ai-accounts/claude/phu (chưa có) nên nó
// tạo luôn, và sau đó ResolveDir ưu tiên v2 — hồ sơ v1 cùng tên bị đè bóng, token
// vẫn nằm trên đĩa nhưng không còn được dùng. Người dùng bị hỏi đăng nhập lại và
// không hiểu vì sao token cũ "biến mất".
func TestThemTrungTenVoiHoSoV1ThiTuChoi(t *testing.T) {
	homeGia(t)
	v1 := filepath.Join(paths.LegacyClaudeAccounts(), "phu")
	if err := os.MkdirAll(v1, 0o755); err != nil {
		t.Fatal(err)
	}
	tok := filepath.Join(v1, ".credentials.json")
	if err := os.WriteFile(tok, []byte(`{"t":"TOKEN-V1"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	ad, ok := provider.Get("claude")
	if !ok {
		t.Fatal("không lấy được adapter claude")
	}
	_, _, err := Create(ad, "phu")
	if err == nil {
		t.Fatal("Create chấp nhận tên đã có ở kho v1 — hồ sơ cũ bị đè bóng trong im lặng")
	}
	// Thông điệp phải chỉ ra hồ sơ cũ NẰM Ở ĐÂU, nếu không người dùng sẽ tưởng
	// mình gõ nhầm tên.
	if !strings.Contains(err.Error(), ".claude-accounts") {
		t.Errorf("lỗi không chỉ ra vị trí hồ sơ cũ: %v", err)
	}

	// Và quan trọng nhất: hồ sơ v1 vẫn là cái được dùng.
	d, ok := ResolveDir("claude", "phu")
	if !ok || d != v1 {
		t.Fatalf("sau khi bị từ chối, ResolveDir trỏ %q (ok=%v) — muốn %q", d, ok, v1)
	}
	if _, err := os.Stat(tok); err != nil {
		t.Fatalf("token v1 bị động: %v", err)
	}
}

// Chạy TÀI KHOẢN GỐC không được đụng vào môi trường.
//
// Bản trước xoá biến của adapter kể cả khi chạy gốc. Với CLAUDE_CONFIG_DIR /
// CODEX_HOME thì vô hại — xoá đi là CLI tự về mặc định. Nhưng biến của Cursor là
// APPDATA và của Antigravity là USERPROFILE: xoá hai cái đó không phải "chạy tài
// khoản gốc" mà là đưa cho CLI một môi trường Windows HỎNG.
//
// Test dựng một adapter giả có EnvVar = APPDATA, chạy một chương trình in ra giá
// trị biến đó, rồi khẳng định nó KHÔNG rỗng.
func TestChayGocKhongDuocXoaBienHeDieuHanh(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("chỉ có nghĩa trên Windows")
	}
	moc := os.Getenv("APPDATA")
	if moc == "" {
		t.Skip("máy này không có APPDATA để đo")
	}

	out := filepath.Join(t.TempDir(), "ra.txt")
	a := adapterMoiTruong{}
	// cmd /c echo %APPDATA% > file
	err := Run(a, "", []string{"/c", "echo %APPDATA%> " + out})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(b))
	if got == "" || got == "%APPDATA%" {
		t.Fatalf("APPDATA bị xoá khi chạy tài khoản gốc (đọc được %q) — "+
			"CLI sẽ chạy trong một môi trường Windows hỏng", got)
	}
	if got != moc {
		t.Errorf("APPDATA = %q, muốn giữ nguyên %q", got, moc)
	}
}

// adapterMoiTruong: EnvVar là một biến của HỆ ĐIỀU HÀNH, giống cursor/antigravity.
type adapterMoiTruong struct{}

func (adapterMoiTruong) Name() string                         { return "moitruong" }
func (adapterMoiTruong) EnvVar() string                       { return "APPDATA" }
func (adapterMoiTruong) Command() (string, error)             { return "cmd.exe", nil }
func (adapterMoiTruong) Version() (string, error)             { return "0", nil }
func (adapterMoiTruong) TachDuocTaiKhoan() bool               { return true }
func (adapterMoiTruong) HeadlessArgs(p string) []string       { return []string{"/c", p} }
func (adapterMoiTruong) PrivateFiles() []string               { return []string{"x.json"} }
func (adapterMoiTruong) SharedKeys() []string                 { return nil }
func (adapterMoiTruong) BaseDir() string                      { return os.TempDir() }
func (adapterMoiTruong) IdentitySource() string               { return "" }
func (adapterMoiTruong) Identity(string) string               { return "" }
func (adapterMoiTruong) HasToken(string) bool                 { return false }
func (adapterMoiTruong) TokenExpiry(string) (time.Time, bool) { return time.Time{}, false }
func (adapterMoiTruong) Verify() []provider.Check             { return nil }

func (adapterMoiTruong) ArgsTuDuyetQuyen() ([]string, bool) { return nil, false }

func (adapterMoiTruong) ArgsThuMuc(string) []string { return nil }

func (adapterMoiTruong) ArgsHoSo(string) []string { return nil }
