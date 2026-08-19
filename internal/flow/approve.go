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

// Huy đánh dấu một lần chạy là ĐÃ HUỶ, kể cả khi không có bước nào đang chờ duyệt.
//
// Vì sao cần: `Reject` chỉ hạ được một bước ĐANG CHỜ DUYỆT. Lần chạy bị cắt
// ngang giữa chừng — máy sập, hoặc người gõ Ctrl-C — thì không bước nào ở trạng
// thái đó, nên nó nằm lại `running` VĨNH VIỄN. Đo hai lần trong ngày 19/08:
// #29 chết theo lần máy tự khởi động lại lúc 01:47, #30 chết khi người dùng
// dừng tay lúc 19:37. Cả hai đều hiện "đang chạy" trong khi không tiến trình nào
// còn sống — bảng lịch sử nói dối đúng thứ nó sinh ra để nói thật.
//
// KHÔNG tự đi giết tiến trình: sổ trạng thái và tiến trình là hai chuyện, gộp
// vào một lệnh thì người dùng không biết mình vừa làm cái nào. Dừng tiến trình
// là việc của `sagent stop`.
func (r *Runner) Huy(runID int64, by string) error {
	run, err := r.DB.GetRun(runID)
	if err != nil {
		return fmt.Errorf("không có lần chạy #%d", runID)
	}
	if run.State == store.RunDone || run.State == store.RunCanceled {
		return fmt.Errorf("lần chạy #%d đã ở trạng thái %q — không huỷ được nữa", runID, run.State)
	}
	steps, err := r.DB.Steps(runID)
	if err != nil {
		return err
	}
	for id, s := range steps {
		if s.State != store.StepRunning && s.State != store.StepWaiting {
			continue
		}
		// Ghi rõ vì sao bước này đỏ. Để trống thì lần sau đọc lại sẽ tưởng nó
		// hỏng vì code, rồi đi sửa nhầm chỗ.
		_ = r.DB.SetStep(runID, id, store.StepFailed, "lượt chạy bị huỷ bởi "+by, s.Attempt)
	}
	if err := r.DB.SetRunState(runID, store.RunCanceled); err != nil {
		return err
	}
	r.Bus.Publish(events.Event{Type: events.FlowRejected, SessionID: runID,
		Msg: fmt.Sprintf("lượt chạy #%d bị huỷ bởi %s", runID, by)})
	return nil
}
