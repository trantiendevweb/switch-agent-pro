package dash

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// Design system cua du an (design-system/switch-agent-pro/MASTER.md) co muc
// "Anti-Patterns (Do NOT Use)" va mot checklist truoc khi giao. Checklist nam
// trong tai lieu thi khong ai doc — ngay 18/08 chinh toi vi pham hai muc cua no
// ngay trong luot lam giao dien: dung emoji lam icon (crown, target, canh bao),
// va thieu prefers-reduced-motion o hai trang.
//
// Nen bien no thanh test. Muc nao may kiem duoc thi de may kiem.

// CAM: emoji / ky tu la lam icon. Ly do trong MASTER.md: emoji doi hinh theo he
// dieu hanh va phong chu, khong chinh duoc net, khong doi mau theo trang thai.
// Dung SVG (co san ham icon() trong flow.html) hoac cham CSS.
func TestKhongDungEmojiLamIcon(t *testing.T) {
	// Ky tu duoc phep: mui ten trong cau van, dau nhan, gach ngang.
	choPhep := map[rune]bool{
		0x2192: true, // ->
		0x2190: true, // <-
		0x00B7: true, // dau cham giua
		0x2014: true, // gach ngang dai
		0x2013: true,
	}
	for _, f := range fileWeb(t) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, dong := range strings.Split(string(b), "\n") {
			for _, r := range dong {
				if choPhep[r] || r < 0x2000 {
					continue
				}
				if unicode.In(r, unicode.So, unicode.Sk) || (r >= 0x1F300 && r <= 0x1FAFF) {
					t.Errorf("%s dong %d: dung ky tu U+%04X lam icon — design system cam, hay dung SVG hoac cham CSS:\n  %s",
						filepath.Base(f), i+1, r, strings.TrimSpace(dong))
					break
				}
			}
		}
	}
}

// CAM: chuyen dong khong ton trong prefers-reduced-motion. Trang nao co animation
// hoac transition thi phai co khoi @media tuong ung.
func TestTonTrongReducedMotion(t *testing.T) {
	for _, f := range fileWeb(t) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		coChuyenDong := strings.Contains(s, "@keyframes") ||
			strings.Contains(s, "animation:") ||
			strings.Contains(s, "transition:")
		if coChuyenDong && !strings.Contains(s, "prefers-reduced-motion") {
			t.Errorf("%s co chuyen dong nhung khong co @media (prefers-reduced-motion: reduce)",
				filepath.Base(f))
		}
	}
}

// mienTru: trang duoc tha, kem LY DO. Danh sach nay chi duoc ngan lai, khong dai ra.
//
// Vi sao co no: ban dau test chi quet web/*.html o tang tren, bo qua ca thu muc
// web/docs/. Do 19/08: rieng hai trang trong docs/ co 82 cho dung emoji lam icon,
// va vi khong bi quet nen chung con lay lan sang trang moi. Nay quet DE QUY.
//
// master-plan.html la trang sinh ra tu MASTER-PLAN.md, chua co khau sinh lai tu
// dong nen don tay se lech voi ban .md. Tha tam, va ghi ro la no NO CHU khong
// phai la khong sao.
var mienTru = map[string]string{
	"master-plan.html": "trang sinh tu MASTER-PLAN.md, chua co khau sinh lai — don tay se lech voi ban .md",
}

func fileWeb(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir("web", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".html") {
			return nil
		}
		if ly, tha := mienTru[d.Name()]; tha {
			t.Logf("bo qua %s — %s", d.Name(), ly)
			return nil
		}
		out = append(out, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("khong thay file web nao — test nay se xanh gia")
	}
	return out
}
