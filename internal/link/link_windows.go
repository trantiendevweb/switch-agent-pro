//go:build windows

package link

import (
	"os"
	"os/exec"
	"syscall"
)

// IsLink: junction trên Windows KHÔNG hiện là ModeSymlink; phải kiểm cờ
// reparse point trực tiếp. Đây là chỗ v1 phải cẩn thận để không xoá nhầm.
func IsLink(path string) (bool, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		return false, err
	}
	return attrs&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}

// LinkDir tạo junction (mklink /J) — KHÔNG đòi quyền quản trị.
func LinkDir(target, linkPath string) error {
	return exec.Command("cmd", "/c", "mklink", "/J", linkPath, target).Run()
}

// LinkFile thử symlink -> hardlink -> chép.
func LinkFile(target, linkPath string) error {
	if err := os.Symlink(target, linkPath); err == nil {
		return nil
	}
	if err := os.Link(target, linkPath); err == nil {
		return nil
	}
	return copyFile(target, linkPath)
}

// Unlink gỡ link. Thư mục (junction) gỡ bằng rmdir để KHÔNG xuyên qua xoá dữ
// liệu thật ở đầu bên kia.
func Unlink(path string, isDir bool) error {
	if isDir {
		return exec.Command("cmd", "/c", "rmdir", path).Run()
	}
	return os.Remove(path)
}
