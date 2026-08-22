package aiapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ĐẦU RA CÓ CẤU TRÚC — ép câu trả lời theo JSON schema.
//
// # VÌ SAO ĐÂY KHÔNG PHẢI "CHỈ THÊM MỘT TRƯỜNG VÀO `yeuCau`"
//
// Thêm `response_format` vào `yeuCau` là xong CHIỀU ĐI, và bảng năng lực sẽ lật
// xanh ngay — nó dò phía dự án bằng reflection. Nhưng chiều VỀ mới là chỗ tính
// năng này sống hay chết, vì cách hỏng thường gặp nhất KHÔNG phải HTTP 400:
//
//	nhà cung cấp NHẬN trường `response_format`, trả HTTP 200, rồi trả về văn xuôi.
//
// Lúc đó `kq.NoiDung` là một câu tiếng Việt tử tế, `Usage` đẹp, không lỗi nào.
// Bước sau của flow gọi `json.Unmarshal` và hỏng — cách nơi gây ra lỗi vài
// bước, sau khi đã tiêu token của cả chuỗi. Đúng cái hình dạng "nuốt lặng lẽ"
// mà phép đo thị giác (donangluc.go) dựng ảnh một màu để chặn.
//
// Nên gói này KHÔNG chỉ gửi schema đi. Nó còn ĐỌC LẠI câu trả lời và nói ra
// mình đang ở ca nào — `DocCoCauTruc`. Ba ca, ba việc phải làm khác nhau:
//
//	(a) JSON đúng schema        → dùng được ngay
//	(b) JSON nhưng THIẾU khoá   → nhà cung cấp nhận `response_format` mà không
//	                              ép `required`; đừng tin schema, phải tự kiểm
//	(c) không phải JSON         → nhà cung cấp NUỐT `response_format`; route này
//	                              không dùng được cho bước cần JSON
//
// Gộp (b) và (c) thành một chữ "hỏng" là mất đúng thứ nói cho biết nên sửa
// prompt hay nên đổi route.

// DangTraLoi là trường `response_format` của giao thức tương thích OpenAI.
//
// Con trỏ ở `yeuCau` chứ không phải giá trị, và `omitempty` đi kèm: lượt gọi
// thường PHẢI gửi thân JSON y hệt như trước khi có file này. Cùng lý do đã viết
// cho `Tools` — deepseek-v4-flash đã trả HTTP 400 vì những thứ nhỏ hơn thế.
type DangTraLoi struct {
	Loai string    `json:"type"`
	SoDo *SoDoJSON `json:"json_schema,omitempty"`
}

// SoDoJSON là phần `json_schema`.
//
// `So` để nguyên `map[string]any` vì đó là thứ người gọi lấy thẳng từ file JSON
// của họ — y hệt `HamTool.Tham` (tool.go), và vì đúng một lý do: JSON Schema là
// thứ người ta chép qua chép lại giữa các dự án, và một lớp kiểu trung gian là
// chỗ `required` rụng mất mà không ai thấy.
type SoDoJSON struct {
	Ten string `json:"name"`

	// Nghiem là `strict`. Đo 22/08: grok-4.5 nhận `strict:true` và trả đúng
	// schema {"mau":"đỏ"}, 980 token.
	Nghiem bool           `json:"strict,omitempty"`
	So     map[string]any `json:"schema"`
}

// SoDoNghiem dựng `response_format` kiểu json_schema với `strict:true`.
//
// Có mặt để chuỗi `"json_schema"` chỉ được gõ ĐÚNG MỘT LẦN trong dự án: gõ sai
// nó ở một chỗ gọi thì nhà cung cấp bỏ qua cả trường, trả HTTP 200 kèm văn
// xuôi, và triệu chứng giống hệt "route này không hỗ trợ".
func SoDoNghiem(ten string, so map[string]any) *DangTraLoi {
	return &DangTraLoi{Loai: "json_schema", SoDo: &SoDoJSON{Ten: ten, Nghiem: true, So: so}}
}

// KhoiCoCauTruc là câu trả lời ĐÃ ĐỌC LẠI, sẵn sàng hiện ra.
//
// Có thẻ JSON vì cùng lý do của `KhoiSuyLuan`: một nguồn cho cả CLI lẫn mặt
// web, để hai mặt không thể nói hai nghĩa về cùng một câu trả lời.
type KhoiCoCauTruc struct {
	// Co: câu trả lời ĐỌC ĐƯỢC thành JSON.
	Co  bool           `json:"co"`
	Gia map[string]any `json:"gia"`

	// Thieu là những khoá `required` của schema mà câu trả lời KHÔNG có.
	// Khác rỗng nghĩa là ca (b) ở đầu file: đúng JSON, sai hợp đồng.
	Thieu []string `json:"thieu"`

	// PhaiGoRao: câu trả lời bọc trong ```json … ```, phải gỡ mới đọc được.
	//
	// Là một QUAN SÁT chứ không phải chuyện vặt: model bọc rào nghĩa là nó đang
	// làm theo prompt, không phải theo `response_format`. Tức là nhà cung cấp
	// nhiều khả năng đã nuốt trường đó, và lượt sau với prompt khác sẽ hỏng.
	PhaiGoRao bool `json:"phai_go_rao"`

	// ViSaoKhong chỉ có khi Co == false, và KHÔNG BAO GIỜ được rỗng lúc đó.
	ViSaoKhong string `json:"vi_sao_khong"`

	// DanToi là lệnh gõ được để tra lại route này có ép được JSON không.
	DanToi string `json:"dan_toi"`
}

// DungDuocNgay nói khối này dùng được làm dữ liệu cho bước sau hay không:
// đọc được thành JSON VÀ không thiếu khoá bắt buộc nào.
func (k KhoiCoCauTruc) DungDuocNgay() bool { return k.Co && len(k.Thieu) == 0 }

// DocCoCauTruc đọc lại câu trả lời của MỘT lượt gọi theo schema đã gửi đi.
//
// KHÔNG chạm mạng và KHÔNG tốn token.
//
// `dang` có thể nil (lượt gọi không gửi schema) — lúc đó vẫn cố đọc JSON, nhưng
// không có `required` nào để đối chiếu. Ca đó có thật: người dùng viết prompt
// "trả về JSON" mà không dùng `--so-do`.
func DocCoCauTruc(kq KetQua, dang *DangTraLoi) KhoiCoCauTruc {
	k := KhoiCoCauTruc{DanToi: lenhNangLuc(kq.Route)}
	noi := strings.TrimSpace(kq.NoiDung)
	if noi == "" {
		k.ViSaoKhong = "câu trả lời rỗng — không có gì để đọc thành JSON. " +
			"Lượt này vẫn tính tiền, xem dòng token bên dưới."
		return k
	}

	than := noi
	if s, daGo := goRaoMa(noi); daGo {
		than, k.PhaiGoRao = s, true
	}
	var gia map[string]any
	if err := json.Unmarshal([]byte(than), &gia); err != nil {
		// Ca (c): nhà cung cấp NUỐT `response_format`. Nói thẳng ra, vì triệu
		// chứng của nó giống hệt "model trả lời không hay" — mà việc phải làm
		// thì khác hẳn: đổi route, không phải sửa prompt.
		k.ViSaoKhong = fmt.Sprintf("câu trả lời KHÔNG phải JSON (%s). Nhà cung cấp trả "+
			"HTTP 200 nhưng nhiều khả năng đã NUỐT `response_format` — bước sau gọi "+
			"json.Unmarshal sẽ hỏng, cách chỗ gây lỗi vài bước. Nguyên văn: %s",
			err.Error(), catGon(noi, 160))
		return k
	}
	k.Co, k.Gia = true, gia
	k.Thieu = thieuKhoaBatBuoc(gia, dang)
	return k
}

// MoTaCoCauTruc dựng câu MỌI MẶT cùng nói về một khối đầu ra có cấu trúc.
//
// Tách ra vì đúng lý do của `MoTaLoiGoiTool`: câu này mang một sự thật dễ hiểu
// nhầm — "HTTP 200 mà vẫn không dùng được" — nên để mỗi mặt tự chế một cách
// diễn đạt là cách chắc chắn để có một mặt quên nói vế đó.
func MoTaCoCauTruc(k KhoiCoCauTruc) string {
	switch {
	case !k.Co:
		return "câu trả lời KHÔNG đọc được thành JSON: " + k.ViSaoKhong
	case len(k.Thieu) > 0:
		return fmt.Sprintf("đọc được JSON nhưng THIẾU %d khoá schema đòi: %s — "+
			"nhà cung cấp nhận `response_format` mà không ép `required`, nên đừng tin "+
			"schema, phải tự kiểm", len(k.Thieu), strings.Join(k.Thieu, ", "))
	case k.PhaiGoRao:
		return fmt.Sprintf("đọc được JSON đúng schema (%d khoá), NHƯNG phải gỡ rào ```json "+
			"mới đọc được — model đang làm theo prompt chứ chưa chắc theo `response_format`",
			len(k.Gia))
	default:
		return fmt.Sprintf("đọc được JSON đúng schema, %d khoá", len(k.Gia))
	}
}

// thieuKhoaBatBuoc so câu trả lời với danh sách `required` của schema.
//
// Trả nil khi không có schema hoặc schema không khai `required` — KHÔNG trả về
// "đủ", vì hai chuyện đó khác nhau: không có hợp đồng thì không kết luận được
// là hợp đồng đã được giữ.
func thieuKhoaBatBuoc(gia map[string]any, dang *DangTraLoi) []string {
	if dang == nil || dang.SoDo == nil || dang.SoDo.So == nil {
		return nil
	}
	raw, co := dang.SoDo.So["required"]
	if !co {
		return nil
	}
	var can []string
	switch v := raw.(type) {
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok {
				can = append(can, s)
			}
		}
	case []string:
		can = append(can, v...)
	}
	var thieu []string
	for _, ten := range can {
		if _, co := gia[ten]; !co {
			thieu = append(thieu, ten)
		}
	}
	sort.Strings(thieu)
	return thieu
}

// goRaoMa gỡ rào ```json … ``` quanh câu trả lời, nếu có.
//
// Trả về (chuỗi đã gỡ, có gỡ hay không). Cờ thứ hai được GIỮ LẠI chứ không nuốt
// đi: nó là bằng chứng nhà cung cấp có thể đã bỏ qua `response_format`, và giấu
// nó đi là làm ca (b)/(c) trông giống ca (a).
func goRaoMa(s string) (string, bool) {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return s, false
	}
	t = strings.TrimPrefix(t, "```")
	if i := strings.IndexByte(t, '\n'); i >= 0 {
		// Bỏ nhãn ngôn ngữ ở dòng đầu (```json, ```JSON, ```…).
		if nhan := strings.TrimSpace(t[:i]); !strings.Contains(nhan, "{") &&
			!strings.Contains(nhan, "[") {
			t = t[i+1:]
		}
	}
	if i := strings.LastIndex(t, "```"); i >= 0 {
		t = t[:i]
	}
	t = strings.TrimSpace(t)
	if t == "" {
		return s, false
	}
	return t, true
}
