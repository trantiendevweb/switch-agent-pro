package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Lượt fleet sau KHÔNG được xoá công việc của lượt trước.
//
// Lỗi thật, đo 18/08: `worktree add -B` đặt lại nhánh. Lần chạy #21, agent
// claude:tns commit 99 dòng lên `sagent/tns-1`; lượt fleet sau cùng tài khoản làm
// commit đó thành mồ côi — còn trong kho nhưng không thuộc nhánh nào, và
// `git log main..sagent/tns-1` hiện trống trơn như chưa ai làm gì.
func TestLuotSauKhongXoaViecLuotTruoc(t *testing.T) {
	repo := gitRepo(t)

	// Lượt 1: tạo worktree rồi commit như một agent làm việc thật.
	wt1, err := Add(repo, "tns-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt1, "viec.txt"), []byte("99 dong code"), 0o644); err != nil {
		t.Fatal(err)
	}
	chay(t, wt1, "add", ".")
	chay(t, wt1, "commit", "-m", "viec cua agent")

	bc1 := Xem(wt1, "main")
	if bc1.Commit != 1 {
		t.Fatalf("lượt 1 phải có 1 commit, được %d", bc1.Commit)
	}
	nhanh1 := bc1.Nhanh

	// Lượt 2: cùng tài khoản, chạy lại.
	wt2, err := Add(repo, "tns-1")
	if err != nil {
		t.Fatal(err)
	}
	bc2 := Xem(wt2, "main")

	// Việc cũ PHẢI còn, và phải còn trên MỘT NHÁNH (không mồ côi).
	out, err := run(repo, "rev-list", "--count", "main.."+nhanh1)
	if err != nil {
		t.Fatalf("nhánh %s biến mất sau lượt 2: %v", nhanh1, err)
	}
	if strings.TrimSpace(out) != "1" {
		t.Fatalf("VIỆC CỦA LƯỢT TRƯỚC BỊ XOÁ: nhánh %s giờ có %s commit, muốn 1",
			nhanh1, strings.TrimSpace(out))
	}
	// Và lượt 2 phải được nhánh KHÁC, không giẫm lên nhánh cũ.
	if bc2.Nhanh == nhanh1 {
		t.Fatalf("lượt 2 dùng lại đúng nhánh %s đang giữ việc chưa trộn", nhanh1)
	}
}

// Nhánh cũ RỖNG (không có việc) thì cứ dùng lại, đừng đẻ nhánh thừa.
func TestNhanhRongThiDungLai(t *testing.T) {
	repo := gitRepo(t)
	wt1, err := Add(repo, "may-1")
	if err != nil {
		t.Fatal(err)
	}
	ten1 := Xem(wt1, "main").Nhanh

	wt2, err := Add(repo, "may-1")
	if err != nil {
		t.Fatal(err)
	}
	if got := Xem(wt2, "main").Nhanh; got != ten1 {
		t.Fatalf("nhánh cũ rỗng mà vẫn đẻ nhánh mới: %s -> %s", ten1, got)
	}
}
