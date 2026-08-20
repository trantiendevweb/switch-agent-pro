// Chuyển route dự phòng — ba ca CƠ BẢN.
//
// Phần siết của hợp đồng nằm ở lichsu_test.go: chỉ chuyển ĐÚNG MỘT LẦN, không
// chuyển khi lỗi do người dùng (401/403/prompt rỗng), và KetQua phải mang lỗi
// gốc + request id của route chính về tận tay người gọi. Sửa AICall thì đọc CẢ
// HAI file, đừng sửa xong chỉ chạy file này.
package api

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

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
	"github.com/trantiendevweb/switch-agent-pro/internal/config"
)

func homeGiaAPI(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}
	t.Setenv("HOME", tmp)
	if err := os.MkdirAll(aiapi.KeysDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aiapi.KeysDir(), "k.key"), []byte("sk-test"), 0o600); err != nil {
		t.Fatal(err)
	}
	return tmp
}

// dungAPI dựng một API với cấu hình route cho sẵn, không đụng đĩa thật.
func dungAPI(t *testing.T, c config.Config) *API {
	t.Helper()
	a, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.cfg.AI = c.AI
	return a
}

func themRoute(c *config.Config, ten, url, model string) {
	c.AI.Routes = append(c.AI.Routes, struct {
		Ten     string `toml:"ten"`
		BaseURL string `toml:"base_url"`
		Model   string `toml:"model"`
		KeyID   string `toml:"key_id"`
	}{Ten: ten, BaseURL: url, Model: model, KeyID: "k"})
}

// Route chính hỏng thì chuyển sang route dự phòng — đúng cảnh của người dùng:
// tài khoản chính hết hạn mức, cần chạy tiếp bằng cái khác.
func TestFallbackChuyenSangRouteDuPhong(t *testing.T) {
	homeGiaAPI(t)
	hong := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limit (request id: RID-CHINH)"}`))
	}))
	defer hong.Close()
	tot := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "m-du-phong",
			"choices": []any{map[string]any{"message": map[string]string{"content": "OK"}}},
			"usage":   map[string]int{"prompt_tokens": 1, "completion_tokens": 2, "total_tokens": 3},
		})
	}))
	defer tot.Close()

	var c config.Config
	themRoute(&c, "chinh", hong.URL, "m1")
	themRoute(&c, "phu", tot.URL, "m2")
	c.AI.DefaultRoute = "chinh"
	c.AI.FallbackRoutes = []string{"phu"}

	a := dungAPI(t, c)
	kq, err := a.AICall(context.Background(), "", "hi")
	if err != nil {
		t.Fatalf("route chính hỏng mà không chuyển sang dự phòng: %v", err)
	}
	if kq.NoiDung != "OK" || kq.Model != "m-du-phong" {
		t.Fatalf("không phải kết quả của route dự phòng: %+v", kq)
	}
	if kq.Usage.Tong != 3 {
		t.Errorf("mất usage khi fallback: %+v", kq.Usage)
	}
}

// Điều kiện KHÓ NHẤT của DoD Pha 4: fallback không được làm mất
// correlation ID / error gốc. Mọi route hỏng thì phải thấy ĐỦ lý do từng cái,
// kể cả request id của nhà cung cấp — đó là thứ duy nhất dùng được khi đi hỏi họ.
func TestMoiRouteHongThiGiuDuLoiGoc(t *testing.T) {
	homeGiaAPI(t)
	mk := func(rid string, ma int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(ma)
			_, _ = w.Write([]byte(`{"error":"hỏng (request id: ` + rid + `)"}`))
		}))
	}
	s1, s2 := mk("RID-MOT", 429), mk("RID-HAI", 500)
	defer s1.Close()
	defer s2.Close()

	var c config.Config
	themRoute(&c, "mot", s1.URL, "m1")
	themRoute(&c, "hai", s2.URL, "m2")
	c.AI.DefaultRoute = "mot"
	c.AI.FallbackRoutes = []string{"hai"}

	a := dungAPI(t, c)
	_, err := a.AICall(context.Background(), "", "hi")
	if err == nil {
		t.Fatal("mọi route hỏng mà không báo lỗi")
	}
	for _, phai := range []string{"RID-MOT", "RID-HAI"} {
		if !strings.Contains(err.Error(), phai) {
			t.Errorf("MẤT correlation ID %s — không còn gì để đi hỏi nhà cung cấp:\n%v", phai, err)
		}
	}
}

// Gọi đích danh một route thì KHÔNG được lặng lẽ chuyển sang route khác: người
// dùng chỉ định nhà cung cấp là có lý do (giá, dữ liệu, hợp đồng).
func TestGoiDichDanhThiKhongTuChuyenRoute(t *testing.T) {
	homeGiaAPI(t)
	hong := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer hong.Close()
	var goiPhu bool
	tot := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		goiPhu = true
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]string{"content": "OK"}}},
		})
	}))
	defer tot.Close()

	var c config.Config
	themRoute(&c, "chinh", hong.URL, "m1")
	themRoute(&c, "phu", tot.URL, "m2")
	c.AI.DefaultRoute = "chinh"
	c.AI.FallbackRoutes = []string{"phu"}

	a := dungAPI(t, c)
	if _, err := a.AICall(context.Background(), "chinh", "hi"); err == nil {
		t.Fatal("route chỉ định hỏng mà báo thành công")
	}
	if goiPhu {
		t.Fatal("gọi đích danh 'chinh' mà tự chuyển sang 'phu' — người dùng chọn nhà cung cấp là có lý do")
	}
	// Và sổ chỉ được có ĐÚNG một dòng: một lời gọi thật thì một dòng.
	ds, err := a.AIHistory(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 || ds[0].Route != "chinh" || ds[0].OK {
		t.Errorf("sổ lời gọi API không khớp với chuyện đã xảy ra: %+v", ds)
	}
}
