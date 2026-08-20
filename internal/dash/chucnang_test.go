package dash

import (
	"os"
	"strings"
	"testing"
)

// Tam duong dan API mat 2D PHAI con goi duoc sau moi lan bay lai giao dien.
//
// VI SAO CAN LA CHAN NAY: bay lai giao dien la luc de danh roi nut nhat. Luot
// chay #41 don sau khoi form vao ngan keo — mot thao tac dung vao ca tam duong
// nay cung luc. Khong co test thi mat mot nut chi lo ra khi nguoi dung di tim
// no, ma luc do khong ai nho lan sua nao lam mat.
//
// Danh sach nay chi duoc DAI RA khi them tinh nang, khong duoc ngan lai vi mot
// lan bay lai giao dien. Bo mot duong that su thi phai sua test VA noi ro vi sao.
var duongPhaiCon = []string{
	"/api/state",       // hạm đội, tài khoản, danh sách lượt chạy
	"/api/events",      // SSE — nguồn cập nhật realtime
	"/api/fleet",       // bật hạm đội
	"/api/stop",        // dừng phiên
	"/api/quet",        // quét tiến trình mồ côi
	"/api/ai",          // hỏi thẳng AI API
	"/api/ai/lich-su",  // sổ lời gọi API: tiêu bao nhiêu, ở đâu, có chạy được không
	"/api/db",          // xem sổ trạng thái
	"/api/flow/run",    // chạy workflow
	"/api/flow/kho",    // chạy khan
	"/api/flow/detail", // tiến độ từng bước
}

func TestMat2DKhongDanhRoiDuongNaoSauKhiBayLai(t *testing.T) {
	b, err := os.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	ma := string(b)
	for _, d := range duongPhaiCon {
		if !strings.Contains(ma, d) {
			t.Errorf("index.html khong con goi %s — bay lai giao dien lam mat mot chuc nang. "+
				"Neu CO Y bo thi sua danh sach duongPhaiCon va ghi ro vi sao.", d)
		}
	}
}

// Tuong tu cho man hoi thoai: no chi song duoc nho hai duong nay.
func TestManHoiThoaiKhongDanhRoiDuongNao(t *testing.T) {
	b, err := os.ReadFile("web/hoi-thoai.html")
	if err != nil {
		t.Fatal(err)
	}
	ma := string(b)
	for _, d := range []string{"/api/state", "/api/events", "/api/flow/detail"} {
		if !strings.Contains(ma, d) {
			t.Errorf("hoi-thoai.html khong con goi %s", d)
		}
	}
}
