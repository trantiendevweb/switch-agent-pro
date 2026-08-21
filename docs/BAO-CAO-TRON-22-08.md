# Báo cáo: trộn hai nhánh 22/08 + rà nhánh plugin

Nhánh: `sagent/tron-22-08` (tạo từ `main` = `f73bc27`).

---

## 1. Đã làm

### 1.1 Trộn — 0 xung đột thật

| Bước | Kết quả | Số đo |
|---|---|---|
| `sagent/phu-1` (2e6c48a) | fast-forward | 930+ / 72−, 4 file |
| `sagent/tns-1` (ba0bda3) | merge, **không xung đột** | 3.119+ / 52−, 26 file |
| Sinh lại HTML | 1 commit | 86b38be |

Ba commit trên nhánh trộn: `2e6c48a` → `f104aaa` (merge) → `86b38be`.

**Ba file dự đoán chắc chắn xung đột thì git tự trộn được cả ba.** Lý do: hai
nhánh sửa các vùng khác nhau của cùng file — phu-1 viết vào khu threat model
(dòng ~423–461), tns-1 viết vào khu plugin (dòng ~670–709). Không bên nào đè bên
nào, nên "giữ cả hai phần nội dung" là thứ git đã làm sẵn. Đã kiểm bằng tay chứ
không tin mặc định:

- `docs/MASTER-PLAN.md:423,448,461` — phần threat model của phu-1 còn nguyên.
- `docs/MASTER-PLAN.md:670-709` — phần plugin của tns-1 còn nguyên.
- Tìm dấu xung đột sót (`<<<<<<<` / `>>>>>>>`) trên cả ba file: rỗng.

### 1.2 Hai bản MASTER-PLAN khớp từng dòng

`docs/MASTER-PLAN.md` và `internal/dash/web/docs/MASTER-PLAN.md`: **1.112 dòng,
`diff` rỗng**. Lỗi lệch 95 dòng từng gặp trước đây không tái diễn.

### 1.3 File .html — sinh lại, không sửa tay

- `internal/dash/web/docs/master-plan.html`: chạy `go run ./tools/sinhkehoach/cmd/sinhkehoach`
  → **nội dung KHÔNG đổi**. Đây là bằng chứng bản git tự trộn ra đúng bằng bản
  generator sinh, chứ không phải phỏng đoán.
- `master-plan.html` (gốc repo, không phải `docs/`): chạy
  `python tools/md2html.py docs/MASTER-PLAN.md master-plan.html "Master Plan"`
  → **có đổi**, 103 KB. File này vốn đã lệch từ trước vì không nhánh nào cập
  nhật nó; giờ đồng bộ.

### 1.4 Nghiệm thu — ba lệnh chạy RIÊNG BIỆT, cả ba xanh

| Lệnh | Kết quả |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./...` | exit 0 — 23 gói ok, 0 FAIL |

---

## 2. Rà nhánh plugin (3.147 dòng, chưa ai đọc lại)

### (a) `internal/plugin/quyen.go` có THẬT SỰ chặn được không?

**Không. Nó chỉ KHÔNG CẤP ĐƯỜNG DẪN — đó không phải là chặn.**

Bản thân `quyen.go` không thực thi gì cả: cả 134 dòng chỉ là bảng mô tả
(`MoiQuyen`, `MoTaCuaQuyen`, `KhoaQuyen`, `ChanDuocThat`). Hàng rào thật nằm ở
`chay.go` và chỉ có đúng ba cơ chế:

| Cơ chế | Chỗ | Làm gì |
|---|---|---|
| `chonThuMuc` | `internal/plugin/chay.go:139-146` | không khai quyền → `cmd.Dir` = thư mục tạm rỗng |
| `dungMoiTruong` | `internal/plugin/chay.go:118-130` | không khai quyền → env dựng lại từ danh sách trắng |
| `docSecret` | `internal/plugin/chay.go:151-153` | không khai quyền → không mở kho key |

Không có ACL, không có Job Object, không có AppContainer, không có bất kỳ lời gọi
hệ điều hành nào giới hạn I/O. **Cả ba đều là "không đưa thông tin", không phải
"dựng hàng rào".**

**PHÉP ĐO THẬT (không suy từ code).** Dựng một plugin có manifest **không khai
quyền nào**, cho nó đường dẫn thư mục dự án nhúng sẵn lúc build, chạy qua host
thật:

```
KET QUA DO: ghi=THANH-CONG  doc=THANH-CONG(noi-dung=MAT-KHAU-DASH-12345)
            liet-ke=THANH-CONG(2 muc)
XAC NHAN: file do plugin ghi CO THAT trong thu muc du an
```

Plugin không xin gì vẫn **ghi được** file vào thư mục dự án, **đọc được** nội
dung bí mật, và **liệt kê được** thư mục. (Test đo là file tạm, đã xoá sau khi
đo — không commit.)

**Chỗ bảng quyền nói quá.** `internal/plugin/quyen.go:60` ghi
`QuyenThuMuc: Chan = ChanThat`. Theo nghĩa hẹp của tác giả thì đúng ("host không
đưa đường dẫn"), nhưng người vận hành đọc cột `[chặn] = chặn thật` sẽ hiểu là
**plugin bị chặn khỏi thư mục dự án** — phép đo trên nói ngược lại. Đây là đúng
thứ mà chính file đó tự cảnh báo ở `internal/plugin/quyen.go:10-14` ("gộp hai
cột lại là chỗ mọi hệ thống quyền nói dối"), chỉ là nó tự vấp vào ở dòng 60.

Công bằng với tác giả: hai quyền còn lại thì **thành thật** — `QuyenGhi` khai
`KhongChanDuoc` (`internal/plugin/quyen.go:69-75`), `QuyenMang` khai `ChuaDo`
(`internal/plugin/quyen.go:101-105`), kèm câu "không khai KHÔNG có nghĩa là nó
bị chặn".

### (b) `rpc.go` xử lý được ba ca không?

**Code xử lý được cả ba. Nhưng chỉ MỘT ca có test đo.**

| Ca | Xử lý ở đâu | Có test không |
|---|---|---|
| Plugin **treo** | `chay.go:285-289` (`context.WithTimeout`, mặc định 60s ở `chay.go:31`) → `chay.go:315-323` hết giờ thì `Process.Kill()` | **KHÔNG** |
| Plugin **chết giữa chừng** | `chay.go:352-356` bắt `io.EOF` → "đóng ống trước khi trả lời", kèm 500 byte cuối stderr (`chay.go:373-381`) | **KHÔNG** |
| **Lệch phiên bản** | hai chiều: plugin từ chối `rpc.go:210-218`; host đối chiếu `chay.go:252-254` | **CÓ** — `TestLechGiaoThucThiDungNgayLucBatTay` (`e2e_test.go:283`) |

Danh sách 34 test của nhánh plugin không có tên nào chạm tới timeout hay tiến
trình chết. Cơ chế viết đúng nhưng **chưa được đo**, mà bộ test này lại rất
nghiêm ở chỗ khác — nên đây là lỗ hổng lệch chuẩn so với chính nó.

Điểm nhỏ khác: `chay.go:333-339` kiểm `t.Error != nil` **trước** khi kiểm
`t.ID != c.id`. Plugin trả lỗi kèm ID sai thì host vẫn nhận lỗi đó. Không nguy
hiểm (một Client = một lượt gọi tuần tự, `chay.go:278-280`) nhưng lệch thứ tự.

### (c) Luật ngang quyền — thiếu MẶT WEB-UI

Luật của dự án là **bốn** mặt, không phải ba (`internal/api/api.go:1-9`).

| Mặt | `plugin.list` | Chỗ |
|---|---|---|
| Hợp đồng `api.Actions` | CÓ | `internal/api/api.go:110` |
| Lệnh CLI | CÓ | `cmd/sagent/main.go:96` → `cmd/sagent/plugin.go` (`list`, `quyen`) |
| Endpoint HTTP | CÓ | `internal/dash/server.go:81` → `internal/dash/plugin_api.go:15` |
| **Đường vào từ web-UI** | **THIẾU** | không có |

Tìm `api/plugins` trong `internal/dash/web/` chỉ ra **2 kết quả, cả hai đều
trong tài liệu** (`internal/dash/web/docs/MASTER-PLAN.md:674`,
`internal/dash/web/docs/master-plan.html:884`). **Không file JS/HTML nào của
dashboard gọi endpoint này.** Người vận hành mở dashboard không có chỗ nào bấm
để thấy bảng plugin.

Test ngang quyền `TestMoiHanhDongDeuCoDuongVaoTuWeb`
(`internal/dash/lachan_test.go:133`) **vẫn xanh**, vì nó chỉ hỏi "đường HTTP có
trả khác 404 không", không hỏi "có ai bấm được không".

**Đây là lỗi lặp lại, repo đã gặp y hệt một lần.**
`internal/dash/suckhoeroute_test.go:13-22` chép lại nguyên văn tình huống cũ của
`route.kiem`: endpoint có, CLI có, test ngang quyền xanh, mà dashboard không có
chỗ nào gọi tới — và phải viết **sáu** test mới để đóng. `plugin.list` đang ở
đúng trạng thái đó.

**Chưa tự sửa, có lý do.** Làm đúng chuẩn repo nghĩa là: thêm khối UI + nút vào
`#ngankeo` của `index.html`, nối vào 2D lẫn 3D, rồi viết bộ test kiểu
`suckhoeroute_test.go`. Vượt xa 30 phút, và `index.html` là file các agent chạy
song song dễ đụng nhất. Ghi lại đây cho lượt sau — xem mục 3.

### (d) Test có ĐỖ khi gõ phần sửa ra không? — Đã thử ba lần, đỏ cả ba

| Gõ ra | Chỗ | Test | Kết quả |
|---|---|---|---|
| Bỏ `m.Co(QuyenThuMuc)` trong `chonThuMuc` | `chay.go:139` | `TestKhongKhaiThuMucThiKhongThayThuMucDuAn` | **ĐỎ** |
| Luôn `os.Environ()` trong `dungMoiTruong` | `chay.go:120-124` | `TestKhongKhaiMoiTruongThiKhongThayBienCuaCha` | **ĐỎ** |
| Vô hiệu đối chiếu quyền exec ↔ manifest | `chay.go:260` | `TestExecXinQuyenNgoaiManifestThiTuChoi` | **ĐỎ** |

Cả ba thông điệp lỗi đều **chỉ đúng chỗ hỏng**, không chỉ báo "không bằng nhau":

- "không khai quyền mà tiến trình con VẪN ĐỨNG trong thư mục dự án (thấy
  sagent-dau-moc.txt) — cmd.Dir đang để rỗng ở đâu đó"
- "plugin xin thêm quyền mang mà host vẫn nhận — bản duyệt manifest thành giấy lộn"

**Kết luận:** ba hàng rào có thật thì được test canh thật. Vấn đề ở (a) không
phải test yếu — mà là hàng rào **vốn chỉ chặn tới đó**, và bảng quyền mô tả nó
rộng hơn thực tế.

`chay.go` đã khôi phục nguyên vẹn (`git diff` rỗng) sau khi đo.

---

## 3. Phán quyết nhánh `sagent/tns-1-2` → **XOÁ**

### "Xoá 5.010 dòng" là ẢO ẢNH của cách đọc diff

| Số đo | Giá trị |
|---|---|
| `git merge-base main sagent/tns-1-2` | `61e8782` |
| main đi được từ merge-base | **22 commit** |
| tns-1-2 đi được từ merge-base | **1 commit** (`a62ae76`) |
| Nội dung thật của `a62ae76` | **14+ / 2−, MỘT file** |

`git diff main..sagent/tns-1-2` so hai đầu mút, nên 22 commit main **thêm vào**
hiện ra thành "nhánh kia xoá". **Không ai xoá gì cả** — nhánh chỉ tụt hậu 22
commit.

### Commit duy nhất đó đã bị main thay thế bằng bản tốt hơn

Cả hai cùng sửa `TestDoi4CatQuyenDocCuaNguoiSoi`, cùng vì một lý do (bước
`bang-chung` làm `doc_duoc` thành `["kiem-2","bang-chung"]`):

- **`a62ae76`** — whitelist ĐÓNG: `maChamHopLe = {kiem-2, bang-chung}` và đòi
  `len(soi.DocDuoc) == 2`. Thêm bất kỳ máy chấm nào sau này là đỏ.
- **main (`818a51f`)** — blacklist theo TÊN THỢ: cấm `ke-hoach`/`code-go`/
  `code-doc`/`sua`, rồi đòi phải có `kiem-2`. Nguyên văn: "cấm theo TÊN THỢ,
  không phải cho phép theo danh sách đóng"
  (`internal/flow/doc_duoc_test.go:219`).

Bản main ghim **ý định** của bài kiểm (người soi không đọc lời tự khai của thợ);
bản `a62ae76` ghim một **danh sách** sẽ mục theo thời gian. Đã xác nhận
`TestDoi4CatQuyenDocCuaNguoiSoi` **PASS** trên nhánh trộn.

**Khuyến nghị: XOÁ `sagent/tns-1-2`.** Commit duy nhất của nó lỗi thời, và bản
thay thế trong main tốt hơn về thiết kế. Không có gì để cứu.

**Chưa xoá gì** — theo đúng yêu cầu, đây chỉ là khuyến nghị.

---

## 2'. Sự cố

**Không có.** Không mất việc, không phải sửa để cho xanh, không dùng `&&` trong
PowerShell, không đụng `main`, không sửa `docs/DO-LUONG.md` /
`docs/SO-NO-DO-LUONG.md`.

Hai chuyện lệch dự đoán ban đầu, đều theo hướng tốt và đã kiểm chứng thay vì tin
mặc định: (1) ba file "chắc chắn xung đột" thì git tự trộn được — đã soi tay cả
ba; (2) nhánh "xoá 5.010 dòng" thật ra chỉ tụt hậu 22 commit.

---

## 3'. Bước tiếp theo

Xếp theo mức đáng làm trước:

1. **Sửa cột `[chặn]` của `QuyenThuMuc` trong `internal/plugin/quyen.go:60`** —
   đổi `ChanThat` thành `KhongChanDuoc` (hoặc tách thành hai dòng: "không được
   cấp đường dẫn" = chặn thật, "không đọc/ghi được thư mục dự án" = không chặn
   được), kèm phép đo ở mục (a) làm bằng chứng. Đây là chỗ bảng đang hứa hộ host.
2. **Mặt web-UI cho `plugin.list`** — khối + nút trong `#ngankeo` của
   `index.html`, đồng bộ 2D/3D, kèm bộ test kiểu `suckhoeroute_test.go`. Dùng
   skill `sagent-dashboard`.
3. **Siết `TestMoiHanhDongDeuCoDuongVaoTuWeb`** để hỏi "có ai bấm được không",
   không chỉ "có khác 404 không" — nếu không thì lỗ này còn tái diễn lần thứ ba.
4. **Hai test còn thiếu cho `rpc.go`**: plugin treo (đo `Process.Kill()` sau
   timeout) và plugin chết giữa chừng (đo thông điệp có kèm stderr).
5. **Xoá `sagent/tns-1-2`** sau khi người vận hành xác nhận mục 3.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Tạo nhánh trộn, merge phu-1 + tns-1 | Opus 5 (1M) | Thấp — 0 xung đột |
| Kiểm nội dung hai bên còn nguyên, hai bản MASTER-PLAN khớp | Opus 5 (1M) | Thấp |
| Sinh lại 2 file .html | Opus 5 (1M) | Thấp |
| Nghiệm thu build/vet/test | Opus 5 (1M) | Thấp — xanh ngay lần đầu |
| Rà (a) quyen.go + **dựng phép đo thật** | Opus 5 (1M) | **Cao** — phải viết plugin thử nghiệm |
| Rà (b) rpc.go ba ca | Opus 5 (1M) | Trung bình |
| Rà (c) luật ngang quyền bốn mặt | Opus 5 (1M) | Trung bình |
| Rà (d) ba lần mutation + khôi phục | Opus 5 (1M) | Trung bình |
| Phán quyết tns-1-2 | Opus 5 (1M) | Trung bình — bẫy đọc diff |

---

## 5. Nhận xét, rủi ro còn lại

**Nhánh plugin viết tốt hơn mức trung bình của repo này.** Test là e2e thật —
build binary thật, bật tiến trình con thật, nói qua stdio thật — nên ba lần gõ
phần sửa ra đều đỏ đúng chỗ. Chú thích giải thích *vì sao* chứ không *làm gì*.
Chuyện đó hiếm.

**Nhưng có một chỗ nó tự vấp vào cái bẫy nó dựng ra để tránh.**
`internal/plugin/quyen.go:10-14` viết hẳn ra rằng gộp "khai" với "chặn" là chỗ
mọi hệ thống quyền nói dối — rồi dòng 60 gắn `ChanThat` cho `QuyenThuMuc`, trong
khi phép đo cho thấy plugin không xin gì vẫn đọc được mật khẩu trong thư mục dự
án. Hai quyền khác thì thành thật đúng chuẩn (`KhongChanDuoc`, `ChuaDo`). Nên
đây không phải cẩu thả — là chỗ "host không đưa đường dẫn" bị đọc thành "plugin
bị chặn", và khoảng cách giữa hai câu đó chỉ lộ ra khi có ai chạy thử.

**Rủi ro lớn nhất còn lại: plugin model đang mời mã của người khác chạy trên máy
mình, với bảng quyền mô tả rộng hơn thực tế.** Người vận hành đọc
`sagent plugin quyen` thấy `chặn thật` sẽ duyệt một plugin mà lẽ ra họ đọc kỹ
hơn. Cách chặn thật duy nhất đang có là **đừng cài plugin không tin được** — và
câu đó phải nằm trong bảng, không nằm trong báo cáo này.

**Rủi ro thứ hai: mặt web thiếu, và test canh nó vẫn xanh.** Đây là lần thứ hai
repo dính đúng một kiểu (`route.kiem` là lần đầu, có hẳn 6 test kể lại). Bịt lỗ
`plugin.list` mà không siết chính bài kiểm thì sẽ có lần thứ ba.

**Rủi ro thứ ba, nhỏ hơn: cách đọc diff.** `git diff A..B` trên một nhánh tụt
hậu hiện ra y hệt như một vụ xoá lớn. Nhánh `tns-1-2` "xoá 5.010 dòng" hoá ra là
**14 dòng thêm vào một file test**. Lần sau, đọc `merge-base` trước khi đọc
`--stat`.

**Chỗ tôi không chắc:** timeout 60s (`internal/plugin/chay.go:31`) chưa ai đo
trên máy thật với plugin làm việc nặng, và `Client.Chay` luôn truyền `han = 0`
(`internal/plugin/chay.go:272`) nên flow **không cấu hình được** timeout riêng
cho từng bước. Chưa phải lỗi, nhưng sẽ thành lỗi ngay khi có plugin đầu tiên
chạy quá một phút.
