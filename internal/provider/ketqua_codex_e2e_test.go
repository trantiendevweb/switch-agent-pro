package provider

import (
	"os"
	"os/exec"
	"testing"
)

// BÀI CANH ĐỊNH KỲ cho bộ đọc kết quả của Codex — gọi CLI THẬT, tốn hạn mức
// thật, nên MẶC ĐỊNH BỎ QUA. Bật bằng: SAGENT_E2E_CODEX=1 go test ./internal/provider/
//
// VÌ SAO TỒN TẠI: mọi bài kiểm khác trong ketqua_codex_test.go chạy trên bản ghi
// ĐÃ CHÉP LẠI. Chúng chứng minh bộ đọc đọc đúng cái nó đã thấy — chúng KHÔNG
// chứng minh được Codex hôm nay vẫn in ra thứ đó. Sổ nợ đo lường đã ghi đúng rủi
// ro này ở ô C2 cho Grok: nhà cung cấp đổi cách in MỘT LẦN là docDuoc về false,
// mọi phiên lặng lẽ tụt về `lost`, mà bảng năng lực vẫn khoe xanh. Bài này để
// biết vào NGÀY nó đổi, không phải ba tuần sau.
//
// Nó cũng là bài DUY NHẤT nối được hai đầu: args do chính adapter dựng
// (`HeadlessArgs` + `ArgsThuMuc`) chạy ra output mà chính adapter đọc lại được.
// Chép cứng dòng lệnh vào đây thì mất đúng nửa đó.
//
// Kết quả lần chạy 21/08/2026 (codex-cli 0.147.0), nguyên văn:
//
//	LỆNH: …\npm\codex.cmd [exec --json Tra loi dung mot tu: ALPHA --cd …\Temp --skip-git-repo-check]
//	ĐỌC ĐƯỢC: TraLoi="ALPHA" CoLoi=false TokenVao=17627 TokenRa=6 Hong=""
//	PhanLoaiChet: "done" ""
//
// Dòng cuối là cái đáng nhìn nhất: trước lượt đo này nó là "" — tức phiên ở lại
// `lost`, và bốn mặt điều khiển in "chết, chưa rõ vì sao" cho một lượt chạy
// thành công.
func TestE2ECodexDocDuocOutputThat(t *testing.T) {
	if os.Getenv("SAGENT_E2E_CODEX") != "1" {
		t.Skip("bỏ qua: gọi CLI thật và tốn hạn mức — bật bằng SAGENT_E2E_CODEX=1")
	}
	ad := codex{}
	bin, err := ad.Command()
	if err != nil {
		t.Skipf("máy này không có codex: %v", err)
	}
	// Args dựng bằng CHÍNH adapter. `--skip-git-repo-check` thêm tay vì thư mục
	// tạm không phải repo git; nó không nằm trong hợp đồng của adapter.
	args := append(ad.HeadlessArgs("Tra loi dung mot tu: ALPHA"), ad.ArgsThuMuc(os.TempDir())...)
	args = append(args, "--skip-git-repo-check")
	t.Logf("LỆNH: %s %v", bin, args)

	c := exec.Command(bin, args...)
	// Stdin phải nối vào NUL, KHÔNG để nil. Đo được 21/08: thấy stdin là ống dẫn
	// thì codex in "Reading additional input from stdin..." rồi TREO vô hạn —
	// lượt đo đầu tiên chết đúng vì chuyện này. profile/clone.go:349 đã biết.
	if devNull, err := os.Open(os.DevNull); err == nil {
		defer devNull.Close()
		c.Stdin = devNull
	}
	out, err := c.Output()
	if err != nil {
		t.Fatalf("chạy codex hỏng: %v\n%s", err, out)
	}
	t.Logf("STDOUT THẬT:\n%s", out)

	k, ok := ad.DocKetQua(string(out))
	if !ok {
		t.Fatalf("Codex ĐÃ ĐỔI ĐỊNH DẠNG: adapter không đọc nổi output thật nữa.\n%s", out)
	}
	if k.TraLoi == "" {
		t.Errorf("đọc được nhưng không lấy ra câu trả lời nào")
	}
	if k.TokenVao == 0 || k.TokenRa == 0 {
		t.Errorf("usage về 0 — trường token có thể vừa đổi tên: vào %d, ra %d", k.TokenVao, k.TokenRa)
	}
	// Lượt chạy trơn tru phải ra `done`, không phải ở lại `lost`.
	if tt, ly, _ := PhanLoaiChet(k, ok); tt != Xong {
		t.Errorf("lượt thành công bị xếp %q (%s), phải là %q", tt, ly, Xong)
	}
}
