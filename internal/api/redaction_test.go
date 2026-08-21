package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/nhatky"
	"github.com/trantiendevweb/switch-agent-pro/internal/redaction"
)

// TẦNG CHE PHẢI ĐƯỢC CẮM VÀO HỢP ĐỒNG, KHÔNG CHỈ TỒN TẠI.
//
// Test ở internal/redaction chứng minh các mẫu che đúng thứ chúng khai. Test ở
// đây chứng minh chuyện khác hẳn: hai CỬA RA thật của nhật ký có gọi nó không.
// Gói che có đủ 13 lớp mẫu mà `SessionNhatKyDoc` quên gọi thì bí mật vẫn ra
// thẳng cổng HTTP của dash — và bộ test kia vẫn xanh từ đầu tới cuối.
//
// Hai cửa, hai test, không gộp: chúng đi qua hai hàm khác nhau (Duoi và BoDau)
// nên gỡ dây ở một cửa mà test kia vẫn xanh thì test kia vô dụng.

// biMatGia ghép bí mật giả lúc chạy — xem ghi chú ở redaction.gia về việc vì
// sao không viết thẳng chuỗi vào file.
func biMatGia() string {
	return strings.Join([]string{"sk-", "ant-", "api03-", strings.Repeat("Z", 40)}, "")
}

// CỬA 1 — đường của NGƯỜI và của DASH: sagent nhat-ky, và
// dash/server.go:1172 phục vụ nhật ký qua HTTP. Dash tự biết mình có thể không
// nằm trên loopback (`s.exposed = !isLoopbackAddr(host)`), nên đây là cửa mà
// nhật ký có thể rời khỏi hẳn cái máy này.
func TestSessionNhatKyDocCheBiMat(t *testing.T) {
	a := moAPI(t)
	khoa := biMatGia()
	than := `{"type":"user","cwd":"C:\\Users\\Administrator\\du-an"}` + "\n" +
		`{"type":"assistant","text":"khoá là ` + khoa + `, thư gửi nguoi-that@vidu.com"}` + "\n"
	id, _ := themPhienCoNhatKy(t, a, than)

	// CẢ HAI nhánh của Duoi: xin nguyên file (dong = 0, nhánh trả về sớm) và
	// xin theo đuôi (dong > 0). Chỉ thử một nhánh thì nhánh kia rò trong im
	// lặng — và nhánh dong = 0 chính là nhánh dash dùng khi người ta bấm "xem
	// tất cả".
	for _, dong := range []int{0, 50} {
		_, noiDung, err := a.SessionNhatKyDoc(id, dong)
		if err != nil {
			t.Fatal(err)
		}
		for _, khong := range []string{khoa, "Administrator", "nguoi-that@vidu.com"} {
			if strings.Contains(noiDung, khong) {
				t.Errorf("dong=%d: %q lọt ra cửa đọc nhật ký:\n%s", dong, khong, noiDung)
			}
		}
		if !strings.Contains(noiDung, redaction.NhanKhoa) {
			t.Errorf("dong=%d: không thấy nhãn che — tầng che chưa được cắm vào", dong)
		}
		// Che chứ KHÔNG phải bịt mắt: phần đọc được vẫn phải đọc được.
		if !strings.Contains(noiDung, "du-an") || !strings.Contains(noiDung, "claude:tns#1") {
			t.Errorf("dong=%d: che luôn cả phần cần cho truy nguyên:\n%s", dong, noiDung)
		}
	}
}

// CỬA 2 — đường của MÁY: api.readLogs lấy nguyên văn nhật ký làm output của
// bước agent rồi nạp thẳng vào PROMPT của bước sau. Bước sau có thể chạy bằng
// một nhà cung cấp KHÁC, nên rò ở đây là gửi bí mật sang một bên thứ ba.
func TestReadLogsCheBiMat(t *testing.T) {
	dir := t.TempDir()
	khoa := biMatGia()
	dau := nhatky.Dau{
		ThoiDiem: time.Now(), Addr: "claude:tns#1",
		HoSo: `C:\clones\claude\tns\1`, Lenh: []string{"-p", "việc"},
	}
	than := "agent nói: khoá " + khoa + ", nhà " + `C:\Users\Administrator\du-an` +
		", thư nguoi-that@vidu.com\n"
	p := filepath.Join(dir, "a.log")
	if err := os.WriteFile(p, []byte(dau.String()+than), 0o600); err != nil {
		t.Fatal(err)
	}

	got := readLogs([]string{p})
	for _, khong := range []string{khoa, "Administrator", "nguoi-that@vidu.com"} {
		if strings.Contains(got, khong) {
			t.Errorf("%q được nạp vào prompt của bước agent sau:\n%s", khong, got)
		}
	}
	if !strings.Contains(got, redaction.NhanKhoa) {
		t.Error("không thấy nhãn che — tầng che chưa được cắm vào readLogs")
	}
	if !strings.Contains(got, "agent nói:") || !strings.Contains(got, "du-an") {
		t.Errorf("che mất cả câu trả lời của agent:\n%s", got)
	}
}

// ĐƯỜNG CỦA MÁY PHÂN LOẠI KHÔNG ĐƯỢC BỊ ĐỤNG.
//
// `phanLoaiPhienChet` đọc thẳng file rồi đưa cho `adapter.DocKetQua` để quyết
// trạng thái phiên. Nếu ai đó "tiện tay" cắm tầng che vào đường đó nữa thì mọi
// phiên rơi về `lost` — tức trả lại đúng cái mù loà mà nhật ký sinh ra để chữa.
func TestPhanLoaiPhienChetVanNhinThayByteGoc(t *testing.T) {
	a := moAPI(t)
	// Bản ghi kết có `cwd` chứa đường dẫn nhà: nếu đường phân loại bị che thì
	// dòng vẫn parse được, nên phải kiểm bằng thứ khác — ở đây là trạng thái
	// đọc ra phải KHÁC rỗng.
	than := `{"type":"result","subtype":"error_during_execution","is_error":true,` +
		`"api_error_status":"429","result":"","num_turns":3,` +
		`"cwd":"C:\\Users\\Administrator\\du-an"}` + "\n"
	id, _ := themPhienCoNhatKy(t, a, than)

	s, err := a.db.Phien(id)
	if err != nil {
		t.Fatal(err)
	}
	ly, chi, _ := phanLoaiPhienChet(s)
	if ly == "" && chi == "" {
		t.Fatal("đường phân loại không đọc ra gì — dấu hiệu byte gốc đã bị can thiệp")
	}
}
