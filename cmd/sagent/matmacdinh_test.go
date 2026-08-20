package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/config"
)

// Pha 5d, phần terminal: gõ `sagent` không tham số phải ra ĐÚNG mặt mà
// `ui.default_surface` khai, không phải luôn luôn là bảng chọn.
//
// Ba nhóm test dưới đây canh ba chỗ dễ trôi nhất:
//   - tên mặt hợp lệ ở config và bảng URL ở CLI phải khớp nhau,
//   - cổng in ra phải là cổng `sagent dash` thật sự mở,
//   - mặt terminal phải rơi về TUI chứ không in lời chỉ đường rỗng.

// matHopLeTheoConfig là bộ tên mà config.validate() cho lọt. Chép tay ở đây CÓ
// CHỦ Ý: nếu ai đó thêm mặt thứ năm vào validate mà quên bảng URL, test dưới
// phải đỏ chứ không được tự động đúng theo.
var matHopLeTheoConfig = []string{"tui", "dashboard", "workflow", "3d"}

func TestMoiMatWebDeuCoDuongDan(t *testing.T) {
	for _, mat := range matHopLeTheoConfig {
		if mat == "tui" {
			continue // mặt terminal không có URL, đó là điểm khác biệt của nó
		}
		loi, laWeb := chiDuongMat(mat, dashPortMacDinh)
		if !laWeb {
			t.Errorf("config nhận %q là mặt hợp lệ nhưng CLI không biết nó nằm ở URL nào", mat)
			continue
		}
		if !strings.Contains(loi, "sagent dash") {
			t.Errorf("%s: lời chỉ đường không nói cách bật server", mat)
		}
	}
}

func TestMatTerminalKhongPhaiMatWeb(t *testing.T) {
	// "" là trạng thái của project chưa khai gì — phải xử như "tui", không được
	// rơi vào nhánh web rồi in URL cho một mặt người dùng chưa chọn.
	for _, mat := range []string{"", "tui"} {
		if _, laWeb := chiDuongMat(mat, dashPortMacDinh); laWeb {
			t.Errorf("%q phải là mặt terminal, không phải mặt web", mat)
		}
	}
}

func TestLoiChiDuongInDungCong(t *testing.T) {
	loi, ok := chiDuongMat("dashboard", 9999)
	if !ok {
		t.Fatal("dashboard phải là mặt web")
	}
	// Cổng phải đi theo tham số, không được chép cứng: người chạy
	// `sagent dash --port 9999` mà được bảo mở 4600 thì nhận trang trắng.
	if !strings.Contains(loi, ":9999/") {
		t.Errorf("lời chỉ đường không mang cổng đã truyền:\n%s", loi)
	}
	if strings.Contains(loi, fmt.Sprintf(":%d", dashPortMacDinh)) {
		t.Errorf("cổng mặc định bị chép cứng vào lời chỉ đường:\n%s", loi)
	}
}

func TestBaMatWebTroToiBaTrangKhacNhau(t *testing.T) {
	thay := map[string]string{}
	for _, mat := range []string{"dashboard", "workflow", "3d"} {
		w, ok := matWeb[mat]
		if !ok {
			t.Fatalf("thiếu mặt %q trong bảng URL", mat)
		}
		if truoc, trung := thay[w.Duong]; trung {
			// Hai mặt cùng URL nghĩa là một trong hai không mở được bao giờ.
			t.Errorf("%q và %q cùng trỏ tới %s", mat, truoc, w.Duong)
		}
		thay[w.Duong] = mat
	}
}

// Bảng URL không được nhận tên mà config sẽ từ chối: khai được ở CLI mà không
// khai được trong project.toml thì đó là mặt ma.
func TestBangURLKhongCoMatLa(t *testing.T) {
	hopLe := map[string]bool{}
	for _, m := range matHopLeTheoConfig {
		hopLe[m] = true
	}
	for mat := range matWeb {
		if !hopLe[mat] {
			t.Errorf("bảng URL có %q nhưng config.validate() sẽ từ chối tên này", mat)
		}
	}
}

// CotMacDinh phải in ra được thành một dòng đọc hiểu — `sagent config` dùng nó
// để nói "đang dùng mặc định" thay vì bỏ trống dòng columns.
func TestCotMacDinhInRaDuoc(t *testing.T) {
	dong := strings.Join(config.CotMacDinh, ", ")
	if dong == "" {
		t.Fatal("CotMacDinh rỗng — dòng ui.columns sẽ in ra chuỗi trống")
	}
	if !strings.Contains(dong, "trang_thai") {
		t.Errorf("bộ cột mặc định không có cột trạng thái: %q", dong)
	}
}
