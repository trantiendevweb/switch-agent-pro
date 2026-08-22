package flow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// ---------------------------------------------------------------------------
// TIẾN TRÌNH TRỢ GIÚP
//
// Bước `shell` chạy argv THẲNG, không qua shell — nên không có `sh -c` để mượn,
// và `echo`/`type` thì khác nhau giữa Windows với Linux. Cách chuẩn của Go là
// gọi lại CHÍNH file test này ở một chế độ khác, nên cùng một bài test chạy
// được ở cả hai hệ điều hành mà không cần cài gì.
//
// PHẢI đi qua tiến trình thật chứ không giả lập: điều cần chứng minh là bước sau
// MỞ ĐƯỢC FILE mà bước trước ghi ra, và một cái giả trả về chuỗi thì không
// chứng minh được gì về đĩa.
// ---------------------------------------------------------------------------

const bienTroGiup = "SAGENT_TEST_TRO_GIUP"

// mocDau nằm ở ĐẦU nội dung — chỗ duy nhất mà cả hai cái trần (MaxInject và
// MaxStepOutput) đều cắt mất. Thấy được nó ở bước sau = artifact đã đi vòng qua
// cả hai.
const mocDau = "MOC-DAU-FILE-DUNG-XOA"

// noiDungDai dài hơn hẳn flow.MaxInject (6.000) và store.MaxStepOutput (32 KiB).
func noiDungDai() string {
	return mocDau + "\n" + strings.Repeat("x", 40*1024) + "\nMOC-CUOI-FILE"
}

// TestTroGiupTienTrinh không phải một bài test — nó là thân của tiến trình con.
// Chạy bình thường thì nó thoát ngay vì biến môi trường không được đặt.
func TestTroGiupTienTrinh(t *testing.T) {
	if os.Getenv(bienTroGiup) != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "trợ giúp: thiếu lệnh")
		os.Exit(2)
	}
	switch args[0] {

	// viet <file>: ghi nội dung dài ra file VÀ in y hệt ra màn hình.
	// In cả ra màn hình là cố ý: nhờ vậy một bài test so được hai đường truyền
	// (chuỗi và file) trên cùng một dữ liệu.
	case "viet":
		if err := os.WriteFile(args[1], []byte(noiDungDai()), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(noiDungDai())

	// doc <file>: in 30 ký tự ĐẦU và tổng độ dài.
	case "doc":
		b, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "không đọc được:", err)
			os.Exit(1)
		}
		fmt.Printf("dau=%s dai=%d\n", string(b[:30]), len(b))

	// khong-ghi-gi: chạy xong, thoát 0, nhưng không để lại file nào.
	case "khong-ghi-gi":
		fmt.Print("tôi xong rồi (nhưng không ghi file)")

	// hong-lan-dau <file> <cờ>: lần đầu ghi file rồi HỎNG; lần sau thoát 0 mà
	// KHÔNG ghi gì. Dùng để soi chuyện dọn thư mục artifact giữa các lần thử.
	case "hong-lan-dau":
		if _, err := os.Stat(args[2]); err != nil {
			_ = os.WriteFile(args[2], []byte("da-chay"), 0o644)
			_ = os.WriteFile(args[1], []byte("RÁC CỦA LẦN THỬ 1"), 0o644)
			fmt.Fprintln(os.Stderr, "hỏng có chủ ý ở lần thử 1")
			os.Exit(1)
		}
		fmt.Print("lần thử 2 chạy trót lọt nhưng không ghi file nào")

	// hong: thoát mã 1 kèm một câu lý do. Không đụng vào đĩa.
	case "hong":
		fmt.Fprintln(os.Stderr, "hỏng có chủ ý")
		os.Exit(1)

	// in <chuoi>: in đúng chuỗi được đưa ra màn hình, không đụng vào đĩa.
	case "in":
		fmt.Print(args[1])

	// doc-sang <nguon> <dich>: HỎNG nếu <nguon> chưa tồn tại, còn có thì chép
	// tên nó sang <dich>. Dùng để đo THỨ TỰ giữa hai bước mà không đụng tới
	// đồng hồ: "đọc được dấu vết của bước kia" = "chạy sau bước kia".
	case "doc-sang":
		if _, err := os.Stat(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "chưa có", args[1], "— tôi chạy TRƯỚC bước kia")
			os.Exit(1)
		}
		if err := os.WriteFile(args[2], []byte(args[1]), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print("đọc được")

	// ghi <file> <chuoi>: ghi đúng chuỗi được đưa.
	case "ghi":
		if err := os.WriteFile(args[1], []byte(args[2]), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print("đã ghi")

	default:
		fmt.Fprintln(os.Stderr, "trợ giúp: không hiểu lệnh "+args[0])
		os.Exit(2)
	}
	os.Exit(0)
}

// argvTroGiup dựng argv gọi lại chính file test này ở chế độ trợ giúp.
func argvTroGiup(t *testing.T, lenh ...string) []string {
	t.Helper()
	t.Setenv(bienTroGiup, "1")
	out := []string{os.Args[0], "-test.run=TestTroGiupTienTrinh", "--"}
	return append(out, lenh...)
}

// ---------------------------------------------------------------------------
// BÀI TEST CHÍNH của mảnh này.
// ---------------------------------------------------------------------------

// TestArtifactChuyenFileNguyenVenGiuaHaiBuoc là bài test quan trọng nhất ở đây.
//
// Nó chạy MỘT lượt flow THẬT qua Runner.Start (không gọi thẳng hàm nào của
// artifact.go) và khẳng định hai điều CÙNG LÚC, trên cùng một dữ liệu:
//
//	1. bước sau MỞ ĐƯỢC file bước trước ghi ra, và thấy cả phần ĐẦU;
//	2. cùng dữ liệu đó đi qua {{steps.x.output}} thì phần ĐẦU BIẾN MẤT.
//
// Vế 2 mới là thứ chứng minh mảnh này đáng tồn tại. Không có nó thì bài test chỉ
// nói "đường mới chạy được", chứ không nói "đường cũ hỏng ở chỗ nào".
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: bỏ dòng trộn `arts` vào env trong runStep
// (step.go) thì `{{artifacts.bao-cao}}` không được thay, bước `doc` là bước
// shell nên `ArtifactConSot` chặn ngay và cả lượt chạy hỏng.
func TestArtifactChuyenFileNguyenVenGiuaHaiBuoc(t *testing.T) {
	r, ag, db := newRunner(t)

	f := Flow{Name: "chuyen-file", Steps: []Step{
		{
			ID:       "viet",
			Type:     TypeShell,
			Run:      argvTroGiup(t, "viet", "{{artifact_dir}}/bao-cao.md"),
			Artifact: map[string]string{"bao-cao": "bao-cao.md"},
		},
		{
			ID:    "doc",
			Type:  TypeShell,
			Needs: []string{"viet"},
			Run:   argvTroGiup(t, "doc", "{{artifacts.bao-cao}}"),
		},
		{
			ID:     "qua-chuoi",
			Type:   TypeAgent,
			Needs:  []string{"viet"},
			Prompt: "Đọc kết quả: {{steps.viet.output}}",
		},
	}}

	if ps := Validate(f); coLoi(ps) {
		t.Fatalf("flow mẫu phải hợp lệ, nhưng: %v", ps)
	}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		buoc, _ := db.Steps(res.RunID)
		t.Fatalf("lượt chạy phải xong, được %q — các bước: %+v", res.State, buoc)
	}

	// (1) ĐƯỜNG FILE: bước sau thấy phần ĐẦU và thấy đủ độ dài.
	buoc, err := db.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	doc := buoc["doc"].Output
	if !strings.Contains(doc, mocDau) {
		t.Fatalf("bước sau KHÔNG đọc được phần đầu của artifact; nó thấy: %q", doc)
	}
	if !strings.Contains(doc, fmt.Sprintf("dai=%d", len(noiDungDai()))) {
		t.Fatalf("bước sau đọc thiếu byte — nó báo: %q, đúng phải là dai=%d", doc, len(noiDungDai()))
	}

	// (2) ĐƯỜNG CHUỖI trên CÙNG dữ liệu: phần đầu đã bị cắt mất.
	//
	// Đây là lý do mảnh artifact tồn tại, viết thành một lời khẳng định chạy
	// được. Ngày nào MaxInject đổi cách cắt thì bài test này phải được đọc lại,
	// chứ không được sửa cho xanh.
	prompts := ag.cacPrompt()
	if len(prompts) != 1 {
		t.Fatalf("bước agent phải chạy đúng 1 lần, được %d", len(prompts))
	}
	if strings.Contains(prompts[0], mocDau) {
		t.Fatal("bất ngờ: {{steps.x.output}} CÒN giữ phần đầu — trần cắt đã đổi, đọc lại MaxInject")
	}
	if !strings.Contains(prompts[0], "đã cắt bớt phần đầu") {
		t.Fatalf("prompt phải nói rõ là đã bị cắt, được: %.200q", prompts[0])
	}
}

// TestArtifactKhaiMaKhongCoFileThiBuocHONG canh HỢP ĐỒNG ĐẦU RA.
//
// Bước thoát mã 0, không lỗi gì, in ra một câu nghe rất xuôi tai — nhưng không
// ghi file nào. Đây đúng kiểu hỏng tệ nhất mà dự án này chống: không sập, chỉ
// lặng lẽ gật đầu. Không có luật này thì bước sau nhận một đường dẫn hợp lệ trỏ
// vào hư không và hỏng ở một chỗ chẳng liên quan tới nguyên nhân.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: bỏ khối `ThieuArtifact` trong runStep thì
// bước `viet` được ghi `done` và cả lượt chạy `completed` — test đỏ ở dòng kiểm
// trạng thái ngay dưới.
func TestArtifactKhaiMaKhongCoFileThiBuocHONG(t *testing.T) {
	r, _, db := newRunner(t)

	f := Flow{Name: "hua-suong", Steps: []Step{{
		ID:       "viet",
		Type:     TypeShell,
		Run:      argvTroGiup(t, "khong-ghi-gi"),
		Artifact: map[string]string{"bao-cao": "bao-cao.md"},
	}}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("bước hứa artifact mà không giao thì lượt chạy phải HỎNG, được %q", res.State)
	}
	buoc, _ := db.Steps(res.RunID)
	if buoc["viet"].State != store.StepFailed {
		t.Fatalf("bước phải failed, được %q", buoc["viet"].State)
	}
	// Thông điệp phải gọi TÊN artifact thiếu — người đọc sổ không được phải đi
	// đoán xem cái gì không có.
	if !strings.Contains(buoc["viet"].Msg, "bao-cao") {
		t.Fatalf("thông điệp lỗi phải nói rõ artifact nào thiếu, được: %q", buoc["viet"].Msg)
	}
	// Và output của bước vẫn phải còn — bằng chứng phải nằm ở chỗ người ta tìm.
	if !strings.Contains(buoc["viet"].Output, "không ghi file") {
		t.Fatalf("output của lần chạy hỏng phải được giữ, được: %q", buoc["viet"].Output)
	}
}

// TestArtifactDonThuMucTruocMoiLanThu canh quyết định #4 của artifact.go.
//
// Kịch bản: lần thử 1 GHI ĐƯỢC file rồi mới hỏng; lần thử 2 thoát 0 nhưng không
// ghi gì. Nếu thư mục artifact không được dọn giữa hai lần thử thì file rác của
// lần 1 làm hợp đồng đầu ra "đủ", và `retry` biến một bước hỏng thành một bước
// xong — cùng loại lỗi im lặng với `phai_co`.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: chuyển lời gọi ChuanBiArtifact ra NGOÀI vòng
// `for attempt` (tức chỉ dọn một lần) thì lượt chạy này thành `completed` và
// test đỏ ngay dòng đầu.
func TestArtifactDonThuMucTruocMoiLanThu(t *testing.T) {
	r, _, db := newRunner(t)
	co := filepath.Join(t.TempDir(), "da-chay-lan-1")

	f := Flow{Name: "thu-lai", Steps: []Step{{
		ID:       "viet",
		Type:     TypeShell,
		Retry:    1,
		Run:      argvTroGiup(t, "hong-lan-dau", "{{artifact_dir}}/ket-qua.txt", co),
		Artifact: map[string]string{"ket-qua": "ket-qua.txt"},
	}}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("lần thử 2 không ghi file nên bước phải HỎNG; file rác của lần thử 1 "+
			"đã được tính là artifact hợp lệ (được %q)", res.State)
	}
	// Và cái file rác đó phải đã bị dọn, không nằm lại chờ ai đó đọc nhầm.
	if _, err := os.Stat(DuongDanArtifact(res.RunID, "viet", "ket-qua.txt")); err == nil {
		t.Fatal("file của lần thử 1 vẫn còn trên đĩa sau khi bước hỏng")
	}
	buoc, _ := db.Steps(res.RunID)
	if buoc["viet"].Attempt != 2 {
		t.Fatalf("phải thử đúng 2 lần, sổ ghi %d", buoc["viet"].Attempt)
	}
}

// TestHaiLuotChaySongSongKhongGiamLenNhau trả lời câu "hai lượt chạy song song
// có giẫm lên nhau không" bằng một phép đo, không bằng một lời hứa.
//
// Chạy CÙNG một flow hai lần trên CÙNG một thư mục dự án, mỗi lần ghi một nội
// dung khác. Sau khi lượt 2 xong, artifact của lượt 1 phải còn nguyên nội dung
// của lượt 1.
func TestHaiLuotChaySongSongKhongGiamLenNhau(t *testing.T) {
	r, _, _ := newRunner(t)
	dir := t.TempDir()

	flowVoi := func(noiDung string) Flow {
		return Flow{Name: "ghi", Steps: []Step{{
			ID:       "viet",
			Type:     TypeShell,
			Run:      argvTroGiup(t, "ghi", "{{artifact_dir}}/ra.txt", noiDung),
			Artifact: map[string]string{"ra": "ra.txt"},
		}}}
	}

	mot, err := r.Start(context.Background(), flowVoi("LUOT-MOT"), dir, nil)
	if err != nil || mot.State != store.RunDone {
		t.Fatalf("lượt 1: %v %q", err, mot.State)
	}
	hai, err := r.Start(context.Background(), flowVoi("LUOT-HAI"), dir, nil)
	if err != nil || hai.State != store.RunDone {
		t.Fatalf("lượt 2: %v %q", err, hai.State)
	}
	if mot.RunID == hai.RunID {
		t.Fatal("hai lượt chạy trùng số — cả bài test này vô nghĩa")
	}

	doc := func(runID int64) string {
		b, err := os.ReadFile(DuongDanArtifact(runID, "viet", "ra.txt"))
		if err != nil {
			t.Fatalf("không đọc được artifact của lượt #%d: %v", runID, err)
		}
		return string(b)
	}
	if got := doc(mot.RunID); got != "LUOT-MOT" {
		t.Fatalf("lượt 2 đã giẫm lên artifact của lượt 1: %q", got)
	}
	if got := doc(hai.RunID); got != "LUOT-HAI" {
		t.Fatalf("artifact lượt 2 sai: %q", got)
	}
}

// TestDocDuocChanCaArtifactChuKhongChiChanChuoi canh cái rào `doc_duoc` không có
// cửa sau.
//
// Nếu chỉ lọc `outputs` mà quên artifact thì bước bị cấm đọc kết quả của bước
// `bi-mat` vẫn nhận ĐƯỜNG DẪN tới file của đúng bước đó — và một đường dẫn thì
// mở ra được toàn bộ, không bị cắt gì cả. Tức là cái rào chặn được bản tóm tắt
// và để lọt bản đầy đủ.
//
// GỠ PHẦN SỬA RA THÌ TEST ĐỎ Ở ĐÂU: bỏ nhánh `artifactChoDoc` trong
// MoiTruongArtifact thì prompt nhận một đường dẫn thật và test đỏ ở dòng cuối.
func TestDocDuocChanCaArtifactChuKhongChiChanChuoi(t *testing.T) {
	r, ag, _ := newRunner(t)

	f := Flow{Name: "rao", Steps: []Step{
		{
			ID:       "bi-mat",
			Type:     TypeShell,
			Run:      argvTroGiup(t, "ghi", "{{artifact_dir}}/kho.txt", "DU-LIEU-MAT"),
			Artifact: map[string]string{"kho": "kho.txt"},
		},
		{
			ID:      "nguoi-la",
			Type:    TypeAgent,
			Needs:   []string{"bi-mat"},
			DocDuoc: []string{}, // khai rỗng = cấm đọc mọi bước, cố ý
			Prompt:  "Mở file này: {{artifacts.kho}}",
		},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("lượt chạy phải xong, được %q", res.State)
	}
	prompts := ag.cacPrompt()
	if len(prompts) != 1 {
		t.Fatalf("phải có đúng 1 prompt, được %d", len(prompts))
	}
	if strings.Contains(prompts[0], ArtifactStepDir(res.RunID, "bi-mat")) {
		t.Fatalf("doc_duoc bị hở: bước không được phép vẫn nhận đường dẫn artifact — %q", prompts[0])
	}
	if !strings.Contains(prompts[0], "không được phép đọc artifact") {
		t.Fatalf("phải NÓI RA việc chặn chứ không im lặng thay bằng chuỗi rỗng — %q", prompts[0])
	}
}

// TestArtifactCuaBuocChuaChayThiNoiThangLaChuaCo: bước sản xuất hỏng thì bước
// tiêu thụ KHÔNG được nhận một đường dẫn trông rất thật trỏ vào hư không.
func TestArtifactCuaBuocChuaChayThiNoiThangLaChuaCo(t *testing.T) {
	r, ag, _ := newRunner(t)

	f := Flow{Name: "hong-truoc", Steps: []Step{
		{
			ID:        "viet",
			Type:      TypeShell,
			Run:       argvTroGiup(t, "khong-ghi-gi"),
			Artifact:  map[string]string{"kq": "kq.txt"},
			OnFailure: OnFailContinue, // hỏng nhưng cho đi tiếp, để soi bước sau
		},
		{
			ID:     "dung",
			Type:   TypeAgent,
			Needs:  []string{"viet"},
			Prompt: "Mở: {{artifacts.kq}}",
		},
	}}

	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	prompts := ag.cacPrompt()
	if len(prompts) != 1 {
		t.Fatalf("bước sau phải chạy (on_failure=continue), được %d prompt", len(prompts))
	}
	if !strings.Contains(prompts[0], "chưa có") {
		t.Fatalf("prompt phải nói thẳng là artifact chưa có, được: %q", prompts[0])
	}
	if strings.Contains(prompts[0], "{{artifacts.") {
		t.Fatalf("placeholder sống lọt vào prompt: %q", prompts[0])
	}
}

// TestBuocShellDungLaiKhiThieuArtifact: bước shell KHÔNG được chốt placeholder
// thành một câu tiếng Việt rồi đưa vào lệnh như một tên file.
func TestBuocShellDungLaiKhiThieuArtifact(t *testing.T) {
	r, _, db := newRunner(t)

	f := Flow{Name: "shell-thieu", Steps: []Step{
		{
			ID:        "viet",
			Type:      TypeShell,
			Run:       argvTroGiup(t, "khong-ghi-gi"),
			Artifact:  map[string]string{"kq": "kq.txt"},
			OnFailure: OnFailContinue,
		},
		{
			ID:    "doc",
			Type:  TypeShell,
			Needs: []string{"viet"},
			Run:   argvTroGiup(t, "doc", "{{artifacts.kq}}"),
		},
	}}

	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	buoc, _ := db.Steps(res.RunID)
	msg := buoc["doc"].Msg
	if !strings.Contains(msg, "artifact") || !strings.Contains(msg, "kq") {
		t.Fatalf("lỗi phải chỉ đúng vào artifact thiếu, được: %q", msg)
	}
	if strings.Contains(msg, "no such file") || strings.Contains(msg, "cannot find") {
		t.Fatalf("lệnh đã chạy với một tên file bịa: %q", msg)
	}
}

// ---------------------------------------------------------------------------
// DỌN RÁC
// ---------------------------------------------------------------------------

// TestDonArtifactGiuLuotChoDuyetVaXoaLuotDaXong trả lời "artifact sống bao lâu,
// dọn lúc nào" bằng một phép đo.
//
// Ba thư mục cùng tuổi (rất cũ), khác nhau ở TRẠNG THÁI lượt chạy. Chỉ lượt đã
// kết thúc mới bị dọn — lượt đang chờ người duyệt phải sống sót, nếu không thì
// duyệt xong resume sẽ mất sạch file của các bước trước.
func TestDonArtifactGiuLuotChoDuyetVaXoaLuotDaXong(t *testing.T) {
	_, _, db := newRunner(t)
	cu := time.Now().Add(-30 * 24 * time.Hour)

	taoLuot := func(state string) int64 {
		id, err := db.CreateRun("f", "d", "{}")
		if err != nil {
			t.Fatal(err)
		}
		if err := db.SetRunState(id, state); err != nil {
			t.Fatal(err)
		}
		dir := ArtifactStepDir(id, "b")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, "x.txt")
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		// Làm cho cũ từ trong ra ngoài: moiDoi soi cả file bên trong.
		for _, q := range []string{p, dir, ArtifactRunDir(id)} {
			if err := os.Chtimes(q, cu, cu); err != nil {
				t.Fatal(err)
			}
		}
		return id
	}

	xong := taoLuot(store.RunDone)
	hong := taoLuot(store.RunFailed)
	cho := taoLuot(store.RunWaiting)
	dangChay := taoLuot(store.RunRunning)

	// Một thư mục MỚI của lượt đã xong: đúng tuổi thì không được đụng vào.
	moi, err := db.CreateRun("f", "d", "{}")
	if err != nil {
		t.Fatal(err)
	}
	_ = db.SetRunState(moi, store.RunDone)
	if err := os.MkdirAll(ArtifactStepDir(moi, "b"), 0o755); err != nil {
		t.Fatal(err)
	}

	n := DonArtifact(db, ArtifactGiuLai, time.Now())
	if n != 2 {
		t.Fatalf("phải dọn đúng 2 lượt (xong + hỏng, đều đã quá hạn), dọn %d", n)
	}

	con := func(id int64) bool {
		_, err := os.Stat(ArtifactRunDir(id))
		return err == nil
	}
	if con(xong) || con(hong) {
		t.Fatal("lượt đã kết thúc và quá hạn phải bị dọn")
	}
	if !con(cho) {
		t.Fatal("LƯỢT ĐANG CHỜ DUYỆT BỊ DỌN — duyệt xong resume sẽ mất hết file bước trước")
	}
	if !con(dangChay) {
		t.Fatal("lượt đang chạy bị dọn")
	}
	if !con(moi) {
		t.Fatal("lượt mới xong (chưa quá hạn) bị dọn")
	}
}

// ---------------------------------------------------------------------------
// KIỂM TRA LÚC VIẾT FLOW — bắt lỗi TRƯỚC khi tốn một lượt chạy.
// ---------------------------------------------------------------------------

func coLoi(ps []Problem) bool {
	for _, p := range ps {
		if !p.Warn {
			return true
		}
	}
	return false
}

func loiChua(ps []Problem, s string) bool {
	for _, p := range ps {
		if !p.Warn && strings.Contains(p.Msg, s) {
			return true
		}
	}
	return false
}

func canhChua(ps []Problem, s string) bool {
	for _, p := range ps {
		if p.Warn && strings.Contains(p.Msg, s) {
			return true
		}
	}
	return false
}

func TestValidateChanArtifactKhaiHong(t *testing.T) {
	buoc := func(s Step) Flow {
		s.Type = orElse(s.Type, TypeShell)
		if s.Type == TypeShell && len(s.Run) == 0 {
			s.Run = []string{"go", "version"}
		}
		return Flow{Name: "t", Steps: []Step{s}}
	}

	cases := []struct {
		ten  string
		f    Flow
		chua string
	}{
		{
			"đường dẫn tuyệt đối kiểu unix",
			buoc(Step{ID: "a", Artifact: map[string]string{"x": "/etc/passwd"}}),
			"TƯƠNG ĐỐI",
		},
		{
			"đường dẫn tuyệt đối kiểu windows",
			buoc(Step{ID: "a", Artifact: map[string]string{"x": `C:\Windows\x.dll`}}),
			"TƯƠNG ĐỐI",
		},
		{
			"đi ngược ra ngoài",
			buoc(Step{ID: "a", Artifact: map[string]string{"x": "../../.ssh/id_rsa"}}),
			"ra ngoài",
		},
		{
			"tên artifact có ký tự lạ",
			buoc(Step{ID: "a", Artifact: map[string]string{"Bao Cao": "x.md"}}),
			"chỉ được dùng chữ thường",
		},
		{
			"approve không ghi được file",
			buoc(Step{ID: "a", Type: TypeApprove, Message: "ok",
				Artifact: map[string]string{"x": "x.md"}}),
			"không có đường nào ghi ra file",
		},
		{
			"notify không ghi được file",
			buoc(Step{ID: "a", Type: TypeNotify, Message: "ok",
				Artifact: map[string]string{"x": "x.md"}}),
			"không có đường nào ghi ra file",
		},
		{
			"foreach + artifact",
			buoc(Step{ID: "a", ForEach: "vars.ds", Run: []string{"go", "version"},
				Artifact: map[string]string{"x": "x.md"}}),
			"foreach",
		},
		{
			"đọc artifact không ai sản xuất",
			buoc(Step{ID: "a", Run: []string{"cat", "{{artifacts.khong-co}}"}}),
			"không bước nào khai artifact",
		},
		{
			"đọc artifact của chính mình",
			buoc(Step{ID: "a", Run: []string{"cat", "{{artifacts.x}}"},
				Artifact: map[string]string{"x": "x.md"}}),
			"CHÍNH bước này",
		},
	}

	for _, c := range cases {
		t.Run(c.ten, func(t *testing.T) {
			ps := Validate(c.f)
			if !loiChua(ps, c.chua) {
				t.Fatalf("phải có LỖI chứa %q, được: %v", c.chua, ps)
			}
		})
	}
}

func TestValidateChanTrungTenArtifact(t *testing.T) {
	f := Flow{Name: "t", Steps: []Step{
		{ID: "a", Type: TypeShell, Run: []string{"go", "version"},
			Artifact: map[string]string{"kq": "a.md"}},
		{ID: "b", Type: TypeShell, Run: []string{"go", "version"},
			Artifact: map[string]string{"kq": "b.md"}},
	}}
	if !loiChua(Validate(f), "đã được bước") {
		t.Fatalf("hai bước khai trùng tên artifact phải là LỖI, được: %v", Validate(f))
	}
}

// Đọc artifact của một bước chạy CÙNG ĐỢT chỉ là CẢNH BÁO, không chặn — cùng
// luật với `doc_duoc`: người viết flow có thể khai trước rồi mới nối `needs`.
func TestValidateCanhBaoArtifactCuaBuocChayCungDot(t *testing.T) {
	f := Flow{Name: "t", Steps: []Step{
		{ID: "a", Type: TypeShell, Run: []string{"go", "version"},
			Artifact: map[string]string{"kq": "a.md"}},
		{ID: "b", Type: TypeShell, Run: []string{"cat", "{{artifacts.kq}}"}}, // quên needs
	}}
	ps := Validate(f)
	if coLoi(ps) {
		t.Fatalf("không được chặn, chỉ cảnh báo: %v", ps)
	}
	if !canhChua(ps, "cùng đợt hoặc sau") {
		t.Fatalf("phải cảnh báo thứ tự đợt, được: %v", ps)
	}
}

func TestValidateCanhBaoArtifactDirONhamCho(t *testing.T) {
	f := Flow{Name: "t", Steps: []Step{
		{ID: "a", Type: TypeShell, Run: []string{"touch", "{{artifact_dir}}/x"}},
	}}
	ps := Validate(f)
	if coLoi(ps) {
		t.Fatalf("không được chặn: %v", ps)
	}
	if !canhChua(ps, "không khai `artifact` nào") {
		t.Fatalf("phải cảnh báo dùng {{artifact_dir}} ở bước không khai artifact, được: %v", ps)
	}
}

// Flow có artifact phải đi qua Save/Load mà không mất trường nào — flows.toml là
// nguồn sự thật, và một trường bốc hơi lúc lưu là một tính năng bốc hơi.
func TestArtifactSongSotQuaSaveVaLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := t.TempDir()

	goc := Flow{Name: "luu-thu", Steps: []Step{
		{ID: "viet", Type: TypeShell, Run: []string{"go", "version"},
			Artifact: map[string]string{"bao-cao": "bao-cao.md", "so-lieu": "so/lieu.json"}},
		{ID: "doc", Type: TypeShell, Needs: []string{"viet"},
			Run: []string{"cat", "{{artifacts.bao-cao}}"}},
	}}
	if _, err := Save(dir, goc); err != nil {
		t.Fatal(err)
	}
	flows, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	lai, ok := flows["luu-thu"]
	if !ok {
		t.Fatal("mất cả flow sau khi lưu")
	}
	got := lai.Steps[0].Artifact
	if len(got) != 2 || got["bao-cao"] != "bao-cao.md" || got["so-lieu"] != "so/lieu.json" {
		t.Fatalf("artifact không sống sót qua flows.toml: %+v", got)
	}
}

func orElse(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
