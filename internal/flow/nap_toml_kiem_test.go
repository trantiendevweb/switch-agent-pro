package flow

import (
	"os"
	"path/filepath"
	"testing"
)

// Khai `tu_duyet_quyen = true` trong flows.toml phải NẠP ĐƯỢC. Tên thẻ toml sai
// một chữ thì BurntSushi bỏ qua im lặng, cờ về false, agent chạy không quyền —
// và flow vẫn báo xong. Đúng kiểu hỏng lặng lẽ đã làm lần chạy #8 báo completed.
func TestNapDuocCoTuDuyetQuyen(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".sagent"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := `version = 1
[flow]
  [flow.thu]
    [[flow.thu.step]]
      id = "a"
      type = "agent"
      tu_duyet_quyen = true
      prompt = "việc"
`
	if err := os.WriteFile(filepath.Join(dir, ".sagent", "flows.toml"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	// flows.toml chỉ được nhận khi có project.toml bên cạnh (xem flow.Paths).
	if err := os.WriteFile(filepath.Join(dir, ".sagent", "project.toml"), []byte("version = 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	flows, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	f, ok := flows["thu"]
	if !ok {
		t.Fatal("không nạp được flow")
	}
	if !f.Steps[0].TuDuyetQuyen {
		t.Fatal("khai tu_duyet_quyen = true trong TOML mà nạp ra FALSE — cờ rơi mất, agent chạy không quyền")
	}
}
