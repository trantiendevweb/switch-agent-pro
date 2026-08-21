// Lệnh sinh lại trang kế hoạch nhúng của dashboard từ file .md cạnh nó.
//
// Chạy:
//
//	go run ./tools/sinhkehoach/cmd/sinhkehoach        # sinh lại vào chỗ mặc định
//	go generate ./tools/...                            # y hệt, qua chỉ thị go:generate
//	go run ./tools/sinhkehoach/cmd/sinhkehoach -kiem   # chỉ KIỂM, không ghi
//
// Cờ -kiem để dùng trong CI/hook: trả mã thoát 1 khi trang đang có lệch với bản
// sinh, mà không đụng vào cây làm việc.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/trantiendevweb/switch-agent-pro/tools/sinhkehoach"
)

func main() {
	if err := chay(); err != nil {
		fmt.Fprintln(os.Stderr, "sinhkehoach:", err)
		os.Exit(1)
	}
}

func chay() error {
	goc, err := gocRepo()
	if err != nil {
		return err
	}
	mac := filepath.Join(goc, filepath.FromSlash(sinhkehoach.ThuMucNhung))

	nguon := flag.String("nguon", filepath.Join(mac, sinhkehoach.TenNguon), "file Markdown nguồn")
	dich := flag.String("dich", filepath.Join(mac, sinhkehoach.TenDich), "file HTML sinh ra")
	tieuDe := flag.String("tieude", sinhkehoach.TieuDe, "tiêu đề trang")
	kiem := flag.Bool("kiem", false, "chỉ kiểm xem file đích có khớp bản sinh không, không ghi")
	flag.Parse()

	md, err := os.ReadFile(*nguon)
	if err != nil {
		return err
	}
	ra := sinhkehoach.Trang(md, *tieuDe)

	if *kiem {
		co, err := os.ReadFile(*dich)
		if err != nil {
			return err
		}
		if !bytes.Equal(co, ra) {
			return fmt.Errorf("%s đã lệch với bản sinh từ %s — chạy lại lệnh này (bỏ -kiem)",
				*dich, filepath.Base(*nguon))
		}
		fmt.Printf("  khớp: %s == bản sinh từ %s\n", *dich, filepath.Base(*nguon))
		return nil
	}

	// Ghi qua file tạm rồi rename: nửa chừng chết thì trang cũ còn nguyên chứ
	// không để lại một file HTML cụt mà server vẫn phục vụ.
	tam := *dich + ".tam"
	if err := os.WriteFile(tam, ra, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tam, *dich); err != nil {
		os.Remove(tam)
		return err
	}
	fmt.Printf("  đã sinh %s -> %s (%d KB)\n", *nguon, *dich, len(ra)/1024)
	return nil
}

// gocRepo đi ngược lên từ thư mục hiện tại tìm go.mod. Nhờ vậy lệnh chạy đúng
// dù gọi từ gốc repo (go run ./tools/...) hay từ thư mục gói (go generate).
func gocRepo() (string, error) {
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		cha := filepath.Dir(d)
		if cha == d {
			return "", fmt.Errorf("không tìm thấy go.mod từ %s trở lên — hãy chạy trong cây mã nguồn", d)
		}
		d = cha
	}
}
