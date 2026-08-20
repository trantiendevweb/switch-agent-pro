package dash

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/config"
)

// Pha 5d, phan mat web: `[ui]` trong .sagent/project.toml phai DIEU KHIEN THAT
// bon mat, khong chi nam do cho `sagent config` doc len.
//
// Truoc luot nay, `internal/dash` khong he cham toi UI.Theme hay
// UI.DefaultSurface — grep ca goi khong ra mot cho nao. Tuc la DoD "hai project
// mo ra hai bo cuc khac nhau, khong sua ma" chua dat, du muc 5b/5c da xanh.

func TestStatePhaiTraVeUI(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	m := doc(t, s, ck, "/api/state")
	raw, co := m["ui"]
	if !co {
		t.Fatalf("/api/state thieu khoa \"ui\" — bon mat web khong co cach nao biet project khai bo cuc gi.\nCo: %v", khoa(m))
	}
	ui, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("ui khong phai object: %T", raw)
	}
	for _, k := range []string{"defaultSurface", "theme", "columns", "enable3d"} {
		if _, co := ui[k]; !co {
			t.Errorf("ui thieu khoa %q", k)
		}
	}
}

// Server phai GIAI SAN bo cot mac dinh, khong tra mang rong roi de moi trang tu
// biet "rong thi dung bon cot kia". Bo mac dinh chep lam nhieu ban trong
// JavaScript la dung cach bang mau tung troi khoi nhau truoc khi co token.css.
func TestUITraVeCotDaGiaiMacDinh(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	m := doc(t, s, ck, "/api/state")
	ui := m["ui"].(map[string]any)
	cot, _ := ui["columns"].([]any)
	if len(cot) != len(config.CotMacDinh) {
		t.Fatalf("khong khai ui.columns thi phai tra ve %d cot mac dinh, duoc %d: %v",
			len(config.CotMacDinh), len(cot), cot)
	}
	for i, c := range config.CotMacDinh {
		if cot[i] != c {
			t.Errorf("cot %d: muon %q, duoc %v", i, c, cot[i])
		}
	}
}

// Mat 3D bat mac dinh: project cu nang cap len ma lang le mat mot mat thi
// khong ai bao loi.
func TestUIMacDinhBat3D(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	ui := doc(t, s, ck, "/api/state")["ui"].(map[string]any)
	if ui["enable3d"] != true {
		t.Errorf("enable3d mac dinh phai la true, duoc %v", ui["enable3d"])
	}
}

// ---- token.css: bang sang phai co DU token, khong thieu cai nao ----

// Thieu mot token o bang sang thi bien do ROI VE gia tri cua bang toi — tuc la
// mot cham mau toi tren nen trang, khong ai bao loi. Day la kieu hong im lang
// ma design system lap ra de chan.
func TestBangSangKhaiDuMoiTokenCuaBangToi(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("web", "vendor", "token.css"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)

	i := strings.Index(s, `:root[data-theme="light"]`)
	if i < 0 {
		t.Fatal("token.css khong co bang sang — ui.theme = \"light\" se khong doi gi")
	}
	sang := s[i:]
	if j := strings.Index(sang, "}"); j > 0 {
		sang = sang[:j]
	}
	toi := s[strings.Index(s, ":root{"):i]

	reBien := regexp.MustCompile(`(--[a-z0-9-]+)\s*:`)
	bo := func(x string) map[string]bool {
		m := map[string]bool{}
		for _, g := range reBien.FindAllStringSubmatch(x, -1) {
			m[g[1]] = true
		}
		return m
	}
	bToi, bSang := bo(toi), bo(sang)
	for ten := range bToi {
		// Font va easing khong phu thuoc chu de — chi mau moi phai khai lai.
		if strings.HasPrefix(ten, "--font") || ten == "--ease" {
			continue
		}
		if !bSang[ten] {
			t.Errorf("bang sang thieu %s — bien do se roi ve mau cua bang toi tren nen trang", ten)
		}
	}
	// Va khong duoc khai THUA: mot ten chi co o bang sang nghia la bang toi
	// dang thieu no, hoac ai do go sai ten.
	for ten := range bSang {
		if !bToi[ten] {
			t.Errorf("bang sang co %s ma bang toi khong co — go sai ten?", ten)
		}
	}
}

// ---- mat.js: mot luat cho ca bon trang ----

func matJS(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("web", "vendor", "mat.js"))
	if err != nil {
		t.Fatalf("thieu web/vendor/mat.js: %v", err)
	}
	return string(b)
}

// Day la bai quan trong nhat cua nhom: bang nhan cot ben JavaScript va
// config.CotTaiKhoan ben Go phai la MOT. Them cot o Go ma quen them nhan thi
// dau bang hien ra ten khoa tho ("bat_dau"), con them o JS ma quen o Go thi
// nguoi dung khai vao project.toml se bi bao loi "ten cot la".
func TestNhanCotJSPhuHopVoiCotTaiKhoanGo(t *testing.T) {
	js := matJS(t)
	i := strings.Index(js, "NHAN_COT = {")
	if i < 0 {
		t.Fatal("mat.js khong co bang NHAN_COT")
	}
	than := js[i:]
	than = than[:strings.Index(than, "}")]

	for _, c := range config.CotTaiKhoan {
		if !regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(c) + `\s*:`).MatchString(than) {
			t.Errorf("mat.js thieu nhan cho cot %q — dau bang se hien ten khoa tho", c)
		}
	}
	reKhoa := regexp.MustCompile(`(?m)^\s*([a-z_0-9]+)\s*:`)
	for _, g := range reKhoa.FindAllStringSubmatch(than, -1) {
		if !config.CotHopLe(g[1]) {
			t.Errorf("mat.js co nhan cho %q nhung config.validate() se tu choi ten nay", g[1])
		}
	}
}

// Tat mat 3D thi phai GO han the <a>, khong phai chi an di: link an van nam
// trong thu tu Tab va van doc duoc bang trinh doc man hinh, nen nguoi dung ban
// phim van lac vao mot mat ma du an da tat.
func TestTat3DThiGoHanLinkChuKhongAn(t *testing.T) {
	js := matJS(t)
	if !strings.Contains(js, ".remove()") {
		t.Error("mat.js khong go link mat 3D — an bang CSS thi ban phim van Tab toi duoc")
	}
	if regexp.MustCompile(`display\s*=\s*['"]none`).MatchString(js) {
		t.Error("mat.js dung display:none de tat mat 3D — phai go han the <a>")
	}
}

// Bon trang deu phai nap mat.js VA chay themeSom() ngay trong <head>. Trang nao
// quen thi doi chu de xong trang do van toi, ma khong co loi nao.
func TestBonTrangDeuNapMatJS(t *testing.T) {
	for _, ten := range []string{"index.html", "trung-tam.html", "flow.html", "hoi-thoai.html"} {
		b, err := os.ReadFile(filepath.Join("web", ten))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		if !strings.Contains(s, `src="vendor/mat.js"`) {
			t.Errorf("%s khong nap vendor/mat.js", ten)
			continue
		}
		if !strings.Contains(s, "Mat.themeSom()") {
			t.Errorf("%s khong goi Mat.themeSom() — mo trang se nhay tu toi sang sang", ten)
		}
		if !strings.Contains(s, "Mat.apDungUI(") {
			t.Errorf("%s khong goi Mat.apDungUI() — [ui] cua project khong toi duoc trang nay", ten)
		}
		// themeSom phai chay TRUOC than trang, khong phai cuoi file.
		if i, j := strings.Index(s, "Mat.themeSom()"), strings.Index(s, "<body"); i > 0 && j > 0 && i > j {
			t.Errorf("%s goi themeSom() sau <body> — van nhay mot nhip", ten)
		}
	}
}

// ---- index.html: bang tai khoan ve theo cot da khai ----

func TestBangTaiKhoanVeTheoCot(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := boComment(string(b))

	if !strings.Contains(s, "Mat.cotHopLe(") {
		t.Error("index.html khong loc cot qua Mat.cotHopLe — cau hinh cu trong dem trinh duyet se lam vo bang")
	}
	if !strings.Contains(s, "profiles-head") {
		t.Error("index.html khong dung lai <thead> luc chay — dau bang se dung yen khi doi cot")
	}
	// Ba cot moi phai VE DUOC THAT, khong phai chi hop le o config.
	for _, c := range []string{"pid", "nhanh", "bat_dau"} {
		if !regexp.MustCompile(`case\s*'` + c + `'`).MatchString(s) {
			t.Errorf("index.html khong ve duoc cot %q — khai vao project.toml se ra o trong", c)
		}
	}
	// Dau bang cu (cot chep cung) phai bien mat, khong thi hai dau bang chong nhau.
	if regexp.MustCompile(`<thead[^>]*><tr><th></th><th>Provider</th>`).MatchString(s) {
		t.Error("index.html van con dau bang chep cung — ui.columns se khong doi duoc gi")
	}
}

// ---- flow.html: flow ghim ----

func TestFlowHTMLGhimFlow(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("web", "flow.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := boComment(string(b))
	if !strings.Contains(s, "Mat.ghimLenDau(") {
		t.Error("flow.html khong ghim flow theo ui.pinned_flows")
	}
	if !strings.Contains(s, "pinnedFlows") {
		t.Error("flow.html khong doc ui.pinnedFlows")
	}
	if !strings.Contains(s, "optgroup") {
		t.Error("flow.html khong tach nhom phan ghim — thu tu la ma khong giai thich doc nhu loi sap xep")
	}
}

// Ten khong co trong flows.toml thi bo qua, khong dung muc ma: nguoi dung bam
// vao se nhan loi tu server va tuong flow hong, chu khong nghi la no da bi xoa.
func TestGhimBoQuaTenKhongCoThat(t *testing.T) {
	js := matJS(t)
	i := strings.Index(js, "function ghimLenDau")
	if i < 0 {
		t.Fatal("mat.js khong co ghimLenDau")
	}
	than := js[i:]
	if len(than) > 800 {
		than = than[:800]
	}
	if !strings.Contains(than, "find(") {
		t.Error("ghimLenDau khong doi chieu ten ghim voi danh sach that — se dung muc ma")
	}
}
