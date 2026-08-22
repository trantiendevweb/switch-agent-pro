package aiapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// GỌI TOOL — gửi định nghĩa tool đi, và đọc lại lời gọi tool model trả về.
//
// # CÂU HỎI PHẢI TRẢ LỜI TRƯỚC KHI VIẾT DÒNG ĐẦU: AI CHẠY CÁI TOOL ĐÓ?
//
// Trả lời dứt khoát: **gói này KHÔNG chạy tool, không bao giờ**. `GoiTool` gửi
// định nghĩa đi, nhận `tool_calls` về, đặt vào `KetQua.ToolCalls` rồi TRẢ VỀ
// cho người gọi. Hết. Không có vòng lặp nào ở đây.
//
// TRẦN VÒNG LẶP LÀ BAO NHIÊU: **không có vòng lặp**, nên trần bằng 0 lượt tự
// chạy và ĐÚNG 1 lượt gọi mạng cho mỗi lần gọi hàm — đúng bằng `Goi`. Ai muốn
// vòng lặp phải tự viết nó ở tầng trên, nơi trần được khai ra và nhìn thấy được.
//
// VÌ SAO KHÔNG CHẠY HỘ, dù chạy hộ thì "tiện hơn":
//
//  1. Chạy tool hộ người gọi biến thư viện này thành NỬA VÒNG LẶP AGENT. Nửa
//     vòng lặp thì bao giờ cũng có người đóng nốt nửa còn lại: gọi tool, nhét
//     kết quả vào messages, hỏi lại, model lại đòi gọi tool. Trần của cái vòng
//     đó nằm ở đâu, ai đếm, ai trả tiền cho lượt thứ tư? Một vòng lặp không có
//     trần khai ra là một hoá đơn không có trần. Cả gói này dựng lên quanh câu
//     "mọi lời gọi đều trả về Usage" — mà `Usage` của lượt nào, khi một lần gõ
//     lệnh thành sáu lượt gọi?
//
//  2. `sagent` đã có một đường chạy-thứ-gì-đó rồi: flow, với `internal/flow` và
//     bảng quyền plugin. Chỗ đó có duyệt, có tường quyền, có sổ. Cho thư viện
//     API mọc thêm một đường chạy lệnh THỨ HAI ở dưới đáy, không đi qua bảng
//     quyền nào, là mở một cửa sau cho chính dự án này.
//
//  3. Định nghĩa tool đến từ người gọi, nhưng phần "chạy" thì đến từ MODEL —
//     tên hàm và tham số là chữ do model sinh ra. Thứ đó phải bị người gọi soi,
//     không phải được một hàm thư viện lặng lẽ thi hành.
//
// NGƯỜI GỌI ĐẦU TIÊN DÙNG NÓ LÀ AI: `sagent api --tool <file.json> "câu hỏi"`
// (cmd/sagent/api.go). Lệnh đó nạp định nghĩa tool từ file JSON, gọi `GoiTool`,
// rồi IN RA lời gọi tool mà model đòi — tên hàm và tham số nguyên văn — kèm câu
// nói rõ là sagent KHÔNG chạy nó. Đó là cả đường đi, và bài
// `TestToolDiHetDuongTuDinhNghiaToiKetQua` canh đúng đường đó.
//
// Nói ra chỗ này vì một lý do cụ thể: bảng `sagent nang-luc-api` dò phía dự án
// bằng reflection trên kiểu thật, nên thêm trường `tools` vào `yeuCau` là ô
// `goi-tool` lật sang ✓ NGAY, kể cả khi không ai gửi tool đi bao giờ. Một ô
// xanh không có người dùng là đúng cái bẫy "có ở mọi tầng trừ tầng cuối" mà dự
// án này dính năm lần trong ngày 22/08.

// Ba cách ép gọi tool, đúng từ vựng của giao thức tương thích OpenAI.
//
// Hằng chứ không phải chuỗi trần ở chỗ gọi: `tool_choice` là chỗ deepseek đã
// trả HTTP 400 (đo 22/08), nên nó là thứ người ta sẽ gõ lại nhiều lần, và gõ
// lại nhiều lần thì sẽ có lần gõ "require" thiếu chữ d.
const (
	ChonToolTuDo    = "auto"     // model tự quyết gọi hay không
	ChonToolBatBuoc = "required" // buộc phải gọi ít nhất một tool
	ChonToolTat     = "none"     // cấm gọi, dù đã gửi định nghĩa
)

// Tool là MỘT định nghĩa tool gửi cho model.
//
// Hình dạng bám đúng giao thức: `{"type":"function","function":{...}}`. Không
// bọc thêm một lớp kiểu "thân thiện" rồi dịch qua lại — lớp dịch đó là chỗ tên
// trường lệch đi trong im lặng, và triệu chứng của nó là nhà cung cấp trả HTTP
// 200 kèm một câu trả lời bằng chữ, y hệt như model quyết định không gọi tool.
type Tool struct {
	Loai string  `json:"type"`
	Ham  HamTool `json:"function"`
}

// HamTool là phần mô tả hàm. `Tham` là JSON Schema, để nguyên dạng `map` vì đó
// là thứ người gọi lấy thẳng từ file JSON của họ.
type HamTool struct {
	Ten  string         `json:"name"`
	Mo   string         `json:"description,omitempty"`
	Tham map[string]any `json:"parameters,omitempty"`
}

// ToolHam dựng một Tool kiểu `function` — dạng duy nhất giao thức này có.
func ToolHam(ten, mo string, tham map[string]any) Tool {
	return Tool{Loai: "function", Ham: HamTool{Ten: ten, Mo: mo, Tham: tham}}
}

// LoiGoiTool là MỘT lời gọi tool model đòi chạy.
type LoiGoiTool struct {
	ID   string     `json:"id,omitempty"`
	Loai string     `json:"type,omitempty"`
	Ham  HamDuocGoi `json:"function"`
}

// HamDuocGoi giữ tên hàm và tham số model sinh ra.
//
// `ThamSo` là CHUỖI chứ không phải map, và đó là đúng giao thức chứ không phải
// làm biếng: nhà cung cấp trả `arguments` dưới dạng một chuỗi JSON. Khai thành
// `map[string]any` là ép `encoding/json` giải mã nó, mà model hoàn toàn có thể
// sinh ra chuỗi JSON hỏng — lúc đó CẢ phản hồi vỡ và ta mất luôn `usage` của
// một lượt đã trả tiền, chỉ vì một trường phụ. Giữ nguyên văn, ai cần thì gọi
// `ThamSoJSON` và tự xử lỗi.
type HamDuocGoi struct {
	Ten    string `json:"name"`
	ThamSo string `json:"arguments"`
}

// ThamSoJSON giải mã tham số. Lỗi trả về kèm NGUYÊN VĂN chuỗi model sinh ra —
// nếu không thì người đọc chỉ thấy "invalid character" mà không biết của cái gì.
func (l LoiGoiTool) ThamSoJSON() (map[string]any, error) {
	s := strings.TrimSpace(l.ThamSo())
	if s == "" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, fmt.Errorf("tham số của tool %q không phải JSON đọc được: %w — nguyên văn: %s",
			l.Ten(), err, s)
	}
	return m, nil
}

// Ten và ThamSo là lối tắt tới phần bên trong, để chỗ gọi không phải viết
// `l.Ham.Ham...` — cái tên đó đọc như lỗi đánh máy.
func (l LoiGoiTool) Ten() string    { return l.Ham.Ten }
func (l LoiGoiTool) ThamSo() string { return l.Ham.ThamSo }

// GoiTool gửi prompt KÈM định nghĩa tool và trả về lời gọi tool model đòi chạy.
//
// KHÔNG CHẠY TOOL. Xem đầu file để biết vì sao đó là một quyết định.
//
// `chon` nhận ChonToolTuDo / ChonToolBatBuoc / ChonToolTat, hoặc rỗng để mặc
// nhà cung cấp tự quyết. Đo 22/08 và đây là chỗ đáng nhớ nhất của cả tính năng:
// grok-4.5 nuốt `required` ngon lành, còn deepseek-v4-flash trả HTTP 400
// "Thinking mode does not support this tool_choice" — cùng một endpoint, cùng
// một dòng mã, hai kết quả. Nên hàm này KHÔNG tự đo lại bằng `auto` như bộ đo
// trong donangluc.go: bộ đo được phép tiêu thêm một lượt để có kết luận đúng,
// còn ở đây lượt thứ hai là tiền của người dùng, tiêu mà không hỏi. Thay vào
// đó, thân lỗi giữ nguyên văn và có thêm một dòng chỉ đúng chỗ gõ lại.
func GoiTool(ctx context.Context, r Route, prompt string, tools []Tool, chon string) (KetQua, error) {
	if len(tools) == 0 {
		// Không có tool mà gọi cửa này là lỗi của người gọi, và im lặng đi tiếp
		// thì họ sẽ nhận về một câu trả lời bằng chữ rồi kết luận nhầm rằng
		// route không hỗ trợ tool.
		return KetQua{}, loiNguoi(r.Ten, 0, "GoiTool không có định nghĩa tool nào — "+
			"dùng Goi nếu chỉ muốn hỏi")
	}
	for i, t := range tools {
		if strings.TrimSpace(t.Ham.Ten) == "" {
			return KetQua{}, loiNguoi(r.Ten, 0, "tool thứ %d không có tên", i+1)
		}
	}
	switch chon {
	case "", ChonToolTuDo, ChonToolBatBuoc, ChonToolTat:
	default:
		return KetQua{}, loiNguoi(r.Ten, 0, "tool_choice %q lạ — nhận %q, %q, %q hoặc rỗng",
			chon, ChonToolTuDo, ChonToolBatBuoc, ChonToolTat)
	}

	kq, err := goiThat(ctx, r, prompt, themVao{Tools: tools, ChonTool: chon})
	if err != nil && chon == ChonToolBatBuoc {
		var l *LoiAPI
		if errors.As(err, &l) && l.Status == http.StatusBadRequest &&
			strings.Contains(l.Chi, "tool_choice") {
			l.Chi += fmt.Sprintf("\n     Nhà cung cấp từ chối CÁCH ÉP GỌI, không phải trường "+
				"`tools`. Đo 22/08: deepseek-v4-flash từ chối %q nhưng chạy với %q. "+
				"Gõ lại với --tool-chon=%s", ChonToolBatBuoc, ChonToolTuDo, ChonToolTuDo)
		}
	}
	return kq, err
}

// MoTaLoiGoiTool dựng câu MỌI MẶT cùng nói khi model đòi gọi tool.
//
// Tách ra vì đúng lý do của `CanhBaoThieuUsage`: câu này mang một sự thật dễ
// hiểu nhầm — sagent KHÔNG chạy tool — nên để mỗi mặt tự chế một cách diễn đạt
// là cách chắc chắn để có một mặt quên nói vế đó.
func MoTaLoiGoiTool(k KetQua) string {
	if len(k.ToolCalls) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "model đòi gọi %d tool. sagent KHÔNG chạy chúng — đây là lời gọi "+
		"trả về nguyên văn để bên gọi tự quyết:", len(k.ToolCalls))
	for _, l := range k.ToolCalls {
		fmt.Fprintf(&b, "\n     %s(%s)", l.Ten(), l.ThamSo())
	}
	return b.String()
}
