package profile

// XOÁ-AN-TOÀN THEO SỔ — hai ca cuối của bộ (a)–(f):
//
//	(e) sổ không nhận sở hữu → KHÔNG xoá, và thư mục phải còn NGUYÊN
//	(f) hồ sơ chính nó là link → chỉ gỡ link, KHÔNG hỏi sổ, KHÔNG xuyên qua
//
// Ca (f) trùng đề bài với TestHoSoChinhNoLaLinkThiKhongDuocXoaXuyenQua
// (junction_test.go) nhưng đo một đường khác: ở đó là `Remove`, ở đây là
// `RemoveTheoSo` — đường mà api dùng từ nay. Thêm một cửa vào cùng chỗ nguy
// hiểm thì phải đo lại chính cái cửa đó; lá chắn của cửa bên cạnh không tự bò
// sang.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/link"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// soGia là một sổ đăng ký giả: trả lời đúng những gì test dựng sẵn.
type soGia struct {
	soHuu map[string]bool
	loi   error
	hoi   int
}

func (s *soGia) SoHuu(dir string) (bool, error) {
	s.hoi++
	if s.loi != nil {
		return false, s.loi
	}
	return s.soHuu[filepath.Clean(dir)], nil
}

// dungHoSo dựng một thư mục hồ sơ thật có file bên trong, trả về đường dẫn.
func dungHoSo(t *testing.T, prov, acc string) string {
	t.Helper()
	dir := Dir(prov, acc)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// ---- ca (e): sổ không nhận sở hữu thì KHÔNG được xoá ----
//
// Đây là cả lý do cái sổ tồn tại. Trước nó, lá chắn duy nhất là `insideStore` —
// một phép kiểm CHUỖI: hễ đường dẫn nằm dưới ~/.ai-accounts thì mặc nhiên xoá đệ
// quy được. Thư mục người dùng tự chép vào kho để dành cũng thoả điều kiện đó.
func TestCaE_SoKhongNhanSoHuuThiKhongXoa(t *testing.T) {
	homeGia(t)
	dir := dungHoSo(t, "claude", "cua-nguoi-khac")

	so := &soGia{soHuu: map[string]bool{}} // sổ biết, nhưng không nhận sở hữu
	err := RemoveTheoSo(dir, so)
	if err == nil {
		t.Fatal("XOÁ MẤT thư mục mà sổ không nhận sở hữu")
	}
	if so.hoi == 0 {
		t.Fatal("không hề hỏi sổ — quyết định dựa vào cái gì?")
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".credentials.json")); statErr != nil {
		t.Fatalf("thư mục bị động dù đã từ chối: %v", statErr)
	}

	// Sổ đọc HỎNG cũng là "không xoá": làm ngược lại nghĩa là một cái DB bị khoá
	// đủ để biến xoá-an-toàn về lại xoá-đệ-quy, trong im lặng.
	if err := RemoveTheoSo(dir, &soGia{loi: errors.New("database is locked")}); err == nil {
		t.Fatal("sổ đọc hỏng mà vẫn xoá")
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Fatalf("thư mục bị xoá khi sổ đọc hỏng: %v", statErr)
	}

	// Chiều ngược lại: sổ nhận sở hữu thì phải xoá được thật, nếu không thì cái
	// lá chắn này chỉ là một cách làm hỏng `xoa`.
	so.soHuu[filepath.Clean(dir)] = true
	if err := RemoveTheoSo(dir, so); err != nil {
		t.Fatalf("sổ nhận sở hữu mà vẫn không xoá được: %v", err)
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		t.Fatal("báo xoá xong nhưng thư mục vẫn còn")
	}

	// Và lá chắn cũ vẫn phải đứng đó: sổ nói "có" cho một đường dẫn NGOÀI kho
	// cũng không mở được cửa. Sổ là dữ liệu, mà dữ liệu thì có thể sai.
	ngoai := filepath.Join(paths.Home(), ".claude")
	if err := os.MkdirAll(ngoai, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RemoveTheoSo(ngoai, &soGia{soHuu: map[string]bool{filepath.Clean(ngoai): true}}); err == nil {
		t.Fatal("một dòng sổ bịa ra đủ để xoá thứ NGOÀI kho hồ sơ")
	}
	if _, statErr := os.Stat(ngoai); statErr != nil {
		t.Fatalf("~/.claude BỊ XOÁ: %v", statErr)
	}
}

// ---- ca (f): hồ sơ chính nó là link ----
//
// Ở đây sổ nói "CÓ, tôi sở hữu" cho đúng đường dẫn của cái link — tình huống tệ
// nhất, và không hề xa vời: thư mục do sagent tạo thật (nên vào sổ hợp lệ), sau
// đó bị thay bằng junction. Nếu `RemoveTheoSo` hỏi sổ trước rồi tin câu trả lời
// mà gọi `os.RemoveAll`, nó sẽ đi xuyên link — đúng cú đã xoá mất ~/.claude ngày
// 2026-08-17.
//
// Luật đúng: gặp link thì KHÔNG hỏi sổ gì cả, chỉ gỡ cái link. Gỡ junction là
// tháo một cánh cửa, không đụng gì tới căn phòng bên kia.
func TestCaF_HoSoLaLinkThiChiGoLinkVaKhongHoiSo(t *testing.T) {
	home := homeGia(t)

	that := filepath.Join(home, ".claude")
	if err := os.MkdirAll(that, 0o755); err != nil {
		t.Fatal(err)
	}
	moi := filepath.Join(that, "quan-trong.txt")
	if err := os.WriteFile(moi, []byte("DỮ LIỆU THẬT"), 0o644); err != nil {
		t.Fatal(err)
	}

	kho := filepath.Join(paths.AccountsRoot(), "claude")
	if err := os.MkdirAll(kho, 0o755); err != nil {
		t.Fatal(err)
	}
	bay := filepath.Join(kho, "evil")
	if err := link.LinkDir(that, bay); err != nil {
		t.Skipf("không tạo được link thư mục ở môi trường này: %v", err)
	}
	if laLink, _ := link.IsLink(bay); !laLink {
		t.Fatalf("bẫy dựng hỏng: %s không phải link, phép đo này vô nghĩa", bay)
	}

	so := &soGia{soHuu: map[string]bool{filepath.Clean(bay): true}}
	if err := RemoveTheoSo(bay, so); err != nil {
		t.Fatalf("xoá một hồ sơ là link vẫn phải làm được (gỡ chính cái link): %v", err)
	}
	if so.hoi != 0 {
		t.Fatal("đã hỏi sổ về một cái LINK — câu trả lời \"có sở hữu\" ở đó là giấy phép " +
			"xoá đệ quy xuyên qua link")
	}
	if _, statErr := os.Lstat(bay); statErr == nil {
		t.Errorf("link hồ sơ %s vẫn còn — chưa gỡ", bay)
	}
	if b, err := os.ReadFile(moi); err != nil || string(b) != "DỮ LIỆU THẬT" {
		t.Fatalf("DỮ LIỆU THẬT Ở ĐẦU BÊN KIA BỊ ĐỘNG: %q, %v", b, err)
	}

	// SagentQuan là bằng chứng-trên-đĩa mà tầng api dùng để NHẬN hồ sơ cũ vào sổ.
	// Nó không được nhận một cái link là "của sagent" — làm vậy thì ca này tự
	// dựng lại chính mình qua đường nhận-vào-sổ.
	lai := filepath.Join(kho, "evil2")
	if err := link.LinkDir(that, lai); err != nil {
		t.Skipf("không tạo lại được link: %v", err)
	}
	if SagentQuan(lai) {
		t.Fatal("SagentQuan nhận một LINK là thư mục do sagent tạo ra")
	}
	that2 := dungHoSo(t, "claude", "that")
	if !SagentQuan(that2) {
		t.Fatal("SagentQuan không nhận một thư mục hồ sơ THẬT trong kho")
	}
	if SagentQuan(filepath.Join(home, "cho-khac")) {
		t.Fatal("SagentQuan nhận cả thứ nằm ngoài kho hồ sơ")
	}
}
