package profile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/acl"
	"github.com/trantiendevweb/switch-agent-pro/internal/link"
	"github.com/trantiendevweb/switch-agent-pro/internal/paths"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
)

// ClonesRoot là nơi chứa bản sao dùng cho chạy song song. Để dưới thư mục có
// dấu chấm ở đầu để `List()` không nhầm nó là một provider.
func ClonesRoot() string { return filepath.Join(paths.AccountsRoot(), ".clones") }

// CloneDir là thư mục config của bản clone thứ n.
func CloneDir(prov, account string, n int) string {
	return filepath.Join(ClonesRoot(), prov, account, strconv.Itoa(n))
}

// Clone tạo N thư mục config biệt lập cho CÙNG một tài khoản, để chạy nhiều
// phiên song song.
//
// Vì sao không cho N tiến trình dùng chung một thư mục: chúng sẽ ĐUA NHAU GHI
// .claude.json và làm hỏng file (trust dialog nằm trong đó). Mỗi bản clone có
// file riêng nên không giẫm chân nhau.
//
// ⚠ ĐÃ ĐO (20/08/2026) — và kết quả tệ hơn phỏng đoán cũ.
//
// Nhà cung cấp XOAY VÒNG refresh token: mỗi lần refresh là một token mới, và
// token cũ CHẾT NGAY. Đo trên máy thật, hai bước:
//
//  1. Ép bản clone refresh → refresh token đổi (5d708911 → 1aa28b8c), hồ sơ
//     gốc vẫn giữ bản cũ.
//  2. Dựng một thư mục config tạm mang bản CŨ rồi chạy thử → nhận đúng câu đã
//     làm hỏng lượt #47: "Failed to authenticate: OAuth session expired and
//     could not be refreshed".
//
// Nghĩa là chép token ra N chỗ KHÔNG cần N tiến trình đua nhau mới hỏng: MỘT
// bản refresh là N-1 bản còn lại chết. Bản gốc cũng là một trong số đó.
//
// Đó là lý do khối đồng bộ ngược bên dưới không phải chuyện dọn dẹp cho gọn —
// nó là thứ giữ cho tài khoản còn đăng nhập được.
func Clone(a provider.Adapter, account string, copies int) ([]string, error) {
	base, ok := ResolveDir(a.Name(), account)
	if !ok {
		return nil, fmt.Errorf("không có %s:%s — tạo trước bằng: sagent them %s:%s",
			a.Name(), account, a.Name(), account)
	}
	if !a.HasToken(base) {
		return nil, fmt.Errorf("%s:%s chưa đăng nhập — chạy `sagent %s:%s` rồi /login trước",
			a.Name(), account, a.Name(), account)
	}

	// MANG TOKEN MỚI VỀ TRƯỚC KHI CHÉP ĐÈ.
	//
	// LỖI THẬT, đo 20/08/2026: `claude:phu` mất phiên giữa lượt chạy #47 với
	// "OAuth session expired and could not be refreshed", và người dùng phải
	// đăng nhập lại. Chuỗi sự kiện:
	//
	//  1. Lượt trước, bản clone tự refresh. Token mới nằm trong clone; hồ sơ gốc
	//     giữ token cũ — mà nhà cung cấp đã vô hiệu nó khi cấp token mới.
	//  2. `SyncBackTokens` CHỈ được gọi trong `ClonesClean`, tức chỉ khi người
	//     dùng chạy `sagent clean`. Không ai chạy, nên token mới nằm im.
	//  3. Lượt sau, `Clone` chép token GỐC (đã chết) đè lên clone (đang sống).
	//  4. Clone refresh bằng token đã bị vô hiệu → thất bại → mất phiên.
	//
	// Bước 3 mới là chỗ hỏng: `Clone` hồi sinh một token đã chết, đè lên đúng
	// bản còn dùng được. Gọi đồng bộ ngược ở đây thì token mới lan ra mọi bản
	// thay vì bị token cũ nuốt mất.
	//
	// Lỗi ở đây KHÔNG chặn việc chạy: đồng bộ hụt thì tệ nhất là quay về đúng
	// hành vi cũ, còn chặn `fleet` vì một phép dọn dẹp thì tệ hơn hẳn.
	if name, err := SyncBackTokens(a, account); err == nil && name != "" {
		_ = name // bên gọi (fleet/api) là chỗ có bus để báo; ở đây chỉ làm việc
	}

	var dirs []string
	for i := 1; i <= copies; i++ {
		dir := CloneDir(a.Name(), account, i)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return dirs, err
		}
		// Đây là chỗ NHÂN TOKEN ra N bản. `profile.Create` đã siết ACL từ trước,
		// nhưng chỗ này thì quên — mà nó còn nguy hiểm hơn: một hồ sơ hở là hở
		// một token, một kho clone hở là hở N bản. Trên Windows `0o755`/`0o600`
		// không bảo vệ gì (đã đo, xem internal/acl).
		_ = acl.Restrict(dir)
		// Phần dùng chung: nối link như hồ sơ thường.
		if _, err := LinkShared(a, dir); err != nil {
			return dirs, err
		}
		// File riêng: chép NGUYÊN VĂN. Đây là cùng một tài khoản nên bản sao
		// mang đúng danh tính đó là điều mong muốn — khác hẳn việc tạo tài
		// khoản mới (chỗ đó phải lọc qua whitelist).
		for _, name := range a.PrivateFiles() {
			src := filepath.Join(base, name)
			data, err := os.ReadFile(src)
			if err != nil {
				if os.IsNotExist(err) {
					continue // chưa có file đó thì thôi, không phải lỗi
				}
				// Thiếu quyền, file đang bị khoá… KHÔNG được im lặng bỏ qua:
				// clone sẽ chạy mà không có token, agent đăng nhập hụt, và người
				// dùng đi tìm nguyên nhân ở tận đâu.
				return dirs, fmt.Errorf("không đọc được %s: %w", src, err)
			}
			// PrivateFiles có thể là đường dẫn LỒNG (Cursor/auth.json), nên phải
			// tạo thư mục cha trước — os.WriteFile không tự tạo.
			dst := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return dirs, err
			}
			if err := os.WriteFile(dst, data, 0o600); err != nil {
				return dirs, err
			}
		}
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

// SyncBackTokens mang token đã được làm mới trong bản clone về hồ sơ gốc.
//
// Vì sao cần: `clone` chép token ra N thư mục riêng (bắt buộc, nếu không N tiến
// trình sẽ đua ghi). Nhưng khi một bản clone tự refresh, hồ sơ gốc KHÔNG hề
// biết — lần chạy sau vẫn dùng token cũ và có thể đã hết hạn. Đây không phải
// phỏng đoán về nhà cung cấp mà là hệ quả thẳng của thiết kế.
//
// Cách xử lý: tìm bản clone có file token MỚI NHẤT; nếu mới hơn bản ở hồ sơ gốc
// thì chép ngược về (có sao lưu).
//
// ĐÃ ĐO 20/08: nhà cung cấp CÓ xoay vòng refresh token. Vậy nên "lấy bản mới
// nhất" không còn là phỏng đoán — bản mới nhất đúng là bản duy nhất còn sống,
// mọi bản cũ hơn đã bị vô hiệu từ lúc nó được cấp.
//
// Trả về tên file đã mang về (rỗng nếu không có gì mới).
func SyncBackTokens(a provider.Adapter, account string) (string, error) {
	base, ok := ResolveDir(a.Name(), account)
	if !ok {
		return "", nil
	}
	root := filepath.Join(ClonesRoot(), a.Name(), account)
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", nil // chưa clone bao giờ — không có gì phải làm
	}

	// Chỉ quan tâm file token; danh tính/khoá/DB thì không mang về.
	tokenFile := a.PrivateFiles()
	if len(tokenFile) == 0 {
		return "", nil
	}
	name := tokenFile[0] // theo quy ước, file đầu tiên là file token

	basePath := filepath.Join(base, name)
	baseTime := time.Time{}
	if st, err := os.Stat(basePath); err == nil {
		baseTime = st.ModTime()
	}

	newest, newestTime := "", baseTime
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(root, e.Name(), name)
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if st.ModTime().After(newestTime) {
			newest, newestTime = p, st.ModTime()
		}
	}
	if newest == "" {
		return "", nil
	}

	data, err := os.ReadFile(newest)
	if err != nil {
		return "", err
	}

	// So NỘI DUNG, không chỉ dấu thời gian.
	//
	// Bản thân việc clone đã ghi file token mới toanh nên mtime của nó LUÔN mới
	// hơn bản gốc — chỉ dựa vào thời gian thì lần nào cũng tưởng là "đã refresh",
	// đè token vô cớ và đẻ ra một file .bak mỗi lần. Chỉ mang về khi token THẬT
	// SỰ khác đi.
	old := mustRead(basePath)
	if string(old) == string(data) {
		return "", nil
	}

	// Sao lưu bản cũ trước khi đè — đây là token, hỏng là phải đăng nhập lại.
	if len(old) > 0 {
		bak := basePath + ".bak-" + time.Now().Format("20060102-150405")
		if err := os.WriteFile(bak, old, 0o600); err != nil {
			// Đây là TOKEN. Không cứu được bản cũ thì đừng đè lên nó — hỏng là
			// người dùng phải đăng nhập lại, mà lúc đó chẳng còn gì để quay về.
			return "", fmt.Errorf("không sao lưu được token cũ ra %s, KHÔNG ghi đè: %w", bak, err)
		}
	}
	// Tên file tạm phải duy nhất: hai lần đồng bộ chạy cùng lúc mà chung một tên
	// thì cái này ghi đè cái kia rồi đổi tên ra một token lai.
	tmp := fmt.Sprintf("%s.tmp-%d", basePath, os.Getpid())
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, basePath); err != nil {
		return "", err
	}
	return name, nil
}

func mustRead(p string) []byte {
	b, _ := os.ReadFile(p)
	return b
}

// CleanClones xoá mọi bản clone của một tài khoản, AN TOÀN.
//
// Bắt buộc phải có hàm này: thư mục clone đầy junction trỏ về ~/.claude, nên
// một cú `rm -rf` hay `Remove-Item -Recurse` của người dùng có thể xuyên qua
// link xoá luôn dữ liệu thật. Ở đây dùng lại Remove() — gỡ từng link, kiểm
// sạch, rồi mới xoá.
func CleanClones(prov, account string) (int, error) {
	root := filepath.Join(ClonesRoot(), prov, account)
	// Kiểm chính `root` TRƯỚC khi ReadDir. Cùng lớp lỗi đã vá ở `Remove`, nhưng
	// sót ở tầng gốc: nếu root là junction trỏ ra ngoài thì ReadDir đi xuyên, và
	// mỗi thư mục con THẬT bên kia sẽ bị Remove xoá — trong khi đường dẫn vẫn
	// nằm gọn trong kho nên `insideStore` chẳng thấy gì bất thường.
	if isLink, _ := link.IsLink(root); isLink {
		if err := link.Unlink(root, true); err != nil {
			return 0, fmt.Errorf("%s là link, không gỡ được: %w", root, err)
		}
		return 0, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := Remove(filepath.Join(root, e.Name())); err != nil {
			return n, err
		}
		n++
	}
	// Thư mục cha giờ rỗng, xoá nốt cho gọn (không đệ quy).
	_ = os.Remove(root)
	return n, nil
}

// StartDetached chạy CLI ở chế độ nền, log đổ vào file, không chiếm terminal.
// workDir là chỗ agent làm việc (git worktree riêng, hoặc rỗng = thư mục hiện tại).
// Trả về PID. Cố ý KHÔNG Wait() — phiên phải sống tiếp sau khi lệnh này thoát.
func StartDetached(a provider.Adapter, dir string, args []string, logPath, workDir string) (int, error) {
	cmdPath, err := a.Command()
	if err != nil {
		return 0, err
	}
	f, err := os.Create(logPath)
	if err != nil {
		return 0, err
	}
	// Gỡ vỏ .cmd trước: vỏ batch cắt đối số nhiều dòng, agent chỉ nhận dòng đầu
	// rồi vẫn trả lời như thường. Xem vo_boc_windows.go.
	thuc, dau := GoiThat(cmdPath)
	c := exec.Command(thuc, append(append([]string{}, dau...), args...)...)
	c.Stdout, c.Stderr = f, f
	// Stdin nối vào NUL/dev-null chứ không để nil: Codex thấy stdin là ống dẫn
	// thì ngồi "Reading additional input from stdin..." chờ dữ liệu không bao
	// giờ tới. Cho nó EOF ngay.
	if devNull, err := os.Open(os.DevNull); err == nil {
		c.Stdin = devNull
		defer devNull.Close()
	}
	c.Env = append(filterEnv(os.Environ(), a.EnvVar()), a.EnvVar()+"="+dir)
	if workDir != "" {
		c.Dir = workDir
	} else if wd, err := os.Getwd(); err == nil {
		// Mặc định: thư mục hiện tại — agent làm trên đúng project bạn đang đứng.
		c.Dir = wd
	}
	if err := c.Start(); err != nil {
		f.Close()
		return 0, err
	}
	// Tiến trình con đã có bản sao handle của riêng nó, nên cha PHẢI đóng bản
	// của mình. Không đóng thì mỗi lần fleet là rò một file descriptor, và trên
	// Windows file log bị khoá tới khi tiến trình cha thoát.
	f.Close()
	return c.Process.Pid, nil
}
