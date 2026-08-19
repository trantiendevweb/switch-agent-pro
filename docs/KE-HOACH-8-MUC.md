# Kế hoạch 8 mục — điều khiển và quan sát đội agent

> Trạng thái: **CHỜ DUYỆT**. Soạn 19/08/2026 21:00.
> Duyệt xong thì bám sát file này; đổi hướng thì sửa file này trước, không đổi ngầm.

## Bối cảnh

Ngày 19/08 máy tự khởi động lại lúc 01:47 (lsass.exe chết), cắt ngang lượt chạy #29.
Đọc lại lượt đó phơi ra một hình dạng lỗi lặp đi lặp lại: **hệ thống báo ỔN trong khi
không có cơ sở để nói thế** — placeholder chưa thay lọt vào prompt, `agent báo lỗi:
success`, `flow run` đốt một lượt có 4 bước chết sẵn, người soi phán ngược nhánh,
`expiresAt=0` bị hiểu nhầm thành "sẵn sàng".

Chủ dự án đặt yêu cầu trước khi cho chạy các lượt đêm dài: **phải nhìn được đội agent
đang làm gì, can thiệp được, và được báo khi có sự cố.** Tám mục dưới đây là yêu cầu đó,
tách nhỏ ra thành việc làm được.

Nguyên tắc xuyên suốt, rút từ chính các lỗi trên: **thà nói "không biết" còn hơn đoán
theo hướng lạc quan.** Mọi mục đều phải trả lời được câu "nếu dữ liệu nguồn đổi hình,
chỗ này im lặng nói ổn hay ồn ào báo không biết?"

---

## Đã xong (19/08)

| Mục | Việc | Commit |
|---|---|---|
| 1 | Realtime + hội thoại từng agent | `14be68b` |
| 2 | Giao diện dễ theo dõi (bản đầu) | `14be68b` |
| — | Cổng kiểm tài khoản trước khi chạy flow | `1f77a01` |
| — | `flow huy` — huỷ lượt chạy bị cắt ngang | `4fa396f` |
| — | Sửa `expiresAt=0`, `HasToken` nói thật | `714730c` |
| — | Chặn placeholder chưa thay lọt vào prompt | `c67a1ff` |

Nền tảng dùng lại cho mọi mục sau: `FlowRunDetail()` trong `internal/api/api.go`,
endpoint `/api/flow/detail`, cột `prompt` (schema v6), bus sự kiện `internal/events`.

---

## Đợt A — Lõi tin cậy (đang chạy)

Ba việc này chạy song song, mỗi việc một worktree riêng, có người soi kiểm bằng `git`.

### Mục 7 — Telegram báo sự cố
- **Vì sao trước tiên**: máy chết lúc 01:47, tới 15:16 mới có người biết. **13 tiếng rưỡi
  chết lặng.** Không có mục này thì mọi lượt đêm đều là đánh cược.
- **Làm gì**: gói mới `internal/tele`, nghe bus sự kiện, gửi khi bước hỏng / lượt hỏng /
  chờ duyệt / xong. Cấu hình token ở `~/.ai-accounts/` (theo lệ `dash-auth.json`),
  **không nằm trong repo**.
- **Xong khi**: `sagent tele --thu` gửi được tin thật; test dùng `httptest.Server` giả,
  khẳng định cả ca "chưa cấu hình thì im lặng, không làm hỏng lượt chạy".

### Mục 3 — AI quản lý báo cáo
- **Vì sao chọn bản "quản lý" thay vì "can thiệp trực tiếp"**: agent headless **không có
  kênh nhận lệnh giữa chừng**. Muốn can thiệp thật phải đổi cách chạy agent, không phải
  thêm một nút bấm. Bản quản lý làm được ngay trên dữ liệu hội thoại đã có.
- **Làm gì**: `sagent flow tom-tat <số>` — một agent đọc toàn bộ hội thoại rồi trả lời:
  ai làm gì, ai chưa làm gì, bước nào hỏng vì sao, việc gì còn treo.
- **Điểm cốt tử**: bản tóm tắt phải kèm **bằng chứng máy tự đọc**. Với mỗi nhánh
  `sagent/*`, tự chạy `git rev-list --count main..<nhánh>` rồi **đối chiếu với lời agent**.
  Lệch nhau thì nói thẳng là lời agent mâu thuẫn với git, và **tin git**.
- **Xong khi**: có test dựng đúng tình huống "agent nói nhánh có việc nhưng git nói không
  có commit nào", và khẳng định bản tóm tắt chỉ ra được mâu thuẫn.

### Mục 6 — Luật chống loop
- **Số đo**: grok gọi `ls -la` **399 lần** trong một lượt. Hiện chỉ có biện pháp thô là
  hạ trần vòng tool xuống 60 (`internal/provider/grok.go`).
- **Làm gì**: đọc từ **dữ liệu có cấu trúc** của provider (theo `docKetQuaClaude`,
  `docKetQuaAntigravity`), đếm lệnh lặp lại. Vượt ngưỡng thì `KetQua.Hong()` báo rõ
  "agent lặp lại lệnh X n lần — nghi chạy quẩn", để người đọc biết **đây không phải lỗi code**.
- **Không được bắt nhầm**: chạy `git status` vài lần là bình thường. Ngưỡng phải ghi rõ
  vì sao chọn con số đó.
- **Phần "context riêng + xin/cho"** tách sang đợt C — nó là thay đổi kiến trúc, không
  phải một hàm.

---

## Đợt B — Mặt 2D (mục 5)

`internal/dash/web/flow.html` **đã là bảng vẽ thật**: `addNode`, `connect`, `autoLayout`,
`hasCycle`, `undo/redo`, `inspector`, `FlowSave` ghi thẳng xuống `flows.toml`. Không làm lại.

Thiếu ba thứ:

1. **Trang setting đầy đủ** — hiện chỉ có `.sagent/project.toml` sửa bằng tay.
   Cần: trần phiên song song, lệnh test/lint, provider mặc định, ngưỡng chống loop,
   cấu hình Telegram. Đi qua `/api/config` (action `config.show` đã có).
2. **Nhúng AI dựng flow** — người dùng gõ yêu cầu tiếng Việt, AI sinh ra flow rồi đổ lên
   bảng vẽ để sửa tay trước khi lưu. Dùng lại đường `internal/aiapi` (đã có, đi theo token
   chứ không tiêu hạn mức thuê bao). **Bắt buộc cho xem trước và sửa** — không tự lưu đè.
3. **Thao tác nâng cao** — nhân bản nhánh, chạy thử một bước, xem trước prompt sau khi
   thay biến (dùng `flow.Expand`, KHÔNG dùng `ExpandChay` vì lúc chưa chạy thì chưa có
   kết quả bước nào là chuyện bình thường).

---

## Đợt C — Mặt 3D (mục 4) và context riêng (phần còn lại mục 6)

`internal/dash/web/3d.html` **đã có robot** (`taoRobot`, `mascotTex`, `vaiTro`,
`buocCuaAddr`), three.js r128, đọc `/api/state` + `/api/flows` + `/api/run`, tô sáng bước
đang chạy. Không làm lại.

Thiếu:
1. **Bong bóng thoại** — nối vào `/api/flow/detail` (đã có) để robot "nói" đúng câu agent
   vừa trả lời.
2. **Phòng** — mỗi flow đang chạy là một phòng; robot đứng theo vai trò trong sơ đồ.
3. **Vai trò hiện rõ** — hiện `vaiTro()` suy từ cạnh `needs`. Nâng thành dữ liệu thật
   (khai trong `flows.toml`), vì "vai trò là dữ liệu hay chỉ là quy ước" là câu hỏi
   `docs/DU-AN-THAM-KHAO.md` đã nêu từ bài học Gas Town.
4. **Context riêng + xin/cho** — mỗi agent một kho context; muốn đọc của nhau phải gửi
   yêu cầu, hoặc có lệnh của AI quản lý (mục 3). Đây là thay đổi kiến trúc, làm sau khi
   mục 3 đã chạy thật.

**Quyết định đang chờ khảo sát**: có dự án mã nguồn mở nào làm sẵn phần 3D multi-agent
dùng được trong **một file HTML tĩnh** (dự án này nhúng file tĩnh, không có bundler).
Có thì học/chép theo giấy phép và ghi vào `docs/OPEN_SOURCE_LEDGER.md`; không có thì tự
build trên nền `3d.html` sẵn có.

---

## Đợt D — Chạy đêm thật và đo

Chỉ chạy sau khi đợt A xong, vì trước đó không có báo Telegram và không có luật chống loop.

- Giao việc thật: mục #2 bảng ưu tiên trong `docs/DU-AN-THAM-KHAO.md` (chương trình dò ACP
  cho 5 provider) — chính bảng đó ghi nó chặn cả hướng ACP.
- Đo và ghi vào `docs/DO-LUONG.md`: lượt chạy có bị cắt không, Telegram có báo không,
  luật chống loop có bắt nhầm không, AI quản lý có phát hiện mâu thuẫn không.

---

## Việc lặt vặt đang treo

| Việc | Vì sao |
|---|---|
| Trộn `f7206fd` (`docs/KHAC-PHUC-SU-CO.md`, 148 dòng) | Antigravity làm ở lượt #33, chưa rà |
| Thêm `--kho` (chạy khan) cho `flow run` | Tôi đã bấm nhầm chạy thật **3 lần** hôm nay (#30, #32, #33) |
| Siết tường lửa cổng 8788 về đúng IP của chủ dự án | Máy đang bị dò ~2.400 lần/giờ |
| Dọn `sagent/may-1-2`, `sagent/may-1-luot29` | Nhánh rác từ các lượt hỏng |

---

## Cách nghiệm thu (áp cho mọi mục)

1. `go build ./...` và `go test ./...` xanh.
2. **Test phải ĐỎ khi gỡ phần sửa ra** — nêu rõ đã chứng minh bằng cách nào. Test chỉ gọi
   thẳng hàm là chưa đủ khi chỗ hỏng nằm ở **chỗ gọi**.
3. **Luật ngang quyền**: mỗi action mới phải có đủ ba mặt — `api.Actions`, lệnh CLI,
   đường vào từ web. Thiếu một mặt là chưa xong.
4. Giao diện: không emoji làm icon, có `@media (prefers-reduced-motion: reduce)`.
5. Chạy thật một lần trên máy này, dán số đo vào `docs/DO-LUONG.md`. Chưa chạy thật thì
   ghi "chưa đo được".

## Rủi ro còn lại

- **Máy vẫn bị brute-force ~2.400 lần/giờ.** Security log 20 MB quay vòng hết trong ~8
  giờ, nên sự cố ban đêm mất log trước khi kịp đọc. Chưa xử lý.
- **grok phán ngược 2/2 lượt.** Mục 3 vá được triệu chứng (đối chiếu git), nhưng vai
  "người soi" giao cho grok vẫn là lựa chọn đáng xét lại.
- **Token Claude hết hạn ~8 tiếng/lần.** Lượt đêm dài phải tính chuyện làm mới giữa chừng;
  hành vi khi nhiều bản clone cùng refresh **chưa đo**.
