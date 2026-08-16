# ccswitch

Đổi qua lại nhiều tài khoản Claude Code trên **Windows**, không phải đăng nhập lại mỗi lần.

Không cần WSL. Không cần Node. Không cài thêm gói npm nào.

```
  Tài khoản Claude Code trên máy này

     1  phu          ban@gmail.com                      sẵn sàng
   * 0  gốc          chu@gmail.com                      sẵn sàng

   Gõ số để mở  ·  t thêm  ·  d đồng bộ  ·  x xoá  ·  ? trợ giúp  ·  Enter thoát

   Chọn:
```

## Nó hoạt động thế nào

Không có phép thuật gì. Claude Code đọc biến môi trường `CLAUDE_CONFIG_DIR` để biết lấy cấu hình và token ở thư mục nào.

`ccswitch` tạo cho mỗi tài khoản một thư mục riêng trong `%USERPROFILE%\.claude-accounts\<tên>`, rồi trỏ biến đó vào trước khi chạy `claude`. Hết.

Hệ quả: mỗi tài khoản đăng nhập **một lần**, đổi qua lại không phải đăng nhập lại, và không tài khoản nào thấy token của tài khoản khác.

Phần còn lại của công cụ chỉ là làm cho việc đó đỡ phiền: nối những thứ dùng chung (skill, plugin, settings, lịch sử phiên) về một chỗ, và chép sang tài khoản mới đúng những khoá cấu hình thuộc về *cái máy* chứ không thuộc về *tài khoản*.

## Cài

Tải repo về rồi chạy trong PowerShell (**không** cần quyền quản trị):

```powershell
git clone https://github.com/trantiendevweb/ccswitch.git
cd ccswitch
.\cai-dat.ps1
```

Nó chép script vào `%USERPROFILE%\.claude\skills\ccswitch\`, tạo lệnh `tk` ở `%USERPROFILE%\bin\tk.cmd`, và nhắc bạn nếu thư mục đó chưa nằm trong `PATH`.

Yêu cầu: Windows, Claude Code đã cài, và Python 3 (dùng để đọc `.claude.json` — lý do ở phần *Ba bẫy* bên dưới).

## Dùng

Gõ `tk` là ra bảng chọn, gõ số là vào. Ai thích gõ lệnh thẳng thì:

| Lệnh | Làm gì |
|---|---|
| `tk` | Bảng chọn có đánh số. Dấu `*` là tài khoản đang dùng |
| `tk <tên>` | Chạy Claude Code bằng tài khoản đó |
| `tk goc` | Chạy bằng tài khoản gốc (thư mục `.claude` mặc định) |
| `tk them <tên>` | Tạo tài khoản mới rồi mở luồng đăng nhập |
| `tk ds` | Liệt kê |
| `tk dong-bo [-XemTruoc]` | Chép cấu hình dùng chung sang mọi tài khoản |
| `tk xoa <tên>` | Xoá tài khoản kèm token của nó |

Lần đầu:

```powershell
tk them phu1      # đăng nhập tài khoản thứ hai, xong gõ /exit
tk                # chọn số để đổi qua lại
```

> Trình duyệt đang đăng nhập sẵn tài khoản cũ thì luồng đăng nhập nối thẳng lại vào tài khoản cũ, và bạn có hai thư mục cùng một tài khoản mà nhìn bảng không biết. Đăng xuất claude.ai trước, hoặc mở link đăng nhập bằng cửa sổ ẩn danh.

Claude Code lấy **thư mục hiện tại** làm nơi làm việc, nên `cd` vào dự án rồi mới gõ `tk <tên>`.

## Cái gì dùng chung, cái gì riêng

**Riêng từng tài khoản** — có vậy mới gọi là tách:

- `.credentials.json` (token)
- `.claude.json` (danh tính)

**Dùng chung** — nối bằng junction/symlink về `%USERPROFILE%\.claude`, sửa một chỗ mọi tài khoản thấy: `skills`, `plugins`, `projects`, `settings.json`, `sessions`, `cache`, `tasks`…

**Chép sang lúc tạo và mỗi lần `tk dong-bo`** — 19 khoá trong `.claude.json` thuộc về *cái máy* và *thói quen làm việc*: trust dialog từng project, `allowedTools`, MCP theo project, đã qua onboarding, `skillUsage`, `pluginUsage`…

**Cố ý KHÔNG chép**: `oauthAccount`, `userID`, và cả nhóm khoá gắn với gói cước hoặc tổ chức (`modelAccessCache`, `passesEligibilityCache`, `penguinModeOrgEnabled`…).

Dùng **danh sách trắng** chứ không phải danh sách đen. Làm ngược lại là kiểu dễ rò: mai sau Claude Code thêm một khoá gói cước mới, danh sách đen sẽ lặng lẽ để nó lọt sang, rồi tài khoản B tưởng mình có quyền của A và báo lỗi khó hiểu lúc dùng. Danh sách nằm ở `src/cfg.py`, sửa được.

## Hai thứ phải cấp lại cho từng tài khoản

- **Claude Code Remote** (điều khiển từ điện thoại)
- **MCP connector claude.ai**: Miro, Gmail, Google Drive

Chúng gắn với tài khoản claude.ai chứ không gắn với máy. Đăng xuất rồi đăng nhập tay cũng vậy thôi — không phải thiếu sót của công cụ. Bù lại chỉ phải làm **một lần cho mỗi tài khoản**.

## Bốn điều đã đo, không phải suy luận

Chạy `.\kiem-tra.ps1` để đo lại trên máy bạn. Bộ kiểm thoát với mã khác 0 nếu có mục đỏ.

1. `claude.exe` **có** đọc `CLAUDE_CONFIG_DIR`.
2. Đặt `CLAUDE_CONFIG_DIR=X` thì Claude ghi `X\.claude.json`, và `claude mcp list` ở đó không thấy MCP của tài khoản khác — **tách thật**, không phải tách trên giấy.
3. Token nằm ở `.credentials.json` trong thư mục cấu hình, không nằm trong Windows Credential Manager. Vì vậy tách thư mục là tách được tài khoản.
4. Khi **không** đặt biến, Claude dùng `%USERPROFILE%\.claude.json`, **không** phải `%USERPROFILE%\.claude\.claude.json`. Trên máy viết công cụ này tồn tại cả hai, file trong `.claude` là file lạc có 1 project chưa trust cái nào — gieo nhầm nó là mất sạch trust dialog mà không có gì báo. Vì vậy `tk goc` **xoá** biến chứ không trỏ vào `.claude`.

## Ba bẫy đã trả giá khi viết

1. **`Remove-Item -Recurse` có thể xuyên qua junction xoá luôn dữ liệu thật.** Thư mục tài khoản toàn junction trỏ về `.claude` gốc. Nên `tk xoa` gỡ từng link trước, kiểm không còn link nào, rồi mới xoá phần còn lại. Bộ kiểm có phép đo riêng cho việc này: đặt một file mồi trong `.claude`, xoá tài khoản, rồi đếm lại.

2. **PowerShell 5.1 chết khi JSON có khoá trùng hoa/thường.** File `.claude.json` thật chứa cả `C:\Users\...\SEO Project` lẫn `c:\users\...\seo project`; `ConvertFrom-Json` ném lỗi ngay. Vì vậy toàn bộ phần JSON giao cho `src/cfg.py` bằng Python (khoá trùng thì lấy cái cuối). Đây là lý do công cụ cần Python.

3. **Đừng đặt tên hàm PowerShell trùng tên lệnh cần gọi.** Hàm tên `Python` gọi `Get-Command python` sẽ nhận về **chính nó** — PowerShell ưu tiên Function hơn Application — `.Source` rỗng, và lỗi báo ra là `expression after '&' ... not valid`, không hề nhắc gì tới đệ quy.

## Gỡ

```powershell
.\go-bo.ps1
```

Xoá lệnh `tk` và thư mục skill. Thư mục `%USERPROFILE%\.claude-accounts` chỉ bị xoá nếu bạn đồng ý — trong đó có token, xoá là phải đăng nhập lại. Thư mục `.claude` gốc **không** bị đụng tới.

## Một câu nói cho sòng phẳng

Công cụ này chạy cục bộ, không gửi gì đi đâu, không đụng tới token của bạn ngoài việc để chúng ở các thư mục khác nhau.

Nhưng dùng nhiều tài khoản để vượt hạn mức nhiều khả năng đi ngược điều khoản dịch vụ của nhà cung cấp, và cái mất nếu bị phát hiện là tài khoản. Có những lý do dùng hoàn toàn bình thường — tài khoản cá nhân và tài khoản công ty trên cùng một máy, hoặc một máy nhiều người dùng chung. Bạn tự cân, đây chỉ là ghi lại cho rõ.

## Giấy phép

MIT. Xem `LICENSE`.
