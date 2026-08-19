package provider

import (
	"strings"
	"testing"
)

// CA DA NO THAT (lan chay #29, buoc `code-go`): Claude tra ve is_error=true
// nhung subtype VAN LA "success". Thong bao cu ghep thang subtype vao sau chu
// "agent bao loi", ra thanh "agent bao loi: success" — tu mau thuan, va giau mat
// ly do that.
func TestCoLoiNhungSubtypeLaSuccess(t *testing.T) {
	k := KetQua{
		CoLoi:  true,
		Loai:   "success",
		TraLoi: "Credit balance is too low",
	}
	ly := k.Hong()
	if ly == "" {
		t.Fatal("co loi ma bao khong hong")
	}
	if strings.Contains(ly, "success") {
		t.Fatalf("khong duoc noi \"bao loi: success\", duoc %q", ly)
	}
	if !strings.Contains(ly, "Credit balance is too low") {
		t.Fatalf("phai dua ly do that trong `result` ra, duoc %q", ly)
	}
}

// Khong co `result` thi lui ve terminal_reason / stop_reason — van phai bo qua
// moi gia tri tu nhan la success.
func TestCoLoiKhongCoResultThiLuiVeLyDoKhac(t *testing.T) {
	k := KetQua{CoLoi: true, Loai: "success", KetCuc: "refusal"}
	if ly := k.Hong(); !strings.Contains(ly, "refusal") {
		t.Fatalf("phai lui ve terminal_reason, duoc %q", ly)
	}

	k = KetQua{CoLoi: true, Loai: "success", DungViCo: "max_tokens"}
	if ly := k.Hong(); !strings.Contains(ly, "max_tokens") {
		t.Fatalf("phai lui ve stop_reason, duoc %q", ly)
	}
}

// Khong con gi de noi thi noi thang la khong biet, dung bia mot chu nghe cho co.
func TestCoLoiMaKhongAiNoiLyDo(t *testing.T) {
	k := KetQua{CoLoi: true, Loai: "success"}
	ly := k.Hong()
	if strings.Contains(ly, "success") {
		t.Fatalf("khong duoc lay \"success\" lam ly do hong, duoc %q", ly)
	}
	if !strings.Contains(ly, "khong noi ly do") && !strings.Contains(ly, "không nói lý do") {
		t.Fatalf("phai noi thang la khong biet ly do, duoc %q", ly)
	}
}

// subtype that su la loi thi van phai giu nguyen, dung cat mat thong tin.
func TestSubtypeLoiThatThiGiuNguyen(t *testing.T) {
	k := KetQua{CoLoi: true, Loai: "error_max_turns"}
	if ly := k.Hong(); !strings.Contains(ly, "error_max_turns") {
		t.Fatalf("subtype loi that phai duoc giu, duoc %q", ly)
	}
}

// LoiAPI van la duong uu tien cao nhat, khong doi.
func TestLoiAPIVanUuTienNhat(t *testing.T) {
	k := KetQua{CoLoi: true, Loai: "error_during_execution", LoiAPI: "429", TraLoi: "gi do"}
	if ly := k.Hong(); !strings.Contains(ly, "429") {
		t.Fatalf("phai uu tien api_error_status, duoc %q", ly)
	}
}
