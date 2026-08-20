// Hợp đồng đầu ra của một bước — xem Step.PhaiCo.
package flow

import (
	"fmt"
	"strings"
)

// ThieuPhaiCo trả câu giải thích nếu output KHÔNG thoả hợp đồng, rỗng nếu thoả.
//
// So sánh KHÔNG phân biệt hoa thường: agent viết "nên trộn" hay "NÊN TRỘN" đều
// là cùng một kết luận, và bắt nó gõ đúng chữ hoa là biến hợp đồng thành một
// bài đố chính tả.
//
// Chuẩn hoá khoảng trắng trước khi so: agent hay xuống dòng giữa câu, và
// "NÊN\nTRỘN" vẫn là "NÊN TRỘN". Không chuẩn hoá thì hợp đồng trượt vì một ký
// tự xuống dòng — đúng kiểu bài kiểm gây phiền mà không bắt được gì.
func ThieuPhaiCo(s Step, out string) string {
	if len(s.PhaiCo) == 0 {
		return ""
	}
	got := strings.ToLower(gonKhoangTrang(out))
	for _, c := range s.PhaiCo {
		if c == "" {
			continue
		}
		if strings.Contains(got, strings.ToLower(gonKhoangTrang(c))) {
			return ""
		}
	}
	// Nói RÕ nó phải giao gì, và nói rõ đây không phải "agent hỏng" mà là "agent
	// chạy xong nhưng không giao ra thứ được giao". Hai chuyện khác nhau, và
	// người đọc báo cáo cần phân biệt được.
	return fmt.Sprintf("chạy xong nhưng KHÔNG có kết luận nào trong %s — "+
		"bước này coi như CHƯA LÀM, không phải đã làm và không có ý kiến",
		motTrong(s.PhaiCo))
}

// gonKhoangTrang đổi mọi cụm khoảng trắng (kể cả xuống dòng) thành một dấu cách.
func gonKhoangTrang(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func motTrong(ds []string) string {
	q := make([]string, 0, len(ds))
	for _, x := range ds {
		q = append(q, fmt.Sprintf("%q", x))
	}
	return strings.Join(q, " hoặc ")
}
