---
name: ccswitch
description: Dùng nhiều tài khoản Claude Code trên cùng một máy Windows và đổi qua lại không phải đăng nhập lại. Dùng khi cần tạo thêm tài khoản, xem đang chạy tài khoản nào, đồng bộ cấu hình dùng chung (trust dialog, MCP theo project, skill, plugin) sang mọi tài khoản, hoặc khi một tài khoản hết hạn mức và muốn sang tài khoản khác. Có script chạy được, không cần WSL và không phụ thuộc gói npm nào.
---

# ccswitch — đổi tài khoản Claude Code trên Windows

## Nó hoạt động thế nào

Claude Code đọc biến môi trường `CLAUDE_CONFIG_DIR` để biết lấy cấu hình và token ở thư mục nào. Công cụ `tk` tạo cho mỗi tài khoản một thư mục riêng trong `%USERPROFILE%\.claude-accounts\<tên>`, rồi trỏ biến đó vào trước khi chạy `claude`.

Hệ quả: mỗi tài khoản đăng nhập **một lần**, đổi qua lại không phải đăng nhập lại, không tài khoản nào thấy token của tài khoản khác.

## Lệnh

| Lệnh | Làm gì |
|---|---|
| `tk` | Bảng chọn có đánh số — gõ số là vào, `t` thêm, `d` đồng bộ, `x` xoá. Dấu `*` là tài khoản đang dùng. Chạy trong script (không có bàn phím) thì nó in trợ giúp chứ không treo |
| `tk <tên>` | Chạy Claude Code bằng tài khoản đó |
| `tk goc` | Chạy bằng tài khoản gốc |
| `tk them <tên>` | Tạo tài khoản mới rồi mở luồng đăng nhập (`-KhongMo` để chỉ tạo) |
| `tk ds` | Liệt kê |
| `tk dong-bo [-XemTruoc]` | Chép cấu hình dùng chung sang mọi tài khoản |
| `tk xoa <tên>` | Xoá tài khoản và token |

## Cái gì dùng chung, cái gì riêng

**Riêng từng tài khoản**: `.credentials.json` (token) và `.claude.json` (danh tính).

**Dùng chung** — nối bằng junction/symlink về `%USERPROFILE%\.claude`, sửa một chỗ mọi tài khoản thấy: `skills`, `plugins`, `projects`, `settings.json`, `sessions`, `cache`, `tasks`…

**Chép sang lúc tạo và mỗi lần `tk dong-bo`** — 19 khoá thuộc về *cái máy* và *thói quen làm việc*: trust dialog từng project, `allowedTools`, MCP theo project, onboarding, `skillUsage`, `pluginUsage`…

**Cố ý KHÔNG chép**: `oauthAccount`, `userID`, và nhóm khoá gắn với gói cước hoặc tổ chức (`modelAccessCache`, `passesEligibilityCache`, `penguinModeOrgEnabled`…). Dùng **danh sách trắng** chứ không phải danh sách đen — danh sách đen thì mai sau thêm một khoá gói cước mới là nó lặng lẽ lọt sang, rồi tài khoản B tưởng mình có quyền của A.

## Phải cấp lại một lần cho mỗi tài khoản

**Claude Code Remote** và **MCP connector claude.ai** (Miro, Gmail, Google Drive) gắn với tài khoản claude.ai chứ không gắn với máy. Đăng xuất rồi đăng nhập tay cũng vậy — không phải thiếu sót của công cụ.

## Bốn điều đã đo, đừng suy luận lại

Chạy `kiem-tra.ps1` trong repo để đo lại; nó thoát mã khác 0 nếu có mục đỏ.

1. `claude.exe` có đọc `CLAUDE_CONFIG_DIR`.
2. Đặt `CLAUDE_CONFIG_DIR=X` thì Claude ghi `X\.claude.json`, và ở đó không thấy MCP của tài khoản khác — tách thật.
3. Token nằm ở `.credentials.json` trong thư mục cấu hình, không nằm trong Credential Manager.
4. Khi **không** đặt biến, Claude dùng `%USERPROFILE%\.claude.json`, **không** phải `%USERPROFILE%\.claude\.claude.json`. Máy có thể tồn tại cả hai và file trong `.claude` là file lạc chưa trust project nào — gieo nhầm nó là mất sạch trust dialog mà không có gì báo. Vì vậy `tk goc` **xoá** biến chứ không trỏ vào `.claude`.

## Ba bẫy đã trả giá

1. **`Remove-Item -Recurse` có thể xuyên qua junction xoá luôn dữ liệu thật.** `tk xoa` gỡ từng link trước, kiểm không còn link nào, rồi mới xoá phần còn lại.
2. **PowerShell 5.1 chết khi JSON có khoá trùng hoa/thường** → toàn bộ phần JSON giao cho `cfg.py` bằng Python.
3. **Hàm PowerShell trùng tên lệnh cần gọi = đệ quy im lặng**: hàm `Python` gọi `Get-Command python` nhận về chính nó, `.Source` rỗng, lỗi báo ra là `expression after '&' ... not valid`.

## File

```
%USERPROFILE%\.claude\skills\ccswitch\
├─ SKILL.md
├─ HDSD.md
└─ scripts\
   ├─ tk.ps1        công cụ chính
   ├─ cfg.py        phần đọc/ghi .claude.json (danh sách trắng ở đây)
   └─ soi-cfg.py    soi xem file .claude.json nào đang thật sự được dùng
```

Lệnh `tk` ở `%USERPROFILE%\bin\tk.cmd`. Mã nguồn: https://github.com/trantiendevweb/ccswitch
