// SOI `can` BẰNG BẢNG NĂNG LỰC — phần `flow validate` không làm được từ gói flow.
//
// ============================================================================
// ĐỂ BẢNG BIẾT MÀ KHÔNG DÙNG LÀ LÃNG PHÍ ĐẮT NHẤT CỦA CẢ ĐƯỜNG API
// ============================================================================
//
// Bảng năng lực (`aiapi.BangNangLuc`) trả lời được "route này có gọi được tool
// không" MÀ KHÔNG chạm mạng và KHÔNG tốn một đồng nào. Nghĩa là câu trả lời có
// sẵn từ lúc `sagent flow validate` — trước lượt chạy, trước bước đầu tiên,
// trước đồng token đầu tiên.
//
// Không nối vào thì hình dạng hỏng là: flow chạy, các bước trước tiêu token,
// tới bước `model` thì nhà cung cấp trả HTTP 400 "This model does not support
// image", và người đọc đi tìm nguyên nhân trong log của bốn bước trước đó.
// Trong khi cùng câu trả lời ấy đã nằm trong repo, có ngày đo, có nguyên văn
// thân lỗi, từ 22/08.
//
// ============================================================================
// VÌ SAO Ở internal/api CHỨ KHÔNG Ở internal/flow
// ============================================================================
//
// Gói `flow` không import `aiapi` (chiều phụ thuộc là api → flow), và nó cũng
// không được biết route nào đã cấu hình. Chép danh sách khoá sang bên đó là
// dựng bản thứ hai của một từ vựng — mà hai bản của một từ vựng là hai bản sẽ
// lệch, và bản lệch bao giờ cũng lệch về phía "khoá này lạ" cho một năng lực
// vừa được thêm.
//
// Nên chia đôi theo đúng thứ mỗi bên biết: `flow.VanDeCan` soi hình dạng
// (khai nhầm chỗ, trùng, rỗng, bước `model` đòi thứ bước `route` không lọc),
// còn ở đây soi NỘI DUNG (khoá có thật không, route đã khai có làm được không).
// `FlowValidate` gọi cả hai, nên mọi mặt gọi `flow.validate` đều thấy đủ.
package api

import (
	"fmt"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// VanDeCanTheoBang soi phần `can` của một flow BẰNG bảng năng lực và sổ route.
//
// `ds` rỗng (dự án chưa cấu hình route API nào) thì chỉ soi được phần khoá —
// và một bước đòi năng lực trong một dự án không có route nào là một cảnh báo
// đáng in, chứ không phải một sự im lặng.
func VanDeCanTheoBang(f flow.Flow, ds []aiapi.Route) []flow.Problem {
	var ps []flow.Problem
	loi := func(id, msg string) { ps = append(ps, flow.Problem{Flow: f.Name, Step: id, Msg: msg}) }
	nhac := func(id, msg string) {
		ps = append(ps, flow.Problem{Flow: f.Name, Step: id, Msg: msg, Warn: true})
	}

	co := map[string]aiapi.Route{}
	for _, r := range ds {
		co[r.Ten] = r
	}

	for _, s := range f.Steps {
		if len(s.Can) == 0 {
			continue
		}
		if s.Type != flow.TypeRoute && s.Type != flow.TypeModel {
			continue // đã có lỗi riêng ở flow.VanDeCan — không báo hai lần
		}

		// KHOÁ LẠ LÀ LỖI, không phải cảnh báo. Gõ `goi-tools` thay vì `goi-tool`
		// thì bộ lọc lúc chạy không biết loại ai, nên nó KHÔNG LỌC GÌ — và
		// người viết flow tin rằng bước của mình đang được canh. Một hàng rào
		// không chặn gì tệ hơn không có hàng rào.
		var sach []string
		for _, k := range s.Can {
			k = strings.TrimSpace(k)
			if k == "" {
				continue // flow.VanDeCan đã báo
			}
			if !laKhoaNangLuc(k) {
				loi(s.ID, fmt.Sprintf("`can` khai %q — không có năng lực nào tên vậy. "+
					"Bảy khoá hợp lệ: %s (xem: sagent nang-luc-api)", k, dsKhoaNangLuc()))
				continue
			}
			sach = append(sach, k)
		}
		if len(sach) == 0 {
			continue
		}

		if len(ds) == 0 {
			nhac(s.ID, fmt.Sprintf("bước này đòi %s nhưng dự án CHƯA cấu hình route API nào — "+
				"không soi được. Xem: sagent api ds", dsCan(sach)))
			continue
		}

		// Ứng viên của bước: `routes` cho node route, `route` cho node model.
		// Node `model` lấy route từ một bước `route` thì KHÔNG soi ở đây — tên
		// đường chỉ có lúc chạy; chỗ soi ca đó là cảnh báo "bước chọn đường
		// không lọc theo năng lực" trong flow.VanDeCan.
		var ung []string
		switch {
		case s.Type == flow.TypeRoute && len(s.Routes) > 0:
			ung = s.Routes
		case s.Type == flow.TypeRoute:
			continue // `routes` rỗng = do cấu hình quyết, thứ tự chỉ biết lúc chạy
		case flow.BuocTrongRoute(s.Route) != "":
			continue
		case s.Route != "":
			ung = []string{s.Route}
		default:
			continue // route rỗng = default_route, cùng lý do như trên
		}

		ps = append(ps, soiUngVien(f.Name, s.ID, ung, sach, co)...)
	}
	return ps
}

// soiUngVien dựng lỗi/cảnh báo cho MỘT bước, theo đúng ba hạng của bộ chọn.
//
// Cùng luật với routeBridge.ChonRoute và cố ý gần nhau tới mức đọc được cạnh
// nhau: nếu hai chỗ này nói khác nhau thì `flow validate` sẽ xanh cho một flow
// mà lượt chạy chặn — hoặc tệ hơn, đỏ cho một flow chạy được.
func soiUngVien(ten, id string, ung, can []string, co map[string]aiapi.Route) []flow.Problem {
	var ps []flow.Problem

	hopLe := make([]string, 0, len(ung))
	for _, r := range ung {
		r = strings.TrimSpace(r)
		if _, ok := co[r]; !ok {
			// Route lạ trong `routes` KHÔNG báo ở đây: bộ chọn lúc chạy vẫn
			// bước qua nó và thử ứng viên kế, nên đây chưa chắc là lỗi. Báo
			// thành lỗi ở đây sẽ đỏ những flow gửi cho nhau giữa hai máy có
			// project.toml khác nhau.
			continue
		}
		hopLe = append(hopLe, r)
	}
	if len(hopLe) == 0 {
		return ps
	}

	h := locNangLuc(co, hopLe, can)
	if len(h.Du) > 0 {
		return ps // có ít nhất một đường đã đo là làm được — không cần nói gì
	}

	if len(h.ChuaRo) > 0 {
		// CHƯA ĐO không phải KHÔNG LÀM ĐƯỢC — cảnh báo, không chặn. Chặn ở đây
		// là bắt người ta chạy `nang-luc-api --do` (tốn token) chỉ để `validate`
		// hết đỏ, và họ sẽ chạy nó cho có.
		ps = append(ps, flow.Problem{Flow: ten, Step: id, Warn: true,
			Msg: fmt.Sprintf("bước này đòi %s, và KHÔNG ứng viên nào đã được đo là làm được. "+
				"Sẽ chạy bằng %q (chưa đo — %s). Đo trước cho chắc: sagent nang-luc-api --do %s",
				dsCan(can), h.ChuaRo[0], h.Vi[h.ChuaRo[0]], h.ChuaRo[0])})
		return ps
	}

	// Mọi ứng viên đều ĐÃ ĐO ĐƯỢC là không làm được. Đây là lỗi CỨNG, và là chỗ
	// đắt giá nhất của cả file: nó chặn một lượt chạy sẽ hỏng, tại thời điểm
	// chưa tiêu một đồng nào.
	var vi []string
	for _, r := range h.Loai {
		vi = append(vi, fmt.Sprintf("%s: %s", r, h.Vi[r]))
	}
	ps = append(ps, flow.Problem{Flow: ten, Step: id,
		Msg: fmt.Sprintf("KHÔNG ứng viên nào làm được %s — lượt chạy sẽ dừng ở bước này. %s. "+
			"Sửa: bỏ khoá đó khỏi `can`, thêm một route làm được, hoặc — nếu số đo đã cũ — "+
			"đo lại: sagent nang-luc-api --do %s",
			dsCan(can), strings.Join(vi, " · "), h.Loai[0])})
	return ps
}

// laKhoaNangLuc hỏi thẳng `aiapi.MoiNangLucAPI` — danh sách CHÍNH THỨC, không
// có bản chép thứ hai ở đâu trong dự án.
func laKhoaNangLuc(k string) bool {
	for _, m := range aiapi.MoiNangLucAPI {
		if m.Khoa == k {
			return true
		}
	}
	return false
}

func dsKhoaNangLuc() string {
	ks := make([]string, 0, len(aiapi.MoiNangLucAPI))
	for _, m := range aiapi.MoiNangLucAPI {
		ks = append(ks, m.Khoa)
	}
	return strings.Join(ks, ", ")
}
