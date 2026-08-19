package dash

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
	"github.com/trantiendevweb/switch-agent-pro/internal/flow"
)

// doc goi mot endpoint da dang nhap va tra ve JSON.
func doc(t *testing.T, s *Server, ck *http.Cookie, duong string) map[string]any {
	t.Helper()
	r := httptest.NewRequest("GET", duong, nil)
	r.Host = "127.0.0.1:4600"
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("%s -> %d: %s", duong, w.Code, w.Body.String())
	}
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("%s tra ve khong phai JSON: %v\n%s", duong, err, w.Body.String())
	}
	return m
}

// /api/state PHAI tra ve runs.
//
// CA DA NO THAT: trong ban do "ngang quyen" (lachan_test.go), hanh dong
// `flow.runs` khai duong vao tu web CHINH LA /api/state. Test ngang quyen chi
// kiem duong dan khong 404 — no xanh, trong khi endpoint KHONG he tra runs.
// Mat web vi the khong co cach nao biet luot chay nao dang chay.
func TestStatePhaiTraVeDanhSachLuotChay(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	m := doc(t, s, ck, "/api/state")
	if _, co := m["runs"]; !co {
		t.Fatalf("/api/state thieu khoa \"runs\" — mat web khong biet luot nao dang chay.\nCo: %v", khoa(m))
	}
}

// Chuoi day du: runner luu CAU HOI -> store -> API -> mat web doc duoc ca hoi
// lan dap. Truoc day chi luu cau tra loi, nen nhin lai mot luot chay thi thay
// agent noi gi ma khong thay no DUOC HOI GI — trong khi phan lon loi cua luot
// #29 nam dung o cau hoi.
func TestChiTietLuotChayCoCaCauHoiVaCauTraLoi(t *testing.T) {
	s := newTestServer(t)
	ck := dangNhap(t, s, "127.0.0.1:4600")

	dir := t.TempDir()
	f := flow.Flow{Name: "thu", Vars: map[string]string{"viec": "ra soat repo"}, Steps: []flow.Step{
		{ID: "hoi-mot-cau", Type: flow.TypeAgent, Profile: "claude:tns", Prompt: "Viec can lam: {{viec}}"},
	}}
	if _, err := s.api.FlowSave(dir, f); err != nil {
		t.Fatal(err)
	}
	// Chay that qua runner (co agent gia) de chuoi luu tru duoc di het.
	res, err := s.api.FlowRunCuChay(context.Background(), dir, "thu", nil,
		api.ParseAddr("claude:tns"), true)
	if err != nil {
		t.Fatal(err)
	}

	m := doc(t, s, ck, "/api/flow/detail?id="+strconv.FormatInt(res.RunID, 10))
	steps, _ := m["steps"].([]any)
	if len(steps) != 1 {
		t.Fatalf("phai co dung 1 buoc, duoc %d", len(steps))
	}
	b, _ := steps[0].(map[string]any)

	prompt, _ := b["prompt"].(string)
	if prompt == "" {
		t.Fatal("buoc khong luu lai cau hoi — man hoi thoai se chi co mot nua")
	}
	// Phai la cau hoi DA THAY BIEN, khong phai mau trong flows.toml.
	if strings.Contains(prompt, "{{") {
		t.Fatalf("luu mau chua thay bien thay vi cau hoi that: %q", prompt)
	}
	if !strings.Contains(prompt, "ra soat repo") {
		t.Fatalf("cau hoi khong mang gia tri that cua bien: %q", prompt)
	}
	// Cac truong man hoi thoai dua vao de ve.
	for _, k := range []string{"id", "type", "profile", "needs", "state", "output"} {
		if _, co := b[k]; !co {
			t.Errorf("buoc thieu truong %q — man hoi thoai can no de ve", k)
		}
	}
	if b["profile"] != "claude:tns" {
		t.Errorf("mat nguoi noi: profile = %v", b["profile"])
	}
}

func khoa(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
