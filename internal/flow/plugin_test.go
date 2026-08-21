package flow

// Node `plugin` nhìn từ phía flow: kiểm tra tĩnh, và đường đi của dữ liệu.
//
// Bài test chạy plugin THẬT (build binary, bật tiến trình con) nằm ở
// internal/plugin/e2e_test.go — nó phải ở đó vì gói này cố ý KHÔNG import
// internal/plugin. Ở đây chỉ đo phần flow chịu trách nhiệm: validate, chuyền
// {{steps.x.output}} sang đầu vào, và cư xử khi bộ chạy chưa được cắm.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// fakePlugin ghi lại thứ nó nhận được. Có khoá vì runner chạy các bước cùng đợt
// song song — cùng lý do với fakeAgent.
type fakePlugin struct {
	mu     sync.Mutex
	ten    []string
	vao    []string
	thamSo []map[string]string
	ra     string
	hong   error
}

func (f *fakePlugin) GoiPlugin(_ context.Context, ten, vao string, ts map[string]string) (string, error) {
	f.mu.Lock()
	f.ten = append(f.ten, ten)
	f.vao = append(f.vao, vao)
	f.thamSo = append(f.thamSo, ts)
	ra, hong := f.ra, f.hong
	f.mu.Unlock()
	if hong != nil {
		return "", hong
	}
	return ra, nil
}

func TestValidateBuocPluginThieuTen(t *testing.T) {
	ps := Validate(Flow{Name: "x", Steps: []Step{{ID: "a", Type: TypePlugin}}})
	if len(ps) == 0 {
		t.Fatal("bước plugin không khai `plugin` mà validate im lặng")
	}
	if !strings.Contains(ps[0].Msg, "TÊN plugin") {
		t.Errorf("thông điệp không chỉ đường: %s", ps[0].Msg)
	}
}

// `plugin` phải là TÊN, không phải đường dẫn — flows.toml là file người ta gửi
// cho nhau, và một đường dẫn trong đó là lời mời chạy binary tuỳ ý.
func TestValidateBuocPluginKhongNhanDuongDan(t *testing.T) {
	for _, xau := range []string{`C:\tools\x.exe`, "../x", "./x", "/bin/sh"} {
		ps := Validate(Flow{Name: "x", Steps: []Step{{ID: "a", Type: TypePlugin, Plugin: xau}}})
		if len(ps) == 0 {
			t.Errorf("plugin = %q lọt qua validate", xau)
		}
	}
}

// Khai `plugin` ở một bước type khác là một dòng CHẾT: nó trông như có tác dụng.
func TestKhaiPluginOBuocKhongPhaiTypePluginThiKeu(t *testing.T) {
	ps := Validate(Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypeShell, Run: []string{"go", "version"}, Plugin: "tom-luoc"},
	}})
	if len(ps) == 0 {
		t.Fatal("bước shell khai `plugin` mà validate im lặng — người viết flow sẽ tưởng nó có chạy")
	}
}

func TestValidateBuocPluginDungThiKhongKeu(t *testing.T) {
	ps := Validate(Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypePlugin, Plugin: "tom-luoc", Vao: "xin chào"},
	}})
	if len(ps) != 0 {
		t.Fatalf("bước plugin hợp lệ mà validate kêu: %v", ps)
	}
}

// Chưa cắm bộ chạy thì bước phải HỎNG kèm lời giải thích, không được lặng lẽ
// trả về chuỗi rỗng cho bước sau dùng.
func TestChuaCamBoChayPluginThiBuocHongRoRang(t *testing.T) {
	r, _, db := newRunner(t)
	f := Flow{Name: "x", Steps: []Step{{ID: "a", Type: TypePlugin, Plugin: "tom-luoc", Vao: "x"}}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("chưa cắm bộ chạy mà lượt chạy vẫn %s", res.State)
	}
	buoc, err := db.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buoc["a"].Msg, "chưa được cắm") {
		t.Errorf("thông điệp không nói ra nguyên nhân: %q", buoc["a"].Msg)
	}
}

// Kết quả bước trước phải tới được đầu vào của plugin, và tham_so phải đi kèm.
func TestKetQuaBuocTruocChayVaoDauVaoCuaPlugin(t *testing.T) {
	r, ag, db := newRunner(t)
	ag.output = "KET QUA CUA BUOC TRUOC"
	pl := &fakePlugin{ra: "da tom luoc"}
	r.Plugin = pl

	f := Flow{Name: "x", Steps: []Step{
		{ID: "truoc", Type: TypeAgent, Prompt: "làm việc"},
		{ID: "gon", Type: TypePlugin, Needs: []string{"truoc"}, Plugin: "tom-luoc",
			Vao: "{{steps.truoc.output}}", ThamSo: map[string]string{"so_dong": "3"}},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("lượt chạy không xong: %s", res.State)
	}
	if len(pl.vao) != 1 || pl.vao[0] != "KET QUA CUA BUOC TRUOC" {
		t.Fatalf("đầu vào của plugin sai: %q", pl.vao)
	}
	if pl.ten[0] != "tom-luoc" {
		t.Errorf("gọi nhầm plugin: %q", pl.ten[0])
	}
	if pl.thamSo[0]["so_dong"] != "3" {
		t.Errorf("tham_so không tới nơi: %v", pl.thamSo[0])
	}
	buoc, err := db.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if buoc["gon"].Output != "da tom luoc" {
		t.Errorf("kết quả plugin không được ghi lại: %q", buoc["gon"].Output)
	}
}

// Plugin hỏng thì bước hỏng — và on_failure của flow vẫn có tác dụng như mọi
// loại node khác. Plugin không được là một ngoại lệ trong luật xử lý hỏng.
func TestPluginHongThiTheoDungLuatOnFailure(t *testing.T) {
	r, _, db := newRunner(t)
	r.Plugin = &fakePlugin{hong: fmt.Errorf("plugin chết giữa chừng")}

	f := Flow{Name: "x", Steps: []Step{
		{ID: "gon", Type: TypePlugin, Plugin: "tom-luoc", Vao: "x", OnFailure: OnFailContinue},
		{ID: "sau", Type: TypeNotify, Needs: []string{"gon"}, Message: "vẫn chạy"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	buoc, err := db.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if buoc["gon"].State != store.StepFailed {
		t.Errorf("plugin trả lỗi mà bước vẫn %s", buoc["gon"].State)
	}
	if !strings.Contains(buoc["gon"].Msg, "chết giữa chừng") {
		t.Errorf("lỗi nguyên văn của plugin bị nuốt: %q", buoc["gon"].Msg)
	}
	if buoc["sau"].State != store.StepDone {
		t.Errorf("on_failure = continue mà bước sau vẫn không chạy: %s", buoc["sau"].State)
	}
}

// Bước plugin đọc kết quả của một bước KHÔNG để lại gì thì phải nhận một câu nói
// rõ là thiếu, chứ không phải chuỗi `{{steps.x.output}}` thô — nếu không, plugin
// sẽ đếm cái placeholder đó như dữ liệu thật và trả về một con số có vẻ đúng.
func TestBuocKhongDeLaiGiThiPluginNhanCauNoiRo(t *testing.T) {
	r, _, _ := newRunner(t)
	pl := &fakePlugin{ra: "xong"}
	r.Plugin = pl

	f := Flow{Name: "x", Steps: []Step{
		{ID: "im", Type: TypeNotify, Message: ""},
		{ID: "gon", Type: TypePlugin, Needs: []string{"im"}, Plugin: "tom-luoc",
			Vao: "{{steps.im.output}}"},
	}}
	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	if len(pl.vao) != 1 {
		t.Fatalf("plugin được gọi %d lần", len(pl.vao))
	}
	if strings.Contains(pl.vao[0], "{{steps.") {
		t.Errorf("placeholder thô lọt vào đầu vào của plugin: %q", pl.vao[0])
	}
}
