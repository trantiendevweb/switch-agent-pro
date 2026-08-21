package dash

import "net/http"

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
	var soChuaChan int
	for _, m := range b.Muc {
		d := mucDTO{Ten: m.Ten, MoTa: m.MoTa, PhienBan: m.PhienBan, GiaoThuc: m.GiaoThuc,
			Duong: m.Duong, CoExec: m.CoExec, Quyen: []quyenDTO{}, Secret: m.Secret}
		if d.Secret == nil {
			d.Secret = []string{}
		}
		for _, q := range m.Quyen {
			if q.Chan != "chan-that" {
				soChuaChan++
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
	// so_chua_chan đếm những quyền ĐANG ĐƯỢC XIN mà host KHÔNG chặn được (hoặc
	// chưa đo). Đây là con số người vận hành cần liếc — để mặt web tự cộng thì
	// mỗi mặt cộng một kiểu, và cách cộng là chỗ dễ nói khác nhau nhất.
	writeJSON(w, map[string]any{"muc": muc, "nguon": nguon, "loi": loi, "so_chua_chan": soChuaChan})
}
