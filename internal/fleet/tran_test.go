package fleet

import (
	"strings"
	"testing"
)

func tranMau() Tran {
	return Tran{Chung: 4, HarnessMacDinh: 3, ProviderMacDinh: 3, HoSoMacDinh: 2}
}

func dang(cap ...string) []Phien {
	out := make([]Phien, 0, len(cap))
	for _, s := range cap {
		i := strings.IndexByte(s, ':')
		out = append(out, Phien{Provider: s[:i], Account: s[i+1:]})
	}
	return out
}

// Bốn trần CỘNG DỒN: mức chật nhất quyết định, không phải mức khai sau cùng.
func TestBonTranCongDonMucChatNhatThang(t *testing.T) {
	cases := []struct {
		ten  string
		tr   Tran
		chay []string
		xin  string
		muon int
		cap  int
		loai string // loại của mức chật nhất, "" = không mức nào cắt
	}{
		{
			ten: "trống trơn thì hồ sơ cắt trước", tr: tranMau(),
			xin: "claude:tns", muon: 4, cap: 2, loai: "hồ sơ",
		},
		{
			ten: "provider chật hơn hồ sơ", tr: Tran{Chung: 9, HarnessMacDinh: 9, ProviderMacDinh: 2, HoSoMacDinh: 9},
			chay: []string{"claude:phu"}, xin: "claude:tns", muon: 4, cap: 1, loai: "provider",
		},
		{
			ten: "trần chung là trần ngoài cùng", tr: Tran{Chung: 2, HarnessMacDinh: 9, ProviderMacDinh: 9, HoSoMacDinh: 9},
			chay: []string{"codex:a", "grok:b"}, xin: "claude:tns", muon: 3, cap: 0, loai: "chung",
		},
		{
			ten: "tắt hết thì cấp đủ", tr: Tran{},
			chay: []string{"claude:tns", "claude:tns", "claude:tns"}, xin: "claude:tns", muon: 5, cap: 5, loai: "",
		},
		{
			ten: "khai riêng đè số mặc định", tr: Tran{Chung: 9, HarnessMacDinh: 9, ProviderMacDinh: 9, HoSoMacDinh: 9,
				HoSo: map[string]int{"claude:tns": 1}},
			xin: "claude:tns", muon: 3, cap: 1, loai: "hồ sơ",
		},
		{
			ten: "tài khoản khác cùng provider KHÔNG tính vào trần hồ sơ",
			tr:  Tran{Chung: 9, HarnessMacDinh: 9, ProviderMacDinh: 9, HoSoMacDinh: 2},
			// ba phiên claude:phu không đụng gì tới trần của claude:tns
			chay: []string{"claude:phu", "claude:phu", "claude:phu"},
			xin:  "claude:tns", muon: 2, cap: 2, loai: "",
		},
	}
	for _, c := range cases {
		t.Run(c.ten, func(t *testing.T) {
			i := strings.IndexByte(c.xin, ':')
			k := XetTran(c.tr, dang(c.chay...), Phien{c.xin[:i], c.xin[i+1:]}, c.muon)
			if k.Cap != c.cap {
				t.Errorf("cấp %d, mong %d — %s", k.Cap, c.cap, k.Bang())
			}
			chat := k.Chat()
			if c.loai == "" {
				if len(chat) > 0 {
					t.Errorf("không mức nào được cắt, mà có %d: %s", len(chat), k.Bang())
				}
				return
			}
			if len(chat) == 0 {
				t.Fatalf("mong mức %q cắt, mà không mức nào cắt — %s", c.loai, k.Bang())
			}
			if chat[0].Loai != c.loai {
				t.Errorf("mức chật nhất là %q, mong %q — %s", chat[0].Loai, c.loai, k.Bang())
			}
		})
	}
}

// Không khai `thuoc_harness` thì harness của một provider chính là tên provider.
// Khai rồi thì hai provider gộp vào MỘT trần máy — đó là điểm duy nhất khiến
// trần harness khác trần provider, nên nó phải chạy đúng.
func TestGopProviderVaoMotHarness(t *testing.T) {
	tr := Tran{Chung: 9, HarnessMacDinh: 2, ProviderMacDinh: 9, HoSoMacDinh: 9,
		ThuocHarness: map[string]string{"codex": "node", "grok": "node"}}

	// Chưa gộp: một phiên grok không đụng tới trần harness của claude.
	k := XetTran(tr, dang("grok:a"), Phien{"claude", "tns"}, 2)
	if k.Cap != 2 {
		t.Errorf("grok không cùng harness với claude mà vẫn cắt: %s", k.Bang())
	}
	// Đã gộp: phiên grok ăn vào trần "node", nên codex chỉ còn 1 chỗ.
	k = XetTran(tr, dang("grok:a"), Phien{"codex", "chinh"}, 2)
	if k.Cap != 1 {
		t.Fatalf("gộp harness không có tác dụng: %s", k.Bang())
	}
	if k.Chat()[0].Ten != "node" {
		t.Errorf("mức cắt phải mang tên harness gộp, được %q", k.Chat()[0].Ten)
	}
}

// Câu người vận hành đọc lúc hai giờ sáng. Không có bài này thì thông báo dễ
// trôi dần về "đã hạ xuống N" — đúng cú pháp, vô dụng.
func TestCauThongBaoNoiDuTranNaoVaConBaoNhieu(t *testing.T) {
	tr := tranMau()
	tr.HoSo = map[string]int{"claude:tns": 1}

	// Ca còn chỗ nhưng ít hơn xin.
	k := XetTran(tr, nil, Phien{"claude", "tns"}, 3)
	cat := k.LoiCatBot()
	for _, phai := range []string{"cắt 3 phiên xuống 1", "hồ sơ claude:tns", "trần 1, đang chạy 0",
		"chung: trần 4", "harness claude: trần 3", "provider claude: trần 3",
		`policy.tran.ho_so."claude:tns"`, "tài khoản khác"} {
		if !strings.Contains(cat, phai) {
			t.Errorf("câu cắt bớt thiếu %q:\n%s", phai, cat)
		}
	}

	// Ca hết sạch chỗ.
	k = XetTran(tr, dang("claude:tns"), Phien{"claude", "tns"}, 2)
	if k.Cap != 0 {
		t.Fatalf("mong hết chỗ, cấp %d", k.Cap)
	}
	het := k.LoiHetCho().Error()
	for _, phai := range []string{"hết chỗ cho claude:tns", "hồ sơ claude:tns đã đầy (trần 1, đang chạy 1)",
		"chung: trần 4, đang chạy 1", `policy.tran.ho_so."claude:tns" = <số>`,
		"sagent ds", "sagent stop"} {
		if !strings.Contains(het, phai) {
			t.Errorf("câu từ chối thiếu %q:\n%s", phai, het)
		}
	}
	// Không được đổ tội nhầm cho trần còn chỗ.
	if strings.Contains(het, "chung đã đầy") || strings.Contains(het, "harness claude đã đầy") {
		t.Errorf("gọi tên nhầm trần chặn:\n%s", het)
	}
}

// Khoá TOML in trong lời khuyên phải CHÉP VÀO ĐƯỢC. Địa chỉ hồ sơ có dấu hai
// chấm nên bắt buộc bọc nháy; in trần ra thì người dùng chép vào file là lỗi
// cú pháp, mà lời khuyên hỏng thì tệ hơn không khuyên.
func TestKhoaTomlChepVaoDuoc(t *testing.T) {
	if got := khoaBang("policy.tran.ho_so", "claude:tns"); got != `policy.tran.ho_so."claude:tns"` {
		t.Errorf("địa chỉ hồ sơ không được bọc nháy: %s", got)
	}
	if got := khoaBang("policy.tran.harness", "claude"); got != "policy.tran.harness.claude" {
		t.Errorf("tên thường không cần nháy: %s", got)
	}
}

// Trần bị hạ xuống dưới số đang chạy (sửa file lúc đang có phiên) là chuyện có
// thật. Số chỗ còn lại phải là 0, không phải số âm — số âm lọt xuống dưới thì
// `Cap` thành âm và mọi phép so sánh sau đó sai.
func TestTranHaXuongDuoiSoDangChayThiConLaKhong(t *testing.T) {
	tr := Tran{Chung: 9, HarnessMacDinh: 9, ProviderMacDinh: 9, HoSoMacDinh: 1}
	k := XetTran(tr, dang("claude:tns", "claude:tns", "claude:tns"), Phien{"claude", "tns"}, 1)
	if k.Cap != 0 {
		t.Fatalf("cấp %d, mong 0", k.Cap)
	}
	for _, m := range k.Mucs {
		if m.Con() < -1 {
			t.Errorf("%s trả số chỗ âm: %d", m.Nhan(), m.Con())
		}
	}
}
