package plugin

// TEST ĐẦU-CUỐI: build plugin mẫu THẬT, chạy nó như một tiến trình con THẬT, nói
// chuyện qua stdio THẬT.
//
// Vì sao không dùng plugin giả trong bộ nhớ: cái đáng hỏng ở đây nằm đúng chỗ mà
// một plugin giả bỏ qua — biến môi trường, thư mục làm việc, khung bản tin, ống
// đóng giữa chừng, hàng rào quyền. Một bài test nói "đã chạy plugin" mà chưa
// từng bật một tiến trình nào thì không đo được cái nào trong số đó, và bảng
// quyen.go sẽ thành một tờ giấy host tự cấp cho mình.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

const (
	tenMau    = "tom-luoc"
	tenDauMoc = "sagent-dau-moc.txt" // phải khớp TenDauMoc của cmd/sagent-plugin-mau
	tenBien   = "SAGENT_PLUGIN_MAU_DAU"
)

var (
	motLan      sync.Once
	duongMau    string
	loiBuild    error
	thuMucBuild string
)

// binMau build plugin mẫu MỘT lần cho cả gói test.
//
// Build một lần vì `go build` là phần chậm nhất ở đây (vài giây); mỗi test tự
// build là nhân con số đó lên bằng số test, và một bộ test chậm là một bộ test
// người ta bắt đầu bỏ qua.
func binMau(t *testing.T) string {
	t.Helper()
	motLan.Do(func() {
		thuMucBuild, loiBuild = os.MkdirTemp("", "sagent-plugin-build-")
		if loiBuild != nil {
			return
		}
		ra := filepath.Join(thuMucBuild, "sagent-plugin-mau")
		if runtime.GOOS == "windows" {
			ra += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", ra,
			"github.com/trantiendevweb/switch-agent-pro/cmd/sagent-plugin-mau")
		if raw, err := cmd.CombinedOutput(); err != nil {
			loiBuild = fmt.Errorf("build plugin mẫu hỏng: %v\n%s", err, raw)
			return
		}
		duongMau = ra
	})
	if loiBuild != nil {
		t.Fatal(loiBuild)
	}
	return duongMau
}

func TestMain(m *testing.M) {
	ma := m.Run()
	if thuMucBuild != "" {
		_ = os.RemoveAll(thuMucBuild)
	}
	os.Exit(ma)
}

// datPlugin dựng một thư mục plugin THẬT dưới goc: manifest + bản sao của
// executable đã build.
//
// Cùng một binary, nhiều manifest khác nhau — đó chính là cách đo hàng rào
// quyền: thứ duy nhất đổi giữa hai lần chạy là mấy dòng TOML.
func datPlugin(t *testing.T, goc, ten string, quyen []Quyen, secret []Secret) Manifest {
	t.Helper()
	bin := binMau(t)

	dir := filepath.Join(goc, ten)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// exec ghi TÊN TRẦN, không đuôi: trên Windows thì TimExec tự thử thêm .exe.
	// Nhờ vậy đúng manifest này chép sang máy khác vẫn dùng được.
	dich := filepath.Join(dir, "sagent-plugin-mau")
	if runtime.GOOS == "windows" {
		dich += ".exe"
	}
	chep(t, bin, dich)

	var sb strings.Builder
	sb.WriteString("version = 1\n\n[plugin]\n")
	fmt.Fprintf(&sb, "ten = %q\n", ten)
	sb.WriteString("mo_ta = \"tóm lược kết quả dài thành vài dòng cuối\"\n")
	sb.WriteString("phien_ban = \"0.1.0\"\n")
	fmt.Fprintf(&sb, "giao_thuc = %d\n", GiaoThuc)
	sb.WriteString("exec = \"sagent-plugin-mau\"\n")
	for _, q := range quyen {
		fmt.Fprintf(&sb, "\n[[quyen]]\nkhoa = %q\nly_do = %q\n", q.Khoa, q.LyDo)
	}
	for _, s := range secret {
		fmt.Fprintf(&sb, "\n[[secret]]\nten = %q\nkey_id = %q\n", s.Ten, s.KeyID)
	}

	duong := filepath.Join(dir, "plugin.toml")
	if err := os.WriteFile(duong, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Doc(duong)
	if err != nil {
		t.Fatalf("manifest vừa dựng đã không đọc được: %v", err)
	}
	return m
}

func chep(t *testing.T, tu, den string) {
	t.Helper()
	b, err := os.ReadFile(tu)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(den, b, 0o755); err != nil {
		t.Fatal(err)
	}
}

// duAnCoDauMoc dựng một thư mục dự án có sẵn file đánh dấu, để đo được tiến
// trình con THẬT SỰ đứng ở đâu (không chỉ nghe host tự khai).
func duAnCoDauMoc(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, tenDauMoc), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// chay mở plugin, gọi một lượt, đóng. Trả về kết quả để test soi.
func chay(t *testing.T, m Manifest, opt TuyChon, vao string, thamSo map[string]string) KetQuaChay {
	t.Helper()
	ctx, huy := context.WithTimeout(context.Background(), 30*time.Second)
	defer huy()
	c, err := Mo(ctx, m, opt)
	if err != nil {
		t.Fatalf("Mo: %v", err)
	}
	defer c.Dong()
	kq, err := c.Chay(ctx, vao, thamSo)
	if err != nil {
		t.Fatalf("Chay: %v", err)
	}
	return kq
}

// ------------------------- giao thức -------------------------

// Lát cắt DỌC ngắn nhất: manifest → tiến trình con → bắt tay có phiên bản → một
// lượt việc → kết quả có cấu trúc.
func TestBatTayRoiChayMotLuotThat(t *testing.T) {
	m := datPlugin(t, t.TempDir(), tenMau, nil, nil)

	ctx := context.Background()
	c, err := Mo(ctx, m, TuyChon{})
	if err != nil {
		t.Fatalf("Mo: %v", err)
	}
	defer c.Dong()

	if c.Ten != tenMau {
		t.Errorf("bắt tay trả tên %q, chờ %q", c.Ten, tenMau)
	}
	if c.PhienBan == "" {
		t.Error("plugin không nói phiên bản của nó — bản ghi lượt chạy sẽ không truy được")
	}

	vao := "dong 1\ndong 2\n\ndong 3\ndong 4\ndong 5\ndong 6 KET LUAN"
	kq, err := c.Chay(ctx, vao, map[string]string{"so_dong": "2"})
	if err != nil {
		t.Fatalf("Chay: %v", err)
	}
	if !strings.Contains(kq.Ra, "dong 6 KET LUAN") {
		t.Errorf("tóm lược mất dòng cuối — chính là dòng có kết luận:\n%s", kq.Ra)
	}
	if strings.Contains(kq.Ra, "dong 1") {
		t.Errorf("xin 2 dòng cuối mà vẫn còn dòng đầu — tham_so không tới nơi:\n%s", kq.Ra)
	}
	if !strings.Contains(kq.Ra, "TÓM LƯỢC") {
		t.Errorf("kết quả không phải của plugin mẫu:\n%s", kq.Ra)
	}
}

// Gọi HAI lượt trên cùng một tiến trình: id phải khớp từng lượt.
//
// Đây là chỗ dễ hỏng nhất của một client JSON-RPC tự viết — trả lời của lượt
// trước còn nằm trong ống thì lượt sau đọc nhầm, và cái sai đó KHÔNG hiện ra ở
// bài test chỉ gọi một lượt.
func TestHaiLuotGoiLienTiepKhongLanKetQua(t *testing.T) {
	m := datPlugin(t, t.TempDir(), tenMau, nil, nil)
	ctx := context.Background()
	c, err := Mo(ctx, m, TuyChon{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Dong()

	mot, err := c.Chay(ctx, "alpha", nil)
	if err != nil {
		t.Fatal(err)
	}
	hai, err := c.Chay(ctx, "beta", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mot.Ra, "alpha") || !strings.Contains(hai.Ra, "beta") {
		t.Errorf("hai lượt lẫn kết quả:\n#1 %s\n#2 %s", mot.Ra, hai.Ra)
	}
}

// Plugin trả lỗi thì host phải trả lỗi — không được nuốt rồi trả chuỗi rỗng.
func TestPluginBaoLoiThiHostBaoLoi(t *testing.T) {
	m := datPlugin(t, t.TempDir(), tenMau, nil, nil)
	ctx := context.Background()
	c, err := Mo(ctx, m, TuyChon{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Dong()

	kq, err := c.Chay(ctx, "   ", nil)
	if err == nil {
		t.Fatalf("đầu vào rỗng mà plugin vẫn báo xong, trả về %q", kq.Ra)
	}
	if !strings.Contains(err.Error(), "rỗng") {
		t.Errorf("lỗi không giữ nguyên văn của plugin: %v", err)
	}
	// Sau một lỗi, tiến trình phải còn sống và còn làm việc được: lỗi CỦA MỘT
	// LƯỢT không được thành lỗi của cả phiên.
	if _, err := c.Chay(ctx, "con song", nil); err != nil {
		t.Errorf("một lượt hỏng làm chết cả tiến trình: %v", err)
	}
}

// Executable XIN nhiều quyền hơn manifest KHAI thì host phải từ chối.
//
// Không dựng bằng tiến trình thật vì phải có một plugin mẫu thứ hai chuyên đi
// nói dối — và một binary như vậy trong repo là thứ người ta sẽ chép nhầm. Đây
// là phép kiểm thuần logic của host, nên kiểm thẳng vào nó.
func TestExecXinQuyenNgoaiManifestThiTuChoi(t *testing.T) {
	m := Manifest{
		Version: PhienBanManifest,
		Plugin:  ThongTin{Ten: tenMau, MoTa: "x", GiaoThuc: GiaoThuc, Exec: "x"},
		Quyen:   []Quyen{{Khoa: QuyenThuMuc, LyDo: "đọc file kết quả"}},
		Duong:   filepath.Join(t.TempDir(), "plugin.toml"),
	}
	c := &Client{m: m}

	err := c.doiChieu(KetQuaBatTay{GiaoThuc: GiaoThuc, Ten: tenMau,
		Quyen: []string{QuyenThuMuc, QuyenMang}})
	if err == nil {
		t.Fatal("plugin xin thêm quyền `mang` mà host vẫn nhận — bản duyệt manifest thành giấy lộn")
	}
	if !strings.Contains(err.Error(), QuyenMang) {
		t.Errorf("lỗi không chỉ ra quyền nào bị xin lén: %v", err)
	}

	// Xin ÍT hơn thì được: manifest là trần, không phải hạn ngạch phải tiêu hết.
	if err := c.doiChieu(KetQuaBatTay{GiaoThuc: GiaoThuc, Ten: tenMau}); err != nil {
		t.Errorf("plugin xin ít hơn manifest mà bị chặn: %v", err)
	}
}

// Lệch số phiên bản GIAO THỨC thì dừng ngay lúc bắt tay.
func TestLechGiaoThucThiDungNgayLucBatTay(t *testing.T) {
	m := Manifest{
		Version: PhienBanManifest,
		Plugin:  ThongTin{Ten: tenMau, MoTa: "x", GiaoThuc: GiaoThuc, Exec: "x"},
		Duong:   filepath.Join(t.TempDir(), "plugin.toml"),
	}
	c := &Client{m: m}
	err := c.doiChieu(KetQuaBatTay{GiaoThuc: GiaoThuc + 1, Ten: tenMau})
	if err == nil {
		t.Fatal("plugin nói giao thức khác mà host vẫn chạy tiếp")
	}
	if !strings.Contains(err.Error(), "giao thức") {
		t.Errorf("thông điệp không nói ra chuyện lệch giao thức: %v", err)
	}
}

// Executable tự xưng tên khác manifest thì từ chối — nếu không, một plugin có
// thể mượn danh plugin khác mà bảng `sagent plugin` vẫn hiện tên cũ.
func TestExecTuXungTenKhacManifestThiTuChoi(t *testing.T) {
	m := Manifest{
		Version: PhienBanManifest,
		Plugin:  ThongTin{Ten: tenMau, MoTa: "x", GiaoThuc: GiaoThuc, Exec: "x"},
		Duong:   filepath.Join(t.TempDir(), "plugin.toml"),
	}
	c := &Client{m: m}
	if err := c.doiChieu(KetQuaBatTay{GiaoThuc: GiaoThuc, Ten: "ai-do-khac"}); err == nil {
		t.Fatal("tên executable khác tên manifest mà host vẫn nhận")
	}
}

// ------------------------- hàng rào quyền -------------------------

// ĐÂY LÀ PHÉP ĐO của dòng thu-muc-lam-viec trong quyen.go.
//
// Cùng một binary, cùng một thư mục dự án, khác đúng ba dòng TOML.
func TestKhongKhaiThuMucThiKhongThayThuMucDuAn(t *testing.T) {
	duAn := duAnCoDauMoc(t)

	khong := datPlugin(t, t.TempDir(), tenMau, nil, nil)
	kq := chay(t, khong, TuyChon{ThuMuc: duAn}, "x", nil)
	if !strings.Contains(kq.GhiChu, "thu-muc=(khong-cap)") {
		t.Errorf("không khai quyền mà host vẫn đưa đường dẫn dự án: %s", kq.GhiChu)
	}
	if !strings.Contains(kq.GhiChu, "cwd-thay-dau-moc=khong") {
		t.Errorf("không khai quyền mà tiến trình con VẪN ĐỨNG trong thư mục dự án "+
			"(thấy %s) — cmd.Dir đang để rỗng ở đâu đó: %s", tenDauMoc, kq.GhiChu)
	}

	co := datPlugin(t, t.TempDir(), tenMau,
		[]Quyen{{Khoa: QuyenThuMuc, LyDo: "đọc file kết quả trong dự án"}}, nil)
	kq = chay(t, co, TuyChon{ThuMuc: duAn}, "x", nil)
	if !strings.Contains(kq.GhiChu, "thu-muc="+duAn) {
		t.Errorf("khai quyền rồi mà không nhận được thư mục — hàng rào chặn cả đường hợp lệ: %s", kq.GhiChu)
	}
	if !strings.Contains(kq.GhiChu, "cwd-thay-dau-moc=co") {
		t.Errorf("khai quyền rồi mà tiến trình vẫn không đứng trong thư mục dự án: %s", kq.GhiChu)
	}
}

// ĐÂY LÀ PHÉP ĐO của dòng bien-moi-truong trong quyen.go.
func TestKhongKhaiMoiTruongThiKhongThayBienCuaCha(t *testing.T) {
	t.Setenv(tenBien, "1")

	khong := datPlugin(t, t.TempDir(), tenMau, nil, nil)
	kq := chay(t, khong, TuyChon{}, "x", nil)
	if !strings.Contains(kq.GhiChu, "bien-danh-dau=khong") {
		t.Errorf("không khai quyền mà plugin vẫn thấy biến môi trường của host: %s", kq.GhiChu)
	}

	co := datPlugin(t, t.TempDir(), tenMau,
		[]Quyen{{Khoa: QuyenMoiTruong, LyDo: "cần biến CI của môi trường chạy"}}, nil)
	kq = chay(t, co, TuyChon{}, "x", nil)
	if !strings.Contains(kq.GhiChu, "bien-danh-dau=co") {
		t.Errorf("khai quyền rồi mà vẫn không thấy biến của host: %s", kq.GhiChu)
	}
}

// ĐÂY LÀ PHÉP ĐO của dòng secret trong quyen.go.
//
// Kiểm cả hai chiều, và chiều thứ hai mới là chiều đáng lo: không khai thì host
// KHÔNG ĐƯỢC MỞ kho key. Đọc rồi mới quyết định không đưa cũng đã là một lần
// chạm vào bí mật không cần thiết.
func TestKhongKhaiSecretThiKhongNhanDuocGiaTri(t *testing.T) {
	const giaTri = "gia-tri-bi-mat-that-khong-duoc-lo"
	daDoc := false
	doc := func(id string) (string, error) {
		daDoc = true
		if id != "kho-thu" {
			return "", fmt.Errorf("key_id lạ %q", id)
		}
		return giaTri, nil
	}

	khong := datPlugin(t, t.TempDir(), tenMau, nil, nil)
	kq := chay(t, khong, TuyChon{DocSecret: doc}, "x", nil)
	if !strings.Contains(kq.GhiChu, "secret=(khong-co)") {
		t.Errorf("không khai secret mà plugin vẫn nhận được gì đó: %s", kq.GhiChu)
	}
	if daDoc {
		t.Error("không khai secret mà host VẪN mở kho key — chạm vào bí mật không cần chạm")
	}

	co := datPlugin(t, t.TempDir(), tenMau,
		[]Quyen{{Khoa: QuyenSecret, LyDo: "gọi API của bên thứ ba"}},
		[]Secret{{Ten: "token", KeyID: "kho-thu"}})
	kq = chay(t, co, TuyChon{DocSecret: doc}, "x", nil)
	if !daDoc {
		t.Error("khai secret rồi mà host không đọc kho key")
	}
	mong := fmt.Sprintf("secret=token:%d", len(giaTri))
	if !strings.Contains(kq.GhiChu, mong) {
		t.Errorf("plugin không nhận đúng secret (chờ %q): %s", mong, kq.GhiChu)
	}
	// Và giá trị KHÔNG được quay ngược ra bản ghi: ghi_chu đi thẳng vào event
	// bus, tức là vào log và vào mặt web.
	if strings.Contains(kq.GhiChu, giaTri) || strings.Contains(kq.Ra, giaTri) {
		t.Error("giá trị secret lọt ra kết quả/ghi chú của plugin")
	}
}

// Thiếu key thì hỏng NGAY, trước khi có tiến trình con nào được bật.
func TestThieuKeyThiHongTruocKhiBatTienTrinh(t *testing.T) {
	m := datPlugin(t, t.TempDir(), tenMau,
		[]Quyen{{Khoa: QuyenSecret, LyDo: "gọi API"}},
		[]Secret{{Ten: "token", KeyID: "khong-co-trong-kho"}})

	_, err := Mo(context.Background(), m, TuyChon{
		DocSecret: func(string) (string, error) { return "", fmt.Errorf("không đọc được key") },
	})
	if err == nil {
		t.Fatal("thiếu key mà vẫn mở được plugin")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("lỗi không nói thiếu secret nào: %v", err)
	}
}

// ------------------------- cắm vào flow -------------------------

// LÁT CẮT DỌC ĐẦY ĐỦ, và là bằng chứng cho implemented[TypePlugin] = true:
// flow.Runner thật → node `plugin` → BoChay → tiến trình con → kết quả chuyền
// sang bước sau.
func TestFlowChayPluginThat(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	db, err := store.OpenAt(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	bus := events.NewBus()
	defer bus.Close()

	duAn := t.TempDir()
	datPlugin(t, filepath.Join(home, ".ai-accounts", "plugins"), tenMau, nil, nil)

	var ghiChu []string
	r := &flow.Runner{DB: db, Bus: bus, Agent: khongCoAgent{},
		Plugin: &BoChay{Dir: duAn, Ghi: func(m string) { ghiChu = append(ghiChu, m) }}}

	// Bước đầu chỉ để SINH RA một khối dài; dùng notify vì nó chạy được ở mọi
	// máy mà không cần lệnh ngoài nào — thứ đang đo là đường đi của kết quả, không
	// phải cách sinh ra nó.
	var dai []string
	for i := 1; i <= 40; i++ {
		dai = append(dai, fmt.Sprintf("dong %d", i))
	}
	f := flow.Flow{Name: "co-plugin", Steps: []flow.Step{
		{ID: "sinh", Type: flow.TypeNotify, Message: strings.Join(dai, "\n")},
		{ID: "gon", Type: flow.TypePlugin, Needs: []string{"sinh"},
			Plugin: tenMau, Vao: "{{steps.sinh.output}}",
			ThamSo: map[string]string{"so_dong": "3"}},
	}}
	if ps := flow.Validate(f); len(ps) > 0 {
		for _, p := range ps {
			t.Errorf("flow hợp lệ mà validate kêu: %s", p)
		}
	}

	res, err := r.Start(context.Background(), f, duAn, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		buoc, _ := db.Steps(res.RunID)
		t.Fatalf("lượt chạy không xong: %s (%+v)", res.State, buoc)
	}

	buoc, err := db.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	ra := buoc["gon"].Output
	if !strings.Contains(ra, "TÓM LƯỢC") {
		t.Fatalf("bước plugin không trả về thứ của plugin mẫu:\n%s", ra)
	}
	// 40 dòng vào, xin 3 dòng ra: nếu bước chỉ chuyền nguyên đầu vào (một lỗi
	// rất dễ mắc khi nối dây) thì dòng 1 vẫn còn đó.
	if !strings.Contains(ra, "dong 40") {
		t.Errorf("mất dòng cuối của đầu vào:\n%s", ra)
	}
	if strings.Contains(ra, "dong 1\n") {
		t.Errorf("plugin không cắt gì cả — đầu vào đi thẳng thành đầu ra:\n%s", ra)
	}
	if len(ghiChu) == 0 {
		t.Error("ghi_chu của plugin không tới được người vận hành")
	}
}

// Bước plugin gọi tên không có thì phải hỏng với một thông điệp CHỈ ĐƯỜNG, chứ
// không phải "không tìm thấy" trống rỗng.
func TestGoiPluginKhongCoThiNoiRoTimODau(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	datPlugin(t, filepath.Join(home, ".ai-accounts", "plugins"), tenMau, nil, nil)

	b := &BoChay{Dir: t.TempDir()}
	_, err := b.GoiPlugin(context.Background(), "khong-ton-tai", "x", nil)
	if err == nil {
		t.Fatal("gọi plugin không có mà không hỏng")
	}
	if !strings.Contains(err.Error(), tenMau) || !strings.Contains(err.Error(), "tìm ở") {
		t.Errorf("thông điệp không nói đang có plugin nào và tìm ở đâu:\n%v", err)
	}
}

// khongCoAgent là AgentRunner giả — flow này không có bước agent nào, và nếu có
// thì test phải đỏ chứ không được lặng lẽ chạy.
type khongCoAgent struct{}

func (khongCoAgent) RunAgents(context.Context, string, string, string, int, bool, bool) (flow.KetQuaAgent, error) {
	return flow.KetQuaAgent{}, fmt.Errorf("test này không được gọi agent")
}
