package api

import (
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// Báo cáo năng lực phải đi ra được qua hợp đồng, không chỉ nằm trong gói
// provider. Đây là điều kiện để CLI, dashboard và 3D cùng đọc MỘT nguồn — luật
// ngang quyền (MASTER-PLAN mục 2c).
func TestNangLucCoTrongHopDong(t *testing.T) {
	var co bool
	for _, a := range Actions {
		if a == "provider.nang-luc" {
			co = true
		}
	}
	if !co {
		t.Fatal("action provider.nang-luc không có trong Actions — mặt web và 3D sẽ không " +
			"bao giờ biết tính năng này tồn tại")
	}
}

// Không truyền tên thì trả về MỌI provider, mỗi provider đủ MỌI năng lực.
//
// `&API{}` là đủ: hàm này chỉ đọc adapter đã đăng ký, không chạm store — cố ý,
// vì bảng năng lực phải trả lời được ngay cả khi sổ trạng thái hỏng.
func TestNangLucTraDuMoiProviderVaMoiMuc(t *testing.T) {
	ds, err := (&API{}).NangLuc("")
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != len(provider.Names()) {
		t.Fatalf("có %d provider đăng ký nhưng báo cáo chỉ có %d", len(provider.Names()), len(ds))
	}
	for _, p := range ds {
		if len(p.Lech) > 0 {
			t.Errorf("%s: bảng năng lực chọi với hành vi thật: %v", p.Provider, p.Lech)
		}
		co := map[string]bool{}
		for _, m := range p.Muc {
			co[m.Khoa] = true
			if m.BangChung == "" {
				t.Errorf("%s: %q không nói đo ở đâu", p.Provider, m.Khoa)
			}
		}
		for _, m := range provider.MoiNangLuc {
			if !co[m.Khoa] {
				t.Errorf("%s: báo cáo thiếu %q", p.Provider, m.Khoa)
			}
		}
	}
}

// Thứ tự phải ỔN ĐỊNH giữa các lần gọi. Trả thẳng thứ tự của map thì bảng trên
// dashboard nhảy loạn mỗi lần làm mới, và người đọc mất chỗ đứng.
func TestNangLucGiuThuTuOnDinh(t *testing.T) {
	mot, err := (&API{}).NangLuc("")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		hai, err := (&API{}).NangLuc("")
		if err != nil {
			t.Fatal(err)
		}
		for j := range mot {
			if mot[j].Provider != hai[j].Provider {
				t.Fatalf("thứ tự đổi giữa hai lần gọi: %q rồi %q", mot[j].Provider, hai[j].Provider)
			}
		}
	}
}

// Lọc theo tên thì chỉ trả đúng một provider; tên lạ thì BÁO LỖI chứ không trả
// danh sách rỗng — rỗng đọc như "provider này không có năng lực nào".
func TestNangLucLocTheoTenVaTuChoiTenLa(t *testing.T) {
	ds, err := (&API{}).NangLuc("claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 || ds[0].Provider != "claude" {
		t.Fatalf("lọc theo tên sai: %+v", ds)
	}
	if _, err := (&API{}).NangLuc("khong-co-provider-nay"); err == nil {
		t.Fatal("tên provider lạ phải báo lỗi, không được im lặng trả rỗng")
	}
}

// Ba trạng thái phải đi RA TỚI hợp đồng, không bị dẹp thành hai lúc đóng gói.
//
// Đây là điều dễ mất nhất khi đi qua một lớp DTO: ai đó thấy "khong-lam-duoc"
// và "chua-do" đều không có cờ nào, rồi gộp cả hai thành `false`. Lúc đó Grok —
// provider KHÔNG có rào duyệt quyền nào, chuyện an ninh — trông y hệt Cursor
// chưa ai đo.
func TestBaTrangThaiDiRaToiHopDong(t *testing.T) {
	ds, err := (&API{}).NangLuc("")
	if err != nil {
		t.Fatal(err)
	}
	dem := map[provider.TrangThaiNangLuc]int{}
	for _, p := range ds {
		for _, m := range p.Muc {
			dem[m.TrangThai]++
		}
	}
	for _, tt := range []provider.TrangThaiNangLuc{
		provider.LamDuoc, provider.KhongLamDuoc, provider.ChuaDo,
	} {
		if dem[tt] == 0 {
			t.Errorf("hợp đồng không mang ra trạng thái %q nào", tt)
		}
	}
}
