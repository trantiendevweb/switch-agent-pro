//go:build windows

// Package process là lớp mỏng hỏi/giết tiến trình, tách theo nền tảng.
package process

import (
	"os/exec"
	"strconv"
	"syscall"
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
