package provider

import "testing"

// Mọi adapter phải khai báo đủ primitive — thiếu một cái là lõi sẽ đoán bừa.
func TestAdapterKhaiBaoDu(t *testing.T) {
	for _, name := range Names() {
		ad, _ := Get(name)
		if ad.Name() == "" {
			t.Fatalf("%s: thiếu Name()", name)
		}
		if ad.EnvVar() == "" {
			t.Fatalf("%s: thiếu EnvVar() — không có biến thì không tách được thư mục", name)
		}
		// PrivateFiles rỗng chỉ được phép khi adapter TỰ KHAI là không tách được
		// tài khoản. Ràng buộc thật nằm ở đây: đã hứa tách được thì phải nói rõ
		// file nào là riêng, không thì LinkShared sẽ nối link cả token.
		//
		// Antigravity khai false vì token của nó nằm trong Windows Credential
		// Manager chứ không phải file — rỗng là mô tả ĐÚNG, không phải thiếu sót.
		if len(ad.PrivateFiles()) == 0 && ad.TachDuocTaiKhoan() {
			t.Fatalf("%s: hứa tách được tài khoản nhưng PrivateFiles() rỗng — "+
				"token sẽ bị nối link dùng chung", name)
		}
		if ad.BaseDir() == "" {
			t.Fatalf("%s: thiếu BaseDir()", name)
		}
	}
}

// HeadlessArgs từng bị hardcode "-p" trong lõi, khiến fleet chạy sai với Codex
// mà không ai biết. Test này giữ cho mỗi provider tự khai kiểu chạy của mình.
func TestMoiProviderCoCachChayHeadlessRieng(t *testing.T) {
	seen := map[string]string{}
	for _, name := range Names() {
		ad, _ := Get(name)
		args := ad.HeadlessArgs("XIN-CHAO")
		if len(args) == 0 {
			t.Fatalf("%s: HeadlessArgs rỗng — fleet sẽ chạy CLI mà không có prompt", name)
		}
		found := false
		for _, a := range args {
			if a == "XIN-CHAO" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: HeadlessArgs không mang theo prompt: %v", name, args)
		}
		seen[name] = args[0]
	}
	// Claude dùng cờ, Codex dùng lệnh con — nếu ai đó "gộp cho gọn" thì test đỏ.
	if seen["claude"] == seen["codex"] {
		t.Fatalf("claude và codex phải khác cách chạy headless, cả hai đang là %q", seen["claude"])
	}
}

// Provider không tách được tài khoản thì `fleet --copies N` phải TỪ CHỐI, không
// được bật lên rồi để N tiến trình giành nhau một danh tính.
//
// Đây là bài học đắt: hai client Claude giành một device slot 1866 lần trong 18
// tiếng rồi làm rơi phiên remote (xem docs/DO-LUONG.md). Cùng một hình dạng lỗi.
func TestKhaiKhongTachDuocThiPhaiNoiRa(t *testing.T) {
	var coCaiKhongTach bool
	for _, name := range Names() {
		ad, _ := Get(name)
		if !ad.TachDuocTaiKhoan() {
			coCaiKhongTach = true
			// Bộ đo của nó phải NÓI RA điều đó, chứ không giấu trong tài liệu.
			var noiRa bool
			for _, c := range ad.Verify() {
				if !c.OK && len(c.Detail) > 0 {
					noiRa = true
				}
			}
			if !noiRa {
				t.Errorf("%s không tách được tài khoản mà Verify() không có ô nào báo điều đó", name)
			}
		}
	}
	if !coCaiKhongTach {
		t.Skip("chưa có provider nào khai không tách được — không có gì để đo")
	}
}
