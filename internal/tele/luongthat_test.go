package tele

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// Ba test dưới đây chạy FLOW THẬT (bộ thực thi thật, SQLite thật) chứ không tự
// bịa ra event rồi tự khen mình.
//
// Vì sao phải làm vậy: phần dễ sai nhất của tính năng này không nằm ở chỗ ghép
// chữ, mà ở chỗ event do internal/flow phát ra có MANG ĐỦ số lượt chạy, tên
// bước và tài khoản hay không. Test chỉ dựng event bằng tay sẽ xanh vĩnh viễn
// kể cả khi bộ thực thi phát ra một dòng chữ trơ trụi — đúng thứ nó đã phát
// trước lượt sửa này (Bus.Failuref).

// agentGia đóng vai bộ chạy agent, hỏng hay không là do test quyết.
type agentGia struct{ hong bool }

func (a agentGia) RunAgents(_ context.Context, _ string, _ string, _ string, _ int, _, _ bool) (flow.KetQuaAgent, error) {
	if a.hong {
		return flow.KetQuaAgent{}, errors.New("phiên chết sau 3 giây")
	}
	return flow.KetQuaAgent{Output: "xong việc"}, nil
}

func dungBoChay(t *testing.T, ag flow.AgentRunner) (*flow.Runner, *buuDien) {
	t.Helper()
	home := homeGia(t)
	bd := moBuuDien(t)
	ghiCauHinh(t, `{"token":"t","chat_id":"1","api":"`+bd.srv.URL+`"}`)

	db, err := store.OpenAt(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	bus := events.NewBus()
	t.Cleanup(bus.Close)
	return &flow.Runner{
		DB: db, Bus: bus, Agent: ag,
		// Bước không khai `profile` thì đây là tài khoản nó chạy bằng — và cũng
		// là câu trả lời duy nhất đúng cho "tài khoản nào" trong tin nhắn.
		DefaultProfile: "claude:phu",
	}, bd
}

// Bước agent hỏng giữa một lượt chạy thật: phải có tin, và tin phải nói đủ.
func TestLuongThat_BuocHongThiNhanDuThongTin(t *testing.T) {
	r, bd := dungBoChay(t, agentGia{hong: true})
	dung := Nghe(r.Bus)

	f := flow.Flow{Name: "ra-soat", Steps: []flow.Step{
		{ID: "doc-repo", Type: flow.TypeAgent, Prompt: "đọc repo"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunFailed {
		t.Fatalf("lượt chạy phải hỏng, được %s", res.State)
	}

	// Hai tin: bước hỏng, rồi lượt chạy hỏng.
	tin := bd.choTin(t, 2)
	dung()

	gop := strings.Join(tin, "\n")
	for _, phai := range []string{
		"BƯỚC HỎNG",
		"LƯỢT CHẠY HỎNG",
		"doc-repo",              // bước nào
		"claude:phu",            // tài khoản nào — chỉ có nếu Runner khai DefaultProfile
		"phiên chết sau 3 giây", // vì sao, nguyên văn lỗi
		`flow "ra-soat"`,        // flow nào
		"sagent flow runs",      // gõ gì tiếp
	} {
		if !strings.Contains(gop, phai) {
			t.Errorf("tin từ lượt chạy thật thiếu %q.\nĐã nhận:\n%s", phai, gop)
		}
	}
	// Số lượt chạy là thứ người dùng gõ lại vào lệnh — sai số này thì tin vô dụng.
	if !strings.Contains(gop, "#"+soLuot(res.RunID)) {
		t.Errorf("tin không nhắc số lượt chạy #%d:\n%s", res.RunID, gop)
	}
}

// Lượt chạy dừng chờ duyệt: đây là lúc máy đứng im đợi người, mà không ai biết.
func TestLuongThat_ChoDuyetThiNhanDuocLenhDuyet(t *testing.T) {
	r, bd := dungBoChay(t, agentGia{})
	dung := Nghe(r.Bus)

	f := flow.Flow{Name: "phat-hanh", Steps: []flow.Step{
		{ID: "chuan-bi", Type: flow.TypeAgent, Prompt: "dựng bản"},
		{ID: "duyet-merge", Type: flow.TypeApprove, Needs: []string{"chuan-bi"}, Message: "gộp vào main?"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunWaiting {
		t.Fatalf("phải dừng chờ duyệt, được %s", res.State)
	}

	tin := bd.choTin(t, 1)[0]
	dung()

	for _, phai := range []string{
		"CHỜ DUYỆT", "duyet-merge", "gộp vào main?",
		"sagent flow approve " + soLuot(res.RunID) + " duyet-merge",
		"sagent flow reject " + soLuot(res.RunID) + " duyet-merge",
	} {
		if !strings.Contains(tin, phai) {
			t.Errorf("tin chờ duyệt thiếu %q.\nTin:\n%s", phai, tin)
		}
	}
}

// Lượt chạy trót lọt: đúng MỘT tin, dù bên trong có bao nhiêu bước.
func TestLuongThat_XongThiChiMotTin(t *testing.T) {
	r, bd := dungBoChay(t, agentGia{})
	dung := Nghe(r.Bus)

	f := flow.Flow{Name: "ra-soat", Steps: []flow.Step{
		{ID: "a", Type: flow.TypeAgent, Prompt: "việc A"},
		{ID: "b", Type: flow.TypeAgent, Needs: []string{"a"}, Prompt: "việc B"},
		{ID: "c", Type: flow.TypeNotify, Needs: []string{"b"}, Message: "báo cáo"},
	}}
	res, err := r.Start(context.Background(), f, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.State != store.RunDone {
		t.Fatalf("lượt chạy phải xong, được %s", res.State)
	}

	tin := bd.choTin(t, 1)
	dung()
	if len(tin) != 1 {
		t.Fatalf("ba bước trót lọt chỉ được đúng 1 tin, đã nhắn %d:\n%s",
			len(tin), strings.Join(tin, "\n---\n"))
	}
	if !strings.Contains(tin[0], "XONG") || !strings.Contains(tin[0], "#"+soLuot(res.RunID)) {
		t.Fatalf("tin báo xong chưa đủ:\n%s", tin[0])
	}
}

func soLuot(n int64) string { return strconv.FormatInt(n, 10) }
