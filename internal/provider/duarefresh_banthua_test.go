package provider

import (
	"strings"
	"testing"
)

// BẢN GHI THẬT của bản THUA trong cuộc đua N-clone cùng refresh (đo 21/08/2026,
// ô Đ5 của `docs/SO-NO-DO-LUONG.md`). Hai clone mang cùng một refresh token,
// cùng bị ép hết hạn, bật cách nhau 16ms: bản A về đích bình thường, bản B chết
// sau 186ms với dòng dưới đây. Chép nguyên văn, chỉ bỏ `session_id`/`uuid`.
//
// Hình dạng của nó là thứ đáng ghim: nhà cung cấp nói `terminal_reason` là
// "api_error" nhưng để `api_error_status` là null, còn `subtype` thì vẫn là
// "success". Ba trường nói ba chuyện khác nhau về cùng một lượt chạy.
const banGhiBanThuaCuocDuaRefresh = `{"is_error":true,"duration_api_ms":0,"num_turns":1,"stop_reason":"stop_sequence","session_id":"(bo)","total_cost_usd":0,"usage":{"output_tokens_details":{"thinking_tokens":0},"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0,"server_tool_use":{"web_search_requests":0,"web_fetch_requests":0},"service_tier":"standard","cache_creation":{"ephemeral_1h_input_tokens":0,"ephemeral_5m_input_tokens":0},"inference_geo":"","iterations":[],"speed":"standard"},"modelUsage":{},"permission_denials":[],"terminal_reason":"api_error","fast_mode_state":"off","fast_mode_disabled_reason":"sdk_opt_in_required","subtype":"success","api_error_status":null,"result":"Failed to authenticate: OAuth session expired and could not be refreshed","type":"result","duration_ms":186,"uuid":"(bo)"}`

// Bản thua PHẢI đọc ra được, và phải nói ra nguyên văn lý do. Nếu bộ đọc bỏ sót
// bản ghi này thì phiên rơi về `lost` vì "không đọc được", che mất một lỗi có
// cách sửa rất cụ thể (đăng nhập lại / đừng chạy N bản trên một tài khoản).
func TestBanThuaCuocDuaRefreshDocDuocVaNoiRaLyDo(t *testing.T) {
	k, ok := docKetQuaClaude(banGhiBanThuaCuocDuaRefresh)
	if !ok {
		t.Fatal("không đọc được bản ghi của bản thua")
	}
	if !k.CoLoi {
		t.Error("is_error=true mà đọc ra không lỗi")
	}
	if k.LoiAPI != "" {
		t.Errorf("api_error_status là null, không được bịa ra mã lỗi: %q", k.LoiAPI)
	}
	if k.KetCuc != "api_error" {
		t.Errorf("mất terminal_reason: %q", k.KetCuc)
	}
	ly := k.Hong()
	if !strings.Contains(ly, "OAuth session expired") {
		t.Errorf("không đưa được câu lỗi thật ra: %q", ly)
	}
	if strings.Contains(ly, "success") {
		t.Errorf("lấy subtype \"success\" làm lý do hỏng: %q", ly)
	}
}

// Và nó KHÔNG được đọc thành "chết, chưa rõ vì sao".
//
// Đây là chỗ đã hỏng lúc đo: `Hong()` nói đúng lý do, nhưng `PhanLoaiChet` chỉ
// xét `api_error_status` nên trả rỗng — phiên ở lại `lost`. Cái chết có lý do rõ
// nhất trong hệ thống lại hiện ra bí ẩn nhất trên cả bốn mặt điều khiển.
func TestBanThuaKhongDuocXepVaoChetChuaRoViSao(t *testing.T) {
	k, ok := docKetQuaClaude(banGhiBanThuaCuocDuaRefresh)
	if !ok {
		t.Fatal("không đọc được bản ghi của bản thua")
	}
	tt, ly, _ := PhanLoaiChet(k, ok)
	if tt == "" {
		t.Fatalf("bản ghi nói rõ terminal_reason=api_error mà vẫn không kết luận được — phiên sẽ ở lại `lost`")
	}
	if tt != ChetLoiAPI {
		t.Errorf("muốn %q, được %q", ChetLoiAPI, tt)
	}
	if !strings.Contains(ly, "OAuth session expired") {
		t.Errorf("lý do phải mang nguyên văn câu lỗi để người vận hành khỏi mở log: %q", ly)
	}
}

// Mặt trái của nhánh mới: KHÔNG được biến mọi lượt hỏng thành `failed`. Hỏng
// theo kiểu nhà cung cấp không đặt tên (chạy quẩn, không trả lời gì) vẫn phải
// trả rỗng — nhét bừa vào `failed` là quay lại đúng thói đoán mà cả gói này
// dựng lên để chặn.
func TestHongKhongCoTenVanKhongBiXepBuaVaoFailed(t *testing.T) {
	k := KetQua{CoLoi: true, Loai: "success", KetCuc: "refusal"}
	if tt, _, _ := PhanLoaiChet(k, true); tt != "" {
		t.Errorf("terminal_reason=refusal không phải lỗi API, mà xếp thành %q", tt)
	}
}
