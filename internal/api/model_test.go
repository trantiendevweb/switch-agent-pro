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
//
// BAI TRUOC DUNG PROVIDER THAT (`antigravity`) LAM VAT THU, va do la cho sai:
// ngay 21/08 antigravity va codex deu do xong `--model`/`-m`, nen khong con
// provider that nao tra nil — bai kiem do gay, KHONG phai vi nhanh canh bao
// hong, ma vi no het vat thu. Nhanh canh bao thi VAN phai song: no la thu duy
// nhat dung giua nguoi dung va mot hoa don chay model mac dinh, cho provider
// tiep theo duoc them vao ma chua ai do. Nen vat thu bay gio la adapter GIA.
func TestProviderChuaDoModelThiPhaiCanhBao(t *testing.T) {
	// daDo=false + co=nil: khong dinh vao nhanh canh bao "khong co rao quyen"
	// (nhanh do GHI DE len canhBao), nen cai doc duoc chac chan la canh bao model.
	ad := giaAdapter{ten: "chua-do-model"}
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

// Antigravity va Codex: DA DO 21/08, nen phai truyen co XUONG THAT va KHONG
// duoc canh bao nua. Canh bao thua cung la mot kieu sai: no day nguoi doc di
// kiem mot van de khong ton tai.
//
// Bang chung dung sau hai dong nay nam o comment cua ModelArgs trong
// internal/provider/antigravity.go va internal/provider/codex.go.
func TestAntigravityVaCodexTruyenDuocModel(t *testing.T) {
	for _, tt := range []struct{ ten, co string }{
		{"antigravity", "--model sonnet"},
		{"codex", "-m sonnet"},
	} {
		t.Run(tt.ten, func(t *testing.T) {
			ad, co := provider.Get(tt.ten)
			if !co {
				t.Fatalf("khong co provider %s", tt.ten)
			}
			args, canhBao, err := argsChoBuoc(ad, "sonnet", "lam viec di", false)
			if err != nil {
				t.Fatal(err)
			}
			if canhBao != "" {
				t.Fatalf("%s da do cach chon model, khong duoc canh bao nua: %s", tt.ten, canhBao)
			}
			got := strings.Join(args, " ")
			if !strings.Contains(got, tt.co) {
				t.Fatalf("khong truyen model xuong CLI: %s", got)
			}
			if !strings.Contains(got, "lam viec di") {
				t.Fatalf("mat prompt khi them co model: %s", got)
			}
		})
	}
}

// CHO DE GAY NHAT CUA CODEX: `-m` la co cua LENH CON `exec`, ma argsChoBuoc lai
// CHEN model args VAO TRUOC HeadlessArgs. Dong that vi the la
// `codex -m <model> exec --json <prompt>` — co dung TRUOC lenh con.
//
// Da chay that 21/08 (ban 0.147.0) dung dang do va Codex nhan: dau ban ghi in
// `model: gpt-5.4-mini` thay vi `gpt-5.6-sol` cua config.toml. Bai kiem nay
// khoa lai THU TU do: ngay nao ai doi argsChoBuoc sang append-vao-sau, dong
// lenh thanh `codex exec --json <prompt> -m <model>` va `-m` roi vao vi tri
// doi so cua prompt.
func TestCodexDatCoModelTruocLenhCon(t *testing.T) {
	ad, _ := provider.Get("codex")
	args, _, err := argsChoBuoc(ad, "gpt-5.4-mini", "lam viec di", false)
	if err != nil {
		t.Fatal(err)
	}
	viTri := func(s string) int {
		for i, a := range args {
			if a == s {
				return i
			}
		}
		return -1
	}
	iM, iExec := viTri("-m"), viTri("exec")
	if iM < 0 || iExec < 0 {
		t.Fatalf("thieu -m hoac exec: %v", args)
	}
	if iM > iExec {
		t.Fatalf("-m phai dung TRUOC `exec` (da do that o dang do): %v", args)
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
