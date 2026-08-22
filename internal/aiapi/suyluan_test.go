package aiapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// PHẦN SUY LUẬN PHẢI ĐI HẾT ĐƯỜNG, KHÔNG CHỈ CÓ MẶT TRONG KIỂU.
//
// Vì sao bài kiểm này tồn tại, và vì sao nó không thừa (22/08):
//
// Bảng năng lực nửa API dò phía dự án bằng REFLECTION trên kiểu thật
// (`nangluc.go`, `coTruong(reflect.TypeOf(phanHoi{}), …)`). Thiết kế đó rất
// tốt — bảng không bịa được về phía mình. Nhưng nó trả lời đúng MỘT câu: *"kiểu
// có trường đó không"*. Nó KHÔNG trả lời: *"có ai chép giá trị đi đâu không"*.
//
// Tức là thêm trường vào `tinNhan` rồi quên gán sang `KetQua` thì bảng lập tức
// in ✓ `reasoning`, và không gì đỏ. Đúng cái hình dạng đã cắn dự án này bốn lần
// trong ngày: `route.kiem`, nút Duyệt/Từ chối, `plugin.list`, `sagent help` —
// thứ gì đó "có" ở mọi tầng trừ tầng cuối cùng.
//
// Nên bài kiểm này đi HẾT đường: nhà cung cấp giả trả `reasoning_content` →
// `Goi()` → `KetQua.SuyLuan`. Xoá dòng gán trong aiapi.go là nó đỏ, trong khi
// bảng năng lực vẫn ✓.
func TestSuyLuanDiHetDuongToiKetQua(t *testing.T) {
	const suyLuan = "Bước 1: đọc câu hỏi. Bước 2: cộng hai số. Bước 3: trả lời."
	const traLoi = "4"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"model":"m","choices":[{"message":{"role":"assistant",`+
			`"content":%q,"reasoning_content":%q}}],`+
			`"usage":{"prompt_tokens":9,"completion_tokens":3,"total_tokens":12}}`,
			traLoi, suyLuan)
	}))
	defer srv.Close()

	kq, err := Goi(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, "2+2 bằng mấy")
	if err != nil {
		t.Fatalf("Goi: %v", err)
	}

	if kq.NoiDung != traLoi {
		t.Errorf("NoiDung = %q, chờ %q", kq.NoiDung, traLoi)
	}
	if kq.SuyLuan != suyLuan {
		t.Fatalf("KetQua.SuyLuan = %q — phần suy luận nhà cung cấp TRẢ VỀ bị vứt "+
			"trước khi ai nhìn thấy. Bảng `sagent nang-luc-api` sẽ vẫn in ✓ cho ô "+
			"`reasoning` vì nó chỉ hỏi kiểu có trường không, không hỏi có ai đọc "+
			"không.", kq.SuyLuan)
	}

	// Và phần nghĩ KHÔNG được lẫn vào câu trả lời: bước sau của một flow nhét
	// `{{steps.x.output}}` sang bước kế, gộp hai thứ vào nhau là đưa cả phần
	// nháp của model đi làm dữ liệu đầu vào.
	if strings.Contains(kq.NoiDung, suyLuan) {
		t.Error("phần suy luận lẫn vào NoiDung — hai thứ này phải tách rời")
	}
}

// Lượt gọi bình thường (nhà cung cấp KHÔNG trả phần nghĩ) không được hỏng, và
// KHÔNG được bịa ra một chuỗi rỗng có vẻ như đã đo.
//
// Đo 22/08: grok-4.5 qua modelapi.vn KHÔNG trả `reasoning_content` lẫn
// `reasoning`, dù lượt đó tiêu 951 token và mất 20,3 giây — model CÓ nghĩ, nhà
// bán lại không trả phần nghĩ ra. Đó là lý do `KetQua.SuyLuan` rỗng mang HAI
// nghĩa, và bảng `nang-luc-api` mới là chỗ phân biệt chúng.
func TestKhongCoSuyLuanThiVanChayBinhThuong(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"m","choices":[{"message":{"role":"assistant","content":"4"}}],`+
			`"usage":{"prompt_tokens":9,"completion_tokens":1,"total_tokens":10}}`)
	}))
	defer srv.Close()

	kq, err := Goi(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, "2+2")
	if err != nil {
		t.Fatalf("Goi: %v", err)
	}
	if kq.NoiDung != "4" {
		t.Errorf("NoiDung = %q", kq.NoiDung)
	}
	if kq.SuyLuan != "" {
		t.Errorf("KetQua.SuyLuan = %q — không có gì trong phản hồi mà vẫn ra chữ", kq.SuyLuan)
	}
}

// Trường phải mang ĐÚNG tên khoá JSON nhà cung cấp dùng. Đổi tên khoá là im
// lặng mất tính năng: không lỗi, không cảnh báo, chỉ là luôn rỗng.
func TestTenKhoaJSONCuaSuyLuanKhongDuocDoi(t *testing.T) {
	f, co := reflect.TypeOf(tinNhan{}).FieldByName("SuyLuan")
	if !co {
		t.Fatal("tinNhan không còn trường SuyLuan")
	}
	if got := f.Tag.Get("json"); got != "reasoning_content,omitempty" {
		t.Errorf("thẻ json = %q, chờ \"reasoning_content,omitempty\" — đo 22/08 thì "+
			"đó là đúng tên khoá deepseek-v4-flash trả về", got)
	}
}

// khoaThu dựng một khoá API giả trong kho tạm và trả về key_id của nó.
func khoaThu(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}
	t.Setenv("HOME", tmp)
	if err := os.MkdirAll(KeysDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(KeysDir(), "thu.key"), []byte("sk-thu"), 0o600); err != nil {
		t.Fatal(err)
	}
	return "thu"
}

// BÀI CANH ĐỊNH KỲ — gọi nhà cung cấp THẬT, tốn tiền thật, nên MẶC ĐỊNH BỎ QUA.
// Bật bằng:  SAGENT_E2E_SUYLUAN=1 go test ./internal/aiapi/
//
// VÌ SAO TỒN TẠI: ba bài kiểm ở trên đo với một nhà cung cấp GIẢ. Chúng khẳng
// định "nếu máy chủ trả `reasoning_content` thì giá trị đi tới KetQua" — một
// câu đúng và cần, nhưng KHÔNG khẳng định máy chủ thật còn trả trường tên đó.
// Đó đúng là chỗ ô C2 của Grok đã cắn một lần: định dạng ta QUAN SÁT được, chứ
// không phải hợp đồng nhà cung cấp cam kết. modelapi.vn đổi tên khoá một lần là
// `KetQua.SuyLuan` lặng lẽ rỗng mãi mãi, và bảng năng lực vẫn khoe ✓ vì phía dự
// án vẫn có trường đó.
//
// KẾT QUẢ LẦN CHẠY 22/08/2026 (route deepseek, model deepseek-v4-flash,
// endpoint https://modelapi.vn/v1) — xem docs/DO-LUONG.md mục cùng ngày.
func TestE2ESuyLuanThatTuNhaCungCap(t *testing.T) {
	if os.Getenv("SAGENT_E2E_SUYLUAN") == "" {
		t.Skip("bỏ qua: tốn tiền thật. Bật bằng SAGENT_E2E_SUYLUAN=1")
	}
	kq, err := Goi(context.Background(), Route{
		Ten:     "deepseek",
		BaseURL: "https://modelapi.vn/v1",
		Model:   "deepseek-v4-flash",
		KeyID:   "deepseek",
	}, "17 nhân 23 bằng bao nhiêu? Nghĩ từng bước rồi trả lời.")
	if err != nil {
		t.Fatalf("Goi thật: %v", err)
	}
	t.Logf("NoiDung  = %d ký tự", len(kq.NoiDung))
	t.Logf("SuyLuan  = %d ký tự", len(kq.SuyLuan))
	t.Logf("Usage    = vào %d / ra %d / tổng %d", kq.Usage.Vao, kq.Usage.Ra, kq.Usage.Tong)
	if kq.SuyLuan == "" {
		t.Fatalf("nhà cung cấp KHÔNG còn trả phần suy luận qua khoá `reasoning_content` " +
			"(hoặc đã đổi tên khoá). Bảng `sagent nang-luc-api` vẫn in ✓ cho ô này vì " +
			"nó chỉ hỏi phía dự án. Đo lại bằng `sagent nang-luc-api --do deepseek`.")
	}
}
