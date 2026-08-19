package flow

import (
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// CA DA NO THAT — hai lan trong ngay 19/08:
//
//	#29 chet theo lan may tu khoi dong lai luc 01:47
//	#30 chet khi nguoi dung dung tay luc 19:37
//
// Ca hai deu nam lai `running` VINH VIEN. `Reject` khong cuu duoc vi no doi mot
// buoc DANG CHO DUYET, ma luot bi cat ngang thi khong co buoc nao nhu vay.
// Bang lich su hien "dang chay" trong khi khong tien trinh nao con song — no
// noi doi dung thu no sinh ra de noi that.
func TestHuyLuotChayBiCatNgang(t *testing.T) {
	r, _, db := newRunner(t)

	id, err := db.CreateRun("dem", t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	// Dung lai hien truong: mot buoc dang chay, mot buoc da xong.
	if err := db.SetStep(id, "code-doc", store.StepDone, "", 1); err != nil {
		t.Fatal(err)
	}
	if err := db.SetStep(id, "soi", store.StepRunning, "", 1); err != nil {
		t.Fatal(err)
	}

	if err := r.Huy(id, "tester"); err != nil {
		t.Fatal(err)
	}

	run, err := db.GetRun(id)
	if err != nil {
		t.Fatal(err)
	}
	if run.State != store.RunCanceled {
		t.Fatalf("luot chay phai thanh %q, duoc %q", store.RunCanceled, run.State)
	}

	steps, err := db.Steps(id)
	if err != nil {
		t.Fatal(err)
	}
	if steps["soi"].State != store.StepFailed {
		t.Fatalf("buoc dang chay phai bi ha xuong failed, duoc %q", steps["soi"].State)
	}
	// Phai ghi RO vi sao buoc nay do. De trong thi lan sau doc lai se tuong no
	// hong vi code, roi di sua nham cho.
	if steps["soi"].Msg == "" {
		t.Fatal("buoc bi ha ma khong ghi ly do")
	}
	// Buoc DA XONG thi khong duoc dung toi — no da lam that va co ket qua that.
	if steps["code-doc"].State != store.StepDone {
		t.Fatalf("buoc da xong bi sua thanh %q", steps["code-doc"].State)
	}
}

// Luot da xong roi thi khong huy nguoc duoc — huy mot thu da hoan thanh la
// viet lai lich su.
func TestKhongHuyDuocLuotDaXong(t *testing.T) {
	r, _, db := newRunner(t)

	id, err := db.CreateRun("dem", t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetRunState(id, store.RunDone); err != nil {
		t.Fatal(err)
	}

	if err := r.Huy(id, "tester"); err == nil {
		t.Fatal("huy duoc mot luot da hoan thanh")
	}
}

func TestHuyLuotKhongCoThiBaoLoi(t *testing.T) {
	r, _, _ := newRunner(t)
	if err := r.Huy(9999, "tester"); err == nil {
		t.Fatal("huy duoc mot luot khong ton tai")
	}
}
