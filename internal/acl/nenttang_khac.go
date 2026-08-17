//go:build !windows

package acl

// Xem internal/process/khong_windows.go — sagent chỉ hỗ trợ Windows.
func Restrict(string) error                { sagentChiHoTroWindows_xemREADME(); return nil }
func Check(string) (bool, string, error)   { return sagentChiHoTroWindows_xemREADME(), "", nil }
