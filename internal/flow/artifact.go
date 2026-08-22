// ARTIFACT — bước để lại một FILE cho bước sau, không chỉ để lại một chuỗi.
//
// VÌ SAO CÓ, và vì sao cái đã có KHÔNG đủ:
//
// Đường truyền duy nhất giữa hai bước cho tới nay là `{{steps.x.output}}`, và nó
// đi qua hai cái trần. `store.MaxStepOutput` (32 KiB) cắt lúc lưu, rồi
// `flow.MaxInject` (6.000 ký tự) cắt lần nữa lúc nhét sang bước sau. Cả hai đều
// cắt phần ĐẦU và giữ phần CUỐI — chọn đúng cho một bản tóm tắt, nhưng sai hẳn
// cho một bản vá, một file JSON, hay một báo cáo có mục lục ở đầu. Bước sau nhận
// một mảnh và không có cách nào biết mình đang đọc một mảnh.
//
// Artifact đi vòng qua cả hai: bước trước GHI RA FILE, bước sau nhận ĐƯỜNG DẪN
// tới file đó. Không byte nào bị cắt, vì không byte nào đi qua prompt.
//
// BỐN QUYẾT ĐỊNH, cả bốn đều cố ý:
//
//  1. Bước sau trỏ tới artifact bằng TÊN (`{{artifacts.bao-cao}}`), không bằng
//     đường dẫn. flows.toml là file người ta gửi cho nhau — một đường dẫn tuyệt
//     đối trong đó là rác trên máy người nhận. Đường dẫn thật do bộ chạy dựng
//     lúc chạy, từ số lượt chạy và id bước.
//
//  2. Mỗi lượt chạy một thư mục riêng (`artifacts/run-<id>/<bước>/`). Số lượt
//     chạy do SQLite cấp, không trùng nhau, nên HAI LƯỢT CHẠY SONG SONG KHÔNG
//     GIẪM LÊN NHAU — kể cả khi chạy cùng một flow trên cùng một thư mục.
//
//  3. Khai artifact là một HỢP ĐỒNG, cùng loại với `phai_co`: chạy xong mà file
//     không có thì bước HỎNG. Không có luật này thì bước sau nhận một đường dẫn
//     trông rất thật trỏ vào hư không, và hỏng ở một chỗ chẳng liên quan gì tới
//     nguyên nhân.
//
//  4. Xoá sạch thư mục artifact của bước TRƯỚC MỖI LẦN THỬ. Không xoá thì lần
//     thử 2 hỏng vẫn "đủ file" nhờ file mà lần thử 1 để lại — tức là retry biến
//     một bước hỏng thành một bước xong.
package flow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// ArtifactGiuLai là thời gian giữ artifact của một lượt chạy ĐÃ KẾT THÚC.
//
// Bảy ngày vì artifact tồn tại để người ta MỞ RA XEM sau khi lượt chạy hỏng, và
// khoảng cách giữa "lượt chạy đêm qua hỏng" với "sáng thứ hai có người đọc" đo
// bằng ngày chứ không bằng giờ.
const ArtifactGiuLai = 7 * 24 * time.Hour

// KhoaArtifactDir là tên biến bước SẢN XUẤT dùng để biết ghi file vào đâu.
//
//	prompt = "Viết báo cáo đầy đủ ra {{artifact_dir}}/bao-cao.md"
//
// Chỉ có mặt ở bước CÓ khai `artifact`. Bước không khai mà gõ biến này thì nó ở
// lại nguyên dạng chữ sống — đúng ý: ghi file vào một thư mục không ai đọc là
// một việc vô nghĩa, và im lặng cho qua thì người viết flow không bao giờ biết.
const KhoaArtifactDir = "artifact_dir"

// ArtifactRoot là gốc chứa artifact của mọi lượt chạy.
//
// Nằm trong ~/.ai-accounts chứ KHÔNG nằm trong thư mục dự án: dự án là một kho
// git, và đổ file tạm vào đó là biến `git status` thành bãi rác — rồi ai đó sẽ
// `git add -A`.
func ArtifactRoot() string { return filepath.Join(paths.AccountsRoot(), "artifacts") }

// ArtifactRunDir là thư mục artifact của một lượt chạy.
func ArtifactRunDir(runID int64) string {
	return filepath.Join(ArtifactRoot(), fmt.Sprintf("run-%d", runID))
}

// ArtifactStepDir là thư mục artifact của MỘT BƯỚC trong một lượt chạy.
//
// Tách theo bước chứ không dùng chung một thư mục cho cả lượt: các bước cùng
// đợt chạy SONG SONG, và hai bước cùng ghi `ket-qua.md` vào một chỗ thì bước nào
// chạy chậm hơn sẽ đè lên bước kia mà không ai thấy.
func ArtifactStepDir(runID int64, stepID string) string {
	return filepath.Join(ArtifactRunDir(runID), stepID)
}

// duongDanArtifact kiểm một đường dẫn khai trong flows.toml có an toàn không.
//
// Chỉ nhận đường dẫn TƯƠNG ĐỐI nằm trong thư mục của bước. Đường dẫn tuyệt đối
// hoặc có `..` cho phép một flow gửi qua mạng ghi đè file bất kỳ trên máy người
// nhận — cùng lớp nguy hiểm với lý do `plugin` chỉ nhận TÊN chứ không nhận
// đường dẫn tới executable.
func duongDanArtifact(rel string) error { return duongDanTuongDoi(rel, "thư mục của bước") }

// duongDanTuongDoi là phép kiểm cú pháp dùng chung cho HAI chỗ gọi có cùng luật
// nhưng khác RANH GIỚI: đường dẫn khai trong flows.toml bị chốt trong thư mục
// của một BƯỚC, còn đường dẫn người gọi đưa vào lúc chạy bị chốt trong thư mục
// của cả LƯỢT CHẠY.
//
// `choNao` là tên ranh giới đó, và nó có mặt vì một lý do cụ thể chứ không phải
// cho đẹp: đo thật ngày 22/08, câu từ chối của endpoint đọc artifact nói "đi ra
// ngoài thư mục của bước" trong khi thứ nó vừa chặn là một đường dẫn tương đối
// so với LƯỢT CHẠY. Người đọc sửa theo câu đó sẽ sửa sai chỗ.
func duongDanTuongDoi(rel, choNao string) error {
	if strings.TrimSpace(rel) == "" {
		return fmt.Errorf("đường dẫn rỗng")
	}
	// Kiểm CẢ hai kiểu gạch: `filepath.IsAbs` trên Linux không coi `C:\x` hay
	// `\x` là tuyệt đối, mà flows.toml thì đi qua lại giữa hai hệ điều hành.
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" ||
		strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "\\") ||
		strings.Contains(rel, ":") {
		return fmt.Errorf("phải là đường dẫn TƯƠNG ĐỐI trong %s, không phải %q", choNao, rel)
	}
	c := filepath.ToSlash(filepath.Clean(rel))
	if c == ".." || strings.HasPrefix(c, "../") {
		return fmt.Errorf("%q đi ra ngoài %s", rel, choNao)
	}
	return nil
}

// DuongDanArtifact là đường dẫn TUYỆT ĐỐI của một artifact.
func DuongDanArtifact(runID int64, stepID, rel string) string {
	return filepath.Join(ArtifactStepDir(runID, stepID), filepath.FromSlash(rel))
}

// ============================================================================
// ĐỌC ARTIFACT TỪ NGOÀI VÀO — CỬA NGUY HIỂM NHẤT CỦA CẢ MẢNH NÀY
// ============================================================================
//
// duongDanArtifact ở trên canh đường dẫn KHAI TRONG flows.toml. Hàm dưới đây
// canh một thứ khác hẳn và nặng hơn: đường dẫn do NGƯỜI GỌI đưa vào lúc chạy —
// tham số dòng lệnh, và query string của một endpoint HTTP trên cái cổng mà
// dashboard tự biết có thể không nằm trên loopback (`s.exposed`).
//
// Ai chưa qua được cửa này thì không đọc được byte nào. BA lớp, và lớp thứ ba
// mới là lớp thật:
//
//  1. Cùng bộ luật cú pháp với đường dẫn khai trong flows.toml (dùng lại đúng
//     hàm đó, không chép luật ra chỗ thứ hai): không tuyệt đối, không ổ đĩa,
//     không `..`.
//  2. Ghép vào thư mục của lượt chạy rồi Clean.
//  3. GIẢI LIÊN KẾT MỀM rồi so lại với gốc ĐÃ GIẢI LIÊN KẾT.
//
// Lớp 3 không thừa, và nó là lớp duy nhất chặn được ca thật ở đây: thư mục
// artifact là chỗ AGENT GHI VÀO. Một agent (hoặc một bước shell trong flow ai
// đó gửi tới) tạo được `run-51/soi/tat.lnk → ~/.ai-accounts/keys/` mà không cần
// một dấu `..` nào. Lúc đó lớp 1 và 2 đều cho qua sạch sẽ, vì chuỗi đường dẫn
// hoàn toàn vô tội — chỉ có ĐĨA mới biết nó trỏ đi đâu.
//
// Hỏng thì HỎNG KÍN: mọi lỗi đều trả về, không có nhánh nào "gần đúng thì cho
// qua". filepath.Rel không tính được thì cũng là từ chối.
func DuongDanArtifactAnToan(runID int64, duong string) (string, error) {
	if err := duongDanTuongDoi(duong, fmt.Sprintf("thư mục artifact của lượt chạy #%d", runID)); err != nil {
		return "", fmt.Errorf("đường dẫn artifact không hợp lệ: %w", err)
	}
	goc := ArtifactRunDir(runID)
	gocThat, err := filepath.EvalSymlinks(goc)
	if err != nil {
		return "", fmt.Errorf("lượt chạy #%d không để lại artifact nào", runID)
	}
	that, err := filepath.EvalSymlinks(filepath.Join(goc, filepath.FromSlash(duong)))
	if err != nil {
		// Cố ý KHÔNG nói ra lỗi hệ thống gốc: "permission denied" với
		// "no such file" là hai câu trả lời khác nhau cho người dò tìm, và
		// chúng vẽ được bản đồ đĩa của máy chủ. Người dùng thật thì đã có
		// danh sách file từ FlowArtifacts.
		return "", fmt.Errorf("không có artifact %q trong lượt chạy #%d", duong, runID)
	}
	if !trongThuMuc(gocThat, that) {
		return "", fmt.Errorf("đường dẫn %q đi ra NGOÀI thư mục artifact của lượt chạy #%d", duong, runID)
	}
	return that, nil
}

// trongThuMuc cho biết p có nằm trong goc không. Cả hai phải đã tuyệt đối và đã
// giải liên kết mềm — hàm này chỉ so chuỗi, nó không biết gì về đĩa.
//
// Dùng filepath.Rel chứ không dùng strings.HasPrefix: `…/run-5` là tiền tố
// chuỗi của `…/run-51`, nên HasPrefix cho một lượt chạy khác lọt qua.
func trongThuMuc(goc, p string) bool {
	rel, err := filepath.Rel(goc, p)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// TenTheoDuong lập bản đồ "<bước>/<đường dẫn tương đối>" → TÊN artifact khai
// trong flows.toml.
//
// Chiều ngược của nguoiSanXuat, và có riêng vì bảng liệt kê đi từ ĐĨA vào: nó
// duyệt file có thật rồi hỏi "cái này có tên không". File không có tên là
// chuyện BÌNH THƯỜNG — bước ghi ba file mà chỉ khai một cái là hợp lệ — nên chỗ
// gọi phải hiện nó ra chứ không được giấu. Giấu thì bảng nói thiếu đúng những
// file mà không ai ngờ tới.
func TenTheoDuong(f Flow) map[string]string {
	out := map[string]string{}
	for _, s := range f.Steps {
		for ten, rel := range s.Artifact {
			k := filepath.ToSlash(filepath.Join(s.ID, filepath.FromSlash(rel)))
			out[k] = ten
		}
	}
	return out
}

// nguoiSanXuat lập bản đồ TÊN artifact → bước khai ra nó.
//
// Tên là khoá TOÀN LƯỢT CHẠY, không phải khoá trong một bước: `{{artifacts.x}}`
// phải trỏ tới đúng một chỗ, nếu không thì bước đọc nó không có cách nào nói
// mình muốn cái `x` của ai. Validate chặn trùng tên, xem VanDeArtifact.
func nguoiSanXuat(f Flow) map[string]Step {
	out := map[string]Step{}
	// Duyệt theo thứ tự khai báo và giữ bước ĐẦU tiên: trùng tên đã bị Validate
	// chặn, nhưng nếu ai đó chạy thẳng một Flow dựng bằng mã Go thì kết quả vẫn
	// phải ổn định giữa các lần chạy.
	for _, s := range f.Steps {
		for _, ten := range tenArtifactSapXep(s) {
			if _, da := out[ten]; da {
				continue
			}
			out[ten] = s
		}
	}
	return out
}

// ChuanBiArtifact xoá sạch rồi tạo lại thư mục artifact của một bước.
//
// Gọi TRƯỚC MỖI LẦN THỬ — xem quyết định #4 ở đầu file. Bước không khai artifact
// thì không đụng vào đĩa một lần nào.
func ChuanBiArtifact(runID int64, s Step) (string, error) {
	if len(s.Artifact) == 0 {
		return "", nil
	}
	dir := ArtifactStepDir(runID, s.ID)
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("không dọn được thư mục artifact %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("không tạo được thư mục artifact %s: %w", dir, err)
	}
	return dir, nil
}

// ThieuArtifact kiểm HỢP ĐỒNG ĐẦU RA của bước: mọi artifact đã khai phải có
// file thật. Trả về câu mô tả cái thiếu, hoặc "" nếu đủ.
//
// Kiểm SỰ TỒN TẠI, không kiểm nội dung. Một file 0 byte vẫn tính là đã sản xuất:
// "file rỗng" là một kết quả hợp lệ (danh sách không có mục nào), và đoán hộ ở
// đây sẽ chặn đúng những flow dùng artifact làm cờ có/không.
//
// Thư mục KHÔNG tính là file. `artifact = { x = "ket-qua" }` mà bước tạo ra một
// thư mục tên `ket-qua` thì bước sau nhận đường dẫn trỏ vào thư mục và mọi lệnh
// đọc file đều hỏng bằng một thông báo chẳng liên quan.
func ThieuArtifact(runID int64, s Step) string {
	thieu := thieuTrongThuMuc(s, ArtifactStepDir(runID, s.ID))
	if thieu == "" {
		return ""
	}
	return fmt.Sprintf("bước khai artifact nhưng chạy xong không có file: %s — bước phải ghi file vào {{%s}} (%s)",
		thieu, KhoaArtifactDir, ArtifactStepDir(runID, s.ID))
}

// thieuTrongThuMuc liệt kê artifact đã khai mà KHÔNG có file thật trong `dir`.
//
// Tách ra vì có HAI ranh giới dùng chung một bộ luật: bước thường kiểm thư mục
// của bước, bước `foreach` kiểm thư mục của TẪP LƯỢT LẶP. Chép luật ra hai chỗ là
// mời hai chỗ lệch nhau, mà lệch ở đây nghĩa là một hợp đồng đầu ra không ai giữ.
func thieuTrongThuMuc(s Step, dir string) string {
	var thieu []string
	for _, ten := range tenArtifactSapXep(s) {
		p := filepath.Join(dir, filepath.FromSlash(s.Artifact[ten]))
		st, err := os.Stat(p)
		switch {
		case err != nil:
			thieu = append(thieu, fmt.Sprintf("%q (%s)", ten, s.Artifact[ten]))
		case st.IsDir():
			thieu = append(thieu, fmt.Sprintf("%q (%s — là THƯ MỤC, không phải file)", ten, s.Artifact[ten]))
		}
	}
	return strings.Join(thieu, ", ")
}

// tenArtifactSapXep trả về tên artifact của một bước theo thứ tự cố định.
//
// Map của Go trả ra ngẫu nhiên, và một thông điệp lỗi đổi thứ tự mỗi lần chạy là
// một thông điệp không grep được.
func tenArtifactSapXep(s Step) []string {
	out := make([]string, 0, len(s.Artifact))
	for ten := range s.Artifact {
		out = append(out, ten)
	}
	sort.Strings(out)
	return out
}

// CauChanArtifact là câu thay vào chỗ một artifact bị `doc_duoc` chặn.
//
// Cùng luật với CauChan: nói ra việc chặn, có tên và có cách mở. Nếu không lọc ở
// đây thì `doc_duoc` thành một cái rào có cửa sau — chặn được chuỗi output
// nhưng vẫn phát đường dẫn tới FILE của đúng bước đó.
func CauChanArtifact(ten, buoc string) string {
	return fmt.Sprintf("(không được phép đọc artifact %q của bước %q — thêm %s vào doc_duoc nếu cần)",
		ten, buoc, buoc)
}

// artifactChoDoc: bước s có được đọc artifact của bước buoc không.
//
// Cùng một phép kiểm tư cách với output và với `merge` — xem ChoDoc trong
// doc_duoc.go. Giữ cái tên này vì nó nói đúng chỗ dùng.
func artifactChoDoc(s Step, buoc string) bool { return ChoDoc(s, buoc) }

// MoiTruongArtifact dựng phần biến artifact cho MỘT bước sắp chạy.
//
// Trả về map khoá → giá trị để trộn vào env:
//
//	artifact_dir        thư mục bước NÀY phải ghi file vào (chỉ khi nó có khai)
//	artifacts.<tên>     đường dẫn tuyệt đối tới artifact của bước KHÁC
//
// Chỉ phát đường dẫn của artifact mà bước sản xuất đã `done`. Bước chưa chạy thì
// KHÔNG có khoá — placeholder ở lại nguyên dạng, và ExpandChay/ConSotArtifact
// biến nó thành một câu nói thẳng là chưa có. Phát bừa một đường dẫn của bước
// chưa chạy là đưa cho bước sau một cái tên file trông rất thật trỏ vào hư
// không: đúng kiểu hỏng của lượt #29, chỉ đổi từ chuỗi sang đường dẫn.
func MoiTruongArtifact(runID int64, f Flow, s Step, states map[string]string) map[string]string {
	out := map[string]string{}
	if len(s.Artifact) > 0 {
		out[KhoaArtifactDir] = ArtifactStepDir(runID, s.ID)
	}
	for ten, p := range nguoiSanXuat(f) {
		if p.ID == s.ID {
			continue // artifact của chính mình thì chưa có lúc mình đang chạy
		}
		// Khoá PHỤ THUỘC HÌNH DẠNG bước sản xuất, và chỉ một khoá duy nhất được phát:
		// bước thường → `artifacts.<tên>`, bước `foreach` →
		// `artifacts.<tên>.danh_sach`. Phát cả hai là để cả hai cú pháp cùng chạy
		// được, và khi đó cái sai sẽ im lặng thay vì báo — xem foreach_artifact.go.
		khoa := "artifacts." + ten
		if p.ForEach != "" {
			khoa += "." + HauToDanhSach
		}
		if !artifactChoDoc(s, p.ID) {
			out[khoa] = CauChanArtifact(ten, p.ID)
			continue
		}
		if states[p.ID] != store.StepDone {
			continue
		}
		if p.ForEach != "" {
			out[khoa] = DuongDanDanhSach(runID, p.ID, ten)
			continue
		}
		out[khoa] = DuongDanArtifact(runID, p.ID, p.Artifact[ten])
	}
	return out
}

// DonArtifact xoá thư mục artifact của các lượt chạy ĐÃ KẾT THÚC và cũ hơn `gia`.
//
// TRẢ LỜI THẲNG "artifact sống bao lâu, ai dọn":
//
//   - Lượt chạy còn `running` hoặc `waiting_approval`: KHÔNG BAO GIỜ bị dọn, dù
//     cũ tới đâu. Một lượt dừng chờ người duyệt ba tuần rồi mới được duyệt vẫn
//     phải tìm lại đủ file của các bước trước nó — dọn ở đây là làm hỏng đúng
//     tính năng resume mà cả Pha 3 dựng lên.
//   - Lượt đã `completed`/`failed`/`cancelled` và thư mục không đổi trong `gia`:
//     xoá.
//   - Thư mục không có lượt chạy nào trong sổ (sổ bị xoá, chép tay từ máy khác):
//     xoá theo cùng ngưỡng tuổi. Không giữ vô hạn thứ không ai tra ngược được.
//
// Gọi ở đầu MỖI lượt chạy mới (Runner.Start). Cố ý KHÔNG có lệnh CLI riêng: một
// nút dọn rác mà người ta phải nhớ bấm là một nút không ai bấm, và cái thư mục
// sẽ phình cho tới ngày hết đĩa.
//
// Best-effort: lỗi ở một thư mục không chặn thư mục khác và không chặn lượt chạy.
func DonArtifact(db *store.DB, gia time.Duration, bayGio time.Time) (daXoa int) {
	ents, err := os.ReadDir(ArtifactRoot())
	if err != nil {
		return 0 // chưa có thư mục nào — chuyện bình thường
	}
	for _, e := range ents {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "run-") {
			continue
		}
		dir := filepath.Join(ArtifactRoot(), e.Name())
		if conSong(db, e.Name()) {
			continue
		}
		if moiDoi(dir, gia, bayGio) {
			continue
		}
		if os.RemoveAll(dir) == nil {
			daXoa++
		}
	}
	return daXoa
}

// conSong: thư mục run-<id> này thuộc một lượt chạy CHƯA kết thúc?
//
// Không tra được sổ (đóng, hỏng, id không phải số) thì trả về false — thư mục
// rơi về luật tuổi. Trả về true ở đây sẽ giữ vĩnh viễn mọi thư mục lạ.
func conSong(db *store.DB, ten string) bool {
	if db == nil {
		return false
	}
	var id int64
	if _, err := fmt.Sscanf(ten, "run-%d", &id); err != nil || id <= 0 {
		return false
	}
	run, err := db.GetRun(id)
	if err != nil {
		return false
	}
	return run.State == store.RunRunning || run.State == store.RunWaiting
}

// moiDoi: thư mục (hoặc bất cứ file nào trong nó) còn mới hơn ngưỡng tuổi.
//
// Nhìn cả file bên trong chứ không chỉ mtime của thư mục: trên Windows, ghi đè
// một file có sẵn KHÔNG cập nhật mtime của thư mục cha, nên chỉ nhìn thư mục là
// xoá nhầm artifact vừa được ghi lại năm phút trước.
func moiDoi(dir string, gia time.Duration, bayGio time.Time) bool {
	moi := false
	_ = filepath.Walk(dir, func(_ string, fi os.FileInfo, err error) error {
		if err != nil || moi {
			return nil
		}
		if bayGio.Sub(fi.ModTime()) < gia {
			moi = true
		}
		return nil
	})
	return moi
}

// ------------------------------- KIỂM TRA -------------------------------

// vanBanCuaBuoc gom mọi chỗ trong một bước có thể chứa {{...}}.
//
// Gom một chỗ để không sót: mỗi trường quên là một chỗ người viết flow gõ
// `{{artifacts.x}}` rồi không được ai soi, và chỉ phát hiện ra lúc chạy thật.
func vanBanCuaBuoc(s Step) []string {
	out := []string{s.Prompt, s.Message, s.Vao, s.When, s.ForEach}
	out = append(out, s.Run...)
	for _, k := range sapXepKhoa(s.ThamSo) {
		out = append(out, s.ThamSo[k])
	}
	return out
}

func sapXepKhoa(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// loaiKhongGhiDuocFile là những loại node KHÔNG có đường nào ghi ra file.
//
// `approve` không bao giờ được thực thi (nó là cái rào, người duyệt mới gỡ);
// `notify` và `model` chỉ trả về chữ. Khai artifact ở những loại này là khai một
// hợp đồng chắc chắn không giữ được — bước sẽ hỏng ở MỌI lượt chạy, và hỏng vì
// một lý do mà lúc viết flow đã nhìn ra được.
//
// `merge` và `route` vào cùng danh sách vì cùng lý do: merge chỉ nối chữ đã có
// sẵn trong bộ nhớ, route chỉ trả về một cái tên. Không loại nào có đường nào
// chạm tới đĩa.
var loaiKhongGhiDuocFile = map[string]bool{
	TypeApprove: true, TypeNotify: true, TypeModel: true,
	TypeMerge: true, TypeRoute: true,
}

// VanDeArtifact soi phần `artifact` của cả flow.
//
// LỖI (chặn lưu, chặn chạy) cho những thứ chắc chắn hỏng:
//   - tên artifact không hợp lệ, hoặc hai bước khai trùng tên;
//   - đường dẫn tuyệt đối hoặc đi ra ngoài thư mục của bước;
//   - khai ở loại node không ghi được file;
//   - hình dạng placeholder không khớp bước sản xuất (tên trần cho bước `foreach`,
//     `.danh_sach` cho bước thường, hoặc một hậu tố không có thật);
//   - `{{artifacts.x}}` mà không bước nào sản xuất `x`, hoặc trỏ vào chính mình.
//
// CẢNH BÁO cho những thứ chỉ là vô nghĩa:
//   - trỏ tới artifact của bước chạy CÙNG ĐỢT hoặc SAU;
//   - dùng `{{artifact_dir}}` ở bước không khai artifact nào.
func VanDeArtifact(f Flow) []Problem {
	var ps []Problem
	loi := func(step, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: step, Msg: msg}) }
	canh := func(step, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: step, Msg: msg, Warn: true}) }

	// 1. Phần KHAI: tên, đường dẫn, loại node, foreach, trùng tên.
	chuCua := map[string]string{} // tên artifact → id bước đã khai
	lapCua := map[string]bool{}   // tên artifact → bước sản xuất có `foreach` không
	for _, s := range f.Steps {
		if len(s.Artifact) == 0 {
			continue
		}
		if loaiKhongGhiDuocFile[s.Type] {
			loi(s.ID, fmt.Sprintf("type %q không có đường nào ghi ra file nên không để lại artifact được "+
				"— dùng agent/shell/test/lint/plugin, hoặc bỏ `artifact`", s.Type))
		}
		for _, ten := range tenArtifactSapXep(s) {
			if !idRe.MatchString(ten) {
				loi(s.ID, fmt.Sprintf("tên artifact %q chỉ được dùng chữ thường, số, - và _", ten))
			}
			if err := duongDanArtifact(s.Artifact[ten]); err != nil {
				loi(s.ID, fmt.Sprintf("artifact %q: %v", ten, err))
			}
			if chu, da := chuCua[ten]; da {
				loi(s.ID, fmt.Sprintf("artifact %q đã được bước %q khai rồi — tên artifact là khoá của cả "+
					"lượt chạy, trùng tên thì `{{artifacts.%s}}` không nói được là của bước nào", ten, chu, ten))
				continue
			}
			chuCua[ten] = s.ID
			lapCua[ten] = s.ForEach != ""
		}
	}

	// 2. Phần ĐỌC: mọi {{artifacts.x}} phải có người sản xuất, và người đó phải
	//    chạy TRƯỚC.
	dotCua := map[string]int{}
	if dots, err := Dot(f); err == nil {
		for _, d := range dots {
			for _, s := range d.Buoc {
				dotCua[s.ID] = d.So
			}
		}
	}

	for _, s := range f.Steps {
		daBao := map[string]bool{}
		for _, v := range vanBanCuaBuoc(s) {
			for _, m := range conSotArtifact.FindAllStringSubmatch(v, -1) {
				ten, hauTo := m[1], m[2]
				if daBao[ten+"|"+hauTo] {
					continue
				}
				daBao[ten+"|"+hauTo] = true
				chu, co := chuCua[ten]
				if !co {
					loi(s.ID, fmt.Sprintf("{{artifacts.%s}} nhưng không bước nào khai artifact %q "+
						"— thêm artifact = { %s = \"<tên file>\" } vào bước sản xuất", ten, ten, ten))
					continue
				}
				if chu == s.ID {
					loi(s.ID, fmt.Sprintf("{{artifacts.%s}} trỏ vào artifact của CHÍNH bước này "+
						"— lúc bước chạy thì file chưa có; ghi file vào {{%s}} thay vì đọc nó",
						ten, KhoaArtifactDir))
					continue
				}
				// HÌNH DẠNG phải KHỚP với bước sản xuất. Một bước `foreach` để lại N
				// file nên tên trần không trỏ được tới cái nào; một bước thường để lại
				// đúng MỘT file nên không có bản kê nào để trỏ. Lệch thì DỪNG ở đây —
				// xem foreach_artifact.go cho lý do không cho tên trần tự đổi nghĩa.
				switch {
				case hauTo == "" && lapCua[ten]:
					loi(s.ID, fmt.Sprintf("{{artifacts.%s}} là artifact của bước %q có `foreach` — bước đó "+
						"để lại MỖI LƯỢT LẶP một file nên tên trần không trỏ được tới cái nào. Dùng "+
						"{{artifacts.%s.%s}} — nó là đường dẫn tới MỘT file liệt kê đủ N đường dẫn, một dòng "+
						"một cái (đi được vào argv của `run` vì nó vẫn chỉ là MỘT đường dẫn)",
						ten, chu, ten, HauToDanhSach))
					continue
				case hauTo == HauToDanhSach && !lapCua[ten]:
					loi(s.ID, fmt.Sprintf("{{artifacts.%s.%s}} nhưng bước %q KHÔNG có `foreach` — nó để lại đúng "+
						"MỘT file, không có bản kê nào để trỏ. Dùng {{artifacts.%s}}",
						ten, hauTo, chu, ten))
					continue
				case hauTo != "" && hauTo != HauToDanhSach:
					loi(s.ID, fmt.Sprintf("{{artifacts.%s.%s}}: hậu tố %q không có — chỉ có `.%s` (cho artifact "+
						"của bước `foreach`) hoặc không hậu tố gì (cho bước thường)",
						ten, hauTo, hauTo, HauToDanhSach))
					continue
				}
				// Thứ tự đợt chỉ có nghĩa khi flow không có chu trình; có chu
				// trình thì Validate đã báo ở chỗ khác và Dot() trả lỗi.
				if len(dotCua) == 0 {
					continue
				}
				if dotCua[chu] >= dotCua[s.ID] {
					canh(s.ID, fmt.Sprintf("{{artifacts.%s}} là của bước %q chạy cùng đợt hoặc sau (đợt %d, "+
						"bước này ở đợt %d) — lúc bước này chạy thì file chưa có; thêm needs = [%q]",
						ten, chu, dotCua[chu], dotCua[s.ID], chu))
				}
			}
			if len(s.Artifact) == 0 && strings.Contains(v, "{{"+KhoaArtifactDir+"}}") {
				canh(s.ID, fmt.Sprintf("dùng {{%s}} nhưng bước không khai `artifact` nào — biến này sẽ "+
					"không được thay, và file ghi ra đó không bước nào đọc được", KhoaArtifactDir))
				break
			}
		}
	}
	return ps
}
