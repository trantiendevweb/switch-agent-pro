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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	Worktree string // git worktree riêng của phiên, rỗng = chạy thẳng thư mục hiện tại
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

// Open mở (và tạo nếu chưa có) cơ sở dữ liệu ở vị trí chuẩn.
func Open() (*DB, error) { return OpenAt(Path()) }

// OpenAt mở cơ sở dữ liệu ở một đường dẫn cụ thể (test dùng đường dẫn tạm).
func OpenAt(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	// busy_timeout: chờ thay vì lỗi ngay khi tiến trình khác đang ghi.
	// WAL: cho đọc và ghi song song.
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := migrate(db, path); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

// migrations chạy theo thứ tự; chỉ số + 1 là số phiên bản schema.
//
// THÊM BƯỚC MỚI THÌ CHỈ ĐƯỢC NỐI VÀO CUỐI. Sửa hay chèn giữa sẽ làm máy đã
// chạy bản cũ bỏ qua mất bước đó.
var migrations = []string{
	// v1 — bảng phiên
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
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_state ON sessions(state);`,

	// v2 — mỗi phiên có thể chạy trong git worktree riêng
	`ALTER TABLE sessions ADD COLUMN worktree TEXT;`,

	// v3 — lần chạy workflow. Đây là thứ cho phép RESUME: tiến trình chết giữa
	// chừng thì trạng thái từng bước vẫn còn trên đĩa.
	`CREATE TABLE IF NOT EXISTS flow_runs (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		flow    TEXT NOT NULL,
		dir     TEXT NOT NULL,
		vars    TEXT,
		state   TEXT NOT NULL,
		started TEXT NOT NULL,
		ended   TEXT
	);
	CREATE TABLE IF NOT EXISTS flow_steps (
		run_id  INTEGER NOT NULL,
		step_id TEXT    NOT NULL,
		state   TEXT    NOT NULL,
		attempt INTEGER NOT NULL DEFAULT 0,
		msg     TEXT,
		started TEXT,
		ended   TEXT,
		PRIMARY KEY (run_id, step_id)
	);
	CREATE INDEX IF NOT EXISTS idx_runs_state ON flow_runs(state);`,

	// v4 — kết quả của mỗi bước, để bước sau dùng lại ({{steps.x.output}})
	`ALTER TABLE flow_steps ADD COLUMN output TEXT;`,
}

// MaxStepOutput là trần kích thước kết quả lưu cho mỗi bước.
//
// Có trần vì output của agent có thể rất dài; để nguyên thì phình DB và làm
// chậm mọi truy vấn. Cắt phần ĐẦU, giữ phần CUỐI — kết luận thường nằm ở cuối.
const MaxStepOutput = 32 * 1024

// Trạng thái một lần chạy flow và từng bước.
const (
	RunRunning  = "running"
	RunWaiting  = "waiting_approval" // dừng chờ người duyệt
	RunDone     = "completed"
	RunFailed   = "failed"
	RunCanceled = "cancelled"

	StepPending = "pending"
	StepRunning = "running"
	StepDone    = "done"
	StepFailed  = "failed"
	StepSkipped = "skipped"
	StepWaiting = "waiting" // bước approve đang chờ người quyết
)

// Run là một lần chạy workflow.
type Run struct {
	ID      int64
	Flow    string
	Dir     string
	Vars    string // JSON
	State   string
	Started time.Time
	Ended   string
}

// StepRun là trạng thái một bước trong một lần chạy.
type StepRun struct {
	RunID   int64
	StepID  string
	State   string
	Attempt int
	Msg     string
	Output  string
}

// CreateRun mở một lần chạy mới.
func (d *DB) CreateRun(flowName, dir, varsJSON string) (int64, error) {
	res, err := d.db.Exec(
		`INSERT INTO flow_runs(flow,dir,vars,state,started) VALUES(?,?,?,?,?)`,
		flowName, dir, varsJSON, RunRunning, time.Now().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetRun đọc một lần chạy.
func (d *DB) GetRun(id int64) (Run, error) {
	var r Run
	var started string
	var ended, vars sql.NullString
	err := d.db.QueryRow(
		`SELECT id,flow,dir,COALESCE(vars,''),state,started,ended FROM flow_runs WHERE id=?`, id).
		Scan(&r.ID, &r.Flow, &r.Dir, &vars, &r.State, &started, &ended)
	if err != nil {
		return r, err
	}
	r.Vars = vars.String
	r.Ended = ended.String
	r.Started, _ = time.Parse(time.RFC3339, started)
	return r, nil
}

// ListRuns liệt kê các lần chạy, mới nhất trước.
func (d *DB) ListRuns(limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.db.Query(
		`SELECT id,flow,dir,COALESCE(vars,''),state,started,COALESCE(ended,'')
		   FROM flow_runs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		var r Run
		var started string
		if err := rows.Scan(&r.ID, &r.Flow, &r.Dir, &r.Vars, &r.State, &started, &r.Ended); err != nil {
			return nil, err
		}
		r.Started, _ = time.Parse(time.RFC3339, started)
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetRunState đổi trạng thái một lần chạy; kết thúc thì ghi luôn thời điểm.
func (d *DB) SetRunState(id int64, state string) error {
	if state == RunRunning || state == RunWaiting {
		_, err := d.db.Exec(`UPDATE flow_runs SET state=? WHERE id=?`, state, id)
		return err
	}
	_, err := d.db.Exec(`UPDATE flow_runs SET state=?, ended=? WHERE id=?`,
		state, time.Now().Format(time.RFC3339), id)
	return err
}

// SetStep ghi trạng thái một bước (tạo mới nếu chưa có).
func (d *DB) SetStep(runID int64, stepID, state, msg string, attempt int) error {
	now := time.Now().Format(time.RFC3339)
	_, err := d.db.Exec(`
		INSERT INTO flow_steps(run_id,step_id,state,attempt,msg,started,ended)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(run_id,step_id) DO UPDATE SET
			state=excluded.state, attempt=excluded.attempt, msg=excluded.msg,
			ended=CASE WHEN excluded.state IN ('done','failed','skipped') THEN excluded.ended ELSE flow_steps.ended END`,
		runID, stepID, state, attempt, msg, now,
		map[bool]string{true: now, false: ""}[state == StepDone || state == StepFailed || state == StepSkipped])
	return err
}

// SetStepOutput lưu kết quả một bước (đã cắt theo trần).
func (d *DB) SetStepOutput(runID int64, stepID, output string) error {
	if len(output) > MaxStepOutput {
		output = "…(đã cắt bớt phần đầu)…\n" + output[len(output)-MaxStepOutput:]
	}
	_, err := d.db.Exec(`UPDATE flow_steps SET output=? WHERE run_id=? AND step_id=?`,
		output, runID, stepID)
	return err
}

// Steps đọc trạng thái mọi bước của một lần chạy.
func (d *DB) Steps(runID int64) (map[string]StepRun, error) {
	rows, err := d.db.Query(
		`SELECT run_id,step_id,state,attempt,COALESCE(msg,''),COALESCE(output,'')
		   FROM flow_steps WHERE run_id=?`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]StepRun{}
	for rows.Next() {
		var s StepRun
		if err := rows.Scan(&s.RunID, &s.StepID, &s.State, &s.Attempt, &s.Msg, &s.Output); err != nil {
			return nil, err
		}
		out[s.StepID] = s
	}
	return out, rows.Err()
}

// migrate đưa schema lên phiên bản mới nhất, chạy trong transaction để không
// bao giờ để lại nửa vời.
func migrate(db *sql.DB, path string) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		return err
	}
	cur, err := schemaVersion(db)
	if err != nil {
		return err
	}

	// HẠ CẤP: file do bản sagent MỚI HƠN tạo ra. Đã đo hành vi cũ — không có
	// chốt này thì binary cũ mở bình thường, đọc được, GHI ĐƯỢC, và để nguyên
	// schema_version của bản mới. Ghi chéo phiên bản trong im lặng: bản cũ không
	// biết ràng buộc mà bản mới đặt ra, nên nó ghi ra những dòng hợp lệ với nó
	// và sai với bản mới. Hỏng kiểu đó chỉ lộ ra sau, ở chỗ khác.
	//
	// Thà không chạy. Đây là cùng một lựa chọn với "chưa đặt mật khẩu thì dash
	// từ chối mở cổng".
	if cur > len(migrations) {
		return fmt.Errorf("%s ở schema v%d, bản sagent này chỉ biết tới v%d — "+
			"nâng cấp sagent, hoặc khôi phục bản sao lưu cũ: sagent db restore <file>",
			path, cur, len(migrations))
	}

	// Sắp đổi schema của một file ĐÃ CÓ DỮ LIỆU: chụp ảnh trước. Migration chạy
	// trong transaction nên không để lại nửa vời, nhưng transaction không cứu
	// được một migration viết đúng cú pháp mà sai ý (ví dụ DROP nhầm cột) —
	// nó commit gọn gàng rồi dữ liệu vẫn đi.
	//
	// cur == 0 là file mới tinh, không có gì để mất, bỏ qua.
	if cur > 0 && cur < len(migrations) {
		bak := autoBackupPath(path, cur)
		if err := snapshot(db, bak); err != nil {
			return fmt.Errorf("không sao lưu được trước khi nâng schema v%d→v%d, KHÔNG nâng: %w",
				cur, len(migrations), err)
		}
		fmt.Fprintf(os.Stderr, "  ℹ nâng schema v%d → v%d, đã sao lưu: %s\n", cur, len(migrations), bak)
	}

	for i := cur; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration v%d hỏng: %w", i+1, err)
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('schema_version',?)`,
			strconv.Itoa(i+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// AddSession ghi lại một phiên vừa khởi chạy.
func (d *DB) AddSession(s Session) (int64, error) {
	res, err := d.db.Exec(
		`INSERT INTO sessions(provider,account,clone,dir,pid,log,worktree,started,state)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		s.Provider, s.Account, s.Clone, s.Dir, s.PID, s.Log, s.Worktree,
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
		`SELECT id,provider,account,clone,dir,pid,COALESCE(log,''),COALESCE(worktree,''),started,state
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
			&s.PID, &s.Log, &s.Worktree, &started, &s.State); err != nil {
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
