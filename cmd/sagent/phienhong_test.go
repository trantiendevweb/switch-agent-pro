package main

import (
	"strings"
	"testing"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Mặt TERMINAL của ba trạng thái phiên đo được.
//
// Trạng thái đã được quyết một lần ở hợp đồng; ở đây chỉ kiểm terminal có NÓI
// RA nó không. Trước bản này `sagent status` chỉ có bảng "phiên đang chạy", nên
// một hạm đội chết vì hết hạn mức trông y hệt một hạm đội đã làm xong việc.

var mocDo = time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)

// Mã máy không được là thứ duy nhất hiện ra: người vận hành đọc "rate_limited"
// rồi vẫn phải đi tra nghĩa.
func TestNhanTrangThaiNoiTiengViet(t *testing.T) {
	muon := map[string]string{
		store.StateHanMuc: "hết hạn mức",
		store.StateChan:   "bị chặn quyền",
		store.StateHong:   "lỗi nhà cung cấp",
	}
	for ma, chu := range muon {
		if got := nhanTrangThai(ma); got != chu {
			t.Errorf("nhanTrangThai(%q) = %q, muốn %q", ma, got, chu)
		}
	}
	// `lost` phải nói thẳng là KHÔNG BIẾT. Một nhãn nghe hợp lý ("đã kết thúc")
	// sẽ khiến người ta thôi không đào tiếp.
	if got := nhanTrangThai(store.StateLost); !strings.Contains(got, "chưa rõ") {
		t.Errorf("nhanTrangThai(lost) = %q — phải nói thẳng là chưa biết vì sao", got)
	}
	// Trạng thái lạ (bản sagent cũ hơn đọc sổ mới) trả nguyên mã, không nuốt.
	if got := nhanTrangThai("chua_biet_la_gi"); got != "chua_biet_la_gi" {
		t.Errorf("trạng thái lạ bị đổi thành %q thay vì hiện nguyên mã", got)
	}
}

// Mốc hạn mức phải thành câu người dùng làm được gì với nó.
func TestDongPhienHongNoiLucNaoChayLaiDuoc(t *testing.T) {
	s := store.Session{
		ID: 7, Provider: "claude", Account: "phu", Clone: 2,
		State: store.StateHanMuc, StateLyDo: "hết hạn mức, chờ được cấp lại",
		HanMucDenLai: mocDo.Add(90 * time.Minute).Unix(),
	}
	d := dongPhienHong(s, mocDo)
	for _, phai := range []string{"#7", "claude:phu#2", "hết hạn mức", "cấp lại sau", "1h30m"} {
		if !strings.Contains(d, phai) {
			t.Errorf("dòng thiếu %q:\n%s", phai, d)
		}
	}
	if !strings.Contains(d, s.StateLyDo) {
		t.Errorf("dòng không mang lý do đo được:\n%s", d)
	}
}

// Mốc đã QUA rồi thì câu phải đổi: "chờ 0 phút" là vô nghĩa, thứ người ta cần
// nghe là "chạy lại được".
func TestDongPhienHongMocDaQuaThiBaoChayLaiDuoc(t *testing.T) {
	s := store.Session{
		ID: 3, Provider: "claude", Account: "phu", State: store.StateHanMuc,
		HanMucDenLai: mocDo.Add(-2 * time.Hour).Unix(),
	}
	d := dongPhienHong(s, mocDo)
	if !strings.Contains(d, "chạy lại được") {
		t.Errorf("mốc cấp lại đã qua mà dòng không nói chạy lại được:\n%s", d)
	}
	if strings.Contains(d, "cấp lại sau") {
		t.Errorf("mốc đã qua mà vẫn bảo chờ:\n%s", d)
	}
}

// KHÔNG ĐO ĐƯỢC thì ô để trống, không lấp bằng chữ suy ra. Phiên `lost` không
// có mốc hạn mức và không có lý do — dòng của nó phải im về cả hai.
func TestDongPhienHongKhongBiaChoPhienKhongDoDuoc(t *testing.T) {
	s := store.Session{ID: 1, Provider: "codex", Account: "a", State: store.StateLost}
	d := dongPhienHong(s, mocDo)
	for _, cam := range []string{"cấp lại", "hạn mức", "chặn quyền", "nhà cung cấp"} {
		if strings.Contains(d, cam) {
			t.Errorf("phiên không đo được mà dòng nhắc %q:\n%s", cam, d)
		}
	}
	if !strings.Contains(d, "chưa rõ") {
		t.Errorf("dòng phải nói thẳng là chưa rõ vì sao:\n%s", d)
	}
}
