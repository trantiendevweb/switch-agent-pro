// Package drift phát hiện CLI bên dưới đã đổi phiên bản.
//
// Vì sao dự án này cần: mọi khẳng định trong docs/DO-LUONG.md đều gắn với MỘT
// phiên bản CLI cụ thể — "đã đo trên codex 0.147.0: `codex exec` chạy không
// tương tác", "đã đo: `claude -p` in kết quả ra stdout". Người dùng gõ
// `npm i -g @openai/codex` một cái là toàn bộ số đo đó thành lời đồn, mà không
// có gì báo.
//
// Đây là phần "hạn sử dụng" của một phép đo. Không có nó thì "đã đo" chỉ đúng
// vào đúng ngày đo.
package drift

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/acl"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
)

// Moc là phiên bản CLI đã được ghi nhận, cùng lúc ghi.
type Moc struct {
	Version string    `json:"version"`
	Duong   string    `json:"duong"`
	GhiLuc  time.Time `json:"ghi_luc"`
}

type soGhi struct {
	Moc map[string]Moc `json:"moc"`
}

func duongDan() string { return filepath.Join(paths.AccountsRoot(), "provider-drift.json") }

// doc đọc sổ mốc. Trả lỗi khi file CÓ nhưng không đọc được — không được im lặng
// coi như sổ rỗng.
//
// Đã nổ thật khi kiểm trên máy: sửa file bằng `Set-Content -Encoding UTF8` của
// PowerShell 5.1 (nó thêm BOM), `json.Unmarshal` hỏng, bản cũ NUỐT lỗi và trả về
// sổ rỗng — `verify` bèn báo "ghi mốc đầu tiên" rồi GHI ĐÈ sổ. Tức là mọi mốc
// biến mất, phát hiện drift thành vô dụng, và không một dòng cảnh báo.
//
// Tệ hơn: lần ghi đè đó xoá luôn cái BOM, nên bằng chứng của lỗi cũng mất. Kiểm
// lại file sau khi chạy thì thấy nó "bình thường".
func doc() (soGhi, error) {
	s := soGhi{Moc: map[string]Moc{}}
	b, err := os.ReadFile(duongDan())
	if os.IsNotExist(err) {
		return s, nil // chưa có sổ là chuyện bình thường
	}
	if err != nil {
		return s, err
	}
	// Trình soạn thảo và cmdlet trên Windows hay thêm BOM. Bỏ qua nó thì đọc
	// được; coi nó là hỏng thì người dùng chẳng làm gì sai mà bị chặn.
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("%s hỏng, không đọc được: %w", duongDan(), err)
	}
	if s.Moc == nil {
		s.Moc = map[string]Moc{}
	}
	return s, nil
}

func ghi(s soGhi) error {
	if err := os.MkdirAll(filepath.Dir(duongDan()), 0o755); err != nil {
		return err
	}
	_ = acl.Restrict(filepath.Dir(duongDan()))
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(duongDan(), append(b, '\n'), 0o600)
}

// KetQua là kết luận cho một provider.
type KetQua struct {
	Provider string
	OK       bool
	Chi      string // câu để in cho người dùng
}

// Kiem so phiên bản hiện tại với mốc đã ghi.
//
// chapNhan=false: chỉ báo. Mốc cũ GIỮ NGUYÊN, nên cảnh báo còn hiện ở những lần
// chạy sau. Đây là chủ ý: tự cập nhật mốc thì cảnh báo hiện đúng một lần rồi
// biến mất, và cái đã trôi thì vẫn trôi — đúng kiểu hỏng im lặng mà cả dự án này
// lập ra để chống.
//
// chapNhan=true: ghi đè mốc, tức người dùng nói "tôi đã đo lại".
func Kiem(provider, versionHienTai, duong string, chapNhan bool) KetQua {
	s, err := doc()
	if err != nil {
		// Không tự dựng lại sổ. Ghi đè một cái sổ hỏng nghĩa là xoá sạch mốc của
		// MỌI provider — im lặng, và đúng lúc không ai ngờ.
		return KetQua{provider, false, err.Error() +
			" — sửa hoặc xoá file đó rồi chạy lại; xoá là mất hết mốc, phải đo lại từ đầu"}
	}
	cu, daCo := s.Moc[provider]

	moi := Moc{Version: versionHienTai, Duong: duong, GhiLuc: time.Now()}

	switch {
	case !daCo:
		if err := ghi(luu(s, provider, moi)); err != nil {
			return KetQua{provider, false, "không ghi được mốc: " + err.Error()}
		}
		return KetQua{provider, true, fmt.Sprintf("ghi mốc đầu tiên: %s", versionHienTai)}

	case cu.Version == versionHienTai:
		return KetQua{provider, true, fmt.Sprintf("%s — không đổi từ %s",
			versionHienTai, cu.GhiLuc.Format("02/01/2006"))}

	case chapNhan:
		if err := ghi(luu(s, provider, moi)); err != nil {
			return KetQua{provider, false, "không ghi được mốc: " + err.Error()}
		}
		return KetQua{provider, true, fmt.Sprintf("đã nhận mốc mới: %s (trước là %s)",
			versionHienTai, cu.Version)}

	default:
		return KetQua{provider, false, fmt.Sprintf(
			"CLI ĐÃ ĐỔI: %s → %s (mốc ghi %s). Số đo cũ trong docs/DO-LUONG.md gắn với "+
				"bản trước, cần đo lại. Đo xong thì: sagent verify --chap-nhan",
			cu.Version, versionHienTai, cu.GhiLuc.Format("02/01/2006"))}
	}
}

func luu(s soGhi, provider string, m Moc) soGhi {
	s.Moc[provider] = m
	return s
}
