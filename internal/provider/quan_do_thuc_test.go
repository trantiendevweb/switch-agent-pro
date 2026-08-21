package provider

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/trantiendevweb/switch-agent-pro/internal/nhatky"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// PHÉP ĐO THỰC cho ngưỡng TranLapLienTiep — ô C5 của docs/SO-NO-DO-LUONG.md.
//
// VÌ SAO BÀI NÀY TỒN TẠI. quan.go tự khai thẳng chỗ hổng của nó: ca quẩn thật
// duy nhất đo được là 399 lần liên tiếp, tức ngưỡng 10 chắc chắn KHÔNG BỎ SÓT
// ca thật. Mặt kia — ngưỡng có BẮT OAN một lượt chạy bình thường không — chưa
// ai đo, vì chưa từng đếm chuỗi lặp dài nhất trên một bản ghi lượt-chạy-bình-
// thường nào. Mọi bài trong quan_test.go đều chạy trên bản ghi DỰNG TAY: chúng
// chứng minh bộ đếm đếm đúng thứ nó được cho ăn, chúng không nói gì về việc
// agent thật lặp bao nhiêu lần khi đang làm việc tử tế.
//
// Bài này đọc BẢN GHI THẬT trên máy: các file nhật ký phiên, tìm qua cột `log`
// của bảng `sessions` trong state.db — tức chỉ lấy những lượt SỔ CÓ GHI, không
// phải file ai đó thả vào thư mục. Rồi tái dùng ĐÚNG bộ đếm sản phẩm (demQuan +
// chuKyTool, qua bộ đọc của từng provider) để lấy chuỗi lặp liên tiếp dài nhất
// mỗi phiên.
//
// KẾT LUẬN CỦA BÀI: chuỗi lặp dài nhất của MỌI phiên bình thường phải NHỎ HƠN
// TranLapLienTiep. Vượt là ngưỡng đang vu oan cho một lượt làm việc thật, và vu
// oan thì tệ hơn bỏ sót — người đọc mất niềm tin rồi tắt lá chắn đi.
//
// Bài KHÔNG dựng dữ liệu giả và KHÔNG gọi CLI nào: không có dữ liệu thì SKIP.
// Trên máy CI trắng nó im lặng; trên máy đã chạy fleet nó đo lại mỗi lần
// `go test`, nên ngày một lượt bình thường tiến sát ngưỡng là biết ngay.

// CoMauToiThieu là số lời gọi tool ÍT NHẤT phải bóc được trên toàn bộ sổ thì
// bài mới dám kết luận.
//
// Không có sàn này thì bài xanh cả trên một máy vừa chạy đúng một lượt hai lời
// gọi — xanh vì không có gì để bắt, chứ không phải vì ngưỡng an toàn. Xanh kiểu
// đó đi thẳng vào sổ nợ thành "đã đo", và đó là cách một ô nợ bị đóng sai.
const CoMauToiThieu = 100

// phienDo là một phiên trong sổ, kèm số đo (hoặc lý do không đo được).
type phienDo struct {
	id    int64
	addr  string
	prov  string
	state string
	duong string
	gop   int // số dòng sổ khác cùng trỏ vào file này (xem gomTheoFile)

	lenh     string // lời gọi bị lặp nhiều nhất
	soLan    int    // độ dài chuỗi lặp dài nhất
	soLoiGoi int    // tổng số lời gọi bóc được — cỡ mẫu của phiên
	docDuoc  bool
	vi       string // lý do KHÔNG ĐO ĐƯỢC, rỗng nếu đo được
}

func TestDoThucChuoiLapCuaLuotBinhThuong(t *testing.T) {
	ds, mat, gop := docPhienCoNhatKy(t)
	if len(ds) == 0 {
		t.Skip("máy này chưa có phiên nào còn nhật ký để đo — không dựng giả, không kết luận")
	}
	if mat > 0 {
		// Không phải lỗi bóc tách: nhật ký có ngân sách đĩa (nhatky.Don) nên
		// phiên đủ cũ mất file là chuyện bình thường. Nói ra số để người đọc
		// biết phép đo này phủ được bao nhiêu phần của sổ.
		t.Logf("bỏ qua %d dòng sổ vì file nhật ký không còn (đã bị dọn theo ngân sách đĩa)", mat)
	}
	if gop > 0 {
		t.Logf("gộp %d dòng sổ trùng file nhật ký (xem gomTheoFile) — mỗi file chỉ đo MỘT lần", gop)
	}

	var caoNhat *phienDo
	var soDoDuoc, tongLoiGoi int
	for i := range ds {
		p := &ds[i]
		doChuoiLap(p)
		if p.vi != "" {
			// Yêu cầu của ô C5: không đo được thì NÓI RA, đừng lặng lẽ bỏ qua —
			// im lặng ở đây đọc y hệt "đã đo và không sao".
			t.Logf("#%d %-18s %-12s KHÔNG ĐO ĐƯỢC: %s", p.id, p.addr, p.state, p.vi)
			continue
		}
		t.Logf("#%d %-18s %-12s %4d lời gọi, chuỗi lặp dài nhất = %d  (%s)",
			p.id, p.addr, p.state, p.soLoiGoi, p.soLan, dongDau(p.lenh))
		if p.soLoiGoi == 0 {
			// docDuoc=true mà cỡ mẫu 0 là tự mâu thuẫn — bộ đếm khai đã bóc được
			// lời gọi tool nhưng lại bảo không bóc được cái nào. Nếu để lọt, cả
			// bài này tụt xuống SKIP ("cỡ mẫu dưới sàn") và ô C5 lặng lẽ mất chỗ
			// dựa. Hỏng ở đây phải ĐỎ, không được thành im lặng.
			t.Errorf("#%d: bóc được lời gọi tool mà cỡ mẫu = 0 — demQuan.soLoiGoi hỏng", p.id)
		}
		if !laBinhThuong(p.state) {
			// `blocked` = mọi tool bị từ chối quyền. ketqua.go/Hong() đã ghi:
			// agent bị chặn quyền gọi lại đúng một lệnh nhiều lần, nhìn từ bộ
			// đếm thì y hệt chạy quẩn. Nó KHÔNG phải mẫu đối chứng cho câu
			// "ngưỡng có bắt oan lượt bình thường không".
			t.Logf("    ^ không tính vào kết luận: trạng thái %q không phải lượt chạy bình thường", p.state)
			continue
		}
		soDoDuoc++
		tongLoiGoi += p.soLoiGoi
		if caoNhat == nil || p.soLan > caoNhat.soLan {
			caoNhat = p
		}
	}

	if caoNhat == nil {
		t.Skipf("KHÔNG ĐO ĐƯỢC: không phiên BÌNH THƯỜNG nào trong %d file bóc được lời gọi tool kèm tham số",
			len(ds))
	}
	t.Logf("SỐ ĐO: %d phiên bình thường, %d lời gọi tool bóc được; chuỗi lặp liên tiếp dài nhất "+
		"trên toàn bộ = %d (phiên #%d %s, lệnh %s). Ngưỡng đang đặt TranLapLienTiep = %d.",
		soDoDuoc, tongLoiGoi, caoNhat.soLan, caoNhat.id, caoNhat.addr,
		dongDau(caoNhat.lenh), TranLapLienTiep)

	if tongLoiGoi < CoMauToiThieu {
		t.Skipf("KHÔNG ĐO ĐƯỢC: chỉ bóc được %d lời gọi tool trên toàn sổ, dưới sàn %d — "+
			"cỡ mẫu này không đủ để nói ngưỡng %d có bắt oan hay không",
			tongLoiGoi, CoMauToiThieu, TranLapLienTiep)
	}

	if caoNhat.soLan >= TranLapLienTiep {
		t.Fatalf("NGƯỠNG ĐANG BẮT OAN: phiên #%d (%s, trạng thái %q) là lượt chạy bình thường "+
			"nhưng lặp %q %d lần liên tiếp ≥ TranLapLienTiep=%d — Quan() sẽ kết tội nó chạy quẩn. "+
			"Nâng ngưỡng theo số đo này, đừng nâng theo cảm giác. Nhật ký: %s",
			caoNhat.id, caoNhat.addr, caoNhat.state, dongDau(caoNhat.lenh), caoNhat.soLan,
			TranLapLienTiep, caoNhat.duong)
	}
}

// TestCoMauDemRiengKhoiChuoiLap ghim CỠ MẪU là một số đếm ĐỘC LẬP với chuỗi lặp.
//
// Chạy được trên mọi máy, kể cả máy trắng: đây là phần bài đo thực ở trên KHÔNG
// tự chứng minh được cho mình. Nếu demQuan quên đếm thì bài đo thực chỉ tụt
// xuống SKIP vì "cỡ mẫu dưới sàn" — im lặng, không đỏ. Bài này làm nó đỏ.
func TestCoMauDemRiengKhoiChuoiLap(t *testing.T) {
	var d demQuan
	d.Them(chuKyTool("Bash", map[string]any{"command": "ls"}))
	d.Them(chuKyTool("Bash", map[string]any{"command": "ls"}))
	d.Them(chuKyTool("Bash", map[string]any{"command": "pwd"}))
	// Không bóc được tham số: NGẮT chuỗi, và KHÔNG tính vào cỡ mẫu — cỡ mẫu là
	// "đã đo được bao nhiêu", không phải "đã nhìn qua bao nhiêu".
	d.Them(chuKyTool("nao_do", nil))

	lenh, soLan, soLoiGoi, docDuoc := d.KetLuan()
	if !docDuoc {
		t.Fatal("bóc được lời gọi có tham số mà báo không đọc được")
	}
	if soLan != 2 || lenh != "Bash ls" {
		t.Errorf("chuỗi lặp dài nhất = %d %q, muốn 2 %q", soLan, lenh, "Bash ls")
	}
	// 3 chứ không phải 4: lời gọi thứ tư không bóc được tham số.
	if soLoiGoi != 3 {
		t.Errorf("cỡ mẫu = %d, muốn 3 — số này phải đếm MỌI lời gọi bóc được, "+
			"không phải độ dài chuỗi lặp", soLoiGoi)
	}
}

// laBinhThuong: trạng thái này có phải một lượt chạy BÌNH THƯỜNG không, tức có
// dùng làm mẫu đối chứng cho câu "ngưỡng có bắt oan không" được không.
//
// Loại đúng một trạng thái: `blocked`. Xem chỗ gọi để biết vì sao. Mọi trạng
// thái còn lại — kể cả `lost` (chết chưa rõ lý do) và `rate_limited` (hết hạn
// mức) — đều là agent đang làm việc thật cho tới lúc dừng, nên chuỗi lặp của
// chúng là số đo hợp lệ.
func laBinhThuong(state string) bool { return state != store.StateChan }

// doChuoiLap bóc chuỗi lặp liên tiếp dài nhất từ nhật ký của một phiên, dùng
// ĐÚNG đường sản phẩm chứ không viết lại phép bóc.
func doChuoiLap(p *phienDo) {
	b, err := os.ReadFile(p.duong)
	if err != nil {
		p.vi = "đọc file nhật ký hỏng: " + err.Error()
		return
	}
	// Cắt khối tiêu đề của sagent y như api.readLogs làm, để thứ đưa vào bộ đọc
	// đúng bằng thứ sản phẩm đưa vào.
	raw := nhatky.BoDau(string(b))

	ad, ok := Get(p.prov)
	if !ok {
		p.vi = "sổ ghi provider " + p.prov + " mà bảng adapter không có tên đó"
		return
	}
	switch k, ok := ad.DocKetQua(raw); {
	case ok:
		p.lenh, p.soLan, p.soLoiGoi, p.docDuoc = k.LenhLap, k.SoLanLap, k.SoLoiGoiTool, k.DemDuocTool
	case p.prov == "claude":
		// Bộ đọc kết quả cần dòng `{"type":"result"}`, mà phiên ĐANG CHẠY hoặc
		// bị giết giữa chừng thì chưa có dòng đó. Phép ĐẾM LẶP không cần nó —
		// nó chỉ cần các dòng tool_use — nên đi thẳng vào bộ đếm thay vì vứt
		// một bản ghi hoàn toàn đọc được. Chỉ làm cho Claude vì chỉ Claude có
		// hàm đếm tách rời khỏi bộ đọc kết quả.
		p.lenh, p.soLan, p.soLoiGoi, p.docDuoc = lapLaiClaude(strings.Split(raw, "\n"))
	default:
		p.vi = "bộ đọc kết quả của " + p.prov + " không đọc nổi bản ghi này " +
			"(thiếu dòng kết quả — phiên đang chạy hoặc bị giết giữa chừng?)"
		return
	}
	if !p.docDuoc {
		// Nói đúng thứ quan sát được: bản ghi đọc được, nhưng không có lời gọi
		// tool nào KÈM THAM SỐ. Hai nguyên nhân khác hẳn nhau đều ra kết quả
		// này, và bài đo không phân biệt được nên KHÔNG được đoán: lượt đó thật
		// sự chưa gọi tool lần nào (bản ghi ngắn, agent chết sớm), hoặc provider
		// có gọi mà bản ghi không phát tham số.
		p.vi = "bản ghi đọc được nhưng KHÔNG có lời gọi tool nào kèm tham số"
		if p.prov == "antigravity" {
			// Ca DUY NHẤT đã biết chắc nguyên nhân, và biết là do thiết kế.
			p.vi += " — với antigravity đây là ca đã đo và có chủ ý, xem ketqua_antigravity.go"
		}
	}
}

// docPhienCoNhatKy đọc bảng `sessions` của state.db THẬT, lấy các phiên có cột
// `log` trỏ tới một file còn tồn tại.
//
// Mở CHỈ ĐỌC (mode=ro), cố ý không đi qua store.Open: store.Open chạy migration,
// tức lấy khoá ghi trên sổ đang được các phiên fleet khác dùng thật. Một phép đo
// không được sửa thứ nó đo.
func docPhienCoNhatKy(t *testing.T) (ds []phienDo, mat, gop int) {
	t.Helper()
	duongSo := store.Path()
	if _, err := os.Stat(duongSo); err != nil {
		t.Skipf("máy này chưa có sổ phiên (%s) — không có gì để đo", duongSo)
	}
	db, err := sql.Open("sqlite", "file:"+duongSo+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Skipf("không mở được sổ phiên để đọc: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(
		`SELECT id,provider,account,clone,COALESCE(log,''),state FROM sessions
		 WHERE COALESCE(log,'') <> '' ORDER BY id`)
	if err != nil {
		t.Skipf("không đọc được bảng sessions: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s store.Session
		if err := rows.Scan(&s.ID, &s.Provider, &s.Account, &s.Clone, &s.Log, &s.State); err != nil {
			t.Fatalf("đọc dòng sessions hỏng: %v", err)
		}
		if _, err := os.Stat(s.Log); err != nil {
			mat++
			continue
		}
		ds = append(ds, phienDo{
			id: s.ID, addr: s.Addr(), prov: s.Provider, state: s.State, duong: s.Log,
		})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("duyệt bảng sessions hỏng: %v", err)
	}
	ds, gop = gomTheoFile(ds)
	return ds, mat, gop
}

// gomTheoFile gộp các dòng sổ TRỎ CHUNG một file nhật ký, giữ lại dòng MỚI NHẤT.
//
// Bắt buộc phải có, và lý do nằm ngay trong đầu internal/nhatky/nhatky.go: trước
// khi có gói nhatky, cột `log` trỏ tới `<thư mục clone>/fleet.log` — đường dẫn
// chỉ phụ thuộc số bản clone chứ không phụ thuộc phiên, và `os.Create` CẮT TRẮNG
// file mỗi lần bật. Nên hàng chục dòng sổ cũ cùng trỏ vào một file, mà nội dung
// file chỉ là của phiên GHI CUỐI CÙNG.
//
// Đếm cả hàng chục dòng đó là nhân bản MỘT số đo lên hàng chục lần: cỡ mẫu phồng
// lên giả tạo, và bảng in ra trông như đã đo nhiều phiên trong khi chỉ đo một.
func gomTheoFile(ds []phienDo) (out []phienDo, gop int) {
	viTri := map[string]int{}
	for _, p := range ds {
		khoa := strings.ToLower(filepath.Clean(p.duong))
		if i, co := viTri[khoa]; co {
			// ORDER BY id nên dòng tới sau luôn mới hơn: nó mới là phiên đã ghi
			// ra nội dung file đang nằm trên đĩa.
			p.gop = out[i].gop + 1
			out[i] = p
			gop++
			continue
		}
		viTri[khoa] = len(out)
		out = append(out, p)
	}
	return out, gop
}
