package flow

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// flowDocDuoc dựng flow dùng chung cho hai test chạy thật:
//
//	đợt 1: a, b chạy song song, mỗi bước để lại một kết quả nhận ra được
//	đợt 2: doc  — agent đọc kết quả CẢ HAI
//
// Dùng `notify` làm bước sinh kết quả vì nó trả về đúng message làm output, nên
// hai bước có hai kết quả KHÁC NHAU — fakeAgent thì trả cùng một chuỗi cho mọi
// lượt gọi, không phân biệt được ai là ai.
func flowDocDuoc(docDuoc []string) Flow {
	return Flow{Name: "quyen-doc", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "KET-QUA-CUA-A"},
		{ID: "b", Type: TypeNotify, Message: "KET-QUA-CUA-B"},
		{ID: "doc", Type: TypeAgent, Needs: []string{"a", "b"}, DocDuoc: docDuoc,
			Prompt: "A: {{steps.a.output}}\nB: {{steps.b.output}}"},
	}}
}

// (1) KHÔNG khai `doc_duoc` thì hành vi phải y NGUYÊN như trước: đọc được kết
// quả của mọi bước đã xong.
//
// Đây là bài giữ cửa cho quyết định "mặc định vẫn MỞ". Mọi flow đang chạy được
// đều dựa vào nó; đổi mặc định thành cấm hết là làm hỏng hết trong một lượt.
func TestKhongKhaiDocDuocThiDocNhuCu(t *testing.T) {
	r, ag, _ := newRunner(t)
	if _, err := r.Start(context.Background(), flowDocDuoc(nil), t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	ps := ag.cacPrompt()
	if len(ps) != 1 {
		t.Fatalf("phải gọi agent đúng 1 lần, được %d", len(ps))
	}
	for _, muon := range []string{"KET-QUA-CUA-A", "KET-QUA-CUA-B"} {
		if !strings.Contains(ps[0], muon) {
			t.Fatalf("bước KHÔNG khai doc_duoc phải đọc được mọi bước trước, thiếu %q trong prompt:\n%s", muon, ps[0])
		}
	}
	if strings.Contains(ps[0], "không được phép đọc") {
		t.Fatalf("bước không khai gì mà bị chặn — mặc định phải MỞ:\n%s", ps[0])
	}
}

// (2) Khai rồi thì bước NGOÀI danh sách bị chặn, và chỗ bị chặn phải mang ĐÚNG
// câu giải thích — không phải chuỗi rỗng.
//
// Vì sao câu chữ đáng kiểm từng chữ: cắt dữ liệu trong im lặng chính là lỗi lượt
// #29. Agent nhận một chỗ trống thì nó đoán; nhận một câu nói rõ "bị chặn, thêm
// b vào doc_duoc nếu cần" thì nó biết chuyện gì đang xảy ra, và người đọc log
// cũng biết cách sửa.
func TestKhaiDocDuocThiChanBuocNgoaiDanhSach(t *testing.T) {
	r, ag, _ := newRunner(t)
	if _, err := r.Start(context.Background(), flowDocDuoc([]string{"a"}), t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	ps := ag.cacPrompt()
	if len(ps) != 1 {
		t.Fatalf("phải gọi agent đúng 1 lần, được %d", len(ps))
	}
	got := ps[0]
	if !strings.Contains(got, "KET-QUA-CUA-A") {
		t.Fatalf("bước `a` NẰM TRONG doc_duoc mà kết quả không tới nơi:\n%s", got)
	}
	if strings.Contains(got, "KET-QUA-CUA-B") {
		t.Fatalf("bước `b` NGOÀI doc_duoc mà kết quả vẫn lọt vào prompt:\n%s", got)
	}
	muon := "(không được phép đọc kết quả bước \"b\" — thêm b vào doc_duoc nếu cần)"
	if !strings.Contains(got, muon) {
		t.Fatalf("chỗ bị chặn phải mang đúng câu giải thích.\nmuốn có: %s\nprompt:\n%s", muon, got)
	}
	// Chặn KHÔNG được biến thành "bước b không để lại kết quả": đổ lỗi cho bước
	// kia trong khi thủ phạm là chính lời khai doc_duoc.
	if strings.Contains(got, "bước \"b\" không để lại kết quả") {
		t.Fatalf("bị chặn mà lại báo thành `không để lại kết quả` — sai thủ phạm:\n%s", got)
	}
}

// (3) `doc_duoc` trỏ tới bước KHÔNG TỒN TẠI phải được cảnh báo. Gõ nhầm tên bước
// là lỗi im lặng hoàn hảo: có khai hay không thì kết quả vẫn y hệt nhau.
func TestDocDuocTroBuocKhongTonTaiThiCanhBao(t *testing.T) {
	f := Flow{Name: "sai-ten", Steps: []Step{
		{ID: "a", Type: TypeNotify, Message: "xong"},
		{ID: "b", Type: TypeNotify, Message: "đọc", Needs: []string{"a"}, DocDuoc: []string{"kiem-2"}},
	}}
	p := timVanDe(t, Validate(f), "doc_duoc")
	if !p.Warn {
		t.Fatalf("khai sai doc_duoc phải là CẢNH BÁO, không được chặn flow: %+v", p)
	}
	if p.Step != "b" {
		t.Fatalf("cảnh báo phải chỉ đúng bước khai sai, được %q", p.Step)
	}
	if !strings.Contains(p.Msg, "kiem-2") || !strings.Contains(p.Msg, "không tồn tại") {
		t.Fatalf("cảnh báo phải nêu tên bước sai và nói rõ là không tồn tại: %q", p.Msg)
	}
	for _, x := range Validate(f) {
		if !x.Warn {
			t.Fatalf("flow này ngoài doc_duoc sai ra thì hợp lệ, không được đẻ lỗi chặn: %+v", x)
		}
	}
}

// (4) `doc_duoc` trỏ tới bước chạy SAU cũng phải cảnh báo: lúc bước này chạy thì
// bước kia chưa có kết quả nào, nên lời khai chẳng mở ra được gì.
//
// Thứ tự đợt lấy từ Dot(). Ai gỡ phép so đợt đi thì test này đỏ — "bước có tồn
// tại" một mình không đủ để lời khai có nghĩa.
func TestDocDuocTroBuocChaySauThiCanhBao(t *testing.T) {
	f := Flow{Name: "nguoc-thu-tu", Steps: []Step{
		// truoc ở đợt 1, sau ở đợt 2 — `truoc` khai đọc `sau` là khai ngược.
		{ID: "truoc", Type: TypeNotify, Message: "xong", DocDuoc: []string{"sau"}},
		{ID: "sau", Type: TypeNotify, Message: "xong", Needs: []string{"truoc"}},
	}}
	p := timVanDe(t, Validate(f), "doc_duoc")
	if !p.Warn {
		t.Fatalf("khai ngược thứ tự phải là CẢNH BÁO: %+v", p)
	}
	if p.Step != "truoc" {
		t.Fatalf("cảnh báo phải chỉ đúng bước khai ngược, được %q", p.Step)
	}
	if !strings.Contains(p.Msg, "sau") || !strings.Contains(p.Msg, "SAU") {
		t.Fatalf("cảnh báo phải nói rõ bước kia chạy SAU: %q", p.Msg)
	}

	// Chiều ngược lại phải IM LẶNG: khai đúng thứ tự là chuyện bình thường, mà
	// cảnh báo nhầm thì người đọc học cách bỏ qua cả cột cảnh báo.
	ok := Flow{Name: "xuoi", Steps: []Step{
		{ID: "truoc", Type: TypeNotify, Message: "xong"},
		{ID: "sau", Type: TypeNotify, Message: "xong", Needs: []string{"truoc"}, DocDuoc: []string{"truoc"}},
	}}
	for _, x := range Validate(ok) {
		if strings.Contains(x.Msg, "doc_duoc") {
			t.Fatalf("khai đúng thứ tự mà vẫn bị kêu: %s", x.Msg)
		}
	}
}

// timVanDe lấy vấn đề đầu tiên có chứa `khoa`, hoặc dừng test nếu không có.
func timVanDe(t *testing.T, ps []Problem, khoa string) Problem {
	t.Helper()
	for _, p := range ps {
		if strings.Contains(p.Msg, khoa) {
			return p
		}
	}
	t.Fatalf("không có vấn đề nào nhắc %q — khai hỏng sẽ trôi qua im lặng. Được: %+v", khoa, ps)
	return Problem{}
}

// Không khai thì LocDocDuoc phải trả về NGUYÊN map cũ, không đụng một giá trị
// nào — đó là lời hứa "mặc định không đổi một byte".
func TestLocDocDuocKhongKhaiThiKhongDung(t *testing.T) {
	outs := map[string]string{"a": "AAA", "b": "BBB"}
	got := LocDocDuoc(Step{ID: "x"}, outs)
	if len(got) != 2 || got["a"] != "AAA" || got["b"] != "BBB" {
		t.Fatalf("không khai doc_duoc mà map bị đụng: %+v", got)
	}
	// Khai RỖNG khác hẳn KHÔNG KHAI: cấm hết, và cấm thì phải nói ra.
	het := LocDocDuoc(Step{ID: "x", DocDuoc: []string{}}, outs)
	if strings.Contains(het["a"], "AAA") {
		t.Fatalf("doc_duoc = [] nghĩa là cấm hết, mà `a` vẫn lọt: %q", het["a"])
	}
	if !strings.Contains(het["a"], "không được phép đọc") {
		t.Fatalf("cấm hết cũng phải NÓI RA, không được trả chuỗi rỗng: %q", het["a"])
	}
	if MoTaDocDuoc(Step{}) != "mọi bước trước" {
		t.Fatalf("chưa khai phải mô tả là `mọi bước trước`, được %q", MoTaDocDuoc(Step{}))
	}
	if MoTaDocDuoc(Step{DocDuoc: []string{"kiem-2"}}) != "kiem-2" {
		t.Fatalf("khai rồi thì mô tả là danh sách, được %q", MoTaDocDuoc(Step{DocDuoc: []string{"kiem-2"}}))
	}
}

// Người soi trong flow `doi-4` của chính dự án này phải bị cắt quyền đọc lời hai
// người thợ, chỉ còn kết quả máy chấm.
//
// Vì sao khoá cứng bằng test: soi mà đọc `code-go`/`code-doc` thì nó bị MỒI —
// nhắc lại lập luận của người làm thay vì tự đi tìm, và lời hứa "soi độc lập,
// khác hãng" chỉ còn là cái tên. Ai gỡ dòng `doc_duoc` đi thì phải gỡ cả bài
// này, tức là phải nói ra.
func TestDoi4CatQuyenDocCuaNguoiSoi(t *testing.T) {
	var file File
	if _, err := toml.DecodeFile(filepath.Join("..", "..", ".sagent", "flows.toml"), &file); err != nil {
		t.Fatal(err)
	}
	f, ok := file.Flows["doi-4"]
	if !ok {
		t.Skip("dự án không có flow doi-4")
	}
	f.Name = "doi-4"
	var soi *Step
	for i := range f.Steps {
		if f.Steps[i].ID == "soi" {
			soi = &f.Steps[i]
		}
	}
	if soi == nil {
		t.Skip("flow doi-4 không còn bước soi")
	}
	if len(soi.DocDuoc) != 1 || soi.DocDuoc[0] != "kiem-2" {
		t.Fatalf("bước `soi` chỉ được đọc kết quả máy chấm `kiem-2`, được %v", soi.DocDuoc)
	}
	// Và lời khai phải HỢP LỆ — khai một cái tên chết thì cũng như không khai.
	for _, p := range Validate(f) {
		if strings.Contains(p.Msg, "doc_duoc") {
			t.Fatalf("doi-4 khai doc_duoc hỏng: %s", p.Msg)
		}
	}
}
