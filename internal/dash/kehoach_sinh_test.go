package dash

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/tools/sinhkehoach"
)

// web/docs/master-plan.html là trang được SINH RA từ web/docs/MASTER-PLAN.md,
// không phải file người ta sửa tay.
//
// Vì sao phải có test này: trước đây trang đó dựng tay, và không ai đo được nó
// còn nói đúng bản .md hay không. Hậu quả để lại dấu ngay trong mã kiểm thử —
// uxui_test.go phải MIỄN TRỪ hẳn trang này khỏi mọi luật giao diện, kèm ghi chú
// "chưa có khâu sinh lại — dọn tay sẽ lệch với bản .md". Đo ngày 21/08/2026:
// riêng bản .md nhúng đã lệch 4 khối lớn so với docs/MASTER-PLAN.md ở gốc repo,
// và trang .html thì lệch tiếp với chính bản .md nằm cạnh nó.
//
// Nay lệch là ĐỎ. Sửa: chạy `go run ./tools/sinhkehoach/cmd/sinhkehoach`.
func TestTrangKeHoachKhopBanSinh(t *testing.T) {
	nguon := filepath.Join("web", "docs", sinhkehoach.TenNguon)
	dich := filepath.Join("web", "docs", sinhkehoach.TenDich)

	md, err := os.ReadFile(nguon)
	if err != nil {
		t.Fatal(err)
	}
	co, err := os.ReadFile(dich)
	if err != nil {
		t.Fatal(err)
	}
	muon := sinhkehoach.Trang(md, sinhkehoach.TieuDe)
	if bytes.Equal(co, muon) {
		return
	}

	a := strings.Split(string(co), "\n")
	b := strings.Split(string(muon), "\n")
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] == b[i] {
			continue
		}
		t.Fatalf("%s đã lệch với bản sinh từ %s — sửa bằng "+
			"`go run ./tools/sinhkehoach/cmd/sinhkehoach`, đừng sửa tay file .html.\n"+
			"  dòng %d\n  đang có: %s\n  bản sinh: %s",
			dich, sinhkehoach.TenNguon, i+1, catNgan(a[i]), catNgan(b[i]))
	}
	t.Fatalf("%s đã lệch với bản sinh từ %s (%d dòng vs %d dòng) — sửa bằng "+
		"`go run ./tools/sinhkehoach/cmd/sinhkehoach`.",
		dich, sinhkehoach.TenNguon, len(a), len(b))
}

// Bộ sinh phải cho ra CÙNG byte với cùng đầu vào. Nếu nó lỡ nhét mốc thời gian
// hay duyệt map theo thứ tự ngẫu nhiên thì test trên sẽ đỏ tuỳ hứng, và cách
// người ta chữa một test đỏ tuỳ hứng luôn là tắt nó đi.
func TestBanSinhKeHoachOnDinh(t *testing.T) {
	md, err := os.ReadFile(filepath.Join("web", "docs", sinhkehoach.TenNguon))
	if err != nil {
		t.Fatal(err)
	}
	mot := sinhkehoach.Trang(md, sinhkehoach.TieuDe)
	for i := 0; i < 3; i++ {
		if !bytes.Equal(mot, sinhkehoach.Trang(md, sinhkehoach.TieuDe)) {
			t.Fatalf("lần sinh thứ %d ra byte khác lần đầu — đầu ra không tất định", i+2)
		}
	}
}

// Một bộ sinh hỏng vẫn ghi ra file HTML hợp lệ (chỉ có cái khung, rỗng ruột), và
// test so khớp ở trên vẫn XANH vì hai bên cùng rỗng. Nên phải đo thêm rằng nội
// dung .md thật sự chảy sang .html: đếm ô việc là phép đếm rẻ và không nói dối.
func TestTrangKeHoachChoDuOViec(t *testing.T) {
	md, err := os.ReadFile(filepath.Join("web", "docs", sinhkehoach.TenNguon))
	if err != nil {
		t.Fatal(err)
	}
	var muon int
	trongRao := false
	for _, d := range strings.Split(strings.ReplaceAll(string(md), "\r\n", "\n"), "\n") {
		dd := strings.TrimSpace(d)
		if strings.HasPrefix(dd, "```") {
			trongRao = !trongRao
			continue
		}
		if trongRao {
			continue
		}
		if strings.HasPrefix(dd, "- [ ] ") || strings.HasPrefix(dd, "- [x] ") || strings.HasPrefix(dd, "- [X] ") {
			muon++
		}
	}
	if muon == 0 {
		t.Fatal("không thấy ô việc nào trong " + sinhkehoach.TenNguon + " — phép đếm này sẽ xanh giả")
	}
	co := strings.Count(string(sinhkehoach.Trang(md, sinhkehoach.TieuDe)), `class="task`)
	if co != muon {
		t.Errorf("bản sinh có %d ô việc nhưng %s có %d — bộ sinh đang nuốt hoặc nhân đôi mục",
			co, sinhkehoach.TenNguon, muon)
	}
}

func catNgan(s string) string {
	if len(s) > 160 {
		return s[:160] + "…"
	}
	return s
}
