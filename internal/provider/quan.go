package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Phát hiện agent CHẠY QUẨN — lặp đi lặp lại đúng một lời gọi tool mà không đi
// tới đâu.
//
// Ca đã nổ thật (lần chạy #21, 18/08, xem docs/DO-LUONG.md): tool `bash` của
// Grok chạy qua cmd.exe nên mọi lệnh Unix đều trượt, mà nó KHÔNG thích nghi —
// gọi đúng `ls -la` 399 lần liên tiếp rồi mới bị trần `--max-tool-rounds` chặn.
// Bước ấy vẫn được đánh dấu `done`, và người đọc màn hình không thấy gì bất
// thường. Cả hạn mức lẫn thời gian của một lượt bay hết vào đó.
//
// Việc đã làm sau #21 là hạ trần vòng tool xuống 60 cho Grok. Trần chỉ giới hạn
// THIỆT HẠI, nó không PHÁT HIỆN ra chuyện gì đang xảy ra: 60 vòng `ls -la` vẫn
// là 60 vòng vô ích, và bước vẫn báo xong. Đây là phần phát hiện.
//
// Bám đúng nguyên tắc của KetQua (xem ketqua.go): kết luận đọc từ CẤU TRÚC DỮ
// LIỆU — tên tool và tham số của từng lời gọi trong bản ghi có cấu trúc — chứ
// không dò chữ trong văn bản.

// TranLapLienTiep là số lần một lời gọi tool được phép lặp LIÊN TIẾP y hệt nhau
// trước khi bị coi là quẩn.
//
// VÌ SAO ĐẾM CHUỖI LIÊN TIẾP, KHÔNG ĐẾM TỔNG SỐ LẦN TRONG LƯỢT:
// một agent chạy `git status` vài lần trong một bước là hoàn toàn bình thường —
// trước khi sửa, sau mỗi commit. Nhưng giữa hai lần đó nó còn đọc file, còn sửa,
// còn commit, tức còn gọi tool KHÁC: chuỗi lặp bị ngắt. Đếm tổng số lần thì một
// bước làm việc thật với 5 commit cũng có thể vượt ngưỡng và bị vu oan. Chuỗi
// LIÊN TIẾP là thứ phân biệt được "lặp vì công việc có nhiều vòng" với "lặp vì
// không nghĩ ra gì khác".
//
// VÌ SAO LÀ 10: lặp lại liên tiếp y hệt nhau chỉ có đúng một lý do chính đáng là
// THỬ LẠI sau một lỗi tạm thời (mạng chập, file đang bị khoá). Thử lại 9 lần
// liên tiếp với ĐÚNG một tham số đã là quá tay, và lần thứ 10 không mang thêm
// thông tin gì so với lần thứ 9.
//
// Nói thẳng con số này ĐƯỢC BAO NHIÊU BẰNG CHỨNG: ca quẩn duy nhất đo được là
// 399 lần liên tiếp — cách ngưỡng rất xa, nên nó chứng minh ngưỡng KHÔNG BỎ SÓT
// ca thật. Mặt kia (có bắt oan lượt bình thường không) thì CHƯA ĐO ĐƯỢC: chưa
// đếm chuỗi lặp dài nhất trên một bản ghi lượt-chạy-bình-thường nào. Khi nào đo
// được thì chỉnh theo số, đừng chỉnh theo cảm giác.
const TranLapLienTiep = 10

// Quan kết luận agent trong lượt này có chạy quẩn hay không.
//
// Ba trạng thái, và trạng thái thứ ba mới là cái đáng giữ: KHÔNG BIẾT khác hẳn
// KHÔNG QUẨN. Nhập hai cái đó làm một là lại nói bừa — đúng thứ ketqua.go sinh ra
// để chấm dứt.
//
//	biet=false        -> bản ghi không mang lời gọi tool KÈM THAM SỐ, không kết luận được
//	biet=true, ly==""  -> đọc được, và không quẩn
//	biet=true, ly!=""  -> quẩn; ly là câu để đưa cho người đọc
func (k KetQua) Quan() (ly string, biet bool) {
	if !k.DemDuocTool {
		return "", false
	}
	if k.SoLanLap < TranLapLienTiep {
		return "", true
	}
	return fmt.Sprintf(
		"agent lặp lại lệnh %q %d lần liên tiếp — nghi chạy quẩn, KHÔNG phải lỗi code: sửa prompt hoặc công cụ của bước này",
		dongDau(k.LenhLap), k.SoLanLap), true
}

// demQuan đếm chuỗi lời gọi tool GIỐNG HỆT NHAU LIÊN TIẾP dài nhất trong một
// lượt. Dùng chung cho mọi provider vì phép đếm thì giống nhau; chỉ việc BÓC ra
// tên tool và tham số là khác nhau, nên phần đó nằm ở từng adapter.
type demQuan struct {
	truoc   string // chữ ký của lời gọi ngay trước
	dai     int    // độ dài chuỗi đang chạy
	chuKy   string // chữ ký của chuỗi dài nhất
	daiNhat int    // độ dài chuỗi dài nhất
	docDuoc bool   // đã bóc được ÍT NHẤT một lời gọi có tham số
}

// Them nạp một lời gọi tool. `ok=false` nghĩa là không bóc được chữ ký dùng
// được — bỏ qua, và KHÔNG tính là đã đọc được gì.
func (d *demQuan) Them(chuKy string, ok bool) {
	if !ok {
		// NGẮT chuỗi đang chạy, đừng lặng lẽ bỏ qua.
		//
		// Bỏ qua thì dãy A A ? A A thành một chuỗi 4 — trong khi dấu hỏi hoàn
		// toàn có thể là một lời gọi KHÁC mà ta chỉ không bóc nổi tham số. Đếm
		// gộp qua chỗ mù là vu oan, mà vu oan thì tệ hơn bỏ sót: người đọc mất
		// niềm tin vào lá chắn rồi tắt nó đi.
		d.truoc, d.dai = "", 0
		return
	}
	d.docDuoc = true
	if chuKy == d.truoc {
		d.dai++
	} else {
		d.truoc, d.dai = chuKy, 1
	}
	if d.dai > d.daiNhat {
		d.daiNhat, d.chuKy = d.dai, chuKy
	}
}

// KetLuan trả về chuỗi lặp dài nhất, kèm cờ CÓ ĐỌC ĐƯỢC lời gọi tool nào không.
func (d *demQuan) KetLuan() (lenh string, soLan int, docDuoc bool) {
	return d.chuKy, d.daiNhat, d.docDuoc
}

// chuKyTool dựng CHỮ KÝ của một lời gọi tool: tên tool CỘNG tham số.
//
// Tham số là phần bắt buộc, và đây là chỗ dễ làm sai nhất. Đếm theo mỗi TÊN TOOL
// thì một bước đọc 30 file khác nhau thành ra "gọi Read 30 lần" — vu oan cho một
// agent đang làm việc tử tế. Cái phân biệt quẩn với làm việc là THAM SỐ KHÔNG
// ĐỔI: cùng một lệnh, cùng một đường dẫn, lặp mãi.
//
// Không có tham số thì trả ok=false, tức KHÔNG KẾT LUẬN GÌ. Antigravity đúng ca
// này (xem ketqua_antigravity.go). Im lặng vì không biết vẫn hơn bắt nhầm.
func chuKyTool(ten string, thamSo map[string]any) (string, bool) {
	if len(thamSo) == 0 {
		return "", false
	}
	// Tool chạy shell: chữ ký LẤY MỖI `command`.
	//
	// Vì sao bỏ các trường còn lại: tool bash đi kèm một trường mô tả tự do do
	// model tự viết lại mỗi lần gọi. Nhét nó vào chữ ký thì 399 lần `ls -la` có
	// thể thành 399 chữ ký khác nhau và lá chắn rơi im lặng — đúng kiểu hỏng tệ
	// nhất, vì nó trông y như đang hoạt động. Danh tính của một lời gọi shell là
	// CÂU LỆNH, không phải lời người gọi tự thuyết minh về nó.
	if c, ok := thamSo["command"].(string); ok && strings.TrimSpace(c) != "" {
		return ten + " " + strings.TrimSpace(c), true
	}
	// Còn lại: gói cả bộ tham số. json.Marshal của map sắp xếp khoá nên cùng một
	// bộ tham số luôn ra cùng một chữ ký, dù provider đảo thứ tự.
	b, err := json.Marshal(thamSo)
	if err != nil {
		return "", false
	}
	return ten + " " + string(b), true
}
