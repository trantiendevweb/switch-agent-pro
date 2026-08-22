package aiapi

import (
	"context"
	"fmt"
	"strings"
)

// GỌI KÈM — một lượt gọi mang thêm ảnh và/hoặc JSON schema.
//
// # VÌ SAO MỘT HÀM CHỨ KHÔNG PHẢI `GoiAnh` VÀ `GoiCoCauTruc`
//
// Vì hai thứ đó GHÉP ĐƯỢC với nhau: "đọc cái hoá đơn trong ảnh này rồi trả về
// JSON {so_tien, ngay}" là một lượt gọi duy nhất mang cả hai. Tách thành hai
// hàm thì ca ghép hoặc không làm được, hoặc phải có hàm thứ ba — và ba đường
// làm gần cùng một việc là ba đường sẽ lệch nhau. Đó đúng là cái giá mà ghi chú
// đầu anh.go vừa nói tới, nên chỗ này trả nó một lần cho xong.
//
// `Goi` và `GoiTool` KHÔNG bị đụng tới: `Goi(ctx, r, prompt)` là chữ ký cả
// `internal/api`, CLI và mặt web đang gọi, và một lượt hỏi bình thường vẫn phải
// gửi thân JSON y hệt như trước.
//
// # CÂU KHÓ THỨ HAI: ROUTE KHÔNG LÀM ĐƯỢC MÀ NGƯỜI DÙNG VẪN GỬI ẢNH THÌ SAO
//
// Trả lời: CHẶN Ở PHÍA MÌNH, trước khi chạm mạng, nhưng CHỈ khi bảng năng lực
// đã ĐO ĐƯỢC là không làm được — và luôn chừa một cửa gõ tiếp.
//
// Vì sao chặn chứ không cứ gửi rồi để nhà cung cấp trả 400:
//
//  1. Bảng `sagent nang-luc-api` ĐÃ BIẾT câu trả lời trước khi gọi (deepseek-
//     v4-flash: HTTP 400 "This model does not support image", đo 22/08). Biết
//     mà không dùng là đúng cái bẫy "có ở mọi tầng trừ tầng cuối cùng" mà dự án
//     này dính năm lần trong ngày 22/08 — và lần này thì thứ bị bỏ phí là một
//     phép đo đã tốn tiền thật để có.
//  2. Một lượt gọi hỏng vẫn có thể bị tính tiền, và ảnh base64 làm phần
//     `prompt_tokens` phình lên trước khi nhà cung cấp kịp từ chối.
//  3. Lỗi của nhà cung cấp là tiếng Anh, nói về một `image_url` mà người dùng
//     chưa từng gõ, và KHÔNG nói route nào trong cấu hình của họ làm được.
//     Lời chặn ở đây nói được điều đó, vì nó cầm cả danh sách route.
//
// Vì sao chỉ chặn khi ĐÃ ĐO, và không bao giờ chặn khi `ChuaDo`:
//
// Ba trạng thái của bảng là một quy ước về NGHĨA, và "chưa ai đo" KHÔNG phải
// "đã đo là không". Chặn một ô `ChuaDo` là bẹp ba trạng thái thành hai ngay tại
// chỗ tiêu tiền — thêm một route mới vào `.sagent/project.toml` là lập tức
// không gửi ảnh được, mà chẳng ai đo gì cả. Ô `ChuaDo` được đi, kèm một câu nói
// rõ là chưa đo.
//
// Và vì sao vẫn phải có cửa `CuGui`: bằng chứng trong sổ có NGÀY (22/08), nhà
// cung cấp thì nâng cấp model. Một lời chặn không gỡ được sẽ biến số đo hôm nay
// thành luật vĩnh viễn, và cách duy nhất để biết nó hết đúng là gửi thử.

// TuyChonGoi là những thứ THÊM vào một lượt gọi thường.
//
// Rỗng hoàn toàn thì `GoiKem` gửi đi đúng thân JSON của `Goi`.
type TuyChonGoi struct {
	// Anh là ảnh gửi kèm. Đọc bằng `DocAnh`/`DocNhieuAnh` — đừng tự nối chuỗi
	// data URL, chỗ đó là nơi kiểu file sai lọt qua rồi tốn một lượt gọi.
	Anh []Anh

	// TraLoiTheoSoDo là `response_format`. Dựng bằng `SoDoNghiem`.
	TraLoiTheoSoDo *DangTraLoi

	// DsRoute là danh sách route đã cấu hình (`API.AIRoutes()`), CHỈ dùng để
	// lời chặn gợi ý được route khác làm được. Nil thì lời chặn vẫn đúng, chỉ
	// là không gợi ý được gì.
	DsRoute []Route

	// CuGui bỏ qua lời chặn của bảng năng lực. Xem ghi chú đầu file.
	CuGui bool
}

// coGiKem nói tuỳ chọn này có mang thêm thứ gì không.
func (t TuyChonGoi) coGiKem() bool { return len(t.Anh) > 0 || t.TraLoiTheoSoDo != nil }

// GoiKem gửi một prompt kèm ảnh và/hoặc JSON schema.
//
// ĐI ĐÍCH DANH ROUTE ĐƯỢC ĐƯA VÀO, KHÔNG nhảy dự phòng — cùng ranh giới với
// `GoiTool` và cùng lý do: bộ chuyển route dự phòng nằm trong `internal/api` và
// nó gọi `aiapi.Goi`, đường KHÔNG mang ảnh lẫn schema. Cho nhánh này mượn đường
// đó thì lượt dự phòng gửi đi một yêu cầu trơ, model trả lời bằng chữ, và người
// dùng nhận về "route này không đọc được ảnh" — một câu sai, sinh ra từ một chỗ
// chuyển route im lặng. Thà không có dự phòng còn hơn có một cái nói dối.
func GoiKem(ctx context.Context, r Route, prompt string, tc TuyChonGoi) (KetQua, error) {
	if !tc.coGiKem() {
		// Gọi cửa này mà không kèm gì là lỗi của người gọi, và đi tiếp thì họ
		// nhận về một câu trả lời bình thường rồi kết luận nhầm rằng ảnh đã gửi
		// mà model không thấy.
		return KetQua{}, loiNguoi(r.Ten, 0, "GoiKem không kèm ảnh lẫn schema — "+
			"dùng Goi nếu chỉ muốn hỏi")
	}

	var canh []string
	if len(tc.Anh) > 0 {
		c, err := SoatTruocKhiGui(r, NLAPIDauVaoAnh, tc.DsRoute, tc.CuGui)
		if err != nil {
			return KetQua{}, err
		}
		canh = append(canh, c...)
		canh = append(canh, canhBaoAnh(tc.Anh)...)
	}
	if tc.TraLoiTheoSoDo != nil {
		c, err := SoatTruocKhiGui(r, NLAPIDauRaCoCauTruc, tc.DsRoute, tc.CuGui)
		if err != nil {
			return KetQua{}, err
		}
		canh = append(canh, c...)
	}

	var phan []PhanNoiDung
	for _, a := range tc.Anh {
		phan = append(phan, PhanAnh(a.DataURL()))
	}
	kq, err := goiThat(ctx, r, prompt, themVao{Phan: phan, DangTraLoi: tc.TraLoiTheoSoDo})
	// Cảnh báo gắn vào KetQua kể cả khi lượt gọi HỎNG: câu "ảnh chỉ có 64 điểm
	// ảnh" chính là thứ giải thích cái HTTP 400 vừa nhận được, và vứt nó đi ở
	// đúng nhánh lỗi là vứt đúng lúc người ta cần nó nhất.
	kq.CanhBaoTruocKhiGui = canh
	if err != nil {
		return kq, err
	}
	return kq, nil
}

// SoatTruocKhiGui soát MỘT năng lực của route TRƯỚC khi chạm mạng.
//
// Trả về (cảnh báo, lỗi):
//
//	lỗi khác nil     → bảng ĐÃ ĐO ĐƯỢC là không làm được. Chặn.
//	cảnh báo khác rỗng → chưa ai đo, hoặc người dùng đã ép gửi. Vẫn đi, nhưng nói ra.
//
// Hai thứ trả về từ MỘT hàm chứ không phải hai hàm rời, có chủ ý: tách ra thì
// sẽ có chỗ gọi hàm chặn mà quên hàm cảnh báo, và ca `ChuaDo` sẽ đi qua trong
// im lặng — tức là "chưa đo" lại một lần nữa bị đọc thành "chạy được".
//
// Lỗi trả về là LỖI NGƯỜI DÙNG (`LoiNguoiDung` → true): tầng trên KHÔNG được
// coi đây là cớ để nhảy route dự phòng, vì đường dự phòng đi qua `Goi` và sẽ
// gửi một yêu cầu không có ảnh.
func SoatTruocKhiGui(r Route, khoa string, ds []Route, cuGui bool) ([]string, error) {
	var m NangLucAPI
	var thay bool
	for _, x := range BangNangLuc(r).Muc {
		if x.Khoa == khoa {
			m, thay = x, true
			break
		}
	}
	if !thay {
		return []string{fmt.Sprintf("không có năng lực %q trong bảng — không soát được "+
			"route %q trước khi gửi", khoa, r.Ten)}, nil
	}

	switch m.TrangThai {
	case LamDuoc:
		return nil, nil
	case ChuaDo:
		return []string{fmt.Sprintf("CHƯA ai đo route %q có %s hay không (%s). Vẫn gửi — "+
			"\"chưa đo\" KHÔNG phải \"không làm được\". Lượt này hỏng thì đo lại bằng: %s --do %s",
			r.Ten, moTaNangLucAPI(khoa), m.BangChung, "sagent nang-luc-api", r.Ten)}, nil
	}

	// Đã đo được là KHÔNG. Dựng lời chặn — hoặc lời cảnh báo, nếu bị ép gửi.
	cau := fmt.Sprintf("route %q KHÔNG %s — %s", r.Ten, moTaNangLucAPI(khoa), m.BangChung)
	if cuGui {
		return []string{cau + "\n     Bạn đã ép gửi (--cu-gui). Số đo trên có NGÀY, và nhà " +
			"cung cấp thì nâng cấp model — nếu lượt này chạy được, ghi lại bằng " +
			"`sagent nang-luc-api --do " + r.Ten + "`."}, nil
	}
	var b strings.Builder
	b.WriteString(cau)
	b.WriteString("\n     Chặn ở phía sagent, CHƯA gửi đi: bảng năng lực đã biết câu trả " +
		"lời trước khi gọi, và một lượt hỏng vẫn có thể bị tính tiền.")
	if g := goiYRouteLamDuoc(ds, r.Ten, khoa); g != "" {
		b.WriteString("\n     " + g)
	}
	b.WriteString(fmt.Sprintf("\n     Số đo có thể đã cũ — vẫn muốn gửi thì thêm --cu-gui, "+
		"hoặc đo lại: sagent nang-luc-api --do %s", r.Ten))
	return nil, loiNguoi(r.Ten, 0, "%s", b.String())
}

// goiYRouteLamDuoc tìm route KHÁC trong cấu hình làm được năng lực này.
//
// Rỗng khi không có route nào — và lúc đó KHÔNG bịa ra một lời khuyên chung
// chung kiểu "hãy dùng route hỗ trợ vision": người dùng không có route đó, câu
// đó chỉ làm họ mất thêm một vòng đi tìm.
func goiYRouteLamDuoc(ds []Route, tru, khoa string) string {
	var duoc []string
	for _, r := range ds {
		if r.Ten == tru {
			continue
		}
		if ok, _ := BangNangLuc(r).LamDuoc(khoa); ok {
			duoc = append(duoc, fmt.Sprintf("%s (%s)", r.Ten, r.Model))
		}
	}
	if len(duoc) == 0 {
		return ""
	}
	return "Route đã cấu hình mà LÀM ĐƯỢC: " + strings.Join(duoc, ", ")
}
