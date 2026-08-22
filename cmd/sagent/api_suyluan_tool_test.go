package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
)

// TẦNG CUỐI CÙNG: CHỮ NGƯỜI DÙNG TERMINAL THẬT SỰ ĐỌC.
//
// `internal/aiapi/suyluan_test.go` canh đường từ nhà cung cấp tới
// `KetQua.SuyLuan` và tới `DocSuyLuan`. Đường đó xanh từ trưa 22/08 mà phần suy
// luận VẪN không tới được ai — vì đoạn cuối, đoạn biến dữ liệu thành chữ trên
// màn hình, không tồn tại. Nhóm bài này canh đúng đoạn đó, và nó gọi CHÍNH hàm
// dựng chữ chứ không grep mã nguồn: một bài grep vẫn xanh khi hàm được gọi
// nhưng in ra câu sai.

// dsThu là danh sách route khớp sổ số đo của bảng năng lực (không chạm mạng).
var dsThu = []aiapi.Route{
	{Ten: "deepseek", BaseURL: "https://modelapi.vn/v1", Model: "deepseek-v4-flash", KeyID: "deepseek"},
	{Ten: "grok", BaseURL: "https://modelapi.vn/v1", Model: "grok-4.5", KeyID: "grok"},
}

// Bật `--suy-luan` thì phần nghĩ phải HIỆN RA — cả khi nó nhiều dòng.
func TestCoSuyLuanInRaPhanNghi(t *testing.T) {
	const nghi = "Bước 1: đếm ngày.\nBước 2: chia đôi."
	ra := strings.Join(dongSuyLuan(
		aiapi.DocSuyLuan(aiapi.KetQua{SuyLuan: nghi, Route: "deepseek"}, dsThu), true), "\n")

	for _, can := range []string{"Bước 1: đếm ngày.", "Bước 2: chia đôi."} {
		if !strings.Contains(ra, can) {
			t.Errorf("phần nghĩ không tới được màn hình, thiếu %q:\n%s", can, ra)
		}
	}
	// Số ký tự in ra để người dùng đối chiếu với hoá đơn token.
	if !strings.Contains(ra, "ký tự") {
		t.Errorf("không nói phần nghĩ dài bao nhiêu:\n%s", ra)
	}
}

// KHÔNG bật cờ mà lượt đó CÓ phần nghĩ thì vẫn phải NHẮC MỘT DÒNG.
//
// Đây là chỗ "đừng bắt người dùng đoán". Đo 22/08: câu trả lời 91 ký tự, phần
// suy luận 477 ký tự — người dùng trả tiền cho cả hai (300 token) mà chỉ nhìn
// thấy một. Im lặng ở đây nghĩa là thứ đã mua coi như không tồn tại, và không
// có cách nào để biết tên cờ mà gõ.
func TestKhongBatCoVanNhacLaCoPhanNghiVaGoGiDeXem(t *testing.T) {
	ra := strings.Join(dongSuyLuan(
		aiapi.DocSuyLuan(aiapi.KetQua{SuyLuan: strings.Repeat("x", 477), Route: "deepseek"},
			dsThu), false), "\n")
	if ra == "" {
		t.Fatal("có 477 ký tự suy luận đã trả tiền mà terminal không nhắc một chữ nào")
	}
	if !strings.Contains(ra, "477") {
		t.Errorf("không nói dài bao nhiêu:\n%s", ra)
	}
	if !strings.Contains(ra, "--suy-luan") {
		t.Errorf("không nói gõ gì để xem — bắt người dùng đoán tên cờ:\n%s", ra)
	}

	// Lượt KHÔNG có phần nghĩ mà cũng không bật cờ thì im lặng: người dùng
	// không hỏi, và một dòng cảnh báo ở mọi lượt là một dòng bị bỏ qua.
	im := dongSuyLuan(aiapi.DocSuyLuan(aiapi.KetQua{Route: "grok"}, dsThu), false)
	if len(im) != 0 {
		t.Errorf("không hỏi mà vẫn nói: %v", im)
	}
}

// BẬT CỜ MÀ KHÔNG CÓ PHẦN NGHĨ — bài quan trọng nhất của cả VIỆC 1.
//
// Người gõ `--suy-luan` đang hỏi một câu cụ thể. Trả lời họ bằng khoảng trắng
// là để họ tự kết luận "model không nghĩ" — mà chuỗi rỗng KHÔNG nói được điều
// đó. Đo 22/08: grok-4.5 không trả phần nghĩ ra, dù lượt đó tiêu 951 token và
// mất 20,3 giây. Model nghĩ nát ra; nhà bán lại không đưa về; người dùng trả
// tiền cho phần đó.
func TestBatCoMaKhongCoPhanNghiThiKhongDuocDeHIEUNHAM(t *testing.T) {
	ra := strings.Join(dongSuyLuan(
		aiapi.DocSuyLuan(aiapi.KetQua{Route: "grok"}, dsThu), true), "\n")

	if strings.TrimSpace(ra) == "" {
		t.Fatal("gõ --suy-luan mà terminal im lặng — người dùng chỉ còn cách tự đoán, " +
			"và cách đoán tự nhiên nhất là \"model không nghĩ\", một câu KHÔNG đo được")
	}
	// Phải đổ đúng chỗ: nhà cung cấp, kèm số đo.
	for _, can := range []string{"grok", "nhà cung cấp", "951"} {
		if !strings.Contains(ra, can) {
			t.Errorf("câu giải thích thiếu %q:\n%s", can, ra)
		}
	}
	// Và phải DẪN TỚI chỗ trả lời được câu hỏi đó.
	if !strings.Contains(ra, "nang-luc-api") {
		t.Errorf("không dẫn người dùng tới bảng `sagent nang-luc-api`:\n%s", ra)
	}
	// Câu chốt: terminal KHÔNG được nói câu này.
	for _, cam := range []string{"model không nghĩ", "model không suy luận"} {
		if strings.Contains(strings.ToLower(ra), cam) {
			t.Errorf("terminal khẳng định %q — biến chuyện của nhà cung cấp thành "+
				"chuyện của model ngay trước mắt người trả tiền:\n%s", cam, ra)
		}
	}
}

// ---------------------------------------------------------------------------
// VIỆC 2 — tầng cuối của đường tool.
// ---------------------------------------------------------------------------

// Lời gọi tool phải hiện ra ĐỦ tên hàm và tham số, KÈM câu nói rõ sagent không
// chạy nó.
//
// Vế sau không phải chuyện lịch sự: người đọc thấy một lời gọi hàm in ra màn
// hình sẽ mặc định là nó đã chạy. Thư viện cố ý không chạy (đầu
// internal/aiapi/tool.go), và một quyết định như vậy mà không nói ra thì thành
// một cái bẫy — người dùng tưởng đã lấy được giờ, thật ra chưa ai gọi gì.
func TestLoiGoiToolHienRaDayDuVaNoiRoSagentKhongChay(t *testing.T) {
	kq := aiapi.KetQua{ToolCalls: []aiapi.LoiGoiTool{{
		ID: "call_01", Loai: "function",
		Ham: aiapi.HamDuocGoi{Ten: "lay_gio", ThamSo: `{"thanh_pho":"Hà Nội"}`},
	}}}
	ra := strings.Join(dongToolCall(kq), "\n")

	for _, can := range []string{"lay_gio", "thanh_pho", "Hà Nội", "call_01"} {
		if !strings.Contains(ra, can) {
			t.Errorf("lời gọi tool tới màn hình mà thiếu %q:\n%s", can, ra)
		}
	}
	if !strings.Contains(ra, "KHÔNG chạy") {
		t.Errorf("không nói ra là sagent KHÔNG chạy tool — người đọc sẽ tưởng đã chạy:\n%s", ra)
	}
}

// Tham số hỏng thì phải NÓI RA, vì đó là chữ do MODEL sinh.
func TestThamSoToolHongThiNoiRa(t *testing.T) {
	kq := aiapi.KetQua{ToolCalls: []aiapi.LoiGoiTool{{
		Ham: aiapi.HamDuocGoi{Ten: "lay_gio", ThamSo: `{"thanh_pho":`},
	}}}
	ra := strings.Join(dongToolCall(kq), "\n")
	if !strings.Contains(ra, "⚠") {
		t.Errorf("tham số JSON hỏng mà không cảnh báo:\n%s", ra)
	}
}

// Model KHÔNG đòi gọi tool cũng phải nói được, và nói luôn cách ép gọi.
func TestKhongCoToolCallThiVanNoiDuocMotCau(t *testing.T) {
	ra := strings.Join(dongToolCall(aiapi.KetQua{}), "\n")
	if strings.TrimSpace(ra) == "" {
		t.Fatal("gửi tool đi mà màn hình im lặng — không phân biệt được \"model không " +
			"gọi\" với \"sagent nuốt mất tool_call\"")
	}
	if !strings.Contains(ra, aiapi.ChonToolBatBuoc) {
		t.Errorf("không nói cách ép gọi:\n%s", ra)
	}
}

// File tool đọc theo ĐÚNG khuôn giao thức, và khuôn sai thì báo kèm ví dụ.
func TestDocFileToolTheoDungKhuonGiaoThuc(t *testing.T) {
	dir := t.TempDir()
	tot := filepath.Join(dir, "tool.json")
	if err := os.WriteFile(tot, []byte(`[{"type":"function","function":{`+
		`"name":"lay_gio","description":"Lấy giờ","parameters":{"type":"object",`+
		`"properties":{"thanh_pho":{"type":"string"}},"required":["thanh_pho"]}}}]`),
		0o600); err != nil {
		t.Fatal(err)
	}
	ds, err := docFileTool(tot)
	if err != nil {
		t.Fatalf("docFileTool: %v", err)
	}
	if len(ds) != 1 || ds[0].Ham.Ten != "lay_gio" {
		t.Fatalf("nạp sai: %+v", ds)
	}
	// Schema phải đi NGUYÊN qua chỗ nạp — rụng `required` là model gọi tool
	// thiếu tham số, và hỏng lúc chạy chứ không hỏng lúc nạp.
	if ds[0].Ham.Tham["required"] == nil {
		t.Errorf("nạp file làm rụng `required`: %+v", ds[0].Ham.Tham)
	}

	xau := filepath.Join(dir, "xau.json")
	if err := os.WriteFile(xau, []byte(`{"name":"lay_gio"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = docFileTool(xau)
	if err == nil {
		t.Fatal("khuôn sai mà vẫn nạp")
	}
	if !strings.Contains(err.Error(), "type") || !strings.Contains(err.Error(), "function") {
		t.Errorf("báo lỗi không kèm khuôn đúng, người dùng phải đoán: %v", err)
	}

	if _, err := docFileTool(filepath.Join(dir, "khong-co-that.json")); err == nil {
		t.Error("file không có mà vẫn nạp được")
	}
}

// Chọn route cho lượt `--tool`: KHÔNG đoán hộ khi thiếu khai báo.
//
// Đoán hộ ở đây là gửi tiền của người dùng tới một nhà cung cấp họ không chọn.
func TestChonRouteToolKhongDoanHo(t *testing.T) {
	if r, err := chonRouteTool(dsThu, "grok", ""); err != nil || r.Model != "grok-4.5" {
		t.Fatalf("gọi đích danh hỏng: %+v %v", r, err)
	}
	if r, err := chonRouteTool(dsThu, "", "deepseek"); err != nil || r.Ten != "deepseek" {
		t.Fatalf("không lấy default_route: %+v %v", r, err)
	}
	_, err := chonRouteTool(dsThu, "", "")
	if err == nil {
		t.Fatal("không có tên lẫn default_route mà vẫn chọn đại một route — " +
			"gửi tiền tới nhà cung cấp người dùng không chọn")
	}
	_, err = chonRouteTool(dsThu, "khong-co", "")
	if err == nil || !strings.Contains(err.Error(), "grok") {
		t.Fatalf("route lạ: lỗi phải liệt kê route đang khai, được: %v", err)
	}
}

// Cờ phải được rút TRƯỚC khi tách tên route.
//
// Bản đầu làm ngược lại: `sagent api --suy-luan grok "câu hỏi"` thì "--suy-luan"
// không khớp tên route nào, cả dãy bị coi là câu hỏi, và chữ "grok" đi thẳng
// vào prompt. Lượt gọi vẫn chạy, vẫn tính tiền, chỉ là hỏi sai câu qua sai
// route — hỏng đúng kiểu không ai để ý.
func TestRutCoTruocKhiTachTenRoute(t *testing.T) {
	args := []string{"--suy-luan", "grok", "câu", "hỏi"}
	xem, con := boolFlag(args, "--suy-luan")
	if !xem {
		t.Fatal("không rút được --suy-luan")
	}
	if len(con) != 3 || con[0] != "grok" {
		t.Fatalf("sau khi rút cờ, tên route phải nằm đầu: %v", con)
	}
	// Và cờ có tham số cũng không được ăn mất câu hỏi.
	f, con2 := strFlag([]string{"--tool", "t.json", "grok", "hỏi"}, "--tool", "")
	if f != "t.json" || len(con2) != 2 || con2[0] != "grok" {
		t.Fatalf("strFlag ăn nhầm phần còn lại: f=%q con=%v", f, con2)
	}
}
