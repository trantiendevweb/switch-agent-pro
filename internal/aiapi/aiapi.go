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
type KetQua struct {
	NoiDung string
	Model   string
	Usage   Usage
	Mat     time.Duration
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
func Goi(ctx context.Context, r Route, prompt string) (KetQua, error) {
	var kq KetQua
	key, err := docKey(r.KeyID)
	if err != nil {
		return kq, err
	}
	if r.BaseURL == "" || r.Model == "" {
		return kq, fmt.Errorf("route %q thiếu base_url hoặc model", r.Ten)
	}

	body, err := json.Marshal(yeuCau{Model: r.Model, Messages: []tinNhan{{Role: "user", Content: prompt}}})
	if err != nil {
		return kq, err
	}
	url := strings.TrimRight(r.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return kq, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	bat := time.Now()
	resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return kq, fmt.Errorf("gọi %s hỏng: %w", url, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if resp.StatusCode != http.StatusOK {
		// GIỮ NGUYÊN VĂN thân lỗi. Nhà cung cấp trả kèm request id trong đó, và
		// đó là thứ duy nhất dùng được khi phải hỏi lại họ. Rút gọn thành "lỗi
		// 400" là vứt mất nó.
		return kq, fmt.Errorf("%s trả HTTP %d: %s", r.Ten, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var ph phanHoi
	if err := json.Unmarshal(raw, &ph); err != nil {
		return kq, fmt.Errorf("%s trả JSON không đọc được: %w", r.Ten, err)
	}
	if len(ph.Choices) == 0 {
		return kq, fmt.Errorf("%s trả về 0 lựa chọn", r.Ten)
	}
	kq = KetQua{
		NoiDung: ph.Choices[0].Message.Content,
		Model:   ph.Model,
		Usage:   ph.Usage,
		Mat:     time.Since(bat),
	}
	return kq, nil
}
