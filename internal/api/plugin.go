package api

// Action "plugin.list" — plugin nào đã cài, xin quyền gì, và host có CHẶN được
// quyền đó không.
//
// Vì sao nằm trong hợp đồng chứ không phải một dòng in ở CLI: đây là câu người
// vận hành phải trả lời được TRƯỚC khi để một flow gọi plugin — thứ này chạy mã
// của người khác trên máy mình. Cùng vai trò với "provider.nang-luc", và cố ý
// cùng hình dạng: khoá, trạng thái, bằng chứng.

import (
	"github.com/trantiendevweb/switch-agent-pro/internal/plugin"
)

// QuyenPlugin là một dòng quyền của một plugin, ĐÃ GỘP hai chiều:
// plugin xin gì (LyDo), và host chặn được tới đâu (Chan, BangChung).
//
// Gộp ở tầng hợp đồng chứ không để mỗi mặt tự gộp: nếu CLI in "đã chặn" mà web
// in "chưa chặn" thì một trong hai đang nói dối, và không ai biết là cái nào.
type QuyenPlugin struct {
	Khoa      string `json:"khoa"`
	Mo        string `json:"mo"`
	LyDo      string `json:"lyDo"`      // plugin xin để làm gì
	Chan      string `json:"chan"`      // chan-that | khong-chan-duoc | chua-do
	BangChung string `json:"bangChung"` // chặn bằng cách nào, hoặc vì sao chưa
}

// MucPlugin là một plugin trong bảng.
type MucPlugin struct {
	Ten      string `json:"ten"`
	MoTa     string `json:"moTa"`
	PhienBan string `json:"phienBan"`
	GiaoThuc int    `json:"giaoThuc"`
	Duong    string `json:"duong"`
	Exec     string `json:"exec"`
	// CoExec: manifest có, nhưng file executable CÓ THẬT trên đĩa không.
	//
	// Tách khỏi việc manifest hợp lệ, vì hai chuyện này hỏng theo hai kiểu và
	// người dùng sửa chúng bằng hai cách khác nhau — "viết sai TOML" và "quên
	// build" không được hiện ra như cùng một lỗi.
	CoExec bool          `json:"coExec"`
	Quyen  []QuyenPlugin `json:"quyen"`
	// Secret liệt kê TÊN THAM CHIẾU, không bao giờ là giá trị.
	Secret []string `json:"secret"`
}

// BangPlugin là kết quả action "plugin.list".
type BangPlugin struct {
	Muc []MucPlugin `json:"muc"`
	// Nguon là các thư mục đã tìm — người dùng cần biết để đặt plugin cho đúng chỗ.
	Nguon []string `json:"nguon"`
	// Loi là manifest hỏng. Không nuốt: một plugin nằm đúng chỗ mà không hiện
	// trong bảng thì người dùng không có cách nào đoán ra vì sao.
	Loi []string `json:"loi"`
}

// Plugins — action "plugin.list".
func (a *API) Plugins(dir string) BangPlugin {
	ds, nguon, loi := plugin.Nap(dir)
	b := BangPlugin{Nguon: nguon, Loi: loi}
	for _, ten := range plugin.Ten(ds) {
		m := ds[ten]
		muc := MucPlugin{
			Ten: m.Plugin.Ten, MoTa: m.Plugin.MoTa, PhienBan: m.Plugin.PhienBan,
			GiaoThuc: m.Plugin.GiaoThuc, Duong: m.Duong, Exec: m.Plugin.Exec,
		}
		if _, err := plugin.TimExec(m); err == nil {
			muc.CoExec = true
		}
		for _, q := range m.Quyen {
			dong := QuyenPlugin{Khoa: q.Khoa, LyDo: q.LyDo}
			if mo := plugin.MoTaCuaQuyen(q.Khoa); mo != nil {
				dong.Mo, dong.Chan, dong.BangChung = mo.Mo, string(mo.Chan), mo.BangChung
			}
			muc.Quyen = append(muc.Quyen, dong)
		}
		for _, s := range m.Secret {
			// key_id, KHÔNG phải giá trị. Cùng luật với route API: mặt nào cũng
			// chỉ được thấy cái tên.
			muc.Secret = append(muc.Secret, s.Ten+" → key_id "+s.KeyID)
		}
		b.Muc = append(b.Muc, muc)
	}
	return b
}
