# Kế hoạch — Mặt 2D gọn lại, và Văn phòng 3D

> Soạn 20/08/2026 sau khi chủ dự án xem bản 2D/3D đầu tiên và chốt 11 lựa chọn.
> File này là nguồn sự thật cho hai mặt. Đổi hướng thì sửa đây trước.

## Bối cảnh

Bản 2D và 3D dựng đêm 19/08 đã đạt yêu cầu kỹ thuật (offline, token chung,
không addon), nhưng chủ dự án nhìn thật thì thấy hai chỗ chưa được:

1. **2D chưa dễ dùng.** Cột phải xếp **6 khối form dọc liền nhau**, tất cả mở
   sẵn: bật hạm đội, chạy workflow, hỏi AI API, báo tin Telegram, máy & dọn dẹp.
   Màn hình đầy thứ để *thao tác*, trong khi thứ cần nhìn liên tục — hạm đội,
   tiến độ, nhật ký — bị đẩy xuống.
2. **3D là sơ đồ, không phải nơi làm việc.** Orb xếp trên ring quanh core thì
   đọc trạng thái nhanh, nhưng không cho biết *đội đang làm việc ra sao*.

Chủ dự án muốn 3D thành **văn phòng có phòng ban**: CEO, leader, coder, tester;
agent đi lại giữa các phòng, giao việc cho nhau, nói chuyện theo thời gian thực.

---

## 11 quyết định đã chốt

| # | Câu hỏi | Chốt |
|---|---|---|
| 1 | Vai trò lấy từ đâu | **Khai trong `flows.toml`** cho từng bước — vai trò là DỮ LIỆU, không phải hình vẽ |
| 2 | Phòng chia theo gì | **Theo LOẠI VIỆC**: phòng họp · phòng code · phòng test · phòng review |
| 3 | Bóng thoại hiện gì | **Câu trả lời THẬT, rút gọn** — lấy từ `/api/flow/detail`, không bịa chữ nào |
| 4 | 2D thiếu gì nhất | **Quá nhiều form bày cùng lúc** |
| 5 | Đi lại | **Vừa đi thật vừa có nhịp thở nhẹ** — di chuyển chỉ khi có việc, đứng yên vẫn cử động nhỏ |
| 6 | Văn phòng thay hay thêm | **Thêm mặt mới**, giữ cả `3d.html` (sơ đồ) lẫn `vanphong.html` (văn phòng) |
| 7 | Nhân vật | **RobotExpressive.glb** của three.js (CC0, 13 clip đặt tên theo trạng thái) |
| 8 | Thứ tự | **2D trước, 3D sau** |
| 9 | Xung đột luật addon | **Nới luật**: cho phép loader, vẫn cấm addon hiệu ứng |
| 10 | Gom form 2D | **Ngăn kéo trượt ra từ phải** |
| 11 | Bước không có vai | **Phòng chung ở giữa** — thấy ngay là "chưa phân vai", không giấu |

---

## Đợt 1 — Mặt 2D gọn lại (làm trước)

`internal/dash/web/index.html`. **Không bỏ chức năng nào** — chỉ đổi cách bày.

**Màn chính chỉ giữ ba thứ cần nhìn liên tục:**
- Hạm đội (lưới card)
- **Tiến độ lượt chạy** — mới: đang ở bước mấy trên tổng bao nhiêu, bước nào hỏng
- Nhật ký sự kiện

**Sáu form chuyển vào ngăn kéo trượt từ phải.** Mỗi form một nút trên thanh công
cụ. Đóng lại là màn sạch. Ngăn kéo phải: đóng được bằng `Esc`, bẫy focus khi mở,
trả focus về nút đã mở nó.

**DoD:** mở dashboard trên màn 1080p thấy đủ hạm đội + tiến độ + nhật ký **không
cuộn**. Mọi thao tác cũ vẫn làm được, chỉ thêm một cú bấm.

---

## Đợt 2 — Vai trò thành dữ liệu

Nền cho cả 3D lẫn 2D. Không có bước này thì văn phòng chỉ là hình vẽ.

- Thêm `vai_tro` vào `flow.Step`: `ceo` · `leader` · `coder` · `tester` · `soi`.
- Rỗng = chưa phân vai, hiện ở **phòng chung**. Không đoán hộ.
- `/api/flow/detail` trả thêm trường này; `--kho` in ra để xem trước.
- Gán vai cho flow `doi-4`: `ke-hoach`→leader, `code-go`→coder, `code-doc`→coder,
  `kiem-*`→tester, `soi`→soi, `gop`→ceo.

**DoD:** đổi `vai_tro` trong `flows.toml` → cả 2D lẫn 3D đổi theo, không sửa mã.

---

## Đợt 3 — Văn phòng 3D

File mới `internal/dash/web/vanphong.html`. `3d.html` giữ nguyên.

**Bố cục:** bốn phòng quanh một sảnh chung. Sàn + vách thấp bằng hình khối đơn
sắc — theo luật, **màu chỉ dành cho trạng thái agent**, phòng ốc giữ đơn sắc.

**Nhân vật:** `RobotExpressive.glb` (CC0). 13 clip đã đặt tên theo trạng thái —
đúng thứ cần, khỏi tự nặn: `Idle` khi rảnh · `Walking` khi chuyển phòng ·
`Running` khi đang chạy bước · `Wave` khi giao việc · `ThumbsUp` khi bước xong ·
`No` khi bước hỏng.

**Chuyển động mang thông tin:**

| Thấy gì | Nghĩa là |
|---|---|
| Đi từ phòng này sang phòng khác | Bước chuyển sang loại việc khác |
| Đi tới agent khác rồi vẫy tay | Giao việc — có cạnh `needs` giữa hai bước |
| Đứng yên, thở nhẹ | Rảnh. Nhịp thở là thứ DUY NHẤT không mã hoá thông tin, có để cảnh không chết cứng |
| Bóng thoại | Dòng đầu output THẬT của agent |

**Máy chấm là nhân vật riêng**, khác hình với agent — để thấy rõ đâu là người,
đâu là máy.

**DoD:** chạy `doi-4` thật, nhìn văn phòng suy được đúng thứ tự các bước đã xảy
ra mà không cần mở log.

---

## Sửa skill — mục addon

`.claude/skills/sagent-dashboard/SKILL.md` hiện cấm **mọi** addon three.js. Luật
đó sinh ra từ một ca thật (prototype để three.js ở CDN → màn trắng), nhưng nó
gộp hai thứ khác nhau làm một.

Tách lại:

- **Vẫn cấm addon HIỆU ỨNG** — `EffectComposer`, `UnrealBloomPass`, `OrbitControls`.
  Chúng kéo theo chuỗi phụ thuộc dài và đều có cách tự viết rẻ hơn.
- **Cho phép LOADER đã vendor** — `GLTFLoader` là một file độc lập, không kéo
  theo gì, và nó mở ra thứ không tự viết được: mô hình có sẵn 13 clip hoạt hình
  đặt tên theo trạng thái.

Điều kiện không đổi: **vendor vào binary, không tải từ mạng lúc chạy**. Binary
tăng thêm ~1 MB (GLTFLoader ~150 KB + .glb ~800 KB), từ 13,7 lên ~14,7 MB.

---

## Cách nghiệm thu (áp cho mọi đợt)

1. `go build ./...` và `go test ./...` xanh.
2. Test phải ĐỎ khi gỡ phần sửa ra.
3. Luật ngang quyền: action mới phải đủ ba mặt — API, CLI, web.
4. Checklist skill: 0 asset ngoài · không emoji làm icon · `prefers-reduced-motion`
   · `:focus-visible` · mobile một cột.
5. Chạy thật một lần, dán số đo vào `docs/DO-LUONG.md`.

## Rủi ro

- **Binary phình.** 12,3 → 13,7 → ~14,7 MB. README phải sửa theo, hiện vẫn ghi 12 MB.
- **`.glb` có tải được không CHƯA ĐO.** Agent phải kiểm thật; không tải được thì
  nói thẳng, đừng tạo file rỗng rồi báo xong.
- **Ba mặt cùng đọc một nguồn.** 2D, sơ đồ 3D và văn phòng 3D phải dùng chung
  `/api/flow/detail` — mỗi mặt tự fetch kiểu riêng là bắt đầu lệch nhau.
