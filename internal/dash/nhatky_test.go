package dash

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/nhatky"
)

// Mặt WEB của nhật ký phiên — action "session.nhat-ky".
//
// Không có endpoint này thì câu "phiên #167 báo xong mà 0 commit, vì sao" chỉ
// trả lời được ở terminal, đúng thứ luật ngang quyền (MASTER-PLAN mục 2c luật
// 2) cấm. TestMoiHanhDongDeuCoDuongVaoTuWeb canh chuyện CÓ đường vào; bài dưới
// canh chuyện đường đó NÓI ĐỦ.

func goiNhatKy(t *testing.T, s *Server, ck interface{ String() string }, target string) (int, map[string]any) {
	t.Helper()
	r := req("GET", target)
	r.Header.Set("Cookie", ck.String())
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

// Bảng nhật ký phải nói ra NGÂN SÁCH ĐĨA, không chỉ danh sách.
//
// Người vận hành đi tìm nhật ký của lượt tuần trước mà không thấy thì phải đọc
// được ở đâu đó rằng nó bị dọn theo trần, chứ không phải nghi công cụ làm mất.
func TestWebNhatKyNoiRaThuMucVaNganSach(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	code, body := goiNhatKy(t, s, ck, "/api/nhat-ky")
	if code != 200 {
		t.Fatalf("GET /api/nhat-ky = %d", code)
	}
	if got, _ := body["thu_muc"].(string); got != nhatky.Root() {
		t.Errorf("thu_muc = %q, chờ %q", got, nhatky.Root())
	}
	if got, _ := body["tran_so_file"].(float64); int(got) != nhatky.SoFileToiDa {
		t.Errorf("tran_so_file = %v, chờ %d", body["tran_so_file"], nhatky.SoFileToiDa)
	}
	if got, _ := body["tran_byte"].(float64); int64(got) != nhatky.TongByteToiDa {
		t.Errorf("tran_byte = %v, chờ %d", body["tran_byte"], nhatky.TongByteToiDa)
	}
	// `muc` phải là một mảng (rỗng cũng được) chứ không phải null — mặt web sẽ
	// lặp trên nó, và null làm trang trắng thay vì hiện "chưa có phiên nào".
	if _, ok := body["muc"].([]any); !ok {
		t.Errorf("muc = %#v, chờ một mảng", body["muc"])
	}
}

// Hỏi số phiên không có thật: trả lỗi ĐỌC ĐƯỢC, không phải 500 trần trụi.
func TestWebNhatKyBaoLoiRoKhiSoPhienKhongCo(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	code, body := goiNhatKy(t, s, ck, "/api/nhat-ky?id=99999")
	if code != 400 {
		t.Fatalf("hỏi phiên không có thật = %d, chờ 400", code)
	}
	if e, _ := body["error"].(string); e == "" {
		t.Errorf("không có câu giải thích trong %v", body)
	}

	code, body = goiNhatKy(t, s, ck, "/api/nhat-ky?id=abc")
	if code != 400 {
		t.Fatalf("id không phải số = %d, chờ 400", code)
	}
	if e, _ := body["error"].(string); e == "" {
		t.Errorf("không có câu giải thích trong %v", body)
	}
}
