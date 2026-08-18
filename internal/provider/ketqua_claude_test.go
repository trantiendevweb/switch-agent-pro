package provider

import "testing"

// Nguyen van dong cuoi do duoc ngay 18/08 tu:
//   claude -p "..." --output-format stream-json --verbose
// Da cat bot cac truong khong dung toi, GIU NGUYEN ten va kieu cua nhung truong
// ma la chan dua vao.
const ketQuaThatXong = `{"is_error":false,"duration_api_ms":4890,"num_turns":1,"stop_reason":"end_turn","session_id":"957255a6","total_cost_usd":0.08446,"usage":{"input_tokens":2,"output_tokens":4},"permission_denials":[],"terminal_reason":"completed","subtype":"success","api_error_status":null,"result":"OK","type":"result"}`

const ketQuaHongXacThuc = `{"is_error":true,"num_turns":0,"stop_reason":null,"total_cost_usd":0,"permission_denials":[],"terminal_reason":"error","subtype":"error_during_execution","api_error_status":"OAuth session expired","result":"","type":"result"}`

const ketQuaCutVong = `{"is_error":true,"num_turns":400,"stop_reason":"max_turns","total_cost_usd":1.2,"permission_denials":[],"terminal_reason":"error","subtype":"error_max_turns","api_error_status":null,"result":"","type":"result"}`

func doc(t *testing.T, raw string) KetQua {
	t.Helper()
	k, ok := docKetQuaClaude(raw)
	if !ok {
		t.Fatal("khong doc duoc dong result")
	}
	return k
}

// Luot chay THANH CONG phai doc ra cau tra loi that va khong bao hong.
func TestDocKetQuaXong(t *testing.T) {
	k := doc(t, `{"type":"system"}
`+ketQuaThatXong)
	if k.TraLoi != "OK" {
		t.Fatalf("cau tra loi = %q, muon %q", k.TraLoi, "OK")
	}
	if ly := k.Hong(); ly != "" {
		t.Fatalf("luot chay thanh cong ma bao hong: %s", ly)
	}
	if k.ChiPhiUSD == 0 || k.SoLuotTu != 1 {
		t.Fatalf("thieu so do: chi phi=%v luot=%d", k.ChiPhiUSD, k.SoLuotTu)
	}
}

// Hong xac thuc: bat bang TRUONG is_error + api_error_status, khong dua vao chu.
func TestDocKetQuaHongXacThuc(t *testing.T) {
	k := doc(t, ketQuaHongXacThuc)
	if k.Hong() == "" {
		t.Fatal("is_error=true ma khong bao hong")
	}
	if k.Loai != "error_during_execution" {
		t.Fatalf("loai = %q", k.Loai)
	}
}

// Cut vong: bat bang subtype/stop_reason, khong dua vao cau "maximum tool rounds".
func TestDocKetQuaCutVong(t *testing.T) {
	k := doc(t, ketQuaCutVong)
	if k.Hong() == "" {
		t.Fatal("error_max_turns ma khong bao hong")
	}
	if k.SoLuotTu != 400 {
		t.Fatalf("so luot = %d", k.SoLuotTu)
	}
}

// Ban ghi khong co dong result thi phai noi thang la KHONG DOC DUOC, dung doan.
func TestKhongCoDongResultThiNoiKhongDocDuoc(t *testing.T) {
	if _, ok := docKetQuaClaude("chi la chu tron, khong phai NDJSON"); ok {
		t.Fatal("khong co dong result ma van bao doc duoc")
	}
}
