package profile

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// Tên hồ sơ đi thẳng từ người dùng vào đường dẫn thư mục, và công cụ này XOÁ
// THƯ MỤC. Vì vậy tên phải qua whitelist, y như whitelist khoá cấu hình: liệt kê
// cái được phép, không liệt kê cái bị cấm.
//
// Đã đo trước khi vá — ghép thẳng tên vào đường dẫn cho ra:
//
//	"phu"          -> ~/.ai-accounts/claude/phu     (đúng)
//	"../../.claude"-> ~/.claude                     (DỮ LIỆU CLAUDE THẬT)
//	""             -> ~/.ai-accounts/claude         (mọi tài khoản claude)
//	".."           -> ~/.ai-accounts                (mọi tài khoản, mọi provider)
//
// Không chỉ dòng lệnh mới nhập được tên: dashboard cũng có form thêm hồ sơ, mà
// dashboard thì đang mở ra internet.
const maxNameLen = 64

// Tên thiết bị của Windows. Mở file tên "NUL" là nói chuyện với thiết bị chứ
// không phải với file, kể cả khi có phần mở rộng ("NUL.txt" vẫn là thiết bị).
var winDevices = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true,
	"com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true,
	"lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// ValidName kiểm tra một tên tài khoản (hoặc tên provider) có an toàn để dùng
// làm một đoạn đường dẫn không.
func ValidName(s string) error {
	if s == "" {
		return fmt.Errorf("tên rỗng")
	}
	if len(s) > maxNameLen {
		return fmt.Errorf("tên dài quá %d ký tự", maxNameLen)
	}
	if s == "." || s == ".." {
		return fmt.Errorf("%q là đường dẫn tương đối, không phải tên", s)
	}
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.'
		if !ok {
			return fmt.Errorf("tên chỉ được dùng chữ, số, '-', '_', '.' — thấy %q", r)
		}
	}
	// Dấu chấm đầu: vừa để ẩn hồ sơ khỏi `ds`, vừa đụng thư mục nội bộ `.clones`.
	if strings.HasPrefix(s, ".") {
		return fmt.Errorf("tên không được bắt đầu bằng dấu chấm")
	}
	if strings.HasPrefix(s, "-") {
		return fmt.Errorf("tên không được bắt đầu bằng '-' (sẽ bị hiểu là cờ dòng lệnh)")
	}
	// Windows lặng lẽ cắt dấu chấm/khoảng trắng ở cuối, nên "phu." và "phu" hoá
	// thành cùng một thư mục — hai hồ sơ khác tên mà chung dữ liệu.
	if strings.HasSuffix(s, ".") {
		return fmt.Errorf("tên không được kết thúc bằng dấu chấm")
	}
	base := strings.ToLower(s)
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	if winDevices[base] {
		return fmt.Errorf("%q là tên thiết bị của Windows, không dùng làm tên hồ sơ được", s)
	}
	return nil
}

// insideStore trả về true nếu dir nằm THỰC SỰ BÊN TRONG một trong các kho hồ sơ
// (không tính chính thư mục kho).
//
// Đây là lớp phòng thủ thứ hai, và là lớp đáng tin hơn: ValidName phải được gọi
// đúng chỗ mới có tác dụng, còn lớp này thì mọi lối xoá đều phải đi qua.
func insideStore(dir string) bool {
	for _, root := range []string{
		paths.AccountsRoot(),
		ClonesRoot(),
		paths.LegacyClaudeAccounts(),
	} {
		if under(root, dir) {
			return true
		}
	}
	return false
}

func under(root, dir string) bool {
	root, dir = filepath.Clean(root), filepath.Clean(dir)
	if runtime.GOOS == "windows" {
		// Windows không phân biệt hoa thường; filepath.Rel thì có.
		root, dir = strings.ToLower(root), strings.ToLower(dir)
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return false
	}
	// "." nghĩa là chính thư mục kho — không được xoá. ".." nghĩa là đã thoát ra.
	return rel != "." && !strings.HasPrefix(rel, "..")
}
