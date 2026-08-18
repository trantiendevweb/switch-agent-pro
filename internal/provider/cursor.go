package provider

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func init() { Register(cursor{}) }

// cursor bọc `cursor-agent` (Cursor CLI).
//
// Mọi con số dưới đây đo trên máy thật, bản 2026.08.11-e8db854 — không suy từ
// tài liệu. Xem docs/DO-LUONG.md.
type cursor struct{}

func (cursor) Name() string { return "cursor" }

// EnvVar là APPDATA, KHÔNG phải USERPROFILE.
//
// Đây là kết quả đo, và nó ngược với dự đoán ban đầu. Đổi riêng USERPROFILE thì
// `cursor-agent status` VẪN báo đã đăng nhập — suýt kết luận nhầm rằng Cursor
// không tách được. Đo từng biến một mới ra:
//
//	chỉ USERPROFILE   -> vẫn đăng nhập
//	chỉ LOCALAPPDATA  -> vẫn đăng nhập
//	chỉ HOME          -> vẫn đăng nhập
//	chỉ APPDATA       -> "Not logged in"   <- đây
//
// Tin tốt: APPDATA hẹp hơn USERPROFILE nhiều. Đổi nó không kéo theo git config,
// ssh key hay npm cache của tiến trình con.
func (cursor) EnvVar() string { return "APPDATA" }

func (cursor) Command() (string, error) {
	if p, err := exec.LookPath("cursor-agent"); err == nil {
		return p, nil
	}
	// Trình cài chính thức đặt ở đây và thêm vào PATH NGƯỜI DÙNG — tiến trình con
	// không phải lúc nào cũng thấy PATH đó.
	for _, n := range []string{"cursor-agent.cmd", "cursor-agent.exe", "cursor-agent"} {
		p := filepath.Join(os.Getenv("LOCALAPPDATA"), "cursor-agent", n)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("không tìm thấy lệnh cursor-agent — cài bằng: irm 'https://cursor.com/install?win32=true' | iex")
}

// HeadlessArgs: đã đo `-p` in kết quả ra stdout.
//
// `--trust` là bắt buộc, và là cờ HẸP NHẤT làm được việc: không có nó thì CLI
// dừng lại đòi người dùng xác nhận thư mục. Cursor còn gợi ý `--yolo`/`-f`,
// nhưng hai cái đó nghĩa là "chạy mọi lệnh không hỏi" — cố ý KHÔNG dùng. Một
// agent chạy nền với quyền chạy mọi thứ là chuyện khác hẳn.
func (cursor) HeadlessArgs(prompt string) []string {
	return []string{"--trust", "-p", prompt}
}

// PrivateFiles: đã đo — chép ĐÚNG file này sang một APPDATA giả là danh tính đi
// theo, `status` báo đúng email. Không cần gì khác.
func (cursor) PrivateFiles() []string { return []string{filepath.Join("Cursor", "auth.json")} }

// Cursor không có file config gộp kiểu .claude.json nên không có whitelist khoá.
func (cursor) SharedKeys() []string { return nil }

func (cursor) BaseDir() string        { return filepath.Join(os.Getenv("APPDATA"), "Cursor") }
func (cursor) IdentitySource() string { return "" }

func (c cursor) Version() (string, error) {
	p, err := c.Command()
	if err != nil {
		return "", err
	}
	return hoiVersion(p, "--version")
}

// authFile là đường dẫn token bên trong một thư mục hồ sơ.
func authFile(configDir string) string { return filepath.Join(configDir, "Cursor", "auth.json") }

func (cursor) HasToken(configDir string) bool {
	fi, err := os.Stat(authFile(configDir))
	return err == nil && fi.Size() > 0
}

// Identity đọc email từ auth.json.
//
// CHỈ đọc trường định danh, không bao giờ trả về hay ghi log phần token.
func (cursor) Identity(configDir string) string {
	b, err := os.ReadFile(authFile(configDir))
	if err != nil {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	for _, k := range []string{"email", "userEmail", "user_email"} {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// TokenExpiry: CHƯA ĐO. auth.json có thể mang dấu thời gian hết hạn, nhưng chưa
// dựng được cảnh token sắp hết hạn để xác nhận đọc đúng trường nào. Trả false
// thay vì đoán — cảnh báo sai giờ còn tệ hơn không cảnh báo.
func (cursor) TokenExpiry(configDir string) (time.Time, bool) { return time.Time{}, false }

func (c cursor) Verify() []Check {
	var out []Check
	p, err := c.Command()
	ct := Check{Name: "tìm thấy lệnh cursor-agent", OK: err == nil, Detail: p}
	if err != nil {
		ct.Detail = "chưa cài — irm 'https://cursor.com/install?win32=true' | iex"
	}
	out = append(out, ct)

	base := c.BaseDir()
	_, e := os.Stat(base)
	out = append(out, Check{Name: `có thư mục base %APPDATA%\Cursor`, OK: e == nil, Detail: base})

	tokOK := c.HasToken(os.Getenv("APPDATA"))
	tk := Check{Name: `token nằm ở file Cursor\auth.json`, OK: tokOK,
		Detail: authFile(os.Getenv("APPDATA"))}
	if !tokOK {
		tk.Detail = "chưa đăng nhập — chạy: cursor-agent login"
	}
	out = append(out, tk)
	return out
}

// Token là file Cursor\auth.json trong thư mục APPDATA — đã đo: chép file đó
// sang APPDATA giả là danh tính đi theo, hồ sơ mới thì "Not logged in".
func (cursor) TachDuocTaiKhoan() bool { return true }

// ArgsTuDuyetQuyen: CHƯA ĐO: máy này không cài cursor-agent nên không chạy `--help` được
func (cursor) ArgsTuDuyetQuyen() ([]string, bool) { return nil, false }

// ArgsThuMuc: CHƯA ĐO: máy này không cài cursor-agent
func (cursor) ArgsThuMuc(dir string) []string { return nil }

func (cursor) ArgsHoSo(string) []string { return nil }

// DocKetQua: CHUA DO cach doc du lieu co cau truc cua provider nay.
func (cursor) DocKetQua(string) (KetQua, bool) { return KetQua{}, false }
