package api

import (
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// flowBaTruong dùng cả ba mảnh vừa trộn vào main: artifact, idempotent,
// compensate — cộng thêm node `route` và `merge` của bản này.
func flowBaTruong() flow.Flow {
	return flow.Flow{
		Name: "batruong", Desc: "flow dùng cả ba trường mới",
		Steps: []flow.Step{
			{ID: "chon", Type: flow.TypeRoute, Routes: []string{"grok", "deepseek"}},
			{ID: "viet", Type: flow.TypeAgent, Profile: "claude:tns", Prompt: "viết bản vá",
				Artifact: map[string]string{"ban-va": "ban-va.diff"}},
			{ID: "hoi", Type: flow.TypeModel, Needs: []string{"chon"},
				Route: "{{steps.chon.output}}", Prompt: "hỏi một câu", Idempotent: true},
			{ID: "ap", Type: flow.TypeShell, Needs: []string{"viet"},
				Run:       []string{"git", "apply", "{{artifacts.ban-va}}"},
				OnFailure: flow.OnFailCompensate, Compensate: "go-lai"},
			// Bước gỡ lại: KHÔNG có `needs`, tức là một GỐC của DAG. Chính vì
			// thế nó nổi lên ngay đợt đầu của kế hoạch và bảng cũ hiện nó như
			// một bước sắp chạy.
			{ID: "go-lai", Type: flow.TypeAgent, Profile: "claude:tns", Prompt: "gỡ lại việc của {{buoc_hong}}"},
			{ID: "gop", Type: flow.TypeMerge, Needs: []string{"hoi", "ap"}},
		},
	}
}

// ĐÂY LÀ BÀI TEST QUAN TRỌNG NHẤT của việc 3.
//
// Bảng chạy khan cũ hiện bước GỠ LẠI y hệt một bước sắp chạy: nó nằm trong đợt,
// có tài khoản, có prompt, và số agent của nó được cộng vào tổng "sắp đốt bao
// nhiêu phiên". Sự thật thì bộ thực thi ghi thẳng nó là `skipped` ngay đầu lượt
// (runner.execute) và chỉ gọi khi có sự cố.
//
// Sai theo hướng THỪA, và thừa ở đây không vô hại: người đọc thấy một bước dọn
// dẹp trong kế hoạch nên yên tâm rằng việc dọn sẽ xảy ra.
func TestChayKhoNoiRoBuocChiChayKhiHong(t *testing.T) {
	khoTam(t)
	dungHoSoClaude(t, "tns", 2*time.Hour)
	dir := t.TempDir()
	if _, err := flow.Save(dir, flowBaTruong()); err != nil {
		t.Fatal(err)
	}

	a := &API{}
	kh, err := a.FlowChayKho(dir, "batruong", nil, Addr{})
	if err != nil {
		t.Fatal(err)
	}

	goLai := timBuocKho(t, kh, "go-lai")
	if len(goLai.GoLaiCho) == 0 {
		t.Fatal("bước `go-lai` KHÔNG được đánh dấu là bước gỡ lại — bảng chạy khan đang " +
			"nói nó sẽ chạy, trong khi nó chỉ chạy khi có sự cố")
	}
	if goLai.GoLaiCho[0] != "ap" {
		t.Errorf("gỡ lại cho bước %v, muốn [ap]", goLai.GoLaiCho)
	}
	if kh.SoBuocGoLai != 1 {
		t.Errorf("SoBuocGoLai = %d, muốn 1", kh.SoBuocGoLai)
	}

	// Chiều ngược lại: bước hỏng phải nói ra nó có bước gỡ.
	if got := timBuocKho(t, kh, "ap").Compensate; got != "go-lai" {
		t.Errorf("bước `ap` không nói ra bước gỡ lại của nó: %q", got)
	}

	// Số agent: chỉ `viet` được đếm. `go-lai` cũng là bước agent nhưng KHÔNG
	// chạy trong lượt suôn sẻ, nên cộng nó vào là trả lời sai câu đang hỏi.
	if kh.SoAgent != 1 {
		t.Fatalf("tổng số agent = %d, muốn 1 — bước gỡ lại KHÔNG được tính vào "+
			"con số \"sắp đốt bao nhiêu phiên\"", kh.SoAgent)
	}
}

// artifact / idempotent / route / gộp phải LỘ RA trong kế hoạch.
//
// Cả ba trường đầu đổi hành vi lượt chạy theo cách không suy được từ sơ đồ phụ
// thuộc: artifact là đường truyền thứ hai (không qua {{steps.x.output}}),
// idempotent nghĩa là bước có thể không chạy, route quyết định đi nhà cung cấp
// nào. Bảng không nói thì người đọc quyết định trên một kế hoạch thiếu.
func TestChayKhoHienArtifactIdempotentRoute(t *testing.T) {
	khoTam(t)
	dungHoSoClaude(t, "tns", 2*time.Hour)
	dir := t.TempDir()
	if _, err := flow.Save(dir, flowBaTruong()); err != nil {
		t.Fatal(err)
	}

	a := &API{}
	kh, err := a.FlowChayKho(dir, "batruong", nil, Addr{})
	if err != nil {
		t.Fatal(err)
	}

	if got := timBuocKho(t, kh, "viet").Artifact["ban-va"]; got != "ban-va.diff" {
		t.Errorf("kế hoạch không nói bước `viet` để lại file nào: %q", got)
	}
	if !timBuocKho(t, kh, "hoi").Idempotent {
		t.Error("kế hoạch không nói bước `hoi` có thể BỊ BỎ QUA vì lượt trước đã làm")
	}
	if got := timBuocKho(t, kh, "chon").Route; got != "grok → deepseek" {
		t.Errorf("kế hoạch không nói bước chọn đường xét những đường nào: %q", got)
	}
	if got := timBuocKho(t, kh, "hoi").Route; got == "" {
		t.Error("kế hoạch không nói bước `model` đi đường nào")
	}
	// Node `model` phải hiện CÂU HỎI. Trước bản này nó rơi vào nhánh `default`
	// của buocKho, tức bảng đọc `s.Message` — một trường luôn rỗng ở node
	// model. Bước gọi model hiện ra không một chữ nào của câu hỏi, đúng ở cái
	// bảng sinh ra để trả lời "nó sẽ hỏi chúng nó cái gì".
	if got := timBuocKho(t, kh, "hoi").Prompt; got != "hỏi một câu" {
		t.Errorf("kế hoạch không hiện câu hỏi của bước `model`: %q", got)
	}

	// Thứ tự gộp phải hiện, và phải ĐÚNG thứ tự `needs`.
	gop := timBuocKho(t, kh, "gop").Gop
	if len(gop) != 2 || gop[0] != "hoi" || gop[1] != "ap" {
		t.Errorf("kế hoạch không nói thứ tự gộp, hoặc nói sai: %v", gop)
	}
}

func timBuocKho(t *testing.T, kh KeHoachKho, id string) BuocKho {
	t.Helper()
	for _, d := range kh.Dot {
		for _, b := range d.Buoc {
			if b.ID == id {
				return b
			}
		}
	}
	t.Fatalf("kế hoạch không có bước %q", id)
	return BuocKho{}
}

// Bảng chạy khan phải nói ra CẢ bước CHẠY THAY, không chỉ bước gỡ lại.
//
// Hai vai giống nhau ở chỗ quan trọng nhất — chúng KHÔNG chạy trong lượt suôn
// sẻ — nên bảng nào nói được vai này mà im về vai kia là bảng nói sai đúng một
// nửa. Đây là cùng lớp lỗi mà mục 1.4 của báo cáo #199 ghi lại: hai nhánh tự
// đếm riêng thì sớm muộn cũng lệch.
func TestChayKhoNoiRoCaBuocChayThay(t *testing.T) {
	khoTam(t)
	dungHoSoClaude(t, "tns", 2*time.Hour)
	dir := t.TempDir()
	f := flow.Flow{Name: "co-du-phong", Desc: "flow có hàng dự phòng", Steps: []flow.Step{
		{ID: "hoi-chinh", Type: flow.TypeAgent, Profile: "claude:tns", Prompt: "hỏi một câu",
			OnFailure: flow.OnFailFallback, Fallback: "hoi-du-phong"},
		// Bước chạy thay: KHÔNG có `needs`, tức một GỐC của DAG. Chính vì thế nó
		// nổi lên ngay đợt đầu và bảng cũ hiện nó như một bước sắp chạy.
		{ID: "hoi-du-phong", Type: flow.TypeAgent, Profile: "claude:tns",
			Prompt: "làm lại việc của {{buoc_hong}}"},
	}}
	if _, err := flow.Save(dir, f); err != nil {
		t.Fatal(err)
	}

	a := &API{}
	kh, err := a.FlowChayKho(dir, "co-du-phong", nil, Addr{})
	if err != nil {
		t.Fatal(err)
	}

	thay := timBuocKho(t, kh, "hoi-du-phong")
	if len(thay.ThayTheCho) == 0 {
		t.Fatal("bước `hoi-du-phong` KHÔNG được đánh dấu là bước chạy thay — bảng chạy khan " +
			"đang nói nó sẽ chạy, trong khi nó chỉ chạy khi có sự cố")
	}
	if thay.ThayTheCho[0] != "hoi-chinh" {
		t.Errorf("chạy thay cho %v, muốn [hoi-chinh]", thay.ThayTheCho)
	}
	if kh.SoBuocThayThe != 1 {
		t.Errorf("SoBuocThayThe = %d, muốn 1", kh.SoBuocThayThe)
	}
	// Chiều ngược lại: bước hỏng phải nói ra hàng dự phòng của nó.
	if got := timBuocKho(t, kh, "hoi-chinh").Fallback; got != "hoi-du-phong" {
		t.Errorf("bước `hoi-chinh` không nói ra bước chạy thay của nó: %q", got)
	}
	// Số agent: chỉ `hoi-chinh` được đếm. Đếm cả hàng dự phòng vào là trả lời
	// sai câu "sắp đốt bao nhiêu phiên" — theo hướng THỪA, tức là người đọc yên
	// tâm nhầm về giá.
	if kh.SoAgent != 1 {
		t.Fatalf("tổng số agent = %d, muốn 1 — bước chạy thay KHÔNG chạy trong lượt suôn sẻ", kh.SoAgent)
	}
}
