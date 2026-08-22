// Phần cắm cho node `route` của flow — xem internal/flow/route.go.
//
// File riêng chứ không nhét vào api.go: cả nội dung của nó là MỘT cây cầu, và
// cây cầu này chỉ gọi lại những hàm đã có sẵn (AIRoutes, ThuTuRoute,
// aiapi.Kiem, aiapi.BangNangLuc). Không một dòng logic chọn đường nào được viết
// lại ở đây — nếu có thì `sagent route kiem`, `sagent api`, `sagent nang-luc-api`
// và node `route` sẽ trôi khỏi nhau, và cái trôi ấy chỉ lộ ra vào đúng lúc một
// nhà cung cấp sập.
//
// ============================================================================
// HAI PHÉP LỌC, VÀ THỨ TỰ GIỮA CHÚNG LÀ MỘT QUYẾT ĐỊNH
// ============================================================================
//
//	NĂNG LỰC — route này có gọi được tool / đọc được ảnh không? Trả lời bằng
//	`aiapi.BangNangLuc`: đọc bảng, KHÔNG chạm mạng, KHÔNG tốn gì.
//
//	SỨC KHOẺ — route này có đang sống không? Trả lời bằng `aiapi.Kiem`:
//	`GET /models`, không tốn token nhưng tốn một lượt đi mạng CHO MỖI ứng viên.
//
// Năng lực chạy TRƯỚC. Ngược lại thì mỗi lần chọn đường sẽ hỏi thăm sức khoẻ
// của những route mà bảng đã biết trước là không làm được việc — trả tiền bằng
// thời gian cho một câu trả lời đã nằm sẵn trong repo.
//
// ============================================================================
// BA HẠNG, KHÔNG PHẢI HAI — VÀ ĐÓ LÀ CẢ LÝ DO BẢNG NĂNG LỰC CÓ BA TRẠNG THÁI
// ============================================================================
//
//	ĐỦ     — mọi khoá `can` đều ĐÃ ĐO và làm được. Ưu tiên tuyệt đối.
//	CHƯA RÕ — không khoá nào bị đo là KHÔNG, nhưng có khoá chưa ai đo.
//	LOẠI   — có ít nhất một khoá ĐÃ ĐO ĐƯỢC là không làm được.
//
// Gộp CHƯA RÕ vào LOẠI là bẹp ba trạng thái thành hai, đúng cái sai mà cả
// internal/aiapi/nangluc.go dựng lên để chặn: "chưa ai đo" KHÔNG phải "không
// làm được". Một dự án vừa thêm route thứ ba sẽ thấy nó bị loại thẳng chỉ vì
// chưa chạy `nang-luc-api --do`, và lỗi hiện ra không nhắc gì tới phép đo.
//
// Gộp CHƯA RÕ vào ĐỦ thì hỏng theo chiều ngược lại: một route chưa đo được
// chọn ngang hàng với route đã đo xong, lượt chạy hỏng ở bước sau mà bảng vẫn
// xanh.
//
// Nên: thử hạng ĐỦ trước; hết đường mới xuống hạng CHƯA RÕ, và lúc xuống thì
// NÓI RA trong nhật ký — cùng luật và gần như cùng câu chữ với
// `aiapi.SoatTruocKhiGui`, để hai chỗ không dạy người dùng hai điều khác nhau.
package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// routeBridge nối node `route` của flow sang sổ route + phép kiểm sức khoẻ.
type routeBridge struct{ a *API }

// hangUngVien là kết quả chia hạng theo bảng năng lực. Xem khối chú thích đầu file.
type hangUngVien struct {
	Du     []string // mọi khoá `can` đều đã đo và làm được
	ChuaRo []string // không khoá nào bị đo là KHÔNG, nhưng có khoá chưa ai đo
	Loai   []string // có khoá ĐÃ ĐO ĐƯỢC là không làm được — không bao giờ chọn

	// Vi giữ MỘT câu vì sao cho mỗi tên trong ChuaRo và Loai. Bằng chứng đi kèm
	// NGUYÊN VĂN từ bảng, không rút gọn: người đọc phải biết được số đo ngày nào
	// và quan sát ra sao, nếu không thì lời từ chối chỉ là một lời khẳng định.
	Vi map[string]string
}

// locNangLuc chia ứng viên thành ba hạng. THUẦN: không chạm mạng, không đọc key.
//
// Tách khỏi ChonRoute có chủ ý — đây là phần duy nhất kiểm được mà không dựng
// máy chủ giả, nên nó là chỗ bài kiểm hỏi thẳng SỔ SỐ ĐO THẬT (modelapi.vn ·
// grok-4.5 / deepseek-v4-flash) chứ không hỏi một bảng dựng cho vừa bài kiểm.
func locNangLuc(co map[string]aiapi.Route, ungVien, can []string) hangUngVien {
	h := hangUngVien{Vi: map[string]string{}}
	for _, ten := range ungVien {
		r, ok := co[ten]
		if !ok {
			continue // ca "không có route này" do ChonRoute xử, trước cả phép lọc
		}
		if len(can) == 0 {
			h.Du = append(h.Du, ten)
			continue
		}

		trang := map[string]aiapi.NangLucAPI{}
		for _, m := range aiapi.BangNangLuc(r).Muc {
			trang[m.Khoa] = m
		}

		var khong, chuaDo []string
		for _, k := range can {
			m, biet := trang[k]
			if !biet {
				// Khoá lạ. KHÔNG được lặng lẽ bỏ qua: bỏ qua nghĩa là bộ lọc
				// không lọc gì trong khi người dùng tưởng nó đang canh.
				// `flow validate` chặn ca này từ trước (xem flow_nangluc.go),
				// nên tới được đây là flow chạy thẳng không qua validate — vẫn
				// phải nói ra, và nói theo hướng an toàn: chưa rõ, không phải đủ.
				chuaDo = append(chuaDo, fmt.Sprintf("%s: không có năng lực này trong bảng", k))
				continue
			}
			switch m.TrangThai {
			case aiapi.LamDuoc:
			case aiapi.KhongLamDuoc:
				khong = append(khong, fmt.Sprintf("%s: %s", k, m.BangChung))
			default:
				chuaDo = append(chuaDo, fmt.Sprintf("%s: %s", k, m.BangChung))
			}
		}

		switch {
		case len(khong) > 0:
			h.Loai = append(h.Loai, ten)
			h.Vi[ten] = "KHÔNG làm được — " + strings.Join(khong, "; ")
		case len(chuaDo) > 0:
			h.ChuaRo = append(h.ChuaRo, ten)
			h.Vi[ten] = "CHƯA ai đo — " + strings.Join(chuaDo, "; ")
		default:
			h.Du = append(h.Du, ten)
		}
	}
	return h
}

// ChonRoute trả về route ĐẦU TIÊN dùng được trong ungVien.
//
// ungVien rỗng = hỏi cấu hình (`default_route` rồi tới route dự phòng) qua
// ThuTuRoute — ĐÚNG danh sách mà `sagent api "câu hỏi"` sẽ thử, chứ không phải
// một danh sách dựng lại ở đây.
//
// `can` rỗng = giữ nguyên hành vi cũ: chọn theo sức khoẻ, theo thứ tự ứng viên.
//
// Phép kiểm sức khoẻ là aiapi.Kiem: `GET /models`, KHÔNG tốn token. Đây là lý do
// node này tồn tại được — chọn đường trước mà phải tiêu tiền thì chẳng ai chọn.
//
// "Dùng được" = SucKhoe.Dung(), tức route sống VÀ model khai có thật. Không hạ
// xuống thành `Song`: một route sống mà model khai sai thì bước sau vẫn hỏng,
// chỉ khác là hỏng muộn hơn và kèm một thông điệp của nhà cung cấp chẳng nhắc
// gì tới cấu hình.
func (b routeBridge) ChonRoute(ctx context.Context, ungVien, can []string) (flow.KetQuaRoute, error) {
	if len(ungVien) == 0 {
		ungVien = b.a.ThuTuRoute("")
	}
	if len(ungVien) == 0 {
		return flow.KetQuaRoute{}, fmt.Errorf("chưa cấu hình route nào — xem: sagent api ds")
	}

	co := map[string]aiapi.Route{}
	for _, r := range b.a.AIRoutes() {
		co[r.Ten] = r
	}

	var kq flow.KetQuaRoute
	var vet []string // lý do từng đường bị loại, để gộp vào lỗi cuối
	ghi := func(ly string) {
		kq.NhatKy = append(kq.NhatKy, ly)
		vet = append(vet, ly)
	}

	// Ứng viên không có trong cấu hình bị loại TRƯỚC mọi phép lọc: đó là lỗi
	// đánh máy trong flows.toml, không phải một kết luận về năng lực.
	hopLe := make([]string, 0, len(ungVien))
	for _, ten := range ungVien {
		if _, ok := co[ten]; !ok {
			ghi(fmt.Sprintf("%s: không có route này trong cấu hình", ten))
			continue
		}
		hopLe = append(hopLe, ten)
	}

	// LỌC NĂNG LỰC — miễn phí, chạy trước phép hỏi thăm sức khoẻ.
	h := locNangLuc(co, hopLe, can)
	loai := map[string]bool{}
	for _, ten := range h.Loai {
		loai[ten] = true
		ghi(fmt.Sprintf("%s: %s", ten, h.Vi[ten]))
	}

	// Hạng ĐỦ trước, hạng CHƯA RÕ sau. Hai vòng riêng chứ không một vòng có cờ:
	// vòng thứ hai phải nói ra rằng nó đang chạy trên một route CHƯA ĐO, và câu
	// đó chỉ đúng ở vòng thứ hai.
	for _, ten := range h.Du {
		sk := aiapi.Kiem(ctx, co[ten])
		if sk.Dung() {
			kq.Ten = ten
			kq.NhatKy = append(kq.NhatKy, fmt.Sprintf("%s: dùng được (model %s)%s — CHỌN",
				ten, sk.Model, duNangLuc(can)))
			return kq, nil
		}
		ghi(fmt.Sprintf("%s: %s", ten, moTaRouteHong(sk)))
	}
	for _, ten := range h.ChuaRo {
		sk := aiapi.Kiem(ctx, co[ten])
		if sk.Dung() {
			kq.Ten = ten
			// Nói ra, MỖI LẦN, kể cả khi lượt chạy xanh. Đây là ca "chạy bằng
			// route tốt nhất có thể" — và nếu nó đi qua im lặng thì lần sau
			// không ai nhớ rằng phép đo vẫn còn thiếu.
			kq.NhatKy = append(kq.NhatKy, fmt.Sprintf("%s: dùng được (model %s) — CHỌN, nhưng "+
				"CHƯA ai đo route này có %s hay không (%s). \"Chưa đo\" KHÔNG phải \"không làm "+
				"được\", nên vẫn đi — bước sau hỏng thì đo lại bằng: sagent nang-luc-api --do %s",
				ten, sk.Model, dsCan(can), h.Vi[ten], ten))
			return kq, nil
		}
		ghi(fmt.Sprintf("%s: %s", ten, moTaRouteHong(sk)))
	}

	// KHÔNG trả về một cái tên đoán bừa khi hết đường. Bước sau sẽ đem cái tên
	// ấy đi gọi thật, tiêu tiền và mất thời gian vào một đường đã biết là chết,
	// rồi hỏng bằng một thông báo không liên quan gì tới nguyên nhân.
	//
	// Ca ĐÁNG NÓI RIÊNG: hết đường vì KHÔNG ROUTE NÀO ĐỦ NĂNG LỰC. Lúc này bảng
	// đã biết câu trả lời trước cả lượt đi mạng đầu tiên, nên lỗi phải nói đúng
	// việc phải làm — đổi route, hay đo lại — chứ không phải một câu tổng kết về
	// sức khoẻ, vì sức khoẻ chưa hề được hỏi tới.
	if len(can) > 0 && len(h.Du) == 0 && len(h.ChuaRo) == 0 && len(h.Loai) > 0 {
		return kq, fmt.Errorf("không route nào trong %d ứng viên làm được %s — DỪNG ở đây, "+
			"CHƯA gọi đi đâu cả: bảng năng lực đã biết câu trả lời trước khi gọi, và một "+
			"lượt hỏng vẫn có thể bị tính tiền.\n     %s\n     Sửa: bỏ khoá đó khỏi `can`, "+
			"thêm một route làm được vào `routes`, hoặc — nếu số đo đã cũ — đo lại: "+
			"sagent nang-luc-api --do %s",
			len(hopLe), dsCan(can), strings.Join(vet, "\n     "), h.Loai[0])
	}
	return kq, fmt.Errorf("không đường nào trong %d ứng viên dùng được: %s",
		len(ungVien), strings.Join(vet, "; "))
}

// duNangLuc là mẩu chữ thêm vào dòng CHỌN khi bước có đòi năng lực. Rỗng khi
// không đòi gì — không in một câu "đã đo là làm được" trống trơn cho một bước
// chưa hỏi gì cả.
func duNangLuc(can []string) string {
	if len(can) == 0 {
		return ""
	}
	return ", đã đo là làm được " + dsCan(can)
}

func dsCan(can []string) string {
	if len(can) == 0 {
		return "(không đòi năng lực nào)"
	}
	return "`" + strings.Join(can, "`, `") + "`"
}

// moTaRouteHong biến một bản khám thành MỘT câu nói được vì sao loại.
//
// Ba câu khác nhau vì ba việc phải làm khác nhau: không chạm được mạng thì xem
// mạng/base_url, key bị từ chối thì đăng nhập lại, model khai sai thì sửa
// project.toml. Gộp cả ba thành "route hỏng" là bắt người đọc tự đi đoán.
func moTaRouteHong(sk aiapi.SucKhoe) string {
	switch {
	case !sk.Song && sk.Status == 0:
		return "không chạm được tới nhà cung cấp" + kemLoi(sk.Loi)
	case !sk.Song:
		return fmt.Sprintf("HTTP %d%s", sk.Status, kemLoi(sk.Loi))
	case !sk.CoModel && !sk.KhongRo:
		s := fmt.Sprintf("route sống nhưng model %q không có trong %d model nhà cung cấp liệt kê",
			sk.Model, sk.SoModel)
		if len(sk.Gan) > 0 {
			s += " (gần giống: " + strings.Join(sk.Gan, ", ") + ")"
		}
		return s
	default:
		return "không dùng được" + kemLoi(sk.Loi)
	}
}

func kemLoi(l string) string {
	if l == "" {
		return ""
	}
	return " — " + l
}
