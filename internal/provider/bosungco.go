package provider

import "strings"

// TrangThaiCua trả về trạng thái một năng lực của adapter, hoặc ChuaDo nếu
// adapter không khai gì về nó.
//
// Không khai = CHƯA ĐO, không phải "không có": `KiemNangLuc` đã bắt mọi adapter
// phải khai đủ, nên rơi vào đây nghĩa là bảng khai vừa bị thủng — và lúc đó
// đoán "làm được" là cách sai nhất.
func TrangThaiCua(ad Adapter, khoa string) TrangThaiNangLuc {
	for _, nl := range ad.NangLuc() {
		if nl.Khoa == khoa {
			return nl.TrangThai
		}
	}
	return ChuaDo
}

// mocPrompt là chỗ giữ chỗ của prompt khi hỏi adapter "bộ cờ của mày gồm gì".
// Dùng một chuỗi không ai gõ ra được, để phân biệt chắc chắn với cờ thật.
const mocPrompt = "\x00prompt\x00"

// CoConThieu trả về những cờ mà `HeadlessArgs` sinh ra nhưng `args` chưa có.
//
// VÌ SAO CẦN: `sagent fleet <addr> -- <args>` truyền args THÔ cho CLI con, không
// đi qua `HeadlessArgs` của adapter. Người dùng gõ `-- -p "việc"` là đủ để agent
// chạy, nhưng THIẾU `--output-format stream-json --verbose` — mà đúng hai cờ đó
// mới làm CLI in ra dòng `{"type":"result"}` chứa is_error, usage, total_cost_usd.
// Không có nó thì `DocKetQua` không đọc được gì và MỌI phiên đều về `lost`.
//
// Đo 20/08/2026: bảng `sagent status` hiện 20 phiên liền, phiên nào cũng "chết,
// chưa rõ vì sao", tokens và chi phí đều "chưa đo" — trong khi cùng lúc đó đường
// flow (đi qua `argsChoBuoc`, tức qua adapter) đo được 99.051 token vào,
// 81.492 token ra, 11,0572 USD cho lượt chạy #47. Cùng một CLI, cùng một tài
// khoản, hai đường: đường nào hỏi adapter thì đo được.
//
// Hàm này KHÔNG chép cứng tên cờ nào. Nó hỏi chính adapter — nên thêm provider
// mới, hay Claude đổi tên cờ, thì chỗ này tự đúng theo.
func CoConThieu(ad Adapter, args []string) []string {
	// Provider chưa đo được cách đọc kết quả có cấu trúc thì KHÔNG thêm gì: cờ
	// khai bừa còn tệ hơn thiếu cờ, vì nó làm CLI con chết ngay từ đầu.
	if TrangThaiCua(ad, NLKetQuaCoCauTruc) != LamDuoc {
		return nil
	}

	mau := ad.HeadlessArgs(mocPrompt)
	daCo := map[string]bool{}
	for _, a := range args {
		daCo[a] = true
	}

	var thieu []string
	for i := 0; i < len(mau); i++ {
		co := mau[i]
		if co == mocPrompt || !strings.HasPrefix(co, "-") {
			// Chỗ của prompt, hoặc lệnh con kiểu `codex exec` — không phải cờ.
			continue
		}
		// Gom giá trị đi kèm: mọi phần tử tới cờ tiếp theo (hoặc tới prompt).
		var giaTri []string
		for j := i + 1; j < len(mau) && mau[j] != mocPrompt && !strings.HasPrefix(mau[j], "-"); j++ {
			giaTri = append(giaTri, mau[j])
		}
		i += len(giaTri)

		if daCo[co] {
			continue // người dùng đã tự gõ cờ này rồi, tôn trọng giá trị của họ
		}
		thieu = append(thieu, co)
		thieu = append(thieu, giaTri...)
	}
	return thieu
}
