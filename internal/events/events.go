// Package events là luồng sự thật duy nhất cho mọi mặt điều khiển.
//
// Luật (MASTER-PLAN mục 2c, luật 3): terminal, dashboard 2D, workflow board và
// 3D đều nghe CÙNG luồng này. Không mặt nào được tự suy trạng thái bằng timer
// hay animation — trạng thái nào không có event thì không được hiển thị.
//
// Hệ quả với code: phần lõi KHÔNG in ra stdout. Nó phát event; CLI chỉ là mặt
// đầu tiên biết cách vẽ event ra màn hình.
package events

import (
	"fmt"
	"sync"
	"time"
)

var sprintf = fmt.Sprintf

// SchemaVersion đi kèm mọi event. Mặt nào đọc event cũng phải kiểm số này —
// sau này đổi cấu trúc thì client cũ biết mà từ chối thay vì hiểu sai.
const SchemaVersion = 1

// Type là loại event. Thêm loại mới thì NỐI VÀO CUỐI, đừng đổi chuỗi cũ:
// chuỗi này là một phần của hợp đồng với các mặt khác.
type Type string

const (
	ProfileCreated Type = "profile.created"
	ProfileRemoved Type = "profile.removed"
	ClonesCreated  Type = "clones.created"
	ClonesCleaned  Type = "clones.cleaned"
	SessionStarted Type = "session.started"
	SessionStopped Type = "session.stopped"
	WorktreeAdded  Type = "worktree.added"
	WorktreeKept   Type = "worktree.kept"
	WorktreeGone   Type = "worktree.removed"
	FlowStarted    Type = "flow.started"
	FlowStep       Type = "flow.step"
	FlowWaiting    Type = "flow.waiting_approval"
	FlowApproved   Type = "flow.approved"
	FlowRejected   Type = "flow.rejected"
	FlowDone       Type = "flow.completed"
	FlowFailed     Type = "flow.failed"
	Warning        Type = "warning"
	Failure        Type = "failure"
	Info           Type = "info"
)

// Event là một chuyện đã xảy ra.
//
// TUYỆT ĐỐI không đặt token, API key hay nội dung file config vào đây: event
// chạy thẳng tới dashboard, mà dashboard không được phép thấy secret.
type Event struct {
	V         int               `json:"v"`
	Type      Type              `json:"type"`
	Time      time.Time         `json:"time"`
	Addr      string            `json:"addr,omitempty"` // provider:account[#clone]
	SessionID int64             `json:"session_id,omitempty"`
	Msg       string            `json:"msg"`
	Detail    map[string]string `json:"detail,omitempty"`
}

// Bus phát event tới mọi người đang nghe. Trong tiến trình thôi — chưa cần
// daemon (xem quyết định "đường gọn", MASTER-PLAN mục 2b).
type Bus struct {
	mu     sync.Mutex
	subs   []chan Event
	closed bool
}

func NewBus() *Bus { return &Bus{} }

// Subscribe trả về kênh nhận event và hàm huỷ đăng ký.
//
// Kênh có đệm và Publish KHÔNG BAO GIỜ chặn: người nghe chậm thì mất event chứ
// không được phép làm treo phần lõi. Mặt nào cần đủ 100% thì phải đọc lại từ
// store, đó mới là nơi giữ trạng thái bền.
func (b *Bus) Subscribe(buffer int) (<-chan Event, func()) {
	if buffer <= 0 {
		buffer = 64
	}
	ch := make(chan Event, buffer)
	b.mu.Lock()
	b.subs = append(b.subs, ch)
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			for i, c := range b.subs {
				if c == ch {
					b.subs = append(b.subs[:i], b.subs[i+1:]...)
					close(c)
					return
				}
			}
		})
	}
	return ch, cancel
}

// Publish gửi event đi. Tự điền V và Time nếu chưa có.
func (b *Bus) Publish(e Event) {
	if e.V == 0 {
		e.V = SchemaVersion
	}
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default: // người nghe chậm: bỏ qua, không chặn lõi
		}
	}
}

// Close đóng mọi kênh đang nghe.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, ch := range b.subs {
		close(ch)
	}
	b.subs = nil
}

// Tiện dụng ------------------------------------------------------------

func (b *Bus) Infof(format string, args ...any)   { b.emit(Info, format, args...) }
func (b *Bus) Warnf(format string, args ...any)   { b.emit(Warning, format, args...) }
func (b *Bus) Failuref(format string, args ...any) { b.emit(Failure, format, args...) }

func (b *Bus) emit(t Type, format string, args ...any) {
	b.Publish(Event{Type: t, Msg: sprintf(format, args...)})
}
