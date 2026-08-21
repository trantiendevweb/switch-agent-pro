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
		Chan: ChanThat,
		BangChung: "host chỉ đặt cmd.Dir vào thư mục dự án khi plugin KHAI quyền này; " +
			"không khai thì cmd.Dir là một thư mục tạm RỖNG do host tạo. Đo bằng " +
			"TestKhongKhaiThuMucThiKhongThayThuMucDuAn (e2e_test.go) — chạy CÙNG một " +
			"binary hai lần, chỉ khác manifest, và plugin báo lại hai thư mục khác nhau.",
	},
	{
		Khoa: QuyenGhi,
		Mo:   "ghi vào thư mục làm việc",
		Chan: KhongChanDuoc,
		BangChung: "ĐÃ ĐO và kết luận là không: một khi tiến trình con thấy được đường dẫn " +
			"thì nó ghi được, host không có cách nào chặn ghi mà không dựng ACL riêng cho " +
			"từng lần chạy (chưa làm). Quyền này CHỈ để KHAI BÁO — đọc manifest thì biết " +
			"plugin định ghi, chứ không phải host đã chặn. Cách chặn thật đang có: không " +
			"khai " + QuyenThuMuc + " thì plugin không có đường dẫn để mà ghi vào.",
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
			"và không qua biến môi trường (con cháu của plugin thừa hưởng hết).",
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
