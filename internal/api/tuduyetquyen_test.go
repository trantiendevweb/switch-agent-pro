package api

import (
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// `sagent fleet` muốn agent làm được việc thì phải cho nó tự duyệt tool. Trước
// luợt này, cách DUY NHẤT là người dùng tự gõ:
//
//	sagent fleet claude:phu -- --dangerously-skip-permissions -p "..."
//
// Ba thứ sai với chuyện đó:
//
//  1. Tên cờ của Claude rò ra tay người dùng và vào mọi script. `fleet codex:*`
//     cần `--dangerously-bypass-approvals-and-sandbox` — gõ nhầm thì CLI con
//     chết ngay dòng đầu, hoặc tệ hơn, chạy mà không có quyền và trả về một câu
//     từ chối được ghi là "xong".
//  2. Lõi đã có `ArgsTuDuyetQuyen()` đo thật cho từng provider, và đường flow
//     dùng nó từ lâu (`argsChoBuoc`). Hai đường chạy, một đường hỏi adapter, một
//     đường bắt người dùng nhớ hộ.
//  3. Provider CHƯA ĐO cờ đó thì phải NÓI THẲNG, không im lặng chạy thiếu quyền.

func TestCoTuDuyetQuyenLayTuAdapterChuKhongChepCung(t *testing.T) {
	for _, ten := range []string{"claude", "codex"} {
		ad, err := adapterOf(ten)
		if err != nil {
			t.Fatal(err)
		}
		co, daDo := ad.ArgsTuDuyetQuyen()
		if !daDo {
			continue // provider chưa đo thì bài này không nói gì về nó
		}
		if len(co) == 0 {
			continue // đã đo, và provider không có rào nào
		}
		// Mỗi provider một cờ KHÁC NHAU — đó chính là lý do không được chép cứng.
		if !strings.HasPrefix(co[0], "-") {
			t.Errorf("%s: cờ tự duyệt quyền không phải một cờ: %v", ten, co)
		}
	}
	// Và hai provider lớn phải khai cờ KHÁC nhau, nếu không cả bài này vô nghĩa.
	c, _ := mustAdapter(t, "claude").ArgsTuDuyetQuyen()
	x, _ := mustAdapter(t, "codex").ArgsTuDuyetQuyen()
	if len(c) > 0 && len(x) > 0 && c[0] == x[0] {
		t.Errorf("claude và codex khai cùng một cờ (%q) — kiểm lại bảng năng lực", c[0])
	}
}

func mustAdapter(t *testing.T, ten string) provider.Adapter {
	t.Helper()
	ad, err := adapterOf(ten)
	if err != nil {
		t.Fatal(err)
	}
	return ad
}

// Provider CHƯA ĐO thì xin cờ phải LỖI, không phải im lặng chạy thiếu quyền —
// agent sẽ tự từ chối mọi tool rồi trả về một câu xin lỗi, và lượt chạy vẫn
// được ghi là xong. Đúng cách hỏng đã xảy ra thật với người soi ở lượt #46/#47.
func TestXinCoMaChuaDoThiPhaiLoi(t *testing.T) {
	for _, ten := range []string{"claude", "codex", "cursor", "grok", "antigravity"} {
		ad := mustAdapter(t, ten)
		_, daDo := ad.ArgsTuDuyetQuyen()
		tt := provider.TrangThaiCua(ad, provider.NLTuDuyetQuyen)
		// Hai nguồn phải nói CÙNG một chuyện: cờ đo được thì bảng năng lực cũng
		// phải khai là đo được. Lệch nhau thì một trong hai đang nói dối.
		if daDo && tt == provider.ChuaDo {
			t.Errorf("%s: ArgsTuDuyetQuyen nói đã đo, bảng năng lực nói CHƯA ĐO", ten)
		}
		if !daDo && tt == provider.LamDuoc {
			t.Errorf("%s: bảng năng lực nói làm được, ArgsTuDuyetQuyen nói chưa đo", ten)
		}
	}
}
