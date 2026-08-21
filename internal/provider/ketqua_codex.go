package provider

import (
	"encoding/json"
	"strings"
)

// Đọc kết quả có cấu trúc của Codex CLI (`codex exec --json`).
//
// ĐO THẬT 21/08/2026 trên codex-cli 0.147.0, bốn lượt chạy thật với tài khoản
// thật — bản ghi nguyên văn nằm ở ketqua_codex_test.go và docs/DO-LUONG.md.
// Trước lượt này `DocKetQua` trả thẳng `(KetQua{}, false)` kèm ghi chú "CHƯA ĐO",
// nên MỌI phiên Codex về `lost`: không is_error, không token, và lá chắn chống
// chạy quẩn (quan.go) không bao giờ chạy cho Codex.
//
// CỜ: `codex exec --json` = "Print events to stdout as JSONL". Đây là cờ có thật
// trong `codex exec --help`, không phải định dạng quan sát được như của Grok
// (xem ketqua_grok.go) — nên đây là hợp đồng, không phải may mắn.
//
// LƯỢC ĐỒ (nguyên văn đo được, lượt "trả lời đúng một từ: XONG"):
//
//	{"type":"thread.started","thread_id":"01a0230f-…"}
//	{"type":"turn.started"}
//	{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"XONG"}}
//	{"type":"turn.completed","usage":{"input_tokens":17625,"cached_input_tokens":11008,
//	  "cache_write_input_tokens":0,"output_tokens":6,"reasoning_output_tokens":0}}
//
// BỐN CÁI BẪY, cả bốn đều ĐO ĐƯỢC chứ không phải đề phòng trên giấy:
//
//  1. MỖI LỜI GỌI TOOL RA HAI DÒNG — `item.started` rồi `item.completed`, cùng
//     `item.id`. Đếm cả hai thì mọi lượt Codex có số lần lặp GẤP ĐÔI sự thật, và
//     lá chắn chạy quẩn (ngưỡng 10) sẽ vu oan cho agent chỉ lặp 5 lần. Hàm này
//     chỉ đọc `item.completed`, và đó cũng là dòng duy nhất có `exit_code` thật.
//
//  2. CÓ NHIỀU `agent_message` TRONG MỘT LƯỢT. Lượt gọi tool đo được mở đầu bằng
//     "Tôi sẽ chạy đúng lệnh hai lần như yêu cầu." rồi mới tới "DONE" ở cuối. Lấy
//     cái ĐẦU thì bước sau nhận được lời hứa thay vì kết quả. Lấy cái CUỐI.
//
//  3. DÒNG `{"type":"error"}` KHÔNG PHẢI LƯỢT HỎNG. Lượt 401 đo được in 11 dòng
//     error ("Reconnecting… 2/5") trước khi chết thật — nhưng đó là các lần THỬ
//     LẠI, và một lượt thử lại rồi THÀNH CÔNG cũng in đúng những dòng ấy. Chỉ
//     `turn.failed` mới là hỏng.
//
//  4. `turn.failed` KHÔNG CÓ `usage`. Lượt hỏng thì token về 0 vì Codex không
//     nói, không phải vì nó miễn phí.
//
// CHƯA ĐO ĐƯỢC, nói thẳng từng cái một — đây là phần dễ bị chép ẩu nhất:
//
//   - ChiPhiUSD: KHÔNG có trường giá nào trong bản ghi. Để 0, y hệt Cursor. Đừng
//     nhân token với đơn giá: đơn giá còn tuỳ model và tuỳ gói. Và vì để 0 nên
//     `ChiPhiDaDo` phải ở nguyên false: đó là cái duy nhất phân biệt số 0 này
//     với số 0 mà Claude THẬT SỰ khai ra ở một lượt hỏng sớm. Mất cờ đó thì mọi
//     mặt hiển thị đọc lượt Codex nào cũng thành "tốn 0đ".
//   - TuChoiSo: ĐÃ ĐO VÀ KHÔNG CÓ. Chạy một lượt bảo agent ghi file mà KHÔNG có
//     `--approve-for-me` (tức sandbox chỉ-đọc): bản ghi ra `turn.completed` bình
//     thường, KHÔNG một dòng nào nói tới quyền — lời từ chối chỉ nằm trong văn
//     xuôi của `agent_message` ("môi trường hiện tại bị khóa chỉ đọc"). Đọc được
//     nó thì phải dò chuỗi tiếng Việt do model tự viết, đúng thứ trangthai.go cấm.
//     Nên TuChoiSo để 0, và `ChetChanQuyen` KHÔNG BAO GIỜ được kết luận cho Codex.
//   - SoLuotTu, HanMucDenLai: không có trường tương ứng. Để 0.
//   - KetCuc: Codex không có `terminal_reason`. Bịa ra một giá trị cho nó thì
//     trangthai.go sẽ so chuỗi với enum của Claude — để rỗng.
//   - TokenVao/TokenRa lấy ĐÚNG `input_tokens`/`output_tokens` như Codex khai.
//     CHƯA ĐO được `cached_input_tokens` là phần CON hay phần THÊM của
//     `input_tokens` (cùng lý do với `reasoning_output_tokens`), nên không cộng
//     và không trừ — lấy nguyên con số nhà cung cấp dán nhãn.
func docKetQuaCodex(raw string) (KetQua, bool) {
	var d demQuan
	var k KetQua
	var xong bool // đã thấy turn.completed hoặc turn.failed chưa

	// Đọc XUÔI: chuỗi lặp liên tiếp chỉ có nghĩa khi giữ đúng thứ tự, và câu trả
	// lời là `agent_message` CUỐI CÙNG chứ không phải đầu tiên.
	for _, dong := range strings.Split(raw, "\n") {
		dong = strings.TrimSpace(dong)
		if !strings.HasPrefix(dong, "{") {
			continue // khối tiêu đề của nhật ký, chữ người đọc — bỏ qua
		}
		var e struct {
			Type string `json:"type"`
			Item struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				Command  string `json:"command"`
				ExitCode *int   `json:"exit_code"`
				Status   string `json:"status"`
			} `json:"item"`
			Usage struct {
				In  int `json:"input_tokens"`
				Out int `json:"output_tokens"`
			} `json:"usage"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(dong), &e) != nil {
			continue
		}

		switch e.Type {
		case "turn.completed":
			k.TokenVao, k.TokenRa = e.Usage.In, e.Usage.Out
			k.Loai = e.Type
			xong = true
		case "turn.failed":
			k.CoLoi = true
			k.Loai = e.Type
			// `LoiAPI` là thứ PHÂN LOẠI một cái chết, không phải ô ghi chú (xem
			// ketqua_cursor.go). Ở đây điền đúng chỗ: lượt 401 đo được mang
			// nguyên `request id: req_1552c2…`, tức thứ hỏi lại nhà cung cấp
			// được. Cắt còn một dòng vì thông báo của Codex dài và kèm cf-ray.
			k.LoiAPI = dongDau(e.Error.Message)
			xong = true
		case "item.completed":
			docItemCodex(&k, &d, e.Item.Type, e.Item.Text, e.Item.Command, e.Item.Status, e.Item.ExitCode)
		}
	}

	// KHÔNG thấy dòng kết thúc = bản ghi CỤT (tiến trình bị giết, log bị cắt).
	// Trả false để phiên ở lại `lost`: nói "agent chạy xong nhưng không trả lời
	// gì" cho một tiến trình bị giết giữa chừng là khẳng định một điều ta không
	// đo được. Cùng luật với Claude/Cursor/Antigravity — phải có dòng kết quả.
	if !xong {
		return KetQua{}, false
	}
	k.LenhLap, k.SoLanLap, k.SoLoiGoiTool, k.DemDuocTool = d.KetLuan()
	return k, true
}

// docItemCodex nạp MỘT `item.completed` vào kết quả và vào bộ đếm chạy quẩn.
//
// Tách ra vì phần quyết định nằm ở nhánh `default`: một loại item CHƯA BIẾT phải
// NGẮT chuỗi đang đếm chứ không được bỏ qua. Codex 0.147.0 còn những loại item
// lượt đo này chưa gặp (sửa file, gọi MCP…); bỏ qua chúng thì dãy A ? A ? A hoá
// thành chuỗi 3 lần liên tiếp trong khi thật ra agent đang làm việc xen kẽ — tức
// vu oan. Ngắt chuỗi thì cùng lắm là BỎ SÓT một ca quẩn thật, và bỏ sót im lặng
// vẫn hơn giết nhầm một lượt chạy lành (xem quan.go, quan_test.go).
func docItemCodex(k *KetQua, d *demQuan, loai, text, lenh, trangThai string, ma *int) {
	switch loai {
	case "agent_message":
		// Lời agent, không phải lời gọi tool: không đụng vào chuỗi đang đếm, y
		// như bộ đọc của Claude bỏ qua khối text giữa hai khối tool_use.
		if t := strings.TrimSpace(text); t != "" {
			k.TraLoi = t // cái CUỐI thắng
		}
	case "command_execution":
		// `status:"failed"` VÀ `exit_code:3` — đo được ở lượt chạy `exit 3`.
		// Xét cả hai vì hai trường này đến từ hai nguồn khác nhau: lệnh bị
		// sandbox chặn có thể có status mà không có mã thoát.
		if trangThai == "failed" || (ma != nil && *ma != 0) {
			k.ToolHong++
		}
		// chuKyTool lấy MỖI `command` làm danh tính — đúng thứ ta cần: hai lần
		// gọi `powershell -Command 'echo mot'` trong lượt đo ra đúng một chữ ký.
		d.Them(chuKyTool(loai, map[string]any{"command": lenh}))
	case "error":
		// Item lỗi TẠM THỜI (đo được: "Falling back from WebSockets to HTTPS").
		// Không phải lời gọi tool và không phải cái chết của lượt — chỉ
		// `turn.failed` mới là chết. Không đụng gì.
	default:
		d.Them("", false) // loại item chưa biết: NGẮT chuỗi, xem doc ở trên
	}
}
