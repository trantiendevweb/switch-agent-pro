package aiapi

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// BẢNG NĂNG LỰC CỦA NỬA API — đối xứng với internal/provider/nangluc.go ở nửa CLI.
//
// Nửa CLI trả lời được "provider nào làm được gì" từ lâu. Nửa này thì KHÔNG, và
// chỗ trống đó có giá cụ thể: một flow dùng node `model` trỏ vào route không gọi
// được tool sẽ hỏng LÚC CHẠY — sau khi đã tiêu token của các bước trước — chứ
// không hỏng lúc `sagent flow validate`. Bảng này là thứ để validate hỏi.
//
// # BA TRẠNG THÁI, GIỮ NGUYÊN NGHĨA CỦA NỬA CLI
//
// Dùng THẲNG kiểu của gói provider bằng type alias, không chép ra bản thứ hai.
// Ba trạng thái là một quy ước về NGHĨA, mà hai bản sao của một quy ước là hai
// bản sẽ lệch nhau — và bản lệch bao giờ cũng là bản gộp "đã đo, KHÔNG" với
// "chưa ai đo" thành một chữ `false`. Alias làm chuyện lệch đó không xảy ra
// được: `aiapi.LamDuoc` và `provider.LamDuoc` là CÙNG MỘT kiểu, nên mặt web vẽ
// hai bảng bằng chung một hàm mà không phải đổi gì.
//
// # HAI CÂU HỎI, KHÔNG PHẢI MỘT
//
// Chỗ này khác nửa CLI, và khác vì bản chất chứ không vì tiện:
//
//	KHÁCH — mã của DỰ ÁN NÀY có gửi/đọc được thứ đó không?
//	NCC   — NHÀ CUNG CẤP của route đó có làm được không?
//
// Trộn hai câu vào một ô là làm mất đúng thứ đáng giá nhất. Ví dụ thật đo được
// hôm nay: modelapi.vn gọi tool ngon lành, nhưng `aiapi.yeuCau` không có trường
// `tools` nên lời gọi của ta chưa bao giờ mang tool đi. Một ô "không làm được"
// trơ trọi sẽ khiến người đọc đi đổi nhà cung cấp — sai hẳn hướng, vì chỗ hỏng
// nằm trong repo này và sửa được trong một buổi.
//
// Nên mỗi dòng mang cả ba: kết luận cuối (thứ `flow validate` đọc), phần khách,
// và phần nhà cung cấp. Cột `Cho` nói chỗ hỏng nằm bên nào.

// TrangThaiNangLuc là BA trạng thái, mượn nguyên của nửa CLI — xem
// internal/provider/nangluc.go để biết vì sao là ba chứ không phải hai.
type TrangThaiNangLuc = provider.TrangThaiNangLuc

const (
	LamDuoc      = provider.LamDuoc
	KhongLamDuoc = provider.KhongLamDuoc
	ChuaDo       = provider.ChuaDo
)

// Khoá của từng năng lực API. TỪ VỰNG CHUNG của cả bốn mặt: CLI, hợp đồng
// api.Actions, endpoint HTTP và mặt web đều gọi tên bằng đúng chuỗi này.
const (
	NLAPIGoiTool        = "goi-tool"
	NLAPIDauVaoAnh      = "dau-vao-anh"
	NLAPIDauRaCoCauTruc = "dau-ra-co-cau-truc"
	NLAPIReasoning      = "reasoning"
	NLAPIStreaming      = "streaming"
	NLAPIDemToken       = "dem-token-that"
	NLAPILietKeModel    = "liet-ke-model"
)

// MoiNangLucAPI là danh sách CHÍNH THỨC mọi năng lực nửa API hỏi tới.
//
// Cùng vai trò với `provider.MoiNangLuc` và với `api.Actions`: thêm một dòng vào
// đây mà quên phép đo tương ứng thì test đỏ, thay vì lõi lặng lẽ coi như "chưa
// đo". Thứ tự ở đây là thứ tự hiện ra trên mọi mặt.
//
// Bảy dòng này CHỌN theo đúng những gì một node `model` trong flow có thể đòi.
// Không có "vào âm thanh", "gọi song song nhiều tool", "cache prompt" — không
// phải vì chúng không tồn tại, mà vì chưa chỗ nào trong dự án hỏi tới, và một
// dòng chưa ai hỏi thì mãi mãi là một ô `ChuaDo` không ai đi đóng.
var MoiNangLucAPI = []struct{ Khoa, Mo string }{
	{NLAPIGoiTool, "gửi định nghĩa tool và nhận lại lời gọi tool"},
	{NLAPIDauVaoAnh, "gửi ảnh trong tin nhắn (vision)"},
	{NLAPIDauRaCoCauTruc, "ép câu trả lời theo JSON schema"},
	{NLAPIReasoning, "đọc được phần suy luận tách khỏi câu trả lời"},
	{NLAPIStreaming, "nhận chữ chảy về dần thay vì chờ cả cục"},
	{NLAPIDemToken, "đếm được token THẬT của mỗi lượt gọi"},
	{NLAPILietKeModel, "liệt kê được model nhà cung cấp có"},
}

// moTaNangLucAPI tra câu mô tả theo khoá. Rỗng = khoá lạ.
func moTaNangLucAPI(khoa string) string {
	for _, m := range MoiNangLucAPI {
		if m.Khoa == khoa {
			return m.Mo
		}
	}
	return ""
}

// Cho nói CHỖ HỎNG nằm bên nào — thứ quyết định người đọc đi sửa ở đâu.
const (
	ChoKhach      = "khach"        // mã trong repo này chưa gửi/đọc được
	ChoNCC        = "nha-cung-cap" // nhà cung cấp không làm
	ChoCaHai      = "ca-hai"       // cả hai bên đều không
	ChoChuaRo     = "chua-ro"      // chưa đo nên chưa biết bên nào
	ChoKhongVuong = "khong-vuong"  // chạy được, không vướng bên nào
)

// MoiCho là danh sách chính thức mọi giá trị `Cho`, để mặt nào vẽ nhãn cũng vẽ
// đủ — và để test bắt được một giá trị lạ lọt ra ngoài.
var MoiCho = []struct{ Ma, Nhan string }{
	{ChoKhach, "vướng ở phía dự án"},
	{ChoNCC, "vướng ở nhà cung cấp"},
	{ChoCaHai, "vướng cả hai bên"},
	{ChoChuaRo, "chưa đo nên chưa biết vướng đâu"},
	{ChoKhongVuong, "không vướng gì"},
}

// NhanCho đổi mã `Cho` sang câu người đọc hiểu ngay. Một nguồn cho cả bốn mặt —
// mỗi mặt tự đặt tên là hai mặt nói hai nghĩa.
func NhanCho(ma string) string {
	for _, c := range MoiCho {
		if c.Ma == ma {
			return c.Nhan
		}
	}
	return ma
}

// NangLucAPI là MỘT dòng của bảng.
//
// Ba cặp (trạng thái, bằng chứng) chứ không phải một, vì lý do ở đầu file. Mọi
// `BangChung*` đều KHÔNG ĐƯỢC RỖNG — luật giữ nguyên từ nửa CLI: một ô khai
// "làm được" mà không nói đo ở đâu thì chỉ là một lời hứa, và với ô `ChuaDo`
// thì lý do CHƯA đo được cũng đáng viết ra y như vậy.
type NangLucAPI struct {
	Khoa string `json:"khoa"`
	Mo   string `json:"mo"`

	// TrangThai là KẾT LUẬN CUỐI: gọi route này QUA `sagent` bây giờ thì có
	// dùng được năng lực đó không. Đây là ô mà `flow validate` đọc.
	TrangThai TrangThaiNangLuc `json:"trang_thai"`
	BangChung string           `json:"bang_chung"`
	Cho       string           `json:"cho"`

	// Khach: mã trong internal/aiapi có gửi/đọc được thứ đó không. Đo bằng
	// reflect trên chính các kiểu của gói — miễn phí, không chạm mạng, và tự
	// lật khi ai đó thêm trường vào struct.
	Khach          TrangThaiNangLuc `json:"khach"`
	BangChungKhach string           `json:"bang_chung_khach"`

	// NCC: nhà cung cấp của route đó có làm được không. Đo bằng mạng thật, ghi
	// vào sổ số đo bên dưới. Giữ lại KỂ CẢ khi phần khách đã chặn — đó chính là
	// thông tin "họ làm được, ta chưa gửi", thứ nói cho biết có đáng sửa không.
	NCC          TrangThaiNangLuc `json:"ncc"`
	BangChungNCC string           `json:"bang_chung_ncc"`
}

// NangLucRoute là bảng của MỘT route.
type NangLucRoute struct {
	Ten     string       `json:"ten"`
	BaseURL string       `json:"base_url"`
	Model   string       `json:"model"`
	Muc     []NangLucAPI `json:"muc"`
	Lech    []string     `json:"lech"`
}

// SoChuaDo đếm ô chưa ai đo của route này — con số người vận hành cần liếc.
func (n NangLucRoute) SoChuaDo() int {
	var s int
	for _, m := range n.Muc {
		if m.TrangThai == ChuaDo {
			s++
		}
	}
	return s
}

// LamDuoc trả lời câu mà `flow validate` hỏi: route này có dùng được năng lực
// `khoa` không, và nếu không thì vì sao.
//
// Trả về (false, lý do) cho CẢ `KhongLamDuoc` LẪN `ChuaDo`, nhưng lý do nói rõ
// hai chuyện khác nhau. Chỗ gọi được phép chạy tiếp khi chưa đo — nhưng phải
// nói ra, không được lặng lẽ coi như chạy được.
func (n NangLucRoute) LamDuoc(khoa string) (bool, string) {
	for _, m := range n.Muc {
		if m.Khoa != khoa {
			continue
		}
		if m.TrangThai == LamDuoc {
			return true, ""
		}
		return false, fmt.Sprintf("route %q: %s — %s", n.Ten, moTaNangLucAPI(khoa), m.BangChung)
	}
	return false, fmt.Sprintf("route %q: không có năng lực %q trong bảng", n.Ten, khoa)
}

// ---------------------------- phần KHÁCH: đo bằng reflect ----------------------------

// phepDoMaNguon trả lời "mã của gói này gửi/đọc được năng lực đó không", bằng
// cách soi CHÍNH các kiểu dựng yêu cầu và đọc phản hồi.
//
// VÌ SAO REFLECT CHỨ KHÔNG PHẢI MỘT BẢNG BOOL VIẾT TAY: một bảng viết tay là
// một lời khai nữa, và lời khai thì mục ruỗng theo thời gian. Hôm nay ai đó
// thêm `Tools` vào `yeuCau` để làm việc khác, bảng bool vẫn nói "không gửi
// được", và ô đó ở lại sai cho tới khi có người tình cờ đọc. Soi kiểu thật thì
// ô tự lật NGAY khi trường xuất hiện — và lúc lật, `KiemNangLucAPI` bắt được là
// phần NCC chưa có số đo, nên nó chuyển sang `ChuaDo` chứ không tự khai làm
// được. Đúng chiều an toàn.
//
// Đây cũng là chỗ khác nửa CLI: ở đó `phepDoGiaTri` gọi thẳng adapter và xem nó
// trả gì. Ở đây không gọi được — mọi phép "gọi" đều tốn tiền — nên thứ đo được
// miễn phí là HÌNH DẠNG của yêu cầu ta gửi đi.
var phepDoMaNguon = map[string]func() (TrangThaiNangLuc, string){
	NLAPIGoiTool: func() (TrangThaiNangLuc, string) {
		guiDuoc := coTruong(reflect.TypeOf(yeuCau{}), "tools")
		docDuoc := coTruong(reflect.TypeOf(phanHoi{}), "choices", "message", "tool_calls")
		// Câu hỏi THỨ BA, và nó là chỗ điểm mù của cả bảng này: có chỗ nào CHỨA
		// giá trị đọc được không. Hai câu trên chỉ hỏi "kiểu có trường đó
		// không" — thêm `tools` vào `yeuCau` là ô này lật xanh NGAY, kể cả khi
		// `Goi` không bao giờ gán gì. Hỏi thêm `KetQua` thu hẹp điểm mù được
		// một nấc, chứ KHÔNG xoá được nó: bảng vẫn không trả lời được câu "có
		// ai chép giá trị đi không". Câu đó chỉ có bài kiểm đi hết đường mới
		// trả lời được — TestToolDiHetDuongTuDinhNghiaToiKetQua (tool_test.go).
		coChoChua := coTruongGo(reflect.TypeOf(KetQua{}), "ToolCalls")
		switch {
		case guiDuoc && docDuoc && coChoChua:
			return LamDuoc, "aiapi.yeuCau có trường `tools`, aiapi.phanHoi đọc `tool_calls`, " +
				"và KetQua.ToolCalls mang nó ra tới người gọi (aiapi.GoiTool, tool.go)"
		case guiDuoc && docDuoc:
			return KhongLamDuoc, "aiapi đọc được `tool_calls` nhưng KetQua KHÔNG có chỗ chứa — " +
				"lời gọi tool dừng lại trong lõi, không ai ngoài gói này thấy"
		case guiDuoc:
			// Gửi được mà không đọc được thì tool_call bay vào hư không: nhà
			// cung cấp trả lời đúng, lõi vứt đi, người dùng thấy câu trả lời rỗng.
			return KhongLamDuoc, "aiapi.yeuCau gửi được `tools` nhưng aiapi.phanHoi KHÔNG đọc `tool_calls` — " +
				"lời gọi tool trả về sẽ bị vứt lặng lẽ"
		default:
			return KhongLamDuoc, "aiapi.yeuCau không có trường `tools` — lời gọi của dự án này " +
				"chưa bao giờ mang định nghĩa tool đi (aiapi.go, kiểu `yeuCau`)"
		}
	},
	NLAPIDauVaoAnh: func() (TrangThaiNangLuc, string) {
		t, co := kieuTruong(reflect.TypeOf(tinNhan{}), "content")
		if co && t.Kind() == reflect.String {
			// `content` là chuỗi thuần thì không có chỗ nào nhét `image_url`:
			// giao thức đòi một MẢNG phần tử {type,text|image_url}.
			return KhongLamDuoc, "aiapi.tinNhan.Content là `string` thuần — giao thức đòi content " +
				"dạng mảng {type, image_url} mới gửi được ảnh (aiapi.go, kiểu `tinNhan`)"
		}
		if !co {
			return ChuaDo, "không tìm thấy trường `content` trong aiapi.tinNhan — kiểu đã đổi, đo lại"
		}
		return LamDuoc, "aiapi.tinNhan.Content không còn là chuỗi thuần, chứa được phần ảnh"
	},
	NLAPIDauRaCoCauTruc: func() (TrangThaiNangLuc, string) {
		if coTruong(reflect.TypeOf(yeuCau{}), "response_format") {
			return LamDuoc, "aiapi.yeuCau có trường `response_format`"
		}
		return KhongLamDuoc, "aiapi.yeuCau không có trường `response_format` — không ép được " +
			"JSON schema, câu trả lời về dạng văn xuôi (aiapi.go, kiểu `yeuCau`)"
	},
	NLAPIReasoning: func() (TrangThaiNangLuc, string) {
		for _, ten := range []string{"reasoning_content", "reasoning"} {
			if !coTruong(reflect.TypeOf(phanHoi{}), "choices", "message", ten) {
				continue
			}
			// Cùng câu hỏi thứ ba như `goi-tool` ở trên: đọc được mà không có
			// chỗ chứa thì phần nghĩ vẫn bị vứt, và ô này vẫn xanh. Đó đúng là
			// trạng thái của gói suốt chiều 22/08 — `tinNhan.SuyLuan` có mặt,
			// `KetQua.SuyLuan` có mặt, mà không mặt nào đọc tới.
			if !coTruongGo(reflect.TypeOf(KetQua{}), "SuyLuan") {
				return KhongLamDuoc, "aiapi.phanHoi đọc `" + ten + "` nhưng KetQua KHÔNG có " +
					"chỗ chứa — phần suy luận dừng trong lõi, không ai ngoài gói này thấy"
			}
			return LamDuoc, "aiapi.phanHoi đọc được trường `" + ten + "`, KetQua.SuyLuan mang nó " +
				"ra tới người gọi, và aiapi.DocSuyLuan dựng câu cho cả CLI lẫn mặt web (suyluan.go)"
		}
		return KhongLamDuoc, "aiapi.phanHoi chỉ đọc `content` — phần suy luận nhà cung cấp trả về " +
			"bị vứt trước khi ai nhìn thấy (aiapi.go, kiểu `phanHoi`)"
	},
	NLAPIStreaming: func() (TrangThaiNangLuc, string) {
		if coTruong(reflect.TypeOf(yeuCauStream{}), "stream") {
			return LamDuoc, "aiapi.GoiStream gửi `stream:true` kèm `stream_options.include_usage` (stream.go)"
		}
		return KhongLamDuoc, "aiapi.yeuCauStream không có trường `stream`"
	},
	NLAPIDemToken: func() (TrangThaiNangLuc, string) {
		if coTruong(reflect.TypeOf(phanHoi{}), "usage") {
			return LamDuoc, "aiapi.phanHoi đọc `usage`, và KetQua.ThieuUsage nói ra khi nhà cung cấp không trả"
		}
		return KhongLamDuoc, "aiapi.phanHoi không đọc `usage` — sổ chi phí ghi 0 cho mọi lượt"
	},
	NLAPILietKeModel: func() (TrangThaiNangLuc, string) {
		if coTruong(reflect.TypeOf(SucKhoe{}), "soModel") {
			return LamDuoc, "aiapi.Kiem gọi GET /models và đếm được model (suckhoe.go)"
		}
		return KhongLamDuoc, "aiapi.SucKhoe không giữ số model đọc được"
	},
}

// coTruongGo tra một trường theo TÊN GO, không theo thẻ JSON.
//
// Cần riêng vì `KetQua` không có thẻ json nào — nó là kiểu trả cho người gọi
// trong Go, không phải hình dạng trên dây. Dùng `coTruong` cho nó thì sẽ luôn
// ra "không có", và ô nào hỏi tới sẽ khai KhongLamDuoc mãi mãi cho một thứ
// đang chạy tốt.
func coTruongGo(t reflect.Type, ten string) bool {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return false
	}
	_, co := t.FieldByName(ten)
	return co
}

// coTruong đi theo một ĐƯỜNG tên trường JSON trong một kiểu và nói có tới nơi
// không. Xuyên qua con trỏ và lát cắt, vì `choices` là lát cắt struct ẩn danh.
func coTruong(t reflect.Type, duong ...string) bool {
	_, co := kieuTruong(t, duong...)
	return co
}

// kieuTruong như coTruong nhưng trả về cả KIỂU của trường cuối — cần cho phép
// đo ảnh, nơi câu hỏi không phải "có trường content không" (luôn có) mà là
// "trường đó còn là chuỗi thuần không".
func kieuTruong(t reflect.Type, duong ...string) (reflect.Type, bool) {
	for _, ten := range duong {
		for t != nil && (t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array) {
			t = t.Elem()
		}
		if t == nil || t.Kind() != reflect.Struct {
			return nil, false
		}
		var thay reflect.Type
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := f.Tag.Get("json")
			if i := strings.Index(tag, ","); i >= 0 {
				tag = tag[:i]
			}
			if tag == ten {
				thay = f.Type
				break
			}
		}
		if thay == nil {
			return nil, false
		}
		t = thay
	}
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t, t != nil
}

// ---------------------------- phần NCC: sổ số đo trên mạng ----------------------------

// MocDo là MỘT ô đã đo trên mạng thật, bằng `sagent nang-luc-api --do`.
//
// Khoá theo (BaseURL, Model) chứ không theo TÊN ROUTE, vì tên route là do người
// dùng đặt trong `.sagent/project.toml`: đổi `deepseek` thành `ds` là mất sạch
// số đo, mà thứ được đo có đổi gì đâu. Ngược lại, đổi `model` của route sang một
// model khác thì số đo cũ KHÔNG còn đúng — và cách khoá này làm nó tự rơi về
// `ChuaDo`, đúng chiều an toàn.
type MocDo struct {
	BaseURL   string
	Model     string
	Khoa      string
	TrangThai TrangThaiNangLuc
	BangChung string
}

// soDoNangLuc là MỌI số đo đã chạy thật trên máy này.
//
// KHÔNG được thêm dòng vào đây bằng tay theo trí nhớ hay theo tài liệu của nhà
// cung cấp. Mỗi dòng phải là đầu ra của một lượt `sagent nang-luc-api --do` —
// và bằng chứng giữ NGUYÊN VĂN quan sát, kể cả thân lỗi, để người sau đọc được
// mà không phải tin ai.
//
// Vì sao luật này gắt tới vậy: ngày 22/08 dự án bắt quả tang bảng quyền plugin
// khai `chan-that` cho một thứ không chặn được. Bảng đó sống lâu được vì không
// ai đòi bằng chứng. Một bảng khai bừa tệ hơn không có bảng, vì người vận hành
// sẽ TIN nó.
var soDoNangLuc = []MocDo{
	// ===== modelapi.vn · deepseek-v4-flash — đo 22/08, `sagent nang-luc-api --do deepseek` =====
	{"https://modelapi.vn/v1", "deepseek-v4-flash", NLAPIGoiTool, LamDuoc,
		"đo 22/08: từ chối tool_choice=required (HTTP 400 \"Thinking mode does not support this " +
			"tool_choice\") → đo lại với tool_choice=auto: trả 1 tool_call, hàm \"lay_gio\", 445 token"},
	{"https://modelapi.vn/v1", "deepseek-v4-flash", NLAPIDauVaoAnh, KhongLamDuoc,
		"đo 22/08: HTTP 400 \"This model does not support image\" với ảnh PNG 32x32 (1024 điểm ảnh)"},
	{"https://modelapi.vn/v1", "deepseek-v4-flash", NLAPIDauRaCoCauTruc, KhongLamDuoc,
		"đo 22/08: HTTP 400 \"This response_format type is unavailable now\" với response_format " +
			"json_schema strict"},
	{"https://modelapi.vn/v1", "deepseek-v4-flash", NLAPIReasoning, LamDuoc,
		"đo 22/08: trả trường reasoning_content dài 152 ký tự, tổng 184 token, mất 1,5 giây"},
	{"https://modelapi.vn/v1", "deepseek-v4-flash", NLAPIStreaming, LamDuoc,
		"đo 22/08: 9 mẩu SSE mang chữ, mẩu cuối có usage (tổng 158 token), mất 1,5 giây"},
	{"https://modelapi.vn/v1", "deepseek-v4-flash", NLAPIDemToken, LamDuoc,
		"đo 22/08: usage thật vào 93 / ra 40 / tổng 133 token"},
	{"https://modelapi.vn/v1", "deepseek-v4-flash", NLAPILietKeModel, LamDuoc,
		"đo 22/08: GET /models trả 2 model, mất 26ms"},

	// ===== modelapi.vn · grok-4.5 — đo 22/08, `sagent nang-luc-api --do grok` =====
	{"https://modelapi.vn/v1", "grok-4.5", NLAPIGoiTool, LamDuoc,
		"đo 22/08: tool_choice=required chạy thẳng, trả 1 tool_call, hàm \"lay_gio\", 479 token"},
	{"https://modelapi.vn/v1", "grok-4.5", NLAPIDauVaoAnh, LamDuoc,
		"đo 22/08: nhận content dạng mảng có image_url và đọc ĐÚNG màu ảnh PNG 32x32 toàn đỏ, " +
			"trả \"đỏ\", 761 token"},
	{"https://modelapi.vn/v1", "grok-4.5", NLAPIDauRaCoCauTruc, LamDuoc,
		"đo 22/08: response_format json_schema strict được nhận, trả JSON đúng schema {\"mau\":\"đỏ\"}, " +
			"980 token"},
	{"https://modelapi.vn/v1", "grok-4.5", NLAPIReasoning, KhongLamDuoc,
		"đo 22/08: HTTP 200 nhưng KHÔNG có trường reasoning_content lẫn reasoning, dù lượt đó tiêu " +
			"951 token và mất 20,3 giây — nghĩa là model CÓ nghĩ, nhà bán lại không trả phần nghĩ ra"},
	{"https://modelapi.vn/v1", "grok-4.5", NLAPIStreaming, LamDuoc,
		"đo 22/08: 10 mẩu SSE mang chữ, mẩu cuối có usage (tổng 619 token), mất 8,9 giây"},
	{"https://modelapi.vn/v1", "grok-4.5", NLAPIDemToken, LamDuoc,
		"đo 22/08: usage thật vào 215 / ra 1088 / tổng 1303 token"},
	{"https://modelapi.vn/v1", "grok-4.5", NLAPILietKeModel, LamDuoc,
		"đo 22/08: GET /models trả 2 model, mất 24ms"},
}

// timMocDo tra số đo theo (base_url, model, khoá).
func timMocDo(baseURL, model, khoa string) (MocDo, bool) {
	b := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	for _, m := range soDoNangLuc {
		if strings.TrimRight(m.BaseURL, "/") == b && m.Model == model && m.Khoa == khoa {
			return m, true
		}
	}
	return MocDo{}, false
}

// coKey nói route này đã có file key chưa — CHỈ `Stat`, không đọc nội dung.
//
// Có mặt để ô `ChuaDo` nói được lý do CỤ THỂ ("chưa có key") thay vì "chưa kiểm
// tra". Hai câu đó dẫn tới hai việc khác nhau: một cái đi đặt key, một cái đi
// chạy phép đo.
//
// Không dùng `docKey` dù nó sẵn đó: hàm này chạy mỗi lần vẽ bảng, ở cả CLI lẫn
// mọi lần làm mới trên web. Đọc nội dung khoá bí mật vào bộ nhớ chỉ để biết file
// có tồn tại không là mở rộng chỗ khoá đi qua mà không được gì.
func coKey(id string) bool {
	if id == "" || strings.ContainsAny(id, `/\:`) || id == "." || id == ".." {
		return false
	}
	st, err := os.Stat(keyPath(id))
	return err == nil && st.Size() > 0
}

// ---------------------------- dựng bảng ----------------------------

// BangNangLuc dựng bảng năng lực của MỘT route.
//
// KHÔNG chạm mạng và KHÔNG tốn token: nó ghép phép đo mã nguồn (miễn phí, chạy
// tại chỗ) với sổ số đo đã chạy trước đó. Nhờ vậy `flow validate` gọi được nó ở
// mọi bước mà không tốn gì — mà đó chính là lý do bảng này tồn tại.
func BangNangLuc(r Route) NangLucRoute {
	out := NangLucRoute{Ten: r.Ten, BaseURL: r.BaseURL, Model: r.Model}
	for _, m := range MoiNangLucAPI {
		out.Muc = append(out.Muc, mucNangLuc(r, m.Khoa, m.Mo))
	}
	out.Lech = KiemNangLucAPI(out)
	return out
}

// BangNangLucNhieu dựng bảng cho nhiều route, thứ tự giữ nguyên như đầu vào.
func BangNangLucNhieu(ds []Route) []NangLucRoute {
	out := make([]NangLucRoute, 0, len(ds))
	for _, r := range ds {
		out = append(out, BangNangLuc(r))
	}
	return out
}

// mucNangLuc dựng MỘT dòng, và đây là chỗ hai câu hỏi được gộp thành kết luận.
//
// Thứ tự quyết định có chủ ý: phần KHÁCH xét trước. Nếu mã của ta chưa gửi được
// thì kết luận là KHÔNG bất kể nhà cung cấp làm được gì — vì câu hỏi của người
// viết flow là "chạy bây giờ có được không", không phải "về lý thuyết có được
// không". Nhưng số đo phía nhà cung cấp vẫn được giữ nguyên trong dòng, nên
// người đọc thấy ngay "họ làm được, ta chưa gửi" và biết chỗ cần sửa.
func mucNangLuc(r Route, khoa, mo string) NangLucAPI {
	m := NangLucAPI{Khoa: khoa, Mo: mo}

	if do, co := phepDoMaNguon[khoa]; co {
		m.Khach, m.BangChungKhach = do()
	} else {
		m.Khach = ChuaDo
		m.BangChungKhach = "chưa viết phép đo mã nguồn cho năng lực này"
	}

	if moc, co := timMocDo(r.BaseURL, r.Model, khoa); co {
		m.NCC, m.BangChungNCC = moc.TrangThai, moc.BangChung
	} else {
		m.NCC = ChuaDo
		switch {
		case r.BaseURL == "" || r.Model == "":
			m.BangChungNCC = fmt.Sprintf("route %q thiếu base_url hoặc model — không có gì để đo", r.Ten)
		case !coKey(r.KeyID):
			m.BangChungNCC = fmt.Sprintf("chưa có key: không thấy file key %q trong %s "+
				"— đặt bằng `sagent api key %s`", r.KeyID, KeysDir(), r.KeyID)
		default:
			m.BangChungNCC = fmt.Sprintf("có key nhưng CHƯA chạy phép đo cho model %q — "+
				"chạy: sagent nang-luc-api --do %s", r.Model, r.Ten)
		}
	}

	switch {
	case m.Khach == KhongLamDuoc:
		m.TrangThai, m.Cho, m.BangChung = KhongLamDuoc, ChoKhach, m.BangChungKhach
		// Phần nhà cung cấp KHÔNG được nuốt chỉ vì phía dự án đã chặn trước.
		// Ba nhánh dưới đây là ba việc phải làm khác hẳn nhau, và bản đầu của
		// file này gộp cả ba thành một câu "vướng ở phía dự án" — đúng kiểu bẹp
		// ba trạng thái thành một mà cả bảng dựng lên để chặn.
		switch m.NCC {
		case LamDuoc:
			// Câu đáng giá nhất của cả bảng: chỗ hỏng ở TRONG repo này.
			m.BangChung += " — nhưng nhà cung cấp LÀM ĐƯỢC (" + m.BangChungNCC +
				"), nên sửa được ở phía dự án"
		case KhongLamDuoc:
			// Sửa phía dự án KHÔNG cứu được ô này. Không nói ra thì ai đó sẽ đi
			// thêm trường vào `yeuCau` rồi mới phát hiện nhà cung cấp cũng chịu.
			m.Cho = ChoCaHai
			m.BangChung += " — và nhà cung cấp CŨNG KHÔNG (" + m.BangChungNCC +
				"), nên sửa phía dự án một mình không đủ"
		default:
			m.BangChung += " — phía nhà cung cấp thì chưa đo (" + m.BangChungNCC + ")"
		}
	case m.Khach == ChuaDo:
		m.TrangThai, m.Cho, m.BangChung = ChuaDo, ChoChuaRo, m.BangChungKhach
	case m.NCC == ChuaDo:
		m.TrangThai, m.Cho, m.BangChung = ChuaDo, ChoChuaRo, m.BangChungNCC
	case m.NCC == KhongLamDuoc:
		m.TrangThai, m.Cho, m.BangChung = KhongLamDuoc, ChoNCC, m.BangChungNCC
	default:
		m.TrangThai, m.Cho = LamDuoc, ChoKhongVuong
		m.BangChung = m.BangChungNCC + " · phía dự án: " + m.BangChungKhach
	}
	return m
}

// ---------------------------- đối chiếu ----------------------------

// KiemNangLucAPI soi CHÍNH BẢNG rồi trả về danh sách chỗ lệch (rỗng = sạch).
//
// Cùng vai trò với `provider.KiemNangLuc` ở nửa CLI, và cùng lý do để nó là hàm
// của gói chứ không nằm trong file _test: bảng đi thẳng ra CLI và dashboard,
// nên chỗ lệch cũng phải đi ra cùng nó. Conformance test gọi đúng hàm này, nên
// hai đường không thể nói ngược nhau.
//
// Điều nó BẮT được, và đây là danh sách những cách khai bừa đã nghĩ tới:
//
//   - thiếu một năng lực trong MoiNangLucAPI;
//   - trạng thái không phải một trong ba;
//   - ô nào đó không có bằng chứng;
//   - khai `LamDuoc` mà phần khách hoặc phần NCC chưa đo — tức là kết luận
//     "chạy được" không có phép đo nào đứng sau;
//   - bằng chứng của một ô `LamDuoc` phía NCC mà không nhắc tới quan sát nào có
//     con số hay mã HTTP — dấu hiệu của một dòng chép tay từ tài liệu.
func KiemNangLucAPI(b NangLucRoute) []string {
	var lech []string
	thay := map[string]bool{}

	for _, m := range b.Muc {
		if moTaNangLucAPI(m.Khoa) == "" {
			lech = append(lech, fmt.Sprintf("năng lực lạ %q — không có trong MoiNangLucAPI", m.Khoa))
			continue
		}
		if thay[m.Khoa] {
			lech = append(lech, fmt.Sprintf("%s: khai hai lần", m.Khoa))
			continue
		}
		thay[m.Khoa] = true

		for _, o := range []struct {
			ten string
			tt  TrangThaiNangLuc
			bc  string
		}{
			{"kết luận", m.TrangThai, m.BangChung},
			{"phần khách", m.Khach, m.BangChungKhach},
			{"phần nhà cung cấp", m.NCC, m.BangChungNCC},
		} {
			switch o.tt {
			case LamDuoc, KhongLamDuoc, ChuaDo:
			default:
				lech = append(lech, fmt.Sprintf("%s/%s: trạng thái %q không phải một trong ba trạng thái",
					m.Khoa, o.ten, o.tt))
			}
			if strings.TrimSpace(o.bc) == "" {
				lech = append(lech, fmt.Sprintf("%s/%s: khai %q mà không có bằng chứng — "+
					"đo ở đâu, hoặc vì sao chưa đo được", m.Khoa, o.ten, o.tt))
			}
		}

		// Kết luận `LamDuoc` phải có CẢ HAI phép đo nói được. Đây là chỗ chặn
		// đúng kiểu khai bừa đã xảy ra với bảng quyền plugin: một ô xanh mà
		// không phép đo nào đứng sau.
		if m.TrangThai == LamDuoc && (m.Khach != LamDuoc || m.NCC != LamDuoc) {
			lech = append(lech, fmt.Sprintf("%s: kết luận %q nhưng phần khách là %q và phần "+
				"nhà cung cấp là %q — một ô xanh không có phép đo nào đứng sau",
				m.Khoa, LamDuoc, m.Khach, m.NCC))
		}
		if m.NCC == LamDuoc && !coQuanSat(m.BangChungNCC) {
			lech = append(lech, fmt.Sprintf("%s: phần nhà cung cấp khai %q nhưng bằng chứng "+
				"không có quan sát nào (con số, mã HTTP, hay nguyên văn) — nghe như chép "+
				"từ tài liệu chứ không phải đo: %q", m.Khoa, LamDuoc, m.BangChungNCC))
		}
		if m.Cho == "" {
			lech = append(lech, fmt.Sprintf("%s: không nói chỗ hỏng nằm bên nào", m.Khoa))
		}
	}

	for _, m := range MoiNangLucAPI {
		if !thay[m.Khoa] {
			lech = append(lech, fmt.Sprintf("%s (%s): KHÔNG KHAI — mặt nào hỏi tới cũng không "+
				"biết route này làm được hay không", m.Khoa, m.Mo))
		}
	}
	sort.Strings(lech)
	return lech
}

// coQuanSat nói bằng chứng này có chứa một QUAN SÁT hay không.
//
// Phép thử thô nhất có thể mà vẫn dùng được: một chữ số ở đâu đó. Mọi quan sát
// thật của bộ đo đều mang số — số mẩu SSE, số token, số model, mã HTTP. Một câu
// chép từ tài liệu nhà cung cấp ("model này hỗ trợ gọi hàm") thì không.
//
// Cố ý KHÔNG chặt hơn: một phép kiểm bằng chứng quá chặt chỉ dạy người ta viết
// bằng chứng vừa đủ lách, và bản lách bao giờ cũng mơ hồ hơn bản thật.
func coQuanSat(s string) bool {
	return strings.ContainsAny(s, "0123456789")
}

// GhiSoDo dựng dòng `MocDo` để dán vào `soDoNangLuc` từ một kết quả đo thật.
//
// Có mặt để đường từ phép đo tới bảng KHÔNG đi qua trí nhớ của ai: chạy
// `sagent nang-luc-api --do --dan`, chép nguyên khối in ra, dán vào file này.
// Thiếu nó thì bước "ghi lại số đo" là bước chép tay, và chép tay là chỗ mọi
// bảng năng lực bắt đầu mục ruỗng.
func GhiSoDo(r Route, kq KetQuaDo, ngay string) MocDo {
	bc := kq.Chi
	if ngay != "" {
		bc = "đo " + ngay + ": " + bc
	}
	return MocDo{
		BaseURL:   strings.TrimRight(r.BaseURL, "/"),
		Model:     r.Model,
		Khoa:      kq.Khoa,
		TrangThai: kq.TrangThai,
		BangChung: bc,
	}
}
