package flow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// ---------------------------------------------------------------------------
// `foreach` + `artifact`
// ---------------------------------------------------------------------------

// TestForEachArtifactMoiLuotMotThuMucVaMotBanKe là bài test quan trọng nhất của
// mảnh này. Nó chạy MỘT lượt flow THẬT với tiến trình con thật và khẳng định ba
// điều cùng lúc:
//
//  1. ba lượt lặp ghi ra BA file khác nhau (không lượt nào đè lượt nào);
//  2. bước sau nhận đúng MỘT phần tử argv mà vẫn tới được cả ba file;
//  3. thứ tự trong bản kê là thứ tự danh sách, không phải thứ tự chạy xong.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU:
//   - bỏ dòng `env[KhoaArtifactDir] = lapDir` trong runForEach (step.go) thì cả
//     ba lượt cùng ghi vào thư mục của BƯỚC, thư mục lượt lặp rỗng, và bước sau
//     hỏng ngay ở hợp đồng đầu ra ("mục 1 … không có file");
//   - bỏ lời gọi GhiDanhSachArtifact thì `gop-ke` không mở được bản kê và bước
//     sau thoát mã 1.
func TestForEachArtifactMoiLuotMotThuMucVaMotBanKe(t *testing.T) {
	r, _, db := newRunner(t)

	f := Flow{Name: "toa-ra", Vars: map[string]string{"ds": "alpha\nbeta\ngamma"},
		Steps: []Step{
			{
				ID:       "viet",
				Type:     TypeShell,
				ForEach:  "vars.ds",
				Run:      argvTroGiup(t, "ghi", "{{artifact_dir}}/phan.txt", "{{item}}"),
				Artifact: map[string]string{"phan": "phan.txt"},
			},
			{
				ID:    "gop",
				Type:  TypeShell,
				Needs: []string{"viet"},
				// MỘT phần tử argv, và nó tới được cả ba file. Đây là câu trả lời
				// cho "argv không có vòng lặp".
				Run: argvTroGiup(t, "gop-ke", "{{artifacts.phan.danh_sach}}"),
			},
		}}

	if ps := Validate(f); coLoi(ps) {
		t.Fatalf("flow mẫu phải hợp lệ, nhưng: %v", ps)
	}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	buoc, _ := db.Steps(res.RunID)
	if res.State != store.RunDone {
		t.Fatalf("lượt chạy phải xong, được %q — các bước: %+v", res.State, buoc)
	}

	// (1)+(2)+(3): bước sau mở được đủ ba file, đúng thứ tự danh sách.
	got := buoc["gop"].Output
	if !strings.Contains(got, "so=3") {
		t.Fatalf("bước sau phải thấy 3 file, nó báo: %q", got)
	}
	if !strings.Contains(got, "gop=alpha+beta+gamma") {
		t.Fatalf("bản kê sai thứ tự hoặc thiếu mục, bước sau đọc được: %q", got)
	}

	// Và trên ĐĨA: ba thư mục lượt lặp riêng biệt, tên là đúng con số {{index}}.
	for i, muc := range []string{"alpha", "beta", "gamma"} {
		p := DuongDanArtifactLap(res.RunID, "viet", i+1, "phan.txt")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("thiếu artifact của lượt %d: %v", i+1, err)
		}
		if string(b) != muc {
			t.Fatalf("lượt %d chứa %q, phải là %q — các lượt đang ghi đè lên nhau",
				i+1, string(b), muc)
		}
	}
}

// Một lượt lặp KHÔNG giao hàng là một MỤC bị mất, không phải một bước "gần xong".
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: bỏ khối gọi ThieuArtifactLap trong
// runForEach thì bước được ghi `done`, bản kê vẫn được sinh, và nó trỏ tới một
// file không tồn tại — đúng kiểu hỏng mà hợp đồng artifact sinh ra để chặn.
func TestForEachArtifactMotLuotCamThiCaBuocHong(t *testing.T) {
	r, _, db := newRunner(t)

	f := Flow{Name: "cam", Vars: map[string]string{"ds": "alpha\nbeta\ngamma"},
		Steps: []Step{{
			ID:       "viet",
			Type:     TypeShell,
			ForEach:  "vars.ds",
			Run:      argvTroGiup(t, "ghi-tru", "{{artifact_dir}}/phan.txt", "{{item}}", "beta"),
			Artifact: map[string]string{"phan": "phan.txt"},
		}}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("một lượt không giao hàng thì cả bước phải hỏng, được %q", res.State)
	}
	buoc, _ := db.Steps(res.RunID)
	msg := buoc["viet"].Msg
	if !strings.Contains(msg, "mục 2") || !strings.Contains(msg, "beta") {
		t.Fatalf("thông báo phải chỉ đúng MỤC nào câm, nó nói: %q", msg)
	}

	// Và KHÔNG được sinh bản kê: một bản kê trỏ tới file của lượt vừa hỏng còn
	// tệ hơn không có bản kê nào.
	if _, err := os.Stat(DuongDanDanhSach(res.RunID, "viet", "phan")); err == nil {
		t.Fatal("bước hỏng mà vẫn sinh bản kê")
	}
}

// `phai_co` ở nhánh lặp trước đây KHÔNG được kiểm một lần nào — một cái cờ nằm
// đó trông như đang bật.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: bỏ khối ThieuPhaiCo trong runForEach thì
// lượt chạy về `completed` và bài này đỏ ở dòng đầu.
func TestForEachKiemPhaiCoTungLuot(t *testing.T) {
	r, _, db := newRunner(t)

	f := Flow{Name: "hop-dong", Vars: map[string]string{"ds": "alpha\nbeta"},
		Steps: []Step{{
			ID:      "in",
			Type:    TypeShell,
			ForEach: "vars.ds",
			Run:     argvTroGiup(t, "in", "{{item}}"),
			PhaiCo:  []string{"KHONG-BAO-GIO-CO-CHUOI-NAY"},
		}}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("`phai_co` không thoả thì bước phải hỏng, được %q", res.State)
	}
	// Hai lượt chạy SONG SONG nên lượt nào báo lỗi trước là không đoán được — chốt
	// "mục 1" là tự dựng một bài kiểm chớp tắt. Thứ phải canh là: thông báo có gọi
	// tên MỘT MỤC CỤ THỂ hay không, chứ không phải mục nào.
	buoc, _ := db.Steps(res.RunID)
	msg := buoc["in"].Msg
	if !strings.Contains(msg, "mục 1") && !strings.Contains(msg, "mục 2") {
		t.Fatalf("phải nói rõ MỤC nào không đạt hợp đồng, nó nói: %q", msg)
	}
	if !strings.Contains(msg, "alpha") && !strings.Contains(msg, "beta") {
		t.Fatalf("thông báo phải kèm NỘI DUNG mục để lần ngược được, nó nói: %q", msg)
	}
}

// ---------------------------------------------------------------------------
// `foreach` + `idempotent`
// ---------------------------------------------------------------------------

// ĐÂY LÀ BÀI TEST TRẢ LỜI ĐÚNG CÂU HỎI CỦA #199: thêm một mục MỚI vào danh sách
// thì chỉ mục MỚI đó chạy, hai mục cũ được bỏ qua — và không mục nào bị bỏ sót.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU:
//   - trả lại lệnh chặn trong VanDeIdempotent → Validate báo lỗi, dòng đầu đỏ;
//   - bỏ khối `thuDungLaiMucCu` trong runForEach → lượt 2 gọi agent 3 lần nữa
//     (tổng 5), dòng "phải gọi thêm ĐÚNG 1 lần" đỏ;
//   - bỏ `ghi("foreach.item", …)` VÀ để prompt không có {{item}} thì mọi lượt
//     chung một khoá — bài TestForEachIdempotentKhongTronKetQuaGiuaCacMuc bắt.
func TestForEachIdempotentChiChayMucMOI(t *testing.T) {
	r, ag, db := newRunner(t)
	dir := t.TempDir()

	dung := func(ds string) Flow {
		return Flow{Name: "lap-nho", Vars: map[string]string{"ds": ds},
			Steps: []Step{{ID: "xu-ly", Type: TypeAgent, ForEach: "vars.ds",
				Prompt: "Xử lý {{item}}", Idempotent: true}}}
	}

	f1 := dung("alpha\nbeta")
	if ps := Validate(f1); coLoi(ps) {
		t.Fatalf("foreach + idempotent phải hợp lệ, nhưng: %v", ps)
	}
	if _, err := r.Start(context.Background(), f1, dir, nil); err != nil {
		t.Fatal(err)
	}
	if n := ag.soLanGoi(); n != 2 {
		t.Fatalf("lượt 1 phải chạy cả 2 mục, gọi %d lần", n)
	}

	// Lượt 2: danh sách có thêm MỘT mục mới.
	res2, err := r.Start(context.Background(), dung("alpha\nbeta\ngamma"), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res2.State != store.RunDone {
		t.Fatalf("lượt 2 phải xong, được %q", res2.State)
	}
	if n := ag.soLanGoi(); n != 3 {
		t.Fatalf("lượt 2 phải gọi thêm ĐÚNG 1 lần (mục mới), tổng đang là %d", n)
	}
	// Mục mới phải THẬT SỰ được xử lý, không phải bị bỏ qua theo khoá của mục khác.
	if !strings.Contains(strings.Join(ag.cacPrompt(), " | "), "gamma") {
		t.Fatalf("mục MỚI không được xử lý — các prompt: %v", ag.cacPrompt())
	}

	buoc, _ := db.Steps(res2.RunID)
	if !strings.Contains(buoc["xu-ly"].Msg, "bỏ qua 2 mục") {
		t.Fatalf("trạng thái bước phải nói ra số mục bỏ qua, nó nói: %q", buoc["xu-ly"].Msg)
	}
	// Kết quả gộp vẫn phải có ĐỦ ba mục — bỏ qua không được làm ngắn danh sách.
	out := buoc["xu-ly"].Output
	for _, muc := range []string{"=== alpha ===", "=== beta ===", "=== gamma ==="} {
		if !strings.Contains(out, muc) {
			t.Fatalf("kết quả gộp thiếu %q:\n%s", muc, out)
		}
	}
}

// Khoá bám NỘI DUNG MỤC, không bám VỊ TRÍ: đảo thứ tự danh sách thì không mục
// nào phải chạy lại.
//
// Đây là chỗ phân biệt lựa chọn "không băm {{index}}" với lựa chọn ngược lại —
// băm index vào thì bài này đỏ với "phải không gọi thêm lần nào".
func TestForEachIdempotentDoiChoKhongMatCache(t *testing.T) {
	r, ag, _ := newRunner(t)
	dir := t.TempDir()

	dung := func(ds string) Flow {
		return Flow{Name: "dao", Vars: map[string]string{"ds": ds},
			Steps: []Step{{ID: "x", Type: TypeAgent, ForEach: "vars.ds",
				Prompt: "làm {{item}}", Idempotent: true}}}
	}
	if _, err := r.Start(context.Background(), dung("a\nb\nc"), dir, nil); err != nil {
		t.Fatal(err)
	}
	if n := ag.soLanGoi(); n != 3 {
		t.Fatalf("lượt 1 phải chạy 3 mục, gọi %d lần", n)
	}
	if _, err := r.Start(context.Background(), dung("c\na\nb"), dir, nil); err != nil {
		t.Fatal(err)
	}
	if n := ag.soLanGoi(); n != 3 {
		t.Fatalf("đảo thứ tự không được làm mất cache, tổng đã gọi %d lần", n)
	}
}

// Hai mục KHÁC NHAU không bao giờ được dùng chung một khoá.
//
// Bài này canh đúng lớp lỗi mà #199 gọi là "lỗ mất việc im lặng": nếu khoá không
// cuốn được mục vào thì mục thứ hai trở đi bị bỏ qua theo kết quả của mục đầu.
func TestForEachIdempotentKhongTronKetQuaGiuaCacMuc(t *testing.T) {
	s := Step{ID: "x", Type: TypeAgent, ForEach: "vars.ds", Idempotent: true,
		// CỐ Ý không dùng {{item}} trong prompt: đây là ca mà `cauHoi` một mình
		// KHÔNG phân biệt được hai lượt, và chỉ `ghi("foreach.item", …)` cứu.
		Prompt: "làm một việc gì đó"}
	k1 := KhoaIdem(s, map[string]string{"item": "alpha", "index": "1"})
	k2 := KhoaIdem(s, map[string]string{"item": "beta", "index": "2"})
	if k1 == "" || k2 == "" {
		t.Fatal("bước bật idempotent phải có khoá")
	}
	if k1 == k2 {
		t.Fatal("hai MỤC khác nhau ra cùng một khoá — mục thứ hai sẽ bị bỏ qua theo " +
			"kết quả của mục thứ nhất, và không có gì báo")
	}
	// Cùng mục, khác VỊ TRÍ → cùng khoá (xem TestForEachIdempotentDoiChoKhongMatCache).
	if KhoaIdem(s, map[string]string{"item": "alpha", "index": "1"}) !=
		KhoaIdem(s, map[string]string{"item": "alpha", "index": "9"}) {
		t.Fatal("vị trí trong danh sách không phải danh tính của việc")
	}
}

// Trúng cache theo từng mục thì ARTIFACT của mục đó phải được chép sang lượt
// mới — và phải chép từ ĐÚNG chỉ số cũ, kể cả khi mục đã đổi chỗ.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: trong chepArtifactMucCu, đổi `cu.Idx` thành
// `chiSo` thì lượt 2 chép nhầm file của mục khác và dòng so nội dung đỏ.
func TestForEachIdempotentChepArtifactTheoDungChiSoCu(t *testing.T) {
	r, _, db := newRunner(t)
	dir := t.TempDir()

	dung := func(ds string) Flow {
		return Flow{Name: "chep", Vars: map[string]string{"ds": ds},
			Steps: []Step{
				{ID: "viet", Type: TypeShell, ForEach: "vars.ds", Idempotent: true,
					Run:      argvTroGiup(t, "ghi", "{{artifact_dir}}/phan.txt", "{{item}}"),
					Artifact: map[string]string{"phan": "phan.txt"}},
				{ID: "gop", Type: TypeShell, Needs: []string{"viet"},
					Run: argvTroGiup(t, "gop-ke", "{{artifacts.phan.danh_sach}}")},
			}}
	}

	if _, err := r.Start(context.Background(), dung("alpha\nbeta\ngamma"), dir, nil); err != nil {
		t.Fatal(err)
	}

	// Lượt 2 ĐẢO thứ tự: mọi mục đều trúng cache, nhưng chỉ số của chúng đổi hết.
	res2, err := r.Start(context.Background(), dung("gamma\nalpha\nbeta"), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	buoc, _ := db.Steps(res2.RunID)
	if res2.State != store.RunDone {
		t.Fatalf("lượt 2 phải xong, được %q — %+v", res2.State, buoc)
	}
	if !strings.Contains(buoc["viet"].Msg, "bỏ qua 3 mục") {
		t.Fatalf("cả ba mục phải trúng cache, trạng thái nói: %q", buoc["viet"].Msg)
	}
	// Bản kê của lượt 2 phải ra đúng THỨ TỰ MỚI, với nội dung ĐÚNG của từng mục.
	if got := buoc["gop"].Output; !strings.Contains(got, "gop=gamma+alpha+beta") {
		t.Fatalf("artifact bị chép nhầm chỉ số — bước sau đọc được: %q", got)
	}
}

// Trúng cache mà artifact cũ đã bị dọn thì coi như KHÔNG trúng: thà chạy lại tốn
// tiền còn hơn báo xong rồi để bước sau mở một file không tồn tại.
//
// Cùng luật với bước thường (xem thuDungLaiViecCu), viết lại cho nhánh lặp vì
// đây là hai đường mã khác nhau.
func TestForEachIdempotentMatArtifactCuThiChayLai(t *testing.T) {
	r, _, db := newRunner(t)
	dir := t.TempDir()

	dung := func() Flow {
		return Flow{Name: "mat", Vars: map[string]string{"ds": "alpha\nbeta"},
			Steps: []Step{{ID: "viet", Type: TypeShell, ForEach: "vars.ds", Idempotent: true,
				Run:      argvTroGiup(t, "ghi", "{{artifact_dir}}/phan.txt", "{{item}}"),
				Artifact: map[string]string{"phan": "phan.txt"}}}}
	}
	res1, err := r.Start(context.Background(), dung(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Dọn tay thư mục artifact của lượt 1 — đúng thứ DonArtifact làm sau bảy ngày.
	if err := os.RemoveAll(ArtifactRunDir(res1.RunID)); err != nil {
		t.Fatal(err)
	}

	res2, err := r.Start(context.Background(), dung(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	buoc, _ := db.Steps(res2.RunID)
	if res2.State != store.RunDone {
		t.Fatalf("lượt 2 phải chạy lại và xong, được %q — %+v", res2.State, buoc)
	}
	if strings.Contains(buoc["viet"].Msg, "bỏ qua") {
		t.Fatalf("artifact cũ đã mất thì KHÔNG được coi là trúng cache: %q", buoc["viet"].Msg)
	}
	for i, muc := range []string{"alpha", "beta"} {
		b, err := os.ReadFile(DuongDanArtifactLap(res2.RunID, "viet", i+1, "phan.txt"))
		if err != nil || string(b) != muc {
			t.Fatalf("lượt %d phải được ghi lại thật: %v / %q", i+1, err, string(b))
		}
	}
}

// Bước KHÔNG bật `idempotent` thì bảng flow_step_items không có một dòng nào.
// Không ai phải trả giá cho một tính năng mình không dùng.
func TestForEachKhongBatIdempotentThiKhongGhiSo(t *testing.T) {
	r, _, db := newRunner(t)
	f := Flow{Name: "khong-so", Vars: map[string]string{"ds": "a\nb"},
		Steps: []Step{{ID: "x", Type: TypeAgent, ForEach: "vars.ds", Prompt: "{{item}}"}}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	muc, err := db.StepItems(res.RunID, "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(muc) != 0 {
		t.Fatalf("bước không bật idempotent mà vẫn ghi %d dòng mục", len(muc))
	}
}

// ---------------------------------------------------------------------------
// Cú pháp: hình dạng placeholder phải KHỚP hình dạng bước sản xuất
// ---------------------------------------------------------------------------

// Placeholder có hậu tố mà KHÔNG được thay thì phải bị chặn ở bước shell, không
// được chốt thành một câu tiếng Việt rồi đi vào argv.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: trả `conSotArtifact` về dạng cũ (không bắt
// hậu tố) thì `{{artifacts.x.danh_sach}}` trượt khỏi cả ExpandChay lẫn
// ArtifactConSot, và lệnh nhận nguyên chuỗi `{{artifacts.x.danh_sach}}` làm tên
// file.
func TestArtifactConSotBatCaDangCoHauTo(t *testing.T) {
	if got := ArtifactConSot("cat {{artifacts.x."+HauToDanhSach+"}}", nil); got != "x" {
		t.Fatalf("phải bắt được placeholder có hậu tố, được %q", got)
	}
	if got := ArtifactConSot("cat {{artifacts.x}}", nil); got != "x" {
		t.Fatalf("phải bắt được placeholder trần, được %q", got)
	}
	vars := map[string]string{"artifacts.x." + HauToDanhSach: "/tmp/ke.txt"}
	if got := ArtifactConSot("cat {{artifacts.x."+HauToDanhSach+"}}", vars); got != "" {
		t.Fatalf("thay được rồi thì không còn sót, được %q", got)
	}
}

// Bản kê nằm đúng chỗ và chứa đường dẫn TUYỆT ĐỐI, mỗi dòng một cái.
func TestGhiDanhSachArtifactRaDuongDanTuyetDoi(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	s := Step{ID: "viet", Type: TypeShell, ForEach: "vars.ds",
		Artifact: map[string]string{"phan": "phan.txt"}}
	if _, err := ChuanBiArtifact(7, s); err != nil {
		t.Fatal(err)
	}
	if err := GhiDanhSachArtifact(7, s, 3); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(DuongDanDanhSach(7, "viet", "phan"))
	if err != nil {
		t.Fatal(err)
	}
	dong := strings.Fields(string(b))
	if len(dong) != 3 {
		t.Fatalf("bản kê phải có 3 dòng, được %d: %q", len(dong), string(b))
	}
	for i, d := range dong {
		if !filepath.IsAbs(d) {
			t.Fatalf("dòng %d không tuyệt đối: %q", i+1, d)
		}
		if want := DuongDanArtifactLap(7, "viet", i+1, "phan.txt"); d != want {
			t.Fatalf("dòng %d là %q, phải là %q", i+1, d, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Bước `model`: khoá idempotency phải cuốn PROMPT vào
// ---------------------------------------------------------------------------

// Hai bước `model` cùng route mà khác hẳn câu hỏi thì KHÔNG được ra cùng khoá.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: bỏ nhánh `case TypeModel` trong cauHoi
// (step.go) thì cả hai khoá bằng nhau, và lượt sau dùng lại câu trả lời của một
// câu hỏi khác.
func TestKhoaIdemBuocModelCuonPromptVao(t *testing.T) {
	a := Step{ID: "hoi", Type: TypeModel, Route: "deepseek", Idempotent: true,
		Prompt: "1 + 1 bằng mấy?"}
	b := a
	b.Prompt = "Thủ đô nước Pháp là gì?"
	if KhoaIdem(a, nil) == KhoaIdem(b, nil) {
		t.Fatal("hai câu hỏi khác nhau ra cùng một khoá — lượt sau sẽ trả lời " +
			"câu hỏi này bằng câu trả lời của câu hỏi kia")
	}
}

// Sổ chép lại ĐÚNG câu hỏi của bước `model` — trước đây cột `prompt` để trống.
func TestSoLuuCauHoiCuaBuocModel(t *testing.T) {
	got := cauHoi(Step{ID: "hoi", Type: TypeModel, Prompt: "hỏi {{ten}}"},
		map[string]string{"ten": "abc"})
	if got != "hỏi abc" {
		t.Fatalf("sổ phải lưu câu hỏi ĐÃ THAY BIẾN của bước model, được %q", got)
	}
}

// Giữ cho thông báo lỗi cú pháp thật sự chỉ ra cách sửa: nó phải nêu đích danh
// hậu tố phải dùng, không chỉ nói "sai".
func TestCauChanCuPhapNoiRaCachSua(t *testing.T) {
	f := Flow{Name: "t", Steps: []Step{
		{ID: "a", Type: TypeShell, ForEach: "vars.ds", Run: []string{"go", "version"},
			Artifact: map[string]string{"bao-cao": "x.md"}},
		{ID: "b", Type: TypeShell, Needs: []string{"a"}, Run: []string{"cat", "{{artifacts.bao-cao}}"}},
	}}
	ps := Validate(f)
	if !loiChua(ps, fmt.Sprintf("{{artifacts.bao-cao.%s}}", HauToDanhSach)) {
		t.Fatalf("câu chặn phải viết ra ĐÚNG cú pháp thay thế, nó nói: %v", ps)
	}
}

// Bảng liệt kê artifact phải gọi ĐÚNG TÊN file của một bước lặp.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: đổi `flow.BanDoTenArtifact` trong
// internal/api/artifact.go về `flow.TenTheoDuong` thì mọi file của bước lặp mất
// tên, và bảng in "(không bước nào khai)" — một lời khẳng định SAI.
func TestBanDoTenArtifactGoiDungTenFileCuaBuocLap(t *testing.T) {
	f := Flow{Name: "t", Steps: []Step{
		{ID: "soi", Type: TypeShell, ForEach: "vars.ds", Run: []string{"go", "version"},
			Artifact: map[string]string{"ban-va": "ban-va.diff"}},
		{ID: "gom", Type: TypeShell, Needs: []string{"soi"}, Run: []string{"go", "version"},
			Artifact: map[string]string{"tong": "tong.md"}},
	}}
	tra := BanDoTenArtifact(f)
	cases := map[string]string{
		"soi/3/ban-va.diff":        "ban-va (mục 3)",
		"soi/danh-sach-ban-va.txt": "ban-va (bản kê)",
		"gom/tong.md":              "tong",
		"soi/12/khong-ai-khai.txt": "",
		"linh-tinh":                "",
	}
	for duong, muon := range cases {
		if got := tra(duong); got != muon {
			t.Fatalf("tra %q ra %q, muốn %q", duong, got, muon)
		}
	}
}
