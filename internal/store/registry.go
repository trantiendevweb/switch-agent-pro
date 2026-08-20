package store

// SỔ ĐĂNG KÝ (registry) — bảng `profiles_so` và `routes_so`, schema v8.
//
// Hai nguồn sự thật, cố ý:
//
//	SỔ  nói "sagent tạo ra thứ này, sagent chịu trách nhiệm về nó"
//	ĐĨA nói "thứ này đang có thật"
//
// Chúng lệch nhau là chuyện bình thường chứ không phải sự cố: người dùng dọn
// tay ~/.ai-accounts, OneDrive đồng bộ thiếu, hồ sơ được tạo từ máy khác, hoặc
// một thư mục lạ được thả vào kho. Cái sai là GỘP hai thứ đó làm một — khi đó
// "có thư mục" tự động thành "được phép xoá đệ quy", và đó chính là cách công cụ
// này xoá mất ~/.claude ngày 2026-08-17 (xem docs/DO-LUONG.md).
//
// Vì vậy `DoiChieuHoSo` CHỈ ĐỌC. Nó không tự "sửa" cho hai bên khớp: một hàm đối
// chiếu mà tự vá thì lần nhìn đầu tiên đã xoá sạch bằng chứng lệch, và không ai
// còn thấy lệch bao giờ nữa. Nhận vào sổ là một hành động RIÊNG (`NhanHoSo`).

import (
	"database/sql"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// HoSo là một dòng trong sổ hồ sơ.
type HoSo struct {
	ID       int64
	Provider string
	Account  string
	Dir      string
	// SoTaoRa: sổ khẳng định THƯ MỤC NÀY do sagent dựng ra. Đây là thứ duy nhất
	// cho phép xoá đệ quy — xem profile.RemoveTheoSo.
	SoTaoRa bool
	GhiLuc  time.Time
}

// Route là một dòng trong sổ route.
//
// KeyID là TÊN FILE khoá, không phải khoá. Xem ghi chú migration v8.
type Route struct {
	ID      int64
	Ten     string
	BaseURL string
	Model   string
	KeyID   string
	GhiLuc  time.Time
}

// Trạng thái của một mục sau khi đối chiếu.
const (
	SoKhop      = "khớp"
	SoThieuDia  = "sổ có, đĩa không" // hồ sơ biến mất sau lưng sagent
	SoThieuSo   = "đĩa có, sổ không" // thư mục lạ trong kho, hoặc hồ sơ có trước v8
	SoLechDuong = "lệch đường dẫn"   // cùng tên nhưng sổ trỏ một nơi, đĩa nằm nơi khác
)

// TrenDia là một hồ sơ NHÌN THẤY TRÊN ĐĨA. Tầng trên (api) dựng danh sách này
// từ profile.List() rồi đưa xuống — store cố ý không tự đi quét đĩa, để nó
// không phải biết luật di trú v1 và cũng để test đối chiếu không cần đĩa thật.
type TrenDia struct {
	Provider string
	Account  string
	Dir      string
}

// MucHoSo là một dòng kết quả đối chiếu HAI CHIỀU.
type MucHoSo struct {
	Provider  string
	Account   string
	DuongSo   string // đường dẫn sổ ghi; rỗng = sổ không biết hồ sơ này
	DuongDia  string // đường dẫn thật trên đĩa; rỗng = trên đĩa không có
	SoTaoRa   bool
	TrangThai string
}

// MucRoute là một dòng đối chiếu sổ route ↔ cấu hình.
//
// Ở đây "đĩa" là FILE CẤU HÌNH (.sagent/project.toml), còn sổ ghi route đã THẬT
// SỰ được gọi. Hai chiều lệch nói hai chuyện khác nhau và cả hai đều đáng biết:
// "cấu hình có, sổ không" = khai rồi nhưng chưa gọi bao giờ (chưa chắc chạy
// được); "sổ có, cấu hình không" = đã tiêu tiền qua route này rồi mà nay không
// còn khai — hoá đơn sẽ có dòng mà cấu hình không giải thích được.
type MucRoute struct {
	Ten            string
	BaseURLSo      string
	BaseURLCauHinh string
	Model          string
	KeyID          string
	TrangThai      string
}

// Lech: mục này có phải một chỗ sổ và đĩa không đồng ý với nhau không.
//
// Có method thay vì bắt người gọi so với hằng số, để mặt web (internal/dash)
// không phải import store chỉ vì một phép so chuỗi — package đó cố ý chỉ nói
// chuyện với internal/api, xem ghi chú đầu dash/server.go.
func (m MucHoSo) Lech() bool { return m.TrangThai != SoKhop }

// Lech: xem MucHoSo.Lech.
func (m MucRoute) Lech() bool { return m.TrangThai != SoKhop }

// GhiHoSo ghi (hoặc cập nhật) một hồ sơ vào sổ.
func (d *DB) GhiHoSo(h HoSo) error {
	_, err := d.db.Exec(`
		INSERT INTO profiles_so(provider,account,dir,so_tao_ra,ghi_luc)
		VALUES(?,?,?,?,?)
		ON CONFLICT(provider,account) DO UPDATE SET
			dir=excluded.dir, so_tao_ra=excluded.so_tao_ra, ghi_luc=excluded.ghi_luc`,
		h.Provider, h.Account, h.Dir, boolInt(h.SoTaoRa), time.Now().Format(time.RFC3339))
	return err
}

// NhanHoSo nhận một hồ sơ vào sổ CHỈ KHI sổ chưa biết nó. Trả về true nếu vừa
// nhận thêm dòng mới.
//
// Khác `GhiHoSo` ở đúng một điểm, và điểm đó là cả lý do hàm này tồn tại: dòng
// ĐÃ CÓ trong sổ là tiếng nói cuối cùng, kể cả (nhất là) khi nó nói "không sở
// hữu". Nếu nhận-lại mà đè lên thì bất cứ ai thả một thư mục vào kho cũng tự
// động được cấp quyền sở hữu ở lần chạy sau — đúng thứ cái sổ sinh ra để chặn.
func (d *DB) NhanHoSo(h HoSo) (bool, error) {
	res, err := d.db.Exec(`
		INSERT INTO profiles_so(provider,account,dir,so_tao_ra,ghi_luc)
		VALUES(?,?,?,?,?)
		ON CONFLICT(provider,account) DO NOTHING`,
		h.Provider, h.Account, h.Dir, boolInt(h.SoTaoRa), time.Now().Format(time.RFC3339))
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// HoSoTrongSo liệt kê mọi hồ sơ sổ đang giữ.
func (d *DB) HoSoTrongSo() ([]HoSo, error) {
	rows, err := d.db.Query(
		`SELECT id,provider,account,dir,so_tao_ra,ghi_luc FROM profiles_so ORDER BY provider,account`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HoSo
	for rows.Next() {
		var h HoSo
		var taoRa int
		var luc string
		if err := rows.Scan(&h.ID, &h.Provider, &h.Account, &h.Dir, &taoRa, &luc); err != nil {
			return nil, err
		}
		h.SoTaoRa = taoRa != 0
		h.GhiLuc, _ = time.Parse(time.RFC3339, luc)
		out = append(out, h)
	}
	return out, rows.Err()
}

// TimHoSo đọc một dòng theo địa chỉ.
func (d *DB) TimHoSo(provider, account string) (HoSo, bool, error) {
	var h HoSo
	var taoRa int
	var luc string
	err := d.db.QueryRow(
		`SELECT id,provider,account,dir,so_tao_ra,ghi_luc FROM profiles_so WHERE provider=? AND account=?`,
		provider, account).Scan(&h.ID, &h.Provider, &h.Account, &h.Dir, &taoRa, &luc)
	if err == sql.ErrNoRows {
		return HoSo{}, false, nil
	}
	if err != nil {
		return HoSo{}, false, err
	}
	h.SoTaoRa = taoRa != 0
	h.GhiLuc, _ = time.Parse(time.RFC3339, luc)
	return h, true, nil
}

// XoaHoSoKhoiSo gỡ một dòng khỏi sổ. Gọi SAU khi thư mục đã đi thật — sổ ghi
// một hồ sơ không còn tồn tại thì lần đối chiếu sau sẽ báo "sổ có, đĩa không".
func (d *DB) XoaHoSoKhoiSo(provider, account string) error {
	_, err := d.db.Exec(`DELETE FROM profiles_so WHERE provider=? AND account=?`, provider, account)
	return err
}

// SoHuu trả lời câu hỏi DUY NHẤT mà xoá-an-toàn cần hỏi sổ: thư mục này có phải
// do sagent dựng ra không.
//
// Đây là hàm thoả mãn interface profile.SoDangKy. So đường dẫn ĐÃ CHUẨN HOÁ chứ
// không so chuỗi thô: Windows không phân biệt hoa thường và đường dẫn đi vào sổ
// có thể mang dấu `\` thừa ở cuối, mà một phép so hụt ở đây thì im lặng biến
// thành "sổ không sở hữu" — tức là người dùng bỗng dưng không xoá được hồ sơ của
// chính mình.
func (d *DB) SoHuu(dir string) (bool, error) {
	ds, err := d.HoSoTrongSo()
	if err != nil {
		return false, err
	}
	for _, h := range ds {
		if cungDuong(h.Dir, dir) {
			return h.SoTaoRa, nil
		}
	}
	return false, nil
}

// GhiRoute ghi (hoặc cập nhật) một route vào sổ. KHÔNG ghi khoá — chỉ key_id.
func (d *DB) GhiRoute(r Route) error {
	_, err := d.db.Exec(`
		INSERT INTO routes_so(ten,base_url,model,key_id,ghi_luc)
		VALUES(?,?,?,?,?)
		ON CONFLICT(ten) DO UPDATE SET
			base_url=excluded.base_url, model=excluded.model,
			key_id=excluded.key_id, ghi_luc=excluded.ghi_luc`,
		r.Ten, r.BaseURL, r.Model, r.KeyID, time.Now().Format(time.RFC3339))
	return err
}

// RouteTrongSo liệt kê route sổ đang giữ.
func (d *DB) RouteTrongSo() ([]Route, error) {
	rows, err := d.db.Query(`SELECT id,ten,base_url,model,key_id,ghi_luc FROM routes_so ORDER BY ten`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Route
	for rows.Next() {
		var r Route
		var luc string
		if err := rows.Scan(&r.ID, &r.Ten, &r.BaseURL, &r.Model, &r.KeyID, &luc); err != nil {
			return nil, err
		}
		r.GhiLuc, _ = time.Parse(time.RFC3339, luc)
		out = append(out, r)
	}
	return out, rows.Err()
}

// XoaRouteKhoiSo gỡ một route khỏi sổ.
func (d *DB) XoaRouteKhoiSo(ten string) error {
	_, err := d.db.Exec(`DELETE FROM routes_so WHERE ten=?`, ten)
	return err
}

// DoiChieuHoSo đối chiếu HAI CHIỀU sổ ↔ đĩa. CHỈ ĐỌC (xem ghi chú đầu file).
//
// Kết quả sắp theo provider rồi account để bảng in ra ổn định — người ta so hai
// lần chạy bằng mắt, thứ tự nhảy là tự tạo báo động giả.
func (d *DB) DoiChieuHoSo(dia []TrenDia) ([]MucHoSo, error) {
	trongSo, err := d.HoSoTrongSo()
	if err != nil {
		return nil, err
	}
	// Khoá là (provider, account) — địa chỉ, không phải đường dẫn. Đường dẫn
	// chính là thứ đang được đem ra so.
	muc := map[string]*MucHoSo{}
	thuTu := []string{}
	lay := func(prov, acc string) *MucHoSo {
		k := prov + "\x00" + acc
		if m, ok := muc[k]; ok {
			return m
		}
		m := &MucHoSo{Provider: prov, Account: acc}
		muc[k] = m
		thuTu = append(thuTu, k)
		return m
	}
	for _, h := range trongSo {
		m := lay(h.Provider, h.Account)
		m.DuongSo, m.SoTaoRa = h.Dir, h.SoTaoRa
	}
	for _, t := range dia {
		lay(t.Provider, t.Account).DuongDia = t.Dir
	}

	out := make([]MucHoSo, 0, len(thuTu))
	for _, k := range thuTu {
		m := muc[k]
		switch {
		case m.DuongSo == "":
			m.TrangThai = SoThieuSo
		case m.DuongDia == "":
			m.TrangThai = SoThieuDia
		case !cungDuong(m.DuongSo, m.DuongDia):
			m.TrangThai = SoLechDuong
		default:
			m.TrangThai = SoKhop
		}
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].Account < out[j].Account
	})
	return out, nil
}

// DoiChieuRoute đối chiếu sổ route ↔ route khai trong cấu hình.
func (d *DB) DoiChieuRoute(cauHinh []Route) ([]MucRoute, error) {
	trongSo, err := d.RouteTrongSo()
	if err != nil {
		return nil, err
	}
	muc := map[string]*MucRoute{}
	var thuTu []string
	lay := func(ten string) *MucRoute {
		if m, ok := muc[ten]; ok {
			return m
		}
		m := &MucRoute{Ten: ten}
		muc[ten] = m
		thuTu = append(thuTu, ten)
		return m
	}
	for _, r := range trongSo {
		m := lay(r.Ten)
		m.BaseURLSo, m.Model, m.KeyID = r.BaseURL, r.Model, r.KeyID
	}
	for _, r := range cauHinh {
		m := lay(r.Ten)
		m.BaseURLCauHinh = r.BaseURL
		// Cấu hình là thứ ĐANG có hiệu lực, nên model/key_id lấy theo nó khi có.
		if r.Model != "" {
			m.Model = r.Model
		}
		if r.KeyID != "" {
			m.KeyID = r.KeyID
		}
	}
	out := make([]MucRoute, 0, len(thuTu))
	for _, ten := range thuTu {
		m := muc[ten]
		switch {
		case m.BaseURLSo == "" && m.BaseURLCauHinh != "":
			m.TrangThai = SoThieuSo
		case m.BaseURLCauHinh == "":
			m.TrangThai = SoThieuDia
		case m.BaseURLSo != m.BaseURLCauHinh:
			m.TrangThai = SoLechDuong
		default:
			m.TrangThai = SoKhop
		}
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ten < out[j].Ten })
	return out, nil
}

// cungDuong so hai đường dẫn theo luật của HỆ ĐIỀU HÀNH đang chạy.
func cungDuong(a, b string) bool {
	a = filepath.Clean(strings.TrimRight(a, `\/`))
	b = filepath.Clean(strings.TrimRight(b, `\/`))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
