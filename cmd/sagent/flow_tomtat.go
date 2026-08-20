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
	if len(tt.MauThuan) > 0 {
		fmt.Printf("  %d chỗ lời agent chọi với git. Đọc nguyên văn từng bước: sagent flow runs %d\n\n",
			len(tt.MauThuan), id)
	}
}
