package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// dungHoSoClaude dung mot ho so claude gia trong kho, voi han token cho truoc.
// hetHan < 0 nghia la da qua han.
func dungHoSoClaude(t *testing.T, ten string, con time.Duration) {
	t.Helper()
	dir := filepath.Join(os.Getenv("USERPROFILE"), ".ai-accounts", "claude", ten)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	cred := map[string]any{
		"claudeAiOauth": map[string]any{
			"expiresAt": time.Now().Add(con).UnixMilli(),
		},
	}
	b, _ := json.Marshal(cred)
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func khoTam(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

// Flow mo phong `dem`: hai buoc Claude cung mot tai khoan, xen mot buoc may cham.
func flowDem() flow.Flow {
	return flow.Flow{Name: "dem", Steps: []flow.Step{
		{ID: "code-go", Type: flow.TypeAgent, Profile: "claude:tns", Prompt: "x"},
		{ID: "kiem-1", Type: flow.TypeShell, Run: []string{"go", "version"}},
		{ID: "sua-1", Type: flow.TypeAgent, Profile: "claude:tns", Prompt: "y"},
	}}
}

// CA DA NO THAT (luot chay #29): token claude:tns het han luc 18/08 23:44, luot
// chay bat dau 01:45 — tuc 2 tieng SAU khi token chet. `flow run` van chay, dot
// 9 buoc, 4 buoc chet chac tu dau. Log day buoc do trong khi nguyen nhan that
// gon mot dong: chua dang nhap lai.
func TestChanKhiTokenDaHetHan(t *testing.T) {
	khoTam(t)
	dungHoSoClaude(t, "tns", -2*time.Hour)

	a := &API{}
	hong, err := a.KiemTaiKhoanFlow(flowDem(), Addr{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hong) != 1 {
		t.Fatalf("phai bat dung 1 tai khoan hong, duoc %d: %+v", len(hong), hong)
	}
	if hong[0].Addr != "claude:tns" {
		t.Fatalf("bat nham tai khoan: %s", hong[0].Addr)
	}
	if !strings.Contains(hong[0].LyDo, "hết hạn") {
		t.Fatalf("phai noi ro la het han, duoc %q", hong[0].LyDo)
	}
	// Phai goi ten CA HAI buoc phu thuoc — nguoi doc can biet mat nhung gi.
	if len(hong[0].Buoc) != 2 {
		t.Fatalf("phai liet ke ca 2 buoc phu thuoc, duoc %v", hong[0].Buoc)
	}
}

// Tai khoan khong co ho so tren may (claude:phu hom 19/08) cung phai bi bat,
// khong duoc lang le bo qua.
func TestChanKhiKhongCoHoSo(t *testing.T) {
	khoTam(t)

	a := &API{}
	f := flow.Flow{Name: "x", Steps: []flow.Step{
		{ID: "ke-hoach", Type: flow.TypeAgent, Profile: "claude:phu", Prompt: "x"},
	}}
	hong, err := a.KiemTaiKhoanFlow(f, Addr{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hong) != 1 || hong[0].Addr != "claude:phu" {
		t.Fatalf("phai bat claude:phu khong co ho so, duoc %+v", hong)
	}
}

// Token con han thi KHONG duoc chan — canh bao thua cung la mot kieu noi doi,
// va no day nguoi ta toi cho luon go --cu-chay cho xong.
func TestTokenConHanThiKhongChan(t *testing.T) {
	khoTam(t)
	dungHoSoClaude(t, "tns", 5*time.Hour)

	a := &API{}
	f := flow.Flow{Name: "x", Steps: []flow.Step{
		{ID: "code-go", Type: flow.TypeAgent, Profile: "claude:tns", Prompt: "x"},
	}}
	hong, err := a.KiemTaiKhoanFlow(f, Addr{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hong) != 0 {
		t.Fatalf("token con han ma van chan: %+v", hong)
	}
}

// Buoc khong phai agent (shell, notify) khong dung tai khoan nao — dung bat.
func TestBuocShellVaNotifyKhongCanTaiKhoan(t *testing.T) {
	khoTam(t)

	a := &API{}
	f := flow.Flow{Name: "x", Steps: []flow.Step{
		{ID: "kiem", Type: flow.TypeShell, Run: []string{"go", "version"}},
		{ID: "bao", Type: flow.TypeNotify, Message: "xong"},
	}}
	hong, err := a.KiemTaiKhoanFlow(f, Addr{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hong) != 0 {
		t.Fatalf("buoc khong dung agent ma van doi tai khoan: %+v", hong)
	}
}

// Thong bao phai noi du BA thu: hong cai gi, keo theo buoc nao, sua bang lenh
// nao. Thieu cai thu ba thi nguoi doc van phai di tra.
func TestThongBaoNoiDuBaThu(t *testing.T) {
	msg := loiTaiKhoanHong([]TaiKhoanHong{
		{Addr: "claude:tns", LyDo: "token hết hạn lúc 18/08 23:44", Buoc: []string{"code-go", "sua-1"}},
	}).Error()

	for _, can := range []string{"claude:tns", "hết hạn", "code-go", "sua-1", "sagent claude:tns", "--cu-chay"} {
		if !strings.Contains(msg, can) {
			t.Fatalf("thong bao thieu %q:\n%s", can, msg)
		}
	}
}

// Test QUAN TRONG NHAT cua cong kiem: no phai nam TREN duong chay that.
//
// Kiem rieng ham KiemTaiKhoanFlow la chua du — cho hong nam o CHO GOI. Bo mot
// dong trong FlowRun la cong kiem bien mat trong khi test ham van xanh.
func TestFlowRunChanTruocKhiGhiLanChayNao(t *testing.T) {
	khoTam(t)
	dungHoSoClaude(t, "tns", -2*time.Hour)

	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if _, err := a.FlowSave(dir, flowDem()); err != nil {
		t.Fatal(err)
	}

	_, err = a.FlowRunCuChay(context.Background(), dir, "dem", nil, Addr{}, false)
	if err == nil {
		t.Fatal("token da het han ma flow van chay")
	}
	if !strings.Contains(err.Error(), "claude:tns") {
		t.Fatalf("loi phai goi ten tai khoan hong, duoc: %v", err)
	}

	// Chan TRUOC khi tieu bat cu thu gi: khong duoc de lai lan chay nao trong so.
	runs, err := a.FlowRuns(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatalf("bi chan roi ma van ghi %d lan chay vao so", len(runs))
	}
}

// --cu-chay thi phai di qua duoc cong — nguoi dung da noi ro la ho biet.
func TestCuChayThiDiQuaDuocCong(t *testing.T) {
	khoTam(t)
	dungHoSoClaude(t, "tns", -2*time.Hour)

	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if _, err := a.FlowSave(dir, flowDem()); err != nil {
		t.Fatal(err)
	}

	_, _ = a.FlowRunCuChay(context.Background(), dir, "dem", nil, Addr{}, true)

	// Di qua cong nghia la CO mot lan chay duoc ghi (du no hong sau do).
	runs, err := a.FlowRuns(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) == 0 {
		t.Fatal("--cu-chay ma van bi chan tu cong, khong lan chay nao duoc ghi")
	}
}
