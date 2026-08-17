// Package workspace cấp cho mỗi phiên một chỗ làm việc riêng.
//
// Vì sao cần: 4 agent chạy song song trên CÙNG một thư mục sẽ sửa đè file của
// nhau — agent A đang đọc file thì agent B ghi lại, kết quả không ai đoán được.
// Git worktree cho mỗi phiên một cây làm việc + một nhánh riêng, nhưng vẫn dùng
// chung một kho .git nên không tốn chỗ như clone cả repo.
package workspace

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// Root là nơi chứa worktree. Cố ý để NGOÀI repo: đặt trong repo thì `git status`
// của chính agent sẽ thấy chúng, và dễ bị commit nhầm.
func Root() string { return filepath.Join(paths.AccountsRoot(), ".worktrees") }

// RepoRoot tìm gốc git repo chứa dir. ok=false nếu không phải repo.
func RepoRoot(dir string) (string, bool) {
	out, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil || out == "" {
		return "", false
	}
	return filepath.Clean(out), true
}

// dirFor sinh đường dẫn worktree: gộp tên repo cho người đọc và hash đường dẫn
// đầy đủ để hai repo trùng tên không đụng nhau.
func dirFor(repoRoot, name string) string {
	sum := sha1.Sum([]byte(strings.ToLower(filepath.Clean(repoRoot))))
	key := filepath.Base(repoRoot) + "-" + hex.EncodeToString(sum[:4])
	return filepath.Join(Root(), key, name)
}

// Add tạo worktree + nhánh mới `sagent/<name>` từ HEAD hiện tại.
// Trả về đường dẫn worktree.
func Add(repoRoot, name string) (string, error) {
	dir := dirFor(repoRoot, name)
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", err
	}
	// Dọn tàn dư của lần trước (nếu có) để lệnh không chết vì "đã tồn tại".
	if _, err := os.Stat(dir); err == nil {
		_ = Remove(repoRoot, dir)
	}
	branch := "sagent/" + name
	// -B: có nhánh cũ thì ghi đè, không thì tạo mới — chạy lại fleet không kẹt.
	if _, err := run(repoRoot, "worktree", "add", "-B", branch, dir); err != nil {
		return "", fmt.Errorf("tạo worktree thất bại: %w", err)
	}
	return dir, nil
}

// Find trả về đường dẫn worktree theo tên nếu nó đang tồn tại.
func Find(repoRoot, name string) (string, bool) {
	dir := dirFor(repoRoot, name)
	if _, err := os.Stat(dir); err != nil {
		return "", false
	}
	return dir, true
}

// Remove gỡ worktree và dọn sổ của git. Không xoá nhánh: công việc của agent có
// thể còn nằm trong đó, xoá là mất.
func Remove(repoRoot, dir string) error {
	if _, err := run(repoRoot, "worktree", "remove", "--force", dir); err != nil {
		// Thư mục có thể đã bị xoá tay; prune để git khỏi giữ mục chết.
		_, _ = run(repoRoot, "worktree", "prune")
		if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
			return nil
		}
		return err
	}
	return nil
}

func run(dir string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.Output()
	return strings.TrimSpace(string(out)), err
}
