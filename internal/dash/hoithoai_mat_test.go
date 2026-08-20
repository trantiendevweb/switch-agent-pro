package dash

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Ra soat mat Hoi thoai — ba cau hoi, ba nhom test.
//
// Mat nay sinh sau ba mat kia va di duong rieng, nen no la cho de troi nhat.
// Do ngay 21/08, truoc luot sua:
//
//  1. `[ui]`: CO goi Mat.apDungUI, va goi dung cho — ngay sau khi /api/state
//     ve, TRUOC ca nhanh "chua co luot nao". Khong phai sua gi; nhom test
//     duoi khoa lai dung the, vi neu ai do day no xuong duoi nhanh do thi mot
//     project chua chay luot nao se KHONG BAO GIO nhan duoc chu de.
//  2. Chuc nang: THIEU hai link ma ca ba mat kia deu co — /docs/ (Plan) va
//     /logout (Thoat) — va goi hai mat kia bang ten khong mat nao khac dung
//     ("Bang dieu khien" cho index, "Luong" cho flow).
//  3. Mau: KHONG khai lai `--run/--error/...` (designsystem_test.go da giu),
//     nhung chep GIA TRI cua chung vao trang: rgba(255,93,121,.08) o hai cho
//     chinh la --error cua bang toi (#FF5D79). Test cu bat cai TEN, khong bat
//     cai GIA TRI, nen no lot.
var bonMatWeb = []string{"index.html", "flow.html", "hoi-thoai.html", "trung-tam.html"}

func nguonMatWeb(t *testing.T, ten string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("web", ten))
	if err != nil {
		t.Fatalf("%s: %v", ten, err)
	}
	return string(b)
}

// ---------------------------------------------------------------- 1. `[ui]`

// Trinh tu trong napDanhSach() phai la: doc /api/state -> ap `[ui]` -> ve.
//
// Cho de dat sai nhat la nhet apDungUI xuong duoi nhanh `if (!runs.length)
// return`. Luc do trang van chay, van dep, chi khac mot dieu: project vua tao
// (chua co luot chay nao) mo ra la nen toi du project.toml khai theme="light",
// va link mat 3D van con du `enable_3d = false`. Khong co loi nao bao — dung
// kieu lech im lang ma ca nhom test nay lap ra de chan.
func TestHoiThoaiApUINgaySauKhiDocStateVaTruocKhiVe(t *testing.T) {
	s := boCommentJS(nguonMatWeb(t, "hoi-thoai.html"))

	viTri := func(mau string) int {
		i := strings.Index(s, mau)
		if i < 0 {
			t.Fatalf("hoi-thoai.html khong con doan %q — bai kiem nay khong con bam vao dau", mau)
		}
		return i
	}
	doc := viTri(`api("/api/state")`)
	ap := viTri("Mat.apDungUI(")
	rong := viTri("!runs.length")
	ve := viTri(`$("#runs").innerHTML = runs.map`)

	if ap < doc {
		t.Error("hoi-thoai.html goi Mat.apDungUI TRUOC khi /api/state ve — no se ap mot `ui` chua co")
	}
	if ap > rong {
		t.Error("hoi-thoai.html goi Mat.apDungUI SAU nhanh \"chua co luot nao\" — " +
			"project moi tao se khong bao gio nhan duoc theme va enable_3d")
	}
	if ap > ve {
		t.Error("hoi-thoai.html goi Mat.apDungUI SAU khi ve danh sach — chu de lat mot nhip truoc mat nguoi dung")
	}
	// Va phai an chinh cai `ui` vua doc ve, khong phai mot object tu che.
	if !regexp.MustCompile(`Mat\.apDungUI\(\s*[A-Za-z_$][\w$]*\.ui\s*\)`).MatchString(s) {
		t.Error("hoi-thoai.html khong truyen `<state>.ui` cho Mat.apDungUI — " +
			"tu dung mot nguon khac thi `[ui]` cua project.toml khong toi duoc trang nay")
	}
}

// ------------------------------------------------------- 2. chuc nang khung

// dichDieuHuong: moi dich cua thanh dieu huong, kem TEN CHINH THUC cua no.
//
// `cam` la nhung ten CHI MOT mat tung dung cho cung mot dich. Giu lai o day chu
// khong xoa di, vi cai bay se quay lai: nguoi viet mat thu nam se dich
// "Workflow" ra tieng Viet mot lan nua, va mot minh no se lai la mat noi khac
// ba mat kia.
var dichDieuHuong = []struct {
	ten   string
	hrefs []string
	nhan  string
	cam   []string
}{
	{"mat 2D", []string{"./", "index.html", "/index.html"}, "2D", []string{"Bảng điều khiển"}},
	{"mat Trung tam", []string{"trung-tam.html", "/trung-tam.html"}, "Trung tâm", nil},
	{"mat Workflow", []string{"flow.html", "/flow.html"}, "Workflow", []string{"Luồng"}},
	{"mat Hoi thoai", []string{"hoi-thoai.html", "/hoi-thoai.html"}, "Hội thoại", nil},
	{"trang ke hoach", []string{"/docs/"}, "Plan", nil},
	{"thoat dang nhap", []string{"/logout"}, "Thoát", nil},
}

var reNeo = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a>`)
var reHref = regexp.MustCompile(`(?is)href\s*=\s*"([^"]*)"`)
var reTheHTML = regexp.MustCompile(`(?is)<[^>]*>`)

// neoToi tra ve nhan cua moi the <a> tren trang tro toi mot trong cac href.
func neoToi(nguon string, hrefs []string) []string {
	var out []string
	for _, m := range reNeo.FindAllStringSubmatch(nguon, -1) {
		h := reHref.FindStringSubmatch(m[1])
		if h == nil {
			continue
		}
		for _, muon := range hrefs {
			if h[1] == muon {
				out = append(out, strings.TrimSpace(reTheHTML.ReplaceAllString(m[2], "")))
				break
			}
		}
	}
	return out
}

// Ca bon mat deu phai co loi ra: /logout va /docs/.
//
// Truoc luot nay hoi-thoai.html khong co ca hai. Thieu "Plan" chi la bat tien;
// thieu "Thoat" la mat chuc nang that — dang doc lai mot luot chay thi khong co
// cach nao bo phien dang nhap, phai roi sang mat khac truoc. Tren cai may nay
// (dang bi do mat khau lien tuc) do la cho khong nen co ma sat.
func TestBonMatDeuCoLoiThoatVaTrangKeHoach(t *testing.T) {
	for _, ten := range bonMatWeb {
		s := nguonMatWeb(t, ten)
		for _, d := range dichDieuHuong {
			if d.ten != "thoat dang nhap" && d.ten != "trang ke hoach" {
				continue
			}
			if len(neoToi(s, d.hrefs)) == 0 {
				t.Errorf("%s khong co link toi %v (%s) — ba mat kia deu co, rieng mat nay thi khong",
					ten, d.hrefs, d.nhan)
			}
		}
	}
}

// Mot dich — mot ten, tren ca bon mat.
//
// Khong doi MOI the <a> phai mang ten chinh thuc: trong than trang van co link
// van xuoi ("Mo hoi thoai", "Ve dashboard 2D") va chung dung la nen doc nhu cau
// van. Chi doi: mat nao co tro toi dich thi phai co IT NHAT MOT the mang dung
// ten cua thanh dieu huong, va khong the nao duoc mang ten thay the.
func TestMotDichDieuHuongChiMangMotTen(t *testing.T) {
	for _, ten := range bonMatWeb {
		s := nguonMatWeb(t, ten)
		for _, d := range dichDieuHuong {
			nhan := neoToi(s, d.hrefs)
			if len(nhan) == 0 {
				continue // mat nay khong tro toi dich do (vd flow.html khong tu tro vao minh)
			}
			co := false
			for _, n := range nhan {
				if n == d.nhan {
					co = true
				}
				for _, x := range d.cam {
					if n == x {
						t.Errorf("%s goi %s la %q — khong mat nao khac dung ten do; "+
							"ten chinh thuc la %q", ten, d.ten, n, d.nhan)
					}
				}
			}
			if !co {
				t.Errorf("%s tro toi %s bang cac ten %q nhung khong the nao mang ten "+
					"chinh thuc %q — di vong bon tab se doc bon ten cho mot dich",
					ten, d.ten, nhan, d.nhan)
			}
		}
	}
}

// ------------------------------------------------------------------ 3. mau

// boCommentCSS / boCommentJS: bo chu thich truoc khi soi ma mau.
//
// Can that: chinh nhung chu thich moi them vao hoi-thoai.html co GHI ra
// #FF5D79 va rgba(255,93,121,.08) de giai thich vi sao khong duoc dung chung.
// Quet ca file thi test se bao loi vi chinh loi giai thich cua no.
var reCmtCSS = regexp.MustCompile(`(?s)/\*.*?\*/`)

func boCommentCSS(s string) string { return reCmtCSS.ReplaceAllString(s, " ") }
func boCommentJS(s string) string  { return boComment(reCmtCSS.ReplaceAllString(s, " ")) }

var reKhoiStyle = regexp.MustCompile(`(?is)<style[^>]*>(.*?)</style>`)

// Ma mau chep tay vao trang la cach lech quay tro lai bang cua sau.
//
// designsystem_test.go bat viec khai lai cai TEN (`--error:` trong trang). Cho
// nay bat viec chep cai GIA TRI: `rgba(255,93,121,.08)` khong dung ten nao nen
// no lot qua test kia, ma hau qua giong het — doi `--error` trong token.css thi
// hai dai nay khong doi theo, va bat `ui.theme="light"` thi chu mang mot sac do
// (#C42B48) con nen mang sac do cua bang toi (#FF5D79).
//
// Chi soi mat Hoi thoai: ba mat kia con nhieu ma mau cu (do 21/08: index 19
// cho, flow 32, trung-tam 17), don het la mot luot rieng.
func TestHoiThoaiKhongChepMaMauVaoTrang(t *testing.T) {
	s := nguonMatWeb(t, "hoi-thoai.html")

	khoi := reKhoiStyle.FindAllStringSubmatch(s, -1)
	if len(khoi) == 0 {
		t.Fatal("hoi-thoai.html khong co khoi <style> — bai kiem nay khong con bam vao dau")
	}
	reMau := regexp.MustCompile(`#[0-9A-Fa-f]{3}\b|#[0-9A-Fa-f]{6}\b|#[0-9A-Fa-f]{8}\b|rgba?\(\s*\d`)
	for _, k := range khoi {
		css := boCommentCSS(k[1])
		for _, tim := range reMau.FindAllString(css, -1) {
			t.Errorf("hoi-thoai.html chep ma mau %q thang vao <style>. Mau chi duoc khai o "+
				"vendor/token.css; trong trang phai la var(--...) hoac color-mix pha tu var(--...). "+
				"Chep gia tri thi doi mau o token.css se khong toi duoc cho nay, va bang sang se lech.",
				tim)
		}
	}

	// Bang mau HANG ben JavaScript cung phai di qua token, khong tu che. Truoc
	// khi co token.css, mat nay ve claude tim #8B5CF6 con man 3D ve claude cam
	// #D97757 — cung mot tai khoan, hai mau, tuy tab dang mo.
	i := strings.Index(s, "const MAU = {")
	if i < 0 {
		t.Fatal("hoi-thoai.html khong con bang MAU cua hang")
	}
	than := s[i:]
	than = than[:strings.Index(than, "}")]
	if regexp.MustCompile(`#[0-9A-Fa-f]{3,8}\b`).MatchString(than) {
		t.Errorf("bang MAU trong hoi-thoai.html co ma mau chep tay:\n%s", than)
	}
	if n := strings.Count(than, "var(--prov-"); n < 5 {
		t.Errorf("bang MAU chi co %d sac hang lay tu var(--prov-*) — con lai la mau tu che", n)
	}
}

// Hai dai bao loi phai PHA TU var(--error), va phai co bac lui dung truoc.
//
// color-mix la thu duy nhat lam duoc "8% cua mot bien" ma khong phai chep ma
// mau. Doi lai no can trinh duyet moi; nen dong `background:var(--panel-2)`
// dung ngay truoc de trinh duyet cu bo qua dong sau ma van co nen — nhat hon y
// muon, nhung khong bao gio ra nen trong suot voi chu do tren nen do.
func TestDaiBaoLoiPhaTuTokenError(t *testing.T) {
	s := nguonMatWeb(t, "hoi-thoai.html")
	khoi := reKhoiStyle.FindStringSubmatch(s)
	if khoi == nil {
		t.Fatal("hoi-thoai.html khong co khoi <style>")
	}
	css := boCommentCSS(khoi[1])

	for _, lop := range []string{".err{", ".tt-lech{"} {
		i := strings.Index(css, lop)
		if i < 0 {
			t.Errorf("hoi-thoai.html khong con lop %s", lop)
			continue
		}
		than := css[i:]
		if j := strings.Index(than, "}"); j > 0 {
			than = than[:j]
		}
		if !strings.Contains(than, "color-mix(in srgb, var(--error)") {
			t.Errorf("%s khong pha nen tu var(--error) — bat ui.theme=\"light\" thi nen "+
				"giu sac do cua bang toi trong khi chu doi sang sac do cua bang sang:\n  %s", lop, than)
		}
		bac := strings.Index(than, "background:var(--panel-2)")
		pha := strings.Index(than, "background:color-mix(")
		if bac < 0 || pha < 0 || bac > pha {
			t.Errorf("%s thieu bac lui `background:var(--panel-2)` DUNG TRUOC dong color-mix — "+
				"trinh duyet khong hieu color-mix se bo ca hai va dai bao loi ra nen trong suot:\n  %s",
				lop, than)
		}
	}
}
