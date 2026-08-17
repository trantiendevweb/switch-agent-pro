// Command giayphep sinh và kiểm thông báo giấy phép của các phụ thuộc.
//
// Vì sao cần một công cụ chứ không chép tay: `docs/OPEN_SOURCE_LEDGER.md` viết
// tay đã TRÔI — đo lúc viết file này thấy nó kê `github.com/google/uuid` (có
// trong go.mod nhưng KHÔNG được liên kết vào binary) và vẫn xếp
// `golang.org/x/sys` vào bảng gián tiếp sau khi nó đã thành trực tiếp. Một sổ
// giấy phép sai thì tệ hơn không có sổ: nó tạo cảm giác đã kiểm.
//
// Nguồn sự thật là `go list -deps ./cmd/sagent` — tức những module THẬT SỰ đi
// vào binary, không phải mọi thứ có mặt trong go.mod.
//
//	go run ./tools/giayphep         # sinh THONG-BAO-GIAY-PHEP.txt
//	go run ./tools/giayphep -kiem   # CI: đỏ nếu file đã lệch, hoặc thiếu giấy phép
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const dichDen = "THONG-BAO-GIAY-PHEP.txt"

// Tên file giấy phép thường gặp. Không đoán theo nội dung — chỉ nhận file có tên
// rõ ràng, thiếu thì báo lỗi để người ta xử lý bằng tay.
var tenGiayPhep = []string{
	"LICENSE", "LICENSE.txt", "LICENSE.md",
	"LICENCE", "LICENCE.txt",
	"COPYING", "COPYING.txt",
	"LICENSE-BSD", "LICENSE-MIT",
}

type modun struct {
	Path, Version, Dir string
}

func main() {
	kiem := flag.Bool("kiem", false, "chỉ kiểm, không ghi (dùng trong CI)")
	flag.Parse()

	mods, err := lietKe()
	if err != nil {
		loi(err)
	}
	noiDung, thieu := dungVanBan(mods)
	if len(thieu) > 0 {
		loi(fmt.Errorf("không tìm thấy file giấy phép cho: %s\n"+
			"  → thêm bằng tay vào %s và ghi rõ nguồn, đừng bỏ qua",
			strings.Join(thieu, ", "), dichDen))
	}

	if !*kiem {
		if err := os.WriteFile(dichDen, []byte(noiDung), 0o644); err != nil {
			loi(err)
		}
		fmt.Printf("  ✓ %s — %d phụ thuộc\n", dichDen, len(mods))
		return
	}

	cu, err := os.ReadFile(dichDen)
	if err != nil {
		loi(fmt.Errorf("chưa có %s — chạy: go run ./tools/giayphep", dichDen))
	}
	if !bytes.Equal(bytes.ReplaceAll(cu, []byte("\r\n"), []byte("\n")), []byte(noiDung)) {
		loi(fmt.Errorf("%s đã lệch với phụ thuộc thật — chạy lại: go run ./tools/giayphep", dichDen))
	}
	fmt.Printf("  ✓ %s khớp %d phụ thuộc\n", dichDen, len(mods))
}

// lietKe hỏi go toolchain những module thật sự được liên kết vào binary.
func lietKe() ([]modun, error) {
	out, err := exec.Command("go", "list", "-deps",
		"-f", "{{if .Module}}{{.Module.Path}}\t{{.Module.Version}}\t{{.Module.Dir}}{{end}}",
		"./cmd/sagent").Output()
	if err != nil {
		return nil, fmt.Errorf("go list hỏng: %w", err)
	}
	seen := map[string]modun{}
	for _, d := range strings.Split(string(out), "\n") {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		f := strings.Split(d, "\t")
		if len(f) != 3 || f[1] == "" {
			continue // module chính (không có version) — không phải phụ thuộc
		}
		seen[f[0]] = modun{f[0], f[1], f[2]}
	}
	var mods []modun
	for _, m := range seen {
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].Path < mods[j].Path })
	return mods, nil
}

func dungVanBan(mods []modun) (string, []string) {
	var b strings.Builder
	var thieu []string
	b.WriteString("THÔNG BÁO GIẤY PHÉP — switch-agent-pro\n")
	b.WriteString(strings.Repeat("=", 72) + "\n\n")
	b.WriteString("File này SINH TỰ ĐỘNG từ `go list -deps ./cmd/sagent`, tức những module\n")
	b.WriteString("thật sự đi vào binary. Đừng sửa tay — chạy: go run ./tools/giayphep\n\n")
	b.WriteString("Bản thân switch-agent-pro phát hành theo giấy phép MIT (xem LICENSE).\n")
	b.WriteString("Các thư viện dưới đây giữ nguyên bản quyền của tác giả gốc; giấy phép của\n")
	b.WriteString("chúng (MIT / BSD) đòi giữ lại thông báo bản quyền khi phát hành bản build,\n")
	b.WriteString("và đó là lý do file này tồn tại.\n\n")

	b.WriteString("Danh sách:\n")
	for _, m := range mods {
		b.WriteString(fmt.Sprintf("  · %s %s\n", m.Path, m.Version))
	}
	b.WriteString("\n")

	for _, m := range mods {
		vb, ten := docGiayPhep(m.Dir)
		if vb == "" {
			thieu = append(thieu, m.Path)
			continue
		}
		b.WriteString(strings.Repeat("-", 72) + "\n")
		b.WriteString(fmt.Sprintf("%s %s  (%s)\n", m.Path, m.Version, ten))
		b.WriteString(strings.Repeat("-", 72) + "\n\n")
		b.WriteString(strings.ReplaceAll(strings.TrimRight(vb, "\r\n \t"), "\r\n", "\n"))
		b.WriteString("\n\n")
	}
	return b.String(), thieu
}

func docGiayPhep(dir string) (noiDung, ten string) {
	for _, n := range tenGiayPhep {
		p := filepath.Join(dir, n)
		if b, err := os.ReadFile(p); err == nil {
			return string(b), n
		}
	}
	return "", ""
}

func loi(err error) {
	fmt.Fprintf(os.Stderr, "  ✗ %v\n", err)
	os.Exit(1)
}
