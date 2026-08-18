package profile

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Vỏ bọc .cmd/.bat CẮT ĐỐI SỐ NHIỀU DÒNG — chỉ dòng đầu tới được chương trình.
//
// Đo 18/08 bằng vỏ giả chạy qua đúng đường của StartDetached: gửi
// "DONG MOT\nDONG HAI\nDONG BA", vỏ .cmd nhận đúng "DONG MOT". Trên máy này
// `claude`, `grok`, `codex` đều là vỏ .cmd (PATHEXT có .CMD, không có .PS1),
// riêng `agy` là .exe thật — nên suốt hôm nay chỉ Antigravity nhận đủ prompt
// nhiều dòng, còn ba cái kia thì im lặng nhận một dòng rồi trả lời tự tin.
//
// Đây là hỏng LẶNG LẼ đúng nghĩa: không lỗi, không cảnh báo, agent vẫn đáp.
//
// GoiThat gỡ lớp vỏ npm ("%~dp0\...\index.js" %*) để gọi thẳng node với đúng
// script, nên đối số đi qua CreateProcess chứ không qua trình thông dịch batch.
// Không nhận ra kiểu vỏ nào thì trả nguyên đường cũ — thà chạy như trước còn
// hơn đoán sai rồi bật nhầm chương trình.
// Bám vào chuỗi .js TRONG NGOẶC KÉP, và lấy cái CUỐI CÙNG — dòng chuyển tiếp
// luôn nằm cuối vỏ npm. Bản đầu bám vào "dp0" và trượt: vỏ có nhiều chỗ dp0
// (`:find_dp0`, `SET dp0=%~dp0`) nên regex bám nhầm chỗ, mà `[^"]*` lại vắt qua
// cả xuống dòng. Trượt im lặng — vẫn build, vẫn chạy, chỉ là không gỡ được vỏ.
var reScriptNpm = regexp.MustCompile(`"([^"
]+\.(?:js|cjs|mjs))"`)

// tienTo là các cách vỏ npm viết "thư mục chứa tôi".
var tienTo = []string{`%~dp0\`, `%dp0%\`, `%~dp0`, `%dp0%`}

func GoiThat(cmdPath string) (string, []string) {
	ext := strings.ToLower(filepath.Ext(cmdPath))
	if ext != ".cmd" && ext != ".bat" {
		return cmdPath, nil
	}
	b, err := os.ReadFile(cmdPath)
	if err != nil {
		return cmdPath, nil
	}
	ms := reScriptNpm.FindAllSubmatch(b, -1)
	if len(ms) == 0 {
		return cmdPath, nil
	}
	rel := string(ms[len(ms)-1][1])
	for _, t := range tienTo {
		if strings.HasPrefix(rel, t) {
			rel = strings.TrimPrefix(rel, t)
			break
		}
	}
	rel = strings.TrimLeft(rel, `\/`)
	script := filepath.Join(filepath.Dir(cmdPath), filepath.FromSlash(rel))
	if _, err := os.Stat(script); err != nil {
		return cmdPath, nil
	}
	node, err := exec.LookPath("node")
	if err != nil {
		return cmdPath, nil
	}
	return node, []string{script}
}
