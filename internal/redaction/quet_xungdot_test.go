package redaction

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CHẶN VẠCH XUNG ĐỘT GIT LỌT VÀO FILE ĐƯỢC COMMIT.
//
// Vì sao có bài kiểm này: ngày 22/08 tìm thấy `docs/DO-LUONG.md` mang một khối
// xung đột chưa gỡ — `<<<<<<< HEAD` ở dòng 3063, `>>>>>>> sagent/phu-1` ở dòng
// cuối file — nằm im trong repo từ commit `5646f5f`. Nội dung hai vế đều đúng
// và không mất gì, nên **không có triệu chứng nào**: `go build`, `go vet`,
// `go test` đều xanh; file `.md` vẫn mở ra đọc được. Chỉ có ba dòng rác nằm
// giữa sổ đo lường trong nhiều giờ mà không ai thấy.
//
// Chi tiết đắt nhất: commit để lại nó có tiêu đề *"Merge sagent/phu-1: …sổ nợ
// đo lường này SẠCH"*. Lời khai của lượt trộn và trạng thái thật của file đi
// ngược nhau — đúng cái bệnh mà `docs/SO-NO-DO-LUONG.md` dựng ra để bắt
// ("tiêu đề commit không phải bằng chứng").
//
// Vì sao nằm cạnh `quet_repo_test.go` chứ không ở gói khác: phần khó và dễ
// hỏng của cả hai bài kiểm là **phép duyệt mọi file được git theo dõi**
// (`fileTrongRepo`). Tách sang gói khác thì phải chép lại phép duyệt đó, và
// hai bản chép sẽ lệch nhau trong im lặng — đúng lý do mà bài kiểm bên cạnh
// từ chối giữ hai danh sách mẫu.
//
// Vì sao là bài kiểm Go chứ không phải git hook: hook nằm trong `.git/`, không
// được clone theo, và tắt được bằng `--no-verify`. `go test ./...` là câu lệnh
// nghiệm thu mà mọi lượt đều chạy.
func TestRepoKhongCoVachXungDot(t *testing.T) {
	goc := gocRepo(t)
	ds := fileTrongRepo(t, goc)
	if len(ds) < 10 {
		t.Fatalf("chỉ liệt kê được %d file — phép duyệt hỏng, và một bài kiểm "+
			"không duyệt được gì thì luôn xanh", len(ds))
	}

	// Ghép lúc chạy, KHÔNG viết thẳng chuỗi bảy ký tự vào mã nguồn. Viết thẳng
	// thì chính file này khớp với luật của chính nó và bài kiểm luôn đỏ — cùng
	// một bẫy mà `designsystem_test` đã dính (viết `--error:` trong bình luận
	// cũng bị bắt là khai lại token).
	vach := []string{
		strings.Repeat("<", 7) + " ",
		strings.Repeat(">", 7) + " ",
		strings.Repeat("|", 7) + " ",
	}

	soQuet := 0
	var loi []string
	for _, tuongDoi := range ds {
		duong := filepath.Join(goc, tuongDoi)
		st, err := os.Stat(duong)
		if err != nil || st.IsDir() {
			continue // file đã xoá khỏi cây làm việc nhưng git còn nhớ
		}
		if st.Size() > 32<<20 {
			continue
		}
		noiDung, err := os.ReadFile(duong)
		if err != nil {
			continue
		}
		soQuet++

		for i, dong := range strings.Split(string(noiDung), "\n") {
			d := strings.TrimRight(dong, "\r")
			for _, v := range vach {
				// Phải khớp từ ĐẦU DÒNG. Git chỉ sinh vạch ở đầu dòng, còn
				// giữa dòng thì `>>>>>>>` là thứ tài liệu hoàn toàn có quyền
				// nhắc tới — kể cả file này.
				if strings.HasPrefix(d, v) {
					loi = append(loi, fmt.Sprintf("%s:%d: %s", tuongDoi, i+1, d))
				}
			}
		}
	}

	if soQuet < 10 {
		t.Fatalf("chỉ đọc được %d file trong %d mục — phép quét hỏng", soQuet, len(ds))
	}
	if len(loi) > 0 {
		t.Fatalf("còn %d vạch xung đột git chưa gỡ trong file đã commit "+
			"(quét %d file):\n%s\n\nGỡ vạch rồi kiểm lại: cả hai vế thường đều "+
			"là việc thật, giữ cả hai trước khi xoá dòng nào.",
			len(loi), soQuet, strings.Join(loi, "\n"))
	}
}
