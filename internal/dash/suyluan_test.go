package dash

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
	"github.com/trantiendevweb/switch-agent-pro/internal/api"
)

// PHẦN SUY LUẬN PHẢI ĐI HẾT ĐƯỜNG TỚI TRÌNH DUYỆT.
//
// Đây là bài kiểm anh em với `TestSuyLuanDiHetDuongToiKetQua` bên
// internal/aiapi, và nó tồn tại vì đúng lý do đã cắn dự án này năm lần trong
// ngày 22/08: một giá trị "có" ở mọi tầng trừ tầng cuối cùng. Chiều 22/08,
// `KetQua.SuyLuan` đã có, đã được `Goi` gán đúng, đã có bài kiểm xanh — mà
// KHÔNG mặt nào đọc nó. Người bấm nút trên dashboard trả tiền cho phần nghĩ
// (nó nằm trong `completion_tokens`) và không có cách nào nhìn thấy.
//
// Nên bài này đi HẾT: nhà cung cấp giả → api.AICall → HTTP /api/ai → thân JSON
// mà trình duyệt thật sự nhận. Bỏ dòng `"suy_luan"` trong server.go là đỏ.

// serverCoRouteAI dựng server thật với MỘT route API trỏ vào `baseURL` (nhà
// cung cấp giả), key thật nằm trong kho tạm.
//
// Không dùng `newTestServer` được: nó gọi `api.New` ngay, mà cấu hình route
// phải có mặt TRƯỚC lúc đó — `config.Load` đọc file một lần rồi thôi.
func serverCoRouteAI(t *testing.T, baseURL string) *Server {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cauHinh := filepath.Join(home, ".sagent")
	if err := os.MkdirAll(cauHinh, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := fmt.Sprintf("[ai]\ndefault_route = \"thu\"\n\n  [[ai.route]]\n  ten = \"thu\"\n"+
		"  base_url = %q\n  model = \"m\"\n  key_id = \"thu\"\n", baseURL)
	if err := os.WriteFile(filepath.Join(cauHinh, "project.toml"), []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(aiapi.KeysDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aiapi.KeysDir(), "thu.key"), []byte("sk-thu"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := SetPassword("Admin", matKhauTest); err != nil {
		t.Fatal(err)
	}
	a, err := api.New(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	if len(a.AIRoutes()) != 1 {
		t.Fatalf("cấu hình route không nạp được: %+v", a.AIRoutes())
	}
	return New(a)
}

// hoiAI bấm đúng nút "Gửi" của mặt web: POST /api/ai qua cửa đăng nhập.
func hoiAI(t *testing.T, s *Server, than string) (int, string) {
	t.Helper()
	r := httptest.NewRequest("POST", "/api/ai", strings.NewReader(than))
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://127.0.0.1:4600")
	r.AddCookie(dangNhap(t, s, "127.0.0.1:4600"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

// nccCoPhanNghi là nhà cung cấp giả TRẢ phần nghĩ, ở cả hai đường.
func nccCoPhanNghi(nghi, traLoi string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var than struct {
			Stream bool `json:"stream"`
		}
		_ = json.NewDecoder(r.Body).Decode(&than)
		if than.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			fl, _ := w.(http.Flusher)
			for _, m := range []string{
				fmt.Sprintf(`{"model":"m","choices":[{"delta":{"reasoning_content":%q}}]}`, nghi),
				fmt.Sprintf(`{"model":"m","choices":[{"delta":{"content":%q}}]}`, traLoi),
				`{"model":"m","choices":[],"usage":{"prompt_tokens":9,"completion_tokens":3,"total_tokens":12}}`,
			} {
				fmt.Fprintf(w, "data: %s\n\n", m)
				if fl != nil {
					fl.Flush()
				}
			}
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"model":"m","choices":[{"message":{"role":"assistant","content":%q,`+
			`"reasoning_content":%q}}],"usage":{"prompt_tokens":9,"completion_tokens":3,`+
			`"total_tokens":12}}`, traLoi, nghi)
	}))
}

// Đường JSON thường: phần nghĩ phải nằm trong thân trả về.
func TestApiAITraVePhanSuyLuan(t *testing.T) {
	const nghi = "Bước 1: cộng hai số. Bước 2: trả lời."
	ncc := nccCoPhanNghi(nghi, "4")
	defer ncc.Close()
	s := serverCoRouteAI(t, ncc.URL)

	ma, than := hoiAI(t, s, `{"route":"thu","prompt":"2+2"}`)
	if ma != 200 {
		t.Fatalf("muốn 200, được %d: %s", ma, than)
	}
	var d struct {
		NoiDung string             `json:"noi_dung"`
		SuyLuan aiapi.KhoiSuyLuan  `json:"suy_luan"`
		Usage   map[string]float64 `json:"usage"`
	}
	if err := json.Unmarshal([]byte(than), &d); err != nil {
		t.Fatalf("thân không đọc được: %v — %s", err, than)
	}
	if !d.SuyLuan.Co || d.SuyLuan.NoiDung != nghi {
		t.Fatalf("phần suy luận KHÔNG tới được trình duyệt: %+v\nthân: %s", d.SuyLuan, than)
	}
	if d.SuyLuan.SoKyTu != len([]rune(nghi)) {
		t.Errorf("SoKyTu = %d, chờ %d", d.SuyLuan.SoKyTu, len([]rune(nghi)))
	}
	// Phần nghĩ KHÔNG được lẫn vào câu trả lời: mặt web nhét `noi_dung` vào
	// chỗ khác được, và trộn hai thứ là đưa bản nháp đi làm dữ liệu.
	if d.NoiDung != "4" {
		t.Errorf("noi_dung = %q — phần nghĩ lẫn vào câu trả lời", d.NoiDung)
	}
}

// Đường STREAM — và đây mới là đường mặt web THẬT SỰ đi: `aiform` luôn gọi
// `hoiAIStream`. Nhánh JSON ở trên xanh mà nhánh này thiếu thì tính năng coi
// như không có, và không bài nào bắt được nếu chỉ đo nhánh kia.
func TestMauTongKetStreamMangPhanSuyLuan(t *testing.T) {
	const nghi = "Ao phủ gấp đôi mỗi ngày nên lùi một ngày là nửa ao."
	ncc := nccCoPhanNghi(nghi, "Ngày 29.")
	defer ncc.Close()
	s := serverCoRouteAI(t, ncc.URL)

	ma, than := hoiAI(t, s, `{"route":"thu","prompt":"ngày nào nửa ao","stream":true}`)
	if ma != 200 {
		t.Fatalf("muốn 200, được %d: %s", ma, than)
	}
	var cuoi map[string]any
	for _, m := range strings.Split(than, "\n\n") {
		d := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(m), "data:"))
		if d == "" {
			continue
		}
		var v map[string]any
		if json.Unmarshal([]byte(d), &v) != nil {
			continue
		}
		if v["xong"] == true {
			cuoi = v
		}
	}
	if cuoi == nil {
		t.Fatalf("luồng không có mẩu tổng kết: %s", than)
	}
	sl, _ := cuoi["suy_luan"].(map[string]any)
	if sl == nil {
		t.Fatalf("mẩu tổng kết KHÔNG mang `suy_luan` — mặt web luôn hỏi bằng stream, "+
			"nên thiếu ở đây là tính năng không tồn tại trên dashboard: %v", cuoi)
	}
	if sl["co"] != true || sl["noi_dung"] != nghi {
		t.Fatalf("phần suy luận sai ở mẩu tổng kết: %v", sl)
	}
}

// KHÔNG có phần nghĩ thì thân trả về phải NÓI RÕ VÌ SAO, không được để trống.
//
// Ô trống không giải thích chính là thứ bị đọc thành "model không nghĩ" — mà
// chuỗi rỗng KHÔNG nói được điều đó. Đo 22/08: grok-4.5 không trả phần nghĩ dù
// lượt đó tiêu 951 token và mất 20,3 giây.
func TestKhongCoPhanSuyLuanThiNoiRoViSao(t *testing.T) {
	ncc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"m","choices":[{"message":{"content":"4"}}],`+
			`"usage":{"total_tokens":10}}`)
	}))
	defer ncc.Close()
	s := serverCoRouteAI(t, ncc.URL)

	ma, than := hoiAI(t, s, `{"route":"thu","prompt":"2+2"}`)
	if ma != 200 {
		t.Fatalf("muốn 200, được %d: %s", ma, than)
	}
	var d struct {
		SuyLuan aiapi.KhoiSuyLuan `json:"suy_luan"`
	}
	if err := json.Unmarshal([]byte(than), &d); err != nil {
		t.Fatal(err)
	}
	if d.SuyLuan.Co {
		t.Fatal("không có phần nghĩ mà server nói có")
	}
	if strings.TrimSpace(d.SuyLuan.ViSaoRong) == "" {
		t.Fatal("server trả ô trống KHÔNG kèm lời giải thích — trình duyệt chỉ còn cách " +
			"hiện một khoảng trắng, và người đọc sẽ tự kết luận \"model không nghĩ\"")
	}
	if !strings.Contains(d.SuyLuan.DanToi, "nang-luc-api") {
		t.Errorf("không dẫn tới bảng năng lực — chỗ DUY NHẤT trả lời được câu đó: %q",
			d.SuyLuan.DanToi)
	}
	for _, cam := range []string{"model không nghĩ", "model không suy luận"} {
		if strings.Contains(strings.ToLower(d.SuyLuan.ViSaoRong), cam) {
			t.Errorf("server khẳng định %q — một chuyện KHÔNG đo được từ chuỗi rỗng: %s",
				cam, d.SuyLuan.ViSaoRong)
		}
	}
}

// TRANG phải thật sự VẼ thứ server gửi. Server trả đủ mà trang không đọc thì
// vẫn là đúng cái lỗi "có ở mọi tầng trừ tầng cuối" — chỉ dời xuống một tầng.
func TestTrangVeDuocPhanSuyLuan(t *testing.T) {
	web := boComment(doc2DThuong(t))
	for _, can := range []string{"veSuyLuan", "suy_luan", "vi_sao_rong", "so_ky_tu"} {
		if !strings.Contains(web, can) {
			t.Errorf("index.html không đọc %q — phần suy luận về tới trình duyệt rồi "+
				"bị vứt ở dòng cuối cùng", can)
		}
	}
	// Cả hai nhánh phải có chỗ hiện: khối phần nghĩ VÀ ô giải thích khi rỗng.
	html := boCommentHTML(doc2DThuong(t))
	for _, id := range []string{`id="ai-nghi"`, `id="ai-nghi-trong"`} {
		if !strings.Contains(html, id) {
			t.Errorf("index.html thiếu %s", id)
		}
	}
	// Lượt mới phải DỌN phần nghĩ của lượt trước: để nó nằm lại cạnh câu trả
	// lời mới là nói dối về một thứ đã tính tiền.
	if !strings.Contains(web, "veSuyLuan(null)") {
		t.Error("trang không dọn phần nghĩ của lượt trước khi hỏi lượt mới")
	}
}
