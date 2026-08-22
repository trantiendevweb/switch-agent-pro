package aiapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// ĐƯỜNG ẢNH PHẢI ĐI HẾT, TỪ FILE TRÊN ĐĨA TỚI THÂN JSON TRÊN DÂY.
//
// Vì sao bài này tồn tại, và vì sao nó KHÔNG thừa dù bảng năng lực đã in ✓:
//
// Bảng `sagent nang-luc-api` dò phía dự án bằng phép đo trong `nangluc.go`. Phép
// đo ô `dau-vao-anh` nay đã mạnh hơn phiên bản reflect cũ — nó DỰNG THẬT một
// tin nhắn có ảnh rồi đọc lại thân JSON — nhưng nó dựng lấy tin nhắn CỦA CHÍNH
// NÓ. Câu nó vẫn KHÔNG trả lời được là: *"`GoiKem` có nhét ảnh của người dùng
// vào đó không"*. Xoá dòng `phan = append(phan, PhanAnh(a.DataURL()))` trong
// goikem.go thì bảng vẫn in ✓, mọi thứ vẫn biên dịch, và người dùng gửi ảnh đi
// nhận về "tôi không thấy ảnh nào".
//
// Đó đúng hình dạng lỗi đã cắn dự án này năm lần trong ngày 22/08 — thứ gì đó
// "có" ở mọi tầng trừ tầng cuối. Nên bài này đi HẾT đường:
//
//	file .png trên đĩa → DocAnh → GoiKem → thân JSON THẬT (nhà cung cấp giả đọc lại)
//	                                     → KetQua.NoiDung
//
// và nó canh luôn LUẬT HỢP NHẤT `Content` + `Phan` khai ở đầu anh.go: câu hỏi
// KHÔNG được bị nuốt khi có ảnh đi kèm.
func TestAnhDiHetDuongToiThanJSON(t *testing.T) {
	const cauHoi = "Ảnh này màu gì?"
	duong := anhDoTam(t, 32, 32)

	var thanNhanDuoc []byte
	var soLuot int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&soLuot, 1)
		thanNhanDuoc, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"role":"assistant",` +
			`"content":"đỏ"}}],"usage":{"prompt_tokens":700,"completion_tokens":3,` +
			`"total_tokens":703}}`))
	}))
	defer srv.Close()

	a, err := DocAnh(duong)
	if err != nil {
		t.Fatalf("DocAnh: %v", err)
	}
	if a.Rong != 32 || a.Cao != 32 {
		t.Fatalf("DocAnh đọc kích thước = %d×%d, chờ 32×32 — không đọc được kích thước "+
			"thì không cảnh báo được ngưỡng %d điểm ảnh", a.Rong, a.Cao, nguongDiemAnhDaDo)
	}
	if a.Loai != "image/png" {
		t.Errorf("kiểu ảnh = %q, chờ image/png", a.Loai)
	}

	kq, err := GoiKem(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, cauHoi, TuyChonGoi{Anh: []Anh{a}})
	if err != nil {
		t.Fatalf("GoiKem: %v", err)
	}

	// ---- CHIỀU ĐI: ảnh có THẬT SỰ lên dây không ----
	var than struct {
		Messages []struct {
			Role    string `json:"role"`
			Content []struct {
				Loai string `json:"type"`
				Chu  string `json:"text"`
				Anh  *struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(thanNhanDuoc, &than); err != nil {
		t.Fatalf("thân gửi đi không có `content` dạng MẢNG (%v) — giao thức đòi mảng "+
			"{type, image_url} mới gửi được ảnh. Thân thật: %s", err, thanNhanDuoc)
	}
	if len(than.Messages) != 1 {
		t.Fatalf("thân gửi đi mang %d tin nhắn, chờ 1", len(than.Messages))
	}
	phan := than.Messages[0].Content
	if len(phan) != 2 {
		t.Fatalf("`content` mang %d mẩu, chờ 2 (1 text + 1 image_url). Thân thật: %s",
			len(phan), thanNhanDuoc)
	}

	// LUẬT HỢP NHẤT: câu hỏi vào mẩu ĐẦU, ảnh theo sau. Nuốt câu hỏi thì lượt
	// gọi vẫn chạy, vẫn tính tiền, và model nhận một tấm ảnh không kèm câu hỏi.
	if phan[0].Loai != "text" || phan[0].Chu != cauHoi {
		t.Errorf("mẩu đầu = {%q,%q}, chờ {text,%q} — `Content` bị nuốt khi có ảnh đi kèm, "+
			"trái luật hợp nhất khai ở đầu anh.go", phan[0].Loai, phan[0].Chu, cauHoi)
	}
	if phan[1].Loai != "image_url" || phan[1].Anh == nil {
		t.Fatalf("mẩu thứ hai = %q, chờ image_url — ẢNH RỤNG trên đường đi, trong khi "+
			"bảng năng lực vẫn in ✓ cho ô `dau-vao-anh` vì phép đo tự dựng lấy tin nhắn "+
			"của nó. Thân thật: %s", phan[1].Loai, thanNhanDuoc)
	}
	// Data URL phải mang ĐÚNG byte của file trên đĩa. Base64 sai một chỗ thì
	// nhà cung cấp trả HTTP 400 bằng tiếng Anh về một ảnh người dùng chưa gõ.
	goc, err := os.ReadFile(duong)
	if err != nil {
		t.Fatal(err)
	}
	dau := "data:image/png;base64,"
	if !strings.HasPrefix(phan[1].Anh.URL, dau) {
		t.Fatalf("data URL không có tiền tố %q: %.60q", dau, phan[1].Anh.URL)
	}
	daGiai, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(phan[1].Anh.URL, dau))
	if err != nil {
		t.Fatalf("phần base64 của data URL hỏng: %v", err)
	}
	if !bytes.Equal(daGiai, goc) {
		t.Errorf("byte ảnh trên dây (%d byte) KHÁC file trên đĩa (%d byte) — nhà cung cấp "+
			"sẽ giải mã ra một thứ khác thứ người dùng chỉ vào", len(daGiai), len(goc))
	}

	// ---- CHIỀU VỀ ----
	if kq.NoiDung != "đỏ" {
		t.Errorf("NoiDung = %q, chờ %q", kq.NoiDung, "đỏ")
	}
	if kq.Usage.Tong != 703 {
		t.Errorf("usage = %d, chờ 703 — lượt gửi ảnh vẫn phải đếm được tiền", kq.Usage.Tong)
	}
	if n := atomic.LoadInt32(&soLuot); n != 1 {
		t.Errorf("GoiKem chạm mạng %d lượt, chờ ĐÚNG 1", n)
	}

	// Route này KHÔNG có trong sổ số đo → ô `dau-vao-anh` là `chua-do`. Luật:
	// chưa đo thì CHO ĐI, nhưng phải NÓI RA. Im lặng ở đây là bẹp ba trạng thái
	// thành hai ngay tại chỗ tiêu tiền.
	if len(kq.CanhBaoTruocKhiGui) == 0 {
		t.Error("route chưa ai đo mà lượt gọi đi qua trong IM LẶNG — `chua-do` lại một " +
			"lần nữa bị đọc thành `chạy được`")
	}
}

// Lượt gọi THƯỜNG không được đổi hình dạng `content`.
//
// Vì sao đáng một bài riêng: `Goi` là đường mà cả `internal/api`, CLI và mặt web
// đang đi. `MarshalJSON` mới (aiapi.go) đứng chắn TRƯỚC mọi lượt gọi đó — viết
// hụt một nhánh là mọi lượt hỏi bình thường bỗng gửi `content` dạng mảng, và
// deepseek-v4-flash đã trả HTTP 400 vì những thứ nhỏ hơn thế (đo 22/08, chỉ vì
// `tool_choice=required`). Hỏng kiểu đó không nổ ở đây; nó nổ trên máy người
// dùng, ở MỌI lượt gọi, sau khi đã trả tiền.
func TestGoiThuongVanGuiContentLaChuoiThuan(t *testing.T) {
	var than string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		than = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"content":"4"}}],` +
			`"usage":{"total_tokens":3}}`))
	}))
	defer srv.Close()

	if _, err := Goi(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, "2+2"); err != nil {
		t.Fatalf("Goi: %v", err)
	}
	var kiem struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(than), &kiem); err != nil {
		t.Fatalf("lượt gọi thường KHÔNG còn gửi `content` là chuỗi thuần (%v) — thân: %s",
			err, than)
	}
	if len(kiem.Messages) != 1 || kiem.Messages[0].Content != "2+2" {
		t.Fatalf("prompt không tới nơi nguyên vẹn: %s", than)
	}
	for _, cam := range []string{"image_url", "response_format", "tools"} {
		if strings.Contains(than, cam) {
			t.Errorf("lượt gọi thường mang khoá %q lên dây: %s", cam, than)
		}
	}
}

// CÂU KHÓ THỨ HAI: route ĐÃ ĐO ĐƯỢC là không đọc được ảnh, mà người dùng vẫn gửi.
//
// Luật: chặn ở phía mình, TRƯỚC khi chạm mạng, và lời chặn phải mang đủ ba thứ
// người dùng cần để đi tiếp — bằng chứng đo được, route nào làm được, và cách
// gõ lại. Xem ghi chú đầu goikem.go để biết vì sao chặn chứ không cứ gửi.
func TestRouteDaDoLaKhongDocDuocAnhThiChanTruocKhiChamMang(t *testing.T) {
	var chamMang bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamMang = true
	}))
	defer srv.Close()

	// ĐÚNG base_url + model của `deepseek` để khớp sổ số đo: đo 22/08, HTTP 400
	// "This model does not support image".
	ds := dsThu
	r := Route{Ten: "deepseek", BaseURL: "https://modelapi.vn/v1",
		Model: "deepseek-v4-flash", KeyID: khoaThu(t)}
	a := anhTam(t, 32, 32)

	_, err := GoiKem(context.Background(), r, "ảnh này màu gì", TuyChonGoi{
		Anh: []Anh{a}, DsRoute: ds,
	})
	if err == nil {
		t.Fatal("route đã đo được là KHÔNG đọc được ảnh mà vẫn gửi đi — bảng năng lực " +
			"biết câu trả lời trước khi gọi, biết mà không dùng là bỏ phí một phép đo " +
			"đã tốn tiền thật")
	}
	if chamMang {
		t.Error("lời chặn xảy ra SAU khi đã chạm mạng — một lượt hỏng vẫn có thể bị tính tiền")
	}
	// Chuyển route dự phòng chỉ lặp lại đúng cái sai đó ở một nhà cung cấp khác,
	// và tệ hơn: đường dự phòng đi qua `Goi`, tức là gửi một yêu cầu KHÔNG có ảnh.
	if !LoiNguoiDung(err) {
		t.Errorf("phải là lỗi người dùng để tầng trên không nhảy route dự phòng: %v", err)
	}
	s := err.Error()
	for _, can := range []struct{ chuoi, visao string }{
		{"does not support image", "mất nguyên văn bằng chứng đo được — người đọc phải tin suông"},
		{"grok", "không nói route nào ĐÃ CẤU HÌNH làm được, người dùng phải tự đi tìm"},
		{"--cu-gui", "không chừa cửa gõ tiếp — số đo có ngày, nhà cung cấp thì nâng cấp model"},
		{"nang-luc-api --do", "không chỉ chỗ đo lại"},
	} {
		if !strings.Contains(s, can.chuoi) {
			t.Errorf("lời chặn thiếu %q: %s\n     → %s", can.chuoi, s, can.visao)
		}
	}
}

// `--cu-gui` phải THẬT SỰ gửi đi, và vẫn nói ra là đang đi ngược bảng.
//
// Nếu cờ này chỉ đổi câu chữ mà vẫn chặn thì nó là một cái cửa vẽ trên tường:
// người dùng gõ theo lời mách của chính lời chặn rồi nhận đúng lời chặn đó.
func TestCuGuiThiVuotDuocLoiChanVaVanNoiRa(t *testing.T) {
	var chamMang bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamMang = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"content":"đỏ"}}],` +
			`"usage":{"total_tokens":9}}`))
	}))
	defer srv.Close()

	// base_url của server giả nhưng dùng phép soát của deepseek: gọi thẳng
	// SoatTruocKhiGui để đo đúng chỗ ra quyết định, rồi kiểm cả đường qua GoiKem.
	rDo := Route{Ten: "deepseek", BaseURL: "https://modelapi.vn/v1",
		Model: "deepseek-v4-flash", KeyID: khoaThu(t)}
	canh, err := SoatTruocKhiGui(rDo, NLAPIDauVaoAnh, dsThu, true)
	if err != nil {
		t.Fatalf("--cu-gui mà vẫn chặn: %v", err)
	}
	if len(canh) == 0 {
		t.Fatal("ép gửi mà KHÔNG cảnh báo gì — người dùng đi ngược một phép đo đã có " +
			"mà không được nhắc, rồi sẽ đọc HTTP 400 như một chuyện lạ")
	}
	if !strings.Contains(strings.Join(canh, " "), "nang-luc-api --do") {
		t.Errorf("cảnh báo không mách chỗ ghi lại số đo mới: %v", canh)
	}

	r := Route{Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t)}
	if _, err := GoiKem(context.Background(), r, "ảnh này màu gì", TuyChonGoi{
		Anh: []Anh{anhTam(t, 32, 32)}, CuGui: true,
	}); err != nil {
		t.Fatalf("GoiKem với --cu-gui: %v", err)
	}
	if !chamMang {
		t.Error("--cu-gui mà lượt gọi vẫn không đi đâu")
	}
}

// Ô `chua-do` KHÔNG ĐƯỢC CHẶN. Ba trạng thái là một quy ước về nghĩa, và "chưa
// ai đo" không phải "đã đo là không" — bẹp hai cái đó lại thì thêm một route
// mới vào project.toml là lập tức không gửi ảnh được, mà chẳng ai đo gì cả.
func TestChuaDoThiVanChoGuiNhungPhaiNoiRa(t *testing.T) {
	r := Route{Ten: "moi", BaseURL: "https://nha-cung-cap-la.example/v1",
		Model: "model-chua-ai-do", KeyID: khoaThu(t)}
	canh, err := SoatTruocKhiGui(r, NLAPIDauVaoAnh, nil, false)
	if err != nil {
		t.Fatalf("ô `chua-do` bị CHẶN: %v\n     Đó là bẹp ba trạng thái thành hai ngay "+
			"tại chỗ tiêu tiền", err)
	}
	if len(canh) == 0 {
		t.Fatal("chưa ai đo mà đi qua trong im lặng — người dùng không có cách nào biết " +
			"lượt này là một phép thử")
	}
	s := strings.Join(canh, " ")
	if !strings.Contains(s, "CHƯA") || !strings.Contains(s, "--do moi") {
		t.Errorf("cảnh báo không nói rõ là chưa đo, hoặc không mách chỗ đo: %q", s)
	}
}

// Route LÀM ĐƯỢC thì không cảnh báo gì — im lặng đúng chỗ cũng là một yêu cầu.
// Cảnh báo ở mọi lượt là cách chắc chắn để người ta học cách bỏ qua nó.
func TestRouteLamDuocThiKhongCanhBaoGi(t *testing.T) {
	r := Route{Ten: "grok", BaseURL: "https://modelapi.vn/v1", Model: "grok-4.5", KeyID: "grok"}
	for _, khoa := range []string{NLAPIDauVaoAnh, NLAPIDauRaCoCauTruc} {
		canh, err := SoatTruocKhiGui(r, khoa, dsThu, false)
		if err != nil {
			t.Fatalf("%s: grok đã đo được là LÀM ĐƯỢC (22/08) mà vẫn bị chặn: %v", khoa, err)
		}
		if len(canh) != 0 {
			t.Errorf("%s: route làm được mà vẫn cảnh báo %v — nhiễu ở mọi lượt thì "+
				"cảnh báo thật cũng bị bỏ qua", khoa, canh)
		}
	}
}

// DocAnh phải từ chối TRƯỚC khi chạm mạng, và từ chối theo NỘI DUNG file.
//
// Đuôi tên là thứ người dùng gõ, nội dung là thứ nhà cung cấp đọc. Tin vào đuôi
// là gửi một file text tên .png đi rồi tốn một lượt gọi để nhận HTTP 400.
func TestDocAnhTuChoiFileKhongPhaiAnh(t *testing.T) {
	dir := t.TempDir()
	giaDangPNG := filepath.Join(dir, "khong-phai-anh.png")
	if err := os.WriteFile(giaDangPNG, []byte("đây là chữ, không phải ảnh"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DocAnh(giaDangPNG); err == nil {
		t.Error("file chữ đổi đuôi thành .png mà vẫn nhận — sẽ tốn một lượt gọi để nhận 400")
	} else if !strings.Contains(err.Error(), "NỘI DUNG") {
		t.Errorf("lỗi không nói ra là kiểu đọc theo nội dung chứ không theo đuôi: %v", err)
	}

	rong := filepath.Join(dir, "rong.png")
	if err := os.WriteFile(rong, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DocAnh(rong); err == nil {
		t.Error("ảnh 0 byte mà vẫn nhận")
	}
	if _, err := DocAnh(filepath.Join(dir, "khong-co-that.png")); err == nil {
		t.Error("file không tồn tại mà vẫn nhận")
	}

	// Số thứ tự trong lỗi của DocNhieuAnh: `--anh a --anh b --anh c` mà báo
	// "không đọc được ảnh" trơ trọi thì người dùng phải thử lại từng cái.
	tot := anhDoTam(t, 32, 32)
	_, err := DocNhieuAnh([]string{tot, giaDangPNG})
	if err == nil {
		t.Fatal("DocNhieuAnh bỏ qua file hỏng")
	}
	if !strings.Contains(err.Error(), "thứ 2/2") {
		t.Errorf("lỗi không nói file thứ mấy: %v", err)
	}
}

// Ảnh NHỎ HƠN ngưỡng đã đo thì CẢNH BÁO, không chặn — và câu cảnh báo phải nói
// rõ lượt hỏng sắp tới là vì ảnh nhỏ, không phải vì route không đọc được ảnh.
//
// Bài học có thật, chép từ donangluc.go: bộ đo bản đầu dùng ảnh 8×8 và grok trả
// HTTP 400 "below the minimum of 512 pixels". Suýt nữa thì ghi vào bảng rằng
// grok KHÔNG đọc được ảnh — trong khi thứ vừa đo được là lỗi của người đo.
func TestAnhNhoHonNguongThiCanhBaoChuKhongChan(t *testing.T) {
	nho := anhTam(t, 8, 8)
	if n := nho.SoDiemAnh(); n >= nguongDiemAnhDaDo {
		t.Fatalf("ảnh thử có %d điểm ảnh, phải nhỏ hơn %d thì bài này mới đo được gì",
			n, nguongDiemAnhDaDo)
	}
	canh := canhBaoAnh([]Anh{nho})
	if len(canh) != 1 {
		t.Fatalf("ảnh %d điểm ảnh mà không cảnh báo gì", nho.SoDiemAnh())
	}
	s := canh[0]
	if !strings.Contains(s, "512") || !strings.Contains(s, "KHÔNG phải vì route") {
		t.Errorf("cảnh báo không chỉ đúng nguyên nhân sắp tới: %q", s)
	}
	// Và ảnh đủ lớn thì im lặng.
	if c := canhBaoAnh([]Anh{anhTam(t, 32, 32)}); len(c) != 0 {
		t.Errorf("ảnh 1024 điểm ảnh mà vẫn cảnh báo: %v", c)
	}
}

// Gọi cửa `GoiKem` mà không kèm gì là lỗi NGƯỜI DÙNG, chặn trước khi chạm mạng:
// đi tiếp thì họ nhận một câu trả lời bình thường rồi kết luận nhầm rằng ảnh đã
// gửi mà model không thấy.
func TestGoiKemKhongKemGiThiChanTruocKhiChamMang(t *testing.T) {
	var chamMang bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamMang = true
	}))
	defer srv.Close()
	r := Route{Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t)}
	if _, err := GoiKem(context.Background(), r, "hỏi", TuyChonGoi{}); err == nil {
		t.Error("GoiKem không kèm gì mà vẫn chạy")
	} else if !LoiNguoiDung(err) {
		t.Errorf("phải là lỗi người dùng: %v", err)
	}
	if chamMang {
		t.Error("lỗi của người gọi mà vẫn tốn một lượt gọi")
	}
}

// ---------------------------------------------------------------------------

// anhTam dựng một ảnh PNG toàn đỏ NGAY TRONG BÀI KIỂM.
//
// Sinh tại chỗ chứ không để một file .png trong repo, và đó là một ràng buộc
// chứ không phải sở thích: một file nhị phân trong repo là thứ không ai đọc
// được diff, không ai biết nó còn đúng không, và git trên Windows với
// `core.autocrlf=true` sẽ sửa nội dung nó nếu quên khai `binary` trong
// .gitattributes — đúng cái bẫy đã ghi cho asset vendor của dashboard.
func anhTam(t *testing.T, rong, cao int) Anh {
	t.Helper()
	a, err := DocAnh(anhDoTam(t, rong, cao))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// anhDoTam ghi ảnh PNG toàn đỏ ra thư mục tạm và trả về đường dẫn.
//
// Toàn MỘT MÀU vì cùng lý do của `anhDoThiGiac` (donangluc.go): chỉ nơi THẬT SỰ
// giải mã ảnh mới trả lời được "đỏ", nên bài E2E dưới đây phân biệt được "nhà
// cung cấp nhìn thấy ảnh" với "nhà cung cấp nuốt phần image_url rồi vẫn trả lời
// tử tế".
func anhDoTam(t *testing.T, rong, cao int) string {
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
	return p
}

// BÀI CANH ĐỊNH KỲ — gọi nhà cung cấp THẬT, tốn tiền thật, nên MẶC ĐỊNH BỎ QUA.
// Bật bằng:  SAGENT_E2E_ANH=1 go test ./internal/aiapi/
//
// Theo đúng lệ của `TestE2ESuyLuanThatTuNhaCungCap` và `TestE2EToolThatTuNhaCungCap`,
// và vì cùng một lý do: mọi bài trên đo với nhà cung cấp GIẢ. Chúng khẳng định
// "nếu máy chủ nhận `content` dạng mảng thì ảnh đi hết đường" — đúng và cần,
// nhưng KHÔNG khẳng định máy chủ thật còn nhận hình dạng đó.
//
// Dùng grok-4.5 chứ không phải deepseek: đo 22/08, deepseek trả HTTP 400 "This
// model does not support image" còn grok đọc ĐÚNG màu ảnh 32×32 toàn đỏ và trả
// "đỏ", 761 token. Bài này gửi đúng một ảnh như thế và đòi lại đúng chữ đó — hỏi
// một câu mà chỉ nơi giải mã ảnh mới trả lời được.
func TestE2EAnhThatTuNhaCungCap(t *testing.T) {
	if os.Getenv("SAGENT_E2E_ANH") == "" {
		t.Skip("bỏ qua: tốn tiền thật. Bật bằng SAGENT_E2E_ANH=1")
	}
	a, err := DocAnh(anhDoTam(t, 32, 32))
	if err != nil {
		t.Fatal(err)
	}
	r := Route{Ten: "grok", BaseURL: "https://modelapi.vn/v1", Model: "grok-4.5", KeyID: "grok"}
	kq, err := GoiKem(context.Background(), r, "Ảnh này màu gì? Trả lời đúng một từ tiếng Việt.",
		TuyChonGoi{Anh: []Anh{a}, DsRoute: dsThu})
	if err != nil {
		t.Fatalf("GoiKem thật: %v", err)
	}
	t.Logf("ảnh gửi đi = %s", a.MoTa())
	t.Logf("NoiDung = %q", kq.NoiDung)
	t.Logf("Usage   = vào %d / ra %d / tổng %d", kq.Usage.Vao, kq.Usage.Ra, kq.Usage.Tong)
	if !coTuDo(kq.NoiDung) {
		t.Fatalf("nhà cung cấp KHÔNG đọc ra màu ảnh (trả %q) — hoặc đã nuốt phần "+
			"`image_url`, hoặc đã đổi hình dạng `content`. Bảng `sagent nang-luc-api` "+
			"vẫn in ✓ cho ô này vì phép đo phía dự án chỉ dựng lấy thân JSON của nó. "+
			"Đo lại bằng `sagent nang-luc-api --do grok`.", kq.NoiDung)
	}
}
