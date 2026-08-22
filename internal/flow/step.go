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
			// Lọc quyền đọc NGAY ở đây, một chỗ duy nhất: `when`, `foreach` và
			// runStep đều lấy env từ chính `outs` này, nên lọc sau đó là để hở
			// ba đường mà chỉ vá một. Bước không khai `doc_duoc` thì LocDocDuoc
			// trả về nguyên map cũ — không đổi một byte nào.
			outs = LocDocDuoc(s, outs)
			// Biến artifact dựng ở ĐÂY, cùng chỗ và cùng lúc với `outs`: cả hai
			// đều là "bước này được thấy gì của bước khác", và tách ra hai chỗ
			// thì `doc_duoc` chặn được một đường mà hở đường kia. MoiTruongArtifact
			// tự lọc theo doc_duoc, xem artifact.go.
			arts := moiTruongThem(runID, f, s, states, outs)
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
				state, msg, out := r.runForEach(waveCtx, runID, f, s, vars, outs, arts, items)
				st.set(s.ID, state, out)
				if state == store.StepFailed {
					if dung, ly := r.xuLyHong(ctx, runID, f, s, msg, vars, st); dung {
						mu.Lock()
						if stopAt == "" {
							stopAt, stopLy = s.ID, ly
						}
						mu.Unlock()
						cancelWave()
					}
				}
				return
			}

			state, msg, out := r.runStep(waveCtx, runID, f, s, vars, outs, arts)
			st.set(s.ID, state, out)

			if state == store.StepFailed {
				if dung, ly := r.xuLyHong(ctx, runID, f, s, msg, vars, st); dung {
					mu.Lock()
					if stopAt == "" {
						stopAt, stopLy = s.ID, ly
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
	vars map[string]string, outs, arts map[string]string, items []string) (state, msg, output string) {

	// `artifact` + `foreach` là một cái bẫy: mọi lượt lặp chạy song song trong
	// CÙNG một thư mục artifact và ghi đè lên nhau, rồi `{{artifacts.<tên>}}` chỉ
	// trỏ được tới một file — tức là bước sau đọc kết quả của một mục ngẫu nhiên
	// và tưởng đó là kết quả của cả bước. Validate đã chặn ở lúc lưu; chặn thêm ở
	// đây vì Flow còn dựng được thẳng bằng mã Go và bằng file chưa qua `validate`.
	// Cả `idempotent` cũng chưa dùng chung với `foreach` được: khoá phải tính
	// trên TỪNG lượt lặp, còn một khoá chung cho cả bước sẽ bỏ qua luôn những mục
	// MỚI xuất hiện trong danh sách. Nói ra chứ không im lặng bỏ qua cái cờ —
	// một cờ bị lờ đi trong im lặng là một tính năng người dùng tưởng đang bật.
	var lyDoChan string
	switch {
	case len(s.Artifact) > 0:
		lyDoChan = "không dùng `artifact` chung với `foreach` — các lượt lặp chạy song song " +
			"trong cùng một thư mục và sẽ ghi đè lên nhau"
	case s.Idempotent:
		lyDoChan = "chưa dùng `idempotent` chung với `foreach` được — một khoá chung cho cả bước " +
			"sẽ bỏ qua cả những mục MỚI trong danh sách"
	case s.Type == TypeMerge:
		lyDoChan = "không dùng `foreach` với `merge` — mỗi lượt lặp sẽ gộp lại đúng cùng một tập nguồn"
	case s.Type == TypeRoute:
		lyDoChan = "không dùng `foreach` với `route` — mỗi lượt lặp sẽ chọn lại đúng cùng một tập đường"
	}
	if lyDoChan != "" {
		_ = r.DB.SetStep(runID, s.ID, store.StepFailed, lyDoChan, 1)
		r.baoBuocHong(runID, f, s, lyDoChan)
		return store.StepFailed, lyDoChan, ""
	}

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
			for k, v := range arts {
				env[k] = v
			}

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

// moiTruongThem gom MỌI biến thuộc loại "bước này được thấy gì của bước khác"
// vào đúng một chỗ: đường dẫn artifact, và khối chữ đã gộp của bước `merge`.
//
// Một chỗ chứ không hai, vì cả hai đều đi qua cùng một bộ lọc quyền đọc
// (`doc_duoc`) và đều cần TRẠNG THÁI các bước nguồn — thứ mà do() không có.
// Tách ra hai chỗ thì `doc_duoc` chặn được một đường mà hở đường kia; đó đúng
// là lý do biến artifact đã được dựng cạnh `outs` ngay từ đầu.
func moiTruongThem(runID int64, f Flow, s Step, states, outs map[string]string) map[string]string {
	m := MoiTruongArtifact(runID, f, s, states)
	gop := MoiTruongGop(s, states, outs)
	if len(gop) == 0 {
		return m
	}
	if m == nil {
		m = make(map[string]string, len(gop))
	}
	for k, v := range gop {
		m[k] = v
	}
	return m
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
	vars map[string]string, outputs, arts map[string]string) (state, msg, output string) {
	// Bước sau dùng được kết quả bước trước.
	env := WithOutputs(vars, outputs)
	// …và FILE bước trước để lại. WithOutputs trả về map mới nên ghi thẳng vào
	// đây không đụng gì tới `vars` của lượt chạy.
	for k, v := range arts {
		env[k] = v
	}
	// IDEMPOTENCY: trước khi tốn một đồng nào, hỏi sổ xem việc CHÍNH XÁC NÀY đã
	// có lượt chạy nào làm xong chưa. Khoá tính từ env ĐÃ THAY BIẾN ở trên, nên
	// nó cuốn theo cả kết quả các bước trước — xem idempotent.go.
	//
	// Bước không bật `idempotent` thì KhoaIdem trả rỗng, và cả khối này không
	// chạm vào sổ một lần nào.
	khoa := KhoaIdem(s, env)
	if khoa != "" {
		if state, out, xong := r.thuDungLaiViecCu(runID, f, s, env, khoa); xong {
			return state, "", out
		}
	}

	tries := s.Retry + 1
	if tries < 1 {
		tries = 1
	}
	var lastErr error
	var kqCuoi KetQuaAgent // giữ kết quả lần thử cuối, kể cả khi nó hỏng
	for attempt := 1; attempt <= tries; attempt++ {
		// Dọn thư mục artifact TRƯỚC MỖI LẦN THỬ, không phải một lần trước vòng
		// lặp. Lần thử 1 ghi được file rồi mới hỏng ở đoạn sau; nếu để file đó
		// nằm lại thì lần thử 2 hỏng vẫn "đủ artifact" và bước được ghi là xong.
		// Retry không được phép biến một bước hỏng thành một bước xong.
		if _, err := ChuanBiArtifact(runID, s); err != nil {
			_ = r.DB.SetStep(runID, s.ID, store.StepFailed, err.Error(), attempt)
			r.baoBuocHong(runID, f, s, err.Error())
			return store.StepFailed, err.Error(), ""
		}
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

		// HỢP ĐỒNG ĐẦU RA. Agent chạy xong, thoát mã 0, không lỗi nào — nhưng nếu
		// nó không GIAO RA thứ được giao thì bước này chưa làm xong.
		//
		// Kiểm ở ĐÂY, ngay trước khi ghi StepDone, chứ không phải ở chỗ đọc báo
		// cáo: ghi done rồi mới nói "à nhưng mà" thì mọi thứ đọc sổ sau đó đều đã
		// tin nhầm — kể cả bước sau đang chờ nó.
		if err == nil {
			if thieu := ThieuPhaiCo(s, kq.Output); thieu != "" {
				err = fmt.Errorf("%s", thieu)
			}
		}

		// HỢP ĐỒNG ARTIFACT — cùng chỗ, cùng lý do với `phai_co` ngay trên.
		//
		// Khai `artifact` là hứa để lại một file. Không kiểm ở đây thì bước được
		// ghi `done`, bước sau nhận một đường dẫn hợp lệ trỏ vào hư không, và nó
		// hỏng bằng "no such file" — một thông báo chỉ vào SAI BƯỚC. Người đọc
		// sổ sẽ đi tìm lỗi ở bước tiêu thụ trong khi thủ phạm là bước sản xuất.
		if err == nil {
			if thieu := ThieuArtifact(runID, s); thieu != "" {
				err = fmt.Errorf("%s", thieu)
			}
		}

		if err == nil {
			_ = r.DB.SetStep(runID, s.ID, store.StepDone, "", attempt)
			// Ghi khoá SAU KHI xong, không phải trước: khoá trong sổ nghĩa là
			// "việc này đã làm XONG". Ghi trước là hứa trước khi làm, và bước
			// hỏng giữa chừng sẽ khiến lượt sau bỏ qua một việc chưa ai làm.
			if khoa != "" {
				_ = r.DB.SetStepIdemKey(runID, s.ID, khoa)
			}
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
		// CỔNG TRẦN ĐỒNG THỜI — bốn chiều, và nó CHỜ chứ không từ chối.
		//
		// Đặt ở đây chứ không ở runWave là cố ý: `foreach` cũng chạy song song
		// (runForEach có semaphore riêng) và cũng đi qua đúng hàm này. Canh ở
		// runWave thì bịt được một đường mà hở đường kia — đúng lớp lỗi cả ngày
		// hôm nay đi sửa.
		the, err := r.xinCho(ctx, s, n)
		if err != nil {
			return KetQuaAgent{}, err
		}
		// Giữ chỗ tới khi RunAgents TRẢ VỀ, tức tới khi phiên đã rời sổ (cầu
		// agentBridge đợi waitSessions xong mới về). Trả sớm hơn là mời bước sau
		// bật phiên trong lúc phiên này còn sống.
		defer the.Tra()
		if the != nil && the.Cap > 0 {
			n = the.Cap
		}
		return r.Agent.RunAgents(ctx, s.Profile, s.Model, ExpandChay(s.Prompt, vars), n, s.Worktree, s.TuDuyetQuyen)

	// Node `model`: gọi THẲNG model API. Khai từ đầu dự án với ghi chú "chờ
	// đường API (Pha 1/4)" — đường đó nay đã có, đã đo thật, và có cả bộ chuyển
	// route dự phòng.
	//
	// Vì sao cần: người soi của doi-4 chạy bằng CLI grok, mà CLI đó vừa hỏng
	// vĩnh viễn (HTTP 410 "Live search is deprecated"). Cùng nhà cung cấp đó qua
	// đường API thì vẫn trả lời bình thường — đã đo. Không có node này thì mọi
	// lượt chạy đều mất người soi chỉ vì một cái CLI đổi API.
	case TypeModel:
		if r.Model == nil {
			return KetQuaAgent{}, fmt.Errorf("node `model` cần đường AI API nhưng chưa được cắm " +
				"(xem internal/api: Runner.Model)")
		}
		// `route` đi qua Expand chứ KHÔNG phải ExpandChay, và thiếu thì dừng
		// ngay — cùng luật với tham số của bước shell, vì cùng lớp nguy hiểm.
		// Chốt placeholder ở đây sẽ cho ra tên route là cả một câu tiếng Việt,
		// rồi lỗi hiện ra là "không có route này" kèm nguyên câu đó: một thông
		// báo chỉ vào sai chỗ hoàn toàn.
		if id := BuocConSot(s.Route, vars); id != "" {
			return KetQuaAgent{}, fmt.Errorf(
				"route cần kết quả của bước %q nhưng bước đó không để lại gì — bước này không biết đi đường nào", id)
		}
		return r.Model.GoiModel(ctx, strings.TrimSpace(Expand(s.Route, vars)), ExpandChay(s.Prompt, vars))

	// Node `route`: CHỌN đường rồi chuyền tên cho bước sau. Xem route.go.
	case TypeRoute:
		if r.Route == nil {
			return KetQuaAgent{}, fmt.Errorf("node `route` cần đường AI API nhưng chưa được cắm " +
				"(xem internal/api: Runner.Route)")
		}
		ung := make([]string, 0, len(s.Routes))
		for _, x := range s.Routes {
			if t := strings.TrimSpace(Expand(x, vars)); t != "" {
				ung = append(ung, t)
			}
		}
		kq, err := r.Route.ChonRoute(ctx, ung)
		// Nhật ký in ra CẢ KHI hỏng: lúc không đường nào sống thì lý do từng
		// đường chết mới là thứ người đọc cần, chứ không phải một câu tổng kết.
		for _, d := range kq.NhatKy {
			r.Bus.Infof("%s.%s: %s", s.ID, "route", d)
		}
		if err != nil {
			return KetQuaAgent{}, err
		}
		ten := strings.TrimSpace(kq.Ten)
		if ten == "" {
			return KetQuaAgent{}, fmt.Errorf("phần cắm route trả về một cái tên RỖNG mà không báo lỗi — " +
				"bước sau sẽ đi đường mặc định thay vì đường được chọn, nên dừng ở đây")
		}
		// Output là ĐÚNG cái tên, không gì khác — nó sẽ đi thẳng vào
		// `route = "{{steps.x.output}}"` của bước sau.
		return KetQuaAgent{Output: ten}, nil

	// Node `merge`: gộp đầu ra của các bước trong `needs`. Xem merge.go.
	//
	// Phần chữ đã được dựng sẵn ở moiTruongThem, cùng chỗ và cùng lúc với biến
	// artifact — vì nó cần TRẠNG THÁI các bước nguồn, mà do() không có.
	case TypeMerge:
		gop, co := vars[KhoaGopDauRa]
		if !co {
			return KetQuaAgent{}, fmt.Errorf("bước merge %q chạy mà bộ chạy chưa dựng phần chữ đã gộp "+
				"— đây là lỗi của chỗ dựng Runner, không phải của flow", s.ID)
		}
		return KetQuaAgent{Output: gop}, nil

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
			// Cùng luật cho artifact, và ở đây còn cần hơn: một placeholder
			// artifact chưa thay sẽ được truyền vào lệnh như một TÊN FILE, và
			// `no such file` là thông báo dẫn người đọc đi sai hướng.
			if ten := ArtifactConSot(a, vars); ten != "" {
				return KetQuaAgent{}, fmt.Errorf(
					"tham số %d cần artifact %q nhưng bước sản xuất nó chưa để lại file nào", i+1, ten)
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

	case TypePlugin:
		if r.Plugin == nil {
			return KetQuaAgent{}, fmt.Errorf("bước %s gọi plugin %q nhưng bộ chạy plugin chưa được cắm "+
				"— đây là lỗi của phần dựng Runner, không phải của flow", s.ID, s.Plugin)
		}
		// ExpandChay chứ không phải Expand: {{steps.x.output}} của một bước không
		// để lại gì phải thành một câu NÓI RÕ là thiếu, chứ không phải chuỗi thô
		// lọt nguyên vào đầu vào của plugin rồi được đếm như dữ liệu thật.
		out, err := r.Plugin.GoiPlugin(ctx, s.Plugin, ExpandChay(s.Vao, vars), s.ThamSo)
		if err != nil {
			return KetQuaAgent{}, err
		}
		return KetQuaAgent{Output: out}, nil

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
	case TypePlugin:
		// Bước plugin cũng là tiếng nói của MÁY: ghi lại ĐÚNG thứ đã gửi đi, vì
		// đó là thứ duy nhất giải thích được vì sao plugin trả về cái nó trả về.
		return ExpandChay(s.Vao, vars)
	// Với `merge`, "câu hỏi" là ĐÚNG khối chữ nó gộp được. Không phải một câu
	// tóm tắt kiểu "gộp a, b, c": khối chữ ấy chính là toàn bộ đầu vào quyết
	// định kết quả bước, nên nó phải nằm trong khoá idempotency (KhoaIdem đọc
	// cauHoi). Tóm tắt thì nguồn đổi kết quả mà khoá không đổi, và lượt sau sẽ
	// dùng lại một khối gộp đã cũ.
	case TypeMerge:
		return vars[KhoaGopDauRa]
	// Với `route`, "câu hỏi" là tập đường được xét — thứ duy nhất quyết định
	// đường nào được chọn.
	case TypeRoute:
		return "chọn đường: " + MoTaRoute(s)
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

// thuDungLaiViecCu tra sổ xem việc này đã có lượt chạy trước làm xong chưa, và
// nếu có thì dựng lại kết quả cho lượt chạy NÀY.
//
// Trả về xong=false nghĩa là "cứ chạy thật" — cả khi không trúng, cả khi trúng
// nhưng không dựng lại được. KHÔNG có nhánh nào trả về done mà thiếu thứ gì:
// một bước `done` nửa vời là bước sau đọc phải một artifact rỗng và tưởng đó là
// kết quả thật.
func (r *Runner) thuDungLaiViecCu(runID int64, f Flow, s Step, env map[string]string,
	khoa string) (state, output string, xong bool) {

	cu, co, err := r.DB.TimBuocDaLam(khoa, runID)
	if err != nil || !co {
		return "", "", false
	}
	da := KetQuaCu{RunID: cu.RunID, StepID: cu.StepID, Output: cu.Output}

	// Artifact phải được CHÉP sang lượt này. Trúng cache mà file cũ đã bị dọn thì
	// coi như KHÔNG trúng — thà chạy lại tốn tiền còn hơn báo xong rồi để bước
	// sau mở một file không tồn tại.
	if err := chepArtifactCu(da, runID, s); err != nil {
		r.Bus.Warnf("%s.%s: lượt #%d đã làm việc này nhưng %v — chạy lại", f.Name, s.ID, cu.RunID, err)
		return "", "", false
	}

	_ = r.DB.SetStep(runID, s.ID, store.StepDone, MoTaIdem(da), 0)
	_ = r.DB.SetStepPrompt(runID, s.ID, cauHoi(s, env))
	_ = r.DB.SetStepIdemKey(runID, s.ID, khoa)
	if da.Output != "" {
		_ = r.DB.SetStepOutput(runID, s.ID, da.Output)
	}
	// CỐ Ý KHÔNG chép chi phí của lượt cũ sang. Lượt này không tiêu một token
	// nào, và ghi lại con số cũ sẽ làm bảng cộng dồn theo ngày đếm cùng một
	// khoản hai lần — đúng cái sổ chi phí sinh ra để chống.
	r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + s.ID,
		SessionID: runID, Msg: MoTaIdem(da),
		Detail: map[string]string{
			"run": fmt.Sprint(runID), "step": s.ID,
			"state": store.StepDone, "type": s.Type,
			// Nói rõ ĐÃ MƯỢN CỦA AI. Không có khoá này thì bảng hiện một bước
			// `done` không tốn gì và không ai lần ngược được về việc thật.
			"idem_tu_run": fmt.Sprint(cu.RunID),
		}})
	return store.StepDone, da.Output, true
}

// xuLyHong quyết định một bước hỏng có làm CẢ LƯỢT dừng lại không, và chạy bước
// GỠ LẠI khi được yêu cầu.
//
// MỘT CHỖ DUY NHẤT cho cả nhánh thường lẫn nhánh `foreach`. Trước đây hai nhánh
// tự xét `OnFailure` riêng, và chúng đã lệch nhau thật: nhánh foreach so bằng
// với hai giá trị cụ thể, nên thêm giá trị thứ tư vào là nó lặng lẽ rơi vào
// nhánh "dừng" mà không ai gỡ gì cả.
//
// ctx ở đây là ctx của CẢ LƯỢT CHẠY, không phải waveCtx: một bước cùng đợt hỏng
// và huỷ đợt thì KHÔNG được giết bước gỡ lại đang chạy dở. Một cái undo bị cắt
// ngang để lại hiện trường tệ hơn cả không undo — nửa gỡ thì không ai biết đang
// ở đâu nữa. (Người dùng huỷ cả lượt thì vẫn dừng: ctx đó là ctx này.)
func (r *Runner) xuLyHong(ctx context.Context, runID int64, f Flow, s Step, msg string,
	vars map[string]string, st *runState) (dungLuot bool, ly string) {

	switch s.OnFailure {
	case OnFailContinue:
		r.Bus.Warnf("%s.%s hỏng nhưng on_failure=continue — đi tiếp", f.Name, s.ID)
		return false, ""

	case OnFailFallback:
		xong, lyThay := r.chayThayThe(ctx, runID, f, s, vars, st)
		if xong {
			// Khác `compensate` ở ĐÚNG chỗ này: bước thay thế làm xong việc thì
			// lượt chạy ĐI TIẾP. `compensate` gỡ lại rồi dừng vì việc chính coi
			// như không làm; `fallback` thì việc chính có người làm thay.
			return false, ""
		}
		return true, fmt.Sprintf("%s — VÀ BƯỚC CHẠY THAY %q CŨNG HỎNG (%s). "+
			"Không còn đường nào khác cho bước này: dừng.", msg, s.Fallback, lyThay)

	case OnFailCompensate:
		xong, lyGo := r.chayGoLai(ctx, runID, f, s, vars, st)
		if xong {
			// Gỡ được rồi thì VẪN DỪNG. `compensate` là "gỡ rồi dừng", không
			// phải "gỡ rồi đi tiếp": chạy tiếp trên nền một việc vừa bị gỡ là
			// chạy tiếp trên nền không có gì.
			return true, fmt.Sprintf("%s — đã chạy bước gỡ lại %q", msg, s.Compensate)
		}
		return true, fmt.Sprintf("%s — VÀ BƯỚC GỠ LẠI %q CŨNG HỎNG (%s). "+
			"Việc chính không xong mà cũng chưa gỡ được: cần người vào xem tay.",
			msg, s.Compensate, lyGo)

	default:
		return true, msg
	}
}

// chayGoLai chạy bước gỡ lại của một bước vừa hỏng.
//
// Bước gỡ lại chạy qua ĐÚNG runStep như mọi bước khác — nó có timeout, có retry,
// có hợp đồng `phai_co`, có artifact, và ghi vào sổ y hệt. Cái nó KHÔNG có là
// `on_failure`: runStep không xét trường đó (việc đó là của runWave), nên không
// có đường nào để gỡ-lại-của-gỡ-lại xảy ra. Đó là một tính chất của chỗ cắm, chứ
// không phải một cái cờ ai cũng tắt được.
func (r *Runner) chayGoLai(ctx context.Context, runID int64, f Flow, hong Step,
	vars map[string]string, st *runState) (xong bool, ly string) {

	g, co := TimBuoc(f, hong.Compensate)
	if !co {
		ly = fmt.Sprintf("compensate trỏ tới bước %q không tồn tại", hong.Compensate)
		r.Bus.Failuref("%s.%s: %s", f.Name, hong.ID, ly)
		return false, ly
	}
	if g.ID == hong.ID {
		ly := "bước không thể tự gỡ lại chính nó"
		r.Bus.Failuref("%s.%s: %s", f.Name, hong.ID, ly)
		return false, ly
	}

	r.Bus.Warnf("%s.%s hỏng — chạy bước gỡ lại %s", f.Name, hong.ID, g.ID)

	states, outs := st.snapshot()
	outs = LocDocDuoc(g, outs)
	arts := moiTruongThem(runID, f, g, states, outs)
	// Bước gỡ lại phải biết mình đang gỡ CÁI GÌ. Không có biến này thì một bước
	// gỡ dùng chung cho ba bước không có cách nào phân biệt, và người viết flow
	// phải chép ra ba bước gỡ gần như giống hệt nhau.
	bien := make(map[string]string, len(vars)+1)
	for k, v := range vars {
		bien[k] = v
	}
	bien[KhoaBuocHong] = hong.ID

	state, msg, out := r.runStep(ctx, runID, f, g, bien, outs, arts)
	st.set(g.ID, state, out)
	if state != store.StepDone {
		return false, msg
	}
	r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + g.ID,
		SessionID: runID, Msg: fmt.Sprintf("đã gỡ lại việc của bước %s", hong.ID),
		Detail: map[string]string{
			"run": fmt.Sprint(runID), "step": g.ID,
			"state": store.StepDone, "type": g.Type,
			// Khoá này để mặt web nối được mũi tên "gỡ cho bước nào" mà không
			// phải tách chuỗi trong câu Msg.
			"go_lai_cho": hong.ID,
		}})
	return true, ""
}

// chayThayThe chạy bước CHẠY THAY của một bước vừa hỏng.
//
// Song song với chayGoLai và cố ý giống nó: cùng đường vào (`runStep`), cùng bộ
// lọc quyền đọc, cùng biến `{{buoc_hong}}`, cùng lý do dùng ctx CỦA CẢ LƯỢT chứ
// không phải waveCtx (một bước cùng đợt hỏng và huỷ đợt thì không được giết bước
// đang chạy thay — nó sẽ bị ghi là "chạy thay cũng hỏng", một câu sai sự thật).
//
// Khác ở phần đuôi: gỡ lại xong thì dừng, chạy thay xong thì ĐI TIẾP — và để đi
// tiếp được thì kết quả phải nằm đúng chỗ bước sau đang tìm. Xem fallback.go.
//
// Không có đường nào để thay-thế-của-thay-thế: runStep không xét `on_failure`
// (việc đó của runWave). Tính chất của chỗ cắm, không phải một cái cờ.
func (r *Runner) chayThayThe(ctx context.Context, runID int64, f Flow, hong Step,
	vars map[string]string, st *runState) (xong bool, ly string) {

	g, co := TimBuoc(f, hong.Fallback)
	if !co {
		ly = fmt.Sprintf("fallback trỏ tới bước %q không tồn tại", hong.Fallback)
		r.Bus.Failuref("%s.%s: %s", f.Name, hong.ID, ly)
		return false, ly
	}
	if g.ID == hong.ID {
		ly := "bước không thể tự chạy thay chính nó"
		r.Bus.Failuref("%s.%s: %s", f.Name, hong.ID, ly)
		return false, ly
	}

	r.Bus.Warnf("%s.%s hỏng — chạy bước %s thay", f.Name, hong.ID, g.ID)

	states, outs := st.snapshot()
	outs = LocDocDuoc(g, outs)
	arts := moiTruongThem(runID, f, g, states, outs)
	// Một bước chạy thay dùng chung cho ba bước phải biết mình đang thay cho ai.
	// CÙNG một biến với bước gỡ lại, không đặt tên thứ hai: câu hỏi giống hệt
	// nhau ("bước nào vừa hỏng"), và hai cái tên cho một thứ là hai cái phải nhớ.
	bien := make(map[string]string, len(vars)+1)
	for k, v := range vars {
		bien[k] = v
	}
	bien[KhoaBuocHong] = hong.ID

	state, msg, out := r.runStep(ctx, runID, f, g, bien, outs, arts)
	st.set(g.ID, state, out)
	if state != store.StepDone {
		return false, msg
	}
	// Kết quả của bước chạy thay đọc được BẰNG TÊN BƯỚC HỎNG — không có dòng này
	// thì bước sau nhận một ô rỗng và `fallback` không dùng được vào việc gì.
	ganKetQuaThayThe(f, st)
	r.Bus.Publish(events.Event{Type: events.FlowStep, Addr: f.Name + "." + g.ID,
		SessionID: runID, Msg: fmt.Sprintf("đã chạy thay cho bước %s", hong.ID),
		Detail: map[string]string{
			"run": fmt.Sprint(runID), "step": g.ID,
			"state": store.StepDone, "type": g.Type,
			// Cùng lý do với `go_lai_cho`: mặt web nối mũi tên bằng khoá có cấu
			// trúc, không bằng cách tách chuỗi trong câu Msg.
			"chay_thay_cho": hong.ID,
		}})
	return true, ""
}
