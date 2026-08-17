//go:build linux

package acl

import "os"

// Trên Linux bit quyền là thật, nên chỉ cần chmod. Thư mục cần bit x để vào
// được, file thì không.
func Restrict(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return os.Chmod(path, 0o700)
	}
	return os.Chmod(path, 0o600)
}

// Check: nhóm hoặc người ngoài có bit nào là hỏng.
func Check(path string) (bool, string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, "", err
	}
	m := fi.Mode().Perm()
	if m&0o077 != 0 {
		return false, "quyền " + m.String() + " — nhóm/người khác đọc được", nil
	}
	return true, "chỉ chủ sở hữu (" + m.String() + ")", nil
}
