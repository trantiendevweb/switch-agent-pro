// Package store là nơi giữ trạng thái bền của công cụ: SQLite ở
// ~/.ai-accounts/state.db.
//
// Vì sao SQLite chứ không phải một file JSON: nhiều tiến trình `sagent` có thể
// cùng ghi (bật fleet ở terminal này trong khi `status` chạy ở terminal kia).
// JSON thì phải tự lo khoá và ghi nguyên tử; SQLite lo sẵn bằng transaction +
// WAL. Dùng driver modernc.org/sqlite (THUẦN Go, không cần cgo) để vẫn giữ
// được "một binary, build thẳng cho Windows và Linux".
package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
	"github.com/trantiendevweb/switch-agent-pro/internal/process"
)

// Trạng thái một phiên.
const (
	StateRunning = "running"
	StateStopped = "stopped"
	StateLost    = "lost" // tiến trình biến mất mà không qua `stop`
)

// Session là một phiên agent đang/đã chạy.
type Session struct {
	ID       int64
	Provider string
	Account  string
	Clone    int // 0 = chạy thẳng hồ sơ gốc, >0 = bản clone thứ n
	Dir      string
	PID      int
	Log      string
	Started  time.Time
	State    string
}

// Addr trả về địa chỉ hiển thị, ví dụ "claude:phu#2".
func (s Session) Addr() string {
	a := s.Provider + ":" + s.Account
	if s.Clone > 0 {
		a += "#" + itoa(s.Clone)
	}
	return a
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// DB bọc *sql.DB để lõi không phải biết SQL.
type DB struct{ db *sql.DB }

// Path là đường dẫn file cơ sở dữ liệu.
func Path() string { return filepath.Join(paths.AccountsRoot(), "state.db") }

// Open mở (và tạo nếu chưa có) cơ sở dữ liệu, chạy migration.
func Open() (*DB, error) {
	if err := os.MkdirAll(paths.AccountsRoot(), 0o755); err != nil {
		return nil, err
	}
	// busy_timeout: chờ thay vì lỗi ngay khi tiến trình khác đang ghi.
	// WAL: cho đọc và ghi song song.
	dsn := "file:" + Path() + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

// migrate tạo schema. Mỗi bước chỉ chạy một lần, ghi lại ở bảng meta.
func migrate(db *sql.DB) error {
	steps := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT    NOT NULL,
			account  TEXT    NOT NULL,
			clone    INTEGER NOT NULL DEFAULT 0,
			dir      TEXT    NOT NULL,
			pid      INTEGER NOT NULL,
			log      TEXT,
			started  TEXT    NOT NULL,
			state    TEXT    NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_state ON sessions(state)`,
		`CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
	}
	for _, s := range steps {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	_, err := db.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('schema_version','1')`)
	return err
}

// AddSession ghi lại một phiên vừa khởi chạy.
func (d *DB) AddSession(s Session) (int64, error) {
	res, err := d.db.Exec(
		`INSERT INTO sessions(provider,account,clone,dir,pid,log,started,state)
		 VALUES(?,?,?,?,?,?,?,?)`,
		s.Provider, s.Account, s.Clone, s.Dir, s.PID, s.Log,
		time.Now().Format(time.RFC3339), StateRunning)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Running trả về các phiên đang chạy, sau khi đối chiếu PID thật.
//
// PID không phải nguồn sự thật — nó chỉ là thuộc tính runtime. Phiên nào có
// PID đã chết thì đánh dấu `lost` ngay tại đây, để bảng không bao giờ báo sống
// một thứ đã chết.
func (d *DB) Running() ([]Session, error) {
	rows, err := d.db.Query(
		`SELECT id,provider,account,clone,dir,pid,COALESCE(log,''),started,state
		   FROM sessions WHERE state=? ORDER BY id`, StateRunning)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Session
	var dead []int64
	for rows.Next() {
		var s Session
		var started string
		if err := rows.Scan(&s.ID, &s.Provider, &s.Account, &s.Clone, &s.Dir,
			&s.PID, &s.Log, &started, &s.State); err != nil {
			return nil, err
		}
		s.Started, _ = time.Parse(time.RFC3339, started)
		if process.IsAlive(s.PID) {
			out = append(out, s)
		} else {
			dead = append(dead, s.ID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range dead {
		_, _ = d.db.Exec(`UPDATE sessions SET state=? WHERE id=?`, StateLost, id)
	}
	return out, nil
}

// SetState đổi trạng thái một phiên.
func (d *DB) SetState(id int64, state string) error {
	_, err := d.db.Exec(`UPDATE sessions SET state=? WHERE id=?`, state, id)
	return err
}
