package fleet

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/process"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// TestHelperProcess KHÔNG phải test thật — nó đóng vai CLI của agent khi được
// fleet spawn ra. Đây là mẫu chuẩn của Go để có một tiến trình con thật mà
// không cần cài thêm gì trên máy chạy test.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("SAGENT_FAKE_AGENT") != "1" {
		return // chạy trong bộ test bình thường: không làm gì cả
	}
	// Sống đủ lâu để bài test kịp thấy "đang chạy" rồi dừng.
	time.Sleep(20 * time.Second)
}

// fakeAgent trỏ Command() vào chính test binary.
type fakeAgent struct{ base string }

func (fakeAgent) Name() string             { return "fake" }
func (fakeAgent) EnvVar() string           { return "FAKE_CONFIG_DIR" }
func (fakeAgent) Command() (string, error) { return os.Executable() }
func (fakeAgent) HeadlessArgs(p string) []string { return []string{"-p", p} }
func (fakeAgent) PrivateFiles() []string   { return []string{".credentials.json", ".claude.json"} }
func (fakeAgent) SharedKeys() []string     { return []string{"projects"} }
func (f fakeAgent) BaseDir() string        { return f.base }
func (fakeAgent) IdentitySource() string   { return "" }
func (fakeAgent) Identity(string) string   { return "" }
func (fakeAgent) HasToken(string) bool     { return true }
func (fakeAgent) TokenExpiry(string) (time.Time, bool) { return time.Time{}, false }
func (fakeAgent) Verify() []provider.Check { return nil }

// setup dựng HOME giả + hồ sơ gốc có token + một store tạm.
func setup(t *testing.T) (*store.DB, *events.Bus, fakeAgent) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SAGENT_FAKE_AGENT", "1") // để tiến trình con biết mình là agent giả

	base := filepath.Join(home, "fakebase")
	if err := os.MkdirAll(filepath.Join(base, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	prof := filepath.Join(home, ".ai-accounts", "fake", "phu")
	if err := os.MkdirAll(prof, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{".credentials.json", ".claude.json"} {
		if err := os.WriteFile(filepath.Join(prof, n), []byte(`{}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	db, err := store.OpenAt(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	// Đăng ký SAU t.TempDir() nên chạy TRƯỚC khi thư mục tạm bị xoá (cleanup
	// chạy ngược thứ tự đăng ký). Cần vậy vì trên Windows tiến trình con còn
	// giữ fleet.log thì không xoá được thư mục.
	t.Cleanup(func() {
		stopAll(t, db)
		db.Close()
	})
	bus := events.NewBus()
	t.Cleanup(bus.Close)
	return db, bus, fakeAgent{base: base}
}

// stopAll giết mọi phiên còn sống VÀ ĐỢI chúng chết hẳn — không đợi thì
// Windows còn giữ file log và thư mục tạm không xoá được.
func stopAll(t *testing.T, db *store.DB) {
	t.Helper()
	list, _ := db.Running()
	for _, s := range list {
		_ = process.Kill(s.PID)
		_ = db.SetState(s.ID, store.StateStopped)
	}
	deadline := time.Now().Add(15 * time.Second)
	for _, s := range list {
		for time.Now().Before(deadline) && process.IsAlive(s.PID) {
			time.Sleep(100 * time.Millisecond)
		}
		if process.IsAlive(s.PID) {
			t.Errorf("PID %d không chịu chết sau khi Kill — sẽ để lại tiến trình mồ côi", s.PID)
		}
	}
}

func TestFanOutStartsAndRecordsSessions(t *testing.T) {
	db, bus, a := setup(t)

	_, err := FanOut(db, bus, a, "phu", Opts{Copies: 3}, []string{"-test.run=TestHelperProcess"})
	if err != nil {
		t.Fatal(err)
	}

	list, err := db.Running()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("muốn 3 phiên đang chạy, được %d", len(list))
	}

	seen := map[int]bool{}
	for _, s := range list {
		if !process.IsAlive(s.PID) {
			t.Fatalf("phiên #%d báo chạy nhưng PID %d đã chết", s.ID, s.PID)
		}
		if seen[s.Clone] {
			t.Fatalf("trùng số bản clone %d — hai phiên dùng chung config dir", s.Clone)
		}
		seen[s.Clone] = true
		// Mỗi phiên PHẢI có config dir riêng, nếu không sẽ đua ghi .claude.json.
		if _, err := os.Stat(filepath.Join(s.Dir, ".credentials.json")); err != nil {
			t.Fatalf("phiên #%d thiếu credential riêng: %v", s.ID, err)
		}
		if _, err := os.Stat(s.Log); err != nil {
			t.Fatalf("phiên #%d không có file log: %v", s.ID, err)
		}
	}
}

// Dừng phiên thì `status` phải phản ánh ngay, không được báo sống thứ đã chết.
func TestStoppedSessionDisappearsFromRunning(t *testing.T) {
	db, bus, a := setup(t)

	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 2}, []string{"-test.run=TestHelperProcess"}); err != nil {
		t.Fatal(err)
	}
	list, _ := db.Running()
	if len(list) != 2 {
		t.Fatalf("muốn 2 phiên, được %d", len(list))
	}

	if err := process.Kill(list[0].PID); err != nil {
		t.Fatalf("không giết được tiến trình: %v", err)
	}
	// Đợi hệ điều hành dọn xong.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && process.IsAlive(list[0].PID) {
		time.Sleep(200 * time.Millisecond)
	}

	after, err := db.Running()
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 {
		t.Fatalf("phiên đã bị giết vẫn còn trong danh sách chạy: %d phiên", len(after))
	}
}

// Thiếu lệnh headless thì phải báo lỗi RÕ RÀNG chứ không bật một đống phiên vô dụng.
func TestFanOutRefusesWithoutCommand(t *testing.T) {
	db, bus, a := setup(t)
	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 2}, nil); err == nil {
		t.Fatal("thiếu lệnh mà vẫn chạy")
	}
	list, _ := db.Running()
	if len(list) != 0 {
		t.Fatalf("không được bật phiên nào khi thiếu lệnh, có %d", len(list))
	}
}

// --worktree ở nơi không phải git repo: phải chết SỚM, trước khi bật phiên nào.
func TestWorktreeRefusesOutsideGitRepo(t *testing.T) {
	db, bus, a := setup(t)
	t.Chdir(t.TempDir()) // thư mục trống, không phải repo

	_, err := FanOut(db, bus, a, "phu", Opts{Copies: 2, Worktree: true}, []string{"-test.run=TestHelperProcess"})
	if err == nil {
		t.Fatal("không phải git repo mà vẫn chạy --worktree")
	}
	list, _ := db.Running()
	if len(list) != 0 {
		t.Fatalf("phải chết trước khi bật phiên nào, mà đã bật %d", len(list))
	}
}
