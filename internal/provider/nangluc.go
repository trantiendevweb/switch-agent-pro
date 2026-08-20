package provider

import "fmt"

// TrangThaiNangLuc là BA trạng thái của một năng lực — và ba, không phải hai,
// mới là điểm chính của file này.
//
// Trước đây mỗi năng lực được mã hoá bằng "trả nil hay không", và cả dự án phải
// đọc bình luận mới biết nil nghĩa là gì. Xem chính interface Adapter: ghi chú
// của ArgsTuDuyetQuyen phải viết hẳn ba đoạn để phân biệt (nil, true) với
// (nil, false), còn ModelArgs thì phải dặn "nil = CHƯA BIẾT, không phải không
// có model". Lời dặn nằm trong bình luận thì không có gì bắt nó đúng.
//
// Hai trạng thái là chỗ hỏng: gộp "đã đo và provider KHÔNG có thứ đó" với "chưa
// ai đo bao giờ" khiến cả hai đều im lặng như nhau, mà hai cái đòi hai cách xử
// lý ngược nhau — cái đầu chạy tiếp được, cái sau phải báo cho người dùng biết
// là lựa chọn của họ vừa bị nuốt.
type TrangThaiNangLuc string

const (
	// LamDuoc: ĐÃ ĐO trên máy thật, và đây là cách làm.
	LamDuoc TrangThaiNangLuc = "lam-duoc"
	// KhongLamDuoc: ĐÃ ĐO, và provider này KHÔNG có thứ đó. Đây là một kết luận,
	// không phải một khoảng trống — Grok không có rào duyệt quyền nào là chuyện
	// an ninh phải nói ra, không phải chuyện "không cần làm gì".
	KhongLamDuoc TrangThaiNangLuc = "khong-lam-duoc"
	// ChuaDo: CHƯA AI ĐO. Người gọi không được lặng lẽ chạy tiếp.
	ChuaDo TrangThaiNangLuc = "chua-do"
)

// NangLuc là MỘT dòng khai báo năng lực của một adapter.
//
// BangChung không phải trang trí và không được để trống: mọi con số trong dự án
// này gắn với một phép đo (docs/DO-LUONG.md), nên một dòng khai "làm được" mà
// không nói đo ở đâu thì chỉ là một lời hứa. Với ChuaDo thì bằng chứng là lý do
// CHƯA đo được — cũng đáng viết ra y như vậy.
type NangLuc struct {
	Khoa      string           `json:"khoa"`
	TrangThai TrangThaiNangLuc `json:"trang_thai"`
	BangChung string           `json:"bang_chung"`
}

// Ba hàm dựng để bảng khai báo trong từng adapter đọc được như một cái bảng,
// chứ không phải một đống struct literal.
func Duoc(khoa, bangChung string) NangLuc {
	return NangLuc{khoa, LamDuoc, bangChung}
}
func Khong(khoa, bangChung string) NangLuc {
	return NangLuc{khoa, KhongLamDuoc, bangChung}
}
func Chua(khoa, bangChung string) NangLuc {
	return NangLuc{khoa, ChuaDo, bangChung}
}

// Khoá của từng năng lực. Đây là TỪ VỰNG CHUNG của cả bốn mặt điều khiển: CLI,
// API, dashboard và 3D đều gọi tên một năng lực bằng đúng chuỗi này.
const (
	NLHeadless        = "chay-headless"
	NLChonModel       = "chon-model"
	NLTuDuyetQuyen    = "tu-duyet-quyen"
	NLThuMuc          = "khai-thu-muc"
	NLCoTuHoSo        = "co-tu-ho-so"
	NLKetQuaCoCauTruc = "ket-qua-co-cau-truc"
	NLTachTaiKhoan    = "tach-nhieu-tai-khoan"
	NLHanToken        = "han-token"
	NLDanhTinh        = "danh-tinh"
)

// MoiNangLuc là danh sách CHÍNH THỨC mọi năng lực hệ thống hỏi tới, kèm câu mô
// tả để mặt nào hiện ra cũng dùng chung một cách gọi.
//
// Cùng vai trò với api.Actions: thêm một năng lực vào đây mà adapter nào đó
// quên khai thì conformance test đỏ, thay vì lõi lặng lẽ coi như "chưa đo".
var MoiNangLuc = []struct{ Khoa, Mo string }{
	{NLHeadless, "chạy không tương tác với một prompt"},
	{NLChonModel, "chọn model từ dòng lệnh"},
	{NLTuDuyetQuyen, "tự duyệt mọi tool ở chế độ headless"},
	{NLThuMuc, "khai tường minh thư mục làm việc"},
	{NLCoTuHoSo, "suy cờ từ chính thư mục hồ sơ đang chạy"},
	{NLKetQuaCoCauTruc, "đọc kết quả có cấu trúc từ bản ghi"},
	{NLTachTaiKhoan, "chạy nhiều tài khoản trên cùng một máy"},
	{NLHanToken, "đọc được token còn hạn tới bao giờ"},
	{NLDanhTinh, "đọc được danh tính để hiển thị"},
}

// moTaNangLuc tra câu mô tả theo khoá.
func moTaNangLuc(khoa string) string {
	for _, m := range MoiNangLuc {
		if m.Khoa == khoa {
			return m.Mo
		}
	}
	return ""
}

// ---------------------------- đối chiếu ----------------------------

// thuMucKhongTonTai là đầu vào thử cho những phép dò cần một thư mục hồ sơ.
// Cố ý là một đường dẫn tương đối không thể tồn tại: phép dò phải cho ra cùng
// một kết quả trên mọi máy, không được phụ thuộc máy đang chạy có gì.
const thuMucKhongTonTai = "sagent-thu-muc-khong-bao-gio-ton-tai-9f3a"

// phepDoBaTrangThai là những năng lực mà CHÍNH adapter đã trả về đủ ba trạng
// thái — dò được chính xác, nên khai lệch một li là bắt được.
var phepDoBaTrangThai = map[string]func(Adapter) TrangThaiNangLuc{
	NLTuDuyetQuyen: func(ad Adapter) TrangThaiNangLuc {
		args, daDo := ad.ArgsTuDuyetQuyen()
		if !daDo {
			return ChuaDo
		}
		if len(args) == 0 {
			return KhongLamDuoc
		}
		return LamDuoc
	},
	// TachDuocTaiKhoan CỐ Ý không có nấc "chưa đo": lõi ACT theo giá trị của nó
	// (fleet --copies N từ chối chạy khi false). Một cái bool mà lõi đã hành
	// động theo thì không được phép khai là chưa biết.
	NLTachTaiKhoan: func(ad Adapter) TrangThaiNangLuc {
		if ad.TachDuocTaiKhoan() {
			return LamDuoc
		}
		return KhongLamDuoc
	},
}

// phepDoGiaTri là những năng lực chỉ dò được "có trả giá trị thật hay không".
//
// haiChieu nói phép dò kết luận được cả hai chiều hay chỉ một:
//
//	true  — đầu vào thử là ĐỦ, nên trả rỗng chắc chắn nghĩa là không cài đặt.
//	false — trả rỗng còn có thể vì đầu vào thử không đủ (thư mục hồ sơ rỗng,
//	        bản ghi không phải của provider đó). Chiều "trả rỗng" KHÔNG kết
//	        luận được, nhưng chiều "trả giá trị thật" thì vẫn kết luận được —
//	        và đó chính là chiều bắt được lỗi khai "chưa đo" mà vẫn trả giá trị.
type phepDoGiaTri struct {
	co       func(Adapter) bool
	haiChieu bool
}

var phepDoGiaTriNangLuc = map[string]phepDoGiaTri{
	NLHeadless: {func(ad Adapter) bool { return len(ad.HeadlessArgs("XIN-CHAO")) > 0 }, true},
	NLChonModel: {func(ad Adapter) bool {
		return len(ad.ModelArgs("MO-HINH-THU")) > 0
	}, true},
	NLThuMuc: {func(ad Adapter) bool {
		return len(ad.ArgsThuMuc(thuMucKhongTonTai)) > 0
	}, true},
	// Một chiều: Grok đọc defaultModel TỪ FILE trong hồ sơ, nên hồ sơ rỗng thì
	// nó trả nil dù đã cài đặt hẳn hoi.
	NLCoTuHoSo: {func(ad Adapter) bool {
		return len(ad.ArgsHoSo(thuMucKhongTonTai)) > 0
	}, false},
	// Một chiều: mỗi provider một định dạng bản ghi, không có mẫu nào chung để
	// bắt adapter đã cài đặt phải trả true. Chiều xuôi (mẫu thật → đọc ra đúng
	// trường) đã có test riêng — xem ketqua_claude_test.go, ketqua_grok_test.go,
	// ketqua_antigravity_test.go.
	NLKetQuaCoCauTruc: {func(ad Adapter) bool {
		_, ok := ad.DocKetQua("KHONG-PHAI-BAN-GHI-CUA-AI-CA")
		return ok
	}, false},
	// Một chiều: cần một thư mục hồ sơ đã đăng nhập thật mới đọc ra hạn.
	NLHanToken: {func(ad Adapter) bool {
		_, ok := ad.TokenExpiry(thuMucKhongTonTai)
		return ok
	}, false},
	// Một chiều: cùng lý do.
	NLDanhTinh: {func(ad Adapter) bool {
		return ad.Identity(thuMucKhongTonTai) != ""
	}, false},
}

// KiemNangLuc đối chiếu LỜI KHAI của một adapter với HÀNH VI THẬT của nó, trả
// về danh sách chỗ lệch (rỗng = khớp).
//
// Vì sao là hàm của gói chứ không nằm trong file _test: báo cáo năng lực đi
// thẳng ra CLI và dashboard, nên chỗ lệch cũng phải đi ra cùng nó. Một bảng
// năng lực đẹp đẽ mà nội dung sai thì tệ hơn không có bảng — người vận hành sẽ
// tin nó. Conformance test gọi đúng hàm này, nên hai đường không thể nói ngược
// nhau.
func KiemNangLuc(ad Adapter) []string {
	var lech []string
	khai := map[string]NangLuc{}

	for _, nl := range ad.NangLuc() {
		if moTaNangLuc(nl.Khoa) == "" {
			lech = append(lech, fmt.Sprintf("khai năng lực lạ %q — không có trong MoiNangLuc", nl.Khoa))
			continue
		}
		if _, trung := khai[nl.Khoa]; trung {
			lech = append(lech, fmt.Sprintf("%s: khai hai lần", nl.Khoa))
			continue
		}
		switch nl.TrangThai {
		case LamDuoc, KhongLamDuoc, ChuaDo:
		default:
			lech = append(lech, fmt.Sprintf("%s: trạng thái %q không phải một trong ba trạng thái",
				nl.Khoa, nl.TrangThai))
			continue
		}
		if nl.BangChung == "" {
			lech = append(lech, fmt.Sprintf("%s: khai %q mà không có bằng chứng — "+
				"đo ở đâu, hoặc vì sao chưa đo được", nl.Khoa, nl.TrangThai))
		}
		khai[nl.Khoa] = nl
	}

	for _, m := range MoiNangLuc {
		nl, co := khai[m.Khoa]
		if !co {
			lech = append(lech, fmt.Sprintf("%s (%s): KHÔNG KHAI — mặt nào hỏi tới cũng "+
				"không biết provider này làm được hay không", m.Khoa, m.Mo))
			continue
		}
		if do, coPhepDo := phepDoBaTrangThai[m.Khoa]; coPhepDo {
			if that := do(ad); that != nl.TrangThai {
				lech = append(lech, fmt.Sprintf("%s: khai %q nhưng adapter hành xử như %q",
					m.Khoa, nl.TrangThai, that))
			}
			continue
		}
		pd, coPhepDo := phepDoGiaTriNangLuc[m.Khoa]
		if !coPhepDo {
			continue
		}
		switch {
		case pd.co(ad) && nl.TrangThai != LamDuoc:
			lech = append(lech, fmt.Sprintf("%s: khai %q mà vẫn TRẢ VỀ giá trị thật — "+
				"lõi sẽ dùng giá trị đó trong khi bảng năng lực bảo là không có",
				m.Khoa, nl.TrangThai))
		case !pd.co(ad) && pd.haiChieu && nl.TrangThai == LamDuoc:
			lech = append(lech, fmt.Sprintf("%s: khai %q nhưng TRẢ RỖNG — "+
				"lời hứa trong bảng năng lực không có gì đứng sau", m.Khoa, LamDuoc))
		}
	}
	return lech
}
