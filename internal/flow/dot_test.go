package flow

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// tenBuoc rút id các bước của một đợt, đã sắp cho so sánh ổn định.
func tenBuoc(d DotChay) string {
	ids := make([]string, len(d.Buoc))
	for i, s := range d.Buoc {
		ids[i] = s.ID
	}
	return strings.Join(ids, ",")
}

// Hai bước không phụ thuộc nhau phải nằm CÙNG một đợt — đó chính là thứ người
// dùng chạy khan để biết: "bấm cái này thì mấy agent bật cùng lúc".
func TestDotGomBuocDocLapVaoMotDot(t *testing.T) {
	f := Flow{Name: "thu", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "a"},
		{ID: "b", Type: TypeNotify, Message: "b"},
		{ID: "c", Type: TypeNotify, Message: "c", Needs: []string{"a", "b"}},
	}}
	dots, err := Dot(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(dots) != 2 {
		t.Fatalf("phải chia 2 đợt, được %d: %v", len(dots), dots)
	}
	if got := tenBuoc(dots[0]); got != "a,b" {
		t.Fatalf("đợt 1 phải là a,b (chạy song song), được %q", got)
	}
	if got := tenBuoc(dots[1]); got != "c" {
		t.Fatalf("đợt 2 phải là c, được %q", got)
	}
	if dots[0].So != 1 || dots[1].So != 2 {
		t.Fatalf("số đợt phải đếm từ 1: %d, %d", dots[0].So, dots[1].So)
	}
}

// Bước approve KHÔNG được nằm chung đợt với bước khác: nó là rào, và rào đứng
// chung hàng với việc đang chạy thì không còn là rào nữa.
func TestDotTachRaoDuyetRaDotRieng(t *testing.T) {
	f := Flow{Name: "thu", Steps: []Step{
		{ID: "lam", Type: TypeNotify, Message: "lam"},
		{ID: "duyet", Type: TypeApprove, Message: "xem rồi duyệt", Needs: []string{"lam"}},
		{ID: "merge", Type: TypeNotify, Message: "merge", Needs: []string{"duyet"}},
	}}
	dots, err := Dot(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(dots) != 3 {
		t.Fatalf("phải chia 3 đợt, được %d", len(dots))
	}
	if !dots[1].ChoDuyet || tenBuoc(dots[1]) != "duyet" {
		t.Fatalf("đợt 2 phải là rào duyệt riêng một mình, được %q (ChoDuyet=%v)",
			tenBuoc(dots[1]), dots[1].ChoDuyet)
	}
	if dots[0].ChoDuyet || dots[2].ChoDuyet {
		t.Fatal("chỉ đợt chứa approve mới được đánh dấu ChoDuyet")
	}
}

// Approve KHÔNG bị kéo lên chạy chung với bước độc lập cùng vòng: runner chỉ
// dựng rào khi cả đợt không còn việc gì khác, và kế hoạch phải nói y hệt.
func TestDotApproveDoiHetViecMoiDungRao(t *testing.T) {
	f := Flow{Name: "thu", Steps: []Step{
		{ID: "cho", Type: TypeApprove, Message: "duyệt đi"},
		{ID: "viec", Type: TypeNotify, Message: "việc khác"},
	}}
	dots, err := Dot(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(dots) != 2 || tenBuoc(dots[0]) != "viec" || !dots[1].ChoDuyet {
		t.Fatalf("phải chạy `viec` trước rồi mới dựng rào `cho`, được %v", dots)
	}
}

func TestDotBaoChuTrinh(t *testing.T) {
	f := Flow{Name: "vong", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "a", Needs: []string{"b"}},
		{ID: "b", Type: TypeNotify, Message: "b", Needs: []string{"a"}},
	}}
	if _, err := Dot(f); err == nil {
		t.Fatal("flow có chu trình mà Dot vẫn trả về kế hoạch")
	}
}

// BÀI TEST QUAN TRỌNG NHẤT của phần này: kế hoạch phải khớp với thứ MÁY THẬT
// SỰ LÀM, không phải một bản chép tay trông na ná.
//
// Runner báo mỗi đợt song song bằng một dòng "chạy song song N bước: a, b".
// Chạy flow thật (agent giả, không tốn gì) rồi so từng đợt với Dot(). Lệch một
// bước là đỏ — nên nếu sau này ai sửa cách chia đợt trong runner mà quên Dot,
// test này bắt được ngay, thay vì để người dùng đọc một kế hoạch đã lỗi thời.
func TestDotKhopVoiDotRunnerThucSuChay(t *testing.T) {
	r, _, _ := newRunner(t)
	f := Flow{Name: "khop", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "a"},
		{ID: "b", Type: TypeNotify, Message: "b"},
		{ID: "c", Type: TypeNotify, Message: "c", Needs: []string{"a"}},
		{ID: "d", Type: TypeNotify, Message: "d", Needs: []string{"b", "c"}},
	}}

	ch, huy := r.Bus.Subscribe(64)
	var mu sync.Mutex
	var thay []string
	xong := make(chan struct{})
	go func() {
		defer close(xong)
		for e := range ch {
			if i := strings.Index(e.Msg, "chạy song song"); i >= 0 {
				mu.Lock()
				thay = append(thay, e.Msg[strings.Index(e.Msg, ":")+2:])
				mu.Unlock()
			}
		}
	}()

	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	huy()
	<-xong

	dots, err := Dot(f)
	if err != nil {
		t.Fatal(err)
	}
	// Chỉ các đợt có TỪ HAI BƯỚC trở lên mới sinh ra dòng "chạy song song".
	var muon []string
	for _, d := range dots {
		if len(d.Buoc) > 1 {
			muon = append(muon, strings.Join(strings.Split(tenBuoc(d), ","), ", "))
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(muon) != len(thay) {
		t.Fatalf("kế hoạch có %d đợt song song, máy chạy thật %d đợt:\n  kế hoạch: %v\n  đã chạy: %v",
			len(muon), len(thay), muon, thay)
	}
	for i := range muon {
		if muon[i] != thay[i] {
			t.Fatalf("đợt %d: kế hoạch nói %q, máy chạy thật %q", i+1, muon[i], thay[i])
		}
	}
}
