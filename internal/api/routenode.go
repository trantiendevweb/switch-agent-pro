// Phần cắm cho node `route` của flow — xem internal/flow/route.go.
//
// File riêng chứ không nhét vào api.go: cả nội dung của nó là MỘT cây cầu, và
// cây cầu này chỉ gọi lại những hàm đã có sẵn (AIRoutes, ThuTuRoute,
// aiapi.Kiem). Không một dòng logic chọn đường nào được viết lại ở đây — nếu
// có thì `sagent route kiem`, `sagent api` và node `route` sẽ trôi khỏi nhau,
// và cái trôi ấy chỉ lộ ra vào đúng lúc một nhà cung cấp sập.
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

// ChonRoute trả về route ĐẦU TIÊN dùng được trong ungVien.
//
// ungVien rỗng = hỏi cấu hình (`default_route` rồi tới route dự phòng) qua
// ThuTuRoute — ĐÚNG danh sách mà `sagent api "câu hỏi"` sẽ thử, chứ không phải
// một danh sách dựng lại ở đây.
//
// Phép kiểm là aiapi.Kiem: `GET /models`, KHÔNG tốn token. Đây là lý do node
// này tồn tại được — chọn đường trước mà phải tiêu tiền thì chẳng ai chọn.
//
// "Dùng được" = SucKhoe.Dung(), tức route sống VÀ model khai có thật. Không hạ
// xuống thành `Song`: một route sống mà model khai sai thì bước sau vẫn hỏng,
// chỉ khác là hỏng muộn hơn và kèm một thông điệp của nhà cung cấp chẳng nhắc
// gì tới cấu hình.
func (b routeBridge) ChonRoute(ctx context.Context, ungVien []string) (flow.KetQuaRoute, error) {
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
	for _, ten := range ungVien {
		r, ok := co[ten]
		if !ok {
			ly := fmt.Sprintf("%s: không có route này trong cấu hình", ten)
			kq.NhatKy = append(kq.NhatKy, ly)
			vet = append(vet, ly)
			continue
		}
		sk := aiapi.Kiem(ctx, r)
		if sk.Dung() {
			kq.Ten = ten
			kq.NhatKy = append(kq.NhatKy, fmt.Sprintf("%s: dùng được (model %s) — CHỌN", ten, sk.Model))
			return kq, nil
		}
		ly := fmt.Sprintf("%s: %s", ten, moTaRouteHong(sk))
		kq.NhatKy = append(kq.NhatKy, ly)
		vet = append(vet, ly)
	}

	// KHÔNG trả về một cái tên đoán bừa khi hết đường. Bước sau sẽ đem cái tên
	// ấy đi gọi thật, tiêu tiền và mất thời gian vào một đường đã biết là chết,
	// rồi hỏng bằng một thông báo không liên quan gì tới nguyên nhân.
	return kq, fmt.Errorf("không đường nào trong %d ứng viên dùng được: %s",
		len(ungVien), strings.Join(vet, "; "))
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
