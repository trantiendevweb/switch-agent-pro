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

// Info mô tả một tiến trình đủ để người dùng nhìn ra nó là cái gì trước khi
// quyết định có giết hay không. Danh sách "sẽ giết" mà chỉ có mấy con số PID thì
// không ai duyệt được, và người ta sẽ bấm đồng ý cho xong.
type Info struct {
	PID    int
	Ten    string
	BatDau time.Time // zero = không đọc được (tiến trình của người dùng khác)
}

// MoCoi tìm tiến trình còn sống là hậu duệ của pid ĐÃ CHẾT, và chỉ nhận những
// đứa bắt đầu SAU mốc `sau`.
//
// Vì sao phải lọc theo thời gian: Windows dùng lại PID. Một tiến trình mới trùng
// PID với phiên đã chết sẽ kéo cả đám con của nó vào danh sách, và ta sẽ giết
// nhầm thứ chẳng liên quan. Con thật của phiên BẮT BUỘC phải sinh sau khi phiên
// bắt đầu — điều kiện đó không đủ để chắc chắn, nhưng đủ để loại phần lớn nhầm
// lẫn, và phần còn lại thì người dùng nhìn danh sách mà quyết.
//
// Tiến trình không đọc được thời điểm bắt đầu thì BỊ LOẠI, không phải được nhận:
// khi không biết, mặc định là không giết.
func MoCoi(pid int, sau time.Time) []Info {
	if IsAlive(pid) {
		return nil // cha còn sống thì đây là cây bình thường, không phải mồ côi
	}
	bang := procTable()
	con := map[int][]int{}
	for p, e := range bang {
		con[e.ppid] = append(con[e.ppid], p)
	}

	var out []Info
	seen := map[int]bool{pid: true}
	queue := []int{pid}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range con[cur] {
			if seen[c] {
				continue
			}
			seen[c] = true
			bd, ok := StartTime(c)
			if !ok || bd.Before(sau) {
				continue // không biết, hoặc có trước cả phiên -> không phải của ta
			}
			out = append(out, Info{PID: c, Ten: bang[c].ten, BatDau: bd})
			queue = append(queue, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PID < out[j].PID })
	return out
}
