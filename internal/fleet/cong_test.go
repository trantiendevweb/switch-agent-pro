package fleet

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// Bộ này đo CHÍNH CÁI CỔNG. Nó không thay được internal/flow/tran_flow_test.go
// (bộ đó đo chỗ CẮM), và ngược lại cũng không: mấy ca dưới đây — đếm hai lần,
// nhả chỗ, cắt số xin — khó dựng qua một lượt chạy flow thật mà vẫn nhìn rõ.

func tranThu() Tran {
	return Tran{Chung: 4, HarnessMacDinh: 3, ProviderMacDinh: 3, HoSoMacDinh: 2}
}

func congThu(t *testing.T, nen []Phien) (*Cong, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var noi []string
	c := MoCong(tranThu(), func() []Phien { return nen }, func(m string) {
		mu.Lock()
		noi = append(noi, m)
		mu.Unlock()
	})
	c.Nhip = 5 * time.Millisecond
	c.ChoKetCung = 150 * time.Millisecond
	return c, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), noi...)
	}
}

// Cổng dùng LẠI XetTran, không đếm bằng một bộ máy thứ hai.
//
// Đo gián tiếp mà chắc: đổi trần hồ sơ thành 1 thì cổng phải đổi hành vi theo,
// nghĩa là nó thật sự hỏi bộ đếm chung chứ không giữ ngưỡng riêng.
func TestCongChanDungTheoTranHoSo(t *testing.T) {
	c, _ := congThu(t, nil)
	tns := Phien{Provider: "claude", Account: "tns"}

	a, err := c.Xin(context.Background(), "a", tns, 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.Xin(context.Background(), "b", tns, 1)
	if err != nil {
		t.Fatal(err)
	}
	if a == nil || b == nil {
		t.Fatal("hai chỗ đầu phải cấp được ngay")
	}

	// Chỗ thứ ba phải ĐỨNG CHỜ, không được từ chối và cũng không được cho qua.
	xong := make(chan *The, 1)
	go func() {
		the, err := c.Xin(context.Background(), "c", tns, 1)
		if err != nil {
			t.Errorf("chờ trần KHÔNG được thành lỗi: %v", err)
		}
		xong <- the
	}()
	select {
	case <-xong:
		t.Fatal("chỗ thứ ba lọt qua trần hồ sơ 2")
	case <-time.After(100 * time.Millisecond):
	}

	a.Tra() // nhả một chỗ -> bước đang chờ phải chạy tiếp
	select {
	case the := <-xong:
		if the == nil {
			t.Fatal("nhả chỗ rồi mà không cấp")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("nhả chỗ rồi mà bước đang chờ vẫn đứng — cổng không đánh thức hàng đợi")
	}
	b.Tra()
}

// KHÔNG ĐẾM HAI LẦN. Đây là chỗ dễ sai nhất của cả file cong.go: phiên do chính
// cổng bật ra cũng nằm trong sổ, nên đọc lại sổ mỗi lượt xét sẽ biến trần hồ sơ
// 2 thành 1 mà chẳng có thông báo nào.
//
// Dựng đúng ca đó: sổ trả về một phiên claude:tns MỚI mỗi lần được hỏi, như thể
// chỗ cổng vừa cấp đã kịp ghi vào sổ.
func TestCongKhongDemPhienCuaChinhNoHaiLan(t *testing.T) {
	tns := Phien{Provider: "claude", Account: "tns"}
	var lanDoc int
	var mu sync.Mutex
	c := MoCong(tranThu(), func() []Phien {
		mu.Lock()
		defer mu.Unlock()
		lanDoc++
		if lanDoc == 1 {
			return nil // lúc mở cổng: chưa có phiên nào
		}
		return []Phien{tns, tns} // lần sau: sổ đã có phiên của chính cổng
	}, nil)
	c.Nhip = 5 * time.Millisecond

	a, err := c.Xin(context.Background(), "a", tns, 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.Xin(context.Background(), "b", tns, 1)
	if err != nil {
		t.Fatalf("chỗ thứ HAI bị chặn: trần hồ sơ 2 đã bị đếm thành 1 vì phiên của chính cổng "+
			"bị cộng thêm một lần từ sổ. Lỗi: %v", err)
	}
	if a == nil || b == nil {
		t.Fatal("phải cấp đủ hai chỗ")
	}
	a.Tra()
	b.Tra()
}

// KẸT CỨNG: chỗ bị chiếm hết bởi phiên NGOÀI cổng, và cổng không giữ chỗ nào
// nên trong lượt này không có gì sẽ nhả ra. Phải BÁO RA, không được đứng im.
func TestCongBaoKetCungChuKhongTreo(t *testing.T) {
	tns := Phien{Provider: "claude", Account: "tns"}
	c, doc := congThu(t, []Phien{tns, tns})

	loi := make(chan error, 1)
	go func() {
		_, err := c.Xin(context.Background(), "bước a", tns, 1)
		loi <- err
	}()
	select {
	case err := <-loi:
		if err == nil {
			t.Fatal("hết sạch chỗ mà vẫn cấp")
		}
		for _, can := range []string{"KẸT CỨNG", "hồ sơ claude:tns", "sagent quet"} {
			if !strings.Contains(err.Error(), can) {
				t.Errorf("lời báo kẹt cứng thiếu %q:\n%s", can, err)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("TREO IM LẶNG: kẹt cứng mà cổng đứng luôn")
	}
	// Và phải nói NGAY từ lúc mới nghi, chứ không im suốt rồi mới bật ra một lỗi.
	if !strings.Contains(strings.Join(doc(), "\n"), "ĐỨNG vì hết chỗ") {
		t.Fatalf("nghi kẹt cứng mà không nói gì trong lúc chờ: %v", doc())
	}
}

// Chỗ bị chiếm bởi một lượt hạm đội NGOÀI sắp xong thì phải CHỜ nó, không được
// giết lượt chạy.
//
// Đây là số đo thật, không phải ca giả định: bản đầu của cong.go báo kẹt cứng
// NGAY khi thấy hình dạng đó, và trên máy (22/08) nó giết sạch bốn bước của một
// lượt flow vì hai phiên bên ngoài còn sống thêm khoảng một giây nữa. Đúng cái
// kết cục mà cả bản sửa này lập ra để chống.
func TestCongChoPhienNgoaiSapXongChuKhongGiet(t *testing.T) {
	tns := Phien{Provider: "claude", Account: "tns"}
	var mu sync.Mutex
	con := []Phien{tns, tns}
	c := MoCong(tranThu(), func() []Phien {
		mu.Lock()
		defer mu.Unlock()
		return con
	}, nil)
	c.Nhip = 5 * time.Millisecond
	c.ChoKetCung = 5 * time.Second // dư sức đợi lượt ngoài xong

	// Lượt hạm đội ngoài kết thúc sau 150ms.
	go func() {
		time.Sleep(150 * time.Millisecond)
		mu.Lock()
		con = nil
		mu.Unlock()
	}()

	xong := make(chan error, 1)
	go func() {
		_, err := c.Xin(context.Background(), "bước a", tns, 1)
		xong <- err
	}()
	select {
	case err := <-xong:
		if err != nil {
			t.Fatalf("lượt ngoài xong rồi mà bước vẫn bị giết: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("lượt ngoài đã xong mà cổng không cho vào")
	}
}

// Sổ đọc lúc mở cổng có thể CŨ. Khi cổng không giữ chỗ nào thì đọc lại là an
// toàn — và phải đọc lại, nếu không một phiên ngoài đã kết thúc từ lâu vẫn chặn
// cả lượt chạy.
func TestCongDocLaiSoKhiKhongGiuChoNao(t *testing.T) {
	tns := Phien{Provider: "claude", Account: "tns"}
	var mu sync.Mutex
	con := []Phien{tns, tns}
	c := MoCong(tranThu(), func() []Phien {
		mu.Lock()
		defer mu.Unlock()
		return con
	}, nil)
	c.Nhip = 5 * time.Millisecond

	// Phiên ngoài đã xong TRƯỚC khi bước xin chỗ.
	mu.Lock()
	con = nil
	mu.Unlock()

	the, err := c.Xin(context.Background(), "a", tns, 1)
	if err != nil {
		t.Fatalf("phiên ngoài đã xong mà cổng vẫn chặn bằng số cũ: %v", err)
	}
	the.Tra()
}

// Xin nhiều hơn mức trần BAO GIỜ cũng cho thì cắt xuống rồi chạy, chứ không xếp
// hàng chờ một con số không bao giờ tới.
func TestCongCatSoXinKhiTranKhongBaoGioDu(t *testing.T) {
	c, doc := congThu(t, nil)
	tns := Phien{Provider: "claude", Account: "tns"}

	the, err := c.Xin(context.Background(), "b1", tns, 4)
	if err != nil {
		t.Fatal(err)
	}
	if the.Cap != 2 {
		t.Fatalf("trần hồ sơ 2 mà cấp %d", the.Cap)
	}
	if !strings.Contains(strings.Join(doc(), "\n"), "cắt 4 phiên xuống 2") {
		t.Fatalf("cắt mà không nói ra: %v", doc())
	}
	the.Tra()
}

// ctx bị huỷ giữa lúc đang chờ thì THOÁT, không giữ goroutine lại.
func TestCongThoatKhiCtxBiHuy(t *testing.T) {
	c, _ := congThu(t, nil)
	tns := Phien{Provider: "claude", Account: "tns"}
	a, _ := c.Xin(context.Background(), "a", tns, 1)
	b, _ := c.Xin(context.Background(), "b", tns, 1)
	defer a.Tra()
	defer b.Tra()

	ctx, huy := context.WithCancel(context.Background())
	loi := make(chan error, 1)
	go func() {
		_, err := c.Xin(ctx, "c", tns, 1)
		loi <- err
	}()
	time.Sleep(30 * time.Millisecond)
	huy()
	select {
	case err := <-loi:
		if err == nil {
			t.Fatal("huỷ ctx mà vẫn cấp chỗ")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("huỷ ctx rồi mà bước đang chờ không thoát")
	}
}

// Cổng nil = không áp trần. Đường nào chưa cắm thì giữ hành vi cũ chứ không nổ.
func TestCongNilThiKhongApTran(t *testing.T) {
	var c *Cong
	the, err := c.Xin(context.Background(), "a", Phien{Provider: "claude", Account: "tns"}, 9)
	if err != nil || the != nil {
		t.Fatalf("cổng nil phải cho qua im lặng, được (%v, %v)", the, err)
	}
	the.Tra() // nil cũng phải trả được
}

// Tra gọi hai lần không được nhả nhầm chỗ của người khác.
func TestTraHaiLanVoHai(t *testing.T) {
	c, _ := congThu(t, nil)
	tns := Phien{Provider: "claude", Account: "tns"}
	a, _ := c.Xin(context.Background(), "a", tns, 1)
	a.Tra()
	a.Tra()

	b, _ := c.Xin(context.Background(), "b", tns, 1)
	d, _ := c.Xin(context.Background(), "d", tns, 1)
	if b == nil || d == nil {
		t.Fatal("phải còn đủ hai chỗ")
	}
	// Chỗ thứ ba vẫn phải bị chặn — Tra thừa không được nhân chỗ ra.
	ctx, huy := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer huy()
	if _, err := c.Xin(ctx, "e", tns, 1); err == nil {
		t.Fatal("Tra gọi hai lần đã nhả thừa một chỗ")
	}
	b.Tra()
	d.Tra()
}

func TestPhienTuTachDiaChi(t *testing.T) {
	for _, ca := range []struct{ vao, prov, acc string }{
		{"claude:tns", "claude", "tns"},
		{"tns", "claude", "tns"},
		{"codex:phu", "codex", "phu"},
	} {
		p := PhienTu(ca.vao)
		if p.Provider != ca.prov || p.Account != ca.acc {
			t.Errorf("PhienTu(%q) = %+v, muốn %s/%s", ca.vao, p, ca.prov, ca.acc)
		}
	}
}
