package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
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
	case "artifacts", "file":
		if len(args) < 2 {
			fail(fmt.Errorf("thiếu số lần chạy. Ví dụ: sagent flow artifacts 51"))
		}
		flowArtifacts(args[1], args[2:])
	default:
		fail(fmt.Errorf("không hiểu 'flow %s' — dùng: list | show | validate | run | runs | tom-tat | "+
			"artifacts | approve | reject | resume | huy", sub))
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
	// SỔ ROUTE lấy TRƯỚC khi đóng API: phần soi `can` ở cuối hàm cần nó để
	// tra bảng năng lực. Không có nó thì `flow show` và `flow validate` nói KHÁC
	// NHAU về cùng một file — và `flow show` là lệnh người ta đọc đúng lúc
	// quyết định có chạy hay không.
	dsRoute := a.AIRoutes()
	done()

	fmt.Printf("\n  %s — %s\n\n", f.Name, f.Desc)
	if len(f.Vars) > 0 {
		fmt.Println("  Biến (ghi đè bằng --var ten=giatri):")
		for k, v := range f.Vars {
			fmt.Printf("    %-10s %s\n", k, truncate(v, 60))
		}
		fmt.Println()
	}
	// Bước nào bị LOẠI khỏi lịch chạy thường — bước gỡ lại (`compensate`) và
	// bước chạy thay (`fallback`). Tính một lần cho cả flow, đúng cách bộ thực
	// thi tính (flow.BuocNgoaiLichThuong — cùng một hàm, không đếm lại).
	ngoaiLich := flow.BuocNgoaiLichThuong(f)

	fmt.Println("  Thứ tự chạy:")
	for i, s := range order {
		dep := ""
		if len(s.Needs) > 0 {
			dep = "  ← " + strings.Join(s.Needs, ", ")
		}
		fmt.Printf("   %d. %-10s [%s]%s\n", i+1, s.ID, s.Type, dep)
		// BƯỚC NGOÀI LỊCH THƯỜNG (gỡ lại / chạy thay): nói ra NGAY dưới tên
		// bước, trước cả prompt.
		//
		// Đây là dòng quan trọng nhất trong cả bảng này. Chúng đứng trong danh
		// sách "Thứ tự chạy" y như mọi bước khác, nên người đọc đếm chúng vào
		// kế hoạch — trong khi bộ thực thi LOẠI chúng khỏi lịch chạy thường và
		// chỉ gọi khi có sự cố. Không có dòng này thì bảng đang nói sai.
		if cho := ngoaiLich[s.ID]; cho != "" {
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
		// Nhu cầu năng lực in cho bước `model` RIÊNG, vì bước `route` đã có nó
		// trong dòng "chọn đường" (MoTaRoute) — in hai lần thì bảng nói cùng
		// một chuyện hai kiểu. Và in KỂ CẢ khi `route` rỗng: bước đi đường mặc
		// định mà đòi `goi-tool` vẫn là một ràng buộc người đọc cần thấy.
		if c := flow.MoTaCan(s); c != "" && s.Type == flow.TypeModel {
			fmt.Printf("      cần đường làm được: %s\n", c)
		}
		for _, d := range moTaBaTruong(s.Artifact, s.Idempotent, buocGoLaiCua(s), buocChayThayCua(s)) {
			fmt.Printf("      %s\n", d)
		}
	}

	// Cảnh báo ngay ở đây, đừng để tới lúc chạy mới biết.
	//
	// Hai nửa, giống hệt `flow validate`: phần hình dạng (flow.Validate) và
	// phần tra bảng năng lực (api.VanDeCanTheoBang). Chỉ gọi nửa đầu thì hai
	// lệnh nói khác nhau về cùng một file, và người đọc sẽ tin lệnh họ đang
	// mở chứ không đi gõ thêm một lệnh nữa.
	ps := append(flow.Validate(f), api.VanDeCanTheoBang(f, dsRoute)...)
	if len(ps) > 0 {
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

// moTaBaTruong dựng dòng mô tả cho những trường mà `flow show` từng KHÔNG hiện:
// `artifact`, `idempotent`, `compensate` — và nay cả `fallback`.
//
// VÌ SAO ĐÚNG NHỮNG TRƯỜNG NÀY: tất cả đổi HÀNH VI của lượt chạy theo cách
// không suy ra được từ tên bước hay từ sơ đồ phụ thuộc.
//
//	artifact    bước để lại FILE cho bước sau — đường truyền thứ hai, không đi
//	            qua {{steps.x.output}} nên nhìn sơ đồ không thấy;
//	idempotent  bước có thể KHÔNG CHẠY lần này vì một lượt trước đã làm xong;
//	compensate  bước hỏng thì một bước khác chạy để GỠ LẠI, rồi cả lượt dừng;
//	fallback    bước hỏng thì một bước khác chạy THAY, và lượt chạy ĐI TIẾP —
//	            với kết quả của bước chạy thay nằm ở chỗ của bước hỏng.
//
// Người đọc `flow show` đang quyết định có bấm chạy hay không. Giấu những thứ
// này đi thì họ quyết định trên một bản kế hoạch thiếu mấy hành vi.
//
// Nhận GIÁ TRỊ chứ không nhận flow.Step: `sagent flow show` đọc từ Step, còn
// bảng chạy khan đọc từ api.BuocKho đi qua JSON. Hai mặt phải in ra ĐÚNG một
// câu chữ, và cách chắc chắn nhất là chúng gọi chung một hàm — chứ không phải
// hai hàm "giống nhau" viết ở hai file.
func moTaBaTruong(artifact map[string]string, idempotent bool, buocGoLai, buocChayThay string) []string {
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
	if buocChayThay != "" {
		// Nói cả chỗ ĐỌC KẾT QUẢ, không chỉ tên bước: đó là nửa hay quên nhất
		// của `fallback`, và quên nó thì người viết flow đi trỏ
		// `{{steps.<bước chạy thay>.output}}` — một ô rỗng ở mọi lượt suôn sẻ.
		out = append(out, "hỏng thì: chạy bước \""+buocChayThay+"\" THAY rồi ĐI TIẾP "+
			"— kết quả của nó đọc bằng tên bước này")
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

// buocChayThayCua trả về id bước chạy thay của s, hoặc "" nếu s không dùng chính
// sách `fallback`. Cùng lý do tách như buocGoLaiCua: `Fallback` là một trường có
// thể nằm đó mà không có tác dụng gì, và in nó ra khi không có tác dụng là nói
// sai với đúng người đang quyết định có chạy hay không.
func buocChayThayCua(s flow.Step) string {
	if s.OnFailure == flow.OnFailFallback {
		return s.Fallback
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
