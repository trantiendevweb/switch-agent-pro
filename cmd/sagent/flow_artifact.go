// `sagent flow artifacts <#> [đường-dẫn] [--tu N]` — xem file một lượt chạy để lại.
//
// Hai câu hỏi, một lệnh, phân biệt bằng việc CÓ đường dẫn hay không:
//
//	sagent flow artifacts 51                  → lượt #51 để lại những gì
//	sagent flow artifacts 51 viet/ban-va.diff → trong file đó có gì
//	sagent flow artifacts 51 viet/ban-va.diff --tu 65536  → khúc tiếp theo
//
// Đường dẫn gõ vào phải là chuỗi ở cột "Đường dẫn" của bảng liệt kê. Mọi thứ
// khác bị từ chối ở tầng hợp đồng — xem flow.DuongDanArtifactAnToan.
package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/trantiendevweb/switch-agent-pro/internal/api"
)

func flowArtifacts(idStr string, args []string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		fail(fmt.Errorf("số lần chạy phải là số, được %q", idStr))
	}

	duong, tu := "", int64(0)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--tu":
			if i+1 >= len(args) {
				fail(fmt.Errorf("--tu cần một con số. Ví dụ: --tu 65536"))
			}
			n, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				fail(fmt.Errorf("--tu phải là số byte, được %q", args[i+1]))
			}
			tu = n
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fail(fmt.Errorf("không hiểu cờ %q — chỉ có --tu <byte>", args[i]))
			}
			duong = args[i]
		}
	}

	a, done := open()
	defer done()

	if duong == "" {
		kho, err := a.FlowArtifacts(id)
		if err != nil {
			fail(err)
		}
		done()
		inBangArtifact(kho)
		return
	}

	nd, err := a.FlowArtifactDoc(id, duong, tu)
	if err != nil {
		fail(err)
	}
	done()
	inNoiDungArtifact(id, nd)
}

func inBangArtifact(kho api.KhoArtifact) {
	fmt.Println()
	fmt.Printf("  Artifact của lượt chạy #%d — flow %s\n", kho.RunID, kho.Flow)
	fmt.Println()
	if len(kho.File) == 0 {
		fmt.Println("  (lượt chạy này không để lại file nào)")
		fmt.Println()
		fmt.Println("  Bước để lại file thì phải khai `artifact` trong flows.toml.")
		fmt.Println("  Xem: sagent flow show <tên>")
		fmt.Println()
		return
	}
	fmt.Printf("   %-14s %-26s %10s  %s\n", "BƯỚC", "TÊN", "BYTE", "ĐƯỜNG DẪN")
	for _, f := range kho.File {
		ten := f.Ten
		if ten == "" {
			// Nói ra chứ không để trống: ô trống đọc là "chưa điền", còn đây là
			// một sự thật — file có trên đĩa mà không bước nào hứa nó.
			ten = "(không bước nào khai)"
		}
		danh := ""
		if f.NhiPhan {
			danh = "  [nhị phân]"
		}
		fmt.Printf("   %-14s %-26s %10d  %s%s\n", f.Buoc, truncate(ten, 26), f.Byte, f.Duong, danh)
	}
	fmt.Println()
	fmt.Printf("  %d file · %d byte · %s\n", len(kho.File), kho.TongByte, kho.Dir)
	if kho.ThieuDinhNghia {
		fmt.Println("  ! không đọc được định nghĩa flow của lượt chạy này nữa — cột TÊN trống " +
			"vì KHÔNG TRA ĐƯỢC, không phải vì không bước nào khai")
	}
	fmt.Println()
	fmt.Printf("  Đọc một file: sagent flow artifacts %d %s\n", kho.RunID, kho.File[0].Duong)
	fmt.Println()
}

func inNoiDungArtifact(runID int64, nd api.NoiDungArtifact) {
	fmt.Println()
	fmt.Printf("  %s — %d byte\n", nd.Duong, nd.Byte)
	if nd.NhiPhan {
		fmt.Println()
		fmt.Println("  (file nhị phân — không in ra được)")
		fmt.Println()
		return
	}
	fmt.Println()
	fmt.Println(nd.Chu)
	// Câu này in ra ở CUỐI, sau nội dung: người đọc cuộn tới đây là lúc họ tưởng
	// mình đã đọc hết. Đặt nó ở đầu thì nó trôi mất trong 64 KiB chữ.
	if nd.BiCat {
		fmt.Println()
		fmt.Printf("  ── BỊ CẮT: mới đọc %d/%d byte, còn %d byte nữa ──\n",
			nd.Tu+nd.DocByte, nd.Byte, nd.ConLai)
		fmt.Printf("  Đọc tiếp: sagent flow artifacts %d %s --tu %d\n",
			runID, nd.Duong, nd.Tu+nd.DocByte)
	}
	fmt.Println()
}
