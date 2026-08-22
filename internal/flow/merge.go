// MERGE — gộp ĐẦU RA của N bước thành một khối chữ.
//
// ============================================================================
// TRƯỚC HẾT: `merge` KHÔNG PHẢI GỘP NHÁNH GIT
// ============================================================================
//
// Dòng khai cũ ở flow.go ghi `merge` là "gộp nhánh — hành động nguy hiểm, mặc
// định cần duyệt", và để `implemented = false` với ghi chú "còn chờ cơ chế merge
// an toàn". Nó treo ở đó suốt vì cái cơ chế ấy không tồn tại: một node tự chạy
// `git merge` là một node có quyền viết đè lên cây mã của người khác, và không
// có cách nào làm việc đó an toàn bằng một dòng TOML.
//
// Node này KHÔNG làm việc đó. Nó gộp ĐẦU RA (chữ) của N bước. Muốn gộp nhánh
// git thì đường cũ vẫn còn và vẫn đúng hơn: một bước `shell` chạy `git merge`
// đứng sau một bước `approve`. Ở đó người duyệt nhìn thấy chính xác lệnh sắp
// chạy, còn ở đây thì không.
//
// Nói ra chuyện đổi nghĩa này chứ không lặng lẽ bật cờ `implemented`: ai đọc
// kế hoạch cũ rồi viết `type = "merge"` mà tưởng nó gộp nhánh sẽ hụt.
//
// ============================================================================
// CÂU HỎI 1: GỘP N BƯỚC THÌ THEO THỨ TỰ NÀO?
// ============================================================================
//
// Theo ĐÚNG thứ tự khai trong `needs`. Không phải thứ tự xong.
//
// Đây là câu quyết định cả node. Ba câu trả lời khả dĩ:
//
//	(a) THỨ TỰ XONG. Sai, và sai theo kiểu tệ nhất. Bộ chạy này chạy theo ĐỢT,
//	    nhiều bước SONG SONG (runWave). Hai bước cùng đợt không có thứ tự xong
//	    nào cả — nó phụ thuộc vào mạng, vào máy, vào việc hôm nay nhà cung cấp
//	    trả lời nhanh hay chậm. Chạy hai lần một flow y hệt nhau sẽ ra hai khối
//	    chữ khác nhau, và không có gì nói cho người đọc biết vì sao. Bước sau
//	    (thường là một agent tổng hợp) nhận đầu vào khác nhau ở hai lượt chạy
//	    "giống hệt" — đó là một lượt chạy không lặp lại được.
//
//	(b) THỨ TỰ KHAI TRONG `needs`. CHỌN. Nó nằm trong flows.toml, người viết
//	    flow nhìn thấy được, sửa được, và nó không đổi giữa hai lần chạy.
//
//	(c) THỨ TỰ ID SẮP XẾP A-Z. Ổn định như (b) nhưng bắt người ta đặt tên bước
//	    theo bảng chữ cái để điều khiển thứ tự đọc. Đó là điều khiển bằng tác
//	    dụng phụ.
//
// Hệ quả trực tiếp của (b): merge KHÔNG CÓ `needs` là LỖI, không phải "gộp mọi
// bước trước". "Mọi bước trước" là một tập hợp không có thứ tự — muốn xếp nó
// thì lại rơi về (a) hoặc (c). Xem VanDeMerge.
//
// ============================================================================
// CÂU HỎI 2: MỘT BƯỚC TRONG SỐ ĐÓ HỎNG THÌ MERGE RA GÌ?
// ============================================================================
//
// Ra một khối có ĐỦ N MỤC, trong đó mục của bước hỏng là một dòng NÓI RÕ nó
// hỏng. Không bao giờ bỏ mục đi.
//
// Vì sao chuyện này xảy ra được: mặc định `on_failure = "stop"` thì bước hỏng
// làm dừng cả lượt và merge không bao giờ chạy. Nhưng bước khai
// `on_failure = "continue"` thì choDiTiep() cho các bước sau chạy tiếp — merge
// sẽ chạy, với một nguồn thiếu. Trạng thái `skipped` (điều kiện `when` không
// thoả, hoặc bước gỡ lại) cũng cho ra đúng tình huống đó.
//
// Bỏ mục đi thì hai lượt chạy — một lượt đủ ba nguồn, một lượt hỏng mất nguồn
// giữa — cho ra hai khối chữ mà nhìn vào KHÔNG PHÂN BIỆT ĐƯỢC cái nào thiếu.
// Agent tổng hợp ở bước sau sẽ viết một bản tổng kết tự tin dựa trên hai phần
// ba dữ liệu và không nói một câu nào về phần thiếu. Đúng lớp hỏng của lượt
// chạy #46 (`phai_co`) và #29 (placeholder còn sót): không sập, chỉ lặng lẽ
// gật đầu.
//
// Nên mọi mục đều có mặt, kể cả bước xong mà không trả về gì — "xong nhưng
// rỗng" và "hỏng" và "bị bỏ qua" là ba câu khác nhau, và cả ba đều khác với
// "không được nhắc tới".
package flow

import (
	"fmt"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/store"
)

// KhoaGopDauRa là khoá env chứa sẵn phần chữ đã gộp của một bước `merge`.
//
// Vì sao đi qua env chứ không tính trong do(): phần gộp cần TRẠNG THÁI của các
// bước nguồn, mà do() chỉ nhận env. Dựng ở cùng chỗ và cùng lúc với `outs` và
// với biến artifact (xem runWave) thì cả ba đi qua đúng một bộ lọc `doc_duoc`;
// tách ra thì sớm muộn có một đường hở.
//
// Đặt vào env còn được thêm một thứ miễn phí mà quan trọng: cauHoi() và KhoaIdem
// đều đọc env, nên khoá idempotency của bước merge tự cuốn theo nội dung các
// nguồn. Nguồn đổi kết quả thì merge chạy lại, không dùng lại khối cũ.
const KhoaGopDauRa = "gop_dau_ra"

// nhanGop là dòng tiêu đề của một mục trong khối gộp.
//
// Có rào hai đầu và có tên bước: bước sau (thường là agent) phải phân biệt được
// chữ của nguồn nào với chữ của nguồn nào, và phải thấy được ranh giới ngay cả
// khi bản thân đầu ra có chứa dấu `===`.
func nhanGop(id string) string { return "=== " + id + " ===" }

// GopDauRa dựng khối chữ mà một bước `merge` trả về.
//
// states/outputs là ảnh chụp trạng thái + kết quả của lượt chạy tại thời điểm
// bước này sắp chạy. Đã qua LocDocDuoc hay chưa đều cho ra cùng một kết quả:
// manhGop tự hỏi ChoDoc TRƯỚC khi nhìn vào giá trị, nên nó không phụ thuộc vào
// việc chỗ gọi có nhớ lọc hay không. Một cái rào chỉ đứng khi người khác nhớ
// dựng nó thì không phải một cái rào.
func GopDauRa(s Step, states, outputs map[string]string) string {
	phan := make([]string, 0, len(s.Needs))
	for _, id := range s.Needs {
		phan = append(phan, nhanGop(id)+"\n"+manhGop(s, id, states, outputs))
	}
	return strings.Join(phan, "\n\n")
}

// manhGop trả về phần thân của MỘT mục — kết quả thật, hoặc một câu nói rõ vì
// sao không có kết quả. Không bao giờ trả về chuỗi rỗng.
func manhGop(s Step, id string, states, outputs map[string]string) string {
	if !ChoDoc(s, id) {
		// `doc_duoc` chặn chính cái bước mình khai `needs` là một mâu thuẫn, và
		// VanDeMerge cảnh báo chuyện đó lúc kiểm. Ở đây thì cứ nói ra.
		return CauChan(id)
	}
	if out := strings.TrimSpace(outputs[id]); out != "" {
		return out
	}
	switch states[id] {
	case store.StepDone:
		return fmt.Sprintf("(bước %q xong nhưng KHÔNG để lại kết quả nào)", id)
	case store.StepFailed:
		return fmt.Sprintf("(bước %q HỎNG — không có kết quả để gộp)", id)
	case store.StepSkipped:
		return fmt.Sprintf("(bước %q bị BỎ QUA — không chạy)", id)
	default:
		// Không nên xảy ra: readySteps chỉ cho merge chạy khi mọi `needs` đã
		// xong hoặc đã hỏng-mà-continue. Vẫn nói ra thay vì để trống, vì "không
		// nên xảy ra" đã xảy ra rồi thì im lặng là cách tệ nhất để phát hiện.
		return fmt.Sprintf("(bước %q chưa chạy xong — không có kết quả để gộp)", id)
	}
}

// MoiTruongGop dựng phần env riêng của bước `merge`. Loại khác trả về nil và
// không thêm một khoá nào — cùng luật với MoiTruongArtifact.
func MoiTruongGop(s Step, states, outputs map[string]string) map[string]string {
	if s.Type != TypeMerge {
		return nil
	}
	return map[string]string{KhoaGopDauRa: GopDauRa(s, states, outputs)}
}

// VanDeMerge soi phần `merge` của cả flow.
//
// LỖI (chặn lưu, chặn chạy) cho những thứ chắc chắn hỏng hoặc không có nghĩa;
// CẢNH BÁO cho những lời khai tự mâu thuẫn nhưng vẫn chạy được.
func VanDeMerge(f Flow) []Problem {
	var ps []Problem
	loi := func(id, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: id, Msg: msg}) }
	nhac := func(id, msg string) { ps = append(ps, Problem{Flow: f.Name, Step: id, Msg: msg, Warn: true}) }

	for _, s := range f.Steps {
		if s.Type != TypeMerge {
			continue
		}

		// Không có nguồn thì không có gì để gộp, và "gộp mọi bước trước" là một
		// tập không có thứ tự — xem CÂU HỎI 1 ở đầu file.
		if len(s.Needs) == 0 {
			loi(s.ID, "bước merge cần `needs` là danh sách bước để gộp, theo ĐÚNG thứ tự muốn gộp "+
				"(ví dụ needs = [\"soi-1\", \"soi-2\"]). Không khai thì không có thứ tự nào để gộp.")
			continue
		}

		// Trùng tên trong `needs`: khối gộp sẽ có hai mục giống hệt nhau. Không
		// hỏng, nhưng chắc chắn không phải ý người viết.
		daThay := map[string]bool{}
		for _, id := range s.Needs {
			if daThay[id] {
				loi(s.ID, fmt.Sprintf("needs khai %q hai lần — khối gộp sẽ có hai mục giống hệt nhau", id))
			}
			daThay[id] = true
		}

		// `foreach`: mỗi lượt lặp sẽ gộp lại cùng một tập nguồn, ra N bản sao
		// giống nhau. Không có nghĩa nào cả.
		if s.ForEach != "" {
			loi(s.ID, "không dùng `foreach` với `merge` — mỗi lượt lặp sẽ gộp lại đúng cùng một tập nguồn")
		}

		// Khai `doc_duoc` mà chặn chính nguồn mình gộp: vẫn chạy, và khối gộp
		// vẫn đủ mục, nhưng mục đó là câu "không được phép đọc". Đây là lời khai
		// tự mâu thuẫn, không phải một lỗi kỹ thuật — nói ra, đừng chặn.
		for _, id := range s.Needs {
			if !ChoDoc(s, id) {
				nhac(s.ID, fmt.Sprintf("needs gộp bước %q nhưng `doc_duoc` không cho đọc bước đó — "+
					"mục của nó trong khối gộp sẽ là câu báo bị chặn, không phải kết quả", id))
			}
		}

		// Nguồn khai `on_failure = "continue"` KHÔNG phải lỗi — nó chính là ca
		// mà CÂU HỎI 2 nói tới. Nhắc một câu để người viết flow biết trước rằng
		// khối gộp có thể chứa một mục "HỎNG", chứ không phải ngã ngửa lúc đọc
		// bản tổng hợp.
		for _, id := range s.Needs {
			n, co := TimBuoc(f, id)
			if co && n.OnFailure == OnFailContinue {
				nhac(s.ID, fmt.Sprintf("nguồn %q khai on_failure = \"continue\" — nó hỏng thì lượt chạy "+
					"vẫn đi tiếp và mục của nó trong khối gộp sẽ là một dòng báo hỏng", id))
			}
		}
	}
	return ps
}
