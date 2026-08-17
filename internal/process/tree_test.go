package process

import (
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// Lệnh chạy đủ lâu để đo mà không treo test nếu có sót.
func lenhCho() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "ping -n 20 127.0.0.1 >nul"}
	}
	return []string{"sh", "-c", "sleep 20"}
}

func lenhChoLongNhau() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "cmd /c ping -n 20 127.0.0.1 >nul"}
	}
	return []string{"sh", "-c", "sh -c 'sleep 20'"}
}

func doiTat(t *testing.T, pids []int, trong time.Duration) []int {
	t.Helper()
	han := time.Now().Add(trong)
	for {
		var con []int
		for _, p := range pids {
			if IsAlive(p) {
				con = append(con, p)
			}
		}
		if len(con) == 0 || time.Now().After(han) {
			return con
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Descendants phải thấy được cháu, không chỉ con.
func TestDescendantsThayCaChau(t *testing.T) {
	a := lenhChoLongNhau()
	cmd := exec.Command(a[0], a[1:]...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { _ = KillTree(pid) })
	time.Sleep(600 * time.Millisecond)

	d := Descendants(pid)
	if len(d) == 0 {
		t.Fatalf("Descendants(%d) rỗng — không thấy tiến trình con nào, lá chắn dọn cây sẽ vô dụng", pid)
	}
	for _, c := range d {
		if c == pid {
			t.Fatalf("Descendants trả về cả chính nó (%d)", pid)
		}
	}
	t.Logf("thấy %d hậu duệ: %v", len(d), d)
}

// Cây bình thường: cha còn sống. Đây là đường mà Kill cũ đã làm đúng — giữ để
// bản vá không phá cái đang chạy được.
func TestKillTreeDonSachCayConSong(t *testing.T) {
	a := lenhChoLongNhau()
	cmd := exec.Command(a[0], a[1:]...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	time.Sleep(600 * time.Millisecond)
	hauDue := Descendants(pid)

	if err := KillTree(pid); err != nil {
		t.Fatalf("KillTree = %v", err)
	}
	if IsAlive(pid) {
		t.Fatal("tiến trình cha vẫn sống sau KillTree")
	}
	if con := doiTat(t, hauDue, 3*time.Second); len(con) > 0 {
		t.Fatalf("còn %d hậu duệ sống: %v", len(con), con)
	}
	_ = cmd.Wait()
}

// MỒ CÔI — chỗ Kill cũ nổ. Đo được trước khi vá:
//
//	cha đã thoát; PING mồ côi vẫn chạy
//	Kill(pid cha đã chết) = exit status 128
//	PING vẫn chạy sau đó
//
// Tiến trình sót lại vẫn tiêu hạn mức, và không ai biết vì Kill chỉ trả một
// chuỗi vô nghĩa. KillTree phải dọn được, vì trên Windows quan hệ cha-con còn
// đọc được sau khi cha thoát.
func TestKillTreeDonDuocMoCoi(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("cách tạo mồ côi khác nhau theo nền tảng; Linux chưa có máy để đo")
	}
	cmd := exec.Command("cmd", "/c", "start /b ping -n 20 127.0.0.1 >nul")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Wait() // cha thoát ngay, con ở lại
	time.Sleep(600 * time.Millisecond)

	if IsAlive(pid) {
		t.Skip("tiến trình cha chưa thoát — không dựng được cảnh mồ côi")
	}
	moCoi := Descendants(pid)
	if len(moCoi) == 0 {
		t.Skip("không dựng được tiến trình mồ côi ở môi trường này")
	}
	t.Logf("mồ côi: %v", moCoi)

	if err := KillTree(pid); err != nil {
		t.Fatalf("KillTree(pid đã chết) = %v — phải dọn được mồ côi chứ không báo lỗi", err)
	}
	if con := doiTat(t, moCoi, 3*time.Second); len(con) > 0 {
		t.Fatalf("MỒ CÔI SỐNG SÓT: %v — chúng vẫn tiêu hạn mức", con)
	}
}

// Một hàm dừng tiến trình mà trả nil trong khi tiến trình vẫn chạy thì tệ hơn
// là không có: người dùng tin là đã dừng rồi đi làm việc khác.
func TestKillTreeKhongNoiDoiKhiConSongSot(t *testing.T) {
	if IsAlive(1) && runtime.GOOS == "windows" {
		// PID 1 không phải tiến trình thật trên Windows; bỏ qua nhánh này.
		t.Skip("không có PID cố định để đo trên nền tảng này")
	}
	// PID chắc chắn không tồn tại: dừng cái không có thì không được coi là lỗi
	// (nó đã chết rồi — đúng ý người dùng).
	if err := KillTree(0x7FFFFFF0); err != nil {
		t.Fatalf("dừng một PID đã chết không được coi là lỗi, được %v", err)
	}
}
