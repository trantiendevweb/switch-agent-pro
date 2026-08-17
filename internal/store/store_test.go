package store

import (
	"os"
	"path/filepath"
	"testing"
)

func open(t *testing.T) *DB {
	t.Helper()
	db, err := OpenAt(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// Migration phải chạy được nhiều lần mà không hỏng (mở lại DB cũ là chuyện
// thường xuyên), và phải lên đúng phiên bản mới nhất.
func TestMigrateIdempotent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.db")
	for i := 0; i < 3; i++ {
		db, err := OpenAt(p)
		if err != nil {
			t.Fatalf("lần mở thứ %d lỗi: %v", i+1, err)
		}
		var v string
		if err := db.db.QueryRow(`SELECT value FROM meta WHERE key='schema_version'`).Scan(&v); err != nil {
			t.Fatal(err)
		}
		want := itoa(len(migrations))
		if v != want {
			t.Fatalf("schema_version = %s, muốn %s", v, want)
		}
		db.Close()
	}
}

// Cột worktree (thêm ở v2) phải dùng được — bắt lỗi nếu ai đó sửa migration cũ
// thay vì nối bước mới vào cuối.
func TestWorktreeColumnRoundTrip(t *testing.T) {
	db := open(t)
	if _, err := db.AddSession(Session{
		Provider: "claude", Account: "phu", Clone: 1,
		Dir: "d", PID: os.Getpid(), Worktree: "/tmp/wt",
	}); err != nil {
		t.Fatal(err)
	}
	list, err := db.Running()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("muốn 1 phiên, được %d", len(list))
	}
	if list[0].Worktree != "/tmp/wt" {
		t.Fatalf("worktree = %q", list[0].Worktree)
	}
}

// PID không phải nguồn sự thật: phiên có PID đã chết phải tự bị đánh dấu `lost`
// chứ không được báo là đang chạy.
func TestRunningReapsDeadPID(t *testing.T) {
	db := open(t)
	// PID còn sống: chính tiến trình test.
	if _, err := db.AddSession(Session{Provider: "claude", Account: "song", PID: os.Getpid(), Dir: "d"}); err != nil {
		t.Fatal(err)
	}
	// PID chắc chắn không tồn tại.
	if _, err := db.AddSession(Session{Provider: "claude", Account: "chet", PID: 0x7FFFFFF0, Dir: "d"}); err != nil {
		t.Fatal(err)
	}

	list, err := db.Running()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Account != "song" {
		t.Fatalf("chỉ phiên còn sống mới được liệt kê, được %+v", list)
	}
	// Gọi lần hai: phiên chết đã chuyển trạng thái nên không hiện lại.
	again, err := db.Running()
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 1 {
		t.Fatalf("lần hai muốn 1, được %d", len(again))
	}
	var state string
	if err := db.db.QueryRow(`SELECT state FROM sessions WHERE account='chet'`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != StateLost {
		t.Fatalf("phiên chết phải là %s, đang là %s", StateLost, state)
	}
}

func TestSetState(t *testing.T) {
	db := open(t)
	id, err := db.AddSession(Session{Provider: "claude", Account: "a", PID: os.Getpid(), Dir: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetState(id, StateStopped); err != nil {
		t.Fatal(err)
	}
	list, _ := db.Running()
	if len(list) != 0 {
		t.Fatalf("phiên đã dừng không được nằm trong danh sách chạy: %+v", list)
	}
}

func TestSessionAddr(t *testing.T) {
	if got := (Session{Provider: "claude", Account: "phu"}).Addr(); got != "claude:phu" {
		t.Fatalf("Addr() = %s", got)
	}
	if got := (Session{Provider: "claude", Account: "phu", Clone: 12}).Addr(); got != "claude:phu#12" {
		t.Fatalf("Addr() = %s", got)
	}
}
