package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// dbCu dựng một state.db ĐÚNG NHƯ bản sagent cũ ở schema v<n> để lại: chỉ chạy
// n migration đầu, schema_version = n.
//
// Không giả lập bằng cách hạ số phiên bản của một DB mới — làm vậy thì migration
// sau sẽ chạy lại trên bảng đã có và hỏng vì lý do KHÔNG liên quan tới thứ đang
// đo, rồi ta lại tưởng mình đo được cái gì đó.
func dbCu(t *testing.T, path string, n int) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if _, err := db.Exec(migrations[i]); err != nil {
			t.Fatalf("dựng DB cũ hỏng ở migration v%d: %v", i+1, err)
		}
	}
	if _, err := db.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('schema_version',?)`, strconv.Itoa(n)); err != nil {
		t.Fatal(err)
	}
}

// HẠ CẤP BINARY. Đo trước khi vá: bản cũ mở state.db của bản mới KHÔNG lỗi, đọc
// được, GHI ĐƯỢC (AddSession trả id=2), rồi để nguyên schema_version=99.
//
// Đó là ghi chéo phiên bản trong im lặng — bản cũ không biết ràng buộc mà bản
// mới đặt ra, nên nó ghi ra những dòng hợp lệ với nó và sai với bản mới. Không
// có tiếng động nào lúc xảy ra; chỉ lộ ra sau, ở chỗ khác, khi đã muộn.
func TestMoFileCuaBanMoiHonThiTuChoi(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.db")
	d, err := OpenAt(p)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	raw, err := sql.Open("sqlite", "file:"+p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('schema_version','99')`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	d2, err := OpenAt(p)
	if err == nil {
		d2.Close()
		t.Fatal("OpenAt chấp nhận file của bản mới hơn — bản cũ sẽ ghi chéo phiên bản trong im lặng")
	}
	for _, phai := range []string{"v99", "v" + strconv.Itoa(LatestSchema()), "db restore"} {
		if !contains(err.Error(), phai) {
			t.Errorf("thông điệp lỗi thiếu %q — người dùng phải biết mình đang ở đâu và làm gì tiếp: %v", phai, err)
		}
	}
}

// Nâng schema của một file ĐÃ CÓ DỮ LIỆU thì phải để lại bản sao lưu. Migration
// chạy trong transaction nên không nửa vời, nhưng transaction không cứu được
// một migration viết đúng cú pháp mà sai ý — nó commit gọn gàng rồi dữ liệu đi.
func TestNangSchemaThiSaoLuuTruoc(t *testing.T) {
	if LatestSchema() < 2 {
		t.Skip("cần ít nhất 2 migration mới đo được việc nâng cấp")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "state.db")
	dbCu(t, p, 1)

	d, err := OpenAt(p)
	if err != nil {
		t.Fatal(err)
	}
	v, err := d.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	d.Close()
	if v != LatestSchema() {
		t.Fatalf("sau khi mở, schema = v%d, muốn v%d", v, LatestSchema())
	}

	bak := autoBackupPath(p, 1)
	fi, err := os.Stat(bak)
	if err != nil {
		t.Fatalf("không có bản sao lưu trước khi nâng schema: %v", err)
	}
	if fi.Size() == 0 {
		t.Fatal("bản sao lưu rỗng")
	}
	// Bản sao lưu phải còn Ở PHIÊN BẢN CŨ, nếu không thì nó chẳng khôi phục về đâu.
	if got, err := inspect(bak); err != nil || got != 1 {
		t.Fatalf("bản sao lưu ở schema v%d (err=%v), muốn v1", got, err)
	}

	// File mới tinh thì không có gì để mất — đừng rác hoá thư mục.
	p2 := filepath.Join(dir, "moi.db")
	d2, err := OpenAt(p2)
	if err != nil {
		t.Fatal(err)
	}
	d2.Close()
	if m, _ := filepath.Glob(p2 + ".bak-v*"); len(m) != 0 {
		t.Fatalf("DB mới tinh không được sinh bản sao lưu, thấy %v", m)
	}
}

// Sao lưu rồi khôi phục phải quay đúng về trạng thái lúc chụp.
func TestSaoLuuRoiKhoiPhucQuayDungVeLucChup(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "state.db")
	d, err := OpenAt(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.AddSession(Session{Provider: "claude", Account: "truoc", PID: os.Getpid(), Dir: "d"}); err != nil {
		t.Fatal(err)
	}

	bak := filepath.Join(dir, "anh.db")
	if err := d.Snapshot(bak); err != nil {
		t.Fatal(err)
	}

	// Sau khi chụp mới thêm — cái này phải biến mất sau khi khôi phục.
	if _, err := d.AddSession(Session{Provider: "claude", Account: "sau", PID: os.Getpid(), Dir: "d"}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cuu, err := Restore(bak, p)
	if err != nil {
		t.Fatal(err)
	}
	if cuu == "" {
		t.Fatal("Restore không cứu bản hiện tại — một lệnh khôi phục không hoàn tác được chỉ là lệnh xoá có thêm bước")
	}
	if _, err := os.Stat(cuu); err != nil {
		t.Fatalf("bản cứu không tồn tại: %v", err)
	}

	d2, err := OpenAt(p)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	list, err := d2.Running()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Account != "truoc" {
		t.Fatalf("sau khôi phục phải chỉ còn phiên \"truoc\", được %+v", list)
	}

	// Bản cứu phải chứa ĐỦ trạng thái ngay trước khi khôi phục (cả "sau"), nếu
	// không thì nó không hoàn tác được gì. Đây cũng là chỗ chép thẳng file sẽ
	// hỏng: dữ liệu mới nhất còn nằm trong -wal.
	dc, err := OpenAt(cuu)
	if err != nil {
		t.Fatal(err)
	}
	defer dc.Close()
	lc, err := dc.Running()
	if err != nil {
		t.Fatal(err)
	}
	if len(lc) != 2 {
		t.Fatalf("bản cứu phải có đủ 2 phiên, được %d — snapshot đã bỏ sót phần nằm trong WAL", len(lc))
	}
}

// Khôi phục nhầm file là mất trắng, mà lúc đó mới biết thì đã muộn.
func TestRestoreTuChoiFileLa(t *testing.T) {
	dir := t.TempDir()
	la := filepath.Join(dir, "anh-meo.jpg")
	if err := os.WriteFile(la, []byte("khong phai sqlite"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "state.db")
	d, err := OpenAt(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.AddSession(Session{Provider: "claude", Account: "quan-trong", PID: os.Getpid(), Dir: "d"}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	if _, err := Restore(la, p); err == nil {
		t.Fatal("Restore nuốt một file không phải state.db")
	}
	if _, err := Restore(filepath.Join(dir, "khong-co-that.db"), p); err == nil {
		t.Fatal("Restore nuốt một đường dẫn không tồn tại")
	}

	// Quan trọng hơn cả hai cái trên: DB thật không được suy suyển sau khi từ chối.
	d2, err := OpenAt(p)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	list, _ := d2.Running()
	if len(list) != 1 || list[0].Account != "quan-trong" {
		t.Fatalf("DB bị động sau một lần khôi phục BỊ TỪ CHỐI: %+v", list)
	}
}

// Ghi đè file trong lúc tiến trình khác đang mở thì SQLite không chống được —
// nó không biết có ai vừa thay file dưới chân nó.
func TestInUsePhatHienKetNoiDangMo(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.db")
	if err := InUse(p); err != nil {
		t.Fatalf("file chưa tồn tại thì không ai giữ, được %v", err)
	}
	d, err := OpenAt(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := InUse(p); err == nil {
		t.Fatal("InUse không thấy kết nối đang mở — lá chắn của `db restore` vô dụng")
	}
	d.Close()
	if err := InUse(p); err != nil {
		t.Fatalf("đóng rồi mà vẫn báo bận: %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
