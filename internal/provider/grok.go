package provider

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func init() { Register(grok{}) }

// grok bọc `grok` (@vibe-kit/grok-cli).
//
// Khác bốn provider kia ở một điểm đáng nói: đây là provider đầu tiên dùng
// **API key** thay vì đăng nhập OAuth. Không có trình duyệt, không có token hết
// hạn — chỉ một chuỗi trong file cấu hình.
//
// Đo trên máy thật:
//
//	base_url https://modelapi.vn/v1  ->  GET /v1/models  HTTP 200
//	model grok-4.5                   ->  trả lời trong 3,6s (gọi API thẳng)
//	qua CLI `grok -p`                ->  21,9s (nó là agent, không phải wrapper mỏng)
//
// Lưu ý: base_url KHÔNG phải api.x.ai. Key của người dùng mua qua một dịch vụ
// trung gian; chính xAI từ chối nó bằng "Incorrect API key provided". Nên adapter
// này không được hardcode endpoint — nó nằm trong file cấu hình của từng hồ sơ.
type grok struct{}

func (grok) Name() string { return "grok" }

// Cấu hình ở ~/.grok/user-settings.json nên tách bằng USERPROFILE.
//
// CLI cũng đọc GROK_API_KEY/GROK_BASE_URL, và biến môi trường thì sạch hơn hẳn.
// Nhưng lõi chỉ đặt ĐÚNG MỘT biến (EnvVar) và biến đó phải trỏ tới thư mục, nên
// dùng đường file cho thống nhất với bốn adapter kia. Đổi sang cơ chế biến-môi-
// trường là việc của lõi, không nên lách riêng ở đây.
func (grok) EnvVar() string { return "USERPROFILE" }

func (grok) Command() (string, error) {
	if p, err := exec.LookPath("grok"); err == nil {
		return p, nil
	}
	for _, n := range []string{"grok.cmd", "grok.exe", "grok"} {
		p := filepath.Join(os.Getenv("APPDATA"), "npm", n)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("không tìm thấy lệnh grok — cài bằng: npm i -g @vibe-kit/grok-cli")
}

// Đã đo: `grok -p "<prompt>"` chạy không tương tác và in kết quả ra stdout.
//
// CỐ Ý KHÔNG tự thêm `-m <model>`, dù thiếu nó thì CLI dùng model dựng sẵn
// `grok-code-fast-1` và endpoint trả 503 "No available channel".
//
// Vì sao không tự thêm: model là thuộc tính của TỪNG HỒ SƠ (mỗi hồ sơ có thể trỏ
// endpoint khác, bán model khác), mà HeadlessArgs chỉ nhận prompt — nó chạy ở
// tiến trình CHA, nơi USERPROFILE vẫn là của tài khoản gốc. Đọc cấu hình ở đó
// rồi áp cho mọi hồ sơ là đoán, và đoán sai model thì lỗi hiện ra ở tận endpoint.
//
// Thà để người dùng truyền tường minh: `sagent goc grok -m grok-4.5 -p "..."`.
// Verify() nói thẳng điều này để không ai phải tự mò.
func (grok) HeadlessArgs(prompt string) []string { return []string{"-p", prompt} }

// user-settings.json chứa CẢ apiKey lẫn baseURL — tức toàn bộ danh tính. Đây là
// file phải chép riêng cho từng hồ sơ, không bao giờ nối link dùng chung.
func (grok) PrivateFiles() []string { return []string{filepath.Join(".grok", "user-settings.json")} }

func (grok) SharedKeys() []string { return nil }

func (grok) BaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".grok")
}
func (grok) IdentitySource() string { return "" }

func (g grok) Version() (string, error) {
	p, err := g.Command()
	if err != nil {
		return "", err
	}
	return hoiVersion(p, "--version")
}

func grokSettings(configDir string) string {
	return filepath.Join(configDir, ".grok", "user-settings.json")
}

type grokCauHinh struct {
	APIKey       string `json:"apiKey"`
	BaseURL      string `json:"baseURL"`
	DefaultModel string `json:"defaultModel"`
}

func docGrok(configDir string) (grokCauHinh, bool) {
	var c grokCauHinh
	b, err := os.ReadFile(grokSettings(configDir))
	if err != nil {
		return c, false
	}
	if json.Unmarshal(b, &c) != nil {
		return c, false
	}
	return c, true
}

func (grok) HasToken(configDir string) bool {
	c, ok := docGrok(configDir)
	return ok && c.APIKey != ""
}

// Identity hiển thị ENDPOINT + MODEL, không phải email — provider này không có
// khái niệm tài khoản người dùng.
//
// TUYỆT ĐỐI không trả về apiKey: hàm này đi thẳng ra bảng `sagent ds` và ra
// dashboard.
func (grok) Identity(configDir string) string {
	c, ok := docGrok(configDir)
	if !ok || c.BaseURL == "" {
		return ""
	}
	if c.DefaultModel != "" {
		return c.BaseURL + " · " + c.DefaultModel
	}
	return c.BaseURL
}

// TokenExpiry: API key không có hạn dùng đọc được từ file. Trả false — đúng sự
// thật, không phải chưa làm.
func (grok) TokenExpiry(string) (time.Time, bool) { return time.Time{}, false }

// Token là file trong thư mục config, tách bằng USERPROFILE — đã đo: chạy trong
// HOME giả thì CLI báo "API key required".
func (grok) TachDuocTaiKhoan() bool { return true }

func (g grok) Verify() []Check {
	var out []Check
	p, err := g.Command()
	c := Check{Name: "tìm thấy lệnh grok", OK: err == nil, Detail: p}
	if err != nil {
		c.Detail = "chưa cài — npm i -g @vibe-kit/grok-cli"
	}
	out = append(out, c)

	home, _ := os.UserHomeDir()
	cfg, ok := docGrok(home)
	kc := Check{Name: "có API key trong user-settings.json", OK: ok && cfg.APIKey != "",
		Detail: grokSettings(home)}
	if !kc.OK {
		kc.Detail = "chưa cấu hình — grok -u <base_url> -k <key> -p test"
	}
	out = append(out, kc)

	uc := Check{Name: "biết gọi endpoint nào", OK: cfg.BaseURL != "", Detail: cfg.BaseURL}
	if cfg.BaseURL == "" {
		uc.Detail = "chưa có baseURL — key sẽ bị gửi tới api.x.ai (mặc định) và bị từ chối"
	}
	out = append(out, uc)

	// Đây là cái bẫy tốn thời gian nhất của provider này, nên nói ra ở chỗ người
	// dùng chắc chắn nhìn thấy.
	//
	// Đã đo: `grok -p "..."` BỎ QUA defaultModel trong user-settings.json và dùng
	// `grok-code-fast-1` dựng sẵn. Endpoint không bán model đó thì trả 503
	// "No available channel" — một thông điệp chẳng chỉ ra nguyên nhân.
	mc := Check{Name: "model cho lệnh headless", OK: false}
	if cfg.DefaultModel != "" {
		mc.Detail = "cấu hình ghi " + cfg.DefaultModel + ", NHƯNG lệnh grok -p bỏ qua nó " +
			"(đã đo) — phải truyền tường minh: sagent goc grok -m " + cfg.DefaultModel + ` -p "..."`
	} else {
		mc.Detail = "chưa đặt model; lệnh grok -p sẽ dùng grok-code-fast-1 dựng sẵn và " +
			"endpoint nhiều khả năng trả 503 — truyền -m <model> khi chạy"
	}
	out = append(out, mc)
	return out
}

// ArgsTuDuyetQuyen: đo `grok --help`: KHÔNG có approval/sandbox/permission nào. Grok chạy tool
// tự do theo thiết kế, nên cờ là thừa — nhưng đó là vì nó KHÔNG có rào, chứ
// không phải vì chưa đo. Chỉ có `--max-tool-rounds` giới hạn số vòng.
func (grok) ArgsTuDuyetQuyen() ([]string, bool) { return nil, true }

// ArgsThuMuc: đo `grok --help`: "-d, --directory <dir>  set working directory"
func (grok) ArgsThuMuc(dir string) []string { return []string{"-d", dir} }

// ArgsHoSo ép model theo user-settings.json của CHÍNH hồ sơ này.
//
// `grok -p` bỏ qua defaultModel (đo 18/08): nó tự chọn grok-code-fast-1, model
// mà modelapi.vn không phục vụ, nên trả 503 — và vì grok in lỗi ra như một câu
// trả lời bình thường, bước vẫn tính là xong. Không ép model thì mọi bước Grok
// đều hỏng lặng lẽ.
func (grok) ArgsHoSo(dir string) []string {
	b, err := os.ReadFile(filepath.Join(dir, ".grok", "user-settings.json"))
	if err != nil {
		return nil
	}
	var c struct {
		DefaultModel string `json:"defaultModel"`
	}
	if err := json.Unmarshal(b, &c); err != nil || c.DefaultModel == "" {
		return nil
	}
	return []string{"-m", c.DefaultModel}
}
