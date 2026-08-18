package dash

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// Mặt web phải trả KẾT QUẢ của từng bước, không chỉ đề bài.
//
// Lần chạy #8 hiện "completed" đủ ba bước trong khi cả hai agent chỉ trả về câu
// từ chối quyền. Không mặt nào đọc được output nên không ai biết. `/api/run` trả
// `detail` = prompt (đề bài) và bỏ hẳn output (bài làm) — nhìn vào chỉ thấy
// mình ĐỊNH bảo agent làm gì, không thấy nó ĐÃ làm gì.
//
// Dùng bước `notify` để khỏi cần agent thật: nó cũng ghi output như mọi bước.
func TestWebTraKetQuaTungBuoc(t *testing.T) {
	s := newTestServer(t)
	dir := t.TempDir()
	t.Chdir(dir)

	const loiNhan = "ket qua that cua buoc nay"
	f := flow.Flow{Name: "thu", Steps: []flow.Step{
		{ID: "bao", Type: flow.TypeNotify, Message: loiNhan},
	}}
	if _, err := s.api.FlowSave(dir, f); err != nil {
		t.Fatal(err)
	}
	res, err := s.api.FlowRun(context.Background(), dir, "thu", nil, api.Addr{})
	if err != nil {
		t.Fatal(err)
	}

	r := req("GET", "/api/run?id="+strconv.FormatInt(res.RunID, 10))
	r.AddCookie(dangNhap(t, s, "127.0.0.1:4600"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("phải 200, được %d — %s", w.Code, w.Body.String())
	}

	var body struct {
		Steps []struct {
			ID     string `json:"id"`
			Output string `json:"output"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Steps) == 0 {
		t.Fatal("không có bước nào trong trả lời")
	}
	for _, st := range body.Steps {
		if st.ID == "bao" {
			if !strings.Contains(st.Output, loiNhan) {
				t.Fatalf("mặt web GIẤU kết quả bước: output = %q, muốn chứa %q", st.Output, loiNhan)
			}
			return
		}
	}
	t.Fatal("không thấy bước 'bao' trong trả lời")
}
