# Switch-Agent-Pro — Master Plan (hợp nhất)

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
  chạy native trên Windows + Linux, một binary, có dashboard quan sát realtime.*

Đích không phải "mở nhiều terminal", mà là: dùng **cả subscription profile lẫn API
profile**, gắn vào **agent harness** hoặc **model route** phù hợp, chạy trong
**workspace biệt lập của từng project**, điều phối bằng **flow khai báo được**, lưu
**trạng thái bền vững**, và **quan sát realtime**.

---

## 1. Sáu nguyên tắc (linh hồn, không đổi)

1. **Đã đo — không suy luận.** Vị trí token, biến env, refresh, session resume,
   trạng thái CLI: chưa có thí nghiệm tái lập thì chưa được coi là đúng.
2. **Whitelist — không blacklist.** Chỉ chia sẻ file/khoá config đã biết là an toàn.
3. **Xoá an toàn.** Không bao giờ xoá credential/dữ liệu/project gốc vì một session bị xoá.
4. **Ghi nguyên tử.** State quan trọng: temp + fsync + rename, hoặc transaction DB.
5. **Local-first, một binary.** Lõi chạy độc lập Windows + Linux; dashboard chỉ là client.
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

```
cmd/sagent/                  # CLI client
cmd/sagentd/                 # daemon (đóng gói chung binary được)
internal/domain/          # pure types + state machines (KHÔNG import provider/db/http/ui)
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
| Lõi Go đổi tài khoản Claude (Windows) | `cmd/ccswitch`, `internal/{paths,provider,jsonutil,link,profile}` | **Phần của Pha 1** (vertical slice Claude subscription) — cần **tách domain** + **đổi tên** |
| Bỏ Python, khoá JSON trùng, ghi nguyên tử, xoá an toàn | `jsonutil`, `profile` + test | Giữ, đưa vào `store`/`harness` mới |
| link junction/symlink đa nền tảng | `internal/link` | Giữ, thành nền `workspace`/materialize |
| `running.json` + fleet prototype | `internal/{registry,fleet}` | **Prototype — sẽ thay bằng SQLite SSOT + daemon** (PID chỉ là runtime attribute) |
| Design tokens + dashboard 3D + mascot | `design-system/`, `dashboard-preview.html`, `plan.html` | **Nguyên mẫu Pha 6** — biến thành client của event API |
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
- [ ] Rà tên trong docs cũ (README/HDSD/SKILL/THIET-KE vẫn ghi ccswitch — pass thẩm mỹ sau).

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
- ⚠ **Blocker cần bạn:** máy/VM **Linux**; **API key** thật (local-only, redaction) cho phần API path.
- **Trạng thái:** Windows/Claude subscription + junction đã đo (`docs/DO-LUONG.md`); còn lại chưa.

### Pha 1 — Lõi domain + storage + Claude slice + 1 API slice
🎯 Thay chức năng v1 bằng lõi có domain rõ, storage an toàn, và **hai lát cắt dọc**.
- [ ] `internal/domain`: types + state machines (không phụ thuộc hạ tầng).
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
- **DoD:** CI Windows+Linux xanh; đổi Claude subscription không đăng nhập lại; API route
  stream + ghi usage/error không lộ key; xoá session không đụng credential/project gốc;
  fault injection không tạo JSON/DB dở; **hết Python**.
- **Trạng thái:** phần subscription-switch đã chạy trên Windows; còn thiếu domain layer,
  SQLite, API slice.

### Pha 2 — Daemon + Project/Workspace + chạy song song
🎯 Biến công cụ profile thành **runtime manager** đa project.
- [ ] `sagentd` daemon + versioned API; CLI thành client.
- [ ] **SQLite là SSOT**; PID chỉ là thuộc tính runtime (thay `running.json`).
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

### Pha 2.5 — Codex + OpenAI-compatible (chống overfit Claude)
🎯 Chứng minh kiến trúc không bị đo ni theo Claude/Anthropic.
- **DoD:** Claude & Codex dùng chung domain/session API (subscription); Anthropic API &
  OpenAI-compatible dùng chung route API (direct); khác biệt auth/config nằm trong adapter,
  khác biệt tương tác nằm trong driver/model client; **conformance suite** chạy cho 2 harness
  + 2 API protocol; capability không hỗ trợ báo trung thực.

### Pha 3 — Flow DAG ghép được
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

### Pha 5 — Dashboard vận hành 2D
🎯 Giao diện vận hành chính xác **trước** khi trang trí 3D.
- Trạng thái chuẩn: `queued·starting·running·waiting_input·blocked·rate_limited·completed·failed·cancelling·cancelled·lost`.
- Bảo mật: **chỉ bind loopback** mặc định; random auth token; Origin validation + CSRF;
  không đưa credential/env/secret-path lên WebSocket; **log redaction**; mutation có audit event.
- **DoD:** tắt dashboard không ảnh hưởng daemon/session; UI phản ánh **event thật**, không
  đoán bằng animation timer; mobile dùng được cho status/approval/stop/log-redacted; tách
  bạch harness/model/API provider/auth profile/route/latency/token/cost khi có dữ liệu.

### Pha 6 — Dashboard 3D
🎯 3D là **projection của cùng event model**, không có business logic riêng.
- Mascot đại diện agent harness **hoặc** AI provider — UI phân biệt rõ hai loại.
- Orb = session thật; InstancedMesh, FogExp2, ACES, reduced-motion, **fallback 2D**.
- Subscription usage vs API token/cost vs rate-limit là chỉ số riêng; chỉ hiện khi có dữ liệu.
- Performance budget + test trên điện thoại tầm trung.
- **Trạng thái:** đã có nguyên mẫu tĩnh (`dashboard-preview.html`, mascot robot biết đi,
  responsive mobile) — Pha này nối nó vào event API thật.

### Pha 7 — Hardening & phát hành
- DB migration/rollback + backup restore; Windows ACL/path-traversal/junction-attack test;
  symlink-escape test Linux; process-tree cancel + orphan cleanup; upgrade/provider-drift
  verify; SBOM/license notices/dependency scan; signed + reproducible build; **migration
  guide từ tk v1 / ccswitch**.

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
```

Route chỉ chứa **ID tham chiếu**; key/token **không** nằm trong project config. Schema
versioned, validation chặt, xử lý đúng Windows path/monorepo/command-có-khoảng-trắng; ưu
tiên **argv** thay vì nối chuỗi shell.

---

## 9. Chiến lược test bắt buộc

- **Unit:** state transitions · DAG validation · path-ownership/safe-delete · JSON
  case-insensitive duplicate · redaction · capability negotiation.
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
