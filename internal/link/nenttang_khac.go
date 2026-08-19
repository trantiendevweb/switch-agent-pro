//go:build !windows

package link

// Xem internal/process/khong_windows.go — sagent chỉ hỗ trợ Windows.
func IsLink(string) (bool, error)   { return sagentChiHoTroWindows_xemREADME(), nil }
func LinkDir(string, string) error  { sagentChiHoTroWindows_xemREADME(); return nil }
func LinkFile(string, string) error { sagentChiHoTroWindows_xemREADME(); return nil }
func Unlink(string, bool) error     { sagentChiHoTroWindows_xemREADME(); return nil }
