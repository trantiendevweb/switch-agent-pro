package provider

import (
	"fmt"
	"strings"
	"testing"
)

// Ba nhom test, ba cau hoi khac nhau:
//  1. quan THAT thi co bat duoc khong,
//  2. lap lai BINH THUONG thi co bat oan khong,
//  3. thieu du lieu thi co noi thang la KHONG BIET khong.
//
// Cau 2 va 3 moi la phan kho. Bat duoc quan ma bat oan ca buoc lam viec that
// thi nguoi dung se hoc cach bo qua canh bao, luc do la chan coi nhu khong co.

// dongToolClaude dung mot dong assistant mang MOT loi goi tool, dung khuon
// stream-json cua Claude.
func dongToolClaude(ten, lenh string) string {
	return fmt.Sprintf(
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":%q,"input":{"command":%q,"description":"lam gi do"}}]}}`,
		ten, lenh)
}

func banGhi(dong ...string) string { return strings.Join(dong, "\n") }

// ---------------------------------------------------------------- 1. QUAN THAT

// Nguyen van ca da no: lan chay #21, buoc `soi` chay bang grok goi `ls -la` 399
// lan lien tiep roi moi bi tran --max-tool-rounds chan. Buoc van duoc danh dau
// `done`, khong ai thay gi bat thuong.
//
// Dung 399 dong that chu khong dung con so gia, de test noi dung ca da xay ra.
func TestBatDuocQuanThat(t *testing.T) {
	var dong []string
	dong = append(dong, `{"type":"system","subtype":"init"}`)
	for i := 0; i < 399; i++ {
		dong = append(dong, dongToolClaude("Bash", "ls -la"))
	}
	dong = append(dong, ketQuaCutVong)

	k := doc(t, banGhi(dong...))
	if !k.DemDuocTool {
		t.Fatal("doc duoc loi goi tool ma bao khong doc duoc")
	}
	if k.SoLanLap != 399 {
		t.Fatalf("chuoi lap dai nhat = %d, muon 399", k.SoLanLap)
	}
	ly, biet := k.Quan()
	if !biet {
		t.Fatal("co du lieu ma bao khong biet")
	}
	if ly == "" {
		t.Fatal("399 lan `ls -la` lien tiep ma khong coi la quan")
	}
	// Loi bao phai NOI RO ba thu: lenh nao, bao nhieu lan, va day khong phai loi
	// code. Thieu cai thu ba thi nguoi doc (hoac buoc sau) se di tim bug trong
	// san pham, dung cho lan chay #21 da mat mot luot vao viec do.
	for _, can := range []string{"ls -la", "399", "quẩn", "KHÔNG phải lỗi code"} {
		if !strings.Contains(ly, can) {
			t.Fatalf("thong bao thieu %q: %s", can, ly)
		}
	}
	// Va no phai noi ra o Hong(), khong chi nam trong truong.
	if h := k.Hong(); !strings.Contains(h, "ls -la") {
		t.Fatalf("Hong() phai bao quan, duoc %q", h)
	}
}

// Ca NGUY HIEM NHAT: quan xong van nan ra duoc mot cau tra loi, is_error=false,
// nhin moi mat deu nhu thanh cong. Neu chi xet quan khi luot da hong thi ca nay
// lot luoi va buoc sau xay tiep tren rac.
func TestQuanNhungLuotVANBAOTHANHCONG(t *testing.T) {
	var dong []string
	for i := 0; i < 60; i++ { // 60 = dung tran vong tool dang dat cho grok
		dong = append(dong, dongToolClaude("Bash", "ls -la"))
	}
	dong = append(dong, ketQuaThatXong) // is_error=false, result="OK"

	k := doc(t, banGhi(dong...))
	if k.TraLoi != "OK" {
		t.Fatalf("cau tra loi = %q", k.TraLoi)
	}
	if k.Hong() == "" {
		t.Fatal("quan 60 vong ma van bao la luot chay tot")
	}
}

// Loi API van la duong uu tien cao nhat: no cu the hon, va no la thu nguoi dung
// phai xu ly truoc.
func TestLoiAPIVanDungTruocQuan(t *testing.T) {
	k := KetQua{
		CoLoi: true, LoiAPI: "OAuth session expired", Loai: "error_during_execution",
		DemDuocTool: true, LenhLap: "Bash ls -la", SoLanLap: 399,
	}
	if h := k.Hong(); !strings.Contains(h, "OAuth session expired") {
		t.Fatalf("phai uu tien loi API, duoc %q", h)
	}
}

// -------------------------------------------------- 2. LAP LAI BINH THUONG

// `git status` chay nhieu lan trong mot buoc la BINH THUONG: truoc khi sua, sau
// moi commit. Cai khac quan la GIUA nhung lan do agent con goi tool khac —
// chuoi bi ngat. Buoc nay lam viec that, khong duoc dung no lai.
func TestGitStatusNhieuLanKhongPhaiQuan(t *testing.T) {
	var dong []string
	for i := 0; i < 20; i++ { // 20 lan `git status` trong mot luot
		dong = append(dong,
			dongToolClaude("Bash", "git status --short"),
			dongToolClaude("Edit", "sua file"),
			dongToolClaude("Bash", "git commit -m x"),
		)
	}
	dong = append(dong, ketQuaThatXong)

	k := doc(t, banGhi(dong...))
	if k.SoLanLap != 1 {
		t.Fatalf("khong co hai loi goi giong nhau lien tiep, chuoi phai la 1, duoc %d (%q)",
			k.SoLanLap, k.LenhLap)
	}
	if ly, _ := k.Quan(); ly != "" {
		t.Fatalf("bat oan mot buoc lam viec that: %s", ly)
	}
	if h := k.Hong(); h != "" {
		t.Fatalf("luot chay tot ma bao hong: %s", h)
	}
}

// Cung MOT tool, THAM SO KHAC NHAU, 50 lan lien tiep: day la doc 50 file, tuc
// dang lam viec. Neu chu ky chi lay ten tool thi ca nay bi ket toi oan.
func TestCungToolKhacThamSoKhongPhaiQuan(t *testing.T) {
	var dong []string
	for i := 0; i < 50; i++ {
		dong = append(dong, dongToolClaude("Bash", fmt.Sprintf("git show HEAD~%d --stat", i)))
	}
	dong = append(dong, ketQuaThatXong)

	k := doc(t, banGhi(dong...))
	if ly, _ := k.Quan(); ly != "" {
		t.Fatalf("50 lenh KHAC NHAU ma bi coi la quan: %s", ly)
	}
}

// Thu lai sau loi tam thoi (mang chap, file bi khoa) la ly do chinh dang duy
// nhat de lap lien tiep. Ngay duoi nguong thi phai tha.
func TestThuLaiVaiLanThiThaChoQua(t *testing.T) {
	var dong []string
	for i := 0; i < TranLapLienTiep-1; i++ {
		dong = append(dong, dongToolClaude("Bash", "go build ./..."))
	}
	dong = append(dong, ketQuaThatXong)

	k := doc(t, banGhi(dong...))
	if k.SoLanLap != TranLapLienTiep-1 {
		t.Fatalf("dem sai: %d", k.SoLanLap)
	}
	if ly, biet := k.Quan(); ly != "" || !biet {
		t.Fatalf("ngay duoi nguong phai tha, duoc %q (biet=%v)", ly, biet)
	}
}

// Truong mo ta tu do (`description`) do model tu viet lai moi lan goi. Neu no
// lot vao chu ky thi 399 lan `ls -la` thanh 399 chu ky khac nhau va la chan roi
// IM LANG — kieu hong te nhat vi no trong y het nhu dang hoat dong.
func TestMoTaDoiKhongLamRoiLaChan(t *testing.T) {
	var dong []string
	for i := 0; i < 30; i++ {
		dong = append(dong, fmt.Sprintf(
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls -la","description":"thu lan thu %d"}}]}}`, i))
	}
	dong = append(dong, ketQuaThatXong)

	k := doc(t, banGhi(dong...))
	if k.SoLanLap != 30 {
		t.Fatalf("mo ta doi tung lan khong duoc lam dut chuoi lap, duoc %d", k.SoLanLap)
	}
}

// ------------------------------------------------------- 3. THIEU DU LIEU

// Antigravity: ban ghi do duoc chi mang `tool_name`, KHONG mang tham so. 30
// buoc `run_command` co the la 30 lenh khac nhau (dang lam viec) hoac mot lenh
// lap 30 lan (quan) — khong phan biet duoc. Phai noi KHONG BIET, khong duoc noi
// "khong quan", va tuyet doi khong duoc ket toi.
func TestAntigravityThieuThamSoThiNoiKhongBiet(t *testing.T) {
	var dong []string
	for i := 0; i < 30; i++ {
		dong = append(dong, `{"event":"step_update","step_update":{"state":"DONE","step_type":"tool","tool_name":"run_command"}}`)
	}
	dong = append(dong, `{"event":"result","result":{"status":"SUCCESS","response":"xong","num_turns":30}}`)

	k, ok := docKetQuaAntigravity(banGhi(dong...))
	if !ok {
		t.Fatal("khong doc duoc dong result cua antigravity")
	}
	if k.DemDuocTool {
		t.Fatal("khong co tham so ma van bao dem duoc loi goi tool")
	}
	ly, biet := k.Quan()
	if biet {
		t.Fatal("thieu tham so ma van dam ket luan — phai noi KHONG BIET")
	}
	if ly != "" {
		t.Fatalf("khong du du lieu ma van ket toi: %s", ly)
	}
	if h := k.Hong(); h != "" {
		t.Fatalf("khong duoc bien 'khong biet' thanh 'hong': %s", h)
	}
}

// Provider chua doc duoc ket qua co cau truc (grok, codex, cursor): KetQua rong
// thi Quan() cung phai noi KHONG BIET. Day dung la ca cua grok — tro tre o cho
// ca quan do duoc lai den tu provider ma ta chua doc duoc du lieu co cau truc,
// nen voi no tran 60 vong van la thu duy nhat cuu.
func TestKetQuaRongThiKhongDamKetLuan(t *testing.T) {
	var k KetQua
	if _, biet := k.Quan(); biet {
		t.Fatal("KetQua rong ma dam ket luan la khong quan")
	}
}

// Ban ghi Claude khong co loi goi tool nao (agent chi tra loi bang chu): doc
// duoc that, nhung khong co gi de dem — cung la KHONG BIET, khong phai "sach".
func TestClaudeKhongGoiToolNaoThiCungLaKhongBiet(t *testing.T) {
	k := doc(t, banGhi(`{"type":"assistant","message":{"content":[{"type":"text","text":"chao"}]}}`, ketQuaThatXong))
	if k.DemDuocTool {
		t.Fatal("khong co loi goi tool nao ma bao dem duoc")
	}
	if _, biet := k.Quan(); biet {
		t.Fatal("khong dem duoc loi goi nao ma dam ket luan")
	}
}
