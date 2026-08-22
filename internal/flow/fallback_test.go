// Bài kiểm cho `on_failure = "fallback"`.
//
// Mọi bài dưới đây chạy Runner.Start THẬT, với store.DB thật và TIẾN TRÌNH CON
// thật (bước `shell` gọi lại chính file test ở chế độ trợ giúp — xem
// artifact_test.go). Lý do: cả năm chỗ hỏng của `fallback` nằm ở CHỖ GỌI chứ
// không ở một hàm nào, nên gọi thẳng hàm thì bài kiểm xanh trong khi tính năng
// vẫn không chạy. Đúng cái bẫy mục 2.1 của báo cáo #199 ghi lại.
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

// ---------------------------------------------------------------------------
// BÀI KIỂM CHÍNH
// ---------------------------------------------------------------------------

// TestFallbackChayBuocThayTheRoiDIETIEP là bài chính của mảnh này.
//
// Bước `chinh` hỏng → bước `du-phong` PHẢI chạy → lượt chạy PHẢI ĐI TIẾP sang
// `sau`. Đúng chỗ `fallback` khác cả ba chính sách kia: `stop` không chạy gì
// thay, `continue` không chạy gì thay, `compensate` chạy thay rồi vẫn dừng.
//
// GỠ PHẦN SỬA RA THÌ ĐỎ Ở ĐÂU: trả `case OnFailFallback` về bản cũ (chỉ
// `Bus.Warnf` rồi `return false, ""`) thì `du-phong` không bao giờ được gọi —
// nó bị `skipped` ngay đầu lượt và không ai đánh thức. Đỏ ở dòng kiểm dấu vết
// trên đĩa, và đỏ tiếp ở dòng kiểm `sau` (bước sau bị chặn vì `chinh` failed mà
// không ai chạy thay).
func TestFallbackChayBuocThayTheRoiDIETIEP(t *testing.T) {
	r, _, db := newRunner(t)
	tmp := t.TempDir()
	vetThay := filepath.Join(tmp, "da-chay-thay.txt")
	vetSau := filepath.Join(tmp, "buoc-sau.txt")

	f := Flow{Name: "co-du-phong", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeShell, Run: argvTroGiup(t, "ghi", vetThay, "DA CHAY THAY")},
		{ID: "sau", Type: TypeShell, Needs: []string{"chinh"},
			Run: argvTroGiup(t, "ghi", vetSau, "BUOC SAU")},
	}}
	if ps := Validate(f); coLoi(ps) {
		t.Fatalf("flow phải hợp lệ: %v", ps)
	}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Bước chạy thay đã chạy THẬT — dấu vết trên đĩa, không phải một dòng log.
	if _, err := os.Stat(vetThay); err != nil {
		t.Fatalf("BƯỚC CHẠY THAY KHÔNG CHẠY — không có dấu vết: %v", err)
	}
	// 2. Và lượt chạy ĐI TIẾP: đây là chỗ khác `compensate`.
	if _, err := os.Stat(vetSau); err != nil {
		t.Fatalf("BƯỚC SAU BỊ CHẶN dù đã có người chạy thay — `fallback` là chạy thay rồi "+
			"ĐI TIẾP, không phải dừng: %v", err)
	}
	if res.State != store.RunDone {
		t.Fatalf("chạy thay xong thì lượt chạy phải completed, được %q", res.State)
	}

	// 3. SỔ VẪN GHI SỰ THẬT: bước chính hỏng thì vẫn là `failed`. Việc gán kết
	//    quả nằm ở bảng biến của lượt chạy, KHÔNG nằm ở sổ — sổ mà nói `chinh`
	//    xong là sổ nói dối, và mọi bảng thống kê đọc từ đó.
	buoc, _ := db.Steps(res.RunID)
	if buoc["chinh"].State != store.StepFailed {
		t.Fatalf("bước hỏng phải VẪN là failed trong sổ, được %q — chạy thay không xoá "+
			"chuyện bước chính đã hỏng", buoc["chinh"].State)
	}
	if buoc["du-phong"].State != store.StepDone {
		t.Fatalf("bước chạy thay phải done, được %q", buoc["du-phong"].State)
	}
}

// TestKetQuaBuocChayThayDocDuocBangTenBuocHONG canh quyết định thiết kế đáng cãi
// nhất của fallback.go.
//
// Bước sau khai `needs = ["chinh"]` và đọc `{{steps.chinh.output}}`. Không gán
// kết quả của bước chạy thay vào đó thì nó nhận một ô rỗng — và với bước `shell`
// đó còn là lỗi cứng (BuocConSot chặn ngay). Tức là KHÔNG có phần này thì
// `fallback` không dùng được vào việc gì.
//
// GỠ PHẦN SỬA RA THÌ ĐỎ Ở ĐÂU: bỏ lời gọi ganKetQuaThayThe trong chayThayThe thì
// `{{steps.chinh.output}}` không được thay, BuocConSot chặn bước `sau`, và cả
// lượt chạy hỏng.
func TestKetQuaBuocChayThayDocDuocBangTenBuocHONG(t *testing.T) {
	r, _, db := newRunner(t)
	vet := filepath.Join(t.TempDir(), "ket-qua.txt")

	f := Flow{Name: "noi-tiep", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeShell, Run: argvTroGiup(t, "in", "HANG-DU-PHONG")},
		// Bước sau KHÔNG nhắc tên `du-phong` một chữ nào: nó không có cách nào
		// biết trước lượt này ai làm việc. Cả điểm của `fallback` nằm ở đây.
		{ID: "sau", Type: TypeShell, Needs: []string{"chinh"},
			Run: argvTroGiup(t, "ghi", vet, "{{steps.chinh.output}}")},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		buoc, _ := db.Steps(res.RunID)
		t.Fatalf("lượt chạy %q — bước `sau` báo %q: %s", res.State,
			buoc["sau"].State, buoc["sau"].Msg)
	}
	b, err := os.ReadFile(vet)
	if err != nil {
		t.Fatalf("bước sau không chạy: %v", err)
	}
	if string(b) != "HANG-DU-PHONG" {
		t.Fatalf("bước sau nhận %q, muốn %q — kết quả của bước chạy thay phải đọc được "+
			"bằng tên bước hỏng, nếu không thì không ai viết được flow dùng `fallback`",
			string(b), "HANG-DU-PHONG")
	}
}

// TestBuocChayThayKHONGChayONhungLuotBinhThuong canh CÁI BẪY LỚN NHẤT, và là
// bài kiểm ghim đúng thứ đo được ở lượt chạy thật #61.
//
// Bước chạy thay thường không có `needs` nào, tức là một GỐC của DAG. Để yên
// thì nó chạy ngay đợt đầu của MỌI lượt chạy — chạy thay cho một bước chưa kịp
// hỏng. Với một bước dự phòng thật (gọi nhà cung cấp API thứ hai, dựng lại máy)
// thì đó là tiêu tiền cho việc không ai cần.
//
// GỠ PHẦN SỬA RA THÌ ĐỎ Ở ĐÂU: bỏ BuocThayThe khỏi BuocNgoaiLichThuong thì bước
// chạy thay không bị loại khỏi lịch thường, nó chạy ở đợt đầu, và dấu vết xuất
// hiện dù không có gì hỏng.
func TestBuocChayThayKHONGChayONhungLuotBinhThuong(t *testing.T) {
	r, _, db := newRunner(t)
	vet := filepath.Join(t.TempDir(), "khong-duoc-co.txt")

	f := Flow{Name: "yen-lanh", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: argvTroGiup(t, "in", "ổn"),
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeShell, Run: argvTroGiup(t, "ghi", vet, "KHONG NEN CO")},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("không có gì hỏng thì lượt chạy phải xong, được %q", res.State)
	}
	if _, err := os.Stat(vet); err == nil {
		t.Fatal("BƯỚC CHẠY THAY ĐÃ CHẠY DÙ KHÔNG CÓ GÌ HỎNG — nó làm thay một việc " +
			"chưa ai hỏng, và tốn đúng hạn mức mà `fallback` sinh ra để tiết kiệm")
	}
	// Ghi `skipped` KÈM LỜI GIẢI THÍCH, không để trống: ô trống trên bảng đọc là
	// "chưa tới lượt", còn đây là "sẽ không chạy trừ khi có chuyện".
	buoc, _ := db.Steps(res.RunID)
	if buoc["du-phong"].State != store.StepSkipped {
		t.Fatalf("bước chạy thay phải là skipped, được %q", buoc["du-phong"].State)
	}
	if !strings.Contains(buoc["du-phong"].Msg, "chỉ chạy khi chinh hỏng") {
		t.Fatalf("phải nói rõ vì sao bỏ qua, được %q", buoc["du-phong"].Msg)
	}
}

// TestBuocChayThayChaySAU KHI bước chính đã hỏng, không phải song song với nó.
//
// Đây là hàng 1 của bảng đo trong fallback.go — cái bẫy TINH VI nhất, vì nó
// từng trông như "tính năng chạy được". Bước chạy thay không có `needs` nên nó
// nằm cùng đợt với bước chính và CHẠY SONG SONG, tức là nó chạy xong trước khi
// bước chính kịp hỏng. Một bước dự phòng không thể phản ứng với sự cố chưa xảy
// ra: nếu bước chính may mà xong thì bước dự phòng vẫn đã tiêu tiền.
//
// Đo bằng THỨ TỰ trên đĩa: bước chính ghi dấu rồi mới hỏng; bước chạy thay đọc
// dấu đó. Đọc được = nó chạy SAU. Cách này không phụ thuộc vào đồng hồ.
func TestBuocChayThayChaySAUKhiBuocChinhDaHong(t *testing.T) {
	r, _, _ := newRunner(t)
	tmp := t.TempDir()
	co := filepath.Join(tmp, "chinh-da-chay.txt")
	rac := filepath.Join(tmp, "rac.txt")
	doc := filepath.Join(tmp, "du-phong-doc-duoc.txt")

	f := Flow{Name: "thu-tu", Steps: []Step{
		// hong-lan-dau: lần chạy đầu ghi cờ `co` rồi thoát 1.
		{ID: "chinh", Type: TypeShell, Run: argvTroGiup(t, "hong-lan-dau", rac, co),
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		// `doc` thoát 1 nếu file chưa tồn tại — tức là bước chạy thay HỎNG nếu
		// nó chạy trước bước chính.
		{ID: "du-phong", Type: TypeShell, Run: argvTroGiup(t, "doc-sang", co, doc)},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("bước chạy thay CHẠY TRƯỚC bước chính — nó không đọc được dấu vết của "+
			"bước chính, tức là nó đang phản ứng với một sự cố chưa xảy ra (lượt chạy %q)",
			res.State)
	}
	if _, err := os.Stat(doc); err != nil {
		t.Fatalf("bước chạy thay không thấy dấu vết của bước chính: %v", err)
	}
}

// TestBuocChayThayCungHONG: không còn đường nào nữa thì DỪNG, và bước sau
// KHÔNG được đi tiếp.
//
// GỠ PHẦN SỬA RA THÌ ĐỎ Ở ĐÂU: bỏ nhánh `case OnFailFallback` trong xuLyHong thì
// bước chạy thay không chạy và lượt chạy thành `completed` — đỏ ở dòng kiểm
// trạng thái. Điều kiện `state(Fallback) == done` trong choDiTiep thì KHÔNG đo
// được ở đây: lượt chạy đã dừng ngay tại chỗ hỏng nên bước sau chưa kịp được
// xét. Chỗ đo nó là TestChayLaiMotLuotDaHONGThiBuocSauVANBiChan.
func TestBuocChayThayCungHONG(t *testing.T) {
	r, _, db := newRunner(t)
	vetSau := filepath.Join(t.TempDir(), "khong-duoc-chay.txt")

	f := Flow{Name: "het-duong", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeShell, Run: argvTroGiup(t, "hong")},
		{ID: "sau", Type: TypeShell, Needs: []string{"chinh"},
			Run: argvTroGiup(t, "ghi", vetSau, "KHONG NEN CO")},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("bước chạy thay cũng hỏng thì lượt chạy phải failed, được %q", res.State)
	}
	if _, err := os.Stat(vetSau); err == nil {
		t.Fatal("BƯỚC SAU ĐÃ CHẠY dù không ai làm được việc của bước trước nó — " +
			"`fallback` vừa biến thành `continue`")
	}
	buoc, _ := db.Steps(res.RunID)
	// Bước chạy thay chạy đúng MỘT lần: không có vòng thay-thế-của-thay-thế.
	if buoc["du-phong"].Attempt != 1 {
		t.Fatalf("bước chạy thay phải chạy đúng 1 lần, sổ ghi %d", buoc["du-phong"].Attempt)
	}
}

// TestChayLaiMotLuotDaHONGThiBuocSauVANBiChan là chỗ đo điều kiện
// `state(Fallback) == done` trong choDiTiep — và nó chỉ lộ ra trên đường CHẠY LẠI.
//
// `sagent flow resume <#>` chạy lại được cả một lượt đã `failed` (Resume chỉ
// dừng sớm với `completed`/`cancelled`). Lúc đó trạng thái dựng lại TỪ SỔ: bước
// chính `failed`, bước chạy thay cũng `failed`, và không còn ai đang ở giữa
// chừng để dừng lượt chạy nữa. Nếu cửa của `fallback` mở mà không kiểm bước chạy
// thay có xong thật không thì bước sau chạy — trên nền một việc KHÔNG AI LÀM
// ĐƯỢC. Đó đúng là lỗi #23 mà choDiTiep sinh ra để chống, chỉ đổi tên trường.
//
// GỠ PHẦN SỬA RA THÌ ĐỎ Ở ĐÂU: bỏ `s.state(st.Fallback) == store.StepDone` khỏi
// choDiTiep thì bước `sau` chạy ở lượt chạy lại và dấu vết xuất hiện.
func TestChayLaiMotLuotDaHONGThiBuocSauVANBiChan(t *testing.T) {
	r, _, _ := newRunner(t)
	vetSau := filepath.Join(t.TempDir(), "khong-duoc-chay.txt")

	f := Flow{Name: "chay-lai-het-duong", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeShell, Run: argvTroGiup(t, "hong")},
		{ID: "sau", Type: TypeShell, Needs: []string{"chinh"},
			Run: argvTroGiup(t, "ghi", vetSau, "KHONG NEN CO")},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("lượt đầu phải failed, được %q", res.State)
	}

	if _, err := r.Resume(context.Background(), res.RunID, f); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(vetSau); err == nil {
		t.Fatal("CHẠY LẠI một lượt đã hỏng thì bước sau LẠI CHẠY — trên nền một việc mà " +
			"cả bước chính lẫn hàng dự phòng đều không làm được")
	}
}

// Thông điệp lỗi cuối phải PHÂN BIỆT ĐƯỢC "bước hỏng" với "bước hỏng VÀ hàng dự
// phòng cũng hỏng". Hai câu đó dẫn tới hai việc phải làm khác hẳn nhau: một cái
// là sửa bước, một cái là "cả hai đường đều tắc".
//
// Đọc thẳng từ sự kiện FlowFailed thật, không đọc bản chép tay.
func TestThongDiepNoiRoCaHaiDuongDeuTac(t *testing.T) {
	lyDoCuaLuot := func(t *testing.T, thayHong bool) string {
		t.Helper()
		r, _, _ := newRunner(t)
		thay := argvTroGiup(t, "in", "hàng dự phòng chạy được")
		if thayHong {
			thay = argvTroGiup(t, "hong")
		}
		f := Flow{Name: "tin-nhan", Steps: []Step{
			{ID: "chinh", Type: TypeShell, Run: argvTroGiup(t, "hong"),
				OnFailure: OnFailFallback, Fallback: "du-phong"},
			{ID: "du-phong", Type: TypeShell, Run: thay},
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

	// Hàng dự phòng chạy được thì KHÔNG có sự kiện hỏng nào cả — lượt chạy xong.
	if ly := lyDoCuaLuot(t, false); ly != "" {
		t.Fatalf("chạy thay xong rồi mà vẫn báo hỏng: %q", ly)
	}
	ban := lyDoCuaLuot(t, true)
	if !strings.Contains(ban, "CŨNG HỎNG") || !strings.Contains(ban, "du-phong") {
		t.Fatalf("cả hai đường đều tắc mà thông điệp không nói ra — người đọc không phân "+
			"biệt được với một bước hỏng bình thường: %q", ban)
	}
}

// TestChayLaiSauRaoDuyetVanTimThayKetQuaBuocChayThay — CHẠY LẠI GIỮA CHỪNG.
//
// Một lượt chạy có thể dừng ở rào duyệt SAU khi bước chạy thay đã làm xong việc.
// Lượt sau `Resume` dựng lại trạng thái từ SQLite, và bảng biến trong bộ nhớ thì
// mất sạch. Không dựng lại được thì bước sau rào duyệt nhận một ô rỗng — và đây
// đúng là kiểu hỏng khó thấy nhất: nó chỉ xảy ra ở flow CÓ rào duyệt, tức là
// đúng những flow người ta cẩn thận nhất.
//
// GỠ PHẦN SỬA RA THÌ ĐỎ Ở ĐÂU: bỏ lời gọi ganKetQuaThayThe trong execute (chỗ
// ngay sau khi nạp sổ) thì bài này đỏ còn bài
// TestKetQuaBuocChayThayDocDuocBangTenBuocHONG vẫn xanh — nửa còn thiếu chỉ lộ
// ra khi có Resume.
func TestChayLaiSauRaoDuyetVanTimThayKetQuaBuocChayThay(t *testing.T) {
	r, _, db := newRunner(t)
	vet := filepath.Join(t.TempDir(), "sau-rao.txt")

	f := Flow{Name: "co-rao", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeShell, Run: argvTroGiup(t, "in", "HANG-DU-PHONG")},
		{ID: "rao", Type: TypeApprove, Needs: []string{"chinh"}, Message: "duyệt chứ?"},
		{ID: "sau", Type: TypeShell, Needs: []string{"rao"},
			Run: argvTroGiup(t, "ghi", vet, "{{steps.chinh.output}}")},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunWaiting {
		t.Fatalf("phải dừng ở rào duyệt, được %q", res.State)
	}

	// Duyệt rồi chạy lại — TỪ SỔ, không còn gì trong bộ nhớ.
	if err := db.SetStep(res.RunID, "rao", store.StepDone, "duyệt", 0); err != nil {
		t.Fatal(err)
	}
	res2, err := r.Resume(context.Background(), res.RunID, f)
	if err != nil {
		t.Fatal(err)
	}
	if res2.State != store.RunDone {
		buoc, _ := db.Steps(res.RunID)
		t.Fatalf("chạy lại không xong: %q — bước `sau` báo %q: %s",
			res2.State, buoc["sau"].State, buoc["sau"].Msg)
	}
	b, err := os.ReadFile(vet)
	if err != nil {
		t.Fatalf("bước sau rào duyệt không chạy: %v", err)
	}
	if string(b) != "HANG-DU-PHONG" {
		t.Fatalf("sau khi chạy lại, bước sau nhận %q — kết quả bước chạy thay KHÔNG được "+
			"dựng lại từ sổ", string(b))
	}
}

// TestBuocChayThayBietMinhDangThayChoBuocNao: một bước dự phòng dùng chung cho
// ba bước phải phân biệt được mình đang thay cho ai, nếu không người viết flow
// phải chép ra ba bước dự phòng gần như giống hệt nhau.
//
// Dùng CHUNG biến {{buoc_hong}} với bước gỡ lại — cùng một câu hỏi thì cùng một
// tên, hai cái tên cho một thứ là hai cái phải nhớ.
func TestBuocChayThayBietMinhDangThayChoBuocNao(t *testing.T) {
	r, ag, _ := newRunner(t)
	ag.output = "xong"

	f := Flow{Name: "biet-thay-ai", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeAgent, Prompt: "làm lại việc của bước {{buoc_hong}}"},
	}}

	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	ps := ag.cacPrompt()
	if len(ps) != 1 {
		t.Fatalf("agent phải được gọi đúng 1 lần, được %d", len(ps))
	}
	if !strings.Contains(ps[0], "bước chinh") {
		t.Fatalf("bước chạy thay không biết mình đang thay cho ai: %q", ps[0])
	}
}

// TestForEachHongCungGoiBuocChayThay: nhánh `foreach` và nhánh thường phải đi
// qua CÙNG MỘT chỗ xét `on_failure`.
//
// Mục 1.4 của báo cáo #199 ghi lại đúng lớp lỗi này: hai nhánh từng tự xét
// riêng, chúng lệch nhau thật, và một giá trị lặng lẽ rơi vào nhánh sai. Bài này
// ghim cho `fallback` chuyện đó không tái diễn.
func TestForEachHongCungGoiBuocChayThay(t *testing.T) {
	r, _, _ := newRunner(t)
	vet := filepath.Join(t.TempDir(), "thay-cho-foreach.txt")

	f := Flow{Name: "lap-hong", Vars: map[string]string{"ds": "a\nb"}, Steps: []Step{
		{ID: "chinh", Type: TypeShell, ForEach: "vars.ds", Run: argvTroGiup(t, "hong"),
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeShell, Run: argvTroGiup(t, "ghi", vet, "DA THAY")},
	}}

	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(vet); err != nil {
		t.Fatalf("bước `foreach` hỏng mà KHÔNG ai chạy thay — hai nhánh lại lệch nhau: %v", err)
	}
}

// ---------------------------------------------------------------------------
// VALIDATE
// ---------------------------------------------------------------------------

// Bước chạy thay bị loại khỏi lịch thường, nên một `needs` trỏ vào nó là một
// bước TREO VĨNH VIỄN. Phải là LỖI chứ không phải cảnh báo.
func TestValidateChanNeedsTroVaoBuocChayThay(t *testing.T) {
	f := Flow{Name: "treo", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: []string{"x"},
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeShell, Run: []string{"x"}},
		{ID: "sau", Type: TypeShell, Needs: []string{"du-phong"}, Run: []string{"x"}},
	}}
	ps := Validate(f)
	if !loiChua(ps, "sẽ không bao giờ tới lượt") {
		t.Fatalf("không chặn `needs` trỏ vào bước chạy thay: %v", ps)
	}
	// Và phải chỉ đường ra, không chỉ nói "sai".
	if !loiChua(ps, "đọc bằng tên bước hỏng") {
		t.Fatalf("báo lỗi mà không nói cách viết cho đúng: %v", ps)
	}
}

func TestValidateChanBuocTuChayThayChinhNo(t *testing.T) {
	f := Flow{Name: "tu-thay", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: []string{"x"},
			OnFailure: OnFailFallback, Fallback: "chinh"},
	}}
	if !loiChua(Validate(f), "không thể tự chạy thay chính nó") {
		t.Fatalf("không chặn bước tự chạy thay chính nó: %v", Validate(f))
	}
}

func TestValidateCanhBaoVeBuocChayThay(t *testing.T) {
	f := Flow{Name: "canh", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: []string{"x"},
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "moc", Type: TypeShell, Run: []string{"x"}},
		{ID: "du-phong", Type: TypeShell, Run: []string{"x"}, Needs: []string{"moc"},
			OnFailure: OnFailContinue},
	}}
	ps := Validate(f)
	if !canhChua(ps, "`needs` của bước chạy thay bị BỎ QUA") {
		t.Fatalf("không cảnh báo `needs` bị bỏ qua: %v", ps)
	}
	if !canhChua(ps, "của bước chạy thay bị BỎ QUA: thay-thế-của-thay-thế") {
		t.Fatalf("không cảnh báo on_failure bị bỏ qua: %v", ps)
	}
}

// `fallback` khai ở một bước không dùng tới nó là một DÒNG CHẾT: nó nằm đó trông
// như có tác dụng. Cùng lớp lỗi với `plugin` khai ở bước không phải type plugin.
func TestValidateCanhBaoFallbackKhaiMaKhongDung(t *testing.T) {
	f := Flow{Name: "dong-chet", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: []string{"x"},
			OnFailure: OnFailStop, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeShell, Run: []string{"x"}},
	}}
	if !canhChua(Validate(f), "`fallback` chỉ có tác dụng") {
		t.Fatalf("không cảnh báo `fallback` khai mà không dùng: %v", Validate(f))
	}
	// Nhưng KHÔNG nói hai lần khi on_failure = "compensate": chỗ đó đã có cảnh
	// báo riêng của VanDeCompensate.
	g := Flow{Name: "ca-hai", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: []string{"x"},
			OnFailure: OnFailCompensate, Compensate: "go-lai", Fallback: "du-phong"},
		{ID: "go-lai", Type: TypeShell, Run: []string{"x"}},
		{ID: "du-phong", Type: TypeShell, Run: []string{"x"}},
	}}
	var dem int
	for _, p := range Validate(g) {
		if strings.Contains(p.Msg, "`fallback` sẽ không được dùng") ||
			strings.Contains(p.Msg, "`fallback` chỉ có tác dụng") {
			dem++
		}
	}
	if dem != 1 {
		t.Fatalf("cùng một chuyện được nói %d lần, muốn 1: %v", dem, Validate(g))
	}
}

func TestValidateChanBuocChayThayLaApprove(t *testing.T) {
	f := Flow{Name: "rao-thay", Steps: []Step{
		{ID: "chinh", Type: TypeShell, Run: []string{"x"},
			OnFailure: OnFailFallback, Fallback: "du-phong"},
		{ID: "du-phong", Type: TypeApprove, Message: "duyệt?"},
	}}
	if !loiChua(Validate(f), "bước chạy thay không được là `approve`") {
		t.Fatalf("không chặn bước chạy thay kiểu approve: %v", Validate(f))
	}
}

// ---------------------------------------------------------------------------
// MỘT NGUỒN, NHIỀU MẶT
// ---------------------------------------------------------------------------

// BuocNgoaiLichThuong là chỗ DUY NHẤT trả lời câu "bước nào sẽ không chạy", và
// cả bộ chạy, `flow show` lẫn bảng chạy khan đều hỏi nó. Bài này ghim rằng nó
// gom được CẢ HAI vai và không bỏ sót vai nào.
func TestBuocNgoaiLichThuongGomCaHaiVai(t *testing.T) {
	f := Flow{Name: "hai-vai", Steps: []Step{
		{ID: "a", Type: TypeShell, Run: []string{"x"}, OnFailure: OnFailCompensate, Compensate: "chung"},
		{ID: "b", Type: TypeShell, Run: []string{"x"}, OnFailure: OnFailFallback, Fallback: "chung"},
		{ID: "c", Type: TypeShell, Run: []string{"x"}, OnFailure: OnFailFallback, Fallback: "rieng"},
		{ID: "chung", Type: TypeShell, Run: []string{"x"}},
		{ID: "rieng", Type: TypeShell, Run: []string{"x"}},
	}}
	m := BuocNgoaiLichThuong(f)
	if len(m) != 2 {
		t.Fatalf("gom được %d bước, muốn 2: %v", len(m), m)
	}
	// Bước mang cả hai vai phải nói ra CẢ HAI: người đọc sổ thấy "skipped" cần
	// biết đủ lý do, không chỉ lý do đầu tiên tìm thấy.
	if !strings.Contains(m["chung"], "gỡ lại") || !strings.Contains(m["chung"], "chạy thay") {
		t.Fatalf("bước mang hai vai chỉ nói ra một: %q", m["chung"])
	}
	if !strings.Contains(m["rieng"], "chỉ chạy khi c hỏng") {
		t.Fatalf("câu mô tả sai: %q", m["rieng"])
	}
}
