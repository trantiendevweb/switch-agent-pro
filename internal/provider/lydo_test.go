package provider

import (
	"strings"
	"testing"
)

// CA DA NO THAT (luot chay #35, buoc code-doc): cau tra loi cua agent mo dau
// bang mot tieu de markdown, nen thong bao hong hoa ra:
//
//	agent bao loi: ### NHANH
//
// Nguoi doc khong suy ra duoc gi, ma lai tuong day la ly do that nen thoi khong
// dao tiep. Tha noi "khong noi ly do" roi de ho di doc output.
func TestKhongLayTieuDeMarkdownLamLyDo(t *testing.T) {
	k := KetQua{CoLoi: true, Loai: "success",
		TraLoi: "### NHANH\n\nsagent/may-1\n\n### COMMIT\n\n1426ac6"}
	ly := k.Hong()
	if strings.Contains(ly, "###") {
		t.Fatalf("lay tieu de markdown lam ly do: %q", ly)
	}
	if !strings.Contains(ly, "khong noi ly do") && !strings.Contains(ly, "không nói lý do") {
		t.Fatalf("khong co ly do dung duoc thi phai noi thang la khong biet, duoc %q", ly)
	}
}

// Cac dinh dang khac cung phai bi loc: gach ngang, khoi ma, bang, trich dan.
func TestLocMoiDinhDangKhongPhaiLyDo(t *testing.T) {
	for _, dong := range []string{
		"--- ket qua ---",
		"```go",
		"| nhanh | commit |",
		"> luu y",
		"* buoc mot",
		"# Bao cao",
		"OK",    // qua ngan, khong mang chan doan nao
		"NHANH", // y het ca da no that
	} {
		k := KetQua{CoLoi: true, TraLoi: dong}
		if ly := k.Hong(); strings.Contains(ly, dong) {
			t.Errorf("dong %q khong phai ly do ma van bi dung lam ly do: %q", dong, ly)
		}
	}
}

// Nguoc lai: mot cau giai thich THAT thi phai duoc giu, dung loc qua tay.
func TestVanGiuCauGiaiThichThat(t *testing.T) {
	for _, dong := range []string{
		"Credit balance is too low",
		"Failed to authenticate: OAuth session expired and could not be refreshed",
		"khong ket noi duoc toi endpoint",
	} {
		k := KetQua{CoLoi: true, TraLoi: dong}
		if ly := k.Hong(); !strings.Contains(ly, dong) {
			t.Errorf("cau giai thich that bi loc mat: %q -> %q", dong, ly)
		}
	}
}
