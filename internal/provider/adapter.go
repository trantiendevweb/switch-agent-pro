// Package provider định nghĩa giao diện adapter cho từng nhà cung cấp AI và
// một registry để đăng ký. Lõi không có nhánh "if provider == claude" nào —
// mọi khác biệt nằm trong adapter.
package provider

// Check là một phép đo trong bộ "đã đo" của adapter.
type Check struct {
	Name   string
	OK     bool
	Detail string
}

// Adapter khai báo nguyên lý cốt lõi (thư mục config biệt lập + file riêng/chung)
// cho một nhà cung cấp.
type Adapter interface {
	Name() string   // "claude", "codex", ...
	EnvVar() string // biến trỏ CLI vào thư mục config biệt lập

	// Command trả về đường dẫn tới CLI để chạy.
	Command() (string, error)

	// PrivateFiles là các file KHÔNG dùng chung (token + danh tính).
	PrivateFiles() []string

	// SharedKeys là whitelist khoá "thói quen máy" copy sang, nếu provider dùng
	// một file config gộp (như .claude.json). Trả nil nếu không có.
	SharedKeys() []string

	// BaseDir là nguồn để nối phần dùng chung (vd ~/.claude).
	BaseDir() string
	// IdentitySource là file config gốc để gieo whitelist (vd ~/.claude.json).
	IdentitySource() string

	Identity(configDir string) string // email/định danh để hiển thị; "" nếu chưa đăng nhập
	HasToken(configDir string) bool

	// Verify chứng minh trên MÁY NÀY việc tách thư mục là tách thật.
	Verify() []Check
}

var registry = map[string]Adapter{}

// Register đăng ký một adapter (gọi trong init của từng provider).
func Register(a Adapter) { registry[a.Name()] = a }

// Get lấy adapter theo tên.
func Get(name string) (Adapter, bool) { a, ok := registry[name]; return a, ok }

// Names liệt kê tên các adapter đã đăng ký.
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	return out
}
