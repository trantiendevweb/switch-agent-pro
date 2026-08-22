package aiapi

import (
	"fmt"
	"strings"
)

// ĐỌC PHẦN SUY LUẬN — và nói cho đúng khi KHÔNG có phần suy luận.
//
// `KetQua.SuyLuan` có từ chiều 22/08 và cho tới lúc file này ra đời thì KHÔNG
// MẶT NÀO đọc nó: không cờ CLI, không chỗ nào trên dashboard. Đo thật với
// deepseek-v4-flash: câu trả lời 91 ký tự, phần suy luận 477 ký tự — phần bị
// vứt dài gấp 5,2 lần phần giữ lại, và người dùng đã trả tiền cho cả hai (300
// token). Đó là đúng cái hình dạng lỗi "có ở mọi tầng trừ tầng cuối cùng" mà dự
// án này dính năm lần trong một ngày.
//
// # VÌ SAO PHẢI CÓ MỘT KIỂU RIÊNG, KHÔNG CHỈ IN `kq.SuyLuan` RA
//
// Vì chuỗi rỗng ở đó mang HAI nghĩa ngược nhau về hậu quả:
//
//	(a) model KHÔNG nghĩ — lượt này nó trả lời thẳng;
//	(b) nhà cung cấp KHÔNG TRẢ phần nghĩ ra, dù model có nghĩ.
//
// Đo 22/08, grok-4.5 qua modelapi.vn là ca (b) rành rành: HTTP 200, không có
// `reasoning_content` lẫn `reasoning`, mà lượt đó tiêu 951 token và mất 20,3
// giây. Model nghĩ nát ra; người dùng trả tiền cho phần nghĩ đó; nhà bán lại
// không đưa nó về.
//
// In một ô trống, hay tệ hơn là in "model không suy luận", biến (b) thành (a)
// ngay trước mắt người trả tiền. Nên mọi mặt hiện phần suy luận đều phải đi qua
// `DocSuyLuan`, và khi rỗng thì nói ra mình đang ở nghĩa nào — bằng cách hỏi
// đúng chỗ có câu trả lời: bảng năng lực của route đó.

// KhoiSuyLuan là phần suy luận ĐÃ SẴN SÀNG ĐỂ HIỆN RA, kèm câu giải thích khi
// không có gì để hiện.
//
// Có thẻ JSON vì mặt web đọc thẳng kiểu này qua `/api/ai` — một nguồn cho cả
// CLI lẫn dashboard, để hai mặt không thể nói hai nghĩa về cùng một chuỗi rỗng.
type KhoiSuyLuan struct {
	// Co: lượt này CÓ phần suy luận đọc được.
	Co      bool   `json:"co"`
	NoiDung string `json:"noi_dung"`
	SoKyTu  int    `json:"so_ky_tu"`

	// ViSaoRong chỉ có khi Co == false, và KHÔNG BAO GIỜ được rỗng lúc đó: một
	// ô trống không giải thích chính là thứ bị đọc thành "model không nghĩ".
	ViSaoRong string `json:"vi_sao_rong"`

	// DanToi là lệnh gõ được để tự kiểm lại — bảng năng lực là chỗ duy nhất
	// trong dự án trả lời được câu "route này có trả phần nghĩ không".
	DanToi string `json:"dan_toi"`
}

// DocSuyLuan dựng khối suy luận của MỘT lượt gọi.
//
// `ds` là danh sách route đã cấu hình (`API.AIRoutes()`), cần để tra bảng năng
// lực của route THẬT SỰ trả lời — `kq.Route`, không phải route người dùng gõ:
// lượt đã chuyển sang route dự phòng thì câu trả lời "có trả phần nghĩ không"
// thuộc về nhà cung cấp đã trả lời, không phải nhà cung cấp đã hỏng.
//
// KHÔNG chạm mạng và KHÔNG tốn token: `BangNangLuc` chỉ ghép phép đo mã nguồn
// với sổ số đo có sẵn.
func DocSuyLuan(kq KetQua, ds []Route) KhoiSuyLuan {
	if s := strings.TrimSpace(kq.SuyLuan); s != "" {
		return KhoiSuyLuan{
			Co: true, NoiDung: s, SoKyTu: len([]rune(s)),
			DanToi: lenhNangLuc(kq.Route),
		}
	}

	k := KhoiSuyLuan{DanToi: lenhNangLuc(kq.Route)}

	r, co := timRoute(ds, kq.Route)
	if !co {
		// Không tra được bảng thì NÓI LÀ KHÔNG TRA ĐƯỢC. Đây là chỗ dễ nhất để
		// tiện tay viết "model không suy luận" — câu đó không đo được từ đâu cả.
		k.ViSaoRong = fmt.Sprintf("lượt này không có phần suy luận, và route %q không "+
			"có trong cấu hình nên không tra được bảng năng lực. CHƯA kết luận được ô "+
			"trống này thuộc về model hay thuộc về nhà cung cấp.", kq.Route)
		return k
	}

	var ncc TrangThaiNangLuc = ChuaDo
	var bcNCC string
	for _, m := range BangNangLuc(r).Muc {
		if m.Khoa == NLAPIReasoning {
			ncc, bcNCC = m.NCC, m.BangChungNCC
			break
		}
	}

	switch ncc {
	case KhongLamDuoc:
		// Ca đã đo được của grok-4.5. Câu này phải nói rõ tiền vẫn bị tiêu.
		k.ViSaoRong = fmt.Sprintf("nhà cung cấp của route %q KHÔNG trả phần nghĩ ra "+
			"(%s). Nghĩa là ô trống này KHÔNG nói được model có nghĩ hay không — nếu nó "+
			"có nghĩ thì token của phần nghĩ vẫn nằm trong hoá đơn.", r.Ten, bcNCC)
	case LamDuoc:
		// Route ĐÃ đo được là có trả. Vậy lượt này trống thì nghiêng về (a) —
		// nhưng vẫn không được khẳng định thay model: cùng một route có thể
		// nghĩ ở câu khó và không nghĩ ở câu dễ.
		k.ViSaoRong = fmt.Sprintf("route %q ĐÃ đo được là có trả phần nghĩ (%s), nhưng "+
			"lượt này không kèm phần nào — nhiều khả năng model trả lời thẳng, không nghĩ "+
			"từng bước.", r.Ten, bcNCC)
	default:
		k.ViSaoRong = fmt.Sprintf("chưa ai đo route %q có trả phần nghĩ ra hay không "+
			"(%s), nên ô trống này chưa nói được gì.", r.Ten, bcNCC)
	}
	if kq.DaStreaming {
		// Đường stream đọc phần nghĩ ở `delta.reasoning_content` (stream.go).
		// Ghi ra đây vì nếu bộ đọc stream có lúc nào đó thôi gom phần đó, mọi
		// lượt trên dashboard sẽ trống trơn và câu giải thích ở trên sẽ đổ tội
		// oan cho nhà cung cấp.
		k.ViSaoRong += " (Lượt này đi đường stream; phần nghĩ ở đường đó về theo " +
			"`delta.reasoning_content`.)"
	}
	return k
}

// lenhNangLuc dựng lệnh dẫn người đọc tới chỗ trả lời được câu hỏi.
func lenhNangLuc(route string) string {
	if strings.TrimSpace(route) == "" {
		return "sagent nang-luc-api"
	}
	return "sagent nang-luc-api " + route
}

// timRoute tra route theo tên trong danh sách đã cấu hình.
func timRoute(ds []Route, ten string) (Route, bool) {
	for _, r := range ds {
		if r.Ten == ten {
			return r, true
		}
	}
	return Route{}, false
}
