package plugin

// Phía HOST: bật tiến trình con, bắt tay, gọi việc, dọn.
//
// Đây là chỗ ràng buộc 4 (capability tối thiểu) thành hàng rào THẬT chứ không
// phải một dòng chữ trong manifest — xem dungMoiTruong và chonThuMuc. Quyền nào
// host chặn được thật, quyền nào không, ghi trong quyen.go và KHÔNG được nói
// khác ở đây.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
)

// TimeoutMacDinh là trần cho MỘT lượt gọi plugin.
//
// Có trần vì tiến trình con là thứ host không điều khiển được: plugin treo thì
// bước flow treo theo, và một lượt flow treo giữa đêm không ai thấy. Người gọi
// đặt Timeout khác được.
const TimeoutMacDinh = 60 * time.Second

// TuyChon là những thứ HOST quyết định, không phải plugin.
type TuyChon struct {
	// ThuMuc là thư mục dự án. CHỈ được dùng khi plugin khai quyền thu-muc-lam-viec.
	ThuMuc string

	// DocSecret đọc giá trị bí mật theo key_id. nil = dùng kho key mặc định
	// (~/.ai-accounts/api-keys). Có mặt để test không phải đụng vào kho thật.
	DocSecret func(id string) (string, error)

	Timeout time.Duration
}

// Client là một tiến trình plugin ĐANG SỐNG.
//
// Vòng đời ngắn có chủ ý: mở → gọi → đóng. Không giữ tiến trình lâu dài giữa các
// bước flow, vì như vậy thì trạng thái của bước trước rò sang bước sau qua một
// đường không ai nhìn thấy, và một plugin rò bộ nhớ sẽ sống bằng tuổi thọ của cả
// lượt chạy.
type Client struct {
	m        Manifest
	cmd      *exec.Cmd
	vao      io.WriteCloser
	ra       *bufio.Reader
	id       int
	Ten      string
	PhienBan string

	tmp string // thư mục tạm host tạo khi KHÔNG cấp quyền thư mục

	mu  sync.Mutex
	loi *dauCuoi // stderr của plugin, giữ phần cuối
}

// dauCuoi giữ N byte CUỐI của một luồng.
//
// Giữ phần cuối chứ không phải phần đầu: khi plugin chết, câu nói ra nguyên nhân
// gần như luôn là câu cuối cùng (panic, stack trace). Có trần vì stderr là thứ
// đầu kia quyết định độ dài.
type dauCuoi struct {
	mu  sync.Mutex
	b   []byte
	max int
}

func (d *dauCuoi) Write(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.b = append(d.b, p...)
	if len(d.b) > d.max {
		d.b = d.b[len(d.b)-d.max:]
	}
	return len(p), nil
}

func (d *dauCuoi) String() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return strings.TrimSpace(string(d.b))
}

// bienToiThieu là danh sách trắng biến môi trường khi plugin KHÔNG khai quyền
// bien-moi-truong.
//
// Không phải "để cho tiện": đây là những biến mà thiếu chúng thì chính runtime
// không khởi động nổi trên Windows (SystemRoot cho winsock, ComSpec, PATHEXT).
// Cắt sạch nghe an toàn hơn nhưng cho ra một plugin không chạy được lần nào, và
// một hàng rào không ai bật được thì không phải hàng rào.
func bienToiThieu() []string {
	var giu []string
	if runtime.GOOS == "windows" {
		giu = []string{"SystemRoot", "SystemDrive", "windir", "ComSpec", "PATHEXT", "PATH", "TEMP", "TMP", "NUMBER_OF_PROCESSORS"}
	} else {
		giu = []string{"PATH", "HOME", "TMPDIR", "LANG", "LC_ALL"}
	}
	var out []string
	for _, k := range giu {
		if v, co := os.LookupEnv(k); co {
			out = append(out, k+"="+v)
		}
	}
	return out
}

// dungMoiTruong dựng môi trường cho tiến trình con — HÀNG RÀO THẬT của quyền
// bien-moi-truong.
func dungMoiTruong(m Manifest) []string {
	var env []string
	if m.Co(QuyenMoiTruong) {
		env = append(env, os.Environ()...)
	} else {
		env = append(env, bienToiThieu()...)
	}
	// Hai biến này host LUÔN đặt: plugin cần biết nó đang được chạy như plugin
	// (chứ không phải người gõ tay) và host nói giao thức số mấy — để nó từ chối
	// sớm thay vì trả về thứ host không đọc nổi.
	env = append(env, "SAGENT_PLUGIN=1", fmt.Sprintf("SAGENT_GIAO_THUC=%d", GiaoThuc))
	return env
}

// chonThuMuc là HÀNG RÀO THẬT của quyền thu-muc-lam-viec.
//
// Không khai quyền thì tiến trình con chạy trong một thư mục tạm RỖNG do host
// tạo, chứ không phải trong thư mục dự án. Cố ý không để cmd.Dir rỗng: Go sẽ lấy
// thư mục hiện hành của HOST — tức là đúng thư mục dự án — nên "không cấp gì"
// hoá ra lại là cấp tất.
func chonThuMuc(m Manifest, opt TuyChon) (dir, tmp string, err error) {
	if m.Co(QuyenThuMuc) && opt.ThuMuc != "" {
		return opt.ThuMuc, "", nil
	}
	t, err := os.MkdirTemp("", "sagent-plugin-")
	if err != nil {
		return "", "", fmt.Errorf("không tạo được thư mục tạm cho plugin: %w", err)
	}
	return t, t, nil
}

// docSecret đọc ĐÚNG những secret đã khai, và chỉ khi có quyền.
func docSecret(m Manifest, opt TuyChon) (map[string]string, error) {
	if !m.Co(QuyenSecret) {
		return nil, nil
	}
	doc := opt.DocSecret
	if doc == nil {
		doc = aiapi.DocKey
	}
	ra := map[string]string{}
	for _, s := range m.Secret {
		v, err := doc(s.KeyID)
		if err != nil {
			return nil, fmt.Errorf("plugin %q cần secret %q (key_id %q): %w",
				m.Plugin.Ten, s.Ten, s.KeyID, err)
		}
		ra[s.Ten] = v
	}
	return ra, nil
}

// Mo bật plugin và BẮT TAY. Trả về Client đã sẵn sàng nhận việc.
//
// Bắt tay ngay trong Mo chứ không để lười: nếu giao thức lệch hoặc executable
// không phải plugin, phải biết TRƯỚC khi có ai kịp gửi dữ liệu (kể cả secret)
// cho nó.
func Mo(ctx context.Context, m Manifest, opt TuyChon) (*Client, error) {
	if err := m.Kiem(); err != nil {
		return nil, err
	}
	duong, err := TimExec(m)
	if err != nil {
		return nil, err
	}
	// Secret đọc TRƯỚC khi bật tiến trình: thiếu key thì hỏng ngay, không để lại
	// một tiến trình con phải đi dọn.
	secret, err := docSecret(m, opt)
	if err != nil {
		return nil, err
	}
	dir, tmp, err := chonThuMuc(m, opt)
	if err != nil {
		return nil, err
	}

	don := func() {
		if tmp != "" {
			_ = os.RemoveAll(tmp)
		}
	}

	cmd := exec.Command(duong, m.Plugin.Args...)
	cmd.Dir = dir
	cmd.Env = dungMoiTruong(m)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		don()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		don()
		return nil, err
	}
	batLoi := &dauCuoi{max: 8 << 10}
	cmd.Stderr = batLoi
	if err := cmd.Start(); err != nil {
		don()
		return nil, fmt.Errorf("không chạy được plugin %q (%s): %w", m.Plugin.Ten, duong, err)
	}

	c := &Client{m: m, cmd: cmd, vao: stdin, ra: bufio.NewReaderSize(stdout, 64<<10), tmp: tmp, loi: batLoi}

	quyen := make([]string, 0, len(m.Quyen))
	for _, q := range m.Quyen {
		quyen = append(quyen, q.Khoa)
	}
	thuMucGui := ""
	if m.Co(QuyenThuMuc) {
		thuMucGui = dir
	}
	var kq KetQuaBatTay
	if err := c.goi(ctx, opt.Timeout, MethodBatTay, ThamSoBatTay{
		GiaoThuc: GiaoThuc, Host: "sagent", Quyen: quyen, ThuMuc: thuMucGui, Secret: secret,
	}, &kq); err != nil {
		c.Dong()
		return nil, fmt.Errorf("bắt tay với plugin %q hỏng: %w", m.Plugin.Ten, err)
	}
	if err := c.doiChieu(kq); err != nil {
		c.Dong()
		return nil, err
	}
	c.Ten, c.PhienBan = kq.Ten, kq.PhienBan
	return c, nil
}

// doiChieu so LỜI TỰ KHAI của executable với MANIFEST.
//
// Vì sao phải so: manifest là thứ người vận hành đọc và duyệt; executable là thứ
// thật sự chạy. Nếu binary xin được nhiều hơn manifest ghi mà host vẫn chạy, thì
// bản duyệt kia chỉ là giấy tờ. Lệch một li là DỪNG, không phải cảnh báo — vì
// người đọc cảnh báo lúc đó thường là log, không phải người.
func (c *Client) doiChieu(kq KetQuaBatTay) error {
	if kq.GiaoThuc != GiaoThuc {
		return fmt.Errorf("plugin %q nói giao thức %d, host nói %d", c.m.Plugin.Ten, kq.GiaoThuc, GiaoThuc)
	}
	if kq.Ten != c.m.Plugin.Ten {
		return fmt.Errorf("manifest ghi tên %q nhưng executable tự xưng là %q",
			c.m.Plugin.Ten, kq.Ten)
	}
	for _, q := range kq.Quyen {
		if !c.m.Co(q) {
			return fmt.Errorf("plugin %q XIN quyền %q mà manifest không khai — "+
				"hoặc thêm [[quyen]] vào %s (và nói ly_do), hoặc plugin phải thôi xin",
				c.m.Plugin.Ten, q, c.m.Duong)
		}
	}
	return nil
}

// Chay gọi một lượt việc.
func (c *Client) Chay(ctx context.Context, vao string, thamSo map[string]string) (KetQuaChay, error) {
	var kq KetQuaChay
	err := c.goi(ctx, 0, MethodChay, ThamSoChay{Vao: vao, ThamSo: thamSo}, &kq)
	return kq, err
}

// goi gửi một yêu cầu và đợi đúng phản hồi của nó.
//
// Tuần tự có chủ ý: một Client là một tiến trình, một lượt gọi. Không ghép nhiều
// yêu cầu song song trên cùng một ống — làm vậy thì phải có bảng tra id, mà đổi
// lại chẳng nhanh hơn chút nào (plugin nào cũng chạy một việc một lúc).
func (c *Client) goi(ctx context.Context, han time.Duration, method string, params, ra any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if han <= 0 {
		han = TimeoutMacDinh
	}
	ctx, huy := context.WithTimeout(ctx, han)
	defer huy()

	c.id++
	b, err := json.Marshal(params)
	if err != nil {
		return err
	}
	goi, err := json.Marshal(goiTin{JSONRPC: "2.0", ID: c.id, Method: method, Params: b})
	if err != nil {
		return err
	}

	type ketQuaDoc struct {
		dong []byte
		err  error
	}
	xong := make(chan ketQuaDoc, 1)
	go func() {
		if _, err := c.vao.Write(append(goi, '\n')); err != nil {
			xong <- ketQuaDoc{err: fmt.Errorf("không gửi được yêu cầu: %w%s", err, c.themLoi())}
			return
		}
		dong, err := c.docDong()
		xong <- ketQuaDoc{dong: dong, err: err}
	}()

	select {
	case <-ctx.Done():
		// Hết giờ thì GIẾT, không đợi: tiến trình con còn sống là còn giữ ống, và
		// goroutine ở trên sẽ nằm đó mãi.
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		return fmt.Errorf("plugin %q quá %v không trả lời (%s)%s",
			c.m.Plugin.Ten, han, method, c.themLoi())
	case r := <-xong:
		if r.err != nil {
			return r.err
		}
		var t traLoi
		if err := json.Unmarshal(r.dong, &t); err != nil {
			return fmt.Errorf("plugin %q trả về thứ không phải JSON-RPC: %s%s",
				c.m.Plugin.Ten, catNgan(string(r.dong)), c.themLoi())
		}
		if t.Error != nil {
			return t.Error
		}
		if t.ID != c.id {
			return fmt.Errorf("plugin %q trả lời id %d trong khi host hỏi id %d",
				c.m.Plugin.Ten, t.ID, c.id)
		}
		if ra == nil {
			return nil
		}
		return json.Unmarshal(t.Result, ra)
	}
}

// docDong đọc một dòng, có trần MaxBanTin.
func (c *Client) docDong() ([]byte, error) {
	var buf []byte
	for {
		phan, tiep, err := c.ra.ReadLine()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("plugin %q đóng ống trước khi trả lời%s",
					c.m.Plugin.Ten, c.themLoi())
			}
			return nil, err
		}
		buf = append(buf, phan...)
		if len(buf) > MaxBanTin {
			return nil, fmt.Errorf("plugin %q gửi bản tin quá %d byte", c.m.Plugin.Ten, MaxBanTin)
		}
		if !tiep {
			return buf, nil
		}
	}
}

// themLoi gắn phần cuối stderr của plugin vào thông điệp lỗi.
//
// Không có nó thì mọi lỗi plugin đều hiện ra là "đóng ống trước khi trả lời" —
// đúng nhưng vô dụng, trong khi nguyên nhân thật đang nằm sẵn ở stderr.
func (c *Client) themLoi() string {
	if c.loi == nil {
		return ""
	}
	if s := c.loi.String(); s != "" {
		return "\n     plugin nói (stderr): " + catNgan(s)
	}
	return ""
}

func catNgan(s string) string {
	const tran = 500
	s = strings.TrimSpace(s)
	if len(s) <= tran {
		return s
	}
	return s[:tran] + "…"
}

// Dong đóng ống, đợi tiến trình chết, và dọn thư mục tạm.
func (c *Client) Dong() error {
	if c.vao != nil {
		_ = c.vao.Close() // EOF trên stdin = tín hiệu kết thúc bình thường
	}
	xong := make(chan error, 1)
	go func() { xong <- c.cmd.Wait() }()
	select {
	case <-xong:
	case <-time.After(3 * time.Second):
		// Không chịu chết thì giết. Ba giây là quá đủ cho một tiến trình chỉ phải
		// thoát khỏi vòng lặp đọc.
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		<-xong
	}
	if c.tmp != "" {
		_ = os.RemoveAll(c.tmp)
	}
	return nil
}
