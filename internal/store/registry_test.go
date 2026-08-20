package store

// SỔ ĐĂNG KÝ (schema v8) — bốn ca đầu của bộ (a)–(f).
//
// Bốn ca này là bốn cách cái sổ trở nên vô dụng hoặc nguy hiểm trong thực tế,
// không phải bốn cách gọi hàm:
//
//	(a) sổ hở miệng   — bảng mới lỡ mang theo một cột chứa secret
//	(b) sổ hai mặt    — nhận-lại một hồ sơ đè mất "không sở hữu" đã ghi trước đó
//	(c) sổ một chiều  — đối chiếu chỉ thấy phía đĩa, không thấy phía sổ
//	(d) sổ tự vá      — đối chiếu lặng lẽ sửa cho khớp, xoá sạch bằng chứng lệch
//
// (e) và (f) đo phần xoá, nằm ở internal/profile/safedelete_test.go.

import (
	"path/filepath"
	"strings"
	"testing"
)

// ---- ca (a): bảng có thật, và KHÔNG có cột nào chứa secret ----
//
// Luật số 1 của cấu hình là "file cấu hình không bao giờ chứa secret". Sổ trạng
// thái không được là cửa sau của luật đó: state.db nằm ngoài repo, nhưng nó là
// thứ người ta chép nguyên cho nhau khi đi nhờ gỡ lỗi, và `db backup` đẻ thêm
// bản sao rải trên đĩa.
//
// Chốt bằng DANH SÁCH CỘT ĐẦY ĐỦ chứ không phải bằng một bộ lọc từ khoá: bộ lọc
// chỉ chặn những cái tên ai đó nghĩ ra trước, còn danh sách đầy đủ thì mọi cột
// thêm sau này đều phải đi qua đây và phải được viết vào test một cách có ý thức.
func TestCaA_SoV8CoBangVaKhongCoCotSecret(t *testing.T) {
	d := open(t)

	muon := map[string][]string{
		// key_id là TÊN FILE khoá trong ~/.ai-accounts/api-keys, không phải khoá.
		// Đây là ngoại lệ DUY NHẤT được phép có chữ "key" trong tên.
		"routes_so":   {"id", "ten", "base_url", "model", "key_id", "ghi_luc"},
		"profiles_so": {"id", "provider", "account", "dir", "so_tao_ra", "ghi_luc"},
	}
	for bang, cotMuon := range muon {
		cot := cotCuaBang(t, d, bang)
		if len(cot) == 0 {
			t.Fatalf("bảng %s không tồn tại — migration v8 chưa chạy", bang)
		}
		if strings.Join(cot, ",") != strings.Join(cotMuon, ",") {
			t.Fatalf("bảng %s có cột %v, muốn %v", bang, cot, cotMuon)
		}
		for _, c := range cot {
			if c == "key_id" {
				continue
			}
			for _, cam := range []string{"key", "secret", "token", "password", "prompt", "credential"} {
				if strings.Contains(c, cam) {
					t.Fatalf("bảng %s có cột %q — sổ trạng thái KHÔNG được chứa secret", bang, c)
				}
			}
		}
	}
}

func cotCuaBang(t *testing.T, d *DB, bang string) []string {
	t.Helper()
	rows, err := d.db.Query(`SELECT name FROM pragma_table_info(?) ORDER BY cid`, bang)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		out = append(out, n)
	}
	return out
}

// ---- ca (b): nhận-lại KHÔNG được đè lên dòng đã có ----
//
// Đây là chỗ cái sổ hoặc có tác dụng, hoặc không có gì cả.
//
// `xoa` nhận hồ sơ chưa có trong sổ vào theo bằng chứng trên đĩa (hồ sơ tạo
// trước v8 thì sổ không biết). Nếu bước nhận đó ĐÈ lên dòng cũ thì bất cứ thứ gì
// từng bị đánh dấu "sổ không sở hữu" sẽ tự được cấp quyền sở hữu ở lần chạy sau
// — và xoá-an-toàn quay về đúng xoá-đệ-quy, không ai thấy gì.
func TestCaB_NhanLaiKhongDuocDeLenDongDaCo(t *testing.T) {
	d := open(t)
	dir := filepath.Join("kho", "claude", "phu")

	// Sổ đã ghi: biết hồ sơ này, nhưng KHÔNG nhận sở hữu.
	if err := d.GhiHoSo(HoSo{Provider: "claude", Account: "phu", Dir: dir, SoTaoRa: false}); err != nil {
		t.Fatal(err)
	}
	moi, err := d.NhanHoSo(HoSo{Provider: "claude", Account: "phu", Dir: dir, SoTaoRa: true})
	if err != nil {
		t.Fatal(err)
	}
	if moi {
		t.Fatal("NhanHoSo báo vừa thêm dòng mới, nhưng sổ đã có dòng đó rồi")
	}
	co, err := d.SoHuu(dir)
	if err != nil {
		t.Fatal(err)
	}
	if co {
		t.Fatal("NHẬN-LẠI ĐÃ ĐÈ MẤT \"không sở hữu\" — xoá-an-toàn coi như không có")
	}

	// Chiều ngược lại: hồ sơ sổ chưa biết thì phải nhận được, và nhận đúng một dòng.
	moi, err = d.NhanHoSo(HoSo{Provider: "codex", Account: "main", Dir: "d2", SoTaoRa: true})
	if err != nil || !moi {
		t.Fatalf("hồ sơ mới phải được nhận vào sổ: moi=%v err=%v", moi, err)
	}
	ds, err := d.HoSoTrongSo()
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 {
		t.Fatalf("sổ có %d dòng, muốn 2 — nhận-lại đẻ thêm dòng trùng?", len(ds))
	}

	// `GhiHoSo` thì ĐƯỢC đè: đó là đường của `them`, nơi sagent vừa tự tay dựng
	// thư mục nên nó biết chắc mình sở hữu.
	if err := d.GhiHoSo(HoSo{Provider: "claude", Account: "phu", Dir: dir, SoTaoRa: true}); err != nil {
		t.Fatal(err)
	}
	if co, _ := d.SoHuu(dir); !co {
		t.Fatal("GhiHoSo không cập nhật được quyền sở hữu")
	}
}

// ---- ca (c): đối chiếu phải nhìn ĐỦ HAI CHIỀU ----
//
// Một hàm chỉ duyệt danh sách trên đĩa rồi tra sổ vẫn "chạy đúng" với mọi hồ sơ
// bình thường — nó chỉ mù đúng một chỗ: hồ sơ SỔ CÓ MÀ ĐĨA KHÔNG. Mà đó lại là
// ca đáng báo nhất: thư mục hồ sơ biến mất sau lưng sagent (dọn tay, OneDrive
// đồng bộ thiếu, ổ mạng rớt) thì token đi theo, và hiện không có gì nói ra.
func TestCaC_DoiChieuDuHaiChieu(t *testing.T) {
	d := open(t)
	nap := []HoSo{
		{Provider: "claude", Account: "khop", Dir: "/kho/claude/khop", SoTaoRa: true},
		{Provider: "claude", Account: "mat-tren-dia", Dir: "/kho/claude/mat", SoTaoRa: true},
		{Provider: "codex", Account: "lech", Dir: "/kho/codex/cho-cu", SoTaoRa: true},
	}
	for _, h := range nap {
		if err := d.GhiHoSo(h); err != nil {
			t.Fatal(err)
		}
	}
	dia := []TrenDia{
		{Provider: "claude", Account: "khop", Dir: "/kho/claude/khop"},
		{Provider: "codex", Account: "lech", Dir: "/kho/codex/cho-moi"},
		{Provider: "codex", Account: "la-hoac-cu", Dir: "/kho/codex/la"},
	}

	ds, err := d.DoiChieuHoSo(dia)
	if err != nil {
		t.Fatal(err)
	}
	muon := map[string]string{
		"claude:khop":         SoKhop,
		"claude:mat-tren-dia": SoThieuDia,
		"codex:lech":          SoLechDuong,
		"codex:la-hoac-cu":    SoThieuSo,
	}
	if len(ds) != len(muon) {
		t.Fatalf("đối chiếu ra %d dòng, muốn %d: %+v", len(ds), len(muon), ds)
	}
	for _, m := range ds {
		k := m.Provider + ":" + m.Account
		if m.TrangThai != muon[k] {
			t.Errorf("%s: trạng thái %q, muốn %q", k, m.TrangThai, muon[k])
		}
	}

	// Dòng "lệch đường dẫn" phải mang CẢ HAI đường dẫn: chỉ có một thì người đọc
	// không biết bên nào sai, và đó đúng là câu hỏi duy nhất họ cần trả lời.
	for _, m := range ds {
		if m.TrangThai != SoLechDuong {
			continue
		}
		if m.DuongSo != "/kho/codex/cho-cu" || m.DuongDia != "/kho/codex/cho-moi" {
			t.Fatalf("dòng lệch thiếu đường dẫn: sổ=%q đĩa=%q", m.DuongSo, m.DuongDia)
		}
	}

	// Thứ tự phải ổn định (provider rồi account) — người ta so hai lần chạy bằng
	// mắt, thứ tự nhảy là tự tạo báo động giả.
	var truoc string
	for _, m := range ds {
		k := m.Provider + ":" + m.Account
		if truoc != "" && k < truoc {
			t.Fatalf("thứ tự không ổn định: %q đứng sau %q", k, truoc)
		}
		truoc = k
	}
}

// ---- ca (d): đối chiếu CHỈ ĐỌC, không tự vá sổ ----
//
// Cám dỗ rất lớn: thấy "đĩa có, sổ không" thì thêm luôn dòng cho gọn. Làm vậy
// thì lần nhìn ĐẦU TIÊN đã xoá sạch bằng chứng, mọi lần sau đều báo "khớp", và
// một thư mục lạ ai đó thả vào kho được cấp quyền sở hữu chỉ vì có người gõ lệnh
// xem bảng.
func TestCaD_DoiChieuKhongTuVaSo(t *testing.T) {
	d := open(t)
	dia := []TrenDia{{Provider: "claude", Account: "la", Dir: "/kho/claude/la"}}

	if _, err := d.DoiChieuHoSo(dia); err != nil {
		t.Fatal(err)
	}
	ds, err := d.HoSoTrongSo()
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 0 {
		t.Fatalf("đối chiếu đã GHI vào sổ %d dòng — nó phải chỉ đọc: %+v", len(ds), ds)
	}
	// Nhìn lần thứ hai vẫn phải thấy đúng chỗ lệch đó.
	lai, err := d.DoiChieuHoSo(dia)
	if err != nil {
		t.Fatal(err)
	}
	if len(lai) != 1 || lai[0].TrangThai != SoThieuSo {
		t.Fatalf("lần nhìn thứ hai mất dấu lệch: %+v", lai)
	}
	if lai[0].SoTaoRa {
		t.Fatal("hồ sơ sổ KHÔNG biết mà lại được coi là sagent sở hữu")
	}
}

// Sổ route: hai chiều lệch nói hai chuyện khác nhau, và cả hai đều là chuyện
// tiền. Đi kèm ca (a) — cùng bảng, cùng luật "không chứa secret".
func TestSoRouteDoiChieuVoiCauHinh(t *testing.T) {
	d := open(t)
	// Đã gọi thật qua hai route.
	for _, r := range []Route{
		{Ten: "grok", BaseURL: "https://a/v1", Model: "grok-4.5", KeyID: "grok"},
		{Ten: "cu", BaseURL: "https://cu/v1", Model: "m-cu", KeyID: "cu"},
	} {
		if err := d.GhiRoute(r); err != nil {
			t.Fatal(err)
		}
	}
	// Cấu hình hôm nay: grok đổi base_url, "cu" đã gỡ, "moi" vừa khai chưa gọi.
	cauHinh := []Route{
		{Ten: "grok", BaseURL: "https://b/v1", Model: "grok-4.5", KeyID: "grok"},
		{Ten: "moi", BaseURL: "https://moi/v1", Model: "m", KeyID: "moi"},
	}
	ds, err := d.DoiChieuRoute(cauHinh)
	if err != nil {
		t.Fatal(err)
	}
	muon := map[string]string{"cu": SoThieuDia, "grok": SoLechDuong, "moi": SoThieuSo}
	if len(ds) != 3 {
		t.Fatalf("ra %d dòng, muốn 3: %+v", len(ds), ds)
	}
	for _, m := range ds {
		if m.TrangThai != muon[m.Ten] {
			t.Errorf("route %s: %q, muốn %q", m.Ten, m.TrangThai, muon[m.Ten])
		}
	}
	// Ghi lại cùng tên route thì CẬP NHẬT, không đẻ dòng thứ hai.
	if err := d.GhiRoute(Route{Ten: "grok", BaseURL: "https://b/v1", Model: "x", KeyID: "grok"}); err != nil {
		t.Fatal(err)
	}
	trongSo, err := d.RouteTrongSo()
	if err != nil {
		t.Fatal(err)
	}
	if len(trongSo) != 2 {
		t.Fatalf("sổ route có %d dòng, muốn 2", len(trongSo))
	}
}
