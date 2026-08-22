# Báo cáo — đưa BẢNG QUYỀN PLUGIN ra mặt web

Nhánh: `sagent/web-plugin-22-08` (tách sạch từ `main`, `git rev-list --count main..HEAD` = 0 lúc bắt đầu).
Ngày: 22/08.

---

## 1. Đã làm

### Vấn đề, nói lại cho gọn

Hợp đồng `api.Actions` có `plugin.list`, CLI có `sagent plugin list` và `sagent plugin quyen`,
endpoint HTTP `/api/plugins` có. Thiếu đúng một mắt: **mặt web**. Người vận hành mở
dashboard không có chỗ nào thấy plugin nào đang cài và nó được cấp gì.

Và lý do việc này đáng làm không phải là "cho đủ mặt". Bảng quyền có cột `[chặn]` trả lời
câu "host có hàng rào THẬT không", và cột đó vừa bị bắt quả tang nói dối hôm 22/08:
`thu-muc-lam-viec` khai `chan-that` trong khi plugin **không xin quyền nào** vẫn đọc được
33 byte, ghi được file, liệt kê được thư mục dự án. Đã hạ xuống `khong-chan-duoc`.

Nên yêu cầu của khối này không phải "hiện danh sách plugin" mà là **hiện cột `[chặn]` với
đủ ba trạng thái rời nhau**. Một mặt web hiện plugin mà bẹp `khong-chan-duoc` với `chua-do`
làm một thì tệ hơn không có mặt web nào — nó cho người ta cảm giác đã kiểm tra.

### Đã làm gì

**a) `internal/dash/plugin_api.go` — 43 dòng đổi**

Thay con số gộp duy nhất `so_chua_chan` bằng **ba ngăn riêng**:

| trường | nghĩa |
|---|---|
| `so_chan_that` | host chặn thật, có phép đo |
| `so_khong_chan_duoc` | ĐÃ ĐO, kết luận là host không chặn được |
| `so_chua_do` | CHƯA ai đo |
| `so_chua_chan` | giữ lại, nay là **tổng** của hai ngăn sau — không mặt nào được hiện mình nó |

Vòng đếm chuyển từ `if q.Chan != "chan-that"` (so chuỗi trần, gộp hai thành một) sang
`switch plugin.TrangThaiChan(q.Chan)` đối chiếu **thẳng với hằng Go**. Nhánh `default` gom
`ChuaDo` và mọi khoá lạ về `so_chua_do` — chiều an toàn: thứ không đọc được phải đếm là
"chưa biết", không phải "đã chặn".

DTO allowlist giữ nguyên: `Secret` vẫn chỉ là `"<tên> → key_id <id>"`, không bao giờ là giá trị.

**b) `internal/dash/web/index.html` — 208 dòng thêm**

Khối **"Plugin đã cài"** trong ô Tổng quan, ngay dưới "Năng lực provider" (cùng loại câu hỏi:
thứ này làm được gì trên máy tôi, và tôi chặn được tới đâu). Mỗi plugin hiện:

- tên · phiên bản · giao thức — **mono**, đúng luật "dữ liệu máy đi mono";
- mô tả;
- **bảng quyền**: khoá quyền (mono) · cột `[chặn]` **viết thành chữ** kèm chấm màu ·
  mô tả quyền · `plugin xin để: <lý do>` · bằng chứng đo;
- dòng secret (chỉ tham chiếu key_id);
- cảnh báo riêng nếu manifest khai `exec` mà không thấy file chạy — "viết sai TOML" và
  "quên build" hỏng theo hai kiểu nên không được hiện ra như cùng một lỗi;
- danh sách manifest hỏng, và thư mục đã tìm.

Cột `[chặn]` — **ba trạng thái, ba lớp CSS, ba màu, ba nhãn chữ**:

| chan | lớp | màu | nhãn |
|---|---|---|---|
| `chan-that` | `.chan` | `--run` | chặn thật |
| `khong-chan-duoc` | `.khongchan` | `--error` | KHÔNG chặn được |
| `chua-do` | `.chuado` | `--pending` | chưa đo |

Vì sao `khong-chan-duoc` ăn `--error` chứ không ăn xám `--idle` như bảng năng lực: ở bảng
năng lực, "provider không có thứ đó" là chuyện trung tính. Ở đây thì không — đó là một
**lỗ hổng đã đo**, đúng dòng vừa bị bắt nói dối. Tô xám thì lời thú nhận ấy đọc như "không
có gì phải làm", tức là bảng lại nói dối lần nữa, chỉ bằng màu thay vì bằng chữ.

Chú thích dưới bảng mở đầu bằng đúng câu của CLI, nguyên văn:

> **Khai một quyền trong manifest KHÔNG tự nó là một hàng rào.**

**c) `internal/dash/plugin_ui_test.go` — 401 dòng, 6 bài kiểm mới**

Ràng buộc dashboard đã giữ:

- **Offline tuyệt đối** — không thêm một URL ngoài nào; không asset mới.
- **Vanilla** — HTML/CSS/JS thuần, không bundler.
- **Không emoji làm icon** — trạng thái vẽ bằng chấm CSS, `TestKhongDungEmojiLamIcon` xanh.
- **`prefers-reduced-motion`** — khối cũ đã có và phủ mọi thứ tôi thêm; tôi **không thêm
  animation nào**, vì không có chuyển động nào ở đây mã hoá được thông tin.
- **Không chế màu mới** — chỉ dùng `--run` `--pending` `--error` `--hi` `--mid` `--lo`.
  `TestKhongTrangNaoKhaiLaiTokenTrangThai` xanh.
- **Mono cho dữ liệu máy** — tên plugin, phiên bản, khoá quyền, nhãn `[chặn]`, thư mục nguồn.
- **Không secret ra client** — có bài kiểm riêng.
- **Số từ state thật** — đếm ở server từ `plugin.Nap`, mặt web không tự cộng.

Dùng lại khung `.nl` có sẵn (container mang cả hai lớp `nl pl`) thay vì chép thành bộ
selector thứ hai — chép là mở đúng khe lệch im lặng mà `token.css` sinh ra để bịt.

### Số đo

| Hạng mục | Số |
|---|---|
| File sửa | 2 (`plugin_api.go`, `web/index.html`) |
| File mới | 1 (`plugin_ui_test.go`) |
| Dòng thêm / bớt (2 file sửa) | +239 / −12 |
| Bài kiểm mới | 6 |
| Trường DTO mới | 3 (`so_chan_that`, `so_khong_chan_duoc`, `so_chua_do`) |
| Đột biến thử để chứng minh test cắn | 11 |
| Đột biến bị bắt | **11 / 11** |
| Asset ngoài thêm vào | 0 |
| Màu mới chế ra | 0 |

### Bài kiểm mới, và bằng chứng chúng ĐỎ thật

Bài chính — `TestDTOPluginKhongDepBaTrangThaiChanThanhHai` — làm theo đúng mẫu
`TestDTOKhongDepTrangThaiNaoThanhTrangThaiKhac`: **đối chiếu từng mục** giữa lớp API
(`(&api.API{}).Plugins(s.workDir())`, đúng thư mục handler dùng) và lớp DTO. Không đếm số
plugin, và cố ý **không** đòi "phải thấy đủ ba trạng thái trong dữ liệu sản phẩm" — đòi thế
thì bài kiểm chỉ xanh chừng nào dự án còn nợ, và nó phạt đúng việc đóng nợ.

Để bài kiểm không xanh vì rỗng, nó dựng một manifest thật (`mau-ba-trang-thai`) trong kho
plugin của HOME test, xin đủ 4 quyền phủ cả ba trạng thái, rồi **chặn cứng** bằng `t.Fatal`
nếu bản mẫu không sinh ra đủ ba.

Bằng chứng cắn — sửa từng chỗ, chạy đúng bài kiểm canh chỗ đó, rồi trả lại ngay
(`/tmp/pha.py`, `/tmp/pha1.py`):

| # | Phá hoại | Bài kiểm | Kết quả |
|---|---|---|---|
| 1 | DTO bẹp `chua-do` thành `khong-chan-duoc` | `...KhongDepBaTrangThaiChanThanhHai` | **ĐỎ** — `mau-ba-trang-thai/mang: lớp API nói chan="chua-do", DTO nói "khong-chan-duoc"` |
| 2 | `so_chua_do` luôn trả 0 | nt | **ĐỎ** — `so_chua_do = 0 nhưng đếm được 1` |
| 3 | DTO nuốt dòng secret | `...ChiMangTenSecretKhongMangGiaTri` | **ĐỎ** — bắt được "xanh vì rỗng" |
| 4 | `PL_LOP` cho hai trạng thái chung một lớp CSS | `...VeBaTrangThaiChanThanhBaThuKhacNhau` | **ĐỎ** — `CÙNG lớp "chuado"` |
| 5 | Hai trạng thái khác lớp nhưng chung một màu | nt | **ĐỎ** — `cùng ăn màu --pending` |
| 6 | `PL_DOC` cho hai trạng thái cùng một nhãn chữ | nt | **ĐỎ** — `cùng đọc là "chưa đo"` |
| 7 | Giá trị lạ rơi về `chan` thay vì `chuado` | nt | **ĐỎ** — `rơi về "chan"... là bảng tự hứa hộ host` |
| 8 | Nhãn đếm đổi sang đọc `so_chua_chan` (tổng đã gộp) | `...KhongHienConSoDaGopCuaCotChan` | **ĐỎ** |
| 9 | Xoá câu "khai quyền không phải hàng rào" khỏi phần hiện ra | `...NoiRoKhaiQuyenKhongPhaiHangRao` | **ĐỎ** |
| 10 | Định nghĩa `napPlugin` nhưng không gọi lúc mở trang | `...NapBangQuyenPluginLucMoTrang` | **ĐỎ** |
| 11 | Bỏ chấm chú thích của `khong-chan-duoc` | `...VeBaTrangThaiChanThanhBaThuKhacNhau` | **ĐỎ** |

Lần chạy đầu, đột biến #1 viết `chan := q.Chan` — `chan` là **từ khoá Go**, nên nó đỏ vì
lỗi biên dịch chứ không phải vì bài kiểm cắn. Làm lại với tên biến khác (`/tmp/pha1.py`) và
nó đỏ vì đúng lý do, như bảng ghi. Ghi ra đây vì một phép đo đỏ-vì-lý-do-khác là một phép
đo hỏng, và nó trông y hệt phép đo tốt.

Hai chi tiết cố ý trong bài kiểm:

- `TestMatWebNoiRoKhaiQuyenKhongPhaiHangRao` **đối chiếu với chính `cmd/sagent/plugin.go`**
  thay vì chép câu đó hai lần — hai mặt phải nói cùng một điều bằng cùng một chữ, và nếu
  CLI đổi câu thì test đỏ ở cả hai đầu.
- Bài đó cắt bình luận HTML trước khi dò (`boCommentHTML`). Bản đầu **không cắn**: câu cần
  tìm nằm sẵn trong một dòng bình luận giải thích, nên test xanh dù trang không hiện chữ nào.
  Đúng cái bẫy mà `boComment` (JS) đã sinh ra để tránh, gặp lại ở dạng HTML.

### Cái tôi CỐ Ý KHÔNG làm

Không đụng `TestMoiHanhDongCuaNguoiDungDeuCoDuongVaoTuWeb`
(`internal/dash/ngangquyen_ui_test.go:21`). Bài đó chỉ liệt kê endpoint **hành động**, và
lý do viết ngay trong bài: "trang không gọi thì cùng lắm là thiếu thông tin, còn hành động
không gọi được là người dùng KHÔNG LÀM ĐƯỢC VIỆC". `/api/plugins` là chỉ-đọc, nên theo đúng
luật đó nó **không** phải một lỗ hổng của bài kiểm ấy. Nhét plugin vào cho có vẻ đầy đủ là
phá một ranh giới đã được suy nghĩ kỹ. Việc "khối plugin có được nạp không" nay do
`TestMatWebNapBangQuyenPluginLucMoTrang` canh, ở file riêng, với luật riêng.

---

## 2. Sự cố

**a) `go test ./...` có MỘT bài đỏ, và nó đỏ sẵn từ trước.**

```
--- FAIL: TestBoChayFlowCoDuCaHaiDuong (0.53s)
    phientrangthai_test.go:220: không mở được sổ trạng thái
    (C:\Users\Administrator\.ai-accounts\state.db): ... ở schema v10,
    bản sagent này chỉ biết tới v9 — nâng cấp sagent, hoặc khôi phục bản sao lưu cũ
```

Nằm ở `internal/api` — vùng tôi không đụng (tôi chỉ sửa `internal/dash/*`). Bài này đọc
**state.db THẬT của máy**, đang ở schema v10, trong khi mã trên `main` mới biết v9.

Không suy luận, đo thật: `git stash push -u`, chạy lại đúng bài đó trên cây làm việc sạch —
**vẫn đỏ, cùng thông điệp** — rồi `git stash pop`. Nên nó là nợ có sẵn của môi trường máy
này, không phải do thay đổi lần này. `go build ./...` và `go vet ./...` đều xanh sạch, và
`go test ./internal/dash/` xanh toàn bộ (46.6s).

**b) Đột biến #1 đỏ vì lỗi biên dịch, không phải vì test cắn.** Đã kể ở mục 1, làm lại rồi.

**c) Script python viết file làm đổi hết CRLF thành LF.** `newline="\n"` viết lại cả file,
nên hai file sửa chuyển từ CRLF sang LF trong cây làm việc. Git tự chuẩn hoá lúc commit
(`git diff --stat` ra đúng +239/−12, không phải "cả file"), nên diff sạch và không cần sửa.
Ghi lại vì lần sau dùng cách này trên file `.ps1` thì **hỏng thật** — `.gitattributes` bắt
`.ps1` phải là CRLF.

**d) Heredoc bash nuốt file Go.** Viết `plugin_ui_test.go` bằng `cat <<'GOEOF'` trong Bash
tool thì bash báo `unexpected EOF while looking for matching '`. Chuyển sang Write tool là
xong. Không mất thời gian, nhưng đây là lần thứ n cùng một loại bẫy trích dẫn.

**e) Không có phép đo bằng trình duyệt thật.** Xem mục 3.

---

## 3. Bước tiếp theo

1. **Mở trang bằng mắt.** Tôi kiểm được cú pháp JS (`node --check` trên toàn bộ khối
   `<script>` nội tuyến của `index.html` — rc = 0) và kiểm được JSON qua đúng ngăn xếp HTTP
   thật (đăng nhập → cookie → `GET /api/plugins`, với một manifest thật trên đĩa). Nhưng
   **chưa ai nhìn khối này vẽ ra trên màn hình**. Chưa đo: cột `[chặn]` rộng 124px có đủ cho
   nhãn "KHÔNG CHẶN ĐƯỢC" viết hoa ở font mono 10px không, và ba hàng ba màu đứng cạnh nhau
   có thật sự phân biệt được ở kích thước đó không.
2. **Mặt 3D (`trung-tam.html`) chưa có khối này.** Hai mặt nay nói khác nhau về plugin: 2D
   có bảng, 3D không có gì. Theo luật "hai surface là một sản phẩm" thì đây là một khoản nợ,
   không phải là xong.
3. **Đưa cột `[chặn]` lên chỗ dễ thấy hơn.** Hiện nó nằm trong ngăn kéo Tổng quan, phải bấm
   mới thấy. Chấp nhận được vì đây là bảng tra "trước khi bấm chạy", nhưng nếu một plugin
   có quyền `khong-chan-duoc` mà đang được một flow gọi thì con số đó xứng đáng lên thẳng
   màn hình chính.
4. **Nợ thật đứng sau cột này** (không phải việc của lượt này, nhưng là lý do cột tồn tại):
   `thu-muc-lam-viec` và `ghi-thu-muc-lam-viec` muốn lên `chan-that` thì phải có ACL riêng
   cho từng lượt chạy, Job Object, hoặc AppContainer. `mang` thì còn chưa ai đo. Bảng nay
   nói ra được ba chuyện đó rời nhau — nhưng nói ra không phải là sửa.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Tạo nhánh sạch, xác nhận `main..HEAD` = 0 | Opus 5 (1M) | thấp |
| Đọc `plugin_api.go`, `api/plugin.go`, `plugin/quyen.go`, `cmd/sagent/plugin.go` | Opus 5 (1M) | trung bình |
| Đọc bài kiểm mẫu `TestDTOKhongDepTrangThaiNaoThanhTrangThaiKhac` | Opus 5 (1M) | trung bình |
| Khảo ràng buộc: designsystem / uxui / domid / mat2d / dto_ten_truong / ngangquyen_ui | Opus 5 (1M) | cao |
| Quyết định KHÔNG sửa `ngangquyen_ui_test.go` (đọc kỹ luật của nó) | Opus 5 (1M) | trung bình |
| Tách `so_chua_chan` thành ba ngăn trong DTO | Opus 5 (1M) | trung bình |
| CSS: ba trạng thái ba màu, dùng lại khung `.nl` | Opus 5 (1M) | trung bình |
| HTML + JS khối "Plugin đã cài" | Opus 5 (1M) | cao |
| Viết 6 bài kiểm (401 dòng) | Opus 5 (1M) | cao |
| Dựng 11 đột biến, chạy, trả lại, ghi bằng chứng | Opus 5 (1M) | cao |
| Chứng minh bài đỏ ở `internal/api` là nợ có sẵn (stash → chạy → pop) | Opus 5 (1M) | thấp |
| `node --check` cú pháp JS nội tuyến | Opus 5 (1M) | thấp |
| Nghiệm thu ba lệnh riêng + viết báo cáo | Opus 5 (1M) | trung bình |

---

## 5. Nhận xét tự do

**Chỗ dễ làm sai nhất của việc này không phải là code, mà là chọn màu.**

Bảng năng lực provider đã có sẵn một bộ ba màu, và cách rẻ nhất là chép nguyên xi:
xanh / xám / vàng. Chép xong thì mọi bài kiểm đều xanh, khối trông đồng bộ với hàng xóm,
và không ai phàn nàn. Nhưng ô xám ở bảng năng lực nghĩa là "provider không có tính năng
đó" — trung tính, không phải việc của ai. Còn ô xám ở bảng quyền sẽ nghĩa là "host **không
chặn được** thứ này, đã đo, đây là lỗ hổng". Cùng một sắc xám, hai nội dung ngược nhau về
mức độ khẩn. Và đúng dòng ấy vừa bị bắt nói dối hôm nay.

Nên nó ăn `--error`. Đây là chỗ tôi lệch khỏi hàng xóm một cách có chủ ý, và tôi ghi lý do
thẳng vào CSS chứ không vào tài liệu — tài liệu thì không ai đọc lúc sửa màu.

**Chuyện thứ hai đáng ghi: tôi suýt sửa nhầm bài kiểm.**

Đề bài cảnh báo trước, và cảnh báo đó đúng chỗ. `TestMoiHanhDongCuaNguoiDungDeuCoDuongVaoTuWeb`
trông y như chỗ để nhét `/api/plugins` vào — nó là bài kiểm "mọi thứ đều có đường vào từ
web", và tôi vừa làm đúng một đường vào từ web. Phản xạ đầu tiên là thêm một dòng vào map
`hanhDong` cho khớp. Nhưng bài đó phân biệt **hành động** với **chỉ-đọc** một cách có chủ ý,
và lý do nằm ngay trong thân bài: thiếu thông tin thì khó chịu, không làm được việc thì
hỏng. Nhét một endpoint chỉ-đọc vào đó là làm loãng đúng cái ranh giới khiến bài kiểm ấy có
nghĩa — lần sau ai đó nhét thêm một cái nữa, rồi map đó thành "danh sách endpoint", và nó
không còn bắt được vụ nút Duyệt biến mất ngày 21/08.

Tôi để nguyên bài đó, và viết một bài mới với luật riêng cho việc riêng. Đắt hơn một dòng,
nhưng hai bài kiểm nhỏ mà mỗi bài nói đúng một điều thì bền hơn một bài to nói mơ hồ.

**Chuyện thứ ba: bài kiểm đầu tiên của tôi không cắn.**

`TestMatWebNoiRoKhaiQuyenKhongPhaiHangRao` bản đầu chỉ `strings.Contains(html, cau)`. Nó
xanh — nhưng nó sẽ **vẫn xanh** nếu tôi xoá câu đó khỏi trang và để lại đúng câu ấy trong
một dòng bình luận. Chính `ngangquyen_ui_test.go` đã kể lại y hệt cái bẫy này ở dạng JS, và
đã viết sẵn `boComment` để tránh. Tôi đọc đoạn đó, hiểu nó, rồi vẫn mắc lại ở dạng HTML mười
lăm phút sau. Đó là lý do tôi chạy 11 đột biến thay vì tin vào việc test đang xanh: **một
bài kiểm xanh không chứng minh gì cả cho tới khi bạn làm nó đỏ.**

**Điều tôi không chứng minh được:** rằng khối này *đọc được* trên màn hình thật. Tôi chứng
minh được nó biên dịch, nó lấy đúng dữ liệu, nó không bẹp trạng thái, và nó nói đúng câu
cần nói. Ba màu khác nhau trong mã nguồn thì chắc chắn; ba màu **phân biệt được bằng mắt**
ở font 10px trong một ngăn kéo hẹp thì chưa ai xác nhận. Đó là nợ, và nó nằm ở mục 3 chứ
không nằm ở đây dưới dạng một câu "đã xong".
