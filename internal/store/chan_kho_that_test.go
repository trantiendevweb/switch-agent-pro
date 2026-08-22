package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Chốt chặn kho thật phải NỔ THẬT, không phải nằm im rồi cả bộ test xanh vì
// tình cờ không ai chạm tới.
//
// Đây là điểm yếu chết người của mọi chốt liên động: một bộ test xanh không
// phân biệt được "không ai vi phạm" với "chốt hỏng". Bài kiểm này tách hai câu
// đó ra.
//
// Cố ý gọi thẳng `kiemKhoThat` chứ không gọi `Open()`: nếu chốt hỏng thì
// `Open()` sẽ MỞ THẬT sổ của máy — tức bài kiểm dùng để canh lại chính là bài
// kiểm gây ra thứ nó canh.
func TestChotChanKhoThatNoThat(t *testing.T) {
	// Mắt xích dễ gãy nhất: phép nhận ra mình đang chạy dưới binary test. Nó
	// dựa vào hậu tố `.test` / `.test.exe` của os.Args[0]. Go đổi cách đặt tên
	// binary test thì chốt tự vô hiệu hoá TRONG IM LẶNG — và không có dòng này
	// thì không gì báo.
	if !duoiTest() {
		t.Fatalf("duoiTest() = false ngay trong một bài kiểm (os.Args[0] = %q) — "+
			"chốt chặn kho thật đang TẮT ở mọi gói, im lặng", os.Args[0])
	}

	err := kiemKhoThat(Path())
	if err == nil {
		t.Fatalf("kiemKhoThat(%q) cho đi qua — chốt không chặn được chính đường "+
			"đã làm chết mặt điều khiển ngày 22/08", Path())
	}
	if !strings.Contains(err.Error(), "SỔ TRẠNG THÁI THẬT") {
		t.Errorf("thông điệp không nói ra chuyện gì đang xảy ra: %v", err)
	}

	// Đường HỢP LỆ phải đi qua trơn tru, nếu không thì mọi bài kiểm khác đỏ và
	// người ta sẽ gỡ chốt thay vì sửa bài kiểm.
	if err := kiemKhoThat(filepath.Join(t.TempDir(), "state.db")); err != nil {
		t.Errorf("chốt chặn nhầm một đường dẫn tạm hợp lệ: %v", err)
	}

	// Cửa thoát phải mở được, và phải mở ĐÚNG BẰNG biến đã ghi trong tài liệu.
	t.Setenv(BienChoPhep, "1")
	if err := kiemKhoThat(Path()); err != nil {
		t.Errorf("đặt %s=1 rồi mà chốt vẫn chặn — bài kiểm nào cố ý cần kho thật "+
			"sẽ không có đường nào đi ngoài việc gỡ chốt: %v", BienChoPhep, err)
	}
}
