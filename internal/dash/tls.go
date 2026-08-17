package dash

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/acl"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// TLS cho dashboard.
//
// Vì sao cần: `--host 0.0.0.0` gửi mật khẩu dạng TRẦN trên đường truyền. Cả một
// chuỗi việc đã làm để bảo vệ nó — băm PBKDF2 210k vòng, siết ACL file, bỏ token
// khỏi URL — đều vô nghĩa nếu ai đó ngồi cùng mạng Wi-Fi đọc được nó lúc đăng nhập.
//
// Vì sao TỰ KÝ chứ không Let's Encrypt: dashboard chạy trên máy cá nhân sau NAT,
// không có tên miền, thường chỉ nghe IP nội bộ. ACME không cấp cho địa chỉ kiểu
// đó. Tự ký nghĩa là trình duyệt sẽ cảnh báo — nên công cụ phải IN VÂN TAY ra
// terminal để người dùng đối chiếu. Bấm "vẫn tiếp tục" mà không đối chiếu thì
// TLS chỉ chống nghe lén thụ động, không chống được kẻ đứng giữa.

const tlsHanDung = 825 * 24 * time.Hour // 825 ngày: giới hạn các trình duyệt chấp nhận

func tlsDir() string  { return filepath.Join(paths.AccountsRoot(), "dash-tls") }
func certPath() string { return filepath.Join(tlsDir(), "cert.pem") }
func keyPath() string  { return filepath.Join(tlsDir(), "key.pem") }

// EnsureCert trả về đường dẫn cert/key dùng được, sinh mới nếu cần.
//
// Sinh lại khi: chưa có · sắp hết hạn · KHÔNG phủ hết IP hiện tại của máy. Vế
// cuối là vế hay bị quên: đổi Wi-Fi là máy có IP khác, cert cũ vẫn còn hạn nhưng
// trình duyệt báo sai tên miền, và người dùng sẽ tưởng mình bị tấn công.
func EnsureCert() (certFile, keyFile, vanTay string, err error) {
	certFile, keyFile = certPath(), keyPath()
	if c, err := docCert(certFile); err == nil && conDung(c) {
		return certFile, keyFile, VanTay(c), nil
	}
	c, err := sinhCert()
	if err != nil {
		return "", "", "", err
	}
	return certFile, keyFile, VanTay(c), nil
}

// VanTay là vân tay SHA-256 của chứng chỉ, định dạng đúng như trình duyệt hiện
// trong hộp thoại "xem chứng chỉ" — để đối chiếu được bằng mắt.
func VanTay(c *x509.Certificate) string {
	sum := sha256.Sum256(c.Raw)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ":")
}

func docCert(path string) (*x509.Certificate, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, fmt.Errorf("%s không phải PEM", path)
	}
	return x509.ParseCertificate(blk.Bytes)
}

func conDung(c *x509.Certificate) bool {
	if time.Now().After(c.NotAfter.Add(-30 * 24 * time.Hour)) {
		return false // còn dưới 30 ngày thì làm mới luôn, đừng đợi hỏng giữa chừng
	}
	co := map[string]bool{}
	for _, ip := range c.IPAddresses {
		co[ip.String()] = true
	}
	for _, ip := range ipCuaMay() {
		if !co[ip.String()] {
			return false
		}
	}
	return true
}

// ipCuaMay liệt kê mọi địa chỉ IP đang gán cho máy, kể cả loopback.
func ipCuaMay() []net.IP {
	out := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if !ok || n.IP.IsLoopback() || n.IP.IsLinkLocalUnicast() {
			continue
		}
		out = append(out, n.IP)
	}
	return out
}

func sinhCert() (*x509.Certificate, error) {
	if err := os.MkdirAll(tlsDir(), 0o700); err != nil {
		return nil, err
	}
	// Khoá riêng nằm ở đây. 0o700 ở trên không bảo vệ gì trên Windows — xem
	// internal/acl. Siết trước khi ghi khoá vào, không phải sau.
	_ = acl.Restrict(tlsDir())

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	ten, _ := os.Hostname()
	if ten == "" {
		ten = "sagent"
	}
	mau := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "sagent dashboard (" + ten + ")"},
		NotBefore:             time.Now().Add(-1 * time.Hour), // lệch đồng hồ giữa máy
		NotAfter:              time.Now().Add(tlsHanDung),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost", ten},
		IPAddresses:           ipCuaMay(),
	}
	der, err := x509.CreateCertificate(rand.Reader, &mau, &mau, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	if err := ghiPEM(certPath(), "CERTIFICATE", der, 0o644); err != nil {
		return nil, err
	}
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	if err := ghiPEM(keyPath(), "EC PRIVATE KEY", kb, 0o600); err != nil {
		return nil, err
	}
	_ = acl.Restrict(keyPath())

	return x509.ParseCertificate(der)
}

func ghiPEM(path, loai string, der []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: loai, Bytes: der})
}
