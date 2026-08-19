package dash

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Dashboard la asset NHUNG THANG TRONG BINARY va chay OFFLINE. Mot the
// <script src="https://cdnjs..."> trong trang nhin thi vo hai — may nao co
// mang van dep — nhung dung may that: cdnjs bi chan, three.js khong ve, man 3D
// TRANG TRON. Font Google Fonts cung vay, chi im hon: chu roi ve font he dieu
// hanh, khong ai bao loi.
//
// Do 19/08: co 3 cho ket noi ra ngoai (3d.html hai the script CDN, docs/index.html
// va docs/master-plan.html moi trang mot <link> Google Fonts). Nay quet may.
//
// Tha the <a>: do la duong NGUOI BAM, bam xong moi di, khong lien quan toi viec
// trang co ve duoc khi khong co mang hay khong.

// hut moi src= / href= trong HTML, va moi url(...) trong CSS nhung.
var (
	reThuoc = regexp.MustCompile(`(?i)(src|href)\s*=\s*("[^"]*"|'[^']*')`)
	reURL   = regexp.MustCompile(`(?i)url\(\s*("[^"]*"|'[^']*'|[^)'"]*)\s*\)`)
	reNgoai = regexp.MustCompile(`(?i)^(https?:)?//`)
)

func TestKhongTaiAssetTuNgoai(t *testing.T) {
	for _, f := range fileHTML(t) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for _, m := range reThuoc.FindAllStringSubmatchIndex(s, -1) {
			gia := boNhay(s[m[4]:m[5]])
			if !reNgoai.MatchString(gia) {
				continue
			}
			if the := theBaoQuanh(s, m[0]); the == "a" {
				continue // duong nguoi bam, khong phai asset de ve trang
			}
			t.Errorf("%s: %s=%q tro ra ngoai — trang phai ve duoc khi may khong co mang.\n"+
				"  Tai file ve internal/dash/web/vendor/ roi tro duong noi bo.",
				ten(f), s[m[2]:m[3]], gia)
		}
		for _, m := range reURL.FindAllStringSubmatchIndex(s, -1) {
			gia := boNhay(s[m[2]:m[3]])
			if reNgoai.MatchString(gia) {
				t.Errorf("%s: url(%q) trong CSS tro ra ngoai — font/anh cung phai vendor.", ten(f), gia)
			}
		}
		if strings.Contains(s, "@import") && strings.Contains(s, "fonts.googleapis") {
			t.Errorf("%s: con @import font tu Google Fonts", ten(f))
		}
	}
}

// Rang buoc nhung: three.js chi MOT file core. OrbitControls la addon — them
// mot file phai tai roi, va chinh no la cai tung nam o cdn.jsdelivr.
func TestKhongDungAddonThreeJS(t *testing.T) {
	for _, f := range fileHTML(t) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for _, addon := range []string{"OrbitControls", "EffectComposer", "UnrealBloomPass"} {
			// bat cho GOI addon, khong bat chu trong comment giai thich vi sao khong dung
			if strings.Contains(s, "THREE."+addon) || strings.Contains(s, "new "+addon+"(") {
				t.Errorf("%s: con goi addon three.js %s — ban nhung chi giu mot file core, "+
					"camera orbit viet tay", ten(f), addon)
			}
		}
	}
}

// Tro duong noi bo ma file khong co (hoac rong) thi cung trang tron y het khi
// tro ra CDN — chi khac la khong ai do loi cho mang. Nen kiem ca noi dung.
func TestAssetVendorCoThat(t *testing.T) {
	can := map[string]string{
		"three.min.js":                  "THREE",
		"inter-variable.woff2":          "wOF2",
		"space-grotesk-variable.woff2":  "wOF2",
		"jetbrains-mono-variable.woff2": "wOF2",
	}
	for f, dau := range can {
		p := filepath.Join("web", "vendor", f)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("thieu asset vendor %s: %v", f, err)
			continue
		}
		if len(b) < 4096 {
			t.Errorf("%s chi %d byte — file rong/hong thi trang van vo, dung bao xong voi",
				f, len(b))
			continue
		}
		if !strings.Contains(string(b[:min(len(b), 4096)]), dau) {
			t.Errorf("%s khong co dau nhan %q o dau file — khong phai asset that", f, dau)
		}
	}
	// three.js phai dung r128: 3d.html viet theo API ban do (outputEncoding,
	// ACESFilmicToneMapping, Geometry cu). Doi ban ma khong doi code la vo.
	// Ban minify ghi so ban thanh `const e="128"` roi `THREE.REVISION=e`.
	b, err := os.ReadFile(filepath.Join("web", "vendor", "three.min.js"))
	if err == nil {
		s := string(b)
		if !regexp.MustCompile(`const \w+="128"`).MatchString(s) || !strings.Contains(s, "REVISION=") {
			t.Error("web/vendor/three.min.js khong phai r128 — 3d.html viet theo API r128")
		}
	}
}

// Moi duong vendor/... viet trong HTML phai tro dung mot file co that.
func TestDuongVendorTroDungFile(t *testing.T) {
	reVendor := regexp.MustCompile(`[\w./-]*vendor/[\w.\[\]-]+`)
	for _, f := range fileHTML(t) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range reVendor.FindAllString(string(b), -1) {
			tenFile := d[strings.LastIndex(d, "/")+1:]
			if _, err := os.Stat(filepath.Join("web", "vendor", tenFile)); err != nil {
				t.Errorf("%s tro toi %q nhung file khong co: %v", filepath.Base(f), d, err)
			}
		}
	}
}

// fileHTML: MOI trang trong web/, khong tha trang nao. Danh sach mien tru cua
// uxui_test.go la chuyen kieu chu; con ket noi ra ngoai thi trang nao cung vo.
func fileHTML(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir("web", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".html") {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("khong thay file HTML nao — test nay se xanh gia")
	}
	return out
}

func boNhay(s string) string {
	return strings.Trim(strings.TrimSpace(s), `"'`)
}

func ten(p string) string {
	return filepath.ToSlash(p)
}

// theBaoQuanh: ten the mo gan nhat truoc vi tri i, viet thuong.
func theBaoQuanh(s string, i int) string {
	j := strings.LastIndexByte(s[:i], '<')
	if j < 0 {
		return ""
	}
	k := j + 1
	for k < len(s) && (s[k] >= 'a' && s[k] <= 'z' || s[k] >= 'A' && s[k] <= 'Z' || s[k] >= '0' && s[k] <= '9') {
		k++
	}
	return strings.ToLower(s[j+1 : k])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Trang /docs/ co y KHONG doi dang nhap. Font cua no cung phai the: bat dang
// nhap o /vendor/ thi khach doc ke hoach se thay chu roi ve font he dieu hanh,
// im lang, khong ai bao loi. Nen /vendor/ mo cong khai — day la asset tinh.
func TestVendorPhucVuCongKhai(t *testing.T) {
	s := newTestServer(t)
	for _, d := range []string{
		"/vendor/three.min.js",
		"/vendor/inter-variable.woff2",
		"/vendor/space-grotesk-variable.woff2",
		"/vendor/jetbrains-mono-variable.woff2",
	} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req("GET", d))
		if w.Code != http.StatusOK {
			t.Errorf("GET %s: ma %d, muon 200 (chua dang nhap van phai lay duoc)", d, w.Code)
			continue
		}
		if n := w.Body.Len(); n < 4096 {
			t.Errorf("GET %s: chi %d byte — khong phai asset that", d, n)
		}
	}
}
