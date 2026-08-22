package flow

// Builtin là ba flow mẫu dựng sẵn (MASTER-PLAN Pha 3).
//
// Có sẵn để `sagent flow list` không rỗng ngay từ đầu, và để người dùng có mẫu
// mà sửa. Đặt cùng tên trong flows.toml của bạn là đè được — không cần xin phép.
func Builtin() map[string]Flow {
	return map[string]Flow{
		// Nhiều agent giải cùng một bài, mỗi bản một nhánh, rồi người chọn.
		"fanout": {
			Name: "fanout",
			Desc: "N agent cùng giải một bài trên N nhánh riêng, rồi bạn chọn bản tốt nhất",
			Vars: map[string]string{
				"task":   "Đọc repo rồi đề xuất một cải tiến nhỏ, làm luôn.",
				"copies": "3",
			},
			Steps: []Step{
				{
					ID: "giai", Type: TypeAgent,
					Prompt: "{{task}}", Copies: 3, Worktree: true,
					TimeoutSec: 1800,
				},
				{
					ID: "chon", Type: TypeApprove, Needs: []string{"giai"},
					Message: "Xem các nhánh sagent/* rồi quyết định giữ bản nào. " +
						"Duyệt hay từ chối đều KHÔNG merge gì — xem bước `chi-lenh` ngay sau.",
				},
				// VÌ SAO CÓ BƯỚC NÀY, và vì sao nó chỉ IN CHỮ (sửa 22/08).
				//
				// Hai lỗi cùng lúc ở bản cũ, và lỗi thứ hai nặng hơn:
				//
				//  1. `sagent flow validate` cảnh báo "bước approve này không chặn
				//     bước nào" cho CẢ `fanout` LẪN `squad` — tức hai trên ba flow
				//     mẫu dựng sẵn kêu ngay lần chạy đầu của người mới.
				//  2. Lời hứa SAI: câu cũ "bỏ qua thì không nhánh nào bị merge" đọc
				//     ra là DUYỆT THÌ CÓ MERGE. Không hề — sau `chon` không có bước
				//     nào cả. Người duyệt bấm xong, tưởng đã giữ bản mình chọn, và
				//     nhánh vẫn nằm nguyên đó.
				//
				// CỐ Ý KHÔNG tự chạy `git merge` ở đây. Đó là quyết định của chủ dự
				// án chứ không phải của một flow mẫu, và `internal/flow/merge.go` đã
				// ghi rõ đường đúng: một bước `shell` chạy `git merge` đứng SAU một
				// bước `approve`, để người duyệt NHÌN THẤY đúng lệnh sắp chạy. Flow
				// mẫu không biết nhánh nào đáng giữ, nên nó in lệnh ra thay vì đoán.
				{
					ID: "chi-lenh", Type: TypeNotify, Needs: []string{"chon"},
					Message: "Đã duyệt. Giữ bản bạn chọn bằng: git merge --no-ff <nhánh>. " +
						"Xem có những nhánh nào: git branch --list sagent/*",
				},
			},
		},

		// Một đội có phân vai: làm → kiểm thử → duyệt.
		"squad": {
			Name: "squad",
			Desc: "Agent làm → chạy test → người duyệt, rồi công cụ in ra đúng lệnh merge để bạn tự chạy",
			Vars: map[string]string{
				"task": "Sửa lỗi được mô tả trong issue mới nhất.",
			},
			Steps: []Step{
				{
					ID: "lam", Type: TypeAgent,
					Prompt: "{{task}}", Copies: 1, Worktree: true, TimeoutSec: 2400,
				},
				{
					ID: "kiem-thu", Type: TypeShell, Needs: []string{"lam"},
					Run: []string{"go", "test", "./..."}, TimeoutSec: 600,
					// Test hỏng thì vẫn đi tiếp để người duyệt thấy kết quả thật.
					OnFailure: OnFailContinue,
				},
				{
					ID: "duyet", Type: TypeApprove, Needs: []string{"kiem-thu"},
					Message: "Xem diff và kết quả test rồi quyết định. " +
						"Duyệt hay từ chối đều KHÔNG merge gì — xem bước `chi-lenh` ngay sau.",
				},
				// Cùng lý do với `fanout.chi-lenh` ở trên: mô tả cũ của flow này hứa
				// "người duyệt trước khi merge", mà sau bước duyệt KHÔNG CÓ bước
				// merge nào. Bước này làm lời hứa đó thành đúng — bằng cách hạ lời
				// hứa xuống cho khớp việc thật, chứ không bằng cách cho một flow mẫu
				// quyền ghi đè cây mã của người dùng.
				{
					ID: "chi-lenh", Type: TypeNotify, Needs: []string{"duyet"},
					Message: "Đã duyệt. Merge bằng: git merge --no-ff <nhánh của bước `lam`>. " +
						"Xem có những nhánh nào: git branch --list sagent/*",
				},
			},
		},

		// Danh sách việc độc lập, chạy theo trần song song của dự án.
		"agents": {
			Name: "agents",
			Desc: "Bật một đội agent headless chạy song song, mỗi agent một worktree",
			Vars: map[string]string{
				"task":   "Rà một phần của repo và báo cáo vấn đề tìm được.",
				"copies": "4",
			},
			Steps: []Step{
				{
					ID: "chay", Type: TypeAgent,
					Prompt: "{{task}}", Copies: 4, Worktree: true, TimeoutSec: 3600,
				},
				{
					ID: "bao", Type: TypeNotify, Needs: []string{"chay"},
					Message: "Đội agent đã chạy xong — xem log từng phiên trong dashboard.",
				},
			},
		},
	}
}
