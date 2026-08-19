//go:build !windows

package process

import "time"

// sagent chỉ hỗ trợ Windows. File này cung cấp đủ stub để build DỪNG với một
// cái tên đọc được, thay vì đổ ra một trang "undefined: parentMap".
//
// Tên file KHÔNG được kết thúc bằng `_windows.go`: Go áp ràng buộc build ngầm
// theo hậu tố tên file, nên `khong_windows.go` chỉ được biên dịch TRÊN Windows —
// mâu thuẫn với `//go:build !windows` ở trên và file không bao giờ vào build.
// Đã dẫm đúng cái bẫy đó một lần khi viết chỗ này.
//
// Vì sao bỏ Linux: mọi thứ khiến công cụ này đáng dùng đều là chi tiết Windows —
// junction thay symlink, ACL thay bit quyền, taskkill thay process group, tên
// thiết bị NUL/COM1, chuyện Windows lặng lẽ cắt dấu chấm cuối tên thư mục. Mỗi
// phép đo trong docs/DO-LUONG.md đều là một phép đo Windows. Giữ nhánh Linux mà
// KHÔNG có máy Linux để chạy thì đó không phải hỗ trợ, đó là lời hứa suông —
// đúng thứ tài liệu kia lập ra để chống.
func IsAlive(int) bool       { return sagentChiHoTroWindows_xemREADME() }
func Kill(int) error         { sagentChiHoTroWindows_xemREADME(); return nil }
func parentMap() map[int]int { sagentChiHoTroWindows_xemREADME(); return nil }

func StartTime(int) (time.Time, bool) { sagentChiHoTroWindows_xemREADME(); return time.Time{}, false }

func procTable() map[int]procEnt { sagentChiHoTroWindows_xemREADME(); return nil }

type procEnt struct {
	ppid int
	ten  string
}
