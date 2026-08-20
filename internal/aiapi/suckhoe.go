package aiapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// SucKhoe là kết quả kiểm một route: nó còn sống không, và model khai trong cấu
// hình có thật không.
//
// Hai câu hỏi đi cùng nhau CÓ CHỦ Ý. Route sống mà model khai sai thì mọi lượt
// gọi vẫn hỏng — mà hỏng ở tận lúc chạy, kèm một thông điệp của nhà cung cấp
// chẳng nhắc gì tới cấu hình. Đã dính đúng lỗi đó: `.sagent/project.toml` từng
// khai `deepseek-chat`, một tên KHÔNG tồn tại ở nhà bán lại đang dùng.
type SucKhoe struct {
	Ten     string        `json:"ten"`
	Song    bool          `json:"song"`
	Status  int           `json:"status"`             // mã HTTP, 0 nếu không chạm được mạng
	Mat     time.Duration `json:"-"`                  // thời gian trả lời
	Model   string        `json:"model"`              // model route khai
	CoModel bool          `json:"coModel"`            // model đó có trong danh sách của nhà cung cấp
	SoModel int           `json:"soModel"`            // nhà cung cấp liệt kê bao nhiêu model
	Gan     []string      `json:"gan,omitempty"`      // vài tên gần giống, để sửa cấu hình cho nhanh
	Loi     string        `json:"loi,omitempty"`      // nguyên văn, giữ cả request id
	KhongRo bool          `json:"khongRo,omitempty"`  // route sống nhưng không liệt kê được model
}

// Dung cho biết route này có dùng được không: sống VÀ model khai có thật.
//
// Tách khỏi `Song` vì hai thứ khác nhau: `Song` trả lời "nhà cung cấp còn đó",
// còn `Dung` trả lời "gọi route này bây giờ thì chạy". Trộn hai câu vào một cờ
// là cách làm mất đúng cái thông tin đáng giá.
func (s SucKhoe) Dung() bool { return s.Song && (s.CoModel || s.KhongRo) }

type dsModel struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// Kiem hỏi `GET {base_url}/models` — KHÔNG tốn token.
//
// Vì sao không gửi một prompt tí hon cho chắc: health check là thứ chạy thường
// xuyên, và một phép kiểm có tính tiền thì người ta sẽ thôi chạy nó. `/models`
// đi qua đúng base_url và đúng key, nên nó vẫn trả lời được ba câu quan trọng
// nhất: mạng có tới không, key có được nhận không, model khai có thật không.
//
// Thứ nó KHÔNG trả lời được: hạn mức còn hay hết. Cái đó chỉ lộ ra khi gọi thật,
// và đó là lý do hàm này trả về `SucKhoe` chứ không phải một chữ "ổn".
func Kiem(ctx context.Context, r Route) SucKhoe {
	sk := SucKhoe{Ten: r.Ten, Model: r.Model}

	key, err := docKey(r.KeyID)
	if err != nil {
		sk.Loi = err.Error()
		return sk
	}
	if r.BaseURL == "" {
		sk.Loi = "route thiếu base_url"
		return sk
	}

	url := strings.TrimRight(r.BaseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		sk.Loi = err.Error()
		return sk
	}
	req.Header.Set("Authorization", "Bearer "+key)

	// Timeout NGẮN hơn hẳn `Goi` (120s): đây là phép kiểm để quyết định có chạy
	// hay không, nên nó phải trả lời nhanh. Một route mất 30 giây mới nói được
	// "tôi còn sống" thì thà coi như đang chập chờn.
	bat := time.Now()
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	sk.Mat = time.Since(bat)
	if err != nil {
		sk.Loi = err.Error()
		return sk
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	sk.Status = resp.StatusCode

	if resp.StatusCode != http.StatusOK {
		// Giữ NGUYÊN VĂN như `Goi`: request id của nhà cung cấp nằm trong đó.
		sk.Loi = strings.TrimSpace(string(raw))
		return sk
	}
	sk.Song = true

	var ds dsModel
	if json.Unmarshal(raw, &ds) != nil || len(ds.Data) == 0 {
		// Nhà cung cấp sống nhưng không nói được model nào — có thật, một số
		// endpoint tương thích OpenAI không cài `/models`. KHÔNG được kết luận
		// "model khai sai" từ chỗ này: im lặng khác với phủ nhận.
		sk.KhongRo = true
		return sk
	}
	sk.SoModel = len(ds.Data)

	ten := make([]string, 0, len(ds.Data))
	for _, m := range ds.Data {
		ten = append(ten, m.ID)
		if m.ID == r.Model {
			sk.CoModel = true
		}
	}
	if !sk.CoModel {
		sk.Gan = tenGan(r.Model, ten)
	}
	return sk
}

// tenGan gợi ý vài model có tên gần giống, để người sửa cấu hình khỏi phải đi
// tra danh sách. Chỉ so tiền tố — đủ để bắt `deepseek-chat` → `deepseek-v4-*`,
// mà không cần kéo thêm thư viện tính khoảng cách chuỗi.
func tenGan(muon string, co []string) []string {
	goc := muon
	if i := strings.IndexAny(goc, "-:"); i > 0 {
		goc = goc[:i]
	}
	var ra []string
	for _, t := range co {
		if strings.HasPrefix(t, goc) {
			ra = append(ra, t)
		}
	}
	sort.Strings(ra)
	if len(ra) > 5 {
		ra = ra[:5]
	}
	return ra
}
