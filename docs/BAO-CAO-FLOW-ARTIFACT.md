# Ba mảnh còn thiếu của engine flow — artifact, idempotency, compensate

Nhánh: `sagent/flow-22-08` · Ngày 22/08/2026 · Vùng đụng tới: `internal/flow/*`, `internal/store/store.go`

Ba commit, mỗi mảnh một commit:

| commit | mảnh |
|---|---|
| `7508712` | ARTIFACT giữa các bước |
| `b4483da` | IDEMPOTENCY KEY |
| `b383a6f` | Failure policy `compensate` |

Nghiệm thu (chạy từng lệnh riêng, `-count=1`, không dùng cache): `go build ./...` ✔ · `go vet ./...` ✔ · `go test ./...` ✔ (25 gói, 0 đỏ).

---

## 1. Đã làm

### 1.1 ARTIFACT — bước trước để lại FILE cho bước sau

**Vấn đề đo được, không phải suy đoán.** Đường truyền duy nhất giữa hai bước cho tới nay là `{{steps.x.output}}`, và nó đi qua **hai** cái trần, cả hai đều **cắt phần ĐẦU**:

```
agent/shell ─ output ─► store.MaxStepOutput  32.768 ký tự  (store.go:386, cắt lúc LƯU)
                     ─► flow.MaxInject        6.000 ký tự  (flow.go:644, cắt lúc NHÉT sang bước sau)
```

Chọn "giữ phần cuối" là đúng cho một bản tóm tắt và **sai hẳn** cho một bản vá, một file JSON, hay một báo cáo có mục lục ở đầu. Bước sau nhận một mảnh và không có cách nào biết mình đang đọc một mảnh.

**Cách khai:**

```toml
[[flow.va-loi.step]]
  id       = "viet"
  type     = "agent"
  prompt   = "Viết bản vá ĐẦY ĐỦ ra {{artifact_dir}}/ban-va.diff"
  artifact = { ban-va = "ban-va.diff" }        # TÊN → đường dẫn TƯƠNG ĐỐI

[[flow.va-loi.step]]
  id    = "ap"
  needs = ["viet"]
  type  = "shell"
  run   = ["git", "apply", "{{artifacts.ban-va}}"]   # nhận ĐƯỜNG DẪN TUYỆT ĐỐI thật
```

**Sơ đồ đường đi:**

```
                    ┌───────────────────────── ~/.ai-accounts/artifacts/ ────────────────────────┐
                    │                                                                            │
  bước "viet"       │   run-51/                          run-52/        ◄── mỗi LƯỢT một thư mục │
  ─────────────     │     ├── viet/                        ├── viet/        (id do SQLite cấp)   │
  artifact = {      │     │     └── ban-va.diff            │     └── ban-va.diff                 │
    ban-va =        │     └── ap/                          └── ap/      ◄── mỗi BƯỚC một thư mục │
    "ban-va.diff" } │                                                       (bước cùng đợt chạy  │
        │           │                                                        song song)          │
        │ ghi vào   └────────────────────────────────────────────────────────────────────────────┘
        ▼  {{artifact_dir}}
   ┌──────────┐        HỢP ĐỒNG: chạy xong mà không có file  ──► bước HỎNG
   │  FILE    │        (ThieuArtifact, cùng chỗ với `phai_co`)
   └────┬─────┘
        │  {{artifacts.ban-va}} ──► đường dẫn tuyệt đối, KHÔNG qua prompt
        ▼                            ⇒ KHÔNG bị MaxInject cắt, KHÔNG bị MaxStepOutput cắt
  bước "ap"

  Lọc quyền: doc_duoc chặn được CẢ đường này (MoiTruongArtifact),
             không thì cái rào chặn bản tóm tắt và để lọt bản đầy đủ.
```

**Ba câu hỏi bắt buộc trả lời:**

| câu hỏi | trả lời | ở đâu |
|---|---|---|
| Artifact sống bao lâu? | 7 ngày sau khi lượt chạy KẾT THÚC (`ArtifactGiuLai`). Lượt còn `running`/`waiting_approval` thì **không bao giờ** bị dọn, dù cũ tới đâu — dọn ở đó là phá đúng tính năng resume. | `artifact.go` `DonArtifact` |
| Dọn lúc nào? | Ở đầu **mỗi lượt chạy mới** (`Runner.Start`). Cố ý không có lệnh CLI riêng: một nút dọn rác phải nhớ bấm là một nút không ai bấm, và thư mục sẽ phình tới ngày hết đĩa. | `runner.go:117` |
| Hai lượt chạy song song có giẫm lên nhau không? | **Không.** Thư mục tách theo `run-<id>` (id do SQLite cấp, không trùng) và theo `<bước>` (bước cùng đợt chạy song song). Đo bằng `TestHaiLuotChaySongSongKhongGiamLenNhau`: chạy hai lượt trên cùng thư mục dự án, sau lượt 2 thì artifact lượt 1 vẫn nguyên nội dung lượt 1. | `artifact.go` `ArtifactStepDir` |

**Bốn quyết định khác, đều cố ý:**

1. Trỏ bằng **TÊN**, không bằng đường dẫn — `flows.toml` là file người ta gửi cho nhau; đường dẫn tuyệt đối trong đó hoặc là rác trên máy người nhận, hoặc là lời mời ghi đè file bất kỳ của họ. Validate chặn đường dẫn tuyệt đối và `..`.
2. **Xoá sạch thư mục artifact trước MỖI LẦN THỬ.** Không xoá thì lần thử 2 hỏng vẫn "đủ file" nhờ file lần thử 1 để lại — tức là `retry` biến một bước hỏng thành một bước xong.
3. `artifact` + `foreach` bị **chặn**: các lượt lặp chạy song song trong cùng một thư mục và ghi đè lên nhau, còn `{{artifacts.<tên>}}` chỉ trỏ được tới một file.
4. Không thêm cột nào vào DB: đường dẫn dựng lại được từ `runID` + `stepID` + lời khai, nên resume sau khi máy khởi động lại vẫn tìm đúng file.

### 1.2 IDEMPOTENCY KEY — "thế nào là đã làm rồi"

Đây là câu hỏi duy nhất của mảnh này. Ba hướng đã cân nhắc, ghi cả ba ở đầu `internal/flow/idempotent.go`:

| hướng | hậu quả nếu chọn |
|---|---|
| (a) khoá từ **ID bước** | Sửa prompt xong chạy lại thì bước **đã đổi** bị bỏ qua, và lượt chạy dùng lại kết quả của **câu hỏi cũ**, không một dòng nào nói vì sao. Kiểu hỏng tệ nhất trong ba. **Không chọn.** |
| (b) khoá từ **prompt** | Đổi một dấu phẩy là mất sạch cache. Nghe như khuyết điểm, nhưng nó hỏng về **phía an toàn**: làm lại việc đã làm chỉ tốn tiền, bỏ qua việc chưa làm thì ra kết quả sai. |
| (c) khoá **do người dùng khai** | Họ phải tự biết cái gì quyết định kết quả bước. Khai thiếu một biến là quay về đúng lỗi của (a), lần này không có gì cảnh báo. |

**Chọn (b) mở rộng**: khoá = băm của **toàn bộ thứ quyết định kết quả bước, SAU KHI ĐÃ THAY BIẾN** — id, loại, câu hỏi/dòng lệnh đã thay biến, profile, model, route, copies, `tu_duyet_quyen`, tham số plugin, `phai_co`, khai báo artifact.

Vì đã thay biến nên khoá **tự cuốn theo kết quả các bước trước**: bước trước ra kết quả khác → prompt bước sau khác → khoá khác → chạy lại. Đây là thứ (c) không tự có được.

```
    flows.toml + vars + output các bước trước + nội dung artifact đầu vào
                              │
                              ▼  cauHoi(s, env)  ── đúng thứ bước GỬI ĐI, không phải mẫu
                     ┌────────────────┐
                     │  sha256 (128b) │  + profile/model/route/copies/phai_co/artifact
                     └───────┬────────┘
                             ▼
              flow_steps.idem_key   ◄── schema v10, MỘT CỘT, không phải bảng cache riêng
                             │
      tra: có lượt chạy TRƯỚC nào làm XONG đúng khoá này chưa?
                    ┌────────┴────────┐
              không trúng          trúng
                    │                 │
               chạy thật      chép artifact cũ sang thư mục lượt NÀY
                                      │
                             ┌────────┴────────┐
                        chép được          thiếu file
                             │                 │
                   done + "bỏ qua,        chạy THẬT
                    lượt #N đã làm"    (thà tốn tiền còn hơn
                    chi phí = 0         báo xong mà rỗng)
```

**Hậu quả của cách chọn này, nói thẳng:** bộ chạy nhìn thấy prompt, biến, kết quả bước trước. Nó **không** nhìn thấy cây mã trên đĩa, HEAD của git, đồng hồ, mạng. Nên `run = ["go","test","./..."]` có khoá **không đổi** khi mã nguồn đổi. Bật idempotency ở đó là tự bảo rằng test hôm nay vẫn xanh vì hôm qua nó xanh. Vì vậy: **tắt mặc định**, bật từng bước một bằng `idempotent = true`, và `Validate` **cảnh báo thẳng** ở `shell`/`test`/`lint`. Không tự đoán hộ ai.

**Phạm vi:** giữa các **lượt chạy**. Trong một lượt thì `Resume` đã lo từ trước (trạng thái ở SQLite, bước `done` bị bỏ qua).

**Lỗi thật bắt được trong lúc làm** (ghi ra vì nó là loại lỗi khó thấy nhất): `{{artifact_dir}}` chứa số lượt chạy, nên mọi bước có khai `artifact` đều đổi khoá ở mỗi lượt và cache **không bao giờ trúng** — tính năng nằm đó, chạy đúng, không tiết kiệm được gì, và không có gì báo lỗi. Sửa bằng cách chuẩn hoá `artifact_dir` trong `envChoKhoa`. Cùng lý do với việc băm **nội dung** artifact thay vì băm đường dẫn.

### 1.3 COMPENSATE — bước hỏng thì chạy một bước GỠ LẠI

```
  stop        dừng, để nguyên hiện trường.   Việc dở dang nằm lại đó.
  continue    kệ, đi tiếp.                   Việc dở dang cũng nằm lại đó.
  compensate  chạy bước GỠ LẠI, RỒI dừng.    ◄── mới
```

Với việc có tác dụng ra ngoài (tạo nhánh, đẩy commit, dựng máy, mở PR), một nửa việc là thứ tệ nhất: không đủ để dùng, mà lại đủ để lần sau chạy đụng vào.

```toml
[[flow.deploy.step]]
  id         = "tao-may"
  on_failure = "compensate"
  compensate = "xoa-may"

[[flow.deploy.step]]
  id     = "xoa-may"           # KHÔNG có needs — và KHÔNG chạy ở lịch thường
  prompt = "Bước {{buoc_hong}} vừa hỏng. Xoá máy nó đã dựng."
```

```
   đợt 1        đợt 2                       ngoài lịch, ngay lúc sự cố
  ┌──────┐    ┌────────┐  hỏng   ┌──────────────────────────────┐
  │  a   │───►│ tao-may│────────►│  xoa-may   (bước GỠ LẠI)     │
  └──────┘    └────┬───┘         └───────────┬──────────────────┘
                   │                    ┌────┴─────┐
              (không chạy)          gỡ được    gỡ KHÔNG được
                   ▼                    │            │
              ┌────────┐                ▼            ▼
              │ buoc-c │           DỪNG, sạch   DỪNG, "trạng thái KHÔNG BIẾT:
              └────────┘                         cần người vào xem tay"

  xoa-may ở lượt chạy BÌNH THƯỜNG (không có gì hỏng):
      ghi `skipped` + "bước gỡ lại — chỉ chạy khi tao-may hỏng"
```

**Câu hỏi khó 1 — bước gỡ lại mà cũng hỏng thì sao?**

Không thử lại vô hạn, không gỡ-lại-của-gỡ-lại, không đi tiếp. Lượt chạy dừng, và thông điệp nói rõ đây là trạng thái **KHÔNG BIẾT**: việc chính không xong mà cũng chưa gỡ được. Đó là câu khác hẳn "bước x hỏng" — nó là câu *"có người phải vào dọn tay"*. Test đọc thẳng từ sự kiện `FlowFailed` thật, không đọc bản chép tay.

- Không tự thử lại thêm: bước gỡ lại **đã có `retry` của riêng nó**, người viết flow đặt con số đó. Tự ý thử thêm là làm hộ một quyết định họ đã nói ra rồi.
- Không gỡ-lại-của-gỡ-lại: đó là một chỗ treo vô hạn, và cái gỡ ở tầng ba thì không ai hình dung được nó đang gỡ cái gì. Đây là **tính chất của chỗ cắm** chứ không phải một cái cờ ai cũng tắt được: bước gỡ lại chạy qua `runStep`, mà `runStep` không xét `on_failure` (việc đó của `runWave`), nên không có đường nào để đệ quy. `Validate` cảnh báo rằng `on_failure` của bước gỡ lại bị bỏ qua.

**Câu hỏi khó 2 — có gỡ lại các bước ĐÃ XONG TRƯỚC đó không?**

**Không. Chỉ gỡ lại chính bước hỏng.** Ba lý do, theo thứ tự quan trọng:

1. **DAG không phải một ngăn xếp.** Bộ chạy này chạy theo **đợt**, nhiều bước song song. Hai bước cùng đợt **không có thứ tự xong nào cả** — nên "gỡ ngược theo thứ tự đã chạy" là một câu không có nghĩa ở đây. Bịa ra một thứ tự để mà gỡ là bịa ra một sự thật.
2. **Gỡ lan là gỡ sang việc không liên quan.** Flow có ba nhánh độc lập; một bước lá ở nhánh 3 hỏng mà kéo theo undo cả nhánh 1 và 2 thì nó phá đúng những việc đã làm xong đàng hoàng.
3. **Cần gỡ cả chuỗi thì NÓI RA ĐƯỢC**: viết một bước gỡ lại làm trọn việc đó (`terraform destroy` một lần, thay vì ba bước undo lồng nhau). Người viết flow biết cái gì cần gỡ cùng nhau; bộ chạy thì không.

Có test ghim: `TestKhongGoLanSangCacBuocDaXongTruoc`.

**Bẫy lớn nhất, suýt làm mảnh này thành công cụ phá hoại:** bước gỡ lại thường không có `needs` nào, tức là một **GỐC của DAG** — để yên thì nó chạy ngay đợt đầu của **mọi** lượt chạy, gỡ một việc chưa ai làm. Với `git branch -D` hay `terraform destroy` thì đó là phá hoại. Nên bước được ai đó trỏ tới bằng `compensate` bị **loại khỏi lịch chạy thường**.

### 1.4 Một chỗ dọn dẹp phải làm để mảnh 3 đúng

Nhánh `foreach` và nhánh thường trước đây **tự xét `on_failure` riêng**, và chúng đã lệch nhau thật: nhánh `foreach` so bằng với đúng hai giá trị (`step.go`), nên thêm giá trị thứ tư vào là nó lặng lẽ rơi vào nhánh "dừng" và không ai gỡ gì cả. Gộp về một chỗ (`xuLyHong`), có test ghim (`TestForEachHongCungGoiBuocGoLai`).

### 1.5 Bằng chứng: test ĐỎ khi gỡ phần sửa ra

Đã thử **thật**, từng mutation một, khôi phục sau mỗi lần. Không phải một lời hứa.

| # | gỡ cái gì ra | test đỏ | đỏ ra sao |
|---|---|---|---|
| 1 | dòng trộn `arts` vào env trong `runStep` | `TestArtifactChuyenFileNguyenVenGiuaHaiBuoc` | `{{artifact_dir}}` không được thay, bước shell chạy với tên file sống, lượt chạy `failed` |
| 2 | khối `ThieuArtifact` | `TestArtifactKhaiMaKhongCoFileThiBuocHONG`, `TestArtifactDonThuMucTruocMoiLanThu` | cả hai thành `completed` thay vì `failed` |
| 3 | `ChuanBiArtifact` ra ngoài vòng `for attempt` | `TestArtifactDonThuMucTruocMoiLanThu` | file rác của lần thử 1 được tính là artifact hợp lệ; retry biến bước hỏng thành bước xong |
| 4 | nhánh `artifactChoDoc` | `TestDocDuocChanCaArtifactChuKhongChiChanChuoi` | prompt của bước bị cấm nhận đường dẫn thật tới file |
| 5 | `conSong` trong `DonArtifact` | `TestDonArtifactGiuLuotChoDuyetVaXoaLuotDaXong` | dọn 4 thay vì 2 — ăn cả lượt đang chờ duyệt |
| 6 | khối `thuDungLaiViecCu` | `TestIdempotentChayLaiKhongLamLaiViecDaXong` | agent bị gọi 2 lần thay vì 1 |
| 7 | `ghi("viec", cauHoi(...))` — tức quay về hướng (a) | `TestSuaPromptThiKhoaDoiVaBuocCHAYLAI`, `TestBuocTruocDoiKetQuaThiBuocSauCHAYLAI` | sửa prompt xong chạy lại vẫn bị bỏ qua |
| 8 | bỏ qua lỗi `chepArtifactCu` | `TestTrungKhoaNhungArtifactCuBienMatThiCHAYLAI` | báo xong dù artifact cũ đã bị dọn |
| 9 | ghi khoá **trước** khi chạy | `TestBuocHongKhongGhiKhoaIdem` | bước hỏng vẫn ghi khoá; lượt sau bỏ qua việc chưa ai làm xong |
| 10 | `case OnFailCompensate` | `TestCompensateChayBuocGoLaiRoiDUNG`, `TestForEachHongCungGoiBuocGoLai` | bước gỡ lại không chạy (cả hai nhánh) |
| 11 | lọc `goLai` trong `execute` | `TestBuocGoLaiKHONGChayONhungLuotBinhThuong` | bước gỡ lại chạy ở đợt đầu dù không có gì hỏng |
| 12 | biến `buoc_hong` | `TestBuocGoLaiBietMinhDangGoChoBuocNao` | bước gỡ lại không biết mình gỡ cho ai |
| 13 | gỡ xong rồi đi tiếp thay vì dừng | `TestCompensateChayBuocGoLaiRoiDUNG` | lượt chạy thành `completed` |

Chỗ hỏng của #1, #2, #3, #5, #6, #8, #9, #10, #11, #13 nằm ở **chỗ gọi** chứ không ở hàm — nên test tương ứng chạy `Runner.Start` thật, với `store.DB` thật và **tiến trình con thật** (bước `shell` gọi lại chính file test ở chế độ trợ giúp, nên chạy được cả trên Windows lẫn Linux mà không cần cài gì). Kiểm bằng cách mở file trên đĩa và đếm số lần agent bị gọi, không kiểm bằng cách đọc log.

### 1.6 Luật ngang quyền — bốn mặt

**Cả ba mảnh đều KHÔNG phải action mới.** Chúng là ba trường mới trong `flows.toml` (`artifact`, `idempotent`, `compensate`) cộng một giá trị mới cho trường đã có (`on_failure = "compensate"`).

Lý do cụ thể, không phải lời khẳng định suông:

- **`api.Actions`**: không có động từ mới nào. Người dùng vẫn `flow.run` / `flow.show` / `flow.validate` / `flow.save` y như cũ; ba mảnh này đổi *cái flow làm gì*, không đổi *người ta bảo công cụ làm gì*. Một action `artifact.list` chẳng hạn sẽ là một tính năng khác (xem mục 3), không phải mặt thứ nhất của tính năng này.
- **CLI**: `sagent flow validate` đã báo mọi lỗi/cảnh báo mới (chúng đi qua `flow.Validate`, cùng đường với `doc_duoc`, `phai_co`, `vai_tro`). `sagent flow run` chạy chúng. Không có cờ mới nào để thêm.
- **HTTP**: `/api/flow/def`, `/api/flow/save`, `/api/flow/runs` trả `Step` nguyên khối; ba trường mới có thẻ `json` nên chúng đi qua sẵn.
- **Web-UI**: xem mục 2 — **ở đây CÓ một lỗ, và nó có từ trước, không phải do commit này**.

---

## 2. Sự cố

### 2.1 (ĐÃ SỬA) `{{artifact_dir}}` làm cache idempotency không bao giờ trúng

Mô tả ở mục 1.2. Đáng ghi lại vì đây là kiểu hỏng khó thấy nhất: **không có gì báo lỗi**, mọi test riêng lẻ xanh, tính năng chạy "đúng" — chỉ là nó không làm được việc nó sinh ra để làm. Bắt được nhờ có sẵn một test đo **hành vi đầu-cuối** (`TestIdempotentChepArtifactSangLuotMoi`) chứ không phải test gọi thẳng `KhoaIdem`.

### 2.2 (CHƯA SỬA — ngoài vùng của tôi) Bảng vẽ workflow XOÁ TRẮNG mọi trường nó không biết

Đây là lỗ **mặt thứ tư** mà bản brief cảnh báo, và tôi tìm ra nó khi soi xem `artifact` có đi qua bốn mặt không.

`internal/dash/web/flow.html:806` — hàm lưu của bảng vẽ **dựng lại `Step` từ đầu**, chỉ chép đúng những trường nó biết:

```js
const s={id:n.id, type:n.type, needs:n.needs||[], x:n.x, y:n.y,
         timeout_sec:..., retry:..., on_failure:...};
if(n.type==='agent'||n.type==='review'){ s.prompt=...; s.copies=...; s.worktree=...; }
else if(n.type==='shell'){ s.run=...; }
else { s.message=n.message; }
```

Mở một flow trên bảng vẽ rồi bấm **Lưu** — dù không sửa gì — sẽ **xoá vĩnh viễn** khỏi `flows.toml`:

`doc_duoc` · `phai_co` · `vai_tro` · `model` · `plugin` · `vao` · `tham_so` · `route` · `separator` · `fallback` · `tu_duyet_quyen` · **và cả ba trường mới của tôi**.

Nặng hơn: `profile`, `when`, `foreach` được hàm `loadFlow` (dòng 775) **đọc vào** nhưng hàm lưu **không ghi ra** — nên chúng cũng bốc hơi.

Đây là mất dữ liệu im lặng: file trên đĩa đổi, không có cảnh báo, và người dùng chỉ phát hiện khi lượt chạy sau cư xử khác. `phai_co` và `doc_duoc` là **cổng an toàn**; một cái nút Lưu gỡ cổng an toàn ra mà không nói gì là đúng loại hỏng mà dự án này sợ nhất.

**Tôi không sửa** vì `internal/dash/web/*` nằm ngoài vùng được giao (`internal/flow/*`, `internal/store/*`) và có agent khác chạy song song. Cách sửa đúng, theo tôi, là ở **hai** chỗ:

1. `flow.html`: giữ nguyên object gốc của node và chỉ ghi đè những trường bảng vẽ thật sự sửa (`{...s.__goc, id, type, needs, x, y, ...}`).
2. `flow.Save` (vùng của tôi, nhưng tôi **cố ý chưa làm**): trộn với bản trên đĩa để không mất trường lạ. Chưa làm vì nó đổi ngữ nghĩa "lưu" thành "trộn" — sau đó sẽ không có cách nào **xoá** một trường bằng bảng vẽ nữa, và đó là một cái bẫy khác. Quyết định này nên do người sửa `flow.html` cùng quyết, không nên làm một nửa ở một phía.

Có thể chứng minh trong một phút: mở flow `doi-4` (có `doc_duoc`, `phai_co`, `vai_tro`) trên bảng vẽ, bấm Lưu, `git diff .sagent/flows.toml`.

### 2.3 (CHƯA SỬA — phát hiện phụ) `on_failure = "fallback"` gần như không làm gì

Soi `step.go` để cắm `compensate`, thấy nhánh `case OnFailFallback` hiện chỉ in một dòng cảnh báo:

```go
case OnFailFallback:
    r.Bus.Warnf("%s.%s hỏng — bước %s sẽ chạy thay", f.Name, s.ID, s.Fallback)
```

Nó **không gọi** bước `fallback`. Tác dụng thật của `fallback` hôm nay chỉ là *"đừng dừng lượt chạy"* — còn bước được trỏ tới sẽ chạy hay không hoàn toàn phụ thuộc vào việc nó có tình cờ là một node độc lập trong DAG hay không. Câu log thì hứa chắc chắn.

Tôi **không sửa** vì nó không nằm trong ba mảnh được giao, và vì sửa nó là đổi hành vi của một tính năng đang có người dùng (`Validate` đang bắt buộc khai `fallback`, nên có flow đang dựa vào nó). Nhưng hai chuyện đáng nói:

- Câu cảnh báo đang **nói sai sự thật**. Sửa riêng câu chữ thì rẻ và nên làm ngay.
- `compensate` giờ là mẫu cho cách làm đúng: `chayGoLai` gọi thẳng `runStep`, loại bước đó khỏi lịch thường, và ghi `skipped` kèm lý do khi không dùng tới. `fallback` sửa theo cùng khuôn được.

### 2.4 Không có sự cố nào về công cụ

Không dùng `&&` trong PowerShell (chạy từng lệnh riêng qua Bash/POSIX). Không đụng `main`, không đụng `internal/config/*`, `internal/fleet/*`, `internal/api/api.go`, `cmd/sagent/main.go`, `.sagent/project.toml`, `internal/aiapi/*`, `docs/MASTER-PLAN.md`, `docs/DO-LUONG.md`, `docs/SO-NO-DO-LUONG.md`, `master-plan.html`, `internal/dash/web/docs/*`. Không có file `.ps1` nào bị sửa nên không có chuyện em dash.

---

## 3. Bước tiếp theo

Theo thứ tự tôi nghĩ là đáng làm nhất:

1. **Vá lỗ mất dữ liệu ở `flow.html`** (mục 2.2). Đây là việc gấp nhất trong danh sách và không liên quan gì tới ba mảnh này — nó đang ăn `phai_co` và `doc_duoc` ngay lúc này. Nên có test ghim vòng tròn `Save → Load → Save` giữ nguyên mọi trường.
2. **Sửa câu cảnh báo của `fallback`, rồi cắm nó theo khuôn `compensate`** (mục 2.3).
3. **`sagent flow show` chưa nói gì về ba trường mới.** Người ta xem trước một flow mà không thấy bước nào để lại artifact, bước nào sẽ bị bỏ qua vì idempotent, bước nào là bước gỡ lại. Cùng chỗ đó, `flow.tom-tat` / chạy khan (`chaykho.go`) cũng chưa biết — chạy khan hiện sẽ nói bước gỡ lại "sẽ chạy", mà thật ra nó chỉ chạy khi có sự cố. **Đây là chỗ tôi cố ý dừng lại**: cả hai file đó nằm trong `internal/api/`, ngoài vùng được giao.
4. **`foreach` + `artifact`** đang bị chặn. Mở được nếu cho mỗi lượt lặp một thư mục con (`<bước>/<index>/`) và một cách trỏ theo chỉ số. Chưa làm vì chưa rõ cú pháp trỏ nên thế nào, và đoán bừa một cú pháp rồi phải đổi thì đắt hơn chờ.
5. **`foreach` + `idempotent`** đang bị chặn, cùng lý do khác: khoá phải tính trên **từng lượt lặp** mới đúng; một khoá chung cho cả bước sẽ bỏ qua luôn những mục **mới** trong danh sách.
6. **Chưa có cách xem artifact từ dashboard.** Hôm nay muốn đọc file thì phải mở `~/.ai-accounts/artifacts/run-<id>/` bằng tay. Đây mới là chỗ **cần một action mới** (`flow.artifacts` + endpoint + panel) — và khi làm thì phải đủ bốn mặt, kể cả một chỗ trên dashboard thật sự gọi tới nó.
7. **Chưa đo trên một lượt chạy thật.** Tất cả bằng chứng ở trên là từ test đầu-cuối với tiến trình con thật, chưa phải từ một lượt `sagent flow run` với agent thật. Con số đáng đo tiếp: bước gộp báo cáo trước/sau khi chuyển sang artifact tốn bao nhiêu token vào (lượt #34 từng là 10.998 token cho một bước gộp).

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Đọc lại engine flow (`flow.go`, `step.go`, `runner.go`, `state.go`, `doc_duoc.go`, `save.go`, `store.go`) để xác nhận hiện trạng brief mô tả | Opus 5 (1M) | thấp — đọc, không suy luận |
| Thiết kế artifact: nơi lưu, vòng đời, chống giẫm, hợp đồng đầu ra, chống thoát thư mục | Opus 5 (1M) | **cao** — phần lớn công của mảnh 1 nằm ở đây, không nằm ở mã |
| Cài đặt artifact (`artifact.go` + cắm vào `runStep`/`runWave`/`Start`) | Opus 5 (1M) | trung bình |
| Test artifact, kể cả tiến trình con chạy được trên hai hệ điều hành | Opus 5 (1M) | trung bình–cao |
| Định nghĩa khoá idempotency + cân ba hướng và hậu quả từng hướng | Opus 5 (1M) | **cao** — đây là toàn bộ mảnh 2 |
| Cài đặt idempotency (`idempotent.go`, schema v10, `thuDungLaiViecCu`) | Opus 5 (1M) | trung bình |
| Tìm ra bug `artifact_dir` làm cache không bao giờ trúng | Opus 5 (1M) | trung bình — test đầu-cuối bắt hộ, không phải tôi soi ra |
| Thiết kế compensate, đặc biệt hai câu hỏi khó | Opus 5 (1M) | **cao** |
| Cài đặt compensate + gộp `xuLyHong` cho hai nhánh | Opus 5 (1M) | trung bình |
| 13 lần mutation, chạy thật từng lần rồi khôi phục | Opus 5 (1M) | thấp — máy móc, nhưng không bỏ được |
| Soi bốn mặt và tìm ra lỗ `flow.html` | Opus 5 (1M) | trung bình |
| Viết báo cáo này | Opus 5 (1M) | trung bình |

Cả lượt chạy trên một model duy nhất: Opus 5 (1M context), effort mặc định của phiên, không gọi subagent và không dùng workflow.

---

## 5. Nhận xét tự do

**Phần khó nhất của cả ba mảnh không phải mã.** Artifact là ~330 dòng Go và phần lớn trong đó là ghi chú; cái đắt là trả lời ba câu "sống bao lâu / dọn lúc nào / có giẫm nhau không" theo cách mà sáu tháng nữa đọc lại vẫn đúng. Bản brief hỏi đúng ba câu đó trước khi cho viết một dòng nào — và đó là lý do mảnh này không thành một `os.MkdirAll` rồi để đó.

**Mảnh 2 là mảnh duy nhất tôi thấy có thể làm sai theo cách không cứu được.** Artifact làm sai thì bước hỏng, thấy ngay. Compensate làm sai thì hoặc là không gỡ (thấy ngay), hoặc là gỡ nhầm (thấy rất nhanh, vì có ai đó mất một cái nhánh git). Idempotency làm sai thì **bỏ qua việc** — và một việc bị bỏ qua trông y hệt một việc đã làm xong. Nó đi vào báo cáo là `done`, đi vào bảng là màu xanh, và không tốn đồng nào nên nhìn còn đẹp hơn. Đó là lý do tôi chọn "tắt mặc định + cảnh báo thẳng vào mặt ở `shell`/`test`/`lint`" thay vì cố làm cho nó thông minh hơn. Một cái cache biết mình không biết gì thì đỡ hại hơn một cái cache đoán giỏi.

**Cái bug `artifact_dir` là bài học đáng nhớ nhất của lượt này.** Nó không làm đỏ một test nào nếu tôi chỉ viết test gọi thẳng `KhoaIdem` — hai lời gọi với cùng một `Step` sẽ ra cùng một khoá, xanh, xong. Nó chỉ lộ ra vì có một test chạy **hai lượt flow thật** rồi hỏi "lượt hai có bỏ qua không". Câu "test chỉ gọi thẳng hàm là chưa đủ khi chỗ hỏng nằm ở chỗ gọi" trong brief không phải một lời khuyên chung chung — nó vừa cứu tôi một lần trong chính lượt này.

**Về lỗ `flow.html`:** tôi tìm ra nó vì brief bắt kiểm bốn mặt chứ không phải ba, và vì nó kể chuyện `plugin.list` đủ ba mặt mà không mặt nào gọi tới. Nếu tôi chỉ kiểm "trường mới có thẻ json chưa" thì đã trả lời "đủ bốn mặt" và sai. Đáng chú ý là lỗ này **không phải kiểu "thiếu một mặt"** như `plugin.list` — nó là kiểu ngược lại: mặt thứ tư có tồn tại, có gọi tới, và đang **phá** ba mặt kia. Có lẽ đáng thêm vào luật ngang quyền một vế: *mặt web-UI không những phải gọi tới, mà còn phải không được làm mất thứ nó không hiểu.*

**Chỗ tôi ít chắc chắn nhất:** quyết định "compensate = gỡ rồi **DỪNG**". Có lập luận cho hướng ngược lại — gỡ xong rồi cho `needs`-đã-thoả đi tiếp, coi như bước đó chưa từng xảy ra. Tôi chọn dừng vì bước sau thường cần **kết quả** của bước vừa bị gỡ, và cho nó chạy tiếp trên nền không có gì là đổi một lỗi thấy được thành một lỗi im lặng. Nhưng đây là quyết định có thể phải xét lại khi có flow thật dùng nó — và nếu xét lại thì nên là một giá trị thứ năm (`compensate_continue`) chứ không phải đổi nghĩa của cái đã có.

**Ba mảnh này ăn khớp nhau hơn tôi dự tính lúc đầu**, và mỗi chỗ khớp là một chỗ suýt hở:
- `doc_duoc` phải chặn cả artifact, không thì cái rào chặn bản tóm tắt và để lọt bản đầy đủ.
- Khoá idempotency phải băm **nội dung** artifact, không thì cache không bao giờ trúng.
- Bước gỡ lại **không được** `idempotent`, không thì lần sự cố thứ hai không ai gỡ.
- Trúng cache mà artifact cũ đã bị `DonArtifact` dọn thì phải coi như **không trúng**.

Bốn chỗ đó không nằm trong yêu cầu của mảnh nào cả — chúng chỉ xuất hiện khi ba mảnh đứng cạnh nhau. Nếu làm ba lượt riêng biệt bởi ba agent thì nhiều khả năng cả bốn đều lọt.
