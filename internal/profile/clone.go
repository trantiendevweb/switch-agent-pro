package profile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// ClonesRoot là nơi chứa bản sao dùng cho chạy song song. Để dưới thư mục có
// dấu chấm ở đầu để `List()` không nhầm nó là một provider.
func ClonesRoot() string { return filepath.Join(paths.AccountsRoot(), ".clones") }

// CloneDir là thư mục config của bản clone thứ n.
func CloneDir(prov, account string, n int) string {
	return filepath.Join(ClonesRoot(), prov, account, strconv.Itoa(n))
}

// Clone tạo N thư mục config biệt lập cho CÙNG một tài khoản, để chạy nhiều
// phiên song song.
//
// Vì sao không cho N tiến trình dùng chung một thư mục: chúng sẽ ĐUA NHAU GHI
// .claude.json và làm hỏng file (trust dialog nằm trong đó). Mỗi bản clone có
// file riêng nên không giẫm chân nhau.
//
// ⚠ CHƯA ĐO: token bị chép ra N chỗ thì khi hết hạn, N tiến trình có thể cùng
// refresh một lúc. Hành vi đó chưa được đo (xem docs/DO-LUONG.md), nên `fleet`
// in cảnh báo chứ không hứa là an toàn.
func Clone(a provider.Adapter, account string, copies int) ([]string, error) {
	base, ok := ResolveDir(a.Name(), account)
	if !ok {
		return nil, fmt.Errorf("không có %s:%s — tạo trước bằng: sagent them %s:%s",
			a.Name(), account, a.Name(), account)
	}
	if !a.HasToken(base) {
		return nil, fmt.Errorf("%s:%s chưa đăng nhập — chạy `sagent %s:%s` rồi /login trước",
			a.Name(), account, a.Name(), account)
	}

	var dirs []string
	for i := 1; i <= copies; i++ {
		dir := CloneDir(a.Name(), account, i)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return dirs, err
		}
		// Phần dùng chung: nối link như hồ sơ thường.
		if _, err := LinkShared(a, dir); err != nil {
			return dirs, err
		}
		// File riêng: chép NGUYÊN VĂN. Đây là cùng một tài khoản nên bản sao
		// mang đúng danh tính đó là điều mong muốn — khác hẳn việc tạo tài
		// khoản mới (chỗ đó phải lọc qua whitelist).
		for _, name := range a.PrivateFiles() {
			src := filepath.Join(base, name)
			data, err := os.ReadFile(src)
			if err != nil {
				continue // chưa có thì thôi, không phải lỗi
			}
			if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
				return dirs, err
			}
		}
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

// CleanClones xoá mọi bản clone của một tài khoản, AN TOÀN.
//
// Bắt buộc phải có hàm này: thư mục clone đầy junction trỏ về ~/.claude, nên
// một cú `rm -rf` hay `Remove-Item -Recurse` của người dùng có thể xuyên qua
// link xoá luôn dữ liệu thật. Ở đây dùng lại Remove() — gỡ từng link, kiểm
// sạch, rồi mới xoá.
func CleanClones(prov, account string) (int, error) {
	root := filepath.Join(ClonesRoot(), prov, account)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := Remove(filepath.Join(root, e.Name())); err != nil {
			return n, err
		}
		n++
	}
	// Thư mục cha giờ rỗng, xoá nốt cho gọn (không đệ quy).
	_ = os.Remove(root)
	return n, nil
}

// StartDetached chạy CLI ở chế độ nền, log đổ vào file, không chiếm terminal.
// workDir là chỗ agent làm việc (git worktree riêng, hoặc rỗng = thư mục hiện tại).
// Trả về PID. Cố ý KHÔNG Wait() — phiên phải sống tiếp sau khi lệnh này thoát.
func StartDetached(a provider.Adapter, dir string, args []string, logPath, workDir string) (int, error) {
	cmdPath, err := a.Command()
	if err != nil {
		return 0, err
	}
	f, err := os.Create(logPath)
	if err != nil {
		return 0, err
	}
	c := exec.Command(cmdPath, args...)
	c.Stdout, c.Stderr = f, f
	c.Stdin = nil
	c.Env = append(filterEnv(os.Environ(), a.EnvVar()), a.EnvVar()+"="+dir)
	if workDir != "" {
		c.Dir = workDir
	} else if wd, err := os.Getwd(); err == nil {
		// Mặc định: thư mục hiện tại — agent làm trên đúng project bạn đang đứng.
		c.Dir = wd
	}
	if err := c.Start(); err != nil {
		f.Close()
		return 0, err
	}
	// Tiến trình con đã có bản sao handle của riêng nó, nên cha PHẢI đóng bản
	// của mình. Không đóng thì mỗi lần fleet là rò một file descriptor, và trên
	// Windows file log bị khoá tới khi tiến trình cha thoát.
	f.Close()
	return c.Process.Pid, nil
}
