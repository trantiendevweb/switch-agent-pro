// Cổng đã có người giữ — nói RÕ là ai.
//
// CHUYỆN ĐÃ XẢY RA (20/08), và nó ăn mất gần trọn một ngày: một bản build cũ
// (`sagent-dem.exe`, bỏ quên trong thư mục repo) đang chạy `dash` và giữ cổng
// 8788 từ 15:27. Suốt buổi chiều tôi sửa mặt văn phòng bốn lần, mỗi lần đều
// build lại, "bật lại dash", rồi `curl` thấy HTTP 303 và kết luận là xong.
//
// Nhưng 303 đó là của SERVER CŨ trả lời. Bản mới bật lên, không mở được cổng,
// thoát ngay — và vì nó chạy nền ẩn nên không ai đọc câu báo lỗi. Người dùng
// gửi ảnh chụp về: giao diện y nguyên bản sáng. Mã đúng, binary đúng, chỉ là
// KHÔNG AI PHỤC VỤ nó.
//
// Bài học nằm ở chỗ kiểm: "có một server trả lời" không chứng minh được
// "server CỦA TÔI đang chạy". Hai câu đó khác nhau, và cả ngày hôm đó tôi kiểm
// câu đầu rồi tin câu sau.
//
// Hàm dưới đây không sửa được lỗi của tôi, nhưng nó cắt đúng chỗ tối: lần sau
// ai gặp cảnh này sẽ đọc được ngay TÊN và PID của thứ đang giữ cổng, thay vì
// câu "Only one usage of each socket address" của Windows.
package dash

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// aiGiuCong tra mô tả tiến trình đang nghe ở cổng này, rỗng nếu không tra được.
//
// Chỉ chạy trên Windows và cố ý KHÔNG báo lỗi ra ngoài: đây là phần phụ trợ cho
// một câu báo lỗi. Tra không ra thì câu báo lỗi ngắn đi, chứ không được biến
// thành một lỗi thứ hai che mất lỗi thật.
func aiGiuCong(cong int) string {
	pid := pidGiuCong(cong)
	if pid == "" {
		return ""
	}
	ten, dong := moTaTienTrinh(pid)
	if ten == "" {
		return "PID " + pid
	}
	if dong == "" {
		return fmt.Sprintf("%s (PID %s)", ten, pid)
	}
	return fmt.Sprintf("%s (PID %s): %s", ten, pid, dong)
}

func pidGiuCong(cong int) string {
	ra, err := exec.Command("powershell", "-NoProfile", "-Command",
		"(Get-NetTCPConnection -LocalPort "+strconv.Itoa(cong)+
			" -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1).OwningProcess").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(ra))
}

func moTaTienTrinh(pid string) (ten, dong string) {
	ra, err := exec.Command("powershell", "-NoProfile", "-Command",
		"$p = Get-CimInstance Win32_Process -Filter \"ProcessId="+pid+"\"; "+
			"if ($p) { $p.Name + \"`n\" + $p.CommandLine }").Output()
	if err != nil {
		return "", ""
	}
	d := strings.SplitN(strings.TrimSpace(string(ra)), "\n", 2)
	ten = strings.TrimSpace(d[0])
	if len(d) > 1 {
		dong = strings.TrimSpace(d[1])
	}
	return ten, dong
}

// loiCongBan dựng câu báo lỗi đầy đủ cho trường hợp không mở được cổng.
func loiCongBan(cong int, err error) error {
	ai := aiGiuCong(cong)
	if ai == "" {
		return fmt.Errorf("không mở được cổng %d: %w (thử --port khác)", cong, err)
	}
	// Nói luôn cách dừng nó. Người đọc câu này đang muốn cổng đó, không muốn đi
	// tra cứu cách tra tiến trình trên Windows.
	return fmt.Errorf("không mở được cổng %d — đang bị %s giữ.\n"+
		"     Dừng nó:  Stop-Process -Id %s -Force\n"+
		"     Hoặc dùng cổng khác:  --port <số>\n"+
		"     (lỗi gốc: %v)", cong, ai, pidGiuCong(cong), err)
}
