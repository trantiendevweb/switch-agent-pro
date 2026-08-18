package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/acl"
	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
)

// `sagent api` — đường THỨ HAI: gọi thẳng AI API, không qua CLI agent.
//
//	sagent api key <id>            đặt API key (đọc từ stdin, KHÔNG hiện trên màn hình)
//	sagent api ds                  liệt kê route đã cấu hình
//	sagent api <route> "prompt"    gọi
func cmdAPI(args []string) {
	if len(args) == 0 {
		apiDs()
		return
	}
	switch args[0] {
	case "key":
		apiDatKey(rest(args))
	case "ds", "list":
		apiDs()
	default:
		apiGoi(args[0], rest(args))
	}
}

func apiDatKey(args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu tên: sagent api key <id>"))
	}
	id := args[0]
	if strings.ContainsAny(id, `/\:`) {
		fail(fmt.Errorf("tên key chỉ được dùng chữ, số, '-', '_'"))
	}

	// Đọc từ STDIN chứ không nhận qua đối số dòng lệnh: đối số nằm trong lịch sử
	// shell và trong bảng tiến trình, ai trên máy cũng đọc được.
	fmt.Print("  Dán API key rồi Enter (sẽ hiện trên màn hình): ")
	var key string
	fmt.Fscanln(os.Stdin, &key)
	key = strings.TrimSpace(key)
	if key == "" {
		fail(fmt.Errorf("không nhận được key"))
	}

	dir := aiapi.KeysDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(err)
	}
	_ = acl.Restrict(dir)
	p := filepath.Join(dir, id+".key")
	if err := os.WriteFile(p, []byte(key), 0o600); err != nil {
		fail(err)
	}
	_ = acl.Restrict(p)
	fmt.Printf("  ✓ đã lưu key %q vào %s\n", id, p)
	fmt.Println("  File này NGOÀI repo và đã siết quyền. Cấu hình chỉ tham chiếu bằng key_id.")
}

func apiDs() {
	a, done := open()
	defer done()
	c := a.Config()
	done()

	fmt.Println()
	if len(c.AI.Routes) == 0 {
		fmt.Println("  Chưa có route API nào. Thêm vào .sagent/project.toml:")
		fmt.Println()
		fmt.Println("    [ai]")
		fmt.Println(`    default_route = "grok"`)
		fmt.Println()
		fmt.Println("      [[ai.route]]")
		fmt.Println(`      ten = "grok"`)
		fmt.Println(`      base_url = "https://modelapi.vn/v1"`)
		fmt.Println(`      model = "grok-4.5"`)
		fmt.Println(`      key_id = "grok"        # tên file, KHÔNG phải key`)
		fmt.Println()
		fmt.Println("  Rồi đặt key: sagent api key grok")
		fmt.Println()
		return
	}
	fmt.Println("  Route API")
	fmt.Println()
	for _, r := range c.AI.Routes {
		dau := " "
		if r.Ten == c.AI.DefaultRoute {
			dau = "*"
		}
		// In key_id, KHÔNG in key.
		fmt.Printf("  %s %-12s %-32s %-16s key:%s\n", dau, r.Ten, r.BaseURL, r.Model, r.KeyID)
	}
	fmt.Println()
}

func apiGoi(ten string, args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu prompt: sagent api %s \"câu hỏi\"", ten))
	}
	a, done := open()
	defer done()

	// Gọi qua api.AICall chứ không tự tìm route: chỗ đó mới biết default_route và
	// fallback_routes. Tự dựng Route ở đây là bỏ qua fallback mà không ai thấy.
	kq, err := a.AICall(context.Background(), ten, strings.Join(args, " "))
	done()
	if err != nil {
		fail(err)
	}
	fmt.Println()
	fmt.Println(kq.NoiDung)
	fmt.Println()
	// Đường API tiêu TIỀN theo token, khác đường CLI tiêu hạn mức. Không in ra
	// thì người dùng không biết mình vừa tiêu gì.
	fmt.Printf("  %s · vào %d, ra %d, tổng %d token · %.1fs\n",
		kq.Model, kq.Usage.Vao, kq.Usage.Ra, kq.Usage.Tong, kq.Mat.Seconds())
}
