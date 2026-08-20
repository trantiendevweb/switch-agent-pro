package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// vietFlows viết một flows.toml (kèm project.toml, vì flow.Paths chỉ nhận
// flows.toml khi có project.toml bên cạnh) rồi trả về thư mục dự án.
func vietFlows(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".sagent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".sagent", "flows.toml"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".sagent", "project.toml"), []byte("version = 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// (a) Khai `vai_tro` trong flows.toml phải NẠP ĐƯỢC, và bước không khai phải ra
// RỖNG chứ không phải một giá trị đoán hộ.
//
// Sai một chữ trong thẻ toml thì BurntSushi bỏ qua im lặng: vai về rỗng, cả
// phòng agent dồn vào "chưa phân vai", mà không có lỗi nào để lần ra.
func TestNapDuocVaiTroTuTOML(t *testing.T) {
	dir := vietFlows(t, `version = 1
[flow]
  [flow.thu]
    [[flow.thu.step]]
      id = "chia"
      type = "agent"
      vai_tro = "leader"
      prompt = "chia việc"

    [[flow.thu.step]]
      id = "kiem"
      type = "shell"
      vai_tro = "tester"
      run = ["go", "version"]

    [[flow.thu.step]]
      id = "bao"
      type = "notify"
      message = "xong"
`)
	flows, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	f, ok := flows["thu"]
	if !ok {
		t.Fatal("không nạp được flow")
	}
	vai := map[string]string{}
	for _, s := range f.Steps {
		vai[s.ID] = s.VaiTro
	}
	if vai["chia"] != VaiLeader {
		t.Fatalf("khai vai_tro = \"leader\" mà nạp ra %q — vai rơi mất khi đọc TOML", vai["chia"])
	}
	// Vai gắn cho MỌI loại bước, không riêng bước agent.
	if vai["kiem"] != VaiTester {
		t.Fatalf("bước shell cũng phải giữ được vai: được %q", vai["kiem"])
	}
	// (d) ở tầng nạp: không khai thì RỖNG, không tự suy từ type hay tên bước.
	if vai["bao"] != "" {
		t.Fatalf("bước không khai vai_tro phải RỖNG (chưa phân vai), máy lại đoán ra %q", vai["bao"])
	}
}

// Ghi flow ra flows.toml rồi đọc lại phải giữ nguyên vai — bảng vẽ lưu flow
// bằng đúng đường này, mất vai ở đây thì mở lại là vai bay sạch.
func TestVaiTroSongSotQuaLuuVaDocLai(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".sagent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".sagent", "project.toml"), []byte("version = 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := Flow{Name: "thu", Steps: []Step{
		{ID: "gop", Type: TypeAgent, Prompt: "gộp", VaiTro: VaiCEO},
		{ID: "bao", Type: TypeNotify, Message: "xong"},
	}}
	if _, err := Save(dir, f); err != nil {
		t.Fatal(err)
	}
	flows, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := flows["thu"].Steps
	if got[0].VaiTro != VaiCEO {
		t.Fatalf("lưu rồi đọc lại mất vai: được %q", got[0].VaiTro)
	}
	if got[1].VaiTro != "" {
		t.Fatalf("bước không phân vai bỗng có vai sau khi lưu: %q", got[1].VaiTro)
	}
}

// (b) Vai lạ chỉ được CẢNH BÁO, tuyệt đối không phải LỖI — người dùng phải đặt
// được vai mới trước khi công cụ hỗ trợ nó. Và thông điệp phải nêu ĐÚNG giá trị
// sai, kèm đủ năm vai hợp lệ, để sửa được mà không phải đi tra tài liệu.
func TestVaiTroLaChiCanhBaoChuKhongChan(t *testing.T) {
	f := Flow{Name: "thu", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "xong", VaiTro: "designer"},
	}}
	ps := Validate(f)
	var thay *Problem
	for i := range ps {
		if strings.Contains(ps[i].Msg, "vai_tro") {
			thay = &ps[i]
			break
		}
	}
	if thay == nil {
		t.Fatal("vai_tro lạ mà không nói gì — người dùng gõ sai chính tả vai sẽ không bao giờ biết")
	}
	if !thay.Warn {
		t.Fatalf("vai lạ phải là CẢNH BÁO (Warn), không được chặn flow: %+v", *thay)
	}
	if !strings.Contains(thay.Msg, "designer") {
		t.Fatalf("cảnh báo phải nêu đúng giá trị sai: %q", thay.Msg)
	}
	for _, v := range VaiTroHopLe() {
		if !strings.Contains(thay.Msg, v) {
			t.Fatalf("cảnh báo phải liệt kê đủ năm vai hợp lệ, thiếu %q: %q", v, thay.Msg)
		}
	}
	if thay.Step != "a" {
		t.Fatalf("cảnh báo phải chỉ đúng bước, được %q", thay.Step)
	}
	// Không được đẻ ra LỖI nào: flow này ngoài vai lạ ra thì hoàn toàn hợp lệ.
	for _, p := range ps {
		if !p.Warn {
			t.Fatalf("vai lạ không được biến thành lỗi chặn flow: %+v", p)
		}
	}
}

// Năm vai hợp lệ thì im lặng — cảnh báo nhầm còn tệ hơn không cảnh báo, vì
// người đọc sẽ học cách bỏ qua cả cột cảnh báo.
func TestNamVaiHopLeThiKhongCanhBao(t *testing.T) {
	if len(VaiTroHopLe()) != 5 {
		t.Fatalf("phải đúng năm vai, được %v", VaiTroHopLe())
	}
	for _, v := range append(VaiTroHopLe(), "") { // "" = chưa phân vai, cũng hợp lệ
		f := Flow{Name: "thu", Steps: []Step{{ID: "a", Type: TypeNotify, Message: "xong", VaiTro: v}}}
		for _, p := range Validate(f) {
			if strings.Contains(p.Msg, "vai_tro") {
				t.Fatalf("vai %q hợp lệ mà vẫn bị kêu: %s", v, p.Msg)
			}
		}
	}
}

// Flow `doi-4` của chính dự án này phải khai đủ vai — đây là flow duy nhất chạy
// thật hằng ngày, nên nó là chỗ vai trò phải đúng trước tiên. `bao` cố ý để
// RỖNG: bước notify không phải việc của ai cả.
func TestDoi4DaPhanVai(t *testing.T) {
	var file File
	if _, err := toml.DecodeFile(filepath.Join("..", "..", ".sagent", "flows.toml"), &file); err != nil {
		t.Fatal(err)
	}
	f, ok := file.Flows["doi-4"]
	if !ok {
		t.Skip("dự án không có flow doi-4")
	}
	muon := map[string]string{
		"ke-hoach": VaiLeader, "code-go": VaiCoder, "code-doc": VaiCoder,
		"kiem-1": VaiTester, "sua": VaiCoder, "kiem-2": VaiTester,
		"soi": VaiSoi, "gop": VaiCEO, "bao": "",
	}
	for _, s := range f.Steps {
		want, co := muon[s.ID]
		if !co {
			continue
		}
		if s.VaiTro != want {
			t.Fatalf("bước %q: vai_tro = %q, muốn %q", s.ID, s.VaiTro, want)
		}
	}
}
