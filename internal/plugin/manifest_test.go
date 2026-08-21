package plugin

// Test phần TĨNH: manifest đọc được gì, và — quan trọng hơn — TỪ CHỐI gì.
//
// Phần lớn giá trị của một lược đồ manifest nằm ở chỗ nó từ chối. Đọc được một
// file đúng là chuyện dễ; im lặng nuốt một file sai mới là chuyện làm người ta
// mất buổi chiều, vì file trông như đã có tác dụng.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// vietManifest ghi nội dung thô ra <dir>/<ten>/plugin.toml rồi đọc lại.
func vietManifest(t *testing.T, ten, noiDung string) (Manifest, error) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ten)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	duong := filepath.Join(dir, "plugin.toml")
	if err := os.WriteFile(duong, []byte(noiDung), 0o644); err != nil {
		t.Fatal(err)
	}
	return Doc(duong)
}

const manifestToiThieu = `version = 1

[plugin]
ten = "thu"
mo_ta = "plugin để test"
phien_ban = "0.1.0"
giao_thuc = 1
exec = "thu"
`

func TestManifestToiThieuDocDuoc(t *testing.T) {
	m, err := vietManifest(t, "thu", manifestToiThieu)
	if err != nil {
		t.Fatalf("manifest hợp lệ mà không đọc được: %v", err)
	}
	if m.Plugin.Ten != "thu" || m.Plugin.GiaoThuc != GiaoThuc {
		t.Errorf("đọc sai: %+v", m.Plugin)
	}
	if len(m.Quyen) != 0 {
		t.Error("không khai quyền nào mà lại có quyền — mặc định phải là KHÔNG CÓ GÌ")
	}
}

// RÀNG BUỘC 1 + 3: khoá lạ trong TOML phải làm cả file hỏng.
//
// Hai ca dưới đây là hai cách người ta sẽ thử phá luật: nhét GIÁ TRỊ bí mật vào
// chỗ đáng lẽ chỉ có tham chiếu, và nhét LOGIC vào chỗ đáng lẽ chỉ có cấu hình.
// Nếu bộ đọc chỉ bỏ qua khoá lạ (mặc định của thư viện TOML), cả hai đều nằm im
// trong file: người viết tin là nó chạy, người soi tin là nó đã được xử lý.
func TestManifestTuChoiKhoaLa(t *testing.T) {
	ca := map[string]string{
		"secret dạng giá trị": manifestToiThieu + `
[[secret]]
ten = "token"
key_id = "kho"
gia_tri = "sk-that-la-bi-mat"
`,
		"logic trong TOML": manifestToiThieu + `
[hook]
truoc_khi_chay = "rm -rf /"
`,
		"khoá lạ trong [plugin]": `version = 1

[plugin]
ten = "thu"
mo_ta = "x"
giao_thuc = 1
exec = "thu"
lenh = "bash -c 'x'"
`,
	}
	for ten, noiDung := range ca {
		t.Run(ten, func(t *testing.T) {
			_, err := vietManifest(t, "thu", noiDung)
			if err == nil {
				t.Fatal("khoá lạ mà manifest vẫn đọc được — TOML đang nuốt im lặng")
			}
			if !strings.Contains(err.Error(), "khoá lạ") {
				t.Errorf("lỗi không nói ra chuyện khoá lạ: %v", err)
			}
		})
	}
}

// RÀNG BUỘC 4, chỗ CHẶN THẬT của secret: khai [[secret]] mà không khai quyền thì
// manifest hỏng ngay lúc đọc.
//
// Đây mới là hàng rào, chứ không phải nhánh kiểm trong docSecret: nhờ nó, đọc
// riêng khối `quyen` là thấy HẾT thứ plugin được nhận. Không có nó thì một
// [[secret]] nằm cuối file là một quyền không xuất hiện trong bảng quyền.
func TestKhaiSecretMaKhongKhaiQuyenThiTuChoi(t *testing.T) {
	_, err := vietManifest(t, "thu", manifestToiThieu+`
[[secret]]
ten = "token"
key_id = "kho"
`)
	if err == nil {
		t.Fatal("khai secret mà không khai quyền vẫn đọc được — bảng quyền nói thiếu một quyền")
	}
	if !strings.Contains(err.Error(), QuyenSecret) {
		t.Errorf("lỗi không chỉ ra thiếu quyền nào: %v", err)
	}
}

// Chiều ngược lại: xin quyền secret mà không có secret nào là xin thừa.
func TestXinQuyenSecretMaKhongCoSecretNaoThiTuChoi(t *testing.T) {
	_, err := vietManifest(t, "thu", manifestToiThieu+`
[[quyen]]
khoa = "secret"
ly_do = "gọi API"
`)
	if err == nil {
		t.Fatal("xin quyền thừa mà vẫn nhận — quyền tối thiểu thành quyền tuỳ thích")
	}
}

// RÀNG BUỘC 3: key_id là TÊN TRẦN, không phải đường dẫn.
func TestKeyIDKhongDuocLaDuongDan(t *testing.T) {
	for _, xau := range []string{"../../etc/passwd", `C:\keys\that`, "kho/con", ".."} {
		_, err := vietManifest(t, "thu", manifestToiThieu+`
[[quyen]]
khoa = "secret"
ly_do = "gọi API"

[[secret]]
ten = "token"
key_id = "`+xau+`"
`)
		if err == nil {
			t.Errorf("key_id %q lọt qua — nó dùng để MỞ FILE bí mật", xau)
		}
	}
}

// exec phải nằm trong thư mục plugin.
func TestExecPhaiNamTrongThuMucPlugin(t *testing.T) {
	xau := []string{`C:\Windows\System32\cmd.exe`, "/bin/sh", "../../ai-do-khac", ""}
	for _, e := range xau {
		_, err := vietManifest(t, "thu", `version = 1

[plugin]
ten = "thu"
mo_ta = "x"
giao_thuc = 1
exec = "`+e+`"
`)
		if err == nil {
			t.Errorf("exec = %q lọt qua — manifest chép từ chỗ khác về sẽ chạy binary có sẵn trên máy", e)
		}
	}
}

// Quyền phải có thật và phải nói lý do.
func TestQuyenPhaiCoThatVaPhaiNoiLyDo(t *testing.T) {
	if _, err := vietManifest(t, "thu", manifestToiThieu+`
[[quyen]]
khoa = "toan-quyen"
ly_do = "cho tiện"
`); err == nil {
		t.Error("quyền bịa mà vẫn nhận")
	}
	if _, err := vietManifest(t, "thu", manifestToiThieu+`
[[quyen]]
khoa = "mang"
ly_do = "   "
`); err == nil {
		t.Error("xin quyền không nói lý do mà vẫn nhận — người duyệt không có gì để duyệt")
	}
}

// Giao thức lệch thì hỏng NGAY LÚC ĐỌC, không phải giữa lượt chạy.
func TestGiaoThucLechThiHongNgayLucDoc(t *testing.T) {
	_, err := vietManifest(t, "thu", `version = 1

[plugin]
ten = "thu"
mo_ta = "x"
giao_thuc = 99
exec = "thu"
`)
	if err == nil {
		t.Fatal("manifest nói giao thức 99 mà host vẫn nhận")
	}
	if !strings.Contains(err.Error(), "giao_thuc") {
		t.Errorf("lỗi không nói ra chuyện giao thức: %v", err)
	}
}

// version của LƯỢC ĐỒ cũng vậy.
func TestVersionLuocDoLechThiTuChoi(t *testing.T) {
	if _, err := vietManifest(t, "thu", strings.Replace(manifestToiThieu, "version = 1", "version = 2", 1)); err == nil {
		t.Error("lược đồ version 2 mà công cụ chỉ hiểu 1 vẫn nhận")
	}
}

// ------------------------- nạp theo tầng -------------------------

func datManifest(t *testing.T, goc, ten, moTa string) {
	t.Helper()
	dir := filepath.Join(goc, ten)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	noi := `version = 1

[plugin]
ten = "` + ten + `"
mo_ta = "` + moTa + `"
giao_thuc = 1
exec = "x"
`
	if err := os.WriteFile(filepath.Join(dir, "plugin.toml"), []byte(noi), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Dự án đè kho toàn cục — cùng luật với flows.toml và project.toml, để người
// dùng không phải học một thứ tự thứ hai.
func TestNapUuTienDuAnDeLenKho(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	duAn := t.TempDir()
	sagent := filepath.Join(duAn, ".sagent")
	if err := os.MkdirAll(sagent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sagent, "project.toml"), []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	datManifest(t, filepath.Join(home, ".ai-accounts", "plugins"), "chung", "bản của KHO")
	datManifest(t, filepath.Join(home, ".ai-accounts", "plugins"), "rieng-kho", "chỉ có ở kho")
	datManifest(t, filepath.Join(sagent, "plugins"), "chung", "bản của DỰ ÁN")

	ds, nguon, loi := Nap(duAn)
	if len(loi) > 0 {
		t.Fatalf("manifest hợp lệ mà báo lỗi: %v", loi)
	}
	if len(nguon) != 2 {
		t.Errorf("phải tìm ở 2 nơi, thấy %v", nguon)
	}
	if got := ds["chung"].Plugin.MoTa; got != "bản của DỰ ÁN" {
		t.Errorf("dự án phải đè kho, nhận %q", got)
	}
	if _, co := ds["rieng-kho"]; !co {
		t.Error("plugin chỉ có ở kho bị mất khi dự án cũng có plugin")
	}
}

// Tên trong manifest phải khớp tên thư mục, nếu không một plugin mượn được danh
// của plugin khác.
func TestTenManifestPhaiKhopTenThuMuc(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	kho := filepath.Join(home, ".ai-accounts", "plugins")
	datManifest(t, kho, "that", "x")
	// thư mục "gia" nhưng bên trong tự xưng là "that"
	dir := filepath.Join(kho, "gia")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	noi := "version = 1\n\n[plugin]\nten = \"that\"\nmo_ta = \"kẻ mượn danh\"\ngiao_thuc = 1\nexec = \"x\"\n"
	if err := os.WriteFile(filepath.Join(dir, "plugin.toml"), []byte(noi), 0o644); err != nil {
		t.Fatal(err)
	}

	ds, _, loi := Nap(t.TempDir())
	if len(loi) == 0 {
		t.Fatal("plugin mượn danh mà không ai kêu")
	}
	if ds["that"].Plugin.MoTa == "kẻ mượn danh" {
		t.Error("kẻ mượn danh chiếm được chỗ trong bảng tra")
	}
}

// Một manifest hỏng KHÔNG được làm biến mất các plugin còn lại.
func TestMotManifestHongKhongLamMatPluginKhac(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	kho := filepath.Join(home, ".ai-accounts", "plugins")
	datManifest(t, kho, "lanh-lan", "vẫn tốt")

	dir := filepath.Join(kho, "hong")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.toml"), []byte("khong phai toml {{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	ds, _, loi := Nap(t.TempDir())
	if len(loi) != 1 {
		t.Errorf("chờ đúng 1 lỗi, được %v", loi)
	}
	if _, co := ds["lanh-lan"]; !co {
		t.Error("một manifest hỏng làm mất plugin lành lặn — người dùng sẽ tưởng mình chưa cài gì")
	}
}

// ------------------------- bảng quyền -------------------------

// Conformance: mọi dòng của bảng quyền phải khai đủ, cùng luật với bảng năng lực
// của provider. Một bảng quyền có ô trống là một lời hứa không ai đứng sau.
func TestBangQuyenKhaiDuBangChung(t *testing.T) {
	thay := map[string]bool{}
	for _, q := range MoiQuyen {
		if q.Khoa == "" || q.Mo == "" {
			t.Errorf("dòng quyền thiếu khoá hoặc mô tả: %+v", q)
		}
		if thay[q.Khoa] {
			t.Errorf("quyền %q khai hai lần trong MoiQuyen", q.Khoa)
		}
		thay[q.Khoa] = true
		switch q.Chan {
		case ChanThat, KhongChanDuoc, ChuaDo:
		default:
			t.Errorf("quyền %q có trạng thái chặn %q không phải một trong ba", q.Khoa, q.Chan)
		}
		if strings.TrimSpace(q.BangChung) == "" {
			t.Errorf("quyền %q khai %q mà KHÔNG có bằng chứng — chặn ở đâu, "+
				"hoặc vì sao chưa chặn được", q.Khoa, q.Chan)
		}
	}
	if MoTaCuaQuyen("quyen-khong-ton-tai") != nil {
		t.Error("quyền không có thật mà tra ra được")
	}
}

// Quyền khai là ChanThat thì bằng chứng phải TRỎ VÀO MỘT BÀI TEST có thật.
//
// Không phải bắt bẻ câu chữ: dòng "host chặn thật" là thứ người vận hành đọc rồi
// yên tâm. Nếu bài test được viện dẫn không tồn tại (đổi tên, xoá đi), lời khai
// vẫn nằm đó nói y như cũ — và không có gì bắt được chuyện đó ngoài bài test này.
func TestQuyenChanThatPhaiTroVaoTestCoThat(t *testing.T) {
	coTest := map[string]bool{
		"TestKhongKhaiThuMucThiKhongThayThuMucDuAn":    true,
		"TestKhongKhaiMoiTruongThiKhongThayBienCuaCha": true,
		"TestKhongKhaiSecretThiKhongNhanDuocGiaTri":    true,
		"TestKhaiSecretMaKhongKhaiQuyenThiTuChoi":      true,
	}
	for _, q := range MoiQuyen {
		if q.Chan != ChanThat {
			continue
		}
		thay := false
		for ten := range coTest {
			if strings.Contains(q.BangChung, ten) {
				thay = true
				break
			}
		}
		if !thay {
			t.Errorf("quyền %q khai CHẶN THẬT nhưng bằng chứng không trỏ vào bài test nào "+
				"trong danh sách đã biết — hoặc đổi bằng chứng, hoặc thêm test vào danh sách "+
				"của bài này: %s", q.Khoa, q.BangChung)
		}
	}
}
