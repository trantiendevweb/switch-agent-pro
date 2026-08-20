---
name: sagent-dashboard
description: Dùng skill này BẤT CỨ KHI NÀO viết hoặc sửa giao diện dashboard của sagent (switch-agent-pro) — cả view 2D lẫn view 3D. Kích hoạt khi đụng tới file HTML/CSS/JS của dashboard, thêm/sửa panel, card agent, scene 3D, event log, meter, hay bất cứ thứ gì hiển thị trạng thái hạm đội agent. Skill này giữ cho output hiện đại, trực quan, đồng bộ giữa 2 surface, và KHÔNG vi phạm ràng buộc nhúng-binary/offline.
---

# Skill: Dashboard sagent

## Bối cảnh sản phẩm (đọc trước khi vẽ gì)

sagent là **local-first control plane** điều phối nhiều coding-agent CLI (Claude Code, Codex, Cursor, Antigravity, Grok) + AI API. Dashboard là assets **nhúng thẳng trong Go binary**, chạy **offline**, chỉ Windows. Có 2 surface: **2D** (bảng quan sát) và **3D** (không gian). Người dùng là operator đang chạy nhiều phiên song song — việc quan trọng nhất của màn hình: **nhìn phát biết hạm đội đang làm gì, tốn bao nhiêu, và dừng/lái được** (glanceable — nhìn phát hiểu ngay).

## Ràng buộc CỨNG (vi phạm là hỏng, không bàn)

1. **Offline tuyệt đối lúc runtime.** KHÔNG load `three.js` từ CDN, KHÔNG load font từ Google Fonts, KHÔNG gọi API ngoài để render. Mọi asset phải **vendor** (tải về nhét vào bundle nhúng). *(Bài học thật: bản prototype để three.js ở cdnjs → màn 3D trắng trơn trong môi trường thật.)*
2. **Vanilla, không Node build.** HTML/CSS/JS thuần. Không React/Vue/bundler. Assets nhúng qua Go `embed`.
3. **three.js: cấm addon HIỆU ỨNG, cho phép LOADER đã vendor.** (Sửa 20/08 — luật cũ cấm *mọi* addon, gộp nhầm hai thứ khác nhau.)
   - **Vẫn cấm**: `OrbitControls`, `EffectComposer`, `UnrealBloomPass`. Chúng kéo theo chuỗi phụ thuộc dài, và đều có cách tự viết rẻ hơn → camera orbit **tự viết tay**; "bloom" bằng **additive glow sprite**, không post-processing.
   - **Cho phép**: loader độc lập như `GLTFLoader`, với điều kiện **vendor vào binary**. Nó không kéo theo gì, và mở ra thứ không tự viết được: `RobotExpressive.glb` (CC0) có sẵn 13 clip hoạt hình **đặt tên theo trạng thái** (Idle/Walking/Running/Wave/ThumbsUp/No) — đúng thứ view văn phòng cần.
   - Điều kiện không đổi: **không tải từ mạng lúc chạy** (INV-UI-1).
4. **Không secret ra client.** Giữ nguyên nguyên tắc DTO allowlist của repo — UI chỉ nhận field trong allowlist, không nhận token/API key.
5. **Đã đo — không suy luận.** Con số hiển thị (token, cost, PID) phải từ state thật; đừng bịa số "cho đẹp".

## Design tokens (dùng đúng, đừng chế thêm màu)

**Nguyên tắc màu:** nền tối trung tính + hairline. **Màu chỉ dành cho status của agent.** Chrome (topbar, panel, grid) giữ đơn sắc. Đây là chỗ "dồn điểm nhấn vào một chỗ" — đừng rải màu ra khung.

```css
--void:#070810;  --panel:#0C0E17;  --panel-2:#11141F;      /* nền, sâu → nổi */
--line:rgba(255,255,255,.06);  --line-2:rgba(255,255,255,.12);
--hi:#EAECF4;  --mid:#969DB0;  --lo:#565D71;               /* chữ: chính/phụ/nhãn */
/* status = màu của agent (dùng cho cả pill 2D lẫn orb 3D) */
--run:#2FE0A0;  --pending:#F6B23D;  --done:#6E8CFF;  --idle:#5A6273;  --error:#FF5D79;
--core:#8FB2FF;  --link:#39D9E0;                            /* core điều phối / đường nối */
```

**Typography (3 vai):**
- Display/heading: **Space Grotesk** — có cá tính, dùng tiết chế.
- Body/UI: **Inter**.
- Data/mono: **JetBrains Mono** — cho MỌI dữ liệu máy: tên `provider:account`, token, cost, PID, timestamp, log.

**Voice/copy (tiếng Việt, giọng sản phẩm sagent):**
- Nhãn khung tiếng Việt ("Hạm đội", "Nhật ký sự kiện", "Tổng quan"); status giữ từ repo đang dùng (`running`/`pending`/`done`, thêm `idle`/`error`).
- Nút nói đúng việc: "Dừng", "Dừng tất cả" (không phải "Submit"). Một hành động giữ một tên xuyên suốt.
- Trạng thái lỗi = chỉ dẫn, không xin lỗi: `503 · endpoint không có model — Thử lại`.

## Ý tưởng xuyên suốt (giữ 2 surface là MỘT sản phẩm)

**"Agent = ánh sáng sống."** 3D: orb phát sáng quay quanh core, có hạt dữ liệu chảy dọc beam. 2D: đúng ánh sáng đó thành quầng status + đường pulse trên card. **Cùng một bộ token** → chuyển 2D↔3D thấy liền mạch.

## Pattern view 2D

- **Bố cục bento:** Hạm đội (lưới card, `auto-fill minmax(238px,1fr)`) chiếm chính · cột phải: Tổng quan (meter) · dưới cùng: Nhật ký full-width.
- **Card agent** (giải phẫu): rail status bên trái (2px, glow theo `--c`) + quầng aura góc trên; hàng 1 = glyph provider + `provider:account` (mono, `acc` tô màu status) + pill status (chấm pulse); dòng task (verb mono + nội dung); sparkline pulse (chỉ agent `run` mới "sống"); meta 3 ô: tokens / cost / elapsed (mono); nút "Dừng" hiện khi hover, chỉ với `run`/`pending`.
- **Meter** token/cost animate bằng transition width; pulse chấm status: `run` nhanh (~1.05s), `pending` chậm (~1.7s).
- **Event log** mono, color-coded theo loại (`edit`/`review`/`queued`/`done`/`error`/`dispatch`), auto-scroll, cap ~60 dòng, dòng mới fade-in.

## Pattern view 3D

- **Bố cục:** core điều phối (icosahedron wireframe + halo) ở giữa; orb agent **xếp đều trên một ring** quanh core (KHÔNG thả trôi ngẫu nhiên — thả trôi là lỗi trực quan số 1 của bản cũ).
- **Orb:** sphere màu status (MeshBasicMaterial, unlit nên tự "phát") + torus ring + **glow sprite additive** phía sau. Pulse scale/opacity theo status (`run` tần số cao hơn).
- **Beam + hạt dữ liệu:** line mờ core→orb; **particle** (additive sprite) lerp dọc beam, màu theo status → tín hiệu "đang chạy thật". Particle chỉ chạy ở `run`/`pending`.
- **Chiều sâu:** `FogExp2` + `GridHelper` mờ dần. Nền = `--void`.
- **Label = HTML overlay**, chiếu toạ độ world→screen mỗi frame (`vector.project(camera)`), ẩn khi ở sau camera. → chữ nét căng, KHÔNG dùng 3D text (mờ).
- **Camera:** orbit tự viết (kéo xoay azimuth/polar, lăn zoom, tự xoay nhẹ khi idle). Clamp polar để không lật.
- **Tương tác:** raycaster click orb → dừng đúng phiên đó (đồng bộ với 2D và với hành vi `bấm orb để dừng` trong README).

## Anti-pattern (thấy là sửa)

- ❌ Thả agent trôi tự do trong không gian → khó đọc. Dùng ring/lưới có trật tự.
- ❌ Dùng addon HIỆU ỨNG three.js (OrbitControls/EffectComposer/BloomPass). Loader đã vendor thì được — xem ràng buộc 3.
- ❌ Load asset (three.js/font/icon) từ CDN lúc runtime.
- ❌ Rải màu ra chrome; màu chỉ cho status.
- ❌ 3D text cho label; dùng HTML overlay.
- ❌ Thêm animation không mã hoá thông tin (motion phải nói "đang chạy", "vừa xong"…), thừa là thấy giả/AI-made.
  - **Ngoại lệ DUY NHẤT** (chốt 20/08): nhịp thở/cử động nhỏ của nhân vật lúc đứng yên, để cảnh không chết cứng. Ngoại lệ này dừng ở đó — đi lại, vẫy tay, đổi phòng đều phải có việc thật đứng sau.
- ❌ Sans-serif cho số liệu; data luôn mono.

## Data contract (chỗ cắm feed thật)

Cả 2 surface đọc **một** mảng model. Thay nội dung mảng = 2D + 3D tự cập nhật.

```js
// một agent:
{ id, prov, acc, st,              // st ∈ run|pending|done|idle|error
  task:{verb, txt}, tok, cost, el } // el = elapsed
```

Nối vào state thật của sagent qua WebSocket/SSE (đúng field trong DTO allowlist). Đừng để mỗi surface tự fetch — một nguồn, hai cách vẽ.

## Sàn chất lượng (bắt buộc mọi thay đổi UI)

- Responsive xuống mobile (grid tự xếp 1 cột).
- `prefers-reduced-motion` → tắt mọi animation ambient (particle, pulse, auto-orbit).
- `:focus-visible` rõ; thao tác được bằng bàn phím.
- View 2D không được phụ thuộc 3D: nếu three.js lỗi, 2D vẫn chạy full (2D là DOM/CSS thuần).

## Checklist trước khi commit thay đổi giao diện

- [ ] Không có URL ngoài nào load lúc runtime (three.js/font/icon đã vendor).
- [ ] Không import addon hiệu ứng; loader (nếu có) đã vendor vào binary.
- [ ] Màu mới (nếu có) nằm trong token status; chrome vẫn đơn sắc.
- [ ] 2D và 3D đọc chung một model; đổi data thấy đồng bộ.
- [ ] Số liệu từ state thật, trong DTO allowlist, không lộ secret.
- [ ] reduced-motion + focus-visible + responsive đã kiểm.
- [ ] Nếu three.js lỗi, 2D vẫn hiển thị đầy đủ.
