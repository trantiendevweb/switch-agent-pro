package flow

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// fakeAgent đếm số lần được gọi — để khẳng định bước SAU approve không chạy.
type fakeAgent struct {
	calls   int
	prompts []string
	fail    bool
}

func (f *fakeAgent) RunAgents(_ context.Context, _ string, prompt string, copies int, _ bool) error {
	f.calls += copies
	f.prompts = append(f.prompts, prompt)
	if f.fail {
		return context.DeadlineExceeded
	}
	return nil
}

func newRunner(t *testing.T) (*Runner, *fakeAgent, *store.DB) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	db, err := store.OpenAt(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	bus := events.NewBus()
	t.Cleanup(bus.Close)
	ag := &fakeAgent{}
	return &Runner{DB: db, Bus: bus, Agent: ag}, ag, db
}

// ĐÂY LÀ BÀI TEST QUAN TRỌNG NHẤT của Pha 3.
//
// MASTER-PLAN đòi "test chứng minh approval không thể bị bỏ qua". Flow dưới đây
// có approve chặn giữa; bước sau nó TUYỆT ĐỐI không được chạy khi chưa duyệt —
// kể cả khi gọi Resume nhiều lần.
func TestApprovalKhongTheBiBoQua(t *testing.T) {
	r, ag, db := newRunner(t)
	f := Flow{Name: "gac", Steps: []Step{
		{ID: "truoc", Type: TypeAgent, Prompt: "làm việc A"},
		{ID: "gac", Type: TypeApprove, Needs: []string{"truoc"}, Message: "duyệt chứ?"},
		{ID: "sau", Type: TypeAgent, Needs: []string{"gac"}, Prompt: "việc NGUY HIỂM"},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunWaiting || res.Waiting != "gac" {
		t.Fatalf("phải dừng chờ duyệt ở bước gac, được %+v", res)
	}
	if ag.calls != 1 {
		t.Fatalf("chỉ bước trước được chạy, agent gọi %d lần", ag.calls)
	}

	// Gọi Resume nhiều lần khi CHƯA duyệt: vẫn phải đứng yên.
	for i := 0; i < 3; i++ {
		res, err = r.Resume(context.Background(), res.RunID, f)
		if err != nil {
			t.Fatal(err)
		}
		if res.State != store.RunWaiting {
			t.Fatalf("lần %d: chưa duyệt mà đã chạy tiếp (%s)", i+1, res.State)
		}
	}
	if ag.calls != 1 {
		t.Fatalf("BƯỚC SAU APPROVE ĐÃ CHẠY KHI CHƯA DUYỆT — agent gọi %d lần", ag.calls)
	}
	for _, p := range ag.prompts {
		if p == "việc NGUY HIỂM" {
			t.Fatal("prompt của bước sau approve đã được thực thi khi chưa duyệt")
		}
	}

	// Duyệt rồi thì mới chạy tiếp.
	if err := r.Approve(res.RunID, "gac", "test"); err != nil {
		t.Fatal(err)
	}
	res, err = r.Resume(context.Background(), res.RunID, f)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("duyệt rồi phải chạy xong, được %s", res.State)
	}
	if ag.calls != 2 {
		t.Fatalf("sau khi duyệt, bước sau phải chạy đúng 1 lần; tổng %d", ag.calls)
	}
	_ = db
}

// Từ chối thì cả lần chạy dừng, bước sau không bao giờ chạy.
func TestTuChoiThiDungHan(t *testing.T) {
	r, ag, _ := newRunner(t)
	f := Flow{Name: "gac", Steps: []Step{
		{ID: "gac", Type: TypeApprove, Message: "duyệt?"},
		{ID: "sau", Type: TypeAgent, Needs: []string{"gac"}, Prompt: "x"},
	}}
	res, _ := r.Start(context.Background(), f, t.TempDir(), nil)
	if err := r.Reject(res.RunID, "gac", "test"); err != nil {
		t.Fatal(err)
	}
	run, _ := r.DB.GetRun(res.RunID)
	if run.State != store.RunCanceled {
		t.Fatalf("từ chối thì lần chạy phải bị huỷ, đang %s", run.State)
	}
	if ag.calls != 0 {
		t.Fatalf("bước sau chạy dù đã từ chối: %d lần", ag.calls)
	}
}

// Approve chỉ chấp nhận bước đang chờ — không "duyệt trước" được.
func TestKhongDuyetTruocDuoc(t *testing.T) {
	r, _, _ := newRunner(t)
	f := Flow{Name: "g", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "x"},
		{ID: "gac", Type: TypeApprove, Needs: []string{"a"}, Message: "?"},
	}}
	id, err := r.DB.CreateRun(f.Name, t.TempDir(), "{}")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Approve(id, "gac", "kẻ gian"); err == nil {
		t.Fatal("duyệt được một bước chưa hề chạy tới — approval gate thủng")
	}
}

// Resume bỏ qua bước đã xong: đây là cơ sở của "chạy tiếp sau khi máy restart".
func TestResumeKhongChayLaiBuocDaXong(t *testing.T) {
	r, ag, _ := newRunner(t)
	f := Flow{Name: "r", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "một lần thôi"},
		{ID: "gac", Type: TypeApprove, Needs: []string{"a"}, Message: "?"},
		{ID: "b", Type: TypeAgent, Needs: []string{"gac"}, Prompt: "sau"},
	}}
	res, _ := r.Start(context.Background(), f, t.TempDir(), nil)
	if ag.calls != 1 {
		t.Fatalf("bước a phải chạy 1 lần, được %d", ag.calls)
	}
	_ = r.Approve(res.RunID, "gac", "test")
	res, _ = r.Resume(context.Background(), res.RunID, f)
	if res.State != store.RunDone {
		t.Fatalf("phải xong, được %s", res.State)
	}
	if ag.calls != 2 {
		t.Fatalf("a không được chạy lại — tổng phải là 2, được %d", ag.calls)
	}
}

// on_failure = stop (mặc định) thì dừng; continue thì đi tiếp.
func TestChinhSachKhiHong(t *testing.T) {
	shellFail := []string{"cmd", "/c", "exit 1"}
	if runtime.GOOS != "windows" {
		shellFail = []string{"sh", "-c", "exit 1"}
	}

	r, ag, _ := newRunner(t)
	stop := Flow{Name: "stop", Steps: []Step{
		{ID: "hong", Type: TypeShell, Run: shellFail},
		{ID: "sau", Type: TypeAgent, Needs: []string{"hong"}, Prompt: "x"},
	}}
	res, _ := r.Start(context.Background(), stop, t.TempDir(), nil)
	if res.State != store.RunFailed {
		t.Fatalf("mặc định phải dừng khi hỏng, được %s", res.State)
	}
	if ag.calls != 0 {
		t.Fatal("bước sau chạy dù bước trước hỏng và on_failure=stop")
	}

	r2, ag2, _ := newRunner(t)
	cont := Flow{Name: "cont", Steps: []Step{
		{ID: "hong", Type: TypeShell, Run: shellFail, OnFailure: OnFailContinue},
		{ID: "sau", Type: TypeNotify, Message: "vẫn chạy"},
	}}
	res2, _ := r2.Start(context.Background(), cont, t.TempDir(), nil)
	if res2.State != store.RunDone {
		t.Fatalf("on_failure=continue thì phải chạy hết, được %s", res2.State)
	}
	_ = ag2
}

// Retry: hỏng thì thử lại đúng số lần khai báo.
func TestRetryDungSoLan(t *testing.T) {
	r, ag, _ := newRunner(t)
	ag.fail = true
	f := Flow{Name: "retry", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "x", Retry: 2}, // 1 lần đầu + 2 lần lại
	}}
	res, _ := r.Start(context.Background(), f, t.TempDir(), nil)
	if res.State != store.RunFailed {
		t.Fatalf("phải hỏng, được %s", res.State)
	}
	if ag.calls != 3 {
		t.Fatalf("retry=2 thì phải gọi 3 lần, được %d", ag.calls)
	}
}

// Biến được thay trong prompt trước khi giao cho agent.
func TestBienDuocThayTrongPrompt(t *testing.T) {
	r, ag, _ := newRunner(t)
	f := Flow{Name: "v", Vars: map[string]string{"task": "mặc định"},
		Steps: []Step{{ID: "a", Type: TypeAgent, Prompt: "làm: {{task}}"}}}
	if _, err := r.Start(context.Background(), f, t.TempDir(), map[string]string{"task": "việc thật"}); err != nil {
		t.Fatal(err)
	}
	if len(ag.prompts) != 1 || ag.prompts[0] != "làm: việc thật" {
		t.Fatalf("prompt = %v (tham số dòng lệnh phải đè giá trị mặc định)", ag.prompts)
	}
}
