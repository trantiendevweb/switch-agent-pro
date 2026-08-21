package provider

import "testing"

// BỐN BẢN GHI DƯỚI ĐÂY LÀ THẬT — chép nguyên văn từ stdout của
// `codex exec --json` chạy ngày 21/08/2026 trên codex-cli 0.147.0, tài khoản
// thật, máy Windows. Không cái nào được viết tay cho vừa bộ đọc. Sửa bộ đọc mà
// phải sửa cả bản ghi ở đây thì đó là dấu hiệu đang bịa, không phải đang sửa.

// Lượt đơn giản nhất: `codex exec --json --skip-git-repo-check -C . "Tra loi
// dung mot tu: XONG"`.
const logCodexXong = `{"type":"thread.started","thread_id":"01a0230f-5486-7772-b11d-950ebaba350f"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"XONG"}}
{"type":"turn.completed","usage":{"input_tokens":17625,"cached_input_tokens":11008,"cache_write_input_tokens":0,"output_tokens":6,"reasoning_output_tokens":0}}`

// Lượt GỌI TOOL: bảo agent chạy đúng một lệnh shell HAI LẦN. Đây là bản ghi
// quan trọng nhất trong file — nó là bản ghi duy nhất chứng minh được lá chắn
// chống chạy quẩn đọc đúng, và nó chứa đúng cái bẫy `item.started` +
// `item.completed` cùng một `item.id`.
const logCodexTool = `{"type":"thread.started","thread_id":"01a02310-6c1a-7f60-b0fb-1443e8eeb90a"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"Tôi sẽ chạy đúng lệnh hai lần như yêu cầu."}}
{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"\"C:\\\\Windows\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\powershell.exe\" -Command 'echo mot'","aggregated_output":"","exit_code":null,"status":"in_progress"}}
{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"\"C:\\\\Windows\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\powershell.exe\" -Command 'echo mot'","aggregated_output":"mot\r\n","exit_code":0,"status":"completed"}}
{"type":"item.started","item":{"id":"item_2","type":"command_execution","command":"\"C:\\\\Windows\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\powershell.exe\" -Command 'echo mot'","aggregated_output":"","exit_code":null,"status":"in_progress"}}
{"type":"item.completed","item":{"id":"item_2","type":"command_execution","command":"\"C:\\\\Windows\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\powershell.exe\" -Command 'echo mot'","aggregated_output":"mot\r\n","exit_code":0,"status":"completed"}}
{"type":"item.completed","item":{"id":"item_3","type":"agent_message","text":"DONE"}}
{"type":"turn.completed","usage":{"input_tokens":32425,"cached_input_tokens":26112,"cache_write_input_tokens":0,"output_tokens":121,"reasoning_output_tokens":0}}`

// Lượt TOOL HỎNG: bảo agent chạy `exit 3` đúng một lần. Đây là chỗ đo được
// `status:"failed"` đi cùng `exit_code:3`.
const logCodexToolHong = `{"type":"thread.started","thread_id":"01a02310-fdd2-7043-855a-aeb4bcd0b2fd"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"Tôi sẽ chạy đúng một lần lệnh đã cho, không sửa và không thử lại."}}
{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"\"powershell.exe\" -Command 'exit 3'","aggregated_output":"","exit_code":null,"status":"in_progress"}}
{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"\"powershell.exe\" -Command 'exit 3'","aggregated_output":"","exit_code":3,"status":"failed"}}
{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"XONG"}}
{"type":"turn.completed","usage":{"input_tokens":32353,"cached_input_tokens":26112,"cache_write_input_tokens":0,"output_tokens":76,"reasoning_output_tokens":0}}`

// Lượt CHẾT THẬT: đặt CODEX_HOME vào một thư mục rỗng rồi chạy. Đã rút bớt 9
// dòng "Reconnecting… n/5" giống hệt nhau ở giữa cho vừa màn hình; giữ lại đủ
// hai dòng để bài kiểm chứng minh chúng KHÔNG bị nhầm thành lượt hỏng.
const logCodexHong = `{"type":"thread.started","thread_id":"01a0230f-d965-7451-b559-c1a04d5ce8b2"}
{"type":"turn.started"}
{"type":"error","message":"Reconnecting... 2/5 (unexpected status 401 Unauthorized: Missing bearer or basic authentication in header, url: wss://api.openai.com/v1/responses, cf-ray: a2e7aac8c9c1105f-HKG)"}
{"type":"item.completed","item":{"id":"item_0","type":"error","message":"Falling back from WebSockets to HTTPS transport. unexpected status 401 Unauthorized: Missing bearer or basic authentication in header, url: wss://api.openai.com/v1/responses, cf-ray: a2e7ab031d959b15-HKG"}}
{"type":"error","message":"unexpected status 401 Unauthorized: Missing bearer or basic authentication in header, url: https://api.openai.com/v1/responses, cf-ray: a2e7ab421c36ddbd-HKG, request id: req_1552c246e9de4d4e8a53bf944570155a"}
{"type":"turn.failed","error":{"message":"unexpected status 401 Unauthorized: Missing bearer or basic authentication in header, url: https://api.openai.com/v1/responses, cf-ray: a2e7ab421c36ddbd-HKG, request id: req_1552c246e9de4d4e8a53bf944570155a"}}`

// Lượt BỊ SANDBOX CHẶN GHI: chạy KHÔNG có `--approve-for-me`, bảo agent tạo
// file. Bản ghi này là bằng chứng của một điều KHÔNG đo được — xem bài
// TestCodexKhongBiaSoToolBiChanQuyen.
const logCodexChanQuyen = `{"type":"thread.started","thread_id":"01a02311-071d-7e43-9dc9-8616a986514c"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"Mình sẽ tạo thu.txt trong thư mục làm việc hiện tại với đúng nội dung xin chao. rồi đọc lại để xác nhận."}}
{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"Không thể tạo thu.txt: môi trường hiện tại bị khóa **chỉ đọc**, và thao tác ghi đã bị hệ thống từ chối. Cần cấp quyền ghi cho thư mục làm việc rồi thử lại."}}
{"type":"turn.completed","usage":{"input_tokens":30781,"cached_input_tokens":26112,"cache_write_input_tokens":0,"output_tokens":209,"reasoning_output_tokens":78}}`

func TestDocKetQuaCodexDocDuocBanGhiThat(t *testing.T) {
	k, ok := docKetQuaCodex(logCodexXong)
	if !ok {
		t.Fatal("không đọc được bản ghi THẬT của `codex exec --json`")
	}
	if k.TraLoi != "XONG" {
		t.Errorf("câu trả lời sai: %q", k.TraLoi)
	}
	if k.CoLoi {
		t.Error("lượt thành công mà báo có lỗi")
	}
	// Codex dùng snake_case như Claude, NHƯNG usage nằm ở CẤP NGOÀI CÙNG của
	// dòng `turn.completed`, không nằm trong một dòng `result`. Chép thẳng bộ đọc
	// của Claude sang thì không có dòng nào khớp và docDuoc=false.
	if k.TokenVao != 17625 || k.TokenRa != 6 {
		t.Errorf("đọc sai usage: vào %d, ra %d", k.TokenVao, k.TokenRa)
	}
	// Codex KHÔNG có trường giá. Bịa một con số ở đây là biến "chưa đo" thành
	// một hoá đơn — cùng lý do với Cursor.
	if k.ChiPhiUSD != 0 {
		t.Errorf("bịa chi phí trong khi Codex không nói giá: %v", k.ChiPhiUSD)
	}
	if k.Hong() != "" {
		t.Errorf("lượt xong xuôi mà bị coi là hỏng: %s", k.Hong())
	}
}

// BÀI QUAN TRỌNG NHẤT FILE NÀY: mỗi lời gọi tool của Codex in RA HAI DÒNG
// (`item.started` rồi `item.completed`, cùng `item.id`). Đếm cả hai thì hai lần
// gọi thật hoá thành bốn — và với ngưỡng TranLapLienTiep=10, một agent lặp 5 lần
// sẽ bị vu oan là chạy quẩn. Vu oan tệ hơn bỏ sót: người vận hành mất niềm tin
// vào lá chắn rồi tắt nó đi.
func TestCodexKhongDemDoiLoiGoiTool(t *testing.T) {
	k, ok := docKetQuaCodex(logCodexTool)
	if !ok {
		t.Fatal("không đọc được bản ghi có lời gọi tool")
	}
	if !k.DemDuocTool {
		t.Fatal("bản ghi có 2 lời gọi tool mà báo không đếm được — lá chắn chạy quẩn sẽ câm")
	}
	// Bản ghi có ĐÚNG 2 lời gọi thật, in thành 4 dòng item.
	if k.SoLanLap != 2 {
		t.Errorf("đếm được %d lần lặp, bản ghi thật chỉ có 2 lời gọi "+
			"(4 dòng item vì mỗi lời gọi có cả started lẫn completed)", k.SoLanLap)
	}
	// Câu trả lời là `agent_message` CUỐI. Lấy cái đầu thì bước sau nhận được
	// "Tôi sẽ chạy…" — một lời hứa, không phải kết quả.
	if k.TraLoi != "DONE" {
		t.Errorf("lấy nhầm agent_message: %q (phải là cái cuối cùng)", k.TraLoi)
	}
}

// Chuỗi lặp 2 lần thì Quan() phải nói ĐỌC ĐƯỢC và KHÔNG QUẨN — hai chuyện khác
// nhau, và cái thứ hai chỉ có nghĩa khi cái thứ nhất đúng.
func TestCodexHaiLanGoiKhongBiKetLuanLaQuan(t *testing.T) {
	k, _ := docKetQuaCodex(logCodexTool)
	ly, biet := k.Quan()
	if !biet {
		t.Fatal("đọc được lời gọi tool mà Quan() vẫn nói không biết")
	}
	if ly != "" {
		t.Errorf("2 lần gọi mà bị kết luận chạy quẩn: %s", ly)
	}
}

// Ngưỡng chạy quẩn phải THẬT SỰ nổ khi có ca thật. Dựng bằng cách nhân đúng cặp
// dòng started/completed đo được — không viết tay lược đồ mới.
func TestCodexBatDuocChayQuan(t *testing.T) {
	const capDong = `{"type":"item.started","item":{"id":"x","type":"command_execution","command":"ls -la","exit_code":null,"status":"in_progress"}}
{"type":"item.completed","item":{"id":"x","type":"command_execution","command":"ls -la","aggregated_output":"","exit_code":0,"status":"completed"}}
`
	raw := `{"type":"turn.started"}` + "\n"
	for i := 0; i < TranLapLienTiep; i++ {
		raw += capDong
	}
	raw += `{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}`

	k, ok := docKetQuaCodex(raw)
	if !ok {
		t.Fatal("không đọc được bản ghi dựng từ đúng lược đồ đo được")
	}
	if k.SoLanLap != TranLapLienTiep {
		t.Fatalf("đếm %d, phải là %d", k.SoLanLap, TranLapLienTiep)
	}
	ly, biet := k.Quan()
	if !biet || ly == "" {
		t.Fatalf("lặp %d lần liên tiếp mà lá chắn không nổ (biet=%v, ly=%q)",
			TranLapLienTiep, biet, ly)
	}
}

// `turn.failed` là cái chết thật, và nó phải đi tới ChetLoiAPI chứ không nằm lại
// `lost`. Trước lượt đo này, đúng bản ghi 401 dưới đây cho ra "chết, chưa rõ vì
// sao" — trong khi lý do nằm nguyên văn trong log.
func TestCodexDocDuocLuotChet(t *testing.T) {
	k, ok := docKetQuaCodex(logCodexHong)
	if !ok {
		t.Fatal("không đọc được bản ghi lượt CHẾT")
	}
	if !k.CoLoi {
		t.Error("turn.failed mà không báo có lỗi")
	}
	if k.LoiAPI == "" {
		t.Error("turn.failed có error.message mà LoiAPI rỗng")
	}
	if k.Hong() == "" {
		t.Fatal("lượt chết mà Hong() nói không sao")
	}
	tt, ly, _ := PhanLoaiChet(k, ok)
	if tt != ChetLoiAPI {
		t.Errorf("lượt 401 bị xếp %q, phải là %q — lý do: %s", tt, ChetLoiAPI, ly)
	}
	// Lượt hỏng KHÔNG có usage: token phải là 0 vì Codex không nói, và đó là
	// điều đúng — không được lấy số của lượt khác lấp vào.
	if k.TokenVao != 0 || k.TokenRa != 0 {
		t.Errorf("turn.failed không mang usage mà vẫn ra token %d/%d", k.TokenVao, k.TokenRa)
	}
}

// Những dòng `{"type":"error"}` trong bản ghi 401 là các lần THỬ LẠI. Một lượt
// thử lại rồi THÀNH CÔNG in đúng những dòng ấy — coi chúng là hỏng thì mọi lượt
// mạng chập đều bị báo chết oan.
func TestCodexKhongCoiDongErrorLaLuotHong(t *testing.T) {
	raw := `{"type":"turn.started"}
{"type":"error","message":"Reconnecting... 2/5 (unexpected status 401 Unauthorized)"}
{"type":"item.completed","item":{"id":"item_0","type":"error","message":"Falling back from WebSockets to HTTPS transport."}}
{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"XONG"}}
{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":2}}`
	k, ok := docKetQuaCodex(raw)
	if !ok {
		t.Fatal("không đọc được")
	}
	if k.CoLoi {
		t.Error("lượt thử lại rồi THÀNH CÔNG mà bị báo có lỗi")
	}
	if ly := k.Hong(); ly != "" {
		t.Errorf("lượt thành công sau khi thử lại bị coi là hỏng: %s", ly)
	}
}

// `status:"failed"` + `exit_code:3` — đo được, nên đếm được.
func TestCodexDemDuocToolHong(t *testing.T) {
	k, ok := docKetQuaCodex(logCodexToolHong)
	if !ok {
		t.Fatal("không đọc được")
	}
	if k.ToolHong != 1 {
		t.Errorf("đếm %d bước tool hỏng, bản ghi thật có đúng 1 (exit_code:3, status:failed)", k.ToolHong)
	}
	// Tool hỏng nhưng agent VẪN trả lời: đây không phải lượt hỏng. Hong() chỉ
	// nhắc tới ToolHong khi agent không nặn ra được câu trả lời nào.
	if k.TraLoi != "XONG" {
		t.Errorf("câu trả lời sai: %q", k.TraLoi)
	}
	if ly := k.Hong(); ly != "" {
		t.Errorf("một lệnh hỏng mà agent vẫn trả lời thì lượt KHÔNG hỏng, được: %s", ly)
	}
}

// ĐO ĐƯỢC MỘT ĐIỀU KHÔNG ĐO ĐƯỢC: Codex bị sandbox chặn ghi thì bản ghi ra
// `turn.completed` BÌNH THƯỜNG — không một trường nào nói tới quyền. Lời từ chối
// chỉ nằm trong văn xuôi tiếng Việt do model tự viết.
//
// Nên `TuChoiSo` phải là 0. Đây KHÔNG phải chỗ chưa làm xong: đọc được nó thì
// phải dò chuỗi trong câu chữ của model, đúng thứ trangthai.go cấm ("LUẬT: KHÔNG
// DÒ CHUỖI"). Hệ quả phải nói thẳng: `ChetChanQuyen` KHÔNG BAO GIỜ kết luận được
// cho Codex, và bài này giữ cho người sau không lặng lẽ nhét một phép dò chuỗi
// vào đó.
func TestCodexKhongBiaSoToolBiChanQuyen(t *testing.T) {
	k, ok := docKetQuaCodex(logCodexChanQuyen)
	if !ok {
		t.Fatal("không đọc được bản ghi lượt bị chặn ghi")
	}
	if k.TuChoiSo != 0 {
		t.Errorf("bịa ra %d tool bị chặn quyền — bản ghi thật KHÔNG có trường nào "+
			"nói về quyền, chỉ có văn xuôi trong agent_message", k.TuChoiSo)
	}
	if tt, _, _ := PhanLoaiChet(k, ok); tt == ChetChanQuyen {
		t.Error("kết luận ChetChanQuyen cho Codex — không có dữ liệu nào đỡ được kết luận đó")
	}
}

// Bản ghi CỤT (tiến trình bị giết giữa chừng) phải trả docDuoc=false. Nói "agent
// chạy xong nhưng không trả lời gì" cho một tiến trình bị giết là khẳng định một
// điều không đo được; để nó ở lại `lost` mới là trung thực.
func TestCodexBanGhiCutThiKhongKetLuan(t *testing.T) {
	cut := `{"type":"thread.started","thread_id":"01a0230f"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"đang làm"}}`
	if _, ok := docKetQuaCodex(cut); ok {
		t.Error("bản ghi thiếu turn.completed/turn.failed mà vẫn kết luận")
	}
	if _, ok := docKetQuaCodex(""); ok {
		t.Error("bản ghi rỗng mà vẫn kết luận")
	}
}

// Loại item CHƯA BIẾT phải NGẮT chuỗi đang đếm. Codex còn những loại item lượt
// đo này chưa gặp (sửa file, gọi MCP…); bỏ qua chúng thì dãy A ? A ? A hoá thành
// chuỗi 3 liên tiếp trong khi agent đang làm việc xen kẽ — vu oan.
func TestCodexItemLaNgatChuoiLap(t *testing.T) {
	const lenh = `{"type":"item.completed","item":{"id":"a","type":"command_execution","command":"ls","exit_code":0,"status":"completed"}}`
	const la = `{"type":"item.completed","item":{"id":"b","type":"file_change","status":"completed"}}`
	raw := `{"type":"turn.started"}
` + lenh + "\n" + la + "\n" + lenh + "\n" + la + "\n" + lenh + `
{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}`
	k, ok := docKetQuaCodex(raw)
	if !ok {
		t.Fatal("không đọc được")
	}
	if k.SoLanLap != 1 {
		t.Errorf("chuỗi dài nhất là %d — item lạ xen giữa phải NGẮT chuỗi, "+
			"không được gộp 3 lần `ls` cách nhau thành một chuỗi", k.SoLanLap)
	}
}

// Cờ `--json` phải nằm trong HeadlessArgs, vì đó là chỗ `CoConThieu` hỏi để bổ
// sung cho đường `sagent fleet` (truyền args thô, không qua adapter). Thiếu nó
// thì Codex in chữ cho người đọc và mọi phiên fleet lại rơi về `lost` — đúng cái
// lỗi đã đo 20/08 với 20 phiên liền.
func TestCodexHeadlessCoCoJSON(t *testing.T) {
	var coJSON bool
	for _, a := range (codex{}).HeadlessArgs("việc") {
		if a == "--json" {
			coJSON = true
		}
	}
	if !coJSON {
		t.Fatal("HeadlessArgs của codex thiếu --json")
	}
	var them string
	for _, x := range CoConThieu(codex{}, []string{"exec", "việc"}) {
		them += x + " "
	}
	if them == "" {
		t.Error("CoConThieu không bổ sung --json cho đường fleet")
	}
}
