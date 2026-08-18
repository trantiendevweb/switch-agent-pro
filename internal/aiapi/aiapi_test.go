package aiapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func homeGia(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}
	t.Setenv("HOME", tmp)
	return tmp
}

func datKey(t *testing.T, id, key string) {
	t.Helper()
	if err := os.MkdirAll(KeysDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(KeysDir(), id+".key"), []byte(key), 0o600); err != nil {
		t.Fatal(err)
	}
}

// key_id đến từ FILE CẤU HÌNH — thứ người khác sửa được — và nó dùng để mở một
// file bí mật. Đúng lớp lỗi đã nổ một lần với tên hồ sơ và xoá mất ~/.claude.
func TestKeyIDKhongDuocThoatThuMuc(t *testing.T) {
	home := homeGia(t)

	// Đặt một "bí mật của người khác" ĐÚNG chỗ mà key_id độc hại trỏ tới. Không
	// có bước này thì test rỗng nghĩa: đường dẫn thoát ra vốn không tồn tại nên
	// ReadFile hỏng dù có lá chắn hay không, và test xanh vì lý do sai. (Đã dẫm
	// đúng chỗ đó khi viết test này.)
	const biMat = "sk-BI-MAT-CUA-NGUOI-KHAC"
	ngoai := filepath.Join(home, "ngoai-kho")
	if err := os.MkdirAll(ngoai, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ngoai, "trom.key"), []byte(biMat), 0o600); err != nil {
		t.Fatal(err)
	}

	// Server ghi lại mọi Authorization nhận được — lá chắn thủng thì bí mật hiện ra đây.
	var nhanDuoc []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nhanDuoc = append(nhanDuoc, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	for _, x := range []string{
		"../ngoai-kho/trom",
		"..\ngoai-kho\trom",
		"../../ngoai-kho/trom",
		"a/b", "", ".", "..",
	} {
		if _, err := Goi(context.Background(), Route{Ten: "x", BaseURL: srv.URL, Model: "m", KeyID: x}, "hi"); err == nil {
			t.Errorf("key_id=%q được chấp nhận", x)
		}
	}
	for _, a := range nhanDuoc {
		if strings.Contains(a, biMat) {
			t.Fatalf("BÍ MẬT NGOÀI KHO BỊ ĐỌC VÀ GỬI ĐI qua key_id thoát thư mục: %q", a)
		}
	}
}

// Lỗi của nhà cung cấp phải giữ NGUYÊN VĂN: họ trả kèm request id, và đó là thứ
// duy nhất dùng được khi phải hỏi lại họ. Rút gọn thành "lỗi 400" là vứt mất nó.
func TestGiuNguyenVanLoiCuaNhaCungCap(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	const than = `{"error":{"message":"Invalid request (request id: 20260818-abc123)"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(than))
	}))
	defer srv.Close()

	_, err := Goi(context.Background(), Route{Ten: "x", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	if err == nil {
		t.Fatal("HTTP 400 mà không báo lỗi")
	}
	if !strings.Contains(err.Error(), "20260818-abc123") {
		t.Fatalf("mất request id trong thông điệp lỗi: %v", err)
	}
}

// Key phải đi trong header Authorization và KHÔNG được rò ra chỗ nào khác.
func TestKeyDiDungChoVaKhongRoRaLoi(t *testing.T) {
	homeGia(t)
	const key = "sk-bi-mat-khong-duoc-lo"
	datKey(t, "k", key)

	var thayAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		thayAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("hong"))
	}))
	defer srv.Close()

	_, err := Goi(context.Background(), Route{Ten: "x", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	if thayAuth != "Bearer "+key {
		t.Fatalf("key không tới được server đúng cách: %q", thayAuth)
	}
	// Thông điệp lỗi đi ra terminal và ra dashboard — key TUYỆT ĐỐI không được ở đó.
	if err != nil && strings.Contains(err.Error(), key) {
		t.Fatalf("KEY RÒ RA THÔNG ĐIỆP LỖI: %v", err)
	}
}

// Usage phải được đọc và trả về: đường API tiêu TIỀN theo token, không đo được
// thì không biết đang tiêu gì.
func TestDocDuocUsage(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "m-1",
			"choices": []any{map[string]any{
				"message": map[string]string{"role": "assistant", "content": "OK"},
			}},
			"usage": map[string]int{"prompt_tokens": 12, "completion_tokens": 3, "total_tokens": 15},
		})
	}))
	defer srv.Close()

	kq, err := Goi(context.Background(), Route{Ten: "x", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if kq.NoiDung != "OK" {
		t.Fatalf("nội dung = %q", kq.NoiDung)
	}
	if kq.Usage.Vao != 12 || kq.Usage.Ra != 3 || kq.Usage.Tong != 15 {
		t.Fatalf("usage sai: %+v", kq.Usage)
	}
	if kq.Model != "m-1" {
		t.Errorf("model server trả về bị bỏ qua: %q", kq.Model)
	}
}

// Thiếu key thì phải nói CHỖ ĐẶT, không chỉ "không đọc được file".
func TestThieuKeyThiChiDuong(t *testing.T) {
	homeGia(t)
	_, err := Goi(context.Background(), Route{Ten: "x", BaseURL: "http://x", Model: "m", KeyID: "chua-co"}, "hi")
	if err == nil {
		t.Fatal("thiếu key mà không báo lỗi")
	}
	if !strings.Contains(err.Error(), "sagent api key") {
		t.Errorf("không chỉ đường đặt key: %v", err)
	}
}
