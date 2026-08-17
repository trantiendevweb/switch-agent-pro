// Package process là lớp mỏng hỏi/giết tiến trình, tách theo nền tảng.
//
// Phần này KHÔNG theo nền tảng: cách dọn một cây tiến trình thì giống nhau, chỉ
// có cách hỏi "ai là con của ai" và cách giết là khác.
package process

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Descendants trả về mọi hậu duệ của pid (con, cháu, chắt), KHÔNG gồm chính pid.
//
// Giới hạn phải nói trước: quan hệ cha-con nhận diện bằng PID, mà PID thì được
// dùng lại. Một tiến trình mới trùng PID với cha đã chết sẽ kéo theo cả đám con
// của nó vào danh sách. Xác suất thấp nhưng không phải không có — nên hàm này
// chỉ dùng để DỌN THEO YÊU CẦU của người dùng (`stop`), không bao giờ tự động
// chạy nền.
func Descendants(pid int) []int {
	ppid := parentMap()
	con := map[int][]int{}
	for p, pp := range ppid {
		con[pp] = append(con[pp], p)
	}
	var out []int
	seen := map[int]bool{pid: true}
	queue := []int{pid}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range con[cur] {
			if seen[c] {
				continue // vòng lặp do PID dùng lại — đừng quay mãi
			}
			seen[c] = true
			out = append(out, c)
			queue = append(queue, c)
		}
	}
	sort.Ints(out)
	return out
}

// KillTree dừng pid và mọi hậu duệ, rồi KIỂM LẠI.
//
// Vì sao không dùng thẳng Kill: đã đo trên Windows, `taskkill /T` đi theo quan
// hệ cha-con của các tiến trình CÒN SỐNG. Nếu tiến trình cha đã thoát trước —
// agent tự chết, hoặc nó chỉ là cái vỏ khởi động rồi nhường chỗ cho con —
// thì con thành mồ côi, `taskkill` báo `exit status 128`, và đám mồ côi chạy
// tiếp. Chúng vẫn tiêu hạn mức của bạn, không ai biết.
//
// Nên: chụp danh sách hậu duệ TRƯỚC, giết cây, rồi quét lại đứa nào còn sống mà
// giết thẳng từng đứa. Và cuối cùng phải kiểm — một hàm dừng tiến trình mà trả
// nil trong khi tiến trình vẫn chạy thì tệ hơn là không có.
func KillTree(pid int) error {
	// Chụp trước: sau khi cha chết thì quan hệ cha-con còn đọc được trên Windows
	// nhưng đã mất trên Linux (con được init nhận nuôi, PPid đổi thành 1).
	hauDue := Descendants(pid)

	killErr := Kill(pid)

	// Quét lại. Kill có thể đã dọn hết — thường là thế khi cha còn sống.
	var songSot []int
	for _, c := range hauDue {
		if IsAlive(c) {
			songSot = append(songSot, c)
		}
	}
	for _, c := range songSot {
		_ = Kill(c)
	}

	// Cho hệ điều hành một nhịp để thật sự thu dọn rồi mới kết luận. Không có
	// chỗ chờ này thì hàm sẽ báo "còn sống" cho những tiến trình đang chết dở.
	if !doiChet(pid, 2*time.Second) {
		return fmt.Errorf("PID %d vẫn chạy sau khi dừng (%v)", pid, killErr)
	}
	var conLai []int
	for _, c := range hauDue {
		if IsAlive(c) {
			conLai = append(conLai, c)
		}
	}
	if len(conLai) > 0 {
		return fmt.Errorf("đã dừng PID %d nhưng còn %d tiến trình con sống sót: %s",
			pid, len(conLai), soList(conLai))
	}
	return nil
}

func doiChet(pid int, trong time.Duration) bool {
	han := time.Now().Add(trong)
	for {
		if !IsAlive(pid) {
			return true
		}
		if time.Now().After(han) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func soList(pids []int) string {
	s := make([]string, len(pids))
	for i, p := range pids {
		s[i] = fmt.Sprint(p)
	}
	return strings.Join(s, ", ")
}
