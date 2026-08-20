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

// ArgsTuDuyetQuyen: `--trust` — ĐÃ CHẠY THẬT 21/08/2026 trên bản 2026.08.11.
//
// `--help` mô tả ba nấc: `--trust` ("Trust the current workspace without
// prompting"), `-f/--force` ("Force allow commands unless explicitly denied") và
// `--yolo` (alias của --force). Đo trong một git repo vứt đi: `--trust` MỘT MÌNH
// đã đủ để agent ghi file trong workspace — không cần tới --force/--yolo.
//
// Giữ nấc hẹp nhất là quyết định có ý thức, giống Codex chọn `--approve-for-me`
// thay vì cờ bỏ cả sandbox: "đủ để làm việc" và "toàn quyền" là hai thứ khác
// nhau, và mặc định phải là cái thứ nhất.
func (cursor) ArgsTuDuyetQuyen() ([]string, bool) { return []string{"--trust"}, true }

// ArgsThuMuc: `cursor-agent` KHÔNG có cờ đổi thư mục — `--help` bản 2026.08.11
// không có `--cwd` lẫn `-C`. Nó làm việc trong thư mục của tiến trình.
//
// Trả nil ở đây là ĐÚNG và đủ: `fleet` đã chạy tiến trình con với `workDir` là
// worktree của phiên (`profile.StartDetached`), nên Cursor vào đúng chỗ mà không
// cần cờ nào. Đây là "đã đo, provider không có thứ đó", KHÁC "chưa ai đo".
func (cursor) ArgsThuMuc(dir string) []string { return nil }

func (cursor) ArgsHoSo(string) []string { return nil }

// ModelArgs: `--model <model>` — có trong `--help` bản 2026.08.11 (help nêu ví
// dụ: gpt-5, sonnet-4-thinking). CHƯA chạy thật để xác nhận nó ĐỔI model đúng,
// nên bảng năng lực vẫn khai ChuaDo — nhưng cờ thì trả về, vì truyền nó là
// chuyện vô hại: sai tên model thì CLI báo lỗi ngay chứ không âm thầm chạy
// model khác.
func (cursor) ModelArgs(model string) []string { return []string{"--model", model} }

// DocKetQua: đọc dòng `{"type":"result"}` của `--output-format stream-json` —
// ĐÃ CHẠY THẬT 21/08/2026.
func (cursor) DocKetQua(raw string) (KetQua, bool) { return docKetQuaCursor(raw) }

// NangLuc — bảng khai báo cho Cursor.
//
// Phần lớn các dòng CHƯA ĐO cũ đã được đóng ngày 21/08/2026: máy dev nay CÓ
// cursor-agent (bản 2026.08.11), nên chạy thật được thay vì suy từ tài liệu.
// Xem docs/DO-LUONG.md.
func (cursor) NangLuc() []NangLuc {
	return []NangLuc{
		Duoc(NLHeadless, "`cursor-agent --trust -p \"<prompt>\"` in kết quả ra stdout; --trust "+
			"là cờ HẸP NHẤT làm được việc, cố ý không dùng --yolo/-f"),
		Duoc(NLChonModel, "`--model <model>` (đo 21/08, CHẠY THẬT): truyền tên không tồn tại "+
			"thì CLI TỪ CHỐI và liệt kê model hợp lệ (\"Cannot use this model: … Available "+
			"models: auto, gpt-5.3-codex, composer-2.5, claude-opus-5-thinking-high, …\") — "+
			"tức cờ được nhận và có hiệu lực, không bị nuốt im lặng"),
		Duoc(NLTuDuyetQuyen, "`--trust` (đo 21/08, CHẠY THẬT trên 2026.08.11): một mình đã đủ "+
			"để agent ghi file trong workspace. Không cần --force/--yolo — cố ý giữ nấc hẹp nhất"),
		Khong(NLThuMuc, "cursor-agent KHÔNG có cờ đổi thư mục: --help (2026.08.11) không có "+
			"--cwd lẫn -C. Không cần: fleet đã chạy tiến trình con với workDir là worktree của phiên"),
		Chua(NLCoTuHoSo, "CHƯA ĐO: chưa gặp thiết lập nào trong Cursor\\auth.json phải chuyển thành cờ"),
		Duoc(NLKetQuaCoCauTruc, "`--output-format stream-json` (đo 21/08, CHẠY THẬT): dòng cuối "+
			"{\"type\":\"result\"} mang is_error, subtype, result, request_id và usage "+
			"(inputTokens/outputTokens — camelCase, KHÁC Claude). Không có total_cost_usd nên "+
			"chi phí vẫn chưa đo được"),
		Duoc(NLTachTaiKhoan, "chép ĐÚNG Cursor\\auth.json sang một APPDATA giả thì danh tính đi "+
			"theo và `status` báo đúng email; hồ sơ mới thì \"Not logged in\". Đo từng biến một: "+
			"chỉ APPDATA mới tách được, USERPROFILE/LOCALAPPDATA/HOME đều không"),
		Chua(NLHanToken, "CHƯA ĐO: auth.json có thể mang dấu thời gian hết hạn, nhưng chưa dựng "+
			"được cảnh token sắp hết hạn để xác nhận đọc đúng trường nào — cảnh báo sai giờ còn "+
			"tệ hơn không cảnh báo"),
		Duoc(NLDanhTinh, "trường email/userEmail/user_email trong Cursor\\auth.json"),
	}
}
