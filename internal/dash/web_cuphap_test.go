package dash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Một chuỗi JS bị ĐỨT GIỮA CHỪNG làm hỏng TOÀN BỘ script của trang, không phải
// chỉ dòng đó. Trang vẫn tải, khung vẫn hiện, nhưng không bảng nào có dữ liệu.
//
// Đã xảy ra thật: commit fe94ebf ghi `dong.join('` rồi xuống dòng thay vì
// `\n`, và bug đó SỐNG QUA NHIỀU BẢN BUILD. Dashboard hiện bảng tài khoản trống
// với dòng "đang kết nối…" mãi mãi — nhìn như lỗi mạng, thật ra là lỗi cú pháp.
// Không test nào bắt vì không ai chạy JS trong CI.
//
// Phép kiểm rẻ mà đủ: trong một dòng, số nháy đơn KHÔNG bị escape phải CHẴN.
// Chuỗi hợp lệ luôn mở và đóng trong cùng một dòng — trừ template literal, nên
// bỏ qua dòng có dấu huyền.
func TestFileWebKhongCoChuoiDut(t *testing.T) {
	dir := "web"
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for i, dong := range strings.Split(string(b), "\n") {
			d := strings.TrimSpace(dong)
			if strings.HasPrefix(d, "//") || strings.Contains(dong, "`") {
				continue
			}
			// bỏ ký tự đã escape trước khi đếm
			sach := strings.NewReplacer(`\'`, "", `\`, "", `\"`, "").Replace(dong)
			if strings.Count(sach, "'")%2 == 1 {
				t.Errorf("%s dòng %d: chuỗi JS đứt giữa chừng — cả script của trang sẽ chết:\n  %s",
					e.Name(), i+1, strings.TrimSpace(dong))
			}
		}
	}
}
