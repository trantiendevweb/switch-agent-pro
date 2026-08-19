package api

import (
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// giaAdapter chỉ để kiểm ba nhánh quyết định quyền.
type giaAdapter struct {
	provider.Adapter
	ten  string
	co   []string
	daDo bool
}

func (g giaAdapter) Name() string                       { return g.ten }
func (g giaAdapter) HeadlessArgs(p string) []string     { return []string{"-p", p} }
func (g giaAdapter) ArgsTuDuyetQuyen() ([]string, bool) { return g.co, g.daDo }

// Không bật cờ thì TUYỆT ĐỐI không được có cờ nguy hiểm trong dòng lệnh.
func TestKhongBatThiKhongCoCoNguyHiem(t *testing.T) {
	ad := giaAdapter{ten: "thu", co: []string{"--dangerously-skip-permissions"}, daDo: true}
	args, _, err := argsChoBuoc(ad, "", "việc", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range args {
		if strings.Contains(a, "dangerous") || strings.Contains(a, "skip-permission") {
			t.Fatalf("BƯỚC KHÔNG XIN QUYỀN MÀ DÒNG LỆNH CÓ CỜ NGUY HIỂM: %v", args)
		}
	}
}

// Bật cờ thì cờ phải đứng TRƯỚC prompt, không được nuốt mất.
func TestBatThiCoPhaiVaoDongLenh(t *testing.T) {
	ad := giaAdapter{ten: "thu", co: []string{"--dangerously-skip-permissions"}, daDo: true}
	args, _, err := argsChoBuoc(ad, "", "việc", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) == 0 || args[0] != "--dangerously-skip-permissions" {
		t.Fatalf("cờ không tới nơi: %v", args)
	}
}

// CHƯA ĐO thì phải BÁO LỖI, không được lặng lẽ chạy không quyền rồi báo xong.
func TestChuaDoThiBaoLoiChuKhongChayLen(t *testing.T) {
	ad := giaAdapter{ten: "cursor", co: nil, daDo: false}
	if _, _, err := argsChoBuoc(ad, "", "việc", true); err == nil {
		t.Fatal("provider CHƯA ĐO mà vẫn chạy tiếp — người dùng tưởng agent có quyền")
	}
}

// ĐÃ ĐO và provider KHÔNG có rào (Grok): không phải lỗi, nhưng PHẢI cảnh báo —
// kể cả khi bước không xin quyền, vì người dùng dễ tưởng không bật cờ là an toàn.
func TestKhongCoRaoThiPhaiCanhBao(t *testing.T) {
	ad := giaAdapter{ten: "grok", co: nil, daDo: true}
	for _, xin := range []bool{true, false} {
		_, canhBao, err := argsChoBuoc(ad, "", "việc", xin)
		if err != nil {
			t.Fatalf("xin=%v: không nên là lỗi: %v", xin, err)
		}
		if canhBao == "" {
			t.Fatalf("xin=%v: provider KHÔNG có rào quyền mà im lặng", xin)
		}
	}
}
