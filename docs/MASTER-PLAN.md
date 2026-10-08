# Switch-Agent-Pro — Master Plan (hợp nhất)

> ## Quyết định 2026-08-17 — CHỈ WINDOWS
>
> Nhánh Linux **đã bị bỏ**. Mọi dòng nhắc tới Linux bên dưới là **lịch sử**, giữ lại để
> hiểu vì sao từng thiết kế như vậy; khối này đè lên tất cả.
>
> Lý do, không phải cảm tính: mọi thứ khiến công cụ này đáng dùng đều là chi tiết
> Windows — junction thay symlink, ACL thay bit quyền (`0o600` ở đó không bảo vệ gì),
> `taskkill` thay process group, tên thiết bị `NUL`/`COM1`, chuyện Windows lặng lẽ cắt
> dấu chấm cuối tên thư mục. **Cả 5 lỗi thật tìm được ở Pha 7 đều là lỗi Windows.**
> Giữ một nhánh Linux không có máy để chạy thì đó không phải hỗ trợ, đó là lời hứa
> suông — đúng thứ `docs/DO-LUONG.md` lập ra để chống.
>
> Kéo theo: `*_linux.go` xoá, `install/cai-dat.sh` xoá, CI chỉ còn `windows-latest`,
> build cho `GOOS` khác dừng ngay với thông điệp đọc được.

> Phiên bản: 2.0 · Cập nhật: 2026-08-17
> Tài liệu này **hợp nhất** hai nguồn thành một lộ trình duy nhất:
> - `CCSWITCH_CLAUDE_DEVELOPMENT_PLAN.md` (v1.1) — kiến trúc control plane, hai đường
>   subscription/API, daemon + SQLite, ACP, workflow DAG, security, học từ OSS.
> - `docs/PLAN.md` + `docs/THIET-KE.md` (của tôi) — phong cách DoD + bẫy + "đã đo",
>   và phần **đã build thật** (lõi Go Pha 1 chạy trên Windows).
>
> Nó thay thế `docs/PLAN.md` làm lộ trình chính. `docs/THIET-KE.md` giữ vai trò
> "vì sao"; `docs/DO-LUONG.md` giữ vai trò báo cáo đo.

---

## 0. Danh tính dự án

- **Tên**: **Switch-Agent-Pro** (đổi từ `ccswitch` — trùng `farion1231/cc-switch`).
- **Module Go**: `github.com/trantiendevweb/switch-agent-pro`
- **CLI**: `sagent` · **Daemon**: `sagentd` · **Config project**: `.sagent/project.toml`
- **Một câu**: *local-first control plane điều phối nhiều coding agent và nhiều AI API,
  chạy native trên Windows, một binary, có dashboard quan sát realtime.*

Đích không phải "mở nhiều terminal", mà là: dùng **cả subscription profile lẫn API
profile**, gắn vào **agent harness** hoặc **model route** phù hợp, chạy trong
**workspace biệt lập của từng project**, điều phối bằng **flow khai báo được**, lưu
**trạng thái bền vững**, **quan sát realtime**, và **điều khiển được từ bốn mặt**
(terminal → 2D → workflow board → 3D) — xem mục 2c.

### Cam kết mã nguồn mở

Đây là tiêu chí sản phẩm, không phải câu khẩu hiệu:

- **Giấy phép MIT**, toàn bộ mã nằm trong repo — không có phần lõi đóng.
- **Không telemetry, không tài khoản, không dịch vụ đám mây bắt buộc.** Clone về là
  chạy được offline; thứ duy nhất ra Internet là chính CLI/API của nhà cung cấp AI
  mà bạn cấu hình.
- **Mọi phụ thuộc phải là mã nguồn mở, giấy phép tương thích**, ghi trong
  `docs/OPEN_SOURCE_LEDGER.md`. Ưu tiên stdlib; thêm dependency phải có lý do.
- **Không có tính năng nào bị khoá sau bản trả phí** — không có "bản pro".
- Dashboard **tự phục vụ tại máy bạn**; không gửi state đi đâu.

---

## 1. Sáu nguyên tắc (linh hồn, không đổi)

1. **Đã đo — không suy luận.** Vị trí token, biến env, refresh, session resume,
   trạng thái CLI: chưa có thí nghiệm tái lập thì chưa được coi là đúng.
2. **Whitelist — không blacklist.** Chỉ chia sẻ file/khoá config đã biết là an toàn.
3. **Xoá an toàn.** Không bao giờ xoá credential/dữ liệu/project gốc vì một session bị xoá.
4. **Ghi nguyên tử.** State quan trọng: temp + fsync + rename, hoặc transaction DB.
5. **Local-first, một binary.** Lõi chạy độc lập trên Windows; dashboard chỉ là client.
6. **Trung thực về năng lực.** Mỗi provider/harness gắn `stable` / `experimental` /
   `unsupported` / `unknown` dựa trên **bằng chứng**, không "ước chừng".

---

## 2. Hai đường sử dụng (khác biệt cốt lõi so với plan cũ của tôi)

1. **Subscription path** — chạy Claude Code, Codex CLI, Gemini CLI, Cursor… bằng
   credential/config do CLI chính thức sở hữu (đây là toàn bộ phạm vi v1).
2. **API path** — gọi thẳng Anthropic, OpenAI, Google Gemini, xAI/Grok, DeepSeek,
   OpenRouter, Mistral, Groq, Ollama/LocalAI hoặc endpoint OpenAI-compatible.

Hai đường **dùng chung** Project, Task, Workspace, Flow, Scheduler, Event, Dashboard.
Chúng chỉ khác ở **auth, protocol, cách agent/model được thực thi**.

> Bài học: plan cũ của tôi gộp mọi thứ vào một "provider". Master plan này **tách**:
> harness ≠ AI provider ≠ model ≠ auth profile ≠ route.

---

## 2b. Quyết định kiến trúc 2026-08-17 — **đường gọn**

Bản v1.1 vẽ kiến trúc của một *service chạy 24/7*. Dự án này là **CLI người ta
clone về chạy trên máy mình**. Cân lại từng món theo tiêu chí "nó mua được gì":

| Món | Quyết định | Lý do |
|---|---|---|
| `internal/domain` (types thuần + state machine) | **BỎ** | Với cỡ codebase này chỉ là thủ tục rườm rà; các package hiện có (`provider`/`profile`/`link`/`jsonutil`) đã tách logic khỏi I/O đủ sạch |
| **SQLite** làm nơi giữ state | **GIỮ** | Nhiều tiến trình `sagent` cùng ghi (fleet ở terminal này, `status` ở terminal kia). JSON thì phải tự lo khoá; SQLite lo sẵn bằng transaction + WAL. Dùng `modernc.org/sqlite` **thuần Go** nên vẫn một binary, không cần cgo |
| Daemon `sagentd` | **BỎ khỏi đường chính** | Phiên do `fleet` sinh ra chạy nền độc lập rồi; không cần tiến trình canh. `sagent dash` sẽ bật server **tạm**, chỉ sống khi dashboard đang mở |
| Tách `harness` ≠ `AIProvider` ≠ `auth profile` ≠ `route` | **GIỮ** | Không phải ceremony — không tách thì lúc thêm Codex hoặc đường API phải viết lại. Giữ ở mức **interface**, không cần package `domain` riêng |
| `verify` + nhãn stable/experimental | **GIỮ** | Đây là thứ làm công cụ đáng tin |

**Khi nào xét lại:** cần daemon nếu có tính năng *flow chạy dài phải sống sót
qua reboot* hoặc *dashboard cần push realtime khi không ai mở terminal*. Chưa có
thì không làm trước.

---

## 2c. Bốn mặt điều khiển (control surfaces)

Hệ thống phải **điều khiển được AI từ bốn mặt**, và cả bốn đều **dùng thật được**
lẫn **cấu hình được** — không có mặt nào chỉ để ngắm:

| Mặt | Dùng khi | Điều khiển được gì | Công nghệ |
|---|---|---|---|
| **1 · Terminal** (CLI + TUI) | Ở trong terminal, SSH, script hoá, CI | Toàn bộ. CLI là mặt bằng đầy đủ nhất | Go stdlib; TUI vẽ tay, không kéo framework nặng |
| **2 · Dashboard 2D** | Muốn nhìn nhanh: phiên nào chạy, hạn mức, log | Xem · bật/dừng phiên · duyệt approval · đọc log | Web cục bộ, HTML/CSS/JS tĩnh nhúng bằng Go `embed` |
| **3 · Workflow board** ✅ | Dựng và chạy flow nhiều bước | Chạy flow, xem từng bước, **duyệt/từ chối**; dựng flow bằng kéo-nối còn để sau | Cùng web app với mặt 2, một tab khác |
| **4 · 3D** | Nhìn toàn cảnh đội agent, trình diễn | Cùng tập hành động với mặt 2, thể hiện bằng không gian | React Three Fiber |

### Ba luật giữ cho bốn mặt không vỡ

Bốn mặt × N tính năng là công thức phình bảo trì. Ba luật sau là thứ giữ nó sống:

1. **Một hợp đồng duy nhất.** Mọi hành động đi qua **API lõi có version**
   (`internal/api`). CLI **không** phải là lõi — CLI chỉ là *client đầu tiên*.
   Mọi mặt khác là client ngang hàng. Không mặt nào được gọi thẳng vào `store`
   hay `profile`.
2. **Ngang quyền (capability parity).** Một tính năng chưa xong nếu **chưa làm
   được từ CLI**. UI được phép làm việc đó *dễ hơn*, không được là *cách duy nhất*.
   Kiểm bằng test: mỗi hành động của UI phải có lệnh CLI tương đương.
3. **Sự thật đến từ event, không phải từ đoán.** Cả bốn mặt cùng nghe **một
   luồng event** có schema/version. Cấm UI tự suy trạng thái bằng timer hay
   animation — trạng thái nào không có event thì không được hiển thị.

### Ba bất biến giao diện (UI Invariants — bắt buộc tuân thủ)

Nguồn từ mục "Ràng buộc" của dashboard (`.claude/skills/sagent-dashboard/SKILL.md`):

1. **INV-UI-1 (Offline tuyệt đối lúc runtime):** KHÔNG load `three.js` từ CDN, KHÔNG load font từ Google Fonts, KHÔNG gọi API ngoài để render. Mọi asset phải **vendor** (tải về nhét vào bundle nhúng qua Go `embed`). *(Bài học thật: bản prototype để three.js ở cdnjs → màn 3D trắng trơn trong môi trường thật).*
2. **INV-UI-2 (Vanilla, không Node build):** HTML/CSS/JS thuần. Không React/Vue/bundler. Toàn bộ assets giao diện nhúng trực tiếp qua Go `embed`.
3. **INV-UI-3 (three.js chỉ MỘT file core):** KHÔNG dùng addon `OrbitControls`, `EffectComposer`, `UnrealBloomPass` — chúng kéo theo nhiều file, dễ vỡ khi nhúng. → Camera orbit **tự viết tay** (drag xoay azimuth/polar, wheel zoom `minDistance 6`/`maxDistance 40`, clamp polar ≤ `Math.PI*0.49`, giữ damping + autoRotate khi không reduce-motion); "bloom" (quầng sáng) làm bằng **additive glow sprite** (sprite radial-gradient chồng lớp), không post-processing.

### Mặt nào cũng bật/tắt được

- Lõi chạy **không cần mặt nào cả** (headless, cho CI/script).
- 3D là **tuỳ chọn**: máy yếu hoặc `prefers-reduced-motion` thì rơi về 2D; tắt hẳn
  cũng không ảnh hưởng lõi.
- Web assets nhúng bằng `embed` nên vẫn **một binary**; không cần Node để chạy.

### Cấu hình theo từng dự án

Tầng cấu hình, dưới đè lên trên: **mặc định của công cụ → global
(`~/.ai-accounts/config.toml`) → project (`.sagent/project.toml`) → cờ dòng lệnh**.
Mỗi project tự khai báo: mặt mặc định, layout, cột nào hiện, flow nào ghim,
route AI nào dùng, giới hạn song song, hành động nào cần duyệt. Xem mục 8.

---

## 3. Kiến trúc đích

```
CLI (sagent) · Dashboard 2D/3D
        │  (chỉ dùng public API, không đọc secret)
Local daemon (sagentd) · versioned API · WebSocket/SSE
        │
DAG scheduler · durable events
        │
Project · Task · Workspace · Session
   ├── Agent Harness  (ACP/PTY)  ── Auth Profile (subscription/API)
   └── Model Route    (direct API) ─┘
```

### Domain object (tách bạch, không gộp)

`AgentHarness` · `AIProvider` · `Model` · `AuthProfile` (mode: subscription/oauth/
api_key/service_account/local) · `SubscriptionProfile` · `APIProfile` (key/base URL/
headers, secret lưu tách) · `HarnessAdapter` · `AIProviderAdapter` · `AgentDriver`
(start/prompt/cancel/resume/stream/permission) · `ModelClient` · `ModelRoute`
(provider+model+auth+fallback/health/cost) · `ProcessBackend` (Windows ConPTY / Linux
PTY; tmux/container tuỳ chọn) · `Project` · `Workspace` (dir/worktree/sandbox) · `Task`
· `Session` · `FlowDefinition` · `FlowRun` · `Artifact` · `Approval`.

```
Session = AuthProfile + (AgentHarness | ModelRoute) + Project + Workspace + Policy
FlowRun = DAG<Task/Action> + State + Events + Artifacts + Approvals
```

### 10 boundary interface

Harness Adapter · AI Provider Adapter · Agent Driver · Model Client · Route Engine ·
Process Backend · Workspace Backend · State Store · Event Bus · Workflow Node.

**Luật boundary bất khả xâm phạm:** harness/provider **không biết** dashboard;
dashboard **không đọc** token/API key; workflow chỉ tham chiếu `auth_profile_id` /
`route_id`, **không** thao tác secret trực tiếp.

### Capability thay vì suy đoán

Mỗi adapter/driver khai báo capability **có version**; capability không có bằng chứng
mặc định `false`/`unknown`. Ví dụ: `config_root_isolation`, `credential_safe_clone`,
`concurrent_refresh`, `auth_subscription`, `auth_api_key`, `headless_execution`,
`structured_events`, `session_resume`, `usage_reporting`, `rate_limit_reporting`,
`acp_transport`, `protocol_anthropic_messages`, `protocol_chat_completions`,
`streaming`, `tool_calling`, `structured_output`, `vision`, `reasoning`,
`model_discovery`, `health_check`…

---

## 4. Cấu trúc Go đích

> Cập nhật theo quyết định "đường gọn" ở mục 2b: **không** có `cmd/sagentd`,
> **không** có `internal/domain`. Dấu ✓ = đã có thật trong repo.

```
cmd/sagent/               ✓ CLI (một binary duy nhất)
internal/store/           ✓ SQLite: sessions + migration (nguồn sự thật)
internal/process/         ✓ IsAlive / Kill theo nền tảng
internal/fleet/           ✓ chạy N phiên song song
internal/profile/         ✓ create · link · clone · run · remove (xoá an toàn)
internal/link/            ✓ junction (Win) / symlink (Linux)
internal/jsonutil/        ✓ .claude.json: khoá trùng, ghi nguyên tử, whitelist
internal/harness/         # Claude Code / Codex / Gemini CLI / Cursor adapters
internal/provider/        # Anthropic / OpenAI / Gemini / xAI / DeepSeek… API adapters
internal/model/           # normalized request/response/capability
internal/routing/         # route · fallback · health · usage/cost
internal/auth/            # subscription/API profiles + secret references
internal/agent/           # ACP / PTY drivers + direct-model agents
internal/process/         # Windows ConPTY / Linux PTY backends
internal/project/         # discovery + .sagent/project.toml
internal/workspace/       # dir / git worktree / sandbox
internal/workflow/        # DAG validate / scheduler / nodes
internal/store/           # SQLite + migrations + repositories (SSOT)
internal/events/          # versioned event envelopes
internal/api/             # local HTTP/IPC + WebSocket/SSE
internal/security/        # redaction · path policy · credential handling
internal/testkit/         # fake provider/agent/clock/process
web/                      # React UI (chỉ dùng public API)
docs/{adr,research,knowledge}/
```

`domain` không import provider/database/HTTP/UI. Tránh phụ thuộc vòng.

---

## 5. Học từ mã nguồn mở (có kỷ luật)

Chu trình cho phần quan trọng/rủi ro cao: **Study → Pin → Extract → Evaluate →
Decide → Adapt → Verify → Compound**. Chỉ tạo hồ sơ đầy đủ (`docs/research/<topic>/`
với SOURCES/FINDINGS/DECISION/TEST-EVIDENCE) khi nghiên cứu lớn hoặc port mã trực tiếp.

**Giấy phép:** học nguyên lý từ mọi dự án; **chỉ** đưa mã trực tiếp vào sản phẩm khi
license tương thích + giữ attribution; AGPL/GPL chỉ để học hành vi trừ khi chủ động
chấp nhận nghĩa vụ. Mọi mã port trực tiếp ghi vào `docs/OPEN_SOURCE_LEDGER.md`.

**Bản đồ tham khảo** (chọn theo boundary, không theo số sao):
- **Nhóm A (lõi):** cc-switch (SQLite SSOT, atomic, live config), Agent Deck (Go
  fleet/worktree/fork-resume), multiclaude (daemon, IPC, recovery), Agent Client
  Protocol + ACP Go SDK (session/prompt/cancel/streaming/permission).
- **Nhóm B (orchestration):** Gas Town, Beads (dependency graph/durable memory),
  Compound Engineering, CCPM (PRD→epic→task, parallel).
- **Nhóm C (dashboard/sandbox):** Agent of Empires, OpenHands, CCManager (PTY không tmux).
- **Nhóm D (API gateway/routing):** LiteLLM, LocalAI, New API (AGPL — chỉ học), cc-switch proxy.

---

## 6. Trạng thái hiện tại & ánh xạ (2026-08-17)

| Đã có | Vị trí | Trong master plan |
|---|---|---|
| Lõi Go đổi tài khoản Claude (Windows) | `cmd/sagent`, `internal/{paths,provider,jsonutil,link,profile}` | **Phần của Pha 1** (vertical slice Claude subscription) — cần **tách domain** + **đổi tên** |
| Bỏ Python, khoá JSON trùng, ghi nguyên tử, xoá an toàn | `jsonutil`, `profile` + test | Giữ, đưa vào `store`/`harness` mới |
| link junction/symlink đa nền tảng | `internal/link` | Giữ, thành nền `workspace`/materialize |
| `running.json` + fleet prototype | ~~`internal/{registry,fleet}`~~ **đã gỡ** | Làm lại trên **SQLite SSOT + daemon** ở Pha 2 (PID chỉ là runtime attribute) |
| Design tokens + dashboard 3D + mascot | `design-system/switch-agent-pro/`, `index.html`, `plan.html` | **Nguyên mẫu Pha 6** — biến thành client của event API |
| Đo Windows/Claude (junction, token file, safe remove) | `docs/DO-LUONG.md` | Bằng chứng Pha 0 (Windows/Claude subscription) |

> **Nợ kỹ thuật đã biết:** (1) `provider` hiện gộp mọi thứ — phải tách harness/
> provider/model/auth/route. (2) `running.json` không phải durable state — chuyển SSOT
> sang SQLite. (3) Chưa có domain layer, daemon, API path, workflow. (4) Chưa có Linux.

---

## 7. Lộ trình (8 pha, mỗi pha ra 1 bản dùng được)

> **Bốn dấu, không phải hai** (soát lại 21/08/2026). Trước lượt soát này chỉ có
> `[x]` và `[ ]`, và hai dấu thì không đủ chỗ cho sự thật: một mục làm được 80%
> phải khai là `[ ]`, và một mục không làm được vì thiếu khoá API bên ngoài cũng
> khai là `[ ]` — y hệt một mục chưa ai đụng tới. Đếm ra 81 xong / 18 còn (82%),
> mà mở mã ra đọc thì phần lớn 18 mục kia đã làm rồi. **Kế hoạch nói sai về chính
> nó, theo chiều bi quan.**
>
> | Dấu | Nghĩa | Luật |
> |---|---|---|
> | `[x]` | XONG | phải dẫn được `file:dòng` hoặc lệnh chạy được. Không dẫn được thì **đừng tick** |
> | `[~]` | XONG MỘT PHẦN | phải ghi rõ **phần nào xong, phần nào chưa** |
> | `[ ]` | CHƯA LÀM | không ai đụng tới, và không có gì bên ngoài cản |
> | `[!]` | BỊ CHẶN | **làm được, nhưng thiếu thứ bên ngoài** (khoá API thật, phần mềm chưa cài) |
>
> Gộp `[ ]` với `[!]` là nói dối theo chiều bi quan: một đằng là nợ của dự án,
> một đằng là thứ dự án không tự gỡ được. Lượt soát 21/08 xếp 18 mục thành
> **7 `[x]` · 9 `[~]` · 2 `[ ]` · 0 `[!]`** — ô `[!]` rỗng vì thứ duy nhất còn
> bị chặn thật (OpenRouter/Ollama, thiếu key) nằm ở dòng `⬜` trong Trạng thái
> Pha 4 chứ không phải một ô tick. Cách tính phần trăm: `[x]`=1, `[~]`=0,5,
> `[ ]`=0, `[!]` **không vào mẫu số**. Bảng đếm ở cuối mục 7.

> **ĐIỂM MỘT DÒNG (soát lại 22/08/2026):**
> `92,0/99 = 93%` *(bảng cũ khai)* → `94,0/99 = 95%` *(đếm lại đúng bản `.md`
> trước lượt này — bảng cũ đã trôi 2 điểm)* → `98,0/100 = 98%` *(sau lượt đó)*
> → **`98,0/99 = 99%`** *(đối chiếu ba merge 22/08)*.
> Chi tiết hiện tại: `97 [x] · 2 [~] · 0 [ ] · 1 [!]`. Hai ô cấp cao đổi dấu:
> `Subscription` `[~]→[x]`, `API` `[~]→[!]`; route theo năng lực đóng nhưng ô
> `Engine flow` vẫn `[~]` vì phần giá chưa đủ dữ liệu. Bảng đầy đủ ở cuối mục 7.



### Bước 0 — Đổi tên  **100%** (4 xong)
- [x] `go.mod` module → `github.com/trantiendevweb/switch-agent-pro`; `cmd/ccswitch`
  → `cmd/sagent` (git mv); cập nhật mọi import; **build + vet + test xanh**; `sagent ds` chạy đúng.
- [x] Cập nhật `install/cai-dat.{ps1,sh}`, CI, `.gitignore` sang `sagent`.
- [x] Alias tương thích `tk`/`ccswitch` → `sagent`. ✅ **LÀM 22/08** — xem
  [`docs/BAO-CAO-CLI-NODE.md`](BAO-CAO-CLI-NODE.md) VIỆC 1. Nguồn sự thật duy
  nhất là `install/cai-dat.ps1:75-76`: `$TenAlias = @('tk','ccswitch')` và
  `$ShimNoiDung = '@"%~dp0sagent.exe" %*'`. Đo trước khi sửa 21/08:
  `grep -ni "alias\|ccswitch" install/` ra **0 dòng**; nay ra 9 dòng.
  - **Chọn shim `.cmd` (23 byte/alias), CỐ Ý bỏ hai cách kia** — bản sao `.exe`
    (+32 MB, đo thật 2 × 16.016.384 byte) và hard link đều **chạy binary CŨ sau
    khi nâng cấp, im lặng**, vì installer thay `sagent.exe` bằng một file mới
    (`Move-Item`). Shim trỏ **theo đường dẫn** nên luôn chạy đúng file đang nằm
    đó. Giá phải trả nói thẳng: thêm một tiến trình `cmd.exe` mỗi lần gọi, và
    Ctrl+C vào `tk dash` thì cmd hỏi "Terminate batch job (Y/N)?". **Cả hai đều
    nhìn thấy được; chạy nhầm phiên bản thì không.**
  - **`%*` chứ không `%1 %2 %3`** — chỉ `%*` giữ nguyên đuôi dòng lệnh, tức giữ
    được cả tham số có dấu cách lẫn tham số thứ mười trở đi.
  - **KHÔNG đọc `os.Args[0]`, có chủ ý**: đọc tên binary chỉ cần khi muốn cư xử
    KHÁC theo tên gọi; ở đây muốn ngược lại — giống hệt. Thêm một nhánh rẽ theo
    tên là thêm một đường cho hai lối gọi lệch nhau.
  - **Đã chạy thật một lần**: installer chạy vào một `USERPROFILE` tạm
    (`hometest-77c6750f`) để không đụng `sagent.exe` đang chạy; `User PATH` chụp
    trước và khôi phục nguyên trạng sau khi đo. Ra `✓ alias: tk` và
    `✓ alias: ccswitch`.
  - **Còn một nút, ghi ra để không ai tưởng đã đóng**: Pha 7 ghi "chưa có bản
    `ccswitch` thật để mở ra xem", nên **tương thích CỜ** của tên cũ vẫn là suy
    đoán. Cái đã đóng là *gõ tên nào cũng chạy đúng binary*, không phải *mọi cờ
    của v1 đều còn nghĩa cũ*.
- [x] Rà tên toàn repo: README viết lại cho Switch-Agent-Pro; bộ PowerShell v1 chuyển vào
  `legacy/v1-powershell/`; `design-system/switch-agent-pro/`; 3 trang HTML sạch tên cũ.

### Pha 0 — Đo giả định & lập hợp đồng  **100%** (6 xong · 1 bị chặn)
🎯 Chứng minh cơ chế của **cả hai đường** trước khi khoá interface.
- [x] Test harness không chứa credential trong repo; mọi output **redaction**.
  - **Xong — không credential trong repo**: giàn test chạy trên HOME giả
    (`homeGia(t)` trong `internal/aiapi/aiapi_test.go`) và khoá bịa
    (`sk-bi-mat-khong-duoc-lo`). Adapter GIẢ trong `internal/fleet/fleet_test.go:296`
    và `internal/profile/{clone_acl,ditru,profile}_test.go` khai `ChuaDo` toàn bộ,
    nên không test nào mượn được danh tính thật.
  - **Xong — key không rò ra thông điệp lỗi**: `internal/aiapi/aiapi_test.go:100`
    (`TestKeyDiDungChoVaKhongRoRaLoi`) khẳng định key tới đúng header
    `Authorization` nhưng KHÔNG có mặt trong `err.Error()` — thông điệp đó đi ra
    cả terminal lẫn dashboard.
  - **Xong — mặt web dùng allowlist chứ không phơi struct**: `internal/dash/server.go:665-667`
    dựng `profileDTO` bằng danh sách trường tường minh, token chỉ còn cờ `HasToken`.
  - ✅ **XONG 22/08 — TẦNG redaction chung**: `internal/redaction/redaction.go:263`
    (`Che`), **16 luật** trong MỘT bảng duy nhất (12 luật bí mật + 4 luật danh
    tính). Trước lượt này `grep -rni "redact" internal/ cmd/` ra **0 dòng**.
    Xem [`docs/BAO-CAO-REDACTION.md`](BAO-CAO-REDACTION.md).
    - **Đo trước, viết mã sau** — quét 22 file nhật ký thật / 6.797 dòng / 10,6 MB.
      Kết quả quan trọng nhất: **0 khoá hay token đang rò** (`sk-ant-`, `ghp_`,
      `AKIA`, `AIza`, JWT đủ ba đoạn, `Bearer <giá trị>` — tất cả đều 0 lần).
      **Cái ĐANG rò là danh tính**: tên tài khoản `Administrator` **2.060 lần**,
      email 179 lần (75 là địa chỉ thật). Trên đúng cái máy đang bị dò mật khẩu
      ~3,9 nghìn lần/giờ và có `dash` tự biết mình `exposed`, đó không phải chuyện nhỏ.
    - **Cắm ở CỬA RA chứ không ở đường ghi, có lý do kiến trúc**: `StartDetached`
      gán thẳng file handle cho tiến trình con rồi cha THOÁT — không còn ai đứng
      giữa dòng ghi. Hai cửa ra thật là `nhatky.Duoi` (người + dash) và
      `nhatky.BoDau` (máy → prompt bước sau); tầng che gắn vào **cả hai**, 20 dòng
      thêm / 0 dòng xoá. Đo lại sau khi sửa — **trên NỘI DUNG ĐI RA, không phải
      trên file đĩa**: 2.060 → **0**, email 179 → **0**,
      chữ ký thinking giữ nguyên **489**, dòng JSON hỏng **0/6.092**.
    - **Đường máy phân loại CỐ Ý không bị đụng** (`api.phanLoaiPhienChet` đọc byte
      gốc) — có test ghim `TestPhanLoaiPhienChetVanNhinThayByteGoc`. Cắm nhầm tầng
      che vào đó thì mọi phiên rơi về `lost`, tức trả lại đúng cái mù loà mà nhật
      ký sinh ra để chữa.
  - ✅ **XONG 22/08 — bài kiểm quét toàn repo tìm credential**:
    `internal/redaction/quet_repo_test.go:31` (`TestRepoKhongCoBiMat`) duyệt
    `git ls-files` — **đúng tập file sẽ đi vào commit** — quét 294 file văn bản,
    bỏ qua 16 (nhị phân/quá 8 MB) và **báo cả hai con số** để không có chuyện cắt
    ngầm mà đọc như đã phủ hết. **Không có danh sách miễn trừ**: bí mật giả trong
    test đều được GHÉP LÚC CHẠY nên bài kiểm không thấy chúng — mở miễn trừ thì
    lần rò thật đầu tiên cũng sẽ được cho vào đó, đúng lúc đang vội. Là test Go
    chứ không phải git hook (hook nằm trong `.git/`, không clone theo, tắt được
    bằng `--no-verify`). Kèm `quet_xungdot_test.go:34` chặn vạch xung đột git.
    Tính chất "repo sạch" nay là một **phép đo tự chạy**, không còn là kỷ luật.
- [x] **Subscription** (Claude ✓Windows, Codex, Gemini CLI, Cursor · Win+Linux): config
  root override có bao trùm config/session/auth? token ở file/env/keyring? file nào
  đọc/ghi lúc login/prompt/refresh/exit? **concurrent refresh** khi 2 process chung
  credential? copy token có tạo session hợp lệ, bao lâu? headless/JSON/stream/ACP/
  resume/cancel? state máy dùng chung ngoài config root?
  - **Xong cho NĂM harness** (claude · codex · cursor · antigravity · grok; Gemini
    CLI đã bỏ), bảng đo ở `docs/DO-LUONG.md:14-20`. Bốn câu đã có câu trả lời đo
    được cho cả năm: **biến tách** (`CLAUDE_CONFIG_DIR` / `CODEX_HOME` / `APPDATA`
    / `USERPROFILE`), **token nằm ở file hay keyring** (Antigravity là Windows
    Credential Manager, khoá cố định `gemini:antigravity` → một máy một tài khoản),
    **headless** và **kết quả có cấu trúc**.
  - **Xong — concurrent refresh**: `docs/DO-LUONG.md:2431` (Đ5, 21/08) chạy cuộc
    đua N-clone thật: một bản thắng, bản thua **tự xoá trắng token của mình**.
  - **Xong — token sống bao lâu**: `docs/DO-LUONG.md:2669` (Đ4) — Claude ~7,5 giờ
    (refresh tới 16/09), Codex ~6,5 ngày, Cursor đọc từ **refresh** token;
    Antigravity là **kết luận KHÔNG đọc được**, không phải khoảng trống.
  - **Xong — state máy dùng chung ngoài config root**: phân loại `~/.codex` thành
    danh tính vs **khoá ghi/SQLite** (`thread-writer-locks`, `*.sqlite*`) ở Pha 2.5.
  - **BỎ KHỎI PHẠM VI — ACP**, và đây là một **quyết định có phép đo**, không phải
    một ô bỏ trống. Dòng này từng ghi *"CHƯA — ACP"*, tức đếm một hướng **đã bị bác
    bỏ** thành nợ, và ai đọc nó sẽ đi làm đúng cái việc dự án đã cố ý không làm.
    Đo 18/08 (`docs/DO-LUONG.md`, mục *"Bỏ dò chuỗi, đọc dữ liệu có cấu trúc"*):
    **không CLI nào trên máy nói ACP** — `claude --help`, `codex --help`,
    `agy --help` đều không có. Nhưng cả ba đều có đường khác **sẵn hôm nay**:
    `claude --output-format stream-json --verbose`, `agy --output-format json`
    kèm `--json-schema`, `codex mcp-server`. Nguyên văn kết luận hôm đó:
    *"đích đến không phải ACP — mà là dữ liệu có cấu trúc, và nó có sẵn"*.
    Thứ ACP hứa (sự kiện có cấu trúc thay cho dò chuỗi) **đã đạt được bằng đường
    khác**: `permission_denials`, `subtype = error_max_turns`, `api_error_status`,
    `rate_limit_event.resetsAt` — bốn kiểu hỏng từng phải đoán bằng chuỗi nay đọc
    thẳng ra trường. Cùng lệ với `~~symlink Linux~~` và `Gemini CLI đã bỏ`.
  - ✅ **XONG 22/08 — resume/huỷ Ở TẦNG HARNESS** — commit `e234b45`:
    đối chiếu cờ của năm CLI; chạy nối lại thật với Codex, Cursor và Antigravity;
    đo huỷ giữa chừng cùng hiện vật còn lại. Claude được chốt bằng transcript trên
    đĩa vì cả hai tài khoản đều đang bận; báo cáo giữ nguyên ranh giới chưa chạy
    một lượt resume Claude thật. Xem [`docs/BAO-CAO-SUBSCRIPTION.md`](BAO-CAO-SUBSCRIPTION.md).
  - ✅ **XONG 22/08 — nhật ký chạm file theo pha** — commit `e234b45`: đã tách
    login/khởi động · prompt · exit cho Claude và ghi nhật ký tương ứng cho Codex,
    Cursor, Antigravity, Grok. Pha refresh Claude chưa ép chạy vì sẽ xoay token của
    phiên đang sống; đây là giới hạn được ghi rõ, không bị đọc thành số đo.
- [!] **API** (Anthropic, OpenAI, Gemini, xAI/Grok, DeepSeek, OpenAI-compatible): auth
  mode, base URL, model naming, headers; protocol (Responses/Chat Completions/Anthropic
  Messages/Gemini native); streaming/tool/reasoning/vision/structured-output/usage;
  error+rate-limit schema, retry headers, health, model discovery.
  - **Xong — 3/6 nhà cung cấp, 1/4 giao thức.** xAI/Grok · DeepSeek ·
    OpenAI-compatible generic đều đo thật qua `modelapi.vn` (`docs/DO-LUONG.md:2188`
    và `:2322`). Auth (Bearer), base URL tuỳ ý, model naming, header: `internal/aiapi/aiapi.go:168`.
  - **Xong — streaming + usage**: `internal/aiapi/stream.go:30` (`GoiStream`). Đo
    được đúng cái bẫy: endpoint tương thích OpenAI KHÔNG trả `usage` khi stream
    trừ khi hỏi bằng `stream_options.include_usage` — `internal/aiapi/stream_test.go:54`
    canh điều đó. Nhà cung cấp không trả thì bật `ThieuUsage` (`:105`) chứ không
    ghi 0 như thể miễn phí.
  - **Xong — error schema + health + model discovery**: lỗi giữ **nguyên văn** kèm
    request id (`stream_test.go:153`); `internal/aiapi/suckhoe.go:55` hỏi
    `GET /models` nên trả lời được cả "route sống không" lẫn "model khai có thật
    không" lẫn "nhà cung cấp liệt kê bao nhiêu model".
  - 🚫 **BỊ CHẶN — ba giao thức còn lại**: Anthropic Messages, Gemini native,
    OpenAI Responses. **Thiếu THỨ GÌ bên ngoài, nói rõ**: cả ba đòi **khoá thật
    của chính nhà cung cấp** (`api.anthropic.com`, `generativelanguage.googleapis.com`,
    `api.openai.com`). Máy này chỉ có khoá của **nhà bán lại** `modelapi.vn`, và
    `docs/DO-LUONG.md` đã đo rằng khoá đó trả **401 ở `api.deepseek.com`** — tức
    nó không đi được cả tới cổng gốc của nhà mà nó bán lại. Sau khi bốn năng lực
    phía dự án đã đóng, đây là lý do ô cha chuyển sang `[!]`: toàn bộ phần còn lại
    đều thiếu thứ bên ngoài, nên không còn đúng khi giữ nó ở `[~]`.
  - ✅ **XONG 22/08 — rate-limit schema và retry header**: trước lượt này
    `grep -rn "429\|Retry-After" internal/aiapi internal/api` ra **0 dòng**; nay
    có `internal/aiapi/cholai.go` (đọc `Retry-After` **cả hai dạng**: số giây và
    mốc HTTP-date) + `aiapi.go:109` (`BiChanTocDo`). Luật chọn: **chờ theo
    `Retry-After` rồi thử lại CHÍNH route đó, có trần; hết lượt mới để tầng trên
    đổi route.** Xem [`docs/BAO-CAO-429.md`](BAO-CAO-429.md).
    - **Vì sao không nhảy ngay** — ba dữ kiện của chính dự án này: (1) hai route
      `deepseek` và `grok` khai **cùng** `base_url = https://modelapi.vn/v1`, nên
      "route dự phòng" là cùng một cổng, cùng máy chủ; chặn theo tài khoản/IP thì
      lượt hai **chắc chắn cũng 429**. (2) 429 khác 5xx ở **bản chất**: 5xx nói
      "chỗ tôi đang hỏng", 429 nói "chỗ tôi vẫn tốt, anh đi nhanh quá" — kèm một
      **lời hứa có thời hạn**; nhảy ngay là vứt bỏ thứ duy nhất 429 cho không.
      (3) Với 5xx thì nhảy ĐÚNG, và đã đo: thân 503 là "No available channel for
      model grok-code-fast-1" — hỏng theo TỪNG MODEL. Nên bản sửa **cố ý không mở
      rộng sang 5xx**.
    - **Không đóng cửa đường kia**: hết lượt thử lại, lỗi vẫn là `*LoiAPI` với
      `Nguoi = false`, nên `internal/api` vẫn đổi route dự phòng theo luật cũ —
      không phải sửa một dòng nào ở tầng đó.
    - ⬜ **Nói thẳng lỗ lớn nhất: CHƯA ĐO ĐƯỢC MỘT CÚ 429 THẬT.** Đo miễn phí qua
      `GET /v1/models` cho thấy modelapi.vn **không gửi header hạn mức nào** ở
      phản hồi thành công (không `X-RateLimit-*`), tức **không có cách nào biết
      trước còn bao nhiêu lượt** — chỉ biết khi đã đâm vào tường. Nhà này chạy
      gateway `new-api` (`X-New-Api-Version`), và **chưa kiểm** new-api có gửi
      `Retry-After` kèm 429 hay không. Mã đường 429 hiện được canh bằng test, chưa
      bằng một phản hồi thật.
  - ✅ **ĐÃ ĐO — 4 ô nợ phía dự án, ĐÃ ĐÓNG CẢ 4**: trước đây dòng này là
    một khoảng trống; nay nó là một **kết luận có số**. Đo thật 22/08 bằng key
    thật, 14 phép đo (7 năng lực × 2 route, ~4.500 token) —
    [`docs/BAO-CAO-NANGLUC-API.md`](BAO-CAO-NANGLUC-API.md):
    **4 trên 7 năng lực bị chặn ở PHÍA DỰ ÁN, không phải ở nhà cung cấp.**
    - ✅ **`reasoning` — ĐÓNG 22/08.** `tinNhan` đọc `reasoning_content`, giá trị
      đi tới `KetQua.SuyLuan`, và có mặt đọc nó (cờ CLI + dashboard). Đo thật
      với deepseek: câu trả lời 91 ký tự, phần suy luận **477 ký tự** — phần
      từng bị vứt dài **gấp 5,2 lần** phần giữ lại, mà người dùng đã trả tiền
      cho cả hai.
    - ✅ **`goi-tool` — ĐÓNG 22/08.** `aiapi` gửi được định nghĩa tool và đọc lại
      `tool_calls`. Cố ý **KHÔNG** chạy tool hộ người gọi: làm vậy là biến thư
      viện thành nửa vòng lặp agent, phá lời hứa "mọi lời gọi đều nói ra nó tiêu
      gì", và mở một đường chạy lệnh thứ hai không qua bảng quyền plugin. Có bài
      kiểm **đếm số lượt chạm mạng và đỏ nếu khác 1**.
      Xem [`docs/BAO-CAO-TOOLS-SUYLUAN.md`](BAO-CAO-TOOLS-SUYLUAN.md).
    - ✅ **`dau-vao-anh` — ĐÓNG 22/08, commit `5f922c1`.** `GoiKem` dựng phần
      `{type, image_url}`; grok đọc đúng ảnh PNG 32×32. Deepseek vẫn trả HTTP 400
      "This model does not support image", nay được phân loại đúng là vướng ở
      **nhà cung cấp**, không còn vướng phía dự án.
    - ✅ **`dau-ra-co-cau-truc` — ĐÓNG 22/08, commit `5f922c1`.** Yêu cầu gửi
      `response_format` với JSON Schema `strict`; grok trả JSON đúng schema.
      Deepseek vẫn trả HTTP 400 "This response_format type is unavailable now",
      cũng là giới hạn nhà cung cấp.

    Xem [`docs/BAO-CAO-ANH-JSON.md`](BAO-CAO-ANH-JSON.md). Nửa API không còn nợ
    trong repo; ba ô ✗ còn lại nằm ở phía nhà cung cấp.
- [x] Junction Windows ✓ / ~~symlink Linux~~ từ Go, không admin. Đo ở
  `docs/DO-LUONG.md:112-122`: `sagent them claude:smoketest` (không quyền quản
  trị) nối **17 mục dùng chung**, PowerShell xác nhận `ReparsePoint = True` cho
  mọi mục, riêng `.claude.json` là `False` (file riêng thật). `link.IsLink` qua
  `GetFileAttributes` nhận đúng. Nhánh Linux **bỏ khỏi phạm vi** theo quyết định
  "CHỈ WINDOWS" ở đầu tài liệu, nên ô này không còn nợ gì.
- [x] Behavior khi stream/process ngắt, reboot, config ghi dở — cả ba đều đã đo,
  và mỗi cái ra một kết luận khác nhau:
  - **stream ngắt**: `internal/aiapi/stream_test.go:131` (`TestManhHongKhongGietCaLuot`)
    — một mảnh SSE hỏng không giết cả lượt; `:192` — stream rỗng bị coi là HỎNG
    chứ không phải "trả lời rỗng".
  - **process ngắt**: `process.KillTree` (đo ở `docs/DO-LUONG.md` mục Pha 7) —
    `taskkill /T` bỏ sót đám con khi cha đã thoát; `status` đối chiếu PID thật rồi
    tự đánh dấu `lost`, `sagent quet` tìm tiến trình mồ côi và **mặc định chỉ báo**
    vì Windows dùng lại PID.
  - **reboot**: `docs/KHAC-PHUC-SU-CO.md` mục 2 — lượt chạy kẹt `running` vĩnh
    viễn vì `state.db` không kịp nhận tín hiệu. Trạng thái từng bước nằm ở SQLite
    nên `flow resume` chạy tiếp được. **Nói thẳng phần chưa đẹp**: không có khâu
    tự dọn lúc khởi động — người dùng phải gõ `sagent flow huy <#>`.
  - **config ghi dở**: `internal/jsonutil/jsonutil.go:28` `AtomicWrite` ghi file
    tạm rồi `Rename`; `Backup` (`:38`) sao lưu trước khi đè. Phía DB thì chặn hạ
    cấp schema + tự sao lưu bằng `VACUUM INTO` (Pha 7).
- [x] **Capability matrix** stable/experimental/unsupported/unknown (harness + API).
  - **Xong cho HARNESS, và sạch**: `internal/provider/nangluc.go:21-28` khai ba
    trạng thái `LamDuoc` / `KhongLamDuoc` / `ChuaDo`, có conformance test đối
    chiếu lời khai với hành vi thật (`internal/provider/nangluc_test.go:78,230`).
    Chạy 21/08: `sagent nang-luc --chua-do` → **"Không còn năng lực nào chưa đo."**
    — 5 provider × 9 năng lực, **0 ô `ChuaDo`**.
  - **Bốn trạng thái rút thành ba, CÓ CHỦ Ý**: `experimental` bị bỏ vì nó trộn hai
    câu khác nhau ("đã đo, hơi rung" và "chưa ai đo"). `KhongLamDuoc` là một **kết
    luận** chứ không phải khoảng trống — đó là chỗ ba trạng thái đáng giá hơn hai.
  - ✅ **XONG 22/08 — nửa API**: `internal/aiapi/nangluc.go` (617 dòng) +
    `donangluc.go` (493 dòng), lệnh `sagent nang-luc-api [--chua-do|--do]`,
    `GET /api/nang-luc-api`, và khối "Năng lực route API" trên mặt web. **7 năng
    lực × 2 route**, đo THẬT 22/08 bằng key thật (~4.500 token) — chứ không phải
    một bảng khai tay. Xem [`docs/BAO-CAO-NANGLUC-API.md`](BAO-CAO-NANGLUC-API.md).
    Câu "route này hỗ trợ tool/vision/reasoning không" nay có người trả lời.
  - **Ba trạng thái KHÔNG bị chép làm hai bản**: `aiapi.TrangThaiNangLuc` là
    **type alias** sang `provider.TrangThaiNangLuc` — `aiapi.LamDuoc` và
    `provider.LamDuoc` là **cùng một kiểu**. Hai bản sao của một quy ước là hai
    bản sẽ lệch nhau, và bản lệch bao giờ cũng lệch theo đúng một chiều: gộp "đã
    đo, KHÔNG" với "chưa ai đo" thành một chữ `false`.
  - **Chỗ nửa API khác nửa CLI, và đây là quyết định đáng giá nhất của nó**: mỗi ô
    trả lời **HAI câu** — *KHÁCH* (mã của DỰ ÁN NÀY có gửi/đọc được thứ đó không —
    đo bằng `reflect` trên chính các kiểu của gói, **miễn phí**) và *NCC* (nhà cung
    cấp có làm được không — đo bằng mạng, **tốn token**), cộng một cột `Cho` nói
    **vướng ở đâu** (`khach` / `nha-cung-cap` / `ca-hai` / `chua-ro` / `khong-vuong`).
    Trộn hai câu vào một ô là làm mất đúng thứ đáng giá nhất: modelapi.vn **gọi
    tool ngon lành**, nhưng `aiapi.yeuCau` không có trường `tools` — một ô "không
    làm được" trơ trọi sẽ đẩy người đọc đi đổi nhà cung cấp, sai hẳn hướng.
    Phần KHÁCH đo bằng reflect chứ không phải bảng bool viết tay: bảng viết tay
    là một **lời khai** nữa, và lời khai thì mục ruỗng.
  - **Hai action chứ không một** (`api.nang-luc` đọc bảng · `api.nang-luc-do` tiêu
    tiền), và đường `/do` **đòi POST** — một GET tiêu tiền là một GET mà trình
    duyệt, bộ nạp trước hay một lần bấm F5 nhầm đều gọi lại được.
  - **Bốn trạng thái rút thành ba vẫn giữ nguyên ở cả hai nửa**: `experimental` bị
    bỏ vì nó trộn "đã đo, hơi rung" với "chưa ai đo" — `grep -rni "experimental"
    internal/` vẫn ra **0 dòng**, và đó là chủ ý chứ không phải thiếu sót.
- [x] **Threat model** cho subscription credential, API key, dashboard, command exec.
  ✅ **VIẾT XONG 21/08 — [`docs/MO-HINH-DE-DOA.md`](MO-HINH-DE-DOA.md)** (tên tiếng
  Việt, thay cho `docs/security/THREAT-MODEL.md` mà artifact cũ hứa).
  - **Có sườn, không phải bài luận**: tài sản → **sáu vị trí đứng của kẻ tấn công
    (K1–K6)** → mỗi mối đe doạ nói rõ *làm được gì · đứng ở đâu · hiện có gì chặn*,
    kèm `file:dòng` thật. Bốn nhãn trạng thái, **không có nhãn "có vẻ ổn"**:
    ✅ ĐÃ CÓ · 🟡 CÓ MỘT PHẦN · 🔴 KHÔNG CÓ GÌ · ⬜ CHƯA ĐO ĐƯỢC (kèm lý do).
  - **Đo trên máy THẬT, không phải máy dev sạch** (21/08 23:45): dash đang chạy
    `--host 0.0.0.0 --port 8788` (PID 16368, bind `::`), firewall `sagent-dash-8788`
    = **Allow**, `4625` **157 lượt/giờ** (sáng ~3.900, 14:20 là 1.867), rule chặn
    đang giữ **61 IP** (14:20 là 53). Kho: **4** `.credentials.json` trong `.clones`,
    **2** file `.key`, **90** worktree.
  - **Nói thẳng 11 chỗ đang hở**, xếp theo hậu quả. Ba dòng đầu: (1) mật khẩu dash
    là hàng rào **duy nhất VÀ cuối cùng** — sau nó không còn tầng nào, mà sau cửa đó
    là `POST /api/fleet` + `/api/flow/save` = **thực thi mã tuỳ ý dưới
    `Administrator`**; (2) agent chạy trong `~/.ai-accounts/.worktrees/…` tức **cách
    kho bí mật đúng ba cấp `..`**, không hộp cát; (3) **không ghi lại lần đăng nhập
    sai nào của dash** — đang bị dò cũng không biết, trong khi RDP thì đếm được
    tới từng lượt.
  - **Không đo được thì ghi là không đo được**: cổng 8788 chưa thử từ máy ngoài;
    chưa chạy phép đo "agent có thật sự đọc được `api-keys/*.key` từ worktree
    không" (tốn một lượt hạn mức thật); `lsass` vẫn **chưa có dump** nên
    brute-force vẫn là **nghi phạm**.
  - **Có lưới an toàn**: mục K liệt kê **18 bài kiểm** đang ghim từng bản vá được
    dẫn trong tài liệu — gỡ bản vá là đỏ, nên tài liệu không trôi thành văn bia.
- **Artifact:** `docs/research/phase0/*` (ENVIRONMENT, CLAUDE, CODEX, GEMINI, CURSOR,
  ANTHROPIC-API, OPENAI-API, …, CAPABILITY-MATRIX), **`docs/MO-HINH-DE-DOA.md`** (thay
  cho `docs/security/THREAT-MODEL.md` — cùng nội dung, tên tiếng Việt như mọi tài liệu
  khác của dự án), `docs/adr/0001-domain-boundaries.md`, `docs/OPEN_SOURCE_LEDGER.md`.
- **DoD:** mỗi kết luận có command/OS/output-redacted; **không token thật** ở đâu;
  capability chưa đo = `unknown`; interface nháp suy ra từ **≥2 harness và ≥2 API protocol**.
- ⚠ **Blocker cần bạn:** **API key** thật (local-only, redaction) cho phần API path.
  (Blocker "máy/VM Linux" đã bỏ cùng nhánh Linux — xem khối quyết định đầu tài liệu.)
- **Trạng thái (soát lại 21/08/2026):** pha này từng ghi **0/7** trong khi phần
  lớn nội dung của nó đã làm xong từ lâu — cả `docs/DO-LUONG.md` (3.400+ dòng số
  đo) lẫn `docs/SO-NO-DO-LUONG.md` (sổ nợ đo lường, nay chỉ còn đúng mục C2) đều
  là artifact của chính pha này. Đếm lại theo mã: **2 xong · 5 xong một phần ·
  0 chưa làm → 4,5/7 = 64%**. Ba thứ còn thiếu thật
  là: tầng redaction chung, bảng năng lực cho nửa API, và tài liệu threat model.
- **Cập nhật 21/08 tối:** ô **threat model** đóng — `docs/MO-HINH-DE-DOA.md`. Đếm lại:
  **3 xong · 4 xong một phần · 0 chưa làm → 5/7 = 71%**. Còn đúng **hai** thứ thiếu:
  tầng redaction chung và bảng năng lực cho nửa API.
  **Và ô này đóng bằng một kết luận khó chịu, ghi ra đây để không ai đọc lướt:**
  gom bốn mặt lại một chỗ mới thấy thứ mà từng mặt riêng lẻ giấu được — **mật khẩu
  dashboard và cờ `--tu-duyet-quyen` gặp nhau ở `POST /api/fleet`**, nên một mật khẩu
  6 ký tự trên một cổng đang phơi ra internet là **thực thi mã tuỳ ý dưới
  `Administrator`**. Không mặt nào trong bốn mặt tự nói ra được điều đó; **chỉ có cái
  sườn mới nói ra**. Đó là lý do ô này đáng làm chứ không phải thủ tục giấy tờ.
- **Cập nhật 22/08:** đóng nốt **hai** ô mà lượt soát 21/08 chỉ đích danh là còn
  thiếu — **tầng redaction chung** (`internal/redaction`, 16 luật, che LÚC ĐỌC:
  nội dung đi ra sạch, file trên đĩa vẫn nguyên văn — 2.060 lần lộ
  danh tính → 0) và **bảng năng lực cho nửa API** (`internal/aiapi/nangluc.go`,
  7 năng lực × 2 route, đo thật bằng key thật). Đếm lại: **5 xong · 2 xong một
  phần · 0 chưa làm → 6,0/7 = 86%**.
  **Cập nhật sau ba merge 22/08:** `Subscription` đóng hai câu cuối ở commit
  `e234b45`; ACP giữ nguyên quyết định **bỏ khỏi phạm vi**, nên ô chuyển `[~]→[x]`.
  `API` đóng vision + structured-output ở commit `5f922c1`; cùng tool + reasoning
  đã đóng trước đó, nửa API không còn nợ trong repo. Ba giao thức còn lại đều bị
  chặn bởi khoá gốc bên ngoài, nên ô chuyển `[~]→[!]`. Đếm hiện tại: **6 xong ·
  0 một phần · 0 chưa làm · 1 bị chặn → 6,0/6 = 100%**.

### Pha 1 — Storage + Claude slice + 1 API slice  **95%** (9 xong · 1 một phần)
🎯 Thay chức năng v1 bằng lõi có ranh giới rõ, storage an toàn, và **hai lát cắt dọc**.
- [x] ~~`internal/domain`~~ **bỏ** theo mục 2b — dùng thẳng các package hiện có.
- [x] `internal/store`: **Sổ đăng ký SQLite** (`profiles_so`, `routes_so`, migration v8) — lưu danh mục hồ sơ và route quản trị trong `state.db`, tuyệt đối **không có cột secret** (không lưu token/API key vào DB).
- [x] `jsonutil` (đã có) + ghi nguyên tử (`AtomicWriteJSON`) + bảo toàn file cấu hình, chống hỏng dở lúc ghi.
- [x] `link` abstraction (đã có) → tạo junction trên Windows để liên kết thư mục cấu hình cho agent.
- [x] **Đối chiếu 2 chiều sổ ↔ đĩa**: kiểm tra đồng bộ giữa hồ sơ trong SQLite và thư mục thực tế trên ổ đĩa (`~/.ai-accounts`), tự động phát hiện mục mồ côi hoặc thư mục chưa đăng ký.
- [x] **Xoá an toàn (Safe delete)**: chỉ cho phép xoá thư mục cấu hình do sổ đăng ký sở hữu; tái sử dụng `link.IsLink` để không bao giờ đi xuyên junction làm mất dữ liệu thư mục gốc.
- [x] **API lõi & CLI cho sổ**: bổ sung action mới vào `api.Actions` và hỗ trợ lệnh CLI `sagent profile list --so`, `sagent route list` để tra cứu danh sách từ sổ SQLite.
- [x] **Claude Harness Adapter chính thức**: bổ sung kiểu năng lực 3 trạng thái (làm được / không làm được / chưa đo) vào interface `provider.Adapter`, khai báo trung thực cho cả 5 adapter (Claude, Codex, Cursor, Antigravity, Grok), xây dựng bộ conformance test dùng chung chống khai báo sai lệch, và công bố báo cáo năng lực qua `api.Actions`, lệnh CLI `sagent capability` / `sagent nang-luc`, cùng endpoint/giao diện dashboard.
- [~] **1 direct-API vertical slice hoàn chỉnh** (kết nối trực tiếp Anthropic API, stream response, ghi nhận usage/error không lộ key). Lý do treo cũ ("client OpenAI-compatible cơ bản", "streaming cần xử lý cẩn trọng để không mất `usage`") **đã hết hạn từ 21/08** — xem `docs/DO-LUONG.md:2322`.
  - **Xong — lát cắt dọc ĐẦY ĐỦ cho giao thức OpenAI-compatible**: gọi thường (`internal/aiapi/aiapi.go:168`) và gọi stream (`internal/aiapi/stream.go:30`) đi CHUNG một đường nên cùng sổ, cùng luật fallback. Đo thật: deepseek 389 token/3,6s, grok 1926 token/31s.
  - **Xong — `usage` không mất**: hỏi bằng `stream_options.include_usage`, thiếu thì bật `ThieuUsage` chứ không ghi 0.
  - **Xong — error không lộ key**: lỗi giữ nguyên văn kèm request id; `internal/aiapi/aiapi_test.go:100` canh key không rò ra `err.Error()`.
  - **Xong — ghi nhận usage**: bảng `api_calls` (`internal/store/store.go:267`, migration v7), ghi cả lượt THÀNH và lượt BẠI (`internal/api/api.go:933`); cố ý không lưu prompt/câu trả lời.
  - **CHƯA — đúng chữ "Anthropic"**: `internal/aiapi` nói giao thức Chat Completions + `Bearer`, không phải Anthropic Messages (`x-api-key` + `anthropic-version` + sự kiện SSE khác hẳn). Phần này **bị chặn**: máy chỉ có khoá của nhà bán lại, đã đo là trả 401 ở endpoint gốc.
- [x] **Verb `verify` và `route test` đầy đủ**. Lý do treo cũ là "đang chờ tích hợp bộ kiểm tra kết nối mạng và khoá API thực tế" — **cả hai đã có**.
  - `sagent verify [provider]` — `cmd/sagent/main.go:402` (`cmdVerify`) → `internal/api/api.go:483` (`ProfileVerify`). Chạy 21/08 trên máy thật: 5 provider × (tìm thấy lệnh · thư mục base · nơi giữ token · **provider drift**), cộng ô kiểm **Windows ACL** của kho hồ sơ. Thoát ≠ 0 khi có mục đỏ — lượt chạy hôm nay bắt được Claude 2.1.234→2.1.235 và Antigravity 1.1.16→1.1.17, cảnh báo cố ý **không tự tắt** (phải `sagent verify --chap-nhan`).
  - `route test` = **`sagent route kiem`** — `cmd/sagent/route.go:96`, `internal/api/api.go:381` (`RouteKiem`), `internal/aiapi/suckhoe.go:55` (`Kiem`). Chạm mạng thật bằng khoá thật. Chạy 21/08: `✓ deepseek dùng được 173ms · ✓ grok dùng được 168ms`.
- **DoD:** CI Windows xanh; đổi Claude subscription không đăng nhập lại; API route
  stream + ghi usage/error không lộ key; xoá session không đụng credential/project gốc;
  fault injection không tạo JSON/DB dở; **hết Python**.
- **Trạng thái:** Đã hoàn thành sổ đăng ký SQLite (migration v8), đối chiếu 2 chiều sổ ↔ đĩa, xoá an toàn chỉ khi sổ sở hữu, lệnh CLI/API tương ứng (`profile list --so`, `route list`), và Claude Harness Adapter chính thức (kèm bảng năng lực 3 trạng thái, conformance test, báo cáo CLI/API/Dashboard). **Soát lại 21/08:** hai hạng mục "còn treo" đã hết treo — `verify` và `route kiem` chạy thật trên máy này (**xong**), lát cắt dọc API hoàn chỉnh đã có cho giao thức OpenAI-compatible (**xong một phần**, chỉ còn đúng chữ "Anthropic Messages" và nó bị chặn vì thiếu khoá). Điểm pha: **9,5/10 = 95%**.

### Pha 2 — Chạy song song + Project/Workspace  **100%** (22 xong)
🎯 Biến công cụ profile thành **runtime manager** đa project.
- [x] **SQLite là SSOT** (`~/.ai-accounts/state.db`); PID chỉ là thuộc tính
  runtime — `status` đối chiếu PID thật và tự đánh dấu `lost` phiên đã chết.
- [x] `clone` — chép credential ra N config dir riêng (mỗi bản `.claude.json`
  riêng nên **không đua ghi**); `fleet` — bật N phiên nền, log ra file;
  `status`; `stop <số|all>`; `clean` — xoá clone **an toàn** (không xuyên junction).
- [x] Cảnh báo thẳng: tiêu hạn mức gấp N, và hậu quả của việc chép token ra N chỗ.
- [x] **Xoay vòng refresh token: ĐÃ ĐO (20/08/2026)** — mỗi lần refresh cấp token
  mới và **giết token cũ ngay**. Nên chép token ra N chỗ **không cần** N tiến trình
  đua nhau mới hỏng: MỘT bản refresh là N−1 bản còn lại chết, hồ sơ gốc cũng nằm
  trong số đó. Chuỗi này đã làm mất phiên `claude:phu` giữa lượt chạy #47.
  **Đã bịt:** `profile.Clone` đồng bộ ngược trước khi chép đè (kiểm trên máy thật).
  **Còn lại một cách hỏng KHÔNG bịt được bằng đồng bộ:** hai bản đang chạy cùng
  lúc, một bản tới mốc refresh thì bản kia chết GIỮA CHỪNG — không có chỗ chen vào
  mà đồng bộ. `fleet` nói thẳng điều đó khi `--copies > 1` và khuyên chia việc cho
  nhiều TÀI KHOẢN thay vì nhiều bản của một tài khoản. Xem `docs/DO-LUONG.md`.
- [x] **Cảnh báo token sắp hết hạn** trước khi bật hạm đội — đã đo được Claude
  hết hạn sau ~7,5 giờ (Codex ~6,5 ngày), nên đội chạy dài chắc chắn vượt mốc.
  `TokenExpiry()` vào interface adapter, CHỈ đọc dấu thời gian.
- [x] **Mang token đã refresh từ bản clone về hồ sơ gốc** (`SyncBackTokens`):
  clone giữ bản token riêng nên refresh trong clone vốn bị mất trắng — đây là hệ
  quả thẳng của thiết kế, không phải phỏng đoán. So **nội dung** chứ không chỉ
  mtime (clone luôn có mtime mới hơn), có sao lưu trước khi đè.
- [x] Sửa bug thật: tài khoản di trú từ v1 không chạy được vì `Dir()` chỉ trỏ
  kho mới → thêm `ResolveDir()` dùng chung cho mọi verb.
- [x] **Workspace backend: git worktree** (`--worktree`) — mỗi phiên một cây làm
  việc + nhánh `sagent/<tên>-<n>`, đặt NGOÀI repo để `git status` không bị rác;
  `clean` gỡ worktree nhưng **giữ nhánh** (việc agent làm nằm trong đó). Không
  bật cờ thì công cụ **cảnh báo** các phiên dùng chung thư mục.
- [x] Migration có version cho SQLite (v1 bảng phiên → v2 cột `worktree`), chạy
  trong transaction; test khẳng định mở lại nhiều lần không hỏng.
- [x] Trả nợ test: `store` (migration, reaping PID chết, SetState) và `clone`
  (file riêng phải là bản sao thật, xoá clone không đụng dữ liệu gốc).
- [x] **Project discovery + `.sagent/project.toml`** — tầng cấu hình mặc định →
  global → project → cờ; `sagent init` / `sagent config`; `fleet` tôn trọng
  `project.workspace` và `policy.max_parallel_sessions` (trần cứng).
- [x] **Lá chắn dữ liệu**: `clean` từ chối gỡ worktree còn thay đổi chưa commit
  (phải `--force` mới bỏ) — trước đó `worktree remove --force` nuốt luôn việc
  agent làm dở, trái nguyên tắc #3.
- [x] Sửa bug thật: `clean` đoán số thứ tự worktree nên gặp khoảng trống là dừng,
  bỏ sót phần còn lại → chuyển sang **quét thư mục thật** (`FindAll`).
- [x] `docs/OPEN_SOURCE_LEDGER.md` — ghi 2 phụ thuộc trực tiếp + giấy phép.
- [x] **Integration test cho `fleet` + `workspace`** (29 test toàn dự án): dùng
  git thật và tiến trình con thật. Bắt được **rò file descriptor** trong
  `StartDetached` — tiến trình cha mở file log rồi không đóng, mỗi lần `fleet`
  rò một handle và trên Windows khoá luôn file. Có test hồi quy cho bug
  "đoán số thứ tự worktree".

> **Dọn ngày 20/08/2026:** danh sách dưới đây từng chép lại nguyên bản kế hoạch
> gốc, nên bốn mục ĐÃ XONG ở trên vẫn còn nằm đây dưới dạng `[ ]` — kế hoạch nói
> sai về chính nó, đúng thứ dự án này lấy "đo được" làm gốc để chống. Mỗi dòng
> dưới đây đã đối chiếu với mã nguồn trước khi đánh dấu.

- [x] ~~`sagentd` daemon~~ **bỏ** (xem mục 2b) — `dash` là server thay thế, đã chạy.
- [x] Project discovery + `.sagent/project.toml` — trùng mục đã xong ở trên.
- [x] Workspace backend: directory + **Git worktree** — trùng mục đã xong ở trên.
- [x] Process backend native Windows (`profile.StartDetached`). Nhánh Linux và
  tmux **bỏ** theo quyết định "CHỈ WINDOWS" ở đầu tài liệu.
- [x] Session state machine + event stream — `internal/events` (bus + SSE) và
  `Session.State` ba trạng thái đo được của Pha 5b.
- [x] Recovery khi process chết; cleanup orphan an toàn — `status` đối chiếu PID
  thật và tự đánh dấu `lost`; `sagent quet` (`SessionSweep`) tìm tiến trình mồ côi.
- [x] **Route engine — mảnh `health` ĐÃ CÓ (soát lại 21/08).** Chọn route ✅,
  fallback một lần ✅, usage/cost event ✅ (bảng `api_calls`, migration v7), và nay
  **hỏi được "route này còn sống không" TRƯỚC khi chạy**: `internal/aiapi/suckhoe.go:55`
  (`Kiem`) → `internal/api/api.go:381` (`RouteKiem`) → `cmd/sagent/route.go:96`
  (`sagent route kiem`), có endpoint web `/api/route/kiem` (`internal/dash/server.go:77`).
  Dòng cũ ở đây căn cứ vào `grep -ri health` không ra gì — mà mã đặt tên tiếng Việt
  (`SucKhoe`), nên **phép grep chứ không phải mã đã sai**; đó đúng là kiểu lệch im
  lặng mà mục này tồn tại để chống.
  Hai quyết định đáng ghi: (1) đi bằng `GET /models` nên **không tốn token** — một
  phép kiểm có tính tiền thì người ta thôi chạy, và health check không ai chạy thì
  bằng không có; (2) trả về `SucKhoe` chứ không phải một chữ "ổn", vì `Song` ("nhà
  cung cấp còn đó") khác `Dung()` ("gọi bây giờ thì chạy") — có thật một ca route
  sống mà `.sagent/project.toml` khai `deepseek-chat`, một tên không tồn tại ở nhà
  bán lại. Timeout 15s, ngắn hơn hẳn `Goi` (120s). **Vẫn nói rõ thứ nó KHÔNG trả
  lời được: hạn mức còn hay hết** — cái đó chỉ lộ khi gọi thật.
  Chạy 21/08: `✓ deepseek dùng được 173ms · ✓ grok dùng được 168ms`.
- **DoD:** 4 session cùng profile + 3 profile khác chạy đúng policy; 10 session đồng thời
  không hỏng config/state; restart daemon phục hồi đúng; ≥2 repo khác stack; không session
  nào ghi vào worktree session khác; chạy song song ≥1 subscription session và ≥1 API node.
- **Ánh xạ:** thay thế prototype `registry`/`fleet` hiện tại.

### Pha 2.5 — Codex + OpenAI-compatible (chống overfit Claude)  **100%** (8 xong)
- [x] **Đo Codex trên Windows** (`@openai/codex` 0.147.0) — xem `docs/DO-LUONG.md`.
  Phép đo quyết định: `CODEX_HOME` trỏ vào thư mục rỗng thì `codex login status`
  báo "Not logged in" dù `~/.codex` thật đang đăng nhập → **tách thật**.
  Token là FILE `auth.json`, không phải keyring.
- [x] **Adapter Codex** (`internal/provider/codex.go`): danh tính đọc từ claim
  `email` trong JWT `id_token` (giải mã cục bộ, không gọi mạng).
- [x] **Phân loại nội dung `~/.codex`** theo hai lý do khác nhau: danh tính
  (`auth.json`, `installation_id`, `cap_sid`) và **khoá ghi / SQLite**
  (`thread-writer-locks`, `tmp`, `.sandbox`, `*.sqlite*`) — nhóm sau nếu nối
  chung thì hai phiên song song sẽ giành nhau ghi và hỏng dữ liệu.
- [x] Chạy thật: `them/ds/dong-bo/xoa` cho `codex:*`; xoá an toàn có file mồi.
- [x] Sửa hai chỗ hardcode `.claude.json` rò vào lõi chung — giờ lấy theo
  `IdentitySource()` của từng adapter.
- [x] **`HeadlessArgs()` vào interface adapter** — sửa chỗ Claude rò vào lõi:
  trước đó `fleet`/`flow` hardcode `-p`, tức là chạy agent bằng Codex sẽ SAI mà
  không ai biết. Có test khẳng định mỗi provider tự khai kiểu chạy của mình và
  hai provider không được trùng cách.
- [x] **Chạy flow thật bằng Codex**: cùng hạ tầng clone/worktree/fleet/flow, chỉ
  khác adapter → đúng mục đích Pha 2.5 (chứng minh không đo ni theo Claude).
- [x] Đường API (OpenAI-compatible) — ~~chờ API key~~ **đã có key và đã chạy thật**.
  `internal/aiapi` + `sagent api <route>` (`cmd/sagent/main.go:81`): base URL tuỳ
  ý, khoá tham chiếu bằng `key_id`, lỗi giữ nguyên văn kèm request id, trả `usage`.
  Đo thật trên `modelapi.vn`: deepseek 127 token/2,3s và grok-4.5 1044 token/13,6s
  (`docs/DO-LUONG.md:2188`), stream 21/08 (`:2322`). Đúng mục đích Pha 2.5: cùng
  hạ tầng cho hai đường, thêm nhà cung cấp **không phải sửa mã nào**.

🎯 Chứng minh kiến trúc không bị đo ni theo Claude/Anthropic.
- **DoD:** Claude & Codex dùng chung domain/session API (subscription); Anthropic API &
  OpenAI-compatible dùng chung route API (direct); khác biệt auth/config nằm trong adapter,
  khác biệt tương tác nằm trong driver/model client; **conformance suite** chạy cho 2 harness
  + 2 API protocol; capability không hỗ trợ báo trung thực.

### Pha 3 — Flow DAG ghép được  **98%** (20 xong · 1 một phần)
🎯 Người dùng định nghĩa workflow mới **không sửa mã Go**.
- [~] Engine: DAG + cycle validation; input/output/artifact giữa step; condition/timeout/
  retry-backoff/cancel; concurrency limit (global/harness/provider/profile/project); route
  theo capability/model/giá/health/fallback; **approval gate**; **resume sau restart**;
  idempotency key; failure policy (stop/continue/fallback/compensate).
  **Đếm từng mảnh — 15 xong / 1 chưa đủ dữ liệu** (22/08: đóng artifact ·
  idempotency · compensate · route theo năng lực, và concurrency limit lên đủ
  5 nấc ở đường hạm đội). Dòng này từng
  ghi `[ ]` trong khi engine đã chạy thật cả ngày:
  - ✅ **DAG + cycle validation** — `internal/flow/flow.go:417` (`Order`, topo ổn
    định) và `:476-516` (dò chu trình, trả về **đúng vòng lặp** chứ không chỉ nói
    "có chu trình").
  - ✅ **input/output giữa step** — `{{steps.<id>.output}}`, lưu ở SQLite
    (migration v4) nên sống qua resume. Hai trần: lưu 32KB, nhét vào prompt 6KB,
    giữ phần CUỐI và **nói rõ đã cắt**.
  - ✅ **condition** — `internal/flow/when.go`, `when = "steps.kiem.output contains LOI"`.
    Cố ý KHÔNG nhúng ngôn ngữ biểu thức đầy đủ; sai cú pháp thì BÁO LỖI chứ không
    âm thầm coi là false.
  - ✅ **timeout** — `internal/flow/step.go:221,308` (`timeout_sec`).
  - ✅ **retry-backoff** — `step.go:286` (`tries := s.Retry + 1`) và `:348`
    (lùi dần 2s, 4s, 6s…).
  - ✅ **cancel** — `internal/flow/approve.go:65` (`Huy`) → `internal/api/api.go:1758`
    (`FlowCancel`) → `sagent flow huy <#>`.
  - ✅ **approval gate** — `Approve()` là hàm DUY NHẤT chuyển bước approve sang
    `done`; có test gọi `Resume` nhiều lần khi chưa duyệt và khẳng định bước sau
    KHÔNG chạy.
  - ✅ **resume sau restart** — bảng `flow_runs`/`flow_steps`, bước đã `done`
    không chạy lại.
  - ✅ **failure policy stop / continue / fallback** — `flow.go:58-60`.
  - ✅ **fallback + health ở tầng route** — `internal/api/api.go:794` (một lần
    chuyển, không lặp hết danh sách), `internal/aiapi/suckhoe.go:55`.
  - ✅ **concurrency limit — 5/5 nấc ở đường hạm đội, 22/08.** Khối
    `[policy.tran]` thêm **bốn trần CỘNG DỒN** (chung · harness · provider · hồ
    sơ), áp ở `api.xetTran` (`internal/api/api.go:1059`) → `FleetStart` (`:1118`),
    kiểu dữ liệu thuần ở `internal/fleet/tran.go:43`. Mặc định
    **chung 4 · harness 3 · provider 3 · hồ sơ 2** — muốn chạy đủ 4 phiên thì
    **buộc phải trải ra ít nhất hai tài khoản**, tức chính sự cố đẻ ra mảnh này
    đã bị chặn mà không ai phải sửa file nào (`TestMacDinhDaChanDonBonPhienVaoMotTaiKhoan`).
    Quy ước **số 0 = TẮT chiều đó**, giống hệt `max_parallel_sessions`; số âm bị
    từ chối ngay lúc đọc file kèm câu chỉ cách tắt — người gõ `-1` với ý "bỏ giới
    hạn" mà để lọt sẽ nhận đúng điều ngược lại. Xem
    [`docs/BAO-CAO-TRAN-DONG-THOI.md`](BAO-CAO-TRAN-DONG-THOI.md).
    **Nói thẳng phần CHƯA phủ**: bộ chạy flow vẫn chỉ nhận một con số —
    `internal/api/api.go:1677` truyền `MaxParallel: a.cfg.Policy.MaxParallelSessions`
    vào `flow.Runner`. Bốn trần mới canh cửa `FleetStart`, **không** canh số bước
    song song trong một lượt flow. Ô này xanh cho đường hạm đội, còn nợ ở đường flow.
  - ✅ **artifact giữa step, 22/08** — `internal/flow/artifact.go`. Bước trước khai
    `artifact = { ban-va = "ban-va.diff" }` và ghi vào `{{artifact_dir}}`; bước sau
    nhận **đường dẫn TUYỆT ĐỐI** qua `{{artifacts.ban-va}}`, mỗi lượt một thư mục
    `~/.ai-accounts/artifacts/run-<id>/<bước>/`. Hợp đồng: chạy xong mà **không có
    file** thì bước HỎNG (`ThieuArtifact`), và `doc_duoc` chặn được cả đường này —
    không thì hàng rào chặn bản tóm tắt mà để lọt bản đầy đủ.
    **Số đo dứt khoát** (lượt #53–#58, 22/08 11:53, **0 token**): bước sinh một
    file **60.094 byte / 1200 dòng**; qua artifact bước sau nhận **1200/1200 dòng**,
    qua `{{steps.x.output}}` chỉ nhận **651/1200** — **mất 45,75% ngay ở tầng LƯU**
    (`MaxStepOutput = 32.768`), trước cả tầng nhét (`MaxInject = 6.000`). Và phần
    mất là phần **cuối**, đúng chỗ mọi báo cáo để kết luận.
  - ✅ **idempotency key, 22/08** — `internal/flow/idempotent.go`, khai
    `idempotent = true`. Đo thật: lượt **#55** in *"bỏ qua — việc này lượt chạy
    #54 đã làm xong (idempotent)"*, **dẫn đích danh lượt cũ**, và artifact được
    **chép sang** `run-55/` nên bước sau vẫn đọc đủ 5.092 byte (kiểm bằng `ls`).
    Lượt **#56** đổi đúng một chỗ trong dòng lệnh (`1..400` → `1..401`) thì bước
    **chạy lại** — khoá bám **nội dung thật**, không bám id bước.
  - ✅ **failure policy `compensate`, 22/08** — `internal/flow/compensate.go`.
    Bước hỏng thì chạy một bước **GỠ LẠI rồi VẪN DỪNG**. Đo thật lượt **#58**:
    bước `dat-cho` tạo file "chỗ đã đặt", bước `hong` thoát 1, engine chạy `go-lai`,
    xác nhận `cho da dat ton tai=True` rồi xoá — kiểm trên đĩa sau đó: file
    **không còn**; lượt chạy kết thúc `failed`. Đúng: `compensate` **gỡ lại**,
    không phải **cứu**. Bước gỡ lại bị cấm khai `idempotent = true` (`:175`) —
    khoá sẽ trùng với lần gỡ trước và lần gỡ thứ hai sẽ bị bỏ qua im lặng.
  - ✅ **route theo capability — XONG 22/08, commit `2b437e4`.** `Step.Can` đi từ
    TOML qua `flow.Validate`/`api.VanDeCanTheoBang` tới `routeBridge.ChonRoute`;
    bộ chọn ưu tiên route đủ năng lực, giữ `ChuaDo` thành hạng riêng và loại route
    đã đo là không làm được. `flow validate` chặn sai trước khi gọi model. Xem
    [`docs/BAO-CAO-ROUTE-NANGLUC.md`](BAO-CAO-ROUTE-NANGLUC.md).
  - 🟡 **route theo giá — CHƯA ĐỦ DỮ LIỆU, không giả vờ đã làm.** Cùng commit
    `2b437e4` đo `api_calls`: 21 dòng nhưng **0 dòng có giá**, repo cũng có **0 bảng
    giá**. Cách dựng bảng giá khai tay đã cân nhắc rồi bỏ vì không có phép đối
    chiếu; ô `Engine flow` vì thế vẫn `[~]`.
- [x] Node built-in: `agent · model · route · shell · test · lint · review · approve · merge · notify`.
  ✅ **10/10 chạy được từ 22/08**, khai ở `internal/flow/flow.go:30-40` và bảng
  `implemented` (`:42-55`) — nguyên tắc trung thực năng lực: loại chưa chạy được
  thì **cảnh báo lúc `flow validate`**, không im lặng chấp nhận. Xem
  [`docs/BAO-CAO-CLI-NODE.md`](BAO-CAO-CLI-NODE.md) VIỆC 2.
  - ✅ `agent` · `shell` · `approve` · `notify` · `test` · `lint` · `review` · `model`
    (`model` bật 20/08 khi đường API đã đo thật). Node `plugin` là đường thứ ba,
    bật cùng ngày (xem ô plugin model bên dưới).
  - ✅ `route` — **tách thành node riêng** (`internal/flow/route.go`,
    `TypeRoute = "route"` ở `flow.go:40`). Trước: route là một **thuộc tính** của
    bước `model`. Nay: một node chọn đường sống rồi **chuyền TÊN cho bước sau**
    (`route = "{{steps.chon.output}}"`). **Vì sao đáng tách**: flow có năm bước
    `model` cùng đi một đường; đường đó chết thì mỗi bước tự hỏi lại và có thể rơi
    sang một đường dự phòng **khác nhau** — năm bước của cùng một lượt trả lời
    bằng năm mô hình khác nhau, mà bảng chỉ ghi "model".
    **Dùng lại, không làm lại**: `internal/flow` khai interface hẹp `RouteChon`,
    phần cắm ở `internal/api/routenode.go` gọi đúng `API.ThuTuRoute` và
    `aiapi.Kiem` (`GET /models`, **không tốn token**) — **không một dòng logic
    chọn đường nào được viết lại**. "Dùng được" = `SucKhoe.Dung()` (route sống
    **VÀ** model khai có thật), không hạ xuống thành `Song`.
    Hai quyết định phụ: **output là ĐÚNG cái tên, không gì khác** (một ký tự thừa
    là một đường không tồn tại; phần người đọc cần đi ra event bus, kể cả khi
    hỏng), và **hết đường thì DỪNG HẲN, không đoán bừa** (đoán bừa là bước sau
    đem tên ấy đi gọi thật — tốn thời gian, tốn tiền nếu trúng, và hỏng bằng một
    thông báo không liên quan tới nguyên nhân).
    **Mắt xích suýt đứt, ghi lại**: `Step.Route` trước đây đi **thẳng** vào lời gọi
    mà **không qua `Expand`**, nên node `route` sẽ chạy đúng mà **vô dụng**. Đã sửa,
    và `TestRouteChonDuongRoiChuyenChoBuocSau` khẳng định cả chuỗi.
  - ✅ `merge` — bật lên, **và đổi nghĩa công khai chứ không lặng lẽ bật cờ**
    (`internal/flow/merge.go`). Ghi chú cũ ở `flow.go:53` là "gộp nhánh git — hành
    động nguy hiểm", `implemented = false`, hẹn "còn chờ cơ chế merge an toàn".
    **Cơ chế ấy sẽ không bao giờ tới**: một node tự chạy `git merge` là một node
    có quyền viết đè lên cây mã của người khác, và không có cách nào làm việc đó
    an toàn bằng một dòng TOML. Muốn gộp nhánh thì đường cũ đúng hơn — `shell`
    chạy `git merge` đứng **sau** `approve`, nơi người duyệt nhìn thấy chính xác
    lệnh sắp chạy. Node này gộp **ĐẦU RA (chữ)** của N bước, theo **đúng thứ tự
    khai trong `needs`**, và "an toàn" ở đây nghĩa là **đo được**: hai lượt chạy
    giống hệt nhau cho ra đúng một khối chữ.
- [x] 3 flow mẫu: `fanout` (nhiều agent → review → chọn), `squad` (API planner → agent
  implementer → reviewer → test → approval), `agents` (danh sách task theo concurrency).
  Dựng sẵn trong binary: `internal/flow/builtin.go:10` (`fanout`), `:31` (`squad`),
  `:56` (`agents`) — dùng ngay không cần file. Đây là **bản trùng** của dòng
  "3 flow mẫu dựng sẵn" đã tick bên dưới; giữ cả hai thì kế hoạch tự nói sai về
  mình, nên tick luôn. Chạy 21/08: `sagent flow list` in **11 flow**, gồm cả ba mẫu.
- [x] Plugin model: TOML chỉ manifest/config tĩnh; logic động là **executable riêng** qua
  JSON-RPC/stdio versioned; secret trong TOML chỉ là reference; plugin chạy capability tối thiểu.
  Làm 22/08 ở `internal/plugin` (manifest · quyen · rpc · chay · bochay), plugin mẫu
  `cmd/sagent-plugin-mau`, node `plugin` trong `internal/flow`, lệnh `sagent plugin`
  và `/api/plugins`. Bốn ràng buộc, và chỗ ĐO được từng cái:
  - **TOML chỉ tĩnh** — lược đồ không có trường nào nhận biểu thức/script/lệnh, và
    bộ đọc **TỪ CHỐI KHOÁ LẠ** (`Doc` soi `md.Undecoded()`). Đây là phần đáng kể:
    thư viện TOML mặc định **bỏ qua** khoá không khai, nên `gia_tri = "sk-..."` hay
    `[hook] truoc_khi_chay = "..."` sẽ nằm im trong file — người viết tưởng nó chạy,
    người soi tưởng nó đã được xử lý. `TestManifestTuChoiKhoaLa` bắt cả ba ca đó.
  - **Executable riêng, JSON-RPC/stdio, có số phiên bản** — `GiaoThuc = 1`, bắt tay
    bằng `sagent.bat_tay` NGAY trong `Mo()`, trước khi có ai kịp gửi dữ liệu (kể cả
    secret) cho tiến trình con. Lệch số thì **dừng ở lượt bắt tay**, không phải nổ
    giữa một lượt flow đang chạy. Khung bản tin dùng CHUNG cho hai phía (`PhucVu`
    cho plugin, `Client` cho host), nên hai bên không có đường nào lệch nhau về khung.
  - **Secret chỉ là tham chiếu** — manifest chỉ có `key_id`; giá trị nằm ở
    `~/.ai-accounts/api-keys/<id>.key` và đọc qua đúng `aiapi.DocKey` (một bản duy
    nhất của luật "kho ở đâu, tên thế nào là hợp lệ"). Giá trị đi qua **stdio** lúc
    bắt tay — cố ý KHÔNG qua argv (mọi tiến trình trên máy đọc được dòng lệnh) và
    KHÔNG qua biến môi trường (con cháu của plugin thừa hưởng hết).
  - **Capability tối thiểu** — plugin không khai thì KHÔNG có. Đo bằng cách chạy
    **cùng một binary** hai lần, chỉ khác mấy dòng TOML, rồi so hai câu trả lời:
    `TestKhongKhaiThuMucThiKhongThayThuMucDuAn`,
    `TestKhongKhaiMoiTruongThiKhongThayBienCuaCha`,
    `TestKhongKhaiSecretThiKhongNhanDuocGiaTri`. Cả ba đã được thử ngược (bỏ hàng
    rào đi thì test đỏ), trừ ca secret — hàng rào thật của nó nằm ở tầng đọc
    manifest (`TestKhaiSecretMaKhongKhaiQuyenThiTuChoi`), nhánh trong `docSecret`
    chỉ là lớp thứ hai.
  Bảng quyền (`sagent plugin quyen`) có **HAI cột chứ không phải một**: plugin KHAI
  gì, và host **CHẶN được tới đâu** — ba trạng thái y như bảng năng lực provider.
  Gộp hai cột lại là chỗ mọi hệ thống quyền nói dối, nên **ba** chỗ này phải nói
  thẳng — 5 quyền: **2 `chan-that` · 2 `khong-chan-duoc` · 1 `chua-do`**
  (`internal/plugin/quyen.go:56-122`):
  - `thu-muc-lam-viec`: **KHÔNG CHẶN ĐƯỢC** — ⚠ **hạ cấp 22/08, dòng này trước đây
    khai `chan-that` và đó là một lời khai SAI**. Phép đo bác bỏ nó:
    `TestBangQuyenThuMucPhaiKhopVoiThucTeChamDuoc` chạy một plugin **không khai
    quyền nào**, host báo `thu-muc=(khong-cap)`, vậy mà plugin vẫn **đọc đúng 33
    byte** một file trong thư mục dự án, **ghi được** file mới và **liệt kê được**
    thư mục. Thứ host làm thật là **không CẤP đường dẫn** (`cmd.Dir` là thư mục tạm
    rỗng), không phải dựng hàng rào — plugin biết đường dẫn bằng cách khác (nhúng
    lúc build, đọc cấu hình riêng, hoặc đoán `C:\Users\...`) thì vào như thường.
    Muốn khai `chan-that` phải có ACL riêng từng lượt chạy, Job Object, hoặc
    AppContainer — chưa cái nào được làm. Số đo ở `docs/DO-LUONG.md` mục 22/08.
  - `ghi-thu-muc-lam-viec`: **KHÔNG CHẶN ĐƯỢC** (đã đo, là kết luận chứ không phải
    khoảng trống). Thấy được đường dẫn thì ghi được; chặn ghi đòi ACL riêng cho từng
    lần chạy, chưa làm. ⚠ Câu chốt cũ *"cách chặn thật đang có là không cấp
    `thu-muc-lam-viec`"* **đã bị chính phép đo trên bác bỏ** và đã gỡ khỏi mã.
  - `mang`: **CHƯA ĐO**. Host hiện KHÔNG chặn tiến trình con mở socket. Còn đường
    WFP/AppContainer trên Windows nhưng chưa ai thử, chưa có phép đo nào trên máy
    thật. **Không khai quyền `mang` KHÔNG có nghĩa là plugin bị chặn** — CLI in
    cảnh báo đúng câu đó, để không ai đọc bảng rồi yên tâm nhầm.
  - Hai ô `chan-that` còn lại (`moi-truong`, `secret`) đứng vững, nhưng ô `secret`
    kèm một câu phải đọc kỹ: `chan-that` ở đây nói **host không ĐƯA secret cho
    plugin không xin** — nó **không** nói plugin không lấy được secret. Plugin chạy
    cùng quyền hệ điều hành với host, nên plugin nào biết đường tới kho key vẫn đọc
    thẳng file được. Hai câu đó khác nhau, và người đọc lướt sẽ gộp chúng làm một.
  **Bảng quyền này đã ra tới mặt web 22/08** — xem ô tương ứng ở Pha 5b.
  Node `plugin` bật `implemented = true` vì đã chạy THẬT đầu-cuối
  (`TestFlowChayPluginThat`: build binary → `flow.Runner` thật → tiến trình con →
  kết quả chuyền sang bước sau), không phải vì đã viết xong mã. Nó là ĐƯỜNG THỨ BA
  bên cạnh agent và model API, và cắm vào đúng `internal/flow` để dùng lại DAG,
  retry, `on_failure`, `{{bien}}`, `doc_duoc`, `phai_co` — chứ không mọc thành hệ
  thứ hai bắt người dùng học hai bộ luật cho cùng một câu "bước này hỏng thì sao".
- [x] `internal/flow`: schema `flows.toml`, tầng đọc (mẫu dựng sẵn → global →
  dự án), **kiểm tra DAG** (chu trình, phụ thuộc ma, id trùng/xấu, type lạ),
  thứ tự chạy topo **ổn định**, `{{bien}}`.
- [x] 3 flow mẫu dựng sẵn: `fanout` · `squad` · `agents` — dùng ngay không cần file.
- [x] Trung thực năng lực: type đã thiết kế nhưng **chưa chạy được** thì CẢNH BÁO
  lúc kiểm tra (`model`/`test`/`lint`/`review`/`merge`), không im lặng chấp nhận.
- [x] `shell` chỉ nhận **argv** (`run = ["go","test"]`), cố ý không nhận chuỗi
  shell để khỏi mở đường injection.
- [x] Lệnh `sagent flow list | show <tên> | validate` (validate thoát ≠ 0 cho CI).
- [x] **Bộ thực thi** (`internal/flow/runner.go`): chạy theo thứ tự topo, timeout,
  retry lùi dần, `on_failure` stop/continue/fallback, biến `{{...}}`.
- [x] **Approval gate không thể bị bỏ qua** — `Approve()` là hàm DUY NHẤT chuyển
  bước approve sang `done`; bộ thực thi không có nhánh nào tự làm việc đó. Có
  test gọi `Resume` nhiều lần khi chưa duyệt và khẳng định bước sau KHÔNG chạy.
- [x] **Resume**: trạng thái từng bước nằm ở SQLite (bảng `flow_runs`/`flow_steps`),
  bước đã `done` không chạy lại — chạy tiếp được sau khi máy khởi động lại.
- [x] Lệnh: `flow run [--kho] | runs | approve | reject | resume | huy`.
- [x] Đã chạy thật: shell → approve → shell; duyệt thì đi tiếp, từ chối thì huỷ.
- [x] **Chạy song song nhiều bước**: vòng chạy theo ĐỢT — mỗi vòng tìm mọi bước
  đã sẵn sàng rồi chạy chúng cùng lúc, có trần lấy từ `policy.max_parallel_sessions`.
  Nhánh độc lập (test + lint + build) không còn xếp hàng. Approval gate vẫn nguyên:
  bước approve không bao giờ chạy trong đợt.
- [x] **Điều kiện `when`** — flow rẽ nhánh được:
  `when = "steps.kiem.output contains LOI"`. Toán tử: `== != contains not-contains
  empty not-empty > < >= <=`; đọc được `steps.<id>.state`, `steps.<id>.output`,
  `vars.<tên>`. Cố ý KHÔNG nhúng ngôn ngữ biểu thức đầy đủ — flow là file người ta
  gửi cho nhau được. Sai cú pháp thì BÁO LỖI chứ không âm thầm coi là false.
- [x] **`foreach` — một bước, nhiều lượt**: `foreach = "steps.liet-ke.output"`
  biến mỗi dòng của nguồn thành một lượt chạy, có `{{item}}` và `{{index}}`, các
  lượt chạy SONG SONG theo trần. Kết quả gộp có đánh dấu từng mục để bước sau
  phân biệt được. **Trần 50 mục**: nguồn thường là output của agent, lỡ in 5000
  dòng thì thành 5000 lượt gọi thật — vượt trần là DỪNG và báo, không âm thầm cắt.
- [x] **Huỷ bước cùng đợt khi có bước hỏng** (`on_failure=stop`): trước đó các
  bước song song vẫn chạy nốt dù flow sắp dừng — tốn hạn mức vô ích.
- [x] Node `test`/`lint`/`review` chạy được: test/lint lấy lệnh từ
  `commands.test`/`commands.lint` của `.sagent/project.toml` nên không phải lặp
  lại trong từng flow.
- [x] **Truyền dữ liệu giữa các bước**: `{{steps.<id>.output}}`. Kết quả lưu ở
  SQLite (migration v4) nên **sống sót qua resume**. shell lấy stdout+stderr,
  agent gộp log các phiên, notify lấy chính lời nhắn. Có hai trần: lưu 32KB,
  nhét vào prompt 6KB — giữ phần CUỐI (kết luận thường ở đó) và **nói rõ đã cắt**.
  Tham chiếu sai id thì giữ nguyên chuỗi để người viết thấy, không im lặng nuốt.
- [x] Workflow board (mặt 4) — **đã có bản vận hành**, xem chi tiết ở Pha 5c.
  `internal/dash/web/flow.html` (954 dòng): kéo node từ bảng trái vào canvas, nối
  cổng ra ↔ cổng vào (= `needs`), pan/zoom, cột phải sửa mọi thuộc tính, **chặn
  vòng lặp ngay trên bảng** trước khi gửi lên server. Lưu ghi thẳng vào
  `flows.toml` kèm toạ độ `x`/`y` (`internal/flow/save.go`) — bảng vẽ KHÔNG có kho
  riêng, nên flow dựng bằng giao diện và flow viết tay là MỘT thứ.
- **DoD:** thêm flow mới không rebuild binary; fake harness/API/agent chạy trong CI; flow
  đang chạy tiếp tục sau restart; test chứng minh **approval không thể bị bỏ qua**.

### Pha 4 — Mở rộng harness + AI API  ✅ xong (trừ OpenRouter/Ollama: chưa có key)
- Subscription: **Gemini CLI, Cursor**, OpenCode (nếu đo được).
- API: **Google Gemini, xAI/Grok, DeepSeek, OpenRouter, Mistral, Groq, Ollama/LocalAI**,
  generic OpenAI-compatible; Azure/Bedrock/Vertex ở lớp plugin/enterprise nếu cần.
- Mỗi tích hợp lặp: measurement → adapter → conformance → streaming/tool/error/usage → capability label.
- **DoD:** Grok & DeepSeek chạy qua API profile riêng, chọn model + stream; generic
  OpenAI-compatible hoạt động với custom base URL/model/headers; fallback không mất
  correlation ID/usage/error gốc; thêm model/provider chỉ khác endpoint bằng manifest;
  chưa xác minh giữ `experimental`/`unknown`.
- **Trạng thái (2026-08-18):**
  - ✅ **Năm harness chạy được**: claude · codex · cursor · antigravity · grok. Bảng đo
    đầy đủ (danh tính nằm ở đâu, tách được hay không) ở `docs/DO-LUONG.md`.
    Gemini CLI **bỏ** — Google cắt khỏi gói miễn phí cho cá nhân.
  - ✅ **Generic OpenAI-compatible qua API**: `internal/aiapi` + `sagent api <route>`.
    Custom base URL/model, key tham chiếu bằng `key_id`, lỗi giữ nguyên văn (còn
    request id), trả `usage`. Đo thật trên `modelapi.vn`: 1195 token / 11,8s.
  - ✅ **Streaming: XONG ở lõi và CLI** (21/08/2026) — `aiapi.GoiStream` +
    `sagent api <route> --stream`. Nỗi lo cũ có thật: endpoint tương thích OpenAI
    KHÔNG gửi `usage` khi stream trừ khi hỏi bằng `stream_options.include_usage`.
    Đã hỏi, và đo được usage đầy đủ: deepseek 389 token/3,6s, grok 1926 token/31s.
    Nhà cung cấp nào không trả thì `ThieuUsage` bật và nói rõ là CHƯA ĐO chứ không
    phải miễn phí. Đi chung đường với lời gọi thường nên cùng sổ, cùng luật
    fallback. **Mặt web cũng xong** (21/08): `/api/ai` nhận cờ `stream` và trả SSE
    trên CHÍNH endpoint đó — không mở đường riêng, vì đây vẫn là action `api.call`,
    chỉ khác cách gửi về. Lỗi đi TRONG luồng chứ không bằng mã HTTP (header 200 đã
    gửi trước khi biết kết quả; đóng ngang thì trình duyệt tự thử lại và trả tiền
    lần nữa). **Pha 4 xong.**
  - ✅ **Fallback route**: chỉ chuyển tiếp một lần sang route dự phòng; không fallback khi
    lỗi do người dùng (401/403/sai key/prompt rỗng); kết quả mang lỗi gốc nguyên văn +
    request id, ghi rõ tên hai route (đã thử và đã dùng), cùng usage của route thành công
    (nếu cả hai route hỏng thì trả cả hai lỗi).
  - ✅ **Lịch sử lời gọi API**: lưu vào bảng `api_calls` ở migration v7 trong `state.db`
    (thời điểm, route, model, token vào-ra, chi phí, thành-bại, lý do hỏng); không lưu
    prompt và câu trả lời để bảo đảm riêng tư.
  - ✅ **DeepSeek: ĐÃ ĐO** (20/08/2026, 22:41) — `sagent api deepseek` chạy thật qua
    `modelapi.vn`, model `deepseek-v4-flash`: vào 90, ra 37, tổng **127 token / 2,3s**.
    Cùng lượt đo `grok-4.5`: vào 214, ra 830, tổng **1044 token / 13,6s**. Đúng như
    dự đoán "chỉ là thêm route" — không phải sửa mã nào. Lưu ý đã đo: key này KHÔNG
    dùng được ở `api.deepseek.com` (401), chỉ dùng được ở nhà bán lại; và route này
    trả HTTP 503 ba lần lúc 16:54–16:56 cùng ngày rồi tự hồi phục.
  - ✅ **HTTP 429 + `Retry-After`: XONG 22/08** — `internal/aiapi/cholai.go`.
    Bị chặn tốc độ thì **chờ theo `Retry-After` rồi thử lại CHÍNH route đó, có
    trần**; hết lượt mới để `internal/api` đổi route dự phòng theo luật sẵn có
    (không sửa một dòng nào ở tầng đó). Đọc được **cả hai dạng** header: số giây
    và mốc HTTP-date. Mọi con số đều có trần — `Retry-After: 3600` mà chờ thật là
    cả lượt flow đứng im một tiếng, không log, không lối ra. Nhật ký `ChoLai`
    (`aiapi.go:79`) giữ lại mọi lần chờ, và khi hết lượt thì lỗi trả về **nguyên
    văn** thân 429 kèm request id. **Cố ý KHÔNG mở rộng sang 5xx**: luật cũ đã
    đúng cho 5xx (đo 20/08: thân 503 hỏng theo TỪNG MODEL nên đổi route cứu được
    thật), chỉ 429 mới cần chờ. Chi tiết và phần chưa chắc ở
    [`docs/BAO-CAO-429.md`](BAO-CAO-429.md).
  - ⬜ **429 THẬT thì chưa đo được** — modelapi.vn không gửi header hạn mức nào
    (`X-RateLimit-*` đều vắng ở phản hồi 200), nên **không biết trước còn bao
    nhiêu lượt**; và chưa kiểm gateway `new-api` mà nhà này dùng có gửi
    `Retry-After` kèm 429 hay không. Đường mã hiện được canh bằng test, chưa bằng
    một phản hồi thật.
  - 🚫 **OpenRouter/Ollama: BỊ CHẶN, thiếu thứ bên ngoài** — OpenRouter thiếu
    **khoá API thật** của openrouter.ai; Ollama thiếu **cả khoá lẫn phần mềm**
    (`ollama` chưa được cài trên máy này, nên không có endpoint cục bộ nào để
    trỏ tới). Cả hai đều "làm được, chỉ là thêm route" như DeepSeek đã chứng
    minh — không có nợ mã nào ở đây, chỉ có nợ thứ bên ngoài.

### Pha 5 — Bốn mặt điều khiển (làm theo thứ tự 5a → 5d)
🎯 Điều khiển được từ mọi mặt, mặt nào cũng cấu hình được. Thứ tự cố ý: mặt càng
gần lõi làm càng trước, để hợp đồng API được thử lửa trước khi vẽ đẹp.

**5a · API lõi + Terminal.**  **100%** (6 xong)
- [x] `internal/api` — hợp đồng duy nhất, `api.Version = 1`, `api.Actions` liệt kê
  mọi hành động hệ thống làm được.
- [x] `internal/events` — event có `SchemaVersion`, bus trong tiến trình; **lõi
  không in stdout nữa**, nó phát event và CLI chỉ là bộ vẽ đầu tiên.
- [x] CLI viết lại thành client của API; bảng lệnh ánh xạ 1-1 với action.
- [x] **Test ngang quyền** (`cmd/sagent/main_test.go`): mọi action đều phải có
  lệnh CLI, và CLI không được có action ngoài hợp đồng. Luật 2 giờ có răng.
- [x] Trần `max_parallel_sessions` giờ chặn cả **tổng số phiên đang chạy**, không
  chỉ `--copies`. **Mở rộng 22/08 — bốn trần CỘNG DỒN** (`[policy.tran]`): chung ·
  harness · provider · hồ sơ, áp cùng chỗ ở `api.xetTran` → `FleetStart`. Một con
  số không đo được hai thứ khác nhau: **RAM và tiến trình con** tính theo harness,
  còn **hạn mức thuê bao** tính theo tài khoản — bốn phiên rơi hết vào `claude:tns`
  vẫn hợp lệ dưới trần chung trong khi `claude:phu` ngồi không. Việc này **không
  đẻ ra action mới** nên luật ngang quyền bốn mặt không thủng: nó siết đúng
  `fleet.start` đã có. Xem [`docs/BAO-CAO-TRAN-DONG-THOI.md`](BAO-CAO-TRAN-DONG-THOI.md).
- [x] **TUI** (`cmd/sagent/tui.go`): gõ `sagent` không tham số ra bảng chọn
  đánh số như `tk` v1 — số=mở · t=thêm · d=đồng bộ · x=xoá · s=phiên · ?=trợ giúp.
  Không có bàn phím (CI/pipe) thì in bảng rồi thoát, KHÔNG treo.
*DoD:* mọi verb hiện có đi qua API; chạy được qua SSH; không mặt nào gọi tắt vào `store`.

**5b · Dashboard 2D.**  **100%** (9 xong)
- [x] `internal/dash`: server localhost bọc `internal/api` (không mở đường riêng
  vào store). Assets nhúng bằng Go `embed` — vẫn một binary.
- [x] `sagent dash [--port N]`: in URL kèm token, mở trình duyệt là thấy.
- [x] Realtime bằng **SSE** (thuần stdlib, KHÔNG thêm dependency WebSocket).
- [x] Dashboard đọc **ảnh chụp đầy đủ** từ `/api/state` khi kết nối rồi mới dùng
  event cập nhật (không dựng UI chỉ từ event — người nghe chậm có thể lỡ).
- [x] Điều khiển: bật hạm đội + dừng phiên qua POST; đã chạy thật.
- [x] **Bảo mật**: chỉ bind loopback · token ngẫu nhiên · chặn Host lạ (DNS-rebind)
  · chặn Origin lạ trên POST (CSRF) · DTO allowlist nên KHÔNG rò secret. Có test.
- [x] **Trạng thái phiên chi tiết (3 trạng thái đo được)**: phân loại chính xác các phiên gặp sự cố sang `rate_limited` (chạm trần hạn mức, có mốc thời gian mở lại), `blocked` (bị chặn quyền thao tác), và `failed` (lỗi từ phía API nhà cung cấp) dựa trên dữ liệu có cấu trúc từ kết quả phiên (`provider.KetQua`, hiện đo được đầy đủ trên Claude; các adapter chưa đo vẫn giữ `lost`). Riêng trạng thái `queued` (xếp hàng chờ) **CHƯA đo được** và chưa thêm vì hệ thống chưa có cơ chế hàng đợi trong mã nguồn (`FleetStart` từ chối thẳng khi chạm trần `max_parallel_sessions`).
- [x] Approval gate — ~~chờ Pha 3 flow~~ Pha 3 xong rồi, và lỗ hồi quy mà lượt
  soát 21/08 lôi ra **đã vá xong cùng ngày**. Ô này giữ dấu `[~]` thêm một ngày
  sau khi cả hai nửa đã xanh — đúng kiểu bi quan mà mục 7 sinh ra để bắt. Đo lại
  22/08: `grep -rn "flow/decide" internal/dash/web/` ra **`index.html:1910`**, và
  `internal/dash/ngangquyen_ui_test.go:21`
  (`TestMoiHanhDongCuaNguoiDungDeuCoDuongVaoTuWeb`) đang canh chiều UI ↔ API.
  Hai nửa của ô:
  - **Xong — đường server**: `/api/flow/decide` có ở `internal/dash/server.go:86`,
    xử lý ở `internal/dash/flow_api.go:159-182`, và nó gọi ĐÚNG `FlowApproveOnly`
    / `FlowApprove` mà CLI dùng — nên approval gate vẫn không thể bị bỏ qua từ
    đường web. `internal/dash/lachan_test.go:151` giữ ánh xạ `flow.approve` →
    `/api/flow/decide`, tức luật ngang quyền vẫn có răng ở tầng hợp đồng.
  - ~~**CHƯA — KHÔNG TRANG WEB NÀO GỌI NÓ.**~~ ✅ **ĐÃ VÁ 21/08** — xem 5c.
    Mô tả dưới đây giữ lại vì nó là cách ĐO ra lỗi, vẫn dùng được lần sau.
  - **(Hiện trạng lúc phát hiện)** `grep -rn "decide" internal/dash/web/`
    ra **0 dòng**; `grep -rni "reject\|tu-choi"` cũng không có nút nào. Người dùng
    mở dashboard thấy bước `waiting` mà **không có chỗ bấm Duyệt / Từ chối** —
    phải quay về terminal gõ `sagent flow approve <#> <bước>`.
  - **Đây là hồi quy, không phải việc chưa làm**: `git log -S"flow/decide" --
    internal/dash/web/` cho thấy nút từng có ở `c55ed0b` ("Mặt 4: workflow board —
    chạy flow và **duyệt ngay trên web**") và `fcf1b39`, rồi **mất im lặng** trong
    một lần vẽ lại giao diện sau đó. Vì vậy dòng "Duyệt / từ chối ngay trên web"
    đang tick ở Pha 5c **hiện không còn đúng** — xem ghi chú ở đó.
  - Bài học đúng thứ dự án này lấy làm gốc: luật ngang quyền canh **API ↔ CLI**,
    không canh **UI ↔ API**. Nút biến mất mà không test nào đỏ.
- [x] **Bảng quyền plugin ra mặt web** (22/08) — khối "Plugin đã cài" trong ô Tổng
  quan của `internal/dash/web/index.html`, đặt ngay dưới "Năng lực provider" vì
  cùng loại câu hỏi: *thứ này làm được gì trên máy tôi, và tôi chặn được tới đâu*.
  Xem [`docs/BAO-CAO-WEB-PLUGIN.md`](BAO-CAO-WEB-PLUGIN.md).
  - **Mắt còn thiếu, nay đã cắm**: hợp đồng có `plugin.list`, CLI có
    `sagent plugin list|quyen`, endpoint `/api/plugins` có — chỉ mặt web là không.
  - **Yêu cầu thật KHÔNG phải "hiện danh sách plugin"** mà là **hiện cột `[chặn]`
    với đủ BA trạng thái rời nhau**. Một mặt web bẹp `khong-chan-duoc` với `chua-do`
    làm một thì **tệ hơn không có mặt web nào** — nó cho người ta cảm giác đã kiểm tra.
    `internal/dash/plugin_api.go:86-88` thay con số gộp `so_chua_chan` bằng ba ngăn
    riêng (`so_chan_that` · `so_khong_chan_duoc` · `so_chua_do`); `so_chua_chan` giữ
    lại nhưng nay là **tổng** của hai ngăn sau, và không mặt nào được hiện mình nó.
    Vòng đếm chuyển từ so chuỗi trần sang `switch plugin.TrangThaiChan(...)` đối
    chiếu **thẳng với hằng Go**, nhánh `default` gom mọi khoá lạ về `so_chua_do` —
    chiều an toàn: thứ không đọc được phải đếm là "chưa biết", không phải "đã chặn".
  - **Đo bằng đột biến chứ không bằng lời**: 11 phép phá hoại thử, **11/11 bị bắt**
    (bẹp `chua-do` thành `khong-chan-duoc`; hai trạng thái chung một lớp CSS; hai
    lớp khác nhau nhưng chung một màu; hai trạng thái cùng một nhãn chữ; nhãn đếm
    đọc nhầm con số đã gộp…). DTO allowlist giữ nguyên: `Secret` vẫn chỉ là
    `"<tên> → key_id <id>"`, không bao giờ là giá trị.
  - **Một cái bẫy gặp thật, ghi lại**: bài kiểm "mặt web có nói rõ *khai quyền
    không phải hàng rào* không" ban đầu **không cắn** — câu cần tìm nằm sẵn trong
    một dòng **bình luận HTML** giải thích, nên test xanh dù trang không hiện chữ
    nào. Đã thêm `boCommentHTML`. Đúng lớp bẫy mà `boComment` (JS) sinh ra để
    tránh, gặp lại ở dạng HTML.
*DoD:* mọi hành động của UI đều có lệnh CLI tương đương (test ngang quyền) ✅.

**5c · Workflow board.**  **93%** (6 xong · 1 một phần)
- [x] `/flow.html`: chọn flow + tài khoản + biến rồi **chạy**; xem lịch sử; mở
  một lần chạy thấy **từng bước và trạng thái** (done/running/waiting/failed/skipped).
- [x] **Duyệt / từ chối ngay trên web** — cùng đường `Approve()` với CLI, nên
  approval gate vẫn không thể bị bỏ qua. ✅ **VÁ LẠI 21/08, cùng ngày phát hiện**:
  nút từng biến mất trong một lần vẽ lại giao diện (`grep -rn "decide"
  internal/dash/web/` ra **0 dòng**), nay dựng lại ở khối tiến độ lượt chạy trên
  mặt 2D — bước mang trạng thái `waiting` thì hiện thẳng **Duyệt / Từ chối**, gọi
  `POST /api/flow/decide {id, step, approve}`. Khoá cả hai nút ngay khi bấm: mạng
  chậm mà bấm hai lần là gửi hai quyết định.
  **Và ghim để không mất lần nữa**: `TestMoiHanhDongCuaNguoiDungDeuCoDuongVaoTuWeb`
  bắt mọi endpoint hành động phải có ít nhất một trang gọi tới. Luật ngang quyền
  cũ chỉ canh **API ↔ CLI**; đây là mảnh **UI ↔ API** còn thiếu, và chính chỗ
  thiếu đó làm nút biến mất mà không bài kiểm nào đỏ.
- [x] Endpoint chạy flow **trả ngay** rồi làm ở nền: bước agent có thể mất hàng
  chục phút, không được treo request HTTP. Tiến độ đi qua luồng event.
- [x] Đã chạy thật qua HTTP: `shell → approve → shell`, dừng đúng ở gate, duyệt
  trên web thì chạy nốt và về `completed`.
- [x] **Trình soạn thảo node trực quan**: kéo node từ bảng trái vào canvas, kéo
  từ cổng ra sang cổng vào để nối (= `needs`), bấm dây để bỏ nối, kéo node để
  sắp xếp, pan/zoom, cột phải sửa mọi thuộc tính. **Chặn vòng lặp ngay trên bảng**
  trước khi gửi lên server.
- [x] Lưu ghi thẳng vào `flows.toml` (kèm toạ độ `x`/`y` để mở lại đúng chỗ) —
  bảng vẽ KHÔNG có kho riêng, nên flow dựng bằng giao diện và flow viết tay là
  MỘT thứ. Đã kiểm vòng tròn: bảng vẽ → file → `sagent flow show` thấy y hệt.
- [x] Trạng thái khi chạy hiện ngay trên node (chấm ✓/●/?/✗).
*DoD:* flow tạo từ board và flow viết tay chạy y hệt nhau.

**5d · Cấu hình theo project.**  **100%** (6 xong · 20/08/2026)
- [x] Hợp đồng `[ui]`: `default_surface`, `theme`, `columns`, `pinned_flows`,
  `enable_3d`, trên tầng global + project. Cấu hình sai kêu **lúc đọc file**
  (theme lạ, tên cột lạ, mâu thuẫn `default_surface="3d"` + `enable_3d=false`),
  không để mặt web tự đoán rồi vẽ ra trang trống.
- [x] `config.CotTaiKhoan` là nguồn duy nhất cho tên cột; có test giữ nó không
  trôi khỏi bảng nhãn bên JavaScript.
- [x] **Mặt terminal**: gõ `sagent` không tham số ra đúng mặt project khai. Ba mặt
  web chỉ CHỈ ĐƯỜNG chứ không tự bật server — cấu hình nói họ thích mặt nào,
  không phải cho phép mở cổng thay họ.
- [x] **Mặt web**: `[ui]` đi kèm `/api/state` (không làm endpoint riêng — mọi
  trang đã đọc ảnh chụp này khi kết nối, và endpoint mới kéo theo một action
  mới trong hợp đồng). `vendor/mat.js` là một luật cho cả bốn trang.
- [x] `token.css` có bảng sáng. Màu trạng thái đậm hơn bản tối chứ không dùng
  lại: `#2FE0A0` đọc tốt trên nền `#070810` nhưng trên nền trắng thì mất chữ.
- [x] `enable_3d = false` **gỡ hẳn** thẻ `<a>` chứ không ẩn bằng CSS — link ẩn vẫn
  nằm trong thứ tự Tab.
*DoD:* hai project khác nhau mở ra hai bố cục khác nhau, không sửa mã. ✅

**Bảo mật chung cho mọi mặt web:** chỉ bind loopback mặc định; random auth token;
Origin validation + CSRF; **không** đưa credential/env/secret-path lên WebSocket;
log redaction; mọi mutation có audit event.

*DoD chung:* tắt mặt nào lõi vẫn chạy · UI phản ánh **event thật**, không đoán bằng
animation timer · mobile dùng được cho status/approval/stop/log.

### Pha 6 — Mặt thứ tư: 3D  ✅ nền tảng xong
🎯 3D là **projection của cùng event model**, không có business logic riêng —
và cũng **điều khiển được**, không chỉ để ngắm (bấm orb → dừng/duyệt phiên đó).
- Mascot đại diện agent harness **hoặc** AI provider — UI phân biệt rõ hai loại.
- Orb = session thật; InstancedMesh, FogExp2, ACES, reduced-motion, **fallback 2D**.
- Subscription usage vs API token/cost vs rate-limit là chỉ số riêng; chỉ hiện khi có dữ liệu.
- Performance budget + test trên điện thoại tầm trung.
- **Trạng thái:** ✅ view 3D thật ở `internal/dash/web/3d.html` — đọc cùng
  `/api/state` + SSE như 2D, orb = phiên THẬT (màu theo trạng thái), mascot theo
  provider, bấm orb → dừng phiên đó. Nav 2D↔3D giữ token. Không tải được Three.js
  (offline) thì **tự rơi về 2D** thay vì màn hình trống. Nguyên mẫu tĩnh cũ ở
  root `index.html` giữ làm bản trình diễn.

### Pha 7 — Hardening & phát hành  ✅ xong (trừ ký số: quyết định không làm)
- DB migration/rollback + backup restore; Windows ACL/path-traversal/junction-attack test;
  symlink-escape test Linux; process-tree cancel + orphan cleanup; upgrade/provider-drift
  verify; SBOM/license notices/dependency scan; signed + reproducible build; **migration
  guide từ tk v1 / ccswitch**.
- **Trạng thái:**
  - ✅ **dependency scan** — `govulncheck ./...` chạy được, **23 lỗ hổng có đường gọi
    thật → 0**. Toàn bộ là thư viện chuẩn Go, chạm qua `dash.Server.Run → http.Serve`;
    bản vá là ghim `toolchain go1.25.13` trong `go.mod`, không dependency nào phải đổi.
    CI đọc `go-version-file: go.mod` và có job `vuln` riêng. Số đo ở `docs/DO-LUONG.md`.
  - ✅ **path-traversal** — tên hồ sơ không thoát được thư mục nữa; lỗ hổng này **đã nổ
    thật một lần** khi kiểm, xoá mất `~/.claude`. Ghi ở `docs/DO-LUONG.md`.
  - ✅ **junction-attack (Windows)** — đo ra **lỗi thật**: hồ sơ chính nó là junction thì
    `os.ReadDir` đi xuyên, và `Remove` gỡ mất junction dùng chung bên trong thư mục nạn
    nhân rồi trả `nil`. Vá bằng cách kiểm `link.IsLink` **trước** `ReadDir`. Test đã được
    chứng minh là bắt được lỗi (tắt lá chắn → đỏ). Số đo ở `docs/DO-LUONG.md`.
  - ✅ **cửa vào dashboard** — bỏ hẳn token trên URL và header `X-Sagent-Token`; chỉ còn
    đăng nhập bằng mật khẩu băm. Chưa đặt mật khẩu thì server **từ chối chạy**.
  - ✅ **DB migration/rollback + backup restore** — đo ra **lỗi thật**: binary cũ mở
    `state.db` của binary mới thì đọc được VÀ ghi được, im lặng. Vá: chặn hạ cấp, tự sao
    lưu trước khi nâng schema, và lệnh `sagent db info|backup|restore`. Sao lưu bằng
    `VACUUM INTO` chứ không chép file (WAL). Số đo ở `docs/DO-LUONG.md`.
  - ✅ **process-tree cancel + orphan cleanup (một phần)** — đo ra **lỗi thật**:
    `taskkill /T` bỏ sót đám con khi tiến trình cha đã thoát; chúng chạy tiếp và tiêu
    hạn mức, còn `Kill` chỉ trả `exit status 128`. Vá bằng `process.KillTree`: chụp hậu
    duệ trước khi giết, quét lại, rồi mới kết luận. Số đo ở `docs/DO-LUONG.md`.
  - ✅ **quét mồ côi của phiên `lost`** — `sagent quet` (và `--giet`). Mặc định CHỈ BÁO
    vì Windows dùng lại PID; lọc theo thời điểm khởi tạo tiến trình, không đọc được thì
    LOẠI chứ không nhận. In kèm tên và thời điểm để người dùng duyệt được.
  - ✅ **Windows ACL** — đo ra **lỗi thật**: `os.WriteFile(..., 0o600)` không bảo vệ gì
    trên Windows; file `0o600` và `0o644` có ACL y hệt (`BUILTIN\Users:(I)(F)`). Token và
    mật khẩu dashboard chỉ kín nhờ MAY MẮN kế thừa từ `C:\Users\<tên>`. Vá bằng package
    `internal/acl`: DACL tường minh + cắt kế thừa, nối vào kho hồ sơ / thư mục hồ sơ /
    dash-auth, và `sagent verify` có ô kiểm nói trạng thái thật. Số đo ở `docs/DO-LUONG.md`.
  - ✅ **build phát hành** — `-trimpath -ldflags "-s -w"`, đo được **16.21 MB → 11.09 MB**;
    `CGO_ENABLED=0` nên binary không phụ thuộc DLL nào. Workflow `phat-hanh.yml` dựng
    amd64 + arm64 kèm `SHA256SUMS.txt`. Trình cài một dòng, không cần Go, không cần admin.
  - ✅ **SBOM + license notices** — `tools/giayphep` sinh `THONG-BAO-GIAY-PHEP.txt` từ
    `go list -deps` (10 phụ thuộc, toàn văn); CI chạy `-kiem` nên không trôi được. Release
    kèm cả `sbom.cdx.json` (CycloneDX 1.6). **Đo được:** `cyclonedx-gomod -licenses` trả
    0/10 trường giấy phép, im lặng — SBOM KHÔNG thay được notices. Sổ viết tay cũ đã sai
    3 chỗ, nay chỉ giữ phần "vì sao". Số đo ở `docs/DO-LUONG.md`.
  - ✅ **provider-drift verify** — `Adapter.Version()` + `internal/drift`. `sagent verify`
    ghi mốc phiên bản CLI và BÁO ĐỘNG khi nó đổi, vì mọi số đo trong `docs/DO-LUONG.md`
    đều gắn với một bản CLI cụ thể. Cảnh báo cố ý **không tự tắt** — phải `--chap-nhan`.
  - ✅ **migration guide v1** — [`docs/DI-TRU-TU-V1.md`](DI-TRU-TU-V1.md). Câu trả lời đo
    được: **không phải làm gì**, `sagent` đọc thẳng kho `~/.claude-accounts`. Kèm theo,
    đo ra **lỗi thật**: `them` một tên đã có ở kho v1 KHÔNG bị từ chối → hồ sơ cũ bị đè
    bóng, token vẫn trên đĩa nhưng thôi được dùng. Đã vá. `ccswitch` **chưa đo** — chưa
    có bản thật để mở ra xem.
  - 🚫 **Ký số binary — QUYẾT ĐỊNH KHÔNG LÀM** (chủ dự án, 2026-08-17). Dự án mã nguồn
    mở, không mua chứng chỉ code-signing.
    Hệ quả, ghi cho đúng chứ không tô hồng: **SmartScreen vẫn cảnh báo lần chạy đầu.**
    Nó xét chữ ký số và độ phổ biến của file, **không** xét giấy phép — mở mã nguồn
    không gỡ được cảnh báo đó. Thứ thay thế là `SHA256SUMS.txt` trong mỗi release: người
    dùng đối chiếu băm để biết file tải về đúng là file CI dựng ra.
    Muốn làm sau: Azure Trusted Signing (~10 USD/tháng, ký được từ Actions) là đường rẻ
    nhất; lúc đó thêm một bước vào `phat-hanh.yml` là xong.
  - ~~symlink-escape Linux~~ — bỏ cùng nhánh Linux.
  - ✅ **TLS cho dashboard** — phơi ra mạng giờ **mặc định HTTPS**, chứng chỉ tự ký sinh
    tự động, phủ mọi IP của máy, in vân tay SHA-256 ra terminal để đối chiếu. Muốn HTTP
    trần phải gõ `--http-tran`, và chốt từ chối nằm trong `Server.Run` chứ không chỉ ở
    CLI. Đo trên cổng thật: `200 HTTPS`. Số đo ở `docs/DO-LUONG.md`.
  - ⚠ **Chứng chỉ TỰ KÝ, không phải CA công cộng.** Trình duyệt vẫn cảnh báo; người dùng
    phải đối chiếu vân tay. Không đối chiếu = chống được nghe lén, không chống kẻ đứng giữa.

### Bảng đếm lại (soát 22/08/2026)

`[x]`=1 · `[~]`=0,5 · `[ ]`=0 · `[!]` không vào mẫu số.

| Pha | `[x]` | `[~]` | `[ ]` | `[!]` | Điểm | % | so với 21/08 |
|---|---|---|---|---|---|---|---|
| Bước 0 — Đổi tên | 4 | 0 | 0 | 0 | 4,0/4 | **100%** | ↑ 75% |
| Pha 0 — Đo giả định | 6 | 0 | 0 | 1 | 6,0/6 | **100%** | ↑ 71% |
| Pha 1 — Storage + slice | 9 | 1 | 0 | 0 | 9,5/10 | **95%** | = |
| Pha 2 — Song song + Workspace | 22 | 0 | 0 | 0 | 22,0/22 | **100%** | = |
| Pha 2.5 — Codex + OpenAI-compat | 8 | 0 | 0 | 0 | 8,0/8 | **100%** | = |
| Pha 3 — Flow DAG | 20 | 1 | 0 | 0 | 20,5/21 | **98%** | ↑ 90% |
| Pha 5a — API lõi + Terminal | 6 | 0 | 0 | 0 | 6,0/6 | **100%** | = |
| Pha 5b — Dashboard 2D | 9 | 0 | 0 | 0 | 9,0/9 | **100%** | ↑ 94% |
| Pha 5c — Workflow board | 7 | 0 | 0 | 0 | 7,0/7 | **100%** | ↑ 93% |
| Pha 5d — Cấu hình theo project | 6 | 0 | 0 | 0 | 6,0/6 | **100%** | = |
| **TỔNG** | **97** | **2** | **0** | **1** | **98,0/99** | **99%** | ↑ 93% |

Pha 4, 6, 7 không dùng ô tick (viết bằng danh sách trạng thái ✅/⬜/🚫), nên không
nằm trong bảng này — trạng thái của chúng đọc thẳng ở mục tương ứng.

⚠ **99% nghĩa là 99% SỐ Ô TICK của mục 7, KHÔNG phải 99% dự án.** Mẫu số là 99 ô
(một ô `[!]` không vào mẫu số);
Pha 4, 6, 7 không có mặt trong đó, và một ô `[~]` đáng nửa điểm có thể chứa khối
lượng như "ba giao thức API còn lại". Một con số tròn trịa không kèm mẫu số là
cách một tài liệu trung thực bắt đầu nói dối.

**93% → 98% ở lượt soát trước → 99% sau khi đối chiếu ba merge, và phải tách
con số đó ra vì các thay đổi khác hẳn nhau:**

1. **Bảng cũ đã sai sẵn 2 điểm trước khi ai làm gì hôm nay.** Bảng khai
   `87 · 10 · 2 = 92,0/99 = 93%`; đếm lại đúng bản `.md` **trước** lượt sửa này ra
   `90 · 8 · 1 = 94,0/99 = 95%`. Ba ô đã được tick trong phần chữ mà **không ai đi
   sửa bảng**: plugin model (Pha 3), threat model (Pha 0), Duyệt/Từ chối trên web
   (Pha 5c). Đây là chính cái bệnh mà lượt soát 21/08 đi chữa, tái phát sau đúng
   một ngày ở dạng nhẹ hơn — lần này là **bảng tổng** trôi khỏi phần chữ, chứ
   không phải phần chữ trôi khỏi mã.
2. **Phần còn lại — 95% → 98% — là việc làm thật hôm 22/08**: **5 ô đổi dấu**
   (alias `tk`/`ccswitch` `[ ]`→`[x]`; redaction `[~]`→`[x]`; bảng năng lực
   `[~]`→`[x]`; node built-in `[~]`→`[x]`; approval gate 5b `[~]`→`[x]`) và **1 ô
   mới thêm vào mẫu số** (bảng quyền plugin ra mặt web, Pha 5b) — nên mẫu số đi từ
   99 lên 100.
3. **Ba merge 22/08 — 98% → 99%**: commit `e234b45` đóng phần còn lại trong repo
   của `Subscription`, nên ô `[~]→[x]`; commit `5f922c1` đóng vision + structured
   output, khiến phần còn lại của `API` chỉ còn blocker bên ngoài, nên `[~]→[!]`.
   Điểm vẫn là 98,0 nhưng mẫu số đúng giảm từ 100 xuống 99. Commit `2b437e4` đóng
   route theo năng lực; ô `Engine flow` vẫn `[~]` vì route theo giá chưa đủ dữ liệu.

**KHÔNG CÒN ô `[ ]` nào.** Hai mục "thật sự chưa làm" mà lượt soát 21/08 chỉ đích
danh — alias ở Bước 0 và plugin model ở Pha 3 — **đã đóng cả hai** trong ngày.

**Ô `[!]` nay có đúng một mục: API ở Pha 0.** Sau commit `5f922c1`, bốn năng lực
phía dự án đã đóng; toàn bộ phần còn lại của ô là ba giao thức thiếu khoá gốc bên
ngoài. Các dòng `⬜`/`🚫` của Pha 4 vẫn không nằm trong bảng tick:

| Bị chặn ở đâu | Thiếu THỨ GÌ bên ngoài |
|---|---|
| Pha 0 · ô API — ba giao thức còn lại | **Khoá thật của chính nhà cung cấp**: Anthropic Messages (`api.anthropic.com`), Gemini native (`generativelanguage.googleapis.com`), OpenAI Responses (`api.openai.com`). Máy chỉ có khoá **nhà bán lại** `modelapi.vn`, và khoá đó đã đo là **401 ở `api.deepseek.com`** |
| Pha 4 · OpenRouter | **Khoá API thật** của openrouter.ai |
| Pha 4 · Ollama | **Khoá VÀ phần mềm** — `ollama` chưa được cài trên máy này, không có endpoint cục bộ nào để trỏ tới |

API nay đủ điều kiện mang `[!]` theo đúng luật ở đầu mục 7: **cả phần còn lại** bị
chặn. OpenRouter/Ollama vẫn là trạng thái của Pha 4, không phải ô tick cấp cao.

**Hai ô `[~]` còn lại, và mỗi ô còn thiếu đúng cái gì:**

| Ô | Ở đâu | Phần chưa xong |
|---|---|---|
| 1 direct-API vertical slice | Pha 1 | xem chi tiết tại ô đó |
| Engine flow | Pha 3 | **route theo giá** — route theo capability đã đóng ở commit `2b437e4`; phần giá chỉ làm khi có dữ liệu và phép đối chiếu |

---

## 8. Project config `.sagent/project.toml` (không chứa secret)

```toml
version = 1
name = "example-app"
[project]      root=".";  default_branch="main";  workspace="worktree"
[commands]     setup=["npm ci"]; lint=["npm run lint"]; test=["npm test"]; build=["npm run build"]
[instructions] files=["AGENTS.md","CLAUDE.md"]
[policy]       max_parallel_sessions=4;  require_approval_for=["merge","deploy","destructive_shell"]
[ai]           default_route="coding-primary";  fallback_routes=["coding-secondary","local-fallback"]
[ai.requirements] capabilities=["tool_calling","streaming"];  preferred_models=["provider/model-id"]
[workspace]    copy=[".env.example"];  link=["node_modules"];  deny=[".env","*.pem","secrets/**"]

# Bốn mặt điều khiển: mỗi project tự chọn mặt mặc định và bày biện riêng.
[ui]
default_surface = "tui"          # tui | dashboard | workflow | 3d
theme           = "dark"
[ui.dashboard]
columns  = ["session","harness","model","state","tokens","cost","elapsed"]
group_by = "project"
[ui.workflow]
pinned_flows = ["fanout","squad"]
autolayout   = true
[ui.3d]
enabled       = true              # tắt được cho máy yếu
max_orbs      = 200               # vượt ngưỡng thì gộp, tránh tụt khung hình
reduced_motion = "auto"
```

Route chỉ chứa **ID tham chiếu**; key/token **không** nằm trong project config. Schema
versioned, validation chặt, xử lý đúng Windows path/monorepo/command-có-khoảng-trắng; ưu
tiên **argv** thay vì nối chuỗi shell.

---

## 9. Chiến lược test bắt buộc

- **Unit:** state transitions · DAG validation · path-ownership/safe-delete · JSON
  case-insensitive duplicate · redaction · capability negotiation.
- **Ngang quyền giữa bốn mặt (mục 2c luật 2):** test liệt kê mọi hành động API và
  khẳng định **mỗi hành động đều có lệnh CLI tương đương**. Thêm nút trên UI mà
  quên lệnh CLI thì test đỏ — đây là thứ giữ cho terminal không bị bỏ rơi.
- **Contract/conformance:** mọi Harness Adapter/Agent Driver chạy cùng bộ test (isolated
  roots, verify capability đúng, start/stop/cancel idempotent, không log secret, behavior
  khi binary/version thiếu). Mọi AI Provider Adapter/Model Client chạy API conformance
  (auth/base-URL/header, streaming lifecycle + cancel, tool/structured/reasoning theo
  capability, error/rate-limit/retry normalize, usage/token/cost, **không rò key**).
- **Integration:** native Win + Linux; worktree create/cleanup; daemon restart/crash;
  SQLite migration + concurrent writes; ACP agent thật khi có + fake PTY cho CI; fake
  HTTP/SSE cho CI + opt-in smoke với API thật; route fallback/circuit-breaker/reconnect.
- **Security:** path traversal, symlink/junction escape, malicious project-config/plugin,
  dashboard cross-origin, command injection, credential trong log/event/error.
- **Performance/reliability:** 10 session baseline, event burst + reconnect, long-run flow
  resume, dashboard 2D/3D budget.

Test dùng fake credentials; test cần credential thật phải **opt-in, local-only, redaction**.

---

## 10. Definition of Done toàn cục

Một feature xong khi: (1) có spec + acceptance; (2) mã port trực tiếp thì có
source/license/attribution (ADR chỉ bắt buộc cho quyết định kiến trúc lớn); (3) có test
chạy trên OS liên quan; (4) error message hướng dẫn bước tiếp; (5) telemetry/event không
lộ secret; (6) có quyết định migration/tương thích; (7) docs + example cập nhật; (8)
`go test`/lint/race xanh; (9) không làm yếu permission/security để test qua; (10) không
tự gắn `stable` khi chưa có evidence matrix.

## 11. Nhịp làm việc

`Understand → Research khi hữu ích → Thin vertical slice → Tests/fault-injection →
Self-review → Docs → Compound knowledge`. Chia PR nhỏ (mỗi PR một quyết định chính); không
refactor rộng ngoài phạm vi; không thêm dependency khi stdlib đủ; ghi giả định mới vào
research backlog; **dừng và báo blocker** nếu cần credential thật, xoá dữ liệu, hoặc mở
dashboard ra mạng ngoài. Sau mỗi pha: `docs/knowledge/PHASE-<n>-RETROSPECTIVE.md`.

---

## 12. Việc còn treo cần bạn quyết/cung cấp

1. ✅ Tên đã chốt: **Switch-Agent-Pro**, lệnh `sagent` (code đã đổi tên, build xanh).
2. ~~**Máy/VM Linux** — chặn Pha 0 Linux + đa nền tảng.~~ ✅ **HẾT HIỆU LỰC** —
   nhánh Linux đã **bỏ khỏi phạm vi** theo quyết định "CHỈ WINDOWS" ở đầu tài
   liệu, nên đây không còn là thứ cần cung cấp.
3. 🚫 **API key thật của CHÍNH nhà cung cấp** — vẫn treo, và nay là thứ duy nhất
   ở mục này còn chặn việc. Đã có key của **nhà bán lại** `modelapi.vn` (đủ để
   đóng đường OpenAI-compatible, streaming, usage, fallback, 429, và bảng năng
   lực 7 × 2 route). **Còn thiếu**: khoá `api.anthropic.com` (Anthropic Messages),
   `generativelanguage.googleapis.com` (Gemini native), `api.openai.com` (OpenAI
   Responses), openrouter.ai (OpenRouter) — và **Ollama thì thiếu cả phần mềm**,
   chưa cài trên máy này. Nhãn `experimental` đã bị bỏ hẳn khỏi dự án: ba trạng
   thái `LamDuoc` / `KhongLamDuoc` / `ChuaDo` thay chỗ nó ở cả hai nửa.
4. ✅ **Đã chốt và đã làm**: SQLite là SSOT từ Pha 2 (`~/.ai-accounts/state.db`,
   migration có version); ~~daemon~~ bỏ theo mục 2b — `dash` là server thay thế.
