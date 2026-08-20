package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// LỖI THẬT, đo 20/08/2026: `claude:phu` mất phiên giữa lượt chạy #47 với
// "OAuth session expired and could not be refreshed", và chủ dự án phải đăng
// nhập lại. Chuỗi sự kiện:
//
//  1. Lượt trước, bản clone tự refresh. Token mới nằm trong clone; hồ sơ gốc
//     giữ token cũ — mà nhà cung cấp đã vô hiệu nó khi cấp token mới.
//  2. `SyncBackTokens` CHỈ được gọi trong `ClonesClean`, tức chỉ khi người dùng
//     chạy `sagent clean`. Không ai chạy, nên token mới nằm im trong clone.
//  3. Lượt sau, `Clone` chép token GỐC (đã chết) ĐÈ lên clone (đang sống).
//  4. Clone refresh bằng token đã bị vô hiệu → thất bại → mất phiên.
//
// Bước 3 là chỗ hỏng, và bài này canh đúng nó: `Clone` không được hồi sinh một
// token cũ đè lên bản mới hơn.
//
// Chú ý phạm vi: bài này KHÔNG khẳng định nhà cung cấp có xoay vòng refresh
// token hay không — cái đó vẫn CHƯA ĐO. Nó chỉ khẳng định công refresh không bị
// đánh rơi, và đó là điều đúng bất kể nhà cung cấp làm gì.

func TestCloneKhongHoiSinhTokenCu(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := fakeAdapter{base: fakeBase, hasToken: true}

	// Lượt 1: tạo clone từ token gốc.
	dirs, err := Clone(a, "phu", 1)
	if err != nil {
		t.Fatal(err)
	}
	cloneTok := filepath.Join(dirs[0], ".credentials.json")

	// Bản clone tự refresh: token trong clone đổi, hồ sơ gốc KHÔNG biết.
	moi := []byte(`{"t":"token-moi-sau-refresh"}`)
	if err := os.WriteFile(cloneTok, moi, 0o600); err != nil {
		t.Fatal(err)
	}
	// mtime phải mới hơn bản gốc thật sự (Windows có độ phân giải thô).
	sau := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(cloneTok, sau, sau); err != nil {
		t.Fatal(err)
	}

	// Lượt 2: bật hạm đội lần nữa. ĐÂY là chỗ token cũ từng hồi sinh.
	if _, err := Clone(a, "phu", 1); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(cloneTok)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(moi) {
		t.Errorf("Clone hồi sinh token cũ đè lên bản mới hơn.\n muốn: %s\n được: %s", moi, got)
	}
	// Và hồ sơ gốc phải nhận được token mới — nếu không thì lần sau lại hỏng y hệt.
	goc, err := os.ReadFile(filepath.Join(homeCuaTest(t), ".ai-accounts", "fake", "phu", ".credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(goc) != string(moi) {
		t.Errorf("hồ sơ gốc không nhận token đã refresh — công refresh vẫn mất trắng.\n được: %s", goc)
	}
}

// homeCuaTest đọc lại HOME mà fakeHome vừa đặt.
func homeCuaTest(t *testing.T) string {
	t.Helper()
	h := os.Getenv("USERPROFILE")
	if h == "" {
		h = os.Getenv("HOME")
	}
	return h
}

// Trường hợp ngược lại: clone KHÔNG mới hơn thì không được đụng gì. Đồng bộ vô
// cớ sẽ đẻ ra một file .bak mỗi lần bật hạm đội, và mỗi file đó là một bản sao
// của token — tức là nhân bản bí mật ra đĩa mà không ai xin.
func TestKhongDongBoNguocKhiKhongCoGiMoi(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := fakeAdapter{base: fakeBase, hasToken: true}

	if _, err := Clone(a, "phu", 2); err != nil {
		t.Fatal(err)
	}
	baseDir := filepath.Join(homeCuaTest(t), ".ai-accounts", "fake", "phu")

	// Bật lại nhiều lần mà không refresh gì.
	for i := 0; i < 3; i++ {
		if _, err := Clone(a, "phu", 2); err != nil {
			t.Fatal(err)
		}
	}

	ents, err := os.ReadDir(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	var bak int
	for _, e := range ents {
		if strings.Contains(e.Name(), ".bak-") {
			bak++
		}
	}
	if bak > 0 {
		t.Errorf("đồng bộ ngược vô cớ: đẻ ra %d file .bak, mỗi file là một bản sao token", bak)
	}
}
