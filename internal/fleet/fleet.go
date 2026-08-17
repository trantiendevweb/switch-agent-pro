// Package fleet điều phối nhiều phiên agent chạy song song.
package fleet

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/trantiendevweb/switch-agent-pro/internal/profile"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
	"github.com/trantiendevweb/switch-agent-pro/internal/workspace"
)

// Opts là các lựa chọn khi bật một hạm đội.
type Opts struct {
	Copies   int
	Worktree bool // mỗi phiên một git worktree riêng
}

// FanOut chạy N phiên song song trên MỘT tài khoản.
//
// args là lệnh headless truyền cho CLI (ví dụ: -p "tóm tắt repo"). Phiên tương
// tác không chạy nền được vì cần bàn phím, nên fleet chỉ dành cho agent.
func FanOut(db *store.DB, a provider.Adapter, account string, o Opts, args []string) error {
	if o.Copies < 1 {
		o.Copies = 1
	}
	if len(args) == 0 {
		return fmt.Errorf(`thiếu lệnh headless sau "--".
  Ví dụ: sagent fleet %s:%s --copies %d -- -p "tóm tắt repo này"`, a.Name(), account, o.Copies)
	}

	// Chuẩn bị worktree TRƯỚC khi bật phiên nào: thà hỏng lúc chưa chạy gì còn
	// hơn bật được 2 phiên rồi mới chết ở phiên thứ 3.
	var repoRoot string
	if o.Worktree {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		root, ok := workspace.RepoRoot(wd)
		if !ok {
			return fmt.Errorf("--worktree cần một git repo, mà %s không phải", wd)
		}
		repoRoot = root
	}

	dirs, err := profile.Clone(a, account, o.Copies)
	if err != nil {
		return err
	}

	// Nói thẳng hai điều, không giấu.
	fmt.Printf("  ⚠ %d phiên trên MỘT tài khoản %s:%s — tiêu hạn mức gấp %d lần.\n",
		o.Copies, a.Name(), account, o.Copies)
	fmt.Printf("  ⚠ Token được chép ra %d chỗ; hành vi khi nhiều phiên cùng refresh CHƯA ĐO.\n", o.Copies)
	if o.Worktree {
		fmt.Printf("  · Mỗi phiên một git worktree riêng từ %s\n", repoRoot)
	} else {
		fmt.Printf("  ⚠ Cả %d phiên dùng CHUNG thư mục hiện tại — chúng có thể sửa đè file của nhau.\n", o.Copies)
		fmt.Printf("    Thêm --worktree để mỗi phiên có cây làm việc riêng.\n")
	}
	fmt.Println()

	started := 0
	for i, dir := range dirs {
		name := fmt.Sprintf("%s-%d", account, i+1)

		workDir := ""
		if o.Worktree {
			wt, err := workspace.Add(repoRoot, name)
			if err != nil {
				fmt.Printf("  ✗ phiên %d: %v\n", i+1, err)
				continue
			}
			workDir = wt
		}

		logPath := filepath.Join(dir, "fleet.log")
		pid, err := profile.StartDetached(a, dir, args, logPath, workDir)
		if err != nil {
			fmt.Printf("  ✗ phiên %d: %v\n", i+1, err)
			if workDir != "" {
				_ = workspace.Remove(repoRoot, workDir)
			}
			continue
		}
		id, err := db.AddSession(store.Session{
			Provider: a.Name(), Account: account, Clone: i + 1,
			Dir: dir, PID: pid, Log: logPath, Worktree: workDir,
		})
		if err != nil {
			fmt.Printf("  ! phiên %d chạy rồi (PID %d) nhưng không ghi được vào sổ: %v\n", i+1, pid, err)
			continue
		}
		if workDir != "" {
			fmt.Printf("  ✓ #%d  phiên %d  PID %-7d  nhánh sagent/%s\n", id, i+1, pid, name)
		} else {
			fmt.Printf("  ✓ #%d  phiên %d  PID %-7d  log: %s\n", id, i+1, pid, logPath)
		}
		started++
	}

	fmt.Printf("\n  Đã khởi chạy %d/%d phiên. Xem: sagent status  ·  Dừng: sagent stop all\n",
		started, len(dirs))
	return nil
}
