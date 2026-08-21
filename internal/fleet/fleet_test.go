package fleet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/nhatky"
	"github.com/trantiendevweb/switch-agent-pro/internal/process"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// TestHelperProcess KHÔNG phải test thật — nó đóng vai CLI của agent khi được
// fleet spawn ra. Đây là mẫu chuẩn của Go để có một tiến trình con thật mà
// không cần cài thêm gì trên máy chạy test.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("SAGENT_FAKE_AGENT") != "1" {
		return // chạy trong bộ test bình thường: không làm gì cả
	}
	// Sống đủ lâu để bài test kịp thấy "đang chạy" rồi dừng.
	time.Sleep(20 * time.Second)
}

// fakeAgent trỏ Command() vào chính test binary.
type fakeAgent struct{ base string }

func (fakeAgent) Name() string                         { return "fake" }
func (fakeAgent) Version() (string, error)             { return "fake 0.0.0", nil }
func (fakeAgent) TachDuocTaiKhoan() bool               { return true }
func (fakeAgent) EnvVar() string                       { return "FAKE_CONFIG_DIR" }
func (fakeAgent) Command() (string, error)             { return os.Executable() }
func (fakeAgent) HeadlessArgs(p string) []string       { return []string{"-p", p} }
func (fakeAgent) ModelArgs(string) []string            { return nil }
func (fakeAgent) PrivateFiles() []string               { return []string{".credentials.json", ".claude.json"} }
func (fakeAgent) SharedKeys() []string                 { return []string{"projects"} }
func (f fakeAgent) BaseDir() string                    { return f.base }
func (fakeAgent) IdentitySource() string               { return "" }
func (fakeAgent) Identity(string) string               { return "" }
func (fakeAgent) HasToken(string) bool                 { return true }
func (fakeAgent) TokenExpiry(string) (time.Time, bool) { return time.Time{}, false }
func (fakeAgent) Verify() []provider.Check             { return nil }

// setup dựng HOME giả + hồ sơ gốc có token + một store tạm.
func setup(t *testing.T) (*store.DB, *events.Bus, fakeAgent) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SAGENT_FAKE_AGENT", "1") // để tiến trình con biết mình là agent giả

	base := filepath.Join(home, "fakebase")
	if err := os.MkdirAll(filepath.Join(base, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	prof := filepath.Join(home, ".ai-accounts", "fake", "phu")
	if err := os.MkdirAll(prof, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{".credentials.json", ".claude.json"} {
		if err := os.WriteFile(filepath.Join(prof, n), []byte(`{}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	db, err := store.OpenAt(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	// Đăng ký SAU t.TempDir() nên chạy TRƯỚC khi thư mục tạm bị xoá (cleanup
	// chạy ngược thứ tự đăng ký). Cần vậy vì trên Windows tiến trình con còn
	// giữ fleet.log thì không xoá được thư mục.
	t.Cleanup(func() {
		stopAll(t, db)
		db.Close()
	})
	bus := events.NewBus()
	t.Cleanup(bus.Close)
	return db, bus, fakeAgent{base: base}
}

// stopAll giết mọi phiên còn sống VÀ ĐỢI chúng chết hẳn — không đợi thì
// Windows còn giữ file log và thư mục tạm không xoá được.
func stopAll(t *testing.T, db *store.DB) {
	t.Helper()
	list, _ := db.Running()
	for _, s := range list {
		_ = process.Kill(s.PID)
		_ = db.SetState(s.ID, store.StateStopped)
	}
	deadline := time.Now().Add(15 * time.Second)
	for _, s := range list {
		for time.Now().Before(deadline) && process.IsAlive(s.PID) {
			time.Sleep(100 * time.Millisecond)
		}
		if process.IsAlive(s.PID) {
			t.Errorf("PID %d không chịu chết sau khi Kill — sẽ để lại tiến trình mồ côi", s.PID)
		}
	}
}

func TestFanOutStartsAndRecordsSessions(t *testing.T) {
	db, bus, a := setup(t)

	_, err := FanOut(db, bus, a, "phu", Opts{Copies: 3}, []string{"-test.run=TestHelperProcess"})
	if err != nil {
		t.Fatal(err)
	}

	list, err := db.Running()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("muốn 3 phiên đang chạy, được %d", len(list))
	}

	seen := map[int]bool{}
	for _, s := range list {
		if !process.IsAlive(s.PID) {
			t.Fatalf("phiên #%d báo chạy nhưng PID %d đã chết", s.ID, s.PID)
		}
		if seen[s.Clone] {
			t.Fatalf("trùng số bản clone %d — hai phiên dùng chung config dir", s.Clone)
		}
		seen[s.Clone] = true
		// Mỗi phiên PHẢI có config dir riêng, nếu không sẽ đua ghi .claude.json.
		if _, err := os.Stat(filepath.Join(s.Dir, ".credentials.json")); err != nil {
			t.Fatalf("phiên #%d thiếu credential riêng: %v", s.ID, err)
		}
		if _, err := os.Stat(s.Log); err != nil {
			t.Fatalf("phiên #%d không có file log: %v", s.ID, err)
		}
	}
}

// Dừng phiên thì `status` phải phản ánh ngay, không được báo sống thứ đã chết.
func TestStoppedSessionDisappearsFromRunning(t *testing.T) {
	db, bus, a := setup(t)

	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 2}, []string{"-test.run=TestHelperProcess"}); err != nil {
		t.Fatal(err)
	}
	list, _ := db.Running()
	if len(list) != 2 {
		t.Fatalf("muốn 2 phiên, được %d", len(list))
	}

	if err := process.Kill(list[0].PID); err != nil {
		t.Fatalf("không giết được tiến trình: %v", err)
	}
	// Đợi hệ điều hành dọn xong.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && process.IsAlive(list[0].PID) {
		time.Sleep(200 * time.Millisecond)
	}

	after, err := db.Running()
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 {
		t.Fatalf("phiên đã bị giết vẫn còn trong danh sách chạy: %d phiên", len(after))
	}
}

// Thiếu lệnh headless thì phải báo lỗi RÕ RÀNG chứ không bật một đống phiên vô dụng.
func TestFanOutRefusesWithoutCommand(t *testing.T) {
	db, bus, a := setup(t)
	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 2}, nil); err == nil {
		t.Fatal("thiếu lệnh mà vẫn chạy")
	}
	list, _ := db.Running()
	if len(list) != 0 {
		t.Fatalf("không được bật phiên nào khi thiếu lệnh, có %d", len(list))
	}
}

// --worktree ở nơi không phải git repo: phải chết SỚM, trước khi bật phiên nào.
func TestWorktreeRefusesOutsideGitRepo(t *testing.T) {
	db, bus, a := setup(t)
	t.Chdir(t.TempDir()) // thư mục trống, không phải repo

	_, err := FanOut(db, bus, a, "phu", Opts{Copies: 2, Worktree: true}, []string{"-test.run=TestHelperProcess"})
	if err == nil {
		t.Fatal("không phải git repo mà vẫn chạy --worktree")
	}
	list, _ := db.Running()
	if len(list) != 0 {
		t.Fatalf("phải chết trước khi bật phiên nào, mà đã bật %d", len(list))
	}
}

func (fakeAgent) ArgsTuDuyetQuyen() ([]string, bool) { return nil, false }

func (fakeAgent) ArgsThuMuc(string) []string { return nil }

func (fakeAgent) ArgsHoSo(string) []string { return nil }

func (fakeAgent) DocKetQua(string) (provider.KetQua, bool) { return provider.KetQua{}, false }

// khoTokenChung là adapter giả cho lớp provider giữ token ở kho dùng chung toàn
// máy (Antigravity: Windows Credential Manager). Dấu hiệu ĐO ĐƯỢC của lớp này
// là `PrivateFiles()` rỗng — không có file nào để chép sang bản clone.
type khoTokenChung struct{ fakeAgent }

func (khoTokenChung) PrivateFiles() []string { return nil }

// gomEvent hút hết event đang nằm trong kênh ra một lát cắt.
func gomEvent(ch <-chan events.Event) []events.Event {
	var out []events.Event
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

func coCauChepRa(evs []events.Event) bool {
	for _, e := range evs {
		if strings.Contains(e.Msg, "chép ra") {
			return true
		}
	}
	return false
}

// Provider KHÔNG có file riêng: fleet không được nói "Token được chép ra N chỗ"
// — `profile.Clone` chỉ chép những gì `PrivateFiles()` khai, mà ở đây là rỗng,
// nên câu đó là SAI SỰ THẬT.
func TestFanOutKhongNoiChepTokenKhiAdapterKhongCoFileRieng(t *testing.T) {
	db, bus, a := setup(t)
	ch, huy := bus.Subscribe(256)
	defer huy()

	if _, err := FanOut(db, bus, khoTokenChung{a}, "phu", Opts{Copies: 1}, []string{"-test.run=TestHelperProcess"}); err != nil {
		t.Fatal(err)
	}

	evs := gomEvent(ch)
	if coCauChepRa(evs) {
		t.Fatalf("adapter khai PrivateFiles() rỗng mà fleet vẫn nói token bị chép ra: %v", evs)
	}
	// Và phải nói ĐÚNG sự thật thay thế, chứ không phải im lặng bỏ qua.
	var coCauDung bool
	for _, e := range evs {
		if strings.Contains(e.Msg, "kho dùng chung toàn máy") && strings.Contains(e.Msg, "một danh tính") {
			coCauDung = true
		}
	}
	if !coCauDung {
		t.Fatalf("thiếu câu nói rõ token nằm ở kho dùng chung và mọi phiên chung một danh tính: %v", evs)
	}
}

// Provider CÓ file riêng: câu cảnh báo cũ phải còn nguyên — token thật sự bị
// nhân ra N bản, và ĐÃ ĐO 20/08 rằng nhà cung cấp XOAY VÒNG refresh token, nên
// MỘT bản refresh là các bản kia chết.
//
// Đo 21/08 (ô Đ5) đã đóng nốt nửa còn lại: hai clone cùng refresh một lúc thì
// ĐÚNG MỘT bản thắng, bản thua nhận "OAuth session expired and could not be
// refreshed" rồi tự ghi đè file token của mình thành RỖNG.
func TestFanOutVanCanhBaoChepTokenKhiAdapterCoFileRieng(t *testing.T) {
	db, bus, a := setup(t)
	ch, huy := bus.Subscribe(256)
	defer huy()

	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 2}, []string{"-test.run=TestHelperProcess"}); err != nil {
		t.Fatal(err)
	}

	evs := gomEvent(ch)
	if !coCauChepRa(evs) {
		t.Fatalf("adapter có file riêng mà mất cảnh báo token bị chép ra: %v", evs)
	}
	// Cảnh báo phải nói ĐÚNG SỐ BẢN và nói ra HẬU QUẢ đã đo được.
	//
	// Bản cũ của bài này đòi giữ chữ "CHƯA ĐO" — đúng vào lúc đó. Ngày 20/08 thì
	// đo được rồi: nhà cung cấp XOAY VÒNG refresh token, nên một bản refresh là
	// các bản còn lại chết ngay. Giữ chữ "chưa đo" sau khi đã đo là nói dối theo
	// hướng khiêm tốn, mà người vận hành thì mất đúng thông tin cần biết.
	var noiDuHau bool
	for _, e := range evs {
		if strings.Contains(e.Msg, "chép ra 2 chỗ") && strings.Contains(e.Msg, "XOAY VÒNG") {
			noiDuHau = true
		}
	}
	if !noiDuHau {
		t.Fatalf("cảnh báo phải nói đúng số bản và nói ra hậu quả của việc xoay vòng token: %v", evs)
	}
}

// NangLuc: adapter GIẢ nên khai CHƯA ĐO hết — nó không đo được gì trên máy nào
// cả. Khai bừa "làm được" ở đây là bộ conformance của gói provider bắt ngay.
func (fakeAgent) NangLuc() []provider.NangLuc {
	out := make([]provider.NangLuc, 0, len(provider.MoiNangLuc))
	for _, m := range provider.MoiNangLuc {
		out = append(out, provider.Chua(m.Khoa, "adapter giả trong test — không đo gì"))
	}
	return out
}

// Chạy nhiều bản trên MỘT tài khoản có một cách hỏng mà đồng bộ ngược không
// cứu được: hai bản đang chạy, một bản tới mốc refresh và xoay token đi, bản
// kia chết GIỮA CHỪNG. Không có chỗ nào để chen vào mà đồng bộ.
//
// Đo 20/08 (xem docs/DO-LUONG.md): nhà cung cấp xoay vòng refresh token thật.
// Người vận hành phải biết chuyện này TRƯỚC khi chia việc, không phải sau khi
// một lượt chạy dài chết ở giữa.
func TestCanhBaoKhiNhieuBanCungChayMotTaiKhoan(t *testing.T) {
	db, bus, a := setup(t)
	ch, huy := bus.Subscribe(256)
	defer huy()
	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 2}, []string{"-test.run=TestHelperProcess"}); err != nil {
		t.Fatal(err)
	}
	var coCanhBao bool
	for _, e := range gomEvent(ch) {
		if strings.Contains(e.Msg, "GIỮA CHỪNG") && strings.Contains(e.Msg, "NHIỀU TÀI KHOẢN") {
			coCanhBao = true
		}
	}
	if !coCanhBao {
		t.Error("chạy 2 bản trên một tài khoản mà không cảnh báo chuyện token bị xoay giữa chừng")
	}
}

// Và chạy MỘT bản thì KHÔNG được cảnh báo — câu đó chỉ đúng khi có nhiều bản
// cùng chạy. Cảnh báo thừa lặp lại mỗi lần sẽ bị đọc lướt, rồi tới lúc nó đúng
// thì không ai còn đọc nữa.
func TestKhongCanhBaoThuaKhiChiMotBan(t *testing.T) {
	db, bus, a := setup(t)
	ch, huy := bus.Subscribe(256)
	defer huy()
	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 1}, []string{"-test.run=TestHelperProcess"}); err != nil {
		t.Fatal(err)
	}
	for _, e := range gomEvent(ch) {
		if strings.Contains(e.Msg, "GIỮA CHỪNG") {
			t.Errorf("một bản mà vẫn cảnh báo chuyện nhiều bản: %q", e.Msg)
		}
	}
}

// ---------------------------- nhật ký phiên ----------------------------
//
// VẤN ĐỀ THẬT, đo 21/08/2026: phiên fleet kết thúc thì mọi thứ agent nói và làm
// BỐC HƠI. Hai phiên (#167, #169) báo `xong` với 0 commit và worktree sạch
// trơn; mất ~15 phút truy nguyên mới ra nguyên nhân là thiếu cờ
// --tu-duyet-quyen. Một phiên khác (#172) sửa 4 file rồi chết vì hết hạn mức,
// cũng không để lại dấu vết.
//
// Nguyên nhân đo được: đường dẫn log cũ là `<thư mục clone>/fleet.log` — chỉ
// phụ thuộc SỐ BẢN CLONE. Lượt sau cắt trắng nhật ký lượt trước, và `sagent
// clean` xoá nguyên thư mục.

// Nhật ký phải nằm ngoài thư mục clone, và phải có khối tiêu đề đọc được.
func TestNhatKyNamNgoaiThuMucCloneVaMangDongLenh(t *testing.T) {
	db, bus, a := setup(t)

	args := []string{"-test.run=TestHelperProcess"}
	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 2}, args); err != nil {
		t.Fatal(err)
	}
	list, _ := db.Running()
	if len(list) != 2 {
		t.Fatalf("muốn 2 phiên, được %d", len(list))
	}
	for _, s := range list {
		if s.Log == "" {
			t.Fatalf("phiên #%d không ghi đường dẫn nhật ký vào sổ", s.ID)
		}
		if !strings.HasPrefix(s.Log, nhatky.Root()) {
			t.Errorf("nhật ký phiên #%d = %q, không nằm trong %q", s.ID, s.Log, nhatky.Root())
		}
		// Chỗ CŨ. Nằm ở đây thì `sagent clean` xoá mất, và lượt sau ghi đè.
		if strings.HasPrefix(s.Log, s.Dir) {
			t.Errorf("nhật ký phiên #%d nằm trong thư mục clone %q — `sagent clean` sẽ xoá mất",
				s.ID, s.Dir)
		}
		raw, err := os.ReadFile(s.Log)
		if err != nil {
			t.Fatalf("phiên #%d không đọc lại được nhật ký: %v", s.ID, err)
		}
		got := string(raw)
		if !strings.HasPrefix(got, nhatky.MocDau) {
			t.Errorf("nhật ký phiên #%d thiếu khối tiêu đề:\n%s", s.ID, got)
		}
		// Dòng lệnh ĐÃ DỰNG XONG — đây là thứ trả lời được ca #167.
		if !strings.Contains(got, "-test.run=TestHelperProcess") {
			t.Errorf("nhật ký phiên #%d không mang dòng lệnh đã dựng:\n%s", s.ID, got)
		}
		if !strings.Contains(got, s.Addr()) {
			t.Errorf("nhật ký phiên #%d không mang địa chỉ %q:\n%s", s.ID, s.Addr(), got)
		}
	}
}

// ĐÂY là bài đo lại đúng ca #167/#169: hai LƯỢT fleet liên tiếp trên cùng một
// bản clone. Cả hai nhật ký phải còn đọc được.
func TestHaiLuotFleetKhongDeLenNhatKyCuaNhau(t *testing.T) {
	db, bus, a := setup(t)
	args := []string{"-test.run=TestHelperProcess"}

	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 1}, args); err != nil {
		t.Fatal(err)
	}
	dau, _ := db.Running()
	if len(dau) != 1 {
		t.Fatalf("lượt 1: muốn 1 phiên, được %d", len(dau))
	}
	logMot := dau[0].Log

	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 1}, args); err != nil {
		t.Fatal(err)
	}
	sau, _ := db.Running()
	if len(sau) != 2 {
		t.Fatalf("lượt 2: muốn 2 phiên đang chạy, được %d", len(sau))
	}
	var logHai string
	for _, s := range sau {
		if s.Log != logMot {
			logHai = s.Log
		}
	}
	if logHai == "" {
		t.Fatal("hai lượt trên cùng bản clone dùng CHUNG một file nhật ký — " +
			"lượt sau xoá tang chứng của lượt trước, đúng cách #169 xoá #167")
	}
	for _, p := range []string{logMot, logHai} {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("nhật ký %q không còn đọc được: %v", p, err)
		}
		if st.Size() == 0 {
			t.Fatalf("nhật ký %q bị cắt trắng", p)
		}
	}
}

// Nhật ký của phiên ĐANG CHẠY không được phép bị lượt sau dọn mất.
func TestLuotSauKhongDonNhatKyCuaPhienDangChay(t *testing.T) {
	db, bus, a := setup(t)
	args := []string{"-test.run=TestHelperProcess"}

	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 1}, args); err != nil {
		t.Fatal(err)
	}
	dang, _ := db.Running()
	giu := dang[0].Log

	// Đẩy thư mục nhật ký vượt trần bằng cách dọn với ngân sách 0 file.
	kq, err := nhatky.Don(logDangChay(db), 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(giu); err != nil {
		t.Fatalf("dọn theo ngân sách đã xoá mất nhật ký của phiên đang chạy: %v (kq=%+v)", err, kq)
	}
}

// logDangChay lặp lại đúng phép chọn mà fleet dùng, để bài test trên đo đúng
// thứ sản phẩm làm chứ không phải một danh sách tự dựng.
func logDangChay(db *store.DB) []string {
	list, err := db.Running()
	if err != nil {
		return nil
	}
	var out []string
	for _, s := range list {
		if s.Log != "" {
			out = append(out, s.Log)
		}
	}
	return out
}

// Fleet phải NÓI RA chỗ đọc nhật ký. Có nhật ký mà không ai tìm ra thì bằng
// không có — đó là nửa còn lại của vấn đề ngày 21/08.
func TestFanOutChiDuongToiNhatKy(t *testing.T) {
	db, bus, a := setup(t)
	ch, huy := bus.Subscribe(256)
	defer huy()

	if _, err := FanOut(db, bus, a, "phu", Opts{Copies: 1}, []string{"-test.run=TestHelperProcess"}); err != nil {
		t.Fatal(err)
	}
	var coChiDuong bool
	for _, e := range gomEvent(ch) {
		if strings.Contains(e.Msg, "sagent nhat-ky") && strings.Contains(e.Msg, nhatky.Root()) {
			coChiDuong = true
		}
	}
	if !coChiDuong {
		t.Error("fleet không nói ra thư mục nhật ký lẫn lệnh đọc lại nó")
	}
}
