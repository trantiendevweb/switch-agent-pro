package provider

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// claude là adapter cho Claude Code — port từ v1 (tk.ps1 + cfg.py).
type claude struct{}

func init() { Register(claude{}) }

func (claude) Name() string   { return "claude" }
func (claude) EnvVar() string { return "CLAUDE_CONFIG_DIR" }

func (claude) Command() (string, error) {
	// ƯU TIÊN .exe THẬT hơn vỏ .cmd. Trên máy này `claude` trên PATH là một vỏ
	// batch tự viết, mà vỏ batch CẮT đối số nhiều dòng: đo 18/08, gửi 3 dòng thì
	// chương trình nhận đúng dòng đầu. Prompt của flow luôn nhiều dòng, nên đi
	// qua vỏ là agent nhận một mẩu rồi vẫn trả lời tự tin — hỏng lặng lẽ.
	if p := timClaudeExe(); p != "" {
		return p, nil
	}
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

// Version: đã đo trên máy dev -> "2.1.229 (Claude Code)".
func (c claude) Version() (string, error) {
	p, err := c.Command()
	if err != nil {
		return "", err
	}
	return hoiVersion(p, "--version")
}

// Đã đo: `claude -p "<prompt>"` chạy không tương tác và in kết quả ra stdout.
// HeadlessArgs bật NDJSON có cấu trúc thay vì chữ trơn.
//
// Đo 18/08: dòng cuối `{"type":"result"}` mang is_error, subtype, stop_reason,
// terminal_reason, permission_denials, api_error_status, usage, total_cost_usd —
// đủ để biết lượt chạy hỏng hay không mà KHÔNG cần dò chuỗi tiếng Anh.
// `--verbose` là bắt buộc: thiếu nó thì stream-json chỉ ra mỗi dòng cuối.
func (claude) HeadlessArgs(prompt string) []string {
	return []string{"-p", prompt, "--output-format", "stream-json", "--verbose"}
}

// ModelArgs: đã đo `claude --model <tên>` (bản 2.1.229). Tên nhận cả bí danh
// ngắn (sonnet, opus, haiku) lẫn id đầy đủ.
//
// VÌ SAO ĐÁNG CÓ, có số đo: lượt chạy #34 bước `code-go` tốn 8,18 USD vì mọi
// bước đều chạy model mạnh nhất. Việc viết tài liệu hay việc gộp báo cáo không
// cần tới đó.
func (claude) ModelArgs(model string) []string { return []string{"--model", model} }

func (claude) DocKetQua(raw string) (KetQua, bool) { return docKetQuaClaude(raw) }

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

// HasToken: file tồn tại là CHƯA ĐỦ.
//
// Đo 19/08 19:30 — một lần đăng nhập DỞ DANG vẫn để lại .credentials.json đầy
// đủ trường (accessToken, refreshToken, scopes, subscriptionType "max"), chỉ
// khác đúng một chỗ: `expiresAt: 0`. Hỏi thẳng CLI trên đúng hồ sơ đó:
//
//	claude auth status  ->  {"loggedIn": false, "authMethod": "none"}
//
// Trong khi tài khoản gốc đang chạy được có expiresAt = 1787156212029 (một mốc
// thật). Nói cách khác `expiresAt: 0` KHÔNG phải "không ghi hạn", nó là dấu vết
// của một token không dùng được.
//
// Bản trước chỉ kiểm file có tồn tại, nên `sagent ds` in "sẵn sàng" và cổng
// kiểm tài khoản cho lượt chạy #31 đi qua — rồi bước code-go chết ngay với
// "OAuth session expired and could not be refreshed". Đúng cái bệnh mà
// Profile.HetHan sinh ra để chữa, chỉ là lọt qua bằng một cửa khác.
func (claude) HasToken(dir string) bool {
	ms, err := hanTokenClaude(dir)
	return err == nil && ms != 0
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

// TokenExpiry đọc claudeAiOauth.expiresAt (mili-giây từ epoch).
// Đã đo 2026-08-17: cửa sổ khoảng 7,5 giờ.
func (claude) TokenExpiry(dir string) (time.Time, bool) {
	ms, err := hanTokenClaude(dir)
	if err != nil || ms == 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(ms), true
}

// hanTokenClaude đọc claudeAiOauth.expiresAt (mili-giây). 0 = không có mốc.
func hanTokenClaude(dir string) (int64, error) {
	data, err := os.ReadFile(filepath.Join(dir, ".credentials.json"))
	if err != nil {
		return 0, err
	}
	var f struct {
		ClaudeAiOauth struct {
			ExpiresAt int64 `json:"expiresAt"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return 0, err
	}
	return f.ClaudeAiOauth.ExpiresAt, nil
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

// Token là file trong thư mục config, tách bằng CLAUDE_CONFIG_DIR — đã đo.
func (claude) TachDuocTaiKhoan() bool { return true }

// ArgsTuDuyetQuyen: đo `claude --help`: "--dangerously-skip-permissions  Bypass all permission checks."
func (claude) ArgsTuDuyetQuyen() ([]string, bool) {
	return []string{"--dangerously-skip-permissions"}, true
}

// ArgsThuMuc: đo `claude --help`: "--add-dir <directories...>  Additional directories to allow tool"
func (claude) ArgsThuMuc(dir string) []string { return []string{"--add-dir", dir} }

func (claude) ArgsHoSo(string) []string { return nil }

// timClaudeExe dò bản Claude Code đóng gói. Hai chỗ, vì gói Microsoft Store
// (MSIX) đổi hướng ghi sang LocalCache riêng chứ không nằm ở %APPDATA%.
// Trong mỗi chỗ, mỗi phiên bản một thư mục con — lấy bản MỚI NHẤT theo tên.
func timClaudeExe() string {
	var goc []string
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		goc = append(goc, filepath.Join(la, "Packages", "Claude_pzs8sxrjxfjjc",
			"LocalCache", "Roaming", "Claude", "claude-code"))
	}
	if ad := os.Getenv("APPDATA"); ad != "" {
		goc = append(goc, filepath.Join(ad, "Claude", "claude-code"))
	}
	for _, g := range goc {
		ents, err := os.ReadDir(g)
		if err != nil {
			continue
		}
		names := make([]string, 0, len(ents))
		for _, e := range ents {
			if e.IsDir() {
				names = append(names, e.Name())
			}
		}
		sort.Sort(sort.Reverse(sort.StringSlice(names)))
		for _, n := range names {
			p := filepath.Join(g, n, "claude.exe")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}
