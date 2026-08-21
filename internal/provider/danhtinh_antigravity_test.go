package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Ba dòng NGUYÊN VĂN đã thấy trong nhật ký `agy` thật (bản 1.1.17), chỉ thay
// email bằng địa chỉ bịa. KHÔNG chép token vào repo — nhật ký không chứa token,
// và đó chính là lý do đọc nó rẻ hơn mở Credential Manager.
const (
	dongApply = "ERROR: logging before google.Init: I0821 21:17:09.186767       1 " +
		"server_oauth.go:190] applyAuthResult: email=EMAIL, authMethod=consumer, quotaProject=\n"
	dongOAuth = "ERROR: logging before google.Init: I0821 21:17:09.186834       1 " +
		"server_oauth.go:195] OAuth: authenticated successfully as EMAIL\n"
	dongTrinhDuyet = "ERROR: logging before google.Init: I0821 14:42:26.136877     450 " +
		"browser.go:161] consumerOAuth: authenticated successfully as EMAIL\n"
	dongRac = "ERROR: logging before google.Init: I0821 21:17:08.000000       1 " +
		"jetski.go:42] starting workspace indexer\n"
)

func dong(mau, email string) string { return strings.ReplaceAll(mau, "EMAIL", email) }

// dungHoSoAgy dựng một thư mục hồ sơ đúng HÌNH DẠNG thật: configDir đóng vai
// USERPROFILE, nhật ký nằm ở <configDir>/.gemini/antigravity-cli/log/.
func dungHoSoAgy(t *testing.T, files map[string]string, mtime map[string]time.Time) string {
	t.Helper()
	goc := t.TempDir()
	logDir := filepath.Join(goc, ".gemini", "antigravity-cli", "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for ten, noiDung := range files {
		p := filepath.Join(logDir, ten)
		if err := os.WriteFile(p, []byte(noiDung), 0o644); err != nil {
			t.Fatal(err)
		}
		if mt, co := mtime[ten]; co {
			if err := os.Chtimes(p, mt, mt); err != nil {
				t.Fatal(err)
			}
		}
	}
	return goc
}

func TestAntigravityDocEmailTuNhatKy(t *testing.T) {
	goc := dungHoSoAgy(t, map[string]string{
		"cli-20260821_211657.log": dongRac +
			dong(dongApply, "nguoi-dung-gia@example.com") +
			dong(dongOAuth, "nguoi-dung-gia@example.com"),
	}, nil)

	if got := danhTinhTuNhatKyAgy(goc); got != "nguoi-dung-gia@example.com" {
		t.Fatalf("đọc nhật ký ra %q, muốn nguoi-dung-gia@example.com", got)
	}
}

// BÀI KIỂM GHIM ĐÚNG Ô NỢ V2. google_accounts.json là file của GEMINI CLI: nó
// chứa một email CÓ THẬT nhưng có thể SAI NGƯỜI, và trên máy đo nó đứng im suốt
// những lượt `agy` mới. Hồ sơ ở đây có CẢ HAI nguồn và chúng nói khác nhau —
// hàm phải lấy nguồn do chính `agy` ghi.
func TestAntigravityKhongLayEmailTuGoogleAccounts(t *testing.T) {
	goc := dungHoSoAgy(t, map[string]string{
		"cli-20260821_211657.log": dong(dongOAuth, "nguoi-that@example.com"),
	}, nil)

	gaJSON := `{"active":"nguoi-cu-cua-gemini-cli@example.com","old":[]}`
	if err := os.WriteFile(filepath.Join(goc, ".gemini", "google_accounts.json"),
		[]byte(gaJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	got := danhTinhTuNhatKyAgy(goc)
	if got == "nguoi-cu-cua-gemini-cli@example.com" {
		t.Fatal("đã đọc google_accounts.json — đúng cái bẫy ô V2 dựng ra để chặn: " +
			"một email CÓ THẬT nhưng SAI NGƯỜI")
	}
	if got != "nguoi-that@example.com" {
		t.Fatalf("đọc ra %q, muốn nguoi-that@example.com (nguồn do chính agy ghi)", got)
	}
}

// Nhật ký MỚI NHẤT thắng: đăng nhập lại thì lượt sau mới là lượt đang có hiệu lực.
func TestAntigravityLayNhatKyMoiNhat(t *testing.T) {
	cu := time.Date(2026, 8, 18, 10, 30, 0, 0, time.Local)
	moi := time.Date(2026, 8, 21, 21, 17, 0, 0, time.Local)
	goc := dungHoSoAgy(t, map[string]string{
		"cli-20260818_103003.log": dong(dongTrinhDuyet, "tai-khoan-cu@example.com"),
		"cli-20260821_211657.log": dong(dongTrinhDuyet, "tai-khoan-moi@example.com"),
	}, map[string]time.Time{
		"cli-20260818_103003.log": cu,
		"cli-20260821_211657.log": moi,
	})

	if got := danhTinhTuNhatKyAgy(goc); got != "tai-khoan-moi@example.com" {
		t.Fatalf("đọc ra %q, muốn tai-khoan-moi@example.com", got)
	}
}

// Trong MỘT file, dòng xác thực CUỐI thắng: phiên dài có thể xác thực lại giữa chừng.
func TestAntigravityLayDongXacThucCuoiTrongFile(t *testing.T) {
	goc := dungHoSoAgy(t, map[string]string{
		"cli-20260821_144204.log": dong(dongTrinhDuyet, "dau-phien@example.com") +
			dongRac +
			dong(dongApply, "cuoi-phien@example.com"),
	}, nil)

	if got := danhTinhTuNhatKyAgy(goc); got != "cuoi-phien@example.com" {
		t.Fatalf("đọc ra %q, muốn cuoi-phien@example.com", got)
	}
}

// Không có nhật ký, hoặc nhật ký không có dòng xác thực → RỖNG. Trả rỗng là câu
// trả lời ĐÚNG khi không biết; ô V2 tồn tại vì chiều ngược lại tệ hơn hẳn.
func TestAntigravityRongKhiKhongCoBangChung(t *testing.T) {
	if got := danhTinhTuNhatKyAgy(filepath.Join(t.TempDir(), "khong-ton-tai")); got != "" {
		t.Fatalf("thư mục không tồn tại mà trả %q", got)
	}

	goc := dungHoSoAgy(t, map[string]string{"cli-20260821_211657.log": dongRac}, nil)
	if got := danhTinhTuNhatKyAgy(goc); got != "" {
		t.Fatalf("nhật ký không có dòng xác thực mà trả %q", got)
	}

	trong := dungHoSoAgy(t, map[string]string{}, nil)
	if got := danhTinhTuNhatKyAgy(trong); got != "" {
		t.Fatalf("thư mục log rỗng mà trả %q", got)
	}
}

// `cli.log` là symlink trỏ tới file mới nhất. Bỏ qua nó, nếu không thì cùng một
// nội dung bị cân nhắc hai lần dưới hai dấu thời gian khác nhau.
func TestAntigravityBoQuaCliLog(t *testing.T) {
	moi := time.Date(2026, 8, 21, 21, 17, 0, 0, time.Local)
	ratMoi := time.Date(2026, 8, 22, 9, 0, 0, 0, time.Local)
	goc := dungHoSoAgy(t, map[string]string{
		"cli-20260821_211657.log": dong(dongOAuth, "dung@example.com"),
		"cli.log":                 dong(dongOAuth, "sai@example.com"),
	}, map[string]time.Time{
		"cli-20260821_211657.log": moi,
		"cli.log":                 ratMoi,
	})

	if got := danhTinhTuNhatKyAgy(goc); got != "dung@example.com" {
		t.Fatalf("đọc ra %q, muốn dung@example.com — cli.log lẽ ra bị bỏ qua", got)
	}
}

// Lời khai phải KHỚP hàm, cả hai chiều — cùng lối với
// TestCursorKhaiDanhTinhKhopVoiHam. Phép dò NLDanhTinh trong nangluc.go là MỘT
// CHIỀU (chỉ bắt được "khai chưa đo mà trả giá trị thật"), nên chiều "khai làm
// được mà luôn trả rỗng" phải có bài kiểm riêng đứng canh — đó chính là chỗ ô V3
// đã chỉ ra và ô này thừa hưởng.
//
// Gọi thẳng danhTinhTuNhatKyAgy chứ không gọi Identity: Identity còn một cổng
// Credential Manager, mà máy chạy bài kiểm chưa chắc có mục `gemini:antigravity`.
// Bài kiểm phải đo phần PHỤ THUỘC HỒ SƠ, không đo phần phụ thuộc máy.
func TestAntigravityKhaiDanhTinhKhopVoiHam(t *testing.T) {
	goc := dungHoSoAgy(t, map[string]string{
		"cli-20260821_211657.log": dong(dongOAuth, "nguoi-dung-gia@example.com"),
	}, nil)
	docDuoc := danhTinhTuNhatKyAgy(goc) != ""

	var khai TrangThaiNangLuc
	for _, nl := range (antigravity{}).NangLuc() {
		if nl.Khoa == NLDanhTinh {
			khai = nl.TrangThai
		}
	}
	switch {
	case docDuoc && khai != LamDuoc:
		t.Fatalf("hàm ĐỌC ĐƯỢC danh tính từ hồ sơ mà bảng năng lực khai %q — "+
			"bảng đang nói thấp hơn sự thật", khai)
	case !docDuoc && khai == LamDuoc:
		t.Fatal("bảng năng lực khai LÀM ĐƯỢC mà hàm trả RỖNG trên hồ sơ đúng hình dạng thật — " +
			"lời hứa không có gì đứng sau")
	}
}

// Identity phải trả RỖNG khi hồ sơ không có gì để đọc, bất kể máy có mục
// Credential Manager hay không. Đây cũng là điều phép dò một chiều của
// KiemNangLuc dựa vào: Identity(thuMucKhongTonTai) phải là "".
func TestAntigravityIdentityRongTrenThuMucKhongTonTai(t *testing.T) {
	if got := (antigravity{}).Identity(filepath.Join(t.TempDir(), "khong-ton-tai")); got != "" {
		t.Fatalf("Identity trên thư mục không tồn tại trả %q, muốn rỗng", got)
	}
}
