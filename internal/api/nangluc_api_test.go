package api

import (
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
	"github.com/trantiendevweb/switch-agent-pro/internal/config"
)

// Bảng năng lực NỬA API phải đi ra được qua hợp đồng.
//
// Cùng ý với TestNangLucCoTrongHopDong của nửa CLI: đây là điều kiện để CLI,
// dashboard và 3D cùng đọc MỘT nguồn. Khác một chỗ — ở đây là HAI hành động,
// vì đọc bảng thì miễn phí còn đo thật thì tiêu tiền, và mặt web phải cho người
// dùng thấy sự khác biệt đó TRƯỚC khi họ bấm.
func TestNangLucAPICoTrongHopDong(t *testing.T) {
	can := map[string]bool{"api.nang-luc": false, "api.nang-luc-do": false}
	for _, a := range Actions {
		if _, co := can[a]; co {
			can[a] = true
		}
	}
	for a, co := range can {
		if !co {
			t.Errorf("action %q không có trong Actions — mặt web và 3D sẽ không bao giờ "+
				"biết tính năng này tồn tại", a)
		}
	}
}

// routeCfg là ALIAS tới đúng kiểu phần tử của `config.Config.AI.Routes`.
//
// Kiểu đó khai ẩn danh ngay trong struct cấu hình nên không gọi tên được từ
// ngoài. Alias khớp từng trường KỂ CẢ THẺ `toml` là cách gọi tên nó mà không
// phải sửa gói config — và nếu ai đó đổi một trường ở đó thì file này KHÔNG
// biên dịch được nữa, tức là lệch bị bắt lúc build chứ không lúc chạy.
type routeCfg = struct {
	Ten     string `toml:"ten"`
	BaseURL string `toml:"base_url"`
	Model   string `toml:"model"`
	KeyID   string `toml:"key_id"`
}

// apiThu dựng một *API chỉ có phần cấu hình route — đủ cho bảng năng lực.
//
// Bảng này CỐ Ý không chạm store, không chạm mạng, không đọc key: nhờ vậy nó
// trả lời được ngay cả khi sổ trạng thái hỏng, đúng tính chất mà bảng năng lực
// nửa CLI đã có (xem ghi chú `&API{}` ở nangluc_test.go).
func apiThu(routes ...routeCfg) *API {
	var cfg config.Config
	cfg.AI.Routes = routes
	return &API{cfg: cfg}
}

var routeThu = routeCfg{
	Ten: "deepseek", BaseURL: "https://modelapi.vn/v1",
	Model: "deepseek-v4-flash", KeyID: "deepseek",
}

// Không truyền tên thì trả về MỌI route, mỗi route đủ MỌI năng lực, không lệch.
func TestNangLucAPITraDuMoiRouteVaMoiMuc(t *testing.T) {
	r2 := routeCfg{Ten: "grok", BaseURL: "https://modelapi.vn/v1",
		Model: "grok-4.5", KeyID: "grok"}
	ds, err := apiThu(routeThu, r2).NangLucAPI("")
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 {
		t.Fatalf("khai 2 route, báo cáo có %d", len(ds))
	}
	for _, b := range ds {
		if len(b.Lech) > 0 {
			t.Errorf("%s: bảng chọi với chính nó: %v", b.Ten, b.Lech)
		}
		co := map[string]bool{}
		for _, m := range b.Muc {
			co[m.Khoa] = true
			if m.BangChung == "" {
				t.Errorf("%s: %q không nói đo ở đâu", b.Ten, m.Khoa)
			}
		}
		for _, m := range aiapi.MoiNangLucAPI {
			if !co[m.Khoa] {
				t.Errorf("%s: báo cáo thiếu %q", b.Ten, m.Khoa)
			}
		}
	}
}

// Thứ tự phải là thứ tự CẤU HÌNH, không sắp lại theo tên.
//
// Khác `NangLuc` ở nửa CLI (sắp theo tên vì thứ tự map là ngẫu nhiên): ở đây
// thứ tự cấu hình LÀ thông tin — route đầu là `default_route`, và người đọc
// bảng cần thấy đúng thứ tự sẽ được thử khi một lượt gọi phải chuyển route.
func TestNangLucAPIGiuThuTuCauHinh(t *testing.T) {
	r2 := routeCfg{Ten: "aaa-dung-dau-bang-chu-cai", BaseURL: "https://modelapi.vn/v1",
		Model: "grok-4.5", KeyID: "grok"}
	ds, err := apiThu(routeThu, r2).NangLucAPI("")
	if err != nil {
		t.Fatal(err)
	}
	if ds[0].Ten != routeThu.Ten {
		t.Fatalf("thứ tự bị sắp lại theo tên: route đầu là %q, cấu hình khai %q trước",
			ds[0].Ten, routeThu.Ten)
	}
}

// Lọc theo route phải chạy, và tên lạ phải BÁO LỖI chứ không trả rỗng.
//
// Gõ nhầm tên route mà nhận một bảng trống thì trông y hệt "route này chưa đo
// gì cả" — hai chuyện khác hẳn nhau.
func TestNangLucAPILocVaBaoLoiTenLa(t *testing.T) {
	a := apiThu(routeThu)
	ds, err := a.NangLucAPI("deepseek")
	if err != nil || len(ds) != 1 || ds[0].Ten != "deepseek" {
		t.Fatalf("lọc theo route sai: %v / %v", ds, err)
	}
	if _, err := a.NangLucAPI("khong-co-route-nay"); err == nil {
		t.Fatal("tên route lạ phải báo lỗi, không được trả bảng rỗng")
	}
	if _, err := apiThu().NangLucAPI(""); err == nil ||
		!strings.Contains(err.Error(), "chưa cấu hình route") {
		t.Fatalf("chưa khai route nào thì phải nói ra: %v", err)
	}
}

// Phép ĐO THẬT phải từ chối tên route lạ TRƯỚC khi chạm mạng.
//
// Không kiểm phần chạy thật ở đây: nó tiêu tiền, và một bộ test tiêu tiền là bộ
// test người ta sẽ thôi chạy. Phần chạy thật đo bằng `sagent nang-luc-api --do`,
// và kết quả của nó nằm trong sổ số đo — xem TestSoSoDoSach bên internal/aiapi.
func TestNangLucAPIDoTuChoiTenLaTruocKhiChamMang(t *testing.T) {
	if _, err := apiThu(routeThu).NangLucAPIDo(nil, "khong-co-route-nay"); err == nil {
		t.Fatal("tên route lạ phải bị chặn trước khi gọi ra ngoài")
	}
	if _, err := apiThu().NangLucAPIDo(nil, ""); err == nil {
		t.Fatal("chưa khai route nào mà vẫn định đo")
	}
}
