package aiapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// mocGoc là "bây giờ" cố định của các test đọc header dạng mốc thời gian.
// Đặt cứng chứ không dùng time.Now(): một test về thời gian mà tự lấy giờ hệ
// thống là test sẽ hỏng vào một buổi sáng nào đó và không ai biết vì sao.
var mocGoc = time.Date(2026, 10, 21, 7, 28, 0, 0, time.UTC)

// Retry-After có ĐÚNG hai dạng RFC 9110 cho phép, và dạng mốc thời gian là dạng
// hay bị quên nhất. Quên nó thì Atoi hỏng, mã tưởng "không có header", rồi tự lùi
// 1 giây — tức gọi lại vào đúng bức tường nhà cung cấp vừa dựng.
func TestDocRetryAfterCaHaiDang(t *testing.T) {
	ca := []struct {
		ten     string
		header  string
		muon    time.Duration
		nguon   string
		docDuoc bool
	}{
		{"số giây", "30", 30 * time.Second, NguonGiay, true},
		{"số giây có khoảng trắng", "  7  ", 7 * time.Second, NguonGiay, true},
		{"số giây bằng 0 — RFC cho phép, nghĩa là thử lại ngay",
			"0", 0, NguonGiay, true},

		// Ba định dạng ngày mà HTTP cho phép. Tự viết time.Parse một dạng là chắc
		// chắn thiếu hai dạng kia.
		{"HTTP-date RFC 1123", "Wed, 21 Oct 2026 07:29:00 GMT",
			60 * time.Second, NguonMoc, true},
		{"HTTP-date RFC 850", "Wednesday, 21-Oct-26 07:30:00 GMT",
			120 * time.Second, NguonMoc, true},
		{"HTTP-date asctime", "Wed Oct 21 07:28:30 2026",
			30 * time.Second, NguonMoc, true},

		// Mốc đã trôi qua: đồng hồ hai bên lệch, hoặc phản hồi tới muộn. Không
		// phải lỗi, và tuyệt đối không được ra thời lượng ÂM.
		{"HTTP-date đã qua", "Wed, 21 Oct 2026 07:00:00 GMT", 0, NguonMoc, true},

		// Không đọc được: phải NÓI là không đọc được, để bên gọi ghi vào nhật ký
		// đúng nguồn. Đoán một con số rồi im lặng là thứ dự án này cấm.
		{"rỗng", "", 0, NguonThieu, false},
		{"chữ vớ vẩn", "soon", 0, NguonKhongRo, false},
		{"số âm", "-5", 0, NguonKhongRo, false},
		{"số thực — RFC KHÔNG cho phép", "1.5", 0, NguonKhongRo, false},
		{"ngày sai định dạng", "2026-10-21T07:29:00Z", 0, NguonKhongRo, false},
	}
	for _, c := range ca {
		t.Run(c.ten, func(t *testing.T) {
			d, nguon, ok := docRetryAfter(c.header, mocGoc)
			if ok != c.docDuoc {
				t.Fatalf("docRetryAfter(%q) đọc được = %v, muốn %v", c.header, ok, c.docDuoc)
			}
			if ok && d != c.muon {
				t.Errorf("docRetryAfter(%q) = %s, muốn %s", c.header, d, c.muon)
			}
			if nguon != c.nguon {
				t.Errorf("docRetryAfter(%q) nguồn = %q, muốn %q", c.header, nguon, c.nguon)
			}
		})
	}
}

// Không đọc được header thì phải LÙI DẦN, và phải tăng theo lần thử. Cố định một
// giá trị là dò một cửa sổ hạn mức mà mình không biết dài bao lâu.
func TestKhongDocDuocThiLuiDan(t *testing.T) {
	for lan, muon := range map[int]time.Duration{1: time.Second, 2: 2 * time.Second, 3: 4 * time.Second} {
		d, nguon, nen := tinhCho("", lan, 0, mocGoc)
		if !nen {
			t.Fatalf("lần %d: không nên vượt trần", lan)
		}
		if d != muon {
			t.Errorf("lần %d: lùi %s, muốn %s", lan, d, muon)
		}
		if nguon != NguonThieu {
			t.Errorf("lần %d: nguồn = %q, muốn %q", lan, nguon, NguonThieu)
		}
	}
}

// Trần chờ. Vượt trần thì THÔI, chứ không chờ — nếu không, một `Retry-After:
// 3600` làm cả lượt flow đứng im một tiếng, không log, không lối ra.
func TestTranCho(t *testing.T) {
	// Một lần chờ vượt TranMotLanCho.
	if _, _, nen := tinhCho("3600", 1, 0, mocGoc); nen {
		t.Error("Retry-After: 3600 mà vẫn chịu chờ — trần một lần không có tác dụng")
	}
	if _, _, nen := tinhCho("31", 1, 0, mocGoc); nen {
		t.Errorf("31s > TranMotLanCho (%s) mà vẫn chịu chờ", TranMotLanCho)
	}
	if _, _, nen := tinhCho("30", 1, 0, mocGoc); !nen {
		t.Errorf("30s == TranMotLanCho (%s) thì phải chờ được", TranMotLanCho)
	}
	// Tổng vượt TranTongCho: mỗi lần lẻ đều dưới trần một lần, nhưng cộng lại thì không.
	if _, _, nen := tinhCho("30", 2, 40*time.Second, mocGoc); nen {
		t.Errorf("đã chờ 40s + 30s = 70s > TranTongCho (%s) mà vẫn chịu chờ", TranTongCho)
	}
}

// ===== Từ đây là các ca chạy qua HTTP thật (httptest), không phải hàm lẻ.

// srv429 dựng một server trả 429 `soLan` lần đầu (kèm header cho trước) rồi trả
// một câu trả lời hợp lệ. Đếm số lần bị gọi.
func srv429(t *testing.T, soLan int, header string, than string) (*httptest.Server, *int32) {
	t.Helper()
	var dem int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&dem, 1)
		if int(n) <= soLan {
			if header != "" {
				w.Header().Set("Retry-After", header)
			}
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, than)
			return
		}
		fmt.Fprint(w, `{"model":"m","choices":[{"message":{"role":"assistant","content":"xong"}}],`+
			`"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)
	}))
	t.Cleanup(srv.Close)
	return srv, &dem
}

// Ca chính: bị chặn tốc độ MỘT lần, chờ theo `Retry-After: 1`, thử lại CHÍNH
// route đó và đi tiếp. Đây là hành vi mà cả tính năng này tồn tại vì nó.
func TestChanTocDoRoiThuLaiChinhRouteDo(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	srv, dem := srv429(t, 1, "1", `{"error":"rate limited"}`)

	bat := time.Now()
	kq, err := Goi(context.Background(), Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	mat := time.Since(bat)
	if err != nil {
		t.Fatalf("429 một lần rồi ổn mà vẫn hỏng: %v", err)
	}
	if kq.NoiDung != "xong" {
		t.Errorf("nội dung = %q, muốn %q", kq.NoiDung, "xong")
	}
	if *dem != 2 {
		t.Errorf("gọi %d lần, muốn 2 (lần đầu bị chặn, lần hai chạy)", *dem)
	}
	// Chờ thật, không phải bỏ qua header rồi gọi lại ngay.
	if mat < time.Second {
		t.Errorf("chỉ mất %s — có vẻ KHÔNG hề chờ theo Retry-After: 1", mat)
	}
	// Lượt chờ THÀNH CÔNG cũng phải để lại dấu vết. Đây là ca hay mất tin nhất:
	// err == nil nên nếu nhật ký chỉ đi kèm lỗi thì nó biến mất, và người dùng
	// thấy một lượt đột nhiên chậm mà không có gì giải thích.
	if len(kq.ChoLai) != 1 {
		t.Fatalf("ChoLai có %d mục, muốn 1 — lần chờ thành công bị nuốt mất", len(kq.ChoLai))
	}
	if kq.ChoLai[0].Header != "1" || kq.ChoLai[0].Nguon != NguonGiay {
		t.Errorf("ChoLai[0] = %+v, muốn Header=1 nguồn=%q", kq.ChoLai[0], NguonGiay)
	}
	if kq.ChoLai[0].Cho != time.Second {
		t.Errorf("ChoLai[0].Cho = %s, muốn 1s", kq.ChoLai[0].Cho)
	}
}

// Dạng mốc thời gian phải đi được hết đường HTTP thật, không chỉ qua hàm lẻ.
func TestChanTocDoVoiMocThoiGian(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	moc := time.Now().UTC().Add(1200 * time.Millisecond).Format(http.TimeFormat)
	srv, dem := srv429(t, 1, moc, `{"error":"slow down"}`)

	bat := time.Now()
	kq, err := Goi(context.Background(), Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	mat := time.Since(bat)
	if err != nil {
		t.Fatalf("hỏng: %v", err)
	}
	if *dem != 2 {
		t.Errorf("gọi %d lần, muốn 2", *dem)
	}
	if kq.ChoLai[0].Nguon != NguonMoc {
		t.Fatalf("nguồn = %q, muốn %q — mốc thời gian bị đọc nhầm thành không-đọc-được",
			kq.ChoLai[0].Nguon, NguonMoc)
	}
	// http.TimeFormat chỉ có độ phân giải tới GIÂY, nên 1200ms bị cắt còn khoảng
	// 200ms-1200ms. Kiểm mốc dưới an toàn là 150ms: đủ để phân biệt với "không
	// chờ gì cả", mà không phụ thuộc vào phần lẻ bị cắt.
	if mat < 150*time.Millisecond {
		t.Errorf("chỉ mất %s — không hề chờ theo mốc thời gian", mat)
	}
}

// Không có header thì phải lùi dần, và nói rõ trong nhật ký là mình TỰ lùi chứ
// không phải nhà cung cấp bảo thế.
func TestKhongCoHeaderThiLuiDanVaNoiRa(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	srv, dem := srv429(t, 1, "", `{"error":"429 no header"}`)

	kq, err := Goi(context.Background(), Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	if err != nil {
		t.Fatalf("hỏng: %v", err)
	}
	if *dem != 2 {
		t.Errorf("gọi %d lần, muốn 2", *dem)
	}
	if kq.ChoLai[0].Nguon != NguonThieu {
		t.Errorf("nguồn = %q, muốn %q", kq.ChoLai[0].Nguon, NguonThieu)
	}
	if kq.ChoLai[0].Header != "" {
		t.Errorf("Header = %q, muốn rỗng", kq.ChoLai[0].Header)
	}
}

// Header có mà đọc không nổi: vẫn lùi dần, NHƯNG nhật ký phải giữ NGUYÊN VĂN cái
// header khó đọc đó. Vứt nó đi là vứt mất manh mối duy nhất để biết nhà cung cấp
// đang gửi kiểu gì.
func TestHeaderKhongDocDuocThiGiuNguyenVan(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	srv, _ := srv429(t, 1, "lat nua quay lai", `{"error":"429"}`)

	kq, err := Goi(context.Background(), Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	if err != nil {
		t.Fatalf("hỏng: %v", err)
	}
	if kq.ChoLai[0].Nguon != NguonKhongRo {
		t.Errorf("nguồn = %q, muốn %q", kq.ChoLai[0].Nguon, NguonKhongRo)
	}
	if kq.ChoLai[0].Header != "lat nua quay lai" {
		t.Errorf("Header = %q — nguyên văn header khó đọc bị vứt mất", kq.ChoLai[0].Header)
	}
}

// Hết lượt thử lại mà vẫn 429: lỗi trả ra phải mang NGUYÊN VĂN thân 429 của nhà
// cung cấp. Thân đó là chỗ họ nói hạn mức nào bị chạm (phút hay ngày, token hay
// lời gọi) và request id. Nuốt nó rồi in "bị chặn tốc độ" là biến một thông điệp
// hành động được thành một lời than.
func TestHetLuotThuLaiVanBaoNguyenVan(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	const than = `{"error":{"message":"rate limit exceeded: 20 RPM","request_id":"req_abc123"}}`
	// 99 = không bao giờ hết 429.
	srv, dem := srv429(t, 99, "0", than)

	_, err := Goi(context.Background(), Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	if err == nil {
		t.Fatal("429 mãi mà vẫn báo thành công")
	}
	if int(*dem) != SoLanThuLai+1 {
		t.Errorf("gọi %d lần, muốn %d (1 lần đầu + %d lần thử lại)",
			*dem, SoLanThuLai+1, SoLanThuLai)
	}
	for _, phai := range []string{"rate limit exceeded: 20 RPM", "req_abc123", "429"} {
		if !strings.Contains(err.Error(), phai) {
			t.Errorf("lỗi KHÔNG chứa %q — đã nuốt mất nguyên văn của nhà cung cấp.\nlỗi: %s",
				phai, err.Error())
		}
	}
	// Và phải kể lại đã chờ mấy lần, mỗi lần vì sao — nếu không, người đọc chỉ
	// thấy một lỗi 429 và không biết mã đã thử lại hay chưa.
	if !strings.Contains(err.Error(), "đã thử lại") {
		t.Errorf("lỗi không kể lại các lần chờ:\n%s", err.Error())
	}
}

// 429 hết lượt KHÔNG phải lỗi người dùng. Đây là điều kiện để tầng
// `internal/api` được phép chuyển sang route dự phòng sau khi ta đã chờ mà vẫn
// hỏng. Đánh nhầm nó thành lỗi người dùng là chặn luôn đường lui cuối cùng.
func Test429KhongPhaiLoiNguoiDung(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	srv, _ := srv429(t, 99, "0", `{"error":"429"}`)

	_, err := Goi(context.Background(), Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	if err == nil {
		t.Fatal("muốn lỗi")
	}
	if LoiNguoiDung(err) {
		t.Error("429 bị đánh là lỗi người dùng — tầng trên sẽ KHÔNG chuyển route dự phòng nữa")
	}
	if !BiChanTocDo(err) {
		t.Errorf("BiChanTocDo = false với lỗi 429: %v", err)
	}
}

// `Retry-After` vượt trần thì KHÔNG chờ, trả lỗi ngay để tầng trên kịp chuyển
// route. Chờ một tiếng rồi mới chuyển là tệ hơn cả không thử lại.
func TestVuotTranThiKhongChoMaBoCuocNgay(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	srv, dem := srv429(t, 99, "3600", `{"error":"daily quota"}`)

	bat := time.Now()
	_, err := Goi(context.Background(), Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	mat := time.Since(bat)
	if err == nil {
		t.Fatal("muốn lỗi")
	}
	if mat > 5*time.Second {
		t.Errorf("mất %s — đã thật sự ngồi chờ Retry-After: 3600", mat)
	}
	if *dem != 1 {
		t.Errorf("gọi %d lần, muốn 1 — vượt trần thì không thử lại lần nào", *dem)
	}
	if !strings.Contains(err.Error(), "vượt trần") {
		t.Errorf("lỗi không nói vì sao bỏ cuộc:\n%s", err.Error())
	}
	if !strings.Contains(err.Error(), "daily quota") {
		t.Errorf("lỗi nuốt mất nguyên văn:\n%s", err.Error())
	}
}

// Ngữ cảnh bị huỷ giữa lúc đang chờ thì phải tỉnh NGAY, không nằm ngủ hết giờ.
// Dùng time.Sleep ở đây là kiểu treo mà không log nào giải thích nổi.
func TestHuyNguCanhThiThoiChoNgay(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	srv, _ := srv429(t, 99, "20", `{"error":"429"}`)

	ctx, huy := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		huy()
	}()

	bat := time.Now()
	_, err := Goi(ctx, Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	mat := time.Since(bat)
	if err == nil {
		t.Fatal("muốn lỗi")
	}
	if mat > 5*time.Second {
		t.Errorf("mất %s — vẫn nằm ngủ hết 20 giây dù ngữ cảnh đã huỷ", mat)
	}
}

// Hạn chót của ctx ngắn hơn thời gian định chờ: đừng chờ. Hỏng nhanh còn hơn
// hỏng muộn, vì tầng trên còn phải kịp gọi route dự phòng trong phần hạn còn lại.
func TestKhongChoQuaHanChotCuaLuot(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")
	srv, _ := srv429(t, 99, "20", `{"error":"429"}`)

	ctx, huy := context.WithTimeout(context.Background(), 3*time.Second)
	defer huy()

	bat := time.Now()
	_, err := Goi(ctx, Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
	mat := time.Since(bat)
	if err == nil {
		t.Fatal("muốn lỗi")
	}
	if mat > 2*time.Second {
		t.Errorf("mất %s — vẫn chờ dù biết chắc sẽ vượt hạn chót", mat)
	}
}

// Đường streaming phải theo CÙNG luật, và quan trọng hơn: thử lại KHÔNG được
// nhân đôi chữ. 429 nằm ở dòng trạng thái nên lúc thử lại chưa có ký tự nào lọt
// qua `nhan` — test này là thứ giữ cho điều đó đúng.
func TestStreamChanTocDoKhongNhanDoiChu(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")

	var dem int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&dem, 1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":"rate limited"}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"model\":\"m\",\"choices\":[{\"delta\":{\"content\":\"xin \"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"chào\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":1,"+
			"\"completion_tokens\":2,\"total_tokens\":3}}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	var thu strings.Builder
	kq, err := GoiStream(context.Background(),
		Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi",
		func(s string) { thu.WriteString(s) })
	if err != nil {
		t.Fatalf("hỏng: %v", err)
	}
	if n := atomic.LoadInt32(&dem); n != 2 {
		t.Errorf("gọi %d lần, muốn 2", n)
	}
	if thu.String() != "xin chào" {
		t.Errorf("chữ nhận qua callback = %q, muốn %q — thử lại đã phát lại nội dung",
			thu.String(), "xin chào")
	}
	if kq.NoiDung != "xin chào" {
		t.Errorf("nội dung = %q, muốn %q", kq.NoiDung, "xin chào")
	}
	if len(kq.ChoLai) != 1 {
		t.Errorf("ChoLai có %d mục, muốn 1 — đường stream mất nhật ký chờ", len(kq.ChoLai))
	}
	// Sổ chi phí vẫn phải đúng sau khi thử lại: usage của lượt THẬT SỰ trả lời.
	if kq.Usage.Tong != 3 || kq.ThieuUsage {
		t.Errorf("usage = %+v, ThieuUsage = %v — thử lại làm hỏng sổ chi phí",
			kq.Usage, kq.ThieuUsage)
	}
}

// Các mã lỗi KHÁC 429 tuyệt đối KHÔNG được thử lại theo đường này.
//
// Vì sao phải có test: 503 đo được 20/08 có thân "No available channel for model
// grok-code-fast-1 under group grok" — hỏng theo TỪNG MODEL, nên thứ cứu được nó
// là đổi route (đổi model), không phải chờ. Nếu 503 cũng bị chờ ở đây thì mỗi lần
// nhà cung cấp sập, người dùng phải chờ thêm 3 giây trước khi nhận đúng cái lỗi
// mà lẽ ra đã được chuyển route ngay.
func TestChiThuLaiVoi429(t *testing.T) {
	homeGia(t)
	datKey(t, "k", "sk-test")

	for _, ma := range []int{400, 401, 403, 404, 500, 502, 503} {
		t.Run(fmt.Sprint(ma), func(t *testing.T) {
			var dem int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&dem, 1)
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(ma)
				fmt.Fprint(w, `{"error":"khong phai 429"}`)
			}))
			defer srv.Close()

			_, err := Goi(context.Background(),
				Route{Ten: "r", BaseURL: srv.URL, Model: "m", KeyID: "k"}, "hi")
			if err == nil {
				t.Fatalf("HTTP %d mà báo thành công", ma)
			}
			if n := atomic.LoadInt32(&dem); n != 1 {
				t.Errorf("HTTP %d bị gọi %d lần — mã khác 429 KHÔNG được thử lại", ma, n)
			}
			if BiChanTocDo(err) {
				t.Errorf("HTTP %d bị nhận nhầm là chặn tốc độ", ma)
			}
		})
	}
}

// Lượt bình thường không được dài ra vô cớ: không chờ lần nào thì lỗi phải giữ
// nguyên như trước, không đính thêm dòng nhật ký rỗng.
func TestKhongChoThiKhongThemChu(t *testing.T) {
	if s := MoTaChoLai(nil); s != "" {
		t.Errorf("MoTaChoLai(nil) = %q, muốn rỗng", s)
	}
	if s := themChoLai(nil); s != "" {
		t.Errorf("themChoLai(nil) = %q, muốn rỗng", s)
	}
}
