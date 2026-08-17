package dash

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/trantiendevweb/switch-agent-pro/internal/acl"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// Auth là thông tin đăng nhập dashboard.
//
// Mật khẩu KHÔNG BAO GIỜ được lưu dạng thô, và file này KHÔNG nằm trong repo —
// nó ở ~/.ai-accounts/dash-auth.json. Nhét mật khẩu vào mã nguồn của một dự án
// mã nguồn mở nghĩa là ai clone về cũng đăng nhập được vào máy bạn.
type Auth struct {
	User string `json:"user"`
	Salt string `json:"salt"` // base64
	Hash string `json:"hash"` // base64, PBKDF2-HMAC-SHA256
	Iter int    `json:"iter"`
}

// AuthPath là nơi giữ thông tin đăng nhập (ngoài repo, trong thư mục người dùng).
func AuthPath() string { return filepath.Join(paths.AccountsRoot(), "dash-auth.json") }

// pbkdf2 (RFC 8018) bằng thư viện chuẩn — không kéo thêm phụ thuộc chỉ để băm
// một mật khẩu. Chậm có chủ đích: kẻ lấy được file cũng phải trả giá khi dò.
func pbkdf2(password, salt []byte, iter, keyLen int) []byte {
	const hashLen = sha256.Size
	blocks := (keyLen + hashLen - 1) / hashLen
	dk := make([]byte, 0, blocks*hashLen)
	u := make([]byte, hashLen)
	var idx [4]byte
	for b := 1; b <= blocks; b++ {
		prf := hmac.New(sha256.New, password)
		prf.Write(salt)
		idx[0], idx[1], idx[2], idx[3] = byte(b>>24), byte(b>>16), byte(b>>8), byte(b)
		prf.Write(idx[:])
		dk = prf.Sum(dk)
		t := dk[len(dk)-hashLen:]
		copy(u, t)
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for i := range u {
				t[i] ^= u[i]
			}
		}
	}
	return dk[:keyLen]
}

const authIter = 210_000 // đủ chậm để dò tốn kém, đủ nhanh để đăng nhập không lag

// SetPassword ghi thông tin đăng nhập mới (băm) ra đĩa.
func SetPassword(user, password string) error {
	if user == "" {
		return errors.New("thiếu tên đăng nhập")
	}
	if len(password) < 6 {
		return fmt.Errorf("mật khẩu quá ngắn (%d ký tự) — cần ít nhất 6", len(password))
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	a := Auth{
		User: user,
		Salt: base64.StdEncoding.EncodeToString(salt),
		Hash: base64.StdEncoding.EncodeToString(pbkdf2([]byte(password), salt, authIter, 32)),
		Iter: authIter,
	}
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(AuthPath()), 0o755); err != nil {
		return err
	}
	// File này là mật khẩu dashboard đã băm. 0o600 bên dưới không bảo vệ nó trên
	// Windows (đã đo, xem internal/acl) — siết thư mục để mọi thứ bên trong kín.
	_ = acl.Restrict(filepath.Dir(AuthPath()))
	// 0600: chỉ chủ máy đọc được.
	tmp := AuthPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, AuthPath())
}

// LoadAuth đọc thông tin đăng nhập. Trả về nil nếu chưa đặt bao giờ.
func LoadAuth() *Auth {
	data, err := os.ReadFile(AuthPath())
	if err != nil {
		return nil
	}
	var a Auth
	if json.Unmarshal(data, &a) != nil || a.Hash == "" || a.Iter <= 0 {
		return nil
	}
	return &a
}

// Check so khớp tên + mật khẩu, so sánh theo thời gian hằng để không rò rỉ qua
// thời gian phản hồi.
func (a *Auth) Check(user, password string) bool {
	if a == nil {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(a.Salt)
	if err != nil {
		return false
	}
	want, err := base64.StdEncoding.DecodeString(a.Hash)
	if err != nil {
		return false
	}
	got := pbkdf2([]byte(password), salt, a.Iter, len(want))
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(a.User))
	passOK := subtle.ConstantTimeCompare(got, want)
	return userOK == 1 && passOK == 1
}
