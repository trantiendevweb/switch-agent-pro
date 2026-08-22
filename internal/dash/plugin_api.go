package dash

import (
	"net/http"

	"github.com/trantiendevweb/switch-agent-pro/internal/plugin"
)

// handlePlugins — action "plugin.list". Plugin nào đã cài, xin quyền gì, và host
// CHẶN được quyền đó tới đâu.
//
// Chỉ ĐỌC, và cố ý chỉ đọc: bảng này trả lời câu hỏi trước khi bấm chạy, còn
// chuyện chạy plugin thì đi qua flow (`flow.run`) như mọi loại node khác. Mở
// thêm một đường chạy riêng từ trình duyệt là mở thêm một chỗ phải canh quyền.
//
// DTO liệt kê tường minh từng trường, cùng luật với /api/nang-luc — và ở đây có
// một lý do thêm: struct của internal/plugin có [[secret]] kèm key_id, nên trả
// nguyên khối là để hở tên kho key ra trình duyệt mà không ai cố ý làm vậy.
func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request) {
	b := s.api.Plugins(s.workDir())

	type quyenDTO struct {
		Khoa      string `json:"khoa"`
		Mo        string `json:"mo"`
		LyDo      string `json:"ly_do"`
		Chan      string `json:"chan"`
		BangChung string `json:"bang_chung"`
	}
	type mucDTO struct {
		Ten      string     `json:"ten"`
		MoTa     string     `json:"mo_ta"`
		PhienBan string     `json:"phien_ban"`
		GiaoThuc int        `json:"giao_thuc"`
		Duong    string     `json:"duong"`
		CoExec   bool       `json:"co_exec"`
		Quyen    []quyenDTO `json:"quyen"`
		Secret   []string   `json:"secret"`
	}

	muc := make([]mucDTO, 0, len(b.Muc))
	var soChanThat, soKhongChanDuoc, soChuaDo int
	for _, m := range b.Muc {
		d := mucDTO{Ten: m.Ten, MoTa: m.MoTa, PhienBan: m.PhienBan, GiaoThuc: m.GiaoThuc,
			Duong: m.Duong, CoExec: m.CoExec, Quyen: []quyenDTO{}, Secret: m.Secret}
		if d.Secret == nil {
			d.Secret = []string{}
		}
		for _, q := range m.Quyen {
			switch plugin.TrangThaiChan(q.Chan) {
			case plugin.ChanThat:
				soChanThat++
			case plugin.KhongChanDuoc:
				soKhongChanDuoc++
			default:
				// Khoá lạ rơi vào đây cùng với ChuaDo, và đó là chiều AN TOÀN:
				// thứ không đọc được phải đếm là "chưa biết", không phải "đã chặn".
				soChuaDo++
			}
			d.Quyen = append(d.Quyen, quyenDTO{q.Khoa, q.Mo, q.LyDo, q.Chan, q.BangChung})
		}
		muc = append(muc, d)
	}
	loi := b.Loi
	if loi == nil {
		loi = []string{}
	}
	nguon := b.Nguon
	if nguon == nil {
		nguon = []string{}
	}
	// BA con số, KHÔNG phải một. Đây là chỗ dễ bẹp nhất của cả route này: cộng
	// "đã đo, host không chặn được" với "chưa ai đo" thành một số cho gọn thì
	// người vận hành đọc được đúng một câu "còn n chỗ chưa ổn" — trong khi hai
	// nửa của n đòi hai việc ngược nhau (một cái phải dựng hàng rào, một cái
	// phải đi đo). Server cộng sẵn để mọi mặt cộng giống nhau; nhưng cộng sẵn
	// thành BA ngăn riêng chứ không gộp.
	//
	// so_chua_chan giữ lại vì nó là tổng của hai ngăn sau và có tên nói đúng
	// điều đó ("chưa chặn"), nhưng KHÔNG mặt nào được hiện mình nó: hiện một
	// mình nó là bẹp ba trạng thái thành hai ngay trên màn hình.
	writeJSON(w, map[string]any{
		"muc":   muc,
		"nguon": nguon,
		"loi":   loi,

		"so_chan_that":       soChanThat,
		"so_khong_chan_duoc": soKhongChanDuoc,
		"so_chua_do":         soChuaDo,
		"so_chua_chan":       soKhongChanDuoc + soChuaDo,
	})
}
