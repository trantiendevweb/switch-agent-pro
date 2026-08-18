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
	branch := nhanhTrong(repoRoot, "sagent/"+name)
	// -B: có nhánh cũ thì ghi đè, không thì tạo mới — chạy lại fleet không kẹt.
	// An toàn được là NHỜ nhanhTrong() ở trên đã đảm bảo nhánh này không giữ
	// công việc nào chưa trộn.
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

// FindAll liệt kê mọi worktree của một tài khoản, theo tên thật trên đĩa.
//
// Cố ý QUÉT THƯ MỤC chứ không đoán "phu-1, phu-2, …": sau một lần clean dở
// (giữ lại bản còn thay đổi chưa commit) thì số thứ tự bị khuyết, kiểu đoán sẽ
// dừng ở lỗ hổng đầu tiên và bỏ sót phần còn lại. Đã dính đúng lỗi này.
func FindAll(repoRoot, account string) []string {
	parent := filepath.Dir(dirFor(repoRoot, "x"))
	entries, err := os.ReadDir(parent)
	if err != nil {
		return nil
	}
	prefix := account + "-"
	var out []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
			out = append(out, filepath.Join(parent, e.Name()))
		}
	}
	return out
}

// IsDirty cho biết worktree còn thay đổi CHƯA COMMIT hay không.
//
// Đây là lá chắn cho nguyên tắc "xoá an toàn": `git worktree remove --force`
// nuốt luôn việc agent làm dở mà không hỏi. Phải kiểm trước khi gỡ.
func IsDirty(dir string) bool {
	out, err := run(dir, "status", "--porcelain")
	if err != nil {
		return true // không kiểm được thì coi như bẩn, thà giữ lại
	}
	return out != ""
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

// nhanhTrong trả về một tên nhánh KHÔNG GIỮ CÔNG VIỆC CHƯA TRỘN.
//
// Có vì một lỗi mất dữ liệu thật, đo ngày 18/08: `worktree add -B` ĐẶT LẠI nhánh,
// nên mỗi lượt fleet mới xoá sạch commit của lượt trước trên cùng tài khoản.
// Lần chạy #21, agent claude:tns commit 99 dòng (có 87 dòng test) lên
// `sagent/tns-1`; lượt fleet sau đó cùng tài khoản làm commit đó thành MỒ CÔI —
// còn trong kho nhưng không thuộc nhánh nào, và `git log main..sagent/tns-1`
// hiện trống trơn như chưa ai làm gì.
//
// Cách sửa: nếu nhánh cũ còn commit chưa có ở nhánh nền thì ĐỔI TÊN nhánh mới
// (thêm hậu tố), giữ nguyên việc cũ. Thà có vài nhánh thừa còn hơn mất việc —
// nhánh thừa thì dọn được, việc mất thì không.
func nhanhTrong(repoRoot, goc string) string {
	nen, err := NhanhMacDinh(repoRoot)
	if err != nil {
		nen = "main"
	}
	ten := goc
	for i := 2; i < 100; i++ {
		if _, err := run(repoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+ten); err != nil {
			return ten // chưa có nhánh này
		}
		out, err := run(repoRoot, "rev-list", "--count", nen+".."+ten)
		if err != nil || strings.TrimSpace(out) == "0" {
			return ten // có nhưng rỗng so với nền -> ghi đè được
		}
		ten = fmt.Sprintf("%s-%d", goc, i)
	}
	return ten
}
