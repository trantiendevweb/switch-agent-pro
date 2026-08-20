package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/console"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// cmdNangLuc in bảng năng lực từng provider — action "provider.nang-luc".
//
//	sagent nang-luc            — mọi provider
//	sagent nang-luc grok       — một provider
//	sagent nang-luc --chua-do  — CHỈ những chỗ chưa ai đo
//
// Tách khỏi `verify` vì hai lệnh trả lời hai câu khác nhau, và trộn vào một
// bảng là mất cả hai: `verify` hỏi "trên MÁY NÀY có ổn không" (đã cài CLI chưa,
// token còn không), lệnh này hỏi "provider này LÀM ĐƯỢC GÌ" — câu trả lời giống
// nhau trên mọi máy, và phải trả lời được cả khi CLI chưa cài.
//
// `--chua-do` là cờ đáng gõ nhất: nó in ra đúng danh sách việc còn phải đo, tức
// là bản đồ những chỗ hệ thống đang phải đoán.
func cmdNangLuc(args []string) {
	chiChuaDo, args := boolFlag(args, "--chua-do")
	name := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name = args[0]
	}
	a, done := open()
	defer done()
	ds, err := a.NangLuc(name)
	if err != nil {
		fail(err)
	}
	done()

	// dau: ba trạng thái, ba ký hiệu KHÁC NHAU. Dùng chung một ký hiệu cho
	// "không làm được" và "chưa đo" là xoá mất đúng thứ bảng này sinh ra để nói.
	dau := map[provider.TrangThaiNangLuc]string{
		provider.LamDuoc:      "✓",
		provider.KhongLamDuoc: "✗",
		provider.ChuaDo:       "?",
	}
	code := 0
	var soChuaDo int
	for _, p := range ds {
		var dong []provider.NangLuc
		for _, m := range p.Muc {
			if m.TrangThai == provider.ChuaDo {
				soChuaDo++
			}
			if chiChuaDo && m.TrangThai != provider.ChuaDo {
				continue
			}
			dong = append(dong, m)
		}
		if len(dong) == 0 {
			continue
		}
		fmt.Printf("\n  [%s]\n", p.Provider)
		for _, m := range dong {
			fmt.Printf("    %s %-20s %s\n", dau[m.TrangThai], m.Khoa, m.BangChung)
		}
		// Chỗ lệch giữa lời khai và hành vi thật phải hiện NGAY dưới bảng, không
		// giấu sau một cờ. Bảng năng lực sai thì tệ hơn không có bảng.
		for _, l := range p.Lech {
			code = 1
			fmt.Printf("    ! LỆCH: %s\n", l)
		}
	}
	fmt.Println()
	if chiChuaDo && soChuaDo == 0 {
		fmt.Println("  Không còn năng lực nào chưa đo.")
		fmt.Println()
	} else if !chiChuaDo {
		fmt.Printf("  ✓ đã đo, làm được · ✗ đã đo, provider KHÔNG có · ? chưa ai đo (%d chỗ)\n", soChuaDo)
		fmt.Println("  Xem riêng phần chưa đo: sagent nang-luc --chua-do")
		fmt.Println()
	}
	console.KhoiPhuc()
	os.Exit(code)
}
