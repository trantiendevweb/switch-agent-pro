package flow

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Bước sau phải NHẬN ĐƯỢC kết quả của bước trước qua {{steps.x.output}}.
func TestTruyenKetQuaGiuaCacBuoc(t *testing.T) {
	r, ag, _ := newRunner(t)
	echo := []string{"cmd", "/c", "echo KET-QUA-BUOC-MOT"}
	if runtime.GOOS != "windows" {
		echo = []string{"sh", "-c", "echo KET-QUA-BUOC-MOT"}
	}
	f := Flow{Name: "noi", Steps: []Step{
		{ID: "mot", Type: TypeShell, Run: echo},
		{ID: "hai", Type: TypeAgent, Needs: []string{"mot"},
			Prompt: "Đọc kết quả rồi tóm tắt: {{steps.mot.output}}"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("phải xong, được %s", res.State)
	}
	if len(ag.prompts) != 1 {
		t.Fatalf("agent phải được gọi 1 lần, được %d", len(ag.prompts))
	}
	if !strings.Contains(ag.prompts[0], "KET-QUA-BUOC-MOT") {
		t.Fatalf("prompt bước sau KHÔNG chứa kết quả bước trước: %q", ag.prompts[0])
	}
}

// Kết quả được lưu xuống DB, nên resume vẫn dùng lại được.
func TestKetQuaSongSotQuaResume(t *testing.T) {
	r, ag, db := newRunner(t)
	ag.output = "BAO-CAO-CUA-AGENT"
	f := Flow{Name: "sot", Steps: []Step{
		{ID: "chay", Type: TypeAgent, Prompt: "làm gì đó"},
		{ID: "gac", Type: TypeApprove, Needs: []string{"chay"}, Message: "?"},
		{ID: "dung", Type: TypeAgent, Needs: []string{"gac"},
			Prompt: "dùng lại: {{steps.chay.output}}"},
	}}
	res, _ := r.Start(context.Background(), f, t.TempDir(), nil)
	if res.State != store.RunWaiting {
		t.Fatalf("phải dừng chờ duyệt, được %s", res.State)
	}
	// Kết quả bước đầu đã nằm trên đĩa
	steps, _ := db.Steps(res.RunID)
	if steps["chay"].Output != "BAO-CAO-CUA-AGENT" {
		t.Fatalf("kết quả chưa được lưu: %q", steps["chay"].Output)
	}

	// Runner MỚI (như thể tiến trình đã chết và chạy lại) vẫn dùng được kết quả cũ
	r2 := &Runner{DB: r.DB, Bus: r.Bus, Agent: ag}
	if err := r2.Approve(res.RunID, "gac", "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := r2.Resume(context.Background(), res.RunID, f); err != nil {
		t.Fatal(err)
	}
	last := ag.prompts[len(ag.prompts)-1]
	if !strings.Contains(last, "BAO-CAO-CUA-AGENT") {
		t.Fatalf("sau resume, bước sau mất kết quả bước trước: %q", last)
	}
}

// Kết quả quá dài phải bị cắt và NÓI RÕ là đã cắt — không được để người đọc
// tưởng đó là toàn bộ.
func TestKetQuaDaiBiCatVaNoiRo(t *testing.T) {
	long := strings.Repeat("x", MaxInject+5000) + "PHAN-CUOI-QUAN-TRONG"
	got := WithOutputs(nil, map[string]string{"a": long})["steps.a.output"]
	if len(got) > MaxInject+200 {
		t.Fatalf("chưa cắt: dài %d", len(got))
	}
	if !strings.Contains(got, "PHAN-CUOI-QUAN-TRONG") {
		t.Fatal("phải giữ phần CUỐI — kết luận thường nằm ở đó")
	}
	if !strings.Contains(got, "cắt bớt") {
		t.Fatal("cắt mà không nói rõ là đã cắt")
	}
}

// Tham chiếu tới bước chưa chạy thì giữ nguyên chuỗi, không thay bằng rỗng —
// để người dùng nhìn thấy mình viết sai id chứ không im lặng nuốt.
func TestThamChieuSaiKhongBiNuot(t *testing.T) {
	got := Expand("dùng {{steps.khong-co.output}}", WithOutputs(nil, map[string]string{"a": "x"}))
	if !strings.Contains(got, "{{steps.khong-co.output}}") {
		t.Fatalf("tham chiếu sai phải giữ nguyên để người dùng thấy, được %q", got)
	}
}
