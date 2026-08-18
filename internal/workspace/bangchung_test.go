package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func chay(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// Ba tình huống đã GẶP THẬT ở lần chạy #21, không phải tình huống bịa:
//   - thợ làm và commit  -> phải đếm đúng số commit
//   - thợ không làm gì   -> phải nói KHÔNG có commit nào (đây là cái đã lọt lưới)
//   - thợ sửa mà quên commit -> phải nói còn thay đổi chưa commit
func TestBangChungPhanBietLamThatVaKhongLamGi(t *testing.T) {
	repo := t.TempDir()
	chay(t, repo, "init", "-b", "main")
	chay(t, repo, "config", "user.email", "test@example.com")
	chay(t, repo, "config", "user.name", "test")
	chay(t, repo, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("goc"), 0o644); err != nil {
		t.Fatal(err)
	}
	chay(t, repo, "add", ".")
	chay(t, repo, "commit", "-m", "khoi tao")

	// 1. nhánh KHÔNG làm gì
	chay(t, repo, "branch", "sagent/rong")
	wtRong := filepath.Join(t.TempDir(), "rong")
	chay(t, repo, "worktree", "add", wtRong, "sagent/rong")
	bc := Xem(wtRong, "main")
	if bc.KhongRo {
		t.Fatal("không đọc được git trong worktree vừa tạo")
	}
	if bc.Commit != 0 {
		t.Fatalf("nhánh chưa làm gì mà đếm %d commit", bc.Commit)
	}
	if !strings.Contains(bc.MotDong(), "KHÔNG có commit nào") {
		t.Fatalf("phải nói rõ là không có commit: %q", bc.MotDong())
	}

	// 2. nhánh CÓ làm và commit
	chay(t, repo, "branch", "sagent/lam")
	wtLam := filepath.Join(t.TempDir(), "lam")
	chay(t, repo, "worktree", "add", wtLam, "sagent/lam")
	if err := os.WriteFile(filepath.Join(wtLam, "b.txt"), []byte("viec that"), 0o644); err != nil {
		t.Fatal(err)
	}
	chay(t, wtLam, "add", ".")
	chay(t, wtLam, "commit", "-m", "lam viec that")
	bc2 := Xem(wtLam, "main")
	if bc2.Commit != 1 {
		t.Fatalf("nhánh có 1 commit mà đếm %d — %s", bc2.Commit, bc2.MotDong())
	}

	// 3. sửa mà QUÊN commit
	if err := os.WriteFile(filepath.Join(wtLam, "c.txt"), []byte("quen commit"), 0o644); err != nil {
		t.Fatal(err)
	}
	bc3 := Xem(wtLam, "main")
	if !bc3.Ban {
		t.Fatal("có file chưa commit mà không báo bẩn")
	}
	if !strings.Contains(bc3.MotDong(), "CHƯA commit") {
		t.Fatalf("phải nhắc còn thay đổi chưa commit: %q", bc3.MotDong())
	}
}
