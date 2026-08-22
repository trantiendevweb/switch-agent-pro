// Xin chỗ ở cổng trần đồng thời trước khi một bước `agent` bật phiên.
package flow

import (
	"context"

	"github.com/trantiendevweb/switch-agent-pro/internal/fleet"
)

// xinCho xin chỗ cho một bước `agent`, CHỜ tới khi có.
//
// Trả về thẻ giữ chỗ (có thể nil khi không áp trần) và lỗi. Lỗi chỉ có hai
// nguồn, và cả hai đều đáng làm bước hỏng: ctx bị huỷ, hoặc KẸT CỨNG — không
// còn chỗ mà cũng không bước nào đang chạy để trả chỗ ra. Trần chật thường thì
// KHÔNG ra lỗi, nó chỉ làm bước đứng chờ.
func (r *Runner) xinCho(ctx context.Context, s Step, muon int) (*fleet.The, error) {
	if r.Cong == nil {
		return nil, nil
	}
	// Phân giải hồ sơ PHẢI trùng từng ký tự với chỗ chạy thật (agentBridge:
	// `addr := fallback; if profile != "" { addr = ParseAddr(profile) }`). Lệch
	// một chút — cắt khoảng trắng ở một bên chẳng hạn — là cổng đi canh một địa
	// chỉ không ai chạy, mà chuyện đó hỏng trong im lặng: trần vẫn "hoạt động",
	// chỉ là đếm nhầm tài khoản. Nên KHÔNG TrimSpace ở đây.
	p := s.Profile
	if p == "" {
		p = r.DefaultProfile
	}
	if p == "" {
		// Bước chưa biết chạy bằng tài khoản nào. Không đoán một cái tên để có
		// cái mà đếm — để RunAgents báo đúng câu lỗi của nó ("đặt `profile`
		// trong flow hoặc truyền --profile"), rõ hơn bất cứ câu nào về trần.
		return nil, nil
	}
	return r.Cong.Xin(ctx, "bước "+s.ID, fleet.PhienTu(p), muon)
}
