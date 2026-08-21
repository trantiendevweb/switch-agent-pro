# Báo cáo — đóng ô nợ C3: chọn model từ dòng lệnh

- **Ngày**: 21/08/2026
- **Nhánh**: `sagent/tns-1`
- **Ô nợ**: `docs/SO-NO-DO-LUONG.md` mục **C3** — *"Ba provider CHƯA ĐO cách chọn
  model từ dòng lệnh"*. Thực tế lúc nhận việc: **hai** (Cursor đã đóng ở
  `bacc137`), là Antigravity và Codex.
- **Trạng thái**: ✅ **ĐÓNG HOÀN TOÀN BẰNG PHÉP ĐO**. Không dùng `KhongLamDuoc`,
  không để lại nửa ô.

---

## 1. Đã làm

### Vì sao ô này là tiền, không phải dọn dẹp

`internal/api/model_test.go:9-12` ghi sẵn con số của lượt chạy #34: **9,40 USD cả
lượt, riêng bước `code-go` 8,18 USD**, vì mọi bước đều chạy model mạnh nhất — kể
cả bước chỉ viết tài liệu hay gộp báo cáo. Trước lượt này, khai `model = "..."`
cho một bước Antigravity/Codex chỉ đổi lấy **một dòng cảnh báo**; bước **vẫn chạy
model mặc định**. Với Codex, "mặc định" có tên cụ thể: `~/.codex/config.toml` khai
`model = "gpt-5.6-sol"` và tự viết lý do — *"MODEL MẶC ĐỊNH = mạnh nhất, có lý do"*.
Nên cái mất không phải giả thuyết.

Điều kiện mới có **đúng hôm nay**: tài khoản Antigravity vừa đăng nhập lại
(`ttseotop1@gmail.com`, CLI **1.1.16**). Trước 21/08 ô này không đo nổi.

### Bằng chứng 1 — tên model bịa, và chỗ tiền lệ Cursor KHÔNG áp được

Tiền lệ `bacc137` đặt khuôn: bằng chứng mạnh nhất không phải `--help` liệt kê cờ
gì, mà là **CLI từ chối một tên model sai**. Đã chạy thật với `khong-ton-tai-9x`.
Khuôn đó **đúng với một provider và sai với provider kia**:

| Provider | Ai chặn? | Nguyên văn |
|---|---|---|
| **Antigravity** 1.1.16 | **CLI, phía máy mình** — chưa tốn token nào | thoát mã 1: `invalid model selection (--model "khong-ton-tai-9x" --effort ""): model … is not recognized as a known model or custom model in settings` + liệt kê **14** model hợp lệ |
| **Codex** 0.147.0 | **MÁY CHỦ**, CLI không chặn gì | CLI nhận cờ, in `model: khong-ton-tai-9x`, chỉ càu nhàu *"Model metadata … not found. Defaulting to fallback metadata"*; rồi HTTP **400**: *"The 'khong-ton-tai-9x' model is not supported when using Codex with a ChatGPT account"* |

Codex vẫn là bằng chứng — thậm chí **đi xa hơn**: giá trị đã đi hết đường xuống
thân yêu cầu API. Nhưng một lượt đo chỉ đi tìm đúng khuôn "CLI từ chối" sẽ kết
luận **nhầm** rằng Codex nuốt cờ.

### Bằng chứng 2 — cờ có ĐỊNH TUYẾN, hay chỉ được đem đi KIỂM TRA?

Từ chối tên sai mới chứng minh cờ **được đọc**. Chưa chứng minh model thật sự
đổi. Hai phép đo riêng, đều là số:

**Antigravity — cùng một prompt (`"tra loi dung mot tu: ok"`), ba model:**

| `--model` | `input_tokens` | `output_tokens` |
|---|---|---|
| `gemini-3.7-flash-low` | **13.747** | 30 |
| `claude-opus-4-6-thinking` | **15.764** | 27 |
| `gpt-oss-120b-medium` | **11.174** | 64 |

Cùng prompt, cùng repo, cùng CLI. Biến duy nhất là `--model`. Ba bộ tách từ khác
nhau đọc cùng một đầu vào ⇒ cờ đổi model thật.

**Codex — cờ ghi đè hồ sơ:**

| dòng lệnh | dòng `model:` ở đầu bản ghi |
|---|---|
| `codex exec "…"` (không cờ) | `gpt-5.6-sol` ← từ `config.toml` |
| `codex exec -m gpt-5.4-mini "…"` | `gpt-5.4-mini`, chạy xong thật, **12.291 token** |

Đây đúng là chiều mà `internal/provider/grok.go:236-238` đã bác một lần
(`grok -p` bỏ qua `defaultModel` trong chính file cấu hình của nó) — *"provider
tự đọc model từ hồ sơ"* **không được mặc định tin**. Phải đo riêng. Ở đây kết quả
ngược với Grok: **cờ thắng hồ sơ**.

### Cái bẫy THỨ TỰ CỜ — đo chứ không suy từ `--help`

`argsChoBuoc` (`internal/api/api.go:1397`) **chèn `ModelArgs` vào TRƯỚC**
`HeadlessArgs`. Với Codex, `-m` lại là cờ của **lệnh con** `exec`, nên dòng thật là
`codex -m <model> exec --json "<prompt>"` — cờ đứng **trước** lệnh con. Suy từ
`--help` thì đây là chỗ hỏng. Đã chạy đúng dạng đó: Codex nhận, đầu bản ghi in
`model: gpt-5.4-mini`. Antigravity cũng đã chạy đúng dạng chèn-trước
(`agy --model gemini-3.7-flash-low --output-format stream-json -p "<prompt>"` →
13.742 token, đúng chữ ký của model đó), chứ không phải chỉ dạng cờ-đứng-sau lúc
thử tay. Thứ tự này nay có bài kiểm khoá lại.

### Mã đã đổi

| File | Đổi gì |
|---|---|
| `internal/provider/antigravity.go:177` | `ModelArgs` `nil` → `--model <model>`; `:188` `Chua(NLChonModel)` → `Duoc(...)` |
| `internal/provider/codex.go:277` | `ModelArgs` `nil` → `-m <model>`; `:292` `Chua(NLChonModel)` → `Duoc(...)` |
| `internal/api/quyen_test.go` | `giaAdapter.ModelArgs` → `nil`, làm vật thử cho nhánh cảnh báo |
| `internal/api/model_test.go` | `TestProviderChuaDoModelThiPhaiCanhBao` đổi vật thử; **MỚI** `TestAntigravityVaCodexTruyenDuocModel`, `TestCodexDatCoModelTruocLenhCon` |
| `docs/SO-NO-DO-LUONG.md` | mục C3 viết lại theo khuôn đóng, văn bản cũ giữ trong `<details>`; cập nhật bảng lưới test và mục "phạm vi không phủ" |
| `docs/DO-LUONG.md` | thêm mục *21/08 — C3* với bản ghi nguyên văn |

**Đếm lại cột `ChuaDo` sau lượt này: còn 5** — antigravity 2, claude 1, codex 1,
cursor 1, grok 0. Cột `NLChonModel` **sạch trên cả năm provider**.

`go build ./...` xanh. `go test ./...` xanh (toàn bộ gói).

---

## 2. Sự cố

### Một phép đo ĐÃ BẮT ĐẦU SAI, và nó suýt cho kết luận ngược

Cách hiển nhiên để kiểm "có đúng model không" là hỏi thẳng agent. Đã thử: chạy
`agy -p "Tra loi DUNG MOT TU: ban do Google hay Anthropic tao ra?" --model
claude-opus-4-6-thinking` — nó trả về `{"status":"SUCCESS","response":"Google\n"}`.

Nếu tin câu đó, kết luận sẽ là *cờ bị nuốt → khai `KhongLamDuoc`* — **sai hoàn
toàn**, và sai theo hướng **đắt hơn hiện trạng**: nó sẽ đóng vĩnh viễn khả năng hạ
model cho Antigravity. **Tự khai danh tính không phải phép đo** — lời nhắc hệ
thống của Antigravity đè lên câu trả lời. Số token thì không biết nói dối. Đây
cùng họ với bài học của Đ3 (*tiêu đề commit không phải bằng chứng*), và đã ghi
lại nguyên văn trong `antigravity.go` để người sau không thử lại rồi tin.

### Một bài kiểm gãy, và nó KHÔNG được "sửa cho xanh"

`TestProviderChuaDoModelThiPhaiCanhBao` dùng **provider thật `antigravity`** làm
vật thử cho nhánh cảnh báo. Đóng ô này xong, nó đỏ. Chẩn đoán quan trọng: **không
phải nhánh cảnh báo hỏng, mà là nó hết vật thử** — không còn provider thật nào
trả `nil`. Nhánh cảnh báo vẫn phải sống, vì nó là thứ duy nhất đứng giữa người
dùng và một hoá đơn chạy model mặc định, cho provider **tiếp theo** được thêm vào
mà chưa ai đo. Nên vật thử đổi sang adapter GIẢ, và thêm hai bài kiểm khẳng định
chiều ngược lại: hai provider vừa đo phải truyền cờ xuống thật **và không được
cảnh báo nữa** — cảnh báo thừa cũng là một kiểu sai, nó đẩy người đọc đi tìm một
vấn đề không tồn tại.

### Nợ MỚI lôi ra, chưa trả

Trong `argsChoBuoc`, nhánh *"provider KHÔNG có rào quyền nào"* **gán đè** lên biến
`canhBao`. Một provider vừa không-có-rào-quyền vừa chưa-đo-model sẽ **mất câu cảnh
báo về model**. Hôm nay không provider nào rơi vào cả hai ô cùng lúc nên nó chưa
cắn được ai — ghi lại đúng vì đó là **lý do duy nhất** nó chưa cắn, chứ không phải
vì mã đúng.

---

## 3. Bước tiếp theo

1. **Trả nợ mới ở `argsChoBuoc`**: gom cảnh báo thành danh sách thay vì một biến
   bị gán đè.
2. **C6** — ô cuối còn mang nhãn "⚠ ĐÃ LẠC HẬU" mà chưa viết lại theo khuôn đóng.
3. **Dùng thật cái vừa mở ra**: rà các flow đang chạy Antigravity/Codex, hạ model
   cho những bước chỉ viết tài liệu / gộp báo cáo. Ô này mới chỉ *cho phép* tiết
   kiệm; chưa ai tiết kiệm.
4. **Bài canh định kỳ** cho hai cờ mới, theo tiền lệ C2/Grok — `--model` là thứ
   nhà cung cấp đổi được mà không báo ai.

---

## 4. Nên xài model gì, effort nào

| Việc | Model | Effort |
|---|---|---|
| Trả nợ `argsChoBuoc` (gom cảnh báo, sửa test) | Sonnet | medium |
| Viết lại C6 theo khuôn đóng (đọc mã + viết tài liệu) | Sonnet | low |
| Rà flow để hạ model từng bước (quyết định tốn tiền) | Opus | high |
| Bài canh định kỳ cho `--model` (viết test e2e) | Sonnet | medium |

---

## 5. Muốn nói gì thì nói ở dưới

Điều đáng giữ lại nhất của lượt này **không phải hai dòng `ModelArgs`** — mà là
chuyện **khuôn bằng chứng của Cursor không áp được cho Codex**. Sổ nợ dạy rằng
"CLI từ chối tên model sai" là bằng chứng mạnh nhất; Codex **không từ chối gì
cả**, và nếu lượt đo dừng ở đó thì nó sẽ ghi "chưa đo" lên một cờ hoạt động hoàn
hảo. Bài học tổng quát: **một khuôn bằng chứng đúng cho provider này là giả
thuyết đối với provider kia** — phải hỏi "ai là người chặn?" trước khi kết luận
"không chặn = không có cờ".

Thứ hai: hai bằng chứng ở đây có **hai mức mạnh khác nhau**, và nói thẳng thì tốt
hơn làm phẳng. Antigravity có phép đo *khách quan* (ba mức token cho ba model —
không phụ thuộc vào bất cứ điều gì CLI tự khai). Codex thì bằng chứng là **dòng
`model:` do chính Codex in ra** cộng với việc máy chủ nhận diện được tên model —
mạnh, nhưng vẫn là lời của CLI. Cả hai đủ để khai `Duoc`; chỉ là nếu ngày nào
Codex đổi hành vi, chỗ đó sẽ mục trước.

Cuối cùng, đây là ô nợ mà **cảnh báo đã che mất mức nghiêm trọng**. Mã cũ xử lý
rất đúng đắn — nó nói thẳng "bỏ qua model = ...". Đọc log thì thấy hệ thống trung
thực, gọn gàng, đâu ra đấy. Nhưng mỗi dòng cảnh báo đó là một bước đang đốt model
đắt nhất. **Một ô nợ được xử lý lịch sự vẫn là một ô nợ**, và sự lịch sự đó chính
là thứ làm nó nằm yên lâu như vậy.
