package api

import (
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// CA CO SO DO (luot chay #34): ca luot ton 9,40 USD, rieng buoc `code-go` la
// 8,18 USD — vi MOI buoc deu chay model manh nhat, ke ca buoc chi viet tai lieu
// hay gop bao cao. Chon model theo tung buoc la thu bien so tien do thanh chon
// duoc.
func TestChonModelChoTungBuoc(t *testing.T) {
	ad, co := provider.Get("claude")
	if !co {
		t.Fatal("khong co provider claude")
	}
	args, canhBao, err := argsChoBuoc(ad, "sonnet", "lam viec di", false)
	if err != nil {
		t.Fatal(err)
	}
	if canhBao != "" {
		t.Fatalf("claude do duoc cach chon model, khong duoc canh bao: %s", canhBao)
	}
	got := strings.Join(args, " ")
	if !strings.Contains(got, "--model sonnet") {
		t.Fatalf("khong truyen model xuong CLI: %s", got)
	}
	// Prompt phai con nguyen — them co chon model khong duoc nuot mat viec.
	if !strings.Contains(got, "lam viec di") {
		t.Fatalf("mat prompt khi them co model: %s", got)
	}
}

// Khong khai model thi khong duoc tu them co — de provider dung mac dinh cua no.
func TestKhongKhaiModelThiKhongThemCo(t *testing.T) {
	ad, _ := provider.Get("claude")
	args, _, err := argsChoBuoc(ad, "", "lam viec di", false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(args, " "), "--model") {
		t.Fatalf("tu them --model du khong ai yeu cau: %v", args)
	}
}

// QUAN TRONG: provider CHUA DO cach chon model thi phai NOI THANG.
//
// Im lang bo qua la kieu hong te nhat o day: nguoi dung khai model = sonnet de
// tiet kiem, thay lenh chay binh thuong, tuong minh vua tiet kiem duoc — ma that
// ra van dot model dat nhat. Ho chi biet khi doc hoa don.
func TestProviderChuaDoModelThiPhaiCanhBao(t *testing.T) {
	ad, co := provider.Get("antigravity")
	if !co {
		t.Fatal("khong co provider antigravity")
	}
	args, canhBao, err := argsChoBuoc(ad, "sonnet", "lam viec di", false)
	if err != nil {
		t.Fatal(err)
	}
	if canhBao == "" {
		t.Fatal("provider chua do cach chon model ma im lang — nguoi dung se tuong minh da tiet kiem duoc")
	}
	if !strings.Contains(canhBao, "sonnet") {
		t.Fatalf("canh bao phai noi ro model nao bi bo qua: %q", canhBao)
	}
	// Van phai chay duoc, chi la chay model mac dinh.
	if len(args) == 0 {
		t.Fatal("chua do model thi van phai chay duoc, khong duoc chan")
	}
}

// Grok BAT BUOC co -m (README: CLI bo qua defaultModel trong file cau hinh cua
// chinh no, endpoint khong ban model dung san se tra 503).
func TestGrokNhanCoModelRieng(t *testing.T) {
	ad, _ := provider.Get("grok")
	args, _, err := argsChoBuoc(ad, "grok-4.5", "lam viec di", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(args, " "), "-m grok-4.5") {
		t.Fatalf("grok phai nhan -m: %v", args)
	}
}
