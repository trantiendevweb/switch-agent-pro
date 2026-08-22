// CỔNG trần đồng thời cho đường FLOW: xin không được thì CHỜ, không từ chối.
package fleet

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// VÌ SAO CÓ FILE NÀY
//
// tran.go trả lời câu "lúc này được cấp mấy phiên" — một phép tính THUẦN, không
// trạng thái, không chờ. Đúng thứ `FleetStart` cần: người vận hành đang đứng ở
// terminal, trần chặn thì nói ngay để họ chọn tài khoản khác.
//
// Đường FLOW cần thứ khác. Một lượt flow đêm dài có bốn bước `agent` cùng đợt,
// cả bốn khai `profile = "claude:tns"`. Trả lời "hết chỗ" cho bước thứ ba nghĩa
// là GIẾT cả lượt chạy lúc 2 giờ sáng — trong khi chỉ cần đợi bước thứ nhất
// xong là chạy được. Chậm còn hơn chết.
//
// Nên Cổng là phần THIẾU: cùng phép tính, nhưng có hàng đợi.
//
// KHÔNG đếm lại từ đầu. Mọi quyết định ở đây đều đi qua XetTran của tran.go —
// hai bản đếm song song sẽ lệch nhau trong im lặng, và lúc đó không ai biết
// con số nào đúng. Cổng chỉ thêm ba thứ tran.go cố ý không có: hàng đợi, lời
// nói lúc chờ, và cái chốt chống treo im lặng.
//
// PHÉP CỘNG PHIÊN — chỗ dễ sai nhất:
//
//	đang chạy = nền (phiên NGOÀI cổng, đọc từ sổ) + chỗ cổng đang giữ
//
// "Nền" chụp MỘT LẦN lúc mở cổng, cố ý. Đọc lại sổ mỗi lượt xét thì phiên do
// chính cổng bật ra sẽ bị đếm HAI LẦN (một lần ở sổ, một lần ở chỗ đang giữ),
// và trần hồ sơ 2 hoá thành 1 mà không ai hiểu vì sao. Chỉ có đúng một lúc đọc
// lại sổ là an toàn: khi cổng KHÔNG giữ chỗ nào — lúc đó sổ không thể chứa
// phiên của cổng. Xem lamMoiNen.
// ---------------------------------------------------------------------------

// The là thẻ giữ chỗ. Trả lại bằng Tra khi bước chạy xong.
type The struct {
	cong *Cong
	id   int64

	// Cap là số phiên ĐƯỢC CẤP, có thể NHỎ HƠN số đã xin khi trần không bao giờ
	// đủ rộng cho số đó (ví dụ `copies = 3` mà trần hồ sơ là 2). Người gọi phải
	// dùng con số này chứ không dùng lại số mình xin.
	Cap  int
	Addr string
}

// Tra trả chỗ và đánh thức các bước đang xếp hàng. Gọi nhiều lần vô hại — hợp
// với `defer` ở chỗ gọi.
func (t *The) Tra() {
	if t == nil || t.cong == nil {
		return
	}
	c := t.cong
	c.mu.Lock()
	delete(c.giu, t.id)
	t.cong = nil
	c.danhThuc()
	c.mu.Unlock()
}

// oGiu là một chỗ đang bị giữ.
type oGiu struct {
	phien Phien
	so    int
	ten   string // tên bước, để nói được "ai đang giữ"
	tu    time.Time
}

// Cong là cổng trần đồng thời có hàng đợi.
//
// Con trỏ nil là hợp lệ và nghĩa là KHÔNG áp trần — mọi phương thức đều chịu
// được nil. Nhờ vậy test cũ của gói flow chạy nguyên như trước, và một đường
// nào chưa cắm cổng thì hỏng theo hướng cũ chứ không panic.
type Cong struct {
	mu   sync.Mutex
	tran Tran
	nen  []Phien
	giu  map[int64]oGiu
	ke   int64
	cho  int           // số bước đang xếp hàng, để nói ra trong tin báo
	doi  chan struct{} // đóng lại mỗi lần có chỗ trả về

	docNen func() []Phien
	bao    func(string)

	// Nhip là khoảng nhắc lại khi một bước vẫn còn chờ. Người vận hành đọc dòng
	// này lúc 2 giờ sáng: im quá thì tưởng treo, dày quá thì trôi mất mọi thứ
	// khác trong nhật ký.
	Nhip time.Duration

	// ChoKetCung là mức KIÊN NHẪN với ca nghi kẹt cứng trước khi bỏ cuộc.
	//
	// Vì sao không bỏ cuộc ngay, và đây là số đo chứ không phải phỏng đoán: bản
	// đầu của file này báo lỗi ngay khi thấy hình dạng kẹt cứng. Đo thật trên
	// máy (22/08, lượt #9 của dự án đo): một lượt `sagent fleet` bên ngoài đang
	// giữ hai chỗ và sẽ trả lại sau khoảng MỘT GIÂY, thế mà cả bốn bước của lượt
	// flow bị giết sạch. Đúng cái kết cục mà bản sửa này lập ra để chống.
	//
	// Nhưng cũng không được chờ vô hạn: chỗ bị chiếm có thể đến từ phiên ĐÃ CHẾT
	// mà sổ vẫn ghi là đang chạy, và chờ một cái xác thì chờ tới sáng. Nên: nói
	// to ngay từ lúc nghi, nhắc lại theo Nhip, và bỏ cuộc sau ChoKetCung.
	ChoKetCung time.Duration
}

// NhipMacDinh là khoảng nhắc lại khi chưa ai đặt.
const NhipMacDinh = 30 * time.Second

// NhipDoNgoai là nhịp ĐỌC LẠI SỔ khi đang nghi kẹt cứng.
//
// Ngắn hơn Nhip vì hai việc khác nhau: Nhip là nhịp NÓI cho người nghe, còn đây
// là nhịp NHÌN sổ. Lúc chỗ trống chỉ có thể đến từ ngoài thì không ai đánh thức
// ta, và nhìn thưa quá là bắt cả lượt chạy đứng thêm nửa phút cho một thứ đã
// xong từ lâu. Hai giây: một lượt đọc sổ SQLite rẻ hơn nhiều so với nửa phút
// chết đứng, mà vẫn không thành vòng lặp nóng.
const NhipDoNgoai = 2 * time.Second

// ChoKetCungMacDinh là mức kiên nhẫn mặc định với ca nghi kẹt cứng.
//
// 15 phút là LỰA CHỌN, không phải số đo: đủ dài để một lượt hạm đội bên ngoài
// làm xong việc rồi trả chỗ, đủ ngắn để một cái xác trong sổ không nuốt cả đêm.
// Trong suốt 15 phút đó bước KHÔNG im lặng — cứ mỗi Nhip lại nói ra nó đang chờ
// ai và sẽ bỏ cuộc lúc nào.
const ChoKetCungMacDinh = 15 * time.Minute

// MoCong mở một cổng cho MỘT lượt chạy flow.
//
// docNen đọc các phiên đang chạy ngoài cổng (thường là sổ phiên). nil = coi như
// không có phiên nào ngoài. bao là chỗ nói ra — nil = im, và im ở đây nghĩa là
// một bước chờ nửa tiếng mà không dòng nào giải thích, nên đừng truyền nil.
func MoCong(t Tran, docNen func() []Phien, bao func(string)) *Cong {
	c := &Cong{
		tran:       t,
		giu:        map[int64]oGiu{},
		doi:        make(chan struct{}),
		docNen:     docNen,
		bao:        bao,
		Nhip:       NhipMacDinh,
		ChoKetCung: ChoKetCungMacDinh,
	}
	c.nen = c.docNenAnToan()
	return c
}

func (c *Cong) docNenAnToan() []Phien {
	if c.docNen == nil {
		return nil
	}
	return c.docNen()
}

// lamMoiNen đọc lại sổ phiên. CHỈ được gọi khi cổng không giữ chỗ nào — xem
// khối chú thích đầu file về phép cộng phiên.
func (c *Cong) lamMoiNen() {
	if len(c.giu) != 0 {
		return
	}
	c.nen = c.docNenAnToan()
}

func (c *Cong) danhThuc() {
	close(c.doi)
	c.doi = make(chan struct{})
}

// dangChay = nền + chỗ đang giữ, dàn phẳng thành danh sách phiên cho XetTran.
func (c *Cong) dangChay() []Phien {
	out := make([]Phien, 0, len(c.nen)+len(c.giu))
	out = append(out, c.nen...)
	for _, o := range c.giu {
		for i := 0; i < o.so; i++ {
			out = append(out, o.phien)
		}
	}
	return out
}

func (c *Cong) noi(s string) {
	if c.bao == nil || s == "" {
		return
	}
	c.bao(s)
}

// Xin xin chỗ cho một bước, CHỜ tới khi có.
//
// ten là tên bước để in trong tin báo. Trả về thẻ giữ chỗ; người gọi phải
// `defer the.Tra()`.
//
// Ba đường ra:
//   - có chỗ ngay hoặc sau khi chờ -> thẻ, err nil
//   - ctx bị huỷ                   -> nil, ctx.Err()
//   - KẸT CỨNG                     -> nil, lỗi kèm chẩn đoán đầy đủ
//
// KẸT CỨNG là ca DUY NHẤT Cổng từ chối, và nó có định nghĩa hẹp: cổng KHÔNG giữ
// chỗ nào (nên trong lượt flow này không có gì sẽ trả chỗ ra), sổ phiên đọc lại
// vẫn chặn, VÀ tình trạng đó kéo dài quá ChoKetCung. Nói ra ngay từ lúc mới
// nghi, chứ không đợi hết kiên nhẫn mới mở miệng — xem tinNghiKetCung.
func (c *Cong) Xin(ctx context.Context, ten string, xin Phien, muon int) (*The, error) {
	if c == nil {
		return nil, nil
	}
	if muon < 1 {
		muon = 1
	}
	nhip := c.Nhip
	if nhip <= 0 {
		nhip = NhipMacDinh
	}
	choKet := c.ChoKetCung
	if choKet <= 0 {
		choKet = ChoKetCungMacDinh
	}
	batDau := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	daBaoCat, daXepHang := false, false
	lanBao := time.Time{}
	nghiTu := time.Time{} // lúc đầu tiên thấy hình dạng kẹt cứng
	defer func() {
		if daXepHang {
			c.cho--
		}
	}()

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Bước 1 — cổng RỖNG thì trần cho tối đa mấy phiên? Con số này không phụ
		// thuộc vào ai đang chạy TRONG cổng, nên nó nói được thứ việc chờ không
		// bao giờ nói được: chờ mãi có ích hay không.
		rong := XetTran(c.tran, c.nen, xin, muon)
		if rong.Cap == 0 && len(c.giu) == 0 {
			c.lamMoiNen() // an toàn ĐÚNG lúc này: cổng không giữ chỗ nào
			rong = XetTran(c.tran, c.nen, xin, muon)
		}

		// Hình dạng KẸT CỨNG: không còn chỗ, và cổng không giữ chỗ nào nên trong
		// lượt flow này không có gì sẽ trả chỗ ra.
		//
		// "Hình dạng", chưa phải kết luận. Chỗ đang bị chiếm là phiên NGOÀI cổng,
		// và phiên ngoài thì hoặc sẽ kết thúc (một lượt hạm đội khác — chờ là
		// đúng), hoặc không bao giờ (phiên đã chết mà sổ vẫn ghi đang chạy — chờ
		// là treo tới sáng). Đứng ở đây không phân biệt được hai ca đó, nên: nói
		// to ngay, và đặt hạn cho sự kiên nhẫn.
		if rong.Cap == 0 && len(c.giu) == 0 {
			if nghiTu.IsZero() {
				nghiTu = time.Now()
				c.noi(c.tinNghiKetCung(ten, rong, choKet))
				lanBao = time.Now()
			}
			if time.Since(nghiTu) >= choKet {
				return nil, c.loiKetCung(ten, rong, time.Since(nghiTu))
			}
		} else {
			nghiTu = time.Time{} // chỗ đã nhúc nhích: đếm lại từ đầu
		}

		// Bước 2 — xin nhiều hơn mức trần BAO GIỜ cũng cho thì cắt xuống, chứ
		// đừng xếp hàng chờ một con số không bao giờ tới. Cắt rồi mới chờ.
		muonThuc := muon
		if rong.Cap > 0 && rong.Cap < muon {
			muonThuc = rong.Cap
			if !daBaoCat {
				c.noi(ten + ": " + rong.LoiCatBot())
				daBaoCat = true
			}
		}

		// Bước 3 — còn chỗ THẬT lúc này không?
		k := XetTran(c.tran, c.dangChay(), xin, muonThuc)
		if k.Cap >= muonThuc {
			c.ke++
			id := c.ke
			c.giu[id] = oGiu{phien: xin, so: muonThuc, ten: ten, tu: time.Now()}
			if daXepHang {
				c.noi(fmt.Sprintf("%s CHẠY TIẾP sau %s xếp hàng — được cấp %d chỗ ở %s. Bốn trần lúc này — %s.",
					ten, gonThoiGian(time.Since(batDau)), muonThuc, xin.Addr(), k.Bang()))
			}
			return &The{cong: c, id: id, Cap: muonThuc, Addr: xin.Addr()}, nil
		}

		// Bước 4 — chờ. Nói ra NGAY lần đầu rồi nhắc lại theo nhịp: một bước
		// đứng im mà nhật ký không có dòng nào là đúng thứ khiến người trực đêm
		// đi giết tiến trình.
		if !daXepHang {
			c.cho++
			daXepHang = true
		}
		if lanBao.IsZero() || time.Since(lanBao) >= nhip {
			c.noi(c.tinCho(ten, k, time.Since(batDau)))
			lanBao = time.Now()
		}

		// Ngủ bao lâu trước khi xét lại.
		//
		// Bình thường Nhip là đủ: chỗ trống sẽ đến từ một bước cùng lượt trả ra,
		// mà việc đó ĐÁNH THỨC hàng đợi ngay lập tức qua kênh `doi` — hẹn giờ chỉ
		// là để nhắc lại cho người đọc.
		//
		// Ca nghi kẹt cứng thì ngược hẳn: chỗ trống sẽ đến từ NGOÀI, và không ai
		// đánh thức ta cả — chỉ có tự đi đọc lại sổ mới biết. Đo thật (22/08):
		// lượt hạm đội ngoài xong sau ~2 giây, mà bốn bước đứng đủ 30 giây mới
		// nhúc nhích, vì đó là lúc hẹn giờ kêu. Nên ca này dò dày hơn.
		ngu := nhip
		if !nghiTu.IsZero() && ngu > NhipDoNgoai {
			ngu = NhipDoNgoai
		}
		ch := c.doi
		c.mu.Unlock()
		hen := time.NewTimer(ngu)
		select {
		case <-ch:
		case <-hen.C:
		case <-ctx.Done():
		}
		hen.Stop()
		c.mu.Lock()
	}
}

// tinCho là dòng người trực đêm đọc. Bốn thứ, đúng thứ tự người ta cần: bước
// nào đang đứng, TRẦN NÀO chặn và còn mấy chỗ, đã chờ bao lâu, ai đang giữ chỗ.
// Thiếu cái cuối thì đọc xong vẫn phải đi mở `sagent status`.
func (c *Cong) tinCho(ten string, k KetTran, lau time.Duration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s CHỜ TRẦN ĐỒNG THỜI (đã chờ %s)", ten, gonThoiGian(lau))
	if chat := k.Chat(); len(chat) > 0 {
		m := chat[0]
		fmt.Fprintf(&b, ": chật nhất là %s — trần %d, đang chạy %d, còn %d chỗ",
			m.Nhan(), m.Tran, m.Dang, m.Con())
	} else {
		fmt.Fprintf(&b, ": xin %d chỗ ở %s", k.Xin, k.Addr)
	}
	fmt.Fprintf(&b, ". Bốn trần lúc này — %s.", k.Bang())
	if g := c.aiDangGiu(); g != "" {
		fmt.Fprintf(&b, " Đang giữ chỗ: %s.", g)
	}
	if c.cho > 1 {
		fmt.Fprintf(&b, " %d bước khác cũng đang xếp hàng.", c.cho-1)
	}
	b.WriteString(" Bước KHÔNG bị huỷ — nó tự chạy tiếp khi có chỗ.")
	return b.String()
}

// aiDangGiu liệt kê các bước đang giữ chỗ kèm thời gian đã giữ.
func (c *Cong) aiDangGiu() string {
	if len(c.giu) == 0 {
		return ""
	}
	ds := make([]string, 0, len(c.giu))
	for _, o := range c.giu {
		ds = append(ds, fmt.Sprintf("%s (%s, %d chỗ, %s)",
			o.ten, o.phien.Addr(), o.so, gonThoiGian(time.Since(o.tu))))
	}
	sapChuoi(ds)
	return strings.Join(ds, ", ")
}

// tinNghiKetCung là tiếng chuông đầu tiên, phát NGAY lúc thấy hình dạng kẹt
// cứng chứ không đợi hết kiên nhẫn. Người trực đêm cần biết ngay là lượt chạy
// đang đứng vì thứ nằm NGOÀI nó, và đứng tới bao giờ thì bỏ cuộc.
func (c *Cong) tinNghiKetCung(ten string, k KetTran, choKet time.Duration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s ĐỨNG vì hết chỗ ở %s, mà lượt flow này KHÔNG có bước nào đang chạy để trả chỗ ra "+
		"— chỗ đang bị chiếm bởi thứ NGOÀI lượt chạy (một lượt `sagent fleet` khác, "+
		"hoặc phiên đã chết mà sổ vẫn ghi là đang chạy).", ten, k.Addr)
	fmt.Fprintf(&b, " Bốn trần lúc này — %s.", k.Bang())
	fmt.Fprintf(&b, " Sẽ chờ tối đa %s rồi mới bỏ cuộc. Trong lúc chờ: `sagent status` xem phiên nào đang giữ, "+
		"`sagent quet` soi tiến trình mồ côi.", gonThoiGian(choKet))
	return b.String()
}

// loiKetCung là lời từ chối DUY NHẤT của cổng — nên nó phải nói được vì sao chờ
// tiếp cũng vô ích, nếu không người đọc sẽ tưởng đây chỉ là một lần xui.
func (c *Cong) loiKetCung(ten string, k KetTran, daCho time.Duration) error {
	var b strings.Builder
	fmt.Fprintf(&b, "KẸT CỨNG ở trần đồng thời: %s xin %d chỗ ở %s mà không còn chỗ nào, "+
		"và suốt %s KHÔNG bước nào của lượt flow này đang chạy để trả chỗ ra.\n  ",
		ten, k.Xin, k.Addr, gonThoiGian(daCho))
	b.WriteString(strings.TrimPrefix(k.LoiHetCho().Error(), "hết chỗ cho "+k.Addr+": "))
	b.WriteString("\n  Chỗ bị chiếm đến từ NGOÀI lượt flow này (một lượt `sagent fleet` khác), " +
		"hoặc từ phiên đã chết mà sổ vẫn ghi là đang chạy — `sagent quet` soi tiến trình mồ côi.")
	return fmt.Errorf("%s", b.String())
}

// PhienTu tách "claude:tns" thành Phien. Thiếu phần provider thì mặc định là
// claude — cùng quy ước với địa chỉ hồ sơ ở mọi mặt.
//
// Ở ĐÂY chứ không phải mỗi gói một bản: cổng phải đếm đúng cái địa chỉ mà bộ
// chạy thật sẽ dùng. Hai cách tách lệch nhau một chút là hai cái tên khác nhau,
// và trần sẽ đi canh một tài khoản không ai chạy.
func PhienTu(s string) Phien {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return Phien{Provider: s[:i], Account: s[i+1:]}
	}
	return Phien{Provider: "claude", Account: s}
}

// gonThoiGian in "1m30s" thay vì "1m30.000481s" — dòng nhật ký đọc bằng mắt.
func gonThoiGian(d time.Duration) string {
	if d < time.Second {
		return d.Truncate(time.Millisecond).String()
	}
	return d.Truncate(time.Second).String()
}

func sapChuoi(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
