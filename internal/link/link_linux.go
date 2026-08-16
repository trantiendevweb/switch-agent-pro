//go:build linux

package link

import "os"

// IsLink trên Linux: symlink hiện thẳng qua Lstat.
func IsLink(path string) (bool, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return fi.Mode()&os.ModeSymlink != 0, nil
}

// LinkDir dùng symlink native (không cần quyền đặc biệt).
func LinkDir(target, linkPath string) error { return os.Symlink(target, linkPath) }

// LinkFile thử symlink -> chép.
func LinkFile(target, linkPath string) error {
	if err := os.Symlink(target, linkPath); err == nil {
		return nil
	}
	return copyFile(target, linkPath)
}

// Unlink gỡ symlink (os.Remove xoá chính link, không xuyên qua target).
func Unlink(path string, _ bool) error { return os.Remove(path) }
