package flow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// ============================================================================
// BÀI KIỂM QUAN TRỌNG NHẤT CỦA MẢNH NÀY — ĐI HẾT ĐƯỜNG, KHÔNG DỪNG Ở KIỂU
// ============================================================================
//
// Dựng theo mẫu TestSuyLuanDiHetDuongToiKetQua (internal/aiapi/suyluan_test.go),
// và dựng theo mẫu ĐÓ vì một lý do cụ thể: bảng năng lực của nửa API đo phía dự
// án bằng REFLECTION trên kiểu thật, nên nó trả lời được câu "kiểu có trường đó
// không" mà KHÔNG trả lời được câu "có ai chép giá trị đi không". Ngày 22/08 dự
// án bắt được đúng một ca như vậy: `tinNhan.SuyLuan` và `KetQua.SuyLuan` đều có
// mặt, ô `reasoning` in xanh, mà cái cầu ở internal/api thì VỨT phần nghĩ.
//
// Trường `can` có đủ mọi điều kiện để lặp lại cái hỏng đó: nó đi qua BỐN mắt
// xích, và ba mắt đầu đứt được trong im lặng —
//
//	 (1) chữ trong flows.toml   → BurntSushi nạp vào `Step.Can`   (sai thẻ toml = rơi im lặng)
//	 (2) Step.Can               → tham số `can` của ChonRoute      (quên truyền = lọc rỗng)
//	 (3) ChonRoute chọn tên     → output của bước                  (đã có bài kiểm cũ)
//	 (4) output                 → `route` của bước `model`         (đã có bài kiểm cũ)
//
// Nên bài này chạy MỘT LƯỢT THẬT qua Runner.Start, đọc flow từ ĐÚNG một file
// flows.toml trên đĩa, và khẳng định cả bốn mắt — chứ không khẳng định rằng
// `Step` có một trường tên là `Can`.
func TestCanDiHetDuongTuTOMLToiBoChonRoute(t *testing.T) {
	dir := t.TempDir()
	viet(t, dir, `version = 1
[flow]
  [flow.doi]
    [[flow.doi.step]]
      id     = "chon"
      type   = "route"
      routes = ["grok", "deepseek"]
      can    = ["goi-tool", "dau-vao-anh"]

    [[flow.doi.step]]
      id     = "hoi"
      type   = "model"
      needs  = ["chon"]
      route  = "{{steps.chon.output}}"
      can    = ["goi-tool", "dau-vao-anh"]
      prompt = "xem tấm ảnh này rồi gọi tool"
`)

	flows, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	f, ok := flows["doi"]
	if !ok {
		t.Fatal("không nạp được flow")
	}

	// MẮT XÍCH 1 — chữ trong file tới được trường Go. Sai một chữ trong thẻ
	// `toml:"can"` thì BurntSushi bỏ qua IM LẶNG: `can` về nil, bộ lọc không
	// lọc gì, và lượt chạy vẫn xanh trong khi hàng rào không tồn tại.
	if got := f.Steps[0].Can; len(got) != 2 || got[0] != "goi-tool" || got[1] != "dau-vao-anh" {
		t.Fatalf("bước `route`: khai can = [\"goi-tool\", \"dau-vao-anh\"] trong TOML mà nạp ra %v "+
			"— nhu cầu rơi mất ngay ở khâu đọc file, bộ lọc sẽ không lọc gì và không ai biết", got)
	}
	if got := f.Steps[1].Can; len(got) != 2 {
		t.Fatalf("bước `model`: can nạp ra %v", got)
	}

	r, _, db := newRunner(t)
	rg := &routeGia{song: map[string]bool{"deepseek": true}} // grok chết
	mg := &modelGia{}
	r.Route, r.Model = rg, mg

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("lượt chạy không xong: %s", res.State)
	}

	// MẮT XÍCH 2 — và đây là mắt xích MỚI, chưa bài kiểm nào canh. `s.Can` phải
	// tới được tham số của ChonRoute. Quên truyền thì mọi thứ khác vẫn chạy
	// đúng: bước chọn đường vẫn chọn, bước model vẫn hỏi, lượt chạy vẫn xanh —
	// chỉ là chọn theo sức khoẻ y như trước khi có trường này.
	if len(rg.daCan) != 1 {
		t.Fatalf("bộ chọn đường được hỏi %d lần, chờ 1", len(rg.daCan))
	}
	if got := rg.daCan[0]; len(got) != 2 || got[0] != "goi-tool" || got[1] != "dau-vao-anh" {
		t.Fatalf("ChonRoute nhận can = %v — nhu cầu KHÔNG tới được bộ chọn. "+
			"Bước vẫn chạy, lượt vẫn xanh, và phép lọc năng lực chưa bao giờ chạy.", got)
	}

	// MẮT XÍCH 3+4 — tên chọn được đi tới bước `model`. Có bài kiểm riêng
	// (TestRouteChonDuongRoiChuyenChoBuocSau) nhưng khẳng định lại ở đây: bài
	// này mới là bài chạy từ FILE, và một mắt xích đúng trong bài kia không có
	// nghĩa nó còn đúng khi flow đi qua đường nạp TOML.
	if dung := mg.daDung(); len(dung) != 1 || dung[0] != "deepseek" {
		t.Fatalf("bước `model` gọi bằng route %v, chờ [deepseek]", dung)
	}

	steps, err := db.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if out := steps["chon"].Output; out != "deepseek" {
		t.Fatalf("output bước chọn đường = %q — phải là ĐÚNG cái tên, không gì khác", out)
	}
}

// Không khai `can` thì mọi thứ chạy y như trước: bộ chọn nhận danh sách RỖNG,
// không phải một danh sách bịa.
//
// Vì sao đáng một bài riêng: đây là hợp đồng tương thích ngược của cả mảnh. Mọi
// flow đang có đều không khai `can`, và nếu chỗ nào đó nhét một khoá mặc định
// vào thì tất cả chúng đổi hành vi cùng lúc mà không ai gõ một chữ nào.
func TestKhongKhaiCanThiBoChonNhanDanhSachRong(t *testing.T) {
	r, _, _ := newRunner(t)
	rg := &routeGia{song: map[string]bool{"grok": true}}
	r.Route, r.Model = rg, &modelGia{}

	f := Flow{Name: "cu", Steps: []Step{
		{ID: "chon", Type: TypeRoute, Routes: []string{"grok"}},
	}}
	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	if len(rg.daCan) != 1 || len(rg.daCan[0]) != 0 {
		t.Fatalf("bước không khai `can` mà bộ chọn nhận %v — không được bịa nhu cầu hộ người dùng",
			rg.daCan)
	}
}

// ---------------------------- phần soi ----------------------------

// `can` ở một bước không đi qua đường API nào là DÒNG CHẾT — phải là lỗi.
//
// Cùng lớp với `routes` khai ở bước không phải type route: nó nằm đó trông như
// có tác dụng, và người viết flow tưởng bước `shell` của mình đang được canh.
func TestCanKhaiNhamChoLaLoi(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{
		{ID: "chay", Type: TypeShell, Run: []string{"go", "version"}, Can: []string{"goi-tool"}},
	}}
	ps := VanDeCan(f)
	if len(ps) != 1 || ps[0].Warn {
		t.Fatalf("chờ ĐÚNG một LỖI, được %+v", ps)
	}
	if !strings.Contains(ps[0].Msg, "shell") {
		t.Errorf("lời báo không nói type nào đang khai sai: %s", ps[0].Msg)
	}
}

func TestCanRongVaTrungLaLoi(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{
		{ID: "chon", Type: TypeRoute, Can: []string{"goi-tool", "  ", "goi-tool"}},
	}}
	ps := VanDeCan(f)
	if len(ps) != 2 {
		t.Fatalf("chờ 2 lỗi (một mục rỗng, một mục trùng), được %+v", ps)
	}
	for _, p := range ps {
		if p.Warn {
			t.Errorf("phải là lỗi, không phải cảnh báo: %s", p.Msg)
		}
	}
}

// ĐÂY LÀ CHỖ TRẢ GIÁ CHO LỰA CHỌN "KHAI TAY".
//
// Khai tay thêm một chỗ người ta quên. Chỗ dễ quên nhất — và tốn nhất — là:
// khai `can` ở bước `model` mà quên khai ở bước CHỌN ĐƯỜNG của nó. Lúc đó bộ
// chọn chọn theo sức khoẻ, trả về một đường không làm được việc, và bước `model`
// chỉ biết lúc gọi thật, sau khi các bước trước đã tiêu token.
//
// Cảnh báo phải in ra ĐÚNG DÒNG cần thêm, không phải một lời nhắc chung chung:
// người đọc đang ở giữa một file TOML, không phải đang đọc tài liệu.
func TestNhacKhiBuocModelDoiMaBuocChonDuongKhongLoc(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{
		{ID: "chon", Type: TypeRoute, Routes: []string{"grok", "deepseek"}},
		{ID: "hoi", Type: TypeModel, Needs: []string{"chon"},
			Route: "{{steps.chon.output}}", Can: []string{"dau-vao-anh"}, Prompt: "?"},
	}}
	ps := VanDeCan(f)
	if len(ps) != 1 || !ps[0].Warn {
		t.Fatalf("chờ ĐÚNG một cảnh báo, được %+v", ps)
	}
	m := ps[0].Msg
	if !strings.Contains(m, `can = ["dau-vao-anh"]`) {
		t.Errorf("cảnh báo không in ra dòng cần thêm — người đọc phải tự đoán cú pháp: %s", m)
	}
	if !strings.Contains(m, `"chon"`) {
		t.Errorf("cảnh báo không nói thêm vào BƯỚC NÀO: %s", m)
	}
}

// Bước chọn đường có lọc, nhưng THIẾU đúng khoá bước sau cần. Tệ hơn ca trên:
// nhìn vào file thì thấy `can` ở cả hai bước, nên trông như đã được canh.
func TestNhacKhiBuocChonDuongLocThieuKhoa(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{
		{ID: "chon", Type: TypeRoute, Routes: []string{"grok"}, Can: []string{"goi-tool"}},
		{ID: "hoi", Type: TypeModel, Needs: []string{"chon"}, Route: "{{steps.chon.output}}",
			Can: []string{"goi-tool", "dau-vao-anh"}, Prompt: "?"},
	}}
	ps := VanDeCan(f)
	if len(ps) != 1 || !ps[0].Warn {
		t.Fatalf("chờ ĐÚNG một cảnh báo, được %+v", ps)
	}
	m := ps[0].Msg
	if !strings.Contains(m, "dau-vao-anh") {
		t.Errorf("cảnh báo không nói THIẾU khoá nào: %s", m)
	}
	// Dòng gợi ý phải là dòng GỘP, không phải dòng thay thế — thay thế thì
	// người ta dán vào và mất khoá cũ.
	if !strings.Contains(m, `can = ["goi-tool", "dau-vao-anh"]`) {
		t.Errorf("dòng gợi ý phải gộp cả khoá cũ lẫn khoá thiếu: %s", m)
	}
}

// Khai đủ ở cả hai bước thì KHÔNG được kêu. Một bộ soi kêu cả khi người ta làm
// đúng là một bộ soi người ta sẽ thôi đọc.
func TestKhaiDuOCaHaiBuocThiImLang(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{
		{ID: "chon", Type: TypeRoute, Routes: []string{"grok"},
			Can: []string{"goi-tool", "dau-vao-anh"}},
		{ID: "hoi", Type: TypeModel, Needs: []string{"chon"}, Route: "{{steps.chon.output}}",
			Can: []string{"dau-vao-anh"}, Prompt: "?"},
	}}
	if ps := VanDeCan(f); len(ps) != 0 {
		t.Fatalf("khai đúng mà vẫn bị kêu: %+v", ps)
	}
}

// `can` phải hiện ra khi xem flow. Giấu nó đi thì bảng đang NÓI SAI: nó in
// "grok → deepseek" trong khi deepseek đã bị loại trước khi ai hỏi thăm nó.
func TestMoTaRouteMangTheoCan(t *testing.T) {
	s := Step{Type: TypeRoute, Routes: []string{"grok", "deepseek"}, Can: []string{"dau-vao-anh"}}
	got := MoTaRoute(s)
	if !strings.Contains(got, "grok → deepseek") {
		t.Errorf("mất phần thứ tự ứng viên: %q", got)
	}
	if !strings.Contains(got, "dau-vao-anh") {
		t.Fatalf("MoTaRoute = %q — giấu mất `can`, nên bảng đang nói rằng cả hai đường "+
			"đều còn trong cuộc chọn", got)
	}
	// Bước không đòi gì thì KHÔNG được mọc thêm chữ.
	if got := MoTaRoute(Step{Type: TypeRoute, Routes: []string{"grok"}}); got != "grok" {
		t.Errorf("bước không khai `can` mà mô tả mọc thêm chữ: %q", got)
	}
}

// viet ghi một flows.toml (kèm project.toml, vì flow.Paths đòi có nó bên cạnh).
func viet(t *testing.T, dir, src string) {
	t.Helper()
	sg := filepath.Join(dir, ".sagent")
	if err := os.MkdirAll(sg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sg, "flows.toml"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sg, "project.toml"), []byte("version = 1"), 0o644); err != nil {
		t.Fatal(err)
	}
}
