# v1 — bộ công cụ PowerShell (`tk`)

Đây là **bản gốc** của dự án, hồi còn tên `ccswitch`: một script PowerShell đổi qua lại
nhiều tài khoản Claude Code trên Windows. Giữ lại để tham chiếu và để ai đang dùng `tk`
vẫn có chỗ tra.

> **Bản đang phát triển là Go — dùng `sagent`.** Xem [README ở gốc repo](../../README.md).

## Vì sao còn ở đây

- Bản Go đã thay được các việc chính (`them` · `ds` · `run` · `dong-bo` · `xoa` · `verify`),
  nhưng **chưa có bảng chọn tương tác đánh số** như `tk`.
- Migration guide chính thức từ `tk` v1 nằm ở Pha 7 của
  [`docs/MASTER-PLAN.md`](../../docs/MASTER-PLAN.md).
- `kiem-tra.ps1` vẫn là bộ đo hữu ích trên Windows; bản Go tương đương là `sagent verify`.

## File

```
cai-dat.ps1     cài tk vào ~/.claude/skills/ + tạo lệnh ~/bin/tk.cmd
go-bo.ps1       gỡ
kiem-tra.ps1    bộ đo (thoát ≠ 0 nếu có mục đỏ)
src/tk.ps1      công cụ chính
src/cfg.py      đọc/ghi .claude.json (whitelist ở đây) — Go đã thay bằng internal/jsonutil
src/soi-cfg.py  soi xem file .claude.json nào đang thật sự được dùng
HDSD.md         hướng dẫn dùng tk
SKILL.md        mô tả skill cho Claude Code
```

## Ba bẫy đã trả giá khi viết v1 (bản Go giữ nguyên bài học)

1. **`Remove-Item -Recurse` xuyên junction xoá luôn dữ liệu thật** → phải gỡ từng link
   trước, kiểm không còn link nào, rồi mới xoá.
2. **PowerShell 5.1 chết khi JSON có khoá trùng hoa/thường** → v1 phải nhờ Python;
   Go xử lý được nên bản mới bỏ hẳn phụ thuộc này.
3. **Đặt tên hàm PowerShell trùng tên lệnh cần gọi = đệ quy im lặng.**

Nếu bạn đã cài `tk` trước đây, nó nằm ở `%USERPROFILE%\.claude\skills\ccswitch\` và
**không bị ảnh hưởng** bởi việc di chuyển thư mục này trong repo.
