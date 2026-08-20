// Canh sổ mã nguồn mở (`docs/OPEN_SOURCE_LEDGER.md`) so với `go.mod`.
//
// Vì sao cần: cam kết ở MASTER-PLAN mục 0 — mọi phụ thuộc phải là mã nguồn mở,
// giấy phép tương thích MIT, và ghi lại ở sổ. Trước file này, cam kết đó KHÔNG
// có gì canh, và sổ viết tay đã trôi thật hai lần (được chính header của sổ
// thừa nhận): kê `github.com/google/uuid` trong khi nó không hề được liên kết
// vào binary, và vẫn xếp `golang.org/x/sys` ở bảng gián tiếp sau khi nó đã
// thành phụ thuộc trực tiếp. Đo lúc viết: go.mod có 11 module require, binary
// thật chỉ liên kết 10 — đúng 1 khoảng hở mà một sổ viết tay hay ngã vào.
package main

import (
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	duongSo    = "../../docs/OPEN_SOURCE_LEDGER.md"
	duongGoMod = "../../go.mod"
	thuMucGoc  = "../.."
)

// Giấy phép tương thích với MIT của dự án: chỉ đòi giữ thông báo bản quyền, KHÔNG
// lây điều kiện sang mã của mình. Copyleft (GPL/LGPL/AGPL) và cả MPL-2.0 cố ý
// không có ở đây — ghi một cái như vậy vào sổ là test đỏ ngay, vì nó phá cam kết
// phát hành theo MIT ở MASTER-PLAN mục 0 chứ không chỉ là chuyện thẩm mỹ.
var giayPhepTuongThich = map[string]bool{
	"MIT":          true,
	"BSD-2-Clause": true,
	"BSD-3-Clause": true,
	"ISC":          true,
	"Apache-2.0":   true,
	"Unlicense":    true,
	"CC0-1.0":      true,
}

// dongSo là một hàng phụ thuộc đọc được từ bảng trong sổ.
type dongSo struct {
	goi      []string // một hàng có thể gộp nhiều module (bảng gián tiếp làm vậy)
	phienBan string   // rỗng nếu bảng không có cột phiên bản
	giayPhep string
	muc      string // tiêu đề "## ..." đang mở — để phân biệt trực tiếp / gián tiếp
	dong     int
}

var reNhayNguoc = regexp.MustCompile("`([^`]+)`")

// docSo đọc sổ và cắt các bảng phụ thuộc ra.
//
// Cột được xác định bằng HÀNG TIÊU ĐỀ chứ không đếm cứng vị trí: bảng trực tiếp
// có 5 cột, bảng gián tiếp có 2, và nếu ai đó thêm cột thì test này vẫn phải
// đúng thay vì đọc nhầm ô.
func docSo(t *testing.T) (noiDung string, hang []dongSo) {
	t.Helper()
	b, err := os.ReadFile(duongSo)
	if err != nil {
		t.Fatalf("không đọc được sổ %s: %v", duongSo, err)
	}
	noiDung = strings.ReplaceAll(string(b), "\r\n", "\n")

	muc := ""
	iGoi, iPB, iGP := -1, -1, -1
	for n, dong := range strings.Split(noiDung, "\n") {
		s := strings.TrimSpace(dong)
		if strings.HasPrefix(s, "#") {
			muc = strings.TrimLeft(s, "# ")
		}
		if !strings.HasPrefix(s, "|") {
			iGoi, iPB, iGP = -1, -1, -1 // hết bảng
			continue
		}
		o := cheoO(s)
		if laVachNgan(o) {
			continue
		}
		if j := chiSoO(o, "Gói"); j >= 0 {
			iGoi, iPB, iGP = j, chiSoO(o, "Phiên bản"), chiSoO(o, "Giấy phép")
			continue
		}
		if iGoi < 0 || iGP < 0 || iGP >= len(o) || iGoi >= len(o) {
			continue // bảng khác, không phải bảng phụ thuộc
		}
		var goi []string
		for _, m := range reNhayNguoc.FindAllStringSubmatch(o[iGoi], -1) {
			if strings.Contains(m[1], "/") { // đường module luôn có dấu gạch chéo
				goi = append(goi, m[1])
			}
		}
		if len(goi) == 0 {
			continue
		}
		h := dongSo{goi: goi, giayPhep: o[iGP], muc: muc, dong: n + 1}
		if iPB >= 0 && iPB < len(o) {
			h.phienBan = strings.Trim(o[iPB], "`")
		}
		hang = append(hang, h)
	}
	if len(hang) == 0 {
		t.Fatalf("%s: không cắt được hàng phụ thuộc nào — sổ đổi định dạng thì sửa test này, đừng bỏ nó", duongSo)
	}
	return noiDung, hang
}

func cheoO(dong string) []string {
	p := strings.Split(strings.Trim(dong, "|"), "|")
	for i := range p {
		p[i] = strings.TrimSpace(p[i])
	}
	return p
}

func laVachNgan(o []string) bool {
	for _, c := range o {
		if strings.Trim(c, "-: ") != "" {
			return false
		}
	}
	return true
}

func chiSoO(o []string, ten string) int {
	for i, c := range o {
		if strings.Contains(c, ten) {
			return i
		}
	}
	return -1
}

// ycau là một dòng require trong go.mod.
type ycau struct {
	duong    string
	phienBan string
	gianTiep bool
}

func docGoMod(t *testing.T) []ycau {
	t.Helper()
	b, err := os.ReadFile(duongGoMod)
	if err != nil {
		t.Fatalf("không đọc được %s: %v", duongGoMod, err)
	}
	var ds []ycau
	trongKhoi := false
	for _, dong := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		s := strings.TrimSpace(dong)
		switch {
		case s == "" || strings.HasPrefix(s, "//"):
			continue
		case strings.HasPrefix(s, "require ("):
			trongKhoi = true
			continue
		case trongKhoi && s == ")":
			trongKhoi = false
			continue
		case strings.HasPrefix(s, "require "):
			s = strings.TrimPrefix(s, "require ")
		case !trongKhoi:
			continue
		}
		gianTiep := strings.Contains(s, "// indirect")
		if i := strings.Index(s, "//"); i >= 0 {
			s = s[:i]
		}
		f := strings.Fields(s)
		if len(f) < 2 || !strings.Contains(f[0], "/") {
			continue
		}
		ds = append(ds, ycau{duong: f[0], phienBan: f[1], gianTiep: gianTiep})
	}
	if len(ds) == 0 {
		t.Fatalf("%s: không đọc được require nào", duongGoMod)
	}
	return ds
}

// tapGoiTrongBang trả về mọi module được KÊ TÊN trong bảng của sổ.
func tapGoiTrongBang(hang []dongSo) map[string]dongSo {
	m := map[string]dongSo{}
	for _, h := range hang {
		for _, g := range h.goi {
			m[g] = h
		}
	}
	return m
}

// coGiaiTrinh: module không nằm trong bảng vẫn được chấp nhận NẾU sổ nói rõ vì
// sao nó không được kê — và lý do hợp lệ duy nhất tới giờ là "không được liên
// kết vào binary" (trường hợp google/uuid). Chỉ nhắc tên suông thì không tính:
// dòng phải nói về chuyện liên kết.
func coGiaiTrinh(noiDung, duong string) bool {
	for _, dong := range strings.Split(noiDung, "\n") {
		if strings.Contains(dong, duong) && strings.Contains(dong, "liên kết") {
			return true
		}
	}
	return false
}

// Thiếu một phụ thuộc trong sổ là VI PHẠM cam kết đã ghi ở MASTER-PLAN mục 0:
// "mọi phụ thuộc phải ... ghi lại ở đây". Đây là test chính của gói này.
func TestSoKeDuMoiPhuThuocTrongGoMod(t *testing.T) {
	noiDung, hang := docSo(t)
	trongBang := tapGoiTrongBang(hang)

	var thieu []string
	for _, y := range docGoMod(t) {
		if _, ok := trongBang[y.duong]; ok {
			continue
		}
		if coGiaiTrinh(noiDung, y.duong) {
			continue // sổ nói rõ vì sao không kê — TestModuleDuocMienPhaiThatSuKhongLienKet kiểm lời đó
		}
		thieu = append(thieu, y.duong)
	}
	if len(thieu) > 0 {
		sort.Strings(thieu)
		t.Errorf("%s thiếu %d phụ thuộc có trong go.mod: %s\n"+
			"  → vi phạm cam kết MASTER-PLAN mục 0 (mọi phụ thuộc phải ghi lại ở sổ).\n"+
			"  → thêm hàng vào bảng kèm giấy phép, hoặc ghi rõ vì sao nó không được liên kết vào binary",
			duongSo, len(thieu), strings.Join(thieu, ", "))
	}
}

// Chiều ngược lại: sổ kê module đã biến mất khỏi go.mod. Cũng là trôi, và nguy
// hơn vì nó tạo cảm giác đã kiểm — đúng lỗi google/uuid mà header của sổ kể lại.
func TestSoKhongKeModuleKhongConTrongGoMod(t *testing.T) {
	_, hang := docSo(t)
	trongGoMod := map[string]bool{}
	for _, y := range docGoMod(t) {
		trongGoMod[y.duong] = true
	}
	for g, h := range tapGoiTrongBang(hang) {
		if !trongGoMod[g] {
			t.Errorf("%s:%d kê %q nhưng go.mod không còn require nó — sổ đang tả một dự án khác", duongSo, h.dong, g)
		}
	}
}

// Giấy phép nào cũng phải nằm trong danh sách tương thích MIT.
func TestMoiGiayPhepTrongSoDeuTuongThichMIT(t *testing.T) {
	_, hang := docSo(t)
	for _, h := range hang {
		if h.giayPhep == "" {
			t.Errorf("%s:%d — %s không ghi giấy phép", duongSo, h.dong, strings.Join(h.goi, ", "))
			continue
		}
		for _, gp := range tachGiayPhep(h.giayPhep) {
			if !giayPhepTuongThich[gp] {
				t.Errorf("%s:%d — %s ghi giấy phép %q, không nằm trong danh sách tương thích MIT %v\n"+
					"  → hoặc gỡ phụ thuộc, hoặc sửa danh sách kèm lý do vì sao nó vẫn tương thích",
					duongSo, h.dong, strings.Join(h.goi, ", "), gp, tenTuongThich())
			}
		}
	}
}

// Một ô có thể ghi nhiều giấy phép ("MIT OR Apache-2.0", "MIT / BSD-3-Clause").
// Tách ra và bắt từng cái — chỉ cần một cái không tương thích là đủ hỏng.
func tachGiayPhep(o string) []string {
	o = strings.Trim(o, "`* ")
	for _, sep := range []string{" OR ", " AND ", " hoặc ", "/", ","} {
		o = strings.ReplaceAll(o, sep, "|")
	}
	var ds []string
	for _, p := range strings.Split(o, "|") {
		if p = strings.TrimSpace(p); p != "" {
			ds = append(ds, p)
		}
	}
	return ds
}

func tenTuongThich() []string {
	var ds []string
	for k := range giayPhepTuongThich {
		ds = append(ds, k)
	}
	sort.Strings(ds)
	return ds
}

// Phiên bản trong sổ phải khớp go.mod. Nâng một dependency mà quên sửa sổ thì
// phần "vì sao không dùng stdlib" đang nói về một bản khác với bản đang build.
func TestPhienBanTrongSoKhopGoMod(t *testing.T) {
	_, hang := docSo(t)
	pb := map[string]string{}
	for _, y := range docGoMod(t) {
		pb[y.duong] = y.phienBan
	}
	for _, h := range hang {
		if h.phienBan == "" {
			continue // bảng gián tiếp không có cột phiên bản — không có gì để so
		}
		for _, g := range h.goi {
			if that, ok := pb[g]; ok && that != h.phienBan {
				t.Errorf("%s:%d — %s ghi %s nhưng go.mod đang dùng %s", duongSo, h.dong, g, h.phienBan, that)
			}
		}
	}
}

// Xếp nhầm bảng cũng là trôi: sổ tự thú `golang.org/x/sys` từng nằm ở bảng gián
// tiếp sau khi đã thành trực tiếp. "Trực tiếp" ở đây có định nghĩa máy kiểm
// được: dòng require KHÔNG có dấu `// indirect`.
func TestBangTrucTiepVaGianTiepKhopGoMod(t *testing.T) {
	_, hang := docSo(t)
	gianTiep := map[string]bool{}
	for _, y := range docGoMod(t) {
		gianTiep[y.duong] = y.gianTiep
	}
	for _, h := range hang {
		trongMucGianTiep := strings.Contains(strings.ToLower(h.muc), "gián tiếp")
		for _, g := range h.goi {
			gt, ok := gianTiep[g]
			if !ok {
				continue // TestSoKhongKeModuleKhongConTrongGoMod đã lo ca này
			}
			if gt != trongMucGianTiep {
				t.Errorf("%s:%d — %s nằm ở mục %q nhưng go.mod đánh dấu indirect=%v", duongSo, h.dong, g, h.muc, gt)
			}
		}
	}
}

// lietKeThat hỏi toolchain những module THẬT SỰ đi vào binary. Bỏ qua khi không
// gọi được `go list` (máy không có toolchain, hoặc kho module chưa tải) — test
// còn lại vẫn chạy thuần trên văn bản.
func lietKeThat(t *testing.T) map[string]bool {
	t.Helper()
	t.Chdir(thuMucGoc) // lietKe gọi `go list ./cmd/sagent`, đường tương đối tính từ gốc repo
	mods, err := lietKe()
	if err != nil {
		t.Skipf("bỏ qua: không chạy được go list (%v)", err)
	}
	m := map[string]bool{}
	for _, x := range mods {
		m[x.Path] = true
	}
	return m
}

// Module được miễn khỏi bảng phải THẬT SỰ không được liên kết. Nếu không, lời
// giải trình trong sổ là một câu sai được viết ra để né việc kê.
func TestModuleDuocMienPhaiThatSuKhongLienKet(t *testing.T) {
	noiDung, hang := docSo(t)
	trongBang := tapGoiTrongBang(hang)
	var mien []string
	for _, y := range docGoMod(t) {
		if _, ok := trongBang[y.duong]; !ok && coGiaiTrinh(noiDung, y.duong) {
			mien = append(mien, y.duong)
		}
	}

	lienKet := lietKeThat(t)
	for _, g := range mien {
		if lienKet[g] {
			t.Errorf("sổ viết %q không được liên kết vào binary, nhưng `go list -deps ./cmd/sagent` có nó — phải kê vào bảng kèm giấy phép", g)
		}
	}
}

// Chiều còn lại và là chiều nặng nhất: mọi module ĐANG ĐI VÀO BINARY đều phải có
// hàng trong bảng kèm giấy phép. Đây là thứ mà MIT/BSD thật sự đòi khi phát hành.
func TestMoiModuleDuocLienKetDeuCoTrongBangSo(t *testing.T) {
	_, hang := docSo(t)
	trongBang := tapGoiTrongBang(hang)
	for g := range lietKeThat(t) {
		if _, ok := trongBang[g]; !ok {
			t.Errorf("%s không kê %q — module này ĐANG được liên kết vào binary, giấy phép của nó phải nằm trong sổ", duongSo, g)
		}
	}
}

// Sổ trỏ người đọc sang THONG-BAO-GIAY-PHEP.txt cho phần "cái gì". Nếu file đó
// trôi thì lời hứa "CI chạy -kiem nên file đó không trôi được" thành lời suông.
func TestThongBaoGiayPhepChuaTroi(t *testing.T) {
	t.Chdir(thuMucGoc)
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("bỏ qua: không có toolchain go (%v)", err)
	}
	mods, err := lietKe()
	if err != nil {
		t.Skipf("bỏ qua: không chạy được go list (%v)", err)
	}
	muon, thieu := dungVanBan(mods)
	if len(thieu) > 0 {
		t.Fatalf("không tìm thấy file giấy phép cho: %s", strings.Join(thieu, ", "))
	}
	dang, err := os.ReadFile(dichDen)
	if err != nil {
		t.Fatalf("chưa có %s — chạy: go run ./tools/giayphep", dichDen)
	}
	if strings.ReplaceAll(string(dang), "\r\n", "\n") != muon {
		t.Errorf("%s đã lệch với phụ thuộc thật — chạy lại: go run ./tools/giayphep", dichDen)
	}
}
