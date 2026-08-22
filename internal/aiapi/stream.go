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

// GoiStream giống Goi, nhưng gọi `nhan` mỗi khi có thêm chữ.
//
// VÌ SAO CÓ: một lượt `grok-4.5` đo được 13,6 giây (docs/DO-LUONG.md). Không có
// streaming thì người dùng nhìn màn hình đứng im 13 giây rồi mới thấy cả cục —
// không phân biệt được "đang nghĩ" với "đã treo".
//
// VÌ SAO MÃI MỚI CÓ: kế hoạch Pha 4 cố ý dừng, ghi rõ "làm nửa vời thì mất
// `usage`, mà `usage` mới là thứ đáng giá nhất của đường API". Lo đó có thật:
// ở chế độ stream, endpoint tương thích OpenAI KHÔNG gửi `usage` trừ khi được
// hỏi bằng `stream_options.include_usage`. Bỏ qua nó là đổi lấy chữ chạy sớm
// bằng cách vứt bỏ sổ chi phí — mà cả bảng `api_calls` (migration v7) dựng lên
// để đếm đúng thứ đó.
//
// Nên hàm này KHÔNG bao giờ im lặng nuốt mất usage: nhà cung cấp không trả thì
// `KetQua.ThieuUsage` bật lên, và bên gọi nói ra.
func GoiStream(ctx context.Context, r Route, prompt string, nhan func(string)) (KetQua, error) {
	var kq KetQua
	if strings.TrimSpace(prompt) == "" {
		return kq, loiNguoi(r.Ten, 0, "prompt rỗng — không gửi gì cho %s", r.Ten)
	}
	key, err := docKey(r.KeyID)
	if err != nil {
		return kq, loiNguoi(r.Ten, 0, "%s: %s", r.Ten, err.Error())
	}
	if r.BaseURL == "" || r.Model == "" {
		return kq, loiMay(r.Ten, 0, "route %q thiếu base_url hoặc model", r.Ten)
	}

	body, err := json.Marshal(yeuCauStream{
		Model:    r.Model,
		Messages: []tinNhan{{Role: "user", Content: prompt}},
		Stream:   true,
		// Đây là dòng giữ lại sổ chi phí. Thiếu nó thì stream chạy đẹp và
		// `usage` về 0 — kiểu hỏng im lặng tệ nhất: mọi thứ trông vẫn ổn.
		StreamOptions: &tuyChonStream{IncludeUsage: true},
	})
	if err != nil {
		return kq, loiMay(r.Ten, 0, "%s: %s", r.Ten, err.Error())
	}

	url := strings.TrimRight(r.BaseURL, "/") + "/chat/completions"
	taoReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Accept", "text/event-stream")
		return req, nil
	}

	bat := time.Now()
	// KHÔNG đặt Timeout trên Client như `Goi`: ở đó 120 giây tính cho cả lời gọi,
	// còn ở đây một stream dài hợp lệ có thể vượt mốc đó mà vẫn đang chảy đều.
	// Hạn chót của cả lượt là việc của ctx — bên gọi quyết định, không phải bên này.
	//
	// 429 đi qua CÙNG lớp chờ-rồi-thử-lại với `Goi`, và ở đây nó an toàn vì một
	// lý do cụ thể: 429 nằm ở DÒNG TRẠNG THÁI, tức trước mẩu SSE đầu tiên, nên
	// lúc thử lại chưa có ký tự nào lọt qua `nhan`. Nếu stream đứt ở giữa thì
	// tuyệt đối KHÔNG được thử lại kiểu này — sẽ phát lại từ đầu và người dùng
	// nhận một câu trả lời ghép từ hai lượt khác nhau. Vòng đọc SSE bên dưới cố
	// ý nằm NGOÀI lớp thử lại.
	resp, choLai, err := goiCoChoLai(ctx, &http.Client{}, taoReq)
	kq.ChoLai = choLai
	if err != nil {
		return kq, &LoiAPI{Route: r.Ten, Chi: fmt.Sprintf("gọi %s hỏng: %s%s",
			url, err.Error(), themChoLai(choLai)), ChoLai: choLai}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Lỗi thì thân KHÔNG phải SSE — đọc thẳng và giữ NGUYÊN VĂN, y như `Goi`:
		// request id của nhà cung cấp nằm trong đó.
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		e := loiMay(r.Ten, resp.StatusCode, "%s trả HTTP %d: %s%s",
			r.Ten, resp.StatusCode, strings.TrimSpace(string(raw)), themChoLai(choLai))
		e.ChoLai = choLai
		e.Nguoi = resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden
		return kq, e
	}

	var chu strings.Builder
	var nghi strings.Builder
	var model string
	var usage Usage
	var coUsage bool

	sc := bufio.NewScanner(resp.Body)
	// Một mẩu SSE có thể dài hơn 64KB mặc định của Scanner — gặp là nó dừng SỚM
	// và ta tưởng stream kết thúc bình thường, mất phần đuôi lẫn usage.
	sc.Buffer(make([]byte, 0, 64*1024), 4<<20)

	for sc.Scan() {
		dong := strings.TrimSpace(sc.Text())
		if dong == "" || !strings.HasPrefix(dong, "data:") {
			// Dòng trống ngăn cách sự kiện; dòng `event:`/`:` là ghi chú SSE.
			continue
		}
		than := strings.TrimSpace(strings.TrimPrefix(dong, "data:"))
		if than == "[DONE]" {
			break
		}
		var m manhStream
		if json.Unmarshal([]byte(than), &m) != nil {
			// Một mẩu hỏng KHÔNG được giết cả lượt: phần đã nhận vẫn dùng được,
			// và nhà cung cấp thỉnh thoảng chèn mẩu giữ nhịp không theo schema.
			continue
		}
		if m.Model != "" {
			model = m.Model
		}
		// Mẩu mang usage thường là mẩu CUỐI và có `choices` rỗng.
		if m.Usage != nil && m.Usage.Tong > 0 {
			usage, coUsage = *m.Usage, true
		}
		for _, c := range m.Choices {
			// Phần nghĩ gom RIÊNG và KHÔNG đẩy qua `nhan`: `nhan` là chỗ chữ
			// chảy ra màn hình như câu trả lời, và bước sau của một flow nhét
			// `{{steps.x.output}}` sang bước kế. Trộn phần nháp của model vào
			// đó là đưa bản nháp đi làm dữ liệu đầu vào.
			if s := c.Delta.SuyLuan; s != "" {
				nghi.WriteString(s)
			}
			if s := c.Delta.SuyLuan2; s != "" {
				nghi.WriteString(s)
			}
			if c.Delta.Content == "" {
				continue
			}
			chu.WriteString(c.Delta.Content)
			if nhan != nil {
				nhan(c.Delta.Content)
			}
		}
	}
	if err := sc.Err(); err != nil {
		// Đứt giữa chừng: trả phần đã nhận KÈM lỗi. Vứt đi phần đã tốn tiền để
		// lấy một thông điệp gọn là đổi sai chiều.
		kq = KetQua{NoiDung: chu.String(), SuyLuan: strings.TrimSpace(nghi.String()),
			Model: model, Usage: usage,
			Mat: time.Since(bat), Route: r.Ten, DaThu: []string{r.Ten}, ChoLai: choLai}
		return kq, loiMay(r.Ten, resp.StatusCode, "%s: stream đứt giữa chừng sau %d ký tự: %s",
			r.Ten, chu.Len(), err.Error())
	}

	kq = KetQua{
		NoiDung:     strings.TrimSpace(chu.String()),
		SuyLuan:     strings.TrimSpace(nghi.String()),
		Model:       model,
		Usage:       usage,
		Mat:         time.Since(bat),
		Route:       r.Ten,
		DaThu:       []string{r.Ten},
		ThieuUsage:  !coUsage,
		DaStreaming: true,
		ChoLai:      choLai,
	}
	if kq.Model == "" {
		kq.Model = r.Model
	}
	if kq.NoiDung == "" {
		return kq, loiMay(r.Ten, resp.StatusCode, "%s: stream kết thúc mà không có chữ nào", r.Ten)
	}
	return kq, nil
}

type yeuCauStream struct {
	Model         string         `json:"model"`
	Messages      []tinNhan      `json:"messages"`
	Stream        bool           `json:"stream"`
	StreamOptions *tuyChonStream `json:"stream_options,omitempty"`
}

type tuyChonStream struct {
	IncludeUsage bool `json:"include_usage"`
}

// manhStream là một mẩu SSE. `Usage` là con trỏ để phân biệt "nhà cung cấp gửi
// usage bằng 0" với "mẩu này không có trường usage" — hai chuyện khác nhau, và
// gộp lại thì `ThieuUsage` sẽ nói dối.
type manhStream struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`

			// Phần NGHĨ cũng chảy về theo mẩu, ở một trường riêng. Hai tên vì
			// các endpoint tương thích OpenAI đang dùng lẫn lộn — bộ đo trong
			// donangluc.go đã phải nhận cả hai, và đường stream không có lý do
			// gì hẹp hơn.
			//
			// VÌ SAO PHẢI ĐỌC Ở ĐÂY, không chỉ ở `Goi`: mặt web hỏi AI BẰNG
			// STREAM, luôn luôn. Bỏ trống chỗ này thì mọi lượt trên dashboard
			// đều không có phần nghĩ, và câu giải thích của `DocSuyLuan` sẽ đổ
			// tội cho nhà cung cấp về một thứ chính ta không thèm đọc.
			SuyLuan  string `json:"reasoning_content"`
			SuyLuan2 string `json:"reasoning"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *Usage `json:"usage"`
}

// CanhBaoThieuUsage trả về câu cần nói với người dùng, hoặc "" nếu không có gì.
//
// Tách ra để cả CLI lẫn mặt web nói CÙNG một câu — usage thiếu là chuyện tiền
// bạc, không phải chi tiết kỹ thuật, nên mỗi mặt tự chế một cách diễn đạt là
// cách để hai mặt nói hai nghĩa.
func CanhBaoThieuUsage(k KetQua) string {
	if !k.ThieuUsage {
		return ""
	}
	return fmt.Sprintf("%s không trả `usage` ở chế độ stream — lượt này KHÔNG đếm được "+
		"token và chi phí. Sổ `sagent api --lich-su` sẽ ghi 0, đó là CHƯA ĐO chứ không "+
		"phải miễn phí.", k.Route)
}
