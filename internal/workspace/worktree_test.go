package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitRepo dựng một repo thật có sẵn một commit. Đây là integration test: dùng
// git thật vì bản thân các bug đã gặp nằm ở chỗ ta hiểu sai hành vi của git.
func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("máy không có git")
	}
	dir := t.TempDir()
	steps := [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
		{"config", "commit.gpgsign", "false"},
	}
	for _, s := range steps {
		c := exec.Command("git", s...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", s, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("xin chao"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, s := range [][]string{{"add", "."}, {"commit", "-m", "khoi tao"}} {
		c := exec.Command("git", s...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", s, err, out)
		}
	}
	return dir
}

func fakeHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func branches(t *testing.T, repo string) string {
	t.Helper()
	c := exec.Command("git", "branch", "--list")
	c.Dir = repo
	out, _ := c.Output()
	return string(out)
}

func TestAddCreatesWorktreeAndBranch(t *testing.T) {
	fakeHome(t)
	repo := gitRepo(t)

	dir, err := Add(repo, "phu-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatalf("worktree phải có file của repo: %v", err)
	}
	if !strings.Contains(branches(t, repo), "sagent/phu-1") {
		t.Fatalf("thiếu nhánh sagent/phu-1:\n%s", branches(t, repo))
	}
	// Worktree phải nằm NGOÀI repo, nếu không `git status` của agent sẽ thấy rác.
	if strings.HasPrefix(filepath.Clean(dir), filepath.Clean(repo)) {
		t.Fatalf("worktree %s không được nằm trong repo %s", dir, repo)
	}
}

// Chạy lại fleet khi worktree cũ còn đó thì không được kẹt.
func TestAddIsRepeatable(t *testing.T) {
	fakeHome(t)
	repo := gitRepo(t)
	if _, err := Add(repo, "phu-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(repo, "phu-1"); err != nil {
		t.Fatalf("gọi lần hai phải chạy được, được lỗi: %v", err)
	}
}

// HỒI QUY cho bug thật: trước đây `clean` đoán số thứ tự phu-1, phu-2… nên khi
// phu-1 đã bị gỡ, vòng lặp dừng ngay và bỏ sót phu-2, phu-3.
func TestFindAllSurvivesGaps(t *testing.T) {
	fakeHome(t)
	repo := gitRepo(t)
	for _, n := range []string{"phu-1", "phu-2", "phu-3"} {
		if _, err := Add(repo, n); err != nil {
			t.Fatal(err)
		}
	}
	if got := FindAll(repo, "phu"); len(got) != 3 {
		t.Fatalf("muốn 3 worktree, được %d", len(got))
	}

	// Gỡ cái ĐẦU để tạo khoảng trống.
	first, ok := Find(repo, "phu-1")
	if !ok {
		t.Fatal("không thấy phu-1")
	}
	if err := Remove(repo, first); err != nil {
		t.Fatal(err)
	}

	got := FindAll(repo, "phu")
	if len(got) != 2 {
		t.Fatalf("sau khi gỡ phu-1, muốn còn 2, được %d — vòng lặp lại dừng ở lỗ hổng", len(got))
	}
}

// FindAll chỉ được trả về worktree của ĐÚNG tài khoản đó.
func TestFindAllFiltersByAccount(t *testing.T) {
	fakeHome(t)
	repo := gitRepo(t)
	for _, n := range []string{"phu-1", "cty-1", "cty-2"} {
		if _, err := Add(repo, n); err != nil {
			t.Fatal(err)
		}
	}
	if got := FindAll(repo, "cty"); len(got) != 2 {
		t.Fatalf("muốn 2 worktree của cty, được %d", len(got))
	}
	if got := FindAll(repo, "phu"); len(got) != 1 {
		t.Fatalf("muốn 1 worktree của phu, được %d", len(got))
	}
}

// IsDirty là lá chắn giữ việc agent làm dở — sai một chút là mất dữ liệu thật.
func TestIsDirtyDetectsUncommitted(t *testing.T) {
	fakeHome(t)
	repo := gitRepo(t)
	dir, err := Add(repo, "phu-1")
	if err != nil {
		t.Fatal(err)
	}
	if IsDirty(dir) {
		t.Fatal("worktree vừa tạo phải sạch")
	}
	if err := os.WriteFile(filepath.Join(dir, "viec-dang-lam.txt"), []byte("dang lam"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsDirty(dir) {
		t.Fatal("file mới chưa commit mà báo sạch — lá chắn thủng")
	}
}

func TestRemoveKeepsBranch(t *testing.T) {
	fakeHome(t)
	repo := gitRepo(t)
	dir, err := Add(repo, "phu-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := Remove(repo, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("thư mục worktree chưa bị gỡ")
	}
	// Nhánh phải CÒN: việc agent làm nằm trong đó.
	if !strings.Contains(branches(t, repo), "sagent/phu-1") {
		t.Fatal("nhánh bị xoá theo worktree — mất việc của agent")
	}
}

// Thư mục đã bị xoá tay thì Remove vẫn phải dọn sổ git, không được báo lỗi.
func TestRemoveTolerantWhenDirGone(t *testing.T) {
	fakeHome(t)
	repo := gitRepo(t)
	dir, err := Add(repo, "phu-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := Remove(repo, dir); err != nil {
		t.Fatalf("thư mục đã mất thì Remove nên im lặng cho qua: %v", err)
	}
}

// Không phải git repo thì phải nói không, chứ đừng đoán bừa.
func TestRepoRootRejectsNonRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("máy không có git")
	}
	if _, ok := RepoRoot(t.TempDir()); ok {
		t.Fatal("thư mục trống mà báo là git repo")
	}
}
