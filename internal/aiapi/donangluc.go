package aiapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Bộ ĐO THẬT của bảng năng lực API.
//
// Bảng trong nangluc.go là LỜI KHAI; file này là thứ sinh ra lời khai đó. Tách
// làm hai vì hai bên có giá khác hẳn nhau: đọc bảng thì miễn phí và chạy được
// mọi lúc (CLI, web, flow validate đều gọi), còn đo thì CHẠM MẠNG THẬT và TỐN
// TOKEN — nên nó chỉ chạy khi có người chủ động gõ `--do`.
//
// Vì sao bộ đo phải nằm trong repo chứ không phải một script chạy một lần rồi
// vứt: bằng chứng trong bảng chỉ đáng tin chừng nào có ai đó chạy lại được nó.
// Hôm 22/08 dự án vừa bắt quả tang một bảng quyền plugin khai `chan-that` cho
// thứ không chặn được, và lý do nó sống lâu là KHÔNG AI ĐO LẠI ĐƯỢC. Một dòng
// bằng chứng chỉ vào một lệnh gõ được thì lời khai sai chỉ sống tới lần gõ sau.
//
// Bộ đo này CỐ Ý không đi qua `Goi`/`GoiStream`: hai hàm đó gửi đúng những gì
// kiểu `yeuCau` cho phép, mà cả câu hỏi ở đây là "nhà cung cấp làm được gì",
// KHÁC câu "mã của dự án này gửi được gì". Trộn hai câu vào một phép đo là cách
// chắc chắn nhất để không bao giờ biết chỗ hỏng nằm bên nào.

// tokenToiDaKhiDo giới hạn mỗi lượt đo. Bộ đo chạm mạng thật bằng key thật, nên
// nó tiêu tiền: bảy phép đo × hai route mà mỗi lượt trả tự do thì một lần gõ
// `--do` có thể tốn hơn cả một lượt làm việc thật. Câu trả lời ở đây chỉ cần đủ
// để KẾT LUẬN (có tool_calls không, có nói "đỏ" không), không cần hay.
const tokenToiDaKhiDo = 64

// anhDoThiGiac là ảnh PNG 32×32 TOÀN MÀU ĐỎ, nhúng thẳng dạng base64 (96 byte).
//
// Vì sao không dùng ảnh 1×1 trong suốt cho gọn: phép đo thị giác phải PHÂN BIỆT
// ĐƯỢC "nhà cung cấp thật sự nhìn thấy ảnh" với "nhà cung cấp nuốt phần ảnh rồi
// vẫn trả lời tử tế". Endpoint tương thích OpenAI hay làm đúng chuyện thứ hai:
// trả HTTP 200 đẹp đẽ trong khi bỏ qua sạch phần `image_url`. Lấy 200 làm bằng
// chứng "đọc được ảnh" là khai khống.
//
// Ảnh một màu giải được chuyện đó: hỏi "ảnh này màu gì", chỉ nơi THẬT SỰ giải
// mã ảnh mới trả lời được "đỏ". Đoán mò thì xác suất trúng thấp, và phép đo ghi
// lại NGUYÊN VĂN câu trả lời nên người đọc tự kiểm lại được.
//
// VÌ SAO 32×32 CHỨ KHÔNG PHẢI 8×8: bản đầu dùng 8×8 và grok trả HTTP 400
// "Image has 64 total pixels (8x8), which is below the minimum of 512 pixels".
// Suýt nữa thì ghi vào bảng rằng grok KHÔNG đọc được ảnh — trong khi thứ vừa đo
// được là phép đo của chính mình gửi ảnh quá nhỏ. Đây đúng là kiểu ô sai mà cả
// file này dựng lên để chặn, nên nó ở lại đây thành một bình luận: một kết luận
// phủ định phải loại trừ được lỗi của người đo trước khi được ghi. 32×32 =
// 1024 điểm ảnh, gấp đôi ngưỡng, vẫn chỉ 96 byte.
const anhDoThiGiac = "iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAIAAAD8GO2jAAAAJ0lEQVR42u3NsQkAAAjAsP7/tF7hIASyp6lTCQQCgUAgEAgEgi/BAjLD/C5w/SM9AAAAAElFTkSuQmCC"

// KetQuaDo là kết quả MỘT phép đo trên MỘT route.
//
// `Chi` giữ NGUYÊN VĂN thứ quan sát được — thân lỗi của nhà cung cấp (kèm
// request id, theo luật 2 ở đầu aiapi.go) hoặc câu trả lời đã cắt ngắn. Đây là
// thứ được chép vào cột bằng chứng của bảng, nên nó phải là quan sát chứ không
// phải kết luận: "trả HTTP 400: model does not support tools" nói được nhiều
// hơn "không làm được" gấp nhiều lần.
type KetQuaDo struct {
	Route     string           `json:"route"`
	Khoa      string           `json:"khoa"`
	TrangThai TrangThaiNangLuc `json:"trang_thai"`
	Chi       string           `json:"chi"`
	Status    int              `json:"status,omitempty"`
	Usage     Usage            `json:"usage"`
	Mat       time.Duration    `json:"-"`

	// ghepChi: `Chi` đã mang sẵn một quan sát (lần đo hụt) và quan sát tiếp
	// theo phải NỐI vào chứ không đè lên. Không xuất ra ngoài — nó là chuyện
	// riêng của cách dựng câu, và ghi đè mất lần đo đầu chính là cách một bài
	// học biến mất khỏi bằng chứng.
	ghepChi bool
}

// noi ghi một quan sát vào `Chi`, nối tiếp nếu đã có quan sát trước đó.
func (k *KetQuaDo) noi(s string) {
	if k.ghepChi {
		k.Chi += s
		return
	}
	k.Chi = s
}

// DoMotNangLuc đo ĐÚNG MỘT năng lực của một route, bằng một lời gọi thật.
//
// Trả về ChuaDo — không phải KhongLamDuoc — khi phép đo không CHẠY được (thiếu
// key, mất mạng, route khai thiếu). Đây là chỗ dễ sai nhất của cả file: một
// phép đo hỏng và một kết luận phủ định trông giống hệt nhau ở đầu ra, nhưng
// chúng ngược nhau về nghĩa. Gộp lại thì mỗi lần rớt mạng là một lần bảng năng
// lực tự khai lùi, và không ai biết vì mọi ô đều có vẻ "đã đo".
func DoMotNangLuc(ctx context.Context, r Route, khoa string) (kq KetQuaDo) {
	// Trả về CÓ TÊN, vì `Mat` được đặt trong một `defer` bên dưới. Bản đầu dùng
	// `kq` thường và mọi số đo in ra `mat=0s`: giá trị trả về đã được sao chép
	// trước khi defer chạy. Một cột thời gian toàn số 0 trông như "nhanh quá
	// không đo nổi" chứ không trông như hỏng, nên nó suýt đi thẳng vào bảng.
	kq = KetQuaDo{Route: r.Ten, Khoa: khoa}
	if moTaNangLucAPI(khoa) == "" {
		kq.TrangThai, kq.Chi = ChuaDo, fmt.Sprintf("không có năng lực %q trong MoiNangLucAPI", khoa)
		return kq
	}
	if _, err := docKey(r.KeyID); err != nil {
		// Đúng câu mà đề bài đòi: lý do CỤ THỂ, không phải "chưa kiểm tra".
		kq.TrangThai, kq.Chi = ChuaDo, "chưa có key: "+err.Error()
		return kq
	}
	if r.BaseURL == "" || r.Model == "" {
		kq.TrangThai, kq.Chi = ChuaDo, fmt.Sprintf("route %q thiếu base_url hoặc model", r.Ten)
		return kq
	}

	bat := time.Now()
	defer func() { kq.Mat = time.Since(bat) }()

	switch khoa {
	case NLAPILietKeModel:
		sk := Kiem(ctx, r)
		switch {
		case sk.Song && sk.SoModel > 0:
			kq.TrangThai = LamDuoc
			kq.Chi = fmt.Sprintf("GET /models trả %d model", sk.SoModel)
		case sk.Song:
			kq.TrangThai = KhongLamDuoc
			kq.Chi = "route sống nhưng GET /models không liệt kê model nào"
		default:
			kq.TrangThai = ChuaDo
			kq.Chi = "không hỏi được /models: " + sk.Loi
		}
		kq.Status = sk.Status
		return kq
	case NLAPIStreaming:
		return doStreaming(ctx, r, kq)
	}

	than, ok := thanDo(khoa, r.Model)
	if !ok {
		kq.TrangThai, kq.Chi = ChuaDo, "chưa viết phép đo cho năng lực này"
		return kq
	}
	raw, status, err := postJSON(ctx, r, than, 90*time.Second, nil)
	kq.Status = status
	if err != nil {
		kq.TrangThai, kq.Chi = ChuaDo, "phép đo không chạy được: "+err.Error()
		return kq
	}
	// Nhà cung cấp từ chối vì `tool_choice`, KHÔNG phải vì `tools`: đo lại một
	// lần bằng `auto`.
	//
	// Đây là bài học thứ hai của cùng một loại, xin ghi lại: deepseek trả
	// HTTP 400 "Thinking mode does not support this tool_choice" — nếu dừng ở
	// đó thì bảng sẽ khai deepseek KHÔNG gọi được tool, trong khi thứ nó vừa từ
	// chối là cái CÁCH ép gọi của phép đo. Một kết luận phủ định chỉ được ghi
	// khi đã loại trừ lỗi của người đo; ở đây "loại trừ" nghĩa là hỏi lại theo
	// cách nhà cung cấp chấp nhận, rồi ghi CẢ HAI quan sát vào bằng chứng.
	if status == http.StatusBadRequest && khoa == NLAPIGoiTool &&
		strings.Contains(string(raw), "tool_choice") {
		than["tool_choice"] = "auto"
		lai, st2, err2 := postJSON(ctx, r, than, 90*time.Second, nil)
		if err2 == nil {
			kq.Chi = fmt.Sprintf("từ chối tool_choice=required (HTTP %d: %s) → đo lại với auto: ",
				status, catGon(strings.TrimSpace(string(raw)), 120))
			kq.ghepChi = true
			raw, status = lai, st2
			kq.Status = status
		}
	}
	if status != http.StatusOK {
		// Nhà cung cấp TỪ CHỐI là một kết luận thật, và thân lỗi của họ nói rõ
		// từ chối vì gì. Giữ nguyên văn, cắt cho vừa một dòng bảng.
		kq.TrangThai = KhongLamDuoc
		kq.noi(fmt.Sprintf("HTTP %d: %s", status, catGon(strings.TrimSpace(string(raw)), 220)))
		return kq
	}

	var ph phanHoiDo
	if err := json.Unmarshal(raw, &ph); err != nil {
		kq.TrangThai, kq.Chi = ChuaDo, "trả JSON không đọc được: "+err.Error()
		return kq
	}
	kq.Usage = ph.Usage
	if len(ph.Choices) == 0 {
		kq.TrangThai, kq.Chi = KhongLamDuoc, "HTTP 200 nhưng 0 lựa chọn"
		return kq
	}
	msg := ph.Choices[0].Message
	noi := strings.TrimSpace(msg.Content)

	switch khoa {
	case NLAPIGoiTool:
		if len(msg.ToolCalls) > 0 {
			kq.noi(fmt.Sprintf("trả %d tool_call, hàm đầu là %q",
				len(msg.ToolCalls), msg.ToolCalls[0].Function.Name))
			kq.TrangThai = LamDuoc
			return kq
		}
		// 200 mà không gọi tool: endpoint nhận trường `tools` nhưng không dùng
		// nó. Với người viết flow thì kết quả giống hệt không hỗ trợ — bước tool
		// sẽ không bao giờ chạy.
		kq.TrangThai = KhongLamDuoc
		kq.noi("HTTP 200 nhưng KHÔNG có tool_calls — trả chữ: " + catGon(noi, 120))
	case NLAPIDauVaoAnh:
		// Hỏi ảnh TOÀN ĐỎ. Chỉ nơi giải mã ảnh mới nói được "đỏ".
		if coTuDo(noi) {
			kq.TrangThai = LamDuoc
			kq.Chi = "nhận content dạng mảng có image_url và đọc ĐÚNG màu ảnh 32x32 đỏ, trả: " + catGon(noi, 80)
			return kq
		}
		kq.TrangThai = KhongLamDuoc
		kq.Chi = "HTTP 200 nhưng KHÔNG đọc ra màu ảnh — nuốt phần image_url. Trả: " + catGon(noi, 120)
	case NLAPIDauRaCoCauTruc:
		var thu map[string]any
		if json.Unmarshal([]byte(noi), &thu) == nil {
			if _, co := thu["mau"]; co {
				kq.TrangThai = LamDuoc
				kq.Chi = "response_format json_schema được nhận, trả JSON đúng schema: " + catGon(noi, 100)
				return kq
			}
			kq.TrangThai = KhongLamDuoc
			kq.Chi = "trả JSON nhưng THIẾU trường schema đòi (`mau`): " + catGon(noi, 120)
			return kq
		}
		kq.TrangThai = KhongLamDuoc
		kq.Chi = "HTTP 200 nhưng nội dung không phải JSON: " + catGon(noi, 120)
	case NLAPIReasoning:
		if s := strings.TrimSpace(msg.ReasoningContent); s != "" {
			kq.TrangThai = LamDuoc
			kq.Chi = fmt.Sprintf("trả trường reasoning_content dài %d ký tự", len(s))
			return kq
		}
		if s := strings.TrimSpace(msg.Reasoning); s != "" {
			kq.TrangThai = LamDuoc
			kq.Chi = fmt.Sprintf("trả trường reasoning dài %d ký tự", len(s))
			return kq
		}
		kq.TrangThai = KhongLamDuoc
		kq.Chi = "HTTP 200 nhưng không có trường reasoning_content lẫn reasoning"
	case NLAPIDemToken:
		if ph.Usage.Tong > 0 {
			kq.TrangThai = LamDuoc
			kq.Chi = fmt.Sprintf("usage thật: vào %d, ra %d, tổng %d token",
				ph.Usage.Vao, ph.Usage.Ra, ph.Usage.Tong)
			return kq
		}
		// Không có usage nghĩa là sổ chi phí của đường này ghi 0 cho mọi lượt —
		// một lỗ thủng về tiền, không phải một chi tiết kỹ thuật.
		kq.TrangThai = KhongLamDuoc
		kq.Chi = "HTTP 200 nhưng usage.total_tokens = 0 — lượt gọi không đếm được"
	}
	return kq
}

// doStreaming đo riêng vì nó không đọc một thân JSON mà đếm mẩu SSE.
//
// Đo HAI thứ trong một lượt, cố ý: có mẩu chảy về không, và mẩu cuối có `usage`
// không. Tách thành hai phép đo thì tốn gấp đôi tiền để biết cùng chừng ấy — mà
// câu trả lời cho cái sau đã nằm sẵn trong lượt của cái trước.
func doStreaming(ctx context.Context, r Route, kq KetQuaDo) KetQuaDo {
	than := map[string]any{
		"model":          r.Model,
		"messages":       []any{map[string]any{"role": "user", "content": "Đếm từ 1 tới 5."}},
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
		"max_tokens":     tokenToiDaKhiDo,
	}
	var soManh int
	var usage Usage
	var coUsage bool
	raw, status, err := postJSON(ctx, r, than, 90*time.Second, func(body io.Reader) {
		sc := bufio.NewScanner(body)
		sc.Buffer(make([]byte, 0, 64*1024), 4<<20)
		for sc.Scan() {
			dong := strings.TrimSpace(sc.Text())
			if !strings.HasPrefix(dong, "data:") {
				continue
			}
			d := strings.TrimSpace(strings.TrimPrefix(dong, "data:"))
			if d == "[DONE]" {
				break
			}
			var m manhStream
			if json.Unmarshal([]byte(d), &m) != nil {
				continue
			}
			for _, c := range m.Choices {
				if c.Delta.Content != "" {
					soManh++
				}
			}
			if m.Usage != nil && m.Usage.Tong > 0 {
				usage, coUsage = *m.Usage, true
			}
		}
	})
	kq.Status = status
	if err != nil {
		kq.TrangThai, kq.Chi = ChuaDo, "phép đo không chạy được: "+err.Error()
		return kq
	}
	if status != http.StatusOK {
		kq.TrangThai = KhongLamDuoc
		kq.Chi = fmt.Sprintf("HTTP %d: %s", status, catGon(strings.TrimSpace(string(raw)), 220))
		return kq
	}
	kq.Usage = usage
	switch {
	case soManh == 0:
		kq.TrangThai, kq.Chi = KhongLamDuoc, "HTTP 200 nhưng không có mẩu SSE nào mang chữ"
	case coUsage:
		kq.TrangThai = LamDuoc
		kq.Chi = fmt.Sprintf("%d mẩu SSE mang chữ, mẩu cuối có usage (tổng %d token)", soManh, usage.Tong)
	default:
		// Stream chạy nhưng KHÔNG có usage: chạy được mà không đếm được tiền.
		// Đây vẫn là LamDuoc cho năng lực `streaming` — nó đúng là chảy được —
		// và chỗ thiếu usage thuộc về năng lực `dem-token-that`, không được
		// trộn sang đây. `CanhBaoThieuUsage` là nơi nói chuyện đó.
		kq.TrangThai = LamDuoc
		kq.Chi = fmt.Sprintf("%d mẩu SSE mang chữ, nhưng KHÔNG mẩu nào mang usage "+
			"dù đã hỏi stream_options.include_usage", soManh)
	}
	return kq
}

// DoMoiNangLuc chạy MỌI phép đo trên một route, theo đúng thứ tự MoiNangLucAPI.
//
// Chạy TUẦN TỰ chứ không song song như `RouteKiem`: ở đó bảy lời gọi là bảy lần
// đọc `/models` không tốn gì, còn ở đây mỗi lời gọi là tiền, và bắn cả bảy cùng
// lúc vào một nhà cung cấp là cách nhanh nhất để ăn HTTP 429 rồi phải đoán xem
// route "không làm được" hay chỉ bị chặn tốc độ.
func DoMoiNangLuc(ctx context.Context, r Route) []KetQuaDo {
	ra := make([]KetQuaDo, 0, len(MoiNangLucAPI))
	for _, m := range MoiNangLucAPI {
		ra = append(ra, DoMotNangLuc(ctx, r, m.Khoa))
	}
	return ra
}

// thanDo dựng thân yêu cầu cho từng phép đo.
//
// Dùng map chứ không dùng struct có kiểu, CÓ CHỦ Ý: struct `yeuCau` của gói này
// chính là thứ đang bị đo (xem phepDoMaNguon trong nangluc.go). Đo nhà cung cấp
// bằng đúng cái struct thiếu trường thì phép đo không bao giờ hỏi được câu cần
// hỏi, và ta sẽ kết luận "nhà cung cấp không hỗ trợ" cho một thứ mình chưa gửi.
func thanDo(khoa, model string) (map[string]any, bool) {
	goc := func(noiDung any) map[string]any {
		return map[string]any{
			"model":      model,
			"messages":   []any{map[string]any{"role": "user", "content": noiDung}},
			"max_tokens": tokenToiDaKhiDo,
		}
	}
	switch khoa {
	case NLAPIGoiTool:
		t := goc("Bây giờ là mấy giờ ở Hà Nội?")
		t["tools"] = []any{map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        "lay_gio",
				"description": "Lấy giờ hiện tại ở một thành phố",
				"parameters": map[string]any{
					"type":       "object",
					"properties": map[string]any{"thanh_pho": map[string]any{"type": "string"}},
					"required":   []any{"thanh_pho"},
				},
			},
		}}
		// `required` chứ không phải `auto`: với `auto` thì model được quyền trả
		// lời bằng chữ, và ta không phân biệt nổi "không hỗ trợ tool" với "hỗ
		// trợ nhưng lần này thấy không cần gọi".
		t["tool_choice"] = "required"
		return t, true
	case NLAPIDauVaoAnh:
		return goc([]any{
			map[string]any{"type": "text", "text": "Ảnh này màu gì? Trả lời đúng một từ tiếng Việt."},
			map[string]any{"type": "image_url", "image_url": map[string]any{
				"url": "data:image/png;base64," + anhDoThiGiac,
			}},
		}), true
	case NLAPIDauRaCoCauTruc:
		t := goc("Màu của lá cờ Việt Nam là gì?")
		t["response_format"] = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "mau_sac",
				"strict": true,
				"schema": map[string]any{
					"type":                 "object",
					"properties":           map[string]any{"mau": map[string]any{"type": "string"}},
					"required":             []any{"mau"},
					"additionalProperties": false,
				},
			},
		}
		return t, true
	case NLAPIReasoning:
		// Câu hỏi phải ĐỦ KHÓ để model có lý do suy nghĩ. Hỏi "1+1" thì model
		// biết suy luận vẫn có thể trả lời thẳng, và ta kết luận nhầm là nó
		// không có reasoning.
		t := goc("Một cái ao có bèo, mỗi ngày bèo phủ gấp đôi. Ngày 30 phủ kín ao. " +
			"Ngày nào phủ nửa ao? Trả lời ngắn.")
		t["max_tokens"] = 512 // reasoning ăn token đầu ra; cắt 64 là cắt mất chỗ cần nhìn
		return t, true
	case NLAPIDemToken:
		return goc("Nói đúng một từ: xong."), true
	}
	return nil, false
}

// postJSON gửi một thân JSON tuỳ ý tới /chat/completions của route.
//
// `doc` khác nil thì thân phản hồi được giao cho nó đọc (dùng cho SSE) và hàm
// trả về raw rỗng — không gom cả stream vào bộ nhớ chỉ để vứt đi.
func postJSON(ctx context.Context, r Route, than map[string]any, hanCho time.Duration,
	doc func(io.Reader)) ([]byte, int, error) {
	key, err := docKey(r.KeyID)
	if err != nil {
		return nil, 0, err
	}
	b, err := json.Marshal(than)
	if err != nil {
		return nil, 0, err
	}
	url := strings.TrimRight(r.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	if doc != nil {
		req.Header.Set("Accept", "text/event-stream")
	}
	resp, err := (&http.Client{Timeout: hanCho}).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if doc != nil && resp.StatusCode == http.StatusOK {
		doc(resp.Body)
		return nil, resp.StatusCode, nil
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return raw, resp.StatusCode, nil
}

// phanHoiDo đọc RỘNG hơn `phanHoi` của aiapi.go: thêm tool_calls và hai cách
// đặt tên trường reasoning mà các endpoint tương thích OpenAI đang dùng lẫn lộn.
//
// Không sửa `phanHoi` cho gọn: gói này đọc `phanHoi` để dựng KetQua trả cho
// người dùng, và thêm trường vào đó là hứa rằng lõi biết dùng chúng. Bộ đo chỉ
// cần NHÌN THẤY, chưa cần dùng.
type phanHoiDo struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Reasoning        string `json:"reasoning"`
			ToolCalls        []struct {
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

// coTuDo bắt câu trả lời nói ảnh màu đỏ, chấp cả tiếng Anh và cả câu dài.
func coTuDo(s string) bool {
	t := strings.ToLower(s)
	for _, tu := range []string{"đỏ", "do do", "red"} {
		if strings.Contains(t, tu) {
			return true
		}
	}
	return false
}

// catGon cắt chuỗi cho vừa một dòng bảng, và nói rõ là đã cắt.
func catGon(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
