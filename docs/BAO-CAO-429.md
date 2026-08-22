# Báo cáo: xử lý HTTP 429 và header `Retry-After` ở đường AI API

Phiên `sagent/429-22-08`, ngày 22/08/2026. Vùng đụng tới: `internal/aiapi/*`.

---

## 0. Trả lời thẳng câu hỏi thiết kế

> 429 là lỗi phía nhà cung cấp, nhưng nhảy sang route dự phòng NGAY có đúng không?
> Hay chờ theo `Retry-After` rồi thử lại CHÍNH route đó trước?

**Đã chọn: chờ theo `Retry-After` rồi thử lại CHÍNH route đó, có trần. Hết lượt
thử lại thì mới để tầng trên chuyển route.**

Không phải vì "thử lại nghe hợp lý hơn". Vì ba dữ kiện của chính dự án này:

**(1) Hai route dùng CHUNG một nhà cung cấp.** `.sagent/project.toml` khai
`deepseek` và `grok` cùng `base_url = https://modelapi.vn/v1`. "Route dự phòng"
ở đây không phải nhà cung cấp thứ hai — nó là cùng một cổng, cùng một máy chủ,
hỏi lại sau vài mili giây. Nếu modelapi.vn tính hạn mức theo tài khoản hoặc theo
IP thì lượt thứ hai **chắc chắn cũng 429**: mất thêm một lời gọi, thêm một lần
chờ, và thông điệp lỗi cuối cùng dài gấp đôi trong khi nguyên nhân vẫn là câu
đầu tiên.

**(2) 429 khác 5xx ở bản chất, không chỉ ở con số.** 5xx nói "chỗ tôi đang hỏng"
— không kèm lời hứa nào. 429 nói "chỗ tôi vẫn tốt, anh đi nhanh quá" — và
thường kèm `Retry-After`, tức một **lời hứa có thời hạn** rằng chờ đủ lâu thì
đúng route này sẽ chạy. Nhảy ngay là vứt bỏ thứ duy nhất mà 429 cho không.

**(3) 503 thì ngược lại, và dự án đã đo được điều đó.** Thân 503 đo được 20/08 là
`No available channel for model grok-code-fast-1 under group grok` — hỏng theo
**từng model**. Đổi route là đổi model, nên với 503 việc nhảy thật sự cứu được.
Đó là lý do bản sửa này **cố ý không mở rộng sang 5xx**: luật cũ đã đúng cho 5xx,
và chỉ 429 mới cần chờ.

### Còn một vế chưa chắc, nói thẳng ra

Hai route khai `key_id` **khác nhau** (`deepseek` và `grok`). Nếu modelapi.vn
chặn theo **key** chứ không theo tài khoản, thì route dự phòng lại cứu được thật.
**Chưa đo được** nhà này chặn theo cái nào (xem mục 2).

Chính vì chưa biết nên thứ tự **"chờ trước, nhảy sau"** mới là thứ tự đúng: nó
làm việc **chắc chắn đúng** (tôn trọng `Retry-After`) trước, rồi mới **đánh cược**
(đổi key) sau. Đảo ngược lại là đánh cược trước khi thử cái chắc. Và quan trọng
là hướng này **không đóng cửa** đường kia: hết lượt thử lại, lỗi trả về vẫn là
`*LoiAPI` với `Nguoi = false`, nên `internal/api` vẫn chuyển route dự phòng theo
luật sẵn có — **không phải sửa một dòng nào ở tầng đó**.

### Hậu quả nếu chọn sai — cả hai chiều

| Chọn sai kiểu | Hỏng ra sao |
|---|---|
| **Nhảy ngay** (đã bỏ) | Một cú nghẽn 5 giây đẩy toàn bộ phần còn lại của lượt flow sang nhà cung cấp thứ hai. Route chính thường được chọn có lý do: đo 20/08, `deepseek` 2,3s/127 token so với `grok` 13,6s/1044 token — **8,2 lần token, 5,9 lần thời gian**. Một cú nghẽn thoáng qua thành một quyết định đổi giá và đổi chất lượng câu trả lời cho cả lượt. Tệ hơn nữa: nếu chặn theo tài khoản thì route thứ hai cũng 429, tức trả tiền cho một lời gọi hỏng để đổi lấy đúng cái lỗi cũ. |
| **Chờ vô hạn** (đã bỏ) | `Retry-After: 3600` là cả lượt flow đứng im một tiếng, không log, không lối ra. Đây là lý do **mọi** con số ở mục 3 đều có trần, và vượt trần thì thôi chờ ngay lập tức. |

---

## 1. Đã làm (kèm số đo)

### Mã

| File | Đổi | Việc |
|---|---|---|
| `internal/aiapi/cholai.go` | **mới, 295 dòng** | Toàn bộ lớp đọc `Retry-After` + chờ + thử lại |
| `internal/aiapi/cholai_test.go` | **mới, 460 dòng** | 15 hàm test + 19 ca con = **34 lần PASS** |
| `internal/aiapi/aiapi.go` | +59 / −9 | `Goi` đi qua lớp thử lại; thêm `KetQua.ChoLai`, `LoiAPI.ChoLai`, `BiChanTocDo()` |
| `internal/aiapi/stream.go` | +25 / −11 | `GoiStream` đi qua cùng lớp đó |
| `internal/aiapi/stream_test.go` | +8 / −2 | Đổi tên một hàm test tên `strconv` (xem mục 2) |

Không đụng file nào ngoài `internal/aiapi/*` — trừ `docs/BAO-CAO-429.md` này.

### Hiện trạng trước và sau — kèm một đính chính về số đo cũ

Đề bài ghi hiện trạng là `grep -rn "429" internal/` = **0 dòng**. **Đo lại thì
không phải.** `git grep -n "429" main -- 'internal/**/*.go'` ra **20 dòng** có
sẵn. Đính chính chỗ này vì nếu tin con số 0 thì dễ kết luận sai rằng dự án chưa
từng chạm tới 429 ở đâu cả.

Nhưng kết luận **của** đề bài thì vẫn đúng, chỉ cần nói chính xác hơn: trong 20
dòng đó **không có dòng nào là mã xử lý 429 ở đường AI API**. Chúng là:

| Ở đâu | Là gì | Có phải xử lý 429 không |
|---|---|---|
| `internal/aiapi/stream_test.go` (3 dòng) | Test **khẳng định** 429 bị đối xử như mọi lỗi nhà cung cấp khác | Không — nó ghi nhận việc *chưa* xử lý |
| `internal/api/*_test.go` (7 dòng) | Dữ liệu mẫu: chuỗi `"429"` trong JSON của agent CLI | Không |
| `internal/provider/*` (6 dòng) | Đọc `api_error_status: "429"` từ đầu ra agent **CLI** — đường thứ nhất, không phải đường API | Không |
| `internal/dash/server.go:436` + test (3 dòng) | Dashboard tự **trả** 429 cho client khi chặn dò mật khẩu | Không — chiều ngược lại |
| `internal/store/store_test.go` (1 dòng) | Chuỗi lý do hỏng trong sổ | Không |

Vậy con số đúng là: **0 dòng mã xử lý 429 trong `internal/aiapi`**, tức đúng
điều đề bài muốn nói.

| | Trước | Sau |
|---|---|---|
| Mã xử lý 429 trong `internal/aiapi` | **0 dòng** | `cholai.go` **295 dòng** + nối vào `Goi`/`GoiStream` |
| `429` trong `internal/**/*.go` | 20 dòng, **không dòng nào là xử lý** | 69 dòng, trong đó 49 dòng mới đều ở `internal/aiapi/` |
| 429 được xử ra sao | Y hệt mọi lỗi HTTP khác: hỏng ngay, để tầng trên chuyển route | Chờ theo `Retry-After`, thử lại chính route đó tối đa 2 lần, rồi mới nhường cho tầng trên |
| `Retry-After` | Không đọc | Đọc **cả hai** dạng RFC 9110 |

**Một thứ có sẵn đáng nối vào sau:** `internal/provider/trangthai.go` đã có khái
niệm `HanMucDenLai` — mốc hạn mức mở lại — nhưng cho đường **CLI**. Hai đường
đang có hai cách hiểu riêng về cùng một chuyện "bị chặn tới bao giờ". Xem nợ số 6
ở mục 3.

### Đọc `Retry-After` — cả hai dạng, đo bằng test

`docRetryAfter` thử `strconv.Atoi` trước, không được thì `http.ParseTime`. Dùng
`http.ParseTime` chứ không tự viết `time.Parse`, vì HTTP cho phép **ba** định
dạng ngày và tự viết là chắc chắn thiếu ít nhất một.

| Header | Đọc ra | Nguồn ghi vào nhật ký |
|---|---|---|
| `30` | 30s | `Retry-After: số giây` |
| `  7  ` | 7s | `Retry-After: số giây` |
| `0` | 0s (RFC cho phép — thử lại ngay) | `Retry-After: số giây` |
| `Wed, 21 Oct 2026 07:29:00 GMT` (RFC 1123) | 60s | `Retry-After: mốc thời gian HTTP-date` |
| `Wednesday, 21-Oct-26 07:30:00 GMT` (RFC 850) | 120s | `Retry-After: mốc thời gian HTTP-date` |
| `Wed Oct 21 07:28:30 2026` (asctime) | 30s | `Retry-After: mốc thời gian HTTP-date` |
| Mốc **đã trôi qua** | 0s, **không bao giờ âm** | `Retry-After: mốc thời gian HTTP-date` |
| *(không có header)* | lùi dần | `tự lùi dần - nhà cung cấp KHÔNG gửi Retry-After` |
| `soon` | lùi dần | `tự lùi dần - Retry-After có mà KHÔNG đọc được` |
| `-5` | lùi dần | `tự lùi dần - Retry-After có mà KHÔNG đọc được` |
| `1.5` (RFC không cho phép số thực) | lùi dần | `tự lùi dần - Retry-After có mà KHÔNG đọc được` |
| `2026-10-21T07:29:00Z` (ISO, không phải HTTP-date) | lùi dần | `tự lùi dần - Retry-After có mà KHÔNG đọc được` |

Quên dạng HTTP-date hỏng theo kiểu tệ nhất: `Atoi` trả lỗi → mã tưởng "không có
header" → tự lùi 1 giây → **gọi lại vào đúng bức tường vừa dựng**. Sự cố đó có
test riêng và đã được chứng minh là bắt được (mục 2).

### Không được đoán bừa rồi im lặng

Yêu cầu 2 của đề bài. Cách làm: mỗi lần chờ ghi một `LanChoLai{Lan, Status,
Header, Cho, Nguon}`, và `Nguon` **bắt buộc** phân biệt bốn thứ trong bảng trên.
"Đọc được header" và "tự lùi dần" khác hẳn nhau về độ tin cậy; người đọc log phải
biết mình đang nhìn cái nào.

Nhật ký này nằm trong **cả `KetQua` lẫn `LoiAPI`**. Để nó chỉ trong lỗi là hỏng
đúng ca hay gặp nhất: **lần chờ THÀNH CÔNG**. Gọi lần một bị chặn, chờ 2 giây,
lần hai chạy — lượt đó `err == nil`, và nếu tin chỉ đi kèm lỗi thì nó biến mất.
Người dùng thấy một lượt đột nhiên mất 15 giây thay vì 2,3 giây mà không có gì
giải thích; `Mat` đo được độ trễ nhưng không nói được **vì sao**.

### Nguyên văn 429 khi hết lượt — yêu cầu 4

Lỗi cuối cùng có dạng:

```
  ✗ deepseek trả HTTP 429: {"error":{"message":"rate limit exceeded: 20 RPM","request_id":"req_abc123"}}
     đã thử lại 2 lần, chờ tổng 3s:
     - lần 1: HTTP 429, Retry-After: 1 -> chờ 1s [Retry-After: số giây]
     - lần 2: HTTP 429, Retry-After: 2 -> chờ 2s [Retry-After: số giây]
```

Thân 429 giữ **nguyên văn** vì đó là chỗ nhà cung cấp nói hạn mức nào bị chạm
(phút hay ngày, token hay lời gọi) và request id. Nuốt nó rồi in "bị chặn tốc độ"
là biến một thông điệp **hành động được** thành một lời than.

**Đã kiểm đường in tới người đọc, không chỉ tin là nó tới:** `cmd/sagent/main.go`
in bằng `fmt.Fprintf(os.Stderr, "  ✗ %v\n", err)` — đầy đủ, không cắt. Hàm
`motDong` (cắt còn 150 ký tự) **chỉ** dùng cho dòng thông báo chuyển route, không
dùng cho lỗi cuối. Vậy nguyên văn tới được người đọc thật.

### Bằng chứng test ĐỎ khi gỡ phần sửa ra

Đề bài yêu cầu thử thật. Đã phá mã **hai lần**, mỗi lần một kiểu, rồi khôi phục.

**Phá A — gỡ hẳn việc thử lại** (trả về đúng hành vi cũ: 429 như mọi lỗi HTTP
khác). Sửa `goiCoChoLai` thành `if true { return resp, ... }`:

```
--- FAIL: TestChanTocDoRoiThuLaiChinhRouteDo (0.01s)
--- FAIL: TestChanTocDoVoiMocThoiGian (0.01s)
--- FAIL: TestKhongCoHeaderThiLuiDanVaNoiRa (0.01s)
--- FAIL: TestHeaderKhongDocDuocThiGiuNguyenVan (0.01s)
--- FAIL: TestHetLuotThuLaiVanBaoNguyenVan (0.01s)
--- FAIL: TestVuotTranThiKhongChoMaBoCuocNgay (0.01s)
--- FAIL: TestStreamChanTocDoKhongNhanDoiChu (0.01s)
FAIL
```

**7 test đỏ.**

**Phá B — gỡ RIÊNG dạng HTTP-date**, giữ nguyên phần còn lại (đúng cái lỗi hay
gặp nhất). Đổi `http.ParseTime(v)` thành `http.ParseTime("KHONG-BAO-GIO-KHOP")`:

```
--- FAIL: TestDocRetryAfterCaHaiDang/HTTP-date_RFC_1123 (0.00s)
--- FAIL: TestDocRetryAfterCaHaiDang/HTTP-date_RFC_850 (0.00s)
--- FAIL: TestDocRetryAfterCaHaiDang/HTTP-date_asctime (0.00s)
--- FAIL: TestDocRetryAfterCaHaiDang/HTTP-date_đã_qua (0.00s)
--- FAIL: TestChanTocDoVoiMocThoiGian (1.01s)
FAIL
```

**5 ca đỏ**, đúng những ca về mốc thời gian, không thừa không thiếu. Đáng chú ý
`TestChanTocDoVoiMocThoiGian` mất **1,01 giây** khi hỏng — đúng bằng bậc lùi mặc
định 1 giây, tức nó đã rơi vào chính xác kiểu hỏng đã mô tả ở trên: bỏ qua mốc
thời gian rồi tự lùi 1 giây.

Sau khi khôi phục: `go test ./internal/aiapi/ -count=1` → `ok ... 10.470s`.

### Nghiệm thu (chạy TỪNG LỆNH RIÊNG BIỆT)

| Lệnh | Kết quả |
|---|---|
| `go build ./...` | ✅ exit 0 |
| `go vet ./...` | ✅ exit 0 |
| `go test ./...` | ✅ **27/27 package xanh**, không có dòng FAIL nào |

`internal/aiapi` chạy hết **10,3 giây** (trước đây ~2s). Phần chênh là thời gian
chờ **thật** trong 4 test — cố ý: một test về chờ mà không chờ thật thì không
chứng minh được là đã chờ.

---

## 2. Sự cố

### a) CHƯA ĐO ĐƯỢC 429 THẬT TỪ NHÀ CUNG CẤP

Đây là lỗ hổng lớn nhất của bản sửa này, ghi ra trước mọi thứ khác.

**Đã đo được (miễn phí, qua `GET /v1/models`):**

```
HTTP/1.1 200 OK
Date: Sat, 22 Aug 2026 03:28:00 GMT
Via: 1.1 Caddy
X-New-Api-Version: v1.0.0-rc.25
X-Oneapi-Request-Id: 202608220328005381051018268d9d6dnQIw51p
```

Hai điều rút ra:

1. **modelapi.vn KHÔNG gửi header hạn mức nào** ở phản hồi thành công — không
   `X-RateLimit-Limit`, không `X-RateLimit-Remaining`, không `X-RateLimit-Reset`.
   Nghĩa là **không có cách nào biết trước mình còn bao nhiêu lượt**; chỉ biết
   khi đã đâm vào tường. Đó là lý do bản sửa này phản ứng sau khi bị chặn, chứ
   không tiết chế trước.
2. Nhà này chạy gateway mã nguồn mở **new-api** (`X-New-Api-Version`,
   `X-Oneapi-Request-Id`). **CHƯA KIỂM** new-api có gửi `Retry-After` kèm 429 hay
   không — chưa đọc mã nguồn của nó, và chưa gặp 429 thật.

**Đã thử chạm 429 nhưng không chạm được:** bắn **20 lần liên tiếp** vào
`/v1/models` (miễn phí, không tốn token) → **20/20 đều HTTP 200**. Không chạm
được hạn mức ở mức đó.

**KHÔNG leo thang thêm.** Bắn dồn vào `/chat/completions` để ép ra 429 thì tốn
tiền thật và là hành vi bắt nạt một dịch vụ đang dùng. Đề bài cũng đã dặn đừng ép.

Vậy **toàn bộ hành vi 429 trong bản này được chứng minh bằng `httptest.Server`
giả lập, không phải bằng 429 thật.** Cái mà giả lập KHÔNG chứng minh được:

- modelapi.vn có gửi `Retry-After` không, và gửi dạng nào;
- họ chặn theo **key** hay theo **tài khoản/IP** — tức route dự phòng có cứu được
  429 hay không (câu hỏi ở mục 0);
- các con số ở mục 3 có hợp với cửa sổ hạn mức thật của họ không.

### b) Một hàm test tên `strconv` làm cả package không biên dịch được

`internal/aiapi/stream_test.go` có `func strconv(s string) string`. Tên đó vô hại
**chừng nào chưa file nào trong package import `strconv` thật**. Hôm nay
`cholai.go` import, và cả package chết:

```
stream_test.go:47:6: strconv already declared through import of package strconv
```

Đã đổi tên thành `chuoiJSON` kèm chú thích giải thích. Đây là gỡ bẫy, không phải
làm đẹp: cái bẫy chờ sẵn cho bất kỳ ai import một package chuẩn tên `strconv`.

### c) `gofmt -l` báo bẩn cả package — nhưng có TRƯỚC phiên này

`gofmt -l internal/aiapi/` liệt kê 5 file. Đã kiểm bằng `git stash`: trên `main`
sạch, **cả 6 file đều đã bị báo** từ trước (kết thúc dòng CRLF của checkout
Windows). Hai file mới của phiên này — `cholai.go`, `cholai_test.go` —
**không** nằm trong danh sách. Không sửa cái này: đụng vào là một diff đổi kết
thúc dòng toàn package, giữa lúc có agent khác chạy song song.

### d) `Goi` giữ `http.Client{Timeout: 120s}` cho MỖI lần thử

Trên lý thuyết một lượt có thể thành 3 × 120s + 60s chờ. Thực tế 429 do bộ giới
hạn trả về gần như tức thì (từ chối, không xử lý), nên phần cộng thêm chỉ là thời
gian chờ. **Chưa đo được** với nhà cung cấp thật. Hạn chót thật của cả lượt vẫn
là `ctx` của bên gọi, và `cho()` đã tôn trọng nó (mục 3).

---

## 3. Bước tiếp theo

### Các con số, và vì sao lại là con số đó

Dự án coi "con số không giải thích được" là một dạng nợ. Bốn con số, bốn lý do:

| Hằng | Giá trị | Vì sao đúng con số này |
|---|---|---|
| `SoLanThuLai` | **2** (tổng 3 lần chạm mạng) | Đây là số đo **duy nhất** dự án có về "nhà cung cấp hỏng liên tiếp mấy lần rồi tự hồi phục": 20/08 lúc 16:54–16:56, route `deepseek` trả **503 ba lần** rồi tự khỏi (`docs/DO-LUONG.md`). Lấy đúng hình dạng đã đo, không bịa một con số đẹp. |
| `TranMotLanCho` | **30s** | Lượt gọi thật chậm nhất từng đo: grok-4.5 **13,6s**, và một lần **31s** ở bản streaming. 30 giây tức là "chờ bằng khoảng một lượt gọi chậm nữa" — vẫn trong thứ người dùng đã quen chịu. Trên mức đó, `Retry-After` không còn nghĩa "chậm lại chút" mà là "quay lại sau" — tình huống **khác**, và câu trả lời đúng cho nó là chuyển route. |
| `TranTongCho` | **60s** | `Goi` đã chốt **120s** cho một lời gọi — con số duy nhất dự án đã cam kết cho "một lượt lâu tới đâu thì vẫn coi là đang chạy". Phần chờ lấy đúng **một nửa**, để lượt có thử lại không vượt mức kiên nhẫn của hai lượt thường, và còn dư ngân sách cho route dự phòng chạy sau khi ta bỏ cuộc. |
| `luiDanGoc` | **1s**, nhân đôi (1s → 2s) | Chỉ dùng khi **không** đọc được `Retry-After`. Tăng dần chứ không cố định, vì cái ta không biết là cửa sổ hạn mức dài bao lâu; tăng dần là cách dò mà không cần đoán. |

**Nói thẳng chỗ `SoLanThuLai = 2` không đủ:** nếu 429 kéo dài đúng như sự kiện
3-lần-liên-tiếp kia thì 3 lần chạm mạng vẫn hỏng hết. Đó là **cố ý** — hết lượt
thì đi tiếp bằng route dự phòng, chứ không chờ mãi. Bốn con số trên đều là **suy
ra từ số đo về độ trễ**, chưa phải số đo về **hạn mức**. Đo được 429 thật thì
phải xem lại cả bốn.

### Nợ đã biết, xếp theo mức đáng làm

1. **Đo 429 thật.** Điều kiện chặn sự cố 2a. Cách rẻ nhất mà không bắt nạt ai: đọc
   mã nguồn **new-api** xem module rate-limit có `Retry-After` không và tính theo
   key hay theo tài khoản. Trả lời được luôn câu hỏi mục 0.
2. **Không có jitter ngẫu nhiên trong bậc lùi.** Nói rõ để không ai tưởng là đã
   có. Hạm đội chạy **nhiều agent song song vào cùng một nhà bán lại** — đúng
   kiểu bầy đàn mà jitter sinh ra để chống: cả bầy bị chặn cùng lúc, cùng lùi 1
   giây, cùng đâm lại vào tường sau đúng 1 giây. Chưa làm vì jitter cần nguồn
   ngẫu nhiên tiêm được từ ngoài, không thì test hết tất định. Đáng làm **ngay
   sau** khi đo được 429 thật, vì lúc đó mới biết cửa sổ rộng bao nhiêu.
3. **Chưa chứng minh đầu-cuối rằng 429-hết-lượt thì tầng trên chuyển route.** Đã
   chứng minh **điều kiện cần**: `Test429KhongPhaiLoiNguoiDung` bảo đảm
   `LoiNguoiDung(err) == false` nên vòng lặp trong `aiCall` không `break`. Chưa
   viết được test đầu-cuối vì vòng lặp đó nằm ở `internal/api/api.go` — file agent
   khác đang giữ trong phiên này.
4. **`ChoLai` chưa hiện ra ở mặt web / sổ `api_calls`.** Nhật ký đã đi tới người
   đọc qua **lỗi** (mục 1), nhưng lần chờ **thành công** thì mới chỉ nằm trong
   `KetQua` chứ chưa được hiển thị hay ghi sổ. Không làm trong phiên này vì phải
   đụng `internal/dash/*` và `cmd/sagent/api.go`, ngoài vùng được giao.
5. **Chưa thử lại khi mất mạng / timeout.** Cố ý bỏ ngoài phạm vi: đề bài là 429,
   và với 5xx/timeout thì luật chuyển route sẵn có **đã đúng** (bằng chứng: thân
   503 đo được là hỏng theo từng model). Mở rộng thử lại sang 5xx sẽ làm mọi lần
   nhà cung cấp sập đều chậm thêm 3 giây trước khi nhận đúng cái lỗi lẽ ra đã
   được chuyển route ngay. Có test `TestChiThuLaiVoi429` chặn đúng chuyện đó cho
   7 mã: 400, 401, 403, 404, 500, 502, 503.
6. **Hai đường có hai cách hiểu riêng về "bị chặn tới bao giờ".** Đường CLI đã có
   `internal/provider/trangthai.go` với trường `HanMucDenLai` — mốc hạn mức mở
   lại, đọc từ đầu ra của agent. Đường API nay có `LanChoLai.Cho` + `Nguon`. Hai
   khái niệm cùng nói một chuyện mà không biết nhau, đúng kiểu "hai bản sao của
   một luật rồi lệch nhau" mà `DocKey` đã được viết ra để tránh. Chưa gộp trong
   phiên này vì `internal/provider` ngoài vùng được giao, và vì gộp trước khi đo
   được 429 thật là gộp hai thứ mà mình mới hiểu rõ một.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Đọc `internal/aiapi/*`, `internal/api/api.go`, `.sagent/project.toml` để dựng lại luật chuyển route hiện hành | claude-opus-5[1m] | Trung bình |
| Tra `docs/DO-LUONG.md` lấy số neo cho các trần (13,6s / 31s / 503 ba lần / 2,3s so 13,6s) | claude-opus-5[1m] | Trung bình |
| Quyết hướng thiết kế: chờ-rồi-thử-lại thay vì nhảy ngay | claude-opus-5[1m] | **Cao** |
| Viết `cholai.go` — đọc `Retry-After` hai dạng, trần, nhật ký nguồn | claude-opus-5[1m] | **Cao** |
| Nối vào `Goi` và `GoiStream` mà không mất `usage`, không nhân đôi chữ stream | claude-opus-5[1m] | **Cao** |
| Viết 15 hàm test / 19 ca con bằng `httptest.Server` | claude-opus-5[1m] | **Cao** |
| Phá mã hai lần để chứng minh test đỏ, rồi khôi phục | claude-opus-5[1m] | Trung bình |
| Đo thật: header của modelapi.vn, bắn 20 lần dò 429 | claude-opus-5[1m] | Thấp |
| Gỡ bẫy hàm test tên `strconv` | claude-opus-5[1m] | Thấp |
| Viết báo cáo này | claude-opus-5[1m] | Trung bình |

---

## 5. Nhận xét tự do

**Ranh giới file lại trùng đúng chỗ thiết kế nên nằm.** Đề bài cấm đụng
`internal/api/api.go` — nơi có vòng lặp chuyển route. Thoạt nhìn tưởng là chướng
ngại: xử lý 429 nghe như việc của tầng điều phối. Nhưng làm xong mới thấy nó
**đúng**: chờ-rồi-thử-lại là chuyện của **một** route với **một** nhà cung cấp,
không phải chuyện của sổ route hay của thứ tự dự phòng. Đặt nó trong
`internal/aiapi` khiến `internal/api` không phải biết 429 là gì — nó chỉ thấy
"route này hỏng vì phía nó", đúng khái niệm nó vốn đã có. Không sửa một dòng nào
ở tầng trên mà tính năng vẫn nối liền mạch. Cái ràng buộc hoá ra là cái chỉ đường.

**Chỗ dễ tự lừa nhất là ca 429 THÀNH CÔNG, không phải ca hỏng.** Khi viết, phản
xạ đầu tiên là nhét nhật ký chờ vào `LoiAPI` — nghe hợp lý, vì 429 là lỗi. Nhưng
ca hay xảy ra nhất lại là: chặn một lần, chờ 2 giây, lần hai chạy. Lượt đó
`err == nil`. Nếu nhật ký chỉ đi cùng lỗi thì đúng ca thường gặp nhất là ca mất
sạch tin, và triệu chứng duy nhất người dùng thấy là "hôm nay sagent chậm bất
thường". Đúng lớp hỏng-im-lặng mà `ThieuUsage` sinh ra để chống — cùng một bài
học, lặp lại ở một chỗ khác trong cùng package.

**Hai con số trong bảng mục 3 vẫn còn yếu, và không nên giấu chuyện đó.**
`TranMotLanCho = 30s` và `TranTongCho = 60s` suy ra từ số đo về **độ trễ gọi**,
không phải số đo về **hạn mức**. Chúng giải thích được — đúng luật của dự án —
nhưng "giải thích được" chưa bằng "đo được". Chúng trả lời câu "người dùng chịu
chờ bao lâu", chứ chưa trả lời câu "modelapi.vn mở cửa lại sau bao lâu". Câu thứ
hai mới là câu đúng, và chưa có số. Nếu hạn mức thật của họ là cửa sổ một phút
thì `TranMotLanCho = 30s` đang cắt cụt đúng những lần chờ lẽ ra sẽ thành công.

**Việc bắn 20 lần rồi được 20 mã 200 là một kết quả, không phải một thất bại.**
Nó nói rằng ở mức dùng bình thường của dự án này, 429 **không phải** chuyện hằng
ngày. Tức tính năng vừa viết là để dành cho lúc hạm đội chạy đông — đúng lúc
nhiều agent song song đâm vào cùng một cổng — và đó cũng chính là lúc **thiếu
jitter** (nợ số 2 mục 3) sẽ đau nhất. Hai chuyện đó liên quan nhau, và bản này
mới xử lý một nửa.
