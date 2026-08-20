package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
)

// TestNgangQuyen giữ LUẬT 2 của MASTER-PLAN mục 2c khỏi thành khẩu hiệu:
//
//	"Một tính năng chưa xong nếu chưa làm được từ CLI. UI được phép làm việc đó
//	 dễ hơn, không được là cách duy nhất."
//
// Thêm hành động vào api.Actions (để dùng cho dashboard/workflow board) mà quên
// lệnh CLI tương đương thì test này đỏ.
func TestNgangQuyenMoiHanhDongDeuCoLenhCLI(t *testing.T) {
	covered := map[string]string{} // action -> tên lệnh
	for name, c := range commands {
		if c.action == "" {
			t.Fatalf("lệnh %q không khai báo action — không kiểm được ngang quyền", name)
		}
		if prev, dup := covered[c.action]; dup {
			t.Fatalf("action %q bị hai lệnh cùng nhận: %q và %q", c.action, prev, name)
		}
		covered[c.action] = name
	}

	var thieu []string
	for _, a := range api.Actions {
		if _, ok := covered[a]; !ok {
			thieu = append(thieu, a)
		}
	}
	if len(thieu) > 0 {
		sort.Strings(thieu)
		t.Fatalf("các hành động sau chưa có lệnh CLI: %s\n"+
			"→ Thêm lệnh vào bảng `commands` trong main.go, hoặc bỏ khỏi api.Actions nếu không cần.",
			strings.Join(thieu, ", "))
	}
}

// Chiều ngược lại: CLI không được có hành động lạ nằm ngoài hợp đồng — nếu
// không, các mặt khác sẽ không bao giờ biết tính năng đó tồn tại.
func TestKhongCoLenhNgoaiHopDong(t *testing.T) {
	known := map[string]bool{}
	for _, a := range api.Actions {
		known[a] = true
	}
	for name, c := range commands {
		if !known[c.action] {
			t.Fatalf("lệnh %q dùng action %q không có trong api.Actions — mặt khác sẽ không thấy tính năng này",
				name, c.action)
		}
	}
}

// Mọi lệnh gõ được đều phải có hàm chạy. `__run` là ngoại lệ có chủ đích: nó
// ứng với việc gõ thẳng địa chỉ hồ sơ chứ không phải một từ khoá.
func TestLenhCoHamChay(t *testing.T) {
	for name, c := range commands {
		if c.run == nil && !strings.HasPrefix(name, "__") {
			t.Fatalf("lệnh %q không có hàm chạy", name)
		}
		if c.summary == "" {
			t.Fatalf("lệnh %q thiếu mô tả", name)
		}
	}
}

func TestParseAddrMacDinhClaude(t *testing.T) {
	if got := api.ParseAddr("phu"); got.Provider != "claude" || got.Account != "phu" {
		t.Fatalf("ParseAddr(\"phu\") = %+v", got)
	}
	if got := api.ParseAddr("codex:main"); got.Provider != "codex" || got.Account != "main" {
		t.Fatalf("ParseAddr(\"codex:main\") = %+v", got)
	}
}

func TestCoCacFlag(t *testing.T) {
	mine, child := splitDashDash([]string{"claude:phu", "--copies", "4", "--", "-p", "xin chao"})
	if len(mine) != 3 || len(child) != 2 {
		t.Fatalf("splitDashDash sai: mine=%v child=%v", mine, child)
	}
	n, rest := intFlag(mine, "--copies", 2)
	if n != 4 || len(rest) != 1 {
		t.Fatalf("intFlag sai: n=%d rest=%v", n, rest)
	}
	b, rest2 := boolFlag([]string{"a", "--force", "b"}, "--force")
	if !b || len(rest2) != 2 {
		t.Fatalf("boolFlag sai: b=%v rest=%v", b, rest2)
	}
}

// `--kho` phải sống sót qua mọi cờ khác đứng trước nó.
//
// Cờ này quyết định lượt chạy CÓ XẢY RA THẬT hay không, nên mất nó là mất tiền:
// mỗi hàm rút cờ trả về phần args còn lại, quên nối vào hàm sau thì --kho không
// bao giờ được thấy và `sagent flow run x --cu-chay --kho` chạy thật.
func TestCoKhoSongSotQuaCacCoKhac(t *testing.T) {
	y := docCoChay([]string{"--var", "a=b", "--profile", "claude:phu", "--cu-chay", "--kho"})
	if !y.kho {
		t.Fatal("--kho bị mất khi đứng sau các cờ khác — lượt chạy sẽ xảy ra THẬT")
	}
	if !y.cuChay || y.prof != "claude:phu" || y.vars["a"] != "b" {
		t.Fatalf("các cờ khác bị hỏng theo: %+v", y)
	}
	if docCoChay([]string{"--profile", "claude:phu"}).kho {
		t.Fatal("không gõ --kho mà vẫn hiểu là chạy khan")
	}
}

// Bảng năng lực phải gõ được bằng ĐÚNG một tên lệnh có thật.
//
// TestNgangQuyen ở trên chỉ đòi "có một mục nào đó nhận action này" — mà mục đó
// có thể là một chỗ giữ chỗ `__xxx` với run=nil, đúng cách mười một action khác
// trong bảng đang được khai (chúng là CỜ của lệnh khác, hợp lệ). Năng lực thì
// không phải cờ của ai cả: nó phải là một lệnh gõ được, nếu không thì người
// dùng terminal vẫn không có cách nào hỏi "provider này làm được gì".
func TestCoLenhNangLucGoDuoc(t *testing.T) {
	c, ok := commands["nang-luc"]
	if !ok {
		t.Fatal("không có lệnh `sagent nang-luc` — bảng năng lực chỉ mặt web xem được, " +
			"đúng thứ luật ngang quyền cấm")
	}
	if c.run == nil {
		t.Fatal("lệnh `nang-luc` không có hàm chạy")
	}
	if c.action != "provider.nang-luc" {
		t.Fatalf("lệnh `nang-luc` gắn nhầm action %q", c.action)
	}
}
