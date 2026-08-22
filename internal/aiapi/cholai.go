package aiapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Xử lý HTTP 429 (Too Many Requests) và header `Retry-After`.
//
// ===== VÌ SAO CHỜ RỒI THỬ LẠI CHÍNH ROUTE ĐÓ, CHỨ KHÔNG NHẢY NGAY SANG DỰ PHÒNG
//
// 429 đúng là lỗi phía nhà cung cấp theo nghĩa mã HTTP, nhưng nó KHÁC 5xx ở bản
// chất: 5xx nói "chỗ tôi đang hỏng", còn 429 nói "chỗ tôi vẫn tốt, ANH đi nhanh
// quá". Trộn hai câu đó vào một nhánh xử lý là bỏ mất thứ duy nhất 429 cho không:
// kèm theo nó thường có `Retry-After`, tức một LỜI HỨA có thời hạn rằng chờ đủ
// lâu thì đúng route này sẽ chạy. 5xx không bao giờ hứa như vậy.
//
// Ba dữ kiện của chính dự án này, không phải suy đoán chung chung:
//
//  1. `.sagent/project.toml` khai `deepseek` và `grok` DÙNG CHUNG một base_url:
//     https://modelapi.vn/v1. Nhảy sang "route dự phòng" khi bị chặn tốc độ là
//     hỏi lại ĐÚNG nhà bán lại vừa từ chối mình, từ cùng một máy, cách đó vài
//     mili giây. Nếu hạn mức tính theo tài khoản hay theo IP thì lượt thứ hai
//     chắc chắn cũng 429 — mất thêm một lời gọi, thêm một lần chờ, và thông điệp
//     lỗi cuối cùng dài gấp đôi trong khi nguyên nhân vẫn là câu đầu.
//
//  2. Hai route khai `key_id` KHÁC nhau, nên nếu modelapi.vn tính hạn mức theo
//     key thì route dự phòng LẠI cứu được. CHƯA ĐO ĐƯỢC nhà này chặn theo key
//     hay theo tài khoản — xem docs/BAO-CAO-429.md mục 2. Vì chưa biết, thứ tự
//     "chờ trước, nhảy sau" là thứ tự đúng: nó làm việc CHẮC CHẮN đúng (tôn
//     trọng lời hứa Retry-After) trước, rồi mới đánh cược (đổi key) sau. Đảo
//     ngược lại là đánh cược trước khi thử cái chắc.
//
//  3. Lần 503 đo được 20/08 có thân lỗi nguyên văn "No available channel for
//     model grok-code-fast-1 under group grok" — hỏng theo TỪNG MODEL. Đổi route
//     là đổi model, nên với 503 việc nhảy thật sự cứu được. Đó là lý do file này
//     KHÔNG mở rộng sang 503: luật cũ đã đúng cho 503, và chỉ 429 mới cần chờ.
//
// HẬU QUẢ NẾU CHỌN SAI, nói thẳng cả hai chiều:
//
//   - Chọn nhảy ngay (hướng đã bỏ): một cú chặn tốc độ 5 giây đẩy toàn bộ phần
//     còn lại của lượt flow sang nhà cung cấp thứ hai. Mà route chính thường là
//     route được chọn có lý do (rẻ hơn, nhanh hơn: đo 20/08 deepseek 2,3s/127
//     token so với grok 13,6s/1044 token). Một cú nghẽn thoáng qua thành một
//     quyết định đổi giá và đổi chất lượng câu trả lời cho cả lượt.
//
//   - Chọn chờ vô hạn (hướng cũng đã bỏ): nhà cung cấp trả `Retry-After: 3600`
//     là cả lượt flow đứng im một tiếng, không log, không lối ra. Vì vậy mọi con
//     số dưới đây đều có TRẦN, và vượt trần thì thôi chờ, trả lỗi để tầng trên
//     chuyển route.
//
// Nên hướng đã chọn là hướng lai: CHỜ THEO Retry-After RỒI THỬ LẠI CHÍNH ROUTE
// ĐÓ, có trần; hết lượt thử lại thì trả `*LoiAPI` với `Nguoi=false`, và tầng
// `internal/api` tự chuyển sang route dự phòng theo luật sẵn có. Không phải sửa
// một dòng nào ở tầng đó.
const (
	// SoLanThuLai là số lần thử LẠI sau lần đầu (tổng 3 lần chạm mạng).
	//
	// Vì sao 3 lần chạm mạng chứ không phải 2 hay 5: đây là con số DUY NHẤT dự án
	// này đo được về "một nhà cung cấp hỏng liên tiếp mấy lần rồi tự hồi phục" —
	// 20/08 lúc 16:54-16:56 route `deepseek` trả HTTP 503 ba lần rồi tự khỏi
	// (docs/DO-LUONG.md). Lấy đúng hình dạng đã đo, không bịa ra một con số đẹp.
	//
	// NÓI THẲNG chỗ nó không đủ: nếu 429 kéo dài đúng như sự kiện 3-lần-liên-tiếp
	// kia thì 3 lần chạm mạng vẫn hỏng hết. Đó là CỐ Ý — hết lượt thì đi tiếp
	// bằng route dự phòng, chứ không chờ mãi.
	SoLanThuLai = 2

	// TranMotLanCho là trần cho MỘT lần chờ. Vượt trần thì KHÔNG chờ.
	//
	// Vì sao 30 giây: lượt gọi thật chậm nhất từng đo được là grok-4.5 hết 13,6s,
	// và một lần 31s ở bản streaming (docs/DO-LUONG.md). 30 giây tức là "chờ
	// bằng khoảng một lượt gọi chậm nữa" — vẫn còn nằm trong thứ người dùng đã
	// quen chịu. Trên mức đó, `Retry-After` không còn nghĩa "chậm lại chút" mà
	// là "quay lại sau" — một tình huống KHÁC, và câu trả lời đúng cho nó là
	// chuyển route chứ không phải ngồi im.
	TranMotLanCho = 30 * time.Second

	// TranTongCho là trần cho TỔNG thời gian chờ của cả lời gọi.
	//
	// Vì sao 60 giây: `Goi` đã chốt 120 giây là hạn cho một lời gọi — con số duy
	// nhất dự án đã cam kết cho "một lượt lâu tới đâu thì vẫn coi là đang chạy".
	// Phần chờ thêm lấy đúng MỘT NỬA hạn đó, để một lượt có thử lại vẫn không
	// vượt quá mức kiên nhẫn của hai lượt thường, và vẫn còn dư ngân sách thời
	// gian cho route dự phòng chạy sau khi ta bỏ cuộc.
	TranTongCho = 60 * time.Second

	// luiDanGoc là bậc lùi đầu tiên khi KHÔNG có `Retry-After` đọc được: 1s, rồi
	// 2s. Nhân đôi chứ không cố định, vì cái ta không biết là hạn mức tính theo
	// cửa sổ dài bao lâu; tăng dần là cách dò mà không cần đoán.
	//
	// KHÔNG có jitter ngẫu nhiên. Nói rõ để không ai tưởng là đã có: hạm đội chạy
	// nhiều agent song song vào CÙNG một nhà bán lại thì các lần lùi này trùng
	// nhịp nhau. Xem docs/BAO-CAO-429.md mục 3.
	luiDanGoc = 1 * time.Second
)

// LanChoLai ghi lại MỘT lần chờ rồi thử lại. Có mặt để không lần chờ nào là im
// lặng: người đọc phải thấy được đã chờ bao lâu, và con số đó ở đâu ra.
type LanChoLai struct {
	Lan    int           // lần thử lại thứ mấy, đếm từ 1
	Status int           // mã HTTP đã gây ra lần chờ này (429)
	Header string        // NGUYÊN VĂN `Retry-After`; rỗng nếu nhà cung cấp không gửi
	Cho    time.Duration // đã chờ thật bao lâu
	Nguon  string        // con số kia ở đâu ra — xem các hằng Nguon* dưới đây
}

// Nguồn của thời gian chờ. Bắt buộc phải phân biệt bốn thứ này, vì yêu cầu của
// dự án là KHÔNG được đoán bừa một con số rồi im lặng: đọc được header và tự lùi
// dần là hai chuyện khác hẳn nhau về độ tin cậy, và người đọc log phải biết mình
// đang nhìn cái nào.
const (
	NguonGiay    = "Retry-After: số giây"
	NguonMoc     = "Retry-After: mốc thời gian HTTP-date"
	NguonThieu   = "tự lùi dần - nhà cung cấp KHÔNG gửi Retry-After"
	NguonKhongRo = "tự lùi dần - Retry-After có mà KHÔNG đọc được"
)

// docRetryAfter đọc header `Retry-After` theo ĐÚNG hai dạng RFC 9110 cho phép.
//
// Dạng thứ hai (HTTP-date) là dạng hay bị quên nhất, và quên nó thì hỏng theo
// kiểu tệ nhất: `strconv.Atoi("Wed, 21 Oct 2026 07:28:00 GMT")` trả lỗi, mã coi
// như "không có header", rồi tự lùi 1 giây — trong khi nhà cung cấp vừa nói rõ
// phải chờ tới sáng. Tự lùi 1 giây ở đó là gọi lại vào đúng bức tường vừa dựng.
//
// Nhận `bayGio` từ ngoài chứ không gọi time.Now() bên trong: dạng mốc thời gian
// chỉ kiểm được nếu "bây giờ" là thứ test đặt được.
//
// Trả về: thời gian chờ, nguồn của nó, và đọc-được-hay-không.
func docRetryAfter(v string, bayGio time.Time) (time.Duration, string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, NguonThieu, false
	}
	// Dạng 1: số giây nguyên. RFC cho phép cả 0 (thử lại ngay).
	if n, err := strconv.Atoi(v); err == nil {
		if n < 0 {
			// Số âm là header hỏng, không phải "chờ ngược thời gian". Coi như
			// không đọc được, đừng tự tiện đổi dấu.
			return 0, NguonKhongRo, false
		}
		return time.Duration(n) * time.Second, NguonGiay, true
	}
	// Dạng 2: HTTP-date. http.ParseTime nhận cả ba định dạng HTTP cho phép
	// (RFC 1123, RFC 850, asctime) — tự viết lại bằng time.Parse là chắc chắn
	// thiếu một dạng.
	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(bayGio)
		if d < 0 {
			// Mốc đã trôi qua: đồng hồ hai bên lệch, hoặc phản hồi tới muộn.
			// Không phải lỗi — nghĩa là "chờ 0 giây", thử lại được ngay.
			d = 0
		}
		return d, NguonMoc, true
	}
	return 0, NguonKhongRo, false
}

// tinhCho quyết định lần thử lại thứ `lan` phải chờ bao lâu.
//
// `daCho` là tổng đã chờ trước đó, để giữ trần tổng. Trả về `nen=false` nghĩa là
// THÔI, đừng chờ nữa — vượt trần.
func tinhCho(h string, lan int, daCho time.Duration, bayGio time.Time) (time.Duration, string, bool) {
	d, nguon, docDuoc := docRetryAfter(h, bayGio)
	if !docDuoc {
		// Đường lui: lùi dần theo lần thử. 1s, 2s, 4s...
		d = luiDanGoc << (lan - 1)
	}
	if d > TranMotLanCho {
		return d, nguon, false
	}
	if daCho+d > TranTongCho {
		return d, nguon, false
	}
	return d, nguon, true
}

// cho ngủ `d`, nhưng tỉnh ngay nếu ngữ cảnh bị huỷ.
//
// Không dùng time.Sleep: một lượt bị người dùng Ctrl-C hay bị hết hạn ctx mà vẫn
// nằm ngủ 30 giây là kiểu treo mà không log nào giải thích nổi.
func cho(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	// Chờ quá hạn chót của ctx thì chờ cũng vô ích — hỏng nhanh còn hơn hỏng muộn,
	// vì tầng trên còn phải kịp gọi route dự phòng trong phần hạn còn lại.
	if hc, co := ctx.Deadline(); co && time.Now().Add(d).After(hc) {
		return fmt.Errorf("chờ %s thì vượt hạn chót của lượt", d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// goiCoChoLai chạy một lời gọi HTTP, gặp 429 thì chờ theo `Retry-After` rồi thử
// LẠI CHÍNH route đó, tối đa `SoLanThuLai` lần.
//
// `taoReq` phải dựng một *http.Request MỚI mỗi lần: thân request là một Reader
// đã đọc cạn sau lần gửi đầu, dùng lại là gửi đi một thân rỗng — hỏng im lặng,
// và nhà cung cấp sẽ trả 400 chứ không nói gì về nguyên nhân thật.
//
// Trả về phản hồi CUỐI CÙNG (có thể vẫn là 429 — bên gọi tự xử), kèm nhật ký mọi
// lần chờ. Nhật ký này KHÔNG được vứt: nó là thứ trả lời "vì sao lượt này mất 34
// giây" mà không có nó thì chỉ đoán được.
//
// CHỈ AN TOÀN VỚI 429 TRƯỚC KHI CÓ CHỮ NÀO. 429 luôn tới ở dòng trạng thái, tức
// trước mẩu SSE đầu tiên, nên thử lại không nhân đôi nội dung. Một stream đứt ở
// giữa thì TUYỆT ĐỐI không được thử lại theo đường này — sẽ ghép hai nửa câu trả
// lời khác nhau. Đó là lý do hàm này bọc lời gọi, chứ không bọc cả vòng đọc SSE.
func goiCoChoLai(ctx context.Context, cl *http.Client, taoReq func() (*http.Request, error)) (*http.Response, []LanChoLai, error) {
	var nhatKy []LanChoLai
	var daCho time.Duration

	for lan := 0; ; lan++ {
		req, err := taoReq()
		if err != nil {
			return nil, nhatKy, err
		}
		resp, err := cl.Do(req)
		if err != nil {
			return nil, nhatKy, err
		}
		if resp.StatusCode != http.StatusTooManyRequests || lan >= SoLanThuLai {
			return resp, nhatKy, nil
		}

		h := resp.Header.Get("Retry-After")
		d, nguon, nen := tinhCho(h, lan+1, daCho, time.Now())
		if !nen {
			// Vượt trần: trả về CHÍNH phản hồi 429 này để bên gọi giữ nguyên văn
			// thân lỗi. Ghi lại lần "định chờ nhưng không chờ" với Cho=0, để nhật
			// ký nói rõ đã bỏ cuộc chứ không phải chưa từng thấy header.
			nhatKy = append(nhatKy, LanChoLai{
				Lan: lan + 1, Status: resp.StatusCode, Header: h, Cho: 0,
				Nguon: fmt.Sprintf("%s - KHÔNG chờ: %s vượt trần (một lần %s, tổng %s)",
					nguon, d, TranMotLanCho, TranTongCho),
			})
			return resp, nhatKy, nil
		}

		// Đọc cạn rồi đóng thân của lần hỏng: bỏ qua là rò kết nối, và lần thử
		// lại sẽ mở socket mới thay vì dùng lại socket cũ.
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()

		if err := cho(ctx, d); err != nil {
			// Không chờ được nữa (ctx huỷ / hết hạn). Không còn phản hồi trong
			// tay vì vừa đóng thân, nên trả lỗi kèm nhật ký — bên gọi dựng câu.
			nhatKy = append(nhatKy, LanChoLai{
				Lan: lan + 1, Status: http.StatusTooManyRequests, Header: h, Cho: 0,
				Nguon: nguon + " - KHÔNG chờ được: " + err.Error(),
			})
			return nil, nhatKy, err
		}
		daCho += d
		nhatKy = append(nhatKy, LanChoLai{
			Lan: lan + 1, Status: http.StatusTooManyRequests, Header: h, Cho: d, Nguon: nguon,
		})
	}
}

// MoTaChoLai dựng câu kể lại mọi lần chờ, để ghép vào lỗi trả cho người đọc.
//
// Rỗng nếu không có lần chờ nào — lượt bình thường không phải đọc thêm gì.
func MoTaChoLai(nk []LanChoLai) string {
	if len(nk) == 0 {
		return ""
	}
	var b strings.Builder
	var tong time.Duration
	for _, l := range nk {
		tong += l.Cho
	}
	fmt.Fprintf(&b, "đã thử lại %d lần, chờ tổng %s:", len(nk), tong)
	for _, l := range nk {
		h := l.Header
		if h == "" {
			h = "(không có)"
		}
		fmt.Fprintf(&b, "\n     - lần %d: HTTP %d, Retry-After: %s -> chờ %s [%s]",
			l.Lan, l.Status, h, l.Cho, l.Nguon)
	}
	return b.String()
}
