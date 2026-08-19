// Chạy MỘT bước: đợt song song, foreach, retry/timeout, và thực thi theo loại node.
package flow

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// runWave trả về ID bước làm cả đợt dừng và LÝ DO, tách làm hai chứ không dán
// thành một chuỗi: mặt nào muốn nhắc đúng tên bước (báo Telegram, workflow
// board) thì phải có tên bước sạch, không lẫn với câu mô tả lỗi.
func (r *Runner) runWave(ctx context.Context, runID int64, f Flow, work []Step,
	vars map[string]string, st *runState) (buoc, ly string) {

	limit := r.MaxParallel
	if limit < 1 {
		limit = 4
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	stopAt, stopLy := "", ""

	// Một bước hỏng với on_failure=stop thì HUỶ luôn các bước cùng đợt: chúng
	// sắp bị bỏ đi anyway, để chạy tiếp chỉ tốn hạn mức.
	waveCtx, cancelWave := context.WithCancel(ctx)
	defer cancelWave()

	if len(work) > 1 {
		r.Bus.Infof("chạy song song %d bước: %s", len(work), stepIDs(work))
	}

	for _, s := range work {
		s := s
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// Điều kiện `when`: không thoả thì bỏ qua, bước sau vẫn chạy.
			states, outs := st.snapshot()
			if s.When != "" {
				ok, err := Eval(s.When, Ctx{Vars: vars, States: states, Outputs: outs})
				if err != nil {
					_ = r.DB.SetStep(runID, s.ID, store.StepFailed, "điều kiện sai: "+err.Error(), 0)
					st.set(s.ID, store.StepFailed, "")
					r.baoBuocHong(runID, f, s, "điều kiện sai: "+err.Error())
					mu.Lock()
					if stopAt == "" && s.OnFailure != OnFailContinue {
						stopAt, stopLy = s.ID, "điều kiện sai: "+err.Error()
					}
					mu.Unlock()
					return
				}
				if !ok {
					_ = r.DB.SetStep(runID, s.ID, store.StepSkipped, "điều kiện không thoả: "+s.When, 0)
					st.set(s.ID, store.StepSkipped, "")
					r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID,
						SessionID: runID, Msg: "bỏ qua — " + s.When})
					return
				}
			}

			// foreach: một bước, nhiều lượt chạy trên một danh sách.
			if s.ForEach != "" {
				items, err := Items(s, Ctx{Vars: vars, States: states, Outputs: outs})
				if err != nil {
					_ = r.DB.SetStep(runID, s.ID, store.StepFailed, err.Error(), 0)
					st.set(s.ID, store.StepFailed, "")
					r.baoBuocHong(runID, f, s, err.Error())
					mu.Lock()
					if stopAt == "" && s.OnFailure != OnFailContinue {
						stopAt, stopLy = s.ID, err.Error()
					}
					mu.Unlock()
					return
				}
				if len(items) == 0 {
					_ = r.DB.SetStep(runID, s.ID, store.StepSkipped, "danh sách rỗng", 0)
					st.set(s.ID, store.StepSkipped, "")
					r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID,
						SessionID: runID, Msg: "bỏ qua — danh sách rỗng"})
					return
				}
				state, msg, out := r.runForEach(waveCtx, runID, f, s, vars, outs, items)
				st.set(s.ID, state, out)
				if state == store.StepFailed && s.OnFailure != OnFailContinue && s.OnFailure != OnFailFallback {
					mu.Lock()
					if stopAt == "" {
						stopAt, stopLy = s.ID, msg
					}
					mu.Unlock()
					cancelWave()
				}
				return
			}

			state, msg, out := r.runStep(waveCtx, runID, f, s, vars, outs)
			st.set(s.ID, state, out)

			if state == store.StepFailed {
				switch s.OnFailure {
				case OnFailContinue:
					r.Bus.Warnf("%s.%s hỏng nhưng on_failure=continue — đi tiếp", f.Name, s.ID)
				case OnFailFallback:
					r.Bus.Warnf("%s.%s hỏng — bước %s sẽ chạy thay", f.Name, s.ID, s.Fallback)
				default:
					mu.Lock()
					if stopAt == "" {
						stopAt, stopLy = s.ID, msg
					}
					mu.Unlock()
					cancelWave() // dừng các bước cùng đợt, khỏi tốn thêm
				}
			}
		}()
	}
	wg.Wait()
	return stopAt, stopLy
}

// baoBuocHong phát event "bước hỏng" ĐỦ THÔNG TIN để mặt khác dùng lại được.
//
// CÓ BẮN CẢ KHI on_failure = continue, và đó là CỐ Ý — không phải sót.
// Hàm này chạy trong runStep, tức TRƯỚC chỗ runWave xét OnFailure. Một bước hỏng
// mà lượt chạy vẫn đi tiếp thì người ở xa CÀNG cần biết: lượt sẽ kết thúc
// "completed" và không còn dấu vết nào nổi lên. Đo tại lần chạy #31 — `code-go`
// hỏng vì hết hạn đăng nhập, lượt vẫn `completed`, và nếu chỉ báo lúc cả lượt
// hỏng thì tin nhắn đó không bao giờ được gửi.
//
// Đổi lại là ồn: flow `dem` có bốn bước khai `continue`, xấu nhất là bốn tin cho
// một lượt. Chấp nhận, vì mất một tin báo hỏng đắt hơn nhận thừa một tin.
//
// Bus.Failuref chỉ đẻ ra một dòng chữ — đủ cho terminal, vì người đang nhìn
// terminal đã biết mình vừa chạy lượt nào. Người nhận tin Telegram lúc 2 giờ
// sáng thì không: họ cần số lượt chạy, tên bước và tài khoản mới mở đúng chỗ mà
// xem. Msg giữ NGUYÊN dạng cũ nên phần in ra màn hình không đổi.
func (r *Runner) baoBuocHong(runID int64, f Flow, s Step, ly string) {
	r.Bus.Publish(events.Event{
		Type:      events.Failure,
		Addr:      f.Name + "." + s.ID,
		SessionID: runID,
		Msg:       fmt.Sprintf("%s.%s: %s", f.Name, s.ID, ly),
		Detail: map[string]string{
			"flow":    f.Name,
			"run":     fmt.Sprint(runID),
			"step":    s.ID,
			"ly_do":   ly,
			"profile": r.taiKhoan(s),
			// Hai khoa duoi cho man hoi thoai: no can biet DUNG O NAO vua doi
			// trang thai de ve lai, thay vi nap lai ca luot chay.
			"state": store.StepFailed,
			"type":  s.Type,
		},
	})
}

// taiKhoan là tài khoản bước sẽ chạy bằng: khai trong bước, hoặc mặc định của
// lượt chạy. Bước không dùng agent thì trả rỗng — nói bừa một cái tên tài khoản
// còn tệ hơn không nói (nguyên tắc #6: chưa biết thì đừng đoán).
func (r *Runner) taiKhoan(s Step) string {
	if s.Type != TypeAgent && s.Type != TypeReview {
		return ""
	}
	if s.Profile != "" {
		return s.Profile
	}
	return r.DefaultProfile
}

// runForEach chạy một bước lặp trên danh sách, các lượt SONG SONG theo trần.
//
// Kết quả gộp lại có đánh dấu từng mục, để bước sau đọc `{{steps.x.output}}`
// vẫn biết mục nào ra kết quả gì.
func (r *Runner) runForEach(ctx context.Context, runID int64, f Flow, s Step,
	vars map[string]string, outs map[string]string, items []string) (state, msg, output string) {

	r.Bus.Infof("%s.%s lặp trên %d mục", f.Name, s.ID, len(items))
	_ = r.DB.SetStep(runID, s.ID, store.StepRunning, fmt.Sprintf("lặp %d mục", len(items)), 1)

	limit := r.MaxParallel
	if limit < 1 {
		limit = 4
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex

	results := make([]string, len(items))
	var firstErr string
	var chiPhi float64
	var tokVao, tokRa int // cộng dồn chi phí mọi lượt của bước lặp

	for i, item := range items {
		i, item := i, item
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			env := WithOutputs(itemVars(vars, item, i), outs)

			stepCtx := ctx
			var cancel context.CancelFunc
			if s.TimeoutSec > 0 {
				stepCtx, cancel = context.WithTimeout(ctx, time.Duration(s.TimeoutSec)*time.Second)
			}
			kq, err := r.do(stepCtx, s, env)
			if cancel != nil {
				cancel()
			}

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == "" {
					firstErr = fmt.Sprintf("mục %d (%s): %v", i+1, short(item, 40), err)
				}
				results[i] = "=== " + item + " === LỖI: " + err.Error()
				return
			}
			results[i] = "=== " + item + " ===\n" + kq.Output
			chiPhi += kq.ChiPhiUSD
			tokVao += kq.TokenVao
			tokRa += kq.TokenRa
		}()
	}
	wg.Wait()

	combined := strings.TrimSpace(strings.Join(results, "\n"))
	if firstErr != "" {
		_ = r.DB.SetStep(runID, s.ID, store.StepFailed, firstErr, 1)
		if combined != "" {
			_ = r.DB.SetStepOutput(runID, s.ID, combined)
		}
		r.baoBuocHong(runID, f, s, firstErr)
		return store.StepFailed, firstErr, combined
	}
	_ = r.DB.SetStep(runID, s.ID, store.StepDone, fmt.Sprintf("xong %d mục", len(items)), 1)
	_ = r.DB.SetStepOutput(runID, s.ID, combined)
	if chiPhi > 0 || tokVao > 0 || tokRa > 0 {
		_ = r.DB.SetStepCost(runID, s.ID, chiPhi, tokVao, tokRa)
	}
	r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID,
		SessionID: runID, Msg: fmt.Sprintf("xong %d mục", len(items))})
	return store.StepDone, "", combined
}

func short(s string, n int) string {
	r := []rune(strings.ReplaceAll(s, "\n", " "))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n-1]) + "…"
}

func stepIDs(ss []Step) string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.ID
	}
	return strings.Join(out, ", ")
}

// runStep chạy một bước, có timeout và retry.
func (r *Runner) runStep(ctx context.Context, runID int64, f Flow, s Step,
	vars map[string]string, outputs map[string]string) (state, msg, output string) {
	// Bước sau dùng được kết quả bước trước.
	env := WithOutputs(vars, outputs)
	tries := s.Retry + 1
	if tries < 1 {
		tries = 1
	}
	var lastErr error
	var kqCuoi KetQuaAgent // giữ kết quả lần thử cuối, kể cả khi nó hỏng
	for attempt := 1; attempt <= tries; attempt++ {
		_ = r.DB.SetStep(runID, s.ID, store.StepRunning, "", attempt)
		// Lưu CÂU HỎI trước khi chạy, không phải sau: bước có thể treo hoặc bị
		// cắt ngang, mà lúc đó câu hỏi lại là thứ cần nhất để hiểu vì sao.
		_ = r.DB.SetStepPrompt(runID, s.ID, cauHoi(s, env))
		r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID, SessionID: runID,
			Msg: fmt.Sprintf("chạy [%s] lần %d/%d", s.Type, attempt, tries),
			// Mặt web cần biết ĐÚNG Ô NÀO vừa đổi để cập nhật, thay vì nạp lại
			// cả lượt chạy mỗi lần có một dòng sự kiện.
			Detail: map[string]string{
				"run": fmt.Sprint(runID), "step": s.ID,
				"state": store.StepRunning, "profile": s.Profile, "type": s.Type,
			}})

		stepCtx := ctx
		var cancel context.CancelFunc
		if s.TimeoutSec > 0 {
			stepCtx, cancel = context.WithTimeout(ctx, time.Duration(s.TimeoutSec)*time.Second)
		}
		kq, err := r.do(stepCtx, s, env)
		if cancel != nil {
			cancel()
		}
		kqCuoi = kq

		if err == nil {
			_ = r.DB.SetStep(runID, s.ID, store.StepDone, "", attempt)
			if kq.Output != "" {
				_ = r.DB.SetStepOutput(runID, s.ID, kq.Output)
			}
			if kq.ChiPhiUSD > 0 || kq.TokenVao > 0 || kq.TokenRa > 0 {
				_ = r.DB.SetStepCost(runID, s.ID, kq.ChiPhiUSD, kq.TokenVao, kq.TokenRa)
			}
			r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID,
				SessionID: runID, Msg: "xong",
				Detail: map[string]string{
					"run": fmt.Sprint(runID), "step": s.ID,
					"state": store.StepDone, "profile": s.Profile, "type": s.Type,
				}})
			return store.StepDone, "", kq.Output
		}
		lastErr = err
		if attempt < tries {
			// lùi dần: 2s, 4s, 6s… đủ để thứ tạm thời tự khỏi
			wait := time.Duration(attempt*2) * time.Second
			r.Bus.Warnf("%s.%s hỏng (%v) — thử lại sau %s", f.Name, s.ID, err, wait)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				break
			}
		}
	}
	emsg := lastErr.Error()
	_ = r.DB.SetStep(runID, s.ID, store.StepFailed, emsg, tries)
	// GIỮ output kể cả khi hỏng. Trước đây SetStepOutput chỉ nằm ở nhánh thành
	// công, nên đúng lúc cần đọc agent nói gì nhất thì không còn gì để đọc — đo
	// tại lượt #35: bước `code-doc` hỏng, tôi phải đi đào fleet.log mới biết
	// antigravity trả status ERROR. Bằng chứng phải còn lại ở chỗ người ta tìm.
	if kqCuoi.Output != "" {
		_ = r.DB.SetStepOutput(runID, s.ID, kqCuoi.Output)
	}
	r.baoBuocHong(runID, f, s, emsg)
	return store.StepFailed, emsg, ""
}

// do thực thi đúng một lần, theo loại node.
func (r *Runner) do(ctx context.Context, s Step, vars map[string]string) (KetQuaAgent, error) {
	switch s.Type {
	case TypeAgent, TypeReview:
		if r.Agent == nil {
			return KetQuaAgent{}, fmt.Errorf("không có bộ chạy agent")
		}
		n := s.Copies
		if n < 1 {
			n = 1
		}
		if r.MaxParallel > 0 && n > r.MaxParallel {
			r.Bus.Warnf("copies=%d vượt trần %d của dự án — hạ xuống", n, r.MaxParallel)
			n = r.MaxParallel
		}
		return r.Agent.RunAgents(ctx, s.Profile, s.Model, ExpandChay(s.Prompt, vars), n, s.Worktree, s.TuDuyetQuyen)

	case TypeShell, TypeTest, TypeLint:
		argv := s.Run
		if len(argv) == 0 {
			// test/lint không cần khai báo lệnh: lấy từ .sagent/project.toml
			switch s.Type {
			case TypeTest:
				argv = r.Commands["test"]
			case TypeLint:
				argv = r.Commands["lint"]
			}
			if len(argv) == 0 {
				return KetQuaAgent{}, fmt.Errorf("bước %s cần `run`, hoặc khai `commands.%s` trong .sagent/project.toml", s.Type, s.Type)
			}
		}
		s.Run = argv
		// argv, KHÔNG qua shell — flow là file người ta gửi cho nhau được.
		//
		// Bước shell KHÔNG được chốt placeholder như prompt: `go test -C
		// (bước "x" không để lại kết quả)` là một đường dẫn bịa, chạy vào rồi
		// hỏng bằng một thông báo chẳng liên quan gì tới nguyên nhân thật.
		// Thiếu giá trị thì dừng ngay và nói rõ thiếu của bước nào.
		args := make([]string, len(s.Run))
		for i, a := range s.Run {
			if id := BuocConSot(a, vars); id != "" {
				return KetQuaAgent{}, fmt.Errorf(
					"tham số %d cần kết quả của bước %q nhưng bước đó không để lại gì", i+1, id)
			}
			args[i] = Expand(a, vars)
		}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		raw, err := cmd.CombinedOutput()
		if err != nil {
			line := strings.TrimSpace(lastLine(string(raw)))
			if line != "" {
				return KetQuaAgent{}, fmt.Errorf("%v — %s", err, line)
			}
			return KetQuaAgent{}, err
		}
		return KetQuaAgent{Output: strings.TrimRight(string(raw), "\r\n")}, nil

	case TypeNotify:
		m := ExpandChay(s.Message, vars)
		r.Bus.Infof("%s", m)
		return KetQuaAgent{Output: m}, nil

	default:
		return KetQuaAgent{}, fmt.Errorf("type %q chưa chạy được ở bản này", s.Type)
	}
}

// Approve đánh dấu một bước approve là ĐÃ DUYỆT.
//
// Đây là hàm DUY NHẤT chuyển một bước approve sang `done`. Bộ thực thi không có
// đường nào tự làm việc đó — nhờ vậy "approval không thể bị bỏ qua" là tính chất
// của kiến trúc chứ không phải một cái cờ ai cũng bật được.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	return lines[len(lines)-1]
}

// cauHoi dựng lại ĐÚNG thứ bước này gửi đi, sau khi đã thay hết biến.
//
// Không dùng lại s.Prompt thô trong flows.toml: cái người ta cần đọc lại là thứ
// agent THẬT SỰ nhận. Lượt chạy #29 cho thấy khoảng cách giữa hai thứ đó có thể
// là cả một lỗi — mẫu ghi `{{steps.kiem-cuoi.output}}`, thứ gửi đi cũng đúng
// chuỗi đó vì bước kia không để lại kết quả.
//
// Bước không hỏi ai (shell/notify) vẫn lưu, vì trong dòng hội thoại chúng là
// tiếng nói của MÁY — "tôi chạy lệnh này" — và bỏ đi thì mạch đứt quãng.
func cauHoi(s Step, vars map[string]string) string {
	switch s.Type {
	case TypeAgent, TypeReview:
		return ExpandChay(s.Prompt, vars)
	case TypeNotify:
		return ExpandChay(s.Message, vars)
	case TypeShell, TypeTest, TypeLint:
		if len(s.Run) == 0 {
			return ""
		}
		args := make([]string, len(s.Run))
		for i, a := range s.Run {
			args[i] = Expand(a, vars)
		}
		return strings.Join(args, " ")
	}
	return ""
}
