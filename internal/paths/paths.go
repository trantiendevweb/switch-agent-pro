// Package paths gom các đường dẫn gốc của công cụ, tách riêng để dễ test.
package paths

import (
	"os"
	"path/filepath"
)

// Home trả về thư mục người dùng ($HOME / %USERPROFILE%).
func Home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// AccountsRoot là kho hồ sơ mới, chung cho mọi provider:
//
//	~/.ai-accounts/<provider>/<account>/
func AccountsRoot() string { return filepath.Join(Home(), ".ai-accounts") }

// LegacyClaudeAccounts là kho v1 (chỉ Claude). Dùng để di trú.
func LegacyClaudeAccounts() string { return filepath.Join(Home(), ".claude-accounts") }
