//go:build windows

package provider

import (
	"syscall"
	"unsafe"
)

// Đọc Windows Credential Manager — chỉ để hỏi "có mục này không".
//
// x/sys/windows không bọc CredReadW nên phải tự khai. Cố ý CHỈ kiểm sự tồn tại
// và giải phóng ngay: không đọc, không sao chép, không log phần bí mật. Cái duy
// nhất cần biết là "đã đăng nhập chưa".
var (
	advapi32     = syscall.NewLazyDLL("advapi32.dll")
	procCredRead = advapi32.NewProc("CredReadW")
	procCredFree = advapi32.NewProc("CredFree")
)

const credTypeGeneric = 1

// coCredential trả true nếu Credential Manager có mục tên `target`.
func coCredential(target string) bool {
	p, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return false
	}
	var pcred uintptr
	r, _, _ := procCredRead.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(credTypeGeneric),
		0,
		uintptr(unsafe.Pointer(&pcred)),
	)
	if r == 0 {
		return false
	}
	procCredFree.Call(pcred)
	return true
}
