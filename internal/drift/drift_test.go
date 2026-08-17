package drift

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func homeGia(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}
	t.Setenv("HOME", tmp)
	return tmp
}

// Lần đầu thì ghi mốc và KHÔNG báo động — chưa có gì để so.
func TestLanDauGhiMoc(t *testing.T) {
	homeGia(t)
	kq := Kiem("claude", "2.1.229", `C:\bin\claude.cmd`, false)
	if !kq.OK {
		t.Fatalf("lần đầu không được báo lỗi: %s", kq.Chi)
	}
	if !strings.Contains(kq.Chi, "2.1.229") {
		t.Errorf("không nhắc phiên bản vừa ghi: %s", kq.Chi)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("USERPROFILE"), ".ai-accounts", "provider-drift.json")); err != nil {
		if _, err2 := os.Stat(filepath.Join(os.Getenv("HOME"), ".ai-accounts", "provider-drift.json")); err2 != nil {
			t.Fatalf("không ghi ra file mốc: %v / %v", err, err2)
		}
	}
}

// Không đổi thì im.
func TestKhongDoiThiIm(t *testing.T) {
	homeGia(t)
	Kiem("claude", "2.1.229", "x", false)
	kq := Kiem("claude", "2.1.229", "x", false)
	if !kq.OK {
		t.Fatalf("phiên bản không đổi mà báo động: %s", kq.Chi)
	}
}

// Đây là toàn bộ lý do package này tồn tại: CLI đổi thì mọi phép đo gắn với bản
// cũ hết hiệu lực, và người dùng PHẢI biết.
func TestDoiPhienBanThiBaoDong(t *testing.T) {
	homeGia(t)
	Kiem("codex", "codex-cli 0.147.0", "x", false)

	kq := Kiem("codex", "codex-cli 0.200.0", "x", false)
	if kq.OK {
		t.Fatal("CLI đổi phiên bản mà không báo động — số đo cũ thành lời đồn trong im lặng")
	}
	for _, phai := range []string{"0.147.0", "0.200.0", "DO-LUONG", "--chap-nhan"} {
		if !strings.Contains(kq.Chi, phai) {
			t.Errorf("thông điệp thiếu %q — người dùng phải biết đổi từ đâu sang đâu và làm gì tiếp: %s", phai, kq.Chi)
		}
	}
}

// Cảnh báo phải CÒN HIỆN ở những lần chạy sau. Tự cập nhật mốc thì nó hiện đúng
// một lần rồi biến mất, mà thứ đã trôi thì vẫn trôi — đúng kiểu hỏng im lặng mà
// cả dự án này lập ra để chống.
func TestCanhBaoKhongTuBienMat(t *testing.T) {
	homeGia(t)
	Kiem("codex", "0.147.0", "x", false)
	for i := 0; i < 3; i++ {
		if kq := Kiem("codex", "0.200.0", "x", false); kq.OK {
			t.Fatalf("lần %d: cảnh báo đã tự tắt dù chưa ai chấp nhận mốc mới", i+1)
		}
	}
}

// Chấp nhận rồi thì im — và im ở cả những lần sau.
func TestChapNhanThiGhiMocMoi(t *testing.T) {
	homeGia(t)
	Kiem("codex", "0.147.0", "x", false)
	kq := Kiem("codex", "0.200.0", "x", true)
	if !kq.OK {
		t.Fatalf("chấp nhận mà vẫn báo lỗi: %s", kq.Chi)
	}
	if !strings.Contains(kq.Chi, "0.147.0") {
		t.Errorf("nên nhắc mốc cũ để còn đối chiếu: %s", kq.Chi)
	}
	if kq2 := Kiem("codex", "0.200.0", "x", false); !kq2.OK {
		t.Fatalf("sau khi chấp nhận vẫn báo động: %s", kq2.Chi)
	}
}

// Mốc của provider này không được đè lên provider kia.
func TestMoiProviderMotMoc(t *testing.T) {
	homeGia(t)
	Kiem("claude", "2.1.229", "x", false)
	Kiem("codex", "0.147.0", "x", false)
	if kq := Kiem("claude", "2.1.229", "x", false); !kq.OK {
		t.Fatalf("mốc claude bị codex đè: %s", kq.Chi)
	}
	if kq := Kiem("codex", "0.147.0", "x", false); !kq.OK {
		t.Fatalf("mốc codex bị claude đè: %s", kq.Chi)
	}
}

// Sổ mốc CÓ nhưng hỏng thì phải BÁO, tuyệt đối không im lặng dựng lại.
//
// Đã nổ thật khi chạy tay trên máy: `Set-Content -Encoding UTF8` của PowerShell
// 5.1 thêm BOM vào file, json.Unmarshal hỏng, bản cũ nuốt lỗi và trả sổ rỗng —
// `verify` báo "ghi mốc đầu tiên" rồi GHI ĐÈ, xoá sạch mốc của mọi provider.
// Không một dòng cảnh báo. Và lần ghi đè đó xoá luôn BOM, nên bằng chứng của
// lỗi cũng biến mất: kiểm lại file sau khi chạy thì thấy nó "bình thường".
func TestSoMocHongThiBaoChuKhongImLangDungLai(t *testing.T) {
	home := homeGia(t)
	Kiem("codex", "0.147.0", "x", false) // dựng mốc thật

	f := filepath.Join(home, ".ai-accounts", "provider-drift.json")
	goc, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, []byte("{ khong phai json"), 0o600); err != nil {
		t.Fatal(err)
	}

	kq := Kiem("codex", "0.147.0", "x", false)
	if kq.OK {
		t.Fatal("sổ hỏng mà báo OK — mốc sẽ bị ghi đè trong im lặng")
	}
	if !strings.Contains(kq.Chi, "hỏng") {
		t.Errorf("thông điệp không nói file hỏng: %s", kq.Chi)
	}
	// Và quan trọng nhất: KHÔNG được ghi đè.
	sau, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(sau) != "{ khong phai json" {
		t.Fatalf("đã ghi đè sổ hỏng — mốc cũ mất sạch. Nội dung giờ là:\n%s\n(trước khi hỏng: %s)",
			sau, goc)
	}
}

// BOM là chuyện bình thường trên Windows (trình soạn thảo, Set-Content, Notepad).
// Đọc được thì đọc, đừng bắt người dùng trả giá cho thói quen của công cụ khác.
func TestChiuDuocBOM(t *testing.T) {
	home := homeGia(t)
	Kiem("codex", "0.147.0", "x", false)

	f := filepath.Join(home, ".ai-accounts", "provider-drift.json")
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, append([]byte{0xEF, 0xBB, 0xBF}, b...), 0o600); err != nil {
		t.Fatal(err)
	}

	// Vẫn phải nhận ra đây là mốc cũ, và vẫn phải báo động khi phiên bản đổi.
	if kq := Kiem("codex", "0.147.0", "x", false); !kq.OK {
		t.Fatalf("BOM làm hỏng việc đọc sổ: %s", kq.Chi)
	}
	kq := Kiem("codex", "0.200.0", "x", false)
	if kq.OK {
		t.Fatal("file có BOM -> mốc bị coi như rỗng -> drift lọt lưới trong im lặng")
	}
}
