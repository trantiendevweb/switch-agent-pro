package dash

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
	"github.com/trantiendevweb/switch-agent-pro/internal/api"
)

// dtoNangLucAPI là hình dạng mặt web TRÔNG ĐỢI. Đọc qua struct chứ không qua
// map[string]any: đổi tên một trường trong handler thì test đỏ ngay, thay vì
// trang lặng lẽ hiện "undefined".
type dtoNangLucAPI struct {
	Route []struct {
		Ten     string `json:"ten"`
		BaseURL string `json:"base_url"`
		Model   string `json:"model"`
		Muc     []struct {
			Khoa           string `json:"khoa"`
			Mo             string `json:"mo"`
			TrangThai      string `json:"trang_thai"`
			BangChung      string `json:"bang_chung"`
			Cho            string `json:"cho"`
			NhanCho        string `json:"nhan_cho"`
			Khach          string `json:"khach"`
			BangChungKhach string `json:"bang_chung_khach"`
			NCC            string `json:"ncc"`
			BangChungNCC   string `json:"bang_chung_ncc"`
		} `json:"muc"`
		Lech []string `json:"lech"`
	} `json:"route"`
	SoChuaDo int `json:"so_chua_do"`
}

// serverCoRoute là newTestServer NHƯNG có sẵn hai route API trong cấu hình.
//
// Phải có bản riêng vì `newTestServer` dựng một HOME rỗng, mà bảng năng lực
// đọc từ `.sagent/project.toml`: không route nào thì mọi bài dưới đây chỉ kiểm
// được đúng câu "chưa cấu hình route nào".
//
// Hai route ghi ở đây là ĐÚNG base_url và model của .sagent/project.toml thật,
// để chúng khớp sổ số đo trong internal/aiapi. Cố ý KHÔNG tạo file key: bảng
// phải trả lời được cả khi chưa có key — nó chỉ `Stat`, không đọc nội dung —
// và bài TestNangLucAPIKhongMangKeyRaNgoai bên dưới tự gieo key của riêng nó.
func serverCoRoute(t *testing.T) *Server {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := SetPassword("Admin", matKhauTest); err != nil {
		t.Fatal(err)
	}
	thuMuc := filepath.Join(home, ".sagent")
	if err := os.MkdirAll(thuMuc, 0o700); err != nil {
		t.Fatal(err)
	}
	toml := `version = 1
name = "thu"

[ai]
default_route = "deepseek"

  [[ai.route]]
  ten      = "deepseek"
  base_url = "https://modelapi.vn/v1"
  model    = "deepseek-v4-flash"
  key_id   = "deepseek"

  [[ai.route]]
  ten      = "grok"
  base_url = "https://modelapi.vn/v1"
  model    = "grok-4.5"
  key_id   = "grok"
`
	if err := os.WriteFile(filepath.Join(thuMuc, "project.toml"), []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := api.New(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return New(a)
}

func layNangLucAPI(t *testing.T, duong string) dtoNangLucAPI {
	t.Helper()
	s := serverCoRoute(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := req("GET", duong)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("GET %s trả %d: %s", duong, w.Code, w.Body.String())
	}
	var d dtoNangLucAPI
	if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatalf("thân trả về không đọc được: %v\n%s", err, w.Body.String())
	}
	return d
}

// Mặt web phải hỏi được câu "route API này làm được gì".
//
// Không hỏi được thì tính năng chỉ có ở terminal — đúng chiều ngược của luật
// ngang quyền, và đúng ba lần dự án đã vấp (route.kiem, nút Duyệt/Từ chối,
// plugin.list).
func TestDuongNangLucAPITraDuMoiMuc(t *testing.T) {
	d := layNangLucAPI(t, "/api/nang-luc-api")
	if len(d.Route) == 0 {
		t.Fatal("đường /api/nang-luc-api không trả route nào")
	}
	for _, b := range d.Route {
		if len(b.Lech) != 0 {
			t.Errorf("%s: bảng chọi với chính nó: %v", b.Ten, b.Lech)
		}
		if b.Model == "" || b.BaseURL == "" {
			t.Errorf("%s: DTO thiếu model hoặc base_url", b.Ten)
		}
		co := map[string]bool{}
		for _, m := range b.Muc {
			co[m.Khoa] = true
			// Đây là toàn bộ nội dung trang vẽ ra. Thiếu một cái là một ô trống
			// không giải thích được.
			for ten, v := range map[string]string{
				"mo": m.Mo, "trang_thai": m.TrangThai, "bang_chung": m.BangChung,
				"cho": m.Cho, "nhan_cho": m.NhanCho,
				"khach": m.Khach, "bang_chung_khach": m.BangChungKhach,
				"ncc": m.NCC, "bang_chung_ncc": m.BangChungNCC,
			} {
				if v == "" {
					t.Errorf("%s/%s: DTO thiếu trường %q", b.Ten, m.Khoa, ten)
				}
			}
		}
		for _, m := range aiapi.MoiNangLucAPI {
			if !co[m.Khoa] {
				t.Errorf("%s: DTO thiếu năng lực %q", b.Ten, m.Khoa)
			}
		}
	}
}

// BA trạng thái phải sống sót qua lớp DTO — cả ba cột, không chỉ cột kết luận.
//
// Đây là chỗ dễ mất nhất, y như đã ghi ở TestDTOKhongDepTrangThaiNaoThanhTrang
// ThaiKhac bên bảng provider: ai đó thấy "khong-lam-duoc" và "chua-do" đều
// không kèm cờ nào rồi gộp thành một `false` cho gọn.
//
// So TỪNG MỤC với lớp API chứ không đòi "phải thấy đủ ba trạng thái trong dữ
// liệu sản phẩm": cách sau chỉ xanh chừng nào dự án CÒN NỢ, tức là nó phạt đúng
// việc mà cuốn sổ nợ tồn tại để thúc đẩy.
func TestDTONangLucAPIKhongDepTrangThai(t *testing.T) {
	d := layNangLucAPI(t, "/api/nang-luc-api")

	s := serverCoRoute(t)
	goc, err := s.api.NangLucAPI("")
	if err != nil {
		t.Fatal(err)
	}
	type bo struct{ ketLuan, khach, ncc string }
	mongDoi := map[string]bo{}
	for _, b := range goc {
		for _, m := range b.Muc {
			mongDoi[b.Ten+"/"+m.Khoa] = bo{string(m.TrangThai), string(m.Khach), string(m.NCC)}
		}
	}

	var soChuaDo, soMuc int
	for _, b := range d.Route {
		for _, m := range b.Muc {
			soMuc++
			khoa := b.Ten + "/" + m.Khoa
			muon, co := mongDoi[khoa]
			if !co {
				t.Errorf("DTO có mục %q mà lớp API không có", khoa)
				continue
			}
			if m.TrangThai != muon.ketLuan || m.Khach != muon.khach || m.NCC != muon.ncc {
				t.Errorf("%s: lớp API nói (%s,%s,%s), DTO nói (%s,%s,%s) — lớp web đang "+
					"đổi trạng thái", khoa, muon.ketLuan, muon.khach, muon.ncc,
					m.TrangThai, m.Khach, m.NCC)
			}
			if m.TrangThai == string(aiapi.ChuaDo) {
				soChuaDo++
			}
		}
	}
	if soMuc != len(mongDoi) {
		t.Errorf("DTO mang %d mục, lớp API có %d — lớp web đang nuốt hoặc nhân bản mục",
			soMuc, len(mongDoi))
	}
	if d.SoChuaDo != soChuaDo {
		t.Errorf("so_chua_do = %d nhưng đếm được %d — mỗi mặt cộng một kiểu thì con số "+
			"trên màn hình không tin được", d.SoChuaDo, soChuaDo)
	}
}

// Cột "vướng ở đâu" phải đi kèm NHÃN đọc được, không phải mã thô.
//
// Nhãn dựng ở lớp Go (aiapi.NhanCho) chứ không chép sang JS: mã `nha-cung-cap`
// mà mặt web tự đặt tên là "Nhà cung cấp" thì hai mặt nói hai kiểu, và sửa một
// bên sẽ quên bên kia. Cùng lý do `mo` đi kèm từ MoiNangLucAPI.
func TestDTOMangNhanVuongODau(t *testing.T) {
	d := layNangLucAPI(t, "/api/nang-luc-api")
	for _, b := range d.Route {
		for _, m := range b.Muc {
			if m.NhanCho == m.Cho {
				t.Errorf("%s/%s: cho=%q không có nhãn — trang sẽ hiện mã thô",
					b.Ten, m.Khoa, m.Cho)
			}
			if m.NhanCho != aiapi.NhanCho(m.Cho) {
				t.Errorf("%s/%s: nhãn của lớp web (%q) khác nhãn của lớp Go (%q)",
					b.Ten, m.Khoa, m.NhanCho, aiapi.NhanCho(m.Cho))
			}
		}
	}
}

// Lọc theo route phải chạy, và tên lạ phải BÁO LỖI chứ không trả rỗng.
func TestDuongNangLucAPILocTheoRoute(t *testing.T) {
	d := layNangLucAPI(t, "/api/nang-luc-api?route=deepseek")
	if len(d.Route) != 1 || d.Route[0].Ten != "deepseek" {
		t.Fatalf("lọc theo route sai: %+v", d.Route)
	}

	s := serverCoRoute(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := req("GET", "/api/nang-luc-api?route=khong-co-route-nay")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Fatalf("tên route lạ phải báo lỗi, được 200: %s", w.Body.String())
	}
}

// Phép ĐO THẬT phải ĐÒI POST.
//
// Không phải để cho đúng lễ nghi REST: một GET tiêu tiền là một GET mà trình
// duyệt, bộ nạp trước, hay một lần bấm F5 nhầm đều có quyền gọi lại. Cùng luật
// đã áp cho `/api/stop` khi giết tiến trình.
//
// Bài này KHÔNG gọi POST: làm vậy là tiêu token thật mỗi lần ai đó chạy
// `go test ./...`, và một bộ test tiêu tiền là bộ test người ta sẽ thôi chạy.
func TestDuongDoNangLucAPIDoiPOST(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := req("GET", "/api/nang-luc-api/do")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Fatal("GET vào đường đo thật được chấp nhận — một lần F5 nhầm là một lần tiêu token")
	}
	if !strings.Contains(w.Body.String(), "POST") {
		t.Errorf("lỗi trả về không nói phải dùng POST: %s", w.Body.String())
	}
}

// Chưa đăng nhập thì cả hai đường đều đóng như mọi đường /api/* khác.
//
// Với đường `/do` thì đây KHÔNG chỉ là chuyện riêng tư: một đường không cần
// đăng nhập mà gọi được ra Internet bằng key thật là một cái máy tiêu tiền
// miễn phí cho người lạ.
func TestNangLucAPIDoiDangNhap(t *testing.T) {
	s := newTestServer(t)
	for _, d := range []string{"/api/nang-luc-api", "/api/nang-luc-api/do"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req("GET", d))
		if w.Code != 401 {
			t.Errorf("%s: chưa đăng nhập phải 401, được %d", d, w.Code)
		}
	}
}

// Bảng này KHÔNG được mang nội dung key ra ngoài.
//
// Nó chỉ mang khoá, trạng thái, bằng chứng và tên `key_id` — toàn chữ viết sẵn
// trong mã nguồn cộng số đo đã duyệt. Gieo một key mang dấu hiệu không thể
// trùng rồi đòi dấu hiệu đó vắng mặt trong thân trả về; kiểm bằng GIÁ TRỊ THẬT
// chứ không bằng tên trường, cùng lý do đã ghi ở bảng năng lực provider.
func TestNangLucAPIKhongMangKeyRaNgoai(t *testing.T) {
	const dauHieu = "KEY-GIA-KHONG-DUOC-LO-RA-8d02"
	s := serverCoRoute(t)
	// HOME đã bị trỏ vào thư mục tạm của bài này, nên `KeysDir()` cũng nằm
	// trong đó — key gieo ở đây KHÔNG đụng tới kho key thật của máy.
	kho := aiapi.KeysDir()
	if err := os.MkdirAll(kho, 0o700); err != nil {
		t.Fatalf("không tạo được kho key thử: %v", err)
	}
	if err := os.WriteFile(filepath.Join(kho, "deepseek.key"), []byte(dauHieu), 0o600); err != nil {
		t.Fatalf("không ghi được key thử: %v", err)
	}
	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := req("GET", "/api/nang-luc-api")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("mã %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), dauHieu) {
		t.Fatal("/api/nang-luc-api mang nội dung file key ra ngoài")
	}
}

// Mặt 2D phải THẬT SỰ gọi hai đường này và vẽ đủ ba trạng thái.
//
// Cùng ý với TestMat2DGoiDuongNangLuc: một nút mất chỉ lộ ra khi người dùng đi
// tìm nó, mà lúc đó không ai nhớ lần sửa nào làm mất. Ba lần dự án vấp đúng chỗ
// này — "endpoint có, CLI có, test ngang quyền xanh, mà không ai bấm được".
func TestMat2DGoiDuongNangLucAPI(t *testing.T) {
	b, err := os.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	ma := string(b)
	for _, can := range []string{"/api/nang-luc-api", "/api/nang-luc-api/do"} {
		if !strings.Contains(ma, can) {
			t.Errorf("index.html không gọi %s — tính năng chỉ còn ở terminal", can)
		}
	}
	if !strings.Contains(ma, "napNangLucAPI()") {
		t.Error("index.html khai hàm napNangLucAPI mà không gọi lúc tải — khối sẽ đứng " +
			"nguyên chữ \"đang đọc…\"")
	}
	// Hai nút phải CÓ THẬT và được nối tay cầm. Một hàm không ai gọi thì bảng
	// vẫn trắng, và không test nào khác bắt được.
	for _, nut := range []string{"$('nla-nap').onclick", "$('nla-do').onclick"} {
		if !strings.Contains(ma, nut) {
			t.Errorf("index.html không nối tay cầm cho %s — nút bấm không làm gì", nut)
		}
	}
	for _, id := range []string{`id="nla"`, `id="nla-nap"`, `id="nla-do"`, `id="nla-dem"`} {
		if !strings.Contains(ma, id) {
			t.Errorf("index.html thiếu %s — JS sẽ chạm vào null và cả script chết", id)
		}
	}
	// Phép đo tiêu tiền phải đi bằng POST từ mặt web luôn, không chỉ ở server.
	if !strings.Contains(ma, "method:'POST'") {
		t.Error("mặt web gọi đường đo thật mà không dùng POST")
	}
	// Nút tiêu tiền phải NÓI TRƯỚC. Hai nút cạnh nhau, một miễn phí một tốn
	// tiền, thì sớm muộn có người bấm nhầm.
	if !strings.Contains(ma, "confirm(") {
		t.Error("nút đo thật không hỏi xác nhận — nó tiêu token thật")
	}
	// Ba trạng thái phải phân biệt được bằng mắt. Khối này dùng LẠI ba lớp của
	// bảng năng lực provider (.nl .m.duoc/.khong/.chua) — bài kiểm của bảng đó
	// đã canh ba lớp ấy, nên ở đây chỉ cần canh phần RIÊNG của khối này.
	if !strings.Contains(ma, ".nla .m .cho") {
		t.Error("index.html không vẽ cột \"vướng ở đâu\" — bảng mất đúng câu nói chỗ cần sửa")
	}
	if !strings.Contains(ma, "bang_chung_khach") || !strings.Contains(ma, "bang_chung_ncc") {
		t.Error("mặt web chỉ vẽ một vế bằng chứng — mất câu \"họ làm được, ta chưa gửi\", " +
			"và người đọc sẽ đi đổi nhà cung cấp cho một chỗ hỏng nằm trong repo này")
	}
	if !strings.Contains(ma, "'lech'") {
		t.Error("index.html không vẽ phần `lech` — bảng năng lực sai sẽ hiện ra như bảng đúng")
	}
}
