package dash

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Mat 2D (web/index.html) co luat rieng, ghi trong skill sagent-dashboard muc
// "Pattern view 2D". Giong mat3d_test.go: muc nao may kiem duoc thi de may kiem,
// vi khong ai mo lai tai lieu luc sua mot dong CSS.
//
// Do ngay 20/08 truoc khi sua, ban cu vi pham dung nhung muc duoi day:
//
//   - ham doi ve bang mot cai BANG 5 cot (#/dia chi/PID/noi lam viec/nut) —
//     mat phai doc theo hang, va bang do khong xuong duoc dien thoai;
//   - khong co card agent nao, nen khong co rail, khong co quang, khong co pill
//     trang thai, khong co nhip tho: nhin vao khong biet cai nao dang song;
//   - khong co muc Tong quan nao, tuc khong co cho nao tra loi "ham doi nay
//     tieu bao nhieu" — trong khi so THAT co san o /api/flow/detail va /api/ai;
//   - nhat ky nam trong cot phai 380px (cau nao cung xuong ba dong), cap 200
//     dong, chen o DAU nen doc nguoc thoi gian, va chi to mau cho dung ba loai;
//   - napFlow() duoc dinh nghia nhung KHONG BAO GIO duoc goi, nen o chon quy
//     trinh rong tron — dung kieu hong im lang: trang van ve day du.
//
// Moi test duoi day gan vao dung mot trong nhung dieu do.
//
// boComment() dung chung voi mat3d_test.go: bo phan sau `//` de chi con MA THAT
// SU CHAY. Can no vi chinh file nay giai thich rat nhieu trong comment — grep
// tho se xanh gia nho mot cau chu thich.

func doc2D(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("web", "index.html"))
	if err != nil {
		t.Fatalf("khong doc duoc web/index.html: %v", err)
	}
	return string(b)
}

// Ham doi la LUOI CARD bento, khong phai bang.
//
// Bang bat mat doc theo HANG, ma thu nguoi van hanh can la "liec phat biet ca
// ham doi dang the nao" — tuc doc theo KHOI. Kich thuoc card khai bang
// auto-fill minmax(238px,1fr) nen no tu xuong mot cot, khong can luat rieng.
func TestHamDoiLaLuoiCardKhongPhaiBang(t *testing.T) {
	s := doc2D(t)
	if !regexp.MustCompile(`repeat\(\s*auto-fill\s*,\s*minmax\(\s*238px\s*,\s*1fr\s*\)\s*\)`).MatchString(s) {
		t.Error("index.html: khong co luoi auto-fill minmax(238px,1fr) — skill muc " +
			"\"Pattern view 2D\" doi ham doi la luoi card tu xep, khong phai bang")
	}
	if strings.Contains(s, `id="sessions"`) {
		t.Error("index.html: con <tbody id=\"sessions\"> — bang phien cu van con day, " +
			"tuc luoi card moi chi la lop son ben canh")
	}
	if !strings.Contains(s, `id="ham"`) {
		t.Error("index.html: khong co khung #ham de do card agent vao")
	}
	// Bento: ham doi chiem chinh, cot phai la Tong quan + o dieu khien.
	if !regexp.MustCompile(`\.bento\{[^}]*grid-template-columns`).MatchString(s) {
		t.Error("index.html: khong co bo cuc .bento hai cot")
	}
}

// Card agent phai du GIAI PHAU. Thieu mot lop thi card van "dep" nhung mat
// mot tin: thieu rail la mat dai mau doc canh, thieu pill la phai doc chu moi
// biet trang thai, thieu meta la khong biet no chay bao lau.
func TestCardAgentDuGiaiPhau(t *testing.T) {
	s := doc2D(t)
	lop := map[string]string{
		".ag .rail":  "dai mau 2px doc canh trai — dau hieu doc duoc tu xa nhat",
		".ag .aura":  "quang goc tren, cung mau trang thai",
		".ag .glyph": "dau nhan hang (sac --prov-*, la DANH TINH)",
		".ag .who":   "provider:account, mono",
		".ag .st":    "pill trang thai co cham",
		".ag .verb":  "verb cua dong viec, mono",
		".ag .meta":  "ba o tokens/cost/elapsed",
		".ag .dung":  "nut Dung an trong card",
	}
	for l, vi := range lop {
		if !strings.Contains(s, l+"{") {
			t.Errorf("index.html: thieu lop %s (%s)", l, vi)
		}
	}
	// Va JS phai THAT SU dung ra tung lop do, khong phai chi khai CSS cheo leo.
	ma := boComment(s)
	for _, dau := range []string{`class="rail"`, `class="aura"`, `class="st"`,
		`class="verb"`, `class="meta"`, `class="btn danger dung"`} {
		if !strings.Contains(ma, dau) {
			t.Errorf("index.html: JS khong dung ra %s — CSS co lop nhung khong card nao mang no", dau)
		}
	}
	// provider:account va moi o so lieu deu phai la mono (design system: sans
	// cho so lieu la anti-pattern, cot so khong thang hang).
	for _, l := range []string{".ag .who{", ".ag .verb{", ".ag .meta .v{"} {
		i := strings.Index(s, l)
		if i < 0 {
			continue
		}
		if j := strings.Index(s[i:], "}"); j > 0 && !strings.Contains(s[i:i+j], "--font-mono") {
			t.Errorf("index.html: %s khong dung var(--font-mono) — day la du lieu may", l)
		}
	}
}

// Nhip tho CHI danh cho thu dang song, va hai toc do phai KHAC nhau.
//
// Mot cham nhap nhay tren card da xong la animation khong ma hoa thong tin —
// skill muc Anti-pattern goi dung ten: thua thi thay gia. Con chay va cho ma
// nhay cung nhip thi nhin nhip khong phan biet duoc gi, phai doc chu.
func TestNhipThoChiChoThuDangSong(t *testing.T) {
	s := doc2D(t)
	chay := regexp.MustCompile(`\.ag\.run\s+\.st\s+i\{[^}]*animation:[^;}]*?1\.05s`)
	cho := regexp.MustCompile(`\.ag\.pending\s+\.st\s+i\{[^}]*animation:[^;}]*?1\.7s`)
	if !chay.MatchString(s) {
		t.Error("index.html: cham cua phien dang chay khong nhip 1.05s")
	}
	if !cho.MatchString(s) {
		t.Error("index.html: cham cua phien dang cho khong nhip 1.7s (cho phai CHAM hon chay)")
	}
	for _, chet := range []string{".ag.done", ".ag.idle", ".ag.error"} {
		if regexp.MustCompile(regexp.QuoteMeta(chet) + `[^{]*\{[^}]*animation:`).MatchString(s) {
			t.Errorf("index.html: %s cung co animation — thu khong con song thi khong duoc tho", chet)
		}
	}
}

// Nut Dung hien khi RE CHUOT vao card dang song, va cung phai hien khi TAB toi
// no. Thieu :focus-within thi nguoi dung ban phim bam vao mot nut vo hinh —
// no van bam duoc, chi la khong nhin thay minh dang o dau.
func TestNutDungHienKhiHoverVaKhiTabToi(t *testing.T) {
	s := boComment(doc2D(t))
	if !regexp.MustCompile(`\.ag\.song:hover\s+\.dung`).MatchString(s) {
		t.Error("index.html: nut Dung khong gan vao .ag.song:hover — hoac no hien " +
			"suot, hoac no hien ca tren card khong dung duoc")
	}
	if !regexp.MustCompile(`\.ag\.song:focus-within\s+\.dung`).MatchString(s) {
		t.Error("index.html: thieu .ag.song:focus-within .dung — tab toi nut Dung " +
			"thi no van vo hinh")
	}
	// Chi card cua phien dang chay/dang cho moi duoc mang lop .song.
	if !regexp.MustCompile(`className\s*=\s*'ag song `).MatchString(s) {
		t.Error("index.html: JS khong gan lop .song cho card phien — luat hover se " +
			"khong bao gio khop")
	}
}

// Token/chi phi cua PHIEN CLI chua co trong DTO, nen o do phai ghi thang la
// chua do.
//
// /api/state tra ve dung id/addr/pid/worktree/log/started. Dien 0 vao o tokens
// la noi doi theo huong de chiu nhat: 0 doc nhu "phien nay mien phi", trong khi
// su that la no dang tieu han muc ma chua ai dem. Day dung luat "da do — khong
// suy luan" cua skill.
func TestTokenCostCuaPhienGhiChuaDo(t *testing.T) {
	s := doc2D(t)
	ma := boComment(s)
	if !regexp.MustCompile(`const\s+CHUA_DO\s*=\s*'chưa đo'`).MatchString(ma) {
		t.Fatal("index.html: khong co hang CHUA_DO — chu cho o chua co so phai khai MOT cho")
	}
	// Hai o tokens/cost cua card phai dung chinh hang do.
	for _, o := range []string{"tokens", "cost"} {
		re := regexp.MustCompile(`>` + o + `</span><span class="v trong"[^>]*>\$\{CHUA_DO\}`)
		if !re.MatchString(ma) {
			t.Errorf("index.html: o %q cua card khong dien ${CHUA_DO} — kiem xem no co "+
				"dang lap mot so 0 vao cho chua do khong", o)
		}
	}
	// Va tuyet doi khong duoc doc truong khong ton tai tren sessionDTO.
	for _, ma_ := range []string{"s.tok", "s.cost", "s.tokens", "s.usd"} {
		if strings.Contains(ma, ma_) {
			t.Errorf("index.html: doc %s tu phien — sessionDTO khong co truong do, "+
				"no se ra undefined roi hien 'NaN' hoac 'undefined'", ma_)
		}
	}
}

// Meter Tong quan chi duoc an SO THAT, va phai chay bang transition width.
//
// Hai nguon duy nhat co that: /api/flow/detail (costUsd/tokensIn/tokensOut da
// do cua tung buoc) va /api/ai (usage.vao/usage.ra). Khong duoc lay mot han
// muc bia ra lam mau so.
func TestMeterAnSoThatVaChayBangTransitionWidth(t *testing.T) {
	s := doc2D(t)
	if !regexp.MustCompile(`\.meter\s+i\{[^}]*transition:\s*width`).MatchString(s) {
		t.Error("index.html: thanh meter khong co transition:width — nhay so thay vi " +
			"chay thi mat khong bat duoc no vua doi bao nhieu")
	}
	ma := boComment(s)
	if !strings.Contains(ma, "/api/flow/detail?id=") {
		t.Error("index.html: khong doc /api/flow/detail — day la cho DUY NHAT co chi phi " +
			"da do cua tung buoc")
	}
	for _, tr := range []string{"tokensIn", "tokensOut", "costUsd"} {
		if !strings.Contains(ma, "b."+tr) {
			t.Errorf("index.html: khong cong %s tu cac buoc cua lan chay", tr)
		}
	}
	for _, tr := range []string{"usage.vao", "usage.ra"} {
		if !strings.Contains(ma, "d."+tr) {
			t.Errorf("index.html: khong cong %s cua /api/ai vao meter", tr)
		}
	}
	// Dong ghi chu phai noi thang phien CLI khong nam trong so nay — neu khong
	// thi nguoi doc tuong day la tong chi phi ca ham doi.
	if !strings.Contains(s, `id="tq-nguon"`) || !strings.Contains(ma, "Phiên CLI") {
		t.Error("index.html: Tong quan khong ghi ro nguon so va cho con trong " +
			"(phien CLI chua bao token/chi phi)")
	}
}

// Nhat ky: full-width, cap ~60 dong, to theo LOAI, doc xuoi thoi gian, fade-in.
//
// To theo loai chu khong theo muc do nghiem trong: mat can phan biet "vua giao
// viec" voi "vua xong" nhanh hon la phan biet info voi warning.
func TestNhatKyFullWidthToTheoLoaiVaCap60(t *testing.T) {
	s := doc2D(t)
	ma := boComment(s)
	if !regexp.MustCompile(`const\s+LOG_CAP\s*=\s*60\b`).MatchString(ma) {
		t.Error("index.html: khong cap nhat ky o 60 dong — log dashboard la thu de LIEC, " +
			"kho luu tru nam o state.db")
	}
	if !regexp.MustCompile(`function\s+loaiLog\(`).MatchString(ma) {
		t.Fatal("index.html: khong co loaiLog() — khong co cho nao xep su kien vao loai")
	}
	// Phai gan vao TEN SU KIEN THAT trong internal/events, khong phai ten bia.
	for _, e := range []string{"session.started", "flow.completed", "flow.waiting_approval",
		"flow.failed", "warning", "failure"} {
		if !strings.Contains(ma, `'`+e+`'`) {
			t.Errorf("index.html: loaiLog() khong nhac toi su kien that %q", e)
		}
	}
	for _, l := range []string{"#log .e.dispatch", "#log .e.queued", "#log .e.done",
		"#log .e.error", "#log .e.warn"} {
		if !strings.Contains(s, l+"{") {
			t.Errorf("index.html: thieu mau cho loai log %s", l)
		}
	}
	// Doc xuoi: chen o CUOI roi tu cuon xuong, giong moi terminal.
	if !strings.Contains(ma, "box.appendChild(d)") {
		t.Error("index.html: log van chen o dau (insertBefore) — doc nguoc thoi gian")
	}
	if !regexp.MustCompile(`box\.scrollTop\s*=\s*box\.scrollHeight`).MatchString(ma) {
		t.Error("index.html: log khong tu cuon xuong dong moi nhat")
	}
	if !regexp.MustCompile(`@keyframes\s+logvao`).MatchString(s) {
		t.Error("index.html: dong log moi khong fade-in — no nhay ra dot ngot, mat " +
			"khong biet dong nao vua them")
	}
	// Full-width: nhat ky nam NGOAI luoi .bento.
	iBento, iLog := strings.Index(s, `class="bento"`), strings.Index(s, `id="log"`)
	iDuoi := strings.Index(s, `class="duoi"`)
	if iBento < 0 || iLog < 0 || iDuoi < 0 || iLog < iDuoi {
		t.Error("index.html: nhat ky khong nam trong khoi .duoi full-width duoi cung — " +
			"nhet no vao cot phai 340px thi cau nao cung xuong ba dong")
	}
}

// Xuong dien thoai: mot cot duoi 720px. Card tu co nho auto-fill nen chi can
// bo cuc bento go hai cot ra.
func TestMobileMotCotDuoi720(t *testing.T) {
	s := doc2D(t)
	re := regexp.MustCompile(`@media\s*\(\s*max-width\s*:\s*720px\s*\)\s*\{[^}]*\.bento\{[^}]*grid-template-columns\s*:\s*1fr`)
	if !re.MatchString(s) {
		t.Error("index.html: khong co @media(max-width:720px) go .bento ve mot cot")
	}
}

// prefers-reduced-motion phai tat DUNG TEN nhip tho, meter va fade log.
//
// Luat chung (`*{animation-duration:.01ms}`) da co, nhung mot animation
// infinite bi rut con .01ms van chay 1 lan roi dung o khung cuoi — voi
// @keyframes nhiptho thi khung cuoi la opacity 1, may man la on; voi keyframe
// khac thi cham co the dung o trang thai mo. Tat thang ten chac hon.
func TestReducedMotionTatNhipThoMeterVaFade(t *testing.T) {
	s := doc2D(t)
	i := strings.Index(s, "prefers-reduced-motion")
	if i < 0 {
		t.Fatal("index.html: khong co khoi prefers-reduced-motion")
	}
	khoi := s[i:]
	if j := strings.Index(khoi, "</style>"); j > 0 {
		khoi = khoi[:j]
	}
	for _, can := range []string{".ag.run .st i", ".meter i", "#log .e"} {
		if !strings.Contains(khoi, can) {
			t.Errorf("index.html: reduced-motion khong tat %s — chuyen dong ambient phai "+
				"tat het khi may bao giam chuyen dong", can)
		}
	}
}

// Toan bo duong API cu phai con nguyen: dung lai mat 2D khong duoc lam rung
// mot endpoint nao, neu khong thi day la viet lai san pham chu khong phai dung
// lai giao dien.
func TestGiuNguyenMoiDuongAPI(t *testing.T) {
	ma := boComment(doc2D(t))
	for _, d := range []string{"/api/state", "/api/events", "/api/fleet", "/api/stop",
		"/api/quet", "/api/ai", "/api/ai/lich-su", "/api/db", "/api/flow/run", "/api/flows",
		"/api/tele"} {
		if !strings.Contains(ma, `'`+d) {
			t.Errorf("index.html: mat duong %s", d)
		}
	}
	// napFlow() dinh nghia ma khong goi = o chon quy trinh rong tron, va trang
	// van ve day du nen khong ai bao loi. Dung kieu hong du an nay so nhat.
	for _, ham := range []string{"napFlow()", "napRoute()", "napAILichSu()", "napDB()", "napTele()", "connect()"} {
		if !regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(ham) + `\s*;`).MatchString(ma) {
			t.Errorf("index.html: %s duoc dinh nghia nhung khong duoc goi o cap cao nhat", ham)
		}
	}
}

// Mau CHI cho trang thai. Khung (nut, panel, luoi, form) giu don sac, va sac
// HANG chi duoc xuat hien o dung mot cho tren card: dau nhan glyph.
func TestMauChiChoTrangThaiVaSacHangChiOGlyph(t *testing.T) {
	s := doc2D(t)
	if !regexp.MustCompile(`\.ag\s+\.glyph\{[^}]*color:\s*var\(--prov\)`).MatchString(s) {
		t.Error("index.html: glyph khong an sac hang qua var(--prov)")
	}
	// --prov chi duoc gan cho card, va chi glyph doc no.
	if n := strings.Count(s, "var(--prov)"); n != 1 {
		t.Errorf("index.html: var(--prov) xuat hien %d lan — sac HANG la danh tinh, "+
			"chi duoc dung o dung mot cho tren card", n)
	}
	// Rail/aura/pill deu an mau TRANG THAI qua var(--c) chung.
	for _, l := range []string{".ag .rail{", ".ag .aura{", ".ag .st{"} {
		i := strings.Index(s, l)
		if i < 0 {
			continue
		}
		j := strings.Index(s[i:], "}")
		if j > 0 && !strings.Contains(s[i:i+j], "var(--c)") {
			t.Errorf("index.html: %s khong an mau trang thai qua var(--c)", l)
		}
	}
}
