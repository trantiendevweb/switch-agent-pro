package api

import "testing"

// Chuỗi NGUYÊN VĂN mà antigravity trả về ở lần chạy #8 (18/08 14:17). Bước
// `ke-hoach` đã được đánh dấu `done` với đúng chuỗi này, và bước sau lấy nó làm
// dữ liệu đầu vào — nên flow báo `completed` trên rác.
const outThatBiChanQuyen = `jetski: no output produced — a tool required the "command" permission that headless mode cannot prompt for, so it was auto-denied. Add an allow-rule under permissions.allow in settings.json (e.g. command(<target>)). Alternatively, re-run with --dangerously-skip-permissions to auto-approve all tools.`

func TestBiChanQuyenKhongPhaiThanhCong(t *testing.T) {
	ly := khongCoKetQua(outThatBiChanQuyen)
	if ly == "" {
		t.Fatal("output từ chối quyền bị coi là THÀNH CÔNG — đúng lỗi đã làm lần chạy #8 báo completed trên rác")
	}
}

func TestKhongInGiLaThatBai(t *testing.T) {
	for _, out := range []string{"", "   ", "\n\n\t\n"} {
		if khongCoKetQua(out) == "" {
			t.Fatalf("output rỗng %q bị coi là thành công", out)
		}
	}
}

// Không được bắt nhầm: kết quả thật phải đi lọt.
func TestKetQuaThatDiLot(t *testing.T) {
	that := []string{
		"Go",
		"Ngôn ngữ chính của repo là Go.\nCó 42 file .go.",
		"Đã sửa internal/api/api.go và chạy test: PASS",
		// có chữ "output" nhưng không phải chữ ký bị chặn
		"Tôi đã kiểm tra output của lệnh build, không có lỗi.",
	}
	for _, out := range that {
		if ly := khongCoKetQua(out); ly != "" {
			t.Fatalf("kết quả thật %q bị coi là hỏng: %s", out, ly)
		}
	}
}

// Hai chuỗi NGUYÊN VĂN đo được ở lần chạy #21. Cả hai bước đều bị đánh dấu
// `done` và cả flow vẫn `completed` — tức lá chắn cũ bỏ lọt hai kiểu hỏng.
const (
	outThatHongXacThuc = `Failed to authenticate: OAuth session expired and could not be refreshed`
	outThatQuanVong    = `{"role":"assistant","content":"Maximum tool execution rounds reached. Stopping to prevent infinite loops."}`
)

func TestHongXacThucKhongPhaiThanhCong(t *testing.T) {
	if khongCoKetQua(outThatHongXacThuc) == "" {
		t.Fatal("agent KHÔNG đăng nhập được mà vẫn tính là xong — đúng lỗi lần chạy #21")
	}
}

func TestQuanVongGoiToolKhongPhaiThanhCong(t *testing.T) {
	if khongCoKetQua(outThatQuanVong) == "" {
		t.Fatal("agent quẩn vòng tới khi hết trần mà vẫn tính là xong — đúng lỗi lần chạy #21")
	}
}

// Vẫn không được bắt nhầm: nói VỀ đăng nhập thì khác với hỏng đăng nhập.
func TestNoiVeDangNhapKhongBiBatNham(t *testing.T) {
	that := []string{
		"Đã thêm test cho luồng đăng nhập, tất cả xanh.",
		"Hàm này trả lỗi khi session hết hiệu lực; đã có test.",
	}
	for _, out := range that {
		if ly := khongCoKetQua(out); ly != "" {
			t.Fatalf("kết quả thật %q bị coi là hỏng: %s", out, ly)
		}
	}
}
