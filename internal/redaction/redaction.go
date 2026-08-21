// Package redaction che BÍ MẬT và DANH TÍNH trong nhật ký phiên, ngay trước
// lúc nội dung nhật ký RỜI KHỎI đĩa để đi tới một mặt đọc.
//
// VÌ SAO CHE LÚC ĐỌC CHỨ KHÔNG CHE LÚC GHI (đo 22/08/2026)
//
// Đường ghi KHÔNG chặn được, và đây là ràng buộc kiến trúc chứ không phải lười:
// `profile.StartDetached` gán thẳng file handle cho tiến trình con
// (`c.Stdout, c.Stderr = f, f`) rồi tiến trình cha THOÁT. Không còn ai đứng
// giữa dòng ghi để lọc. Muốn lọc lúc ghi thì phải nuôi một tiến trình trung
// gian sống suốt lượt chạy — tức bỏ hẳn kiến trúc "bật rồi buông" mà cả công
// cụ đang dựa vào. (Cùng lý do khiến `nhatky.Don` phải chặn ngân sách đĩa ở
// lượt SAU thay vì xoay file giữa chừng.)
//
// May là chỗ đáng chặn cũng không phải lúc ghi. File nằm ở ~/.ai-accounts/
// .nhat-ky với quyền 0600, trong thư mục nhà của chính người dùng. Rủi ro thật
// là lúc nội dung RA KHỎI chỗ đó, và có đúng hai cửa ra:
//
//   - `dash` phục vụ nhật ký qua HTTP. `server.go` tự biết mình có thể không
//     nằm trên loopback (`s.exposed = !isLoopbackAddr(host)`).
//   - `api.readLogs` lấy nguyên văn nhật ký làm OUTPUT của bước agent rồi nạp
//     thẳng vào PROMPT của bước SAU — tức trao cho một agent khác, có thể là
//     một nhà cung cấp khác hẳn.
//
// Cả hai cửa đều đi qua `nhatky.Duoi` hoặc `nhatky.BoDau`. Che ở đó thì không
// mặt gọi nào quên được — kể cả mặt gọi viết sau này.
//
// ĐƯỜNG MÁY KHÔNG BỊ ĐỤNG: `api.phanLoaiPhienChet` đọc thẳng `os.ReadFile` rồi
// đưa cho `adapter.DocKetQua`. Nó không đi qua hai hàm trên, nên phép quyết
// trạng thái phiên vẫn nhìn thấy byte gốc.
//
// # MẪU CHE DỰA TRÊN SỐ ĐO THẬT, KHÔNG PHẢI DANH SÁCH ĐOÁN
//
// Quét 22 file / 6.797 dòng / 10,6 MB nhật ký thật ngày 21-22/08/2026 (bỏ nhật
// ký của chính phiên đang đo — xem ghi chú cuối gói về hiệu ứng người quan sát):
//
//	tên tài khoản, mọi dạng      2.060 khớp
//	  trong đó: đường dẫn nhà    1.536
//	            đường dẫn bẹp       46
//	email                          179 khớp / 14 file (75 là địa chỉ thật,
//	                                   còn lại là đồ giả trong bài kiểm)
//	token, khoá API (CÓ giá trị)     0 khớp
//
// Sau khi cắm tầng che, đo lại qua đúng cửa `nhatky.Duoi`: cả bốn nhóm về 0,
// 489 trường "signature" còn nguyên, 6.092 dòng JSON không dòng nào hỏng.
//
// Hai điều rút ra, cả hai đều đổi thiết kế:
//
//  1. KHÔNG có khoá nào đang rò. Các lớp mẫu khoá dưới đây là PHÒNG NGỪA, và
//     bài viết này nói thẳng thế. Chúng vẫn đáng có: nhật ký là NDJSON thô của
//     mọi thứ agent nhìn thấy, nên chỉ cách một lệnh `cat .credentials.json`
//     là có khoá thật nằm trong đó.
//  2. KHÔNG được đặt luật "chuỗi base64 dài = bí mật". 471 khối base64 (dài
//     tới 9.108 ký tự) trong nhật ký là trường `"signature"` của Claude — chữ
//     ký khối thinking, không phải khoá. Luật đó sẽ băm nát 471 trường hợp lệ
//     để đổi lấy 0 bí mật. Nên mẫu JWT dưới đây bắt buộc phải có ĐỦ BA đoạn
//     ngăn bằng dấu chấm; khối signature không có dấu chấm nào nên không dính.
//
// HIỆU ỨNG NGƯỜI QUAN SÁT — cái bẫy của chính phép đo này. Lượt quét đầu tiên
// báo có khoá Anthropic, token GitHub, token Slack, mỗi thứ đúng 1 lần. Cả ba
// nằm trong CÙNG MỘT file: nhật ký của chính phiên đang quét. Nhật ký ghi lại
// từng lệnh agent gõ, nên câu lệnh `grep sk-ant- ...` vừa chạy đã tự trở thành
// một dòng khớp. Ai đo nhật ký của phiên đang chạy mà không trừ file của chính
// mình ra thì sẽ tìm thấy đúng thứ mình vừa đi tìm — và tưởng là đã phát hiện
// ra một vụ rò.
package redaction

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Nhãn thay thế. KHÔNG chứa dấu nháy kép hay gạch chéo ngược, vì nhật ký là
// NDJSON: nhét hai ký tự đó vào giữa một chuỗi JSON là làm hỏng dòng, và bên
// đọc (provider.DocKetQua, hoặc agent ở bước sau) sẽ vấp.
const (
	NhanKhoa      = "[đã-che:khoá]"
	NhanNguoiDung = "[đã-che:người-dùng]"
)

// mau là một luật che.
//
// biMat tách "bí mật thật" khỏi "danh tính". Chỉ nhóm biMat mới được dùng để
// CHẶN COMMIT (xem TimBiMat): repo này có email trong trailer git và có đường
// dẫn `C:\Users\...` nằm trong tài liệu — chặn hai thứ đó thì bài kiểm đỏ ngay
// từ dòng đầu và không ai giữ nó.
type mau struct {
	ten   string
	re    *regexp.Regexp
	thay  string
	biMat bool
}

// bang là NGUỒN DUY NHẤT: cả phép che (Che) lẫn bài kiểm quét repo (TimBiMat)
// đọc chung một bảng. Tách hai danh sách ra thì chúng lệch nhau, và cái lệch
// đó im lặng — bài kiểm vẫn xanh trong khi tầng che đã bỏ sót.
var bang = []mau{
	// ---- Bí mật thật (chặn cả ở nhật ký lẫn ở commit) ----
	{
		ten: "khoá Anthropic (sk-ant-)",
		// Phủ cả khoá API (sk-ant-api03-) lẫn token OAuth (sk-ant-oat01-,
		// sk-ant-ort01-) — cùng tiền tố, khác hậu tố.
		re:    regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{16,}`),
		thay:  NhanKhoa,
		biMat: true,
	},
	{
		ten:   "khoá tiền tố sk- (OpenAI và tương tự)",
		re:    regexp.MustCompile(`\bsk-[A-Za-z0-9]{24,}`),
		thay:  NhanKhoa,
		biMat: true,
	},
	{
		ten:   "token GitHub (ghp_/gho_/ghu_/ghs_/ghr_)",
		re:    regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}`),
		thay:  NhanKhoa,
		biMat: true,
	},
	{
		ten:   "token GitHub dạng mới (github_pat_)",
		re:    regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}`),
		thay:  NhanKhoa,
		biMat: true,
	},
	{
		ten:   "khoá Google (AIza)",
		re:    regexp.MustCompile(`\bAIza[A-Za-z0-9_-]{35}`),
		thay:  NhanKhoa,
		biMat: true,
	},
	{
		ten:   "khoá truy cập AWS (AKIA/ASIA)",
		re:    regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`),
		thay:  NhanKhoa,
		biMat: true,
	},
	{
		ten:   "token Slack (xox*-)",
		re:    regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{12,}`),
		thay:  NhanKhoa,
		biMat: true,
	},
	{
		ten: "JWT (ba đoạn ngăn bằng dấu chấm)",
		// BA đoạn là bắt buộc — xem ghi chú đầu gói về 471 trường "signature".
		re:    regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`),
		thay:  NhanKhoa,
		biMat: true,
	},
	{
		ten:   "khối khoá riêng PEM",
		re:    regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
		thay:  NhanKhoa,
		biMat: true,
	},
	{
		ten: "trường JSON chứa token/khoá",
		// Bắt GIÁ TRỊ, không bắt tên trường. Đã đo: 218 lần chữ "refreshToken"
		// trong nhật ký đều là văn xuôi hoặc mã nguồn đang được đọc, không kèm
		// giá trị. Bắt theo tên trường thôi thì che nhầm 218 chỗ vô hại.
		//
		// GIÁ TRỊ phải nằm trọn trong BỘ CHỮ của token (không khoảng trắng,
		// không nháy ngược, không ngoặc) và dài từ 20 ký tự. Bản đầu viết
		// `[^"]{12,}` và bài kiểm quét repo bắt được ngay 21 chỗ sai — tất cả
		// là mã nguồn Go nối chuỗi kiểu
		//
		//	`{"accessToken":"` + jwtGia(con) + `"}`
		//
		// tức thứ khớp là CÚ PHÁP NỐI CHUỖI chứ không phải bí mật. Ngưỡng 20
		// cũng loại nốt các giá trị giả ngắn kiểu "khong-co-dau-cham".
		re:    regexp.MustCompile(`(?i)("(?:access|refresh|id|session|client|api)[_-]?(?:token|key|secret)"\s*:\s*")[A-Za-z0-9._~+/=-]{20,}(")`),
		thay:  "${1}" + NhanKhoa + "${2}",
		biMat: true,
	},
	{
		ten:   "tiêu đề Authorization: Bearer",
		re:    regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]{16,}`),
		thay:  "${1}" + NhanKhoa,
		biMat: true,
	},
	{
		ten: "gán biến môi trường chứa khoá",
		// Bắt buộc có dấu `=`. Cho phép cả `:` thì văn xuôi kiểu
		// "GROK_API_KEY: cần đặt biến này" (đã đo, 9 lần) bị che nhầm.
		//
		// PHÂN BIỆT HOA THƯỜNG (không có `(?i)`): tên biến môi trường là CHỮ
		// HOA GẠCH DƯỚI. Bản đầu có `(?i)` và bài kiểm quét repo bắt ngay hai
		// chỗ sai — `p.HasToken = ad.HasToken(...)` và
		// `const duongToken = "vendor/token.css"`. Định danh camelCase trong mã
		// Go đông hơn biến môi trường vài bậc, nên `(?i)` ở đây đổi một chút
		// phủ sóng lấy rất nhiều tiếng ồn.
		re:    regexp.MustCompile(`\b([A-Z][A-Z0-9_]*(?:TOKEN|SECRET|PASSWORD|APIKEY|API_KEY)[A-Z0-9_]*\s*=\s*)["']?[^\s"',;]{12,}`),
		thay:  "${1}" + NhanKhoa,
		biMat: true,
	},

	// ---- Danh tính (che ở nhật ký, KHÔNG chặn commit) ----
	{
		ten: "tên người dùng trong đường dẫn nhà",
		// Đo được 1.523 khớp / 22 file, và CẢ 1.523 đều là đường dẫn nhà thật
		// (0 dương tính giả). Chỉ thay đoạn TÊN, giữ nguyên phần đuôi: phần
		// đuôi là chỗ agent làm việc, mất nó thì nhật ký hết truy nguyên được.
		//
		// `{1,4}` DẤU NGĂN, không phải `{1,2}`. Bản đầu viết `{1,2}` và chạy
		// thử trên nhật ký thật thì còn sót 916 chỗ: JSON LỒNG JSON. Một dòng
		// nhật ký là JSON, bên trong có chuỗi ghi lại lời gọi công cụ vốn cũng
		// là JSON, nên gạch ngược bị thoát HAI lần:
		//
		//	Read {"file_path":"C:\\\\Users\\\\Administrator\\\\..."}
		//
		// Bốn gạch ngược. Mẫu `{1,2}` ăn hai cái rồi đòi chữ cái, gặp gạch
		// ngược thứ ba nên trượt sạch — trượt IM LẶNG, vì không có gì báo.
		//
		// `~` nằm trong bộ chữ của tên để phủ tên 8.3 của Windows
		// (`C:\Users\ADMINI~1\AppData`, đo được 18 lần).
		//
		// CHỈ phủ dạng Windows — máy này là Windows và dạng `/home/<tên>` đo
		// được 0 khớp. Thêm mẫu cho nó là đoán, nên không thêm.
		re:   regexp.MustCompile(`(?i)(Users[\\/]{1,4})([A-Za-z0-9._~-]+)`),
		thay: "${1}" + NhanNguoiDung,
	},
	{
		ten: "tên người dùng trong đường dẫn đã bẹp thành gạch nối",
		// Claude Code đặt tên thư mục dự án bằng cách bẹp cả đường dẫn, thay
		// mọi dấu ngăn bằng gạch nối:
		//
		//	.clones/claude/tns/1/projects/C--Users-Administrator-Projects-...
		//
		// Không còn gạch chéo nào nên luật trên không thấy. Đo được 46 lần.
		// Phần tên dừng ở gạch nối kế tiếp (`[A-Za-z0-9._]+`, KHÔNG có `-`),
		// nếu không nó nuốt nốt phần đuôi đường dẫn — mà đuôi là chỗ cần cho
		// truy nguyên.
		re:   regexp.MustCompile(`(?i)([A-Za-z]--Users-)([A-Za-z0-9._]+)`),
		thay: "${1}" + NhanNguoiDung,
	},
	{
		ten: "địa chỉ email",
		// Giữ ký tự đầu và tên miền: đủ để phân biệt "email của người dùng" với
		// "email của bot CI" khi truy nguyên, mà không còn là địa chỉ gửi được.
		//
		// NHÁNH `\uXXXX` Ở ĐẦU LÀ BẮT BUỘC, không phải cho đẹp. Đo trên nhật ký
		// thật: Antigravity ghi dấu ngoặc nhọn thành chuỗi thoát JSON, nên trong
		// file có `<noreply@anthropic.com>`. Bản đầu không có nhánh
		// này thì phần cục bộ nuốt luôn `u003c`, thay xong còn lại `\` đứng trước
		// nhãn — tức `\[` — và `\[` KHÔNG phải chuỗi thoát JSON hợp lệ. Hai dòng
		// JSON hỏng thật trong hai file antigravity, và hỏng ở phía bên kia:
		// bước agent sau nhận một dòng không parse được.
		//
		// Nhánh này ăn trọn chuỗi thoát rồi TRẢ LẠI NGUYÊN VẸN qua ${1}, nên
		// email vẫn được che mà dòng vẫn hợp lệ.
		re:   regexp.MustCompile(`(\\u[0-9a-fA-F]{4}|^|[^A-Za-z0-9._%+-])([A-Za-z0-9._%+-])[A-Za-z0-9._%+-]*(@[A-Za-z0-9.-]+\.[A-Za-z]{2,})`),
		thay: "${1}${2}***${3}",
	},
}

// Che thay mọi bí mật và danh tính trong s bằng nhãn.
//
// Giữ nguyên cấu trúc NDJSON: nhãn thay thế không chứa dấu nháy kép, gạch chéo
// ngược hay ký tự xuống dòng, nên một dòng JSON hợp lệ đi vào vẫn là một dòng
// JSON hợp lệ đi ra.
func Che(s string) string {
	if s == "" {
		return s
	}
	for _, m := range bang {
		s = m.re.ReplaceAllString(s, m.thay)
	}
	if re := reTenNguoiDung(); re != nil {
		s = re.ReplaceAllString(s, NhanNguoiDung)
	}
	return s
}

// tenNguoiDungCache giữ regex đã dựng cho từng tên, vì Che chạy trên chuỗi cỡ
// megabyte và biên dịch lại mỗi lần là phí. KHÔNG dùng sync.Once: tên lấy từ
// biến môi trường, mà test thì đổi biến môi trường (t.Setenv) — nhớ cứng một
// lần thì test thứ hai đọc phải tên của test thứ nhất.
var tenNguoiDungCache sync.Map // string -> *regexp.Regexp

// reTenNguoiDung dựng luật che CHÍNH TÊN tài khoản đăng nhập, ở mọi chỗ nó
// xuất hiện chứ không riêng trong đường dẫn.
//
// VÌ SAO CẦN, dù đã có hai luật đường dẫn ở trên: chạy thử trên nhật ký thật
// thì sau khi che hai luật kia vẫn còn 525 lần tên tài khoản. Chúng không nằm
// trong đường dẫn nào cả — chúng là CỘT CHỦ SỞ HỮU của `ls -l`:
//
//	drwxr-xr-x 1 Administrator 197121 0 Aug 21 14:47 .
//
// Không mẫu hình dạng nào bắt được một cái tên đứng trơ như thế. Nhưng ta BIẾT
// cái tên đó — nó là tên thư mục nhà. Đây cũng là luật duy nhất trong gói phụ
// thuộc môi trường, và đó là chủ ý: thứ cần giấu là danh tính của MÁY NÀY.
//
// NGƯỠNG 4 KÝ TỰ. Tên ngắn (`dev`, `ci`, `adm`) là từ thường gặp trong văn
// xuôi lẫn mã nguồn; thay bừa thì băm nát nhật ký để đổi lấy một chút riêng tư
// mà kẻ đọc đoán ra trong một nốt nhạc. Thà không che còn hơn che hỏng.
func reTenNguoiDung() *regexp.Regexp {
	nha, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	ten := filepath.Base(nha)
	if len([]rune(ten)) < 4 || ten == "." || ten == string(filepath.Separator) {
		return nil
	}
	if v, ok := tenNguoiDungCache.Load(ten); ok {
		return v.(*regexp.Regexp)
	}
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(ten) + `\b`)
	tenNguoiDungCache.Store(ten, re)
	return re
}

// Phat là một chỗ nghi có bí mật, đủ để người đọc đi tới đúng dòng mà xem.
type Phat struct {
	Mau  string // tên luật đã khớp
	Dong int    // số dòng, đếm từ 1
	// Trich là đoạn khớp ĐÃ CHE. Báo lỗi mà in nguyên khoá ra thì chính bản
	// báo lỗi (log CI, ảnh chụp màn hình) trở thành chỗ rò tiếp theo.
	Trich string
}

// TimBiMat tìm các đoạn khớp nhóm BÍ MẬT (bỏ qua nhóm danh tính).
//
// Dùng cho bài kiểm quét repo: che nhật ký và chặn commit phải soi chung một
// bảng luật, nếu không thì thêm luật ở chỗ này mà quên chỗ kia.
func TimBiMat(s string) []Phat {
	var ra []Phat
	for _, m := range bang {
		if !m.biMat {
			continue
		}
		for _, vt := range m.re.FindAllStringIndex(s, -1) {
			ra = append(ra, Phat{
				Mau:   m.ten,
				Dong:  strings.Count(s[:vt[0]], "\n") + 1,
				Trich: Che(s[vt[0]:vt[1]]),
			})
		}
	}
	return ra
}
