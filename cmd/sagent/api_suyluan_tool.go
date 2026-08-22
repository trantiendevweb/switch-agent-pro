package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
)

// CHỖ NGƯỜI DÙNG TERMINAL THẬT SỰ NHÌN THẤY phần suy luận và lời gọi tool.
//
// Hai thứ trong file này đều là "tầng cuối cùng" của hai đường đã có đủ mã ở
// mọi tầng dưới — và tầng cuối cùng đúng là chỗ dự án này đánh rơi năm lần
// trong ngày 22/08 (`route.kiem`, nút Duyệt/Từ chối, `plugin.list`,
// `sagent help`, và chính bảng năng lực nửa API).
//
// Vì vậy phần dựng chữ được TÁCH RA THÀNH HÀM THUẦN, trả về `[]string` thay vì
// tự `fmt.Println`: bài kiểm gọi được đúng hàm mà người dùng nhìn thấy đầu ra
// của nó. Một bài kiểm chỉ `grep` mã nguồn xem có gọi hàm nào không thì vẫn
// xanh khi hàm đó in ra một câu sai.

// dongSuyLuan dựng những dòng in ra cho PHẦN NGHĨ của model.
//
// `hien` là cờ `--suy-luan`. Hai nhánh, và cả hai đều phải nói được điều gì đó:
//
//   - BẬT cờ: in cả phần nghĩ, hoặc — nếu không có — in câu giải thích ô trống
//     đó thuộc về ai, kèm lệnh tra lại. Người gõ cờ này đang hỏi một câu cụ
//     thể; trả lời họ bằng khoảng trắng là tệ nhất.
//   - TẮT cờ: im lặng, TRỪ khi lượt đó CÓ phần nghĩ — lúc đó nhắc một dòng là
//     nó dài bao nhiêu và gõ gì để xem. Người dùng đã trả tiền cho số ký tự đó
//     rồi; không nhắc thì họ không có cách nào biết mình đang bỏ phí cái gì.
func dongSuyLuan(k aiapi.KhoiSuyLuan, hien bool) []string {
	if !hien {
		if !k.Co {
			return nil
		}
		return []string{fmt.Sprintf("  ⌥ lượt này kèm %d ký tự suy luận của model "+
			"(đã tính tiền trong token ra) — xem bằng cờ --suy-luan", k.SoKyTu)}
	}
	if k.Co {
		out := []string{fmt.Sprintf("  ┌─ phần suy luận của model · %d ký tự", k.SoKyTu)}
		for _, d := range strings.Split(strings.ReplaceAll(k.NoiDung, "\r\n", "\n"), "\n") {
			out = append(out, "  │ "+d)
		}
		return append(out, "  └─")
	}
	// KHÔNG có phần nghĩ. Đây là chỗ dễ viết sai nhất của cả tính năng: một
	// dòng "model không suy luận" ở đây là một lời khai không đo được từ đâu.
	// Chuỗi rỗng mang HAI nghĩa (model không nghĩ / nhà cung cấp không trả), và
	// `aiapi.DocSuyLuan` đã tra bảng năng lực để biết mình đang ở nghĩa nào.
	return []string{
		"  ⌥ lượt này KHÔNG có phần suy luận đọc được.",
		"    " + k.ViSaoRong,
		"    Tra lại: " + k.DanToi,
	}
}

// dongToolCall dựng những dòng in ra cho lời gọi tool model đòi chạy.
//
// Câu "sagent KHÔNG chạy chúng" là bắt buộc và không được rút gọn: người đọc
// thấy một lời gọi hàm in ra màn hình sẽ mặc định là nó đã chạy. Thư viện cố ý
// không chạy (xem đầu internal/aiapi/tool.go), và một quyết định như vậy mà
// không nói ra thì thành một cái bẫy.
func dongToolCall(kq aiapi.KetQua) []string {
	if len(kq.ToolCalls) == 0 {
		return []string{
			"  ⌥ model KHÔNG đòi gọi tool nào ở lượt này — nó trả lời thẳng bằng chữ.",
			"    Muốn ép gọi: --tool-chon " + aiapi.ChonToolBatBuoc +
				" (có nhà cung cấp từ chối cách ép này, xem sagent nang-luc-api)",
		}
	}
	out := []string{fmt.Sprintf("  ┌─ model đòi gọi %d tool. sagent KHÔNG chạy chúng — "+
		"đây là lời gọi nguyên văn, bạn tự quyết:", len(kq.ToolCalls))}
	for _, l := range kq.ToolCalls {
		out = append(out, fmt.Sprintf("  │ %s(%s)", l.Ten(), l.ThamSo()))
		if l.ID != "" {
			out = append(out, "  │   id: "+l.ID)
		}
		// Tham số hỏng thì NÓI RA ngay đây: nó là chữ do model sinh, và người
		// định chạy tool bằng tay cần biết trước khi dán vào đâu đó.
		if _, err := l.ThamSoJSON(); err != nil {
			out = append(out, "  │   ⚠ "+err.Error())
		}
	}
	return append(out, "  └─")
}

// docFileTool nạp định nghĩa tool từ một file JSON.
//
// Định dạng là MỘT MẢNG đúng khuôn giao thức, tức chính thứ dán được thẳng từ
// tài liệu nhà cung cấp:
//
//	[{"type":"function","function":{"name":"lay_gio","description":"…",
//	  "parameters":{"type":"object","properties":{…},"required":["…"]}}}]
//
// Không bịa một định dạng "gọn hơn" của riêng sagent: định nghĩa tool là thứ
// người ta chép qua chép lại giữa các dự án, và một khuôn riêng bắt họ dịch tay
// — chỗ dịch tay là chỗ `required` rụng mất mà không ai thấy, rồi model gọi tool
// thiếu tham số và hỏng lúc chạy.
func docFileTool(duong string) ([]aiapi.Tool, error) {
	b, err := os.ReadFile(duong)
	if err != nil {
		return nil, fmt.Errorf("không đọc được file tool: %w", err)
	}
	var ds []aiapi.Tool
	if err := json.Unmarshal(b, &ds); err != nil {
		return nil, fmt.Errorf("file tool %s không phải JSON đúng khuôn: %w\n"+
			`     Khuôn cần: [{"type":"function","function":{"name":"…","description":"…",`+
			`"parameters":{…}}}]`, duong, err)
	}
	if len(ds) == 0 {
		return nil, fmt.Errorf("file tool %s không có định nghĩa nào", duong)
	}
	return ds, nil
}

// apiGoiTool chạy nhánh `--tool`: gửi định nghĩa tool đi, in lời gọi tool về.
//
// ĐI ĐÍCH DANH MỘT ROUTE, KHÔNG NHẢY DỰ PHÒNG, và nói ra điều đó. Lý do là một
// ranh giới thật chứ không phải làm biếng: bộ chuyển route dự phòng nằm trong
// `internal/api` và nó gọi `aiapi.Goi` — đường không mang tool. Cho nhánh này
// mượn đường đó thì lượt dự phòng sẽ gửi đi một yêu cầu KHÔNG có `tools`, model
// trả lời bằng chữ, và người dùng nhận về "model không đòi gọi tool nào" — một
// câu sai, sinh ra từ một chỗ chuyển route im lặng. Thà không có dự phòng còn
// hơn có một cái dự phòng nói dối.
func apiGoiTool(ten, prompt, fileTool, chonTool string, xemSuyLuan bool) {
	tools, err := docFileTool(fileTool)
	if err != nil {
		fail(err)
	}
	a, done := open()
	defer done()
	routes := a.AIRoutes()
	macDinh := a.Config().AI.DefaultRoute
	done()

	r, err := chonRouteDichDanh(routes, ten, macDinh)
	if err != nil {
		fail(err)
	}

	kq, err := aiapi.GoiTool(context.Background(), r, prompt, tools, chonTool)
	if err != nil {
		fail(err)
	}

	fmt.Println()
	if s := strings.TrimSpace(kq.NoiDung); s != "" {
		fmt.Println(s)
		fmt.Println()
	}
	for _, d := range dongToolCall(kq) {
		fmt.Println(d)
	}
	for _, d := range dongSuyLuan(aiapi.DocSuyLuan(kq, routes), xemSuyLuan) {
		fmt.Println(d)
	}
	fmt.Printf("  %s · %s · vào %d, ra %d, tổng %d token · %.1fs\n",
		kq.Route, kq.Model, kq.Usage.Vao, kq.Usage.Ra, kq.Usage.Tong, kq.Mat.Seconds())
	fmt.Println("  ! lượt --tool đi ĐÍCH DANH route này, KHÔNG nhảy route dự phòng " +
		"(đường dự phòng chưa mang tool đi được).")
}

// chonRouteDichDanh tìm route sẽ gọi cho những nhánh KHÔNG có dự phòng
// (`--tool`, `--anh`, `--so-do`), và nói rõ khi không tìm được.
//
// Tên rỗng thì dùng `default_route`. KHÔNG tự lấy route đầu danh sách khi thiếu
// khai báo: đoán hộ ở đây là gửi tiền của người dùng tới một nhà cung cấp họ
// không chọn.
func chonRouteDichDanh(ds []aiapi.Route, ten, macDinh string) (aiapi.Route, error) {
	if ten == "" {
		ten = macDinh
	}
	if ten == "" {
		return aiapi.Route{}, fmt.Errorf("chưa biết gọi route nào: gõ `sagent api --tool " +
			"<file> <route> \"câu hỏi\"` hoặc khai `default_route` trong .sagent/project.toml")
	}
	for _, r := range ds {
		if r.Ten == ten {
			return r, nil
		}
	}
	var co []string
	for _, r := range ds {
		co = append(co, r.Ten)
	}
	if len(co) == 0 {
		return aiapi.Route{}, fmt.Errorf("chưa khai route API nào — xem mẫu: sagent api ds")
	}
	return aiapi.Route{}, fmt.Errorf("không có route %q. Đang khai: %s", ten, strings.Join(co, ", "))
}
