package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/aiapi"
)

// TẦNG CUỐI CÙNG CỦA ĐƯỜNG ẢNH VÀ ĐƯỜNG JSON SCHEMA.
//
// `internal/aiapi/anh_test.go` và `cocautruc_test.go` canh đường từ file trên
// đĩa tới thân JSON trên dây, rồi từ câu trả lời về tới `KetQua`. Nhóm bài này
// canh nốt đoạn cuối: dữ liệu thành CHỮ TRÊN MÀN HÌNH.
//
// Đoạn đó đáng một nhóm bài riêng vì nó là chỗ dự án này đánh rơi năm lần trong
// ngày 22/08. Và cả hai đường ở đây đều có một ca hỏng mà chữ trên màn hình là
// thứ DUY NHẤT phân biệt được:
//
//	ảnh   — "model bảo không thấy ảnh" vs "sagent quên gắn ảnh"
//	schema — "JSON đúng hợp đồng" vs "HTTP 200 kèm văn xuôi"
//
// Bài gọi CHÍNH hàm dựng chữ, không grep mã nguồn: một bài grep vẫn xanh khi
// hàm được gọi nhưng in ra câu sai.

// Ảnh đã gửi phải HIỆN RA, kèm số đo — không có nó thì "nhà cung cấp nuốt ảnh"
// và "sagent quên gắn ảnh" trông giống hệt nhau trên màn hình.
func TestDongAnhGuiDiNoiRaAnhNaoDaThatSuDi(t *testing.T) {
	a := anhThu(t, 32, 32)
	ra := strings.Join(dongAnhGuiDi([]aiapi.Anh{a}), "\n")
	if ra == "" {
		t.Fatal("gửi ảnh đi mà terminal không nói một chữ nào — người dùng không có " +
			"cách nào phân biệt \"nhà cung cấp nuốt ảnh\" với \"sagent quên gắn ảnh\"")
	}
	for _, can := range []string{a.Ten, "image/png", "1024"} {
		if !strings.Contains(ra, can) {
			t.Errorf("thiếu %q trên màn hình:\n%s", can, ra)
		}
	}
	// Không gửi ảnh thì im lặng.
	if len(dongAnhGuiDi(nil)) != 0 {
		t.Error("không có ảnh nào mà vẫn in khối ảnh")
	}
}

// BA CA CỦA ĐẦU RA CÓ CẤU TRÚC PHẢI RA BA CÂU KHÁC NHAU TRÊN MÀN HÌNH.
//
// Ca đáng sợ nhất là ca (c): HTTP 200 kèm văn xuôi. Câu trả lời trông tử tế,
// `Usage` đẹp, không lỗi nào — và bước sau của flow gọi json.Unmarshal rồi
// hỏng, cách chỗ gây lỗi vài bước. Gộp nó với ca (b) thành một chữ "hỏng" là
// mất đúng thứ nói cho biết nên sửa prompt hay nên đổi route.
func TestDongCoCauTrucNoiDungCaTrongBaCa(t *testing.T) {
	dang := aiapi.SoDoNghiem("mau_sac", map[string]any{
		"type":       "object",
		"properties": map[string]any{"mau": map[string]any{"type": "string"}},
		"required":   []any{"mau"},
	})

	// (a) đúng hợp đồng — phải in ra chính JSON mà bước sau sẽ nhận.
	dung := strings.Join(dongCoCauTruc(aiapi.DocCoCauTruc(
		aiapi.KetQua{Route: "grok", NoiDung: `{"mau":"đỏ"}`}, dang)), "\n")
	if !strings.Contains(dung, "mau") || !strings.Contains(dung, "đỏ") {
		t.Errorf("JSON đúng schema mà giá trị không hiện ra:\n%s", dung)
	}

	// (b) đúng JSON, thiếu khoá bắt buộc — phải KỂ TÊN khoá thiếu.
	thieu := strings.Join(dongCoCauTruc(aiapi.DocCoCauTruc(
		aiapi.KetQua{Route: "grok", NoiDung: `{"ghi_chu":"không rõ"}`}, dang)), "\n")
	if !strings.Contains(thieu, "mau") {
		t.Errorf("không kể ra khoá `mau` bị thiếu, người dùng phải tự so schema bằng "+
			"mắt:\n%s", thieu)
	}
	if !strings.Contains(thieu, "KHÔNG dùng được ngay") {
		t.Errorf("thiếu khoá bắt buộc mà màn hình không nói là chưa dùng được:\n%s", thieu)
	}

	// (c) văn xuôi — phải nói ra là nhà cung cấp nuốt `response_format`, và
	// phải mách chỗ tra lại. Đây là chỗ dễ viết "model trả lời không hay" nhất,
	// mà câu đó dẫn người ta đi sửa prompt cho một lỗi phải đổi route.
	vanXuoi := strings.Join(dongCoCauTruc(aiapi.DocCoCauTruc(
		aiapi.KetQua{Route: "grok", NoiDung: "Lá cờ có nền đỏ, sao vàng."}, dang)), "\n")
	if !strings.Contains(vanXuoi, "NUỐT") || !strings.Contains(vanXuoi, "response_format") {
		t.Errorf("không nói ra là nhà cung cấp nuốt response_format:\n%s", vanXuoi)
	}
	if !strings.Contains(vanXuoi, "nang-luc-api") {
		t.Errorf("không mách chỗ tra lại route này có ép được JSON không:\n%s", vanXuoi)
	}

	// Ba ca phải cho ba câu KHÁC NHAU. Trùng nhau thì màn hình đã bẹp chúng lại.
	if dung == thieu || thieu == vanXuoi || dung == vanXuoi {
		t.Error("hai trong ba ca in ra chữ giống hệt nhau — màn hình đang bẹp ba tình " +
			"huống cần ba việc khác nhau thành một")
	}
}

// Cảnh báo soát trước khi gửi phải TỚI MÀN HÌNH, kể cả phần nhiều dòng.
//
// Đây là chỗ ba trạng thái của bảng năng lực đi tới người dùng. Nuốt câu "CHƯA
// ai đo route này" là làm một lượt gọi thử nghiệm trông y hệt một lượt chắc
// chắn — và khi nó hỏng thì không ai biết vì sao.
func TestDongCanhBaoTruocKhiGuiKhongNuotDongNao(t *testing.T) {
	canh := []string{"CHƯA ai đo route \"moi\"\n     dòng thứ hai của cùng một cảnh báo"}
	ra := dongCanhBaoTruocKhiGui(canh)
	if len(ra) != 2 {
		t.Fatalf("cảnh báo 2 dòng in ra %d dòng: %v", len(ra), ra)
	}
	if !strings.Contains(ra[0], "CHƯA") || !strings.Contains(ra[1], "dòng thứ hai") {
		t.Errorf("nuốt mất một phần cảnh báo: %v", ra)
	}
	if len(dongCanhBaoTruocKhiGui(nil)) != 0 {
		t.Error("không có cảnh báo nào mà vẫn in")
	}
}

// Nạp JSON Schema từ file: schema phải đi NGUYÊN, và tên schema có sẵn.
//
// Rụng `required` ở chỗ nạp là model trả JSON thiếu khoá, và hỏng lúc chạy chứ
// không hỏng lúc nạp — cùng lớp lỗi đã ghi cho `docFileTool`.
func TestDocFileSoDoGiuNguyenSchema(t *testing.T) {
	dir := t.TempDir()
	tot := filepath.Join(dir, "mau_sac.json")
	if err := os.WriteFile(tot, []byte(`{"type":"object","properties":`+
		`{"mau":{"type":"string"}},"required":["mau"],"additionalProperties":false}`),
		0o600); err != nil {
		t.Fatal(err)
	}
	dang, err := docFileSoDo(tot)
	if err != nil {
		t.Fatalf("docFileSoDo: %v", err)
	}
	if dang.Loai != "json_schema" || dang.SoDo == nil {
		t.Fatalf("dựng sai khuôn response_format: %+v", dang)
	}
	if !dang.SoDo.Nghiem {
		t.Error("không đặt strict=true — đo 22/08 grok-4.5 nhận strict và trả đúng schema")
	}
	if dang.SoDo.Ten != "mau_sac" {
		t.Errorf("tên schema = %q, chờ lấy từ tên file: mau_sac", dang.SoDo.Ten)
	}
	if dang.SoDo.So["required"] == nil || dang.SoDo.So["properties"] == nil {
		t.Errorf("nạp file làm rụng phần schema: %+v", dang.SoDo.So)
	}

	xau := filepath.Join(dir, "xau.json")
	if err := os.WriteFile(xau, []byte(`không phải json`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := docFileSoDo(xau); err == nil {
		t.Error("file không phải JSON mà vẫn nạp")
	} else if !strings.Contains(err.Error(), "required") {
		t.Errorf("báo lỗi không kèm khuôn đúng, người dùng phải đoán: %v", err)
	}
	if _, err := docFileSoDo(filepath.Join(dir, "khong-co-that.json")); err == nil {
		t.Error("file không có mà vẫn nạp được")
	}
}

// `--anh` phải LẶP LẠI ĐƯỢC và giữ đúng THỨ TỰ gõ.
//
// Thứ tự không phải chi tiết vặt: `--anh truoc.png --anh sau.png "cái nào mới
// hơn"` chỉ trả lời đúng khi hai ảnh tới model theo đúng thứ tự người dùng gõ.
// Đảo đi thì câu trả lời vẫn trôi chảy, vẫn tính tiền, và sai ngược hoàn toàn.
func TestStrFlagNhieuGiuDuNhieuGiaTriVaDungThuTu(t *testing.T) {
	vals, con := strFlagNhieu(
		[]string{"--anh", "truoc.png", "grok", "--anh", "sau.png", "cái nào mới hơn"}, "--anh")
	if len(vals) != 2 || vals[0] != "truoc.png" || vals[1] != "sau.png" {
		t.Fatalf("rút cờ lặp lại hỏng: %v — gõ hai lần mà mất một cái thì model nhận "+
			"thiếu ảnh, và câu trả lời vẫn trôi chảy", vals)
	}
	if strings.Join(con, " ") != "grok cái nào mới hơn" {
		t.Errorf("phần còn lại sai: %v", con)
	}
	// Không gõ cờ thì không rút gì, và không được nuốt tham số nào.
	vals, con = strFlagNhieu([]string{"grok", "hỏi gì đó"}, "--anh")
	if len(vals) != 0 || len(con) != 2 {
		t.Errorf("không có cờ mà vẫn động vào tham số: %v / %v", vals, con)
	}
	// Cờ đứng cuối, không có giá trị: không được nhận bừa một chuỗi rỗng.
	vals, _ = strFlagNhieu([]string{"hỏi", "--anh"}, "--anh")
	if len(vals) != 0 {
		t.Errorf("cờ thiếu giá trị mà vẫn nhận: %v", vals)
	}
}

// anhThu dựng một `aiapi.Anh` thật từ ảnh PNG sinh tại chỗ.
//
// Sinh tại chỗ chứ không để file .png trong repo: git trên Windows với
// `core.autocrlf=true` sẽ sửa nội dung file nhị phân nếu quên khai `binary`
// trong .gitattributes — đúng cái bẫy đã ghi cho asset vendor của dashboard.
func anhThu(t *testing.T, rong, cao int) aiapi.Anh {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, rong, cao))
	for y := 0; y < cao; y++ {
		for x := 0; x < rong; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "do.png")
	if err := os.WriteFile(p, b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := aiapi.DocAnh(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
