package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// Khối `[policy.tran]` phải ĐỌC RA ĐƯỢC từ file thật, cả ba số mặc định lẫn ba
// bảng khai riêng. Sai một thẻ `toml:` thì khoá vẫn decode "thành công" về 0 —
// tức TẮT trần đó — và không ai báo gì.
func TestDocKhoiTranTuFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	duAn := filepath.Join(tmp, "duan")
	write(t, filepath.Join(duAn, ProjectDirName, "project.toml"), `
name = "x"
[policy]
max_parallel_sessions = 6
[policy.tran]
harness_mac_dinh  = 5
provider_mac_dinh = 4
ho_so_mac_dinh    = 3
[policy.tran.harness]
node = 2
[policy.tran.provider]
codex = 2
[policy.tran.ho_so]
"claude:tns" = 1
[policy.tran.thuoc_harness]
codex = "node"
grok  = "node"
`)
	c, err := Load(duAn)
	if err != nil {
		t.Fatal(err)
	}
	tr := c.Policy.Tran
	if tr.HarnessMacDinh != 5 || tr.ProviderMacDinh != 4 || tr.HoSoMacDinh != 3 {
		t.Errorf("ba số mặc định đọc sai: %+v", tr)
	}
	if tr.Harness["node"] != 2 || tr.Provider["codex"] != 2 || tr.HoSo["claude:tns"] != 1 {
		t.Errorf("ba bảng khai riêng đọc sai: %+v", tr)
	}
	if tr.ThuocHarness["codex"] != "node" || tr.ThuocHarness["grok"] != "node" {
		t.Errorf("bảng thuoc_harness đọc sai: %+v", tr.ThuocHarness)
	}
}

// Không khai gì thì tính năng vẫn phải chạy — và chạy bằng bộ số ĐÃ NGĂN được
// bốn phiên dồn vào một tài khoản. Đây là yêu cầu "đừng bắt ai cấu hình mới
// dùng được", nên nó là một bài kiểm chứ không phải một lời hứa trong tài liệu.
func TestKhongKhaiGiVanCoTranMacDinh(t *testing.T) {
	c := Default()
	tr := c.Policy.Tran
	if tr.HoSoMacDinh <= 0 {
		t.Fatal("mặc định không có trần hồ sơ — bốn phiên vẫn dồn được vào một tài khoản")
	}
	if tr.HoSoMacDinh >= c.Policy.MaxParallelSessions {
		t.Errorf("trần hồ sơ mặc định (%d) không siết gì thêm so với trần chung (%d)",
			tr.HoSoMacDinh, c.Policy.MaxParallelSessions)
	}
	if tr.HarnessMacDinh <= 0 || tr.ProviderMacDinh <= 0 {
		t.Errorf("trần harness/provider mặc định bị tắt: %+v", tr)
	}
}

// Số âm phải kêu NGAY lúc đọc file. Để lọt thì `Con()` kẹp về 0 và lượt chạy bị
// TỪ CHỐI HẲN — người gõ `-1` với ý "bỏ giới hạn" nhận đúng điều ngược lại.
func TestTranAmBiTuChoiVaChiCachTat(t *testing.T) {
	for _, than := range []string{
		"[policy.tran]\nho_so_mac_dinh = -1\n",
		"[policy.tran]\nharness_mac_dinh = -2\n",
		"[policy.tran.provider]\ncodex = -1\n",
	} {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Setenv("USERPROFILE", tmp)
		duAn := filepath.Join(tmp, "duan")
		write(t, filepath.Join(duAn, ProjectDirName, "project.toml"), "name = \"x\"\n"+than)
		_, err := Load(duAn)
		if err == nil {
			t.Errorf("số âm lọt qua: %s", than)
			continue
		}
		// Câu lỗi phải chỉ cách TẮT, nếu không người dùng chỉ biết là "sai".
		if !strings.Contains(err.Error(), "khai 0") {
			t.Errorf("câu lỗi không chỉ cách tắt: %v", err)
		}
	}
}

// Khoá của `[policy.tran.ho_so]` phải là địa chỉ đầy đủ. Gõ thiếu thành "tns"
// thì bảng vẫn decode, trần im lặng không khớp phiên nào, và người dùng tưởng
// đã siết xong — đúng lớp hỏng-mà-không-báo mà dự án này lập ra để chống.
func TestKhoaHoSoThieuProviderThiKeu(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	duAn := filepath.Join(tmp, "duan")
	write(t, filepath.Join(duAn, ProjectDirName, "project.toml"),
		"name = \"x\"\n[policy.tran.ho_so]\ntns = 1\n")
	_, err := Load(duAn)
	if err == nil {
		t.Fatal(`khoá "tns" thiếu provider mà vẫn qua`)
	}
	if !strings.Contains(err.Error(), "claude:tns") {
		t.Errorf("câu lỗi không nêu dạng đúng: %v", err)
	}
}

// File mẫu của `sagent init` phải khai được khối này — mẫu mà không dùng nổi
// thì người mới không bao giờ biết tính năng tồn tại.
func TestFileMauKhaiDuocKhoiTran(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	duAn := filepath.Join(tmp, "duan")
	write(t, filepath.Join(duAn, ProjectDirName, "project.toml"), strings.Replace(Sample, "%s", "thu", 1))
	c, err := Load(duAn)
	if err != nil {
		t.Fatalf("file mẫu không đọc được: %v", err)
	}
	if c.Policy.Tran.HoSoMacDinh <= 0 {
		t.Errorf("file mẫu không khai trần hồ sơ: %+v", c.Policy.Tran)
	}
}
