package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Refresh xảy ra trong bản clone thì phải mang được về hồ sơ gốc — nếu không,
// lần chạy sau vẫn dùng token cũ đã hết hạn.
func TestMangTokenMoiTuCloneVeGoc(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := fakeAdapter{base: fakeBase, hasToken: true}

	dirs, err := Clone(a, "phu", 2)
	if err != nil {
		t.Fatal(err)
	}

	// Giả lập: bản clone thứ 2 tự refresh (ghi token mới, dấu thời gian mới hơn)
	newTok := `{"t":"TOKEN-DA-REFRESH"}`
	p := filepath.Join(dirs[1], ".credentials.json")
	if err := os.WriteFile(p, []byte(newTok), 0o600); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Minute)
	if err := os.Chtimes(p, future, future); err != nil {
		t.Fatal(err)
	}

	name, err := SyncBackTokens(a, "phu")
	if err != nil {
		t.Fatal(err)
	}
	if name != ".credentials.json" {
		t.Fatalf("phải mang file token về, được %q", name)
	}

	base, _ := ResolveDir(a.Name(), "phu")
	got, err := os.ReadFile(filepath.Join(base, ".credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != newTok {
		t.Fatalf("hồ sơ gốc chưa nhận token mới: %s", got)
	}
	// phải có bản sao lưu, vì đây là token — hỏng là phải đăng nhập lại
	found := false
	entries, _ := os.ReadDir(base)
	for _, e := range entries {
		if len(e.Name()) > 20 && e.Name()[:24] == ".credentials.json.bak-20" {
			found = true
		}
	}
	if !found {
		t.Fatal("đè token mà không sao lưu bản cũ")
	}
}

// Không có gì mới hơn thì KHÔNG được đụng vào hồ sơ gốc.
func TestKhongCoGiMoiThiKhongDung(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := fakeAdapter{base: fakeBase, hasToken: true}
	if _, err := Clone(a, "phu", 2); err != nil {
		t.Fatal(err)
	}
	base, _ := ResolveDir(a.Name(), "phu")
	before, _ := os.ReadFile(filepath.Join(base, ".credentials.json"))

	name, err := SyncBackTokens(a, "phu")
	if err != nil {
		t.Fatal(err)
	}
	if name != "" {
		t.Fatalf("không có gì mới mà vẫn mang về %q", name)
	}
	after, _ := os.ReadFile(filepath.Join(base, ".credentials.json"))
	if string(before) != string(after) {
		t.Fatal("token gốc bị thay đổi dù không có bản clone nào mới hơn")
	}
}
