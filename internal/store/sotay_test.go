package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sinh ra từ lượt soát bằng codex; mỗi cáo buộc đã đọc lại code để xác nhận.

// Sao lưu HỎNG thì bản sao lưu CŨ phải còn nguyên.
//
// Bản trước xoá đích trước rồi mới `VACUUM INTO`. Hết đĩa (hay bất cứ lý do gì)
// là mất bản cũ mà chưa có bản mới: người dùng đi sao lưu và kết quả là không còn
// bản nào. Ở đây ép VACUUM hỏng bằng cách trỏ đích vào một THƯ MỤC.
func TestSaoLuuHongThiKhongMatBanCu(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "state.db")
	d, err := OpenAt(p)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	tot := filepath.Join(dir, "tot.db")
	if err := d.Snapshot(tot); err != nil {
		t.Fatal(err)
	}
	truoc, err := os.ReadFile(tot)
	if err != nil {
		t.Fatal(err)
	}

	// Lần sao lưu thứ hai vào CÙNG đích, nhưng ép hỏng: chiếm chỗ file tạm bằng
	// một thư mục KHÔNG RỖNG. (Thư mục rỗng thì `os.Remove` dọn được — đã dẫm
	// đúng chỗ đó khi viết test này, và nó lặng lẽ chạy thành công.)
	chiem := tot + ".dang-chup"
	if err := os.MkdirAll(chiem, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chiem, "giu-cho.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := d.Snapshot(tot); err == nil {
		t.Fatal("Snapshot báo thành công dù không ghi nổi file tạm")
	}

	sau, err := os.ReadFile(tot)
	if err != nil {
		t.Fatalf("BẢN SAO LƯU CŨ ĐÃ MẤT sau một lần sao lưu hỏng: %v", err)
	}
	if len(sau) != len(truoc) {
		t.Fatalf("bản cũ bị sửa: %d byte, trước là %d", len(sau), len(truoc))
	}
}

// InUse: lỗi Stat KHÁC "không tồn tại" không được coi là "không ai dùng".
//
// Coi nó như đường quang nghĩa là cho `db restore` ghi đè trong khi không hề biết
// ai đang mở file — lá chắn tự tắt đúng lúc cần nhất.
func TestInUseKhongCoiLoiLaDuongQuang(t *testing.T) {
	dir := t.TempDir()

	// Không tồn tại -> đúng là không ai giữ.
	if err := InUse(filepath.Join(dir, "khong-co.db")); err != nil {
		t.Fatalf("file không tồn tại mà báo lỗi: %v", err)
	}

	// Đường dẫn không hợp lệ (có ký tự NUL) -> Stat lỗi khác IsNotExist.
	err := InUse(filepath.Join(dir, "hong\x00ten.db"))
	if err == nil {
		t.Fatal("Stat hỏng mà InUse vẫn bảo 'không ai dùng' — restore sẽ ghi đè mù")
	}
	if !strings.Contains(err.Error(), "kiểm ai đang dùng") {
		t.Errorf("thông điệp chưa nói rõ vì sao: %v", err)
	}
}

// Bước flow thử lại: về trạng thái CHƯA kết thúc thì `ended` phải bị xoá, không
// được giữ thời điểm kết thúc của lần chạy trước.
func TestThuLaiBuocThiXoaThoiDiemKetThuc(t *testing.T) {
	d := open(t)
	id, err := d.CreateRun("f", "d", "")
	if err != nil {
		t.Fatal(err)
	}
	// StepRun không phơi `ended` ra ngoài, nên hỏi thẳng DB.
	ended := func() string {
		var e *string
		if err := d.db.QueryRow(`SELECT ended FROM flow_steps WHERE run_id=? AND step_id=?`,
			id, "b1").Scan(&e); err != nil {
			t.Fatal(err)
		}
		if e == nil {
			return ""
		}
		return *e
	}

	if err := d.SetStep(id, "b1", StepFailed, "hỏng", 1); err != nil {
		t.Fatal(err)
	}
	if ended() == "" {
		t.Fatal("bước failed phải có thời điểm kết thúc")
	}

	// Thử lại: quay về running thì thời điểm kết thúc cũ phải biến mất, không
	// thì bảng hiện một bước "đang chạy" mà đã có giờ kết thúc.
	if err := d.SetStep(id, "b1", StepRunning, "", 2); err != nil {
		t.Fatal(err)
	}
	if got := ended(); got != "" {
		t.Fatalf("bước đang chạy lại mà vẫn mang thời điểm kết thúc cũ: %q", got)
	}
}
