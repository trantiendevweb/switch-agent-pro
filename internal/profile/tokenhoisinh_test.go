package profile

import (
	"encoding/json"
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

// adapterDocThat kiểm token bằng cách ĐỌC FILE THẬT, không trả một cờ cứng.
//
// Cần bản này vì hai lỗi dưới đây chỉ lộ ra khi `HasToken` phân biệt được "thư
// mục có token dùng được" với "thư mục có file token rỗng" — mà `fakeAdapter`
// thì trả `hasToken` cố định cho mọi thư mục.
type adapterDocThat struct{ fakeAdapter }

func (adapterDocThat) HasToken(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, ".credentials.json"))
	if err != nil {
		return false
	}
	var f struct {
		Oauth struct {
			RefreshToken string `json:"refreshToken"`
		} `json:"claudeAiOauth"`
	}
	return json.Unmarshal(b, &f) == nil && f.Oauth.RefreshToken != ""
}

func ghiToken(t *testing.T, p, refresh string) {
	t.Helper()
	than := `{"claudeAiOauth":{"refreshToken":"` + refresh + `"}}`
	if refresh == "" {
		than = `{"claudeAiOauth":{}}` // đúng cái CLI để lại sau refresh thất bại
	}
	if err := os.WriteFile(p, []byte(than), 0o600); err != nil {
		t.Fatal(err)
	}
}

// LỖI THẬT, đo 21/08/2026 trên `claude:tns`: sau một lần refresh THẤT BẠI, CLI
// ghi đè file token của hồ sơ gốc thành bản RỖNG (không còn refreshToken) —
// trong khi bản clone vẫn giữ token còn hạn.
//
// Hai chỗ hỏng cùng lúc, và phải sửa cả hai mới cứu được:
//
//  1. `Clone` kiểm "đã đăng nhập chưa" TRƯỚC khi đồng bộ ngược, nên nó từ chối
//     ngay với câu "chạy /login" — bắt người dùng đăng nhập lại trong khi một
//     token còn sống nằm ngay trong bản clone bên cạnh.
//  2. `SyncBackTokens` chọn bản theo mtime. Việc ghi đè làm mtime của GỐC thành
//     mới nhất, nên clone bị coi là "cũ hơn" và bị bỏ qua. mtime chỉ nói "file
//     vừa bị ghi", KHÔNG nói "nội dung còn dùng được".
func TestCuuDuocKhiGocMatTokenMaCloneConToken(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := adapterDocThat{fakeAdapter{base: fakeBase, hasToken: true}}
	home := homeCuaTest(t)
	gocTok := filepath.Join(home, ".ai-accounts", "fake", "phu", ".credentials.json")

	ghiToken(t, gocTok, "token-goc")
	dirs, err := Clone(a, "phu", 1)
	if err != nil {
		t.Fatal(err)
	}
	cloneTok := filepath.Join(dirs[0], ".credentials.json")

	// Clone tự refresh (token mới), rồi hồ sơ gốc bị ghi đè thành RỖNG — và ghi
	// SAU, nên mtime của gốc mới hơn clone. Đây đúng thứ tự đã xảy ra thật.
	ghiToken(t, cloneTok, "token-moi-con-song")
	ghiToken(t, gocTok, "")
	sau := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(gocTok, sau, sau); err != nil {
		t.Fatal(err)
	}

	if _, err := Clone(a, "phu", 1); err != nil {
		t.Fatalf("từ chối trong khi clone còn token dùng được: %v", err)
	}
	got, err := os.ReadFile(gocTok)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "token-moi-con-song") {
		t.Errorf("không cứu được token từ clone về hồ sơ gốc.\n được: %s", got)
	}
}

// Ngược lại: gốc VÀ clone đều rỗng thì vẫn phải từ chối, và nói đúng câu cũ.
// Không có gì để cứu mà vẫn chạy tiếp thì agent bật lên rồi đăng nhập hụt.
func TestVanTuChoiKhiKhongConBanNaoCoToken(t *testing.T) {
	_, fakeBase := fakeHome(t)
	a := adapterDocThat{fakeAdapter{base: fakeBase, hasToken: true}}
	gocTok := filepath.Join(homeCuaTest(t), ".ai-accounts", "fake", "phu", ".credentials.json")

	ghiToken(t, gocTok, "token-goc")
	dirs, err := Clone(a, "phu", 1)
	if err != nil {
		t.Fatal(err)
	}
	ghiToken(t, filepath.Join(dirs[0], ".credentials.json"), "")
	ghiToken(t, gocTok, "")

	if _, err := Clone(a, "phu", 1); err == nil {
		t.Error("không bản nào còn token mà vẫn chạy tiếp")
	} else if !strings.Contains(err.Error(), "chưa đăng nhập") {
		t.Errorf("thông điệp không chỉ ra việc phải làm: %v", err)
	}
}
