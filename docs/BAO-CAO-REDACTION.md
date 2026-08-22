# Báo cáo: tầng redaction cho nhật ký phiên

Nhánh `sagent/redaction-22-08`, tách từ `main` (lúc bắt đầu `git rev-list --count main..HEAD` = 0).
Đóng ô `[~]` của Pha 0. Ngày 22/08/2026.

---

## 1. Đã làm

### 1.1 Đo trước, viết mã sau

Quét thư mục `~/.ai-accounts/.nhat-ky/`: **23 file**, quét **22** (trừ nhật ký của
chính phiên đang chạy — lý do ở mục 2.1), **6.797 dòng**, **10,6 MB**.

**Kết quả quan trọng nhất: KHÔNG có khoá hay token nào đang bị rò.** Bằng chứng:

| Mẫu | Khớp trên 22 file |
|---|---|
| `sk-ant-`, `sk-` (24+ ký tự), `ghp_`/`gho_`/`github_pat_`, `xoxb-`, `AKIA`, `AIza`, `PRIVATE KEY` | **0** |
| JWT đầy đủ (ba đoạn ngăn bằng dấu chấm) | **0** |
| `Bearer <giá trị>` | **0** |

Ba chỗ trông như rò nhưng đo ra thì không:

- **`refreshToken` 218 lần, `accessToken` 195 lần** — đều là văn xuôi hoặc mã
  nguồn đang được agent đọc. Chỗ duy nhất có giá trị thật thì agent đã tự cắt:
  `refreshToken |type= str |len= 424 |preview= eyJhbGciOiJI...` (12 ký tự đầu).
- **469 khối base64 dài, dài nhất 9.108 ký tự** — tất cả là trường `"signature"`
  của Claude (chữ ký khối thinking), không phải khoá.
- **`API_KEY` 9 lần, `password` 46 lần** — đều là TÊN biến hoặc tên hàm
  (`GROK_API_KEY environment variable`, `--set-password`, `func pbkdf2(password,...)`),
  không kèm giá trị nào.

**Cái ĐANG rò là danh tính, không phải bí mật:**

| Nhóm | Khớp |
|---|---|
| Tên tài khoản `Administrator`, mọi dạng | **2.060** |
| — trong đường dẫn nhà (`Users\<tên>`) | 1.536 |
| — trong đường dẫn bẹp (`C--Users-Administrator-...`) | 46 |
| — còn lại: đứng trơ, chủ yếu là cột chủ sở hữu của `ls -l` | ~478 |
| Email | 179 (trong đó 75 là địa chỉ thật: `ttseotop1@gmail.com` 71 lần, và 2 địa chỉ gmail khác) |

Đây không phải chuyện nhỏ trên đúng cái máy này: `Administrator` là tài khoản
đang bị dò mật khẩu ~3,9 nghìn lần/giờ, và `dash` phục vụ nhật ký qua HTTP với
`s.exposed = !isLoopbackAddr(host)` — tức nó tự biết mình có thể không ở loopback.

### 1.2 Gói redaction dùng chung

`internal/redaction` (trước đó `grep -rni redact` toàn repo = 0 dòng).
**16 luật**, chia hai nhóm trong MỘT bảng duy nhất:

- **12 luật bí mật** — dùng cho CẢ phép che nhật ký lẫn bài kiểm chặn commit.
- **4 luật danh tính** — chỉ che nhật ký, không chặn commit (repo này có email
  trong trailer git và có `C:\Users\...` nằm trong tài liệu).

Một bảng chứ không hai, vì hai danh sách sẽ lệch nhau trong im lặng: thêm luật
cho nhật ký mà quên thêm cho commit thì bài kiểm vẫn xanh trong khi đã thủng.

**Mẫu dựa trên số đo, và số đo đã bác hai thiết kế:**

- Luật "chuỗi base64 dài = bí mật" bị loại: nó sẽ băm nát 489 trường
  `"signature"` hợp lệ để đổi lấy 0 bí mật. Mẫu JWT vì thế bắt buộc đủ ba đoạn.
- Luật "thấy tên trường `refreshToken` thì che" bị loại: che nhầm 218 chỗ vô hại.
  Mẫu chỉ bắt GIÁ TRỊ.

### 1.3 Cắm vào đâu, và vì sao không cắm vào đường ghi

**Đường ghi không chặn được**, và đây là ràng buộc kiến trúc chứ không phải bỏ
sót: `profile.StartDetached` gán thẳng file handle cho tiến trình con
(`c.Stdout, c.Stderr = f, f`) rồi tiến trình cha THOÁT. Không còn ai đứng giữa
dòng ghi. Lọc lúc ghi thì phải nuôi một tiến trình trung gian sống suốt lượt
chạy, tức bỏ hẳn kiến trúc "bật rồi buông". (Cùng lý do khiến `nhatky.Don` phải
chặn ngân sách đĩa ở lượt SAU.)

Chỗ đáng chặn cũng không phải lúc ghi: file nằm ở thư mục nhà, quyền 0600. Rủi
ro là lúc nội dung RA KHỎI đó, và có đúng **hai cửa ra**:

| Cửa | Hàm | Đi tới đâu |
|---|---|---|
| Người + dash | `nhatky.Duoi` | `sagent nhat-ky`, và `dash/server.go:1172` qua HTTP |
| Máy | `nhatky.BoDau` | `api.readLogs` → output bước agent → **prompt của bước SAU** |

Tầng che gắn vào cả hai hàm, nên không mặt gọi nào quên được — kể cả mặt gọi
viết sau này. Diff vào `internal/nhatky/nhatky.go`: **20 dòng thêm, 0 dòng xoá**.

**Đường của máy phân loại không bị đụng:** `api.phanLoaiPhienChet` đọc thẳng
`os.ReadFile` rồi đưa cho `adapter.DocKetQua`, không qua hai hàm trên. Có test
riêng ghim chuyện này (`TestPhanLoaiPhienChetVanNhinThayByteGoc`) — cắm nhầm
tầng che vào đó thì mọi phiên rơi về `lost`, tức trả lại đúng cái mù loà mà
nhật ký sinh ra để chữa.

### 1.4 Đo lại sau khi sửa, qua đúng cửa `nhatky.Duoi`

| Nhóm | Trước | Sau |
|---|---|---|
| Đường dẫn nhà | 1.536 | **0** |
| Đường dẫn bẹp | 46 | **0** |
| Tên tài khoản (mọi dạng) | 2.060 | **0** |
| Email | 179 | **0** |
| Chữ ký thinking (PHẢI giữ) | 489 | **489** |
| Dòng JSON hỏng | 0 | **0** (trên 6.092 dòng) |

23 chỗ chữ `Administrator` còn lại sau khi che đều nằm trong từ `Administrators`
— tên NHÓM Windows, không phải danh tính người dùng. Đã kiểm riêng: số lần tên
tài khoản đứng một mình còn sót = **0**.

### 1.5 Test đỏ thật khi gỡ phần sửa ra

Đây là phần được yêu cầu chứng minh bằng thử nghiệm, không phải bằng lời. Cách
làm: gỡ đúng một chỗ, chạy lại, xem test nào đỏ, rồi khôi phục.

| Gỡ cái gì | Test đỏ | Test vẫn xanh |
|---|---|---|
| Bỏ `redaction.Che` khỏi `nhatky.Duoi` | `TestSessionNhatKyDocCheBiMat` | `TestReadLogsCheBiMat` |
| Bỏ `redaction.Che` khỏi `nhatky.BoDau` | `TestReadLogsCheBiMat` | `TestSessionNhatKyDocCheBiMat` |
| Bỏ nhánh `\uXXXX` khỏi mẫu email | `TestCheGiuNguyenNDJSONHopLe` (`invalid character '*' in \u hexadecimal character escape`) | — |
| Gài một file `.env` có khoá giả vào repo | `TestRepoKhongCoBiMat`, chỉ đúng `file:dòng` | — |

Hai hàng đầu là điểm mấu chốt: mỗi cửa đỏ riêng, cửa kia vẫn xanh. Nghĩa là hai
dây độc lập, và không dây nào được test kia "gánh hộ". Test gọi thẳng
`redaction.Che` thì cả hai chỗ gỡ này đều xanh — nên chúng nằm ở `internal/api`,
đúng CHỖ GỌI.

### 1.6 Bài kiểm quét repo chặn khoá lọt vào commit

`TestRepoKhongCoBiMat` duyệt `git ls-files` — đúng tập file sẽ đi vào commit.
Quét **294 file văn bản**, bỏ qua 16 file (nhị phân hoặc quá 8 MB), báo cáo cả
hai con số để không có chuyện cắt ngầm mà đọc như "đã phủ hết".

- Dùng chung bảng luật với tầng che (`TimBiMat` đọc cùng `bang`).
- **Không có danh sách miễn trừ.** Bí mật giả trong mọi file test đều được GHÉP
  LÚC CHẠY nên bài kiểm không thấy chúng. Mở miễn trừ thì lần rò thật đầu tiên
  cũng sẽ được cho vào danh sách ấy, đúng lúc đang vội.
- Bản báo lỗi tự che khoá (`Trich: Che(...)`) — nếu không thì chính log CI trở
  thành chỗ rò tiếp theo.
- Là test Go chứ không phải git hook: hook nằm trong `.git/`, không được clone
  theo, và tắt được bằng `--no-verify`.

### 1.7 Nghiệm thu

Chạy từng lệnh riêng biệt:

```
go build ./...   EXIT=0
go vet ./...     EXIT=0
go test ./...    EXIT=0   (không package nào FAIL)
```

---

## 2. Sự cố

Có bốn, tất cả đều là lỗi của lượt này và đều đã sửa. Ba trong bốn cái do chính
phép đo bắt được chứ không phải do suy luận.

### 2.1 Suýt báo cáo một vụ rò không có thật (hiệu ứng người quan sát)

Lượt quét ĐẦU TIÊN báo: có khoá Anthropic, token GitHub, token Slack, mỗi thứ
đúng 1 lần. Nhìn qua thì đúng là "tìm thấy rò".

Cả ba nằm trong CÙNG MỘT file: `claude-tns-c1-20260822-005141.907.log` — nhật ký
của **chính phiên đang quét**. Nhật ký ghi lại từng lệnh agent gõ, nên câu lệnh
`grep 'sk-ant-' ...` vừa chạy đã tự trở thành một dòng khớp.

Đo nhật ký của phiên đang chạy mà không trừ file của chính mình ra thì sẽ luôn
tìm thấy đúng thứ mình đi tìm. Mọi con số trong báo cáo này đã trừ file đó.
Chuyện này được ghi vào tài liệu gói để lượt sau không dẫm lại.

### 2.2 Mẫu bản đầu bắt nhầm 23 chỗ trong chính repo

Bài kiểm quét repo đỏ ngay lần chạy đầu, 23 chỗ, do hai lỗi mẫu:

- Luật biến môi trường có `(?i)` nên bắt cả định danh camelCase của Go:
  `p.HasToken = ad.HasToken(...)`, `const duongToken = "vendor/token.css"`.
  → Bỏ `(?i)`; tên biến môi trường vốn là CHỮ HOA GẠCH DƯỚI.
- Luật trường JSON viết `[^"]{12,}` nên bắt cả CÚ PHÁP NỐI CHUỖI Go:
  ``{"accessToken":"` + jwtGia(con) + `"}`` — thứ khớp là dấu nháy ngược và dấu
  cộng, không phải bí mật.
  → Giá trị phải nằm trọn trong bộ chữ của token và dài từ 20 ký tự.

Đáng nói: **bài kiểm ở bước 4 là thứ bắt được lỗi của bước 2.** Nếu làm ngược
thứ tự, hoặc nếu bài kiểm có danh sách miễn trừ, cả hai lỗi này đã đi thẳng vào
commit.

### 2.3 Tầng che làm HỎNG 2 dòng JSON thật

Chạy tầng che trên 22 file nhật ký thật rồi phân tích lại từng dòng: **2 dòng
JSON hợp lệ trở thành không parse được**. Trước khi che: 0 dòng hỏng.

Nguyên nhân: Antigravity ghi dấu ngoặc nhọn thành chuỗi thoát JSON, nên trong
file có `\u003cnoreply@anthropic.com\u003e`. Mẫu email nuốt luôn `u003c` làm phần
cục bộ, thay xong còn lại `\` đứng trước nhãn — tức `\[`, không phải chuỗi thoát
JSON hợp lệ.

Đây là kiểu hỏng tệ nhất: **hỏng ở phía bên kia**. Nhật ký vẫn mở ra đọc được;
chỉ có agent ở bước sau nhận một dòng không parse được, và không ai biết vì sao.

Đã sửa (mẫu ăn trọn chuỗi thoát rồi trả lại nguyên vẹn) và ghim bằng test hồi
quy. Ví dụ tự nghĩ sẽ không bao giờ tìm ra ca này — chỉ chạy trên dữ liệu thật
mới ra.

### 2.4 Mẫu đường dẫn bỏ sót 581 chỗ

Sau khi che bằng bộ mẫu bản đầu, đếm lại thì vẫn còn 581 lần tên tài khoản. Ba
dạng chưa phủ, cả ba đều đo được:

- **916 chỗ** — JSON LỒNG JSON: dòng nhật ký là JSON, bên trong có chuỗi ghi lại
  lời gọi công cụ vốn cũng là JSON, nên gạch ngược bị thoát HAI lần
  (`C:\\\\Users\\\\Administrator`). Mẫu `{1,2}` dấu ngăn trượt sạch, và trượt IM
  LẶNG. → `{1,4}`.
- **46 chỗ** — đường dẫn bẹp thành gạch nối: Claude Code đặt tên thư mục dự án
  bằng `C--Users-Administrator-Projects-...`, không còn gạch chéo nào. → luật riêng.
- **~478 chỗ** — tên tài khoản đứng trơ, là cột chủ sở hữu của `ls -l`. Không mẫu
  hình dạng nào bắt được một cái tên đứng một mình. → luật lấy tên từ thư mục nhà,
  có ngưỡng 4 ký tự (tên ngắn kiểu `dev` mà thay bừa thì băm nát văn xuôi) và có
  ranh giới từ (`Administrators` là tên NHÓM, phải giữ).

---

## 3. Bước tiếp theo

Theo thứ tự đáng làm trước:

1. **Che ở tầng vận chuyển của dash, không chỉ tầng nhật ký.** Lượt này đóng cửa
   `SessionNhatKyDoc`. Còn các mặt khác của dash (tóm tắt phiên, sự kiện, output
   bước flow) chưa được soi xem chúng có đường nào chạm nội dung nhật ký mà không
   đi qua `Duoi`/`BoDau` không. Chưa đo nên chưa dám nói là có hay không.
2. **`sagent nhat-ky --tho`** để người chủ máy đọc bản chưa che khi cần truy
   nguyên thật. Hiện không có đường nào lấy lại bản gốc qua CLI (file trên đĩa
   vẫn nguyên vẹn, chỉ là phải mở bằng tay).
3. **Đo lại trên nhật ký của harness khác.** Số đo lượt này nghiêng hẳn về Claude
   (19/22 file). Codex và Grok chưa có file nào trong thư mục, nên định dạng bản
   ghi của chúng chưa được soi lần nào.
4. **Cân nhắc bỏ trường `"signature"` khỏi nhật ký.** 489 khối, khối dài nhất
   9.108 ký tự — chúng không phải bí mật nhưng chiếm phần lớn dung lượng và không
   ai đọc. Không làm ở lượt này vì đó là đổi NỘI DUNG nhật ký, khác hẳn việc che.
5. **Ghim ô `[~]` Pha 0 trong `docs/MASTER-PLAN.md`.** Không tự sửa vì có agent
   khác đang chạy trên file đó.

---

## 4. Bảng

| Việc | Model | Effort |
|---|---|---|
| Đo nhật ký thật, bác bỏ vụ rò không có thật (mục 2.1) | claude-opus-5[1m] | cao |
| Thiết kế + viết `internal/redaction` (16 luật) | claude-opus-5[1m] | cao |
| Cắm vào `internal/nhatky`, giữ đường máy phân loại nguyên byte | claude-opus-5[1m] | trung bình |
| Test chỗ gọi + chứng minh đỏ bằng cách gỡ dây (4 lần) | claude-opus-5[1m] | cao |
| Bài kiểm quét repo, sửa 23 dương tính giả nó bắt được | claude-opus-5[1m] | trung bình |
| Chạy tầng che trên 10,6 MB nhật ký thật, sửa 2 dòng JSON hỏng + 581 chỗ sót | claude-opus-5[1m] | cao |
| Viết báo cáo | claude-opus-5[1m] | thấp |

Cả lượt chạy trên một model, một phiên. Cột effort là mức tự đánh giá theo số
vòng đo-sửa-đo phải làm, không phải tham số của công cụ.

---

## 5. Nhận xét và rủi ro còn lại

**Điều đáng nói nhất: việc này suýt trở thành một báo cáo bịa.** Lượt quét đầu
tiên cho ra đúng cái mà một người muốn tìm ra sẽ mừng: khoá Anthropic, token
GitHub, token Slack. Chỉ cần dừng ở đó và viết "đã phát hiện 5 bí mật rò rỉ" là
xong, nghe rất được việc. Cái ngăn lại không phải sự cẩn thận mà là một thói
quen máy móc: mở ngữ cảnh của từng chỗ khớp ra đọc. Đọc rồi mới thấy cả 5 nằm
trong nhật ký của chính phiên đang quét.

**Giá trị thật của lượt này không nằm ở chỗ chặn khoá.** Khoá đang rò = 0, và
12 luật khoá là phòng ngừa — báo cáo này nói thẳng thế thay vì để người đọc
tưởng vừa bịt được một lỗ thủng. Thứ thật sự bịt được là **2.060 lần lộ tên tài
khoản và 179 lần lộ email**, qua một cổng HTTP tự nó biết mình có thể không nằm
trên loopback, trên một máy đang bị dò mật khẩu vào đúng cái tên đó.

**Ba lần đo cứu ba lỗi.** Bài kiểm quét repo bắt lỗi của bộ mẫu (2.2). Chạy trên
nhật ký thật bắt lỗi làm hỏng JSON (2.3) và 581 chỗ sót (2.4). Không lần nào
trong ba lần đó tìm ra được bằng cách ngồi nghĩ — và cả ba đều là loại hỏng im
lặng, tức nếu không đo thì sẽ không bao giờ có ai báo.

### Rủi ro còn lại

- **Che lúc đọc, không che lúc ghi.** File trên đĩa vẫn chứa nguyên văn. Ai đọc
  được đĩa thì tầng này không cản. Nó chỉ cản nội dung ĐI RA — và đó là ranh giới
  thật của kiến trúc "bật rồi buông", không phải chỗ có thể vá thêm.
- **Luật tên tài khoản phụ thuộc môi trường.** Nó lấy tên từ thư mục nhà. Chạy
  dưới một tài khoản khác thì che tên khác — đúng ý, nhưng nghĩa là hành vi của
  `Che` không thuần tuý theo đầu vào. Đây là luật duy nhất như vậy và đã ghi rõ.
- **Ngưỡng 4 ký tự là một lỗ hổng có chủ ý.** Tài khoản tên `dev`, `ci`, `adm`
  sẽ KHÔNG được che. Đổi lại là không băm nát nhật ký. Đánh đổi này nên được xem
  lại nếu công cụ chạy trên máy có tên tài khoản ngắn.
- **Mẫu khoá chưa lần nào gặp khoá thật.** Cả 12 luật bí mật mới chỉ được thử
  bằng bí mật GIẢ tự dựng. Chúng đúng về hình dạng, nhưng chưa có bằng chứng
  thực địa nào — và không nên trình bày như thể có.
- **Danh sách nhà cung cấp là hữu hạn.** Nhà cung cấp mới, định dạng khoá mới thì
  không có luật nào bắt. Không có tầng phòng ngừa chung nào ở dưới, vì mọi tầng
  chung mà nghĩ ra được ("base64 dài", "entropy cao") đều đã bị chính số đo bác:
  chúng sẽ băm 489 trường `"signature"` hợp lệ để đổi lấy 0 bí mật.
- **`gofmt -l internal/` báo gần như mọi file trong repo** (repo dùng CRLF).
  Không phải do lượt này, nhưng nghĩa là gofmt không dùng được làm cổng kiểm cho
  repo này — nếu muốn nó thành cổng kiểm thì phải xử lý riêng chuyện xuống dòng.
