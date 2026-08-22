package aiapi

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"  // đăng ký bộ đọc kích thước GIF
	_ "image/jpeg" // … JPEG
	_ "image/png"  // … PNG
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ẢNH TRONG TIN NHẮN — và câu hỏi phải trả lời trước khi gõ dòng đầu:
// `tinNhan.Content` đang là `string` THUẦN, đổi nó thì đụng tới đâu?
//
// # BA HƯỚNG ĐÃ CÂN, VÀ CÁI GIÁ CỦA TỪNG HƯỚNG
//
//  1. ĐỔI KIỂU `Content` sang `any`. Mọi chỗ DỰNG tin nhắn phải sửa, và tệ hơn
//     nhiều: mọi chỗ ĐỌC câu trả lời cũng phải sửa, vì `ph.Choices[0].Message.
//     Content` thôi là chuỗi. Chiều về của giao thức này LUÔN là chuỗi — đổi
//     kiểu của nó là bắt cả dự án viết một phép ép kiểu ở mỗi chỗ đọc, để phục
//     vụ một chiều đi ít dùng. Một phép ép kiểu hụt ở đó hỏng LÚC CHẠY, sau khi
//     lượt gọi đã trả tiền, chứ không hỏng lúc biên dịch.
//
//  2. `json.RawMessage`. Người gọi tự dựng JSON, tức là mỗi chỗ gọi phải tự
//     biết giao thức. Trình biên dịch thôi kiểm được gì, và chỗ gõ sai tên khoá
//     `image_url` sẽ trả HTTP 200 kèm một câu trả lời tử tế — nhà cung cấp nuốt
//     phần nó không hiểu. Đúng cái hình dạng lỗi mà `anhDoThiGiac`
//     (donangluc.go) dựng ra để chặn.
//
//  3. THÊM MỘT TRƯỜNG (`Phan`) + `MarshalJSON`. Hướng đã chọn.
//
// # CÁI GIÁ CỦA HƯỚNG ĐÃ CHỌN, NÓI THẲNG
//
// Thêm trường là có HAI đường cùng dựng phần nội dung của một tin nhắn:
// `Content` (chuỗi) và `Phan` (mảng). Hai đường làm cùng một việc thì sẽ có lúc
// lệch nhau, và câu hỏi "gán cả hai thì cái nào thắng" phải có câu trả lời
// KHAI RA, chứ không phải một hành vi tình cờ của hàm marshal.
//
// LUẬT HỢP NHẤT, và nó là lý do hướng này không đẻ ra hai đường lệch nhau:
//
//	Phan rỗng      → thân JSON y HỆT trước khi có file này (`content` là chuỗi).
//	Phan khác rỗng → `content` thành MẢNG, và `Content` KHÔNG bị vứt: nó vào
//	                 làm phần `text` ĐẦU TIÊN của mảng.
//
// Tức là hai trường KHÔNG tranh nhau — chúng ghép lại. Không có ca nào một
// trường bị nuốt lặng lẽ, nên không có ca nào người gọi mất chữ mà không biết.
// `TestAnhDiHetDuongToiThanJSON` canh đúng luật đó.
//
// Hệ quả thứ hai, đáng nói vì nó làm HỎNG MỘT PHÉP ĐO: sau khi có `MarshalJSON`
// thì KIỂU GO CỦA `Content` KHÔNG CÒN QUYẾT ĐỊNH HÌNH DẠNG TRÊN DÂY nữa. Phép
// đo `dau-vao-anh` cũ hỏi "Content còn là chuỗi thuần không" — từ nay câu đó
// trả lời SAI. Phép đo mới trong nangluc.go dựng thật một tin nhắn có ảnh rồi
// ĐỌC LẠI thân JSON. Xem ghi chú ở đó.

// PhanNoiDung là MỘT mẩu của tin nhắn nhiều phần.
//
// Hình dạng bám đúng giao thức tương thích OpenAI — `{"type":"text","text":…}`
// và `{"type":"image_url","image_url":{"url":…}}` — vì đúng lý do đã viết cho
// `Tool` (tool.go): một lớp bọc "thân thiện" rồi dịch qua lại là chỗ tên trường
// lệch đi trong im lặng, mà triệu chứng của nó là HTTP 200 kèm câu trả lời
// trông bình thường.
type PhanNoiDung struct {
	Loai string  `json:"type"`
	Chu  string  `json:"text,omitempty"`
	Anh  *AnhURL `json:"image_url,omitempty"`
}

// AnhURL bọc `url` — giao thức đòi một object ở đây chứ không phải chuỗi trần.
type AnhURL struct {
	URL string `json:"url"`
}

// PhanChu và PhanAnh dựng hai loại mẩu. Có mặt để chuỗi `"image_url"` chỉ được
// gõ ĐÚNG MỘT LẦN trong cả dự án.
func PhanChu(s string) PhanNoiDung { return PhanNoiDung{Loai: "text", Chu: s} }
func PhanAnh(dataURL string) PhanNoiDung {
	return PhanNoiDung{Loai: "image_url", Anh: &AnhURL{URL: dataURL}}
}

// Giới hạn của `DocAnh`, cả hai đều xét TRƯỚC khi chạm mạng.
const (
	// anhToiDaByte: ảnh to đi qua base64 phình thêm 1/3, và nhà cung cấp tính
	// tiền theo điểm ảnh. Một lượt gửi nhầm file 40 MB là một hoá đơn, không
	// phải một lỗi kỹ thuật.
	anhToiDaByte = 5 << 20

	// nguongDiemAnhDaDo là NGƯỠNG ĐÃ ĐO ĐƯỢC của grok-4.5 qua modelapi.vn, chép
	// nguyên văn từ bài học trong donangluc.go: ảnh 8×8 bị trả HTTP 400 "Image
	// has 64 total pixels (8x8), which is below the minimum of 512 pixels".
	//
	// CẢNH BÁO chứ KHÔNG CHẶN, và khác biệt đó có chủ ý: 512 là số của MỘT nhà
	// cung cấp đo được MỘT lần, không phải luật chung. Chặn theo nó là biến một
	// quan sát thành quy tắc cho mọi nhà cung cấp — đúng kiểu khai bừa mà bảng
	// năng lực dựng lên để chống.
	nguongDiemAnhDaDo = 512
)

// loaiAnhNhanDuoc là những định dạng gửi đi được.
//
// Danh sách trắng chứ không phải "cứ gửi đại": `http.DetectContentType` nhận ra
// hàng chục kiểu, và gửi một file .exe đổi đuôi thành .png cho nhà cung cấp là
// tốn một lượt gọi để nhận về HTTP 400 bằng tiếng Anh.
var loaiAnhNhanDuoc = []string{"image/png", "image/jpeg", "image/gif", "image/webp"}

// Anh là MỘT ảnh đã đọc xong và sẵn sàng lên dây.
//
// Giữ cả `Ten`, `SoByte`, `Rong`, `Cao` chứ không chỉ mỗi data URL: đó là những
// thứ mặt CLI in ra để người dùng biết ảnh nào THẬT SỰ đã đi. Không in ra thì
// "model bảo không thấy ảnh" và "sagent quên gắn ảnh" trông giống hệt nhau.
type Anh struct {
	Ten    string `json:"ten"`
	Loai   string `json:"loai"`
	SoByte int    `json:"so_byte"`

	// Rong/Cao bằng 0 nghĩa là KHÔNG ĐỌC ĐƯỢC kích thước (webp — thư viện chuẩn
	// của Go không có bộ giải mã), chứ không phải ảnh rỗng. Hai chuyện đó khác
	// nhau, nên chỗ nào dùng số này cũng phải hỏi `BietKichThuoc` trước.
	Rong int `json:"rong"`
	Cao  int `json:"cao"`

	dataURL string
}

// DataURL trả về chuỗi `data:image/png;base64,…` để nhét vào `image_url`.
func (a Anh) DataURL() string { return a.dataURL }

// BietKichThuoc nói có đọc được chiều rộng/cao hay không.
func (a Anh) BietKichThuoc() bool { return a.Rong > 0 && a.Cao > 0 }

// SoDiemAnh trả về số điểm ảnh, hoặc 0 khi không đọc được kích thước.
func (a Anh) SoDiemAnh() int {
	if !a.BietKichThuoc() {
		return 0
	}
	return a.Rong * a.Cao
}

// MoTa dựng một dòng người đọc hiểu ngay.
func (a Anh) MoTa() string {
	kt := fmt.Sprintf("kích thước KHÔNG đọc được (thư viện chuẩn không giải mã %s)", a.Loai)
	if a.BietKichThuoc() {
		kt = fmt.Sprintf("%d×%d = %d điểm ảnh", a.Rong, a.Cao, a.SoDiemAnh())
	}
	return fmt.Sprintf("%s · %s · %d byte · %s", a.Ten, a.Loai, a.SoByte, kt)
}

// DocAnh đọc một file ảnh từ đĩa và dựng `Anh`.
//
// MỌI lời từ chối ở đây xảy ra TRƯỚC khi chạm mạng, và đó là cả lý do hàm này
// tồn tại thay vì để người gọi tự nối chuỗi data URL: một lượt gọi hỏng vì file
// sai kiểu vẫn có thể bị tính tiền, và thân lỗi trả về là tiếng Anh của nhà
// cung cấp, nói về một `image_url` mà người dùng chưa từng gõ.
//
// Kiểu ảnh đọc bằng NỘI DUNG (`http.DetectContentType`), không theo đuôi file:
// đuôi là thứ người dùng gõ, nội dung là thứ nhà cung cấp đọc. Lệch nhau thì
// nhà cung cấp thắng, nên ta phải hỏi đúng thứ họ hỏi.
func DocAnh(duong string) (Anh, error) {
	b, err := os.ReadFile(duong)
	if err != nil {
		return Anh{}, fmt.Errorf("không đọc được ảnh %s: %w", duong, err)
	}
	if len(b) == 0 {
		return Anh{}, fmt.Errorf("ảnh %s rỗng (0 byte)", duong)
	}
	if len(b) > anhToiDaByte {
		return Anh{}, fmt.Errorf("ảnh %s nặng %d byte, quá mức %d byte của sagent — "+
			"qua base64 nó còn phình thêm 1/3, và nhà cung cấp tính tiền theo điểm ảnh",
			duong, len(b), anhToiDaByte)
	}
	loai := http.DetectContentType(b)
	if i := strings.Index(loai, ";"); i >= 0 {
		loai = strings.TrimSpace(loai[:i])
	}
	if !nhanDuocLoaiAnh(loai) {
		return Anh{}, fmt.Errorf("file %s không phải ảnh gửi được: nội dung là %q, "+
			"sagent chỉ gửi %s.\n     (Kiểu đọc theo NỘI DUNG file, không theo đuôi tên — "+
			"đuôi là thứ bạn gõ, nội dung là thứ nhà cung cấp đọc.)",
			duong, loai, strings.Join(loaiAnhNhanDuoc, ", "))
	}
	a := Anh{
		Ten:     filepath.Base(duong),
		Loai:    loai,
		SoByte:  len(b),
		dataURL: "data:" + loai + ";base64," + base64.StdEncoding.EncodeToString(b),
	}
	// Kích thước là thứ ĐỌC ĐƯỢC MIỄN PHÍ và nó nói trước được một lỗi đã xảy
	// ra thật (ảnh quá nhỏ, xem `nguongDiemAnhDaDo`). Không đọc được thì để 0
	// và nói ra ở chỗ dùng — không đoán.
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(b)); err == nil {
		a.Rong, a.Cao = cfg.Width, cfg.Height
	}
	return a, nil
}

// DocNhieuAnh đọc cả danh sách, dừng ở file hỏng ĐẦU TIÊN kèm số thứ tự.
//
// Nói ra file thứ mấy vì `--anh a.png --anh b.png --anh c.png` mà báo "không
// đọc được ảnh" trơ trọi thì người dùng phải thử lại từng cái một.
func DocNhieuAnh(duong []string) ([]Anh, error) {
	out := make([]Anh, 0, len(duong))
	for i, d := range duong {
		a, err := DocAnh(d)
		if err != nil {
			return nil, fmt.Errorf("ảnh thứ %d/%d: %w", i+1, len(duong), err)
		}
		out = append(out, a)
	}
	return out, nil
}

func nhanDuocLoaiAnh(loai string) bool {
	for _, l := range loaiAnhNhanDuoc {
		if l == loai {
			return true
		}
	}
	return false
}

// canhBaoAnh soi những ảnh sắp gửi và trả về các câu cần nói TRƯỚC khi gọi.
//
// Rỗng ở ca bình thường. Đây KHÔNG phải chỗ chặn — xem `nguongDiemAnhDaDo`.
func canhBaoAnh(ds []Anh) []string {
	var out []string
	for _, a := range ds {
		if a.BietKichThuoc() {
			if a.SoDiemAnh() < nguongDiemAnhDaDo {
				out = append(out, fmt.Sprintf("ảnh %s chỉ có %d điểm ảnh (%d×%d). Đo 22/08: "+
					"grok-4.5 qua modelapi.vn trả HTTP 400 cho ảnh 64 điểm ảnh — \"below the "+
					"minimum of 512 pixels\". Vẫn gửi, nhưng lượt này hỏng thì hỏng vì ảnh nhỏ, "+
					"KHÔNG phải vì route không đọc được ảnh.",
					a.Ten, a.SoDiemAnh(), a.Rong, a.Cao))
			}
			continue
		}
		out = append(out, fmt.Sprintf("ảnh %s (%s): sagent KHÔNG đọc được kích thước nên "+
			"không kiểm được ngưỡng %d điểm ảnh. Ảnh quá nhỏ thì nhà cung cấp trả HTTP 400, "+
			"và lỗi đó dễ bị đọc nhầm thành \"route không đọc được ảnh\".",
			a.Ten, a.Loai, nguongDiemAnhDaDo))
	}
	return out
}
