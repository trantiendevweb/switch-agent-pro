package dash

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// dtoNangLuc là hình dạng mặt web TRÔNG ĐỢI. Test đọc qua struct này chứ không
// qua map[string]any: đổi tên một trường trong handler thì test đỏ ngay, thay vì
// trang lặng lẽ hiện "undefined".
type dtoNangLuc struct {
	Provider []struct {
		Provider string `json:"provider"`
		Muc      []struct {
			Khoa      string `json:"khoa"`
			Mo        string `json:"mo"`
			TrangThai string `json:"trang_thai"`
			BangChung string `json:"bang_chung"`
		} `json:"muc"`
		Lech []string `json:"lech"`
	} `json:"provider"`
	SoChuaDo int `json:"so_chua_do"`
}

func layNangLuc(t *testing.T, duong string) dtoNangLuc {
	t.Helper()
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := req("GET", duong)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("GET %s trả %d: %s", duong, w.Code, w.Body.String())
	}
	var d dtoNangLuc
	if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatalf("thân trả về không đọc được: %v\n%s", err, w.Body.String())
	}
	return d
}

// Mặt web phải hỏi được câu "provider nào làm được gì" — nếu không thì tính năng
// này chỉ có ở terminal, đúng chiều ngược của luật ngang quyền.
func TestDuongNangLucTraDuMoiProvider(t *testing.T) {
	d := layNangLuc(t, "/api/nang-luc")
	if len(d.Provider) != len(provider.Names()) {
		t.Fatalf("có %d provider đăng ký, đường /api/nang-luc chỉ trả %d",
			len(provider.Names()), len(d.Provider))
	}
	for _, p := range d.Provider {
		if len(p.Lech) != 0 {
			t.Errorf("%s: bảng năng lực chọi với hành vi thật: %v", p.Provider, p.Lech)
		}
		co := map[string]bool{}
		for _, m := range p.Muc {
			co[m.Khoa] = true
			// Ba trường này là toàn bộ nội dung mà trang vẽ ra. Thiếu một cái là
			// một ô trống không giải thích được.
			if m.Mo == "" || m.BangChung == "" || m.TrangThai == "" {
				t.Errorf("%s/%s: DTO thiếu trường (mo=%q trang_thai=%q bang_chung=%q)",
					p.Provider, m.Khoa, m.Mo, m.TrangThai, m.BangChung)
			}
		}
		for _, m := range provider.MoiNangLuc {
			if !co[m.Khoa] {
				t.Errorf("%s: DTO thiếu năng lực %q", p.Provider, m.Khoa)
			}
		}
	}
}

// BA trạng thái phải sống sót qua lớp DTO.
//
// Đây là chỗ dễ mất nhất: JSON hoá thì ai đó thấy "khong-lam-duoc" và "chua-do"
// đều không kèm cờ nào rồi gộp thành một `false` cho gọn. Lúc đó Grok — provider
// KHÔNG có rào duyệt quyền nào, một chuyện an ninh đã đo — trông y hệt Cursor mà
// máy này chưa đo được gì.
func TestBaTrangThaiSongSotQuaDTO(t *testing.T) {
	d := layNangLuc(t, "/api/nang-luc")
	dem := map[string]int{}
	for _, p := range d.Provider {
		for _, m := range p.Muc {
			dem[m.TrangThai]++
		}
	}
	for _, tt := range []string{
		string(provider.LamDuoc), string(provider.KhongLamDuoc), string(provider.ChuaDo),
	} {
		if dem[tt] == 0 {
			t.Errorf("DTO không mang ra trạng thái %q nào — ba trạng thái đang bị dẹp "+
				"thành hai ở lớp web", tt)
		}
	}
	if d.SoChuaDo != dem[string(provider.ChuaDo)] {
		t.Errorf("so_chua_do = %d nhưng đếm được %d — mỗi mặt cộng một kiểu thì "+
			"con số trên màn hình không tin được", d.SoChuaDo, dem[string(provider.ChuaDo)])
	}
}

// Lọc theo provider phải chạy, và tên lạ phải BÁO LỖI chứ không trả rỗng.
func TestDuongNangLucLocTheoProvider(t *testing.T) {
	d := layNangLuc(t, "/api/nang-luc?provider=grok")
	if len(d.Provider) != 1 || d.Provider[0].Provider != "grok" {
		t.Fatalf("lọc theo provider sai: %+v", d.Provider)
	}

	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := req("GET", "/api/nang-luc?provider=khong-co-provider-nay")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Fatalf("provider lạ phải báo lỗi, được 200: %s", w.Body.String())
	}
}

// Bảng này KHÔNG được chạm vào token của bất kỳ hồ sơ nào.
//
// Kiểm bằng GIÁ TRỊ THẬT chứ không bằng tên trường: bằng chứng của mỗi năng lực
// nhắc tới ".credentials.json" và "refreshTokenExpiresAt" là ĐÚNG — đó là mô tả
// chỗ phép đo lấy dữ liệu, không phải dữ liệu. Cấm tên trường thì chỉ khiến
// người ta viết bằng chứng mập mờ hơn, không an toàn hơn.
//
// Nên gieo token giả mang một dấu hiệu không thể trùng, rồi đòi dấu hiệu đó
// KHÔNG có mặt trong thân trả về. Đây cũng là phép đo cho một tính chất đáng
// giá khác: bảng năng lực trả lời được ngay cả khi chưa đăng nhập provider nào,
// vì nó không đọc file hồ sơ nào cả.
func TestNangLucKhongChamTokenCuaHoSoNao(t *testing.T) {
	s := newTestServer(t)
	home := os.Getenv("USERPROFILE")
	const dauHieu = "TOKEN-GIA-KHONG-DUOC-LO-RA-7c41"
	gieo := map[string]string{
		".credentials.json": `{"claudeAiOauth":{"accessToken":"` + dauHieu + `","expiresAt":1}}`,
		".claude.json":      `{"oauthAccount":{"emailAddress":"` + dauHieu + `@vd.com"}}`,
		filepath.Join(".grok", "user-settings.json"): `{"apiKey":"` + dauHieu + `","baseURL":"https://` + dauHieu + `"}`,
	}
	for ten, noi := range gieo {
		p := filepath.Join(home, ten)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(noi), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := req("GET", "/api/nang-luc")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("mã %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), dauHieu) {
		t.Fatal("/api/nang-luc mang nội dung file hồ sơ ra ngoài — bảng này chỉ được " +
			"mang chữ viết sẵn trong mã nguồn")
	}
}

// Chưa đăng nhập thì đường này cũng đóng như mọi đường /api/* khác.
func TestNangLucDoiDangNhap(t *testing.T) {
	s := newTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req("GET", "/api/nang-luc"))
	if w.Code != 401 {
		t.Fatalf("chưa đăng nhập phải 401, được %d", w.Code)
	}
}

// Mặt 2D phải THẬT SỰ gọi đường này. Cùng ý với duongPhaiCon trong
// chucnang_test.go: một nút mất chỉ lộ ra khi người dùng đi tìm nó, mà lúc đó
// không ai nhớ lần sửa nào làm mất.
func TestMat2DGoiDuongNangLuc(t *testing.T) {
	b, err := os.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	ma := string(b)
	if !strings.Contains(ma, "/api/nang-luc") {
		t.Fatal("index.html không gọi /api/nang-luc — bảng năng lực chỉ còn ở terminal")
	}
	if !strings.Contains(ma, "napNangLuc()") {
		t.Error("index.html khai hàm napNangLuc mà không gọi lúc tải — khối sẽ đứng " +
			"nguyên chữ \"đang đọc…\"")
	}
	// Ba trạng thái phải có ba lớp CSS khác nhau. Dùng chung một lớp cho
	// "không có" và "chưa đo" là xoá mất đúng thứ khối này sinh ra để nói.
	for _, lop := range []string{".nl .m.duoc", ".nl .m.khong", ".nl .m.chua"} {
		if !strings.Contains(ma, lop) {
			t.Errorf("index.html thiếu lớp %s — ba trạng thái không phân biệt được bằng mắt", lop)
		}
	}
	// Chỗ lệch phải hiện ra được, không giấu.
	if !strings.Contains(ma, "class = 'lech'") && !strings.Contains(ma, `'lech'`) {
		t.Error("index.html không vẽ phần `lech` — bảng năng lực sai sẽ hiện ra như bảng đúng")
	}
}
