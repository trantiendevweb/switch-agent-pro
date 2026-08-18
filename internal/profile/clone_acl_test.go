package profile

import (
	"os"
	"time"
	"os/exec"
	"runtime"
	"path/filepath"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/acl"
	"github.com/trantiendevweb/switch-agent-pro/internal/link"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// Thư mục clone là chỗ token bị NHÂN RA N BẢN. Một hồ sơ hở là hở một token;
// một kho clone hở là hở N. `profile.Create` đã siết ACL từ trước, nhưng chỗ này
// thì quên — đúng cái chỗ đáng siết nhất.
func TestThuMucCloneCungPhaiSietQuyen(t *testing.T) {
	home := homeGia(t)

	fakeBase := filepath.Join(home, "fakebase")
	if err := os.MkdirAll(fakeBase, 0o755); err != nil {
		t.Fatal(err)
	}
	// Hồ sơ gốc phải tồn tại trong kho thì mới nhân bản được.
	base := filepath.Join(home, ".ai-accounts", "fake", "phu")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".credentials.json"), []byte(`{"t":"TOKEN"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a := fakeAdapter{base: fakeBase, hasToken: true}

	// NỚI LỎNG thư mục cha trước. Không có bước này thì test rỗng nghĩa: thư mục
	// clone vốn đã kín nhờ kế thừa, nên nó xanh kể cả khi bản vá bị gỡ — đã dẫm
	// đúng chỗ đó khi viết test này.
	if err := os.MkdirAll(ClonesRoot(), 0o755); err != nil {
		t.Fatal(err)
	}
	noiLongCho(t, ClonesRoot())
	if ok, _, _ := acl.Check(ClonesRoot()); ok {
		t.Skip("không nới lỏng được ACL để dựng bẫy — bỏ qua, đừng báo xanh giả")
	}

	dirs, err := Clone(a, "phu", 2)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if len(dirs) != 2 {
		t.Fatalf("muốn 2 clone, được %d", len(dirs))
	}
	for _, d := range dirs {
		ok, detail, err := acl.Check(d)
		if err != nil {
			t.Fatalf("không đọc được quyền %s: %v", d, err)
		}
		if !ok {
			t.Errorf("thư mục clone %s đang hở: %s", d, detail)
		}
	}
}

// CleanClones phải kiểm CHÍNH thư mục gốc có phải link không, trước khi ReadDir.
//
// Cùng lớp lỗi đã vá ở `Remove` nhưng sót ở tầng gốc: root là junction trỏ ra
// ngoài thì ReadDir đi xuyên, và mỗi thư mục con THẬT bên kia bị xoá — trong khi
// đường dẫn vẫn nằm gọn trong kho nên `insideStore` chẳng thấy gì bất thường.
func TestCleanClonesKhongXuyenJunctionOGoc(t *testing.T) {
	home := homeGia(t)

	// "Nạn nhân": thư mục thật, có dữ liệu, nằm ngoài kho.
	nanNhan := filepath.Join(home, "du-lieu-that")
	con := filepath.Join(nanNhan, "quan-trong")
	if err := os.MkdirAll(con, 0o755); err != nil {
		t.Fatal(err)
	}
	moi := filepath.Join(con, "giu-lai.txt")
	if err := os.WriteFile(moi, []byte("DỮ LIỆU THẬT"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Bẫy: gốc kho clone của một tài khoản LÀ junction trỏ sang nạn nhân.
	root := filepath.Join(ClonesRoot(), "fake", "phu")
	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := link.LinkDir(nanNhan, root); err != nil {
		t.Skipf("không tạo được link ở môi trường này: %v", err)
	}

	if _, err := CleanClones("fake", "phu"); err != nil {
		t.Fatalf("CleanClones: %v", err)
	}

	if _, err := os.Stat(moi); err != nil {
		t.Fatalf("DỮ LIỆU THẬT BỊ XOÁ QUA JUNCTION Ở GỐC: %v", err)
	}
	if _, err := os.Stat(con); err != nil {
		t.Fatalf("THƯ MỤC THẬT BỊ XOÁ QUA JUNCTION Ở GỐC: %v", err)
	}
}

// Lỗi đọc file token KHÁC "không tồn tại" không được im lặng bỏ qua: clone sẽ
// chạy mà thiếu token, agent đăng nhập hụt, người dùng đi tìm nguyên nhân ở tận đâu.
func TestCloneKhongImLangBoQuaLoiDocToken(t *testing.T) {
	homeGia(t)
	ad, ok := provider.Get("claude")
	if !ok {
		t.Skip("không có adapter claude")
	}
	// Không có base dir -> HasToken false -> Clone phải từ chối ngay, có lý do.
	if _, err := Clone(ad, "khong-co-that", 1); err == nil {
		t.Fatal("Clone chấp nhận tài khoản chưa đăng nhập")
	}
}

// noiLongCho mở rộng quyền một thư mục để dựng đúng cái bẫy cần đo. Dùng SID
// (`*S-1-5-32-545` = nhóm Users dựng sẵn) chứ không dùng tên nhóm: tên đổi theo
// ngôn ngữ Windows, SID thì không.
func noiLongCho(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		return
	}
	_ = exec.Command("icacls", dir, "/grant", "*S-1-5-32-545:(OI)(CI)F").Run()
}

// Token khai bằng ĐƯỜNG DẪN LỒNG (Cursor/auth.json) vẫn phải được coi là RIÊNG.
//
// Lỗi đã đo trên máy: `LinkShared` so tên cơ sở `auth.json` với danh sách riêng
// tư `Cursor\auth.json`, không khớp, nên token bị NỐI LINK về file gốc. Hồ sơ mới
// hiện `auth.json [SymbolicLink]`. Nghĩa là mọi hồ sơ dùng chung một danh tính —
// đúng thứ công cụ này sinh ra để tránh.
//
// Nó vô hại chỉ vì Cursor tìm token ở tầng khác, tức là MAY chứ không phải đúng.
func TestTokenKhaiDuongDanLongVanLaRiengTu(t *testing.T) {
	home := homeGia(t)

	// Base có đúng một file, và file đó là token.
	base := filepath.Join(home, "base-long")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "auth.json"), []byte(`{"t":"TOKEN"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(home, "ho-so")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	n, err := LinkShared(adapterLong{base: base}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("nối %d mục dùng chung — token lẽ ra phải bị coi là riêng tư", n)
	}
	if _, err := os.Lstat(filepath.Join(dir, "auth.json")); err == nil {
		t.Fatal("token đã bị nối link vào hồ sơ mới — mọi hồ sơ sẽ dùng chung một danh tính")
	}
}

// adapterLong khai token bằng đường dẫn lồng, y như adapter cursor thật.
type adapterLong struct{ base string }

func (adapterLong) Name() string                     { return "long" }
func (adapterLong) EnvVar() string                   { return "APPDATA" }
func (adapterLong) Command() (string, error)         { return "", nil }
func (adapterLong) Version() (string, error)         { return "long 0.0.0", nil }
func (adapterLong) HeadlessArgs(p string) []string   { return []string{"-p", p} }
func (adapterLong) PrivateFiles() []string           { return []string{filepath.Join("Cursor", "auth.json")} }
func (adapterLong) SharedKeys() []string             { return nil }
func (a adapterLong) BaseDir() string                { return a.base }
func (adapterLong) IdentitySource() string           { return "" }
func (adapterLong) Identity(string) string           { return "" }
func (adapterLong) HasToken(string) bool             { return false }
func (adapterLong) TokenExpiry(string) (time.Time, bool) { return time.Time{}, false }
func (adapterLong) Verify() []provider.Check         { return nil }
