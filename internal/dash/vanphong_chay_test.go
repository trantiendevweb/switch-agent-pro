package dash

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Muoi hai bai trong vanphong_test.go deu doc MA NGUON: co chuoi nay khong, ham
// kia co goi ham no khong. Chung bat duoc rat nhieu thu, nhung KHONG bat duoc
// mot loai: ma dung ca, doc ra thay hop ly ca, ma chay len thi nem loi o dong
// thu 400 va 399 dong dau da chay xong roi ca man hinh dung im.
//
// DAY LA CHUYEN DA XAY RA THAT: mat 2D tung chet tu dong 363 vi mot ham nhan
// `#wf-ten` trong khi ben trong no goi getElementById. Nut dung, form ham doi VA
// CA EventSource cung chet mot luc — tuc la 2D khong he realtime suot nhieu
// ngay — ma khong ai biet, vi trang van ve ra binh thuong.
//
// Nen bai nay CHAY THAT doan script do: DOM gia, THREE gia, du lieu flow gia,
// quay 60 khung hinh. Ngoai chuyen "khong nem loi", no con DO ba thu ma mat
// thuong moi nhin ra duoc, con toi thi khong co mat:
//
//   - khong nhan nao de len nhan nao sau khi chieu xuong man hinh;
//   - so duong giao viec dung bang so canh `needs` giua hai nhan vat khac nhau;
//   - noi that: khong mon nao xuyen vach, cam vao cho dung, hay cam vao mon khac.
func TestVanPhongChayThatVoiDomGia(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("khong co node — bo qua. Bai nay chi chay duoc o may co node.")
	}

	ma := layScriptTrong(docVanPhong(t))
	if len(ma) < 5000 {
		t.Fatalf("chi rut duoc %d ky tu script tu vanphong.html — gan nhu chac chan "+
			"la bo rut nham, khong phai trang bi rong", len(ma))
	}

	tam := filepath.Join(t.TempDir(), "vanphong.js")
	if err := os.WriteFile(tam, []byte(ma), 0o600); err != nil {
		t.Fatal(err)
	}
	ra, err := exec.Command(node, filepath.Join("testdata", "vanphong_harness.js"), tam).CombinedOutput()
	if err != nil {
		t.Fatalf("van phong hong khi chay that:\n%s", ra)
	}
	t.Logf("\n%s", ra)
}

// reScriptTrong bat moi khoi <script> KHONG co thuoc tinh src — tuc la ma viet
// trong trang, khong phai ba file vendor.
var reScriptTrong = regexp.MustCompile(`(?s)<script(?:\s[^>]*)?>(.*?)</script>`)

func layScriptTrong(html string) string {
	var b strings.Builder
	for _, m := range reScriptTrong.FindAllStringSubmatch(html, -1) {
		if strings.Contains(m[0][:strings.Index(m[0], ">")], "src=") {
			continue
		}
		b.WriteString(m[1])
		b.WriteString("\n")
	}
	return b.String()
}
