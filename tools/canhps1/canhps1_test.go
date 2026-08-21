// Package canhps1 khong co ma san pham. No chi giu MOT bai kiem: moi file .ps1
// trong kho phai la ASCII thuan.
//
// VI SAO (bay D1 trong docs/VAN-HANH-VPS.md, da can that): PowerShell 5.1 doc
// file .ps1 UTF-8 KHONG BOM bang CP1252. Dau gach ngang dai nam trong mot CHUOI
// bien thanh nhay cong dong va KET THUC CHUOI SOM, tuc hong cu phap. Vi cac
// script nay chay AN, khong ai thay loi - lan truoc mat ca 4 server lan
// dashboard, va nhin tu ngoai giong het hai su co khac xay ra cung luc.
//
// Bai kiem nay bien mot cai bay im lang thanh mot dong test do.
package canhps1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPS1KhongBOMPhaiLaASCIIThuan(t *testing.T) {
	goc := filepath.Join("..", "..")
	var daKiem, coBOM int

	err := filepath.Walk(goc, func(duong string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil // thu muc doc khong duoc thi bo qua, khong lam do bai kiem
		}
		if fi.IsDir() {
			if fi.Name() == ".git" || fi.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(duong), ".ps1") {
			return nil
		}
		daKiem++

		b, err := os.ReadFile(duong)
		if err != nil {
			t.Errorf("khong doc duoc %s: %v", duong, err)
			return nil
		}
		// CO BOM UTF-8 thi KHONG phai kiem gi ca. Bay D1 noi RO la file
		// "UTF-8 KHONG BOM": thieu BOM thi PowerShell 5.1 doan nham ra CP1252.
		// Co BOM la no doc dung UTF-8, nen ky tu tieng Viet hoan toan an toan.
		// BOM chinh la CACH CHUA cai bay nay, khong phai cai bay.
		if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
			coBOM++
			return nil
		}

		// Bao cao dong+cot cua byte dau tien vi pham: "co ky tu la o dau do"
		// khong du de ai di sua.
		dong, cot := 1, 1
		for _, c := range b {
			if c == '\n' {
				dong, cot = dong+1, 1
				continue
			}
			if c > 127 {
				t.Errorf("%s:%d:%d co byte ngoai ASCII (0x%02X) ma file KHONG CO BOM UTF-8. "+
					"PowerShell 5.1 se doc file nay bang CP1252, va ky tu la trong mot CHUOI "+
					"lam hong cu phap IM LANG (bay D1). Sua bang MOT trong hai cach: dung "+
					"'-' thuong thay gach ngang dai va nhay thang thay nhay cong, HOAC luu "+
					"lai file kem BOM UTF-8.",
					duong, dong, cot, c)
				return nil // mot loi moi file la du de di sua
			}
			cot++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Bai kiem quet ma khong thay file nao thi no dang xanh vi RONG, khong phai
	// vi sach - dung kieu "hong ma den xanh" ma ca du an nay canh chung.
	if daKiem == 0 {
		t.Fatal("khong quet duoc file .ps1 nao — duong dan goc sai, bai kiem nay dang vo nghia")
	}
	t.Logf("da kiem %d file .ps1 (%d file co BOM nen duoc mien)", daKiem, coBOM)
}
