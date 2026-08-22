// CHỌN ĐƯỜNG THEO NĂNG LỰC — phần soi THUẦN, không cần bảng năng lực.
//
// ============================================================================
// BƯỚC KHAI NHU CẦU Ở ĐÂU: TRONG flows.toml, KHÔNG SUY TỪ NỘI DUNG BƯỚC
// ============================================================================
//
// Hai hướng đã cân nhắc, và hai hậu quả KHÔNG đối xứng nhau:
//
//	SUY TỪ NỘI DUNG — đọc prompt rồi đoán "câu này chắc cần gọi tool". Đoán sai
//	thì bộ chọn LOẠI một route đang sống và đang làm được việc, rồi đi đường
//	khác — hoặc dừng hẳn với câu "không route nào đủ năng lực". Người viết flow
//	KHÔNG có cách nào cãi lại, vì họ chưa từng khai gì để mà sửa. Tệ hơn nữa:
//	đổi một chữ trong prompt là đổi đường đi, mà bảng thì vẫn ghi "model".
//
//	KHAI TAY — `can = ["goi-tool"]`. Quên khai thì bước chạy y hệt như trước khi
//	trường này tồn tại: chọn đường theo sức khoẻ, hỏng lúc chạy nếu route không
//	làm được. Tức là hậu quả của việc quên ĐÚNG BẰNG hiện trạng, không tệ hơn.
//
// Chọn KHAI TAY, vì đó là hướng mà lỗi của công cụ không lấn quyền quyết định
// của người dùng. Giá phải trả là "thêm một chỗ người ta quên", và giá đó được
// trả bằng ba cảnh báo dựng ngay dưới đây, chứ không bằng một lời hứa:
//
//   - bước `model` đòi năng lực mà bước chọn đường của nó KHÔNG lọc theo năng
//     lực nào → cảnh báo, kèm đúng dòng cần thêm;
//   - bước `model` đòi X mà bước chọn đường lọc theo Y (thiếu X) → cảnh báo,
//     kèm danh sách khoá còn thiếu;
//   - `can` khai ở bước không phải `route`/`model` → LỖI: đó là một dòng chết
//     nằm đó trông như có tác dụng, cùng lớp với `routes` khai nhầm chỗ.
//
// ============================================================================
// VÌ SAO PHẦN SOI KHOÁ KHÔNG NẰM Ở ĐÂY
// ============================================================================
//
// Danh sách khoá hợp lệ là `aiapi.MoiNangLucAPI`. Gói `flow` KHÔNG import
// `aiapi` — chiều phụ thuộc của dự án là api → flow, và bẻ chiều đó để soi một
// danh sách chuỗi là đổi kiến trúc lấy một tiện nghi. Chép danh sách sang đây
// thì tệ hơn: hai bản của một từ vựng là hai bản sẽ lệch, và bản lệch bao giờ
// cũng lệch về phía "khoá này lạ" cho một năng lực vừa được thêm.
//
// Nên phần soi khoá — và phần tra bảng để biết route có làm được không — nằm ở
// internal/api/flow_nangluc.go, chỗ CÓ cả hai. Xem ở đó.
package flow

import (
	"fmt"
	"strings"
)

// VanDeCan soi phần `can` của cả flow, bằng những gì gói này tự biết.
func VanDeCan(f Flow) []Problem {
	var ps []Problem
	loi := func(id, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: id, Msg: msg}) }
	nhac := func(id, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: id, Msg: msg, Warn: true}) }

	buoc := map[string]Step{}
	for _, s := range f.Steps {
		buoc[s.ID] = s
	}

	for _, s := range f.Steps {
		if len(s.Can) == 0 {
			continue
		}

		// `can` ở một bước không đi qua đường API nào là một dòng chết. Nó nằm
		// đó trông như có tác dụng — và người viết flow tưởng bước `shell` của
		// mình đang được canh.
		if s.Type != TypeRoute && s.Type != TypeModel {
			loi(s.ID, fmt.Sprintf("khai `can` nhưng type = %q — chỉ bước `route` (lọc ứng viên) "+
				"và bước `model` (kiểm đường đã khai) mới hỏi tới năng lực của route", s.Type))
			continue
		}

		thay := map[string]bool{}
		for _, k := range s.Can {
			t := strings.TrimSpace(k)
			switch {
			case t == "":
				loi(s.ID, "`can` có một mục rỗng")
			case thay[t]:
				loi(s.ID, fmt.Sprintf("`can` khai %q hai lần", t))
			}
			thay[t] = true
		}
	}

	// Chiều đáng giá nhất: bước `model` đòi một năng lực, mà bước CHỌN ĐƯỜNG cho
	// nó lại không biết điều đó. Bộ chọn sẽ chọn theo sức khoẻ rồi trả về một
	// đường không làm được việc — và bước `model` chỉ phát hiện ra lúc gọi thật,
	// sau khi các bước trước đã tiêu token.
	//
	// Đây chính là chỗ trả giá cho lựa chọn KHAI TAY, nên nó phải nói ra ĐÚNG
	// dòng cần thêm, không phải một lời nhắc chung chung.
	for _, s := range f.Steps {
		if s.Type != TypeModel || len(s.Can) == 0 {
			continue
		}
		id := BuocTrongRoute(s.Route)
		if id == "" {
			continue // route khai cứng — phần tra bảng nằm ở internal/api
		}
		ch, co := buoc[id]
		if !co || ch.Type != TypeRoute {
			continue // đã có lỗi riêng cho hai ca đó ở VanDeRoute
		}
		thieu := thieuKhoa(ch.Can, s.Can)
		if len(thieu) == 0 {
			continue
		}
		if len(ch.Can) == 0 {
			nhac(s.ID, fmt.Sprintf("bước này đòi %s, nhưng bước chọn đường %q KHÔNG lọc theo "+
				"năng lực nào — nó sẽ chọn đường chỉ theo sức khoẻ, và có thể trả về một đường "+
				"không làm được việc. Thêm vào bước %q: can = [%s]",
				dsKhoa(s.Can), id, id, nhayKhoa(s.Can)))
			continue
		}
		nhac(s.ID, fmt.Sprintf("bước này đòi %s, nhưng bước chọn đường %q chỉ lọc theo %s — "+
			"thiếu %s. Thêm vào bước %q: can = [%s]",
			dsKhoa(s.Can), id, dsKhoa(ch.Can), dsKhoa(thieu), id, nhayKhoa(gopKhoa(ch.Can, s.Can))))
	}
	return ps
}

// thieuKhoa trả về những khoá trong `can` mà `co` không có.
func thieuKhoa(co, can []string) []string {
	sn := map[string]bool{}
	for _, k := range co {
		sn[strings.TrimSpace(k)] = true
	}
	var ra []string
	for _, k := range can {
		if t := strings.TrimSpace(k); t != "" && !sn[t] {
			ra = append(ra, t)
		}
	}
	return ra
}

// gopKhoa gộp hai danh sách, giữ thứ tự và bỏ trùng.
func gopKhoa(a, b []string) []string {
	var ra []string
	thay := map[string]bool{}
	for _, k := range append(append([]string(nil), a...), b...) {
		if t := strings.TrimSpace(k); t != "" && !thay[t] {
			thay[t] = true
			ra = append(ra, t)
		}
	}
	return ra
}

func dsKhoa(ks []string) string { return "`" + strings.Join(ks, "`, `") + "`" }
func nhayKhoa(ks []string) string {
	q := make([]string, len(ks))
	for i, k := range ks {
		q[i] = `"` + k + `"`
	}
	return strings.Join(q, ", ")
}

// MoTaCan là câu mô tả nhu cầu năng lực của một bước, để mọi mặt in giống nhau.
// Rỗng = bước không đòi gì.
func MoTaCan(s Step) string {
	if len(s.Can) == 0 {
		return ""
	}
	return strings.Join(s.Can, ", ")
}
