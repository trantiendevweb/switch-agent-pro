package api

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Sáu hành động dưới đây đều biến tên tài khoản thành đường dẫn thư mục, và ba
// trong số đó XOÁ. Test này bắt từng cái phải từ chối tên thoát thư mục.
//
// Vì sao liệt kê thay vì tin vào một chốt chặn: thêm action mới mà quên gọi
// Validate là chuyện rất dễ xảy ra, và hậu quả thì không sửa lại được — token và
// dữ liệu người dùng đã bị xoá rồi. Thêm action nhận Addr thì thêm một dòng ở đây.
func TestMoiHanhDongTuChoiTenThoatThuMuc(t *testing.T) {
	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	} else {
		t.Setenv("HOME", tmp)
	}
	// Dữ liệu thật mà kẻ tấn công nhắm tới.
	that := filepath.Join(tmp, ".claude")
	if err := os.MkdirAll(that, 0o755); err != nil {
		t.Fatal(err)
	}

	a, err := New(tmp)
	if err != nil {
		t.Skipf("không mở được store trong môi trường test: %v", err)
	}
	defer a.Close()

	doc := Addr{"claude", filepath.Join("..", "..", ".claude")}

	hanhDong := map[string]func() error{
		"profile.create": func() error { _, _, e := a.ProfileCreate(doc); return e },
		"profile.remove": func() error { return a.ProfileRemove(doc) },
		"profile.run":    func() error { return a.ProfileRun(doc, []string{"--version"}) },
		"clones.create":  func() error { _, e := a.ClonesCreate(doc, 2); return e },
		"clones.clean":   func() error { _, e := a.ClonesClean(doc, tmp, false); return e },
		"fleet.start":    func() error { _, e := a.FleetStart(FleetRequest{Addr: doc, Copies: 1}); return e },
	}

	for ten, goi := range hanhDong {
		err := goi()
		if err == nil {
			t.Errorf("%s: nhận tên thoát thư mục mà không báo lỗi", ten)
			continue
		}
		// Phải hỏng vì TÊN, không phải tình cờ hỏng vì lý do khác.
		if !strings.Contains(err.Error(), "không hợp lệ") {
			t.Errorf("%s: từ chối nhưng không phải vì tên: %v", ten, err)
		}
	}

	if _, err := os.Stat(that); err != nil {
		t.Fatalf("DỮ LIỆU THẬT ~/.claude BỊ ĐỤNG: %v", err)
	}
}
