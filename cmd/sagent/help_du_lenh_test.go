package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// MỌI LỆNH CẤP MỘT PHẢI XUẤT HIỆN TRONG `sagent help`.
//
// Vì sao có bài kiểm này (22/08): `cmdHelp()` là một **chuỗi viết tay**, không
// sinh ra từ bảng `commands`. Thêm một lệnh vào bảng là xong về mặt chạy được
// — nhưng nó **không tự hiện ra trong help**, và không có gì báo. Lệnh
// `nang-luc-api` vừa thêm hôm nay rơi đúng vào khe đó: chạy được ngay, mà
// người dùng gõ `sagent help` thì không bao giờ biết nó tồn tại.
//
// Đây là cùng một hình dạng với ba lỗi khác của repo trong ngày: `route.kiem`,
// nút Duyệt/Từ chối, và `plugin.list` — thứ gì đó "có" ở mọi tầng trừ tầng
// người dùng thật sự chạm vào. Luật ngang quyền canh api.Actions ↔ CLI ↔ HTTP ↔
// web-UI; nó **không** canh CLI ↔ help, và help mới là chỗ người ta đi tìm.
//
// Bài kiểm KHÔNG đòi help giải thích mọi cờ. Nó chỉ đòi: gõ `sagent help` thì
// thấy được tên lệnh. Một lệnh không ai biết là một lệnh không tồn tại.
func TestMoiLenhCapMotDeuXuatHienTrongHelp(t *testing.T) {
	than := batHelp(t)
	if len(than) < 500 {
		t.Fatalf("help chỉ dài %d ký tự — nghi là không bắt được đầu ra, và một "+
			"bài kiểm không đọc được gì thì luôn xanh", len(than))
	}

	var thieu []string
	for ten := range commands {
		// `__` là quy ước của bảng này cho lệnh CON hoặc CỜ (`route kiem`,
		// `ds --so`, `api --lich-su`). Chúng khai ở đây để test ngang quyền
		// thấy, nhưng không phải tên gõ được ở cấp một.
		if strings.HasPrefix(ten, "__") {
			continue
		}
		if !strings.Contains(than, "sagent "+ten) {
			thieu = append(thieu, ten)
		}
	}

	if len(thieu) > 0 {
		t.Errorf("%d lệnh chạy được nhưng KHÔNG xuất hiện trong `sagent help`: %s\n"+
			"Người dùng không có đường nào biết chúng tồn tại. Thêm một dòng vào "+
			"chuỗi help trong cmdHelp() (main.go).", len(thieu), strings.Join(thieu, ", "))
	}
}

// batHelp bắt đầu ra của cmdHelp().
//
// Bắt stdout chứ không tách chuỗi help ra thành hằng số: tách ra là sửa mã sản
// phẩm cho vừa bài kiểm, và bài kiểm sẽ đo cái hằng số đó thay vì đo thứ người
// dùng thật sự thấy.
func batHelp(t *testing.T) string {
	t.Helper()
	cu := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	xong := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		xong <- buf.String()
	}()

	cmdHelp()

	w.Close()
	os.Stdout = cu
	return <-xong
}
