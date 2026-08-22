package api

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// TestMain ĐẨY CẢ GÓI sang một HOME tạm trước khi chạy bài kiểm nào.
//
// SỰ CỐ 22/08 — vì sao có file này:
//
// `TestBoChayFlowCoDuCaHaiDuong` gọi `New(t.TempDir())` và trông như đã cô lập.
// Nhưng tham số đó là thư mục DỰ ÁN, còn sổ trạng thái nằm ở
// `paths.AccountsRoot()` = `<HOME>/.ai-accounts` — tính từ HOME, không tính từ
// tham số. Bài kiểm ấy quên gọi `homeGiaAPI(t)`, nên nó mở
// `C:\Users\Administrator\.ai-accounts\state.db` THẬT.
//
// Hậu quả không dừng ở một bài kiểm đọc nhầm file. Một agent chạy song song
// đang thêm bản di trú **v10** cho idempotency key; nó chạy `go test ./...`
// trong worktree của nó, bài kiểm này mở sổ THẬT, và `migrate()` nâng luôn
// `state.db` của máy lên v10 lúc 11:01. Từ giây đó, mọi binary dựng từ `main`
// (v9) — kể cả `sagent` đang cài — TỪ CHỐI mở sổ, đúng theo chốt hạ cấp. Toàn
// bộ mặt điều khiển của người vận hành chết vì một dòng thiếu trong một file
// test, và nhánh gây ra chuyện đó còn chưa được trộn.
//
// Chốt hạ cấp làm đúng việc của nó (từ chối, và tự sao lưu `bak-v9` trước khi
// nâng). Thứ hỏng là RANH GIỚI: một bài kiểm không được phép chạm vào trạng
// thái sản phẩm của máy đang chạy.
//
// VÌ SAO SỬA Ở ĐÂY chứ không chỉ thêm `homeGiaAPI(t)` vào đúng bài kiểm đó:
// thêm một dòng chỉ vá bài kiểm đã biết. Bài kiểm TIẾP THEO ai đó viết vẫn có
// thể quên, và lần quên sau lại là một lần cả máy chết. `TestMain` là chỗ duy
// nhất bao được cả bài kiểm chưa ai viết.
//
// Bài kiểm nào cần HOME riêng của nó vẫn gọi `homeGiaAPI(t)` như cũ — `t.Setenv`
// đè lên giá trị đặt ở đây và tự trả lại khi bài kiểm xong.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "sagent-api-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "không tạo được HOME tạm cho gói test:", err)
		os.Exit(1)
	}

	// Đặt CẢ HAI biến: Windows đọc USERPROFILE, còn lại đọc HOME. Đặt thiếu một
	// cái thì trên đúng nền tảng đó phép cô lập im lặng không có tác dụng — mà
	// im lặng đúng là cách sự cố này xảy ra lần đầu.
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tmp)
	}
	os.Setenv("HOME", tmp)

	// Kiểm NGAY rằng phép đổi có tác dụng, đừng tin là nó có. Nếu một ngày
	// paths.Home() đọc biến khác đi, bài kiểm này phải chết ồn ào ở đây chứ
	// không phải im lặng quay lại ghi vào kho thật.
	if got := homeDangDung(); got != tmp {
		fmt.Fprintf(os.Stderr, "HOME tạm KHÔNG có tác dụng: đang dùng %q, chờ %q — "+
			"dừng để không chạm vào kho hồ sơ thật\n", got, tmp)
		os.RemoveAll(tmp)
		os.Exit(1)
	}

	ma := m.Run()
	os.RemoveAll(tmp)
	os.Exit(ma)
}

// homeDangDung hỏi ĐÚNG hàm mà mã sản phẩm dùng, không tự ghép lại đường dẫn.
// Tự ghép thì bài kiểm đang xác nhận phép ghép của chính nó, không xác nhận
// thứ `paths` thật sự trả về.
func homeDangDung() string {
	return filepath.Dir(paths.AccountsRoot())
}
