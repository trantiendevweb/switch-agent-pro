package provider

// Trạng thái KẾT THÚC BẤT THƯỜNG của một phiên, suy từ dữ liệu CÓ CẤU TRÚC.
//
// Ba chuỗi này phải TRÙNG store.StateHanMuc/StateChan/StateHong. Cố ý không
// import store: gói này không được biết gì về sổ trên đĩa. Có test ghim hai bên
// bằng nhau — xem internal/api/phientrangthai_test.go.
const (
	// ChetHanMuc: agent dừng vì hết hạn mức, và CLI nói được lúc nào cấp lại.
	ChetHanMuc = "rate_limited"
	// ChetChanQuyen: mọi tool bị TỪ CHỐI QUYỀN nên agent không làm được gì.
	ChetChanQuyen = "blocked"
	// ChetLoiAPI: nhà cung cấp trả về mã lỗi (api_error_status).
	ChetLoiAPI = "failed"

	// Xong KHÔNG phải một kiểu chết — nó là lượt chạy kết thúc bình thường.
	// Để chung khối này vì cùng đi qua một điểm quyết định (PhanLoaiChet), và
	// chuỗi phải TRÙNG store.StateXong.
	Xong = "done"
)

// PhanLoaiChet nói một phiên vừa được phát hiện CHẾT nên mang trạng thái nào.
//
// Đây là chỗ DUY NHẤT trong dự án quyết định điều đó (xem
// internal/api.phanLoaiPhienChet cắm nó vào sổ). Bốn mặt điều khiển chỉ đọc lại
// `Session.State`, không mặt nào tự suy — nên luật ngang quyền tự giữ được.
//
// LUẬT: KHÔNG DÒ CHUỖI. Mọi kết luận đọc từ trường có tên của KetQua, tức từ
// những thứ Claude phát ra dưới dạng dữ liệu — `rate_limit_info.resetsAt` là
// một timestamp thật, `permission_denials` là một mảng, `api_error_status` là
// một enum. Provider đổi câu chữ thì kết luận ở đây không đổi.
//
// LUẬT THỨ HAI, quan trọng hơn: THIẾU DỮ LIỆU THÌ KHÔNG SUY.
// docDuoc=false (Codex, Cursor — và Grok/Antigravity khi bản ghi không có dòng
// result) thì trả về "" và phiên ở lại `lost`. "Không biết vì sao nó chết" là
// câu trả lời trung thực; đoán bừa một trong ba trạng thái kia thì người vận
// hành sẽ đi ngồi chờ hạn mức trong khi thật ra token đã hết hạn.
//
// trangThai == "" nghĩa là KHÔNG KẾT LUẬN ĐƯỢC — người gọi giữ nguyên `lost`.
func PhanLoaiChet(k KetQua, docDuoc bool) (trangThai, lyDo string, hanMucDenLai int64) {
	if !docDuoc {
		return "", "", 0
	}
	// Lượt chạy XONG XUÔI thì không có gì bất thường để nói. Phiên vẫn rời sổ
	// "đang chạy" (tiến trình đã thoát), nhưng gán cho nó một trong ba trạng
	// thái hỏng là vu oan. Hong() là cùng một phép đo mà flow đang dùng, nên
	// hai chỗ không thể nói ngược nhau.
	ly := k.Hong()
	if ly == "" {
		// Đọc được kết quả VÀ không hỏng gì = lượt chạy XONG XUÔI. Trước đây chỗ
		// này trả rỗng, tức phiên thành công ở lại `lost` và bảng hiện "chết, chưa
		// rõ vì sao" — đo được 20/08 với phiên #157: agent trả lời đúng, NDJSON có
		// dòng result, không lỗi nào, mà vẫn bị xếp chung sọt với phiên chết bí ẩn.
		//
		// Đây KHÔNG phải suy đoán: chỉ nói "xong" khi ĐỌC ĐƯỢC bản ghi kết quả.
		// Provider chưa đo được kết quả có cấu trúc vẫn rơi về nhánh `!docDuoc` ở
		// trên và ở lại `lost`, đúng như cũ.
		return Xong, "", 0
	}
	switch {
	// HẠN MỨC xét TRƯỚC vì nó là chẩn đoán CỤ THỂ NHẤT và là chẩn đoán duy nhất
	// đi kèm một việc phải làm rõ ràng: chờ tới mốc `resetsAt` rồi chạy lại.
	// Cùng một lượt hết hạn mức thường KÈM cả `api_error_status` — xếp lỗi API
	// lên trước thì lời khuyên hoá thành "agent báo lỗi 429", đúng mà vô dụng.
	// Mốc thời gian không mất đi ở hai nhánh dưới: cột han_muc_den_lai vẫn ghi.
	case k.HanMucDenLai > 0:
		return ChetHanMuc, "hết hạn mức, chờ được cấp lại — " + ly, k.HanMucDenLai
	// Cùng điều kiện mà KetQua.Hong() đã dùng: bị từ chối quyền cho MỌI tool và
	// không nặn ra được câu trả lời nào.
	case k.TuChoiSo > 0 && k.TraLoi == "":
		return ChetChanQuyen, ly, k.HanMucDenLai
	case k.CoLoi && k.LoiAPI != "":
		return ChetLoiAPI, ly, k.HanMucDenLai
	}
	// Hỏng theo kiểu khác (chạy quẩn, không trả lời gì, hết vòng tool). Đo được
	// là nó HỎNG, nhưng ba trạng thái trên không cái nào tả đúng — nên không
	// nhét bừa vào cái gần nhất.
	return "", "", 0
}
