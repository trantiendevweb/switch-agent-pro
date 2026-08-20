package dash

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// Moi file tinh PHAI ra kem mot van xac thuc. Khong co van thi trinh duyet duoc
// phep tu doan han dung, va no doan rat rong rai — sua giao dien xong nguoi dung
// van thay ban cu, ma khong co ly do gi de nghi ra chuyen bam Ctrl+F5.
//
// Bai nay sinh ra tu mot ngay that: bon lan sua mat van phong, bon lan build va
// cai lai, bon lan bao "xong". Anh chup nguoi dung gui ve: y nguyen ban cu. Ma
// dung, nam trong binary, chi la trinh duyet khong them hoi lai lan nao.

func tepNhungThu(t *testing.T) http.Handler {
	t.Helper()
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		t.Fatal(err)
	}
	return tepNhung(sub, http.FileServer(http.FS(sub)))
}

// Danh sach duoi day la nhung duong nguoi dung GO THANG vao thanh dia chi. Hai
// duong dau ket thuc bang "/" nen di qua nhanh "tu tra index.html" cua
// http.FileServer — dung cho de sot van nhat, vi ten file khong he xuat hien
// trong duong dan.
func TestMoiTrangTinhDeuCoVanXacThuc(t *testing.T) {
	h := tepNhungThu(t)
	for _, d := range []string{
		"/",                 // 2D — tu tra index.html
		"/docs/",            // ke hoach — cung tu tra index.html
		"/vanphong.html",    // van phong 3D
		"/3d.html",          // so do 3D
		"/flow.html",        // bang workflow
		"/hoi-thoai.html",   // hoi thoai
		"/vendor/token.css", // bang mau dung chung
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, d, nil))
		if et := w.Header().Get("ETag"); et == "" {
			t.Errorf("%s khong co ETag (ma %d) — trinh duyet se giu ban cu vo han sau moi lan build",
				d, w.Code)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("%s co Cache-Control %q, phai la \"no-cache\" — tuc la luu thi luu, "+
				"nhung lan nao cung phai hoi lai", d, cc)
		}
	}
}

// Hoi lai voi dung ETag phai duoc tra 304 RONG. Neu van tra ca file thi van xac
// thuc chi la trang tri: dung ve mat dung sai, nhung moi lan mo trang van tai
// lai gan 1 MB vendor.
func TestHoiLaiDungETagThiTra304(t *testing.T) {
	h := tepNhungThu(t)
	const d = "/vendor/token.css"

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, d, nil))
	et := w.Header().Get("ETag")
	if et == "" || w.Body.Len() == 0 {
		t.Fatalf("lan dau: ETag=%q, %d byte", et, w.Body.Len())
	}

	r := httptest.NewRequest(http.MethodGet, d, nil)
	r.Header.Set("If-None-Match", et)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r)
	if w2.Code != http.StatusNotModified {
		t.Errorf("hoi lai voi dung ETag tra %d, phai la 304", w2.Code)
	}
	if w2.Body.Len() != 0 {
		t.Errorf("304 ma van gui %d byte noi dung", w2.Body.Len())
	}
}

// ETag phai bam NOI DUNG chu khong phai gio build: hai lan build cung mot ma
// phai ra cung mot ETag, khong thi moi lan khoi dong lai dash la moi trinh duyet
// tai lai gan 1 MB vendor trong khi khong co gi doi.
//
// Va nguoc lai: noi dung doi mot byte thi ETag PHAI doi — khong thi quay ve dung
// cai bay cu.
func TestETagBamNoiDungChuKhongPhaiGioBuild(t *testing.T) {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		t.Fatal(err)
	}
	a, b := bangETag(sub), bangETag(sub)
	if len(a) < 5 {
		t.Fatalf("chi bam duoc %d file — gan nhu chac chan la duyet nham cho", len(a))
	}
	for k, v := range a {
		if b[k] != v {
			t.Errorf("%s: hai lan bam ra hai ETag khac nhau (%s vs %s)", k, v, b[k])
		}
	}

	x := bangETag(fstest.MapFS{"a.txt": {Data: []byte("xin chao")}})
	y := bangETag(fstest.MapFS{"a.txt": {Data: []byte("xin chao!")}})
	if x["a.txt"] == y["a.txt"] {
		t.Error("doi noi dung ma ETag khong doi — van xac thuc vo dung")
	}
}

// Va bo boc phai duoc CAM THAT vao server, khong phai chi ton tai. `/vendor/` la
// duong cong khai nen di thang qua duoc, khong vuong dang nhap.
func TestServerThatCoCamVanXacThuc(t *testing.T) {
	s := New(nil)
	r := httptest.NewRequest(http.MethodGet, "/vendor/token.css", nil)
	r.Host = "127.0.0.1:8788"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Header().Get("ETag") == "" {
		t.Errorf("server that khong gan ETag cho /vendor/token.css (ma %d) — "+
			"bo boc co viet nhung chua cam vao New()", w.Code)
	}
}
