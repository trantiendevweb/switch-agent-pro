package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Ba trạng thái phiên đo được, ĐI QUA CẢ ĐƯỜNG THẬT: file log trên đĩa →
// adapter đọc bản ghi → provider.PhanLoaiChet → sổ → SessionHong.
//
// Test ở internal/provider chứng minh phép suy đúng; test ở đây chứng minh nó
// ĐƯỢC CẮM VÀO. Hai chuyện khác nhau — bỏ dòng DungPhanLoaiChet trong New() thì
// mọi test của provider vẫn xanh trong khi sản phẩm quay về `lost`.

// Tên trạng thái phải trùng nhau ở hai gói. Sổ trên đĩa lưu chuỗi thô, nên hai
// bên lệch một chữ là dữ liệu cũ không đọc ra được trạng thái nào — và không ai
// báo lỗi.
func TestTenTrangThaiTrungNhauGiuaProviderVaStore(t *testing.T) {
	cap := []struct{ p, s string }{
		{provider.ChetHanMuc, store.StateHanMuc},
		{provider.ChetChanQuyen, store.StateChan},
		{provider.ChetLoiAPI, store.StateHong},
	}
	for _, c := range cap {
		if c.p != c.s {
			t.Errorf("provider nói %q, store nói %q", c.p, c.s)
		}
	}
}

func moAPI(t *testing.T) *API {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	a, err := New(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return a
}

// themPhienChet ghi một phiên với PID chắc chắn đã chết, kèm file log nội dung
// `raw` (rỗng = không tạo file, tức phiên không có bản ghi nào để đọc).
func themPhienChet(t *testing.T, a *API, prov, acc, raw string) int64 {
	t.Helper()
	logPath := ""
	if raw != "" {
		logPath = filepath.Join(t.TempDir(), acc+".ndjson")
		if err := os.WriteFile(logPath, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	id, err := a.db.AddSession(store.Session{
		Provider: prov, Account: acc, Dir: "d", PID: 0x7FFFFFF0, Log: logPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

const (
	logAPIHetHanMuc = `{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1755700000}}
{"type":"result","subtype":"error_during_execution","is_error":true,"api_error_status":"429","result":"","num_turns":3}`

	logAPIChanQuyen = `{"type":"result","subtype":"success","is_error":false,"result":"","num_turns":2,"permission_denials":[{"tool_name":"Bash"}]}`

	logAPILoiAPI = `{"type":"result","subtype":"error_during_execution","is_error":true,"api_error_status":"invalid_api_key","result":"","num_turns":1}`
)

// (a) SUY ĐÚNG, qua đúng đường sản phẩm đi.
func TestPhienChetDuocPhanLoaiTuBanGhiThat(t *testing.T) {
	a := moAPI(t)
	themPhienChet(t, a, "claude", "hanmuc", logAPIHetHanMuc)
	themPhienChet(t, a, "claude", "chan", logAPIChanQuyen)
	themPhienChet(t, a, "claude", "loi", logAPILoiAPI)

	// SessionList là chỗ phiên chết bị phát hiện (nó đối chiếu PID thật).
	if list, err := a.SessionList(); err != nil || len(list) != 0 {
		t.Fatalf("SessionList = %+v, %v — cả ba phiên đều có PID đã chết", list, err)
	}

	hong, err := a.SessionHong(10)
	if err != nil {
		t.Fatal(err)
	}
	theoTK := map[string]store.Session{}
	for _, s := range hong {
		theoTK[s.Account] = s
	}
	if len(theoTK) != 3 {
		t.Fatalf("muốn 3 phiên chết, được %d: %+v", len(theoTK), hong)
	}
	if got := theoTK["hanmuc"]; got.State != store.StateHanMuc {
		t.Errorf("phiên hết hạn mức: state = %q, muốn %q", got.State, store.StateHanMuc)
	} else if got.HanMucDenLai != 1755700000 {
		t.Errorf("mốc cấp lại = %d, muốn 1755700000", got.HanMucDenLai)
	} else if got.StateLyDo == "" {
		t.Error("phiên hết hạn mức không mang lý do nào")
	}
	if got := theoTK["chan"].State; got != store.StateChan {
		t.Errorf("phiên bị chặn quyền: state = %q, muốn %q", got, store.StateChan)
	}
	if got := theoTK["loi"].State; got != store.StateHong {
		t.Errorf("phiên lỗi API: state = %q, muốn %q", got, store.StateHong)
	}
}

// (b) THIẾU DỮ LIỆU THÌ KHÔNG SUY — bốn kiểu thiếu, cả bốn phải ở lại `lost`.
//
// Đây là test ĐỎ KHI GỠ theo chiều ngược lại: nó bắt việc thêm suy đoán, không
// bắt việc thiếu suy đoán. Cả hai chiều đều cần — một chiều canh tính năng, một
// chiều canh luật "không bịa".
func TestPhienThieuDuLieuOLaiLost(t *testing.T) {
	a := moAPI(t)
	// 1. không có file log (phiên chạy thẳng terminal, không phải fleet)
	themPhienChet(t, a, "claude", "khong-log", "")
	// 2. provider chưa đo được cách đọc kết quả — dù bản ghi là của Claude
	themPhienChet(t, a, "codex", "chua-do", logAPIHetHanMuc)
	// 3. log có nhưng không phải bản ghi có cấu trúc
	themPhienChet(t, a, "claude", "log-rac", "no output produced\nfailed to authenticate\n")
	// 4. provider không tồn tại
	themPhienChet(t, a, "khonghe", "provider-la", logAPIHetHanMuc)
	// 5. log trỏ tới file đã bị dọn
	id, err := a.db.AddSession(store.Session{Provider: "claude", Account: "log-mat",
		Dir: "d", PID: 0x7FFFFFF0, Log: filepath.Join(t.TempDir(), "khong-co.ndjson")})
	if err != nil {
		t.Fatal(err)
	}
	_ = id

	if _, err := a.SessionList(); err != nil {
		t.Fatal(err)
	}
	hong, err := a.SessionHong(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hong) != 5 {
		t.Fatalf("muốn 5 phiên chết, được %d", len(hong))
	}
	for _, s := range hong {
		if s.State != store.StateLost {
			t.Errorf("%s: state = %q — không đo được thì phải ở lại %q, "+
				"đoán bừa sẽ đẩy người vận hành đi sửa nhầm chỗ",
				s.Account, s.State, store.StateLost)
		}
		if s.StateLyDo != "" || s.HanMucDenLai != 0 {
			t.Errorf("%s: không đo được mà vẫn ghi (%q,%d)", s.Account, s.StateLyDo, s.HanMucDenLai)
		}
	}
}

// Lượt chạy XONG XUÔI rồi tiến trình thoát: vẫn rời sổ "đang chạy", nhưng không
// được mang một trong ba trạng thái hỏng.
func TestPhienChayXongKhongBiGanTrangThaiHong(t *testing.T) {
	a := moAPI(t)
	themPhienChet(t, a, "claude", "xong", `{"type":"result","subtype":"success","is_error":false,`+
		`"result":"Đã sửa và chạy test: PASS","num_turns":5}`)
	if _, err := a.SessionList(); err != nil {
		t.Fatal(err)
	}
	hong, err := a.SessionHong(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hong) != 1 {
		t.Fatalf("muốn 1, được %d", len(hong))
	}
	if hong[0].State != store.StateLost {
		t.Fatalf("lượt chạy THÀNH CÔNG bị gán %q", hong[0].State)
	}
}

// `sagent quet` (SessionSweep) phải nhìn thấy phiên mang trạng thái mới. Đám
// tiến trình con của một phiên hết-hạn-mức vẫn sống và vẫn tiêu hạn mức y hệt
// phiên `lost`.
func TestSweepVanThayPhienMangTrangThaiMoi(t *testing.T) {
	a := moAPI(t)
	themPhienChet(t, a, "claude", "hanmuc", logAPIHetHanMuc)
	if _, err := a.SessionList(); err != nil {
		t.Fatal(err)
	}
	lost, err := a.db.Lost()
	if err != nil {
		t.Fatal(err)
	}
	if len(lost) != 1 || lost[0].State != store.StateHanMuc {
		t.Fatalf("phiên hết hạn mức tàng hình khỏi đường quét mồ côi: %+v", lost)
	}
	// SessionSweep chạy được (không có tiến trình mồ côi thật nên danh sách rỗng).
	if _, err := a.SessionSweep(false); err != nil {
		t.Fatal(err)
	}
}
