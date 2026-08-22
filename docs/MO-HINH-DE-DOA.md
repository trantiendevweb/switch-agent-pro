# Mô hình đe doạ — Switch-Agent-Pro

> Chốt **21/08/2026, 23:45**. Mọi số trong tài liệu này **đo trên chính cái máy đang
> chạy dashboard**, không phải trên máy dev sạch. Lệnh đo nằm ở mục **J** — chạy lại
> được, và **phải chạy lại** trước khi tin bất cứ dòng nào ở đây.
>
> Tài liệu này là mục cuối cùng còn treo của **Pha 0** trong
> [`MASTER-PLAN.md`](MASTER-PLAN.md). Nó **không đo lại từ đầu**: bằng chứng gốc nằm ở
> [`DO-LUONG.md`](DO-LUONG.md) và [`VAN-HANH-VPS.md`](VAN-HANH-VPS.md); việc của tài
> liệu này là **gom lại thành một sườn** — tài sản → kẻ tấn công đứng ở đâu → làm được
> gì → hiện có gì chặn.

## Cách đọc — bốn nhãn, không có nhãn thứ năm

| Nhãn | Nghĩa |
|---|---|
| ✅ **ĐÃ CÓ** | Có mã chặn, dẫn được `file:dòng`, và có bài kiểm ghim để nó không bị gỡ lén. |
| 🟡 **CÓ MỘT PHẦN** | Chặn được đường chính, còn **đường vòng đã biết** — ghi rõ đường vòng đó. |
| 🔴 **KHÔNG CÓ GÌ** | Không có dòng mã nào chặn. Đây **không** phải "nên cân nhắc" — là **đang hở**. |
| ⬜ **CHƯA ĐO ĐƯỢC** | Chưa có phép đo. **Kèm lý do**, và không được viết như đã biết. |

Không có nhãn "có vẻ ổn". Theo đúng nguyên tắc chẩn đoán ở
[`VAN-HANH-VPS.md`](VAN-HANH-VPS.md) mục F: **không đo được thì ghi thẳng "không đo
được" kèm lý do, đừng đoán.**

---

## A. Bối cảnh — vì sao mô hình này khác mô hình của một công cụ chạy trên laptop

Đây không phải giả định. Đây là số đo, và nó đổi hạng của gần như mọi mối đe doạ bên dưới.

| Sự thật | Số đo | Nguồn |
|---|---|---|
| Máy nằm **thẳng trên internet**, không sau NAT | SAN của chứng chỉ tự ký chứa IP công cộng `103.97.134.90` | `DO-LUONG.md:713` |
| Dashboard **đang phơi ra mạng ngay lúc này** | PID `16368` = `sagent.exe dash --host 0.0.0.0 --port 8788`, `LocalAddress = ::` (dual-stack) | đo 21/08 23:45 |
| Tường lửa **mở đúng cổng đó vào từ ngoài** | rule inbound `sagent-dash-8788`, `Action = Allow`, `Enabled = True` | đo 21/08 23:45 |
| Máy **đang bị brute-force RDP thật** | `4625`: **~3.900/giờ** (sáng 21/08) → **1.867/giờ** (14:20) → **157/giờ** (23:45) | `VAN-HANH-VPS.md` mục E + đo lại 23:45 |
| Việc chặn IP **có tác dụng, và vẫn đang chạy** | task `TNS Chan Bruteforce` mỗi 5 phút; rule `Block` đang giữ **61 IP** (14:20 là **53**) | đo 21/08 23:45 |
| `lsass` **đã chết 3 lần** làm máy tự reboot | event `1015`: `lsass.exe failed with status code 1` | `VAN-HANH-VPS.md` mục A3 |
| `RunAsPPL` **đã đặt nhưng CHƯA có hiệu lực** | `HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\RunAsPPL = 1`, **chưa reboot** | `VAN-HANH-VPS.md` mục E |

**Hai điều rút ra, và cả hai đều bất lợi:**

1. **Cuộc tấn công đang diễn ra chưa nhắm vào `sagent`** — nó nhắm vào RDP. Nhưng nó
   chứng minh máy này **có người đang gõ cửa liên tục**, nên mọi mối đe doạ hạng
   "cần đứng ngoài internet" bên dưới đều là **có thật**, không phải giả định lý thuyết.
2. **`RunAsPPL` chưa hiệu lực nghĩa là `lsass` vẫn chạy KHÔNG được bảo vệ.** Ai lấy
   được quyền quản trị là đọc được credential trong bộ nhớ. Và như `VAN-HANH-VPS.md`
   đã ghi: `RunAsPPL` chống **trộm credential**, nó **không** chữa việc `lsass` chết —
   hai chuyện khác nhau, đừng trông nó chữa reboot.

> ⚠ **Điểm phải nói thẳng ngay đầu tài liệu:** brute-force RDP vẫn chỉ là **nghi phạm**
> của ba lần reboot, **chưa phải nguyên nhân**, vì **chưa có dump của `lsass`**. Đừng
> đọc mục A này thành "đã kết luận".

---

## B. Tài sản — cái gì đáng để người ta tấn công

Tất cả nằm dưới **một gốc duy nhất**: `~/.ai-accounts` (`paths.AccountsRoot()`).
Đó vừa là điểm mạnh (siết một chỗ là siết cả cây) vừa là điểm yếu (thủng một chỗ là
thủng cả cây). Số đo 21/08 23:45:

| Tài sản | Nằm ở đâu | Đếm được | Lấy được thì làm gì |
|---|---|---|---|
| **Token thuê bao Claude** | `<hồ sơ>/.credentials.json` + **N bản trong `.clones/`** | **4** file `.credentials.json` trong `.clones`, trên **1.291** thư mục con | Dùng hạn mức gói thuê bao của chủ máy; đọc mọi thứ agent đọc được |
| **Token/khoá các provider khác** | Codex: `auth.json` · Cursor: `Cursor/auth.json` (qua `APPDATA`) · Antigravity: qua `USERPROFILE` · Grok: `user-settings.json` (`apiKey`) | — | như trên, theo từng nhà cung cấp |
| **API key** (grok/deepseek) | `~/.ai-accounts/api-keys/<id>.key` | **2** file `.key` | **Tiêu tiền thật theo token**, không có trần |
| **Mật khẩu dashboard (đã băm)** | `~/.ai-accounts/dash-auth.json` | **có** | Dò offline; băm được là điều khiển được cả hạm đội |
| **Khoá riêng TLS** | `~/.ai-accounts/dash-tls/key.pem` | có | Giả mạo dashboard, đọc mật khẩu người dùng gõ vào |
| **Token bot Telegram** | `~/.ai-accounts/tele.json` | **KHÔNG có trên máy này** (đã đo) | Gửi tin giả danh; không áp dụng lúc này |
| **Nhật ký phiên agent** | `~/.ai-accounts/.nhat-ky/` | — | **Toàn văn prompt và câu trả lời** của mọi phiên đã chạy |
| **Cây làm việc của agent** | `~/.ai-accounts/.worktrees/` | **90** worktree | Mã nguồn dự án; và là **bàn đạp** — xem mục G |
| **Sổ `state.db`** | `~/.ai-accounts/state.db` | — | Lịch sử phiên, route, chi phí. **Cố ý không có cột secret** (`internal/store/store.go:296`) |

**Một quyết định thiết kế đáng ghi vào đây vì nó cắt bớt cả một lớp đe doạ:**
`state.db` và mọi file cấu hình **không bao giờ chứa secret**. Route chỉ ghi `key_id` —
tên file, không phải khoá (`internal/aiapi/aiapi.go:36`, `internal/store/registry.go:43`).
Nghĩa là **lộ `state.db` không lộ khoá nào**, và một bản backup `state.db` gửi đi đâu
cũng không mang theo bí mật.

---

## C. Kẻ tấn công đứng ở đâu — sáu vị trí, và chỉ sáu

Mọi mối đe doạ bên dưới đều tham chiếu một trong sáu vị trí này. Nói "kẻ tấn công" mà
không nói **đứng ở đâu** thì không kiểm chứng được.

| Mã | Vị trí | Có thật trên máy này chưa |
|---|---|---|
| **K1** | **Ngoài internet, không có tài khoản gì** — chỉ gõ được vào cổng đang mở | ✅ đang xảy ra: 157 lượt `4625`/giờ |
| **K2** | **Ngoài internet, ĐÃ có mật khẩu dashboard** (đoán ra, hoặc rò qua đường khác) | ⬜ chưa đo được — xem mục I |
| **K3** | **Cùng đường mạng, đứng giữa** (MITM) trên đường tới `:8788` | ⬜ chưa đo được |
| **K4** | **Tiến trình khác trên cùng máy, KHÁC người dùng** | ✅ có ý nghĩa: máy nhiều dịch vụ, cổng 80/443/3000/8080/8090 đều mở |
| **K5** | **Chính agent đang chạy** — có tool, có thể có `--tu-duyet-quyen` | ✅ **đang xảy ra ngay lúc bạn đọc dòng này** |
| **K6** | **Đã có quyền quản trị / đọc được `lsass`** | ⚠ `RunAsPPL` chưa hiệu lực nên rào này **đang mở** |

**K5 là vị trí bị coi nhẹ nhất và nguy hiểm nhất.** Nó không phải "kẻ tấn công" theo
nghĩa thông thường — nó là **công cụ của chính chủ máy**, chạy với **đúng quyền của chủ
máy**, và nó nhận **chỉ thị bằng tiếng Việt từ một file văn bản**. Xem mục G.

---

## D. Mặt 1 — Credential của tài khoản thuê bao

Đây là mặt có **nhiều bản vá đo được nhất**, và cũng là mặt có **cái hở còn lại nặng nhất**.

### D1. Chép token ra N bản clone → một bản refresh là N-1 bản chết

**Làm được gì:** không phải tấn công — là **tự bắn vào chân**, nhưng hậu quả giống hệt
một cuộc tấn công: **mất tài khoản, phải đăng nhập lại**.

**Đứng ở đâu:** không cần đứng đâu cả. Chỉ cần chạy `sagent fleet --copies N`.

**Vì sao nó là chuyện an ninh chứ không phải chuyện tiện nghi:** nhà cung cấp **XOAY
VÒNG refresh token** — đo 20/08/2026, ghi ở `internal/profile/clone.go:34-46`. Mỗi lần
refresh sinh token mới và **giết token cũ ngay**. Chép token ra N chỗ **không cần N tiến
trình đua nhau mới hỏng**: **một** bản refresh là **N-1** bản còn lại chết, **kể cả bản
gốc**.

Cuộc đua thật đã đo (`clone.go:48-59`, ô Đ5 của [`SO-NO-DO-LUONG.md`](SO-NO-DO-LUONG.md)),
hai clone bật cách nhau **16ms**:

```
A: exit 0, refresh 86fb2200 -> d22e079d, lượt chạy xong bình thường
B: exit 1 sau 186ms, "OAuth session expired and could not be refreshed",
   và TỰ GHI ĐÈ .credentials.json của mình thành "" / "" / expiresAt 0
```

**Chi tiết quyết định:** bản **THUA** ghi file **SAU** bản thắng (nó hỏng nhanh, bản
thắng còn phải đợi hết lượt gọi API). Nên **"clone có mtime mới nhất" chính là clone
RỖNG** — một phép đồng bộ ngây thơ theo mtime sẽ lan cái rỗng ra mọi bản.

**Hiện có gì chặn:** 🟡 **CÓ MỘT PHẦN**

| Lớp | Ở đâu | Chặn được gì |
|---|---|---|
| Đồng bộ ngược **TRƯỚC khi chép đè** | `internal/profile/clone.go:88` gọi `SyncBackTokens` (`:164`) | Không để `Clone` hồi sinh một token đã chết đè lên bản đang sống — đúng chuỗi đã làm hỏng lượt #47 |
| Kiểm "đã đăng nhập chưa" **SAU** khi đồng bộ | `clone.go:98` | Cứu được ca hồ sơ gốc bị CLI xoá sạch `refreshToken` trong khi clone vẫn còn token sống |
| Không lan file rỗng của bản thua | `internal/profile/duarefresh_test.go:50,102` | Bản thắng không bị file rỗng của bản thua "lấn sang" |

**Đường vòng vẫn còn hở — nói thẳng:** đồng bộ ngược chỉ chạy **lúc `Clone` được gọi**.
Giữa hai lần gọi, N clone vẫn cầm N bản của **cùng một** refresh token. Không có khoá,
không có bản ghi "ai đang giữ token hiện hành". Nếu hai phiên tình cờ refresh gần nhau
thì **cuộc đua ở trên xảy ra lại**, và bản vá hiện tại chỉ **dọn hậu quả ở lượt sau**
chứ **không ngăn cuộc đua**.

> 🪤 Bẫy liên quan, đã ghi ở `VAN-HANH-VPS.md` D2: `sagent fleet --copies 1` chạy **hai
> lần** không cho hai worktree — cả hai rơi vào **bản clone số 1** và giẫm lên nhau.
> Muốn hai việc song song thì dùng **hai tài khoản khác nhau**.

### D2. Kho clone hở quyền → hở N bản token một lúc

**Làm được gì:** đọc thẳng `.credentials.json` của mọi bản clone.

**Đứng ở đâu:** **K4** (tiến trình khác trên cùng máy, khác người dùng).

**Vì sao nó từng là lỗ thật:** trên Windows, `os.WriteFile(path, data, 0o600)` **không
làm gì cả**. Đo ở `DO-LUONG.md:497-514`:

```
secret-0600.json   ->  BUILTIN\Users:(I)(F)     ← Users TOÀN QUYỀN
public-0644.json   ->  BUILTIN\Users:(I)(F)     ← y hệt
```

Bit quyền Unix là **trang trí**; quyền thật đến từ ACL **kế thừa** của thư mục cha. Trên
máy dev nó tình cờ an toàn vì `C:\Users\<tên>` vốn kín — nhưng đó là **MAY**, không phải
**BẢO ĐẢM**.

**Hiện có gì chặn:** ✅ **ĐÃ CÓ**

- `internal/acl/acl_windows.go:32` — `Restrict` dựng DACL tường minh cho **chủ sở hữu +
  SYSTEM + Administrators** và **CẮT KẾ THỪA** (`PROTECTED_DACL_SECURITY_INFORMATION`).
  Thư mục được cấp cờ `SUB_CONTAINERS_AND_OBJECTS_INHERIT` nên **file tạo sau vẫn kín** —
  đây là lý do `.nhat-ky/` (tạo bằng `MkdirAll 0o755`, không tự gọi `Restrict`) vẫn được
  che, vì nó nằm dưới gốc đã siết.
- Bảy chỗ gọi `acl.Restrict`, trong đó **chỗ quan trọng nhất là `clone.go:114`** —
  chính chỗ token bị nhân ra N bản.

**Bài học đã trả giá, ghi lại vì nó là kiểu hỏng hay lặp:** bản vá ACL **đầu tiên** nối
`Restrict` vào ba chỗ tạo thư mục và **bỏ sót đúng `clone.go`** (`DO-LUONG.md:1129-1137`).
Ô kiểm lúc đó **chỉ soi thư mục gốc** nên nó vẫn **xanh**. Nay `khoHoSoCheck`
(`internal/api/api.go:563`) **quét CẢ CÂY** bằng `filepath.Walk` — vì "một phép kiểm chỉ
nhìn một điểm thì không phát hiện được *quên một chỗ*, mà *quên một chỗ* mới là kiểu hỏng
hay xảy ra".

**Lưới an toàn:** `internal/profile/clone_acl_test.go:19`
(`TestThuMucCloneCungPhaiSietQuyen`), `internal/acl/acl_test.go:33,65`.

### D3. Tên hồ sơ đi thẳng vào đường dẫn → xoá mất dữ liệu thật

**Làm được gì:** khiến `sagent` **xoá thư mục ngoài kho**, kể cả `~/.claude` thật.

**Đứng ở đâu:** **K2** (form thêm hồ sơ trên dashboard cũng nhập được tên), hoặc **K5**.

**Đây không phải giả thuyết — nó đã NỔ một lần** và **xoá mất `~/.claude`** ngày
2026-08-17 (`internal/profile/safedelete.go:44`). Ghép thẳng tên vào đường dẫn cho ra:

```
"phu"           -> ~/.ai-accounts/claude/phu     (đúng)
"../../.claude" -> ~/.claude                     (DỮ LIỆU CLAUDE THẬT)
""              -> ~/.ai-accounts/claude         (mọi tài khoản claude)
".."            -> ~/.ai-accounts                (mọi tài khoản, mọi provider)
```

**Hiện có gì chặn:** ✅ **ĐÃ CÓ — bốn lớp, cố ý chồng nhau**

| Lớp | Ở đâu | Chặn |
|---|---|---|
| Whitelist ký tự | `internal/profile/name.go:39` `ValidName` | Chỉ chữ/số/`-`/`_`/`.`; chặn `.`/`..`, tên bắt đầu bằng `.` hoặc `-`, kết thúc bằng `.`, và **tên thiết bị Windows** (`NUL`, `COM1`…) |
| Phải nằm trong kho | `name.go:83` `insideStore` + `:96` `under` | So sánh **không phân biệt hoa thường** trên Windows; `"."` (chính thư mục kho) cũng bị từ chối |
| Không đi xuyên link | `safedelete.go:64` `link.IsLink` → chỉ `Unlink`, **không** `RemoveAll` | `os.Lstat` **không** thấy junction; phải kiểm cờ reparse point |
| Sổ phải nhận sở hữu | `safedelete.go:70-79` | Trả lời câu mà kiểm-trên-đĩa không trả lời được: **"ai dựng ra nó"**. Sổ đọc hỏng thì **KHÔNG xoá** |

Thứ tự bốn lớp là cố ý và **không được đảo**: kiểm "ngoài kho" đặt **trước** câu hỏi cho
sổ, để **một dòng sổ bịa ra `~/.claude` cũng không mở được cửa**.

**Lỗ cùng lớp đã bịt ở tầng gốc:** `CleanClones` (`clone.go:284`) kiểm chính `root` **TRƯỚC**
khi `ReadDir` — nếu `root` là junction trỏ ra ngoài thì `ReadDir` **đi xuyên**, và mỗi
thư mục con **thật** bên kia sẽ bị xoá, trong khi đường dẫn vẫn nằm gọn trong kho nên
`insideStore` **chẳng thấy gì bất thường**.

**Lưới an toàn:** `name_test.go:32`, `safedelete_test.go:57,116`, `junction_test.go:39`,
`clone_acl_test.go:70`.

### D4. Đọc credential từ bộ nhớ `lsass`

**Làm được gì:** lấy credential Windows, rồi từ đó lấy mọi thứ trong mục B.

**Đứng ở đâu:** **K6**.

**Hiện có gì chặn:** 🟡 **CÓ MỘT PHẦN — và phần đang thiếu là phần có tác dụng**

`RunAsPPL = 1` **đã đặt** ngày 21/08 nhưng **CHƯA reboot**, nên `lsass` **vẫn đang chạy
không được bảo vệ** (`VAN-HANH-VPS.md` mục E). Máy **không có Secure Boot/UEFI**
(`Confirm-SecureBootUEFI` báo `0xC0000002`) nên không có khoá UEFI và việc bật này **gỡ
lại được** — tức là nó cũng **không phải rào không thể tháo**.

**`sagent` không có phần nào ở đây.** Ghi vào tài liệu này vì nó là **đường đi vòng qua
mọi thứ ở mục D2**: siết ACL bao nhiêu cũng vô nghĩa trước một kẻ đã có quyền quản trị.

---

## E. Mặt 2 — API key

Mặt này **sạch nhất trong bốn mặt**, và lý do là một quyết định kiến trúc chứ không phải
một loạt bản vá.

### E1. Đọc khoá từ file cấu hình hoặc từ sổ

**Làm được gì:** tiêu tiền thật của chủ máy theo token, không trần.

**Đứng ở đâu:** **K5** (agent đọc được repo), hoặc bất cứ ai lấy được `state.db` /
`.sagent/project.toml` — mà `project.toml` thì **nằm trong repo và được commit**.

**Hiện có gì chặn:** ✅ **ĐÃ CÓ — luật, không phải bản vá**

> **FILE CẤU HÌNH KHÔNG BAO GIỜ CHỨA SECRET.** Route chỉ ghi `key_id`; key thật nằm ở
> `~/.ai-accounts/api-keys/<id>.key`, trong kho đã siết ACL.
> — `internal/aiapi/aiapi.go:9-11`

- `Route` **không có trường khoá** (`aiapi.go:31-37`).
- Sổ SQLite **không có cột secret** — `routes_so` ghi `key_id` là **tên file**
  (`internal/store/store.go:296`, `internal/store/registry.go:43,207`).
- Khoá đọc bằng `docKey` (`aiapi.go:126`), gọi từ đúng **ba** chỗ: `aiapi.go:176` (gọi
  thường), `stream.go:35` (gọi stream), `suckhoe.go:58` (kiểm route).

**Hệ quả thực dụng, đáng nói to:** lộ `project.toml`, lộ `state.db`, lộ một bản backup —
**không cái nào lộ khoá**.

### E2. `key_id` bịa ra để đọc file ngoài kho

**Làm được gì:** biến hàm đọc-khoá thành hàm đọc-file-bất-kỳ. Đúng **cùng lớp lỗi đã nổ
một lần với tên hồ sơ** (D3) — và lần này đầu vào cũng đến từ **file cấu hình mà người
khác sửa được**.

**Đứng ở đâu:** **K5**, hoặc ai gửi cho bạn một repo có sẵn `.sagent/project.toml`.

**Hiện có gì chặn:** ✅ **ĐÃ CÓ** — `aiapi.go:130-132` chặn `id` chứa `/ \ :` và chặn
`.` / `..`. Ghim bằng `internal/aiapi/aiapi_test.go:37`
(`TestKeyIDKhongDuocThoatThuMuc`).

### E3. Khoá rò ra qua thông báo lỗi hoặc qua dashboard

**Làm được gì:** đọc khoá mà không cần chạm vào file — lỗi API thường được chép nguyên
văn vào log, vào báo cáo, vào ảnh chụp màn hình.

**Đứng ở đâu:** **K2**, hoặc bất cứ ai đọc được log.

**Hiện có gì chặn:** ✅ **ĐÃ CÓ**

- Lỗi giữ **nguyên văn của nhà cung cấp kèm request id**, nhưng **không kèm khoá** —
  ghim bằng `internal/aiapi/aiapi_test.go:100` (`TestKeyDiDungChoVaKhongRoRaLoi`).
- Dashboard **không trả cả `key_id`** ra `/api/ai` (`internal/dash/server.go:839`) — với
  lý do ghi thẳng trong mã: *"nó là tên file bí mật trong kho, không việc gì phải nói ra"*.
- Cùng luật đó áp cho token Telegram: **không có hàm nào trả token ra ngoài**
  (`internal/tele/tele.go:133`), vì *"token đi tới dashboard là token đi vào lịch sử
  trình duyệt và ảnh chụp màn hình"*.

### E4. Ai đăng nhập được dashboard thì tiêu được tiền

**Làm được gì:** gọi `POST /api/ai` bao nhiêu lượt tuỳ thích, tiêu hạn mức API thật.

**Đứng ở đâu:** **K2**.

**Hiện có gì chặn:** 🔴 **KHÔNG CÓ GÌ**

`/api/ai` (`server.go:829`) chỉ đi qua `guard` — tức là **chỉ cần một phiên đăng nhập
hợp lệ**. Không có trần số lượt, không có trần token, không có hạn mức ngày, không có
xác nhận lại cho lượt gọi đắt. `api_calls` (`internal/store/store.go:267`) **ghi lại**
mọi lượt (cả thành lẫn bại) — nhưng ghi lại là **kể lại sau**, không phải **chặn trước**.

Ghi thẳng: đây là **cái hở còn lại của mặt API key**, và nó chỉ nguy hiểm đúng bằng mức
mà mật khẩu dashboard nguy hiểm — xem mục F.

---

## F. Mặt 3 — Dashboard

**Đây là mặt nguy hiểm nhất, vì nó là mặt duy nhất kẻ tấn công chạm được từ internet mà
không cần gì cả.** Và trên máy này nó **đang mở thật**, không phải giả định.

### Hiện trạng đo được, 21/08 23:45

```
PID 16368  "C:\Users\Administrator\bin\sagent.exe" dash --host 0.0.0.0 --port 8788
LocalAddress = ::   (dual-stack, nghe mọi giao diện)
firewall inbound "sagent-dash-8788"  Action=Allow  Enabled=True

curl -sk https://127.0.0.1:8788/docs/DO-LUONG.md  ->  200, 139.541 byte   (KHONG can dang nhap)
curl -sk https://127.0.0.1:8788/api/state         ->  401                  (dung)
curl -s  http://127.0.0.1:8788/docs/              ->  400                  (TLS dang bat)
```

> 🪤 **Cái 400 ở dòng cuối KHÔNG phải lỗi server** — nó có nghĩa bạn gõ `http://` vào một
> cổng HTTPS. Bẫy này đã ăn thời gian thật, xem `VAN-HANH-VPS.md` A1.

### F1. Vào thẳng dashboard mà không cần mật khẩu

**Làm được gì:** toàn quyền — xem mục F4.

**Đứng ở đâu:** **K1**.

**Hiện có gì chặn:** ✅ **ĐÃ CÓ — và chặn ở chỗ không đi vòng được**

| Lớp | Ở đâu |
|---|---|
| Chưa đặt mật khẩu thì **từ chối chạy**, không phải cảnh báo | `server.go:315` |
| Phơi ra mạng mà không TLS thì **từ chối chạy** | `server.go:324` — chặn trong `Server.Run` **chứ không chỉ ở CLI**, vì đây là chỗ không đi vòng được |
| Mọi endpoint API đều bọc `guard` | `server.go:66-97` — 26 đường, không sót đường nào |
| Không đăng nhập → `401` cho API, `303` về form cho trình duyệt | `server.go:437-446` |
| Cửa vào **duy nhất** là form đăng nhập; **không còn token trên URL** | `server.go:54-56` — *"một secret nằm trong địa chỉ sẽ rơi vào log proxy, lịch sử trình duyệt và ảnh chụp màn hình"* |

### F2. Dò mật khẩu qua cổng đang mở ra internet

**Làm được gì:** đoạt phiên đăng nhập → F4.

**Đứng ở đâu:** **K1**. Và trên máy này, kẻ ở vị trí K1 **đã có mặt và đang gõ** — 157
lượt `4625`/giờ chứng minh điều đó, dù họ đang gõ vào RDP chứ chưa gõ vào `:8788`.

**Hiện có gì chặn:** 🟡 **CÓ MỘT PHẦN**

Có:
- **PBKDF2-HMAC-SHA256, 210.000 vòng** (`internal/dash/auth.go:62`) — **chậm có chủ đích**.
- **So sánh theo thời gian hằng** (`auth.go:108` `Check` → `crypto/subtle`).
- **Làm chậm dần khi sai**: sai ≥5 lần thì khoá 2 giây, tăng dần tới **tối đa 30 giây**
  (`server.go:463-476`).
- Bộ đếm **chỉ đếm ở `/login`, và chỉ đăng nhập THÀNH CÔNG mới xoá** (`server.go:428`,
  `:447`). Đây là bản vá của **hai lỗi thật đã đo**, và cả hai đáng nhắc lại vì chúng là
  ví dụ mẫu của "lá chắn quay ra đánh đúng người nó bảo vệ":

| Lỗi cũ | Hậu quả đo được | Nguồn |
|---|---|---|
| `guard` gọi `noteFail()` cho mọi request vô danh | **8 dòng `curl`** là khoá luôn người đang đăng nhập hợp lệ (429) | `DO-LUONG.md:1006` |
| `guard` gọi `noteOK()` cho mọi request hợp lệ | dashboard tự poll 5s/lần → **chỉ cần một tab đang mở là chống-dò-mật-khẩu vô hiệu** | `DO-LUONG.md:1007` |

Không có, và đây là chỗ hở:

- 🔴 **Không ghi lại lần đăng nhập sai nào — ở đâu cả.** `noteFail()` (`server.go:477`)
  chỉ tăng một biến đếm trong bộ nhớ. Không `bus.Warnf`, không ghi vào `state.db`, không
  ghi ra file. **Hệ quả cụ thể:** nếu ngay lúc này có người đang dò mật khẩu dashboard
  từ internet, **không có cách nào biết** — trong khi cuộc dò RDP song song thì đếm được
  tới từng lượt nhờ event `4625`. Mặt bị tấn công thì im lặng, mặt không bị tấn công thì
  có số.
- 🔴 **Bộ đếm là MỘT bộ đếm chung, không theo IP.** Kẻ tấn công thứ hai và người dùng
  hợp lệ dùng chung con số đó.
- 🟡 **Trần 30 giây là trần thấp.** Sai bao nhiêu lần cũng không bị khoá hẳn; nhịp sàn
  là ~2 lần thử/phút, mãi mãi. Với PBKDF2 210k vòng thì mật khẩu mạnh vẫn an toàn, nhưng
  **mật khẩu yếu thì chỉ được bảo vệ bởi con số 30 giây đó** — mà `SetPassword`
  (`auth.go:65-70`) chỉ đòi **tối thiểu 6 ký tự**, không đòi gì thêm.
- 🟡 **Bộ đếm nằm trong bộ nhớ, mất khi tiến trình khởi động lại.** Trên máy này có
  watchdog nhịp **~60 giây** dựng lại dash mỗi lần nó chết (`VAN-HANH-VPS.md` mục B) —
  tức là mỗi lần dash chết, **bộ đếm sai về 0 và mọi phiên đăng nhập bị huỷ**
  (`session.go:17-19`, cố ý không lưu ra đĩa).

### F3. Nghe lén / đứng giữa trên đường tới `:8788`

**Làm được gì:** đọc mật khẩu lúc người dùng gõ, hoặc lấy cookie phiên.

**Đứng ở đâu:** **K3**.

**Hiện có gì chặn:** 🟡 **CÓ MỘT PHẦN — và phần thiếu là phần người dùng phải tự làm**

Có:
- **HTTPS bật mặc định khi phơi ra mạng**, phải **gõ thêm chữ** (`--http-tran`) mới hạ
  cấp (`cmd/sagent/main.go:598-599`). Đã xác nhận đang bật: `http://` trả `400`.
- Chứng chỉ **phủ mọi IP của máy**, **dùng lại chứ không sinh mới mỗi lần chạy** — sinh
  mới thì vân tay đổi liên tục, người dùng quen tay bấm "vẫn tiếp tục", và **cả cơ chế
  đối chiếu thành vô dụng** (`DO-LUONG.md:733-737`).
- Khoá riêng ở `~/.ai-accounts/dash-tls/`, **siết ACL TRƯỚC khi ghi khoá vào**
  (`internal/dash/tls.go:120` rồi `:161`) — không phải sau.
- Cookie: `HttpOnly` + `SameSite=Lax`, và `Secure` **bám theo TLS thật** chứ không bật
  vô điều kiện (`session.go:66-80`) — bật cứng thì trên `http://127.0.0.1` trình duyệt
  **vứt cookie đi và không ai đăng nhập được**, "an toàn hơn" kiểu đó chỉ khiến người ta
  tắt bảo mật cho xong.
- ID phiên **32 byte ngẫu nhiên từ `crypto/rand`** (`session.go:27-30`) — không đoán được.

Thiếu:
- 🟡 **Chứng chỉ TỰ KÝ.** Công cụ in vân tay SHA-256 ra terminal và **nói thẳng trong
  chính dòng in**: *"Không đối chiếu thì TLS chỉ chống nghe lén, không chống kẻ đứng
  giữa."* (`server.go:377`). Nhưng **việc đối chiếu là việc của người dùng**, và
  **không có cách nào đo được người dùng có làm hay không**. Không đối chiếu thì K3 vẫn
  ăn được toàn bộ.
- 🟡 **Phiên sống 12 giờ** (`session.go:13`), **không buộc vào IP, không buộc vào
  User-Agent**. Cookie rò ra là dùng được suốt 12 giờ đó, và **không có đường nào huỷ
  một phiên cụ thể** ngoài `/logout` của chính phiên đó — hoặc khởi động lại server
  (huỷ sạch mọi phiên).

### F4. Đã đăng nhập rồi thì làm được gì

**Đây là câu quan trọng nhất của cả tài liệu**, vì nó quyết định mật khẩu dashboard đáng
giá bao nhiêu.

**Đứng ở đâu:** **K2**.

**Làm được gì — đo bằng cách đếm endpoint, không đoán:**

| Endpoint | Ở đâu | Làm được gì |
|---|---|---|
| `POST /api/fleet` | `server.go:1245` | **Bật agent, có cả cờ `tuDuyetQuyen`** — xem mục G |
| `POST /api/flow/run` | `server.go:87` | Chạy flow, tức chạy **bước `shell`** |
| `POST /api/flow/save` / `delete` | `server.go:92-93` | **Ghi thẳng vào `flows.toml`** — tức là **soạn ra lệnh sẽ chạy** |
| `POST /api/flow/decide` | `server.go:89` | **Bấm "Duyệt"** thay người — cửa duyệt tay không còn là cửa |
| `POST /api/stop` | `server.go:69` | Dừng mọi phiên |
| `POST /api/ai` | `server.go:72` | Tiêu tiền API thật (E4) |
| `GET /api/nhat-ky` | `server.go:80` | **Đọc toàn văn prompt + câu trả lời của mọi phiên đã chạy** |
| `GET /api/state`, `/api/so/*`, `/api/db` | `server.go:67,75-77` | Kiểm kê máy: hồ sơ, route, đường dẫn `state.db`, phiên bản schema |

**Kết luận thẳng:** **mật khẩu dashboard = thực thi mã tuỳ ý dưới quyền `Administrator`
trên một máy nằm thẳng trên internet.** Chính công cụ đã nói đúng chừng đó lúc khởi động
(`server.go:355`): *"Ai đăng nhập được đều BẬT/DỪNG được agent của bạn và tiêu hạn mức."*
Câu đó **đúng nhưng còn nhẹ** — thiếu vế "và chạy được lệnh tuỳ ý".

**Hiện có gì chặn ở tầng sau mật khẩu:** 🔴 **KHÔNG CÓ GÌ.** Không có vai trò, không có
chế độ chỉ-đọc, không có xác nhận hai bước cho hành động phá huỷ, không có nhật ký "ai
đã bấm gì". Mật khẩu là hàng rào **duy nhất**, và **là hàng rào cuối cùng**.

**Điều ĐÃ được giữ đúng, đáng ghi nhận:** dashboard **không mở đường riêng vào store** —
mọi thứ đi qua `internal/api`, nên **dashboard và CLI luôn ngang quyền**
(`server.go:2-4`). Nhờ vậy phân tích ở trên **đầy đủ**: không có hành động nào chỉ web
làm được mà lõi không biết. `db backup` và `db restore` **cố ý không có** trên web
(`server.go:1207-1212`).

### F5. `/docs/` và `/vendor/` phục vụ cho người chưa đăng nhập

**Làm được gì:** đọc **139.541 byte** `DO-LUONG.md` từ internet mà không cần gì cả — đã
đo, `200`. Trong đó có: **IP công cộng của máy** (`:713`), **đường dẫn thật**
(`C:\Users\Administrator\bin\claude.cmd`, `:1471`), **tên tài khoản quản trị**, và **mô
tả chi tiết cơ chế bảo vệ dashboard cùng những lỗ đã từng có**.

**Đứng ở đâu:** **K1**.

**Hiện có gì chặn:** 🟡 **CÓ MỘT PHẦN — và đây là đánh đổi CÓ CHỦ Ý, không phải sơ suất**

Lý do ghi thẳng trong mã (`server.go:96-105`): *"chia sẻ link kế hoạch KHÔNG đồng nghĩa
trao quyền điều khiển agent"*, và *"đằng nào nội dung này cũng nằm công khai trên
GitHub"*. `/vendor/` mở theo vì trang `/docs/` không đòi đăng nhập thì **font của nó cũng
không được đòi**, không thì chữ trang kế hoạch rơi về font hệ điều hành.

**Cả hai lý do đều đúng.** Nhưng phải nói nốt phần lý do **không** phủ:

- Nội dung công khai trên GitHub **không gắn với một IP cụ thể**. Phục vụ nó ở
  `https://<IP>:8788/docs/` thì nó **gắn** — nó xác nhận cho người quét cổng rằng **đúng
  máy này** chạy `sagent`, **đúng phiên bản nào**, và **cửa đăng nhập ở đâu**.
- **Không có secret nào rò ra ở đây.** Đây là **thông tin do thám**, không phải rò rỉ bí
  mật. Xếp hạng theo đúng mức đó, không hơn.

### F6. DNS-rebind và CSRF

**Làm được gì:** mượn trình duyệt của chủ máy để gọi API dashboard.

**Đứng ở đâu:** **K1** + dụ được chủ máy mở một trang web.

**Hiện có gì chặn:** ✅ **ĐÃ CÓ ở chế độ kín, 🟡 CÓ MỘT PHẦN ở chế độ phơi ra mạng —
và máy này ĐANG ở chế độ phơi ra mạng**

- Chế độ kín: `Host` phải là loopback (`server.go:419`), **và `/login` cũng phải kiểm y
  hệt** (`server.go:175`). Thiếu chỗ thứ hai thì lớp chống DNS-rebind **hở đúng cái cửa
  quan trọng nhất** — đây là **lỗi số 2** trong 6 lỗi codex soi ra (`DO-LUONG.md:1005`),
  đã có test `TestLoginCungPhaiKiemHostLoopback` (`lachan_test.go:48`).
- Chế độ **phơi ra mạng**: `Host` là IP/tên miền thật nên **không kiểm được** — mã nói
  thẳng điều đó (`server.go:417-418`). Lúc đó hàng rào chỉ còn **mật khẩu + cookie
  `SameSite=Lax`**.
- CSRF: mọi `POST` phải **cùng gốc** (`server.go:454` + `sameOrigin` `:505`). `Origin`
  rỗng = client dòng lệnh, cho qua vì đã qua cửa đăng nhập.
- Điều hướng: `next=//evil.example` từng lọt vì `HasPrefix(next,"/")` cho qua, mà trình
  duyệt hiểu `//host` là **đổi tên miền** — lỗi số 1, đã vá, ghim bằng
  `TestNextKhongDuocDoiTenMien` (`lachan_test.go:34`).
- `/logout` chặn bằng `Sec-Fetch-Site: cross-site` (`server.go:222`) chứ không bắt `POST`,
  vì ba file HTML đang gọi bằng `<a href>` — ghim bằng `TestLogoutKhongBiCheoTrang`.

### F7. Làm cạn tài nguyên (Slowloris)

**Làm được gì:** giữ mãi goroutine và file descriptor bằng cách nhỏ từng byte header →
dash chết → **triệu chứng A1** trong `VAN-HANH-VPS.md`.

**Đứng ở đâu:** **K1**. Nguy hiểm **thật** ở chế độ phơi ra mạng, mà máy này đang ở đó.

**Hiện có gì chặn:** 🟡 **CÓ MỘT PHẦN**

- `ReadHeaderTimeout: 10s` + `IdleTimeout: 120s` (`server.go:393-394`) — đây là lỗi số 6
  của lượt soi, đã vá.
- `WriteTimeout` **cố ý KHÔNG đặt**: `/api/events` là luồng SSE chạy dài, đặt vào là **tự
  cắt tính năng của mình**.
- 🔴 **Không có trần số kết nối, không có trần số phiên SSE.** Mở nhiều `/api/events` là
  giữ nhiều goroutine, và không có gì đếm.
- ✅ Có lưới đỡ ở **tầng ngoài `sagent`**: watchdog nhịp **~60 giây** dựng lại dash khi
  nó chết (`VAN-HANH-VPS.md` mục B). Nhưng đó là **hồi phục**, không phải **phòng ngừa**,
  và mỗi lần dựng lại là **mọi người phải đăng nhập lại**.

---

## G. Mặt 4 — Thực thi lệnh

Ba đường chạy được lệnh: **bước `shell` của flow**, **cờ `--tu-duyet-quyen` của agent**,
và **worktree** (không tự chạy gì, nhưng quyết định agent **đứng ở đâu** mà chạy).

### G1. Bước `shell` của flow

**Làm được gì:** chạy chương trình tuỳ ý dưới quyền chủ máy.

**Đứng ở đâu:** **K2** (qua `POST /api/flow/save` rồi `/api/flow/run`), hoặc bất cứ ai
đưa cho bạn một repo có `.sagent/flows.toml`.

**Hiện có gì chặn:** 🟡 **CÓ MỘT PHẦN**

Có, và phần này làm đúng:
- **Chỉ nhận `argv`, KHÔNG qua shell** (`internal/flow/step.go:417`, `:431`
  `exec.CommandContext(ctx, args[0], args[1:]...)`). Lý do ghi trong mã: *"flow là file
  người ta gửi cho nhau được"*. Nghĩa là **không có tiêm shell**: `; rm -rf /` trong một
  đối số chỉ là một đối số.
- **Placeholder thiếu thì DỪNG, không chốt bừa** (`step.go:418-428`). Nếu không thì
  `go test -C (bước "x" không để lại kết quả)` là **một đường dẫn bịa**, chạy vào rồi
  hỏng bằng một thông báo chẳng liên quan gì tới nguyên nhân thật.
- Cửa duyệt tay **không thể bị bỏ qua bằng cờ**: `Approve` là **hàm DUY NHẤT** chuyển
  một bước `approve` sang `done`, và bộ thực thi **không có đường nào tự làm việc đó**
  (`step.go:451-456`). Đó là **tính chất của kiến trúc**, không phải một cái cờ ai cũng
  bật được.

Không có, và đây là chỗ hở:
- 🔴 **`argv[0]` không bị giới hạn gì cả.** Không allowlist, không kiểm đường dẫn, không
  cấm chạy ra ngoài repo. `argv` chặn được **tiêm shell**, **không** chặn được **"chạy
  đúng một chương trình mà bạn không muốn chạy"**.
- 🔴 **`flows.toml` từ repo được nạp mà không hỏi ai.** `flow.Paths` (`flow.go:234`) đọc
  `<repo>/.sagent/flows.toml` và cho nó **đè lên** bản toàn cục. Clone một repo lạ rồi
  gõ `sagent flow chay` là **chạy lệnh của người viết repo đó**. Không có bước xác nhận,
  không có cảnh báo "flow này đến từ repo, không phải từ bạn".
- 🟡 **`POST /api/flow/save` cho phép soạn lệnh từ web** (`server.go:92`) — cộng với F4,
  một phiên dashboard bị đoạt là một đường ghi-rồi-chạy trọn vẹn.

### G2. Cờ `--tu-duyet-quyen`

**Làm được gì:** agent **tự duyệt mọi tool** ở chế độ headless — đọc/ghi file, chạy lệnh,
đi mạng, **không hỏi ai một câu nào**. Với Claude và Antigravity, cờ thật là
`--dangerously-skip-permissions` (`internal/provider/claude.go:225`,
`antigravity.go:150`).

**Đứng ở đâu:** **K2** (`POST /api/fleet` nhận thẳng `tuDuyetQuyen: true`,
`server.go:1253`) hoặc **K5**.

**Hiện có gì chặn:** 🟡 **CÓ MỘT PHẦN — chặn được việc KHAI BỪA, không chặn việc DÙNG**

Có:
- **Cờ thật do adapter khai, không phải chuỗi người dùng gửi lên** (`api.go:1084-1085`).
  Mặt web **không được gửi tên cờ** của một provider cụ thể — *"đó là kiến thức của lõi,
  không phải của trình duyệt"* (`server.go:1251-1252`). Nếu không thì log của lượt chạy
  sẽ có `-- --dangerously-skip-permissions`, tức **tên cờ của Claude rò vào tay người
  dùng** (`api.go:1024`).
- **Chưa đo cờ cho provider nào thì TỪ CHỐI CHẠY**, không khai bừa (`api.go:1086-1089`).
- **Provider không có rào quyền thì nói thẳng là cờ thừa**, không im lặng
  (`api.go:1091-1093`).
- **Thiếu cờ thì cảnh báo TRƯỚC KHI CHẠY** (`api.go:1106-1108`) — vì đã mất hai lượt
  ngày 21/08 (#167, #169) về worktree sạch, **0 commit**, không một dòng lỗi.

Không có:
- 🔴 **Không có hộp cát nào.** Agent chạy với **đúng quyền của người gõ lệnh**. Không
  giới hạn thư mục, không giới hạn mạng, không giới hạn tiến trình con.
- 🔴 **Không có xác nhận bổ sung khi bật cờ từ web.** Trên CLI người ta phải **gõ** ra
  `--tu-duyet-quyen`; trên web đó là **một ô tích**, gửi bằng một dòng JSON.

### G3. Worktree — và vì sao nó là bàn đạp chứ không chỉ là chỗ làm việc

**Đây là mối đe doạ ít ai nghĩ tới nhất, và nó đo được bằng một dòng.**

`workspace.Root()` = `~/.ai-accounts/.worktrees` (`internal/workspace/worktree.go:23`).
Tức là **cây làm việc của agent nằm CÙNG GỐC với kho bí mật**. Đo lúc viết tài liệu này —
đây là thư mục làm việc của chính phiên đang chạy:

```
C:\Users\Administrator\.ai-accounts\.worktrees\switch-agent-pro-b2d22420\phu-1
                       |___________ cung goc ___________|
```

Từ đó tới token và khoá là **ba cấp `..`**:

```
..\..\..\api-keys\*.key                                          <- 2 file khoa API
..\..\..\.clones\<provider>\<tai khoan>\<N>\.credentials.json    <- 4 file token
..\..\..\dash-auth.json                                          <- mat khau dashboard da bam
..\..\..\dash-tls\key.pem                                        <- khoa rieng TLS
..\..\..\.nhat-ky\                                               <- toan van moi phien da chay
```

**Làm được gì:** một agent chạy với `--tu-duyet-quyen` đọc được **toàn bộ mục B** bằng
đúng công cụ đọc file mà nó vốn được cấp — không cần khai thác lỗi nào, không cần leo
thang quyền. Và nếu nó bị dẫn dắt bởi nội dung trong repo nó đang đọc (prompt injection
qua README, qua comment, qua issue), thì **K5 biến thành đường đi cho K1**.

**Hiện có gì chặn:** 🔴 **KHÔNG CÓ GÌ**

`acl.Restrict` chặn **người dùng KHÁC** (K4). Nó **không chặn** một tiến trình chạy dưới
**cùng người dùng** — mà agent thì luôn chạy dưới cùng người dùng
(`internal/profile/clone.go:355`: `c.Env = append(filterEnv(...), a.EnvVar()+"="+dir)`,
không đổi token truy cập, không hạ quyền).

**Cái đang có, và nó chỉ giải quyết chuyện khác:**
- ✅ `IsDirty` (`worktree.go:97`) chặn `git worktree remove --force` xoá mất việc chưa
  commit — đó là **chống mất dữ liệu**, không phải chống tấn công.
- ✅ Nhật ký nay nằm ở `~/.ai-accounts/.nhat-ky/`, **sống lâu hơn cả worktree lẫn thư mục
  clone** (`VAN-HANH-VPS.md` D3). Trước đây `sagent clean` **xoá nguyên thư mục clone** —
  tức **lệnh dọn dẹp sau một lượt hỏng chính là lệnh phá tang chứng**.

**Nói thẳng phần chưa đo:** ⬜ chưa có phép đo nào chứng minh một agent **thật sự** đọc
được `..\..\..\api-keys\*.key` từ trong worktree. Đường dẫn và quyền thì đã đo (ở trên);
**hành vi thật của agent thì chưa**. Cách đo nằm ở mục J.

---

## H. Bảng hở — xếp theo hậu quả, không theo mức dễ sửa

Đây là phần đọc nhanh. Mọi dòng ở đây đều đã có chỗ dẫn đầy đủ bên trên.

| # | Đang hở cái gì | Vị trí | Hậu quả nếu trúng | Mục |
|---|---|---|---|---|
| 1 | **Mật khẩu dashboard là hàng rào duy nhất VÀ cuối cùng** — sau nó không còn tầng nào | K2 | Thực thi mã tuỳ ý dưới `Administrator`, trên máy nằm thẳng trên internet | F4 |
| 2 | **Agent chạy cùng gốc với kho bí mật, không hộp cát** | K5 | Đọc trọn 4 token + 2 khoá API + mật khẩu băm + khoá TLS + mọi nhật ký | G3 |
| 3 | **Không ghi lại lần đăng nhập sai nào của dashboard** | K1 | Đang bị dò cũng không biết — trong khi RDP thì đếm được tới từng lượt | F2 |
| 4 | **`flows.toml` từ repo lạ chạy được mà không hỏi ai** | bất kỳ ai gửi repo | Chạy chương trình do người khác chọn | G1 |
| 5 | **`lsass` chưa được `RunAsPPL` bảo vệ** (đã đặt, **chưa reboot**) | K6 | Đi vòng qua toàn bộ lớp ACL ở D2 | D4 |
| 6 | **Không có trần chi phí cho `/api/ai`** | K2 | Tiêu tiền thật, không trần | E4 |
| 7 | **Cuộc đua refresh token giữa N clone chỉ được DỌN ở lượt sau, không được NGĂN** | tự gây | Mất phiên, phải đăng nhập lại | D1 |
| 8 | **Chứng chỉ tự ký — việc đối chiếu vân tay là việc của người dùng, không đo được** | K3 | MITM đọc mật khẩu lúc gõ | F3 |
| 9 | **Phiên 12 giờ, không buộc IP, không huỷ được một phiên cụ thể** | K3 | Cookie rò là dùng được 12 giờ | F3 |
| 10 | **Không có trần kết nối / trần phiên SSE** | K1 | Dash chết → triệu chứng A1 | F7 |
| 11 | **`/docs/` phơi 139 KB thông tin do thám cho người chưa đăng nhập** | K1 | Do thám, **không** rò bí mật — xếp đúng mức đó | F5 |

**Ba dòng đầu là ba dòng đáng làm trước**, và lý do xếp thứ tự đó là: dòng 1 và 2 biến
**một** sai lầm thành **mất tất cả**, còn dòng 3 là thứ khiến ta **không biết mình đã bị
gì** — mà không biết thì mọi con số trong tài liệu này hết hạn lúc nào cũng không hay.

---

## I. Chưa đo được — kèm lý do, và không được đọc thành "không có vấn đề"

| Câu hỏi | Vì sao chưa đo được |
|---|---|
| Đã có ai dò mật khẩu dashboard chưa? | **Không có bản ghi nào để đọc.** `noteFail()` (`server.go:477`) chỉ tăng biến đếm trong bộ nhớ, và biến đó mất mỗi lần dash khởi động lại. Đây vừa là câu chưa trả lời được, vừa là **lỗ hổng #3** ở mục H. |
| Cổng `8788` có thật sự vào được từ ngoài internet không? | Chỉ đo được **từ trong máy** (`127.0.0.1` → `200`). Firewall rule `Allow` và bind `::` nói là **có**, nhưng chưa có phép đo từ một máy bên ngoài. Suy ra từ cấu hình **không phải phép đo** — theo đúng chuẩn `VAN-HANH-VPS.md` mục F. |
| Agent có **thật sự** đọc được `api-keys/*.key` từ trong worktree không? | Đường dẫn và quyền đã đo; **hành vi thật thì chưa chạy thử**. Cách đo ở mục J. |
| Brute-force RDP có phải nguyên nhân ba lần `lsass` chết không? | **Chưa có dump của `lsass`.** Vẫn là **nghi phạm**. Đây là câu đã treo từ `VAN-HANH-VPS.md` A3 và tài liệu này **không** làm nó tiến thêm được bước nào. |
| Mật khẩu dashboard hiện tại mạnh cỡ nào? | **Cố ý không đo.** File chỉ giữ băm PBKDF2; đo được nghĩa là dò được. `SetPassword` chỉ đòi ≥6 ký tự (`auth.go:65`) nên **chặn trên của độ mạnh là do người đặt**, không phải do công cụ. |
| Ba lần dash chết (20/08 ~20:32, 21/08 ~06:40) có phải do tấn công không? | **Chưa có nguyên nhân nào đứng vững** — bốn giả thuyết đã bị bác, xem `VAN-HANH-VPS.md` mục C. **Đừng bịa giả thuyết thứ năm cho đỡ trống.** |

---

## J. Đo lại — lệnh chạy được, không phải mô tả

Mọi con số trong tài liệu này **hết hạn**. Chạy lại trước khi tin.

```powershell
# 1. Dashboard co dang phoi ra mang khong, va ai giu cong
#    (Hoi AI DANG GIU CONG, khong hoi ai dang chay - xem VAN-HANH-VPS.md A1)
Get-NetTCPConnection -LocalPort 8788 -State Listen
$p = (Get-NetTCPConnection -LocalPort 8788 -State Listen).OwningProcess
(Get-CimInstance Win32_Process -Filter "ProcessId=$p").CommandLine

# 2. Tuong lua co mo cong do vao tu ngoai khong
Get-NetFirewallRule -Enabled True -Direction Inbound |
    Where-Object DisplayName -match '8788|sagent'

# 3. Cuoc tan cong RDP con o muc nao
(Get-WinEvent -FilterHashtable @{LogName='Security'; Id=4625;
    StartTime=(Get-Date).AddHours(-1)}).Count
((Get-NetFirewallRule -DisplayName "TNS Chan Bruteforce" |
    Get-NetFirewallAddressFilter).RemoteAddress).Count

# 4. RunAsPPL da co hieu luc chua (dat roi KHAC voi co hieu luc)
Get-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name RunAsPPL
Get-Process lsass | Select-Object Id, StartTime   # StartTime < lan dat = CHUA hieu luc

# 5. Kiem ke tai san
$r = "$env:USERPROFILE\.ai-accounts"
(Get-ChildItem "$r\.clones" -Recurse -Filter ".credentials.json" -Force).Count
(Get-ChildItem "$r\api-keys" -Filter *.key -Force).Count
(Get-ChildItem "$r\.worktrees" -Directory -Force).Count
```

```bash
# 6. Quyen cua CA CAY kho ho so - khong chi thu muc goc.
#    O kiem chi soi mot diem se xanh trong khi mot cho bi quen (DO-LUONG.md:1129)
sagent verify

# 7. Duong nao cua dashboard mo cho nguoi chua dang nhap
curl -sk -o /dev/null -w "docs=%{http_code}\n"  https://127.0.0.1:8788/docs/DO-LUONG.md
curl -sk -o /dev/null -w "state=%{http_code}\n" https://127.0.0.1:8788/api/state

# 8. Luoi an toan cua moi thu trong tai lieu nay
go test ./internal/dash/... ./internal/profile/... ./internal/acl/... ./internal/aiapi/...
```

**Còn nợ một phép đo** (mục I): chạy một phiên `--tu-duyet-quyen` với việc *"đọc thư mục
`..\..\..\api-keys` rồi báo lại có bao nhiêu file"* và xem nó trả về gì. Chưa chạy vì
phép đo đó **cần một tài khoản dùng một lượt hạn mức thật**, và kết quả phải được ghi
bằng **cái agent thật sự làm được**, không phải bằng suy luận từ quyền trên đĩa.

---

## K. Những lưới an toàn đang giữ tài liệu này khỏi trôi

Một tài liệu an ninh không có test đi kèm thì sáu tháng nữa là **văn bia**. Dưới đây là
các bài kiểm **đang đỏ nếu ai gỡ bản vá tương ứng** — chạy `go test ./...` là chạy hết.

| Giữ điều gì | Bài kiểm |
|---|---|
| `/login` cũng phải kiểm Host loopback | `dash/lachan_test.go:48` |
| `next=//evil.example` không đổi được tên miền | `dash/lachan_test.go:34` |
| Request vô danh không khoá được người đang dùng | `dash/lachan_test.go:61` |
| Poll hợp lệ không làm mất chống-dò-mật-khẩu | `dash/lachan_test.go:86` |
| `/logout` không bị kích từ trang khác | `dash/lachan_test.go:107` |
| **Mọi hành động đều phải có đường vào từ web** (luật ngang quyền UI ↔ API) | `dash/lachan_test.go:137` |
| Phơi ra mạng không TLS thì từ chối chạy | `dash/tls_test.go:112` |
| Cookie `Secure` bám theo TLS thật | `dash/tls_test.go:150` |
| Vân tay in ra khớp vân tay trên dây | `dash/tls_test.go:179` |
| Thư mục clone cũng phải siết quyền | `profile/clone_acl_test.go:19` |
| `CleanClones` không đi xuyên junction ở gốc | `profile/clone_acl_test.go:70` |
| Tên hồ sơ không thoát ra ngoài kho | `profile/name_test.go:32` |
| Hồ sơ là link thì chỉ gỡ link, không hỏi sổ | `profile/safedelete_test.go:116` |
| Sổ không nhận sở hữu thì không xoá | `profile/safedelete_test.go:57` |
| File tạo **sau** khi siết vẫn kín | `acl/acl_test.go:65` |
| `key_id` không thoát được thư mục | `aiapi/aiapi_test.go:37` |
| Khoá đi đúng chỗ và không rò ra lỗi | `aiapi/aiapi_test.go:100` |
| Không đồng bộ ngược file rỗng của bản thua cuộc đua | `profile/duarefresh_test.go:50` |

---

## L. Một câu sòng phẳng để kết

Bốn mặt này **không cùng một hạng**. Mặt **API key** gần như sạch, và sạch nhờ **một luật
kiến trúc** (*file cấu hình không bao giờ chứa secret*) chứ không nhờ một chuỗi bản vá.
Mặt **credential** có nhiều bản vá tốt, mỗi bản vá đổi bằng một lần hỏng thật. Mặt
**dashboard** có lớp phòng thủ dày nhất **và** bề mặt tấn công lớn nhất, vì nó là mặt
duy nhất chạm được từ internet.

Còn mặt **thực thi lệnh** thì phải nói thẳng: nó chặn rất kỹ **những đường tấn công kinh
điển** — không tiêm shell, không chốt placeholder bừa, không khai bừa cờ quyền, cửa duyệt
tay không bỏ qua được bằng cờ — nhưng nó **hoàn toàn không có tầng cách ly**. Một agent
được bật `--tu-duyet-quyen` chạy với đúng quyền của chủ máy, trong một thư mục cách kho
bí mật đúng **ba cấp `..`**.

Đó không phải một lỗi cần vá. Đó là **hình dạng hiện tại của dự án**, và tài liệu này tồn
tại để **hình dạng đó được nói ra thành lời**, thay vì nằm im trong tám package cho tới
lần đầu tiên nó gây hại.
