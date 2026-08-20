// Package fleet điều phối nhiều phiên agent chạy song song.
//
// Gói này KHÔNG in ra stdout. Nó phát event (MASTER-PLAN mục 2c luật 3), để cả
// terminal lẫn dashboard cùng nhìn một nguồn sự thật.
package fleet

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/profile"
	"github.com/trantiendevweb/switch-agent-pro/internal/provider"
	"github.com/trantiendevweb/switch-agent-pro/internal/store"
	"github.com/trantiendevweb/switch-agent-pro/internal/workspace"
)

// Opts là các lựa chọn khi bật một hạm đội.
type Opts struct {
	Copies   int
	Worktree bool // mỗi phiên một git worktree riêng
}

// Result tóm tắt một lần bật hạm đội, để mặt gọi biết kết quả mà không phải
// đọc lại event.
type Result struct {
	Started int
	Wanted  int
	IDs     []int64
}

// FanOut chạy N phiên song song trên MỘT tài khoản.
//
// args là lệnh headless truyền cho CLI (ví dụ: -p "tóm tắt repo"). Phiên tương
// tác không chạy nền được vì cần bàn phím, nên fleet chỉ dành cho agent.
func FanOut(db *store.DB, bus *events.Bus, a provider.Adapter, account string, o Opts, args []string) (Result, error) {
	res := Result{Wanted: o.Copies}
	if o.Copies < 1 {
		o.Copies = 1
		res.Wanted = 1
	}
	// Provider giữ token ở chỗ dùng chung toàn máy thì KHÔNG chạy nhiều bản được:
	// N tiến trình sẽ giành nhau đúng một danh tính. Từ chối và nói rõ, thay vì
	// bật lên rồi để người dùng tự phát hiện lúc các phiên đá nhau.
	//
	// Đây đúng lớp sự cố đã trả giá: hai client Claude giành một device slot,
	// 1866 lần trong 18 tiếng, rồi rơi phiên remote (xem docs/DO-LUONG.md).
	if o.Copies > 1 && !a.TachDuocTaiKhoan() {
		return res, fmt.Errorf(
			"%s không chạy song song nhiều bản được: token của nó nằm ở chỗ dùng chung "+
				"toàn máy, không theo thư mục hồ sơ. Mỗi máy một tài khoản %s.\n"+
				"     Chạy một bản: bỏ --copies, hoặc --copies 1", a.Name(), a.Name())
	}

	if len(args) == 0 {
		return res, fmt.Errorf(`thiếu lệnh headless sau "--".
  Ví dụ: sagent fleet %s:%s --copies %d -- -p "tóm tắt repo này"`, a.Name(), account, o.Copies)
	}
	addr := a.Name() + ":" + account

	// Chuẩn bị worktree TRƯỚC khi bật phiên nào: thà hỏng lúc chưa chạy gì còn
	// hơn bật được 2 phiên rồi mới chết ở phiên thứ 3.
	var repoRoot string
	if o.Worktree {
		wd, err := os.Getwd()
		if err != nil {
			return res, err
		}
		root, ok := workspace.RepoRoot(wd)
		if !ok {
			return res, fmt.Errorf("--worktree cần một git repo, mà %s không phải", wd)
		}
		repoRoot = root
	}

	dirs, err := profile.Clone(a, account, o.Copies)
	if err != nil {
		return res, err
	}
	bus.Publish(events.Event{
		Type: events.ClonesCreated, Addr: addr,
		Msg:    fmt.Sprintf("đã chuẩn bị %d thư mục cấu hình riêng", len(dirs)),
		Detail: map[string]string{"copies": itoa(len(dirs))},
	})

	// Bổ sung cờ để lượt chạy này ĐO ĐƯỢC.
	//
	// `fleet` truyền args THÔ cho CLI con, khác đường flow (đi qua `argsChoBuoc`
	// nên có adapter dựng args). Người dùng gõ `-- -p "việc"` là agent chạy được,
	// nhưng thiếu cờ in bản ghi có cấu trúc thì `DocKetQua` không có gì để đọc và
	// phiên nào cũng về `lost`. Đo 20/08: 20 phiên liền "chết, chưa rõ vì sao",
	// tokens và chi phí đều "chưa đo", trong khi flow cùng tài khoản đo được đủ.
	//
	// Thêm chứ không chỉ cảnh báo: không thêm thì bốn mặt điều khiển đều mù, mà
	// mù im lặng là đúng thứ dự án này lập ra để chống. Nhưng thêm thì PHẢI NÓI —
	// nó đổi định dạng stdout của agent, và người dùng có quyền biết.
	if them := provider.CoConThieu(a, args); len(them) > 0 {
		args = append(args, them...)
		bus.Warnf("Đã thêm %s để lượt chạy này đo được — thiếu nó thì mọi phiên về \"chết, chưa rõ vì sao\".",
			strings.Join(them, " "))
	}

	// Nói thẳng hai điều, không giấu — và nói bằng event nên mặt nào cũng thấy.
	bus.Warnf("%d phiên trên MỘT tài khoản %s — tiêu hạn mức gấp %d lần.", o.Copies, addr, o.Copies)
	// Câu về token phải SUY TỪ ADAPTER, không nói bừa. `profile.Clone` chỉ chép
	// những gì `PrivateFiles()` khai; provider khai rỗng (Antigravity giữ token
	// trong Windows Credential Manager) thì KHÔNG có file nào được chép, và câu
	// "chép ra N chỗ" là một câu SAI SỰ THẬT in ra mỗi lần chạy.
	//
	// Rẽ theo năng lực của adapter chứ không theo tên provider: lõi không được
	// có nhánh `if provider == "antigravity"` (luật ở internal/provider/adapter.go).
	if len(a.PrivateFiles()) == 0 {
		bus.Warnf("Token của %s nằm ở kho dùng chung toàn máy, không chép đi đâu; mọi phiên dùng chung một danh tính.", a.Name())
	} else {
		bus.Warnf("Token được chép ra %d chỗ. Nhà cung cấp XOAY VÒNG refresh token "+
			"(đo 20/08), nên bản nào refresh trước thì các bản kia chết — công cụ tự "+
			"mang bản mới nhất về hồ sơ gốc trước mỗi lần chép.", o.Copies)
		// Đồng bộ ngược chỉ cứu được GIỮA CÁC LƯỢT, không cứu được TRONG LÚC CHẠY:
		// hai bản đang chạy cùng lúc, một bản tới mốc refresh và xoay token đi, thì
		// bản kia cầm token đã chết ngay giữa việc — không có chỗ nào để chen vào
		// mà đồng bộ. Nói thẳng chuyện đó, vì nó quyết định cách chia việc.
		if o.Copies > 1 {
			bus.Warnf("%d bản CÙNG CHẠY trên một tài khoản: bản nào tới mốc refresh trước "+
				"sẽ giết token của các bản kia GIỮA CHỪNG, và đồng bộ ngược không chen vào "+
				"được lúc đó. Lượt chạy dài thì nên chia cho NHIỀU TÀI KHOẢN thay vì nhiều "+
				"bản của một tài khoản.", o.Copies)
		}
	}
	if o.Worktree {
		bus.Infof("Mỗi phiên một git worktree riêng từ %s", repoRoot)
	} else {
		bus.Warnf("Cả %d phiên dùng CHUNG thư mục hiện tại — chúng có thể sửa đè file của nhau. Thêm --worktree để tách.", o.Copies)
	}

	for i, dir := range dirs {
		name := fmt.Sprintf("%s-%d", account, i+1)
		cloneAddr := fmt.Sprintf("%s#%d", addr, i+1)

		workDir := ""
		if o.Worktree {
			wt, err := workspace.Add(repoRoot, name)
			if err != nil {
				bus.Failuref("phiên %d: %v", i+1, err)
				continue
			}
			workDir = wt
			bus.Publish(events.Event{
				Type: events.WorktreeAdded, Addr: cloneAddr,
				Msg:    "nhánh sagent/" + name,
				Detail: map[string]string{"path": wt, "branch": "sagent/" + name},
			})
		}

		// Khai TƯỜNG MINH thư mục làm việc: ở git worktree thì `.git` là file con
		// trỏ, có provider dò workspace hụt và trả "chưa có repository nào được
		// mở" — mà bước vẫn tính là xong. Xem docs/DO-LUONG.md.
		var truoc []string
		if workDir != "" {
			truoc = append(truoc, a.ArgsThuMuc(workDir)...)
		}
		truoc = append(truoc, a.ArgsHoSo(dir)...)
		argsPhien := args
		if len(truoc) > 0 {
			argsPhien = append(truoc, args...)
		}

		logPath := filepath.Join(dir, "fleet.log")
		pid, err := profile.StartDetached(a, dir, argsPhien, logPath, workDir)
		if err != nil {
			bus.Failuref("phiên %d: %v", i+1, err)
			if workDir != "" {
				_ = workspace.Remove(repoRoot, workDir)
			}
			continue
		}
		id, err := db.AddSession(store.Session{
			Provider: a.Name(), Account: account, Clone: i + 1,
			Dir: dir, PID: pid, Log: logPath, Worktree: workDir,
		})
		if err != nil {
			// Tiến trình đã chạy nhưng không ghi được sổ: nói rõ, đừng im lặng.
			bus.Failuref("phiên %d chạy rồi (PID %d) nhưng không ghi được vào sổ: %v", i+1, pid, err)
			continue
		}
		bus.Publish(events.Event{
			Type: events.SessionStarted, Addr: cloneAddr, SessionID: id,
			Msg: fmt.Sprintf("PID %d", pid),
			Detail: map[string]string{
				"pid": itoa(pid), "log": logPath, "worktree": workDir,
			},
		})
		res.Started++
		res.IDs = append(res.IDs, id)
	}
	return res, nil
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
