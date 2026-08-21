package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ĐO THẬT 21/08/2026 (ô Đ5 của `docs/SO-NO-DO-LUONG.md`): hai bản clone mang
// CÙNG một refresh token, cùng bị ép hết hạn, bật cách nhau 16ms.
//
// Kết quả đo, nguyên văn số:
//
//	A (thắng)  exit=0  is_error=false   refresh 86fb2200 -> d22e079d
//	B (thua)   exit=1  is_error=true    "Failed to authenticate: OAuth session
//	                                     expired and could not be refreshed"
//
// Chi tiết quyết định — thứ KHÔNG đoán ra được nếu không chạy thật:
//
//	A/.credentials.json  ghi lúc 11:19:35.484
//	B/.credentials.json  ghi lúc 11:19:35.500   <- MỚI HƠN 16ms
//
// Bản THUA ghi file SAU bản thắng, vì nó hỏng nhanh (186ms) còn bản thắng phải
// đợi hết lượt gọi API. Và thứ nó ghi ra là một file RỖNG: accessToken "",
// refreshToken "", expiresAt 0 — nhưng `refreshTokenExpiresAt` và
// `subscriptionType: "max"` thì còn nguyên, nên nhìn qua vẫn giống hồ sơ đã
// đăng nhập.
//
// Ghép hai điều đó lại thành cái bẫy mà bài này canh: `SyncBackTokens` chọn bản
// clone theo mtime MỚI NHẤT khi hồ sơ gốc còn token. Sau một cuộc đua, bản mới
// nhất là bản THUA — tức là file rỗng. Đồng bộ ngược sẽ chép cái rỗng đó đè lên
// hồ sơ gốc và giết nốt bản cuối cùng còn dùng được.
//
// Đây đúng là "mất tài khoản lần thứ hai theo một chuỗi khác nhưng cùng một
// gốc" mà sổ nợ đã cảnh báo.
func ghiTokenBanThua(t *testing.T, p string) {
	t.Helper()
	// Nguyên văn hình dạng đã đo được ở bản thua, chỉ bỏ phần bí mật (vốn đã
	// rỗng) — giữ lại các trường còn sót để bài kiểm không dễ hơn thực tế.
	than := `{"claudeAiOauth":{"accessToken":"","refreshToken":"","expiresAt":0,` +
		`"refreshTokenExpiresAt":1789634099575,"scopes":["user:inference"],` +
		`"subscriptionType":"max","rateLimitTier":"default_claude_max_20x"}}`
	if err := os.WriteFile(p, []byte(than), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestKhongDongBoNguocFileRongCuaBanThuaCuocDua(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := adapterDocThat{fakeAdapter{base: fakeBase, hasToken: true}}
	home := homeCuaTest(t)
	gocTok := filepath.Join(home, ".ai-accounts", "fake", "phu", ".credentials.json")

	// Hồ sơ gốc giữ token CŨ. Nhà cung cấp đã vô hiệu nó từ lúc cấp bản mới,
	// nhưng file thì vẫn đủ trường nên `HasToken` vẫn nói "có" — đúng như thật.
	ghiToken(t, gocTok, "token-cu-da-bi-vo-hieu")
	dirs, err := Clone(a, "phu", 2)
	if err != nil {
		t.Fatal(err)
	}

	// Cuộc đua: bản 1 thắng và ghi token mới; bản 2 thua và ghi đè file RỖNG,
	// SAU bản thắng. Thứ tự này là thứ tự đã đo được, không phải giả định.
	thang := filepath.Join(dirs[0], ".credentials.json")
	thua := filepath.Join(dirs[1], ".credentials.json")
	ghiToken(t, thang, "token-moi-duy-nhat-con-song")
	ghiTokenBanThua(t, thua)
	t1 := time.Now().Add(2 * time.Second)
	t2 := t1.Add(1 * time.Second) // bản thua mới hơn
	if err := os.Chtimes(thang, t1, t1); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(thua, t2, t2); err != nil {
		t.Fatal(err)
	}

	// Lượt bật hạm đội kế tiếp.
	if _, err := Clone(a, "phu", 2); err != nil {
		t.Fatalf("từ chối trong khi vẫn còn một bản token sống: %v", err)
	}

	goc, err := os.ReadFile(gocTok)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(goc), `"refreshToken":""`) {
		t.Fatalf("đồng bộ ngược chép file RỖNG của bản thua đè lên hồ sơ gốc — "+
			"mất nốt bản cuối cùng còn dùng được.\n được: %s", goc)
	}
	if !strings.Contains(string(goc), "token-moi-duy-nhat-con-song") {
		t.Errorf("hồ sơ gốc không nhận token của bản THẮNG.\n được: %s", goc)
	}
}

// Mặt còn lại: cuộc đua không được làm hỏng bản clone đang THẮNG.
//
// Cùng một lượt `Clone`, sau khi đồng bộ ngược là bước chép đè từ gốc ra mọi
// bản. Nếu bước đồng bộ chọn nhầm file rỗng thì cái rỗng đó lan ra CẢ N bản —
// hỏng một chỗ thành hỏng hết, và lúc đó không còn gì để cứu.
func TestBanThangKhongBiFileRongCuaBanThuaLanSang(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := adapterDocThat{fakeAdapter{base: fakeBase, hasToken: true}}
	gocTok := filepath.Join(homeCuaTest(t), ".ai-accounts", "fake", "phu", ".credentials.json")

	ghiToken(t, gocTok, "token-cu-da-bi-vo-hieu")
	dirs, err := Clone(a, "phu", 2)
	if err != nil {
		t.Fatal(err)
	}
	thang := filepath.Join(dirs[0], ".credentials.json")
	ghiToken(t, thang, "token-moi-duy-nhat-con-song")
	ghiTokenBanThua(t, filepath.Join(dirs[1], ".credentials.json"))
	t1 := time.Now().Add(2 * time.Second)
	t2 := t1.Add(1 * time.Second)
	if err := os.Chtimes(thang, t1, t1); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(dirs[1], ".credentials.json"), t2, t2); err != nil {
		t.Fatal(err)
	}

	if _, err := Clone(a, "phu", 2); err != nil {
		t.Fatal(err)
	}
	for i, d := range dirs {
		got, err := os.ReadFile(filepath.Join(d, ".credentials.json"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), "token-moi-duy-nhat-con-song") {
			t.Errorf("clone #%d không có token sống sau lượt bật lại.\n được: %s", i+1, got)
		}
	}
}
