package provider

import (
	"strings"
	"testing"
	"time"
)

// BỘ CONFORMANCE — chạy chung cho MỌI adapter đã đăng ký.
//
// Vì sao cần: bốn mặt điều khiển sắp đọc bảng năng lực để quyết định hiện gì và
// từ chối gì. Một bảng năng lực SAI thì tệ hơn không có bảng — người vận hành
// tin nó. Trước đây "năng lực" được mã hoá bằng "trả nil hay không" và giải
// thích bằng bình luận; bình luận thì không có gì bắt nó đúng.
//
// Bộ này bắt ba kiểu hỏng, và cả ba đều đã có chỗ đứng thật trong repo này:
//
//	1. Khai "làm được" mà trả nil — lời hứa không có gì đứng sau. Lõi thấy nil
//	   sẽ im lặng chạy model mặc định trong khi bảng bảo là chọn được.
//	2. Khai "chưa đo" mà vẫn trả giá trị thật — ngược lại: lõi DÙNG giá trị đó
//	   còn bảng thì bảo chưa biết gì, nên không ai đi kiểm nó.
//	3. Adapter thiếu cài đặt — thêm provider mới mà quên khai một năng lực.
//	   (Quên khai CẢ hàm NangLuc thì không biên dịch được, đó là lá chắn thứ nhất;
//	   đây là lá chắn thứ hai cho trường hợp khai thiếu dòng.)

// Đây là test chính: mọi adapter thật phải khớp lời khai với hành vi.
func TestMoiAdapterKhaiNangLucDungVoiHanhViThat(t *testing.T) {
	if len(Names()) == 0 {
		t.Fatal("không có adapter nào đăng ký — bộ conformance không đo được gì")
	}
	for _, name := range Names() {
		ad, _ := Get(name)
		for _, l := range KiemNangLuc(ad) {
			t.Errorf("%s: %s", name, l)
		}
	}
}

// Mọi năng lực trong MoiNangLuc phải được MỌI adapter khai, không sót dòng nào.
//
// Tách khỏi test trên để câu báo lỗi chỉ thẳng vào việc phải làm: thêm một khoá
// vào MoiNangLuc là phải đi khai ở cả năm adapter.
func TestKhongAdapterNaoBoSotNangLuc(t *testing.T) {
	for _, name := range Names() {
		ad, _ := Get(name)
		co := map[string]bool{}
		for _, nl := range ad.NangLuc() {
			co[nl.Khoa] = true
		}
		for _, m := range MoiNangLuc {
			if !co[m.Khoa] {
				t.Errorf("%s: không khai %q (%s)", name, m.Khoa, m.Mo)
			}
		}
	}
}

// Mỗi dòng khai phải có BẰNG CHỨNG. "Làm được" thì bằng chứng là phép đo, "chưa
// đo" thì bằng chứng là lý do chưa đo được — cả hai đều đáng viết ra.
//
// Không có ràng buộc này thì bảng năng lực trượt dần thành một danh sách yes/no,
// đúng thứ mà docs/DO-LUONG.md sinh ra để chống.
func TestMoiDongKhaiDeuCoBangChung(t *testing.T) {
	for _, name := range Names() {
		ad, _ := Get(name)
		for _, nl := range ad.NangLuc() {
			if strings.TrimSpace(nl.BangChung) == "" {
				t.Errorf("%s: %q khai %q mà không nói đo ở đâu", name, nl.Khoa, nl.TrangThai)
			}
		}
	}
}

// Ba trạng thái phải THẬT SỰ dùng cả ba, nếu không thì kiểu ba trạng thái chỉ là
// kiểu hai trạng thái mặc áo mới.
//
// Repo hiện có đủ cả ba ví dụ thật: Grok KHÔNG có rào duyệt quyền (đã đo),
// Antigravity KHÔNG tách được tài khoản (đã đo), Cursor thì CHƯA ĐO gì nhiều vì
// máy dev không cài. Test này giữ cho việc phân biệt "không có" với "chưa đo"
// không bị ai gộp lại cho gọn.
func TestDungDuCaBaTrangThai(t *testing.T) {
	dem := map[TrangThaiNangLuc]int{}
	for _, name := range Names() {
		ad, _ := Get(name)
		for _, nl := range ad.NangLuc() {
			dem[nl.TrangThai]++
		}
	}
	for _, tt := range []TrangThaiNangLuc{LamDuoc, KhongLamDuoc, ChuaDo} {
		if dem[tt] == 0 {
			t.Errorf("không adapter nào khai %q — nếu \"không làm được\" và \"chưa đo\" đang bị "+
				"gộp làm một thì kiểu ba trạng thái mất hết ý nghĩa", tt)
		}
	}
}

// ---------------------------- adapter giả để đo chính bộ đo ----------------------------
//
// Ba test dưới đây kiểm CHÍNH KiemNangLuc: gỡ một nhánh trong nó ra thì một
// trong ba test này phải đỏ. Không có chúng thì bộ conformance có thể im lặng
// hỏng mà mọi adapter thật vẫn xanh — một lá chắn không ai kiểm là một lá chắn
// đã mục.

// adapterGia là một adapter tối giản, hành vi CỐ ĐỊNH và biết trước: cài đặt
// đúng ba thứ (headless, chọn model, khai thư mục), có rào duyệt quyền, tách
// được tài khoản. Bảng khai của nó thì lấy từ ngoài vào để test bẻ cong.
type adapterGia struct{ khai []NangLuc }

func (adapterGia) Name() string                             { return "gia" }
func (adapterGia) EnvVar() string                           { return "GIA_CONFIG_DIR" }
func (adapterGia) Command() (string, error)                 { return "gia", nil }
func (adapterGia) HeadlessArgs(p string) []string           { return []string{"-p", p} }
func (adapterGia) ModelArgs(m string) []string              { return []string{"--model", m} }
func (adapterGia) ArgsTuDuyetQuyen() ([]string, bool)       { return []string{"--yolo"}, true }
func (adapterGia) ArgsThuMuc(d string) []string             { return []string{"--dir", d} }
func (adapterGia) ArgsHoSo(string) []string                 { return nil }
func (adapterGia) DocKetQua(string) (KetQua, bool)          { return KetQua{}, false }
func (adapterGia) PrivateFiles() []string                   { return []string{"token.json"} }
func (adapterGia) SharedKeys() []string                     { return nil }
func (adapterGia) BaseDir() string                          { return "gia" }
func (adapterGia) IdentitySource() string                   { return "" }
func (adapterGia) Identity(string) string                   { return "" }
func (adapterGia) HasToken(string) bool                     { return false }
func (adapterGia) TokenExpiry(string) (time.Time, bool)     { return time.Time{}, false }
func (adapterGia) TachDuocTaiKhoan() bool                   { return true }
func (adapterGia) Version() (string, error)                 { return "gia 0.0.1", nil }
func (adapterGia) Verify() []Check                          { return nil }
func (a adapterGia) NangLuc() []NangLuc                     { return a.khai }

// khaiDung là bảng khai KHỚP với hành vi của adapterGia — điểm xuất phát để mỗi
// test chỉ bẻ cong đúng một dòng.
func khaiDung() []NangLuc {
	return []NangLuc{
		Duoc(NLHeadless, "đo giả"),
		Duoc(NLChonModel, "đo giả"),
		Duoc(NLTuDuyetQuyen, "đo giả"),
		Duoc(NLThuMuc, "đo giả"),
		Chua(NLCoTuHoSo, "đo giả"),
		Chua(NLKetQuaCoCauTruc, "đo giả"),
		Duoc(NLTachTaiKhoan, "đo giả"),
		Chua(NLHanToken, "đo giả"),
		Chua(NLDanhTinh, "đo giả"),
	}
}

// doi thay trạng thái của một khoá trong bảng khai.
func doi(khai []NangLuc, khoa string, tt TrangThaiNangLuc) []NangLuc {
	for i := range khai {
		if khai[i].Khoa == khoa {
			khai[i].TrangThai = tt
		}
	}
	return khai
}

// bo gỡ hẳn một khoá khỏi bảng khai.
func bo(khai []NangLuc, khoa string) []NangLuc {
	var out []NangLuc
	for _, nl := range khai {
		if nl.Khoa != khoa {
			out = append(out, nl)
		}
	}
	return out
}

func phaiBat(t *testing.T, ad Adapter, chua string) {
	t.Helper()
	lech := KiemNangLuc(ad)
	for _, l := range lech {
		if strings.Contains(l, chua) {
			return
		}
	}
	t.Fatalf("bộ conformance KHÔNG bắt được lỗi này. Chỗ lệch tìm ra: %v", lech)
}

// Kiểu hỏng 1: khai "làm được" mà hàm trả rỗng.
func TestBatKhaiLamDuocMaTraNil(t *testing.T) {
	// adapterGia.ArgsHoSo trả nil, nhưng bảng khai bảo làm được.
	ad := adapterGia{doi(khaiDung(), NLCoTuHoSo, LamDuoc)}
	// co-tu-ho-so là phép dò MỘT CHIỀU (hồ sơ rỗng thì Grok cũng trả nil), nên
	// nó cố ý KHÔNG bị bắt — đó là giới hạn đã biết, không phải lỗ hổng lặng lẽ.
	if len(KiemNangLuc(ad)) != 0 {
		t.Fatalf("phép dò một chiều không được kết luận từ chiều trả rỗng: %v", KiemNangLuc(ad))
	}
	// Còn chọn model thì dò được HAI CHIỀU, nên nói dối ở đây là bị bắt.
	ad2 := adapterKhongCoModel{adapterGia{khaiDung()}}
	phaiBat(t, ad2, "TRẢ RỖNG")
}

// adapterKhongCoModel: y hệt adapterGia nhưng ModelArgs trả nil, trong khi bảng
// khai vẫn nói "làm được".
type adapterKhongCoModel struct{ adapterGia }

func (adapterKhongCoModel) ModelArgs(string) []string { return nil }

// Kiểu hỏng 2: khai "chưa đo" mà hàm vẫn trả giá trị thật.
func TestBatKhaiChuaDoMaVanTraGiaTriThat(t *testing.T) {
	ad := adapterGia{doi(khaiDung(), NLThuMuc, ChuaDo)}
	phaiBat(t, ad, "vẫn TRẢ VỀ giá trị thật")

	// Cũng phải bắt khi khai "không làm được" — hai lời khai khác nhau, cùng một
	// hậu quả: lõi dùng giá trị mà bảng bảo là không có.
	ad2 := adapterGia{doi(khaiDung(), NLThuMuc, KhongLamDuoc)}
	phaiBat(t, ad2, "vẫn TRẢ VỀ giá trị thật")
}

// Kiểu hỏng 3: adapter thiếu cài đặt — khai sót một năng lực.
func TestBatAdapterKhaiSotNangLuc(t *testing.T) {
	ad := adapterGia{bo(khaiDung(), NLHanToken)}
	phaiBat(t, ad, "KHÔNG KHAI")
}

// Ba trạng thái của ArgsTuDuyetQuyen phải ánh xạ ĐÚNG, không được gộp.
//
// Đây là chỗ nguy hiểm nhất: (nil, true) và (nil, false) trông y hệt nhau nếu
// chỉ nhìn phần args. Gộp chúng lại thì Grok — provider KHÔNG có rào nào — bị
// xếp chung với Cursor chưa đo, và câu chuyện an ninh biến mất.
func TestPhanBietKhongCoRaoVoiChuaDo(t *testing.T) {
	khongCoRao := adapterKhongRao{adapterGia{doi(khaiDung(), NLTuDuyetQuyen, KhongLamDuoc)}}
	if l := KiemNangLuc(khongCoRao); len(l) != 0 {
		t.Fatalf("(nil, true) phải đọc là KHÔNG LÀM ĐƯỢC: %v", l)
	}
	phaiBat(t, adapterKhongRao{adapterGia{doi(khaiDung(), NLTuDuyetQuyen, ChuaDo)}},
		"hành xử như")

	chuaDo := adapterChuaDoRao{adapterGia{doi(khaiDung(), NLTuDuyetQuyen, ChuaDo)}}
	if l := KiemNangLuc(chuaDo); len(l) != 0 {
		t.Fatalf("(nil, false) phải đọc là CHƯA ĐO: %v", l)
	}
	phaiBat(t, adapterChuaDoRao{adapterGia{doi(khaiDung(), NLTuDuyetQuyen, KhongLamDuoc)}},
		"hành xử như")
}

type adapterKhongRao struct{ adapterGia }

func (adapterKhongRao) ArgsTuDuyetQuyen() ([]string, bool) { return nil, true }

type adapterChuaDoRao struct{ adapterGia }

func (adapterChuaDoRao) ArgsTuDuyetQuyen() ([]string, bool) { return nil, false }

// Khai một khoá không có trong MoiNangLuc thì cũng phải bị bắt: mặt nào đọc
// bảng cũng sẽ bỏ qua nó, nên nó là công sức đổ xuống sông.
func TestBatKhaiKhoaLa(t *testing.T) {
	ad := adapterGia{append(khaiDung(), Duoc("bay-len-troi", "đo giả"))}
	phaiBat(t, ad, "năng lực lạ")
}
