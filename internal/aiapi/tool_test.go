package aiapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

// ĐƯỜNG TOOL PHẢI ĐI HẾT, HAI CHIỀU — không dừng ở chỗ "kiểu có trường đó".
//
// Vì sao bài này tồn tại, và vì sao nó KHÔNG thừa dù bảng năng lực đã in ✓:
//
// Bảng `sagent nang-luc-api` dò phía dự án bằng REFLECTION trên kiểu thật
// (`nangluc.go`). Thiết kế đó tốt, nhưng nó trả lời đúng một câu: *"kiểu có
// trường đó không"*. Nó KHÔNG trả lời: *"có ai chép giá trị đi đâu không"*.
// Tức là chỉ cần thêm `Tools []Tool` vào `yeuCau` rồi không gán gì hết, ô
// `goi-tool` lập tức lật sang ✓ và không gì đỏ. Đó đúng là hình dạng lỗi đã cắn
// dự án này năm lần trong ngày 22/08 — thứ gì đó "có" ở mọi tầng trừ tầng cuối.
//
// Nên bài này đi HẾT đường, và đo ở CẢ HAI CHIỀU:
//
//	chiều đi:  ToolHam(...) → thân JSON THẬT trên dây (nhà cung cấp giả đọc lại)
//	chiều về:  tool_calls của nhà cung cấp → KetQua.ToolCalls → tên hàm + tham số
//
// Xoá dòng `Tools: tools` trong aiapi.go là chiều đi đỏ; xoá dòng
// `ToolCalls: ph.Choices[0].Message.ToolCalls` là chiều về đỏ. Trong cả hai
// trường hợp bảng năng lực vẫn in ✓.
func TestToolDiHetDuongTuDinhNghiaToiKetQua(t *testing.T) {
	var thanNhanDuoc []byte
	var soLuot int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&soLuot, 1)
		thanNhanDuoc, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		// Khuôn `tool_calls` đúng như modelapi.vn trả về (đo 22/08).
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"role":"assistant",` +
			`"content":"","tool_calls":[{"id":"call_01","type":"function","function":` +
			`{"name":"lay_gio","arguments":"{\"thanh_pho\":\"Hà Nội\"}"}}]}}],` +
			`"usage":{"prompt_tokens":90,"completion_tokens":15,"total_tokens":105}}`))
	}))
	defer srv.Close()

	tool := ToolHam("lay_gio", "Lấy giờ hiện tại ở một thành phố", map[string]any{
		"type":       "object",
		"properties": map[string]any{"thanh_pho": map[string]any{"type": "string"}},
		"required":   []any{"thanh_pho"},
	})

	kq, err := GoiTool(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, "Bây giờ là mấy giờ ở Hà Nội?", []Tool{tool}, ChonToolBatBuoc)
	if err != nil {
		t.Fatalf("GoiTool: %v", err)
	}

	// ---- CHIỀU ĐI: định nghĩa tool có THẬT SỰ lên dây không ----
	var than struct {
		Tools []struct {
			Loai string `json:"type"`
			Ham  struct {
				Ten  string         `json:"name"`
				Mo   string         `json:"description"`
				Tham map[string]any `json:"parameters"`
			} `json:"function"`
		} `json:"tools"`
		ToolChoice string `json:"tool_choice"`
	}
	if err := json.Unmarshal(thanNhanDuoc, &than); err != nil {
		t.Fatalf("thân gửi đi không đọc được: %v — %s", err, thanNhanDuoc)
	}
	if len(than.Tools) != 1 {
		t.Fatalf("thân gửi đi mang %d tool, chờ 1 — định nghĩa tool KHÔNG lên dây, "+
			"trong khi bảng năng lực vẫn in ✓ cho ô `goi-tool` vì kiểu `yeuCau` có "+
			"trường `tools`. Thân thật: %s", len(than.Tools), thanNhanDuoc)
	}
	if than.Tools[0].Loai != "function" || than.Tools[0].Ham.Ten != "lay_gio" {
		t.Errorf("tool gửi đi sai hình: type=%q name=%q", than.Tools[0].Loai, than.Tools[0].Ham.Ten)
	}
	if than.Tools[0].Ham.Mo == "" {
		t.Error("mô tả hàm bị rụng trên đường đi — model chọn tool bằng chính câu mô tả đó")
	}
	// JSON Schema phải đi NGUYÊN, không bị một lớp kiểu trung gian gọt mất.
	// Rụng `required` là model gọi tool thiếu tham số, và hỏng lúc chạy.
	if than.Tools[0].Ham.Tham["required"] == nil {
		t.Errorf("schema tham số rụng `required` trên đường đi: %v", than.Tools[0].Ham.Tham)
	}
	if than.ToolChoice != ChonToolBatBuoc {
		t.Errorf("tool_choice gửi đi = %q, chờ %q", than.ToolChoice, ChonToolBatBuoc)
	}

	// ---- CHIỀU VỀ: tool_calls có tới tay người gọi không ----
	if len(kq.ToolCalls) != 1 {
		t.Fatalf("KetQua.ToolCalls có %d phần tử, chờ 1 — nhà cung cấp trả lời đúng mà lõi "+
			"vứt đi, y hệt chuyện `reasoning_content` bị vứt suốt tới chiều 22/08", len(kq.ToolCalls))
	}
	l := kq.ToolCalls[0]
	if l.Ten() != "lay_gio" {
		t.Errorf("tên hàm = %q, chờ %q", l.Ten(), "lay_gio")
	}
	if l.ID != "call_01" {
		t.Errorf("id lời gọi = %q — mất id thì không nối được kết quả tool về đúng lời gọi", l.ID)
	}
	tham, err := l.ThamSoJSON()
	if err != nil {
		t.Fatalf("ThamSoJSON: %v", err)
	}
	if tham["thanh_pho"] != "Hà Nội" {
		t.Errorf("tham số = %v, chờ thanh_pho=\"Hà Nội\" — tham số là thứ DUY NHẤT nói tool "+
			"phải chạy với cái gì", tham)
	}

	// ---- KHÔNG CÓ VÒNG LẶP AGENT NÀO NÚP TRONG THƯ VIỆN ----
	//
	// Đây là bài canh cho lời hứa ở đầu tool.go. Nếu có ai đó thêm "chạy tool
	// hộ rồi hỏi lại" vào `GoiTool`, số lượt gọi mạng sẽ nhảy lên 2 — và một
	// vòng lặp agent nằm trong thư viện mà không ai nói ra là thứ nguy hiểm:
	// hoá đơn nhân lên mà `Usage` trả về vẫn chỉ là của một lượt.
	if n := atomic.LoadInt32(&soLuot); n != 1 {
		t.Fatalf("GoiTool chạm mạng %d lượt, chờ ĐÚNG 1 — thư viện đang tự chạy tool và "+
			"hỏi lại, tức là có nửa vòng lặp agent núp trong này. Trần vòng lặp phải "+
			"nằm ở tầng trên, chỗ khai ra được", n)
	}
	if kq.Usage.Tong != 105 {
		t.Errorf("usage = %d, chờ 105 — lượt gọi tool vẫn phải đếm được tiền", kq.Usage.Tong)
	}
}

// Lượt gọi THƯỜNG không được mang `tools` đi.
//
// Vì sao đáng một bài riêng: `Goi` là đường mà cả `internal/api`, CLI và mặt
// web đang đi. Bỏ `omitempty` là mọi lượt bỗng gửi kèm `"tools":null` — và
// deepseek-v4-flash đã trả HTTP 400 vì những thứ nhỏ hơn thế (đo 22/08, chỉ vì
// `tool_choice=required`). Hỏng kiểu đó không nổ ở đây, nó nổ trên máy người
// dùng, ở mọi lượt gọi, sau khi đã trả tiền.
func TestGoiThuongKhongMangToolsDi(t *testing.T) {
	var than string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		than = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"content":"4"}}],` +
			`"usage":{"total_tokens":3}}`))
	}))
	defer srv.Close()

	if _, err := Goi(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, "2+2"); err != nil {
		t.Fatalf("Goi: %v", err)
	}
	for _, cam := range []string{"tools", "tool_choice"} {
		if strings.Contains(than, cam) {
			t.Errorf("lượt gọi thường mang khoá %q lên dây: %s", cam, than)
		}
	}
}

// `tool_calls` rỗng phải ở lại rỗng — không bịa ra một phần tử trông như đã gọi.
func TestKhongCoToolCallThiKetQuaRong(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"content":"8 giờ sáng"}}],` +
			`"usage":{"total_tokens":20}}`))
	}))
	defer srv.Close()

	kq, err := GoiTool(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, "mấy giờ", []Tool{ToolHam("lay_gio", "lấy giờ", nil)}, ChonToolTuDo)
	if err != nil {
		t.Fatalf("GoiTool: %v", err)
	}
	if len(kq.ToolCalls) != 0 {
		t.Errorf("không có tool_calls trong phản hồi mà KetQua có %d", len(kq.ToolCalls))
	}
	if MoTaLoiGoiTool(kq) != "" {
		t.Error("MoTaLoiGoiTool nói có lời gọi tool trong khi không có cái nào")
	}
	if kq.NoiDung != "8 giờ sáng" {
		t.Errorf("câu trả lời bằng chữ bị mất: %q", kq.NoiDung)
	}
}

// Nhà cung cấp từ chối CÁCH ÉP GỌI thì lỗi phải nói ra chỗ gõ lại.
//
// Ca thật, đo 22/08: deepseek-v4-flash trả HTTP 400 "Thinking mode does not
// support this tool_choice" cho `required`, rồi chạy ngon với `auto`. Người
// nhận nguyên văn câu đó mà không có gợi ý sẽ kết luận route không gọi được
// tool — sai, và sai theo hướng đi đổi nhà cung cấp.
func TestTuChoiToolChoiceThiNoiChoGoLai(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Thinking mode does not support this tool_choice",` +
			`"request_id":"req-abc123"}}`))
	}))
	defer srv.Close()

	_, err := GoiTool(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, "mấy giờ", []Tool{ToolHam("lay_gio", "lấy giờ", nil)}, ChonToolBatBuoc)
	if err == nil {
		t.Fatal("HTTP 400 mà không có lỗi")
	}
	// Luật 2 của gói (đầu aiapi.go): nguyên văn nhà cung cấp phải còn, KÈM
	// request id — đó là thứ duy nhất dùng được khi phải đi hỏi họ.
	if !strings.Contains(err.Error(), "req-abc123") {
		t.Errorf("request id của nhà cung cấp bị nuốt: %v", err)
	}
	if !strings.Contains(err.Error(), ChonToolTuDo) {
		t.Errorf("lỗi không chỉ được chỗ gõ lại (%q): %v", ChonToolTuDo, err)
	}
	if !LoiNguoiDung(err) && !BiChanTocDo(err) {
		// 400 vẫn là lỗi máy theo phân loại của gói — chuyển route dự phòng
		// được phép thử. Chỉ khẳng định là nó KHÔNG bị xếp nhầm.
		if s := err.Error(); !strings.Contains(s, "400") {
			t.Errorf("lỗi mất mã HTTP: %v", err)
		}
	}
}

// Gọi cửa tool mà không đưa tool nào là lỗi NGƯỜI DÙNG, và phải chặn TRƯỚC khi
// chạm mạng: đi tiếp thì nhận về một câu trả lời bằng chữ, rồi kết luận nhầm là
// route không hỗ trợ tool — sau khi đã trả tiền cho lượt đó.
func TestGoiToolKhongCoToolThiChanTruocKhiChamMang(t *testing.T) {
	var chamMang bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamMang = true
	}))
	defer srv.Close()

	r := Route{Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t)}
	if _, err := GoiTool(context.Background(), r, "hỏi", nil, ""); err == nil {
		t.Error("GoiTool không tool nào mà vẫn chạy")
	} else if !LoiNguoiDung(err) {
		t.Errorf("phải là lỗi người dùng (route dự phòng không cứu được): %v", err)
	}
	if _, err := GoiTool(context.Background(), r, "hỏi",
		[]Tool{ToolHam("", "không tên", nil)}, ""); err == nil {
		t.Error("tool không tên mà vẫn gửi đi — nhà cung cấp sẽ trả 400 và tính tiền")
	}
	if _, err := GoiTool(context.Background(), r, "hỏi",
		[]Tool{ToolHam("x", "", nil)}, "bat-buoc-nhe"); err == nil {
		t.Error("tool_choice lạ mà vẫn gửi đi")
	}
	if chamMang {
		t.Error("một trong ba ca hỏng đã chạm mạng — lỗi của người gọi không được tốn tiền")
	}
}

// BÀI CANH ĐỊNH KỲ — gọi nhà cung cấp THẬT, tốn tiền thật, nên MẶC ĐỊNH BỎ QUA.
// Bật bằng:  SAGENT_E2E_TOOL=1 go test ./internal/aiapi/
//
// Theo đúng lệ của `TestE2ESuyLuanThatTuNhaCungCap`, và vì cùng một lý do: bốn
// bài trên đo với nhà cung cấp GIẢ. Chúng khẳng định "nếu máy chủ nhận `tools`
// và trả `tool_calls` thì giá trị đi hết đường" — đúng và cần, nhưng KHÔNG
// khẳng định máy chủ thật còn nhận trường tên đó.
//
// Dùng grok-4.5 chứ không phải deepseek: đo 22/08, deepseek từ chối
// `tool_choice=required` (HTTP 400 "Thinking mode does not support this
// tool_choice") còn grok chạy thẳng, trả 1 tool_call hàm "lay_gio", 479 token.
// Bài này cố ý gọi bằng `required` để đo đúng đường khó.
func TestE2EToolThatTuNhaCungCap(t *testing.T) {
	if os.Getenv("SAGENT_E2E_TOOL") == "" {
		t.Skip("bỏ qua: tốn tiền thật. Bật bằng SAGENT_E2E_TOOL=1")
	}
	kq, err := GoiTool(context.Background(), Route{
		Ten:     "grok",
		BaseURL: "https://modelapi.vn/v1",
		Model:   "grok-4.5",
		KeyID:   "grok",
	}, "Bây giờ là mấy giờ ở Hà Nội?", []Tool{ToolHam("lay_gio",
		"Lấy giờ hiện tại ở một thành phố", map[string]any{
			"type":       "object",
			"properties": map[string]any{"thanh_pho": map[string]any{"type": "string"}},
			"required":   []any{"thanh_pho"},
		})}, ChonToolBatBuoc)
	if err != nil {
		t.Fatalf("GoiTool thật: %v", err)
	}
	t.Logf("ToolCalls = %d", len(kq.ToolCalls))
	for _, l := range kq.ToolCalls {
		t.Logf("  %s(%s)", l.Ten(), l.ThamSo())
	}
	t.Logf("Usage = vào %d / ra %d / tổng %d", kq.Usage.Vao, kq.Usage.Ra, kq.Usage.Tong)
	if len(kq.ToolCalls) == 0 {
		t.Fatal("nhà cung cấp KHÔNG còn trả `tool_calls` (hoặc đã đổi tên khoá). Bảng " +
			"`sagent nang-luc-api` vẫn in ✓ cho ô này vì nó chỉ hỏi phía dự án. " +
			"Đo lại bằng `sagent nang-luc-api --do grok`.")
	}
}
