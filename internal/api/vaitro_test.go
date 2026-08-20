package api

import (
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// Kế hoạch chạy khan phải nói VAI TRÒ từng bước, cạnh tài khoản và model.
//
// Vì sao đáng có: đọc kế hoạch trước khi bấm chạy là lúc người ta hỏi "ai làm
// gì". Tài khoản trả lời "bằng nick nào", model trả lời "tốn bao nhiêu", còn
// vai trò trả lời "với tư cách gì" — thiếu nó thì `sagent flow run x --kho` và
// POST /api/flow/kho biết ít hơn chính file flows.toml mà chúng vừa đọc.
func TestChayKhoNoiVaiTroTungBuoc(t *testing.T) {
	khoTam(t)
	dir := t.TempDir()
	f := flow.Flow{
		Name: "vai", Desc: "flow thử vai trò",
		Steps: []flow.Step{
			{ID: "chia", Type: flow.TypeAgent, Profile: "claude:phu", Model: "sonnet",
				VaiTro: flow.VaiLeader, Prompt: "chia việc"},
			// Bước shell cũng có vai: `kiem-1` của doi-4 là việc của tester.
			{ID: "kiem", Type: flow.TypeShell, VaiTro: flow.VaiTester,
				Run: []string{"go", "version"}, Needs: []string{"chia"}},
			// Cố ý KHÔNG khai vai: phải ra rỗng, không được đoán hộ.
			{ID: "bao", Type: flow.TypeNotify, Message: "xong", Needs: []string{"kiem"}},
		},
	}
	if _, err := flow.Save(dir, f); err != nil {
		t.Fatal(err)
	}

	kh, err := (&API{}).FlowChayKho(dir, "vai", nil, Addr{})
	if err != nil {
		t.Fatal(err)
	}
	vai := map[string]string{}
	for _, d := range kh.Dot {
		for _, b := range d.Buoc {
			vai[b.ID] = b.VaiTro
		}
	}
	if vai["chia"] != flow.VaiLeader {
		t.Fatalf("bước agent mất vai trong kế hoạch chạy khan: được %q", vai["chia"])
	}
	if vai["kiem"] != flow.VaiTester {
		t.Fatalf("bước shell cũng phải mang vai: được %q", vai["kiem"])
	}
	if vai["bao"] != "" {
		t.Fatalf("bước không khai vai phải RỖNG, máy lại đoán ra %q", vai["bao"])
	}
}

// Vai lạ đi qua chạy khan thành CẢNH BÁO trong kế hoạch, và kế hoạch vẫn dựng
// được — đúng luật "chỉ nói, không chặn".
func TestChayKhoBaoVaiLaMaVanDungKeHoach(t *testing.T) {
	khoTam(t)
	dir := t.TempDir()
	f := flow.Flow{Name: "vai", Steps: []flow.Step{
		{ID: "a", Type: flow.TypeNotify, Message: "xong", VaiTro: "designer"},
	}}
	if _, err := flow.Save(dir, f); err != nil {
		t.Fatal(err)
	}
	kh, err := (&API{}).FlowChayKho(dir, "vai", nil, Addr{})
	if err != nil {
		t.Fatalf("vai lạ KHÔNG được chặn kế hoạch: %v", err)
	}
	if len(kh.Dot) == 0 {
		t.Fatal("vai lạ mà mất luôn kế hoạch — lẽ ra chỉ cảnh báo")
	}
	found := false
	for _, v := range kh.Van {
		if v.Warn && v.Buoc == "a" {
			found = true
		}
		if !v.Warn {
			t.Fatalf("không được biến vai lạ thành lỗi: %+v", v)
		}
	}
	if !found {
		t.Fatalf("kế hoạch phải mang cảnh báo vai lạ ra mặt web: %+v", kh.Van)
	}
	// Vai lạ vẫn phải hiện nguyên văn: giấu đi thì người dùng không thấy chỗ sai.
	if kh.Dot[0].Buoc[0].VaiTro != "designer" {
		t.Fatalf("vai lạ phải hiện nguyên văn, được %q", kh.Dot[0].Buoc[0].VaiTro)
	}
}
