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
					Message: "Xem các nhánh sagent/* rồi chọn bản để giữ. Bỏ qua thì không nhánh nào bị merge.",
				},
			},
		},

		// Một đội có phân vai: làm → kiểm thử → duyệt.
		"squad": {
			Name: "squad",
			Desc: "Agent làm → chạy test → người duyệt trước khi merge",
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
					Message: "Xem diff và kết quả test rồi quyết định có merge không.",
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
