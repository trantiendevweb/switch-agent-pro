package workspace

import (
	"errors"
	"strconv"
	"strings"
)

// BangChung là thứ ĐO ĐƯỢC về việc một agent đã làm trong worktree của nó.
//
// Có vì lời agent kể không đáng tin. Lần chạy #21 (18/08): bước `tho-2` trả về
// "I am waiting for `go test ./...` to complete", được đánh dấu `done`, và cả
// flow báo `completed` — trong khi nhánh `sagent/may-1` KHÔNG có commit nào.
// Không cách nào biết điều đó nếu chỉ đọc chữ agent in ra.
type BangChung struct {
	Nhanh   string // tên nhánh của worktree
	Commit  int    // số commit đi trước nhánh gốc
	Ban     bool   // còn thay đổi chưa commit
	KhongRo bool   // không đọc được (không phải repo, git lỗi…)
}

// Xem đọc bằng chứng từ một thư mục worktree. `goc` là nhánh nền để so, thường
// là nhánh mặc định của repo.
func Xem(dir, goc string) BangChung {
	bc := BangChung{KhongRo: true}
	nhanh, err := run(dir, "branch", "--show-current")
	if err != nil {
		return bc
	}
	bc.Nhanh = strings.TrimSpace(nhanh)
	bc.KhongRo = false

	// `goc..HEAD` = commit CÓ ở nhánh này mà chưa có ở gốc. Dùng hai chấm chứ
	// không phải ba: ba chấm là đối xứng, sẽ đếm cả commit mới của nhánh gốc.
	out, err := run(dir, "rev-list", "--count", goc+"..HEAD")
	if err != nil {
		bc.KhongRo = true
		return bc
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		bc.KhongRo = true
		return bc
	}
	bc.Commit = n
	bc.Ban = IsDirty(dir)
	return bc
}

// MotDong tóm bằng chứng thành một dòng để nhét vào kết quả của bước.
func (b BangChung) MotDong() string {
	if b.KhongRo {
		return "không đọc được trạng thái git của worktree"
	}
	var sb strings.Builder
	sb.WriteString("nhánh " + b.Nhanh + ": ")
	if b.Commit == 0 {
		sb.WriteString("KHÔNG có commit nào")
	} else {
		sb.WriteString(strconv.Itoa(b.Commit) + " commit")
	}
	if b.Ban {
		sb.WriteString(", còn thay đổi CHƯA commit")
	}
	return sb.String()
}

// NhanhMacDinh đoán nhánh nền của repo. Không phải repo nào cũng dùng `main`;
// đoán sai thì số commit đếm ra vô nghĩa, mà lại trông rất thuyết phục.
//
// Thứ tự: hỏi origin/HEAD trước (nguồn chuẩn nhất), rồi mới thử tên quen thuộc.
func NhanhMacDinh(repoRoot string) (string, error) {
	if out, err := run(repoRoot, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		if n := strings.TrimSpace(out); n != "" {
			return strings.TrimPrefix(n, "origin/"), nil
		}
	}
	for _, ten := range []string{"main", "master"} {
		if _, err := run(repoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+ten); err == nil {
			return ten, nil
		}
	}
	return "", errKhongRoNhanh
}

var errKhongRoNhanh = errors.New("không xác định được nhánh mặc định của repo")
