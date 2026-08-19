package provider

import (
	"fmt"
	"strings"
)

// KetQua là kết quả một lượt chạy agent, ĐỌC TỪ DỮ LIỆU CÓ CẤU TRÚC chứ không
// đoán từ chữ tiếng Anh trong output.
//
// Vì sao cần: suốt ngày 18/08, sagent nhận biết agent hỏng bằng cách dò chuỗi —
// "no output produced", "failed to authenticate", "maximum tool execution rounds
// reached", "timeout waiting for response". Bốn chuỗi, thêm dần mỗi lần lọt lưới
// một kiểu hỏng mới. Mỗi chuỗi là chữ ký của MỘT provider ở MỘT phiên bản: họ đổi
// câu chữ là lá chắn rơi im lặng.
//
// Đội agent đọc ACP và Gas Town cùng rút ra một câu (xem docs/DU-AN-THAM-KHAO.md):
// hỏng phải là CẤU TRÚC DỮ LIỆU, không phải CHỮ TRONG VĂN BẢN.
//
// Đo ngày 18/08: `claude -p --output-format stream-json --verbose` phát ra dòng
// cuối `{"type":"result", ...}` mang đủ thứ cần — is_error, subtype, stop_reason,
// terminal_reason, permission_denials, api_error_status, usage, total_cost_usd.
type KetQua struct {
	// TraLoi là câu trả lời thật của agent (trường `result`), đã tách khỏi đống
	// sự kiện. Đây mới là thứ nên đưa cho bước sau, không phải cả bản ghi NDJSON.
	TraLoi string

	CoLoi    bool   // is_error
	Loai     string // subtype: success | error_max_turns | error_during_execution…
	DungViCo string // stop_reason: end_turn | max_tokens | …
	KetCuc   string // terminal_reason: completed | …
	LoiAPI   string // api_error_status, rỗng nếu không có
	SoLuotTu int    // num_turns
	TuChoiSo int    // số tool bị TỪ CHỐI QUYỀN (Claude: len(permission_denials))
	// ToolHong: số bước tool KẾT THÚC LỖI. Tách khỏi TuChoiSo vì không phải
	// provider nào cũng phân biệt được: Antigravity chỉ báo `state:"ERROR"` trên
	// step_update, không nói lỗi vì bị chặn quyền hay vì lệnh sai.
	ToolHong  int
	ChiPhiUSD float64
	TokenVao  int
	TokenRa   int

	// HanMucDenLai: mốc thời gian (unix) hạn mức được cấp lại, 0 nếu không rõ.
	HanMucDenLai int64

	// Ba trường dưới đây phục vụ phát hiện CHẠY QUẨN — xem quan.go.
	//
	// LenhLap là lời gọi tool bị lặp liên tiếp nhiều nhất trong lượt, SoLanLap là
	// độ dài chuỗi lặp đó. DemDuocTool nói bản ghi có mang lời gọi tool KÈM THAM
	// SỐ hay không: thiếu nó thì SoLanLap=0 KHÔNG có nghĩa "không quẩn", mà là
	// "không biết" — hai thứ đó không được nhập làm một.
	LenhLap     string
	SoLanLap    int
	DemDuocTool bool
}

// Hong trả về lý do nếu lượt chạy này KHÔNG thành công, hoặc "" nếu ổn.
//
// Đây là bản thay thế cho việc dò chuỗi: mọi kết luận đều đọc từ trường có tên,
// nên provider đổi câu chữ cũng không ảnh hưởng.
func (k KetQua) Hong() string {
	if k.CoLoi && k.LoiAPI != "" {
		return "agent báo lỗi (" + k.Loai + "): " + k.LoiAPI
	}
	// Chạy quẩn xét TRƯỚC mọi kết luận còn lại (trừ lỗi API, thứ luôn cụ thể hơn)
	// vì nó GIẢI THÍCH được những kết luận kia. Lượt quẩn thường kết thúc bằng
	// `error_max_turns` hoặc không trả lời gì; nói "hết vòng tool" thì đúng mà vô
	// dụng, nói "lặp `ls -la` 399 lần" thì người đọc biết ngay phải sửa gì.
	//
	// Cũng vì thế nó phải xét cả khi lượt chạy trông như THÀNH CÔNG: ca đo được ở
	// lần chạy #21 quẩn 399 vòng mà bước vẫn `done`. Quẩn xong vẫn nặn ra được
	// một câu trả lời là ca nguy hiểm nhất, không phải ca vô hại.
	// NGOẠI LỆ: bị chặn quyền cho MỌI tool xét trước cả quẩn.
	//
	// Một agent bị từ chối quyền sẽ gọi lại đúng một lệnh nhiều lần — nhìn từ bộ
	// đếm thì y hệt chạy quẩn. Nhưng hai chuyện này cần hai cách sửa khác hẳn:
	// quẩn thì sửa prompt hoặc công cụ, còn bị chặn quyền thì cấp quyền. Nói
	// "nghi chạy quẩn" cho một agent đang xin quyền là đẩy người đọc đi sai
	// đường, và đây là chẩn đoán CỤ THỂ HƠN nên nó phải được nói trước.
	if k.TuChoiSo > 0 && k.TraLoi == "" {
		return "agent bị từ chối quyền cho mọi tool và không trả lời được gì"
	}
	if ly, biet := k.Quan(); biet && ly != "" {
		return ly
	}
	if k.CoLoi {
		return "agent báo lỗi: " + k.lyDo()
	}
	if k.TraLoi == "" {
		if k.ToolHong > 0 {
			return fmt.Sprintf("agent không trả lời gì, %d bước tool kết thúc lỗi", k.ToolHong)
		}
		return "agent chạy xong nhưng không trả lời gì"
	}
	return ""
}

// lyDo tìm câu giải thích ĐÁNG ĐỌC nhất cho một lượt có CoLoi.
//
// Vì sao cần: `subtype` KHÔNG phải lúc nào cũng là lý do hỏng. Đo tại lần chạy
// #29 bước `code-go`: Claude trả về is_error=true nhưng subtype vẫn là
// "success", nên thông báo hoá ra "agent báo lỗi: success" — tự mâu thuẫn, và
// người đọc không biết được gì. Lý do thật nằm trong trường `result`.
//
// Thứ tự ưu tiên: lời agent (result) → terminal_reason → stop_reason → subtype.
// Bỏ qua mọi giá trị tự nhận là "success": đặt sau chữ "báo lỗi" thì nó vô
// nghĩa, mà nói vô nghĩa còn tệ hơn nói thẳng là không biết.
func (k KetQua) lyDo() string {
	if t := dongDau(k.TraLoi); t != "" {
		return t
	}
	for _, v := range []string{k.KetCuc, k.DungViCo, k.Loai} {
		if v != "" && !strings.EqualFold(v, "success") {
			return v
		}
	}
	return "không nói lý do"
}

// dongDau lấy dòng đầu của s, cắt theo RUNE cho vừa một dòng thông báo (cắt
// theo byte sẽ xé đôi ký tự tiếng Việt).
func dongDau(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	const tran = 200
	if r := []rune(s); len(r) > tran {
		return string(r[:tran]) + "…"
	}
	return s
}
