//go:build windows

package process

import (
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"
)

// IsAlive mở tiến trình bằng quyền tối thiểu; mở được và chưa thoát = còn sống.
func IsAlive(pid int) bool {
	const queryLimited = 0x1000
	h, err := syscall.OpenProcess(queryLimited, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return true // mở được nhưng không đọc được mã thoát -> coi như sống
	}
	const stillActive = 259
	return code == stillActive
}

// Kill giết tiến trình cùng cả cây con (/T) — agent hay sinh tiến trình con.
func Kill(pid int) error {
	return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
}

// parentMap đọc bảng "pid -> ppid" của cả máy bằng ảnh chụp Toolhelp32.
//
// Dùng syscall thẳng chứ không gọi `tasklist`/`wmic`: `wmic` đã bị gỡ khỏi
// Windows Server 2022 (đo được lúc viết hàm này — nó trả về rỗng, im lặng), và
// bất cứ thứ gì phải parse chữ đều hỏng theo ngôn ngữ hệ thống.
//
// Trên Windows, ParentProcessID VẪN đọc được sau khi cha đã thoát — nhờ vậy dọn
// được tiến trình mồ côi. Đây là khác biệt quan trọng so với Linux.
func parentMap() map[int]int {
	h, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(h)

	out := map[int]int{}
	var e syscall.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := syscall.Process32First(h, &e); err != nil {
		return out
	}
	for {
		out[int(e.ProcessID)] = int(e.ParentProcessID)
		if err := syscall.Process32Next(h, &e); err != nil {
			return out
		}
	}
}
