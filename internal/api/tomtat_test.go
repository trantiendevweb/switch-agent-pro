package api

import (
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
	"github.com/trantiendevweb/switch-agent-pro/internal/workspace"
)

// nhanh dựng bằng chứng git của một nhánh mà KHÔNG cần repo thật — phần luật của
// bản tóm tắt cố ý không chạm git, đúng để test được ba cái bẫy ở đầu tomtat.go.
func nhanh(ten string, commit int) NhanhVao {
	return NhanhVao{BC: workspace.BangChung{Nhanh: ten, Commit: commit}}
}

// luot dựng một lượt chạy một bước agent với output cho trước.
func luot(out string) (store.Run, map[string]store.StepRun, flow.Flow) {
	run := store.Run{ID: 42, Flow: "soi", Dir: `C:\du-an`, State: store.RunDone,
		Started: time.Unix(1_700_000_000, 0)}
	def := flow.Flow{Name: "soi", Steps: []flow.Step{
		{ID: "soi-nhanh", Type: flow.TypeAgent, Profile: "claude:tns"},
	}}
	steps := map[string]store.StepRun{
		"soi-nhanh": {RunID: 42, StepID: "soi-nhanh", State: store.StepDone, Output: out},
	}
	return run, steps, def
}

func tomTatCua(out string, vao ...NhanhVao) TomTat {
	run, steps, def := luot(out)
	return lamTomTat(run, steps, def, "main", vao)
}

// CA CHÍNH: agent nói nhánh CÓ VIỆC, git nói 0 commit.
//
// Đây đúng là lượt #21: bước được đánh `done`, cả flow `completed`, agent kể là
// đã làm — mà `sagent/may-1` không có commit nào. Bản tóm tắt phải chỉ thẳng ra
// chỗ lệch chứ không chép lại lời agent.
func TestTomTatChiRaAgentNoiCoViecMaGitNoiKhong(t *testing.T) {
	tt := tomTatCua(
		"Đã rà soát xong. Nhánh sagent/may-1 có commit đầy đủ, sẵn sàng trộn.",
		nhanh("sagent/may-1", 0))

	if len(tt.MauThuan) == 0 {
		t.Fatalf("agent khai nhánh có việc mà git đếm 0 commit — bản tóm tắt KHÔNG chỉ ra mâu thuẫn.\n"+
			"lời khai dò được: %+v\nbản in:\n%s", tt.Khai, tt.VanBan)
	}
	if !strings.Contains(tt.VanBan, "lời agent mâu thuẫn với git") {
		t.Errorf("bản in không nói thẳng câu \"lời agent mâu thuẫn với git\":\n%s", tt.VanBan)
	}
	if !strings.Contains(tt.VanBan, "TIN GIT") {
		t.Errorf("bản in không nói rõ tin bên nào:\n%s", tt.VanBan)
	}
	// Phải kèm đúng con số git đo được, không phải lời kể.
	if !strings.Contains(tt.VanBan, "KHÔNG có commit nào") {
		t.Errorf("bản in thiếu bằng chứng git đi kèm:\n%s", tt.VanBan)
	}
	// Cả hai loại lời khai trong câu đó (có commit + sẵn sàng trộn) đều phải bị soi.
	loai := map[string]bool{}
	for _, m := range tt.MauThuan {
		loai[m.Khai.Loai] = true
	}
	if !loai[KhaiCoViec] || !loai[KhaiTron] {
		t.Errorf("mới đối chiếu %v — phải soi MỌI trường agent khai, không chọn một cái", loai)
	}
}

// LUẬT 1 — PHỦ ĐỊNH. "KHÔNG nên trộn" không được tính là khai "nên trộn".
//
// Gỡ nhánh phủ định trong doKhai ra là test này đỏ: cụm "nên trộn" vẫn nằm
// nguyên trong câu, nên bộ dò sẽ báo mâu thuẫn cho một nhánh mà agent vừa nói
// đúng y hệt git.
func TestPhuDinhKhongBiTinhLaLoiKhai(t *testing.T) {
	for _, cau := range []string{
		"sagent/phu-1: KHÔNG có commit nào, KHÔNG nên trộn.",
		"Nhánh sagent/phu-1 chưa có việc gì nên đừng trộn vội.",
		"nhánh sagent/phu-1: KHÔNG có commit nào",
	} {
		tt := tomTatCua(cau, nhanh("sagent/phu-1", 0))
		if len(tt.MauThuan) > 0 {
			t.Errorf("câu %q là lời PHỦ ĐỊNH và trùng khớp với git, "+
				"nhưng bản tóm tắt vẫn kết tội agent:\n  %s", cau, tt.MauThuan[0].Noi)
		}
	}
	// Lá chắn ngược: bỏ chữ phủ định đi thì đúng câu đó PHẢI bị bắt — nếu không,
	// test trên xanh chỉ vì bộ dò mù chứ không phải vì nó hiểu phủ định.
	tt := tomTatCua("sagent/phu-1: nên trộn.", nhanh("sagent/phu-1", 0))
	if len(tt.MauThuan) == 0 {
		t.Fatal("bỏ chữ KHÔNG đi mà bộ dò vẫn im — nó không hề dò được gì")
	}
}

// LUẬT 2 — BIÊN TỪ. `sagent/may-1` là chuỗi con của `sagent/may-1-2`.
//
// Đây là tình huống có thật của dự án: nhanhTrong() trong internal/workspace đẻ
// ra hậu tố `-2` khi nhánh cũ còn việc chưa trộn, nên hai tên chỉ khác nhau ở
// đuôi là chuyện thường ngày. Khớp chuỗi lỏng sẽ vu cho agent một câu về nhánh
// nó không hề nhắc tới.
func TestKhopTenNhanhPhaiCoBienTu(t *testing.T) {
	tt := tomTatCua(
		"Nhánh sagent/may-1-2 có commit và nên trộn.",
		nhanh("sagent/may-1", 0),   // rỗng — KHÔNG ai nói về nó
		nhanh("sagent/may-1-2", 5), // có việc thật, agent nói đúng
	)
	for _, m := range tt.MauThuan {
		t.Errorf("khớp chuỗi không biên từ: agent nói về sagent/may-1-2 "+
			"nhưng bản tóm tắt kết tội qua %v:\n  %s", m.Khai.Nhanh, m.Noi)
	}
	// Và lời khai phải được gán ĐÚNG nhánh, không kéo theo nhánh anh em.
	if len(tt.Khai) == 0 {
		t.Fatal("không dò được lời khai nào trong câu có cả \"có commit\" lẫn \"nên trộn\"")
	}
	for _, k := range tt.Khai {
		for _, n := range k.Nhanh {
			if n != "sagent/may-1-2" {
				t.Errorf("lời khai %q bị gán cho %q — câu chỉ nhắc sagent/may-1-2", k.Cum, n)
			}
		}
	}
}

// LUẬT 3 — NHÁNH RỖNG. 0 commit thì nói thẳng là rỗng, và TUYỆT ĐỐI không kèm
// tiêu đề commit nào.
//
// `git log -1 sagent/x` trên nhánh chưa có commit riêng trả về ĐỈNH CỦA MAIN —
// một dòng rất thuyết phục về việc chẳng ai làm. Test đưa vào đúng cái tiêu đề
// bẩn đó để khẳng định bản tóm tắt vứt nó đi chứ không in ra.
func TestNhanhRongNoiThangLaRongKhongKhoeDinhMain(t *testing.T) {
	vao := NhanhVao{
		BC:   workspace.BangChung{Nhanh: "sagent/phu-1", Commit: 0},
		Dinh: "docs: bo sung thiet ke Mat 2D vao THIET-KE.md", // đỉnh của main
	}
	tt := tomTatCua("không có gì để nói", vao)

	if len(tt.Nhanh) != 1 {
		t.Fatalf("muốn 1 nhánh, được %d", len(tt.Nhanh))
	}
	n := tt.Nhanh[0]
	if !n.Rong {
		t.Fatal("nhánh 0 commit mà không được đánh dấu là rỗng")
	}
	if n.Dinh != "" {
		t.Errorf("nhánh rỗng vẫn giữ tiêu đề commit %q — đó là đỉnh của main, không phải việc của nhánh này", n.Dinh)
	}
	if !strings.Contains(n.MotDong, "KHÔNG có commit nào") {
		t.Errorf("nhánh rỗng không được nói thẳng là rỗng: %q", n.MotDong)
	}
	if strings.Contains(tt.VanBan, "THIET-KE.md") || strings.Contains(tt.VanBan, "đỉnh:") {
		t.Errorf("bản in khoe đỉnh commit cho một nhánh rỗng:\n%s", tt.VanBan)
	}
	// Nhánh CÓ việc thì ngược lại: đỉnh phải hiện, không thì bằng chứng cụt.
	co := NhanhVao{BC: workspace.BangChung{Nhanh: "sagent/tns-1", Commit: 3}, Dinh: "sửa lỗi X"}
	tt2 := tomTatCua("xong", co)
	if tt2.Nhanh[0].Dinh != "sửa lỗi X" || !strings.Contains(tt2.VanBan, "đỉnh: sửa lỗi X") {
		t.Errorf("nhánh có 3 commit mà mất tiêu đề commit đỉnh:\n%s", tt2.VanBan)
	}
}

// Bốn ngăn phải xếp đúng: ai làm, ai chưa, hỏng vì sao, việc gì treo.
func TestTomTatXepDungBonNgan(t *testing.T) {
	run := store.Run{ID: 7, Flow: "squad", State: store.RunWaiting, Started: time.Unix(1, 0)}
	def := flow.Flow{Name: "squad", Steps: []flow.Step{
		{ID: "code", Type: flow.TypeAgent, Profile: "claude:tns"},
		{ID: "kiem", Type: flow.TypeShell},
		{ID: "duyet", Type: flow.TypeApprove},
		{ID: "bao", Type: flow.TypeNotify},
		{ID: "don", Type: flow.TypeShell},
	}}
	steps := map[string]store.StepRun{
		"code":  {State: store.StepDone, Output: "đã sửa xong"},
		"kiem":  {State: store.StepFailed, Msg: "go test hỏng: 2 test đỏ"},
		"duyet": {State: store.StepWaiting},
		"bao":   {State: store.StepSkipped},
		// "don" không có dòng nào trong sổ -> chưa chạy
	}
	tt := lamTomTat(run, steps, def, "main", nil)

	if len(tt.DaLam) != 1 || tt.DaLam[0].ID != "code" {
		t.Errorf("AI LÀM GÌ sai: %+v", tt.DaLam)
	}
	if len(tt.Hong) != 1 || !strings.Contains(tt.Hong[0].LyDo, "2 test đỏ") {
		t.Errorf("BƯỚC HỎNG phải kèm LÝ DO đo được: %+v", tt.Hong)
	}
	if len(tt.Treo) != 1 || tt.Treo[0].ID != "duyet" {
		t.Errorf("VIỆC TREO sai: %+v", tt.Treo)
	}
	// bỏ qua + chưa chạy đều là "chưa làm", nhưng lý do phải khác nhau.
	if len(tt.ChuaLam) != 2 {
		t.Fatalf("AI CHƯA LÀM phải có 2 bước (bỏ qua + chưa chạy), được %+v", tt.ChuaLam)
	}
	if tt.ChuaLam[0].LyDo == tt.ChuaLam[1].LyDo {
		t.Errorf("bỏ qua và chưa chạy bị kể chung một lý do: %+v", tt.ChuaLam)
	}
}

// Bước `done` mà không in ra gì phải bị nói thẳng — mã thoát 0 không phải bằng
// chứng đã làm việc (xem khongCoKetQua trong api.go).
func TestBuocXongMaKhongInGiPhaiBiNoiRa(t *testing.T) {
	tt := tomTatCua("")
	if len(tt.DaLam) != 1 {
		t.Fatalf("%+v", tt.DaLam)
	}
	if !strings.Contains(tt.DaLam[0].LyDo, "KHÔNG in ra gì") {
		t.Errorf("bước xong mà rỗng lại được kể như đã làm việc: %q", tt.DaLam[0].LyDo)
	}
}

// Nhánh agent nhắc mà git không biết: KHÔNG kết tội, nhưng phải nói rõ là chưa
// đối chiếu được — im lặng cho qua thì người đọc tưởng đã kiểm hết.
func TestNhanhLaKhongBiKetToiNhungPhaiNoiRoLaChuaKiem(t *testing.T) {
	tt := tomTatCua("Nhánh sagent/la-hoac không nên trộn nhưng sagent/la-hoac có việc.",
		nhanh("sagent/tns-1", 2))
	if len(tt.MauThuan) > 0 {
		t.Errorf("kết tội một nhánh không có bằng chứng git: %s", tt.MauThuan[0].Noi)
	}
	if len(tt.ChuaDoiChieu) != 1 || tt.ChuaDoiChieu[0] != "sagent/la-hoac" {
		t.Fatalf("phải liệt kê nhánh chưa đối chiếu được, được %v", tt.ChuaDoiChieu)
	}
	if !strings.Contains(tt.VanBan, "CHƯA đối chiếu được") {
		t.Errorf("bản in giấu chỗ chưa kiểm được:\n%s", tt.VanBan)
	}
}

// Nhánh KHÔNG ĐỌC ĐƯỢC khác hẳn nhánh rỗng: "không biết" không phải bằng chứng
// của bất cứ điều gì, nên không được dùng nó để kết tội ai.
func TestNhanhKhongDocDuocThiKhongKetToi(t *testing.T) {
	vao := NhanhVao{BC: workspace.BangChung{Nhanh: "sagent/tns-1", KhongRo: true}}
	tt := tomTatCua("sagent/tns-1 có commit, nên trộn.", vao)
	if len(tt.MauThuan) > 0 {
		t.Errorf("dùng \"không đọc được\" làm bằng chứng kết tội: %s", tt.MauThuan[0].Noi)
	}
}

// Lời khai GIẪM CHÂN cũng phải bị soi: một nhánh 0 commit thì không thể giẫm
// lên việc của ai (lượt #34).
func TestKhaiGiamChanCungBiDoiChieu(t *testing.T) {
	tt := tomTatCua("Hai nhánh sagent/phu-1 và sagent/tns-1 giẫm chân nhau ở cùng bộ file.",
		nhanh("sagent/phu-1", 0), nhanh("sagent/tns-1", 4))
	if len(tt.MauThuan) != 1 {
		t.Fatalf("muốn đúng 1 mâu thuẫn (nhánh rỗng), được %d: %+v", len(tt.MauThuan), tt.MauThuan)
	}
	if tt.MauThuan[0].Khai.Loai != KhaiGiamChan {
		t.Errorf("loại lời khai sai: %+v", tt.MauThuan[0].Khai)
	}
	if !strings.Contains(tt.MauThuan[0].Noi, "sagent/phu-1") {
		t.Errorf("kết tội nhầm nhánh: %s", tt.MauThuan[0].Noi)
	}
}

// Agent gần như không gõ đủ tên nhánh. Lượt #38 bước `soi` viết "TNS: NEN TRON",
// lượt #34 viết "may-1 giẫm chân tns-1". Bộ đối chiếu phải nhận DẠNG RÚT GỌN —
// không thì nó im lặng ở đúng những lượt nó sinh ra để soi.
func TestNhanDangTenNhanhRutGon(t *testing.T) {
	tt := tomTatCua("tns-1: NEN TRON — có commit code + test.", nhanh("sagent/tns-1", 0))
	if len(tt.MauThuan) == 0 {
		t.Fatalf("agent viết tắt \"tns-1\" thì bộ đối chiếu mù:\n%s", tt.VanBan)
	}

	// Nhưng KHÔNG được nhận tên tài khoản trần: chữ "tns" nằm khắp mọi báo cáo,
	// khớp nó là vu oan hàng loạt.
	tt2 := tomTatCua("Nhóm tns nên trộn phần tài liệu vào main.", nhanh("sagent/tns-1", 0))
	if len(tt2.MauThuan) > 0 {
		t.Errorf("khớp cả tên tài khoản trần \"tns\": %s", tt2.MauThuan[0].Noi)
	}

	// Và biên từ vẫn phải giữ ở dạng rút gọn: "may-1" không được khớp "may-1-2".
	tt3 := tomTatCua("may-1-2 có commit và nên trộn.",
		nhanh("sagent/may-1", 0), nhanh("sagent/may-1-2", 5))
	if len(tt3.MauThuan) > 0 {
		t.Errorf("dạng rút gọn làm mất biên từ: %s", tt3.MauThuan[0].Noi)
	}
}

// KHUÔN KHAI của flow dự án (.sagent/flows.toml, bước `soi`):
//
//	TNS: NEN TRON / CAN SUA THEM / VUT — lý do một câu
//	GIAM CHAN: <hai nhánh có sửa cùng file không>
//
// Đây mới là "trường lời agent khai", và cũng đúng là chỗ đã sai: lượt #29 và
// #31 phán NEN TRON cho nhánh 0 commit. Mấy dòng này gọi người bằng TÊN TÀI
// KHOẢN chứ không viết tên nhánh, nên bộ dò văn xuôi không thấy chúng.
func TestKhuonKhaiTheoTaiKhoanBiDoiChieu(t *testing.T) {
	out := "TNS: NEN TRON — code đã xong đủ test\nMAY: VUT — chưa làm gì\nGIAM CHAN: có"
	tt := tomTatCua(out, nhanh("sagent/tns-1", 0), nhanh("sagent/may-1", 0))

	loai := map[string]bool{}
	for _, m := range tt.MauThuan {
		loai[m.Khai.Loai] = true
		if len(m.Khai.Nhanh) != 1 || m.Khai.Nhanh[0] != "sagent/tns-1" {
			t.Errorf("kết tội nhầm người: %+v", m.Khai)
		}
	}
	if !loai[KhaiTron] {
		t.Errorf("dòng \"TNS: NEN TRON\" cho nhánh 0 commit KHÔNG bị bắt:\n%s", tt.VanBan)
	}
	// GIAM CHAN nói về đúng những người vừa bị chấm ở trên. MAY bị phán VUT (đúng)
	// nên không có lời khai nào, chỉ còn TNS.
	if !loai[KhaiGiamChan] {
		t.Errorf("dòng \"GIAM CHAN: có\" trên nhánh 0 commit KHÔNG bị bắt:\n%s", tt.VanBan)
	}
}

// VUT là câu trả lời ĐÚNG cho nhánh rỗng — không được tính là lời khai.
// "GIAM CHAN: không" cũng vậy.
func TestKhuonKhaiVutVaKhongThiKhongBiKetToi(t *testing.T) {
	out := "TNS: VUT — nhánh trống, chưa ai làm\nMAY: VUT — chưa làm gì\nGIAM CHAN: không"
	tt := tomTatCua(out, nhanh("sagent/tns-1", 0), nhanh("sagent/may-1", 0))
	if len(tt.MauThuan) > 0 {
		t.Fatalf("agent phán VUT (đúng y git) mà vẫn bị kết tội: %s", tt.MauThuan[0].Noi)
	}
	if len(tt.Khai) > 0 {
		t.Errorf("VUT bị đếm thành lời khai: %+v", tt.Khai)
	}
}

// Một tài khoản có MẤY nhánh (workspace.nhanhTrong đẻ hậu tố `-2`). Lời khai gọi
// tên người, nên chỉ sai khi CẢ NHÓM rỗng — còn một nhánh có việc thì lời khai
// vẫn đứng được, kết tội nó là vu oan.
func TestKhaiTheoNguoiChiSaiKhiCaNhomRong(t *testing.T) {
	con := tomTatCua("MAY: NEN TRON — docs đã xong",
		nhanh("sagent/may-1", 0), nhanh("sagent/may-1-2", 3))
	if len(con.MauThuan) > 0 {
		t.Errorf("nhóm còn nhánh 3 commit mà vẫn bị kết tội: %s", con.MauThuan[0].Noi)
	}
	het := tomTatCua("MAY: NEN TRON — docs đã xong",
		nhanh("sagent/may-1", 0), nhanh("sagent/may-1-2", 0))
	if len(het.MauThuan) != 1 {
		t.Fatalf("cả nhóm rỗng mà không bị bắt: %+v\n%s", het.MauThuan, het.VanBan)
	}
	if len(het.MauThuan[0].Khai.Nhanh) != 2 {
		t.Errorf("bản in phải kể ĐỦ nhánh của người đó: %+v", het.MauThuan[0].Khai)
	}
}

// Bằng chứng git đọc LÚC TÓM TẮT chứ không phải lúc chạy — nhánh có thể đã bị
// trộn hoặc đặt lại từ lúc đó. Con số không ghi mốc thì người đọc mặc định coi
// là số của lúc chạy, và đó là hiểu sai.
func TestBanInNoiRoBangChungDocLucNao(t *testing.T) {
	tt := tomTatCua("xong", nhanh("sagent/tns-1", 2))
	if !strings.Contains(tt.VanBan, "NGAY LÚC NÀY") {
		t.Errorf("bản in không ghi mốc thời gian của bằng chứng git:\n%s", tt.VanBan)
	}
}
