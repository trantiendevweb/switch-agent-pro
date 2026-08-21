package provider

import (
	"encoding/base64"
	"strings"
	"testing"
)

// BAI KIEM CHO O V3 — "khai Duoc ma Identity() luon tra rong".
//
// Vi sao phai co file nay, chep lai tu docs/SO-NO-DO-LUONG.md muc V3: phep do
// NLDanhTinh trong nangluc.go la MOT CHIEU (haiChieu: false). No chi ket luan
// duoc chieu "khai chua do ma lai tra gia tri that". Chieu nguoc lai — "khai
// LAM DUOC ma luon tra rong" — KHONG CO AI CANH, nen KiemNangLuc xanh suot
// trong khi loi khai sai. So no ghi ro cach bit lo: khong phai doi phep do (mot
// chieu la DUNG, vi can ho so that moi do duoc), ma bang mot bai kiem chay tren
// ho so that. Day la bai kiem do.
//
// HINH DANG THAT cua %APPDATA%\Cursor\auth.json — do 21/08/2026 bang cach giai
// payload JWT tren ho so DANG DANG NHAP, ban CLI 2026.08.11-e8db854.
//
// Thu muc %APPDATA%\Cursor chi co DUNG MOT file: auth.json (893 byte). Tang
// ngoai co DUNG HAI khoa accessToken/refreshToken. Payload hai JWT giong het
// nhau, co DUNG TAM claim:
//
//	iss         https://authentication.cursor.sh
//	aud         https://cursor.com
//	sub         google-oauth2|user_01<...>          <- truong danh tinh DUY NHAT
//	scope       openid profile email offline_access
//	type        session
//	randomness  <uuid cut>
//	time        1787022539
//	exp         1792206539
//
// Khong nhet token that vao repo: chu ky khong duoc kiem nen JWT gia la du, va
// mot token that trong ma nguon la mot ro ri. `subGia` giu DUNG dang cua sub
// that (`google-oauth2|user_01…`) nhung id la bia.
const subGia = "google-oauth2|user_01BIAxxxxxxxxxxxxxxxxxxxxx"

// jwtDanhTinhGia dung mot JWT co dung 8 claim nhu file that, cho phep thay sub
// va them claim email de kiem thu tu uu tien.
func jwtDanhTinhGia(sub, email string) string {
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := `{"aud":"https://cursor.com","iss":"https://authentication.cursor.sh",` +
		`"scope":"openid profile email offline_access","type":"session",` +
		`"randomness":"869fef74-9445-4afc","time":1787022539,"exp":1792206539`
	if sub != "" {
		claims += `,"sub":"` + sub + `"`
	}
	if email != "" {
		claims += `,"email":"` + email + `"`
	}
	claims += `}`
	return head + "." + base64.RawURLEncoding.EncodeToString([]byte(claims)) + ".chu-ky-gia-khong-ai-kiem"
}

// authThat dung lai DUNG hinh dang file that: hai khoa, khong gi khac.
func authThat() string {
	t := jwtDanhTinhGia(subGia, "")
	return `{"accessToken":"` + t + `","refreshToken":"` + t + `"}`
}

// O TRUNG TAM CUA V3: tren mot ho so co hinh dang GIONG HET file that, Identity
// phai tra ra MOT CAI GI DO. Truoc ban va no tra "" — trong khi bang nang luc
// khai Duoc(NLDanhTinh). Do la loi khai NOI DOI theo huong khoe, thu ma so no
// xep nguy hiem hon ChuaDo vi loi KHONG chan va nguoi van hanh tuong da kiem.
func TestCursorDocDuocDanhTinhTuJWT(t *testing.T) {
	dir := ghiAuthCursor(t, authThat())

	got := (cursor{}).Identity(dir)
	if got == "" {
		t.Fatal("Identity() tra RONG tren ho so co hinh dang y het file that — " +
			"day dung la o V3: bang nang luc khai Duoc(NLDanhTinh) ma ham luon tra rong")
	}
	if got != subGia {
		t.Fatalf("phai tra claim sub %q, tra %q", subGia, got)
	}
}

// LOI KHAI PHAI KHOP HAM — chieu ma phep do mot chieu cua nangluc.go bo sot.
//
// Bai kiem nay la thu duy nhat trong ca du an bat duoc ca "khai LamDuoc ma ham
// tra rong" cho NLDanhTinh. Ha khai xuong ChuaDo/KhongLamDuoc thi cung phai sua
// Identity cho khop, va nguoc lai — hai thu khong duoc phep noi nguoc nhau.
func TestCursorKhaiDanhTinhKhopVoiHam(t *testing.T) {
	var khai NangLuc
	for _, nl := range (cursor{}).NangLuc() {
		if nl.Khoa == NLDanhTinh {
			khai = nl
		}
	}
	if khai.Khoa == "" {
		t.Fatal("bang nang luc Cursor khong con dong NLDanhTinh nao")
	}

	dir := ghiAuthCursor(t, authThat())
	docDuoc := (cursor{}).Identity(dir) != ""

	if khai.TrangThai == LamDuoc && !docDuoc {
		t.Fatal("khai Duoc(NLDanhTinh) ma Identity() tra rong tren ho so that — " +
			"dung o V3. Hoac sua ham, hoac ha loi khai; khong duoc de lech")
	}
	if khai.TrangThai != LamDuoc && docDuoc {
		t.Fatalf("khai %q ma Identity() doc duoc that — nang loi khai len Duoc",
			khai.TrangThai)
	}
}

// BAY DA TRANH, ghim lai de khong ai roi vao lan nua: claim `scope` CO CHU
// "email" (openid profile email offline_access), nhung do la pham vi OAuth da
// xin, KHONG phai mot claim email. Doc luot thay chu "email" roi khai la doc
// duoc email chinh la kieu sai cu.
func TestCursorKhongNhamScopeLaEmail(t *testing.T) {
	dir := ghiAuthCursor(t, authThat())

	got := (cursor{}).Identity(dir)
	if strings.Contains(got, "openid") || strings.Contains(got, "offline_access") {
		t.Fatalf("tra ra claim scope thay vi danh tinh: %q", got)
	}
	if strings.Contains(got, "@") {
		t.Fatalf("file that KHONG co email ma tra ra %q — hien nham email con te "+
			"hon khong hien gi (bai hoc o V2, antigravity)", got)
	}
}

// Thu tu uu tien: email that LUON hon `sub` duc. Ban CLI 2026.08.11 khong ghi
// email o dau ca, nhung neu ban sau them thi card phai tu doi sang email ma
// khong phai sua ham.
func TestCursorUuTienEmailHonSub(t *testing.T) {
	t.Run("email o tang ngoai", func(t *testing.T) {
		tok := jwtDanhTinhGia(subGia, "")
		dir := ghiAuthCursor(t, `{"email":"ai-do@vi-du.test","accessToken":"`+tok+`"}`)
		if got := (cursor{}).Identity(dir); got != "ai-do@vi-du.test" {
			t.Fatalf("co email o tang ngoai ma tra %q", got)
		}
	})
	t.Run("claim email trong JWT", func(t *testing.T) {
		tok := jwtDanhTinhGia(subGia, "ai-do@vi-du.test")
		dir := ghiAuthCursor(t, `{"accessToken":"`+tok+`"}`)
		if got := (cursor{}).Identity(dir); got != "ai-do@vi-du.test" {
			t.Fatalf("JWT co claim email ma tra %q", got)
		}
	})
}

// Khong co accessToken thi lui ve refreshToken — hai token hom nay giong het
// nhau tung claim, nhung ham khong duoc phu thuoc vao chuyen do.
func TestCursorChiCoRefreshVanDocDuocDanhTinh(t *testing.T) {
	dir := ghiAuthCursor(t, `{"refreshToken":"`+jwtDanhTinhGia(subGia, "")+`"}`)
	if got := (cursor{}).Identity(dir); got != subGia {
		t.Fatalf("chi co refreshToken thi van phai doc duoc sub, tra %q", got)
	}
}

// TUYET DOI KHONG RO RI TOKEN. Identity di thang ra bang `sagent ds` va ra
// dashboard; tra nham chuoi token ra do la ro ri bi mat xac thuc, khong phai
// mot loi hien thi.
func TestCursorDanhTinhKhongBaoGioLaToken(t *testing.T) {
	tok := jwtDanhTinhGia(subGia, "")
	dir := ghiAuthCursor(t, `{"accessToken":"`+tok+`","refreshToken":"`+tok+`"}`)

	got := (cursor{}).Identity(dir)
	if got == tok || strings.Contains(got, ".") && strings.Contains(got, "chu-ky-gia") {
		t.Fatalf("Identity tra ra chinh chuoi token: %q", got)
	}
	if strings.Contains(tok, got) && got != subGia {
		t.Fatalf("Identity tra ra mot mau cua token: %q", got)
	}
}

// MOI NGO HONG DEU PHAI TRA RONG — khong duoc bia ra mot danh tinh. Mot cai ten
// bia khien nguoi van hanh tuong minh dang chay bang tai khoan khac.
func TestCursorNgoHongThiDanhTinhRong(t *testing.T) {
	for ten, noiDung := range map[string]string{
		"file rong":              ``,
		"khong phai json":        `khong-phai-json`,
		"json rong":              `{}`,
		"token khong phai jwt":   `{"accessToken":"khong-co-dau-cham"}`,
		"payload khong base64":   `{"accessToken":"aaa.@@@khong-phai-base64@@@.bbb"}`,
		"payload khong json":     `{"accessToken":"aaa.` + base64.RawURLEncoding.EncodeToString([]byte(`khong-phai-json`)) + `.bbb"}`,
		"khong co sub lan email": `{"accessToken":"` + jwtDanhTinhGia("", "") + `"}`,
		"email rong":             `{"email":"","accessToken":"aaa.` + base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"x"}`)) + `.bbb"}`,
	} {
		t.Run(ten, func(t *testing.T) {
			dir := ghiAuthCursor(t, noiDung)
			if got := (cursor{}).Identity(dir); got != "" {
				t.Fatalf("bia ra danh tinh %q tu ban ghi hong", got)
			}
		})
	}
}

// Khong co file thi khong co danh tinh — va khong duoc hoang bao loi.
func TestCursorKhongCoFileThiDanhTinhRong(t *testing.T) {
	if got := (cursor{}).Identity(t.TempDir()); got != "" {
		t.Fatalf("khong co file ma tra danh tinh %q", got)
	}
}

// ===== O C6: `--workspace`, khong phai nil =====
//
// Tach khoi phan danh tinh o tren, nhung de chung file vi cung mot bai hoc:
// mot LOI KHAI khong khop HAM THAT. Khac o V3 o cho V3 khai thua (Duoc ma tra
// rong), con C6 khai THIEU (Khong ma provider that su lam duoc).

// ArgsThuMuc phai tra `--workspace <dir>`.
//
// Do 21/08/2026 tren ban 2026.08.11-e8db854, CO DOI CHUNG: tao mot file van tay
// trong thu muc dich roi chay tu mot cwd khac han.
//
//	co --workspace <dir>  -> agent liet ke "VAN-TAY-9F3A2B.txt"
//	khong co co           -> "no", va tu khai cwd cu
//
// Loi khai cu Khong(NLThuMuc) dung tren mot cau hoi dat HEP: no hoi "co --cwd
// hay -C khong" (dung la khong co), roi ket luan "provider khong co co doi thu
// muc" — trong khi CUNG BAN --help do co --workspace.
func TestCursorArgsThuMucDungWorkspace(t *testing.T) {
	got := (cursor{}).ArgsThuMuc(`C:\mot\thu\muc`)
	if len(got) == 0 {
		t.Fatal("tra nil — dung o C6: cursor-agent 2026.08.11 CO --workspace, " +
			"do that roi. Khai Khong(NLThuMuc) la ket luan sai, khong phai khoang trong")
	}
	if len(got) != 2 || got[0] != "--workspace" || got[1] != `C:\mot\thu\muc` {
		t.Fatalf("phai la [--workspace <dir>], tra %q", got)
	}
}

// GHIM LUA CHON --workspace CHU KHONG PHAI --add-dir.
//
// Ban 2026.08.11 co CA HAI co. Hop dong ArgsThuMuc (adapter.go:59-66) doi khai
// TUONG MINH thu muc lam viec, ma --add-dir chi "Add an additional workspace
// root" — them mot goc nua, khong phai dat goc. Claude/Antigravity dung
// --add-dir vi CLI cua chung khong co co dat thang; Cursor co.
func TestCursorKhongDungAddDirChoThuMuc(t *testing.T) {
	got := (cursor{}).ArgsThuMuc(`C:\mot\thu\muc`)
	for _, a := range got {
		if a == "--add-dir" {
			t.Fatal("dung --add-dir: no THEM mot goc workspace nua, khong phai dat " +
				"thu muc lam viec. Ban nay co --workspace dung nghia hon")
		}
	}
}

// LOI KHAI PHAI KHOP HAM — nua con lai cua bai hoc V3, ap cho NLThuMuc.
//
// Phep do NLThuMuc trong nangluc.go la HAI CHIEU nen KiemNangLuc da canh duoc ca
// hai huong; test nay ghim them THONG DIEP, de nguoi sua sau doc ra ly do chu
// khong chi thay mot dong lech.
func TestCursorKhaiThuMucKhopVoiHam(t *testing.T) {
	var khai NangLuc
	for _, nl := range (cursor{}).NangLuc() {
		if nl.Khoa == NLThuMuc {
			khai = nl
		}
	}
	if khai.Khoa == "" {
		t.Fatal("bang nang luc Cursor khong con dong NLThuMuc nao")
	}
	coCo := len((cursor{}).ArgsThuMuc(`C:\x`)) > 0

	if khai.TrangThai == KhongLamDuoc && coCo {
		t.Fatal("khai Khong(NLThuMuc) ma ArgsThuMuc tra co that — dung o C6")
	}
	if khai.TrangThai == LamDuoc && !coCo {
		t.Fatal("khai Duoc(NLThuMuc) ma ArgsThuMuc tra nil")
	}
}
