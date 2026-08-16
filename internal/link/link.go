// Package link trừu tượng thao tác "nối phần dùng chung" — điểm khác biệt OS
// duy nhất đáng kể. Windows dùng junction (không đòi quyền admin), Linux dùng
// symlink. Xem link_windows.go / link_linux.go cho phần theo nền tảng.
package link

import "os"

// copyFile là đường lui khi không tạo được link (chép nội dung).
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
