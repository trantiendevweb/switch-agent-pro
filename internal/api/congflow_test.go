package api

import (
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/fleet"
)

// TẦNG CUỐI — bài này chỉ hỏi đúng một câu: bộ chạy flow mà `sagent flow run`
// thật sự dùng CÓ được cắm cổng trần đồng thời không?
//
// Vì sao một câu hỏi tầm thường như vậy lại đáng một bài kiểm riêng: hôm nay dự
// án dính NĂM lần cùng một hình dạng lỗi — thứ gì đó có đủ ở mọi tầng trừ tầng
// người dùng thật sự chạm vào (`plugin.list`, `route.kiem`, nút Duyệt/Từ chối,
// 5 lệnh vắng mặt trong `help`, và chính bốn trần này ở cửa flow). Mọi bài kiểm
// khác vẫn xanh trong từng lần đó, vì chúng hỏi từng mảnh chứ không hỏi cái nối.
//
// internal/flow/tran_flow_test.go chứng minh CỔNG chặn đúng khi đã cắm. Bài này
// chứng minh nó ĐƯỢC cắm. Thiếu một trong hai là lại đúng cái bẫy cũ.
func TestRunnerCuaFlowDaCamCongTranDongThoi(t *testing.T) {
	a := moAPI(t)
	r := a.runner(Addr{Provider: "claude", Account: "tns"}, t.TempDir())
	if r.Cong == nil {
		t.Fatal("BỘ CHẠY FLOW KHÔNG CÓ CỔNG TRẦN: bốn trần đồng thời chỉ canh cửa `fleet.start`, " +
			"còn `sagent flow run` vẫn chạy bằng mỗi `max_parallel_sessions`")
	}
	if r.MaxParallel != a.cfg.Policy.MaxParallelSessions {
		t.Fatalf("bề rộng đợt phải giữ nguyên chính sách dự án: %d vs %d",
			r.MaxParallel, a.cfg.Policy.MaxParallelSessions)
	}
}

// Cổng phải đọc sổ phiên qua ĐÚNG một đường với cửa `fleet.start`.
//
// Hai bản đọc song song sẽ lệch nhau trong im lặng ở đúng chỗ khó thấy nhất —
// một bên lọc phiên mồ côi, bên kia không — và lúc đó không ai biết con số nào
// đúng. Bài này neo cái đường đó lại.
func TestCongVaFleetStartDocCungMotSoPhien(t *testing.T) {
	a := moAPI(t)
	themPhienSong(t, a, "claude", "tns")
	themPhienSong(t, a, "codex", "phu")

	dang := a.PhienDangChay()
	if len(dang) != 2 {
		t.Fatalf("PhienDangChay đọc được %d phiên, sổ có 2", len(dang))
	}
	// Cùng con số đó phải là con số `FleetStart` xét trần trên.
	k, ok := a.xetTran(Addr{Provider: "claude", Account: "tns"}, 1)
	if !ok {
		t.Fatal("dự án có trần mặc định mà xetTran báo không áp trần nào")
	}
	for _, m := range k.Mucs {
		if m.Loai == "chung" && m.Dang != 2 {
			t.Fatalf("trần chung đếm %d phiên, PhienDangChay trả %d — hai đường đọc đã lệch",
				m.Dang, len(dang))
		}
	}
}

// ParseAddr và fleet.PhienTu phải tách địa chỉ y hệt nhau.
//
// Cổng đếm theo địa chỉ do fleet.PhienTu tách ra, còn phiên thật chạy theo địa
// chỉ do ParseAddr tách ra. Lệch một ca là trần đi canh một tài khoản không ai
// chạy — và nó hỏng trong im lặng: trần vẫn "hoạt động", chỉ là vô dụng.
func TestParseAddrVaPhienTuKhongLech(t *testing.T) {
	for _, s := range []string{"claude:tns", "tns", "codex:phu", "", "claude:", ":tns", "a:b:c"} {
		ad := ParseAddr(s)
		p := fleet.PhienTu(s)
		if ad.Provider != p.Provider || ad.Account != p.Account {
			t.Errorf("lệch ở %q: ParseAddr -> %s/%s, fleet.PhienTu -> %s/%s",
				s, ad.Provider, ad.Account, p.Provider, p.Account)
		}
	}
}
