// Command sagent-plugin-mau là PLUGIN MẪU — một executable riêng nói JSON-RPC
// trên stdio với sagent (xem internal/plugin).
//
// Nó làm một việc thật, không phải một việc giả để có cái mà chạy: tóm lược một
// khối kết quả dài thành vài dòng cuối kèm số đo. Việc này đáng làm bằng plugin
// vì flow.MaxInject chỉ cho nhét 6000 ký tự sang bước sau — nhét thô thì phần bị
// cắt thường lại là phần có kết luận. Đặt một bước `tom-luoc` vào giữa thì bước
// sau nhận đúng phần cuối, và cách tóm lược sửa được mà không phải build lại
// sagent.
//
// KHÔNG XIN QUYỀN NÀO. Đây là chỗ nguyên tắc "capability tối thiểu" hiện ra
// trong một ví dụ chứ không phải trong một câu khẩu hiệu: plugin này chỉ cần chữ
// gửi qua stdio, nên nó không cần thư mục, không cần biến môi trường, không cần
// secret — và manifest của nó không khai gì cả. Nó vẫn BÁO LẠI (qua ghi_chu) thứ
// nó nhận được, để người vận hành đối chiếu được hàng rào của host với thực tế.
//
// Luật của phía plugin: stdout CHỈ dành cho JSON-RPC. Mọi thứ nói cho người đọc
// đi qua stderr hoặc qua trường ghi_chu.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/plugin"
)

// PhienBan của chính plugin này — khác GiaoThuc (số phiên bản cách nói chuyện).
const PhienBan = "0.1.0"

// SoDongMacDinh là số dòng cuối giữ lại khi bước flow không khai `so_dong`.
const SoDongMacDinh = 5

// TenDauMoc là tên file mà plugin dò trong THƯ MỤC HIỆN HÀNH để báo cáo lại cho
// host. Chỉ dùng cho phép đo hàng rào quyền (xem baoCaoQuyen) — plugin không đọc
// nội dung file, và không có nó thì plugin vẫn chạy bình thường.
const TenDauMoc = "sagent-dau-moc.txt"

// ThamSoThuDuongDan là tham số dùng CHỈ để đo hàng rào, không phải tính năng.
//
// Nó trả lời một câu mà `TenDauMoc` ở trên không trả lời được: host không ĐƯA
// đường dẫn dự án cho plugin — nhưng nếu plugin biết đường dẫn đó bằng cách
// khác (nhúng sẵn lúc build, đọc từ file cấu hình của chính nó, hoặc đoán
// `C:\Users\...`), thì nó có chạm tới nơi không?
//
// Vì sao đi qua THAM SỐ chứ không qua biến môi trường: biến môi trường CÓ hàng
// rào thật (không khai `bien-moi-truong` thì host dựng lại env từ danh sách
// trắng), nên dùng nó thì phép đo lẫn hai hàng rào vào nhau và không kết luận
// được gì. Tham số đi trong bản tin JSON-RPC, không hàng rào nào đụng tới — đó
// đúng là thứ cần cho một phép đo về hàng rào.
const ThamSoThuDuongDan = "duong-dan-thu"

// TenFileThuGhi là tên file plugin thử tạo trong thư mục được dò.
const TenFileThuGhi = "sagent-plugin-thu-ghi.tmp"

// Hai tham số dựng đúng hai ca HỎNG mà host khai là xử lý được nhưng chưa đo:
// plugin TREO, và plugin CHẾT giữa chừng.
//
// Vì sao phải dựng được chúng: cả hai ca đều nằm ở phía plugin, và host không
// có cách nào tự tạo ra chúng. Không có hai cần gạt này thì `chay.go` chỉ được
// đọc bằng mắt — mà "đọc bằng mắt thấy đúng" chính là thứ đã để lọt ô quyền
// `thu-muc-lam-viec` khai `chan-that` suốt từ lúc viết tới 22/08.
//
// Chúng KHÔNG phải cửa hậu: plugin này không xin quyền nào, treo hay chết thì
// cũng chỉ treo/chết chính nó, và host phải bắt được — đó đúng là thứ đang đo.
const (
	ThamSoThuTreoGiay = "thu-treo-giay" // ngủ N giây rồi mới trả lời
	ThamSoThuChet     = "thu-chet"      // thoát ngay, KHÔNG trả lời
)

type mau struct{}

// ThongTin: tên phải khớp manifest, và danh sách quyền là LỜI TỰ KHAI mà host
// đối chiếu với manifest. Trả nil = không cần gì.
func (mau) ThongTin() (string, string, []string) { return "tom-luoc", PhienBan, nil }

func (mau) Chay(moi plugin.MoiTruongChay, ts plugin.ThamSoChay) (plugin.KetQuaChay, error) {
	// Hai cần gạt đo hàng rào phía host. Đặt TRƯỚC mọi thứ khác: ca "chết giữa
	// chừng" phải chết trong khi host đang đợi phản hồi, không phải sau khi đã
	// trả lời xong.
	if v := strings.TrimSpace(ts.ThamSo[ThamSoThuChet]); v != "" {
		fmt.Fprintln(os.Stderr, "sagent-plugin-mau: thoat giua chung theo yeu cau cua phep do")
		os.Exit(3)
	}
	if v := strings.TrimSpace(ts.ThamSo[ThamSoThuTreoGiay]); v != "" {
		giay, err := strconv.Atoi(v)
		if err != nil || giay < 0 {
			return plugin.KetQuaChay{}, fmt.Errorf("tham_so.%s = %q phải là số nguyên ≥ 0",
				ThamSoThuTreoGiay, v)
		}
		time.Sleep(time.Duration(giay) * time.Second)
	}

	vao := strings.TrimSpace(ts.Vao)
	if vao == "" {
		// Hỏng to còn hơn trả về một bản tóm lược rỗng: bước sau sẽ nhận chuỗi
		// rỗng và chạy tiếp như không có gì xảy ra — đúng kiểu hỏng im lặng mà
		// flow.PhaiCo sinh ra để chặn.
		return plugin.KetQuaChay{}, fmt.Errorf("đầu vào rỗng — không có gì để tóm lược")
	}

	soDong := SoDongMacDinh
	if v := ts.ThamSo["so_dong"]; v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return plugin.KetQuaChay{}, fmt.Errorf("tham_so.so_dong = %q phải là số nguyên ≥ 1", v)
		}
		soDong = n
	}

	dong := strings.Split(strings.ReplaceAll(vao, "\r\n", "\n"), "\n")
	var coND []string
	soTu := 0
	for _, d := range dong {
		soTu += len(strings.Fields(d))
		if strings.TrimSpace(d) != "" {
			coND = append(coND, strings.TrimRight(d, " \t"))
		}
	}
	cuoi := coND
	if len(cuoi) > soDong {
		cuoi = cuoi[len(cuoi)-soDong:]
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "TÓM LƯỢC · %d dòng (%d dòng có nội dung) · %d từ · %d ký tự\n",
		len(dong), len(coND), soTu, len([]rune(vao)))
	fmt.Fprintf(&sb, "--- %d dòng cuối ---\n", len(cuoi))
	sb.WriteString(strings.Join(cuoi, "\n"))

	ghiChu := baoCaoQuyen(moi)
	if d := strings.TrimSpace(ts.ThamSo[ThamSoThuDuongDan]); d != "" {
		ghiChu += " " + thuChamDuongDan(d)
	}
	return plugin.KetQuaChay{Ra: sb.String(), GhiChu: ghiChu}, nil
}

// thuChamDuongDan thử ĐỌC, GHI và LIỆT KÊ một đường dẫn plugin được cho biết,
// rồi kể lại kết quả. Không phải tính năng — đây là dụng cụ đo.
//
// Vì sao dụng cụ này nằm trong plugin mẫu chứ không trong một binary "plugin
// xấu" riêng: cả hai hàng rào đáng đo (thư mục, biến môi trường) đều chỉ hiện
// ra khi chạy CÙNG MỘT binary với hai manifest khác nhau — thêm một binary thứ
// hai là thêm một biến số vào phép so, và bảng quyen.go sẽ lại được chấm bằng
// thứ khác với thứ nó nói về.
//
// KHÔNG in nội dung đọc được, chỉ in độ dài. Đường dẫn đem đi đo có thể là kho
// key thật; một plugin mẫu in nội dung ra ghi_chu là dạy sai ngay ở ví dụ đầu
// tiên, và ghi_chu thì đi thẳng vào log lẫn mặt web.
func thuChamDuongDan(duong string) string {
	phan := []string{"thu-duong-dan=" + duong}

	if b, err := os.ReadFile(duong); err == nil {
		phan = append(phan, fmt.Sprintf("thu-doc=duoc:%d", len(b)))
	} else {
		phan = append(phan, "thu-doc=khong")
	}

	// Ghi vào THƯ MỤC chứa đường dẫn được cho, không đè lên chính nó: phép đo
	// không được phá thứ nó đang đo.
	thuMuc := duong
	if st, err := os.Stat(duong); err == nil && !st.IsDir() {
		thuMuc = filepath.Dir(duong)
	}
	dich := filepath.Join(thuMuc, TenFileThuGhi)
	if err := os.WriteFile(dich, []byte("sagent do hang rao quyen"), 0o644); err == nil {
		phan = append(phan, "thu-ghi=duoc")
		_ = os.Remove(dich) // dọn ngay: dụng cụ đo không để lại rác
	} else {
		phan = append(phan, "thu-ghi=khong")
	}

	if muc, err := os.ReadDir(thuMuc); err == nil {
		phan = append(phan, fmt.Sprintf("thu-liet-ke=duoc:%d", len(muc)))
	} else {
		phan = append(phan, "thu-liet-ke=khong")
	}

	return strings.Join(phan, " ")
}

// baoCaoQuyen kể lại thứ host THẬT SỰ cấp cho lượt chạy này.
//
// Định dạng cố ý máy đọc được (khoa=gia-tri, cách nhau bằng dấu cách): test
// đầu-cuối của host dùng chính chuỗi này để ĐO hàng rào quyền — chạy cùng một
// binary với hai manifest khác nhau rồi so hai câu trả lời. Không có nó thì
// "host có chặn thật không" chỉ còn là lời của host tự nói về mình.
//
// KHÔNG BAO GIỜ in giá trị secret — chỉ tên và độ dài. Một plugin mẫu mà in
// secret ra là dạy sai ngay ở ví dụ đầu tiên người ta đọc.
func baoCaoQuyen(moi plugin.MoiTruongChay) string {
	phan := []string{}

	thuMuc := "(khong-cap)"
	if moi.ThuMuc != "" {
		thuMuc = moi.ThuMuc
	}
	phan = append(phan, "thu-muc="+thuMuc)

	// Hai câu trả lời khác nhau, và phải hỏi cả hai:
	//
	//	thu-muc=       — host NÓI nó cấp thư mục nào (đọc từ lượt bắt tay).
	//	cwd-thay-dau-moc= — tiến trình này THẬT SỰ đứng ở đâu.
	//
	// Chỉ hỏi câu đầu thì test của host là host tự chấm bài mình: nó khẳng định
	// "tôi không gửi đường dẫn", chứ không khẳng định "tiến trình con không đứng
	// sẵn trong thư mục dự án" — mà đó mới là chỗ hàng rào có thể thủng, vì
	// cmd.Dir để rỗng là Go lấy thư mục hiện hành của chính host.
	dauMoc := "khong"
	if _, err := os.Stat(TenDauMoc); err == nil {
		dauMoc = "co"
	}
	phan = append(phan, "cwd-thay-dau-moc="+dauMoc)

	// Biến đánh dấu do test của host đặt. Plugin thường KHÔNG cần đọc os.Getenv;
	// ở đây đọc là để báo cáo được hàng rào biến môi trường.
	dau := "khong"
	if os.Getenv("SAGENT_PLUGIN_MAU_DAU") != "" {
		dau = "co"
	}
	phan = append(phan, "bien-danh-dau="+dau)

	if len(moi.Secret) == 0 {
		phan = append(phan, "secret=(khong-co)")
	} else {
		ten := make([]string, 0, len(moi.Secret))
		for t := range moi.Secret {
			ten = append(ten, t)
		}
		sort.Strings(ten)
		for _, t := range ten {
			phan = append(phan, fmt.Sprintf("secret=%s:%d", t, len(moi.Secret[t])))
		}
	}

	quyen := append([]string(nil), moi.Quyen...)
	sort.Strings(quyen)
	if len(quyen) == 0 {
		quyen = []string{"(khong-co)"}
	}
	phan = append(phan, "quyen="+strings.Join(quyen, ","))

	return strings.Join(phan, " ")
}

func main() {
	if err := plugin.PhucVu(os.Stdin, os.Stdout, mau{}); err != nil {
		fmt.Fprintln(os.Stderr, "sagent-plugin-mau:", err)
		os.Exit(1)
	}
}
