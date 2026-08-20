package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// ---------------------------- sổ lời gọi API (v7) ----------------------------

// Bảng api_calls (v7) phải ghi đủ: thời điểm, route, model, token vào/ra, chi
// phí, thành-bại, lý do hỏng. Ghi cả lần THÀNH lẫn lần BẠI — "route chính hỏng
// bao nhiêu lần tuần này" chỉ trả lời được khi lần bại cũng có dòng.
func TestAPICallsGhiDuVaTraMoiNhatTruoc(t *testing.T) {
	db := open(t)
	luc := time.Date(2026, 8, 20, 9, 30, 0, 0, time.UTC)
	if _, err := db.AddAPICall(GoiAPI{
		Luc: luc, Route: "chinh", Model: "grok-4.5",
		TokensIn: 120, TokensOut: 45, CostUSD: 0, OK: true, Mili: 1500,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.AddAPICall(GoiAPI{
		Luc: luc.Add(time.Minute), Route: "phu", OK: false,
		LyDo: "phu trả HTTP 429: rate limit (request id: RID-7)",
	}); err != nil {
		t.Fatal(err)
	}

	ds, err := db.APICalls(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 {
		t.Fatalf("sổ có %d dòng, muốn 2", len(ds))
	}
	// Mới nhất trước — cùng quy ước với ListRuns.
	if ds[0].Route != "phu" || ds[1].Route != "chinh" {
		t.Fatalf("thứ tự sai, muốn mới nhất trước: %+v", ds)
	}
	if ds[0].OK {
		t.Error("dòng hỏng bị ghi thành chạy được")
	}
	// Lý do hỏng phải NGUYÊN VĂN, kèm request id — đó là thứ duy nhất dùng được
	// khi phải đi hỏi nhà cung cấp.
	if !strings.Contains(ds[0].LyDo, "RID-7") {
		t.Errorf("mất request id trong lý do hỏng: %q", ds[0].LyDo)
	}
	g := ds[1]
	if !g.OK || g.Model != "grok-4.5" || g.TokensIn != 120 || g.TokensOut != 45 || g.Mili != 1500 {
		t.Errorf("dòng thành công mất dữ liệu: %+v", g)
	}
	if !g.Luc.Equal(luc) {
		t.Errorf("thời điểm = %v, muốn %v", g.Luc, luc)
	}
}

// Lý do hỏng phải bị cắt: thân lỗi là nguyên văn của nhà cung cấp, và họ có thể
// trả về cả trang HTML khi hạ tầng của họ sập. Không cắt thì một sự cố bên kia
// làm phình state.db bên này.
func TestAPICallsCatLyDoQuaDai(t *testing.T) {
	db := open(t)
	dai := strings.Repeat("x", MaxLyDoAPI*2)
	if _, err := db.AddAPICall(GoiAPI{Route: "r", LyDo: dai}); err != nil {
		t.Fatal(err)
	}
	ds, err := db.APICalls(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds[0].LyDo) > MaxLyDoAPI+32 {
		t.Errorf("lý do dài %d ký tự, không bị cắt ở trần %d", len(ds[0].LyDo), MaxLyDoAPI)
	}
	// Cắt phần ĐUÔI, giữ phần ĐẦU: ở lỗi HTTP thì mã và request id nằm ngay đầu.
	if !strings.HasPrefix(ds[0].LyDo, "xxxx") {
		t.Errorf("cắt nhầm đầu — mã lỗi và request id nằm ở đầu: %.40q", ds[0].LyDo)
	}
}

// Bảng api_calls CỐ Ý không có cột prompt và cột câu trả lời.
//
// Khác flow_steps (lưu cả hai, xem migration v4 và v6): ở đó agent chạy trên máy
// người dùng với dữ liệu của chính họ. Ở đây người ta dán mã, khoá và dữ liệu
// khách vào prompt gửi cho nhà cung cấp bên ngoài — ghi thêm một bản sao vĩnh
// viễn xuống đĩa là tự tạo kho bí mật thứ hai mà không ai xin.
//
// Test soi thẳng SCHEMA chứ không soi dữ liệu: thêm cột rồi để trống thì nó vẫn
// phải đỏ, vì cột trống hôm nay là cột được điền vào bản sau.
func TestLichSuAPIKhongLuuPromptVaCauTraLoi(t *testing.T) {
	db := open(t)
	rows, err := db.db.Query(`PRAGMA table_info(api_calls)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var cot []string
	for rows.Next() {
		var cid int
		var ten, kieu string
		var notnull int
		var mac any
		var pk int
		if err := rows.Scan(&cid, &ten, &kieu, &notnull, &mac, &pk); err != nil {
			t.Fatal(err)
		}
		cot = append(cot, ten)
	}
	if len(cot) == 0 {
		t.Fatal("không có bảng api_calls — migration v7 chưa chạy")
	}
	cam := []string{"prompt", "cau_hoi", "cauhoi", "noi_dung", "noidung",
		"tra_loi", "traloi", "answer", "content", "message", "response", "body"}
	for _, c := range cot {
		l := strings.ToLower(c)
		for _, x := range cam {
			if strings.Contains(l, x) {
				t.Errorf("api_calls có cột %q — sổ này chỉ được ghi tiền và token, "+
					"không được ghi nội dung hội thoại (xem ghi chú migration v7)", c)
			}
		}
	}
	// Và phải có đủ những cột nó CÓ nhiệm vụ ghi.
	for _, phai := range []string{"luc", "route", "model", "tokens_in", "tokens_out",
		"cost_usd", "ok", "ly_do"} {
		co := false
		for _, c := range cot {
			if c == phai {
				co = true
			}
		}
		if !co {
			t.Errorf("api_calls thiếu cột %q — thấy %v", phai, cot)
		}
	}
}
