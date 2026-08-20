package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/acl"
	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
)

// `sagent api` — đường THỨ HAI: gọi thẳng AI API, không qua CLI agent.
//
//	sagent api key <id>            đặt API key (đọc từ stdin, KHÔNG hiện trên màn hình)
//	sagent api ds                  liệt kê route đã cấu hình
//	sagent api --lich-su [n]       sổ lời gọi: tiêu bao nhiêu, ở đâu, có chạy được không
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
	case "--lich-su", "lich-su", "--history":
		apiLichSu(rest(args))
	default:
		ten, hoi := tachRouteVaPrompt(args)
		apiGoi(ten, hoi)
	}
}

// tachRouteVaPrompt quyết định tham số đầu là TÊN ROUTE hay đã là câu hỏi.
//
// VÌ SAO CẦN: `sagent api <route> "câu hỏi"` luôn coi tham số đầu là tên route,
// nên KHÔNG có cách nào truyền route rỗng — mà route rỗng mới là đường đi qua
// `default_route` và bộ chuyển route dự phòng. Tức là cả tính năng fallback của
// Pha 4 có mã, có test, có đường web, mà từ terminal không ai với tới được.
//
// Đo được 20/08: route `deepseek` trả HTTP 503 (hỏng phía nhà cung cấp — đúng
// loại đáng chuyển tiếp), nhưng `sagent api deepseek "..."` vẫn chỉ báo lỗi rồi
// dừng, vì gọi đích danh thì cố ý không chuyển route. Không có lối nào khác.
//
// Cách phân biệt: tham số đầu KHỚP tên một route đã khai thì nó là route. Không
// khớp thì cả dãy là câu hỏi, và lượt gọi đi đường mặc định + dự phòng. Tên
// route là định danh ngắn không dấu cách, còn câu hỏi thì hầu như luôn có dấu
// cách — nên hai thứ này không đụng nhau trong thực tế.
//
// Gõ nhầm tên route thì sao? Thì câu đó thành prompt và vẫn được trả lời, kèm
// dòng "route · model" in ra ngay dưới — nhìn là biết mình vừa hỏi đường nào.
// Thà vậy còn hơn chặn người dùng lại vì một cái tên họ không định gõ.
func tachRouteVaPrompt(args []string) (string, []string) {
	a, done := open()
	routes := a.AIRoutes()
	done()
	for _, r := range routes {
		if r.Ten == args[0] {
			return args[0], rest(args)
		}
	}
	return "", args
}

// apiLichSu in sổ lời gọi API — action "api.history".
//
// KHÔNG in prompt và câu trả lời: sổ không lưu chúng, cố ý. Xem ghi chú
// migration v7 trong internal/store.
func apiLichSu(args []string) {
	n := 20
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
			n = v
		}
	}
	a, done := open()
	defer done()
	ds, err := a.AIHistory(n)
	done()
	if err != nil {
		fail(err)
	}
	fmt.Println()
	if len(ds) == 0 {
		fmt.Println("  Sổ lời gọi API còn trống. Gọi thử: sagent api <route> \"câu hỏi\"")
		fmt.Println()
		return
	}
	fmt.Println("  Lịch sử lời gọi API (không lưu câu hỏi và câu trả lời)")
	fmt.Println()
	var tongVao, tongRa, hong int
	for _, g := range ds {
		dau := "✓"
		if !g.OK {
			dau, hong = "✗", hong+1
		}
		fmt.Printf("  %s %s  %-12s %-20s vào %5d, ra %5d  %.1fs\n",
			dau, g.Luc.Format("02/01 15:04"), g.Route, g.Model,
			g.TokensIn, g.TokensOut, float64(g.Mili)/1000)
		if !g.OK {
			// Nguyên văn lỗi, kèm request id của nhà cung cấp — đó là thứ duy
			// nhất dùng được khi phải đi hỏi họ.
			fmt.Printf("      %s\n", motDong(g.LyDo))
		}
		tongVao += g.TokensIn
		tongRa += g.TokensOut
	}
	fmt.Println()
	fmt.Printf("  %d lời gọi, %d hỏng · vào %d, ra %d token\n", len(ds), hong, tongVao, tongRa)
	// Chưa có bảng giá theo model nên KHÔNG in một con số tiền: bịa ra thì nó
	// trông như đã đo. Token là thứ đo được, in token.
	fmt.Println()
}

// motDong ép lý do hỏng về một dòng để bảng không vỡ. Thân lỗi nguyên văn của
// nhà cung cấp có thể dài nhiều dòng; đây chỉ là chỗ liếc, muốn đọc đủ thì xem
// mặt web hoặc sổ.
func motDong(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\r", " "), "\n", " "))
	if len(s) > 150 {
		s = s[:150] + "…"
	}
	return s
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
		// PHẢI in dòng này. Khai hai [[ai.route]] rồi tưởng là có dự phòng là
		// hiểu sai — danh sách dự phòng khai RIÊNG, và không khai thì route
		// chính hỏng là dừng luôn. Đã mất một lúc đi tìm chính vì chỗ này.
		fmt.Println(`    fallback_routes = ["deepseek"]   # route chính hỏng thì nhảy sang đây`)
		fmt.Println()
		fmt.Println("      [[ai.route]]")
		fmt.Println(`      ten = "grok"`)
		fmt.Println(`      base_url = "https://modelapi.vn/v1"`)
		fmt.Println(`      model = "grok-4.5"`)
		fmt.Println(`      key_id = "grok"        # tên file, KHÔNG phải key`)
		fmt.Println()
		fmt.Println("  Rồi đặt key: sagent api key grok")
		fmt.Println()
		fmt.Println(`  Gọi:  sagent api "câu hỏi"        → default_route, hỏng thì nhảy dự phòng`)
		fmt.Println(`        sagent api grok "câu hỏi"   → ĐÍCH DANH, cố ý không nhảy đi đâu`)
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
	// `--stream`: in chữ ra NGAY khi nhận được thay vì đợi cả câu.
	//
	// Đo được: một lượt grok-4.5 mất 13,6 giây (docs/DO-LUONG.md). Không có cờ
	// này thì người dùng nhìn màn hình đứng im 13 giây — không phân biệt được
	// "đang nghĩ" với "đã treo".
	stream, args := boolFlag(args, "--stream")
	if len(args) == 0 {
		fail(fmt.Errorf("thiếu prompt: sagent api %s \"câu hỏi\"", ten))
	}
	a, done := open()
	defer done()

	// Gọi qua api.AICall chứ không tự tìm route: chỗ đó mới biết default_route và
	// fallback_routes. Tự dựng Route ở đây là bỏ qua fallback mà không ai thấy.
	var kq aiapi.KetQua
	var err error
	if stream {
		fmt.Println()
		kq, err = a.AICallStream(context.Background(), ten, strings.Join(args, " "),
			func(s string) { fmt.Print(s) })
		fmt.Println()
	} else {
		kq, err = a.AICall(context.Background(), ten, strings.Join(args, " "))
	}
	done()
	if err != nil {
		fail(err)
	}
	if !stream {
		fmt.Println()
		fmt.Println(kq.NoiDung)
	}
	fmt.Println()
	// Nhà cung cấp không trả usage ở chế độ stream thì NÓI RA. Im lặng ghi 0 là
	// biến "chưa đo" thành "miễn phí" ngay trước mắt người trả tiền.
	if canh := aiapi.CanhBaoThieuUsage(kq); canh != "" {
		fmt.Printf("  ⚠ %s\n", canh)
	}
	// Đường API tiêu TIỀN theo token, khác đường CLI tiêu hạn mức. Không in ra
	// thì người dùng không biết mình vừa tiêu gì.
	fmt.Printf("  %s · %s · vào %d, ra %d, tổng %d token · %.1fs\n",
		kq.Route, kq.Model, kq.Usage.Vao, kq.Usage.Ra, kq.Usage.Tong, kq.Mat.Seconds())
	// ĐÃ CHUYỂN ROUTE thì nói ra ngay dưới câu trả lời, kèm lỗi NGUYÊN VĂN của
	// route chính. CLI không nghe bus event, nên nếu chỉ bus.Warnf thì người
	// đứng ở terminal không bao giờ biết câu này đến từ nhà cung cấp khác.
	if kq.DaChuyenRoute() {
		fmt.Printf("  ! route %q hỏng, câu trên do %q trả lời:\n      %s\n",
			kq.RouteChinh, kq.Route, motDong(kq.LoiChinh))
	}
}
