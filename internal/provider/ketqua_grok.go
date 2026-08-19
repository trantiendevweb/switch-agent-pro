package provider

import (
	"encoding/json"
	"strings"
)

// Đọc kết quả có cấu trúc của Grok CLI.
//
// VÌ SAO ĐÁNG LÀM, VÀ VÌ SAO LÀ VIỆC GẤP:
// lá chắn chống chạy quẩn (quan.go) được viết ra vì lần chạy #21 — Grok gọi
// `ls -la` 399 lần liên tiếp. Nhưng nó chỉ chạy được khi provider đọc được dữ
// liệu CÓ CẤU TRÚC, mà `grok.DocKetQua` lại trả về `false`. Nghĩa là lá chắn
// KHÔNG BAO GIỜ chạy cho đúng kẻ đã gây ra nó. Đọc commit và tài liệu thì tưởng
// đã bịt; đo thì chưa. Người soi độc lập bắt được chỗ này.
//
// Chuyện thứ hai, cũng đo được ở lần chạy #29: vì không đọc được, output của
// bước `soi` được giữ nguyên là khối NDJSON thô — cả prompt, cả từng lời gọi
// tool, cả nội dung file dài. Nó vừa khó đọc vừa bị nhét nguyên vào prompt của
// bước sau. Đọc được cấu trúc thì bước sau chỉ nhận CÂU TRẢ LỜI.
//
// ĐỊNH DẠNG (nguyên văn đo được ở lần chạy #29, token đã thay bằng chữ giả):
//
//	{"role":"user","content":"..."}
//	{"role":"assistant","content":"Using tools to help you...","tool_calls":[
//	   {"id":"call-1","type":"function",
//	    "function":{"name":"bash","arguments":"{\"command\":\"git log --oneline\"}"}}]}
//	{"role":"tool","tool_call_id":"call-1","content":"..."}
//	{"role":"assistant","content":"TNS: NEN TRON ..."}
//
// Lưu ý `arguments` là một CHUỖI chứa JSON, phải bóc hai lớp.
//
// CHƯA ĐO ĐƯỢC, nói thẳng: Grok không có cờ nào bảo nó xuất JSON (xem
// HeadlessArgs ở grok.go) — định dạng trên là thứ nó tự in ra và ta quan sát
// được, không phải hợp đồng nó cam kết. Nó đổi cách in là hàm này trả
// docDuoc=false, và mọi kết luận quay về "không biết" chứ không đoán bừa.
// Không có trường chi phí/token nào trong bản ghi, nên ChiPhiUSD và token để 0.
func docKetQuaGrok(raw string) (KetQua, bool) {
	var d demQuan
	var traLoi string
	coDong := false

	for _, dong := range strings.Split(raw, "\n") {
		dong = strings.TrimSpace(dong)
		if !strings.HasPrefix(dong, "{") {
			continue
		}
		var m struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				Function struct {
					Name string `json:"name"`
					Args string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		}
		if json.Unmarshal([]byte(dong), &m) != nil {
			continue
		}
		coDong = true

		for _, tc := range m.ToolCalls {
			var thamSo map[string]any
			if json.Unmarshal([]byte(tc.Function.Args), &thamSo) != nil {
				// Bóc không nổi tham số: KHÔNG kết luận, và ngắt chuỗi đang đếm.
				d.Them("", false)
				continue
			}
			d.Them(chuKyTool(tc.Function.Name, thamSo))
		}

		// Câu trả lời THẬT là lời assistant CUỐI CÙNG không kèm lời gọi tool.
		// Những dòng có tool_calls chỉ là câu dẫn ("Using tools to help you…"),
		// lấy nhầm nó thì bước sau nhận được một câu vô nghĩa.
		if m.Role == "assistant" && len(m.ToolCalls) == 0 && strings.TrimSpace(m.Content) != "" {
			traLoi = strings.TrimSpace(m.Content)
		}
	}

	if !coDong {
		return KetQua{}, false
	}
	lenh, soLan, demDuoc := d.KetLuan()
	return KetQua{
		TraLoi:      traLoi,
		DemDuocTool: demDuoc,
		LenhLap:     lenh,
		SoLanLap:    soLan,
	}, true
}
