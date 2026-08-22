# `fallback` chạy thật, và xem được artifact từ dashboard

Nhánh: `sagent/fallback-artifact-22-08` · Ngày 22/08/2026
Vùng đụng tới: `internal/flow/*`, `internal/api/*`, `internal/dash/*` (trừ `web/docs/`), `cmd/sagent/*`

Nghiệm thu (chạy **từng lệnh riêng**, `-count=1`): `go build ./...` ✔ · `go vet ./...` ✔ · `go test ./...` ✔

---

## 1. Đã làm

### 1.1 Việc 1 — `on_failure = "fallback"` gần như không làm gì

#### Kiểm lại lời của #199 TRƯỚC khi sửa

Mục 2.3 của `docs/BAO-CAO-FLOW-ARTIFACT.md` nói `fallback` "gần như không làm
gì". Tôi **không sửa theo lời**, mà đo lại bằng đúng khuôn của mục "22/08 — Ba
mảnh flow chạy thật" (`docs/DO-LUONG.md`): một dự án tạm ở
`C:\Users\Administrator\do-fallback-that`, flow **chỉ gồm bước `shell`**, chạy
qua `sagent flow run`. **Tám lượt chạy thật, 0 token, 0 đồng.**

**Kết quả: #199 nói ĐÚNG, và nói THIẾU một hàng.** Ba lượt chạy trước khi sửa:

| # | cách khai bước thay thế | bước chính | bước thay thế | lượt chạy |
|---|---|---|---|---|
| **#59** | không `needs` (gốc DAG) | **hỏng** | **CHẠY — nhưng SONG SONG, ở đợt đầu, TRƯỚC khi bước chính kịp hỏng** | `completed` |
| **#60** | `needs = ["chinh"]` | **hỏng** | **KHÔNG chạy** | `completed` |
| **#61** | không `needs` (gốc DAG) | **XONG** | **CHẠY — dù chẳng ai hỏng** | `completed` |

Ba hàng, không hàng nào là thứ người ta gõ `on_failure = "fallback"` để có:

- **#60 là hàng #199 mô tả**: bước được trỏ tới không bao giờ chạy, và lượt chạy
  vẫn được ghi là `completed` trong khi câu log hứa chắc nịch *"bước thay sẽ chạy
  thay"*. Một lời hứa sai, im lặng.
- **#59 là hàng #199 nói tới nhưng chưa đo**: bước thay thế chạy *đúng bước* mà
  **sai lúc**. Nó không có `needs` nên nằm cùng đợt với bước chính; hai bước chạy
  song song và bước thay thế xong trước khi bước chính kịp hỏng. Một hàng dự
  phòng không thể phản ứng với sự cố chưa xảy ra.
- **#61 là hàng #199 KHÔNG nhắc tới, và nó nặng hơn cả hai hàng trên.** Đây là
  bản sao đúng cái bẫy mà `compensate` đã tránh: bước thay thế là một **GỐC của
  DAG** nên nó chạy ở **mọi** lượt, kể cả lượt suôn sẻ. Với hàng dự phòng thật
  (gọi nhà cung cấp API thứ hai, dựng lại máy) thì đó là tiêu tiền cho việc
  không ai cần — đúng thứ `fallback` sinh ra để tiết kiệm.

Nói cách khác: tác dụng thật của `fallback` hôm qua chỉ là *"đừng dừng lượt
chạy"*, còn bước được trỏ tới chạy hay không phụ thuộc vào việc nó **tình cờ** là
node độc lập hay không — và nếu tình cờ đúng thì nó lại chạy cả khi không cần.

#### Sửa theo khuôn `compensate`

`internal/flow/fallback.go` là bản song song của `compensate.go`, khác **đúng một
chỗ về ngữ nghĩa**:

```
  compensate   chạy bước GỠ LẠI,    rồi DỪNG.      Việc chính coi như KHÔNG làm.
  fallback     chạy bước THAY THẾ,  rồi ĐI TIẾP    nếu nó xong.
```

Mọi thứ còn lại **cố ý giống hệt**: bước thay thế bị loại khỏi lịch chạy thường,
được ghi `skipped` kèm lý do, chạy qua đúng `runStep`, biết mình đang thay cho ai
qua `{{buoc_hong}}` (**dùng chung một biến** với bước gỡ lại — cùng câu hỏi thì
cùng một tên), và không có đường nào để thay-thế-của-thay-thế vì `runStep` không
xét `on_failure`.

```
   đợt 1        đợt 2                       ngoài lịch, ngay lúc sự cố
  ┌──────┐    ┌────────┐  hỏng   ┌──────────────────────────────┐
  │  a   │───►│ hoi-1  │────────►│  hoi-2   (bước CHẠY THAY)    │
  └──────┘    └────┬───┘         └───────────┬──────────────────┘
                   │                    ┌────┴─────┐
              (không chạy)          thay được   thay KHÔNG được
                   ▼                    │            │
              ┌────────┐                ▼            ▼
              │ buoc-c │◄──────── ĐI TIẾP        DỪNG, "cả hai
              └────────┘          (kết quả của    đường đều tắc"
                                   hoi-2 nằm ở
                                   chỗ của hoi-1)

  hoi-2 ở lượt chạy BÌNH THƯỜNG (không có gì hỏng):
      ghi `skipped` + "bước chạy thay — chỉ chạy khi hoi-1 hỏng"
```

**Quyết định đáng cãi nhất, nên nói thẳng: kết quả của bước thay thế đọc được
bằng TÊN BƯỚC HỎNG.**

```toml
[[flow.x.step]] id = "hoi-claude"  on_failure = "fallback"  fallback = "hoi-grok"
[[flow.x.step]] id = "gop"  needs = ["hoi-claude"]  prompt = "{{steps.hoi-claude.output}}"
```

Bước `gop` khai `needs` tới `hoi-claude` và đọc kết quả của `hoi-claude`. Không
gán thì nó nhận một ô rỗng — với bước `shell` còn là lỗi cứng (`BuocConSot`). Mà
người viết flow **không có cách nào viết cho đúng cả hai đường**:
`{{steps.hoi-grok.output}}` rỗng ở mọi lượt suôn sẻ, vì lượt suôn sẻ thì
`hoi-grok` bị `skipped`. Tức là: không gán thì `fallback` không dùng được vào
việc gì. Chọn gán.

**Sổ thì vẫn ghi sự thật**: `hoi-claude` là `failed`, `hoi-grok` là `done`.
Không dòng nào trong sổ nói `hoi-claude` xong cả — chỗ gán nằm ở bảng biến của
lượt chạy, không nằm ở sổ. Và `runState.datOutput` có riêng một hàm (thay vì đi
qua `set`) đúng để không ai lỡ tay truyền `done` vào đó.

**Chạy lại giữa chừng**: một lượt có thể dừng ở rào duyệt SAU khi bước thay thế
đã xong; `Resume` dựng lại từ SQLite và bảng biến trong bộ nhớ thì mất. Không
thêm cột, không nhét dấu vết vào câu chữ để rồi tách chuỗi: `ganKetQuaThayThe`
suy ngược từ hai dòng sổ đã có ("A `failed` với on_failure = fallback", "bước
thay thế của A `done`") và được gọi ở **cả hai** chỗ — lúc chạy thay xong, và
lúc nạp lại sổ — nên hai đường không lệch nhau được.

#### Câu cảnh báo cũ đã bị xoá

Câu `"%s.%s hỏng — bước %s sẽ chạy thay"` nói sai sự thật nên **không sửa chữ, mà
thay bằng hành vi thật**. Giờ nó là `"... hỏng — chạy bước %s thay"` và in ra
NGAY TRƯỚC khi bước đó thật sự chạy. Cả hai đường tắc thì thông điệp khác hẳn:

```
✗ fb-thay-hong.chinh: exit status 1
⚠ fb-thay-hong.chinh hỏng — chạy bước du-phong thay
✗ fb-thay-hong.du-phong: exit status 3
· dừng ở bước chinh: exit status 1 — VÀ BƯỚC CHẠY THAY "du-phong" CŨNG HỎNG
  (exit status 3). Không còn đường nào khác cho bước này: dừng.
```

#### Bốn mặt của việc 1

`fallback` **không phải action mới** (nó là giá trị đã có của một trường đã có),
nhưng ba mặt hiển thị đều đang nói SAI sau khi hành vi đổi, nên phải sửa cùng lúc:

| mặt | trước | sau |
|---|---|---|
| `flow.Validate` | chỉ bắt "thiếu `fallback`" / "trỏ tới bước không tồn tại" | thêm `VanDeFallback`: tự-thay-chính-mình, `approve` làm hàng dự phòng, `needs` trỏ vào bước chạy thay (bước treo vĩnh viễn), `fallback` khai mà không dùng, `on_failure`/`needs` của bước chạy thay bị bỏ qua |
| `sagent flow show` | hiện bước chạy thay y hệt một bước sắp chạy | in `bước chạy thay — chỉ chạy khi X hỏng` + `hỏng thì: chạy bước "Y" THAY rồi ĐI TIẾP — kết quả của nó đọc bằng tên bước này` |
| `flow.kho` (chạy khan) | đếm nó vào tổng số agent | `ThayTheCho` + `Fallback` trong DTO, `SoBuocThayThe` ở cấp kế hoạch, **không** cộng vào tổng agent |
| dashboard (bảng chạy khan) | **không nói gì cả — kể cả về bước gỡ lại** | thêm dòng "KHÔNG chạy nếu suôn sẻ — bước gỡ lại/chạy thay cho …". Lỗ này có từ trước cho `compensate`; vá luôn cả hai |

Ba nơi (bộ chạy, `flow show`, chạy khan) giờ hỏi **cùng một hàm**
`flow.BuocNgoaiLichThuong` cho câu "bước nào sẽ không chạy". Ba nơi tự đi đếm là
ba cơ hội để lệch — đúng lớp lỗi mục 1.4 của #199 ghi lại.

#### Số đo SAU khi sửa (cùng khuôn, 0 token)

| # | cảnh | bước thay thế | bước sau | lượt chạy |
|---|---|---|---|---|
| **#62** | thay là gốc DAG, chính **hỏng** | **chạy, và chạy SAU** khi chính hỏng | — | `completed` |
| **#63** | thay khai `needs`, chính **hỏng** | **chạy** (trước đây không bao giờ) | — | `completed` |
| **#64** | **không có gì hỏng** | **KHÔNG chạy** (trước đây có) | — | `completed` |
| **#65** | bước sau đọc `{{steps.chinh.output}}` | chạy, in `KET-QUA-CUA-DU-PHONG` | ghi ra đĩa `[KET-QUA-CUA-DU-PHONG]` | `completed` |
| **#66** | bước thay thế **cũng hỏng** | hỏng | **KHÔNG chạy** | `failed` |

`sagent flow show` và `sagent flow run --kho` đo trực tiếp trên `fb-goc`, cả hai
in ra đúng dòng cảnh báo (xem trích ở trên).

### 1.2 Việc 2 — xem ARTIFACT từ dashboard

Action mới: **`flow.artifacts`**. Đây là một **động từ mới**, khác hẳn ba mảnh
của #199 (chúng đổi *cái flow làm gì*, không đổi *người ta bảo công cụ làm gì*).
Một action cho cả hai câu hỏi vì chúng cùng một cái giá — đọc file cục bộ, không
tốn token. Khác cặp `api.nang-luc` / `api.nang-luc-do`: cặp đó tách vì một cái
đọc bảng, một cái tiêu tiền.

**Bốn mặt, làm đủ ngay từ đầu:**

| mặt | thứ đã cắm |
|---|---|
| `api.Actions` | `"flow.artifacts"` · `API.FlowArtifacts(runID)` + `API.FlowArtifactDoc(runID, duong, tu)` |
| CLI | `sagent flow artifacts <#>` · `sagent flow artifacts <#> <đường-dẫn> [--tu <byte>]` · lệnh `__fart` trong bảng `commands` · **và cả trong `sagent help`** |
| HTTP | `GET /api/flow/artifacts?id=` · `GET /api/flow/artifact?id=&duong=&tu=` |
| web-UI | khối **"File để lại"** trong *Tiến độ lượt chạy* (`index.html`): liệt kê, bấm để đọc, nút **Đọc tiếp** khi bị cắt |

Đã đo thật trên lượt chạy **#68** (bước `shell`, 0 token):

```
  Artifact của lượt chạy #68 — flow do-artifact-xem

   BƯỚC           TÊN                              BYTE  ĐƯỜNG DẪN
   sinh           bao-cao                        126893  sinh/bao-cao.txt

  1 file · 126893 byte · C:\Users\Administrator\.ai-accounts\artifacts\run-68
```

#### TRẦN: 65.536 byte mỗi lần đọc, và nói rõ khi bị cắt

Artifact đo được hôm qua là **60.094 byte**; artifact tôi đo hôm nay là
**126.893 byte** — tức là ca "lớn hơn nhiều" không phải giả định, nó xảy ra ngay
ở phép đo thứ hai. Trần **65.536** chọn có lý do: nó **đủ chứa artifact lớn nhất
đã đo trước đó** (60.094) nên ca thường không bao giờ bị cắt, mà vẫn có trần
thật cho ca bất thường.

Bị cắt thì **không im lặng**, và trần là một **cửa sổ trượt** chứ không phải bức
tường:

```
dong 1042 chua mot
  ── BỊ CẮT: mới đọc 65536/126893 byte, còn 61357 byte nữa ──
  Đọc tiếp: sagent flow artifacts 68 sinh/bao-cao.txt --tu 65536
```

```
$ sagent flow artifacts 68 sinh/bao-cao.txt --tu 65536
it chu tieng Viet: bao cao, ban va, ket qua        ◄── nối liền "dong 1042 chua mot "
dong 1043 chua mot it chu tieng Viet: ...
```

Ghép các khúc lại ra **đúng** file, không mất byte nào, không lặp — có bài kiểm
ghim (`TestArtifactLonHonTranThiNOIRAVaDocTiepDuoc`). Đọc quá đuôi **không phải
lỗi**: nó trả khúc rỗng, để vòng lặp của người gọi dừng được tự nhiên.

Cắt theo byte chặt đôi ký tự UTF-8 ở mép — chuyện xảy ra thật với tiếng Việt
(gần như mọi chữ có dấu là 2–3 byte). Mẩu cụt ở đuôi bị bỏ đi **chỉ khi còn phần
sau**; cắt ở cuối file thì mẩu đó là dữ liệu thật và giấu nó là nói dối về file.

#### KHÔNG để lọt đường dẫn tuỳ ý

Đây là endpoint **đọc file** trên một cổng mà dash tự biết có thể không nằm trên
loopback (`s.exposed`). Ba lớp, và **lớp thứ ba mới là lớp thật**:

```
  1. cú pháp   dùng lại ĐÚNG hàm canh đường dẫn của flows.toml (duongDanTuongDoi):
               không tuyệt đối, không tên ổ đĩa, không `..`, không UNC
  2. ghép      Join vào artifacts/run-<id>/ rồi Clean
  3. ĐĨA       EvalSymlinks CẢ HAI vế rồi so bằng filepath.Rel
```

Lớp 3 chặn ca mà hai lớp lọc chuỗi **không thể** bắt: thư mục artifact là chỗ
**agent ghi vào**. Một agent — hoặc một bước `shell` trong flow ai đó gửi tới —
tạo được `run-51/soi/tat.lnk → ~/.ai-accounts/keys/` mà **không cần một dấu `..`
nào**. Chuỗi đường dẫn hoàn toàn vô tội; chỉ có đĩa mới biết nó đi đâu.
`filepath.Rel` chứ không `HasPrefix`: `…/run-5` là tiền tố chuỗi của `…/run-51`.

Đo thật bằng lệnh:

| đường dẫn gõ vào | kết quả |
|---|---|
| `../../telegram.json` | ✗ `"…" đi ra ngoài thư mục artifact của lượt chạy #68` |
| `../run-67/sinh/bao-cao.txt` | ✗ (dù nó vẫn nằm trong `artifacts/`) |
| `C:/Windows/win.ini` | ✗ `phải là đường dẫn TƯƠNG ĐỐI trong …` |
| `/etc/passwd` | ✗ |
| `..` | ✗ |
| `sinh/../../../keys` | ✗ |

Thông điệp lỗi **cố ý không phân biệt** "không có file" với "không có quyền": hai
câu đó là hai câu trả lời khác nhau cho người dò tìm, và chúng vẽ được bản đồ đĩa
của máy chủ. Người dùng thật thì đã có danh sách file từ `flow.artifacts`.

#### DTO: không để secret ra client

Artifact là **CỬA RA THỨ BA** của chữ do agent sinh ra — hai cửa kia là nhật ký
phiên, và `redaction_test.go` canh chúng. Một agent viết báo cáo có dán khoá API
vào là chuyện **đã xảy ra** ở nhật ký; không có lý do gì nó không xảy ra ở
artifact, và artifact thì đi thẳng ra cổng HTTP. Nên **nội dung đi qua đúng
`redaction.Che`** mà nhật ký đi qua.

Che **nội dung**, **không** che `Dir`: đường dẫn là thứ người vận hành cần để mở
file bằng tay, `sagent flow list` đã in nguyên đường dẫn như vậy từ trước, và
`KeHoachKho.Dir` cũng thế. Che nó đi thì trường đó thành vô dụng ở đúng mặt cần
nó nhất. Đây là một lựa chọn, không phải một chỗ sót — nói ra để ai không đồng ý
thì đổi được.

DTO chỉ mang những gì bảng cần: `duong` (**tương đối**, không tuyệt đối — cho
người gọi quen đường dẫn tuyệt đối là mời họ thử sửa nó), `buoc`, `ten`, `byte`,
`sua`, `nhiPhan`. File nhị phân bị đánh dấu **ngay ở bảng liệt kê** và trả về
`chu` rỗng: ném byte thô vào trình duyệt không giúp ai đọc được gì.

Ba chỗ **cố ý nói ra thay vì giấu**:
- file có trên đĩa mà **không bước nào khai** → `ten` rỗng, vẫn hiện, ghi rõ
  `(không bước nào khai)`;
- **mất định nghĩa flow** → `thieuDinhNghia`, vì cột tên trống vì *không tra
  được* khác hẳn trống vì *không ai khai*;
- lượt chạy **không có trong sổ** → báo lỗi thẳng, không trả bảng rỗng (bảng rỗng
  đọc thành "chưa có artifact").

#### Dashboard

Khối "File để lại" nằm trong *Tiến độ lượt chạy* — cùng chỗ với danh sách bước,
vì artifact là đường truyền **thứ hai** giữa các bước và danh sách bước không hé
lộ một chữ nào về nó. Ràng buộc đã giữ: offline tuyệt đối (không CDN), vanilla
HTML/CSS/JS, **không emoji làm icon**, chỉ dùng token có sẵn (chrome đơn sắc, chỉ
mượn token trạng thái *chờ* cho câu "bị cắt" — vì file vẫn nguyên vẹn trên đĩa,
đó không phải lỗi), dữ liệu máy dùng font mono, `:focus-visible` cho mọi nút.
`prefers-reduced-motion` đã có sẵn ở tầng trang và khối này **không thêm animation
nào**.

### 1.3 Bằng chứng: test ĐỎ khi gỡ phần sửa ra

Thử **thật**, từng đột biến một, khôi phục sau mỗi lần. Không phải một lời hứa.

| # | gỡ cái gì ra | test đỏ | đỏ ra sao |
|---|---|---|---|
| 1 | `case OnFailFallback` về bản cũ (chỉ `Warnf`) | 8 bài, dẫn đầu `TestFallbackChayBuocThayTheRoiDIETIEP` | bước chạy thay không để lại dấu vết nào trên đĩa |
| 2 | `BuocThayThe` khỏi `BuocNgoaiLichThuong` | `TestBuocChayThayKHONGChayONhungLuotBinhThuong` +3 | bước chạy thay chạy ở đợt đầu dù không có gì hỏng |
| 3 | `ganKetQuaThayThe` trong `chayThayThe` | `TestKetQuaBuocChayThayDocDuocBangTenBuocHONG` | `tham số 6 cần kết quả của bước "chinh" nhưng bước đó không để lại gì` |
| 4 | `ganKetQuaThayThe` trong `execute` | **chỉ** `TestChayLaiSauRaoDuyet…` | #3 vẫn XANH — nửa còn thiếu chỉ lộ ra khi có `Resume` |
| 5 | đóng hẳn cửa `fallback` trong `choDiTiep` | 3 bài | bước sau bị chặn dù đã có người làm thay |
| 6 | điều kiện `state(Fallback) == done` | `TestChayLaiMotLuotDaHONGThiBuocSauVANBiChan` | chạy lại một lượt đã hỏng thì bước sau LẠI chạy |
| 7 | `VanDeFallback` khỏi `Validate` | 5 bài validate | mọi cảnh báo/lỗi về bước chạy thay biến mất |
| 8 | đánh dấu `ThayTheCho` trong `chaykho` | `TestChayKhoNoiRoCaBuocChayThay` | bảng chạy khan nói bước chạy thay sẽ chạy |
| 9 | `flow.DuongDanArtifactAnToan` khỏi `FlowArtifactDoc` | 3 bài api + `TestEndpointArtifactChanThoatThuMuc` | `ĐỌC ĐƯỢC "../../bi-mat.txt"` — và qua HTTP là **200** kèm nguyên nội dung |
| 10 | **chỉ** hai lời gọi `EvalSymlinks` | **chỉ** `TestKhongDocDuocQuaLienKetMem` | đọc được `KHOA-API-THAT` qua liên kết mềm — chứng minh lớp 3 là lớp **duy nhất** chặn được ca này |
| 11 | `out.BiCat = out.ConLai > 0` | `TestArtifactLonHonTran…`, `TestCatGiuaChuTiengViet…` | `nói không bị cắt mà còn 98304 byte` |
| 12 | `redaction.Che` trong `FlowArtifactDoc` | `TestNoiDungArtifactDuocCheBiMat` | `sk-ant-api03-…` và email ra thẳng client |

Chỗ hỏng của **1, 2, 3, 4, 5, 6, 8** nằm ở **chỗ gọi** chứ không ở hàm — nên test
tương ứng chạy `Runner.Start`/`Runner.Resume` thật, với `store.DB` thật và **tiến
trình con thật**. Chỗ hỏng của **9** được đo ở **hai tầng**: tầng hàm
(`internal/api`) và tầng HTTP thật qua toàn bộ mux + guard (`internal/dash`) —
hai bài riêng, vì một cái chứng minh phép lọc đúng, cái kia chứng minh nó **được
cắm vào**.

---

## 2. Sự cố

### 2.1 Một lời chú trong bài kiểm của chính tôi nói SAI chỗ

`TestBuocChayThayCungHONG` bản đầu ghi *"gỡ điều kiện `state(Fallback) == done`
trong `choDiTiep` thì bài này đỏ"*. Chạy đột biến ra **XANH**.

Lý do: khi bước chạy thay hỏng, `xuLyHong` trả về `dungLuot = true`, lượt chạy
dừng ngay tại đó và bước sau **chưa kịp được xét** — `choDiTiep` không được gọi
lần nào. Điều kiện ấy chỉ gánh việc trên **một** đường: `sagent flow resume <#>`
chạy lại một lượt đã `failed` (`Resume` chỉ dừng sớm với `completed`/`cancelled`).
Lúc đó trạng thái dựng lại từ sổ, không còn ai đang ở giữa chừng để dừng lượt
chạy, và nếu cửa mở mà không kiểm thì bước sau chạy **trên nền một việc không ai
làm được** — đúng lỗi #23 mà `choDiTiep` sinh ra để chống, chỉ đổi tên trường.

Đã sửa: viết `TestChayLaiMotLuotDaHONGThiBuocSauVANBiChan` đo đúng đường đó (đột
biến #6 giờ đỏ), và sửa lại lời chú của bài cũ cho khớp sự thật. **Ghi ra vì đây
là kiểu sai nguy hiểm nhất trong một bộ test**: một dòng "gỡ ra thì đỏ ở đây" nói
sai làm người sau tin rằng chỗ đó đã có người canh.

### 2.2 Câu từ chối đường dẫn nói sai RANH GIỚI

Đo lần đầu, `sagent flow artifacts 68 ../../telegram.json` trả lời *"đi ra ngoài
**thư mục của bước**"* — trong khi thứ vừa bị chặn là một đường dẫn tương đối so
với **lượt chạy**. Nguyên nhân: tôi dùng lại `duongDanArtifact`, mà câu chữ của
hàm đó viết cho đường dẫn khai trong `flows.toml`.

Không nghiêm trọng về an toàn (nó vẫn chặn đúng), nhưng nó **dẫn người đọc sửa
sai chỗ**. Đã tách `duongDanTuongDoi(rel, choNao)` để hai chỗ gọi dùng chung một
phép kiểm mà nói đúng ranh giới của mình. Giờ: *"đi ra ngoài thư mục artifact của
lượt chạy #68"*.

### 2.3 Một bài kiểm của tôi xanh vì LÝ DO SAI

`TestEndpointArtifactChanThoatThuMuc` bản đầu xanh ngay lần chạy đầu — nhưng vì
lượt chạy #1 chưa có thư mục artifact nào, nên hàm dừng ở *"lượt chạy này không
để lại artifact nào"* **trước khi** chạm tới lớp chặn. Đã sửa: dựng thư mục và
một file thật, cộng thêm một khẳng định rằng **đường dẫn hợp lệ phải ra 200** —
không có dòng đó thì mọi khẳng định còn lại cũng đúng với một endpoint từ chối
tất cả. Sau khi sửa, đột biến #9 mới làm nó đỏ.

### 2.4 Một bài kiểm chập chờn CÓ SẴN (không phải do tôi)

`TestMergeHaiLuotXongNguocThuTuVanRaGiongNhau` đỏ **một lần** trong lượt
`go test ./...` giữa buổi:

```
merge_test.go:104: hai lượt xong CÙNG thứ tự ([p-cam p-banh p-an]) —
                   bài test không kiểm được gì, cần tăng độ trễ
```

Chính bài kiểm tự nói ra rằng nó phụ thuộc độ trễ. Đã kiểm: chạy lại 5 lần trên
cây có sửa → xanh; **stash sạch toàn bộ phần sửa của tôi** rồi chạy 8 lần → cũng
xanh. Tức là nó chập chờn theo tải máy chứ không liên quan tới thay đổi này.
**Không sửa** — `merge.go`/`merge_test.go` không nằm trong việc được giao, và sửa
một bài kiểm chập chờn khi chưa đo được nguyên nhân là cách nhanh nhất để làm nó
im lặng thay vì hết chập chờn. Ghi lại để người sau không phải tìm lại từ đầu.

### 2.5 Chưa mở được dashboard bằng trình duyệt thật

Đã dựng một bản dash từ binary vừa build (`--port 8901`) và nó trả `303` (chuyển
sang trang đăng nhập), tức server sống và route mới nằm sau đúng guard. Nhưng
**không đăng nhập được**: tôi không biết mật khẩu dashboard của bạn, và
`sagent dash --set-password` sẽ **ghi đè mật khẩu thật** — một tác dụng ra ngoài
mà tôi không được phép gây ra để phục vụ một phép đo. Tiến trình tạm đã được dừng
(PID 1552, cổng 8901 đã nhả).

Nên phần web-UI được chứng minh bằng: (a) `TestEndpointArtifactChanThoatThuMuc`
gọi **HTTP thật** qua toàn bộ mux + guard với đúng hình dạng URL mà trang dùng
(`?id=1&duong=x%2Fok.txt` → **200**); (b) `TestTrangGoiCaHaiDuongArtifact` ghim
rằng `index.html` **thật sự gọi** cả hai đường (đã cắt bình luận trước khi dò);
(c) `TestMatWebDocDungTenTruongArtifact` ghim rằng trang đọc **đúng tên trường**
mà DTO phát ra, lấy tên từ chính struct qua `json.Marshal`. Thứ **chưa** đo được
là khối đó **trông** thế nào trên màn hình thật.

### 2.6 Không có sự cố nào về công cụ

Không dùng `&&` trong PowerShell (mọi lệnh chạy từng cái một qua Bash/POSIX).
Không sửa file `.ps1` nào nên không có chuyện em dash. Không đụng `main`. Không
đụng `docs/MASTER-PLAN.md`, `internal/dash/web/docs/*`, `master-plan.html`,
`tools/*`, `docs/DO-LUONG.md`, `docs/SO-NO-DO-LUONG.md`.

---

## 3. Bước tiếp theo

Theo thứ tự tôi nghĩ là đáng làm nhất:

1. **`{{artifacts.x}}` KHÔNG đi qua `fallback`, chỉ `{{steps.x.output}}` đi
   qua.** Đây là ranh giới đã biết của bản này, nói ra chứ không giấu: `Validate`
   chặn hai bước khai trùng tên artifact, nên bước chạy thay **không thể** hứa
   cùng cái tên với bước nó thay. Bước sau đọc `{{artifacts.bao-cao}}` sẽ nhận ô
   trống dù hàng dự phòng đã làm xong việc. Cách sửa đúng theo tôi là cho phép
   trùng tên **khi và chỉ khi** một bước là hàng dự phòng của bước kia, rồi
   `MoiTruongArtifact` chọn theo bước nào thật sự `done` — nhưng đó là đụng vào
   luật "một tên một chỗ" của `nguoiSanXuat`, và tôi không muốn đổi luật đó bằng
   một quyết định vội.
2. **Lỗ mất dữ liệu ở `flow.html` (mục 2.2 của #199) VẪN CHƯA VÁ.** Bấm Lưu trên
   bảng vẽ workflow xoá trắng `doc_duoc`, `phai_co`, `vai_tro`, `fallback`,
   `compensate`… — kể cả `fallback` mà bản này vừa làm cho chạy thật. Tôi không
   sửa vì `internal/dash/web/flow.html` có agent khác đang làm việc quanh đó.
   Đây vẫn là việc gấp nhất trong danh sách.
3. **Xoá artifact từ dashboard.** `DonArtifact` chỉ chạy tự động theo tuổi
   (7 ngày) ở đầu mỗi lượt chạy. Một lượt để lại 400 MB thì hôm nay không có
   cách nào dọn ngay ngoài `rm -rf` bằng tay. Nếu làm thì đó **là** một action
   mới nữa, và là action **có tác dụng ra ngoài** — phải có xác nhận, khác hẳn
   `flow.artifacts` chỉ đọc.
4. **Tải artifact về nguyên file** (`Content-Disposition`), cho ca file nhị phân
   và ca file to hơn nhiều lần trần. Cố ý chưa làm: nó là một cửa ra **khác** với
   cửa đọc-có-che ở trên, và một đường tải nguyên byte thì `redaction.Che` không
   áp được — cần quyết định riêng chứ không nên gộp vào cùng endpoint.
5. **`foreach` + `fallback`** chưa đo. Có bài kiểm `TestForEachHongCungGoiBuocChayThay`
   ghim rằng hai nhánh cùng gọi một chỗ, nhưng **một** bước chạy thay cho **N**
   lượt lặp hỏng nghĩa là gì thì chưa ai trả lời — hiện nó chạy đúng một lần cho
   cả bước.
6. **Đo với bước `agent` thật.** Toàn bộ số trong báo cáo này từ bước `shell`
   (0 token). Con số đáng đo nhất còn thiếu: một flow `hoi-claude → hoi-grok`
   thật, xem hàng dự phòng có cứu được lượt chạy khi tài khoản chính hết hạn mức
   không — đó mới là cảnh `fallback` sinh ra để phục vụ.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Đọc mục 2.3 của `#199`, `compensate.go`, `step.go`, `runner.go`, `state.go` để xác nhận hiện trạng | Opus 5 (1M) | thấp — đọc, không suy luận |
| **Tự kiểm lại lời của #199 bằng 3 lượt chạy thật** (#59–#61) | Opus 5 (1M) | trung bình — soạn flow đúng khuôn đo, đọc kết quả trên ĐĨA chứ không đọc log |
| Thiết kế `fallback`: gán kết quả vào đâu, cửa cho bước sau mở khi nào, dựng lại thế nào sau `Resume` | Opus 5 (1M) | **cao** — phần lớn công của việc 1 nằm ở đây, không nằm ở mã |
| Cắm `chayThayThe` + `BuocNgoaiLichThuong` + `VanDeFallback` | Opus 5 (1M) | trung bình |
| Vá ba mặt hiển thị (`flow show`, chạy khan, dashboard) cho khớp hành vi mới | Opus 5 (1M) | trung bình — nhiều chỗ nhỏ, mỗi chỗ một câu chữ phải đúng |
| Viết + chạy 8 đột biến của việc 1, khôi phục sau mỗi lần | Opus 5 (1M) | trung bình — chậm nhưng máy móc; giá trị nằm ở việc BẮT ĐƯỢC lời chú sai (mục 2.1) |
| **Thiết kế cửa an toàn cho endpoint đọc file**: ba lớp, và nghĩ ra ca liên kết mềm | Opus 5 (1M) | **cao** — đây là chỗ sai một lần là mất khoá API |
| Thiết kế trần + cửa sổ trượt + cắt UTF-8 ở mép | Opus 5 (1M) | trung bình |
| Viết `internal/api/artifact.go` + CLI + 2 endpoint + panel dashboard | Opus 5 (1M) | trung bình |
| Viết + chạy 4 đột biến của việc 2 | Opus 5 (1M) | trung bình |
| Đo thật `flow artifacts` trên lượt #68 (126.893 byte), 6 ca thoát thư mục | Opus 5 (1M) | thấp |
| Viết báo cáo này | Opus 5 (1M) | trung bình |

---

## 5. Nhận xét tự do

**Việc đáng giá nhất tôi làm hôm nay không phải là sửa `fallback` — mà là ĐO nó
trước khi sửa.** #199 nói đúng, nhưng nó nói thiếu hàng nặng nhất: bước dự phòng
chạy **cả khi không có gì hỏng**. Nếu tôi sửa theo mô tả thay vì theo phép đo,
tôi sẽ cắm `chayThayThe` vào rồi vẫn để bước đó là gốc DAG — và tính năng sẽ vừa
chạy thay đúng lúc, vừa vẫn đốt hạn mức ở mọi lượt suôn sẻ. Bản vá đó sẽ **trông
như đã xong**, và đó là loại bản vá tệ nhất.

**Chiều ngược lại cũng đúng, và tôi nghĩ nó đáng nói hơn.** Đột biến #6 chạy ra
XANH, tức là một dòng "gỡ ra thì đỏ ở đây" mà chính tôi vừa viết là **sai**. Nếu
tôi tin lời chú của mình thay vì chạy thử, bộ test này sẽ có một chỗ trống mang
biển "đã có người canh". Cả hai chuyện — không tin mô tả của #199, và không tin
mô tả của chính mình — là **cùng một kỷ luật**, và nó chỉ có giá khi được áp cho
cả hai phía.

**Về `fallback` và `compensate`:** hai thứ này khác nhau đúng một chữ ("rồi dừng"
với "rồi đi tiếp") nhưng có chung sáu tính chất, và cả sáu đều là những cái bẫy
đã được ai đó trả giá để tìm ra. Cách rẻ nhất để `fallback` không lặp lại lần thứ
hai không phải là viết cẩn thận — mà là bắt hai bên **dùng chung một hàm** cho
câu hỏi chung ("bước nào sẽ không chạy"). `BuocNgoaiLichThuong` tồn tại vì lý do
đó, không vì gọn gàng. Bài học mục 1.4 của #199 (hai nhánh tự xét riêng thì lệch)
đúng ở tầng bộ chạy, và nó cũng đúng ở tầng hiển thị: trước bản này, dashboard
đã im lặng về bước gỡ lại suốt từ hôm qua mà không bài kiểm nào đỏ.

**Về endpoint đọc file:** lớp chống `..` là lớp ai cũng nghĩ ra, và nó là lớp
**ít quan trọng nhất** trong ba lớp. Thứ thật sự nguy hiểm ở đây là thư mục
artifact chính là chỗ **agent ghi vào** — tức là kẻ có thể tạo liên kết mềm nằm
ngay bên trong hàng rào. Đột biến #10 (gỡ **chỉ** `EvalSymlinks`, giữ nguyên hai
lớp kia) làm đúng một bài đỏ và bài đó đọc được nguyên khoá API. Nếu tôi chỉ viết
bài kiểm cho `..` thì bộ test sẽ xanh, tài liệu sẽ nói "đã chặn thoát thư mục", và
lỗ vẫn còn nguyên.

**Chỗ tôi cố ý dừng lại và không tự quyết:** `{{artifacts.x}}` chưa đi qua
`fallback` (mục 3.1). Sửa nó là đụng vào luật "một tên artifact một chỗ sản
xuất" — luật đang giữ cho `{{artifacts.x}}` trỏ tới đúng một nơi. Đổi luật đó để
chạy cho được một ca là kiểu đánh đổi mà ba tuần sau không ai nhớ lý do. Thà để
nó là một dòng ghi rõ trong báo cáo còn hơn là một ngoại lệ không ai giải thích
được trong mã.
