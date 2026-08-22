// Bài kiểm cho action "flow.artifacts".
//
// Ba nhóm, theo thứ tự nguy hiểm giảm dần:
//
//  1. THOÁT THƯ MỤC — đây là endpoint đọc file trên một cổng có thể đang phơi
//     ra internet. Nhóm này phải kín trước, mọi thứ khác tính sau.
//  2. TRẦN — file lớn hơn trần thì phải NÓI RA, và phải đọc tiếp được.
//  3. CHE BÍ MẬT — artifact là cửa ra thứ ba của chữ do agent sinh ra.
package api

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// dungLuotCoArtifact dựng một lượt chạy THẬT trong sổ, với flows.toml thật và
// file artifact thật trên đĩa. Trả về runID.
//
// Đi qua sổ + đĩa chứ không giả lập, vì mọi thứ bài này đo đều nằm ở CHỖ NỐI:
// tra sổ ra thư mục dự án, đọc flows.toml ra bản đồ tên, rồi duyệt đĩa.
func dungLuotCoArtifact(t *testing.T, a *API, f flow.Flow, file map[string]string) int64 {
	t.Helper()
	dir := t.TempDir()
	if _, err := flow.Save(dir, f); err != nil {
		t.Fatal(err)
	}
	runID, err := a.db.CreateRun(f.Name, dir, "{}")
	if err != nil {
		t.Fatal(err)
	}
	for rel, noiDung := range file {
		p := filepath.Join(flow.ArtifactRunDir(runID), filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(noiDung), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return runID
}

func flowCoArtifact() flow.Flow {
	return flow.Flow{Name: "co-file", Desc: "flow để lại file", Steps: []flow.Step{
		{ID: "viet", Type: flow.TypeShell, Run: []string{"x"},
			Artifact: map[string]string{"ban-va": "ban-va.diff"}},
		{ID: "doc", Type: flow.TypeShell, Needs: []string{"viet"}, Run: []string{"y"}},
	}}
}

// ---------------------------------------------------------------------------
// 1. THOÁT THƯ MỤC
// ---------------------------------------------------------------------------

// TestKhongDocDuocFileNgoaiThuMucArtifact là bài kiểm quan trọng nhất của cả
// mảnh này.
//
// Endpoint /api/flow/artifact nhận một đường dẫn từ query string và trả về nội
// dung file — đúng hình dạng của một lỗ đọc file tuỳ ý, trên một cổng mà dash
// tự biết có thể không nằm trên loopback (`s.exposed`).
//
// Bài này dựng một file BÍ MẬT THẬT ở ngoài thư mục artifact rồi thử mọi kiểu
// trỏ tới nó. Không kiểu nào được ra một byte.
//
// GỠ PHẦN SỬA RA THÌ ĐỎ Ở ĐÂU: bỏ lời gọi flow.DuongDanArtifactAnToan trong
// FlowArtifactDoc và ghép thẳng filepath.Join(gốc, duong) thì các ca `..` đọc
// được nguyên file bí mật.
func TestKhongDocDuocFileNgoaiThuMucArtifact(t *testing.T) {
	a := moAPI(t)
	runID := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{
		"viet/ban-va.diff": "một bản vá vô hại",
	})

	// File bí mật NGOÀI thư mục artifact, ngay cạnh nó — đúng chỗ thật:
	// ~/.ai-accounts/ là nơi để khoá API và hồ sơ đăng nhập.
	biMat := filepath.Join(filepath.Dir(flow.ArtifactRoot()), "bi-mat.txt")
	if err := os.WriteFile(biMat, []byte("KHOA-API-THAT"), 0o600); err != nil {
		t.Fatal(err)
	}

	xau := []string{
		"../bi-mat.txt",
		"../../bi-mat.txt",
		"viet/../../bi-mat.txt",
		"viet/../../../bi-mat.txt",
		`..\bi-mat.txt`,
		`viet\..\..\bi-mat.txt`,
		"./../bi-mat.txt",
		biMat,                      // tuyệt đối
		"/etc/passwd",              // tuyệt đối kiểu POSIX
		`C:\Windows\win.ini`,       // tuyệt đối kiểu Windows
		`\\may-khac\o-chung\x.txt`, // UNC
		"..",
		"",
	}
	for _, d := range xau {
		nd, err := a.FlowArtifactDoc(runID, d, 0)
		if err == nil {
			t.Errorf("ĐỌC ĐƯỢC %q — endpoint này đang là một lỗ đọc file tuỳ ý", d)
			continue
		}
		if strings.Contains(nd.Chu, "KHOA-API-THAT") {
			t.Errorf("%q: báo lỗi NHƯNG vẫn trả về nội dung file bí mật", d)
		}
		// Thông điệp lỗi không được kể ra đường dẫn thật trên máy chủ: người dò
		// tìm dùng đúng những câu đó để vẽ bản đồ đĩa.
		if strings.Contains(err.Error(), "KHOA-API-THAT") {
			t.Errorf("%q: thông điệp lỗi lộ nội dung file: %v", d, err)
		}
	}

	// Và đường dẫn HỢP LỆ thì vẫn phải đọc được — nếu không thì bài kiểm trên
	// xanh vì hàm từ chối tất cả, chứ không phải vì nó lọc đúng.
	nd, err := a.FlowArtifactDoc(runID, "viet/ban-va.diff", 0)
	if err != nil {
		t.Fatalf("đường dẫn hợp lệ mà không đọc được: %v", err)
	}
	if nd.Chu != "một bản vá vô hại" {
		t.Fatalf("đọc ra %q", nd.Chu)
	}
}

// LIÊN KẾT MỀM — lớp mà hai lớp lọc chuỗi KHÔNG bắt được.
//
// Thư mục artifact là chỗ AGENT GHI VÀO. Một agent, hoặc một bước shell trong
// flow ai đó gửi tới, tạo được một liên kết mềm trỏ ra ngoài mà không cần một
// dấu `..` nào: chuỗi đường dẫn hoàn toàn vô tội, chỉ có ĐĨA mới biết nó đi đâu.
//
// GỠ PHẦN SỬA RA THÌ ĐỎ Ở ĐÂU: bỏ hai lời gọi filepath.EvalSymlinks trong
// DuongDanArtifactAnToan (so thẳng đường dẫn chưa giải) thì bài này đọc được
// nguyên file bí mật.
func TestKhongDocDuocQuaLienKetMem(t *testing.T) {
	a := moAPI(t)
	runID := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{
		"viet/ban-va.diff": "vô hại",
	})
	biMat := filepath.Join(filepath.Dir(flow.ArtifactRoot()), "khoa.txt")
	if err := os.WriteFile(biMat, []byte("KHOA-API-THAT"), 0o600); err != nil {
		t.Fatal(err)
	}

	lien := filepath.Join(flow.ArtifactRunDir(runID), "viet", "tat.txt")
	if err := os.Symlink(biMat, lien); err != nil {
		// Windows không cho tạo symlink nếu chưa bật Developer Mode. Bỏ qua có
		// GHI LÝ DO, chứ không xanh im lặng: một bài kiểm bị bỏ qua mà không ai
		// biết là một bài kiểm không tồn tại.
		t.Skipf("không tạo được liên kết mềm trên %s (cần quyền): %v", runtime.GOOS, err)
	}

	if nd, err := a.FlowArtifactDoc(runID, "viet/tat.txt", 0); err == nil {
		t.Fatalf("ĐỌC ĐƯỢC qua liên kết mềm — chuỗi đường dẫn vô tội nhưng đĩa thì không: %q", nd.Chu)
	}
}

// Đường dẫn của lượt chạy KHÁC cũng là đường dẫn ngoài: `run-5` là tiền tố
// chuỗi của `run-51`, nên một phép so bằng HasPrefix sẽ cho lọt.
func TestKhongDocDuocArtifactCuaLuotChayKhac(t *testing.T) {
	a := moAPI(t)
	mot := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{"viet/ban-va.diff": "của lượt một"})
	hai := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{"viet/ban-va.diff": "CỦA LƯỢT HAI"})

	nd, err := a.FlowArtifactDoc(mot, "viet/ban-va.diff", 0)
	if err != nil {
		t.Fatal(err)
	}
	if nd.Chu != "của lượt một" {
		t.Fatalf("lượt #%d đọc ra %q", mot, nd.Chu)
	}
	// Trỏ chéo sang lượt kia bằng `..` phải bị chặn, dù nó vẫn nằm trong
	// artifacts/ và về mặt "quyền" thì người dùng cũng xem được nếu hỏi đúng.
	// Chặn ở đây vì hàm này hứa MỘT chuyện: chỉ trong thư mục của lượt chạy đó.
	if _, err := a.FlowArtifactDoc(mot, "../run-"+itoa(hai)+"/viet/ban-va.diff", 0); err == nil {
		t.Fatal("đi chéo được sang lượt chạy khác bằng `..`")
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// 2. TRẦN
// ---------------------------------------------------------------------------

// File lớn hơn trần thì phải NÓI RA, và con số phải khớp với đĩa.
//
// Artifact đo được ngày 22/08 là 60.094 byte và không có gì chặn nó lớn hơn.
// Ném cả file vào trình duyệt là cách chắc chắn treo đúng cái tab người ta cần.
//
// GỠ PHẦN SỬA RA THÌ ĐỎ Ở ĐÂU: bỏ khối tính ConLai/BiCat thì `biCat` luôn false
// và mặt web in ra một khúc cụt như thể đó là cả file — kiểu sai tệ nhất mà một
// ô văn bản gây ra.
func TestArtifactLonHonTranThiNOIRAVaDocTiepDuoc(t *testing.T) {
	a := moAPI(t)
	// 2,5 lần trần: đủ để cần BA lần đọc, tức là kiểm được cả khúc giữa chứ
	// không chỉ khúc đầu và khúc cuối.
	dai := TranDocArtifact*2 + TranDocArtifact/2
	than := strings.Repeat("x", dai)
	runID := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{"viet/ban-va.diff": than})

	var gop strings.Builder
	var tu int64
	for vong := 0; vong < 10; vong++ {
		nd, err := a.FlowArtifactDoc(runID, "viet/ban-va.diff", tu)
		if err != nil {
			t.Fatal(err)
		}
		if nd.Byte != int64(dai) {
			t.Fatalf("kích thước THẬT phải là %d, DTO nói %d", dai, nd.Byte)
		}
		if nd.DocByte > TranDocArtifact {
			t.Fatalf("một lần đọc trả về %d byte, vượt trần %d", nd.DocByte, TranDocArtifact)
		}
		gop.WriteString(nd.Chu)
		if !nd.BiCat {
			if nd.ConLai != 0 {
				t.Fatalf("nói không bị cắt mà còn %d byte", nd.ConLai)
			}
			break
		}
		if nd.ConLai != nd.Byte-nd.Tu-nd.DocByte {
			t.Fatalf("ConLai=%d không khớp: byte=%d tu=%d docByte=%d",
				nd.ConLai, nd.Byte, nd.Tu, nd.DocByte)
		}
		tu = nd.Tu + nd.DocByte
	}
	// Ghép mọi khúc lại phải ra ĐÚNG file. Đây là chỗ chứng minh trần là một
	// CỬA SỔ TRƯỢT chứ không phải một bức tường: không mất byte nào, không lặp.
	if gop.Len() != dai {
		t.Fatalf("ghép các khúc lại được %d byte, file thật %d — trần đang LÀM MẤT dữ liệu",
			gop.Len(), dai)
	}
	if gop.String() != than {
		t.Fatal("ghép các khúc lại KHÁC nội dung file")
	}

	// Đọc quá đuôi không phải lỗi: nó là câu "hết rồi", để vòng lặp của người
	// gọi dừng được tự nhiên.
	nd, err := a.FlowArtifactDoc(runID, "viet/ban-va.diff", int64(dai)+100)
	if err != nil {
		t.Fatalf("đọc quá đuôi phải trả khúc rỗng, không phải lỗi: %v", err)
	}
	if nd.Chu != "" || nd.BiCat {
		t.Fatalf("đọc quá đuôi ra %+v", nd)
	}
}

// Cắt theo BYTE chặt đôi ký tự UTF-8 ở mép — chuyện xảy ra thật với tiếng Việt,
// nơi gần như mọi chữ có dấu đều là 2-3 byte. Khúc trả về phải luôn là chữ hợp
// lệ, và ghép lại vẫn phải đủ.
func TestCatGiuaChuTiengVietKhongLamHongChu(t *testing.T) {
	a := moAPI(t)
	// "ế" là 3 byte; lặp cho tới khi vượt trần thì mép cắt gần như chắc chắn
	// rơi vào giữa một ký tự.
	than := strings.Repeat("ế", TranDocArtifact/2)
	runID := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{"viet/ban-va.diff": than})

	var gop strings.Builder
	var tu int64
	for vong := 0; vong < 10; vong++ {
		nd, err := a.FlowArtifactDoc(runID, "viet/ban-va.diff", tu)
		if err != nil {
			t.Fatal(err)
		}
		if strings.ContainsRune(nd.Chu, '\uFFFD') {
			t.Fatalf("khúc từ %d có ký tự hỏng — cắt giữa một chữ tiếng Việt", tu)
		}
		gop.WriteString(nd.Chu)
		if !nd.BiCat {
			break
		}
		tu = nd.Tu + nd.DocByte
	}
	if gop.String() != than {
		t.Fatalf("ghép lại được %d byte, file thật %d", gop.Len(), len(than))
	}
}

// ---------------------------------------------------------------------------
// 3. CHE BÍ MẬT
// ---------------------------------------------------------------------------

// Artifact là CỬA RA THỨ BA của chữ do agent sinh ra — hai cửa kia là nhật ký
// phiên, và redaction_test.go canh chúng.
//
// Một agent viết báo cáo có dán khoá API vào là chuyện đã xảy ra ở nhật ký;
// không có lý do gì nó không xảy ra ở artifact. Khác biệt duy nhất: artifact đi
// thẳng ra cổng HTTP mà không qua một tầng nào khác.
//
// GỠ PHẦN SỬA RA THÌ ĐỎ Ở ĐÂU: bỏ redaction.Che trong FlowArtifactDoc.
func TestNoiDungArtifactDuocCheBiMat(t *testing.T) {
	a := moAPI(t)
	khoa := biMatGia()
	than := "Báo cáo\n\nkhoá là " + khoa + ", thư gửi nguoi-that@vidu.com\n"
	runID := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{"viet/ban-va.diff": than})

	nd, err := a.FlowArtifactDoc(runID, "viet/ban-va.diff", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, khong := range []string{khoa, "nguoi-that@vidu.com"} {
		if strings.Contains(nd.Chu, khong) {
			t.Errorf("bí mật %q ra thẳng client qua artifact", khong)
		}
	}
	// Và phần chữ thật vẫn còn — che mà nuốt cả báo cáo thì tính năng vô dụng.
	if !strings.Contains(nd.Chu, "Báo cáo") {
		t.Fatalf("che xong mất luôn nội dung: %q", nd.Chu)
	}
}

// ---------------------------------------------------------------------------
// 4. BẢNG LIỆT KÊ
// ---------------------------------------------------------------------------

func TestLietKeArtifactNoiRoTenBuocKichThuoc(t *testing.T) {
	a := moAPI(t)
	runID := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{
		"viet/ban-va.diff": "12345",
		// File có trên đĩa mà KHÔNG bước nào khai. Hợp lệ, và phải hiện ra:
		// giấu nó đi thì bảng nói thiếu đúng chỗ không ai ngờ tới.
		"viet/rac.txt": "rác",
	})

	kho, err := a.FlowArtifacts(runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(kho.File) != 2 {
		t.Fatalf("liệt kê %d file, muốn 2: %+v", len(kho.File), kho.File)
	}
	if kho.Flow != "co-file" {
		t.Errorf("tên flow = %q", kho.Flow)
	}
	if kho.TongByte != int64(len("12345")+len("rác")) {
		t.Errorf("tổng byte = %d", kho.TongByte)
	}

	var thay, chuaKhai *FileArtifact
	for i := range kho.File {
		switch kho.File[i].Duong {
		case "viet/ban-va.diff":
			thay = &kho.File[i]
		case "viet/rac.txt":
			chuaKhai = &kho.File[i]
		}
	}
	if thay == nil || chuaKhai == nil {
		t.Fatalf("thiếu file trong bảng: %+v", kho.File)
	}
	if thay.Ten != "ban-va" {
		t.Errorf("file khai trong flows.toml phải mang TÊN của nó, được %q", thay.Ten)
	}
	if thay.Buoc != "viet" {
		t.Errorf("bước sản xuất = %q, muốn viet", thay.Buoc)
	}
	if thay.Byte != 5 {
		t.Errorf("kích thước = %d, muốn 5", thay.Byte)
	}
	if chuaKhai.Ten != "" {
		t.Errorf("file không bước nào khai phải có TÊN RỖNG, được %q — không được bịa tên",
			chuaKhai.Ten)
	}
}

// File nhị phân phải bị đánh dấu Ở BẢNG LIỆT KÊ, không đợi tới lúc bấm mở:
// người ta bấm vào một file 40 MB rồi mới biết nó là ảnh thì đã tốn một vòng.
func TestFileNhiPhanBiDanhDauTuBangLietKe(t *testing.T) {
	a := moAPI(t)
	runID := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{
		"viet/ban-va.diff": "chữ bình thường",
	})
	// Ghi thêm một file có byte NUL — dấu hiệu chắc chắn nhất của file nhị phân.
	p := filepath.Join(flow.ArtifactRunDir(runID), "viet", "anh.png")
	if err := os.WriteFile(p, []byte{0x89, 'P', 'N', 'G', 0x00, 0x1A, 0x0A}, 0o644); err != nil {
		t.Fatal(err)
	}

	kho, err := a.FlowArtifacts(runID)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range kho.File {
		muon := f.Duong == "viet/anh.png"
		if f.NhiPhan != muon {
			t.Errorf("%s: nhiPhan=%v, muốn %v", f.Duong, f.NhiPhan, muon)
		}
	}
	// Và đọc nó thì KHÔNG ném byte thô ra client.
	nd, err := a.FlowArtifactDoc(runID, "viet/anh.png", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !nd.NhiPhan || nd.Chu != "" {
		t.Fatalf("file nhị phân phải trả về Chu rỗng: %+v", nd)
	}
}

// Lượt chạy không có trong sổ thì nói THẲNG, đừng trả bảng rỗng: bảng rỗng đọc
// là "chưa có artifact", một câu trả lời khác hẳn.
func TestLuotChayKhongCoTrongSoThiNoiThang(t *testing.T) {
	a := moAPI(t)
	if _, err := a.FlowArtifacts(99999); err == nil {
		t.Fatal("lượt chạy không tồn tại mà trả về bảng bình thường")
	}
}

// Lượt chạy CÓ trong sổ nhưng chưa để lại file nào: bảng RỖNG, không phải lỗi.
// Đây là ca thường gặp nhất — hầu hết flow không khai `artifact` nào.
func TestLuotChayKhongDeLaiFileNaoThiBangRONG(t *testing.T) {
	a := moAPI(t)
	runID := dungLuotCoArtifact(t, a, flowCoArtifact(), nil)
	kho, err := a.FlowArtifacts(runID)
	if err != nil {
		t.Fatalf("lượt chạy không để lại file nào KHÔNG phải lỗi: %v", err)
	}
	if len(kho.File) != 0 {
		t.Fatalf("bảng phải rỗng, được %+v", kho.File)
	}
	if kho.ThieuDinhNghia {
		t.Error("flow vẫn đọc được mà báo là thiếu định nghĩa")
	}
}

// Định nghĩa flow mất rồi thì bảng VẪN ra, chỉ là không có tên — và phải NÓI RA
// chuyện đó. Một cột trống vì "không bước nào khai" khác hẳn một cột trống vì
// "không tra được"; trộn hai chuyện đó lại là để người đọc tự kết luận sai.
func TestMatDinhNghiaFlowThiVanLietKeVaNoiRa(t *testing.T) {
	a := moAPI(t)
	runID := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{"viet/ban-va.diff": "x"})

	run, err := a.db.GetRun(runID)
	if err != nil {
		t.Fatal(err)
	}
	// Xoá đúng file flow.Save đã ghi ra — hỏi chính nó chứ đừng đoán đường dẫn:
	// TargetFile đi ngược lên tìm .sagent/project.toml và ngã về kho chung khi
	// không thấy, nên đoán "<dir>/.sagent/flows.toml" là xoá nhầm chỗ và bài
	// kiểm sẽ xanh vì lý do sai.
	if err := os.Remove(flow.TargetFile(run.Dir)); err != nil {
		t.Fatal(err)
	}

	kho, err := a.FlowArtifacts(runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(kho.File) != 1 {
		t.Fatalf("mất định nghĩa mà cũng mất luôn bảng: %+v", kho.File)
	}
	if kho.File[0].Ten != "" {
		t.Errorf("không tra được thì không được đoán tên, được %q", kho.File[0].Ten)
	}
	if !kho.ThieuDinhNghia {
		t.Error("cột TÊN trống vì KHÔNG TRA ĐƯỢC mà bảng không nói ra — người đọc sẽ " +
			"kết luận là không bước nào khai")
	}
}

// Trạng thái lượt chạy không ảnh hưởng: artifact của một lượt đang chạy cũng
// xem được. Đó đúng là lúc người ta cần nhìn nhất.
func TestXemDuocArtifactCuaLuotDangChay(t *testing.T) {
	a := moAPI(t)
	runID := dungLuotCoArtifact(t, a, flowCoArtifact(), map[string]string{"viet/ban-va.diff": "dở dang"})
	if err := a.db.SetRunState(runID, store.RunRunning); err != nil {
		t.Fatal(err)
	}
	kho, err := a.FlowArtifacts(runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(kho.File) != 1 {
		t.Fatalf("lượt đang chạy mà không xem được artifact: %+v", kho)
	}
}
