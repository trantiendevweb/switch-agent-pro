package provider

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// codex là adapter cho OpenAI Codex CLI.
//
// Mọi con số dưới đây ĐÃ ĐO trên Windows với @openai/codex 0.147.0 — xem
// docs/DO-LUONG.md. Phép đo quyết định: đặt CODEX_HOME vào thư mục rỗng thì
// `codex login status` trả "Not logged in" dù ~/.codex thật đang đăng nhập →
// tách thư mục là tách tài khoản THẬT, không phải trên giấy.
type codex struct{}

func init() { Register(codex{}) }

func (codex) Name() string   { return "codex" }
func (codex) EnvVar() string { return "CODEX_HOME" }

func (codex) Command() (string, error) {
	if p, err := exec.LookPath("codex"); err == nil {
		return p, nil
	}
	// npm global trên Windows không phải lúc nào cũng nằm trong PATH của tiến trình con
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		for _, n := range []string{"codex.cmd", "codex"} {
			p := filepath.Join(appdata, "npm", n)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}
	return "", errors.New("không tìm thấy lệnh codex — cài bằng: npm i -g @openai/codex")
}

// Version: đã đo trên máy dev -> "codex-cli 0.147.0".
func (c codex) Version() (string, error) {
	p, err := c.Command()
	if err != nil {
		return "", err
	}
	return hoiVersion(p, "--version")
}

// Đã đo trên 0.147.0: `codex exec "<prompt>"` = "Run Codex non-interactively".
// KHÁC hẳn Claude (`-p`) — đây chính là lý do phải để việc này cho adapter.
func (codex) HeadlessArgs(prompt string) []string { return []string{"exec", prompt} }

func (codex) BaseDir() string { return filepath.Join(paths.Home(), ".codex") }

// Codex không có file "cấu hình gộp" kiểu .claude.json để gieo whitelist khoá:
// thói quen máy nằm ở config.toml và AGENTS.md, mà hai thứ đó nối link dùng
// chung được nguyên vẹn. Nên không cần seed.
func (codex) IdentitySource() string { return "" }
func (codex) SharedKeys() []string   { return nil }

// PrivateFiles là những thứ KHÔNG nối link sang hồ sơ khác.
//
// Hai nhóm, vì hai lý do khác nhau:
//  1. Danh tính: auth.json (token), installation_id, cap_sid — dùng chung là
//     hai hồ sơ hoá thành một tài khoản.
//  2. Khoá ghi và cơ sở dữ liệu: thư mục khoá, tmp, sandbox và các file SQLite.
//     Nối chung thì hai phiên chạy song song sẽ giành nhau ghi và hỏng dữ liệu —
//     đúng loại lỗi khó tái hiện nhất.
//
// (Thư mục nằm trong danh sách này cũng không bị `clone` chép, vì clone chỉ chép
// được file thường.)
func (codex) PrivateFiles() []string {
	return []string{
		// danh tính
		"auth.json", "installation_id", "cap_sid",
		// khoá ghi / tạm / sandbox
		"thread-writer-locks", "tmp", ".sandbox", ".sandbox_migration",
		// cơ sở dữ liệu cục bộ (kèm file phụ của SQLite)
		"state_5.sqlite", "state_5.sqlite-shm", "state_5.sqlite-wal",
		"goals_1.sqlite", "goals_1.sqlite-shm", "goals_1.sqlite-wal",
		"queue_1.sqlite", "queue_1.sqlite-shm", "queue_1.sqlite-wal",
		"memories_1.sqlite", "memories_1.sqlite-shm", "memories_1.sqlite-wal",
		"logs_2.sqlite", "logs_2.sqlite-shm", "logs_2.sqlite-wal",
	}
}

func (c codex) HasToken(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "auth.json"))
	if err != nil {
		return false
	}
	var a codexAuth
	if json.Unmarshal(data, &a) != nil {
		return false
	}
	return a.Tokens.AccessToken != "" || a.OpenAIAPIKey != ""
}

type codexAuth struct {
	AuthMode     string `json:"auth_mode"`
	OpenAIAPIKey string `json:"OPENAI_API_KEY"`
	Tokens       struct {
		IDToken     string `json:"id_token"`
		AccessToken string `json:"access_token"`
		AccountID   string `json:"account_id"`
	} `json:"tokens"`
}

// Identity đọc email từ phần payload của id_token (JWT).
//
// Chỉ giải mã base64 phần claim — KHÔNG xác thực chữ ký và KHÔNG gọi mạng: ở
// đây ta chỉ cần một cái tên để hiển thị trong bảng, không phải để cấp quyền.
func (c codex) Identity(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "auth.json"))
	if err != nil {
		return ""
	}
	var a codexAuth
	if json.Unmarshal(data, &a) != nil {
		return ""
	}
	if email := emailFromJWT(a.Tokens.IDToken); email != "" {
		return email
	}
	// Đăng nhập bằng API key thì không có JWT — nói rõ kiểu đăng nhập.
	if a.OpenAIAPIKey != "" {
		return "(API key)"
	}
	if a.Tokens.AccountID != "" {
		return "(tài khoản " + short(a.Tokens.AccountID, 8) + ")"
	}
	return ""
}

func emailFromJWT(tok string) string {
	parts := strings.Split(tok, ".")
	if len(parts) < 2 {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(raw, &claims) != nil {
		return ""
	}
	return claims.Email
}

func short(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// TokenExpiry đọc claim `exp` trong JWT access_token.
// Đã đo 2026-08-17: cửa sổ khoảng 6,5 ngày — dài hơn Claude rất nhiều.
func (c codex) TokenExpiry(dir string) (time.Time, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "auth.json"))
	if err != nil {
		return time.Time{}, false
	}
	var a codexAuth
	if json.Unmarshal(data, &a) != nil {
		return time.Time{}, false
	}
	parts := strings.Split(a.Tokens.AccessToken, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(raw, &claims) != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}

func (c codex) Verify() []Check {
	var out []Check
	cmd, err := c.Command()
	out = append(out, Check{"tìm thấy lệnh codex", err == nil, cmd})

	_, e2 := os.Stat(c.BaseDir())
	out = append(out, Check{"có thư mục base ~/.codex", e2 == nil, c.BaseDir()})

	// Đây là phép đo có giá trị nhất: token phải là FILE trong config dir, chứ
	// không nằm ở keyring của hệ điều hành.
	authPath := filepath.Join(c.BaseDir(), "auth.json")
	_, e3 := os.Stat(authPath)
	out = append(out, Check{"token nằm ở file auth.json", e3 == nil, authPath})

	return out
}

// Token là file trong thư mục config, tách bằng CODEX_HOME — đã đo.
func (codex) TachDuocTaiKhoan() bool { return true }
