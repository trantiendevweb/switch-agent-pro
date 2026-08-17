package provider

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// claude là adapter cho Claude Code — port từ v1 (tk.ps1 + cfg.py).
type claude struct{}

func init() { Register(claude{}) }

func (claude) Name() string   { return "claude" }
func (claude) EnvVar() string { return "CLAUDE_CONFIG_DIR" }

func (claude) Command() (string, error) {
	if p, err := exec.LookPath("claude"); err == nil {
		return p, nil
	}
	alt := filepath.Join(paths.Home(), ".local", "bin", "claude")
	if _, err := os.Stat(alt); err == nil {
		return alt, nil
	}
	if _, err := os.Stat(alt + ".exe"); err == nil {
		return alt + ".exe", nil
	}
	return "", errors.New("không tìm thấy lệnh claude — cài Claude Code trước")
}

// Đã đo: `claude -p "<prompt>"` chạy không tương tác và in kết quả ra stdout.
func (claude) HeadlessArgs(prompt string) []string { return []string{"-p", prompt} }

func (claude) PrivateFiles() []string { return []string{".credentials.json", ".claude.json"} }

func (claude) BaseDir() string { return filepath.Join(paths.Home(), ".claude") }

// IdentitySource: khi KHÔNG đặt biến, Claude dùng ~/.claude.json, KHÔNG phải
// ~/.claude/.claude.json (bẫy file lạc đã đo ở v1). Ưu tiên file ngoài.
func (claude) IdentitySource() string {
	a := filepath.Join(paths.Home(), ".claude.json")
	if _, err := os.Stat(a); err == nil {
		return a
	}
	return filepath.Join(paths.Home(), ".claude", ".claude.json")
}

func (claude) HasToken(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".credentials.json"))
	return err == nil
}

func (claude) Identity(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, ".claude.json"))
	if err != nil {
		return ""
	}
	var f struct {
		OauthAccount struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"oauthAccount"`
	}
	if json.Unmarshal(data, &f) != nil {
		return ""
	}
	return f.OauthAccount.EmailAddress
}

func (c claude) Verify() []Check {
	var out []Check
	cmd, err := c.Command()
	out = append(out, Check{"tìm thấy lệnh claude", err == nil, cmd})
	_, e2 := os.Stat(c.BaseDir())
	out = append(out, Check{"có thư mục base ~/.claude", e2 == nil, c.BaseDir()})
	_, e3 := os.Stat(c.IdentitySource())
	out = append(out, Check{"đọc được file danh tính gốc", e3 == nil, c.IdentitySource()})
	return out
}

func (claude) SharedKeys() []string { return claudeSharedKeys }

// Danh sách TRẮNG — thứ thuộc về CÁI MÁY và THÓI QUEN LÀM VIỆC. Không đổi sang
// danh sách đen: mai sau Claude thêm khoá gói cước mới, blacklist sẽ lặng lẽ
// để nó lọt sang tài khoản khác.
var claudeSharedKeys = []string{
	"projects", // trust dialog, allowedTools, MCP theo từng project
	"hasCompletedOnboarding",
	"lastOnboardingVersion",
	"remoteDialogSeen",
	"remoteControlSurfacesSeen",
	"claudeAiMcpEverConnected",
	"githubRepoPaths",
	"tipsHistory",
	"tipLifetimeShownCounts",
	"skillUsage",
	"pluginUsage",
	"autoUpdates",
	"installMethod",
	"seenNotifications",
	"announcementImpressions",
	"lastReleaseNotesSeen",
	"hasCompletedClaudeInChromeOnboarding",
	"claudeInChromeDefaultEnabled",
	"cachedChromeExtensionInstalled",
}
