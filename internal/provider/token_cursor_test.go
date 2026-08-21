package provider

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// HINH DANG THAT cua %APPDATA%\Cursor\auth.json — do 21/08/2026 tren ho so dang
// nhap that, ban CLI 2026.08.11-e8db854.
//
// File co DUNG HAI khoa, ca hai la JWT alg HS256, va KHONG co truong dau-thoi-
// gian nao o tang ngoai. Claim doc duoc, accessToken va refreshToken giong het
// nhau tung claim mot:
//
//	iss   https://authentication.cursor.sh
//	aud   https://cursor.com
//	scope openid profile email offline_access
//	type  session
//	time  1787022539 = 2026-08-18T03:08:59Z   (luc dang nhap)
//	exp   1792206539 = 2026-10-17T03:08:59Z   (dung 60 ngay sau)
//
// Cac test duoi day dung JWT GIA co dung hinh dang do. Khong nhet token that
// vao repo: chu ky khong duoc kiem nen gia la du, va mot token that trong ma
// nguon la mot ro ri.
func jwtGia(exp int64) string {
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString([]byte(
		`{"aud":"https://cursor.com","iss":"https://authentication.cursor.sh",` +
			`"scope":"openid profile email offline_access","sub":"google-oauth|x",` +
			`"type":"session","time":1787022539,"exp":` + itoa64(exp) + `}`))
	return head + "." + body + ".chu-ky-gia-khong-ai-kiem"
}

// ghiAuthCursor dung dung cay thu muc ma authFile() cho: <hoSo>\Cursor\auth.json.
func ghiAuthCursor(t *testing.T, noiDung string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "Cursor"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Cursor", "auth.json"), []byte(noiDung), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Ca thuong: doc duoc han tu JWT. Truoc ban va, ham nay tra false cung mot cach
// nhu khi KHONG CO FILE — tuc "chua dang nhap" va "da dang nhap, con 57 ngay"
// nhin giong het nhau tu ngoai vao.
func TestCursorDocDuocHanTuJWT(t *testing.T) {
	con := time.Now().Add(60 * 24 * time.Hour).Unix()
	dir := ghiAuthCursor(t, `{"accessToken":"`+jwtGia(con)+`","refreshToken":"`+jwtGia(con)+`"}`)

	exp, ok := (cursor{}).TokenExpiry(dir)
	if !ok {
		t.Fatal("auth.json that co claim exp ma bao khong doc duoc")
	}
	if exp.Unix() != con {
		t.Fatalf("doc nham moc: %d, phai la %d", exp.Unix(), con)
	}
}

// GHIM LUA CHON: doc REFRESH token, khong phai access token.
//
// Tren may nay hai moc DANG BANG NHAU nen chon cai nao cung ra mot so — nghia la
// phep do that KHONG phan biet duoc hai chieu. Test nay phan biet ho no. Bai hoc
// goc o claude.go: tra han access token lam cong kiem CHAN oan luot chay #39,
// trong khi tai khoan van dung duoc vi CLI tu doi token moi.
func TestCursorDocRefreshChuKhongPhaiAccess(t *testing.T) {
	acc := time.Now().Add(2 * time.Hour).Unix()       // access sap het
	ref := time.Now().Add(60 * 24 * time.Hour).Unix() // refresh con 60 ngay
	dir := ghiAuthCursor(t, `{"accessToken":"`+jwtGia(acc)+`","refreshToken":"`+jwtGia(ref)+`"}`)

	exp, ok := (cursor{}).TokenExpiry(dir)
	if !ok {
		t.Fatal("phai doc duoc han")
	}
	if exp.Unix() == acc {
		t.Fatal("doc han ACCESS token — tai khoan van dung duoc ma se bao sap het han, " +
			"dung cai bay da tra gia o claude.go luot chay #39")
	}
	if exp.Unix() != ref {
		t.Fatalf("phai tra han refresh %d, tra %d", ref, exp.Unix())
	}
}

// Khong co refresh thi moc duy nhat biet duoc la han access token.
func TestCursorChiCoAccessThiDungAccess(t *testing.T) {
	acc := time.Now().Add(5 * time.Hour).Unix()
	dir := ghiAuthCursor(t, `{"accessToken":"`+jwtGia(acc)+`"}`)

	exp, ok := (cursor{}).TokenExpiry(dir)
	if !ok {
		t.Fatal("con mot moc doc duoc ma bao khong doc duoc")
	}
	if exp.Unix() != acc {
		t.Fatalf("tra %d, phai la %d", exp.Unix(), acc)
	}
}

// Token HET HAN that thi van phai doc duoc moc — de con noi "het han luc may
// gio" thay vi de trong.
func TestCursorTokenHetHanVanDocDuocMoc(t *testing.T) {
	qua := time.Now().Add(-48 * time.Hour).Unix()
	dir := ghiAuthCursor(t, `{"accessToken":"`+jwtGia(qua)+`","refreshToken":"`+jwtGia(qua)+`"}`)

	if !(cursor{}).HasToken(dir) {
		t.Fatal("token het han van la CO token, khac han chua dang nhap")
	}
	exp, ok := (cursor{}).TokenExpiry(dir)
	if !ok {
		t.Fatal("phai doc duoc moc het han")
	}
	if !time.Now().After(exp) {
		t.Fatalf("token da qua han ma khong bi bat: %s", exp)
	}
}

// MOI NGO HONG DEU PHAI TRA false, khong duoc bia ra mot moc.
//
// Day la nua con lai cua "canh bao sai gio con te hon khong canh bao": mot moc
// bia ra khien nguoi van hanh hoan luot chay vo co, hoac te hon, YEN TAM vao mot
// moc sai. time.Time{} tra kem ok=true se doc thanh "het han tu nam 1".
func TestCursorNgoHongThiTraFalse(t *testing.T) {
	for ten, noiDung := range map[string]string{
		"file rong":            ``,
		"khong phai json":      `khong-phai-json`,
		"json rong":            `{}`,
		"token khong phai jwt": `{"refreshToken":"khong-co-dau-cham"}`,
		"payload khong base64": `{"refreshToken":"aaa.@@@khong-phai-base64@@@.bbb"}`,
		"payload khong json":   `{"refreshToken":"aaa.` + base64.RawURLEncoding.EncodeToString([]byte(`khong-phai-json`)) + `.bbb"}`,
		"khong co claim exp":   `{"refreshToken":"aaa.` + base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"x"}`)) + `.bbb"}`,
		"exp bang 0":           `{"refreshToken":"` + jwtGia(0) + `"}`,
	} {
		t.Run(ten, func(t *testing.T) {
			dir := ghiAuthCursor(t, noiDung)
			if exp, ok := (cursor{}).TokenExpiry(dir); ok {
				t.Fatalf("bia ra mot moc tu ban ghi hong: %s", exp)
			}
		})
	}
}

// Khong co file thi khong co token — va khong duoc hoang bao loi.
func TestCursorKhongCoFileThiKhongDocDuocHan(t *testing.T) {
	dir := t.TempDir()
	if (cursor{}).HasToken(dir) {
		t.Fatal("khong co file ma bao co token")
	}
	if _, ok := (cursor{}).TokenExpiry(dir); ok {
		t.Fatal("khong co file ma bao doc duoc han")
	}
}
