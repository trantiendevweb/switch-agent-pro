package flow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// CHUYEN DA XAY RA (luot chay #46): buoc `soi` — nguoi soi doc lap, cong kiem
// cuoi cung truoc khi tron nhanh — nhan HTTP 503 tu nha cung cap va tra ve dung
// mot cau: "Sorry, I encountered an error: Grok API error: 503 Service
// temporarily unavailable".
//
// CLI thoat ma 0. Ban ghi khong co truong loi nao. Nen buoc duoc ghi la DONE,
// ban tom tat in "khong buoc nao hong", va viec tron nhanh dien ra ma KHONG AI
// SOI.
//
// Do la kieu hong te nhat cua mot cong kiem: no khong sap, no chi lang le gat
// dau. "Da soi va khong thay gi" voi "chua tung duoc soi" la hai cau khac han
// nhau, ma nhin vao bang thi giong het.
//
// PhaiCo la hop dong dau ra: buoc phai GIAO RA thu no duoc giao. Day KHONG phai
// do chuoi loi — do chuoi loi la doan xem nha cung cap viet cau xin loi the nao,
// ho doi cau chu la hong. Con day la kiem buoc co giao hang khong, va danh sach
// do NGUOI VIET FLOW khai trong flows.toml chu khong nam cung trong ma.

func TestKhongKhaiPhaiCoThiKhongKiemGi(t *testing.T) {
	s := Step{ID: "soi"}
	if ly := ThieuPhaiCo(s, "bat cu thu gi"); ly != "" {
		t.Errorf("khong khai phai_co ma van chan: %q", ly)
	}
	if ly := ThieuPhaiCo(s, ""); ly != "" {
		t.Errorf("khong khai phai_co, output rong, van chan: %q", ly)
	}
}

func TestCauXinLoiCuaNhaCungCapKhongQuaDuocHopDong(t *testing.T) {
	s := Step{ID: "soi", PhaiCo: []string{"NEN TRON", "CAN SUA THEM", "VUT"}}
	// Nguyen van cau ma grok da tra ve o luot #46.
	that := "Sorry, I encountered an error: Grok API error: 503 Service temporarily unavailable"
	ly := ThieuPhaiCo(s, that)
	if ly == "" {
		t.Fatal("cau xin loi 503 van duoc tinh la da soi — dung loi da xay ra o luot #46")
	}
	// Cau bao phai noi RO day la "chua lam", khong phai "lam roi va khong y kien".
	if !strings.Contains(ly, "CHƯA LÀM") {
		t.Errorf("cau bao khong noi ro buoc CHUA LAM: %q", ly)
	}
}

func TestCoKetLuanThiQua(t *testing.T) {
	s := Step{ID: "soi", PhaiCo: []string{"NEN TRON", "CAN SUA THEM", "VUT"}}
	for _, out := range []string{
		"TNS: NEN TRON — co commit va diff day du\nMAY: VUT — nhanh rong",
		"tns: can sua them, thieu test",            // thuong/hoa khong phan biet
		"TNS: NEN\n TRON — xuong dong giua cum tu", // khoang trang duoc gon lai
	} {
		if ly := ThieuPhaiCo(s, out); ly != "" {
			t.Errorf("output %q bi chan nham: %s", out, ly)
		}
	}
}

// Buoc khong phan xet tron/khong van dung duoc hop dong — chi la ten dong thay
// vi tu ket luan. Moi buoc mot hop dong RIENG, lay tu chinh prompt cua no.
func TestHopDongKhongChiDanhChoBuocSoi(t *testing.T) {
	s := Step{ID: "tho", PhaiCo: []string{"THAY DOI:"}}
	if ly := ThieuPhaiCo(s, "THAY DOI: co 2 commit\nNOI DUNG: khong thay"); ly != "" {
		t.Errorf("bi chan nham: %s", ly)
	}
	if ThieuPhaiCo(s, "toi da chay xong hai lenh") == "" {
		t.Error("khong giao ra dong THAY DOI: ma van qua")
	}
}

// MOI buoc `soi` trong flows.toml PHAI co hop dong dau ra.
//
// Bai nay gac dung cho da thung: nguoi soi la cong kiem cuoi truoc khi tron
// nhanh, va mot cong kiem lang le gat dau thi te hon la khong co cong nao —
// khong co cong thi con biet la chua kiem.
func TestMoiBuocSoiDeuCoHopDongDauRa(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", ".sagent", "flows.toml"))
	if err != nil {
		t.Skipf("khong doc duoc flows.toml: %v", err)
	}
	// Cat theo tung khoi [[...]] roi chi xet khoi co id = "soi".
	var khoi []string
	var nay []string
	for _, d := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(strings.TrimSpace(d), "[[") {
			khoi = append(khoi, strings.Join(nay, "\n"))
			nay = nil
		}
		nay = append(nay, d)
	}
	khoi = append(khoi, strings.Join(nay, "\n"))

	so := 0
	for _, k := range khoi {
		if !strings.Contains(k, `id = "soi"`) {
			continue
		}
		so++
		if !strings.Contains(k, "phai_co = [") {
			ten := k
			if i := strings.Index(ten, "\n"); i > 0 {
				ten = strings.TrimSpace(ten[:i])
			}
			t.Errorf("buoc soi trong %s khong khai phai_co — mot cong kiem lang le gat dau "+
				"thi te hon la khong co cong nao", ten)
		}
	}
	if so == 0 {
		t.Fatal("khong tim thay buoc soi nao trong flows.toml — bo cat khoi hong roi")
	}
	t.Logf("%d buoc soi, tat ca deu co hop dong dau ra", so)
}

// BAI QUAN TRONG NHAT trong file nay: chay QUA BO CHAY THAT.
//
// Bon bai tren chi kiem HAM ThieuPhaiCo. Ham dung ma khong ai goi thi van hong y
// nguyen — va do dung la cai bay toi vua dinh: go phan kiem hop dong ra khoi
// step.go, bon bai tren VAN XANH.
//
// "Cho hong nam o CHO GOI" — bai hoc da tra gia mot lan voi {{steps.x.output}}.
func TestBoChayThatChanBuocKhongGiaoRaKetLuan(t *testing.T) {
	r, ag, db := newRunner(t)
	// Nguyen van cau grok da tra ve o luot #46 khi nha cung cap tra 503.
	ag.output = "Sorry, I encountered an error: Grok API error: 503 Service temporarily unavailable"

	f := Flow{Name: "thu", Steps: []Step{{
		ID: "soi", Type: TypeAgent, Profile: "grok:api",
		PhaiCo: []string{"NEN TRON", "CAN SUA THEM", "VUT"},
	}}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	tt := trangThaiBuoc(t, db, res.RunID, "soi")
	if tt == store.StepDone {
		t.Fatalf("buoc soi ra DONE trong khi no chi tra ve cau xin loi 503 — dung loi da " +
			"xay ra o luot #46: cong kiem lang le gat dau")
	}
	if tt != store.StepFailed {
		t.Errorf("buoc soi o trang thai %q, cho doi %q", tt, store.StepFailed)
	}
}

// trangThaiBuoc doc trang thai mot buoc tu so.
func trangThaiBuoc(t *testing.T, db *store.DB, runID int64, id string) string {
	t.Helper()
	ds, err := db.Steps(runID)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := ds[id]
	if !ok {
		t.Fatalf("khong thay buoc %q trong so", id)
	}
	return b.State
}

// Va nguoc lai: giao ra ket luan that thi phai QUA. Khong co bai nay thi cach
// "sua" de nhat cho bai tren la chan tat ca, va cong kiem thanh cai cua khoa.
func TestBoChayThatChoQuaKhiCoKetLuan(t *testing.T) {
	r, ag, db := newRunner(t)
	ag.output = "TNS: NEN TRON — co commit va diff day du\nMAY: VUT — nhanh rong"

	f := Flow{Name: "thu", Steps: []Step{{
		ID: "soi", Type: TypeAgent, Profile: "grok:api",
		PhaiCo: []string{"NEN TRON", "CAN SUA THEM", "VUT"},
	}}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if tt := trangThaiBuoc(t, db, res.RunID, "soi"); tt != store.StepDone {
		t.Errorf("buoc soi giao ra ket luan that ma van bi chan: state=%q", tt)
	}
}
