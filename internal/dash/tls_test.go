package dash

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// Chứng chỉ phải phủ MỌI địa chỉ máy đang có, không chỉ localhost. Vế này hay
// bị quên: mở dashboard từ điện thoại là gõ IP LAN, cert không phủ IP đó thì
// trình duyệt báo sai tên miền và người dùng tưởng mình bị tấn công.
func TestChungChiPhuHetDiaChiCuaMay(t *testing.T) {
	newTestServer(t) // dựng HOME giả
	certFile, keyFile, vanTay, err := EnsureCert()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		t.Fatalf("không có khoá riêng: %v", err)
	}
	c, err := docCert(certFile)
	if err != nil {
		t.Fatal(err)
	}

	co := map[string]bool{}
	for _, ip := range c.IPAddresses {
		co[ip.String()] = true
	}
	for _, ip := range ipCuaMay() {
		if !co[ip.String()] {
			t.Errorf("chứng chỉ thiếu IP %s — mở từ máy khác sẽ báo sai tên miền", ip)
		}
	}
	var coLocalhost bool
	for _, d := range c.DNSNames {
		if d == "localhost" {
			coLocalhost = true
		}
	}
	if !coLocalhost {
		t.Error("chứng chỉ thiếu DNS name localhost")
	}

	// Vân tay phải đúng định dạng trình duyệt hiện, nếu không thì không đối
	// chiếu bằng mắt được — mà đối chiếu là toàn bộ giá trị của việc in nó ra.
	if n := len(strings.Split(vanTay, ":")); n != 32 {
		t.Errorf("vân tay có %d nhóm, SHA-256 phải là 32: %s", n, vanTay)
	}
	if vanTay != VanTay(c) {
		t.Error("vân tay trả về không khớp chứng chỉ trên đĩa")
	}
	// 825 ngày là trần các trình duyệt chấp nhận cho cert tự ký.
	if d := time.Until(c.NotAfter); d > 826*24*time.Hour {
		t.Errorf("hạn dùng %v quá dài, trình duyệt sẽ từ chối", d)
	}
}

// Gọi lần hai phải DÙNG LẠI, không sinh cert mới: sinh mới mỗi lần chạy thì vân
// tay đổi liên tục, người dùng quen với việc bỏ qua cảnh báo, và cả cơ chế đối
// chiếu trở thành vô dụng.
func TestGoiLanHaiDungLaiChungChi(t *testing.T) {
	newTestServer(t)
	_, _, vt1, err := EnsureCert()
	if err != nil {
		t.Fatal(err)
	}
	_, _, vt2, err := EnsureCert()
	if err != nil {
		t.Fatal(err)
	}
	if vt1 != vt2 {
		t.Fatalf("sinh lại chứng chỉ dù cái cũ còn dùng được:\n  %s\n  %s", vt1, vt2)
	}
}

// Cert còn hạn nhưng KHÔNG phủ IP hiện tại (đổi Wi-Fi chẳng hạn) thì phải sinh lại.
func TestThieuIPThiSinhLai(t *testing.T) {
	newTestServer(t)
	_, _, vt1, err := EnsureCert()
	if err != nil {
		t.Fatal(err)
	}
	c, err := docCert(certPath())
	if err != nil {
		t.Fatal(err)
	}
	// Giả lập máy vừa có thêm địa chỉ mới: bỏ bớt một IP khỏi cert đang có bằng
	// cách kiểm trực tiếp hàm quyết định.
	giaThieu := &x509.Certificate{
		NotAfter:    c.NotAfter,
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, // thiếu ::1 và IP LAN
	}
	if conDung(giaThieu) && len(ipCuaMay()) > 1 {
		t.Fatal("conDung chấp nhận chứng chỉ thiếu IP — sẽ không bao giờ sinh lại")
	}
	if !conDung(c) {
		t.Fatalf("chứng chỉ vừa sinh mà bị coi là hỏng (vân tay %s)", vt1)
	}
}

// Phơi ra mạng mà không mã hoá thì mọi thứ làm để bảo vệ mật khẩu đều vô nghĩa.
// Chốt này nằm trong Run chứ không chỉ ở CLI, vì Run là chỗ không đi vòng được.
func TestPhoiRaMangKhongTLSThiTuChoiChay(t *testing.T) {
	s := newTestServer(t)

	// Chạy trong goroutine kèm hạn giờ. Gọi thẳng `s.Run(...)` thì khi ai đó gỡ
	// mất lá chắn, test sẽ TREO chứ không đỏ — vì lúc đó Run phục vụ thật và
	// không bao giờ trả về. Một test hỏng-kiểu-treo còn tệ hơn không có test:
	// nó chỉ hiện ra sau 10 phút, dưới dạng panic khó đọc. (Đã dẫm đúng lỗi này
	// khi thử gỡ lá chắn để kiểm chứng test.)
	loi := make(chan error, 1)
	go func() { loi <- s.Run("0.0.0.0", 0) }()
	var err error
	select {
	case err = <-loi:
	case <-time.After(3 * time.Second):
		t.Fatal("Run KHÔNG từ chối mà đã mở cổng phơi ra mạng bằng HTTP trần — " +
			"mật khẩu sẽ đi qua dây dưới dạng chữ thường")
	}
	if err == nil {
		t.Fatal("Run trả về nil khi phơi ra mạng bằng HTTP trần")
	}
	if !strings.Contains(err.Error(), "TLS") {
		t.Errorf("thông điệp lỗi không nói tới TLS: %v", err)
	}
	// Nhưng người CHỦ ĐỘNG chấp nhận (đã có SSH tunnel/VPN) thì không bị chặn.
	s2 := newTestServer(t)
	s2.ChiuHTTPTran()
	done := make(chan error, 1)
	go func() { done <- s2.Run("127.0.0.1", 0) }()
	select {
	case err := <-done:
		t.Fatalf("bị chặn dù đã --http-tran: %v", err)
	case <-time.After(300 * time.Millisecond):
		// vẫn đang phục vụ = đúng
	}
}

// Cookie chỉ được gắn Secure khi THẬT SỰ chạy TLS. Gắn vô điều kiện thì trên
// http://127.0.0.1 trình duyệt vứt cookie đi và không ai đăng nhập được.
func TestCookieSecureBamTheoTLS(t *testing.T) {
	for _, tls := range []bool{false, true} {
		s := newTestServer(t)
		s.DungTLS(tls)
		form := url.Values{"user": {"Admin"}, "password": {matKhauTest}}
		r := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
		r.Host = "127.0.0.1:4600"
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", "http://127.0.0.1:4600")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		var found *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == cookieName && c.Value != "" {
				found = c
			}
		}
		if found == nil {
			t.Fatalf("TLS=%v: đăng nhập không cấp cookie", tls)
		}
		if found.Secure != tls {
			t.Errorf("TLS=%v nhưng cookie.Secure=%v", tls, found.Secure)
		}
	}
}

// Bắt tay TLS THẬT trên cổng thật, và client ghim đúng chứng chỉ đã in vân tay.
// Đây là phép đo duy nhất chứng minh đường truyền được mã hoá — mọi test trên
// chỉ đo cấu hình.
func TestBatTayTLSThat(t *testing.T) {
	s := newTestServer(t)
	s.DungTLS(true)

	certFile, keyFile, vanTay, err := EnsureCert()
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	srv := &http.Server{Handler: s}
	go func() { _ = srv.ServeTLS(ln, certFile, keyFile) }()
	defer srv.Close()

	c, err := docCert(certFile)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(c)
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{
			RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS12,
		}},
	}
	// Cổng thật nghe ở 127.0.0.1 nhưng ta nói ServerName=localhost để ép xác
	// thực theo TÊN, tức là kiểm luôn cả SAN chứ không chỉ kiểm bắt tay.
	addr := ln.Addr().(*net.TCPAddr)
	resp, err := client.Get("https://localhost:" + itoa(addr.Port) + "/login")
	if err != nil {
		t.Fatalf("bắt tay TLS hỏng: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Đăng nhập") {
		t.Fatalf("qua TLS không lấy được form đăng nhập: mã %d", resp.StatusCode)
	}
	if resp.TLS == nil {
		t.Fatal("kết nối không phải TLS")
	}
	if resp.TLS.Version < tls.VersionTLS12 {
		t.Errorf("bắt tay bằng TLS cũ: %x", resp.TLS.Version)
	}
	// Chứng chỉ server trả về phải ĐÚNG cái đã in vân tay ra terminal, nếu không
	// thì việc đối chiếu bằng mắt chẳng chứng minh được gì.
	if got := VanTay(resp.TLS.PeerCertificates[0]); got != vanTay {
		t.Fatalf("chứng chỉ trên dây khác cái đã in:\n  in ra: %s\n  trên dây: %s", vanTay, got)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
