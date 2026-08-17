//go:build !windows

package console

// sagent chỉ hỗ trợ Windows — xem internal/process/nenttang_khac.go.
func Dat() func() { return func() {} }
func KhoiPhuc()   {}
