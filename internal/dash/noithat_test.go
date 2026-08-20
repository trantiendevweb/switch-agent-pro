package dash

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Bo noi that .glb (Kenney Furniture Kit, CC0) nam trong binary, giong
// RobotExpressive.glb. Ba cach hong da luong truoc, moi cach mot bai kiem:
//
//   - ten mau trong MAU_NOI_THAT khong tro toi file co that -> phong trong tron
//     va KHONG co dong log nao, vi ma da co duong lui im lang sang ban khoi hop;
//   - file co nhung khong phai .glb that (tai ve loi, git-lfs con tro) -> canh
//     nap ra loi giua chung;
//   - file .glb nam trong binary ma khong trang nao dung -> can nang binary
//     bang mot thu chet.
//
// TestDuongVendorTroDungFile o offline_asset_test.go KHONG gac duoc cho nay: no
// cat lay doan sau dau "/" cuoi cung, ma duong o day dung noi chuoi
// ('vendor/noithat/' + ten + '.glb') nen doan do rong, va os.Stat("web/vendor/")
// tra ve thu muc — xanh ma khong kiem gi ca.

var reMauNoiThat = regexp.MustCompile(`(?s)const MAU_NOI_THAT\s*=\s*\[(.*?)\]`)

func mauNoiThat(t *testing.T) []string {
	t.Helper()
	m := reMauNoiThat.FindStringSubmatch(docTrungTam(t))
	if m == nil {
		t.Fatal("vanphong.html khong con mang MAU_NOI_THAT — bo noi that phai khai o MOT cho")
	}
	var ra []string
	for _, x := range strings.Split(m[1], ",") {
		x = strings.Trim(strings.TrimSpace(x), `'"`)
		if x != "" {
			ra = append(ra, x)
		}
	}
	return ra
}

func TestMoiMauNoiThatTroDungMotFileGLBThat(t *testing.T) {
	ds := mauNoiThat(t)
	if len(ds) < 8 {
		t.Fatalf("chi khai %d mau noi that — gan nhu chac chan la bo doc nham, "+
			"khong phai bo noi that bi teo lai", len(ds))
	}
	for _, ten := range ds {
		p := filepath.Join("web", "vendor", "noithat", ten+".glb")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("mau %q khai trong MAU_NOI_THAT nhung khong co file: %v", ten, err)
			continue
		}
		// Bon byte dau cua mot .glb LA magic "glTF". Mot file HTML bao loi tai
		// ve nham cung co kich thuoc, cung nam dung cho, va chi lo ra luc chay.
		if len(b) < 4 || string(b[:4]) != "glTF" {
			t.Errorf("%s khong phai .glb that (bon byte dau = %q, phai la \"glTF\")",
				p, string(b[:min(4, len(b))]))
			continue
		}
		if len(b) < 1024 {
			t.Errorf("%s chi %d byte — qua nho de la mot mo hinh that", p, len(b))
		}
	}
}

// File nam trong thu muc noi that ma khong trang nao dung = can nang binary
// bang mot thu chet. Binary nay la MOT file duy nhat nguoi dung tai ve.
func TestKhongCoMauNoiThatThuaTrongBinary(t *testing.T) {
	dung := map[string]bool{}
	for _, ten := range mauNoiThat(t) {
		dung[ten+".glb"] = true
	}
	ds, err := os.ReadDir(filepath.Join("web", "vendor", "noithat"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ds {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".glb") {
			continue
		}
		if !dung[e.Name()] {
			t.Errorf("web/vendor/noithat/%s nam trong binary ma khong trang nao dung — "+
				"xoa di, hoac them vao MAU_NOI_THAT neu dinh dung", e.Name())
		}
	}
}

// CC0 khong BAT BUOC ghi cong, nhung mang nguyen van giay phep di theo thi ai
// doc ma nguon cung tra loi duoc ngay cau "cho nay lay o dau, dung duoc khong".
func TestGiuGiayPhepBoNoiThat(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("web", "vendor", "noithat", "GIAY-PHEP.txt"))
	if err != nil {
		t.Fatalf("thieu nguyen van giay phep cua bo noi that: %v", err)
	}
	for _, c := range []string{"CC0", "Kenney"} {
		if !strings.Contains(string(b), c) {
			t.Errorf("giay phep khong nhac toi %q — co dung file khong?", c)
		}
	}
	// Va ma nguon phai noi ro no lay o dau.
	s := docTrungTam(t)
	for _, c := range []string{"Kenney", "CC0"} {
		if !strings.Contains(s, c) {
			t.Errorf("vanphong.html khong ghi nguon/giay phep bo noi that (%q)", c)
		}
	}
}
