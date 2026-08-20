// Quyền ĐỌC KẾT QUẢ giữa các bước — `doc_duoc` trong flows.toml.
//
// Mặc định của bộ thực thi là MỞ HẾT: một bước đọc được kết quả của MỌI bước đã
// chạy xong trước nó (xem WithOutputs). Tiện, nhưng có giá đo được — lượt chạy
// #34, bước `gop` nhận 10.998 token vào, phần lớn là output của những bước nó
// không cần đọc.
//
// `doc_duoc` cho bước tự khai nó được đọc kết quả của ĐÚNG những bước nào:
//
//	[[flow.doi-4.step]]
//	  id = "soi"
//	  doc_duoc = ["kiem-2"]
//
// BA quyết định ở đây, và cả ba đều cố ý:
//
//  1. KHÔNG khai = giữ nguyên hành vi cũ (đọc mọi bước trước). Đổi mặc định
//     thành cấm hết sẽ làm hỏng mọi flow đang chạy được — không đáng.
//  2. Bước bị chặn KHÔNG bị thay bằng chuỗi rỗng mà bằng một câu NÓI RA việc
//     chặn. Cắt dữ liệu trong im lặng đúng là loại lỗi lượt #29 (prompt trỏ tới
//     kết quả không tồn tại, agent nhận chữ sống làm đề bài) mà dự án này chống.
//  3. Khai sai chỉ CẢNH BÁO, không chặn: người viết flow có thể khai trước rồi
//     mới thêm bước, y như `vai_tro`.
package flow

import (
	"fmt"
	"strings"
)

// CauChan là câu thay vào chỗ kết quả của một bước KHÔNG được phép đọc.
//
// Có tên bước và có cả cách sửa: người đọc prompt (hoặc agent đang đọc nó) biết
// ngay vì sao chỗ này trống và phải thêm gì vào đâu để mở ra.
func CauChan(id string) string {
	return fmt.Sprintf("(không được phép đọc kết quả bước %q — thêm %s vào doc_duoc nếu cần)", id, id)
}

// LocDocDuoc lọc map kết quả theo `doc_duoc` của bước s, TRƯỚC khi dựng env.
//
// s.DocDuoc == nil (không khai) thì trả về nguyên map cũ — hành vi cũ, không
// một byte nào đổi. Khai rồi thì bước ngoài danh sách bị thay bằng CauChan chứ
// không bị xoá khỏi map: xoá đi thì ExpandChay lại nói "bước x không để lại kết
// quả", tức là đổ lỗi cho bước kia trong khi thủ phạm là chính lời khai này.
func LocDocDuoc(s Step, outputs map[string]string) map[string]string {
	if s.DocDuoc == nil {
		return outputs
	}
	cho := make(map[string]bool, len(s.DocDuoc))
	for _, id := range s.DocDuoc {
		cho[id] = true
	}
	out := make(map[string]string, len(outputs))
	for id, o := range outputs {
		if cho[id] {
			out[id] = o
			continue
		}
		out[id] = CauChan(id)
	}
	return out
}

// MoTaDocDuoc là câu mô tả quyền đọc của một bước, để mọi mặt in ra giống nhau.
//
// Ba trạng thái phải phân biệt được, vì chúng khác nhau thật:
//
//	nil          → "mọi bước trước"  (chưa khai — mặc định mở)
//	[]           → "không bước nào"  (khai rỗng — cấm hết, cố ý)
//	["a", "b"]   → "a, b"
func MoTaDocDuoc(s Step) string {
	if s.DocDuoc == nil {
		return "mọi bước trước"
	}
	if len(s.DocDuoc) == 0 {
		return "không bước nào"
	}
	return strings.Join(s.DocDuoc, ", ")
}

// VanDeDocDuoc soi phần `doc_duoc` của cả flow và trả về CẢNH BÁO (không phải
// lỗi) cho hai kiểu khai hỏng:
//
//   - trỏ tới bước không tồn tại — gõ nhầm tên, và hậu quả là im lặng: bước đó
//     có được nhắc hay không thì kết quả vẫn y hệt nhau.
//   - trỏ tới bước chạy CÙNG ĐỢT hoặc SAU — lúc bước này chạy thì bước kia chưa
//     có kết quả nào, nên lời khai chẳng mở ra được gì.
//
// Thứ tự đợt lấy từ Dot() — đúng cách bộ thực thi chia đợt, không phải một bản
// chép tay dễ lệch. Flow có chu trình thì bỏ qua: Validate đã báo chỗ đó rồi, và
// flow như thế không có thứ tự đợt nào để mà so.
func VanDeDocDuoc(f Flow) []Problem {
	coKhai := false
	for _, s := range f.Steps {
		if s.DocDuoc != nil {
			coKhai = true
			break
		}
	}
	if !coKhai {
		return nil
	}

	co := map[string]bool{}
	for _, s := range f.Steps {
		co[s.ID] = true
	}
	dots, err := Dot(f)
	if err != nil {
		return nil
	}
	dotCua := map[string]int{}
	for _, d := range dots {
		for _, s := range d.Buoc {
			dotCua[s.ID] = d.So
		}
	}

	var ps []Problem
	for _, s := range f.Steps {
		for _, id := range s.DocDuoc {
			if !co[id] {
				ps = append(ps, Problem{Flow: f.Name, Step: s.ID, Warn: true,
					Msg: fmt.Sprintf("doc_duoc trỏ tới bước %q không tồn tại — lời khai này không mở ra gì cả", id)})
				continue
			}
			minh, kia := dotCua[s.ID], dotCua[id]
			if kia < minh {
				continue
			}
			if kia == minh {
				ps = append(ps, Problem{Flow: f.Name, Step: s.ID, Warn: true,
					Msg: fmt.Sprintf("doc_duoc trỏ tới bước %q chạy CÙNG ĐỢT %d (song song) — lúc bước này chạy thì bước đó chưa có kết quả", id, kia)})
				continue
			}
			ps = append(ps, Problem{Flow: f.Name, Step: s.ID, Warn: true,
				Msg: fmt.Sprintf("doc_duoc trỏ tới bước %q chạy SAU (đợt %d, còn bước này ở đợt %d) — lúc bước này chạy thì bước đó chưa có kết quả", id, kia, minh)})
		}
	}
	return ps
}
