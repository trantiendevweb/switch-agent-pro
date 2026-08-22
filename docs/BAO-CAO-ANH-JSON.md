# Báo cáo — đóng hai ô cuối của bảng năng lực API: `dau-vao-anh` và `dau-ra-co-cau-truc`

Nhánh `sagent/anh-json-22-08`, rẽ từ `main` (`git rev-list --count main..HEAD` = 0 lúc bắt đầu).
Ngày 22/08/2026.

---

## 1. Đã làm (kèm số đo)

### Kết quả một dòng

Trước bản vá, cả hai ô đều ✗ ở **cả hai** route — nhưng vì hai lý do ngược nhau.
Sau bản vá, chúng tách ra đúng như phải tách:

| ô | grok-4.5 | deepseek-v4-flash |
|---|---|---|
| `dau-vao-anh` | ✗ → **✓** | ✗ `ca-hai` → ✗ **`nha-cung-cap`** |
| `dau-ra-co-cau-truc` | ✗ → **✓** | ✗ `ca-hai` → ✗ **`nha-cung-cap`** |

Đếm bằng chính đầu ra của `sagent nang-luc-api`:

```
"vướng ở phía dự án"   → 0 ô
"vướng cả hai bên"     → 0 ô
"vướng ở nhà cung cấp" → 3 ô   (deepseek/ảnh, deepseek/json, grok/reasoning)
```

**Nửa API không còn nợ ô nào.** Ba ô ✗ còn lại đều nằm ngoài tầm với của repo này.

### Đo thật với key thật (hôm nay)

Ảnh tự sinh trong test (PNG 32×32 toàn đỏ, 97–103 byte) — **không có file nhị phân nào
vào repo**. Lý do: `core.autocrlf=true` sẽ sửa nội dung file nhị phân nếu quên khai
`binary` trong `.gitattributes`, đúng cái bẫy đã ghi cho asset vendor của dashboard.

| phép đo | route | kết quả | token | thời gian |
|---|---|---|---|---|
| `TestE2EAnhThatTuNhaCungCap` | grok-4.5 | đọc đúng màu, trả `"đỏ …"` | vào 224 / ra 558 / **782** | 11,0 s |
| `TestE2ECoCauTrucThatTuNhaCungCap` | grok-4.5 | `{"mau":"đỏ và vàng"}` đúng schema | vào 263 / ra 1035 / **1298** | 15,5 s |
| CLI `--anh` | grok-4.5 | `đỏ`, ảnh 32×32 = 1024 điểm ảnh | **933** | 14,9 s |
| CLI `--anh` + `--so-do` | grok-4.5 | `{"mau_nen":"red","so_diem_anh":0}`, 2 khoá | **1523** | 17,8 s |
| CLI `--anh` (bị chặn) | deepseek | chặn, **0 byte lên mạng** | 0 | tức thì |
| CLI `--anh --cu-gui` | deepseek | gửi thật → HTTP 400 `"This model does not support image"` | ~0 ra | — |

Dòng cuối đáng chú ý: nó **xác nhận số đo 22/08 trong sổ hôm nay vẫn đúng**, bằng chính
cửa thoát mà bản vá chừa ra.

Hai bài E2E gate bằng biến môi trường theo đúng lệ `TestE2ESuyLuanThatTuNhaCungCap`:
`SAGENT_E2E_ANH=1`, `SAGENT_E2E_JSON=1`.

### Câu khó 1 — `tinNhan.Content` là `string` thuần, đổi nó thì đụng tới đâu?

**Đã chọn: THÊM TRƯỜNG `Phan []PhanNoiDung` + `MarshalJSON`. Không đổi kiểu.**

Ba hướng và cái giá của từng hướng (ghi đủ ở đầu `internal/aiapi/anh.go`):

1. **Đổi `Content` sang `any`** — mọi chỗ *dựng* tin nhắn phải sửa, và tệ hơn nhiều:
   mọi chỗ *đọc* câu trả lời cũng phải sửa. Chiều về của giao thức này **luôn** là
   chuỗi; đổi kiểu của nó là bắt cả dự án viết một phép ép kiểu ở mỗi chỗ đọc, để phục
   vụ một chiều đi ít dùng. Ép kiểu hụt ở đó hỏng **lúc chạy**, sau khi lượt gọi đã trả
   tiền, chứ không hỏng lúc biên dịch.
2. **`json.RawMessage`** — người gọi tự dựng JSON, tức mỗi chỗ gọi phải tự biết giao
   thức. Trình biên dịch thôi kiểm được gì, và chỗ gõ sai tên khoá `image_url` sẽ trả
   HTTP 200 kèm câu trả lời tử tế (nhà cung cấp nuốt phần nó không hiểu).
3. **Thêm trường** — cái giá là **có hai đường cùng dựng nội dung một tin nhắn**, và
   hai đường làm cùng một việc thì sẽ có lúc lệch nhau.

**Cách trả cái giá của hướng 3:** câu hỏi "gán cả hai thì cái nào thắng" được **khai ra
thành luật**, không để là hành vi tình cờ của hàm marshal:

```
Phan rỗng      → thân JSON y HỆT trước bản vá (`content` là chuỗi)
Phan khác rỗng → `content` thành MẢNG, và `Content` KHÔNG bị vứt:
                 nó vào làm mẩu `text` ĐẦU TIÊN của mảng
```

Hai trường **ghép lại chứ không tranh nhau**, nên không có ca nào người gọi mất chữ mà
không biết. `TestAnhDiHetDuongToiThanJSON` canh đúng luật đó, và phép phá M2 (bỏ dòng
ghép `Content` vào mẩu đầu) làm **6 bài đỏ**.

**Hệ quả phải nói ra vì nó làm hỏng một phép đo:** sau khi có `MarshalJSON`, **kiểu Go
của `Content` không còn quyết định hình dạng trên dây**. Phép đo `dau-vao-anh` cũ hỏi
*"Content còn là chuỗi thuần không"* — từ nay câu đó **trả lời sai**, và sai theo hướng
tệ nhất: khai "không gửi được ảnh" cho một đường đang chạy, lùa người ta đi sửa thứ
không hỏng. Xem mục 1 phần "điểm mù" bên dưới.

### Câu khó 2 — route không làm được mà người dùng vẫn gửi ảnh thì sao?

**Đã chọn: CHẶN ở phía mình, trước khi chạm mạng — nhưng CHỈ khi bảng đã ĐO ĐƯỢC là
không làm được, và luôn chừa cửa `--cu-gui`.**

Vì sao chặn chứ không cứ gửi rồi để nhà cung cấp trả 400:

1. Bảng năng lực **đã biết câu trả lời trước khi gọi**. Biết mà không dùng là đúng cái
   bẫy "có ở mọi tầng trừ tầng cuối cùng" — và lần này thứ bị bỏ phí là một phép đo đã
   tốn tiền thật để có.
2. Một lượt hỏng vẫn có thể bị tính tiền, và ảnh base64 làm `prompt_tokens` phình lên
   trước khi nhà cung cấp kịp từ chối.
3. Lỗi của nhà cung cấp là tiếng Anh, nói về một `image_url` người dùng chưa từng gõ, và
   **không** nói route nào trong cấu hình của họ làm được. Lời chặn nói được, vì nó cầm
   cả danh sách route.

Vì sao **không bao giờ chặn ô `ChuaDo`:** ba trạng thái là một quy ước về nghĩa, và
"chưa ai đo" **không phải** "đã đo là không". Chặn một ô `ChuaDo` là bẹp ba trạng thái
thành hai *ngay tại chỗ tiêu tiền* — thêm một route mới vào `project.toml` là lập tức
không gửi ảnh được, mà chẳng ai đo gì cả. Ô `ChuaDo` được đi, kèm câu nói rõ là chưa đo.

Vì sao vẫn phải có `--cu-gui`: bằng chứng trong sổ có **ngày**, nhà cung cấp thì nâng cấp
model. Một lời chặn không gỡ được sẽ biến số đo hôm nay thành luật vĩnh viễn.

Lời chặn thật, in ra từ binary:

```
✗ route "deepseek" KHÔNG gửi ảnh trong tin nhắn (vision) — đo 22/08: HTTP 400
  "This model does not support image" với ảnh PNG 32x32 (1024 điểm ảnh)
     Chặn ở phía sagent, CHƯA gửi đi: bảng năng lực đã biết câu trả lời trước khi
     gọi, và một lượt hỏng vẫn có thể bị tính tiền.
     Route đã cấu hình mà LÀM ĐƯỢC: grok (grok-4.5)
     Số đo có thể đã cũ — vẫn muốn gửi thì thêm --cu-gui, hoặc đo lại:
     sagent nang-luc-api --do deepseek
```

Lỗi này là `LoiNguoiDung` → tầng trên **không** được coi là cớ nhảy route dự phòng: đường
dự phòng đi qua `aiapi.Goi`, tức sẽ gửi một yêu cầu **không có ảnh** rồi báo kết quả như
thật.

### Điểm mù của bảng năng lực — đã thu hẹp thêm một nấc

Bảng dò phía dự án bằng **reflection trên kiểu thật**, nên nó chỉ trả lời "kiểu có
trường đó không". Hai ô này nay **thôi dùng reflect** và chuyển sang **dựng thật một
thân JSON rồi đọc lại nó**:

- ô ảnh: marshal một `tinNhan` có ảnh → đòi `content` là mảng, có mẩu `image_url` mang
  data URL, **và** có mẩu `text` (bắt ca nuốt mất câu hỏi);
- ô JSON: marshal một `yeuCau` có schema → đòi `response_format.json_schema.schema` còn
  nguyên `required`, **và** đòi có kiểu `KhoiCoCauTruc.Thieu` tức có người *đọc lại*.

Vẫn miễn phí, vẫn không chạm mạng, vẫn tự lật khi mã đổi — nhưng câu hỏi mạnh hơn hẳn:
không phải "kiểu có chỗ chứa không" mà **"thứ ta gửi đi có mang nó không"**.

**Điểm mù còn lại, nói thẳng:** phép đo tự dựng lấy tin nhắn của nó. Nó **không** trả lời
được *"`GoiKem` có nhét ảnh của người dùng vào đó không"*. Câu đó chỉ bài kiểm đi hết
đường mới trả lời được — và phép phá M1 chứng minh điều đó: bỏ dòng gán ảnh trong
`GoiKem` thì **bảng vẫn in ✓**, chỉ `TestAnhDiHetDuongToiThanJSON` đỏ.

### Bài kiểm đi hết đường tới chỗ người dùng thấy

Theo chuẩn `TestSuyLuanDiHetDuongToiKetQua` và `tool_test.go`:

```
file .png trên đĩa → DocAnh → GoiKem → thân JSON THẬT (nhà cung cấp giả đọc lại)
                                     → KetQua → chữ trên màn hình (manHinhKem)
```

### Bản chứng minh: test ĐỎ khi gỡ phần sửa ra

Đã dựng một bộ **phép thử phá hoại**: sửa đúng một dòng mã sản phẩm, chạy `go test`, rồi
`git checkout --` lùi lại. **14/14 phép đều làm test đỏ.**

| # | gỡ cái gì ra | bài đỏ |
|---|---|---|
| M1 | `GoiKem` quên gán ảnh vào `Phan` | `TestAnhDiHetDuongToiThanJSON` |
| M2 | `MarshalJSON` nuốt câu hỏi khi có ảnh | 6 bài, gồm `TestPhepDoAnhVaJSONDoThanJSONThatChuKhongDoKieu` |
| M3 | `GoiKem` quên gửi `response_format` | `TestCoCauTrucDiHetDuongToiKetQua` |
| M4 | gỡ lời chặn của bảng năng lực | 3 bài chặn/`--cu-gui` |
| M5 | không đối chiếu `required` | `TestJSONDungNhungThieuKhoaBatBuocThiKeRaTungKhoa` |
| M6 | chặn cả ô `chua-do` | 5 bài |
| M7 / M7b / M7c | tầng cuối bỏ khối ảnh / khối JSON / cảnh báo | `TestManHinhKemKhongDeRotKhoiNao` |
| M8 | `--anh` chỉ nhận một ảnh (`strFlag`) | `TestRutCoGoiVaKiemCoGoiCanhDungLuatCua_apiGoi` |
| M9 / M9b / M9c | `kiemCoGoi` luôn nil / `quyetDinhGoi` bỏ soát / chốt nhầm nhánh | `TestQuyetDinhGoiSoatCoTruocRoiMoiChotNhanh` |
| M10 | `DocAnh` tin vào đuôi file thay vì nội dung | `TestDocAnhTuChoiFileKhongPhaiAnh` |

**Vòng đầu có 2 phép SỐNG SÓT, và cả hai cùng một hình dạng** — hàm thuần *đúng*, nhưng
không ai kiểm nó có được **gọi** không:

- M7 gốc: bỏ hẳn vòng lặp in khối ảnh khỏi `apiGoiKem` → **toàn bộ test vẫn xanh**.
  `dongAnhGuiDi` có bài kiểm riêng và nó vẫn xanh, vì bài đó gọi thẳng hàm.
- M8 gốc: đổi `--anh` sang cờ chỉ-một-giá-trị → **toàn bộ test vẫn xanh**.

Đó đúng là hình dạng lỗi mà cả bản vá này dựng lên để chống, lần này **nằm trong chính
bản vá**. Đã vá bằng hai cách khác nhau, cố ý:

- `manHinhKem()` — gom **toàn bộ** chữ in ra sau một lượt vào một hàm thuần. Bỏ sót một
  khối là đỏ ngay. (*viết thêm bài kiểm*)
- `quyetDinhGoi()` — soát cờ **rồi** chốt nhánh, và `apiGoi` `switch` theo `v.Nhanh`,
  thứ chỉ hàm đó sinh ra. **Gỡ lời gọi đi là hỏng lúc biên dịch.** (*làm cho lời gọi
  không bỏ được* — cách này mạnh hơn, vì nó không dựa vào việc có ai nhớ viết test)

### Bốn mặt

**Không có action mới**, nên không phát sinh nghĩa vụ bốn mặt. Đi đúng tiền lệ của
`--tool` (thêm cờ cho `api.call` sẵn có, gọi thẳng `aiapi.*` từ CLI, không đụng
`api.Actions`) — đã kiểm: `--tool` cũng không có mặt web, và test ngang quyền vẫn xanh.

`internal/api/api.go` và `internal/flow/*` **không bị đụng một dòng nào** (`git diff main
--name-only | grep internal/flow` = 0).

Nếu điều phối muốn đưa hai năng lực này lên mặt web thì cần **thêm 2 dòng vào
`internal/api/api.go`, trong `var Actions`, ngay sau dòng `"api.call",`** — tôi không tự
cắm vì đó là vùng của agent kia:

```go
	// Gọi KÈM ảnh và/hoặc JSON schema. Tách khỏi `api.call` vì hai lẽ: nó KHÔNG
	// có route dự phòng (đường dự phòng qua aiapi.Goi, không mang ảnh), và nó bị
	// bảng năng lực CHẶN trước khi chạm mạng — hai tính chất mà mặt web phải cho
	// người dùng thấy trước khi họ bấm.
	"api.call-kem",
```

Kèm đó, phía tôi đã có sẵn mọi thứ để nối: `aiapi.GoiKem`, `aiapi.DocCoCauTruc`,
`KetQua.CanhBaoTruocKhiGui` (có thẻ JSON, đọc thẳng được từ `/api/ai`), và
`aiapi.SoatTruocKhiGui` cho mặt web soát trước khi vẽ nút.

### File đã thêm / sửa

| file | vai trò |
|---|---|
| `internal/aiapi/anh.go` (+247) | `PhanNoiDung`, `Anh`, `DocAnh` (soi kiểu theo **nội dung**, không theo đuôi), cảnh báo ngưỡng 512 điểm ảnh |
| `internal/aiapi/cocautruc.go` (+220) | `DangTraLoi`, `SoDoNghiem`, `DocCoCauTruc` (ba ca), `MoTaCoCauTruc` |
| `internal/aiapi/goikem.go` (+200) | `GoiKem`, `TuyChonGoi`, `SoatTruocKhiGui` |
| `internal/aiapi/aiapi.go` (+91) | `tinNhan.Phan` + `MarshalJSON`, `yeuCau.DangTraLoi`, `themVao`, `KetQua.CanhBaoTruocKhiGui` |
| `internal/aiapi/nangluc.go` (+121) | hai phép đo mới, đo thân JSON thật |
| `cmd/sagent/api_anh_json.go` (+235) | `--anh` / `--so-do`, `manHinhKem` |
| `cmd/sagent/api.go` | `coGoi`, `rutCoGoi`, `kiemCoGoi`, `quyetDinhGoi` |
| `cmd/sagent/main.go` (+29) | `strFlagNhieu`, help |
| 4 file `_test.go` | +935 dòng bài kiểm |

---

## 2. Sự cố

**1. Bộ phép thử phá hoại nuốt mất bản sửa chưa commit.** Script dọn dẹp bằng
`git checkout -- <file>`, và tôi chạy nó khi cây làm việc còn bẩn → mất một sửa đổi của
`cmd/sagent/api_anh_json.go`, kéo theo build hỏng vì file test còn tham chiếu hàm vừa
biến mất. Đã làm lại và đổi lệ: **commit trước, phá sau**. Đã ghi cảnh báo đó vào đầu
script.

**2. Hai phép phá hoại tôi viết là phép phá HỎNG, không phải lỗ hổng.**
`switch { case false: ... }` trong Go là một nhánh không bao giờ khớp — nó **không đổi
hành vi gì**, nên "test vẫn xanh" ở đó không nói lên điều gì. Suýt ghi vào báo cáo là một
lỗ hổng. Bài học đúng loại đã ghi trong `donangluc.go`: *một kết luận phủ định phải loại
trừ lỗi của người đo trước khi được ghi.* Đã thay bằng phép phá thật (`case true: return
nil`) và nó đỏ ngay.

**3. `go test ./...` KHÔNG ra 0 — vì `internal/flow`, và không phải do tôi.**
Hai bài đo đồng thời `TestNoiRaKhiDangChoTran` và `TestCacNhanhDocLapChaySongSong` thay
nhau đỏ khi máy đang tải nặng; chạy riêng thì xanh 3/3.

Đã chứng minh là có sẵn từ trước, không phải hệ quả của bản vá:

- `git diff main --name-only | grep -c internal/flow` → **0** (không đụng một file nào);
- dựng một worktree tạm tại **đúng gốc nhánh `04f7ebc`** (không có một dòng nào của tôi)
  và chạy `go test ./internal/flow/` ba lần → **đỏ 2/3 lần với đúng bài đó**.

Hai bài này đến từ `3d8b1dc` "Tran dong thoi cho duong flow" — vùng của agent kia. Tôi
không sửa. Worktree tạm đã gỡ.

**4. Heredoc `<<'EOF'` qua Bash tool hỏng hai lần** với nội dung Go/Python nhiều
backslash và nháy — đúng lớp lỗi đã ghi trong trí nhớ (`bay-backslash-python-heredoc`).
Đã chuyển sang công cụ Write/Edit cho mọi file lớn. Mất khoảng hai lượt.

---

## 3. Bước tiếp theo

1. **Nối mặt web** (cần điều phối cắm 1 dòng vào `internal/api/api.go`, xem mục 1). Phía
   `aiapi` đã sẵn sàng: mọi kiểu đều có thẻ JSON.
2. **`flow validate` nên hỏi hai ô này.** Bảng đã trả lời được, `SoatTruocKhiGui` đã có,
   nhưng một node `model` khai `anh = [...]` trỏ vào deepseek vẫn chỉ hỏng lúc chạy. Đây
   đúng là lý do bảng năng lực tồn tại — vẫn còn một chỗ chưa đi hỏi nó.
3. **Đường `--stream` chưa mang ảnh/schema.** Hiện từ chối thẳng và nói rõ, không im
   lặng bỏ cờ. Muốn đi chung thì `yeuCauStream` cần hai trường tương ứng.
4. **Cân lại `nguongDiemAnhDaDo = 512`** khi có route thứ ba. Nó là số của *một* nhà
   cung cấp đo được *một* lần, nên hiện chỉ cảnh báo chứ không chặn — đúng, nhưng nếu có
   thêm số đo thì nên khoá theo `(base_url, model)` như sổ `soDoNangLuc`.
5. **Ô `reasoning` của grok** là ô ✗ duy nhất còn lại mà không ai đóng được ở phía dự án.
   Cách duy nhất là đổi nhà bán lại — đáng ghi vào sổ nợ chứ không đáng thử sửa.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Khảo sát `aiapi`, đọc chuẩn `suyluan_test.go` / `tool_test.go` | Opus 5 (1M) | thấp |
| Cân ba hướng đổi `tinNhan.Content`, chốt luật hợp nhất | Opus 5 (1M) | **cao** |
| `anh.go` — `PhanNoiDung`, `DocAnh`, sniff theo nội dung, ngưỡng điểm ảnh | Opus 5 (1M) | trung bình |
| `cocautruc.go` — `response_format`, `DocCoCauTruc` ba ca, gỡ rào ```json | Opus 5 (1M) | trung bình |
| `goikem.go` — `GoiKem` + `SoatTruocKhiGui` (câu khó 2) | Opus 5 (1M) | **cao** |
| Viết lại hai phép đo `nangluc.go` sang đo thân JSON thật | Opus 5 (1M) | **cao** |
| Dời neo bài kiểm cũ, thêm bài canh "bảng còn phân biệt được" | Opus 5 (1M) | trung bình |
| Mặt CLI: `--anh`, `--so-do`, `--cu-gui`, `manHinhKem` | Opus 5 (1M) | trung bình |
| Bộ 14 phép thử phá hoại + vá 2 lỗ nó chỉ ra | Opus 5 (1M) | **cao** |
| Đo thật bằng key thật (E2E + binary) | Opus 5 (1M) | thấp |
| Truy `internal/flow` đỏ, dựng worktree tại gốc nhánh để loại trừ | Opus 5 (1M) | trung bình |
| Viết báo cáo | Opus 5 (1M) | thấp |

Nghiệm thu, đo bằng **mã thoát**:

```
go build ./...                                  → 0
go vet   ./...                                  → 0
go test  ./... trừ internal/flow                → 0   (24 gói)
go test  ./internal/flow/                       → chập chờn, CÓ SẴN TỪ TRƯỚC (mục 2)
```

Chi phí token thật đã tiêu cho việc đo: khoảng **4 500 token** qua modelapi.vn
(782 + 1298 + 933 + 1523, cộng một lượt 400 gần như 0).

---

## 5. Nhận xét tự do

**Thứ đáng giá nhất của lượt này không phải hai ô xanh, mà là hai phép phá hoại sống
sót.** Bộ test của tôi đã xanh, đã "đi hết đường" theo đúng chuẩn của
`TestSuyLuanDiHetDuongToiKetQua`, đã có bài E2E chạm nhà cung cấp thật — và vẫn để lọt
đúng cái lỗi mà cả ngày 22/08 dự án này dựng lên để chống. Bỏ hẳn khối "ảnh nào đã gửi"
ra khỏi màn hình: xanh. Lặng lẽ vứt ảnh thứ hai: xanh.

Lý do rất cụ thể và đáng nhớ: **tôi đã tách hàm dựng chữ ra thành hàm thuần để test được
— rồi test đúng cái hàm thuần đó.** Việc tách ra là đúng; nhưng nó dời chỗ hỏng lên đúng
một tầng, tới cái dòng gọi hàm, và cái dòng đó thì không ai canh. "Tách ra cho dễ test"
không đồng nghĩa với "đã test", và một hàm thuần có bài kiểm đẹp là chỗ trốn tốt nhất cho
một lời gọi đã biến mất.

Điều đó dẫn tới một nhận xét mạnh hơn, và tôi nghĩ nó đáng thành lệ chung của repo:
**giữa "viết thêm một bài kiểm" và "làm cho lỗi không xảy ra được", luôn chọn cái thứ
hai khi giá ngang nhau.** Tôi đã vá hai lỗ bằng hai cách khác nhau để so:

- `manHinhKem` là cách thứ nhất — gom màn hình lại rồi viết bài kiểm đòi đủ khối. Nó
  hoạt động, nhưng nó chỉ mạnh bằng trí nhớ của người sau: thêm khối thứ tám mà quên
  thêm một dòng vào bài kiểm thì khe mở lại y như cũ.
- `quyetDinhGoi` là cách thứ hai — `apiGoi` `switch` theo `v.Nhanh`, thứ chỉ hàm soát cờ
  sinh ra. Gỡ lời soát đi là **hỏng lúc biên dịch**. Không cần ai nhớ gì cả.

Cách thứ hai đắt hơn lúc viết chừng mười phút, và rẻ hơn vĩnh viễn sau đó.

**Về bảng năng lực.** Tôi nghĩ nó vừa chứng minh được giá trị thật của mình, không phải
qua bảy ô nó in ra, mà qua chuyện nó **thay đổi được một quyết định thiết kế**. Câu khó 2
sẽ không có câu trả lời tử tế nếu không có bảng: không có nó thì lựa chọn duy nhất là
"cứ gửi rồi để nhà cung cấp từ chối", và người dùng nhận về một câu tiếng Anh không nói
họ phải làm gì. Có bảng thì lời chặn nói được cả ba thứ — đo được gì, route nào thay
được, gõ gì để đi tiếp. Một phép đo tốn tiền hồi sáng hôm nay trả lại giá trị của nó vào
buổi tối, ở một chỗ không ai định trước.

Nhưng cũng chính vì thế mà **ba trạng thái phải được giữ nghiêm ở chỗ tiêu tiền**. Chỗ dễ
sai nhất của cả lượt này là dòng `case ChuaDo:` trong `SoatTruocKhiGui`: viết `return
loiNguoi(...)` ở đó thì mọi thứ vẫn chạy, mọi bài kiểm cũ vẫn xanh, và bảng vừa lặng lẽ
mất một trạng thái — "chưa ai đo" bị đọc thành "đã đo là không", ngay tại nơi nó gây thiệt
hại lớn nhất. Phép phá M6 làm 5 bài đỏ, và tôi cố ý viết nó để chuyện đó không bao giờ
lọt qua.

**Một điều cuối, về phép đo tự soi mình.** `dau-vao-anh` là ô đầu tiên trong bảng mà
**phép đo cũ trở nên sai vì bản vá làm đúng**: sau khi có `MarshalJSON`, kiểu Go của
`Content` thôi quyết định hình dạng trên dây, nên câu hỏi "Content còn là chuỗi thuần
không" trả lời ngược. Nếu tôi chỉ thêm trường rồi để nguyên phép đo, ô đó sẽ khai ✗ cho
một đường đang chạy tốt — và người đọc bảng sẽ đi sửa thứ không hỏng. Reflection trên
kiểu thật là một thiết kế tốt và nó đã phục vụ sáu ô kia rất đúng, nhưng nó đo *hình
dạng của kiểu*, mà thứ nhà cung cấp đọc là *hình dạng trên dây*. Chừng nào hai thứ đó
còn trùng nhau thì nó đúng; hôm nay là ngày đầu tiên chúng tách ra. Phép đo mới —
marshal thật rồi đọc lại — đúng hơn ở cả bảy ô, và tôi nghĩ năm ô còn lại nên lần lượt
chuyển sang nó khi có dịp, chứ đừng đợi tới lần thứ hai có người phát hiện bảng đang nói
sai về chính mình.
