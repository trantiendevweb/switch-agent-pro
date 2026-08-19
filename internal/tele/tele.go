// Package tele báo về Telegram khi một lượt chạy flow có sự cố.
//
// Vì sao cần: bốn mặt điều khiển (terminal, dashboard 2D, workflow board, 3D)
// đều phải MỞ RA mới thấy chuyện gì đang xảy ra. Mà một flow chạy hàng chục
// phút — không ai ngồi nhìn suốt. Bước hỏng lúc 2 giờ sáng, hay lượt chạy dừng
// chờ duyệt rồi đứng im tới trưa, là mất trắng khoảng thời gian đó. Telegram là
// mặt DUY NHẤT tự tìm tới người dùng thay vì đợi người dùng tìm tới nó.
//
// Ba luật của gói này, không được lách:
//
//  1. TOKEN KHÔNG NẰM TRONG REPO. Nó ở ~/.ai-accounts/telegram.json, cùng kho đã
//     siết ACL với dash-auth.json và api-keys/. Nhét token bot vào mã nguồn của
//     một dự án mã nguồn mở nghĩa là ai clone về cũng nhắn được vào nhóm của bạn.
//  2. CHƯA CẤU HÌNH THÌ IM LẶNG TUYỆT ĐỐI. Báo tin là việc phụ; nó không được
//     phép làm hỏng, làm chậm hay làm dừng lượt chạy. Mọi lỗi gửi tin chết ở
//     trong gói này, không bao giờ nổi lên tới bộ thực thi flow.
//  3. KHÔNG SPAM. Chỉ bốn chuyện đáng làm điện thoại rung: bước hỏng, lượt chạy
//     hỏng, lượt chạy chờ duyệt, lượt chạy xong. Mọi event khác bỏ qua — báo tin
//     mà báo hết thì người ta tắt thông báo, và thế là mất luôn cái đáng báo.
package tele

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/acl"
	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// APIMacDinh là gốc API Telegram công cộng.
const APIMacDinh = "https://api.telegram.org"

// Config là thông tin bot. KHÔNG BAO GIỜ nằm trong repo — xem ConfigPath.
type Config struct {
	Token  string `json:"token"`
	ChatID string `json:"chat_id"`

	// API là gốc API. Rỗng = APIMacDinh. Có trường này vì hai lý do thật: người
	// dùng có thể chạy bot-api server riêng, và test dựng được API giả bằng
	// httptest thay vì bắn tin thật ra Internet.
	API string `json:"api,omitempty"`
}

// ConfigPath là nơi giữ token bot — trong kho hồ sơ của người dùng, NGOÀI repo.
func ConfigPath() string { return filepath.Join(paths.AccountsRoot(), "telegram.json") }

// Load đọc cấu hình. Trả nil nghĩa là CHƯA CẤU HÌNH và phải im lặng.
//
// Thiếu token hay thiếu chat id cũng tính là chưa cấu hình: gửi nửa vời chỉ đẻ
// ra lỗi 400 lặp lại mỗi lượt chạy, mà không ai đọc được tin nào.
func Load() *Config {
	b, err := os.ReadFile(ConfigPath())
	if err != nil {
		return nil
	}
	var c Config
	if json.Unmarshal(b, &c) != nil {
		return nil
	}
	c.Token = strings.TrimSpace(c.Token)
	c.ChatID = strings.TrimSpace(c.ChatID)
	if c.Token == "" || c.ChatID == "" {
		return nil
	}
	return &c
}

// Save ghi cấu hình ra đĩa, quyền siết như mọi bí mật khác trong kho.
func Save(c Config) error {
	c.Token = strings.TrimSpace(c.Token)
	c.ChatID = strings.TrimSpace(c.ChatID)
	if c.Token == "" {
		return errors.New("thiếu token bot — xin ở @BotFather trên Telegram")
	}
	if c.ChatID == "" {
		return errors.New("thiếu chat id — nhắn cho bot một câu rồi mở /bot<token>/getUpdates để xem")
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ConfigPath()), 0o755); err != nil {
		return err
	}
	// 0o600 KHÔNG bảo vệ file này trên Windows (đã đo, xem internal/acl) — siết
	// cả thư mục thì mọi thứ bên trong mới thật sự kín.
	_ = acl.Restrict(filepath.Dir(ConfigPath()))
	tmp := ConfigPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ConfigPath())
}

// Bao là bộ gửi tin. Con trỏ nil = CHƯA CẤU HÌNH, và mọi phương thức trên nil
// đều là việc rỗng — nhờ vậy nơi gọi không phải rải `if` khắp nơi để giữ luật 2.
type Bao struct {
	cfg Config
	hc  *http.Client
}

// New dựng bộ gửi tin từ một cấu hình có sẵn.
func New(c Config) *Bao {
	if c.API == "" {
		c.API = APIMacDinh
	}
	// Có hạn giờ: Telegram treo thì lượt chạy vẫn phải đi tiếp. Không đặt hạn là
	// giao số phận của flow cho đường mạng.
	return &Bao{cfg: c, hc: &http.Client{Timeout: 10 * time.Second}}
}

// Mo dựng bộ gửi tin từ cấu hình trên đĩa; nil nếu chưa cấu hình.
func Mo() *Bao {
	c := Load()
	if c == nil {
		return nil
	}
	return New(*c)
}

// DaCauHinh cho biết có gửi được tin không, để các mặt hiện trạng thái.
func (b *Bao) DaCauHinh() bool { return b != nil }

// ChatID trả về đích nhận tin. Cố ý KHÔNG có hàm nào trả token ra ngoài: token
// đi tới dashboard là token đi vào lịch sử trình duyệt và ảnh chụp màn hình.
func (b *Bao) ChatID() string {
	if b == nil {
		return ""
	}
	return b.cfg.ChatID
}

type traLoi struct {
	OK    bool   `json:"ok"`
	MoTa  string `json:"description"`
	MaLoi int    `json:"error_code"`
}

// Gui gửi một tin. Trên bộ gửi nil thì không làm gì và không báo lỗi.
func (b *Bao) Gui(ctx context.Context, text string) error {
	if b == nil {
		return nil
	}
	body, err := json.Marshal(map[string]any{
		"chat_id": b.cfg.ChatID,
		"text":    text,
		// Tin nhắn chứa lệnh `sagent ...`; để Telegram tự dựng preview link chỉ
		// làm tin dài thêm mà không thêm thông tin nào.
		"disable_web_page_preview": true,
	})
	if err != nil {
		return err
	}
	url := strings.TrimRight(b.cfg.API, "/") + "/bot" + b.cfg.Token + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := b.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var tl traLoi
	_ = json.NewDecoder(res.Body).Decode(&tl)
	if res.StatusCode/100 != 2 || !tl.OK {
		// Giữ NGUYÊN VĂN lời Telegram: "chat not found" và "bot was blocked by
		// the user" là hai bệnh khác hẳn nhau, tự viết lại là mất phân biệt.
		if tl.MoTa != "" {
			return fmt.Errorf("Telegram từ chối (%d): %s", res.StatusCode, tl.MoTa)
		}
		return fmt.Errorf("Telegram trả mã %d", res.StatusCode)
	}
	return nil
}

// Thu gửi một tin thử, để người dùng biết đường báo tin đã thông TRƯỚC khi
// trông cậy vào nó lúc nửa đêm.
func (b *Bao) Thu(ctx context.Context) error {
	if b == nil {
		return errors.New("chưa cấu hình Telegram — đặt bằng: sagent tele --set-token <token> --chat <id>")
	}
	return b.Gui(ctx, "Switch-Agent-Pro — tin thử\n"+
		"Đường báo tin đã thông. Từ giờ máy sẽ nhắn khi lượt chạy flow hỏng, chờ duyệt hoặc xong.\n"+
		"Xem lịch sử: sagent flow runs")
}

// Nghe đăng ký vào bus và gửi tin cho những chuyện đáng đánh thức người dùng.
//
// Chưa cấu hình thì KHÔNG đăng ký gì cả — không tốn một kênh nào của bus, và
// tuyệt đối không có đường nào để việc báo tin chạm vào lượt chạy.
func Nghe(bus *events.Bus) func() { return Mo().Nghe(bus) }

// Nghe (trên một bộ gửi cụ thể) trả về hàm dừng.
func (b *Bao) Nghe(bus *events.Bus) func() {
	if b == nil || bus == nil {
		return func() {}
	}
	// Đệm rộng có chủ đích: goroutine dưới đây có lúc đang chờ mạng, mà
	// Bus.Publish KHÔNG BAO GIỜ chặn — người nghe chậm thì mất event. Rộng tay ở
	// đây rẻ hơn nhiều so với đánh rơi đúng cái tin báo hỏng.
	ch, huy := bus.Subscribe(256)
	xong := make(chan struct{})
	go func() {
		defer close(xong)
		for e := range ch {
			text := TinNhan(e)
			if text == "" {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			// Lỗi gửi tin CHẾT Ở ĐÂY. Không có đường nào để nó nổi lên lượt chạy.
			_ = b.Gui(ctx, text)
			cancel()
		}
	}()
	return func() {
		huy()
		select {
		case <-xong:
		case <-time.After(20 * time.Second):
			// Mạng treo thì bỏ, không giữ cả tiến trình lại vì một tin nhắn.
		}
	}
}
