// Bài kiểm cho hai endpoint artifact — và cho MẶT THỨ TƯ của chúng.
//
// Dự án này vấp BỐN lần trong ngày 22/08 vì cùng một hình dạng lỗi: thứ gì đó
// "có" ở mọi tầng TRỪ tầng người dùng thật sự chạm vào (route.kiem, nút
// Duyệt/Từ chối, plugin.list, và `sagent help` bỏ sót 5 lệnh). Bài
// TestTrangGoiCaHaiDuongArtifact dưới đây là chốt cho lần thứ năm.
package dash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// getJSON gọi một endpoint đã đăng nhập và trả về (mã HTTP, thân).
func getJSON(t *testing.T, s *Server, ck *http.Cookie, duong string) (int, string) {
	t.Helper()
	r := httptest.NewRequest("GET", duong, nil)
	r.Host = "127.0.0.1:4600"
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

// ---------------------------------------------------------------------------
// MẶT THỨ TƯ
// ---------------------------------------------------------------------------

// Có endpoint mà không trang nào gọi thì người dùng KHÔNG LÀM ĐƯỢC VIỆC.
//
// Cắt bình luận trước khi dò — bản đầu của bài kiểm anh em (ngangquyen_ui_test)
// đã xanh oan vì tên endpoint còn nằm trong một dòng bình luận giải thích chính
// lỗi đó. Dùng lại đúng boComment.
func TestTrangGoiCaHaiDuongArtifact(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	than := boComment(string(raw))

	for _, duong := range []string{"/api/flow/artifacts", "/api/flow/artifact?"} {
		if !strings.Contains(than, duong) {
			t.Errorf("KHÔNG trang nào gọi %q — muốn đọc file một lượt chạy để lại thì vẫn "+
				"phải mở thư mục trên máy chủ bằng tay, đúng thứ tính năng này sinh ra để bỏ",
				duong)
		}
	}

	// Và endpoint phải CÓ THẬT bên server, không thì bài kiểm này canh một cái
	// tên chết.
	srv, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, duong := range []string{"/api/flow/artifacts", "/api/flow/artifact"} {
		if !strings.Contains(string(srv), duong) {
			t.Errorf("server.go không còn %q", duong)
		}
	}
}

// Mặt web phải đọc ĐÚNG TÊN TRƯỜNG mà DTO thật sự phát ra — cùng cái bẫy mà
// dto_ten_truong_test.go bắt được ngày 21/08 (mặt 3D đọc `s.cost`, `s.tokens`,
// không cái nào tồn tại; `|| 0` nuốt hết và hai ô đứng ở dấu gạch vĩnh viễn).
//
// Tên lấy từ CHÍNH struct qua json.Marshal, không chép tay.
func TestMatWebDocDungTenTruongArtifact(t *testing.T) {
	than, err := os.ReadFile(filepath.Join("web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	ma := boComment(string(than))

	kiem := func(v any, canCo []string) {
		t.Helper()
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		for _, k := range canCo {
			if _, co := m[k]; !co {
				t.Fatalf("DTO không còn trường %q — đổi tên trong Go thì phải sửa cả mặt web", k)
			}
			if !strings.Contains(ma, k) {
				t.Errorf("mặt web KHÔNG đọc trường %q — ô đó sẽ đứng ở dấu gạch vĩnh viễn", k)
			}
		}
	}

	// Dùng giá trị KHÁC ZERO cho mọi trường: `omitempty` sẽ nuốt trường zero và
	// bài kiểm xanh oan vì nó không thấy trường nào để đòi.
	kiem(api.KhoArtifact{RunID: 1, Flow: "f", Dir: "d", TongByte: 9, ThieuDinhNghia: true,
		File: []api.FileArtifact{{Duong: "a/b", Byte: 1}}},
		[]string{"file", "tongByte", "thieuDinhNghia"})
	kiem(api.FileArtifact{Duong: "a/b", Buoc: "a", Ten: "t", Byte: 1, Sua: "s", NhiPhan: true},
		[]string{"duong", "buoc", "ten", "byte", "nhiPhan"})
	kiem(api.NoiDungArtifact{Duong: "a/b", Byte: 9, Tu: 1, DocByte: 2, BiCat: true,
		ConLai: 6, NhiPhan: true, Chu: "x"},
		[]string{"byte", "tu", "docByte", "biCat", "conLai", "nhiPhan", "chu"})
}

// ---------------------------------------------------------------------------
// TẦNG HTTP: THOÁT THƯ MỤC
// ---------------------------------------------------------------------------

// internal/api/artifact_test.go chứng minh HÀM chặn đúng. Bài này chứng minh
// chuyện khác hẳn: nó ĐƯỢC CẮM VÀO endpoint, và endpoint không tự "sửa nhẹ" cho
// dễ dùng trên đường đi.
//
// Đây đúng chỗ dự án đã vấp: `plugin.list` có đủ hợp đồng + CLI + endpoint mà
// không mặt nào gọi tới, và mọi bài kiểm vẫn xanh vì chúng chỉ hỏi từng mảnh.
func TestEndpointArtifactChanThoatThuMuc(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	// Thư mục artifact của lượt #1 phải CÓ THẬT, và trong đó phải có một file
	// đọc được. Không dựng thì hàm dừng ngay ở "lượt chạy này không để lại
	// artifact nào" và bài kiểm xanh TRƯỚC KHI chạm tới lớp chặn — xanh vì lý
	// do sai là dạng bài kiểm tệ nhất.
	if err := os.MkdirAll(filepath.Join(flow.ArtifactRunDir(1), "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(flow.ArtifactRunDir(1), "x", "ok.txt"),
		[]byte("vô hại"), 0o644); err != nil {
		t.Fatal(err)
	}
	biMat := filepath.Join(filepath.Dir(flow.ArtifactRoot()), "khoa.txt")
	if err := os.MkdirAll(filepath.Dir(biMat), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(biMat, []byte("KHOA-API-THAT"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Chốt: đường dẫn HỢP LỆ phải ra 200. Không có dòng này thì mọi khẳng định
	// bên dưới cũng đúng với một endpoint từ chối tất cả.
	if ma, than := getJSON(t, s, ck, "/api/flow/artifact?id=1&duong=x%2Fok.txt"); ma != http.StatusOK {
		t.Fatalf("đường dẫn hợp lệ trả về %d: %s", ma, than)
	}

	for _, xau := range []string{
		"../khoa.txt", "../../khoa.txt", "x/../../khoa.txt",
		`..\khoa.txt`, biMat, `C:\Windows\win.ini`, "/etc/passwd",
	} {
		ma, than := getJSON(t, s, ck,
			"/api/flow/artifact?id=1&duong="+url.QueryEscape(xau))
		if strings.Contains(than, "KHOA-API-THAT") {
			t.Errorf("duong=%q (HTTP %d) trả về nội dung file NGOÀI thư mục artifact — "+
				"endpoint này đang là một lỗ đọc file tuỳ ý trên cổng có thể phơi ra internet",
				xau, ma)
		}
		if ma == http.StatusOK {
			t.Errorf("duong=%q được trả về 200", xau)
		}
	}
}

// Thiếu tham số thì báo rõ, không im lặng trả về rỗng: một khối rỗng đọc là
// "không có gì", một câu trả lời khác hẳn "bạn gọi thiếu".
func TestEndpointArtifactThieuThamSoThiNoiRo(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	if ma, _ := getJSON(t, s, ck, "/api/flow/artifacts"); ma == http.StatusOK {
		t.Error("thiếu id mà vẫn trả về 200")
	}
	if ma, than := getJSON(t, s, ck, "/api/flow/artifact?id=1"); ma == http.StatusOK {
		t.Errorf("thiếu duong mà vẫn trả về 200: %s", than)
	}
	if _, than := getJSON(t, s, ck, "/api/flow/artifact?id=1&duong=x&tu=abc"); !strings.Contains(than, "tu") {
		t.Errorf("tu sai kiểu mà không nói ra: %s", than)
	}
}

// Chưa đăng nhập thì KHÔNG đọc được gì. Đây là endpoint đọc file, nên nó phải
// nằm sau đúng cái guard mà mọi endpoint khác nằm sau.
func TestEndpointArtifactPhaiDangNhapMoiGoiDuoc(t *testing.T) {
	s := newTestServer(t)
	for _, duong := range []string{"/api/flow/artifacts?id=1", "/api/flow/artifact?id=1&duong=x"} {
		r := httptest.NewRequest("GET", duong, nil)
		r.Host = "127.0.0.1:4600"
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			t.Errorf("%s trả về 200 khi CHƯA đăng nhập", duong)
		}
	}
}
