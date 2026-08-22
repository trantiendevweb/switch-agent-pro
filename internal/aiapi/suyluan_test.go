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

// ---------------------------------------------------------------------------
// PHẦN THỨ HAI (22/08, chiều muộn): CÓ MẶT TRONG `KetQua` VẪN CHƯA LÀ TỚI NƠI.
//
// Ba bài trên canh đường từ nhà cung cấp tới `KetQua.SuyLuan`. Đường đó đã
// xanh từ trưa 22/08 — và phần suy luận VẪN không tới được người dùng, vì
// không mặt nào đọc `KetQua.SuyLuan`: không cờ CLI, không chỗ nào trên
// dashboard. Đo thật: câu trả lời 91 ký tự, phần suy luận 477 ký tự — phần bị
// vứt dài gấp 5,2 lần phần giữ lại, và người dùng đã trả tiền cho cả hai.
//
// Nhóm bài dưới đây canh nốt đoạn đường còn lại: `DocSuyLuan` là chỗ CẢ HAI
// mặt lấy chữ để hiện ra, nên nó phải nói đúng ở cả ca "có" lẫn hai ca "rỗng".
// ---------------------------------------------------------------------------

// dsThu là danh sách route như `.sagent/project.toml` khai — khớp sổ số đo, để
// `DocSuyLuan` tra được bảng năng lực mà không chạm mạng.
var dsThu = []Route{
	{Ten: "deepseek", BaseURL: "https://modelapi.vn/v1", Model: "deepseek-v4-flash", KeyID: "deepseek"},
	{Ten: "grok", BaseURL: "https://modelapi.vn/v1", Model: "grok-4.5", KeyID: "grok"},
}

// Có phần suy luận thì khối phải mang nguyên nó ra, kèm số ký tự đếm bằng RUNE.
//
// Đếm bằng byte thì "477 ký tự" tiếng Việt hiện thành ~800 — một con số vô
// nghĩa in ngay cạnh câu nói về tiền, và không ai kiểm lại được.
func TestDocSuyLuanMangNguyenPhanNghiRaNgoai(t *testing.T) {
	const nghi = "Bèo phủ gấp đôi mỗi ngày, nên ngày trước đó phủ nửa ao."
	k := DocSuyLuan(KetQua{SuyLuan: nghi, Route: "deepseek"}, dsThu)
	if !k.Co {
		t.Fatal("có phần suy luận mà khối nói không")
	}
	if k.NoiDung != nghi {
		t.Errorf("NoiDung = %q, chờ %q", k.NoiDung, nghi)
	}
	if k.SoKyTu != len([]rune(nghi)) {
		t.Errorf("SoKyTu = %d, chờ %d (đếm bằng rune, không phải byte)", k.SoKyTu, len([]rune(nghi)))
	}
	if !strings.Contains(k.DanToi, "nang-luc-api") {
		t.Errorf("không dẫn tới bảng năng lực: %q", k.DanToi)
	}
}

// RỖNG MANG HAI NGHĨA, VÀ KHỐI PHẢI NÓI RÕ ĐANG Ở NGHĨA NÀO.
//
// Đây là bài quan trọng nhất của cả tính năng. Đo 22/08: grok-4.5 qua
// modelapi.vn KHÔNG trả `reasoning_content` lẫn `reasoning`, dù lượt đó tiêu
// 951 token và mất 20,3 giây — model CÓ nghĩ, nhà bán lại không trả phần nghĩ
// ra. In một ô trống, hay tệ hơn là in "model không suy luận", biến chuyện của
// nhà cung cấp thành chuyện của model ngay trước mắt người trả tiền.
func TestSuyLuanRongPhaiNoiRoDangONghiaNao(t *testing.T) {
	for _, c := range []struct {
		ten    string
		kq     KetQua
		phaiCo []string
	}{
		{
			// Ca (b): nhà cung cấp không trả. Phải đổ đúng chỗ, và phải nhắc
			// rằng token của phần nghĩ vẫn nằm trong hoá đơn.
			"nhà cung cấp không trả phần nghĩ",
			KetQua{Route: "grok"},
			[]string{"grok", "nhà cung cấp", "KHÔNG trả", "951"},
		},
		{
			// Ca (a): route ĐÃ đo được là có trả, lượt này thì không.
			"route có trả nhưng lượt này không nghĩ",
			KetQua{Route: "deepseek"},
			[]string{"deepseek", "ĐÃ đo được", "152"},
		},
		{
			// Không tra được bảng thì phải NÓI là không tra được, chứ không
			// tiện tay kết luận thay model.
			"route lạ, không tra được bảng",
			KetQua{Route: "route-la-hoac"},
			[]string{"route-la-hoac", "CHƯA kết luận được"},
		},
	} {
		t.Run(c.ten, func(t *testing.T) {
			k := DocSuyLuan(c.kq, dsThu)
			if k.Co {
				t.Fatal("không có phần suy luận mà khối nói có")
			}
			if strings.TrimSpace(k.ViSaoRong) == "" {
				t.Fatal("ô trống KHÔNG có câu giải thích — đây chính là thứ bị đọc " +
					"thành \"model không nghĩ\"")
			}
			for _, can := range c.phaiCo {
				if !strings.Contains(k.ViSaoRong, can) {
					t.Errorf("câu giải thích thiếu %q:\n  %s", can, k.ViSaoRong)
				}
			}
			// Câu chốt của cả bài: không mặt nào được nói câu này.
			for _, cam := range []string{"model không nghĩ", "model không suy luận"} {
				if strings.Contains(strings.ToLower(k.ViSaoRong), cam) {
					t.Errorf("khối khẳng định %q — một chuyện KHÔNG đo được từ chuỗi rỗng:\n  %s",
						cam, k.ViSaoRong)
				}
			}
			// Và phải dẫn người đọc tới chỗ trả lời được câu hỏi.
			if !strings.Contains(k.DanToi, "nang-luc-api") {
				t.Errorf("không dẫn tới bảng năng lực: %q", k.DanToi)
			}
		})
	}
}

// Phần nghĩ cũng phải đi hết đường Ở ĐƯỜNG STREAM — và đó KHÔNG phải chuyện phụ:
// mặt web hỏi AI bằng stream, LUÔN LUÔN. Bỏ trống chỗ này thì mọi lượt trên
// dashboard đều không có phần nghĩ, và `DocSuyLuan` sẽ đổ tội cho nhà cung cấp
// về đúng một thứ mà chính ta không thèm đọc — một lời khai sai, tự tin, có bằng
// chứng giả.
func TestSuyLuanDiHetDuongOCaDuongStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		for _, m := range []string{
			`{"model":"m","choices":[{"delta":{"reasoning_content":"Ngày 30 phủ kín. "}}]}`,
			`{"model":"m","choices":[{"delta":{"reasoning_content":"Gấp đôi mỗi ngày."}}]}`,
			`{"model":"m","choices":[{"delta":{"content":"Ngày "}}]}`,
			`{"model":"m","choices":[{"delta":{"content":"29."}}]}`,
			`{"model":"m","choices":[],"usage":{"prompt_tokens":30,"completion_tokens":12,"total_tokens":42}}`,
		} {
			fmt.Fprintf(w, "data: %s\n\n", m)
			if fl != nil {
				fl.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	var chayRaManHinh strings.Builder
	kq, err := GoiStream(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, "ngày nào phủ nửa ao", func(s string) { chayRaManHinh.WriteString(s) })
	if err != nil {
		t.Fatalf("GoiStream: %v", err)
	}
	if kq.NoiDung != "Ngày 29." {
		t.Errorf("NoiDung = %q", kq.NoiDung)
	}
	if kq.SuyLuan != "Ngày 30 phủ kín. Gấp đôi mỗi ngày." {
		t.Fatalf("KetQua.SuyLuan = %q — đường stream vứt phần nghĩ, nên dashboard sẽ "+
			"trống trơn ở MỌI lượt", kq.SuyLuan)
	}
	// Phần nghĩ KHÔNG được chảy ra màn hình như câu trả lời: bước sau của một
	// flow nhét `{{steps.x.output}}` sang bước kế, và trộn hai thứ là đưa bản
	// nháp của model đi làm dữ liệu đầu vào.
	if strings.Contains(chayRaManHinh.String(), "Ngày 30 phủ kín") {
		t.Error("phần nghĩ chảy qua `nhan` lẫn vào câu trả lời — hai thứ này phải tách rời")
	}
	if kq.Usage.Tong != 42 {
		t.Errorf("usage = %d, chờ 42 — thêm phần nghĩ không được làm rơi sổ chi phí", kq.Usage.Tong)
	}
}
