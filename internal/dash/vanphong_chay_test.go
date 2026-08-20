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

	// PHAI doc file harness bang Go, du chi de vut di: node moi la thu chay no,
	// ma go test khong nhin thay file mot tien trinh con mo. Khong doc thi Go
	// tuong bai kiem khong phu thuoc gi vao harness, va sau khi sua harness no
	// tra ket qua CACHED — tuc la sua bai kiem xong chay lai van thay cai cu.
	//
	// Toi vua dinh dung cai bay do that: them ba phep kiem cu bam vao harness,
	// chay lai, "ok (cached)", va suyt tuong la chung da chay.
	hn := filepath.Join("testdata", "vanphong_harness.js")
	if _, err := os.ReadFile(hn); err != nil {
		t.Fatalf("khong doc duoc %s: %v", hn, err)
	}
	ra, err := exec.Command(node, hn, tam).CombinedOutput()
	if err != nil {
		t.Fatalf("van phong hong khi chay that:\n%s", ra)
	}
	t.Logf("\n%s", ra)
}

// LUAT NGANG QUYEN (MASTER-PLAN §2c): mat nao cung phai DIEU KHIEN duoc, khong
// chi de ngam. Truoc 20/08 van phong la mat duy nhat bam vao khong ra gi.
//
// Bai chay-that o tren da chung minh cu bam MO duoc bang chi tiet. Bai nay gac
// them mot thu ma bai kia khong gac noi: bang do phai co duong DIEU KHIEN that.
// Mot lan bay lai giao dien co the giu nguyen bang chi tiet ma go mat cai nut —
// luc do bai chay-that van xanh, vi bang van mo.
func TestVanPhongDieuKhienDuocChuKhongChiDeNgam(t *testing.T) {
	s := maVanPhong(t)
	can := map[string]string{
		"/api/flow/cancel": "huy luot chay — hanh dong nang nhat, phai lam duoc tu day",
		"hoi-thoai.html":   "mo hoi thoai cua dung luot chay dang xem",
		"Raycaster":        "bam trung nhan vat trong canh 3D",
	}
	for d, vi := range can {
		if !strings.Contains(s, d) {
			t.Errorf("vanphong.html khong con %q (%s) — mat nay quay ve chi de ngam. "+
				"Neu CO Y bo thi sua bai kiem nay VA ghi ro vi sao.", d, vi)
		}
	}
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
