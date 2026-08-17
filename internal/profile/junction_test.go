package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/link"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// Junction-attack (DoD Pha 7). Khác hai phép đo trước ở một điểm quyết định:
//
//   - "xoá không xuyên link" (profile_test.go) đo link nằm BÊN TRONG thư mục hồ sơ.
//   - "tên không thoát kho" (name_test.go) đo đường dẫn trỏ RA NGOÀI kho về mặt chữ.
//
// Ở đây thư mục hồ sơ **chính nó** là link: đường dẫn hợp lệ hoàn toàn về mặt
// chữ (`~/.ai-accounts/claude/evil` nằm trong kho, `insideStore` cho qua), nhưng
// cái tên đó là một cánh cửa mở sang `~/.claude`. Lá chắn của hai lớp trước là
// kiểm chuỗi, mà kiểm chuỗi thì không nhìn thấy reparse point.
//
// Đo được gì khi viết test này (Windows, go1.25.13):
//
//	os.Lstat(junction).Mode() -> ModeIrregular, KHÔNG phải ModeSymlink, IsDir()=false
//	link.IsLink(junction)     -> true (kiểm cờ FILE_ATTRIBUTE_REPARSE_POINT)
//	os.ReadDir(junction)      -> ĐI XUYÊN, liệt kê ruột thư mục thật
//
// Vế thứ ba là chỗ nổ: trước bản vá, `Remove` gỡ mất junction `~/.claude/skills`
// của người dùng rồi trả về `nil`. Dữ liệu không mất, nhưng cấu trúc link dùng
// chung bị phá trong im lặng — thứ khó phát hiện hơn hẳn một lần xoá ồn ào.
//
// Vì sao đo lớp lỗi này chứ không lớp khác: đúng nó đã **nổ thật** trên máy dev
// ngày 2026-08-17 và xoá mất `~/.claude` (xem docs/DO-LUONG.md). Lần đó thủ phạm
// là script chứ không phải hàm này — nhưng hàm này gọi `os.RemoveAll`, nên phải
// có bằng chứng nó không lặp lại được cú đó.
//
// Tạo junction bằng `mklink /J` KHÔNG cần quyền quản trị: kẻ tấn công chỉ cần
// ghi được vào kho hồ sơ là dựng xong bẫy. Trên Linux tương đương là symlink.
func TestHoSoChinhNoLaLinkThiKhongDuocXoaXuyenQua(t *testing.T) {
	home := homeGia(t)

	// Dữ liệu dùng chung thật, ở đầu bên kia của link lồng bên trong nạn nhân.
	shared := filepath.Join(home, "shared")
	if err := os.MkdirAll(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "skill.md"), []byte("DÙNG CHUNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Nạn nhân: ~/.claude thật, bên trong có junction trỏ ra ~/shared — đúng
	// cách chính dự án này nối phần dùng chung.
	that := filepath.Join(home, ".claude")
	if err := os.MkdirAll(that, 0o755); err != nil {
		t.Fatal(err)
	}
	moi := filepath.Join(that, "quan-trong.txt")
	if err := os.WriteFile(moi, []byte("DỮ LIỆU THẬT"), 0o644); err != nil {
		t.Fatal(err)
	}
	noiChung := filepath.Join(that, "skills")
	if err := link.LinkDir(shared, noiChung); err != nil {
		t.Skipf("không tạo được link thư mục ở môi trường này: %v", err)
	}

	// Cái bẫy: một "hồ sơ" tên hợp lệ, nằm đúng trong kho, nhưng là link.
	kho := filepath.Join(paths.AccountsRoot(), "claude")
	if err := os.MkdirAll(kho, 0o755); err != nil {
		t.Fatal(err)
	}
	bay := filepath.Join(kho, "evil")
	if err := link.LinkDir(that, bay); err != nil {
		t.Skipf("không tạo được link thư mục ở môi trường này: %v", err)
	}
	if isLink, _ := link.IsLink(bay); !isLink {
		t.Fatalf("bẫy dựng hỏng: %s không phải link, phép đo này vô nghĩa", bay)
	}
	// Tên "evil" hợp lệ — cố ý, để phép đo chạm đúng vào lớp cuối cùng chứ không
	// bị ValidName chặn hộ ở lớp trên.
	if err := ValidName("evil"); err != nil {
		t.Fatalf("tiền đề sai: %q lẽ ra hợp lệ, được %v", "evil", err)
	}

	err := Remove(Dir("claude", "evil"))
	if err != nil {
		t.Fatalf("Remove trả lỗi %v — xoá một hồ sơ là link vẫn phải làm được (gỡ chính cái link)", err)
	}

	// 1. Bản thân cái bẫy phải biến mất: người dùng gõ `xoa` thì hồ sơ phải đi.
	if _, statErr := os.Lstat(bay); statErr == nil {
		t.Errorf("link hồ sơ %s vẫn còn — Remove chưa gỡ nó", bay)
	}

	// 2. Thư mục thật và nội dung còn nguyên.
	if _, statErr := os.Stat(that); statErr != nil {
		t.Fatalf("THƯ MỤC THẬT BỊ XOÁ QUA LINK: %v", statErr)
	}
	if b, readErr := os.ReadFile(moi); readErr != nil || string(b) != "DỮ LIỆU THẬT" {
		t.Fatalf("DỮ LIỆU THẬT BỊ ĐỘNG QUA LINK: %q, %v", b, readErr)
	}

	// 3. Chỗ từng nổ: link LỒNG BÊN TRONG nạn nhân phải còn. Trước bản vá,
	//    os.ReadDir đi xuyên bẫy nên vòng gỡ link tháo mất đúng cái này.
	if _, statErr := os.Lstat(noiChung); statErr != nil {
		t.Fatalf("LINK DÙNG CHUNG BÊN TRONG NẠN NHÂN BỊ GỠ: %v", statErr)
	}
	if isLink, _ := link.IsLink(noiChung); !isLink {
		t.Fatalf("%s không còn là link — cấu trúc dùng chung bị phá", noiChung)
	}
	if b, readErr := os.ReadFile(filepath.Join(shared, "skill.md")); readErr != nil || string(b) != "DÙNG CHUNG" {
		t.Fatalf("dữ liệu dùng chung bị động: %q, %v", b, readErr)
	}
}
