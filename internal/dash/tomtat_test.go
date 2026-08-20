package dash

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// luotThat chạy một flow một bước shell (không đụng agent, không đốt hạn mức)
// rồi trả về số lượt chạy — đủ để endpoint có thứ thật mà tóm tắt.
func luotThat(t *testing.T, s *Server) int64 {
	t.Helper()
	dir := t.TempDir()
	f := flow.Flow{Name: "thu", Steps: []flow.Step{
		{ID: "kiem", Type: flow.TypeShell, Run: []string{"go", "version"}},
	}}
	if _, err := s.api.FlowSave(dir, f); err != nil {
		t.Fatal(err)
	}
	res, err := s.api.FlowRunCuChay(context.Background(), dir, "thu", nil, api.Addr{}, true)
	if err != nil {
		t.Fatal(err)
	}
	return res.RunID
}

// Mặt web phải lấy được BẢN TÓM TẮT, không chỉ nguyên liệu.
//
// /api/flow/detail đổ ra mọi thứ agent nói; đọc rồi tự kết luận là việc người
// dùng đang làm bằng tay, và lượt #21/#29/#31/#34 cho thấy kết luận rút từ lời
// agent đã sai bốn lần. Endpoint này trả về câu trả lời, kèm bằng chứng git.
func TestWebLayDuocBanTomTatLuotChay(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	id := luotThat(t, s)

	m := doc(t, s, ck, "/api/flow/tom-tat?id="+strconv.FormatInt(id, 10))
	for _, k := range []string{"daLam", "chuaLam", "hong", "treo", "nhanh", "khai", "mauThuan", "vanBan"} {
		if _, co := m[k]; !co {
			t.Errorf("bản tóm tắt thiếu khoá %q — mặt web không dựng được mục đó.\nCó: %v", k, khoa(m))
		}
	}
	van, _ := m["vanBan"].(string)
	for _, phai := range []string{"AI LÀM GÌ", "AI CHƯA LÀM", "BƯỚC NÀO HỎNG", "VIỆC CÒN TREO",
		"BẰNG CHỨNG GIT", "ĐỐI CHIẾU LỜI AGENT VỚI GIT"} {
		if !strings.Contains(van, phai) {
			t.Errorf("bản in thiếu mục %q:\n%s", phai, van)
		}
	}
	daLam, _ := m["daLam"].([]any)
	if len(daLam) != 1 {
		t.Errorf("lượt chạy một bước shell mà mục AI LÀM GÌ có %d bước", len(daLam))
	}
}

// id sai thì báo lỗi rõ, không trả 200 kèm một bản tóm tắt rỗng — bản tóm tắt
// rỗng trông y hệt "lượt chạy chẳng ai làm gì", đúng kiểu hỏng im lặng.
func TestTomTatIdSaiThiBaoLoi(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	for _, d := range []string{"/api/flow/tom-tat", "/api/flow/tom-tat?id=abc", "/api/flow/tom-tat?id=999999"} {
		r := httptest.NewRequest("GET", d, nil)
		r.Host = "127.0.0.1:4600"
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			t.Errorf("%s trả 200 trong khi không có lượt chạy nào để tóm tắt: %s", d, w.Body.String())
		}
	}
}

// Endpoint phải nằm sau lớp đăng nhập như mọi endpoint khác: bản tóm tắt mang
// theo đường dẫn thư mục làm việc và tên nhánh nội bộ.
func TestTomTatPhaiDangNhapMoiXem(t *testing.T) {
	s := newTestServer(t)
	r := httptest.NewRequest("GET", "/api/flow/tom-tat?id=1", nil)
	r.Host = "127.0.0.1:4600"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code == http.StatusOK {
		t.Fatal("chưa đăng nhập vẫn đọc được bản tóm tắt")
	}
}

// Mặt web phải có ĐƯỜNG BẤM tới bản tóm tắt, không chỉ có endpoint. Endpoint
// không ai bấm được thì tính năng vẫn chỉ tồn tại ở terminal — đúng thứ luật
// ngang quyền cấm.
func TestManHoiThoaiCoNutTomTat(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("web", "hoi-thoai.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "/api/flow/tom-tat") {
		t.Error("hoi-thoai.html không gọi /api/flow/tom-tat — bản tóm tắt chỉ có ở terminal")
	}
	if !strings.Contains(s, "nut-tom-tat") {
		t.Error("hoi-thoai.html không có nút mở bản tóm tắt")
	}
}

// Bản tóm tắt trả về đúng JSON hợp lệ kể cả khi lượt chạy không có nhánh nào để
// đối chiếu — mảng rỗng, KHÔNG phải null (null làm .map() ở trình duyệt nổ).
func TestTomTatTraMangRongChuKhongPhaiNull(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	id := luotThat(t, s)

	r := httptest.NewRequest("GET", "/api/flow/tom-tat?id="+strconv.FormatInt(id, 10), nil)
	r.Host = "127.0.0.1:4600"
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"daLam", "chuaLam", "hong", "treo", "nhanh", "khai", "mauThuan", "chuaDoiChieu"} {
		if string(m[k]) == "null" {
			t.Errorf("khoá %q trả về null — trình duyệt gọi .map() lên nó là cả trang chết", k)
		}
	}
}
