# Bảng năng lực cho NỬA API — ô `[~]` của Pha 0

Nhánh `sagent/nangluc-api-22-08`, tách sạch từ `main` (`git rev-list --count main..HEAD` = 0 lúc tạo).
Ngày 22/08.

---

## 1. Đã làm (kèm số đo)

### 1.1 Bảng năng lực — `internal/aiapi/nangluc.go` (617 dòng)

Bảy năng lực × mỗi route đã cấu hình, giữ **đúng ba trạng thái** của nửa CLI.

Ba trạng thái KHÔNG phải bản sao: `TrangThaiNangLuc` là **type alias** sang
`provider.TrangThaiNangLuc`. Hai bản sao của một quy ước là hai bản sẽ lệch
nhau, và bản lệch bao giờ cũng là bản gộp "đã đo, KHÔNG" với "chưa ai đo" thành
một chữ `false`. Alias làm chuyện đó không xảy ra được — `aiapi.LamDuoc` và
`provider.LamDuoc` là **cùng một kiểu**.

Bảy năng lực (`MoiNangLucAPI`): `goi-tool`, `dau-vao-anh`, `dau-ra-co-cau-truc`,
`reasoning`, `streaming`, `dem-token-that`, `liet-ke-model` — đúng danh sách đề
bài gợi ý, không thêm bớt. Cố ý **không** có "vào âm thanh", "gọi song song
nhiều tool", "cache prompt": chưa chỗ nào trong dự án hỏi tới, mà một dòng chưa
ai hỏi thì mãi mãi là một ô `ChuaDo` không ai đi đóng.

### 1.2 Chỗ khác nửa CLI: MỖI Ô TRẢ LỜI HAI CÂU

Đây là quyết định thiết kế đáng nói nhất của lượt này.

```
KHÁCH — mã của DỰ ÁN NÀY có gửi/đọc được thứ đó không?  (đo bằng reflect, MIỄN PHÍ)
NCC   — NHÀ CUNG CẤP của route đó có làm được không?    (đo bằng mạng, TỐN TOKEN)
```

Trộn hai câu vào một ô là làm mất đúng thứ đáng giá nhất. Ví dụ thật đo được hôm
nay: **modelapi.vn gọi tool ngon lành, nhưng `aiapi.yeuCau` không có trường
`tools`** nên lời gọi của dự án chưa bao giờ mang tool đi. Một ô "không làm
được" trơ trọi sẽ khiến người đọc đi đổi nhà cung cấp — sai hẳn hướng, vì chỗ
hỏng nằm trong repo này.

Nên mỗi dòng mang: kết luận cuối (thứ `flow validate` đọc) + phần khách + phần
nhà cung cấp + cột `Cho` nói **vướng ở đâu** (`khach` / `nha-cung-cap` /
`ca-hai` / `chua-ro` / `khong-vuong`).

Phần KHÁCH đo bằng **reflect trên chính các kiểu của gói** chứ không phải một
bảng bool viết tay. Bảng viết tay là một lời khai nữa, và lời khai thì mục ruỗng:
hôm nay ai đó thêm `Tools` vào `yeuCau` để làm việc khác, bảng bool vẫn nói
"không gửi được" và ô đó ở lại sai cho tới khi có người tình cờ đọc. Soi kiểu
thật thì ô **tự lật ngay khi trường xuất hiện** — và lúc lật, `KiemNangLucAPI`
thấy phần NCC chưa có số đo nên chuyển sang `ChuaDo` chứ không tự khai làm được.
Đúng chiều an toàn.

### 1.3 Bộ ĐO THẬT — `internal/aiapi/donangluc.go` (493 dòng)

Bộ đo nằm **trong repo** chứ không phải một script chạy một lần rồi vứt: bằng
chứng chỉ đáng tin chừng nào có ai đó chạy lại được nó. Bảng quyền plugin khai
`chan-that` cho một thứ không chặn được sống lâu được vì **không ai đo lại được**.

Bộ đo cố ý **không** đi qua `Goi`/`GoiStream` — hai hàm đó gửi đúng những gì
`yeuCau` cho phép, mà cả câu hỏi ở đây là "nhà cung cấp làm được gì". Nó dựng
thân yêu cầu bằng `map[string]any`.

Chạy tuần tự, `max_tokens: 64` (riêng phép đo reasoning là 512 vì reasoning ăn
token đầu ra).

### 1.4 SỐ ĐO THẬT — chạy 22/08, key thật, modelapi.vn

`sagent route kiem` chạy trước (GET /models, không tốn token):
`✓ deepseek dùng được 157ms · ✓ grok dùng được 156ms`, mỗi route liệt kê 2 model.

Sau đó 14 phép đo thật (7 × 2 route), tổng **~4.500 token**:

| năng lực | deepseek-v4-flash | grok-4.5 |
|---|---|---|
| `goi-tool` | ✓ (auto; `required` bị từ chối) 445 tok | ✓ `required` chạy thẳng, 479 tok |
| `dau-vao-anh` | ✗ HTTP 400 "This model does not support image" | ✓ đọc đúng màu ảnh 32×32 đỏ, trả "đỏ", 761 tok |
| `dau-ra-co-cau-truc` | ✗ HTTP 400 "This response_format type is unavailable now" | ✓ trả `{"mau":"đỏ"}` đúng schema, 980 tok |
| `reasoning` | ✓ `reasoning_content` dài 152 ký tự, 184 tok | ✗ không có trường nào, dù tiêu 951 tok / 20,3 giây |
| `streaming` | ✓ 9 mẩu SSE + usage (158 tok), 1,5 giây | ✓ 10 mẩu SSE + usage (619 tok), 8,9 giây |
| `dem-token-that` | ✓ vào 93 / ra 40 / tổng 133 | ✓ vào 215 / ra 1088 / tổng 1303 |
| `liet-ke-model` | ✓ 2 model, 26ms | ✓ 2 model, 24ms |

**Không route nào ghi `ChuaDo`** — cả hai đều có key và đều đo được. Đường
`ChuaDo` kèm lý do cụ thể ("chưa có key: không thấy file key %q trong %s") vẫn
có và có bài kiểm riêng (`TestRouteChuaCoKeyRaChuaDoKemLyDo`); nó sẽ hiện ra
ngay khi có ai khai thêm route thứ ba.

### 1.5 Kết luận của bảng, sau khi ghép hai vế

Đây là **phát hiện chính** của lượt này:

> **4 trên 7 năng lực bị chặn ở PHÍA DỰ ÁN, không phải ở nhà cung cấp.**

- `goi-tool` — `aiapi.yeuCau` không có trường `tools`, `aiapi.phanHoi` không đọc
  `tool_calls`. **Cả hai nhà cung cấp đều làm được.**
- `dau-vao-anh` — `aiapi.tinNhan.Content` là `string` thuần; giao thức đòi content
  dạng mảng `{type, image_url}`. **grok làm được.**
- `dau-ra-co-cau-truc` — `aiapi.yeuCau` không có `response_format`. **grok làm được.**
- `reasoning` — `aiapi.phanHoi` chỉ đọc `content`; phần suy luận bị vứt trước khi
  ai nhìn thấy. **deepseek làm được.**

Ba năng lực còn lại (`streaming`, `dem-token-that`, `liet-ke-model`) xanh ở cả
hai bên trên cả hai route.

Riêng **grok / `reasoning` vướng CẢ HAI BÊN** (`cho = ca-hai`): mã không đọc,
mà nhà bán lại cũng không trả — sửa phía dự án một mình không đủ. Bản đầu của
bảng gộp ca này vào "vướng ở phía dự án"; đã sửa, và có bài kiểm riêng
(`TestVuongCaHaiBenKhongDocThanhVuongMotBen`).

### 1.6 Luật ngang quyền — BỐN MẶT, đủ cả bốn

| mặt | `api.nang-luc` (miễn phí) | `api.nang-luc-do` (tốn token) |
|---|---|---|
| hợp đồng `api.Actions` | có | có |
| lệnh CLI | `sagent nang-luc-api` | `sagent nang-luc-api --do` |
| endpoint HTTP | `GET /api/nang-luc-api` | `POST /api/nang-luc-api/do` |
| đường vào web-UI | khối "Năng lực route API" + nút *Làm mới* | nút *Đo thật (tốn token)* |

**Hai action chứ không một**, cố ý: một cái đọc bảng, một cái tiêu tiền. Gộp lại
thì mặt web không có cách nào cho người dùng thấy sự khác biệt trước khi họ bấm —
mà đây đúng là kiểu nút người ta bấm hai lần cho chắc.

Đường `/do` **đòi POST**. Không phải cho đúng lễ nghi REST: một GET tiêu tiền là
một GET mà trình duyệt, bộ nạp trước, hay một lần bấm F5 nhầm đều gọi lại được.
Nút trên web hỏi `confirm()` trước; CLI thì không hỏi (cờ `--do` chính là lời xác
nhận, và một câu hỏi giữa chừng làm lệnh không dùng được trong script).

Đã cập nhật `internal/dash/lachan_test.go` — bảng `duong` map action → đường HTTP,
đúng cơ chế bắt "endpoint có, CLI có, mà không ai bấm được".

### 1.7 Mặt web

Khối "Năng lực route API" đặt ngay dưới "Sức khoẻ route" trong ô AI API — hai
khối trả lời hai câu nối tiếp: *route này còn gọi được không* → *gọi được rồi thì
nó LÀM ĐƯỢC GÌ*.

- Dùng **lại nguyên** bộ lớp `.nl` của bảng năng lực provider (`nl nla`): hàng,
  chấm, ba màu, cột tên, cột bằng chứng là MỘT bộ luật. Hai bảng đứng cạnh nhau
  trong cùng trang và người đọc so chúng bằng mắt.
- Ba trạng thái ba màu, **không bẹp thành hai**. Cột "vướng ở đâu" viết **thành
  chữ** chứ không chỉ là màu — người mù màu và người in đen trắng phải đọc được.
- Nhãn cột `Cho` dựng ở lớp Go (`aiapi.NhanCho`) chứ không chép sang JS.
- **Không token màu mới**, không CDN, không emoji làm icon, vanilla HTML/CSS/JS.
- Cố ý không mượn sắc đỏ báo lỗi cho ô "chưa làm được": không có gì hỏng ở đây,
  và tiêu sắc đỏ cho việc-còn-phải-làm là cách làm nó mất nghĩa ở chỗ thật sự cần.
- `@media (prefers-reduced-motion: reduce)` của trang giữ nguyên, khối này không
  thêm chuyển động nào.

### 1.8 Bài kiểm — 27 bài mới

`internal/aiapi/nangluc_test.go` (12), `internal/dash/nangluc_api_test.go` (8),
`internal/api/nangluc_api_test.go` (5), `cmd/sagent/nangluc_api_test.go` (2).

Bài quan trọng nhất là `TestKiemBatDuocBangKhaiBua` — bảy cách khai bừa cụ thể:
ô xanh mà phần khách chưa đo, ô xanh mà phần NCC chưa đo, bằng chứng rỗng, bằng
chứng không có quan sát nào ("nghe như chép từ tài liệu"), trạng thái thứ tư,
khoá lạ, không nói vướng ở đâu.

**Đã kiểm chứng test đỏ khi gỡ phần sửa ra.** Thay hai lệnh `if` trong
`KiemNangLucAPI` bằng `if false`:

```
--- FAIL: TestKiemBatDuocBangKhaiBua/kết_luận_xanh_mà_phần_nhà_cung_cấp_chưa_đo
--- FAIL: TestKiemBatDuocBangKhaiBua/kết_luận_xanh_mà_phần_khách_chưa_đo
--- FAIL: TestKiemBatDuocBangKhaiBua/bằng_chứng_không_có_quan_sát_nào
```

Khôi phục → xanh lại.

### 1.9 Nghiệm thu

Chạy **từng lệnh riêng biệt**, không dùng `&&`:

```
go build ./...   EXIT=0
go vet ./...     EXIT=0
go test ./...    EXIT=0   (28 gói, không gói nào đỏ)
```

Thêm: trích toàn bộ `<script>` của `index.html` và `node --check` → parse sạch
(một lỗi cú pháp ở bất kỳ đâu là cả script của trang chết).

### 1.10 RANH GIỚI — `cmd/sagent/main.go` KHÔNG bị chạm

Xác nhận bằng `git status`: chỉ 4 file sửa (`internal/api/api.go`,
`internal/dash/server.go`, `internal/dash/web/index.html`,
`internal/dash/lachan_test.go`) + 7 file mới. Không đụng `install/*`,
`internal/flow/*`, `internal/api/chaykho.go`, `internal/provider/*`, cũng không
đụng tài liệu bị cấm.

**Hai dòng dispatch** của lệnh mới **tự ghi vào bảng `commands` từ `init()` của
`cmd/sagent/nangluc_api.go`**, nên `go test ./...` xanh ngay bây giờ mà không
cần ai sửa `main.go`. Việc này an toàn vì init trong một gói chạy theo thứ tự
**tên file**: `main.go` trước `nangluc_api.go`, nên bảng đã dựng xong trước khi
init này ghi vào. Chỗ đó tinh vi nên có hẳn `TestLenhNangLucAPIGoDuoc` canh —
nếu giả định sai thì lệnh sẽ biến mất khỏi `sagent help` mà không báo gì.

Nếu người điều phối muốn nó **tường minh** trong `main.go`, đây là đúng hai dòng
cần cắm vào bảng `commands` (ghi đè bằng chính nó, hoàn toàn vô hại):

```go
"nang-luc-api": {"api.nang-luc", "route API nào làm được gì (làm được / không / chưa đo)", cmdNangLucAPI},
"__nlado":      {"api.nang-luc-do", "đo THẬT năng lực route API (chạm mạng, tốn token)", nil},
```

---

## 2. Sự cố

### 2.1 Hai kết luận SAI suýt đi thẳng vào bảng — cả hai là lỗi của người đo

Đây là sự cố đáng kể nhất, và nó xảy ra đúng kiểu mà đề bài cảnh báo.

**Lần một — ảnh quá nhỏ.** Phép đo thị giác bản đầu dùng ảnh PNG 8×8. grok trả:

```
HTTP 400: Image has 64 total pixels (8x8), which is below the minimum of 512 pixels.
```

Nếu dừng ở đó thì bảng sẽ ghi **grok KHÔNG đọc được ảnh** — trong khi thứ vừa đo
được là phép đo của chính mình gửi ảnh quá nhỏ. Đổi sang 32×32 (1024 điểm ảnh,
vẫn chỉ 96 byte) → grok đọc đúng màu, trả "đỏ".

**Lần hai — `tool_choice` chứ không phải `tools`.** deepseek trả:

```
HTTP 400: Thinking mode does not support this tool_choice
```

Thứ nó từ chối là **cách ép gọi** của phép đo, không phải năng lực gọi tool. Bộ
đo giờ tự hỏi lại bằng `tool_choice: auto` khi thân lỗi 400 có nhắc `tool_choice`,
và ghi **cả hai quan sát** vào bằng chứng. Kết quả: deepseek **có** gọi được tool.

Bài học chung, đã ghi thành bình luận ngay trong `donangluc.go`: **một kết luận
phủ định chỉ được ghi khi đã loại trừ lỗi của người đo.** Hai lần này là hai ô
`KhongLamDuoc` sai suýt vào bảng, mà bảng sai thì tệ hơn không có bảng.

### 2.2 Cột thời gian toàn `0s`

Lượt đo đầu in `mat=0s` cho cả 14 ô. Nguyên nhân: `defer func(){ kq.Mat = ... }()`
sửa biến cục bộ, mà giá trị trả về đã được sao chép trước khi defer chạy. Đổi
sang **named return**. Đáng ghi vì một cột thời gian toàn số 0 trông như "nhanh
quá không đo nổi" chứ không trông như hỏng — nó suýt đi thẳng vào bảng.

### 2.3 Ba bài kiểm sẵn có bắt được ba lỗi thật

- `TestKhongTrangNaoKhaiLaiTokenTrangThai` — bình luận CSS của tôi có nhắc tên
  một token màu; bài kiểm bắt cả trong comment. Đã viết lại không nhắc tên token.
- `TestFileWebKhongCoChuoiDut` — một `\n` trong chuỗi JS bị viết thành xuống dòng
  thật khi ghi file, làm đứt chuỗi. Cả script của trang sẽ chết. Đã ghép lại.
- `TestMoiHanhDongDeuCoDuongVaoTuWeb` — đỏ ngay khi thêm action vào `api.Actions`
  mà chưa khai đường web. Đúng cơ chế bắt cái mà dự án đã vấp ba lần.

### 2.4 Chưa mở dashboard thật bằng trình duyệt

Đã kiểm qua `httptest` (đi đúng `ServeHTTP` và đúng `index.html` được nhúng), đã
kiểm ID + tay cầm `onclick` + parse JS bằng `node --check`. Nhưng **chưa có ai
nhìn bằng mắt** trên trình duyệt thật. Ghi ra đây thay vì im lặng.

### 2.5 Không có sự cố nào về `&&`, em-dash `.ps1`, hay nhánh `main`

Không viết file `.ps1` nào, không chạm `main`, mọi lệnh chạy riêng lẻ.

---

## 3. Bước tiếp theo

Theo đúng thứ tự đáng làm, và cả bốn việc đầu đều là **sửa trong repo này**, vì
bảng vừa chỉ ra chỗ hỏng nằm bên mình:

1. **Nối bảng vào `flow validate`.** `NangLucRoute.LamDuoc(khoa)` đã sẵn sàng và
   trả về **lý do đọc được**, không chỉ một chữ `false`. Đây là toàn bộ lý do
   bảng tồn tại: một flow dùng node `model` đòi tool phải hỏng lúc validate, chứ
   không hỏng lúc chạy sau khi các bước trước đã tiêu token. *Việc này nằm trong
   `internal/flow/*` — vùng của agent khác, nên tôi không chạm.*

2. **Thêm `tools` + `tool_calls` vào `aiapi`.** Cả hai nhà cung cấp đều làm được;
   đây là ô đắt nhất đang bị chặn ở phía mình.

3. **Đổi `tinNhan.Content` sang content-part.** Mở `dau-vao-anh` cho grok. Phải
   giữ tương thích ngược: chuỗi thuần vẫn phải gửi được.

4. **Thêm `response_format` và đọc `reasoning_content`.** Hai trường, hai ô lật xanh.

   Sau mỗi việc 2–4: chạy `sagent nang-luc-api --do --dan` và dán lại số đo. Phép
   đo mã nguồn sẽ tự lật ô sang `ChuaDo` cho tới khi có số đo mới — không tự khai
   làm được.

5. **Đo thêm nhà cung cấp khác** khi có key (Anthropic Messages là ô còn treo của
   Pha 0). Bảng đã có sẵn đường `ChuaDo` kèm lý do "chưa có key".

6. **Mở dashboard bằng mắt** để đóng mục 2.4.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Đọc mẫu `provider/nangluc.go`, dò cách `route.kiem` đi qua bốn mặt | claude-opus-5[1m] | trung bình |
| Thiết kế bảng hai vế (khách / nhà cung cấp) + phép đo bằng reflect | claude-opus-5[1m] | cao |
| Viết bộ đo thật `donangluc.go`, chạy 14 phép đo bằng key thật | claude-opus-5[1m] | cao |
| Sửa hai kết luận sai của phép đo (ảnh 8×8, `tool_choice`) rồi đo lại | claude-opus-5[1m] | cao |
| Nối hợp đồng + lớp API (`NangLucAPI`, `NangLucAPIDo`) | claude-opus-5[1m] | thấp |
| Lệnh CLI `sagent nang-luc-api` (file mới, tự ghi dispatch) | claude-opus-5[1m] | trung bình |
| Hai endpoint HTTP + DTO ba cặp trạng thái | claude-opus-5[1m] | trung bình |
| Khối web + CSS dùng lại `.nl`, nút đo có `confirm()` | claude-opus-5[1m] | trung bình |
| 27 bài kiểm + kiểm chứng test đỏ khi gỡ phần sửa | claude-opus-5[1m] | cao |
| Nghiệm thu ba lệnh riêng + `node --check` + báo cáo | claude-opus-5[1m] | thấp |

---

## 5. Nhận xét tự do

**Câu hỏi khó nhất của lượt này không phải "đo thế nào" mà "ô này nói về ai".**
Nửa CLI dễ hơn nhiều: một adapter làm được hay không là chuyện của chính adapter
đó. Nửa API thì mỗi ô ngồi giữa hai bên — mã của mình và nhà cung cấp — và một
bảng một-vế sẽ nói dối theo cách rất khó bắt: nó **đúng** ("route này không gọi
được tool qua sagent") mà vẫn dẫn người đọc đi sai chỗ (đổi nhà cung cấp, trong
khi grok gọi tool ngon lành). Hai vế tốn thêm hai trường và một cột, nhưng nó
biến bảng từ "danh sách thứ không chạy" thành "danh sách việc phải làm, kèm địa
chỉ".

**Bộ đo tự bắt được lỗi của chính nó hai lần, và đó là phần đáng giá nhất.** Cả
hai lần đều ra `KhongLamDuoc` trông rất thuyết phục — có mã HTTP, có nguyên văn
nhà cung cấp, đủ mọi dấu hiệu của một số đo tử tế. Cả hai đều sai. Thứ cứu được
là thói quen đọc *nội dung* thân lỗi thay vì chỉ đọc mã 400: "below the minimum
of 512 pixels" và "does not support this tool_choice" đều đang nói về **yêu cầu
tôi gửi**, không phải về năng lực của họ. Nếu đề bài không nhấn mạnh chuyện bảng
khai bừa thì tôi đã ghi cả hai vào bảng rồi — chúng qua được mọi phép kiểm hình
thức mà `KiemNangLucAPI` áp: có bằng chứng, có con số, có nguyên văn.

Điều đó dẫn tới giới hạn thật của `KiemNangLucAPI`: nó chặn được bảng **rỗng
bằng chứng** và bảng **chép từ tài liệu**, nhưng không chặn được một **phép đo
sai**. Không có cách nào tự động chặn cái sau — nên thứ thay thế là để bộ đo nằm
trong repo, chạy lại được bằng một lệnh, và ghi nguyên văn quan sát vào bằng
chứng để người sau đọc mà không phải tin ai. Bằng chứng của `goi-tool` bây giờ
mang cả lần đo hụt lẫn lần đo lại — dài hơn, nhưng đó là dòng dạy được nhiều nhất
trong cả bảng.

**Một chỗ tôi cân nhắc lâu rồi vẫn chọn cách "nặng" hơn:** dùng type alias sang
`provider.TrangThaiNangLuc` thay vì khai ba hằng riêng cho `aiapi`. Alias tạo một
liên kết hơi trái chiều — nửa API vốn cố ý không đi qua nửa CLI. Nhưng ba trạng
thái là một **quy ước về nghĩa**, không phải chi tiết cài đặt, và hai bản sao của
một quy ước là hai bản sẽ lệch. Với alias thì lệch **không xảy ra được**, thay vì
chỉ **bị phát hiện**. Cái giá là một dòng import; cái được là mặt web vẽ hai bảng
bằng chung một bộ lớp CSS mà không sợ hai bên hiểu "chua-do" khác nhau.

**Con số đáng nhớ nhất:** 4 trên 7 năng lực bị chặn ở phía dự án, không phải ở
nhà cung cấp. Trước lượt này không ai trả lời được câu đó — kể cả câu "có ai
chặn không". Bảng không sửa được ô nào, nhưng nó biến bốn chỗ hỏng vô hình thành
bốn việc có địa chỉ, và mỗi việc là vài trường trong hai struct.

**Về ranh giới:** phần khó chịu nhất là cần một lệnh CLI mà không được sửa
`main.go`. Cách tự ghi vào bảng `commands` từ `init()` của file mình chạy được và
giữ `go test ./...` xanh ngay, nhưng nó dựa vào thứ tự init theo tên file — một
giả định đúng nhưng tinh vi. Tôi không muốn để nó sống bằng một bình luận, nên có
hẳn `TestLenhNangLucAPIGoDuoc` nói ra: nếu giả định sai thì `sagent nang-luc-api`
sẽ bị hiểu thành một địa chỉ hồ sơ và im lặng biến mất khỏi `help`. Hai dòng
dispatch tường minh vẫn nằm ở mục 1.10 để cắm vào `main.go` nếu muốn.
