// Package fleet điều phối nhiều phiên agent chạy song song.
//
// Gói này KHÔNG in ra stdout. Nó phát event (MASTER-PLAN mục 2c luật 3), để cả
// terminal lẫn dashboard cùng nhìn một nguồn sự thật.
package fleet

import (
	"fmt"
	"os"
	"path/filepath"

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

	// Nói thẳng hai điều, không giấu — và nói bằng event nên mặt nào cũng thấy.
	bus.Warnf("%d phiên trên MỘT tài khoản %s — tiêu hạn mức gấp %d lần.", o.Copies, addr, o.Copies)
	bus.Warnf("Token được chép ra %d chỗ; hành vi khi nhiều phiên cùng refresh CHƯA ĐO.", o.Copies)
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
