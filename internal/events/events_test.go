package events

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Bus là xương sống của cả bốn mặt điều khiển (CLI, dashboard 2D, workflow
// board, 3D) nên bài test ở đây canh HỢP ĐỒNG chứ không canh cách viết:
//   - tên chuỗi Type và tên trường JSON (mặt web đọc thẳng qua SSE),
//   - Publish không bao giờ chặn phần lõi,
//   - Close/huỷ đăng ký không làm nổ tiến trình.
//
// Đệm dùng thật trong repo: cmd/sagent/main.go 128, internal/dash/server.go
// 256, internal/tele/tele.go 256 — các con số dưới đây bám theo đó.

// hutHet lấy hết event đang nằm sẵn trong đệm rồi trả về ngay.
//
// Gọi được vì Publish gửi ĐỒNG BỘ (đang giữ b.mu): Publish trả về là event đã
// nằm trong đệm, không có độ trễ nào để phải chờ.
func hutHet(t *testing.T, ch <-chan Event) []Event {
	t.Helper()
	var out []Event
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, e)
		default:
			return out
		}
	}
}

// dongChua báo kênh đã đóng hay chưa, chờ tối đa d.
func dongChua(ch <-chan Event, d time.Duration) bool {
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return true
			} // còn event thì bỏ qua, đọc tiếp
		case <-time.After(d):
			return false
		}
	}
}

// --- Phát và nhận ------------------------------------------------------

// Ba mặt cùng nghe một bus thì cả ba phải thấy y hệt nhau. Đây chính là luật
// "một luồng sự thật" của MASTER-PLAN mục 2c.
func TestPhatToiMoiNguoiNghe(t *testing.T) {
	b := NewBus()
	defer b.Close()

	var chans []<-chan Event
	for i := 0; i < 3; i++ {
		ch, huy := b.Subscribe(8)
		defer huy()
		chans = append(chans, ch)
	}

	goc := Event{Type: SessionStarted, Addr: "claude:phu#1", SessionID: 7,
		Msg: "đã chạy", Detail: map[string]string{"path": "C:\\wt"}}
	b.Publish(goc)

	for i, ch := range chans {
		got := hutHet(t, ch)
		if len(got) != 1 {
			t.Fatalf("người nghe %d: có %d event, muốn 1", i, len(got))
		}
		e := got[0]
		if e.V != SchemaVersion || e.Time.IsZero() {
			t.Errorf("người nghe %d: V=%d time=%v — Publish phải tự điền", i, e.V, e.Time)
		}
		e.V, e.Time = 0, time.Time{} // bỏ hai trường Publish tự điền rồi so phần còn lại
		if !reflect.DeepEqual(e, goc) {
			t.Errorf("người nghe %d nhận %+v, muốn %+v", i, e, goc)
		}
	}
}

// Đệm <= 0 phải rơi về 64. Kiểm bằng cách đổ 65 event: đúng 64 cái lọt.
func TestDemKhongHopLeVeMacDinh64(t *testing.T) {
	b := NewBus()
	defer b.Close()
	ch, huy := b.Subscribe(0)
	defer huy()

	for i := 0; i < 65; i++ {
		b.Publish(Event{Type: Info, Msg: strconv.Itoa(i)})
	}
	got := hutHet(t, ch)
	if len(got) != 64 {
		t.Fatalf("đệm mặc định giữ %d event, muốn 64", len(got))
	}
	if got[0].Msg != "0" || got[63].Msg != "63" {
		t.Errorf("giữ [%s..%s], muốn [0..63]", got[0].Msg, got[63].Msg)
	}
}

// --- Người nghe chậm ---------------------------------------------------

// Điều kiện sống còn: một mặt điều khiển chậm (dashboard mở tab nền, tele đang
// chờ mạng 15s) KHÔNG được làm treo phần lõi. Publish phải rơi event chứ không
// được chặn.
//
// Đo thật trên máy này: 500 lần Publish với một người nghe đệm 4 hết 0–124µs
// (có lần ra đúng 0s vì đồng hồ Windows thô ~0,5ms). Mốc 10s dưới đây rộng gấp
// hơn 80.000 lần, chỉ để bắt CHẶN THẬT (deadlock), không nhạy với máy chậm.
func TestNguoiNgheChamKhongChanLoi(t *testing.T) {
	b := NewBus()
	defer b.Close()

	cham, huyCham := b.Subscribe(4) // không đọc gì suốt lúc phát
	defer huyCham()
	nhanh, huyNhanh := b.Subscribe(1000)
	defer huyNhanh()

	const soLan = 500
	xong := make(chan time.Duration, 1)
	go func() {
		batDau := time.Now()
		for i := 0; i < soLan; i++ {
			b.Publish(Event{Type: Info, Msg: strconv.Itoa(i)})
		}
		xong <- time.Since(batDau)
	}()

	select {
	case d := <-xong:
		t.Logf("%d lần Publish hết %v", soLan, d)
	case <-time.After(10 * time.Second):
		t.Fatal("Publish bị CHẶN bởi người nghe chậm — phần lõi sẽ treo theo dashboard")
	}

	// Người nghe chậm mất event, nhưng người nghe nhanh vẫn đủ 500.
	if got := hutHet(t, nhanh); len(got) != soLan {
		t.Errorf("người nghe nhanh nhận %d/%d event", len(got), soLan)
	}
	got := hutHet(t, cham)
	if len(got) != 4 {
		t.Fatalf("người nghe chậm giữ %d event, muốn đúng 4 (bằng đệm)", len(got))
	}
	// Rơi cái MỚI, giữ cái CŨ. Mặt nào cần đủ 100% phải đọc lại từ store —
	// ghi rõ ở đây để không ai tưởng dashboard luôn thấy trạng thái mới nhất.
	if got[0].Msg != "0" || got[3].Msg != "3" {
		t.Errorf("người nghe chậm giữ [%s..%s], muốn [0..3] (rơi cái đến sau)",
			got[0].Msg, got[3].Msg)
	}
}

// --- Huỷ đăng ký -------------------------------------------------------

func TestHuyDangKyDongKenhVaNgungNhan(t *testing.T) {
	b := NewBus()
	defer b.Close()

	ch, huy := b.Subscribe(8)
	con, huyCon := b.Subscribe(8)
	defer huyCon()

	b.Publish(Event{Type: Info, Msg: "trước"})
	huy()
	b.Publish(Event{Type: Info, Msg: "sau"})

	// Kênh đã huỷ: nhận nốt event cũ rồi ĐÓNG. Các mặt đều dùng `for range ch`
	// (cmd/sagent/main.go:181, internal/tele/tele.go:216) nên không đóng là
	// goroutine của họ kẹt vĩnh viễn.
	got := hutHet(t, ch)
	if len(got) != 1 || got[0].Msg != "trước" {
		t.Errorf("kênh đã huỷ nhận %+v, muốn đúng 1 event \"trước\"", got)
	}
	if !dongChua(ch, time.Second) {
		t.Error("huỷ đăng ký nhưng kênh không đóng")
	}
	// Người còn lại không bị ảnh hưởng bởi việc người kia rút.
	if got := hutHet(t, con); len(got) != 2 {
		t.Errorf("người nghe còn lại nhận %d event, muốn 2", len(got))
	}

	huy() // gọi lần hai: sync.Once phải chặn, nếu không là panic close of closed
	huy()
}

// --- Close -------------------------------------------------------------

// Close phải chịu được mọi thứ tự gọi. Panic ở đây giết cả tiến trình sagent,
// kéo theo mọi phiên agent đang chạy.
func TestCloseKhongPanic(t *testing.T) {
	b := NewBus()
	ch, huy := b.Subscribe(8)

	b.Publish(Event{Type: Info, Msg: "trước khi đóng"})
	b.Close()

	if !dongChua(ch, time.Second) {
		t.Error("Close không đóng kênh của người đang nghe")
	}
	b.Close()                                    // đóng lần hai
	b.Publish(Event{Type: Info, Msg: "sau khi"}) // phát sau khi đóng
	b.Infof("cũng sau khi đóng")
	huy() // huỷ đăng ký sau khi đã Close: kênh đã đóng rồi, không được đóng lại
	huy()
}

// Huỷ đăng ký TRƯỚC rồi mới Close — thứ tự này xảy ra thật khi dashboard ngắt
// kết nối SSE (internal/dash/server.go:657 `defer cancel()`) ngay lúc tiến
// trình đang tắt.
func TestHuyRoiCloseKhongPanic(t *testing.T) {
	b := NewBus()
	a1, huy1 := b.Subscribe(4)
	_, huy2 := b.Subscribe(4)

	huy1()
	if !dongChua(a1, time.Second) {
		t.Error("huỷ đăng ký nhưng kênh không đóng")
	}
	b.Close() // chỉ còn lại kênh của huy2, không được đụng vào kênh đã đóng
	huy2()
}

// --- Chạy đồng thời (chạy với -race) -----------------------------------

// Bus bị nhiều goroutine gọi cùng lúc là chuyện thường ngày: fleet.FanOut phát
// từ mỗi worker, dash mở thêm người nghe mỗi lần có tab mới.
func TestDongThoiPhatVaDangKy(t *testing.T) {
	b := NewBus()
	defer b.Close()

	const soPhat, moiPhat, soNghe = 8, 200, 4
	var wg sync.WaitGroup

	wg.Add(soPhat)
	for i := 0; i < soPhat; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < moiPhat; j++ {
				b.Publish(Event{Type: FlowStep, SessionID: int64(id), Msg: strconv.Itoa(j)})
			}
		}(i)
	}

	// Vừa phát vừa có người vào/ra — đây là chỗ dễ đụng slice b.subs nhất.
	wg.Add(soNghe)
	for i := 0; i < soNghe; i++ {
		go func() {
			defer wg.Done()
			for k := 0; k < 20; k++ {
				ch, huy := b.Subscribe(16)
				for n := 0; n < 3; n++ {
					select {
					case <-ch:
					default:
					}
				}
				huy()
				for range ch { // huy() đã đóng kênh nên vòng này chắc chắn kết thúc
				}
			}
		}()
	}
	wg.Wait()
}

// Close chạy song song với Publish: nếu thiếu cờ closed thì đây là "send on
// closed channel" — panic không cứu được.
func TestDongThoiCloseVaPhat(t *testing.T) {
	b := NewBus()

	var doc sync.WaitGroup
	for i := 0; i < 5; i++ {
		ch, _ := b.Subscribe(8)
		doc.Add(1)
		go func(ch <-chan Event) {
			defer doc.Done()
			for range ch { // kết thúc khi Close đóng kênh
			}
		}(ch)
	}

	var wg sync.WaitGroup
	wg.Add(8)
	for i := 0; i < 4; i++ {
		go func() { defer wg.Done(); b.Close() }()
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				b.Publish(Event{Type: Warning, Msg: "đang tắt"})
			}
		}()
	}
	wg.Wait()
	doc.Wait() // mọi người nghe phải thoát được, không kẹt
}

// --- Hợp đồng với các mặt khác -----------------------------------------

func TestSchemaVersionOnDinh(t *testing.T) {
	// Đổi số này là mọi mặt cũ phải từ chối event. Chỉ đổi khi CỐ Ý phá hợp
	// đồng, và phải sửa cả cmd/sagent/main.go:648 lẫn mặt web cùng lúc.
	if SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, hợp đồng hiện tại là 1", SchemaVersion)
	}

	b := NewBus()
	defer b.Close()
	ch, huy := b.Subscribe(4)
	defer huy()

	truoc := time.Now()
	b.Publish(Event{Type: Info, Msg: "tự điền"})

	dat := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	b.Publish(Event{V: 99, Type: Info, Msg: "tự khai", Time: dat})

	got := hutHet(t, ch)
	if len(got) != 2 {
		t.Fatalf("nhận %d event, muốn 2", len(got))
	}
	if got[0].V != SchemaVersion {
		t.Errorf("V tự điền = %d, muốn %d", got[0].V, SchemaVersion)
	}
	if got[0].Time.Before(truoc) {
		t.Errorf("Time tự điền = %v, phải >= %v", got[0].Time, truoc)
	}
	// Người gọi khai tường minh thì Publish không được ghi đè: store phát lại
	// event cũ vẫn phải giữ nguyên mốc thời gian gốc.
	if got[1].V != 99 || !got[1].Time.Equal(dat) {
		t.Errorf("event tự khai bị sửa: V=%d time=%v", got[1].V, got[1].Time)
	}
}

// Tên chuỗi của Type là một phần hợp đồng với mặt web và 3D. Đổi chuỗi ở đây
// là mặt kia im lặng hiển thị sai — không có lỗi biên dịch nào bắt được.
func TestTenLoaiEventKhongDoi(t *testing.T) {
	muon := map[Type]string{
		ProfileCreated: "profile.created",
		ProfileRemoved: "profile.removed",
		ClonesCreated:  "clones.created",
		ClonesCleaned:  "clones.cleaned",
		SessionStarted: "session.started",
		SessionStopped: "session.stopped",
		WorktreeAdded:  "worktree.added",
		WorktreeKept:   "worktree.kept",
		WorktreeGone:   "worktree.removed",
		FlowStarted:    "flow.started",
		FlowStep:       "flow.step",
		FlowWaiting:    "flow.waiting_approval",
		FlowApproved:   "flow.approved",
		FlowRejected:   "flow.rejected",
		FlowDone:       "flow.completed",
		FlowFailed:     "flow.failed",
		Warning:        "warning",
		Failure:        "failure",
		Info:           "info",
	}
	for got, want := range muon {
		if string(got) != want {
			t.Errorf("loại event = %q, hợp đồng là %q", string(got), want)
		}
	}
	// Trùng chuỗi giữa hai hằng khác nhau thì mặt web không phân biệt nổi.
	daGap := make(map[string]bool, len(muon))
	for k := range muon {
		if daGap[string(k)] {
			t.Errorf("hai hằng cùng dùng chuỗi %q", string(k))
		}
		daGap[string(k)] = true
	}
}

// dash/server.go:673 marshal thẳng Event ra SSE, nên tên trường JSON là thứ
// mặt web nhìn thấy. Đổi tên trường = gãy dashboard.
func TestHopDongJSON(t *testing.T) {
	b, err := json.Marshal(Event{
		V: SchemaVersion, Type: SessionStarted, Time: time.Unix(0, 0).UTC(),
		Addr: "claude:phu#1", SessionID: 12, Msg: "đã chạy",
		Detail: map[string]string{"path": "C:\\wt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, ten := range []string{"v", "type", "time", "addr", "session_id", "msg", "detail"} {
		if _, ok := m[ten]; !ok {
			t.Errorf("JSON thiếu trường %q — mặt web đọc trường này", ten)
		}
	}
	if len(m) != 7 {
		t.Errorf("JSON có %d trường, hợp đồng là 7: %s", len(m), b)
	}

	// Event rỗng: các trường omitempty phải biến mất để dòng SSE khỏi phình.
	b2, err := json.Marshal(Event{V: SchemaVersion, Type: Info, Msg: "x", Time: time.Unix(0, 0).UTC()})
	if err != nil {
		t.Fatal(err)
	}
	for _, ten := range []string{"addr", "session_id", "detail"} {
		if strings.Contains(string(b2), `"`+ten+`"`) {
			t.Errorf("trường %q phải omitempty, nhưng vẫn có trong %s", ten, b2)
		}
	}
}

// Event chạy thẳng ra dashboard qua SSE, mà dashboard KHÔNG được thấy secret
// (nói rõ ở doc comment của Event). Chặn ngay từ hình dạng struct: ai thêm
// trường tên kiểu Token/APIKey vào Event sẽ đỏ test chứ không lọt review.
func TestKhongCoTruongBiMat(t *testing.T) {
	cam := []string{"token", "secret", "apikey", "password", "credential", "cookie"}
	tp := reflect.TypeOf(Event{})
	for i := 0; i < tp.NumField(); i++ {
		f := tp.Field(i)
		soi := strings.ToLower(f.Name + " " + f.Tag.Get("json"))
		for _, tu := range cam {
			if strings.Contains(soi, tu) {
				t.Errorf("Event có trường %q dính từ cấm %q — secret không được ra tới dashboard", f.Name, tu)
			}
		}
	}
}

func TestHamTienDung(t *testing.T) {
	b := NewBus()
	defer b.Close()
	ch, huy := b.Subscribe(8)
	defer huy()

	b.Infof("có %d phiên", 3)
	b.Warnf("worktree %s còn việc dở", "phu-1")
	b.Failuref("không mở được %s: %v", "db", "khoá")

	got := hutHet(t, ch)
	if len(got) != 3 {
		t.Fatalf("nhận %d event, muốn 3", len(got))
	}
	muon := []struct {
		t   Type
		msg string
	}{
		{Info, "có 3 phiên"},
		{Warning, "worktree phu-1 còn việc dở"},
		{Failure, "không mở được db: khoá"},
	}
	for i, m := range muon {
		if got[i].Type != m.t || got[i].Msg != m.msg {
			t.Errorf("event %d = (%s, %q), muốn (%s, %q)", i, got[i].Type, got[i].Msg, m.t, m.msg)
		}
	}
}

// --- Khiếm khuyết đã biết ----------------------------------------------

// GHI NHẬN HIỆN TRẠNG (chưa sửa vì lượt này chỉ được thêm file test):
// Subscribe KHÔNG kiểm cờ closed, nên đăng ký sau khi Close sẽ nhận một kênh
// vĩnh viễn im lặng và KHÔNG BAO GIỜ đóng. Người gọi dùng `for range ch`
// (cmd/sagent/main.go:181, internal/tele/tele.go:216) sẽ kẹt goroutine, và hàm
// đóng của tele còn chờ thêm 20s vô ích trước khi bỏ cuộc.
//
// Cách sửa khi được đụng events.go: trong Subscribe, nếu b.closed thì close(ch)
// ngay rồi trả về. Lúc đó test này phải đổi thành "kênh đóng ngay".
func TestSubscribeSauCloseTraKenhKhongBaoGioDong(t *testing.T) {
	b := NewBus()
	b.Close()

	ch, huy := b.Subscribe(4)
	defer huy()
	b.Publish(Event{Type: Info, Msg: "rơi vào hư không"})

	if dongChua(ch, 200*time.Millisecond) {
		t.Fatal("kênh đã đóng — khiếm khuyết đã được sửa, hãy cập nhật lại bài test này")
	}
	if got := hutHet(t, ch); len(got) != 0 {
		t.Errorf("bus đã đóng mà vẫn phát ra %d event", len(got))
	}
}
