# Route theo năng lực — mảnh cuối của engine flow

Nhánh `sagent/route-nangluc-22-08`, dựng sạch từ `main` (`git rev-list --count
main..HEAD` = **0** lúc bắt đầu).

---

## 1. Đã làm

### Sơ đồ — đường đi của một nhu cầu năng lực

```
flows.toml                     internal/flow                internal/api                internal/aiapi
──────────                     ────────────                 ────────────                ──────────────
[[flow.x.step]]
  id     = "chon"
  type   = "route"
  routes = ["deepseek","grok"]  ──▶ Step.Routes
  can    = ["dau-vao-anh"]      ──▶ Step.Can  ─┐
                                              │
                                   VanDeCan   │   (soi HÌNH DẠNG: khai nhầm chỗ,
                                   nangluc.go │    trùng, rỗng, bước model đòi thứ
                                              │    bước route không lọc)
                                              │
                                              ├─▶ VanDeCanTheoBang ──▶ MoiNangLucAPI
                                              │   flow_nangluc.go       (soi KHOÁ có thật)
                                              │                    ──▶ BangNangLuc
                                              │                         (route đã khai
                                              │                          có làm được không)
                                              │        ▲
                                              │        └── chạy lúc `flow validate`
                                              │            và `flow show`: TRƯỚC lượt
                                              │            chạy, TRƯỚC đồng token đầu tiên
                                              │
                                   ┌──────────┘
                                   ▼
                          RouteChon.ChonRoute(ctx, ungVien, can)
                                   │
                                   └──▶ routeBridge.ChonRoute ──▶ locNangLuc ──▶ BangNangLuc
                                            (internal/api)          │              (MIỄN PHÍ,
                                                                    │               không chạm mạng)
                                                     ┌──────────────┼──────────────┐
                                                     ▼              ▼              ▼
                                                   hạng ĐỦ      hạng CHƯA RÕ    hạng LOẠI
                                                     │              │          (không bao giờ
                                                  Kiem()         Kiem()          hỏi thăm)
                                                (GET /models,  (chọn kèm lời
                                                 0 token)       nói ra)
                                                     │              │
                                                     └──────┬───────┘
                                                            ▼
                                             output của bước = ĐÚNG tên route
                                                            │
                                                            ▼
                                          route = "{{steps.chon.output}}" của bước `model`
```

### Ba câu hỏi, ba câu trả lời

#### 1. CHỌN THEO NĂNG LỰC — nhu cầu khai ở đâu?

**Chọn KHAI TAY trong `flows.toml`**: `can = ["goi-tool", "dau-vao-anh"]`.
Khoá lấy thẳng từ `aiapi.MoiNangLucAPI` — cùng từ vựng với `sagent
nang-luc-api`, không có bảng thứ hai ở đâu.

Hai hướng đã cân nhắc, và hậu quả của chúng **không đối xứng**:

| | Suy từ nội dung bước | Khai tay |
|---|---|---|
| Đoán/quên sai thì sao | **LOẠI một route đang sống** và đang làm được việc, hoặc dừng hẳn | Bước chạy **y hệt hiện trạng**: chọn theo sức khoẻ |
| Người dùng cãi lại được không | Không — họ chưa từng khai gì để mà sửa | Có — sửa đúng dòng họ viết |
| Đổi một chữ trong prompt | **Đổi đường đi**, mà bảng vẫn ghi "model" | Không đổi gì |

Nói thẳng cái giá của hướng đã chọn: **thêm một chỗ người dùng quên**. Giá đó
được trả bằng ba lời soi, không bằng một lời hứa:

- `can` khai ở bước không phải `route`/`model` → **LỖI** (dòng chết, cùng lớp
  với `routes` khai nhầm chỗ);
- khoá gõ nhầm → **LỖI**, kèm cả bảy khoá hợp lệ — đo thật:

  ```
  ✗ sai.hoi   `can` khai "goi-tools" — không có năng lực nào tên vậy. Bảy khoá
              hợp lệ: goi-tool, dau-vao-anh, dau-ra-co-cau-truc, reasoning,
              streaming, dem-token-that, liet-ke-model (xem: sagent nang-luc-api)
  ```

  Khoá lạ phải là **lỗi cứng** chứ không phải cảnh báo: bộ lọc không biết loại
  ai nên nó **không lọc gì**, trong khi người viết flow tin rằng bước của mình
  đang được canh. Một hàng rào không chặn gì tệ hơn không có hàng rào.
- bước `model` đòi năng lực mà bước **chọn đường** của nó không lọc theo → cảnh
  báo, **in ra đúng dòng cần thêm** (không phải một lời nhắc chung chung — người
  đọc đang ở giữa một file TOML, không đang đọc tài liệu):

  ```
  ! sai.hoi   bước này đòi `dau-vao-anh`, `goi-tools`, nhưng bước chọn đường
              "chon" KHÔNG lọc theo năng lực nào — nó sẽ chọn đường chỉ theo sức
              khoẻ, và có thể trả về một đường không làm được việc.
              Thêm vào bước "chon": can = ["dau-vao-anh", "goi-tools"]
  ```

**Thứ tự hai phép lọc là một quyết định**: năng lực (đọc bảng, miễn phí, tại
chỗ) chạy **trước** sức khoẻ (`GET /models`, một lượt đi mạng **cho mỗi** ứng
viên). Ngược lại là trả tiền bằng thời gian cho một câu trả lời đã nằm sẵn
trong repo. Có bài kiểm canh đúng chiều này.

#### 2. CHỌN THEO GIÁ — **KHÔNG ĐỦ DỮ LIỆU. CHỈ LÀM PHẦN NĂNG LỰC.**

Không suy diễn, đây là số đo trên `~/.ai-accounts/state.db` thật (đọc chế độ
chỉ-đọc, `immutable=1`, không chạm sổ):

| Đo gì | Kết quả |
|---|---|
| Số dòng trong `api_calls` | **21** |
| Trong đó `cost_usd != 0` | **0** |
| Tổng token đã đi qua đường API | **8.942 vào / 19.787 ra** |
| Số bảng giá trong toàn repo (`grep` `price`/`don_gia`/`per_token`/`bang_gia`) | **0** |

Nghĩa là: **28.729 token đã đi qua đường API mà không một đồng nào được gắn vào
chúng.** `store.GoiAPI.CostUSD` tự nói ra lý do — endpoint `chat/completions`
không trả giá, và `ghiSoAPI` (`internal/api/api.go:978`) chưa bao giờ gán trường
đó.

Vậy hai nguồn giá đã cân nhắc đều **không dùng được**:

- **Tính ngược từ `api_calls`**: không tính ngược được từ 0. Đây không phải "dữ
  liệu thưa", là **không có dữ liệu**. Một bộ chọn theo giá dựng trên cột này sẽ
  thấy mọi route giá bằng nhau và bằng không — tức là nó sẽ chọn theo thứ tự
  danh sách trong khi khai với người dùng rằng nó đang chọn theo giá.
- **Khai tay trong `project.toml`**: khai được, nhưng đây là **lời khai không
  có gì đối chiếu**. Dự án đã có đúng một tiền lệ cho việc này và nó nổ ngày
  22/08: bảng quyền plugin khai `chan-that` cho một thứ không chặn được, sống
  lâu được vì không ai đòi bằng chứng. Bảng năng lực sinh ra sau đó với luật
  "mỗi ô phải là đầu ra của một lượt đo thật". Một bảng giá khai tay đi **ngược
  lại** đúng luật ấy — và hậu quả nặng hơn: giá sai/cũ dẫn tới **chọn sai route
  mà lượt chạy vẫn xanh**, không ai biết, cho tới lúc đọc hoá đơn.

*(Có một con số dễ đọc nhầm: `flow_steps` có **52 bước** với tổng
**122,99 USD**. Đó là tiền của bước **`agent`** — CLI của nhà cung cấp tự khai
`total_cost_usd` cho cả phiên. Nó **không** quy ra được đơn giá theo token của
một route API, và `modelBridge.GoiModel` không hề gán `ChiPhiUSD`, nên mọi bước
`model` vẫn ghi 0.)*

**Cách chặn, nếu sau này vẫn làm phần giá** (chưa làm, ghi ra để người sau không
phải nghĩ lại): bảng giá khai tay phải mang **ngày khai** và **nguồn** y như
`aiapi.soDoNangLuc`, và phải có một phép **đối chiếu** — mỗi lượt gọi lấy
`usage` thật nhân đơn giá đã khai, so với hoá đơn/`api_calls`; lệch quá ngưỡng
thì bảng tự chuyển ô đó về `ChuaDo` thay vì im lặng dùng tiếp. Không có phép đối
chiếu đó thì đừng làm — làm một nửa cho thật hơn hẳn hai nửa cho đoán.

#### 3. KHÔNG CÓ ROUTE NÀO ĐỦ NĂNG LỰC — **DỪNG HẲN**, kèm câu nói rõ

Nhưng **ba hạng, không phải hai** — và đó chính là lý do bảng năng lực có ba
trạng thái chứ không phải hai:

| Hạng | Điều kiện | Xử |
|---|---|---|
| **ĐỦ** | mọi khoá `can` **đã đo** và làm được | ưu tiên tuyệt đối; hỏi sức khoẻ theo thứ tự |
| **CHƯA RÕ** | không khoá nào bị đo là KHÔNG, nhưng có khoá **chưa ai đo** | vẫn chạy được — nhưng chỉ khi hạng ĐỦ đã hết, và **phải nói ra** |
| **LOẠI** | có khoá **đã đo được** là không làm được | không bao giờ chọn, và **không hỏi thăm sức khoẻ** |

Vì sao **DỪNG** cho hạng LOẠI chứ không "chạy bằng route tốt nhất rồi cảnh báo":
chạy tiếp nghĩa là gọi thật, tiêu token, rồi nhận về một câu trả lời **không có
tool call** — thứ trông y hệt một câu trả lời bình thường. Đó đúng hình dạng
hỏng mà `phai_co` được dựng ra để chặn (lượt #46): *cổng kiểm không sập, nó chỉ
lặng lẽ gật đầu*. Và bảng **đã biết** câu trả lời trước cả lượt đi mạng đầu
tiên — để nó biết mà không dùng là lãng phí đắt nhất của cả đường API.

Vì sao **KHÔNG dừng** cho hạng CHƯA RÕ: gộp nó vào LOẠI là bẹp ba trạng thái
thành hai, đúng cái sai mà `internal/aiapi/nangluc.go` dựng lên để chặn. Một dự
án vừa thêm route thứ ba sẽ thấy nó bị loại thẳng chỉ vì chưa chạy
`nang-luc-api --do`, và lỗi hiện ra chẳng nhắc gì tới phép đo. Câu chữ lúc xuống
hạng lấy gần như nguyên của `aiapi.SoatTruocKhiGui`, để hai chỗ không dạy người
dùng hai điều khác nhau.

**Cố ý KHÔNG thêm cờ ép chọn** (kiểu `--cu-gui`). Đường thoát đã có sẵn và trung
thực hơn: **bỏ khoá đó khỏi `can`** — tức là khai rằng bạn không còn đòi năng
lực ấy. Một cờ nghĩa là *"tôi cần X, hãy chọn một đường không làm được X"* là vô
nghĩa với một **bộ chọn**. `--cu-gui` hợp lý ở `SoatTruocKhiGui` vì ở đó bạn
muốn thử lại thực tế khi nghi số đo đã cũ; với bộ chọn thì việc phải làm là
`sagent nang-luc-api --do`, và lời từ chối nói đúng câu đó. Bớt được một cờ là
bớt một thứ phải cắm đủ bốn mặt.

### Bốn mặt

Mảnh này **không đẻ ra action mới** — có chủ ý, và đây là phần đã kiểm:

| Thứ mới | Đi ra bằng đường nào | Trạng thái |
|---|---|---|
| Trường `can` trong flows.toml | `Step.Can` có thẻ `json:"can,omitempty"` → `/api/flow/def` trả `f.Steps` nguyên vẹn | **đủ** (web nhận tự động) |
| Lời soi `can` | action **`flow.validate`** đã có sẵn trong `api.Actions` | **đủ ở CLI**, xem sự cố #2 |
| `can` khi xem flow | `sagent flow show` — in trong dòng "chọn đường" (node `route`) và dòng "cần đường làm được" (node `model`) | **đủ** |
| Bảng năng lực bộ chọn tra | `api.nang-luc` → `/api/nang-luc-api` + `sagent nang-luc-api` | đã có từ trước |

`MoTaRoute` nay **mang theo `can`** thay vì giấu nó: một bảng in
`grok → deepseek` mà không nói `can = ["dau-vao-anh"]` đang **nói sai** —
deepseek đã bị loại trước khi ai hỏi thăm nó. Có bài kiểm canh.

`sagent flow show` trước đây chỉ chạy `flow.Validate` (nửa hình dạng), nên nó
**im lặng** về khoá gõ nhầm trong khi `flow validate` báo lỗi. Đã nối nốt nửa
tra bảng vào (`cmd/sagent/flow.go`) — hai lệnh giờ nói giống nhau về cùng một
file, đo lại xác nhận.

### Đo thật — 0 token, 0 đồng

Theo khuôn `docs/DO-LUONG.md` mục *"22/08 — Ba mảnh flow chạy thật qua
`sagent flow run`"*: dự án tạm, flow chỉ gồm bước `route` + `shell`.

**Sổ trạng thái được cách ly**: sổ thật trên đĩa là **schema v9**, mã nhánh này
có **12 migration** — chạy thẳng là nâng v9→v12 và giết dash của agent song
song. Nên toàn bộ phép đo chạy với `USERPROFILE`/`HOME` trỏ vào một `home` tạm.
Kiểm lại sau khi đo: sổ thật **vẫn v9, vẫn 52 lượt chạy**.

**Lượt #1 — năng lực thắng thứ tự ưu tiên** (`routes = ["deepseek","grok"]`,
`can = ["dau-vao-anh"]`, deepseek đứng **đầu**):

```
· chạy [route] lần 1/1
· chon.route: deepseek: KHÔNG làm được — dau-vao-anh: đo 22/08: HTTP 400
  "This model does not support image" với ảnh PNG 32x32 (1024 điểm ảnh)
· chon.route: grok: dùng được (model grok-4.5), đã đo là làm được `dau-vao-anh` — CHỌN
· xong
✓ Lần chạy #1 đã xong.
```

**Lượt #2 — không đường nào đủ năng lực** (`routes = ["deepseek"]`):

```
✗ bitac.chon: không route nào trong 1 ứng viên làm được `dau-vao-anh` — DỪNG ở
  đây, CHƯA gọi đi đâu cả: bảng năng lực đã biết câu trả lời trước khi gọi, và
  một lượt hỏng vẫn có thể bị tính tiền.
     deepseek: KHÔNG làm được — dau-vao-anh: đo 22/08: HTTP 400 "This model
     does not support image" với ảnh PNG 32x32 (1024 điểm ảnh)
     Sửa: bỏ khoá đó khỏi `can`, thêm một route làm được vào `routes`, hoặc —
     nếu số đo đã cũ — đo lại: sagent nang-luc-api --do deepseek
```

**Giá của cả hai lượt**: `sagent api lich-su` → *"Sổ lời gọi API còn trống"*.
**0 lượt gọi model, 0 token, 0 đồng.** Bước `route` chỉ dùng `GET /models`, và
lượt #2 thì **không chạm mạng một lần nào** — bảng đã trả lời xong trước đó.

**`sagent flow validate` bắt được ba lỗi trước khi chạy** (cũng 0 token): khoá
gõ nhầm, bước `model` đòi thứ bước chọn đường không lọc, và route khai cứng đã
đo được là không làm được.

### Bài kiểm đi HẾT ĐƯỜNG, và điểm mù nó nhắm vào

Bảng năng lực đo phía dự án bằng **reflection trên kiểu thật**, nên nó trả lời
được *"kiểu có trường đó không"* mà **không** trả lời được *"có ai chép giá trị
đi không"*. Ngày 22/08 dự án bắt được đúng một ca như vậy: `tinNhan.SuyLuan` và
`KetQua.SuyLuan` đều có mặt, ô `reasoning` in ✓, mà cái cầu ở `internal/api` thì
**vứt** phần nghĩ.

Trường `can` có đủ điều kiện lặp lại cái hỏng đó — nó đi qua **bốn mắt xích**,
ba mắt đầu đứt được trong **im lặng**. Nên
`TestCanDiHetDuongTuTOMLToiBoChonRoute` (dựng theo mẫu
`TestSuyLuanDiHetDuongToiKetQua`) chạy **một lượt thật qua `Runner.Start`**, đọc
flow từ **đúng một file `flows.toml` trên đĩa**, và khẳng định cả bốn:

| Mắt xích | Đứt thì hỏng thế nào |
|---|---|
| chữ trong flows.toml → `Step.Can` | thẻ `toml` sai một chữ = BurntSushi bỏ qua im lặng, `can` về nil |
| `Step.Can` → tham số `can` của `ChonRoute` | **mắt xích mới**: quên truyền thì mọi thứ vẫn xanh, chỉ là lọc rỗng |
| `ChonRoute` → output của bước | (đã có bài kiểm cũ) |
| output → `route` của bước `model` | (đã có bài kiểm cũ) |

Và `TestFlowValidateChayCaHaiNuaPhepSoiCan` đi từ chữ trong `flows.toml` qua
**đúng `API.FlowValidate`** — hàm mà action `flow.validate` gọi — chứ không gọi
thẳng hàm con. Một hàm đúng mà không ai gọi là đúng hình dạng lỗi *"có ở mọi
tầng trừ tầng cuối"* dự án đã vấp sáu lần trong một ngày.

Phép lọc năng lực được kiểm bằng **sổ số đo THẬT** (`grok-4.5` /
`deepseek-v4-flash` @ modelapi.vn), không bằng một bảng dựng cho vừa bài kiểm —
nên bài kiểm còn canh thêm rằng đường `Route → BangNangLuc → TrangThai` vẫn
thông.

**Bốn phép đột biến — gỡ phần sửa ra thì test đỏ** (đã chạy thật, rồi khôi phục):

| Gỡ gì | Test đỏ |
|---|---|
| `step.go` truyền `nil` thay vì `s.Can` | `TestCanDiHetDuongTuTOMLToiBoChonRoute` |
| bẹp hạng CHƯA RÕ vào hạng ĐỦ | `TestChuaDoLaHangRiengKhongPhaiLoai`, `TestKhoaLaKhongDuocCoiLaDuNangLuc`, `TestValidateChuaDoChiLaCanhBao` |
| bỏ `VanDeCanTheoBang` khỏi `FlowValidate` | `TestFlowValidateChayCaHaiNuaPhepSoiCan` |
| bỏ `VanDeCan` khỏi `flow.Validate` | `TestFlowValidateChayCaHaiNuaPhepSoiCan` |

### Nghiệm thu

Chạy **từng lệnh riêng biệt**, kiểm bằng mã thoát:

| Lệnh | Mã thoát |
|---|---|
| `go build ./...` | **0** |
| `go vet ./...` | **0** |
| `go test ./...` | **0** (26 gói) |

### File đã chạm — toàn bộ trong vùng được giao

| File | Việc |
|---|---|
| `internal/flow/flow.go` | thêm `Step.Can`; nối `VanDeCan` vào `Validate` |
| `internal/flow/nangluc.go` | **mới** — `VanDeCan`, `MoTaCan`; ghi cả hai hướng đã cân nhắc và hậu quả |
| `internal/flow/route.go` | `RouteChon.ChonRoute` nhận thêm `can`; `MoTaRoute` mang theo `can` |
| `internal/flow/step.go` | truyền `s.Can` xuống bộ chọn |
| `internal/api/routenode.go` | bộ chọn ba hạng: `locNangLuc` (thuần) + `ChonRoute` |
| `internal/api/flow_nangluc.go` | **mới** — `VanDeCanTheoBang`: soi khoá + tra bảng |
| `internal/api/api.go` | `FlowValidate` chạy cả hai nửa phép soi |
| `cmd/sagent/flow.go` | `flow show` in `can` và chạy cả hai nửa phép soi |
| `internal/flow/nangluc_test.go`, `internal/api/flow_nangluc_test.go` | **mới** — 22 bài kiểm |
| `internal/flow/routenode_test.go` | phần cắm giả ghi lại `can` nó nhận được |

---

## 2. Sự cố

**a. Phần GIÁ không làm được — đã nói ở mục 1.2.** Không phải bỏ qua, mà là đo
xong rồi mới bỏ: `api_calls` có 21 dòng, **0 dòng có giá**; repo có **0 bảng
giá**. Làm một nửa cho thật hơn hẳn hai nửa cho đoán.

**b. Lời soi của `flow.validate` KHÔNG đi ra được mặt web** — không phải lỗi
mảnh này gây ra, nhưng mảnh này làm nó đắt hơn hẳn.

`internal/dash/lachan_test.go:162` ánh xạ `flow.validate → /api/flow/def`, nhưng
`handleFlowDef` (`internal/dash/flow_api.go:327`, khối `writeJSON` ở dòng **339**) chỉ trả `steps/vars/desc` —
**không gọi `FlowValidate`, không trả `Problem` nào**. Nghĩa là trường `can` thì
mặt web thấy (nó nằm trong `f.Steps`), còn lời chặn *"route này đã đo được là
không đọc được ảnh"* thì **không**. Đúng hình dạng *"có ở mọi tầng trừ tầng
cuối"*.

`internal/dash/*` **ngoài vùng của tôi**. Dòng cần cắm, ghi đích danh cho người
điều phối:

- `internal/dash/flow_api.go`, trong `handleFlowDef`, thêm vào `writeJSON`:
  ```go
  "problems": func() []flow.Problem { ps, _ := s.api.FlowValidate(s.workDir()); return ps }(),
  ```
  (hoặc gọn hơn: tính `ps` trước rồi thêm khoá `"problems": ps`)
- và mặt web vẽ nó ra — nếu không thì lại đúng một tầng nữa bị hụt.

**c. `sagent flow run` thoát mã 0 cho một lượt chạy HỎNG.** Bắt được trong lúc
đo: lượt #2 dừng vì không route nào đủ năng lực, CLI in `✗ Lần chạy #2 hỏng`
nhưng `$LASTEXITCODE` = **0**. Chỗ hỏng: `cmd/sagent/flow_run.go:296-297` —
nhánh `case "failed"` chỉ `fmt.Printf`, không đặt mã thoát. Có sẵn tiền lệ đúng
ngay cạnh: `flowValidate()` đếm lỗi rồi thoát khác 0.

Đây là file **trong** vùng của tôi nhưng **ngoài** phạm vi việc được giao, và
sửa nó là đổi mã thoát của một lệnh mà CI/script khác có thể đang dựa vào — nên
tôi **không tự sửa**. Hậu quả nếu để: mọi lượt chạy hỏng đều "thành công" dưới
mắt một script bao ngoài.

**d. Bẫy đã gặp, ghi lại để khỏi mất giờ lần sau.**

- `[[ai.routes]]` trong `project.toml` là **sai** — thẻ TOML là
  `route` (số ít), nên phải viết `[[ai.route]]`. Khai sai thì cấu hình nạp
  **rỗng trong im lặng** và mọi lời soi năng lực chuyển thành *"dự án CHƯA cấu
  hình route API nào"* — một câu đúng ngữ pháp và sai hoàn toàn về nguyên nhân.
- Heredoc `<<'EOF'` của Bash trên máy này **vẫn nuốt một lớp `\`** (đúng như sổ
  tay đã ghi), nên `\\n` trong chuỗi Python trở thành ký tự xuống dòng thật và
  phép `replace` trượt không báo gì. Dùng `chr(92)` để ghép, hoặc ghi file rồi
  chắp.
- Heredoc còn vỡ khi nội dung có dấu nháy/backtick dày — khối Go dài thì ghi
  thẳng bằng công cụ ghi file, đừng qua shell.

---

## 3. Bước tiếp theo

Xếp theo thứ tự giá trị trên mỗi giờ bỏ ra:

1. **Cắm lời soi vào mặt web** — mục 2.b, hai dòng, và nó đóng một lỗ đã há sẵn
   từ trước mảnh này. Người điều phối cắm, vì `internal/dash/*` không phải vùng
   tôi.
2. **`sagent flow run` phải thoát khác 0 khi lượt chạy hỏng** — mục 2.c. Một
   dòng, nhưng cần người quyết vì nó đổi hợp đồng của một lệnh.
3. **Đo nốt hai ô `ChuaDo`** để hạng CHƯA RÕ bớt đường phải đi: hiện cả 14 ô của
   hai route đều đã đo, nhưng route thứ ba nào thêm vào cũng vào thẳng hạng CHƯA
   RÕ. `sagent nang-luc-api --do <route>` — có tốn token, nên là việc người dùng
   bấm, không phải việc agent tự chạy.
4. **Phần GIÁ** — chỉ làm khi có phép đối chiếu như mô tả ở mục 1.2. Trước đó
   thì một bảng giá khai tay là một lời khai nữa không ai đòi bằng chứng, và dự
   án đã có đúng một cái như thế nổ trong ngày 22/08.
5. **`can` cho node `agent`?** Chưa làm, và nghiêng về **không**: bảng năng lực
   nửa CLI (`provider.nang-luc`) trả lời câu khác hẳn, và trộn hai từ vựng vào
   một trường là bước đầu của việc hai bảng nói ngược nhau.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Đọc hiện trạng: `route.go`, `nangluc.go`, `goikem.go`, hợp đồng `Actions`, bốn mặt | Opus 5 (1M) | trung bình — nhiều file dài, nhưng đọc là chính |
| Quyết hướng khai nhu cầu (khai tay vs suy từ nội dung) | Opus 5 (1M) | **cao** — quyết định không lùi lại được, hậu quả bất đối xứng |
| Quyết ba hạng thay vì hai (giữ `ChuaDo` riêng) | Opus 5 (1M) | **cao** — đây là chỗ dễ bẹp nhất, và bẹp thì không ai thấy |
| Đo phần giá rồi kết luận **không đủ dữ liệu** | Opus 5 (1M) | thấp — một truy vấn SQL chỉ-đọc là xong |
| Viết `locNangLuc` + `ChonRoute` ba hạng | Opus 5 (1M) | trung bình |
| Viết `VanDeCan` + `VanDeCanTheoBang` (chia đôi theo thứ mỗi gói biết) | Opus 5 (1M) | trung bình |
| Viết bài kiểm **đi hết đường** (4 mắt xích, từ file TOML) | Opus 5 (1M) | **cao** — đây là phần nhắm thẳng vào điểm mù đã biết |
| Bốn phép đột biến để chứng minh test đỏ được | Opus 5 (1M) | trung bình |
| Đo thật trong sandbox (né nâng schema v9→v12) | Opus 5 (1M) | trung bình — phần né sổ là phần phải cẩn thận |
| Viết báo cáo | Opus 5 (1M) | trung bình |

---

## 5. Nhận xét tự do

**Thứ đắt nhất tôi làm hôm nay là một phép trừ, không phải một phép cộng.** Việc
được giao gồm hai nửa; nửa giá đo ra là **0 dòng dữ liệu, 0 bảng giá**. Viết một
bộ chọn theo giá trên nền đó thì nó vẫn chạy, vẫn xanh, vẫn in ra những câu tự
tin — và mọi route sẽ có giá bằng nhau và bằng không, nên nó thực chất chọn theo
thứ tự danh sách trong khi khai với người dùng rằng nó chọn theo giá. Đó không
phải một tính năng chưa hoàn thiện, đó là **một lời nói dối có mã nguồn**. Dự án
này đã bắt được đúng một thứ như vậy hôm 22/08 (bảng quyền plugin khai
`chan-that` cho thứ không chặn được), và bài học ghi lại là *"một bảng khai bừa
tệ hơn không có bảng, vì người vận hành sẽ TIN nó"*.

**Chỗ dễ sai nhất của mảnh này không phải phần lọc, mà là số hạng.** Hai hạng —
"được" và "không được" — viết nhanh hơn, đọc dễ hơn, và **sai theo cách không ai
phát hiện ra**. Route chưa đo bị loại thẳng thì người dùng chỉ thấy "không đường
nào dùng được" và đi kiểm mạng; route chưa đo được coi là đủ thì lượt chạy hỏng
ở bước sau và bảng vẫn xanh. Cả gói `internal/aiapi/nangluc.go` được dựng lên để
chặn đúng cái bẹp này ở tầng bảng — nếu tầng **dùng** bảng lại bẹp nó lần nữa
thì công của tầng dưới bằng không. Tôi để lại ba bài kiểm chỉ để canh riêng
chuyện đó, và một phép đột biến chứng minh chúng đỏ được.

**Điểm mù của bảng năng lực vẫn còn, và giờ nó có thêm một chỗ để trốn.** Bảng
hỏi *"kiểu có trường đó không"*, không hỏi *"có ai chép giá trị đi không"*.
Trường `can` đi qua bốn mắt xích và ba mắt đứt được trong im lặng — đứt ở mắt
thứ hai (quên truyền `s.Can`) thì **mọi thứ vẫn xanh**: bước chọn đường vẫn
chọn, bước model vẫn hỏi, lượt chạy vẫn `completed`, chỉ là hàng rào chưa bao
giờ tồn tại. Đó là lý do bài kiểm chính đọc flow từ **file thật trên đĩa** và
khẳng định `ChonRoute` **nhận được đúng hai khoá**, chứ không khẳng định rằng
`Step` có một trường tên là `Can`. Tôi khuyên mọi mảnh sau cũng làm vậy: bài
kiểm dừng ở kiểu là bài kiểm đo chính nó.

**Một thứ tôi cố ý KHÔNG thêm: cờ ép chọn.** Cám dỗ rất lớn — `--cu-gui` đã có
tiền lệ ngay trong `SoatTruocKhiGui`. Nhưng ở đó cờ có nghĩa: *"số đo có thể đã
cũ, cho tôi thử lại thực tế"*. Với một **bộ chọn** thì *"tôi cần X, hãy chọn
đường không làm được X"* là một câu vô nghĩa, và đường thoát trung thực đã có
sẵn: bỏ khoá khỏi `can`. Mỗi cờ không thêm là một thứ không phải cắm đủ bốn mặt,
không phải giải thích trong help, và không phải sửa khi nó lệch. Dự án này vấp
sáu lần trong một ngày vì thứ *"có ở mọi tầng trừ tầng cuối"* — cách rẻ nhất để
không vấp lần thứ bảy là **đừng đẻ thêm tầng**.

**Cuối cùng, một lời cho người sửa tiếp.** `flow show` và `flow validate` từng
nói khác nhau về cùng một file — `show` chỉ chạy nửa phép soi. Tôi phát hiện ra
không phải bằng đọc mã mà bằng **chạy thật rồi so hai màn hình**. Bộ test không
bắt được vì không bài nào so hai lệnh với nhau. Đó là hình dạng lỗi rẻ nhất để
đẻ ra và đắt nhất để tìm: hai đường đọc cùng một dữ liệu bằng hai đoạn mã khác
nhau. Cả gói `internal/aiapi` được viết với ám ảnh đó ("không chép lại một dòng
nào"), và mảnh này vẫn suýt tái phạm ở một chỗ không ai ngờ tới — cái lệnh dùng
để *xem*.
