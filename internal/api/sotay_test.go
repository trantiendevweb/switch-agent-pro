package api

// SỔ ĐĂNG KÝ nhìn từ tầng HỢP ĐỒNG.
//
// Bốn ca ở internal/store và hai ca ở internal/profile đo từng mảnh rời. Ở đây
// đo thứ khác hẳn: DÂY NỐI. Một cái sổ đúng đến mấy mà `them` quên ghi vào, hoặc
// `xoa` quên hỏi, thì vẫn là một bảng rỗng trong DB — mà đó lại đúng là kiểu
// hỏng không test đơn vị nào bắt được.
//
//	(g) them → sổ có dòng, và dòng đó nhận sở hữu
//	(h) xoa  → hỏi sổ; sổ nói không thì DỪNG, thư mục còn nguyên
//	(i) xoa xong → dòng sổ phải đi theo, không để lại rác
//	(j) hồ sơ có TRƯỚC sổ (v8) vẫn phải xoá được
//	(k) route đã gọi thật thì vào sổ; RouteList đối chiếu hai chiều

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/config"
	"github.com/trantiendevweb/switch-agent-pro/internal/profile"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// dungAPISo dựng API trên một HOME giả — mọi thứ (kể cả state.db) nằm trong
// thư mục tạm, không đụng máy thật.
func dungAPISo(t *testing.T) *API {
	t.Helper()
	home := homeGiaAPI(t)
	// `them` nối phần dùng chung từ thư mục config gốc của provider, nên thư mục
	// đó phải tồn tại — trên máy thật nó luôn có, ở đây phải dựng.
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Và file cấu hình gộp, nguồn để gieo whitelist "thói quen máy".
	if err := os.WriteFile(filepath.Join(home, ".claude", ".claude.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := New(t.TempDir())
	if err != nil {
		t.Skipf("không mở được store trong môi trường test: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	return a
}

func timMuc(ds []store.MucHoSo, prov, acc string) (store.MucHoSo, bool) {
	for _, m := range ds {
		if m.Provider == prov && m.Account == acc {
			return m, true
		}
	}
	return store.MucHoSo{}, false
}

// ---- ca (g)+(i): tạo thì vào sổ, xoá thì ra khỏi sổ ----
func TestCaG_TaoHoSoThiVaoSoVaXoaThiRaKhoiSo(t *testing.T) {
	a := dungAPISo(t)
	addr := Addr{"claude", "phu"}

	if _, _, err := a.ProfileCreate(addr); err != nil {
		t.Fatalf("không tạo được hồ sơ: %v", err)
	}
	h, co, err := a.db.TimHoSo("claude", "phu")
	if err != nil {
		t.Fatal(err)
	}
	if !co {
		t.Fatal("`them` KHÔNG ghi hồ sơ vào sổ — sau này `xoa` sẽ không có gì để hỏi")
	}
	if !h.SoTaoRa {
		t.Fatal("sổ ghi hồ sơ do chính sagent vừa tạo mà không nhận sở hữu")
	}
	if h.Dir != profile.Dir("claude", "phu") {
		t.Fatalf("sổ ghi sai đường dẫn: %q", h.Dir)
	}

	// Đối chiếu ngay sau khi tạo: phải khớp.
	ds, err := a.ProfileSo()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := timMuc(ds, "claude", "phu")
	if !ok || m.TrangThai != store.SoKhop {
		t.Fatalf("vừa tạo xong mà đối chiếu đã lệch: %+v", ds)
	}

	// ca (i): xoá xong thì dòng sổ đi theo. Để lại thì lần đối chiếu sau báo
	// "sổ có, đĩa không" cho một hồ sơ chính sagent vừa xoá — báo động giả, và
	// báo động giả thì làm người ta ngừng đọc bảng.
	if err := a.ProfileRemove(addr); err != nil {
		t.Fatalf("không xoá được hồ sơ chính sagent tạo ra: %v", err)
	}
	if _, co, _ := a.db.TimHoSo("claude", "phu"); co {
		t.Fatal("xoá hồ sơ xong mà dòng sổ vẫn còn")
	}
	ds, err = a.ProfileSo()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := timMuc(ds, "claude", "phu"); ok {
		t.Fatalf("hồ sơ đã xoá vẫn hiện trong bảng đối chiếu: %+v", ds)
	}
}

// ---- ca (h): sổ nói KHÔNG thì `xoa` phải dừng ----
//
// Dựng đúng tình huống thật: một thư mục nằm gọn trong kho, tên hợp lệ, nhưng
// sagent không tạo ra nó — và sổ đã ghi lại điều đó. Trước khi có sổ, `xoa`
// nuốt gọn thư mục này vì `insideStore` cho qua.
func TestCaH_SoNoiKhongSoHuuThiXoaPhaiDung(t *testing.T) {
	a := dungAPISo(t)
	dir := profile.Dir("claude", "cua-nguoi-khac")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	giu := filepath.Join(dir, "cua-toi.txt")
	if err := os.WriteFile(giu, []byte("ĐỪNG ĐỤNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.db.GhiHoSo(store.HoSo{
		Provider: "claude", Account: "cua-nguoi-khac", Dir: dir, SoTaoRa: false,
	}); err != nil {
		t.Fatal(err)
	}

	err := a.ProfileRemove(Addr{"claude", "cua-nguoi-khac"})
	if err == nil {
		t.Fatal("XOÁ MẤT thư mục mà sổ ghi rõ là không sở hữu")
	}
	if !strings.Contains(err.Error(), "sổ đăng ký") {
		t.Fatalf("từ chối nhưng không nói vì sổ, người dùng sẽ không hiểu: %v", err)
	}
	if b, readErr := os.ReadFile(giu); readErr != nil || string(b) != "ĐỪNG ĐỤNG" {
		t.Fatalf("file trong thư mục bị động: %q, %v", b, readErr)
	}
	// Và lần từ chối đó KHÔNG được lặng lẽ đổi dòng sổ thành "sở hữu" để lần sau
	// trôi qua.
	h, _, _ := a.db.TimHoSo("claude", "cua-nguoi-khac")
	if h.SoTaoRa {
		t.Fatal("một lần `xoa` bị từ chối đã tự cấp quyền sở hữu cho lần sau")
	}
}

// ---- ca (j): hồ sơ có TRƯỚC khi có sổ vẫn phải xoá được ----
//
// Sổ chỉ có từ schema v8. Mọi hồ sơ tạo trước đó — và mọi hồ sơ di trú từ v1 —
// đều không có dòng nào. Nếu "không có trong sổ" bị hiểu là "không được xoá" thì
// bản vá an toàn này làm gãy `xoa` cho đúng những người dùng lâu năm nhất.
func TestCaJ_HoSoCoTruocSoVanXoaDuoc(t *testing.T) {
	a := dungAPISo(t)
	dir := profile.Dir("claude", "cu")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, co, _ := a.db.TimHoSo("claude", "cu"); co {
		t.Fatal("tiền đề sai: sổ lẽ ra chưa biết hồ sơ này")
	}
	if err := a.ProfileRemove(Addr{"claude", "cu"}); err != nil {
		t.Fatalf("hồ sơ tạo trước khi có sổ không xoá được nữa: %v", err)
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		t.Fatal("báo xoá xong nhưng thư mục vẫn còn")
	}
}

// ---- ca (k): route gọi thật thì vào sổ, và RouteList đối chiếu hai chiều ----
func TestCaK_RouteGoiThatThiVaoSoVaDoiChieuHaiChieu(t *testing.T) {
	homeGiaAPI(t)
	may, _ := mayTot("m-1", "xong", 3, 4)
	defer may.Close()

	var c config.Config
	c.AI.DefaultRoute = "chinh"
	themRoute(&c, "chinh", may.URL, "m-1")
	a := dungAPI(t, c)

	// Trước khi gọi: cấu hình khai "chinh", sổ chưa có gì.
	ds, err := a.RouteList()
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 || ds[0].TrangThai != store.SoThieuSo {
		t.Fatalf("route khai mà chưa gọi bao giờ phải là %q: %+v", store.SoThieuSo, ds)
	}

	if _, err := a.AICall(context.Background(), "", "hỏi"); err != nil {
		t.Fatalf("gọi API hỏng: %v", err)
	}

	ds, err = a.RouteList()
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 || ds[0].TrangThai != store.SoKhop {
		t.Fatalf("gọi xong rồi mà sổ route vẫn không khớp cấu hình: %+v", ds)
	}
	// Sổ KHÔNG được giữ khoá — chỉ key_id (tên file). Xem migration v8.
	trongSo, err := a.db.RouteTrongSo()
	if err != nil {
		t.Fatal(err)
	}
	if len(trongSo) != 1 || trongSo[0].KeyID != "k" {
		t.Fatalf("sổ route ghi sai: %+v", trongSo)
	}

	// Gỡ route khỏi cấu hình: sổ vẫn nhớ là đã tiêu tiền qua đó. Đây là chiều
	// mà một mình file cấu hình không bao giờ nói được.
	a.cfg.AI.Routes = nil
	ds, err = a.RouteList()
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 || ds[0].TrangThai != store.SoThieuDia {
		t.Fatalf("route đã gọi thật rồi bị gỡ khỏi cấu hình phải là %q: %+v", store.SoThieuDia, ds)
	}
}
