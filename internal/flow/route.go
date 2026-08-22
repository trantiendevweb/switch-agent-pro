// ROUTE — node CHỌN ĐƯỜNG API, tách khỏi node gọi model.
//
// ============================================================================
// TRƯỚC ĐÂY ROUTE LÀ MỘT THUỘC TÍNH, KHÔNG PHẢI MỘT NODE
// ============================================================================
//
// `Step.Route` cho bước `model` khai đi đường nào. Nó vẫn còn và vẫn đúng cho
// trường hợp thường: một bước, một đường, khai cứng.
//
// Cái nó KHÔNG làm được là chia sẻ quyết định. Flow có năm bước `model` cùng
// phải đi một đường; đường đó chết thì cả năm bước tự đi hỏi lại, mỗi bước một
// lần kiểm, và — tệ hơn — mỗi bước có thể rơi sang một đường dự phòng KHÁC
// NHAU tuỳ lúc nó chạy. Năm bước của cùng một lượt trả lời bằng năm mô hình
// khác nhau, mà bảng chỉ ghi "model".
//
// Node `route` tách quyết định ấy ra một chỗ: kiểm một lần, chọn một lần, rồi
// chuyền TÊN đường cho các bước sau qua `{{steps.<id>.output>}}`.
//
//	[[flow.x.step]]
//	  id     = "chon-duong"
//	  type   = "route"
//	  routes = ["grok", "deepseek"]      # thứ tự ưu tiên; bỏ trống = default_route
//
//	[[flow.x.step]]
//	  id     = "hoi"
//	  type   = "model"
//	  needs  = ["chon-duong"]
//	  route  = "{{steps.chon-duong.output}}"
//	  prompt = "..."
//
// ============================================================================
// KHÔNG LÀM LẠI THỨ ĐÃ CÓ
// ============================================================================
//
// Phần khó của "đường này còn sống không" đã nằm ở internal/aiapi (Kiem — hỏi
// danh sách model, KHÔNG tốn token) và ở internal/api (ThuTuRoute — thứ tự
// route chính + đúng một route dự phòng, cùng luật với `sagent api`). Node này
// KHÔNG chép lại một dòng nào của chúng: nó khai một interface hẹp và để phần
// cắm ở internal/api gọi lại đúng những hàm đó.
//
// Nhờ vậy `sagent route kiem`, `sagent api "câu hỏi"` và node này không thể
// trôi khỏi nhau — chúng là cùng một đoạn mã.
//
// ============================================================================
// ĐẦU RA CỦA NODE NÀY LÀ DỮ LIỆU, KHÔNG PHẢI VĂN BẢN
// ============================================================================
//
// Output của bước `route` là ĐÚNG cái tên route, không gì khác: không lời chào,
// không dấu xuống dòng, không "đã chọn: grok". Vì nó sẽ được nhét thẳng vào
// `route = "{{steps.chon-duong.output}}"` của bước sau, và ở đó một ký tự thừa
// là một tên route không tồn tại.
//
// Phần người đọc cần — ứng viên nào bị loại vì sao — đi ra event bus, chứ không
// trộn vào output. Đây là điểm khác với mọi node khác của flow, và nó đủ dễ
// quên để đáng một khối chú thích riêng.
package flow

import (
	"context"
	"fmt"
	"strings"
)

// RouteChon là thứ node `route` cần từ phần cắm.
//
// Interface riêng, không nhét vào ModelRunner: một bên TIÊU TIỀN theo token và
// trả về chữ, một bên chỉ hỏi thăm sức khoẻ và trả về một cái tên. Gộp lại thì
// mọi cài đặt của ModelRunner phải mang thêm một hàm nó không dùng, và test của
// gói này mất khả năng dựng một cái giả chỉ cho một việc.
type RouteChon interface {
	// ChonRoute trả về route ĐẦU TIÊN dùng được trong ungVien, theo đúng thứ
	// tự đó. ungVien rỗng = để phần cắm quyết định theo cấu hình (default_route
	// rồi tới route dự phòng), y như `sagent api "câu hỏi"`.
	//
	// Không route nào dùng được thì trả về lỗi — KHÔNG trả về một cái tên đoán
	// bừa. Bước sau sẽ đem cái tên ấy đi gọi thật và tiêu tiền vào một đường đã
	// biết là chết.
	ChonRoute(ctx context.Context, ungVien []string) (KetQuaRoute, error)
}

// KetQuaRoute là câu trả lời của một lần chọn đường.
type KetQuaRoute struct {
	// Ten là route chọn được. Đây là thứ DUY NHẤT thành output của bước.
	Ten string

	// NhatKy là một dòng cho MỖI ứng viên đã xét, theo đúng thứ tự xét: đã chọn,
	// hay bị loại vì sao. Đi ra event bus, không vào output.
	//
	// Vì sao phải có: chọn được đường thứ hai nghĩa là đường thứ nhất đã chết,
	// và đó là tin đáng biết nhất trong cả lượt chạy. Không có nhật ký thì lượt
	// chạy vẫn xanh và không ai biết mình vừa hỏi một mô hình khác.
	NhatKy []string
}

// VanDeRoute soi phần `route` của cả flow.
func VanDeRoute(f Flow) []Problem {
	var ps []Problem
	loi := func(id, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: id, Msg: msg}) }
	nhac := func(id, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: id, Msg: msg, Warn: true}) }

	coBuocRoute := map[string]bool{}
	for _, s := range f.Steps {
		if s.Type == TypeRoute {
			coBuocRoute[s.ID] = true
		}
	}

	for _, s := range f.Steps {
		// `routes` ở một bước KHÔNG phải type route là một dòng chết: nó nằm đó
		// trông như có tác dụng. Cùng lớp với `plugin` khai nhầm chỗ.
		if len(s.Routes) > 0 && s.Type != TypeRoute {
			loi(s.ID, fmt.Sprintf("khai `routes` nhưng type = %q — chỉ bước type = \"route\" mới chọn đường; "+
				"bước `model` khai MỘT đường bằng `route = \"...\"`", s.Type))
		}

		if s.Type != TypeRoute {
			continue
		}

		// Bước `route` khai `route` (số ít) là lẫn hai trường. Không đoán hộ:
		// đọc nhầm ý ở đây nghĩa là chạy vào một đường người ta không chọn.
		if s.Route != "" {
			loi(s.ID, "bước `route` chọn đường bằng `routes = [...]` (số nhiều), không phải `route = \"...\"` — "+
				"một đường thì viết routes = [\""+s.Route+"\"]")
		}
		if s.Prompt != "" {
			loi(s.ID, "bước `route` không hỏi ai câu nào — bỏ `prompt` đi (nó chỉ chọn đường, không gọi model)")
		}
		if s.ForEach != "" {
			loi(s.ID, "không dùng `foreach` với `route` — mỗi lượt lặp sẽ chọn lại đúng cùng một tập đường")
		}
		for _, r := range s.Routes {
			if strings.TrimSpace(r) == "" {
				loi(s.ID, "`routes` có một mục rỗng")
			}
		}

		// Chọn được đường mà không bước nào dùng thì cả bước này là công cốc —
		// và người viết flow thường tưởng nó tự áp cho các bước sau.
		duocDung := false
		for _, k := range f.Steps {
			if strings.Contains(k.Route, "{{steps."+s.ID+".output}}") {
				duocDung = true
				break
			}
		}
		if !duocDung {
			nhac(s.ID, "không bước nào dùng kết quả của bước chọn đường này — "+
				"bước `model` phải khai route = \"{{steps."+s.ID+".output}}\" thì mới đi theo đường đã chọn")
		}
	}

	// Chiều ngược lại: bước `model` trỏ route vào một bước KHÔNG phải type route.
	// Chuỗi đó sẽ được thay bằng đầu ra của bước kia — thường là cả một đoạn văn
	// — rồi đem đi tra tên route. Lỗi hiện ra sẽ là "không có route này" kèm
	// nguyên đoạn văn, và nó chỉ vào sai chỗ.
	for _, s := range f.Steps {
		id := BuocTrongRoute(s.Route)
		if id == "" || coBuocRoute[id] {
			continue
		}
		if _, co := TimBuoc(f, id); !co {
			loi(s.ID, fmt.Sprintf("route trỏ tới kết quả bước %q không tồn tại", id))
			continue
		}
		loi(s.ID, fmt.Sprintf("route trỏ tới kết quả bước %q, mà bước đó không phải type = \"route\" — "+
			"đầu ra của nó là văn bản, không phải một tên đường", id))
	}
	return ps
}

// BuocTrongRoute trả về id bước mà `route` của s đang trỏ tới, hoặc "" nếu
// `route` là một tên đường khai cứng.
func BuocTrongRoute(route string) string {
	if m := conSotOutput.FindStringSubmatch(route); m != nil {
		return m[1]
	}
	return ""
}

// MoTaRoute là câu mô tả đường đi của một bước, để mọi mặt in ra giống nhau.
func MoTaRoute(s Step) string {
	switch {
	case s.Type == TypeRoute && len(s.Routes) > 0:
		return strings.Join(s.Routes, " → ")
	case s.Type == TypeRoute:
		return "(default_route rồi tới route dự phòng)"
	case s.Route != "":
		return s.Route
	default:
		return ""
	}
}
