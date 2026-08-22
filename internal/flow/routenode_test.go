package flow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// routeGia là phần cắm chọn đường giả: nó KHÔNG chạm mạng, chỉ trả về đường đầu
// tiên có trong danh sách `song`.
type routeGia struct {
	mu      sync.Mutex
	song    map[string]bool
	daHoi   [][]string // từng lần được hỏi, để khẳng định ứng viên đi tới đúng
	daCan   [][]string // từng lần được hỏi, phần `can` — xem nangluc_test.go
	batBuoc string     // khác rỗng = luôn trả về đúng đường này
}

func (r *routeGia) ChonRoute(_ context.Context, ungVien, can []string) (KetQuaRoute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.daHoi = append(r.daHoi, append([]string(nil), ungVien...))
	r.daCan = append(r.daCan, append([]string(nil), can...))
	if r.batBuoc != "" {
		return KetQuaRoute{Ten: r.batBuoc}, nil
	}
	var kq KetQuaRoute
	for _, t := range ungVien {
		if r.song[t] {
			kq.Ten = t
			kq.NhatKy = append(kq.NhatKy, t+": dùng được - CHỌN")
			return kq, nil
		}
		kq.NhatKy = append(kq.NhatKy, t+": chết")
	}
	return kq, fmt.Errorf("không đường nào dùng được")
}

// modelGia ghi lại route mà nó ĐƯỢC GỌI BẰNG.
type modelGia struct {
	mu    sync.Mutex
	route []string
}

func (m *modelGia) GoiModel(_ context.Context, route, prompt string) (KetQuaAgent, error) {
	m.mu.Lock()
	m.route = append(m.route, route)
	m.mu.Unlock()
	return KetQuaAgent{Output: "trả lời qua " + route}, nil
}

func (m *modelGia) daDung() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.route...)
}

// ĐÂY LÀ BÀI TEST QUAN TRỌNG NHẤT của node `route`.
//
// Nó chạy một lượt THẬT qua Runner.Start và khẳng định cả CHUỖI: bước `route`
// loại đường chết, chọn đường sống, và bước `model` sau nó ĐI ĐÚNG ĐƯỜNG ĐÓ.
//
// Mắt xích dễ đứt nhất nằm ở khâu cuối: trước bản này `Step.Route` đi thẳng vào
// lời gọi mà KHÔNG qua Expand, nên viết `route = "{{steps.chon.output}}"` cho
// ra một tên route không tồn tại — cả node `route` sẽ chạy đúng mà vô dụng.
func TestRouteChonDuongRoiChuyenChoBuocSau(t *testing.T) {
	r, _, db := newRunner(t)
	rg := &routeGia{song: map[string]bool{"deepseek": true}} // grok chết
	mg := &modelGia{}
	r.Route, r.Model = rg, mg

	f := Flow{Name: "duong", Steps: []Step{
		{ID: "chon", Type: TypeRoute, Routes: []string{"grok", "deepseek"}},
		{ID: "hoi", Type: TypeModel, Needs: []string{"chon"},
			Route: "{{steps.chon.output}}", Prompt: "một câu hỏi"},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
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
	// Output của bước `route` là ĐÚNG cái tên, không gì khác: nó đi thẳng vào
	// trường `route` của bước sau, nơi một ký tự thừa là một đường không có.
	if got := steps["chon"].Output; got != "deepseek" {
		t.Errorf("output bước route = %q, muốn đúng \"deepseek\" (không thêm chữ nào)", got)
	}
	if got := rg.daHoi; len(got) != 1 || strings.Join(got[0], ",") != "grok,deepseek" {
		t.Errorf("ứng viên tới phần cắm sai thứ tự hoặc sai số lần: %v", got)
	}

	// Mắt xích cuối.
	if got := mg.daDung(); len(got) != 1 || got[0] != "deepseek" {
		t.Fatalf("bước `model` KHÔNG đi theo đường đã chọn — nó được gọi với route %v.\n"+
			"Đường mà node `route` chọn không tới được bước sau thì cả node này vô dụng.", got)
	}
}

// Hết đường thì DỪNG HẲN, không đoán bừa một cái tên.
//
// Đoán bừa nghĩa là bước sau đem cái tên ấy đi gọi thật: tốn thời gian chờ,
// tốn tiền nếu trúng, và hỏng bằng một thông báo chẳng nhắc gì tới nguyên nhân.
func TestRouteHetDuongThiDungChuKhongDoanBua(t *testing.T) {
	r, _, db := newRunner(t)
	rg := &routeGia{song: map[string]bool{}} // không đường nào sống
	mg := &modelGia{}
	r.Route, r.Model = rg, mg

	f := Flow{Name: "duong", Steps: []Step{
		{ID: "chon", Type: TypeRoute, Routes: []string{"grok", "deepseek"}},
		{ID: "hoi", Type: TypeModel, Needs: []string{"chon"},
			Route: "{{steps.chon.output}}", Prompt: "một câu hỏi"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("không đường nào sống mà lượt chạy vẫn %s", res.State)
	}
	if n := len(mg.daDung()); n != 0 {
		t.Fatalf("bước `model` vẫn được gọi %d lần dù chưa chọn được đường nào", n)
	}
	steps, _ := db.Steps(res.RunID)
	if steps["chon"].State != store.StepFailed {
		t.Errorf("bước route không được ghi là hỏng: %s", steps["chon"].State)
	}
}

// Bước `model` trỏ route vào một bước KHÔNG để lại kết quả thì phải dừng ngay
// và nói rõ thiếu của bước nào — KHÔNG được chốt placeholder thành câu tiếng
// Việt rồi đem cả câu đó đi tra tên route.
//
// Cùng luật với tham số của bước shell, và cùng lý do: thông báo lỗi sinh ra từ
// một giá trị bịa sẽ chỉ người đọc đi sai hướng.
func TestRouteThieuKetQuaThiNoiThangChuKhongChotBua(t *testing.T) {
	r := &Runner{Model: &modelGia{}}
	_, err := r.do(context.Background(), Step{
		ID: "hoi", Type: TypeModel, Route: "{{steps.chua-co.output}}", Prompt: "hỏi",
	}, map[string]string{})
	if err == nil {
		t.Fatal("route trỏ vào bước không có kết quả mà bước vẫn chạy")
	}
	if !strings.Contains(err.Error(), "chua-co") {
		t.Errorf("lỗi không nói ra thiếu kết quả của bước nào: %v", err)
	}
}

// Khai `routes` ở một bước KHÔNG phải type route là một dòng chết: nó nằm đó
// trông như có tác dụng. Cùng lớp với `plugin` khai nhầm chỗ.
func TestValidateRoutesKhaiNhamCho(t *testing.T) {
	ps := Validate(Flow{Name: "x", Steps: []Step{
		{ID: "hoi", Type: TypeModel, Prompt: "p", Routes: []string{"grok"}},
	}})
	if !coLoiChua(ps, "hoi", "`routes`", false) {
		t.Fatalf("khai routes ở bước model mà Validate không báo LỖI: %v", ps)
	}
}

// Bước `model` trỏ route vào một bước KHÔNG phải type route: đầu ra của bước đó
// là văn bản, không phải tên đường. Bắt lúc kiểm, đừng để tới lúc chạy.
func TestValidateRouteTroVaoBuocKhongPhaiRoute(t *testing.T) {
	ps := Validate(Flow{Name: "x", Steps: []Step{
		{ID: "viet", Type: TypeAgent, Prompt: "viết gì đó"},
		{ID: "hoi", Type: TypeModel, Needs: []string{"viet"},
			Route: "{{steps.viet.output}}", Prompt: "p"},
	}})
	if !coLoiChua(ps, "hoi", "không phải type = \"route\"", false) {
		t.Fatalf("route trỏ vào bước thường mà Validate không báo LỖI: %v", ps)
	}
}

// Chọn được đường mà không bước nào dùng thì cả bước chọn là công cốc — và
// người viết flow thường tưởng nó tự áp cho các bước sau.
func TestValidateRouteChonRoiKhongAiDung(t *testing.T) {
	ps := Validate(Flow{Name: "x", Steps: []Step{
		{ID: "chon", Type: TypeRoute, Routes: []string{"grok"}},
		{ID: "hoi", Type: TypeModel, Needs: []string{"chon"}, Prompt: "p"},
	}})
	if !coLoiChua(ps, "chon", "không bước nào dùng", true) {
		t.Fatalf("bước chọn đường không ai dùng mà không có cảnh báo: %v", ps)
	}
}

// Khoá idempotency phải cuốn theo ĐƯỜNG THẬT SỰ ĐI, không phải chuỗi thô trong
// flows.toml.
//
// Không thì hai lượt chạy đi HAI ĐƯỜNG KHÁC NHAU vẫn ra cùng một khoá, và lượt
// sau dùng lại câu trả lời của một mô hình khác hẳn mà không một dòng nào nói.
func TestKhoaIdemDoiTheoDuongThatSuDi(t *testing.T) {
	s := Step{ID: "hoi", Type: TypeModel, Route: "{{steps.chon.output}}",
		Prompt: "cùng một câu hỏi", Idempotent: true}
	k1 := KhoaIdem(s, map[string]string{"steps.chon.output": "grok"})
	k2 := KhoaIdem(s, map[string]string{"steps.chon.output": "deepseek"})
	if k1 == "" {
		t.Fatal("bước bật idempotent mà không sinh khoá")
	}
	if k1 == k2 {
		t.Fatal("đi hai đường khác nhau mà khoá idempotency GIỐNG NHAU — " +
			"lượt sau sẽ dùng lại câu trả lời của mô hình khác")
	}
}
