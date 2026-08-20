package dash

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Man van phong la ~1000 dong JavaScript trong mot file HTML. Go khong doc duoc
// no, `go vet` khong doc duoc no, va mot loi luc chay o dong 400 thi 399 dong
// dau van chay binh thuong roi ca man hinh dung im.
//
// DAY LA CHUYEN DA XAY RA THAT: mat 2D tung chet tu dong 363 vi mot ham nhan
// `#wf-ten` trong khi no goi getElementById. Nut dung, form ham doi VA CA
// EventSource cung chet mot luc — tuc la 2D khong he realtime — ma khong ai
// biet, vi trang van ve ra binh thuong.
//
// Nen bai kiem tra nay CHAY THAT doan script do: DOM gia, THREE gia, du lieu
// flow gia, quay 60 khung hinh. Nem loi o bat cu dau la do.
func TestVanPhongChayThatVoiDomGia(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("khong co node — bo qua. Bai nay chi chay duoc o may co node.")
	}

	b, err := os.ReadFile("web/vanphong.html")
	if err != nil {
		t.Fatal(err)
	}
	ma := layScriptTrong(string(b))
	if len(ma) < 5000 {
		t.Fatalf("chi rut duoc %d ky tu script tu vanphong.html — "+
			"gan nhu chac chan la bo rut nham, khong phai trang bi rong", len(ma))
	}

	tam := filepath.Join(t.TempDir(), "vanphong.js")
	if err := os.WriteFile(tam, []byte(ma), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, filepath.Join("testdata", "vanphong_harness.js"), tam)
	ra, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("van phong hong khi chay that:\n%s", ra)
	}
	t.Logf("\n%s", ra)
}

// reScriptTrong bat moi khoi <script> KHONG co thuoc tinh src — tuc la ma nguon
// viet trong trang, khong phai ba file vendor.
var reScriptTrong = regexp.MustCompile(`(?s)<script(?:\s[^>]*)?>(.*?)</script>`)

func layScriptTrong(html string) string {
	var b strings.Builder
	for _, m := range reScriptTrong.FindAllStringSubmatch(html, -1) {
		// Bo qua the co src: noi dung cua chung rong, nhung bo qua cho ro y.
		if strings.Contains(m[0][:strings.Index(m[0], ">")], "src=") {
			continue
		}
		b.WriteString(m[1])
		b.WriteString("\n")
	}
	return b.String()
}
