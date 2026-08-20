package main

import (
	"fmt"
	"strconv"
	"strings"
)

// flowTomTat in bản tóm tắt một lượt chạy — action "flow.tom-tat".
//
// Khác `flow runs <N>`: lệnh kia đổ RA HẾT những gì agent nói, lệnh này TRẢ LỜI.
// Ai làm gì, ai chưa làm, bước nào hỏng vì sao, việc gì còn treo — rồi tự đếm
// commit trên các nhánh `sagent/*` và đối chiếu với những gì agent khai. Lệch
// thì nói thẳng "lời agent mâu thuẫn với git" và tin git.
//
// Xem đầu internal/api/tomtat.go để biết vì sao không tin lời agent.
func flowTomTat(arg string) {
	id, err := strconv.ParseInt(strings.TrimPrefix(arg, "#"), 10, 64)
	if err != nil {
		fail(fmt.Errorf("số lần chạy phải là số, không phải %q. Ví dụ: sagent flow tom-tat 8", arg))
	}
	a, done := open()
	defer done()
	tt, err := a.FlowTomTat(id)
	if err != nil {
		fail(err)
	}
	done()

	fmt.Println()
	for _, d := range strings.Split(strings.TrimRight(tt.VanBan, "\n"), "\n") {
		fmt.Println("  " + d)
	}
	fmt.Println()
	// Đếm RIÊNG hai loại. Gộp chung thì một lượt đã trộn xong hiện "4 chỗ lời
	// agent chọi với git" trong khi không có chỗ nào chọi cả — đúng kiểu vu oan
	// làm người đọc thôi tin bộ dò.
	sai, chuaRo := 0, 0
	for _, m := range tt.MauThuan {
		if m.ChuaChacSai {
			chuaRo++
		} else {
			sai++
		}
	}
	switch {
	case sai > 0 && chuaRo > 0:
		fmt.Printf("  %d chỗ lời agent CHỌI với git, %d chỗ chưa kết luận được. "+
			"Đọc nguyên văn từng bước: sagent flow runs %d\n\n", sai, chuaRo, id)
	case sai > 0:
		fmt.Printf("  %d chỗ lời agent CHỌI với git. Đọc nguyên văn từng bước: sagent flow runs %d\n\n",
			sai, id)
	case chuaRo > 0:
		fmt.Printf("  Không chỗ nào chọi với git. %d chỗ chưa kết luận được vì nhánh "+
			"có thể đã trộn sau khi chạy.\n\n", chuaRo)
	}
}
