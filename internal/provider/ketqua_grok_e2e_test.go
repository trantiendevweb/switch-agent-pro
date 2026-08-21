package provider

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// BÀI CANH ĐỊNH KỲ cho bộ đọc kết quả của Grok — gọi CLI THẬT, tốn hạn mức
// thật, nên MẶC ĐỊNH BỎ QUA. Bật bằng: SAGENT_E2E_GROK=1 go test ./internal/provider/
//
// VÌ SAO TỒN TẠI: ô **C2** trong docs/SO-NO-DO-LUONG.md ghi đúng cái nợ này —
// `docKetQuaGrok` đọc một định dạng ta QUAN SÁT được ở lần chạy #29, không phải
// một hợp đồng Grok cam kết (Grok không có cờ nào bảo nó xuất JSON). Grok đổi
// cách in MỘT LẦN là `docDuoc=false`, và khi đó:
//   - mọi phiên Grok lặng lẽ tụt về `lost`, mà bảng năng lực vẫn khoe xanh;
//   - mất luôn lá chắn chống chạy quẩn cho đúng provider khai
//     `Khong(NLTuDuyetQuyen)` — provider chạy tool tự do nhất.
//
// Chính ô C2 kê biện pháp: *"chỉ giảm được bằng cách canh: một bài kiểm định kỳ
// chạy `grok -p` thật rồi khẳng định `docDuoc==true`"*. File này là bài đó.
//
// Nó cũng nối được hai đầu như bài canh của Codex: args do CHÍNH adapter dựng
// (`HeadlessArgs` + `ModelArgs` + `ArgsThuMuc`) chạy ra output mà CHÍNH adapter
// đọc lại được. Chép cứng dòng lệnh vào đây thì mất đúng nửa đó.
//
// KẾT QUẢ LẦN CHẠY 21/08/2026 (grok-cli 1.0.1, endpoint modelapi.vn, model
// grok-4.5), NGUYÊN VĂN — hai dòng stdout, mã thoát 0:
//
//	{"role":"user","content":"Tra loi dung mot tu: ALPHA"}
//	{"role":"assistant","content":"Sorry, I encountered an error: Grok API error:
//	 410 Live search is deprecated. Please switch to the Agent Tools API: …"}
//
// Đọc ra: docDuoc=true, TraLoi=chính câu lỗi đó, PhanLoaiChet="done".
//
// HAI KẾT LUẬN, ĐỪNG GỘP LÀM MỘT:
//
//  1. Định dạng NDJSON VẪN CÒN — `docKetQuaGrok` đọc được đúng như trước. Nên
//     `Duoc(NLKetQuaCoCauTruc)` ở grok.go vẫn KHỚP SỰ THẬT, không hạ. Hạ xuống
//     `Khong`/`Chua` lúc này là khai đỏ cho một thứ vừa đo được là xanh — sai
//     theo đúng chiều ngược lại.
//  2. CLI thì HỎNG THẬT: HTTP 410 "Live search is deprecated", đúng cái đã đo
//     20/08 và đã khiến bước `soi` của `doi-4` phải đổi sang node `model`
//     (.sagent/flows.toml:862, docs/DO-LUONG.md, internal/flow/modelnode_test.go).
//     Grok in lỗi API RA NHƯ MỘT CÂU TRẢ LỜI BÌNH THƯỜNG, `docKetQuaGrok` để
//     `CoLoi=false`, nên `PhanLoaiChet` xếp lượt này là "done".
//
// Nên bài này KHÔNG che chuyện (2) để mình luôn xanh. Nó đỏ, và sẽ còn đỏ tới
// ngày nhà cung cấp trả lại được câu trả lời thật. Một bài canh xanh trong lúc
// CLI hỏng vĩnh viễn thì chính nó là thứ nợ mà C2 nói tới.
func TestE2EGrokDocDuocOutputThat(t *testing.T) {
	if os.Getenv("SAGENT_E2E_GROK") != "1" {
		t.Skip("bỏ qua: gọi CLI thật và tốn hạn mức — bật bằng SAGENT_E2E_GROK=1")
	}
	ad := grok{}
	bin, err := ad.Command()
	if err != nil {
		t.Skipf("máy này không có grok: %v", err)
	}
	// Model phải truyền TƯỜNG MINH: `grok -p` bỏ qua defaultModel trong chính
	// user-settings.json của nó và dùng grok-code-fast-1, model mà endpoint
	// không bán (503 "No available channel"). Lấy tên model từ cấu hình thật
	// của máy chứ không chép cứng — mỗi hồ sơ trỏ một endpoint bán model khác.
	home, _ := os.UserHomeDir()
	cfg, ok := docGrok(home)
	if !ok || cfg.DefaultModel == "" {
		t.Skipf("máy này chưa cấu hình grok: %s", grokSettings(home))
	}

	// Args dựng bằng CHÍNH adapter, cả ba mảnh.
	args := append(ad.HeadlessArgs("Tra loi dung mot tu: ALPHA"), ad.ModelArgs(cfg.DefaultModel)...)
	args = append(args, ad.ArgsThuMuc(os.TempDir())...)
	t.Logf("LỆNH: %s %v", bin, args)

	c := exec.Command(bin, args...)
	// Stdin nối vào NUL, không để nil — cùng lý do đã đo ở bài canh của Codex:
	// CLI thấy stdin là ống dẫn thì có thể ngồi chờ đầu vào và treo vô hạn.
	if devNull, err := os.Open(os.DevNull); err == nil {
		defer devNull.Close()
		c.Stdin = devNull
	}
	var loiChay error
	out, err := c.Output()
	if err != nil {
		// KHÔNG dừng ở đây: grok đã có lần thoát khác 0 mà vẫn in đủ bản ghi.
		// Ghi lại rồi vẫn thử đọc — đọc được hay không mới là câu hỏi của ô C2.
		loiChay = err
		t.Logf("LỖI CHẠY: %v", err)
		if ee, la := err.(*exec.ExitError); la {
			t.Logf("STDERR THẬT:\n%s", ee.Stderr)
		}
	}
	t.Logf("STDOUT THẬT:\n%s", out)

	k, docDuoc := ad.DocKetQua(string(out))
	if !docDuoc {
		t.Fatalf("Grok ĐÃ ĐỔI ĐỊNH DẠNG: adapter không đọc nổi output thật nữa — "+
			"ô C2 vừa nổ, mọi phiên Grok từ giờ tụt về `lost` và mất lá chắn chạy quẩn.\n"+
			"lỗi chạy: %v\noutput:\n%s", loiChay, out)
	}
	tt, ly, _ := PhanLoaiChet(k, docDuoc)
	t.Logf("ĐỌC ĐƯỢC: TraLoi=%q DemDuocTool=%v SoLoiGoiTool=%d LenhLap=%q SoLanLap=%d",
		k.TraLoi, k.DemDuocTool, k.SoLoiGoiTool, k.LenhLap, k.SoLanLap)
	t.Logf("PhanLoaiChet: %q %q", tt, ly)

	if k.TraLoi == "" {
		t.Errorf("đọc được bản ghi nhưng không lấy ra câu trả lời nào — " +
			"lời assistant cuối cùng không kèm tool_calls đã biến mất khỏi output")
	}
	// CLI có trả lời được thật không? Grok in lỗi API ra Y NHƯ một câu trả lời,
	// nên chỗ duy nhất lỗi còn hiện ra là NỘI DUNG câu trả lời. Đây là dò chuỗi,
	// và nó cố ý: không có trường nào để dò cả — chính đó là cái phải ghi ra.
	if strings.Contains(k.TraLoi, "Grok API error") || strings.Contains(k.TraLoi, "I encountered an error") {
		t.Errorf("CLI grok KHÔNG trả lời được — nó in lỗi API ra như một câu trả lời "+
			"bình thường, `CoLoi` vẫn là false nên PhanLoaiChet xếp lượt này là %q.\n"+
			"Nguyên văn: %s\n"+
			"Đã đo 20/08 và lại 21/08: HTTP 410 \"Live search is deprecated\". "+
			"Bước `soi` của doi-4 đã phải đổi sang node `model` vì đúng chuyện này "+
			"(.sagent/flows.toml:862). Bài canh này ĐỎ là đúng — đừng làm nó xanh "+
			"bằng cách bỏ khẳng định đi.", tt, k.TraLoi)
	}
}
