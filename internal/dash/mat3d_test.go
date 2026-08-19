package dash

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Mat 3D (web/3d.html) co mot bo luat rieng, ghi trong skill sagent-dashboard
// muc "Pattern view 3D" + "Anti-pattern". Tai lieu thi khong ai doc lai luc sua
// mot dong JS, nen muc nao may kiem duoc thi de may kiem — giong cach
// uxui_test.go da lam voi checklist design system.
//
// Do ngay 20/08 truoc khi sua, ban cu vi pham dung nhung muc duoi day:
//
//   - loi dieu phoi la OctahedronGeometry dac, khong wireframe, khong halo;
//   - orb phien tha quanh TUNG robot chu khong nam tren mot vanh quanh loi,
//     va pha cua hat chay lay bang Math.random() — moi lan dung canh lai khac;
//   - khong co glow sprite additive nao, tuc khong co "bloom" hop le cua ban
//     nhung (EffectComposer/UnrealBloomPass la addon, bi cam);
//   - ma mau chep cung vao JS (MAU_TT, TINT, 0x39D9E0...) thay vi doc tu
//     vendor/token.css — tuc bang mau thu tu quay ve bang duong vong: sua
//     token.css thi 3D van giu mau cu, im lang;
//   - dong den `key` bi comment nuot mat: `// --core: den cua loi dieu phoi
//     key.position.set(0,14,0); scene.add(key);` — ca cau lenh nam sau `//`
//     tren CUNG mot dong, nen den KHONG BAO GIO duoc them vao scene.
//
// Moi test duoi day gan vao dung mot trong nhung dieu do.

func doc3D(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("web", "3d.html"))
	if err != nil {
		t.Fatalf("khong doc duoc web/3d.html: %v", err)
	}
	return string(b)
}

// boComment: bo phan sau `//` cua tung dong, giu lai phan MA THAT SU CHAY.
//
// Day la phep kiem cot loi cua TestDenKeyThatSuVaoScene: chuoi "scene.add(key)"
// van co mat trong file cu, chi la no nam sau dau `//` nen khong chay. Grep
// thuong khong phan biet duoc; bo comment truoc roi grep thi phan biet duoc.
//
// Khong xu ly `//` nam trong chuoi (vi du "https://") — o file nay chuyen do
// chi xuat hien trong comment, va neu co nham thi test chi bot nhay, khong bao
// loi gia.
func boComment(s string) string {
	var b strings.Builder
	for _, dong := range strings.Split(s, "\n") {
		if i := strings.Index(dong, "//"); i >= 0 {
			dong = dong[:i]
		}
		b.WriteString(dong)
		b.WriteByte('\n')
	}
	return b.String()
}

// Loi dieu phoi phai la icosahedron WIREFRAME + halo, khong phai khoi dac.
// Khoi dac che mat orb phia sau no; wireframe thi nhin xuyen qua duoc, va no
// doc ra "bo khung dieu phoi" chu khong phai mot hon da giua san.
func TestLoiDieuPhoiLaIcosahedronWireframeCoHalo(t *testing.T) {
	s := boComment(doc3D(t))
	if !strings.Contains(s, "IcosahedronGeometry") {
		t.Error("3d.html: loi dieu phoi khong dung IcosahedronGeometry — skill muc " +
			"\"Pattern view 3D\" doi icosahedron wireframe o giua")
	}
	if strings.Contains(s, "OctahedronGeometry") {
		t.Error("3d.html: con OctahedronGeometry — day la khoi DAC cua ban cu, no che " +
			"mat orb phia sau loi moi khi camera xoay qua")
	}
	if !regexp.MustCompile(`wireframe\s*:\s*true`).MatchString(s) {
		t.Error("3d.html: loi dieu phoi khong co wireframe:true")
	}
	if !strings.Contains(s, "halo") {
		t.Error("3d.html: khong thay halo quanh loi dieu phoi")
	}
}

// Agent xep DEU tren MOT vanh quanh loi. Tha troi tu do la loi truc quan so 1
// cua ban cu: mat khong doc duoc thu tu, va hai lan mo trang thi bo cuc khac
// nhau nen khong nho duoc ai dung o dau.
//
// Phep kiem: (1) file khong con Math.random() — vi tri va pha deu phai tinh
// duoc; (2) co hang so ban kinh vanh dung chung; (3) goc lay bang i/tong*2PI.
func TestAgentXepDeuTrenMotVanhKhongThaTroi(t *testing.T) {
	s := boComment(doc3D(t))
	if strings.Contains(s, "Math.random") {
		t.Error("3d.html: con Math.random() — vi tri/pha trong canh phai tinh duoc, " +
			"ngau nhien thi moi lan dung canh lai khac va mat khong nho duoc gi")
	}
	if !regexp.MustCompile(`const\s+R_VANH\s*=\s*[\d.]+`).MatchString(s) {
		t.Error("3d.html: khong co hang so R_VANH — ban kinh vanh agent phai la MOT so " +
			"khai mot cho, khong rai so 8 khap noi")
	}
	if !regexp.MustCompile(`\(\s*i\s*/\s*[\w.]+\s*\)\s*\*\s*Math\.PI\s*\*\s*2`).MatchString(s) {
		t.Error("3d.html: goc cua agent khong tinh bang (i/tong)*Math.PI*2 — " +
			"khong dam bao xep DEU tren vanh")
	}
}

// Quang sang lam bang glow sprite ADDITIVE, khong phai post-processing.
// EffectComposer/UnrealBloomPass la addon three.js: them file roi phai tai,
// dung vao la vo rang buoc nhung-binary/offline (offline_asset_test.go giu
// phan cam; test nay giu phan THAY THE co that).
func TestQuangSangLaGlowSpriteAdditive(t *testing.T) {
	s := boComment(doc3D(t))
	if !strings.Contains(s, "THREE.Sprite(") {
		t.Error("3d.html: khong tao THREE.Sprite nao — quang sang phai la glow sprite")
	}
	if !strings.Contains(s, "AdditiveBlending") {
		t.Error("3d.html: glow sprite khong dung AdditiveBlending — chong lop khong cong " +
			"sang thi khong ra quang, chi ra mot mieng mo duc")
	}
	if !regexp.MustCompile(`createRadialGradient`).MatchString(s) {
		t.Error("3d.html: khong co createRadialGradient — texture quang phai la " +
			"gradient toa tron, khong phai o vuong dac")
	}
}

// Mau doc tu vendor/token.css qua getComputedStyle(:root), khong chep ma mau
// vao JS. Chep lai la cach "bang mau thu tu" quay ve bang duong vong: sua
// token.css mot lan thi 3D van giu mau cu, khong ai bao loi.
//
// Phep kiem manh: trong file KHONG duoc con so hex mau 6 chu so nao (0xRRGGBB)
// ngoai dung mot gia tri du phong duy nhat — de nhieu du phong thi thuc chat
// la chep bang mau lan hai.
func TestMauLayTuTokenCSSChuKhongChepVaoJS(t *testing.T) {
	s := boComment(doc3D(t))
	if !strings.Contains(s, "getComputedStyle(document.documentElement)") {
		t.Error("3d.html: khong doc getComputedStyle(document.documentElement) — " +
			"mau phai lay tu :root cua vendor/token.css")
	}
	for _, tk := range []string{"--run", "--pending", "--done", "--idle", "--error",
		"--core", "--link", "--void"} {
		if !strings.Contains(s, `mauToken('`+tk+`')`) {
			t.Errorf("3d.html: khong doc token %s tu token.css qua mauToken()", tk)
		}
	}
	hex := regexp.MustCompile(`0x[0-9A-Fa-f]{6}\b`).FindAllString(s, -1)
	if len(hex) > 1 {
		t.Errorf("3d.html: con %d ma mau chep cung trong JS (%s) — chi duoc giu DUNG MOT "+
			"gia tri du phong khi token.css khong tai duoc; con lai phai qua mauToken()",
			len(hex), strings.Join(hex, ", "))
	}
	if len(hex) == 0 {
		t.Error("3d.html: khong co gia tri du phong nao — token.css hong thi ca canh den thui")
	}
}

// Hat chay doc beam CHI chay khi phien that su dang lam viec (run/pending).
// Hat chay o trang thai done/idle la animation khong ma hoa thong tin — skill
// muc Anti-pattern goi dung ten: thua thi thay gia.
func TestHatChiChayKhiRunHoacPending(t *testing.T) {
	s := boComment(doc3D(t))
	re := regexp.MustCompile(`const\s+CHAY\s*=[^\n]*`)
	dn := re.FindString(s)
	if dn == "" {
		t.Fatal("3d.html: khong co ham CHAY() quyet dinh trang thai nao duoc chay hat")
	}
	if !strings.Contains(dn, "running") || !strings.Contains(dn, "pending") {
		t.Errorf("3d.html: CHAY() khong gan vao dung running/pending:\n  %s", dn)
	}
	if !regexp.MustCompile(`if\s*\(\s*CHAY\(`).MatchString(s) {
		t.Error("3d.html: viec tao hat khong duoc chan boi CHAY(...) — hat se chay ca " +
			"khi phien da xong hoac dang nghi")
	}
	// Beam phai noi LOI voi orb, khong phai treo lo lung.
	if !strings.Contains(s, "beam") {
		t.Error("3d.html: khong thay beam noi loi dieu phoi toi orb")
	}
}

// prefers-reduced-motion phai tat HET hat, khong phai chi dung cap nhat vi tri:
// hat dung yen giua beam la mot cham sang vo nghia. Nhung canh VAN phai ve.
func TestReducedMotionKhongTaoHat(t *testing.T) {
	s := boComment(doc3D(t))
	if !regexp.MustCompile(`CHAY\([^)]*\)\s*&&\s*!RM`).MatchString(s) &&
		!regexp.MustCompile(`!RM\s*&&\s*CHAY\(`).MatchString(s) {
		t.Error("3d.html: tao hat khong kiem !RM — reduced-motion phai KHONG tao hat nao, " +
			"chu khong phai tao roi de dung im")
	}
	// Canh van ve: renderer.render phai nam ngoai moi nhanh if(!RM).
	for _, dong := range strings.Split(s, "\n") {
		if strings.Contains(dong, "renderer.render(") && strings.Contains(dong, "!RM") {
			t.Errorf("3d.html: renderer.render bi gan vao !RM — reduced-motion se ra man den:\n  %s",
				strings.TrimSpace(dong))
		}
	}
}

// Den `key` phai THAT SU vao scene.
//
// Loi that trong ban cu: `const key = new THREE.PointLight(...);  // --core: den
// cua loi dieu phoi key.position.set(0,14,0); scene.add(key);` — nguoi viet
// xuong dong trong dau nhung tren file thi ca cau lenh nam sau `//`, cung mot
// dong. Den duoc tao, duoc dat ten, va khong bao gio sang. Kieu hong im lang
// nhat: khong loi console, chi la canh toi hon dang le.
func TestDenKeyThatSuVaoScene(t *testing.T) {
	ma := boComment(doc3D(t))
	if !regexp.MustCompile(`scene\.add\(\s*key\s*\)`).MatchString(ma) {
		t.Error("3d.html: den `key` khong duoc scene.add() trong phan MA CHAY — " +
			"kiem lai xem cau lenh co bi dau // cung dong nuot mat khong")
	}
	if !regexp.MustCompile(`key\.position\.set\(`).MatchString(ma) {
		t.Error("3d.html: den `key` khong duoc dat vi tri trong phan ma chay")
	}
}

// Nhan la HTML overlay chieu world->screen moi frame, KHONG phai 3D text.
// 3D text bi mo khi xoay va khong dung duoc font mono cua token; overlay thi
// net cang va dung dung --font-mono nhu moi du lieu may khac.
func TestNhanLaOverlayHTMLKhongPhai3DText(t *testing.T) {
	s := boComment(doc3D(t))
	for _, cam := range []string{"TextGeometry", "FontLoader", "THREE.Font("} {
		if strings.Contains(s, cam) {
			t.Errorf("3d.html: dung %s — nhan phai la HTML overlay, 3D text bi mo", cam)
		}
	}
	if !regexp.MustCompile(`\.project\(\s*camera\s*\)`).MatchString(s) {
		t.Error("3d.html: khong chieu toa do world->screen bang vector.project(camera)")
	}
	if !regexp.MustCompile(`\.z\s*<\s*1`).MatchString(s) {
		t.Error("3d.html: khong an nhan khi z>=1 — nhan cua vat sau lung camera se " +
			"lat nguoc ra truoc mat")
	}
}

// Bam orb la dung dung phien do — cung hanh vi voi nut Dung o bang 2D.
func TestBamOrbGoiApiStop(t *testing.T) {
	s := boComment(doc3D(t))
	if !strings.Contains(s, "THREE.Raycaster()") {
		t.Error("3d.html: khong co Raycaster — khong bam duoc vao orb")
	}
	if !regexp.MustCompile(`fetch\('/api/stop',\s*\{method:'POST'`).MatchString(s) {
		t.Error("3d.html: khong POST /api/stop — bam orb phai dung duoc phien, giong bang 2D")
	}
	if !strings.Contains(s, "intersectObjects(clickable)") {
		t.Error("3d.html: raycaster khong ban vao danh sach clickable cua orb")
	}
}

// three.js hong (khong co THREE, hoac trinh duyet khong dung duoc WebGL) thi
// phai hien fallback tu te va CHI hong mat 3D. Bang 2D la DOM/CSS thuan, nam o
// file khac, nen no khong duoc dinh gi.
func TestThreeHongThiCoFallback(t *testing.T) {
	s := doc3D(t)
	if !strings.Contains(s, `id="fallback"`) {
		t.Fatal("3d.html: khong co khoi #fallback")
	}
	ma := boComment(s)
	if !strings.Contains(ma, "typeof THREE === 'undefined'") {
		t.Error("3d.html: khong kiem THREE co ton tai truoc khi dung")
	}
	// WebGLRenderer nem khi trinh duyet khong dung duoc WebGL — phai bat.
	if !regexp.MustCompile(`(?s)try\s*\{[^}]*new THREE\.WebGLRenderer`).MatchString(ma) {
		t.Error("3d.html: new THREE.WebGLRenderer khong nam trong try — trinh duyet " +
			"khong dung duoc WebGL se nem, va trang ra man den thay vi hien fallback")
	}
	// Fallback phai chi duong ve 2D, khong bo nguoi dung o lai man trong.
	if !strings.Contains(s, "Ve dashboard 2D") && !strings.Contains(s, "dashboard 2D") {
		t.Error("3d.html: fallback khong chi duong ve ban 2D")
	}
}

// Ba thu ban cu lam dung — giu lai, dung sua lung tung roi mat.
func TestGiuNguyenRobotVaOrbitTayViet(t *testing.T) {
	s := boComment(doc3D(t))
	for _, ham := range []string{"function taoRobot(", "function mascotTex(",
		"function vaiTro(", "function buocCuaAddr("} {
		if !strings.Contains(s, ham) {
			t.Errorf("3d.html: mat %s — day la phan ban cu lam dung, giu nguyen", ham)
		}
	}
	// Orbit tay: khong duoc quay ve OrbitControls (offline_asset_test.go cam
	// addon, con day giu phan THAY THE that su co mat).
	if !strings.Contains(s, "POLAR_MAX") || !strings.Contains(s, "controls.update()") {
		t.Error("3d.html: mat camera orbit tu viet (clamp polar + update moi frame)")
	}
}

// Chieu sau: FogExp2 + GridHelper mo dan tren nen --void. Khong co suong thi
// vat o xa va vat o gan sang nhu nhau, canh bet lai thanh mot mieng phang.
func TestCoChieuSauFogVaLuoi(t *testing.T) {
	s := boComment(doc3D(t))
	if !strings.Contains(s, "THREE.FogExp2(") {
		t.Error("3d.html: khong co FogExp2 — canh bet, khong doc duoc xa/gan")
	}
	if !strings.Contains(s, "THREE.GridHelper(") {
		t.Error("3d.html: khong co GridHelper — khong co mat san thi orb troi trong hu khong")
	}
	if !regexp.MustCompile(`grid\.material\.opacity\s*=`).MatchString(s) {
		t.Error("3d.html: luoi khong ha opacity — luoi dam se tranh nhin voi orb")
	}
}
