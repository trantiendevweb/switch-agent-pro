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

// Identity đọc email từ NHẬT KÝ của chính `agy` trong thư mục hồ sơ — xem
// danhtinh_antigravity.go để biết vì sao nguồn đó phân biệt được với cái bẫy
// google_accounts.json, và phần nào còn nợ.
//
// Cổng `coCredential` đứng trước là cố ý: đăng xuất thì mục
// `gemini:antigravity` biến mất khỏi Credential Manager, nhưng nhật ký cũ vẫn
// nằm nguyên trên đĩa. Không có cổng này thì một hồ sơ đã đăng xuất vẫn khoe
// email — đúng kiểu "hiện nhầm còn tệ hơn không hiện gì" mà ô V2 dựng ra để
// chặn. Cổng chỉ HỎI mục có tồn tại không, không mở bí mật ra.
func (antigravity) Identity(configDir string) string {
	if !coCredential(credTarget) {
		return ""
	}
	return danhTinhTuNhatKyAgy(configDir)
}

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

// ArgsHoSo: KHÔNG CẦN — đã mở cả cây hồ sơ ra tra, 21/08/2026, chứ không chỉ
// "chưa gặp".
//
// Ba file cấu hình duy nhất tìm được trong một hồ sơ `agy`:
//
//	.gemini/settings.json                    security.auth.selectedType = oauth-personal
//	.gemini/antigravity-cli/settings.json    trustedWorkspaces = [...]
//	.gemini/config/config.json               userSettings.remoteControlHostname
//
// Không cái nào có cờ dòng lệnh tương ứng trong `agy --help`, và cũng không cần:
// `agy` đọc thẳng ba file đó từ HOME được truyền vào. Riêng `trustedWorkspaces`
// trông giống việc của `--add-dir`, nhưng thư mục làm việc thật đã do ArgsThuMuc
// truyền vào từng lượt — đọc lại danh sách cũ trong file là ép agent vào
// workspace của lượt trước.
//
// Đáng ghi: `agy --help` CÓ `--model`, nhưng trong hồ sơ KHÔNG có thiết lập model
// nào để mà chuyển. Nên ô này đóng ở "không có gì để suy", không phải "không có
// cờ để dùng".
func (antigravity) ArgsHoSo(string) []string { return nil }

// ModelArgs: CHUA DO cach chon model tu dong lenh cho provider nay.
// nil = chua biet, KHONG phai "khong co model" — ben goi se canh bao thay vi
// im lang bo qua lua chon cua nguoi dung.
func (antigravity) ModelArgs(string) []string { return nil }

func (antigravity) DocKetQua(raw string) (KetQua, bool) { return docKetQuaAntigravity(raw) }

// NangLuc — bảng khai báo cho Antigravity. Dòng đáng đọc nhất là
// tach-nhieu-tai-khoan: đây là provider DUY NHẤT khai KHÔNG LÀM ĐƯỢC, và đó là
// một kết luận đã đo chứ không phải một khoảng trống.
func (antigravity) NangLuc() []NangLuc {
	return []NangLuc{
		Duoc(NLHeadless, "`agy -p \"<prompt>\"` chạy không tương tác; --output-format stream-json "+
			"cho bản ghi NDJSON có cấu trúc"),
		Chua(NLChonModel, "CHƯA ĐO cách chọn model từ dòng lệnh"),
		Duoc(NLTuDuyetQuyen, "`agy --help` + chạy thật (lần chạy #10, #11): agent đọc được repo "+
			"và trả đúng \"Go\" với --dangerously-skip-permissions"),
		Duoc(NLThuMuc, "`agy --help`: --add-dir. Chạy thật trong git worktree: không có cờ thì "+
			"1/3 đúng (hai lượt kia báo \"chưa có repository nào được mở\"), có cờ thì 4/4 đúng"),
		Khong(NLCoTuHoSo, "KHÔNG CẦN — đã mở cả cây hồ sơ ra tra (đo 21/08): chỉ có ba file "+
			"cấu hình — settings.json (kiểu xác thực), antigravity-cli/settings.json "+
			"(trustedWorkspaces), config/config.json (remoteControlHostname). Không cái nào "+
			"cần chuyển thành cờ, và `agy` tự đọc chúng từ HOME được truyền vào. "+
			"trustedWorkspaces KHÔNG dùng làm --add-dir: thư mục thật do ArgsThuMuc truyền "+
			"từng lượt, đọc lại danh sách cũ là ép agent vào workspace của lượt trước"),
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
		Duoc(NLDanhTinh, "email đọc từ NHẬT KÝ của chính `agy` "+
			"(<hồ sơ>/.gemini/antigravity-cli/log/cli-*.log, dòng `applyAuthResult: email=` / "+
			"`authenticated successfully as`), KHÔNG phải từ google_accounts.json. Đo 21/08: "+
			"google_accounts.json là file của GEMINI CLI, đứng im ở 18/08 10:09 qua cả ba lượt "+
			"`agy` ngày 21/08 — đọc nó là ra email CÓ THẬT nhưng SAI NGƯỜI. Nhật ký phân biệt "+
			"được vì nó vẫn in email trong HAI thư mục KHÔNG HỀ CÓ google_accounts.json: HOME "+
			"giả, và hồ sơ thật ~/.ai-accounts/antigravity/may. Còn nợ: đăng nhập lại bằng tài "+
			"khoản khác ở HOME khác thì hồ sơ này hiện email cũ tới lượt `agy` kế tiếp"),
	}
}
