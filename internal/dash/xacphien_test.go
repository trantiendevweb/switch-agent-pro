package dash

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// LỖI THẬT, ảnh chụp 20/08/2026: panel ghi "HẠM ĐỘI 0" trong khi bên dưới là 20
// thẻ to bằng nhau, thẻ nào cũng "chết, chưa rõ vì sao" — nhìn vào không biết
// hạm đội đang chạy hay đã tắt hẳn.
//
// Hai thứ sai cùng lúc:
//   - xác đi chung lưới với phiên sống nên chiếm hết chỗ của thứ đang chạy;
//   - ô "chưa có phiên nào" ẩn đi vì `ss.length` đếm cả xác, nên hạm đội tắt
//     hết mà màn hình không nói một câu nào.
//
// Xác VẪN phải giữ lại (nguyên tắc #3: việc đã xảy ra là dữ liệu thật), chỉ
// thôi tranh chỗ.

func doc2DThuong(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestXacGomRiengKhongDiChungLuoiPhienSong(t *testing.T) {
	s := doc2DThuong(t)
	if !strings.Contains(s, `id="xac"`) {
		t.Fatal("index.html: không có khối gom phiên đã chết — xác vẫn đi chung lưới với phiên sống")
	}
	ma := boComment(s)
	// Vòng vẽ lưới phải RẼ SỚM cho phiên chết, không vẽ thẻ đầy đủ rồi mới ẩn.
	if !regexp.MustCompile(`if\s*\(\s*!dangSong\(s\)\s*\)`).MatchString(ma) {
		t.Error("index.html: vòng vẽ hạm đội không tách phiên chết ra khỏi lưới card")
	}
	if !strings.Contains(ma, "dongXac(") {
		t.Error("index.html: không có hàm dựng dòng gọn cho phiên đã chết")
	}
}

// Khối xác phải ĐÓNG SẴN. Mở sẵn thì 20 dòng vẫn đẩy mọi thứ khác xuống dưới
// màn hình — đúng vấn đề cũ, chỉ khác kiểu chữ.
func TestKhoiXacDongSan(t *testing.T) {
	s := doc2DThuong(t)
	i := strings.Index(s, `id="xac"`)
	if i < 0 {
		t.Fatal("không có khối xác")
	}
	the := s[i:]
	if j := strings.Index(the, ">"); j > 0 {
		the = the[:j]
	}
	if strings.Contains(the, " open") {
		t.Error("khối xác mở sẵn — 20 dòng vẫn đẩy phần còn lại xuống dưới màn hình")
	}
	// <details> chứ không phải div + JS: mở/đóng phải làm được bằng bàn phím,
	// và phải còn hoạt động cả khi script chết giữa chừng.
	if !regexp.MustCompile(`<details[^>]*id="xac"`).MatchString(s) {
		t.Error("khối xác không phải <details> — mất khả năng mở bằng bàn phím")
	}
}

// Ô "chưa có phiên nào" phải theo số phiên SỐNG, không theo tổng.
func TestOTrongTheoSoPhienSongChuKhongPhaiTong(t *testing.T) {
	ma := boComment(doc2DThuong(t))
	if regexp.MustCompile(`ham-empty'\)\.style\.display\s*=\s*ss\.length`).MatchString(ma) {
		t.Error("index.html: ô \"chưa có phiên nào\" vẫn đếm cả xác — hạm đội tắt hết " +
			"mà màn hình không nói gì")
	}
	if !regexp.MustCompile(`ham-empty'\)\.style\.display\s*=\s*nSong`).MatchString(ma) {
		t.Error("index.html: ô trống không đọc số phiên đang sống")
	}
}

// Dòng xác không được có nút Dừng: dừng một phiên đã chết là nút vô nghĩa.
func TestDongXacKhongCoNutDung(t *testing.T) {
	ma := boComment(doc2DThuong(t))
	i := strings.Index(ma, "function dongXac")
	if i < 0 {
		t.Fatal("không có dongXac")
	}
	than := ma[i:]
	if j := strings.Index(than, "\nfunction "); j > 0 {
		than = than[:j]
	}
	if strings.Contains(than, "data-stop") {
		t.Error("dòng xác có nút Dừng — phiên đã chết thì không dừng được nữa")
	}
	// Nhưng PHẢI nói vì sao chết: đó là lý do duy nhất còn giữ nó lại.
	if !strings.Contains(than, "lyDo") {
		t.Error("dòng xác không hiện lý do chết — giữ xác lại mà không nói vì sao thì giữ làm gì")
	}
}

// Mũi tên đóng/mở phải vẽ bằng CSS, không bằng ký tự — design system cấm dùng
// ký tự làm icon, mà hộp đóng/mở không có dấu hiệu thì không ai biết bấm được.
func TestMuiTenKhoiXacVeBangCSS(t *testing.T) {
	s := doc2DThuong(t)
	if !strings.Contains(s, ".xac[open]>summary::before") {
		t.Error("index.html: khối xác không có dấu hiệu đóng/mở vẽ bằng CSS")
	}
	if !strings.Contains(s, "list-style:none") {
		t.Error("index.html: không tắt marker mặc định của <summary> — sẽ có hai mũi tên")
	}
}
