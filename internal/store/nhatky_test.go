package store

import (
	"path/filepath"
	"testing"
)

// Hai đường hỏi sổ mà nhật ký phiên dựa vào.
//
// `sagent nhat-ky <id>` hỏi ĐÚNG MỘT phiên, và phiên đáng hỏi nhất là phiên ĐÃ
// CHẾT — lúc đó Running() không còn thấy nó, còn PhienChet() thì có trần nên
// phiên đủ cũ rơi ra ngoài. Không có đường hỏi thẳng theo số thì cột `log` chỉ
// tra được bằng tay.

func moSo(t *testing.T) *DB {
	t.Helper()
	db, err := OpenAt(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestPhienHoiDuocTheoSoKeCaKhiDaKetThuc(t *testing.T) {
	db := moSo(t)
	id, err := db.AddSession(Session{
		Provider: "claude", Account: "tns", Clone: 1, Dir: "d", PID: 123,
		Log: `C:\nk\a.log`, Worktree: `C:\wt\a`,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Đẩy nó sang một trạng thái đã kết thúc: đây mới là ca cần hỏi.
	if err := db.SetStateChiTiet(id, StateHanMuc, "hết hạn mức", 1755700000); err != nil {
		t.Fatal(err)
	}

	s, err := db.Phien(id)
	if err != nil {
		t.Fatal(err)
	}
	if s.Log != `C:\nk\a.log` || s.Worktree != `C:\wt\a` || s.State != StateHanMuc {
		t.Fatalf("Phien() trả về %+v", s)
	}
	if s.HanMucDenLai != 1755700000 || s.StateLyDo != "hết hạn mức" {
		t.Fatalf("mất chi tiết trạng thái: %+v", s)
	}
	// Số không có thật phải báo lỗi ĐỌC ĐƯỢC, không phải trả về Session rỗng —
	// Session rỗng sẽ thành "phiên #0 không có nhật ký" ở mặt trên.
	if _, err := db.Phien(99999); err == nil {
		t.Fatal("hỏi phiên không tồn tại mà không báo lỗi")
	}
}

// Phép đọc KHÔNG được có tác dụng phụ lên trạng thái. Running() cố ý đánh dấu
// `lost` cho PID đã chết; Phien() thì không được — nó phục vụ một lệnh xem.
func TestPhienLaPhepDocThuanTuy(t *testing.T) {
	db := moSo(t)
	id, err := db.AddSession(Session{Provider: "claude", Account: "tns", Dir: "d", PID: 0x7FFFFFF0})
	if err != nil {
		t.Fatal(err)
	}
	s, err := db.Phien(id)
	if err != nil {
		t.Fatal(err)
	}
	if s.State != StateRunning {
		t.Fatalf("Phien() đổi trạng thái thành %q — phép đọc không được có tác dụng phụ", s.State)
	}
}

// Bảng nhật ký liệt kê CẢ phiên sống lẫn phiên chết, mới nhất trước: người vận
// hành hỏi "lượt vừa rồi có những phiên nào", không hỏi "phiên nào còn sống".
func TestPhienGanDayLayCaSongLanChetMoiNhatTruoc(t *testing.T) {
	db := moSo(t)
	var ids []int64
	for i := 0; i < 4; i++ {
		id, err := db.AddSession(Session{Provider: "claude", Account: "tns", Dir: "d", PID: 100 + i})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := db.SetState(ids[0], StateXong); err != nil {
		t.Fatal(err)
	}
	if err := db.SetState(ids[1], StateStopped); err != nil {
		t.Fatal(err)
	}

	ds, err := db.PhienGanDay(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 4 {
		t.Fatalf("PhienGanDay lấy %d phiên, chờ 4 (cả sống lẫn chết)", len(ds))
	}
	for i := 1; i < len(ds); i++ {
		if ds[i-1].ID < ds[i].ID {
			t.Fatalf("không xếp mới nhất trước: %d rồi %d", ds[i-1].ID, ds[i].ID)
		}
	}
	// Trần phải có hiệu lực: sổ giữ mọi phiên từng chạy, không chặn thì bảng
	// dài dần theo tháng.
	if ds, err := db.PhienGanDay(2); err != nil || len(ds) != 2 {
		t.Fatalf("PhienGanDay(2) = %d phiên, lỗi %v", len(ds), err)
	}
}
