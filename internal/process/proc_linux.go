//go:build linux

// Package process là lớp mỏng hỏi/giết tiến trình, tách theo nền tảng.
package process

import "syscall"

// IsAlive: signal 0 không gửi gì, chỉ kiểm tra tiến trình có tồn tại.
func IsAlive(pid int) bool { return syscall.Kill(pid, 0) == nil }

// Kill gửi SIGTERM cho cả nhóm tiến trình (âm PID) để dọn luôn con.
func Kill(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGTERM); err == nil {
		return nil
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}
