package provider

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// hoiVersion chạy `<cli> --version` và lấy dòng đầu.
//
// Vì sao phải CHẠY chứ không băm file: trên Windows, `claude` và `codex` là shim
// `.cmd` trỏ tới gói cài ở chỗ khác (đo được: shim claude là file .cmd 1486 byte
// sửa tay). Nâng cấp CLI **không** làm đổi shim, nên băm shim là đo nhầm thứ.
//
// Có hạn giờ: một CLI treo không được phép làm `sagent verify` treo theo. Đã đo
// trên máy dev — claude 499 ms, codex 769 ms, nên 10 giây là rộng rãi.
func hoiVersion(cmdPath string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, cmdPath, args...).Output()
	if ctx.Err() != nil {
		return "", fmt.Errorf("%s %s quá 10 giây, bỏ qua", cmdPath, strings.Join(args, " "))
	}
	if err != nil {
		return "", err
	}
	for _, d := range strings.Split(string(out), "\n") {
		if d = strings.TrimSpace(d); d != "" {
			return d, nil
		}
	}
	return "", fmt.Errorf("%s không in ra phiên bản nào", cmdPath)
}
