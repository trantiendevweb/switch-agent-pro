package profile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Vỏ npm THẬT trên máy này (grok.cmd), chép nguyên văn.
//
// Bản đầu của GoiThat bám vào chuỗi "dp0" và TRƯỢT trên đúng file này: vỏ có
// nhiều chỗ dp0 (:find_dp0, SET dp0=%~dp0) nên regex bám nhầm chỗ, mà [^"]* lại
// vắt qua cả xuống dòng. Trượt mà vẫn build, vẫn chạy — chỉ là không gỡ được vỏ
// và prompt tiếp tục bị cắt còn một dòng. Giữ nguyên văn ở đây để không trượt lại.
const voNpmThat = `@ECHO off
GOTO start
:find_dp0
SET dp0=%~dp0
EXIT /b
:start
SETLOCAL
CALL :find_dp0

IF EXIST "%dp0%\node.exe" (
  SET "_prog=%dp0%\node.exe"
) ELSE (
  SET "_prog=node"
)

endLocal & goto #_undefined_# 2>NUL || title %COMSPEC% & "%_prog%"  "%dp0%\node_modules\@vibe-kit\grok-cli\dist\index.js" %*
`

func TestGoiThatGoDuocVoNpm(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("chỉ có nghĩa trên Windows")
	}
	dir := t.TempDir()
	shim := filepath.Join(dir, "grok.cmd")
	if err := os.WriteFile(shim, []byte(voNpmThat), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "node_modules", "@vibe-kit", "grok-cli", "dist", "index.js")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("// giả"), 0o644); err != nil {
		t.Fatal(err)
	}

	thuc, dau := GoiThat(shim)
	if strings.EqualFold(thuc, shim) {
		t.Fatal("KHÔNG gỡ được vỏ .cmd — prompt nhiều dòng sẽ tiếp tục bị cắt còn một dòng")
	}
	if len(dau) != 1 || !strings.EqualFold(dau[0], script) {
		t.Fatalf("gỡ ra sai script: %v, muốn %q", dau, script)
	}
}

// File thường thì phải trả NGUYÊN đường cũ — thà chạy như cũ còn hơn đoán sai
// rồi bật nhầm chương trình.
func TestGoiThatKhongDungVaoFileThuong(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "agy.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o755); err != nil {
		t.Fatal(err)
	}
	thuc, dau := GoiThat(exe)
	if thuc != exe || dau != nil {
		t.Fatalf("đổi nhầm file thường: %q %v", thuc, dau)
	}
}

// Vỏ .cmd KHÔNG phải kiểu npm (không trỏ tới .js) thì cũng giữ nguyên.
func TestGoiThatGiuNguyenVoLa(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("chỉ có nghĩa trên Windows")
	}
	dir := t.TempDir()
	shim := filepath.Join(dir, "la.cmd")
	voLa := "@echo off\r\n" + `"C:\Program Files\thu\thu.exe" %*` + "\r\n"
	if err := os.WriteFile(shim, []byte(voLa), 0o755); err != nil {
		t.Fatal(err)
	}
	thuc, _ := GoiThat(shim)
	if thuc != shim {
		t.Fatalf("vỏ lạ mà vẫn đổi: %q", thuc)
	}
}
