// IDEMPOTENCY — chạy lại một lượt thì KHÔNG làm lại việc đã làm xong.
//
// ============================================================================
// "THẾ NÀO LÀ ĐÃ LÀM RỒI" — định nghĩa, và hậu quả của mọi cách định nghĩa khác
// ============================================================================
//
// Đây là câu hỏi duy nhất của cả mảnh này. Ba câu trả lời khả dĩ:
//
//	(a) Khoá tính từ ID BƯỚC.
//	    Hậu quả: sửa prompt xong chạy lại thì bước ĐÃ ĐỔI bị bỏ qua, và lượt
//	    chạy dùng lại kết quả của câu hỏi CŨ. Đây là kiểu hỏng tệ nhất trong cả
//	    ba: người ta sửa đúng cái mình muốn sửa, chạy lại, và nhận về y nguyên
//	    kết quả cũ mà không một dòng nào nói vì sao. KHÔNG CHỌN.
//
//	(b) Khoá tính từ PROMPT.
//	    Hậu quả: đổi một dấu phẩy là mất sạch cache. Nghe như một khuyết điểm,
//	    nhưng nó hỏng về phía AN TOÀN — làm lại một việc đã làm chỉ tốn tiền,
//	    còn bỏ qua một việc chưa làm thì cho ra kết quả sai. Vấn đề THẬT của (b)
//	    nằm chỗ khác: prompt KHÔNG phải toàn bộ đầu vào. Bước đọc
//	    `{{steps.x.output}}` mà x đổi kết quả thì prompt SAU KHI THAY BIẾN cũng
//	    đổi — nhưng chỉ khi ta băm prompt ĐÃ THAY BIẾN, chứ băm mẫu trong
//	    flows.toml thì không.
//
//	(c) Khoá do người dùng khai (`idempotency_key = "build-{{version}}"`).
//	    Hậu quả: người dùng phải tự biết cái gì quyết định kết quả bước. Khai
//	    thiếu một biến là quay về đúng lỗi của (a), chỉ khác là lần này do họ tự
//	    gây ra và không có gì cảnh báo.
//
// CHỌN (b) MỞ RỘNG: khoá = băm của TOÀN BỘ thứ quyết định kết quả bước, sau khi
// đã thay hết biến — id, loại, câu lệnh/prompt đã thay biến, tài khoản, model,
// route, số bản sao, tham số plugin, hợp đồng `phai_co`, và danh sách artifact.
// Đổi bất cứ thứ nào trong đó thì khoá đổi và bước CHẠY LẠI.
//
// Vì đã thay biến nên khoá cuốn theo cả kết quả của các bước trước: bước trước
// ra kết quả khác thì prompt của bước sau khác, khoá khác, chạy lại. Đây là thứ
// (c) không tự có được.
//
// ============================================================================
// THỨ KHOÁ KHÔNG NHÌN THẤY — và vì sao đây là OPT-IN
// ============================================================================
//
// Bộ chạy nhìn thấy prompt, biến, và kết quả các bước trước. Nó KHÔNG nhìn thấy:
// cây mã trên đĩa, HEAD của git, đồng hồ, mạng, và mọi thứ agent tự đi đọc.
//
// Nên `run = ["go", "test", "./..."]` có khoá KHÔNG ĐỔI khi mã nguồn đổi. Bật
// idempotency cho một bước như thế là tự bảo với mình rằng test vẫn xanh vì hôm
// qua nó xanh.
//
// Vì vậy: TẮT MẶC ĐỊNH, bật từng bước một bằng `idempotent = true`, và Validate
// CẢNH BÁO thẳng vào mặt ở các bước shell/test/lint. Không tự đoán hộ ai.
//
// ============================================================================
// PHẠM VI: giữa các LƯỢT CHẠY, không phải trong một lượt
// ============================================================================
//
// Trong MỘT lượt chạy, "bước đã xong thì đừng làm lại" đã có sẵn từ trước —
// trạng thái nằm ở SQLite và Resume bỏ qua bước `done`. Mảnh này lo chuyện khác:
// lượt chạy #52 bỏ qua việc mà lượt #51 đã làm xong.
package flow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// KhoaIdem dựng khoá "việc này đã làm rồi hay chưa" cho một bước.
//
// Trả về "" nếu bước không bật idempotency — chỗ gọi coi rỗng là "không tra sổ,
// không ghi sổ", nên tắt cờ là quay về đúng hành vi cũ, không một byte nào đổi.
//
// `env` phải là env ĐÃ THAY BIẾN mà bước sắp chạy với — chính nó, không phải một
// bản dựng lại. Dựng lại là mở đường cho hai chỗ lệch nhau, và lệch ở đây nghĩa
// là bỏ qua nhầm việc.
func KhoaIdem(s Step, env map[string]string) string {
	if !s.Idempotent {
		return ""
	}
	h := sha256.New()
	ghi := func(nhan, v string) {
		// Có nhãn và có độ dài: không thì hai trường nối lại thành một chuỗi
		// giống hệt một cặp trường khác ("ab"+"c" == "a"+"bc") và hai bước khác
		// nhau ra cùng một khoá.
		fmt.Fprintf(h, "%s\x00%d\x00%s\x00", nhan, len(v), v)
	}

	ghi("id", s.ID)
	ghi("type", s.Type)
	// cauHoi() là ĐÚNG thứ bước gửi đi sau khi thay hết biến — prompt cho agent,
	// dòng lệnh cho shell, đầu vào cho plugin. Dùng lại nó thay vì băm từng
	// trường thô để khoá không thể lệch khỏi thứ thật sự được thực thi.
	ghi("viec", cauHoi(s, envChoKhoa(env)))
	ghi("profile", s.Profile)
	ghi("model", s.Model)
	ghi("route", s.Route)
	ghi("copies", fmt.Sprint(s.Copies))
	ghi("worktree", fmt.Sprint(s.Worktree))
	ghi("tu_duyet", fmt.Sprint(s.TuDuyetQuyen))
	ghi("plugin", s.Plugin)
	for _, k := range sapXepKhoa(s.ThamSo) {
		ghi("tham_so."+k, s.ThamSo[k])
	}
	// `phai_co` là một phần của việc: nới hợp đồng đầu ra rồi chạy lại thì kết
	// quả cũ có thể không còn đạt, và ngược lại.
	for _, v := range s.PhaiCo {
		ghi("phai_co", v)
	}
	for _, ten := range tenArtifactSapXep(s) {
		ghi("artifact."+ten, s.Artifact[ten])
	}
	// 128 bit là quá đủ để không đụng nhau trong một cái sổ vài nghìn dòng, và
	// ngắn thì đọc bằng mắt trong sổ vẫn được.
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// envChoKhoa thay ĐƯỜNG DẪN artifact bằng BĂM NỘI DUNG của file.
//
// VÌ SAO BẮT BUỘC: đường dẫn artifact chứa số lượt chạy
// (`artifacts/run-51/viet/bao-cao.md`), nên nó KHÁC NHAU ở mọi lượt chạy. Băm
// thẳng đường dẫn thì bước nào đọc artifact cũng có khoá mới mỗi lượt, và cache
// không bao giờ trúng — tính năng có mặt nhưng không làm gì, kiểu hỏng khó thấy
// nhất vì không có gì báo lỗi.
//
// Băm NỘI DUNG thì đúng ngữ nghĩa: cùng đầu vào = cùng khoá, đổi một byte trong
// file = khoá đổi = chạy lại.
//
// File không đọc được thì đưa vào khoá đúng chuyện đó, kèm đường dẫn: không đọc
// được là một trạng thái đầu vào khác hẳn với đọc được, và trộn hai thứ đó vào
// một khoá là mời gọi trúng cache nhầm.
func envChoKhoa(env map[string]string) map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		// `artifact_dir` cũng chứa số lượt chạy, và nó KHÔNG phải một phần danh
		// tính của việc: nó là chỗ ĐỔ KẾT QUẢ, không phải đầu vào. Hai lượt cùng
		// ghi "file này vào thư mục của chính tôi" là cùng một việc.
		//
		// ĐO ĐƯỢC: thiếu dòng này thì mọi bước có khai `artifact` đều đổi khoá ở
		// mỗi lượt chạy và cache KHÔNG BAO GIỜ trúng — tính năng nằm đó, chạy
		// đúng, và không tiết kiệm được gì. Bắt được ở
		// TestIdempotentChepArtifactSangLuotMoi.
		if k == KhoaArtifactDir {
			out[k] = "<thu-muc-artifact-cua-buoc>"
			continue
		}
		if !strings.HasPrefix(k, "artifacts.") {
			out[k] = v
			continue
		}
		if b, err := bamFile(v); err == nil {
			out[k] = "sha256:" + b
		} else {
			out[k] = "khong-doc-duoc:" + v
		}
	}
	return out
}

func bamFile(p string) (string, error) {
	fh, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer fh.Close()
	h := sha256.New()
	if _, err := io.Copy(h, fh); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:32], nil
}

// KetQuaCu là một việc đã làm xong ở lượt chạy trước, dùng lại được.
type KetQuaCu struct {
	RunID  int64
	StepID string
	Output string
}

// chepArtifactCu chép artifact của bước cũ sang thư mục của lượt chạy này.
//
// VÌ SAO CHÉP CHỨ KHÔNG TRỎ THẲNG SANG THƯ MỤC CŨ: DonArtifact xoá thư mục của
// lượt chạy đã kết thúc sau bảy ngày. Trỏ thẳng thì một bước đang ở trạng thái
// `done` của lượt HÔM NAY có thể mất file vào tuần sau, khi cái lượt sinh ra nó
// bị dọn. Trạng thái `done` phải tự đứng được, không treo vào tuổi thọ của một
// lượt chạy khác.
//
// Thiếu bất cứ file nào thì trả lỗi — và chỗ gọi coi đó là KHÔNG TRÚNG CACHE,
// tức là chạy lại thật. Chép nửa vời rồi báo done là cách chắc chắn nhất để bước
// sau đọc một artifact rỗng mà tưởng là kết quả.
func chepArtifactCu(cu KetQuaCu, runID int64, s Step) error {
	if len(s.Artifact) == 0 {
		return nil
	}
	if _, err := ChuanBiArtifact(runID, s); err != nil {
		return err
	}
	for _, ten := range tenArtifactSapXep(s) {
		rel := s.Artifact[ten]
		tu := DuongDanArtifact(cu.RunID, cu.StepID, rel)
		den := DuongDanArtifact(runID, s.ID, rel)
		if err := chepFile(tu, den); err != nil {
			return fmt.Errorf("artifact %q của lượt #%d không còn: %w", ten, cu.RunID, err)
		}
	}
	return nil
}

func chepFile(tu, den string) error {
	st, err := os.Stat(tu)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return fmt.Errorf("%s là thư mục", tu)
	}
	if err := os.MkdirAll(filepath.Dir(den), 0o755); err != nil {
		return err
	}
	src, err := os.Open(tu)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(den)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

// VanDeIdempotent soi phần `idempotent` của cả flow.
//
// LỖI chỉ ở những chỗ chắc chắn hỏng; còn lại là CẢNH BÁO, vì "cái gì quyết định
// kết quả bước này" là câu chỉ người viết flow trả lời được.
func VanDeIdempotent(f Flow) []Problem {
	var ps []Problem
	loi := func(step, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: step, Msg: msg}) }
	canh := func(step, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: step, Msg: msg, Warn: true}) }

	for _, s := range f.Steps {
		if !s.Idempotent {
			continue
		}
		switch s.Type {
		case TypeApprove:
			loi(s.ID, "bước approve không bao giờ được bộ chạy thực thi nên không có gì để bỏ qua "+
				"— và một cái rào duyệt tự bỏ qua vì \"lần trước đã duyệt rồi\" thì không còn là rào")
		case TypeNotify:
			loi(s.ID, "bước notify chỉ để báo cho người đọc — bỏ qua nó nghĩa là im lặng đúng lúc "+
				"cần lên tiếng, mà chẳng tiết kiệm được gì")
		case TypeShell, TypeTest, TypeLint:
			canh(s.ID, "bộ chạy KHÔNG nhìn thấy cây mã trên đĩa, HEAD của git hay đồng hồ, nên khoá "+
				"của bước này không đổi khi những thứ đó đổi. Bước sẽ bị bỏ qua kể cả sau khi mã nguồn "+
				"đã khác. Chỉ bật nếu kết quả bước phụ thuộc HOÀN TOÀN vào dòng lệnh và biến")
		}
		if s.ForEach != "" {
			loi(s.ID, "chưa dùng `idempotent` chung với `foreach` được: khoá phải tính trên TỪNG lượt lặp "+
				"mới đúng, còn bỏ qua cả bước theo một khoá chung sẽ bỏ qua cả những mục MỚI trong danh sách")
		}
	}
	return ps
}

// MoTaIdem là câu mô tả ngắn cho sổ và cho mặt web, để mọi nơi in giống nhau.
func MoTaIdem(cu KetQuaCu) string {
	return fmt.Sprintf("bỏ qua — việc này lượt chạy #%d đã làm xong (idempotent)", cu.RunID)
}
