package api

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/fleet"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// TRẦN ĐỒNG THỜI — kiểm ở CHỖ GỌI, không phải chỗ tính.
//
// internal/fleet/tran_test.go chứng minh phép tính đúng. Bài này chứng minh nó
// ĐƯỢC CẮM VÀO `FleetStart`. Hai chuyện khác nhau, và đúng chỗ nối là chỗ dự án
// này đã vấp: `plugin.list` có đủ hợp đồng + CLI + endpoint mà không mặt nào
// gọi tới, test vẫn xanh vì nó chỉ hỏi từng mảnh.
//
// Đo 22/08: gỡ khối `a.xetTran(...)` trong FleetStart ra và trả về khối trần
// chung cũ thì BA bài dưới đỏ (TuChoiKhiHetChoTheoHoSo, CatBotThiPhaiNoiTranNao,
// TranChungVanLaTranNgoaiCung) — trong khi internal/fleet và internal/config
// vẫn xanh sạch. Đó chính là lý do bộ này tồn tại tách khỏi hai gói kia.

// themPhienSong ghi một phiên ĐANG CHẠY thật: PID của chính tiến trình test,
// nên `store.Running` (vốn reap PID chết) giữ nó lại.
func themPhienSong(t *testing.T, a *API, prov, acc string) int64 {
	t.Helper()
	id, err := a.db.AddSession(store.Session{
		Provider: prov, Account: acc, Dir: "d", PID: os.Getpid(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// batEvent thu mọi event phát ra trong lúc chạy f.
func batEvent(t *testing.T, a *API, f func()) []events.Event {
	t.Helper()
	ch, thoi := a.bus.Subscribe(64)
	defer thoi()
	f()
	// Bus phát bất đồng bộ; vét cho tới khi im 200ms.
	var out []events.Event
	for {
		select {
		case e := <-ch:
			out = append(out, e)
		case <-time.After(200 * time.Millisecond):
			return out
		}
	}
}

// Ca đắt nhất, và là lý do tính năng này tồn tại: trần chung còn thừa chỗ
// (4 phiên, đang chạy 1) nhưng TÀI KHOẢN thì hết. Trần chung một mình cho qua,
// rồi cả bốn phiên đốt chung một hạn mức.
func TestFleetStartTuChoiKhiHetChoTheoHoSo(t *testing.T) {
	a := moAPI(t)
	a.cfg.Policy.MaxParallelSessions = 4
	a.cfg.Policy.Tran.HarnessMacDinh = 3
	a.cfg.Policy.Tran.ProviderMacDinh = 3
	a.cfg.Policy.Tran.HoSoMacDinh = 2
	a.cfg.Policy.Tran.HoSo = map[string]int{"claude:tns": 1}
	themPhienSong(t, a, "claude", "tns")

	_, err := a.FleetStart(FleetRequest{
		Addr: Addr{"claude", "tns"}, Copies: 3,
		Args: []string{"-p", "việc gì đó"},
	})
	if err == nil {
		t.Fatal("trần hồ sơ đã đầy mà FleetStart vẫn cho chạy")
	}
	msg := err.Error()
	// In ra để `go test -v` đọc được NGUYÊN VĂN. Câu này là sản phẩm chính của
	// lượt việc, không phải phụ phẩm — ai sửa nó phải nhìn thấy nó đổi thành gì.
	t.Logf("nguyên văn câu từ chối:\n%s", msg)
	// Câu từ chối phải nói ĐỦ BA THỨ, vì thiếu thứ nào người vận hành lúc hai
	// giờ sáng cũng phải đi tìm bằng tay:
	for _, phai := range []string{
		"hồ sơ claude:tns",           // trần NÀO chặn
		"trần 1, đang chạy 1",        // và nó đang ở đâu
		"chung: trần 4, đang chạy 1", // trần chung vẫn còn chỗ — nên đừng đổ cho nó
		"policy.tran.ho_so",          // khoá cần sửa nếu muốn nới
		"tài khoản khác",             // cách đi tiếp không tốn hạn mức
	} {
		if !strings.Contains(msg, phai) {
			t.Errorf("câu từ chối thiếu %q.\nNguyên văn:\n%s", phai, msg)
		}
	}
	// Và KHÔNG được đổ tội cho trần chung: nó còn 3 chỗ.
	if strings.Contains(msg, "chung đã đầy") {
		t.Errorf("nói sai trần chặn — chung vẫn còn chỗ.\nNguyên văn:\n%s", msg)
	}
}

// Còn chỗ nhưng ít hơn số xin: phải CẮT và nói ra, không im lặng hạ số.
//
// Args để rỗng nên FanOut từ chối ngay ở bước "thiếu lệnh headless" — không
// tiến trình nào được bật, không hạn mức nào bị đốt, mà đoạn xét trần thì đã
// chạy xong rồi.
func TestFleetStartCatBotThiPhaiNoiTranNao(t *testing.T) {
	a := moAPI(t)
	a.cfg.Policy.MaxParallelSessions = 4
	a.cfg.Policy.Tran.HarnessMacDinh = 3
	a.cfg.Policy.Tran.ProviderMacDinh = 3
	a.cfg.Policy.Tran.HoSoMacDinh = 2

	var err error
	evs := batEvent(t, a, func() {
		_, err = a.FleetStart(FleetRequest{Addr: Addr{"claude", "phu"}, Copies: 4})
	})
	if err == nil {
		t.Fatal("thiếu args mà FanOut vẫn chạy — bài này không còn chạy khô nữa")
	}
	var canh string
	for _, e := range evs {
		if strings.Contains(e.Msg, "TRẦN ĐỒNG THỜI") {
			canh = e.Msg
		}
	}
	if canh == "" {
		t.Fatalf("cắt 4 xuống 2 mà không có cảnh báo nào. Event thu được: %v", tomTatEvent(evs))
	}
	for _, phai := range []string{
		"cắt 4 phiên xuống 2",
		"hồ sơ claude:phu",
		"còn 2 chỗ",
		"chia sang tài khoản khác",
	} {
		if !strings.Contains(canh, phai) {
			t.Errorf("câu cắt bớt thiếu %q.\nNguyên văn:\n%s", phai, canh)
		}
	}
}

// Trần chung vẫn là trần NGOÀI CÙNG: ba trần theo chiều nới rộng bao nhiêu cũng
// không vượt được nó. Đây là điều dễ hỏng nhất khi thêm tầng mới — thêm xong
// rồi quên `and` nó lại với tầng cũ.
func TestTranChungVanLaTranNgoaiCung(t *testing.T) {
	a := moAPI(t)
	a.cfg.Policy.MaxParallelSessions = 2
	a.cfg.Policy.Tran.HarnessMacDinh = 99
	a.cfg.Policy.Tran.ProviderMacDinh = 99
	a.cfg.Policy.Tran.HoSoMacDinh = 99
	// Hai phiên của provider KHÁC — ba trần theo chiều đều không đếm chúng,
	// chỉ trần chung thấy.
	themPhienSong(t, a, "codex", "chinh")
	themPhienSong(t, a, "grok", "chinh")

	_, err := a.FleetStart(FleetRequest{
		Addr: Addr{"claude", "tns"}, Copies: 2,
		Args: []string{"-p", "việc gì đó"},
	})
	if err == nil {
		t.Fatal("trần chung đã đầy mà vẫn cho chạy")
	}
	if !strings.Contains(err.Error(), "chung đã đầy (trần 2, đang chạy 2)") {
		t.Errorf("không gọi tên trần chung.\nNguyên văn:\n%s", err.Error())
	}
}

// Tắt hết trần thì lượt chạy KHÔNG bị chặn và cũng KHÔNG bị cảnh báo thừa.
// Cảnh báo mỗi lượt là cách nhanh nhất để người vận hành học cách bỏ qua nó.
func TestTatHetTranThiKhongChanVaKhongNoiGi(t *testing.T) {
	a := moAPI(t)
	a.cfg.Policy.MaxParallelSessions = 0
	a.cfg.Policy.Tran.HarnessMacDinh = 0
	a.cfg.Policy.Tran.ProviderMacDinh = 0
	a.cfg.Policy.Tran.HoSoMacDinh = 0
	a.cfg.Policy.Tran.Harness = nil
	a.cfg.Policy.Tran.Provider = nil
	a.cfg.Policy.Tran.HoSo = nil
	for i := 0; i < 5; i++ {
		themPhienSong(t, a, "claude", "tns")
	}

	var err error
	evs := batEvent(t, a, func() {
		_, err = a.FleetStart(FleetRequest{Addr: Addr{"claude", "tns"}, Copies: 9})
	})
	// Vẫn hỏng, nhưng phải hỏng vì THIẾU ARGS chứ không phải vì trần.
	if err == nil || !strings.Contains(err.Error(), "thiếu lệnh headless") {
		t.Fatalf("mong lỗi thiếu args, được: %v", err)
	}
	for _, e := range evs {
		if strings.Contains(e.Msg, "TRẦN ĐỒNG THỜI") || strings.Contains(e.Msg, "hết chỗ") {
			t.Errorf("mọi trần đã tắt mà vẫn nói chuyện trần: %q", e.Msg)
		}
	}
}

// Bộ số MẶC ĐỊNH (không ai khai gì) phải tự ngăn được đúng sự cố đã đẻ ra tính
// năng này: bốn phiên dồn vào một tài khoản. Bài này canh chính con số mặc
// định — hạ `ho_so_mac_dinh` lên 4 là nó đỏ.
func TestMacDinhDaChanDonBonPhienVaoMotTaiKhoan(t *testing.T) {
	a := moAPI(t) // cfg nạp từ config.Default(), không sửa gì
	k, ok := a.xetTran(Addr{"claude", "tns"}, 4)
	if !ok {
		t.Fatal("mặc định mà không có trần nào bật")
	}
	if k.Cap >= 4 {
		t.Fatalf("mặc định vẫn cho 4 phiên dồn vào một tài khoản (cấp %d)", k.Cap)
	}
	if k.Cap < 2 {
		t.Fatalf("mặc định siết quá tay, `fleet` hết ý nghĩa song song (cấp %d)", k.Cap)
	}
	// Và trần chặn phải là trần HỒ SƠ, không phải trần chung — nếu là trần
	// chung thì tính năng này chưa làm được gì mới.
	chat := k.Chat()
	if len(chat) == 0 || chat[0].Loai != "hồ sơ" {
		t.Fatalf("mặc định: mức chật nhất phải là hồ sơ, được %v", chat)
	}
}

// Bảng trần đọc từ config phải tới được gói fleet NGUYÊN VẸN. Quên chép một
// trường trong `TranDongThoi()` thì trần đó im lặng không có tác dụng — đúng
// kiểu hỏng mà không ai báo.
func TestConfigToiFleetKhongRoiTruongNao(t *testing.T) {
	a := moAPI(t)
	a.cfg.Policy.MaxParallelSessions = 7
	a.cfg.Policy.Tran.HarnessMacDinh = 6
	a.cfg.Policy.Tran.ProviderMacDinh = 5
	a.cfg.Policy.Tran.HoSoMacDinh = 4
	a.cfg.Policy.Tran.Harness = map[string]int{"node": 3}
	a.cfg.Policy.Tran.Provider = map[string]int{"codex": 2}
	a.cfg.Policy.Tran.HoSo = map[string]int{"codex:chinh": 1}
	a.cfg.Policy.Tran.ThuocHarness = map[string]string{"codex": "node", "grok": "node"}

	got := a.TranDongThoi()
	want := fleet.Tran{
		Chung: 7, HarnessMacDinh: 6, ProviderMacDinh: 5, HoSoMacDinh: 4,
		Harness:      map[string]int{"node": 3},
		Provider:     map[string]int{"codex": 2},
		HoSo:         map[string]int{"codex:chinh": 1},
		ThuocHarness: map[string]string{"codex": "node", "grok": "node"},
	}
	if got.Chung != want.Chung || got.HarnessMacDinh != want.HarnessMacDinh ||
		got.ProviderMacDinh != want.ProviderMacDinh || got.HoSoMacDinh != want.HoSoMacDinh {
		t.Errorf("số mặc định rơi: %+v", got)
	}
	if got.Harness["node"] != 3 || got.Provider["codex"] != 2 || got.HoSo["codex:chinh"] != 1 {
		t.Errorf("bảng khai riêng rơi: %+v", got)
	}
	if got.HarnessCua("grok") != "node" {
		t.Errorf("bảng thuoc_harness rơi: grok -> %q", got.HarnessCua("grok"))
	}
}

// Đường FLOW cũng phải chịu đúng bốn trần đó.
//
// Bước `agent` của flow đi qua `agentBridge.RunAgents` → `FleetStart`, nên trần
// áp cho cả hai đường. Bài này canh cái nối ấy: ai đó cho RunAgents tự bật phiên
// (bỏ qua FleetStart) là flow lách được trần mà `sagent fleet` thì không — hai
// đường điều khiển bất đồng về chính sách, đúng lớp sự cố dự án này chống.
func TestDuongFlowChiuChungTranVoiFleet(t *testing.T) {
	a := moAPI(t)
	a.cfg.Policy.MaxParallelSessions = 9
	a.cfg.Policy.Tran.HoSoMacDinh = 1
	themPhienSong(t, a, "claude", "tns")

	b := agentBridge{a: a, fallback: Addr{"claude", "tns"}}
	_, err := b.RunAgents(context.Background(), "claude:tns", "", "việc gì đó", 2, false, false)
	if err == nil {
		t.Fatal("bước agent của flow lách được trần hồ sơ")
	}
	if !strings.Contains(err.Error(), "hồ sơ claude:tns đã đầy") {
		t.Errorf("flow bị chặn vì lý do khác, không phải trần.\nNguyên văn:\n%s", err.Error())
	}
}

func tomTatEvent(evs []events.Event) []string {
	out := make([]string, 0, len(evs))
	for _, e := range evs {
		out = append(out, e.Msg)
	}
	return out
}
