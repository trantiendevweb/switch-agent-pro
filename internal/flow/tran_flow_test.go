package flow

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/fleet"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// TRẦN ĐỒNG THỜI TRÊN ĐƯỜNG FLOW — kiểm ở CHỖ GỌI, không phải chỗ tính.
//
// internal/fleet/tran_test.go đã chứng minh phép tính bốn chiều đúng, và
// internal/api/tran_test.go đã chứng minh nó được cắm vào `FleetStart`. Cả hai
// bộ đó XANH SẠCH suốt trong khi đường flow vẫn chỉ nhận một con số
// (`MaxParallel`) — đúng lớp lỗi "có ở mọi tầng trừ tầng cuối" mà dự án này dính
// năm lần trong ngày. Bộ dưới đây canh đúng cái tầng cuối đó.
//
// Cách đo: không hỏi hàm nào cả, mà CHẠY MỘT FLOW THẬT qua Runner.Start rồi đếm
// xem có mấy bước `agent` cùng chạy tại một thời điểm. Gọi thẳng fleet.XetTran
// trong bài kiểm thì bỏ cổng ra khỏi step.go vẫn xanh — chính là cái bẫy này.
//
// ĐÃ THỬ THẬT (22/08): xoá khối `r.xinCho(...)` trong step.go rồi chạy lại thì
// bốn bài đầu đỏ, kèm số đo "đỉnh cùng lúc = 4, trần hồ sơ = 2". Ghi lại vì một
// bài kiểm chưa từng đỏ thì chưa chứng minh được gì.

// agentDo là bộ chạy agent giả CÓ ĐO ĐỈNH: nó ghi lại số lượt chạy chồng nhau
// cao nhất theo TỪNG hồ sơ, và cả trên tổng.
//
// Ngủ thật vài chục mili giây chứ không trả về ngay: hai lượt chạy nối tiếp nhau
// trong cùng một micro-giây cũng cho "đỉnh 1" mà chẳng chứng minh được gì. Muốn
// đo được chồng lấn thì phải có chồng lấn để mà đo.
type agentDo struct {
	mu       sync.Mutex
	dangChay map[string]int
	dinh     map[string]int
	dinhTong int
	tong     int
	goi      int
	capSo    []int // số copies THẬT của từng lượt, sau khi cổng cắt
	rao      chan struct{}
	daMoRao  bool

	Lau time.Duration

	// ChoDu > 0 bật CHỐT HẸN: mỗi lượt đứng lại tới khi có đủ ChoDu lượt cùng
	// lúc, hoặc hết Lau thì đi tiếp.
	//
	// Vì sao cần, và đây là một con flake đã gặp thật lúc soạn bộ này: bài "bốn
	// bước PHẢI chạy được cùng lúc" ngủ 60ms mỗi lượt vẫn ra đỉnh 3/4, vì mỗi
	// bước ghi vài dòng vào SQLite trước khi chạy và trên Windows các lượt ghi
	// đó xếp hàng đủ lâu để bước cuối khởi động sau khi bước đầu đã xong. Kéo
	// dài giấc ngủ chỉ làm con flake hiếm đi chứ không mất. Chốt hẹn thì dứt
	// khoát: chặn được thì chốt KHÔNG BAO GIỜ đầy, và bài đỏ ngay.
	ChoDu int
}

func newAgentDo(lau time.Duration) *agentDo {
	return &agentDo{
		dangChay: map[string]int{}, dinh: map[string]int{},
		rao: make(chan struct{}), Lau: lau,
	}
}

func (a *agentDo) RunAgents(ctx context.Context, profile, _ string, _ string, copies int, _, _ bool) (KetQuaAgent, error) {
	a.mu.Lock()
	a.goi++
	a.capSo = append(a.capSo, copies)
	a.dangChay[profile]++
	a.tong++
	if a.dangChay[profile] > a.dinh[profile] {
		a.dinh[profile] = a.dangChay[profile]
	}
	if a.tong > a.dinhTong {
		a.dinhTong = a.tong
	}
	if a.ChoDu > 0 && a.tong >= a.ChoDu && !a.daMoRao {
		a.daMoRao = true
		close(a.rao)
	}
	rao := a.rao
	a.mu.Unlock()

	if a.ChoDu > 0 {
		select {
		case <-rao:
		case <-time.After(a.Lau):
		case <-ctx.Done():
		}
	} else {
		select {
		case <-time.After(a.Lau):
		case <-ctx.Done():
		}
	}

	a.mu.Lock()
	a.dangChay[profile]--
	a.tong--
	a.mu.Unlock()
	return KetQuaAgent{Output: "xong"}, nil
}

func (a *agentDo) doc() (dinh map[string]int, dinhTong, goi int, cap []int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	d := map[string]int{}
	for k, v := range a.dinh {
		d[k] = v
	}
	return d, a.dinhTong, a.goi, append([]int(nil), a.capSo...)
}

// runnerCoTran dựng một Runner đã cắm cổng, y như internal/api dựng nó.
//
// `nen` là các phiên đang chạy NGOÀI lượt flow này (một lượt `sagent fleet`
// khác chẳng hạn) — chính là thứ sổ phiên trả về ở đời thật.
func runnerCoTran(t *testing.T, tr fleet.Tran, nen []fleet.Phien, lau time.Duration) (*Runner, *agentDo, *[]string) {
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

	var mu sync.Mutex
	noi := []string{}
	ag := newAgentDo(lau)
	cong := fleet.MoCong(tr, func() []fleet.Phien { return nen }, func(m string) {
		mu.Lock()
		noi = append(noi, m)
		mu.Unlock()
	})
	// Nhịp nhắc ngắn để bài kiểm không phải ngồi chờ 30 giây thật.
	cong.Nhip = 10 * time.Millisecond
	cong.ChoKetCung = 300 * time.Millisecond

	r := &Runner{DB: db, Bus: bus, Agent: ag, MaxParallel: 8, Cong: cong}
	return r, ag, &noi
}

// tranMacDinh là đúng bộ số mặc định của dự án: chung 4 · harness 3 ·
// provider 3 · hồ sơ 2. Bài kiểm dùng nguyên nó chứ không bịa số riêng — cái
// đáng canh là hành vi MẶC ĐỊNH, thứ mà người dùng không sửa file nào cũng gặp.
func tranMacDinh() fleet.Tran {
	return fleet.Tran{Chung: 4, HarnessMacDinh: 3, ProviderMacDinh: 3, HoSoMacDinh: 2}
}

func buocAgent(n int, profile string) []Step {
	out := make([]Step, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, Step{
			ID: fmt.Sprintf("b%d", i), Type: TypeAgent,
			Profile: profile, Prompt: "làm việc",
		})
	}
	return out
}

// BÀI CHÍNH — đúng cái sự cố đã đẻ ra tính năng này, nhưng đi vào CỬA FLOW.
//
// Bốn bước `agent` không phụ thuộc nhau nên rơi hết vào MỘT đợt, cả bốn khai
// chung `profile = "claude:tns"`. Trần hồ sơ mặc định là 2. Trước bản sửa: cả
// bốn cùng chạy (MaxParallel=8 cho qua, và bốn lượt FleetStart đọc sổ trước khi
// lượt nào kịp ghi phiên vào). Sau bản sửa: đỉnh phải là 2.
func TestBonBuocAgentCungHoSoKhongVuotTranHoSo(t *testing.T) {
	r, ag, _ := runnerCoTran(t, tranMacDinh(), nil, time.Second)
	// PHẢI dùng chốt hẹn. Bản đầu của bài này chỉ ngủ 40ms mỗi lượt, và khi tôi
	// gỡ bản sửa ra để thử thì nó vẫn XANH — bốn lượt chạy bị các lượt ghi
	// SQLite làm lệch pha nên đỉnh đo được là 2 dù chẳng có trần nào chặn. Một
	// bài kiểm xanh vì tình cờ còn tệ hơn không có bài kiểm.
	ag.ChoDu = 4
	f := Flow{Name: "dong-thoi", Steps: buocAgent(4, "claude:tns")}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	dinh, _, goi, _ := ag.doc()

	if dinh["claude:tns"] > 2 {
		t.Fatalf("TRẦN HỒ SƠ BỊ VƯỢT Ở CỬA FLOW: đỉnh cùng lúc %d, trần 2 — "+
			"bốn bước dồn vào một tài khoản đúng như sự cố cũ", dinh["claude:tns"])
	}
	// Và phải chạy ĐỦ bốn bước, không bước nào bị bỏ.
	if goi != 4 {
		t.Fatalf("phải chạy đủ 4 bước, chỉ chạy %d", goi)
	}
	if res.State != store.RunDone {
		t.Fatalf("trần chật KHÔNG được giết lượt chạy — trạng thái %s", res.State)
	}
}

// Trần chặn thì CHỜ RỒI CHẠY, không từ chối. Một lượt flow đêm dài bị giết vì
// trần là tệ hơn là chạy chậm.
//
// Đo bằng trạng thái từng bước trong sổ, không bằng trạng thái tổng: một bước
// `failed` mà `on_failure = continue` vẫn cho lượt chạy về `done`, nên chỉ nhìn
// con số tổng là bỏ lọt đúng ca cần bắt.
func TestBuocBiTranChanThiChoChuKhongGietLuotChay(t *testing.T) {
	tr := tranMacDinh()
	tr.HoSoMacDinh = 1 // chật nhất có thể mà vẫn chạy được: đúng một phiên
	r, ag, _ := runnerCoTran(t, tr, nil, 400*time.Millisecond)
	ag.ChoDu = 2 // chồng lấn dù chỉ HAI lượt cũng là vượt trần 1
	f := Flow{Name: "xep-hang", Steps: buocAgent(4, "claude:tns")}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("lượt chạy phải XONG, được %s", res.State)
	}
	buoc, err := r.DB.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	for id, s := range buoc {
		if s.State != store.StepDone {
			t.Fatalf("bước %s không xong (%s: %s) — trần đã GIẾT bước thay vì bắt nó chờ",
				id, s.State, s.Msg)
		}
	}
	dinh, _, goi, _ := ag.doc()
	if dinh["claude:tns"] != 1 {
		t.Fatalf("trần hồ sơ 1 mà đỉnh cùng lúc là %d", dinh["claude:tns"])
	}
	if goi != 4 {
		t.Fatalf("phải chạy đủ 4 bước, chỉ chạy %d", goi)
	}
}

// Đang chờ thì phải NÓI RA. Người vận hành đọc dòng này lúc 2 giờ sáng, và họ
// cần đúng ba thứ: trần nào chặn, còn mấy chỗ, chờ bao lâu rồi.
func TestNoiRaKhiDangChoTran(t *testing.T) {
	r, _, noi := runnerCoTran(t, tranMacDinh(), nil, 60*time.Millisecond)
	f := Flow{Name: "noi-ra", Steps: buocAgent(4, "claude:tns")}

	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	gop := strings.Join(*noi, "\n")
	if gop == "" {
		t.Fatal("TREO IM LẶNG: bước phải chờ mà không nói một dòng nào")
	}
	// Ba thứ, kiểm từng thứ một để lúc đỏ biết ngay thiếu cái gì.
	for _, can := range []struct{ chuoi, vi string }{
		{"CHỜ TRẦN ĐỒNG THỜI", "không nói là đang chờ vì trần"},
		{"hồ sơ claude:tns", "không gọi tên trần nào đang chặn"},
		{"còn 0 chỗ", "không nói còn mấy chỗ"},
		{"đã chờ", "không nói đã chờ bao lâu"},
		{"KHÔNG bị huỷ", "không trấn an là bước vẫn sẽ chạy — người đọc sẽ đi giết tiến trình"},
	} {
		if !strings.Contains(gop, can.chuoi) {
			t.Errorf("tin báo %s.\nĐã nói:\n%s", can.vi, gop)
		}
	}
}

// KẸT CỨNG: mọi chỗ bị chiếm bởi phiên NGOÀI lượt flow này, và cổng không giữ
// chỗ nào nên không có gì sẽ trả chỗ ra. Chờ tiếp là treo tới sáng.
//
// Bài này có HẠN GIỜ thật: treo im lặng thì nó phải đỏ vì hết giờ, chứ không
// phải ngồi đợi mãi rồi CI tự giết cả gói.
func TestKetCungThiBaoRaChuKhongTreo(t *testing.T) {
	nen := []fleet.Phien{{Provider: "claude", Account: "tns"}, {Provider: "claude", Account: "tns"}}
	r, _, noi := runnerCoTran(t, tranMacDinh(), nen, 10*time.Millisecond)
	f := Flow{Name: "ket-cung", Steps: buocAgent(1, "claude:tns")}

	xong := make(chan Result, 1)
	go func() {
		res, _ := r.Start(context.Background(), f, t.TempDir(), nil)
		xong <- res
	}()
	select {
	case res := <-xong:
		if res.State != store.RunFailed {
			t.Fatalf("kẹt cứng phải làm lượt chạy hỏng CÓ LÝ DO, được %s", res.State)
		}
		buoc, err := r.DB.Steps(res.RunID)
		if err != nil {
			t.Fatal(err)
		}
		s := buoc["b1"]
		if !strings.Contains(s.Msg, "KẸT CỨNG") {
			t.Fatalf("lý do hỏng không nói là kẹt cứng: %q", s.Msg)
		}
		if !strings.Contains(s.Msg, "hồ sơ claude:tns") {
			t.Fatalf("lý do hỏng không gọi tên trần đang chặn: %q", s.Msg)
		}
		_ = noi
	case <-time.After(10 * time.Second):
		t.Fatal("TREO IM LẶNG: kẹt cứng mà bộ chạy đứng luôn, không báo gì")
	}
}

// Trần hồ sơ chặn theo TÀI KHOẢN, không phải chặn tất. Hai bước claude:tns và
// hai bước claude:phu phải chạy được cả bốn cùng lúc.
//
// Bài này canh hướng hỏng NGƯỢC LẠI của bản sửa: xếp hàng tất cả vào một cổng
// chung thì các bài trên vẫn xanh, mà hạm đội mất sạch ý nghĩa.
//
// Nới trần harness và provider lên 4 để CHỈ còn chiều hồ sơ là chiều đang được
// đo. Giữ bộ mặc định thì bài này vẫn đỏ, nhưng đỏ vì trần provider (3) chứ
// không phải vì lỗi — đo được lúc soạn bài, và nó là con số của bài dưới.
func TestHaiTaiKhoanKhacNhauVanChayCungLuc(t *testing.T) {
	tr := tranMacDinh()
	tr.HarnessMacDinh, tr.ProviderMacDinh = 4, 4
	r, ag, _ := runnerCoTran(t, tr, nil, 3*time.Second)
	ag.ChoDu = 4 // chốt hẹn: đủ bốn lượt cùng lúc thì đi ngay, không thì đứng 3s
	f := Flow{Name: "trai-ra", Steps: []Step{
		{ID: "a1", Type: TypeAgent, Profile: "claude:tns", Prompt: "x"},
		{ID: "a2", Type: TypeAgent, Profile: "claude:tns", Prompt: "x"},
		{ID: "b1", Type: TypeAgent, Profile: "claude:phu", Prompt: "x"},
		{ID: "b2", Type: TypeAgent, Profile: "claude:phu", Prompt: "x"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("trạng thái %s", res.State)
	}
	_, dinhTong, goi, _ := ag.doc()
	if goi != 4 {
		t.Fatalf("phải chạy đủ 4 bước, chỉ chạy %d", goi)
	}
	if dinhTong != 4 {
		t.Fatalf("hai tài khoản khác nhau, trần chung 4 — phải chạy được cả 4 cùng lúc, đỉnh chỉ %d. "+
			"Cổng đang xếp hàng theo TỔNG thay vì theo từng chiều", dinhTong)
	}
}

// SỐ ĐO đáng biết, tìm ra lúc soạn bài trên: với bộ trần MẶC ĐỊNH, một flow có
// bốn bước `agent` cùng nhà cung cấp `claude` KHÔNG BAO GIỜ chạy quá 3 phiên
// cùng lúc — dù trải ra bao nhiêu tài khoản, và dù `max_parallel_sessions` là 4.
// Chặn nó là trần PROVIDER (mặc định 3), không phải trần chung.
//
// Ghi thành bài kiểm chứ không chỉ ghi vào tài liệu: đây là thứ người vận hành
// sẽ gặp rồi tưởng công cụ hỏng ("tôi có 4 tài khoản mà nó chỉ chạy 3"), nên
// con số phải có chỗ neo và phải đỏ nếu ai đó lặng lẽ đổi mặc định.
func TestTranProviderLaTranNgoaiCungCuaMotNhaCungCap(t *testing.T) {
	r, ag, _ := runnerCoTran(t, tranMacDinh(), nil, 600*time.Millisecond)
	ag.ChoDu = 4
	f := Flow{Name: "bon-tai-khoan", Steps: []Step{
		{ID: "a", Type: TypeAgent, Profile: "claude:tns", Prompt: "x"},
		{ID: "b", Type: TypeAgent, Profile: "claude:phu", Prompt: "x"},
		{ID: "c", Type: TypeAgent, Profile: "claude:ba", Prompt: "x"},
		{ID: "d", Type: TypeAgent, Profile: "claude:bon", Prompt: "x"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("trạng thái %s — bốn tài khoản khác nhau thì không được hỏng, chỉ được CHỜ", res.State)
	}
	_, dinhTong, goi, _ := ag.doc()
	if goi != 4 {
		t.Fatalf("phải chạy đủ 4 bước, chỉ chạy %d", goi)
	}
	if dinhTong != 3 {
		t.Fatalf("trần provider claude mặc định là 3 nên đỉnh phải là 3, đo được %d", dinhTong)
	}
}

// `foreach` cũng phải qua cổng.
//
// Nó có semaphore RIÊNG (runForEach), nên canh ở runWave thôi là bịt một đường
// hở một đường — đúng lớp lỗi cả ngày hôm nay đi sửa. Một bước `foreach` trên 4
// mục vẫn là 4 phiên trên một tài khoản.
func TestForEachCungPhaiQuaCongTran(t *testing.T) {
	r, ag, _ := runnerCoTran(t, tranMacDinh(), nil, 120*time.Millisecond)
	f := Flow{Name: "lap", Steps: []Step{{
		ID: "lap", Type: TypeAgent, Profile: "claude:tns",
		ForEach: "vars.ds", Prompt: "làm {{item}}",
	}}}
	res, err := r.Start(context.Background(), f, t.TempDir(), map[string]string{"ds": "a\nb\nc\nd"})
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("trạng thái %s", res.State)
	}
	dinh, _, goi, _ := ag.doc()
	if goi != 4 {
		t.Fatalf("phải chạy đủ 4 lượt lặp, chỉ chạy %d", goi)
	}
	if dinh["claude:tns"] > 2 {
		t.Fatalf("FOREACH LỌT CỔNG: đỉnh cùng lúc %d, trần hồ sơ 2", dinh["claude:tns"])
	}
}

// `copies` lớn hơn trần thì CẮT XUỐNG rồi chạy, chứ không xếp hàng chờ một con
// số không bao giờ tới — và bộ chạy phải nhận con số ĐÃ CẮT.
func TestCopiesVuotTranThiCatXuongChuKhongTreo(t *testing.T) {
	r, ag, noi := runnerCoTran(t, tranMacDinh(), nil, 10*time.Millisecond)
	f := Flow{Name: "cat", Steps: []Step{
		{ID: "b1", Type: TypeAgent, Profile: "claude:tns", Copies: 4, Prompt: "x"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("trạng thái %s", res.State)
	}
	_, _, _, capSo := ag.doc()
	if len(capSo) != 1 || capSo[0] != 2 {
		t.Fatalf("copies phải bị cắt còn 2 (trần hồ sơ), bộ chạy nhận %v", capSo)
	}
	if !strings.Contains(strings.Join(*noi, "\n"), "TRẦN ĐỒNG THỜI cắt 4 phiên xuống 2") {
		t.Fatalf("cắt mà không nói ra:\n%s", strings.Join(*noi, "\n"))
	}
}

// Không cắm cổng thì mọi thứ chạy y như cũ. Bài này canh cho các đường chưa cắm
// (và cho toàn bộ test cũ của gói) không đổi hành vi trong im lặng.
func TestKhongCamCongThiKhongApTran(t *testing.T) {
	r, ag, db := newRunner(t)
	r.MaxParallel = 8
	do := newAgentDo(3 * time.Second)
	do.ChoDu = 4
	r.Agent = do
	_ = ag
	_ = db
	f := Flow{Name: "khong-cong", Steps: buocAgent(4, "claude:tns")}
	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	dinh, _, _, _ := do.doc()
	if dinh["claude:tns"] != 4 {
		t.Fatalf("không cắm cổng thì phải giữ nguyên hành vi cũ (4 cùng lúc), đỉnh %d", dinh["claude:tns"])
	}
}
