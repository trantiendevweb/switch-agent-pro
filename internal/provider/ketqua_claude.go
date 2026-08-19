package provider

import (
	"encoding/json"
	"strings"
)

// docKetQuaClaude đọc dòng `{"type":"result", ...}` trong bản ghi NDJSON của
// `claude -p --output-format stream-json --verbose`.
//
// Đọc từ CUỐI lên: dòng result là dòng cuối cùng của một lượt. Đọc xuôi thì gặp
// phải `session_id`/`uuid` của các sự kiện khác trước, và với bản ghi dài thì tốn
// công vô ích.
func docKetQuaClaude(raw string) (KetQua, bool) {
	dong := strings.Split(raw, "\n")
	for i := len(dong) - 1; i >= 0; i-- {
		d := strings.TrimSpace(dong[i])
		if !strings.HasPrefix(d, "{") || !strings.Contains(d, `"type":"result"`) {
			continue
		}
		var r struct {
			IsError     bool    `json:"is_error"`
			Subtype     string  `json:"subtype"`
			StopReason  string  `json:"stop_reason"`
			Terminal    string  `json:"terminal_reason"`
			APIErr      *string `json:"api_error_status"`
			NumTurns    int     `json:"num_turns"`
			Result      string  `json:"result"`
			CostUSD     float64 `json:"total_cost_usd"`
			PermDenials []any   `json:"permission_denials"`
			Usage       struct {
				In  int `json:"input_tokens"`
				Out int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(d), &r); err != nil {
			continue
		}
		k := KetQua{
			TraLoi:    strings.TrimSpace(r.Result),
			CoLoi:     r.IsError,
			Loai:      r.Subtype,
			DungViCo:  r.StopReason,
			KetCuc:    r.Terminal,
			SoLuotTu:  r.NumTurns,
			TuChoiSo:  len(r.PermDenials),
			ChiPhiUSD: r.CostUSD,
			TokenVao:  r.Usage.In,
			TokenRa:   r.Usage.Out,
		}
		if r.APIErr != nil {
			k.LoiAPI = *r.APIErr
		}
		k.HanMucDenLai = hanMucClaude(dong)
		k.LenhLap, k.SoLanLap, k.DemDuocTool = lapLaiClaude(dong)
		return k, true
	}
	return KetQua{}, false
}

// hanMucClaude nhặt mốc hạn mức được cấp lại từ sự kiện rate_limit_event.
// Có ích khi agent dừng vì hết hạn mức: nói được "quay lại lúc mấy giờ" thay vì
// chỉ báo hỏng.
func hanMucClaude(dong []string) int64 {
	for i := len(dong) - 1; i >= 0; i-- {
		d := strings.TrimSpace(dong[i])
		if !strings.Contains(d, `"rate_limit_event"`) {
			continue
		}
		var e struct {
			Info struct {
				ResetsAt int64 `json:"resetsAt"`
			} `json:"rate_limit_info"`
		}
		if json.Unmarshal([]byte(d), &e) == nil && e.Info.ResetsAt > 0 {
			return e.Info.ResetsAt
		}
	}
	return 0
}

// lapLaiClaude đếm chuỗi lời gọi tool giống hệt nhau LIÊN TIẾP dài nhất trong
// bản ghi stream-json, để phát hiện agent chạy quẩn (xem quan.go).
//
// Đọc XUÔI, khác hai hàm trên: chúng tìm MỘT dòng nên đi ngược cho nhanh, còn
// đây cần đúng THỨ TỰ các lời gọi — "liên tiếp" chỉ có nghĩa khi giữ thứ tự.
//
// Lời gọi tool nằm trong khối nội dung `{"type":"tool_use","name":…,"input":{…}}`
// của dòng assistant. Hàm KHÔNG chặn theo `type` của dòng bao ngoài: nếu Claude
// đổi vỏ mà vẫn giữ khối tool_use thì vẫn đếm được. Còn nếu chính khối tool_use
// đổi tên trường thì hàm trả docDuoc=false, và KetQua.Quan() nói KHÔNG BIẾT thay
// vì nói "không quẩn" — hỏng về phía im lặng, không hỏng về phía vu oan.
func lapLaiClaude(dong []string) (lenh string, soLan int, docDuoc bool) {
	var d demQuan
	for _, l := range dong {
		l = strings.TrimSpace(l)
		// Lọc thô cho khỏi phân tích JSON từng dòng của một bản ghi dài: khối
		// tool_use nào cũng phải mang đúng chuỗi này trong JSON.
		if !strings.HasPrefix(l, "{") || !strings.Contains(l, `"tool_use"`) {
			continue
		}
		var e struct {
			Message struct {
				Content []struct {
					Type  string         `json:"type"`
					Name  string         `json:"name"`
					Input map[string]any `json:"input"`
				} `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal([]byte(l), &e) != nil {
			continue
		}
		for _, c := range e.Message.Content {
			if c.Type != "tool_use" {
				continue
			}
			d.Them(chuKyTool(c.Name, c.Input))
		}
	}
	return d.KetLuan()
}
