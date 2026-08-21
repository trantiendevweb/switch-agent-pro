package main

import (
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Mặt TERMINAL của nhật ký phiên.
//
// Có nhật ký mà không ai tìm ra thì bằng không có — đó là nửa còn lại của vấn
// đề ngày 21/08. Đường dẫn nhật ký nằm ở cột `log` của sổ, và người vận hành
// chỉ biết SỐ PHIÊN, nên hai bảng của `sagent status` phải nói ra cách tra.

// Lệnh phải GÕ ĐƯỢC bằng một tên có thật.
//
// TestNgangQuyen ở main_test.go chỉ đòi "có mục nào đó nhận action này" — mà
// mục đó có thể là chỗ giữ chỗ `__xxx` với run=nil (hợp lệ cho các CỜ của lệnh
// khác). Nhật ký thì không phải cờ của ai cả.
func TestCoLenhNhatKyGoDuoc(t *testing.T) {
	c, ok := commands["nhat-ky"]
	if !ok {
		t.Fatal("không có lệnh `sagent nhat-ky` — nhật ký chỉ mặt web đọc được, " +
			"đúng thứ luật ngang quyền cấm")
	}
	if c.run == nil {
		t.Fatal("lệnh `nhat-ky` không có hàm chạy")
	}
	if c.action != "session.nhat-ky" {
		t.Fatalf("lệnh `nhat-ky` gắn nhầm action %q", c.action)
	}
}

// Bảng phiên ĐÃ CHẾT là chỗ câu hỏi "vì sao" được hỏi. `StateLyDo` chỉ nói được
// một dòng, và với `lost` thì nó RỖNG — nên dòng chỉ đường tới nhật ký là thứ
// duy nhất giữ người đọc không dừng lại ở "chết, chưa rõ vì sao".
func TestDongPhienHongChiDuongToiNhatKy(t *testing.T) {
	s := store.Session{
		ID: 167, Provider: "claude", Account: "tns", Clone: 1,
		State: store.StateLost, Log: `C:\Users\x\.ai-accounts\.nhat-ky\claude-tns-c1-20260821-113905.000.log`,
	}
	d := dongPhienHong(s, mocDo)
	if !strings.Contains(d, "sagent nhat-ky 167") {
		t.Errorf("phiên chết mà không chỉ đường đọc nhật ký:\n%s", d)
	}
}

// Phiên KHÔNG có nhật ký (chạy trước bản này, hoặc không phải phiên fleet):
// không được bịa ra một lệnh chạy xong sẽ báo lỗi.
func TestDongPhienHongKhongBiaLenhKhiKhongCoNhatKy(t *testing.T) {
	s := store.Session{ID: 9, Provider: "claude", Account: "tns", State: store.StateLost}
	if d := dongPhienHong(s, mocDo); strings.Contains(d, "nhat-ky") {
		t.Errorf("phiên không có nhật ký mà vẫn chỉ đường đọc:\n%s", d)
	}
	if d := dongNhatKyPhien(s); d != "" {
		t.Errorf("dongNhatKyPhien() = %q, chờ rỗng", d)
	}
}

// Bảng nhật ký in TÊN FILE, và phải NÓI THẲNG khi file không còn.
//
// Im lặng ở đây thì người đọc tưởng công cụ làm mất, rồi đi tìm ở chỗ khác.
func TestTenNganNoiRoKhiNhatKyKhongCon(t *testing.T) {
	duong := `C:\Users\x\.ai-accounts\.nhat-ky\claude-tns-c1-20260821-113905.000.log`
	got := tenNgan(duong, true)
	if got != "claude-tns-c1-20260821-113905.000.log" {
		t.Errorf("tenNgan() = %q, chờ đúng tên file", got)
	}
	mat := tenNgan(duong, false)
	if !strings.Contains(mat, "đã dọn") {
		t.Errorf("nhật ký không còn mà bảng không nói ra: %q", mat)
	}
	if trong := tenNgan("", false); !strings.Contains(trong, "không có nhật ký") {
		t.Errorf("phiên không có nhật ký hiện ra %q", trong)
	}
}

func TestCoChuDocDuocSoByte(t *testing.T) {
	cap := []struct {
		n    int64
		chua string
	}{{12, "12 B"}, {2048, "2.0 KB"}, {3 << 20, "3.0 MB"}}
	for _, c := range cap {
		if got := coChu(c.n); got != c.chua {
			t.Errorf("coChu(%d) = %q, chờ %q", c.n, got, c.chua)
		}
	}
}
