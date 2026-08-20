package provider

import (
	"encoding/json"
	"strings"
)

// docKetQuaCursor đọc bản ghi NDJSON của `cursor-agent -p --output-format stream-json`.
//
// ĐO THẬT 21/08/2026 trên bản 2026.08.11: một lượt ngắn ra 7 dòng JSON —
// system, user, thinking ×3, assistant, và dòng cuối:
//
//	{"type":"result","subtype":"success","duration_ms":3879,"is_error":false,
//	 "result":"OK","session_id":"…","request_id":"…",
//	 "usage":{"inputTokens":8445,"outputTokens":31,"cacheReadTokens":5632,"cacheWriteTokens":0}}
//
// HAI KHÁC BIỆT với Claude, và cả hai đều đủ để làm hỏng nếu chép nhầm bộ đọc:
//
//  1. `usage` dùng camelCase (`inputTokens`) chứ không phải snake_case
//     (`input_tokens`). Chép bộ đọc của Claude sang thì token về 0 — mà 0 đọc
//     như "miễn phí", không như "chưa đọc được".
//  2. KHÔNG có `total_cost_usd`. Cursor không nói giá, nên chi phí ở đây phải để
//     0 và vẫn là CHƯA ĐO. Đừng suy ra tiền từ số token: đơn giá còn tuỳ model
//     và tuỳ gói.
//
// Quét NGƯỢC từ cuối như bộ đọc của Claude: dòng `result` không nhất thiết là
// dòng cuối cùng của file — Claude 2.1.234 đã từng thêm một dòng `task_summary`
// sau nó, và bộ đọc lấy cứng dòng cuối thì mù ngay hôm CLI cập nhật.
func docKetQuaCursor(raw string) (KetQua, bool) {
	dong := strings.Split(raw, "\n")
	for i := len(dong) - 1; i >= 0; i-- {
		d := strings.TrimSpace(dong[i])
		if !strings.HasPrefix(d, "{") || !strings.Contains(d, `"type":"result"`) {
			continue
		}
		var r struct {
			IsError bool   `json:"is_error"`
			Subtype string `json:"subtype"`
			Result  string `json:"result"`
			ReqID   string `json:"request_id"`
			Usage   struct {
				In  int `json:"inputTokens"`
				Out int `json:"outputTokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(d), &r); err != nil {
			continue
		}
		k := KetQua{
			TraLoi:   strings.TrimSpace(r.Result),
			CoLoi:    r.IsError,
			Loai:     r.Subtype,
			TokenVao: r.Usage.In,
			TokenRa:  r.Usage.Out,
			// ChiPhiUSD để 0: Cursor KHÔNG trả total_cost_usd. Đây là chưa đo
			// được, không phải miễn phí — và mặt nào hiện số 0 thì phải hiện kèm
			// chữ "chưa đo" như mặt 2D đang làm.
		}
		// `LoiAPI` là `api_error_status` — nó là THỨ PHÂN LOẠI `failed`, không
		// phải một ô ghi chú. Nhét `request_id` vào đó thì mọi lượt Cursor đều
		// mang một "mã lỗi", và một lượt hỏng bất kỳ sẽ bị `PhanLoaiChet` xếp
		// thành `failed` kèm một chuỗi vô nghĩa với người đọc. Chỉ điền khi
		// THẬT SỰ có lỗi, và điền cái dùng được để hỏi lại nhà cung cấp.
		if r.IsError && r.ReqID != "" {
			k.LoiAPI = "request_id " + r.ReqID
		}
		return k, true
	}
	return KetQua{}, false
}
