package plugin

// Giao thức host <-> plugin: JSON-RPC 2.0 trên stdio, MỘT bản tin một dòng.
//
// Vì sao stdio chứ không phải socket/HTTP: cổng phải cấp phát, phải dọn, và ai
// cũng nối vào được. Ống stdin/stdout thì chỉ hai đầu nói chuyện, tự chết theo
// tiến trình, và không cần một dòng cấu hình nào.
//
// Vì sao MỘT DÒNG MỘT BẢN TIN: đọc bằng bufio.Scanner ở mọi ngôn ngữ, và bản ghi
// stdio đọc bằng mắt được. Không dùng khung Content-Length như LSP vì nó bắt phía
// plugin phải viết bộ phân tích khung trước khi làm được gì.
//
// Vì sao ĐÁNH SỐ PHIÊN BẢN: host và plugin nâng cấp rời nhau. Không có số phiên
// bản thì bất đồng giao thức hiện ra dưới dạng một trường thiếu, và cái đó nổ
// muộn — giữa một lượt flow đang chạy — thay vì nổ ngay lúc bắt tay.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// GiaoThuc là số phiên bản GIAO THỨC hiện tại. Host từ chối bắt tay với số khác.
//
// Luật nâng cấp: thêm trường tuỳ chọn thì GIỮ NGUYÊN số; đổi ý nghĩa hoặc bỏ
// trường thì TĂNG số. Nhờ vậy "số khác nhau" luôn có nghĩa là "không nói chuyện
// được", chứ không phải "có thể vẫn chạy, thử xem".
const GiaoThuc = 1

// Tên hai phương thức. Tiền tố sagent. để plugin còn chỗ đặt phương thức riêng
// mà không đụng tên với phương thức của khung.
const (
	MethodBatTay = "sagent.bat_tay"
	MethodChay   = "sagent.chay"
)

// Mã lỗi JSON-RPC. Bốn mã đầu là chuẩn; MaLoiPlugin là lỗi của chính plugin.
const (
	MaLoiCuPhap     = -32700
	MaLoiYeuCau     = -32600
	MaLoiKhongCoPT  = -32601
	MaLoiThamSo     = -32602
	MaLoiGiaoThuc   = -32001
	MaLoiChuaBatTay = -32002
	MaLoiPlugin     = -32000
)

// goiTin là một yêu cầu JSON-RPC.
type goiTin struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// traLoi là một phản hồi JSON-RPC.
type traLoi struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *LoiRPC         `json:"error,omitempty"`
}

// LoiRPC là lỗi plugin trả về. Giữ NGUYÊN VĂN Message, cùng luật với lỗi nhà
// cung cấp API (internal/aiapi): cắt gọt thông điệp là vứt đi thứ duy nhất dùng
// được khi phải đi hỏi người viết plugin.
type LoiRPC struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *LoiRPC) Error() string { return fmt.Sprintf("plugin trả lỗi %d: %s", e.Code, e.Message) }

// ThamSoBatTay là thứ HOST gửi cho plugin ở lượt bắt tay.
//
// Đây cũng là chỗ DUY NHẤT secret đi qua. Cố ý không đi qua argv (mọi tiến trình
// trên máy đọc được dòng lệnh của nhau) và không đi qua biến môi trường (tiến
// trình con cháu của plugin thừa hưởng hết).
type ThamSoBatTay struct {
	GiaoThuc int      `json:"giao_thuc"`
	Host     string   `json:"host"`
	Quyen    []string `json:"quyen"`   // quyền host ĐÃ CẤP
	ThuMuc   string   `json:"thu_muc"` // rỗng = không cấp quyền thư mục
	// Secret chỉ có mặt khi plugin khai quyền secret, và chỉ gồm ĐÚNG những tên
	// khai trong [[secret]].
	Secret map[string]string `json:"secret,omitempty"`
}

// KetQuaBatTay là thứ PLUGIN trả lời: nó là ai, và nó CẦN quyền gì.
//
// Quyen ở đây là lời tự khai của executable, để host đối chiếu với manifest —
// hai bên lệch nhau thì host từ chối chạy. Nhờ vậy manifest (thứ người ta ĐỌC
// ĐƯỢC) không thể tụt hậu so với thứ binary thật sự xin.
type KetQuaBatTay struct {
	GiaoThuc int      `json:"giao_thuc"`
	Ten      string   `json:"ten"`
	PhienBan string   `json:"phien_ban"`
	Quyen    []string `json:"quyen"`
}

// ThamSoChay là một lượt gọi việc.
type ThamSoChay struct {
	Vao    string            `json:"vao"`
	ThamSo map[string]string `json:"tham_so,omitempty"`
}

// KetQuaChay là kết quả một lượt gọi.
//
// Tách Ra khỏi GhiChu vì hai thứ đi hai đường: Ra là thứ bước sau của flow dùng
// ({{steps.x.output}}), GhiChu là thứ người vận hành đọc khi soi. Trộn lại thì
// mọi lời chú thích của plugin sẽ chui vào prompt của bước kế tiếp.
type KetQuaChay struct {
	Ra     string `json:"ra"`
	GhiChu string `json:"ghi_chu,omitempty"`
}

// ------------------------- phía PLUGIN -------------------------

// MoiTruongChay là những gì host ĐÃ CẤP cho lượt chạy này.
//
// Plugin đọc ở đây chứ không đọc os.Getenv/os.Getwd: thứ host cấp và thứ hệ điều
// hành để lọt có thể khác nhau, và plugin tử tế thì dùng cái được cấp.
type MoiTruongChay struct {
	ThuMuc string
	Quyen  []string
	Secret map[string]string
}

// Co cho biết một quyền có được cấp cho lượt chạy này không.
func (m MoiTruongChay) Co(khoa string) bool {
	for _, q := range m.Quyen {
		if q == khoa {
			return true
		}
	}
	return false
}

// XuLy là thứ người viết plugin phải cài đặt. Khung lo hết phần khung.
type XuLy interface {
	// ThongTin trả về danh tính và DANH SÁCH QUYỀN plugin cần. Phải khớp với
	// manifest, nếu không host từ chối chạy.
	ThongTin() (ten, phienBan string, quyen []string)
	// Chay làm việc chính.
	Chay(moi MoiTruongChay, ts ThamSoChay) (KetQuaChay, error)
}

// PhucVu chạy vòng lặp JSON-RPC của MỘT plugin: đọc từ vao, ghi ra ra.
//
// Ở trong gói này chứ không phải trong thư mục plugin mẫu, vì đây là mã DÙNG
// CHUNG cho cả hai phía: plugin mẫu gọi PhucVu, host gọi Client. Hai phía dùng
// chung một định nghĩa kiểu thì không có đường nào để chúng lệch nhau về khung
// bản tin — lệch giao thức chỉ còn xảy ra giữa các PHIÊN BẢN, và đó là đúng thứ
// số GiaoThuc sinh ra để bắt.
//
// Trả về nil khi host đóng stdin (kết thúc bình thường).
func PhucVu(vao io.Reader, ra io.Writer, x XuLy) error {
	sc := bufio.NewScanner(vao)
	sc.Buffer(make([]byte, 0, 64*1024), MaxBanTin)
	w := bufio.NewWriter(ra)

	var daBatTay bool
	var moi MoiTruongChay

	gui := func(t traLoi) error {
		t.JSONRPC = "2.0"
		b, err := json.Marshal(t)
		if err != nil {
			return err
		}
		if _, err := w.Write(append(b, '\n')); err != nil {
			return err
		}
		return w.Flush()
	}
	guiLoi := func(id, ma int, dinh string, a ...any) error {
		return gui(traLoi{ID: id, Error: &LoiRPC{Code: ma, Message: fmt.Sprintf(dinh, a...)}})
	}

	for sc.Scan() {
		dong := strings.TrimSpace(sc.Text())
		if dong == "" {
			continue
		}
		var g goiTin
		if err := json.Unmarshal([]byte(dong), &g); err != nil {
			if err := guiLoi(0, MaLoiCuPhap, "%v", err); err != nil {
				return err
			}
			continue
		}
		if g.JSONRPC != "2.0" {
			if err := guiLoi(g.ID, MaLoiYeuCau, "jsonrpc phải là \"2.0\", nhận %q", g.JSONRPC); err != nil {
				return err
			}
			continue
		}

		switch g.Method {
		case MethodBatTay:
			var ts ThamSoBatTay
			if err := json.Unmarshal(g.Params, &ts); err != nil {
				if err := guiLoi(g.ID, MaLoiThamSo, "%v", err); err != nil {
					return err
				}
				continue
			}
			if ts.GiaoThuc != GiaoThuc {
				// Từ chối chứ không cố đoán: host nói giao thức khác thì mọi
				// trường sau đó đều là phỏng đoán.
				if err := guiLoi(g.ID, MaLoiGiaoThuc,
					"host nói giao thức %d, plugin này nói %d", ts.GiaoThuc, GiaoThuc); err != nil {
					return err
				}
				continue
			}
			moi = MoiTruongChay{ThuMuc: ts.ThuMuc, Quyen: ts.Quyen, Secret: ts.Secret}
			daBatTay = true
			ten, pb, quyen := x.ThongTin()
			b, _ := json.Marshal(KetQuaBatTay{GiaoThuc: GiaoThuc, Ten: ten, PhienBan: pb, Quyen: quyen})
			if err := gui(traLoi{ID: g.ID, Result: b}); err != nil {
				return err
			}

		case MethodChay:
			if !daBatTay {
				// Chạy trước khi bắt tay = chạy khi chưa biết được cấp quyền gì.
				if err := guiLoi(g.ID, MaLoiChuaBatTay, "phải gọi %s trước", MethodBatTay); err != nil {
					return err
				}
				continue
			}
			var ts ThamSoChay
			if len(g.Params) > 0 {
				if err := json.Unmarshal(g.Params, &ts); err != nil {
					if err := guiLoi(g.ID, MaLoiThamSo, "%v", err); err != nil {
						return err
					}
					continue
				}
			}
			kq, err := x.Chay(moi, ts)
			if err != nil {
				if err := guiLoi(g.ID, MaLoiPlugin, "%v", err); err != nil {
					return err
				}
				continue
			}
			b, err := json.Marshal(kq)
			if err != nil {
				if err := guiLoi(g.ID, MaLoiPlugin, "không đóng gói được kết quả: %v", err); err != nil {
					return err
				}
				continue
			}
			if err := gui(traLoi{ID: g.ID, Result: b}); err != nil {
				return err
			}

		default:
			if err := guiLoi(g.ID, MaLoiKhongCoPT, "không có phương thức %q", g.Method); err != nil {
				return err
			}
		}
	}
	return sc.Err()
}

// MaxBanTin là trần một bản tin, cả hai phía.
//
// Có trần vì bản tin đi qua ống là thứ đầu kia quyết định độ dài: không chặn thì
// một plugin hỏng (hoặc một vòng lặp in ra vô tận) làm host nuốt hết bộ nhớ.
const MaxBanTin = 8 << 20 // 8 MiB
