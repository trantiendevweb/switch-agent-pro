// Package fleet điều phối nhiều phiên agent chạy song song.
package fleet

import (
	"fmt"
	"path/filepath"

	"github.com/trantiendevweb/switch-agent-pro/internal/profile"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// FanOut chạy N phiên song song trên MỘT tài khoản.
//
// args là lệnh headless truyền cho CLI (ví dụ: -p "tóm tắt repo"). Phiên tương
// tác không chạy nền được vì cần bàn phím, nên fleet chỉ dành cho agent.
func FanOut(db *store.DB, a provider.Adapter, account string, copies int, args []string) error {
	if copies < 1 {
		copies = 1
	}
	if len(args) == 0 {
		return fmt.Errorf(`thiếu lệnh headless sau "--".
  Ví dụ: sagent fleet %s:%s --copies %d -- -p "tóm tắt repo này"`, a.Name(), account, copies)
	}

	dirs, err := profile.Clone(a, account, copies)
	if err != nil {
		return err
	}

	// Nói thẳng hai điều, không giấu.
	fmt.Printf("  ⚠ %d phiên trên MỘT tài khoản %s:%s — tiêu hạn mức gấp %d lần.\n",
		copies, a.Name(), account, copies)
	fmt.Printf("  ⚠ Token được chép ra %d chỗ; hành vi khi nhiều phiên cùng refresh CHƯA ĐO.\n", copies)
	fmt.Println()

	started := 0
	for i, dir := range dirs {
		logPath := filepath.Join(dir, "fleet.log")
		pid, err := profile.StartDetached(a, dir, args, logPath)
		if err != nil {
			fmt.Printf("  ✗ phiên %d: %v\n", i+1, err)
			continue
		}
		id, err := db.AddSession(store.Session{
			Provider: a.Name(), Account: account, Clone: i + 1,
			Dir: dir, PID: pid, Log: logPath,
		})
		if err != nil {
			fmt.Printf("  ! phiên %d chạy rồi (PID %d) nhưng không ghi được vào sổ: %v\n", i+1, pid, err)
			continue
		}
		fmt.Printf("  ✓ #%d  phiên %d  PID %-7d  log: %s\n", id, i+1, pid, logPath)
		started++
	}

	fmt.Printf("\n  Đã khởi chạy %d/%d phiên. Xem: sagent status  ·  Dừng: sagent stop all\n",
		started, len(dirs))
	return nil
}
