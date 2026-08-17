package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
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

func flowRun(name string, args []string) {
	vars, args := varFlags(args)
	prof, _ := strFlag(args, "--profile", "")

	a, done := open()
	defer done()
	wd, _ := os.Getwd()
	res, err := a.FlowRun(context.Background(), wd, name, vars, api.ParseAddr(prof))
	if err != nil {
		fail(err)
	}
	done()
	reportRun(res)
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
