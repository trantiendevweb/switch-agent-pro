package dash

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	cookieName    = "sagent_session"
	sessionMaxAge = 12 * time.Hour
)

// sessions giữ các phiên đăng nhập trong bộ nhớ.
//
// Cố ý KHÔNG lưu ra đĩa: khởi động lại server thì mọi người phải đăng nhập lại.
// Với một công cụ chạy trên máy cá nhân thì đó là hành vi đúng — an toàn hơn là
// tiện, và cũng chẳng ai muốn phiên đăng nhập sống lâu hơn tiến trình.
type sessions struct {
	mu sync.Mutex
	m  map[string]time.Time // id -> hết hạn
}

func newSessions() *sessions { return &sessions{m: map[string]time.Time{}} }

func (s *sessions) create() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	s.mu.Lock()
	s.m[id] = time.Now().Add(sessionMaxAge)
	// dọn phiên hết hạn nhân tiện, khỏi cần goroutine riêng
	now := time.Now()
	for k, exp := range s.m {
		if now.After(exp) {
			delete(s.m, k)
		}
	}
	s.mu.Unlock()
	return id
}

func (s *sessions) valid(id string) bool {
	if id == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.m[id]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.m, id)
		return false
	}
	return true
}

func (s *sessions) drop(id string) {
	s.mu.Lock()
	delete(s.m, id)
	s.mu.Unlock()
}

// setCookie đặt cookie phiên. HttpOnly để JS không đọc được (giảm thiệt hại nếu
// có XSS); SameSite=Lax để trình duyệt không gửi kèm khi trang khác POST sang.
//
// Secure chỉ bật khi ĐANG chạy TLS. Bật vô điều kiện thì trên http://127.0.0.1
// trình duyệt sẽ vứt cookie đi và không ai đăng nhập được — "an toàn hơn" kiểu
// đó chỉ làm người ta tắt bảo mật cho xong.
func setCookie(w http.ResponseWriter, id string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionMaxAge.Seconds()),
	})
}

func clearCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", HttpOnly: true, Secure: secure,
		SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}

func cookieID(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
