# Báo cáo — `KetQua.SuyLuan` có người đọc, và `goi-tool` hết vướng ở phía dự án

Nhánh `sagent/tools-suyluan-22-08`, tách sạch từ `main` (`git rev-list --count main..HEAD` = 0 lúc tạo).

---

## 1. Đã làm

### VIỆC 1 — phần suy luận nay có ba mặt đọc nó

Trước lượt này, `KetQua.SuyLuan` là một trường **có ở mọi tầng trừ tầng cuối**: nhà
cung cấp trả `reasoning_content` → `Goi()` gán đúng → có bài kiểm xanh → **không ai
đọc**. Đo 22/08 với deepseek-v4-flash: câu trả lời 91 ký tự, phần suy luận 477 ký tự
— phần bị vứt dài **gấp 5,2 lần** phần giữ lại, và người dùng đã trả tiền cho cả hai
(300 token).

| Chỗ thêm | Là gì |
|---|---|
| `internal/aiapi/suyluan.go` (mới, 142 dòng) | `KhoiSuyLuan` + `DocSuyLuan(kq, routes)` — **một nguồn** cho cả CLI lẫn mặt web |
| `cmd/sagent/api.go` + `api_suyluan_tool.go` (mới) | cờ `--suy-luan`, hàm thuần `dongSuyLuan` dựng chữ |
| `internal/dash/server.go` | trường `suy_luan` trong **cả hai** thân trả về của `/api/ai` (JSON và mẩu tổng kết SSE) |
| `internal/dash/web/index.html` | khối `<details id="ai-nghi">` đóng sẵn + ô `#ai-nghi-trong`, hàm `veSuyLuan` |
| `internal/aiapi/stream.go` | đường stream gom `delta.reasoning_content` — **bắt buộc**, vì mặt web luôn hỏi bằng stream |

**Chỗ khó nhất là chuỗi rỗng, không phải chuỗi có chữ.** Rỗng mang HAI nghĩa ngược
nhau: model không nghĩ, HOẶC nhà cung cấp không trả phần nghĩ ra. Đo 22/08: grok-4.5
là ca thứ hai — HTTP 200, không có `reasoning_content` lẫn `reasoning`, mà lượt đó
tiêu **951 token** và mất **20,3 giây**. `DocSuyLuan` tra bảng năng lực của route
**đã trả lời** (không phải route người dùng gõ — lượt chuyển dự phòng thì hai cái đó
khác nhau) rồi dựng sẵn câu giải thích, kèm lệnh `sagent nang-luc-api <route>`.

Cả ba mặt bị **cấm bằng test** nói câu "model không nghĩ" / "model không suy luận":
`TestSuyLuanRongPhaiNoiRoDangONghiaNao`, `TestBatCoMaKhongCoPhanNghiThiKhongDuocDeHIEUNHAM`,
`TestKhongCoPhanSuyLuanThiNoiRoViSao`.

Chạy thật, nhà cung cấp giả, đúng chữ người dùng thấy:

```
  ⌥ lượt này kèm 69 ký tự suy luận của model (đã tính tiền trong token ra) — xem bằng cờ --suy-luan
```
```
  ┌─ phần suy luận của model · 69 ký tự
  │ Buoc 1: 17 nhan 20 la 340. Buoc 2: 17 nhan 3 la 51. Buoc 3: cong lai.
  └─
```
```
  ⌥ lượt này KHÔNG có phần suy luận đọc được.
    chưa ai đo route "gialap" có trả phần nghĩ ra hay không (…) nên ô trống này chưa nói được gì.
    Tra lại: sagent nang-luc-api gialap
```

Không bật cờ mà lượt đó **có** phần nghĩ thì vẫn nhắc một dòng: người dùng đã mua nó
rồi, im lặng nghĩa là thứ đã mua coi như không tồn tại — và họ không có cách nào đoán
ra tên cờ.

### VIỆC 2 — `goi-tool` hết vướng ở phía dự án

`internal/aiapi/tool.go` (mới, 190 dòng): `Tool` / `HamTool` / `ToolHam()` /
`LoiGoiTool` / `HamDuocGoi` / `GoiTool()` / `MoTaLoiGoiTool()`; `yeuCau` thêm
`tools` + `tool_choice` (cả hai `omitempty`), `tinNhan` thêm `tool_calls`, `KetQua`
thêm `ToolCalls`. Người gọi đầu tiên: `sagent api --tool <file.json> [--tool-chon …]`.

```
  ┌─ model đòi gọi 1 tool. sagent KHÔNG chạy chúng — đây là lời gọi nguyên văn, bạn tự quyết:
  │ lay_gio({"thanh_pho": "Ha Noi"})
  │   id: call_9f2
  └─
```

Bảng năng lực đổi từ `✗ goi-tool` (cả hai route) sang:

```
    ✓ goi-tool   đo 22/08: tool_choice=required chạy thẳng, trả 1 tool_call, hàm "lay_gio",
                 479 token · phía dự án: aiapi.yeuCau có trường `tools`, aiapi.phanHoi đọc
                 `tool_calls`, và KetQua.ToolCalls mang nó ra tới người gọi
```

**CÂU HỎI ĐỀ BÀI ĐÒI TRẢ LỜI: sau khi model trả về một tool_call thì AI có chạy tool đó không?**

**KHÔNG. `aiapi` không chạy tool, không bao giờ.** `GoiTool` gửi định nghĩa đi, nhận
`tool_calls`, đặt vào `KetQua.ToolCalls`, trả về cho người gọi. Hết.

**Trần vòng lặp: KHÔNG CÓ VÒNG LẶP.** Trần bằng **0 lượt tự chạy** và **đúng 1 lượt
gọi mạng** cho mỗi lần gọi hàm — bằng đúng `Goi`. Điều này được canh bằng test chứ
không bằng lời hứa: `TestToolDiHetDuongTuDinhNghiaToiKetQua` đếm số lượt chạm nhà
cung cấp giả và đỏ nếu nó khác 1.

Ba lý do chọn hướng này:

1. Chạy tool hộ biến thư viện thành **nửa vòng lặp agent**, và nửa vòng lặp thì bao
   giờ cũng có người đóng nốt nửa kia. Trần nằm ở đâu, ai đếm, ai trả tiền cho lượt
   thứ tư? Cả gói `aiapi` dựng lên quanh câu "mọi lời gọi đều trả về `Usage`" — mà
   `Usage` của lượt nào, khi một lần gõ lệnh thành sáu lượt gọi?
2. `sagent` đã có một đường chạy-thứ-gì-đó: flow, với `internal/flow` và bảng quyền
   plugin — có duyệt, có tường quyền, có sổ. Cho thư viện API mọc thêm đường chạy
   lệnh **thứ hai** ở dưới đáy, không qua bảng quyền nào, là mở cửa sau cho chính
   dự án này.
3. Tên hàm và tham số là **chữ do model sinh**. Thứ đó phải bị người gọi soi, không
   phải được một hàm thư viện lặng lẽ thi hành.

**Ai là người gọi đầu tiên:** `sagent api --tool` (`cmd/sagent/api_suyluan_tool.go`,
hàm `apiGoiTool`). Nó nạp định nghĩa từ file JSON đúng khuôn giao thức, gọi `GoiTool`,
in tên hàm + tham số + id nguyên văn, kèm câu "sagent KHÔNG chạy chúng" — câu đó bị
test bắt buộc phải có, vì người đọc thấy một lời gọi hàm in ra màn hình sẽ mặc định
là nó đã chạy.

**Về ĐIỂM MÙ đã biết của bảng năng lực.** Đề bài nói đúng: bảng dò phía dự án bằng
reflection nên thêm trường `tools` là ô lật ✓ kể cả khi không ai dùng. Tôi làm hai
việc, và chỉ việc thứ hai mới thật sự đóng được lỗ:

- Thu hẹp một nấc: `phepDoMaNguon` nay hỏi thêm câu thứ ba — `KetQua` **có chỗ chứa**
  giá trị không (`coTruongGo`). Áp cho cả `goi-tool` lẫn `reasoning`. Bảng vẫn không
  trả lời được "có ai chép giá trị đi không", và bình luận trong `nangluc.go` nói
  thẳng ra điều đó thay vì để người sau tưởng lỗ đã bịt.
- Đóng lỗ bằng bài kiểm **đi hết đường, hai chiều**, theo đúng khuôn
  `TestSuyLuanDiHetDuongToiKetQua`: `ToolHam(...)` → thân JSON **thật trên dây**
  (nhà cung cấp giả đọc lại và kiểm `name`/`description`/`required`/`tool_choice`)
  → `tool_calls` → `KetQua.ToolCalls` → tên hàm + tham số đã giải mã.

### Bằng chứng test ĐỎ khi gỡ phần sửa ra

Không tin lời khai; đo bằng cách gỡ từng mắt xích rồi chạy lại (mọi lần đều khôi phục
nguyên trạng sau khi đo):

| Gỡ cái gì | Bài kiểm | Kết quả |
|---|---|---|
| bỏ `ToolCalls: ph.Choices[0].Message.ToolCalls` | `TestToolDiHetDuongTuDinhNghiaToiKetQua` | ĐỎ |
| `Tools: tools` → `Tools: nil` (không gửi tool lên dây) | `TestToolDiHetDuongTuDinhNghiaToiKetQua` | ĐỎ |
| bỏ `SuyLuan: ph.Choices[0].Message.SuyLuan` | `TestSuyLuanDiHetDuongToiKetQua` | ĐỎ |
| bỏ gom phần nghĩ ở đường stream | `TestSuyLuanDiHetDuongOCaDuongStream` | ĐỎ |
| bỏ `suy_luan` khỏi mẩu tổng kết SSE | `TestMauTongKetStreamMangPhanSuyLuan` | ĐỎ |
| bỏ `suy_luan` khỏi thân JSON | `TestApiAITraVePhanSuyLuan` | ĐỎ |
| trang thôi gọi `veSuyLuan` | `TestTrangVeDuocPhanSuyLuan` | ĐỎ |
| CLI thôi giải thích ô trống | `TestBatCoMaKhongCoPhanNghi…` | ĐỎ |
| CLI thôi nhắc tên cờ | `TestKhongBatCoVanNhac…` | ĐỎ |
| CLI thôi nói "sagent KHÔNG chạy tool" | `TestLoiGoiToolHienRaDayDu…` | ĐỎ |
| CLI thôi in tên hàm + tham số | `TestLoiGoiToolHienRaDayDu…` | ĐỎ |

Lần đo thứ mười một ban đầu ra **XANH** — nhưng vì tôi gỡ nhầm chỗ (sửa vế sau của
câu, trong khi test canh vế trước). Gỡ đúng mốc thì nó ĐỎ. Ghi lại vì đây là cách một
"bằng chứng test đỏ" giả có thể ra đời: gỡ nhầm chỗ rồi kết luận về bài kiểm.

### Nghiệm thu bằng mã thoát (không lọc chữ)

```
go build ./...  → BUILD_EXIT=0
go vet ./...    → VET_EXIT=0
go test ./...   → TEST_EXIT=0   (25 gói `ok`, 0 dòng FAIL)
```

Bài canh định kỳ gọi nhà cung cấp thật đã có, gated đúng lệ sẵn có:
`SAGENT_E2E_TOOL=1 go test ./internal/aiapi/` (`TestE2EToolThatTuNhaCungCap`, cố ý
dùng grok-4.5 với `tool_choice=required` để đo đúng đường khó).

---

## 2. Sự cố

**a. Hai bài kiểm đỏ ngay khi thêm `tools` — và đó là đúng thiết kế.**
`TestPhepDoMaNguonSoiKieuThat` và `TestLamDuocTraVeLyDoDocDuoc` neo vào sự thật
"`yeuCau` CHƯA có `tools`", kèm bình luận "ngày ai đó thêm vào thì bài này đỏ — và đỏ
ĐÚNG LÚC, vì lúc đó bảng phải được đo lại". Tôi **dời neo sang sự thật mới chứ không
gỡ neo**: nay nó canh cả ba mắt của đường tool (gửi được / đọc được / có chỗ chứa),
và câu "lý do từ chối phải nói được route nào, vướng gì" được chuyển sang ô
`dau-vao-anh` — ô còn vướng ở phía dự án — để vế đó vẫn có người canh.

**b. `gofmt -w` trên cả thư mục đụng vào file không liên quan.** Nó đổi căn lề bình
luận trong `internal/aiapi/suckhoe.go` (file tôi không sửa gì). Đã `git checkout --`
trả lại và từ đó chỉ `gofmt` đúng file đã sửa. Sáu file khác hiện `M` trong
`git status` nhưng `git diff --numstat` **rỗng** — chỉ là CRLF/LF, đã trả về nguyên
trạng.

**c. Cờ bị rút SAU khi tách tên route — bắt được lúc tự đọc lại.**
`sagent api --suy-luan grok "câu hỏi"`: `--suy-luan` không khớp tên route nào nên cả
dãy bị coi là câu hỏi, và chữ `grok` đi thẳng vào prompt. Lượt gọi vẫn chạy, vẫn tính
tiền, chỉ là **hỏi sai câu qua sai route** — hỏng đúng kiểu không ai để ý. Đã đảo thứ
tự (rút cờ trước, tách route sau) và canh bằng `TestRutCoTruocKhiTachTenRoute`.

**d. Không có `internal/api`, `--tool` không có route dự phòng.** Bộ chuyển route dự
phòng nằm trong `internal/api/api.go` (vùng của agent khác) và nó gọi `aiapi.Goi` —
đường không mang tool. Mượn đường đó thì lượt dự phòng gửi đi một yêu cầu **không có
`tools`**, model trả lời bằng chữ, người dùng nhận về "model không đòi gọi tool nào"
— một câu sai sinh ra từ một chỗ chuyển route im lặng. Tôi chọn **không có dự phòng
còn hơn có một cái dự phòng nói dối**, và in ra dòng nói rõ điều đó sau mỗi lượt
`--tool`. Cũng chặn `--stream --tool` đi chung, vì đường stream chưa mang tool đi được.

**e. Hai cái bẫy môi trường.** `os.UserHomeDir()` trên Windows đọc `%USERPROFILE%`
chứ không đọc `$HOME`, nên lần chạy thử CLI đầu tiên đi tìm key ở HOME thật. Và một
file `html.py` lạc trong `%TEMP%` che mất `http.server` của Python, làm nhà cung cấp
giả chết lúc khởi động — phải chạy nó từ thư mục khác.

---

## 3. Bước tiếp theo

1. **Ghi món nợ đã trả vào `docs/DO-LUONG.md` và `docs/SO-NO-DO-LUONG.md`** — hai file
   này ngoài vùng của tôi. Nội dung cần ghi: `reasoning` nay có ba mặt đọc; `goi-tool`
   đã lật ✓ cho cả hai route; mục "ĐÃ ĐO, CHƯA LÀM — tool/reasoning" trong MASTER-PLAN
   nay chỉ còn `vision` và `structured-output`.
2. **Đường tool trên dashboard.** Hiện `--tool` chỉ có ở CLI. Luật ngang quyền canh
   **action** chứ không canh cờ, nên không có test nào đỏ — nhưng chênh lệch là thật.
   Nếu điều phối muốn đóng, cần **một dòng** trong `internal/api/api.go`, danh sách
   `Actions` (khoảng dòng 89, ngay cạnh `"api.call"`):

   ```go
   	"api.tool",
   ```

   kèm một hàm `func (a *API) AICallTool(ctx context.Context, route, prompt string, tools []aiapi.Tool, chon string) (aiapi.KetQua, error)` gọi thẳng `aiapi.GoiTool`
   (KHÔNG mượn bộ chuyển route dự phòng — xem sự cố **d**). **Tôi không xin dòng này
   trong lượt này**: chưa có mặt web nào cần tới, và một action không ai gọi là một
   dòng nữa trong hợp đồng phải nuôi.
3. **Hai ô còn vướng ở phía dự án của grok-4.5**: `dau-vao-anh` (`tinNhan.Content`
   còn là chuỗi thuần) và `dau-ra-co-cau-truc` (`yeuCau` chưa có `response_format`).
   Nhà cung cấp đã đo được là **làm được cả hai** (761 và 980 token). Cùng một hình
   dạng việc như lượt này, và `donangluc.go` đã có sẵn phép đo cho cả hai.
4. **Chạy `SAGENT_E2E_TOOL=1` và `SAGENT_E2E_SUYLUAN=1` một lượt** để xác nhận trên
   mạng thật. Tôi không chạy: nó tiêu tiền thật, và đề bài không yêu cầu.

---

## 4. Bảng

| Việc | Model | Effort |
|---|---|---|
| Đọc lõi `aiapi` + bảng năng lực + hai mặt CLI/web trước khi sửa | Opus 5 (1M) | Cao |
| `internal/aiapi/suyluan.go` — `DocSuyLuan`, chỗ trả lời "rỗng nghĩa là gì" | Opus 5 (1M) | Cao |
| `internal/aiapi/tool.go` — `GoiTool` + quyết định không chạy tool | Opus 5 (1M) | Cao |
| Vá đường stream để gom `delta.reasoning_content` | Opus 5 (1M) | Trung bình |
| CLI: `--suy-luan`, `--tool`, `--tool-chon`, dựng chữ thuần để test được | Opus 5 (1M) | Trung bình |
| Dashboard: `suy_luan` ở cả hai thân + khối `#ai-nghi` + `veSuyLuan` | Opus 5 (1M) | Trung bình |
| Bài kiểm đi hết đường (aiapi 9 bài, cmd 8 bài, dash 4 bài) | Opus 5 (1M) | Cao |
| Dời neo hai bài kiểm cũ sang sự thật mới | Opus 5 (1M) | Trung bình |
| Đo 11 lần gỡ-mã-để-xem-test-có-đỏ | Opus 5 (1M) | Trung bình |
| Chạy thật CLI với nhà cung cấp giả (4 nhánh đầu ra) | Opus 5 (1M) | Thấp |

---

## 5. Nhận xét tự do

**Điều đáng nói nhất không phải là hai tính năng, mà là chỗ chúng suýt dừng lại.**

Cả hai món trong lượt này đều đã "gần xong" từ trước. `KetQua.SuyLuan` có mã, có
bài kiểm xanh, có ô ✓ trên bảng năng lực — và không ai đọc nó. `goi-tool` có bộ đo
chạy thật với hai nhà cung cấp, có sổ số đo, có chẩn đoán chỉ đúng dòng phải sửa — và
lời gọi của dự án chưa bao giờ mang tool đi. Cái chung của hai món: **thứ đo được đã
đủ, thứ thiếu là đoạn cuối cùng, và đoạn cuối cùng thì không có ai đo nó.**

Bảng năng lực là một thiết kế tốt bị chính điểm mạnh của nó phản lại. Nó dò bằng
reflection nên không bịa được — nhưng đúng vì thế mà nó chỉ trả lời được câu *"kiểu có
trường đó không"*, và câu đó **nghe rất giống** câu *"tính năng đó có chạy không"*.
Khoảng cách giữa hai câu ấy chính là chỗ bốn tính năng nằm chết trong ngày 22/08. Tôi
thu hẹp được một nấc (hỏi thêm `KetQua` có chỗ chứa không) nhưng cố ý **không** viết
bình luận kiểu "nay đã bịt": nấc đó vẫn không trả lời được "có ai chép giá trị đi
không". Thứ duy nhất trả lời được là một bài kiểm chạy hết đường, và cái giá của nó là
phải dựng một nhà cung cấp giả rồi đọc lại thân JSON trên dây — đắt hơn reflection rất
nhiều, nên nó sẽ luôn có lý do để không được viết.

Chuyện `tool_choice` đáng để lại một dòng riêng. Cùng một endpoint, cùng một dòng mã:
grok nuốt `required` ngon lành, deepseek trả HTTP 400. Bộ đo trong `donangluc.go` đã
gặp và xử đúng — đo lại bằng `auto` rồi ghi **cả hai** quan sát vào bằng chứng. Tôi cố
ý **không** cho `GoiTool` bắt chước: bộ đo được phép tiêu thêm một lượt để có kết luận
đúng, còn trong thư viện thì lượt thứ hai là tiền của người dùng, tiêu mà không hỏi.
Cùng một tình huống, hai chỗ, hai cách xử ngược nhau — và cả hai đều đúng. Đó là loại
chi tiết mà một bản "cho nhất quán" sẽ san phẳng mất.

Còn câu hỏi vòng lặp agent: tôi nghĩ đề bài hỏi đúng chỗ đau nhất. Cái cám dỗ ở đây
rất cụ thể — thêm mười lăm dòng nữa là `GoiTool` tự chạy tool, nhét kết quả vào
`messages`, hỏi lại, và demo trông "thông minh" hẳn lên. Nhưng lúc đó `KetQua.Usage`
trả về là của lượt nào? Cả gói này dựng lên quanh đúng một lời hứa — mọi lời gọi đều
nói ra nó tiêu gì — và mười lăm dòng đó phá lời hứa ấy trong im lặng, ở tầng thấp
nhất, nơi không mặt nào nhìn thấy. Nên trần vòng lặp không chỉ được viết trong bình
luận: có một bài kiểm đếm số lượt chạm mạng và đỏ nếu nó khác 1. Bình luận thì mục
ruỗng; con số thì không.
