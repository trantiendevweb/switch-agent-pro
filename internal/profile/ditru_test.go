package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
