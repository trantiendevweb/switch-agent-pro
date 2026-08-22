package plugin

// Bảng QUYỀN của plugin — ràng buộc 4: plugin chạy với capability TỐI THIỂU.
//
// Bảng này có hai cột chứ không phải một, và đó là điểm chính của file:
//
//	KHAI  — plugin nói nó cần gì (nằm trong plugin.toml).
//	CHẶN  — host có THẬT SỰ chặn được thứ đó hay không.
//
// Gộp hai cột lại là chỗ mọi hệ thống quyền nói dối: liệt kê một quyền trong
// manifest trông như đã có hàng rào, trong khi hàng rào chỉ tồn tại nếu host cài
// đặt được nó. Ví dụ sống: quyền "mang" — host CHƯA có cách chặn một tiến trình
// con mở socket trên Windows, nên nếu chỉ có cột KHAI thì người vận hành sẽ đọc
// bảng và tưởng plugin không khai mạng thì không ra được mạng. Không phải vậy.
//
// Cột CHẶN dùng đúng ba trạng thái của internal/provider (nangluc.go), vì cùng
// một lý do: gộp "đã đo, làm không được" với "chưa ai đo" thì hai chuyện đòi hai
// cách xử lý ngược nhau trở nên im lặng như nhau.
type TrangThaiChan string

const (
	// ChanThat: host CHẶN THẬT, và có test đo được chuyện đó.
	ChanThat TrangThaiChan = "chan-that"
	// KhongChanDuoc: ĐÃ ĐO, host không chặn được bằng cơ chế đang có. Đây là một
	// KẾT LUẬN phải nói ra, không phải một khoảng trống.
	KhongChanDuoc TrangThaiChan = "khong-chan-duoc"
	// ChuaDo: CHƯA AI ĐO. Không được lặng lẽ coi như đã chặn.
	ChuaDo TrangThaiChan = "chua-do"
)

// Khoá của từng quyền — TỪ VỰNG CHUNG cho manifest, CLI và mặt web.
const (
	QuyenThuMuc    = "thu-muc-lam-viec"
	QuyenGhi       = "ghi-thu-muc-lam-viec"
	QuyenMoiTruong = "bien-moi-truong"
	QuyenSecret    = "secret"
	QuyenMang      = "mang"
)

// MoTaQuyen là MỘT dòng của bảng quyền.
//
// BangChung bắt buộc, cùng luật với provider.NangLuc: ChanThat mà không nói đo ở
// đâu thì chỉ là một lời hứa; ChuaDo mà không nói vì sao chưa đo thì không ai
// biết phải làm gì tiếp.
type MoTaQuyen struct {
	Khoa      string        `json:"khoa"`
	Mo        string        `json:"mo"`
	Chan      TrangThaiChan `json:"chan"`
	BangChung string        `json:"bangChung"`
}

// MoiQuyen là danh sách CHÍNH THỨC mọi quyền plugin xin được.
//
// Cố ý NGẮN. Mỗi dòng thêm vào đây là một thứ host phải chặn được thật hoặc phải
// thú nhận là chưa chặn được — không có nấc thứ ba.
var MoiQuyen = []MoTaQuyen{
	{
		Khoa: QuyenThuMuc,
		Mo:   "thấy thư mục làm việc của dự án",
		Chan: KhongChanDuoc,
		BangChung: "ĐÃ ĐO 22/08 và kết luận là KHÔNG chặn được. Thứ host làm được, và " +
			"làm thật: không khai quyền thì cmd.Dir là một thư mục tạm RỖNG chứ không " +
			"phải thư mục dự án — đo bằng TestKhongKhaiThuMucThiKhongThayThuMucDuAn " +
			"(e2e_test.go). Nhưng đó là KHÔNG CẤP ĐƯỜNG DẪN, không phải dựng hàng rào: " +
			"plugin biết đường dẫn bằng cách khác (nhúng sẵn lúc build, đọc file cấu " +
			"hình của chính nó, hoặc đoán C:\\Users\\...) thì vẫn đọc, ghi và liệt kê " +
			"được như thường. Đo bằng TestBangQuyenThuMucPhaiKhopVoiThucTeChamDuoc " +
			"(e2e_test.go): plugin KHÔNG khai quyền nào, host báo thu-muc=(khong-cap), " +
			"vậy mà plugin vẫn đọc đúng 33 byte của một file trong thư mục dự án, ghi " +
			"được file mới, và liệt kê được thư mục. Muốn khai ChanThat thì phải có " +
			"ACL riêng cho từng lượt chạy, Job Object, hoặc AppContainer — chưa cái nào " +
			"được làm. ĐO THÊM 22/08 15:45: một DACL thường (`icacls /deny`) CHẶN THẬT " +
			"được tiến trình con dù nó chạy cùng tài khoản Administrator — nhưng nó chặn " +
			"LUÔN CẢ HOST: tiến trình vừa tạo thư mục sau đó không xoá nổi thư mục của " +
			"chính nó. Plugin và host dùng chung một danh tính, nên DENY không phân biệt " +
			"được ai là ai. Hàng rào chỉ dùng được nếu plugin chạy dưới DANH TÍNH KHÁC " +
			"(token hạn chế / tài khoản riêng / AppContainer) — thay đổi kiến trúc, không " +
			"phải một dòng mã. Và CHƯA ĐO được plugin thù địch có tự `takeown` gỡ rào " +
			"không, nên chưa ai được khai đây là hàng rào. Xem docs/DO-LUONG.md 22/08.",
	},
	{
		Khoa: QuyenGhi,
		Mo:   "ghi vào thư mục làm việc",
		Chan: KhongChanDuoc,
		BangChung: "ĐÃ ĐO và kết luận là không: một khi tiến trình con thấy được đường dẫn " +
			"thì nó ghi được, host không có cách nào chặn ghi mà không dựng ACL riêng cho " +
			"từng lần chạy (chưa làm). Quyền này CHỈ để KHAI BÁO — đọc manifest thì biết " +
			"plugin định ghi, chứ không phải host đã chặn. LƯU Ý 22/08: dòng này từng kết " +
			"lại bằng câu \"cách chặn thật đang có: không khai " + QuyenThuMuc + " thì " +
			"plugin không có đường dẫn để mà ghi vào\" — câu đó ĐÃ BỊ PHÉP ĐO BÁC BỎ. " +
			"Không có đường dẫn do host cấp không có nghĩa là không có đường dẫn; xem " +
			"bằng chứng ở dòng " + QuyenThuMuc + ".",
	},
	{
		Khoa: QuyenMoiTruong,
		Mo:   "thừa hưởng biến môi trường của tiến trình cha",
		Chan: ChanThat,
		BangChung: "không khai thì host dựng môi trường TỪ ĐẦU, chỉ gồm danh sách trắng tối " +
			"thiểu để chạy được binary (xem bienToiThieu trong chay.go). Đo bằng " +
			"TestKhongKhaiMoiTruongThiKhongThayBienCuaCha (e2e_test.go): host đặt một biến " +
			"đánh dấu rồi hỏi plugin có thấy không.",
	},
	{
		Khoa: QuyenSecret,
		Mo:   "nhận giá trị secret đã khai trong [[secret]]",
		Chan: ChanThat,
		BangChung: "hàng rào nằm ở TẦNG ĐỌC MANIFEST: khai [[secret]] mà không khai quyền " +
			"này thì manifest hỏng ngay, không chạy được — đo bằng " +
			"TestKhaiSecretMaKhongKhaiQuyenThiTuChoi (manifest_test.go). Lúc chạy, host chỉ " +
			"đọc kho key khi có quyền, và chỉ đọc ĐÚNG những key_id đã khai — đo bằng " +
			"TestKhongKhaiSecretThiKhongNhanDuocGiaTri (e2e_test.go), bài này khẳng định cả " +
			"chuyện host KHÔNG mở kho key khi plugin không xin. Giá trị đi qua stdio trong " +
			"lượt bắt tay: CỐ Ý không qua argv (mọi tiến trình trên máy đọc được dòng lệnh) " +
			"và không qua biến môi trường (con cháu của plugin thừa hưởng hết). " +
			"ĐỌC KÈM DÒNG " + QuyenThuMuc + ", đừng đọc rời. ChanThat ở đây nói host " +
			"không ĐƯA secret cho plugin không xin — nó KHÔNG nói plugin không lấy được " +
			"secret. Plugin chạy cùng quyền hệ điều hành với host và không có hàng rào " +
			"tệp nào, nên plugin nào biết đường tới kho key vẫn đọc thẳng file được. " +
			"Hai câu đó khác nhau, và người đọc lướt sẽ gộp chúng làm một.",
	},
	{
		Khoa: QuyenMang,
		Mo:   "gọi ra mạng",
		Chan: ChuaDo,
		BangChung: "CHƯA ĐO, và phải nói thẳng: host hiện KHÔNG chặn tiến trình con mở " +
			"socket. Chặn được hay không thì chưa biết — trên Windows còn đường WFP/AppContainer " +
			"nhưng chưa ai thử, và chưa có phép đo nào trên máy thật. Khai quyền này nghĩa là " +
			"plugin TỰ NÓI nó ra mạng; không khai KHÔNG có nghĩa là nó bị chặn.",
	},
}

// MoTaCuaQuyen tra một dòng theo khoá; nil nếu khoá không có thật.
func MoTaCuaQuyen(khoa string) *MoTaQuyen {
	for i := range MoiQuyen {
		if MoiQuyen[i].Khoa == khoa {
			return &MoiQuyen[i]
		}
	}
	return nil
}

// KhoaQuyen liệt kê khoá hợp lệ, theo đúng thứ tự của MoiQuyen — thông điệp lỗi
// và bảng in ra phải liệt kê giống nhau ở mọi lần chạy.
func KhoaQuyen() []string {
	out := make([]string, 0, len(MoiQuyen))
	for _, q := range MoiQuyen {
		out = append(out, q.Khoa)
	}
	return out
}

// ChanDuocThat cho biết host có hàng rào THẬT cho quyền này không. Mặt nào in
// bảng quyền cũng phải phân biệt được, nếu không thì bảng đang hứa hộ host.
func ChanDuocThat(khoa string) bool {
	q := MoTaCuaQuyen(khoa)
	return q != nil && q.Chan == ChanThat
}
