package flow

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// CA DA NO THAT — lan chay #29, flow `dem`.
//
// Buoc `kiem-cuoi` (may cham) HONG nen khong de lai output nao. Buoc `soi` khai
// prompt "May cham noi gi:\n{{steps.kiem-cuoi.output}}". Expand chi thay nhung
// khoa CO trong map, nen placeholder di thang vao prompt duoi dang chu song:
// nguoi soi khong he nhan duoc phan quyet cua may cham, nhung van phan nhu the
// co — va khong ai duoc bao. Ca loi hua "may cham quyet dinh, khong phai loi
// agent" boc hoi trong im lang.
//
// Test nay di QUA runner that (khong goi thang ExpandChay) vi cho hong nam o
// CHO GOI: doi ExpandChay thanh Expand trong do() la benh tai phat.
func TestPromptGuiDiKhongDuocMangPlaceholderConSot(t *testing.T) {
	shellFail := []string{"cmd", "/c", "exit 1"}
	if runtime.GOOS != "windows" {
		shellFail = []string{"sh", "-c", "exit 1"}
	}

	r, ag, _ := newRunner(t)
	f := Flow{Name: "dem", Steps: []Step{
		{ID: "kiem-cuoi", Type: TypeShell, Run: shellFail, OnFailure: OnFailContinue},
		{ID: "soi", Type: TypeAgent, Needs: []string{"kiem-cuoi"},
			Prompt: "May cham noi gi:\n{{steps.kiem-cuoi.output}}\n\nTra loi dung khuon."},
	}}
	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}

	ps := ag.cacPrompt()
	if len(ps) != 1 {
		t.Fatalf("buoc soi phai duoc goi dung 1 lan, duoc %d", len(ps))
	}
	if strings.Contains(ps[0], "{{") {
		t.Fatalf("placeholder chua thay lot vao prompt gui cho agent:\n%s", ps[0])
	}
	if !strings.Contains(ps[0], "kiem-cuoi") {
		t.Fatalf("phai noi ro buoc nao khong de lai ket qua:\n%s", ps[0])
	}
}

// Co ket qua that thi phai thay binh thuong — dung chot nham thanh "khong co".
func TestCoKetQuaThatThiVanThayNhuCu(t *testing.T) {
	echo := []string{"cmd", "/c", "echo ok-that"}
	if runtime.GOOS != "windows" {
		echo = []string{"sh", "-c", "echo ok-that"}
	}

	r, ag, _ := newRunner(t)
	f := Flow{Name: "dem", Steps: []Step{
		{ID: "kiem", Type: TypeShell, Run: echo},
		{ID: "soi", Type: TypeAgent, Needs: []string{"kiem"}, Prompt: "May cham: {{steps.kiem.output}}"},
	}}
	if _, err := r.Start(context.Background(), f, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}

	ps := ag.cacPrompt()
	if len(ps) != 1 || !strings.Contains(ps[0], "ok-that") {
		t.Fatalf("khong thay ket qua that vao prompt: %v", ps)
	}
	if strings.Contains(ps[0], "khong de lai") || strings.Contains(ps[0], "không để lại") {
		t.Fatalf("co ket qua that ma van bao la khong co:\n%s", ps[0])
	}
}

// Buoc shell thi KHONG duoc chot nhu prompt: `go test -C (buoc "x" khong de lai
// ket qua)` la mot duong dan bia — no se hong bang mot thong bao chang lien quan
// gi toi nguyen nhan that. Phai dung ngay va goi ten buoc con thieu.
func TestShellThieuKetQuaThiDungVaGoiTenBuoc(t *testing.T) {
	shellFail := []string{"cmd", "/c", "exit 1"}
	if runtime.GOOS != "windows" {
		shellFail = []string{"sh", "-c", "exit 1"}
	}

	r, _, db := newRunner(t)
	f := Flow{Name: "dem", Steps: []Step{
		{ID: "kiem-1", Type: TypeShell, Run: shellFail, OnFailure: OnFailContinue},
		{ID: "kiem-2", Type: TypeShell, Needs: []string{"kiem-1"},
			Run: []string{"go", "test", "-C", "{{steps.kiem-1.output}}", "./..."}},
	}}
	res, _ := r.Start(context.Background(), f, t.TempDir(), nil)
	if res.State != store.RunFailed {
		t.Fatalf("thieu gia tri ma van chay tiep, trang thai = %s", res.State)
	}

	steps, err := db.Steps(res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	msg := steps["kiem-2"].Msg
	if !strings.Contains(msg, "kiem-1") {
		t.Fatalf("phai goi ten buoc con thieu trong thong bao, duoc %q", msg)
	}
	// Khong duoc chi kiem "co chu kiem-1": khong co chot thi lenh VAN CHAY voi
	// duong dan bia, va `go` cung nhac lai nguyen van `{{steps.kiem-1.output}}`
	// trong thong bao loi cua no — test se xanh vi nham ly do. Con dau ngoac nhon
	// nghia la placeholder da bi day thang ra he dieu hanh.
	if strings.Contains(msg, "{{") {
		t.Fatalf("placeholder bi day thang ra lenh thay vi dung lai: %q", msg)
	}
}

// Expand (ban goc) phai giu nguyen hanh vi: `sagent flow show` dung no de in thu
// prompt luc CHUA chay, khi do chua buoc nao co ket qua la chuyen binh thuong.
// Chot o do la noi doi theo chieu nguoc lai.
func TestExpandGocKhongTuChot(t *testing.T) {
	got := Expand("{{steps.kiem-cuoi.output}}", map[string]string{})
	if got != "{{steps.kiem-cuoi.output}}" {
		t.Fatalf("Expand khong duoc tu chot, no con dung cho `flow show`: %q", got)
	}
}
