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
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/trantiendevweb/switch-agent-pro/internal/acl"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
	"github.com/trantiendevweb/switch-agent-pro/internal/process"
)

// Trạng thái một phiên.
//
// Ba trạng thái dưới `StateLost` là `lost` ĐÃ BIẾT VÌ SAO. Chúng không phải một
// tầng phân loại riêng: phiên nào cũng đi qua đúng một điểm quyết định
// (provider.PhanLoaiChet, cắm vào sổ qua DungPhanLoaiChet), và điểm đó chỉ nói
// tên trạng thái khi ĐO ĐƯỢC. Không đo được thì phiên ở lại `lost`.
const (
	StateRunning = "running"
	StateStopped = "stopped"
	StateLost    = "lost" // tiến trình biến mất mà không qua `stop`

	// StateXong: lượt chạy KẾT THÚC BÌNH THƯỜNG và đọc được kết quả.
	//
	// Thiếu tên này là một lỗ hổng đo được ngày 20/08/2026: bảng `sagent status`
	// hiện 20 phiên liền "chết, chưa rõ vì sao", và ít nhất một trong số đó
	// (#157) đã chạy XONG XUÔI — agent trả lời đúng, NDJSON có dòng result,
	// không lỗi gì. `PhanLoaiChet` chỉ đặt tên cho các kiểu HỎNG, nên lượt chạy
	// thành công rơi vào cùng một sọt với phiên chết bí ẩn.
	//
	// Hậu quả không chỉ là chữ xấu: người vận hành nhìn bảng toàn "chưa rõ vì
	// sao" thì hoặc là hoảng vô cớ, hoặc quen mắt rồi thôi đọc — và lần có phiên
	// chết thật thì nó lẫn vào đám đông.
	StateXong = "done"

	// Ba chuỗi này phải TRÙNG provider.ChetHanMuc/ChetChanQuyen/ChetLoiAPI —
	// có test ghim, xem internal/api/phientrangthai_test.go.
	StateHanMuc = "rate_limited" // hết hạn mức; han_muc_den_lai nói lúc nào cấp lại
	StateChan   = "blocked"      // mọi tool bị từ chối quyền
	StateHong   = "failed"       // nhà cung cấp trả mã lỗi (api_error_status)
)

// TuKetThuc liệt kê mọi trạng thái nghĩa là "tiến trình tự kết thúc, không qua
// `stop`" — kể cả kết thúc TỐT ĐẸP.
//
// Dùng ở đúng những chỗ trước đây so bằng với StateLost. Thêm trạng thái mới mà
// quên chỗ nào thì phiên đó tàng hình khỏi `session.sweep`, tức đám tiến trình
// con của nó tiêu hạn mức mà không ai nhìn thấy.
//
// `done` NẰM TRONG danh sách này, dù nó không phải một kiểu chết: một lượt chạy
// xong xuôi vẫn có thể để lại tiến trình con, và con của phiên thành công tiêu
// hạn mức y hệt con của phiên hỏng. Đây là lý do hàm đổi tên từ
// `ChetBatThuong` — cái tên cũ sẽ khiến người đọc sau này tưởng bỏ `done` ra là
// đúng, rồi lặng lẽ mất một đường quét.
func TuKetThuc() []string {
	return []string{StateLost, StateHanMuc, StateChan, StateHong, StateXong}
}

// LaTuKetThuc: state có phải một kiểu tự-kết-thúc-không-qua-stop không.
func LaTuKetThuc(state string) bool {
	for _, s := range TuKetThuc() {
		if s == state {
			return true
		}
	}
	return false
}

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

	// StateLyDo là câu giải thích ĐO ĐƯỢC đi kèm State, rỗng khi không đo được.
	// Không phải nơi để nhét văn bản tự do: nó đến từ provider.KetQua.Hong().
	StateLyDo string
	// HanMucDenLai là mốc unix hạn mức được cấp lại, 0 = không rõ. Ghi cho MỌI
	// trạng thái đọc được nó, không riêng `rate_limited` — mốc thời gian là thứ
	// đo được, và vứt đi thì không lấy lại được.
	HanMucDenLai int64
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
type DB struct {
	db *sql.DB
	// phanLoai: xem DungPhanLoaiChet. nil = giữ nguyên StateLost.
	phanLoai PhanLoaiChet
}

// PhanLoaiChet nói một phiên vừa bị phát hiện chết nên mang trạng thái nào.
//
// Vì sao là một hàm cắm vào chứ không phải mã trong gói này: câu trả lời nằm
// trong BẢN GHI của agent, mà đọc bản ghi là việc của provider — `store` không
// được biết Claude hay Grok là gì. Đây cũng là cách giữ "một chỗ quyết định":
// tầng trên cắm ĐÚNG MỘT hàm, và mọi đường phát hiện phiên chết đều đi qua nó.
//
// Trả về trangThai == "" nghĩa là KHÔNG KẾT LUẬN ĐƯỢC → phiên ở lại `lost`.
type PhanLoaiChet func(Session) (trangThai, lyDo string, hanMucDenLai int64)

// DungPhanLoaiChet cắm bộ phân loại vào sổ. Không cắm (CLI cũ, test của gói
// khác) thì hành vi y hệt trước: phiên chết là `lost`, không lý do.
func (d *DB) DungPhanLoaiChet(f PhanLoaiChet) { d.phanLoai = f }

// Path là đường dẫn file cơ sở dữ liệu.
func Path() string { return filepath.Join(paths.AccountsRoot(), "state.db") }

// Open mở (và tạo nếu chưa có) cơ sở dữ liệu ở vị trí chuẩn.
func Open() (*DB, error) { return OpenAt(Path()) }

// OpenAt mở cơ sở dữ liệu ở một đường dẫn cụ thể (test dùng đường dẫn tạm).
func OpenAt(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	// 0o755 ở trên KHÔNG bảo vệ gì trên Windows — bit quyền Unix ở đó chỉ là
	// trang trí, quyền thật kế thừa từ thư mục cha (đã đo, xem internal/acl).
	// Siết một lần ở đây thì mọi thứ tạo sau bên trong đều thừa hưởng.
	//
	// Best-effort: ổ mạng hoặc FAT32 có thể từ chối. Không nuốt trong im lặng —
	// `sagent verify` có ô kiểm nói đúng trạng thái thật.
	_ = acl.Restrict(filepath.Dir(path))
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
	return &DB{db: db}, nil
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

	// v5 — chi phí và token của mỗi bước. Đọc được từ dữ liệu CÓ CẤU TRÚC của
	// agent (trường total_cost_usd/usage), không phải đoán. Lưu để cộng dồn theo
	// ngày/tài khoản và để dashboard hiện "lượt này tốn bao nhiêu".
	`ALTER TABLE flow_steps ADD COLUMN cost_usd REAL NOT NULL DEFAULT 0;
	 ALTER TABLE flow_steps ADD COLUMN tokens_in INTEGER NOT NULL DEFAULT 0;
	 ALTER TABLE flow_steps ADD COLUMN tokens_out INTEGER NOT NULL DEFAULT 0;`,

	// v6 — CÂU HỎI đã gửi cho agent, sau khi thay hết biến.
	//
	// Trước đây chỉ lưu câu trả lời. Nhìn lại một lượt chạy thì thấy agent nói
	// gì mà không thấy nó ĐƯỢC HỎI GÌ — mà phần lớn lỗi của lượt #29 nằm đúng ở
	// câu hỏi: placeholder chưa thay lọt nguyên văn vào prompt, và không ai nhìn
	// ra cho tới khi đọc bản ghi NDJSON thô.
	//
	// Lưu prompt ĐÃ THAY BIẾN (không phải mẫu trong flows.toml) vì đó mới là thứ
	// agent thật sự nhận.
	`ALTER TABLE flow_steps ADD COLUMN prompt TEXT;`,

	// v7 — LỊCH SỬ lời gọi API (đường thứ hai).
	//
	// Vì sao cần: đường này tiêu TIỀN theo token, không phải hạn mức thuê bao.
	// Trước bảng này, một lời gọi in usage ra màn hình rồi biến mất — hết tháng
	// nhìn hoá đơn thì không có gì đối chiếu, và "route chính hỏng bao nhiêu lần
	// tuần này" là câu không ai trả lời được.
	//
	// CỐ Ý KHÔNG CÓ CỘT prompt VÀ CỘT CÂU TRẢ LỜI.
	//
	// Khác với flow_steps (lưu cả hai, xem v4 và v6): ở đó agent chạy trên máy
	// người dùng với dữ liệu của chính họ, và không đọc lại được prompt thì không
	// gỡ được lỗi thay-biến. Ở đây thì khác — người ta dán cả đoạn mã, khoá, dữ
	// liệu khách vào prompt gửi cho nhà cung cấp bên ngoài. Ghi thêm một bản sao
	// vĩnh viễn xuống đĩa là tự tạo một kho bí mật thứ hai mà không ai xin, ngay
	// trong một dự án mà luật số 1 là "file cấu hình không bao giờ chứa secret".
	// Muốn xem lại câu hỏi thì nhìn màn hình lúc gõ; sổ này chỉ trả lời "tốn bao
	// nhiêu, ở đâu, có chạy được không".
	//
	// Có test canh điều này: xem TestLichSuAPIKhongLuuPromptVaCauTraLoi.
	`CREATE TABLE IF NOT EXISTS api_calls (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		luc        TEXT    NOT NULL,
		route      TEXT    NOT NULL,
		model      TEXT    NOT NULL DEFAULT '',
		tokens_in  INTEGER NOT NULL DEFAULT 0,
		tokens_out INTEGER NOT NULL DEFAULT 0,
		cost_usd   REAL    NOT NULL DEFAULT 0,
		ok         INTEGER NOT NULL DEFAULT 0,
		ly_do      TEXT    NOT NULL DEFAULT '',
		mili       INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_api_calls_luc ON api_calls(luc);`,

	// v8 — SỔ ĐĂNG KÝ: cái gì trên đĩa là do sagent tạo ra.
	//
	// Vì sao cần một cái sổ trong khi thư mục đã nằm ngay đó: đĩa nói "có thư
	// mục này", nó KHÔNG nói "sagent tạo ra thư mục này". Hai câu đó khác nhau ở
	// đúng chỗ nguy hiểm nhất của dự án — `xoa`. Cho tới trước bảng này, lá chắn
	// duy nhất là kiểm chuỗi đường dẫn (`insideStore`), tức là hễ thứ gì rơi vào
	// ~/.ai-accounts thì mặc nhiên bị coi là của sagent và xoá đệ quy được.
	// MASTER-PLAN Pha 1 đòi ngược lại: "chỉ xoá thư mục mà registry SỞ HỮU".
	//
	// Sổ cũng là thứ duy nhất trả lời được "sổ ghi có mà đĩa không có" — hồ sơ
	// bị xoá sau lưng sagent (dọn tay, đồng bộ OneDrive, máy khác) hiện không
	// để lại dấu vết nào.
	//
	// CỐ Ý KHÔNG CÓ CỘT NÀO CHỨA SECRET.
	//
	// `routes_so` ghi `key_id` — TÊN FILE khoá trong ~/.ai-accounts/api-keys,
	// không phải khoá. Đây đúng là điều luật số 1 của config đòi ("file cấu hình
	// không bao giờ chứa secret"), và sổ trạng thái không được là cái cửa sau
	// của luật đó: state.db nằm ngoài repo nhưng vẫn bị chép theo khi người ta
	// gửi nhau bản sao lưu để gỡ lỗi. Có test canh danh sách cột:
	// xem TestCaA_SoV8CoBangVaKhongCoCotSecret.
	`CREATE TABLE IF NOT EXISTS profiles_so (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		provider  TEXT    NOT NULL,
		account   TEXT    NOT NULL,
		dir       TEXT    NOT NULL,
		so_tao_ra INTEGER NOT NULL DEFAULT 0,
		ghi_luc   TEXT    NOT NULL,
		UNIQUE(provider, account)
	);
	CREATE INDEX IF NOT EXISTS idx_profiles_so_dir ON profiles_so(dir);
	CREATE TABLE IF NOT EXISTS routes_so (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		ten      TEXT    NOT NULL UNIQUE,
		base_url TEXT    NOT NULL DEFAULT '',
		model    TEXT    NOT NULL DEFAULT '',
		key_id   TEXT    NOT NULL DEFAULT '',
		ghi_luc  TEXT    NOT NULL
	);`,

	// v9 — VÌ SAO phiên chết, khi đo được.
	//
	// Trước bước này cột `state` chỉ có ba giá trị, và mọi phiên tự chết đều bị
	// gán cứng `lost`. `lost` trả lời "nó không còn chạy" nhưng không trả lời
	// câu người vận hành thật sự hỏi: chạy lại được ngay, hay phải chờ hạn mức,
	// hay phải cấp quyền, hay token đã hết hạn. Bốn việc phải làm khác hẳn nhau
	// mà nhìn bảng thì giống hệt nhau.
	//
	// Hai cột này là chỗ giữ câu trả lời khi provider ĐO ĐƯỢC nó:
	//   state_ly_do     — câu giải thích, lấy từ provider.KetQua.Hong();
	//   han_muc_den_lai — mốc unix hạn mức cấp lại, từ rate_limit_info.resetsAt.
	//
	// KHÔNG có cột nào ở đây chứa output của agent. Bản ghi đã nằm sẵn ở file
	// log (cột `log`); chép một bản thứ hai vào sổ trạng thái là tự tạo thêm một
	// kho dữ liệu khách — cùng lý do đã ghi ở v7.
	//
	// Cả hai đều NOT NULL DEFAULT: phiên ghi từ bản sagent cũ đọc lại được ngay,
	// mang giá trị rỗng đúng nghĩa "không đo được".
	`ALTER TABLE sessions ADD COLUMN state_ly_do TEXT NOT NULL DEFAULT '';
	 ALTER TABLE sessions ADD COLUMN han_muc_den_lai INTEGER NOT NULL DEFAULT 0;`,
}

// cotPhien là danh sách cột đọc ra một Session, dùng chung cho mọi truy vấn để
// hai chỗ không thể lệch nhau. COALESCE cho cột thêm sau: file cũ có thể có
// dòng NULL nếu ai đó ghi tay.
const cotPhien = `id,provider,account,clone,dir,pid,COALESCE(log,''),COALESCE(worktree,''),
	started,state,COALESCE(state_ly_do,''),COALESCE(han_muc_den_lai,0)`

// quetPhien đọc một dòng theo đúng thứ tự cotPhien.
func quetPhien(rows *sql.Rows) (Session, error) {
	var s Session
	var started string
	err := rows.Scan(&s.ID, &s.Provider, &s.Account, &s.Clone, &s.Dir,
		&s.PID, &s.Log, &s.Worktree, &started, &s.State, &s.StateLyDo, &s.HanMucDenLai)
	if err != nil {
		return s, err
	}
	s.Started, _ = time.Parse(time.RFC3339, started)
	return s, nil
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
	RunID     int64
	StepID    string
	State     string
	Attempt   int
	Msg       string
	Output    string
	CostUSD   float64
	TokensIn  int
	TokensOut int
	// Prompt là câu hỏi ĐÃ THAY BIẾN gửi cho agent. Rỗng với bước không hỏi ai
	// (shell/notify lưu lệnh hoặc lời nhắn).
	Prompt string
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
			ended=CASE WHEN excluded.state IN ('done','failed','skipped') THEN excluded.ended ELSE NULL END`,
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

// SetStepPrompt lưu CÂU HỎI đã gửi cho agent (đã thay hết biến).
//
// Cắt theo cùng trần với output: prompt của bước gộp có thể nhét cả kết quả
// nhiều bước trước vào, dài không kém câu trả lời.
func (d *DB) SetStepPrompt(runID int64, stepID, prompt string) error {
	if len(prompt) > MaxStepOutput {
		prompt = "…(đã cắt bớt phần đầu)…\n" + prompt[len(prompt)-MaxStepOutput:]
	}
	_, err := d.db.Exec(`UPDATE flow_steps SET prompt=? WHERE run_id=? AND step_id=?`,
		prompt, runID, stepID)
	return err
}

// SetStepCost ghi chi phí và token của một bước. Tách khỏi SetStepOutput vì
// output đến từ bản ghi còn chi phí đến từ dòng result có cấu trúc — hai nguồn,
// và không phải provider nào cũng cho được chi phí (chỉ ghi khi > 0).
func (d *DB) SetStepCost(runID int64, stepID string, costUSD float64, tokIn, tokOut int) error {
	_, err := d.db.Exec(
		`UPDATE flow_steps SET cost_usd=?, tokens_in=?, tokens_out=? WHERE run_id=? AND step_id=?`,
		costUSD, tokIn, tokOut, runID, stepID)
	return err
}

// Steps đọc trạng thái mọi bước của một lần chạy.
func (d *DB) Steps(runID int64) (map[string]StepRun, error) {
	rows, err := d.db.Query(
		`SELECT run_id,step_id,state,attempt,COALESCE(msg,''),COALESCE(output,''),
		        COALESCE(cost_usd,0),COALESCE(tokens_in,0),COALESCE(tokens_out,0),
		        COALESCE(prompt,'')
		   FROM flow_steps WHERE run_id=?`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]StepRun{}
	for rows.Next() {
		var s StepRun
		if err := rows.Scan(&s.RunID, &s.StepID, &s.State, &s.Attempt, &s.Msg, &s.Output,
			&s.CostUSD, &s.TokensIn, &s.TokensOut, &s.Prompt); err != nil {
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
		// Đọc LẠI phiên bản bên trong transaction. `cur` ở trên đọc ngoài giao
		// dịch, nên hai tiến trình sagent khởi động cùng lúc có thể cùng thấy
		// cur=1 rồi cùng chạy migration v2 — đứa sau nhận `duplicate column name`
		// và không mở được DB. Hiếm, nhưng đúng lúc tệ nhất: ngay sau khi nâng cấp.
		var trong int
		if trong, err = schemaVersion(tx); err != nil {
			tx.Rollback()
			return err
		}
		if trong > i {
			tx.Rollback()
			continue // tiến trình khác vừa làm bước này rồi
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
// Lost trả về các phiên đã tự chết (tiến trình biến mất mà không qua `stop`).
//
// Cần riêng hàm này vì hậu duệ của chúng có thể VẪN CHẠY và vẫn tiêu hạn mức —
// `Running()` cố ý không trả về chúng, nên nếu không có đường nào khác thì đám
// tiến trình đó không ai nhìn thấy.
//
// Từ schema v9, "tự chết" có bốn tên chứ không còn một: `lost` là chưa biết vì
// sao, ba tên kia là ĐÃ BIẾT. Hàm này lấy CẢ BỐN — biết vì sao nó chết không
// làm cho đám tiến trình con của nó bớt tiêu hạn mức.
func (d *DB) Lost() ([]Session, error) {
	return d.locPhien(`state IN (`+dauHoi(len(TuKetThuc()))+`) ORDER BY id`, dsChet()...)
}

// PhienChet trả về các phiên đã kết thúc bất thường, MỚI NHẤT TRƯỚC, có trần.
//
// Khác Lost() ở thứ tự và ở cái trần: đây là đường cho MẶT NGƯỜI DÙNG (bảng
// `sagent status`, /api/state). Sổ giữ mọi phiên từng chạy, nên không giới hạn
// thì bảng dài dần theo tháng và cái vừa hỏng lại nằm tít dưới đáy.
func (d *DB) PhienChet(limit int) ([]Session, error) {
	if limit <= 0 {
		limit = 20
	}
	return d.locPhien(`state IN (`+dauHoi(len(TuKetThuc()))+`) ORDER BY id DESC LIMIT ?`,
		append(dsChet(), limit)...)
}

// dauHoi sinh đúng số dấu hỏi cho câu SQL `IN (...)`.
//
// Trước đây chuỗi `(?,?,?,?)` chép cứng ở hai câu SQL, kèm bình luận rằng thêm
// trạng thái sẽ "lỗi ngay, không lệch im lặng". Đúng — nhưng lỗi lúc CHẠY, và
// chỉ ở đường nào có người gọi tới. Sinh theo độ dài danh sách thì không còn
// chỗ nào để lệch.
func dauHoi(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// dsChet là TuKetThuc() đóng gói làm tham số cho câu SQL `IN (...)`.
func dsChet() []any {
	ds := TuKetThuc()
	out := make([]any, 0, len(ds))
	for _, s := range ds {
		out = append(out, s)
	}
	return out
}

func (d *DB) locPhien(where string, args ...any) ([]Session, error) {
	rows, err := d.db.Query(`SELECT `+cotPhien+` FROM sessions WHERE `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		s, err := quetPhien(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (d *DB) Running() ([]Session, error) {
	rows, err := d.db.Query(`SELECT `+cotPhien+
		` FROM sessions WHERE state=? ORDER BY id`, StateRunning)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Session
	var dead []Session
	for rows.Next() {
		s, err := quetPhien(rows)
		if err != nil {
			return nil, err
		}
		if process.IsAlive(s.PID) {
			out = append(out, s)
		} else {
			dead = append(dead, s)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// KHÔNG nuốt lỗi ở đây. Đánh dấu `lost` hỏng thì lần sau gọi lại sẽ tự sửa
	// (thao tác này lặp lại được), nhưng im lặng thì không ai biết DB đang bị
	// khoá quá lâu. Trả kèm lỗi; tầng api quyết định cảnh báo hay bỏ qua — nó có
	// bus sự kiện, còn store thì không.
	var loiDanhDau error
	for _, s := range dead {
		// ĐIỂM QUYẾT ĐỊNH DUY NHẤT của trạng thái phiên chết. Trước v9 dòng dưới
		// gán cứng StateLost; nay nó hỏi bộ phân loại đã cắm, và CHỈ đổi khi bộ
		// đó nói được một cái tên. Không cắm, hoặc cắm mà không đo được → `lost`
		// y như cũ. Không suy đoán ở bất kỳ chỗ nào khác.
		st, ly, han := StateLost, "", int64(0)
		if d.phanLoai != nil {
			if st2, ly2, han2 := d.phanLoai(s); st2 != "" {
				st, ly, han = st2, ly2, han2
			}
		}
		if err := d.SetStateChiTiet(s.ID, st, ly, han); err != nil {
			loiDanhDau = err
		}
	}
	return out, loiDanhDau
}

// SetState đổi trạng thái một phiên.
//
// Xoá luôn lý do cũ: một phiên chuyển sang trạng thái mới mà giữ lại câu giải
// thích của trạng thái trước là nói dối bằng dữ liệu cũ.
func (d *DB) SetState(id int64, state string) error {
	return d.SetStateChiTiet(id, state, "", 0)
}

// SetStateChiTiet đổi trạng thái KÈM lý do đo được và mốc hạn mức cấp lại.
func (d *DB) SetStateChiTiet(id int64, state, lyDo string, hanMucDenLai int64) error {
	_, err := d.db.Exec(
		`UPDATE sessions SET state=?, state_ly_do=?, han_muc_den_lai=? WHERE id=?`,
		state, lyDo, hanMucDenLai, id)
	return err
}

// ---------------------------- lịch sử lời gọi API ----------------------------

// GoiAPI là MỘT dòng trong sổ lời gọi API (bảng api_calls, schema v7).
//
// Sổ này ghi "tốn bao nhiêu / ở đâu / có chạy được không". Nó KHÔNG ghi câu hỏi
// và câu trả lời — lý do ở ghi chú migration v7, đừng thêm vào.
type GoiAPI struct {
	ID    int64
	Luc   time.Time
	Route string
	Model string // model nhà cung cấp BÁO LẠI; rỗng khi lời gọi hỏng
	// TokensIn/TokensOut lấy từ `usage` của phản hồi, không phải đếm tay.
	TokensIn  int
	TokensOut int
	// CostUSD là chi phí quy ra tiền. Endpoint chat/completions KHÔNG trả giá,
	// nên hiện luôn là 0: chỗ này chờ bảng giá theo model, và bịa một con số
	// nhân bừa đơn giá thì tệ hơn để trống — nó trông như đã đo.
	CostUSD float64
	OK      bool
	// LyDo là lỗi NGUYÊN VĂN khi hỏng (kèm request id của nhà cung cấp). Rỗng
	// khi OK.
	LyDo string
	Mili int // thời gian lời gọi, mili giây
}

// MaxLyDoAPI là trần độ dài lý do hỏng lưu xuống sổ.
//
// Có trần vì thân lỗi là nguyên văn của nhà cung cấp và họ có thể trả về cả
// trang HTML khi hạ tầng của họ sập. Cắt phần ĐUÔI, giữ phần ĐẦU — ngược với
// output của bước workflow, vì ở lỗi HTTP thì mã và request id nằm ngay đầu.
const MaxLyDoAPI = 4 * 1024

// AddAPICall ghi một lời gọi vào sổ. Ghi cả lần THÀNH lẫn lần BẠI: "route chính
// hỏng bao nhiêu lần tuần này" chỉ trả lời được khi lần bại cũng có dòng.
func (d *DB) AddAPICall(g GoiAPI) (int64, error) {
	luc := g.Luc
	if luc.IsZero() {
		luc = time.Now()
	}
	ly := g.LyDo
	if len(ly) > MaxLyDoAPI {
		ly = ly[:MaxLyDoAPI] + "…(cắt)"
	}
	res, err := d.db.Exec(
		`INSERT INTO api_calls(luc,route,model,tokens_in,tokens_out,cost_usd,ok,ly_do,mili)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		luc.Format(time.RFC3339), g.Route, g.Model, g.TokensIn, g.TokensOut,
		g.CostUSD, boolInt(g.OK), ly, g.Mili)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// APICalls liệt kê lịch sử, mới nhất trước.
func (d *DB) APICalls(limit int) ([]GoiAPI, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.db.Query(
		`SELECT id,luc,route,COALESCE(model,''),COALESCE(tokens_in,0),COALESCE(tokens_out,0),
		        COALESCE(cost_usd,0),ok,COALESCE(ly_do,''),COALESCE(mili,0)
		   FROM api_calls ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GoiAPI
	for rows.Next() {
		var g GoiAPI
		var luc string
		var ok int
		if err := rows.Scan(&g.ID, &luc, &g.Route, &g.Model, &g.TokensIn, &g.TokensOut,
			&g.CostUSD, &ok, &g.LyDo, &g.Mili); err != nil {
			return nil, err
		}
		g.Luc, _ = time.Parse(time.RFC3339, luc)
		g.OK = ok != 0
		out = append(out, g)
	}
	return out, rows.Err()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
