package aiapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Streaming là ô ⬜ cuối cùng của Pha 4, và kế hoạch ghi rõ vì sao nó bị hoãn:
// "làm nửa vời thì mất usage, mà usage mới là thứ đáng giá nhất của đường API".
//
// Nỗi lo đó CÓ THẬT: ở chế độ stream, endpoint tương thích OpenAI không gửi
// `usage` trừ khi được hỏi bằng `stream_options.include_usage`. Nhóm bài dưới
// đây canh đúng chỗ đó — cùng vài chỗ khác mà một bản streaming cẩu thả hay
// đánh rơi.

// mayStream dựng nhà cung cấp giả phát SSE, và trả về thân request nó nhận được.
func mayStream(t *testing.T, manh []string) (Route, *string) {
	t.Helper()
	var than string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		than = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		for _, m := range manh {
			fmt.Fprintf(w, "data: %s\n\n", m)
			if fl != nil {
				fl.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return Route{Ten: "thu", BaseURL: srv.URL, Model: "m-1", KeyID: "thu"}, &than
}

func manhChu(s string) string {
	return `{"model":"m-1","choices":[{"delta":{"content":` + strconv(s) + `}}]}`
}

func strconv(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// Bài quan trọng nhất: yêu cầu PHẢI hỏi usage. Thiếu dòng này thì stream chạy
// đẹp và sổ chi phí ghi 0 — kiểu hỏng im lặng tệ nhất, vì mọi thứ trông vẫn ổn.
func TestYeuCauStreamPhaiHoiUsage(t *testing.T) {
	keyTam(t, "thu", "k")
	r, than := mayStream(t, []string{manhChu("xin chao")})

	if _, err := GoiStream(context.Background(), r, "hoi", nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(*than, `"stream":true`) {
		t.Errorf("không bật stream trong yêu cầu: %s", *than)
	}
	if !strings.Contains(*than, `"include_usage":true`) {
		t.Fatalf("KHÔNG hỏi usage — stream sẽ chạy đẹp và sổ chi phí ghi 0: %s", *than)
	}
}

func TestStreamGhepDuChuVaGoiNhanTungManh(t *testing.T) {
	keyTam(t, "thu", "k")
	r, _ := mayStream(t, []string{
		manhChu("Xin "), manhChu("chào "), manhChu("bạn"),
		`{"choices":[],"usage":{"prompt_tokens":7,"completion_tokens":3,"total_tokens":10}}`,
	})

	var manh []string
	k, err := GoiStream(context.Background(), r, "hoi", func(s string) { manh = append(manh, s) })
	if err != nil {
		t.Fatal(err)
	}
	if k.NoiDung != "Xin chào bạn" {
		t.Errorf("ghép chữ sai: %q", k.NoiDung)
	}
	// Gọi callback từng mẩu mới là điểm của streaming. Gom hết rồi gọi một lần
	// thì người dùng vẫn nhìn màn hình đứng im — đúng thứ tính năng này để chữa.
	if len(manh) != 3 {
		t.Errorf("muốn 3 lần gọi nhận, được %d: %v", len(manh), manh)
	}
	if k.Usage.Tong != 10 || k.Usage.Vao != 7 {
		t.Errorf("mất usage: %+v", k.Usage)
	}
	if k.ThieuUsage {
		t.Error("có usage mà vẫn báo thiếu")
	}
	if !k.DaStreaming {
		t.Error("không đánh dấu lượt này đi đường stream")
	}
}

// Nhà cung cấp KHÔNG trả usage: phải NÓI RA, không im lặng ghi 0.
//
// `Usage.Tong == 0` một mình không đủ để phân biệt — một lượt thật cũng có thể
// tốn 0 token nếu hỏng sớm. Cờ riêng mới tách được "không tốn gì" khỏi "không
// đếm được".
func TestThieuUsageThiPhaiNoiRa(t *testing.T) {
	keyTam(t, "thu", "k")
	r, _ := mayStream(t, []string{manhChu("chào")})

	k, err := GoiStream(context.Background(), r, "hoi", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !k.ThieuUsage {
		t.Fatal("nhà cung cấp không trả usage mà không đánh dấu — sổ chi phí sẽ ghi 0 như thật")
	}
	canh := CanhBaoThieuUsage(k)
	if canh == "" {
		t.Fatal("không có câu cảnh báo nào cho người dùng")
	}
	// Câu cảnh báo phải nói rõ đây là CHƯA ĐO, không phải miễn phí.
	if !strings.Contains(canh, "CHƯA ĐO") {
		t.Errorf("cảnh báo không phân biệt chưa-đo với miễn-phí: %q", canh)
	}
	if CanhBaoThieuUsage(KetQua{}) != "" {
		t.Error("không thiếu usage mà vẫn cảnh báo — cảnh báo thừa sẽ bị đọc lướt")
	}
}

// Mẩu hỏng KHÔNG được giết cả lượt: phần đã nhận vẫn dùng được, và nhà cung cấp
// thỉnh thoảng chèn mẩu giữ nhịp không theo schema.
func TestManhHongKhongGietCaLuot(t *testing.T) {
	keyTam(t, "thu", "k")
	r, _ := mayStream(t, []string{
		manhChu("phan mot "), "{khong-phai-json", ": ghi chu SSE",
		manhChu("phan hai"),
		`{"choices":[],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`,
	})

	k, err := GoiStream(context.Background(), r, "hoi", nil)
	if err != nil {
		t.Fatal(err)
	}
	if k.NoiDung != "phan mot phan hai" {
		t.Errorf("mẩu hỏng làm mất chữ: %q", k.NoiDung)
	}
	if k.Usage.Tong != 3 {
		t.Errorf("mẩu hỏng làm mất usage: %+v", k.Usage)
	}
}

// Lỗi HTTP ở chế độ stream phải giữ NGUYÊN VĂN thân lỗi, y như `Goi` — request
// id của nhà cung cấp nằm trong đó và là thứ duy nhất dùng được khi phải hỏi lại.
func TestStreamLoiHTTPGiuNguyenVan(t *testing.T) {
	keyTam(t, "thu", "k")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited","request_id":"req_xyz789"}}`))
	}))
	defer srv.Close()

	_, err := GoiStream(context.Background(), Route{Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: "thu"}, "hoi", nil)
	if err == nil {
		t.Fatal("HTTP 429 mà không báo lỗi")
	}
	if !strings.Contains(err.Error(), "req_xyz789") {
		t.Errorf("mất request id: %v", err)
	}
	// 429 là lỗi phía NHÀ CUNG CẤP — route dự phòng có thể cứu, nên không được
	// xếp vào lỗi người dùng.
	if LoiNguoiDung(err) {
		t.Error("429 bị xếp là lỗi người dùng — bộ chuyển route sẽ không cứu")
	}
}

// 401 thì NGƯỢC LẠI: key của route NÀY sai, đổi route chỉ tốn thêm tiền.
func TestStream401LaLoiNguoiDung(t *testing.T) {
	keyTam(t, "thu", "k")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer srv.Close()

	_, err := GoiStream(context.Background(), Route{Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: "thu"}, "hoi", nil)
	if err == nil || !LoiNguoiDung(err) {
		t.Errorf("401 phải là lỗi người dùng để không fallback vô ích: %v", err)
	}
}

// Stream kết thúc mà không có chữ nào là HỎNG, không phải "câu trả lời rỗng".
// Trả về rỗng kèm nil sẽ được ghi vào sổ như một lượt thành công.
func TestStreamRongLaHong(t *testing.T) {
	keyTam(t, "thu", "k")
	r, _ := mayStream(t, nil)
	if _, err := GoiStream(context.Background(), r, "hoi", nil); err == nil {
		t.Error("stream không có chữ nào mà vẫn báo thành công")
	}
}

func TestStreamPromptRongChanTruocKhiChamMang(t *testing.T) {
	keyTam(t, "thu", "k")
	_, err := GoiStream(context.Background(), Route{Ten: "thu", BaseURL: "http://127.0.0.1:1", Model: "m", KeyID: "thu"}, "   ", nil)
	if err == nil || !LoiNguoiDung(err) {
		t.Errorf("prompt rỗng phải bị chặn trước khi gửi, và là lỗi người dùng: %v", err)
	}
}
