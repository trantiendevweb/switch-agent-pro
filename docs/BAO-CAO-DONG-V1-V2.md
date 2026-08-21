# Báo cáo lượt: đóng ô V2 (danh tính Antigravity) và ô V1 (suy cờ từ hồ sơ)

- **Ngày**: 21/08/2026
- **Nhánh**: `sagent/phu-1`
- **Ô nợ đụng tới**: `docs/SO-NO-DO-LUONG.md` — **V2** (vàng), **V1** (vàng).
- **Kết quả một câu**: cả hai ô đóng theo chiều **có bằng chứng chạy thật** —
  V2 tìm được nguồn danh tính **phân biệt được** với cái bẫy, V1 đổi bốn ô
  `ChuaDo` thành **kết luận** `Khong` kèm chỗ đã tra. Số ô `ChuaDo` trong bảng
  năng lực: **7 → 2**.

---

# PHẦN A — V2: Antigravity ĐỌC ĐƯỢC danh tính

## A.1 Cái bẫy, nói lại cho rõ trước khi nói đã tránh thế nào

Banner của Antigravity CLI 1.1.17 in `ttseotop1@gmail.com`.
`~/.gemini/google_accounts.json` cũng ghi `active = ttseotop1@gmail.com`.

**Hai giá trị trùng nhau KHÔNG chứng minh gì.** Chúng trùng vì cùng một người
từng đăng nhập Gemini CLI trên máy này. Đọc `google_accounts.json` có thể cho ra
một email **CÓ THẬT nhưng SAI NGƯỜI** — đúng định dạng nên không ai nghi. Ô V2
chốt: *hiện nhầm email còn tệ hơn không hiện gì*.

Nên việc phải làm không phải "tìm một chỗ có email", mà **tìm một nguồn phân
biệt được nó với `google_accounts.json`**.

## A.2 Trước hết: xác nhận lời cũ trong sổ vẫn đúng

Sổ nợ viết *"không file nào trong ~/.gemini bị cập nhật email;
google_accounts.json vẫn mang dấu thời gian của lần đăng nhập Gemini CLI CŨ"*.

Đo lại 21/08:

```
~/.gemini/google_accounts.json      18/08 10:09     <- ĐỨNG IM
```

Ba lượt `agy` ngày 21/08 (**21:16**, **21:17**, **21:18**) không đụng vào nó.
Liệt kê **mọi** file trong `~/.gemini` đổi trong ngày 21/08 — 24 file, toàn bộ
nằm dưới `antigravity-cli/` (builtin, cache, crashes, jetski_state, updater,
log) — **không có `google_accounts.json`**.

Lời cũ **đúng**. Nó chỉ dừng lại sớm một bước.

## A.3 Nguồn phân biệt được: nhật ký của CHÍNH `agy`

`agy` tự ghi nhật ký vào `<hồ sơ>/.gemini/antigravity-cli/log/cli-*.log`. Trong
đó có ba dạng dòng (nguyên văn, chỉ cắt bớt cho vừa trang):

```
server_oauth.go:190] applyAuthResult: email=ttseotop1@gmail.com, authMethod=consumer, quotaProject=
server_oauth.go:195] OAuth: authenticated successfully as ttseotop1@gmail.com
browser.go:161]      consumerOAuth: authenticated successfully as ttseotop1@gmail.com
```

Dòng thứ ba chỉ xuất hiện ở **lượt đăng nhập bằng trình duyệt**; hai dòng đầu
xuất hiện ở **mọi** lượt chạy, ngay lúc token được áp dụng.

## A.4 HAI PHÉP ĐO chứng minh nguồn này KHÁC `google_accounts.json`

Đây là phần trả lời đúng câu hỏi được giao. Cả hai phép đo đều là **thư mục
KHÔNG HỀ CÓ `google_accounts.json`, mà nhật ký VẪN in ra email**.

### Phép đo 1 — HOME giả (dựng riêng cho lượt này)

Đổi cả `USERPROFILE` + `APPDATA` + `LOCALAPPDATA` + `HOME` sang một thư mục
trống rồi chạy `agy --log-file <tạm> -p "say OK"` (cùng lối đã dùng để đo
`TachDuocTaiKhoan`).

```
thư mục mới dựng:  .gemini/{antigravity-cli, config}     <- KHÔNG có google_accounts.json
nhật ký ghi ra:    applyAuthResult: email=ttseotop1@gmail.com
```

### Phép đo 2 — hồ sơ THẬT đang dùng (không phải môi trường dựng)

`~/.ai-accounts/antigravity/may` là hồ sơ thật của `sagent` (`EnvVar` của
Antigravity là `USERPROFILE`, nên thư mục hồ sơ đóng vai HOME).

```
cây .gemini của hồ sơ:  antigravity-cli/ + config/      <- KHÔNG có google_accounts.json
log/cli-20260821_144204.log (21/08 14:42):
    browser.go:161] consumerOAuth: authenticated successfully as ttseotop1@gmail.com
```

Dòng đó do **chính lượt đăng nhập bằng trình duyệt hôm nay** ghi ra.

**Kết luận**: email trong nhật ký đến từ **token trong Windows Credential
Manager** (khoá `gemini:antigravity`), không phải từ file của Gemini CLI. Đó
chính là thứ mà "hai giá trị trùng nhau" một mình không chứng minh nổi.

## A.5 Đã loại hai nguồn khác, ghi lại để khỏi tra lại

**`CredRead().UserName`** — nghe rất hứa hẹn: nó là metadata, không phải blob bí
mật, nên đọc được mà không mở token. Đo bằng `cmdkey /list:gemini:antigravity`:

```
User: antigravity        <- HẰNG SỐ, không phải email
```

Không dùng được.

**Quét cả cây hồ sơ tìm chuỗi email**: nhật ký là file **DUY NHẤT** mang nó. Các
file cấu hình khác chỉ có `trustedWorkspaces`, kiểu xác thực,
`remoteControlHostname`.

## A.6 MỘT RÀO ĐÃ ĐO RỒI BỎ — phần đáng giữ nhất của lượt này

Nhật ký có một điểm yếu thật, phải nói ra: **đăng nhập lại bằng tài khoản KHÁC ở
một HOME khác** sẽ để nhật ký cũ nằm lại trong hồ sơ này.

Rào định dựng: dùng `LastWritten` của mục Credential Manager làm mốc — *"nhật ký
cũ hơn mốc thì trả rỗng"*. Đo thật trước khi viết:

```
Credential `gemini:antigravity`.LastWritten   21/08 21:16:59
nhật ký HỢP LỆ của hồ sơ `may`               21/08 20:15:39     <- CŨ HƠN
```

Rào đó sẽ trả **RỖNG cho đúng cái hồ sơ đang chạy đúng**. Nguyên nhân:
`LastWritten` đổi **cả khi làm mới token** lẫn **khi đăng nhập lại**, và không mở
blob ra thì không phân biệt được hai việc — mà mở blob chính là thứ `TokenExpiry`
đã cố ý từ chối (ô Đ4). **Rào sai hướng thì thà không có.** Đã bỏ, và ghi lý do
vào mã để người sau khỏi thử lại.

Cửa sổ rủi ro còn lại hẹp vì `TachDuocTaiKhoan() = false` — **mỗi máy MỘT tài
khoản Antigravity**, nên "nhầm" ở đây là nhầm giữa danh tính hiện tại và một
danh tính cũ, không phải giữa hai tài khoản đang chạy song song. Nó có thật, và
bảng năng lực nói thẳng ra thay vì giấu.

## A.7 Đã đổi gì trong mã

| Chỗ | Trước | Sau |
|---|---|---|
| `antigravity.go` `Identity` | `return ""` | đọc nhật ký, có cổng `coCredential` |
| `antigravity.go` `NangLuc` | `Chua(NLDanhTinh, …)` | `Duoc(NLDanhTinh, …)` |
| `danhtinh_antigravity.go` | *(mới)* | bộ đọc + toàn bộ phép đo viết thành bình luận |
| `danhtinh_antigravity_test.go` | *(mới)* | 8 bài kiểm |

**Cổng `coCredential` đứng trước là cố ý**: đăng xuất thì mục
`gemini:antigravity` biến mất khỏi Credential Manager, nhưng nhật ký cũ vẫn nằm
nguyên trên đĩa. Không có cổng này thì một hồ sơ **đã đăng xuất** vẫn khoe email
— đúng kiểu hỏng mà ô V2 dựng ra để chặn. Cổng chỉ **hỏi mục có tồn tại không**,
không mở bí mật ra (đúng kỷ luật của `cred_windows.go`).

## A.8 Bài kiểm

Bài đáng kể nhất là `TestAntigravityKhongLayEmailTuGoogleAccounts`: dựng một hồ
sơ có **CẢ HAI** nguồn nói **KHÁC NHAU** rồi bắt hàm lấy nguồn do `agy` ghi. Nếu
người sau "đơn giản hoá" bằng cách đọc `google_accounts.json`, bài này đỏ ngay
với đúng câu chữ của ô nợ.

Còn lại: lấy nhật ký mới nhất; lấy dòng xác thực **cuối cùng** trong file (phiên
dài có thể xác thực lại giữa chừng); bỏ qua symlink `cli.log`; trả rỗng khi
không có bằng chứng; và `TestAntigravityKhaiDanhTinhKhopVoiHam` bắt **lời khai
khớp hàm cả hai chiều** — cùng lối với `TestCursorKhaiDanhTinhKhopVoiHam` của ô
V3, vì phép dò `NLDanhTinh` trong `nangluc.go:168` là **một chiều** và không tự
bắt được chiều "khai làm được mà luôn trả rỗng".

Bài kiểm gọi thẳng `danhTinhTuNhatKyAgy` chứ không gọi `Identity`: `Identity`
còn cổng Credential Manager, thứ phụ thuộc **máy** chứ không phụ thuộc **hồ sơ**.
Bài kiểm phải đo phần hồ sơ.

## A.9 Đã chạy trên hồ sơ thật

Bài kiểm tạm (**không** commit — nó phụ thuộc máy):

```
C:\Users\Administrator\.ai-accounts\antigravity\may  -> Identity="ttseotop1@gmail.com"
C:\Users\Administrator                               -> Identity="ttseotop1@gmail.com"
```

Không còn rỗng.

---

# PHẦN B — V1: bốn ô "suy cờ từ chính thư mục hồ sơ"

Ô này không rỗng — Grok khai `Duoc(NLCoTuHoSo)` vì nó **thật sự** phải đọc
`defaultModel` trong `.grok/user-settings.json`. Nên bốn ô kia phải được tra
thật, không được kết luận bằng cách nhìn Grok rồi đoán ngược.

**Câu hỏi phân biệt** (rút từ chính ca Grok): không phải *"hồ sơ có thiết lập
không"* mà là ***"CLI có TỰ ĐỌC thiết lập đó không"***. Grok **có**
`defaultModel` mà CLI **phớt lờ** — nên phải ép `-m`, nếu không mọi bước hỏng
lặng lẽ với 503.

## B.1 Codex — ô đáng đo nhất, vì nó trông y hệt ca Grok

`$CODEX_HOME/config.toml` trên máy này **CÓ** thiết lập model:

```toml
model = "gpt-5.6-sol"
model_reasoning_effort = "high"
```

Nên phải chạy thật để biết CLI có đọc không. Dựng một `CODEX_HOME` tạm chỉ có
`auth.json` và một `config.toml` khai model **bịa**, rồi `codex exec --json`:

```
item.completed  "Model metadata for `khong-co-model-nay-dau` not found. Defaulting to fallback…"
turn.failed     400 "The 'khong-co-model-nay-dau' model is not supported when using Codex with a ChatGPT account."
```

Tên model bịa đi **thẳng ra tới máy chủ**. Tức Codex **ĐỌC** `config.toml` của
chính `CODEX_HOME` được truyền vào → **không cần ai chuyển nó thành cờ**.

→ `Khong(NLCoTuHoSo, …)` kèm nguyên văn phép đo. Đây là ô **ngược hẳn** Grok, và
đó mới là lý do đáng ghi.

## B.2 Claude — tra hồ sơ thật, không có thiết lập nào để chuyển

Hồ sơ `~/.ai-accounts/claude/tns` (`CLAUDE_CONFIG_DIR`):

- `.claude.json`: **48 khoá** tầng ngoài. Bốn khoá mang chữ "model"
  (`additionalModelCostsCache`, `additionalModelOptionsCache`, `modelAccessCache`,
  `orgModelDefaultCache`) đều là **CACHE của máy chủ**, không khoá nào là **lựa
  chọn** của người dùng.
- `settings.json`: chỉ có `env.DISABLE_AUTOUPDATER` và `tui`.

Không có gì phải chuyển thành cờ, và model đã đi tường minh qua `--model`.

## B.3 Cursor — hồ sơ RỖNG, chứ không phải chưa nhìn

`%APPDATA%\Cursor` có **ĐÚNG MỘT file** — `auth.json`, 893 byte — với **ĐÚNG HAI
khoá** `accessToken` / `refreshToken`. Không có thiết lập nào tồn tại. (Cùng
phép tra đã đóng ô V3 về danh tính.)

## B.4 Antigravity — có ba file cấu hình, không cái nào thành cờ

```
.gemini/settings.json                    security.auth.selectedType = oauth-personal
.gemini/antigravity-cli/settings.json    trustedWorkspaces = [...]
.gemini/config/config.json               userSettings.remoteControlHostname
```

Không cái nào có cờ tương ứng trong `agy --help`, và `agy` đọc thẳng chúng từ
HOME được truyền vào.

**Một cái bẫy đã tránh**: `trustedWorkspaces` **trông** giống việc của
`--add-dir`. Nhưng thư mục làm việc thật đã do `ArgsThuMuc` truyền vào **từng
lượt** — đọc lại danh sách cũ trong file là **ép agent vào workspace của lượt
trước**. Đúng kiểu "đoán sai theo chiều thừa" mà ô V1 cảnh báo.

**Ghi thêm, không thuộc ô này**: `agy --help` **CÓ** `--model`. Ô `chon-model`
của Antigravity vẫn `ChuaDo` vì lượt này không chạy thật để xác nhận cờ có hiệu
lực — thấy trong `--help` là đúng cái mức bằng chứng mà sổ nợ xếp vào loại nguy
hiểm. Nhưng nay đã biết chỗ để bắt đầu.

---

# PHẦN C — Kết quả

## Số ô `ChuaDo` trong bảng năng lực

Đếm trực tiếp trên mã, `git show HEAD` so với cây làm việc:

| Provider | Trước | Sau |
|---|---|---|
| claude | 1 | **0** |
| codex | 2 | **1** |
| cursor | 1 | **0** |
| antigravity | 3 | **1** |
| grok | 0 | 0 |
| **Tổng** | **7** | **2** |

Hai ô còn lại là `chon-model` của **codex** và **antigravity** — cùng một câu
hỏi, cùng một cách đóng.

*(Sổ nợ ghi "còn 9" ở phần mở đầu. Con số đó **đã lạc hậu** từ trước lượt này:
đếm thật trên `HEAD` là **7**. Đã sửa trong sổ.)*

## Kiểm

```
go build ./...    OK
go vet ./...      OK
go test ./...     OK (toàn bộ gói)
```

`KiemNangLuc` vẫn xanh cho cả năm adapter: `Duoc(NLDanhTinh)` không đụng phép dò
một chiều vì `Identity(thuMucKhongTonTai)` vẫn trả `""`, và `Khong(NLCoTuHoSo)`
khớp với `ArgsHoSo` trả `nil`.

## Còn nợ, nói thẳng

1. **Nhật ký cũ sau khi đăng nhập lại ở HOME khác** (A.6). Hẹp, nhưng có thật.
2. **`chon-model` của codex và antigravity** — hai ô `ChuaDo` cuối cùng.
3. **Chiều "khai làm được mà luôn trả rỗng"** vẫn chưa có phép dò chung: nay có
   bài kiểm riêng cho `NLDanhTinh` của **cursor** và **antigravity**, nhưng
   `NLCoTuHoSo`, `NLKetQuaCoCauTruc`, `NLHanToken` vẫn phải trông vào từng bài
   kiểm rời.
