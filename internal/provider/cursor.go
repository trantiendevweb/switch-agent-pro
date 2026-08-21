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

// Identity đọc email từ auth.json — nhưng ĐO 21/08/2026 thì KHÔNG CÓ email ở đó.
//
// File auth.json thật có ĐÚNG hai khoá, `accessToken` và `refreshToken` (xem
// TokenExpiry). Không `email`, không `userEmail`, không `user_email`. Trong
// payload JWT cũng không: `sub` là một mã đục dạng `google-oauth|<id>`, không
// phải địa chỉ thư. Nên trên hồ sơ ĐANG ĐĂNG NHẬP THẬT, hàm này trả `""`.
//
// Vòng lặp khoá dưới đây GIỮ LẠI có chủ đích: nó không tốn gì, và ngày một bản
// CLI mới ghi thêm trường email thì nó đọc được ngay. Nhưng bảng năng lực đã
// hạ NLDanhTinh xuống ChuaDo — khai xanh cho một hàm luôn trả rỗng là đúng thứ
// mà `KiemNangLuc` sinh ra để chặn, chỉ tiếc là phép dò của NLDanhTinh một
// chiều nên nó không bắt được ca này.
//
// Đường còn lại chưa đo: `cursor-agent status` IN ĐƯỢC email (đã thấy khi đo
// NLTachTaiKhoan). Đọc danh tính bằng cách chạy CLI con là một đánh đổi khác
// hẳn đọc file — chưa đo nên chưa làm.
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

// TokenExpiry đọc claim `exp` trong JWT của `Cursor\auth.json` — ĐÃ ĐO
// 21/08/2026 trên hồ sơ ĐĂNG NHẬP THẬT, bản CLI 2026.08.11-e8db854.
//
// Rào chắn cũ ("auth.json CÓ THỂ mang dấu thời gian, chưa biết trường nào") đứng
// trên một chỗ tìm sai: file KHÔNG nằm ở `~/.cursor/auth.json` — chỗ đó không hề
// tồn tại — mà ở `%APPDATA%\Cursor\auth.json`, đúng chỗ EnvVar và PrivateFiles
// đã chỉ từ đầu.
//
// HÌNH DẠNG FILE, đo chứ không đoán: auth.json có ĐÚNG HAI khoá, `accessToken`
// và `refreshToken`, cả hai là JWT alg HS256. KHÔNG có trường dấu-thời-gian nào
// ở tầng ngoài — mốc hết hạn nằm TRONG payload của JWT, y hệt Codex. Nên "đoán
// tên trường" là ngõ cụt ngay từ đầu: không có trường nào để đoán.
//
// Claim đọc được (accessToken và refreshToken giống hệt nhau từng claim một):
//
//	iss    https://authentication.cursor.sh
//	aud    https://cursor.com
//	scope  openid profile email offline_access
//	type   session
//	time   1787022539  = 2026-08-18T03:08:59Z   (lúc đăng nhập)
//	exp    1792206539  = 2026-10-17T03:08:59Z   (ĐÚNG 60 ngày sau)
//
// Vì sao đọc REFRESH token chứ không phải access token: đúng bài học đã trả giá
// ở claude.go — câu hỏi của người vận hành là "tài khoản này còn chạy được
// không", và câu đó nằm ở refresh token. Ở Cursor hôm nay hai mốc BẰNG NHAU nên
// chọn cái nào cũng ra một số; chọn refresh để ngày nhà cung cấp tách hai mốc
// ra thì hàm này vẫn trả đúng thứ cần trả mà không phải sửa lại.
//
// CÒN CHƯA ĐO: token có bị XOAY VÒNG khi refresh hay không (đây mới là thứ cảnh
// báo hạm đội thật sự sợ). Bằng chứng gián tiếp là CLI chưa hề ghi lại file này:
// mtime của auth.json vẫn là 2026-08-18 10:08:58 (+07) — đúng giây đăng nhập —
// trong khi `.cursor/cli-config.json` bị ghi lại lúc 2026-08-21 00:46 bởi chính
// các lượt chạy thật của ô Đ3. Tức qua ba ngày dùng, cursor-agent không đụng vào
// file token.
//
// Hệ quả với cảnh báo hạm đội: cửa sổ 60 ngày dài hơn mọi lượt chạy, nên nhánh
// "còn dưới 2 tiếng" ở internal/api/api.go gần như sẽ không kêu cho Cursor. Đó
// là ĐÚNG, không phải hỏng — cái thay đổi thật là nó thôi im lặng vì KHÔNG BIẾT
// và bắt đầu im lặng vì ĐÃ BIẾT là còn hạn. Ngày token thật sự hết, `sagent ds`
// nói được "hết hạn lúc mấy giờ" thay vì để trống.
func (cursor) TokenExpiry(configDir string) (time.Time, bool) {
	b, err := os.ReadFile(authFile(configDir))
	if err != nil {
		return time.Time{}, false
	}
	var t struct {
		Access  string `json:"accessToken"`
		Refresh string `json:"refreshToken"`
	}
	if json.Unmarshal(b, &t) != nil {
		return time.Time{}, false
	}
	if exp, ok := hanTuJWT(t.Refresh); ok {
		return exp, true
	}
	// Không có refresh: mốc duy nhất biết được là hạn access token.
	return hanTuJWT(t.Access)
}

// hanTuJWT đọc claim `exp` (giây epoch) trong payload của một JWT.
//
// CỐ Ý KHÔNG kiểm chữ ký: chữ ký là HS256 bằng khoá của nhà cung cấp — ta không
// có khoá, và cũng không cần. Ta không XÁC THỰC token, chỉ hỏi nó tự khai hết
// hạn lúc nào; token nằm sẵn trong hồ sơ của chính người dùng nên "token giả"
// không phải mối đe doạ ở đây. Chữ ký hỏng thì lời gọi API hỏng, không phải
// việc của hàm này.
//
// Trả false thay vì một mốc bịa ra ở mọi ngõ hỏng — cảnh báo sai giờ còn tệ hơn
// không cảnh báo.
func hanTuJWT(tok string) (time.Time, bool) {
	parts := strings.Split(tok, ".")
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

// ModelArgs: `--model <model>` — ĐÃ CHẠY THẬT 21/08/2026 trên bản 2026.08.11.
//
// Truyền một tên model không tồn tại thì CLI TỪ CHỐI dòng lệnh và liệt kê model
// hợp lệ ("Cannot use this model: … Available models: auto, gpt-5.3-codex,
// composer-2.5, claude-opus-5-thinking-high, …"). Tức cờ ĐƯỢC NHẬN và CÓ HIỆU
// LỰC, không bị nuốt im lặng — nên bảng năng lực khai `Duoc(NLChonModel)`.
// Xem docs/DO-LUONG.md, mục 21/08 "Đ3: Cursor".
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
		Duoc(NLHanToken, "claim `exp` trong JWT của %APPDATA%\\Cursor\\auth.json (đo 21/08 trên "+
			"hồ sơ đăng nhập thật). File có ĐÚNG hai khoá accessToken/refreshToken, không có "+
			"trường dấu-thời-gian tầng ngoài — mốc nằm trong payload JWT như Codex. Hai token "+
			"cùng exp=1792206539 (2026-10-17T03:08:59Z), cấp lúc time=1787022539 "+
			"(2026-08-18T03:08:59Z): cửa sổ ĐÚNG 60 ngày. Đọc refresh theo tiền lệ claude.go. "+
			"CHƯA ĐO: có xoay vòng khi refresh hay không — qua 3 ngày dùng, mtime auth.json vẫn "+
			"là giây đăng nhập nên CLI chưa ghi lại file này lần nào"),
		Chua(NLDanhTinh, "CHƯA ĐỌC ĐƯỢC (đo 21/08 trên hồ sơ đăng nhập thật): Cursor\\auth.json "+
			"có ĐÚNG hai khoá accessToken/refreshToken, không có email/userEmail/user_email; "+
			"payload JWT cũng chỉ có `sub` dạng mã đục `google-oauth|<id>`. Dòng khai cũ nói "+
			"đọc được ba trường đó là SAI — Identity() trả rỗng trên chính máy đã đăng nhập. "+
			"`cursor-agent status` in được email nhưng đọc bằng cách chạy CLI con là đánh đổi "+
			"khác, chưa đo"),
	}
}
