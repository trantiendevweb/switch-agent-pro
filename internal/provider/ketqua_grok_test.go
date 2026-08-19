package provider

import (
	"strings"
	"testing"
)

// Nguyen van hinh dang do duoc o lan chay #29, buoc `soi` (token thay bang chu gia).
const grokMotLuot = `{"role":"user","content":"Ban la nguoi soi doc lap..."}
{"role":"assistant","content":"Using tools to help you...","tool_calls":[{"id":"call-1","type":"function","function":{"name":"bash","arguments":"{\"command\":\"git log --oneline main..sagent/tns-1\"}"}}]}
{"role":"tool","tool_call_id":"call-1","content":"Command executed successfully (no output)"}
{"role":"assistant","content":"Using tools to help you...","tool_calls":[{"id":"call-2","type":"function","function":{"name":"bash","arguments":"{\"command\":\"git diff main...sagent/may-1\"}"}}]}
{"role":"tool","tool_call_id":"call-2","content":"diff --git a/README.md ..."}
{"role":"assistant","content":"TNS: NEN TRON (khong co commit nao tren nhanh)\nMAY: CAN SUA THEM"}`

// Truoc ban va, grok.DocKetQua tra ve (KetQua{}, false), nen:
//   - la chan chong chay quan KHONG BAO GIO chay cho chinh ke da quan 399 lan
//   - output cua buoc `soi` giu nguyen khoi NDJSON tho, roi bi nhet ca vao prompt
//     cua buoc sau
func TestGrokDocDuocCauTraLoiThat(t *testing.T) {
	k, ok := docKetQuaGrok(grokMotLuot)
	if !ok {
		t.Fatal("khong doc duoc ban ghi grok")
	}
	if !strings.HasPrefix(k.TraLoi, "TNS:") {
		t.Fatalf("phai lay loi assistant CUOI CUNG khong kem tool_calls, duoc %q", k.TraLoi)
	}
	// Cau dan "Using tools to help you..." KHONG duoc coi la cau tra loi.
	if strings.Contains(k.TraLoi, "Using tools") {
		t.Fatalf("lay nham cau dan lam cau tra loi: %q", k.TraLoi)
	}
	if !k.DemDuocTool {
		t.Fatal("co tool_calls kem tham so ma bao la khong dem duoc")
	}
	if ly := k.Hong(); ly != "" {
		t.Fatalf("luot binh thuong ma bao hong: %s", ly)
	}
}

// CA DA NO THAT (lan chay #21): Grok goi dung mot lenh lien tiep rat nhieu lan.
func TestGrokBatDuocChayQuan(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"role":"user","content":"lam viec di"}` + "\n")
	for i := 0; i < 30; i++ {
		b.WriteString(`{"role":"assistant","content":"Using tools to help you...","tool_calls":[{"id":"c","type":"function","function":{"name":"bash","arguments":"{\"command\":\"ls -la\"}"}}]}` + "\n")
		b.WriteString(`{"role":"tool","tool_call_id":"c","content":"'ls' is not recognized"}` + "\n")
	}
	b.WriteString(`{"role":"assistant","content":"Toi da xem xong thu muc."}`)

	k, ok := docKetQuaGrok(b.String())
	if !ok {
		t.Fatal("khong doc duoc")
	}
	ly := k.Hong()
	if ly == "" {
		t.Fatal("lap 30 lan lien tiep ma khong bat duoc — la chan khong chay cho grok")
	}
	if !strings.Contains(ly, "ls -la") {
		t.Fatalf("phai goi ten LENH bi lap de nguoi doc biet sua gi, duoc %q", ly)
	}
	// Ca nguy hiem nhat: quan xong VAN nan ra duoc mot cau tra loi nghe hop ly.
	if k.TraLoi == "" {
		t.Fatal("gia lap sai: ca nay phai co cau tra loi cuoi")
	}
}

// Dung bat oan: goi cung mot TEN tool nhung THAM SO khac nhau la lam viec that.
func TestGrokKhongBatOanKhiThamSoKhacNhau(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 30; i++ {
		b.WriteString(`{"role":"assistant","tool_calls":[{"function":{"name":"bash","arguments":"{\"command\":\"cat file` +
			string(rune('a'+i%26)) + `.go\"}"}}]}` + "\n")
	}
	b.WriteString(`{"role":"assistant","content":"Da doc xong."}`)

	k, ok := docKetQuaGrok(b.String())
	if !ok {
		t.Fatal("khong doc duoc")
	}
	if ly := k.Hong(); ly != "" {
		t.Fatalf("tham so khac nhau la lam viec that, khong duoc bao quan: %s", ly)
	}
}

// Grok doi cach in thi phai noi KHONG BIET, dung doan.
func TestGrokDoiDinhDangThiNoiKhongBiet(t *testing.T) {
	if _, ok := docKetQuaGrok("Toi da chay xong viec.\nKhong co JSON nao o day."); ok {
		t.Fatal("ban ghi khong phai NDJSON ma van bao doc duoc")
	}
}

// La chan phai nam TREN DUONG CHAY THAT, khong chi la mot ham dung rieng.
//
// Cho hong nam o CHO NOI: `grok.DocKetQua` tra false thi moi thu tren deu vo
// nghia, ma test goi thang docKetQuaGrok van xanh. Day la ban dung dung adapter
// nhu internal/api goi no.
func TestAdapterGrokThatSuDungBoDoc(t *testing.T) {
	ad, co := Get("grok")
	if !co {
		t.Fatal("khong co provider grok")
	}
	var b strings.Builder
	for i := 0; i < 30; i++ {
		b.WriteString(`{"role":"assistant","tool_calls":[{"function":{"name":"bash","arguments":"{\"command\":\"ls -la\"}"}}]}` + "\n")
	}
	k, ok := ad.DocKetQua(b.String())
	if !ok {
		t.Fatal("adapter grok van tra false — la chan chong quan khong bao gio chay cho grok")
	}
	if ly := k.Hong(); !strings.Contains(ly, "ls -la") {
		t.Fatalf("adapter doc duoc nhung khong bat quan: %q", ly)
	}
}
