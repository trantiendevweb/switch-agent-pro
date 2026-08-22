package aiapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// soDoMauSac là schema dùng chung cho các bài dưới đây — đúng schema mà bộ đo
// đã gửi thật hôm 22/08 (donangluc.go), để bài kiểm và phép đo nói cùng một thứ.
func soDoMauSac() *DangTraLoi {
	return SoDoNghiem("mau_sac", map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"mau": map[string]any{"type": "string"}},
		"required":             []any{"mau"},
		"additionalProperties": false,
	})
}

// ĐƯỜNG JSON SCHEMA PHẢI ĐI HẾT, HAI CHIỀU.
//
// Vì sao bài này tồn tại dù bảng năng lực đã in ✓: phép đo phía dự án
// (nangluc.go) dựng lấy một `yeuCau` CỦA CHÍNH NÓ rồi đọc lại thân JSON. Nó
// KHÔNG trả lời được câu "`GoiKem` có nhét schema của người dùng vào đó không"
// — xoá `DangTraLoi: tc.TraLoiTheoSoDo` trong goikem.go thì bảng vẫn xanh, mọi
// thứ vẫn biên dịch, và người dùng nhận về văn xuôi.
//
// Chiều về cũng phải đi hết, và ở tính năng này chiều về mới là chỗ đáng sợ:
// cách hỏng thường gặp nhất KHÔNG phải HTTP 400 mà là HTTP 200 kèm văn xuôi.
// Xem ba ca (a)/(b)/(c) ở đầu cocautruc.go.
func TestCoCauTrucDiHetDuongToiKetQua(t *testing.T) {
	var thanNhanDuoc []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		thanNhanDuoc, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"role":"assistant",` +
			`"content":"{\"mau\":\"đỏ\"}"}}],"usage":{"prompt_tokens":90,` +
			`"completion_tokens":8,"total_tokens":98}}`))
	}))
	defer srv.Close()

	dang := soDoMauSac()
	kq, err := GoiKem(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, "Màu của lá cờ Việt Nam là gì?", TuyChonGoi{TraLoiTheoSoDo: dang})
	if err != nil {
		t.Fatalf("GoiKem: %v", err)
	}

	// ---- CHIỀU ĐI ----
	var than struct {
		Dang *struct {
			Loai string `json:"type"`
			SoDo *struct {
				Ten    string         `json:"name"`
				Nghiem bool           `json:"strict"`
				So     map[string]any `json:"schema"`
			} `json:"json_schema"`
		} `json:"response_format"`
	}
	if err := json.Unmarshal(thanNhanDuoc, &than); err != nil {
		t.Fatalf("thân gửi đi không đọc được: %v — %s", err, thanNhanDuoc)
	}
	if than.Dang == nil || than.Dang.SoDo == nil {
		t.Fatalf("thân gửi đi KHÔNG mang `response_format.json_schema` — schema rụng "+
			"trên đường đi, trong khi bảng năng lực vẫn in ✓ vì phép đo tự dựng lấy "+
			"yêu cầu của nó. Thân thật: %s", thanNhanDuoc)
	}
	if than.Dang.Loai != "json_schema" {
		t.Errorf("type = %q, chờ %q — gõ sai chuỗi này thì nhà cung cấp bỏ qua cả trường "+
			"và trả văn xuôi, triệu chứng giống hệt \"route không hỗ trợ\"",
			than.Dang.Loai, "json_schema")
	}
	if !than.Dang.SoDo.Nghiem {
		t.Error("`strict` không lên dây — đo 22/08 grok-4.5 nhận strict:true và trả đúng schema")
	}
	if than.Dang.SoDo.Ten != "mau_sac" {
		t.Errorf("tên schema = %q, chờ mau_sac", than.Dang.SoDo.Ten)
	}
	// Schema phải đi NGUYÊN, không bị một lớp kiểu trung gian gọt mất. Rụng
	// `required` là model trả JSON thiếu khoá, và bước sau hỏng lúc chạy.
	if than.Dang.SoDo.So["required"] == nil {
		t.Errorf("schema rụng `required` trên đường đi: %v", than.Dang.SoDo.So)
	}
	if than.Dang.SoDo.So["properties"] == nil {
		t.Errorf("schema rụng `properties` trên đường đi: %v", than.Dang.SoDo.So)
	}

	// ---- CHIỀU VỀ: câu trả lời có được ĐỌC LẠI không ----
	k := DocCoCauTruc(kq, dang)
	if !k.Co {
		t.Fatalf("câu trả lời đúng JSON mà DocCoCauTruc nói không: %s", k.ViSaoKhong)
	}
	if !k.DungDuocNgay() {
		t.Fatalf("JSON đúng schema mà vẫn nói chưa dùng được: thiếu %v", k.Thieu)
	}
	if k.Gia["mau"] != "đỏ" {
		t.Errorf("giá trị đọc ra = %v, chờ mau=\"đỏ\" — đọc được JSON mà mất giá trị thì "+
			"bước sau của flow không có gì để dùng", k.Gia)
	}
	if k.PhaiGoRao {
		t.Error("câu trả lời không có rào ```json mà lại báo phải gỡ rào")
	}
	if kq.Usage.Tong != 98 {
		t.Errorf("usage = %d, chờ 98", kq.Usage.Tong)
	}
}

// CA (c) — HTTP 200 KÈM VĂN XUÔI. Cách hỏng đáng sợ nhất của tính năng này.
//
// Nhà cung cấp NHẬN trường `response_format`, trả 200, rồi trả về một câu tiếng
// Việt tử tế. Không lỗi nào, `Usage` đẹp. Bước sau của flow gọi json.Unmarshal
// và hỏng — cách nơi gây ra lỗi vài bước, sau khi đã tiêu token của cả chuỗi.
//
// Gói này phải NÓI RA ngay tại lượt đó, và nói rõ việc phải làm là đổi route
// chứ không phải sửa prompt.
func TestNhaCungCapNuotResponseFormatThiPhaiNoiRa(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"content":` +
			`"Lá cờ Việt Nam có nền màu đỏ và ngôi sao vàng năm cánh ở giữa."}}],` +
			`"usage":{"total_tokens":40}}`))
	}))
	defer srv.Close()

	dang := soDoMauSac()
	kq, err := GoiKem(context.Background(), Route{
		Ten: "thu", BaseURL: srv.URL, Model: "m", KeyID: khoaThu(t),
	}, "Màu của lá cờ Việt Nam là gì?", TuyChonGoi{TraLoiTheoSoDo: dang})
	if err != nil {
		t.Fatalf("GoiKem: %v", err)
	}
	k := DocCoCauTruc(kq, dang)
	if k.Co || k.DungDuocNgay() {
		t.Fatal("văn xuôi mà báo là JSON dùng được — bước sau của flow sẽ hỏng, cách " +
			"chỗ gây lỗi vài bước")
	}
	if k.ViSaoKhong == "" {
		t.Fatal("nói KHÔNG mà không nói vì sao — một ô trống không giải thích thì bị " +
			"đọc thành \"model trả lời không hay\"")
	}
	for _, can := range []string{"NUỐT", "response_format", "HTTP 200"} {
		if !strings.Contains(k.ViSaoKhong, can) {
			t.Errorf("lời giải thích thiếu %q: %q", can, k.ViSaoKhong)
		}
	}
	// Nguyên văn câu trả lời phải còn: không có nó thì người đọc không tự kiểm
	// lại được, phải tin suông.
	if !strings.Contains(k.ViSaoKhong, "ngôi sao vàng") {
		t.Errorf("nguyên văn câu trả lời bị nuốt khỏi lời giải thích: %q", k.ViSaoKhong)
	}
	if !strings.Contains(MoTaCoCauTruc(k), "KHÔNG đọc được") {
		t.Errorf("MoTaCoCauTruc nói sai ca: %q", MoTaCoCauTruc(k))
	}
}

// CA (b) — ĐÚNG JSON, SAI HỢP ĐỒNG.
//
// Nhà cung cấp nhận `response_format` nhưng không ép `required`. Gộp ca này với
// ca (c) thành một chữ "hỏng" là mất đúng thứ nói cho biết nên sửa prompt hay
// nên đổi route.
func TestJSONDungNhungThieuKhoaBatBuocThiKeRaTungKhoa(t *testing.T) {
	dang := soDoMauSac()
	k := DocCoCauTruc(KetQua{Route: "thu", NoiDung: `{"ghi_chu":"không rõ"}`}, dang)
	if !k.Co {
		t.Fatal("JSON hợp lệ mà báo không đọc được")
	}
	if len(k.Thieu) != 1 || k.Thieu[0] != "mau" {
		t.Fatalf("Thieu = %v, chờ [mau] — không kể ra khoá nào thiếu thì người dùng "+
			"phải tự so schema với câu trả lời bằng mắt", k.Thieu)
	}
	if k.DungDuocNgay() {
		t.Error("thiếu khoá bắt buộc mà vẫn nói dùng được ngay")
	}
	if s := MoTaCoCauTruc(k); !strings.Contains(s, "mau") || !strings.Contains(s, "required") {
		t.Errorf("MoTaCoCauTruc không nói ra khoá thiếu và nguyên nhân: %q", s)
	}
}

// Rào ```json phải gỡ được, NHƯNG chuyện đã phải gỡ thì không được giấu.
//
// Model bọc rào nghĩa là nó đang làm theo prompt, không theo `response_format`
// — tức nhà cung cấp nhiều khả năng đã nuốt trường đó, và lượt sau với prompt
// khác sẽ hỏng. Gỡ rào rồi im lặng là làm ca (c) trông y hệt ca (a).
func TestGoRaoMaNhungVanNoiLaDaPhaiGoRao(t *testing.T) {
	dang := soDoMauSac()
	k := DocCoCauTruc(KetQua{Route: "thu", NoiDung: "```json\n{\"mau\":\"đỏ\"}\n```"}, dang)
	if !k.Co {
		t.Fatalf("không gỡ được rào ```json: %s", k.ViSaoKhong)
	}
	if k.Gia["mau"] != "đỏ" {
		t.Errorf("gỡ rào xong mất giá trị: %v", k.Gia)
	}
	if !k.PhaiGoRao {
		t.Fatal("đã phải gỡ rào mà không nói ra — ca \"nhà cung cấp nuốt response_format\" " +
			"sẽ trông y hệt ca chạy đúng")
	}
	if !strings.Contains(MoTaCoCauTruc(k), "gỡ rào") {
		t.Errorf("MoTaCoCauTruc giấu chuyện phải gỡ rào: %q", MoTaCoCauTruc(k))
	}
	// Không có rào thì không được báo là có.
	k2 := DocCoCauTruc(KetQua{Route: "thu", NoiDung: `{"mau":"đỏ"}`}, dang)
	if k2.PhaiGoRao {
		t.Error("JSON trần mà báo là phải gỡ rào")
	}
}

// Không có schema thì KHÔNG kết luận là hợp đồng đã được giữ.
//
// `Thieu` rỗng vì không có `required` nào để đối chiếu — khác hẳn "đã đối chiếu
// và không thiếu gì". Nhập nhèm hai cái đó là đúng kiểu bẹp trạng thái mà cả
// bảng năng lực dựng lên để chống.
func TestKhongCoSoDoThiVanDocJSONNhungKhongDoiChieuGi(t *testing.T) {
	k := DocCoCauTruc(KetQua{Route: "thu", NoiDung: `{"gi_do":1}`}, nil)
	if !k.Co {
		t.Fatalf("JSON hợp lệ mà không đọc: %s", k.ViSaoKhong)
	}
	if len(k.Thieu) != 0 {
		t.Errorf("không có schema mà vẫn kể khoá thiếu: %v", k.Thieu)
	}
	// Câu trả lời rỗng là một ca riêng, và nó vẫn tốn tiền.
	k2 := DocCoCauTruc(KetQua{Route: "thu"}, nil)
	if k2.Co {
		t.Error("câu trả lời rỗng mà báo đọc được JSON")
	}
	if !strings.Contains(k2.ViSaoKhong, "tính tiền") {
		t.Errorf("không nhắc lượt rỗng vẫn tính tiền: %q", k2.ViSaoKhong)
	}
}

// Route ĐÃ ĐO ĐƯỢC là không ép được JSON thì chặn TRƯỚC khi chạm mạng — cùng
// luật với ảnh, xem ghi chú đầu goikem.go.
//
// Ca thật, đo 22/08: deepseek-v4-flash trả HTTP 400 "This response_format type
// is unavailable now".
func TestRouteDaDoLaKhongEpDuocJSONThiChanTruocKhiChamMang(t *testing.T) {
	var chamMang bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamMang = true
	}))
	defer srv.Close()

	r := Route{Ten: "deepseek", BaseURL: "https://modelapi.vn/v1",
		Model: "deepseek-v4-flash", KeyID: khoaThu(t)}
	_, err := GoiKem(context.Background(), r, "hỏi", TuyChonGoi{
		TraLoiTheoSoDo: soDoMauSac(), DsRoute: dsThu,
	})
	if err == nil {
		t.Fatal("route đã đo được là KHÔNG ép được JSON mà vẫn gửi đi")
	}
	if chamMang {
		t.Error("chặn SAU khi đã chạm mạng — lượt hỏng vẫn có thể bị tính tiền")
	}
	s := err.Error()
	if !strings.Contains(s, "response_format type is unavailable") {
		t.Errorf("mất nguyên văn bằng chứng đo được: %v", err)
	}
	if !strings.Contains(s, "grok") {
		t.Errorf("không mách route đã cấu hình mà làm được: %v", err)
	}
}

// BÀI CANH ĐỊNH KỲ — gọi nhà cung cấp THẬT, tốn tiền thật, nên MẶC ĐỊNH BỎ QUA.
// Bật bằng:  SAGENT_E2E_JSON=1 go test ./internal/aiapi/
//
// Cùng lệ và cùng lý do với `TestE2ESuyLuanThatTuNhaCungCap`: mọi bài trên đo
// với nhà cung cấp GIẢ, nên chúng không khẳng định được máy chủ thật còn nhận
// `response_format` kiểu json_schema.
//
// Dùng grok-4.5: đo 22/08, deepseek trả HTTP 400 "This response_format type is
// unavailable now" còn grok nhận strict và trả JSON đúng schema {"mau":"đỏ"},
// 980 token.
func TestE2ECoCauTrucThatTuNhaCungCap(t *testing.T) {
	if os.Getenv("SAGENT_E2E_JSON") == "" {
		t.Skip("bỏ qua: tốn tiền thật. Bật bằng SAGENT_E2E_JSON=1")
	}
	dang := soDoMauSac()
	r := Route{Ten: "grok", BaseURL: "https://modelapi.vn/v1", Model: "grok-4.5", KeyID: "grok"}
	kq, err := GoiKem(context.Background(), r, "Màu của lá cờ Việt Nam là gì?",
		TuyChonGoi{TraLoiTheoSoDo: dang, DsRoute: dsThu})
	if err != nil {
		t.Fatalf("GoiKem thật: %v", err)
	}
	k := DocCoCauTruc(kq, dang)
	t.Logf("NoiDung = %q", kq.NoiDung)
	t.Logf("khối    = %s", MoTaCoCauTruc(k))
	t.Logf("Usage   = vào %d / ra %d / tổng %d", kq.Usage.Vao, kq.Usage.Ra, kq.Usage.Tong)
	if !k.DungDuocNgay() {
		t.Fatalf("nhà cung cấp KHÔNG còn ép được JSON theo schema: %s. Bảng "+
			"`sagent nang-luc-api` vẫn in ✓ cho ô này vì phép đo phía dự án chỉ dựng "+
			"lấy thân JSON của nó. Đo lại bằng `sagent nang-luc-api --do grok`.",
			MoTaCoCauTruc(k))
	}
}
