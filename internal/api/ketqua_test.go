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


// Ca BAO DONG GIA do duoc o lan chay #23: buoc `hoc-acp` clone xong hai repo va
// viet bao cao that, nhung GIUA DUONG co mot lan bi chan quyen. La chan ban dau
// soi ca ban ghi nen giet oan ca buoc.
//
// Agent gap tro ngai roi di duong khac la chuyen binh thuong. Chi khi no KET
// THUC bang cau hong thi moi la hong.
func TestGapTroNgaiGiuaDuongVanLaThanhCong(t *testing.T) {
	out := `Bat dau doc repo.
jetski: no output produced - a tool required the "command" permission that headless mode cannot prompt for, so it was auto-denied.
Doi cach: dung cong cu doc file thay cho shell.
Da clone nong vao /tmp/acp-study, chi doc.

## 1. ACP dinh nghia cac loai su kien
session/update, session/request_permission, tool_call - xem schema.json:120.
## 4. Viec nho nhat de bat dau: boc mot adapter ACP cho Claude Code.`
	if ly := khongCoKetQua(out); ly != "" {
		t.Fatalf("agent gap tro ngai giua duong roi van lam xong ma bi coi la hong: %s", ly)
	}
}

// Nhung KET THUC bang cau bi chan quyen thi van phai bat - du truoc do co lam gi.
func TestKetThucBangBiChanQuyenVanBiBat(t *testing.T) {
	out := `Dang doc repo, da liet ke 40 file.
` + outThatBiChanQuyen
	if khongCoKetQua(out) == "" {
		t.Fatal("agent ket thuc bang cau tu choi quyen ma van tinh la xong")
	}
}
