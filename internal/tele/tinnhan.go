package tele

import (
	"fmt"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
)

// tranKyTu là trần độ dài một tin. Telegram cắt ở 4096 ký tự và trả lỗi khi
// vượt; cắt sẵn ở đây thì phần QUAN TRỌNG (lượt chạy, bước, lệnh gõ tiếp) luôn
// tới nơi, chỉ có đuôi log dài mới bị bỏ.
const tranKyTu = 3500

// tranLyDo giữ phần "lý do" đủ dài để hiểu bệnh mà không nuốt cả tin nhắn.
const tranLyDo = 600

// TinNhan biến một event thành nội dung tin nhắn.
//
// Trả về chuỗi RỖNG nghĩa là "không đáng nhắn" — và đó là mặc định. Đây là chỗ
// duy nhất quyết định gửi hay không, nên luật "không spam" kiểm được bằng test
// mà không cần dựng mạng.
func TinNhan(e events.Event) string {
	switch e.Type {
	case events.FlowFailed:
		return dung("LƯỢT CHẠY HỎNG", "Lý do", e, []string{
			"Xem lại:  sagent flow runs",
			"Chi tiết: sagent flow show " + tenFlow(e),
		})

	case events.FlowWaiting:
		buoc := buocCua(e)
		return dung("CHỜ DUYỆT", "Nội dung", e, []string{
			fmt.Sprintf("Duyệt:    sagent flow approve %d %s", e.SessionID, buoc),
			fmt.Sprintf("Từ chối:  sagent flow reject %d %s", e.SessionID, buoc),
		})

	case events.FlowDone:
		// Không kèm dòng lý do: Msg của event xong chỉ là "xong #N", lặp lại
		// đúng thứ dòng "Lượt chạy" đã nói.
		return dung("LƯỢT CHẠY XONG", "", e, []string{
			"Xem lại: sagent flow runs",
		})

	case events.Failure:
		// Failure là loại event dùng chung (phiên không dừng được, mồ côi không
		// giết được…). Chỉ cái nào MANG THEO số lượt chạy và tên bước mới là
		// "bước hỏng" của flow — những cái khác không thuộc phạm vi báo tin này.
		if e.SessionID == 0 || buocCua(e) == "" {
			return ""
		}
		return dung("BƯỚC HỎNG", "Lý do", e, []string{
			"Xem lại: sagent flow runs",
			fmt.Sprintf("Chạy tiếp sau khi sửa: sagent flow resume %d", e.SessionID),
		})
	}
	return ""
}

// dung ghép tin theo một khuôn duy nhất, để mọi tin đều trả lời đủ bốn câu
// "lượt chạy nào — bước nào — tài khoản nào — vì sao", rồi mới tới lệnh gõ tiếp.
func dung(tieuDe, nhan string, e events.Event, lenh []string) string {
	var b strings.Builder
	b.WriteString("Switch-Agent-Pro — " + tieuDe + "\n")

	if flow := tenFlow(e); flow != "" {
		if e.SessionID > 0 {
			fmt.Fprintf(&b, "Lượt chạy: #%d · flow %q\n", e.SessionID, flow)
		} else {
			fmt.Fprintf(&b, "Flow: %q\n", flow)
		}
	} else if e.SessionID > 0 {
		fmt.Fprintf(&b, "Lượt chạy: #%d\n", e.SessionID)
	}

	if buoc := buocCua(e); buoc != "" {
		b.WriteString("Bước: " + buoc + "\n")
	}
	if tk := e.Detail["profile"]; tk != "" {
		b.WriteString("Tài khoản: " + tk + "\n")
	}
	if nhan != "" {
		if ly := lyDo(e); ly != "" {
			b.WriteString(nhan + ": " + ly + "\n")
		}
	}
	if len(lenh) > 0 {
		b.WriteString("\n" + strings.Join(lenh, "\n") + "\n")
	}
	return cat(b.String(), tranKyTu)
}

// tenFlow lấy tên flow. Addr của event flow là "ten-flow" hoặc "ten-flow.buoc"
// (xem internal/flow), nên Detail là nguồn chuẩn còn Addr là đường lùi.
func tenFlow(e events.Event) string {
	if v := e.Detail["flow"]; v != "" {
		return v
	}
	if i := strings.IndexByte(e.Addr, '.'); i >= 0 {
		return e.Addr[:i]
	}
	return e.Addr
}

func buocCua(e events.Event) string {
	if v := e.Detail["step"]; v != "" {
		return v
	}
	if i := strings.IndexByte(e.Addr, '.'); i >= 0 {
		return e.Addr[i+1:]
	}
	return ""
}

// lyDo ưu tiên lỗi có cấu trúc; không có thì dùng Msg — thà thừa còn hơn gửi
// một tin báo hỏng mà không nói hỏng vì cái gì.
func lyDo(e events.Event) string {
	if v := e.Detail["ly_do"]; v != "" {
		return cat(v, tranLyDo)
	}
	return cat(e.Msg, tranLyDo)
}

func cat(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…(đã cắt bớt)"
}
