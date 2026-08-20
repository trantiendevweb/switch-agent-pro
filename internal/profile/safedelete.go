package profile

// XOÁ-AN-TOÀN THEO SỔ ĐĂNG KÝ.
//
// MASTER-PLAN Pha 1: "Safe delete: chỉ xoá materialized session directory mà
// registry sở hữu."
//
// `Remove` (profile.go) đã có hai lá chắn: đường dẫn phải nằm trong kho, và
// không xoá xuyên link. Cả hai đều là phép kiểm trên ĐĨA, và cả hai đều trả lời
// câu hỏi "thư mục này có tồn tại đúng chỗ không" — không cái nào trả lời được
// câu "ai dựng ra nó". Với `insideStore`, hễ thứ gì rơi vào ~/.ai-accounts thì
// mặc nhiên bị coi là của sagent và xoá đệ quy được: một thư mục người dùng tự
// chép vào đó để dành cũng vậy.
//
// Sổ đăng ký là chỗ trả lời câu còn thiếu. Tầng này chỉ HỎI sổ, không tự đoán.

import (
	"fmt"

	"github.com/trantiendevweb/switch-agent-pro/internal/link"
)

// SoDangKy là phần TỐI THIỂU mà xoá-an-toàn cần ở sổ đăng ký.
//
// Khai bằng interface chứ không nhận thẳng *store.DB, vì `profile` là tầng dưới:
// kéo cả SQLite (và driver của nó) vào đây để hỏi đúng một câu hỏi bool là đổi
// hướng phụ thuộc của dự án cho một tiện nghi. store.DB thoả mãn interface này
// sẵn — xem store.(*DB).SoHuu.
type SoDangKy interface {
	// SoHuu trả về true nếu SỔ khẳng định thư mục này do sagent dựng ra.
	SoHuu(dir string) (bool, error)
}

// RemoveTheoSo xoá một hồ sơ, nhưng chỉ khi sổ đăng ký nhận sở hữu nó.
//
// Ba nhánh, mỗi nhánh vì một cách hỏng khác nhau:
//
//  1. NGOÀI KHO → từ chối. Giữ nguyên lá chắn cũ, đặt trước cả câu hỏi cho sổ:
//     một dòng sổ bịa ra đường dẫn ~/.claude cũng không được mở cửa.
//
//  2. CHÍNH NÓ LÀ LINK → chỉ gỡ link, KHÔNG hỏi sổ và KHÔNG xoá đệ quy. Gỡ một
//     junction là tháo cánh cửa chứ không đụng gì tới căn phòng bên kia, nên
//     việc đó luôn an toàn và luôn đúng ý người gõ `xoa`. Hỏi sổ ở đây còn TỆ
//     HƠN: nếu sổ (vì bất cứ lý do gì) nhận sở hữu đúng đường dẫn đó, câu trả
//     lời "có" sẽ cấp phép cho `os.RemoveAll` đi xuyên link — đúng cú đã xoá
//     mất ~/.claude ngày 2026-08-17. link.IsLink kiểm cờ reparse point, thứ duy
//     nhất nhìn ra junction trên Windows (os.Lstat KHÔNG thấy).
//
//  3. THƯ MỤC THẬT → phải được sổ nhận sở hữu mới xoá.
//
// Sổ đọc hỏng thì KHÔNG xoá. Đây là chỗ chọn "thà không làm còn hơn làm nhầm":
// làm ngược lại nghĩa là một cái DB bị khoá cũng đủ biến xoá-an-toàn về lại
// xoá-đệ-quy, và không ai thấy gì cả.
func RemoveTheoSo(dir string, so SoDangKy) error {
	if !insideStore(dir) {
		return fmt.Errorf("từ chối xoá %s — nằm ngoài kho hồ sơ", dir)
	}
	if laLink, _ := link.IsLink(dir); laLink {
		return link.Unlink(dir, true)
	}
	if so == nil {
		return fmt.Errorf("từ chối xoá %s — không có sổ đăng ký để hỏi", dir)
	}
	coSo, err := so.SoHuu(dir)
	if err != nil {
		return fmt.Errorf("không đọc được sổ đăng ký nên KHÔNG xoá %s: %w", dir, err)
	}
	if !coSo {
		return fmt.Errorf("sổ đăng ký không nhận sở hữu %s — sagent chỉ xoá thư mục do chính nó tạo ra. "+
			"Muốn bỏ thứ này thì xoá bằng tay", dir)
	}
	return Remove(dir)
}

// SagentQuan là BẰNG CHỨNG TRÊN ĐĨA để nhận một hồ sơ có sẵn vào sổ.
//
// Sổ mới có từ schema v8, nên mọi hồ sơ tạo trước đó (và mọi hồ sơ di trú từ v1)
// đều không có dòng nào. Không có đường nhận chúng vào thì `xoa` gãy cho đúng
// những người dùng lâu năm nhất — nên chỗ nhận vào sổ dùng hàm này để quyết
// SoTaoRa.
//
// Hai điều kiện, và cả hai đều là điều kiện về CẤU TRÚC chứ không phải về nội
// dung: thư mục nằm trong kho hồ sơ (kho là của sagent, không dùng chung với ai)
// và bản thân nó không phải link (link thì dữ liệu thật ở đầu bên kia, sagent
// không sở hữu chỗ đó). Cố tình KHÔNG suy ra sở hữu từ "có file .credentials.json"
// hay bất cứ dấu hiệu nội dung nào — nội dung thì ai chép vào cũng được.
func SagentQuan(dir string) bool {
	if !insideStore(dir) {
		return false
	}
	laLink, err := link.IsLink(dir)
	return err == nil && !laLink
}
