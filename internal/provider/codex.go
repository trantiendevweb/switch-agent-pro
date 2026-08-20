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

// ArgsTuDuyetQuyen: `--approve-for-me` — ĐÃ CHẠY THẬT 21/08/2026, và đây là nấc
// HẸP NHẤT làm được việc, giống cách Cursor cố ý chọn `--trust` thay vì `--yolo`.
//
// Ba nấc đã thử, theo đúng thứ tự hẹp → rộng:
//
//  1. `--sandbox workspace-write` một mình: KHÔNG ĐỦ. Agent đọc được file nhưng
//     ghi thì bị chặn — nguyên văn: "patch rejected: writing is blocked by
//     read-only sandbox; rejected by user approval settings". Và `codex exec`
//     KHÔNG có cờ `--ask-for-approval` (chỉ chế độ tương tác mới có), nên không
//     có cách nào nâng nấc approval từ dòng lệnh.
//  2. `--approve-for-me`: ĐỦ. Help nói nó "route approval requests through
//     automatic review using the workspace-write sandbox" — tự duyệt NHƯNG VẪN
//     TRONG SANDBOX. Chạy thật: agent tạo được file trong worktree, 9.402 token.
//  3. `--dangerously-bypass-approvals-and-sandbox`: không cần tới.
//
// VÌ SAO ĐÁNG ĐỔI: bản cũ khai nấc 3 cho MỌI lượt Codex. Cờ đó bỏ CẢ approval
// LẪN sandbox — `adapter.go` nói thẳng cái giá: "agent duyệt cả xoá file và chạy
// lệnh tuỳ ý trong worktree của repo thật". Nấc 2 giữ lại sandbox, tức vẫn giới
// hạn được vùng ghi. Rộng hơn cần thiết trên mọi lượt là một khoản nợ an ninh
// không ai đòi, cho tới ngày có người đòi.
func (codex) ArgsTuDuyetQuyen() ([]string, bool) {
	return []string{"--approve-for-me"}, true
}

// ArgsThuMuc: `-C, --cd <DIR>` — ĐÃ CHẠY THẬT 21/08/2026. Chạy `codex exec -C
// <thư mục tạm>` rồi bảo agent đọc một file chỉ có ở đó: nó đọc đúng, in ra đúng
// nội dung. Trước đó dòng này chỉ đo bằng `--help`.
//
// Con số để so là của Antigravity: KHÔNG có cờ thì 1/3 lượt đúng, hai lượt kia
// báo "chưa có repository nào được mở". Cờ này mà hụt thì mất 2/3 số lượt — và
// mất theo kiểu agent trả lời trôi chảy về một repo KHÁC, không phải báo lỗi.
func (codex) ArgsThuMuc(dir string) []string { return []string{"--cd", dir} }

func (codex) ArgsHoSo(string) []string { return nil }

// DocKetQua: CHUA DO cach doc du lieu co cau truc cua provider nay.
// ModelArgs: CHUA DO cach chon model tu dong lenh cho provider nay.
// nil = chua biet, KHONG phai "khong co model" — ben goi se canh bao thay vi
// im lang bo qua lua chon cua nguoi dung.
func (codex) ModelArgs(string) []string { return nil }

func (codex) DocKetQua(string) (KetQua, bool) { return KetQua{}, false }

// NangLuc — bảng khai báo cho Codex. Hai dòng cờ quyền và cờ thư mục ĐÃ CHẠY
// THẬT ngày 21/08/2026; trước đó chúng chỉ được đo bằng `--help`, và sổ nợ đo
// lường xếp chúng vào mức ĐỎ đúng vì lý do đó — xem docs/DO-LUONG.md.
func (codex) NangLuc() []NangLuc {
	return []NangLuc{
		Duoc(NLHeadless, "`codex exec \"<prompt>\"` = \"Run Codex non-interactively\" (0.147.0) — "+
			"lệnh con, KHÁC hẳn cờ -p của Claude"),
		Chua(NLChonModel, "CHƯA ĐO cách chọn model từ dòng lệnh; nil ở ModelArgs nghĩa là chưa "+
			"biết, không phải \"không có model\""),
		Duoc(NLTuDuyetQuyen, "`--approve-for-me` (đo 21/08, CHẠY THẬT): tự duyệt nhưng VẪN "+
			"trong sandbox workspace-write. `--sandbox workspace-write` một mình KHÔNG đủ — "+
			"\"writing is blocked by read-only sandbox; rejected by user approval settings\", "+
			"và `codex exec` không có --ask-for-approval. Không cần tới cờ dangerously-bypass"),
		Duoc(NLThuMuc, "`-C, --cd <DIR>` (đo 21/08, CHẠY THẬT): chạy trong thư mục tạm rồi bảo "+
			"agent đọc một file chỉ có ở đó — nó đọc đúng"),
		Chua(NLCoTuHoSo, "CHƯA ĐO: chưa gặp thiết lập nào trong ~/.codex phải chuyển thành cờ"),
		Chua(NLKetQuaCoCauTruc, "CHƯA ĐO cách đọc dữ liệu có cấu trúc; phiên Codex chết vì lý do "+
			"gì thì sổ để nguyên `lost` chứ không đoán"),
		Duoc(NLTachTaiKhoan, "đặt CODEX_HOME vào thư mục rỗng thì `codex login status` trả "+
			"\"Not logged in\" dù ~/.codex thật đang đăng nhập"),
		Duoc(NLHanToken, "claim `exp` trong JWT access_token của auth.json; đo 2026-08-17: "+
			"cửa sổ ~6,5 ngày"),
		Duoc(NLDanhTinh, "email lấy từ phần claim của id_token (chỉ giải base64, không gọi "+
			"mạng); đăng nhập bằng API key thì hiện \"(API key)\""),
	}
}
