package dash

import (
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

func fileWeb(t *testing.T) []string {
	t.Helper()
	ents, err := os.ReadDir("web")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".html") {
			out = append(out, filepath.Join("web", e.Name()))
		}
	}
	if len(out) == 0 {
		t.Fatal("khong thay file web nao — test nay se xanh gia")
	}
	return out
}
