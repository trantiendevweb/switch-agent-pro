// Package plugin thêm NĂNG LỰC MỚI cho sagent mà không phải sửa mã Go và không
// phải biên dịch lại binary.
//
// Bốn ràng buộc của thiết kế (MASTER-PLAN Pha 3, mục "Plugin model"). Mỗi cái là
// một quyết định có lý do, không phải sở thích:
//
//  1. TOML CHỈ MANIFEST/CẤU HÌNH TĨNH — không một dòng logic nào. File cấu hình
//     là thứ người ta chép cho nhau qua chat; cho nó chạy được logic là biến mỗi
//     lần chép thành một lần chạy mã lạ mà người chép không đọc nổi. Lược đồ
//     dưới đây cố ý KHÔNG có trường nào nhận biểu thức, script hay lệnh.
//
//  2. LOGIC ĐỘNG LÀ MỘT EXECUTABLE RIÊNG, nói chuyện qua JSON-RPC trên stdio, có
//     ĐÁNH SỐ PHIÊN BẢN (xem rpc.go). Tiến trình riêng thì chết riêng: plugin
//     panic không kéo theo sagent, và host còn chỗ để đặt hàng rào quyền.
//
//  3. SECRET TRONG TOML CHỈ LÀ THAM CHIẾU — key_id = "grok", không bao giờ là
//     giá trị. Cùng luật với route API (MASTER-PLAN mục 0): giá trị nằm ở
//     ~/.ai-accounts/api-keys/<id>.key, trong kho đã siết ACL.
//
//  4. CAPABILITY TỐI THIỂU — plugin phải KHAI TRƯỚC nó cần gì; không khai thì
//     không có. Xem quyen.go, trong đó ghi rõ quyền nào host CHẶN ĐƯỢC THẬT và
//     quyền nào CHƯA CHẶN ĐƯỢC (kèm lý do), theo đúng lối khai ba trạng thái của
//     internal/provider.
package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/trantiendevweb/switch-agent-pro/internal/config"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// PhienBanManifest là số phiên bản LƯỢC ĐỒ của plugin.toml.
//
// Tách khỏi GiaoThuc (rpc.go) vì hai thứ đổi vì hai lý do khác nhau: lược đồ đổi
// khi manifest có trường mới, giao thức đổi khi cách hai tiến trình nói chuyện
// đổi. Gộp một số thì mỗi lần thêm một trường TOML là ép mọi plugin đã biên dịch
// phải build lại.
const PhienBanManifest = 1

// Manifest là toàn bộ phần TĨNH của một plugin — đọc từ plugin.toml.
type Manifest struct {
	Version int      `toml:"version" json:"version"`
	Plugin  ThongTin `toml:"plugin" json:"plugin"`
	Quyen   []Quyen  `toml:"quyen" json:"quyen"`
	Secret  []Secret `toml:"secret" json:"secret"`

	// Duong là đường dẫn file manifest, điền lúc đọc. Không đọc từ TOML.
	Duong string `toml:"-" json:"duong"`
}

// ThongTin là khối [plugin].
type ThongTin struct {
	Ten      string   `toml:"ten" json:"ten"`
	MoTa     string   `toml:"mo_ta" json:"moTa"`
	PhienBan string   `toml:"phien_ban" json:"phienBan"`
	GiaoThuc int      `toml:"giao_thuc" json:"giaoThuc"`
	Exec     string   `toml:"exec" json:"exec"`
	Args     []string `toml:"args" json:"args"`
}

// Quyen là MỘT dòng khai quyền: plugin cần gì, và VÌ SAO.
//
// LyDo bắt buộc, cùng lý do với BangChung của provider.NangLuc: một dòng xin
// quyền không nói vì sao thì người duyệt chỉ còn cách bấm đồng ý.
type Quyen struct {
	Khoa string `toml:"khoa" json:"khoa"`
	LyDo string `toml:"ly_do" json:"lyDo"`
}

// Secret là một THAM CHIẾU tới bí mật, không phải bí mật.
//
// Ten là tên plugin sẽ nhận được; KeyID là tên file trong kho key. Không có
// trường nào nhận giá trị, và khoá lạ trong manifest bị Doc() từ chối — nên
// gia_tri = "sk-..." không lọt qua im lặng mà làm cả file hỏng.
type Secret struct {
	Ten   string `toml:"ten" json:"ten"`
	KeyID string `toml:"key_id" json:"keyId"`
}

var tenRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
var tenSecretRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// Co cho biết plugin CÓ KHAI quyền này hay không. Đây là hàm mọi hàng rào trong
// gói hỏi trước khi đưa thứ gì cho tiến trình con.
func (m Manifest) Co(khoa string) bool {
	for _, q := range m.Quyen {
		if q.Khoa == khoa {
			return true
		}
	}
	return false
}

// DuongExec là đường dẫn executable, tính từ thư mục chứa manifest.
func (m Manifest) DuongExec() string {
	return filepath.Join(filepath.Dir(m.Duong), m.Plugin.Exec)
}

// TimExec tìm executable THẬT trên đĩa, và nói rõ đã tìm ở đâu khi không thấy.
//
// Có nấc thử thêm đuôi .exe vì manifest là thứ TĨNH và chép được sang máy khác:
// bắt người viết ghi "tom-luoc.exe" là khoá manifest đó vào Windows. Chỉ thử
// thêm đuôi trên Windows — ở nơi khác, một file tên "x.exe" là một file khác
// hẳn, không phải cùng một chương trình.
func TimExec(m Manifest) (string, error) {
	duong := m.DuongExec()
	if _, err := os.Stat(duong); err == nil {
		return duong, nil
	}
	if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(duong), ".exe") {
		if _, err := os.Stat(duong + ".exe"); err == nil {
			return duong + ".exe", nil
		}
		return "", fmt.Errorf("plugin %q: không thấy executable %s (cũng đã thử %s.exe)",
			m.Plugin.Ten, duong, duong)
	}
	return "", fmt.Errorf("plugin %q: không thấy executable %s", m.Plugin.Ten, duong)
}

// Doc đọc và KIỂM một manifest.
//
// Điểm quan trọng nhất của hàm này là chỗ từ chối KHOÁ LẠ. BurntSushi/toml mặc
// định bỏ qua khoá không có trong struct — nghĩa là gia_tri = "sk-..." hay
// script = "..." sẽ nằm im trong file: người viết tưởng nó có tác dụng, người
// đọc tưởng nó đã được xử lý. Đây là đúng lớp hỏng mà ràng buộc 1 và 3 sinh ra
// để chặn, nên phải chặn ở tầng đọc chứ không phải bằng một câu dặn.
func Doc(duong string) (Manifest, error) {
	var m Manifest
	md, err := toml.DecodeFile(duong, &m)
	if err != nil {
		return Manifest{}, fmt.Errorf("%s: %w", duong, err)
	}
	if u := md.Undecoded(); len(u) > 0 {
		khoa := make([]string, 0, len(u))
		for _, k := range u {
			khoa = append(khoa, k.String())
		}
		sort.Strings(khoa)
		return Manifest{}, fmt.Errorf("%s: khoá lạ %s — manifest CHỈ nhận khoá có trong lược đồ. "+
			"Nếu định để giá trị bí mật ở đây thì dừng lại: secret chỉ được là tham chiếu key_id",
			duong, strings.Join(khoa, ", "))
	}
	m.Duong = duong
	if err := m.Kiem(); err != nil {
		return Manifest{}, fmt.Errorf("%s: %w", duong, err)
	}
	return m, nil
}

// Kiem kiểm phần tĩnh. Trả về lỗi ĐẦU TIÊN — manifest hỏng thì không chạy được,
// không có nấc "cảnh báo rồi chạy tiếp" như flow.Validate.
func (m Manifest) Kiem() error {
	if m.Version != PhienBanManifest {
		return fmt.Errorf("version = %d, công cụ này chỉ hiểu %d", m.Version, PhienBanManifest)
	}
	if !tenRe.MatchString(m.Plugin.Ten) {
		return fmt.Errorf("plugin.ten %q không hợp lệ — chỉ chữ thường, số, dấu - và _", m.Plugin.Ten)
	}
	if m.Plugin.MoTa == "" {
		return fmt.Errorf("plugin.mo_ta trống — bảng `sagent plugin` sẽ có một dòng không ai biết là gì")
	}
	if m.Plugin.GiaoThuc != GiaoThuc {
		return fmt.Errorf("plugin.giao_thuc = %d, host này nói giao thức %d", m.Plugin.GiaoThuc, GiaoThuc)
	}
	if err := kiemExec(m.Plugin.Exec); err != nil {
		return err
	}

	// Quyền: khoá phải có thật, không trùng, và phải nói lý do.
	thay := map[string]bool{}
	for _, q := range m.Quyen {
		if MoTaCuaQuyen(q.Khoa) == nil {
			return fmt.Errorf("quyen %q không có trong danh sách quyền (%s)",
				q.Khoa, strings.Join(KhoaQuyen(), ", "))
		}
		if thay[q.Khoa] {
			return fmt.Errorf("quyen %q khai hai lần", q.Khoa)
		}
		if strings.TrimSpace(q.LyDo) == "" {
			return fmt.Errorf("quyen %q không nói ly_do — người duyệt không có gì để duyệt", q.Khoa)
		}
		thay[q.Khoa] = true
	}

	// Secret: tên biến sạch, key_id là TÊN TRẦN chứ không phải đường dẫn.
	tenSecret := map[string]bool{}
	for _, s := range m.Secret {
		if !tenSecretRe.MatchString(s.Ten) {
			return fmt.Errorf("secret.ten %q không hợp lệ — chỉ chữ, số và _", s.Ten)
		}
		if tenSecret[s.Ten] {
			return fmt.Errorf("secret.ten %q khai hai lần", s.Ten)
		}
		tenSecret[s.Ten] = true
		if err := KiemKeyID(s.KeyID); err != nil {
			return fmt.Errorf("secret %q: %w", s.Ten, err)
		}
	}
	// Khai [[secret]] mà không khai quyền secret là mâu thuẫn — và phải nổ ở đây
	// chứ không im lặng bỏ qua lúc chạy. Nhờ luật này, đọc riêng khối quyen là
	// thấy HẾT thứ plugin được nhận, không phải dò cả file.
	if len(m.Secret) > 0 && !m.Co(QuyenSecret) {
		return fmt.Errorf("khai %d secret mà không khai quyen %q — host sẽ không đưa gì cả",
			len(m.Secret), QuyenSecret)
	}
	if len(m.Secret) == 0 && m.Co(QuyenSecret) {
		return fmt.Errorf("khai quyen %q mà không có [[secret]] nào — xin quyền thừa", QuyenSecret)
	}
	return nil
}

// kiemExec chặn manifest trỏ ra ngoài thư mục plugin.
//
// Không phải hàng rào an ninh tuyệt đối (người viết được manifest thì cũng đặt
// được file cạnh nó), mà là để một plugin chép từ chỗ khác về KHÔNG lặng lẽ chạy
// một binary đã có sẵn trên máy — thứ mà người nhận đọc manifest sẽ không nhận
// ra, vì một đường dẫn tuyệt đối trông cũng như mọi đường dẫn khác.
func kiemExec(exec string) error {
	if exec == "" {
		return fmt.Errorf("plugin.exec trống — plugin không có logic thì không phải plugin")
	}
	// Kiểm cả "bắt đầu bằng / hoặc \" chứ không chỉ filepath.IsAbs: trên Windows,
	// IsAbs("/bin/sh") trả về FALSE vì thiếu ký tự ổ đĩa — nhưng "/bin/sh" vẫn là
	// một đường dẫn có gốc, và nó lọt qua manifest trên Windows trong khi bị chặn
	// trên Linux. Một hàng rào chỉ đứng ở một nửa số máy là một hàng rào người ta
	// sẽ tin nhầm. (Bắt được bằng TestExecPhaiNamTrongThuMucPlugin, chạy trên
	// Windows — bản đầu chỉ có IsAbs và test đỏ ngay ở ca "/bin/sh".)
	if filepath.IsAbs(exec) || strings.HasPrefix(exec, "/") || strings.HasPrefix(exec, `\`) ||
		strings.Contains(exec, ":") {
		return fmt.Errorf("plugin.exec %q phải là đường dẫn TƯƠNG ĐỐI trong thư mục plugin", exec)
	}
	for _, phan := range strings.FieldsFunc(exec, func(r rune) bool { return r == '/' || r == '\\' }) {
		if phan == ".." {
			return fmt.Errorf("plugin.exec %q đi ra ngoài thư mục plugin", exec)
		}
	}
	return nil
}

// KiemKeyID kiểm một tham chiếu secret.
//
// Cùng luật với aiapi: tên đến từ file cấu hình mà người khác sửa được, và nó
// dùng để mở một file bí mật — nên chỉ nhận TÊN TRẦN, không nhận dấu phân cách.
func KiemKeyID(id string) error {
	if id == "" {
		return fmt.Errorf("thiếu key_id (secret trong TOML chỉ là THAM CHIẾU, không phải giá trị)")
	}
	if strings.ContainsAny(id, `/\:`) || id == "." || id == ".." {
		return fmt.Errorf("key_id %q không hợp lệ — chỉ dùng chữ, số, dấu - và _", id)
	}
	return nil
}

// ThuMucPlugin là các nơi tìm plugin, dưới đè lên trên — cùng tầng với flow.Paths.
func ThuMucPlugin(dir string) []string {
	var out []string
	if g := filepath.Join(paths.AccountsRoot(), "plugins"); laThuMuc(g) {
		out = append(out, g)
	}
	if p := config.FindProjectFile(dir); p != "" {
		if pp := filepath.Join(filepath.Dir(p), "plugins"); laThuMuc(pp) {
			out = append(out, pp)
		}
	}
	return out
}

func laThuMuc(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// Nap đọc mọi plugin áp dụng cho dir: <kho>/plugins/<ten>/plugin.toml, rồi
// <dự án>/.sagent/plugins/<ten>/plugin.toml đè lên.
//
// Trả về cả DANH SÁCH LỖI chứ không dừng ở cái hỏng đầu tiên: một plugin viết
// sai không được làm cho chín plugin còn lại biến mất khỏi bảng — người vận hành
// sẽ tưởng mình chưa cài gì.
func Nap(dir string) (map[string]Manifest, []string, []string) {
	ra := map[string]Manifest{}
	var loi []string
	nguon := ThuMucPlugin(dir)
	for _, thuMuc := range nguon {
		muc, err := os.ReadDir(thuMuc)
		if err != nil {
			loi = append(loi, fmt.Sprintf("%s: %v", thuMuc, err))
			continue
		}
		for _, e := range muc {
			if !e.IsDir() {
				continue
			}
			duong := filepath.Join(thuMuc, e.Name(), "plugin.toml")
			if _, err := os.Stat(duong); err != nil {
				continue
			}
			m, err := Doc(duong)
			if err != nil {
				loi = append(loi, err.Error())
				continue
			}
			// Tên trong manifest phải KHỚP tên thư mục. Nếu không, một plugin có
			// thể tự xưng là plugin khác và cướp chỗ của nó trong bảng tra —
			// người vận hành nhìn cây thư mục sẽ không thấy chuyện đó.
			if m.Plugin.Ten != e.Name() {
				loi = append(loi, fmt.Sprintf("%s: plugin.ten = %q nhưng thư mục tên %q",
					duong, m.Plugin.Ten, e.Name()))
				continue
			}
			ra[m.Plugin.Ten] = m
		}
	}
	sort.Strings(loi)
	return ra, nguon, loi
}

// Ten trả về tên plugin đã sắp xếp — map của Go trả ra ngẫu nhiên, mà bảng in ra
// cho người đọc thì không được đổi thứ tự mỗi lần chạy.
func Ten(m map[string]Manifest) []string {
	out := make([]string, 0, len(m))
	for t := range m {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
