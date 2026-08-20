// Package aiapi là ĐƯỜNG THỨ HAI của dự án: gọi thẳng AI API, không qua CLI agent.
//
// Khác đường CLI ở bản chất: đường CLI mượn tài khoản đăng nhập sẵn và tiêu hạn
// mức của gói thuê bao; đường này dùng API key và tiêu tiền theo token. Vì vậy
// mọi lời gọi ở đây đều trả về `Usage` — không đo được thì không biết đang tiêu gì.
//
// Hai luật giữ từ MASTER-PLAN mục 0, không được lách:
//
//  1. FILE CẤU HÌNH KHÔNG BAO GIỜ CHỨA SECRET. Route chỉ ghi `key_id`; key thật
//     nằm ở ~/.ai-accounts/api-keys/<id>.key, trong kho đã siết ACL.
//  2. Lỗi của nhà cung cấp phải giữ NGUYÊN VĂN. Họ trả kèm request id — vứt nó đi
//     là vứt luôn thứ duy nhất dùng được khi phải hỏi lại họ.
package aiapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// Route là một đường gọi API đã cấu hình. KHÔNG chứa key.
type Route struct {
	Ten     string // tên route, ví dụ "grok"
	BaseURL string // ví dụ https://modelapi.vn/v1
	Model   string // ví dụ grok-4.5
	KeyID   string // tên file trong ~/.ai-accounts/api-keys, KHÔNG phải key
}

// Usage là phần đếm token. Nhà cung cấp nào cũng trả, và đây là thứ cho biết
// một lời gọi tốn bao nhiêu.
type Usage struct {
	Vao  int `json:"prompt_tokens"`
	Ra   int `json:"completion_tokens"`
	Tong int `json:"total_tokens"`
}

// KetQua là kết quả một lời gọi.
//
// Bốn trường cuối nói về CHUYỆN ĐÃ XẢY RA trên đường đi, không phải nội dung câu
// trả lời. Có chúng vì khi tầng trên tự chuyển sang route dự phòng, người gọi
// phải biết: câu này ai trả lời, route chính hỏng vì gì, và request id của lần
// hỏng đó là bao nhiêu. Trước đây những thứ đó chỉ được `bus.Warnf` — ai gọi qua
// web hay CLI mà không nghe bus thì mất sạch.
type KetQua struct {
	NoiDung string
	Model   string
	Usage   Usage // usage của route THẬT SỰ trả lời, không phải của route hỏng
	Mat     time.Duration

	// DaStreaming: lượt này đi đường stream. ThieuUsage: nhà cung cấp KHÔNG trả
	// `usage`, nên Usage ở trên là số 0 vì CHƯA ĐO chứ không phải vì miễn phí.
	//
	// Phải có cờ riêng chứ không suy từ `Usage.Tong == 0`: một lượt thật cũng có
	// thể tốn 0 token nếu hỏng sớm, và gộp hai chuyện đó lại thì sổ chi phí mất
	// khả năng phân biệt "không tốn gì" với "không đếm được".
	DaStreaming bool
	ThieuUsage  bool

	// Route là tên route đã trả lời câu này.
	Route string
	// DaThu là tên mọi route đã gọi, theo thứ tự, kể cả route hỏng.
	DaThu []string
	// RouteChinh là route đã hỏng khiến phải chuyển. Rỗng nếu không chuyển.
	RouteChinh string
	// LoiChinh là lỗi NGUYÊN VĂN của route chính, kèm request id của nhà cung
	// cấp. Rỗng nếu không chuyển route.
	LoiChinh string
}

// DaChuyenRoute cho biết câu trả lời này đến từ route dự phòng.
func (k KetQua) DaChuyenRoute() bool { return k.RouteChinh != "" }

// LoiAPI là lỗi của một lời gọi, giữ đủ thứ cần để tầng trên QUYẾT ĐỊNH có nên
// chuyển route hay không — chứ không chỉ một chuỗi để in ra.
//
// `Chi` là nguyên văn (đã kèm tên route và mã HTTP), theo luật 2 ở đầu file.
type LoiAPI struct {
	Route  string // route đã gọi
	Status int    // mã HTTP; 0 khi hỏng trước lúc có phản hồi
	Chi    string // nguyên văn
	Nguoi  bool   // lỗi từ phía người dùng
}

func (e *LoiAPI) Error() string { return e.Chi }

// LoiNguoiDung cho biết lỗi này do phía NGƯỜI DÙNG: key sai hoặc hết quyền
// (401/403), không đọc được key, prompt rỗng.
//
// Vì sao phải phân biệt: chuyển sang route dự phòng chỉ lặp lại đúng cái sai đó
// ở nhà cung cấp thứ hai. Nó không cứu được gì, mà tốn thêm một lượt gọi, nhân
// đôi thời gian chờ, và làm lỗi trả về dài gấp đôi trong khi nguyên nhân thật
// vẫn là câu đầu tiên.
func LoiNguoiDung(err error) bool {
	var l *LoiAPI
	return errors.As(err, &l) && l.Nguoi
}

// loiNguoi/loiMay dựng LoiAPI cho hai nhánh.
func loiNguoi(route string, status int, dinh string, a ...any) *LoiAPI {
	return &LoiAPI{Route: route, Status: status, Chi: fmt.Sprintf(dinh, a...), Nguoi: true}
}
func loiMay(route string, status int, dinh string, a ...any) *LoiAPI {
	return &LoiAPI{Route: route, Status: status, Chi: fmt.Sprintf(dinh, a...)}
}

// KeysDir là nơi giữ API key — trong kho đã siết ACL, NGOÀI repo.
func KeysDir() string { return filepath.Join(paths.AccountsRoot(), "api-keys") }

func keyPath(id string) string { return filepath.Join(KeysDir(), id+".key") }

// docKey đọc key theo ID.
//
// Không nhận đường dẫn từ ngoài: `id` phải là tên trần, không có dấu phân cách.
// Tên đến từ file cấu hình mà người khác có thể sửa, và hàm này đọc file bí mật —
// đúng lớp lỗi đã nổ một lần với tên hồ sơ (xem docs/DO-LUONG.md).
func docKey(id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("route thiếu key_id")
	}
	if strings.ContainsAny(id, `/\:`) || id == "." || id == ".." {
		return "", fmt.Errorf("key_id %q không hợp lệ — chỉ dùng chữ, số, '-', '_'", id)
	}
	b, err := os.ReadFile(keyPath(id))
	if err != nil {
		return "", fmt.Errorf("không đọc được key %q: %w\n     đặt bằng: sagent api key %s", id, err, id)
	}
	k := strings.TrimSpace(string(b))
	if k == "" {
		return "", fmt.Errorf("key %q rỗng", id)
	}
	return k, nil
}

type yeuCau struct {
	Model    string    `json:"model"`
	Messages []tinNhan `json:"messages"`
}
type tinNhan struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type phanHoi struct {
	Model   string `json:"model"`
	Choices []struct {
		Message tinNhan `json:"message"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

// Goi gửi một prompt và trả về câu trả lời.
//
// Không streaming ở bản này — streaming là việc riêng, và làm nửa vời thì mất
// usage. Đo trước, thêm sau.
//
// Mọi lỗi trả về đều là *LoiAPI, để tầng trên phân biệt được "thử route khác có
// thể cứu" với "thử route khác chỉ tốn thêm tiền" — xem LoiNguoiDung.
func Goi(ctx context.Context, r Route, prompt string) (KetQua, error) {
	var kq KetQua
	// Prompt rỗng chặn TRƯỚC khi chạm mạng: nhà cung cấp sẽ trả 400, và một lượt
	// gọi hỏng vẫn có thể bị tính tiền. Đây cũng là lỗi của người dùng, nên route
	// dự phòng không cứu được.
	if strings.TrimSpace(prompt) == "" {
		return kq, loiNguoi(r.Ten, 0, "prompt rỗng — không gửi gì cho %s", r.Ten)
	}
	key, err := docKey(r.KeyID)
	if err != nil {
		return kq, loiNguoi(r.Ten, 0, "%s: %s", r.Ten, err.Error())
	}
	// Route khai thiếu là lỗi CẤU HÌNH CỦA RIÊNG ROUTE ĐÓ — route dự phòng khai
	// đủ thì vẫn chạy được. Nên đây KHÔNG phải lỗi người dùng theo nghĩa chặn
	// fallback.
	if r.BaseURL == "" || r.Model == "" {
		return kq, loiMay(r.Ten, 0, "route %q thiếu base_url hoặc model", r.Ten)
	}

	body, err := json.Marshal(yeuCau{Model: r.Model, Messages: []tinNhan{{Role: "user", Content: prompt}}})
	if err != nil {
		return kq, loiMay(r.Ten, 0, "%s: %s", r.Ten, err.Error())
	}
	url := strings.TrimRight(r.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return kq, loiMay(r.Ten, 0, "%s: %s", r.Ten, err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	bat := time.Now()
	resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return kq, loiMay(r.Ten, 0, "gọi %s hỏng: %s", url, err.Error())
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if resp.StatusCode != http.StatusOK {
		// GIỮ NGUYÊN VĂN thân lỗi. Nhà cung cấp trả kèm request id trong đó, và
		// đó là thứ duy nhất dùng được khi phải hỏi lại họ. Rút gọn thành "lỗi
		// 400" là vứt mất nó.
		e := loiMay(r.Ten, resp.StatusCode, "%s trả HTTP %d: %s",
			r.Ten, resp.StatusCode, strings.TrimSpace(string(raw)))
		// 401/403 = key sai hoặc hết quyền. Route dự phòng dùng key KHÁC nhưng
		// cái sai ở đây là key của route NÀY: nếu người dùng gõ nhầm key thì họ
		// cần thấy đúng câu đó, không phải một câu ghép hai lỗi của hai nhà cung
		// cấp mà nguyên nhân thật bị chôn ở dòng đầu.
		e.Nguoi = resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden
		return kq, e
	}
	var ph phanHoi
	if err := json.Unmarshal(raw, &ph); err != nil {
		return kq, loiMay(r.Ten, resp.StatusCode, "%s trả JSON không đọc được: %s", r.Ten, err.Error())
	}
	if len(ph.Choices) == 0 {
		return kq, loiMay(r.Ten, resp.StatusCode, "%s trả về 0 lựa chọn", r.Ten)
	}
	kq = KetQua{
		NoiDung: ph.Choices[0].Message.Content,
		Model:   ph.Model,
		Usage:   ph.Usage,
		Mat:     time.Since(bat),
		Route:   r.Ten,
		DaThu:   []string{r.Ten},
	}
	return kq, nil
}
