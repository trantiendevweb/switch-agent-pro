# Mở `foreach` đi cùng `artifact` và `idempotent` — và chỗ phần suy luận phải đi

Nhánh: `sagent/foreach-22-08` · Ngày 22/08/2026 · Vùng đụng tới: `internal/flow/*`,
`internal/store/store.go`, `internal/api/{api,artifact}.go`, `cmd/sagent/flow_run.go`

Một commit mã (`6b000d6`) cộng commit báo cáo này.

Nghiệm thu (chạy từng lệnh riêng, `-count=1`, không cache):

| lệnh | mã thoát |
|---|---|
| `go build ./...` | **0** |
| `go vet ./...` | **0** |
| `go test ./...` | **0** — 25 gói, và **6 lượt liên tiếp** đều 0 (xem mục 2 về hai bài chớp tắt) |

---

## 1. Đã làm

### 1.1 `foreach` + `artifact` — CHỌN cú pháp, và vì sao chọn cái này

Báo cáo `#199` chặn tổ hợp này và nói thẳng: *"chưa rõ cú pháp trỏ nên thế nào,
và đoán bừa một cú pháp rồi phải đổi thì đắt hơn chờ."* Đây là câu trả lời.

**Cú pháp đã chọn — đúng MỘT dạng:**

```toml
[[flow.va-loi.step]]
  id       = "soi"
  type     = "agent"
  foreach  = "steps.liet-ke.output"
  prompt   = "Rà soát {{item}}, ghi bản vá ra {{artifact_dir}}/ban-va.diff"
  artifact = { ban-va = "ban-va.diff" }

[[flow.va-loi.step]]
  id    = "gom"
  needs = ["soi"]
  type  = "shell"
  run   = ["python", "gop.py", "--danh-sach", "{{artifacts.ban-va.danh_sach}}"]
```

**Bố cục trên đĩa:**

```
artifacts/run-52/soi/
                 ├── 1/ban-va.diff          ← lượt lặp 1  ({{index}} == 1)
                 ├── 2/ban-va.diff
                 ├── 3/ban-va.diff
                 └── danh-sach-ban-va.txt   ← {{artifacts.ban-va.danh_sach}}
                                              3 đường dẫn TUYỆT ĐỐI, mỗi dòng một
```

#### Vì sao LOẠI hướng "trỏ theo chỉ số" (`{{artifacts.ten[0]}}` / `{{artifacts.ten.0}}`)

Nghe tự nhiên nhất, và sai ở đúng chỗ chết người: **không ai biết N lúc viết
flows.toml**. Nguồn của `foreach` gần như luôn là output của bước trước, nên độ
dài danh sách chỉ có lúc chạy.

Hậu quả cụ thể: danh sách ra 5 mục, người viết flow gõ `[0] [1] [2]`, và mục 4–5
biến mất. Không lệnh nào hỏng, không dòng nào đỏ, bản tóm tắt vẫn nói "xong". Đó
đúng là lớp **lỗ mất việc im lặng** mà cả mảnh idempotency đã đi đường vòng để
tránh — không có lý do gì mở lại nó ở cửa bên cạnh.

Và `Validate` **không cứu được**: muốn biết `[3]` có vượt biên không thì phải
biết N, mà N chưa tồn tại lúc kiểm. Một cú pháp mà bộ kiểm không soi nổi là một
cú pháp chỉ hỏng lúc chạy thật.

Biến thể "chỉ số động" — bước sau cũng `foreach`, dùng `{{index}}` của chính nó
để lấy file thứ `index` của bước trước — còn tệ hơn: nó **giả định** hai bước lặp
trên cùng một danh sách theo cùng một thứ tự. Không có gì trong engine kiểm được
giả định đó, và khi nó sai thì bước nhận một đường dẫn hợp lệ trỏ vào file của
**một mục khác**. Sai mà trông đúng, lại còn im.

#### Trả lời thẳng: bước `shell` nhận thế nào, argv không có vòng lặp?

**Đúng — argv không có vòng lặp, và không có cú pháp nào bịa ra được một cái.**
Nên thứ đi vào argv **không phải N đường dẫn**: nó là **một** đường dẫn, tới cái
file liệt kê N đường dẫn kia. Một phần tử argv, luôn luôn, bất kể N bằng 3 hay
50. Chương trình được gọi đọc file đó rồi tự lặp — **nó** có vòng lặp, argv thì
không.

Đây là lý do tôi **không** chọn "nhét N đường dẫn vào một chuỗi ngăn bởi xuống
dòng": chuỗi đó lọt vào argv thành một phần tử chứa ba dòng, và lệnh nhận một
"tên file" có ký tự xuống dòng ở giữa — hỏng bằng một thông báo chẳng liên quan,
ở một bước chẳng liên quan.

Bước `agent` dùng **đúng một cú pháp đó**: prompt nói *"đọc danh sách file trong
{{artifacts.ban-va.danh_sach}} rồi gộp lại"*. Một cú pháp cho mọi loại node,
không có ngoại lệ nào phải nhớ.

#### Tên TRẦN của một bước lặp là **LỖI**, không phải mặc định

Cám dỗ lớn nhất là để `{{artifacts.ban-va}}` tự đổi nghĩa thành "danh sách" khi
bước sản xuất có `foreach`. **Không.** Khắp hệ thống, tên trần nghĩa là MỘT file,
và `run = ["git","apply","{{artifacts.ban-va}}"]` là câu mẫu nằm ngay trong tài
liệu. Đổi nghĩa nó theo hình dạng của bước sản xuất là làm cho cùng một dòng
flows.toml có hai nghĩa tuỳ nơi khác viết gì.

Nên hình dạng phải **khớp**, và lệch thì dừng ở lúc lưu:

| bước sản xuất | placeholder | kết quả |
|---|---|---|
| thường | `{{artifacts.x}}` | ✔ đường dẫn tới file |
| thường | `{{artifacts.x.danh_sach}}` | ✗ LỖI — "bước `a` KHÔNG có `foreach`" |
| `foreach` | `{{artifacts.x}}` | ✗ LỖI — nêu đích danh `{{artifacts.x.danh_sach}}` |
| `foreach` | `{{artifacts.x.danh_sach}}` | ✔ đường dẫn tới bản kê |
| bất kỳ | `{{artifacts.x.danhsach}}` | ✗ LỖI — "hậu tố không có" |

#### Ba thứ chạy theo **từng lượt lặp**, không theo cả bước

- **Thư mục artifact.** Thư mục của cả bước bị dọn **một lần** trước khi phát các
  lượt (dọn trong từng lượt là `RemoveAll` trên thư mục cha đúng lúc lượt bên
  cạnh đang ghi). Mỗi lượt sau đó tự tạo `<index>/` của mình.
- **Hợp đồng đầu ra.** 49 lượt giao hàng và 1 lượt câm là một **mục bị mất**,
  không phải một bước "gần xong". Bước hỏng, và thông báo gọi tên đúng mục đó.
  Bản kê **chỉ** được sinh khi mọi lượt đã giao đủ.
- **`phai_co`.** Ở nhánh lặp, trường này trước đây **không được kiểm một lần
  nào** — nó nằm trong flows.toml, `validate` không kêu, và nó không làm gì cả.
  Cùng lớp với mọi cái cờ bị lờ đi trong im lặng. Nay kiểm từng lượt.

Tên thư mục lượt lặp là **đúng con số của `{{index}}`**, không đệm số 0. Đệm cho
đẹp thứ tự chữ cái (`01`…`10`) sẽ làm `{{index}}` và tên thư mục lệch nhau; thứ
tự đúng đã có ở bản kê rồi, đó mới là kênh có thứ tự.

### 1.2 `foreach` + `idempotent` — khoá trên TỪNG lượt lặp (sổ v11)

`#199` chặn tổ hợp này vì *"một khoá chung cho cả bước sẽ bỏ qua luôn những mục
MỚI trong danh sách"*. Có hai cách hiểu "một khoá chung", và **cả hai đều sai**:

| cách | hậu quả |
|---|---|
| khoá băm **cả danh sách** | thêm 1 mục mới → khoá đổi → **cả 30 mục chạy lại**. Không mất việc, nhưng 29 lần gọi agent bị đốt để xử lý 1 mục — đúng thứ idempotency sinh ra để tránh. |
| khoá băm **một mục** | bỏ qua cả bước theo một mục, tức bỏ qua luôn những mục MỚI. **Lỗ mất việc im lặng.** |

Nên mỗi lượt lặp phải có dòng sổ của riêng nó → **schema v11**, bảng
`flow_step_items(run_id, step_id, idx, idem_key, output)`.

Bảng này **không phải một cache độc lập** — v10 đã cố ý từ chối một cái như thế.
Nó là phần **chi tiết** của một dòng `flow_steps` đã có sẵn: cùng `run_id`, cùng
`step_id`, chỉ thêm chỉ số. Dòng nào biến mất khỏi lịch sử lượt chạy thì mục của
nó biến mất theo. Và **chỉ ghi khi bước bật `idempotent`** — bước không bật thì
bảng không có một dòng nào (có bài kiểm canh).

**Hai quyết định về nội dung khoá:**

- **CÓ băm `{{item}}`.** Hôm nay `cauHoi()` đã cuốn `{{item}}` vào cho mọi loại
  node, nhưng một trường mới quên nối vào đó sẽ làm **mọi** lượt lặp chung một
  khoá — 49 mục bị bỏ qua theo kết quả của mục đầu tiên. Thêm một chuỗi vào băm
  không tốn gì; bỏ sót thì không cứu được.
- **KHÔNG băm `{{index}}`.** Vị trí trong danh sách không phải danh tính của
  việc. Một mục đổi chỗ vì danh sách được sắp lại vẫn là đúng việc đó, và bắt nó
  chạy lại là làm cache vô dụng đúng lúc nó cần có ích nhất. (Prompt có gõ
  `{{index}}` thì `cauHoi` đã cuốn nó vào rồi — lúc đó vị trí **thật sự** là một
  phần của câu hỏi, và chạy lại mới đúng.)

Hệ quả: trúng cache thì artifact được chép từ **đúng chỉ số cũ** (`cu.Idx`), có
thể khác chỉ số bây giờ. Chép từ chỉ số hiện tại là chép nhầm file của một mục
khác, im lặng.

Giữ nguyên từ mảnh cũ: trúng cache mà artifact cũ đã bị dọn thì coi như **không
trúng** — thà chạy lại tốn tiền còn hơn báo xong rồi để bước sau mở file rỗng.

Và một chi tiết nhỏ nhưng quan trọng: kết quả của lượt trúng cache đi vào khối
gộp **y nguyên byte**, không thêm chữ "bỏ qua". Khối gộp là thứ đi sang bước sau,
nên một dòng chú thích thêm vào sẽ làm khoá của **bước sau** đổi — tức trúng cache
ở bước này lại bắt bước sau chạy lại. Việc bỏ qua được nói ở cột trạng thái
(`xong 3 mục (bỏ qua 2 mục — idempotent)`) và ở nhật ký.

### 1.3 Chỗ vẫn CHẶN, và câu nói rõ vì sao

`flow.Validate` vẫn chặn (và `runForEach` chặn lần nữa, vì `Flow` còn dựng được
thẳng bằng mã Go):

| tổ hợp | câu chặn |
|---|---|
| `foreach` + `merge` | "mỗi lượt lặp sẽ gộp lại đúng cùng một tập nguồn" |
| `foreach` + `route` | "mỗi lượt lặp sẽ chọn lại đúng cùng một tập đường" |
| `foreach` + `approve` | "mỗi lượt sẽ là một lần chờ người duyệt" |
| `artifact` ở `model`/`merge`/`route`/`approve`/`notify` | "không có đường nào ghi ra file" |
| `idempotent` ở `approve`/`notify` | "không còn là rào" / "im lặng đúng lúc cần lên tiếng" |

Hai cái đầu **không phải "chưa làm"** — lặp chúng là chạy đúng một việc N lần rồi
vứt N−1 kết quả.

Giữ nguyên **cảnh báo** (không chặn) cho `idempotent` ở `shell`/`test`/`lint`: bộ
chạy không nhìn thấy cây mã trên đĩa. Cảnh báo này hiện ra thật trong lượt đo bên
dưới.

### 1.4 Node `model` — phần suy luận đi đâu (sổ v12)

**Câu hỏi:** phần suy luận có nên đi vào `{{steps.x.output}}` không?

**Trả lời: KHÔNG, và không có placeholder nào trỏ tới nó cả.** Phần nghĩ là thứ
để **người đọc**, không phải để bước sau ăn.

`output` không phải một ô hiển thị — nó là **đường truyền** sang bước sau, và
trộn phần nghĩ vào đó làm hỏng ba thứ:

1. **Hợp đồng đầu ra.** `phai_co = ["NÊN TRỘN"]` sẽ đạt khi model chỉ **nghĩ**
   tới chuỗi đó mà không **nói** ra. Một cổng kiểm gật đầu vì đọc được ý nghĩ —
   đúng lớp hỏng mà `phai_co` sinh ra để chặn (lượt #46).
2. **Trần nhét.** Đo thật deepseek-v4-flash 22/08: câu trả lời **91** ký tự, phần
   nghĩ **477** — gấp 5,2 lần. Nhét cả hai vào một đường 6.000 ký tự là lấy chỗ
   của câu trả lời để chứa bản nháp.
3. **Khoá idempotency.** Output của bước này là đầu vào của bước sau, nên nó nằm
   trong khoá của bước sau. Phần nghĩ **không ổn định** giữa hai lượt gọi, nên
   trộn vào là làm khoá bước sau đổi mỗi lượt và cache không bao giờ trúng.

Nên: **cột riêng** (`flow_steps.suy_luan`, schema v12), lưu ở cả nhánh thành công
lẫn nhánh hỏng (lúc hỏng mới là lúc cần đọc nó nhất), và in ra CLI dưới một cái
nhãn nói rõ nó **không** đi sang bước sau:

```
   ✓ hoi  completed
      │ 1 + 1 = 2
      └ phần suy luận (KHÔNG đi sang bước sau):
        · Người dùng hỏi một phép cộng đơn giản…
```

**Cùng chỗ, tìm ra một lỗi thứ hai chưa ai báo:** `cauHoi()` thiếu hẳn nhánh
`TypeModel`. Hai thứ hỏng cùng lúc và cả hai đều im — (a) sổ **không lưu câu hỏi
nào** cho bước `model`, đọc lại lượt chạy thì thấy câu trả lời mà không thấy đã
hỏi gì; (b) `KhoaIdem` băm `cauHoi`, nên khoá của bước `model` **không cuốn
prompt vào** — hai bước `model` cùng route mà khác hẳn câu hỏi ra **cùng một
khoá**, và lượt sau trả lời câu này bằng câu trả lời của câu kia. Đã vá, có bài
kiểm riêng.

### 1.5 Bảng liệt kê artifact không được nói sai

`flow.TenTheoDuong` chỉ tra được đường dẫn của bước **thường**. Không sửa gì thì
`sagent flow artifacts` in `(không bước nào khai)` cho **mọi** file của một bước
lặp — đó không phải một ô trống vô hại, nó là một **lời khẳng định, và nó sai**.
Thêm `flow.BanDoTenArtifact` và nối vào `api.FlowArtifacts` (một chỗ, ba mặt CLI
/ HTTP / web dùng chung). Kết quả trên lượt chạy thật ở mục 1.6.

### 1.6 SỐ ĐO THẬT — `sagent flow run`, 0 token, 0 đồng

Theo khuôn `docs/DO-LUONG.md` mục *"22/08 — Ba mảnh flow chạy thật qua `sagent
flow run`"*: dự án tạm, flow chỉ gồm bước **`shell`**, đi qua đúng bộ chạy / đúng
sổ / đúng đường artifact như bước `agent`, chỉ khác là không gọi model nào.

**Khác một điểm so với khuôn, và cố ý:** chạy trong **hồ sơ sandbox**
(`USERPROFILE=C:\Users\Administrator\do-foreach\sandbox`) chứ không dùng
`~/.ai-accounts` thật. Lý do ở mục 2. Nên số lượt chạy là **#1–#8 của sổ
sandbox**, không nối tiếp dãy #53–#66 của sổ chính.

**A. `foreach` + `artifact` (lượt #1)** — `ds = "alpha\nbeta\ngamma"`

`sagent flow artifacts 1` in ra:

```
   BƯỚC           TÊN                              BYTE  ĐƯỜNG DẪN
   sinh           phan (mục 1)                       17  sinh/1/phan.txt
   sinh           phan (mục 2)                       16  sinh/2/phan.txt
   sinh           phan (mục 3)                       17  sinh/3/phan.txt
   sinh           phan (bản kê)                     261  sinh/danh-sach-phan.txt
```

Ba file **riêng biệt** — không lượt nào đè lượt nào. Bản kê chứa đúng 3 đường dẫn
tuyệt đối. Và bước `gop` — một bước `shell` nhận **đúng một** phần tử argv
(`{{artifacts.phan.danh_sach}}`) — để lại output:

```
BAN-KE=…\artifacts\run-1\sinh\danh-sach-phan.txt
SO-FILE=3
  <MUC=alpha CHISO=1>
  <MUC=beta CHISO=2>
  <MUC=gamma CHISO=3>
```

Đó là câu *"argv không có vòng lặp"* được trả lời bằng một lượt chạy thật.

**B. `foreach` + `idempotent` (lượt #2 → #6)** — đếm bằng một file cộng dồn, mỗi
lần bước chạy **thật** thì thêm một dòng.

| lượt | danh sách | engine nói | file đếm |
|---|---|---|---|
| #2 | `alpha, beta` | xong 2 mục | 2 dòng |
| #3 | `alpha, beta` (y nguyên) | `xong 2 mục (bỏ qua 2 mục — idempotent)` | **vẫn 2 dòng** |
| #4 | `alpha, beta, gamma` (**thêm 1**) | `xong 3 mục (bỏ qua 2 mục)` | **3 dòng** |
| #6 | `gamma, beta, alpha` (**đảo**) | `xong 3 mục (bỏ qua 3 mục)` | **vẫn 3 dòng** |

Lượt **#4** là con số đáng giá nhất của cả mảnh: thêm một mục mới thì **đúng một
mục** chạy, hai mục cũ không tốn gì. Với khoá băm-cả-danh-sách thì cả ba đã phải
chạy lại.

Lượt **#6** chứng minh khoá bám **mục** chứ không bám **vị trí**, và nhật ký nói
thẳng ra điều đó:

```
· do-idem.xu-ly mục 1 (gamma): bỏ qua — việc này lượt chạy #5 (mục 3) đã làm xong
· do-idem.xu-ly mục 3 (alpha): bỏ qua — việc này lượt chạy #5 (mục 1) đã làm xong
```

Đọc sổ `flow_step_items` thì thấy đúng cùng một tập khoá, chỉ đổi vị trí:

```
(5, 'sinh', 1, '2d5c070573')      (6, 'sinh', 1, 'ce764257ad')
(5, 'sinh', 2, 'fc587ec439')      (6, 'sinh', 2, 'fc587ec439')
(5, 'sinh', 3, 'ce764257ad')      (6, 'sinh', 3, '2d5c070573')
```

**C. Cả ba cùng lúc — `foreach` + `artifact` + `idempotent` (lượt #7 → #8)**

Lượt #7 chạy `alpha, beta, gamma`; lượt #8 chạy `gamma, alpha, beta`, **cả ba mục
trúng cache**. Bước `gop` của lượt #8 đọc bản kê mới và in ra:

```
FILE-CUA=gamma
FILE-CUA=alpha
FILE-CUA=beta
```

Đúng thứ tự **mới**. Nếu artifact được chép theo chỉ số **hiện tại** thay vì chỉ
số cũ thì dòng này đã là `alpha / beta / gamma`. Đây là bằng chứng cho quyết định
"chép từ `cu.Idx`", đo trên một lượt chạy thật chứ không phải trong test.

**D. Cảnh báo hiện đúng lúc.** `sagent flow validate` in cảnh báo `idempotent` cho
bước `shell` ngay từ đầu, không chặn:

```
! do-idem.xu-ly  bộ chạy KHÔNG nhìn thấy cây mã trên đĩa, HEAD của git hay đồng hồ…
```

### 1.7 11 lần đột biến — gỡ phần sửa ra thì bài kiểm có đỏ không

Chạy thật từng lần: gỡ một phần sửa, chạy đúng bài kiểm canh nó, rồi khôi phục.

| # | gỡ cái gì | bài kiểm | kết quả |
|---|---|---|---|
| 1 | `env[KhoaArtifactDir] = lapDir` (thư mục riêng từng lượt) | `…MoiLuotMotThuMucVaMotBanKe` | ĐỎ |
| 2 | lời gọi `GhiDanhSachArtifact` | như trên | ĐỎ |
| 3 | `ThieuArtifactLap` (hợp đồng từng lượt) | `…MotLuotCamThiCaBuocHong` | ĐỎ |
| 4 | `ThieuPhaiCo` ở nhánh lặp | `TestForEachKiemPhaiCoTungLuot` | ĐỎ |
| 5 | khối `thuDungLaiMucCu` | `…IdempotentChiChayMucMOI` | ĐỎ |
| 6 | `cu.Idx` → `chiSo` khi chép artifact | `…ChepArtifactTheoDungChiSoCu` | ĐỎ |
| 7 | `ghi("foreach.item", …)` | `…KhongTronKetQuaGiuaCacMuc` | ĐỎ |
| 8 | **thêm** `ghi("foreach.index", …)` | `…DoiChoKhongMatCache` | ĐỎ |
| 9 | bỏ báo lỗi khi artifact cũ đã mất | `…MatArtifactCuThiChayLai` | ĐỎ |
| 10 | nhánh `TypeModel` trong `cauHoi` | `…BuocModelCuonPromptVao` | ĐỎ |
| 11 | trả `conSotArtifact` về regex cũ | `…BatCaDangCoHauTo` | ĐỎ |

Đột biến **#8 ngược chiều** với các đột biến khác: nó **thêm** một thứ nghe rất
hợp lý (băm cả vị trí) và bài kiểm đỏ. Đó là bài kiểm giữ cho quyết định
"không băm `{{index}}`" khỏi bị ai đó vô tình đảo lại.

Lần chạy đầu, đột biến **#9 VẪN XANH** — không phải vì bài kiểm yếu, mà vì chuỗi
tôi dùng để đột biến trùng với hàm `chepArtifactCu` của bước **thường** (nằm
trước trong file), nên phép thay trúng nhầm hàm. Nhắm lại đúng `chepArtifactMucCu`
thì đỏ ngay, kèm đúng câu: *"artifact cũ đã mất thì KHÔNG được coi là trúng cache:
`xong 2 mục (bỏ qua 2 mục — idempotent)`"*.

---

## 2. Sự cố

### 2.1 Hai bài kiểm CHỚP TẮT có sẵn — đo được, và đã vá nguyên nhân

`go test ./...` **không** ra 0 một cách đáng tin lúc tôi bắt đầu. Hai bài, và tôi
đã kiểm chúng **trên cây sạch** (`git stash -u`, commit `04f7ebc`, trước mọi thay
đổi của lượt này) để chắc chắn không phải do tôi:

| bài | tần suất | nguyên nhân |
|---|---|---|
| `TestNoiRaKhiDangChoTran` | **3/8** lần đỏ trên cây sạch | độ trễ agent giả 60ms không đủ để hai bước còn đang giữ chỗ lúc hai bước kia đi xin → **không ai phải chờ**, nên không có dòng nào, nên bài đỏ vì một lý do chẳng liên quan gì tới thứ nó canh |
| `TestMergeHaiLuotXongNguocThuTuVanRaGiongNhau` | ~1/5 lượt `go test ./...` | chênh lệch 120ms nhỏ hơn độ giật của bộ lập lịch trên máy đang tải → hai lượt xong **cùng** thứ tự, và bài tự phát hiện ("cần tăng độ trễ") rồi tự đỏ |

Cả hai đều là **điều kiện tiên quyết của cảnh dựng** không xảy ra, chứ không phải
khẳng định sai. Đã vặn núm cho đủ rộng (60ms → 400ms; 0/60/120 → 0/200/400) và
**không nới một khẳng định nào**. Sau đó: `TestNoiRaKhiDangChoTran` 20/20 xanh,
`TestMerge…` 15/15 xanh, và `go test ./...` **6 lượt liên tiếp** mã thoát 0.

Hai bài này thuộc mảnh "trần đồng thời" và "merge" của các lượt trước; tôi đụng
vào vì chúng nằm trong vùng file được giao và vì một cổng nghiệm thu chớp tắt thì
không nghiệm thu được gì.

### 2.2 SỔ LÊN v10 → v12 — cái bẫy đã biết, và cách tôi né hôm nay

Ghi vào đây vì nó **sẽ** cắn ai đó: `store.migrate` **từ chối mở** một `state.db`
có `schema_version` cao hơn số migration mà binary biết ("thà không chạy"). Đó là
đúng thiết kế. Hệ quả hôm nay:

> Một binary `sagent` build từ nhánh này mở `~/.ai-accounts/state.db` là **nâng
> nó lên v12**. Sau đó **mọi binary v10 đang chạy** — dashboard của agent chạy
> song song, chẳng hạn — sẽ **từ chối mở sổ**.

Vì có agent khác đang làm việc trên cùng máy này, tôi **không** chạy phép đo trên
`~/.ai-accounts` thật. Cả 8 lượt chạy ở mục 1.6 dùng
`USERPROFILE=C:\Users\Administrator\do-foreach\sandbox`, có `state.db` riêng. Sổ
chính vẫn đang ở phiên bản cũ, chưa bị đụng.

**Việc phải làm khi trộn nhánh:** dừng dash và mọi tiến trình `sagent` đang chạy
**trước**, rồi mới build lại. Đây đúng là lớp lỗi mà ghi chú "state.db v10 vs mã
v9" đã cảnh báo là **sẽ tái phát**; lần này nó tái phát vì một lý do chính đáng
(hai bảng/cột thật sự cần), nhưng cái giá vẫn phải trả và phải trả có chuẩn bị.

### 2.3 Bẫy công cụ — heredoc nuốt dấu gạch chéo ngược

Ba lần trong lượt này, viết Go bằng `python - <<'PY'` cho ra mã hỏng: `\\n` trong
chuỗi Python đến nơi thành **một ký tự xuống dòng thật** chứ không phải hai ký tự
`\` + `n`, làm vỡ chuỗi Go ("newline in string"), và một lần `\\x00` thành một
byte NUL thật trong mã nguồn. Heredoc dùng nháy (`<<'PY'`) mà vẫn bị. Cách vòng
qua: dựng dấu gạch chéo ngược bằng `chr(92)`, hoặc viết nội dung ra file bằng
công cụ ghi file rồi mới ghép.

Thêm một lần dẫm lại đúng cái bẫy đã ghi trong ghi chú cá nhân: `print` tiếng
Việt trong Python trên máy này chết vì cp1252 — và nó chết **sau** khi đã ghi
file, nên nhìn traceback thì tưởng lệnh thất bại trong khi nó đã làm xong việc.

### 2.4 Một phép đo tự làm hỏng mình

Flow đo mục 1.6-C ban đầu cho mỗi lượt lặp `Add-Content` vào **cùng một** file
đếm. Ba lượt chạy song song → tranh file → PowerShell báo *"The process cannot
access the file … used by another process"* và mất một dòng đếm. Không phải lỗi
engine (artifact của từng lượt vẫn đúng, và bằng chứng thật nằm ở output bước
`gop`), nhưng nó nhắc rằng **một phép đo về song song phải tự nó không được có
tài nguyên chung**.

### 2.5 Điều tôi ít chắc chắn nhất

Quyết định **"không có placeholder nào trỏ tới phần suy luận"** (mục 1.4). Có lập
luận cho hướng ngược lại: một bước "soi lại lập luận của người soi" thì phần nghĩ
đúng là dữ liệu nó cần. Tôi không mở vì một kênh đọc mới phải đi qua `doc_duoc`
cho đủ — thêm nửa vời một đường đọc mà quên hàng rào quyền là đúng cái lỗ mà
mảnh artifact đã suýt để lại. Nếu sau này mở thì phải là `{{steps.x.suy_luan}}`
đi qua `LocDocDuoc`, chứ không phải nối vào `output`.

---

## 3. Bước tiếp theo

Theo thứ tự tôi nghĩ là đáng làm nhất:

1. **Mặt dashboard chưa thấy phần suy luận của bước flow.** Đây là chỗ tôi **cố ý
   dừng** vì ngoài vùng file được giao. Hai dòng cụ thể cần cắm, người điều phối
   cầm nguyên:
   - `internal/dash/flow_api.go:220` — trong `buocDTO`, ngay dưới
     `Output string \`json:"output"\``, thêm
     `SuyLuan string \`json:"suyLuan,omitempty"\``;
   - `internal/dash/flow_api.go:244` — trong `themR`, ngay cạnh `Output: st.Output,`
     thêm `SuyLuan: st.SuyLuan,`;
   - rồi một chỗ trong `internal/dash/web/` **thật sự hiện nó ra**, tách khỏi khối
     output và có nhãn "không đi sang bước sau" — nếu chỉ thêm hai dòng trên thì
     đây đúng là kiểu "có ở mọi tầng trừ tầng cuối" mà hôm nay đã dính năm lần.

   Dữ liệu đã có sẵn ở `store.StepRun.SuyLuan`, không phải sửa gì thêm phía sổ.
   **Bảng artifact thì KHÔNG cần đụng** — nó lấy tên từ `api.FlowArtifacts`, và
   chỗ đó đã sửa rồi (mục 1.5), mặt web hưởng theo.

2. **`sagent flow show` và chạy khan chưa nói gì về `foreach` + `artifact`.** Xem
   trước một flow thì không thấy bước lặp sẽ để lại N file, cũng không thấy bước
   sau đọc bằng bản kê. `chaykho.go` có in `Lap` / `Artifact` / `Idempotent` rời
   nhau, nhưng không nói ra **quan hệ** giữa chúng. Đây là món nợ số 3 của `#199`,
   vẫn còn.

3. **Chưa đo với bước `agent` thật.** Cả 8 lượt ở mục 1.6 là bước `shell`. Con số
   đáng giá nhất còn thiếu vẫn đúng như `#199` nói: một bước gộp báo cáo tốn bao
   nhiêu token vào **trước/sau** khi chuyển sang artifact — và nay có thêm một
   câu hỏi mới, **rẻ hơn nhiều để đo**: một bước `foreach` 30 mục chạy lần thứ hai
   với 1 mục mới thì tiết kiệm được bao nhiêu.

4. **`MaxForEachItems = 50` giờ có nghĩa nặng hơn trước.** Một bước lặp 50 mục có
   `artifact` để lại 50 thư mục con mỗi lượt chạy, và `DonArtifact` chỉ dọn theo
   tuổi 7 ngày. Chưa đo bao nhiêu đĩa; nên đo trước khi ai đó dùng thật.

5. **Bản kê không có cách nào nói "mục này bị bỏ qua vì `when`".** Hôm nay
   `foreach` không có `when` theo từng mục nên chưa thành vấn đề, nhưng nếu thêm
   thì bản kê phải trả lời được "N dòng này là đủ hay đã lọc".

6. **Hai bài kiểm ở mục 2.1 chỉ được vặn núm, chưa được làm thành xác định.** Cách
   đúng hơn là để bài kiểm **ép** thứ tự bằng một tín hiệu (channel) thay vì bằng
   đồng hồ. Tôi không làm vì nó đụng vào thiết kế bài kiểm của người khác nhiều
   hơn mức một lượt sửa chớp tắt nên đụng.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Đọc `#199` mục "Bước tiếp theo" + engine flow (`foreach.go`, `artifact.go`, `idempotent.go`, `step.go`, `store.go`) | Opus 5 (1M) | thấp — đọc, xác nhận hiện trạng |
| **CHỌN cú pháp trỏ tới artifact của bước lặp, và loại hai hướng kia** | Opus 5 (1M) | **cao** — đây là toàn bộ việc 1a; mã đến sau và ngắn hơn nhiều |
| Trả lời câu "argv không có vòng lặp thì `shell` nhận thế nào" | Opus 5 (1M) | **cao** — chính câu này loại hướng "chuỗi nhiều dòng" |
| Cài đặt `foreach_artifact.go` + cắm vào `runForEach` | Opus 5 (1M) | trung bình |
| Cân ba cách định nghĩa khoá cho bước lặp, và hậu quả từng cách | Opus 5 (1M) | **cao** — riêng quyết định "băm `item`, KHÔNG băm `index`" là chỗ đắt nhất |
| Sổ v11 (`flow_step_items`) + chép artifact theo đúng chỉ số cũ | Opus 5 (1M) | trung bình |
| Bịt lỗ `phai_co` không được kiểm ở nhánh lặp | Opus 5 (1M) | thấp — thấy lúc viết lại `runForEach` |
| Trả lời "phần suy luận có vào `output` không" + sổ v12 + mặt CLI | Opus 5 (1M) | trung bình — câu trả lời rõ ngay, phần cài đặt máy móc |
| Tìm ra `cauHoi` thiếu nhánh `TypeModel` (hai lỗi im lặng) | Opus 5 (1M) | trung bình — lộ ra khi đi soi xem khoá idempotency cuốn được những gì |
| 11 lần đột biến, chạy thật từng lần rồi khôi phục | Opus 5 (1M) | thấp — máy móc, nhưng không bỏ được |
| Truy hai bài kiểm chớp tắt, chứng minh chúng có sẵn trên cây sạch | Opus 5 (1M) | trung bình — tốn nhiều lượt chạy, ít suy nghĩ |
| 8 lượt `sagent flow run` thật trong hồ sơ sandbox | Opus 5 (1M) | trung bình |
| Viết báo cáo này | Opus 5 (1M) | trung bình |

Cả lượt chạy trên một model duy nhất: **Opus 5 (1M context)**, effort mặc định
của phiên, **không gọi subagent, không dùng workflow**.

---

## 5. Nhận xét tự do

**Cái đắt nhất của lượt này là một câu hỏi, không phải một dòng mã.** `#199` dừng
lại đúng chỗ nên dừng — nó không thiếu thời gian để viết `os.MkdirAll`, nó thiếu
một câu trả lời cho "bước sau trỏ tới N file bằng cú pháp gì". Và câu trả lời hoá
ra được quyết định bởi một ràng buộc mà thoạt nhìn tưởng là chuyện phụ: **`run`
là argv**. Chính nó loại sạch hướng "chuỗi nhiều dòng" và đẩy tới file bản kê.
Nếu bước `shell` nhận chuỗi shell thì tôi đã chọn khác, và đã chọn sai.

**"Không ai biết N lúc viết flows.toml" là loại lập luận đáng tin nhất ở đây.**
Nó không dựa vào thẩm mỹ hay thói quen — nó chỉ ra rằng bộ kiểm **không thể** soi
cú pháp chỉ số, dù có muốn. Một cú pháp mà `Validate` bất lực là một cú pháp chỉ
hỏng lúc chạy thật, trên máy người khác, với danh sách dài hơn danh sách người
viết nghĩ tới. Tôi thấy đây là tiêu chí nên dùng lại: **trước khi thêm một cú
pháp, hỏi bộ kiểm có soi nổi nó không.**

**Quyết định "không băm `{{index}}`" là chỗ tôi tự tin nhất, và cũng là chỗ dễ bị
đảo lại nhất.** Nó ngược trực giác: `index` có trong env, băm nó vào thì "an toàn
hơn", và người sửa sau rất dễ thêm vào với ý tốt. Đó là lý do tôi viết hẳn một
đột biến **thêm** dòng đó vào và một bài kiểm đỏ khi thấy nó (đột biến #8). Bài
kiểm giữ được cả một quyết định thiết kế, không chỉ một hành vi — và tôi nghĩ mọi
quyết định kiểu "cố ý KHÔNG làm X" đều nên có một bài như thế, nếu không thì nó
chỉ là một câu bình luận mà lượt sau sẽ đọc lướt qua.

**Hai lỗ tôi tìm thấy đều không nằm trong yêu cầu, và cả hai đều cùng một hình
dạng.** `phai_co` không được kiểm ở nhánh lặp, và `cauHoi` không có nhánh
`TypeModel`. Cả hai là **một trường có mặt, được khai, được validate, và không ai
đọc tới** — đúng cái hình dạng mà `docs/DO-LUONG.md` đã ghi năm lần trong ngày
22/08 (`route.kiem`, nút Duyệt/Từ chối, `plugin.list`, `sagent help`, bảng năng
lực `reasoning`). Chúng lộ ra không phải vì tôi đi soi, mà vì hai nhánh mã bị đặt
cạnh nhau: viết lại `runForEach` thì thấy `runStep` kiểm hai hợp đồng còn nó kiểm
không cái nào; đi hỏi "khoá idempotency cuốn được những gì" thì thấy `cauHoi` trả
về chuỗi rỗng cho một loại node. **Cách tin cậy nhất để tìm lớp lỗi này có vẻ là
đặt hai đường mã làm-cùng-một-việc cạnh nhau rồi so từng dòng**, chứ không phải
đọc từng đường một cho kỹ.

**Về luật ngang quyền bốn mặt:** lượt này **không đẻ ra action mới** nên không
phải làm bốn mặt, và tôi nói thẳng lý do thay vì im lặng bỏ qua. `foreach` +
`artifact` / `idempotent` là **hành vi bên trong** của `flow.run` và `flow.validate`
— hai action đã tồn tại, đã có đủ CLI/HTTP/web. Nhưng luật bốn mặt còn một vế mà
`#199` đã đề nghị thêm và tôi thấy đúng ở đây: *mặt web không những phải gọi tới,
mà còn phải không được nói sai về thứ nó không hiểu.* Bảng artifact là đúng ca đó
— nó **có** gọi tới, nó **có** hiện ra, và nếu tôi không sửa `TenTheoDuong` thì nó
sẽ in "(không bước nào khai)" cho những file mà bước **có** khai. Một mặt nói sai
còn khó phát hiện hơn một mặt vắng mặt, vì mặt vắng mặt thì ai cũng thấy trống.

**Chỗ tôi thấy dự án đang trả giá đều đặn:** mỗi mảnh mới đều đúng, và mỗi mảnh
mới đều thêm một cặp trường có thể đi cùng nhau hoặc không. `foreach` × `artifact`
× `idempotent` × `merge` × `route` × `phai_co` × `doc_duoc` × `on_failure` — số tổ
hợp lớn hơn số bài kiểm rất nhanh, và `Validate` đang là thứ duy nhất giữ cho
người dùng không rơi vào một ô chưa ai nghĩ tới. Tôi nghĩ đáng có một bài kiểm
**sinh tổ hợp**: dựng mọi cặp (loại node × cờ) và khẳng định mỗi cặp hoặc **chạy
được** hoặc **bị chặn bằng một câu có lý do** — không ô nào được rơi vào khoảng
giữa "không chặn mà cũng không làm gì". Hôm nay `phai_co` × `foreach` đã nằm đúng
trong khoảng giữa đó suốt một thời gian, và không có gì báo.
