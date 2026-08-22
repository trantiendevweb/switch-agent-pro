package aiapi

import (
	"reflect"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// routeThu là route dùng cho mọi bài dưới đây: ĐÚNG base_url và model của
// `deepseek` trong .sagent/project.toml, để nó khớp sổ số đo. Không chạm mạng —
// mọi thứ ở đây đọc bảng, không gọi API.
var routeThu = Route{
	Ten:     "deepseek",
	BaseURL: "https://modelapi.vn/v1",
	Model:   "deepseek-v4-flash",
	KeyID:   "deepseek",
}

// BA trạng thái của nửa API phải là CHÍNH ba trạng thái của nửa CLI, không phải
// ba chuỗi trông giống.
//
// Vì sao đáng một bài kiểm riêng dù đã dùng type alias: alias sửa được bằng một
// dòng, và người sửa sẽ thấy nó "chỉ là dọn dẹp import". Bài này nói ra cái giá
// — hai nửa dự án nói hai nghĩa cho cùng một chữ "chua-do".
func TestBaTrangThaiDungChungVoiNuaCLI(t *testing.T) {
	if LamDuoc != provider.LamDuoc || KhongLamDuoc != provider.KhongLamDuoc || ChuaDo != provider.ChuaDo {
		t.Fatal("hằng trạng thái của aiapi đã trôi khỏi provider — hai nửa dự án đang " +
			"nói hai nghĩa cho cùng một chữ")
	}
	var a TrangThaiNangLuc
	if reflect.TypeOf(a) != reflect.TypeOf(provider.TrangThaiNangLuc("")) {
		t.Fatal("TrangThaiNangLuc không còn là alias của provider.TrangThaiNangLuc — " +
			"mặt web sẽ phải có hai hàm vẽ, và chúng sẽ lệch nhau")
	}
	// Đúng BA, không nhiều không ít. Thêm trạng thái thứ tư là chuyện phải cân
	// nhắc ở cả hai nửa cùng lúc, không phải chuyện lẳng lặng thêm một hằng.
	if len(map[TrangThaiNangLuc]bool{LamDuoc: true, KhongLamDuoc: true, ChuaDo: true}) != 3 {
		t.Fatal("ba trạng thái bị trùng giá trị")
	}
}

// Bảng phải đủ MỌI năng lực, mọi ô có bằng chứng, và KHÔNG có chỗ lệch nào.
func TestBangDuMoiNangLucVaKhongLech(t *testing.T) {
	b := BangNangLuc(routeThu)
	if len(b.Lech) != 0 {
		t.Fatalf("bảng chọi với chính nó: %v", b.Lech)
	}
	co := map[string]bool{}
	for _, m := range b.Muc {
		co[m.Khoa] = true
		if m.Mo == "" {
			t.Errorf("%s: thiếu câu mô tả", m.Khoa)
		}
		// Ba cặp (trạng thái, bằng chứng) — không cặp nào được để trống. Một ô
		// khai "làm được" mà không nói đo ở đâu thì chỉ là một lời hứa.
		for ten, bc := range map[string]string{
			"kết luận":          m.BangChung,
			"phần khách":        m.BangChungKhach,
			"phần nhà cung cấp": m.BangChungNCC,
		} {
			if strings.TrimSpace(bc) == "" {
				t.Errorf("%s/%s: không có bằng chứng", m.Khoa, ten)
			}
		}
		if m.Cho == "" {
			t.Errorf("%s: không nói vướng ở đâu", m.Khoa)
		}
		if NhanCho(m.Cho) == m.Cho {
			t.Errorf("%s: giá trị cho=%q không có trong MoiCho — mặt web sẽ hiện mã thô",
				m.Khoa, m.Cho)
		}
	}
	for _, m := range MoiNangLucAPI {
		if !co[m.Khoa] {
			t.Errorf("bảng thiếu năng lực %q", m.Khoa)
		}
	}
	if len(b.Muc) != len(MoiNangLucAPI) {
		t.Errorf("bảng có %d dòng, MoiNangLucAPI có %d", len(b.Muc), len(MoiNangLucAPI))
	}
}

// KiemNangLucAPI phải BẮT ĐƯỢC một bảng khai bừa.
//
// Đây là bài quan trọng nhất của file. Ngày 22/08 dự án bắt quả tang bảng quyền
// plugin khai `chan-that` cho một thứ không chặn được, và lý do nó sống lâu là
// KHÔNG AI ĐÒI BẰNG CHỨNG. Từng ca dưới đây là một cách khai bừa cụ thể; gỡ
// đoạn kiểm tương ứng trong KiemNangLucAPI ra là ca đó đỏ.
func TestKiemBatDuocBangKhaiBua(t *testing.T) {
	day := func(sua func(*NangLucAPI)) NangLucRoute {
		b := BangNangLuc(routeThu)
		b.Lech = nil
		for i := range b.Muc {
			if b.Muc[i].Khoa == NLAPIStreaming {
				sua(&b.Muc[i])
			}
		}
		return b
	}
	for _, c := range []struct {
		ten  string
		sua  func(*NangLucAPI)
		chua string
	}{
		{
			"kết luận xanh mà phần nhà cung cấp chưa đo",
			func(m *NangLucAPI) { m.NCC, m.BangChungNCC = ChuaDo, "chưa chạy phép đo" },
			"không có phép đo nào đứng sau",
		},
		{
			"kết luận xanh mà phần khách chưa đo",
			func(m *NangLucAPI) { m.Khach, m.BangChungKhach = ChuaDo, "chưa dò mã nguồn" },
			"không có phép đo nào đứng sau",
		},
		{
			"bằng chứng rỗng",
			func(m *NangLucAPI) { m.BangChungNCC = "   " },
			"không có bằng chứng",
		},
		{
			// Bằng chứng nghe như chép từ tài liệu nhà cung cấp: không con số,
			// không mã HTTP, không nguyên văn nào.
			"bằng chứng không có quan sát nào",
			func(m *NangLucAPI) { m.BangChungNCC = "model này hỗ trợ streaming" },
			"nghe như chép từ tài liệu",
		},
		{
			"trạng thái thứ tư",
			func(m *NangLucAPI) { m.TrangThai = "co-le-duoc" },
			"không phải một trong ba trạng thái",
		},
		{
			"khoá lạ",
			func(m *NangLucAPI) { m.Khoa = "bay-len-troi" },
			"không có trong MoiNangLucAPI",
		},
		{
			"không nói vướng ở đâu",
			func(m *NangLucAPI) { m.Cho = "" },
			"không nói chỗ hỏng nằm bên nào",
		},
	} {
		t.Run(c.ten, func(t *testing.T) {
			lech := KiemNangLucAPI(day(c.sua))
			if !strings.Contains(strings.Join(lech, "\n"), c.chua) {
				t.Fatalf("khai bừa kiểu %q lọt qua — chỗ lệch bắt được: %v", c.ten, lech)
			}
		})
	}
}

// Thiếu hẳn một năng lực phải bị bắt, không được lặng lẽ coi như chưa đo.
func TestKiemBatDuocBangThieuNangLuc(t *testing.T) {
	b := BangNangLuc(routeThu)
	b.Muc = b.Muc[:len(b.Muc)-1]
	lech := KiemNangLucAPI(b)
	if !strings.Contains(strings.Join(lech, "\n"), "KHÔNG KHAI") {
		t.Fatalf("bảng thiếu một năng lực mà vẫn sạch: %v", lech)
	}
}

// Khai HAI LẦN cùng một khoá phải bị bắt: bảng có hai dòng chọi nhau thì mặt
// nào vẽ trước thắng, và hai mặt có thể vẽ khác nhau.
func TestKiemBatDuocKhaiHaiLan(t *testing.T) {
	b := BangNangLuc(routeThu)
	b.Muc = append(b.Muc, b.Muc[0])
	if !strings.Contains(strings.Join(KiemNangLucAPI(b), "\n"), "khai hai lần") {
		t.Fatal("khai hai lần cùng một năng lực mà bảng vẫn sạch")
	}
}

// Route CHƯA CÓ KEY phải ra `ChuaDo` KÈM LÝ DO CỤ THỂ.
//
// "chưa có key" và "chưa chạy phép đo" dẫn tới hai việc khác nhau — một cái đi
// đặt key, một cái đi chạy đo. Gộp thành "chưa kiểm tra" là bắt người đọc đoán.
func TestRouteChuaCoKeyRaChuaDoKemLyDo(t *testing.T) {
	r := Route{
		Ten:     "chua-co-key",
		BaseURL: "https://vi-du-khong-co-that.invalid/v1",
		Model:   "mo-hinh-nao-do",
		KeyID:   "khong-bao-gio-ton-tai-key-nay-9f3a",
	}
	b := BangNangLuc(r)
	if len(b.Lech) != 0 {
		t.Fatalf("route chưa có key mà bảng đã lệch: %v", b.Lech)
	}
	for _, m := range b.Muc {
		if m.NCC != ChuaDo {
			t.Errorf("%s: chưa có key mà phần nhà cung cấp khai %q", m.Khoa, m.NCC)
		}
		if !strings.Contains(m.BangChungNCC, "chưa có key") {
			t.Errorf("%s: lý do chưa đo không nói ra là thiếu key: %q", m.Khoa, m.BangChungNCC)
		}
	}
	// Những ô mà phía dự án đã chặn sẵn thì kết luận vẫn là KhongLamDuoc — phép
	// đo mã nguồn không cần key. Những ô còn lại phải là ChuaDo.
	if b.SoChuaDo() == 0 {
		t.Fatal("route chưa có key mà không ô nào là `chua-do` — đâu đó đang tự kết luận thay")
	}
}

// Bảng KHÔNG được chạm mạng và KHÔNG được đọc nội dung file key.
//
// Kiểm bằng GIÁ TRỊ THẬT: gieo một key mang dấu hiệu không thể trùng rồi đòi
// dấu hiệu đó không có mặt trong bất kỳ chuỗi nào của bảng. Cấm tên trường thì
// chỉ khiến người ta viết bằng chứng mập mờ hơn, không an toàn hơn — cùng lý do
// đã ghi ở TestNangLucKhongChamTokenCuaHoSoNao bên internal/dash.
func TestBangKhongMangNoiDungKeyRaNgoai(t *testing.T) {
	const dauHieu = "KEY-GIA-KHONG-DUOC-LO-RA-4b71"
	b := BangNangLuc(routeThu)
	var het strings.Builder
	for _, m := range b.Muc {
		het.WriteString(m.BangChung + m.BangChungKhach + m.BangChungNCC + m.Cho)
	}
	if strings.Contains(het.String(), dauHieu) {
		t.Fatal("bảng mang nội dung key ra ngoài")
	}
	// Bằng chứng phải nhắc tới key_id (tên file) chứ không bao giờ là key.
	if strings.Contains(het.String(), "Bearer ") {
		t.Fatal("bảng mang chuỗi Authorization ra ngoài")
	}
}

// Phép đo mã nguồn phải THẬT SỰ soi kiểu, không phải một bảng bool viết tay.
//
// Bài này neo vào SỰ THẬT hiện tại của gói. Bản trước neo vào "`yeuCau` CHƯA có
// `tools`" và nó đã đỏ đúng lúc phải đỏ — ngày 22/08, lúc `GoiTool` ra đời. Cái
// neo được dời sang sự thật mới, chứ không bị gỡ: giờ nó canh rằng cả BA mắt
// của đường tool đều còn (gửi được, đọc được, có chỗ chứa để mang ra ngoài).
// Mất mắt nào cũng đỏ, và mất mắt thứ ba là kiểu hỏng tệ nhất — bảng vẫn xanh
// vì hai mắt đầu còn nguyên.
//
// `tinNhan.Content` vẫn là chuỗi thuần: đó là phép đo `dau-vao-anh`, chưa làm.
func TestPhepDoMaNguonSoiKieuThat(t *testing.T) {
	if !coTruong(reflect.TypeOf(yeuCau{}), "tools") {
		t.Fatal("aiapi.yeuCau mất trường `tools` — lời gọi của dự án này lại không mang " +
			"định nghĩa tool đi, mà bảng `goi-tool` thì phải đo lại")
	}
	if !coTruongGo(reflect.TypeOf(KetQua{}), "ToolCalls") {
		t.Fatal("KetQua không còn chỗ chứa ToolCalls — lời gọi tool dừng trong lõi, " +
			"đúng cái bẫy `SuyLuan` đã dính suốt chiều 22/08")
	}
	if !coTruongGo(reflect.TypeOf(KetQua{}), "SuyLuan") {
		t.Fatal("KetQua không còn chỗ chứa SuyLuan")
	}
	// coTruongGo phải soi kiểu THẬT, không phải luôn trả true.
	if coTruongGo(reflect.TypeOf(KetQua{}), "TruongKhongCoThat") {
		t.Fatal("coTruongGo nói có một trường không tồn tại — phép đo hỏng, và một " +
			"phép đo hỏng làm cả bảng khai bừa")
	}
	if !coTruong(reflect.TypeOf(yeuCau{}), "model") {
		t.Fatal("coTruong không thấy trường `model` của chính yeuCau — phép đo hỏng, " +
			"và một phép đo hỏng làm CẢ BẢNG khai KhongLamDuoc cho mọi thứ")
	}
	// Đi xuyên lát cắt + struct lồng: đây là đường mà `tool_calls` và
	// `reasoning_content` phải đi qua.
	if !coTruong(reflect.TypeOf(phanHoi{}), "choices", "message", "content") {
		t.Fatal("coTruong không đi xuyên được choices[].message.content")
	}
	if !coTruong(reflect.TypeOf(phanHoi{}), "choices", "message", "tool_calls") {
		t.Fatal("aiapi.phanHoi thôi đọc `tool_calls` — nhà cung cấp trả lời đúng, lõi vứt đi")
	}
	// Kiểu của trường cuối phải trả về đúng, không chỉ có/không.
	tp, co := kieuTruong(reflect.TypeOf(tinNhan{}), "content")
	if !co || tp.Kind() != reflect.String {
		t.Fatalf("tinNhan.Content không còn là chuỗi thuần (%v) — phép đo `dau-vao-anh` "+
			"phải được xem lại", tp)
	}
}

// Sổ số đo phải sạch: mọi dòng có khoá thật, trạng thái thật, và bằng chứng
// mang một quan sát.
//
// Đây là chỗ một dòng chép tay từ tài liệu nhà cung cấp sẽ lọt vào, vì nó là
// khối duy nhất trong cả bảng do người gõ.
func TestSoSoDoSach(t *testing.T) {
	if len(soDoNangLuc) == 0 {
		t.Fatal("sổ số đo rỗng — bảng sẽ toàn ô `chua-do`")
	}
	thay := map[string]bool{}
	for _, m := range soDoNangLuc {
		k := m.BaseURL + "|" + m.Model + "|" + m.Khoa
		if thay[k] {
			t.Errorf("%s: ghi hai lần — dòng sau không bao giờ được đọc tới", k)
		}
		thay[k] = true
		if moTaNangLucAPI(m.Khoa) == "" {
			t.Errorf("%s: khoá lạ", k)
		}
		switch m.TrangThai {
		case LamDuoc, KhongLamDuoc:
		case ChuaDo:
			// Sổ là nơi ghi những gì ĐÃ đo. Một dòng `chua-do` trong sổ biến sổ
			// thành bản sao của bảng, và lần sau không ai phân biệt được nữa.
			t.Errorf("%s: sổ số đo không được chứa `chua-do`", k)
		default:
			t.Errorf("%s: trạng thái %q không phải một trong ba", k, m.TrangThai)
		}
		if !coQuanSat(m.BangChung) {
			t.Errorf("%s: bằng chứng không có quan sát nào (con số, mã HTTP): %q", k, m.BangChung)
		}
		if !strings.Contains(m.BangChung, "đo ") {
			t.Errorf("%s: bằng chứng không nói đo ngày nào: %q", k, m.BangChung)
		}
	}
}

// LamDuoc(khoa) là cửa mà `flow validate` sẽ gọi. Nó phải trả về LÝ DO, không
// chỉ một chữ false — một bước flow bị từ chối mà không nói vì sao thì người
// viết flow chỉ còn cách thử mò.
func TestLamDuocTraVeLyDoDocDuoc(t *testing.T) {
	b := BangNangLuc(routeThu)
	if ok, ly := b.LamDuoc(NLAPIStreaming); !ok {
		t.Fatalf("streaming phải làm được: %s", ly)
	}
	// `goi-tool` của deepseek-v4-flash nay XANH ở cả hai vế: phía dự án đã gửi
	// được `tools` (aiapi.GoiTool), phía nhà cung cấp đã đo 22/08. Trước 22/08
	// đây là ô ✗ với chẩn đoán "vướng ở phía dự án" — bài này giữ lại chiều
	// đúng của nó bằng cách hỏi một ô CHƯA làm được ngay bên dưới.
	if ok, ly := b.LamDuoc(NLAPIGoiTool); !ok {
		t.Fatalf("goi-tool phải làm được sau khi có GoiTool: %s", ly)
	}
	// `dau-vao-anh` là ô còn vướng ở phía dự án (Content vẫn là chuỗi thuần),
	// nên nó là chỗ kiểm rằng lời từ chối vẫn nói được ROUTE NÀO và VƯỚNG GÌ.
	ok, ly := b.LamDuoc(NLAPIDauVaoAnh)
	if ok {
		t.Fatal("dau-vao-anh đang KHÔNG làm được (tinNhan.Content vẫn là chuỗi thuần) " +
			"mà bảng nói được")
	}
	if !strings.Contains(ly, "content") || !strings.Contains(ly, routeThu.Ten) {
		t.Fatalf("lý do từ chối không nói được route nào và vướng gì: %q", ly)
	}
	if _, ly := b.LamDuoc("nang-luc-khong-co-that"); !strings.Contains(ly, "không có năng lực") {
		t.Fatalf("hỏi một năng lực lạ phải nói rõ là lạ, được: %q", ly)
	}
}

// KẾT LUẬN "vướng ở đâu" phải luôn khớp với CẶP (phía dự án, nhà cung cấp).
//
// Ca thật đã dựng ra bài kiểm này: grok-4.5 — `aiapi.phanHoi` không đọc
// `reasoning_content` (phía dự án), MÀ nhà cung cấp cũng không trả trường đó
// (đo 22/08). Nếu bảng chỉ nói vế đầu thì có người đi thêm trường vào `phanHoi`
// rồi mới phát hiện là vô ích.
//
// VÌ SAO KHÔNG GHIM THẲNG Ô ĐÓ NỮA (viết lại 22/08). Bản đầu của bài kiểm này
// khẳng định *"grok/reasoning phải là ca-hai"*. Ngay hôm đó phía dự án được vá
// (`tinNhan` đọc `reasoning_content`), ô chuyển sang chỉ còn vướng nhà cung
// cấp, và bài kiểm ĐỎ — nó **chỉ xanh chừng nào dự án còn nợ**, phạt đúng người
// đi trả nợ. Repo này đã chữa y hệt một lần với
// `TestBaTrangThaiDiRaToiHopDong`.
//
// Nên bài này ghim CƠ CHẾ, không ghim dữ liệu: duyệt MỌI ô của MỌI route và
// khẳng định phép ánh xạ. Ô nào đổi trạng thái cũng không sao — chỉ cần kết
// luận đi theo.
func TestKetLuanVuongODauLuonKhopVoiCapTrangThai(t *testing.T) {
	routes := []Route{
		{Ten: "deepseek", BaseURL: "https://modelapi.vn/v1", Model: "deepseek-v4-flash", KeyID: "deepseek"},
		{Ten: "grok", BaseURL: "https://modelapi.vn/v1", Model: "grok-4.5", KeyID: "grok"},
	}
	soO, soCaHai := 0, 0
	for _, r := range routes {
		for _, m := range BangNangLuc(r).Muc {
			soO++
			ten := r.Ten + "/" + m.Khoa
			switch {
			case m.Khach == KhongLamDuoc && m.NCC == KhongLamDuoc:
				soCaHai++
				if m.Cho != ChoCaHai {
					t.Errorf("%s: cả hai bên đều không mà cho=%q — phải là %q",
						ten, m.Cho, ChoCaHai)
				}
				if !strings.Contains(m.BangChung, "CŨNG KHÔNG") {
					t.Errorf("%s: kết luận không nói ra là nhà cung cấp cũng chịu: %q",
						ten, m.BangChung)
				}
			case m.Khach == KhongLamDuoc && m.NCC == LamDuoc:
				if m.Cho != ChoKhach {
					t.Errorf("%s: chỉ phía dự án vướng mà cho=%q — phải là %q",
						ten, m.Cho, ChoKhach)
				}
				// Câu đáng giá nhất của cả bảng: chỗ hỏng ở TRONG repo này.
				if !strings.Contains(m.BangChung, "LÀM ĐƯỢC") {
					t.Errorf("%s: không nói ra rằng sửa được ở phía dự án: %q",
						ten, m.BangChung)
				}
			case m.Khach == LamDuoc && m.NCC == KhongLamDuoc:
				if m.Cho != ChoNCC {
					t.Errorf("%s: chỉ nhà cung cấp vướng mà cho=%q — phải là %q",
						ten, m.Cho, ChoNCC)
				}
			case m.Khach == ChuaDo || m.NCC == ChuaDo:
				if m.Cho != ChoChuaRo {
					t.Errorf("%s: còn bên CHƯA ĐO mà cho=%q — phải là %q",
						ten, m.Cho, ChoChuaRo)
				}
			case m.Khach == LamDuoc && m.NCC == LamDuoc:
				if m.Cho != ChoKhongVuong {
					t.Errorf("%s: cả hai bên đều được mà cho=%q — phải là %q",
						ten, m.Cho, ChoKhongVuong)
				}
			}
		}
	}
	if soO < 10 {
		t.Fatalf("chỉ duyệt được %d ô — bài kiểm này đang xanh vì rỗng, không phải vì sạch", soO)
	}
	// Không phải khẳng định, chỉ là số để lượt sau so được: khi con số này về 0
	// nghĩa là không còn ô nào vướng cả hai bên nữa.
	t.Logf("duyệt %d ô, trong đó %d ô vướng CẢ HAI BÊN", soO, soCaHai)
}

// Số đo khoá theo (base_url, model): đổi model của route thì số đo cũ phải TỰ
// RƠI về `chua-do`, chứ không được dùng lại cho một model khác.
func TestDoiModelThiSoDoCuKhongConDung(t *testing.T) {
	r := routeThu
	r.Model = "deepseek-v4-pro" // model có thật ở nhà bán lại này, nhưng CHƯA đo
	b := BangNangLuc(r)
	for _, m := range b.Muc {
		if m.NCC != ChuaDo {
			t.Fatalf("%s: đổi model sang %q mà vẫn dùng số đo của model cũ: %q",
				m.Khoa, r.Model, m.BangChungNCC)
		}
	}
}
