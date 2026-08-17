//go:build windows

package process

import (
	"os/exec"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
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

// procEnt là một dòng trong bảng tiến trình của máy.
type procEnt struct {
	ppid int
	ten  string // tên file thực thi, để người dùng nhìn ra đó là cái gì
}

// procTable đọc bảng tiến trình của cả máy bằng ảnh chụp Toolhelp32.
//
// Dùng syscall thẳng chứ không gọi `tasklist`/`wmic`: `wmic` đã bị gỡ khỏi
// Windows Server 2022 (đo được lúc viết hàm này — nó trả về rỗng, im lặng), và
// bất cứ thứ gì phải parse chữ đều hỏng theo ngôn ngữ hệ thống.
//
// Trên Windows, ParentProcessID VẪN đọc được sau khi cha đã thoát — nhờ vậy dọn
// được tiến trình mồ côi. Đây là khác biệt quan trọng so với Linux.
func procTable() map[int]procEnt {
	h, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(h)

	out := map[int]procEnt{}
	var e syscall.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := syscall.Process32First(h, &e); err != nil {
		return out
	}
	for {
		out[int(e.ProcessID)] = procEnt{
			ppid: int(e.ParentProcessID),
			ten:  syscall.UTF16ToString(e.ExeFile[:]),
		}
		if err := syscall.Process32Next(h, &e); err != nil {
			return out
		}
	}
}

func parentMap() map[int]int {
	t := procTable()
	out := make(map[int]int, len(t))
	for pid, e := range t {
		out[pid] = e.ppid
	}
	return out
}

// StartTime là thời điểm tiến trình được tạo.
//
// Đây là thứ DUY NHẤT phân biệt được "con của phiên đã chết" với "con của một
// tiến trình mới tình cờ trùng PID". Không có nó thì dọn mồ côi là đánh bạc.
func StartTime(pid int) (time.Time, bool) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return time.Time{}, false
	}
	defer windows.CloseHandle(h)
	var tao, thoat, nhan, nguoi windows.Filetime
	if err := windows.GetProcessTimes(h, &tao, &thoat, &nhan, &nguoi); err != nil {
		return time.Time{}, false
	}
	return time.Unix(0, tao.Nanoseconds()), true
}
