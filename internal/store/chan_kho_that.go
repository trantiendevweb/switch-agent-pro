package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// CHỐT LIÊN ĐỘNG: bài kiểm KHÔNG được mở sổ trạng thái THẬT của máy.
//
// SỰ CỐ 22/08 dựng ra chốt này. `internal/api/phientrangthai_test.go` gọi
// `New(t.TempDir())` và trông như đã cô lập — nhưng tham số đó là thư mục DỰ
// ÁN, còn sổ nằm ở `<HOME>/.ai-accounts`. Bài kiểm quên đổi HOME nên nó mở sổ
// thật. Cùng lúc, một agent đang thêm bản di trú v10 chạy `go test ./...`;
// `migrate()` nâng luôn sổ của máy lên v10, và từ giây đó mọi binary dựng từ
// `main` (v9) từ chối mở sổ. Mặt điều khiển của người vận hành chết vì một
// dòng thiếu trong một file test, và nhánh gây ra chuyện đó còn chưa trộn.
//
// `TestMain` thêm cho `internal/api` bịt đúng gói đó. Chốt này bịt phần còn
// lại: mọi gói, kể cả gói chưa ai viết. Nó nằm ở `OpenAt` vì đó là NÚT THẮT —
// mọi đường mở sổ đều đi qua đây, nên không có cửa sau nào để quên.
//
// Vì sao là LỖI chứ không phải cảnh báo: cảnh báo trôi qua trong hàng nghìn
// dòng output của `go test ./...`, và cái giá của một lần lọt là cả máy mất
// mặt điều khiển. Cùng lựa chọn với "chưa đặt mật khẩu thì dash từ chối mở
// cổng" và với chốt hạ cấp schema — thà không chạy.

// khoThat là kho hồ sơ THẬT, chụp lúc NẠP GÓI.
//
// Thời điểm quan trọng: `t.Setenv("HOME", …)` và `TestMain` đều chạy SAU khi
// gói được nạp, nên giá trị này luôn là kho thật kể cả trong một bài kiểm đã
// cô lập đúng cách. Đọc muộn hơn thì nó sẽ trỏ vào thư mục tạm và chốt tự vô
// hiệu hoá — im lặng, đúng kiểu hỏng mà chốt này sinh ra để chặn.
var khoThat = paths.AccountsRoot()

// BienChoPhep cho một bài kiểm cố ý dùng kho thật đi qua chốt.
//
// Có cửa thoát vì cấm tuyệt đối sẽ khiến người ta đi vòng bằng cách tệ hơn
// (gọi thẳng sql.Open, bỏ qua cả migrate). Cửa thoát phải LỘ RA trong mã của
// bài kiểm, ở đó người rà thấy được.
const BienChoPhep = "SAGENT_CHO_PHEP_KHO_THAT"

// duoiTest đoán xem tiến trình này có phải một binary test không.
//
// Dùng tên tiến trình chứ KHÔNG dùng testing.Testing(): hàm đó buộc gói sản
// phẩm import `testing`, kéo cờ dòng lệnh của bộ test vào mọi binary phát
// hành. Cái giá không đáng cho một phép đoán mà hậu tố `.test` đã trả lời đủ.
func duoiTest() bool {
	ten := strings.ToLower(filepath.Base(os.Args[0]))
	return strings.HasSuffix(ten, ".test") || strings.HasSuffix(ten, ".test.exe")
}

// kiemKhoThat trả lỗi nếu một binary test đang định mở sổ trong kho thật.
func kiemKhoThat(path string) error {
	if !duoiTest() || os.Getenv(BienChoPhep) != "" {
		return nil
	}
	thu, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil // không kết luận được thì không chặn — chốt này không được đoán bừa
	}
	that, err := filepath.Abs(khoThat)
	if err != nil {
		return nil
	}
	if !strings.EqualFold(thu, that) {
		return nil
	}
	return fmt.Errorf("bài kiểm đang mở SỔ TRẠNG THÁI THẬT của máy (%s) — từ chối.\n"+
		"     Bài kiểm chạm kho thật là cách mặt điều khiển của cả máy chết ngày 22/08:\n"+
		"     một nhánh chưa trộn thêm bản di trú mới, `go test ./...` nâng luôn schema\n"+
		"     của sổ thật, và mọi binary dựng từ `main` từ chối mở nó.\n"+
		"     Cách sửa: đổi HOME sang thư mục tạm TRƯỚC khi mở sổ (xem TestMain trong\n"+
		"     internal/api), hoặc gọi OpenAt() với một đường dẫn trong t.TempDir().\n"+
		"     Cố ý muốn dùng kho thật thì đặt %s=1 ngay trong bài kiểm đó.",
		path, BienChoPhep)
}
