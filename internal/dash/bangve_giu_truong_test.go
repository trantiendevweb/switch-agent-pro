package dash

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// BẢNG VẼ WORKFLOW KHÔNG ĐƯỢC XOÁ TRẮNG TRƯỜNG NÓ KHÔNG BIẾT VẼ.
//
// SỰ CỐ 22/08 — vì sao có bài kiểm này. Hàm lưu của `flow.html` DỰNG LẠI mỗi
// bước từ đầu, chỉ chép đúng những trường nó biết:
//
//	const s={id, type, needs, x, y, timeout_sec, retry, on_failure}
//
// Mở một flow trên bảng vẽ rồi bấm Lưu — DÙ KHÔNG SỬA GÌ — là xoá vĩnh viễn
// khỏi `flows.toml`: doc_duoc · phai_co · vai_tro · model · plugin · vao ·
// tham_so · route · separator · fallback · artifact · idempotent · compensate.
// Nặng hơn: `profile`, `when`, `foreach` được `loadFlow` ĐỌC VÀO nhưng hàm lưu
// không ghi ra, nên chúng cũng bốc hơi.
//
// `phai_co` và `doc_duoc` là CỔNG AN TOÀN của flow. Một cái nút Lưu gỡ cổng an
// toàn ra mà không nói gì là mất dữ liệu im lặng: file trên đĩa đổi, không
// cảnh báo, người dùng chỉ phát hiện khi lượt chạy sau cư xử khác.
//
// Cách vá là trải bản gốc ra trước rồi mới đè (`{...(n.__goc||{}), id, ...}`),
// nên bài kiểm này canh đúng hai mắt xích của phép trải đó. Bài kiểm KHÔNG cố
// liệt kê từng tên trường: danh sách chép tay sẽ lệch với `flow.Step` đúng vào
// lần thêm trường tiếp theo — mà lần đó mới là lần nguy hiểm.
func TestBangVeGiuTruongNoKhongBietVe(t *testing.T) {
	than := docTrangFlow(t)

	// Mắt 1: loadFlow phải GIỮ bản gốc lại trên node.
	if !strings.Contains(than, "__goc:s") {
		t.Error("loadFlow không giữ bản gốc của bước (`__goc:s`) — hàm lưu sẽ không " +
			"có gì để trải ra, và mọi trường bảng vẽ không biết vẽ sẽ bị xoá khi bấm Lưu")
	}
	// Mắt 2: hàm lưu phải TRẢI bản gốc đó ra.
	if !strings.Contains(than, "...(n.__goc") {
		t.Error("hàm lưu không trải bản gốc (`...(n.__goc||{})`) mà dựng lại bước từ " +
			"đầu — bấm Lưu là xoá trắng doc_duoc, phai_co, vai_tro, model, plugin… " +
			"Đây là mất dữ liệu im lặng, không phải thiếu tính năng")
	}

	// Và giải thích VÌ SAO phép trải là bắt buộc, bằng số chứ không bằng lời:
	// đếm xem `flow.Step` có bao nhiêu trường mà hàm lưu KHÔNG hề nhắc tên.
	// Còn dù chỉ một trường như vậy thì dựng-lại-từ-đầu là mất dữ liệu.
	var khongNhac []string
	for _, ten := range truongCuaStep() {
		if !strings.Contains(than, ten) {
			khongNhac = append(khongNhac, ten)
		}
	}
	if len(khongNhac) == 0 {
		t.Log("flow.html hiện nhắc tên MỌI trường của flow.Step — phép trải vẫn nên " +
			"giữ, nhưng bài kiểm này tạm thời không chứng minh được nó bắt buộc")
	} else {
		t.Logf("%d trường của flow.Step không hề xuất hiện trong flow.html "+
			"(%s) — đây chính là những thứ phép trải đang cứu",
			len(khongNhac), strings.Join(khongNhac, ", "))
	}
}

// truongCuaStep đọc tên trường từ THẺ json của flow.Step, không chép tay.
//
// Dùng thẻ json chứ không phải thẻ toml: bảng vẽ nói chuyện với máy chủ bằng
// JSON, nên đó mới là tên nó nhìn thấy.
func truongCuaStep() []string {
	tt := reflect.TypeOf(flow.Step{})
	var ra []string
	for i := 0; i < tt.NumField(); i++ {
		the := tt.Field(i).Tag.Get("json")
		if the == "" || the == "-" {
			continue
		}
		ten := strings.Split(the, ",")[0]
		if ten != "" {
			ra = append(ra, ten)
		}
	}
	return ra
}

func docTrangFlow(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("web", "flow.html"))
	if err != nil {
		t.Fatalf("không đọc được bảng vẽ workflow: %v", err)
	}
	return string(b)
}
