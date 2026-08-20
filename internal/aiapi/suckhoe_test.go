package aiapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// `route.kiem` trả lời câu người vận hành hỏi TRƯỚC khi bấm một lượt flow dài:
// route này gọi bây giờ thì chạy hay hỏng.
//
// LỖI THẬT làm nên tính năng này (đo 20/08/2026): route `deepseek` trả HTTP 503
// ba lần lúc 16:54–16:56 rồi tự hồi phục. Khi chưa có lệnh kiểm, cách duy nhất
// để biết là gọi thật rồi hỏng — tức là hỏng ở giữa lượt chạy, chứ không phải
// lúc còn kịp đổi route.

// keyTam dựng HOME giả rồi đặt sẵn một key — gộp hai bước vì mọi bài dưới đây
// đều cần cả hai. (`datKey` của aiapi_test.go chỉ làm bước sau.)
func keyTam(t *testing.T, id, key string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	datKey(t, id, key)
}

// mayGia dựng một nhà cung cấp giả và trả về route trỏ vào nó.
func mayGia(t *testing.T, xuLy http.HandlerFunc) Route {
	t.Helper()
	srv := httptest.NewServer(xuLy)
	t.Cleanup(srv.Close)
	return Route{Ten: "thu", BaseURL: srv.URL, Model: "deepseek-v4-flash", KeyID: "thu"}
}

func TestRouteSongVaModelCoThat(t *testing.T) {
	keyTam(t, "thu", "k")
	r := mayGia(t, func(w http.ResponseWriter, req *http.Request) {
		// Phải đi bằng GET /models — KHÔNG được gửi prompt, vì phép kiểm mà tính
		// tiền thì người ta sẽ thôi chạy nó.
		if req.Method != http.MethodGet || !strings.HasSuffix(req.URL.Path, "/models") {
			t.Errorf("kiểm phải là GET /models, được %s %s", req.Method, req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("không gửi key: %q", req.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"deepseek-v4-flash"},{"id":"deepseek-v4-pro"}]}`))
	})

	s := Kiem(context.Background(), r)
	if !s.Song || !s.CoModel || !s.Dung() {
		t.Fatalf("route lành mà báo hỏng: %+v", s)
	}
	if s.SoModel != 2 {
		t.Errorf("đếm sai số model: %d", s.SoModel)
	}
}

// Route sống nhưng model khai không có thật KHÁC HẲN route chết: cách sửa là
// sửa cấu hình, không phải ngồi đợi. Gộp hai cái vào một chữ "hỏng" là làm mất
// đúng thông tin đáng giá.
func TestModelKhaiSaiThiKhongDungDuocDuRouteSong(t *testing.T) {
	keyTam(t, "thu", "k")
	r := mayGia(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"deepseek-v4-pro"},{"id":"grok-4.5"}]}`))
	})
	r.Model = "deepseek-chat" // tên KHÔNG tồn tại — đúng lỗi đã dính thật

	s := Kiem(context.Background(), r)
	if !s.Song {
		t.Fatal("route trả 200 mà báo chết")
	}
	if s.CoModel || s.Dung() {
		t.Error("model khai sai mà vẫn báo dùng được")
	}
	// Gợi ý phải trỏ đúng họ model, để sửa cấu hình không phải đi tra danh sách.
	if len(s.Gan) == 0 || !strings.HasPrefix(s.Gan[0], "deepseek") {
		t.Errorf("không gợi ý tên gần đúng: %v", s.Gan)
	}
	// Và KHÔNG gợi ý model của họ khác — nhiễu còn tệ hơn im lặng.
	for _, g := range s.Gan {
		if strings.HasPrefix(g, "grok") {
			t.Errorf("gợi ý lạc sang họ khác: %v", s.Gan)
		}
	}
}

func TestRouteChetGiuNguyenVanLoi(t *testing.T) {
	keyTam(t, "thu", "k")
	than := `{"error":{"message":"Service temporarily unavailable","request_id":"req_abc123"}}`
	r := mayGia(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(than))
	})

	s := Kiem(context.Background(), r)
	if s.Song || s.Dung() {
		t.Fatal("HTTP 503 mà báo dùng được")
	}
	if s.Status != http.StatusServiceUnavailable {
		t.Errorf("mất mã HTTP: %d", s.Status)
	}
	// request_id là thứ DUY NHẤT dùng được khi phải hỏi lại nhà cung cấp. Rút
	// gọn thành "lỗi 503" là vứt mất nó.
	if !strings.Contains(s.Loi, "req_abc123") {
		t.Errorf("không giữ nguyên văn thân lỗi: %q", s.Loi)
	}
}

// Một số endpoint tương thích OpenAI không cài `/models`. Im lặng KHÁC phủ
// nhận: không được kết luận "model khai sai" từ chỗ không có danh sách.
func TestEndpointKhongLietKeModelThiKhongKetLuanBua(t *testing.T) {
	keyTam(t, "thu", "k")
	r := mayGia(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	})

	s := Kiem(context.Background(), r)
	if !s.Song {
		t.Fatal("trả 200 mà báo chết")
	}
	if !s.KhongRo {
		t.Error("không đánh dấu 'không rõ' khi endpoint không liệt kê model")
	}
	if s.CoModel {
		t.Error("khẳng định model có thật trong khi không có danh sách nào để đối chiếu")
	}
	// Không rõ thì vẫn coi là dùng được: chặn người dùng chạy vì MÌNH không
	// kiểm được là đổi một khoảng trống thành một lời từ chối.
	if !s.Dung() {
		t.Error("không kiểm được tên model mà lại chặn route — im lặng bị xử như phủ nhận")
	}
}

func TestThieuKeyThiNoiThang(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	s := Kiem(context.Background(), Route{Ten: "thu", BaseURL: "http://127.0.0.1:1", Model: "m", KeyID: "khong-co"})
	if s.Song || s.Loi == "" {
		t.Errorf("thiếu key mà không báo gì: %+v", s)
	}
	// Chưa chạm mạng thì không được bịa ra mã HTTP.
	if s.Status != 0 {
		t.Errorf("bịa mã HTTP khi chưa gọi: %d", s.Status)
	}
}

func TestThieuBaseURLThiNoiThang(t *testing.T) {
	keyTam(t, "thu", "k")
	s := Kiem(context.Background(), Route{Ten: "thu", Model: "m", KeyID: "thu"})
	if s.Song || !strings.Contains(s.Loi, "base_url") {
		t.Errorf("thiếu base_url mà không nói rõ: %+v", s)
	}
}
