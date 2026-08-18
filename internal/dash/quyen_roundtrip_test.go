package dash

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// Cờ tự-duyệt-quyền phải SỐNG SÓT vòng lưu → đọc lại của mặt web.
//
// Bảng vẽ nạp flow, người ta kéo một node rồi bấm Lưu — nếu cờ rơi ở bất kỳ
// khâu nào thì flow im lặng mất quyền: agent không đọc nổi file nào, mà file
// flows.toml trông vẫn bình thường. Bản đầu của trình soạn thảo đã rơi đúng chỗ
// này (danh sách trường lúc nạp thiếu tuDuyetQuyen).
//
// Chú ý cái bẫy tên: LƯU thì Flow.Steps mang thẻ json "step" (số ít, theo TOML),
// nhưng ĐỌC LẠI thì handler trả về khoá "steps". Lệch tên giữa hai chiều.
func TestCoQuyenSongSotVongLuuVaDocLai(t *testing.T) {
	s := newTestServer(t)
	dir := t.TempDir()
	t.Chdir(dir)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	body := `{"name":"thu","step":[
		{"id":"co","type":"agent","profile":"claude:phu","prompt":"việc","tuDuyetQuyen":true},
		{"id":"khong","type":"agent","profile":"grok:api","prompt":"soi","needs":["co"]}
	]}`
	r := httptest.NewRequest("POST", "/api/flow/save", strings.NewReader(body))
	r.Host = "127.0.0.1:4600"
	r.Header.Set("Origin", "http://127.0.0.1:4600")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("lưu phải 200, được %d — %s", w.Code, w.Body.String())
	}

	r2 := req("GET", "/api/flow/def?name=thu")
	r2.AddCookie(ck)
	w2 := httptest.NewRecorder()
	s.ServeHTTP(w2, r2)
	if w2.Code != 200 {
		t.Fatalf("đọc lại phải 200, được %d — %s", w2.Code, w2.Body.String())
	}
	var got struct {
		Steps []struct {
			ID           string `json:"id"`
			TuDuyetQuyen bool   `json:"tuDuyetQuyen"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	thay := map[string]bool{}
	for _, st := range got.Steps {
		thay[st.ID] = st.TuDuyetQuyen
	}
	if len(thay) != 2 {
		t.Fatalf("phải đọc lại 2 bước, được %d", len(thay))
	}
	if !thay["co"] {
		t.Fatal("BƯỚC KHAI tu_duyet_quyen BỊ MẤT CỜ sau vòng lưu — flow im lặng mất quyền")
	}
	if thay["khong"] {
		t.Fatal("bước KHÔNG khai gì lại được cấp toàn quyền sau vòng lưu")
	}
}
