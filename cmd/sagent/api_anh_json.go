package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
)

// CHỖ NGƯỜI DÙNG TERMINAL THẬT SỰ GỬI ĐƯỢC ẢNH VÀ ĐÒI ĐƯỢC JSON.
//
// Cùng vai trò và cùng luật với api_suyluan_tool.go: đây là "tầng cuối cùng"
// của hai đường vừa có đủ mã ở mọi tầng dưới — và tầng cuối đúng là chỗ dự án
// này đánh rơi năm lần trong ngày 22/08.
//
// Nên phần dựng chữ TÁCH RA THÀNH HÀM THUẦN trả về `[]string` thay vì tự
// `fmt.Println`: bài kiểm gọi được đúng hàm mà người dùng nhìn thấy đầu ra của
// nó. Một bài kiểm chỉ `grep` mã nguồn xem có gọi hàm nào không thì vẫn xanh
// khi hàm đó in ra một câu sai.

// dongAnhGuiDi dựng những dòng nói ẢNH NÀO ĐÃ THẬT SỰ ĐI.
//
// VÌ SAO IN LUÔN, KHÔNG CẦN CỜ: khi model trả lời "tôi không thấy ảnh nào", có
// hai nguyên nhân trông giống hệt nhau — nhà cung cấp nuốt phần `image_url`,
// hoặc sagent quên gắn ảnh. Không in ra thì người dùng không có cách nào phân
// biệt, và họ sẽ đi đổi nhà cung cấp cho một lỗi nằm trong máy mình.
//
// In cả số byte và số điểm ảnh vì cả hai đều là thứ nhà cung cấp từ chối theo:
// đo 22/08, grok-4.5 trả HTTP 400 cho ảnh 64 điểm ảnh.
func dongAnhGuiDi(ds []aiapi.Anh) []string {
	if len(ds) == 0 {
		return nil
	}
	out := []string{fmt.Sprintf("  ┌─ đã gửi kèm %d ảnh:", len(ds))}
	for _, a := range ds {
		out = append(out, "  │ "+a.MoTa())
	}
	return append(out, "  └─")
}

// dongCanhBaoTruocKhiGui in những gì đã soát ra TRƯỚC khi chạm mạng.
//
// Đây là chỗ ba trạng thái của bảng năng lực đi tới người dùng. Câu "CHƯA ai đo
// route này" phải hiện ra chứ không được nuốt: nuốt nó đi là lượt gọi thử
// nghiệm trông y hệt lượt gọi chắc chắn, và khi nó hỏng thì không ai biết vì sao.
func dongCanhBaoTruocKhiGui(canh []string) []string {
	var out []string
	for _, c := range canh {
		for i, d := range strings.Split(c, "\n") {
			if i == 0 {
				out = append(out, "  ⚠ "+d)
				continue
			}
			out = append(out, "  "+d)
		}
	}
	return out
}

// dongCoCauTruc dựng những dòng cho câu trả lời ĐÃ ĐỌC LẠI theo schema.
//
// Ba ca, ba câu khác nhau — và đó là cả lý do hàm này tồn tại thay vì in thẳng
// `kq.NoiDung`. Ca tệ nhất KHÔNG phải HTTP 400 mà là HTTP 200 kèm văn xuôi: lúc
// đó câu trả lời trông tử tế, `Usage` đẹp, không lỗi nào — và bước sau của flow
// gọi `json.Unmarshal` rồi hỏng, cách chỗ gây lỗi vài bước. Xem cocautruc.go.
func dongCoCauTruc(k aiapi.KhoiCoCauTruc) []string {
	if k.DungDuocNgay() {
		out := []string{"  ┌─ " + aiapi.MoTaCoCauTruc(k)}
		// In lại JSON đã CHUẨN HOÁ, thụt lề: đây là thứ bước sau của flow sẽ
		// nhận, và nó có thể khác thứ model in ra (rào ```json đã gỡ).
		if b, err := json.MarshalIndent(k.Gia, "  │ ", "  "); err == nil {
			out = append(out, "  │ "+string(b))
		}
		return append(out, "  └─")
	}
	return []string{
		"  ⌥ câu trả lời KHÔNG dùng được ngay làm dữ liệu có cấu trúc.",
		"    " + aiapi.MoTaCoCauTruc(k),
		"    Tra lại: " + k.DanToi,
	}
}

// docFileSoDo nạp JSON Schema từ file và dựng `response_format`.
//
// Định dạng là CHÍNH JSON Schema, tức thứ dán được thẳng từ tài liệu nhà cung
// cấp — không bọc thêm một lớp "gọn hơn" của riêng sagent. Cùng lý do đã viết
// cho `docFileTool`: schema là thứ người ta chép qua chép lại giữa các dự án,
// và một khuôn riêng bắt họ dịch tay — chỗ dịch tay là chỗ `required` rụng mất
// mà không ai thấy, rồi model trả JSON thiếu khoá và hỏng lúc chạy.
//
// Tên schema lấy từ tên file: giao thức đòi một cái tên, và bắt người dùng gõ
// thêm một cờ nữa chỉ để đặt tên cho thứ họ vừa đặt tên là thừa.
func docFileSoDo(duong string) (*aiapi.DangTraLoi, error) {
	b, err := os.ReadFile(duong)
	if err != nil {
		return nil, fmt.Errorf("không đọc được file schema: %w", err)
	}
	var so map[string]any
	if err := json.Unmarshal(b, &so); err != nil {
		return nil, fmt.Errorf("file schema %s không phải JSON đọc được: %w\n"+
			`     Khuôn cần là chính JSON Schema: {"type":"object","properties":{…},"required":[…]}`,
			duong, err)
	}
	if len(so) == 0 {
		return nil, fmt.Errorf("file schema %s rỗng", duong)
	}
	if so["required"] == nil {
		// KHÔNG chặn — schema không có `required` vẫn hợp lệ. Nhưng nói ra, vì
		// thiếu nó thì không ai đối chiếu được câu trả lời với hợp đồng, và ca
		// "đúng JSON, sai hợp đồng" sẽ đi qua trong im lặng.
		fmt.Printf("  ⚠ schema %s không khai `required` — sagent sẽ không đối chiếu được "+
			"câu trả lời với hợp đồng nào cả.\n", duong)
	}
	return aiapi.SoDoNghiem(tenSoDoTuDuong(duong), so), nil
}

// tenSoDoTuDuong lấy tên schema từ tên file, bỏ đuôi.
func tenSoDoTuDuong(duong string) string {
	ten := duong
	if i := strings.LastIndexAny(ten, `/\`); i >= 0 {
		ten = ten[i+1:]
	}
	if i := strings.LastIndex(ten, "."); i > 0 {
		ten = ten[:i]
	}
	if ten = strings.TrimSpace(ten); ten == "" {
		return "so_do"
	}
	return ten
}

// apiGoiKem chạy nhánh `--anh` / `--so-do`.
//
// ĐI ĐÍCH DANH MỘT ROUTE, KHÔNG NHẢY DỰ PHÒNG, và nói ra điều đó — cùng ranh
// giới với nhánh `--tool` và cùng lý do: bộ chuyển route dự phòng nằm trong
// `internal/api` và nó gọi `aiapi.Goi`, đường KHÔNG mang ảnh lẫn schema. Cho
// nhánh này mượn đường đó thì lượt dự phòng gửi đi một yêu cầu trơ, và người
// dùng nhận về "route này không đọc được ảnh" — một câu sai, sinh ra từ một chỗ
// chuyển route im lặng.
func apiGoiKem(ten, prompt string, anhFile []string, fileSoDo string, cuGui, xemSuyLuan bool) {
	anh, err := aiapi.DocNhieuAnh(anhFile)
	if err != nil {
		fail(err)
	}
	var dang *aiapi.DangTraLoi
	if fileSoDo != "" {
		if dang, err = docFileSoDo(fileSoDo); err != nil {
			fail(err)
		}
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

	kq, err := aiapi.GoiKem(context.Background(), r, prompt, aiapi.TuyChonGoi{
		Anh:            anh,
		TraLoiTheoSoDo: dang,
		DsRoute:        routes,
		CuGui:          cuGui,
	})
	if err != nil {
		// Cảnh báo đã soát ra được in TRƯỚC lời báo lỗi: câu "ảnh chỉ có 64
		// điểm ảnh" chính là thứ giải thích cái HTTP 400 sắp hiện ra, và in nó
		// sau thì người đọc đã bỏ đi rồi.
		fmt.Println()
		for _, d := range dongCanhBaoTruocKhiGui(kq.CanhBaoTruocKhiGui) {
			fmt.Println(d)
		}
		fail(err)
	}

	fmt.Println()
	for _, d := range dongCanhBaoTruocKhiGui(kq.CanhBaoTruocKhiGui) {
		fmt.Println(d)
	}
	if s := strings.TrimSpace(kq.NoiDung); s != "" && dang == nil {
		// Có schema thì câu trả lời được in trong khối JSON bên dưới — in hai
		// lần là bắt người đọc tự so hai cục chữ xem có khác nhau không.
		fmt.Println(s)
		fmt.Println()
	}
	for _, d := range dongAnhGuiDi(anh) {
		fmt.Println(d)
	}
	if dang != nil {
		for _, d := range dongCoCauTruc(aiapi.DocCoCauTruc(kq, dang)) {
			fmt.Println(d)
		}
	}
	for _, d := range dongSuyLuan(aiapi.DocSuyLuan(kq, routes), xemSuyLuan) {
		fmt.Println(d)
	}
	if canh := aiapi.CanhBaoThieuUsage(kq); canh != "" {
		fmt.Printf("  ⚠ %s\n", canh)
	}
	fmt.Printf("  %s · %s · vào %d, ra %d, tổng %d token · %.1fs\n",
		kq.Route, kq.Model, kq.Usage.Vao, kq.Usage.Ra, kq.Usage.Tong, kq.Mat.Seconds())
	fmt.Println("  ! lượt này đi ĐÍCH DANH route trên, KHÔNG nhảy route dự phòng " +
		"(đường dự phòng chưa mang ảnh/schema đi được).")
}
