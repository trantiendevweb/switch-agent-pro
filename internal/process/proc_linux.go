//go:build linux

package process

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// IsAlive: signal 0 không gửi gì, chỉ kiểm tra tiến trình có tồn tại.
func IsAlive(pid int) bool { return syscall.Kill(pid, 0) == nil }

// Kill gửi SIGTERM cho cả nhóm tiến trình (âm PID) để dọn luôn con.
func Kill(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGTERM); err == nil {
		return nil
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}

// parentMap đọc bảng "pid -> ppid" từ /proc.
//
// CHƯA ĐO trên máy Linux thật — dự án chưa có chỗ chạy (xem docs/DO-LUONG.md).
// Đọc /proc/<pid>/stat thay vì /status vì trường thứ 4 của stat là PPid ở vị trí
// cố định; tên tiến trình nằm trong ngoặc đơn và CÓ THỂ chứa khoảng trắng, nên
// phải cắt từ dấu ')' cuối cùng chứ không tách theo khoảng trắng từ đầu.
//
// Khác Windows ở một điểm quyết định: khi cha chết, con được init/systemd nhận
// nuôi và PPid đổi thành 1 — quan hệ cũ MẤT HẲN. Nên trên Linux, muốn dọn được
// cây thì phải chụp danh sách trước khi giết (KillTree đã làm vậy), và phần mồ
// côi-từ-trước thì không truy ra được bằng đường này.
func parentMap() map[int]int {
	ents, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	out := map[int]int{}
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // /proc còn nhiều thứ không phải tiến trình
		}
		b, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue // tiến trình vừa thoát giữa lúc đọc — chuyện thường
		}
		s := string(b)
		i := strings.LastIndexByte(s, ')')
		if i < 0 || i+2 >= len(s) {
			continue
		}
		f := strings.Fields(s[i+2:]) // [0]=state, [1]=ppid
		if len(f) < 2 {
			continue
		}
		if ppid, err := strconv.Atoi(f[1]); err == nil {
			out[pid] = ppid
		}
	}
	return out
}
