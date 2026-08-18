package profile

import (
	"os"
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
