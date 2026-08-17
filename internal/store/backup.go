package store

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Sao lưu / khôi phục state.db (Pha 7).
//
// Vì sao KHÔNG chép thẳng file: state.db chạy ở chế độ WAL, nên phần dữ liệu mới
// nhất nằm trong `state.db-wal` chứ chưa vào file chính. Chép mình file chính ra
// được một bản **thiếu mà trông như đủ** — kiểu hỏng tệ nhất, vì chỉ lộ ra đúng
// lúc cần khôi phục. `VACUUM INTO` để SQLite tự gộp WAL và ghi ra một file nhất
// quán, chạy được cả khi đang có kết nối khác.

// LatestSchema là phiên bản schema mà bản binary NÀY biết.
func LatestSchema() int { return len(migrations) }

// SchemaVersion đọc phiên bản schema đang nằm trong file.
func (d *DB) SchemaVersion() (int, error) { return schemaVersion(d.db) }

func schemaVersion(db *sql.DB) (int, error) {
	var s string
	err := db.QueryRow(`SELECT value FROM meta WHERE key='schema_version'`).Scan(&s)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("schema_version không phải số: %q", s)
	}
	return n, nil
}

// autoBackupPath là tên bản sao lưu tự động sinh TRƯỚC khi nâng schema. Gắn số
// phiên bản cũ vào tên để biết bản đó khôi phục về đâu.
func autoBackupPath(path string, ver int) string {
	return fmt.Sprintf("%s.bak-v%d", path, ver)
}

// Snapshot ghi một bản sao nhất quán của cơ sở dữ liệu ra dst.
func (d *DB) Snapshot(dst string) error { return snapshot(d.db, dst) }

func snapshot(db *sql.DB, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	// VACUUM INTO từ chối ghi đè file đã có — dọn trước cho thao tác lặp lại được.
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	// VACUUM INTO không nhận tham số bind, phải nối chuỗi. Nháy đơn trong đường
	// dẫn nhân đôi theo đúng luật của SQL — đường dẫn đến từ người dùng
	// (`sagent db backup <file>`), nên chỗ này là chỗ tiêm SQL nếu ẩu.
	_, err := db.Exec(`VACUUM INTO '` + strings.ReplaceAll(dst, "'", "''") + `'`)
	return err
}

// Restore ghi đè cơ sở dữ liệu ở dst bằng bản sao lưu src.
//
// PHẢI gọi khi KHÔNG có tiến trình nào đang mở dst (dừng dash trước). Hàm này
// tự bảo vệ theo đúng thứ tự đã học được từ những lần hỏng trước:
//
//  1. Kiểm src có thật là state.db không, và schema của nó binary này có đọc
//     nổi không — khôi phục nhầm file là mất trắng, mà lúc đó mới biết thì muộn.
//  2. Chụp ảnh bản HIỆN TẠI trước khi ghi đè, để chính thao tác khôi phục cũng
//     hoàn tác được. Một lệnh khôi phục không hoàn tác được thì chỉ là một lệnh
//     xoá có thêm bước.
//  3. Mới ghi đè, và dọn -wal/-shm cũ: WAL sót lại của file cũ mà đứng cạnh
//     file mới thì SQLite sẽ áp nhầm dữ liệu của bản trước lên bản vừa khôi phục.
func Restore(src, dst string) (backupOfCurrent string, err error) {
	if _, err := os.Stat(src); err != nil {
		return "", fmt.Errorf("không đọc được bản sao lưu %s: %w", src, err)
	}
	ver, err := inspect(src)
	if err != nil {
		return "", err
	}
	if ver > LatestSchema() {
		return "", fmt.Errorf("bản sao lưu ở schema v%d, bản sagent này chỉ biết tới v%d — nâng cấp sagent trước khi khôi phục", ver, LatestSchema())
	}

	// Bước 2: cứu bản hiện tại (nếu có) trước khi đụng vào nó.
	if _, err := os.Stat(dst); err == nil {
		cur, err := sql.Open("sqlite", "file:"+dst)
		if err != nil {
			return "", err
		}
		backupOfCurrent = dst + ".truoc-khi-khoi-phuc"
		serr := snapshot(cur, backupOfCurrent)
		cur.Close()
		if serr != nil {
			return "", fmt.Errorf("không cứu được bản hiện tại, KHÔNG khôi phục: %w", serr)
		}
	}

	if err := copyFile(src, dst); err != nil {
		return backupOfCurrent, fmt.Errorf("ghi đè hỏng: %w", err)
	}
	// Bước 3: WAL/shm của bản CŨ phải đi cùng bản cũ.
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(dst + suffix); err != nil && !os.IsNotExist(err) {
			return backupOfCurrent, fmt.Errorf("không dọn được %s: %w", dst+suffix, err)
		}
	}
	return backupOfCurrent, nil
}

// inspect mở file chỉ để đọc phiên bản schema — không migrate, không sửa gì.
func inspect(path string) (int, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='meta'`).Scan(&n); err != nil {
		return 0, fmt.Errorf("%s không phải state.db đọc được: %w", path, err)
	}
	if n == 0 {
		return 0, fmt.Errorf("%s không có bảng meta — đây không phải state.db của sagent", path)
	}
	return schemaVersion(db)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".dang-ghi"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	// fsync trước khi đổi tên: mất điện giữa chừng thì thà không có file mới còn
	// hơn có một file mới rỗng ruột đè lên bản đang dùng được.
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// InUse trả lỗi nếu còn tiến trình khác đang mở cơ sở dữ liệu.
//
// Cách đo: xin `locking_mode=EXCLUSIVE` rồi ép SQLite thật sự lấy khoá bằng một
// giao dịch ghi, với busy_timeout(0) để nó báo ngay thay vì chờ. Ở chế độ WAL,
// khoá độc quyền chỉ lấy được khi KHÔNG còn kết nối nào khác — nên lấy được là
// bằng chứng đường ta đang quang, chứ không phải phỏng đoán.
//
// Giới hạn đã biết, nói trước cho khỏi tin nhầm: nó phát hiện *kết nối đang mở*,
// không phát hiện một tiến trình sagent vừa đóng kết nối và sắp mở lại.
func InUse(path string) error {
	if _, err := os.Stat(path); err != nil {
		return nil // chưa có file thì không ai giữ
	}
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(0)")
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA locking_mode=EXCLUSIVE`); err != nil {
		return err
	}
	if _, err := db.Exec(`BEGIN EXCLUSIVE; COMMIT;`); err != nil {
		return fmt.Errorf("%s đang được tiến trình khác dùng (%v)", path, err)
	}
	return nil
}
