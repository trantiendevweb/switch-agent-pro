// Package aiapi là ĐƯỜNG THỨ HAI của dự án: gọi thẳng AI API, không qua CLI agent.
//
// Khác đường CLI ở bản chất: đường CLI mượn tài khoản đăng nhập sẵn và tiêu hạn
// mức của gói thuê bao; đường này dùng API key và tiêu tiền theo token. Vì vậy
// mọi lời gọi ở đây đều trả về `Usage` — không đo được thì không biết đang tiêu gì.
//
// Hai luật giữ từ MASTER-PLAN mục 0, không được lách:
//
//  1. FILE CẤU HÌNH KHÔNG BAO GIỜ CHỨA SECRET. Route chỉ ghi `key_id`; key thật
//     nằm ở ~/.ai-accounts/api-keys/<id>.key, trong kho đã siết ACL.
//  2. Lỗi của nhà cung cấp phải giữ NGUYÊN VĂN. Họ trả kèm request id — vứt nó đi
//     là vứt luôn thứ duy nhất dùng được khi phải hỏi lại họ.
package aiapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// Route là một đường gọi API đã cấu hình. KHÔNG chứa key.
type Route struct {
	Ten     string // tên route, ví dụ "grok"
	BaseURL string // ví dụ https://modelapi.vn/v1
	Model   string // ví dụ grok-4.5
	KeyID   string // tên file trong ~/.ai-accounts/api-keys, KHÔNG phải key
}

// Usage là phần đếm token. Nhà cung cấp nào cũng trả, và đây là thứ cho biết
// một lời gọi tốn bao nhiêu.
type Usage struct {
	Vao  int `json:"prompt_tokens"`
	Ra   int `json:"completion_tokens"`
	Tong int `json:"total_tokens"`
}

// KetQua là kết quả một lời gọi.
//
// Bốn trường cuối nói về CHUYỆN ĐÃ XẢY RA trên đường đi, không phải nội dung câu
// trả lời. Có chúng vì khi tầng trên tự chuyển sang route dự phòng, người gọi
// phải biết: câu này ai trả lời, route chính hỏng vì gì, và request id của lần
// hỏng đó là bao nhiêu. Trước đây những thứ đó chỉ được `bus.Warnf` — ai gọi qua
// web hay CLI mà không nghe bus thì mất sạch.
type KetQua struct {
	NoiDung string

	// SuyLuan là phần NGHĨ của model, tách khỏi câu trả lời.
	//
	// Rỗng có HAI nghĩa khác nhau và người đọc phải phân biệt được: model không
	// nghĩ, hoặc nhà cung cấp không trả phần nghĩ ra. Bảng `sagent nang-luc-api`
	// là chỗ trả lời câu đó cho từng route — đừng suy từ chuỗi rỗng này.
	SuyLuan string

	// ToolCalls là những lời gọi tool model ĐÒI chạy. Gói này KHÔNG chạy chúng
	// — nó chỉ chuyển về cho người gọi. Vì sao là một quyết định chứ không phải
	// một chỗ làm dở: xem đầu file tool.go.
	ToolCalls []LoiGoiTool

	Model string
	Usage Usage // usage của route THẬT SỰ trả lời, không phải của route hỏng
	Mat   time.Duration

	// DaStreaming: lượt này đi đường stream. ThieuUsage: nhà cung cấp KHÔNG trả
	// `usage`, nên Usage ở trên là số 0 vì CHƯA ĐO chứ không phải vì miễn phí.
	//
	// Phải có cờ riêng chứ không suy từ `Usage.Tong == 0`: một lượt thật cũng có
	// thể tốn 0 token nếu hỏng sớm, và gộp hai chuyện đó lại thì sổ chi phí mất
	// khả năng phân biệt "không tốn gì" với "không đếm được".
	DaStreaming bool
	ThieuUsage  bool

	// Route là tên route đã trả lời câu này.
	Route string
	// DaThu là tên mọi route đã gọi, theo thứ tự, kể cả route hỏng.
	DaThu []string
	// RouteChinh là route đã hỏng khiến phải chuyển. Rỗng nếu không chuyển.
	RouteChinh string
	// LoiChinh là lỗi NGUYÊN VĂN của route chính, kèm request id của nhà cung
	// cấp. Rỗng nếu không chuyển route.
	LoiChinh string

	// ChoLai là nhật ký mọi lần bị HTTP 429 rồi chờ và thử lại CHÍNH route này.
	// Rỗng ở lượt bình thường.
	//
	// Phải nằm trong KetQua chứ không chỉ trong lỗi, vì lần chờ hay gặp nhất là
	// lần chờ THÀNH CÔNG: gọi lần một bị chặn, chờ 2 giây, lần hai chạy. Lượt đó
	// trả về err=nil, và nếu tin này chỉ đi kèm lỗi thì nó biến mất — người dùng
	// thấy một lượt đột nhiên mất 15 giây thay vì 2,3 giây mà không có gì giải
	// thích. `Mat` đo được độ trễ nhưng không nói được vì sao.
	ChoLai []LanChoLai

	// CanhBaoTruocKhiGui là những câu SOÁT RA TRƯỚC khi chạm mạng: bảng năng
	// lực chưa đo route này, ảnh nhỏ hơn ngưỡng đã đo được, người dùng ép gửi
	// dù bảng nói không. Rỗng ở lượt bình thường.
	//
	// Nằm trong KetQua chứ không chỉ in ra ở CLI, vì đúng lý do của `ChoLai`:
	// mặt web gọi cùng một hàm và phải nói được cùng một câu. Và nó được gắn
	// vào KetQua KỂ CẢ khi lượt gọi hỏng — câu "ảnh chỉ có 64 điểm ảnh" chính
	// là thứ giải thích cái HTTP 400 vừa nhận về.
	CanhBaoTruocKhiGui []string
}

// DaChuyenRoute cho biết câu trả lời này đến từ route dự phòng.
func (k KetQua) DaChuyenRoute() bool { return k.RouteChinh != "" }

// LoiAPI là lỗi của một lời gọi, giữ đủ thứ cần để tầng trên QUYẾT ĐỊNH có nên
// chuyển route hay không — chứ không chỉ một chuỗi để in ra.
//
// `Chi` là nguyên văn (đã kèm tên route và mã HTTP), theo luật 2 ở đầu file.
type LoiAPI struct {
	Route  string // route đã gọi
	Status int    // mã HTTP; 0 khi hỏng trước lúc có phản hồi
	Chi    string // nguyên văn
	Nguoi  bool   // lỗi từ phía người dùng

	// ChoLai là nhật ký các lần chờ vì 429 trước khi bỏ cuộc. Rỗng nếu không có.
	ChoLai []LanChoLai
}

func (e *LoiAPI) Error() string { return e.Chi }

// BiChanTocDo cho biết lỗi này là HTTP 429 — nhà cung cấp chặn tốc độ, chứ không
// phải hỏng.
//
// Tách ra thành hàm chứ không để tầng trên tự so `Status == 429`: đó là kiểu
// kiến thức bị chép ra nhiều chỗ rồi lệch nhau, và bản lệch bao giờ cũng là bản
// coi 429 như một lỗi 4xx bình thường.
func BiChanTocDo(err error) bool {
	var l *LoiAPI
	return errors.As(err, &l) && l.Status == http.StatusTooManyRequests
}

// LoiNguoiDung cho biết lỗi này do phía NGƯỜI DÙNG: key sai hoặc hết quyền
// (401/403), không đọc được key, prompt rỗng.
//
// Vì sao phải phân biệt: chuyển sang route dự phòng chỉ lặp lại đúng cái sai đó
// ở nhà cung cấp thứ hai. Nó không cứu được gì, mà tốn thêm một lượt gọi, nhân
// đôi thời gian chờ, và làm lỗi trả về dài gấp đôi trong khi nguyên nhân thật
// vẫn là câu đầu tiên.
func LoiNguoiDung(err error) bool {
	var l *LoiAPI
	return errors.As(err, &l) && l.Nguoi
}

// loiNguoi/loiMay dựng LoiAPI cho hai nhánh.
func loiNguoi(route string, status int, dinh string, a ...any) *LoiAPI {
	return &LoiAPI{Route: route, Status: status, Chi: fmt.Sprintf(dinh, a...), Nguoi: true}
}
func loiMay(route string, status int, dinh string, a ...any) *LoiAPI {
	return &LoiAPI{Route: route, Status: status, Chi: fmt.Sprintf(dinh, a...)}
}

// KeysDir là nơi giữ API key — trong kho đã siết ACL, NGOÀI repo.
func KeysDir() string { return filepath.Join(paths.AccountsRoot(), "api-keys") }

func keyPath(id string) string { return filepath.Join(KeysDir(), id+".key") }

// docKey đọc key theo ID.
//
// Không nhận đường dẫn từ ngoài: `id` phải là tên trần, không có dấu phân cách.
// Tên đến từ file cấu hình mà người khác có thể sửa, và hàm này đọc file bí mật —
// đúng lớp lỗi đã nổ một lần với tên hồ sơ (xem docs/DO-LUONG.md).
func docKey(id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("route thiếu key_id")
	}
	if strings.ContainsAny(id, `/\:`) || id == "." || id == ".." {
		return "", fmt.Errorf("key_id %q không hợp lệ — chỉ dùng chữ, số, '-', '_'", id)
	}
	b, err := os.ReadFile(keyPath(id))
	if err != nil {
		return "", fmt.Errorf("không đọc được key %q: %w\n     đặt bằng: sagent api key %s", id, err, id)
	}
	k := strings.TrimSpace(string(b))
	if k == "" {
		return "", fmt.Errorf("key %q rỗng", id)
	}
	return k, nil
}

// DocKey là docKey cho gói khác dùng — cùng một kho key, cùng một luật tên.
//
// Có mặt để internal/plugin KHÔNG phải viết bản sao thứ hai của "secret nằm ở
// đâu và tên thế nào là hợp lệ". Hai bản sao của một luật an ninh là hai bản sẽ
// lệch nhau, và bản lệch bao giờ cũng là bản lỏng hơn.
func DocKey(id string) (string, error) { return docKey(id) }

type yeuCau struct {
	Model    string    `json:"model"`
	Messages []tinNhan `json:"messages"`

	// Tools là định nghĩa tool gửi kèm; ToolChoice là cách ép gọi.
	//
	// `omitempty` ở CẢ HAI, và đó là điều kiện để thêm hai trường này không đổi
	// gì với lượt gọi thường: `Goi` truyền nil nên thân JSON gửi đi giống hệt
	// trước. Bỏ `omitempty` thì mọi lượt bỗng mang `"tools":null` — và
	// deepseek-v4-flash đã từ chối những thứ nhỏ hơn thế (HTTP 400 chỉ vì
	// `tool_choice=required`, đo 22/08).
	Tools      []Tool `json:"tools,omitempty"`
	ToolChoice string `json:"tool_choice,omitempty"`

	// DangTraLoi là `response_format` — ép câu trả lời theo JSON schema.
	//
	// Con trỏ + `omitempty` vì đúng lý do của `Tools` ngay trên: lượt gọi
	// thường phải gửi thân JSON y HỆT như trước. Khai giá trị thường thì mọi
	// lượt bỗng mang `"response_format":{"type":""}` — và deepseek-v4-flash đã
	// trả HTTP 400 chỉ vì `tool_choice=required` (đo 22/08).
	//
	// Cách dùng và cách ĐỌC LẠI câu trả lời: cocautruc.go.
	DangTraLoi *DangTraLoi `json:"response_format,omitempty"`
}
type tinNhan struct {
	Role    string `json:"role"`
	Content string `json:"content"`

	// Phan là nội dung NHIỀU MẨU — đường duy nhất gửi ảnh đi được, vì giao thức
	// đòi `content` dạng mảng {type, text|image_url}.
	//
	// `json:"-"` vì trường này KHÔNG lên dây dưới tên đó: `MarshalJSON` bên
	// dưới ghép nó vào chính khoá `content`. Luật hợp nhất giữa `Content` và
	// `Phan` — cái nào thắng, cái nào bị vứt — khai ở đầu anh.go, và
	// `TestAnhDiHetDuongToiThanJSON` canh nó.
	Phan []PhanNoiDung `json:"-"`

	// SuyLuan là phần NGHĨ của model, nhà cung cấp trả tách khỏi câu trả lời.
	//
	// `omitempty` vì kiểu này dùng cho CẢ hai chiều: gửi đi thì không bao giờ
	// mang trường này, nhận về thì có thể có.
	//
	// Đo 22/08 (`sagent nang-luc-api --do`): deepseek-v4-flash TRẢ
	// `reasoning_content` dài 152 ký tự; grok-4.5 KHÔNG trả trường nào, dù lượt
	// đó tiêu 951 token và mất 20,3 giây — model CÓ nghĩ, nhà bán lại không trả
	// phần nghĩ ra. Nên trường này rỗng KHÔNG có nghĩa là model không suy luận.
	SuyLuan string `json:"reasoning_content,omitempty"`

	// ToolCalls là lời gọi tool model trả về. `omitempty` vì kiểu này dùng cho
	// CẢ hai chiều, y như `SuyLuan` ở trên: gửi đi thì không bao giờ mang, nhận
	// về thì có thể có.
	ToolCalls []LoiGoiTool `json:"tool_calls,omitempty"`
}

// MarshalJSON dựng thân của MỘT tin nhắn, và đây là chỗ luật hợp nhất
// `Content` + `Phan` được thi hành. Đọc ghi chú đầu anh.go trước khi sửa.
//
// Hai nhánh, và nhánh đầu quan trọng hơn nhánh sau: KHÔNG có phần nào thì thân
// JSON phải giống HỆT thời chưa có file anh.go. Mọi lượt gọi của cả dự án đi
// qua đây, và một khoá lạ mọc thêm là thứ deepseek-v4-flash đã từ chối vì
// những cớ nhỏ hơn thế.
func (t tinNhan) MarshalJSON() ([]byte, error) {
	// `tran` là kiểu TRẦN cùng bố cục nhưng KHÔNG mang phương thức này — thiếu
	// nó thì json.Marshal gọi lại chính hàm đang chạy và đệ quy tới hết stack.
	type tran tinNhan
	if len(t.Phan) == 0 {
		return json.Marshal(tran(t))
	}
	phan := t.Phan
	// `Content` KHÔNG bị vứt: nó vào làm mẩu `text` ĐẦU TIÊN. Đây là cả điểm
	// của luật hợp nhất — hai trường ghép lại chứ không tranh nhau, nên không
	// có ca nào người gọi mất chữ mà không được báo.
	if s := strings.TrimSpace(t.Content); s != "" {
		phan = append([]PhanNoiDung{PhanChu(s)}, phan...)
	}
	return json.Marshal(struct {
		Role      string        `json:"role"`
		Content   []PhanNoiDung `json:"content"`
		SuyLuan   string        `json:"reasoning_content,omitempty"`
		ToolCalls []LoiGoiTool  `json:"tool_calls,omitempty"`
	}{Role: t.Role, Content: phan, SuyLuan: t.SuyLuan, ToolCalls: t.ToolCalls})
}

type phanHoi struct {
	Model   string `json:"model"`
	Choices []struct {
		Message tinNhan `json:"message"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

// Goi gửi một prompt và trả về câu trả lời.
//
// Không streaming ở bản này — streaming là việc riêng, và làm nửa vời thì mất
// usage. Đo trước, thêm sau.
//
// Mọi lỗi trả về đều là *LoiAPI, để tầng trên phân biệt được "thử route khác có
// thể cứu" với "thử route khác chỉ tốn thêm tiền" — xem LoiNguoiDung.
func Goi(ctx context.Context, r Route, prompt string) (KetQua, error) {
	return goiThat(ctx, r, prompt, themVao{})
}

// themVao gom MỌI thứ có thể thêm vào một lượt gọi ngoài prompt.
//
// Là struct chứ không phải bốn tham số rời, và đó là bài học của chính file
// này: `goiThat` ra đời với ba tham số, lên năm khi có tool, và sẽ lên bảy khi
// có ảnh + schema. Một hàm bảy tham số mà bốn cái là zero value ở hầu hết chỗ
// gọi là chỗ người ta gõ nhầm thứ tự — mà gõ nhầm `tools` với `chonTool` thì
// không có lỗi biên dịch nào, chỉ có một yêu cầu sai gửi lên mạng và tính tiền.
type themVao struct {
	Tools      []Tool
	ChonTool   string
	Phan       []PhanNoiDung
	DangTraLoi *DangTraLoi
}

// goiThat là thân thật của `Goi`, có thêm chỗ nhận tool / ảnh / schema.
//
// Tách ra chứ không thêm tham số vào `Goi`: `Goi(ctx, r, prompt)` là chữ ký mà
// cả `internal/api`, CLI và mặt web đang gọi, và đổi nó chỉ để mang thêm mấy
// trường mà hầu hết chỗ gọi bỏ trống là bắt cả dự án trả giá cho một tính năng
// ít dùng. `GoiTool` (tool.go) và `GoiKem` (goikem.go) là cửa cho ai cần.
func goiThat(ctx context.Context, r Route, prompt string, them themVao) (KetQua, error) {
	var kq KetQua
	// Prompt rỗng chặn TRƯỚC khi chạm mạng: nhà cung cấp sẽ trả 400, và một lượt
	// gọi hỏng vẫn có thể bị tính tiền. Đây cũng là lỗi của người dùng, nên route
	// dự phòng không cứu được.
	if strings.TrimSpace(prompt) == "" {
		return kq, loiNguoi(r.Ten, 0, "prompt rỗng — không gửi gì cho %s", r.Ten)
	}
	key, err := docKey(r.KeyID)
	if err != nil {
		return kq, loiNguoi(r.Ten, 0, "%s: %s", r.Ten, err.Error())
	}
	// Route khai thiếu là lỗi CẤU HÌNH CỦA RIÊNG ROUTE ĐÓ — route dự phòng khai
	// đủ thì vẫn chạy được. Nên đây KHÔNG phải lỗi người dùng theo nghĩa chặn
	// fallback.
	if r.BaseURL == "" || r.Model == "" {
		return kq, loiMay(r.Ten, 0, "route %q thiếu base_url hoặc model", r.Ten)
	}

	body, err := json.Marshal(yeuCau{
		Model:      r.Model,
		Messages:   []tinNhan{{Role: "user", Content: prompt, Phan: them.Phan}},
		Tools:      them.Tools,
		ToolChoice: them.ChonTool,
		DangTraLoi: them.DangTraLoi,
	})
	if err != nil {
		return kq, loiMay(r.Ten, 0, "%s: %s", r.Ten, err.Error())
	}
	url := strings.TrimRight(r.BaseURL, "/") + "/chat/completions"
	// Dựng request MỚI mỗi lần thử: `bytes.NewReader(body)` đã đọc cạn sau lần
	// gửi đầu, dùng lại là gửi thân rỗng.
	taoReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)
		return req, nil
	}

	bat := time.Now()
	resp, choLai, err := goiCoChoLai(ctx, &http.Client{Timeout: 120 * time.Second}, taoReq)
	kq.ChoLai = choLai
	if err != nil {
		return kq, &LoiAPI{Route: r.Ten, Chi: fmt.Sprintf("gọi %s hỏng: %s%s",
			url, err.Error(), themChoLai(choLai)), ChoLai: choLai}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if resp.StatusCode != http.StatusOK {
		// GIỮ NGUYÊN VĂN thân lỗi. Nhà cung cấp trả kèm request id trong đó, và
		// đó là thứ duy nhất dùng được khi phải hỏi lại họ. Rút gọn thành "lỗi
		// 400" là vứt mất nó.
		//
		// Với 429 đã hết lượt thử lại, nguyên văn CÀNG quan trọng: thân 429 là
		// chỗ nhà cung cấp nói hạn mức nào bị chạm (phút hay ngày, token hay lời
		// gọi). Nuốt nó đi rồi in "bị chặn tốc độ" là biến một thông điệp hành
		// động được thành một lời than.
		e := loiMay(r.Ten, resp.StatusCode, "%s trả HTTP %d: %s%s",
			r.Ten, resp.StatusCode, strings.TrimSpace(string(raw)), themChoLai(choLai))
		e.ChoLai = choLai
		// 401/403 = key sai hoặc hết quyền. Route dự phòng dùng key KHÁC nhưng
		// cái sai ở đây là key của route NÀY: nếu người dùng gõ nhầm key thì họ
		// cần thấy đúng câu đó, không phải một câu ghép hai lỗi của hai nhà cung
		// cấp mà nguyên nhân thật bị chôn ở dòng đầu.
		//
		// 429 CỐ Ý không nằm ở đây: nó KHÔNG phải lỗi người dùng, nên tầng trên
		// vẫn được phép chuyển route sau khi ta đã chờ và thử lại mà vẫn hỏng.
		e.Nguoi = resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden
		return kq, e
	}
	var ph phanHoi
	if err := json.Unmarshal(raw, &ph); err != nil {
		return kq, loiMay(r.Ten, resp.StatusCode, "%s trả JSON không đọc được: %s", r.Ten, err.Error())
	}
	if len(ph.Choices) == 0 {
		return kq, loiMay(r.Ten, resp.StatusCode, "%s trả về 0 lựa chọn", r.Ten)
	}
	kq = KetQua{
		NoiDung: ph.Choices[0].Message.Content,
		SuyLuan: ph.Choices[0].Message.SuyLuan,
		// Lời gọi tool đi THẲNG ra KetQua, không bị nuốt: đây đúng là chỗ mà
		// `reasoning_content` đã bị vứt suốt từ lúc có kiểu tới chiều 22/08 —
		// trường có mặt ở mọi tầng, trừ tầng cuối cùng.
		ToolCalls: ph.Choices[0].Message.ToolCalls,
		Model:     ph.Model,
		Usage:     ph.Usage,
		Mat:       time.Since(bat),
		Route:     r.Ten,
		DaThu:     []string{r.Ten},
		ChoLai:    choLai,
	}
	return kq, nil
}

// themChoLai ghép nhật ký chờ vào cuối một thông điệp lỗi, có xuống dòng.
// Rỗng nếu không có lần chờ nào, để lỗi thường không dài ra vô cớ.
func themChoLai(nk []LanChoLai) string {
	if s := MoTaChoLai(nk); s != "" {
		return "\n     " + s
	}
	return ""
}
