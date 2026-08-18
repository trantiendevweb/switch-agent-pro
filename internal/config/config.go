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

	"github.com/BurntSushi/toml"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// ProjectDirName là thư mục cấu hình nằm trong repo của người dùng.
const ProjectDirName = ".sagent"

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

	UI struct {
		DefaultSurface string `toml:"default_surface"` // tui | dashboard | workflow | 3d
		Theme          string `toml:"theme"`
	} `toml:"ui"`

	// AI là ĐƯỜNG THỨ HAI: gọi thẳng API thay vì qua CLI agent.
	//
	// `key_id` là TÊN file trong ~/.ai-accounts/api-keys, KHÔNG phải key. Luật
	// "file cấu hình không bao giờ chứa secret" (MASTER-PLAN mục 0) — file này
	// đi vào git của người dùng, còn key thì nằm trong kho đã siết ACL.
	AI struct {
		DefaultRoute   string   `toml:"default_route"`
		FallbackRoutes []string `toml:"fallback_routes"`
		Routes       []struct {
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
theme           = "dark"

# LƯU Ý: file này KHÔNG được chứa API key hay token. Chỉ tham chiếu bằng ID.
`
