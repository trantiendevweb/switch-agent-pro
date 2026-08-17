package flow

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Một bước, nhiều lượt chạy — mỗi dòng của nguồn là một lượt.
func TestForEachChayTungMuc(t *testing.T) {
	r, ag, _ := newRunner(t)
	f := Flow{Name: "lap", Vars: map[string]string{"ds": "alpha\nbeta\ngamma"},
		Steps: []Step{
			{ID: "xu-ly", Type: TypeAgent, ForEach: "vars.ds", Prompt: "Xử lý {{item}} (số {{index}})"},
		}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("phải xong, được %s", res.State)
	}
	if len(ag.prompts) != 3 {
		t.Fatalf("3 mục phải thành 3 lượt, được %d: %v", len(ag.prompts), ag.prompts)
	}
	joined := strings.Join(ag.prompts, " | ")
	for _, want := range []string{"alpha", "beta", "gamma", "số 1", "số 3"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("thiếu %q trong các prompt: %s", want, joined)
		}
	}
}

// Nguồn là KẾT QUẢ bước trước — đây mới là cách dùng mạnh nhất.
func TestForEachLayTuKetQuaBuocTruoc(t *testing.T) {
	r, ag, _ := newRunner(t)
	list := []string{"cmd", "/c", "echo mot& echo hai"}
	if runtime.GOOS != "windows" {
		list = []string{"sh", "-c", "printf 'mot\\nhai\\n'"}
	}
	f := Flow{Name: "noi", Steps: []Step{
		{ID: "liet-ke", Type: TypeShell, Run: list},
		{ID: "xu-ly", Type: TypeAgent, Needs: []string{"liet-ke"},
			ForEach: "steps.liet-ke.output", Prompt: "làm {{item}}"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("phải xong, được %s", res.State)
	}
	if len(ag.prompts) != 2 {
		t.Fatalf("output 2 dòng phải thành 2 lượt, được %d: %v", len(ag.prompts), ag.prompts)
	}
}

// Kết quả gộp phải đánh dấu từng mục, để bước sau còn phân biệt được.
func TestForEachGopKetQuaCoDanhDau(t *testing.T) {
	r, ag, db := newRunner(t)
	ag.output = "OK"
	f := Flow{Name: "gop", Vars: map[string]string{"ds": "mot\nhai"},
		Steps: []Step{{ID: "x", Type: TypeAgent, ForEach: "vars.ds", Prompt: "{{item}}"}}}
	res, _ := r.Start(context.Background(), f, t.TempDir(), nil)
	steps, _ := db.Steps(res.RunID)
	out := steps["x"].Output
	for _, want := range []string{"=== mot ===", "=== hai ===", "OK"} {
		if !strings.Contains(out, want) {
			t.Fatalf("kết quả gộp thiếu %q:\n%s", want, out)
		}
	}
}

// Danh sách rỗng thì bỏ qua, không phải lỗi.
func TestForEachDanhSachRongThiBoQua(t *testing.T) {
	r, ag, db := newRunner(t)
	f := Flow{Name: "rong", Vars: map[string]string{"ds": "   \n  "},
		Steps: []Step{
			{ID: "x", Type: TypeAgent, ForEach: "vars.ds", Prompt: "{{item}}"},
			{ID: "sau", Type: TypeNotify, Needs: []string{"x"}, Message: "vẫn chạy"},
		}}
	res, _ := r.Start(context.Background(), f, t.TempDir(), nil)
	if res.State != store.RunDone {
		t.Fatalf("phải xong, được %s", res.State)
	}
	if ag.calls != 0 {
		t.Fatalf("danh sách rỗng mà vẫn gọi agent %d lần", ag.calls)
	}
	steps, _ := db.Steps(res.RunID)
	if steps["x"].State != store.StepSkipped {
		t.Fatalf("phải skipped, đang %s", steps["x"].State)
	}
	if steps["sau"].State != store.StepDone {
		t.Fatal("bước sau phải vẫn chạy")
	}
}

// Trần số mục: nguồn quá dài thì DỪNG và báo, không âm thầm chạy hàng nghìn lượt.
func TestForEachVuotTranThiBao(t *testing.T) {
	r, ag, _ := newRunner(t)
	var sb strings.Builder
	for i := 0; i < MaxForEachItems+5; i++ {
		sb.WriteString("muc\n")
	}
	f := Flow{Name: "qua-nhieu", Vars: map[string]string{"ds": sb.String()},
		Steps: []Step{{ID: "x", Type: TypeAgent, ForEach: "vars.ds", Prompt: "{{item}}"}}}
	res, _ := r.Start(context.Background(), f, t.TempDir(), nil)
	if res.State != store.RunFailed {
		t.Fatalf("vượt trần phải dừng, được %s", res.State)
	}
	if ag.calls != 0 {
		t.Fatalf("KHÔNG được gọi agent lần nào khi vượt trần, đã gọi %d", ag.calls)
	}
}

// Các lượt foreach chạy song song, tôn trọng trần.
func TestForEachChaySongSongTheoTran(t *testing.T) {
	r, _, _ := newRunner(t)
	sa := &slowAgent{}
	r.Agent = sa
	r.MaxParallel = 3
	f := Flow{Name: "ss", Vars: map[string]string{"ds": "a\nb\nc\nd\ne\nf"},
		Steps: []Step{{ID: "x", Type: TypeAgent, ForEach: "vars.ds", Prompt: "{{item}}"}}}
	start := time.Now()
	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	took := time.Since(start)
	if sa.maxSeen < 2 {
		t.Fatalf("các lượt phải chạy song song, tối đa đồng thời chỉ %d", sa.maxSeen)
	}
	if sa.maxSeen > 3 {
		t.Fatalf("trần là 3 mà chạy tới %d lượt cùng lúc", sa.maxSeen)
	}
	// 6 mục, trần 3, mỗi lượt 250ms → ~500ms; tuần tự sẽ là 1.5s
	if took > 1100*time.Millisecond {
		t.Fatalf("có vẻ chạy tuần tự: mất %s", took)
	}
}

// foreach trỏ bậy thì bắt ngay lúc kiểm tra, không đợi tới lúc chạy.
func TestValidateBatForEachSai(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "p", ForEach: "linh-tinh"},
	}}
	if len(errs(Validate(f))) == 0 {
		t.Fatal("foreach trỏ bậy mà không báo lỗi")
	}
	f2 := Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypeApprove, Message: "m", ForEach: "vars.ds"},
	}}
	if len(errs(Validate(f2))) == 0 {
		t.Fatal("approve mà lặp thì phải báo lỗi")
	}
}
