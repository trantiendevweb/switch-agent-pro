# Di trú từ `tk` v1 (PowerShell) sang `sagent` v2

> Mọi câu trong tài liệu này đều đo được trên máy thật. Chỗ nào chưa đo thì ghi rõ
> là chưa đo. Xem [`DO-LUONG.md`](DO-LUONG.md) cho nguyên tắc.

## Câu trả lời ngắn: bạn không phải làm gì

`sagent` đọc **thẳng** kho hồ sơ v1 ở `~/.claude-accounts/`. Không copy, không đổi
tên, không chạy lệnh di trú nào.

Đo trên máy có sẵn hồ sơ v1, khi `~/.ai-accounts/claude` **chưa hề tồn tại**:

```
$ sagent ds

  Tài khoản AI trên máy này

     1  claude  phu          trantien.developer.frontend@gmail.com sẵn sàng
```

Hồ sơ `phu` nằm ở `~/.claude-accounts/phu`, và `sagent` dùng nó nguyên trạng — kể cả
email hiển thị cũng đọc từ đúng file cũ. Test `TestHoSoV1VanDungDuoc` chốt hành vi này,
nên nó không lặng lẽ mất đi ở bản sau.

## Hai kho, và cái nào được ưu tiên

| Kho | Đường dẫn | Ai tạo |
|---|---|---|
| v1 | `~/.claude-accounts/<tên>` | `tk` (PowerShell), chỉ Claude |
| v2 | `~/.ai-accounts/<provider>/<tên>` | `sagent`, mọi provider |

`ResolveDir` tìm **chỗ v2 trước**, không có thì tìm v1. Nghĩa là hai hồ sơ trùng tên thì
v2 thắng — và đó từng là một cái bẫy:

> **Lỗi đã đo và đã vá (2026-08-17).** `sagent them claude:phu` khi đã có hồ sơ **v1** tên
> `phu` thì *không* bị từ chối: `Create` chỉ kiểm `~/.ai-accounts/claude/phu` (chưa có)
> nên nó tạo luôn. Sau đó `ResolveDir` trỏ sang v2, và token v1 **vẫn nằm trên đĩa nhưng
> không còn được dùng**. Người dùng bị hỏi đăng nhập lại, không hiểu vì sao token cũ
> "biến mất". Nay `Create` kiểm bằng `ResolveDir` nên nó từ chối và **chỉ ra hồ sơ cũ nằm
> ở đâu**.

Giờ gõ trùng tên sẽ nhận:

```
✗ đã có tài khoản claude:phu tại C:\Users\<bạn>\.claude-accounts\phu
```

## Cấu trúc một hồ sơ v1, đo trên máy thật

```
~/.claude-accounts/phu/
  .claude.json                      file THẬT  (danh tính + thói quen máy)
  .credentials.json                 file THẬT  (token)
  .claude.json.bak-20260817-131438  bản lưu do tk v1 sinh
  history.jsonl, file-history/      file thật
  backups  -> ~/.claude/backups     junction
  cache    -> ~/.claude/cache       junction
  chrome   -> ~/.claude/chrome      junction
  plugins  -> ~/.claude/plugins     junction
```

Nguyên lý giống hệt v2: **file riêng là file thật, phần dùng chung là junction trỏ về
`~/.claude`.** Đó là lý do v2 đọc được kho v1 mà không cần chuyển đổi gì.

## Nếu bạn vẫn muốn dời hẳn sang kho v2

**`sagent` cố ý KHÔNG có lệnh tự dời.** Lý do không phải lười:

- Kho v1 đầy junction trỏ vào `~/.claude`. Một lệnh dời viết ẩu sẽ đi **xuyên** junction
  và xoá dữ liệu Claude thật ở đầu bên kia. Lớp lỗi này **đã nổ một lần trên máy dev**
  ngày 2026-08-17 và xoá sạch `~/.claude` — xem [`DO-LUONG.md`](DO-LUONG.md).
- Kho v1 **đang chạy tốt**, có test bảo vệ. Đổi một thứ đang chạy được để lấy sự gọn gàng
  là đánh đổi tồi.

Muốn dời thì làm tay, và làm theo đúng thứ tự này:

```powershell
# 1. Dừng mọi phiên và dash trước — không sửa file đang có tiến trình mở.
sagent stop all

# 2. Chép RIÊNG hai file thật. KHÔNG chép cả thư mục: chép cả thư mục là chép
#    xuyên junction, kéo theo toàn bộ ~/.claude vào chỗ mới.
$src = "$env:USERPROFILE\.claude-accounts\phu"
$dst = "$env:USERPROFILE\.ai-accounts\claude\phu"
New-Item -ItemType Directory -Force $dst | Out-Null
Copy-Item "$src\.credentials.json" $dst
Copy-Item "$src\.claude.json"      $dst

# 3. Nối lại phần dùng chung bằng chính sagent (nó tạo junction, không chép).
sagent dong-bo claude:phu

# 4. Kiểm rồi mới xoá bản cũ.
sagent verify
sagent ds
Remove-Item -Recurse "$src"     # chỉ khi bước 4 sạch
```

Bước 2 là chỗ chết người: `Copy-Item -Recurse` trên thư mục đầy junction sẽ **đi xuyên**
và nhân bản cả `~/.claude`. Chép đúng hai file thật thôi.

## `ccswitch`

**Chưa đo.** Dự án chưa có bản `ccswitch` nào trên máy để mở ra xem cấu trúc, nên không
viết hướng dẫn di trú cho nó. Có bản thật thì đo rồi bổ sung — đoán mò một đường di trú
động tới token là cách nhanh nhất để mất token.

## Cái v1 có mà v2 chưa có

**Chưa đo đủ để liệt kê.** Bản v1 nằm ở [`legacy/v1-powershell/`](../legacy/v1-powershell)
để đối chiếu. Chỗ nào bạn thấy thiếu thì mở issue — đừng để nó thành "chắc là có".
