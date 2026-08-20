# Sổ nợ đo lường

Danh sách MỌI chỗ trong mã Go còn ghi `CHƯA ĐO` / `CHUA DO` / `TODO` / `FIXME`,
kèm **hậu quả nếu đoán sai**, xếp theo mức nguy hiểm giảm dần.

- **Quét lúc**: 21/08/2026, trên nhánh `sagent/phu-2`.
- **Quét bằng**: `git grep -n -i -E "CHƯA ĐO|CHUA DO|TODO|FIXME" -- '*.go'` →
  **55 dòng**, trong **23 file**.
- **Số `TODO` / `FIXME` / `XXX` / `HACK` tìm được: 0.** Dự án này không nợ theo
  kiểu "để mai làm"; nó nợ theo đúng một kiểu — **chưa đo**. Điều đó tốt, vì mọi
  ô nợ đều được đặt tên bằng cùng một từ và đều tra được bằng một câu lệnh.
- **`internal/aiapi/` không có ô nào** (đã kiểm riêng), nên sổ này không phải né
  vùng người khác đang sửa.

## Vì sao cần sổ này khi đã có `sagent nang-luc --chua-do`

Lệnh đó (`cmd/sagent/nangluc.go:16`) in ra **bảng khai năng lực** — 14 ô `ChuaDo`
của 4 adapter. Sổ này thêm ba thứ lệnh đó không có:

1. **Hậu quả nếu đoán sai**, tức lý do để xếp hạng việc nào đo trước.
2. **Những ô nợ nằm ngoài bảng năng lực**: ngưỡng chạy quẩn, cuộc đua N-clone
   cùng refresh, ô token/chi phí của phiên CLI trên dashboard.
3. **Ô khai "LÀM ĐƯỢC" mà bằng chứng chỉ là `--help`** — nguy hiểm hơn `ChuaDo`,
   vì lõi KHÔNG chặn và người vận hành tưởng đã kiểm.

Nguyên văn bài học ở cuối `docs/DO-LUONG.md` (mục 20/08) là lý do sổ này tồn tại:

> Chỗ nào trong mã còn chữ CHƯA ĐO thì chỗ đó là một cái bẫy đang chờ, không phải
> một ghi chú lịch sự.

## Thang mức nguy hiểm

| Mức | Nghĩa |
|---|---|
| 🔴 **ĐỎ** | Đoán sai là **mất tài khoản, mất phiên, hoặc thủng an ninh**. |
| 🟠 **CAM** | Đoán sai là **mất tiền hoặc mất kết luận** — hệ thống chạy tiếp nhưng mù. |
| 🟡 **VÀNG** | Đoán sai chỉ **mất tiện nghi**, hoặc hiện nhầm một thông tin phụ. |
| ⚪ **NỢ NGƯỢC** | **Đã đo rồi mà sổ chưa xoá** — dòng chữ cũ đang nói dối theo hướng khiêm tốn. |

---

# 🔴 MỨC ĐỎ — đoán sai là mất tài khoản hoặc thủng an ninh

## Đ1. Codex khai "LÀM ĐƯỢC" cờ tự-duyệt-quyền nhưng CHƯA CHẠY THẬT

- **Ở đâu**: `internal/provider/codex.go:213-219` (`ArgsTuDuyetQuyen`),
  `internal/provider/codex.go:244` (dòng `Duoc(NLTuDuyetQuyen, ...)`),
  `internal/provider/codex.go:234-236` (ghi chú đầu bảng `NangLuc`).
- **Nó nói gì**: đo `codex --help` thấy
  `--dangerously-bypass-approvals-and-sandbox`, và Codex còn có nấc trung gian
  `--sandbox read-only|workspace-write|danger-full-access` và
  `--ask-for-approval untrusted|on-request|never` mà provider khác không có.
  Nguyên văn: *"CHƯA chạy thật được để xác nhận hành vi (hết hạn mức tới 20/08)"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: đây là ô nguy hiểm **nhất** trong sổ, và nguy hiểm
  chính vì nó **không** khai `ChuaDo`. Khai `LamDuoc` nghĩa là `daDo=true`, nên
  cả hai chốt chặn — `internal/api/api.go:977` (cho `sagent fleet
  --tu-duyet-quyen`) và `internal/api/api.go:1295` (cho bước flow xin
  `tu_duyet_quyen`) — đều **cho qua**. Hỏng theo hai chiều ngược nhau, cả hai đều
  im lặng:
  - **Cờ yếu hơn tưởng** → agent vẫn gặp rào duyệt trong chế độ headless, ngồi
    chờ một câu trả lời không bao giờ tới. Phiên treo, đốt hạn mức, rồi về `lost`
    — mà Codex lại đúng là provider không đọc được kết quả có cấu trúc (xem C1),
    nên `lost` đó sẽ không mang theo lý do nào.
  - **Cờ mạnh hơn tưởng** → `--dangerously-bypass-approvals-and-sandbox` bỏ
    **cả** approval **lẫn** sandbox. `internal/provider/adapter.go:55-56` nói
    thẳng cái giá: *"agent duyệt cả xoá file và chạy lệnh tuỳ ý trong worktree của
    repo thật"*. Nếu `--sandbox workspace-write` mới là nấc hẹp nhất đủ dùng, thì
    ta đang mở rộng hơn cần thiết trên **mọi** lượt Codex.
- **Đối chiếu**: Cursor chọn nấc hẹp nhất một cách có ý thức
  (`internal/provider/cursor.go:161-162`: *"--trust là cờ HẸP NHẤT làm được việc,
  cố ý không dùng --yolo/-f"*). Codex thì chưa ai đo được nấc hẹp nhất là nấc nào.
- **Đo thế nào để đóng**: một lượt `codex exec` thật trong worktree vứt đi, thử
  `--sandbox workspace-write --ask-for-approval never` **trước**, chỉ tụt xuống
  `--dangerously-bypass-...` nếu nấc trên không đủ.

## Đ2. Codex khai "LÀM ĐƯỢC" cờ thư mục nhưng CHƯA CHẠY THẬT

- **Ở đâu**: `internal/provider/codex.go:221-222`, `internal/provider/codex.go:246`.
- **Nó nói gì**: `codex --help` có `-C, --cd <DIR>`. Nguyên văn: *"CHƯA chạy thật"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: agent chạy **sai thư mục** mà không ai biết. Con số
  để so là của Antigravity, đã đo thật (`internal/provider/antigravity.go:132-133`):
  **không có cờ thì 1/3 lượt đúng**, hai lượt còn lại báo *"chưa có repository
  nào được mở"*; có cờ thì 4/4. Tức nếu `-C` không có tác dụng như `--add-dir`,
  ta mất **2/3 số lượt** — và mất theo kiểu agent trả lời trôi chảy về một repo
  khác, chứ không phải theo kiểu báo lỗi.
- Cùng lý do với Đ1: khai `LamDuoc` nên không có chốt nào chặn.

## Đ3. Cursor CHƯA ĐO cờ tự-duyệt-quyền

- **Ở đâu**: `internal/provider/cursor.go:140-141` (`ArgsTuDuyetQuyen` trả
  `(nil, false)`), `internal/provider/cursor.go:164` (`Chua(NLTuDuyetQuyen, ...)`).
- **Nó nói gì**: *"CHƯA ĐO: máy này không cài cursor-agent nên không chạy `--help`
  được"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: hôm nay ô này **đang được xử lý đúng** — hai chốt
  chặn ở `internal/api/api.go:977` và `internal/api/api.go:1295` đều báo lỗi
  (*"không khai bừa; bỏ cờ hoặc dùng provider khác"*) thay vì chạy tiếp. Nợ ở đây
  là nợ **rủi ro người sau**: cách rẻ nhất để "sửa" lỗi đó là mở `cursor.go` chép
  một tên cờ từ tài liệu vào. Lúc đó:
  - Chép **đúng** tên nhưng sai ngữ nghĩa → xem Đ1, thành lỗ hổng an ninh.
  - Chép **sai** tên → `cursor-agent` từ chối dòng lệnh, mọi lượt chết ngay từ
    đối số, nhưng bảng năng lực vẫn khoe xanh trên cả bốn mặt điều khiển.
  `internal/provider/adapter.go:139-141` viết đúng lý do bảng này tồn tại: người
  vận hành đứng trước `sagent fleet cursor:x` **không có cách nào khác** để biết
  điều này trước khi lượt chạy dừng lại hỏi.
- **Chặn được vì**: `internal/api/tuduyetquyen_test.go:61-72` bắt bảng khai lệch
  với `ArgsTuDuyetQuyen()` thật; `internal/api/quyen_test.go:48-52` bắt việc chạy
  tiếp khi chưa đo. Xoá hai bài đó là mở lại cửa.

## Đ4. Antigravity và Cursor CHƯA ĐO hạn token

- **Ở đâu**:
  - `internal/provider/antigravity.go:90-93` (`TokenExpiry` trả `false`),
    `internal/provider/antigravity.go:163` (`Chua(NLHanToken, ...)`).
  - `internal/provider/cursor.go:108-111`, `internal/provider/cursor.go:171`.
- **Nó nói gì**: Antigravity — token nằm trong Windows Credential Manager, *"đọc
  nội dung nó nghĩa là chạm vào chính thứ cần bảo vệ chỉ để lấy một dấu thời
  gian. Chưa đủ lý do"*. Cursor — `auth.json` **có thể** mang dấu thời gian hết
  hạn, nhưng chưa dựng được cảnh token sắp hết hạn để xác nhận đọc đúng trường
  nào.
- **HẬU QUẢ NẾU ĐOÁN SAI**: ô này nối thẳng vào cái bẫy **đã cắn một lần**.
  `internal/api/api.go:950-964` cảnh báo trước khi bung hạm đội — *"token của %s
  còn %s… Qua mốc đó là token bị xoay vòng, mọi bản sao cũ chết"* — nhưng cảnh
  báo đó **chỉ chạy khi `TokenExpiry` trả `ok=true`**. Với hai provider này nó
  không bao giờ chạy. Nghĩa là:
  - Bung hạm đội Cursor/Antigravity ngay trước mốc refresh thì **không có lời
    cảnh báo nào**, trong khi cùng tình huống với Claude/Codex thì có. Sự im lặng
    đó đọc như "an toàn", không đọc như "không biết".
  - Rotation đã đo thật (`docs/DO-LUONG.md`, 20/08): **một** bản refresh là
    **N−1** bản còn lại chết ngay, và hồ sơ gốc cũng nằm trong số đó.
  - Chiều đoán bừa còn tệ hơn chiều im lặng: `internal/provider/cursor.go:109-110`
    nói thẳng *"cảnh báo sai giờ còn tệ hơn không cảnh báo"* — một mốc đoán bừa
    khiến người vận hành hoãn lượt chạy vô cớ, hoặc tệ hơn, **yên tâm** vào một
    mốc sai.
- **Riêng Antigravity**: ô này có thể **không nên đóng**. Lý do từ chối đã ghi
  trong mã và nó là một lý do tốt — đánh đổi "chạm vào Credential Manager" lấy
  "một dấu thời gian" là đánh đổi tồi. Nếu quyết định giữ, nên đổi từ `ChuaDo`
  sang `KhongLamDuoc` kèm chính lý do đó, để nó thôi nằm trong sổ nợ. Grok đã có
  tiền lệ đúng kiểu này: `internal/provider/grok.go:250` khai
  `Khong(NLHanToken, ...)` vì API key **không có** hạn đọc được từ file — một kết
  luận, không phải một khoảng trống.

## Đ5. N bản clone CÙNG refresh một lúc — chưa đo

- **Ở đâu**: `internal/fleet/fleet_test.go:264-265`; `docs/DO-LUONG.md` mục 20/08
  (phần *"Còn treo"*).
- **Nó nói gì**: *"token thật sự bị nhân ra N bản và hành vi refresh đồng thời
  thì CHƯA ĐO"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: phép đo 20/08 đã trả lời **một nửa** câu hỏi —
  provider xoay vòng refresh token, nên **không cần** N tiến trình đua nhau mới
  hỏng. Nửa còn chưa đo là **cuộc đua thật sự**: hai clone cùng vượt mốc hết hạn
  trong cùng vài giây thì bản nào thắng, và bản thua nhận lỗi gì.
  - Bản sửa hiện tại (`profile.Clone` gọi `SyncBackTokens` **trước** khi chép đè,
    `internal/profile/clone.go:35`) đúng **giữa hai lượt chạy**. Nó không nói gì
    về **trong lòng một lượt**: hai tiến trình con đang sống cùng lúc, mỗi đứa
    một thư mục config, `SyncBackTokens` không chạy giữa chừng.
  - Đoán rằng "đã sửa `Clone` là xong" chính là cách mất tài khoản lần thứ hai,
    theo một chuỗi khác nhưng cùng một gốc.
- **Đo thế nào để đóng**: đúng thủ tục đã dùng ở mục 20/08 của `DO-LUONG.md` (sao
  lưu, vân tay SHA-256 8 ký tự đầu, thư mục config tạm nên không mất gì), nhưng
  ép `expiresAt` của **hai** clone về quá khứ rồi bật cả hai cùng lúc.

---

# 🟠 MỨC CAM — đoán sai là mất tiền hoặc mất kết luận

## C1. Codex và Cursor CHƯA ĐO cách đọc kết quả có cấu trúc

- **Ở đâu**: `internal/provider/codex.go:226,248` và
  `internal/provider/cursor.go:148,167` — cả hai `DocKetQua` trả
  `(KetQua{}, false)`.
- **Nó nói gì**: Codex — *"CHƯA ĐO cách đọc dữ liệu có cấu trúc; phiên Codex chết
  vì lý do gì thì sổ để nguyên `lost` chứ không đoán"*. Cursor — *"CHƯA ĐO cách
  đọc dữ liệu có cấu trúc của provider này"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: hai provider này **mù toàn tập** ở mọi mặt hiển thị.
  Không `is_error`, không `usage`, không `total_cost_usd`, không
  `permission_denials`. Cụ thể mất những gì:
  - Mọi phiên Codex/Cursor về `lost` — `internal/provider/trangthai_test.go:128`
    canh đúng điều này và nó **đúng**, nhưng đó là cái đúng của người thành thật,
    không phải của người biết việc.
  - `PhanLoaiChet` không phân biệt được hết-hạn-mức với bị-chặn-quyền với
    lỗi-API. Người vận hành thấy *"chết, chưa rõ vì sao"* rồi phải tự mở log.
  - Không đếm được token/chi phí → không so được provider nào rẻ hơn cho việc gì.
  - **Lá chắn chống chạy quẩn không hoạt động**: `internal/provider/quan.go` cần
    `DemDuocTool`, mà `DemDuocTool` đến từ `DocKetQua`. Một agent Codex chạy quẩn
    399 lần sẽ đốt sạch hạn mức mà không ai chặn.
- **Đoán sai theo chiều ngược**: nếu ai đó "đoán" một bộ đọc cho Codex rồi nó đọc
  nhầm, hậu quả **nặng hơn** không đọc. `internal/provider/quan_test.go:212-214`
  ghim đúng chỗ này: một kết luận "chạy quẩn" đến từ bản ghi ta chưa đọc được là
  kết luận không được phép có — vì nó sẽ giết một lượt chạy lành.

## C2. Bộ đọc kết quả của Grok dựa trên quan sát, không phải hợp đồng

- **Ở đâu**: `internal/provider/ketqua_grok.go:33-36`.
- **Nó nói gì**: *"CHƯA ĐO ĐƯỢC, nói thẳng: Grok không có cờ nào bảo nó xuất JSON
  — định dạng trên là thứ nó tự in ra và ta quan sát được, không phải hợp đồng nó
  cam kết"*. Bằng chứng là lần chạy #29.
- **HẬU QUẢ NẾU ĐOÁN SAI**: khác Codex/Cursor ở chỗ Grok đang khai
  `Duoc(NLKetQuaCoCauTruc)` (`internal/provider/grok.go:246`) — bảng nói xanh.
  Nhà cung cấp đổi cách in **một lần** là:
  - `docDuoc=false` → mọi phiên Grok lặng lẽ tụt về `lost`, hệt Codex, nhưng bảng
    năng lực vẫn khoe xanh và không ai biết vì sao chất lượng vừa tụt.
  - Mất luôn lá chắn chạy quẩn cho Grok — mà Grok là provider khai
    `Khong(NLTuDuyetQuyen)` (`internal/provider/grok.go:239`): **không có rào
    quyền nào để mở**, nó chạy tool tự do theo thiết kế. Đúng provider chạy tự do
    nhất lại là provider mất lá chắn dễ nhất.
  - `ChiPhiUSD` và token của Grok vốn đã là 0 vì bản ghi không có trường nào —
    xem C4, số 0 đó đọc như "miễn phí".
- **Không đóng được bằng cách đo thêm**; chỉ giảm được bằng cách canh: một bài
  kiểm định kỳ chạy `grok -p` thật rồi khẳng định `docDuoc==true`, để ngày nó đổi
  thì ta biết vào **ngày đó**, không phải ba tuần sau.

## C3. Ba provider CHƯA ĐO cách chọn model từ dòng lệnh

- **Ở đâu**: `internal/provider/antigravity.go:137-140,151`,
  `internal/provider/codex.go:227-230,241`,
  `internal/provider/cursor.go:149-152,163`.
- **Nó nói gì**: `ModelArgs` trả `nil`, và cả ba file đều dặn cùng một câu —
  *"nil = chua biet, KHONG phai 'khong co model' — ben goi se canh bao thay vi im
  lang bo qua lua chon cua nguoi dung"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: đây là ô nợ **tốn tiền trực tiếp**.
  `internal/api/api.go:1274-1281` xử lý đúng: bước có `model = "haiku"` nhận cảnh
  báo *"CHƯA ĐO cách chọn model từ dòng lệnh — bỏ qua `model = …`, bước này chạy
  model mặc định"*. Nhưng cảnh báo không phải phép sửa: bước **vẫn chạy model mặc
  định**, tức thường là model đắt nhất. Bình luận ở `internal/api/api.go:1273-1275`
  nói thẳng cái mất: *"im lặng bỏ qua thì người dùng tưởng mình vừa tiết kiệm
  được, mà thật ra vẫn đốt model đắt nhất"*.
  - Đoán bừa tên cờ → CLI con chết ngay từ dòng lệnh, cả bước hỏng, và hỏng theo
    kiểu khó đọc (lỗi đối số của một CLI lạ, không phải lỗi của sagent).
  - Con số để so: Grok **bắt buộc** phải có `-m`
    (`internal/provider/grok.go:236-238`: `grok -p` bỏ qua `defaultModel` trong
    chính file cấu hình của nó). Tức giả định "provider tự đọc model từ hồ sơ" đã
    bị bác **ít nhất một lần** — không được mặc định tin.
- **Chặn được vì**: `internal/api/model_test.go:48-70` bắt cả hai chiều — im lặng
  bỏ qua là đỏ, và chặn không cho chạy cũng là đỏ.

## C4. Token và chi phí của PHIÊN CLI chưa có trong DTO

- **Ở đâu**: `internal/dash/mat2d_test.go:164-182`,
  `internal/dash/ngankeo_test.go:273`.
- **Nó nói gì**: *"Token/chi phi cua PHIEN CLI chua co trong DTO, nen o do phai
  ghi thang la chua do"*. `/api/state` chỉ trả `id/addr/pid/worktree/log/started`.
- **HẬU QUẢ NẾU ĐOÁN SAI**: bình luận trong bài kiểm đã viết sẵn hậu quả và nó
  chính xác: *"Dien 0 vao o tokens la noi doi theo huong de chiu nhat: 0 doc nhu
  'phien nay mien phi', trong khi su that la no dang tieu han muc ma chua ai
  dem"*. Ba cách hỏng đã được ghim thành ba phép kiểm riêng:
  - Điền `0` → card đọc như phiên miễn phí.
  - Đọc trường không tồn tại (`s.tok`, `s.cost`, `s.tokens`, `s.usd`) → `NaN`
    hoặc `undefined` hiện thẳng ra mặt dashboard.
  - Khối tiến độ không đi qua `datSo()` → ô chưa đo hiện số 0
    (`internal/dash/ngankeo_test.go:273`).
- **Nợ thật còn lại**: ô vẫn trống. Không ai biết một phiên CLI tiêu bao nhiêu,
  nên không ghép được chi phí về đúng lượt chạy. Đóng ô này thì `sagent route
  kiem` mới có số thật để so.

## C5. Ngưỡng chạy quẩn `TranLapLienTiep = 10` mới đo được một chiều

- **Ở đâu**: `internal/provider/quan.go:42-47`.
- **Nó nói gì**: *"ca quẩn duy nhất đo được là 399 lần liên tiếp — cách ngưỡng
  rất xa, nên nó chứng minh ngưỡng KHÔNG BỎ SÓT ca thật. Mặt kia (có bắt oan lượt
  bình thường không) thì CHƯA ĐO ĐƯỢC: chưa đếm chuỗi lặp dài nhất trên một bản
  ghi lượt-chạy-bình-thường nào"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: hai chiều, không đối xứng.
  - **Ngưỡng quá thấp** → vu oan một lượt làm việc thật là "chạy quẩn" rồi giết
    nó. Mất công việc đã làm được, và mất theo kiểu người dùng không tin nữa —
    một lần bắt oan là lần sau người ta tắt hẳn lá chắn.
  - **Ngưỡng quá cao** → 399 lần vẫn bị bắt, nên chiều này đang an toàn **với ca
    đã gặp**. Không có gì bảo đảm ca sau cũng lặp tới 399.
  - Khoảng cách 10 → 399 là **gần 40 lần**. Rộng thế nghĩa là ngưỡng hiện tại gần
    như chắc chắn không bỏ sót, nhưng cũng nghĩa là ta không biết mình đang đứng
    cách mép "bắt oan" bao xa.
- **Đo thế nào để đóng**: đếm chuỗi lặp liên tiếp dài nhất trên các bản ghi lượt
  chạy **bình thường** đã có trong sổ. Chính `internal/provider/quan.go:45-46`
  dặn: *"Khi nào đo được thì chỉnh theo số, đừng chỉnh theo cảm giác"*.

## C6. Cursor CHƯA ĐO cách khai thư mục làm việc

- **Ở đâu**: `internal/provider/cursor.go:143-144` (`ArgsThuMuc` trả `nil`),
  `internal/provider/cursor.go:165`.
- **Nó nói gì**: *"CHƯA ĐO: máy này không cài cursor-agent"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: giống Đ2 nhưng nhẹ hơn một bậc, vì bảng khai **thành
  thật** (`ChuaDo`, không phải `LamDuoc`) nên ít nhất `sagent nang-luc` nói ra
  được. Hậu quả thực tế vẫn là agent chạy sai thư mục: theo số đo của Antigravity
  (`internal/provider/antigravity.go:132-133`), không có cờ thì **1/3 đúng**. Với
  Cursor ta thậm chí không biết cờ tên gì để mà thiếu.

---

# 🟡 MỨC VÀNG — đoán sai chỉ mất tiện nghi

## V1. Cả bốn provider CHƯA ĐO "suy cờ từ chính thư mục hồ sơ"

- **Ở đâu**: `internal/provider/claude.go:280`,
  `internal/provider/antigravity.go:156`, `internal/provider/codex.go:247`,
  `internal/provider/cursor.go:166`.
- **Nó nói gì**: bốn câu gần như y hệt — *"CHƯA ĐO: chưa gặp thiết lập nào trong
  ~/.gemini / ~/.codex / Cursor\auth.json phải chuyển thành cờ"*. Claude nói rõ
  hơn: *"model đã truyền tường minh qua --model"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: thấp nhất trong sổ. Bốn ô này gần với "đã đo và không
  cần" hơn là "chưa ai nhìn": ba trong bốn nói *"chưa **gặp** thiết lập nào"*,
  tức đã ngó qua rồi. Đoán sai theo chiều thừa → sinh một cờ CLI con không hiểu,
  lượt chạy chết ngay từ đối số.
- **Đối chứng cho thấy ô này KHÔNG rỗng**: Grok khai `Duoc(NLCoTuHoSo)`
  (`internal/provider/grok.go:243`) vì nó **thật sự** phải đọc `defaultModel`
  trong `.grok/user-settings.json` của chính hồ sơ. Nghĩa là năng lực này có
  thật, chỉ là bốn provider kia chưa lộ ra nhu cầu. Ngày một bản CLI mới thêm
  thiết lập kiểu đó, ô này im lặng chuyển từ "vô hại" thành "chạy sai cấu hình".

## V2. Antigravity CHƯA ĐỌC ĐƯỢC danh tính

- **Ở đâu**: `internal/provider/antigravity.go:84-88` (`Identity` trả `""`),
  `internal/provider/antigravity.go:165`.
- **Nó nói gì**: *"CHƯA ĐỌC ĐƯỢC: sau khi đăng nhập bằng `agy`, không file nào
  trong ~/.gemini bị cập nhật email (google_accounts.json vẫn mang dấu thời gian
  của lần đăng nhập Gemini CLI cũ)"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: dashboard và CLI không hiện được email của tài khoản
  Antigravity — bản thân điều đó là mất tiện nghi. Nhưng chiều đoán bừa **cụ thể**
  ở đây tệ hơn hẳn mức vàng thông thường, và mã đã nhận ra: đọc
  `google_accounts.json` sẽ cho ra **email của lần đăng nhập Gemini CLI cũ** —
  tức một email **có thật, sai người**. Người vận hành nhìn card thấy đúng định
  dạng một email nên không có lý do gì nghi ngờ, rồi bung hạm đội bằng tài khoản
  mình tưởng là tài khoản khác. `internal/provider/antigravity.go:86-87` chốt
  đúng: *"hiện nhầm email còn tệ hơn không hiện gì"*.
- **Bối cảnh giảm nhẹ**: Antigravity khai `Khong(NLTachTaiKhoan)` — **mỗi máy một
  tài khoản Antigravity**. Nên "nhầm tài khoản" ở đây là nhầm giữa danh tính hiện
  tại và một danh tính cũ, không phải nhầm giữa hai tài khoản đang dùng song song.

---

# ⚪ NỢ NGƯỢC — đã đo rồi mà sổ chưa xoá

## N1. Một bình luận vẫn nói rotation "CHƯA ĐO", nhưng đã đo từ 20/08

- **Ở đâu**: `internal/profile/tokenhoisinh_test.go:26-28`.
- **Nó nói gì**: *"bài này KHÔNG khẳng định nhà cung cấp có xoay vòng refresh
  token hay không — cái đó vẫn CHƯA ĐO"*.
- **Thực tế đã đo**: `docs/DO-LUONG.md`, mục *"20/08 — ĐÃ ĐO: nhà cung cấp XOAY
  VÒNG refresh token"*, với vân tay token trước/sau (`5d708911` → `1aa28b8c`) và
  nguyên văn lỗi `OAuth session expired and could not be refreshed`. Ba chỗ khác
  trong mã đã cập nhật theo: `internal/api/api.go:951`,
  `internal/fleet/fleet.go:117`, `internal/profile/clone.go:35`. Commit `7e57225`
  ghi rõ là đóng ô này.
- **HẬU QUẢ NẾU ĐOÁN SAI**: đây là kiểu nợ nguy hiểm riêng — **sổ nói dối theo
  hướng khiêm tốn**. `internal/fleet/fleet_test.go:281-284` đã gặp đúng bẫy này
  một lần và ghi lại bài học: *"Giữ chữ 'chưa đo' sau khi đã đo là nói dối theo
  hướng khiêm tốn, mà người vận hành thì mất đúng thông tin cần biết"*. Người đọc
  `tokenhoisinh_test.go` hôm nay sẽ tưởng vẫn còn cửa "có thể nhà cung cấp không
  xoay vòng", rồi kết luận bản sửa `Clone` → `SyncBackTokens` là phòng xa quá mức
  và nới nó ra. Chính bản sửa đó đang giữ tài khoản sống.
- **Cách trả**: sửa hai câu đó thành "đã đo 20/08, nhà cung cấp XOAY VÒNG; bài này
  vẫn đúng bất kể điều đó, vì nó chỉ khẳng định công refresh không bị đánh rơi".
  **Sổ này không sửa** — luật của lượt làm việc này là chỉ tạo đúng một file.

---

# Lưới an toàn — những bài kiểm đang giữ các ô trên

Phần lớn dòng `CHƯA ĐO` còn lại trong repo **không phải nợ**: chúng là bài kiểm
canh cho các ô nợ ở trên không lặng lẽ biến thành lời nói bừa. Ghi ra đây để
người sau đừng dọn nhầm.

| File:dòng | Canh cái gì | Giữ ô nào |
|---|---|---|
| `internal/api/quyen_test.go:48-52` | Provider CHƯA ĐO mà vẫn chạy tiếp = đỏ | Đ3 |
| `internal/api/tuduyetquyen_test.go:24,61,72` | Bảng khai nói "đã đo" mà `ArgsTuDuyetQuyen()` nói chưa = đỏ | Đ1, Đ3 |
| `internal/api/model_test.go:48,63,70` | Chưa đo model thì phải **cảnh báo** và vẫn **chạy được** — không im lặng, không chặn | C3 |
| `internal/provider/nangluc_test.go:78,230` | `(nil, false)` phải đọc là `ChuaDo`; conformance đối chiếu bảng khai với hành vi thật | toàn bộ mức Đỏ |
| `internal/provider/trangthai_test.go:128` | Provider chưa đọc được kết quả thì phiên **ở lại `lost`**, không kết luận | C1 |
| `internal/provider/quan_test.go:212-214` | Không kết luận "chạy quẩn" từ bản ghi chưa đọc được | C1, C5 |
| `internal/provider/bosungco_test.go:50` | Chưa đọc được kết quả có cấu trúc thì không khai bừa cờ | C1 |
| `internal/dash/mat2d_test.go:165,182` | Ô chưa có số phải ghi `CHUA_DO`; cấm điền `0`, cấm đọc trường không tồn tại | C4 |
| `internal/dash/ngankeo_test.go:273` | Khối tiến độ phải đi qua `datSo()` | C4 |
| `internal/fleet/fleet_test.go:281-289` | Cảnh báo phải nói **đúng số bản** và nói ra hậu quả xoay vòng | Đ5, N1 |
| `internal/fleet/fleet_test.go:296`; `internal/profile/{clone_acl,ditru,profile}_test.go` | Adapter GIẢ khai `ChuaDo` toàn bộ — khai bừa "làm được" là conformance bắt | toàn bộ |

Hai chỗ trong mã thường (không phải test) cũng thuộc lưới này chứ không phải nợ:

- `internal/provider/adapter.go:45-52` — định nghĩa ba trạng thái của
  `ArgsTuDuyetQuyen`, và dặn `(nil, false)` là CHƯA ĐO, người gọi **phải** báo
  lỗi, *"không được lặng lẽ chạy tiếp"*.
- `internal/provider/bosungco.go:5-10` — `TrangThaiCua` trả `ChuaDo` khi adapter
  không khai gì: *"rơi vào đây nghĩa là bảng khai vừa bị thủng — và lúc đó đoán
  'làm được' là cách sai nhất"*.

---

# Thứ tự đề nghị đo

1. **Đ1 + Đ2** (Codex chạy thật) — rẻ nhất trong nhóm đỏ, chỉ cần một lượt
   `codex exec` trong worktree vứt đi, và đang là ô **duy nhất** khai xanh mà
   chưa có bằng chứng chạy thật. Lý do hoãn cũ là hết hạn mức tới 20/08; lý do đó
   đã hết hạn theo chính hạn mức.
2. **Đ5** (cuộc đua N-clone) — thủ tục đã có sẵn ở `DO-LUONG.md` 20/08, chỉ nhân
   đôi. Đây là nửa còn lại của cái bẫy đã cắn một lần.
3. **N1** (xoá dòng nợ ngược) — sửa hai câu bình luận, không phải đo gì.
4. **Đ4 Cursor**, **Đ3**, **C6** — cùng một rào chắn: máy dev chưa cài
   `cursor-agent`. Cài một lần là đóng được ba ô.
5. **C5** (ngưỡng chạy quẩn) — đo từ bản ghi đã có, không cần chạy gì mới.
6. **C4** (token/chi phí phiên CLI) — cần thêm trường vào DTO, đụng nhiều mặt.
7. **Đ4 Antigravity** — cân nhắc **không đóng**, đổi sang `KhongLamDuoc` kèm lý
   do, theo tiền lệ `Khong(NLHanToken)` của Grok.

# Phạm vi sổ này KHÔNG phủ

- **Chỉ quét mã Go.** `internal/dash/web/index.html` có hằng `CHUA_DO` và `docs/`
  có nhiều mục "còn treo"; sổ chỉ nhắc tới khi một dòng Go trỏ vào.
- **Không đo gì mới.** Mọi con số ở đây chép lại từ phép đo đã ghi trong mã hoặc
  trong `docs/DO-LUONG.md`. Không có suy luận nào được nâng lên thành phép đo.
- **Không sửa file nào khác.** Ô N1 nhìn thấy được nhưng để nguyên tại chỗ.
