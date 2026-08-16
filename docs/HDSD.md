# Hướng dẫn dùng ccswitch

## Chỉ cần nhớ một chữ: `tk`

Mở PowerShell, gõ `tk` rồi Enter:

```
  Tài khoản Claude Code trên máy này

     1  phu1         ban@gmail.com                      sẵn sàng
   * 0  gốc          chu@gmail.com                      sẵn sàng

   Gõ số để mở  ·  t thêm  ·  d đồng bộ  ·  x xoá  ·  ? trợ giúp  ·  Enter thoát
```

Dấu `*` là tài khoản đang dùng. Gõ số rồi Enter là vào thẳng Claude Code.

## Thêm tài khoản

Bấm `t`, gõ tên (chữ thường không dấu, ví dụ `phu1`), Enter. Claude Code mở ra:

1. Nếu nó chưa hỏi đăng nhập, gõ `/login`
2. **Trước khi mở link đăng nhập**: đăng xuất claude.ai trên trình duyệt, hoặc dán link vào cửa sổ ẩn danh. Không làm bước này thì trình duyệt nối thẳng lại vào tài khoản cũ, và bạn có hai thư mục cùng một tài khoản mà nhìn bảng không biết
3. Đăng nhập bằng tài khoản Claude **mới**
4. Gõ `/status` để xác nhận đúng email, rồi `/exit`

Về PowerShell gõ `tk` — dòng vừa tạo phải chuyển thành `sẵn sàng` kèm email mới.

## Hằng ngày

```powershell
tk            # chọn số
tk phu1       # hoặc gõ thẳng tên
tk goc        # về tài khoản gốc
```

Claude Code lấy **thư mục hiện tại** làm nơi làm việc, nên `cd` vào dự án trước rồi mới gõ `tk <tên>`.

## Khi nào cần `tk dong-bo`

Mỗi tài khoản có file danh tính riêng, nên những thứ bạn bấm ở tài khoản này **không tự sang** tài khoản kia: trust dialog của project mới, bật MCP cho một project, `allowedTools`.

Làm mấy việc đó xong thì gõ `tk` rồi bấm `d`, hoặc:

```powershell
tk dong-bo -XemTruoc     # xem sẽ đổi gì, không ghi
tk dong-bo               # ghi thật, có sao lưu kèm đuôi .bak-<ngày giờ>
```

Skill, plugin, `settings.json`, lịch sử phiên thì **không cần** — chúng nối thẳng về `%USERPROFILE%\.claude`, sửa một chỗ mọi tài khoản thấy ngay.

## Hai thứ phải cấp lại cho từng tài khoản

- **Claude Code Remote** (điều khiển từ điện thoại)
- **MCP connector claude.ai**: Miro, Gmail, Google Drive

Chúng gắn với tài khoản claude.ai, không gắn với máy. Chỉ phải làm một lần cho mỗi tài khoản.

## Hỏng thì xem gì

**Gõ `tk` báo không nhận lệnh** — thư mục `%USERPROFILE%\bin` chưa nằm trong `PATH`. Chạy lại `cai-dat.ps1` và chọn thêm vào PATH, rồi mở lại cửa sổ PowerShell.

**Báo thiếu Python** — công cụ cần Python 3 để đọc `.claude.json`. Lý do: PowerShell 5.1 ném lỗi khi JSON có hai khoá chỉ khác nhau hoa/thường, mà file thật thường có đúng lỗi đó.

**Đăng nhập xong vẫn hiện `(chưa đăng nhập)`** — nhiều khả năng trình duyệt nối lại tài khoản cũ. Vào `tk <tên>`, gõ `/status` xem email nào; sai thì `/logout` rồi `/login` lại bằng cửa sổ ẩn danh.

**Muốn biết máy đang đọc file cấu hình nào** — chạy:

```powershell
python "$env:USERPROFILE\.claude\skills\ccswitch\scripts\soi-cfg.py"
```

## Gỡ

```powershell
.\go-bo.ps1
```

Xoá lệnh `tk` và thư mục skill. Thư mục tài khoản chỉ xoá khi bạn đồng ý — trong đó có token. Thư mục `.claude` gốc không bị đụng tới.
