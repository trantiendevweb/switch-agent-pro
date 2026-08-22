package flow

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// agentTre là agent giả CHẬM KHÁC NHAU tuỳ prompt, và ghi lại thứ tự XONG.
//
// Cả bài test quan trọng nhất của mảnh merge dựa vào nó: muốn chứng minh thứ tự
// gộp KHÔNG phải thứ tự xong thì phải làm cho thứ tự xong đổi thật giữa hai
// lượt chạy, chứ không phải giả định là nó đổi.
type agentTre struct {
	mu    sync.Mutex
	tre   map[string]time.Duration
	ra    map[string]string
	hong  map[string]bool
	thuTu []string // prompt theo thứ tự XONG
}

func (a *agentTre) RunAgents(_ context.Context, _, _, prompt string, _ int, _, _ bool) (KetQuaAgent, error) {
	a.mu.Lock()
	d, out, hong := a.tre[prompt], a.ra[prompt], a.hong[prompt]
	a.mu.Unlock()
	time.Sleep(d)
	a.mu.Lock()
	a.thuTu = append(a.thuTu, prompt)
	a.mu.Unlock()
	if hong {
		return KetQuaAgent{}, context.DeadlineExceeded
	}
	return KetQuaAgent{Output: out}, nil
}

func (a *agentTre) xong() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.thuTu...)
}

// flowGop dựng flow ba nguồn chạy song song rồi một bước merge.
func flowGop() Flow {
	return Flow{Name: "gop", Steps: []Step{
		{ID: "an", Type: TypeAgent, Prompt: "p-an"},
		{ID: "banh", Type: TypeAgent, Prompt: "p-banh"},
		{ID: "cam", Type: TypeAgent, Prompt: "p-cam"},
		// Thứ tự trong `needs` LÀ thứ tự gộp. Cố ý KHÔNG theo a-b-c của id để
		// bài test phân biệt được "theo needs" với "theo tên đã sắp xếp".
		{ID: "gop", Type: TypeMerge, Needs: []string{"cam", "an", "banh"}},
	}}
}

// ĐÂY LÀ BÀI TEST QUAN TRỌNG NHẤT của node `merge`.
//
// Nó chạy CÙNG MỘT FLOW hai lượt, và ép hai lượt XONG THEO HAI THỨ TỰ NGƯỢC
// NHAU (bằng độ trễ đảo chiều). Khối chữ gộp phải giống nhau ĐẾN TỪNG BYTE.
//
// Vì sao đây là bài quan trọng nhất: nếu gộp theo thứ tự xong thì bài này đỏ,
// và đó đúng là cách hỏng không ai nhìn ra bằng mắt — flow chạy xanh cả hai
// lượt, chỉ có bước sau nhận hai đầu vào khác nhau. Một lượt chạy "giống hệt"
// mà không lặp lại được là một lượt chạy không kiểm chứng được gì.
func TestMergeHaiLuotXongNguocThuTuVanRaGiongNhau(t *testing.T) {
	chay := func(tre map[string]time.Duration) (string, []string) {
		t.Helper()
		r, _, db := newRunner(t)
		ag := &agentTre{
			tre: tre,
			ra:  map[string]string{"p-an": "AN nói A", "p-banh": "BANH nói B", "p-cam": "CAM nói C"},
		}
		r.Agent = ag
		res, err := r.Start(context.Background(), flowGop(), t.TempDir(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.State != store.RunDone {
			t.Fatalf("lượt chạy không xong: %s", res.State)
		}
		steps, err := db.Steps(res.RunID)
		if err != nil {
			t.Fatal(err)
		}
		return steps["gop"].Output, ag.xong()
	}

	// ĐỘ TRỄ 0 / 200 / 400ms chứ không phải 0 / 60 / 120.
	//
	// Chính bài này tự nói ra cách sửa ở câu báo lỗi bên dưới ("cần tăng độ trễ"),
	// và đây là lúc phải nghe: đo thật 22/08, chạy `go test ./...` năm lượt liền thì
	// một lượt đỏ với thuTu1 == thuTu2 == [p-cam p-banh p-an] — tức lượt 2 xong
	// theo ĐÚNG chiều ngược với độ trễ đặt ra, vì chênh lệch 120ms nhỏ hơn độ giật
	// của bộ lập lịch trên máy đang tải.
	//
	// KHÔNG nới khẳng định nào — vẫn đòi hai lượt xong theo hai thứ tự khác nhau,
	// chỉ làm cho điều kiện đó xảy ra thật.

	// Lượt 1: cam xong trước, rồi banh, rồi an.
	gop1, thuTu1 := chay(map[string]time.Duration{
		"p-an": 400 * time.Millisecond, "p-banh": 200 * time.Millisecond, "p-cam": 0,
	})
	// Lượt 2: ĐẢO NGƯỢC.
	gop2, thuTu2 := chay(map[string]time.Duration{
		"p-an": 0, "p-banh": 200 * time.Millisecond, "p-cam": 400 * time.Millisecond,
	})

	// Bằng chứng rằng bài test này có kiểm được cái nó nói: hai lượt phải XONG
	// theo hai thứ tự khác nhau thật. Không có khẳng định này thì máy chạy
	// nhanh quá có thể cho ra cùng thứ tự và bài test xanh mà chẳng chứng minh
	// được gì.
	if strings.Join(thuTu1, ",") == strings.Join(thuTu2, ",") {
		t.Fatalf("hai lượt xong CÙNG thứ tự (%v) — bài test không kiểm được gì, cần tăng độ trễ", thuTu1)
	}

	if gop1 != gop2 {
		t.Fatalf("hai lượt chạy giống hệt nhau ra hai khối gộp KHÁC NHAU.\n"+
			"xong lượt 1: %v\nxong lượt 2: %v\n--- lượt 1 ---\n%s\n--- lượt 2 ---\n%s",
			thuTu1, thuTu2, gop1, gop2)
	}

	// …và giống nhau theo đúng thứ tự KHAI TRONG `needs` (cam, an, banh), chứ
	// không phải theo id sắp xếp (an, banh, cam) hay theo thứ tự xong.
	muon := "=== cam ===\nCAM nói C\n\n=== an ===\nAN nói A\n\n=== banh ===\nBANH nói B"
	if gop1 != muon {
		t.Errorf("khối gộp không theo thứ tự khai trong needs.\nđược:\n%s\nmuốn:\n%s", gop1, muon)
	}
}

// Nguồn HỎNG mà lượt chạy vẫn đi tiếp (on_failure = "continue") thì khối gộp
// phải VẪN ĐỦ MỤC, và mục đó phải nói ra là hỏng.
//
// Bỏ mục đi mới là hành vi nguy hiểm: hai lượt chạy — một đủ ba nguồn, một mất
// nguồn giữa — cho ra hai khối mà nhìn vào không phân biệt được cái nào thiếu.
func TestMergeNguonHongVanConDuMuc(t *testing.T) {
	r, _, db := newRunner(t)
	r.Agent = &agentTre{
		ra:   map[string]string{"p-an": "AN nói A", "p-cam": "CAM nói C"},
		hong: map[string]bool{"p-banh": true},
	}
	f := flowGop()
	for i := range f.Steps {
		if f.Steps[i].ID == "banh" {
			f.Steps[i].OnFailure = OnFailContinue
		}
	}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	steps, err := db.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	got := steps["gop"].Output

	for _, muon := range []string{"=== cam ===", "=== an ===", "=== banh ==="} {
		if !strings.Contains(got, muon) {
			t.Errorf("khối gộp THIẾU mục %s — người đọc bước sau không có cách nào biết nguồn này vắng mặt:\n%s",
				muon, got)
		}
	}
	if !strings.Contains(got, `(bước "banh" HỎNG`) {
		t.Errorf("mục của nguồn hỏng không nói ra là nó hỏng:\n%s", got)
	}
	// Số mục phải đúng bằng số nguồn khai, không hơn không kém.
	if n := strings.Count(got, "=== "); n != 3 {
		t.Errorf("khối gộp có %d mục, muốn đúng 3", n)
	}
}

// Bước bị BỎ QUA (điều kiện `when` không thoả) khác hẳn bước HỎNG và khác hẳn
// bước xong-mà-rỗng. Cả ba đều phải hiện ra bằng ba câu khác nhau.
func TestMergePhanBietBoQuaVoiXongRongVoiHong(t *testing.T) {
	r, _, db := newRunner(t)
	r.Agent = &agentTre{ra: map[string]string{"p-rong": ""}}
	f := Flow{Name: "ba", Steps: []Step{
		{ID: "bo-qua", Type: TypeAgent, Prompt: "p-bo", When: "vars.chay == \"co\""},
		{ID: "rong", Type: TypeAgent, Prompt: "p-rong"},
		{ID: "gop", Type: TypeMerge, Needs: []string{"bo-qua", "rong"}},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), map[string]string{"chay": "khong"})
	if err != nil {
		t.Fatal(err)
	}
	steps, err := db.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	got := steps["gop"].Output
	if !strings.Contains(got, `(bước "bo-qua" bị BỎ QUA`) {
		t.Errorf("không phân biệt được bước bị bỏ qua:\n%s", got)
	}
	if !strings.Contains(got, `(bước "rong" xong nhưng KHÔNG để lại kết quả nào)`) {
		t.Errorf("không phân biệt được bước xong-mà-rỗng:\n%s", got)
	}
}

// merge KHÔNG CÓ `needs` là LỖI, không phải "gộp mọi bước trước".
//
// "Mọi bước trước" là một tập hợp không có thứ tự: bộ chạy giữ chúng trong map,
// và map của Go trả ra ngẫu nhiên. Cho phép nó tức là đẻ ra đúng cái không xác
// định mà cả node này sinh ra để chống.
func TestValidateMergeThieuNeeds(t *testing.T) {
	ps := Validate(Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "p"},
		{ID: "gop", Type: TypeMerge},
	}})
	if !coLoiChua(ps, "gop", "needs", false) {
		t.Fatalf("merge không có needs mà Validate không báo LỖI: %v", ps)
	}
}

func TestValidateMergeNeedsTrungTen(t *testing.T) {
	ps := Validate(Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "p"},
		{ID: "gop", Type: TypeMerge, Needs: []string{"a", "a"}},
	}})
	if !coLoiChua(ps, "gop", "hai lần", false) {
		t.Fatalf("needs trùng tên mà Validate không báo: %v", ps)
	}
}

// merge KHÔNG ghi ra file được, nên khai `artifact` ở đó là một hợp đồng chắc
// chắn không giữ được — cùng luật với approve/notify/model.
func TestValidateMergeKhongKhaiArtifactDuoc(t *testing.T) {
	ps := Validate(Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "p"},
		{ID: "gop", Type: TypeMerge, Needs: []string{"a"}, Artifact: map[string]string{"f": "f.txt"}},
	}})
	if !coLoiChua(ps, "gop", "artifact", false) {
		t.Fatalf("merge khai artifact mà Validate không báo LỖI: %v", ps)
	}
}

// `doc_duoc` chặn chính nguồn mình gộp là một lời khai TỰ MÂU THUẪN. Vẫn chạy
// được (nên chỉ cảnh báo), nhưng mục đó trong khối gộp là câu báo bị chặn —
// và người viết flow phải biết trước chuyện đó.
func TestMergeDocDuocChanNguon(t *testing.T) {
	s := Step{ID: "gop", Type: TypeMerge, Needs: []string{"a", "b"}, DocDuoc: []string{"a"}}
	got := GopDauRa(s,
		map[string]string{"a": store.StepDone, "b": store.StepDone},
		map[string]string{"a": "A nói", "b": "B nói"})
	if !strings.Contains(got, "A nói") {
		t.Errorf("bước ĐƯỢC phép đọc lại không có kết quả:\n%s", got)
	}
	if strings.Contains(got, "B nói") {
		t.Errorf("`doc_duoc` bị merge đi vòng qua — kết quả bước bị chặn vẫn lọt vào khối gộp:\n%s", got)
	}
	if !strings.Contains(got, "không được phép đọc kết quả") {
		t.Errorf("bị chặn mà không nói ra là bị chặn:\n%s", got)
	}

	ps := Validate(Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "p"},
		{ID: "b", Type: TypeAgent, Prompt: "p"},
		s,
	}})
	if !coLoiChua(ps, "gop", "doc_duoc", true) {
		t.Errorf("khai doc_duoc chặn chính nguồn mình gộp mà không có cảnh báo nào: %v", ps)
	}
}

// Khoá idempotency của merge phải CUỐN THEO nội dung nguồn.
//
// Không thì lượt sau dùng lại một khối gộp đã cũ trong khi nguồn đã đổi hẳn —
// đúng cái lỗi (a) mà idempotent.go liệt là tệ nhất.
func TestKhoaIdemMergeDoiTheoNguon(t *testing.T) {
	s := Step{ID: "gop", Type: TypeMerge, Needs: []string{"a"}, Idempotent: true}
	k1 := KhoaIdem(s, map[string]string{KhoaGopDauRa: "=== a ===\nlần một"})
	k2 := KhoaIdem(s, map[string]string{KhoaGopDauRa: "=== a ===\nlần hai"})
	if k1 == "" {
		t.Fatal("bước merge bật idempotent mà không sinh khoá")
	}
	if k1 == k2 {
		t.Fatal("nguồn đổi nội dung mà khoá idempotency KHÔNG đổi — lượt sau sẽ dùng lại khối gộp cũ")
	}
}

// coLoiChua tìm một vấn đề của bước `buoc` có chứa `manh`, đúng mức warn/lỗi.
func coLoiChua(ps []Problem, buoc, manh string, warn bool) bool {
	for _, p := range ps {
		if p.Step == buoc && p.Warn == warn && strings.Contains(p.Msg, manh) {
			return true
		}
	}
	return false
}
