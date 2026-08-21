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
- **Cập nhật 21/08, lượt "hạn token"**: đóng **Đ4** (Cursor đo thật, Antigravity
  đổi sang `KhongLamDuoc`) và viết lại **nửa Cursor của C1** cho khớp mã. Lượt
  này lôi ra một ô nợ MỚI: **V3** — Cursor khai `Duoc(NLDanhTinh)` mà
  `Identity()` trả rỗng trên chính hồ sơ đang đăng nhập. Xem
  `docs/BAO-CAO-DONG-D4.md`.
- **Cập nhật 21/08, lượt "dọn sổ nợ"**: đóng **N1** (sửa bình luận trong mã) và
  **Đ3** (đối chiếu mã thật, không chỉ tiêu đề commit). Sổ này từng lệch với mã
  vì nó được sửa lúc **00:44** còn commit `bacc137` vào lúc **00:54** — sổ viết
  trước, mã đổi sau. Bài học đi kèm: **tiêu đề commit không phải bằng chứng**;
  bằng chứng là hàm trả về gì và bảng năng lực khai gì. Lần này đã mở
  `internal/provider/cursor.go` và hai chốt trong `internal/api/api.go` ra đọc
  trước khi tin tiêu đề. Xem `docs/BAO-CAO-DON-SO-NO.md`.
- **Cập nhật 21/08, lượt "đóng C1 phần Codex"**: đóng **C1** hoàn toàn — Codex
  đọc được kết quả có cấu trúc bằng `codex exec --json`, 6 lượt chạy thật. Quét
  lại: **51 dòng** trong **26 file** (trước: 55 / 23). Số dòng giảm 4 nhưng số
  **file tăng 3**, và đó không phải nghịch lý: hai file mới
  (`ketqua_codex.go`, `ketqua_codex_test.go`) cùng
  `ketqua_codex_e2e_test.go` đều mang chữ "CHƯA ĐO" **có chủ ý** — chúng ghi
  đúng cái đã đo được là *không* đo được (chi phí, số tool bị chặn quyền). Nợ
  được **nói ra** thì đếm vào sổ; đó là mục đích của sổ, không phải lỗi của nó.
  Xem `docs/BAO-CAO-DONG-C1-CODEX.md`.
- **Cập nhật 21/08, lượt "đóng C5 ngưỡng chạy quẩn"**: đóng **C5** — đo chuỗi lặp
  trên các bản ghi chạy bình thường thật trong `~/.ai-accounts/.nhat-ky/` (các phiên
  #174, #175, #176, #177), chuỗi lặp dài nhất = **1** (không có hai lần gọi tool
  nào giống hệt nhau liên tiếp). Ngưỡng `TranLapLienTiep = 10` an toàn tuyệt đối,
  cách xa 10 lần so với lượt chạy bình thường và cách gần 40 lần so với ca quẩn #21
  (399 lần). Xem `docs/DO-LUONG.md`, mục 21/08 C5.
- **Cập nhật 21/08, lượt "đóng C4 token và chi phí phiên CLI"**: đóng **C4** — đưa
  token và chi phí thật của phiên CLI vào `sessionDTO`, đọc từ nhật ký (`s.Log`)
  của phiên đã kết thúc qua `provider.DocKetQua()`. Phân biệt rạch ròi giữa số thật
  (Claude có `total_cost_usd`) và "chưa đo" (Codex/Cursor/Antigravity không có trường giá).
  DTO không để lộ số 0 giả cho chi phí chưa đo, bảo toàn lưới an toàn không đụng file web.
  Xem `docs/DO-LUONG.md`, mục 21/08 C4.
- **Cập nhật 21/08, lượt "đóng C3 chọn model từ dòng lệnh"**: đóng **C3** hoàn
  toàn — Antigravity (`--model`) và Codex (`-m`) đều CHẠY THẬT, hai kiểu bằng
  chứng khác hẳn nhau: Antigravity từ chối tên model bịa **ngay ở phía máy mình**
  (thoát 1, liệt kê 14 model), còn Codex **không chặn gì** ở tầng CLI — máy chủ
  mới trả HTTP 400. Nửa thứ hai (cờ có ĐỊNH TUYẾN không, chứ không chỉ được kiểm
  tra) đo bằng số: cùng một prompt qua ba model Antigravity cho ba mức
  `input_tokens` (13.747 / 15.764 / 11.174), và `-m` của Codex **ghi đè**
  `model = "gpt-5.6-sol"` trong `config.toml`. **Đếm lại cột `ChuaDo` sau lượt
  này: còn 5** — antigravity 2, claude 1, codex 1, cursor 1, grok 0; cột
  `NLChonModel` sạch trên cả năm provider. Xem `docs/BAO-CAO-DONG-C3.md`.
- **Cập nhật 21/08, lượt "giảm rủi ro C2 Grok"**: đã thiết lập bài canh định kỳ gọi
  CLI thật `TestE2EGrokDocDuocOutputThat` (`SAGENT_E2E_GROK=1`) trong
  `internal/provider/ketqua_grok_e2e_test.go`. Phép đo thật hôm nay xác nhận CLI Grok
  vẫn in ra cấu trúc NDJSON nhưng trả về lỗi HTTP 410 ("Live search is deprecated")
  từ xAI. Bộ đọc `docKetQuaGrok` bóc tách được dòng NDJSON (`docDuoc=true`), còn tình
  trạng lỗi 410 từ upstream được ghi nhận trung thực. Xem `docs/DO-LUONG.md`, mục 21/08 C2.



## Vì sao cần sổ này khi đã có `sagent nang-luc --chua-do`

Lệnh đó (`cmd/sagent/nangluc.go:16`) in ra **bảng khai năng lực** — 14 ô `ChuaDo`
của 4 adapter lúc lập sổ. **Đếm lại 21/08 sau lượt đóng Đ4: còn 9** — claude 1,
codex 3, cursor 2, antigravity 3, grok 0. (Con số này rơi nhờ nhiều lượt cộng
lại, không riêng lượt nào; ghi ra để lần quét sau đừng so với số cũ.) Sổ này thêm
ba thứ lệnh đó không có:

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

## ~~Đ1~~ ✅ ĐÃ ĐÓNG 21/08 — Codex cờ tự-duyệt-quyền: ĐÃ CHẠY THẬT

> **Kết quả**: nấc hẹp nhất đủ dùng là `--approve-for-me`, KHÔNG phải
> `--dangerously-bypass-approvals-and-sandbox`. `--sandbox workspace-write` một
> mình không đủ, và `codex exec` không có `--ask-for-approval`. Đã đổi mã.
> Xem `docs/DO-LUONG.md`, mục 21/08.

<details><summary>Nội dung ô nợ khi còn mở</summary>


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

</details>

## ~~Đ2~~ ✅ ĐÃ ĐÓNG 21/08 — Codex cờ thư mục: ĐÃ CHẠY THẬT

> **Kết quả**: `-C, --cd <DIR>` có tác dụng thật — chạy trong thư mục tạm rồi
> bảo agent đọc một file chỉ có ở đó, nó đọc đúng.

<details><summary>Nội dung ô nợ khi còn mở</summary>


- **Ở đâu**: `internal/provider/codex.go:221-222`, `internal/provider/codex.go:246`.
- **Nó nói gì**: `codex --help` có `-C, --cd <DIR>`. Nguyên văn: *"CHƯA chạy thật"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: agent chạy **sai thư mục** mà không ai biết. Con số
  để so là của Antigravity, đã đo thật (`internal/provider/antigravity.go:132-133`):
  **không có cờ thì 1/3 lượt đúng**, hai lượt còn lại báo *"chưa có repository
  nào được mở"*; có cờ thì 4/4. Tức nếu `-C` không có tác dụng như `--add-dir`,
  ta mất **2/3 số lượt** — và mất theo kiểu agent trả lời trôi chảy về một repo
  khác, chứ không phải theo kiểu báo lỗi.
- Cùng lý do với Đ1: khai `LamDuoc` nên không có chốt nào chặn.

</details>

## ~~Đ3~~ ✅ ĐÃ ĐÓNG 21/08 — Cursor cờ tự-duyệt-quyền: ĐÃ CHẠY THẬT

> **Kết quả**: `--trust` MỘT MÌNH đã đủ để agent ghi file trong workspace —
> không cần `--force`/`--yolo`. Giữ nấc hẹp nhất, cùng lối chọn với `--approve-for-me`
> của Codex ở Đ1. Đã đổi mã. Xem `docs/DO-LUONG.md`, mục 21/08 *"Đ3: Cursor"*.
>
> **Bằng chứng đã đối chiếu với mã, không chỉ với tiêu đề commit** (kiểm ngày
> 21/08, sau khi thấy sổ sửa lúc 00:44 còn commit `bacc137` lúc 00:54):
>
> - `internal/provider/cursor.go:150` — `ArgsTuDuyetQuyen()` trả
>   `([]string{"--trust"}, true)`, tức `daDo == true`. Không còn `(nil, false)`.
> - `internal/provider/cursor.go:188-189` — bảng năng lực khai
>   `Duoc(NLTuDuyetQuyen, "--trust (đo 21/08, CHẠY THẬT trên 2026.08.11) …")`.
>   Bảng khai và hàm thật **khớp nhau**, nên `internal/api/tuduyetquyen_test.go:61-72`
>   không có gì để bắt.
> - Hai chốt ở `internal/api/api.go:1002-1005` và `internal/api/api.go:1320-1322`
>   giờ **cho Cursor qua một cách hợp lệ**: chúng chặn theo `daDo`, và `daDo` đã
>   là `true` vì có phép đo thật đứng sau, không phải vì ai đó khai bừa.
> - `git show bacc137 --stat`: 4 file, +234/−16 — `cursor.go`, `ketqua_cursor.go`,
>   `ketqua_cursor_test.go`, `DO-LUONG.md`. Có mã, có bài kiểm, có phép đo.
>
> **Sai lầm nền của ô này** (đáng giữ lại): bảng năng lực cũ ghi *"máy này không
> cài cursor-agent"*, nhưng `Get-Command cursor-agent` cho ra đường dẫn thật, bản
> 2026.08.11. Ghi chú đúng vào lúc viết, sai vào lúc đọc — **một dòng CHƯA ĐO
> không tự hết hạn**.
>
> **Cùng commit đó còn đóng thật ba ô nữa của Cursor**, nhưng sổ này chưa viết
> lại chúng vì lượt làm việc được giao đúng ô Đ3: **C1** (`DocKetQua` →
> `docKetQuaCursor`, `cursor.go:173` + `Duoc(NLKetQuaCoCauTruc)` ở `:193`),
> **C3** (`ModelArgs` → `--model`, `cursor.go:374` + `Duoc(NLChonModel)`) — ô C3
> nay **đã đóng hoàn toàn**, gồm cả Antigravity và Codex; xem mục C3,
> **C6** (`ArgsThuMuc` → `Khong(NLThuMuc)`, `cursor.go:158` + `:190`).
> C1 và C3 nay đã viết lại theo khuôn đóng; chỉ còn **C6** mang nhãn
> "⚠ ĐÃ LẠC HẬU" — đọc nhãn đó trước khi đọc gạch đầu dòng bên dưới nó.

<details><summary>Nội dung ô nợ khi còn mở</summary>


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

</details>

## ~~Đ4~~ ✅ ĐÃ ĐÓNG 21/08 — hạn token: Cursor ĐÃ ĐO THẬT, Antigravity là KẾT LUẬN

> **Hai nửa của ô này đóng bằng hai cách khác nhau, và đó là điểm chính.**
>
> ### Nửa Cursor — ĐO THẬT, đọc được hạn
>
> Rào chắn cũ (*"auth.json CÓ THỂ mang dấu thời gian, chưa biết trường nào"*)
> đứng trên một chỗ tìm sai. `~/.cursor/auth.json` **không hề tồn tại**; file
> thật nằm ở `%APPDATA%\Cursor\auth.json` — đúng chỗ mà `EnvVar()` và
> `PrivateFiles()` đã chỉ từ đầu (`internal/provider/cursor.go:37,66`). Ô này mở
> ba tuần vì tìm nhầm thư mục, không phải vì thiếu phép đo.
>
> **Hình dạng file, đo chứ không đoán** (hồ sơ đăng nhập thật, CLI
> `2026.08.11-e8db854`): auth.json có **ĐÚNG HAI khoá** — `accessToken` và
> `refreshToken`, cả hai là JWT `alg:HS256`. **Không có trường dấu-thời-gian nào
> ở tầng ngoài.** Nên lời dặn *"đừng đoán tên trường"* hoá ra còn đúng hơn cả ý
> định ban đầu: **không có trường nào để mà đoán**. Mốc nằm trong payload JWT, y
> hệt Codex.
>
> Claim đọc được — `accessToken` và `refreshToken` giống hệt nhau từng claim:
>
> | claim | giá trị | nghĩa |
> |---|---|---|
> | `iss` | `https://authentication.cursor.sh` | |
> | `aud` | `https://cursor.com` | |
> | `scope` | `openid profile email offline_access` | |
> | `type` | `session` | |
> | `time` | `1787022539` | 2026-08-18T03:08:59Z — lúc đăng nhập |
> | `exp` | `1792206539` | 2026-10-17T03:08:59Z — **đúng 60 ngày** sau |
>
> **Chạy thật trên máy này sau khi sửa mã**: `HasToken=true`,
> `TokenExpiry ok=true`, `exp=2026-10-17T03:08:59Z`, còn `1364h` (≈ 56,8 ngày).
> Trước bản vá, cùng hồ sơ đó trả `ok=false` — tức **không phân biệt được với
> "chưa đăng nhập"**.
>
> **Đọc REFRESH token, không phải access token** — ghim bằng
> `TestCursorDocRefreshChuKhongPhaiAccess`. Hôm nay hai mốc BẰNG NHAU nên phép đo
> thật *không* phân biệt được hai lựa chọn; bài kiểm phân biệt hộ. Lý do là bài
> học đã trả giá ở `claude.go`: trả hạn access token làm cổng kiểm **chặn oan**
> lượt chạy #39 trong khi tài khoản vẫn dùng được.
>
> **Còn chưa đo, nói thẳng**: token có **xoay vòng** khi refresh hay không — mà
> đó mới là thứ cảnh báo hạm đội thật sự sợ. Bằng chứng gián tiếp là CLI **chưa
> hề ghi lại file này**: `mtime` của auth.json vẫn là `2026-08-18 10:08:58 (+07)`
> — đúng giây đăng nhập — trong khi `.cursor/cli-config.json` bị ghi lại lúc
> `2026-08-21 00:46` bởi chính các lượt chạy thật của ô Đ3. Qua ba ngày dùng,
> `cursor-agent` không đụng vào file token.
>
> ### Nửa Antigravity — KHÔNG ĐO, và đó là câu trả lời cuối
>
> Đổi `Chua(NLHanToken)` → **`Khong(NLHanToken)`**
> (`internal/provider/antigravity.go`). Lý do giữ nguyên lý do cũ, chỉ đổi tư
> cách của nó: token nằm trong Windows Credential Manager dưới khoá
> `gemini:antigravity`, `CredRead` trả về **cả blob** chứ không có cách hỏi riêng
> mốc hết hạn. Mở chính thứ cần bảo vệ để đổi lấy **một dấu thời gian** là đánh
> đổi tồi. Đây là một **kết luận**, không phải khoảng trống — nên nó thôi nằm
> trong sổ nợ.
>
> Khác Grok một chỗ đáng nói ra: ở `grok.go:250` hạn **KHÔNG TỒN TẠI** (API key,
> không OAuth); ở đây hạn **CÓ**, nhưng nằm sau một cánh cửa ta cố ý không mở.
> Cùng trạng thái `KhongLamDuoc`, khác lý do — và bằng chứng viết rõ cả hai.
>
> ### Cảnh báo hạm đội đổi ra sao
>
> Chốt ở `internal/api/api.go` chỉ chạy khi `TokenExpiry` trả `ok=true`, nên:
>
> - **Cursor**: nay **có** chạy. Cửa sổ 60 ngày dài hơn mọi lượt chạy nên nhánh
>   *"còn dưới 2 tiếng"* gần như sẽ không kêu — **đó là đúng, không phải hỏng**.
>   Cái đổi thật là nó thôi im lặng vì KHÔNG BIẾT và bắt đầu im lặng vì ĐÃ BIẾT
>   là còn hạn. `ProfileList` (`api.go:264`) cũng điền được `HanToi`/`HetHan`,
>   nên `sagent ds` nói được "hết hạn lúc mấy giờ" thay vì để trống.
> - **Antigravity**: **vẫn không** chạy, và điều đó **không đổi** — đây là hệ quả
>   phải nói ra, không phải thứ bản vá này che đi. Khác biệt duy nhất: bảng năng
>   lực nay khai thẳng *"không đọc được hạn"* thay vì để trống, nên sự im lặng
>   đọc được đúng nghĩa. Bằng chứng trong mã ghi hẳn câu này.
>
> ### Bài kiểm giữ ô
>
> `internal/provider/token_cursor_test.go` — 6 bài + 8 ca con. Ghim hình dạng
> thật của file vào bình luận, ghim lựa chọn refresh-chứ-không-access, và ghim
> **mọi ngõ hỏng phải trả `false`** (không JSON, không JWT, payload không
> base64, thiếu claim `exp`, `exp=0`, không có file). Nửa còn lại của *"cảnh báo
> sai giờ còn tệ hơn không cảnh báo"* nằm ở đó: `time.Time{}` trả kèm `ok=true`
> sẽ đọc thành "hết hạn từ năm 1". JWT trong bài kiểm là **giả** — chữ ký không
> được kiểm nên giả là đủ, và một token thật trong mã nguồn là một rò rỉ.

<details><summary>Nội dung ô nợ khi còn mở</summary>

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

</details>

## ~~Đ5~~ ✅ ĐÃ ĐÓNG 21/08 — Cuộc đua N-clone: ĐÃ CHẠY THẬT

> **Kết quả**: hai clone cùng token, cùng bị ép hết hạn, bật cách nhau **16ms**
> → **đúng MỘT bản thắng** (`86fb2200` → `d22e079d`), không có cửa sổ ân hạn.
> Bản thua chết sau **186ms** với *"Failed to authenticate: OAuth session
> expired and could not be refreshed"* và **tự ghi đè file token của chính nó
> thành rỗng** (`expiresAt: 0`).
>
> **Hai lỗi thật tìm được nhờ chạy**: (1) bản thua ghi file **SAU** bản thắng,
> nên `SyncBackTokens` chọn-theo-mtime sẽ chép **file rỗng đè lên hồ sơ gốc** —
> dựng lại được bằng test, và kết cục là công cụ đòi `/login` trong khi token
> sống vẫn nằm trong bản thắng; (2) `PhanLoaiChet` trả rỗng cho bản ghi này
> (`api_error_status` là `null`), nên cái chết **có lý do rõ nhất** lại hiện ra
> là *"chết, chưa rõ vì sao"*. Đã sửa cả hai.
> Xem `docs/DO-LUONG.md`, mục 21/08, và `internal/profile/duarefresh_test.go`.

<details><summary>Nội dung ô nợ khi còn mở</summary>


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

</details>

---

# 🟠 MỨC CAM — đoán sai là mất tiền hoặc mất kết luận

## ~~C1~~ ✅ ĐÃ ĐÓNG HOÀN TOÀN 21/08 — Codex và Cursor ĐỌC ĐƯỢC kết quả có cấu trúc

> Hai nửa đóng ở **hai lượt chạy khác nhau trong cùng một ngày**, nên bằng chứng
> dưới đây gộp từ cả hai. Không nửa nào được đóng bằng suy luận.
>
> ### Nửa Cursor — đóng trong `bacc137`, đã MỞ MÃ RA ĐỌC để xác nhận
>
> (đúng bài học của Đ3: sổ sửa lúc **00:44** còn commit `bacc137` lúc **00:54** —
> sổ viết trước, mã đổi sau, nên **tiêu đề commit không phải bằng chứng**)
>
> - `internal/provider/cursor.go:272` — `DocKetQua` trả thẳng
>   `docKetQuaCursor(raw)`. **Không còn** `(KetQua{}, false)`.
> - `internal/provider/cursor.go:292` — bảng khai
>   `Duoc(NLKetQuaCoCauTruc, "--output-format stream-json (đo 21/08, CHẠY THẬT)…")`.
>   Dòng cuối `{"type":"result"}` mang `is_error`, `subtype`, `result`,
>   `request_id` và `usage` — `inputTokens`/`outputTokens` **camelCase, KHÁC Claude**.
> - Bộ đọc thật và bài kiểm của nó: `internal/provider/ketqua_cursor.go`,
>   `internal/provider/ketqua_cursor_test.go`.
>
> *(Số dòng trên là của bản mã sau lượt đóng **Đ4**, muộn hơn `bacc137` — nội dung
> không đổi, chỉ trôi dòng vì `cursor.go` dài thêm.)*
>
> **Vẫn còn thiếu, để không đọc thành đóng trọn**: bản ghi Cursor **không có**
> `total_cost_usd`, nên chi phí của Cursor vẫn chưa đo được — xem **C4**.
>
> ### Nửa Codex — đóng 21/08 bằng 6 lượt `codex exec` CHẠY THẬT
>
> **Codex** đóng ngày 21/08 bằng **6 lượt `codex exec` chạy thật** trên
> codex-cli 0.147.0, tài khoản thật. Cờ là `--json` (*"Print events to stdout as
> JSONL"*, có thật trong `codex exec --help`). Mã đã đổi:
>
> - `internal/provider/ketqua_codex.go` (mới) — bộ đọc, kèm bốn cái bẫy đo được.
> - `internal/provider/codex.go` — `DocKetQua` gọi bộ đọc thật; `HeadlessArgs`
>   thêm `--json`; bảng khai `Chua(NLKetQuaCoCauTruc)` → `Duoc(...)`.
> - 11 bài kiểm trên **bản ghi thật** + 1 bài canh gọi CLI thật
>   (`ketqua_codex_e2e_test.go`, `SAGENT_E2E_CODEX=1`).
>
> **Bằng chứng đầu-cuối** (args dựng bằng chính adapter, đọc lại bằng chính adapter):
>
> ```
> ĐỌC ĐƯỢC: TraLoi="ALPHA" CoLoi=false TokenVao=17627 TokenRa=6 Hong=""
> PhanLoaiChet: "done" ""
> ```
>
> Trước lượt đo, dòng cuối là `""` — phiên ở lại `lost`, bốn mặt điều khiển in
> *"chết, chưa rõ vì sao"* cho một lượt chạy **thành công**.
>
> **HAI THỨ VẪN CHƯA ĐO ĐƯỢC, và chúng đã được đo để biết là không đo được:**
>
> 1. **`ChiPhiUSD` = 0** — bản ghi Codex không có trường giá nào, y hệt Cursor.
>    Số 0 đó vẫn phải hiện kèm chữ "chưa đo" (xem **C4**), không được đọc thành
>    "miễn phí". Đừng nhân token với đơn giá.
> 2. **`TuChoiSo` = 0, VĨNH VIỄN** — chạy không có `--approve-for-me` rồi bắt
>    agent ghi file: bản ghi ra `turn.completed` **bình thường**, không một
>    trường nào nói tới quyền. Lời từ chối chỉ nằm trong **văn xuôi do model tự
>    viết** ("môi trường hiện tại bị khóa **chỉ đọc**"). Đọc được nó thì phải dò
>    chuỗi — đúng thứ `trangthai.go` cấm. **Hệ quả: `ChetChanQuyen` KHÔNG BAO GIỜ
>    kết luận được cho Codex.** Bài `TestCodexKhongBiaSoToolBiChanQuyen` giữ chỗ.
>
> Chưa đo nốt: `SoLuotTu`, `HanMucDenLai`, `KetCuc` (không có trường tương ứng);
> và `cached_input_tokens` là phần **con** hay phần **thêm** của `input_tokens`
> — nên không cộng, không trừ.
>
> **Nợ mới sinh ra từ ô này**: bộ đọc Codex nay có đúng điểm yếu mà **C2** mô tả
> cho Grok, chỉ nhẹ hơn một bậc — `--json` là cờ có thật nên đây là hợp đồng, còn
> Grok chỉ là quan sát. Bài canh định kỳ ở trên là cách biết vào **ngày** Codex
> đổi định dạng, không phải ba tuần sau.
>
> Xem `docs/DO-LUONG.md`, mục *21/08 — Codex đọc được kết quả có cấu trúc*.

<details><summary>Nội dung ô nợ khi còn mở</summary>

- **Ở đâu**: `internal/provider/codex.go:226,248` — `DocKetQua` trả
  `(KetQua{}, false)`. *(Hai dòng `cursor.go:148,167` từng đứng ở đây đã hết hiệu
  lực — xem ghi chú đóng bên trên.)*
- **Nó nói gì**: Codex — *"CHƯA ĐO cách đọc dữ liệu có cấu trúc; phiên Codex chết
  vì lý do gì thì sổ để nguyên `lost` chứ không đoán"*.
- **HẬU QUẢ NẾU ĐOÁN SAI**: provider này **mù toàn tập** ở mọi mặt hiển thị.
  Không `is_error`, không `usage`, không `total_cost_usd`, không
  `permission_denials`. Cụ thể mất những gì:
  - Mọi phiên Codex về `lost` — `internal/provider/trangthai_test.go:128`
    canh đúng điều này và nó **đúng**, nhưng đó là cái đúng của người thành thật,
    không phải của người biết việc.
  - `PhanLoaiChet` không phân biệt được hết-hạn-mức với bị-chặn-quyền với
    lỗi-API. Người vận hành thấy *"chết, chưa rõ vì sao"* rồi phải tự mở log.
  - Không đếm được token/chi phí → không so được provider nào rẻ hơn cho việc gì.
    (Cursor nay đếm được token nhưng vẫn chưa có chi phí — xem C4.)
  - **Lá chắn chống chạy quẩn không hoạt động**: `internal/provider/quan.go` cần
    `DemDuocTool`, mà `DemDuocTool` đến từ `DocKetQua`. Một agent Codex chạy quẩn
    399 lần sẽ đốt sạch hạn mức mà không ai chặn.
- **Đoán sai theo chiều ngược**: nếu ai đó "đoán" một bộ đọc cho Codex rồi nó đọc
  nhầm, hậu quả **nặng hơn** không đọc. `internal/provider/quan_test.go:212-214`
  ghim đúng chỗ này: một kết luận "chạy quẩn" đến từ bản ghi ta chưa đọc được là
  kết luận không được phép có — vì nó sẽ giết một lượt chạy lành.

</details>

## C2. Bộ đọc kết quả của Grok dựa trên quan sát, không phải hợp đồng

> ### Cập nhật 21/08: Đã có bài canh định kỳ gọi CLI thật, xác nhận hiện trạng lỗi HTTP 410
>
> - **Biện pháp giảm rủi ro đã có**: Đã xây dựng bài canh định kỳ gọi CLI thật `TestE2EGrokDocDuocOutputThat`
>   trong `internal/provider/ketqua_grok_e2e_test.go`, kích hoạt bằng `$env:SAGENT_E2E_GROK="1"` trên Windows PowerShell
>   hoặc `SAGENT_E2E_GROK=1 go test ./internal/provider/` (theo đúng khuôn mẫu `ketqua_codex_e2e_test.go`).
> - **Kết quả đo CLI thật ngày 21/08/2026**:
>   - Lệnh chạy dựng bằng chính adapter: `grok --max-tool-rounds 60 -p "Tra loi dung mot tu: ALPHA" -m grok-4.5 -d <tempdir>`.
>   - Output nguyên văn từ CLI (@vibe-kit/grok-cli 1.0.1):
>     ```json
>     {"role":"user","content":"Tra loi dung mot tu: ALPHA"}
>     {"role":"assistant","content":"Sorry, I encountered an error: Grok API error: 410 Live search is deprecated. Please switch to the Agent Tools API: https://***.***.x.ai/***/***/***/***"}
>     ```
>   - **Phân tích số đo**: Output vẫn mang cấu trúc 2 dòng NDJSON, `docKetQuaGrok` bóc tách được `{"role":"assistant", ...}` và trả về `docDuoc = true`. Tuy nhiên, nội dung câu trả lời là lỗi HTTP 410 do tính năng Live Search cũ của Grok CLI đã bị xAI khai tử.
>   - **Tình trạng thật của `NLKetQuaCoCauTruc`**: Bộ đọc NDJSON hoạt động đúng trên cấu trúc quan sát được (`docDuoc=true`), nhưng CLI Grok đang hỏng do lỗi 410 từ upstream xAI nên không thể hoàn thành nhiệm vụ agent. Bảng năng lực `internal/provider/grok.go:246` giữ đúng sự thật đo được, và các luồng làm việc thực tế (`.sagent/flows.toml:862`) đã chuyển bước soi sang node `model` gọi API trực tiếp.
>   - Xem chi tiết tại `docs/DO-LUONG.md`, mục *21/08 — C2: Đo CLI Grok thật bằng bài canh định kỳ*.

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


## ~~C3~~ ✅ ĐÃ ĐÓNG HOÀN TOÀN 21/08 — cả năm provider chọn được model từ dòng lệnh

> **Kết quả**: `ModelArgs` của **Antigravity** trả `--model <model>`
> (`internal/provider/antigravity.go:177`), của **Codex** trả `-m <model>`
> (`internal/provider/codex.go:277`); cả hai khai `Duoc(NLChonModel)`
> (`antigravity.go:188`, `codex.go:292`). Cursor đã đóng từ `bacc137`. Cột
> `NLChonModel` nay **không còn ô `ChuaDo` nào** trên cả năm provider.
>
> ### Vì sao ô này đáng đóng bằng tiền, không phải bằng cảm giác gọn gàng
>
> Sổ này ghi ô C3 là ô **tốn tiền trực tiếp**, và con số ở `internal/api/model_test.go:9-12`
> là chỗ nó thành tiền thật: **lượt chạy #34 tốn 9,40 USD, riêng bước `code-go`
> 8,18 USD** — vì MỌI bước đều chạy model mạnh nhất, kể cả bước chỉ viết tài liệu.
> Trước hôm nay, khai `model = "..."` cho một bước Antigravity/Codex chỉ đổi lấy
> một dòng cảnh báo; bước vẫn chạy model mặc định. Với Codex, "mặc định" có tên
> cụ thể: `~/.codex/config.toml` khai `model = "gpt-5.6-sol"` và nói thẳng lý do
> — *"MODEL MẶC ĐỊNH = mạnh nhất, có lý do"*. Nên cái mất không phải giả thuyết.
>
> ### Phép đo: hai provider, HAI KIỂU BẰNG CHỨNG KHÁC HẲN NHAU
>
> Tiền lệ Cursor nói bằng chứng mạnh nhất là **CLI từ chối một tên model bịa**.
> Đúng với Antigravity, **sai với Codex** — và chỗ khác nhau đó mới là phần đáng đọc.
>
> **Antigravity (bản 1.1.16)** — chặn ở phía máy mình, trước khi tốn token:
>
> ```
> $ agy -p "..." --model khong-ton-tai-9x
> Error: invalid model selection (--model "khong-ton-tai-9x" --effort ""):
>        model khong-ton-tai-9x is not recognized as a known model or custom model in settings
> Available models: [liệt kê đủ 14 model]        → thoát mã 1
> ```
>
> **Codex (bản 0.147.0)** — KHÔNG chặn gì cả. CLI nhận cờ, in `model:
> khong-ton-tai-9x` ở đầu bản ghi, chỉ càu nhàu *"Model metadata … not found.
> Defaulting to fallback metadata"*, rồi **máy chủ** mới chặn: HTTP 400 *"The
> 'khong-ton-tai-9x' model is not supported when using Codex with a ChatGPT
> account"*. Đó vẫn là bằng chứng — thậm chí là bằng chứng đi xa hơn: giá trị
> **đã đi hết đường xuống thân yêu cầu API**. Nhưng nếu lượt đo chỉ tìm đúng
> khuôn "CLI từ chối" thì nó sẽ kết luận nhầm là Codex nuốt cờ.
>
> ### Nửa thứ hai, và vì sao KHÔNG được bỏ
>
> Từ chối tên sai chỉ chứng minh cờ được đọc **để kiểm tra**. Nó không chứng minh
> cờ được dùng **để định tuyến**. Hai phép đo riêng cho hai provider:
>
> - **Antigravity — cùng một prompt, ba model, ba mức `input_tokens`**:
>   `gemini-3.7-flash-low` → **13.747**, `claude-opus-4-6-thinking` → **15.764**,
>   `gpt-oss-120b-medium` → **11.174**. Cùng prompt, cùng repo, cùng CLI; biến duy
>   nhất là `--model`. Ba bộ tách từ khác nhau đọc cùng một đầu vào ⇒ cờ đổi model thật.
> - **Codex — cờ GHI ĐÈ hồ sơ**: `config.toml` khai `gpt-5.6-sol`. Chạy không cờ,
>   đầu bản ghi in `model: gpt-5.6-sol`; chạy `-m gpt-5.4-mini`, đầu bản ghi đổi
>   thành `model: gpt-5.4-mini` và lượt chạy xong thật (12.291 token). Đây đúng là
>   chiều mà `grok.go:236-238` đã bác một lần — *"provider tự đọc model từ hồ sơ"*
>   không được mặc định tin — nên phải đo riêng. Ở đây kết quả là **cờ thắng hồ sơ**.
>
> ### Một phép đo ĐÃ BẮT ĐẦU SAI, giữ lại vì người sau sẽ thử đúng cách đó
>
> Cách hiển nhiên để kiểm "có đúng model không" là hỏi thẳng agent. Đã thử: chạy
> `--model claude-opus-4-6-thinking` rồi hỏi *"bạn do Google hay Anthropic tạo ra?"*
> — nó trả **"Google"**. Nếu tin câu đó thì kết luận sẽ là *cờ bị nuốt, khai
> `KhongLamDuoc`* — **sai hoàn toàn**, và sai theo hướng đắt hơn hiện trạng.
> **Tự khai danh tính không phải phép đo**: lời nhắc hệ thống của Antigravity đè
> lên câu trả lời. Số token thì không biết nói dối. Cùng họ với bài học của Đ3:
> *tiêu đề commit không phải bằng chứng*.
>
> ### Cái bẫy THỨ TỰ CỜ, đã đo chứ không suy
>
> `argsChoBuoc` (`internal/api/api.go:1397`) **chèn `ModelArgs` VÀO TRƯỚC**
> `HeadlessArgs`. Với Codex, `-m` lại là cờ của **lệnh con** `exec`, nên dòng thật
> là `codex -m <model> exec --json <prompt>` — cờ đứng **trước** lệnh con. Suy từ
> `--help` thì đây là chỗ hỏng. Đã chạy đúng dạng đó và Codex nhận. Antigravity
> cũng đã chạy đúng dạng chèn-trước (`agy --model <m> --output-format stream-json
> -p <prompt>` → 13.742 token, đúng chữ ký của `gemini-3.7-flash-low`), chứ không
> phải chỉ dạng cờ-đứng-sau lúc thử tay. Thứ tự này nay có bài kiểm khoá lại:
> `TestCodexDatCoModelTruocLenhCon`.
>
> ### Bài kiểm phải sửa, và vì sao đó không phải "sửa test cho xanh"
>
> `TestProviderChuaDoModelThiPhaiCanhBao` dùng **provider thật `antigravity`** làm
> vật thử cho nhánh cảnh báo. Đóng ô này xong, bài kiểm gãy — **không phải vì
> nhánh cảnh báo hỏng, mà vì nó hết vật thử**: không còn provider thật nào trả
> `nil`. Nhánh cảnh báo vẫn phải sống, vì nó là thứ duy nhất đứng giữa người dùng
> và một hoá đơn chạy model mặc định, cho provider **tiếp theo** được thêm vào mà
> chưa ai đo. Nên vật thử đổi sang adapter GIẢ (`giaAdapter.ModelArgs` →
> `nil`), và thêm hai bài kiểm khẳng định chiều ngược lại: hai provider vừa đo
> phải truyền cờ xuống thật **và không được cảnh báo nữa** (cảnh báo thừa cũng là
> một kiểu sai — nó đẩy người đọc đi tìm một vấn đề không tồn tại).
>
> ### Nợ MỚI lượt này lôi ra (chưa trả, chưa cắn được ai hôm nay)
>
> Trong `argsChoBuoc`, nhánh *"provider KHÔNG có rào quyền nào"* **gán đè** lên
> biến `canhBao`. Một provider vừa không-có-rào-quyền vừa chưa-đo-model sẽ **mất
> câu cảnh báo về model**. Hôm nay không provider nào rơi vào cả hai ô cùng lúc
> nên nó chưa cắn được ai — ghi lại đúng vì đó là lý do duy nhất nó chưa cắn.

<details><summary>Nội dung ô nợ khi còn mở</summary>


> ⚠ **ĐÃ LẠC HẬU MỘT PHẦN (đối chiếu mã 21/08)**: **Cursor** đã đóng trong commit
> `bacc137` — `internal/provider/cursor.go:169` trả `--model <model>`, bảng khai
> `Duoc(NLChonModel)` ở `:184`, bằng chứng là CLI TỪ CHỐI tên model sai và liệt kê
> model hợp lệ. Còn lại **hai** provider (Antigravity, Codex), không phải ba.

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

</details>

## ~~C4~~ ✅ ĐÃ ĐÓNG 21/08 — Token và chi phí phiên CLI: Claude ĐỌC ĐƯỢC SỐ THẬT, Codex/Cursor LÀ KẾT LUẬN "CHƯA ĐO"

> **Kết quả**: DTO của phiên (`sessionDTO`) nay đọc trực tiếp token và chi phí từ
> nhật ký (`s.Log`) của các phiên đã kết thúc thông qua `provider.Get(s.Provider).DocKetQua()`,
> thay vì để trống hoặc để mặt web phải tự đoán.
>
> ### Bằng chứng số thật lấy được từ DTO:
>
> - **Claude**: Đọc được **cả token lẫn chi phí thật** từ dòng kết quả `{"type":"result", ...}`
>   trong nhật ký (trường `usage.input_tokens`/`usage.output_tokens` và `total_cost_usd`).
>   Ví dụ các phiên chạy thật đo được: phiên #174 tiêu 70 tools, phiên #177 tiêu 0,5235 USD / 6.102 tokens.
>   DTO điền số thật vào `tokens` và `costUsd`, cờ `ChiPhiDaDo = true`.
> - **Cursor**: Đọc được **token thật** (`inputTokens`, `outputTokens` dạng camelCase từ `usage`),
>   nhưng chi phí USD **chưa đo được** (`ChiPhiDaDo = false`) vì bản ghi stream-json của Cursor
>   **hoàn toàn không có trường `total_cost_usd`**.
> - **Codex**: Đọc được **token thật** (`input_tokens`, `output_tokens` từ `turn.completed`),
>   nhưng chi phí USD **chưa đo được** (`ChiPhiDaDo = false`) vì các sự kiện JSONL của Codex
>   **không có bất kỳ trường giá nào**.
> - **Antigravity**: Đọc được **token thật** từ `result.usage`, không có trường giá (`ChiPhiDaDo = false`).
> - **Grok**: Bản ghi không có cấu trúc usage/cost, cả token và chi phí đều chưa đo.
>
> ### Ba nguyên tắc cốt lõi đã giữ vững khi đóng ô này:
>
> 1. **Codex và Cursor "chưa đo" chi phí là KẾT LUẬN, không phải lỗi**: Bản ghi của nhà cung
>    cấp không xuất trường giá. Ta không tự ý nhân token với một đơn giá giả định (vì đơn giá thay
>    đổi theo từng model, từng gói cước và bộ nhớ đệm). Việc ghi nhận "chưa đo" cho chi phí là một
>    kết luận trung thực, giống hệt bài học ở Đ4 (Antigravity) và C1 (Codex).
> 2. **DTO KHÔNG lộ số 0 giả cho chi phí chưa đo**: Với các provider chưa đo được chi phí (hoặc
>    lượt chạy chưa có số giá), DTO không được trả về `costUsd = 0` (tránh để giao diện hay người
>    dùng hiểu lầm là phiên "miễn phí"). Ô chưa đo phải được phân biệt rõ với ô có giá trị 0 thật.
> 3. **Chỉ đọc nhật ký của phiên ĐÃ KẾT THÚC**: Tránh đọc file log đang ghi dở của các phiên đang
>    sống (`running`/`pending`), ngăn chặn xung đột đọc-ghi và dữ liệu chắp vá nửa vời.
> 4. **Giữ nguyên lưới an toàn**: Tuyệt đối không sửa đổi mã giao diện `internal/dash/web/index.html`
>    và giữ hai bài kiểm ghim (`TestTokenCostCuaPhienGhiChuaDo`, `TestTienDoNoiBuocThuMayHongODauVaTonBaoNhieu`)
>    xanh nguyên trạng.
>
> Xem `docs/DO-LUONG.md`, mục 21/08 C4.

<details><summary>Nội dung ô nợ khi còn mở</summary>

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

</details>

## ~~C5~~ ✅ ĐÃ ĐÓNG 21/08 — Ngưỡng chạy quẩn `TranLapLienTiep = 10` ĐÃ ĐO TRÊN BẢN GHI THẬT

> **Kết quả**: đo trên toàn bộ 4 bản ghi lượt chạy bình thường thật trong
> `~/.ai-accounts/.nhat-ky/` (đối chiếu cột `log` của bảng `sessions` trong `state.db`,
> các phiên #174, #175, #176, #177 có `state = "done"`, từ 9 tới 70 tool call mỗi phiên):
> **chuỗi lặp liên tiếp dài nhất trên MỌI phiên là 1** (không có bất kỳ hai lời gọi tool
> nào giống hệt nhau liên tiếp).
>
> **Ngưỡng 10 là AN TOÀN TUYỆT ĐỐI cả hai chiều**:
> - **Chiều không bắt oan**: cách mức chạy bình thường dài nhất (1 lần) tới **10 lần**.
>   Agent trong phiên làm việc thật liên tục đổi lệnh, đổi tham số, đổi file nên không
>   bao giờ chạm tới ngưỡng 10.
> - **Chiều không bỏ sót**: cách ca quẩn thật duy nhất (#21 với 399 lần `ls -la`) tới
>   **gần 40 lần**.
>
> **Riêng Antigravity**: ghi rõ **KHÔNG ĐO ĐƯỢC** vì định dạng nhật ký của Antigravity
> chỉ mang `tool_name` (`run_command`), không có trường tham số/input để dựng chữ ký.
> Bộ đọc trả `DemDuocTool=false` và `Quan()` trả `biet=false` ("không biết") đúng theo
> thiết kế — thà không kết luận còn hơn bắt nhầm.
>
> Giữ nguyên `TranLapLienTiep = 10`. Xem `docs/DO-LUONG.md`, mục 21/08 C5.

<details><summary>Nội dung ô nợ khi còn mở</summary>

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

</details>

## ~~C6~~ ✅ ĐÃ ĐÓNG 21/08 — nhưng đóng NGƯỢC chiều sổ tưởng: Cursor **có** cờ đổi thư mục

> ⚠ **CẢNH BÁO CHO NGƯỜI ĐỌC SỔ**: ghi chú *"ĐÃ LẠC HẬU"* trước đây ở đầu mục này
> nói ô đóng ở commit `bacc137` theo chiều **KhongLamDuoc** — *"`--help` bản
> 2026.08.11 không có `--cwd` lẫn `-C`, nên `ArgsThuMuc` trả `nil` CÓ CĂN CỨ"*.
> **Câu đó SAI**, và sai theo kiểu tệ nhất: nó tự xưng là một **kết luận đã đo**.
> Mở `--help` của **đúng bản đó** ra đọc hết thì có cờ đổi thư mục thật.

- **Ở đâu**: `internal/provider/cursor.go:363` (`ArgsThuMuc`),
  `internal/provider/cursor.go:395` (dòng khai `NLThuMuc`).
- **Bài kiểm mới**: `internal/provider/danhtinh_cursor_test.go`
  (`TestCursorArgsThuMucDungWorkspace`, `TestCursorKhongDungAddDirChoThuMuc`,
  `TestCursorKhaiThuMucKhopVoiHam`).

### Câu hỏi đặt hẹp, nên kết luận sai

Phép đo cũ hỏi: *"có `--cwd` hay `-C` không?"* → **không**. Rồi nhảy thẳng sang
*"provider này không có cờ đổi thư mục"*. Vế đầu **đúng**; bước nhảy **sai**. Câu
đáng hỏi là *"có cách nào khai thư mục làm việc không?"* — và `--help` của
`cursor-agent 2026.08.11-e8db854` có:

```
--workspace <path-or-name>  Workspace directory or saved workspace name to use
                            (defaults to current working directory)
--add-dir <path>            Add an additional workspace root directory
```

### Phép đo, CÓ ĐỐI CHỨNG

Tạo `%TEMP%\do-thumuc-cursor\VAN-TAY-9F3A2B.txt`, rồi chạy từ cwd
`C:\Users\Administrator` — **không** phải thư mục đó:

```
có --workspace <dir>  -> agent liệt kê "VAN-TAY-9F3A2B.txt"        <- thấy
không có cờ           -> "no", và tự khai cwd C:\Users\Administrator
```

Có đối chứng nên kết luận được rằng **chính cái cờ** gây ra khác biệt, không phải
thứ gì khác. Cờ **được nhận và có hiệu lực**, không bị nuốt im lặng.

**Chọn `--workspace` chứ không phải `--add-dir`** dù bản này có cả hai:
`--add-dir` là *"Add an **additional** workspace root"* — thêm một gốc nữa, trong
khi hợp đồng `ArgsThuMuc` (`internal/provider/adapter.go:59-66`) đòi khai **tường
minh** thư mục làm việc. Claude và Antigravity phải dùng `--add-dir` vì CLI của
chúng không có cờ đặt thẳng; Cursor có, nên dùng cái đúng nghĩa hơn.

### Vì sao lý lẽ "không cần" cũng sai

Sổ cũ chống chế: *"Không cần: fleet đã chạy tiến trình con với `workDir` là
worktree của phiên"*. Lý lẽ đó bỏ qua **đúng cái** mà hợp đồng `ArgsThuMuc` sinh
ra để chặn — và hợp đồng nói thẳng ngay tại chỗ: fleet chạy agent trong **git
worktree**, mà ở worktree `.git` là **FILE con trỏ** chứ không phải thư mục, nên
provider dò workspace bị **hụt dù cwd đã đúng**. Đó không phải lo xa: đã đo trên
Antigravity — cùng lệnh cùng cờ, ở repo thật **3/3**, ở worktree chỉ **1/3**;
thêm cờ thì **4/4**. **cwd đúng KHÔNG bảo đảm workspace đúng.**

Và Cursor là provider chạy trong worktree nhiều nhất. Nên `nil` ở đây không phải
"vô hại vì không cần" — nó là cái bẫy 1/3 đang mở, chỉ chưa ai đo trúng.

### Hậu quả thật của lời khai sai

Mục *"Vì sao cần sổ này"* xếp `LamDuoc` sai nguy hiểm hơn `ChuaDo`. `KhongLamDuoc`
**cùng hạng nguy hiểm đó** và ô này là ví dụ: `KhongLamDuoc` tự xưng là **một kết
luận** (*"đã đo, provider không có thứ đó"*), nên **không ai đi đo lại** — khác
`ChuaDo`, thứ tự nó mời người sau đến đo. Một `ChuaDo` sai thì tốn công; một
`KhongLamDuoc` sai thì **đóng vĩnh viễn** một năng lực có thật.

Cụ thể ở đây: `sagent nang-luc` **nói dối theo chiều thiếu** — báo Cursor không
đổi được thư mục, trong khi nó đổi được. Người vận hành đọc bảng rồi tự đi tìm
đường vòng cho một vấn đề đã có cờ giải sẵn.

### Chỗ dễ vấp cho người sửa sau

Phép dò `NLThuMuc` (`internal/provider/nangluc.go:145-148`) là **HAI CHIỀU**
(`haiChieu: true`), nên `KiemNangLuc` **bắt buộc** hàm và lời khai đi cùng nhau:
đổi `ArgsThuMuc` mà quên dòng khai (hoặc ngược lại) là đỏ ngay. Nhưng nó chỉ bắt
được **lệch giữa hai vế trong repo** — nó **không** biết CLI thật có cờ gì. Cả
hai vế cùng sai một cách nhất quán, như suốt từ `bacc137` tới nay, thì nó vẫn
xanh. Chỉ có đọc `--help` thật mới bắt được.

### Bài học nền, nay có ví dụ thứ hai

Ô này vốn đã mang sẵn một bài học: *"một dòng CHƯA ĐO không tự hết hạn"* — bảng cũ
ghi *"máy này không cài cursor-agent"* trong khi `Get-Command` cho ra đường dẫn
thật. Nay thêm vế mạnh hơn, và là vế mà chính ô này vừa dính:

**một dòng ĐÃ ĐO cũng không tự hết hạn — và nó còn nguy hơn, vì nó không tự nhận
là khoảng trống.** Đọc `--help` sót một dòng vào năm ngoái thì hôm nay vẫn sót,
trừ khi có người mở lại file thật. Đúng cách ô V3 vừa được đóng: không tin lời
khai, không tin tiêu đề commit — mở file thật ra đọc.

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

## ~~V3~~ ✅ ĐÃ ĐÓNG 21/08 — `Identity()` đọc được thật, lời khai `Duoc` nay có căn cứ

> **Đã trả**: ô này mở ra vì lời khai `Duoc(NLDanhTinh)` không khớp một hàm luôn
> trả rỗng. Lượt trước vá bằng cách **hạ lời khai** xuống `ChuaDo` — khớp được
> hai vế, nhưng phần **đo cách đọc danh tính** vẫn còn nợ, và sổ ghi rõ là còn
> nợ. Lượt này trả nốt phần đo đó, và phép đo lật lại kết luận cũ: **có** trường
> danh tính trong file. Nên ô đóng theo chiều ngược với dự kiến — sửa **hàm** cho
> đúng lời khai, chứ không hạ lời khai cho vừa hàm.

- **Ở đâu**: `internal/provider/cursor.go:137` (`Identity`),
  `internal/provider/cursor.go:170` (`danhTinhTuJWT`),
  `internal/provider/cursor.go:416` (dòng khai `NLDanhTinh`).
- **Bài kiểm mới**: `internal/provider/danhtinh_cursor_test.go`.

### Phép đo đã còn nợ, nay làm xong

Mở `%APPDATA%\Cursor\auth.json` thật trên hồ sơ **đang đăng nhập** và **giải
payload JWT** ra (lượt trước mới ngó tầng ngoài rồi dừng):

- Thư mục `%APPDATA%\Cursor` có **ĐÚNG MỘT file**: `auth.json`, 893 byte. Không
  còn chỗ nào khác để mà tìm danh tính — điều này chốt luôn phạm vi.
- Tầng ngoài: **ĐÚNG HAI khoá** `accessToken`/`refreshToken`. Không `email`,
  không `userEmail`, không `user_email`. **Phần này lượt trước nói đúng.**
- Payload hai JWT giống hệt nhau, có **ĐÚNG TÁM claim**:

  ```
  iss         https://authentication.cursor.sh
  aud         https://cursor.com
  sub         google-oauth2|user_01<...>          <- trường danh tính DUY NHẤT
  scope       openid profile email offline_access
  type        session
  randomness  <uuid cụt>
  time        1787022539
  exp         1792206539
  ```

**Chỗ lượt trước sai**: sổ cũ viết *"payload JWT cũng chỉ có `sub` dạng mã đục,
không phải địa chỉ thư"* → rồi kết luận **CHƯA ĐỌC ĐƯỢC**. Vế mô tả đúng, bước
kết luận sai. Câu mà năng lực này hỏi là *"đọc được danh tính để hiển thị"*,
không phải *"đọc được email"*. `sub` là subject của OIDC: bền, mỗi tài khoản một
giá trị — nó **là** danh tính, chỉ là không ở dạng email.

### Vì sao trả `sub` mà khai `Duoc` là hợp lệ

Tiền lệ có sẵn trong chính repo: `internal/provider/grok.go` — `Identity` ở đó
trả `baseURL · defaultModel` thay cho email và vẫn khai `Duoc(NLDanhTinh)`, kèm
ghi chú *"provider này không có khái niệm tài khoản người dùng"*. Tức bảng năng
lực **đã** chấp nhận danh tính phi-email từ trước.

Với Cursor thì còn thẳng hơn Grok: Cursor **có** khái niệm tài khoản, và
`NLTachTaiKhoan` = `Duoc` nghĩa là người vận hành chạy nhiều tài khoản Cursor
song song. Câu họ cần trả lời khi nhìn card là *hồ sơ này là tài khoản nào* —
`sub` trả lời đúng câu đó, `""` không trả lời gì.

**Không mâu thuẫn với V2** (*"hiện nhầm email còn tệ hơn không hiện gì"*): chỗ
Antigravity nguy hiểm vì `google_accounts.json` cho ra một email **có thật nhưng
sai người**, nhìn đúng định dạng nên không ai nghi. `google-oauth2|user_01…`
không thể bị nhầm là địa chỉ thư, và nó là danh tính của **chính** hồ sơ đang
đọc. Trông xấu, nhưng không nói dối — và đó mới là ranh giới V2 vạch ra.

### BẪY đã tránh, ghi lại vì suýt lặp đúng lỗi cũ

Claim `scope` **có chữ "email"** (`openid profile email offline_access`). Đó là
**phạm vi OAuth đã xin**, KHÔNG phải một claim email — payload không có claim
`email`. Đọc lướt thấy chữ "email" rồi khai "đọc được email" chính là **đúng kiểu
sai** đã sinh ra ô này. Đã ghim bằng `TestCursorKhongNhamScopeLaEmail`.

### Đã bịt lỗ "phép dò một chiều"

Đây mới là phần đáng giữ của ô này. Phép dò `NLDanhTinh` trong
`internal/provider/nangluc.go:168` là **một chiều** (`haiChieu: false`): nó chỉ
kết luận được chiều *"khai chưa đo mà lại trả giá trị thật"*. Chiều ngược lại —
*"khai làm được mà luôn trả rỗng"* — **không có ai canh**, nên `KiemNangLuc` xanh
suốt trong khi lời khai sai. Cùng lỗ đó đang che cho `NLCoTuHoSo`,
`NLKetQuaCoCauTruc` và `NLHanToken`.

Sổ đã chốt cách bịt: **không** đổi phép dò (một chiều là **đúng** — cần hồ sơ
thật mới dò được), mà bằng một bài kiểm chạy trên hồ sơ thật. Nay có:
`TestCursorKhaiDanhTinhKhopVoiHam` dựng một `auth.json` **đúng hình dạng file
thật** rồi bắt hai vế khớp nhau **cả hai chiều** — khai `Duoc` mà trả rỗng thì
đỏ, mà khai thấp hơn `Duoc` trong khi đọc được cũng đỏ.

Không nhét token thật vào repo: chữ ký JWT không được kiểm nên token giả là đủ,
và một token thật trong mã nguồn là một rò rỉ. `subGia` giữ **đúng dạng** của
`sub` thật nhưng id là bịa.

**Đã kiểm trên hồ sơ thật**: chạy `Identity()` lên `%APPDATA%` thật cho ra
`google-oauth2|user_01M09…` — không còn rỗng. (Bài kiểm tạm đó **không** commit:
nó phụ thuộc máy.)

### CÒN NỢ (thu hẹp, không mất)

`cursor-agent status` **in được email** (đã thấy khi đo `NLTachTaiKhoan`). Tức
vẫn còn một đường đọc ra **email thật**, đẹp hơn `sub` đục. Nhưng đọc danh tính
bằng cách **chạy CLI con** là đánh đổi khác hẳn đọc file — tốn một tiến trình mỗi
lần vẽ card — nên **chưa đo, chưa làm**. `Identity()` đã dọn sẵn chỗ: nó ưu tiên
`email` (tầng ngoài, rồi claim JWT) trước `sub`, nên ngày nào có email thì card
tự đổi mà không phải sửa hàm.

---

# ⚪ NỢ NGƯỢC — đã đo rồi mà sổ chưa xoá

## ~~N1~~ ✅ ĐÃ ĐÓNG 21/08 — bình luận rotation đã sửa cho khớp phép đo 20/08

> **Đã trả**: `internal/profile/tokenhoisinh_test.go:26-35` không còn câu *"cái
> đó vẫn CHƯA ĐO"*. Bình luận mới giữ nguyên phạm vi thật của bài kiểm (nó không
> **dựa vào** rotation), rồi ghi thẳng phép đo 20/08 kèm vân tay
> `5d708911` → `1aa28b8c` và nguyên văn *"OAuth session expired and could not be
> refreshed"* — và nói ra hệ quả mà người đọc cần: bước 3 hồi sinh một token đã
> **chết**, nên bản sửa `Clone` → `SyncBackTokens` là cần thiết, không phải
> phòng xa quá mức.
>
> **Sửa thêm một chỗ cùng loại** tìm được bằng chính câu lệnh quét:
> `internal/fleet/fleet_test.go:264-268`. Bình luận đầu bài viện lý do *"hành vi
> refresh đồng thời thì CHƯA ĐO"* để giữ câu cảnh báo, trong khi thân bài
> (dòng 284-289) đã đòi câu cảnh báo phải chứa chữ **XOAY VÒNG** — tức đầu bài và
> thân bài nói ngược nhau. Nay đầu bài nói đúng cái đã đo, và vẫn ghi rõ phần
> **chưa** đo (cuộc đua N tiến trình cùng refresh — ô Đ5) thay vì gộp hai thứ đó
> làm một.
>
> Ô Đ5 vẫn **mở**: đóng N1 là đóng chuyện *rotation có xảy ra không*, không phải
> chuyện *N bản cùng refresh thì ai thắng*.

<details><summary>Nội dung ô nợ khi còn mở</summary>


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

</details>

---

# Lưới an toàn — những bài kiểm đang giữ các ô trên

Phần lớn dòng `CHƯA ĐO` còn lại trong repo **không phải nợ**: chúng là bài kiểm
canh cho các ô nợ ở trên không lặng lẽ biến thành lời nói bừa. Ghi ra đây để
người sau đừng dọn nhầm.

| File:dòng | Canh cái gì | Giữ ô nào |
|---|---|---|
| `internal/api/quyen_test.go:48-52` | Provider CHƯA ĐO mà vẫn chạy tiếp = đỏ | Đ3 |
| `internal/api/tuduyetquyen_test.go:24,61,72` | Bảng khai nói "đã đo" mà `ArgsTuDuyetQuyen()` nói chưa = đỏ | Đ1, Đ3 |
| `internal/api/model_test.go` (`TestProviderChuaDoModelThiPhaiCanhBao`) | Chưa đo model thì phải **cảnh báo** và vẫn **chạy được** — không im lặng, không chặn. Vật thử nay là adapter GIẢ: từ 21/08 không provider thật nào còn `nil` | C3 |
| `internal/api/model_test.go` (`TestAntigravityVaCodexTruyenDuocModel`, `TestCodexDatCoModelTruocLenhCon`) | Hai provider đã đo phải truyền cờ xuống thật, **không cảnh báo thừa**, và `-m` của Codex phải đứng **trước** lệnh con `exec` | C3 |
| `internal/provider/nangluc_test.go:78,230` | `(nil, false)` phải đọc là `ChuaDo`; conformance đối chiếu bảng khai với hành vi thật | toàn bộ mức Đỏ |
| `internal/provider/token_cursor_test.go` | Hạn token Cursor đọc từ **refresh** chứ không phải access; và **mọi ngõ hỏng phải trả `false`**, cấm bịa ra một mốc | Đ4 |
| `internal/provider/trangthai_test.go:128` | Provider chưa đọc được kết quả thì phiên **ở lại `lost`**, không kết luận | C1 |
| `internal/provider/quan_test.go:212-214` | Không kết luận "chạy quẩn" từ bản ghi chưa đọc được | C1, C5 |
| `internal/provider/bosungco_test.go:50` | Chưa đọc được kết quả có cấu trúc thì không khai bừa cờ | C1 |
| `internal/dash/mat2d_test.go:165,182` | Ô chưa có số phải ghi `CHUA_DO`; cấm điền `0`, cấm đọc trường không tồn tại | C4 |
| `internal/dash/ngankeo_test.go:273` | Khối tiến độ phải đi qua `datSo()` | C4 |
| `internal/fleet/fleet_test.go:281-289` | Cảnh báo phải nói **đúng số bản** và nói ra hậu quả xoay vòng | Đ5, N1 |
| `internal/profile/duarefresh_test.go` | Đồng bộ ngược KHÔNG được chọn file rỗng của bản thua cuộc đua, dù mtime mới nhất | Đ5 |
| `internal/provider/duarefresh_banthua_test.go` | Bản ghi của bản thua phải ra `failed` kèm nguyên văn câu lỗi, không rơi về `lost` | Đ5, C1 |
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
2. ~~**Đ5** (cuộc đua N-clone)~~ — **đã đóng 21/08**, và nó không chỉ trả lời
   câu hỏi cũ: phép đo lôi ra hai lỗi thật trong `SyncBackTokens` và
   `PhanLoaiChet`. Nửa còn lại của cái bẫy đã cắn một lần thì cắn theo một kiểu
   khác hẳn nửa đầu.
3. ~~**N1**~~ — **đã trả 21/08**, chỉ sửa bình luận, không phải đo gì.
4. ~~**Đ4 Cursor**~~ — **đã đóng 21/08**, và rào chắn cũ sai lần thứ hai theo
   đúng một kiểu: lần đầu tưởng máy chưa cài `cursor-agent`, lần này tìm
   `auth.json` **nhầm thư mục** (`~/.cursor/` thay vì `%APPDATA%\Cursor\`). Cũng
   không cần "dựng cảnh token sắp hết hạn" như đề nghị cũ: mốc nằm trong payload
   JWT, đọc là ra. **Một dòng CHƯA ĐO không tự hết hạn — và lý do hoãn của nó
   cũng không tự đúng lại.**
5. ~~**C5** (ngưỡng chạy quẩn)~~ — **đã đóng 21/08**, đo trên các bản ghi bình
   thường thật trong kho nhật ký, chuỗi lặp dài nhất = 1, ngưỡng 10 an toàn.
6. ~~**C4** (token/chi phí phiên CLI)~~ — **đã đóng 21/08**, DTO đọc từ log của phiên đã kết thúc qua `DocKetQua()`, phân biệt số thật của Claude với "chưa đo" của Codex/Cursor/Antigravity, không lộ số 0 giả.
7. ~~**Đ4 Antigravity**~~ — **đã trả 21/08 theo đúng đề nghị này**: đổi sang
   `Khong(NLHanToken)` kèm lý do, theo tiền lệ của Grok. Không đo gì cả — đó là
   nội dung của việc, không phải chỗ cắt bớt.
8. ~~**C3** (chọn model từ dòng lệnh)~~ — **đã đóng 21/08**, và nó bác chính
   khuôn bằng chứng mà tiền lệ Cursor đặt ra: "CLI từ chối tên model bịa" đúng
   với Antigravity nhưng **sai với Codex** (Codex nhận cờ, máy chủ mới chặn).
   Lượt này còn suýt kết luận ngược vì tin lời agent tự khai danh tính — xem mục
   C3, phần "một phép đo đã bắt đầu sai".
9. **V3** (danh tính Cursor) — ô mới của lượt 21/08. Lời khai đã sửa; phần còn
   nợ là đo cách đọc email qua `cursor-agent status`.

# Phạm vi sổ này KHÔNG phủ

- **Chỉ quét mã Go.** `internal/dash/web/index.html` có hằng `CHUA_DO` và `docs/`
  có nhiều mục "còn treo"; sổ chỉ nhắc tới khi một dòng Go trỏ vào.
- **Không đo gì mới.** Mọi con số ở đây chép lại từ phép đo đã ghi trong mã hoặc
  trong `docs/DO-LUONG.md`. Không có suy luận nào được nâng lên thành phép đo.
- **Bản quét gốc không sửa file nào khác** — luật của lượt lập sổ là chỉ tạo
  đúng một file, nên ô N1 nhìn thấy được mà để nguyên tại chỗ. **Lượt 21/08 sau
  đó đã trả N1 trong mã** (`internal/profile/tokenhoisinh_test.go`,
  `internal/fleet/fleet_test.go`); giới hạn trên chỉ còn đúng với bản quét đầu.
- **C1 đã viết lại 21/08** (nửa Cursor đóng, tiêu đề bỏ Cursor ra, nửa Codex giữ
  nguyên từng chữ cho phiên đang làm nó). **C3 đã viết lại theo khuôn đóng 21/08**
  (đo thật Antigravity + Codex; văn bản cũ giữ trong khối `<details>`). **C6 vẫn
  mới chỉ được gắn nhãn "đã lạc hậu", chưa viết lại theo khuôn đóng** — nội dung
  mô tả bên dưới nhãn đó vẫn là văn bản cũ, còn nói Cursor chưa đo. Đọc nhãn
  trước, đừng đọc thẳng gạch đầu dòng.
