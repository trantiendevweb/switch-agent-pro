package dash

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Tieu de trang phai KHOP voi ten tren thanh dieu huong.
//
// Bai nay sinh ra tu mot cho sot that: gop hai mat ba chieu thanh "Trung tam",
// doi ten o thanh dieu huong, doi ten o topbar, doi ten file, sua het bai kiem —
// va bo quen dung the <title>. Trang van ghi "Van phong" tren tab trinh duyet.
//
// Kieu sot nay khong bai kiem nao cu bat duoc, vi <title> khong dinh gi toi
// chuc nang: trang chay dung, nut bam dung, chi la ten tren tab noi mot dang va
// ten trong trang noi mot dang. Nguoi dung mo bon tab thi doc dung cai ten sai.
var tenMat = map[string]string{
	"index.html":     "Dashboard",
	"flow.html":      "Workflow",
	"hoi-thoai.html": "Hội thoại",
	"trung-tam.html": "Trung tâm",
}

var reTieuDe = regexp.MustCompile(`(?is)<title>\s*([^<]*?)\s*</title>`)

func TestTieuDeTrangKhopTenTrongThanhDieuHuong(t *testing.T) {
	for f, ten := range tenMat {
		b, err := os.ReadFile(filepath.Join("web", f))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		s := string(b)
		m := reTieuDe.FindStringSubmatch(s)
		if m == nil {
			t.Errorf("%s khong co the <title>", f)
			continue
		}
		if !strings.Contains(m[1], ten) {
			t.Errorf("%s: <title> ghi %q nhung mat nay ten la %q — mo nhieu tab thi nguoi "+
				"dung doc dung cai ten tren tab, khong phai ten trong trang", f, m[1], ten)
		}
		// Va khong duoc con ten CU cua hai mat da gop.
		for _, cu := range []string{"Văn phòng", "3D"} {
			if strings.Contains(m[1], cu) {
				t.Errorf("%s: <title> con ten cu %q — hai mat do da gop thanh \"Trung tâm\" 20/08",
					f, cu)
			}
		}
	}
}

// Va khong trang nao duoc con tro toi hai trang da xoa. Mot the <a> tro toi
// trang khong ton tai la mot nut bam vao ra 404 — nguoi dung khong biet la no
// da bi gop, chi biet la no hong.
func TestKhongConTroToiTrangDaXoa(t *testing.T) {
	for f := range tenMat {
		b, err := os.ReadFile(filepath.Join("web", f))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for _, cu := range []string{"3d.html", "vanphong.html"} {
			if strings.Contains(s, cu) {
				t.Errorf("%s con tro toi %q — trang do da bi xoa khi gop thanh trung-tam.html", f, cu)
			}
		}
	}
}
