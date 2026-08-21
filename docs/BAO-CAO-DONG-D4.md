# Báo cáo lượt: đóng ô Đ4 (mức ĐỎ) và viết lại C1 cho khớp mã

- **Ngày**: 21/08/2026
- **Nhánh**: `sagent/tns-1`
- **Ô nợ đụng tới**: `docs/SO-NO-DO-LUONG.md` — **Đ4** (đỏ), **C1** (nửa Cursor).
  Sinh thêm một ô mới: **V3**.

---

## 1. Đã làm

### VIỆC 1 — Đ4 nửa Antigravity: đổi lời khai, không đo

`internal/provider/antigravity.go`: `Chua(NLHanToken, ...)` → **`Khong(NLHanToken, ...)`**.

Lý do giữ nguyên lý do cũ, chỉ đổi **tư cách** của nó từ "khoảng trống" thành
"kết luận": token nằm trong Windows Credential Manager dưới khoá
`gemini:antigravity`; `CredRead` trả về **cả blob**, không có cách hỏi riêng mốc
hết hạn. Mở chính thứ cần bảo vệ để đổi lấy **một dấu thời gian** là đánh đổi
tồi. Theo tiền lệ `Khong(NLHanToken)` của Grok — nhưng bằng chứng viết rõ chỗ
**khác** Grok: ở Grok hạn **không tồn tại** (API key, không OAuth), ở đây hạn
**có** nhưng nằm sau một cánh cửa ta cố ý không mở.

`TokenExpiry` vẫn trả `false` — **không đổi hành vi, chỉ đổi lời khai**. Bình
luận hàm ghi thẳng cái giá phải trả (xem mục 2).

### VIỆC 2 — Đ4 nửa Cursor: ĐO THẬT, đọc được hạn

**Trước hết là tìm ra chỗ lưu thật.** `~/.cursor/auth.json` **không tồn tại**;
file thật ở `%APPDATA%\Cursor\auth.json` (893 byte) — đúng chỗ mà `EnvVar()` và
`PrivateFiles()` trong `cursor.go` đã chỉ từ đầu. Ô này mở không phải vì phép đo
khó mà vì **đi tìm sai thư mục**.

**Hình dạng file, đo chứ không đoán**: có **ĐÚNG HAI khoá**, `accessToken` và
`refreshToken`, cả hai là JWT `alg:HS256`. **Không có trường dấu-thời-gian nào ở
tầng ngoài.** Nên lời dặn của chính mã — *"đừng đoán tên trường"* — hoá ra còn
đúng hơn ý định ban đầu: **không có trường nào để mà đoán**. Mốc nằm trong payload
JWT, y hệt Codex.

Claim đọc được (hai token giống hệt nhau từng claim):

| claim | giá trị | nghĩa |
|---|---|---|
| `iss` | `https://authentication.cursor.sh` | |
| `aud` | `https://cursor.com` | |
| `scope` | `openid profile email offline_access` | |
| `type` | `session` | |
| `time` | `1787022539` | 2026-08-18T03:08:59Z — lúc đăng nhập |
| `exp` | `1792206539` | 2026-10-17T03:08:59Z |

`exp − time = 5 184 000 s` = **đúng 60 ngày**.

**Chạy thật trên máy này sau khi sửa mã**:

```
HasToken=true
TokenExpiry ok=true  exp=2026-10-17T03:08:59Z  con=1364h  (≈ 56,8 ngày)
```

Trước bản vá, cùng hồ sơ đó trả `ok=false` — **không phân biệt được với "chưa
đăng nhập"**.

Đã sửa `internal/provider/cursor.go`: `TokenExpiry` đọc claim `exp`, cộng một
hàm nhỏ `hanTuJWT`. Bảng khai lên `Duoc(NLHanToken, ...)` kèm nguyên văn số đo.

**Đọc REFRESH token, không phải access token** — theo tiền lệ `claude.go` (trả
hạn access token từng làm cổng kiểm **chặn oan** lượt chạy #39). Hôm nay hai mốc
bằng nhau nên phép đo thật *không* phân biệt được hai lựa chọn; bài kiểm phân
biệt hộ bằng hai JWT giả có `exp` khác nhau.

**Bài kiểm**: `internal/provider/token_cursor_test.go` — 6 bài + 8 ca con. Ghim
hình dạng thật của file vào bình luận, ghim lựa chọn refresh-chứ-không-access, và
ghim **mọi ngõ hỏng phải trả `false`** (không JSON, không JWT, payload không
base64, thiếu `exp`, `exp=0`, không có file). JWT trong bài kiểm là **giả** — chữ
ký không được kiểm nên giả là đủ, và một token thật trong mã nguồn là một rò rỉ.

### VIỆC 3 — C1: đã MỞ MÃ RA ĐỌC rồi viết lại

Xác nhận bằng mã, không bằng tiêu đề commit:

- `internal/provider/cursor.go:272` — `DocKetQua` trả thẳng `docKetQuaCursor(raw)`.
  Không còn `(KetQua{}, false)`.
- `internal/provider/cursor.go:292` — `Duoc(NLKetQuaCoCauTruc, ...)`.
- Bộ đọc và bài kiểm của nó có mặt: `ketqua_cursor.go`, `ketqua_cursor_test.go`.

*(Số dòng là của bản mã **sau** lượt này, muộn hơn `bacc137`; nội dung không đổi,
chỉ trôi dòng vì `cursor.go` dài thêm. Sổ cũ ghi `:173`/`:193`.)*

Đã đổi tiêu đề **C1** thành *"Codex CHƯA ĐO cách đọc kết quả có cấu trúc"*, thay
băng ⚠ *"đã lạc hậu một nửa"* bằng ghi chú ✅ đóng cho nửa Cursor, và gỡ Cursor
khỏi các gạch đầu dòng dùng chung.

**Phần Codex: không đụng.** Mọi câu về Codex giữ **nguyên từng chữ**, kể cả số
dòng cũ `codex.go:226,248` (nay thật ra là `:254` và `:272`) — cố ý **không** sửa
để khỏi giẫm chân phiên đang làm nửa đó.

### Kiểm cảnh báo hạm đội (theo lưu ý được giao)

Chốt trong `internal/api/api.go` chỉ chạy khi `TokenExpiry` trả `ok=true`. Có
**hai** chỗ gọi, không phải một:

| Chỗ | Trước | Sau |
|---|---|---|
| `api.go:264` `ProfileList` | Cursor để trống `HanToi`/`HetHan` | Cursor điền được → `sagent ds` nói được "hết hạn lúc mấy giờ" |
| `api.go:1065` cảnh báo trước khi bung hạm đội | Cursor **không bao giờ** cảnh báo | Cursor **có** chạy nhánh cảnh báo |
| cả hai, với Antigravity | không chạy | **vẫn không chạy — không đổi** |

Cửa sổ 60 ngày dài hơn mọi lượt chạy nên nhánh *"còn dưới 2 tiếng"* của Cursor
gần như sẽ không kêu. **Đó là đúng, không phải hỏng**: cái đổi thật là nó thôi im
lặng vì KHÔNG BIẾT và bắt đầu im lặng vì ĐÃ BIẾT là còn hạn.

### Xanh

```
go build ./...              -> OK
go vet ./internal/provider/ -> OK
go test ./...               -> toàn bộ 23 gói ok, không gói nào FAIL
```

Conformance (`KiemNangLuc`) xanh với cả ba lời khai mới.

---

## 2. Sự cố

**Có ba, nói thẳng.**

### 2.1 — Tự gây: dấu `\` bị nuốt, suýt commit chữ hỏng

Viết `Cursor\auth.json` qua script Python không dùng chuỗi thô, `\a` biến thành
ký tự BEL (`0x07`). Kết quả: bình luận đọc thành `Cursoruth.json`, và **một chuỗi
Go có `\C` — cái này là lỗi biên dịch thật**. Đã bắt bằng `go build` và `grep`,
đã sửa, đã kiểm `BEL count == 0` trong cả `.go` lẫn `.md`. Cùng họ với bẫy
em-dash trong `.ps1`: **ký tự lạ làm hỏng file một cách im lặng**.

### 2.2 — Tìm thấy: Cursor khai xanh `NLDanhTinh` mà `Identity()` LUÔN trả rỗng

Không tìm nó — nó rơi ra khi mở `auth.json` thật.

Bảng khai cũ: `Duoc(NLDanhTinh, "trường email/userEmail/user_email trong
Cursor\auth.json")`. File thật **không có khoá nào trong ba khoá đó**; `sub`
trong JWT là mã đục `google-oauth|<id>`, không phải email. Chạy trên hồ sơ **đang
đăng nhập**: `Identity=""`.

Đây đúng thứ mà chính sổ nợ xếp **nguy hiểm hơn `ChuaDo`**: ô khai *"LÀM ĐƯỢC"*
mà lõi không chặn và người vận hành tưởng đã kiểm. So với **V2** (Antigravity
cũng không đọc được email): V2 khai `ChuaDo` — trung thực; ô này nói dối theo
hướng khoe.

**Vì sao conformance không bắt được** — chỗ đáng nhớ nhất của lượt này: phép dò
`NLDanhTinh` trong `nangluc.go` là **một chiều** (`haiChieu: false`), chỉ kết
luận được chiều *"khai chưa đo mà lại trả giá trị thật"*. Chiều *"khai làm được
mà luôn trả rỗng"* **không có ai canh**. Cùng lỗ hổng đó đang che cho
`NLCoTuHoSo`, `NLKetQuaCoCauTruc` và `NLHanToken`. Một chiều là **đúng** (cần hồ
sơ thật mới dò được), nên bịt lỗ không phải bằng cách đổi phép dò.

**Đã làm gì**: hạ xuống `Chua(NLDanhTinh, ...)` kèm nguyên văn phép đo, và ghi
thành ô **V3** trong sổ nợ.

**⚠ ĐÂY LÀ VIỆC NGOÀI PHẠM VI ĐƯỢC GIAO — nói ra để chủ dự án quyết.** Lượt này
được giao Đ4 + C1, không được giao danh tính Cursor. Tôi vẫn sửa vì để nguyên một
ô **xanh mà tôi vừa tự tay chứng minh là sai** là điều tệ hơn. Nếu muốn phiên
khác cầm ô đó thì đây là một dòng revert trong `cursor.go` — phần Đ4 không phụ
thuộc vào nó.

### 2.3 — Không đo được: token Cursor có xoay vòng khi refresh không

Đây mới là thứ cảnh báo hạm đội **thật sự** sợ (mục 20/08: một bản refresh là
N−1 bản kia chết). Chỉ có bằng chứng **gián tiếp**:

```
%APPDATA%\Cursor\auth.json     mtime 2026-08-18 10:08:58 (+07)  <- đúng giây đăng nhập
~\.cursor\cli-config.json      mtime 2026-08-21 00:46:44 (+07)
~\.cursor\statsig-cache.json   mtime 2026-08-21 00:51:08 (+07)
```

Hai file dưới bị ghi lại bởi chính các lượt chạy thật của ô Đ3 ngày 21/08; file
token thì **không**. Qua ba ngày dùng, `cursor-agent` không đụng vào nó. **Đó là
dấu hiệu, không phải phép đo** — chưa ép nó refresh nên chưa biết lúc refresh thì
sao. Đã ghi thẳng câu này vào bình luận mã và vào bằng chứng của bảng khai, để
không ai đọc `Duoc(NLHanToken)` thành "đã hiểu hết về token Cursor".

---

## 3. Bước tiếp theo

1. **C3 và C6** — hai ô còn lại mới chỉ gắn nhãn *"đã lạc hậu"*, chưa viết lại
   theo khuôn đóng. Rẻ nhất còn lại: chỉ đối chiếu mã, không đo gì.
2. **V3** — đo cách đọc email Cursor qua `cursor-agent status`. Cần cân đánh đổi:
   chạy một tiến trình con mỗi lần vẽ card là cái giá khác hẳn đọc file.
3. **Bịt lỗ "khai xanh mà luôn trả rỗng"** — không sửa phép dò một chiều (nó
   đúng), mà thêm một bài kiểm chạy trên **hồ sơ thật** khi có, đối chiếu bốn ô
   `NLCoTuHoSo` / `NLKetQuaCoCauTruc` / `NLHanToken` / `NLDanhTinh`. Đây là bài
   học đắt nhất của lượt này và nó **chưa được đóng**.
4. **Xoay vòng token Cursor** (2.3) — ép một lần refresh rồi xem `auth.json` có
   bị ghi lại không. Cửa sổ 60 ngày làm việc này khó dựng tự nhiên.
5. **Đ1+Đ2 Codex** và **C5** vẫn đứng nguyên trong "thứ tự đề nghị đo".

---

## 4. Nên xài model gì, effort nào

| Việc | Model | Effort |
|---|---|---|
| Đóng ô nợ đỏ có đo thật (kiểu Đ4) | Opus 5 | high |
| Viết lại mục sổ cho khớp mã (kiểu C1) | Sonnet 5 | medium |
| Đối chiếu nhãn "đã lạc hậu" C3/C6 | Sonnet 5 | medium |
| Đo `cursor-agent status` cho V3 | Sonnet 5 | medium |
| Thiết kế bài kiểm bịt lỗ conformance một chiều | Opus 5 | high |
| Ép refresh để đo xoay vòng token | Opus 5 | high |
| Sửa chính tả / số dòng trong docs | Haiku 4.5 | low |

---

## 5. Muốn nói gì thì nói ở dưới

**Bài học chính**: hai nửa của một ô nợ có thể đóng bằng hai cách **ngược nhau**.
Nửa Cursor đóng bằng **đo**; nửa Antigravity đóng bằng **kết luận không đo**.
Trước lượt này cả hai cùng mang chữ `ChuaDo` nên nhìn giống nhau như đúc — mà một
cái là việc **chưa làm**, một cái là việc **đã quyết định không làm**. Ba trạng
thái của `TrangThaiNangLuc` sinh ra đúng để phân biệt chuyện đó, và ô Đ4 là bằng
chứng nó cần thật.

**Rào chắn của Đ4 sai lần thứ hai theo đúng một kiểu.** Đ3 hoãn vì tưởng máy chưa
cài `cursor-agent` — máy có. Đ4 hoãn vì tưởng phải "dựng cảnh token sắp hết hạn"
— thật ra chỉ cần tìm đúng thư mục. Cả hai lần, lý do hoãn nghe **hợp lý** và
**sai**. Câu *"một dòng CHƯA ĐO không tự hết hạn"* trong sổ nên đọc thêm một vế:
**lý do hoãn của nó cũng không tự đúng lại.**

**Rủi ro còn lại, xếp theo mức tôi lo:**

1. **Lỗ hổng conformance một chiều** (2.2) — bốn ô đang được che. Tôi bắt được ô
   Cursor vì tình cờ mở đúng file; ba ô kia chưa ai mở. Đây là rủi ro **hệ
   thống**, không phải một lỗi lẻ.
2. **Xoay vòng token Cursor** (2.3) — nay `Duoc(NLHanToken)` khai xanh, mà xanh
   dễ đọc thành "đã hiểu hết". Bằng chứng có ghi phần chưa biết, nhưng bằng chứng
   là thứ người ta đọc sau cùng.
3. **Cursor vẫn chưa có `total_cost_usd`** — bản ghi không có trường đó, nên chi
   phí Cursor vẫn là 0 và số 0 đó đọc như "miễn phí" (xem C4).

**Một chỗ tôi cố ý không tối ưu**: `hanTuJWT` trong `cursor.go` trùng logic với
đoạn giải mã JWT nằm sẵn trong `codex.go`. Gộp lại thành một hàm chung sẽ gọn
hơn, nhưng phải sửa `codex.go` — file mà một phiên khác đang làm. Trùng một hàm
mười dòng rẻ hơn một lần giẫm chân. Nếu gộp thì làm sau khi phiên kia xong.
