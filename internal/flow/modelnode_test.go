package flow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Node `model` goi THANG model API — duong thu hai cua du an, khong qua CLI agent.
//
// VI SAO BAT NO LEN 20/08: CLI grok hong VINH VIEN — HTTP 410 "Live search is
// deprecated. Please switch to the Agent Tools API". Hai luot lien (#46, #47)
// nguoi soi tra ve dung mot cau xin loi. Cung nha cung cap do qua duong API thi
// van tra loi binh thuong: da do 13,7s / 903 token.
//
// Truoc do TypeModel duoc khai tu dau du an voi ghi chu "cho duong API (Pha
// 1/4)" va implemented[TypeModel] = false. Duong do nay da co, da do that, co ca
// bo chuyen route du phong va so loi goi.

type fakeModel struct {
	mu      sync.Mutex
	routes  []string
	prompts []string
	out     string
	err     error
}

func (f *fakeModel) GoiModel(_ context.Context, route, prompt string) (KetQuaAgent, error) {
	f.mu.Lock()
	f.routes = append(f.routes, route)
	f.prompts = append(f.prompts, prompt)
	out, err := f.out, f.err
	f.mu.Unlock()
	if err != nil {
		return KetQuaAgent{}, err
	}
	return KetQuaAgent{Output: out, TokenVao: 100, TokenRa: 50}, nil
}

func TestNodeModelGoiDuongAPIChuKhongGoiAgent(t *testing.T) {
	r, ag, db := newRunner(t)
	fm := &fakeModel{out: "TNS: NEN TRON — on"}
	r.Model = fm

	f := Flow{Name: "thu", Steps: []Step{
		{ID: "soi", Type: TypeModel, Prompt: "soi giup: {{viec}}"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), map[string]string{"viec": "sua X"})
	if err != nil {
		t.Fatal(err)
	}
	if tt := trangThaiBuoc(t, db, res.RunID, "soi"); tt != store.StepDone {
		t.Fatalf("node model khong chay duoc: state=%q", tt)
	}
	// KHONG duoc dung sang duong agent: hai duong tieu hai loai tien khac nhau.
	if ag.soLanGoi() != 0 {
		t.Errorf("node model lai goi agent %d lan — sai duong, va sai ca loai chi phi",
			ag.soLanGoi())
	}
	if len(fm.prompts) != 1 {
		t.Fatalf("goi duong API %d lan, cho doi 1", len(fm.prompts))
	}
	// Bien phai duoc thay TRUOC khi gui di, y het node agent.
	if !strings.Contains(fm.prompts[0], "sua X") {
		t.Errorf("prompt gui di chua thay bien: %q", fm.prompts[0])
	}
	// route rong = default_route roi toi du phong, cung luat voi `sagent api`.
	if fm.routes[0] != "" {
		t.Errorf("route = %q, cho doi rong (default + du phong)", fm.routes[0])
	}
}

// Chua cam duong API thi phai BAO RO, khong duoc im lang bo qua buoc.
//
// Im lang bo qua o day nghia la: nguoi soi bien mat, lo trinh van chay tiep, va
// khong ai biet. Dung kieu hong ma hop dong dau ra sinh ra de chan.
func TestNodeModelChuaCamDuongAPIThiBaoRo(t *testing.T) {
	r, _, db := newRunner(t)
	r.Model = nil // chua cam

	f := Flow{Name: "thu", Steps: []Step{{ID: "soi", Type: TypeModel, Prompt: "soi"}}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if tt := trangThaiBuoc(t, db, res.RunID, "soi"); tt != store.StepFailed {
		t.Errorf("chua cam duong API ma buoc ra %q — phai la %q", tt, store.StepFailed)
	}
}

// Node model phai duoc danh dau CHAY DUOC, khong thi bo kiem tra flow se tu choi
// moi flow dung no truoc ca khi chay.
func TestNodeModelDuocDanhDauChayDuoc(t *testing.T) {
	if !implemented[TypeModel] {
		t.Error("implemented[TypeModel] van false — flow dung node model se bi tu choi " +
			"luc kiem tra, du bo chay da cai xong")
	}
}

// Va nguoi soi cua doi-4 phai THAT SU dung duong API, khong quay ve CLI grok.
func TestNguoiSoiCuaDoi4DungDuongAPI(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", ".sagent", "flows.toml"))
	if err != nil {
		t.Skipf("khong doc duoc flows.toml: %v", err)
	}
	s := string(b)
	i := strings.Index(s, "[flow.doi-4]")
	if i < 0 {
		t.Skip("khong co flow doi-4")
	}
	doan := s[i:]
	if j := strings.Index(doan[1:], "\n  [flow."); j > 0 {
		doan = doan[:j]
	}
	k := strings.Index(doan, `id = "soi"`)
	if k < 0 {
		t.Fatal("doi-4 khong con buoc soi")
	}
	khoi := doan[k:]
	if j := strings.Index(khoi, "[[flow."); j > 0 {
		khoi = khoi[:j]
	}
	if !strings.Contains(khoi, `type = "model"`) {
		t.Error("nguoi soi cua doi-4 khong dung node model — CLI grok da hong vinh vien " +
			"(HTTP 410), quay ve no la moi luot chay lai mat nguoi soi")
	}
	if strings.Contains(khoi, `profile = "grok:api"`) {
		t.Error("nguoi soi cua doi-4 con khai profile CLI — node model di duong API, " +
			"khong dung profile")
	}
}
