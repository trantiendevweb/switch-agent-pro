// FALLBACK — bước hỏng thì chạy một bước KHÁC THAY CHO NÓ.
//
// ============================================================================
// TRƯỚC BẢN NÀY: `fallback` GẦN NHƯ KHÔNG LÀM GÌ
// ============================================================================
//
// Nhánh `case OnFailFallback` trong xuLyHong chỉ in ra một dòng cảnh báo rồi
// trả về "đừng dừng lượt chạy". Nó KHÔNG gọi bước được trỏ tới. Đo thật bằng
// ba lượt `sagent flow run` (#59, #60, #61 — bước `shell`, 0 token):
//
//	cách khai bước thay thế     bước chính  bước thay thế  lượt chạy
//	───────────────────────────────────────────────────────────────────
//	không `needs` (gốc DAG)     hỏng        CHẠY — nhưng   completed
//	                                        SONG SONG, ở
//	                                        đợt đầu, TRƯỚC
//	                                        khi bước chính
//	                                        kịp hỏng
//	`needs = ["chinh"]`         hỏng        KHÔNG chạy     completed
//	không `needs` (gốc DAG)     XONG        CHẠY — dù      completed
//	                                        chẳng ai hỏng
//
// Ba hàng, không hàng nào là thứ người viết flow gõ `on_failure = "fallback"`
// để có. Hàng 1 chạy đúng bước nhưng SAI LÚC: nó không thể phản ứng với một sự
// cố chưa xảy ra. Hàng 2 im lặng không làm gì và lượt chạy vẫn được ghi là
// `completed`. Hàng 3 là bản sao đúng cái bẫy mà `compensate` đã tránh: bước
// thay thế không có `needs` là một GỐC của DAG, nên nó chạy ở mọi lượt.
//
// ============================================================================
// KHÁC `compensate` Ở ĐÚNG MỘT CHỖ: SAU ĐÓ CÓ ĐI TIẾP KHÔNG
// ============================================================================
//
//	compensate  chạy bước GỠ LẠI, rồi DỪNG.   Việc chính coi như KHÔNG làm.
//	fallback    chạy bước THAY THẾ, rồi ĐI TIẾP nếu nó xong.
//
// Đó là toàn bộ khác biệt về ngữ nghĩa. Mọi thứ còn lại giống hệt, và giống hệt
// là CỐ Ý: bước thay thế bị loại khỏi lịch chạy thường, được ghi `skipped` kèm
// lý do, chạy qua đúng `runStep`, biết mình đang thay cho ai qua `{{buoc_hong}}`,
// và không có đường nào để thay-thế-của-thay-thế.
//
// ============================================================================
// KẾT QUẢ CỦA BƯỚC THAY THẾ ĐỌC BẰNG TÊN BƯỚC HỎNG
// ============================================================================
//
// Đây là quyết định thiết kế đáng cãi nhất của file này, nên nói thẳng lý do.
//
//	[[flow.x.step]] id = "hoi-claude"  on_failure = "fallback"  fallback = "hoi-grok"
//	[[flow.x.step]] id = "gop"  needs = ["hoi-claude"]  prompt = "{{steps.hoi-claude.output}}"
//
// Bước `gop` khai `needs` tới `hoi-claude` và đọc kết quả của `hoi-claude`. Nếu
// kết quả của bước thay thế KHÔNG được gán vào chỗ đó thì `gop` nhận một ô
// rỗng — và với bước `shell` thì đó còn là lỗi cứng (BuocConSot). Người viết
// flow cũng không có cách nào viết cho đúng cả hai đường: `{{steps.hoi-grok.output}}`
// rỗng ở mọi lượt suôn sẻ, vì lượt suôn sẻ thì `hoi-grok` bị `skipped`.
//
// Tức là: KHÔNG gán kết quả thì `fallback` không dùng được vào việc gì. Gán thì
// nó đúng nghĩa "chạy thay". Chọn cái thứ hai.
//
// Sổ trạng thái thì VẪN GHI SỰ THẬT: `hoi-claude` là `failed`, `hoi-grok` là
// `done`. Không có dòng nào trong sổ nói `hoi-claude` xong cả — chỗ gán nằm ở
// bảng biến của lượt chạy, không nằm ở sổ.
//
// ============================================================================
// CHẠY LẠI GIỮA CHỪNG (RESUME) THÌ LẤY LẠI Ở ĐÂU
// ============================================================================
//
// Một lượt chạy có thể dừng ở rào duyệt SAU khi bước thay thế đã chạy. Lượt sau
// `Resume` dựng lại trạng thái từ SQLite, và bảng biến trong bộ nhớ thì mất.
//
// Không lưu thêm cột nào, không nhét dấu vết vào câu chữ để rồi tách chuỗi:
// dựng LẠI từ chính hai dòng sổ đã có — "bước A `failed` với on_failure =
// fallback, và bước thay thế của nó `done`" là một sự thật đủ để suy ra mọi
// thứ. ganKetQuaThayThe làm đúng việc đó, và nó được gọi ở CẢ HAI chỗ (lúc
// chạy thay xong, và lúc nạp lại sổ) nên hai đường không lệch nhau được.
package flow

import (
	"fmt"
	"sort"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// BuocThayThe là tập id các bước ĐANG LÀM NHIỆM VỤ CHẠY THAY cho một bước khác.
//
// Dùng để loại chúng khỏi lịch chạy thường — xem ghi chú ở đầu file.
func BuocThayThe(f Flow) map[string]bool {
	out := map[string]bool{}
	for _, s := range f.Steps {
		if s.OnFailure == OnFailFallback && s.Fallback != "" && s.Fallback != s.ID {
			out[s.Fallback] = true
		}
	}
	return out
}

// BuocDuocThayTheBoi trả về các bước mà `id` làm nhiệm vụ CHẠY THAY cho, đã sắp xếp.
//
// Chiều danh sách của MoTaChoThayThe: bảng chạy khan và mặt web cần nối được
// "thay cho bước nào" thành liên kết, mà tách chuỗi từ một câu tiếng Việt thì
// sớm muộn cũng sai. Một nguồn, hai dạng.
func BuocDuocThayTheBoi(f Flow, id string) []string {
	var out []string
	for _, s := range f.Steps {
		if s.OnFailure == OnFailFallback && s.Fallback == id && s.ID != id {
			out = append(out, s.ID)
		}
	}
	sort.Strings(out)
	return out
}

// MoTaChoThayThe là câu ghi vào sổ cho một bước chạy thay KHÔNG được gọi tới.
func MoTaChoThayThe(f Flow, id string) string {
	return fmt.Sprintf("bước chạy thay — chỉ chạy khi %s hỏng", nhomTen(BuocDuocThayTheBoi(f, id)))
}

// BuocNgoaiLichThuong gom MỌI bước bị loại khỏi lịch chạy thường về MỘT chỗ:
// bước gỡ lại (`compensate`) và bước chạy thay (`fallback`). Trả về id → câu
// ghi vào sổ.
//
// Một chỗ chứ không hai, và đây không phải sở thích. Mục 1.4 của báo cáo #199
// ghi lại đúng lớp lỗi này: nhánh `foreach` và nhánh thường từng tự xét
// `on_failure` riêng, chúng lệch nhau thật, và giá trị thứ tư lặng lẽ rơi vào
// nhánh sai. Bộ chạy, `flow show` và bảng chạy khan đều hỏi cùng một câu "bước
// nào sẽ không chạy" — ba nơi tự đi đếm là ba cơ hội để lệch.
func BuocNgoaiLichThuong(f Flow) map[string]string {
	out := map[string]string{}
	for id := range BuocGoLai(f) {
		out[id] = MoTaChoGoLai(f, id)
	}
	for id := range BuocThayThe(f) {
		if cu, co := out[id]; co {
			// Vừa là bước gỡ lại của người này vừa là bước chạy thay của người
			// kia. Hợp lệ, và người đọc sổ cần thấy CẢ HAI vai.
			out[id] = cu + "; và " + MoTaChoThayThe(f, id)
			continue
		}
		out[id] = MoTaChoThayThe(f, id)
	}
	return out
}

// VanDeFallback soi phần `fallback` của cả flow.
//
// Song song với VanDeCompensate và cố ý giống nó tới từng câu chữ ở những chỗ
// hai bên có cùng cái bẫy: người đọc cảnh báo không nên phải học hai bộ từ vựng
// cho hai thứ chỉ khác nhau ở chỗ "sau đó có đi tiếp không".
func VanDeFallback(f Flow) []Problem {
	var ps []Problem
	loi := func(step, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: step, Msg: msg}) }
	canh := func(step, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: step, Msg: msg, Warn: true}) }

	thayThe := BuocThayThe(f)

	aiCho := map[string][]string{}
	for _, s := range f.Steps {
		for _, n := range s.Needs {
			aiCho[n] = append(aiCho[n], s.ID)
		}
	}

	for _, s := range f.Steps {
		if s.OnFailure == OnFailFallback && s.Fallback == s.ID {
			loi(s.ID, "bước không thể tự chạy thay chính nó — nó vừa hỏng, chạy lại y nguyên "+
				"thì hỏng y nguyên")
		}
		// `fallback` khai ở một bước không dùng tới nó là một dòng chết: nó nằm
		// đó trông như có tác dụng. Trường hợp on_failure = "compensate" đã có
		// cảnh báo riêng ở VanDeCompensate, đừng nói hai lần.
		if s.Fallback != "" && s.OnFailure != OnFailFallback && s.OnFailure != OnFailCompensate {
			canh(s.ID, fmt.Sprintf("khai `fallback` nhưng on_failure = %q — `fallback` chỉ có tác dụng "+
				"khi on_failure = \"fallback\"", s.OnFailure))
		}

		if !thayThe[s.ID] {
			continue
		}

		// Từ đây trở xuống: s LÀ một bước chạy thay.
		switch s.Type {
		case TypeApprove:
			loi(s.ID, "bước chạy thay không được là `approve`: nó chạy ngoài lịch thường, ngay tại "+
				"chỗ một bước vừa hỏng — runStep không dựng rào duyệt, nên nó sẽ không dừng chờ ai cả")
		case TypeNotify:
			canh(s.ID, "bước chạy thay kiểu `notify` chỉ in ra một dòng chữ, nó KHÔNG làm thay việc gì "+
				"— và bước sau sẽ đọc chính dòng chữ đó như thể đó là kết quả của bước hỏng")
		}
		if s.OnFailure != "" {
			canh(s.ID, fmt.Sprintf("on_failure = %q của bước chạy thay bị BỎ QUA: thay-thế-của-thay-thế "+
				"là một chỗ treo vô hạn. Bước chạy thay mà hỏng thì lượt chạy DỪNG", s.OnFailure))
		}
		if len(s.Needs) > 0 {
			canh(s.ID, "`needs` của bước chạy thay bị BỎ QUA: nó chạy ngoài lịch thường, ngay lúc sự cố")
		}
		if ai := aiCho[s.ID]; len(ai) > 0 {
			loi(s.ID, fmt.Sprintf("bước %s khai `needs` tới bước chạy thay này, nhưng bước chạy thay "+
				"KHÔNG chạy ở lịch thường — %s sẽ không bao giờ tới lượt. Muốn đọc kết quả của nó thì "+
				"khai `needs` tới BƯỚC HỎNG: kết quả bước chạy thay đọc bằng tên bước hỏng",
				nhomTen(ai), nhomTen(ai)))
		}
	}
	return ps
}

// ganKetQuaThayThe gán kết quả của bước CHẠY THAY vào chỗ của bước HỎNG.
//
// Xem phần "KẾT QUẢ CỦA BƯỚC THAY THẾ ĐỌC BẰNG TÊN BƯỚC HỎNG" ở đầu file cho lý
// do, và phần "CHẠY LẠI GIỮA CHỪNG" cho lý do hàm này suy ra mọi thứ từ trạng
// thái thay vì được gọi một lần rồi nhớ.
//
// Chỉ chạm vào bảng KẾT QUẢ, không chạm vào bảng TRẠNG THÁI: bước hỏng vẫn là
// `failed`, ở cả bộ nhớ lẫn sổ.
func ganKetQuaThayThe(f Flow, st *runState) {
	for _, s := range f.Steps {
		if s.OnFailure != OnFailFallback || s.Fallback == "" || s.Fallback == s.ID {
			continue
		}
		if st.state(s.ID) != store.StepFailed || st.state(s.Fallback) != store.StepDone {
			continue
		}
		if out := st.output(s.Fallback); out != "" {
			st.datOutput(s.ID, out)
		}
	}
}
