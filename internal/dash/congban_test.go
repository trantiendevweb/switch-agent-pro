package dash

import (
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Cong ban thi phai noi RO ai dang giu. Cau bao loi cu chi tra lai nguyen van
// cua Windows — "Only one usage of each socket address..." — mot cau khong tra
// loi cau hoi duy nhat nguoi doc dang co: AI dang giu cong cua toi?
//
// Ngay 20/08 cau hoi do lam mat gan tron mot ngay: mot ban build cu bo quen
// trong thu muc repo giu cong 8788 tu 15:27, moi lan "bat lai dash" deu chet
// ngay, va vi chay nen an nen khong ai doc duoc cau bao loi.
func TestCongBanThiNoiRoAiDangGiu(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	cong := ln.Addr().(*net.TCPAddr).Port

	ai := aiGiuCong(cong)
	if ai == "" {
		t.Skip("khong tra duoc tien trinh giu cong o may nay (khong co PowerShell?) — " +
			"cau bao loi se rut gon lai, khong sai, chi la it thong tin hon")
	}

	// Chinh tien trinh test dang giu cong, nen PID phai la cua no.
	pid := strconv.Itoa(os.Getpid())
	if !strings.Contains(ai, pid) {
		t.Errorf("aiGiuCong(%d) = %q, phai chua PID cua chinh tien trinh nay (%s)", cong, ai, pid)
	}

	msg := loiCongBan(cong, errors.New("bind: address already in use")).Error()
	for _, c := range []string{strconv.Itoa(cong), pid, "Stop-Process", "--port"} {
		if !strings.Contains(msg, c) {
			t.Errorf("cau bao loi thieu %q:\n%s", c, msg)
		}
	}
	// Loi goc phai con nguyen van trong cau: bo no di la nguoi doc mat manh
	// thong tin duy nhat co the tra cuu duoc.
	if !strings.Contains(msg, "address already in use") {
		t.Errorf("cau bao loi nuot mat loi goc:\n%s", msg)
	}
}

// Tra khong ra thi cau bao loi phai NGAN DI, khong duoc bien thanh mot loi thu
// hai che mat loi that.
func TestTraKhongRaThiVanConCauBaoLoiCu(t *testing.T) {
	// Cong 0 khong bao gio co ai nghe, nen aiGiuCong chac chan tra rong.
	msg := loiCongBan(0, errors.New("loi goc gi do")).Error()
	if !strings.Contains(msg, "loi goc gi do") {
		t.Errorf("mat loi goc:\n%s", msg)
	}
	if !strings.Contains(msg, "--port") {
		t.Errorf("mat goi y doi cong:\n%s", msg)
	}
}
