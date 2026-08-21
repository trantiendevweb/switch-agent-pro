package redaction

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// CHẶN KHOÁ LỌT VÀO FILE ĐƯỢC COMMIT.
//
// Tầng che ở internal/nhatky lo nhật ký — thứ KHÔNG nằm trong repo. Bài kiểm
// này lo phía còn lại: file thật sự được commit. Hai chỗ rò khác nhau và không
// chỗ nào che chỗ nào.
//
// Vì sao là bài kiểm Go chứ không phải git hook: hook nằm trong .git/, tức
// KHÔNG được clone theo, không nằm trong lịch sử, và tắt bằng `--no-verify`.
// Còn `go test ./...` là câu lệnh nghiệm thu mà mọi lượt đều chạy — người mới
// vào repo có nó ngay, không phải cài gì.
//
// Bài kiểm dùng chung bảng luật với tầng che (TimBiMat đọc `bang`). Viết một
// danh sách mẫu riêng cho bài kiểm thì hai danh sách sẽ lệch nhau trong im
// lặng: thêm luật cho nhật ký mà quên thêm cho commit, hoặc ngược lại.
//
// KHÔNG CÓ DANH SÁCH MIỄN TRỪ, và đó là chủ ý. Bí mật giả trong các file test
// đều được GHÉP LÚC CHẠY (xem hàm `gia`) nên bài kiểm này không thấy chúng.
// Mở một danh sách miễn trừ thì lần rò thật đầu tiên cũng sẽ được ai đó cho
// vào danh sách ấy — đúng lúc đang vội, đúng lúc nó quan trọng nhất.
func TestRepoKhongCoBiMat(t *testing.T) {
	goc := gocRepo(t)
	ds := fileTrongRepo(t, goc)
	if len(ds) < 10 {
		t.Fatalf("chỉ liệt kê được %d file — phép duyệt hỏng, và một bài kiểm "+
			"không duyệt được gì thì luôn xanh", len(ds))
	}

	soQuet, soBoQua := 0, 0
	for _, tuongDoi := range ds {
		duong := filepath.Join(goc, tuongDoi)
		st, err := os.Stat(duong)
		if err != nil || st.IsDir() {
			continue // file đã xoá khỏi cây làm việc nhưng git còn nhớ
		}
		// Trần 8 MB: file to hơn thế trong repo này là tài sản nhúng, và đọc
		// hết chúng làm bài kiểm chậm tới mức người ta bỏ chạy `go test`.
		if st.Size() > 8<<20 {
			soBoQua++
			continue
		}
		raw, err := os.ReadFile(duong)
		if err != nil {
			t.Errorf("%s: không đọc được: %v", tuongDoi, err)
			continue
		}
		// File nhị phân: bỏ qua. Không có "dòng" để chỉ, và mọi khớp regex trên
		// đó đều là ngẫu nhiên.
		if bytes.IndexByte(raw, 0) >= 0 {
			soBoQua++
			continue
		}
		soQuet++
		for _, p := range TimBiMat(string(raw)) {
			t.Errorf("%s:%d — nghi có bí mật (%s): %s",
				tuongDoi, p.Dong, p.Mau, p.Trich)
		}
	}
	t.Logf("đã quét %d file văn bản, bỏ qua %d file (nhị phân hoặc quá 8 MB)",
		soQuet, soBoQua)
}

// gocRepo đi ngược lên tới thư mục có go.mod.
func gocRepo(t *testing.T) string {
	t.Helper()
	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		cha := filepath.Dir(d)
		if cha == d {
			t.Fatal("không tìm thấy go.mod khi đi ngược lên từ thư mục hiện tại")
		}
		d = cha
	}
}

// fileTrongRepo trả về danh sách file mà git ĐANG THEO DÕI — tức đúng tập file
// sẽ đi vào commit. Quét cả cây làm việc thì vướng thư mục build, thư mục tạm
// và chính .git/ (nơi chứa bản gói của mọi blob, đọc ra là nhị phân).
//
// Không có git thì lùi về duyệt cây, chứ KHÔNG Skip: một bài kiểm bí mật tự bỏ
// qua chính mình là một bài kiểm luôn xanh.
func fileTrongRepo(t *testing.T, goc string) []string {
	t.Helper()
	c := exec.Command("git", "-C", goc, "ls-files", "-z")
	ra, err := c.Output()
	if err == nil {
		var ds []string
		for _, s := range strings.Split(string(ra), "\x00") {
			if s != "" {
				ds = append(ds, filepath.FromSlash(s))
			}
		}
		return ds
	}
	t.Logf("không chạy được `git ls-files` (%v) — lùi về duyệt cây thư mục", err)
	var ds []string
	_ = filepath.Walk(goc, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", "vendor", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if r, err := filepath.Rel(goc, p); err == nil {
			ds = append(ds, r)
		}
		return nil
	})
	return ds
}
