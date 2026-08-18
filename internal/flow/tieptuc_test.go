package flow

import (
	"context"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Hình dạng đúng bằng lần chạy #23: ba bước học song song, một bước gộp phía sau.
// Một bước học hỏng, nhưng nó khai on_failure=continue nên bước gộp PHẢI vẫn chạy
// với hai kết quả còn lại.
//
// Trước bản vá: readySteps chỉ coi done/skipped là xong, nên bước gộp không bao
// giờ đủ điều kiện. Runner hết việc, lần chạy vẫn ghi `completed`, còn bước gộp
// hiện "(chưa chạy)" — im lặng hoàn toàn.
func TestBuocSauVanChayKhiChaHongMaKhaiContinue(t *testing.T) {
	r, ag, _ := newRunner(t)
	ag.fail = true // mọi lượt gọi agent đều hỏng

	f := Flow{Name: "gop", Steps: []Step{
		{ID: "hoc-a", Type: TypeAgent, Prompt: "học A", OnFailure: OnFailContinue},
		{ID: "bao", Type: TypeNotify, Needs: []string{"hoc-a"}, Message: "đã gộp"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	steps, err := r.DB.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if steps["hoc-a"].State != store.StepFailed {
		t.Fatalf("bước học phải hỏng, được %q", steps["hoc-a"].State)
	}
	if steps["bao"].State != store.StepDone {
		t.Fatalf("BƯỚC SAU KHÔNG CHẠY dù bước trước khai on_failure=continue (trạng thái %q) — "+
			"đúng lỗi đã làm `tong-hop` của lần chạy #23 im lặng không chạy", steps["bao"].State)
	}
}

// Ngược lại: on_failure mặc định (stop) thì bước sau KHÔNG được chạy.
func TestBuocSauBiChanKhiChaHongMaKhongKhaiGi(t *testing.T) {
	r, ag, _ := newRunner(t)
	ag.fail = true

	f := Flow{Name: "chan", Steps: []Step{
		{ID: "hoc-a", Type: TypeAgent, Prompt: "học A"}, // mặc định = stop
		{ID: "bao", Type: TypeNotify, Needs: []string{"hoc-a"}, Message: "không được chạy"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	steps, err := r.DB.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if steps["bao"].State == store.StepDone {
		t.Fatal("bước trước hỏng với on_failure mặc định mà bước sau vẫn chạy")
	}
}
