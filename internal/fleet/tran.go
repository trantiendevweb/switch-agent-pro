package fleet

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// TRẦN ĐỒNG THỜI THEO TỪNG CHIỀU
//
// Trước đây chỉ có MỘT con số: `policy.max_parallel_sessions`. Nó đếm tổng số
// phiên đang chạy, và chỉ thế. Nghĩa là bốn phiên rơi hết vào MỘT tài khoản
// claude vẫn hợp lệ — chúng đốt sạch hạn mức thuê bao của tài khoản đó trong
// khi các tài khoản khác ngồi không. Trần chung không nhìn thấy chuyện đó vì
// nó không biết phiên nào thuộc về ai.
//
// Ba chiều dưới đây đo ba thứ KHÁC NHAU, nên phải là ba con số khác nhau:
//
//   - HỒ SƠ (tài khoản): hạn mức thuê bao tính theo TÀI KHOẢN. Hai phiên
//     `claude:tns` tiêu gấp đôi hạn mức của đúng tài khoản đó; cùng lúc ấy
//     `claude:phu` vẫn còn nguyên. Đây là chiều đắt nhất khi vượt, vì hết hạn
//     mức thì phải chờ tới mốc cấp lại chứ không mua thêm được bằng RAM.
//   - PROVIDER: nhà cung cấp còn siết ở tầng tổ chức/nhà bán lại, tức là trần
//     nằm trên TỔNG các tài khoản chứ không riêng từng tài khoản.
//   - HARNESS: tiến trình CLI chạy trên máy này. Chiều này đo RAM và số tiến
//     trình con, không liên quan gì tới hạn mức — dừng hết tài khoản trả phí
//     rồi chạy toàn tài khoản miễn phí thì máy vẫn sập như thường.
//
// Bốn trần CỘNG DỒN, không thay thế nhau: một lượt chạy phải qua được cả bốn.
// `policy.max_parallel_sessions` vẫn là trần ngoài cùng và vẫn có hiệu lực.
// ---------------------------------------------------------------------------

// Tran là bộ trần đã khai, ở dạng dữ liệu thuần.
//
// Cố ý KHÔNG nhận `config.Config`: gói fleet không được phụ thuộc gói config,
// và dữ liệu thuần thì test dựng được bằng tay mà không phải ghi file TOML ra
// đĩa.
//
// Số 0 ở mọi trường số nghĩa là TẮT chiều đó, không phải "cấm chạy". Cùng quy
// ước với `policy.max_parallel_sessions` vốn có (`m > 0` mới áp).
type Tran struct {
	Chung int // policy.max_parallel_sessions

	HarnessMacDinh  int
	ProviderMacDinh int
	HoSoMacDinh     int

	Harness  map[string]int // tên harness  -> trần riêng
	Provider map[string]int // tên provider -> trần riêng
	HoSo     map[string]int // "claude:tns" -> trần riêng

	// ThuocHarness gộp nhiều provider vào MỘT harness.
	//
	// Đo được trên máy này (22/08): năm provider ra năm binary khác nhau —
	// claude.exe, codex, cursor-agent, agy.exe, grok. Nên MẶC ĐỊNH harness của
	// một provider là chính tên provider đó, và khi không ai khai bảng này thì
	// trần harness với trần provider đếm đúng cùng một tập phiên (mức chật hơn
	// thắng). Hai chiều chỉ TÁCH RA khi có khai gộp — ví dụ codex và grok đều
	// là gói npm chạy trên node và người vận hành muốn một trần chung cho cả
	// hai. Không khai bừa quan hệ gộp ở đây: mã không tự đoán được binary nào
	// dùng chung runtime nào.
	ThuocHarness map[string]string
}

// HarnessCua trả tên harness của một provider. Không khai gộp thì harness
// chính là tên provider.
func (t Tran) HarnessCua(provider string) string {
	if h, ok := t.ThuocHarness[provider]; ok && strings.TrimSpace(h) != "" {
		return h
	}
	return provider
}

func (t Tran) tranHarness(ten string) int {
	if v, ok := t.Harness[ten]; ok {
		return v
	}
	return t.HarnessMacDinh
}

func (t Tran) tranProvider(ten string) int {
	if v, ok := t.Provider[ten]; ok {
		return v
	}
	return t.ProviderMacDinh
}

func (t Tran) tranHoSo(addr string) int {
	if v, ok := t.HoSo[addr]; ok {
		return v
	}
	return t.HoSoMacDinh
}

// Phien là phần tối thiểu của một phiên cần cho phép đếm.
type Phien struct {
	Provider string
	Account  string
}

// Addr là địa chỉ hồ sơ, ví dụ "claude:tns". Không kèm số bản clone: các bản
// clone của cùng một tài khoản dùng CHUNG hạn mức, nên chúng phải đếm chung.
func (p Phien) Addr() string { return p.Provider + ":" + p.Account }

// Muc là một mức trần đã quy ra số cụ thể cho lượt chạy này.
type Muc struct {
	Loai string // "chung" | "harness" | "provider" | "hồ sơ"
	Ten  string // "claude", "claude:tns"; rỗng với mức chung
	Tran int    // 0 = tắt
	Dang int    // đang chạy, tính theo đúng chiều này
	Khoa string // khoá TOML để nới
}

// Tat: chiều này không áp trần.
func (m Muc) Tat() bool { return m.Tran <= 0 }

// Con là số chỗ còn lại, hoặc -1 khi chiều này tắt.
//
// Không bao giờ trả số âm khác -1: trần bị hạ xuống dưới số đang chạy là
// chuyện có thật (sửa file lúc đang có phiên chạy), và số âm chỉ làm phép tính
// phía sau sai chứ không nói thêm được gì.
func (m Muc) Con() int {
	if m.Tat() {
		return -1
	}
	if c := m.Tran - m.Dang; c > 0 {
		return c
	}
	return 0
}

// Nhan là tên đọc được: "chung", "harness claude", "hồ sơ claude:tns".
func (m Muc) Nhan() string {
	if m.Ten == "" {
		return m.Loai
	}
	return m.Loai + " " + m.Ten
}

// MotDong là một mục trong bảng liệt kê bốn trần.
func (m Muc) MotDong() string {
	if m.Tat() {
		return fmt.Sprintf("%s: TẮT (%s = 0)", m.Nhan(), m.Khoa)
	}
	return fmt.Sprintf("%s: trần %d, đang chạy %d, còn %d", m.Nhan(), m.Tran, m.Dang, m.Con())
}

// KetTran là kết quả xét trần cho một lượt xin chạy.
type KetTran struct {
	Xin  int   // số phiên người dùng xin
	Cap  int   // số phiên được cấp (0 = từ chối hẳn)
	Mucs []Muc // đủ bốn chiều, theo thứ tự chung -> harness -> provider -> hồ sơ
	Addr string
}

// Chat trả các mức đã cắt lượt này xuống, mức chật nhất đứng đầu.
func (k KetTran) Chat() []Muc {
	var out []Muc
	for _, m := range k.Mucs {
		if !m.Tat() && m.Con() < k.Xin {
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Con() < out[j].Con() })
	return out
}

// Du: cấp đủ số đã xin, không phải nói gì cả.
func (k KetTran) Du() bool { return k.Cap >= k.Xin }

// Bang in bốn trần thành một dòng, để người đọc thấy NGAY còn chỗ ở đâu thay
// vì phải chạy thêm lệnh khác rồi tự đoán.
func (k KetTran) Bang() string {
	parts := make([]string, 0, len(k.Mucs))
	for _, m := range k.Mucs {
		parts = append(parts, m.MotDong())
	}
	return strings.Join(parts, " · ")
}

// XetTran tính xem lượt này được cấp mấy phiên.
//
// dangChay là các phiên ĐANG CHẠY, kể cả của tài khoản khác và provider khác —
// trần chung và trần harness cần chúng. xin là hồ sơ sắp bật, muon là số bản.
func XetTran(t Tran, dangChay []Phien, xin Phien, muon int) KetTran {
	if muon < 1 {
		muon = 1
	}
	harness := t.HarnessCua(xin.Provider)

	var dChung, dHarness, dProvider, dHoSo int
	for _, p := range dangChay {
		dChung++
		if t.HarnessCua(p.Provider) == harness {
			dHarness++
		}
		if p.Provider == xin.Provider {
			dProvider++
		}
		if p.Addr() == xin.Addr() {
			dHoSo++
		}
	}

	k := KetTran{Xin: muon, Addr: xin.Addr()}
	k.Mucs = []Muc{
		{Loai: "chung", Tran: t.Chung, Dang: dChung, Khoa: "policy.max_parallel_sessions"},
		{Loai: "harness", Ten: harness, Tran: t.tranHarness(harness), Dang: dHarness,
			Khoa: khoaBang("policy.tran.harness", harness)},
		{Loai: "provider", Ten: xin.Provider, Tran: t.tranProvider(xin.Provider), Dang: dProvider,
			Khoa: khoaBang("policy.tran.provider", xin.Provider)},
		{Loai: "hồ sơ", Ten: xin.Addr(), Tran: t.tranHoSo(xin.Addr()), Dang: dHoSo,
			Khoa: khoaBang("policy.tran.ho_so", xin.Addr())},
	}

	k.Cap = muon
	for _, m := range k.Mucs {
		if m.Tat() {
			continue
		}
		if c := m.Con(); c < k.Cap {
			k.Cap = c
		}
	}
	return k
}

// khoaBang dựng khoá TOML để in trong lời khuyên. Tên có dấu hai chấm (địa chỉ
// hồ sơ) thì phải bọc nháy, nếu không người dùng chép nguyên câu vào file sẽ
// gặp lỗi cú pháp — lời khuyên mà chép vào là hỏng thì tệ hơn không khuyên.
func khoaBang(bang, ten string) string {
	if ten == "" {
		return bang
	}
	if strings.ContainsAny(ten, ":. \t\"") {
		return bang + "." + strconv.Quote(ten)
	}
	return bang + "." + ten
}

// LoiCatBot là câu cảnh báo khi CÒN chỗ nhưng ít hơn số đã xin.
//
// Viết cho người vận hành đọc lúc hai giờ sáng: nói trần NÀO cắt, cắt từ mấy
// xuống mấy, bốn trần lúc này đứng ở đâu, và làm gì tiếp. Câu "đã hạ xuống N"
// trơ trọi thì đọc xong vẫn phải đi tìm xem vì sao.
func (k KetTran) LoiCatBot() string {
	chat := k.Chat()
	if len(chat) == 0 {
		return ""
	}
	m := chat[0]
	var b strings.Builder
	fmt.Fprintf(&b, "TRẦN ĐỒNG THỜI cắt %d phiên xuống %d cho %s. Chật nhất: %s — trần %d, đang chạy %d, còn %d chỗ.",
		k.Xin, k.Cap, k.Addr, m.Nhan(), m.Tran, m.Dang, m.Con())
	if len(chat) > 1 {
		ten := make([]string, 0, len(chat)-1)
		for _, c := range chat[1:] {
			ten = append(ten, fmt.Sprintf("%s (còn %d)", c.Nhan(), c.Con()))
		}
		fmt.Fprintf(&b, " Cũng chật: %s.", strings.Join(ten, ", "))
	}
	fmt.Fprintf(&b, " Bốn trần lúc này — %s.", k.Bang())
	fmt.Fprintf(&b, " Muốn đủ %d: chia sang tài khoản khác (`sagent ds` xem còn tài khoản nào), "+
		"hoặc nới `%s = <số>` trong .sagent/project.toml.", k.Xin, m.Khoa)
	if m.Loai == "hồ sơ" {
		b.WriteString(" Nhưng nới trần hồ sơ nghĩa là các bản đó CÙNG đốt một hạn mức thuê bao — " +
			"chia sang tài khoản khác thì không.")
	}
	return b.String()
}

// LoiHetCho là câu từ chối khi không còn chỗ nào.
//
// Từ chối mà không nói trần nào chặn là đúng thứ dự án này lập ra để chống:
// người vận hành nhìn `max_parallel_sessions = 4` với 1 phiên đang chạy rồi
// không hiểu vì sao bị chặn — vì cái chặn họ là trần hồ sơ, không phải trần
// chung. Nên câu này gọi tên trần, in số, và chỉ đúng khoá cần sửa.
func (k KetTran) LoiHetCho() error {
	var het []Muc
	for _, m := range k.Mucs {
		if !m.Tat() && m.Con() == 0 {
			het = append(het, m)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "hết chỗ cho %s: ", k.Addr)
	if len(het) == 0 {
		// Không xảy ra qua đường XetTran, nhưng đừng bao giờ trả câu rỗng.
		b.WriteString("không cấp được phiên nào")
	} else {
		ten := make([]string, 0, len(het))
		for _, m := range het {
			ten = append(ten, fmt.Sprintf("%s đã đầy (trần %d, đang chạy %d)", m.Nhan(), m.Tran, m.Dang))
		}
		b.WriteString(strings.Join(ten, "; "))
	}
	fmt.Fprintf(&b, ".\n  Bốn trần lúc này — %s.", k.Bang())
	b.WriteString("\n  Ba cách đi tiếp: (1) chạy cùng việc trên tài khoản khác — `sagent ds` xem còn tài khoản nào;" +
		" (2) `sagent status` rồi `sagent stop <id>` dừng bớt;" +
		" (3) nới trần trong .sagent/project.toml — " + strings.Join(khoaCua(het), ", ") + ".")
	return fmt.Errorf("%s", b.String())
}

func khoaCua(ms []Muc) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Khoa+" = <số>")
	}
	if len(out) == 0 {
		out = append(out, "policy.max_parallel_sessions = <số>")
	}
	return out
}
