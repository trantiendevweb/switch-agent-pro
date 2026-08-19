// Chia flow thành CÁC ĐỢT chạy — không chạy gì cả.
//
// Vòng chạy thật trong runner.execute lặp { tìm bước sẵn sàng → chạy song song
// → chờ hết đợt }. Cách chia đợt đó là thứ quyết định "bước nào chạy cùng lúc
// với bước nào", tức là thứ người dùng cần biết TRƯỚC khi bấm chạy. Nhưng nó
// nằm khoá cứng trong execute, cạnh những dòng ghi DB và bật agent, nên muốn
// biết kế hoạch thì chỉ có một cách: chạy thật.
//
// Đo 19/08: ba lượt chạy thật (#30, #32, #33) được bấm chỉ để xem cổng kiểm nói
// gì — mỗi lượt đốt hạn mức thuê bao và đẻ ra một lượt rác phải huỷ tay.
//
// Dot() tách phần chia đợt ra, dùng lại ĐÚNG runState.readySteps mà runner
// dùng. Dùng lại chứ không chép: chép ra thì hai bản lệch nhau lúc nào không
// hay, và bản lệch ấy lại đúng là bản người dùng đọc để ra quyết định.
package flow

import "github.com/trantiendevweb/switch-agent-pro/internal/store"

// DotChay là MỘT ĐỢT: mọi bước trong đợt chạy SONG SONG với nhau.
type DotChay struct {
	So   int    // số thứ tự đợt, đếm từ 1
	Buoc []Step // các bước của đợt

	// ChoDuyet = đợt này là một rào duyệt. Lượt chạy THẬT dừng lại ở đây và
	// không đi tiếp cho tới khi có người bấm duyệt; các đợt sau chỉ xảy ra nếu
	// bước này được duyệt.
	ChoDuyet bool
}

// Dot chia các bước của f thành các đợt chạy, theo đúng luật của bộ thực thi:
//
//   - một bước sẵn sàng khi mọi bước nó `needs` đã xong;
//   - mọi bước sẵn sàng trong cùng một vòng thành MỘT đợt, chạy song song;
//   - bước `approve` KHÔNG bao giờ nằm chung đợt với bước khác — nó đứng riêng
//     một đợt, đánh dấu ChoDuyet, đúng như runner dựng rào rồi trả quyền cho
//     con người.
//
// Đây là kế hoạch của trường hợp MỌI BƯỚC ĐỀU CHẠY XONG. Điều kiện `when` không
// thoả, bước hỏng với on_failure=stop, hay người từ chối ở rào duyệt đều làm
// lượt chạy thật ngắn hơn kế hoạch này. Nói ra chỗ nào không chắc là việc của
// người gọi — ở đây chỉ trả về sự thật "nếu suôn sẻ thì thứ tự là thế này".
func Dot(f Flow) ([]DotChay, error) {
	// Chu trình thì không có kế hoạch nào cả — bắt ở đây thay vì để vòng lặp
	// dưới im lặng bỏ sót những bước không bao giờ sẵn sàng.
	if _, err := Order(f); err != nil {
		return nil, err
	}

	st := &runState{states: map[string]string{}, outputs: map[string]string{}}
	var dots []DotChay
	for {
		ready, _ := st.readySteps(f.Steps)
		if len(ready) == 0 {
			break
		}

		var work []Step
		for _, s := range ready {
			if s.Type == TypeApprove {
				continue
			}
			work = append(work, s)
		}

		// Còn việc chạy được thì đợt này là chúng — approve chờ đợt sau, y như
		// runner: rào chỉ dựng khi cả đợt không còn gì khác để làm.
		if len(work) > 0 {
			for _, s := range work {
				st.set(s.ID, store.StepDone, "")
			}
			dots = append(dots, DotChay{So: len(dots) + 1, Buoc: work})
			continue
		}

		// Cả đợt chỉ còn approve: rào ở cái đầu tiên. Runner DỪNG tại đây; kế
		// hoạch thì đi tiếp để người đọc thấy phần "sau khi duyệt" là những gì.
		s := ready[0]
		st.set(s.ID, store.StepDone, "")
		dots = append(dots, DotChay{So: len(dots) + 1, Buoc: []Step{s}, ChoDuyet: true})
	}
	return dots, nil
}
