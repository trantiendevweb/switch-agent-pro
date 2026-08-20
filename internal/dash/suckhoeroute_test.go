package dash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Sáu test dưới đây giữ MẶT WEB của action "route.kiem".
//
// Vì sao chúng phải tồn tại: endpoint /api/route/kiem có từ trước, CLI có
// `sagent route kiem` từ trước, và TestMoiHanhDongDeuCoDuongVaoTuWeb
// (lachan_test.go) vẫn XANH suốt thời gian đó — vì nó chỉ hỏi "đường HTTP có
// trả khác 404 không", không hỏi "có ai bấm được đường đó không". Kết quả:
// dashboard không có một chỗ nào gọi tới, và luật ngang quyền bốn mặt
// (MASTER-PLAN 2c) bị vi phạm ngay trong repo có test canh nó.
//
// Đây đúng kiểu hỏng mà dự án sợ nhất: im lặng, và trông có vẻ ổn.

// Khoá JSON mà handleRouteKiem gửi ra. Đổi tên một khoá ở server mà quên trang
// thì trang đọc undefined — không lỗi đỏ, chỉ là mọi route hiện "0ms" hoặc mất
// hẳn lý do hỏng. Danh sách này bắt cả hai chiều.
var khoaRouteKiem = []string{
	"ten", "dung", "song", "status", "matMs", "model", "soModel", "khongRo", "gan", "loi",
}

// coModel KHÔNG nằm trong danh sách trên, và đó là cố ý: `dung` đã gộp sẵn
// (Dung() = Song && (CoModel || KhongRo)), nên trang không cần đọc lại cờ thô.
// Ghi ra đây để lần sau không ai tưởng là bỏ sót.
const lyDoBoCoModel = "dung đã gộp coModel; trang chỉ cần kết luận, không cần cờ thô"

// thanHam cắt lấy thân MỘT hàm JS, từ dòng khai báo tới dấu `}` đứng một mình
// ở đầu dòng.
//
// Phải chịu được cả hai kiểu xuống dòng: web/index.html trên máy này lưu CRLF,
// nên cắt bằng "\n}\n" trần thì không khớp — và cái không khớp đó im lặng trả
// về CẢ PHẦN CÒN LẠI của trang, khiến mọi phép kiểm "trong hàm này có X không"
// đều đọc nhầm sang hàm khác. Đúng kiểu xanh giả / đỏ giả mà file này canh.
func thanHam(t *testing.T, s, dau string) string {
	t.Helper()
	i := strings.Index(s, dau)
	if i < 0 {
		t.Fatalf("index.html không có %q", dau)
	}
	than := s[i:]
	if loc := regexp.MustCompile(`\r?\n\}\r?\n`).FindStringIndex(than); loc != nil {
		than = than[:loc[0]]
	}
	return than
}

func TestMatWebCoDuongVaoRouteKiem(t *testing.T) {
	s := doc2D(t)
	if !strings.Contains(s, "/api/route/kiem") {
		t.Fatal("index.html không gọi /api/route/kiem — endpoint và CLI đều có, " +
			"riêng mặt web không có chỗ nào bấm được")
	}
	// Phải có KHỐI hiện kết quả và NÚT gọi nó, không chỉ là một chuỗi nằm trong
	// chú thích: một đường dẫn viết trong comment thì test tìm chuỗi vẫn xanh.
	for _, id := range []string{`id="rk"`, `id="rk-nap"`, `id="rk-dem"`} {
		if !strings.Contains(s, id) {
			t.Errorf("index.html thiếu %s — khối sức khoẻ route không dựng đủ", id)
		}
	}
	if !regexp.MustCompile(`\$\('rk-nap'\)\.onclick\s*=\s*napRouteKiem`).MatchString(s) {
		t.Error("nút #rk-nap không nối vào napRouteKiem — nút vẽ ra mà bấm không ra gì")
	}
	// Nút phải nằm TRONG ngăn kéo: sáu ô thao tác đã dọn hết vào đó, để hở một
	// nút ngoài màn chính là phá đúng bố cục vừa dọn xong.
	i := strings.Index(s, `id="ngankeo"`)
	if i < 0 {
		t.Fatal("index.html: không có #ngankeo")
	}
	if !strings.Contains(s[i:], `id="rk-nap"`) {
		t.Error("nút Kiểm route nằm ngoài #ngankeo")
	}
}

// Hợp đồng DTO: server gửi khoá nào thì trang phải đọc đúng khoá đó.
func TestSucKhoeRouteDocDungKhoaDTO(t *testing.T) {
	trang := doc2D(t)
	b, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	sv := string(b)
	// Chỉ soi đúng thân handleRouteKiem, không soi cả file: khoá "model" xuất
	// hiện ở chục endpoint khác, soi cả file thì test xanh giả.
	j := strings.Index(sv, "func (s *Server) handleRouteKiem(")
	if j < 0 {
		t.Fatal("server.go không còn handleRouteKiem — mặt web mất nguồn dữ liệu")
	}
	than := sv[j:]
	if k := strings.Index(than, "\nfunc "); k > 0 {
		than = than[:k]
	}

	for _, k := range khoaRouteKiem {
		if !regexp.MustCompile(`json:"` + k + `[,"]`).MatchString(than) {
			t.Errorf("handleRouteKiem không còn gửi khoá %q — trang vẫn đọc nó và sẽ nhận undefined", k)
		}
		if !regexp.MustCompile(`\.`+k+`\b`).MatchString(trang) {
			t.Errorf("index.html không đọc khoá %q của /api/route/kiem", k)
		}
	}
	if strings.Contains(trang, ".coModel") {
		t.Errorf("index.html đọc coModel — %s", lyDoBoCoModel)
	}
}

// Ba trạng thái phải là BA màu khác nhau.
//
// "Chết" và "sống nhưng model khai không có" cùng dẫn tới một hậu quả (gọi là
// hỏng) nhưng cách sửa ngược nhau: cái đầu đợi hoặc đổi route, cái sau sửa một
// dòng trong .sagent/project.toml. Gộp thành một màu đỏ là xoá đúng thứ đáng
// giá nhất mà phép kiểm này đo được — y hệt lý do bảng năng lực tách "đã đo,
// không có" khỏi "chưa ai đo".
func TestBaTrangThaiRouteKhongBiGop(t *testing.T) {
	s := doc2D(t)
	re := regexp.MustCompile(`#rk \.d\.(duoc|song|chet)\{--c:var\((--[a-z]+)\)\}`)
	thay := map[string]string{}
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		thay[m[1]] = m[2]
	}
	for _, lop := range []string{"duoc", "song", "chet"} {
		if thay[lop] == "" {
			t.Errorf("index.html: thiếu luật màu cho trạng thái route %q", lop)
		}
	}
	if len(thay) < 3 {
		return
	}
	nguoc := map[string]string{}
	for lop, tk := range thay {
		if cu, trung := nguoc[tk]; trung {
			t.Errorf("trạng thái %q và %q dùng chung màu %s — hai thứ đó sửa theo hai cách khác nhau, "+
				"cùng màu là bảo người dùng đi sai đường", cu, lop, tk)
		}
		nguoc[tk] = lop
	}
	// Và ba màu đó phải là token trạng thái dùng chung, không phải mã màu chép tay.
	for lop, tk := range thay {
		if !strings.Contains(",--run,--pending,--error,--idle,--done,", ","+tk+",") {
			t.Errorf("trạng thái %q tô bằng %s — không phải token trạng thái dùng chung", lop, tk)
		}
	}
}

// Phép kiểm này chạm MẠNG THẬT ra tới từng nhà cung cấp, nên nó KHÔNG được tự
// chạy lúc mở trang.
//
// Đo được: dashboard tự poll 5 giây/lần và người dùng thường mở 2-3 tab. Nếu
// napRouteKiem nằm trong khối tự chạy cuối trang thì mỗi lần mở tab là một loạt
// request đi ra Internet — cùng lý do khiến "Quét tiến trình mồ côi" phải do
// người dùng bấm.
func TestKiemRouteKhongTuChayLucMoTrang(t *testing.T) {
	s := doc2D(t)
	// Chỉ bắt LỜI GỌI `napRouteKiem();` — dòng khai báo `async function
	// napRouteKiem(){` cũng chứa chuỗi "napRouteKiem()", tìm trần là đỏ giả.
	if regexp.MustCompile(`napRouteKiem\(\)\s*;`).MatchString(s) {
		t.Error("napRouteKiem() bị gọi thẳng ở đâu đó — nó phải chờ người dùng bấm nút, " +
			"vì mỗi lần chạy là một loạt request đi ra tới từng nhà cung cấp")
	}
	// Nút phải khoá lại trong lúc đang hỏi: bấm liên tiếp là nhân đôi số request
	// mạng, mà kết quả lượt trước còn chưa về.
	than := thanHam(t, s, "async function napRouteKiem()")
	if !strings.Contains(than, "nut.disabled = true") || !strings.Contains(than, "nut.disabled = false") {
		t.Error("napRouteKiem không khoá nút trong lúc chờ — bấm liên tiếp sẽ bắn chồng request mạng")
	}
}

// Lý do hỏng phải NGUYÊN VĂN: request id của nhà cung cấp nằm trong đó, và đó
// là thứ duy nhất dùng được khi phải đi hỏi họ.
func TestLyDoHongInNguyenVanVaKhongBiCat(t *testing.T) {
	s := doc2D(t)
	than := thanHam(t, s, "function rkDong(")
	if !strings.Contains(than, "textContent") {
		t.Error("rkDong không dùng textContent — lý do hỏng là chữ của nhà cung cấp, " +
			"đổ vào innerHTML là mở cửa cho thẻ lạ")
	}
	if strings.Contains(than, "innerHTML") {
		t.Error("rkDong dùng innerHTML cho chữ do nhà cung cấp trả về")
	}
	// CSS không được cắt lý do bằng ellipsis: cắt mất request id thì dòng đó vô dụng.
	re := regexp.MustCompile(`(?s)#rk \.ly\{(.*?)\}`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		t.Fatal("index.html: thiếu luật CSS cho #rk .ly")
	}
	if !strings.Contains(m[1], "pre-wrap") {
		t.Error("#rk .ly không cho xuống dòng (white-space:pre-wrap)")
	}
	if strings.Contains(m[1], "ellipsis") {
		t.Error("#rk .ly cắt bằng ellipsis — request id nằm ở cuối chuỗi, cắt là mất")
	}
}

// Đường thật vẫn phải trả JSON đúng hình dạng trang đang đọc: {"muc": [...]}.
//
// Máy test không khai route nào nên danh sách rỗng — nhưng khoá "muc" phải có
// mặt, vì trang làm `d.muc || []`. Mất khoá đó thì trang im lặng hiện "Chưa khai
// route nào" kể cả khi máy thật có mười route.
func TestEndpointRouteKiemTraKhoaMuc(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := req("GET", "/api/route/kiem")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/route/kiem: mã %d, muốn 200", w.Code)
	}
	var d map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatalf("thân trả về không phải JSON: %v", err)
	}
	if _, co := d["muc"]; !co {
		t.Errorf("thiếu khoá \"muc\" — trang đọc d.muc, mất khoá là bảng rỗng im lặng. Được: %s", w.Body.String())
	}
}
