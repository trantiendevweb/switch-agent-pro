package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Đọc danh tính Antigravity từ NHẬT KÝ CỦA CHÍNH `agy`, không từ
// ~/.gemini/google_accounts.json.
//
// VÌ SAO KHÔNG PHẢI google_accounts.json — đây là cái bẫy mà sổ nợ (ô V2) cảnh
// báo, và phép đo 21/08/2026 xác nhận nó có thật:
//
//   - google_accounts.json là file của GEMINI CLI, một sản phẩm khác. Trên máy
//     đo, nó mang dấu thời gian 18/08 10:09 và KHÔNG hề đổi qua các lượt chạy
//     `agy` ngày 21/08 (21:16, 21:17, 21:18).
//   - Nó có thể chứa một email CÓ THẬT nhưng SAI NGƯỜI — đúng định dạng nên
//     không ai nghi. `antigravity.go` chốt: hiện nhầm email tệ hơn không hiện gì.
//
// VÌ SAO NHẬT KÝ LÀ NGUỒN PHÂN BIỆT ĐƯỢC — hai phép đo độc lập, cả hai đều là
// thư mục KHÔNG HỀ CÓ google_accounts.json mà nhật ký VẪN in ra email:
//
//  1. HOME giả (đổi cả USERPROFILE + APPDATA + LOCALAPPDATA + HOME) rồi chạy
//     `agy -p`: thư mục mới dựng không có google_accounts.json, nhật ký vẫn ghi
//     `applyAuthResult: email=…`.
//  2. Hồ sơ thật `~/.ai-accounts/antigravity/may`: cả cây `.gemini` của nó
//     KHÔNG có google_accounts.json, mà nhật ký 21/08 14:42 vẫn ghi
//     `consumerOAuth: authenticated successfully as …` — dòng của chính lượt
//     đăng nhập bằng trình duyệt.
//
// Tức email trong nhật ký đến từ TOKEN trong Windows Credential Manager (khoá
// `gemini:antigravity`), không phải từ file của Gemini CLI. Đó là điều mà việc
// "hai giá trị trùng nhau" một mình KHÔNG chứng minh nổi.
//
// Quét cả cây hồ sơ tìm chuỗi email: nhật ký là file DUY NHẤT mang nó. Các file
// cấu hình khác chỉ có trustedWorkspaces, kiểu xác thực, remoteControlHostname.
//
// PHẦN CÒN NỢ, nói thẳng: một lần ĐĂNG NHẬP LẠI bằng tài khoản KHÁC, thực hiện ở
// một HOME khác, sẽ để lại nhật ký cũ trong hồ sơ này — và hồ sơ này sẽ hiện
// email cũ cho tới lượt `agy` kế tiếp chạy trong nó. Cửa sổ đó hẹp vì
// TachDuocTaiKhoan() = false (mỗi máy MỘT tài khoản Antigravity), nhưng nó có
// thật.
//
// MỘT RÀO ĐÃ ĐO RỒI BỎ, ghi lại để người sau khỏi thử lại: dùng
// `LastWritten` của mục Credential Manager làm mốc — "nhật ký cũ hơn mốc thì trả
// rỗng". Đo 21/08: LastWritten = 21:16:59 (một lượt chạy ở HOME thật vừa làm mới
// token), còn nhật ký hợp lệ của hồ sơ `may` là 20:15:39. Rào đó sẽ trả RỖNG cho
// đúng cái hồ sơ đang chạy đúng. Nguyên nhân: LastWritten đổi cả khi LÀM MỚI
// token lẫn khi đăng nhập lại, mà không mở blob ra thì không phân biệt được hai
// việc — và mở blob là thứ TokenExpiry đã cố ý từ chối. Rào sai hướng thì thà
// không có.
const nhatKyToiDaAgy = 16 << 20

// Hai dạng dòng đã thấy trong nhật ký thật:
//
//	server_oauth.go:190] applyAuthResult: email=<addr>, authMethod=consumer, quotaProject=
//	server_oauth.go:195] OAuth: authenticated successfully as <addr>
//	browser.go:161]      consumerOAuth: authenticated successfully as <addr>
var reEmailNhatKyAgy = regexp.MustCompile(
	`(?:applyAuthResult: email=|authenticated successfully as )([^\s,]+@[^\s,]+\.[^\s,]+)`)

// nhatKyMoiNhatAgy trả đường dẫn file `cli-*.log` MỚI NHẤT trong thư mục nhật ký
// của một hồ sơ, hoặc "" nếu không có file nào.
//
// Bỏ qua `cli.log` — nó là symlink trỏ tới file mới nhất, đọc cả hai là đọc hai
// lần cùng một nội dung. Xếp theo ModTime, hoà thì theo tên (tên có dấu thời
// gian nên thứ tự tên cũng là thứ tự thời gian).
func nhatKyMoiNhatAgy(logDir string) string {
	ds, err := os.ReadDir(logDir)
	if err != nil {
		return ""
	}
	type muc struct {
		ten  string
		info os.FileInfo
	}
	var ds2 []muc
	for _, d := range ds {
		ten := d.Name()
		if d.IsDir() || ten == "cli.log" || !strings.HasPrefix(ten, "cli-") ||
			!strings.HasSuffix(ten, ".log") {
			continue
		}
		fi, err := d.Info()
		if err != nil {
			continue
		}
		ds2 = append(ds2, muc{ten, fi})
	}
	if len(ds2) == 0 {
		return ""
	}
	sort.Slice(ds2, func(i, j int) bool {
		if ds2[i].info.ModTime().Equal(ds2[j].info.ModTime()) {
			return ds2[i].ten < ds2[j].ten
		}
		return ds2[i].info.ModTime().Before(ds2[j].info.ModTime())
	})
	return filepath.Join(logDir, ds2[len(ds2)-1].ten)
}

// danhTinhTuNhatKyAgy đọc email từ nhật ký mới nhất của hồ sơ `configDir`.
//
// Lấy dòng xác thực CUỐI CÙNG trong file, không phải dòng đầu: một phiên dài có
// thể xác thực lại giữa chừng, và lần sau mới là lần đang có hiệu lực.
//
// KHÔNG chạm vào token: hàm này chỉ đọc file văn bản mà `agy` tự ghi.
func danhTinhTuNhatKyAgy(configDir string) string {
	f := nhatKyMoiNhatAgy(filepath.Join(configDir, ".gemini", "antigravity-cli", "log"))
	if f == "" {
		return ""
	}
	fi, err := os.Stat(f)
	if err != nil || fi.Size() > nhatKyToiDaAgy {
		return ""
	}
	b, err := os.ReadFile(f)
	if err != nil {
		return ""
	}
	m := reEmailNhatKyAgy.FindAllSubmatch(b, -1)
	if len(m) == 0 {
		return ""
	}
	return string(m[len(m)-1][1])
}
