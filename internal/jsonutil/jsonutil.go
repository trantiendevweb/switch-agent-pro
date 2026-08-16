// Package jsonutil làm việc với .claude.json.
//
// Vì sao Go giải được bài của v1: encoding/json khi gặp khoá JSON trùng
// hoa/thường (file thật hay có "C:\\...Du An" và "c:\\...du an") thì lấy khoá
// CUỐI, không ném lỗi — nên bỏ được phụ thuộc Python mà v1 phải dùng.
package jsonutil

import (
	"encoding/json"
	"os"
	"time"
)

// ReadObject đọc JSON gốc thành map (giữ nguyên giá trị dạng thô).
func ReadObject(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// AtomicWrite ghi ra file tạm rồi đổi tên — không bao giờ để lại file nửa vời.
func AtomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Backup lưu bản sao kèm dấu thời gian (file này chứa trust dialog, hỏng là
// phải bấm lại hết).
func Backup(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	bak := path + ".bak-" + time.Now().Format("20060102-150405")
	return os.WriteFile(bak, data, 0o600)
}

// Seed tạo file đích CHỈ từ whitelist keys. Cố ý KHÔNG chép cả file rồi xoá
// vài khoá — làm vậy là mang theo cache gói cước của tài khoản cũ.
func Seed(src, dst string, keys []string) (int, error) {
	obj, err := ReadObject(src)
	if err != nil {
		return 0, err
	}
	out := map[string]json.RawMessage{}
	for _, k := range keys {
		if v, ok := obj[k]; ok {
			out[k] = v
		}
	}
	data, err := json.Marshal(out)
	if err != nil {
		return 0, err
	}
	if err := AtomicWrite(dst, data); err != nil {
		return 0, err
	}
	return len(out), nil
}

// SyncKeys đẩy whitelist từ src sang dst, chỉ ghi khi có thay đổi, có sao lưu.
// Trả về số khoá đã đổi.
func SyncKeys(src, dst string, keys []string) (int, error) {
	srcObj, err := ReadObject(src)
	if err != nil {
		return 0, err
	}
	dstObj, err := ReadObject(dst)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, k := range keys {
		sv, ok := srcObj[k]
		if !ok {
			continue
		}
		if dv, dok := dstObj[k]; !dok || string(dv) != string(sv) {
			dstObj[k] = sv
			changed++
		}
	}
	if changed == 0 {
		return 0, nil
	}
	if err := Backup(dst); err != nil {
		return 0, err
	}
	data, err := json.Marshal(dstObj)
	if err != nil {
		return 0, err
	}
	if err := AtomicWrite(dst, data); err != nil {
		return 0, err
	}
	return changed, nil
}
