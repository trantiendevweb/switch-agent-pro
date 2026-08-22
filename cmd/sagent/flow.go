package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/console"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// cmdFlow gom các lệnh con về workflow: list · show · validate.
func cmdFlow(args []string) {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "", "list", "ds":
		flowList()
	case "show", "xem":
		if len(args) < 2 {
			fail(fmt.Errorf("thiếu tên flow. Ví dụ: sagent flow show fanout"))
		}
		flowShow(args[1])
	case "validate", "kiem":
		flowValidate()
	case "run", "chay":
		if len(args) < 2 {
			fail(fmt.Errorf("thiếu tên flow. Ví dụ: sagent flow run squad --profile claude:phu"))
		}
		flowRun(args[1], args[2:])
	case "runs", "lich-su":
		if len(args) > 1 {
			flowRunChiTiet(args[1])
		} else {
			flowRuns()
		}
	case "tom-tat", "tomtat":
		if len(args) < 2 {
			fail(fmt.Errorf("thiếu số lần chạy. Ví dụ: sagent flow tom-tat 38"))
		}
		flowTomTat(args[1])
	case "approve", "duyet":
		flowDecide(args[1:], true)
	case "reject", "tu-choi":
		flowDecide(args[1:], false)
	case "huy", "cancel":
		if len(args) < 2 {
			fail(fmt.Errorf("thiếu số lần chạy. Ví dụ: sagent flow huy 30"))
		}
		flowHuy(args[1])
	case "resume", "tiep":
		if len(args) < 2 {
			fail(fmt.Errorf("thiếu số lần chạy. Ví dụ: sagent flow resume 3"))
		}
		flowResume(args[1])
	default:
		fail(fmt.Errorf("không hiểu 'flow %s' — dùng: list | show | validate | run | runs | tom-tat | approve | reject | resume | huy", sub))
	}
}

func flowList() {
	a, done := open()
	defer done()
	wd, _ := os.Getwd()
	flows, srcs, err := a.FlowList(wd)
	if err != nil {
		fail(err)
	}
	done()
	fmt.Println()
	fmt.Println("  Workflow")
	fmt.Println()
	for _, n := range flow.Names(flows) {
		f := flows[n]
		fmt.Printf("   %-10s %-2d bước  %s\n", n, len(f.Steps), f.Desc)
	}
	fmt.Println()
	if len(srcs) == 0 {
		fmt.Println("  (chỉ có flow mẫu dựng sẵn — tạo flows.toml cạnh .sagent/project.toml để thêm)")
	} else {
		fmt.Println("  Đọc từ:")
		for _, s := range srcs {
			fmt.Println("    ·", s)
		}
	}
	fmt.Println()
	fmt.Println("  Xem chi tiết: sagent flow show <tên>")
	fmt.Println()
}

func flowShow(name string) {
	a, done := open()
	defer done()
	wd, _ := os.Getwd()
	f, order, err := a.FlowShow(wd, name)
	if err != nil {
		fail(err)
	}
	done()

	fmt.Printf("\n  %s — %s\n\n", f.Name, f.Desc)
	if len(f.Vars) > 0 {
		fmt.Println("  Biến (ghi đè bằng --var ten=giatri):")
		for k, v := range f.Vars {
			fmt.Printf("    %-10s %s\n", k, truncate(v, 60))
		}
		fmt.Println()
	}
	// Bước nào đang làm nhiệm vụ GỠ LẠI cho bước khác — tính một lần cho cả
	// flow, đúng cách bộ thực thi tính (flow.BuocGoLai).
	goLai := flow.BuocGoLai(f)

	fmt.Println("  Thứ tự chạy:")
	for i, s := range order {
		dep := ""
		if len(s.Needs) > 0 {
			dep = "  ← " + strings.Join(s.Needs, ", ")
		}
		fmt.Printf("   %d. %-10s [%s]%s\n", i+1, s.ID, s.Type, dep)
		// BƯỚC GỠ LẠI: nói ra NGAY dưới tên bước, trước cả prompt.
		//
		// Đây là dòng quan trọng nhất trong cả bảng này. Bước gỡ lại đứng trong
		// danh sách "Thứ tự chạy" y như mọi bước khác, nên người đọc đếm nó vào
		// kế hoạch — trong khi bộ thực thi LOẠI nó khỏi lịch chạy thường và chỉ
		// gọi khi có sự cố. Không có dòng này thì bảng đang nói sai.
		if cho := flow.MoTaChoGoLai(f, s.ID); goLai[s.ID] && cho != "" {
			fmt.Printf("      %s\n", cho)
		}
		switch s.Type {
		case flow.TypeAgent, flow.TypeReview:
			n := s.Copies
			if n < 1 {
				n = 1
			}
			wt := ""
			if s.Worktree {
				wt = " · worktree riêng"
			}
			// Cờ toàn quyền PHẢI hiện khi xem flow: người ta đọc `flow show` đúng lúc
			// quyết định có chạy hay không. Giấu ở đây thì phải mở flows.toml mới biết
			// bước nào được xoá file và chạy lệnh tuỳ ý.
			if s.TuDuyetQuyen {
				wt += " · ⚠ TỰ DUYỆT MỌI QUYỀN"
			}
			fmt.Printf("      %d agent%s\n      prompt: %s\n", n, wt, truncate(flow.Expand(s.Prompt, f.Vars), 70))
		case flow.TypeShell, flow.TypeTest, flow.TypeLint:
			fmt.Printf("      chạy: %s\n", strings.Join(s.Run, " "))
		case flow.TypeApprove, flow.TypeNotify:
			fmt.Printf("      %s\n", truncate(s.Message, 70))
		case flow.TypeMerge:
			// Thứ tự gộp LÀ thứ tự `needs`. In lại ở đây chứ không bắt người đọc
			// suy từ dòng "← a, b, c" phía trên: dòng đó nói phụ thuộc, còn đây
			// nói THỨ TỰ TRONG KẾT QUẢ — hai câu khác nhau, chỉ tình cờ cùng
			// một danh sách.
			fmt.Printf("      gộp đầu ra theo thứ tự: %s\n", strings.Join(s.Needs, " → "))
		case flow.TypeRoute:
			fmt.Printf("      chọn đường: %s\n", flow.MoTaRoute(s))
		}
		if s.Type == flow.TypeModel && s.Route != "" {
			fmt.Printf("      đường: %s\n", s.Route)
		}
		for _, d := range moTaBaTruong(s.Artifact, s.Idempotent, buocGoLaiCua(s)) {
			fmt.Printf("      %s\n", d)
		}
	}

	// Cảnh báo ngay ở đây, đừng để tới lúc chạy mới biết.
	if ps := flow.Validate(f); len(ps) > 0 {
		fmt.Println()
		for _, p := range ps {
			fmt.Println("  " + p.String())
		}
	}
	fmt.Println()
}

func flowValidate() {
	a, done := open()
	defer done()
	wd, _ := os.Getwd()
	ps, err := a.FlowValidate(wd)
	if err != nil {
		fail(err)
	}
	done()

	nErr := 0
	fmt.Println()
	for _, p := range ps {
		if !p.Warn {
			nErr++
		}
		fmt.Println("  " + p.String())
	}
	if len(ps) == 0 {
		fmt.Println("  ✓ mọi flow đều hợp lệ")
	}
	fmt.Println()
	if nErr > 0 {
		console.KhoiPhuc()
		os.Exit(1) // để dùng được trong CI
	}
}

// moTaBaTruong dựng dòng mô tả cho ba trường mà `flow show` từng KHÔNG hiện:
// `artifact`, `idempotent` và `compensate`.
//
// VÌ SAO ĐÚNG BA TRƯỜNG NÀY: cả ba đổi HÀNH VI của lượt chạy theo cách không
// suy ra được từ tên bước hay từ sơ đồ phụ thuộc.
//
//	artifact    bước để lại FILE cho bước sau — đường truyền thứ hai, không đi
//	            qua {{steps.x.output}} nên nhìn sơ đồ không thấy;
//	idempotent  bước có thể KHÔNG CHẠY lần này vì một lượt trước đã làm xong;
//	compensate  bước hỏng thì một bước khác chạy để GỠ LẠI, rồi cả lượt dừng.
//
// Người đọc `flow show` đang quyết định có bấm chạy hay không. Giấu ba thứ này
// đi thì họ quyết định trên một bản kế hoạch thiếu ba hành vi.
//
// Nhận ba GIÁ TRỊ chứ không nhận flow.Step: `sagent flow show` đọc từ Step, còn
// bảng chạy khan đọc từ api.BuocKho đi qua JSON. Hai mặt phải in ra ĐÚNG một
// câu chữ, và cách chắc chắn nhất là chúng gọi chung một hàm — chứ không phải
// hai hàm "giống nhau" viết ở hai file.
func moTaBaTruong(artifact map[string]string, idempotent bool, buocGoLai string) []string {
	var out []string
	if len(artifact) > 0 {
		ten := make([]string, 0, len(artifact))
		for k := range artifact {
			ten = append(ten, k)
		}
		sort.Strings(ten) // map của Go trả ra ngẫu nhiên; bảng phải in giống nhau mọi lần
		cap := make([]string, 0, len(ten))
		for _, k := range ten {
			cap = append(cap, k+" → "+artifact[k])
		}
		out = append(out, "để lại file (artifact): "+strings.Join(cap, ", "))
	}
	if idempotent {
		out = append(out, "idempotent: lượt trước đã làm xong đúng việc này thì bước NÀY BỊ BỎ QUA")
	}
	if buocGoLai != "" {
		out = append(out, "hỏng thì: chạy bước gỡ lại \""+buocGoLai+"\" rồi DỪNG cả lượt")
	}
	return out
}

// buocGoLaiCua trả về id bước gỡ lại của s, hoặc "" nếu s không dùng chính sách
// `compensate`. Tách ra vì `Compensate` chỉ có tác dụng khi đi kèm on_failure.
func buocGoLaiCua(s flow.Step) string {
	if s.OnFailure == flow.OnFailCompensate {
		return s.Compensate
	}
	return ""
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}
