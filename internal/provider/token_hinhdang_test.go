package provider

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func ghiCred(t *testing.T, noiDung string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), []byte(noiDung), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// CA DA NO THAT — do 19/08 19:30, sau mot lan dang nhap DO DANG.
//
// File .credentials.json duoc ghi ra DAY DU TRUONG: accessToken, refreshToken,
// scopes, subscriptionType "max", refreshTokenExpiresAt cua thang sau. Chi khac
// dung mot cho: `expiresAt: 0`. Hoi thang CLI tren dung ho so do:
//
//	claude auth status  ->  {"loggedIn": false, "authMethod": "none"}
//
// Ban cu chi kiem FILE CO TON TAI nen bao "san sang", cong kiem tai khoan cho
// luot chay #31 di qua, roi buoc code-go chet ngay voi "OAuth session expired
// and could not be refreshed".
const credDangNhapDoDang = `{"claudeAiOauth":{
  "accessToken":"gia-lap","refreshToken":"gia-lap",
  "expiresAt":0,
  "refreshTokenExpiresAt":1789617873859,
  "scopes":"user:inference user:profile",
  "subscriptionType":"max","rateLimitTier":"default_claude_max_20x"}}`

func TestDangNhapDoDangKhongDuocTinhLaCoToken(t *testing.T) {
	dir := ghiCred(t, credDangNhapDoDang)

	if (claude{}).HasToken(dir) {
		t.Fatal("expiresAt=0 la token khong dung duoc — khong duoc tinh la da dang nhap")
	}
}

// Doi chung: tai khoan gocDANG CHAY DUOC luon co expiresAt khac 0.
func TestTokenThatThiCoExpiresAtKhac0(t *testing.T) {
	con := time.Now().Add(3 * time.Hour).UnixMilli()
	dir := ghiCred(t, `{"claudeAiOauth":{"expiresAt":`+itoa64(con)+`,"refreshTokenExpiresAt":1789617873859}}`)

	if !(claude{}).HasToken(dir) {
		t.Fatal("token that ma bao la khong co")
	}
	exp, ok := (claude{}).TokenExpiry(dir)
	if !ok {
		t.Fatal("khong doc duoc han cua token that")
	}
	if time.Now().After(exp) {
		t.Fatalf("han %s bi coi la da qua — kiem lai don vi mili-giay", exp)
	}
}

// Token het han that thi van phai doc duoc han (de con noi "het han luc may
// gio"), va van tinh la CO token — day la chuyen khac han "chua dang nhap".
func TestTokenHetHanVanDocDuocMoc(t *testing.T) {
	qua := time.Now().Add(-48 * time.Hour).UnixMilli()
	dir := ghiCred(t, `{"claudeAiOauth":{"expiresAt":`+itoa64(qua)+`}}`)

	if !(claude{}).HasToken(dir) {
		t.Fatal("token het han van la CO token, khac han chua dang nhap")
	}
	exp, ok := (claude{}).TokenExpiry(dir)
	if !ok {
		t.Fatal("phai doc duoc moc het han")
	}
	if !time.Now().After(exp) {
		t.Fatalf("token da qua han ma khong bi bat: %s", exp)
	}
}

// Khong co file thi khong co token — va khong duoc hoang bao loi.
func TestKhongCoFileThiKhongCoToken(t *testing.T) {
	dir := t.TempDir()
	if (claude{}).HasToken(dir) {
		t.Fatal("khong co file ma bao co token")
	}
	if _, ok := (claude{}).TokenExpiry(dir); ok {
		t.Fatal("khong co file ma bao doc duoc han")
	}
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
