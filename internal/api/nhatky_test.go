package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/nhatky"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Nhật ký phiên đi qua CẢ ĐƯỜNG THẬT: file trên đĩa → sổ → hợp đồng.
//
// Test ở internal/nhatky chứng minh gói làm đúng việc của nó; test ở đây chứng
// minh nó ĐƯỢC CẮM VÀO hợp đồng. Hai chuyện khác nhau.

// themPhienCoNhatKy ghi một phiên kèm file nhật ký thật (có khối tiêu đề).
func themPhienCoNhatKy(t *testing.T, a *API, than string) (int64, string) {
	t.Helper()
	moc := time.Now()
	p := nhatky.Duong("claude", "tns", 1, moc)
	dau := nhatky.Dau{
		ThoiDiem: moc, Addr: "claude:tns#1", HoSo: `C:\clones\1`,
		Lenh: []string{"-p", "sửa lỗi", "--output-format", "stream-json"},
	}
	if err := nhatky.Tao(p, dau); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(than)
	f.Close()

	id, err := a.db.AddSession(store.Session{
		Provider: "claude", Account: "tns", Clone: 1, Dir: "d",
		PID: 0x7FFFFFF0, Log: p,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id, p
}

// KHỐI TIÊU ĐỀ KHÔNG ĐƯỢC LÀM HỎNG PHÉP ĐỌC KẾT QUẢ.
//
// Đây là ràng buộc dễ vỡ nhất của cả tính năng: nhật ký vừa là thứ NGƯỜI đọc,
// vừa là thứ `phanLoaiPhienChet` đọc để quyết trạng thái phiên. Thêm chữ vào
// đầu file mà làm bộ đọc vấp thì mọi phiên quay về `lost` — tức là đi sửa mù
// loà rồi trả lại đúng cái mù loà cũ, chỉ tốn thêm đĩa.
func TestKhoiTieuDeKhongLamHongPhepDocKetQua(t *testing.T) {
	a := moAPI(t)
	than := `{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1755700000}}
{"type":"result","subtype":"error_during_execution","is_error":true,"api_error_status":"429","result":"","num_turns":3}
`
	id, _ := themPhienCoNhatKy(t, a, than)

	// Running() là chỗ phát hiện phiên chết và gọi bộ phân loại.
	if _, err := a.db.Running(); err != nil {
		t.Fatal(err)
	}
	s, err := a.db.Phien(id)
	if err != nil {
		t.Fatal(err)
	}
	if s.State != store.StateHanMuc {
		t.Fatalf("phiên #%d ra trạng thái %q, chờ %q — khối tiêu đề đang làm hỏng phép đọc bản ghi",
			id, s.State, store.StateHanMuc)
	}
	if s.HanMucDenLai != 1755700000 {
		t.Errorf("mốc hạn mức cấp lại = %d, chờ 1755700000", s.HanMucDenLai)
	}
}

// Đo thẳng ở tầng provider cho chắc: bộ đọc bản ghi phải BỎ QUA khối tiêu đề.
func TestDocKetQuaBoQuaKhoiTieuDe(t *testing.T) {
	dau := nhatky.Dau{ThoiDiem: time.Now(), Addr: "claude:tns#1", Lenh: []string{"-p", "x"}}
	than := `{"type":"result","subtype":"success","is_error":false,"result":"xong","num_turns":2}` + "\n"
	ad, err := adapterOf("claude")
	if err != nil {
		t.Fatal(err)
	}
	k, ok := ad.DocKetQua(dau.String() + than)
	if !ok {
		t.Fatal("có khối tiêu đề thì DocKetQua không đọc ra bản ghi nữa")
	}
	if k.TraLoi != "xong" || k.CoLoi {
		t.Fatalf("KetQua sai: %+v", k)
	}
	if st, _, _ := provider.PhanLoaiChet(k, ok); st != store.StateXong {
		t.Fatalf("PhanLoaiChet = %q, chờ %q", st, store.StateXong)
	}
}

// Đường của MÁY phải CẮT khối tiêu đề: kết quả này thành output của bước agent
// và được nạp thẳng vào prompt của bước SAU. Để nguyên thì ghi chú nội bộ của
// sagent (đường dẫn hồ sơ, dòng lệnh) bị bước sau đọc như dữ liệu thật.
func TestReadLogsCatKhoiTieuDeCuaSagent(t *testing.T) {
	dir := t.TempDir()
	dau := nhatky.Dau{
		ThoiDiem: time.Now(), Addr: "claude:tns#1",
		HoSo: `C:\clones\claude\tns\1`, Lenh: []string{"-p", "việc"},
	}
	than := "câu trả lời của agent\n"
	p := filepath.Join(dir, "a.log")
	if err := os.WriteFile(p, []byte(dau.String()+than), 0o600); err != nil {
		t.Fatal(err)
	}
	got := readLogs([]string{p})
	if strings.Contains(got, nhatky.MocDau) || strings.Contains(got, `C:\clones\claude\tns\1`) {
		t.Fatalf("ghi chú nội bộ của sagent lọt vào output của bước agent:\n%s", got)
	}
	if !strings.Contains(got, "câu trả lời của agent") {
		t.Fatalf("cắt nhầm cả chữ của agent:\n%s", got)
	}
}

// Đường của NGƯỜI thì GIỮ khối tiêu đề — nó chính là phần trả lời được "vì sao
// phiên này không làm gì cả" (ca #167: nhìn dòng lệnh là thấy thiếu cờ).
func TestSessionNhatKyDocGiuKhoiTieuDeChoNguoiDoc(t *testing.T) {
	a := moAPI(t)
	id, p := themPhienCoNhatKy(t, a, "chữ của agent\n")

	m, noiDung, err := a.SessionNhatKyDoc(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.Duong != p || !m.ConFile || m.Co == 0 {
		t.Fatalf("MucNhatKy sai: %+v", m)
	}
	for _, phai := range []string{"claude:tns#1", "--output-format stream-json", "chữ của agent"} {
		if !strings.Contains(noiDung, phai) {
			t.Errorf("người đọc mất %q:\n%s", phai, noiDung)
		}
	}
}

// Trần số dòng: một nhật ký hàng chục MB không được đổ hết ra mặt nào.
func TestSessionNhatKyDocCatTheoDuoi(t *testing.T) {
	a := moAPI(t)
	var b strings.Builder
	for i := 0; i < 500; i++ {
		b.WriteString("dòng\n")
	}
	b.WriteString("DONG-CUOI\n")
	id, _ := themPhienCoNhatKy(t, a, b.String())

	_, noiDung, err := a.SessionNhatKyDoc(id, 10)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.Split(noiDung, "\n")); n > 10 {
		t.Fatalf("xin 10 dòng cuối mà nhận %d dòng", n)
	}
	// Đuôi là chỗ đáng đọc nhất: bản ghi `{"type":"result"}` và lý do chết đều
	// nằm ở cuối.
	if !strings.Contains(noiDung, "DONG-CUOI") {
		t.Fatal("cắt theo đuôi mà mất mất dòng cuối cùng")
	}
}

// Phiên KHÔNG có nhật ký: nói rõ, đừng trả về chuỗi rỗng như thể mọi thứ ổn.
func TestSessionNhatKyDocNoiRoKhiKhongCoNhatKy(t *testing.T) {
	a := moAPI(t)
	id, err := a.db.AddSession(store.Session{Provider: "claude", Account: "tns", Dir: "d", PID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.SessionNhatKyDoc(id, 10); err == nil {
		t.Fatal("phiên không có nhật ký mà không báo gì")
	}
	if _, _, err := a.SessionNhatKyDoc(99999, 10); err == nil {
		t.Fatal("hỏi số phiên không tồn tại mà không báo lỗi")
	}
}

// Bảng nhật ký phải phân biệt "file đã bị dọn" với "file rỗng" — hai manh mối
// khác hẳn nhau. Trộn lại thì người đọc đi sửa nhầm chỗ.
func TestSessionNhatKyDSPhanBietDaDonVaFileRong(t *testing.T) {
	a := moAPI(t)
	idCo, _ := themPhienCoNhatKy(t, a, "có chữ\n")
	idMat, pMat := themPhienCoNhatKy(t, a, "sẽ bị dọn\n")
	if err := os.Remove(pMat); err != nil {
		t.Fatal(err)
	}

	ds, err := a.SessionNhatKyDS(10)
	if err != nil {
		t.Fatal(err)
	}
	theo := map[int64]MucNhatKy{}
	for _, m := range ds {
		theo[m.ID] = m
	}
	if !theo[idCo].ConFile || theo[idCo].Co == 0 {
		t.Errorf("phiên còn nhật ký mà bảng nói không: %+v", theo[idCo])
	}
	if theo[idMat].ConFile {
		t.Errorf("nhật ký đã bị dọn mà bảng vẫn nói còn: %+v", theo[idMat])
	}
	// Đường dẫn vẫn phải giữ, kể cả khi file không còn: nó nói được nhật ký ĐÃ
	// TỪNG ở đâu, khác hẳn phiên chưa bao giờ có nhật ký.
	if theo[idMat].Duong == "" {
		t.Error("mất luôn đường dẫn của nhật ký đã bị dọn")
	}
	// Mới nhất trước.
	if len(ds) < 2 || ds[0].ID < ds[1].ID {
		t.Errorf("bảng không xếp mới nhất trước: %+v", ds)
	}
}
