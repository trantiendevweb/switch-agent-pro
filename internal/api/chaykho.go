// Chạy khan (dry run) — action "flow.kho".
//
// Trả lời đúng câu người ta hỏi trước khi bấm chạy: "cái này sẽ bật mấy agent,
// bằng tài khoản nào, hỏi chúng nó cái gì, và có chỗ nào hỏng sẵn không?" —
// mà KHÔNG ghi một dòng nào vào sổ, không tạo worktree, không gọi agent.
//
// Vì sao cần, bằng số thật: ngày 19/08 có ba lượt chạy thật (#30, #32, #33) mà
// mục đích chỉ là xem cổng kiểm tài khoản nói gì. Mỗi lượt đốt hạn mức thuê bao
// và để lại một lượt rác trong `flow runs` phải huỷ tay. Cổng kiểm ấy vốn đã
// biết câu trả lời TRƯỚC khi bật agent nào — chỉ là không có đường nào hỏi nó
// mà không chạy thật.
package api

import (
	"fmt"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// BuocKho là một bước trong kế hoạch chạy khan.
type BuocKho struct {
	ID    string   `json:"id"`
	Type  string   `json:"type"`
	Needs []string `json:"needs"`

	// TaiKhoan là tài khoản bước này sẽ chạy bằng — khai trong bước, hoặc mặc
	// định của lượt chạy. Bước không dùng agent thì RỖNG: đoán bừa một cái tên
	// tài khoản còn tệ hơn không nói.
	TaiKhoan string `json:"taiKhoan,omitempty"`
	// Model phải hiện ở đây: cả lý do sinh ra `--kho` là xem TRƯỚC khi tiêu tiền,
	// mà model chính là thứ quyết định tiêu bao nhiêu. Không hiện thì người dùng
	// khai `model = "sonnet"` để tiết kiệm rồi vẫn phải chạy thật mới biết nó có
	// vào hay không — đúng kiểu "làm rồi mà không kiểm được".
	Model    string `json:"model,omitempty"`
	SoAgent  int    `json:"soAgent,omitempty"`
	Worktree bool   `json:"worktree,omitempty"`
	// TuDuyetQuyen phải hiện ở đây: đọc kế hoạch đúng là lúc người ta quyết định
	// có chạy hay không, và "bước này được tự duyệt mọi quyền" là thứ nặng nhất
	// trong quyết định đó.
	TuDuyetQuyen bool `json:"tuDuyetQuyen,omitempty"`

	// Prompt là thứ agent SẼ nhận thật, sau khi đã thay biến. Bước shell thì là
	// dòng lệnh, bước notify/approve thì là lời nhắn.
	Prompt string `json:"prompt,omitempty"`

	// Lap là nguồn danh sách của bước `foreach`. Có nó thì SoAgent của bước này
	// chỉ là mức tối thiểu: danh sách dài bao nhiêu chỉ lộ ra lúc chạy thật, và
	// mỗi mục là một lượt agent nữa. Nói rõ chỗ chưa biết còn hơn in một con số
	// gọn gàng mà sai.
	Lap string `json:"lap,omitempty"`

	// ConSot là id bước mà prompt này đang chờ kết quả NHƯNG bước đó không chạy
	// xong trước nó (chạy sau, chạy cùng đợt, hoặc không hề tồn tại). Rỗng =
	// không có chỗ nào hụt.
	//
	// Đây là lỗi kiểu lượt chạy #29: prompt ghi {{steps.kiem-cuoi.output}},
	// bước đó không để lại gì, và agent nhận nguyên chữ sống làm đề bài.
	ConSot string `json:"conSot,omitempty"`
}

// DotKho là một đợt của kế hoạch: mọi bước trong đó chạy SONG SONG.
type DotKho struct {
	So       int       `json:"so"`
	ChoDuyet bool      `json:"choDuyet"`
	Buoc     []BuocKho `json:"buoc"`
}

// KeHoachKho là toàn bộ câu trả lời của một lượt chạy khan.
type KeHoachKho struct {
	Flow string            `json:"flow"`
	Desc string            `json:"desc"`
	Dir  string            `json:"dir"`
	Vars map[string]string `json:"vars"`
	Dot  []DotKho          `json:"dot"`

	// Van là lỗi + cảnh báo của flow.Validate. Kế hoạch KHÔNG bị chặn vì chúng:
	// xem được vấn đề mà không phải chạy chính là việc của chạy khan.
	Van []VanDe `json:"vanDe"`

	// TaiKhoanHong là những tài khoản flow cần mà dùng không được — cùng cổng
	// kiểm mà `flow run` dùng để chặn, chỉ khác là ở đây nó chỉ nói.
	TaiKhoanHong []TaiKhoanHong `json:"taiKhoanHong"`

	// SoAgent là tổng số phiên agent lượt chạy thật sẽ bật (cộng cả copies).
	// Con số này là lý do chính người ta chạy khan: nó tỉ lệ thẳng với hạn mức
	// sắp bị đốt.
	SoAgent int `json:"soAgent"`

	// CoLap = có bước `foreach`, nên SoAgent ở trên là mức TỐI THIỂU chứ không
	// phải con số cuối cùng.
	CoLap bool `json:"coLap"`
}

// VanDe là flow.Problem dưới dạng gửi đi được cho mặt web.
type VanDe struct {
	Buoc string `json:"buoc"`
	Msg  string `json:"msg"`
	Warn bool   `json:"warn"`
}

// FlowChayKho — action "flow.kho". Dựng kế hoạch chạy mà KHÔNG chạy.
//
// Cố ý KHÔNG nhận context và KHÔNG đụng a.db: nó không có gì để huỷ giữa chừng
// và không có gì để ghi. Ai đọc hàm này cũng thấy ngay điều đó, thay vì phải
// tin vào một lời hứa trong bình luận.
func (a *API) FlowChayKho(dir, name string, vars map[string]string, defaultProfile Addr) (KeHoachKho, error) {
	flows, _, err := flow.Load(dir)
	if err != nil {
		return KeHoachKho{}, err
	}
	f, ok := flows[name]
	if !ok {
		return KeHoachKho{}, fmt.Errorf("không có flow %q (xem: sagent flow list)", name)
	}

	// Biến: mặc định của flow, rồi tham số dòng lệnh đè lên — đúng thứ tự
	// Runner.Start làm, nếu không thì prompt in ra sẽ khác prompt gửi đi.
	bien := map[string]string{}
	for k, v := range f.Vars {
		bien[k] = v
	}
	for k, v := range vars {
		bien[k] = v
	}

	kh := KeHoachKho{Flow: name, Desc: f.Desc, Dir: dir, Vars: bien}
	for _, p := range flow.Validate(f) {
		kh.Van = append(kh.Van, VanDe{Buoc: p.Step, Msg: p.Msg, Warn: p.Warn})
	}
	hong, err := a.KiemTaiKhoanFlow(f, defaultProfile)
	if err != nil {
		return kh, err
	}
	kh.TaiKhoanHong = hong

	dots, err := flow.Dot(f)
	if err != nil {
		return kh, err
	}

	// env lớn dần theo từng đợt: bước ở đợt sau đọc được kết quả của MỌI bước
	// đợt trước, và KHÔNG đọc được bước cùng đợt (chúng chạy song song, chưa
	// bước nào xong). Nhờ vậy BuocConSot bắt đúng chỗ hụt thật.
	env := map[string]string{}
	for k, v := range bien {
		env[k] = v
	}
	for _, d := range dots {
		dk := DotKho{So: d.So, ChoDuyet: d.ChoDuyet}
		for _, s := range d.Buoc {
			dk.Buoc = append(dk.Buoc, a.buocKho(s, env, defaultProfile, &kh.SoAgent))
			if s.ForEach != "" {
				kh.CoLap = true
			}
		}
		kh.Dot = append(kh.Dot, dk)
		for _, s := range d.Buoc {
			env["steps."+s.ID+".output"] = fmt.Sprintf("(kết quả bước %q)", s.ID)
		}
	}
	return kh, nil
}

// buocKho mô tả một bước, cộng dồn số agent vào tong.
func (a *API) buocKho(s flow.Step, env map[string]string, mac Addr, tong *int) BuocKho {
	b := BuocKho{ID: s.ID, Type: s.Type, Needs: s.Needs, Worktree: s.Worktree,
		TuDuyetQuyen: s.TuDuyetQuyen, Lap: s.ForEach}
	if b.Needs == nil {
		b.Needs = []string{}
	}
	switch s.Type {
	case flow.TypeAgent, flow.TypeReview:
		b.Model = s.Model
		b.TaiKhoan = s.Profile
		if b.TaiKhoan == "" && mac.Account != "" {
			b.TaiKhoan = mac.String()
		}
		b.SoAgent = s.Copies
		if b.SoAgent < 1 {
			b.SoAgent = 1
		}
		*tong += b.SoAgent
		b.Prompt = flow.Expand(s.Prompt, env)
		b.ConSot = flow.BuocConSot(s.Prompt, env)
	case flow.TypeShell, flow.TypeTest, flow.TypeLint:
		dong := ""
		for i, arg := range s.Run {
			if i > 0 {
				dong += " "
			}
			dong += flow.Expand(arg, env)
			if b.ConSot == "" {
				b.ConSot = flow.BuocConSot(arg, env)
			}
		}
		b.Prompt = dong
	default:
		b.Prompt = flow.Expand(s.Message, env)
		b.ConSot = flow.BuocConSot(s.Message, env)
	}
	return b
}
