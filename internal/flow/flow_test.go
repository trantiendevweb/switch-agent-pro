package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func errs(ps []Problem) []Problem {
	var out []Problem
	for _, p := range ps {
		if !p.Warn {
			out = append(out, p)
		}
	}
	return out
}

// Ba flow mẫu phải luôn hợp lệ — chúng là mẫu để người dùng chép theo.
func TestBuiltinHopLe(t *testing.T) {
	for name, f := range Builtin() {
		if e := errs(Validate(f)); len(e) > 0 {
			t.Fatalf("flow mẫu %q có lỗi: %v", name, e)
		}
		if _, err := Order(f); err != nil {
			t.Fatalf("flow mẫu %q không sắp xếp được: %v", name, err)
		}
	}
}

// Chu trình là lỗi nguy hiểm nhất: không bắt thì bộ thực thi sẽ chạy vô tận.
func TestBatChuTrinh(t *testing.T) {
	f := Flow{Name: "vong", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "x", Needs: []string{"c"}},
		{ID: "b", Type: TypeNotify, Message: "x", Needs: []string{"a"}},
		{ID: "c", Type: TypeNotify, Message: "x", Needs: []string{"b"}},
	}}
	e := errs(Validate(f))
	if len(e) == 0 {
		t.Fatal("có chu trình a→c→b→a mà không báo lỗi")
	}
	found := false
	for _, p := range e {
		if strings.Contains(p.Msg, "chu trình") {
			found = true
		}
	}
	if !found {
		t.Fatalf("báo lỗi nhưng không nói là chu trình: %v", e)
	}
	if _, err := Order(f); err == nil {
		t.Fatal("Order phải lỗi khi có chu trình")
	}
}

func TestBatPhuThuocKhongTonTai(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "x", Needs: []string{"khong-co"}},
	}}
	e := errs(Validate(f))
	if len(e) == 0 {
		t.Fatal("needs trỏ tới bước không tồn tại mà không báo")
	}
}

func TestBatIdTrungVaIdXau(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "m"},
		{ID: "a", Type: TypeNotify, Message: "m"},
		{ID: "Có Dấu", Type: TypeNotify, Message: "m"},
	}}
	e := errs(Validate(f))
	if len(e) < 2 {
		t.Fatalf("phải bắt cả id trùng lẫn id xấu, chỉ được %d lỗi: %v", len(e), e)
	}
}

func TestBatTypeLa(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{{ID: "a", Type: "bay-len-troi"}}}
	if len(errs(Validate(f))) == 0 {
		t.Fatal("type lạ mà không báo lỗi")
	}
}

// Trung thực về năng lực: loại đã thiết kế nhưng chưa chạy được phải CẢNH BÁO,
// không phải im lặng chấp nhận.
func TestCanhBaoTypeChuaChayDuoc(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{{ID: "a", Type: TypeMerge}}}
	ps := Validate(f)
	if len(errs(ps)) > 0 {
		t.Fatalf("type hợp lệ nhưng chưa hỗ trợ thì KHÔNG nên là lỗi: %v", errs(ps))
	}
	warned := false
	for _, p := range ps {
		if p.Warn && strings.Contains(p.Msg, "CHƯA chạy được") {
			warned = true
		}
	}
	if !warned {
		t.Fatal("thiếu cảnh báo cho type chưa chạy được")
	}
}

func TestShellPhaiDungArgv(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{{ID: "a", Type: TypeShell}}}
	if len(errs(Validate(f))) == 0 {
		t.Fatal("bước shell thiếu run mà không báo")
	}
	ok := Flow{Name: "x", Steps: []Step{{ID: "a", Type: TypeShell, Run: []string{"go", "test"}}}}
	if e := errs(Validate(ok)); len(e) > 0 {
		t.Fatalf("shell có run mà vẫn lỗi: %v", e)
	}
}

func TestFallbackPhaiTonTai(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "m", OnFailure: OnFailFallback, Fallback: "khong-co"},
	}}
	if len(errs(Validate(f))) == 0 {
		t.Fatal("fallback trỏ bậy mà không báo")
	}
}

// Thứ tự chạy phải tôn trọng phụ thuộc, và ổn định giữa các lần gọi.
func TestOrderTonTrongPhuThuocVaOnDinh(t *testing.T) {
	f := Flow{Name: "x", Steps: []Step{
		{ID: "c", Type: TypeNotify, Message: "m", Needs: []string{"a", "b"}},
		{ID: "b", Type: TypeNotify, Message: "m", Needs: []string{"a"}},
		{ID: "a", Type: TypeNotify, Message: "m"},
	}}
	first, err := Order(f)
	if err != nil {
		t.Fatal(err)
	}
	pos := map[string]int{}
	for i, s := range first {
		pos[s.ID] = i
	}
	if !(pos["a"] < pos["b"] && pos["b"] < pos["c"]) {
		t.Fatalf("thứ tự sai: %v", ids(first))
	}
	second, _ := Order(f)
	if strings.Join(ids(first), ",") != strings.Join(ids(second), ",") {
		t.Fatalf("thứ tự không ổn định: %v rồi %v", ids(first), ids(second))
	}
}

func ids(ss []Step) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.ID
	}
	return out
}

func TestExpandBien(t *testing.T) {
	got := Expand("làm {{task}} trên {{repo}}", map[string]string{"task": "việc A", "repo": "X"})
	if got != "làm việc A trên X" {
		t.Fatalf("Expand = %q", got)
	}
}

// flows.toml của dự án đè được flow mẫu cùng tên.
func TestFileDuAnDeFlowMau(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".sagent"), 0o755); err != nil {
		t.Fatal(err)
	}
	// cần project.toml để tìm được thư mục .sagent
	if err := os.WriteFile(filepath.Join(proj, ".sagent", "project.toml"), []byte("version=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := `version = 1
[flow.fanout]
desc = "bản của tôi"
[[flow.fanout.step]]
id = "x"
type = "notify"
message = "chào"
`
	if err := os.WriteFile(filepath.Join(proj, ".sagent", "flows.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	flows, srcs, err := Load(proj)
	if err != nil {
		t.Fatal(err)
	}
	if len(srcs) != 1 {
		t.Fatalf("muốn 1 nguồn, được %v", srcs)
	}
	if flows["fanout"].Desc != "bản của tôi" {
		t.Fatalf("flows.toml dự án phải đè flow mẫu, desc = %q", flows["fanout"].Desc)
	}
	// các flow mẫu khác vẫn còn
	if _, ok := flows["squad"]; !ok {
		t.Fatal("flow mẫu không bị đè thì phải giữ nguyên")
	}
}

func TestFileHongBaoLoiRo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()
	os.MkdirAll(filepath.Join(proj, ".sagent"), 0o755)
	os.WriteFile(filepath.Join(proj, ".sagent", "project.toml"), []byte("version=1\n"), 0o644)
	os.WriteFile(filepath.Join(proj, ".sagent", "flows.toml"), []byte("version = 99\n"), 0o644)
	if _, _, err := Load(proj); err == nil {
		t.Fatal("version lạ mà không báo lỗi")
	}
}
