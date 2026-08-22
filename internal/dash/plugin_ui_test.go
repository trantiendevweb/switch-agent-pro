package dash

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
	"github.com/trantiendevweb/switch-agent-pro/internal/plugin"
)

// dtoPlugin là hình dạng mặt web TRÔNG ĐỢI ở /api/plugins. Đọc qua struct chứ
// không qua map[string]any, cùng lý do với dtoNangLuc: đổi tên một trường trong
// handler thì test đỏ ngay, thay vì trang lặng lẽ hiện "undefined".
type dtoPlugin struct {
	Muc []struct {
		Ten      string `json:"ten"`
		PhienBan string `json:"phien_ban"`
		Quyen    []struct {
			Khoa      string `json:"khoa"`
			Mo        string `json:"mo"`
			LyDo      string `json:"ly_do"`
			Chan      string `json:"chan"`
			BangChung string `json:"bang_chung"`
		} `json:"quyen"`
		Secret []string `json:"secret"`
	} `json:"muc"`
	SoChanThat      int `json:"so_chan_that"`
	SoKhongChanDuoc int `json:"so_khong_chan_duoc"`
	SoChuaDo        int `json:"so_chua_do"`
	SoChuaChan      int `json:"so_chua_chan"`
}

// datPluginMau dựng một manifest THẬT trong kho plugin toàn cục của HOME test.
//
// Không cần executable: bảng quyền là thứ đọc được TRƯỚC khi plugin chạy, và
// đó chính là lúc người vận hành cần nó.
func datPluginMau(t *testing.T, ten string, quyen [][2]string, secret [][2]string) {
	t.Helper()
	dir := filepath.Join(paths.AccountsRoot(), "plugins", ten)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	sb.WriteString("version = 1\n\n[plugin]\n")
	fmt.Fprintf(&sb, "ten = %q\n", ten)
	sb.WriteString("mo_ta = \"plugin dựng riêng cho bài kiểm bảng quyền\"\n")
	sb.WriteString("phien_ban = \"9.9.9\"\n")
	fmt.Fprintf(&sb, "giao_thuc = %d\n", plugin.GiaoThuc)
	sb.WriteString("exec = \"khong-co-that\"\n")
	for _, q := range quyen {
		fmt.Fprintf(&sb, "\n[[quyen]]\nkhoa = %q\nly_do = %q\n", q[0], q[1])
	}
	for _, s := range secret {
		fmt.Fprintf(&sb, "\n[[secret]]\nten = %q\nkey_id = %q\n", s[0], s[1])
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.toml"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// pluginBaTrangThai dựng một plugin xin ĐỦ CẢ BA trạng thái chặn, rồi trả về
// thân /api/plugins đã đọc.
func pluginBaTrangThai(t *testing.T) (*Server, dtoPlugin) {
	t.Helper()
	s := newTestServer(t)
	datPluginMau(t, "mau-ba-trang-thai", [][2]string{
		{plugin.QuyenMoiTruong, "đọc biến PATH của tiến trình cha"},  // ChanThat
		{plugin.QuyenThuMuc, "đọc file kết quả trong thư mục dự án"}, // KhongChanDuoc
		{plugin.QuyenMang, "gọi API tóm lược của bên thứ ba"},        // ChuaDo
		{plugin.QuyenSecret, "lấy khoá gọi API đó"},                  // ChanThat
	}, [][2]string{{"token", "grok"}})

	ck := dangNhap(t, s, "127.0.0.1:4600")
	r := req("GET", "/api/plugins")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("GET /api/plugins trả %d: %s", w.Code, w.Body.String())
	}
	var d dtoPlugin
	if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatalf("thân trả về không đọc được: %v\n%s", err, w.Body.String())
	}
	return s, d
}

// BA trạng thái của cột [chặn] phải sống sót qua lớp DTO.
//
// Làm theo đúng lối của TestDTOKhongDepTrangThaiNaoThanhTrangThaiKhac: đối
// chiếu TỪNG MỤC giữa lớp API và lớp DTO, không đếm số plugin và không đòi
// "phải thấy đủ ba trạng thái trong dữ liệu sản phẩm" (đòi thế thì bài kiểm chỉ
// xanh chừng nào dự án CÒN NỢ, và nó phạt đúng việc đóng nợ).
//
// Vì sao chỗ này đáng canh: JSON hoá thì ai đó thấy "khong-chan-duoc" và
// "chua-do" đều KHÔNG phải chan-that, rồi gộp thành một cờ `chan: false` cho
// gọn. Lúc ấy `thu-muc-lam-viec` — ĐÃ ĐO 22/08, host không chặn được, plugin
// không xin quyền nào vẫn đọc được 33 byte và ghi được file trong thư mục dự án
// — trông y hệt `mang`, thứ chưa ai đo. Hai dòng đó đòi hai việc ngược nhau:
// một cái đi dựng hàng rào, một cái đi đo xem có dựng được không.
func TestDTOPluginKhongDepBaTrangThaiChanThanhHai(t *testing.T) {
	s, d := pluginBaTrangThai(t)

	// NGUỒN SỰ THẬT là chính lớp API mà handler gọi, với ĐÚNG thư mục handler
	// dùng — nếu không thì hai bên đang đọc hai kho plugin khác nhau.
	goc := (&api.API{}).Plugins(s.workDir())

	mongDoi := map[string]string{}
	for _, m := range goc.Muc {
		for _, q := range m.Quyen {
			mongDoi[m.Ten+"/"+q.Khoa] = q.Chan
		}
	}

	// Chống "xanh vì rỗng": bản mẫu ở trên phải thật sự sinh ra cả ba trạng
	// thái, nếu không thì bài kiểm này không đo được cú bẹp nào cả.
	coTrangThai := map[string]bool{}
	for _, tt := range mongDoi {
		coTrangThai[tt] = true
	}
	for _, tt := range []plugin.TrangThaiChan{plugin.ChanThat, plugin.KhongChanDuoc, plugin.ChuaDo} {
		if !coTrangThai[string(tt)] {
			t.Fatalf("bản mẫu không sinh ra trạng thái %q — bài kiểm này đang xanh vì "+
				"thiếu dữ liệu, không phải vì lớp web trung thực. Sửa datPluginMau.", tt)
		}
	}

	var soChanThat, soKhongChanDuoc, soChuaDo, soMuc int
	for _, m := range d.Muc {
		for _, q := range m.Quyen {
			soMuc++
			khoa := m.Ten + "/" + q.Khoa
			muon, co := mongDoi[khoa]
			if !co {
				t.Errorf("DTO có mục %q mà lớp API không có", khoa)
				continue
			}
			if q.Chan != muon {
				t.Errorf("%s: lớp API nói chan=%q, DTO nói %q — lớp web đang đổi trạng thái",
					khoa, muon, q.Chan)
			}
			switch plugin.TrangThaiChan(q.Chan) {
			case plugin.ChanThat:
				soChanThat++
			case plugin.KhongChanDuoc:
				soKhongChanDuoc++
			default:
				soChuaDo++
			}
			// Ba trường này là toàn bộ nội dung một hàng vẽ ra. Thiếu bằng chứng
			// thì "chặn thật" chỉ là một lời hứa, còn "chưa đo" thì không ai biết
			// phải làm gì tiếp — đúng luật của provider.NangLuc.
			if q.Mo == "" || q.BangChung == "" || q.LyDo == "" {
				t.Errorf("%s: DTO thiếu trường (mo=%q ly_do=%q bang_chung=%q)",
					khoa, q.Mo, q.LyDo, q.BangChung)
			}
		}
	}
	if soMuc != len(mongDoi) {
		t.Errorf("DTO mang %d dòng quyền, lớp API có %d — lớp web đang nuốt hoặc nhân bản dòng",
			soMuc, len(mongDoi))
	}

	// Ba con số phải là BA NGĂN RIÊNG. Đây là chỗ cú bẹp hay xảy ra lần thứ hai:
	// dữ liệu từng dòng thì đúng, nhưng con số liếc-một-cái ở đầu khối lại cộng
	// "không chặn được" với "chưa đo" làm một.
	if d.SoChanThat != soChanThat {
		t.Errorf("so_chan_that = %d nhưng đếm được %d", d.SoChanThat, soChanThat)
	}
	if d.SoKhongChanDuoc != soKhongChanDuoc {
		t.Errorf("so_khong_chan_duoc = %d nhưng đếm được %d", d.SoKhongChanDuoc, soKhongChanDuoc)
	}
	if d.SoChuaDo != soChuaDo {
		t.Errorf("so_chua_do = %d nhưng đếm được %d", d.SoChuaDo, soChuaDo)
	}
	if d.SoChuaChan != soKhongChanDuoc+soChuaDo {
		t.Errorf("so_chua_chan = %d nhưng %d + %d = %d — tên nó nói là TỔNG của hai ngăn sau",
			d.SoChuaChan, soKhongChanDuoc, soChuaDo, soKhongChanDuoc+soChuaDo)
	}
	// Cả hai nửa phải khác 0, nếu không thì phép so tổng ở trên vẫn xanh khi ai
	// đó cho ba con số bằng nhau.
	if soKhongChanDuoc == 0 || soChuaDo == 0 {
		t.Fatalf("bản mẫu cho khong_chan_duoc=%d chua_do=%d — cần cả hai khác 0 thì "+
			"phép so tổng ở trên mới có nghĩa", soKhongChanDuoc, soChuaDo)
	}
}

// Bảng quyền KHÔNG được mang giá trị secret ra trình duyệt — chỉ tên tham chiếu.
func TestPluginDTOChiMangTenSecretKhongMangGiaTri(t *testing.T) {
	_, d := pluginBaTrangThai(t)
	var thay bool
	for _, m := range d.Muc {
		for _, s := range m.Secret {
			thay = true
			// Server ghép "<ten> -> key_id <id>". Tên kho key là thứ được phép
			// thấy; giá trị thì nằm ở ~/.ai-accounts/api-keys và không bao giờ
			// đi qua route này.
			if !strings.Contains(s, "key_id") {
				t.Errorf("dòng secret %q không nói rõ đây là key_id — người đọc sẽ "+
					"tưởng mình đang nhìn giá trị", s)
			}
		}
	}
	if !thay {
		t.Fatal("bản mẫu có [[secret]] mà DTO không trả dòng nào — bài kiểm này đang xanh vì rỗng")
	}
}

// docWeb đọc index.html một lần.
func docWeb(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Mặt web phải vẽ BA trạng thái thành BA thứ NHÌN THẤY KHÁC NHAU.
//
// Không đếm số plugin: số plugin đúng mà cả ba trạng thái cùng một màu xám thì
// trang vẫn vô dụng — tệ hơn là không có trang nào, vì nó cho người ta cảm giác
// đã kiểm tra.
//
// Đi theo ĐÚNG CHUỖI mà trình duyệt đi: giá trị chan (lấy từ hằng Go, không chép
// tay) tới lớp CSS trong PL_LOP, rồi tới luật màu `.pl .m.<lớp>`. Đứt khúc nào
// cũng đỏ.
func TestMatWebVeBaTrangThaiChanThanhBaThuKhacNhau(t *testing.T) {
	s := docWeb(t)

	lop := banhTraJS(t, s, "PL_LOP")
	doc := banhTraJS(t, s, "PL_DOC")

	moiTrangThai := []plugin.TrangThaiChan{plugin.ChanThat, plugin.KhongChanDuoc, plugin.ChuaDo}
	mau := map[string]string{}   // token màu -> trạng thái đã chiếm
	daLop := map[string]string{} // lớp CSS  -> trạng thái đã chiếm
	daDoc := map[string]string{} // nhãn chữ -> trạng thái đã chiếm

	for _, tt := range moiTrangThai {
		k := string(tt)
		l, co := lop[k]
		if !co {
			t.Errorf("PL_LOP không có %q — hằng trong internal/plugin đổi tên mà mặt web "+
				"không đổi theo, trạng thái này sẽ rơi về nhánh mặc định", k)
			continue
		}
		if cu, trung := daLop[l]; trung {
			t.Errorf("PL_LOP cho %q và %q CÙNG lớp %q — hai trạng thái này bị bẹp làm một "+
				"ngay trên màn hình", cu, k, l)
		}
		daLop[l] = k

		d, co := doc[k]
		if !co {
			t.Errorf("PL_DOC không có %q — hàng sẽ hiện nhãn mặc định", k)
		} else if cu, trung := daDoc[d]; trung {
			t.Errorf("PL_DOC cho %q và %q cùng đọc là %q — chấm màu có khác nhau thì "+
				"chữ vẫn nói hai thứ đó là một", cu, k, d)
		} else {
			daDoc[d] = k
		}

		// Lớp CSS phải có luật màu THẬT, và màu phải là một token trạng thái.
		re := regexp.MustCompile(`\.pl \.m\.` + regexp.QuoteMeta(l) + `\{--c:var\((--[a-z0-9-]+)\)\}`)
		m := re.FindStringSubmatch(s)
		if m == nil {
			t.Errorf("không có luật `.pl .m.%s{--c:var(--...)}` — trạng thái %q sẽ vẽ ra "+
				"không màu, trông giống hàng bên cạnh", l, k)
			continue
		}
		if cu, trung := mau[m[1]]; trung {
			t.Errorf("trạng thái %q và %q cùng ăn màu %s — BA trạng thái phải là BA màu",
				cu, k, m[1])
		}
		mau[m[1]] = k

		// Và chú thích dưới bảng phải có đúng chấm ấy, nếu không thì màu trên
		// bảng là một mật mã không ai giải được.
		if !strings.Contains(s, `class="ch `+l+`"`) {
			t.Errorf("chú thích thiếu chấm `ch %s` cho trạng thái %q", l, k)
		}
		if !regexp.MustCompile(`\.pl-ct \.ch\.` + regexp.QuoteMeta(l) + `\{--c:var\(`).MatchString(s) {
			t.Errorf("chấm chú thích `ch %s` không có luật màu — nó sẽ vẽ ra trong suốt", l)
		}
	}

	// Nhánh mặc định phải là CHƯA ĐO, không phải chặn thật: thứ không đọc được
	// phải đếm về phía "không biết".
	macDinh := regexp.MustCompile(`function plLop\(c\)\{ return PL_LOP\[c\] \|\| '([a-z]+)'; \}`).
		FindStringSubmatch(s)
	if macDinh == nil {
		t.Fatal("không tìm thấy plLop() — bảng tra không còn nhánh mặc định nào")
	}
	if muon := lop[string(plugin.ChuaDo)]; macDinh[1] != muon {
		t.Errorf("plLop rơi về %q khi gặp giá trị lạ, phải là %q (chưa đo) — giá trị "+
			"không đọc được mà rơi về phía 'đã chặn' là bảng tự hứa hộ host",
			macDinh[1], muon)
	}
}

// banhTraJS đọc một bảng tra JS dạng `const TEN = {'a':'b', ...};` thành map.
func banhTraJS(t *testing.T, s, ten string) map[string]string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^const ` + regexp.QuoteMeta(ten) + ` = \{([^}]*)\};`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		t.Fatalf("index.html không còn bảng tra %s", ten)
	}
	out := map[string]string{}
	cap := regexp.MustCompile(`'([^']*)'\s*:\s*'([^']*)'`)
	for _, c := range cap.FindAllStringSubmatch(m[1], -1) {
		out[c[1]] = c[2]
	}
	if len(out) == 0 {
		t.Fatalf("bảng tra %s rỗng", ten)
	}
	return out
}

// Con số liếc-một-cái ở đầu khối cũng KHÔNG được bẹp.
//
// so_chua_chan là TỔNG của hai ngăn "không chặn được" và "chưa đo". Hiện mình nó
// lên màn hình là làm đúng cái việc mà cả khối này dựng ra để chặn — nên mặt web
// phải đọc HAI số rời, và phải KHÔNG đọc cái tổng.
func TestMatWebKhongHienConSoDaGopCuaCotChan(t *testing.T) {
	s := boComment(docWeb(t))
	for _, truong := range []string{"so_khong_chan_duoc", "so_chua_do"} {
		if !strings.Contains(s, truong) {
			t.Errorf("index.html không đọc %s — không có nó thì hai nửa của cột [chặn] "+
				"không thể hiện rời", truong)
		}
	}
	if strings.Contains(s, "so_chua_chan") {
		t.Error("index.html đọc so_chua_chan — đó là TỔNG của 'không chặn được' và " +
			"'chưa đo'. Hiện một con số cho hai chuyện đòi hai cách xử lý ngược nhau " +
			"là bẹp ba trạng thái thành hai ngay trên màn hình.")
	}
}

// Một dòng phải nói thẳng: khai quyền không phải là hàng rào.
//
// Câu này đã có trong bản in của `sagent plugin quyen`. Mặt web hiện danh sách
// quyền mà KHÔNG nói câu đó thì bảng đọc như một bản cam kết an ninh — người
// vận hành thấy [[quyen]] liệt kê đầy đủ và tưởng host đang giữ hàng rào.
func TestMatWebNoiRoKhaiQuyenKhongPhaiHangRao(t *testing.T) {
	s := docWeb(t)
	// Đối chiếu với chính bản CLI, không chép tay hai câu rồi để chúng trôi xa
	// nhau: hai mặt phải nói cùng một điều bằng cùng một chữ.
	const cau = "Khai một quyền trong manifest KHÔNG tự nó là một hàng rào."
	cli, err := os.ReadFile(filepath.Join("..", "..", "cmd", "sagent", "plugin.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cli), cau) {
		t.Fatalf("cmd/sagent/plugin.go không còn câu %q — sửa cả hai mặt cùng lúc", cau)
	}
	// Phải nằm trong phần HIỆN RA, không phải trong một dòng bình luận HTML.
	if !strings.Contains(boCommentHTML(s), cau) {
		t.Errorf("index.html không hiện câu %q ở chỗ người dùng đọc được", cau)
	}
}

// boCommentHTML cắt mọi khối bình luận HTML.
//
// Cần nó vì bản đầu của bài kiểm trên KHÔNG CẮN: câu cần tìm nằm sẵn trong một
// bình luận giải thích, nên test xanh dù trang không hiện chữ nào. Cùng cái bẫy
// mà boComment (JS) sinh ra để tránh.
var reCommentHTML = regexp.MustCompile(`(?s)<!--.*?-->`)

func boCommentHTML(s string) string { return reCommentHTML.ReplaceAllString(s, " ") }

// Khối plugin phải THẬT SỰ được nạp, không chỉ được định nghĩa.
//
// napFlow() từng được định nghĩa mà không gọi — ô quy trình rỗng trơn, trang vẫn
// vẽ đầy đủ nên không ai báo lỗi. Đây đúng kiểu hỏng dự án này sợ nhất.
func TestMatWebNapBangQuyenPluginLucMoTrang(t *testing.T) {
	ma := boComment(docWeb(t))
	if !strings.Contains(ma, "'/api/plugins'") {
		t.Error("index.html không gọi '/api/plugins' trong mã thật")
	}
	if !regexp.MustCompile(`(?m)^\s*napPlugin\(\)\s*;`).MatchString(ma) {
		t.Error("napPlugin() được định nghĩa nhưng không được gọi ở cấp cao nhất — " +
			"khối plugin sẽ đứng ở 'đang đọc…' vĩnh viễn")
	}
	// Đường này phải có thật bên server, nếu không bài kiểm đang canh một cái
	// tên chết.
	srv, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(srv), "/api/plugins") {
		t.Error("server.go không còn /api/plugins")
	}
}
