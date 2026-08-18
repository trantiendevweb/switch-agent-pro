// Trạng thái trong lúc chạy một flow.
//
// Tách khỏi runner.go vì một đợt chạy nhiều bước song song — tất cả cùng đọc/ghi
// vào đây, nên phần khoá cần đứng riêng cho dễ soi.
package flow

import (
	"sync"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// runState giữ trạng thái/kết quả trong lúc chạy. Có khoá vì một đợt chạy nhiều
// bước song song, tất cả cùng ghi vào đây.
type runState struct {
	mu      sync.Mutex
	states  map[string]string
	outputs map[string]string
}

func (s *runState) set(id, state, out string) {
	s.mu.Lock()
	s.states[id] = state
	if out != "" {
		s.outputs[id] = out
	}
	s.mu.Unlock()
}

func (s *runState) snapshot() (map[string]string, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := make(map[string]string, len(s.states))
	b := make(map[string]string, len(s.outputs))
	for k, v := range s.states {
		a[k] = v
	}
	for k, v := range s.outputs {
		b[k] = v
	}
	return a, b
}

func (s *runState) finished(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.states[id]
	return v == store.StepDone || v == store.StepSkipped
}

func (s *runState) state(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.states[id]
}

// readySteps tìm các bước có thể chạy ngay. waiting là id bước approve đang chặn
// (nếu có) để lời gọi biết vì sao hết việc.
func (s *runState) readySteps(steps []Step) (ready []Step, waiting string) {
	for _, step := range steps {
		if s.state(step.ID) != "" && (s.finished(step.ID) || s.state(step.ID) == store.StepFailed) {
			continue
		}
		if s.state(step.ID) == store.StepWaiting {
			waiting = step.ID
			continue
		}
		ok := true
		for _, n := range step.Needs {
			if s.finished(n) || choDiTiep(steps, n, s.state(n)) {
				continue
			}
			ok = false
			if s.state(n) == store.StepWaiting {
				waiting = n
			}
			break
		}
		if ok {
			ready = append(ready, step)
		}
	}
	return ready, waiting
}

// runWave chạy một đợt bước song song, có trần đồng thời. Trả về id bước làm cả
// flow phải dừng (rỗng nếu không có).

// choDiTiep: bước phụ thuộc HỎNG nhưng chính nó khai `on_failure = "continue"`
// thì vẫn cho bước sau chạy.
//
// Trước đây không có hàm này và hậu quả im lặng: `finished()` chỉ tính done/skipped,
// nên một bước cha `failed` khoá cứng mọi bước con — chúng không bao giờ đủ điều
// kiện, runner hết việc, và lần chạy vẫn được ghi là `completed`. Đo tại lần chạy
// #23: `hoc-acp` hỏng, `tong-hop` (cần cả ba bản nghiên cứu) hiện "(chưa chạy)"
// mà không một dòng cảnh báo nào.
//
// Người viết flow gõ "continue" là đã nói rõ ý: hỏng thì cứ đi tiếp. Chặn bước
// sau lại là làm ngược ý họ, và tệ hơn là làm ngược trong im lặng.
func choDiTiep(steps []Step, id, state string) bool {
	if state != store.StepFailed {
		return false
	}
	for _, st := range steps {
		if st.ID == id {
			return st.OnFailure == OnFailContinue
		}
	}
	return false
}
