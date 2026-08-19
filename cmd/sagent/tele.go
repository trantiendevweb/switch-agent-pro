package main

import (
	"context"
	"fmt"

	"github.com/trantiendevweb/switch-agent-pro/internal/tele"
)

// cmdTele — mặt terminal của action "tele.notify".
//
//	sagent tele                                 xem trạng thái
//	sagent tele --set-token <token> --chat <id> đặt bot + nơi nhận
//	sagent tele --thu                           gửi một tin thử
//
// Token nhận qua ĐỐI SỐ chứ không hỏi tương tác, để việc này tự động hoá được
// (script cài máy mới). Đổi lại nó nằm trong lịch sử shell — nên lệnh in luôn
// lời nhắc xoá dòng đó, chứ không im lặng để người dùng tự dẫm phải.
func cmdTele(args []string) {
	thu, args := boolFlag(args, "--thu")
	token, args := strFlag(args, "--set-token", "")
	chat, _ := strFlag(args, "--chat", "")

	a, done := open()
	defer done()

	if token != "" {
		if err := a.TeleDat(token, chat); err != nil {
			fail(err)
		}
		done()
		fmt.Println()
		fmt.Printf("  Đã lưu vào %s (ngoài repo).\n", tele.ConfigPath())
		fmt.Println("  Token vừa gõ còn nằm trong lịch sử shell — nên xoá dòng đó đi.")
		fmt.Println("  Thử ngay: sagent tele --thu")
		fmt.Println()
		return
	}

	if thu {
		if err := a.TeleThu(context.Background()); err != nil {
			fail(err)
		}
		done()
		fmt.Println()
		fmt.Println("  Đã gửi tin thử. Không thấy gì trong Telegram thì kiểm lại chat id.")
		fmt.Println()
		return
	}

	da, chatID, duongDan := a.TeleTrangThai()
	done()
	fmt.Println()
	fmt.Println("  Báo tin Telegram khi lượt chạy flow có sự cố")
	fmt.Println()
	if !da {
		fmt.Println("  Trạng thái: CHƯA CẤU HÌNH — máy sẽ im lặng, lượt chạy không bị ảnh hưởng.")
		fmt.Println()
		fmt.Println("  Bật lên:")
		fmt.Println("    1. Nhắn @BotFather trên Telegram, gõ /newbot, lấy token.")
		fmt.Println("    2. Nhắn cho bot vừa tạo một câu bất kỳ.")
		fmt.Println("    3. sagent tele --set-token <token> --chat <chat id>")
		fmt.Printf("  Chat id lấy ở: https://api.telegram.org/bot<token>/getUpdates\n")
	} else {
		fmt.Println("  Trạng thái: ĐÃ CẤU HÌNH")
		fmt.Printf("  Nhắn tới:   %s\n", chatID)
		fmt.Println("  Nhắn khi:   bước hỏng · lượt chạy hỏng · lượt chạy chờ duyệt · lượt chạy xong")
		fmt.Println("  Gửi thử:    sagent tele --thu")
	}
	fmt.Printf("  Cấu hình:   %s (KHÔNG nằm trong repo)\n", duongDan)
	fmt.Println()
}
