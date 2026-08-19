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

// CA DA NO THAT — do 2026-08-19 19:30, CLI 2.1.229.
//
// Mot lan dang nhap MOI ghi ra `expiresAt: 0` kem `refreshTokenExpiresAt`.
// Ban cu chi doc expiresAt, gap 0 thi tra "khong doc duoc han" -> Profile.HetHan
// mac dinh false -> `sagent ds` in "san sang" VO DIEU KIEN cho mot tai khoan ma
// no khong he biet con song hay khong. Va cong kiem tai khoan cua `flow run`
// cung mu theo, vi no dua tren dung truong do.
//
// Day la nguyen van hinh dang do duoc (token da thay bang chu gia).
const credHinhDangMoi = `{"claudeAiOauth":{
  "accessToken":"gia-lap","refreshToken":"gia-lap",
  "expiresAt":0,
  "refreshTokenExpiresAt":1789617873859,
  "scopes":"user:inference user:profile",
  "subscriptionType":"max","rateLimitTier":"default_claude_max_20x"}}`

func TestDocDuocHanKhiExpiresAtLaSo0(t *testing.T) {
	dir := ghiCred(t, credHinhDangMoi)

	exp, ok := claude{}.TokenExpiry(dir)
	if !ok {
		t.Fatal("expiresAt=0 nhung co refreshTokenExpiresAt — phai doc duoc han, khong duoc bo cuoc")
	}
	muon := time.UnixMilli(1789617873859)
	if !exp.Equal(muon) {
		t.Fatalf("han sai: duoc %s, muon %s", exp, muon)
	}
	// 17/09/2026 la tuong lai so voi luc do -> tai khoan con dung duoc.
	if time.Now().After(exp) {
		t.Fatalf("moc %s bi coi la da qua — kiem lai don vi mili-giay", exp)
	}
}

// Hinh dang CU (expiresAt that) phai giu nguyen hanh vi, va phai duoc UU TIEN:
// no la han cua access token, sat hon refresh token.
func TestExpiresAtThatVanDuocUuTien(t *testing.T) {
	dir := ghiCred(t, `{"claudeAiOauth":{"expiresAt":1755000000000,"refreshTokenExpiresAt":1789617873859}}`)

	exp, ok := claude{}.TokenExpiry(dir)
	if !ok {
		t.Fatal("expiresAt that ma khong doc duoc")
	}
	if !exp.Equal(time.UnixMilli(1755000000000)) {
		t.Fatalf("phai uu tien expiresAt, duoc %s", exp)
	}
}

// Khong truong nao doc duoc thi phai noi THAT la khong biet. Doan bua theo huong
// lac quan ("chac con han") chinh la loi ma commit 0bcb903 da sua mot lan roi.
func TestKhongTruongNaoThiNoiKhongBiet(t *testing.T) {
	dir := ghiCred(t, `{"claudeAiOauth":{"accessToken":"gia-lap"}}`)

	if _, ok := (claude{}).TokenExpiry(dir); ok {
		t.Fatal("khong co truong han nao ma van bao doc duoc")
	}
}

// Token da het han that (hinh dang moi) phai bi bat.
func TestRefreshTokenHetHanThiBiBat(t *testing.T) {
	qua := time.Now().Add(-48 * time.Hour).UnixMilli()
	dir := ghiCred(t, `{"claudeAiOauth":{"expiresAt":0,"refreshTokenExpiresAt":`+itoa64(qua)+`}}`)

	exp, ok := claude{}.TokenExpiry(dir)
	if !ok {
		t.Fatal("phai doc duoc han")
	}
	if !time.Now().After(exp) {
		t.Fatalf("refresh token da qua han ma khong bi bat: %s", exp)
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
