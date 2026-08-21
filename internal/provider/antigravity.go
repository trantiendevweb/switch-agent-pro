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
// HeadlessArgs bật NDJSON có cấu trúc, cùng lý do như Claude.
func (antigravity) HeadlessArgs(prompt string) []string {
	return []string{"--output-format", "stream-json", "-p", prompt}
}

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

// TokenExpiry: KHÔNG ĐỌC — và đây là một KẾT LUẬN, không phải một khoảng trống.
//
// Token của Antigravity nằm trong Windows Credential Manager dưới khoá
// `gemini:antigravity` (xem NLTachTaiKhoan). Đọc được hạn nghĩa là phải MỞ chính
// bí mật đó ra: CredRead trả về cả blob, không có cách hỏi riêng "khoá này hết
// hạn lúc nào". Đánh đổi là chạm vào đúng thứ cần bảo vệ để đổi lấy MỘT dấu thời
// gian — đánh đổi tồi, nên câu trả lời là không, chứ không phải chưa.
//
// Cùng dạng với grok.go: ở đó API key KHÔNG CÓ hạn đọc được từ file, ở đây hạn
// CÓ nhưng nằm sau một cánh cửa ta cố ý không mở. Cả hai đều là chuyện đã ngã
// ngũ, nên khai `Khong(NLHanToken)` chứ không phải `Chua`.
//
// Cái giá phải nói ra: cảnh báo "token sắp hết hạn" trước khi bung hạm đội
// (internal/api/api.go, nhánh `TokenExpiry(...) ok`) sẽ KHÔNG BAO GIỜ kêu cho
// Antigravity. Sự im lặng đó không có nghĩa là an toàn — nó có nghĩa là không
// biết, và giờ bảng năng lực nói thẳng điều đó thay vì để trống.
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
func (antigravity) ArgsTuDuyetQuyen() ([]string, bool) {
	return []string{"--dangerously-skip-permissions"}, true
}

// ArgsThuMuc: đo `agy --help`: "--add-dir  Add a directory to the workspace". Chạy thật ở
// worktree: không có cờ 1/3 đúng, có cờ 4/4 đúng.
func (antigravity) ArgsThuMuc(dir string) []string { return []string{"--add-dir", dir} }

func (antigravity) ArgsHoSo(string) []string { return nil }

// ModelArgs: `--model <model>` — ĐÃ CHẠY THẬT 21/08/2026, bản CLI 1.1.16.
//
// Bằng chứng mạnh nhất là CLI TỪ CHỐI tên model sai NGAY TỪ DÒNG LỆNH, trước khi
// tốn một token nào: `agy -p ... --model khong-ton-tai-9x` thoát mã 1 với
// "invalid model selection (--model \"khong-ton-tai-9x\" --effort \"\"): model ...
// is not recognized as a known model or custom model in settings" rồi liệt kê 14
// model hợp lệ. Tức cờ ĐƯỢC ĐỌC và được đối chiếu với sổ model thật.
//
// CÒN MỘT NỬA nữa mới đủ: từ chối tên sai chỉ chứng minh cờ được đọc để KIỂM
// TRA, chưa chứng minh nó được dùng để ĐỊNH TUYẾN. Phép đo quyết định: chạy ĐÚNG
// MỘT PROMPT qua ba model, so `input_tokens` trong `--output-format json`:
//
//	gemini-3.7-flash-low      → 13.747
//	claude-opus-4-6-thinking  → 15.764
//	gpt-oss-120b-medium       → 11.174
//
// Cùng prompt, cùng repo, cùng CLI — biến duy nhất là `--model`. Ba con số khác
// nhau nghĩa là ba bộ tách từ khác nhau đã đọc cùng một đầu vào: cờ thật sự đổi model.
//
// PHÉP ĐO ĐÃ BẮT ĐẦU SAI, giữ lại để người sau không lặp: hỏi thẳng agent
// "Google hay Anthropic tạo ra bạn?" với --model claude-opus-4-6-thinking thì nó
// trả "Google". Tự khai danh tính KHÔNG dùng được ở đây — lời nhắc hệ thống của
// Antigravity đè lên câu trả lời. Số token thì không biết nói dối.
//
// THỨ TỰ CỜ: `argsChoBuoc` chèn ModelArgs VÀO TRƯỚC HeadlessArgs, nên dòng thật là
// `agy --model <m> --output-format stream-json -p <prompt>`. Đã chạy đúng dạng đó,
// không phải dạng cờ-đứng-sau: 13.742 token, đúng chữ ký của gemini-3.7-flash-low.
func (antigravity) ModelArgs(model string) []string { return []string{"--model", model} }

func (antigravity) DocKetQua(raw string) (KetQua, bool) { return docKetQuaAntigravity(raw) }

// NangLuc — bảng khai báo cho Antigravity. Dòng đáng đọc nhất là
// tach-nhieu-tai-khoan: đây là provider DUY NHẤT khai KHÔNG LÀM ĐƯỢC, và đó là
// một kết luận đã đo chứ không phải một khoảng trống.
func (antigravity) NangLuc() []NangLuc {
	return []NangLuc{
		Duoc(NLHeadless, "`agy -p \"<prompt>\"` chạy không tương tác; --output-format stream-json "+
			"cho bản ghi NDJSON có cấu trúc"),
		Duoc(NLChonModel, "`--model <model>` (đo 21/08, CHẠY THẬT, bản 1.1.16): tên model sai "+
			"bị TỪ CHỐI ngay từ dòng lệnh (thoát 1, \"model ... is not recognized\" + liệt kê 14 "+
			"model hợp lệ). Cờ CÓ ĐỊNH TUYẾN chứ không chỉ để kiểm tra: cùng một prompt chạy "+
			"ba model cho ba mức input_tokens khác nhau — 13.747 / 15.764 / 11.174"),
		Duoc(NLTuDuyetQuyen, "`agy --help` + chạy thật (lần chạy #10, #11): agent đọc được repo "+
			"và trả đúng \"Go\" với --dangerously-skip-permissions"),
		Duoc(NLThuMuc, "`agy --help`: --add-dir. Chạy thật trong git worktree: không có cờ thì "+
			"1/3 đúng (hai lượt kia báo \"chưa có repository nào được mở\"), có cờ thì 4/4 đúng"),
		Chua(NLCoTuHoSo, "CHƯA ĐO: chưa gặp thiết lập nào trong ~/.gemini phải chuyển thành cờ"),
		Duoc(NLKetQuaCoCauTruc, "docKetQuaAntigravity đọc bản ghi NDJSON của `--output-format "+
			"stream-json`"),
		Khong(NLTachTaiKhoan, "KHÔNG — token nằm trong Windows Credential Manager dưới khoá TÊN "+
			"CỐ ĐỊNH `gemini:antigravity`, không theo thư mục config. Đo: chạy trong HOME giả "+
			"(đổi cả USERPROFILE + APPDATA + LOCALAPPDATA) VẪN dùng đúng danh tính đã đăng nhập. "+
			"Mỗi máy một tài khoản Antigravity"),
		Khong(NLHanToken, "KHÔNG ĐỌC, và đây là kết luận chứ không phải khoảng trống: token "+
			"nằm trong Windows Credential Manager dưới khoá `gemini:antigravity`, CredRead trả "+
			"về cả blob chứ không có cách hỏi riêng mốc hết hạn. Mở chính thứ cần bảo vệ để đổi "+
			"lấy MỘT dấu thời gian là đánh đổi tồi — cùng dạng với grok.go. Hệ quả nói thẳng: "+
			"cảnh báo token-sắp-hết-hạn trước khi bung hạm đội không bao giờ kêu cho provider này"),
		Chua(NLDanhTinh, "CHƯA ĐỌC ĐƯỢC: sau khi đăng nhập bằng `agy`, không file nào trong "+
			"~/.gemini bị cập nhật email (google_accounts.json vẫn mang dấu thời gian của lần "+
			"đăng nhập Gemini CLI cũ)"),
	}
}
