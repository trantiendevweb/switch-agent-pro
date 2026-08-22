package flow

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// TestIdempotentChayLaiKhongLamLaiViecDaXong là bài test chính của mảnh này.
//
// Chạy CÙNG một flow HAI lần qua Runner.Start. Lượt 2 phải KHÔNG gọi agent thêm
// lần nào, nhưng vẫn phải có đủ kết quả để bước sau dùng.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: bỏ khối `thuDungLaiViecCu` trong runStep thì
// agent bị gọi 2 lần và test đỏ ngay dòng đếm.
func TestIdempotentChayLaiKhongLamLaiViecDaXong(t *testing.T) {
	r, ag, db := newRunner(t)
	ag.output = "KET QUA CUA LUOT MOT"
	dir := t.TempDir()

	f := Flow{Name: "lam-mot-lan", Steps: []Step{
		{ID: "nang", Type: TypeAgent, Prompt: "việc rất tốn tiền", Idempotent: true},
	}}
	if ps := Validate(f); coLoi(ps) {
		t.Fatalf("flow phải hợp lệ: %v", ps)
	}

	mot, err := r.Start(context.Background(), f, dir, nil)
	if err != nil || mot.State != store.RunDone {
		t.Fatalf("lượt 1: %v %q", err, mot.State)
	}
	if ag.soLanGoi() != 1 {
		t.Fatalf("lượt 1 phải gọi agent đúng 1 lần, được %d", ag.soLanGoi())
	}

	// Đổi kết quả giả: nếu bước CÓ chạy lại thì output sẽ khác, và ta nhìn ra ngay.
	ag.mu.Lock()
	ag.output = "KET QUA CUA LUOT HAI"
	ag.mu.Unlock()

	hai, err := r.Start(context.Background(), f, dir, nil)
	if err != nil || hai.State != store.RunDone {
		t.Fatalf("lượt 2: %v %q", err, hai.State)
	}
	if n := ag.soLanGoi(); n != 1 {
		t.Fatalf("LƯỢT 2 ĐÃ LÀM LẠI VIỆC ĐÃ XONG — agent bị gọi tổng %d lần", n)
	}

	buoc, _ := db.Steps(hai.RunID)
	if buoc["nang"].State != store.StepDone {
		t.Fatalf("bước phải là done, được %q", buoc["nang"].State)
	}
	if buoc["nang"].Output != "KET QUA CUA LUOT MOT" {
		t.Fatalf("phải dùng lại kết quả của lượt 1, được %q", buoc["nang"].Output)
	}
	// Sổ phải NÓI RA là mượn của ai. Một bước `done` không tốn gì mà không giải
	// thích được là một bước không ai lần ngược về việc thật được.
	if !strings.Contains(buoc["nang"].Msg, "#1") {
		t.Fatalf("sổ phải ghi rõ mượn kết quả của lượt nào, được %q", buoc["nang"].Msg)
	}
	// Và KHÔNG được chép chi phí sang: lượt này không tiêu token nào.
	if buoc["nang"].CostUSD != 0 {
		t.Fatalf("bước bỏ qua mà vẫn ghi chi phí %v — sổ cộng dồn sẽ đếm hai lần", buoc["nang"].CostUSD)
	}
}

// TestSuaPromptThiKhoaDoiVaBuocCHAYLAI là nửa còn lại, và là nửa quan trọng hơn.
//
// Định nghĩa "đã làm rồi" TÍNH TỪ VIỆC chứ không tính từ id bước. Nếu tính từ id
// thì sửa prompt xong chạy lại sẽ nhận về câu trả lời cho CÂU HỎI CŨ, không một
// dòng nào nói vì sao — kiểu hỏng tệ nhất trong ba cách khai khoá đã cân nhắc
// (xem đầu idempotent.go).
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: bỏ dòng ghi("viec", cauHoi(...)) trong
// KhoaIdem thì khoá chỉ còn phụ thuộc id + loại, prompt mới bị bỏ qua, và test
// đỏ ở dòng đếm số lần gọi.
func TestSuaPromptThiKhoaDoiVaBuocCHAYLAI(t *testing.T) {
	r, ag, _ := newRunner(t)
	dir := t.TempDir()

	cu := Flow{Name: "sua-de", Steps: []Step{
		{ID: "hoi", Type: TypeAgent, Prompt: "Câu hỏi bản A", Idempotent: true},
	}}
	moi := Flow{Name: "sua-de", Steps: []Step{
		{ID: "hoi", Type: TypeAgent, Prompt: "Câu hỏi bản B", Idempotent: true},
	}}

	if _, err := r.Start(context.Background(), cu, dir, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Start(context.Background(), moi, dir, nil); err != nil {
		t.Fatal(err)
	}
	if n := ag.soLanGoi(); n != 2 {
		t.Fatalf("sửa prompt rồi chạy lại PHẢI làm lại bước — agent gọi %d lần, đúng phải là 2", n)
	}
	prompts := ag.cacPrompt()
	if prompts[1] != "Câu hỏi bản B" {
		t.Fatalf("lượt 2 phải hỏi câu MỚI, nó hỏi %q", prompts[1])
	}
}

// TestBuocTruocDoiKetQuaThiBuocSauCHAYLAI: khoá tính từ câu hỏi ĐÃ THAY BIẾN,
// nên nó tự cuốn theo kết quả các bước trước. Đây là thứ mà một khoá do người
// dùng tự khai (`idempotency_key = "..."`) không tự có được — họ phải nhớ liệt
// kê mọi biến, và quên một cái là quay về đúng lỗi của khoá-theo-id.
func TestBuocTruocDoiKetQuaThiBuocSauCHAYLAI(t *testing.T) {
	r, ag, _ := newRunner(t)
	dir := t.TempDir()

	flowVoi := func(dauRa string) Flow {
		return Flow{Name: "noi-tiep", Steps: []Step{
			{ID: "truoc", Type: TypeShell, Run: argvTroGiup(t, "in", dauRa)},
			{ID: "sau", Type: TypeAgent, Needs: []string{"truoc"}, Idempotent: true,
				Prompt: "Tóm tắt: {{steps.truoc.output}}"},
		}}
	}

	if _, err := r.Start(context.Background(), flowVoi("BAN-A"), dir, nil); err != nil {
		t.Fatal(err)
	}
	if n := ag.soLanGoi(); n != 1 {
		t.Fatalf("lượt 1 phải gọi agent 1 lần, được %d", n)
	}
	// Bước trước ra kết quả KHÁC → prompt của bước sau khác → khoá khác → chạy lại.
	if _, err := r.Start(context.Background(), flowVoi("BAN-B"), dir, nil); err != nil {
		t.Fatal(err)
	}
	if n := ag.soLanGoi(); n != 2 {
		t.Fatalf("bước trước đổi kết quả mà bước sau vẫn bị bỏ qua — agent gọi %d lần", n)
	}
	// Còn chạy lại y nguyên thì KHÔNG làm lại.
	if _, err := r.Start(context.Background(), flowVoi("BAN-B"), dir, nil); err != nil {
		t.Fatal(err)
	}
	if n := ag.soLanGoi(); n != 2 {
		t.Fatalf("đầu vào y hệt mà vẫn làm lại — agent gọi %d lần", n)
	}
}

// TestIdempotentChepArtifactSangLuotMoi: bước bỏ qua vẫn phải để lại đủ FILE cho
// bước sau, và file phải nằm trong thư mục của lượt chạy NÀY.
//
// Trỏ thẳng sang thư mục lượt cũ thì một bước đang `done` của hôm nay sẽ mất file
// vào tuần sau, khi DonArtifact dọn cái lượt sinh ra nó. Trạng thái `done` phải
// tự đứng được.
func TestIdempotentChepArtifactSangLuotMoi(t *testing.T) {
	r, _, db := newRunner(t)
	dir := t.TempDir()

	f := Flow{Name: "art-idem", Steps: []Step{
		{ID: "viet", Type: TypeShell, Idempotent: true,
			Run:      argvTroGiup(t, "ghi", "{{artifact_dir}}/ra.txt", "NOI-DUNG-GOC"),
			Artifact: map[string]string{"ra": "ra.txt"}},
		{ID: "doc", Type: TypeShell, Needs: []string{"viet"},
			Run: argvTroGiup(t, "doc", "{{artifacts.ra}}")},
	}}

	mot, err := r.Start(context.Background(), f, dir, nil)
	if err != nil || mot.State != store.RunDone {
		t.Fatalf("lượt 1: %v %q", err, mot.State)
	}
	hai, err := r.Start(context.Background(), f, dir, nil)
	if err != nil || hai.State != store.RunDone {
		b, _ := db.Steps(hai.RunID)
		t.Fatalf("lượt 2: %v %q — %+v", err, hai.State, b)
	}

	// File phải nằm trong thư mục của lượt 2, không phải mượn của lượt 1.
	p := DuongDanArtifact(hai.RunID, "viet", "ra.txt")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("bước bỏ qua không để lại artifact trong thư mục lượt mình: %v", err)
	}
	if string(b) != "NOI-DUNG-GOC" {
		t.Fatalf("nội dung artifact chép sang bị sai: %q", string(b))
	}

	// Và bước sau đọc được nó thật.
	buoc, _ := db.Steps(hai.RunID)
	if !strings.Contains(buoc["doc"].Output, "NOI-DUNG-GOC") {
		t.Fatalf("bước sau không đọc được artifact đã chép: %q", buoc["doc"].Output)
	}
	if buoc["viet"].State != store.StepDone || !strings.Contains(buoc["viet"].Msg, "idempotent") {
		t.Fatalf("bước viet phải được ghi là bỏ qua vì idempotent: %+v", buoc["viet"])
	}
}

// TestTrungKhoaNhungArtifactCuBienMatThiCHAYLAI canh nhánh nguy hiểm nhất của
// cache: trúng khoá nhưng không dựng lại được đầy đủ.
//
// Thà chạy lại tốn tiền còn hơn báo `done` rồi để bước sau mở một file không tồn
// tại. Không có nhánh này thì mảnh idempotency biến thành một cái máy sinh ra
// bước xong-mà-rỗng.
func TestTrungKhoaNhungArtifactCuBienMatThiCHAYLAI(t *testing.T) {
	r, _, db := newRunner(t)
	dir := t.TempDir()

	f := Flow{Name: "mat-file", Steps: []Step{
		{ID: "viet", Type: TypeShell, Idempotent: true,
			Run:      argvTroGiup(t, "ghi", "{{artifact_dir}}/ra.txt", "X"),
			Artifact: map[string]string{"ra": "ra.txt"}},
	}}

	mot, err := r.Start(context.Background(), f, dir, nil)
	if err != nil || mot.State != store.RunDone {
		t.Fatalf("lượt 1: %v %q", err, mot.State)
	}
	// Giả lập DonArtifact đã dọn lượt 1.
	if err := os.RemoveAll(ArtifactRunDir(mot.RunID)); err != nil {
		t.Fatal(err)
	}

	hai, err := r.Start(context.Background(), f, dir, nil)
	if err != nil || hai.State != store.RunDone {
		t.Fatalf("lượt 2 phải chạy lại và xong bình thường: %v %q", err, hai.State)
	}
	buoc, _ := db.Steps(hai.RunID)
	if strings.Contains(buoc["viet"].Msg, "idempotent") {
		t.Fatal("BÁO XONG DÙ ARTIFACT CŨ ĐÃ MẤT — bước sau sẽ mở một file không tồn tại")
	}
	if _, err := os.Stat(DuongDanArtifact(hai.RunID, "viet", "ra.txt")); err != nil {
		t.Fatalf("chạy lại rồi mà vẫn không có file: %v", err)
	}
}

// Bước HỎNG thì KHÔNG được ghi khoá — nếu không thì lượt sau bỏ qua một việc
// chưa ai làm xong.
func TestBuocHongKhongGhiKhoaIdem(t *testing.T) {
	r, ag, db := newRunner(t)
	dir := t.TempDir()
	ag.fail = true

	f := Flow{Name: "hong-idem", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "việc", Idempotent: true},
	}}
	res, _ := r.Start(context.Background(), f, dir, nil)
	buoc, _ := db.Steps(res.RunID)
	if buoc["a"].State != store.StepFailed {
		t.Fatalf("bước phải hỏng, được %q", buoc["a"].State)
	}
	if buoc["a"].IdemKey != "" {
		t.Fatalf("bước HỎNG mà đã ghi khoá %q — lượt sau sẽ bỏ qua một việc chưa xong", buoc["a"].IdemKey)
	}

	// Chạy lại, lần này cho chạy được: bước PHẢI chạy thật.
	ag.mu.Lock()
	ag.fail = false
	ag.mu.Unlock()
	hai, err := r.Start(context.Background(), f, dir, nil)
	if err != nil || hai.State != store.RunDone {
		t.Fatalf("lượt 2: %v %q", err, hai.State)
	}
	if n := ag.soLanGoi(); n != 2 {
		t.Fatalf("phải gọi agent 2 lần (1 hỏng + 1 thật), được %d", n)
	}
}

// Bước KHÔNG bật `idempotent` thì không đụng tới cơ chế này một chút nào — hành
// vi cũ giữ nguyên, không một byte nào đổi.
func TestKhongBatThiKhongDoiGi(t *testing.T) {
	r, ag, db := newRunner(t)
	dir := t.TempDir()

	f := Flow{Name: "nhu-cu", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "việc"},
	}}
	for i := 0; i < 3; i++ {
		res, err := r.Start(context.Background(), f, dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		buoc, _ := db.Steps(res.RunID)
		if buoc["a"].IdemKey != "" {
			t.Fatalf("bước không bật idempotent mà vẫn ghi khoá: %q", buoc["a"].IdemKey)
		}
	}
	if n := ag.soLanGoi(); n != 3 {
		t.Fatalf("không bật thì mỗi lượt phải chạy thật — agent gọi %d lần, đúng là 3", n)
	}
}

// Khoá phải nhạy với MỌI thứ quyết định kết quả, không chỉ prompt.
func TestKhoaIdemDoiKhiThuQuyetDinhKetQuaDoi(t *testing.T) {
	goc := Step{ID: "a", Type: TypeAgent, Prompt: "hỏi", Idempotent: true,
		Profile: "claude:phu", Model: "sonnet", Copies: 1,
		PhaiCo: []string{"XONG"}, Artifact: map[string]string{"r": "r.md"}}
	env := map[string]string{}
	nen := KhoaIdem(goc, env)
	if nen == "" {
		t.Fatal("bật idempotent mà khoá rỗng")
	}

	doi := []struct {
		ten string
		sua func(Step) Step
	}{
		{"prompt", func(s Step) Step { s.Prompt = "hỏi khác"; return s }},
		{"profile", func(s Step) Step { s.Profile = "grok:tns"; return s }},
		{"model", func(s Step) Step { s.Model = "opus"; return s }},
		{"copies", func(s Step) Step { s.Copies = 3; return s }},
		{"route", func(s Step) Step { s.Route = "duphong"; return s }},
		{"tu_duyet_quyen", func(s Step) Step { s.TuDuyetQuyen = true; return s }},
		{"phai_co", func(s Step) Step { s.PhaiCo = []string{"XONG", "OK"}; return s }},
		{"artifact", func(s Step) Step { s.Artifact = map[string]string{"r": "khac.md"}; return s }},
		{"id", func(s Step) Step { s.ID = "b"; return s }},
		{"type", func(s Step) Step { s.Type = TypeReview; return s }},
	}
	for _, d := range doi {
		t.Run(d.ten, func(t *testing.T) {
			if k := KhoaIdem(d.sua(goc), env); k == nen {
				t.Fatalf("đổi %s mà khoá không đổi — bước sẽ bị bỏ qua nhầm", d.ten)
			}
		})
	}

	// Còn thứ KHÔNG quyết định kết quả thì không được làm đổi khoá: bảng vẽ dịch
	// một node sang phải không phải là một việc khác.
	xe := goc
	xe.X, xe.Y = 400, 900
	if KhoaIdem(xe, env) != nen {
		t.Fatal("kéo node trên bảng vẽ mà khoá đổi — sẽ chạy lại việc đã xong vô cớ")
	}
}

// Khoá phải băm NỘI DUNG artifact chứ không băm đường dẫn: đường dẫn chứa số
// lượt chạy nên nó khác nhau ở mọi lượt, và băm nó thì cache KHÔNG BAO GIỜ trúng
// — một tính năng có mặt mà không làm gì, kiểu hỏng khó thấy nhất.
func TestKhoaIdemBamNoiDungArtifactChuKhongBamDuongDan(t *testing.T) {
	dir := t.TempDir()
	a := dir + "/a.txt"
	b := dir + "/b.txt"
	if err := os.WriteFile(a, []byte("GIONG NHAU"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("GIONG NHAU"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := Step{ID: "x", Type: TypeAgent, Idempotent: true, Prompt: "Đọc {{artifacts.kq}}"}
	k1 := KhoaIdem(s, map[string]string{"artifacts.kq": a})
	k2 := KhoaIdem(s, map[string]string{"artifacts.kq": b})
	if k1 != k2 {
		t.Fatal("hai file khác đường dẫn nhưng cùng nội dung phải ra CÙNG khoá, " +
			"không thì cache không bao giờ trúng")
	}

	if err := os.WriteFile(b, []byte("DA DOI"), 0o644); err != nil {
		t.Fatal(err)
	}
	if KhoaIdem(s, map[string]string{"artifacts.kq": b}) == k1 {
		t.Fatal("nội dung artifact đổi mà khoá không đổi — bước sẽ dùng lại kết quả cũ")
	}
}

// Validate phải nói ra TRƯỚC khi tốn một lượt chạy.
func TestValidateSoiIdempotent(t *testing.T) {
	// approve và notify: bỏ qua chúng là vô nghĩa hoặc nguy hiểm.
	rao := Flow{Name: "t", Steps: []Step{
		{ID: "gac", Type: TypeApprove, Message: "duyệt?", Idempotent: true},
	}}
	if !loiChua(Validate(rao), "không còn là rào") {
		t.Fatalf("approve + idempotent phải là LỖI: %v", Validate(rao))
	}

	bao := Flow{Name: "t", Steps: []Step{
		{ID: "bao", Type: TypeNotify, Message: "xong rồi", Idempotent: true},
	}}
	if !loiChua(Validate(bao), "im lặng đúng lúc") {
		t.Fatalf("notify + idempotent phải là LỖI: %v", Validate(bao))
	}

	lap := Flow{Name: "t", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "p", ForEach: "vars.ds", Idempotent: true},
	}}
	if !loiChua(Validate(lap), "foreach") {
		t.Fatalf("foreach + idempotent phải là LỖI: %v", Validate(lap))
	}

	// shell/test/lint: CẢNH BÁO chứ không chặn — bộ chạy không nhìn thấy cây mã,
	// nhưng "cái gì quyết định kết quả bước này" là câu chỉ người viết flow trả
	// lời được.
	sh := Flow{Name: "t", Steps: []Step{
		{ID: "kiem", Type: TypeTest, Idempotent: true},
	}}
	ps := Validate(sh)
	if coLoi(ps) {
		t.Fatalf("không được chặn, chỉ cảnh báo: %v", ps)
	}
	if !canhChua(ps, "KHÔNG nhìn thấy cây mã") {
		t.Fatalf("phải cảnh báo rõ giới hạn của khoá: %v", ps)
	}
}

// Flow có `idempotent` phải sống sót qua flows.toml.
func TestIdempotentSongSotQuaSaveVaLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := t.TempDir()

	goc := Flow{Name: "luu-idem", Steps: []Step{
		{ID: "a", Type: TypeAgent, Prompt: "p", Idempotent: true},
		{ID: "b", Type: TypeAgent, Prompt: "q", Needs: []string{"a"}},
	}}
	if _, err := Save(dir, goc); err != nil {
		t.Fatal(err)
	}
	flows, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	lai := flows["luu-idem"]
	theo := map[string]bool{}
	for _, s := range lai.Steps {
		theo[s.ID] = s.Idempotent
	}
	if !theo["a"] || theo["b"] {
		t.Fatalf("cờ idempotent không sống sót qua flows.toml: %+v", theo)
	}
}
