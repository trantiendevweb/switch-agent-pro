package api

import (
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// flowKho là flow dùng chung cho các test dưới: một đợt hai bước song song, rồi
// một rào duyệt, rồi một bước đọc kết quả bước trước.
func flowKho() flow.Flow {
	return flow.Flow{
		Name: "kho", Desc: "flow thử",
		Vars: map[string]string{"viec": "sửa lỗi X"},
		Steps: []flow.Step{
			{ID: "code", Type: flow.TypeAgent, Profile: "claude:tns", Prompt: "làm: {{viec}}",
				Copies: 3, Worktree: true},
			{ID: "kiem", Type: flow.TypeShell, Run: []string{"go", "version"}},
			{ID: "duyet", Type: flow.TypeApprove, Message: "xem rồi duyệt", Needs: []string{"code", "kiem"}},
			{ID: "bao", Type: flow.TypeNotify, Message: "kết quả: {{steps.code.output}}", Needs: []string{"duyet"}},
		},
	}
}

// Kế hoạch phải nói đủ: đợt nào, bước nào song song, tài khoản mỗi bước, và
// prompt ĐÃ THAY BIẾN (thứ agent thật sự nhận, không phải mẫu thô).
func TestChayKhoTraKeHoachDayDu(t *testing.T) {
	khoTam(t)
	dungHoSoClaude(t, "tns", 2*time.Hour)
	dir := t.TempDir()
	if _, err := flow.Save(dir, flowKho()); err != nil {
		t.Fatal(err)
	}

	a := &API{} // KHÔNG có db: chạy khan mà chạm sổ thì test này nổ ngay
	kh, err := a.FlowChayKho(dir, "kho", nil, Addr{})
	if err != nil {
		t.Fatal(err)
	}

	if len(kh.Dot) != 3 {
		t.Fatalf("phải chia 3 đợt (code+kiem · duyệt · bao), được %d: %+v", len(kh.Dot), kh.Dot)
	}
	if len(kh.Dot[0].Buoc) != 2 {
		t.Fatalf("đợt 1 phải có 2 bước chạy song song, được %d", len(kh.Dot[0].Buoc))
	}
	if !kh.Dot[1].ChoDuyet {
		t.Fatal("đợt 2 phải được đánh dấu là rào duyệt")
	}

	code := kh.Dot[0].Buoc[0]
	if code.ID != "code" {
		t.Fatalf("bước đầu đợt 1 phải là code, được %q", code.ID)
	}
	if code.TaiKhoan != "claude:tns" {
		t.Fatalf("phải nói bước chạy bằng tài khoản nào, được %q", code.TaiKhoan)
	}
	if code.SoAgent != 3 || !code.Worktree {
		t.Fatalf("phải nói 3 agent + worktree riêng, được %d agent worktree=%v", code.SoAgent, code.Worktree)
	}
	// Prompt đã thay biến: {{viec}} phải biến mất, giá trị thật phải có mặt.
	if strings.Contains(code.Prompt, "{{viec}}") || !strings.Contains(code.Prompt, "sửa lỗi X") {
		t.Fatalf("prompt chưa thay biến: %q", code.Prompt)
	}
	if kh.SoAgent != 3 {
		t.Fatalf("tổng số agent phải là 3, được %d", kh.SoAgent)
	}
}

// Bước `bao` đọc {{steps.code.output}} và `code` chạy ở đợt TRƯỚC — hợp lệ, nên
// không được báo hụt. Đây là chiều dễ báo nhầm nhất của phép kiểm này.
func TestChayKhoKhongBaoNhamKhiKetQuaCoThat(t *testing.T) {
	khoTam(t)
	dir := t.TempDir()
	if _, err := flow.Save(dir, flowKho()); err != nil {
		t.Fatal(err)
	}
	kh, err := (&API{}).FlowChayKho(dir, "kho", nil, Addr{})
	if err != nil {
		t.Fatal(err)
	}
	bao := kh.Dot[2].Buoc[0]
	if bao.ID != "bao" {
		t.Fatalf("đợt 3 phải là bước bao, được %q", bao.ID)
	}
	if bao.ConSot != "" {
		t.Fatalf("bước `code` chạy ở đợt trước nên kết quả CÓ THẬT, không được báo hụt %q", bao.ConSot)
	}
	if !strings.Contains(bao.Prompt, "code") {
		t.Fatalf("chỗ {{steps.code.output}} phải được nói rõ là kết quả bước code: %q", bao.Prompt)
	}
}

// CA DA NO THAT (lượt chạy #29): prompt trỏ tới kết quả của một bước KHÔNG chạy
// xong trước nó, nên agent nhận nguyên chữ sống `{{steps.x.output}}` làm đề bài.
// Chạy khan phải chỉ đúng chỗ đó ra TRƯỚC khi tốn một đồng nào.
func TestChayKhoBatPlaceholderKhongCoKetQuaTruoc(t *testing.T) {
	khoTam(t)
	dir := t.TempDir()
	f := flow.Flow{Name: "hut", Steps: []flow.Step{
		// Hai bước CÙNG một đợt: chúng chạy song song, nên `sau` không thể đọc
		// được kết quả của `truoc`.
		{ID: "truoc", Type: flow.TypeNotify, Message: "xong"},
		{ID: "sau", Type: flow.TypeNotify, Message: "đọc: {{steps.truoc.output}}"},
	}}
	if _, err := flow.Save(dir, f); err != nil {
		t.Fatal(err)
	}
	kh, err := (&API{}).FlowChayKho(dir, "hut", nil, Addr{})
	if err != nil {
		t.Fatal(err)
	}
	var thay string
	for _, d := range kh.Dot {
		for _, b := range d.Buoc {
			if b.ID == "sau" {
				thay = b.ConSot
			}
		}
	}
	if thay != "truoc" {
		t.Fatalf("phải báo bước `sau` đang chờ kết quả của `truoc` (chạy cùng đợt), được %q", thay)
	}
}

// Cổng kiểm tài khoản phải nói ở đây — đó chính là câu hỏi mà ba lượt chạy thật
// ngày 19/08 (#30, #32, #33) đã phải đốt hạn mức để hỏi.
func TestChayKhoNoiTaiKhoanHongMaKhongChan(t *testing.T) {
	khoTam(t)
	dungHoSoClaude(t, "tns", -2*time.Hour) // token hết hạn 2 tiếng trước
	dir := t.TempDir()
	if _, err := flow.Save(dir, flowKho()); err != nil {
		t.Fatal(err)
	}
	kh, err := (&API{}).FlowChayKho(dir, "kho", nil, Addr{})
	if err != nil {
		t.Fatalf("chạy khan KHÔNG được chặn vì tài khoản hỏng — nó sinh ra để nói chuyện đó: %v", err)
	}
	if len(kh.TaiKhoanHong) != 1 || kh.TaiKhoanHong[0].Addr != "claude:tns" {
		t.Fatalf("phải nói claude:tns dùng không được, được %+v", kh.TaiKhoanHong)
	}
	if !strings.Contains(kh.TaiKhoanHong[0].LyDo, "hết hạn") {
		t.Fatalf("phải nói rõ là hết hạn, được %q", kh.TaiKhoanHong[0].LyDo)
	}
	// Kế hoạch vẫn phải có mặt: người ta cần thấy cả hai để quyết định.
	if len(kh.Dot) == 0 {
		t.Fatal("tài khoản hỏng thì vẫn phải trả kế hoạch, không được trả rỗng")
	}
}

// ĐÂY LÀ LỜI HỨA CHÍNH của tính năng: chạy khan KHÔNG ghi lượt chạy nào vào sổ
// và KHÔNG bật agent nào.
//
// Kiểm bằng thứ quan sát được, không bằng lời hứa trong bình luận:
//   - số lượt chạy trong sổ TRƯỚC và SAU phải y nguyên;
//   - không phiên agent nào được mở.
//
// Gỡ chạy khan ra và gọi FlowRun thay vào là test này đỏ ngay: sổ sẽ có 1 lượt.
func TestChayKhoKhongGhiSoKhongGoiAgent(t *testing.T) {
	khoTam(t)
	dungHoSoClaude(t, "tns", 2*time.Hour)
	dir := t.TempDir()

	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if _, err := a.FlowSave(dir, flowKho()); err != nil {
		t.Fatal(err)
	}

	truoc, err := a.FlowRuns(100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.FlowChayKho(dir, "kho", nil, ParseAddr("claude:tns")); err != nil {
		t.Fatal(err)
	}
	sau, err := a.FlowRuns(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(sau) != len(truoc) {
		t.Fatalf("chạy khan đã GHI %d lượt chạy vào sổ — nó phải không để lại dấu vết nào",
			len(sau)-len(truoc))
	}
	phien, err := a.SessionList()
	if err != nil {
		t.Fatal(err)
	}
	if len(phien) != 0 {
		t.Fatalf("chạy khan đã bật %d phiên agent — nó phải không gọi agent nào", len(phien))
	}
}

// Chạy khan phải nói QUYỀN ĐỌC của từng bước — bước nào đọc được bước nào.
//
// Đây là chỗ duy nhất thấy được điều đó trước khi tiêu tiền: `doc_duoc` quyết
// định bước sau nuốt bao nhiêu output của bước trước, tức quyết định thẳng số
// token vào. Lượt #34, bước gộp nhận 10.998 token vào phần lớn là output bước
// khác — không có dòng này thì phải chạy thật mới biết.
//
// Và prompt in ra phải ĐI QUA đúng bộ lọc mà bộ thực thi dùng: chạy khan mà bỏ
// qua bộ lọc thì nó in một prompt khác prompt sẽ gửi đi, đúng thứ tính năng
// chạy khan sinh ra để chống.
func TestChayKhoNoiQuyenDocTungBuoc(t *testing.T) {
	khoTam(t)
	dir := t.TempDir()
	f := flow.Flow{Name: "quyen", Steps: []flow.Step{
		{ID: "a", Type: flow.TypeNotify, Message: "KET-QUA-CUA-A"},
		{ID: "b", Type: flow.TypeNotify, Message: "KET-QUA-CUA-B"},
		{ID: "mo", Type: flow.TypeNotify, Needs: []string{"a", "b"},
			Message: "A={{steps.a.output}} B={{steps.b.output}}"},
		{ID: "hep", Type: flow.TypeNotify, Needs: []string{"a", "b"}, DocDuoc: []string{"a"},
			Message: "A={{steps.a.output}} B={{steps.b.output}}"},
	}}
	if _, err := flow.Save(dir, f); err != nil {
		t.Fatal(err)
	}
	kh, err := (&API{}).FlowChayKho(dir, "quyen", nil, Addr{})
	if err != nil {
		t.Fatal(err)
	}
	buoc := map[string]BuocKho{}
	for _, d := range kh.Dot {
		for _, b := range d.Buoc {
			buoc[b.ID] = b
		}
	}
	if buoc["mo"].DocDuoc != "mọi bước trước" {
		t.Fatalf("bước chưa khai doc_duoc phải hiện là `mọi bước trước`, được %q", buoc["mo"].DocDuoc)
	}
	if buoc["hep"].DocDuoc != "a" {
		t.Fatalf("bước khai doc_duoc = [\"a\"] phải hiện đúng danh sách, được %q", buoc["hep"].DocDuoc)
	}
	// Prompt của bước bị siết phải nói ra chỗ bị chặn, và KHÔNG được vờ như
	// bước `b` không để lại kết quả — nó có kết quả, chỉ là không được đọc.
	if !strings.Contains(buoc["hep"].Prompt, "không được phép đọc kết quả bước \"b\"") {
		t.Fatalf("chạy khan phải in ra chỗ bị chặn y như lúc chạy thật: %q", buoc["hep"].Prompt)
	}
	if strings.Contains(buoc["hep"].Prompt, "{{steps.b.output}}") {
		t.Fatalf("chỗ bị chặn còn nguyên chữ sống trong kế hoạch: %q", buoc["hep"].Prompt)
	}
	// Bước không khai gì thì kế hoạch phải y như trước — không lây cái chặn.
	if strings.Contains(buoc["mo"].Prompt, "không được phép đọc") {
		t.Fatalf("bước không khai doc_duoc mà bị chặn trong kế hoạch: %q", buoc["mo"].Prompt)
	}
	if !strings.Contains(buoc["mo"].Prompt, "kết quả bước \"b\"") {
		t.Fatalf("bước không khai doc_duoc phải đọc được `b` như cũ: %q", buoc["mo"].Prompt)
	}
}

// Chạy khan một flow không có thì phải nói thẳng, đừng trả kế hoạch rỗng trông
// như "flow này chẳng làm gì cả".
func TestChayKhoBaoFlowKhongCo(t *testing.T) {
	khoTam(t)
	if _, err := (&API{}).FlowChayKho(t.TempDir(), "khong-co-dau", nil, Addr{}); err == nil {
		t.Fatal("flow không tồn tại mà vẫn trả kế hoạch")
	}
}
