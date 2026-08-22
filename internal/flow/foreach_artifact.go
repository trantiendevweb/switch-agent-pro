// ARTIFACT của một bước `foreach` — N lượt lặp, N file, và MỘT cách trỏ tới cả N.
//
// ============================================================================
// CÂU HỎI DUY NHẤT CỦA MẢNH NÀY: bước sau trỏ tới N file bằng cú pháp gì
// ============================================================================
//
// Báo cáo #199 chặn `foreach` + `artifact` và nói thẳng lý do: chưa rõ cú pháp
// trỏ nên thế nào, mà đoán bừa rồi phải đổi thì đắt hơn chờ. Đây là câu trả lời,
// kèm hai hướng đã cân và vì sao chúng bị loại.
//
// ---------------------------------------------------------------------------
// LOẠI: trỏ theo CHỈ SỐ — `{{artifacts.ten[0]}}` hoặc `{{artifacts.ten.0}}`
// ---------------------------------------------------------------------------
//
// Nghe tự nhiên nhất, và sai ở đúng chỗ chết người: KHÔNG AI BIẾT N LÚC VIẾT
// flows.toml. Nguồn của `foreach` gần như luôn là output của bước trước
// (`foreach = "steps.liet-ke.output"`), nên độ dài danh sách chỉ có lúc chạy.
//
// Hậu quả cụ thể: danh sách ra 5 mục, người viết flow gõ `[0] [1] [2]`, và mục
// 4–5 biến mất. Không lệnh nào hỏng, không dòng nào đỏ, bản tóm tắt vẫn nói
// "xong". Đó ĐÚNG là lớp "lỗ mất việc im lặng" mà cả mảnh idempotency đã đi
// đường vòng để tránh — không có lý do gì mở lại nó ở cửa bên cạnh.
//
// Và `Validate` KHÔNG cứu được: muốn biết `[3]` có vượt biên không thì phải biết
// N, mà N chưa tồn tại lúc kiểm. Một cú pháp mà bộ kiểm không soi nổi là một cú
// pháp chỉ hỏng lúc chạy thật.
//
// (Hướng "chỉ số động" — bước sau cũng `foreach`, dùng `{{index}}` của chính nó
// để lấy file thứ `index` của bước trước — còn tệ hơn: nó giả định hai bước lặp
// trên CÙNG một danh sách theo CÙNG một thứ tự. Không có gì trong engine kiểm
// được giả định đó, và khi nó sai thì bước nhận một đường dẫn hợp lệ trỏ vào
// file của MỘT MỤC KHÁC. Sai mà trông đúng, lại còn im.)
//
// ---------------------------------------------------------------------------
// CHỌN: CẢ DANH SÁCH, và bước sau nhận một FILE BẢN KÊ
// ---------------------------------------------------------------------------
//
//	{{artifacts.<tên>.danh_sach}}
//
// → đường dẫn TUYỆT ĐỐI tới một file do bộ chạy sinh ra, chứa N đường dẫn tuyệt
// đối, mỗi dòng một cái, theo đúng thứ tự lượt lặp.
//
// TRẢ LỜI THẲNG CÂU "BƯỚC `shell` NHẬN THẾ NÀO, ARGV KHÔNG CÓ VÒNG LẶP":
//
// Đúng — argv không có vòng lặp, và không có cú pháp nào bịa ra được một cái.
// Nên thứ đi vào argv KHÔNG PHẢI N đường dẫn: nó là MỘT đường dẫn, tới cái file
// liệt kê N đường dẫn kia. Một phần tử argv, luôn luôn, bất kể N bằng 3 hay 50:
//
//	run = ["python", "gop.py", "--danh-sach", "{{artifacts.bao-cao.danh_sach}}"]
//
// Chương trình được gọi đọc file đó rồi tự lặp — nó có vòng lặp, argv thì không.
//
// VÌ SAO KHÔNG NHÉT N ĐƯỜNG DẪN VÀO MỘT CHUỖI NGĂN BỞI XUỐNG DÒNG: vì nó lọt
// vào argv thành MỘT phần tử chứa ba dòng, và lệnh nhận một "tên file" có ký tự
// xuống dòng ở giữa. Hỏng bằng một thông báo chẳng liên quan, ở một bước chẳng
// liên quan — cùng lớp với cái bẫy `{{artifacts.x}}` chưa thay mà ArtifactConSot
// sinh ra để chặn.
//
// Bước `agent` cũng dùng đúng một cú pháp đó: prompt nói "đọc danh sách file
// trong {{artifacts.bao-cao.danh_sach}} rồi gộp lại". Một cú pháp cho mọi loại
// node, không có ngoại lệ nào phải nhớ.
//
// ---------------------------------------------------------------------------
// BARE `{{artifacts.<tên>}}` CỦA MỘT BƯỚC LẶP LÀ **LỖI**, không phải mặc định
// ---------------------------------------------------------------------------
//
// Cám dỗ lớn nhất ở đây là để `{{artifacts.ban-va}}` tự đổi nghĩa thành "danh
// sách" khi bước sản xuất có `foreach`. TUYỆT ĐỐI KHÔNG: khắp hệ thống, tên trần
// nghĩa là MỘT file, và `run = ["git","apply","{{artifacts.ban-va}}"]` là câu
// mẫu nằm ngay trong tài liệu. Đổi nghĩa nó theo hình dạng của bước sản xuất là
// làm cho cùng một dòng flows.toml có hai nghĩa tuỳ nơi khác viết gì.
//
// Nên: bước sản xuất có `foreach` mà bước sau viết tên trần → `Validate` BÁO LỖI
// và chỉ đúng cú pháp phải dùng. Ngược lại, bước sản xuất KHÔNG lặp mà bước sau
// viết `.danh_sach` → cũng báo lỗi. Hình dạng phải khớp, và lệch thì dừng ở lúc
// lưu chứ không phải lúc chạy.
//
// ---------------------------------------------------------------------------
// BỐ CỤC TRÊN ĐĨA
// ---------------------------------------------------------------------------
//
//	artifacts/run-52/soi/            ← thư mục của BƯỚC
//	                 ├── 1/ban-va.diff   ← lượt lặp 1  ({{index}} == 1)
//	                 ├── 2/ban-va.diff
//	                 ├── 3/ban-va.diff
//	                 └── danh-sach-ban-va.txt   ← {{artifacts.ban-va.danh_sach}}
//
// Tên thư mục lượt lặp là ĐÚNG con số của `{{index}}` — không đệm số 0. Đệm cho
// đẹp thứ tự chữ cái (`01`, `02`, … `10`) sẽ làm `{{index}}` và tên thư mục lệch
// nhau, và ai đó ghép `{{artifact_dir}}/../{{index}}/x` sẽ trỏ vào hư không. Thứ
// tự đúng đã có ở file bản kê rồi; đó mới là kênh có thứ tự.
package flow

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// HauToDanhSach là hậu tố DUY NHẤT của `{{artifacts.<tên>.<hậu tố>}}`.
//
// Chỉ có một, và cố ý: mỗi hậu tố thêm vào là một thứ người viết flow phải nhớ,
// và là một nhánh nữa `Validate` phải soi. Xem đầu file cho ba hướng đã loại.
const HauToDanhSach = "danh_sach"

// ArtifactLapDir là thư mục artifact của MỘT LƯỢT LẶP.
//
// `index` là chỉ số 1-based, cùng con số với `{{index}}`.
func ArtifactLapDir(runID int64, stepID string, index int) string {
	return filepath.Join(ArtifactStepDir(runID, stepID), strconv.Itoa(index))
}

// DuongDanArtifactLap là đường dẫn TUYỆT ĐỐI tới artifact của một lượt lặp.
func DuongDanArtifactLap(runID int64, stepID string, index int, rel string) string {
	return filepath.Join(ArtifactLapDir(runID, stepID, index), filepath.FromSlash(rel))
}

// tenFileDanhSach là tên file bản kê của một artifact.
//
// Nằm trong thư mục của BƯỚC, cạnh các thư mục lượt lặp. Không đụng tên được với
// thư mục lượt lặp vì tên thư mục lượt lặp chỉ gồm chữ số.
func tenFileDanhSach(ten string) string { return "danh-sach-" + ten + ".txt" }

// DuongDanDanhSach là đường dẫn TUYỆT ĐỐI của file bản kê.
func DuongDanDanhSach(runID int64, stepID, ten string) string {
	return filepath.Join(ArtifactStepDir(runID, stepID), tenFileDanhSach(ten))
}

// BanDoTenArtifact trả về hàm tra TÊN artifact từ một đường dẫn tương đối so
// với thư mục LƯỢT CHẠY (`<bước>/…`, dùng dấu `/`).
//
// VÌ SAO KHÔNG DÙNG THẲNG TenTheoDuong: bản đồ đó chỉ tra được đường dẫn của
// bước THƯỜNG. Một bước `foreach` để lại `<bước>/<index>/<file>` và một file bản
// kê, mà `<index>` chỉ có lúc chạy — nên không map tĩnh nào liệt kê được chúng.
//
// Không có hàm này thì bảng `sagent flow artifacts` in "(không bước nào khai)"
// cho MỌI file của một bước lặp. Đó không phải một ô trống vô hại: câu đó là một
// lời khẳng định, và nó SAI — bước có khai, chỉ là bảng không tra nổi. Cùng lớp
// với mọi chỗ trong dự án này mà một mặt nói sai về mặt khác.
func BanDoTenArtifact(f Flow) func(slash string) string {
	exact := TenTheoDuong(f)
	// bước có `foreach` → (đường dẫn tương đối trong lượt lặp → tên)
	lap := map[string]map[string]string{}
	// bước có `foreach` → (tên file bản kê → tên)
	ke := map[string]map[string]string{}
	for _, s := range f.Steps {
		if s.ForEach == "" || len(s.Artifact) == 0 {
			continue
		}
		lap[s.ID] = map[string]string{}
		ke[s.ID] = map[string]string{}
		for ten, rel := range s.Artifact {
			lap[s.ID][filepath.ToSlash(filepath.Clean(rel))] = ten
			ke[s.ID][tenFileDanhSach(ten)] = ten
		}
	}

	return func(slash string) string {
		if ten := exact[slash]; ten != "" {
			return ten
		}
		phan := strings.SplitN(slash, "/", 2)
		if len(phan) != 2 {
			return ""
		}
		buoc, con := phan[0], phan[1]
		if ten := ke[buoc][con]; ten != "" {
			return ten + " (bản kê)"
		}
		// `<index>/<đường dẫn còn lại>` — chỉ số là số nguyên, không đệm số 0.
		sau := strings.SplitN(con, "/", 2)
		if len(sau) != 2 {
			return ""
		}
		chiSo, err := strconv.Atoi(sau[0])
		if err != nil || chiSo < 1 {
			return ""
		}
		if ten := lap[buoc][sau[1]]; ten != "" {
			return fmt.Sprintf("%s (mục %d)", ten, chiSo)
		}
		return ""
	}
}

// ChuanBiArtifactLap tạo thư mục artifact cho MỘT lượt lặp.
//
// KHÔNG xoá gì: thư mục của cả bước đã được ChuanBiArtifact dọn sạch MỘT LẦN
// trước khi phát các lượt. Dọn ở đây là dọn trong lúc các lượt khác đang ghi —
// chúng chạy song song, và một `RemoveAll` trên thư mục cha sẽ ăn mất file của
// lượt bên cạnh.
func ChuanBiArtifactLap(runID int64, s Step, index int) (string, error) {
	if len(s.Artifact) == 0 {
		return "", nil
	}
	dir := ArtifactLapDir(runID, s.ID, index)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("không tạo được thư mục artifact %s: %w", dir, err)
	}
	return dir, nil
}

// ThieuArtifactLap kiểm HỢP ĐỒNG ĐẦU RA của MỘT lượt lặp.
//
// Kiểm từng lượt chứ không kiểm cả bước, và đây là chỗ dễ làm sai nhất: nếu chỉ
// hỏi "thư mục của bước có file nào không" thì 49 lượt ghi được và 1 lượt câm sẽ
// vẫn qua, rồi file bản kê trỏ tới một đường dẫn không tồn tại. Một lượt không
// giao hàng là một MỤC bị mất — phải hỏng, và phải nói ra mục nào.
func ThieuArtifactLap(runID int64, s Step, index int, item string) string {
	thieu := thieuTrongThuMuc(s, ArtifactLapDir(runID, s.ID, index))
	if thieu == "" {
		return ""
	}
	return fmt.Sprintf("mục %d (%s): khai artifact nhưng chạy xong không có file: %s "+
		"— mỗi lượt lặp phải tự ghi file của mình vào {{%s}} (%s)",
		index, short(item, 40), thieu, KhoaArtifactDir, ArtifactLapDir(runID, s.ID, index))
}

// GhiDanhSachArtifact sinh file bản kê cho MỌI artifact của một bước lặp.
//
// Gọi SAU KHI mọi lượt lặp đã xong và đã qua hợp đồng đầu ra. Ghi sớm hơn là
// phát ra một danh sách trỏ tới file chưa tồn tại — đúng kiểu hỏng mà hợp đồng
// artifact sinh ra để chặn, chỉ lùi ra sau một bước.
//
// Đường dẫn trong bản kê là TUYỆT ĐỐI: bước tiêu thụ có thể chạy ở thư mục làm
// việc bất kỳ, và một danh sách đường dẫn tương đối là một danh sách chỉ đúng ở
// đúng một chỗ đứng.
func GhiDanhSachArtifact(runID int64, s Step, soLuot int) error {
	if len(s.Artifact) == 0 {
		return nil
	}
	for _, ten := range tenArtifactSapXep(s) {
		var b strings.Builder
		for i := 1; i <= soLuot; i++ {
			b.WriteString(DuongDanArtifactLap(runID, s.ID, i, s.Artifact[ten]))
			b.WriteString("\n")
		}
		p := DuongDanDanhSach(runID, s.ID, ten)
		if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
			return fmt.Errorf("không ghi được bản kê artifact %q: %w", ten, err)
		}
	}
	return nil
}
