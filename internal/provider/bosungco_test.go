package provider

import (
	"strings"
	"testing"
)

// LỖI THẬT, đo 20/08/2026: `sagent status` hiện 20 phiên liền, phiên nào cũng
// "chết, chưa rõ vì sao", cột tokens và chi phí đều "chưa đo" — trong khi CÙNG
// LÚC đó lượt chạy flow #47 trên đúng những tài khoản ấy đo được 99.051 token
// vào, 81.492 token ra, 11,0572 USD.
//
// Nguyên nhân: `fleet` truyền args THÔ cho CLI con, còn flow đi qua
// `argsChoBuoc` nên được adapter dựng args. Người dùng gõ `-- -p "việc"` là đủ
// để agent chạy, nhưng thiếu cờ in bản ghi có cấu trúc thì `DocKetQua` không có
// gì để đọc, và mọi phiên rơi về `lost`.

func TestBoSungDuCoChoClaude(t *testing.T) {
	them := CoConThieu(claude{}, []string{"-p", "làm việc gì đó"})
	got := strings.Join(them, " ")
	// Phải thêm ĐÚNG phần thiếu — không thêm lại `-p` mà người dùng đã gõ.
	for _, can := range []string{"--output-format", "stream-json", "--verbose"} {
		if !strings.Contains(got, can) {
			t.Errorf("thiếu %q trong phần bổ sung: %q", can, got)
		}
	}
	if strings.Contains(got, "-p ") || got == "-p" {
		t.Errorf("thêm lại cờ người dùng đã gõ: %q", got)
	}
}

func TestKhongThemGiKhiDaDuCo(t *testing.T) {
	day := []string{"-p", "việc", "--output-format", "stream-json", "--verbose"}
	if them := CoConThieu(claude{}, day); len(them) > 0 {
		t.Errorf("args đã đủ mà vẫn thêm %v", them)
	}
}

// Người dùng tự chọn định dạng khác thì TÔN TRỌNG giá trị của họ: cờ đã có mặt
// là đủ, không ghi đè. Ghi đè lựa chọn tường minh còn tệ hơn im lặng.
func TestTonTrongGiaTriNguoiDungDaChon(t *testing.T) {
	them := CoConThieu(claude{}, []string{"-p", "việc", "--output-format", "json"})
	for _, x := range them {
		if x == "--output-format" || x == "stream-json" {
			t.Errorf("ghi đè --output-format mà người dùng đã tự chọn: %v", them)
		}
	}
}

// Provider CHƯA ĐO cách đọc kết quả có cấu trúc thì không được khai bừa cờ:
// một cờ không tồn tại làm CLI con chết ngay dòng đầu, tức là đổi "không đo
// được" thành "không chạy được" — tệ hơn hẳn.
func TestKhongKhaiBuaCoChoProviderChuaDo(t *testing.T) {
	for _, ad := range []Adapter{codex{}, cursor{}} {
		if TrangThaiCua(ad, NLKetQuaCoCauTruc) == LamDuoc {
			continue // đã đo rồi thì bài này không nói gì về nó
		}
		if them := CoConThieu(ad, []string{"-p", "việc"}); len(them) > 0 {
			t.Errorf("%s chưa đo %s mà vẫn thêm %v", ad.Name(), NLKetQuaCoCauTruc, them)
		}
	}
}

// Lệnh con (`codex exec`) KHÔNG phải cờ — thêm nó vào cuối dòng lệnh là hỏng cú
// pháp. Chỉ những phần tử bắt đầu bằng "-" mới được coi là cờ.
func TestKhongCoiLenhConLaCo(t *testing.T) {
	for _, x := range CoConThieu(codex{}, nil) {
		if !strings.HasPrefix(x, "-") && x != "stream-json" && x != "json" {
			t.Errorf("coi %q là cờ — nó là lệnh con hoặc giá trị", x)
		}
	}
}

func TestTrangThaiCuaKhongKhaiThiChuaDo(t *testing.T) {
	// Adapter thủng bảng khai thì phải ra ChuaDo, không phải LamDuoc: đoán
	// "làm được" ở đây là cách sai nhất, vì nó dẫn tới khai bừa cờ.
	if got := TrangThaiCua(claude{}, "nang-luc-khong-ton-tai"); got != ChuaDo {
		t.Errorf("năng lực không khai phải ra ChuaDo, được %q", got)
	}
}
