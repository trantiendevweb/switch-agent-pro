package provider

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func init() { Register(antigravity{}) }

// antigravity bọc `agy` (Antigravity CLI của Google) — bản thay thế cho Gemini
// CLI sau khi Google cắt gói "Gemini Code Assist for individuals" khỏi client cũ.
//
// KHÁC BA PROVIDER KIA Ở ĐIỂM QUAN TRỌNG NHẤT: token KHÔNG nằm trong thư mục
// config mà nằm trong Windows Credential Manager, dưới một khoá TÊN CỐ ĐỊNH
// (`gemini:antigravity`). Nghĩa là tách thư mục không tách được danh tính.
//
// Đo được, không suy từ tài liệu:
//
//	đăng nhập xong    -> Credential Manager 5 -> 6 mục, mục mới `gemini:antigravity`
//	chạy ở HOME thật  -> OK
//	chạy ở HOME GIẢ (đổi cả USERPROFILE + APPDATA + LOCALAPPDATA) -> VẪN OK
//
// Vế cuối là bằng chứng dứt điểm: danh tính đọc từ kho của Windows bất kể môi
// trường. Vì vậy TachDuocTaiKhoan() trả false, và `fleet` sẽ từ chối chạy nhiều
// bản cho provider này thay vì để hai phiên giành nhau một danh tính.
type antigravity struct{}

func (antigravity) Name() string { return "antigravity" }

// Thư mục làm việc (hội thoại, cache, cấu hình) VẪN tách được bằng USERPROFILE —
// đã đo: chạy ở HOME giả thì nó dựng ~/.gemini/antigravity-cli ở đó. Chỉ có
// token là không tách được.
func (antigravity) EnvVar() string { return "USERPROFILE" }

func (antigravity) Command() (string, error) {
	if p, err := exec.LookPath("agy"); err == nil {
		return p, nil
	}
	p := filepath.Join(os.Getenv("LOCALAPPDATA"), "agy", "bin", "agy.exe")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", errors.New("không tìm thấy lệnh agy — cài theo https://antigravity.google/docs/cli/install")
}

// Đã đo: `agy -p "<prompt>"` chạy không tương tác và in kết quả ra stdout.
// Có thêm `--output-format json` trả về cả thống kê token, nhưng lõi hiện chỉ
// cần văn bản nên giữ mặc định.
func (antigravity) HeadlessArgs(prompt string) []string { return []string{"-p", prompt} }

// Không có file riêng nào để chép: token nằm ở Credential Manager. Trả rỗng là
// mô tả ĐÚNG sự thật, chứ không phải thiếu sót.
func (antigravity) PrivateFiles() []string { return nil }
func (antigravity) SharedKeys() []string   { return nil }

func (antigravity) BaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gemini", "antigravity-cli")
}
func (antigravity) IdentitySource() string { return "" }

func (a antigravity) Version() (string, error) {
	p, err := a.Command()
	if err != nil {
		return "", err
	}
	return hoiVersion(p, "--version")
}

// credTarget là khoá cố định `agy` dùng trong Credential Manager (đo bằng cách
// so danh sách trước/sau khi đăng nhập).
const credTarget = "gemini:antigravity"

// HasToken bỏ qua configDir — token không nằm ở đó. Tham số giữ lại vì interface
// dùng chung; bỏ qua nó là điều ĐÚNG với provider này.
func (antigravity) HasToken(string) bool { return coCredential(credTarget) }

// Identity: CHƯA ĐỌC ĐƯỢC. Sau khi đăng nhập bằng `agy`, không file nào trong
// ~/.gemini bị cập nhật email (google_accounts.json vẫn mang dấu thời gian của
// lần đăng nhập Gemini CLI cũ). Trả rỗng thay vì đoán — hiện nhầm email còn tệ
// hơn không hiện gì.
func (antigravity) Identity(string) string { return "" }

// TokenExpiry: CHƯA ĐO. Token nằm trong Credential Manager, và đọc nội dung nó
// nghĩa là chạm vào chính thứ cần bảo vệ chỉ để lấy một dấu thời gian. Chưa đủ
// lý do.
func (antigravity) TokenExpiry(string) (time.Time, bool) { return time.Time{}, false }

func (a antigravity) Verify() []Check {
	var out []Check
	p, err := a.Command()
	c := Check{Name: "tìm thấy lệnh agy", OK: err == nil, Detail: p}
	if err != nil {
		c.Detail = "chưa cài — xem https://antigravity.google/docs/cli/install"
	}
	out = append(out, c)

	tok := coCredential(credTarget)
	tc := Check{Name: "token trong Credential Manager", OK: tok,
		Detail: "khoá " + credTarget}
	if !tok {
		tc.Detail = "chưa đăng nhập — chạy: agy"
	}
	out = append(out, tc)

	// Nói thẳng giới hạn ngay trong bộ đo, chứ không giấu ở tài liệu.
	out = append(out, Check{
		Name: "tách được nhiều tài khoản", OK: false,
		Detail: "KHÔNG — token nằm ở Credential Manager theo khoá cố định, " +
			"không theo thư mục config. Mỗi máy một tài khoản Antigravity.",
	})
	return out
}

// KHÔNG tách được. Đã đo: chạy trong HOME giả (đổi cả USERPROFILE + APPDATA +
// LOCALAPPDATA) vẫn dùng đúng danh tính đã đăng nhập, vì token đọc từ Windows
// Credential Manager theo khoá cố định `gemini:antigravity`.
func (antigravity) TachDuocTaiKhoan() bool { return false }

// ArgsTuDuyetQuyen: đo `agy --help` + chạy thật (lần chạy #10, #11): agent đọc được repo, trả đúng "Go"
func (antigravity) ArgsTuDuyetQuyen() ([]string, bool) { return []string{"--dangerously-skip-permissions"}, true }

// ArgsThuMuc: đo `agy --help`: "--add-dir  Add a directory to the workspace". Chạy thật ở
// worktree: không có cờ 1/3 đúng, có cờ 4/4 đúng.
func (antigravity) ArgsThuMuc(dir string) []string { return []string{"--add-dir", dir} }
