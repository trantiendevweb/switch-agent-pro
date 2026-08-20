package provider

import "testing"

// Ba bản ghi NDJSON kiểu Claude, mỗi bản một cách chết. Cố ý đi qua
// docKetQuaClaude chứ không dựng KetQua bằng tay: thứ cần chứng minh là kết
// luận đến từ TRƯỜNG CÓ TÊN trong bản ghi thật, không phải từ một struct đã
// được điền sẵn đúng ý người viết test.
const (
	// rate_limit_event mang resetsAt — một timestamp THẬT, không phải câu chữ.
	logHetHanMuc = `{"type":"system","subtype":"init"}
{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1755700000}}
{"type":"result","subtype":"error_during_execution","is_error":true,"api_error_status":"429","result":"","num_turns":3}`

	// permission_denials là một MẢNG, độ dài của nó là số đo.
	logBiChanQuyen = `{"type":"system","subtype":"init"}
{"type":"result","subtype":"success","is_error":false,"result":"","num_turns":2,"permission_denials":[{"tool_name":"Bash"},{"tool_name":"Write"}]}`

	// api_error_status là enum của Claude.
	logLoiAPI = `{"type":"result","subtype":"error_during_execution","is_error":true,"api_error_status":"invalid_api_key","result":"","num_turns":1}`

	// Lượt chạy XONG XUÔI: không có gì bất thường để nói.
	logXongXuoi = `{"type":"result","subtype":"success","is_error":false,"result":"Đã sửa internal/api/api.go và chạy test: PASS","num_turns":5}`

	// Hỏng thật, nhưng không thuộc ba loại đo được (không trả lời gì).
	logHongKieuKhac = `{"type":"result","subtype":"success","is_error":false,"result":"","num_turns":4}`
)

func phanLoaiTuLog(t *testing.T, raw string) (string, string, int64) {
	t.Helper()
	k, ok := docKetQuaClaude(raw)
	if !ok {
		t.Fatalf("bản ghi mẫu phải đọc được, docKetQuaClaude trả false")
	}
	return PhanLoaiChet(k, ok)
}

// (a) SUY ĐÚNG TRẠNG THÁI — ba cách chết, ba tên khác nhau.
func TestPhanLoaiChetSuyDungBaTrangThai(t *testing.T) {
	tt, ly, han := phanLoaiTuLog(t, logHetHanMuc)
	if tt != ChetHanMuc {
		t.Errorf("hết hạn mức: được %q, muốn %q", tt, ChetHanMuc)
	}
	if han != 1755700000 {
		t.Errorf("mốc cấp lại = %d, muốn 1755700000 (đọc từ rate_limit_info.resetsAt)", han)
	}
	if ly == "" {
		t.Error("hết hạn mức mà không nói lý do gì")
	}

	if tt, _, _ := phanLoaiTuLog(t, logBiChanQuyen); tt != ChetChanQuyen {
		t.Errorf("bị chặn quyền: được %q, muốn %q", tt, ChetChanQuyen)
	}
	if tt, _, _ := phanLoaiTuLog(t, logLoiAPI); tt != ChetLoiAPI {
		t.Errorf("lỗi API: được %q, muốn %q", tt, ChetLoiAPI)
	}
}

// Hạn mức xét TRƯỚC lỗi API: bản ghi logHetHanMuc mang CẢ HAI (resetsAt và
// api_error_status "429"). Nếu thứ tự đảo lại thì phiên đó bị gọi là `failed`,
// và người vận hành mất câu duy nhất có ích — "chờ tới lúc mấy giờ".
//
// Test này ĐỎ nếu ai đó xếp nhánh CoLoi && LoiAPI lên trên nhánh HanMucDenLai.
func TestHanMucDungTruocLoiAPIKhiBanGhiCoCaHai(t *testing.T) {
	k, _ := docKetQuaClaude(logHetHanMuc)
	if k.LoiAPI == "" {
		t.Fatal("bản ghi mẫu phải mang CẢ api_error_status thì test này mới có nghĩa")
	}
	if tt, _, _ := PhanLoaiChet(k, true); tt != ChetHanMuc {
		t.Fatalf("bản ghi mang cả resetsAt lẫn api_error_status cho ra %q — "+
			"hạn mức là chẩn đoán cụ thể hơn, phải thắng", tt)
	}
}

// (b) DỮ LIỆU THIẾU THÌ KHÔNG SUY.
//
// Đây là test quan trọng nhất của file. Đoán bừa một trạng thái nghe hợp lý thì
// người vận hành ngồi chờ hạn mức trong khi thật ra token đã hết hạn.
func TestThieuDuLieuThiKhongSuy(t *testing.T) {
	// docDuoc=false: Codex và Cursor khai thẳng DocKetQua trả false. Kể cả khi
	// KetQua tình cờ mang giá trị trông như đo được, không đọc được là không kết luận.
	moi := KetQua{HanMucDenLai: 1755700000, TuChoiSo: 3, CoLoi: true, LoiAPI: "429"}
	if tt, ly, han := PhanLoaiChet(moi, false); tt != "" || ly != "" || han != 0 {
		t.Fatalf("adapter KHÔNG đọc được bản ghi mà vẫn kết luận (%q,%q,%d) — "+
			"phiên đó phải ở lại `lost`", tt, ly, han)
	}
	// Bản ghi rỗng / không phải NDJSON của agent.
	for _, raw := range []string{"", "   ", "no output produced", "{\"type\":\"system\"}"} {
		k, ok := docKetQuaClaude(raw)
		if tt, _, _ := PhanLoaiChet(k, ok); tt != "" {
			t.Fatalf("bản ghi %q cho ra trạng thái %q — không có gì để đo mà vẫn kết luận", raw, tt)
		}
	}
}

// Lượt chạy XONG XUÔI không được mang một trong ba trạng thái hỏng. Tiến trình
// thoát rồi thì phiên vẫn rời sổ "đang chạy", nhưng gán cho nó `failed` là vu oan.
func TestChayXongXuoiKhongBiGanTrangThaiHong(t *testing.T) {
	if tt, _, _ := phanLoaiTuLog(t, logXongXuoi); tt != "" {
		t.Fatalf("lượt chạy thành công bị gán %q", tt)
	}
}

// Hỏng theo kiểu KHÁC (không trả lời gì) — đo được là hỏng, nhưng ba trạng thái
// kia không cái nào tả đúng. Không được nhét bừa vào cái gần nhất.
func TestHongKieuKhacKhongBiNhetVaoBaTrangThai(t *testing.T) {
	k, ok := docKetQuaClaude(logHongKieuKhac)
	if k.Hong() == "" {
		t.Fatal("bản ghi mẫu phải là một lượt HỎNG thì test này mới có nghĩa")
	}
	if tt, _, _ := PhanLoaiChet(k, ok); tt != "" {
		t.Fatalf("lượt hỏng-không-trả-lời-gì bị gán %q — nó không phải hạn mức, "+
			"không phải chặn quyền, không phải lỗi API", tt)
	}
}

// Provider CHƯA ĐO ĐƯỢC phải ở lại `lost`, kể cả khi đưa cho nó đúng bản ghi
// của Claude. Đây là chỗ luật "không suy đoán" gặp thực tế: Codex/Cursor khai
// thẳng là không đọc được.
func TestProviderChuaDoDuocThiKhongKetLuan(t *testing.T) {
	for _, ten := range []string{"codex", "cursor"} {
		ad, ok := Get(ten)
		if !ok {
			t.Fatalf("không có provider %s", ten)
		}
		k, doc := ad.DocKetQua(logHetHanMuc)
		if tt, _, _ := PhanLoaiChet(k, doc); tt != "" {
			t.Fatalf("%s chưa đo được cách đọc kết quả mà phiên của nó vẫn bị "+
				"kết luận %q", ten, tt)
		}
	}
}
