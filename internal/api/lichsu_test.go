package api

// Sổ lời gọi API (bảng api_calls, schema v7) và hợp đồng của AICall khi phải
// chuyển route.
//
// Năm ca dưới đây là năm cách chuyện này hỏng trong thực tế, không phải năm cách
// gọi hàm:
//
//	(a) đường thẳng    — route chính chạy được, KHÔNG được đụng tới route dự phòng
//	(b) chuyển một lần — route chính hỏng, và người gọi phải BIẾT là đã chuyển
//	(c) lỗi người dùng — 401/403 thì chuyển route chỉ tốn thêm tiền
//	(d) hỏng cả hai    — lỗi trả về phải mang NGUYÊN VĂN của cả hai
//	(e) sổ kín miệng   — ghi tiền và token, KHÔNG ghi câu hỏi lẫn câu trả lời

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/config"
)

// mayTot dựng một server trả lời như OpenAI, đếm số lần bị gọi.
func mayTot(model, noiDung string, vao, ra int) (*httptest.Server, *int32) {
	var n int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   model,
			"choices": []any{map[string]any{"message": map[string]string{"content": noiDung}}},
			"usage":   map[string]int{"prompt_tokens": vao, "completion_tokens": ra, "total_tokens": vao + ra},
		})
	}))
	return s, &n
}

// mayHong dựng một server luôn hỏng với mã và thân lỗi cho sẵn.
func mayHong(ma int, than string) (*httptest.Server, *int32) {
	var n int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		w.WriteHeader(ma)
		_, _ = w.Write([]byte(than))
	}))
	return s, &n
}

type soDong struct {
	Route, Model string
	OK           bool
	Vao, Ra      int
	LyDo         string
}

func lichSu(t *testing.T, a *API) []soDong {
	t.Helper()
	ds, err := a.AIHistory(50)
	if err != nil {
		t.Fatalf("không đọc được sổ lời gọi API: %v", err)
	}
	out := make([]soDong, 0, len(ds))
	for _, g := range ds {
		out = append(out, soDong{Route: g.Route, Model: g.Model, OK: g.OK,
			Vao: g.TokensIn, Ra: g.TokensOut, LyDo: g.LyDo})
	}
	return out
}

// ---- ca (a): route chính chạy được thì KHÔNG đụng tới route dự phòng ----
//
// Nghe hiển nhiên, nhưng đây đúng là chỗ một lần "gọi cho chắc" biến thành hai
// nhà cung cấp cho mỗi câu hỏi — tức là trả tiền gấp đôi mà không ai thấy.
func TestCaA_RouteChinhChayThiKhongDungRouteDuPhong(t *testing.T) {
	homeGiaAPI(t)
	chinh, demChinh := mayTot("m-chinh", "xong", 11, 22)
	defer chinh.Close()
	phu, demPhu := mayTot("m-phu", "khong-nen-thay", 1, 1)
	defer phu.Close()

	var c config.Config
	themRoute(&c, "chinh", chinh.URL, "m1")
	themRoute(&c, "phu", phu.URL, "m2")
	c.AI.DefaultRoute = "chinh"
	c.AI.FallbackRoutes = []string{"phu"}

	a := dungAPI(t, c)
	kq, err := a.AICall(context.Background(), "", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(demPhu); got != 0 {
		t.Errorf("route dự phòng bị gọi %d lần trong khi route chính chạy được — trả tiền gấp đôi", got)
	}
	if got := atomic.LoadInt32(demChinh); got != 1 {
		t.Errorf("route chính phải được gọi đúng 1 lần, thấy %d", got)
	}
	if kq.Route != "chinh" {
		t.Errorf("KetQua.Route = %q, muốn \"chinh\"", kq.Route)
	}
	if kq.RouteChinh != "" || kq.LoiChinh != "" {
		t.Errorf("không chuyển route mà vẫn báo là đã chuyển: %+v", kq)
	}
	if len(kq.DaThu) != 1 || kq.DaThu[0] != "chinh" {
		t.Errorf("DaThu = %v, muốn [chinh]", kq.DaThu)
	}
	so := lichSu(t, a)
	if len(so) != 1 {
		t.Fatalf("sổ có %d dòng, muốn 1: %+v", len(so), so)
	}
	if !so[0].OK || so[0].Route != "chinh" || so[0].Vao != 11 || so[0].Ra != 22 {
		t.Errorf("dòng sổ sai: %+v", so[0])
	}
}

// ---- ca (b): chuyển ĐÚNG MỘT LẦN, và người gọi phải biết là đã chuyển ----
//
// Trước bản này, việc đổi nhà cung cấp chỉ được nói qua bus.Warnf. CLI và mặt web
// KHÔNG nghe bus, nên với họ nó im lặng: câu trả lời hiện ra, không ai biết nó
// đến từ đâu, và request id của lần hỏng đầu — thứ duy nhất dùng được khi đi hỏi
// nhà cung cấp — bốc hơi.
func TestCaB_ChuyenRouteThiKetQuaMangDuChungCu(t *testing.T) {
	homeGiaAPI(t)
	hong, _ := mayHong(429, `{"error":"rate limit (request id: RID-CHINH-9)"}`)
	defer hong.Close()
	tot, _ := mayTot("m-du-phong", "OK", 3, 4)
	defer tot.Close()

	var c config.Config
	themRoute(&c, "chinh", hong.URL, "m1")
	themRoute(&c, "phu", tot.URL, "m2")
	c.AI.DefaultRoute = "chinh"
	c.AI.FallbackRoutes = []string{"phu"}

	a := dungAPI(t, c)
	kq, err := a.AICall(context.Background(), "", "hi")
	if err != nil {
		t.Fatalf("route chính hỏng mà không chuyển sang dự phòng: %v", err)
	}

	if kq.Route != "phu" {
		t.Errorf("KetQua.Route = %q, muốn \"phu\" — không biết ai trả lời thì không biết đang tiêu tiền của ai", kq.Route)
	}
	if kq.RouteChinh != "chinh" {
		t.Errorf("KetQua.RouteChinh = %q, muốn \"chinh\"", kq.RouteChinh)
	}
	if !kq.DaChuyenRoute() {
		t.Error("DaChuyenRoute() = false sau khi đã chuyển route")
	}
	// Đây là điều kiện khó nhất của DoD Pha 4: lỗi gốc + request id phải theo
	// KẾT QUẢ về tận tay người gọi, không chỉ nằm trên bus.
	for _, phai := range []string{"RID-CHINH-9", "429"} {
		if !strings.Contains(kq.LoiChinh, phai) {
			t.Errorf("KetQua.LoiChinh mất %q — không còn gì để đi hỏi nhà cung cấp:\n%s", phai, kq.LoiChinh)
		}
	}
	if len(kq.DaThu) != 2 || kq.DaThu[0] != "chinh" || kq.DaThu[1] != "phu" {
		t.Errorf("DaThu = %v, muốn [chinh phu]", kq.DaThu)
	}
	// Usage phải là của route THẮNG. Route hỏng không có usage nào cả, nên lẫn
	// lộn ở đây nghĩa là con số tiêu tiền hiện ra sai.
	if kq.Usage.Vao != 3 || kq.Usage.Ra != 4 || kq.Usage.Tong != 7 {
		t.Errorf("usage không phải của route thắng: %+v", kq.Usage)
	}

	so := lichSu(t, a)
	if len(so) != 2 {
		t.Fatalf("sổ có %d dòng, muốn 2 (một hỏng + một chạy được): %+v", len(so), so)
	}
	// APICalls trả mới nhất trước.
	if !so[0].OK || so[0].Route != "phu" {
		t.Errorf("dòng mới nhất phải là lần gọi route phụ chạy được, thấy %+v", so[0])
	}
	if so[1].OK || so[1].Route != "chinh" || !strings.Contains(so[1].LyDo, "RID-CHINH-9") {
		t.Errorf("sổ không giữ lý do hỏng nguyên văn của route chính: %+v", so[1])
	}
}

// ---- ca (b2): ĐÚNG MỘT lần chuyển, dù khai bao nhiêu route dự phòng ----
//
// Vòng lặp cũ chạy hết fallback_routes. Khai bốn route thì một prompt hỏng ở nhà
// cung cấp đầu thành bốn lời gọi thật, mỗi lời gọi chờ tới 120 giây — người dùng
// ngồi nhìn tám phút rồi mới nhận lỗi, và trả tiền cho từng lần thử.
func TestCaB2_ChiChuyenDungMotLanDuKhaiNhieuRouteDuPhong(t *testing.T) {
	homeGiaAPI(t)
	h1, d1 := mayHong(500, "hỏng 1")
	h2, d2 := mayHong(500, "hỏng 2")
	t3, d3 := mayTot("m3", "khong-nen-toi-luot-nay", 1, 1)
	defer h1.Close()
	defer h2.Close()
	defer t3.Close()

	var c config.Config
	themRoute(&c, "r1", h1.URL, "m1")
	themRoute(&c, "r2", h2.URL, "m2")
	themRoute(&c, "r3", t3.URL, "m3")
	c.AI.DefaultRoute = "r1"
	c.AI.FallbackRoutes = []string{"r2", "r3"}

	a := dungAPI(t, c)
	if _, err := a.AICall(context.Background(), "", "hi"); err == nil {
		t.Fatal("hai route đầu đều hỏng mà báo thành công")
	}
	if got := atomic.LoadInt32(d3); got != 0 {
		t.Errorf("route dự phòng THỨ HAI bị gọi %d lần — chỉ được chuyển đúng một lần, "+
			"nếu không thì mỗi lần hỏng là một chuỗi lời gọi tính tiền", got)
	}
	if atomic.LoadInt32(d1) != 1 || atomic.LoadInt32(d2) != 1 {
		t.Errorf("số lần gọi hai route đầu = %d/%d, muốn 1/1", atomic.LoadInt32(d1), atomic.LoadInt32(d2))
	}
	if so := lichSu(t, a); len(so) != 2 {
		t.Errorf("sổ có %d dòng, muốn đúng 2 lời gọi thật: %+v", len(so), so)
	}
}

// ---- ca (c): 401/403 thì KHÔNG chuyển route ----
//
// Key sai hoặc hết quyền là lỗi của phía người dùng. Route dự phòng lặp lại đúng
// cái sai đó ở nhà cung cấp thứ hai: không cứu được gì, nhân đôi thời gian chờ,
// và chôn nguyên nhân thật xuống dòng thứ hai của một lỗi ghép.
func TestCaC_KhongFallbackKhiLoiDoNguoiDung(t *testing.T) {
	// Mỗi mã một sổ RIÊNG (homeGiaAPI đặt HOME tạm, mà state.db nằm dưới HOME).
	// Dùng chung một sổ thì phép đếm dòng của ca sau cộng dồn dòng của ca trước
	// và không nói lên điều gì.
	for _, ma := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(strconv.Itoa(ma), func(t *testing.T) {
			homeGiaAPI(t)
			hong, _ := mayHong(ma, `{"error":"invalid api key"}`)
			defer hong.Close()
			tot, demPhu := mayTot("m-phu", "khong-nen-thay", 1, 1)
			defer tot.Close()

			var c config.Config
			themRoute(&c, "chinh", hong.URL, "m1")
			themRoute(&c, "phu", tot.URL, "m2")
			c.AI.DefaultRoute = "chinh"
			c.AI.FallbackRoutes = []string{"phu"}

			a := dungAPI(t, c)
			_, err := a.AICall(context.Background(), "", "hi")
			if err == nil {
				t.Fatalf("HTTP %d mà báo thành công", ma)
			}
			if got := atomic.LoadInt32(demPhu); got != 0 {
				t.Errorf("route dự phòng bị gọi %d lần — key sai thì route khác cũng không cứu được", got)
			}
			// Lỗi trả về phải là ĐÚNG lỗi đó, không bọc thêm "cả hai đều hỏng".
			if strings.Contains(err.Error(), "đều hỏng") {
				t.Errorf("lỗi bị bọc thành lỗi ghép, nguyên nhân thật bị chôn:\n%v", err)
			}
			if !strings.Contains(err.Error(), "invalid api key") {
				t.Errorf("mất thân lỗi nguyên văn: %v", err)
			}
			if so := lichSu(t, a); len(so) != 1 {
				t.Errorf("sổ có %d dòng, muốn 1: %+v", len(so), so)
			}
		})
	}
}

// Prompt rỗng cũng là lỗi người dùng: chặn TRƯỚC khi chạm mạng, và không được
// đem đi thử lại ở route thứ hai.
func TestCaC2_PromptRongThiKhongGoiRouteNao(t *testing.T) {
	homeGiaAPI(t)
	m1, d1 := mayTot("m1", "x", 1, 1)
	m2, d2 := mayTot("m2", "x", 1, 1)
	defer m1.Close()
	defer m2.Close()

	var c config.Config
	themRoute(&c, "chinh", m1.URL, "m1")
	themRoute(&c, "phu", m2.URL, "m2")
	c.AI.DefaultRoute = "chinh"
	c.AI.FallbackRoutes = []string{"phu"}

	a := dungAPI(t, c)
	if _, err := a.AICall(context.Background(), "", "   "); err == nil {
		t.Fatal("prompt rỗng mà vẫn báo thành công")
	}
	if atomic.LoadInt32(d1) != 0 || atomic.LoadInt32(d2) != 0 {
		t.Errorf("prompt rỗng vẫn bắn ra mạng %d/%d lần — một lời gọi hỏng vẫn có thể bị tính tiền",
			atomic.LoadInt32(d1), atomic.LoadInt32(d2))
	}
}

// ---- ca (d): hỏng cả hai thì lỗi mang NGUYÊN VĂN của cả hai ----
func TestCaD_HaiRouteHongThiTraCaHaiLoi(t *testing.T) {
	homeGiaAPI(t)
	h1, _ := mayHong(429, `{"error":"quá tải (request id: RID-MOT)"}`)
	h2, _ := mayHong(503, `{"error":"bảo trì (request id: RID-HAI)"}`)
	defer h1.Close()
	defer h2.Close()

	var c config.Config
	themRoute(&c, "mot", h1.URL, "m1")
	themRoute(&c, "hai", h2.URL, "m2")
	c.AI.DefaultRoute = "mot"
	c.AI.FallbackRoutes = []string{"hai"}

	a := dungAPI(t, c)
	_, err := a.AICall(context.Background(), "", "hi")
	if err == nil {
		t.Fatal("mọi route hỏng mà không báo lỗi")
	}
	for _, phai := range []string{"RID-MOT", "RID-HAI", "mot", "hai"} {
		if !strings.Contains(err.Error(), phai) {
			t.Errorf("lỗi thiếu %q — phải nói rõ đã thử route nào và mỗi cái hỏng vì gì:\n%v", phai, err)
		}
	}
	so := lichSu(t, a)
	if len(so) != 2 {
		t.Fatalf("sổ có %d dòng, muốn 2 dòng hỏng: %+v", len(so), so)
	}
	for _, d := range so {
		if d.OK {
			t.Errorf("dòng %+v báo chạy được trong khi cả hai route đều hỏng", d)
		}
	}
}

// ---- ca (e): sổ ghi tiền, KHÔNG ghi nội dung hội thoại ----
//
// Người ta dán cả đoạn mã, khoá và dữ liệu khách vào prompt gửi cho nhà cung cấp
// bên ngoài. Ghi thêm một bản sao vĩnh viễn xuống đĩa là tự tạo kho bí mật thứ
// hai mà không ai xin — trong đúng một dự án mà luật số 1 là "file cấu hình
// không bao giờ chứa secret".
func TestCaE_SoKhongLuuPromptVaCauTraLoi(t *testing.T) {
	homeGiaAPI(t)
	const bimat = "TOKEN-RIENG-TU-KHONG-DUOC-GHI-XUONG-DIA"
	const traLoi = "CAU-TRA-LOI-KHONG-DUOC-GHI-XUONG-DIA"
	tot, _ := mayTot("m1", traLoi, 5, 6)
	defer tot.Close()

	var c config.Config
	themRoute(&c, "chinh", tot.URL, "m1")
	c.AI.DefaultRoute = "chinh"

	a := dungAPI(t, c)
	if _, err := a.AICall(context.Background(), "", "phân tích giúp: "+bimat); err != nil {
		t.Fatal(err)
	}
	ds, err := a.AIHistory(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 {
		t.Fatalf("sổ có %d dòng, muốn 1", len(ds))
	}
	// Soi TOÀN BỘ dòng sổ chứ không chỉ vài trường mình nhớ tên: thêm một trường
	// mới rồi nhét prompt vào đó thì test này vẫn phải đỏ.
	raw, err := json.Marshal(ds[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, cam := range []string{bimat, traLoi} {
		if strings.Contains(string(raw), cam) {
			t.Errorf("sổ lời gọi API đã ghi %q xuống đĩa — nó chỉ được ghi tiền và token, "+
				"không được ghi nội dung hội thoại:\n%s", cam, raw)
		}
	}
	// Nhưng phải ghi ĐỦ những thứ nó có nhiệm vụ ghi.
	if ds[0].Route != "chinh" || ds[0].Model != "m1" || !ds[0].OK {
		t.Errorf("dòng sổ thiếu route/model/thành-bại: %+v", ds[0])
	}
	if ds[0].TokensIn != 5 || ds[0].TokensOut != 6 {
		t.Errorf("dòng sổ mất token vào/ra: %+v", ds[0])
	}
	if ds[0].Luc.IsZero() {
		t.Error("dòng sổ không có thời điểm")
	}
}
