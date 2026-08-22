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
func duongDanArtifact(rel string) error {
	if strings.TrimSpace(rel) == "" {
		return fmt.Errorf("đường dẫn rỗng")
	}
	// Kiểm CẢ hai kiểu gạch: `filepath.IsAbs` trên Linux không coi `C:\x` hay
	// `\x` là tuyệt đối, mà flows.toml thì đi qua lại giữa hai hệ điều hành.
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" ||
		strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "\\") ||
		strings.Contains(rel, ":") {
		return fmt.Errorf("phải là đường dẫn TƯƠNG ĐỐI trong thư mục của bước, không phải %q", rel)
	}
	c := filepath.ToSlash(filepath.Clean(rel))
	if c == ".." || strings.HasPrefix(c, "../") {
		return fmt.Errorf("%q đi ra ngoài thư mục của bước", rel)
	}
	return nil
}

// DuongDanArtifact là đường dẫn TUYỆT ĐỐI của một artifact.
func DuongDanArtifact(runID int64, stepID, rel string) string {
	return filepath.Join(ArtifactStepDir(runID, stepID), filepath.FromSlash(rel))
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
	var thieu []string
	for _, ten := range tenArtifactSapXep(s) {
		p := DuongDanArtifact(runID, s.ID, s.Artifact[ten])
		st, err := os.Stat(p)
		switch {
		case err != nil:
			thieu = append(thieu, fmt.Sprintf("%q (%s)", ten, s.Artifact[ten]))
		case st.IsDir():
			thieu = append(thieu, fmt.Sprintf("%q (%s — là THƯ MỤC, không phải file)", ten, s.Artifact[ten]))
		}
	}
	if len(thieu) == 0 {
		return ""
	}
	return fmt.Sprintf("bước khai artifact nhưng chạy xong không có file: %s — bước phải ghi file vào {{%s}} (%s)",
		strings.Join(thieu, ", "), KhoaArtifactDir, ArtifactStepDir(runID, s.ID))
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
func artifactChoDoc(s Step, buoc string) bool {
	if s.DocDuoc == nil {
		return true // chưa khai = mở hết, y như output
	}
	for _, id := range s.DocDuoc {
		if id == buoc {
			return true
		}
	}
	return false
}

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
		if !artifactChoDoc(s, p.ID) {
			out["artifacts."+ten] = CauChanArtifact(ten, p.ID)
			continue
		}
		if states[p.ID] != store.StepDone {
			continue
		}
		out["artifacts."+ten] = DuongDanArtifact(runID, p.ID, p.Artifact[ten])
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
// `notify` và `model` chỉ trả về chữ. Khai artifact ở ba loại này là khai một
// hợp đồng chắc chắn không giữ được — bước sẽ hỏng ở MỌI lượt chạy, và hỏng vì
// một lý do mà lúc viết flow đã nhìn ra được.
var loaiKhongGhiDuocFile = map[string]bool{
	TypeApprove: true, TypeNotify: true, TypeModel: true,
}

// VanDeArtifact soi phần `artifact` của cả flow.
//
// LỖI (chặn lưu, chặn chạy) cho những thứ chắc chắn hỏng:
//   - tên artifact không hợp lệ, hoặc hai bước khai trùng tên;
//   - đường dẫn tuyệt đối hoặc đi ra ngoài thư mục của bước;
//   - khai ở loại node không ghi được file;
//   - khai chung với `foreach`;
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
	for _, s := range f.Steps {
		if len(s.Artifact) == 0 {
			continue
		}
		if loaiKhongGhiDuocFile[s.Type] {
			loi(s.ID, fmt.Sprintf("type %q không có đường nào ghi ra file nên không để lại artifact được "+
				"— dùng agent/shell/test/lint/plugin, hoặc bỏ `artifact`", s.Type))
		}
		if s.ForEach != "" {
			loi(s.ID, "không dùng `artifact` chung với `foreach`: mọi lượt lặp chạy SONG SONG trong cùng "+
				"một thư mục artifact và sẽ ghi đè lên nhau, còn `{{artifacts.<tên>}}` thì chỉ trỏ được tới MỘT file")
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
				ten := m[1]
				if daBao[ten] {
					continue
				}
				daBao[ten] = true
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
