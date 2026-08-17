//go:build windows

package acl

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Những SID "rộng": ACE cấp cho một trong số này nghĩa là file không còn riêng.
var sidRong = map[string]string{
	"S-1-1-0":      "Everyone",
	"S-1-5-32-545": "Users",
	"S-1-5-11":     "Authenticated Users",
	"S-1-5-32-546": "Guests",
	"S-1-5-4":      "Interactive",
}

// Restrict dựng DACL tường minh: chủ sở hữu + SYSTEM + nhóm quản trị, và CẮT
// KẾ THỪA.
//
// Cắt kế thừa là phần không được quên: chỉ thêm ACE cho chủ sở hữu mà vẫn để
// ACE rộng của thư mục cha chảy xuống thì siết được đúng con số không. Cờ
// PROTECTED_DACL_SECURITY_INFORMATION là cái làm việc đó (tương đương
// `icacls /inheritance:r`).
//
// Giữ SYSTEM và Administrators vì bỏ chúng đi thì sao lưu hệ thống, quét virus
// và chính người quản trị máy sẽ không đọc được — đổi một rủi ro lấy một rủi ro
// khác thì không phải là siết.
func Restrict(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}

	me, err := chuSoHuu()
	if err != nil {
		return err
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return err
	}
	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return err
	}

	// Thư mục phải cho kế thừa xuống con, nếu không thì file tạo sau lại lỏng.
	keThua := uint32(windows.NO_INHERITANCE)
	if fi.IsDir() {
		keThua = windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT
	}

	ea := make([]windows.EXPLICIT_ACCESS, 0, 3)
	for _, sid := range []*windows.SID{me, system, admins} {
		ea = append(ea, windows.EXPLICIT_ACCESS{
			AccessPermissions: windows.GENERIC_ALL,
			AccessMode:        windows.GRANT_ACCESS,
			Inheritance:       keThua,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeType:  windows.TRUSTEE_IS_USER,
				TrusteeValue: windows.TrusteeValueFromSID(sid),
			},
		})
	}
	dacl, err := windows.ACLFromEntries(ea, nil)
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil)
}

// Check duyệt từng ACE và trả lời một câu duy nhất: có ai ngoài chủ sở hữu,
// SYSTEM và nhóm quản trị đang được cấp quyền không.
func Check(path string) (bool, string, error) {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return false, "", err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return false, "", err
	}
	if dacl == nil {
		// DACL rỗng (NULL) nghĩa là AI CŨNG VÀO ĐƯỢC — không phải "không có ai".
		return false, "không có DACL — mọi người đều truy cập được", nil
	}
	var thay []string
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return false, "", err
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		s := sid.String()
		if ten, rong := sidRong[s]; rong {
			thay = append(thay, ten)
		}
	}
	if len(thay) > 0 {
		return false, fmt.Sprintf("cấp quyền cho %v — chạy `sagent verify` sau khi vá, hoặc siết lại bằng sagent", thay), nil
	}
	return true, "chỉ chủ sở hữu, SYSTEM và nhóm quản trị", nil
}

func chuSoHuu() (*windows.SID, error) {
	tok := windows.GetCurrentProcessToken()
	u, err := tok.GetTokenUser()
	if err != nil {
		return nil, err
	}
	return u.User.Sid, nil
}
