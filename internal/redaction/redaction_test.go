package redaction

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gia ghép một bí mật GIẢ lúc chạy.
//
// KHÔNG viết thẳng chuỗi `sk-ant-xxxx...` vào file test, dù nó là đồ giả. Bài
// kiểm quét repo (TestRepoKhongCoBiMat) đọc chính file này; viết thẳng thì nó
// đỏ, và cách sửa duy nhất khi đó là thêm danh sách miễn trừ — mà một bài kiểm
// bí mật có danh sách miễn trừ thì lần rò thật đầu tiên cũng sẽ được ai đó cho
// vào danh sách ấy. Ghép lúc chạy thì bài kiểm giữ nguyên độ chặt, 0 ngoại lệ.
func gia(phan ...string) string { return strings.Join(phan, "") }

// nguoc là ĐÚNG MỘT dấu gạch chéo ngược, dựng từ mã byte 92.
var nguoc = string([]byte{92})

// Mỗi lớp mẫu phải che được đúng thứ nó khai là che được.
func TestCheBiMat(t *testing.T) {
	ca := []struct {
		ten string
		vao string
	}{
		{"khoá Anthropic", gia("sk-", "ant-", "api03-", strings.Repeat("A", 40))},
		{"token OAuth Anthropic", gia("sk-", "ant-", "oat01-", strings.Repeat("B", 40))},
		{"khoá OpenAI", gia("sk-", strings.Repeat("C", 40))},
		{"token GitHub cổ điển", gia("ghp_", strings.Repeat("d", 36))},
		{"token GitHub dạng mới", gia("github", "_pat_", strings.Repeat("e", 40))},
		{"khoá Google", gia("AI", "za", strings.Repeat("f", 35))},
		{"khoá AWS", gia("AK", "IA", strings.Repeat("G", 16))},
		{"token Slack", gia("xox", "b-", strings.Repeat("1", 20))},
		{"JWT", gia("ey", "J", strings.Repeat("h", 20), ".", strings.Repeat("i", 20), ".", strings.Repeat("j", 20))},
		{"Bearer", gia("Authorization: Bea", "rer ", strings.Repeat("k", 40))},
		{"gán biến môi trường", gia("ANTHROPIC_API", "_KEY=", strings.Repeat("m", 40))},
		{"trường JSON accessToken", gia(`{"access`, `Token":"`, strings.Repeat("n", 40), `"}`)},
		{"khối khoá riêng", gia("-----BEGIN RSA PRI", "VATE KEY-----\n", strings.Repeat("o", 40), "\n-----END RSA PRI", "VATE KEY-----")},
	}
	for _, c := range ca {
		t.Run(c.ten, func(t *testing.T) {
			ra := Che(c.vao)
			if ra == c.vao {
				t.Fatalf("%s: không che gì cả\nvào: %s", c.ten, c.vao)
			}
			if !strings.Contains(ra, NhanKhoa) {
				t.Fatalf("%s: che rồi mà không có nhãn %q\nra: %s", c.ten, NhanKhoa, ra)
			}
			// Phần thân bí mật phải BIẾN MẤT, không phải chỉ bị cắt ngắn.
			if than := than(c.vao); than != "" && strings.Contains(ra, than) {
				t.Fatalf("%s: thân bí mật vẫn còn nguyên trong kết quả\nra: %s", c.ten, ra)
			}
		})
	}
}

// than lấy đoạn ký tự lặp dài nhất trong s — chính là phần "ruột" của bí mật
// giả ở trên. Nó còn sót lại nghĩa là mới cắt chứ chưa che.
func than(s string) string {
	dai, batDau := 0, 0
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && s[j] == s[i] {
			j++
		}
		if j-i > dai {
			dai, batDau = j-i, i
		}
		i = j
	}
	if dai < 16 {
		return ""
	}
	return s[batDau : batDau+dai]
}

// KHÔNG ĐƯỢC che nhầm. Đây là nửa quan trọng hơn: một tầng che băm nát nhật ký
// thì người ta tắt nó đi, và tắt rồi thì nó chống được đúng 0 vụ rò.
func TestCheKhongChamThuVoHai(t *testing.T) {
	ca := []struct {
		ten string
		vao string
	}{
		{
			// 471 khối như thế này trong 22 file nhật ký thật. Luật "base64 dài
			// = bí mật" sẽ băm hết chúng.
			"chữ ký khối thinking của Claude",
			`{"signature":"` + strings.Repeat("Ab9+/", 200) + `"}`,
		},
		{
			// Đo được 9 lần: nói TÊN biến, không kèm giá trị.
			"văn xuôi nhắc tên biến môi trường",
			"Error: API key required. Set GROK_API_KEY environment variable, or use --api-key",
		},
		{
			// Đo được 218 lần: chữ "refreshToken" trong mã nguồn đang được đọc.
			"tên trường trong mã nguồn",
			"refreshToken \"\" / expiresAt 0",
		},
		{"thẻ struct json", "OpenAIAPIKey string `json:\"OPENAI_API_KEY\"`"},
		{"tên cờ dòng lệnh", "dashboard — chạy: sagent dash --set-password"},
		{"tên hàm", "func pbkdf2(password, salt []byte, iter, keyLen int) []byte {"},
	}
	for _, c := range ca {
		t.Run(c.ten, func(t *testing.T) {
			if ra := Che(c.vao); ra != c.vao {
				t.Fatalf("che nhầm thứ vô hại\nvào: %s\nra:  %s", c.vao, ra)
			}
		})
	}
}

// Đường dẫn nhà: bỏ TÊN, giữ ĐUÔI. Mất đuôi thì nhật ký hết truy nguyên được —
// đuôi chính là chỗ agent đang làm việc.
func TestCheDuongDanNhaGiuLaiDuoi(t *testing.T) {
	ca := []string{
		`C:\Users\Administrator\.ai-accounts\.nhat-ky`,
		`C:/Users/Administrator/.ai-accounts/.nhat-ky`,
		`C:\\Users\\Administrator\\.ai-accounts\\.nhat-ky`, // dạng đã thoát trong NDJSON
	}
	for _, vao := range ca {
		ra := Che(vao)
		if strings.Contains(ra, "Administrator") {
			t.Errorf("tên người dùng còn nguyên: %s", ra)
		}
		if !strings.Contains(ra, NhanNguoiDung) {
			t.Errorf("không thấy nhãn: %s", ra)
		}
		if !strings.Contains(ra, ".nhat-ky") {
			t.Errorf("che mất cả phần đuôi, hết truy nguyên được: %s", ra)
		}
	}
}

// BA DẠNG ĐƯỜNG DẪN DƯỚI ĐÂY ĐỀU LÀ CHỖ SÓT THẬT, tìm ra bằng cách chạy tầng
// che trên 22 file nhật ký thật rồi ĐẾM XEM CÒN LẠI BAO NHIÊU — chứ không phải
// bằng cách ngồi nghĩ xem đường dẫn còn dạng nào. Bản đầu che xong vẫn còn 581
// lần tên tài khoản nằm trơ trong nhật ký.
func TestCheCacDangDuongDanBiSotLucDau(t *testing.T) {
	ca := []struct {
		ten string
		vao string
	}{
		{
			// 916 chỗ. Dòng nhật ký là JSON, bên trong có chuỗi ghi lại lời gọi
			// công cụ vốn cũng là JSON → gạch ngược bị thoát HAI lần.
			"JSON lồng JSON, bốn gạch ngược",
			`Read {\"file_path\":\"C:` + strings.Repeat(nguoc, 4) + `Users` +
				strings.Repeat(nguoc, 4) + `Administrator` + strings.Repeat(nguoc, 4) + `du-an\"}`,
		},
		{
			// 46 chỗ. Claude Code bẹp cả đường dẫn thành tên thư mục dự án.
			"đường dẫn bẹp thành gạch nối",
			".clones/claude/tns/1/projects/C--Users-Administrator-Projects-switch-agent-pro",
		},
		{
			// 18 chỗ. Tên 8.3 của Windows.
			"tên ngắn 8.3",
			`C:\Users\ADMINI~1\AppData\Local\Temp`,
		},
	}
	for _, c := range ca {
		t.Run(c.ten, func(t *testing.T) {
			ra := Che(c.vao)
			if !strings.Contains(ra, NhanNguoiDung) {
				t.Fatalf("không che gì:\nvào: %s\nra:  %s", c.vao, ra)
			}
		})
	}
}

// TÊN TÀI KHOẢN ĐỨNG MỘT MÌNH, ngoài mọi đường dẫn.
//
// 525 chỗ sót sau khi đã có cả hai luật đường dẫn, và chúng là cột chủ sở hữu
// của `ls -l`. Không mẫu hình dạng nào bắt được một cái tên đứng trơ — nhưng ta
// biết cái tên đó, vì nó là tên thư mục nhà.
func TestCheTenTaiKhoanDungMotMinh(t *testing.T) {
	nha := filepath.Join(t.TempDir(), "NguoiDungThu")
	if err := os.MkdirAll(nha, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", nha)
	t.Setenv("USERPROFILE", nha)

	ra := Che("drwxr-xr-x 1 NguoiDungThu 197121 0 Aug 21 14:47 .")
	if strings.Contains(ra, "NguoiDungThu") {
		t.Errorf("tên tài khoản ở cột chủ sở hữu vẫn còn: %s", ra)
	}
	if !strings.Contains(ra, NhanNguoiDung) {
		t.Errorf("không thấy nhãn: %s", ra)
	}
	// RANH GIỚI TỪ: tên nhóm `Administrators` KHÁC tên người dùng
	// `Administrator`. Thiếu `\b` thì mọi tài liệu nói về nhóm Windows đều bị
	// băm — đo được 23 chỗ như thế trong nhật ký thật.
	if ra := Che("nhóm NguoiDungThuVien giữ nguyên"); !strings.Contains(ra, "NguoiDungThuVien") {
		t.Errorf("che lẹm sang từ khác: %s", ra)
	}
}

// TÊN QUÁ NGẮN THÌ KHÔNG CHE. `dev`, `ci`, `adm` là từ thường gặp trong văn
// xuôi lẫn mã nguồn: thay bừa thì băm nát nhật ký để đổi lấy một chút riêng tư
// mà kẻ đọc đoán ra ngay. Thà không che còn hơn che hỏng.
func TestCheBoQuaTenTaiKhoanQuaNgan(t *testing.T) {
	nha := filepath.Join(t.TempDir(), "dev")
	if err := os.MkdirAll(nha, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", nha)
	t.Setenv("USERPROFILE", nha)

	vao := "chạy dev server, dev mode, thư mục dev"
	if ra := Che(vao); ra != vao {
		t.Fatalf("tên 3 ký tự mà vẫn thay, băm nát văn xuôi:\nvào: %s\nra:  %s", vao, ra)
	}
}

// Email: còn phân biệt được người, nhưng không còn gửi tới được.
func TestCheEmail(t *testing.T) {
	ra := Che("tác giả: nguoi-that@vidu.com và bot@ci.example.org")
	for _, khong := range []string{"nguoi-that@", "bot@"} {
		if strings.Contains(ra, khong) {
			t.Errorf("email chưa che: %s", ra)
		}
	}
	// Giữ tên miền: đủ để nói "của người dùng" hay "của bot CI".
	for _, phai := range []string{"n***@vidu.com", "b***@ci.example.org"} {
		if !strings.Contains(ra, phai) {
			t.Errorf("mất tên miền, hết phân biệt được ai với ai: %s", ra)
		}
	}
}

// NHẬT KÝ VẪN PHẢI LÀ NDJSON HỢP LỆ SAU KHI CHE.
//
// Ràng buộc dễ vỡ nhất của cả tầng này. Nhật ký không chỉ để người đọc: nó là
// output của bước agent, nạp thẳng vào prompt bước sau, và `provider.DocKetQua`
// cũng phân tích từng dòng. Nhãn thay thế mà lọt một dấu nháy kép hay một gạch
// chéo ngược thì dòng JSON hỏng — im lặng, và hỏng ở phía bên kia.
func TestCheGiuNguyenNDJSONHopLe(t *testing.T) {
	dong := []string{
		`{"type":"assistant","cwd":"C:\\Users\\Administrator\\du-an","text":"xong"}`,
		`{"type":"user","email":"ai-do@vidu.com","token":"` + gia("sk-", "ant-", strings.Repeat("z", 40)) + `"}`,
		`{"type":"result","signature":"` + strings.Repeat("Qq7+/", 100) + `","is_error":false}`,
		// HỒI QUY, bắt được bằng cách chạy tầng che trên 22 file nhật ký THẬT
		// chứ không phải bằng ví dụ tự nghĩ ra. Antigravity ghi dấu ngoặc nhọn
		// thành chuỗi thoát `<`/`>`, và mẫu email bản đầu nuốt luôn
		// `u003c` — thay xong còn `\` đứng trước nhãn, tức `\[`, không phải
		// chuỗi thoát JSON hợp lệ. Hỏng thật 2 dòng, và hỏng ở phía bên kia.
		// `nguoc` là ĐÚNG MỘT dấu gạch chéo ngược, dựng từ mã byte. Viết thẳng
		// chuỗi thoát vào đây thì mỗi tầng công cụ (trình soạn, bản vá, bảng
		// tạm) lại có cơ hội "giúp" ta bằng cách diễn giải nó — và ca hỏng này
		// mất đúng cái đó thì test vẫn xanh trong khi lỗi còn nguyên.
		`{"text":"Co-Authored-By: Claude ` + nguoc + `u003c` +
			`noreply@anthropic.com` + nguoc + `u003e"}`,
	}
	for _, d := range dong {
		ra := Che(d)
		var v map[string]any
		if err := json.Unmarshal([]byte(ra), &v); err != nil {
			t.Fatalf("che xong dòng JSON hỏng: %v\nra: %s", err, ra)
		}
	}
}

// TimBiMat chỉ nhìn nhóm BÍ MẬT. Email và đường dẫn nhà bị che ở nhật ký nhưng
// KHÔNG được chặn commit — repo này có email trong trailer git và có đường dẫn
// `C:\Users\...` nằm trong tài liệu.
func TestTimBiMatBoQuaNhomDanhTinh(t *testing.T) {
	if p := TimBiMat(`C:\Users\Administrator\du-an, liên hệ ai-do@vidu.com`); len(p) != 0 {
		t.Fatalf("chặn nhầm nhóm danh tính: %+v", p)
	}
	vao := "dòng 1\ndòng 2\nkhoá: " + gia("sk-", "ant-", strings.Repeat("y", 40))
	p := TimBiMat(vao)
	if len(p) != 1 {
		t.Fatalf("mong 1 phát hiện, được %d: %+v", len(p), p)
	}
	if p[0].Dong != 3 {
		t.Errorf("chỉ sai dòng: %d (mong 3)", p[0].Dong)
	}
	// Bản báo lỗi KHÔNG được là chỗ rò tiếp theo.
	if !strings.Contains(p[0].Trich, NhanKhoa) || strings.Contains(p[0].Trich, "yyyy") {
		t.Errorf("trích dẫn in nguyên khoá ra: %q", p[0].Trich)
	}
}
