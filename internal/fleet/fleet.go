// Package fleet điều phối nhiều phiên agent chạy song song.
//
// Gói này KHÔNG in ra stdout. Nó phát event (MASTER-PLAN mục 2c luật 3), để cả
// terminal lẫn dashboard cùng nhìn một nguồn sự thật.
package fleet

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/trantiendevweb/switch-agent-pro/internal/events"
	"github.com/trantiendevweb/switch-agent-pro/internal/nhatky"
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
			// Đo 21/08 (ô Đ5): đua thật, hai bản cách nhau 16ms. ĐÚNG MỘT bản
			// thắng; bản thua chết sau 186ms với đúng câu dưới đây rồi tự ghi đè
			// file token của mình thành rỗng. Nói ra nguyên văn câu lỗi để người
			// vận hành đọc log là nhận ra ngay, thay vì đi tìm nguyên nhân khác.
			bus.Warnf("Đo thật: chỉ MỘT bản refresh thành công; bản thua dừng ngay với "+
				"\"OAuth session expired and could not be refreshed\" và để lại file token "+
				"RỖNG (expiresAt 0) — không phải lỗi mạng, không phải hết hạn mức.")
		}
	}
	if o.Worktree {
		bus.Infof("Mỗi phiên một git worktree riêng từ %s", repoRoot)
	} else {
		bus.Warnf("Cả %d phiên dùng CHUNG thư mục hiện tại — chúng có thể sửa đè file của nhau. Thêm --worktree để tách.", o.Copies)
	}

	// Dọn nhật ký cũ TRƯỚC khi bật, không phải sau: sau thì lượt nào cũng để
	// lại đỉnh dung lượng của chính nó, và người dùng Ctrl-C giữa chừng là
	// không bao giờ dọn. Nhật ký của phiên CÒN ĐANG CHẠY được giữ lại (xem
	// nhatky.Don) — lượt mới không được phép xoá tang chứng của lượt cũ còn sống.
	donNhatKy(db, bus)
	bus.Infof("Nhật ký phiên: %s — đọc lại bằng `sagent nhat-ky <số phiên>`", nhatky.Root())

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

		// NHẬT KÝ PHIÊN. Đường dẫn ra ngoài thư mục clone — xem internal/nhatky
		// để biết vì sao (tóm tắt: chỗ cũ bị lượt sau cắt trắng và bị `sagent
		// clean` xoá, nên đúng lúc cần đọc thì không còn gì).
		logPath := nhatky.Duong(a.Name(), account, i+1, time.Now())
		dau := nhatky.Dau{
			ThoiDiem: time.Now(), Addr: cloneAddr, HoSo: dir,
			ThuMuc: workDir, Lenh: argsPhien,
		}
		if err := nhatky.Tao(logPath, dau); err != nil {
			// Không ghi được nhật ký thì KHÔNG bật phiên. Bật mù là quay lại
			// đúng tình trạng ngày 21/08: phiên chạy, phiên chết, không ai biết
			// vì sao. Thà hỏng to ở đây — câu lỗi này nói thẳng quyền ghi file.
			bus.Failuref("phiên %d: không tạo được nhật ký %s: %v", i+1, logPath, err)
			if workDir != "" {
				_ = workspace.Remove(repoRoot, workDir)
			}
			continue
		}

		pid, err := profile.StartDetached(a, dir, argsPhien, logPath, workDir)
		if err != nil {
			// Chết trước khi agent kịp in chữ nào: chỉ sagent biết lý do, nên
			// sagent phải là người ghi nó xuống.
			_ = nhatky.GhiLoi(logPath, fmt.Sprintf("không bật được tiến trình: %v", err))
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
			//
			// Ghi vào NHẬT KÝ nữa, không chỉ ra bus: đây đúng là ca mà mất sổ
			// nghĩa là mất luôn đường tìm về file — nhật ký là thứ duy nhất còn
			// lại, nên nó phải tự nói được mình mồ côi.
			_ = nhatky.GhiLoi(logPath, fmt.Sprintf(
				"tiến trình chạy (PID %d) nhưng KHÔNG ghi được vào sổ: %v — "+
					"phiên này sẽ không hiện ra ở `sagent status`", pid, err))
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

// donNhatKy giữ thư mục nhật ký trong ngân sách, và NÓI RA những gì đã xoá.
//
// Xoá dữ liệu của người dùng trong im lặng là thứ dự án này cấm: người vận hành
// đi tìm nhật ký của lượt tuần trước mà không thấy, thì họ phải đọc được ở đâu
// đó rằng nó đã bị dọn theo ngân sách, chứ không phải nghi công cụ làm mất.
//
// Lỗi ở đây KHÔNG chặn fleet: dọn hụt thì tệ nhất là tốn đĩa, còn chặn một lượt
// chạy vì phép dọn dẹp thì tệ hơn hẳn.
func donNhatKy(db *store.DB, bus *events.Bus) {
	// Nhật ký của phiên CÒN SỐNG là thứ không được đụng vào. Hỏi sổ hụt thì
	// truyền danh sách rỗng — thà giữ ít hơn cần còn hơn xoá nhầm... nên khi
	// không biết phiên nào đang chạy thì thôi, không dọn.
	list, err := db.Running()
	if err != nil {
		return
	}
	var giu []string
	for _, s := range list {
		if s.Log != "" {
			giu = append(giu, s.Log)
		}
	}
	kq, err := nhatky.Don(giu, nhatky.SoFileToiDa, nhatky.TongByteToiDa)
	if err != nil {
		bus.Warnf("không dọn được nhật ký cũ ở %s: %v", nhatky.Root(), err)
		return
	}
	if kq.DaXoa > 0 {
		bus.Infof("Dọn nhật ký: xoá %d file cũ (%s), còn %d file (%s) — trần %d file / %s.",
			kq.DaXoa, coChu(kq.ByteXoa), kq.ConLai, coChu(kq.ByteCon),
			nhatky.SoFileToiDa, coChu(nhatky.TongByteToiDa))
	}
	if kq.KhongXoa > 0 {
		bus.Warnf("%d nhật ký quá ngân sách nhưng xoá không được (file đang bị khoá?) — %s.",
			kq.KhongXoa, nhatky.Root())
	}
}

// coChu đọc số byte thành chữ. Chỉ ba mốc: nhật ký một phiên hiếm khi tới GB,
// và cái trần đã chặn ở 256 MB.
func coChu(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
