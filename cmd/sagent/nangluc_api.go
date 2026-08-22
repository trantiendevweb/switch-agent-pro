package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
)

// `sagent nang-luc-api` — bảng năng lực của NỬA API.
//
//	sagent nang-luc-api                 bảng mọi route đã cấu hình
//	sagent nang-luc-api <route>         chỉ một route
//	sagent nang-luc-api --chua-do       chỉ những ô chưa ai đo
//	sagent nang-luc-api --do [<route>]  ĐO THẬT: chạm mạng, TIÊU TOKEN
//	sagent nang-luc-api --do --dan      in sẵn khối Go để dán vào sổ số đo
//
// Đối xứng với `sagent nang-luc` ở nửa CLI, và cố ý gọi tên gần giống: hai bảng
// trả lời cùng một câu ("thứ này làm được gì") cho hai đường khác nhau của dự
// án. Chỗ khác nằm ở giá của câu trả lời — bảng bên kia dò adapter tại chỗ nên
// miễn phí tuyệt đối, còn bên này phải chạm mạng mới biết, nên phần chạm mạng
// tách hẳn ra sau cờ `--do`.
func cmdNangLucAPI(args []string) {
	// `sagent help` là một chuỗi viết TAY trong main.go — file của agent khác,
	// nên lệnh này không tự thêm mình vào đó được (hai dòng cần cắm nằm trong
	// docs/BAO-CAO-NANGLUC-API.md). Chừng nào chưa cắm, `--giup` là cách duy
	// nhất người dùng terminal đọc được cách gõ, nên nó phải có ở đây.
	if giup, _ := boolFlag(args, "--giup"); giup {
		giupNangLucAPI()
		return
	}
	do, args := boolFlag(args, "--do")
	chuaDo, args := boolFlag(args, "--chua-do")
	dan, args := boolFlag(args, "--dan")

	var route string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			fail(fmt.Errorf("chưa có cờ %q. Xem: sagent nang-luc-api --giup", a))
		}
		route = a
	}
	if dan && !do {
		fail(fmt.Errorf("--dan chỉ dùng cùng --do: nó in số đo VỪA đo được để dán vào sổ, " +
			"không phải in lại sổ cũ"))
	}
	if do {
		doNangLucAPI(route, dan)
		return
	}
	inBangNangLucAPI(route, chuaDo)
}

// giupNangLucAPI in cách gõ. Cùng một khối chữ với phần chú thích của
// cmdNangLucAPI ở trên — chép làm hai bản là hai bản sẽ lệch, nên chỗ này in ra
// đúng thứ người đọc mã cũng thấy.
func giupNangLucAPI() {
	fmt.Print(`
  sagent nang-luc-api — route API nào LÀM ĐƯỢC GÌ

    sagent nang-luc-api                 bảng mọi route đã cấu hình
    sagent nang-luc-api <route>         chỉ một route
    sagent nang-luc-api --chua-do       chỉ những ô chưa ai đo
    sagent nang-luc-api --do [<route>]  ĐO THẬT: chạm mạng bằng key thật, TỐN TOKEN
    sagent nang-luc-api --do --dan      in sẵn khối Go để dán vào sổ số đo

  Bảng KHÔNG tốn token: nó ghép phép đo mã nguồn (chạy tại chỗ) với sổ số đo đã
  chạy trước đó. Chỉ cờ --do mới chạm mạng.

  Ba trạng thái, và chúng KHÁC NHAU:

    ✓ làm được      đã đo, chạy được
    ✗ đã đo, KHÔNG  một KẾT LUẬN, không phải một khoảng trống
    ? chưa ai đo    chưa biết — không được lặng lẽ coi như chạy được

  Mỗi ô nói luôn VƯỚNG Ở ĐÂU: phía dự án (sửa được trong repo này), phía nhà
  cung cấp (sửa trong repo không cứu được), hay cả hai.

  Bảng bên nửa CLI: sagent nang-luc

`)
}

// inBangNangLucAPI in bảng. KHÔNG chạm mạng, KHÔNG tốn token.
func inBangNangLucAPI(route string, chiChuaDo bool) {
	a, done := open()
	defer done()
	ds, err := a.NangLucAPI(route)
	done()
	if err != nil {
		fail(err)
	}

	fmt.Println()
	fmt.Println("  Bảng năng lực — nửa API (không tốn token)")
	var tongChuaDo, tongLech int
	for _, b := range ds {
		fmt.Println()
		fmt.Printf("  %s  %s · %s\n", b.Ten, b.Model, b.BaseURL)
		tongChuaDo += b.SoChuaDo()
		tongLech += len(b.Lech)

		for _, m := range b.Muc {
			if chiChuaDo && m.TrangThai != aiapi.ChuaDo {
				continue
			}
			fmt.Printf("    %s %-22s %s\n", dauNangLuc(m.TrangThai), m.Khoa, m.Mo)
			fmt.Printf("        %s\n", m.BangChung)
			// Dòng "vướng đâu" là dòng quyết định người đọc đi sửa ở chỗ nào.
			// Không in cho ô xanh: ở đó không có gì để đi sửa, và một dòng
			// "không vướng gì" dưới mỗi ô xanh chỉ làm bảng dài gấp đôi.
			if m.TrangThai != aiapi.LamDuoc {
				fmt.Printf("        → %s\n", aiapi.NhanCho(m.Cho))
			}
		}
		// Chỗ lệch KHÔNG được giấu: một bảng năng lực sai thì tệ hơn không có
		// bảng, vì người vận hành sẽ tin nó.
		for _, l := range b.Lech {
			fmt.Printf("    ! LỆCH: %s\n", l)
		}
	}

	fmt.Println()
	switch {
	case tongLech > 0:
		fmt.Printf("  ! %d chỗ bảng chọi với chính nó — đọc phần LỆCH ở trên trước khi tin bảng này.\n", tongLech)
	case chiChuaDo && tongChuaDo == 0:
		fmt.Println("  Không còn năng lực nào chưa đo.")
	case tongChuaDo > 0:
		fmt.Printf("  %d ô chưa ai đo. Đo thật: sagent nang-luc-api --do\n", tongChuaDo)
	default:
		fmt.Println("  Mọi ô đều đã đo.")
	}
	fmt.Println()
}

// doNangLucAPI chạy phép đo THẬT — action "api.nang-luc-do".
//
// Nói trước là sẽ tốn tiền, rồi mới chạy. Không hỏi xác nhận: đây là lệnh người
// ta chủ động gõ kèm hẳn một cờ `--do`, và một câu hỏi "có chắc không?" ở giữa
// làm lệnh không dùng được trong script.
func doNangLucAPI(route string, dan bool) {
	a, done := open()
	defer done()

	fmt.Println()
	fmt.Println("  Đo thật năng lực route API — CHẠM MẠNG bằng key thật và TIÊU TOKEN.")
	fmt.Println("  Mỗi route 7 phép đo, chạy tuần tự để khỏi tự gây HTTP 429.")
	fmt.Println()

	// Hạn chót của cả lượt nằm ở đây chứ không ở từng lời gọi: 7 phép đo × N
	// route, mỗi cái tự chờ 90 giây thì một route treo có thể giữ terminal hơn
	// mười phút mà không nói gì.
	ctx, huy := context.WithTimeout(context.Background(), 15*time.Minute)
	defer huy()

	ds, err := a.NangLucAPIDo(ctx, route)
	done()
	if err != nil {
		fail(err)
	}

	var tongToken int
	truoc := ""
	for _, k := range ds {
		if k.Route != truoc {
			fmt.Printf("  %s\n", k.Route)
			truoc = k.Route
		}
		fmt.Printf("    %s %-22s %s\n", dauNangLuc(k.TrangThai), k.Khoa,
			k.Mat.Round(time.Millisecond))
		fmt.Printf("        %s\n", k.Chi)
		tongToken += k.Usage.Tong
	}

	fmt.Println()
	fmt.Printf("  Lượt đo này tiêu %d token.\n", tongToken)

	if !dan {
		fmt.Println("  Dán vào sổ số đo: chạy lại kèm --dan")
		fmt.Println()
		return
	}

	// Đường từ phép đo tới bảng đi qua một lần DÁN TAY có chủ ý: số đo là thứ
	// được đọc và duyệt, không phải thứ tự bò vào mã nguồn sau lưng người ta.
	// Nhưng chép tay từng con số thì sai — nên in sẵn đúng khối cần dán.
	ngay := time.Now().Format("02/01")
	routes := map[string]aiapi.Route{}
	for _, r := range a.AIRoutes() {
		routes[r.Ten] = r
	}
	fmt.Println("  Dán khối dưới đây vào `soDoNangLuc` trong internal/aiapi/nangluc.go:")
	fmt.Println()
	for _, k := range ds {
		if k.TrangThai == aiapi.ChuaDo {
			// Ô chưa đo được KHÔNG vào sổ: sổ là nơi ghi những gì ĐÃ đo. Ghi
			// vào đó một ô "chưa đo" là biến sổ thành một bản sao của bảng, và
			// lần sau không ai phân biệt được nữa.
			continue
		}
		m := aiapi.GhiSoDo(routes[k.Route], k, ngay)
		fmt.Printf("\t{%q, %q, %q, %s,\n\t\t%q},\n",
			m.BaseURL, m.Model, m.Khoa, tenHangGo(m.TrangThai), m.BangChung)
	}
	fmt.Println()
}

// dauNangLuc đổi ba trạng thái thành ba dấu KHÁC NHAU.
//
// Ba dấu chứ không phải hai, ở mọi mặt. Gộp "đã đo, KHÔNG" với "chưa ai đo"
// thành một dấu ✗ là xoá mất đúng thứ bảng này sinh ra để nói.
func dauNangLuc(t aiapi.TrangThaiNangLuc) string {
	switch t {
	case aiapi.LamDuoc:
		return "✓"
	case aiapi.KhongLamDuoc:
		return "✗"
	default:
		return "?"
	}
}

// tenHangGo đổi trạng thái sang TÊN HẰNG trong mã Go, để khối `--dan` dán vào
// là biên dịch được. In chuỗi thô ("lam-duoc") thì dán xong không build.
func tenHangGo(t aiapi.TrangThaiNangLuc) string {
	switch t {
	case aiapi.LamDuoc:
		return "LamDuoc"
	case aiapi.KhongLamDuoc:
		return "KhongLamDuoc"
	default:
		return "ChuaDo"
	}
}

// ---------------------------- ghi tên lệnh vào bảng dispatch ----------------------------

// RANH GIỚI: `cmd/sagent/main.go` thuộc một agent khác đang chạy song song, nên
// hai dòng dispatch của lệnh này tự ghi vào bảng TỪ ĐÂY thay vì sửa file đó.
//
// An toàn được vì thứ tự init trong một gói là thứ tự TÊN FILE: "main.go" đứng
// trước "nangluc_api.go", nên bảng `commands` đã được main.go dựng xong trước
// khi init này chạy. Chỗ đó tinh vi, nên có hẳn một bài kiểm giữ nó —
// TestLenhNangLucAPIGoDuoc trong nangluc_api_test.go đỏ ngay nếu giả định này
// sai, thay vì lệnh lặng lẽ biến mất khỏi `sagent help`.
//
// Ghi theo KHOÁ nên nếu người điều phối cắm thêm đúng hai dòng này vào main.go
// thì cũng không sao: cùng khoá, cùng action, ghi đè bằng chính nó.
func init() {
	if commands == nil {
		commands = map[string]command{}
	}
	commands["nang-luc-api"] = command{
		"api.nang-luc",
		"route API nào làm được gì (làm được / không / chưa đo)",
		cmdNangLucAPI,
	}
	// Cờ `--do` của cùng lệnh, không phải tên lệnh cấp một — khai ở đây để test
	// ngang quyền thấy nó, giống `__apils` và `__rkiem`.
	commands["__nlado"] = command{
		"api.nang-luc-do",
		"đo THẬT năng lực route API (chạm mạng, tốn token)",
		nil,
	}
}
