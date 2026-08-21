# Báo cáo — đóng ô nợ C1 phần CODEX

- **Ngày**: 21/08/2026
- **Nhánh**: `sagent/phu-1`
- **Ô nợ**: `docs/SO-NO-DO-LUONG.md` mục **C1** — *"Codex và Cursor CHƯA ĐO cách
  đọc kết quả có cấu trúc"*, phần Codex.
- **Trạng thái**: ✅ **ĐÓNG ĐƯỢC BẰNG PHÉP ĐO**, không phải bằng `KhongLamDuoc`.

---

## 1. Đã làm

### Đo thật, không đọc `--help` rồi suy

Chạy **6 lượt `codex exec`** thật trên codex-cli 0.147.0, tài khoản thật
(`~/.codex/auth.json`), máy Windows. Cờ tìm được: **`--json`** — nguyên văn
`codex exec --help`: *"Print events to stdout as JSONL"*.

| # | Lượt chạy | Đo được gì |
|---|---|---|
| 1 | `codex exec --json … "Tra loi dung mot tu: XONG"` (stdin là ống dẫn) | **TREO 5 phút, 0 byte** — xem §2 |
| 2 | như trên, stdin nối `/dev/null` | lược đồ lượt thành công + `usage` |
| 3 | chạy một lệnh shell **hai lần** | `command_execution`, bẫy `item.started`/`item.completed` |
| 4 | chạy `exit 3` | `exit_code:3` + `status:"failed"` |
| 5 | `CODEX_HOME` = thư mục rỗng | `turn.failed` + `error.message` (401 kèm request id) |
| 6 | ghi file **không** có `--approve-for-me` | **KHÔNG có trường quyền nào** — xem §5 |

Bản ghi nguyên văn của cả 6 lượt nằm ở `docs/DO-LUONG.md` (mục *21/08 — Codex đọc
được kết quả có cấu trúc*) và được chép thành hằng số test trong
`internal/provider/ketqua_codex_test.go`.

### Mã đã đổi

| File | Đổi gì |
|---|---|
| `internal/provider/ketqua_codex.go` | **MỚI** — `docKetQuaCodex`, 160 dòng |
| `internal/provider/codex.go` | `DocKetQua` gọi bộ đọc thật; `HeadlessArgs` thêm `--json`; `Chua(NLKetQuaCoCauTruc)` → `Duoc(...)` |
| `internal/provider/ketqua_codex_test.go` | **MỚI** — 11 bài kiểm trên bản ghi THẬT |
| `internal/provider/ketqua_codex_e2e_test.go` | **MỚI** — bài canh gọi CLI thật (`SAGENT_E2E_CODEX=1`) |
| `internal/provider/trangthai_test.go` | sửa bình luận đã lạc hậu (nó còn nói Codex không đọc được) |
| `docs/DO-LUONG.md`, `docs/SO-NO-DO-LUONG.md` | ghi phép đo, đóng ô C1 |

### Bằng chứng đầu-cuối — đây mới là thứ đáng tin

Args dựng bằng **chính adapter** (`HeadlessArgs` + `ArgsThuMuc`), chạy CLI thật,
rồi đọc lại bằng **chính adapter**:

```
LỆNH: …\npm\codex.cmd [exec --json Tra loi dung mot tu: ALPHA --cd …\Temp --skip-git-repo-check]
STDOUT THẬT:
{"type":"thread.started","thread_id":"01a0231a-7c1b-7fc0-bf1c-24c7b8999c88"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"ALPHA"}}
{"type":"turn.completed","usage":{"input_tokens":17627,"cached_input_tokens":11008,
 "cache_write_input_tokens":0,"output_tokens":6,"reasoning_output_tokens":0}}

ĐỌC ĐƯỢC: TraLoi="ALPHA" CoLoi=false TokenVao=17627 TokenRa=6 ToolHong=0 Hong=""
PhanLoaiChet: "done" ""
```

**Dòng cuối là con số của cả lượt làm việc này.** Trước khi sửa nó là `""` — tức
phiên ở lại `lost` và bốn mặt điều khiển in *"chết, chưa rõ vì sao"* cho một lượt
chạy **thành công**.

### Số đo

- `go build ./...` — **xanh**.
- `go test ./...` — **xanh**, 22 gói.
- 11 bài kiểm mới, tất cả PASS; bài canh e2e PASS khi chạy thật (6,65 s).
- Chi phí phép đo: 6 lượt, ~146k token vào / ~418 token ra. **Không quy ra tiền
  được** — Codex không in giá, đúng cái ô `ChiPhiUSD` vẫn để 0.

---

## 2. Sự cố

**Có, một cái, và nó suýt làm hỏng cả phép đo.**

Lượt đo đầu tiên **treo 5 phút rồi bị giết**, `out.jsonl` **0 byte**. Không lỗi,
không output — chỉ hết giờ. Lý do nằm ở stderr:

```
Reading additional input from stdin...
```

`codex exec` thấy stdin là **ống dẫn** thì coi đó là phần prompt nối thêm và đợi
vô hạn (help: *"If stdin is piped and a prompt is also provided, stdin is appended
as a `<stdin>` block"*). Nối stdin vào `NUL` là hết.

**Dự án đã biết chuyện này** — `internal/profile/clone.go:349` nối
`c.Stdin = devNull` kèm ghi chú đúng về Codex. Nên đây **không phải lỗi mới**.
Nhưng nó là lời nhắc đắt: bất kỳ đường nào chạy Codex mà để `Stdin = nil` hoặc
ống dẫn đều **treo im lặng**, và treo im lặng là kiểu hỏng khó truy nhất.

**Một chuyện thứ hai, nhỏ hơn nhưng đúng chủ đề**: sổ nợ trỏ tới
`docs/BAO-CAO-DON-SO-NO.md` — **file đó chưa từng được commit**
(`git log --all -- docs/BAO-CAO-DON-SO-NO.md` rỗng). Tức lượt trước đã khai một
báo cáo mà nó chưa bao giờ tồn tại trong repo. Đúng cái bẫy đề bài cảnh báo: lời
khai trong phiên bay mất, chỉ commit mới là bằng chứng. Tôi **không** sửa hộ ô
đó — nó không thuộc phạm vi lượt này — chỉ ghi lại ở đây.

---

## 3. Bước tiếp theo

1. **C3 — Codex `ModelArgs`**: `codex exec --help` có `-m, --model <MODEL>`. Tôi
   **cố ý không đụng vào**: nhìn thấy cờ trong help chưa phải đo, và đúng ô C3 là
   ô *tốn tiền trực tiếp*. Đo đúng cách là chạy hai lượt với hai model khác nhau
   rồi đối chiếu, hoặc đưa tên model sai xem CLI có từ chối không (đúng cách
   Cursor đã đóng ô này).
2. **C4 — `ChiPhiUSD = 0` phải hiện kèm chữ "chưa đo"** trên mọi mặt. Nay Codex
   nhập hội Cursor: đọc được token nhưng **không** có giá. Số 0 đứng một mình đọc
   như "miễn phí".
3. **Bật bài canh e2e trong CI định kỳ** (`SAGENT_E2E_CODEX=1`) — để biết vào
   **ngày** Codex đổi định dạng, không phải ba tuần sau. Đây là việc ô C2 đã đề
   nghị cho Grok và giờ áp dụng được cho cả Codex.

---

## 4. Nên xài model gì, effort nào

| Việc | Model | Effort |
|---|---|---|
| Đo CLI lạ (chạy thật, đọc lược đồ, tìm bẫy) | Opus 5 | high |
| Viết bộ đọc + test từ bản ghi đã có | Sonnet 5 | medium |
| Đóng ô C3 (`ModelArgs`) — có rủi ro tốn tiền | Opus 5 | high |
| Sửa bình luận lạc hậu, cập nhật sổ nợ | Sonnet 5 | low |
| Chạy lại bài canh e2e định kỳ | Haiku 4.5 | low |

---

## 5. Muốn nói gì thì nói ở dưới

**Phần đáng giá nhất của lượt này không phải cái đọc được, mà là cái đo ra là
KHÔNG đọc được.**

Chạy Codex **không** có `--approve-for-me` rồi bắt nó ghi file. Sandbox chặn.
Bản ghi ra như sau:

```
{"type":"item.completed","item":{"id":"item_1","type":"agent_message",
 "text":"Không thể tạo thu.txt: môi trường hiện tại bị khóa **chỉ đọc**,
         và thao tác ghi đã bị hệ thống từ chối. …"}}
{"type":"turn.completed","usage":{…}}
```

Lượt kết thúc **`turn.completed` bình thường**. Không một trường nào nói tới
quyền. Không có item cho patch bị từ chối. Lời từ chối **chỉ nằm trong văn xuôi
tiếng Việt do model tự viết ra**.

Nên `TuChoiSo` để **0**, và tôi ghi thẳng vào mã lẫn sổ rằng **`ChetChanQuyen`
không bao giờ kết luận được cho Codex**. Đọc được nó thì phải dò chuỗi trong câu
chữ của model — đúng thứ `trangthai.go` cấm ở dòng đầu (*"LUẬT: KHÔNG DÒ
CHUỖI"*). Đây là chỗ dễ bị "sửa" nhất bởi người sau: thêm một
`strings.Contains(text, "từ chối")` là xong, trông như vá được một lỗ hổng, thật
ra là gài một quả mìn — model đổi cách diễn đạt, hoặc chạy bằng tiếng Anh, là
lá chắn rơi im lặng. Bài `TestCodexKhongBiaSoToolBiChanQuyen` đứng canh đúng chỗ đó.

**Bẫy suýt dẫm phải, và nó là bẫy đếm.** Mỗi lời gọi tool của Codex in ra **hai
dòng** — `item.started` rồi `item.completed`, cùng `item.id`. Lượt đo bảo agent
chạy một lệnh **hai lần** ra **bốn** dòng `command_execution`. Nếu đếm cả hai thì
mọi con số lặp gấp đôi sự thật, và với `TranLapLienTiep = 10` thì một agent lặp
**5 lần** bị kết luận là chạy quẩn. Vu oan tệ hơn bỏ sót: `quan.go` đã viết sẵn
lý do — người vận hành mất niềm tin vào lá chắn rồi tắt nó đi. Không chạy thật
một lượt **có gọi tool** thì không thấy bẫy này; chỉ đọc lượt "trả lời một từ" là
đủ để viết một bộ đọc **trông** đúng.

**Rủi ro còn lại, nói thẳng.** Bộ đọc Codex nay mang đúng điểm yếu mà ô **C2** mô
tả cho Grok, chỉ nhẹ hơn một bậc: `--json` là **cờ có thật** nên đây là hợp đồng,
còn định dạng của Grok chỉ là quan sát. Nhưng hợp đồng vẫn đổi được. Codex đổi
tên `usage.input_tokens` một lần là token về 0 — mà 0 đọc như "miễn phí", không
như "chưa đọc được". Bài canh e2e là cách duy nhất biết được vào ngày nó đổi, và
nó chỉ có tác dụng nếu **thật sự được chạy**. Một bài kiểm bị `t.Skip` mặc định
mà không ai bật thì bằng không.
