// Package flow đọc và kiểm tra định nghĩa workflow khai báo được.
//
// Mục tiêu (MASTER-PLAN Pha 3): người dùng định nghĩa workflow mới mà KHÔNG sửa
// mã Go. File `flows.toml` là nguồn sự thật — workflow board sau này chỉ đọc/ghi
// đúng file này, nên flow tạo từ giao diện và flow viết tay chạy y hệt nhau.
//
// Gói này CHỈ lo đọc + kiểm tra. Phần thực thi nằm ở gói khác, để bộ kiểm tra
// chạy được ở mọi nơi (CI, workflow board) mà không cần khởi động agent nào.
package flow

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/trantiendevweb/switch-agent-pro/internal/config"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// Loại node. Danh sách theo MASTER-PLAN Pha 3.
//
// Trung thực về năng lực (nguyên tắc #6): loại nào CHƯA thực thi được thì phải
// nói ra lúc kiểm tra, chứ không để người dùng viết flow rồi mới ngã ngửa.
const (
	TypeAgent   = "agent"   // chạy coding agent qua subscription profile
	TypeShell   = "shell"   // chạy lệnh (argv, không qua shell)
	TypeApprove = "approve" // dừng chờ người duyệt
	TypeNotify  = "notify"  // báo cho người dùng
	TypeModel   = "model"   // gọi thẳng model API — chờ đường API (Pha 1/4)
	TypeTest    = "test"    // chạy commands.test của project
	TypeLint    = "lint"    // chạy commands.lint
	TypeReview  = "review"  // agent đọc kết quả bước trước
	TypeMerge   = "merge"   // gộp nhánh — hành động nguy hiểm, mặc định cần duyệt
	TypePlugin  = "plugin"  // gọi một plugin ngoài (executable riêng, JSON-RPC/stdio)
)

// implemented đánh dấu loại nào chạy được ở phiên bản hiện tại.
var implemented = map[string]bool{
	TypeAgent: true, TypeShell: true, TypeApprove: true, TypeNotify: true,
	// test/lint chạy bằng `commands.test` / `commands.lint` của .sagent/project.toml
	TypeTest: true, TypeLint: true,
	// review = agent đọc kết quả bước trước; chỉ là agent có prompt dựng sẵn
	TypeReview: true,
	// model = gọi thẳng model API. Bật 20/08: đường API đã có, đã đo thật (có
	// usage, có chuyển route dự phòng, có sổ lời gọi). Xem TypeModel trong
	// internal/flow/step.go.
	TypeModel: true,
	// còn chờ cơ chế merge an toàn
	TypeMerge: false,
	// plugin = gọi một executable ngoài qua JSON-RPC/stdio (internal/plugin).
	// Bật vì đã chạy THẬT đầu-cuối, không phải vì đã viết xong mã: xem
	// TestFlowChayPluginThat trong internal/plugin/e2e_test.go — nó build plugin
	// mẫu, dựng flow.Runner thật, và khẳng định output của bước đến từ tiến trình
	// con. Bộ chạy chưa cắm thì bước báo lỗi rõ ràng chứ không im lặng bỏ qua.
	TypePlugin: true,
}

// Chính sách khi một bước hỏng.
const (
	OnFailStop     = "stop"     // mặc định: dừng cả flow
	OnFailContinue = "continue" // ghi nhận rồi đi tiếp
	OnFailFallback = "fallback" // chạy bước fallback đã khai báo
)

// Vai trò của một bước — LOẠI VIỆC bước đó đại diện, không phải tài khoản chạy
// nó (tài khoản là `profile`).
//
// Vì sao là DỮ LIỆU chứ không phải mã: đổi `vai_tro` trong flows.toml thì cả
// mặt 2D lẫn mặt 3D đổi theo mà không ai phải sửa một dòng Go nào.
const (
	VaiCEO    = "ceo"    // quyết định cuối, gộp và chịu trách nhiệm báo cáo
	VaiLeader = "leader" // chia việc, điều phối — không tự làm
	VaiCoder  = "coder"  // viết mã hoặc viết tài liệu
	VaiTester = "tester" // chạy máy chấm, kiểm lại thứ người khác làm
	VaiSoi    = "soi"    // soi độc lập, chỉ đọc
)

// VaiTroHopLe là năm vai hợp lệ, theo thứ tự cố định để thông điệp lỗi và mặt
// web luôn liệt kê giống nhau (map của Go trả ra ngẫu nhiên).
func VaiTroHopLe() []string {
	return []string{VaiCEO, VaiLeader, VaiCoder, VaiTester, VaiSoi}
}

// LaVaiTro cho biết v có phải một trong năm vai hợp lệ không.
//
// RỖNG là KHÔNG hợp lệ ở đây theo nghĩa "không nằm trong danh sách", nhưng chỗ
// gọi phải tự bỏ qua giá trị rỗng: rỗng nghĩa là CHƯA PHÂN VAI, và đó là trạng
// thái hợp lệ. Không suy đoán vai từ tên bước hay từ sơ đồ — thà thấy rõ chỗ
// chưa khai còn hơn để máy đoán hộ rồi hiện sai.
func LaVaiTro(v string) bool {
	for _, x := range VaiTroHopLe() {
		if x == v {
			return true
		}
	}
	return false
}

// Step là một node trong DAG.
//
// Có CẢ thẻ toml lẫn json: toml cho file người dùng sửa tay, json cho bảng vẽ
// trên web. Hai bên phải khớp tên, nếu không thì flow lưu từ giao diện sẽ khác
// flow viết tay — đúng cái bug đã dính.
type Step struct {
	ID    string   `toml:"id" json:"id"`
	Type  string   `toml:"type" json:"type"`
	Needs []string `toml:"needs,omitempty" json:"needs"` // các bước phải xong trước

	// VaiTro là LOẠI VIỆC bước này đại diện: ceo | leader | coder | tester | soi.
	// Xem các hằng Vai* ở trên.
	//
	// RỖNG = CHƯA PHÂN VAI, và đó là giá trị hợp lệ: bước không khai thì hiện ở
	// phòng chung. Cố ý không có giá trị mặc định và không suy từ `type` hay từ
	// sơ đồ — máy đoán hộ rồi hiện sai còn tệ hơn để trống thấy rõ.
	VaiTro string `toml:"vai_tro,omitempty" json:"vaiTro,omitempty"`

	// agent / review
	Profile string `toml:"profile,omitempty" json:"profile,omitempty"` // "claude:phu"; rỗng = mặc định lúc chạy
	// Model chọn model cho RIÊNG bước này, ví dụ "sonnet" cho việc nhẹ và "opus"
	// cho việc nặng. Rỗng = để provider dùng mặc định của nó.
	//
	// VÌ SAO CÓ, có số đo: lượt chạy #34 tốn 9,40 USD, trong đó riêng bước
	// `code-go` là 8,18 USD — vì mọi bước đều chạy model mạnh nhất, kể cả bước
	// chỉ viết tài liệu hay gộp báo cáo.
	Model    string `toml:"model,omitempty" json:"model,omitempty"`
	Prompt   string `toml:"prompt,omitempty" json:"prompt,omitempty"`     // hỗ trợ {{bien}}
	Copies   int    `toml:"copies,omitempty" json:"copies,omitempty"`     // số agent song song, mặc định 1
	Worktree bool   `toml:"worktree,omitempty" json:"worktree,omitempty"` // mỗi agent một git worktree
	// TuDuyetQuyen: cho agent tự duyệt mọi tool ở bước NÀY. Mặc định tắt.
	TuDuyetQuyen bool `toml:"tu_duyet_quyen,omitempty" json:"tuDuyetQuyen,omitempty"`

	// shell
	Run []string `toml:"run,omitempty" json:"run,omitempty"` // argv — CỐ Ý không nhận chuỗi shell

	// approve / notify
	Message string `toml:"message,omitempty" json:"message,omitempty"`

	// plugin: TÊN plugin đã cài (<kho>/plugins/<ten> hoặc <dự án>/.sagent/plugins/<ten>).
	//
	// Chỉ là cái tên, cố ý không phải đường dẫn tới executable: flows.toml là
	// file người ta gửi cho nhau, và một đường dẫn trong đó là một lời mời chạy
	// binary tuỳ ý. Thứ chạy được phải nằm trong một thư mục plugin có manifest
	// đã khai quyền — xem internal/plugin.
	Plugin string `toml:"plugin,omitempty" json:"plugin,omitempty"`

	// Vao là ĐẦU VÀO gửi cho plugin; hỗ trợ {{bien}} và {{steps.x.output}}.
	//
	// Không dùng lại `prompt` vì hai thứ khác bản chất: prompt là chữ gửi cho một
	// mô hình, còn đây là dữ liệu gửi cho một chương trình. Trộn tên thì bảng
	// tóm tắt lượt chạy sẽ hiện "prompt" cho một bước không hỏi ai câu nào.
	Vao string `toml:"vao,omitempty" json:"vao,omitempty"`

	// ThamSo là tham số TĨNH cho plugin (`tham_so = { so_dong = "10" }`).
	//
	// Chuỗi hết, không phải kiểu tuỳ ý: giá trị đi qua JSON tới một chương trình
	// khác, và mỗi kiểu thêm vào là một chỗ hai bên hiểu khác nhau.
	ThamSo map[string]string `toml:"tham_so,omitempty" json:"thamSo,omitempty"`

	// ForEach cho phép MỘT bước chạy lặp trên một danh sách:
	//
	//	foreach = "steps.liet-ke.output"   # hoặc "vars.danh_sach"
	//	prompt  = "Rà soát file: {{item}}"
	//
	// Mỗi dòng của nguồn thành một lượt chạy, có {{item}} và {{index}}. Các lượt
	// chạy SONG SONG theo trần của dự án. Kết quả gộp lại thành output của bước.
	ForEach   string `toml:"foreach,omitempty" json:"foreach,omitempty"`
	Separator string `toml:"separator,omitempty" json:"separator,omitempty"` // mặc định: xuống dòng

	// When là điều kiện chạy; rỗng = luôn chạy. Xem when.go.
	// Không thoả thì bước bị BỎ QUA (skipped), và bước sau vẫn chạy tiếp.
	When string `toml:"when,omitempty" json:"when,omitempty"`

	// DocDuoc giới hạn bước này được đọc kết quả của ĐÚNG những bước nào:
	//
	//	doc_duoc = ["kiem-2"]
	//
	// KHÔNG khai (nil) = đọc được MỌI bước đã xong trước nó — hành vi mặc định
	// từ đầu, giữ nguyên. Khai rồi thì bước ngoài danh sách bị thay bằng một câu
	// nói rõ là đã bị chặn, chứ không phải chuỗi rỗng. Xem doc_duoc.go.
	DocDuoc []string `toml:"doc_duoc,omitempty" json:"docDuoc,omitempty"`

	// PhaiCo là HỢP ĐỒNG ĐẦU RA: kết quả bước phải chứa ít nhất một trong những
	// chuỗi này thì mới tính là xong.
	//
	//	phai_co = ["NÊN TRỘN", "KHÔNG NÊN TRỘN"]
	//
	// VÌ SAO CÓ: lượt chạy #46, bước `soi` (grok) nhận HTTP 503 từ nhà cung cấp
	// và trả về đúng một câu — "Sorry, I encountered an error: Grok API error:
	// 503 Service temporarily unavailable". CLI thoát mã 0, bản ghi không có
	// trường lỗi nào, nên bước được ghi là DONE. Bản tóm tắt in "không bước nào
	// hỏng", lượt chạy đi tiếp, và việc trộn nhánh diễn ra mà KHÔNG AI SOI.
	//
	// Đó là kiểu hỏng tệ nhất của một cổng kiểm: nó không sập, nó chỉ lặng lẽ
	// gật đầu. "Đã soi và không thấy gì" với "chưa từng được soi" là hai câu
	// khác hẳn nhau, mà nhìn vào bảng thì giống hệt.
	//
	// ĐÂY KHÔNG PHẢI DÒ CHUỖI LỖI. Dò chuỗi lỗi là đoán xem nhà cung cấp viết
	// câu xin lỗi thế nào — họ đổi câu chữ là hỏng. Còn đây là kiểm bước có
	// GIAO RA thứ nó được giao hay không, và danh sách do người viết flow khai
	// trong flows.toml chứ không nằm cứng trong mã.
	//
	// Không khai (nil) = không kiểm gì, y như trước.
	PhaiCo []string `toml:"phai_co,omitempty" json:"phaiCo,omitempty"`

	// Artifact khai những FILE bước này để lại cho bước sau: TÊN → đường dẫn
	// TƯƠNG ĐỐI trong thư mục artifact của bước.
	//
	//	[[flow.x.step]]
	//	  id       = "viet"
	//	  prompt   = "Ghi bản vá đầy đủ ra {{artifact_dir}}/ban-va.diff"
	//	  artifact = { ban-va = "ban-va.diff" }
	//
	//	[[flow.x.step]]
	//	  id     = "ap"
	//	  needs  = ["viet"]
	//	  run    = ["git", "apply", "{{artifacts.ban-va}}"]
	//
	// VÌ SAO KHÔNG DÙNG `{{steps.viet.output}}`: đường đó đi qua hai cái trần
	// (store.MaxStepOutput 32 KiB, rồi flow.MaxInject 6.000 ký tự) và cả hai đều
	// CẮT PHẦN ĐẦU. Một bản vá mất phần đầu vẫn trông như một bản vá.
	//
	// Đường dẫn phải TƯƠNG ĐỐI: flows.toml là file người ta gửi cho nhau, và
	// đường dẫn tuyệt đối trong đó hoặc là rác trên máy người nhận, hoặc là một
	// lời mời ghi đè file bất kỳ của họ. Xem artifact.go.
	Artifact map[string]string `toml:"artifact,omitempty" json:"artifact,omitempty"`

	// Idempotent: chạy lại thì KHÔNG làm lại việc lượt trước đã làm xong.
	//
	//	idempotent = true
	//
	// "Đã làm rồi" = có một lượt chạy TRƯỚC đây làm XONG một bước có cùng KHOÁ,
	// mà khoá là băm của toàn bộ thứ quyết định kết quả bước — id, loại, câu hỏi
	// ĐÃ THAY BIẾN, tài khoản, model, tham số, hợp đồng đầu ra. Sửa prompt là đổi
	// khoá, và bước chạy lại.
	//
	// TẮT MẶC ĐỊNH, và cố ý không có cách bật cho cả flow một lượt: bộ chạy
	// KHÔNG nhìn thấy cây mã trên đĩa, HEAD của git hay đồng hồ, nên với bước
	// `shell`/`test`/`lint` thì "khoá không đổi" KHÔNG có nghĩa là "kết quả không
	// đổi". Bật nhầm ở đó là tin rằng test hôm nay vẫn xanh vì hôm qua nó xanh.
	//
	// Xem idempotent.go — ở đó có cả ba cách định nghĩa khoá đã cân nhắc và hậu
	// quả của từng cách.
	Idempotent bool `toml:"idempotent,omitempty" json:"idempotent,omitempty"`

	// Route là route API cho node `model`. Rỗng = `default_route` rồi tới route
	// dự phòng, y như `sagent api "câu hỏi"`. Không dùng cho node khác.
	Route string `toml:"route,omitempty" json:"route,omitempty"`

	// điều khiển chung
	TimeoutSec int    `toml:"timeout_sec,omitempty" json:"timeout_sec,omitempty"`
	Retry      int    `toml:"retry,omitempty" json:"retry,omitempty"`
	OnFailure  string `toml:"on_failure,omitempty" json:"on_failure,omitempty"` // stop | continue | fallback
	Fallback   string `toml:"fallback,omitempty" json:"fallback,omitempty"`

	// Vị trí trên bảng vẽ. Chỉ để trình soạn thảo bày lại đúng chỗ; bộ thực thi
	// hoàn toàn bỏ qua. Sửa file bằng tay mà không có x/y thì bảng tự xếp.
	X int `toml:"x,omitempty" json:"x,omitempty"`
	Y int `toml:"y,omitempty" json:"y,omitempty"`
}

// Flow là một workflow.
type Flow struct {
	Name  string            `toml:"-" json:"name"`
	Desc  string            `toml:"desc" json:"desc"`
	Vars  map[string]string `toml:"vars,omitempty" json:"vars"`
	Steps []Step            `toml:"step" json:"step"`
}

// File là nội dung một flows.toml.
type File struct {
	Version int             `toml:"version"`
	Flows   map[string]Flow `toml:"flow"`
}

// Problem là một lỗi hoặc cảnh báo khi kiểm tra.
type Problem struct {
	Flow string
	Step string
	Msg  string
	Warn bool // true = cảnh báo (vẫn chạy được), false = lỗi
}

func (p Problem) String() string {
	where := p.Flow
	if p.Step != "" {
		where += "." + p.Step
	}
	kind := "✗"
	if p.Warn {
		kind = "!"
	}
	return fmt.Sprintf("%s %-24s %s", kind, where, p.Msg)
}

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Paths là các nơi tìm flows.toml, dưới đè lên trên — cùng tầng với config.
func Paths(dir string) []string {
	var out []string
	global := filepath.Join(paths.AccountsRoot(), "flows.toml")
	if _, err := os.Stat(global); err == nil {
		out = append(out, global)
	}
	if p := config.FindProjectFile(dir); p != "" {
		fp := filepath.Join(filepath.Dir(p), "flows.toml")
		if _, err := os.Stat(fp); err == nil {
			out = append(out, fp)
		}
	}
	return out
}

// Load đọc mọi flows.toml áp dụng cho dir, gộp lại (dự án đè toàn cục), và
// thêm các flow mẫu dựng sẵn nếu người dùng chưa định nghĩa trùng tên.
func Load(dir string) (map[string]Flow, []string, error) {
	out := map[string]Flow{}
	for name, f := range Builtin() {
		out[name] = f
	}
	srcs := Paths(dir)
	for _, p := range srcs {
		var file File
		if _, err := toml.DecodeFile(p, &file); err != nil {
			return nil, srcs, fmt.Errorf("%s: %w", p, err)
		}
		if file.Version != 0 && file.Version != 1 {
			return nil, srcs, fmt.Errorf("%s: version = %d, công cụ này chỉ hiểu 1", p, file.Version)
		}
		for name, f := range file.Flows {
			f.Name = name
			out[name] = f
		}
	}
	for name, f := range out {
		f.Name = name
		out[name] = f
	}
	return out, srcs, nil
}

// Names trả về tên flow đã sắp xếp.
func Names(m map[string]Flow) []string {
	out := make([]string, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Validate kiểm tra một flow: id hợp lệ, phụ thuộc có thật, KHÔNG có chu trình,
// và loại node đã thực thi được chưa.
func Validate(f Flow) []Problem {
	var ps []Problem
	add := func(step, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: step, Msg: msg}) }
	warn := func(step, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: step, Msg: msg, Warn: true}) }

	if len(f.Steps) == 0 {
		add("", "flow không có bước nào")
		return ps
	}

	seen := map[string]bool{}
	for _, s := range f.Steps {
		switch {
		case s.ID == "":
			add("", "có bước thiếu id")
			continue
		case !idRe.MatchString(s.ID):
			add(s.ID, "id chỉ được dùng chữ thường, số, - và _")
		case seen[s.ID]:
			add(s.ID, "id bị trùng")
		}
		seen[s.ID] = true
	}

	for _, s := range f.Steps {
		// Vai lạ chỉ là CẢNH BÁO, không chặn: người dùng phải đặt được vai mới
		// (ví dụ "designer") trước khi công cụ hỗ trợ nó, chứ không phải chờ
		// bản mới. Kiểm trước phần `type` để bước thiếu type vẫn được soi vai.
		if s.VaiTro != "" && !LaVaiTro(s.VaiTro) {
			warn(s.ID, fmt.Sprintf("vai_tro = %q không nằm trong danh sách; năm vai hợp lệ: %s (để RỖNG = chưa phân vai)",
				s.VaiTro, strings.Join(VaiTroHopLe(), ", ")))
		}

		if s.Type == "" {
			add(s.ID, "thiếu type")
			continue
		}
		impl, known := implemented[s.Type]
		if !known {
			add(s.ID, fmt.Sprintf("type %q không có; các loại hợp lệ: %s", s.Type, strings.Join(knownTypes(), ", ")))
			continue
		}
		if !impl {
			warn(s.ID, fmt.Sprintf("type %q đã có trong thiết kế nhưng CHƯA chạy được ở bản này", s.Type))
		}

		// yêu cầu riêng theo loại
		switch s.Type {
		case TypeAgent, TypeReview:
			if s.Prompt == "" {
				add(s.ID, "bước agent cần `prompt`")
			}
			if s.Copies < 0 {
				add(s.ID, "copies không được âm")
			}
		case TypeShell, TypeTest, TypeLint:
			if len(s.Run) == 0 && s.Type == TypeShell {
				add(s.ID, "bước shell cần `run` (dạng danh sách đối số, ví dụ run = [\"go\", \"test\", \"./...\"])")
			}
		case TypeApprove, TypeNotify:
			if s.Message == "" {
				warn(s.ID, "nên có `message` để người đọc biết đang duyệt/báo cái gì")
			}
		case TypePlugin:
			if s.Plugin == "" {
				add(s.ID, "bước plugin cần `plugin` là TÊN plugin đã cài, ví dụ plugin = \"tom-luoc\"")
			} else if !idRe.MatchString(s.Plugin) {
				add(s.ID, fmt.Sprintf("plugin = %q không phải một tên hợp lệ — chỉ chữ thường, số, - và _ "+
					"(đây là TÊN plugin, không phải đường dẫn)", s.Plugin))
			}
		}

		// Khai `plugin` ở một bước KHÔNG phải type plugin là một dòng chết: nó
		// nằm đó trông như có tác dụng, và người viết flow sẽ tưởng bước đang gọi
		// plugin. Cùng lớp hỏng với khoá lạ trong manifest, nên cùng cách xử lý.
		if s.Plugin != "" && s.Type != TypePlugin {
			add(s.ID, fmt.Sprintf("khai `plugin` nhưng type = %q — chỉ bước type = \"plugin\" mới gọi plugin", s.Type))
		}

		if s.ForEach != "" {
			if !strings.HasPrefix(s.ForEach, "steps.") && !strings.HasPrefix(s.ForEach, "vars.") {
				add(s.ID, fmt.Sprintf("foreach = %q phải trỏ vào steps.<id>.output hoặc vars.<tên>", s.ForEach))
			}
			if s.Type == TypeApprove {
				add(s.ID, "bước approve không lặp được — mỗi lượt sẽ là một lần chờ người duyệt")
			}
		}

		switch s.OnFailure {
		case "", OnFailStop, OnFailContinue:
		case OnFailFallback:
			if s.Fallback == "" {
				add(s.ID, "on_failure = \"fallback\" thì phải khai báo `fallback` là id bước khác")
			} else if !seen[s.Fallback] {
				add(s.ID, fmt.Sprintf("fallback trỏ tới bước %q không tồn tại", s.Fallback))
			}
		default:
			add(s.ID, fmt.Sprintf("on_failure = %q không hợp lệ (stop | continue | fallback)", s.OnFailure))
		}

		for _, n := range s.Needs {
			if !seen[n] {
				add(s.ID, fmt.Sprintf("needs trỏ tới bước %q không tồn tại", n))
			}
			if n == s.ID {
				add(s.ID, "bước không thể phụ thuộc chính nó")
			}
		}
	}

	if cyc := findCycle(f.Steps); len(cyc) > 0 {
		add("", "có chu trình phụ thuộc: "+strings.Join(cyc, " → "))
	}
	// Bước approve KHÔNG phải hàng rào toàn cục — nó chỉ chặn những bước có khai
	// `needs` tới nó. Người viết flow rất dễ đặt một bước `approve` rồi tưởng cả
	// luồng dừng lại chờ mình, trong khi bước `deploy` bên cạnh vẫn chạy vì quên
	// khai phụ thuộc. Đúng ngữ nghĩa DAG, nhưng sai ý định — và sai theo hướng
	// nguy hiểm.
	//
	// Không tự ý bắt mọi bước phụ thuộc vào approve: sửa ngữ nghĩa sau lưng người
	// dùng còn tệ hơn. Chỉ nói ra.
	coAiCho := map[string]bool{}
	for _, s := range f.Steps {
		for _, n := range s.Needs {
			coAiCho[n] = true
		}
	}
	for _, s := range f.Steps {
		if s.Type == TypeApprove && !coAiCho[s.ID] {
			warn(s.ID, "bước approve này không chặn bước nào — không có bước nào khai `needs = [\""+s.ID+"\"]`. "+
				"Nó sẽ dừng luồng nhưng các bước khác VẪN CHẠY song song với nó.")
		}
	}

	// `doc_duoc` khai hỏng chỉ CẢNH BÁO — xem doc_duoc.go. Đặt cuối cùng vì nó
	// cần thứ tự đợt, mà thứ tự đợt chỉ có nghĩa khi phần `needs` đã được soi.
	ps = append(ps, VanDeDocDuoc(f)...)

	// `artifact` — xem artifact.go. Cũng cần thứ tự đợt, cùng lý do.
	ps = append(ps, VanDeArtifact(f)...)

	// `idempotent` — xem idempotent.go.
	ps = append(ps, VanDeIdempotent(f)...)

	return ps
}

// Order sắp xếp các bước theo thứ tự chạy (topological). Lỗi nếu có chu trình.
func Order(f Flow) ([]Step, error) {
	byID := map[string]Step{}
	indeg := map[string]int{}
	children := map[string][]string{}
	for _, s := range f.Steps {
		byID[s.ID] = s
		if _, ok := indeg[s.ID]; !ok {
			indeg[s.ID] = 0
		}
		for _, n := range s.Needs {
			indeg[s.ID]++
			children[n] = append(children[n], s.ID)
		}
	}
	// Sắp xếp tên để thứ tự ổn định giữa các lần chạy — không thì log mỗi lần một khác.
	var ready []string
	for id, d := range indeg {
		if d == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)

	var out []Step
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		out = append(out, byID[id])
		var next []string
		for _, c := range children[id] {
			indeg[c]--
			if indeg[c] == 0 {
				next = append(next, c)
			}
		}
		sort.Strings(next)
		ready = append(ready, next...)
		sort.Strings(ready)
	}
	if len(out) != len(f.Steps) {
		return nil, fmt.Errorf("flow %q có chu trình phụ thuộc", f.Name)
	}
	return out, nil
}

// findCycle trả về một chu trình nếu có (để báo lỗi cho người đọc hiểu).
func findCycle(steps []Step) []string {
	needs := map[string][]string{}
	for _, s := range steps {
		needs[s.ID] = s.Needs
	}
	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := map[string]int{}
	var path []string
	var cycle []string

	var visit func(string) bool
	visit = func(id string) bool {
		color[id] = grey
		path = append(path, id)
		for _, n := range needs[id] {
			if _, ok := needs[n]; !ok {
				continue // phụ thuộc không tồn tại — đã báo ở chỗ khác
			}
			switch color[n] {
			case grey:
				for i, p := range path {
					if p == n {
						cycle = append(append([]string{}, path[i:]...), n)
						return true
					}
				}
				cycle = append(append([]string{}, path...), n)
				return true
			case white:
				if visit(n) {
					return true
				}
			}
		}
		path = path[:len(path)-1]
		color[id] = black
		return false
	}

	ids := make([]string, 0, len(needs))
	for id := range needs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if color[id] == white {
			path = nil
			if visit(id) {
				return cycle
			}
		}
	}
	return nil
}

func knownTypes() []string {
	out := make([]string, 0, len(implemented))
	for t := range implemented {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// MaxInject là trần phần kết quả được nhét vào prompt của bước sau.
//
// Nhỏ hơn trần lưu trữ: agent có thể xuất hàng chục nghìn ký tự, nhét hết vào
// prompt là đốt ngữ cảnh (và tiền) mà thường chỉ phần cuối mới có kết luận.
const MaxInject = 6000

// Expand thay {{bien}} bằng giá trị trong vars (và ghi đè từ tham số dòng lệnh).
func Expand(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}

// conSotOutput bắt các {{steps.<id>.output}} mà Expand KHÔNG thay được.
var conSotOutput = regexp.MustCompile(`\{\{steps\.([^.{}]+)\.output\}\}`)

// conSotArtifact bắt các {{artifacts.<tên>}} mà Expand KHÔNG thay được.
//
// Cùng lớp nguy hiểm với conSotOutput nhưng tệ hơn một bậc: một placeholder
// output còn sót lọt vào prompt thì agent đọc ra chữ vô nghĩa; một placeholder
// ARTIFACT còn sót lọt vào `run` thì lệnh nhận một chuỗi trông y như đường dẫn.
var conSotArtifact = regexp.MustCompile(`\{\{artifacts\.([^.{}]+)\}\}`)

// ExpandChay thay biến như Expand, rồi CHỐT các {{steps.<id>.output}} còn sót
// lại bằng một câu nói thật thay vì để nguyên chữ sống.
//
// Vì sao cần: Expand chỉ thay những khoá CÓ trong map. Bước hỏng — hoặc chạy
// xong mà không trả về gì — không để lại `steps.<id>.output` nào, nên
// placeholder đi thẳng vào prompt dưới dạng văn bản.
//
// Đo tại lần chạy #29: bước `kiem-cuoi` hỏng, và người soi được gửi nguyên văn
//
//	Máy chấm nói gì:
//	{{steps.kiem-cuoi.output}}
//
// Người soi không hề nhận được phán quyết của máy chấm, nhưng vẫn phán như thể
// có. Cả lời hứa "máy chấm quyết định, không phải lời agent" bốc hơi trong im
// lặng — đúng kiểu hỏng mà dự án này sợ nhất.
//
// Vì sao KHÔNG chốt thẳng trong Expand: `sagent flow show` dùng Expand để in
// thử prompt lúc CHƯA chạy, khi đó chưa bước nào có kết quả là chuyện bình
// thường. Chốt ở đó là nói dối theo chiều ngược lại.
func ExpandChay(s string, vars map[string]string) string {
	out := conSotOutput.ReplaceAllString(Expand(s, vars), `(bước "$1" không để lại kết quả)`)
	// Artifact chưa có thì cũng phải NÓI RA, không để nguyên chữ sống. Bước sản
	// xuất chưa chạy xong (hoặc hỏng) là lúc duy nhất chuyện này xảy ra — xem
	// MoiTruongArtifact.
	return conSotArtifact.ReplaceAllString(out, `(artifact "$1" chưa có — bước sản xuất nó chưa chạy xong)`)
}

// BuocConSot trả về id bước đầu tiên còn placeholder chưa thay, hoặc "" nếu
// không còn. Dùng cho chỗ KHÔNG được phép đoán bừa — xem bước shell trong do().
func BuocConSot(s string, vars map[string]string) string {
	if m := conSotOutput.FindStringSubmatch(Expand(s, vars)); m != nil {
		return m[1]
	}
	return ""
}

// ArtifactConSot trả về TÊN artifact đầu tiên còn placeholder chưa thay, hoặc ""
// nếu không còn.
//
// Cùng vai trò với BuocConSot và dùng ở cùng chỗ: bước shell KHÔNG được chốt
// placeholder thành một câu tiếng Việt. `cat (artifact "x" chưa có...)` là một
// tên file bịa; lệnh sẽ hỏng bằng "no such file" và người đọc đi tìm sai chỗ
// suốt buổi. Thiếu thì dừng ngay và nói rõ thiếu artifact nào.
func ArtifactConSot(s string, vars map[string]string) string {
	if m := conSotArtifact.FindStringSubmatch(Expand(s, vars)); m != nil {
		return m[1]
	}
	return ""
}

// WithOutputs trả về bản sao của vars, thêm khoá `steps.<id>.output` để bước sau
// dùng được kết quả bước trước:
//
//	prompt = "Đọc kết quả rà soát rồi tóm tắt: {{steps.ra-soat.output}}"
//
// Kết quả dài thì cắt phần ĐẦU, giữ phần CUỐI (kết luận thường ở cuối) và nói
// rõ là đã cắt — thà mất phần giữa còn hơn để người đọc tưởng đó là toàn bộ.
func WithOutputs(vars map[string]string, outputs map[string]string) map[string]string {
	out := make(map[string]string, len(vars)+len(outputs))
	for k, v := range vars {
		out[k] = v
	}
	for id, o := range outputs {
		if len(o) > MaxInject {
			o = "…(đã cắt bớt phần đầu, giữ " + itoa(MaxInject) + " ký tự cuối)…\n" +
				o[len(o)-MaxInject:]
		}
		out["steps."+id+".output"] = o
	}
	return out
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
