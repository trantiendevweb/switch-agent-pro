package flow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/trantiendevweb/switch-agent-pro/internal/config"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// TargetFile là file sẽ được ghi khi lưu flow: ưu tiên `.sagent/flows.toml` của
// dự án, không có dự án thì rơi về file chung của máy.
//
// Cố ý ghi vào ĐÚNG file mà người dùng sửa tay được: `flows.toml` vẫn là nguồn
// sự thật, trình soạn thảo chỉ là một cách sửa nó. Nhờ vậy flow dựng bằng giao
// diện và flow viết tay chạy y hệt nhau.
func TargetFile(dir string) string {
	if p := config.FindProjectFile(dir); p != "" {
		return filepath.Join(filepath.Dir(p), "flows.toml")
	}
	return filepath.Join(paths.AccountsRoot(), "flows.toml")
}

// readFile đọc flows.toml hiện có (rỗng nếu chưa có).
func readFile(path string) (File, error) {
	f := File{Version: 1, Flows: map[string]Flow{}}
	if _, err := os.Stat(path); err != nil {
		return f, nil
	}
	if _, err := toml.DecodeFile(path, &f); err != nil {
		return f, fmt.Errorf("%s: %w", path, err)
	}
	if f.Flows == nil {
		f.Flows = map[string]Flow{}
	}
	if f.Version == 0 {
		f.Version = 1
	}
	return f, nil
}

// writeFile ghi nguyên tử: file tạm rồi đổi tên, không để lại bản nửa vời.
func writeFile(path string, f File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	fh, err := os.Create(tmp)
	if err != nil {
		return err
	}
	header := "# File này do `sagent` và trình soạn thảo workflow cùng dùng.\n" +
		"# Sửa tay thoải mái — bảng vẽ đọc lại được, chỉ mất vị trí x/y nếu bạn xoá.\n\n"
	if _, err := fh.WriteString(header); err != nil {
		fh.Close()
		return err
	}
	enc := toml.NewEncoder(fh)
	if err := enc.Encode(f); err != nil {
		fh.Close()
		return err
	}
	if err := fh.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Save ghi (hoặc ghi đè) một flow vào flows.toml, giữ nguyên các flow khác.
// Trả về đường dẫn file đã ghi.
func Save(dir string, f Flow) (string, error) {
	name := f.Name
	if name == "" {
		return "", fmt.Errorf("flow phải có tên")
	}
	if !idRe.MatchString(name) {
		return "", fmt.Errorf("tên flow chỉ dùng chữ thường, số, - và _")
	}
	// Không cho lưu thứ hỏng: sai DAG mà nằm trong file thì lần sau mở ra là kẹt.
	for _, p := range Validate(f) {
		if !p.Warn {
			return "", fmt.Errorf("%s: %s", p.Step, p.Msg)
		}
	}

	path := TargetFile(dir)
	file, err := readFile(path)
	if err != nil {
		return "", err
	}
	f.Name = "" // tên là khoá của map, đừng lặp lại trong thân
	file.Flows[name] = f
	return path, writeFile(path, file)
}

// Delete xoá một flow khỏi flows.toml. Flow mẫu dựng sẵn thì không xoá được
// (nó nằm trong mã nguồn), nhưng bản đè trong file thì xoá được.
func Delete(dir, name string) (string, error) {
	path := TargetFile(dir)
	file, err := readFile(path)
	if err != nil {
		return "", err
	}
	if _, ok := file.Flows[name]; !ok {
		return "", fmt.Errorf("flows.toml không có flow %q", name)
	}
	delete(file.Flows, name)
	return path, writeFile(path, file)
}

// Import nạp một file flows.toml rời vào file của dự án (dùng cho CLI).
func Import(dir, src string) (string, []string, error) {
	var in File
	if _, err := toml.DecodeFile(src, &in); err != nil {
		return "", nil, fmt.Errorf("%s: %w", src, err)
	}
	if in.Version != 0 && in.Version != 1 {
		return "", nil, fmt.Errorf("%s: version = %d, công cụ này chỉ hiểu 1", src, in.Version)
	}
	path := TargetFile(dir)
	file, err := readFile(path)
	if err != nil {
		return "", nil, err
	}
	var names []string
	for name, f := range in.Flows {
		f.Name = name
		for _, p := range Validate(f) {
			if !p.Warn {
				return "", nil, fmt.Errorf("flow %q lỗi ở %s: %s", name, p.Step, p.Msg)
			}
		}
		f.Name = ""
		file.Flows[name] = f
		names = append(names, name)
	}
	if len(names) == 0 {
		return "", nil, fmt.Errorf("%s không có flow nào", src)
	}
	return path, names, writeFile(path, file)
}

// IsBuiltin cho biết tên này có phải flow mẫu dựng sẵn không.
func IsBuiltin(name string) bool {
	_, ok := Builtin()[name]
	return ok
}

// SanitizeID biến chuỗi bất kỳ thành id hợp lệ (dùng khi thêm node từ bảng vẽ).
func SanitizeID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-_")
	if out == "" || !idRe.MatchString(out) {
		return "buoc"
	}
	return out
}
