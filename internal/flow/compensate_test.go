package flow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// TestCompensateChayBuocGoLaiRoiDUNG là bài test chính của mảnh này.
//
// Bước `tao` hỏng → bước `go-lai` PHẢI chạy → lượt chạy PHẢI dừng (không đi tiếp
// sang `dung-tiep`). Đây đúng là chỗ `compensate` khác cả `stop` lẫn `continue`:
// `stop` không gỡ gì, `continue` không dừng.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: bỏ nhánh `case OnFailCompensate` trong
// xuLyHong thì nó rơi vào `default` — lượt chạy vẫn dừng, nhưng bước gỡ lại
// KHÔNG chạy, và test đỏ ở dòng kiểm file dấu vết.
func TestCompensateChayBuocGoLaiRoiDUNG(t *testing.T) {
	r, ag, db := newRunner(t)
	dauVet := filepath.Join(t.TempDir(), "da-go-lai.txt")

	f := Flow{Name: "co-undo", Steps: []Step{
		{ID: "tao", Type: TypeShell, Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailCompensate, Compensate: "go-lai"},
		{ID: "go-lai", Type: TypeShell, Run: argvTroGiup(t, "ghi", dauVet, "DA GO")},
		{ID: "dung-tiep", Type: TypeAgent, Needs: []string{"tao"}, Prompt: "dùng thứ vừa tạo"},
	}}
	if ps := Validate(f); coLoi(ps) {
		t.Fatalf("flow phải hợp lệ: %v", ps)
	}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("compensate là GỠ RỒI DỪNG — lượt chạy phải failed, được %q", res.State)
	}

	// 1. Bước gỡ lại đã chạy THẬT (có dấu vết trên đĩa, không chỉ có dòng log).
	if _, err := os.Stat(dauVet); err != nil {
		t.Fatalf("BƯỚC GỠ LẠI KHÔNG CHẠY — không có dấu vết: %v", err)
	}

	// 2. Lượt chạy dừng: bước sau bước hỏng KHÔNG được chạy.
	if ag.soLanGoi() != 0 {
		t.Fatalf("gỡ xong rồi vẫn chạy tiếp trên nền một việc vừa bị gỡ — agent gọi %d lần",
			ag.soLanGoi())
	}

	buoc, _ := db.Steps(res.RunID)
	if buoc["tao"].State != store.StepFailed {
		t.Fatalf("bước hỏng phải vẫn là failed, được %q", buoc["tao"].State)
	}
	if buoc["go-lai"].State != store.StepDone {
		t.Fatalf("bước gỡ lại phải done, được %q", buoc["go-lai"].State)
	}
}

// TestBuocGoLaiKHONGChayONhungLuotBinhThuong canh cái bẫy lớn nhất của thiết kế
// này: bước gỡ lại thường không có `needs` nào, tức là một GỐC của DAG.
//
// Để yên thì nó chạy ngay đợt đầu của MỌI lượt chạy — gỡ một việc chưa ai làm.
// Với một bước gỡ thật (`git branch -D`, `terraform destroy`) thì đó là phá hoại.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: bỏ nhánh `if goLai[s.ID]` trong execute thì
// bước gỡ lại chạy ở đợt đầu và dấu vết xuất hiện dù không có gì hỏng.
func TestBuocGoLaiKHONGChayONhungLuotBinhThuong(t *testing.T) {
	r, _, db := newRunner(t)
	dauVet := filepath.Join(t.TempDir(), "khong-duoc-co.txt")

	f := Flow{Name: "yen-lanh", Steps: []Step{
		{ID: "tao", Type: TypeShell, Run: argvTroGiup(t, "in", "ổn"),
			OnFailure: OnFailCompensate, Compensate: "go-lai"},
		{ID: "go-lai", Type: TypeShell, Run: argvTroGiup(t, "ghi", dauVet, "KHONG NEN CO")},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("không có gì hỏng thì lượt chạy phải xong, được %q", res.State)
	}
	if _, err := os.Stat(dauVet); err == nil {
		t.Fatal("BƯỚC GỠ LẠI ĐÃ CHẠY DÙ KHÔNG CÓ GÌ HỎNG — nó gỡ một việc chưa ai làm")
	}
	// Và nó phải được ghi `skipped` KÈM LỜI GIẢI THÍCH, không để trống: ô trống
	// trên bảng đọc là "chưa tới lượt", còn đây là "sẽ không chạy trừ khi có chuyện".
	buoc, _ := db.Steps(res.RunID)
	if buoc["go-lai"].State != store.StepSkipped {
		t.Fatalf("bước gỡ lại phải là skipped, được %q", buoc["go-lai"].State)
	}
	if !strings.Contains(buoc["go-lai"].Msg, "chỉ chạy khi tao hỏng") {
		t.Fatalf("phải nói rõ vì sao bỏ qua, được %q", buoc["go-lai"].Msg)
	}
}

// TestBuocGoLaiCungHONG — CÂU HỎI KHÓ số 1.
//
// Không thử lại vô hạn, không gỡ-lại-của-gỡ-lại, không đi tiếp. Lượt chạy dừng,
// và thông điệp phải nói rõ đây là trạng thái KHÔNG BIẾT: việc chính không xong
// mà cũng chưa gỡ được. Đó là câu khác hẳn "bước x hỏng" — nó là câu "có người
// phải vào dọn tay".
func TestBuocGoLaiCungHONG(t *testing.T) {
	r, _, db := newRunner(t)

	f := Flow{Name: "go-cung-hong", Steps: []Step{
		{ID: "tao", Type: TypeShell, Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailCompensate, Compensate: "go-lai"},
		{ID: "go-lai", Type: TypeShell, Run: argvTroGiup(t, "hong")},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("phải failed, được %q", res.State)
	}
	buoc, _ := db.Steps(res.RunID)
	if buoc["go-lai"].State != store.StepFailed {
		t.Fatalf("bước gỡ lại phải được ghi là failed, được %q", buoc["go-lai"].State)
	}
	// Bước gỡ lại chạy đúng MỘT lần (retry của chính nó là 0). Không có vòng
	// gỡ-lại-của-gỡ-lại nào.
	if buoc["go-lai"].Attempt != 1 {
		t.Fatalf("bước gỡ lại phải chạy đúng 1 lần, sổ ghi %d", buoc["go-lai"].Attempt)
	}
}

// Thông điệp lỗi cuối cùng phải PHÂN BIỆT ĐƯỢC hai tình huống — người đọc lúc 2
// giờ sáng cần biết hiện trường đã sạch hay chưa, và hai câu đó dẫn tới hai việc
// phải làm khác hẳn nhau.
//
// Đọc từ SỰ KIỆN THẬT mà bộ chạy bắn ra, không phải từ một bản chép tay: câu
// người ta thật sự nhận được là câu đi qua sự kiện.
func TestThongDiepPhanBietGoDuocVoiGoKhongDuoc(t *testing.T) {
	lyDoCuaLuot := func(t *testing.T, goHong bool) string {
		t.Helper()
		r, _, _ := newRunner(t)
		go_ := argvTroGiup(t, "in", "đã gỡ")
		if goHong {
			go_ = argvTroGiup(t, "hong")
		}
		f := Flow{Name: "tin-nhan", Steps: []Step{
			{ID: "tao", Type: TypeShell, Run: argvTroGiup(t, "hong"),
				OnFailure: OnFailCompensate, Compensate: "go-lai"},
			{ID: "go-lai", Type: TypeShell, Run: go_},
		}}

		ch, huy := r.Bus.Subscribe(64)
		var mu sync.Mutex
		var ly string
		xong := make(chan struct{})
		go func() {
			defer close(xong)
			for e := range ch {
				if e.Type == events.FlowFailed {
					mu.Lock()
					ly = e.Detail["ly_do"]
					mu.Unlock()
				}
			}
		}()
		if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
			t.Fatal(err)
		}
		huy()
		<-xong
		mu.Lock()
		defer mu.Unlock()
		return ly
	}

	sach := lyDoCuaLuot(t, false)
	ban := lyDoCuaLuot(t, true)

	if !strings.Contains(sach, "đã chạy bước gỡ lại") {
		t.Fatalf("gỡ được thì phải nói ra: %q", sach)
	}
	if strings.Contains(sach, "xem tay") {
		t.Fatalf("gỡ được rồi mà vẫn gọi người vào dọn: %q", sach)
	}
	if !strings.Contains(ban, "CŨNG HỎNG") || !strings.Contains(ban, "cần người vào xem tay") {
		t.Fatalf("gỡ không được thì phải nói đây là trạng thái KHÔNG BIẾT: %q", ban)
	}
	if sach == ban {
		t.Fatal("hai tình huống ra cùng một câu — người đọc không phân biệt được")
	}
}

// Bước gỡ lại phải biết mình đang gỡ CÁI GÌ — không thì một bước gỡ dùng chung
// cho ba bước không có cách nào phân biệt.
func TestBuocGoLaiBietMinhDangGoChoBuocNao(t *testing.T) {
	r, ag, _ := newRunner(t)

	f := Flow{Name: "biet-go-gi", Steps: []Step{
		{ID: "mot", Type: TypeShell, Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailCompensate, Compensate: "don"},
		{ID: "don", Type: TypeAgent, Prompt: "Dọn hộ đống bước {{buoc_hong}} vừa làm dở"},
	}}

	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	prompts := ag.cacPrompt()
	if len(prompts) != 1 {
		t.Fatalf("bước gỡ lại phải chạy đúng 1 lần, được %d", len(prompts))
	}
	if prompts[0] != "Dọn hộ đống bước mot vừa làm dở" {
		t.Fatalf("bước gỡ lại không biết mình gỡ cho ai: %q", prompts[0])
	}
}

// CÂU HỎI KHÓ số 2, viết thành một lời khẳng định chạy được: KHÔNG gỡ lan sang
// các bước đã xong trước đó.
//
// `xong-truoc` chạy xong đàng hoàng ở đợt 1. `tao` hỏng ở đợt 2 và gỡ lại chính
// nó. `xong-truoc` phải NGUYÊN VẸN — nó không liên quan gì tới sự cố này, và một
// DAG không phải một ngăn xếp để mà tháo ngược.
func TestKhongGoLanSangCacBuocDaXongTruoc(t *testing.T) {
	r, _, db := newRunner(t)

	f := Flow{Name: "khong-lan", Steps: []Step{
		{ID: "xong-truoc", Type: TypeShell, Run: argvTroGiup(t, "in", "VIEC-DA-XONG")},
		{ID: "tao", Type: TypeShell, Needs: []string{"xong-truoc"}, Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailCompensate, Compensate: "go-lai"},
		{ID: "go-lai", Type: TypeShell, Run: argvTroGiup(t, "in", "da go")},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	buoc, _ := db.Steps(res.RunID)
	if buoc["xong-truoc"].State != store.StepDone {
		t.Fatalf("bước đã xong trước đó bị đụng vào: %q", buoc["xong-truoc"].State)
	}
	if buoc["xong-truoc"].Output != "VIEC-DA-XONG" {
		t.Fatalf("kết quả của bước không liên quan bị mất: %q", buoc["xong-truoc"].Output)
	}
	if buoc["go-lai"].State != store.StepDone {
		t.Fatalf("bước gỡ lại phải chạy: %q", buoc["go-lai"].State)
	}
}

// Nhánh `foreach` phải đi qua CÙNG một chỗ xử lý hỏng với nhánh thường.
//
// Trước khi gộp lại, hai nhánh tự xét `on_failure` riêng: nhánh foreach so bằng
// với đúng hai giá trị, nên một giá trị thứ tư sẽ lặng lẽ rơi vào nhánh "dừng"
// và không ai gỡ gì cả.
func TestForEachHongCungGoiBuocGoLai(t *testing.T) {
	r, _, db := newRunner(t)
	dauVet := filepath.Join(t.TempDir(), "go-tu-foreach.txt")

	f := Flow{Name: "lap-hong", Vars: map[string]string{"ds": "a\nb"}, Steps: []Step{
		{ID: "lap", Type: TypeShell, ForEach: "vars.ds", Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailCompensate, Compensate: "go-lai"},
		{ID: "go-lai", Type: TypeShell, Run: argvTroGiup(t, "ghi", dauVet, "DA GO")},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("phải failed, được %q", res.State)
	}
	if _, err := os.Stat(dauVet); err != nil {
		t.Fatalf("bước lặp hỏng mà KHÔNG gọi bước gỡ lại — nhánh foreach lệch khỏi nhánh thường: %v", err)
	}
	buoc, _ := db.Steps(res.RunID)
	if buoc["go-lai"].State != store.StepDone {
		t.Fatalf("bước gỡ lại phải done, được %q", buoc["go-lai"].State)
	}
}

// -------------------------- kiểm lúc viết flow --------------------------

func TestValidateSoiCompensate(t *testing.T) {
	cases := []struct {
		ten  string
		f    Flow
		chua string
	}{
		{
			"thiếu compensate",
			Flow{Name: "t", Steps: []Step{
				{ID: "a", Type: TypeNotify, Message: "m", OnFailure: OnFailCompensate},
			}},
			"phải khai báo `compensate`",
		},
		{
			"tự gỡ chính mình",
			Flow{Name: "t", Steps: []Step{
				{ID: "a", Type: TypeNotify, Message: "m", OnFailure: OnFailCompensate, Compensate: "a"},
			}},
			"tự gỡ lại chính nó",
		},
		{
			"trỏ tới bước không có",
			Flow{Name: "t", Steps: []Step{
				{ID: "a", Type: TypeNotify, Message: "m", OnFailure: OnFailCompensate, Compensate: "x"},
			}},
			"không tồn tại",
		},
		{
			"bước gỡ lại là approve — sẽ treo mãi",
			Flow{Name: "t", Steps: []Step{
				{ID: "a", Type: TypeNotify, Message: "m", OnFailure: OnFailCompensate, Compensate: "g"},
				{ID: "g", Type: TypeApprove, Message: "duyệt?"},
			}},
			"treo mãi",
		},
		{
			"bước gỡ lại idempotent — lần sự cố thứ hai không ai gỡ",
			Flow{Name: "t", Steps: []Step{
				{ID: "a", Type: TypeNotify, Message: "m", OnFailure: OnFailCompensate, Compensate: "g"},
				{ID: "g", Type: TypeAgent, Prompt: "gỡ", Idempotent: true},
			}},
			"lần sự cố thứ hai không ai gỡ",
		},
		{
			"có bước khai needs tới bước gỡ lại — nó sẽ treo",
			Flow{Name: "t", Steps: []Step{
				{ID: "a", Type: TypeNotify, Message: "m", OnFailure: OnFailCompensate, Compensate: "g"},
				{ID: "g", Type: TypeAgent, Prompt: "gỡ"},
				{ID: "sau", Type: TypeAgent, Prompt: "p", Needs: []string{"g"}},
			}},
			"không bao giờ tới lượt",
		},
		{
			"on_failure lạ vẫn phải liệt kê đủ bốn giá trị",
			Flow{Name: "t", Steps: []Step{
				{ID: "a", Type: TypeNotify, Message: "m", OnFailure: "undo"},
			}},
			"stop | continue | fallback | compensate",
		},
	}

	for _, c := range cases {
		t.Run(c.ten, func(t *testing.T) {
			if !loiChua(Validate(c.f), c.chua) {
				t.Fatalf("phải có LỖI chứa %q, được: %v", c.chua, Validate(c.f))
			}
		})
	}
}

func TestValidateCanhBaoVeBuocGoLai(t *testing.T) {
	f := Flow{Name: "t", Steps: []Step{
		{ID: "a", Type: TypeShell, Run: []string{"go", "version"},
			OnFailure: OnFailCompensate, Compensate: "g"},
		{ID: "g", Type: TypeShell, Run: []string{"go", "version"},
			OnFailure: OnFailStop, Needs: []string{"a"}},
	}}
	ps := Validate(f)
	if coLoi(ps) {
		t.Fatalf("mấy chuyện này chỉ đáng cảnh báo: %v", ps)
	}
	if !canhChua(ps, "bị BỎ QUA: gỡ-lại-của-gỡ-lại") {
		t.Fatalf("phải nói rõ on_failure của bước gỡ lại bị bỏ qua: %v", ps)
	}
	if !canhChua(ps, "`needs` của bước gỡ lại bị BỎ QUA") {
		t.Fatalf("phải nói rõ needs của bước gỡ lại bị bỏ qua: %v", ps)
	}
}

// Bước gỡ lại kiểu `notify` không gỡ gì cả — chỉ in ra một dòng chữ. Cảnh báo
// chứ không chặn: có người thật sự chỉ muốn một dòng chữ, và đó là quyền của họ.
func TestValidateCanhBaoBuocGoLaiChiLaNotify(t *testing.T) {
	f := Flow{Name: "t", Steps: []Step{
		{ID: "a", Type: TypeShell, Run: []string{"go", "version"},
			OnFailure: OnFailCompensate, Compensate: "g"},
		{ID: "g", Type: TypeNotify, Message: "hỏng rồi"},
	}}
	ps := Validate(f)
	if coLoi(ps) {
		t.Fatalf("không được chặn: %v", ps)
	}
	if !canhChua(ps, "KHÔNG gỡ lại gì cả") {
		t.Fatalf("phải cảnh báo: %v", ps)
	}
}

func TestCompensateSongSotQuaSaveVaLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := t.TempDir()

	goc := Flow{Name: "luu-undo", Steps: []Step{
		{ID: "a", Type: TypeShell, Run: []string{"go", "version"},
			OnFailure: OnFailCompensate, Compensate: "g"},
		{ID: "g", Type: TypeShell, Run: []string{"go", "version"}},
	}}
	if _, err := Save(dir, goc); err != nil {
		t.Fatal(err)
	}
	flows, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	lai := flows["luu-undo"]
	var thay bool
	for _, s := range lai.Steps {
		if s.ID == "a" {
			thay = s.OnFailure == OnFailCompensate && s.Compensate == "g"
		}
	}
	if !thay {
		t.Fatalf("compensate không sống sót qua flows.toml: %+v", lai.Steps)
	}
}
