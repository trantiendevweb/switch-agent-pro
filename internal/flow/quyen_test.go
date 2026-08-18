package flow

import (
	"context"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Cờ tự-duyệt-quyền cho agent chạy MỌI tool, kể cả xoá file và lệnh tuỳ ý,
// trong worktree của repo thật. Nó phải TẮT trừ khi bước khai rõ — mặc định
// rò sang "bật" là lỗ hổng, không phải tiện lợi.
func TestMacDinhKhongTuDuyetQuyen(t *testing.T) {
	r, ag, _ := newRunner(t)
	f := Flow{Name: "mac-dinh", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "việc thường"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("phải xong, được %s", res.State)
	}
	if len(ag.quyen) != 1 {
		t.Fatalf("phải gọi agent 1 lần, được %d", len(ag.quyen))
	}
	if ag.quyen[0] {
		t.Fatal("BƯỚC KHÔNG KHAI GÌ MÀ ĐƯỢC TỰ DUYỆT MỌI QUYỀN")
	}
}

// Khai rõ thì mới bật, và chỉ bật ĐÚNG BƯỚC ĐÓ — không lây sang bước khác.
func TestKhaiRoThiBatVaKhongLay(t *testing.T) {
	r, ag, _ := newRunner(t)
	f := Flow{Name: "khai-ro", Steps: []Step{
		{ID: "mo", Type: TypeAgent, Prompt: "cần quyền", TuDuyetQuyen: true},
		{ID: "thuong", Type: TypeAgent, Needs: []string{"mo"}, Prompt: "không cần"},
	}}
	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	if len(ag.quyen) != 2 {
		t.Fatalf("phải gọi agent 2 lần, được %d", len(ag.quyen))
	}
	if !ag.quyen[0] {
		t.Fatal("bước khai tu_duyet_quyen mà cờ không tới nơi")
	}
	if ag.quyen[1] {
		t.Fatal("CỜ LÂY SANG BƯỚC SAU — bước không khai gì cũng được duyệt mọi quyền")
	}
}
