# Báo cáo: alias CLI · node `route` + `merge` · ba trường mới hiện ra

Nhánh: `sagent/cli-22-08` (tạo từ `main`, `git rev-list --count main..HEAD` = 0 lúc tạo).
Ngày: 22/08/2026.

Nghiệm thu, chạy **từng lệnh riêng biệt**, cả ba xanh:

```
go build ./...     exit 0
go vet ./...       exit 0
go test ./...      exit 0   (25 gói ok, không gói nào FAIL)
```

---

## 1. Đã làm

### VIỆC 1 — alias `tk` và `ccswitch`

**Lý do hoãn đã hết hạn.** Kế hoạch ghi "làm khi viết installer phát hành";
installer có từ Pha 7 (`install/cai-dat.ps1`, `install/get.ps1`,
`.github/workflows/phat-hanh.yml`). Hiện trạng đo được trước khi sửa:
`grep -ni "alias\|ccswitch" install/` = **0 dòng**.

**Chọn cách nào, và vì sao.** Ba cách làm được trên Windows:

| Cách | Tốn đĩa | Sau khi NÂNG CẤP |
|---|---|---|
| (a) file `.cmd` nhỏ cạnh binary — **CHỌN** | 23 byte/alias | luôn đúng |
| (b) bản sao `tk.exe`, `ccswitch.exe` | +32 MB (2 × 16.016.384 byte, đo thật) | **chạy binary CŨ, im lặng** |
| (c) hard link tới `sagent.exe` | 0 | **chạy binary CŨ, im lặng** |

(b) và (c) hỏng theo cùng một kiểu: bản cài đặt thay `sagent.exe` bằng một
**file mới** (`Move-Item`), nên bản sao vẫn là bản cũ và hard link vẫn trỏ tới
nội dung cũ. Người dùng gõ `tk` sau khi nâng cấp chạy phiên bản cũ mà **không
một dòng nào nói ra**. Đó đúng là lớp lỗi dự án này sợ nhất: hai thứ cùng tên,
lặng lẽ lệch nhau.

(a) không có vấn đề đó — shim trỏ tới `sagent.exe` **theo đường dẫn**, nên nó
luôn chạy đúng file đang nằm đó, kể cả khi ai đó tự `go build -o $Exe`.

Giá phải trả của (a), nói thẳng: mỗi lần gọi qua alias tốn thêm một tiến trình
`cmd.exe`, và bấm Ctrl+C vào lệnh chạy lâu (`tk dash`) thì cmd hỏi
"Terminate batch job (Y/N)?". **Cả hai đều nhìn thấy được. Chạy nhầm phiên bản
thì không.**

**Vì sao KHÔNG đọc `os.Args[0]`.** Đọc tên binary chỉ cần khi chương trình muốn
**cư xử khác** theo tên gọi. Ở đây ta muốn ngược lại — giống hệt. Thêm một
nhánh rẽ theo tên là thêm một đường cho hai lối gọi lệch nhau.

**Nội dung shim** (nguồn sự thật duy nhất, `install/cai-dat.ps1`):

```powershell
$TenAlias = @('tk','ccswitch')
$ShimNoiDung = '@"%~dp0sagent.exe" %*'
```

`%*` chứ không phải `%1 %2 %3`: chỉ `%*` giữ nguyên phần đuôi dòng lệnh, tức
giữ được cả tham số có dấu cách lẫn tham số thứ mười trở đi.

**ĐÃ CHẠY THẬT một lần.** Chạy installer vào một `USERPROFILE` tạm để không đụng
`sagent.exe` đang chạy trên máy; `User PATH` đã chụp trước và khôi phục nguyên
trạng sau khi đo.

```
  ✓ Go: C:\Users\Administrator\AppData\Local\Programs\Go\bin\go.exe
  ✓ Đã cài: ...\hometest-77c6750f\bin\sagent.exe
  ✓ alias: tk
  ✓ alias: ccswitch

Name           Length
----           ------
ccswitch.cmd       23
sagent.exe   16016384
tk.cmd             23
```

Bốn phép đo trên bản vừa cài:

```
=== 1. sagent.exe version ===      sagent dev · windows/amd64 · go go1.25.13
=== 2. tk version ===              sagent dev · windows/amd64 · go go1.25.13   LASTEXITCODE=0
=== 3. ccswitch version ===        sagent dev · windows/amd64 · go go1.25.13   LASTEXITCODE=0
=== 4. ma thoat khac 0 (flow validate tren flow hong) ===
  -> tk       LASTEXITCODE=1
  -> ccswitch LASTEXITCODE=1
  -> sagent   LASTEXITCODE=1
```

Điểm số 4 là điểm đáng đo nhất: shim **không nuốt mã thoát**. Nuốt mã thoát thì
CI đọc mọi lượt hỏng thành xanh.

**ĐƯỜNG GỠ.** Trước bản này **không có** đường gỡ nào — không script, không lệnh.
Thêm cờ `-Go` vào `cai-dat.ps1`: xoá binary, xoá **cả hai** shim, dọn các bản
`sagent.exe.cu-*` mà mỗi lần nâng cấp để lại, và hỏi có gỡ `PATH` không. Chạy
thật:

```
  ✓ da xoa alias: tk
  ✓ da xoa alias: ccswitch
  ✓ da xoa: ...\hometest-77c6750f\bin\sagent.exe
  ! van giu ...\bin trong PATH
--- con lai trong ...\bin ---  (rong)
```

Trả lời thẳng câu "`sagent xoa` có dọn cả alias không": **KHÔNG, và không nên**.
`sagent xoa` là action `profile.remove` — xoá một **tài khoản** trong sổ hồ sơ,
không dính gì tới binary trên đĩa. Đường gỡ binary là `cai-dat.ps1 -Go`.

**Bằng chứng test đỏ khi gỡ phần sửa ra** (`cmd/sagent/alias_test.go`):

| Gỡ cái gì | Kết quả |
|---|---|
| đổi shim thành `%1 %2 %3` | `TestInstallerKhaiDuAlias` ĐỎ **và** `TestShimAliasChayThat/tk` + `/ccswitch` ĐỎ ("không thấy tham số [in]") |
| bỏ `'ccswitch'` khỏi `$TenAlias` | `TestInstallerKhaiDuAlias` ĐỎ: `installer không còn tạo alias "ccswitch" (đang có: [tk])` |

`TestShimAliasChayThat` **đọc thẳng hai dòng `$TenAlias` / `$ShimNoiDung` ra khỏi
`cai-dat.ps1` rồi CHẠY shim đó thật** (chép file test thành `sagent.exe` trong
thư mục tạm, gọi qua `cmd /c tk.cmd`, kiểm tham số có dấu cách và mã thoát 7).
Không chép lại nội dung shim vào Go — chép ra là mở đường cho hai bản lệch nhau,
mà bản lệch ấy đúng là bản người dùng nhận.

---

### VIỆC 2 — node `route` và node `merge`

#### `route`: tách thành node riêng

Trước: `route` là **thuộc tính** của bước `model` (`Step.Route`). Sau: một node
chọn đường rồi **chuyền tên cho bước sau**.

```toml
[[flow.x.step]]
  id     = "chon"
  type   = "route"
  routes = ["grok", "deepseek"]     # thứ tự ưu tiên; bỏ trống = default_route

[[flow.x.step]]
  id     = "hoi"
  type   = "model"
  needs  = ["chon"]
  route  = "{{steps.chon.output}}"
  prompt = "..."
```

**Vì sao đáng tách.** Flow có năm bước `model` cùng đi một đường; đường đó chết
thì mỗi bước tự đi hỏi lại, và mỗi bước có thể rơi sang một đường dự phòng
**khác nhau** tuỳ lúc nó chạy. Năm bước của cùng một lượt trả lời bằng năm mô
hình khác nhau, mà bảng chỉ ghi "model".

**Dùng lại, không làm lại.** Gói `internal/flow` khai một interface hẹp
`RouteChon`; phần cắm ở `internal/api/routenode.go` gọi lại đúng
`API.ThuTuRoute` (thứ tự route chính + một route dự phòng, cùng luật với
`sagent api`) và `aiapi.Kiem` (`GET /models`, **không tốn token**). **Không một
dòng logic chọn đường nào được viết lại** — nhờ vậy `sagent route kiem`,
`sagent api` và node này không thể trôi khỏi nhau.

"Dùng được" = `SucKhoe.Dung()`, tức route **sống VÀ model khai có thật**. Không
hạ xuống thành `Song`: route sống mà model khai sai thì bước sau vẫn hỏng, chỉ
khác là hỏng muộn hơn và kèm một thông điệp chẳng nhắc gì tới cấu hình.

Hai quyết định phụ, mỗi cái vì một cách hỏng:

- **Output là ĐÚNG cái tên, không gì khác** — không "đã chọn: grok", không xuống
  dòng. Nó đi thẳng vào `route = "{{steps.chon.output}}"`, nơi một ký tự thừa là
  một đường không tồn tại. Phần người đọc cần (ứng viên nào bị loại vì sao) đi ra
  event bus, **kể cả khi hỏng**.
- **Hết đường thì DỪNG HẲN**, không đoán bừa một cái tên. Đoán bừa nghĩa là bước
  sau đem tên ấy đi gọi thật: tốn thời gian chờ, tốn tiền nếu trúng, và hỏng
  bằng một thông báo không liên quan tới nguyên nhân.

**Mắt xích suýt đứt.** `Step.Route` trước đây đi **thẳng** vào lời gọi mà không
qua `Expand` — viết `route = "{{steps.chon.output}}"` sẽ cho ra một tên route
không tồn tại, tức là cả node `route` chạy đúng mà **vô dụng**. Đã sửa, và
`TestRouteChonDuongRoiChuyenChoBuocSau` khẳng định cả chuỗi: bước `route` loại
đường chết → chọn đường sống → `modelGia` ghi lại rằng nó **được gọi bằng đúng
đường đó**.

Thiếu kết quả ở `route` thì **dừng ngay và nói tên bước**, không chốt placeholder
thành câu tiếng Việt — cùng luật với tham số bước `shell`, cùng lý do: thông báo
lỗi sinh ra từ giá trị bịa sẽ chỉ người đọc đi sai hướng.

#### `merge`: bật lên, và trả lời hai câu khó

**Trước hết: `merge` KHÔNG phải gộp nhánh git.** Ghi chú cũ ở `flow.go:53` là
"gộp nhánh — hành động nguy hiểm" với `implemented = false` và lời hẹn "còn chờ
cơ chế merge an toàn". Cái cơ chế ấy **sẽ không bao giờ tới**: một node tự chạy
`git merge` là một node có quyền viết đè lên cây mã của người khác, và không có
cách nào làm việc đó an toàn bằng một dòng TOML. Muốn gộp nhánh thì đường cũ vẫn
đúng hơn: `shell` chạy `git merge` đứng sau `approve`, ở đó người duyệt nhìn
thấy chính xác lệnh sắp chạy.

Node này gộp **ĐẦU RA (chữ)** của N bước, và "an toàn" ở đây có nghĩa **đo
được**: hai lượt chạy giống hệt nhau cho ra đúng một khối chữ. Đã nói ra chuyện
đổi nghĩa này ngay trong mã (`internal/flow/merge.go`) chứ không lặng lẽ bật cờ.

**CÂU HỎI 1 — gộp N bước thì theo thứ tự nào?**

**Theo ĐÚNG thứ tự khai trong `needs`.** Ba câu trả lời khả dĩ:

- **(a) thứ tự XONG** — sai, và sai theo kiểu tệ nhất. Bộ chạy này chạy theo
  **đợt**, nhiều bước song song (`runWave`). Hai bước cùng đợt không có thứ tự
  xong nào cả: nó phụ thuộc vào mạng, vào máy, vào việc hôm nay nhà cung cấp trả
  lời nhanh hay chậm. Chạy hai lần một flow y hệt nhau ra hai khối chữ khác nhau,
  và không gì nói cho người đọc biết vì sao.
- **(b) thứ tự khai trong `needs`** — **CHỌN**. Nằm trong `flows.toml`, người
  viết flow nhìn thấy, sửa được, và không đổi giữa hai lần chạy.
- **(c) thứ tự id A-Z** — ổn định như (b) nhưng bắt người ta đặt tên bước theo
  bảng chữ cái để điều khiển thứ tự đọc. Điều khiển bằng tác dụng phụ.

Hệ quả trực tiếp: **merge không có `needs` là LỖI**, không phải "gộp mọi bước
trước". "Mọi bước trước" là một tập hợp không có thứ tự (map của Go trả ra ngẫu
nhiên); muốn xếp nó thì lại rơi về (a) hoặc (c).

**CÂU HỎI 2 — một bước trong số đó HỎNG thì merge ra gì?**

**Ra một khối có ĐỦ N MỤC, trong đó mục của bước hỏng là một dòng nói rõ nó
hỏng. Không bao giờ bỏ mục đi.**

Chuyện này xảy ra được khi nguồn khai `on_failure = "continue"` (`choDiTiep`
cho bước sau chạy tiếp), hoặc khi nguồn bị `skipped` (`when` không thoả). Bỏ mục
đi thì hai lượt chạy — một đủ ba nguồn, một mất nguồn giữa — cho ra hai khối chữ
mà nhìn vào **không phân biệt được cái nào thiếu**. Agent tổng hợp ở bước sau sẽ
viết một bản tổng kết tự tin dựa trên hai phần ba dữ liệu và không nói một câu
nào về phần thiếu. Đúng lớp hỏng của lượt #46 (`phai_co`) và #29 (placeholder
còn sót): không sập, chỉ lặng lẽ gật đầu.

Bốn câu khác nhau cho bốn tình huống khác nhau — vì chúng **khác nhau thật**:

```
=== cam ===
CAM nói C

=== banh ===
(bước "banh" HỎNG — không có kết quả để gộp)

=== rong ===
(bước "rong" xong nhưng KHÔNG để lại kết quả nào)

=== bo-qua ===
(bước "bo-qua" bị BỎ QUA — không chạy)
```

Cộng thêm: `doc_duoc` chặn nguồn thì mục đó là câu báo bị chặn (`merge` **không
đi vòng qua** `doc_duoc`; `manhGop` hỏi `ChoDoc` **trước khi** nhìn vào giá trị,
nên nó không phụ thuộc vào việc chỗ gọi có nhớ lọc hay không).

**Bài test quan trọng nhất** (`TestMergeHaiLuotXongNguocThuTuVanRaGiongNhau`):
chạy cùng một flow hai lượt, ép hai lượt **xong theo hai thứ tự ngược nhau**
bằng độ trễ đảo chiều, rồi đòi khối gộp giống nhau **đến từng byte**. Bài test
tự khẳng định luôn rằng hai lượt đã xong khác thứ tự thật — không có khẳng định
đó thì máy chạy nhanh có thể cho cùng thứ tự và bài test xanh mà chẳng chứng
minh được gì.

**Ràng buộc phụ đã cài** (cùng luật với những mảnh đã có):

- `merge` + `route` vào `loaiKhongGhiDuocFile`: khai `artifact` ở đó là hợp đồng
  chắc chắn không giữ được (merge chỉ nối chữ trong bộ nhớ, route chỉ trả về một
  cái tên).
- `foreach` + `merge`/`route`: chặn ở `Validate` **và** ở runtime (Flow còn dựng
  được thẳng bằng mã Go, không qua `validate`).
- `needs` trùng tên ở merge: lỗi (khối gộp sẽ có hai mục giống hệt nhau).
- `KhoaIdem`: bước `merge` băm **chính khối chữ đã gộp** (qua `cauHoi`), nên
  nguồn đổi kết quả thì merge chạy lại. Và `route` giờ được băm **sau khi thay
  biến** — trước đó băm chuỗi thô, nên hai lượt đi **hai đường khác nhau** vẫn ra
  cùng một khoá và lượt sau dùng lại câu trả lời của một mô hình khác hẳn.

**Một bài test có sẵn phải sửa:** `TestCanhBaoTypeChuaChayDuoc` mượn `TypeMerge`
làm ví dụ "loại khai rồi mà chưa chạy được". Bật merge + route xong thì **không
còn loại nào `implemented = false`**, nên bài đó đỏ dù hành vi nó kiểm không sai
một chút nào. Đã sửa để nó dựng một **loại giả** chỉ sống trong lúc chạy nó — bài
kiểm CƠ CHẾ mà buộc vào một node cụ thể thì sẽ đỏ vào đúng ngày node đó chạy được.

---

### VIỆC 3 — ba trường mới hiện ra

`artifact`, `idempotent`, `compensate` đã vào `main` nhưng `sagent flow show` và
chạy khan (`internal/api/chaykho.go`) chưa biết gì về chúng.

**Tệ nhất, và đã sửa:** chạy khan hiện bước gỡ lại **y hệt một bước sắp chạy** —
nó nằm trong đợt, có tài khoản, có prompt, và số agent của nó **được cộng vào
tổng "sắp đốt bao nhiêu phiên"**. Sự thật thì `runner.execute` ghi thẳng nó là
`skipped` ngay đầu lượt và chỉ gọi khi có sự cố. Sai theo hướng **THỪA**, và thừa
ở đây không vô hại: người đọc thấy một bước dọn dẹp trong kế hoạch nên yên tâm
rằng việc dọn sẽ xảy ra.

`sagent flow show batruong` — chạy thật:

```
   1. chon       [route]
      chọn đường: grok → deepseek
   2. go-lai     [shell]
      bước gỡ lại — chỉ chạy khi ap hỏng
      chạy: git checkout -- .
   3. hoi        [model]  ← chon
      đường: {{steps.chon.output}}
      idempotent: lượt trước đã làm xong đúng việc này thì bước NÀY BỊ BỎ QUA
   4. viet       [agent]
      1 agent
      prompt: Ghi ban va day du ra {{artifact_dir}}/ban-va.diff
      để lại file (artifact): ban-va → ban-va.diff
   5. ap         [shell]  ← viet
      chạy: git apply {{artifacts.ban-va}}
      hỏng thì: chạy bước gỡ lại "go-lai" rồi DỪNG cả lượt
   6. gop        [merge]  ← hoi, ap
      gộp đầu ra theo thứ tự: hoi → ap
```

`sagent flow run batruong --kho` — chạy thật (trích):

```
  Đợt 1 — 3 bước chạy SONG SONG
   · chon         [route] · đường grok → deepseek
   · viet         [agent]  claude:tns · 1 agent
       để lại file (artifact): ban-va → ban-va.diff
   · go-lai       [shell]
       ↩ BƯỚC GỠ LẠI — KHÔNG chạy trong lượt suôn sẻ; chỉ chạy khi ap hỏng
  Đợt 2 — 2 bước chạy SONG SONG
   · hoi          [model] · đường {{steps.chon.output}}
       Doc ky roi tra loi
       idempotent: lượt trước đã làm xong đúng việc này thì bước NÀY BỊ BỎ QUA
   · ap           [shell]
       hỏng thì: chạy bước gỡ lại "go-lai" rồi DỪNG cả lượt
  Đợt 3
   · gop          [merge]
       gộp đầu ra theo thứ tự: hoi → ap

  Tổng: 1 phiên agent
  Trong đó 1 bước là BƯỚC GỠ LẠI — không chạy nếu mọi thứ suôn sẻ, và KHÔNG tính vào tổng trên
```

`flow show` và bảng chạy khan gọi **chung một hàm** `moTaBaTruong(artifact,
idempotent, buocGoLai)` — hai mặt phải in đúng một câu chữ, và cách chắc chắn
nhất là chúng dùng chung một hàm chứ không phải hai hàm "giống nhau" ở hai file.

**Một lỗ có sẵn vá luôn vì đúng phạm vi:** node `model` và node `plugin` rơi vào
nhánh `default` của `buocKho`, tức bảng chạy khan đọc `s.Message` của chúng — một
trường **luôn rỗng** ở hai loại đó. Bước gọi model hiện ra **không một chữ nào
của câu hỏi**, đúng ở cái bảng sinh ra để trả lời "nó sẽ hỏi chúng nó cái gì".
Đã sửa; `TestChayKhoHienArtifactIdempotentRoute` khẳng định.

**Bằng chứng test đỏ khi gỡ phần sửa ra:** xoá khối đánh dấu bước gỡ lại trong
`FlowChayKho` →

```
--- FAIL: TestChayKhoNoiRoBuocChiChayKhiHong (0.03s)
    bước `go-lai` KHÔNG được đánh dấu là bước gỡ lại — bảng chạy khan đang
    nói nó sẽ chạy, trong khi nó chỉ chạy khi có sự cố
```

---

### LUẬT NGANG QUYỀN — bốn mặt

**Không việc nào trong ba việc trên đẻ ra action mới.** `api.Actions` không đổi
một dòng, và `TestNgangQuyenMoiHanhDongDeuCoLenhCLI` +
`TestKhongCoLenhNgoaiHopDong` vẫn xanh. Cụ thể:

| Việc | Có phải action mới? | Lý do |
|---|---|---|
| alias `tk`/`ccswitch` | **Không** | Nó là cách GỌI cùng một binary, không phải một việc mới hệ thống làm được. Không có gì để `api.Actions` biết. |
| node `route`, node `merge` | **Không** | Là **loại node** trong `flows.toml`, không phải hành động. Chúng đi qua đúng `flow.validate` / `flow.show` / `flow.run` / `flow.kho` đã có. |
| ba trường trong show/chạy khan | **Không** | Làm giàu dữ liệu của `flow.show` và `flow.kho` đã có. |

Bốn mặt của `flow.kho` sau bản này:

| Mặt | Trạng thái |
|---|---|
| hợp đồng `api.Actions` | không đổi (đúng — không có action mới) |
| lệnh CLI | ✅ `sagent flow show`, `sagent flow run --kho` đã in đủ (dán ở trên) |
| endpoint HTTP | ✅ `POST /api/flow/kho` `writeJSON(kh)` nguyên khối struct, nên 6 trường JSON mới (`goLaiCho`, `compensate`, `artifact`, `idempotent`, `route`, `gop`, `soBuocGoLai`) đi ra **tự động** |
| web-UI | ⚠ **CHƯA VẼ** — dữ liệu tới nơi, nhưng `internal/dash/web/flow.html` chưa render. **Nằm ngoài vùng của tôi lượt này** (agent khác đang ở `internal/dash/*`). |

Nói thẳng: mặt thứ tư là **chỗ hở duy nhất** của lượt này, và nó hở vì ranh giới
file chứ không vì quên. Việc phải làm nằm ở mục 3 dưới.

---

## 2. Sự cố

1. **`main` chạy trước tôi 2 commit trong lúc tôi làm.** `git rev-list --count
   HEAD..main` = 2 (`9a52526`, `d7c0653`). Nhánh tôi vẫn sạch so với gốc của nó
   (`main..HEAD` = 0 lúc tạo). **Không đụng vào `main`.** Đáng chú ý:
   `d7c0653 "Do ba manh flow tren luot chay THAT qua sagent flow run"` đụng cùng
   khu vực với VIỆC 3 — **khả năng xung đột lúc trộn**, xem mục 3.

2. **Bài test có sẵn đỏ vì fixture cũ, không phải vì hành vi sai.**
   `TestCanhBaoTypeChuaChayDuoc` mượn `TypeMerge` làm ví dụ "loại chưa chạy
   được"; bật merge/route xong thì không còn loại nào như thế. Đã sửa để nó dựng
   loại giả (chi tiết ở mục 1). Đây là bằng chứng sống cho một bài học: bài kiểm
   CƠ CHẾ mà buộc vào một node cụ thể sẽ đỏ vào đúng ngày node đó chạy được.

3. **`gofmt -l` ồn, nhưng KHÔNG phải lỗi của lượt này.** Nó liệt ~70 file kể cả
   những file tôi không đụng. Đã đo bằng `git stash -u` rồi chạy lại: danh sách
   y nguyên khi chưa có thay đổi nào của tôi. Mọi file **mới/sửa** của lượt này
   đều sạch `gofmt`.

4. **Một lời gọi python heredoc thất bại im lặng** (`str.replace` không khớp,
   không assert) làm mất một khối sửa ở `flow_run.go`; phát hiện khi đọc lại file
   và đã vá bằng công cụ Edit. Bài học: mọi phép thay chuỗi phải có `assert`.

5. **Chạy installer thật có rủi ro đã lường trước.** `cai-dat.ps1` ghi cứng
   `$Bin = $env:USERPROFILE\bin` và **tự thêm PATH khi stdin bị chuyển hướng**
   (mặc định `$tra = 'c'`). Đã đo bằng cách đổi `USERPROFILE` sang thư mục tạm,
   chụp `User PATH` trước, và **khôi phục nguyên trạng** sau — đã xác nhận PATH
   về đúng như cũ và thư mục tạm đã xoá. `sagent.exe` thật trên máy **không bị
   đụng**.

6. **Chưa gọi API thật.** Node `route` được kiểm bằng phần cắm giả
   (`routeGia`/`modelGia`) chứ chưa chạy qua `aiapi.Kiem` với key thật. Lý do:
   `internal/aiapi/*` nằm trong danh sách cấm lượt này, và một lần đo thật ở đó
   cần key + mạng. Cầu `routeBridge` **chỉ gọi lại** những hàm đã có (`AIRoutes`,
   `ThuTuRoute`, `aiapi.Kiem`) nên không có logic mới nào chưa được đo — nhưng
   nói thẳng: **chuỗi đầu-cuối tới nhà cung cấp thật chưa đo lượt này**.

---

## 3. Bước tiếp theo

Xếp theo mức nguy hiểm nếu bỏ qua:

1. **Vẽ 6 trường mới lên `internal/dash/web/flow.html`** — mặt thứ tư còn hở.
   Quan trọng nhất là `goLaiCho`: mặt web đang hiện bước gỡ lại như một bước sắp
   chạy, **đúng cái lỗi vừa vá ở CLI**. Dữ liệu đã sẵn trong response của
   `POST /api/flow/kho`. Ước lượng: nhỏ, chỉ là render.
2. **Trộn nhánh này vào `main` và xử lý xung đột với `d7c0653`** — cả hai đụng
   `internal/api/chaykho.go` và `internal/flow/step.go`. Trộn sớm, đừng để lệch
   thêm.
3. **Đo node `route` đầu-cuối với một route thật** (`sagent api ds` có gì thì
   dùng cái đó): dựng flow hai bước `route` → `model`, chạy `--kho` rồi chạy
   thật, và khẳng định sổ `api_calls` ghi đúng route mà node đã chọn.
4. **Cập nhật `docs/MASTER-PLAN.md`**: hai dòng "⬜ `merge`" và "⬜ `route`"
   (`:660`, `:663`) nay đã xong — Pha 3 node built-in từ **8/10** lên **10/10**.
   Không tự sửa lượt này vì file nằm trong danh sách cấm.
5. **Cân nhắc đưa `-Go` thành một lệnh gõ được** (`sagent go-cai-dat`), vì hiện
   tại muốn gỡ thì phải có sẵn repo hoặc tải lại `cai-dat.ps1`. Nếu làm thì **đó
   LÀ một action mới** và phải đủ bốn mặt.
6. **`gofmt` cả kho một lượt** — 70 file bẩn sẵn làm `gofmt -l` mất hết giá trị
   làm cổng kiểm. Nên làm thành một commit riêng, không trộn với thay đổi hành vi.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Khảo sát mã: installer, `cmd/sagent`, `internal/flow`, `chaykho.go` | Opus 5 (1M) | Trung bình |
| VIỆC 1 — chọn cách làm alias (cân 3 phương án), viết shim + `-Go` | Opus 5 (1M) | Trung bình |
| VIỆC 1 — test đọc thẳng `.ps1` rồi chạy shim thật; chạy installer thật | Opus 5 (1M) | Cao |
| VIỆC 2 — node `route` + cầu `routeBridge` (dùng lại `aiapi`) | Opus 5 (1M) | Trung bình |
| VIỆC 2 — node `merge`: hai câu khó (thứ tự gộp, nguồn hỏng) | Opus 5 (1M) | **Cao** |
| VIỆC 2 — test xác định: hai lượt xong ngược thứ tự ra giống nhau | Opus 5 (1M) | Cao |
| VIỆC 3 — `flow show` + chạy khan hiện ba trường, đánh dấu bước gỡ lại | Opus 5 (1M) | Trung bình |
| Chứng minh test ĐỎ khi gỡ phần sửa (3 lần, 3 chỗ) | Opus 5 (1M) | Thấp |
| Nghiệm thu `build`/`vet`/`test` + viết báo cáo | Opus 5 (1M) | Thấp |

Không dùng subagent, không dùng workflow — cả lượt chạy trong một ngữ cảnh.

---

## 5. Nhận xét tự do

**Lý do hoãn hết hạn mà không ai gỡ xuống là một lớp nợ riêng.** Mục alias treo
vì "làm khi viết installer phát hành"; installer viết xong từ Pha 7 và mục vẫn
treo — vì **không có gì đỏ lên khi nó thiếu**. Đó là lý do tôi không chỉ viết
shim mà viết một bài test **đọc thẳng `cai-dat.ps1`**: cả tính năng nằm ngoài mã
Go, nên `go build` không thấy, `go vet` không thấy, và người sửa installer lần
sau cũng sẽ không thấy. Một điều kiện hoãn nên có ngày hết hạn **kiểm được bằng
máy**, không chỉ một câu trong kế hoạch.

**`merge` treo hai tháng vì một hiểu lầm về tên, không vì kỹ thuật.** Ghi chú
"còn chờ cơ chế merge an toàn" ngầm hiểu merge = `git merge`, và cơ chế an toàn
cho việc đó **không tồn tại và sẽ không tới**. Một node treo vô hạn vì lời hẹn
của nó không bao giờ đến hạn được. Bài học rộng hơn: khi một mục treo lâu, đáng
hỏi "định nghĩa của nó có sai không" trước khi hỏi "khi nào làm được".

**Câu hỏi khó của `merge` không phải câu hỏi kỹ thuật.** Nối chữ thì dễ; chọn
**thứ tự nào** mới là chỗ quyết định node này đúng hay sai. Và cái sai của "thứ
tự xong" nguy hiểm ở chỗ nó **chạy xanh cả hai lượt** — chỉ có bước sau nhận hai
đầu vào khác nhau, và không ai nhìn ra bằng mắt. Đó là lý do bài test quan trọng
nhất của mảnh này không kiểm nội dung mà kiểm **tính lặp lại**, và tự khẳng định
luôn rằng nó có kiểm được cái nó nói (hai lượt phải xong khác thứ tự thật).

**Cùng một cách sai xuất hiện ba lần trong lượt này, và luôn theo hướng "im
lặng".** Alias bằng bản sao → chạy nhầm phiên bản, không ai báo. `route` không
qua `Expand` → node chạy đúng mà vô dụng, không ai báo. Bước gỡ lại trong bảng
chạy khan → kế hoạch nói thừa một bước, không ai báo. Ba chỗ khác hẳn nhau về
mã, giống hệt nhau về hình dạng lỗi: **không sập, chỉ lặng lẽ nói sai**. Dự án
này đã đặt tên cho nó nhiều lần (lượt #29, #46) — nó vẫn quay lại, và có lẽ cách
duy nhất chống được là mỗi lần thêm một đường truyền thì hỏi thẳng "nếu đường
này hỏng, có gì đỏ lên không".

**Ranh giới file là một ràng buộc thật, và nó để lại một vết hở nhìn thấy được.**
Web-UI không vẽ 6 trường mới không phải vì tôi quên mà vì `internal/dash/*` của
agent khác. Tôi thấy nên ghi lại: khi chia việc theo file, cái hở sẽ luôn rơi
đúng vào **mặt cuối cùng của chuỗi** — mặt người dùng nhìn. Lần sau nên chia
theo **luồng dữ liệu** (ai làm `flow.kho` thì làm cả bốn mặt của nó) thay vì
theo thư mục.
