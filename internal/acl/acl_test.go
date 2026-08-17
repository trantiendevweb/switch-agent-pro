package acl

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// noiLong mở rộng quyền của một thư mục để dựng đúng cái bẫy cần đo.
//
// Dùng `icacls` với SID chứ không với tên nhóm: tên nhóm đổi theo ngôn ngữ
// Windows ("Users" / "Utilisateurs" / "Benutzer"), SID thì không. Đây là code
// TEST nên gọi lệnh ngoài được; code sản phẩm thì không (xem acl_windows.go).
func noiLong(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		// S-1-5-32-545 = nhóm Users dựng sẵn.
		if err := exec.Command("icacls", dir, "/grant", "*S-1-5-32-545:(OI)(CI)F").Run(); err != nil {
			t.Skipf("không nới lỏng được ACL để dựng bẫy: %v", err)
		}
		return
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
}

// Phép đo gốc của package này: file bí mật ghi bằng 0o600 KHÔNG được bảo vệ trên
// Windows. Test khẳng định Restrict sửa được điều đó, và Check nhìn ra được
// trạng thái hỏng TRƯỚC khi vá — một hàm kiểm mà lúc nào cũng nói "ổn" thì vô dụng.
func TestSietQuyenThuMucBiMat(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kho")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	noiLong(t, dir)

	ok, detail, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("Check nói %q là kín trong khi vừa nới lỏng xong — hàm kiểm này không đo gì cả (%s)", dir, detail)
	}
	t.Logf("trước khi siết: %s", detail)

	if err := Restrict(dir); err != nil {
		t.Fatalf("Restrict = %v", err)
	}

	ok, detail, err = Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("sau Restrict vẫn hở: %s", detail)
	}
	t.Logf("sau khi siết:  %s", detail)
}

// File tạo SAU khi siết phải thừa hưởng quyền chặt, nếu không thì mỗi hồ sơ mới
// lại hở một lần.
func TestFileTaoSauKhiSietVanKin(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kho")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	noiLong(t, dir)
	if err := Restrict(dir); err != nil {
		t.Fatal(err)
	}

	f := filepath.Join(dir, ".credentials.json")
	if err := os.WriteFile(f, []byte(`{"token":"bi-mat"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ok, detail, err := Check(f)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("file tạo sau khi siết vẫn hở: %s", detail)
	}
}

// Siết hai lần không được hỏng: các lối gọi đều chạy mỗi lần mở hồ sơ.
func TestSietNhieuLanKhongHong(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kho")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := Restrict(dir); err != nil {
			t.Fatalf("lần siết thứ %d: %v", i+1, err)
		}
	}
	if ok, detail, _ := Check(dir); !ok {
		t.Fatalf("sau 3 lần siết vẫn hở: %s", detail)
	}
}
