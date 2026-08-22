# Trần đồng thời cho đường FLOW — mảnh cuối của engine flow

Nhánh `sagent/tran-flow-22-08`, 22/08/2026.

---

## 1. Đã làm

### Vấn đề, nói bằng số

Sáng 22/08 dự án đã có **bốn trần đồng thời** (chung 4 · harness 3 · provider 3 ·
hồ sơ 2) trong `internal/fleet/tran.go`. Nhưng chúng chỉ canh **một cửa**:
`FleetStart`. Bộ chạy flow nhận đúng một con số — `MaxParallel:
a.cfg.Policy.MaxParallelSessions` — và con số đó chỉ đếm TỔNG.

Hậu quả **đo được**, không phải suy luận. Dựng một flow bốn bước `agent`, cả bốn
khai `profile = "claude:gia"` (trần hồ sơ = 2), chạy qua `sagent flow run` thật:

| | bản trước khi sửa | bản sau khi sửa |
|---|---|---|
| Phiên bật cùng lúc trên MỘT tài khoản | **4** (trong 79 ms) | **2** |
| Mốc khởi động | 14:49:19.291 / .338 / .342 / .370 | 14:49:23.777 / .821 · rồi .860 / .880 |
| Trần hồ sơ khai trong dự án | 2 | 2 |
| Bước bị GIẾT vì trần | 0 lượt này, **1 lượt khác** | **0** |
| Token tiêu | 0 | 0 |

Bản cũ hỏng theo **hai kiểu**, tuỳ cuộc đua thắng hay thua:

- **Thắng đua → vượt trần.** Bốn bước gọi `FleetStart` gần như cùng lúc; mỗi
  lượt đọc sổ phiên TRƯỚC khi các lượt kia kịp ghi phiên vào, nên cả bốn cùng
  qua. Đo được ở trên: 4 phiên trong 79 ms trên một tài khoản có trần 2.
- **Thua đua → giết cả lượt chạy.** Lượt khác của đúng flow đó, công cụ tự in ra:
  `hết chỗ cho claude:gia: harness claude đã đầy (trần 3, đang chạy 3); provider
  claude đã đầy (trần 3, đang chạy 3); hồ sơ claude:gia đã đầy (trần 2, đang
  chạy 3)`. Chú ý con số **"đang chạy 3"** trên một trần 2 — chính tang chứng
  của kiểu hỏng thứ nhất, in ra bởi kiểu hỏng thứ hai.

Trần hồ sơ 2 không cứu được ca này vì nó không đứng ở cửa này.

### Bản sửa

**Một cái cổng có hàng đợi, dựng trên đúng bộ đếm đã có.**

`internal/fleet/cong.go` (mới). Mọi quyết định đi qua `XetTran` của `tran.go` —
không có bản đếm thứ hai. Cổng chỉ thêm ba thứ `tran.go` cố ý không có: hàng
đợi, lời nói lúc chờ, và cái chốt chống treo im lặng.

Chỗ dễ sai nhất là **phép cộng phiên**:

```
đang chạy = nền (phiên NGOÀI cổng, đọc từ sổ) + chỗ cổng đang giữ
```

"Nền" chụp **một lần** lúc mở cổng. Đọc lại sổ mỗi lượt xét thì phiên do chính
cổng bật ra bị đếm **hai lần** (một ở sổ, một ở chỗ đang giữ), và trần hồ sơ 2
lặng lẽ hoá thành 1. Chỉ có đúng một lúc đọc lại sổ là an toàn: khi cổng **không
giữ chỗ nào** — lúc đó sổ không thể chứa phiên của cổng. Có bài kiểm riêng canh
đúng cái bẫy này (`TestCongKhongDemPhienCuaChinhNoHaiLan`).

**Cắm ở `do()` chứ không ở `runWave`.** `foreach` có semaphore RIÊNG
(`runForEach`) và cũng đi qua `do()`; canh ở `runWave` là bịt một đường hở một
đường — đúng lớp lỗi cả ngày hôm nay đi sửa. Bài
`TestForEachCungPhaiQuaCongTran` canh đường đó.

**Phân giải hồ sơ dùng chung một hàm.** `api.ParseAddr` nay uỷ thẳng cho
`fleet.PhienTu`. Cổng phải đếm đúng cái địa chỉ mà bộ chạy thật sẽ dùng; lệch
một ca là trần đi canh một tài khoản không ai chạy — và hỏng trong im lặng: trần
vẫn "hoạt động", chỉ là vô dụng. `api.PhienDangChay` cũng gom về một chỗ đọc sổ
cho cả hai cửa.

### Bốn yêu cầu, và bằng chứng từng cái

**(2) Bị chặn thì CHỜ RỒI CHẠY, không từ chối.** Đo trên máy: hai bước đứng chờ
rồi chạy tiếp, lượt chạy kết thúc `done`, đủ 4/4 bước:

```
bước a2 CHẠY TIẾP sau 1s xếp hàng — được cấp 1 chỗ ở claude:gia. ...
bước a3 CHẠY TIẾP sau 2s xếp hàng — được cấp 1 chỗ ở claude:gia. ...
```

**(3) Nói ra khi đang chờ.** Dòng thật, in ra từ lượt chạy thật:

```
bước a3 CHỜ TRẦN ĐỒNG THỜI (đã chờ 0s): chật nhất là hồ sơ claude:gia — trần 2,
đang chạy 2, còn 0 chỗ. Bốn trần lúc này — chung: trần 4, đang chạy 2, còn 2 ·
harness claude: trần 3, đang chạy 2, còn 1 · provider claude: trần 3, đang chạy
2, còn 1 · hồ sơ claude:gia: trần 2, đang chạy 2, còn 0. Đang giữ chỗ: bước a1
(claude:gia, 1 chỗ, 13ms), bước a4 (claude:gia, 1 chỗ, 53ms). Bước KHÔNG bị huỷ
— nó tự chạy tiếp khi có chỗ.
```

Trần nào chặn · còn mấy chỗ · chờ bao lâu · **ai đang giữ** · mấy bước khác đang
xếp hàng · và câu trấn an cuối cùng, để người trực 2 giờ sáng đừng đi giết tiến
trình.

**(4) Kẹt cứng thì báo ra, không treo im lặng.** Hình dạng kẹt cứng = cổng không
giữ chỗ nào (nên trong lượt này không có gì sẽ trả chỗ ra) mà sổ đọc lại vẫn
chặn. Nói to **ngay** từ lúc mới nghi, nhắc lại theo nhịp, bỏ cuộc sau 15 phút:

```
bước a3 ĐỨNG vì hết chỗ ở claude:gia, mà lượt flow này KHÔNG có bước nào đang
chạy để trả chỗ ra — chỗ đang bị chiếm bởi thứ NGOÀI lượt chạy (một lượt
`sagent fleet` khác, hoặc phiên đã chết mà sổ vẫn ghi là đang chạy). ...
Sẽ chờ tối đa 15m0s rồi mới bỏ cuộc. Trong lúc chờ: `sagent status` xem phiên
nào đang giữ, `sagent quet` soi tiến trình mồ côi.
```

**Một điểm cần nói thẳng về yêu cầu (4).** Đề bài nêu ca "ai đó đặt trần hồ sơ
= 0". Ca đó **không gây kẹt**: theo quy ước sẵn có của dự án (`tran.go`, và
`m > 0` của `max_parallel_sessions`), **0 nghĩa là TẮT chiều đó**, không phải
"cấm chạy". Vì mọi trần đang bật đều ≥ 1, cấu hình một mình **không bao giờ** đẻ
ra kẹt cứng. Ca kẹt THẬT — và tôi đã ép nó xảy ra trên máy — là chỗ bị chiếm bởi
phiên **ngoài** lượt flow: một lượt `sagent fleet` khác, hoặc phiên đã chết mà
sổ vẫn ghi đang chạy. Chốt chống treo canh đúng ca đó.

### Số đo đi kèm, đáng biết

`TestTranProviderLaTranNgoaiCungCuaMotNhaCungCap`: với bộ trần **mặc định**, một
flow có bốn bước `agent` cùng nhà cung cấp `claude` **không bao giờ** chạy quá
**3** phiên cùng lúc — dù trải ra bốn tài khoản khác nhau, và dù
`max_parallel_sessions = 4`. Chặn nó là trần **provider** (mặc định 3), không
phải trần chung. Ghi thành bài kiểm chứ không chỉ ghi vào tài liệu, vì đây đúng
là thứ người vận hành sẽ gặp rồi tưởng công cụ hỏng ("tôi có 4 tài khoản mà nó
chỉ chạy 3").

### Bài kiểm — và bằng chứng chúng ĐỎ khi gỡ bản sửa

Kiểm ở **chỗ gọi**: chạy một flow thật qua `Runner.Start` rồi đếm số bước `agent`
chồng nhau. Gọi thẳng `fleet.XetTran` trong bài kiểm thì bỏ cổng ra khỏi
`step.go` vẫn xanh — chính là cái bẫy đã để lọt chuyện này.

Thử thật: chèn `return nil, nil` vào đầu `Runner.xinCho` (tức gỡ đúng bản sửa ở
cửa flow) rồi chạy lại. **7 bài đỏ**, kèm số đo:

```
--- FAIL: TestBonBuocAgentCungHoSoKhongVuotTranHoSo
    TRẦN HỒ SƠ BỊ VƯỢT Ở CỬA FLOW: đỉnh cùng lúc 4, trần 2
--- FAIL: TestBuocBiTranChanThiChoChuKhongGietLuotChay
    trần hồ sơ 1 mà đỉnh cùng lúc là 2
--- FAIL: TestNoiRaKhiDangChoTran
    TREO IM LẶNG: bước phải chờ mà không nói một dòng nào
--- FAIL: TestKetCungThiBaoRaChuKhongTreo
    kẹt cứng phải làm lượt chạy hỏng CÓ LÝ DO, được completed
--- FAIL: TestTranProviderLaTranNgoaiCungCuaMotNhaCungCap
    trần provider claude mặc định là 3 nên đỉnh phải là 3, đo được 4
--- FAIL: TestForEachCungPhaiQuaCongTran
    FOREACH LỌT CỔNG: đỉnh cùng lúc 4, trần hồ sơ 2
--- FAIL: TestCopiesVuotTranThiCatXuongChuKhongTreo
    copies phải bị cắt còn 2 (trần hồ sơ), bộ chạy nhận [4]
```

Thêm `internal/api/congflow_test.go` canh **tầng cuối**: bộ chạy mà `sagent flow
run` thật sự dùng có được cắm cổng không, hai cửa có đọc chung một con số phiên
không, hai cách tách địa chỉ có lệch nhau không.

### Nghiệm thu

```
go build ./...   -> 0
go vet ./...     -> 0
go test ./...    -> 0
```

### File đã đụng

| File | Việc |
|---|---|
| `internal/fleet/cong.go` (mới) | Cổng có hàng đợi, dựng trên `XetTran` sẵn có |
| `internal/fleet/cong_test.go` (mới) | Kiểm chính cái cổng: đếm hai lần, nhả chỗ, cắt số xin, huỷ ctx |
| `internal/flow/cong.go` (mới) | Xin chỗ cho bước `agent`, phân giải hồ sơ trùng chỗ chạy thật |
| `internal/flow/tran_flow_test.go` (mới) | Kiểm ở CHỖ GỌI, qua `Runner.Start` |
| `internal/api/congflow_test.go` (mới) | Canh tầng cuối: cổng có được cắm không |
| `internal/flow/runner.go` | Thêm trường `Cong` |
| `internal/flow/step.go` | Xin chỗ trong `do()`, dùng số chỗ ĐƯỢC CẤP |
| `internal/api/api.go` | Cắm cổng vào runner; `PhienDangChay`; `ParseAddr` uỷ cho `fleet.PhienTu` |

---

## 2. Sự cố

### 2.1 Bản đầu của tôi GIẾT lượt chạy vì một thứ sắp xong sau một giây

Nặng nhất, và chỉ lộ ra vì đo thật.

Bản đầu báo lỗi **ngay** khi thấy hình dạng kẹt cứng. Ép ca đó trên máy: một
lượt `sagent fleet` bên ngoài giữ hai chỗ và sẽ trả lại sau **khoảng một giây**
— thế mà **cả bốn bước** của lượt flow bị giết sạch. Đúng cái kết cục mà bản sửa
này lập ra để chống, do chính bản sửa gây ra.

Đã sửa: nói to ngay từ lúc **nghi**, chờ tối đa `ChoKetCung` (15 phút), bỏ cuộc
sau đó. Đo lại: cả bốn bước sống sót. Có bài kiểm riêng
(`TestCongChoPhienNgoaiSapXongChuKhongGiet`).

### 2.2 Chờ 30 giây cho một thứ đã xong sau 2 giây

Đo tiếp bản đã vá 2.1: bốn bước đứng **đủ 30 giây** rồi mới chạy, dù lượt hạm
đội ngoài xong sau ~2 giây. Vì khi bị chặn bởi phiên NGOÀI thì không ai đánh
thức hàng đợi — chỉ có tự đi đọc lại sổ mới biết, mà nhịp đọc đang là nhịp NÓI
(30 giây).

Đã sửa: tách hai nhịp. `Nhip` là nhịp **nói cho người nghe**; `NhipDoNgoai`
(2 giây) là nhịp **nhìn sổ** khi đang nghi kẹt cứng. Đo lại: `CHẠY TIẾP sau 1s`
và `sau 2s` thay cho `sau 30s`.

### 2.3 Bài kiểm quan trọng nhất XANH khi đã gỡ bản sửa

Bản đầu của `TestBonBuocAgentCungHoSoKhongVuotTranHoSo` chỉ cho mỗi lượt agent
giả ngủ 40 ms. Khi tôi gỡ bản sửa ra để thử thì nó **vẫn xanh**: bốn lượt bị các
lượt ghi SQLite làm lệch pha, nên đỉnh đo được là 2 dù chẳng có trần nào chặn.
Một bài kiểm xanh vì tình cờ còn tệ hơn không có bài kiểm.

Đã sửa bằng **chốt hẹn** (`agentDo.ChoDu`): mỗi lượt đứng lại tới khi có đủ N
lượt cùng lúc. Chặn được thì chốt không bao giờ đầy, và bài đỏ ngay. Kéo dài
giấc ngủ chỉ làm con flake hiếm đi chứ không mất.

### 2.4 Kỳ vọng của tôi sai, không phải mã sai

`TestHaiTaiKhoanKhacNhauVanChayCungLuc` đỏ với "đỉnh chỉ 3". Không phải lỗi:
trần **provider** mặc định là 3, nên bốn phiên `claude` không bao giờ chạy cùng
lúc dù trải ra hai tài khoản. Đã tách hai chiều ra cho đúng, và biến con số 3 đó
thành một bài kiểm riêng (xem mục 1).

### 2.5 Một bài của gói khác đỏ một lần rồi không tái hiện

`internal/aiapi/TestKhongChoQuaHanChotCuaLuot` đỏ đúng **một lần** trong lượt
`go test ./...` đầu tiên. Đo bằng stash chứ không đoán: cây sạch chạy riêng gói
đó → xanh; cây có bản sửa chạy riêng gói đó 3 lần → xanh; `go test -count=1
./...` toàn bộ với bản sửa → xanh. Kết luận: bài đó nhạy với tải máy (nó đo một
hạn chót ~2 giây), và bộ kiểm mới của tôi làm nặng máy thêm. Đã rút bớt thời
gian ngủ của bài chậm nhất. **Không đụng vào `internal/aiapi`** — vùng của agent
khác.

### 2.6 CLI giả không được dùng, và điều đó lại tốt

Để đo đường agent mà không tốn token, tôi dựng một `claude.cmd` giả trong sandbox
riêng (đổi `USERPROFILE`, không đụng kho tài khoản thật). Go `LookPath` **có**
tìm ra nó, nhưng `sagent` chạy qua vẫn gọi Claude thật. Chưa truy ra vì sao.

Không đi tiếp, vì Claude thật đã cho đúng thứ cần: phiên dừng ở
`authentication_failed`, `input_tokens: 0`, `total_cost_usd: 0` — **sống đủ lâu
để đo chồng lấn mà không tiêu một đồng nào**. Ghi lại vì đây là một lỗ hổng nhỏ
trong hiểu biết, không phải một thứ đã giải quyết.

---

## 3. Bước tiếp theo

1. **`ChoKetCung` = 15 phút đang chép cứng, CỐ Ý chưa đưa vào cấu hình.** Đưa vào
   `policy.tran` thì phải in được ở `sagent config` — mà chỗ in (`inTran` trong
   `cmd/sagent/main.go`) nằm ngoài vùng của tôi. Một khoá cấu hình siết thật
   nhưng không hiện ở `sagent config` chính là hình dạng lỗi dự án dính năm lần
   hôm nay. Thà chép cứng và nói ra, còn hơn cắm nửa vời. Khi nào có người thật
   vấp phải 15 phút thì thêm cả hai đầu cùng lúc.
2. **Chưa đo với agent THẬT tiêu hạn mức.** Mọi số ở trên đo bằng phiên dừng ở
   khâu xác thực (0 token). Chồng lấn thật của bốn agent chạy 10 phút có thể lộ
   ra chuyện khác — ví dụ chỗ giữ bị treo khi `waitSessions` mất dấu phiên.
3. **Trần chưa nhìn thấy phiên bật giữa chừng bởi tiến trình khác.** "Nền" chụp
   một lần lúc mở cổng, và chỉ đọc lại khi cổng rỗng (xem mục 1 — đọc lại lúc
   khác là đếm hai lần). Một lượt `sagent fleet` bật SAU khi flow đã chạy sẽ
   không được cổng đếm; `FleetStart` vẫn chặn nó, nhưng chiều ngược lại thì
   không. Muốn kín cả hai chiều thì phải đánh dấu phiên theo chủ sở hữu trong sổ,
   không vá được bằng cách đọc sổ dày hơn.
4. **Dọn dự án đo tạm** `C:\Users\Administrator\do-tran-flow` (đã xoá cuối lượt
   này) — trong đó có hồ sơ `claude:gia` giả và một `.credentials.json` chỉ chứa
   hai mốc thời gian, không phải token thật.

### Luật ngang quyền — bốn mặt

**Việc này KHÔNG đẻ ra action mới**, và đây là lý do, không phải lời chống chế:

- **`api.Actions`**: không thêm mục nào. Đây là hành vi của action `flow.run`
  đã có, không phải một việc mới người dùng gọi được.
- **Lệnh CLI**: không cần lệnh mới. Trần vẫn khai ở `policy.tran` trong
  `.sagent/project.toml`, và `sagent config` **đã** in đủ bốn trần từ trước
  (`inTran` trong `cmd/sagent/main.go`) — đã kiểm, không phải giả định.
- **Endpoint HTTP**: không thêm. Không có trạng thái mới nào để hỏi: "đang chờ
  trần" là một khoảnh khắc trong lúc bước chạy, không phải một trạng thái bước
  lưu vào sổ.
- **Web-UI**: không sửa file nào. Mọi tin báo chờ / cắt bớt / kẹt cứng đi qua
  `a.bus.Warnf`, tức là **mặt nào đang đọc bus thì thấy ngay** — TUI, dashboard,
  workflow board, Telegram — không cần một dòng mã nào ở phía đó.

**Không có dòng nào cần người điều phối cắm hộ.** Nếu muốn thêm (không bắt
buộc), dòng duy nhất đáng thêm là ở `cmdHelp()` mục `flow run`, nói rằng bước
`agent` nay **xếp hàng** thay vì hỏng khi trần chật.

---

## 4. Bảng việc

| Việc | Model | Effort |
|---|---|---|
| Đọc `tran.go`, `runner.go`, `step.go`, tìm đúng chỗ hở | Opus 5 (1M) | Trung bình |
| Thiết kế cổng có hàng đợi trên `XetTran` sẵn có | Opus 5 (1M) | Cao |
| Chọn phép cộng phiên chống đếm hai lần | Opus 5 (1M) | Cao |
| Viết `internal/fleet/cong.go` | Opus 5 (1M) | Cao |
| Cắm vào `do()` + phân giải hồ sơ dùng chung | Opus 5 (1M) | Trung bình |
| Viết bộ kiểm ở CHỖ GỌI, và làm nó đỏ được | Opus 5 (1M) | Cao |
| Dựng sandbox + CLI giả để đo không tốn token | Opus 5 (1M) | Cao |
| Đo thật, phát hiện bản đầu giết lượt chạy, sửa lại | Opus 5 (1M) | Cao |
| Viết báo cáo | Opus 5 (1M) | Thấp |

---

## 5. Nhận xét tự do

**Phép đo thật đã bắt được lỗi mà bộ kiểm không bắt được.** Mười bài kiểm xanh
sạch, `go vet` sạch, và bản sửa vẫn có một chỗ hỏng đủ nặng để phủ định chính
mục đích của nó: giết cả bốn bước vì hai phiên ngoài sắp xong sau một giây. Bộ
kiểm không bắt được vì tôi viết bài kiểm theo đúng cái mô hình trong đầu mình —
và chỗ sai nằm trong chính mô hình đó. Cái sửa nó là năm phút chạy hai lệnh trên
máy thật. Đây là lần thứ hai trong ngày dự án học lại cùng một bài.

**"Đo thật" rẻ hơn tôi tưởng nhiều.** Cả bốn phép đo trên tiêu **0 token, 0
đồng**: dựng sandbox bằng cách đổi `USERPROFILE`, hồ sơ giả, và để Claude thật
dừng ở khâu xác thực. Phiên vẫn chạy qua đúng `FleetStart`, đúng sổ phiên, đúng
`waitSessions` — chỉ là không gọi model nào. Cùng một mẹo dùng lại được cho mọi
mảnh còn nợ "chưa đo trên lượt chạy thật".

**Chỗ nguy hiểm nhất của bản sửa này không phải phần đồng thời.** Là hai chỗ
đếm có thể lệch nhau trong im lặng: (a) phiên của chính cổng bị đếm hai lần nếu
đọc lại sổ sai lúc, (b) cổng đếm một địa chỉ hồ sơ khác với địa chỉ bộ chạy thật
dùng. Cả hai đều **không** gây lỗi, không gây cảnh báo — trần vẫn "hoạt động",
chỉ là canh sai. Nên cả hai đều có bài kiểm riêng và chú thích dài trong mã.
Deadlock thì ầm ĩ và dễ thấy; đếm sai thì im lặng và đắt.

**Trần mặc định chật hơn người dùng tưởng, và đó là một món nợ tài liệu.** Với
bộ mặc định, `max_parallel_sessions = 4` là một lời hứa mà trần provider (3)
không cho giữ: bốn bước `agent` cùng nhà cung cấp `claude` không bao giờ chạy
quá 3 cùng lúc, dù có bao nhiêu tài khoản. Trước bản sửa này chuyện đó vô hình
vì cửa flow không áp trần; sau bản sửa nó thành hành vi thấy được mỗi ngày. Tôi
đã neo nó bằng một bài kiểm, nhưng chỗ đúng để giải thích là tài liệu người
dùng — mà tài liệu thì nằm ngoài vùng của lượt này.
