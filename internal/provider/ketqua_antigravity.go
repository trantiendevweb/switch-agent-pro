package provider

import (
	"encoding/json"
	"strings"
)

// docKetQuaAntigravity đọc NDJSON của `agy --output-format stream-json`.
//
// Lược đồ KHÁC HẲN Claude — đó chính là lý do việc đọc kết quả phải nằm trong
// adapter chứ không phải một hàm chung:
//
//	{"event":"result","result":{"status":"SUCCESS","response":"OK\n","num_turns":1,"usage":{…}}}
//	{"event":"step_update","step_update":{"state":"ERROR","step_type":"tool","tool_name":"run_command",…}}
//
// ĐO ĐƯỢC VÀ ĐÁNG NHỚ: bị chặn quyền thì `status` VẪN LÀ "SUCCESS", chỉ có
// `response` rỗng. Nên với provider này, `status` không đủ tin — dấu hiệu thật là
// response rỗng, cộng số bước tool kết thúc ERROR.
func docKetQuaAntigravity(raw string) (KetQua, bool) {
	dong := strings.Split(raw, "\n")

	var k KetQua
	var thay bool
	for i := len(dong) - 1; i >= 0; i-- {
		d := strings.TrimSpace(dong[i])
		if !strings.HasPrefix(d, "{") || !strings.Contains(d, `"event":"result"`) {
			continue
		}
		var r struct {
			Result struct {
				Status   string `json:"status"`
				Response string `json:"response"`
				NumTurns int    `json:"num_turns"`
				Usage    struct {
					In  int `json:"input_tokens"`
					Out int `json:"output_tokens"`
				} `json:"usage"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(d), &r); err != nil {
			continue
		}
		k = KetQua{
			TraLoi:   strings.TrimSpace(r.Result.Response),
			Loai:     r.Result.Status,
			KetCuc:   r.Result.Status,
			SoLuotTu: r.Result.NumTurns,
			TokenVao: r.Result.Usage.In,
			TokenRa:  r.Result.Usage.Out,
			CoLoi:    r.Result.Status != "" && r.Result.Status != "SUCCESS",
		}
		thay = true
		break
	}
	if !thay {
		return KetQua{}, false
	}

	// KHÔNG xét chạy quẩn cho provider này, và đây là quyết định có lý do chứ
	// không phải chỗ chưa làm xong (xem quan.go).
	//
	// Bản ghi đo được chỉ mang `tool_name`, KHÔNG mang tham số của lời gọi. Đếm
	// theo mỗi tên tool thì một bước chạy `run_command` 15 lệnh KHÁC NHAU — tức
	// đang làm việc tử tế — sẽ bị kết tội y hệt một bước lặp đúng một lệnh 15
	// lần. Tên tool không phân biệt được hai ca đó, mà bắt oan một bước tốt thì
	// còn tệ hơn bỏ sót: người dùng sẽ học cách bỏ qua cảnh báo.
	//
	// Thêm một cái bẫy nữa: `step_update` phát nhiều lần cho CÙNG một bước (mỗi
	// lần đổi trạng thái), nên số dòng không phải số lời gọi.
	//
	// Nên DemDuocTool để nguyên false: KetQua.Quan() sẽ nói KHÔNG BIẾT. Khi nào
	// đo được Antigravity phát tham số ở trường nào thì bóc ra và đếm như Claude.

	// Đếm bước tool kết thúc lỗi. Antigravity không nói lỗi VÌ SAO, nên chỉ đếm
	// chứ không suy diễn nguyên nhân.
	for _, d := range dong {
		if strings.Contains(d, `"event":"step_update"`) &&
			strings.Contains(d, `"state":"ERROR"`) &&
			strings.Contains(d, `"step_type":"tool"`) {
			k.ToolHong++
		}
	}
	return k, true
}
