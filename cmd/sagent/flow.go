package main

import (
	"fmt"
	"os"
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
		flowRuns()
	case "approve", "duyet":
		flowDecide(args[1:], true)
	case "reject", "tu-choi":
		flowDecide(args[1:], false)
	case "resume", "tiep":
		if len(args) < 2 {
			fail(fmt.Errorf("thiếu số lần chạy. Ví dụ: sagent flow resume 3"))
		}
		flowResume(args[1])
	default:
		fail(fmt.Errorf("không hiểu 'flow %s' — dùng: list | show | validate | run | runs | approve | reject | resume", sub))
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
	fmt.Println("  Thứ tự chạy:")
	for i, s := range order {
		dep := ""
		if len(s.Needs) > 0 {
			dep = "  ← " + strings.Join(s.Needs, ", ")
		}
		fmt.Printf("   %d. %-10s [%s]%s\n", i+1, s.ID, s.Type, dep)
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
			fmt.Printf("      %d agent%s\n      prompt: %s\n", n, wt, truncate(flow.Expand(s.Prompt, f.Vars), 70))
		case flow.TypeShell, flow.TypeTest, flow.TypeLint:
			fmt.Printf("      chạy: %s\n", strings.Join(s.Run, " "))
		case flow.TypeApprove, flow.TypeNotify:
			fmt.Printf("      %s\n", truncate(s.Message, 70))
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

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}
