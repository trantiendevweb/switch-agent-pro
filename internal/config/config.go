// Package config đọc cấu hình theo TẦNG, dưới đè lên trên:
//
//	mặc định của công cụ  →  global (~/.ai-accounts/config.toml)
//	→  project (.sagent/project.toml)  →  cờ dòng lệnh
//
// Nguyên tắc: **file cấu hình KHÔNG BAO GIỜ chứa secret**. Route/profile chỉ
// được tham chiếu bằng ID; API key và token nằm chỗ khác.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// ProjectDirName là thư mục cấu hình nằm trong repo của người dùng.
const ProjectDirName = ".sagent"

// CotPhien liệt kê MỌI cột mà bảng phiên của mặt 2D vẽ được, và là nguồn duy
// nhất để `ui.columns` được kiểm.
//
// Khai ở tầng config chứ không ở tầng dash, vì `sagent config` phải báo được
// tên cột sai NGAY LÚC ĐỌC FILE — bắt lỗi lúc mở trình duyệt thì đã muộn, và
// mặt web nào cũng phải đọc chung một danh sách này chứ không tự chế bản riêng.
var CotPhien = []string{"provider", "tai_khoan", "danh_tinh", "trang_thai", "pid", "nhanh", "bat_dau"}

// CotMacDinh là bộ cột dùng khi `ui.columns` không khai — đúng bốn cột mà bảng
// phiên đang vẽ từ trước, nên không khai gì thì giao diện KHÔNG đổi.
var CotMacDinh = []string{"provider", "tai_khoan", "danh_tinh", "trang_thai"}

// CotHopLe cho biết tên cột có vẽ được không.
func CotHopLe(ten string) bool {
	for _, c := range CotPhien {
		if c == ten {
			return true
		}
	}
	return false
}

// Config là cấu hình đã gộp đủ các tầng.
type Config struct {
	Version int    `toml:"version"`
	Name    string `toml:"name"`

	Project struct {
		Root          string `toml:"root"`
		DefaultBranch string `toml:"default_branch"`
		Workspace     string `toml:"workspace"` // "dir" | "worktree"
	} `toml:"project"`

	Commands struct {
		Setup []string `toml:"setup"`
		Lint  []string `toml:"lint"`
		Test  []string `toml:"test"`
		Build []string `toml:"build"`
	} `toml:"commands"`

	Policy struct {
		MaxParallelSessions int      `toml:"max_parallel_sessions"`
		RequireApprovalFor  []string `toml:"require_approval_for"`
	} `toml:"policy"`

	// UI là hợp đồng của Pha 5d: hai project khác nhau mở ra hai bố cục khác
	// nhau mà KHÔNG sửa mã. Mọi khoá ở đây chỉ là dữ liệu — mặt nào tiêu thụ
	// được thì tiêu thụ, tắt mặt đó đi lõi vẫn chạy.
	UI struct {
		DefaultSurface string `toml:"default_surface"` // tui | dashboard | workflow | 3d
		Theme          string `toml:"theme"`           // dark | light

		// Columns chọn cột nào hiện trên bảng phiên của mặt 2D, THEO ĐÚNG THỨ
		// TỰ khai. Rỗng = giữ bộ mặc định; không phải "giấu hết cột".
		Columns []string `toml:"columns"`

		// PinnedFlows ghim flow lên đầu workflow board. Tên phải khớp tên trong
		// flows.toml; tên lạ thì mặt board bỏ qua chứ không dựng flow ma.
		PinnedFlows []string `toml:"pinned_flows"`

		// Enable3D = false thì ẩn hẳn lối vào mặt ba chiều. Mặc định true.
		//
		// Là bool thường chứ không phải *bool có lý do: `merge` decode chồng lên
		// cấu hình đã có mặc định, nên khoá KHÔNG khai thì giữ true của tầng
		// trước, còn khai `enable_3d = false` thì ghi đè thật. Đủ để phân biệt
		// "không nói gì" với "nói không".
		Enable3D bool `toml:"enable_3d"`
	} `toml:"ui"`

	// AI là ĐƯỜNG THỨ HAI: gọi thẳng API thay vì qua CLI agent.
	//
	// `key_id` là TÊN file trong ~/.ai-accounts/api-keys, KHÔNG phải key. Luật
	// "file cấu hình không bao giờ chứa secret" (MASTER-PLAN mục 0) — file này
	// đi vào git của người dùng, còn key thì nằm trong kho đã siết ACL.
	AI struct {
		DefaultRoute   string   `toml:"default_route"`
		FallbackRoutes []string `toml:"fallback_routes"`
		Routes         []struct {
			Ten     string `toml:"ten"`
			BaseURL string `toml:"base_url"`
			Model   string `toml:"model"`
			KeyID   string `toml:"key_id"`
		} `toml:"route"`
	} `toml:"ai"`

	// Nguồn đã đọc, theo thứ tự áp dụng — để `sagent config` in ra cho người
	// dùng biết giá trị đến từ đâu thay vì phải đoán.
	Sources []string `toml:"-"`
}

// Default là cấu hình khi không có file nào.
func Default() Config {
	var c Config
	c.Version = 1
	c.Project.Workspace = "dir"
	c.Project.DefaultBranch = "main"
	c.Policy.MaxParallelSessions = 4
	c.UI.DefaultSurface = "tui"
	c.UI.Theme = "dark"
	c.UI.Enable3D = true
	return c
}

// GlobalPath là file cấu hình chung của máy.
func GlobalPath() string { return filepath.Join(paths.AccountsRoot(), "config.toml") }

// FindProjectFile đi ngược lên từ dir để tìm <repo>/.sagent/project.toml.
// Dừng ở gốc ổ đĩa. Trả về "" nếu không có.
func FindProjectFile(dir string) string {
	cur := dir
	for {
		p := filepath.Join(cur, ProjectDirName, "project.toml")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return ""
		}
		cur = parent
	}
}

// Load gộp các tầng cho thư mục làm việc dir.
func Load(dir string) (Config, error) {
	c := Default()

	if p := GlobalPath(); fileExists(p) {
		if err := merge(&c, p); err != nil {
			return c, fmt.Errorf("cấu hình chung %s: %w", p, err)
		}
		c.Sources = append(c.Sources, p)
	}
	if p := FindProjectFile(dir); p != "" {
		if err := merge(&c, p); err != nil {
			return c, fmt.Errorf("cấu hình dự án %s: %w", p, err)
		}
		c.Sources = append(c.Sources, p)
	}
	if err := c.validate(); err != nil {
		return c, err
	}
	return c, nil
}

// merge đọc file đè lên cấu hình đang có. TOML chỉ ghi những khoá có mặt trong
// file, nên khoá thiếu tự động giữ giá trị tầng trước — đúng ý nghĩa "đè".
func merge(c *Config, path string) error {
	_, err := toml.DecodeFile(path, c)
	return err
}

func (c Config) validate() error {
	switch c.Project.Workspace {
	case "", "dir", "worktree":
	default:
		return fmt.Errorf("project.workspace = %q — chỉ nhận \"dir\" hoặc \"worktree\"", c.Project.Workspace)
	}
	switch c.UI.DefaultSurface {
	case "", "tui", "dashboard", "workflow", "3d":
	default:
		return fmt.Errorf("ui.default_surface = %q — chỉ nhận tui|dashboard|workflow|3d", c.UI.DefaultSurface)
	}
	switch c.UI.Theme {
	case "", "dark", "light":
	default:
		return fmt.Errorf("ui.theme = %q — chỉ nhận dark|light", c.UI.Theme)
	}
	for _, cot := range c.UI.Columns {
		if !CotHopLe(cot) {
			return fmt.Errorf("ui.columns có %q — chỉ nhận %s", cot, strings.Join(CotPhien, "|"))
		}
	}
	// Bắt mâu thuẫn NGAY Ở ĐÂY thay vì để mặt web tự xử: khai mặt mặc định là
	// 3d rồi lại tắt 3d thì mở dashboard sẽ ra trang trống, mà người dùng không
	// hiểu vì sao — cấu hình sai phải kêu lúc đọc file, không phải lúc vẽ.
	if c.UI.DefaultSurface == "3d" && !c.UI.Enable3D {
		return fmt.Errorf("ui.default_surface = \"3d\" nhưng ui.enable_3d = false — mâu thuẫn")
	}
	if c.Policy.MaxParallelSessions < 0 {
		return fmt.Errorf("policy.max_parallel_sessions không được âm")
	}
	return nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// Sample là nội dung mẫu cho `sagent init`.
const Sample = `version = 1
name = "%s"

[project]
default_branch = "main"
workspace      = "worktree"   # "dir" = dùng chung thư mục | "worktree" = mỗi phiên một cây

[commands]
# lint = ["npm run lint"]
# test = ["npm test"]

[policy]
max_parallel_sessions = 4
require_approval_for  = ["merge", "deploy"]

[ui]
default_surface = "tui"       # tui | dashboard | workflow | 3d
theme           = "dark"      # dark | light
# columns      = ["provider", "tai_khoan", "danh_tinh", "trang_thai"]
# pinned_flows = ["kiem-tra-nhanh"]
# enable_3d    = true

# LƯU Ý: file này KHÔNG được chứa API key hay token. Chỉ tham chiếu bằng ID.
`
