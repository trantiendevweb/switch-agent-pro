// XEM ARTIFACT CỦA MỘT LƯỢT CHẠY — action "flow.artifacts".
//
// ============================================================================
// VÌ SAO ĐÂY LÀ MỘT ACTION MỚI CHỨ KHÔNG PHẢI MỘT TRƯỜNG THÊM VÀO
// ============================================================================
//
// Ba mảnh của báo cáo #199 (artifact, idempotent, compensate) cố ý KHÔNG là
// action mới: chúng đổi *cái flow làm gì*, không đổi *người ta bảo công cụ làm
// gì*. Cái này thì ngược hẳn — nó là một ĐỘNG TỪ mới. Trước bản này, muốn đọc
// file một bước để lại thì phải mở `~/.ai-accounts/artifacts/run-<id>/<bước>/`
// bằng tay, tức là tính năng chỉ dùng được bởi người đang ngồi trước máy chủ.
//
// ============================================================================
// MỘT ACTION, HAI CÂU HỎI
// ============================================================================
//
//	FlowArtifacts     lượt chạy này để lại NHỮNG GÌ  (tên, bước, kích thước, đường dẫn)
//	FlowArtifactDoc   trong MỘT file đó có gì
//
// Một action chứ không hai, vì chúng cùng một thứ đắt (không đắt gì cả: đọc file
// cục bộ) và cùng một câu người dùng hỏi. Đây khác với `api.nang-luc` /
// `api.nang-luc-do` — cặp đó tách ra vì một cái đọc bảng còn một cái TIÊU TIỀN.
//
// ============================================================================
// TRẦN: 65.536 BYTE MỖI LẦN ĐỌC, VÀ NÓI RÕ KHI BỊ CẮT
// ============================================================================
//
// Artifact đo được ngày 22/08 là 60.094 byte, và không có gì chặn nó lớn hơn:
// một bản `git diff` của lượt chạy dài, một file JSON gộp, một bản log. Ném cả
// file vào trình duyệt là cách chắc chắn để treo đúng cái tab người ta đang cần.
//
// 65.536 chọn có lý do chứ không phải một số tròn: nó ĐỦ CHỨA artifact lớn nhất
// đã đo được (60.094), nên ca thường không bao giờ bị cắt — mà vẫn có trần thật
// cho ca bất thường.
//
// Bị cắt thì KHÔNG im lặng: `BiCat` bật, `Byte` là kích thước THẬT trên đĩa, và
// `Tu`+`DocByte` nói chính xác đang đọc khúc nào. Có `tu` nên trần này là một
// CỬA SỔ TRƯỢT chứ không phải một bức tường — người đọc lấy tiếp khúc sau được,
// và cái giá của một trần thành thật chỉ là thêm một tham số.
//
// ============================================================================
// CHE BÍ MẬT — CỬA RA THỨ BA
// ============================================================================
//
// redaction_test.go canh hai cửa ra của NHẬT KÝ. Artifact là cửa thứ ba, và nó
// cùng loại: chữ do agent sinh ra, đi thẳng ra cổng HTTP. Một agent viết báo cáo
// có dán khoá API vào là chuyện đã xảy ra ở nhật ký; không có lý do gì nó không
// xảy ra ở artifact.
//
// Che NỘI DUNG, KHÔNG che đường dẫn thư mục (`Dir`): đường dẫn là thứ người vận
// hành cần để mở file bằng tay, và `sagent flow list` đã in nguyên đường dẫn như
// vậy từ trước. Che nó đi thì trường đó thành vô dụng ở đúng mặt cần nó nhất.
package api

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
	"github.com/trantiendevweb/switch-agent-pro/internal/redaction"
)

// TranDocArtifact là số byte tối đa MỘT lần đọc trả về. Xem ghi chú đầu file.
const TranDocArtifact = 64 * 1024

// FileArtifact là một file trong thư mục artifact của một lượt chạy.
type FileArtifact struct {
	// Duong là đường dẫn TƯƠNG ĐỐI trong thư mục lượt chạy, luôn dùng `/`:
	// "<bước>/<đường dẫn khai trong flows.toml>".
	//
	// Tương đối chứ không tuyệt đối, và đây là chuyện an toàn chứ không phải
	// thẩm mỹ: đây chính là chuỗi mà người gọi gửi ngược lại cho FlowArtifactDoc.
	// Cho họ quen với đường dẫn tuyệt đối là mời họ thử sửa nó.
	Duong string `json:"duong"`

	// Buoc là bước sản xuất ra file — chính là thư mục cấp một của Duong.
	Buoc string `json:"buoc"`

	// Ten là TÊN khai trong flows.toml (`artifact = { bao-cao = "x.txt" }`),
	// tức cái tên bước sau dùng ở `{{artifacts.bao-cao}}`.
	//
	// RỖNG là hợp lệ và có nghĩa thật: file có trên đĩa mà không bước nào khai
	// nó. Bước ghi ba file rồi chỉ hứa một cái là chuyện bình thường. Giấu
	// những file này đi thì bảng nói thiếu đúng chỗ không ai ngờ tới.
	Ten string `json:"ten,omitempty"`

	Byte int64  `json:"byte"`
	Sua  string `json:"sua"` // RFC3339

	// NhiPhan = đọc ra chữ không được. Nói trước ở BẢNG LIỆT KÊ chứ không đợi
	// tới lúc bấm mở: người ta bấm vào một file 40 MB rồi mới biết nó là ảnh
	// thì đã tốn một vòng.
	NhiPhan bool `json:"nhiPhan,omitempty"`
}

// KhoArtifact là toàn bộ artifact một lượt chạy để lại.
type KhoArtifact struct {
	RunID int64  `json:"runId"`
	Flow  string `json:"flow"`

	// Dir là thư mục TUYỆT ĐỐI chứa artifact của lượt chạy này.
	//
	// Có mặt vì câu hỏi ngay sau "có những file gì" luôn là "mở bằng tay ở
	// đâu". Cùng loại với `KeHoachKho.Dir` và với dòng "Đọc từ: …" của
	// `sagent flow list`.
	Dir string `json:"dir"`

	File     []FileArtifact `json:"file"`
	TongByte int64          `json:"tongByte"`

	// ThieuDinhNghia = không đọc được `flows.toml` của lượt chạy này nữa (flow
	// bị xoá, thư mục dự án đã đổi). Bảng vẫn liệt kê đủ file, chỉ là cột `Ten`
	// trống hết.
	//
	// Nói ra chứ không im: một cột trống vì "không bước nào khai" khác hẳn một
	// cột trống vì "không tra được", và trộn hai chuyện đó lại là để người đọc
	// tự kết luận sai.
	ThieuDinhNghia bool `json:"thieuDinhNghia,omitempty"`
}

// NoiDungArtifact là một khúc nội dung của MỘT file artifact.
type NoiDungArtifact struct {
	Duong string `json:"duong"`

	// Byte là kích thước THẬT trên đĩa — không phải số byte trả về.
	Byte int64 `json:"byte"`

	// Tu là vị trí bắt đầu của khúc này, DocByte là độ dài của nó.
	Tu      int64 `json:"tu"`
	DocByte int64 `json:"docByte"`

	// BiCat = còn byte phía sau chưa đọc. Đi kèm ConLai để câu "bị cắt" có số.
	BiCat  bool  `json:"biCat"`
	ConLai int64 `json:"conLai"`

	// NhiPhan = không phải chữ. Khi đó Chu RỖNG: ném byte thô vào trình duyệt
	// không giúp ai đọc được gì, chỉ làm hỏng trang.
	NhiPhan bool `json:"nhiPhan,omitempty"`

	// Chu là nội dung ĐÃ CHE BÍ MẬT. Xem ghi chú đầu file.
	Chu string `json:"chu"`
}

// FlowArtifacts — action "flow.artifacts". Liệt kê artifact của một lượt chạy.
//
// KHÔNG đọc nội dung file nào: duyệt thư mục và hỏi kích thước. Nhờ vậy bảng
// liệt kê một lượt chạy để lại 400 MB cũng chạy trong một nháy.
func (a *API) FlowArtifacts(runID int64) (KhoArtifact, error) {
	kho := KhoArtifact{RunID: runID, Dir: flow.ArtifactRunDir(runID), File: []FileArtifact{}}

	// Tra sổ TRƯỚC: lượt chạy không có trong sổ thì nói thẳng, đừng để người
	// dùng nhìn một bảng rỗng rồi tự đoán là "chưa có artifact".
	run, err := a.db.GetRun(runID)
	if err != nil {
		return kho, fmt.Errorf("không có lượt chạy #%d (xem: sagent flow runs)", runID)
	}
	kho.Flow = run.Flow

	// Bản đồ TÊN: cần định nghĩa flow, mà định nghĩa có thể đã đổi hoặc mất kể
	// từ lượt chạy đó. Mất thì bảng vẫn ra, chỉ là không có tên — xem
	// ThieuDinhNghia.
	ten := func(string) string { return "" }
	if flows, _, err := flow.Load(run.Dir); err == nil {
		if f, co := flows[run.Flow]; co {
			ten = flow.BanDoTenArtifact(f)
		} else {
			kho.ThieuDinhNghia = true
		}
	} else {
		kho.ThieuDinhNghia = true
	}

	goc := kho.Dir
	err = filepath.WalkDir(goc, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// Một thư mục con không đọc được không được làm hỏng cả bảng: phần
			// còn lại vẫn là câu trả lời đúng cho phần còn lại.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(goc, p)
		if err != nil {
			return nil
		}
		slash := filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return nil
		}
		f := FileArtifact{
			Duong: slash,
			Buoc:  buocCuaDuong(slash),
			Ten:   ten(slash),
			Byte:  info.Size(),
			Sua:   info.ModTime().UTC().Format(time.RFC3339),
		}
		f.NhiPhan = laNhiPhan(p)
		kho.File = append(kho.File, f)
		kho.TongByte += f.Byte
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return kho, err
	}

	// Sắp xếp CỐ ĐỊNH: WalkDir đã đi theo thứ tự tên, nhưng nói ra vẫn hơn —
	// hai lần gọi phải ra hai bảng giống hệt nhau, nếu không thì mặt web nhấp
	// nháy và không ai so được hai lượt chạy với nhau.
	sort.Slice(kho.File, func(i, j int) bool { return kho.File[i].Duong < kho.File[j].Duong })
	return kho, nil
}

// FlowArtifactDoc — action "flow.artifacts". Đọc MỘT KHÚC nội dung một artifact.
//
// `duong` là chuỗi lấy nguyên từ FlowArtifacts. Mọi thứ khác đều bị từ chối —
// xem flow.DuongDanArtifactAnToan cho ba lớp chặn và lý do lớp thứ ba tồn tại.
//
// `tu` là vị trí byte bắt đầu; 0 là từ đầu file. Đọc tối đa TranDocArtifact byte.
func (a *API) FlowArtifactDoc(runID int64, duong string, tu int64) (NoiDungArtifact, error) {
	var out NoiDungArtifact
	if tu < 0 {
		return out, fmt.Errorf("vị trí đọc không được âm, được %d", tu)
	}
	that, err := flow.DuongDanArtifactAnToan(runID, duong)
	if err != nil {
		return out, err
	}
	st, err := os.Stat(that)
	if err != nil {
		return out, fmt.Errorf("không đọc được artifact %q", duong)
	}
	if st.IsDir() {
		return out, fmt.Errorf("%q là một THƯ MỤC, không phải file", duong)
	}

	out = NoiDungArtifact{Duong: filepath.ToSlash(filepath.Clean(duong)), Byte: st.Size(), Tu: tu}
	if tu >= st.Size() {
		// Đọc quá đuôi không phải lỗi: nó là câu trả lời "hết rồi". Trả về khúc
		// rỗng chứ không báo lỗi, để vòng lặp lấy tiếp của người gọi dừng được
		// một cách tự nhiên.
		return out, nil
	}

	f, err := os.Open(that)
	if err != nil {
		return out, fmt.Errorf("không mở được artifact %q", duong)
	}
	defer f.Close()
	if _, err := f.Seek(tu, io.SeekStart); err != nil {
		return out, fmt.Errorf("không đọc được artifact %q từ vị trí %d", duong, tu)
	}
	buf := make([]byte, TranDocArtifact)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return out, fmt.Errorf("không đọc được artifact %q", duong)
	}
	buf = buf[:n]

	out.DocByte = int64(n)
	out.ConLai = st.Size() - tu - int64(n)
	out.BiCat = out.ConLai > 0

	if !doDuocRaChu(buf) {
		out.NhiPhan = true
		return out, nil
	}
	// Cắt theo BYTE có thể chặt đôi một ký tự UTF-8 ở mép — chuyện xảy ra thật
	// với tiếng Việt, nơi gần như mọi chữ có dấu đều là 2-3 byte. Bỏ mẩu cụt ở
	// đuôi đi (chỉ khi CÒN phần sau; cắt ở cuối file thì mẩu đó là dữ liệu thật
	// và giấu nó là nói dối về file).
	if out.BiCat {
		buf = boMauCutCuoi(buf)
		out.DocByte = int64(len(buf))
		out.ConLai = st.Size() - tu - out.DocByte
	}
	out.Chu = redaction.Che(string(buf))
	return out, nil
}

// buocCuaDuong lấy thư mục cấp một của một đường dẫn tương đối dùng `/`.
//
// Thư mục cấp một CHÍNH LÀ id bước — ArtifactStepDir dựng nó như vậy. File nằm
// ngay gốc lượt chạy (không có bước nào) trả về rỗng chứ không đoán bừa.
func buocCuaDuong(slash string) string {
	for i := 0; i < len(slash); i++ {
		if slash[i] == '/' {
			return slash[:i]
		}
	}
	return ""
}

// laNhiPhan đoán một file có đọc ra chữ được không, bằng cách nhìn 8 KiB ĐẦU.
//
// Chỉ nhìn phần đầu chứ không đọc cả file: bảng liệt kê phải chạy nhanh kể cả
// khi lượt chạy để lại vài trăm MB, và một file có 8 KiB đầu là văn bản sạch mà
// byte thứ một triệu là NUL thì cũng chỉ làm khúc đó hiện ra xấu, chứ không
// treo trang.
//
// Không đọc được thì trả về false: bảng liệt kê không phải chỗ báo lỗi quyền
// truy cập, và lúc BẤM MỞ thì FlowArtifactDoc sẽ nói ra tử tế.
func laNhiPhan(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 8*1024)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return false
	}
	return !doDuocRaChu(buf[:n])
}

// doDuocRaChu: một khúc byte có phải VĂN BẢN không.
//
// Hai phép, và cần cả hai. Byte NUL là dấu hiệu chắc chắn nhất của file nhị
// phân, còn UTF-8 hỏng bắt được phần còn lại — trừ một chỗ: mẩu cụt ở ĐUÔI do
// cắt theo byte không phải file hỏng, nên bỏ tối đa 3 byte cuối trước khi hỏi.
func doDuocRaChu(b []byte) bool {
	if bytes.IndexByte(b, 0) >= 0 {
		return false
	}
	return utf8.Valid(boMauCutCuoi(b))
}

// boMauCutCuoi bỏ mẩu ký tự UTF-8 bị chặt đôi ở cuối khúc.
//
// Một ký tự UTF-8 dài tối đa 4 byte, nên lùi tối đa 3 byte là đủ. Lùi mà vẫn
// không hợp lệ thì trả về nguyên khúc: lúc đó nó hỏng thật, không phải bị cắt.
func boMauCutCuoi(b []byte) []byte {
	if utf8.Valid(b) {
		return b
	}
	for i := 1; i <= 3 && i <= len(b); i++ {
		if utf8.Valid(b[:len(b)-i]) {
			return b[:len(b)-i]
		}
	}
	return b
}
