package provider

import "testing"

// Nguyen van do duoc ngay 18/08 tu `agy --output-format stream-json`.
const agyXong = `{"event":"result","result":{"conversation_id":"4c4a7e9f","status":"SUCCESS","response":"OK\n","duration_seconds":4.15,"num_turns":1,"usage":{"input_tokens":14639,"output_tokens":101,"total_tokens":14740}}}`

// CA BAY: bi chan quyen, tool run_command bao ERROR, nhung status VAN LA SUCCESS.
// Chi co response rong. Neu tin `status` thi ket luan nguoc hoan toan.
const agyBiChanQuyen = `{"event":"step_update","step_update":{"conversation_id":"d98632d4","step_index":3,"state":"ERROR","step_type":"tool","tool_name":"run_command","duration_seconds":0.02}}
{"event":"result","result":{"conversation_id":"d98632d4","status":"SUCCESS","response":"","duration_seconds":2.33,"num_turns":1,"usage":{"input_tokens":14658,"output_tokens":415,"total_tokens":15073}}}`

func TestAgyDocDuocLuotXong(t *testing.T) {
	k, ok := docKetQuaAntigravity(agyXong)
	if !ok {
		t.Fatal("khong doc duoc dong result cua agy")
	}
	if k.TraLoi != "OK" {
		t.Fatalf("cau tra loi = %q", k.TraLoi)
	}
	if ly := k.Hong(); ly != "" {
		t.Fatalf("luot xong ma bao hong: %s", ly)
	}
	if k.TokenVao != 14639 || k.TokenRa != 101 {
		t.Fatalf("doc sai token: %d/%d", k.TokenVao, k.TokenRa)
	}
}

// Day la ca quan trong nhat: status noi SUCCESS nhung thuc te khong lam duoc gi.
func TestAgyStatusSuccessNhungThucTeHong(t *testing.T) {
	k, ok := docKetQuaAntigravity(agyBiChanQuyen)
	if !ok {
		t.Fatal("khong doc duoc")
	}
	if k.Loai != "SUCCESS" {
		t.Fatalf("phai giu nguyen status goc de con doi chieu, duoc %q", k.Loai)
	}
	if k.ToolHong != 1 {
		t.Fatalf("phai dem 1 buoc tool loi, duoc %d", k.ToolHong)
	}
	if k.Hong() == "" {
		t.Fatal("agent khong tra loi gi va 1 tool loi ma van tinh la xong - " +
			"day la luc TIN VAO status:SUCCESS se ket luan nguoc")
	}
}

func TestAgyKhongCoDongResultThiNoiKhongDocDuoc(t *testing.T) {
	if _, ok := docKetQuaAntigravity("chi la chu tron"); ok {
		t.Fatal("khong co dong result ma van bao doc duoc")
	}
}
