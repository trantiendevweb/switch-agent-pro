package provider

import "testing"

// Bản ghi THẬT của `cursor-agent -p --output-format stream-json`, chép nguyên
// văn từ phép đo 21/08/2026 (bản 2026.08.11) — chỉ rút ngắn phần thinking.
const logCursorThat = `{"type":"system","subtype":"init","session_id":"a05639"}
{"type":"user","message":{"role":"user"}}
{"type":"thinking","text":"…"}
{"type":"assistant","message":{"role":"assistant"}}
{"type":"result","subtype":"success","duration_ms":3879,"duration_api_ms":3879,"is_error":false,` +
	`"result":"OK","session_id":"a0563938","request_id":"9d4f016c",` +
	`"usage":{"inputTokens":8445,"outputTokens":31,"cacheReadTokens":5632,"cacheWriteTokens":0}}`

func TestDocKetQuaCursorDocDuocBanGhiThat(t *testing.T) {
	k, ok := docKetQuaCursor(logCursorThat)
	if !ok {
		t.Fatal("không đọc được bản ghi THẬT của cursor-agent")
	}
	if k.TraLoi != "OK" {
		t.Errorf("câu trả lời sai: %q", k.TraLoi)
	}
	if k.CoLoi {
		t.Error("lượt thành công mà báo có lỗi")
	}
	// ĐÂY là bài quan trọng nhất: Cursor dùng camelCase (`inputTokens`), Claude
	// dùng snake_case (`input_tokens`). Chép bộ đọc của Claude sang thì token về
	// 0 — mà 0 đọc như "miễn phí", không như "chưa đọc được".
	if k.TokenVao != 8445 || k.TokenRa != 31 {
		t.Errorf("đọc sai usage camelCase: vào %d, ra %d", k.TokenVao, k.TokenRa)
	}
	// Cursor KHÔNG trả total_cost_usd. Bịa ra một con số ở đây là biến "chưa đo"
	// thành một hoá đơn.
	if k.ChiPhiUSD != 0 {
		t.Errorf("bịa chi phí trong khi Cursor không nói giá: %v", k.ChiPhiUSD)
	}
}

// `LoiAPI` là `api_error_status` — thứ PHÂN LOẠI trạng thái `failed`, không phải
// ô ghi chú. Nhét `request_id` vào đó ở mọi lượt thì một lượt hỏng bất kỳ sẽ bị
// xếp thành `failed` kèm một chuỗi vô nghĩa với người đọc.
func TestLuotThanhCongKhongMangMaLoi(t *testing.T) {
	k, _ := docKetQuaCursor(logCursorThat)
	if k.LoiAPI != "" {
		t.Errorf("lượt THÀNH CÔNG mà vẫn mang mã lỗi %q", k.LoiAPI)
	}
	if ly := k.Hong(); ly != "" {
		t.Errorf("lượt thành công bị coi là hỏng: %q", ly)
	}
}

func TestLuotHongThiGiuRequestID(t *testing.T) {
	log := `{"type":"result","subtype":"error","is_error":true,"result":"",` +
		`"request_id":"9d4f016c","usage":{"inputTokens":10,"outputTokens":0}}`
	k, ok := docKetQuaCursor(log)
	if !ok {
		t.Fatal("không đọc được bản ghi lỗi")
	}
	if !k.CoLoi {
		t.Error("is_error=true mà không đánh dấu lỗi")
	}
	// request id là thứ DUY NHẤT dùng được khi phải hỏi lại nhà cung cấp.
	if k.LoiAPI == "" {
		t.Error("lượt hỏng mà mất request id")
	}
}

// Dòng `result` KHÔNG nhất thiết là dòng cuối file. Claude 2.1.234 đã từng thêm
// một dòng `task_summary` sau nó, và bộ đọc lấy cứng dòng cuối thì mù ngay hôm
// CLI cập nhật — đo được 20/08, phiên #157 về `lost` dù chạy xong xuôi.
func TestDongResultKhongPhaiDongCuoiVanDocDuoc(t *testing.T) {
	log := logCursorThat + "\n" + `{"type":"system","subtype":"task_summary"}`
	k, ok := docKetQuaCursor(log)
	if !ok || k.TraLoi != "OK" {
		t.Errorf("có dòng lạ sau result là mất kết quả: ok=%v %+v", ok, k)
	}
}

func TestBanGhiKhongPhaiCuaCursorThiTraFalse(t *testing.T) {
	for _, raw := range []string{"", "   ", "khong phai json", `{"type":"system"}`} {
		if _, ok := docKetQuaCursor(raw); ok {
			t.Errorf("bản ghi %q không có dòng result mà vẫn báo đọc được", raw)
		}
	}
}
