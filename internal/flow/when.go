package flow

import (
	"fmt"
	"strconv"
	"strings"
)

// Điều kiện `when` — cho phép flow RẼ NHÁNH thay vì chỉ chạy thẳng.
//
// Cố ý KHÔNG nhúng một ngôn ngữ biểu thức đầy đủ. Flow là file người ta gửi cho
// nhau được; một ngôn ngữ mạnh trong đó là mở đường cho thứ khó lường. Ở đây chỉ
// có đúng một dạng:
//
//	<vế trái> <toán tử> <vế phải>
//
// Ví dụ:
//
//	when = "steps.kiem-thu.state == done"
//	when = "steps.ra-soat.output contains LOI"
//	when = "vars.moi_truong != prod"
//	when = "steps.tom-tat.output not-empty"
//
// Vế trái đọc được: steps.<id>.state · steps.<id>.output · vars.<tên>
// Toán tử: == != contains not-contains empty not-empty > < (số)

// Ctx là dữ liệu để đánh giá điều kiện.
type Ctx struct {
	Vars    map[string]string
	States  map[string]string // id bước -> trạng thái
	Outputs map[string]string // id bước -> kết quả
}

// Eval trả về điều kiện có đúng không. Chuỗi rỗng = luôn đúng.
//
// Sai cú pháp thì trả lỗi chứ KHÔNG âm thầm coi là false — người viết flow phải
// biết mình gõ sai, thay vì ngồi đoán vì sao bước không chạy.
func Eval(expr string, c Ctx) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, nil
	}

	// Toán tử một ngôi trước: "<vế trái> empty"
	for _, op := range []string{"not-empty", "empty"} {
		if strings.HasSuffix(expr, " "+op) {
			left := strings.TrimSpace(strings.TrimSuffix(expr, " "+op))
			v, err := resolve(left, c)
			if err != nil {
				return false, err
			}
			isEmpty := strings.TrimSpace(v) == ""
			if op == "empty" {
				return isEmpty, nil
			}
			return !isEmpty, nil
		}
	}

	// Toán tử hai ngôi. Xét cụm dài trước để "not-contains" không bị "contains" nuốt.
	for _, op := range []string{"not-contains", "contains", "==", "!=", ">=", "<=", ">", "<"} {
		idx := strings.Index(expr, " "+op+" ")
		if idx < 0 {
			continue
		}
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+len(op)+2:])

		lv, err := resolve(left, c)
		if err != nil {
			return false, err
		}
		rv, err := resolveLiteral(right, c)
		if err != nil {
			return false, err
		}

		switch op {
		case "==":
			return lv == rv, nil
		case "!=":
			return lv != rv, nil
		case "contains":
			return strings.Contains(strings.ToLower(lv), strings.ToLower(rv)), nil
		case "not-contains":
			return !strings.Contains(strings.ToLower(lv), strings.ToLower(rv)), nil
		default: // so sánh số
			a, err1 := strconv.ParseFloat(strings.TrimSpace(lv), 64)
			b, err2 := strconv.ParseFloat(strings.TrimSpace(rv), 64)
			if err1 != nil || err2 != nil {
				return false, fmt.Errorf("toán tử %q cần hai số, được %q và %q", op, lv, rv)
			}
			switch op {
			case ">":
				return a > b, nil
			case "<":
				return a < b, nil
			case ">=":
				return a >= b, nil
			default:
				return a <= b, nil
			}
		}
	}
	return false, fmt.Errorf("không hiểu điều kiện %q — dạng đúng: <vế trái> <toán tử> <vế phải>, ví dụ \"steps.a.state == done\"", expr)
}

// resolve đọc vế trái: steps.<id>.state | steps.<id>.output | vars.<tên>.
func resolve(ref string, c Ctx) (string, error) {
	switch {
	case strings.HasPrefix(ref, "steps."):
		rest := strings.TrimPrefix(ref, "steps.")
		i := strings.LastIndex(rest, ".")
		if i < 0 {
			return "", fmt.Errorf("thiếu phần sau id bước trong %q (cần .state hoặc .output)", ref)
		}
		id, field := rest[:i], rest[i+1:]
		switch field {
		case "state":
			return c.States[id], nil
		case "output":
			return c.Outputs[id], nil
		default:
			return "", fmt.Errorf("bước chỉ có .state và .output, không có .%s", field)
		}
	case strings.HasPrefix(ref, "vars."):
		return c.Vars[strings.TrimPrefix(ref, "vars.")], nil
	default:
		return "", fmt.Errorf("vế trái phải bắt đầu bằng steps. hoặc vars., được %q", ref)
	}
}

// resolveLiteral: vế phải là hằng, trừ khi cũng trỏ vào steps./vars.
func resolveLiteral(v string, c Ctx) (string, error) {
	if strings.HasPrefix(v, "steps.") || strings.HasPrefix(v, "vars.") {
		return resolve(v, c)
	}
	return strings.Trim(v, `"'`), nil
}
