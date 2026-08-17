package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// `sagent db` — xem, sao lưu, khôi phục state.db (Pha 7).
//
// Cố ý KHÔNG đi qua api: đây là thao tác trên chính cái file mà api mở. Mở api
// để rồi ghi đè file bên dưới nó là tự dẫm chân. Lệnh này chạm thẳng vào đĩa và
// đòi người dùng dừng dash trước.
func cmdDB(args []string) {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "", "info":
		dbInfo()
	case "backup", "sao-luu":
		dbBackup(rest(args))
	case "restore", "khoi-phuc":
		dbRestore(rest(args))
	default:
		fail(fmt.Errorf("không biết `db %s` — dùng: info | backup [file] | restore <file>", sub))
	}
}

func dbInfo() {
	p := store.Path()
	fmt.Println()
	fmt.Println("  File:", p)
	fi, err := os.Stat(p)
	if err != nil {
		fmt.Println("  Chưa có — sẽ được tạo ở lần chạy đầu tiên.")
		fmt.Println()
		return
	}
	fmt.Printf("  Kích thước: %.1f KB · sửa lần cuối %s\n",
		float64(fi.Size())/1024, fi.ModTime().Format("2006-01-02 15:04"))

	d, err := store.Open()
	if err != nil {
		// Lỗi ở đây thường là hạ cấp binary — thông điệp của store đã nói rõ
		// phải làm gì, đừng nuốt nó.
		fail(err)
	}
	defer d.Close()
	v, err := d.SchemaVersion()
	if err != nil {
		fail(err)
	}
	fmt.Printf("  Schema: v%d (bản sagent này biết tới v%d)\n", v, store.LatestSchema())

	// Bản sao lưu tự động do lần nâng schema trước để lại.
	if m, _ := filepath.Glob(p + ".bak-v*"); len(m) > 0 {
		fmt.Println("  Bản sao lưu tự động:")
		for _, f := range m {
			s, _ := os.Stat(f)
			fmt.Printf("    · %s (%.1f KB)\n", filepath.Base(f), float64(s.Size())/1024)
		}
	}
	fmt.Println()
	fmt.Println("  Sao lưu ngay:   sagent db backup")
	fmt.Println("  Khôi phục:      sagent db restore <file>   (dừng dash trước)")
	fmt.Println()
}

func dbBackup(args []string) {
	dst := ""
	if len(args) > 0 {
		dst = args[0]
	}
	if dst == "" {
		dst = fmt.Sprintf("%s.bak-%s", store.Path(), time.Now().Format("20060102-150405"))
	}
	d, err := store.Open()
	if err != nil {
		fail(err)
	}
	defer d.Close()
	if err := d.Snapshot(dst); err != nil {
		fail(err)
	}
	fi, _ := os.Stat(dst)
	fmt.Printf("\n  ✓ đã sao lưu → %s (%.1f KB)\n\n", dst, float64(fi.Size())/1024)
}

func dbRestore(args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu file: sagent db restore <file>"))
	}
	src := args[0]
	dst := store.Path()

	// Khôi phục là ghi đè. Ghi đè trong lúc tiến trình khác đang mở file thì
	// SQLite không chống được — nó không biết có ai vừa thay file dưới chân nó.
	if err := store.InUse(dst); err != nil {
		fail(fmt.Errorf("%w\n     dừng dash và mọi phiên sagent rồi chạy lại", err))
	}

	bak, err := store.Restore(src, dst)
	if bak != "" {
		fmt.Printf("\n  bản trước khi khôi phục đã cứu ở: %s\n", bak)
	}
	if err != nil {
		fail(err)
	}
	fmt.Printf("  ✓ đã khôi phục %s → %s\n\n", src, dst)
}
