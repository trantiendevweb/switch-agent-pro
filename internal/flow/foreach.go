package flow

import (
	"fmt"
	"strconv"
	"strings"
)

// MaxForEachItems là trần số lượt chạy của một bước `foreach`.
//
// Có trần vì nguồn thường là output của bước trước — một agent lỡ in ra 5000
// dòng sẽ biến thành 5000 lượt gọi agent và đốt sạch hạn mức trước khi ai kịp
// nhận ra. Vượt trần thì DỪNG và báo, chứ không âm thầm cắt bớt.
const MaxForEachItems = 50

// Items tách nguồn của `foreach` thành danh sách.
//
// Bỏ dòng trống và khoảng trắng thừa: nguồn hay là output của lệnh/agent, luôn
// có dòng rỗng ở cuối.
func Items(step Step, c Ctx) ([]string, error) {
	raw, err := resolve(step.ForEach, c)
	if err != nil {
		return nil, fmt.Errorf("foreach: %w", err)
	}
	sep := step.Separator
	if sep == "" {
		sep = "\n"
	}
	var out []string
	for _, part := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), sep) {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil, nil // không có gì để lặp — bước sẽ bị bỏ qua
	}
	if len(out) > MaxForEachItems {
		return nil, fmt.Errorf("foreach có %d mục, vượt trần %d — thu hẹp nguồn lại "+
			"(mỗi mục là một lượt chạy thật, rất tốn)", len(out), MaxForEachItems)
	}
	return out, nil
}

// itemVars trả về bản sao vars có thêm {{item}} và {{index}} cho một lượt.
func itemVars(vars map[string]string, item string, index int) map[string]string {
	out := make(map[string]string, len(vars)+2)
	for k, v := range vars {
		out[k] = v
	}
	out["item"] = item
	out["index"] = strconv.Itoa(index + 1)
	return out
}
