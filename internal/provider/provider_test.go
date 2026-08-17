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
		if len(ad.PrivateFiles()) == 0 {
			t.Fatalf("%s: PrivateFiles() rỗng — token sẽ bị nối link dùng chung", name)
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
