# Báo cáo: đóng ô V3 và viết lại ô C6 (`docs/SO-NO-DO-LUONG.md`)

Ngày 21/08/2026. Máy Windows Server 2022, `cursor-agent 2026.08.11-e8db854`.

---

## 1. Việc được giao và việc thật sự làm

| Ô | Giao là gì | Hoá ra là gì |
|---|---|---|
| **V3** | Lỗi thật: khai `Duoc(NLDanhTinh)` mà `Identity()` luôn trả rỗng | Lượt trước **đã** hạ khai xuống `Chua`, nên hai vế đã khớp. Phần còn nợ là **phép đo**. Đo xong thì lật lại kết luận cũ: **CÓ** trường danh tính → sửa **hàm**, nâng lại khai `Duoc`. |
| **C6** | Chỉ là sổ lạc hậu, không phải lỗi mã | **Là lỗi mã thật.** `cursor-agent` **CÓ** cờ đổi thư mục (`--workspace`); `ArgsThuMuc` trả `nil` và khai `Khong(NLThuMuc)` đều **SAI**. |

Hai ô đều đóng, nhưng **cả hai đều đóng ngược chiều dự kiến**. Điểm chung:
kết luận cũ đứng trên một phép đo **đặt câu hỏi quá hẹp**, chứ không phải trên
một khoảng trống.

---

## 2. V3 — đo xong, và phép đo lật lại kết luận

### Đã làm gì

Mở `%APPDATA%\Cursor\auth.json` **thật** trên hồ sơ đang đăng nhập và **giải
payload JWT** (lượt trước mới ngó tầng ngoài rồi dừng lại).

- `%APPDATA%\Cursor` có **ĐÚNG MỘT file**: `auth.json`, 893 byte → chốt luôn
  phạm vi, không còn chỗ nào khác để tìm danh tính.
- Tầng ngoài: **đúng hai khoá** `accessToken`/`refreshToken`, không
  `email`/`userEmail`/`user_email`. **Lượt trước nói đúng phần này.**
- Payload hai JWT giống hệt nhau, **đúng tám claim**:

```
iss         https://authentication.cursor.sh
aud         https://cursor.com
sub         google-oauth2|user_01<...>          <- trường danh tính DUY NHẤT
scope       openid profile email offline_access
type        session
randomness  <uuid cụt>
time        1787022539
exp         1792206539
```

### Chỗ kết luận cũ sai

Sổ cũ mô tả đúng (*"`sub` là mã đục, không phải địa chỉ thư"*) rồi kết luận
**CHƯA ĐỌC ĐƯỢC**. Bước kết luận sai: câu năng lực này hỏi *"đọc được **danh
tính** để hiển thị"*, không phải *"đọc được **email**"*. `sub` là subject của
OIDC — bền, mỗi tài khoản một giá trị. Nó **là** danh tính.

**Tiền lệ có sẵn trong repo**: `internal/provider/grok.go` trả
`baseURL · defaultModel` thay cho email và vẫn khai `Duoc(NLDanhTinh)`. Bảng
năng lực **đã** chấp nhận danh tính phi-email từ trước.

**Không mâu thuẫn V2** (*"hiện nhầm email còn tệ hơn không hiện gì"*): chỗ
Antigravity nguy vì cho ra email **có thật nhưng sai người**, nhìn đúng định dạng
nên không ai nghi. `google-oauth2|user_01…` không thể bị nhầm là địa chỉ thư, và
là danh tính của **chính** hồ sơ đang đọc. Trông xấu, không nói dối.

### BẪY đã tránh

Claim `scope` **có chữ "email"** (`openid profile email offline_access`) — đó là
**phạm vi OAuth đã xin**, KHÔNG phải claim email. Đọc lướt thấy chữ "email" rồi
khai "đọc được email" chính là **đúng kiểu sai** đã sinh ra ô V3. Ghim bằng
`TestCursorKhongNhamScopeLaEmail`.

### Mã đã sửa

- `internal/provider/cursor.go:137` — `Identity()` nay đọc thêm claim JWT, thứ tự
  ưu tiên: `email` tầng ngoài → claim `email` → claim `sub`.
- `internal/provider/cursor.go:170` — thêm `danhTinhTuJWT()`.
- `internal/provider/cursor.go:276` — tách `payloadJWT()` ra khỏi `hanTuJWT()`,
  vì nay có **hai** người đọc payload (hạn token và danh tính).
- `internal/provider/cursor.go:416` — khai lại `Duoc(NLDanhTinh, …)`.

### Kiểm trên hồ sơ THẬT

Chạy `Identity()` lên `%APPDATA%` thật:

```
Identity tren ho so THAT = "google-oauth2|user_01M09DEFFZVVCRKNV1H882WG3A"
```

Không còn rỗng. Bài kiểm tạm đó **không commit** (phụ thuộc máy).

### Đã bịt lỗ "phép dò một chiều" — phần đáng giữ nhất

Phép dò `NLDanhTinh` (`internal/provider/nangluc.go:168`) là **một chiều**
(`haiChieu: false`): chỉ bắt được chiều *"khai chưa đo mà lại trả giá trị thật"*.
Chiều *"khai làm được mà luôn trả rỗng"* **không ai canh** → `KiemNangLuc` xanh
suốt trong khi lời khai sai. Cùng lỗ đó đang che cho `NLCoTuHoSo`,
`NLKetQuaCoCauTruc`, `NLHanToken`.

Sổ đã chốt cách bịt: **không** đổi phép dò (một chiều là **đúng** — cần hồ sơ
thật mới dò được), mà bằng **một bài kiểm chạy trên hồ sơ thật**. Nay có:
`TestCursorKhaiDanhTinhKhopVoiHam` dựng `auth.json` **đúng hình dạng file thật**
rồi bắt hai vế khớp **cả hai chiều**.

Không nhét token thật vào repo: chữ ký JWT không được kiểm nên token giả là đủ,
và một token thật trong mã nguồn là một rò rỉ.

---

## 3. C6 — không phải sổ lạc hậu, là lỗi mã

Đây là phần **ngoài dự kiến** của lượt này, và là phần nặng hơn.

### Câu hỏi đặt hẹp nên kết luận sai

Phép đo cũ hỏi *"có `--cwd` hay `-C` không?"* → **không**, rồi nhảy sang
*"provider này không có cờ đổi thư mục"*. Vế đầu **đúng**, bước nhảy **sai**.
Đọc hết `--help` của **đúng bản 2026.08.11-e8db854**:

```
--workspace <path-or-name>  Workspace directory or saved workspace name to use
                            (defaults to current working directory)
--add-dir <path>            Add an additional workspace root directory
```

### Phép đo, CÓ ĐỐI CHỨNG

Tạo `%TEMP%\do-thumuc-cursor\VAN-TAY-9F3A2B.txt`, chạy từ cwd
`C:\Users\Administrator` (**không** phải thư mục đó):

```
có --workspace <dir>  -> agent liệt kê "VAN-TAY-9F3A2B.txt"        <- thấy
không có cờ           -> "no", và tự khai cwd C:\Users\Administrator
```

Có đối chứng nên kết luận được **chính cái cờ** gây ra khác biệt. Cờ được nhận và
**có hiệu lực**, không bị nuốt im lặng.

### Vì sao `--workspace` chứ không `--add-dir`

Bản này có **cả hai**. `--add-dir` là *"Add an **additional** workspace root"* —
thêm một gốc nữa, trong khi hợp đồng `ArgsThuMuc`
(`internal/provider/adapter.go:59-66`) đòi khai **tường minh** thư mục làm việc.
Claude/Antigravity phải dùng `--add-dir` vì CLI của chúng không có cờ đặt thẳng;
Cursor có.

### Vì sao lý lẽ "không cần" cũng sai

Sổ cũ chống chế *"fleet đã chạy tiến trình con với `workDir` rồi"*. Lý lẽ đó bỏ
qua **đúng cái** hợp đồng `ArgsThuMuc` sinh ra để chặn: fleet chạy agent trong
**git worktree**, mà ở worktree `.git` là **FILE con trỏ** chứ không phải thư
mục, nên provider dò workspace **hụt dù cwd đã đúng**. Đã đo trên Antigravity:
cùng lệnh cùng cờ, ở repo thật **3/3**, ở worktree chỉ **1/3**; thêm cờ thì
**4/4**. **cwd đúng KHÔNG bảo đảm workspace đúng** — và Cursor là provider chạy
trong worktree nhiều nhất.

### Mã đã sửa

- `internal/provider/cursor.go:363` — `ArgsThuMuc` trả `[]string{"--workspace", dir}`.
- `internal/provider/cursor.go:395` — khai lại `Duoc(NLThuMuc, …)`.

Phép dò `NLThuMuc` (`nangluc.go:145-148`) là **HAI CHIỀU**, nên `KiemNangLuc`
bắt buộc hai vế đi cùng nhau — sửa một vế mà quên vế kia là đỏ ngay.

### Vì sao `KhongLamDuoc` sai nguy hiểm ngang `LamDuoc` sai

`KhongLamDuoc` tự xưng là **một kết luận** (*"đã đo, provider không có thứ đó"*)
nên **không ai đi đo lại** — khác `ChuaDo`, thứ tự nó mời người sau đến đo. Một
`ChuaDo` sai thì tốn công; một `KhongLamDuoc` sai thì **đóng vĩnh viễn** một năng
lực có thật. Cụ thể: `sagent nang-luc` nói dối **theo chiều thiếu**, người vận
hành đi tìm đường vòng cho vấn đề đã có cờ giải sẵn.

---

## 4. Bài học chung của cả hai ô

Sổ đã có: *"một dòng **CHƯA ĐO** không tự hết hạn"*. Lượt này thêm vế mạnh hơn:

> **Một dòng ĐÃ ĐO cũng không tự hết hạn — và nó nguy hơn, vì nó không tự nhận là
> khoảng trống.**

Cả V3 lẫn C6 đều hỏng ở **bước từ quan sát sang kết luận**, không ở quan sát:

- V3: *"không có trường `email`"* (đúng) → *"không đọc được danh tính"* (sai).
- C6: *"không có `--cwd`/`-C`"* (đúng) → *"không có cờ đổi thư mục"* (sai).

Cả hai lần, cách bắt là **mở file/CLI thật ra đọc HẾT**, không tin lời khai và
không tin tiêu đề commit.

Và `KiemNangLuc` **không** bắt được C6 dù phép dò hai chiều: nó chỉ so hai vế
**trong repo** với nhau: hai vế cùng sai một cách nhất quán thì vẫn xanh. Chỉ có
đọc `--help` thật mới bắt được.

---

## 5. Phạm vi và ràng buộc

- Trong `docs/SO-NO-DO-LUONG.md` **chỉ** sửa hai mục **V3** và **C6**. Xác nhận
  bằng `git diff -U0`: đúng **hai** hunk (`@@ -481` = C6, `@@ -539` = V3).
- **Không** đụng mục **C1** (đang có phiên khác sửa phần Codex), cũng không đụng
  `internal/provider/codex.go`.
- **Cố ý KHÔNG sửa** dòng tham chiếu chéo ở `SO-NO-DO-LUONG.md:151` — nó nằm
  trong mục **Đ3** (không thuộc phạm vi) và nhắc tới C1/C3/C6 cùng lúc, sửa vào
  là chạm phần phiên khác đang giữ. **Dòng đó nay lạc hậu**: nó vẫn mô tả C6 là
  đóng theo chiều `Khong(NLThuMuc)`, và số dòng `cursor.go:158`/`:190` cũng đã
  cũ. **Để lại cho lượt sau.**

## 6. Kiểm chứng

```
gofmt   — sạch (kiểm trên nội dung LF; repo để core.autocrlf=true nên
          `gofmt -l` liệt kê mọi file, kể cả file không đụng tới)
go vet ./...    — sạch
go build ./...  — XANH
go test ./...   — XANH, toàn bộ 22 gói
```

Bài kiểm mới: `internal/provider/danhtinh_cursor_test.go` — 11 test
(6 cho V3 / danh tính, 3 cho C6 / thư mục, kèm bảng ngõ hỏng).
