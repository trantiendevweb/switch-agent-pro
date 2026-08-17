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

### Bước 0 — Đổi tên ✅ XONG (2026-08-17)
- [x] `go.mod` module → `github.com/trantiendevweb/switch-agent-pro`; `cmd/ccswitch`
  → `cmd/sagent` (git mv); cập nhật mọi import; **build + vet + test xanh**; `sagent ds` chạy đúng.
- [x] Cập nhật `install/cai-dat.{ps1,sh}`, CI, `.gitignore` sang `sagent`.
- [ ] Alias tương thích `tk`/`ccswitch` → `sagent` (làm khi viết installer phát hành).
- [x] Rà tên toàn repo: README viết lại cho Switch-Agent-Pro; bộ PowerShell v1 chuyển vào
  `legacy/v1-powershell/`; `design-system/switch-agent-pro/`; 3 trang HTML sạch tên cũ.

### Pha 0 — Đo giả định & lập hợp đồng
🎯 Chứng minh cơ chế của **cả hai đường** trước khi khoá interface.
- [ ] Test harness không chứa credential trong repo; mọi output **redaction**.
- [ ] **Subscription** (Claude ✓Windows, Codex, Gemini CLI, Cursor · Win+Linux): config
  root override có bao trùm config/session/auth? token ở file/env/keyring? file nào
  đọc/ghi lúc login/prompt/refresh/exit? **concurrent refresh** khi 2 process chung
  credential? copy token có tạo session hợp lệ, bao lâu? headless/JSON/stream/ACP/
  resume/cancel? state máy dùng chung ngoài config root?
- [ ] **API** (Anthropic, OpenAI, Gemini, xAI/Grok, DeepSeek, OpenAI-compatible): auth
  mode, base URL, model naming, headers; protocol (Responses/Chat Completions/Anthropic
  Messages/Gemini native); streaming/tool/reasoning/vision/structured-output/usage;
  error+rate-limit schema, retry headers, health, model discovery.
- [ ] Junction Windows ✓ / symlink Linux từ Go, không admin.
- [ ] Behavior khi stream/process ngắt, reboot, config ghi dở.
- [ ] **Capability matrix** stable/experimental/unsupported/unknown (harness + API).
- [ ] **Threat model** cho subscription credential, API key, dashboard, command exec.
- **Artifact:** `docs/research/phase0/*` (ENVIRONMENT, CLAUDE, CODEX, GEMINI, CURSOR,
  ANTHROPIC-API, OPENAI-API, …, CAPABILITY-MATRIX), `docs/security/THREAT-MODEL.md`,
  `docs/adr/0001-domain-boundaries.md`, `docs/OPEN_SOURCE_LEDGER.md`.
- **DoD:** mỗi kết luận có command/OS/output-redacted; **không token thật** ở đâu;
  capability chưa đo = `unknown`; interface nháp suy ra từ **≥2 harness và ≥2 API protocol**.
- ⚠ **Blocker cần bạn:** **API key** thật (local-only, redaction) cho phần API path.
  (Blocker "máy/VM Linux" đã bỏ cùng nhánh Linux — xem khối quyết định đầu tài liệu.)
- **Trạng thái:** Windows/Claude subscription + junction đã đo (`docs/DO-LUONG.md`); còn lại chưa.

### Pha 1 — Storage + Claude slice + 1 API slice
🎯 Thay chức năng v1 bằng lõi có ranh giới rõ, storage an toàn, và **hai lát cắt dọc**.
- [x] ~~`internal/domain`~~ **bỏ** theo mục 2b — dùng thẳng các package hiện có.
- [ ] `internal/store`: **SQLite** schema/migration cho auth profile, harness, provider,
  route, project, session, event.
- [ ] `jsonutil` (đã có) + atomic replace + preservation test (giữ).
- [ ] link abstraction (đã có) → materialize config root cho harness.
- [ ] **Claude Harness Adapter** + subscription capability report + conformance test
  (nâng từ `provider/claude.go` hiện tại).
- [ ] **1 direct-API vertical slice** (provider Pha 0 xác nhận phù hợp, vd Anthropic API):
  stream response, ghi usage/error, **không lộ key**.
- [ ] Verb: `profile create/list/verify/remove`, `route create/list/test`,
  `session run/list/stop`.
- [ ] Safe delete: chỉ xoá **materialized session directory** mà registry sở hữu.
- **DoD:** CI Windows xanh; đổi Claude subscription không đăng nhập lại; API route
  stream + ghi usage/error không lộ key; xoá session không đụng credential/project gốc;
  fault injection không tạo JSON/DB dở; **hết Python**.
- **Trạng thái:** phần subscription-switch đã chạy trên Windows; còn thiếu domain layer,
  SQLite, API slice.

### Pha 2 — Chạy song song + Project/Workspace  🟡 một phần
🎯 Biến công cụ profile thành **runtime manager** đa project.
- [x] **SQLite là SSOT** (`~/.ai-accounts/state.db`); PID chỉ là thuộc tính
  runtime — `status` đối chiếu PID thật và tự đánh dấu `lost` phiên đã chết.
- [x] `clone` — chép credential ra N config dir riêng (mỗi bản `.claude.json`
  riêng nên **không đua ghi**); `fleet` — bật N phiên nền, log ra file;
  `status`; `stop <số|all>`; `clean` — xoá clone **an toàn** (không xuyên junction).
- [x] Cảnh báo thẳng: tiêu hạn mức gấp N, và **concurrent refresh chưa đo**.
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
- [ ] ~~`sagentd` daemon~~ **bỏ** (xem mục 2b) — `dash` sẽ là server tạm.
- [ ] Project discovery + `.sagent/project.toml`.
- [ ] Workspace backend: directory + **Git worktree**.
- [ ] Process backend native Windows/Linux; tmux tuỳ chọn (Linux).
- [ ] Session state machine + event stream.
- [ ] Route engine cơ bản: chọn provider/model/profile, health, fallback, usage/cost event.
- [ ] Recovery khi daemon/process chết; cleanup orphan an toàn.
- **DoD:** 4 session cùng profile + 3 profile khác chạy đúng policy; 10 session đồng thời
  không hỏng config/state; restart daemon phục hồi đúng; ≥2 repo khác stack; không session
  nào ghi vào worktree session khác; chạy song song ≥1 subscription session và ≥1 API node.
- **Ánh xạ:** thay thế prototype `registry`/`fleet` hiện tại.

### Pha 2.5 — Codex + OpenAI-compatible (chống overfit Claude)  🟡 Codex xong
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
- [ ] Đường API (OpenAI-compatible) — chờ API key.
🎯 Chứng minh kiến trúc không bị đo ni theo Claude/Anthropic.
- **DoD:** Claude & Codex dùng chung domain/session API (subscription); Anthropic API &
  OpenAI-compatible dùng chung route API (direct); khác biệt auth/config nằm trong adapter,
  khác biệt tương tác nằm trong driver/model client; **conformance suite** chạy cho 2 harness
  + 2 API protocol; capability không hỗ trợ báo trung thực.

### Pha 3 — Flow DAG ghép được  🟢 chạy được (còn workflow board)
🎯 Người dùng định nghĩa workflow mới **không sửa mã Go**.
- [ ] Engine: DAG + cycle validation; input/output/artifact giữa step; condition/timeout/
  retry-backoff/cancel; concurrency limit (global/harness/provider/profile/project); route
  theo capability/model/giá/health/fallback; **approval gate**; **resume sau restart**;
  idempotency key; failure policy (stop/continue/fallback/compensate).
- [ ] Node built-in: `agent · model · route · shell · test · lint · review · approve · merge · notify`.
- [ ] 3 flow mẫu: `fanout` (nhiều agent → review → chọn), `squad` (API planner → agent
  implementer → reviewer → test → approval), `agents` (danh sách task theo concurrency).
- [ ] Plugin model: TOML chỉ manifest/config tĩnh; logic động là **executable riêng** qua
  JSON-RPC/stdio versioned; secret trong TOML chỉ là reference; plugin chạy capability tối thiểu.
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
- [x] Lệnh: `flow run | runs | approve | reject | resume`.
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
- [ ] Workflow board (mặt 4).
- **DoD:** thêm flow mới không rebuild binary; fake harness/API/agent chạy trong CI; flow
  đang chạy tiếp tục sau restart; test chứng minh **approval không thể bị bỏ qua**.

### Pha 4 — Mở rộng harness + AI API
- Subscription: **Gemini CLI, Cursor**, OpenCode (nếu đo được).
- API: **Google Gemini, xAI/Grok, DeepSeek, OpenRouter, Mistral, Groq, Ollama/LocalAI**,
  generic OpenAI-compatible; Azure/Bedrock/Vertex ở lớp plugin/enterprise nếu cần.
- Mỗi tích hợp lặp: measurement → adapter → conformance → streaming/tool/error/usage → capability label.
- **DoD:** Grok & DeepSeek chạy qua API profile riêng, chọn model + stream; generic
  OpenAI-compatible hoạt động với custom base URL/model/headers; fallback không mất
  correlation ID/usage/error gốc; thêm model/provider chỉ khác endpoint bằng manifest;
  chưa xác minh giữ `experimental`/`unknown`.

### Pha 5 — Bốn mặt điều khiển (làm theo thứ tự 5a → 5d)
🎯 Điều khiển được từ mọi mặt, mặt nào cũng cấu hình được. Thứ tự cố ý: mặt càng
gần lõi làm càng trước, để hợp đồng API được thử lửa trước khi vẽ đẹp.

**5a · API lõi + Terminal.**  ✅ XONG
- [x] `internal/api` — hợp đồng duy nhất, `api.Version = 1`, `api.Actions` liệt kê
  mọi hành động hệ thống làm được.
- [x] `internal/events` — event có `SchemaVersion`, bus trong tiến trình; **lõi
  không in stdout nữa**, nó phát event và CLI chỉ là bộ vẽ đầu tiên.
- [x] CLI viết lại thành client của API; bảng lệnh ánh xạ 1-1 với action.
- [x] **Test ngang quyền** (`cmd/sagent/main_test.go`): mọi action đều phải có
  lệnh CLI, và CLI không được có action ngoài hợp đồng. Luật 2 giờ có răng.
- [x] Trần `max_parallel_sessions` giờ chặn cả **tổng số phiên đang chạy**, không
  chỉ `--copies`.
- [x] **TUI** (`cmd/sagent/tui.go`): gõ `sagent` không tham số ra bảng chọn
  đánh số như `tk` v1 — số=mở · t=thêm · d=đồng bộ · x=xoá · s=phiên · ?=trợ giúp.
  Không có bàn phím (CI/pipe) thì in bảng rồi thoát, KHÔNG treo.
*DoD:* mọi verb hiện có đi qua API; chạy được qua SSH; không mặt nào gọi tắt vào `store`.

**5b · Dashboard 2D.**  ✅ nền tảng xong
- [x] `internal/dash`: server localhost bọc `internal/api` (không mở đường riêng
  vào store). Assets nhúng bằng Go `embed` — vẫn một binary.
- [x] `sagent dash [--port N]`: in URL kèm token, mở trình duyệt là thấy.
- [x] Realtime bằng **SSE** (thuần stdlib, KHÔNG thêm dependency WebSocket).
- [x] Dashboard đọc **ảnh chụp đầy đủ** từ `/api/state` khi kết nối rồi mới dùng
  event cập nhật (không dựng UI chỉ từ event — người nghe chậm có thể lỡ).
- [x] Điều khiển: bật hạm đội + dừng phiên qua POST; đã chạy thật.
- [x] **Bảo mật**: chỉ bind loopback · token ngẫu nhiên · chặn Host lạ (DNS-rebind)
  · chặn Origin lạ trên POST (CSRF) · DTO allowlist nên KHÔNG rò secret. Có test.
- [ ] Trạng thái phiên chi tiết (queued/blocked/rate_limited…) — hiện mới running/stopped/lost.
- [ ] Approval gate (chờ Pha 3 flow).
*DoD:* mọi hành động của UI đều có lệnh CLI tương đương (test ngang quyền) ✅.

**5c · Workflow board.**  ✅ bản vận hành xong
- [x] `/flow.html`: chọn flow + tài khoản + biến rồi **chạy**; xem lịch sử; mở
  một lần chạy thấy **từng bước và trạng thái** (done/running/waiting/failed/skipped).
- [x] **Duyệt / từ chối ngay trên web** — cùng đường `Approve()` với CLI, nên
  approval gate vẫn không thể bị bỏ qua.
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

**5d · Cấu hình theo project.** `[ui]` trong `.sagent/project.toml` (mặt mặc định,
cột, flow ghim, bật/tắt 3D) + tầng global.
*DoD:* hai project khác nhau mở ra hai bố cục khác nhau, không sửa mã.

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

### Pha 7 — Hardening & phát hành  🔄 đang làm
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
    duệ trước khi giết, quét lại, rồi mới kết luận. **Chưa bịt:** hậu duệ của phiên tự
    chết (`lost`) vẫn không ai quét. Số đo ở `docs/DO-LUONG.md`.
  - ✅ **Windows ACL** — đo ra **lỗi thật**: `os.WriteFile(..., 0o600)` không bảo vệ gì
    trên Windows; file `0o600` và `0o644` có ACL y hệt (`BUILTIN\Users:(I)(F)`). Token và
    mật khẩu dashboard chỉ kín nhờ MAY MẮN kế thừa từ `C:\Users\<tên>`. Vá bằng package
    `internal/acl`: DACL tường minh + cắt kế thừa, nối vào kho hồ sơ / thư mục hồ sơ /
    dash-auth, và `sagent verify` có ô kiểm nói trạng thái thật. Số đo ở `docs/DO-LUONG.md`.
  - ✅ **build phát hành** — `-trimpath -ldflags "-s -w"`, đo được **16.21 MB → 11.09 MB**;
    `CGO_ENABLED=0` nên binary không phụ thuộc DLL nào. Workflow `phat-hanh.yml` dựng
    amd64 + arm64 kèm `SHA256SUMS.txt`. Trình cài một dòng, không cần Go, không cần admin.
  - ⬜ Còn lại: quét mồ côi của phiên `lost` · upgrade/provider-drift verify · SBOM +
    license notices · **ký số** binary (đã có băm, chưa có chữ ký) · migration guide v1.
  - ~~symlink-escape Linux~~ — bỏ cùng nhánh Linux.
  - ⚠ **HTTP không mã hoá.** `--host 0.0.0.0` gửi mật khẩu dạng trần trên đường truyền.
    Chưa có TLS — phải giải quyết trước khi gọi là "phát hành được".

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
2. **Máy/VM Linux** — chặn Pha 0 Linux + đa nền tảng.
3. **API key thật** (local-only, redaction) để đo + làm API path — hoặc hoãn API path,
   giữ nhãn `experimental`, làm subscription path trước.
4. Mức độ dùng **daemon + SQLite** ngay từ Pha 2 (đúng plan) hay giữ prototype nhẹ thêm
   một nhịp — khuyến nghị: theo plan (SQLite SSOT) để không phải viết lại.
