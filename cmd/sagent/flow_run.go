package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// varFlags rút các cờ `--var ten=giatri`, trả về phần còn lại.
func varFlags(args []string) (map[string]string, []string) {
	vars := map[string]string{}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--var" && i+1 < len(args) {
			if k, v, ok := strings.Cut(args[i+1], "="); ok {
				vars[k] = v
			}
			i++
			continue
		}
		out = append(out, args[i])
	}
	return vars, out
}

// yChay là những gì người dùng gõ sau `sagent flow run <tên>`.
type yChay struct {
	vars   map[string]string
	prof   string
	cuChay bool // biết tài khoản hỏng mà vẫn muốn chạy — xem API.KiemTaiKhoanFlow
	kho    bool // chỉ hỏi "chạy thì sẽ ra sao", không chạy — xem API.FlowChayKho
}

// docCoChay rút các cờ. Tách ra khỏi flowRun để TEST ĐƯỢC: flowRun tự mở sổ và
// tự thoát tiến trình khi lỗi, nên không test thẳng được, mà cờ --kho lại đúng
// là thứ quyết định lượt chạy CÓ XẢY RA THẬT hay không.
func docCoChay(args []string) yChay {
	vars, args := varFlags(args)
	prof, args := strFlag(args, "--profile", "")
	cuChay, args := boolFlag(args, "--cu-chay")
	kho, _ := boolFlag(args, "--kho")
	return yChay{vars: vars, prof: prof, cuChay: cuChay, kho: kho}
}

func flowRun(name string, args []string) {
	y := docCoChay(args)
	if y.kho {
		flowChayKho(name, y.vars, y.prof)
		return
	}

	a, done := open()
	defer done()
	wd, _ := os.Getwd()
	res, err := a.FlowRunCuChay(context.Background(), wd, name, y.vars, api.ParseAddr(y.prof), y.cuChay)
	if err != nil {
		fail(err)
	}
	done()
	reportRun(res)
}

// flowChayKho in KẾ HOẠCH của một lượt chạy rồi dừng — không bật agent nào,
// không ghi dòng nào vào sổ.
//
// Vì sao đáng có: ba lượt chạy thật ngày 19/08 (#30, #32, #33) được bấm chỉ để
// xem cổng kiểm tài khoản nói gì. Mỗi lượt đốt hạn mức thuê bao và để lại một
// lượt rác phải huỷ tay. Câu trả lời vốn đã biết được trước khi chạy.
func flowChayKho(name string, vars map[string]string, prof string) {
	a, done := open()
	defer done()
	wd, _ := os.Getwd()
	kh, err := a.FlowChayKho(wd, name, vars, api.ParseAddr(prof))
	if err != nil {
		fail(err)
	}
	done()

	fmt.Printf("\n  CHẠY KHAN — %s", kh.Flow)
	if kh.Desc != "" {
		fmt.Printf(" · %s", kh.Desc)
	}
	fmt.Printf("\n  Thư mục: %s\n", kh.Dir)
	if len(kh.Vars) > 0 {
		fmt.Println()
		fmt.Println("  Biến dùng cho lượt này:")
		for _, k := range tenBienSapXep(kh.Vars) {
			fmt.Printf("    %-12s %s\n", k, truncate(kh.Vars[k], 60))
		}
	}

	for _, d := range kh.Dot {
		fmt.Println()
		switch {
		case d.ChoDuyet:
			fmt.Printf("  Đợt %d — RÀO DUYỆT (lượt chạy thật DỪNG ở đây tới khi có người duyệt)\n", d.So)
		case len(d.Buoc) > 1:
			fmt.Printf("  Đợt %d — %d bước chạy SONG SONG\n", d.So, len(d.Buoc))
		default:
			fmt.Printf("  Đợt %d\n", d.So)
		}
		for _, b := range d.Buoc {
			dong := fmt.Sprintf("   · %-12s [%s]", b.ID, b.Type)
			if b.TaiKhoan != "" {
				dong += "  " + b.TaiKhoan
			} else if b.Type == flow.TypeAgent || b.Type == flow.TypeReview {
				// Nói thẳng là CHƯA BIẾT thay vì im lặng: bước này sẽ hỏng ngay
				// lúc chạy thật, và đây là chỗ duy nhất báo được trước.
				dong += "  (chưa biết tài khoản — thiếu `profile` và --profile)"
			}
			// Model đứng ngay sau tài khoản: đọc kế hoạch chạy khan là lúc người
			// ta cân "bước này có đáng model đắt không", nên hai thứ phải nằm cạnh.
			if b.Model != "" {
				dong += " · model " + b.Model
			}
			// Vai trò đứng cạnh tài khoản và model: ba thứ này cùng trả lời một
			// câu hỏi — "ai làm bước này, với tư cách gì, bằng model nào". Bước
			// chưa phân vai thì im lặng, KHÔNG in "vai ?" hay đoán hộ.
			if b.VaiTro != "" {
				dong += " · vai " + b.VaiTro
			}
			if b.SoAgent > 0 {
				dong += fmt.Sprintf(" · %d agent", b.SoAgent)
			}
			if b.Worktree {
				dong += " · worktree riêng"
			}
			if b.TuDuyetQuyen {
				dong += " · ⚠ TỰ DUYỆT MỌI QUYỀN"
			}
			fmt.Println(dong)
			// Quyền đọc in cho MỌI bước, kể cả bước chưa khai `doc_duoc`. In
			// riêng bước có khai thì im lặng lại thành "mặc định là gì" — mà
			// mặc định ở đây là MỞ HẾT, đúng thứ người đọc kế hoạch cần biết
			// trước khi tiêu tiền: bước này sẽ nuốt output của những ai.
			fmt.Printf("       đọc được: %s\n", b.DocDuoc)
			if b.Lap != "" {
				fmt.Printf("       lặp trên %s — mỗi mục thêm một lượt agent, dài bao nhiêu thì lúc chạy mới biết\n", b.Lap)
			}
			if b.Prompt != "" {
				fmt.Printf("       %s\n", truncate(b.Prompt, 100))
			}
			if b.ConSot != "" {
				fmt.Printf("       ✗ đang chờ kết quả bước %q, nhưng bước đó KHÔNG xong trước nó\n", b.ConSot)
			}
		}
	}

	fmt.Println()
	if kh.CoLap {
		fmt.Printf("  Tổng: TỪ %d phiên agent trở lên (có bước lặp)\n", kh.SoAgent)
	} else {
		fmt.Printf("  Tổng: %d phiên agent\n", kh.SoAgent)
	}
	for _, v := range kh.Van {
		mark := "✗"
		if v.Warn {
			mark = "!"
		}
		if v.Buoc != "" {
			fmt.Printf("  %s %s: %s\n", mark, v.Buoc, v.Msg)
		} else {
			fmt.Printf("  %s %s\n", mark, v.Msg)
		}
	}
	for _, h := range kh.TaiKhoanHong {
		fmt.Printf("  ✗ %s — %s (kéo theo bước: %s)\n", h.Addr, h.LyDo, strings.Join(h.Buoc, ", "))
		fmt.Printf("      sửa: sagent %s\n", h.Addr)
	}

	fmt.Println()
	fmt.Println("  Chưa có gì xảy ra: không phiên agent nào được bật, không dòng nào vào sổ.")
	fmt.Printf("  Chạy thật: sagent flow run %s", kh.Flow)
	if prof != "" {
		fmt.Printf(" --profile %s", prof)
	}
	fmt.Println()
	fmt.Println()
}

// tenBienSapXep để thứ tự in ra ổn định — map của Go trả về ngẫu nhiên, và một
// bản kế hoạch mỗi lần in một khác thì không so được với lần trước.
func tenBienSapXep(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func flowResume(idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		fail(fmt.Errorf("số lần chạy phải là số, được %q", idStr))
	}
	a, done := open()
	defer done()
	res, err := a.FlowResume(context.Background(), id, api.Addr{})
	if err != nil {
		fail(err)
	}
	done()
	reportRun(res)
}

// flowDecide duyệt (ok=true) hoặc từ chối (ok=false) một bước đang chờ.
func flowDecide(args []string, ok bool) {
	verb := "approve"
	if !ok {
		verb = "reject"
	}
	if len(args) < 2 {
		fail(fmt.Errorf("thiếu tham số. Ví dụ: sagent flow %s 3 duyet", verb))
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fail(fmt.Errorf("số lần chạy phải là số, được %q", args[0]))
	}
	a, done := open()
	defer done()
	res, err := a.FlowApprove(context.Background(), id, args[1], whoAmI(), ok, api.Addr{})
	if err != nil {
		fail(err)
	}
	done()
	reportRun(res)
}

// whoAmI ghi lại AI đã duyệt — để sau này còn truy được trách nhiệm.
func whoAmI() string {
	for _, k := range []string{"USERNAME", "USER"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return "người dùng"
}

func reportRun(res flow.Result) {
	fmt.Println()
	switch res.State {
	case "waiting_approval":
		fmt.Printf("  ⏸  Lần chạy #%d đang CHỜ DUYỆT ở bước %q\n", res.RunID, res.Waiting)
		fmt.Printf("     Duyệt:   sagent flow approve %d %s\n", res.RunID, res.Waiting)
		fmt.Printf("     Từ chối: sagent flow reject %d %s\n", res.RunID, res.Waiting)
	case "completed":
		fmt.Printf("  ✓ Lần chạy #%d đã xong.\n", res.RunID)
	case "failed":
		fmt.Printf("  ✗ Lần chạy #%d hỏng. Xem lại: sagent flow runs\n", res.RunID)
	case "cancelled":
		fmt.Printf("  ⃠ Lần chạy #%d đã huỷ.\n", res.RunID)
	default:
		fmt.Printf("  Lần chạy #%d: %s\n", res.RunID, res.State)
	}
	fmt.Println()
}

func flowRuns() {
	a, done := open()
	defer done()
	runs, err := a.FlowRuns(15)
	if err != nil {
		fail(err)
	}
	done()

	fmt.Println()
	if len(runs) == 0 {
		fmt.Println("  Chưa có lần chạy nào.")
		fmt.Println("  Thử: sagent flow run fanout --profile claude:<tên>")
		fmt.Println()
		return
	}
	fmt.Println("  Lịch sử chạy flow")
	fmt.Println()
	marks := map[string]string{
		"completed": "✓", "failed": "✗", "waiting_approval": "⏸",
		"cancelled": "⃠", "running": "…",
	}
	for _, r := range runs {
		mark := marks[r.State]
		if mark == "" {
			mark = " "
		}
		fmt.Printf("   %s #%-4d %-10s %-18s %s\n",
			mark, r.ID, r.Flow, r.State, r.Started.Format("02/01 15:04"))
	}
	fmt.Println()
	fmt.Println("  Chờ duyệt thì: sagent flow approve <#> <bước>")
	fmt.Println()
}

// flowRunChiTiet in kết quả TỪNG BƯỚC của một lần chạy.
//
// Có `FlowRunDetail` trong lõi từ lâu nhưng KHÔNG mặt nào gọi — nên chạy xong
// một flow thì không có cách nào đọc được agent đã trả về cái gì. Đo được lần
// chạy #8: `flow runs 8` chỉ in lại danh sách vì flowRuns() không nhận tham số.
func flowRunChiTiet(arg string) {
	id, err := strconv.ParseInt(strings.TrimPrefix(arg, "#"), 10, 64)
	if err != nil {
		fail(fmt.Errorf("số lần chạy phải là số, không phải %q. Ví dụ: sagent flow runs 8", arg))
	}
	a, done := open()
	defer done()
	run, steps, def, err := a.FlowRunDetail(id)
	if err != nil {
		fail(err)
	}
	done()

	marks := map[string]string{
		"completed": "✓", "failed": "✗", "waiting_approval": "⏸",
		"cancelled": "⃠", "running": "…", "skipped": "–",
	}
	fmt.Println()
	fmt.Printf("  Lần chạy #%d — %s (%s)\n", run.ID, run.Flow, run.State)
	fmt.Printf("  Bắt đầu %s · thư mục %s\n", run.Started.Format("02/01 15:04:05"), run.Dir)
	fmt.Println()

	// Đi theo thứ tự ĐỊNH NGHĨA để đọc được mạch trên xuống dưới, rồi mới vét
	// những bước có trong DB mà flows.toml đã bỏ (flow sửa sau khi chạy).
	xong := map[string]bool{}
	var tongChiPhi float64
	var tongVao, tongRa int
	in := func(id string, s store.StepRun, def *flow.Step) {
		xong[id] = true
		mark := marks[s.State]
		if mark == "" {
			mark = " "
		}
		dong := fmt.Sprintf("   %s %s", mark, id)
		if def != nil && def.Profile != "" {
			dong += "  [" + def.Profile + "]"
		}
		if s.State == "" {
			dong += "  (chưa chạy)"
		} else {
			dong += "  " + s.State
		}
		if s.Attempt > 1 {
			dong += fmt.Sprintf(" · lần thử %d", s.Attempt)
		}
		// Chi phí đo được của bước — chỉ hiện khi provider cho biết (>0), đừng in
		// "0 USD" để khỏi hiểu nhầm là chạy không tốn gì.
		if s.CostUSD > 0 || s.TokensIn > 0 || s.TokensOut > 0 {
			dong += fmt.Sprintf(" · %.4f USD (%d→%d tok)", s.CostUSD, s.TokensIn, s.TokensOut)
			tongChiPhi += s.CostUSD
			tongVao += s.TokensIn
			tongRa += s.TokensOut
		}
		fmt.Println(dong)
		if s.Msg != "" {
			fmt.Printf("      ! %s\n", s.Msg)
		}
		out := strings.TrimRight(s.Output, "\n")
		if out == "" {
			if s.State == "completed" {
				fmt.Println("      (không có output — agent chạy xong nhưng không in gì)")
			}
		} else {
			for _, d := range strings.Split(out, "\n") {
				fmt.Printf("      │ %s\n", d)
			}
		}
		fmt.Println()
	}
	for i := range def.Steps {
		st := def.Steps[i]
		in(st.ID, steps[st.ID], &st)
	}
	for id, s := range steps {
		if !xong[id] {
			in(id, s, nil)
		}
	}
	if tongChiPhi > 0 || tongVao > 0 || tongRa > 0 {
		fmt.Printf("   ── cả lượt: %.4f USD · %d token vào / %d ra\n\n",
			tongChiPhi, tongVao, tongRa)
	}
}

// flowHuy đánh dấu một lượt chạy dở dang là đã huỷ. Không giết tiến trình nào —
// nhắc người dùng kiểm `sagent status` để họ biết mình mới làm cái nào.
func flowHuy(idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		fail(fmt.Errorf("số lần chạy phải là số, được %q", idStr))
	}
	a, done := open()
	defer done()
	if err := a.FlowCancel(id, whoAmI()); err != nil {
		fail(err)
	}
	fmt.Printf("  ✓ đã đánh dấu lượt chạy #%d là ĐÃ HUỶ.\n", id)
	fmt.Println("    Tiến trình thì KHÔNG bị đụng — kiểm bằng `sagent status`, dừng bằng `sagent stop all`.")
}
