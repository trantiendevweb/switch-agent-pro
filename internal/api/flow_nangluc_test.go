package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// ============================================================================
// PHÉP LỌC NĂNG LỰC HỎI THẲNG SỔ SỐ ĐO THẬT, KHÔNG HỎI MỘT BẢNG DỰNG CHO VỪA
// ============================================================================
//
// Hai route dưới đây là hai route THẬT của dự án, và cặp (base_url, model) của
// chúng khớp đúng những dòng đã đo ngày 22/08 trong `aiapi.soDoNangLuc`:
//
//	grok-4.5          · dau-vao-anh = LÀM ĐƯỢC   (đọc đúng màu ảnh PNG 32x32, 761 token)
//	deepseek-v4-flash · dau-vao-anh = KHÔNG      (HTTP 400 "This model does not support image")
//
// Dựng một bảng giả ở đây thì bài kiểm sẽ xanh mãi mãi kể cả khi phép tra bảng
// hỏng — nó chỉ đo cái bảng giả. Hỏi bảng thật thì bài này còn canh thêm một
// thứ nữa: đường từ `Route` tới `BangNangLuc` tới `TrangThai` phải còn thông.
var (
	rGrok = aiapi.Route{Ten: "grok", BaseURL: "https://modelapi.vn/v1",
		Model: "grok-4.5", KeyID: "grok"}
	rDeep = aiapi.Route{Ten: "deepseek", BaseURL: "https://modelapi.vn/v1",
		Model: "deepseek-v4-flash", KeyID: "deepseek"}
	// Route CHƯA AI ĐO: model không có dòng nào trong sổ số đo.
	rLa = aiapi.Route{Ten: "la", BaseURL: "https://modelapi.vn/v1",
		Model: "chua-ai-do-model-nay", KeyID: "grok"}
)

func soRoute(ds ...aiapi.Route) map[string]aiapi.Route {
	m := map[string]aiapi.Route{}
	for _, r := range ds {
		m[r.Ten] = r
	}
	return m
}

// Đo được là KHÔNG → hạng LOẠI. Đo được là CÓ → hạng ĐỦ. Không lẫn hai hạng.
func TestLocNangLucChiaDungTheoSoDoThat(t *testing.T) {
	h := locNangLuc(soRoute(rDeep, rGrok), []string{"deepseek", "grok"},
		[]string{"dau-vao-anh"})

	if len(h.Du) != 1 || h.Du[0] != "grok" {
		t.Fatalf("hạng ĐỦ = %v, chờ [grok] — số đo 22/08 nói grok-4.5 đọc được ảnh", h.Du)
	}
	if len(h.Loai) != 1 || h.Loai[0] != "deepseek" {
		t.Fatalf("hạng LOẠI = %v, chờ [deepseek] — số đo 22/08 nói deepseek-v4-flash "+
			"trả HTTP 400 \"This model does not support image\"", h.Loai)
	}
	if len(h.ChuaRo) != 0 {
		t.Fatalf("hạng CHƯA RÕ = %v, phải rỗng: cả hai ô đều đã có số đo", h.ChuaRo)
	}
	// Lý do phải mang NGUYÊN VĂN quan sát, không rút thành "không làm được".
	// Người đọc cần biết số đo ngày nào và nhà cung cấp nói gì, nếu không thì
	// lời từ chối chỉ là một lời khẳng định.
	if vi := h.Vi["deepseek"]; !strings.Contains(vi, "does not support image") {
		t.Errorf("lý do loại deepseek mất bằng chứng gốc: %q", vi)
	}
}

// CHƯA ĐO KHÔNG PHẢI KHÔNG LÀM ĐƯỢC — đây là bài kiểm giữ cho ba trạng thái
// không bị bẹp thành hai.
//
// Bẹp về phía LOẠI: một dự án vừa thêm route thứ ba thấy nó bị loại thẳng vì
// chưa chạy `nang-luc-api --do`, và lỗi hiện ra không nhắc gì tới phép đo.
// Bẹp về phía ĐỦ: route chưa đo được chọn ngang hàng route đã đo, lượt chạy
// hỏng ở bước sau mà bảng vẫn xanh.
func TestChuaDoLaHangRiengKhongPhaiLoai(t *testing.T) {
	h := locNangLuc(soRoute(rLa), []string{"la"}, []string{"dau-vao-anh"})
	if len(h.Loai) != 0 {
		t.Fatalf("route CHƯA AI ĐO bị xếp vào hạng LOẠI (%v) — \"chưa đo\" đã bị đọc "+
			"thành \"không làm được\", đúng cái sai mà bảng ba trạng thái dựng lên để chặn",
			h.Loai)
	}
	if len(h.Du) != 0 {
		t.Fatalf("route CHƯA AI ĐO bị xếp vào hạng ĐỦ (%v) — nó sẽ được chọn ngang hàng "+
			"với route đã đo, và lượt chạy hỏng ở bước sau mà bảng vẫn xanh", h.Du)
	}
	if len(h.ChuaRo) != 1 || h.ChuaRo[0] != "la" {
		t.Fatalf("hạng CHƯA RÕ = %v, chờ [la]", h.ChuaRo)
	}
}

// Khoá LẠ đi về phía CHƯA RÕ, không đi về phía ĐỦ.
//
// `flow validate` chặn khoá lạ từ trước, nhưng flow chạy thẳng không qua
// validate thì tới được đây. Xếp vào ĐỦ nghĩa là bộ lọc lặng lẽ không lọc gì
// trong khi người dùng tin rằng bước của họ đang được canh — một hàng rào
// không chặn gì tệ hơn không có hàng rào.
func TestKhoaLaKhongDuocCoiLaDuNangLuc(t *testing.T) {
	h := locNangLuc(soRoute(rGrok), []string{"grok"}, []string{"goi-tools-go-nham"})
	if len(h.Du) != 0 {
		t.Fatalf("khoá lạ được coi là đủ năng lực (%v) — bộ lọc không lọc gì mà "+
			"không ai biết", h.Du)
	}
	if len(h.ChuaRo) != 1 {
		t.Fatalf("hạng CHƯA RÕ = %v, chờ [grok]", h.ChuaRo)
	}
}

// Không khai `can` thì KHÔNG route nào bị loại — hợp đồng tương thích ngược.
func TestKhongCanThiKhongLocGi(t *testing.T) {
	h := locNangLuc(soRoute(rDeep, rGrok), []string{"deepseek", "grok"}, nil)
	if len(h.Du) != 2 || len(h.Loai) != 0 || len(h.ChuaRo) != 0 {
		t.Fatalf("bước không đòi năng lực mà vẫn bị lọc: đủ=%v loại=%v chưa rõ=%v",
			h.Du, h.Loai, h.ChuaRo)
	}
}

// ============================================================================
// BỘ CHỌN CHẠY THẬT — máy chủ giả, KHÔNG chạm modelapi.vn, KHÔNG tốn token
// ============================================================================

// srvModels dựng một máy chủ trả `GET /models` chứa đúng những model cho trước.
// Danh sách rỗng = máy chủ trả HTTP 500, tức route "chết".
func srvModels(t *testing.T, models ...string) string {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(models) == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var b strings.Builder
		b.WriteString(`{"data":[`)
		for i, m := range models {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(`{"id":"` + m + `"}`)
		}
		b.WriteString(`]}`)
		_, _ = w.Write([]byte(b.String()))
	}))
	t.Cleanup(s.Close)
	return s.URL
}

// khoaGia đặt một file key giả trong kho tạm để aiapi.Kiem có cái mà gửi đi.
func khoaGia(t *testing.T, id string) {
	t.Helper()
	dir := aiapi.KeysDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".key"), []byte("sk-gia"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ĐÂY LÀ BÀI KIỂM QUAN TRỌNG NHẤT CỦA BỘ CHỌN: NĂNG LỰC THẮNG THỨ TỰ ƯU TIÊN.
//
// `deepseek` đứng ĐẦU danh sách ứng viên — trước mảnh này thì nó được chọn, vì
// bộ chọn cũ chỉ hỏi "còn sống không". Bảng thì đã đo được từ 22/08 rằng
// deepseek-v4-flash trả HTTP 400 "This model does not support image".
//
// Bài này khẳng định ba điều, và cả ba đều là điều mảnh này sinh ra để làm:
//
//  1. deepseek bị loại DÙ đứng đầu — năng lực thắng thứ tự;
//  2. lý do loại mang NGUYÊN VĂN quan sát của phép đo, không rút gọn;
//  3. deepseek KHÔNG bị hỏi thăm sức khoẻ — base_url của nó là modelapi.vn
//     thật, nên nếu bộ chọn hỏi sức khoẻ trước khi lọc năng lực thì bài này đi
//     ra Internet và lời từ chối sẽ mang chữ của một lượt HTTP.
//
// Route thứ hai trỏ vào máy chủ giả nên nó ở hạng CHƯA RÕ — và đó cũng là điều
// đáng khẳng định: hạng CHƯA RÕ vẫn được chọn, chứ không bị loại theo.
func TestChonRouteBoQuaDuongKhongDuNangLucDuNoDungDau(t *testing.T) {
	khoTam(t)
	khoaGia(t, "deepseek")
	khoaGia(t, "thu")
	a := apiThu(
		// (base_url, model) THẬT — khớp đúng dòng đã đo trong aiapi.soDoNangLuc.
		routeCfg{Ten: "deepseek", BaseURL: "https://modelapi.vn/v1",
			Model: "deepseek-v4-flash", KeyID: "deepseek"},
		// Máy chủ giả: sống thật, nhưng chưa ai đo năng lực của nó.
		routeCfg{Ten: "thu", BaseURL: srvModels(t, "m"), Model: "m", KeyID: "thu"},
	)

	kq, err := routeBridge{a}.ChonRoute(context.Background(),
		[]string{"deepseek", "thu"}, []string{"dau-vao-anh"})
	if err != nil {
		t.Fatal(err)
	}
	if kq.Ten != "thu" {
		t.Fatalf("chọn %q — deepseek đứng đầu danh sách và bảng ĐÃ ĐO được là nó không "+
			"đọc được ảnh, mà bộ chọn vẫn lấy nó. Bước sau sẽ gửi ảnh đi và tiêu tiền "+
			"cho một câu HTTP 400 đã nằm sẵn trong repo từ 22/08.", kq.Ten)
	}

	nk := strings.Join(kq.NhatKy, "\n")
	if !strings.Contains(nk, "does not support image") {
		t.Errorf("nhật ký không nói vì sao loại deepseek, kèm bằng chứng gốc:\n%s", nk)
	}
	// Điều 3: không lượt hỏi thăm sức khoẻ nào tới modelapi.vn. Nếu có thì lý
	// do loại deepseek sẽ là một câu về mạng/HTTP chứ không phải câu của bảng.
	for _, xau := range []string{"không chạm được tới nhà cung cấp", "HTTP 401", "HTTP 403"} {
		if strings.Contains(nk, xau) {
			t.Errorf("bộ chọn đã hỏi thăm sức khoẻ deepseek TRƯỚC khi lọc năng lực (%q) — "+
				"trả tiền bằng thời gian cho một câu trả lời đã nằm sẵn trong bảng:\n%s", xau, nk)
		}
	}
	// Chọn một route CHƯA ĐO thì phải NÓI RA, mỗi lần, kể cả khi lượt chạy xanh.
	// Đi qua im lặng thì lần sau không ai nhớ rằng phép đo vẫn còn thiếu.
	if !strings.Contains(nk, "CHƯA ai đo") {
		t.Fatalf("chọn một route chưa đo mà nhật ký không nói gì:\n%s", nk)
	}
	if !strings.Contains(nk, "nang-luc-api --do") {
		t.Errorf("nhật ký không nói cách đóng ô chưa đo:\n%s", nk)
	}
}

// Bảng ĐÃ ĐO ĐƯỢC là không làm được, và không còn ứng viên nào khác → DỪNG.
//
// ĐÂY LÀ CÂU TRẢ LỜI CHO "KHÔNG ROUTE NÀO ĐỦ NĂNG LỰC THÌ SAO", và nó là DỪNG
// chứ không phải "chạy bằng route tốt nhất rồi cảnh báo". Lý do: chạy tiếp
// nghĩa là gọi thật, tiêu token, và nhận về một câu trả lời KHÔNG có tool call —
// thứ trông y hệt một câu trả lời bình thường. Đó đúng hình dạng hỏng mà
// `phai_co` được dựng ra để chặn: cổng kiểm không sập, nó chỉ lặng lẽ gật đầu.
//
// Và ở đây bảng đã biết câu trả lời TRƯỚC cả lượt đi mạng đầu tiên — để nó biết
// mà không dùng là lãng phí đắt nhất của cả đường API.
func TestKhongRouteNaoDuNangLucThiDungHan(t *testing.T) {
	khoTam(t)
	khoaGia(t, "deepseek")
	a := apiThu(routeCfg{Ten: "deepseek", BaseURL: "https://modelapi.vn/v1",
		Model: "deepseek-v4-flash", KeyID: "deepseek"})

	kq, err := routeBridge{a}.ChonRoute(context.Background(),
		[]string{"deepseek"}, []string{"dau-vao-anh"})
	if err == nil {
		t.Fatalf("chọn được %q trong khi bảng đã đo được là route đó KHÔNG đọc được ảnh — "+
			"bước sau sẽ gửi ảnh đi và tiêu tiền cho một câu HTTP 400 đã biết trước", kq.Ten)
	}
	m := err.Error()
	for _, phai := range []string{
		"dau-vao-anh",            // đòi gì
		"does not support image", // bằng chứng NGUYÊN VĂN, có ngày đo
		"CHƯA gọi đi đâu cả",     // nói rõ chưa tiêu đồng nào
		"nang-luc-api --do",      // việc phải làm nếu số đo đã cũ
	} {
		if !strings.Contains(m, phai) {
			t.Errorf("lời từ chối thiếu %q:\n%s", phai, m)
		}
	}
	// Và nó KHÔNG được chạm mạng: base_url là modelapi.vn thật, nên nếu bộ chọn
	// hỏi thăm sức khoẻ trước khi lọc năng lực thì bài này sẽ đi ra Internet.
	// Lời từ chối không được nhắc gì tới HTTP của lượt kiểm.
	if strings.Contains(m, "không chạm được tới nhà cung cấp") {
		t.Error("bộ chọn đã hỏi thăm sức khoẻ TRƯỚC khi lọc năng lực — trả tiền bằng " +
			"thời gian cho một câu trả lời đã nằm sẵn trong bảng")
	}
}

// Không khai `can` thì bộ chọn giữ nguyên hành vi cũ: đường đầu tiên còn sống.
func TestKhongCanThiChonNhuCu(t *testing.T) {
	khoTam(t)
	khoaGia(t, "grok")
	khoaGia(t, "deepseek")
	chet := srvModels(t)                      // trả 500
	song := srvModels(t, "deepseek-v4-flash") // sống
	a := apiThu(
		routeCfg{Ten: "grok", BaseURL: chet, Model: "grok-4.5", KeyID: "grok"},
		routeCfg{Ten: "deepseek", BaseURL: song, Model: "deepseek-v4-flash", KeyID: "deepseek"},
	)
	kq, err := routeBridge{a}.ChonRoute(context.Background(), []string{"grok", "deepseek"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if kq.Ten != "deepseek" {
		t.Fatalf("chọn %q, chờ deepseek (grok trả HTTP 500)", kq.Ten)
	}
}

// ============================================================================
// SOI LÚC `flow validate` — trước lượt chạy, trước đồng token đầu tiên
// ============================================================================

// Khoá gõ nhầm phải là LỖI, và lời báo phải in ra bảy khoá hợp lệ.
//
// Không in danh sách thì người đọc phải đi mở mã nguồn để biết mình gõ nhầm chữ
// nào — mà cả điểm của mảnh này là trả lời được câu hỏi mà không phải chạy.
func TestValidateBatKhoaCanGoNham(t *testing.T) {
	f := flow.Flow{Name: "x", Steps: []flow.Step{
		{ID: "chon", Type: flow.TypeRoute, Routes: []string{"grok"}, Can: []string{"goi-tools"}},
	}}
	ps := VanDeCanTheoBang(f, []aiapi.Route{rGrok})
	if len(ps) != 1 || ps[0].Warn {
		t.Fatalf("chờ ĐÚNG một LỖI, được %+v", ps)
	}
	if !strings.Contains(ps[0].Msg, "goi-tool,") && !strings.Contains(ps[0].Msg, "goi-tool ") {
		t.Errorf("lời báo không liệt kê khoá hợp lệ: %s", ps[0].Msg)
	}
}

// Route KHAI CỨNG mà bảng đã đo được là không làm được → LỖI lúc validate.
//
// Đây là ô đắt nhất của cả file: nó chặn một lượt chạy chắc chắn hỏng, tại thời
// điểm chưa tiêu một đồng nào. Không có nó thì hình dạng hỏng là: flow chạy,
// bốn bước trước tiêu token, bước `model` ăn HTTP 400 "This model does not
// support image", và người đọc đi tìm nguyên nhân trong log của bốn bước kia.
func TestValidateChanBuocModelTroVaoRouteDaDoLaKhong(t *testing.T) {
	f := flow.Flow{Name: "x", Steps: []flow.Step{
		{ID: "hoi", Type: flow.TypeModel, Route: "deepseek",
			Can: []string{"dau-vao-anh"}, Prompt: "ảnh này màu gì"},
	}}
	ps := VanDeCanTheoBang(f, []aiapi.Route{rDeep, rGrok})
	if len(ps) != 1 || ps[0].Warn {
		t.Fatalf("chờ ĐÚNG một LỖI, được %+v", ps)
	}
	if !strings.Contains(ps[0].Msg, "does not support image") {
		t.Errorf("lời chặn mất bằng chứng gốc: %s", ps[0].Msg)
	}
}

// Có ÍT NHẤT MỘT ứng viên làm được thì im lặng — bộ chọn sẽ tìm ra nó lúc chạy.
func TestValidateImLangKhiConMotDuongLamDuoc(t *testing.T) {
	f := flow.Flow{Name: "x", Steps: []flow.Step{
		{ID: "chon", Type: flow.TypeRoute, Routes: []string{"deepseek", "grok"},
			Can: []string{"dau-vao-anh"}},
	}}
	if ps := VanDeCanTheoBang(f, []aiapi.Route{rDeep, rGrok}); len(ps) != 0 {
		t.Fatalf("còn grok làm được mà vẫn kêu: %+v", ps)
	}
}

// CHƯA ĐO chỉ là CẢNH BÁO, không chặn.
//
// Chặn ở đây là bắt người ta chạy `nang-luc-api --do` (tốn token) chỉ để cho
// `validate` hết đỏ — và họ sẽ chạy nó cho có, đúng kiểu phép đo dựng lên để
// làm hài lòng một bộ kiểm chứ không phải để biết sự thật.
func TestValidateChuaDoChiLaCanhBao(t *testing.T) {
	f := flow.Flow{Name: "x", Steps: []flow.Step{
		{ID: "chon", Type: flow.TypeRoute, Routes: []string{"la"}, Can: []string{"dau-vao-anh"}},
	}}
	ps := VanDeCanTheoBang(f, []aiapi.Route{rLa})
	if len(ps) != 1 || !ps[0].Warn {
		t.Fatalf("chờ ĐÚNG một CẢNH BÁO (không phải lỗi), được %+v", ps)
	}
	if !strings.Contains(ps[0].Msg, "nang-luc-api --do") {
		t.Errorf("cảnh báo không nói cách đóng ô chưa đo: %s", ps[0].Msg)
	}
}

// Dự án chưa cấu hình route nào mà bước vẫn đòi năng lực → nói ra, đừng im.
func TestValidateNoiRaKhiChuaCoRouteNao(t *testing.T) {
	f := flow.Flow{Name: "x", Steps: []flow.Step{
		{ID: "chon", Type: flow.TypeRoute, Routes: []string{"grok"}, Can: []string{"goi-tool"}},
	}}
	ps := VanDeCanTheoBang(f, nil)
	if len(ps) != 1 || !ps[0].Warn {
		t.Fatalf("chờ ĐÚNG một cảnh báo, được %+v", ps)
	}
}

// Bước KHÔNG khai `can` thì phần soi này không được nói một câu nào.
func TestValidateKhongDungToiBuocKhongKhaiCan(t *testing.T) {
	f := flow.Flow{Name: "x", Steps: []flow.Step{
		{ID: "hoi", Type: flow.TypeModel, Route: "deepseek", Prompt: "?"},
		{ID: "chay", Type: flow.TypeShell, Run: []string{"go", "version"}},
	}}
	if ps := VanDeCanTheoBang(f, []aiapi.Route{rDeep}); len(ps) != 0 {
		t.Fatalf("kêu về bước không khai `can`: %+v", ps)
	}
}

// ============================================================================
// MẮT XÍCH CUỐI: PHẢI ĐI QUA ĐÚNG ACTION `flow.validate`
// ============================================================================
//
// Mọi bài trên gọi thẳng `VanDeCanTheoBang`. Một hàm đúng mà không ai gọi là
// đúng hình dạng lỗi dự án đã vấp SÁU lần trong ngày 22/08: thứ gì đó "có" ở
// mọi tầng TRỪ tầng người dùng thật sự chạm vào.
//
// Bài này đi từ CHỮ trong một flows.toml trên đĩa, qua `API.FlowValidate` —
// đúng hàm mà action `flow.validate` gọi, tức đúng thứ `sagent flow validate`
// và mặt web nhìn thấy — và đòi cả hai nửa phải có mặt trong CÙNG một danh
// sách: nửa hình dạng (flow.VanDeCan) và nửa tra bảng (VanDeCanTheoBang).
func TestFlowValidateChayCaHaiNuaPhepSoiCan(t *testing.T) {
	khoTam(t)
	dir := t.TempDir()
	sg := filepath.Join(dir, ".sagent")
	if err := os.MkdirAll(sg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sg, "project.toml"), []byte("version = 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Ba bước, ba lỗi khác nhau, cố ý trong CÙNG một file: nếu chỉ một nửa phép
	// soi được gọi thì danh sách sẽ thiếu đúng nửa kia, và bài đỏ ngay.
	src := `version = 1
[flow]
  [flow.thu]
    [[flow.thu.step]]
      id     = "nham-cho"
      type   = "shell"
      run    = ["go", "version"]
      can    = ["goi-tool"]

    [[flow.thu.step]]
      id     = "khoa-la"
      type   = "route"
      routes = ["grok"]
      can    = ["goi-tools"]

    [[flow.thu.step]]
      id     = "do-anh"
      type   = "model"
      route  = "deepseek"
      can    = ["dau-vao-anh"]
      prompt = "anh nay mau gi"
`
	if err := os.WriteFile(filepath.Join(sg, "flows.toml"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	a := apiThu(
		routeCfg{Ten: "deepseek", BaseURL: "https://modelapi.vn/v1",
			Model: "deepseek-v4-flash", KeyID: "deepseek"},
		routeCfg{Ten: "grok", BaseURL: "https://modelapi.vn/v1",
			Model: "grok-4.5", KeyID: "grok"},
	)
	ps, err := a.FlowValidate(dir)
	if err != nil {
		t.Fatal(err)
	}

	gop := map[string]string{}
	for _, p := range ps {
		gop[p.Step] += p.Msg + "\n"
	}

	// Nửa HÌNH DẠNG — gói flow tự soi được, không cần bảng.
	if !strings.Contains(gop["nham-cho"], "chi bu\u1edbc `route`") &&
		!strings.Contains(gop["nham-cho"], "type = \"shell\"") {
		t.Errorf("flow.VanDeCan KHÔNG được gọi qua FlowValidate — `can` khai ở bước shell "+
			"lọt qua:\n%s", gop["nham-cho"])
	}
	// Nửa TRA BẢNG — chỉ internal/api trả lời được.
	if !strings.Contains(gop["khoa-la"], "goi-tools") {
		t.Errorf("VanDeCanTheoBang KHÔNG được gọi qua FlowValidate — khoá gõ nhầm lọt qua, "+
			"và lúc chạy bộ lọc sẽ không lọc gì:\n%s", gop["khoa-la"])
	}
	if !strings.Contains(gop["do-anh"], "does not support image") {
		t.Errorf("bước `model` trỏ vào route ĐÃ ĐO ĐƯỢC là không đọc được ảnh mà "+
			"`flow validate` im lặng:\n%s", gop["do-anh"])
	}

	// Và hai lỗi kia phải là LỖI, không phải cảnh báo: chúng chặn được một lượt
	// chạy chắc chắn hỏng, tại thời điểm chưa tiêu một đồng nào.
	for _, id := range []string{"khoa-la", "do-anh"} {
		var coLoi bool
		for _, p := range ps {
			if p.Step == id && !p.Warn {
				coLoi = true
			}
		}
		if !coLoi {
			t.Errorf("bước %q chỉ được cảnh báo — `sagent flow validate` sẽ thoát mã 0 "+
				"và lượt chạy vẫn được bấm", id)
		}
	}
}
