package tele

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// KHÔNG test nào ở đây chạm vào mạng thật. API Telegram được đóng giả bằng
// httptest, và cấu hình trỏ vào đúng cái server giả đó — nhờ trường `api`.
// Bắn tin thật ra Internet trong `go test ./...` là chuyện không được phép xảy
// ra trên máy người khác, kể cả khi nó "chỉ một tin".

// buuDien là API Telegram giả: ghi lại mọi tin nhận được.
type buuDien struct {
	srv *httptest.Server

	mu    sync.Mutex
	duong []string // đường dẫn từng request (chứa /bot<token>/sendMessage)
	tin   []string // trường text của từng tin
	chat  []string // trường chat_id của từng tin
}

func moBuuDien(t *testing.T) *buuDien {
	t.Helper()
	b := &buuDien{}
	b.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var than struct {
			ChatID string `json:"chat_id"`
			Text   string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&than)
		b.mu.Lock()
		b.duong = append(b.duong, r.URL.Path)
		b.tin = append(b.tin, than.Text)
		b.chat = append(b.chat, than.ChatID)
		b.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	t.Cleanup(b.srv.Close)
	return b
}

func (b *buuDien) soTin() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.tin)
}

// choTin đợi tới khi có đủ n tin, hoặc hết giờ. Gửi tin chạy ở goroutine khác
// nên không được phép "sleep một cái rồi tin là xong".
func (b *buuDien) choTin(t *testing.T, n int) []string {
	t.Helper()
	han := time.Now().Add(3 * time.Second)
	for time.Now().Before(han) {
		if b.soTin() >= n {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.tin) < n {
		t.Fatalf("đợi %d tin, chỉ nhận được %d: %q", n, len(b.tin), b.tin)
	}
	return append([]string(nil), b.tin...)
}

func homeGia(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}
	t.Setenv("HOME", tmp)
	return tmp
}

// ghiCauHinh viết thẳng file JSON (không qua Save) để dựng được cả những cấu
// hình HỎNG mà Save từ chối tạo — đó mới là thứ cần kiểm.
func ghiCauHinh(t *testing.T, noiDung string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ConfigPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigPath(), []byte(noiDung), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------

// Luật 1: token KHÔNG nằm trong repo. Nó phải ở trong kho hồ sơ người dùng.
func TestTokenNamNgoaiRepoTrongKhoHoSo(t *testing.T) {
	homeGia(t)
	if got, muon := ConfigPath(), filepath.Join(paths.AccountsRoot(), "telegram.json"); got != muon {
		t.Fatalf("cấu hình phải nằm ở %s, đang là %s", muon, got)
	}
}

// Tin báo bước hỏng phải trả lời ĐỦ: lượt chạy nào, bước nào, tài khoản nào, vì
// sao, và gõ gì tiếp. Thiếu một trong năm thì người nhận vẫn phải mở máy lên tra.
func TestTinBuocHongNoiDuNamThu(t *testing.T) {
	homeGia(t)
	bd := moBuuDien(t)
	ghiCauHinh(t, `{"token":"123:ABC","chat_id":"-100777","api":"`+bd.srv.URL+`"}`)

	bus := events.NewBus()
	dung := Nghe(bus)

	bus.Publish(events.Event{
		Type: events.Failure, Addr: "ra-soat.chay-test", SessionID: 12,
		Msg: "ra-soat.chay-test: exit status 1",
		Detail: map[string]string{
			"flow": "ra-soat", "run": "12", "step": "chay-test",
			"ly_do": "exit status 1 — FAIL internal/flow", "profile": "claude:phu",
		},
	})
	tin := bd.choTin(t, 1)[0]
	dung()
	bus.Close()

	for _, phai := range []string{
		"#12",                                // lượt chạy nào
		`flow "ra-soat"`,                     // của flow nào
		"chay-test",                          // bước nào
		"claude:phu",                         // tài khoản nào
		"exit status 1 — FAIL internal/flow", // vì sao
		"sagent flow runs",                   // gõ gì để xem tiếp
		"sagent flow resume 12",              // gõ gì để chạy tiếp
	} {
		if !strings.Contains(tin, phai) {
			t.Errorf("tin thiếu %q.\nTin nhận được:\n%s", phai, tin)
		}
	}

	// Gửi đúng chỗ: token trong đường dẫn, chat id trong thân tin.
	bd.mu.Lock()
	defer bd.mu.Unlock()
	if bd.duong[0] != "/bot123:ABC/sendMessage" {
		t.Errorf("gọi sai endpoint: %s", bd.duong[0])
	}
	if bd.chat[0] != "-100777" {
		t.Errorf("gửi sai chat: %s", bd.chat[0])
	}
}

// Chờ duyệt là tin đáng giá nhất: lượt chạy đang ĐỨNG IM đợi người. Tin phải
// kèm sẵn hai lệnh để duyệt/từ chối, không bắt người dùng tự nhớ cú pháp.
func TestTinChoDuyetKemLenhDuyetVaTuChoi(t *testing.T) {
	homeGia(t)
	bd := moBuuDien(t)
	ghiCauHinh(t, `{"token":"t","chat_id":"1","api":"`+bd.srv.URL+`"}`)

	bus := events.NewBus()
	dung := Nghe(bus)
	bus.Publish(events.Event{
		Type: events.FlowWaiting, Addr: "phat-hanh.duyet-merge", SessionID: 7,
		Msg:    "chờ duyệt: gộp vào main?",
		Detail: map[string]string{"flow": "phat-hanh", "run": "7", "step": "duyet-merge", "ly_do": "gộp vào main?"},
	})
	tin := bd.choTin(t, 1)[0]
	dung()
	bus.Close()

	for _, phai := range []string{
		"#7", "duyet-merge", "gộp vào main?",
		"sagent flow approve 7 duyet-merge",
		"sagent flow reject 7 duyet-merge",
	} {
		if !strings.Contains(tin, phai) {
			t.Errorf("tin chờ duyệt thiếu %q.\nTin:\n%s", phai, tin)
		}
	}
}

// Lượt chạy hỏng và lượt chạy xong: cũng phải nói được lượt nào, bước nào.
func TestTinLuotChayHongVaXong(t *testing.T) {
	homeGia(t)
	bd := moBuuDien(t)
	ghiCauHinh(t, `{"token":"t","chat_id":"1","api":"`+bd.srv.URL+`"}`)

	bus := events.NewBus()
	dung := Nghe(bus)
	bus.Publish(events.Event{
		Type: events.FlowFailed, Addr: "ra-soat", SessionID: 3,
		Msg:    "dừng ở bước build: exit status 2",
		Detail: map[string]string{"flow": "ra-soat", "run": "3", "step": "build", "ly_do": "exit status 2"},
	})
	bus.Publish(events.Event{
		Type: events.FlowDone, Addr: "ra-soat", SessionID: 4,
		Msg:    "xong #4",
		Detail: map[string]string{"flow": "ra-soat", "run": "4"},
	})
	tin := bd.choTin(t, 2)
	dung()
	bus.Close()

	if !strings.Contains(tin[0], "#3") || !strings.Contains(tin[0], "build") ||
		!strings.Contains(tin[0], "exit status 2") {
		t.Errorf("tin lượt chạy hỏng chưa đủ:\n%s", tin[0])
	}
	if !strings.Contains(tin[1], "#4") || !strings.Contains(tin[1], "XONG") {
		t.Errorf("tin lượt chạy xong chưa đủ:\n%s", tin[1])
	}
}

// Luật 3: KHÔNG spam. Bus mang đủ thứ event (mỗi bước chạy, mỗi cảnh báo, mỗi
// dòng info) — báo hết thì người ta tắt thông báo, và mất luôn cái đáng báo.
func TestKhongSpamMoiSuKien(t *testing.T) {
	homeGia(t)
	bd := moBuuDien(t)
	ghiCauHinh(t, `{"token":"t","chat_id":"1","api":"`+bd.srv.URL+`"}`)

	bus := events.NewBus()
	dung := Nghe(bus)
	for _, e := range []events.Event{
		{Type: events.FlowStarted, Addr: "ra-soat", SessionID: 1, Msg: "bắt đầu #1 — 5 bước"},
		{Type: events.FlowStep, Addr: "ra-soat.a", SessionID: 1, Msg: "chạy [shell] lần 1/1"},
		{Type: events.FlowStep, Addr: "ra-soat.a", SessionID: 1, Msg: "xong"},
		{Type: events.Info, Msg: "chạy song song 3 bước"},
		{Type: events.Warning, Msg: "ra-soat.a hỏng — thử lại sau 2s"},
		{Type: events.SessionStarted, Addr: "claude:phu", SessionID: 9},
		{Type: events.FlowApproved, SessionID: 1, Msg: "đã duyệt"},
		// Failure KHÔNG thuộc flow (phiên không dừng được) cũng không được nhắn:
		// nó không có lượt chạy nào để mở ra xem.
		{Type: events.Failure, Msg: "#9 (PID 4242) không dừng được: access denied"},
	} {
		bus.Publish(e)
	}

	// Mốc: một event ĐÁNG nhắn phát sau cùng. Nó tới nơi nghĩa là bộ nghe đã
	// xử lý xong tất cả những cái trước — không có chuyện "chưa kịp gửi".
	bus.Publish(events.Event{Type: events.FlowDone, Addr: "ra-soat", SessionID: 1, Msg: "xong #1",
		Detail: map[string]string{"flow": "ra-soat", "run": "1"}})
	tin := bd.choTin(t, 1)
	dung()
	bus.Close()

	if len(tin) != 1 {
		t.Fatalf("chỉ được nhắn 1 tin (lượt chạy xong), đã nhắn %d:\n%s", len(tin), strings.Join(tin, "\n---\n"))
	}
}

// ĐIỀU KIỆN QUAN TRỌNG NHẤT của tính năng này: chưa cấu hình thì IM LẶNG.
//
// Bốn kiểu "chưa cấu hình" đều phải im. Ba kiểu sau là bài test THẬT chứ không
// rỗng nghĩa: file có trường `api` trỏ đúng vào bưu điện giả, nên nếu Load()
// dễ dãi cho qua thì server sẽ nhận được tin và test đỏ ngay.
func TestChuaCauHinhThiKhongGuiGi(t *testing.T) {
	cases := []struct{ ten, file string }{
		{"không có file", ""},
		{"thiếu token", `{"token":"","chat_id":"1","api":"%s"}`},
		{"thiếu chat id", `{"token":"t","chat_id":"","api":"%s"}`},
		{"JSON hỏng", `{"token":"t","chat_id":`},
	}
	for _, c := range cases {
		t.Run(c.ten, func(t *testing.T) {
			homeGia(t)
			bd := moBuuDien(t)
			if c.file != "" {
				ghiCauHinh(t, strings.ReplaceAll(c.file, "%s", bd.srv.URL))
			}

			if b := Mo(); b.DaCauHinh() {
				t.Error("cấu hình nửa vời mà vẫn coi là đã bật")
			}

			bus := events.NewBus()
			// Người nghe thật (như CLI hay dashboard) vẫn phải nhận đủ event:
			// việc báo tin im lặng KHÔNG được làm hỏng luồng sự thật.
			ch, huy := bus.Subscribe(8)
			defer huy()

			dung := Nghe(bus)
			bus.Publish(events.Event{
				Type: events.FlowFailed, Addr: "ra-soat", SessionID: 5, Msg: "dừng ở bước build",
				Detail: map[string]string{"flow": "ra-soat", "run": "5", "step": "build"},
			})
			select {
			case e := <-ch:
				if e.Type != events.FlowFailed {
					t.Fatalf("mặt khác nhận sai event: %+v", e)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("event không tới được mặt khác — báo tin đã cản luồng sự thật")
			}
			dung()
			bus.Close()

			if n := bd.soTin(); n != 0 {
				t.Fatalf("chưa cấu hình mà vẫn gửi %d tin", n)
			}
		})
	}
}

// Luật 2: Telegram hỏng thì lượt chạy vẫn phải chạy. Server trả 500, trả rác,
// hoặc chết hẳn — không cái nào được làm luồng event nghẽn hay hàm dừng treo.
func TestTelegramHongKhongLamHongLuotChay(t *testing.T) {
	homeGia(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":403,"description":"bot was blocked by the user"}`))
	}))
	defer srv.Close()
	ghiCauHinh(t, `{"token":"t","chat_id":"1","api":"`+srv.URL+`"}`)

	bus := events.NewBus()
	ch, huy := bus.Subscribe(16)
	defer huy()
	dung := Nghe(bus)

	for i := 0; i < 5; i++ {
		bus.Publish(events.Event{Type: events.FlowFailed, Addr: "f", SessionID: int64(i + 1),
			Msg: "dừng ở bước x", Detail: map[string]string{"flow": "f", "run": "1", "step": "x"}})
	}
	// Mặt khác nhận đủ 5 event dù mọi lời gọi Telegram đều hỏng.
	for i := 0; i < 5; i++ {
		select {
		case <-ch:
		case <-time.After(2 * time.Second):
			t.Fatalf("mất event thứ %d vì lỗi gửi tin", i+1)
		}
	}

	xong := make(chan struct{})
	go func() { dung(); close(xong) }()
	select {
	case <-xong:
	case <-time.After(5 * time.Second):
		t.Fatal("hàm dừng treo khi Telegram hỏng")
	}
	bus.Close()

	// Và lỗi phải giữ NGUYÊN VĂN lời Telegram khi có ai đó thật sự hỏi tới.
	err := New(Config{Token: "t", ChatID: "1", API: srv.URL}).Gui(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "bot was blocked by the user") {
		t.Fatalf("phải giữ nguyên văn lời Telegram, được: %v", err)
	}
}

// Save từ chối cấu hình nửa vời ngay tại chỗ ghi — để không bao giờ có file
// nửa vời nằm trên đĩa mà chủ máy tưởng là đã bật.
func TestSaveTuChoiCauHinhNuaVoi(t *testing.T) {
	homeGia(t)
	if err := Save(Config{Token: "", ChatID: "1"}); err == nil {
		t.Error("thiếu token mà vẫn lưu")
	}
	if err := Save(Config{Token: "t", ChatID: ""}); err == nil {
		t.Error("thiếu chat id mà vẫn lưu")
	}
	if err := Save(Config{Token: " 123:ABC ", ChatID: " -100 "}); err != nil {
		t.Fatal(err)
	}
	c := Load()
	if c == nil || c.Token != "123:ABC" || c.ChatID != "-100" {
		t.Fatalf("đọc lại không khớp: %+v", c)
	}
}

// Thu phải gửi thật, và khi chưa cấu hình thì báo lỗi CHỈ ĐƯỜNG chứ không im —
// người bấm "gửi thử" đang chủ động hỏi, im lặng lúc này là bỏ rơi họ.
func TestThuGuiThatVaBaoChiDuongKhiChuaCauHinh(t *testing.T) {
	homeGia(t)
	if err := Mo().Thu(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "sagent tele --set-token") {
		t.Fatalf("phải chỉ đúng lệnh cần gõ, được: %v", err)
	}

	bd := moBuuDien(t)
	ghiCauHinh(t, `{"token":"t","chat_id":"1","api":"`+bd.srv.URL+`"}`)
	if err := Mo().Thu(context.Background()); err != nil {
		t.Fatal(err)
	}
	tin := bd.choTin(t, 1)[0]
	if !strings.Contains(tin, "tin thử") {
		t.Fatalf("tin thử không đúng nội dung:\n%s", tin)
	}
}
