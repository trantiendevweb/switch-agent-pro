package dash

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Ngan keo + Tien do luot chay (web/index.html).
//
// mat2d_test.go da canh giai phau card agent, nhat ky va meter Tong quan. File
// nay canh hai thu MOI cua lan bay lai mat 2D, va chung deu la thu de "lam cho
// co" ma khong ai bao loi:
//
//   - Ngan keo: mot cai <aside> truot ra thi ai cung dung duoc bang chuot. Cai
//     de rot la ban phim — khong Esc, khong bay tieu diem, khong tra tieu diem
//     ve nut da mo. Ba thieu sot do KHONG lam trang do, chi lam nguoi dung ban
//     phim lac duong: Tab tiep tuc di ra sau lung tam man, con tro nam tren thu
//     ho khong nhin thay.
//
//   - Tien do luot chay: khoi nay de ve ra "day du" ma so khong bao gio doi.
//     Ban truoc khoa cache theo `run.id + ':' + run.state`, ma run.state dung
//     nguyen la "running" suot ca luot — nen neu bung nguyen cach khoa do sang
//     day thi tien do dong bang o buoc dau tien, va trang van trong nhu that.
//
// Moi test duoi day gan vao dung mot trong nhung dieu do. doc2D() va boComment()
// dung chung voi mat2d_test.go / mat3d_test.go.

// sauO: sau o thao tac, theo dung thu tu tren thanh cong cu. Khoa (data-nk /
// data-o) di kem id cua thu NAM TRONG o do — de test khong chi dem cai vo.
var sauO = []struct{ khoa, ruot, vi string }{
	{"fleet", `id="fleet"`, "bat ham doi"},
	{"wf", `id="wfform"`, "chay workflow"},
	{"ai", `id="aiform"`, "hoi thang AI API"},
	{"tele", `id="tele"`, "bao tin Telegram"},
	{"may", `id="quet"`, "may & don dep"},
	{"tq", `id="tq-nguon"`, "tong quan"},
}

// khoiCSS cat dung mot khoi CSS/JS ke tu `mo` cho toi dau } can bang. Can no vi
// `[^}]*` cua regexp khong di qua noi mot khoi long nhau — ma @media va ham JS
// deu long nhau.
func khoiCSS(t *testing.T, s, mo string) string {
	t.Helper()
	i := strings.Index(s, mo)
	if i < 0 {
		return ""
	}
	j := strings.Index(s[i:], "{")
	if j < 0 {
		return ""
	}
	sau := 0
	for k := i + j; k < len(s); k++ {
		switch s[k] {
		case '{':
			sau++
		case '}':
			sau--
			if sau == 0 {
				return s[i : k+1]
			}
		}
	}
	return s[i:]
}

// Man chinh CHI con ba khoi: Ham doi, Tien do luot chay, Nhat ky.
//
// Day la ca ly do ngan keo ton tai. Neu sau o thao tac van con nam trong .bento
// thi ngan keo chi la mot ban sao thu hai, va man hinh khong hoi lai duoc chut
// cho nao — tuc cong viec chua lam, chi moi bay them do.
func TestManChinhChiConHamDoiTienDoVaNhatKy(t *testing.T) {
	s := doc2D(t)
	iBento := strings.Index(s, `class="bento"`)
	iMan := strings.Index(s, `id="man"`)
	if iBento < 0 || iMan < 0 || iMan < iBento {
		t.Fatal("index.html: khong tim thay man chinh (.bento) va tam man (#man) theo dung " +
			"thu tu — ngan keo phai nam SAU toan bo man chinh")
	}
	chinh := s[iBento:iMan]
	for _, can := range []string{`id="ham"`, `id="td"`, `id="log"`} {
		if !strings.Contains(chinh, can) {
			t.Errorf("index.html: man chinh thieu %s — ba khoi phai nhin lien tuc la "+
				"Ham doi, Tien do luot chay, Nhat ky", can)
		}
	}
	for _, o := range sauO {
		if strings.Contains(chinh, o.ruot) {
			t.Errorf("index.html: o %q (%s) van con bay ra man chinh — no phai nam trong "+
				"ngan keo, neu khong thi man hinh khong hoi lai duoc cho nao", o.vi, o.ruot)
		}
	}
	// Bang tai khoan cung phai roi man chinh: no la du lieu tra cuu, khong phai
	// thu liec lien tuc.
	if strings.Contains(chinh, `id="profiles"`) {
		t.Error("index.html: bang Tai khoan van o man chinh — no thuoc o Tong quan trong ngan keo")
	}
}

// Ca sau o phai NAM THAT trong ngan keo, khong phai chi doi ten khung.
func TestSauFormNamTrongNganKeo(t *testing.T) {
	s := doc2D(t)
	i := strings.Index(s, `id="ngankeo"`)
	if i < 0 {
		t.Fatal("index.html: khong co #ngankeo — sau o thao tac van dang bay ca ra man hinh")
	}
	keo := s[i:]
	for _, o := range sauO {
		if !strings.Contains(keo, o.ruot) {
			t.Errorf("index.html: o %q (%s) khong nam trong #ngankeo", o.vi, o.ruot)
		}
	}
	// Ngan keo la hop thoai: thieu role/aria-modal thi trinh doc man hinh doc no
	// nhu mot doan van bat ky nam cuoi trang.
	dau := s[i:]
	if j := strings.Index(dau, ">"); j > 0 {
		dau = dau[:j]
	}
	for _, can := range []string{`role="dialog"`, `aria-modal="true"`, `aria-labelledby=`, `tabindex="-1"`} {
		if !strings.Contains(dau, can) {
			t.Errorf("index.html: the #ngankeo thieu %s", can)
		}
	}
}

// Moi o MOT nut tren thanh cong cu, va khoa cua nut phai khop khoa cua o.
//
// Lech mot khoa la bam nut ra ngan keo trong: khong the nao thay bang mat khi
// doc ma, nhung nguoi dung thi bam vao hu khong.
func TestMoiFormMotNutTrenThanhCongCu(t *testing.T) {
	s := doc2D(t)
	nut := regexp.MustCompile(`data-nk="([a-z]+)"`).FindAllStringSubmatch(s, -1)
	o := regexp.MustCompile(`data-o="([a-z]+)"`).FindAllStringSubmatch(s, -1)
	lay := func(m [][]string) []string {
		out := []string{}
		for _, g := range m {
			out = append(out, g[1])
		}
		sort.Strings(out)
		return out
	}
	kNut, kO := lay(nut), lay(o)
	muon := []string{}
	for _, x := range sauO {
		muon = append(muon, x.khoa)
	}
	sort.Strings(muon)
	if strings.Join(kNut, ",") != strings.Join(muon, ",") {
		t.Errorf("index.html: nut thanh cong cu co khoa %v, muon %v", kNut, muon)
	}
	if strings.Join(kO, ",") != strings.Join(muon, ",") {
		t.Errorf("index.html: o trong ngan keo co khoa %v, muon %v", kO, muon)
	}
	// Nut phai bao trang thai dong/mo cho trinh doc man hinh.
	if !strings.Contains(s, "aria-expanded") {
		t.Error("index.html: nut mo ngan keo khong co aria-expanded")
	}
}

// Ban phim: Esc dong, Tab khong thoat ra khoi ngan keo.
//
// Day la phan de bo quen nhat, vi bang chuot thi moi thu van chay ngon.
func TestNganKeoDongBangEscVaBayTieuDiem(t *testing.T) {
	ma := boComment(doc2D(t))
	if !strings.Contains(ma, `'Escape'`) {
		t.Error("index.html: khong bat phim Escape — mo ngan keo ra roi phai ra chuot " +
			"tim nut Dong moi thoat duoc")
	}
	if !strings.Contains(ma, `'Tab'`) || !strings.Contains(ma, "shiftKey") {
		t.Error("index.html: khong bay tieu diem theo Tab/Shift+Tab — Tab se di tiep ra " +
			"sau lung tam man, con tro nam tren thu nguoi dung khong nhin thay")
	}
	// Bay tieu diem phai vong tu cuoi ve dau va nguoc lai, tuc phai co danh sach
	// thu bam duoc dang HIEN trong ngan keo.
	if !regexp.MustCompile(`function\s+focusDuoc\s*\(`).MatchString(ma) {
		t.Fatal("index.html: khong co focusDuoc() — khong co cho nao liet ke thu bam duoc trong ngan keo")
	}
	fd := khoiCSS(t, ma, "function focusDuoc")
	for _, can := range []string{"ngankeo", "querySelectorAll", "offsetParent"} {
		if !strings.Contains(fd, can) {
			t.Errorf("index.html: focusDuoc() khong nhac %q — no phai chi lay thu bam duoc "+
				"DANG HIEN ben trong ngan keo", can)
		}
	}
	kd := khoiCSS(t, ma, "document.addEventListener('keydown'")
	if !strings.Contains(kd, "preventDefault") {
		t.Error("index.html: bay tieu diem khong chan hanh vi mac dinh cua Tab nen no van thoat ra ngoai")
	}
}

// Dong ngan keo phai TRA tieu diem ve dung nut da mo no.
//
// Khong tra thi con tro roi ve <body>, va nguoi dung ban phim phai Tab lai tu
// dau trang moi quay ve cho cu.
func TestNganKeoTraTieuDiemVeNutDaMo(t *testing.T) {
	ma := boComment(doc2D(t))
	mo := khoiCSS(t, ma, "function moNganKeo")
	dong := khoiCSS(t, ma, "function dongNganKeo")
	if mo == "" || dong == "" {
		t.Fatal("index.html: thieu moNganKeo()/dongNganKeo()")
	}
	if !regexp.MustCompile(`nutMo\s*=`).MatchString(mo) {
		t.Error("index.html: moNganKeo() khong nho lai nut da mo (nutMo)")
	}
	if !regexp.MustCompile(`nutMo[^\n]*\.focus\(\)`).MatchString(dong) {
		t.Error("index.html: dongNganKeo() khong tra tieu diem ve nut da mo — con tro roi " +
			"ve body, phai Tab lai tu dau trang")
	}
	// Mo xong phai dua tieu diem VAO trong ngan keo, neu khong thi Esc/Tab deu
	// khong co gi de bat.
	if !strings.Contains(mo, ".focus()") {
		t.Error("index.html: moNganKeo() khong dat tieu diem vao trong ngan keo")
	}
}

// Tien do luot chay doc dung hai duong da co, khong bia them endpoint nao.
func TestTienDoDocRunsVaFlowDetail(t *testing.T) {
	ma := boComment(doc2D(t))
	if !regexp.MustCompile(`state\.runs`).MatchString(ma) {
		t.Error("index.html: khong doc state.runs — day la cho DUY NHAT /api/state noi " +
			"luot chay nao dang chay")
	}
	if !strings.Contains(ma, "/api/flow/detail?id=") {
		t.Error("index.html: khong doc /api/flow/detail — khong co cho nao biet tung buoc")
	}
	if !regexp.MustCompile(`function\s+veTienDo\s*\(`).MatchString(ma) {
		t.Fatal("index.html: khong co veTienDo() — khong co cho nao ve khoi tien do")
	}
	if !regexp.MustCompile(`(?m)veTienDo\(`).MatchString(strings.Replace(ma, "function veTienDo(", "", 1)) {
		t.Error("index.html: veTienDo() dinh nghia ma khong ai goi — khoi tien do se trong tron")
	}
}

// Khoi tien do phai tra loi DU ba cau: toi buoc thu may tren tong may, hong o
// dau, va an het bao nhieu token/tien.
//
// Thieu cau nao thi phai mo log hoac go `sagent flow runs N` moi biet — dung
// cai ma khoi nay sinh ra de xoa bo.
func TestTienDoNoiBuocThuMayHongODauVaTonBaoNhieu(t *testing.T) {
	s := doc2D(t)
	ma := boComment(s)
	for _, id := range []string{"td-thu", "td-tong", "td-hong", "td-vao", "td-ra", "td-usd", "td-mach", "td-ds"} {
		if !strings.Contains(s, `id="`+id+`"`) {
			t.Errorf("index.html: thieu o #%s trong khoi tien do", id)
		}
	}
	ve := khoiCSS(t, ma, "function veTienDo")
	if ve == "" {
		t.Fatal("index.html: khong doc duoc than ham veTienDo()")
	}
	// Buoc thu may / tong may: phai lay tu chinh mang steps, khong phai mot so cung.
	if !strings.Contains(ve, "running") || !regexp.MustCompile(`\.length`).MatchString(ve) {
		t.Error("index.html: veTienDo() khong suy ra buoc dang chay tren tong so buoc that")
	}
	// Buoc hong phai duoc keo LEN, kem loi nhan cua no.
	if !strings.Contains(ve, `'failed'`) {
		t.Error("index.html: veTienDo() khong tim buoc state==='failed' — khoi nay ton tai " +
			"chinh de tra loi cau \"hong cho nao\"")
	}
	if !regexp.MustCompile(`hong\.msg`).MatchString(ve) {
		t.Error("index.html: khong hien loi nhan cua buoc hong — biet no hong ma khong biet vi sao")
	}
	// Cost + token cua luot chay lay tu tung buoc.
	for _, tr := range []string{"b.tokensIn", "b.tokensOut", "b.costUsd"} {
		if !strings.Contains(ve, tr) {
			t.Errorf("index.html: veTienDo() khong cong %s cua tung buoc", tr)
		}
	}
	// O chua co so phai ghi CHUA_DO, khong duoc lap so 0 — dung luat cua card agent.
	if !strings.Contains(ve, "datSo(") {
		t.Error("index.html: khoi tien do khong di qua datSo() nen o chua do se hien so 0")
	}
}

// Cache cua tien do KHONG duoc dong bang luot dang chay.
//
// Ban Tong quan cu khoa theo `run.id + ':' + run.state`. Voi mot luot dang chay
// thi run.state dung nguyen la "running" tu dau den cuoi, nen khoa do khong bao
// gio doi — bung nguyen sang tien do la khoi luon dung o buoc dau tien trong
// khi thuc te da chay xong. Trang van ve day du, khong ai bao loi.
func TestTienDoKhongDongBangLuotDangChay(t *testing.T) {
	ma := boComment(doc2D(t))
	nap := khoiCSS(t, ma, "async function napDoTuFlow")
	if nap == "" {
		t.Fatal("index.html: khong co napDoTuFlow()")
	}
	if !strings.Contains(nap, "doFlow.khoa") {
		t.Fatal("index.html: napDoTuFlow() khong con khoa cache nao")
	}
	if !regexp.MustCompile(`r\.state\s*!==\s*'running'`).MatchString(nap) {
		t.Error("index.html: napDoTuFlow() dung lai cache ke ca khi luot dang chay — " +
			"run.state la 'running' suot ca luot nen tien do se dong bang o buoc dau tien")
	}
}

// Duoi 720px: mot cot, va ngan keo phu FULL man.
//
// Tren dien thoai mot tam panel 430px vua che gan het man vua de ho mot dai nen
// vo dung — te hon ca hai lua chon.
func TestNganKeoFullManDuoi720(t *testing.T) {
	s := doc2D(t)
	kh := khoiCSS(t, s, "@media(max-width:720px)")
	if kh == "" {
		t.Fatal("index.html: khong co khoi @media(max-width:720px)")
	}
	if !strings.Contains(kh, ".bento") {
		t.Error("index.html: @media 720px khong go .bento ve mot cot")
	}
	nk := khoiCSS(t, kh, ".ngankeo")
	if nk == "" || !regexp.MustCompile(`width\s*:\s*100%`).MatchString(nk) {
		t.Error("index.html: duoi 720px ngan keo khong phu full man (.ngankeo{width:100%})")
	}
}

// reduced-motion phai tat CA cu truot cua ngan keo, khong chi nhip tho va log.
//
// Mot tam panel bay ngang man hinh la thu kho chiu nhat trong ba loai chuyen
// dong voi nguoi bi roi loan tien dinh. Luat chung `*{transition-duration:.01ms}`
// co the dinh, nhung ngan keo con dung `visibility 0s .26s` — mot delay, khong
// phai duration — nen luat chung KHONG cham toi no.
func TestReducedMotionTatTruotNganKeo(t *testing.T) {
	s := doc2D(t)
	i := strings.Index(s, "prefers-reduced-motion")
	if i < 0 {
		t.Fatal("index.html: khong co khoi prefers-reduced-motion")
	}
	khoi := s[i:]
	if j := strings.Index(khoi, "</style>"); j > 0 {
		khoi = khoi[:j]
	}
	for _, can := range []string{".ngankeo", ".man"} {
		if !strings.Contains(khoi, can) {
			t.Errorf("index.html: reduced-motion khong tat chuyen dong cua %s", can)
		}
	}
}

// Mau trong khoi tien do CHI ma hoa trang thai, khung van don sac.
//
// Cung dung luat da ap cho card agent: rail/aura/pill an var(--c); o day la
// vach mach, cham va pill cua luot chay.
func TestTienDoChiToMauTrangThai(t *testing.T) {
	s := doc2D(t)
	for _, l := range []string{".td .st{", ".td .mach b{", ".td .ds li i{"} {
		kh := khoiCSS(t, s, l)
		if kh == "" {
			t.Errorf("index.html: thieu lop %s", l)
			continue
		}
		if !strings.Contains(kh, "var(--c)") {
			t.Errorf("index.html: %s khong an mau trang thai qua var(--c)", l)
		}
	}
	// Bang mau buoc phai tro het ve token trang thai, khong ma mau nao viet thang.
	ma := boComment(s)
	bang := khoiCSS(t, ma, "const MAU_BUOC")
	if bang == "" {
		t.Fatal("index.html: khong co bang MAU_BUOC — mau tung buoc dang duoc rai tai cho")
	}
	for _, st := range []string{"done", "running", "failed", "waiting", "pending", "skipped"} {
		if !strings.Contains(bang, st) {
			t.Errorf("index.html: MAU_BUOC thieu trang thai %q — /api/flow/detail tra ve no", st)
		}
	}
	if regexp.MustCompile(`#[0-9A-Fa-f]{3,8}`).MatchString(bang) {
		t.Error("index.html: MAU_BUOC viet thang ma mau — mau trang thai chi duoc lay tu vendor/token.css")
	}
}

// Thao tac duoc bang ban phim thi cung phai NHIN THAY minh dang o dau.
func TestNganKeoVaThanhCongCuCoFocusVisible(t *testing.T) {
	s := doc2D(t)
	for _, l := range []string{".cong .nk:focus-visible", ".ngankeo:focus-visible"} {
		if !strings.Contains(s, l) {
			t.Errorf("index.html: thieu %s — tab toi day thi khong thay vien tieu diem", l)
		}
	}
}
