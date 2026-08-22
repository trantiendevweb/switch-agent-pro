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
	Model string `json:"model,omitempty"`
	// VaiTro là LOẠI VIỆC bước này đại diện (ceo/leader/coder/tester/soi), lấy
	// nguyên từ flows.toml. Đọc kế hoạch chạy khan là lúc người ta hỏi "ai làm
	// gì" — tài khoản trả lời "bằng nick nào", vai trò trả lời "với tư cách gì".
	// Bước không khai thì RỖNG: không đoán hộ.
	VaiTro   string `json:"vaiTro,omitempty"`
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

	// DocDuoc là câu mô tả quyền đọc kết quả của bước này: "mọi bước trước" khi
	// nó chưa khai `doc_duoc`, hoặc danh sách bước nó được đọc.
	//
	// Là CÂU chứ không phải danh sách, vì ba trạng thái (chưa khai / khai rỗng /
	// khai danh sách) phải phân biệt được, mà một mảng JSON rỗng thì không nói
	// được nó là "chưa khai" hay "cấm hết".
	DocDuoc string `json:"docDuoc,omitempty"`

	// ConSot là id bước mà prompt này đang chờ kết quả NHƯNG bước đó không chạy
	// xong trước nó (chạy sau, chạy cùng đợt, hoặc không hề tồn tại). Rỗng =
	// không có chỗ nào hụt.
	//
	// Đây là lỗi kiểu lượt chạy #29: prompt ghi {{steps.kiem-cuoi.output}},
	// bước đó không để lại gì, và agent nhận nguyên chữ sống làm đề bài.
	ConSot string `json:"conSot,omitempty"`

	// ---------------------------------------------------------------------
	// BỐN TRƯỜNG DƯỚI ĐÂY TRẢ LỜI CÂU "BƯỚC NÀY CÓ THẬT SỰ CHẠY KHÔNG, VÀ NÓ
	// ĐỂ LẠI GÌ" — thứ mà bản chạy khan trước đây KHÔNG nói được.
	// ---------------------------------------------------------------------

	// GoLaiCho là các bước mà bước NÀY làm nhiệm vụ gỡ lại cho.
	//
	// KHÁC RỖNG NGHĨA LÀ BƯỚC NÀY SẼ KHÔNG CHẠY trong lượt chạy suôn sẻ. Đây là
	// lỗi tệ nhất của bản chạy khan cũ: bước gỡ lại nằm trong `needs`-graph như
	// mọi bước khác nên nó hiện ra trong kế hoạch y hệt một bước sắp chạy, và
	// người đọc cộng nó vào đầu việc. Bộ thực thi thì LOẠI nó khỏi lịch chạy
	// thường (runner.execute ghi thẳng `skipped`) và chỉ gọi khi có sự cố.
	//
	// Là DANH SÁCH chứ không phải một cờ bool: một bước gỡ có thể được nhiều
	// bước trỏ tới, và "gỡ cho bước nào" mới là câu người đọc cần.
	GoLaiCho []string `json:"goLaiCho,omitempty"`

	// Compensate là id bước gỡ lại của bước NÀY (khi on_failure = "compensate").
	// Chiều ngược của GoLaiCho.
	Compensate string `json:"compensate,omitempty"`

	// Artifact là các FILE bước này hứa để lại: tên → đường dẫn tương đối.
	//
	// Đây là đường truyền thứ hai giữa các bước, không đi qua
	// {{steps.x.output}}, nên nhìn sơ đồ phụ thuộc KHÔNG thấy nó.
	Artifact map[string]string `json:"artifact,omitempty"`

	// Idempotent = bước này có thể KHÔNG CHẠY trong lượt tới, vì một lượt trước
	// đã làm xong đúng việc đó. Số agent của nó vẫn được cộng vào SoAgent: đó
	// là mức TRẦN, còn có tiêu thật hay không chỉ biết lúc tra sổ.
	Idempotent bool `json:"idempotent,omitempty"`

	// Route là đường API bước này đi — tên khai cứng, hay danh sách ứng viên
	// của node `route`, hay câu "để cấu hình quyết định". Rỗng = bước không đi
	// đường API nào.
	Route string `json:"route,omitempty"`

	// Gop là các nguồn của bước `merge`, THEO ĐÚNG THỨ TỰ SẼ GỘP.
	Gop []string `json:"gop,omitempty"`
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

	// SoBuocGoLai là số bước trong kế hoạch CHỈ chạy khi có sự cố.
	//
	// Có ở cấp kế hoạch chứ không chỉ ở từng bước, vì câu người đọc hỏi là "kế
	// hoạch này có mấy bước, mấy cái sẽ chạy thật" — mà bảng cũ trả lời sai câu
	// đó theo hướng THỪA, và thừa ở đây nghĩa là người ta tưởng có một bước dọn
	// dẹp sẽ chạy trong khi nó chỉ chạy nếu hỏng.
	SoBuocGoLai int `json:"soBuocGoLai,omitempty"`
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

	// `xong` lớn dần theo từng đợt: bước ở đợt sau đọc được kết quả của MỌI bước
	// đợt trước, và KHÔNG đọc được bước cùng đợt (chúng chạy song song, chưa
	// bước nào xong). Nhờ vậy BuocConSot bắt đúng chỗ hụt thật.
	//
	// Giữ nguyên dạng `id -> kết quả` chứ không dựng thẳng khoá
	// `steps.<id>.output`, để còn chạy qua ĐÚNG bộ lọc quyền đọc mà bộ thực thi
	// dùng (flow.LocDocDuoc). Chạy khan mà bỏ qua bộ lọc thì prompt in ra khác
	// prompt gửi đi — đúng thứ tính năng này sinh ra để chống.
	// Bước nào là bước GỠ LẠI — tính một lần, đúng cách bộ thực thi tính.
	goLai := flow.BuocGoLai(f)

	xong := map[string]string{}
	for _, d := range dots {
		dk := DotKho{So: d.So, ChoDuyet: d.ChoDuyet}
		for _, s := range d.Buoc {
			env := flow.WithOutputs(bien, flow.LocDocDuoc(s, xong))
			b := a.buocKho(s, env, defaultProfile, &kh.SoAgent, goLai[s.ID])
			if goLai[s.ID] {
				b.GoLaiCho = flow.BuocDuocGoLaiBoi(f, s.ID)
				kh.SoBuocGoLai++
			}
			dk.Buoc = append(dk.Buoc, b)
			if s.ForEach != "" {
				kh.CoLap = true
			}
		}
		kh.Dot = append(kh.Dot, dk)
		for _, s := range d.Buoc {
			xong[s.ID] = fmt.Sprintf("(kết quả bước %q)", s.ID)
		}
	}
	return kh, nil
}

// buocKho mô tả một bước, cộng dồn số agent vào tong.
//
// laGoLai = bước này là bước GỠ LẠI của bước khác, tức nó KHÔNG chạy trong lượt
// suôn sẻ. Khi đó số agent của nó KHÔNG được cộng vào tổng: cả lý do người ta
// chạy khan là xem "sắp đốt bao nhiêu hạn mức", và đếm cả một bước chỉ chạy khi
// hỏng vào đó là trả lời sai câu đang được hỏi.
func (a *API) buocKho(s flow.Step, env map[string]string, mac Addr, tong *int, laGoLai bool) BuocKho {
	// VaiTro gán cho MỌI loại bước, không riêng bước agent: `kiem-1` là bước
	// shell nhưng vẫn là việc của tester.
	b := BuocKho{ID: s.ID, Type: s.Type, Needs: s.Needs, Worktree: s.Worktree,
		TuDuyetQuyen: s.TuDuyetQuyen, Lap: s.ForEach, VaiTro: s.VaiTro,
		DocDuoc: flow.MoTaDocDuoc(s), Artifact: s.Artifact, Idempotent: s.Idempotent,
		Route: flow.MoTaRoute(s)}
	if s.OnFailure == flow.OnFailCompensate {
		b.Compensate = s.Compensate
	}
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
		if !laGoLai {
			*tong += b.SoAgent
		}
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
	// Node `model` và `plugin` TỪNG rơi vào nhánh `default` bên dưới, tức là
	// bảng chạy khan đọc `s.Message` của chúng — một trường luôn rỗng. Hậu quả:
	// bước gọi model hiện ra không kèm một chữ nào của câu hỏi, đúng ở cái bảng
	// sinh ra để trả lời "nó sẽ hỏi chúng nó cái gì".
	case flow.TypeModel:
		b.Model = s.Model
		b.Prompt = flow.Expand(s.Prompt, env)
		b.ConSot = flow.BuocConSot(s.Prompt, env)
		if b.ConSot == "" {
			// `route` cũng nhận {{steps.x.output}} từ bản này. Thiếu kết quả ở
			// đó thì bước không biết đi đường nào — cùng mức nghiêm trọng với
			// thiếu prompt, nên báo cùng một chỗ.
			b.ConSot = flow.BuocConSot(s.Route, env)
		}
	case flow.TypePlugin:
		// Với plugin thì "câu hỏi" là `vao` — dữ liệu gửi cho một chương trình,
		// không phải chữ gửi cho một mô hình. Cùng cách phân biệt mà Step.Vao đã
		// nói rõ, giữ nguyên ở đây.
		b.Prompt = flow.Expand(s.Vao, env)
		b.ConSot = flow.BuocConSot(s.Vao, env)
	case flow.TypeMerge:
		// Thứ tự gộp LÀ thứ tự `needs`, và đó là thứ đáng hiện nhất ở đây: hai
		// lượt chạy giống hệt nhau chỉ ra cùng một kết quả khi thứ tự này cố
		// định. Xem internal/flow/merge.go.
		b.Gop = s.Needs
		if b.Gop == nil {
			b.Gop = []string{}
		}
	case flow.TypeRoute:
		// Bước `route` không hỏi ai câu nào; `Route` ở trên đã nói đủ.
	default:
		b.Prompt = flow.Expand(s.Message, env)
		b.ConSot = flow.BuocConSot(s.Message, env)
	}
	return b
}
