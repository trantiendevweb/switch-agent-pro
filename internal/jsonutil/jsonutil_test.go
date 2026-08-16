package jsonutil

import (
	"os"
	"path/filepath"
	"testing"
)

// Khoá JSON trùng y hệt: Go lấy cái CUỐI, không lỗi. Đây là bài làm PowerShell
// 5.1 chết ở v1 và là lý do bỏ được Python.
func TestReadObjectDuplicateKeys(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.json")
	if err := os.WriteFile(p, []byte(`{"a":1,"a":2,"b":3}`), 0o644); err != nil {
		t.Fatal(err)
	}
	obj, err := ReadObject(p)
	if err != nil {
		t.Fatalf("không nên lỗi khi khoá trùng: %v", err)
	}
	if string(obj["a"]) != "2" {
		t.Fatalf("khoá trùng phải lấy cái cuối, được %s", obj["a"])
	}
}

// Seed CHỈ mang whitelist — không kéo theo danh tính hay cache gói cước.
func TestSeedOnlyWhitelist(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.json")
	dst := filepath.Join(tmp, "dst.json")
	if err := os.WriteFile(src, []byte(`{"projects":{"x":1},"oauthAccount":{"emailAddress":"a@b.c"},"userID":"secret"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := Seed(src, dst, []string{"projects"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("gieo 1 khoá, được %d", n)
	}
	obj, _ := ReadObject(dst)
	if _, ok := obj["oauthAccount"]; ok {
		t.Fatal("KHÔNG được mang oauthAccount sang tài khoản mới")
	}
	if _, ok := obj["userID"]; ok {
		t.Fatal("KHÔNG được mang userID sang tài khoản mới")
	}
	if _, ok := obj["projects"]; !ok {
		t.Fatal("thiếu projects")
	}
}

// SyncKeys cập nhật whitelist nhưng giữ nguyên danh tính của đích và không kéo
// khoá ngoài whitelist.
func TestSyncKeys(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.json")
	dst := filepath.Join(tmp, "dst.json")
	os.WriteFile(src, []byte(`{"projects":{"x":2},"userID":"secret"}`), 0o644)
	os.WriteFile(dst, []byte(`{"projects":{"x":1},"oauthAccount":{"emailAddress":"keep@me"}}`), 0o644)
	n, err := SyncKeys(src, dst, []string{"projects"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("đổi 1 khoá, được %d", n)
	}
	obj, _ := ReadObject(dst)
	if string(obj["projects"]) != `{"x":2}` {
		t.Fatalf("projects chưa cập nhật: %s", obj["projects"])
	}
	if _, ok := obj["oauthAccount"]; !ok {
		t.Fatal("KHÔNG được xoá danh tính của đích")
	}
	if _, ok := obj["userID"]; ok {
		t.Fatal("KHÔNG được kéo userID (ngoài whitelist) sang")
	}
}
