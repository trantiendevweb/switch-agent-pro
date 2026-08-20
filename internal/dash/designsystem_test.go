package dash

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Cac trang cua dashboard truoc day moi trang tu khai mot bang mau.
//
// Do ngay 19/08 truoc khi sua: index/flow/hoi-thoai dung bo
// --bg:#0F172A --run:#22C55E --warn:#F59E0B --limit:#EF4444 --api:#4285F4,
// con 3d.html dung bo THU TU (--idle:#64748B, MAU_TT.done = 0x38BDF8) va la
// trang DUY NHAT co @font-face. Hau qua do duoc:
//
//   - mot buoc "xong" hien xanh la o bang 2D nhung xanh duong o man 3D — cung
//     mot du lieu, hai mau, tuy tab dang mo;
//   - ba trang 2D roi ve font he dieu hanh vi khong trang nao nhung font;
//   - sua mot ma mau thi phai nho sua bon cho, quen mot cho la lech im lang.
//
// Nay chi vendor/token.css duoc khai mau; trang chi duoc DUNG. Hai test duoi
// giu dung dieu do — va chung phai do ngay khi ai do tra mot khai bao rieng ve
// bat ky trang nao.

// cacTrang: MOI mat cua dashboard. Co dinh danh sach, khong quet thu muc: them
// trang moi thi phai them vao day, va viec do dang de nguoi ta doc lai luat mau.
//
// Ngay 20/08 them mat thu nam — vanphong.html. Truoc do bien nay ten `bonTrang`
// va co dung bon phan tu; giu ten cu thi cai ten se noi doi ve so trang, dung
// kieu lech im lang ma ca file nay sinh ra de chan.
var cacTrang = []string{"index.html", "flow.html", "hoi-thoai.html", "3d.html", "vanphong.html"}

// tokenStatus: nam bien TRANG THAI — cai duy nhat trong bo token duoc phep co
// mau. Chung phai co dung MOT nguon.
var tokenStatus = []string{"--run", "--pending", "--done", "--idle", "--error"}

const duongToken = "vendor/token.css"

func TestCacTrangDungChungMotBoToken(t *testing.T) {
	for _, ten := range cacTrang {
		b, err := os.ReadFile(filepath.Join("web", ten))
		if err != nil {
			t.Fatalf("%s: %v", ten, err)
		}
		s := string(b)
		if !strings.Contains(s, duongToken) {
			t.Errorf("%s khong <link> toi %s — trang nay se tu ve bang mau rieng, "+
				"va no se lech voi ba trang kia ma khong ai bao loi", ten, duongToken)
			continue
		}
		// Phai la the <link rel="stylesheet">, khong phai nhac ten trong comment.
		re := regexp.MustCompile(`(?is)<link[^>]*href\s*=\s*["'][^"']*` +
			regexp.QuoteMeta(duongToken) + `["'][^>]*>`)
		the := re.FindString(s)
		if the == "" {
			t.Errorf("%s co nhac %s nhung khong phai trong the <link href=...>", ten, duongToken)
			continue
		}
		if !strings.Contains(strings.ToLower(the), "stylesheet") {
			t.Errorf("%s: the tro toi %s thieu rel=\"stylesheet\" nen trinh duyet khong ap dung:\n  %s",
				ten, duongToken, the)
		}
	}
}

// Khai lai mot token trang thai o cap trang la cach lech quay tro lai: trang do
// van chay, van dep, chi la no khong con noi cung ngon ngu voi ba trang kia.
func TestKhongTrangNaoKhaiLaiTokenTrangThai(t *testing.T) {
	for _, ten := range cacTrang {
		b, err := os.ReadFile(filepath.Join("web", ten))
		if err != nil {
			t.Fatalf("%s: %v", ten, err)
		}
		s := string(b)
		for _, tk := range tokenStatus {
			// Khai bao la `--run:` — dung ho voi `var(--run)` (khong co dau hai cham).
			re := regexp.MustCompile(regexp.QuoteMeta(tk) + `\s*:`)
			if loc := re.FindStringIndex(s); loc != nil {
				dong := 1 + strings.Count(s[:loc[0]], "\n")
				t.Errorf("%s dong %d: khai lai %s ngay trong trang. "+
					"Mau trang thai chi duoc khai o %s — de o day thi doi mau mot lan "+
					"phai nho doi bon cho, va cho nao quen se lech im lang.",
					ten, dong, tk, duongToken)
			}
		}
	}
}

// Hai test tren se XANH GIA neu token.css bien mat hoac rong: khong trang nao
// khai lai, va cai <link> van tro toi mot file khong ton tai. Nen kiem ca nguon.
func TestTokenCSSLaNguonThatSu(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("web", "vendor", "token.css"))
	if err != nil {
		t.Fatalf("thieu web/vendor/token.css: %v", err)
	}
	s := string(b)
	for _, tk := range tokenStatus {
		if !regexp.MustCompile(regexp.QuoteMeta(tk) + `\s*:\s*#[0-9A-Fa-f]{3,8}`).MatchString(s) {
			t.Errorf("token.css khong khai %s thanh ma mau — bon trang dang dung mot bien rong", tk)
		}
	}
	// Ba vai chu, va ba @font-face nuoi chung. Thieu font thi chu roi ve font he
	// dieu hanh — im lang, khong ai bao loi, dung kieu hong da xay ra.
	for _, vai := range []string{"--font-display", "--font-body", "--font-mono"} {
		if !strings.Contains(s, vai+":") {
			t.Errorf("token.css thieu vai chu %s", vai)
		}
	}
	for _, lop := range []string{".t-display", ".t-body", ".t-mono"} {
		if !strings.Contains(s, lop+"{") {
			t.Errorf("token.css thieu lop vai chu %s", lop)
		}
	}
	if n := strings.Count(s, "@font-face"); n != 3 {
		t.Errorf("token.css co %d @font-face, muon 3 (Space Grotesk + Inter + JetBrains Mono)", n)
	}
	// Duong font trong token.css la tuong doi voi CHINH file nay (no nam san
	// trong vendor/), nen viet "vendor/inter..." o day la tro sang vendor/vendor/.
	for _, f := range []string{"space-grotesk-variable.woff2", "inter-variable.woff2",
		"jetbrains-mono-variable.woff2"} {
		if !strings.Contains(s, `url("`+f+`")`) {
			t.Errorf("token.css khong tro toi %q bang duong tuong doi — file nam canh no trong vendor/", f)
		}
		if _, err := os.Stat(filepath.Join("web", "vendor", f)); err != nil {
			t.Errorf("token.css tro toi %s nhung file khong co: %v", f, err)
		}
	}
}

// token.css nam trong /vendor/ — vung server mo CONG KHAI (server.go). Neu no
// bi bat dang nhap thi trang /login va /docs/ se hien ra khong mau: nen trang,
// chu den, font he dieu hanh. Do dung kieu hong "im lang, trong co ve on".
func TestTokenCSSPhucVuCongKhai(t *testing.T) {
	s := newTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req("GET", "/vendor/token.css"))
	if w.Code != 200 {
		t.Fatalf("GET /vendor/token.css: ma %d, muon 200 (chua dang nhap van phai lay duoc)", w.Code)
	}
	if !strings.Contains(w.Body.String(), "--run:") {
		t.Error("GET /vendor/token.css khong tra ve noi dung token")
	}
}
