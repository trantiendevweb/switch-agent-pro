package dash

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// Mat VAN PHONG (web/trung-tam.html) la mat thu nam cua dashboard, them 20/08
// theo docs/KE-HOACH-VAN-PHONG.md — Dot 3. No khac ba mat 2D va khac ca
// 3d.html (so do): day la NOI LAM VIEC, bon phong theo LOAI VIEC + mot sanh
// chung, nhan vat di lai giua cac phong khi buoc doi loai viec.
//
// Mat nay de hong IM LANG hon moi mat khac, vi no phu thuoc hai asset nhi phan
// nam trong binary (GLTFLoader.js + RobotExpressive.glb). Thieu mot trong hai
// thi canh trong tron, va khong co dong log nao noi vi sao. Nen tung test duoi
// day gan vao dung mot cach hong da luong truoc:
//
//   - trang tro toi asset ma asset khong co / rong -> offline_asset_test.go;
//   - trang co asset nhung goi sai ten clip -> nhan vat dung yen tu the T;
//   - trang bia chu vao bong thoai thay vi lay `output` that;
//   - trang chep ma mau vao JS -> bang mau thu nam quay ve bang duong vong;
//   - trang tien tay dung OrbitControls -> pha rang buoc nhung mot file core;
//   - bon mat cu khong co link sang -> tinh nang co ma khong ai tim thay.

func docTrungTam(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("web", "trung-tam.html"))
	if err != nil {
		t.Fatalf("khong doc duoc web/trung-tam.html: %v", err)
	}
	return string(b)
}

// maTrungTam: phan MA THAT SU CHAY (da bo comment). boComment o mat3d_test.go.
func maTrungTam(t *testing.T) string {
	t.Helper()
	return boComment(docTrungTam(t))
}

func docGLB(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("web", "vendor", "RobotExpressive.glb"))
	if err != nil {
		t.Fatalf("khong doc duoc web/vendor/RobotExpressive.glb: %v", err)
	}
	return b
}

// Ba file phai co MAT trong trang, va deu la duong noi bo trong vendor/.
// (TestDuongVendorTroDungFile o offline_asset_test.go kiem tiep rang chung tro
// dung file co that.)
func TestVanPhongNapDuBaAssetNhung(t *testing.T) {
	s := maTrungTam(t)
	can := map[string]string{
		"vendor/three.min.js":        "nhan ba chieu",
		"vendor/GLTFLoader.js":       "bo doc .glb — loader da vendor, khong phai addon hieu ung",
		"vendor/RobotExpressive.glb": "mo hinh nhan vat CC0, 13 clip dat ten theo trang thai",
		"vendor/token.css":           "bang mau + font dung chung voi bon mat kia",
	}
	for d, vi := range can {
		if !strings.Contains(s, d) {
			t.Errorf("vanphong.html khong nap %s (%s)", d, vi)
		}
	}
}

// SAU dieu, moi dieu mot viec that. Ten clip phai KHOP voi ten trong .glb: go
// nham mot chu thi mixer khong tim thay action, nhan vat dung im o tu the bind
// pose — canh van ve ra, khong loi nao, va nguoi xem tuong la "dang ranh".
func TestVanPhongDungDungSauClipCoThatTrongGLB(t *testing.T) {
	s := maTrungTam(t)
	glb := string(docGLB(t))

	// clip -> viec that dang sau no (theo ke hoach Dot 3)
	clip := map[string]string{
		"Idle":     "ranh",
		"Walking":  "chuyen phong (buoc doi loai viec)",
		"Running":  "dang chay buoc",
		"Wave":     "giao viec — co canh needs giua hai buoc",
		"ThumbsUp": "buoc vua xong",
		"No":       "buoc hong",
	}
	for ten, viec := range clip {
		if !strings.Contains(s, `'`+ten+`'`) {
			t.Errorf("vanphong.html khong dung clip %q (%s) — sau dieu deu phai co mat", ten, viec)
		}
		if !strings.Contains(glb, `"name":"`+ten+`"`) {
			t.Errorf("RobotExpressive.glb khong co clip ten %q — goi ten nay thi mixer "+
				"khong tim thay action va nhan vat dung im o tu the bind pose", ten)
		}
	}
}

// Bon phong theo LOAI VIEC + mot sanh chung o giua. NAM vai hop le cua
// internal/flow phai co cho ngoi; vai RONG (chua phan vai) phai roi ve sanh
// chung, khong duoc doan ho vao mot phong nao cho dep doi hinh.
func TestVanPhongDuBonPhongVaSanhChung(t *testing.T) {
	s := maTrungTam(t)

	for _, p := range []string{"Phòng họp", "Phòng code", "Phòng test", "Phòng review", "Sảnh chung"} {
		if !strings.Contains(s, p) {
			t.Errorf("vanphong.html thieu %q — ke hoach chot bon phong theo loai viec + mot sanh chung", p)
		}
	}

	// Moi vai hop le phai duoc mot phong nhan. Quet bang danh sach vai:[...]
	// trong bang PHONG, khong doc tung dong bang mat.
	reVai := regexp.MustCompile(`vai\s*:\s*\[([^\]]*)\]`)
	nhan := map[string]int{}
	for _, m := range reVai.FindAllStringSubmatch(s, -1) {
		for _, v := range strings.Split(m[1], ",") {
			v = strings.Trim(strings.TrimSpace(v), `'"`)
			if v != "" {
				nhan[v]++
			}
		}
	}
	for _, v := range flow.VaiTroHopLe() {
		switch nhan[v] {
		case 1: // dung mot phong nhan vai nay
		case 0:
			t.Errorf("vai %q khong phong nao nhan — nhan vat mang vai do se roi ve sanh chung, "+
				"tuc mat web bao \"chua phan vai\" trong khi flows.toml da khai", v)
		default:
			t.Errorf("vai %q duoc %d phong cung nhan — nhan vat se nhay giua hai phong tuy thu tu duyet",
				v, nhan[v])
		}
	}

	// Vai RONG phai ra sanh chung. Ham tra phong khong tim thay vai nao thi
	// phai tra SANH, chu khong phai PHONG[0].
	rePhong := regexp.MustCompile(`(?s)function phongCuaVai\s*\([^)]*\)\s*\{(.*?)\n\}`)
	m := rePhong.FindStringSubmatch(s)
	if m == nil {
		t.Fatal("vanphong.html khong co ham phongCuaVai — vai tro phai la thu QUYET DINH phong")
	}
	if !strings.Contains(m[1], "return SANH") {
		t.Error("phongCuaVai khong tra SANH cho vai la/rong — buoc chua phan vai se bi xep vao " +
			"mot phong nao do, va nguoi doc tuong no da duoc phan vai")
	}
}

// Vai tro phai lay tu /api/flow/detail (truong vaiTro) — MOT nguon cho ca ba
// mat. Mat nay tu suy vai tu ten buoc hay tu do thi needs la bat dau lech voi
// hai mat kia, va lech kieu do khong ai bao loi.
func TestVanPhongLayVaiTuFlowDetail(t *testing.T) {
	s := maTrungTam(t)
	for _, d := range []string{"/api/flows", "/api/flow/detail", "/api/events"} {
		if !strings.Contains(s, d) {
			t.Errorf("vanphong.html khong goi %s — mat nay phai doc cung nguon voi 2D va 3D", d)
		}
	}
	if !strings.Contains(s, "vaiTro") {
		t.Error("vanphong.html khong doc truong vaiTro cua /api/flow/detail — vai tro la DU LIEU, " +
			"khong duoc suy tu ten buoc")
	}
}

// Bong thoai phai la cau THAT cua agent (truong output), va phai la HTML
// overlay chu khong phai 3D text (3D text nhoe khi xoay, va khong dung duoc
// --font-mono cua token).
func TestVanPhongBongThoaiLayOutputThat(t *testing.T) {
	s := maTrungTam(t)

	// Chu trong bong phai DEN TU truong output, va den qua dung mot duong: ham
	// loiThoai. Kiem `strings.Contains(s, ".output")` cho ca file la chua du —
	// chuoi do co the con sot lai o cho khac trong khi bong da bia chu.
	re := regexp.MustCompile(`(?s)function loiThoai\s*\([^)]*\)\s*\{(.*?)\n\}`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		t.Fatal("vanphong.html khong co ham loiThoai — bong thoai phai lay tu mot cho duy nhat")
	}
	if !strings.Contains(m[1], ".output") {
		t.Error("loiThoai khong doc truong output cua buoc — bong thoai phai la cau THAT cua " +
			"agent, khong duoc bia mot cau cho canh sinh dong")
	}
	if !strings.Contains(m[1], `return ''`) {
		t.Error("loiThoai khong co duong tra RONG — buoc chua noi gi thi phai KHONG co bong, " +
			"tha trong con hon bia")
	}
	// Va bong chi duoc dung khi loiThoai tra ve chu that.
	if !strings.Contains(s, "loiThoai(st)") || !strings.Contains(s, "if(chu){") {
		t.Error("bong thoai khong duoc dung tu ket qua loiThoai(st) — mot nguon chu duy nhat")
	}
	if !strings.Contains(s, "'thoai'") {
		t.Error("vanphong.html khong co bong thoai HTML overlay (lop .thoai)")
	}
	for _, cam := range []string{"TextGeometry", "FontLoader"} {
		if strings.Contains(s, cam) {
			t.Errorf("vanphong.html dung %s — chu 3D nhoe khi xoay va con keo them mot file "+
				"addon nua; nhan phai la HTML overlay", cam)
		}
	}
	// Chu cua agent la du lieu NGUOI KHAC viet: dat bang textContent, khong
	// bang innerHTML. Mot dong output co `<` la du de no chay thanh the HTML.
	if strings.Contains(s, "innerHTML") {
		t.Error("vanphong.html con dung innerHTML — output cua agent phai vao bang textContent")
	}
}

// May cham la NHAN VAT RIENG, khac hinh voi agent. Buoc test/lint/shell/merge
// khong co agent nao dung sau; ve chung thanh nguoi la noi doi ve ai da lam.
func TestVanPhongMayChamKhacHinhAgent(t *testing.T) {
	s := maTrungTam(t)
	if !strings.Contains(s, "taoMayCham") {
		t.Fatal("vanphong.html khong co ham dung may cham — buoc may se deo lot hinh agent")
	}
	// May cham dung bang khoi hinh cua chinh trang nay, KHONG dung .glb robot.
	re := regexp.MustCompile(`(?s)function taoMayCham\s*\([^)]*\)\s*\{(.*?)\n\}`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		t.Fatal("khong doc duoc than ham taoMayCham")
	}
	if strings.Contains(m[1], "loader.parse") || strings.Contains(m[1], "RobotExpressive") {
		t.Error("taoMayCham dung lai mo hinh robot — nhin vao khong phan biet duoc dau la nguoi, dau la may")
	}
	for _, typ := range []string{flow.TypeTest, flow.TypeLint, flow.TypeShell, flow.TypeMerge} {
		if !strings.Contains(s, `'`+typ+`'`) {
			t.Errorf("vanphong.html khong xep loai buoc %q vao may cham", typ)
		}
	}
}

// Camera orbit TU VIET. TestKhongDungAddonThreeJS da chan OrbitControls o moi
// trang; test nay kiem chieu con lai — thay the phai co that, khong phai bo
// addon di roi de camera dung im.
func TestVanPhongCameraOrbitTuViet(t *testing.T) {
	s := maTrungTam(t)
	can := map[string]string{
		"pointerdown":   "keo de xoay",
		"wheel":         "lan de zoom",
		"camera.lookAt": "camera phai nhin vao target sau moi lan doi goc",
		"POLAR_MAX":     "clamp polar de camera khong lat qua cuc",
	}
	for k, vi := range can {
		if !strings.Contains(s, k) {
			t.Errorf("vanphong.html thieu %q trong camera tu viet (%s)", k, vi)
		}
	}
}

// Mau phai doc tu vendor/token.css qua mauToken(), khong chep ma mau vao JS.
// Do la duong vong ma "bang mau thu tu" da quay ve mot lan: token.css doi mot
// ma thi trang nay giu ma cu, im lang, khong ai bao loi.
func TestVanPhongKhongChepMaMauVaoTrang(t *testing.T) {
	s := maTrungTam(t)
	if !strings.Contains(s, "mauToken(") {
		t.Fatal("vanphong.html khong doc mau tu token.css qua mauToken()")
	}
	if hex := regexp.MustCompile(`#[0-9A-Fa-f]{6}\b`).FindAllString(s, -1); len(hex) > 0 {
		t.Errorf("vanphong.html chep %d ma mau thang vao trang (%v) — mau chi duoc khai o "+
			"vendor/token.css, trang chi duoc dung", len(hex), hex)
	}
	// Mot gia tri du phong duy nhat (XAM). Nhieu hon mot tuc la chep lai bang
	// mau lan hai bang so hex.
	if n := len(regexp.MustCompile(`0x[0-9A-Fa-f]{6}`).FindAllString(s, -1)); n > 1 {
		t.Errorf("vanphong.html co %d hang so mau dang 0xRRGGBB — chi duoc DUNG MOT gia tri du phong", n)
	}
}

// prefers-reduced-motion phai TAT HAN chuyen dong, khong phai giam bot: mixer
// khong chay, nhan vat khong di, camera khong tu xoay.
func TestVanPhongTatChuyenDongKhiGiamChuyenDong(t *testing.T) {
	s := maTrungTam(t)
	if !strings.Contains(s, "prefers-reduced-motion") {
		t.Fatal("vanphong.html khong doc prefers-reduced-motion")
	}
	if !regexp.MustCompile(`RM\s*=\s*matchMedia\(`).MatchString(s) {
		t.Error("vanphong.html khong dat bien RM tu matchMedia — CSS tat animation nhung " +
			"vong lap JS van chay thi nhan vat van di lai")
	}
	// Vong lap phai co nhanh chan: mixer.update va buoc di deu nam trong if(!RM).
	//
	// Quet MOI nhanh if(!RM), khong chi nhanh dau tien. Ban truoc chi lay nhanh
	// dau, nen chi can them mot ham co if(!RM) o phia tren la test do trong khi
	// ma van dung — va chuyen do da xay ra that luc gop hai mat. Mot bai kiem
	// gay vi THU TU HAM trong file thi lan sau nguoi ta sua no cho im, chu khong
	// sua cai no dinh bat.
	re := regexp.MustCompile(`(?s)if\s*\(\s*!RM\s*\)\s*\{(.*?)\n  \}`)
	coChan := false
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		if strings.Contains(m[1], "mixer.update") {
			coChan = true
			break
		}
	}
	if !coChan {
		t.Error("mixer.update khong nam trong nhanh if(!RM) nao — bat giam chuyen dong ma nhan " +
			"vat van hoat hinh la khong ton trong lua chon cua nguoi dung")
	}
	if !strings.Contains(s, "RM ? 0 :") {
		t.Error("camera van tu xoay khi bat giam chuyen dong")
	}
}

// Hong thi phai noi THIEU FILE NAO. Man den im lang la kieu hong du an nay so
// nhat: nguoi dung doan la loi mang, trong khi asset nam ngay trong binary.
func TestVanPhongBaoRoThieuFileNao(t *testing.T) {
	s := maTrungTam(t)
	if !strings.Contains(s, "function baoThieu") {
		t.Fatal("vanphong.html khong co duong bao loi — canh trong tron thi khong ai biet vi sao")
	}
	for _, f := range []string{"vendor/three.min.js", "vendor/GLTFLoader.js", "vendor/RobotExpressive.glb"} {
		re := regexp.MustCompile(`baoThieu\(\s*['"]` + regexp.QuoteMeta(f))
		if !re.MatchString(s) {
			t.Errorf("thieu %s thi trang khong noi ten file do — bao chung chung thi nguoi dung "+
				"khong biet phai kiem cai gi", f)
		}
	}
}

// Tinh nang co ma khong ai tim thay thi bang khong co. Moi mat deu phai co
// duong sang Trung tam.
//
// GHI CHU KHI GOP (20/08): hai mat ba chieu — "3D" (so do quy dao) va "Van
// phong" (san lam viec) — gop lam MOT, ten "Trung tam". Chung von ve CUNG mot
// du lieu bang hai ngon ngu, va nguoi dung phai tu ghep trong dau. Nen bai nay
// doi ten link theo, va danh sach trang giam tu nam xuong bon.
func TestMoiMatDeuCoLinkSangTrungTam(t *testing.T) {
	for _, ten := range []string{"index.html", "flow.html", "hoi-thoai.html", "trung-tam.html"} {
		b, err := os.ReadFile(filepath.Join("web", ten))
		if err != nil {
			t.Fatalf("%s: %v", ten, err)
		}
		s := string(b)
		re := regexp.MustCompile(`(?is)<a[^>]*href\s*=\s*["']/?trung-tam\.html["'][^>]*>([^<]*)</a>`)
		m := re.FindStringSubmatch(s)
		if m == nil {
			t.Errorf("%s khong co the <a> nao tro toi trung-tam.html — mat gop khong co duong vao "+
				"tu cac mat khac thi khong ai mo duoc no", ten)
			continue
		}
		if !strings.Contains(m[1], "Trung tâm") {
			t.Errorf("%s: link toi trung-tam.html ghi %q, muon chu \"Trung tâm\" — mot hanh dong "+
				"giu mot ten xuyen suot", ten, strings.TrimSpace(m[1]))
		}
	}
}

// Trang phai phuc vu duoc that, va phai DOI DANG NHAP nhu ba mat kia: no bay
// ra prompt, output va ten tai khoan cua ca luot chay.
func TestVanPhongPhucVuVaDoiDangNhap(t *testing.T) {
	s := newTestServer(t)

	w := httptest.NewRecorder()
	s.ServeHTTP(w, req("GET", "/trung-tam.html"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("GET /trung-tam.html chua dang nhap: ma %d, muon 401 — trang nay bay ra "+
			"output cua agent", w.Code)
	}

	r := req("GET", "/trung-tam.html")
	r.AddCookie(dangNhap(t, s, "127.0.0.1:4600"))
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /trung-tam.html da dang nhap: ma %d, muon 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "vendor/RobotExpressive.glb") {
		t.Error("trang phuc vu ra khong phai vanphong.html")
	}
}
