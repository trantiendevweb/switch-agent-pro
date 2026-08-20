package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Ba trạng thái phiên đo được (schema v9) và cái sổ phải giữ chúng qua nâng cấp.
//
// Bốn nhóm test dưới đây gắn vào bốn điều kiện của việc: suy đúng, thiếu dữ
// liệu thì không suy, nâng schema hai lần liên tiếp không hỏng, và phiên ghi
// bằng bản CŨ vẫn đọc lại được sau khi nâng.

const pidChacChanChet = 0x7FFFFFF0

// dungSoOPhienBan dựng một file state.db ở ĐÚNG schema v<n> bằng cách chạy tay n
// bước migration đầu.
//
// Cần tự dựng vì OpenAt luôn kéo lên phiên bản mới nhất — không có cách nào
// khác để có trong tay một file "của bản sagent cũ" mà kiểm việc nâng cấp.
func dungSoOPhienBan(t *testing.T, path string, n int) {
	t.Helper()
	if n > len(migrations) {
		t.Fatalf("chỉ có %d bước migration, xin %d", len(migrations), n)
	}
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
			t.Fatalf("migration v%d: %v", i+1, err)
		}
	}
	if _, err := db.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('schema_version',?)`,
		strconv.Itoa(n)); err != nil {
		t.Fatal(err)
	}
}

func phienBan(t *testing.T, db *DB) int {
	t.Helper()
	v, err := db.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// (c) NÂNG SCHEMA HAI LẦN LIÊN TIẾP KHÔNG HỎNG.
//
// Hai bước nâng nối nhau là ca thật: người dùng bỏ qua một bản, cập nhật một
// phát từ v7 lên v9. Mỗi bước nâng còn CHỤP ẢNH file trước khi đổi (xem
// migrate), nên đây cũng là chỗ kiểm hai lần chụp không giẫm lên nhau.
func TestNangSchemaHaiLanLienTiepKhongHong(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "state.db")
	// v7: trước cả sổ đăng ký (v8) lẫn trạng thái phiên chi tiết (v9).
	dungSoOPhienBan(t, p, 7)

	// Lần nâng thứ nhất: v7 → v9 (hai bước migration trong một lần mở).
	db, err := OpenAt(p)
	if err != nil {
		t.Fatalf("nâng v7 lên mới nhất hỏng: %v", err)
	}
	if got := phienBan(t, db); got != len(migrations) {
		t.Fatalf("sau lần mở đầu schema = v%d, muốn v%d", got, len(migrations))
	}
	id, err := db.AddSession(Session{Provider: "claude", Account: "sau-nang", PID: os.Getpid(), Dir: "d"})
	if err != nil {
		t.Fatalf("sổ vừa nâng mà không ghi được phiên: %v", err)
	}
	db.Close()

	// Lần mở thứ hai: đã ở phiên bản mới nhất, không được đụng gì thêm.
	db2, err := OpenAt(p)
	if err != nil {
		t.Fatalf("mở lại sổ đã nâng hỏng: %v", err)
	}
	defer db2.Close()
	if got := phienBan(t, db2); got != len(migrations) {
		t.Fatalf("lần mở thứ hai schema = v%d, muốn v%d", got, len(migrations))
	}
	if err := db2.SetStateChiTiet(id, StateHanMuc, "hết hạn mức", 1755700000); err != nil {
		t.Fatalf("cột v9 không dùng được sau hai lần mở: %v", err)
	}
	hong, err := db2.PhienChet(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hong) != 1 || hong[0].State != StateHanMuc || hong[0].HanMucDenLai != 1755700000 {
		t.Fatalf("đọc lại sau hai lần nâng ra %+v", hong)
	}

	// Nâng có sao lưu, và mỗi lần nâng một file riêng — chồng file thì bản sao
	// của bước trước bị bước sau đè, tức lưới an toàn chỉ còn một mắt.
	bak, _ := filepath.Glob(filepath.Join(dir, "*.bak-v*"))
	if len(bak) == 0 {
		bak, _ = filepath.Glob(filepath.Join(dir, "*v7*"))
	}
	if len(bak) == 0 {
		t.Error("nâng schema trên file CÓ DỮ LIỆU mà không để lại bản sao lưu nào")
	}
}

// (d) PHIÊN GHI BẰNG BẢN CŨ ĐỌC LẠI ĐƯỢC SAU KHI NÂNG.
//
// Ca hỏng thật mà test này chặn: v9 thêm cột NOT NULL. Với dòng đã có sẵn,
// SQLite lấp bằng DEFAULT — nhưng chỉ khi migration KHAI default. Quên chữ
// `DEFAULT` thì `ALTER TABLE` gãy ngay, còn quét ra rồi Scan vào string mà gặp
// NULL thì gãy lúc đọc. Cả hai đều là "mở sổ cũ lên là không dùng được nữa".
func TestPhienGhiBangBanCuDocLaiDuocSauNangCap(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.db")
	dungSoOPhienBan(t, p, 8) // v8: sessions chưa có state_ly_do / han_muc_den_lai

	// Ghi thẳng bằng SQL của bản CŨ — cố ý không dùng AddSession, vì hàm đó là
	// mã của bản MỚI và sẽ tự biết mọi cột mới.
	cu, err := sql.Open("sqlite", "file:"+p)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cu.Exec(`INSERT INTO sessions(provider,account,clone,dir,pid,log,worktree,started,state)
		VALUES('claude','cu',0,'d',?,'','','2026-08-19T10:00:00Z',?)`, pidChacChanChet, StateLost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cu.Exec(`INSERT INTO sessions(provider,account,clone,dir,pid,log,worktree,started,state)
		VALUES('claude','cu-chay',0,'d',?,'','','2026-08-19T10:00:00Z',?)`, os.Getpid(), StateRunning); err != nil {
		t.Fatal(err)
	}
	cu.Close()

	db, err := OpenAt(p)
	if err != nil {
		t.Fatalf("mở sổ v8 có dữ liệu hỏng: %v", err)
	}
	defer db.Close()

	// Phiên `lost` cũ: đọc được, và hai cột mới rỗng đúng nghĩa "không đo được".
	hong, err := db.PhienChet(10)
	if err != nil {
		t.Fatalf("đọc phiên cũ sau nâng cấp hỏng: %v", err)
	}
	if len(hong) != 1 {
		t.Fatalf("muốn 1 phiên cũ, được %d", len(hong))
	}
	if hong[0].Account != "cu" || hong[0].State != StateLost {
		t.Fatalf("phiên cũ đọc ra sai: %+v", hong[0])
	}
	if hong[0].StateLyDo != "" || hong[0].HanMucDenLai != 0 {
		t.Fatalf("phiên ghi trước v9 phải KHÔNG có lý do đo được, đang là (%q,%d) — "+
			"bịa dữ liệu cho dòng cũ còn tệ hơn để trống",
			hong[0].StateLyDo, hong[0].HanMucDenLai)
	}
	// Phiên `running` cũ cũng phải đọc được qua đường Running().
	chay, err := db.Running()
	if err != nil {
		t.Fatal(err)
	}
	if len(chay) != 1 || chay[0].Account != "cu-chay" {
		t.Fatalf("phiên running ghi bằng bản cũ đọc ra %+v", chay)
	}
	// Và đọc lại được cả bằng Lost() — đường mà `sagent quet` dùng.
	lost, err := db.Lost()
	if err != nil || len(lost) != 1 {
		t.Fatalf("Lost() sau nâng cấp: %+v, %v", lost, err)
	}
}

// (a) tại tầng sổ: bộ phân loại cắm vào PHẢI được dùng ở đúng chỗ phát hiện
// phiên chết, và trạng thái + lý do + mốc hạn mức phải nằm lại trên đĩa.
//
// Gỡ dòng gọi d.phanLoai trong Running() ra là test này đỏ.
func TestRunningDungBoPhanLoaiDaCam(t *testing.T) {
	db := open(t)
	goi := 0
	db.DungPhanLoaiChet(func(s Session) (string, string, int64) {
		goi++
		if s.Account != "chet" {
			t.Errorf("bộ phân loại bị gọi cho phiên %q — chỉ phiên CHẾT mới được hỏi", s.Account)
		}
		if s.Provider != "claude" || s.PID != pidChacChanChet {
			t.Errorf("bộ phân loại nhận phiên thiếu thông tin: %+v", s)
		}
		return StateHanMuc, "hết hạn mức, chờ được cấp lại", 1755700000
	})
	if _, err := db.AddSession(Session{Provider: "claude", Account: "song", PID: os.Getpid(), Dir: "d"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.AddSession(Session{Provider: "claude", Account: "chet", PID: pidChacChanChet, Dir: "d"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Running(); err != nil {
		t.Fatal(err)
	}
	if goi != 1 {
		t.Fatalf("bộ phân loại được gọi %d lần, muốn đúng 1", goi)
	}
	hong, err := db.PhienChet(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hong) != 1 {
		t.Fatalf("muốn 1 phiên chết, được %d", len(hong))
	}
	if hong[0].State != StateHanMuc {
		t.Errorf("state = %q, muốn %q", hong[0].State, StateHanMuc)
	}
	if hong[0].StateLyDo != "hết hạn mức, chờ được cấp lại" {
		t.Errorf("lý do = %q", hong[0].StateLyDo)
	}
	if hong[0].HanMucDenLai != 1755700000 {
		t.Errorf("mốc cấp lại = %d", hong[0].HanMucDenLai)
	}
}

// (b) tại tầng sổ: bộ phân loại nói KHÔNG BIẾT ("" ) thì phiên ở lại `lost`,
// KHÔNG được nhận một trạng thái nào khác và KHÔNG được có lý do bịa.
//
// Cùng test cho cả trường hợp không cắm bộ phân loại nào (CLI cũ, gói khác).
func TestKhongDoDuocThiPhienOLaiLost(t *testing.T) {
	for _, ca := range []struct {
		ten  string
		cam  bool
		tra  string
		lyDo string
	}{
		{ten: "không cắm bộ phân loại", cam: false},
		{ten: "cắm nhưng không kết luận được", cam: true, tra: "", lyDo: "đáng lẽ không được ghi"},
	} {
		t.Run(ca.ten, func(t *testing.T) {
			db := open(t)
			if ca.cam {
				db.DungPhanLoaiChet(func(Session) (string, string, int64) {
					return ca.tra, ca.lyDo, 1755700000
				})
			}
			if _, err := db.AddSession(Session{Provider: "codex", Account: "chet",
				PID: pidChacChanChet, Dir: "d"}); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Running(); err != nil {
				t.Fatal(err)
			}
			hong, err := db.PhienChet(10)
			if err != nil {
				t.Fatal(err)
			}
			if len(hong) != 1 {
				t.Fatalf("muốn 1 phiên chết, được %d", len(hong))
			}
			if hong[0].State != StateLost {
				t.Fatalf("không đo được mà state = %q — phải là %q", hong[0].State, StateLost)
			}
			if hong[0].StateLyDo != "" || hong[0].HanMucDenLai != 0 {
				t.Fatalf("không kết luận được mà vẫn ghi (%q,%d) xuống sổ",
					hong[0].StateLyDo, hong[0].HanMucDenLai)
			}
		})
	}
}

// `sagent quet` tìm tiến trình mồ côi qua Lost(). Thêm ba trạng thái mới mà
// quên nới câu SQL đó thì phiên hết-hạn-mức tàng hình khỏi lệnh quét, và đám
// tiến trình con của nó tiêu hạn mức mà không ai nhìn thấy.
func TestLostLayCaBaTrangThaiMoi(t *testing.T) {
	db := open(t)
	for _, st := range TuKetThuc() {
		id, err := db.AddSession(Session{Provider: "claude", Account: "a" + st, PID: os.Getpid(), Dir: "d"})
		if err != nil {
			t.Fatal(err)
		}
		if err := db.SetStateChiTiet(id, st, "vì "+st, 0); err != nil {
			t.Fatal(err)
		}
	}
	// Một phiên dừng ĐÚNG CÁCH: không được lẫn vào.
	id, _ := db.AddSession(Session{Provider: "claude", Account: "dung-han-hoi", PID: os.Getpid(), Dir: "d"})
	if err := db.SetState(id, StateStopped); err != nil {
		t.Fatal(err)
	}

	lost, err := db.Lost()
	if err != nil {
		t.Fatal(err)
	}
	if len(lost) != len(TuKetThuc()) {
		t.Fatalf("Lost() trả %d phiên, muốn %d (%v)", len(lost), len(TuKetThuc()), TuKetThuc())
	}
	for _, s := range lost {
		if !LaTuKetThuc(s.State) {
			t.Errorf("Lost() trả về phiên %q — không phải chết bất thường", s.State)
		}
	}
}

// SetState (đổi trạng thái trần) phải XOÁ lý do cũ. Giữ lại thì một phiên
// `stopped` vẫn mang câu "hết hạn mức" của kiếp trước, và bảng nói dối.
func TestSetStateXoaLyDoCu(t *testing.T) {
	db := open(t)
	id, _ := db.AddSession(Session{Provider: "claude", Account: "a", PID: os.Getpid(), Dir: "d"})
	if err := db.SetStateChiTiet(id, StateHanMuc, "hết hạn mức", 1755700000); err != nil {
		t.Fatal(err)
	}
	if err := db.SetState(id, StateStopped); err != nil {
		t.Fatal(err)
	}
	var ly string
	var han int64
	if err := db.db.QueryRow(`SELECT state_ly_do, han_muc_den_lai FROM sessions WHERE id=?`, id).
		Scan(&ly, &han); err != nil {
		t.Fatal(err)
	}
	if ly != "" || han != 0 {
		t.Fatalf("đổi sang %q mà còn giữ (%q,%d) của trạng thái cũ", StateStopped, ly, han)
	}
}

// PhienChet là đường cho MẶT NGƯỜI DÙNG: mới nhất trước, và có trần.
func TestPhienChetMoiNhatTruocVaCoTran(t *testing.T) {
	db := open(t)
	for i := 0; i < 5; i++ {
		id, _ := db.AddSession(Session{Provider: "claude", Account: "a" + itoa(i), PID: os.Getpid(), Dir: "d"})
		if err := db.SetStateChiTiet(id, StateHong, "lỗi", 0); err != nil {
			t.Fatal(err)
		}
	}
	list, err := db.PhienChet(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("xin 2, được %d — trần không có tác dụng thì bảng dài dần theo tháng", len(list))
	}
	if list[0].ID < list[1].ID {
		t.Fatalf("PhienChet phải trả MỚI NHẤT trước, được %d rồi %d", list[0].ID, list[1].ID)
	}
}
