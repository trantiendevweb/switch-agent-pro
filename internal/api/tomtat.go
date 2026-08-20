// Tóm tắt lượt chạy — action "flow.tom-tat".
//
// Trả lời bốn câu người ta hỏi ngay sau khi một lượt chạy kết thúc: AI LÀM GÌ,
// AI CHƯA LÀM, BƯỚC NÀO HỎNG VÌ SAO, VIỆC GÌ CÒN TREO. Rồi làm thêm một việc mà
// không mặt nào đang làm: ĐỐI CHIẾU lời agent khai với git.
//
// Vì sao phải đối chiếu chứ không tin lời agent — bằng số đo thật của dự án này:
//
//   - lượt #21: bước `tho-2` trả về "I am waiting for `go test ./...` to
//     complete", được đánh dấu `done`, cả flow báo `completed` — trong khi nhánh
//     `sagent/may-1` KHÔNG có commit nào;
//   - lượt #29 và #31: người soi phán NGƯỢC nhãn trộn, gọi một nhánh 0 commit là
//     "nên trộn";
//   - lượt #34: người soi bảo hai nhánh giẫm chân nhau trong khi giao tập file
//     của chúng rỗng.
//
// Nên: lời agent là DỮ LIỆU CẦN KIỂM, không phải kết luận. Lệch thì TIN GIT.
//
// Ba cái bẫy khi viết bộ đối chiếu, cả ba đều làm bản tóm tắt sai theo kiểu im
// lặng (trông vẫn rất thuyết phục):
//
//  1. PHỦ ĐỊNH làm tắt bộ dò. Câu "KHÔNG nên trộn sagent/x" chứa đúng cụm chữ
//     "nên trộn"; đếm nó là lời khai thì bản tóm tắt vu cho agent một câu nó
//     không hề nói — và tệ hơn, nó nói ngược lại.
//  2. KHỚP CHUỖI KHÔNG BIÊN TỪ vu oan. `sagent/may-1` là chuỗi con của
//     `sagent/may-1-2`; nhánh -2 có việc thật mà nhánh -1 rỗng, khớp lỏng là
//     báo mâu thuẫn ở một nhánh không ai nhắc tới.
//  3. `git log -1 sagent/x` TRÊN NHÁNH RỖNG in ra đỉnh của main. Nhánh 0 commit
//     mà khoe một tiêu đề commit là bằng chứng giả — đúng thứ bản tóm tắt này
//     sinh ra để chống.
package api

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
	"github.com/trantiendevweb/switch-agent-pro/internal/workspace"
)

// ------------------------------------------------------------ bằng chứng git

// NhanhVao là bằng chứng git THÔ của một nhánh: đọc được sao thì đưa vào vậy.
// Tách khỏi NhanhChung để phần LUẬT (bên dưới) test được mà không cần repo thật.
type NhanhVao struct {
	BC   workspace.BangChung
	Dinh string // tiêu đề commit đỉnh, nếu người gọi có đọc
}

// NhanhChung là bằng chứng git của một nhánh SAU khi đã áp luật trình bày.
type NhanhChung struct {
	Nhanh   string `json:"nhanh"`
	Commit  int    `json:"commit"`
	Ban     bool   `json:"ban"`
	KhongRo bool   `json:"khongRo"`
	// Rong = đọc được và ĐÚNG 0 commit. Khác KhongRo: một cái là "không có gì",
	// cái kia là "không biết" — gộp hai thứ đó lại là nói bừa.
	Rong    bool   `json:"rong"`
	Dinh    string `json:"dinh,omitempty"`
	MotDong string `json:"motDong"`
}

// lamNhanhChung áp BẪY 3: nhánh rỗng thì nói thẳng là rỗng, và VỨT tiêu đề
// commit dù người gọi có đưa vào — `git log -1` trên nhánh 0 commit trả về đỉnh
// của nhánh nền, in ra là dựng bằng chứng giả cho một nhánh chẳng có gì.
func lamNhanhChung(v NhanhVao) NhanhChung {
	n := NhanhChung{
		Nhanh: v.BC.Nhanh, Commit: v.BC.Commit, Ban: v.BC.Ban,
		KhongRo: v.BC.KhongRo, Dinh: strings.TrimSpace(v.Dinh),
		MotDong: v.BC.MotDong(),
	}
	n.Rong = !v.BC.KhongRo && v.BC.Commit == 0
	if n.Rong {
		n.Dinh = ""
	}
	return n
}

// ------------------------------------------------------------------- lời khai

// Ba loại lời khai đối chiếu được bằng số commit.
const (
	KhaiTron     = "nhan-tron"
	KhaiGiamChan = "giam-chan"
	KhaiCoViec   = "co-viec"
)

var tenLoaiKhai = map[string]string{
	KhaiTron:     "nhãn trộn",
	KhaiGiamChan: "giẫm chân",
	KhaiCoViec:   "nhánh có việc",
}

// cumKhai là các cụm chữ agent hay dùng khi khai. Cố ý có cả bản không dấu:
// agent chạy trên console Windows nhiều lúc trả về tiếng Việt mất dấu.
var cumKhai = []struct{ Loai, Cum string }{
	{KhaiTron, "nên trộn"}, {KhaiTron, "nen tron"},
	{KhaiTron, "đáng trộn"}, {KhaiTron, "trộn được"}, {KhaiTron, "sẵn sàng trộn"},
	{KhaiTron, "nên gộp"}, {KhaiTron, "gộp được"}, {KhaiTron, "nên merge"},

	{KhaiGiamChan, "giẫm chân"}, {KhaiGiamChan, "dẫm chân"}, {KhaiGiamChan, "giam chan"},
	{KhaiGiamChan, "đè lên nhau"}, {KhaiGiamChan, "đụng nhau"}, {KhaiGiamChan, "trùng file"},

	{KhaiCoViec, "có commit"}, {KhaiCoViec, "co commit"}, {KhaiCoViec, "có việc"},
	{KhaiCoViec, "co viec"}, {KhaiCoViec, "đã commit"}, {KhaiCoViec, "có thay đổi"},
	{KhaiCoViec, "đã làm xong"}, {KhaiCoViec, "làm được việc"},
}

// tuPhuDinh: gặp một trong các từ này TRONG CÙNG MỆNH ĐỀ, trước cụm khai, thì
// lời khai bị lật — không tính. Thà bỏ sót một lời khai còn hơn vu cho agent
// câu ngược hẳn với điều nó nói.
var tuPhuDinh = []string{"không", "khong", "chưa", "chua", "đừng", "chẳng", "khỏi", "not"}

// Khai là MỘT lời agent khai về nhánh, dò được trong output của một bước.
type Khai struct {
	Buoc  string   `json:"buoc"`
	Loai  string   `json:"loai"`
	Cum   string   `json:"cum"`
	Nhanh []string `json:"nhanh"`
	Cau   string   `json:"cau"`
	// Nhom = Nhanh là MỌI nhánh của một tài khoản chứ không phải một nhánh cụ
	// thể. Lời khai theo khuôn `TNS: NEN TRON` nói về NGƯỜI, mà một người có thể
	// có mấy nhánh (workspace.nhanhTrong đẻ hậu tố `-2` khi nhánh cũ còn việc).
	// Nên nó chỉ sai khi CẢ NHÓM rỗng — kết tội vì một nhánh anh em rỗng là vu oan.
	Nhom bool `json:"nhom,omitempty"`
}

// MauThuan là một chỗ lời agent chọi với git.
//
// ChuaChacSai = nhánh rỗng NHƯNG lượt chạy có mã commit thật, tức nhánh nhiều
// khả năng đã được trộn vào nhánh nền sau khi chạy. Không kết tội, chỉ nêu ra.
type MauThuan struct {
	ChuaChacSai bool   `json:"chuaChacSai,omitempty"`
	Khai        Khai   `json:"khai"`
	Git         string `json:"git"`
	Noi         string `json:"noi"`
}

// ---------------------------------------------------------------- bản tóm tắt

// BuocTomTat là một bước đã được xếp vào đúng ngăn (làm / chưa làm / hỏng / treo).
type BuocTomTat struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Profile string `json:"profile"`
	State   string `json:"state"`
	LyDo    string `json:"lyDo,omitempty"`
}

// TomTat là toàn bộ câu trả lời của action "flow.tom-tat".
type TomTat struct {
	// CommitThat là các mã commit agent khai trong lượt VÀ git xác nhận có thật.
	// Có nó nghĩa là lượt này đã đẻ ra việc thật, nên nhánh rỗng nhiều khả năng
	// là đã trộn chứ không phải chưa làm gì.
	CommitThat []string `json:"commitThat,omitempty"`
	RunID      int64    `json:"id"`
	Flow       string   `json:"flow"`
	State      string   `json:"state"`
	Dir        string   `json:"dir"`
	Started    int64    `json:"started"`
	Goc        string   `json:"goc"`

	DaLam   []BuocTomTat `json:"daLam"`
	ChuaLam []BuocTomTat `json:"chuaLam"`
	Hong    []BuocTomTat `json:"hong"`
	Treo    []BuocTomTat `json:"treo"`

	Nhanh    []NhanhChung `json:"nhanh"`
	Khai     []Khai       `json:"khai"`
	MauThuan []MauThuan   `json:"mauThuan"`
	// ChuaDoiChieu là nhánh agent có nhắc mà git ở đây KHÔNG biết. Không kết tội:
	// nói rõ chỗ mình không kiểm được còn hơn im lặng cho qua.
	ChuaDoiChieu []string `json:"chuaDoiChieu"`

	VanBan string `json:"vanBan"`
}

// FlowTomTat — action "flow.tom-tat". Đọc một lượt chạy rồi đối chiếu với git.
func (a *API) FlowTomTat(runID int64) (TomTat, error) {
	run, steps, def, err := a.FlowRunDetail(runID)
	if err != nil {
		return TomTat{}, err
	}
	goc, vao := docNhanhSagent(run.Dir)
	return lamTomTat(run, steps, def, goc, vao), nil
}

// lamTomTat là TOÀN BỘ phần luật, và nó KHÔNG chạm git lẫn sổ trạng thái. Tách
// ra vì đúng chỗ này mới là chỗ dễ sai (ba cái bẫy ở đầu file), mà bắt test
// dựng repo thật thì không ai kiểm nổi ba luật đó cho tử tế.
func lamTomTat(run store.Run, steps map[string]store.StepRun, def flow.Flow,
	goc string, vao []NhanhVao) TomTat {

	t := TomTat{
		RunID: run.ID, Flow: run.Flow, State: run.State, Dir: run.Dir,
		Started: run.Started.Unix(), Goc: goc,
		DaLam: []BuocTomTat{}, ChuaLam: []BuocTomTat{}, Hong: []BuocTomTat{},
		Treo: []BuocTomTat{}, Nhanh: []NhanhChung{}, Khai: []Khai{},
		MauThuan: []MauThuan{}, ChuaDoiChieu: []string{},
	}

	bang := map[string]NhanhChung{}
	var ten []string
	for _, v := range vao {
		if strings.TrimSpace(v.BC.Nhanh) == "" {
			continue
		}
		n := lamNhanhChung(v)
		if _, trung := bang[n.Nhanh]; trung {
			continue
		}
		bang[n.Nhanh] = n
		ten = append(ten, n.Nhanh)
		t.Nhanh = append(t.Nhanh, n)
	}
	// Gom nhánh theo TÀI KHOẢN: khuôn khai của flow (`TNS: NEN TRON`) gọi người
	// chứ không gọi nhánh, mà một người có thể có mấy nhánh.
	nhomTK := map[string][]string{}
	for _, n := range ten {
		if tk := taiKhoanCua(n); tk != "" {
			nhomTK[tk] = append(nhomTK[tk], n)
		}
	}

	// Đi theo THỨ TỰ ĐỊNH NGHĨA để đọc được mạch trên xuống dưới, rồi mới vét
	// bước có trong sổ mà flows.toml đã bỏ — giấu chúng đi thì bản tóm tắt kể
	// thiếu việc đã thật sự xảy ra.
	daCo := map[string]bool{}
	nhacToi := map[string]bool{}
	t.CommitThat = timCommitCoThat(run.Dir, steps)
	xep := func(id, typ, prof string, st store.StepRun) {
		daCo[id] = true
		b := BuocTomTat{ID: id, Type: typ, Profile: prof, State: st.State}
		switch st.State {
		case store.StepFailed:
			b.LyDo = lyDoHong(st)
			t.Hong = append(t.Hong, b)
		case store.StepRunning, store.StepWaiting:
			b.LyDo = lyDoTreo(st)
			t.Treo = append(t.Treo, b)
		case store.StepSkipped:
			b.LyDo = "bị bỏ qua (điều kiện `when` không thoả)"
			t.ChuaLam = append(t.ChuaLam, b)
		case store.StepDone:
			if strings.TrimSpace(st.Output) == "" {
				b.LyDo = "xong nhưng KHÔNG in ra gì — không có bằng chứng nào là đã làm"
			} else {
				b.LyDo = gonCau(motDongCau(st.Output))
			}
			t.DaLam = append(t.DaLam, b)
		default: // "" hoặc pending
			if b.State == "" {
				b.State = store.StepPending
			}
			b.LyDo = "chưa chạy"
			t.ChuaLam = append(t.ChuaLam, b)
		}
		khai := append(doTruong(id, st.Output, nhomTK), doKhai(id, st.Output, ten)...)
		for _, k := range khai {
			t.Khai = append(t.Khai, k)
			t.MauThuan = append(t.MauThuan, doiChieu(k, bang, len(t.CommitThat) > 0)...)
		}
		for _, n := range timTokenNhanh(st.Output) {
			if _, biet := bang[n]; !biet {
				nhacToi[n] = true
			}
		}
	}
	for i := range def.Steps {
		d := def.Steps[i]
		xep(d.ID, d.Type, d.Profile, steps[d.ID])
	}
	var thua []string
	for id := range steps {
		if !daCo[id] {
			thua = append(thua, id)
		}
	}
	sort.Strings(thua)
	for _, id := range thua {
		xep(id, "", "", steps[id])
	}

	for n := range nhacToi {
		t.ChuaDoiChieu = append(t.ChuaDoiChieu, n)
	}
	sort.Strings(t.ChuaDoiChieu)

	t.VanBan = vietTomTat(t)
	return t
}

// doiChieu là LUẬT "lệch thì tin git": lời khai nào nói một nhánh có việc (trộn
// được / giẫm chân người khác / có commit) mà git đếm ra ĐÚNG 0 commit thì đó
// là mâu thuẫn, và bên sai là lời agent.
//
// Nhánh KhongRo KHÔNG bị tính là mâu thuẫn: "không đọc được" không phải bằng
// chứng của bất cứ điều gì.
func doiChieu(k Khai, bang map[string]NhanhChung, daTron bool) []MauThuan {
	if k.Nhom {
		return doiChieuNhom(k, bang, daTron)
	}
	var ra []MauThuan
	for _, ten := range k.Nhanh {
		n, ok := bang[ten]
		if !ok || n.KhongRo || !n.Rong {
			continue
		}
		mt := MauThuan{
			Khai: k, Git: n.MotDong,
			Noi: fmt.Sprintf("lời agent mâu thuẫn với git: bước %q khai %s cho %s (%q), "+
				"nhưng git nói %s — TIN GIT.",
				k.Buoc, tenLoaiKhai[k.Loai], ten, k.Cau, n.MotDong),
		}
		if daTron {
			mt.ChuaChacSai = true
			mt.Noi = fmt.Sprintf("chưa kết luận được: bước %q khai %s cho %s (%q), "+
				"git nói %s — NHƯNG lượt này có commit thật, nhánh nhiều khả năng đã "+
				"được trộn vào nhánh nền sau khi chạy.",
				k.Buoc, tenLoaiKhai[k.Loai], ten, k.Cau, n.MotDong)
		}
		ra = append(ra, mt)
	}
	return ra
}

// doiChieuNhom soi lời khai nói về MỘT NGƯỜI: chỉ kết luận sai khi mọi nhánh của
// người đó đều rỗng. Còn một nhánh có việc thì lời khai vẫn đứng được.
func doiChieuNhom(k Khai, bang map[string]NhanhChung, daTron bool) []MauThuan {
	var co []string
	for _, ten := range k.Nhanh {
		n, ok := bang[ten]
		if !ok || n.KhongRo || !n.Rong {
			return nil // thiếu bằng chứng, hoặc có nhánh làm được việc thật
		}
		co = append(co, n.MotDong)
	}
	if len(co) == 0 {
		return nil
	}
	mt := MauThuan{
		Khai: k, Git: strings.Join(co, "; "),
		Noi: fmt.Sprintf("lời agent mâu thuẫn với git: bước %q khai %s cho %s (%q), "+
			"nhưng git nói %s — TIN GIT.",
			k.Buoc, tenLoaiKhai[k.Loai], strings.Join(k.Nhanh, ", "), k.Cau, strings.Join(co, "; ")),
	}
	if daTron {
		mt.ChuaChacSai = true
		mt.Noi = fmt.Sprintf("chưa kết luận được: bước %q khai %s cho %s (%q), "+
			"git nói %s — NHƯNG lượt này có commit thật, nhánh nhiều khả năng đã "+
			"được trộn vào nhánh nền sau khi chạy.",
			k.Buoc, tenLoaiKhai[k.Loai], strings.Join(k.Nhanh, ", "), k.Cau, strings.Join(co, "; "))
	}
	return []MauThuan{mt}
}

// doTruong dò lời khai theo KHUÔN DÒNG mà flow của dự án bắt agent trả về:
//
//	TNS: NEN TRON / CAN SUA THEM / VUT — lý do một câu
//	MAY: NEN TRON / CAN SUA THEM / VUT — lý do một câu
//	GIAM CHAN: <hai nhánh có sửa cùng file không>
//
// Đây mới là "trường lời agent khai" thật sự, và cũng là chỗ đã sai: lượt #29 và
// #31 phán NEN TRON cho nhánh 0 commit, lượt #34 phán GIAM CHAN cho hai nhánh có
// giao tập file rỗng. Bộ dò văn xuôi ở doKhai KHÔNG bắt được chúng, vì mấy dòng
// này gọi người bằng tên tài khoản (`TNS`) chứ không viết tên nhánh.
func doTruong(buoc, out string, nhomTK map[string][]string) []Khai {
	if strings.TrimSpace(out) == "" || len(nhomTK) == 0 {
		return nil
	}
	var ra []Khai
	var trongBai []string // tài khoản có dòng khai riêng trong chính bài này
	var giamChan string   // dòng GIAM CHAN khẳng định, xử sau vì cần trongBai
	for _, dong := range strings.Split(out, "\n") {
		nhan, gt, ok := strings.Cut(strings.Trim(dong, "*#`|-> \t"), ":")
		if !ok {
			continue
		}
		nhan = chuanNhan(nhan)
		gt = strings.TrimSpace(gt)
		if gt == "" {
			continue
		}
		if laNhanGiamChan(nhan) {
			if giamChan == "" && khangDinh(gt) {
				giamChan = gonCau(strings.TrimSpace(dong))
			}
			continue
		}
		nhanh, biet := nhomTK[nhan]
		if !biet {
			continue
		}
		loai := loaiTuGiaTri(gt)
		if loai == "" {
			continue // VUT, hoặc câu phủ định, hoặc không nhận ra: không phải lời khai
		}
		trongBai = them(trongBai, nhan)
		ra = append(ra, Khai{Buoc: buoc, Loai: loai, Cum: "khuôn `" + strings.ToUpper(nhan) + ":`",
			Nhanh: nhanh, Cau: gonCau(strings.TrimSpace(dong)), Nhom: true})
	}
	// GIAM CHAN không nói tên ai — nó nói về đúng những người vừa được chấm ở
	// trên. Không có ai thì im: đoán xem nó nói về nhánh nào chính là kiểu suy
	// diễn bản tóm tắt này sinh ra để chống.
	if giamChan != "" {
		for _, tk := range trongBai {
			ra = append(ra, Khai{Buoc: buoc, Loai: KhaiGiamChan, Cum: "khuôn `GIAM CHAN:`",
				Nhanh: nhomTK[tk], Cau: giamChan, Nhom: true})
		}
	}
	return ra
}

// chuanNhan gọt nhãn của một dòng khai về dạng so được: bỏ markdown, hạ chữ, gom
// khoảng trắng. Agent bọc `**TNS**:` là chuyện thường.
func chuanNhan(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '*' || r == '`' || r == '_' {
			return -1
		}
		return r
	}, s)
	return strings.Join(strings.Fields(haThap(s)), " ")
}

func laNhanGiamChan(nhan string) bool {
	switch nhan {
	case "giam chan", "giẫm chân", "dẫm chân", "giam-chan", "giam chan nhau":
		return true
	}
	return false
}

// loaiTuGiaTri đọc phán quyết của một dòng khai. VUT là câu trả lời ĐÚNG cho
// nhánh rỗng nên nó KHÔNG phải lời khai — trả về "".
func loaiTuGiaTri(gt string) string {
	g := haThap(gt)
	if coPhuDinh(g) {
		return ""
	}
	if coTu(g, "vut", "") || coTu(g, "vứt", "") || coTu(g, "bỏ", "") {
		return ""
	}
	switch {
	case strings.Contains(g, "nen tron") || strings.Contains(g, "nên trộn") ||
		strings.Contains(g, "nen gop") || strings.Contains(g, "nên gộp"):
		return KhaiTron
	case strings.Contains(g, "can sua them") || strings.Contains(g, "cần sửa thêm"):
		return KhaiCoViec
	}
	return ""
}

// khangDinh đọc một giá trị kiểu có/không.
func khangDinh(gt string) bool {
	g := haThap(gt)
	if coPhuDinh(g) {
		return false
	}
	for _, t := range []string{"có", "co", "yes", "trùng", "trung", "cùng file", "cung file"} {
		if coTu(g, t, "") {
			return true
		}
	}
	return false
}

func them(ds []string, s string) []string {
	for _, x := range ds {
		if x == s {
			return ds
		}
	}
	return append(ds, s)
}

// taiKhoanCua rút tên tài khoản từ tên nhánh: `sagent/may-1-2` -> `may`. Khuôn
// tên do workspace.Add đặt (`sagent/<tài khoản>-<số>`).
func taiKhoanCua(nhanh string) string {
	ten := strings.TrimPrefix(nhanh, "sagent/")
	if ten == nhanh {
		return ""
	}
	if i := strings.IndexByte(ten, '-'); i > 0 {
		ten = ten[:i]
	}
	return haThap(ten)
}

// doKhai dò lời khai trong output của một bước.
//
// Chỉ nhận lời khai NÊU ĐÍCH DANH một nhánh mà git biết: câu chung chung
// ("mọi thứ đều ổn") không đối chiếu được, mà đoán xem nó nói về nhánh nào thì
// lại là kiểu suy diễn bản tóm tắt này sinh ra để chống.
func doKhai(buoc, out string, ten []string) []Khai {
	if strings.TrimSpace(out) == "" || len(ten) == 0 {
		return nil
	}
	l := haThap(out) // giữ nguyên vị trí byte, xem haThap
	var ra []Khai
	daCo := map[string]bool{}
	for _, ck := range cumKhai {
		for i := 0; i < len(l); {
			j := strings.Index(l[i:], ck.Cum)
			if j < 0 {
				break
			}
			j += i
			i = j + len(ck.Cum)

			// BẪY 1 — PHỦ ĐỊNH. Nhìn lại trong CÙNG MỆNH ĐỀ, tính từ dấu ngắt
			// gần nhất: "sagent/x: KHÔNG nên trộn" phải im, không được kể lại
			// thành "x nên trộn".
			m0, _ := doanQuanh(l, j, nganMenh)
			if coPhuDinh(l[m0:j]) {
				continue
			}

			c0, c1 := doanQuanh(l, j, nganCau)
			var nh []string
			for _, tn := range ten {
				// BẪY 2 — BIÊN TỪ. Xem coNhanh.
				if coNhanh(l[c0:c1], haThap(tn)) || coNhanh(l[c0:c1], haThap(rutGonNhanh(tn))) {
					nh = append(nh, tn)
				}
			}
			if len(nh) == 0 {
				continue
			}
			k := Khai{Buoc: buoc, Loai: ck.Loai, Cum: ck.Cum, Nhanh: nh,
				Cau: gonCau(strings.TrimSpace(out[c0:c1]))}
			khoa := k.Loai + "\x00" + strings.Join(nh, ",") + "\x00" + k.Cau
			if daCo[khoa] {
				continue
			}
			daCo[khoa] = true
			ra = append(ra, k)
		}
	}
	return ra
}

// --------------------------------------------------------------- chữ nghĩa

const (
	nganCau  = "\n.!?;"   // ranh giới CÂU — phạm vi tìm tên nhánh
	nganMenh = "\n.!?;:," // ranh giới MỆNH ĐỀ — phạm vi tìm từ phủ định
)

// doanQuanh trả về đoạn [đầu, cuối) chứa vị trí i, cắt bởi các dấu trong ngan.
func doanQuanh(s string, i int, ngan string) (int, int) {
	dau := strings.LastIndexAny(s[:i], ngan) + 1
	cuoi := strings.IndexAny(s[i:], ngan)
	if cuoi < 0 {
		return dau, len(s)
	}
	return dau, i + cuoi
}

func coPhuDinh(menh string) bool {
	for _, tu := range tuPhuDinh {
		if coTu(menh, tu, "") {
			return true
		}
	}
	return false
}

// rutGonNhanh trả về tên nhánh bỏ tiền tố `sagent/`, hoặc "" nếu phần còn lại
// quá ngắn để nhận dạng chắc chắn.
//
// Có vì agent gần như không bao giờ gõ đủ `sagent/tns-1`: lượt #38 bước `soi`
// viết "TNS: NEN TRON", lượt #34 viết "may-1 giẫm chân tns-1". Bỏ qua dạng rút
// gọn là bộ đối chiếu im lặng ở đúng những lượt nó sinh ra để soi.
//
// Chỉ nhận phần đuôi CÓ DẤU GẠCH NGANG (`tns-1`, `may-1-2`) — đó là khuôn tên
// worktree do workspace.Add đẻ ra. Không nhận tên tài khoản trần ("tns"): chữ
// đó nằm khắp nơi trong báo cáo, khớp nó là vu oan hàng loạt.
func rutGonNhanh(nhanh string) string {
	ten := strings.TrimPrefix(nhanh, "sagent/")
	if ten == nhanh || !strings.Contains(ten, "-") || len([]rune(ten)) < 4 {
		return ""
	}
	return ten
}

// coNhanh khớp tên nhánh CÓ BIÊN TỪ, và biên ở đây phải chặt hơn biên chữ
// thường: `-`, `_`, `/`, `.` đều là ký tự HỢP LỆ trong tên nhánh, nên
// `sagent/may-1` KHÔNG được coi là khớp bên trong `sagent/may-1-2`.
func coNhanh(s, nhanh string) bool { return coTu(s, nhanh, "-_/.") }

// coTu tìm tu trong s với biên hai đầu. them là các ký tự KHÔNG phải chữ số
// nhưng vẫn tính là "dính liền" (dùng cho tên nhánh).
func coTu(s, tu, them string) bool {
	if tu == "" {
		return false
	}
	for i := 0; i+len(tu) <= len(s); {
		j := strings.Index(s[i:], tu)
		if j < 0 {
			return false
		}
		j += i
		if bienTrai(s, j, them) && bienPhai(s, j+len(tu), them) {
			return true
		}
		i = j + 1
	}
	return false
}

func bienTrai(s string, i int, them string) bool {
	if i == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return roiNhau(r, them)
}

func bienPhai(s string, i int, them string) bool {
	if i >= len(s) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return roiNhau(r, them)
}

func roiNhau(r rune, them string) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return false
	}
	return !strings.ContainsRune(them, r)
}

// haThap hạ chữ hoa mà GIỮ NGUYÊN vị trí byte của mọi ký tự.
//
// strings.ToLower có thể đổi độ dài (một số ký tự Unicode hạ ra nhiều byte hơn),
// và bộ dò này dùng chỉ số của bản đã hạ để cắt câu trên bản GỐC. Lệch một byte
// là cắt giữa ký tự tiếng Việt, ra chữ rác ngay trong bản báo cáo. Ký tự nào hạ
// xuống mà đổi độ dài thì giữ nguyên — thà bỏ sót còn hơn lệch chỉ số.
func haThap(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		h := unicode.ToLower(r)
		if utf8.RuneLen(h) != utf8.RuneLen(r) {
			h = r
		}
		b.WriteRune(h)
	}
	return b.String()
}

// timTokenNhanh nhặt mọi tên nhánh dạng `sagent/...` xuất hiện trong văn bản.
func timTokenNhanh(s string) []string {
	const dau = "sagent/"
	var ra []string
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], dau)
		if j < 0 {
			break
		}
		j += i
		i = j + len(dau)
		if !bienTrai(s, j, "-_/.") {
			continue
		}
		k := i
		for k < len(s) {
			r, n := utf8.DecodeRuneInString(s[k:])
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("-_.", r) {
				break
			}
			k += n
		}
		ten := strings.TrimRight(s[j:k], ".-_")
		if ten != dau[:len(dau)-1] && len(ten) > len(dau) {
			ra = append(ra, ten)
		}
		i = k
	}
	return ra
}

func motDongCau(s string) string {
	for _, d := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(d); t != "" {
			return t
		}
	}
	return ""
}

// gonCau cắt theo RUNE chứ không theo byte — cắt giữa ký tự tiếng Việt là ra
// dấu hỏi đen ngay trong bản báo cáo.
func gonCau(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= 160 {
		return s
	}
	return string(r[:159]) + "…"
}

func lyDoHong(st store.StepRun) string {
	if t := strings.TrimSpace(st.Msg); t != "" {
		return gonCau(motDongCau(t))
	}
	if ly := khongCoKetQua(st.Output); ly != "" {
		return gonCau(ly)
	}
	if t := strings.TrimSpace(st.Output); t != "" {
		return gonCau(motDongCau(t))
	}
	return "không rõ lý do — bước không để lại lời nhắn lẫn output"
}

func lyDoTreo(st store.StepRun) string {
	if st.State == store.StepWaiting {
		return "đang CHỜ NGƯỜI DUYỆT — lượt chạy đứng yên tới khi có người quyết"
	}
	return "còn ĐANG CHẠY (hoặc lượt chạy bị cắt ngang mà sổ chưa kịp ghi)"
}

// --------------------------------------------------------------- in ra chữ

func vietTomTat(t TomTat) string {
	var b strings.Builder
	fmt.Fprintf(&b, "TÓM TẮT lượt chạy #%d — %s (%s)\n", t.RunID, t.Flow, t.State)
	fmt.Fprintf(&b, "Thư mục: %s · nhánh nền: %s\n", t.Dir, t.Goc)

	nhom := func(tieuDe string, bs []BuocTomTat, trong string) {
		b.WriteString("\n" + tieuDe + "\n")
		if len(bs) == 0 {
			b.WriteString("  (" + trong + ")\n")
			return
		}
		for _, x := range bs {
			dong := "  · " + x.ID
			if x.Profile != "" {
				dong += " [" + x.Profile + "]"
			}
			if x.Type != "" {
				dong += " (" + x.Type + ")"
			}
			dong += " — " + x.State
			b.WriteString(dong + "\n")
			if x.LyDo != "" {
				b.WriteString("      " + x.LyDo + "\n")
			}
		}
	}
	nhom("AI LÀM GÌ", t.DaLam, "không bước nào chạy xong")
	nhom("AI CHƯA LÀM", t.ChuaLam, "không bước nào bị bỏ lại")
	nhom("BƯỚC NÀO HỎNG, VÌ SAO", t.Hong, "không bước nào hỏng")
	nhom("VIỆC CÒN TREO", t.Treo, "không có việc treo")

	// Nói rõ bằng chứng đọc LÚC NÀO: nhánh có thể đã bị trộn hoặc đặt lại kể từ
	// lúc lượt chạy kết thúc, và một con số không ghi mốc thời gian thì người đọc
	// mặc định coi là số của lúc chạy.
	b.WriteString("\nBẰNG CHỨNG GIT (máy tự đếm NGAY LÚC NÀY, không hỏi agent)\n")
	if len(t.Nhanh) == 0 {
		b.WriteString("  (không đọc được nhánh sagent/* nào từ thư mục này)\n")
	}
	for _, n := range t.Nhanh {
		dong := "  · " + n.MotDong
		// BẪY 3: nhánh rỗng KHÔNG được kèm tiêu đề commit nào.
		if n.Dinh != "" {
			dong += " — đỉnh: " + n.Dinh
		}
		b.WriteString(dong + "\n")
	}

	b.WriteString("\nĐỐI CHIẾU LỜI AGENT VỚI GIT\n")
	switch {
	case len(t.MauThuan) > 0:
		// Tách hai loại: kết tội thật, và chỗ chưa kết luận được. Trộn chung thì
		// một lượt đã trộn xong hiện toàn dấu ✗ — người đọc mất niềm tin vào bộ
		// dò rồi tắt nó đi, và mất luôn cả những lần nó bắt đúng.
		for _, m := range t.MauThuan {
			if !m.ChuaChacSai {
				b.WriteString("  ✗ " + m.Noi + "\n")
			}
		}
		for _, m := range t.MauThuan {
			if m.ChuaChacSai {
				b.WriteString("  ? " + m.Noi + "\n")
			}
		}
		if len(t.CommitThat) > 0 {
			fmt.Fprintf(&b, "    (git xác nhận %d mã commit agent khai là CÓ THẬT: %s)\n",
				len(t.CommitThat), strings.Join(t.CommitThat, ", "))
		}
	case len(t.Khai) > 0:
		fmt.Fprintf(&b, "  ✓ %d lời khai về nhánh, không lời nào chọi với git\n", len(t.Khai))
	default:
		b.WriteString("  (không bước nào khai gì về nhánh để mà đối chiếu)\n")
	}
	for _, n := range t.ChuaDoiChieu {
		b.WriteString("  ? agent nhắc " + n + " nhưng git ở đây không biết nhánh đó — CHƯA đối chiếu được\n")
	}
	return b.String()
}

// -------------------------------------------------------------- đọc git thật

// docNhanhSagent đọc bằng chứng của mọi nhánh `sagent/*` quanh thư mục dir.
//
// Phần đếm commit KHÔNG viết lại: worktree còn sống thì hỏi workspace.Xem (nó
// đã đếm `goc..HEAD` đúng cách và đã biết nói "KHÔNG có commit nào").
func docNhanhSagent(dir string) (string, []NhanhVao) {
	root, ok := workspace.RepoRoot(dir)
	if !ok {
		wd, err := os.Getwd()
		if err != nil {
			return "main", nil
		}
		if root, ok = workspace.RepoRoot(wd); !ok {
			return "main", nil
		}
	}
	goc, err := workspace.NhanhMacDinh(root)
	if err != nil || goc == "" {
		goc = "main"
	}

	daCo := map[string]bool{}
	var ra []NhanhVao
	for _, d := range thuMucWorktree(root) {
		bc := workspace.Xem(d, goc)
		if bc.KhongRo || !strings.HasPrefix(bc.Nhanh, "sagent/") || daCo[bc.Nhanh] {
			continue
		}
		daCo[bc.Nhanh] = true
		ra = append(ra, NhanhVao{BC: bc, Dinh: dinhCua(root, bc.Nhanh, bc.Commit)})
	}
	// Nhánh còn sống mà worktree đã bị gỡ (`sagent clean`) vẫn phải có mặt: đó
	// đúng là chỗ việc cũ nằm lại, và cũng đúng là chỗ hay bị khai sai nhất.
	for _, ten := range nhanhSagent(root) {
		if daCo[ten] {
			continue
		}
		daCo[ten] = true
		bc := workspace.BangChung{Nhanh: ten}
		if n, err := demCommit(root, goc, ten); err != nil {
			bc.KhongRo = true
		} else {
			bc.Commit = n
		}
		ra = append(ra, NhanhVao{BC: bc, Dinh: dinhCua(root, ten, bc.Commit)})
	}
	sort.Slice(ra, func(i, j int) bool { return ra[i].BC.Nhanh < ra[j].BC.Nhanh })
	return goc, ra
}

// dinhCua đọc tiêu đề commit đỉnh — và KHÔNG hỏi git khi nhánh rỗng. Đây là
// BẪY 3 chặn ngay từ nguồn: `git log -1` trên nhánh 0 commit vẫn trả về một
// dòng rất thuyết phục, chính là đỉnh của main.
func dinhCua(root, nhanh string, commit int) string {
	if commit <= 0 {
		return ""
	}
	out, err := gitRa(root, "log", "-1", "--format=%s", nhanh)
	if err != nil {
		return ""
	}
	return out
}

func demCommit(root, goc, nhanh string) (int, error) {
	out, err := gitRa(root, "rev-list", "--count", goc+".."+nhanh)
	if err != nil {
		return 0, err
	}
	n := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(out), "%d", &n); err != nil {
		return 0, err
	}
	return n, nil
}

func thuMucWorktree(root string) []string {
	out, err := gitRa(root, "worktree", "list", "--porcelain")
	if err != nil {
		return nil
	}
	var ra []string
	for _, d := range strings.Split(out, "\n") {
		if p, ok := strings.CutPrefix(strings.TrimSpace(d), "worktree "); ok {
			ra = append(ra, strings.TrimSpace(p))
		}
	}
	return ra
}

func nhanhSagent(root string) []string {
	out, err := gitRa(root, "for-each-ref", "--format=%(refname:short)", "refs/heads/sagent")
	if err != nil {
		return nil
	}
	var ra []string
	for _, d := range strings.Split(out, "\n") {
		if t := strings.TrimSpace(d); strings.HasPrefix(t, "sagent/") {
			ra = append(ra, t)
		}
	}
	return ra
}

func gitRa(dir string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.Output()
	return strings.TrimSpace(string(out)), err
}

// ---------------------------------------------------------------------------
// BẪY THỨ TƯ, phát hiện 20/08 khi chạy thật `flow tom-tat 34`.
//
// Bộ đối chiếu so lời agent với git Ở HIỆN TẠI. Nhưng lượt chạy #34 đã được
// trộn vào main xong, nên `sagent/tns-1` giờ đúng 0 commit — trong khi lúc lượt
// đó chạy nó CÓ commit 3ad36a1. Kết quả: bộ dò kết tội cả bốn lời khai đúng.
//
// Vu oan còn tệ hơn bỏ sót: người đọc mất niềm tin vào bộ dò rồi tắt nó đi, và
// thế là mất luôn cả những lần nó bắt đúng.
//
// Phân biệt bằng BẰNG CHỨNG, không bằng suy đoán: agent khai mã commit trong
// output ("COMMIT: 3ad36a1"). Mã đó còn tồn tại trong repo nghĩa là commit CÓ
// THẬT — nhánh rỗng chỉ vì đã trộn. Không tồn tại mới là khai khống.

var reCommit = regexp.MustCompile(`\b[0-9a-f]{7,40}\b`)

// timCommitCoThat quét mọi output của lượt chạy, lấy các chuỗi trông như mã
// commit rồi HỎI GIT xem có thật không. Trả về danh sách mã có thật.
//
// Hỏi git chứ không tin hình dạng: "deadbeef" cũng khớp mẫu hex.
func timCommitCoThat(dir string, steps map[string]store.StepRun) []string {
	thay := map[string]bool{}
	var ra []string
	for _, st := range steps {
		for _, m := range reCommit.FindAllString(st.Output, -1) {
			if thay[m] {
				continue
			}
			thay[m] = true
			if workspace.CommitCoThat(dir, m) {
				ra = append(ra, m)
			}
		}
	}
	sort.Strings(ra)
	return ra
}
