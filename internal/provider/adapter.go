// Package provider định nghĩa giao diện adapter cho từng nhà cung cấp AI và
// một registry để đăng ký. Lõi không có nhánh "if provider == claude" nào —
// mọi khác biệt nằm trong adapter.
package provider

import "time"

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

	// HeadlessArgs là đối số để chạy CLI KHÔNG tương tác với một prompt.
	//
	// Mỗi nhà cung cấp một kiểu, và khác nhau thật chứ không chỉ khác tên cờ:
	//   claude -p "<prompt>"
	//   codex exec "<prompt>"
	// Vì vậy nó phải nằm ở adapter. Trước đây lõi hardcode `-p` — tức là Claude
	// rò vào code dùng chung, và `fleet codex:*` sẽ chạy sai mà không ai biết.
	HeadlessArgs(prompt string) []string

	// ArgsTuDuyetQuyen trả về cờ để agent TỰ DUYỆT MỌI TOOL ở chế độ headless,
	// kèm daDo = đã đo được cách làm của provider này hay chưa.
	//
	// Ba trạng thái KHÁC NHAU, đừng gộp:
	//   (cờ, true)  — có rào, và đây là cách mở.
	//   (nil, true)  — ĐÃ ĐO và provider KHÔNG có rào nào; cờ là thừa. Grok thuộc
	//                  nhóm này: `grok --help` không có approval/sandbox, nó chạy
	//                  tool tự do theo thiết kế. Đây là chuyện AN NINH cần nói ra,
	//                  không phải chuyện "không cần làm gì".
	//   (nil, false) — CHƯA ĐO. Người gọi phải báo lỗi, không được lặng lẽ chạy
	//                  tiếp: người dùng bật cờ là có chủ ý, nuốt mất ý đó rồi báo
	//                  thành công là dối.
	//
	// Nguy hiểm thật: agent duyệt cả xoá file và chạy lệnh tuỳ ý trong worktree
	// của repo thật. Mặc định TẮT, chỉ bật theo từng bước trong flows.toml.
	ArgsTuDuyetQuyen() (args []string, daDo bool)

	// ArgsThuMuc là đối số khai TƯỜNG MINH thư mục làm việc cho phiên headless.
	//
	// Cần vì fleet chạy mỗi agent trong một git worktree, mà ở worktree thì `.git`
	// là FILE con trỏ chứ không phải thư mục — Antigravity dò workspace hụt. Đo:
	// cùng lệnh cùng cờ, chạy ở thư mục repo đúng 3/3, chạy ở worktree chỉ 1/3
	// (hai lượt kia trả "chưa có repository nào được mở"). Thêm `--add-dir` thì
	// 4/4 đúng. Trả nil = provider không cần / chưa đo.
	ArgsThuMuc(dir string) []string

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

	// TokenExpiry cho biết token còn hạn tới bao giờ (ok=false nếu không đọc được).
	//
	// CHỈ đọc dấu thời gian, không bao giờ trả về hay ghi log giá trị token.
	// Dùng để cảnh báo trước khi bật hạm đội chạy dài: đã đo được Claude hết hạn
	// sau ~7,5 giờ, nên một đội chạy qua đêm chắc chắn vượt mốc refresh.
	TokenExpiry(configDir string) (time.Time, bool)

	// TachDuocTaiKhoan: provider này có chạy được NHIỀU tài khoản trên cùng một
	// máy không.
	//
	// Nằm trong interface chứ không phải một cái bảng ở đâu đó, vì đây là tính
	// chất ĐO ĐƯỢC của từng provider và lõi không được có nhánh
	// `if provider == "antigravity"`.
	//
	// false = token nằm ở chỗ dùng chung toàn máy (Antigravity giữ nó trong
	// Windows Credential Manager dưới khoá tên cố định). Với provider như vậy,
	// `fleet --copies N` là lời hứa hão: N tiến trình sẽ giành nhau đúng một
	// danh tính. Thà từ chối và nói rõ.
	TachDuocTaiKhoan() bool

	// Version là chuỗi phiên bản của CLI bên dưới.
	//
	// Nằm trong interface chứ không phải một helper dùng chung, vì cách hỏi có
	// thể khác nhau (`--version`, `version`, `-v`) và lõi thì không được có
	// nhánh "if provider == ...". Đây cũng là dữ liệu để phát hiện provider
	// drift: mọi phép đo trong docs/DO-LUONG.md đều gắn với MỘT phiên bản CLI,
	// CLI đổi thì phép đo hết hiệu lực.
	Version() (string, error)

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
