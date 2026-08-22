# Báo cáo: trần đồng thời theo harness / provider / hồ sơ

Nhánh `sagent/tran-22-08`, ngày 22/08/2026.

---

## 1. Đã làm

### 1.1 Vấn đề đo được trước lượt này

`.sagent/project.toml` chỉ có **một** con số điều tiết đồng thời:

```toml
[policy]
max_parallel_sessions = 4
```

Nó đếm TỔNG số phiên đang chạy, và chỉ thế. Chỗ áp nó là `internal/api/api.go`
trong `FleetStart` — `room := m - len(running)`, không hỏi phiên nào thuộc về ai.

Hệ quả: bốn phiên rơi **hết vào một tài khoản** `claude:tns` vẫn hợp lệ. Chúng
đốt sạch hạn mức của đúng tài khoản đó, trong khi `claude:phu` ngồi không. Trần
chung không nhìn thấy chuyện đó vì nó không biết khái niệm "tài khoản".

Trên máy này: 5 provider (claude, codex, cursor, antigravity, grok) và nhiều hồ
sơ (`claude:phu`, `claude:tns`, …). Hạn mức thuê bao tính theo **tài khoản**;
RAM và tiến trình con tính theo **harness**. Một con số không đo được cả hai.

### 1.2 Bốn trần, cộng dồn

Thêm khối `[policy.tran]`. Bốn trần **cộng dồn chứ không thay thế nhau** — một
lượt chạy phải qua được cả bốn, và `max_parallel_sessions` vẫn là trần ngoài
cùng:

| Trần | Đếm gì | Vì sao tách riêng | Khoá |
|---|---|---|---|
| chung | mọi phiên đang chạy | trần ngoài cùng, giữ nguyên hành vi cũ | `policy.max_parallel_sessions` |
| harness | phiên cùng một chương trình CLI | RAM + tiến trình con của MÁY, không liên quan hạn mức | `policy.tran.harness_mac_dinh`, `[policy.tran.harness]` |
| provider | phiên cùng nhà cung cấp, cộng mọi tài khoản | nhà cung cấp siết ở tầng tổ chức/nhà bán lại | `policy.tran.provider_mac_dinh`, `[policy.tran.provider]` |
| hồ sơ | phiên cùng MỘT tài khoản | hạn mức thuê bao tính theo tài khoản — chiều đắt nhất khi vượt | `policy.tran.ho_so_mac_dinh`, `[policy.tran.ho_so]` |

Quy ước: **số 0 = TẮT chiều đó**, giống hệt `max_parallel_sessions` vốn có
(`m > 0` mới áp). Số âm bị từ chối ngay lúc đọc file, kèm câu chỉ cách tắt — vì
người gõ `-1` với ý "bỏ giới hạn" mà để lọt sẽ nhận đúng điều ngược lại (`Con()`
kẹp về 0 = từ chối hẳn).

### 1.3 Mặc định — chạy được ngay, không phải cấu hình gì

```
chung 4 · harness 3 · provider 3 · hồ sơ 2
```

Muốn chạy đủ 4 phiên thì **buộc phải trải ra ít nhất hai tài khoản**. Đó chính
là sự cố đã đẻ ra lượt việc này, và mặc định đã chặn nó mà không cần ai sửa file
nào. Có bài kiểm canh đúng con số ấy (`TestMacDinhDaChanDonBonPhienVaoMotTaiKhoan`,
`TestKhongKhaiGiVanCoTranMacDinh`).

**Nói thẳng về ba con số này:** chúng là **lựa chọn**, không phải số đo. Không có
phép đo nào trên máy này nói "claude chịu được đúng 3 tiến trình". Trần harness
mặc định chỉ là hàng rào để một lượt lỡ tay không kéo sập máy; ai đo được máy
mình chịu bao nhiêu thì khai đè. Riêng `hồ sơ = 2` thì có lý do cụ thể: 2 là số
thấp nhất còn giữ được ý nghĩa song song của `sagent fleet`, hạ tiếp xuống 1 là
vô hiệu hoá lệnh đó cho mọi người dùng đang có.

**Đây là đổi hành vi, không phải thêm tính năng câm.** `sagent fleet claude:tns
--copies 4` trước lượt này chạy 4 bản, sau lượt này chạy 2 bản và in cảnh báo
giải thích. Cố ý — nhưng người vận hành cần biết trước.

### 1.4 Harness ≠ provider ở chỗ nào

Đo 22/08 bằng `Command()` của từng adapter: năm provider ra **năm binary khác
nhau** — `claude.exe`, `codex`, `cursor-agent`, `agy.exe`, `grok`. Nên mặc định
harness của một provider **chính là tên provider đó**.

Nghĩa là khi không ai khai gì, trần harness và trần provider đếm **đúng cùng một
tập phiên** và mức chật hơn thắng. Hai chiều chỉ tách nhau khi có khai gộp:

```toml
[policy.tran.thuoc_harness]
codex = "node"
grok  = "node"
```

Lúc đó một phiên `grok` ăn vào trần máy của `codex` (cùng runtime node), nhưng
hạn mức của grok thì không đụng gì tới hạn mức của codex. Mã **không tự đoán**
binary nào dùng chung runtime nào — đoán thì sẽ sai, và sai kiểu đó không ai báo.

### 1.5 Thông báo khi chạm trần

Yêu cầu là "không được im lặng xếp hàng hay im lặng từ chối". Hai ca:

- **Còn chỗ nhưng ít hơn xin** → cắt bớt + cảnh báo qua `bus.Warnf` (nên cả bốn
  mặt điều khiển đều thấy, không riêng terminal).
- **Hết sạch chỗ** → từ chối, và câu lỗi gọi tên trần chặn.

Cả hai câu đều nói: trần **nào** chặn, nó đang ở **đâu** (trần/đang chạy/còn),
**bốn trần** lúc này đứng ở đâu, và **làm gì tiếp** kèm đúng khoá TOML cần sửa.
Khoá in ra có bọc nháy khi tên chứa dấu hai chấm, để người vận hành chép thẳng
vào file không bị lỗi cú pháp — lời khuyên chép vào là hỏng thì tệ hơn không
khuyên (có bài kiểm: `TestKhoaTomlChepVaoDuoc`).

### 1.6 Một lỗ đã kiểm và KHÔNG có

Đường `flow` (bước `agent`) đi qua `agentBridge.RunAgents` → `FleetStart`, nên
nó chịu **đúng bốn trần đó**, không lách được. Đã khoá bằng
`TestDuongFlowChiuChungTranVoiFleet`: nếu ai đó sau này cho `RunAgents` tự bật
phiên, bài này đỏ.

### 1.7 File đã sửa

| File | Sửa gì |
|---|---|
| `internal/fleet/tran.go` (mới) | `Tran`, `XetTran`, `Muc`, `KetTran`, hai câu thông báo |
| `internal/fleet/tran_test.go` (mới) | phép tính: cộng dồn, gộp harness, câu chữ, số âm |
| `internal/config/config.go` | `TranDongThoi`, mặc định 3/3/2, validate, `Sample` |
| `internal/config/tran_test.go` (mới) | đọc TOML thật, mặc định, số âm, khoá hồ sơ thiếu provider |
| `internal/api/api.go` | `TranDongThoi()`, `xetTran()`, nối vào `FleetStart` |
| `internal/api/tran_test.go` (mới) | 7 bài ở CHỖ GỌI |
| `cmd/sagent/main.go` | `sagent config` in ba trần + các mục khai riêng |
| `.sagent/project.toml` | khai khối `[policy.tran]` kèm chú thích |

### 1.8 Số đo

| Số | Giá trị |
|---|---|
| Trần trước lượt này | 1 (chung) |
| Trần sau lượt này | 4 (chung + harness + provider + hồ sơ) |
| Bài kiểm mới | 17 hàm test (7 ở `internal/api`, 5 ở `internal/fleet`, 5 ở `internal/config`), một số chạy nhiều ca con |
| Bài đỏ khi gỡ chỗ nối trong `FleetStart` | 3 |
| Bài đỏ khi đổi mặc định `ho_so` 2 → 4 | 2 |
| `go build ./...` | xanh |
| `go vet ./...` | xanh |
| `go test ./...` | xanh, 26 gói |
| Hạn mức thuê bao đã tiêu để đo | 0 (chạy khô, HOME tạm, không hồ sơ thật) |

---

## 2. Sự cố

### 2.1 Đo thật một lượt vượt trần — NGUYÊN VĂN

**Ca cắt bớt.** Chạy khô qua **CLI thật** (`sagent fleet`), HOME trỏ vào thư mục
tạm nên không có hồ sơ nào và không tiêu một đồng hạn mức:

```
cd /tmp/khotran
HOME=/tmp/homekho sagent fleet claude:tns --copies 4 -- -p "tom tat repo"
```

Nguyên văn in ra:

```
  ⚠ TRẦN ĐỒNG THỜI cắt 4 phiên xuống 2 cho claude:tns. Chật nhất: hồ sơ claude:tns — trần 2, đang chạy 0, còn 2 chỗ. Cũng chật: harness claude (còn 3), provider claude (còn 3). Bốn trần lúc này — chung: trần 4, đang chạy 0, còn 4 · harness claude: trần 3, đang chạy 0, còn 3 · provider claude: trần 3, đang chạy 0, còn 3 · hồ sơ claude:tns: trần 2, đang chạy 0, còn 2. Muốn đủ 4: chia sang tài khoản khác (`sagent ds` xem còn tài khoản nào), hoặc nới `policy.tran.ho_so."claude:tns" = <số>` trong .sagent/project.toml. Nhưng nới trần hồ sơ nghĩa là các bản đó CÙNG đốt một hạn mức thuê bao — chia sang tài khoản khác thì không.
```

**Ca từ chối hẳn.** Ca này bắt buộc phải có phiên đang chạy trong sổ (trần ≥ 1
mà 0 phiên đang chạy thì luôn còn ít nhất 1 chỗ), và `sagent` không có lệnh nào
thêm phiên bằng tay. Nên đo qua bài kiểm — vẫn là `API.FleetStart` thật, sổ
SQLite thật, phiên đang chạy thật (PID của tiến trình test), chỉ khác là gọi qua
Go thay vì qua CLI. Câu chữ là cùng một hàm nên nguyên văn không đổi:

```
go test ./internal/api/ -run TestFleetStartTuChoiKhiHetChoTheoHoSo -v
```

Nguyên văn:

```
hết chỗ cho claude:tns: hồ sơ claude:tns đã đầy (trần 1, đang chạy 1).
  Bốn trần lúc này — chung: trần 4, đang chạy 1, còn 3 · harness claude: trần 3, đang chạy 1, còn 2 · provider claude: trần 3, đang chạy 1, còn 2 · hồ sơ claude:tns: trần 1, đang chạy 1, còn 0.
  Ba cách đi tiếp: (1) chạy cùng việc trên tài khoản khác — `sagent ds` xem còn tài khoản nào; (2) `sagent status` rồi `sagent stop <id>` dừng bớt; (3) nới trần trong .sagent/project.toml — policy.tran.ho_so."claude:tns" = <số>.
```

Điểm đáng chú ý của câu này: trần chung còn **3 chỗ trống**. Với thông báo cũ
(`đã đạt trần 4 phiên đang chạy`) người vận hành sẽ đọc xong rồi đi kiểm
`max_parallel_sessions`, thấy 4 và 1 phiên chạy, và không hiểu gì cả. Câu mới
gọi thẳng tên trần thật sự chặn.

`t.Logf` in nguyên văn được **giữ lại trong test**, không phải mã tạm: ai sửa
câu chữ này về sau phải nhìn thấy nó đổi thành gì.

### 2.2 Bằng chứng test ĐỎ khi gỡ phần sửa ra

Chạy hai phép gỡ thật, rồi khôi phục.

**Phép gỡ 1 — bỏ chỗ nối, giữ nguyên `internal/fleet`.** Thay khối
`a.xetTran(...)` trong `FleetStart` bằng đúng khối trần chung cũ:

```
--- FAIL: TestFleetStartTuChoiKhiHetChoTheoHoSo
    câu từ chối thiếu "hồ sơ claude:tns"
    Nguyên văn: không có claude:tns — tạo trước bằng: sagent them claude:tns
--- FAIL: TestFleetStartCatBotThiPhaiNoiTranNao
    cắt 4 xuống 2 mà không có cảnh báo nào
--- FAIL: TestTranChungVanLaTranNgoaiCung
    không gọi tên trần chung. Nguyên văn: da dat tran 2 phien dang chay
FAIL	internal/api

ok  	internal/fleet   (vẫn xanh)
ok  	internal/config  (vẫn xanh)
```

Đây đúng là điều cần chứng minh, và là bài học mà dự án này vừa trả giá hôm nay:
**test ở chỗ tính một mình là chưa đủ.** `internal/fleet` và `internal/config`
vẫn xanh sạch trong khi sản phẩm đã mất hẳn tính năng. Chỉ bài ở CHỖ GỌI mới đỏ.

**Phép gỡ 2 — đổi con số mặc định** `ho_so_mac_dinh` từ 2 lên 4:

```
--- FAIL: TestKhongKhaiGiVanCoTranMacDinh
    trần hồ sơ mặc định (4) không siết gì thêm so với trần chung (4)
--- FAIL: TestMacDinhDaChanDonBonPhienVaoMotTaiKhoan
    mặc định: mức chật nhất phải là hồ sơ, được [{harness claude 3 0 ...} {provider claude 3 0 ...}]
```

Tức con số mặc định cũng có bài canh, không phải một hằng số ai đổi cũng được.

Đã khôi phục cả hai file (`internal/api/api.go`, `internal/config/config.go`) và
chạy lại — xanh.

### 2.3 Luật ngang quyền bốn mặt — việc này KHÔNG đẻ ra action mới

Nói thẳng theo yêu cầu, không để người đọc tự suy.

**Lượt này không thêm action nào vào `api.Actions`.** Đây là **cấu hình + siết
chặt hơn ở một đường đã có**, cụ thể là `fleet.start` — action đó đã đủ bốn mặt
từ trước:

| Mặt | Đường vào |
|---|---|
| hợp đồng `api.Actions` | `"fleet.start"` (`internal/api/api.go:77`) |
| lệnh CLI | `sagent fleet` (`cmd/sagent/main.go:74`) |
| endpoint HTTP | `/api/fleet` |
| đường vào từ web-UI | `internal/dash/web/index.html:1676` — `POST /api/fleet` (đã grep, không suy đoán) |

Nên `TestMoiHanhDongDeuCoDuongVaoTuWeb` **không phải sửa**, và tôi cũng không
thêm tên nào vào danh sách miễn trừ.

Cái mới của lượt này đi tới bốn mặt bằng ba đường đã có sẵn, không cần đường mới:

1. **Chặn** nằm trong `FleetStart` — mọi mặt gọi `fleet.start` đều đi qua nó,
   kể cả đường flow (mục 1.6).
2. **Thông báo** phát bằng `bus.Warnf` — event log của dashboard nghe cùng một
   bus, nên câu cảnh báo hiện ở cả terminal lẫn mặt web mà không phải viết
   thêm JS nào.
3. **Xem trần đang là bao nhiêu** thì mở rộng `sagent config` (`config.show`),
   là lệnh đã có và đã hiện trong `/api/state`.

Đối chiếu với ca `plugin.list` hôm nay: nó có đủ ba mặt hình thức mà không file
JS/HTML nào của dashboard gọi tới — tức mặt thứ tư trống. Ở đây mặt thứ tư không
trống, vì nó dùng lại đúng nút `fleet.start` mà dashboard vẫn bấm và đúng event
log mà dashboard vẫn hiển thị. Tôi **không** kiểm chuyện đó bằng cách hỏi
"endpoint có trả khác 404 không" — tôi kiểm bằng bài đi qua `FleetStart` thật
(mục 2.2), là chỗ mọi mặt đều phải đi qua.

### 2.4 Một điều sửa thêm, không nằm trong yêu cầu

Đường cũ nuốt lỗi đọc sổ: `running, _ := a.db.Running()`. Nuốt xong thì trần
được tính trên một con số **hụt** — tức hỏng theo hướng **cho qua nhiều hơn mức
đáng cho**, đúng hướng tốn hạn mức. Giờ nó cảnh báo, kèm số phiên thật sự đọc
được.

---

## 3. Bước tiếp theo

Xếp theo mức đáng làm, không phải theo thứ tự tiện tay.

1. **Chưa có hàng đợi — vẫn là từ chối, không phải chờ.** Chạm trần thì lượt đó
   bị cắt hoặc bị từ chối ngay. `master-plan.html` đã ghi trạng thái `queued`
   là "CHƯA đo được" vì hệ thống không có cơ chế hàng đợi. Lượt này **không**
   thêm hàng đợi — thêm thì phải có nơi giữ yêu cầu, cơ chế đánh thức, và cách
   huỷ, tức là một tính năng riêng chứ không phải phần đuôi của tính năng này.
   Yêu cầu chỉ nói "không được **im lặng** xếp hàng"; tôi chọn không xếp hàng và
   nói rõ vì sao từ chối.

2. **Trần harness chưa nối với phép đo RAM thật.** Số 3 là hàng rào chọn tay.
   Việc đáng làm tiếp: đo RSS thật của từng loại tiến trình CLI trên máy này
   (gói `internal/process` đã liệt kê được tiến trình con), rồi đề xuất số dựa
   trên RAM còn trống thay vì để người dùng đoán.

3. **`fleet --copies` chưa tự chia sang tài khoản khác.** Lời khuyên trong câu
   thông báo là "chia sang tài khoản khác", nhưng người vận hành phải tự chạy
   lệnh thứ hai. Bước tự nhiên tiếp theo: `sagent fleet claude --copies 4` (không
   nêu tài khoản) tự trải 4 phiên qua các tài khoản claude còn chỗ. Cần cẩn thận
   — tự chọn tài khoản là tự tiêu tiền của người dùng ở chỗ họ chưa nói tới.

4. **Mặt dashboard chưa có ô "còn bao nhiêu chỗ".** Hiện người vận hành chỉ thấy
   trần khi **chạm** vào nó. `KetTran.Bang()` đã dựng sẵn dòng đó; đưa nó vào
   `/api/state` là một lượt việc nhỏ, nhưng đụng `internal/dash` nên tôi để
   ngoài ranh giới file của lượt này.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Khảo sát hiện trạng (`project.toml`, `FleetStart`, `store.Session`, 5 adapter) | Opus 5 (1M) | Trung bình |
| Thiết kế 3 chiều trần + quy ước 0 = tắt + bảng `thuoc_harness` | Opus 5 (1M) | Cao |
| `internal/fleet/tran.go` — phép tính và hai câu thông báo | Opus 5 (1M) | Cao |
| `internal/config` — khai báo, mặc định 3/3/2, validate | Opus 5 (1M) | Trung bình |
| Nối vào `FleetStart` + `sagent config` | Opus 5 (1M) | Thấp |
| 15 bài kiểm, trong đó 7 bài ở CHỖ GỌI | Opus 5 (1M) | Cao |
| Đo thật: chạy khô CLI + hai phép gỡ chứng minh test đỏ | Opus 5 (1M) | Trung bình |
| Báo cáo này | Opus 5 (1M) | Trung bình |

---

## 5. Nhận xét tự do

**Chỗ khó nhất không phải phép tính, mà là chọn con số mặc định.** Đề bài nói
"có mặc định hợp lý, đừng bắt ai cấu hình mới chạy được". Nhưng mặc định hợp lý
duy nhất — `hồ sơ = 2` — **phá vỡ hành vi đang có**: `fleet --copies 4` từ nay
chạy 2 bản. Hai lựa chọn đều dở theo cách riêng: mặc định TẮT thì tính năng nằm
im cho tới khi có người đọc tài liệu (và không ai đọc), mặc định BẬT thì đổi
hành vi dưới chân người dùng. Tôi chọn bật, vì cái sự cố mà đề bài mô tả —
4 phiên đốt sạch một tài khoản qua đêm — đắt hơn hẳn chuyện `--copies 4` chỉ
chạy 2 bản kèm một câu giải thích. Nhưng đây là **đánh đổi**, không phải câu trả
lời đúng, và người quyết định cuối là người vận hành chứ không phải tôi.

**Trần harness gần như trùng trần provider, và tôi để nguyên như vậy.** Đo được:
5 provider → 5 binary khác nhau, nên khi không khai `thuoc_harness` thì hai trần
đếm cùng một tập phiên. Có thể coi đó là một chiều thừa. Tôi giữ vì hai lý do:
chúng đo hai thứ khác hẳn nhau (RAM máy vs. hạn mức nhà cung cấp) nên rồi sẽ
lệch nhau, và ngay bây giờ bảng `thuoc_harness` đã cho gộp `codex` + `grok` vào
một trần node — đó là điểm chúng tách ra thật. Nhưng nếu ai gọi đây là "hai khoá
cho một phép đếm" thì lời chê đó đúng với cấu hình mặc định.

**Điều làm tôi tin lượt này không phải làm hình thức** là phép gỡ ở mục 2.2:
gỡ chỗ nối ra, `internal/fleet` với `internal/config` vẫn xanh sạch trong khi
sản phẩm mất trắng tính năng. Nếu tôi chỉ viết test cho hàm thuần — điều rất dễ
làm, vì hàm thuần dễ test hơn nhiều — thì tôi đã giao đúng cái bẫy mà `plugin.list`
và `route.kiem` đã dính hôm nay: ba mặt đủ, mặt thứ tư trống, test xanh.

**Chuyện tôi không làm và nghĩ là đúng khi không làm:** hàng đợi. Chạm trần lúc
2 giờ sáng thì người vận hành muốn biết "còn bao nhiêu chỗ, làm gì tiếp", chứ
một yêu cầu bị treo trong hàng đợi vô hình còn khó chịu hơn một lời từ chối rõ
ràng. Từ chối thì đọc log là biết; treo thì phải đi tìm xem nó treo ở đâu, và
`master-plan.html` đã ghi sẵn rằng trạng thái `queued` chưa đo được.

**Rủi ro còn lại:** `Running()` là ảnh chụp tại một thời điểm. Hai lệnh `sagent
fleet` chạy đồng thời trong hai terminal đều đọc "đang chạy 0" rồi cùng được cấp
2 — tổng thành 4 trên một tài khoản, đúng cái ta chặn. Đây là đua thật, không
phải giả thuyết: dự án này đã đo được một cuộc đua 16ms ở chỗ refresh token
(ghi trong `fleet.go`). Chặn được nó cần khoá ở tầng sổ, tức là một lượt việc
riêng đụng `internal/store` — ngoài ranh giới file của lượt này. Cửa sổ đua hẹp
(cỡ vài chục ms giữa lúc đọc sổ và lúc ghi phiên đầu tiên), nhưng nó **có thật**
và tôi không muốn nó nằm im trong một báo cáo nói rằng mọi thứ đã xong.
