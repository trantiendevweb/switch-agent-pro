package flow

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

func TestEvalCacToanTu(t *testing.T) {
	c := Ctx{
		Vars:    map[string]string{"moi_truong": "prod", "so": "7"},
		States:  map[string]string{"a": "done", "b": "failed"},
		Outputs: map[string]string{"a": "Tìm thấy 3 LỖI nghiêm trọng", "trong": ""},
	}
	cases := []struct {
		expr string
		want bool
	}{
		{"", true},
		{"steps.a.state == done", true},
		{"steps.a.state == failed", false},
		{"steps.b.state != done", true},
		{"steps.a.output contains lỗi", true}, // không phân biệt hoa thường
		{"steps.a.output contains xong", false},
		{"steps.a.output not-contains xong", true},
		{"vars.moi_truong == prod", true},
		{"vars.so > 5", true},
		{"vars.so < 5", false},
		{"steps.trong.output empty", true},
		{"steps.a.output not-empty", true},
		{"steps.a.state == steps.a.state", true}, // vế phải cũng tham chiếu được
	}
	for _, tc := range cases {
		got, err := Eval(tc.expr, c)
		if err != nil {
			t.Fatalf("%q lỗi: %v", tc.expr, err)
		}
		if got != tc.want {
			t.Fatalf("%q = %v, muốn %v", tc.expr, got, tc.want)
		}
	}
}

// Cú pháp sai phải BÁO LỖI, không được âm thầm coi là false — người viết flow
// cần biết mình gõ sai thay vì ngồi đoán vì sao bước không chạy.
func TestEvalSaiCuPhapBaoLoi(t *testing.T) {
	for _, bad := range []string{"linh tinh", "steps.a == done", "abc.x == y", "steps.a.mau == do"} {
		if _, err := Eval(bad, Ctx{}); err == nil {
			t.Fatalf("%q sai cú pháp mà không báo lỗi", bad)
		}
	}
}

// Điều kiện không thoả thì bước bị bỏ qua, và bước SAU nó vẫn chạy.
func TestWhenBoQuaNhungKhongChanBuocSau(t *testing.T) {
	r, ag, _ := newRunner(t)
	f := Flow{Name: "dk", Vars: map[string]string{"che_do": "nhanh"}, Steps: []Step{
		{ID: "day-du", Type: TypeAgent, Prompt: "chạy đầy đủ", When: "vars.che_do == day-du"},
		{ID: "cuoi", Type: TypeNotify, Needs: []string{"day-du"}, Message: "vẫn tới đây"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("phải chạy xong, được %s", res.State)
	}
	if ag.calls != 0 {
		t.Fatalf("bước có điều kiện không thoả mà vẫn chạy: %d lần", ag.calls)
	}
	steps, _ := r.DB.Steps(res.RunID)
	if steps["day-du"].State != store.StepSkipped {
		t.Fatalf("phải là skipped, đang %s", steps["day-du"].State)
	}
	if steps["cuoi"].State != store.StepDone {
		t.Fatalf("bước sau phải vẫn chạy, đang %s", steps["cuoi"].State)
	}
}

// Rẽ nhánh theo KẾT QUẢ bước trước — đây là điều làm flow "thông minh".
func TestReNhanhTheoKetQuaBuocTruoc(t *testing.T) {
	r, ag, _ := newRunner(t)
	echo := []string{"cmd", "/c", "echo CO-LOI-NGHIEM-TRONG"}
	if runtime.GOOS != "windows" {
		echo = []string{"sh", "-c", "echo CO-LOI-NGHIEM-TRONG"}
	}
	f := Flow{Name: "re", Steps: []Step{
		{ID: "kiem", Type: TypeShell, Run: echo},
		{ID: "sua", Type: TypeAgent, Needs: []string{"kiem"}, Prompt: "sửa lỗi",
			When: "steps.kiem.output contains CO-LOI"},
		{ID: "bo-qua", Type: TypeAgent, Needs: []string{"kiem"}, Prompt: "không cần sửa",
			When: "steps.kiem.output not-contains CO-LOI"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("phải xong, được %s", res.State)
	}
	steps, _ := r.DB.Steps(res.RunID)
	if steps["sua"].State != store.StepDone {
		t.Fatalf("nhánh đúng phải chạy, đang %s", steps["sua"].State)
	}
	if steps["bo-qua"].State != store.StepSkipped {
		t.Fatalf("nhánh sai phải bỏ qua, đang %s", steps["bo-qua"].State)
	}
	if len(ag.prompts) != 1 || ag.prompts[0] != "sửa lỗi" {
		t.Fatalf("chỉ nhánh đúng được chạy, prompts = %v", ag.prompts)
	}
}

// slowAgent đo xem các bước có THẬT SỰ chạy cùng lúc không.
type slowAgent struct {
	mu      sync.Mutex
	now     int
	maxSeen int
}

func (s *slowAgent) RunAgents(_ context.Context, _, prompt string, copies int, _, _ bool) (string, error) {
	s.mu.Lock()
	s.now++
	if s.now > s.maxSeen {
		s.maxSeen = s.now
	}
	s.mu.Unlock()

	time.Sleep(250 * time.Millisecond)

	s.mu.Lock()
	s.now--
	s.mu.Unlock()
	return prompt, nil
}

// Ba nhánh độc lập phải chạy CÙNG LÚC, không xếp hàng.
func TestCacNhanhDocLapChaySongSong(t *testing.T) {
	r, _, _ := newRunner(t)
	sa := &slowAgent{}
	r.Agent = sa
	r.MaxParallel = 4

	f := Flow{Name: "ss", Steps: []Step{
		{ID: "mo-dau", Type: TypeNotify, Message: "bắt đầu"},
		{ID: "a", Type: TypeAgent, Needs: []string{"mo-dau"}, Prompt: "việc A"},
		{ID: "b", Type: TypeAgent, Needs: []string{"mo-dau"}, Prompt: "việc B"},
		{ID: "c", Type: TypeAgent, Needs: []string{"mo-dau"}, Prompt: "việc C"},
		{ID: "gop", Type: TypeNotify, Needs: []string{"a", "b", "c"}, Message: "gộp"},
	}}

	start := time.Now()
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	took := time.Since(start)

	if res.State != store.RunDone {
		t.Fatalf("phải xong, được %s", res.State)
	}
	if sa.maxSeen < 3 {
		t.Fatalf("ba nhánh độc lập phải chạy cùng lúc; số bước chạy đồng thời tối đa chỉ %d", sa.maxSeen)
	}
	// Tuần tự sẽ mất ~750ms; song song thì quanh ~250ms.
	if took > 600*time.Millisecond {
		t.Fatalf("có vẻ vẫn chạy tuần tự: mất %s", took)
	}
}

// Trần song song của dự án phải được tôn trọng.
func TestTonTrongTranSongSong(t *testing.T) {
	r, _, _ := newRunner(t)
	sa := &slowAgent{}
	r.Agent = sa
	r.MaxParallel = 2

	steps := []Step{{ID: "mo-dau", Type: TypeNotify, Message: "x"}}
	for _, id := range []string{"a", "b", "c", "d"} {
		steps = append(steps, Step{ID: id, Type: TypeAgent, Needs: []string{"mo-dau"}, Prompt: id})
	}
	f := Flow{Name: "tran", Steps: steps}
	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	if sa.maxSeen > 2 {
		t.Fatalf("trần là 2 mà chạy tới %d bước cùng lúc", sa.maxSeen)
	}
}

// Approval gate vẫn phải nguyên vẹn khi chạy song song.
func TestApprovalVanChanKhiChaySongSong(t *testing.T) {
	r, ag, _ := newRunner(t)
	f := Flow{Name: "gac-ss", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "song song 1"},
		{ID: "b", Type: TypeAgent, Prompt: "song song 2"},
		{ID: "gac", Type: TypeApprove, Needs: []string{"a", "b"}, Message: "duyệt?"},
		{ID: "nguy-hiem", Type: TypeAgent, Needs: []string{"gac"}, Prompt: "VIEC-NGUY-HIEM"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunWaiting {
		t.Fatalf("phải dừng chờ duyệt, được %s", res.State)
	}
	for _, p := range ag.prompts {
		if strings.Contains(p, "NGUY-HIEM") {
			t.Fatal("bước sau approve chạy khi chưa duyệt — gate thủng khi chạy song song")
		}
	}
	if ag.calls != 2 {
		t.Fatalf("chỉ hai bước song song trước gate được chạy, được %d", ag.calls)
	}
}
