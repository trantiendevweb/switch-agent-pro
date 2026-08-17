// Cổng duyệt (approval gate).
//
// Approve() là hàm DUY NHẤT chuyển một bước approve sang `done`. Bộ thực thi
// không có nhánh nào tự làm việc đó — nhờ vậy "approval không thể bị bỏ qua" là
// tính chất của kiến trúc, không phải một cái cờ ai cũng bật được.
package flow

import (
	"fmt"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

func (r *Runner) Approve(runID int64, stepID string, by string) error {
	steps, err := r.DB.Steps(runID)
	if err != nil {
		return err
	}
	st, ok := steps[stepID]
	if !ok || st.State != store.StepWaiting {
		return fmt.Errorf("bước %q không ở trạng thái chờ duyệt", stepID)
	}
	if err := r.DB.SetStep(runID, stepID, store.StepDone, "duyệt bởi "+by, 0); err != nil {
		return err
	}
	r.Bus.Publish(events.Event{Type: events.FlowApproved, SessionID: runID,
		Addr: stepID, Msg: "đã duyệt bởi " + by})
	return nil
}

// Reject từ chối một bước approve — cả lần chạy dừng lại.
func (r *Runner) Reject(runID int64, stepID, by string) error {
	steps, err := r.DB.Steps(runID)
	if err != nil {
		return err
	}
	st, ok := steps[stepID]
	if !ok || st.State != store.StepWaiting {
		return fmt.Errorf("bước %q không ở trạng thái chờ duyệt", stepID)
	}
	if err := r.DB.SetStep(runID, stepID, store.StepFailed, "từ chối bởi "+by, 0); err != nil {
		return err
	}
	if err := r.DB.SetRunState(runID, store.RunCanceled); err != nil {
		return err
	}
	r.Bus.Publish(events.Event{Type: events.FlowRejected, SessionID: runID,
		Addr: stepID, Msg: "bị từ chối bởi " + by})
	return nil
}

