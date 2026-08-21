package dash

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Mặt web phải đọc ĐÚNG TÊN TRƯỜNG mà sessionDTO thật sự phát ra.
//
// VÌ SAO CÓ (21/08): mặt 3D đọc `s.tokens`, `s.costUSD`, `s.cost` — KHÔNG cái
// nào tồn tại trong DTO (tên thật là `tokensIn`, `tokensOut`, `costUsd`). Hậu
// quả không phải một lỗi ồn ào: `|| 0` nuốt luôn, nên hai ô token/chi phí đứng ở
// dấu gạch VĨNH VIỄN, trông y hệt "chưa đo được". Mặt 2D đọc đúng từ đầu, nên
// hai mặt nói hai chuyện về cùng một dữ liệu mà không ai thấy.
//
// Mục C4 của docs/SO-NO-DO-LUONG.md đã nêu đúng cái bẫy này, kể cả đúng hai cái
// tên `s.cost` và `s.tokens`. Nó vẫn xảy ra — vì một dòng cảnh báo trong tài
// liệu không chặn được gì. Bài kiểm này mới chặn.
//
// Tên trường lấy từ CHÍNH struct qua json.Marshal, không chép tay: đổi tên trong
// Go mà quên đổi ở web thì đỏ ngay, thay vì trang lặng lẽ hiện dấu gạch.
func TestMatWebDocDungTenTruongCuaDTO(t *testing.T) {
	usd := 1.25
	b, err := json.Marshal(sessionDTO{
		ID: 1, TokenVao: 10, TokenRa: 20, ChiPhiUSD: &usd,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}

	// Ba trường số liệu PHẢI có mặt trong DTO. Thiếu một cái là bài kiểm này mất
	// nghĩa, nên bắt luôn ở đây thay vì để nó xanh vì rỗng.
	var canCo []string
	for _, k := range []string{"tokensIn", "tokensOut", "costUsd"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("sessionDTO không còn trường %q — đổi tên thì phải sửa cả mặt web", k)
		}
		canCo = append(canCo, k)
	}

	// Tên CŨ/SAI đã từng dùng thật. Ai gõ lại là đỏ.
	//
	// Dùng regexp có RANH GIỚI chứ không phải strings.Contains: `s.tokens` là
	// tiền tố của `s.tokensIn` hợp lệ, nên dò chuỗi trần sẽ bắt oan đúng cái tên
	// đúng. Bản đầu của bài kiểm này đã mắc lỗi đó.
	camDung := []*regexp.Regexp{
		regexp.MustCompile(`s\.tokens($|[^A-Za-z])`),
		regexp.MustCompile(`s\.costUSD($|[^A-Za-z])`),
		regexp.MustCompile(`s\.cost($|[^A-Za-z])`),
		regexp.MustCompile(`s\.tok($|[^A-Za-z])`),
		regexp.MustCompile(`s\.usd($|[^A-Za-z])`),
	}

	for _, ten := range []string{"index.html", "trung-tam.html"} {
		duong := filepath.Join("web", ten)
		raw, err := os.ReadFile(duong)
		if err != nil {
			t.Fatalf("không đọc được %s: %v", duong, err)
		}
		s := string(raw)

		// Bỏ dòng bình luận trước khi dò: bình luận GIẢI THÍCH cái bẫy thì được
		// phép nhắc tên sai — cấm cả bình luận là cấm luôn việc ghi lại bài học.
		var ma []string
		for _, d := range strings.Split(s, "\n") {
			t := strings.TrimSpace(d)
			if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "*") || strings.HasPrefix(t, "/*") {
				continue
			}
			ma = append(ma, d)
		}
		thanMa := strings.Join(ma, "\n")

		for _, re := range camDung {
			if m := re.FindString(thanMa); m != "" {
				t.Errorf("%s đọc %q — tên đó KHÔNG có trong DTO; dùng %v",
					ten, strings.TrimSpace(m), canCo)
			}
		}
	}

	// Và mặt 3D phải THẬT SỰ đọc ba trường đó, không phải chỉ "không đọc sai".
	raw, err := os.ReadFile(filepath.Join("web", "trung-tam.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range canCo {
		if !strings.Contains(string(raw), k) {
			t.Errorf("trung-tam.html không hề đọc %q — ô số liệu sẽ đứng ở dấu gạch mãi", k)
		}
	}
}
