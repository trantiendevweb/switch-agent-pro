package dash

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// $ trong cac trang web cua du an la getElementById, KHONG phai querySelector.
var goiDollar = regexp.MustCompile(`\$\((['"])([^'"]+)['"]\)`)

// CA DA NO THAT (do 19/08): index.html khai `const $ = id => document.getElementById(id)`
// nhung 12 cho goi `$('#wf-ten')`. getElementById('#wf-ten') tra ve null, nen dong
//
//	$('#wf-ten').onchange = veMotaFlow
//
// nem TypeError o CAP CAO NHAT — chan sach moi thu phia sau no: nut Dung, form ham
// doi, va ca EventSource. Nghia la bang dieu khien 2D KHONG HE realtime, ma nhin
// vao thi khong thay gi bat thuong: trang van ve ra day du, chi la khong bam duoc.
//
// Day dung kieu hong ma du an nay so nhat — im lang, va trong co ve on.
//
// Test kiem hai thu, deu re:
//  1. Doi so cua $() khong duoc bat dau bang '#' hoac '.' (do la cu phap selector).
//  2. Moi id duoc goi phai co that trong chinh file do.
//
// (2) con bat duoc ca loi go nham ten id — cung ngam theo dung mot duong.
func TestGoiDOMPhaiTrungIdCoThat(t *testing.T) {
	for _, f := range fileWeb(t) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)

		// Trang nao khong dung kieu $ = getElementById thi bo qua, dung bat va.
		if !strings.Contains(s, "getElementById") {
			continue
		}
		if !regexp.MustCompile(`\$\s*=\s*\w*\s*=>\s*document\.getElementById`).MatchString(s) {
			continue
		}

		for _, m := range goiDollar.FindAllStringSubmatch(s, -1) {
			id := m[2]
			if strings.HasPrefix(id, "#") || strings.HasPrefix(id, ".") {
				t.Errorf("%s: $(%q) dung cu phap selector, nhung $ la getElementById "+
					"— no se tra null va nem TypeError chan het phan sau", f, id)
				continue
			}
			// Bo qua thu ro rang khong phai id (co khoang trang, dau ngoac...).
			if strings.ContainsAny(id, " >:[]()") {
				continue
			}
			if !strings.Contains(s, `id="`+id+`"`) && !strings.Contains(s, `id='`+id+`'`) {
				t.Errorf("%s: $(%q) nhung khong co phan tu nao mang id do", f, id)
			}
		}
	}
}
