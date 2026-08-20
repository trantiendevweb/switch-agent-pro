package paths

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// datHome trỏ thư mục người dùng vào một chỗ tạm. Phải đặt ĐÚNG biến của từng
// nền tảng: os.UserHomeDir đọc %USERPROFILE% trên Windows và $HOME ở nơi khác —
// đặt nhầm biến thì test trông như chạy mà thật ra vẫn đo máy thật.
func datHome(t *testing.T, home string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}
}

// Phép đo cốt lõi của package: cả ba hàm phải ĐỌC biến môi trường tại lúc gọi,
// không phải chụp một lần lúc nạp package.
//
// Vì sao đáng đo: nếu giá trị bị nhớ trong biến toàn cục lúc init, thì mọi test
// dùng t.Setenv("USERPROFILE", tmp) trong toàn dự án — internal/profile,
// internal/config, internal/dash — sẽ âm thầm đo trên HOME THẬT và có thể xoá
// dữ liệu thật, vì công cụ này xoá thư mục. Test này chốt tại gốc.
func TestBaHamDoiTheoBienMoiTruongChuKhongNhoMotLan(t *testing.T) {
	homeA := t.TempDir()
	datHome(t, homeA)

	if got := Home(); got != homeA {
		t.Fatalf("Home() = %q, chờ %q", got, homeA)
	}
	if got, want := AccountsRoot(), filepath.Join(homeA, ".ai-accounts"); got != want {
		t.Fatalf("AccountsRoot() = %q, chờ %q", got, want)
	}
	if got, want := LegacyClaudeAccounts(), filepath.Join(homeA, ".claude-accounts"); got != want {
		t.Fatalf("LegacyClaudeAccounts() = %q, chờ %q", got, want)
	}

	// Đổi HOME lần thứ hai TRONG CÙNG một tiến trình: đây mới là chỗ bẫy "nhớ
	// một lần" lộ ra. Lần đầu luôn đúng dù có cache hay không.
	homeB := t.TempDir()
	datHome(t, homeB)

	if got := Home(); got != homeB {
		t.Fatalf("Home() còn dính giá trị cũ %q sau khi đổi biến sang %q — "+
			"giá trị đang bị nhớ lúc init, mọi test dùng t.Setenv sẽ đo nhầm máy thật", got, homeB)
	}
	if got, want := AccountsRoot(), filepath.Join(homeB, ".ai-accounts"); got != want {
		t.Fatalf("AccountsRoot() = %q, chờ %q", got, want)
	}
	if got, want := LegacyClaudeAccounts(), filepath.Join(homeB, ".claude-accounts"); got != want {
		t.Fatalf("LegacyClaudeAccounts() = %q, chờ %q", got, want)
	}
}

// Hai kho phải là HAI chỗ khác nhau, và cùng nằm dưới HOME.
//
// Cả tính năng di trú (v1 -> v2, xem internal/profile) dựa trên đúng điều này:
// đọc từ ~/.claude-accounts rồi ghi sang ~/.ai-accounts. Nếu hai hàm trả cùng
// một đường dẫn thì di trú thành "chép đè lên chính nó" — mất dữ liệu v1 mà
// không lỗi nào nổi lên.
func TestHaiKhoLaHaiChoKhacNhauVaDeuNamDuoiHome(t *testing.T) {
	home := t.TempDir()
	datHome(t, home)

	moi, cu := AccountsRoot(), LegacyClaudeAccounts()
	if moi == cu {
		t.Fatalf("kho mới và kho v1 trỏ cùng một chỗ (%q) — di trú sẽ ghi đè lên chính nó", moi)
	}
	for ten, p := range map[string]string{"AccountsRoot": moi, "LegacyClaudeAccounts": cu} {
		if !strings.HasPrefix(p, home) {
			t.Fatalf("%s() = %q, nằm ngoài HOME %q", ten, p, home)
		}
		// Đúng một tầng dưới HOME: nếu thành hai tầng thì các hàm ghép tiếp
		// (ClonesRoot, KeysDir, config.GlobalPath...) đều lệch theo.
		if filepath.Dir(p) != home {
			t.Fatalf("%s() = %q, không nằm ngay dưới HOME %q", ten, p, home)
		}
	}
	if filepath.Base(moi) != ".ai-accounts" {
		t.Fatalf("tên kho mới đổi thành %q — đổi tên kho là đổi chỗ dữ liệu của người dùng", filepath.Base(moi))
	}
	if filepath.Base(cu) != ".claude-accounts" {
		t.Fatalf("tên kho v1 đổi thành %q — di trú sẽ không tìm thấy dữ liệu cũ nữa", filepath.Base(cu))
	}
}

// HOME có DẤU CÁCH và dấu tiếng Việt. Không phải ca hiếm: %USERPROFILE% mặc
// định lấy theo tên tài khoản Windows, nên C:\Users\Nguyen Van A là bình thường.
// Đường dẫn ghép ra phải giữ nguyên tên đó, không bị cắt ở dấu cách.
func TestHomeCoDauCachVaDauTiengVietThiGhepVanDung(t *testing.T) {
	home := filepath.Join(t.TempDir(), "Nguyễn Văn A")
	datHome(t, home)

	if got := Home(); got != home {
		t.Fatalf("Home() = %q, chờ %q", got, home)
	}
	got := AccountsRoot()
	if !strings.Contains(got, "Nguyễn Văn A") {
		t.Fatalf("AccountsRoot() = %q — tên người dùng bị cắt/đổi", got)
	}
	if got != filepath.Join(home, ".ai-accounts") {
		t.Fatalf("AccountsRoot() = %q, chờ %q", got, filepath.Join(home, ".ai-accounts"))
	}
}

// Đường dẫn phải TUYỆT ĐỐI và đã sạch (không còn "." hay ".." ở giữa).
// Chỗ gọi dùng chúng làm gốc để ghép rồi XOÁ; một đường dẫn tương đối sẽ tính
// theo thư mục hiện hành của tiến trình — thứ thay đổi tuỳ người dùng gõ lệnh
// ở đâu, và không ai kiểm soát được.
func TestDuongDanTraVeLaTuyetDoiVaDaSach(t *testing.T) {
	datHome(t, t.TempDir())

	for ten, p := range map[string]string{
		"Home":                 Home(),
		"AccountsRoot":         AccountsRoot(),
		"LegacyClaudeAccounts": LegacyClaudeAccounts(),
	} {
		if !filepath.IsAbs(p) {
			t.Fatalf("%s() = %q, không phải đường dẫn tuyệt đối", ten, p)
		}
		if p != filepath.Clean(p) {
			t.Fatalf("%s() = %q, chưa sạch (Clean ra %q)", ten, p, filepath.Clean(p))
		}
	}
}

// GHI NHẬN MỘT CÁI BẪY, không phải khen một tính năng.
//
// Khi biến môi trường trống, os.UserHomeDir trả lỗi và Home() nuốt lỗi đó, trả
// chuỗi rỗng. Hệ quả đo được: AccountsRoot() thành ".ai-accounts" — một đường
// dẫn TƯƠNG ĐỐI, tức kho hồ sơ sẽ mọc ra ngay tại thư mục người dùng đang đứng.
//
// Test này chốt lại hai điều: (1) Home() rỗng chứ không panic, (2) đúng lúc đó
// hai hàm kia mất tính tuyệt đối. Ai sửa Home() để báo lỗi tử tế hơn thì test
// này gãy — và gãy ở đây là tin tốt, nó chỉ thẳng chỗ cần xem lại.
func TestThieuBienMoiTruongThiHomeRongVaKhoTroThanhDuongDanTuongDoi(t *testing.T) {
	// os.UserHomeDir chỉ đọc đúng một biến trên mỗi nền tảng (%USERPROFILE%
	// trên Windows, $HOME ở nơi khác), nên đặt rỗng một biến là dựng xong bẫy.
	datHome(t, "")

	if got := Home(); got != "" {
		t.Skipf("máy này vẫn suy ra được HOME (%q) — không dựng được bẫy", got)
	}
	kho := AccountsRoot()
	if filepath.IsAbs(kho) {
		t.Fatalf("Home() rỗng mà AccountsRoot() = %q vẫn tuyệt đối — hành vi đã đổi, xem lại chú thích trên", kho)
	}
	if kho != ".ai-accounts" {
		t.Fatalf("AccountsRoot() = %q, chờ %q", kho, ".ai-accounts")
	}
	t.Logf("bẫy đã ghi nhận: HOME rỗng -> AccountsRoot() = %q (tương đối, mọc theo thư mục hiện hành)", kho)
}
