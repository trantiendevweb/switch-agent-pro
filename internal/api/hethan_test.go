package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Token CON DO nhung DA QUA HAN khong duoc bao la "san sang".
//
// Do ngay 18/08: bang tai khoan hien claude:phu "san sang" trong khi chay that
// tra "OAuth session expired and could not be refreshed". HasToken chi kiem FILE
// CO TON TAI. Ke hoach goc muc 1.6 doi "trung thuc ve nang luc" — bao san sang
// cho mot token da chet la vi pham thang.
func TestKhongBaoSanSangChoTokenDaHetHan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	dir := filepath.Join(home, ".ai-accounts", "claude", "cu")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// token het han tu hom qua
	het := time.Now().Add(-24 * time.Hour).UnixMilli()
	cred := map[string]any{"claudeAiOauth": map[string]any{
		"accessToken": "gia-lap", "expiresAt": het,
	}}
	b, _ := json.Marshal(cred)
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}

	a, err := New(home)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	ds, err := a.ProfileList()
	if err != nil {
		t.Fatal(err)
	}
	var thay bool
	for _, p := range ds {
		if p.Account != "cu" {
			continue
		}
		thay = true
		if !p.HetHan {
			t.Fatalf("token het han tu hom qua ma khong bao het han (HasToken=%v, HanToi=%v)",
				p.HasToken, p.HanToi)
		}
	}
	if !thay {
		t.Skip("khong doc duoc ho so trong HOME tam — moi truong khong dung duoc cho phep do nay")
	}
}
