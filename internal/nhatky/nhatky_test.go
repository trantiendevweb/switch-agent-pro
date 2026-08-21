package nhatky

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// datHome trỏ thư mục người dùng vào chỗ tạm. Phải đặt ĐÚNG biến của từng nền
// tảng — đặt nhầm thì test trông như chạy mà thật ra đang đo (và xoá) HOME thật,
// mà gói này có hàm xoá file.
func datHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}
	return home
}

// Phép đo cốt lõi của gói: nhật ký phải nằm NGOÀI thư mục clone.
//
// Đây không phải chuyện gọn gàng. `sagent clean` xoá nguyên thư mục clone, và
// đó đúng là lệnh người ta chạy sau một lượt hỏng — tức lệnh dọn dẹp bình
// thường lại là lệnh phá tang chứng. Nhật ký nằm cạnh state.db thì sống sót.
func TestNhatKyNamNgoaiThuMucCloneVaNgoaiRepo(t *testing.T) {
	home := datHome(t)

	root := Root()
	if !strings.HasPrefix(root, paths.AccountsRoot()) {
		t.Fatalf("Root() = %q, không nằm trong kho hồ sơ %q", root, paths.AccountsRoot())
	}
	// `.clones` là chỗ `sagent clean` xoá. Nhật ký mà rơi vào đó là quay lại
	// đúng lỗi ngày 21/08.
	clones := filepath.Join(paths.AccountsRoot(), ".clones")
	if strings.HasPrefix(root, clones) {
		t.Fatalf("Root() = %q nằm trong %q — `sagent clean` sẽ xoá mất nhật ký", root, clones)
	}
	if filepath.Dir(root) != paths.AccountsRoot() {
		t.Fatalf("Root() = %q, không nằm ngay dưới %q", root, paths.AccountsRoot())
	}
	// Dấu chấm đầu tên: `profile.List()` quét kho hồ sơ và sẽ coi mọi thư mục
	// không có dấu chấm là một provider. Mất dấu chấm là mọc ra provider ma.
	if !strings.HasPrefix(filepath.Base(root), ".") {
		t.Fatalf("tên thư mục %q không bắt đầu bằng dấu chấm — profile.List() sẽ nhầm nó là provider",
			filepath.Base(root))
	}
	// Và tuyệt đối không nằm trong repo người dùng đang làm việc.
	if !strings.HasPrefix(root, home) {
		t.Fatalf("Root() = %q nằm ngoài HOME %q", root, home)
	}
}

// LỖI THẬT ngày 21/08: phiên #169 chạy trên bản clone 1 XOÁ TRẮNG nhật ký của
// phiên #167 cũng chạy trên bản clone 1, vì đường dẫn cũ chỉ phụ thuộc số clone.
//
// Hai lượt khác nhau PHẢI ra hai đường dẫn khác nhau.
func TestHaiLuotTrenCungBanCloneRaHaiDuongDanKhacNhau(t *testing.T) {
	datHome(t)
	if err := os.MkdirAll(Root(), 0o755); err != nil {
		t.Fatal(err)
	}

	t1 := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	p1 := Duong("claude", "tns", 1, t1)
	if err := Tao(p1, Dau{ThoiDiem: t1, Addr: "claude:tns#1", Lenh: []string{"-p", "việc 1"}}); err != nil {
		t.Fatal(err)
	}

	t2 := t1.Add(90 * time.Second)
	p2 := Duong("claude", "tns", 1, t2)
	if p1 == p2 {
		t.Fatalf("hai lượt cùng bản clone dùng chung một file %q — lượt sau xoá nhật ký lượt trước", p1)
	}
	if err := Tao(p2, Dau{ThoiDiem: t2, Addr: "claude:tns#1", Lenh: []string{"-p", "việc 2"}}); err != nil {
		t.Fatal(err)
	}
	// Cả hai phải CÒN đọc được. Đây mới là điều ca #167/#169 cần.
	for _, p := range []string{p1, p2} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("nhật ký %q biến mất: %v", p, err)
		}
	}
}

// Cùng một mốc thời gian TỚI MILI GIÂY (đồng hồ thô, hoặc hai lượt bật sát
// nhau): vẫn không được trùng file.
func TestTrungMocThoiGianThiThemHauToChuKhongDeLen(t *testing.T) {
	datHome(t)
	moc := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)

	p1 := Duong("claude", "tns", 1, moc)
	if err := Tao(p1, Dau{ThoiDiem: moc, Addr: "a", Lenh: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	p2 := Duong("claude", "tns", 1, moc)
	if p1 == p2 {
		t.Fatalf("file %q đã tồn tại mà Duong vẫn trả về đúng nó", p1)
	}
}

// Hai BẢN CLONE của cùng một lượt bật trong cùng một giây là chuyện bình thường
// — số bản phải nằm trong tên để chúng không thể trùng.
func TestHaiBanCloneCungLuotKhongTrungTen(t *testing.T) {
	datHome(t)
	moc := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	seen := map[string]bool{}
	for i := 1; i <= 4; i++ {
		p := Duong("claude", "tns", i, moc)
		if seen[p] {
			t.Fatalf("bản clone %d trùng đường dẫn %q với bản trước", i, p)
		}
		seen[p] = true
	}
}

// Tên tài khoản có ký tự lạ không được đẻ ra đường dẫn ngoài thư mục nhật ký.
func TestTenTaiKhoanLaKhongThoatRaNgoaiThuMuc(t *testing.T) {
	datHome(t)
	p := Duong("claude", "../../hiểm", 1, time.Now())
	if filepath.Dir(p) != Root() {
		t.Fatalf("Duong() = %q, thoát ra khỏi %q", p, Root())
	}
}

// ĐÂY là chỗ ca #167 tự lộ ra: khối tiêu đề mang DÒNG LỆNH đã dựng xong.
//
// Bản ghi của agent không chứa thông tin này — nó chỉ biết những gì nó nhận
// được, không biết những gì lẽ ra nó phải nhận. Thiếu `--tu-duyet-quyen` chỉ
// nhìn thấy được ở đây.
func TestKhoiTieuDeMangDongLenhDaDung(t *testing.T) {
	datHome(t)
	moc := time.Date(2026, 8, 21, 11, 39, 5, 0, time.UTC)
	p := Duong("claude", "tns", 1, moc)
	d := Dau{
		ThoiDiem: moc, Addr: "claude:tns#1",
		HoSo: `C:\clones\claude\tns\1`, ThuMuc: `C:\wt\tns-1`,
		Lenh: []string{"-p", "sửa lỗi X", "--output-format", "stream-json"},
	}
	if err := Tao(p, d); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, phai := range []string{
		"claude:tns#1", `C:\clones\claude\tns\1`, `C:\wt\tns-1`,
		"-p sửa lỗi X --output-format stream-json", MocDau, MocHet,
	} {
		if !strings.Contains(got, phai) {
			t.Errorf("khối tiêu đề thiếu %q:\n%s", phai, got)
		}
	}
	// Mọi dòng của khối phải bắt đầu bằng "#": đó là điều kiện để bộ đọc bản
	// ghi có cấu trúc (provider.DocKetQua chỉ nhận dòng mở đầu bằng "{") bỏ qua
	// tự nhiên, không phải nhờ một luật riêng ở đâu đó.
	for _, dong := range strings.Split(strings.TrimRight(got, "\n"), "\n") {
		if !strings.HasPrefix(dong, "#") {
			t.Errorf("dòng %q trong khối tiêu đề không bắt đầu bằng '#' — bộ đọc kết quả sẽ vấp phải nó", dong)
		}
	}
}

// Nối thêm chứ không cắt trắng: đây là hành vi đã cứu ca #167.
func TestTaoLanHaiNoiThemChuKhongCatTrang(t *testing.T) {
	datHome(t)
	moc := time.Now()
	p := Duong("claude", "tns", 1, moc)
	if err := Tao(p, Dau{ThoiDiem: moc, Addr: "a", Lenh: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"type":"result","result":"xong"}` + "\n")
	f.Close()

	// Gọi Tao LẠI đúng đường dẫn đó (không xảy ra trong sản phẩm vì Duong tránh
	// trùng, nhưng đây là phép đo trực tiếp hành vi cắt/không cắt).
	if err := Tao(p, Dau{ThoiDiem: moc, Addr: "a", Lenh: []string{"y"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if !strings.Contains(string(raw), `"type":"result"`) {
		t.Fatalf("nội dung cũ bị cắt trắng khi mở lại file:\n%s", raw)
	}
}

// GhiLoi phải nối được lý do chết vào cuối, kể cả khi agent chưa in chữ nào.
func TestGhiLoiGiuLaiLyDoChetSagentBiet(t *testing.T) {
	datHome(t)
	moc := time.Now()
	p := Duong("claude", "tns", 1, moc)
	if err := Tao(p, Dau{ThoiDiem: moc, Addr: "a", Lenh: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	if err := GhiLoi(p, "không bật được tiến trình: access is denied"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if !strings.Contains(string(raw), "access is denied") {
		t.Fatalf("lý do chết không vào được nhật ký:\n%s", raw)
	}
	// Vẫn phải là chữ của SAGENT, không lẫn vào chữ của agent.
	cuoi := strings.TrimRight(string(raw), "\n")
	cuoi = cuoi[strings.LastIndex(cuoi, "\n")+1:]
	if !strings.HasPrefix(cuoi, "#") {
		t.Fatalf("dòng lý do %q không mang dấu '#' — sẽ bị đọc nhầm là output của agent", cuoi)
	}
}

func TestBoDauCatDungKhoiTieuDe(t *testing.T) {
	d := Dau{ThoiDiem: time.Now(), Addr: "claude:tns#1", Lenh: []string{"-p", "việc"}}
	than := `{"type":"result","result":"xong"}` + "\n"

	if got := BoDau(d.String() + than); got != than {
		t.Fatalf("BoDau() = %q, chờ %q", got, than)
	}
	// Không có khối thì trả nguyên văn — không được cắt nhầm log của phiên cũ.
	if got := BoDau(than); got != than {
		t.Fatalf("nội dung không có khối tiêu đề bị sửa: %q", got)
	}
	// Khối chưa đóng (phiên chết ngay lúc đang ghi): vẫn không được để lọt chữ
	// của sagent ra ngoài.
	cut := MocDau + "\n# địa chỉ:   claude:tns#1\n"
	if got := BoDau(cut); got != "" {
		t.Fatalf("khối chưa đóng mà BoDau trả về %q", got)
	}
	if got := BoDau(cut + than); got != than {
		t.Fatalf("khối chưa đóng + thân: BoDau() = %q, chờ %q", got, than)
	}
}

func TestDuoiLayDongCuoi(t *testing.T) {
	raw := "a\nb\nc\nd\ne\n"
	if got := Duoi(raw, 2); got != "d\ne" {
		t.Fatalf("Duoi(2) = %q", got)
	}
	if got := Duoi(raw, 99); got != "a\nb\nc\nd\ne" {
		t.Fatalf("xin nhiều hơn số dòng có mà mất chữ: %q", got)
	}
	if got := Duoi(raw, 0); got != raw {
		t.Fatalf("Duoi(0) phải trả nguyên văn, được %q", got)
	}
}

// ghiFile tạo một nhật ký giả với kích thước và mốc thời gian định trước.
func ghiFile(t *testing.T, ten string, co int, khi time.Time) string {
	t.Helper()
	p := filepath.Join(Root(), ten)
	if err := os.MkdirAll(Root(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, make([]byte, co), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, khi, khi); err != nil {
		t.Fatal(err)
	}
	return p
}

func conLai(t *testing.T) []string {
	t.Helper()
	es, err := os.ReadDir(Root())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

// Trần SỐ FILE: cũ nhất bị xoá trước.
func TestDonGiuDungSoFileVaXoaCuNhatTruoc(t *testing.T) {
	datHome(t)
	goc := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		ghiFile(t, "p"+string(rune('0'+i))+".log", 10, goc.Add(time.Duration(i)*time.Hour))
	}
	kq, err := Don(nil, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if kq.DaXoa != 2 || kq.ConLai != 3 {
		t.Fatalf("Don() = %+v, chờ xoá 2 còn 3", kq)
	}
	con := strings.Join(conLai(t), " ")
	for _, phai := range []string{"p2.log", "p3.log", "p4.log"} {
		if !strings.Contains(con, phai) {
			t.Errorf("xoá nhầm file mới %q — còn lại: %s", phai, con)
		}
	}
	for _, khong := range []string{"p0.log", "p1.log"} {
		if strings.Contains(con, khong) {
			t.Errorf("file cũ %q lẽ ra phải bị dọn — còn lại: %s", khong, con)
		}
	}
}

// Trần TỔNG BYTE: ràng buộc nào chạm trước thì có hiệu lực.
func TestDonGiuDungTranDungLuong(t *testing.T) {
	datHome(t)
	goc := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		ghiFile(t, "q"+string(rune('0'+i))+".log", 100, goc.Add(time.Duration(i)*time.Hour))
	}
	// 250 byte đủ cho 2 file (200), file thứ 3 vượt trần.
	kq, err := Don(nil, 0, 250)
	if err != nil {
		t.Fatal(err)
	}
	if kq.ConLai != 2 || kq.DaXoa != 2 {
		t.Fatalf("Don() = %+v, chờ còn 2 xoá 2", kq)
	}
	if kq.ByteCon > 250 {
		t.Fatalf("còn %d byte, quá trần 250", kq.ByteCon)
	}
}

// Nhật ký của phiên CÒN ĐANG CHẠY không được xoá, dù nó nằm ngoài ngân sách.
//
// Thiếu vế này thì một lượt fleet dài bị lượt sau xoá nhật ký ngay giữa lúc
// đang ghi — tức là tính năng tự phá chính nó ở đúng ca nó sinh ra để phục vụ.
func TestDonKhongDungToiNhatKyPhienDangChay(t *testing.T) {
	datHome(t)
	goc := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	// r0 là file CŨ NHẤT, tức đứa đầu tiên bị nhắm tới khi quá trần.
	dangChay := ghiFile(t, "r0.log", 10, goc)
	for i := 1; i < 5; i++ {
		ghiFile(t, "r"+string(rune('0'+i))+".log", 10, goc.Add(time.Duration(i)*time.Hour))
	}
	kq, err := Don([]string{dangChay}, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dangChay); err != nil {
		t.Fatalf("xoá mất nhật ký của phiên đang chạy: %v", err)
	}
	if kq.DaXoa == 0 {
		t.Fatal("giữ được file đang chạy nhưng cũng không dọn gì cả")
	}
}

// Chưa chạy fleet lần nào: không có thư mục, và đó KHÔNG phải lỗi.
func TestDonKhiChuaCoThuMucThiKhongPhaiLoi(t *testing.T) {
	datHome(t)
	kq, err := Don(nil, 10, 0)
	if err != nil {
		t.Fatalf("thư mục chưa tồn tại mà báo lỗi: %v", err)
	}
	if kq.DaXoa != 0 {
		t.Fatalf("xoá %d file trong một thư mục không tồn tại", kq.DaXoa)
	}
}

// Chỉ đụng file `.log` — thư mục nhật ký nằm cạnh state.db, và một phép dọn
// quét bừa ở đó là một phép xoá dữ liệu người dùng.
func TestDonChiDungToiFileLog(t *testing.T) {
	datHome(t)
	goc := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	ghiFile(t, "giu-lai.txt", 999, goc)
	for i := 0; i < 3; i++ {
		ghiFile(t, "s"+string(rune('0'+i))+".log", 10, goc.Add(time.Duration(i+1)*time.Hour))
	}
	if _, err := Don(nil, 1, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(Root(), "giu-lai.txt")); err != nil {
		t.Fatalf("dọn nhật ký mà xoá cả file không phải .log: %v", err)
	}
}
