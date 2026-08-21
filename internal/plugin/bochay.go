package plugin

// Cầu nối giữa plugin và internal/flow.
//
// Vì sao plugin cắm vào flow chứ không mọc thành hệ thứ hai: flow đã có mô hình
// node, có kiểm tra DAG, có retry/timeout, có on_failure, có {{bien}}, có hàng
// rào doc_duoc và phai_co. Một đường chạy plugin riêng sẽ phải làm lại từng thứ
// đó, và mỗi thứ làm lại là một thứ có thể làm khác đi — người dùng sẽ phải học
// hai bộ luật cho cùng một câu hỏi "bước này hỏng thì sao".
//
// Cố ý KHÔNG import internal/flow ở đây: flow khai interface, gói này chỉ tình cờ
// khớp chữ ký. Nhờ vậy flow vẫn kiểm tra được mà không kéo theo os/exec, và test
// của flow vẫn chạy được không cần biên dịch một plugin nào.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// BoChay chạy plugin theo TÊN, cho một thư mục dự án.
type BoChay struct {
	// Dir là thư mục dự án — dùng để tìm .sagent/plugins, và là thư mục làm việc
	// cấp cho plugin nào KHAI quyền thu-muc-lam-viec.
	Dir string

	Timeout time.Duration

	// Ghi nhận `ghi_chu` của plugin (phần cho người đọc, không phải phần chuyền
	// cho bước sau). nil = bỏ. Xem KetQuaChay.
	Ghi func(string)

	// DocSecret để test không phải đụng kho key thật. nil = kho mặc định.
	DocSecret func(id string) (string, error)
}

// GoiPlugin bật plugin, chạy một lượt, rồi đóng.
//
// Chữ ký khớp flow.PluginRunner. Trả về CHUỖI chứ không phải struct: bước flow
// chỉ chuyền output sang bước sau, và mọi thứ khác plugin muốn nói thì đi đường
// ghi_chu.
func (b *BoChay) GoiPlugin(ctx context.Context, ten, vao string, thamSo map[string]string) (string, error) {
	m, err := b.Tim(ten)
	if err != nil {
		return "", err
	}
	c, err := Mo(ctx, m, TuyChon{ThuMuc: b.Dir, Timeout: b.Timeout, DocSecret: b.DocSecret})
	if err != nil {
		return "", err
	}
	defer c.Dong()

	kq, err := c.Chay(ctx, vao, thamSo)
	if err != nil {
		return "", fmt.Errorf("plugin %q: %w", ten, err)
	}
	if kq.GhiChu != "" && b.Ghi != nil {
		b.Ghi(fmt.Sprintf("plugin %s: %s", ten, kq.GhiChu))
	}
	return kq.Ra, nil
}

// Tim tra một plugin theo tên.
//
// Không thấy thì thông điệp phải nói ĐỦ BA thứ: tên không có, có những tên nào,
// và manifest nào đang hỏng. Thiếu vế cuối là ca hay gặp nhất mà lại khó đoán
// nhất — plugin nằm đúng chỗ, gõ đúng tên, nhưng manifest sai một dòng nên nó
// không có trong bảng, và người dùng không có cách nào biết.
func (b *BoChay) Tim(ten string) (Manifest, error) {
	ds, nguon, loi := Nap(b.Dir)
	if m, co := ds[ten]; co {
		return m, nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "không có plugin %q", ten)
	if co := Ten(ds); len(co) > 0 {
		fmt.Fprintf(&sb, " — đang có: %s", strings.Join(co, ", "))
	} else {
		sb.WriteString(" — chưa cài plugin nào")
	}
	if len(nguon) > 0 {
		sort.Strings(nguon)
		fmt.Fprintf(&sb, "\n     tìm ở: %s", strings.Join(nguon, ", "))
	} else {
		fmt.Fprintf(&sb, "\n     tìm ở: <kho>/plugins/<ten>/plugin.toml và <dự án>/.sagent/plugins/<ten>/plugin.toml (chưa có thư mục nào)")
	}
	for _, l := range loi {
		fmt.Fprintf(&sb, "\n     manifest hỏng: %s", l)
	}
	return Manifest{}, fmt.Errorf("%s", sb.String())
}
