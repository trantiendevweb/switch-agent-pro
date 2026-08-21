package dash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Mọi ENDPOINT HÀNH ĐỘNG đều phải có ít nhất một trang web gọi tới.
//
// VÌ SAO CÓ (21/08): nút "Duyệt / Từ chối" biến mất khỏi mọi trang trong một lần
// vẽ lại giao diện. `/api/flow/decide` vẫn còn nguyên bên server, và luật ngang
// quyền cũ vẫn xanh — vì nó canh **API ↔ CLI**, không canh **UI ↔ API**. Người
// dùng mở dashboard thấy bước `waiting` mà không có chỗ bấm, phải quay về gõ
// `sagent flow approve`. Tệ hơn: `flow.html` vẫn in "Flow sẽ dừng ở đây tới khi
// bạn bấm Duyệt" — một lời hứa về cái nút không tồn tại.
//
// Bài kiểm này là mảnh còn thiếu đó. Nó KHÔNG kiểm nút trông thế nào; nó chỉ
// kiểm điều rẻ nhất mà đủ chặn: có đường nào từ web đi tới hành động đó không.
func TestMoiHanhDongCuaNguoiDungDeuCoDuongVaoTuWeb(t *testing.T) {
	// Chỉ liệt kê endpoint là HÀNH ĐỘNG của người dùng. Endpoint chỉ-đọc không
	// nằm ở đây: trang không gọi thì cùng lắm là thiếu thông tin, còn hành động
	// không gọi được là người dùng KHÔNG LÀM ĐƯỢC VIỆC.
	hanhDong := map[string]string{
		"/api/flow/decide": "duyệt / từ chối bước đang chờ",
		"/api/flow/run":    "chạy một flow",
		"/api/flow/cancel": "huỷ một lượt chạy",
	}

	trang, err := filepath.Glob(filepath.Join("web", "*.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(trang) == 0 {
		t.Fatal("không thấy trang web nào — bài kiểm này đang xanh vì rỗng, không phải vì sạch")
	}

	than := map[string]string{}
	for _, d := range trang {
		b, err := os.ReadFile(d)
		if err != nil {
			t.Fatalf("không đọc được %s: %v", d, err)
		}
		than[filepath.Base(d)] = boComment(string(b))
	}

	for duong, viec := range hanhDong {
		// Đối chiếu endpoint phải CÓ THẬT bên server, không thì bài kiểm này canh
		// một cái tên chết và xanh vô nghĩa.
		srv, err := os.ReadFile("server.go")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(srv), duong) {
			t.Errorf("server.go không còn %q — đổi tên endpoint thì sửa cả danh sách này", duong)
			continue
		}

		var goiBoi []string
		for ten, s := range than {
			if strings.Contains(s, duong) {
				goiBoi = append(goiBoi, ten)
			}
		}
		if len(goiBoi) == 0 {
			t.Errorf("KHÔNG trang web nào gọi %q (%s) — người dùng không làm được việc này "+
				"trên dashboard, phải quay về CLI. Đây đúng cách nút Duyệt biến mất ngày 21/08.",
				duong, viec)
		}
	}
}

// Dùng lại `boComment` có sẵn ở trungtam_quydao_test.go: nó cắt phần sau `//`
// của từng dòng, giữ lại phần MÃ THẬT SỰ CHẠY.
//
// VÌ SAO CẦN: bản đầu của bài kiểm này KHÔNG CẮN. Gỡ lời gọi thật ra khỏi
// `index.html` mà nó vẫn xanh — vì chuỗi `/api/flow/decide` còn nằm trong một
// dòng BÌNH LUẬN giải thích chính lỗi đó. Bình luận nhắc tên endpoint là chuyện
// tốt, không nên cấm; thứ phải sửa là phép dò.
//
// Cùng loại lỗi với TestMatWebDocDungTenTruongCuaDTO, và cùng được phát hiện
// bằng một cách: gỡ bản sửa ra xem bài kiểm có đỏ không.
