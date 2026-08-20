package dash

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Ba trạng thái phiên đo được (schema v9) ở MẶT WEB.
//
// Luật của việc này: bốn mặt chỉ ĐỌC LẠI `Session.State` từ /api/state, không
// mặt nào tự suy. Ba nhóm test dưới đây canh đúng điều đó — một nhóm cho hợp
// đồng dữ liệu, hai nhóm cho hai trang THẬT SỰ vẽ phiên (index.html và 3d.html;
// flow.html và vanphong.html đọc /api/flow/detail, không đụng tới phiên).

const logChanQuyenWeb = `{"type":"result","subtype":"success","is_error":false,"result":"",` +
	`"num_turns":2,"permission_denials":[{"tool_name":"Bash"}]}`

// /api/state phải mang trạng thái phiên chết ra tới trình duyệt. Thiếu nó thì
// trang không có cách nào biết, và mọi phần vẽ ở dưới đều vô nghĩa.
func TestAPIStateMangTrangThaiPhienChet(t *testing.T) {
	s := newTestServer(t)

	logPath := filepath.Join(t.TempDir(), "phien.ndjson")
	if err := os.WriteFile(logPath, []byte(logChanQuyenWeb), 0o600); err != nil {
		t.Fatal(err)
	}
	// Ghi phiên thẳng vào sổ — cùng file mà server đang mở (newTestServer đã trỏ
	// HOME về thư mục tạm). Cố ý không đi qua `fleet.start`: bật agent thật
	// trong test là đốt hạn mức, và thứ cần kiểm ở đây là ĐƯỜNG ĐỌC.
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.AddSession(store.Session{
		Provider: "claude", Account: "chan", Dir: "d", PID: 0x7FFFFFF0, Log: logPath,
	}); err != nil {
		t.Fatal(err)
	}
	db.Close()

	ck := dangNhap(t, s, "127.0.0.1:7777")
	r := req("GET", "/api/state")
	r.Host = "127.0.0.1:7777"
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("/api/state: mã %d — %s", w.Code, w.Body.String())
	}
	var got struct {
		Sessions []struct {
			Addr         string `json:"addr"`
			State        string `json:"state"`
			LyDo         string `json:"lyDo"`
			HanMucDenLai int64  `json:"hanMucDenLai"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Sessions) != 1 {
		t.Fatalf("muốn 1 phiên trong /api/state, được %d — phiên vừa chết bất thường "+
			"phải đi cùng mảng `sessions` với phiên đang chạy", len(got.Sessions))
	}
	p := got.Sessions[0]
	if p.State != store.StateChan {
		t.Errorf("state = %q, muốn %q", p.State, store.StateChan)
	}
	if p.LyDo == "" {
		t.Error("phiên bị chặn quyền ra tới web mà không mang lý do — trang sẽ chỉ hiện một mã máy")
	}
}

// index.html KHÔNG được tự suy trạng thái từ pid nữa.
//
// Bản cũ có đúng dòng `function trangThai(s){ return (s.pid > 0) ? 'running' :
// 'pending'; }`. Nó là một luật thứ hai đặt trong trình duyệt, và một phiên đã
// chết vì hết hạn mức vẫn được nó gọi là `running`.
func TestMat2DDocLaiStateChuKhongSuyTuPID(t *testing.T) {
	ma := boComment(doc2D(t))
	if !regexp.MustCompile(`function\s+trangThai\s*\([^)]*\)\s*\{[^}]*s\.state`).MatchString(ma) {
		t.Error("index.html: trangThai() không đọc s.state — mặt 2D đang tự suy trạng thái phiên " +
			"thay vì dùng cái hợp đồng đã quyết")
	}
	// Ba trạng thái v9 phải có mặt trong bảng màu VÀ bảng nhãn.
	for _, tt := range []string{"rate_limited", "blocked", "failed"} {
		if !strings.Contains(ma, tt) {
			t.Errorf("index.html không biết trạng thái %q — card của phiên đó sẽ rơi về màu mặc định", tt)
		}
	}
	// Lý do và mốc hạn mức phải được vẽ ra, không chỉ nhận về rồi vứt.
	for _, truong := range []string{"lyDo", "hanMucDenLai"} {
		if !strings.Contains(ma, truong) {
			t.Errorf("index.html nhận %s từ /api/state nhưng không hiển thị ở đâu cả", truong)
		}
	}
	// Nút Dừng chỉ cho phiên còn sống.
	if !regexp.MustCompile(`song\s*\?[^;]*data-stop`).MatchString(ma) {
		t.Error("index.html: nút Dừng không bị chặn ở phiên đã chết — bấm nó là thao tác vô nghĩa")
	}
}

// 3d.html cũng vậy: orb phải lấy màu từ `state`, không phải mặc định "sống thì
// đang chạy". Bản cũ có đúng dòng `const tt = A && A.state === 'pending' ?
// 'pending' : 'running';` — một phiên chết vẫn ra orb xanh đang đập.
func TestMat3DDocLaiStateChuKhongMacDinhRunning(t *testing.T) {
	ma := boComment(doc3D(t))
	if !regexp.MustCompile(`function\s+ttPhien\s*\([^)]*\)\s*\{[^}]*s\.state`).MatchString(ma) {
		t.Error("3d.html: không có ttPhien() đọc s.state — màn 3D đang tự đoán trạng thái phiên")
	}
	if regexp.MustCompile(`const\s+tt\s*=\s*A\s*&&\s*A\.state\s*===\s*'pending'\s*\?\s*'pending'\s*:\s*'running'`).MatchString(ma) {
		t.Error("3d.html: còn dòng mặc-định-running cũ — phiên đã chết vẫn ra orb xanh")
	}
	for _, tt := range []string{"rate_limited", "blocked", "failed"} {
		if !strings.Contains(ma, tt) {
			t.Errorf("3d.html không biết trạng thái %q", tt)
		}
	}
	// Màu phải đi qua mauToken (tức vendor/token.css), không phải mã màu chép cứng.
	for _, d := range []string{"MAU_PHIEN", "mauPhien"} {
		if !strings.Contains(ma, d) {
			t.Errorf("3d.html thiếu %s — bảng màu trạng thái phiên không có nguồn chung", d)
		}
	}
	if regexp.MustCompile(`MAU_PHIEN\s*=\s*\{[^}]*0x[0-9a-fA-F]{6}`).MatchString(ma) {
		t.Error("3d.html: MAU_PHIEN chép cứng mã màu thay vì đọc token.css — " +
			"sửa token.css thì 3D giữ màu cũ, im lặng")
	}
}

// Cả hai mặt phải nói ĐÚNG MỘT BỘ nhãn cho ba trạng thái. Hai mặt hai chữ khác
// nhau cho cùng một `state` là kiểu lệch mà token.css sinh ra để chặn — chỉ có
// điều lần này lệch ở chữ chứ không phải ở màu.
func TestHaiMatNoiCungMotBoNhanTrangThai(t *testing.T) {
	nhan := map[string]string{
		"rate_limited": "hết hạn mức",
		"blocked":      "bị chặn quyền",
		"failed":       "lỗi nhà cung cấp",
	}
	for _, ten := range []string{"index.html", "3d.html"} {
		b, err := os.ReadFile(filepath.Join("web", ten))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for ma, chu := range nhan {
			if !strings.Contains(s, chu) {
				t.Errorf("%s: trạng thái %q không có nhãn %q — hai mặt đang gọi cùng "+
					"một thứ bằng hai cái tên", ten, ma, chu)
			}
		}
	}
}
