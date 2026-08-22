# Báo cáo: cập nhật `docs/MASTER-PLAN.md` cho đúng với mã nguồn ngày 22/08

Nhánh `sagent/kehoach-22-08`, tách sạch từ `main` (`git rev-list --count main..HEAD` = **0**
lúc tạo). Ngày 22/08/2026.

Vùng đụng tới, đúng ranh giới được giao: `docs/MASTER-PLAN.md` ·
`internal/dash/web/docs/{MASTER-PLAN.md, master-plan.html}` · `master-plan.html`.
**Không** đụng `internal/flow/*`, `internal/dash/*` (ngoài `docs/`), `internal/api/*`,
`internal/aiapi/*`, `cmd/sagent/*`, `docs/DO-LUONG.md`, `docs/SO-NO-DO-LUONG.md`.

Nghiệm thu, chạy **từng lệnh riêng biệt**, cả ba xanh:

```
go build ./...   EXIT=0
go vet ./...     EXIT=0
go test ./...    EXIT=0   (28 gói, 0 gói FAIL)
```

`TestTrangKeHoachChoDuOViec` — bài đếm ô việc giữa `.md` và `.html` — **PASS**, cùng
`TestTrangKeHoachKhopBanSinh` và `TestBanSinhKeHoachOnDinh`.

---

## 1. Đã làm

### 1.1 Số đo: trước / sau

| | `[x]` | `[~]` | `[ ]` | `[!]` | Điểm | % |
|---|---|---|---|---|---|---|
| **Bảng cũ KHAI** (soát 21/08) | 87 | 10 | 2 | 0 | 92,0/99 | **93%** |
| **Đếm lại bản `.md` TRƯỚC lượt này** | 90 | 8 | 1 | 0 | 94,0/99 | **95%** |
| **SAU lượt này** | 96 | 4 | 0 | 0 | 98,0/100 | **98%** |

**Ba dòng chứ không hai, và đó là điểm chính của báo cáo.** Khoảng cách giữa dòng 1 và
dòng 2 **không phải việc làm hôm nay** — nó là bảng tổng đã trôi khỏi phần chữ trước khi
tôi bắt đầu. Gộp hai khoảng cách ấy lại thành một câu "93% → 98%" là báo công cho một
lỗi kế toán.

**Bao nhiêu ô đổi dấu: 5 ô đổi dấu + 1 ô mới thêm = 6 ô động.**

| # | Ô | Ở đâu | Trước | Sau | Bằng chứng dẫn được |
|---|---|---|---|---|---|
| 1 | Alias `tk`/`ccswitch` | Bước 0 | `[ ]` | `[x]` | `install/cai-dat.ps1:75-76` (`$TenAlias`, `$ShimNoiDung`) |
| 2 | Tầng redaction chung | Pha 0 | `[~]` | `[x]` | `internal/redaction/redaction.go:263`, `quet_repo_test.go:31` |
| 3 | Capability matrix | Pha 0 | `[~]` | `[x]` | `internal/aiapi/nangluc.go` (617 dòng), `donangluc.go` (493 dòng) |
| 4 | Node built-in 10/10 | Pha 3 | `[~]` | `[x]` | `internal/flow/flow.go:38,40`, `route.go`, `merge.go` |
| 5 | Approval gate mặt web | Pha 5b | `[~]` | `[x]` | `internal/dash/web/index.html:1910`, `ngangquyen_ui_test.go:21` |
| 6 | Bảng quyền plugin ra web | Pha 5b | *(chưa có ô)* | `[x]` | `internal/dash/plugin_api.go:86-88` |

Ô #6 là ô **mới thêm vào mẫu số**, nên mẫu số đi từ 99 lên 100. Nói ra vì một mẫu số
đổi mà không nói là cách rẻ nhất để làm đẹp một tỷ lệ.

### 1.2 Ba ô bảng cũ đã bỏ sót (lệch 2 điểm trước khi tôi động vào)

Ba ô này đã được tick **trong phần chữ** nhưng **không ai đi sửa bảng tổng**:

| Ô | Ở đâu | Tick từ commit |
|---|---|---|
| Plugin model đầy đủ | Pha 3 | `ba0bda3` (22/08) |
| Threat model | Pha 0 | `2e6c48a` (22/08) |
| Duyệt / Từ chối trên web | Pha 5c | `3712d99` (21/08) |

Đề bài giao cho tôi ghi hai ô đầu là "trước ghi `[ ]` CHƯA LÀM" — mở file ra thì cả hai
**đã là `[x]` rồi**. Tôi giữ nguyên dấu, chỉ đối chiếu bằng chứng trong mã
(`internal/plugin/{manifest,rpc,chay,quyen}.go`, `docs/MO-HINH-DE-DOA.md` 782 dòng) rồi
đi sửa **bảng tổng** — chỗ thật sự sai. Đây là lý do việc đầu tiên tôi làm là **đếm lại
bằng script** thay vì tin danh sách được giao.

### 1.3 Từng mục, và nó nói sai theo chiều nào

Đề bài dự đoán mọi thứ lệch theo chiều **bi quan**. Đúng với 9/10 mục. **Một mục lệch
theo chiều TÔ HỒNG**, và đó là mục đáng giá nhất của lượt soát này:

**Bảng quyền plugin ở Pha 3 khai thiếu một ô `khong-chan-duoc`.** Kế hoạch viết "hai chỗ
này phải nói thẳng" và liệt kê `ghi-thu-muc-lam-viec` (không chặn được) + `mang` (chưa
đo). Mở `internal/plugin/quyen.go:56-80` ra thì `thu-muc-lam-viec` **cũng đã bị hạ xuống
`khong-chan-duoc` ngày 22/08** — nó từng khai `chan-that` trong khi plugin **không xin
quyền nào** vẫn đọc được 33 byte, ghi được file, liệt kê được thư mục dự án
(`TestBangQuyenThuMucPhaiKhopVoiThucTeChamDuoc`). Kế hoạch đang trình bày hàng rào an
ninh **rộng hơn thực tế**. Đã sửa thành **5 quyền: 2 `chan-that` · 2 `khong-chan-duoc` ·
1 `chua-do`**, kèm câu chốt cũ *"cách chặn thật đang có là không cấp `thu-muc-lam-viec`"*
được đánh dấu **đã bị phép đo bác bỏ** (nó cũng đã bị gỡ khỏi mã).

Chín mục còn lại, theo chiều bi quan:

- **Alias** — "lý do hoãn đã hết hạn" nay thành đã làm, kèm **lý do CHỌN shim `.cmd`**
  và bỏ hai cách kia: bản sao `.exe` (+32 MB) và hard link đều **chạy binary CŨ sau khi
  nâng cấp, im lặng**. Giữ lại một nút chưa đóng: tương thích **cờ** của `ccswitch` vẫn
  là suy đoán vì chưa có bản v1 thật để mở ra xem.
- **Redaction** — thêm số đo: 2.060 lần lộ tên tài khoản → **0**, email 179 → **0**, chữ
  ký thinking giữ nguyên **489**, JSON hỏng **0/6.092**; và lý do kiến trúc vì sao chặn ở
  **cửa ra** chứ không ở đường ghi.
- **Bảng năng lực API** — thêm quyết định đáng nói nhất của nó: **mỗi ô trả lời HAI câu**
  (khách / nhà cung cấp) + cột "vướng ở đâu", vì trộn hai câu sẽ đẩy người đọc đi đổi nhà
  cung cấp trong khi chỗ hỏng nằm trong repo này.
- **Ba mảnh flow** — thay ba dòng `⬜` bằng số đo thật của lượt #53–#58 (0 token): artifact
  đưa được **1200/1200 dòng** so với **651/1200** qua `{{steps.x.output}}` — **mất 45,75%
  ngay ở tầng LƯU**, và mất phần **cuối**.
- **Node `route`/`merge`** — kèm hai điều dễ hiểu nhầm: `route` tách ra vì năm bước cùng
  đường có thể rơi sang **năm mô hình khác nhau** mà bảng chỉ ghi "model"; `merge`
  **KHÔNG phải gộp nhánh git**, và lời hẹn "chờ cơ chế merge an toàn" được ghi thẳng là
  **sẽ không bao giờ tới**.
- **429** — kèm ba dữ kiện vì sao chờ-rồi-thử-lại đúng hơn nhảy-ngay, và **cố ý không mở
  rộng sang 5xx**.
- **Bốn trần đồng thời** — ghi rõ mặc định `chung 4 · harness 3 · provider 3 · hồ sơ 2`
  buộc phải trải ra ít nhất hai tài khoản, và quy ước **0 = tắt**.
- **Approval gate 5b** — ô này giữ `[~]` thêm **một ngày** sau khi cả hai nửa đã xanh.
- **Mục 12 (Việc còn treo)** — mục "Máy/VM Linux" đã hết hiệu lực (nhánh Linux bỏ khỏi
  phạm vi), mục "daemon + SQLite" đã chốt và đã làm xong từ Pha 2.

### 1.4 Thứ BỊ CHẶN, ghi rõ thiếu cái gì bên ngoài

Ô `[!]` vẫn **rỗng**, và tôi cố ý không tạo ra một ô `[!]` nào. Lý do: luật ở đầu mục 7
nói `[!]` là **cả ô** bị chặn, mà thứ bị chặn hôm nay đều nằm **bên trong** một ô còn
nửa kia là nợ của chính dự án. Nâng cả ô lên `[!]` là đổ cho bên ngoài một khoản nợ của
mình — tô hồng, đúng chiều đắt hơn. Thay vào đó đã thêm một **bảng riêng** ở cuối mục 7:

| Bị chặn ở đâu | Thiếu THỨ GÌ bên ngoài |
|---|---|
| Pha 0 · ô API — ba giao thức còn lại | Khoá thật của **chính nhà cung cấp**: `api.anthropic.com` (Anthropic Messages) · `generativelanguage.googleapis.com` (Gemini native) · `api.openai.com` (OpenAI Responses). Máy chỉ có khoá **nhà bán lại** `modelapi.vn`, đã đo là **401 ở `api.deepseek.com`** |
| Pha 4 · OpenRouter | Khoá API thật của openrouter.ai |
| Pha 4 · Ollama | **Khoá VÀ phần mềm** — `ollama` chưa cài trên máy này, không có endpoint cục bộ để trỏ tới |

Và bốn ô `[~]` còn lại đều có một bảng ghi rõ **phần nào xong, phần nào chưa** — đúng
luật của dấu `[~]`.

### 1.5 Hai trang HTML — sinh lại, không sửa tay

```
go run ./tools/sinhkehoach/cmd/sinhkehoach
  -> internal/dash/web/docs/master-plan.html (148 KB)

python tools/md2html.py docs/MASTER-PLAN.md master-plan.html "Master Plan"
  -> master-plan.html (131 KB)
```

Không một ký tự `.html` nào gõ tay. Đối chiếu số ô việc chảy được sang trang:

| | ô việc |
|---|---|
| `docs/MASTER-PLAN.md` (đếm bằng script) | 96 `[x]` + 4 `[~]` = **100** |
| `internal/dash/web/docs/master-plan.html` | 96 `class="task done"` + 4 `class="task part"` = **100** |
| `master-plan.html` (gốc repo) | **100** |

### 1.6 Bản nhúng khớp từng dòng

```
diff docs/MASTER-PLAN.md internal/dash/web/docs/MASTER-PLAN.md
EXIT=0          (1411 dòng = 1411 dòng, không một dòng lệch)
```

Đây là chỗ đã từng lệch 95 dòng, nên tôi kiểm bằng `diff` chứ không bằng cảm giác —
và kiểm **hai lần**: một lần ngay sau khi chép, một lần sau khi sinh xong cả hai trang.

Thay đổi tổng cộng: **4 file, +1.693 / −383 dòng**; riêng `MASTER-PLAN.md` từ 1.112 lên
**1.411 dòng**.

---

## 2. Sự cố

### 2.1 Bộ sinh đọc bản NHÚNG, không đọc bản gốc — và KHÔNG bài kiểm nào canh hai bản `.md`

Đây là sự cố đáng kể nhất, và nó suýt làm hỏng chính lượt việc này.

`tools/sinhkehoach/cmd/sinhkehoach` mặc định lấy nguồn là
`internal/dash/web/docs/MASTER-PLAN.md`, **không phải** `docs/MASTER-PLAN.md`. Nghĩa là
nếu tôi sửa bản gốc rồi chạy thẳng bộ sinh:

- trang nhúng được sinh lại **từ bản cũ**, không có một dòng nào tôi vừa viết;
- `TestTrangKeHoachKhopBanSinh` vẫn **XANH**, vì nó chỉ so `web/docs/*.md` với
  `web/docs/*.html` — hai thứ vẫn khớp nhau, cùng cũ;
- `TestTrangKeHoachChoDuOViec` cũng **XANH**, vì số ô hai bên vẫn bằng nhau.

Đo lại để chắc chắn: `grep -rn "MASTER-PLAN.md" --include=*.go .` cho thấy **không một
bài kiểm nào đối chiếu `docs/MASTER-PLAN.md` với `internal/dash/web/docs/MASTER-PLAN.md`**.
Chính cái lệch 95 dòng mà đề bài nhắc **có thể tái phát y hệt, và mọi test vẫn xanh**.
Bộ ba test hiện có canh khâu `.md nhúng → .html nhúng` rất chặt, nhưng khâu
`.md gốc → .md nhúng` thì **không ai canh**.

Cách tôi tránh: chép trước, `diff` xác nhận `EXIT=0`, rồi mới sinh. Đó là kỷ luật, và
kỷ luật thì không có răng — xem mục 3.

### 2.2 Danh sách việc được giao lệch với file thật (2 mục)

Đề bài ghi plugin model và threat model là "trước ghi `[ ]` CHƯA LÀM". Mở file ra thì cả
hai **đã là `[x]`** (commit `ba0bda3`, `2e6c48a`, cùng ngày). Nếu tôi tin danh sách và
"sửa `[ ]` thành `[x]`" thì không tìm thấy chuỗi cần sửa, hoặc tệ hơn là đi thêm một ô
trùng. Chỗ thật sự sai là **bảng tổng**, mà danh sách không nhắc tới.

Điều này đúng bằng đúng luật mà đề bài đặt ra: *tên mục nghe giống một thứ đã làm không
phải bằng chứng*. Nó áp cho **cả danh sách việc**, không chỉ cho kế hoạch. Vì vậy việc
đầu tiên tôi làm không phải sửa file mà là đếm lại toàn bộ bằng script, và đó là cách
chênh lệch 87/10/2 ↔ 90/8/1 lộ ra.

### 2.3 Python in ra tiếng Việt thì chết ngay ở dòng `print` đầu tiên

Script đếm ô chạy đúng nhưng đổ:

```
UnicodeEncodeError: 'charmap' codec can't encode characters in position 1-2
  File "C:\Program Files\Python313\Lib\encodings\cp1252.py"
```

Không phải lỗi đọc file (`open(..., encoding='utf-8')` đã đúng) mà là lỗi **ghi ra
stdout**: Python trên Windows lấy codepage của console, ở đây là `cp1252`, và "Bước 0"
không nằm trong bảng đó. Nguy hiểm ở chỗ nó chết **sau khi đã đếm xong**, tức nếu tôi
đọc lướt cái traceback thành "script sai" thì đã đi viết lại một script vốn đúng. Chữa
bằng `PYTHONIOENCODING=utf-8` đứng trước lệnh.

Ghi lại vì mọi phép đo trong dự án này đều in tiếng Việt.

### 2.4 Ghi chú miễn trừ trong `uxui_test.go` đã hết hiệu lực nhưng vẫn nằm đó

`internal/dash/uxui_test.go:81` vẫn miễn trừ `master-plan.html` với lý do
*"chua co khau sinh lai — don tay se lech voi ban .md"*. Khâu sinh lại **đã có** từ
21/08 (`tools/sinhkehoach`), nên lý do miễn trừ không còn đúng. File đó nằm ngoài vùng
được giao nên tôi **không sửa**; ghi ra ở mục 3.

---

## 3. Bước tiếp theo

Xếp theo mức đáng làm, ai làm cũng được:

1. **Thêm bài kiểm đối chiếu `docs/MASTER-PLAN.md` với `internal/dash/web/docs/MASTER-PLAN.md`**
   (mục 2.1). Đây là lỗ duy nhất còn lại trên đường từ bản gốc tới trang web, và nó là
   đúng cái lỗ đã đẻ ra lệch 95 dòng. Rẻ: một `bytes.Equal` cộng câu chỉ cách chữa.
   Cân nhắc thêm: cho `cmd/sinhkehoach` **tự chép** bản gốc sang bản nhúng trước khi
   sinh, để không còn hai bước tay.
2. **Xem lại miễn trừ `master-plan.html` trong `uxui_test.go`** (mục 2.4) — lý do miễn
   trừ đã hết hiệu lực, nên hoặc gỡ miễn trừ, hoặc viết lại lý do cho đúng hiện trạng.
3. **Đóng mảnh cuối của engine flow**: route theo **capability/giá**. Bảng năng lực API
   đã có từ 22/08 nhưng bộ chọn đường vẫn đi theo TÊN — hai thứ này đang nằm cạnh nhau
   mà chưa nối. Đóng xong thì Pha 3 lên 100%.
4. **Bốn năng lực API bị chặn ở PHÍA DỰ ÁN**: thêm `tools` + `response_format` vào
   `aiapi.yeuCau`, đọc `tool_calls` + `reasoning_content` ở `aiapi.phanHoi`, cho
   `tinNhan.Content` nhận mảng. Đây là nợ của repo, không phải của nhà cung cấp — đo
   22/08 đã chỉ đích danh.
5. **Đưa bốn trần đồng thời vào bộ chạy flow**: `internal/api/api.go:1677` vẫn truyền
   một con số duy nhất cho `flow.Runner`. Bốn trần mới canh `FleetStart`, không canh số
   bước song song trong một lượt flow.
6. **Đo một cú 429 THẬT** — cả đường mã 429 hiện chỉ được canh bằng test.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Tạo nhánh sạch, xác nhận `main..HEAD` = 0 | claude-opus-5[1m] | thấp |
| Đếm lại 99 ô bằng script, phát hiện bảng tổng lệch 2 điểm | claude-opus-5[1m] | trung bình |
| Đọc 8 báo cáo 22/08 + mục 22/08 của `DO-LUONG.md` để lấy bằng chứng | claude-opus-5[1m] | cao |
| Kiểm chứng 10 mục **trong mã** (`grep`/`sed` vào `internal/*`, `install/*`) trước khi tick | claude-opus-5[1m] | cao |
| Viết lại 6 ô + 5 khối trạng thái của mục 7 | claude-opus-5[1m] | cao |
| Bắt chỗ kế hoạch **tô hồng** bảng quyền plugin, hạ xuống cho đúng mã | claude-opus-5[1m] | cao |
| Viết lại bảng đếm + điểm một dòng + bảng "thiếu thứ gì bên ngoài" | claude-opus-5[1m] | trung bình |
| Cập nhật mục 12 cho khỏi nói sai | claude-opus-5[1m] | thấp |
| Chép bản nhúng, `diff` xác nhận, sinh lại hai trang | claude-opus-5[1m] | thấp |
| Nghiệm thu ba lệnh riêng + chạy riêng bộ ba test trang kế hoạch | claude-opus-5[1m] | thấp |
| Viết báo cáo này | claude-opus-5[1m] | trung bình |

---

## 5. Nhận xét tự do

**Thứ khó chịu nhất của lượt này không phải kế hoạch nói sai về mã, mà là kế hoạch nói
sai về chính kế hoạch.** Phần chữ của mục 7 đúng; bảng tổng ở cuối mục 7 thì lệch 2
điểm so với phần chữ nằm ngay phía trên nó, trong cùng một file, cách nhau vài trăm
dòng. Lượt soát 21/08 chữa bệnh "phần chữ trôi khỏi mã" và làm rất kỹ. Đúng một ngày
sau, cùng lớp bệnh tái phát ở một tầng nông hơn: **bảng tổng trôi khỏi phần chữ**. Ba
người khác nhau tick ba ô ở ba nhánh khác nhau, mỗi người tick đúng ô của mình, và không
ai thấy mình có phần trong cái bảng ở cuối. Không ai làm sai, mà kết quả vẫn sai.

Điều đó nói một chuyện về hình dạng của lỗi: **cứ chỗ nào có hai bản của cùng một sự
thật là chỗ đó sẽ lệch**, và tầng nào cũng vậy. Mã ↔ kế hoạch, phần chữ ↔ bảng tổng, bản
gốc ↔ bản nhúng, `.md` ↔ `.html`. Dự án này đã dựng răng cho ba trong bốn chỗ: bảng năng
lực đo bằng `reflect` thay vì khai tay, `TestTrangKeHoachKhopBanSinh` canh `.md` ↔ `.html`,
luật ngang quyền canh API ↔ CLI ↔ UI. Chỗ thứ tư — hai bản `.md` — vẫn đang chạy bằng
kỷ luật, và mục 2.1 cho thấy kỷ luật đó **im lặng** khi hỏng: bộ sinh vẫn chạy, ba bài
kiểm vẫn xanh, chỉ có nội dung là cũ. Đó chính xác là hình dạng của cái lệch 95 dòng lần
trước.

**Về chiều lệch.** Đề bài dự đoán mọi thứ lệch theo chiều bi quan và đúng 9/10 mục — cái
giá của bi quan là việc "còn phải làm gì" bị chôn dưới những dòng nhìn giống nhau. Nhưng
mục thứ 10 lệch theo chiều **tô hồng**, và nó lệch ở đúng chỗ đắt nhất: một bảng quyền
plugin nói hàng rào rộng hơn thực tế. Ô `thu-muc-lam-viec` khai `chan-that` trong khi
plugin không xin quyền nào vẫn đọc được file, ghi được file, liệt kê được thư mục. Một
người đọc kế hoạch để quyết định "có nên cho plugin lạ chạy không" sẽ quyết định sai. Đó
là lý do tôi giữ nguyên luật *không dẫn được bằng chứng thì không tick* theo cả hai
chiều, kể cả khi đề bài đã đưa sẵn danh sách 10 mục "đã vào main" — và cũng là lý do
việc đầu tiên tôi làm là mở mã ra đếm chứ không phải mở file ra sửa. Nếu tôi tin danh
sách, hai mục đầu đã bị tick trùng và ô plugin tô hồng kia đã ở lại thêm một ngày nữa.

**Còn một điều đáng nói về con số 98%.** Nó cao, và cao là chỗ dễ đọc nhầm nhất. Mẫu số
của nó là 100 ô tick của mục 7 — mà Pha 4, 6, 7 **không dùng ô tick**, nên chúng không
có mặt trong con số ấy; và bốn ô `[~]` còn lại chứa những thứ như "ba giao thức API còn
lại" mà một ô nửa điểm không phản ánh nổi khối lượng. 98% nghĩa là *98% số ô tick của
mục 7*, không phải *98% dự án*. Tôi đã ghi câu đó vào ngay dưới bảng, vì một con số tròn
trịa không kèm mẫu số là cách một tài liệu trung thực bắt đầu nói dối.
