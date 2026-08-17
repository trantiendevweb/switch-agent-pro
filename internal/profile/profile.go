// Package profile là các động từ nguyên thuỷ thao tác trên một hồ sơ
// (provider, account): create / link / list / remove / run.
package profile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/jsonutil"
	"github.com/trantiendevweb/switch-agent-pro/internal/link"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// Dir là thư mục config biệt lập của một hồ sơ (vị trí chuẩn của v2).
func Dir(prov, account string) string {
	return filepath.Join(paths.AccountsRoot(), prov, account)
}

// ResolveDir tìm thư mục hồ sơ THẬT: chỗ chuẩn trước, không có thì tìm trong
// kho v1 (~/.claude-accounts/<tên>).
//
// Vì sao cần: `List()` đã biết di trú tài khoản v1, nhưng mọi verb khác lại gọi
// thẳng `Dir()` nên tài khoản cũ hiện ra trong bảng mà chạy thì báo "không có".
// Dùng hàm này ở mọi chỗ nhận địa chỉ từ người dùng.
func ResolveDir(prov, account string) (string, bool) {
	d := Dir(prov, account)
	if _, err := os.Stat(d); err == nil {
		return d, true
	}
	if prov == "claude" {
		legacy := filepath.Join(paths.LegacyClaudeAccounts(), account)
		if _, err := os.Stat(legacy); err == nil {
			return legacy, true
		}
	}
	return d, false
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// LinkShared nối phần dùng chung từ BaseDir vào thư mục hồ sơ. Bỏ qua file
// riêng (token/danh tính) và mục đã tồn tại.
func LinkShared(a provider.Adapter, dir string) (int, error) {
	entries, err := os.ReadDir(a.BaseDir())
	if err != nil {
		return 0, err
	}
	priv := a.PrivateFiles()
	n := 0
	for _, e := range entries {
		if contains(priv, e.Name()) {
			continue
		}
		dst := filepath.Join(dir, e.Name())
		if _, err := os.Lstat(dst); err == nil {
			continue // đã có
		}
		src := filepath.Join(a.BaseDir(), e.Name())
		if e.IsDir() {
			if link.LinkDir(src, dst) == nil {
				n++
			}
		} else {
			if link.LinkFile(src, dst) == nil {
				n++
			}
		}
	}
	return n, nil
}

// Create tạo hồ sơ mới: thư mục + nối phần dùng chung + gieo whitelist.
func Create(a provider.Adapter, account string) (linked, seeded int, err error) {
	dir := Dir(a.Name(), account)
	if _, e := os.Stat(dir); e == nil {
		return 0, 0, fmt.Errorf("đã có tài khoản %s:%s", a.Name(), account)
	}
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return 0, 0, err
	}
	if linked, err = LinkShared(a, dir); err != nil {
		return linked, 0, err
	}
	// Không phải provider nào cũng có "file cấu hình gộp" để gieo whitelist khoá.
	// Claude có .claude.json; Codex thì thói quen máy nằm ở config.toml/AGENTS.md
	// và hai thứ đó nối link dùng chung được nguyên vẹn — không cần gieo gì.
	src := a.IdentitySource()
	if src == "" || len(a.SharedKeys()) == 0 {
		return linked, 0, nil
	}
	// Tên file đích lấy theo chính nguồn, đừng hardcode tên của một provider.
	seeded, err = jsonutil.Seed(src, filepath.Join(dir, filepath.Base(src)), a.SharedKeys())
	return linked, seeded, err
}

// Account là một dòng trong bảng liệt kê.
type Account struct {
	Provider string
	Name     string
	Dir      string
}

// List quét mọi hồ sơ ở ~/.ai-accounts/<provider>/<account>, và di trú tài
// khoản v1 ở ~/.claude-accounts/* (nhận là provider claude).
func List() ([]Account, error) {
	var out []Account
	seen := map[string]bool{}
	add := func(prov, name, dir string) {
		k := prov + "\x00" + name
		if seen[k] {
			return
		}
		seen[k] = true
		out = append(out, Account{prov, name, dir})
	}

	root := paths.AccountsRoot()
	if provs, err := os.ReadDir(root); err == nil {
		for _, p := range provs {
			// Bỏ qua thư mục nội bộ như .clones (bản sao dùng cho fleet) —
			// chúng không phải provider.
			if !p.IsDir() || strings.HasPrefix(p.Name(), ".") {
				continue
			}
			accs, _ := os.ReadDir(filepath.Join(root, p.Name()))
			for _, a := range accs {
				if a.IsDir() {
					add(p.Name(), a.Name(), filepath.Join(root, p.Name(), a.Name()))
				}
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// Di trú v1: ~/.claude-accounts/<tên> -> provider claude.
	if les, err := os.ReadDir(paths.LegacyClaudeAccounts()); err == nil {
		for _, a := range les {
			if a.IsDir() {
				add("claude", a.Name(), filepath.Join(paths.LegacyClaudeAccounts(), a.Name()))
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Remove xoá an toàn: gỡ TỪNG link trước, kiểm không còn link nào, rồi mới
// RemoveAll. Đây là chỗ nguy hiểm nhất — RemoveAll có thể xuyên junction xoá
// dữ liệu thật ở ~/.claude.
func Remove(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if isLink, _ := link.IsLink(p); isLink {
			if err := link.Unlink(p, e.IsDir()); err != nil {
				return fmt.Errorf("không gỡ được link %s: %w", p, err)
			}
		}
	}
	remain, err := countLinks(dir)
	if err != nil {
		return err
	}
	if remain > 0 {
		return fmt.Errorf("còn %d link chưa gỡ, không dám xoá đệ quy: %s", remain, dir)
	}
	return os.RemoveAll(dir)
}

func countLinks(dir string) (int, error) {
	n := 0
	err := filepath.Walk(dir, func(p string, _ os.FileInfo, err error) error {
		if err != nil || p == dir {
			return nil
		}
		if isLink, _ := link.IsLink(p); isLink {
			n++
		}
		return nil
	})
	return n, err
}

// Run đặt biến môi trường trong tiến trình CON rồi chạy CLI. dir rỗng =
// tài khoản gốc: XOÁ biến, KHÔNG trỏ vào ~/.claude (bẫy file lạc của v1).
func Run(a provider.Adapter, dir string, args []string) error {
	cmdPath, err := a.Command()
	if err != nil {
		return err
	}
	c := exec.Command(cmdPath, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	env := filterEnv(os.Environ(), a.EnvVar())
	if dir != "" {
		env = append(env, a.EnvVar()+"="+dir)
	}
	c.Env = env
	return c.Run()
}

func filterEnv(env []string, key string) []string {
	pref := key + "="
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, pref) {
			continue
		}
		out = append(out, e)
	}
	return out
}
