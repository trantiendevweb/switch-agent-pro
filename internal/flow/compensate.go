// COMPENSATE — bước hỏng thì chạy một bước GỠ LẠI (undo).
//
// ============================================================================
// KHÁC `stop` VÀ `continue` Ở CHỖ NÀO
// ============================================================================
//
//	stop        dừng, để nguyên hiện trường. Việc dở dang nằm lại đó.
//	continue    kệ, đi tiếp. Việc dở dang cũng nằm lại đó, chỉ khác là không ai dừng.
//	compensate  chạy một bước GỠ LẠI, RỒI dừng.
//
// Cả `stop` lẫn `continue` đều để lại một nửa việc. Với những việc có tác dụng
// ra ngoài — tạo nhánh, đẩy commit, dựng máy, mở PR — nửa việc là thứ tệ nhất:
// nó không đủ để dùng, mà lại đủ để lần sau chạy đụng vào.
//
// ============================================================================
// CÂU HỎI 1: BƯỚC GỠ LẠI MÀ CŨNG HỎNG THÌ SAO?
// ============================================================================
//
// KHÔNG thử lại vô hạn, KHÔNG gỡ-lại-của-gỡ-lại, KHÔNG đi tiếp.
//
// Lượt chạy dừng, và sự kiện báo hỏng nói RÕ RÀNG rằng đây là một trạng thái
// KHÔNG BIẾT: việc chính không xong, mà cũng không gỡ được. Đây là câu khác hẳn
// với "bước x hỏng" — nó là câu "có người phải vào dọn tay".
//
// Vì sao không tự thử lại thêm: bước gỡ lại đã có `retry` của riêng nó, và
// người viết flow đặt con số đó. Tự ý thử thêm ở tầng này là làm hộ một quyết
// định mà họ đã nói ra rồi.
//
// Vì sao không gỡ-lại-của-gỡ-lại: một chuỗi undo lồng nhau là một chỗ để treo
// vô hạn, và cái gỡ ở tầng thứ ba thì không ai còn hình dung được nó đang gỡ
// cái gì. `on_failure` của bước gỡ lại bị BỎ QUA — Validate nói ra chuyện đó.
//
// ============================================================================
// CÂU HỎI 2: CÓ GỠ LẠI CÁC BƯỚC ĐÃ XONG TRƯỚC ĐÓ KHÔNG?
// ============================================================================
//
// KHÔNG. Chỉ gỡ lại CHÍNH BƯỚC HỎNG. Ba lý do, theo thứ tự quan trọng:
//
//  1. DAG KHÔNG PHẢI MỘT NGĂN XẾP. Bộ chạy này chạy theo ĐỢT, nhiều bước song
//     song. Hai bước cùng đợt không có thứ tự xong nào cả — nên "gỡ ngược theo
//     thứ tự đã chạy" là một câu không có nghĩa ở đây. Bịa ra một thứ tự để mà
//     gỡ là bịa ra một sự thật.
//
//  2. GỠ LAN LÀ GỠ SANG VIỆC KHÔNG LIÊN QUAN. Flow có ba nhánh độc lập; một
//     bước lá ở nhánh 3 hỏng mà kéo theo undo cả nhánh 1 và 2 thì nó phá đúng
//     những việc đã làm xong đàng hoàng.
//
//  3. Cần gỡ cả chuỗi thì NÓI RA ĐƯỢC: viết một bước gỡ lại làm trọn việc đó
//     (`terraform destroy` một lần, thay vì ba bước undo lồng nhau). Người viết
//     flow biết cái gì cần gỡ cùng nhau; bộ chạy thì không.
//
// ============================================================================
// BƯỚC GỠ LẠI KHÔNG PHẢI MỘT NODE BÌNH THƯỜNG
// ============================================================================
//
// Bước được ai đó trỏ tới bằng `compensate` bị LOẠI khỏi lịch chạy thường. Nếu
// không thì nó là một node không có `needs` — tức là một GỐC của DAG — và sẽ
// chạy ngay ở đợt đầu tiên của MỌI lượt chạy, gỡ một việc chưa ai làm.
//
// Nó được ghi `skipped` ngay từ đầu với lời giải thích, chứ không để trống: một
// ô trống trên bảng đọc là "chưa tới lượt", còn đây là "sẽ không chạy trừ khi
// có chuyện".
package flow

import (
	"fmt"
	"sort"
)

// KhoaBuocHong là tên biến bước gỡ lại dùng để biết mình đang gỡ cái gì.
//
//	prompt = "Bước {{buoc_hong}} vừa hỏng. Xoá nhánh nó đã tạo."
//
// Có mặt CHỈ ở bước gỡ lại, và chỉ lúc nó chạy như một bước gỡ.
const KhoaBuocHong = "buoc_hong"

// BuocGoLai là tập id các bước ĐANG LÀM NHIỆM VỤ GỠ cho một bước khác.
//
// Dùng để loại chúng khỏi lịch chạy thường — xem ghi chú ở đầu file.
func BuocGoLai(f Flow) map[string]bool {
	out := map[string]bool{}
	for _, s := range f.Steps {
		if s.OnFailure == OnFailCompensate && s.Compensate != "" {
			out[s.Compensate] = true
		}
	}
	return out
}

// TimBuoc trả về bước theo id.
func TimBuoc(f Flow, id string) (Step, bool) {
	for _, s := range f.Steps {
		if s.ID == id {
			return s, true
		}
	}
	return Step{}, false
}

// MoTaChoGoLai là câu ghi vào sổ cho một bước gỡ lại KHÔNG được gọi tới.
func MoTaChoGoLai(f Flow, id string) string {
	var chu []string
	for _, s := range f.Steps {
		if s.OnFailure == OnFailCompensate && s.Compensate == id {
			chu = append(chu, s.ID)
		}
	}
	return fmt.Sprintf("bước gỡ lại — chỉ chạy khi %s hỏng", nhomTen(chu))
}

func nhomTen(ids []string) string {
	switch len(ids) {
	case 0:
		return "(không bước nào)"
	case 1:
		return ids[0]
	}
	out := ids[0]
	for _, id := range ids[1:] {
		out += " hoặc " + id
	}
	return out
}

// VanDeCompensate soi phần `compensate` của cả flow.
func VanDeCompensate(f Flow) []Problem {
	var ps []Problem
	loi := func(step, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: step, Msg: msg}) }
	canh := func(step, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: step, Msg: msg, Warn: true}) }

	co := map[string]bool{}
	for _, s := range f.Steps {
		co[s.ID] = true
	}
	goLai := BuocGoLai(f)

	// Ai đang phụ thuộc vào một bước gỡ lại: bước gỡ lại KHÔNG chạy ở lịch
	// thường, nên một `needs` trỏ vào nó là một bước treo vĩnh viễn.
	aiCho := map[string][]string{}
	for _, s := range f.Steps {
		for _, n := range s.Needs {
			aiCho[n] = append(aiCho[n], s.ID)
		}
	}

	for _, s := range f.Steps {
		if s.OnFailure == OnFailCompensate {
			switch {
			case s.Compensate == "":
				loi(s.ID, "on_failure = \"compensate\" thì phải khai báo `compensate` là id bước gỡ lại")
			case s.Compensate == s.ID:
				loi(s.ID, "bước không thể tự gỡ lại chính nó — nó vừa hỏng, chạy lại y nguyên "+
					"thì hỏng y nguyên")
			case !co[s.Compensate]:
				loi(s.ID, fmt.Sprintf("compensate trỏ tới bước %q không tồn tại", s.Compensate))
			}
		}
		if s.Fallback != "" && s.OnFailure == OnFailCompensate {
			canh(s.ID, "khai cả `fallback` lẫn on_failure = \"compensate\" — `fallback` sẽ không được dùng")
		}

		if !goLai[s.ID] {
			continue
		}

		// Từ đây trở xuống: s LÀ một bước gỡ lại.
		switch s.Type {
		case TypeApprove:
			loi(s.ID, "bước gỡ lại không được là `approve`: nó chỉ chạy khi có sự cố, và một lượt "+
				"chạy đang dừng vì hỏng thì không có ai ngồi đó bấm duyệt — nó sẽ treo mãi")
		case TypeNotify:
			canh(s.ID, "bước gỡ lại kiểu `notify` chỉ in ra một dòng chữ, nó KHÔNG gỡ lại gì cả "+
				"— nếu chỉ muốn báo thì dùng on_failure = \"stop\", sự kiện báo hỏng đã có sẵn")
		}
		if s.Idempotent {
			loi(s.ID, "bước gỡ lại không được `idempotent = true`: khoá sẽ trùng với lần gỡ trước "+
				"và lần này bị bỏ qua — tức là lần sự cố thứ hai không ai gỡ")
		}
		if s.OnFailure != "" {
			canh(s.ID, fmt.Sprintf("on_failure = %q của bước gỡ lại bị BỎ QUA: gỡ-lại-của-gỡ-lại là "+
				"một chỗ treo vô hạn. Gỡ mà hỏng thì lượt chạy dừng và báo là trạng thái KHÔNG BIẾT",
				s.OnFailure))
		}
		if len(s.Needs) > 0 {
			canh(s.ID, "`needs` của bước gỡ lại bị BỎ QUA: nó chạy ngoài lịch thường, ngay lúc sự cố")
		}
		if ai := aiCho[s.ID]; len(ai) > 0 {
			loi(s.ID, fmt.Sprintf("bước %s khai `needs` tới bước gỡ lại này, nhưng bước gỡ lại KHÔNG "+
				"chạy ở lịch thường — %s sẽ không bao giờ tới lượt", nhomTen(ai), nhomTen(ai)))
		}
	}
	return ps
}

// BuocDuocGoLaiBoi trả về các bước mà `id` làm nhiệm vụ GỠ LẠI cho, đã sắp xếp.
//
// Cùng dữ liệu với MoTaChoGoLai nhưng ở dạng DANH SÁCH thay vì một câu: mặt web
// và bảng chạy khan cần nối được "gỡ cho bước nào" thành liên kết, mà tách chuỗi
// từ một câu tiếng Việt thì sớm muộn cũng sai. Một nguồn, hai dạng — chứ không
// phải hai chỗ tự đi đếm.
func BuocDuocGoLaiBoi(f Flow, id string) []string {
	var out []string
	for _, s := range f.Steps {
		if s.OnFailure == OnFailCompensate && s.Compensate == id {
			out = append(out, s.ID)
		}
	}
	sort.Strings(out)
	return out
}
