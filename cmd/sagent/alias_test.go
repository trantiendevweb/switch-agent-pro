package main

// ALIAS `tk` / `ccswitch` — bài kiểm CHẠY THẬT, không phải bài kiểm đọc chữ.
//
// Vì sao cần một bài kiểm riêng cho hai file .cmd bốn chục byte: cả tính năng
// này nằm NGOÀI mã Go. `go build` không thấy nó, `go vet` không thấy nó, và
// người sửa cai-dat.ps1 lần sau cũng sẽ không thấy nó. Kế hoạch đã ghi mục này
// là "làm khi viết installer phát hành"; installer viết xong từ Pha 7 mà mục
// vẫn treo, đúng vì không có gì đỏ lên khi nó thiếu.
//
// Hai bài dưới đây bịt hai lỗ khác nhau:
//
//	TestInstallerKhaiDuAlias   — installer còn khai cả hai tên, và đường GỠ còn
//	                             dọn chúng. Đỏ khi ai đó xoá tính năng đi.
//	TestShimAliasChayThat      — nội dung shim ĐANG SHIP có thật sự chuyển được
//	                             tham số và mã thoát không. Đỏ khi shim viết sai.
//
// Bài thứ hai đọc thẳng $ShimNoiDung ra khỏi cai-dat.ps1 rồi chạy nó, chứ không
// chép lại một bản trong Go. Chép ra là mở đường cho hai bản lệch nhau, mà bản
// lệch ấy lại đúng là bản người dùng nhận.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

const bienTroGiupAlias = "SAGENT_TRO_GIUP_ALIAS"

// duongCaiDat là đường tới cai-dat.ps1 tính từ thư mục gói này (cmd/sagent).
const duongCaiDat = "../../install/cai-dat.ps1"

var (
	reTenAlias = regexp.MustCompile(`\$TenAlias\s*=\s*@\(([^)]*)\)`)
	reShim     = regexp.MustCompile(`\$ShimNoiDung\s*=\s*'([^']*)'`)
)

// docCauHinhAlias moi hai dòng NGUỒN SỰ THẬT ra khỏi installer.
func docCauHinhAlias(t *testing.T) (ten []string, shim string) {
	t.Helper()
	b, err := os.ReadFile(duongCaiDat)
	if err != nil {
		t.Fatalf("không đọc được %s: %v", duongCaiDat, err)
	}
	// File có BOM (bắt buộc — xem install/get.ps1). Cắt đi trước khi khớp.
	s := strings.TrimPrefix(string(b), "\ufeff")

	m := reTenAlias.FindStringSubmatch(s)
	if m == nil {
		t.Fatalf("%s không còn dòng `$TenAlias = @(...)` — phần alias đã bị gỡ hoặc đổi dạng", duongCaiDat)
	}
	for _, p := range strings.Split(m[1], ",") {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "'\"")
		if p != "" {
			ten = append(ten, p)
		}
	}

	m = reShim.FindStringSubmatch(s)
	if m == nil {
		t.Fatalf("%s không còn dòng `$ShimNoiDung = '...'`", duongCaiDat)
	}
	return ten, m[1]
}

// TestInstallerKhaiDuAlias giữ cho hai cái tên khỏi lặng lẽ biến mất.
func TestInstallerKhaiDuAlias(t *testing.T) {
	ten, shim := docCauHinhAlias(t)

	for _, muon := range []string{"tk", "ccswitch"} {
		co := false
		for _, x := range ten {
			if x == muon {
				co = true
			}
		}
		if !co {
			t.Errorf("installer không còn tạo alias %q (đang có: %v)", muon, ten)
		}
	}

	// Shim phải trỏ tới binary CẠNH NÓ (%~dp0), không phải một đường tuyệt đối
	// đóng cứng lúc cài: người dùng chuyển thư mục bin đi là hỏng hết.
	if !strings.Contains(shim, "%~dp0sagent.exe") {
		t.Errorf("shim không trỏ tới %%~dp0sagent.exe: %q", shim)
	}
	// `%*` chứ không phải %%1 %%2 %%3: chỉ %%* mới giữ nguyên phần đuôi dòng
	// lệnh, tức là giữ được cả tham số có dấu cách lẫn tham số thứ mười trở đi.
	if !strings.Contains(shim, "%*") {
		t.Errorf("shim không chuyển tham số bằng %%*: %q", shim)
	}

	// ĐƯỜNG GỠ. Cài mà không gỡ được thì hai file .cmd nằm lại trong PATH mãi
	// mãi, và lần sau người dùng gõ `tk` sẽ nhận "sagent.exe không tồn tại" —
	// một thông báo không nói được nó đến từ đâu.
	b, err := os.ReadFile(duongCaiDat)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "function Go-CaiDat") {
		t.Fatal("installer không có đường GỠ (function Go-CaiDat) — alias cài vào rồi không gỡ ra được")
	}
	if !strings.Contains(s, "if ($Go) { Go-CaiDat }") {
		t.Error("có hàm Go-CaiDat nhưng không chỗ nào gọi nó — cờ -Go là một cái nút chết")
	}
	// Vòng gỡ phải chạy trên CHÍNH $TenAlias, không phải một danh sách chép tay:
	// thêm alias thứ ba mà quên sửa vòng gỡ là để lại rác.
	gan := s[strings.Index(s, "function Go-CaiDat"):]
	if !strings.Contains(gan, "foreach ($ten in $TenAlias)") {
		t.Error("Go-CaiDat không lặp trên $TenAlias — thêm alias mới sẽ không được dọn")
	}
}

// TestTroGiupAlias KHÔNG phải một bài test: nó là thân của tiến trình con mà
// shim sẽ gọi. Chạy bình thường thì nó thoát ngay vì biến môi trường chưa đặt.
func TestTroGiupAlias(t *testing.T) {
	if os.Getenv(bienTroGiupAlias) != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	// In MỖI THAM SỐ MỘT DÒNG có rào hai đầu: đó là cách duy nhất phân biệt
	// một tham số "xin chao" với hai tham số "xin" và "chao".
	var b strings.Builder
	for _, a := range args {
		b.WriteString("[" + a + "]\n")
	}
	os.Stdout.WriteString(b.String())
	os.Exit(7) // mã lạ, khác 0 và khác 1: để kiểm chuyện chuyển mã thoát
}

// TestShimAliasChayThat chạy THẬT nội dung shim đang ship.
//
// Cách dựng: chép chính file test này thành `sagent.exe` trong một thư mục tạm,
// ghi `tk.cmd` cạnh nó bằng đúng chuỗi lấy từ installer, rồi gọi `tk.cmd`. Nếu
// shim đúng thì tiến trình con nhận nguyên tham số và mã thoát đi ngược lên.
//
// Không build cmd/sagent thật vì không cần: thứ đang được kiểm là CÁI SHIM,
// không phải cái binary. Dùng lại file test làm binary giả là mẹo đã dùng ở
// internal/flow/artifact_test.go.
func TestShimAliasChayThat(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("shim .cmd chỉ có nghĩa trên Windows")
	}
	tenAlias, shim := docCauHinhAlias(t)

	dir := t.TempDir()
	tuBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(tuBinary)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sagent.exe"), raw, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, ten := range tenAlias {
		ten := ten
		t.Run(ten, func(t *testing.T) {
			p := filepath.Join(dir, ten+".cmd")
			// \r\n: cmd.exe đọc file .cmd theo dòng, và một file chỉ có \n vẫn
			// chạy nhưng đây là thứ installer ghi ra (Set-Content trên Windows).
			if err := os.WriteFile(p, []byte(shim+"\r\n"), 0o755); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command("cmd", "/c", p,
				"-test.run=TestTroGiupAlias", "--", "in", "xin chao", "ba")
			cmd.Env = append(os.Environ(), bienTroGiupAlias+"=1")
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			err := cmd.Run()

			ee, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("gọi qua %s.cmd không trả về mã thoát của chương trình con (err=%v)\n%s",
					ten, err, out.String())
			}
			if ee.ExitCode() != 7 {
				t.Errorf("mã thoát = %d, muốn 7 — shim NUỐT mã thoát, và CI sẽ đọc mọi lượt hỏng là xanh\n%s",
					ee.ExitCode(), out.String())
			}

			got := out.String()
			for _, muon := range []string{"[in]", "[xin chao]", "[ba]"} {
				if !strings.Contains(got, muon) {
					t.Errorf("qua %s.cmd không thấy tham số %s trong đầu ra:\n%s", ten, muon, got)
				}
			}
			// Bẫy kinh điển: `%1 %2` tách "xin chao" thành hai tham số. Nếu
			// chuyện đó xảy ra thì có dòng [xin] riêng.
			if strings.Contains(got, "[xin]\n") {
				t.Errorf("tham số có dấu cách bị tách đôi khi đi qua %s.cmd:\n%s", ten, got)
			}
		})
	}
}
