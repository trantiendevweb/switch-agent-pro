// Package nhatky giữ lại NHẬT KÝ của từng phiên fleet, đọc lại được SAU KHI
// phiên đã kết thúc.
//
// VÌ SAO CÓ GÓI NÀY (đo 21/08/2026)
//
// Fleet vẫn luôn đổ stdout/stderr của agent vào một file — nhưng file đó là
//
//	<thư mục clone>/fleet.log
//
// tức đường dẫn CHỈ phụ thuộc số bản clone, không phụ thuộc phiên. Hai hệ quả
// đã trả giá thật trong cùng một ngày:
//
//   - `os.Create` CẮT TRẮNG file. Phiên #169 chạy trên clone 1 xoá sạch nhật ký
//     của phiên #167 cũng chạy trên clone 1. Cả hai báo "xong" mà 0 commit,
//     worktree sạch trơn, và không còn gì để đọc. Mất ~15 phút truy nguyên mới
//     ra nguyên nhân là thiếu cờ --tu-duyet-quyen.
//   - `sagent clean` xoá nguyên thư mục clone, tức xoá luôn nhật ký. Đúng lệnh
//     người ta chạy sau một lượt hỏng để dọn dẹp lại là lệnh phá tang chứng.
//
// Phiên #172 sửa 4 file rồi chết vì hết hạn mức cũng không để lại dấu vết nào.
//
// CHỖ ĐẶT: ~/.ai-accounts/.nhat-ky/, NGANG HÀNG với state.db chứ không nằm
// trong worktree hay trong thư mục clone. Ba lý do, theo thứ tự quan trọng:
//
//  1. Nhật ký phải SỐNG LÂU HƠN thứ nó nói về. Worktree bị `git worktree
//     remove`, clone bị `sagent clean` — cả hai đều là thao tác dọn dẹp bình
//     thường sau một lượt hỏng. Sổ phiên (state.db) sống sót qua cả hai, nên
//     nhật ký nằm cạnh sổ thì cặp "sổ ghi + nhật ký" không bao giờ lệch nhau.
//  2. Một chỗ duy nhất để nhìn. `sagent nhat-ky` liệt kê được cả hạm đội mà
//     không phải đi quét N thư mục clone của M tài khoản.
//  3. Ngoài repo của người dùng. Log của agent lẫn vào `git status` là rác, và
//     tệ hơn là có thể bị commit nhầm — nhật ký stream-json chứa nguyên văn
//     prompt lẫn nội dung file.
//
// TÊN FILE KHÔNG mang số phiên, và đó là chủ ý chứ không phải thiếu sót: số
// phiên do sổ cấp trong `db.AddSession`, mà lệnh đó chỉ chạy được SAU khi tiến
// trình đã bật (cần PID). Lúc đặt tên file thì chưa có số. Đổi tên file sau đó
// không được: tiến trình con đang mở nó, và trên Windows `os.Rename` một file
// đang mở sẽ hỏng. Nên tên file mang ĐỊA CHỈ + MỐC THỜI GIAN (tự nó đã đủ để
// người đọc nhận ra), còn ánh xạ "số phiên → đường dẫn" nằm ở cột `log` của sổ
// — đúng chỗ nó vốn phải nằm, và `sagent nhat-ky <id>` đọc từ đó.
package nhatky

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// Khối tiêu đề do sagent ghi vào ĐẦU nhật ký, trước khi agent in chữ nào.
//
// Đây là phần trả lời được ca #167/#169: nhìn dòng `lệnh:` là thấy ngay lượt
// chạy đó thiếu cờ nào. Bản ghi của agent KHÔNG chứa thông tin này — nó chỉ
// biết những gì nó nhận được, không biết những gì lẽ ra nó phải nhận.
const (
	// MocDau là dòng mở khối. Có nó ở đầu file thì BoDau biết là có khối.
	MocDau = "# sagent nhật ký phiên"
	// MocHet đóng khối. Mọi thứ sau dòng này là chữ của AGENT, không phải của
	// sagent — ranh giới phải rạch ròi vì `readLogs` lấy phần sau làm output
	// của bước flow.
	MocHet = "# ---"
)

// Ngân sách đĩa. Xem Don để biết vì sao chặn ở đây mà không chặn từng file.
const (
	// SoFileToiDa: giữ bao nhiêu nhật ký. 200 là khoảng 50 lượt fleet 4 bản —
	// đủ để lần lại vài ngày làm việc, mà vẫn liệt kê được trong một màn hình.
	SoFileToiDa = 200
	// TongByteToiDa: trần tổng dung lượng. Ràng buộc nào chạm trước thì ràng
	// buộc đó có hiệu lực.
	TongByteToiDa int64 = 256 << 20 // 256 MB
)

// Root là thư mục nhật ký: ~/.ai-accounts/.nhat-ky
//
// Dấu chấm ở đầu tên để `profile.List()` không nhầm nó là một provider — cùng
// quy ước với .clones (xem profile.ClonesRoot).
func Root() string { return filepath.Join(paths.AccountsRoot(), ".nhat-ky") }

// Dau là khối tiêu đề của một phiên: những gì sagent BIẾT lúc bật, mà agent
// thì không.
type Dau struct {
	ThoiDiem time.Time
	Addr     string   // "claude:tns#1"
	HoSo     string   // thư mục config riêng của bản clone
	ThuMuc   string   // chỗ agent làm việc (worktree, hoặc thư mục hiện tại)
	Lenh     []string // args ĐÃ DỰNG XONG, đúng thứ tiến trình con nhận
}

// String dựng khối tiêu đề. Mọi dòng đều bắt đầu bằng "# " nên bộ đọc kết quả
// có cấu trúc (provider.DocKetQua) bỏ qua tự nhiên — chúng chỉ nhận dòng mở
// đầu bằng "{".
func (d Dau) String() string {
	var b strings.Builder
	b.WriteString(MocDau + "\n")
	b.WriteString("# thời điểm: " + d.ThoiDiem.Format(time.RFC3339) + "\n")
	b.WriteString("# địa chỉ:   " + d.Addr + "\n")
	if d.HoSo != "" {
		b.WriteString("# hồ sơ:     " + d.HoSo + "\n")
	}
	if d.ThuMuc != "" {
		b.WriteString("# thư mục:   " + d.ThuMuc + "\n")
	}
	// Dòng quan trọng nhất của cả khối: đây là chỗ ca #167 tự lộ ra.
	b.WriteString("# lệnh:      " + strings.Join(d.Lenh, " ") + "\n")
	b.WriteString(MocHet + "\n")
	return b.String()
}

// Duong dựng đường dẫn nhật ký cho một phiên sắp bật.
//
// Mốc thời gian tới MILI GIÂY: fleet bật N bản trong cùng một giây là chuyện
// bình thường. Số bản clone cũng nằm trong tên nên hai bản của cùng một lượt
// không thể trùng; mili giây lo nốt hai LƯỢT sát nhau. Vẫn kiểm tồn tại và
// thêm hậu tố, vì trùng tên ở đây nghĩa là mất nhật ký — đúng thứ gói này lập
// ra để chống.
func Duong(prov, account string, clone int, t time.Time) string {
	goc := fmt.Sprintf("%s-%s-c%d-%s", vesinh(prov), vesinh(account), clone,
		t.Format("20060102-150405.000"))
	p := filepath.Join(Root(), goc+".log")
	for i := 2; i < 100; i++ {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
		p = filepath.Join(Root(), fmt.Sprintf("%s-%d.log", goc, i))
	}
	return p
}

// vesinh bỏ mọi ký tự không an toàn cho tên file. Tên tài khoản vốn đã là tên
// thư mục nên gần như luôn sạch; đây là lớp chắn cuối, không phải lớp chính.
func vesinh(s string) string {
	if s == "" {
		return "_"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// Tao tạo file nhật ký và ghi khối tiêu đề vào.
//
// 0o600 chứ không phải mặc định của os.Create: nhật ký stream-json chứa nguyên
// văn prompt và nội dung file được đọc. (Trên Windows bit này không bảo vệ gì —
// quyền thật kế thừa từ thư mục cha, xem internal/acl — nhưng nơi khác thì có.)
func Tao(duong string, d Dau) error {
	if err := os.MkdirAll(filepath.Dir(duong), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(duong, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(d.String())
	return err
}

// GhiLoi nối một dòng lý do CHẾT vào cuối nhật ký.
//
// Dùng cho những kiểu chết mà agent không kịp nói: không bật được tiến trình,
// không ghi được vào sổ. Không có dòng này thì đúng các ca hỏng SỚM NHẤT lại là
// các ca không để lại gì — mà chúng mới là ca khó đoán nhất.
//
// Vẫn bắt đầu bằng "# " để nằm cùng phía với khối tiêu đề: chữ của sagent, không
// phải chữ của agent.
func GhiLoi(duong, msg string) error {
	f, err := os.OpenFile(duong, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "# ✗ sagent: %s\n", msg)
	return err
}

// BoDau cắt khối tiêu đề ra khỏi nội dung nhật ký.
//
// Cần vì flow lấy nguyên văn nhật ký làm OUTPUT của bước agent (api.readLogs)
// và truyền sang bước sau. Không cắt thì lời chỉ dẫn nội bộ của sagent trở
// thành một phần câu trả lời của agent — nhìn thì nhỏ, nhưng bước sau là một
// agent khác đang đọc nó như dữ liệu thật.
func BoDau(raw string) string {
	if !strings.HasPrefix(raw, MocDau) {
		return raw
	}
	dong := strings.SplitAfter(raw, "\n")
	for i, d := range dong {
		if strings.TrimRight(d, "\r\n") == MocHet {
			return strings.Join(dong[i+1:], "")
		}
	}
	// Không tìm thấy dòng đóng: phiên chết trước khi khối được ghi xong. Bỏ mọi
	// dòng "#" liền đầu — thà cắt hụt còn hơn trả về nguyên khối.
	for i, d := range dong {
		if !strings.HasPrefix(d, "#") {
			return strings.Join(dong[i:], "")
		}
	}
	return ""
}

// Duoi lấy n dòng CUỐI của nội dung nhật ký.
//
// Mặc định của mọi mặt đọc nhật ký, và đây là chỗ chặn thứ hai của ngân sách
// đĩa: một file 200 MB đổ thẳng ra terminal thì không đọc được mà còn treo máy.
// Dòng cuối cũng đúng chỗ đáng đọc nhất — bản ghi `{"type":"result"}` của
// Claude nằm ở cuối, và lý do chết luôn nằm ở cuối.
func Duoi(raw string, n int) string {
	if n <= 0 {
		return raw
	}
	dong := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if len(dong) <= n {
		return strings.Join(dong, "\n")
	}
	return strings.Join(dong[len(dong)-n:], "\n")
}

// KetQuaDon là những gì Don đã làm, để mặt gọi nói ra thay vì dọn trong im lặng.
type KetQuaDon struct {
	DaXoa    int
	ByteXoa  int64
	ConLai   int
	ByteCon  int64
	KhongXoa int // file quá ngân sách nhưng xoá không được (đang bị khoá)
}

// Don giữ thư mục nhật ký trong NGÂN SÁCH, xoá cũ trước.
//
// VÌ SAO CHẶN Ở ĐÂY MÀ KHÔNG CHẶN TỪNG FILE. Một phiên fleet là tiến trình
// TÁCH RỜI: `profile.StartDetached` trao thẳng file handle cho tiến trình con
// rồi tiến trình cha THOÁT. Không còn ai đứng giữa dòng ghi, nên không có chỗ
// nào để xoay file hay cắt ngang khi nó vượt ngưỡng — muốn làm được thì phải
// nuôi một tiến trình trung gian sống suốt lượt chạy, tức đổi hẳn kiến trúc
// "bật rồi buông" mà công cụ đang dựa vào.
//
// Nên trần đĩa được giữ ở hai chỗ khác, cả hai đều đo được:
//
//   - LÚC ĐỌC: mọi mặt đọc theo đuôi (Duoi), file to không tràn ra màn hình.
//   - LÚC BẬT LƯỢT SAU: hàm này, chạy ở đầu mỗi lần fleet. Thư mục nhật ký lớn
//     dần trong một lượt, rồi co lại về ngân sách ở lượt kế.
//
// giu là danh sách đường dẫn KHÔNG ĐƯỢC XOÁ — nhật ký của các phiên còn đang
// chạy. Chúng vẫn TÍNH vào ngân sách (chúng chiếm đĩa thật), chỉ là không bị
// chọn để xoá. Thiếu vế này thì một lượt fleet dài sẽ bị lượt sau xoá nhật ký
// ngay giữa lúc đang ghi.
//
// Xoá hụt (Windows khoá file đang mở) KHÔNG phải lỗi: đếm vào KhongXoa rồi đi
// tiếp. Dừng cả phép dọn vì một file bị khoá thì các file cũ hơn không bao giờ
// được dọn.
func Don(giu []string, soFileToiDa int, tongByteToiDa int64) (KetQuaDon, error) {
	var kq KetQuaDon
	entries, err := os.ReadDir(Root())
	if err != nil {
		if os.IsNotExist(err) {
			return kq, nil // chưa chạy fleet lần nào
		}
		return kq, err
	}

	giuLai := map[string]bool{}
	for _, p := range giu {
		if p != "" {
			giuLai[strings.ToLower(filepath.Clean(p))] = true
		}
	}

	type muc struct {
		duong string
		khi   time.Time
		co    int64
	}
	var ds []muc
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		st, err := e.Info()
		if err != nil {
			continue
		}
		ds = append(ds, muc{filepath.Join(Root(), e.Name()), st.ModTime(), st.Size()})
	}
	// Mới nhất trước: cái đáng giữ nhất đứng đầu, cái bị đẩy ra khỏi ngân sách
	// là cái cũ nhất.
	sort.Slice(ds, func(i, j int) bool { return ds[i].khi.After(ds[j].khi) })

	n, tong := 0, int64(0)
	for _, m := range ds {
		n++
		tong += m.co
		quaTran := (soFileToiDa > 0 && n > soFileToiDa) ||
			(tongByteToiDa > 0 && tong > tongByteToiDa)
		if !quaTran || giuLai[strings.ToLower(filepath.Clean(m.duong))] {
			kq.ConLai++
			kq.ByteCon += m.co
			continue
		}
		if err := os.Remove(m.duong); err != nil {
			kq.KhongXoa++
			kq.ConLai++
			kq.ByteCon += m.co
			continue
		}
		kq.DaXoa++
		kq.ByteXoa += m.co
		// Đã xoá thì trả lại chỗ trong ngân sách cho file cũ hơn phía sau —
		// không trừ thì một file khổng lồ ở giữa sẽ kéo cả đuôi đi theo.
		n--
		tong -= m.co
	}
	return kq, nil
}
