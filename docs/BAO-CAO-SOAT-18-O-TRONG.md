# Soát lại 18 ô còn trống trong `docs/MASTER-PLAN.md`

- **Soát lúc**: 21/08/2026, nhánh `sagent/tns-1`.
- **Vì sao soát**: đếm ô tick ra **81 xong / 18 còn = 82%**, nhưng mở ba mục
  bất kỳ trong 18 ô trống ra kiểm thì **cả ba đều đã làm xong**. Kế hoạch đang
  nói sai về chính nó, theo **chiều bi quan**.
- **Cách soát, không có ngoại lệ**: mở mã ra đọc rồi mới kết luận. *"Tên mục
  nghe giống một thứ đã làm"* không phải bằng chứng, y như *"tiêu đề commit
  không phải bằng chứng"* — bài học của chính dự án này (`docs/SO-NO-DO-LUONG.md`).
  Không dẫn được `file:dòng` hoặc lệnh chạy được thì **không tick**.

---

## Kết quả một dòng

| | Trước | Sau |
|---|---|---|
| Tổng | 81/99 = **82%** | 92,0/99 = **93%** |
| `[x]` xong | 81 | 87 |
| `[~]` xong một phần | *(không có dấu này)* | 10 |
| `[ ]` chưa làm | 18 | 2 |
| `[!]` bị chặn | *(không có dấu này)* | 0 |

**11 điểm phần trăm này KHÔNG phải việc làm thêm hôm nay.** Không dòng mã sản
phẩm nào được viết để đóng chúng. Đây là số đo cũ được ghi lại cho đúng.

---

## 1. Vì sao hai dấu là không đủ

Trước lượt soát, bản kế hoạch chỉ có `[x]` và `[ ]`. Hai dấu thì ba tình huống
khác hẳn nhau bị nhét chung vào một ô:

- một mục làm được 80% (engine flow đã chạy thật cả ngày với `when`,
  `timeout_sec`, `on_failure`, chỉ thiếu artifact/idempotency);
- một mục không làm được vì thiếu **khoá API thật** của Anthropic;
- một mục **chưa ai đụng tới** (plugin model).

Cả ba đều hiện là `[ ]`. Người đọc kế hoạch để quyết định "làm gì tiếp" nhìn vào
18 dòng giống hệt nhau và không có cách nào biết dòng nào đáng làm.

Nay có bốn dấu, ghi thành luật ngay đầu mục 7 của MASTER-PLAN:

| Dấu | Nghĩa | Luật |
|---|---|---|
| `[x]` | XONG | phải dẫn được `file:dòng` hoặc lệnh chạy được |
| `[~]` | XONG MỘT PHẦN | phải ghi rõ **phần nào xong, phần nào chưa** |
| `[ ]` | CHƯA LÀM | không ai đụng, và không có gì bên ngoài cản |
| `[!]` | BỊ CHẶN | làm được, nhưng thiếu thứ bên ngoài |

Tính điểm: `[x]`=1 · `[~]`=0,5 · `[ ]`=0 · `[!]` **không vào mẫu số** (đếm một
thứ dự án không tự gỡ được vào phần "chưa làm" là tự phạt oan).

**Ô `[!]` lượt này rỗng — và đó là kết luận, không phải sơ suất.** Đã tìm: thứ
duy nhất còn bị chặn thật là OpenRouter/Ollama (chưa có key, Ollama chưa cài),
nhưng nó nằm ở dòng `⬜` trong khối *Trạng thái* của Pha 4 chứ không phải một ô
tick, nên không vào bảng đếm.

---

## 2. Phán quyết từng mục (18/18)

### `[x]` XONG — 7 mục

| # | Mục | Bằng chứng |
|---|---|---|
| 1 | Pha 0 · Junction Windows / symlink Linux | `docs/DO-LUONG.md:112-122` — `sagent them claude:smoketest` không cần quyền quản trị, nối **17 mục dùng chung**, PowerShell xác nhận `ReparsePoint=True` cho mọi mục và `False` cho `.claude.json`. Nhánh Linux **bỏ khỏi phạm vi** theo quyết định "CHỈ WINDOWS" |
| 2 | Pha 0 · Behavior khi stream/process ngắt, reboot, config ghi dở | bốn nguồn khác nhau: `internal/aiapi/stream_test.go:131,192` · `process.KillTree` + `sagent quet` · `docs/KHAC-PHUC-SU-CO.md` mục 2 · `internal/jsonutil/jsonutil.go:28` |
| 3 | Pha 1 · Verb `verify` và `route test` đầy đủ | `cmd/sagent/main.go:402` → `internal/api/api.go:483`; `route test` = `sagent route kiem` (`cmd/sagent/route.go:96`). **Chạy thật hôm nay**, xem mục 4 dưới |
| 4 | Pha 2 · Route engine — mảnh `health` | `internal/aiapi/suckhoe.go:55` → `internal/api/api.go:381` → `cmd/sagent/route.go:96` → `/api/route/kiem` (`internal/dash/server.go:77`) |
| 5 | Pha 2.5 · Đường API (OpenAI-compatible) | `internal/aiapi/aiapi.go:168` + `sagent api <route>`; đo thật `docs/DO-LUONG.md:2188` và `:2322` |
| 6 | Pha 3 · 3 flow mẫu `fanout`/`squad`/`agents` | `internal/flow/builtin.go:10,31,56`; `sagent flow list` in **11 flow** |
| 7 | Pha 3 · Workflow board (mặt 4) | `internal/dash/web/flow.html` (954 dòng) + `internal/flow/save.go` |

### `[~]` XONG MỘT PHẦN — 9 mục

| # | Mục | Xong | Chưa |
|---|---|---|---|
| 8 | Pha 0 · Test harness không credential + redaction | HOME giả + khoá bịa; `aiapi_test.go:100` canh key không rò ra `err.Error()`; DTO allowlist `server.go:665` | **không có tầng redaction chung** (`grep -rni redact` = 0 dòng); không có bài kiểm quét repo |
| 9 | Pha 0 · Subscription | 5 harness × biến tách / nơi giữ token / headless / kết quả có cấu trúc; **cuộc đua N-clone** (`DO-LUONG.md:2431`); **hạn token** (`:2669`); state ngoài config root | **ACP** (0 dòng trong repo); **resume/cancel ở tầng harness**; nhật ký chạm file theo pha login/refresh/exit |
| 10 | Pha 0 · API | 3/6 nhà cung cấp, 1/4 giao thức; streaming + `usage` + error nguyên văn + health + model discovery | Anthropic Messages · Gemini native · OpenAI Responses (**thiếu key**); **rate-limit + `Retry-After`** (`grep 429` = 0 dòng); tool/reasoning/vision/structured-output |
| 11 | Pha 0 · Capability matrix | harness **sạch**: `sagent nang-luc --chua-do` → *"Không còn năng lực nào chưa đo"*, 5×9 ô, 0 `ChuaDo`; `internal/provider/nangluc.go:21-28` | **nửa API không có bảng nào**; `experimental` bị bỏ có chủ ý (bốn trạng thái rút thành ba) |
| 12 | Pha 0 · Threat model | cả 4 mặt đều có phân tích tấn công + vá **đo được** trong `DO-LUONG.md` | **`docs/security/THREAT-MODEL.md` không tồn tại** — artifact vẫn thiếu, chỉ còn việc gom lại |
| 13 | Pha 1 · Direct-API vertical slice | lát cắt dọc **đầy đủ** cho OpenAI-compatible: stream + usage + error + sổ `api_calls` + không rò key | đúng chữ **Anthropic Messages** (`x-api-key`, `anthropic-version`, SSE khác) — **bị chặn vì thiếu khoá** |
| 14 | Pha 3 · Engine | **10 mảnh**: DAG+cycle · output giữa step · condition · timeout · retry-backoff · cancel · approval gate · resume · failure policy 3 loại · fallback+health | **5 mảnh**: concurrency chỉ 2/5 nấc · artifact · idempotency key · route theo capability/giá · `compensate` |
| 15 | Pha 3 · Node built-in | **8/10** chạy được | `merge` khai rồi nhưng `implemented=false` (`flow.go:53`); `route` **chưa có node riêng** — mới là thuộc tính của bước `model` |
| 16 | Pha 5b · Approval gate | endpoint `/api/flow/decide` + đi đúng đường `Approve()` của CLI | **không trang web nào gọi nó** — xem mục 3 dưới |

### `[ ]` CHƯA LÀM — 2 mục

| # | Mục | Đã kiểm bằng gì |
|---|---|---|
| 17 | Bước 0 · Alias `tk`/`ccswitch` → `sagent` | `grep -ni "alias\|ccswitch" install/` = **0 dòng**; `cmd/sagent` không đọc `os.Args[0]` |
| 18 | Pha 3 · Plugin model | `grep -rni "plugin\|json-rpc" internal/ cmd/` ra **một** dòng duy nhất, và đó là chuỗi `"pluginUsage"` ở `internal/provider/claude.go:210` — một khoá JSON của Claude, không liên quan |

Cả hai xếp `[ ]` chứ không `[!]`: **không có thứ gì bên ngoài cản**. Riêng mục
17 có một chi tiết đáng nói — nó ghi lý do hoãn là *"làm khi viết installer phát
hành"*, nhưng installer đã có từ Pha 7 (`install/cai-dat.ps1`, `install/get.ps1`,
`phat-hanh.yml`). **Lý do hoãn đã hết hạn mà không ai gỡ nó xuống.** Một dòng
"chờ X" không tự hết hiệu lực khi X đã tới.

---

## 3. Thứ lượt soát này lôi ra ngoài dự kiến: một lỗ hồi quy thật

**Nút Duyệt / Từ chối trên mặt web đã biến mất im lặng.**

- Đường server còn nguyên: `/api/flow/decide` (`internal/dash/server.go:86`,
  `internal/dash/flow_api.go:159-182`), và nó gọi đúng `FlowApprove` /
  `FlowApproveOnly` mà CLI dùng.
- Nhưng `grep -rn "decide" internal/dash/web/` ra **0 dòng**. `grep -rni
  "reject\|tu-choi"` cũng không có nút nào. Người dùng mở dashboard thấy bước
  `waiting` mà **không có chỗ bấm**, phải quay về terminal.
- Đây là hồi quy chứ không phải việc chưa làm: `git log -S"flow/decide" --
  internal/dash/web/` cho thấy nút từng có ở `c55ed0b` (*"Mặt 4: workflow board
  — chạy flow và **duyệt ngay trên web**"*) và `fcf1b39`, rồi mất trong một lần
  vẽ lại giao diện sau đó.

**Vì sao không test nào đỏ.** Luật ngang quyền của dự án canh **API ↔ CLI**
(`internal/dash/lachan_test.go:151` giữ ánh xạ `flow.approve` →
`/api/flow/decide`) — nó bắt được việc *thêm nút trên UI mà quên lệnh CLI*.
Chiều ngược lại, **UI ↔ API**, không ai canh. Nút biến mất, hợp đồng vẫn đủ,
test vẫn xanh.

Hệ quả cho bản kế hoạch: dòng *"Duyệt / từ chối ngay trên web"* đang tick `[x]`
ở Pha 5c **hiện không còn đúng**. Đã hạ xuống `[~]` kèm ghi chú — đây là mục
DUY NHẤT nằm ngoài 18 ô trống mà lượt soát này đụng vào, và lý do là để lại một
`[x]` đã biết là sai thì mâu thuẫn thẳng với việc soát này tồn tại để làm.

---

## 4. Số đo chạy thật trong lượt soát

```
$ sagent nang-luc --chua-do
  Không còn năng lực nào chưa đo.

$ sagent route kiem
  Kiểm route (GET /models — không tốn token)
  ✓ deepseek     dùng được                        173ms
      model "deepseek-v4-flash" · nhà cung cấp liệt kê 2 model
  ✓ grok         dùng được                        168ms
      model "grok-4.5" · nhà cung cấp liệt kê 2 model
  Lưu ý: phép kiểm này KHÔNG biết hạn mức còn hay hết — cái đó chỉ lộ ra khi gọi thật.

$ sagent flow list          → 11 flow, có đủ fanout · squad · agents
$ sagent verify             → 5 provider + ô kiểm Windows ACL; thoát 1 vì bắt được
                              Claude 2.1.234→2.1.235 và Antigravity 1.1.16→1.1.17
```

Riêng `route kiem` là bằng chứng dứt điểm cho ô "Route engine — còn thiếu đúng
một mảnh: `health`": dòng cũ kết luận là thiếu vì `grep -ri health internal/aiapi
internal/api` không ra gì — mà mã đặt tên **tiếng Việt** (`SucKhoe`, `Kiem`).
**Phép grep sai, không phải mã thiếu.** Đúng kiểu lệch im lặng mà cả cuốn sổ nợ
đo lường tồn tại để chặn.

---

## 5. Thay đổi kèm theo trong mã

Bốn dấu cần bộ sinh trang hiểu được, nếu không thì `[~]` và `[!]` sẽ hiện trên
trang HTML **y hệt ô đã xong** — chính là lỗi mà bản kế hoạch vừa đi sửa.

- `tools/sinhkehoach/sinhkehoach.go` — `reViec` nhận thêm `~` và `!`; thêm hàm
  `lopViec()` dịch dấu sang lớp CSS. Luật cũ là *"khác `[ ]` thì vẽ tick xanh"*,
  tức mọi dấu mới đều thành "đã xong". Nhánh mặc định cố ý rơi về **chưa xong**:
  đoán sai theo chiều tô hồng tốn hơn đoán sai theo chiều dè dặt.
- CSS: `[~]` là **ô đầy một nửa** màu `--warn` (cố ý không dùng tick mờ dần —
  mờ dần đọc ra là *"xong, hơi nhạt"*, nửa ô đọc ra là *"mới được một nửa"*);
  `[!]` là **dấu chấm than** màu `--limit`, không phải ô trống.
- `internal/dash/kehoach_sinh_test.go` — `TestTrangKeHoachChoDuOViec` đếm thêm
  hai tiền tố mới. Quên chỗ này thì phép đếm bên `.md` nhỏ hơn số ô bên `.html`
  và thông báo lỗi sẽ đổ oan cho bộ sinh là *"đang nhân đôi mục"*.
- `tools/md2html.py` — bộ dựng trang `master-plan.html` ở gốc repo học cùng bốn
  dấu, và **được vá thêm `~~gạch ngang~~` → `<del>`**: nó không bật extension
  đó, nên trước 21/08 đã có **4 chỗ** lọt ra trang dưới dạng hai dấu ngã thô.
  Hai trang dựng từ cùng một file `.md` mà hiện khác nhau là lỗi.
- `docs/MASTER-PLAN.md` ↔ `internal/dash/web/docs/MASTER-PLAN.md` — bản nhúng đã
  **lệch 95 dòng** với bản gốc, nay chép lại cho khớp rồi sinh lại trang.

Đã chạy `go run ./tools/sinhkehoach/cmd/sinhkehoach`. Đếm trên trang sinh ra:
**87 `task done` · 10 `task part` · 2 `task` · 0 `task block`** — khớp đúng bảng
đếm trong MASTER-PLAN.

`go build ./...` · `go vet ./...` · `go test ./...` — **xanh cả ba**.

---

## 6. Còn lại phải làm gì (đọc được sau khi soát, trước thì không)

Đây là giá trị thật của lượt soát: 18 dòng nhìn giống nhau nay tách thành một
danh sách xếp được thứ tự.

**Nhỏ và đứng một mình:**
1. Nối lại nút Duyệt / Từ chối trên mặt web (endpoint đã có, chỉ thiếu UI) —
   và thêm bài kiểm canh chiều **UI ↔ API** để nó không mất lần nữa.
2. Alias `tk`/`ccswitch` — installer đã có sẵn chỗ để nhét.
3. Node `route` cho flow; bật node `merge` khi có cơ chế merge an toàn.

**Vừa, có sườn rõ:**
4. Gom `docs/security/THREAT-MODEL.md` từ những phân tích đã có sẵn trong
   `DO-LUONG.md` — không phải đo lại, chỉ dựng sườn tài sản → kẻ tấn công →
   đường vào → biện pháp.
5. Trần đồng thời theo **harness / provider / profile**, không chỉ trần chung.
6. Bảng năng lực cho **nửa API** (route hỗ trợ tool/vision/reasoning không).
7. Xử lý **429 + `Retry-After`** ở đường API — hiện 429 bị đối xử như mọi lỗi HTTP.

**Cần thứ bên ngoài:**
8. Anthropic Messages / Gemini native / OpenAI Responses — chờ khoá thật.
9. OpenRouter / Ollama — chờ khoá và chờ cài Ollama.

**Chưa có sườn:**
10. Plugin model (manifest TOML + executable JSON-RPC/stdio).
11. Tầng redaction chung cho nhật ký phiên.
12. `artifact` giữa các bước; `idempotency key`; failure policy `compensate`.
