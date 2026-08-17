package process

import (
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// Cha CÒN SỐNG thì đó là cây bình thường, không phải mồ côi — MoCoi phải im lặng,
// nếu không thì `sagent quet` sẽ rủ người dùng giết phiên đang chạy của chính họ.
func TestMoCoiImLangKhiChaConSong(t *testing.T) {
	a := lenhChoLongNhau()
	cmd := exec.Command(a[0], a[1:]...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { _ = KillTree(pid) })
	time.Sleep(600 * time.Millisecond)

	if got := MoCoi(pid, time.Now().Add(-time.Hour)); len(got) != 0 {
		t.Fatalf("cha còn sống mà MoCoi trả về %d tiến trình: %+v", len(got), got)
	}
}

// Đường chính: cha thoát, con ở lại, MoCoi phải thấy con kèm TÊN và THỜI ĐIỂM —
// một danh sách chỉ có số PID thì không ai duyệt được và người ta sẽ bấm đồng ý
// cho xong.
func TestMoCoiThayConCuaChaDaThoat(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("cách tạo mồ côi khác nhau theo nền tảng")
	}
	moc := time.Now().Add(-2 * time.Second)
	cmd := exec.Command("cmd", "/c", "start /b ping -n 20 127.0.0.1 >nul")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Wait()
	time.Sleep(600 * time.Millisecond)
	if IsAlive(pid) {
		t.Skip("cha chưa thoát — không dựng được cảnh mồ côi")
	}

	got := MoCoi(pid, moc)
	if len(got) == 0 {
		t.Fatal("không thấy tiến trình mồ côi nào — lệnh `quet` sẽ vô dụng")
	}
	t.Cleanup(func() {
		for _, p := range got {
			_ = Kill(p.PID)
		}
	})
	for _, p := range got {
		if p.Ten == "" {
			t.Errorf("PID %d không có tên tiến trình", p.PID)
		}
		if p.BatDau.IsZero() {
			t.Errorf("PID %d không có thời điểm bắt đầu", p.PID)
		}
		if p.BatDau.Before(moc) {
			t.Errorf("PID %d bắt đầu %v, trước mốc %v — lẽ ra phải bị loại", p.PID, p.BatDau, moc)
		}
	}
	t.Logf("mồ côi: %+v", got)
}

// Lá chắn chống PID dùng lại. Windows cấp lại PID cho tiến trình mới; nếu không
// lọc theo thời điểm bắt đầu thì `quet --giet` sẽ giết nhầm thứ chẳng liên quan.
//
// Đặt mốc ở TƯƠNG LAI thì mọi tiến trình đều "có trước phiên" và phải bị loại
// sạch. Không loại sạch nghĩa là bộ lọc không hoạt động.
func TestMoCoiLoaiTienTrinhCoTruocMoc(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("cách tạo mồ côi khác nhau theo nền tảng")
	}
	cmd := exec.Command("cmd", "/c", "start /b ping -n 20 127.0.0.1 >nul")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Wait()
	time.Sleep(600 * time.Millisecond)
	t.Cleanup(func() {
		for _, p := range MoCoi(pid, time.Now().Add(-time.Hour)) {
			_ = Kill(p.PID)
		}
	})
	if IsAlive(pid) {
		t.Skip("cha chưa thoát")
	}
	if len(MoCoi(pid, time.Now().Add(-time.Hour))) == 0 {
		t.Skip("không dựng được mồ côi ở môi trường này")
	}

	if got := MoCoi(pid, time.Now().Add(time.Hour)); len(got) != 0 {
		t.Fatalf("bộ lọc thời gian không hoạt động: vẫn nhận %d tiến trình bắt đầu TRƯỚC mốc: %+v",
			len(got), got)
	}
}
