package dash

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// docTep đọc một file trong gói này (đường dẫn tương đối với internal/dash).
func docTep(t *testing.T, ten string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(ten))
	if err != nil {
		t.Fatalf("không đọc được %s: %v", ten, err)
	}
	return string(b)
}

// LỖI THẬT, ảnh chụp 20/08/2026: Phòng review (vai `soi`) TRỐNG TRƠN, không có
// một nhân vật nào — trong khi bước `soi` của lượt #47 đã chạy xong.
//
// Nguyên nhân: bước `soi` là node `model` (đi thẳng API vì CLI grok hỏng vĩnh
// viễn — HTTP 410). Node `model` KHÔNG có `profile`, mà mặt Trung tâm thì
// `if(!st.profile) return;` — bỏ qua sạch. Nhìn vào tưởng chưa ai soi.
//
// Người đại diện của node `model` là ROUTE. Ba bài dưới đây giữ đúng chỗ đó.

func TestNodeModelCoNguoiDaiDien(t *testing.T) {
	ma := boComment(docTrungTam(t))
	if !strings.Contains(ma, "laModel(") {
		t.Fatal("trung-tam.html: không phân biệt node `model` — nó vẫn rơi vào nhánh 'không có profile thì bỏ qua'")
	}
	// Và KHÔNG được gộp vào bước máy: có một AI thật làm việc đó, chỉ là nó đi
	// thẳng API thay vì qua CLI. Vẽ thành máy chấm là nói dối theo hướng ngược lại.
	if regexp.MustCompile(`LOAI_MAY\s*=\s*\[[^\]]*'model'`).MatchString(ma) {
		t.Error("trung-tam.html: coi node `model` là bước máy — bảo rằng không ai làm, trong khi có hẳn một nhà cung cấp")
	}
	if !regexp.MustCompile(`khoa\s*=\s*'route:'`).MatchString(ma) {
		t.Error("trung-tam.html: nhân vật của node `model` không khoá theo route — hai bước cùng route sẽ thành hai người")
	}
}

// Route dự phòng phải hiện, VÀ phải nói rõ là dự phòng.
//
// Giấu đi thì lúc route chính sập, tự nhiên có một nhân vật lạ xuất hiện mà
// không ai hiểu ở đâu ra. Còn hiện mà không phân biệt thì hai người cùng đứng
// một phòng đọc như hai người đang cùng làm.
func TestRouteDuPhongHienVaNoiRoLaDuPhong(t *testing.T) {
	ma := boComment(docTrungTam(t))
	if !strings.Contains(ma, "duPhong") {
		t.Fatal("trung-tam.html: không đánh dấu route dự phòng")
	}
	if !strings.Contains(ma, "dự phòng") {
		t.Error("trung-tam.html: có cờ dự phòng nhưng không hiện chữ nào — người xem không phân biệt được")
	}
}

// Nhãn phải nói rõ đây là ROUTE, không phải tài khoản: "deepseek" đứng một
// mình đọc như tên một hồ sơ đã đăng nhập.
func TestNhanNoiRoLaRoute(t *testing.T) {
	ma := boComment(docTrungTam(t))
	if !strings.Contains(ma, "'route · '") {
		t.Error("trung-tam.html: nhãn không phân biệt route với tài khoản")
	}
}

// Server phải GIẢI SẴN danh sách route. `route` rỗng trong flows.toml nghĩa là
// default_route rồi tới dự phòng — luật đó nằm ở `api.ThuTuRoute`, và bắt mỗi
// mặt web tự suy lại là cách để bốn mặt nói bốn kiểu về cùng một bước.
func TestDTOBuocMangRoute(t *testing.T) {
	b := docTep(t, "flow_api.go")
	if !strings.Contains(b, "`json:\"route,omitempty\"`") {
		t.Fatal("flow_api.go: DTO bước không mang route — mặt web không biết ai đứng sau node `model`")
	}
	if !strings.Contains(b, "ThuTuRoute(") {
		t.Error("flow_api.go: không giải sẵn thứ tự route — mặt web sẽ phải tự đoán default_route")
	}
	// Chỉ node `model` mới có route: gắn route cho bước agent là nói sai về
	// đường nó thật sự đi.
	if !regexp.MustCompile(`d\.Type\s*==\s*flow\.TypeModel`).MatchString(b) {
		t.Error("flow_api.go: không giới hạn route cho đúng node `model`")
	}
}

// Sắc hãng của deepseek phải có trong token.css, ở CẢ HAI bảng màu. Thiếu thì
// nhân vật rơi về xám "hãng lạ" — mà deepseek không hề lạ, nó là route mặc định.
func TestDeepseekCoSacHang(t *testing.T) {
	css := docTep(t, "web/vendor/token.css")
	if n := strings.Count(css, "--prov-deepseek:"); n != 2 {
		t.Errorf("token.css khai --prov-deepseek %d lần, muốn 2 (bảng tối + bảng sáng)", n)
	}
}
