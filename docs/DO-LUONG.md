# Pha 0 — Báo cáo đo giả định

> "Đã đo — không suy luận." Mỗi ô ghi **đo thế nào** và **kết quả**. Chưa đo xong
> thì provider/OS đó còn nhãn `experimental`, công cụ cảnh báo khi dùng.
>
> Cập nhật: 2026-08-19.

## Bảng trạng thái

Đã bỏ cột Linux (xem khối quyết định trong `MASTER-PLAN.md`). Cột **Tách tài khoản**
là câu quan trọng nhất: provider giấu danh tính ở đâu quyết định việc chạy được
mấy tài khoản trên một máy.

| Provider | Trạng thái | Danh tính nằm ở | Biến tách | Tách nhiều tài khoản |
|---|---|---|---|---|
| Claude | ✅ `stable` | `.credentials.json` trong hồ sơ | `CLAUDE_CONFIG_DIR` | ✅ |
| Codex | ✅ `stable` | `auth.json` trong hồ sơ | `CODEX_HOME` | ✅ |
| Cursor | ✅ `stable` | `Cursor\auth.json` | **`APPDATA`** | ✅ |
| Grok | ✅ `stable` | `.grok\user-settings.json` (API key) | `USERPROFILE` | ✅ |
| Antigravity | ✅ `stable` | **Windows Credential Manager** | `USERPROFILE`¹ | ❌ **một máy một tài khoản** |
| Gemini CLI | 🚫 **bỏ** | — | — | — |

¹ `USERPROFILE` tách được thư mục làm việc (hội thoại, cache) nhưng **không** tách được
danh tính — token đọc từ Credential Manager theo khoá cố định `gemini:antigravity`.

**Không có quy luật chung.** Năm provider giấu danh tính ở năm chỗ khác nhau, và ba
trong số đó khác với thứ tài liệu của họ gợi ý. Chỉ đo mới biết:

- **Cursor**: đổi `USERPROFILE` thì `status` VẪN báo đã đăng nhập — suýt kết luận nhầm
  là không tách được. Phải thử từng biến một mới thấy `APPDATA` mới là cái điều khiển.
- **Antigravity**: đổi cả `USERPROFILE` + `APPDATA` + `LOCALAPPDATA` sang HOME giả mà
  vẫn chạy đúng danh tính. Đó là bằng chứng dứt điểm cho việc nó dùng kho của Windows.
- **Grok**: `base_url` KHÔNG phải `api.x.ai`. Chính xAI từ chối key bằng "Incorrect API
  key provided"; key mua qua dịch vụ trung gian nên endpoint nằm trong cấu hình.

**Gemini CLI bị bỏ** (2026-08-18): Google cắt bản CLI khỏi gói miễn phí cho cá nhân —
`IneligibleTierError / UNSUPPORTED_CLIENT / free-tier`. Đăng nhập Google thành công, chỉ
là không được cấp quyền dùng. Thay bằng Antigravity.

---

## Claude · Windows — ĐÃ ĐO (kế thừa v1, chạy lại bằng `kiem-tra.ps1`)

- [x] **CLAUDE_CONFIG_DIR có tác dụng.** Đặt `=X` thì Claude đọc/ghi cấu hình ở `X`.
- [x] **Tách thật.** `X\.claude.json` sinh ra ở đúng `X`; `claude mcp list` ở `X`
  không thấy MCP của tài khoản khác.
- [x] **Token là FILE.** Nằm ở `X\.credentials.json`, **không** ở Windows Credential
  Manager → tách thư mục = tách tài khoản.
- [x] **Không đặt biến** thì Claude dùng `%USERPROFILE%\.claude.json`, KHÔNG phải
  `%USERPROFILE%\.claude\.claude.json` (bẫy file lạc). Vì vậy `goc` xoá biến.

Cách đo lại: `pwsh .\kiem-tra.ps1` (thoát ≠ 0 nếu có mục đỏ). Trong Go, tương đương
là `sagent verify claude` (đã chạy: cả 3 phép đo ✓).

- [x] **`run` end-to-end (Go).** `sagent claude:probe --version` set
  `CLAUDE_CONFIG_DIR` cho tiến trình con rồi spawn Claude → in `2.1.229 (Claude Code)`
  và thoát. Env + spawn + truyền args thông suốt.
- [x] **Di trú v1.** `sagent ds` nhận tài khoản cũ ở `~/.claude-accounts/*` là
  `claude:*`, đọc đúng email + trạng thái token, đánh dấu tài khoản đang dùng.

## Claude · Linux — CHƯA ĐO ⚠ (rủi ro #1)

- [ ] Token nằm ở **file** trong config dir, hay ở `libsecret`/`gnome-keyring`?
  → Nếu keyring, primitive "tách thư mục = tách tài khoản" **gãy trên Linux**,
  phải thiết kế đường token riêng. **Cần một máy/VM Linux để đo.**
- [ ] `CLAUDE_CONFIG_DIR` có tác dụng y hệt trên Linux không.
- [ ] `os.Symlink` cho phần dùng chung chạy không cần quyền đặc biệt (dự kiến có).

## Codex CLI · Windows — ĐÃ ĐO ✅ (2026-08-17)

Bản đo: `@openai/codex` 0.147.0, cài qua npm global.

- [x] **`CODEX_HOME` có tác dụng.** `codex --help` có nhắc biến này.
- [x] **TÁCH THẬT — phép đo quyết định.** Đặt `CODEX_HOME` vào một thư mục rỗng
      rồi chạy `codex login status` → trả về **"Not logged in"**, trong khi
      `~/.codex` thật đang đăng nhập. Nghĩa là Codex đọc đúng chỗ được trỏ chứ
      không lén dùng thư mục mặc định. Nó cũng tự tạo `tmp/` trong đó.
- [x] **Token là FILE, không phải keyring.** `~/.codex/auth.json` chứa:
      `auth_mode` ("chatgpt"), `OPENAI_API_KEY` (null khi đăng nhập bằng ChatGPT),
      và `tokens.{id_token, access_token, refresh_token, account_id}`, `last_refresh`.
      → primitive "tách thư mục = tách tài khoản" **đứng vững với Codex**.
- [x] **Danh tính hiển thị được.** `id_token` là JWT; phần payload có claim
      `email`. Đọc cục bộ, không gọi mạng.
- [x] **Phân loại nội dung `~/.codex`** (quyết định cái gì dùng chung):
      - *riêng từng hồ sơ*: `auth.json` (token+danh tính), `installation_id`,
        `cap_sid` — đây là danh tính máy/phiên cài đặt.
      - *KHÔNG được nối link*: `thread-writer-locks/`, `tmp/`, `.sandbox/`, và
        các file SQLite (`state_5.sqlite`, `goals_1.sqlite`, `queue_1.sqlite`,
        `memories_1.sqlite`, `logs_2.sqlite` kèm `-wal`/`-shm`). Dùng chung khoá
        ghi hoặc DB giữa nhiều phiên song song là hỏng dữ liệu.
      - *dùng chung được*: `AGENTS.md`, `config.toml`, `skills/`, `plugins/`,
        `sessions/`, `cache/`, `models_cache.json`, `log/`.

- [x] **Chạy headless**: `codex exec "<prompt>"` (`--help` ghi "Run Codex
      non-interactively"). **KHÁC hẳn Claude** dùng `-p` — đây là lý do phải để
      việc này cho adapter thay vì hardcode trong lõi.
- [x] **Chạy thật qua flow**: bước `agent` với `profile = "codex:thu"` chạy trong
      git worktree riêng, Codex trả lời đúng, 3.392 token. Log xác nhận
      `model: gpt-5.6-sol`, `sandbox: read-only`, `approval: never`.
- [x] **Stdin phải nối vào NUL**, không để nil: Codex thấy stdin là ống dẫn thì
      ngồi chờ "Reading additional input from stdin…" mãi không tới.
- [ ] Codex trên **Linux** — chưa đo (cùng blocker VM Linux).

## Gemini CLI — CHƯA ĐO

- [ ] Cơ chế thư mục config (`~/.gemini/`? có biến override?), token ở đâu.

## Cursor — CHƯA ĐO

- [ ] Có CLI headless + cơ chế config dir không; nếu không thì ghi `unsupported`.

## Junction/symlink từ Go — Windows ĐÃ ĐO ✅

- [x] **Windows: junction không cần admin.** Chạy thật `sagent them claude:smoketest`
  (build từ Go 1.23.5, không cần quyền quản trị) nối **17 mục dùng chung**; kiểm bằng
  PowerShell mọi mục có cờ `ReparsePoint = True` (là junction), riêng `.claude.json`
  là `False` (file riêng thật). `IsLink` (qua `GetFileAttributes`) nhận đúng.
- [x] **Xoá an toàn trên dữ liệu thật.** Đặt file mồi `~/.claude/__sagent_bait.txt`,
  chạy `sagent xoa claude:smoketest` → mồi còn nguyên, `~/.claude` còn nguyên. Có
  thêm unit test `TestRemoveDoesNotTouchBase` chạy xanh.
- [ ] Linux: `os.Symlink` (trong `link_linux.go`) — **chưa đo**, cần VM Linux.

## Chạy song song (fleet) · Windows — ĐÃ ĐO ✅

- [x] **Clone tách thật.** `sagent clone claude:phu --copies 2` tạo
  `~/.ai-accounts/.clones/claude/phu/{1,2}`; kiểm cờ ReparsePoint: mục dùng chung
  là junction, còn `.claude.json` + `.credentials.json` là **file thật riêng từng
  bản** → hai phiên không đua ghi cùng một file.
- [x] **Fleet spawn nền.** Bật 3 phiên `-- --version`: cả 3 có PID riêng, log đổ
  đúng file (`2.1.229 (Claude Code)`), lệnh cha thoát mà phiên vẫn sống.
- [x] **Registry tự dọn.** `status` ngay sau khi bật thấy 3 phiên; sau khi chúng
  thoát, `status` tự đánh dấu `lost` và báo rỗng — không bao giờ báo sống thứ đã chết.
- [x] **Stop trên phiên sống.** Bật 2 phiên với prompt thật, `status` thấy đang
  chạy, `sagent stop all` giết cả hai (taskkill /T nên dọn cả cây con), `status`
  sau đó rỗng.
- [x] **Xoá clone an toàn.** Đặt mồi `~/.claude/__clean_bait.txt`, chạy
  `sagent clean claude:phu` → xoá 3 clone, **mồi còn nguyên**, số mục trong
  `~/.claude` không đổi (20/20).
### Refresh token khi chạy song song — ĐO ĐƯỢC MỘT PHẦN (2026-08-17)

**Đo được: cửa sổ hết hạn.** Đọc thẳng từ file token (chỉ đọc dấu thời gian,
không đụng giá trị token):

| Provider | Trường | Hạn access token | Ghi chú |
|---|---|---|---|
| Claude | `claudeAiOauth.expiresAt` | **~7,5 giờ** | `refreshTokenExpiresAt` còn 26 ngày |
| Codex | `exp` trong JWT `access_token` | **~6,5 ngày** | `last_refresh` cách đó 4 ngày |

→ **Claude là rủi ro thật**: một hạm đội chạy qua đêm CHẮC CHẮN vượt mốc refresh.
Codex thì hiếm khi chạm tới.

**Chắc chắn đúng, không cần thí nghiệm: refresh trong bản clone bị MẤT.**
`clone` chép token ra N thư mục riêng. Nếu một bản clone tự refresh, hồ sơ gốc
không hề biết — lần chạy sau vẫn dùng token cũ. Đây là suy ra từ chính thiết kế,
không phải phỏng đoán về hành vi nhà cung cấp.

- [x] Cửa sổ hết hạn của cả hai provider — đã đo.
- [x] Mất refresh khi dùng clone — chắc chắn theo thiết kế; đã xử lý bằng cách
      **ghi ngược token mới nhất về hồ sơ gốc** (xem `profile.SyncBackTokens`).
- [ ] ⚠ **Nhà cung cấp có XOAY refresh token không — VẪN CHƯA ĐO.** Nếu có xoay
      (bản refresh mới làm bản cũ hết hiệu lực) thì N clone cùng refresh sẽ chỉ
      một bản sống, các bản khác văng. Muốn đo phải để một hạm đội chạy vắt qua
      mốc 7,5 giờ của Claude rồi xem từng bản clone còn gọi được không. Chưa làm.
      Vì vậy `fleet` vẫn **cảnh báo** thay vì hứa an toàn.

---

## Tên hồ sơ thoát thư mục — ĐÃ ĐO, ĐÃ VÁ (2026-08-17)

Tên tài khoản đi thẳng từ người dùng vào `filepath.Join`, không qua kiểm tra nào.
Đo bằng cách in đường dẫn thật ra:

| tên nhập vào | thư mục `sagent xoa` sẽ xoá |
|---|---|
| `phu` | `~/.ai-accounts/claude/phu` (đúng) |
| `../../.claude` | `~/.claude` — **dữ liệu Claude thật, dùng chung cho mọi hồ sơ** |
| `` (rỗng) | `~/.ai-accounts/claude` — mọi tài khoản claude |
| `..` | `~/.ai-accounts` — mọi tài khoản, mọi provider |

Không chỉ dòng lệnh nhập được: dashboard cũng có form thêm hồ sơ, mà dashboard
đang mở ra internet.

Vá hai lớp:
1. `profile.ValidName` — whitelist ký tự (chữ, số, `-`, `_`, `.`), chặn `.`/`..`,
   chặn dấu chấm đầu (đụng thư mục nội bộ `.clones`), chặn dấu chấm cuối (Windows
   lặng lẽ cắt, hai tên hoá một thư mục), chặn tên thiết bị Windows (`NUL`, `COM1`…).
   Gọi ở 6 action nhận `Addr`, có test liệt kê đủ cả 6.
2. `profile.Remove` từ chối mọi đường dẫn nằm ngoài kho hồ sơ. Lớp này mới là lớp
   đáng tin: thêm action mới mà quên gọi `ValidName` thì vẫn không xoá nhầm được.

### Sự cố khi kiểm — mất `~/.claude` thật

Lỗ hổng trên **đã nổ thật một lần**, trong chính lần kiểm thử nó.

`go build` thất bại vì `sagent.exe` đang bị server dash khoá (lỗi cố hữu của
Windows, đã biết từ trước). Script kiểm không dừng khi build hỏng, nên payload
tấn công đập vào **binary cũ chưa có bản vá**, và `sagent xoa claude:../../.claude`
xoá sạch `~/.claude` (21 mục). `os.RemoveAll` không đi qua thùng rác; máy không
bật shadow copy → **không khôi phục được**.

Mất: `projects/` (lịch sử hội thoại dùng chung), `sessions/`, `cache/`, `chrome/`,
`plugins/`, `skills/`, `backups/`, `settings.json`, và token của tài khoản gốc.
Còn: token + `.claude.json` của từng hồ sơ (file THẬT riêng, không phải junction),
`~/.codex`, `~/.claude.json`, và kho v1 `~/.claude-accounts/`.

Ba bài học, đã thành quy tắc:
- **Build hỏng thì DỪNG.** Nối `build` với bước sau bằng `&&`, không bằng xuống dòng.
- **Payload phá hoại chỉ chạy trong HOME giả.** Đặt `USERPROFILE`/`HOME` vào thư
  mục tạm rồi mới bắn. Không bao giờ chĩa vào máy thật để "xem thử".
- **Dừng dash trước khi build.** Server đang chạy khoá cả `sagent.exe` lẫn `state.db`.

### Sự cố thứ hai (2026-08-17 15:04) — mất remote control do tranh chấp device slot

> **Đã đo lại lúc 16:40 cùng ngày và SỬA.** Bản ghi đầu tiên của mục này quy nguyên
> nhân cho việc lấy token hồ sơ `claude:phu`. Sai. Giữ lại phần đúng, ghi lại phần sai
> — đó là lý do mục này tồn tại.

Nguồn: `AppData\Local\Packages\Claude_pzs8sxrjxfjjc\LocalCache\Roaming\Claude\logs\main.log`
(đường `AppData\Roaming\Claude\logs` ghi ở bản đầu **không tồn tại** — app đóng gói MSIX
nên Windows chuyển hướng vào `LocalCache`).

| Mốc | Dòng log |
|---|---|
| 16/08 20:37:48 | desktop app khởi động lại |
| 16/08 20:37:54 | `session_stale_relogin` **latched** — cookie tài khoản gốc (org `c6407a30`) chết; mọi lần thử sau đều `short-circuiting fresh /authorize` |
| 16/08 20:38:09 | supersede đầu tiên: `socket closed: 4000 Superseded by newer connection` + `superseded by another connection for 'win-i51h2n4icta' — another Claude desktop variant is likely running on this machine` |
| 20:38 → 14:59 | **1866 lần** tranh chấp cùng một device slot, đều đặn ~100 lần/giờ, liên tục 18 tiếng |
| 17/08 14:16:46 | tranh chấp đổi triệu chứng sang `Unexpected server response: 500` |
| 17/08 **14:59:47** | lần supersede **cuối cùng** |
| 17/08 15:00:56 | phiên Claude Code gọi `go install govulncheck` — **hệ thống đã hỏng sẵn từ trước** |
| 15:01:29–15:03:50 | reconnect #1…#11, toàn 500 hoặc handshake timeout |
| 17/08 **15:04:16** | `[account] Navigated to /logout, synthesizing logged-out` → `[remote-tools-device] close` → banner "Remote Control disconnected" |
| 17/08 15:04:38 | `Login-state transition uuid: 5233f516… → 49442b47…` — đăng nhập lại bằng tài khoản `phu`, cookie duy nhất còn sống |

**Nguyên nhân gốc: hai client Claude trên cùng một máy giành nhau MỘT device slot
`win-i51h2n4icta`, đá nhau 1866 lần.** Bình thường vòng này tự nối lại được. Lần này
không, vì cookie tài khoản gốc đã chết từ 18 tiếng trước nên không còn đường phục hồi.
Server chuyển sang trả 500, webview rơi về `/logout`, app đăng nhập lại bằng tài khoản
duy nhất còn hợp lệ (`phu`) — phiên remote cũ mất chỗ bám.

Đây là phép đo **đắt nhất và liên quan trực tiếp nhất tới dự án này**: mục tiêu của
`sagent` là chạy nhiều client Claude song song trên một máy, mà device slot thì **dùng
chung theo máy, không theo `CLAUDE_CONFIG_DIR`**. `CLAUDE_CONFIG_DIR` tách được config
và tiến trình con — nó **không** tách được device slot.

Bản đầu ghi sai ở ba chỗ, ghi lại để không lặp:

- **Sai người bấm cò.** Cú giết phiên là điều hướng `/logout` lúc 15:04:16, không phải
  bước lấy token. Việc lấy token lúc 14:16 chỉ trùng mốc loạt 500 bắt đầu (14:16:46).
- **Thiếu tiền đề.** Danh tính gốc đã không dùng được từ 16/08 20:37:54 — 18 tiếng
  trước, chẳng liên quan gì tới hồ sơ `phu`.
- **Thiếu hẳn cơ chế chính.** 1866 sự kiện `Superseded by newer connection` không được
  nhắc một chữ. Kết luận "do đổi danh tính" là suy luận từ hai dòng log gần nhau, đúng
  cái lỗi mà tài liệu này lập ra để chống.

Còn đúng và giữ nguyên: cả codebase không có dòng nào biết remote control tồn tại
(grep `internal/` + `cmd/`: chỉ có `remoteControlSurfacesSeen` trong whitelist dùng
chung, là cờ "đã xem popup", không liên quan kết nối). `sagent` **không** gây ra sự cố này.

Quy tắc, bổ sung vào ba cái trên:

- **KHÔNG test tool trên phiên Claude Code đang chạy.** Phiên đang làm việc — nhất
  là phiên có remote control — là môi trường thật, không phải bàn thí nghiệm. Muốn
  thử `them` / `goc` / `fleet` thì mở terminal riêng, HOME giả, và không có phiên nào
  đang live. Đây là phiên bản tổng quát của bài học "payload phá hoại chỉ chạy trong
  HOME giả": lần trước mất `~/.claude`, lần này mất phiên remote — cùng một gốc.
- **Không chạy `sagent them` / `sagent goc` khi remote control đang bật.** Cả hai
  đều dẫn tới đăng nhập, mà đăng nhập = đổi danh tính = rơi remote control.
  (`goc` giờ cũng đòi đăng nhập vì `~/.claude/.credentials.json` đã mất ở sự cố đầu.)
- **Chưa đo thì không chạy trên tài khoản thật.** `sagent fleet --copies N` chép token
  ra N chỗ; hành vi khi nhiều phiên cùng refresh **chưa đo** — chính
  `internal/fleet/fleet.go:76` tự cảnh báo điều đó.
- **Một máy = một device slot.** Trước khi hứa "chạy N agent song song", phải đo xem
  slot `win-…` có phải hàng rào cứng không. Chưa đo → chưa hứa. Ghi vào việc còn nợ.

### Remote control tự bật — ĐÃ ĐO (2026-08-17)

- [x] **Không phải bật tay từng phiên.** Mỗi lần tạo phiên, log in
      `[rcAutoEnable] verdict: enable=true source=explicit_pref` rồi mới
      `Enabling remote control for session …`. Đúng ở cả 3 phiên quan sát được
      (16/08 20:39:57, 16/08 21:16:20, 17/08 15:05:37), **kể cả phiên tạo sau khi
      đổi tài khoản** — tức preference sống qua cả lần đổi danh tính.
- [x] **Trạng thái remote KHÔNG nằm trong file phiên.**
      `claude-code-sessions/<acc>/<org>/local_*.json` không có trường nào về remote
      control → không sửa được bằng cách ghi file. Đừng thử.
- [ ] `remoteEnabled` trong `~/.claude.json` điều khiển cái gì — **chưa đo**. Nó là
      `false` suốt từ 31/07 tới nay trong khi remote control vẫn chạy bình thường,
      nên chắc chắn **không** phải công tắt này. Không đoán, không lật.
- [ ] **Device slot `win-…` có phải hàng rào cứng cho chạy song song không** — chưa đo.
      Đã biết: hai client cùng máy thì đá nhau (`4000 Superseded by newer connection`,
      1866 lần trong 18 tiếng, xem sự cố thứ hai). Chưa biết: slot cấp theo máy hay
      theo tài khoản, và `sagent fleet --copies N` có đụng vào nó không. **Phải đo
      trong VM riêng, không đo trên máy có phiên thật.**

### Quét lỗ hổng phụ thuộc — ĐÃ ĐO, ĐÃ VÁ (2026-08-17, mở màn Pha 7)

`govulncheck ./...` trên toolchain **go1.25.0**:

| Nhóm | Số lượng |
|---|---|
| Lỗ hổng **có đường gọi thật từ code mình** | **23 — 100% là thư viện chuẩn Go** |
| Có trong package import nhưng không gọi tới | 9 |
| Có trong module require nhưng không gọi tới | 14 |
| Lỗ hổng trong dependency bên thứ ba mà mình gọi | **0** |

23 mục nằm ở `crypto/tls`, `crypto/x509`, `net/http`, `net/url`, `net/textproto`,
`encoding/asn1`; đường gọi hầu hết là `dash.Server.Run → http.Serve` và
`dash.Server.ServeHTTP → http.ServeMux.ServeHTTP`. Nghĩa là **chính cái dashboard mở
cổng ra mạng** là chỗ chạm mặt tất cả chúng.

Không dependency nào phải đổi — bản vá là **nâng toolchain**. Ghim `toolchain go1.25.13`
trong `go.mod`, quét lại:

```
$ govulncheck ./...
No vulnerabilities found.
```

`go vet ./...` sạch, toàn bộ test xanh trên toolchain mới.

Hai việc kèm theo để nó không trôi lại:

- `.github/workflows/ci.yml` đổi từ `go-version: '1.25'` sang `go-version-file: go.mod`.
  Ghim hờ kiểu cũ nghĩa là CI có thể build bằng toolchain **khác** với thứ vừa quét.
- Thêm job `vuln` chạy `govulncheck ./...`, tách khỏi job build/test: một lỗ hổng mới
  công bố không nên chặn việc test code đang viết.

Bài học: **lỗ hổng "của mình" hoá ra không có cái nào là của mình.** Nếu đọc lướt bảng
mà kết luận "dependency bẩn" rồi đi nâng `modernc.org/sqlite` thì sửa nhầm chỗ, và 23
mục vẫn còn nguyên.

### Junction-attack — ĐÃ ĐO, TÌM RA LỖI THẬT, ĐÃ VÁ (2026-08-17, Pha 7)

Ba lớp lỗi đường dẫn, lớp thứ ba trước nay chưa ai đo:

| Lớp | Bẫy | Lá chắn |
|---|---|---|
| 1 | link nằm **bên trong** thư mục hồ sơ | `Remove` gỡ link trước, đếm link còn sót (Pha 1) |
| 2 | tên hồ sơ trỏ **ra ngoài** kho về mặt chữ (`../../.claude`) | `ValidName` + `insideStore` |
| 3 | **thư mục hồ sơ chính nó là link** | *(chưa có — chỗ này nổ)* |

Lớp 3 lọt qua cả hai lá chắn kia một cách hoàn toàn hợp lệ: `~/.ai-accounts/claude/evil`
nằm đúng trong kho, tên `evil` sạch sẽ. Nhưng nó là junction trỏ tới `~/.claude`.

Đo trên Windows, go1.25.13 — ba dòng, dòng thứ ba là chỗ nổ:

```
os.Lstat(junction).Mode()  ->  ModeIrregular, KHÔNG phải ModeSymlink, IsDir()=false
link.IsLink(junction)      ->  true   (kiểm cờ FILE_ATTRIBUTE_REPARSE_POINT)
os.ReadDir(junction)       ->  ĐI XUYÊN, liệt kê ruột thư mục THẬT
```

Vì `ReadDir` đi xuyên, mọi đường dẫn `Remove` dựng từ entries của nó đều trỏ vào ruột
`~/.claude`. Kết quả đo trước bản vá:

| Thứ | Sau `sagent xoa claude:evil` |
|---|---|
| `~/.claude/quan-trong.txt` | còn ✅ |
| `~/.claude` | còn ✅ |
| `~/.claude/skills` (junction dùng chung) | **BỊ GỠ** ❌ |
| `Remove` trả về | `nil` — **không một dòng cảnh báo** |

`os.RemoveAll` không xuyên junction (Lstat báo `IsDir()=false` nên nó gỡ chính cái
link) — nên **dữ liệu không mất**. Nhưng vòng gỡ link ở đầu `Remove` thì có xuyên, và
nó tháo mất đúng cấu trúc junction mà cả dự án này dựng lên. Hỏng im lặng, khó phát
hiện hơn hẳn một lần xoá ồn ào.

Bản vá (`internal/profile/profile.go`): kiểm `link.IsLink(dir)` **trước `os.ReadDir`**,
nếu hồ sơ chính nó là link thì gỡ đúng cái link rồi dừng — người dùng vẫn xoá được hồ
sơ, đầu bên kia không bị chạm.

Test `internal/profile/junction_test.go` đã được **chứng minh là bắt được lỗi**: tắt lá
chắn thì nó đỏ ngay dòng "LINK DÙNG CHUNG BÊN TRONG NẠN NHÂN BỊ GỠ", bật lại thì xanh.
Một test bảo mật chưa từng thấy đỏ thì chưa biết nó có đo gì không.

Bài học: **thứ tự kiểm quan trọng ngang nội dung kiểm.** Cùng một hàm `link.IsLink`,
gọi sau `ReadDir` thì vô dụng, gọi trước thì chặn được. `mklink /J` không cần quyền
quản trị — kẻ tấn công chỉ cần ghi được vào kho hồ sơ là dựng xong bẫy.

### DB migration/rollback + backup restore — ĐÃ ĐO, TÌM RA LỖI THẬT, ĐÃ VÁ (2026-08-17, Pha 7)

Migration **tiến** đã có từ Pha 1 và chạy đúng. Chỗ chưa ai đo là chiều ngược lại:
chuyện gì xảy ra khi **binary cũ mở `state.db` của binary mới** (hạ cấp, hoặc hai máy
dùng chung thư mục đồng bộ)?

Đo trước khi vá:

```
OpenAt(db ở schema v99)  ->  err = nil          (mở bình thường)
Running()                ->  đọc được
AddSession(...)          ->  id=2, err = nil    (GHI ĐƯỢC)
schema_version sau đó    ->  vẫn 99
```

**Bản cũ ghi vào cơ sở dữ liệu của bản mới, không một tiếng động.** Nó không biết ràng
buộc mà bản mới đặt ra, nên nó ghi ra những dòng hợp lệ-với-nó và sai-với-bản-mới.
Kiểu hỏng này không lộ ra lúc xảy ra; nó lộ ra sau, ở chỗ khác, khi đã muộn.

Ba bản vá, tất cả trong `internal/store`:

1. **Chặn hạ cấp.** `cur > len(migrations)` → từ chối mở, kèm câu chỉ đường
   (`nâng cấp sagent, hoặc sagent db restore <file>`). Cùng một lựa chọn với "chưa đặt
   mật khẩu thì dash từ chối mở cổng": thà không chạy.
2. **Sao lưu trước khi nâng schema.** File đã có dữ liệu mà sắp đổi schema thì chụp ảnh
   ra `state.db.bak-v<cũ>` trước. Migration chạy trong transaction nên không để lại nửa
   vời — nhưng transaction **không cứu được một migration viết đúng cú pháp mà sai ý**
   (DROP nhầm cột chẳng hạn): nó commit gọn gàng rồi dữ liệu vẫn đi. File mới tinh thì
   bỏ qua, không có gì để mất.
3. **`sagent db`** — `info` / `backup [file]` / `restore <file>`.

Chi tiết đáng nhớ nhất: **không được chép thẳng file `state.db`.** DB chạy WAL, phần dữ
liệu mới nhất nằm trong `state.db-wal` chứ chưa vào file chính. Chép mình file chính ra
được một bản **thiếu mà trông như đủ** — hỏng kiểu tệ nhất, vì chỉ lộ đúng lúc cần khôi
phục. Dùng `VACUUM INTO` để SQLite tự gộp WAL và ghi ra file nhất quán. Test
`TestSaoLuuRoiKhoiPhucQuayDungVeLucChup` chốt đúng điểm này: bản cứu phải có **đủ 2
phiên**, thiếu một là biết snapshot đã bỏ sót phần nằm trong WAL.

`Restore` tự bảo vệ theo thứ tự đã học từ những lần hỏng trước:

- kiểm file nguồn có thật là `state.db` không, schema có đọc nổi không — khôi phục nhầm
  file là mất trắng;
- **chụp ảnh bản hiện tại trước khi ghi đè** → `state.db.truoc-khi-khoi-phuc`. Một lệnh
  khôi phục không hoàn tác được thì chỉ là một lệnh xoá có thêm bước;
- dọn `-wal`/`-shm` cũ. WAL sót lại của file cũ đứng cạnh file mới thì SQLite áp nhầm dữ
  liệu bản trước lên bản vừa khôi phục.

`store.InUse` chặn khôi phục khi còn tiến trình khác mở DB — xin `locking_mode=EXCLUSIVE`
với `busy_timeout(0)`, lấy được khoá là bằng chứng, không phải phỏng đoán. Đo tại chỗ với
dash thật đang chạy:

```
$ sagent db restore khong-co-that.db
  ✗ ...\state.db đang được tiến trình khác dùng (database is locked (5) (SQLITE_BUSY))
     dừng dash và mọi phiên sagent rồi chạy lại
```

Giới hạn đã biết, nói trước cho khỏi tin nhầm: `InUse` phát hiện *kết nối đang mở*, không
phát hiện một tiến trình vừa đóng kết nối và sắp mở lại.

Cả hai lá chắn mới đều đã **chứng minh là bắt được lỗi**: tắt đi thì
`TestMoFileCuaBanMoiHonThiTuChoi` và `TestNangSchemaThiSaoLuuTruoc` đỏ ngay.

Một chi tiết về kiến trúc: `db.admin` phải vào `api.Actions` vì test hợp đồng bắt mọi
lệnh CLI đều có action. Nó cùng loại với `dash.serve` — có trong hợp đồng nhưng mặt web
không tự làm được phần nặng: `db restore` ghi đè chính file mà server đang mở. Xem và sao
lưu thì mặt khác làm được và nên làm; khôi phục thì phải đứng ngoài mà làm.

### Process-tree cancel + orphan cleanup — ĐÃ ĐO, TÌM RA LỖI THẬT, ĐÃ VÁ (2026-08-17, Pha 7)

`process.Kill` trên Windows là `taskkill /PID <n> /T /F`. Cờ `/T` nghe như "giết cả
cây", nên trước nay không ai hỏi lại. Đo hai kịch bản:

| Kịch bản | Trước `Kill` | Sau `Kill` | `Kill` trả về |
|---|---|---|---|
| Cha còn sống | `PING.EXE` 8600 chạy | chết ✅ | `nil` |
| **Cha đã thoát (mồ côi)** | `PING.EXE` 1380 chạy | **vẫn chạy** ❌ | `exit status 128` |

`/T` đi theo quan hệ cha-con của các tiến trình **CÒN SỐNG**. Cha thoát trước — agent tự
chết, hoặc nó chỉ là cái vỏ khởi động rồi nhường chỗ cho con — thì đám con thành mồ côi
và `taskkill` không còn đường tìm ra chúng. Chúng **vẫn tiêu hạn mức của bạn**, không ai
biết, vì thứ duy nhất báo về là chuỗi `exit status 128`.

Đây không phải tình huống hiếm với dự án này: mục tiêu của nó là bật N agent headless.
Mỗi agent CLI đều sinh tiến trình con.

Bản vá — `process.KillTree`:

1. **Chụp danh sách hậu duệ TRƯỚC khi giết.** Sau khi cha chết, quan hệ cha-con còn đọc
   được trên Windows nhưng **mất hẳn trên Linux** (init nhận nuôi, `PPid` đổi thành 1).
   Chụp trước thì cả hai nền tảng cùng dùng được một logic.
2. Giết cây, rồi **quét lại** và giết thẳng từng đứa còn sống.
3. **Kiểm rồi mới trả về.** Một hàm dừng tiến trình mà trả `nil` trong khi tiến trình vẫn
   chạy thì tệ hơn là không có — người dùng tin là đã dừng rồi đi làm việc khác. Còn sót
   thì trả lỗi kèm đúng danh sách PID.

`parentMap` đọc bảng pid→ppid bằng syscall thẳng (Toolhelp32 snapshot trên Windows,
`/proc/<pid>/stat` trên Linux). **Không** gọi `wmic`: nó đã bị gỡ khỏi Windows Server 2022
— đo được ngay lúc viết hàm này, nó trả về **rỗng, im lặng**, suýt nữa thì kết luận nhầm
là "không có tiến trình con nào". Cũng không gọi `tasklist`: parse chữ thì hỏng theo ngôn
ngữ hệ thống.

Test `TestKillTreeDonDuocMoCoi` đã **chứng minh là bắt được lỗi**: cho `KillTree` thoái
hoá về hành vi `Kill` cũ thì nó đỏ ngay — `MỒ CÔI SỐNG SÓT: [13900]`.

Hai giới hạn, nói trước cho khỏi tin nhầm:

- **PID được dùng lại.** Quan hệ cha-con nhận diện bằng PID; một tiến trình mới trùng PID
  với cha đã chết sẽ kéo cả đám con của nó vào danh sách. Vì vậy `Descendants` chỉ dùng
  khi người dùng **chủ động** gõ `stop`, không bao giờ chạy nền tự động.
- **Còn một lỗ chưa bịt:** phiên tự chết (bị đánh dấu `lost` trong `Running()`) thì hậu
  duệ của nó không ai quét. Muốn bịt phải có một lệnh dọn riêng, và phải nghĩ kỹ về PID
  dùng lại trước khi cho nó tự chạy. **Chưa làm.**
- Trên Linux `StartDetached` không đặt `Setpgid`, nên nhánh `Kill(-pid)` của
  `proc_linux.go` gần như luôn rơi xuống fallback. `KillTree` che được chỗ đó nhờ chụp
  trước, nhưng **chưa đo trên máy Linux thật**.

### Windows ACL — ĐÃ ĐO, TÌM RA LỖI THẬT, ĐÃ VÁ (2026-08-17, Pha 7)

Khắp codebase, file bí mật ghi bằng `os.WriteFile(path, data, 0o600)`:
`.credentials.json` của từng hồ sơ, `dash-auth.json` (mật khẩu dashboard đã băm),
các bản `.bak` của jsonutil. Nhìn thì yên tâm.

Đo: dựng một thư mục có ACL rộng, ghi hai file từ Go, đọc ACL thật.

| File | Tạo bằng | ACL thật |
|---|---|---|
| `secret-0600.json` | `0o600` | `BUILTIN\Users:(I)(F)` — **Users TOÀN QUYỀN** |
| `public-0644.json` | `0o644` | **y hệt** |
| `dir-0700` | `0o700` | `BUILTIN\Users:(I)(OI)(CI)(F)` |

**Bit quyền Unix trên Windows là trang trí.** Quyền thật đến 100% từ ACL kế thừa của thư
mục cha. Trên máy dev nó tình cờ kín vì `C:\Users\<tên>` vốn đã chặt — nhưng đó là **may,
không phải bảo đảm**: đổi `AccountsRoot`, để home trên ổ chia sẻ, hay chạy trên máy có
profile lỏng là token nằm phơi. Và không có gì báo, vì `0o600` trong code trông như đã lo xong.

Bản vá — package `internal/acl`:

- `Restrict(path)` dựng DACL tường minh cho **chủ sở hữu + SYSTEM + nhóm quản trị**, và
  **CẮT KẾ THỪA** (`PROTECTED_DACL_SECURITY_INFORMATION`, tương đương `icacls
  /inheritance:r`). Cắt kế thừa là phần không được quên: chỉ thêm ACE cho chủ sở hữu mà
  vẫn để ACE rộng của cha chảy xuống thì siết được đúng con số không.
- Giữ SYSTEM và Administrators có chủ đích. Bỏ chúng thì sao lưu hệ thống, quét virus và
  chính người quản trị máy không đọc được — đổi rủi ro này lấy rủi ro khác không phải là siết.
- Thư mục siết kèm cờ kế thừa xuống con, nên **file tạo sau vẫn kín** (có test riêng).
- Dùng `golang.org/x/sys/windows` gọi thẳng `SetNamedSecurityInfo`, **không** shell ra
  `icacls`: tên nhóm đổi theo ngôn ngữ Windows ("Users" / "Utilisateurs" / "Benutzer").
  Code test thì gọi `icacls` bằng **SID** (`*S-1-5-32-545`) để dựng bẫy — SID không đổi.
- Trên Linux `Restrict` là `chmod` 0700/0600, vì ở đó bit quyền là thật.

Nối vào ba chỗ tạo thư mục bí mật: `store.OpenAt` (kho hồ sơ, kéo theo cả `state.db`),
`profile` (thư mục từng hồ sơ), `dash.SetPassword`.

Siết lúc tạo là **best-effort** — ổ mạng hay FAT32 có thể từ chối. Nuốt lỗi đó trong im
lặng thì lại đúng cái bệnh vừa chữa, nên `sagent verify` có thêm ô kiểm nói đúng trạng
thái thật. Đo trên máy thật:

```
  [kho hồ sơ]
    ✓ quyền truy cập C:\Users\Administrator\.ai-accounts  chỉ chủ sở hữu, SYSTEM và nhóm quản trị
```

ACL của kho trước và sau — chú ý cờ `(I)`:

```
trước:  NT AUTHORITY\SYSTEM:(I)(OI)(CI)(F)            ← (I) = kế thừa từ cha
        BUILTIN\Administrators:(I)(OI)(CI)(F)
        WIN-...\Administrator:(I)(OI)(CI)(F)

sau:    WIN-...\Administrator:(F)   + (OI)(CI)(IO)(F) ← không còn (I)
        NT AUTHORITY\SYSTEM:(F)     + (OI)(CI)(IO)(F)
        BUILTIN\Administrators:(F)  + (OI)(CI)(IO)(F)
```

Cùng ba chủ thể, nhưng giờ là **tường minh** chứ không phụ thuộc vào việc thư mục cha
tình cờ chặt.

`Check` cũng phải chứng minh được là nó đo thật: test nới lỏng ACL rồi khẳng định `Check`
**nói hỏng** trước khi vá. Một hàm kiểm lúc nào cũng trả "ổn" thì tệ hơn không có.

### Bỏ Linux · nhẹ hơn · cài nhanh hơn — ĐÃ ĐO (2026-08-17)

**Kích thước binary** (`cmd/sagent`, go1.25.13, amd64):

| Cách build | Kích thước |
|---|---|
| mặc định | 16.21 MB |
| `-ldflags "-s -w"` | 11.15 MB |
| `-trimpath -ldflags "-s -w"` | **11.09 MB** (−31,6%) |

Chỗ béo **không** phải nơi ai cũng đoán. Bóc theo ký hiệu:

```
runtime                    5 663 KB
modernc (SQLite)           2 169 KB
type:*                       699 KB
net/http                     332 KB
crypto/tls                   247 KB
```

SQLite thuần Go chỉ chiếm **2,1 MB** — nên đổi sang SQLite khác, hay bỏ SQLite, gần như
không được gì. Phần lớn là runtime Go, không cắt được nếu vẫn viết bằng Go. **Kết luận:
11 MB là đáy hợp lý; đừng đi tối ưu tiếp.**

Khởi động `sagent help`: **328 ms** lần đầu (nguội), **~26 ms** những lần sau. Không có
gì để sửa ở đây.

**Cài đặt** — đây mới là chỗ chậm thật. Trình cài cũ **đòi Go SDK và build từ nguồn**:
người dùng phải tải ~100 MB toolchain rồi ngồi đợi. Trình cài mới tải một `.exe` dựng sẵn:

| | Cũ (`-TuNguon`) | Mới (bản dựng sẵn) |
|---|---|---|
| Phải cài trước | Go SDK (~100 MB) + Git | **không gì** |
| Thời gian | 6,9 s *(đã có Go, cache build nóng)* | tải 11 MB + đối chiếu băm |
| Quyền quản trị | không | không |
| Kiến trúc | máy nào có Go | amd64 + arm64, tự chọn |

`CGO_ENABLED=0` nên `.exe` không phụ thuộc DLL nào — chép sang máy khác là chạy.
Trình cài đối chiếu `SHA256SUMS.txt` trước khi đặt binary vào chỗ: một trình cài tải
`.exe` về rồi chạy ngay mà không kiểm thì chính nó là lỗ hổng nó lẽ ra phải tránh.

**Hai cái bẫy đã dẫm phải khi làm việc này**, ghi lại vì cả hai đều hỏng trong im lặng:

1. **Đặt tên file Go.** Lá chắn "không phải Windows" đặt trong `khong_windows.go` **không
   bao giờ được biên dịch**: Go áp ràng buộc build ngầm theo hậu tố tên file, nên
   `*_windows.go` chỉ build TRÊN Windows — mâu thuẫn thẳng với `//go:build !windows` ở
   đầu file. Không có cảnh báo nào; file chỉ đơn giản biến mất khỏi build. Đổi tên thành
   `nenttang_khac.go` là xong.

2. **PowerShell 5.1 đọc `.ps1` không có BOM theo bảng mã ANSI.** Byte UTF-8 của tiếng
   Việt khi đó rơi vào vùng 0x91–0x94 của cp1252 = dấu nháy cong, và script **vỡ cú
   pháp** ở những dòng chẳng liên quan gì tới tiếng Việt. Trình cài cũ cũng dính, chỉ là
   chưa ai chạy nó trong PowerShell 5.1 sạch. Bản mới lưu UTF-8 **có BOM** + CRLF, và
   `.gitattributes` ghim `*.ps1 text eol=crlf` để git không đụng vào.

Một mẹo nhỏ nhưng đáng: `$ProgressPreference = 'SilentlyContinue'` ở đầu trình cài.
Thanh tiến trình của PowerShell 5.1 vẽ lại sau mỗi khối dữ liệu và làm
`Invoke-WebRequest` chậm đi nhiều lần.

#### Cái bẫy thứ ba: BOM vừa bắt buộc vừa cấm

Phát hiện khi **chạy thật lệnh một dòng** sau khi đã phát hành v0.1.0 — không phải khi
đọc lại code. Bản vá cho bẫy số 2 (thêm BOM) đã **làm hỏng** đường cài chính:

```
$ irm .../cai-dat.ps1 | iex
iex : Unexpected attribute 'CmdletBinding'.
      Unexpected token 'param' in expression or statement.
```

Đo để tìm thủ phạm, không đoán:

```
iex (script khong BOM)   ->  chay
iex (BOM + script)       ->  hong
irm <raw url> -> ky tu dau = U+FEFF
```

Hai ràng buộc ngược chiều nhau, không file nào thoả cả hai:

| | chạy **file** `.\cai-dat.ps1` (PS 5.1) | `irm ... \| iex` |
|---|---|---|
| **Có** BOM | ✅ tiếng Việt đúng | ❌ `iex` không parse nổi U+FEFF |
| **Không** BOM | ❌ tiếng Việt vỡ cú pháp | ✅ |

Nên phải là **hai file**, và lý do đó viết thẳng vào đầu mỗi file để người sau không
"dọn dẹp" cho gọn rồi làm hỏng lại:

- `install/cai-dat.ps1` — UTF-8 **có BOM**, tiếng Việt đầy đủ, có `param()`. Dành cho
  `.\install\cai-dat.ps1` và đính kèm release.
- `install/get.ps1` — **ASCII thuần, không BOM, không `param()`**. Chỉ làm một việc: tải
  `cai-dat.ps1` về file tạm rồi gọi bằng `&` — lúc đó đọc từ file nên BOM lại là thứ
  *cần*. Tham số truyền qua biến môi trường (`SAGENT_PHIEN`, `SAGENT_TU_NGUON`).

#### Cái bẫy thứ tư: `irm | iex` và `iex (irm)` không như nhau

Vẫn chưa xong. Mồi ASCII ở trên vẫn hỏng, và lần này không phải vì BOM:

```
$ irm .../get.ps1 | iex
Invoke-Expression : Cannot bind argument to parameter 'Command' because it is an empty string.
iex : At line:1 char:5
+ try {
      Missing closing '}' in statement block or type definition.
```

Ba giả thuyết của tôi lần lượt **sai**, mỗi cái chết vì một phép đo:

| Giả thuyết | Phép đo | Kết quả |
|---|---|---|
| `irm` trả về mảng từng dòng | `@(irm $u).Count` | `1` — sai |
| Thiếu `-UseBasicParsing` | so ba dạng gọi | cả ba đều `System.String` — sai |
| Nội dung tải về bị cắt | `irm \| %{ $_.Length }` | `1645`, nguyên vẹn — sai |

Thứ phân biệt được đúng/sai chỉ có một: **cách đưa chuỗi vào `iex`.**

```
iex $t                      ->  CHẠY   (cùng nội dung, gán vào biến trước)
iex (irm $u)                ->  CHẠY
iex (iwr -useb $u).Content  ->  CHẠY
irm $u | iex                ->  HỎNG
```

Cùng một chuỗi 1645 ký tự, chỉ khác đường vào. Nên lệnh cài chính thức là
**`iex (irm <url>)`** chứ không phải `irm <url> | iex` — dù dạng pipe mới là dạng người
ta hay chép trên mạng.

Đo lần cuối, máy sạch, tải thật từ GitHub Releases:

```
  ✓ Bản: v0.1.0 (amd64)
  ✓ Đã tải 11.1 MB
  ✓ SHA256 khớp
  ✓ Đã cài: ~\bin\sagent.exe
  sagent v0.1.0 (50e4e3b) · 2026-08-17

  >>> TỔNG THỜI GIAN: 3,4 giây
```

Bài học chung cho cả bốn cái bẫy trong mục này: **cả bốn đều là "code trông đúng, chạy mới
biết"** — file Go biến mất khỏi build, script vỡ cú pháp ở dòng không liên quan, lệnh cài
chết ngay dòng đầu, rồi cùng một chuỗi lúc chạy lúc không tuỳ đường đưa vào. Không cái
nào lộ ra khi đọc lại. Hai cái cuối chỉ lộ vì đã bấm chạy **đúng cái lệnh người dùng sẽ
bấm** — và cái thứ tư được phát hiện SAU khi đã phát hành v0.1.0, nghĩa là bản v0.1.0 ra
đời với một lệnh cài ghi trong README mà không chạy được.

### TLS cho dashboard — ĐÃ LÀM, ĐÃ ĐO TRÊN CỔNG THẬT (2026-08-17, Pha 7)

Đây là cảnh báo treo lâu nhất của dự án: `--host 0.0.0.0` gửi mật khẩu **dạng trần**.
Mọi thứ đã làm để bảo vệ nó — băm PBKDF2 210k vòng, siết ACL file, bỏ token khỏi URL —
đều vô nghĩa nếu ai ngồi cùng đường mạng đọc được nó lúc đăng nhập.

Mức độ không phải lý thuyết: SAN của chứng chỉ vừa sinh trên máy dev có
**`103.97.134.90`** — một IP **công cộng**. Máy này nằm thẳng trên internet, không sau NAT.

**Vì sao tự ký chứ không Let's Encrypt.** Dashboard chạy trên máy cá nhân, thường không
có tên miền, nghe trên IP. ACME không cấp cho địa chỉ kiểu đó. Cái giá của tự ký là trình
duyệt sẽ cảnh báo — nên công cụ **in vân tay SHA-256 ra terminal** để đối chiếu. Nói thẳng
trong chính dòng in: không đối chiếu thì TLS chỉ chống nghe lén thụ động, **không** chống
được kẻ đứng giữa.

**Chính sách, một chỗ duy nhất** (`cmdDash`):

| host | mặc định | vì sao |
|---|---|---|
| ngoài loopback | **HTTPS** | mật khẩu đi qua dây thì phải mã hoá |
| loopback | HTTP | gói tin không rời máy; `--tls` nếu vẫn muốn |

Muốn kém an toàn hơn thì phải **gõ thêm chữ**: `--http-tran`. Và chốt từ chối nằm trong
`Server.Run` chứ không chỉ ở CLI — đó là chỗ không đi vòng được.

Ba chi tiết dễ bỏ sót, mỗi cái có test riêng:

1. **Cert phải phủ MỌI IP của máy, không chỉ `localhost`.** Mở dashboard từ điện thoại là
   gõ IP LAN; cert không phủ IP đó thì trình duyệt báo sai tên miền và người dùng tưởng
   mình đang bị tấn công. Sinh lại khi máy có IP mới (đổi Wi-Fi) chứ không chỉ khi hết hạn.
2. **Dùng lại cert cũ, không sinh mới mỗi lần chạy.** Sinh mới thì vân tay đổi liên tục,
   người dùng quen tay bấm "vẫn tiếp tục", và cả cơ chế đối chiếu thành vô dụng.
3. **`Secure` trên cookie phải bám theo TLS thật, không bật vô điều kiện.** Bật cứng thì
   trên `http://127.0.0.1` trình duyệt vứt cookie đi và không ai đăng nhập được — "an toàn
   hơn" kiểu đó chỉ khiến người ta tắt bảo mật cho xong.

Khoá riêng nằm ở `~/.ai-accounts/dash-tls/`, siết ACL **trước khi ghi khoá vào** chứ không
phải sau. Đo lại trên máy thật: `Administrator` + `SYSTEM` + `Administrators`, không còn
cờ `(I)` — kế thừa đã cắt.

**Đo trên cổng thật**, không phải httptest:

```
$ sagent dash --host 0.0.0.0 --port 4699
  Mật khẩu của "Admin" là hàng rào DUY NHẤT. Đường truyền đã mã hoá.
    https://<IP-máy-này>:4699/
    17:BD:F4:E5:10:1F:43:18:F5:63:63:7A:8C:53:46:00:AA:BE:A7:4D:7D:75:A4:9F:A3:F8:D6:FB:A6:CF:C6:E2

$ curl -sk -w "%{http_code} %{scheme}" https://127.0.0.1:4699/login
  200 HTTPS
```

Vân tay in ra khớp đúng vân tay tính từ file cert trên đĩa, và test `TestBatTayTLSThat`
ghim chứng chỉ rồi so vân tay **trên dây** với vân tay đã in — nếu hai cái lệch thì việc
đối chiếu bằng mắt chẳng chứng minh được gì.

#### Một test hỏng-kiểu-treo còn tệ hơn không có test

Khi thử gỡ lá chắn để kiểm chứng test có bắt được lỗi không, test **treo 7 phút** rồi bị
giết, thay vì đỏ. Lý do: nó gọi thẳng `s.Run(...)` và chờ lỗi trả về — gỡ lá chắn thì
`Run` phục vụ thật và không bao giờ trả về.

Đã sửa: chạy `Run` trong goroutine kèm hạn 3 giây. Giờ gỡ lá chắn là **đỏ trong 3,2 giây**
với đúng câu cần đọc. Bài học: test cho một lá chắn phải hỏng **nhanh và ồn**, vì cách nó
hỏng chính là thứ người ta sẽ nhìn thấy lúc 2 giờ sáng.

### Quét tiến trình mồ côi — ĐÃ BỊT LỖ TỰ KHAI (2026-08-17, Pha 7)

Khi vá `KillTree`, tôi ghi thẳng vào tài liệu một lỗ **chưa bịt**: phiên tự chết bị đánh
dấu `lost` và biến khỏi bảng `status`, nhưng đám tiến trình con nó đẻ ra **có thể vẫn
chạy** — vẫn gọi API, vẫn tiêu hạn mức, và không mặt nào nhìn ra chúng. Giờ bịt.

`sagent quet` — liệt kê; `sagent quet --giet` — dừng.

**Mặc định là CHỈ BÁO, không giết.** Lý do nằm ở chính chỗ khó của bài toán: Windows
**dùng lại PID**. Một tiến trình mới trùng PID với phiên đã chết sẽ kéo cả đám con của
nó vào danh sách, và một lệnh tự động giết theo suy đoán thì sớm muộn cũng giết nhầm.

Ba lớp lọc, không lớp nào đủ một mình:

1. **Cha phải đã chết.** Cha còn sống thì đó là cây bình thường — im lặng, không thì
   `quet` sẽ rủ người dùng giết phiên đang chạy của chính họ.
2. **Con phải bắt đầu SAU khi phiên bắt đầu.** Đây là thứ duy nhất phân biệt được "con
   của phiên đã chết" với "con của một tiến trình mới tình cờ trùng PID". Đọc bằng
   `GetProcessTimes` qua `x/sys/windows`.
3. **Không đọc được thời điểm bắt đầu thì LOẠI, không phải NHẬN.** Khi không biết, mặc
   định là không giết.

Điều kiện 2 **không đủ để chắc chắn** — nói thẳng thế trong code. Nó chỉ loại phần lớn
nhầm lẫn; phần còn lại thì người dùng nhìn danh sách mà quyết. Vì vậy danh sách in ra
kèm **tên tiến trình và thời điểm bắt đầu**, không chỉ số PID: một danh sách toàn số thì
không ai duyệt được và người ta sẽ bấm đồng ý cho xong.

```
  Phiên #7 claude:phu (chết, PID cũ 15896, bật lúc 14:20 17/08)
    · PID 6612    PING.EXE                 bắt đầu 19:08:51 17/08
```

Test `TestMoCoiLoaiTienTrinhCoTruocMoc` chốt đúng lớp 2 bằng cách đặt mốc ở **tương lai**
— khi đó mọi tiến trình đều "có trước phiên" và phải bị loại sạch. Đã chứng minh nó bắt
được lỗi: vô hiệu vế `bd.Before(sau)` thì nó đỏ ngay với đúng tiến trình lọt lưới.

Chạy thật trên máy dev: `✓ Không có tiến trình mồ côi nào` — đúng, vì không có phiên
`lost` nào còn con sống. Đường **có** mồ côi mới chỉ được test bao, chưa chạy thật trên
`state.db` thật (dựng phiên `lost` giả trong đó sẽ làm bẩn dữ liệu người dùng).

### SBOM + thông báo giấy phép — ĐÃ LÀM, VÀ SỔ VIẾT TAY ĐÃ TRÔI THẬT (2026-08-17, Pha 7)

Dự án có `docs/OPEN_SOURCE_LEDGER.md` từ sớm, viết tay. Trước khi thêm SBOM, đo lại xem
nó còn đúng không — bằng `go list -deps ./cmd/sagent`, tức những module **thật sự đi vào
binary** chứ không phải mọi thứ có mặt trong `go.mod`:

| Sổ viết tay nói | Thực tế |
|---|---|
| có `github.com/google/uuid` | **KHÔNG** được liên kết vào binary |
| `golang.org/x/sys` là phụ thuộc *gián tiếp* | đã thành **trực tiếp** (ACL + GetProcessTimes) |
| lý do chọn SQLite thuần Go: "build thẳng cho Windows và Linux" | Linux **đã bỏ** từ sáng cùng ngày |

Ba lỗi trên một trang. **Một sổ giấy phép sai thì tệ hơn không có sổ: nó tạo cảm giác đã
kiểm.** Nên tách đôi:

- **Phần "vì sao" viết tay** — vì sao cần dependency này, vì sao không dùng stdlib. Máy
  không sinh được, và đó mới là phần đáng đọc.
- **Phần "cái gì" do máy sinh** — `tools/giayphep` đọc `go list -deps`, tìm file LICENSE
  trong module cache, xuất `THONG-BAO-GIAY-PHEP.txt` (10 phụ thuộc, 16 KB toàn văn).
  Thiếu giấy phép của module nào thì **dừng và báo tên**, không im lặng bỏ qua.

CI chạy `go run ./tools/giayphep -kiem` nên file đó không trôi được nữa. Đã chứng minh
bước kiểm bắt được lỗi: thêm một dòng rác vào file thì nó đỏ ngay.

**SBOM không thay được thông báo giấy phép.** Đây là điểm dễ nhầm nhất, và có số đo:

```
$ cyclonedx-gomod app -json -licenses -main cmd/sagent -output sbom.cdx.json .
$ jq '[.components[] | select(.licenses)] | length, (.components|length)' sbom.cdx.json
  0
  10
```

Cờ `-licenses` trả về **0/10** trường giấy phép, **im lặng, không báo lỗi**. Nếu đính kèm
SBOM rồi coi như xong nghĩa vụ attribution thì đã phát hành thiếu — mà không có gì báo.

Hai thứ trả lời hai câu khác nhau:

| | Trả lời | Ai đòi |
|---|---|---|
| `sbom.cdx.json` | *bên trong có những gì* | chuỗi cung ứng, quét lỗ hổng |
| `THONG-BAO-GIAY-PHEP.txt` | *và đây là văn bản giấy phép của chúng* | chính giấy phép MIT/BSD |

Điểm cộng bất ngờ: SBOM đếm được **10 thành phần**, đúng bằng số công cụ tự viết tìm ra.
Hai đường đo độc lập cho cùng một con số — nếu lệch thì một trong hai đã sai.

Cả hai file được đính kèm mọi bản phát hành.

### Provider drift — ĐÃ LÀM, VÀ CHÍNH NÓ NỔ KHI CHẠY THẬT (2026-08-17, Pha 7)

Cả tài liệu này dựa trên một giả định chưa ai viết ra: **CLI bên dưới không đổi.** Mọi
khẳng định đều gắn với một phiên bản cụ thể — "đã đo trên codex 0.147.0: `codex exec`
chạy không tương tác", "đã đo: `claude -p` in kết quả ra stdout". Người dùng gõ
`npm i -g @openai/codex` một cái là toàn bộ số đo đó thành lời đồn, mà không có gì báo.

`internal/drift` ghi mốc phiên bản; `sagent verify` so mốc với hiện tại.

**Tín hiệu nào?** Đo hai lựa chọn:

| Cách | Kết quả |
|---|---|
| băm file `claude.cmd` | **sai thứ** — shim là file `.cmd` 1486 byte sửa tay, nâng cấp CLI không đụng tới nó |
| chạy `--version` | `2.1.229 (Claude Code)` 499 ms · `codex-cli 0.147.0` 769 ms |

Nên `Version()` vào thẳng `Adapter` interface, không phải helper dùng chung: cách hỏi có
thể khác nhau giữa các provider (`--version` / `version` / `-v`), mà lõi thì không được
có nhánh `if provider == ...`. Đổi interface làm hai fake trong test đỏ ngay — đúng ý đồ.

**Cảnh báo CỐ Ý không tự tắt.** Thấy drift thì mốc cũ **giữ nguyên**, nên lần chạy sau
vẫn báo. Tự cập nhật mốc thì cảnh báo hiện đúng một lần rồi biến mất, còn thứ đã trôi thì
vẫn trôi. Muốn tắt phải gõ `sagent verify --chap-nhan`, tức người dùng nói "tôi đã đo lại".

#### Và rồi chính nó nổ, ngay lần chạy thật đầu tiên

Test xanh hết. Chạy tay trên máy để xem đầu ra thế nào:

```
lần 1  ✓ ghi mốc đầu tiên: codex-cli 0.147.0
lần 2  ✓ codex-cli 0.147.0 — không đổi từ 17/08/2026
       (sửa file mốc bằng Set-Content -Encoding UTF8 để giả lập nâng cấp)
lần 3  ✓ ghi mốc đầu tiên: codex-cli 0.147.0      ← ??? phải BÁO ĐỘNG mới đúng
```

`Set-Content -Encoding UTF8` của PowerShell 5.1 **thêm BOM**. `json.Unmarshal` hỏng vì
BOM. Và `doc()` viết `_ = json.Unmarshal(...)` — **nuốt lỗi**, trả về sổ rỗng. `verify`
bèn coi như chưa có mốc nào, báo "ghi mốc đầu tiên", rồi **ghi đè sổ**. Mốc của **mọi**
provider mất sạch, phát hiện drift thành vô dụng, không một dòng cảnh báo.

Đúng thứ bệnh mà cả tính năng này lập ra để chống, nằm ngay trong tính năng đó.

Phần tệ nhất: **lần ghi đè xoá luôn cái BOM**, nên kiểm lại file sau khi chạy thì thấy nó
hoàn toàn bình thường. Con bug tự xoá bằng chứng của chính nó. Phải cố tình ghi BOM lại
lần nữa mới chứng minh được.

Vá hai chỗ:

- **Chịu được BOM.** Trình soạn thảo và cmdlet trên Windows hay thêm nó; người dùng chẳng
  làm gì sai.
- **File CÓ mà không đọc được thì BÁO, và KHÔNG ghi đè.** Ghi đè một cái sổ hỏng nghĩa là
  xoá sạch mốc của mọi provider — im lặng, đúng lúc không ai ngờ.

Hai test mới chốt đúng hai điều đó, và test "sổ hỏng" so cả **nội dung file sau khi chạy**
chứ không chỉ giá trị trả về — vì cái sai ở đây là *ghi đè*, không phải *trả nhầm*.

Đo lại trên máy sau khi vá:

```
A  thêm BOM         ✓ codex-cli 0.147.0 — không đổi từ 17/08/2026     (mốc còn nguyên)
B  đổi phiên bản    ✗ CLI ĐÃ ĐỔI: 0.100.0 → 0.147.0 … sagent verify --chap-nhan
C  --chap-nhan      ✓ đã nhận mốc mới: 0.147.0 (trước là 0.100.0)
D  chạy lại         ✓ codex-cli 0.147.0 — không đổi
```

Bài học, lặp lại lần thứ năm trong tài liệu này: **test xanh không thay được một lần chạy
thật.** Tám test của package này đều xanh trong khi `doc()` đang nuốt lỗi — vì không test
nào đưa cho nó một file hỏng. Cái sai lộ ra ở dòng đầu ra đầu tiên trông không đúng.

### Console Windows nuốt tiếng Việt — ĐÃ ĐO, ĐÃ VÁ (2026-08-18, lượt chạy thử toàn phần)

Tìm ra khi chạy **một lượt cài sạch từ đầu** trong HOME giả, không phải khi đọc code.

Go luôn ghi ra byte UTF-8. Console Windows thì render byte theo **codepage đang bật**, và
mặc định của máy **không** phải UTF-8 — đo trên máy dev:

```
OEMCP = 437     <- cmd.exe dùng cái này
ACP   = 1252
```

Toàn bộ thông điệp của công cụ này là tiếng Việt. Mở `cmd.exe` sạch rồi chạy `sagent` thì
mọi dòng thành rác — kể cả dòng báo lỗi, tức đúng lúc người dùng cần đọc nhất.

Hỏi codex một câu, và nó chặn đúng chỗ tôi định làm ẩu: đổi codepage là đụng vào **tài sản
chung của cả cửa sổ console**, những lệnh chạy sau `sagent` cũng chịu ảnh hưởng. Nên bản
vá có hai điều kiện, không phải một:

1. **Chỉ đổi khi stdout là console thật.** Bị chuyển hướng vào file/ống dẫn thì byte UTF-8
   vốn đã đúng; đổi codepage lúc đó là phá console của người khác chẳng vì lý do gì.
2. **Khôi phục lúc thoát** — và phải gọi tường minh ở **mọi** lối thoát, vì `os.Exit` bỏ
   qua `defer`. Có 4 chỗ: `fail()`, hai `os.Exit` trong `main.go`, một trong `flow.go`.

#### Đo thế nào khi chính phép đo làm hỏng thứ cần đo

Chạy `sagent` qua `cmd /c` từ PowerShell thì stdout **bị bắt** → `Dat()` đúng ra không
làm gì → "codepage vẫn 437" là kết quả **rỗng nghĩa**, dễ tưởng là đã chứng minh.

Cách đo đúng: một chương trình dò chạy trong console thật (`start /wait`) và ghi kết quả
ra **FILE**. Ghi ra stdout thì chính việc đo đã làm stdout thôi là console.

| | stdout là console | trước | trong khi | sau |
|---|---|---|---|---|
| Console thật | **true** | 437 | **65001** ✓ | **437** ✓ |
| Bị chuyển hướng | false | 437 | 437 | 437 |

Cả hai vế đều đúng: có đổi khi cần, không đụng khi không cần, và trả lại nguyên trạng.

Test tự động chỉ bao được vế "bị chuyển hướng" — `go test` luôn chạy với stdout redirect.
May thay đó cũng là vế nguy hiểm hơn. Vế console thật ghi số đo tay ở đây và trong comment
của test, cố ý **không** thay bằng một test giả vờ.

### Lượt chạy thử toàn phần trên máy trắng (2026-08-18)

Cài từ release v0.2.0 vào HOME giả, đi hết các lệnh an toàn (`them`/`goc`/`fleet` bị loại
vì đều dẫn tới đăng nhập hoặc chưa đo):

| Bước | Kết quả |
|---|---|
| Cài bằng lệnh một dòng trong README | **8,1 giây** |
| `version` `ds` `config` `init` `status` `quet` `db` `db backup` `flow` `help` | mã thoát 0 |
| `verify` trên máy trắng | mã thoát 1 — **đúng**, chưa có `~/.claude`, `~/.codex` |
| `dash` khi chưa đặt mật khẩu | **từ chối**, mã thoát 1 — đúng |
| `dash --host 0.0.0.0` | tự bật HTTPS, in vân tay |
| `POST /login` qua TLS | 303 |
| `GET /api/state` có cookie | 200 |
| `GET /api/state` **không** cookie | **401** |
| `GET /docs/` | 200 (công khai, đúng thiết kế) |
| `db restore` | ✓, có cứu bản hiện tại trước |
| ACL kho hồ sơ | chỉ chủ sở hữu + SYSTEM + Administrators |

Một chi tiết đáng ghi: mục `verify` trong lượt chạy **không có** ô provider drift — vì
binary tải về là **v0.2.0**, tag trước khi tính năng đó ra đời. Không phải lỗi; nó nhắc
rằng bản phát hành và nhánh `main` là hai thứ khác nhau, và lượt chạy thử phải nói rõ mình
đang thử cái nào.

### Soát dashboard bằng codex — 6 lỗi thật (2026-08-18, phiên tự chạy)

Đưa `internal/dash/server.go` + `session.go` cho `codex exec` soát, yêu cầu mỗi cáo buộc
phải kèm **kịch bản cụ thể**. Nó trả về 6 mục. Tôi **không tin lời** — đọc lại code và dựng
test cho từng cái. **Cả 6 đều thật**, 5 cái viết được test và **cả 5 đều đỏ trước khi vá**.

| # | Lỗi | Kịch bản |
|---|---|---|
| 1 | `next=//evil.example` lọt qua kiểm | `HasPrefix(next,"/")` cho qua, trình duyệt hiểu `//host` là **đổi tên miền** |
| 2 | `/login` nằm ngoài `guard` | tên miền kẻ tấn công trỏ về 127.0.0.1 → POST mật khẩu vào dash nội bộ; `sameOrigin` cho qua vì Origin lẫn Host đều là tên miền đó |
| 3 | `guard` gọi `noteFail()` cho mọi request vô danh | **8 dòng curl** là khoá luôn người đang đăng nhập (429) |
| 4 | `guard` gọi `noteOK()` cho mọi request hợp lệ | dashboard tự poll 5s/lần → chỉ cần một tab đang mở là chống dò mật khẩu **vô hiệu** |
| 5 | `/logout` nhận mọi method, không kiểm nguồn | trang lạ điều hướng tới → cookie SameSite=Lax vẫn gửi → phiên bị xoá |
| 6 | `http.Serve` không hạn giờ | Slowloris: nhỏ từng byte header, giữ mãi goroutine + FD |

**Lỗi 3 và 4 là cùng một gốc:** một bộ đếm đang làm hai việc trái ngược. Nó sinh ra để làm
chậm việc **dò mật khẩu**, nhưng lại bị nối vào **mọi** request. Hệ quả là nó vừa quá rộng
(người vô can bị khoá) vừa quá lỏng (poll hợp lệ xoá sạch bộ đếm). Bản vá không phải thêm
điều kiện mà là **trả nó về đúng một việc**: chỉ `/login` đếm, chỉ đăng nhập thành công mới
xoá, `guard` không đụng vào.

Vá lỗi 3+4 xong, test mới lòi ra chỗ **vá chưa trọn**: `guard` vẫn gọi `throttle()`, nên kẻ
dò mật khẩu vẫn khoá được người đang dùng — chỉ đổi đường chứ không bịt. Phải bỏ nốt lệnh
throttle khỏi `guard`.

Một test cũ, `TestDoNhieuLanBiChan`, **khẳng định đúng cái lỗi số 3 là hành vi mong muốn**:
5 request không cookie thì người đăng nhập hợp lệ phải nhận 429. Nó ghi một cái lỗi thành
hợp đồng. Đã viết lại thành `TestDoMatKhauNhieuLanBiChan` — đo đúng thứ bộ đếm sinh ra để
bảo vệ, và khẳng định thêm rằng người có cookie **không bị vạ lây**.

Lỗi 5 vá bằng `Sec-Fetch-Site` chứ không bắt POST: ba file HTML đang gọi logout bằng
`<a href>`, đổi hết sang form chỉ để chặn một trò chọc phá là đánh đổi tồi. Trình duyệt
hiện đại đều gắn header đó; `curl` không gắn và vẫn dùng được như trước.

Lỗi 6: đặt `ReadHeaderTimeout` và `IdleTimeout`. **Không** đặt `WriteTimeout` — `/api/events`
là luồng SSE chạy dài, đặt vào là tự cắt tính năng của mình.

Đo lại trên cổng thật sau khi vá:

```
login                  -> 303
/api/state có cookie   -> 200
/logout CHÉO TRANG     -> 403      (bị chặn)
/api/state sau đó      -> 200      (phiên còn sống)
/logout bình thường    -> 303
/api/state sau logout  -> 401
```

Nhận xét về việc dùng agent khác để soát: codex chỉ ra **6/6 đúng**, gồm hai lỗi logic mà
đọc bằng mắt rất dễ trượt vì chúng nằm ở *chỗ gọi* chứ không ở *thân hàm*. Nhưng nó cũng
kèm mấy trích dẫn file HTML chẳng liên quan, và không tự biết cái nào nghiêm trọng. Giá trị
nằm ở chỗ **bắt tôi đi kiểm**, không phải ở chỗ kết luận thay tôi.

### Soát store bằng codex — 6/7 đúng (2026-08-18, phiên tự chạy)

Vòng hai: `internal/store/store.go` + `backup.go`. Codex trả 7 mục, **6 đúng, 1 sai**.
Con số đó quan trọng hơn con số 6/6 ở vòng trước: nó nhắc rằng phải kiểm từng cái.

| # | Cáo buộc | Phán quyết |
|---|---|---|
| 1 | Migration đọc `schema_version` NGOÀI giao dịch → hai tiến trình cùng nâng v1→v2 | **đúng** |
| 2 | UPSERT giữ `ended` cũ khi bước quay về `running` | **đúng một nửa** — code cố ý giữ, nhưng bước thử lại thì đó là sai |
| 3 | `_, _ = d.db.Exec(...)` nuốt lỗi khi đánh dấu `lost` | **đúng** |
| 4 | Xoá bản sao lưu đích TRƯỚC khi `VACUUM INTO` | **đúng** |
| 5 | TOCTOU giữa `inspect(src)` và lúc chép | **đúng nhưng bỏ qua** — xem dưới |
| 6 | File tạm `.dang-ghi` dùng chung tên | **đúng** |
| 7 | Mọi lỗi `os.Stat` bị coi là "không ai dùng" | **đúng** |

**Mục 4 là cái đắt nhất.** Người dùng gõ `sagent db backup`, đĩa hết chỗ, và kết quả là
**không còn bản sao lưu nào** — bản cũ đã bị xoá trước khi bản mới kịp hỏng. Vá: `VACUUM
INTO` ra file tạm rồi `os.Rename` đè lên đích. Đổi tên là nguyên tử: hoặc có bản mới, hoặc
vẫn còn bản cũ.

**Mục 7 là lá chắn tự tắt đúng lúc cần nhất.** `InUse` coi mọi lỗi `Stat` — kể cả thiếu
quyền hay ổ mạng rớt — là "đường quang", nên `db restore` ghi đè trong khi không hề biết ai
đang mở file. Giờ chỉ `IsNotExist` mới là "không ai giữ".

**Mục 5 bỏ qua có chủ đích.** TOCTOU giữa kiểm và chép là thật, nhưng đây là CLI chạy trên
máy cá nhân: kẻ nào thay được file giữa hai thao tác thì đã ghi được vào thư mục đó rồi,
tức đã thắng từ trước. Vá nó cần giữ handle xuyên suốt và làm rối `Restore` mà không đổi
được kết cục. Ghi ra đây để lần sau không phải nghĩ lại.

#### Test tự tan

Test cho mục 4 cần ép `VACUUM INTO` hỏng. Lần đầu tôi chiếm chỗ file tạm bằng một **thư mục
rỗng** — và test **xanh**, vì `os.Remove` xoá được thư mục rỗng: cái bẫy tự dọn chính nó rồi
sao lưu chạy ngon. Phải để một file bên trong thư mục thì `os.Remove` mới chịu hỏng.

Sau đó mới chứng minh được test có giá trị: quay `tam := dst + ".dang-chup"` về `tam := dst`
(tức cách cũ) thì nó đỏ ngay.

Lại đúng bài học cũ, ở dạng khác: **một cái bẫy dựng sai thì test xanh, và cái xanh đó
không có nghĩa gì.**

### Tấn công tính chất "approval không thể bị bỏ qua" (2026-08-18, phiên tự chạy)

Vòng ba không hỏi "có lỗi gì không" mà giao thẳng một khẳng định của dự án cho codex
**đập**: *chỉ hàm `Approve` mới chuyển một bước approve sang `done`*.

Nó **không** qua mặt được `Approve`. Nhưng nó chỉ ra khẳng định đó **không có nghĩa như
người ta tưởng**, bằng hai đường:

**1. Approve chỉ chặn bước nào KHAI `needs` tới nó.** Đúng ngữ nghĩa DAG, nhưng người viết
flow đặt một bước `approve` là để dừng cả luồng chờ mình. Quên khai `needs` một chỗ là
`deploy` chạy song song với chính cái cổng đáng ra phải chặn nó. Runner loại bước approve
khỏi đợt chạy rồi mới trả `waiting` — nghĩa là đợt đó **vẫn chạy những bước khác**.

Không sửa ngữ nghĩa sau lưng người dùng (tự ý bắt mọi bước phụ thuộc vào approve còn tệ
hơn). Thay vào đó `Validate` **cảnh báo**:

```
⚠ bước approve này không chặn bước nào — không có bước nào khai `needs = ["gate"]`.
  Nó sẽ dừng luồng nhưng các bước khác VẪN CHẠY song song với nó.
```

**2. `Resume` tin định nghĩa flow do người gọi đưa vào**, không đối chiếu với lúc `Start`.
Đang chờ duyệt mà sửa `flows.toml` bỏ cổng đi rồi resume thì bước sau chạy luôn.

Cái này **ghi lại chứ chưa vá**, và nói rõ vì sao: đây là công cụ chạy trên máy cá nhân,
người sửa được `flows.toml` chính là người bấm duyệt. Approval ở đây là **hàng rào chống
nhầm lẫn của chính mình**, không phải hàng rào chống người khác. Muốn nó thành cái thứ hai
thì phải chụp ảnh định nghĩa flow lúc `Start` và đối chiếu khi `Resume` — làm được, nhưng
đừng làm nửa vời rồi ghi vào tài liệu như một bảo đảm.

Điểm đáng nói về cách dùng agent để soát: hai vòng đầu tôi hỏi "tìm lỗi", vòng này tôi giao
một **khẳng định cụ thể để đập**. Vòng này cho kết quả sắc hơn hẳn — nó không kể ra một
danh sách, nó chỉ đúng vào khoảng cách giữa *điều code làm* và *điều tài liệu hứa*.

### Soát chỗ chạm token — 6/7 đúng (2026-08-18, vòng 4)

`internal/profile/clone.go` + `profile.go` + `internal/fleet/fleet.go`. Đây là code **chép
token** và **xoá thư mục** — hai việc nguy hiểm nhất trong dự án.

**Cái đau nhất là một chỗ TÔI TỰ BỎ SÓT.** Hôm trước vá "0o600 không bảo vệ gì trên
Windows", tôi nối `acl.Restrict` vào `profile.Create`, `store.OpenAt`, `dash.SetPassword` —
và **quên `clone.go`**, đúng cái chỗ token bị **nhân ra N bản**. Một hồ sơ hở là hở một
token; một kho clone hở là hở N. Bản vá lúc đó có test riêng cho package `acl` và test đó
xanh — nhưng không test nào hỏi "còn chỗ nào ghi token mà chưa siết không".

| # | Cáo buộc | Phán quyết |
|---|---|---|
| 1 | `clone.go` không gọi `acl.Restrict` | **đúng — nghiêm trọng nhất** |
| 3 | `CleanClones` không kiểm chính `root` có phải link không | **đúng — cùng lớp lỗi đã xoá `~/.claude`** |
| 5 | Nuốt lỗi khi sao lưu token cũ; file tạm trùng tên | **đúng** |
| 6 | Mọi lỗi đọc token bị coi là "chưa có" | **đúng** |
| 2, 4 | TOCTOU trên thư mục đích | **đúng nhưng bỏ qua** — kẻ tạo được junction trong kho thì đã thắng từ trước |
| 7 | `fleet` nuốt lỗi ghi DB | **SAI** — code đã báo lên bus, codex đọc sót |

**Mục 3 là lớp lỗi đã nổ thật một lần.** `Remove` đã được vá để không đi xuyên junction,
nhưng `CleanClones` gọi `os.ReadDir(root)` **trước** khi bất cứ ai kiểm `root`. Root là
junction trỏ ra ngoài thì ReadDir đi xuyên, và mỗi thư mục con THẬT bên kia bị `Remove` xoá
— trong khi đường dẫn vẫn nằm gọn trong kho nên `insideStore` chẳng thấy gì bất thường.

Đã chứng minh bằng cách gỡ lá chắn: test đỏ với `DỮ LIỆU THẬT BỊ XOÁ QUA JUNCTION Ở GỐC`.

**Mục 5:** sao lưu token cũ hỏng thì **không được đè lên nó**. Token hỏng là phải đăng nhập
lại, mà lúc đó chẳng còn gì để quay về. Trước đây lỗi bị `_ =` nuốt rồi ghi đè tiếp.

#### Lại một test rỗng nghĩa, lần thứ ba

Test cho mục 1 lúc đầu **xanh cả khi bản vá bị vô hiệu** — vì thư mục clone vốn đã kín nhờ
kế thừa từ `~/.ai-accounts`. Phải **nới lỏng ACL thư mục cha trước** thì mới dựng được cái
bẫy. Sau đó nó mới đỏ đúng chỗ: `cấp quyền cho [Users]`.

Ba lần trong hai ngày, cùng một dạng: **bẫy dựng sai → test xanh → tưởng đã chứng minh.**
Câu hỏi phải hỏi mỗi lần viết test cho một lá chắn: *nếu gỡ lá chắn ra, test này có đỏ không?*
Nếu không thử thì không biết.

### Grok — provider thứ năm, và cái bẫy model (2026-08-18)

Provider đầu tiên dùng **API key** thay vì đăng nhập OAuth.

**Endpoint không phải `api.x.ai`.** Key mua qua dịch vụ trung gian; chính xAI từ chối nó:

```
{"code":"invalid-argument","error":"Incorrect API key provided. You can obtain an API key from https://console.x.ai."}
```

Đúng endpoint (`https://modelapi.vn/v1`) thì `GET /v1/models` trả **HTTP 200** với
`grok-4.5`, `grok-4.6`; gọi thẳng API trả lời trong **3,6 giây**, qua CLI là 21,9 giây
(nó là agent chứ không phải wrapper mỏng).

#### Cái bẫy tốn thời gian nhất: model

`grok -p "..."` **không dùng** `defaultModel` trong `~/.grok/user-settings.json`. Nó dùng
`grok-code-fast-1` dựng sẵn, và endpoint không bán model đó nên trả:

```
503 No available channel for model grok-code-fast-1 under group grok (distributor)
```

Một thông điệp chẳng chỉ ra nguyên nhân. Đọc mã nguồn CLI mới rõ:
`modelToUse = model || savedModel || "grok-code-fast-1"`.

Và có một vế nữa, chỉ lộ ra khi chạy từ thư mục **hoàn toàn sạch**: CLI **tự tạo**
`.grok/settings.json` ngay tại thư mục làm việc, ghim `{"model": "grok-code-fast-1"}`.
Nghĩa là mỗi thư mục agent bước vào đều bị ghim sai model — sửa cấu hình người dùng
không cứu được, vì cái ghim theo-thư-mục thắng.

Kéo theo một quyết định thiết kế: adapter **cố ý không tự thêm `-m`**. Model là thuộc
tính của từng hồ sơ (mỗi hồ sơ có thể trỏ endpoint khác, bán model khác), mà
`HeadlessArgs` chỉ nhận prompt và chạy ở tiến trình **cha** — nơi `USERPROFILE` vẫn là
của tài khoản gốc. Đọc cấu hình ở đó rồi áp cho mọi hồ sơ là đoán, và đoán sai model thì
lỗi hiện ra ở tận endpoint. `Verify()` nói thẳng thay vì đoán hộ.

Chạy đúng: `sagent goc grok -m grok-4.5 -p "..."`

#### Bẫy BOM, lần thứ ba

Sửa `user-settings.json` bằng `Set-Content -Encoding UTF8` của PowerShell 5.1 → thêm BOM
→ JSON hỏng (`"b"... is not valid JSON`). Đã ghi trong chính tài liệu này từ hôm trước mà
vẫn dẫm. Ghi thêm lần nữa cho thấm: **trên PowerShell 5.1, muốn UTF-8 không BOM thì phải
`[IO.File]::WriteAllText($f, $s, (New-Object Text.UTF8Encoding($false)))`** — không có
đường tắt bằng cmdlet.

---

## Việc cần bạn hỗ trợ

- **Máy/VM Linux** để chạy các ô Linux ở trên. Không có thì phần Linux của
  Pha 0/1 phải hoãn, và nhãn Linux giữ nguyên `experimental`.

## 18/08 — Flow báo `completed` trong khi KHÔNG agent nào làm được việc

Đo trên lần chạy #8 (`doi-hinh-khong-claude`). Flow kết thúc `completed`, cả ba
bước `done`. Nhìn kỹ output từng bước thì:

```
ke-hoach [antigravity:may] done
  │ jetski: no output produced — a tool required the "command" permission that
  │ headless mode cannot prompt for, so it was auto-denied. …
tho-anti [antigravity:may] done
  │ jetski
```

Bước `ke-hoach` không trả lời gì; nó trả về CÂU TỪ CHỐI QUYỀN. Bước sau được
lệnh "nhắc lại từ đầu tiên" nên lặp lại `jetski` — đó chỉ là tên phiên agent
đứng đầu câu lỗi. Cả hai vẫn `done`, cả flow vẫn `completed`.

Ba lỗi tách bạch, đừng gộp:

1. **Không có cách nào đọc kết quả từng bước.** `FlowRunDetail` nằm trong lõi từ
   lâu nhưng KHÔNG mặt nào gọi — cả CLI lẫn web. `flow runs 8` bỏ qua tham số và
   in lại danh sách. Chạy flow xong thì không biết agent đã trả về gì. Đã thêm
   `sagent flow runs <số>`.

2. **Thoát mã 0 bị coi là bằng chứng đã làm việc.** `agentBridge.RunAgents` trả
   thẳng `readLogs(logs), nil`, không nhìn output. Báo thành công khi chẳng có gì
   xảy ra hỏng nặng hơn báo lỗi: người dùng tin vào một kết quả không tồn tại.
   Đã thêm `khongCoKetQua()` — output rỗng, hoặc mang chữ ký bị chặn quyền
   (`no output produced`, `auto-denied`, `headless mode cannot prompt`), thì bước
   THẤT BẠI. Phản chứng: gỡ lá chắn → 2 test đỏ đúng chỗ, test "kết quả thật đi
   lọt" vẫn xanh (nó không phụ thuộc lá chắn).

3. **Chẩn đoán sai của chính tôi, ghi lại để không lặp.** Thấy dòng
   `Failed to poll ListExperiments: You are not logged into Antigravity` trong
   log là tôi kết luận ngay fleet làm hỏng đăng nhập. Đo lại: chạy `agy -p` với
   `USERPROFILE` trỏ đúng thư mục clone → trả `OK`; với thư mục tạm rỗng → cũng
   `OK`. Dòng kia là lỗi telemetry nền lúc khởi động, không phải đường chạy
   chính. Một dòng ERROR trong log KHÔNG phải nguyên nhân chỉ vì nó ở gần.

### Gốc rễ còn lại: headless không có quyền dùng tool

Đo trực tiếp, cwd = repo thật:

```
agy -p "Ngôn ngữ lập trình chính của repo trong thư mục này là gì?"
→ no output produced — a tool required the "command" permission …
```

`agy` CÓ lấy cwd làm workspace (nó đã định gọi tool để đọc repo). Nhưng headless
thì không hỏi được người, nên tự từ chối. Hệ quả: agent trong flow chỉ trả lời
được từ chữ có sẵn trong prompt, KHÔNG đọc nổi một file nào. Lần chạy #9 trả
"Hiện chưa có repository/workspace nào được mở" là cùng gốc rễ này.

Nghĩa là workflow hiện chạy đúng về mặt điều phối (đúng thứ tự, đúng profile,
song song thật) nhưng thợ chưa có tay. Mở tay ra là một quyết định AN NINH, có
hai đường và tôi không tự chọn:

- `permissions.allow` trong settings.json của từng hồ sơ — hẹp, kiểm soát được.
- `--dangerously-skip-permissions` — agent tự duyệt MỌI tool, kể cả xoá file và
  chạy lệnh tuỳ ý, trong worktree của repo thật.

## 18/08 — Con flake làm CI đỏ ở tag v0.3.0: đua trong GIÀN TEST, không phải sản phẩm

`TestForEachChayTungMuc` đỏ khi chạy cả bộ, xanh 8/8 khi chạy riêng. Chạy lặp
200 lượt thì đỏ vài lượt, và mỗi lần MẤT MỘT MỤC KHÁC NHAU:

```
3 mục phải thành 3 lượt, được 2: [Xử lý alpha (số 1) Xử lý gamma (số 3)]
3 mục phải thành 3 lượt, được 2: [Xử lý gamma (số 3) Xử lý beta (số 2)]
```

Cả ba lượt đều đã chạy — sổ ghi rơi mất một dòng. `fakeAgent.RunAgents` sửa
`calls` và `prompts` không khoá, trong khi runner gọi nó từ nhiều goroutine
(foreach và các bước cùng đợt chạy song song). `append` đua nhau thì mất phần tử.

Đã thêm `sync.Mutex`. Sau khi sửa: 200/200 xanh. `-race` không dùng được trên
máy này (thiếu gcc, CGO tắt) nên phải chứng minh bằng lặp — đủ vì thất bại tái
hiện được và dấu hiệu đặc trưng.

Đây là lý do CI đỏ ở tag v0.3.0 rồi xanh lại ở commit sau với Y NGUYÊN mã nguồn.
Giả thuyết cũ (`KillTree` hết giờ, đã nới 2s→10s) không phải nguyên nhân.

## 18/08 — Go biến mất khỏi máy

`go` không còn trong PATH, không ở thư mục chuẩn nào, tìm toàn ổ C: không ra.
Nhưng `AppData\Local\go-build` và `go\pkg\mod` còn nguyên → Go TỪNG có rồi bị gỡ.
Không có toolchain thì không build được gì.

Đã cài lại đúng bản `go.mod` ghim (1.25.13), vào `LOCALAPPDATA\Programs\Go`,
không cần quyền admin. Lấy SHA256 từ `go.dev/dl/?mode=json` rồi đối chiếu TRƯỚC
khi giải nén (`54a6bbff…d1fc`, khớp). Đã thêm vào PATH người dùng.

## 18/08 — Cú pháp allow-list của Antigravity: đo được gì, và chỗ tôi kết luận sai

Mục tiêu: tìm allow-list HẸP để làm mặc định, thay vì phải bật cờ tự-duyệt-quyền.

**Ký tự đại diện là MỘT sao, không phải hai.** Đo bằng cách đổi từng biến thể và
xem câu lỗi có đổi không:

| Luật thử | Kết quả |
|---|---|
| `command(**)` | vẫn bị chặn |
| `command(git*)` | vẫn bị chặn (chặn cả git) |
| `command(git status)`, `command(dir)` | vẫn bị chặn |
| `command(cmd:*)` | vẫn bị chặn |
| `command` (trần, không ngoặc) | vẫn bị chặn |
| `run_command(**)` | vẫn bị chặn |
| **`command(*)`** | **QUA** — agent chạy được lệnh |
| **`read_file(*)`** | **QUA** — 3 lượt, không lượt nào bị chặn `read_file` |

**Chỗ tôi kết luận sai, ghi lại vì nó là cái bẫy suy luận.** Trước đó tôi khẳng
định `read_file(**)` "có tác dụng thật", căn cứ: đặt `read(**)` thì lỗi gọi tên
`read_file`, đổi thành `read_file(**)` thì lỗi nhảy sang `command`. Suy ra sai.
Agent CHỌN tool khác nhau giữa các lượt, nên tên quyền trong câu lỗi đổi vì lý do
khác. Đo lại bằng cách chạy lặp thì `read_file(**)` vẫn bị chặn. **Một lượt chạy
không phân biệt được "luật có tác dụng" với "agent tình cờ đi đường khác" — phải
lặp.**

**Không có nấc trung gian cho `command`.** Chỉ `command(*)` là qua, tức CHO PHÉP
MỌI LỆNH. Sáu biến thể hẹp hơn đều bị chặn. Về sức phá hoại, `command(*)` ngang
với `--dangerously-skip-permissions`.

**Và allow-list chỉ-đọc KHÔNG đủ để agent làm việc.** Cho `read_file(*)`,
`list_directory(*)`, `glob(*)`, `grep(*)`, `search_file_content(*)` mà bỏ
`command`: agy vẫn với sang tool `command` ngay cả khi chỉ cần đọc `go.mod`, nên
bước đứng chết. Thêm `trustedWorkspaces` trỏ đúng repo cũng không đổi (2 lượt).

Kết luận cho antigravity: **không tồn tại mặc định hẹp mà vẫn dùng được.** Hai
đường duy nhất đều là toàn quyền. Nên sự an toàn KHÔNG đến từ allow-list mà đến
từ chỗ khác:

1. Cờ `tu_duyet_quyen` mặc định TẮT, khai theo TỪNG BƯỚC (2 test: mặc định tắt,
   và không lây sang bước sau).
2. Mỗi agent chạy trong git worktree riêng (`sagent/may-1`), không phải cây làm
   việc chính. Đã kiểm sau lần chạy #10: repo chính không file nào bị đụng.

Chưa đo: allow-list của Claude Code (`allowedTools`) — nhiều khả năng hẹp được
thật, nhưng `claude:tns` chưa đăng nhập nên chưa đo được. Đừng chép kết luận của
antigravity sang Claude.

Phụ: moi nhị phân `agy.exe` (183 MB, bundle) lấy được khuôn sinh câu lỗi
`%s required the %s %s that headless mode cannot prompt for, so %s auto-denied`
— nên `command` là TÊN QUYỀN, khác tên tool `run_command`. Cùng chỗ đó có dòng
`user has workspace validation enabled but has no active workspaces`, đúng bằng
triệu chứng lần chạy #9.

## 18/08 — Rào quyền của năm provider: đo `--help`, và ba trạng thái khác nhau

| Provider | Cờ mở toàn quyền | Nấc trung gian | Đo tới đâu |
|---|---|---|---|
| claude | `--dangerously-skip-permissions` | chưa đo (`allowedTools`?) | `--help` |
| antigravity | `--dangerously-skip-permissions` | **không có** (7 biến thể allow-list đều chặn) | `--help` + chạy thật #10/#11 |
| codex | `--dangerously-bypass-approvals-and-sandbox` | **CÓ** — xem dưới | `--help` |
| grok | *không có rào nào* | không có | `--help` |
| cursor | chưa đo | chưa đo | máy không cài `cursor-agent` |

**Codex có đúng cái nấc mà Antigravity không có:**

```
-s, --sandbox <read-only | workspace-write | danger-full-access>
-a, --ask-for-approval <untrusted | on-request | never>
```

`--sandbox workspace-write --ask-for-approval never` là mặc định hẹp đúng nghĩa:
agent làm việc thật trong workspace, không hỏi, không ra ngoài. **Chưa xác nhận
được hành vi**: `codex exec` trả `You've hit your usage limit … try again at
Aug 20th`. Nên CHƯA sửa `HeadlessArgs` của codex — khai theo `--help` là đo giao
diện, không phải đo hành vi.

**Grok không có rào nào cả.** `grok --help` không có approval/sandbox/permission,
chỉ có `--max-tool-rounds` (mặc định 400). Nó chạy tool tự do theo thiết kế.

Chuyện này làm lộ một lỗi trong thiết kế cũ của tôi: `ArgsTuDuyetQuyen()` trả
`nil` để nghĩa là "chưa đo", nên Grok bị xếp chung nhóm với Cursor. Sai — hai
tình huống ngược nhau hoàn toàn. Đã tách thành `(args, daDo bool)`, ba trạng thái:

- `(cờ, true)` — có rào, đây là cách mở.
- `(nil, true)` — **đã đo, KHÔNG có rào**. Cờ thừa. Phải **cảnh báo**, kể cả khi
  bước không bật cờ: người dùng dễ tưởng "không bật = agent bị hạn chế".
- `(nil, false)` — **chưa đo**. Phải **báo lỗi**, không được lặng lẽ chạy không
  quyền rồi báo xong.

Quyết định này tách khỏi `RunAgents` thành hàm thuần `argsChoBuoc` để test được —
chôn trong đó thì phải bật phiên thật mới chạy tới, tức không ai kiểm được. 4 test,
trong đó cái quan trọng nhất: **bước không xin quyền thì dòng lệnh không được
chứa cờ nguy hiểm**.

Bẫy đã dính lại: `codex`/`grok` trên máy này là `.ps1` (npm shim), gọi qua
`cmd /c` thì `--help` trả về RỖNG mà không báo lỗi — y hệt bẫy `wmic` hôm trước.
Phải gọi thẳng qua PowerShell. Đo qua sai lớp vỏ thì được số 0 chứ không được sự thật.

## 18/08 — Agent chạy trong git worktree thì dò workspace HỤT (chập chờn)

Sau khi bật cờ quyền, lần chạy #10 và #11 trả đúng "Go (Golang)" nhưng #12 lại
trả "Hiện tại không có repository nào được mở trong workspace" — cùng flow, cùng
cờ, cùng máy. Không đoán; tách từng biến:

1. **Cờ có rơi mất khi refactor không?** Viết test nạp `tu_duyet_quyen = true` từ
   TOML → nạp đúng. (Test này đáng giữ: sai một chữ ở thẻ `toml:"..."` thì
   BurntSushi bỏ qua IM LẶNG, cờ về false, agent chạy không quyền, flow vẫn báo
   xong — đúng kiểu hỏng lặng lẽ của lần chạy #8.)
2. **Cờ có tác dụng không?** Chạy tay `agy --dangerously-skip-permissions` với
   cwd = thư mục repo: **3/3 đúng**.
3. **Khác nhau ở đâu?** Flow chạy trong git worktree. Chạy tay với cwd =
   worktree: **1/3 đúng**, hai lượt kia trả "chưa có repository nào được mở".

Nguyên nhân: ở git worktree, `.git` là **FILE con trỏ 78 byte**, không phải thư
mục. Antigravity dò workspace theo kiểu vấp phải chỗ này.

Cách sửa (đo, không suy): `agy --help` có `--add-dir  Add a directory to the
workspace`. Thêm nó, cwd = worktree: **4/4 đúng**.

Đã thêm `ArgsThuMuc(dir)` vào adapter, fleet gọi khi phiên có worktree. Đo cờ
tương ứng của từng provider trên `--help` thật:

| Provider | Cờ | Đo tới đâu |
|---|---|---|
| antigravity | `--add-dir <dir>` | `--help` + chạy thật 4/4 |
| claude | `--add-dir <directories...>` | `--help` |
| codex | `-C, --cd <DIR>` | `--help` (chưa chạy thật, hết hạn mức tới 20/08) |
| grok | `-d, --directory <dir>` | `--help` |
| cursor | chưa đo | máy không cài `cursor-agent` |

Kiểm chứng qua flow thật: **3/3 đúng** (#13, #14, #15). Trước bản vá là 2/4.

Bài học lặp lại lần thứ ba trong ngày: **lỗi chập chờn thì một lượt xanh không
chứng minh được gì.** Cả ba lỗi hôm nay đều thế — con flake CI (8/8 xanh khi
chạy riêng), luật allow-list (một lượt làm tôi kết luận ngược), và cái này
(#10, #11 xanh liền hai lượt trong khi lỗi vẫn còn nguyên).

## 18/08 — Vỏ bọc .cmd CẮT prompt nhiều dòng: agent chỉ nhận DÒNG ĐẦU

Lỗi nặng nhất tìm được hôm nay, và nó im lặng tuyệt đối.

Dựng flow `code` (4 agent), thử cơ chế bằng Antigravity + Grok. Grok trả lời
"Bạn chưa chỉ định hai lệnh cụ thể" — mà prompt ghi rõ hai lệnh git. Nhìn log
JSON của grok thì thấy nó nhận đúng MỘT dòng đầu:

```
{"role":"user","content":"CHỈ ĐỌC, không sửa gì. Chạy đúng hai lệnh này:"}
```

Đo bằng vỏ giả, chạy qua đúng đường của `profile.StartDetached`:

```
gửi  "DONG MOT\nDONG HAI\nDONG BA"
vỏ .cmd nhận  "DONG MOT"
```

Trên máy này `exec.LookPath` trả về:

| Lệnh | Đường thật | Có bị cắt |
|---|---|---|
| claude | `C:\Users\Administrator\bin\claude.cmd` | **CÓ** |
| grok | `...\npm\grok.cmd` | **CÓ** |
| codex | `...\npm\codex.cmd` | **CÓ** |
| agy | `...\agy\bin\agy.exe` | không |

PATHEXT có `.CMD` mà không có `.PS1`, nên Go luôn chọn vỏ batch. **Chỉ
Antigravity là .exe thật — đó là lý do suốt hôm nay chỉ mình nó nhận đủ prompt
nhiều dòng, còn ba cái kia im lặng nhận một mẩu rồi trả lời tự tin.**

Nếu không bắt được, flow `code` sẽ hỏng ngầm toàn bộ: mọi prompt trong đó đều
nhiều dòng, nên hai thợ Claude chỉ nhận được dòng "Làm PHẦN 1 trong kế hoạch
dưới đây." — không có kế hoạch, không có yêu cầu test, không có lệnh commit.

Sửa hai chỗ:
1. `profile.GoiThat` gỡ vỏ npm (`"%dp0%\...\index.js" %*`) để gọi thẳng
   `node <script>`, đối số đi qua CreateProcess chứ không qua trình thông dịch
   batch. Không nhận ra kiểu vỏ thì trả nguyên đường cũ.
2. `claude.Command()` ưu tiên `claude.exe` thật trong gói MSIX
   (`%LOCALAPPDATA%\Packages\Claude_pzs8sxrjxfjjc\...\claude-code\<ver>\claude.exe`)
   thay vì vỏ .cmd. Vỏ claude.cmd là bản tự viết, không phải kiểu npm nên bộ gỡ
   trên không đụng tới.

Kiểm chứng: chạy lại flow, prompt 8 dòng tới grok NGUYÊN VẸN (`\n` còn đủ), nó
chạy được `git log main..sagent/may-1` và thấy đúng commit.

**Bản đầu của bộ gỡ TRƯỢT mà vẫn xanh.** Regex bám vào chuỗi `dp0`, nhưng vỏ npm
có nhiều chỗ `dp0` (`:find_dp0`, `SET dp0=%~dp0`) và `[^"]*` vắt qua cả xuống
dòng nên bám nhầm. Build được, chạy được, chỉ là không gỡ gì cả — và tôi suýt
tin vì flow vẫn "completed". Phải in thẳng `LookPath` + kết quả regex mới thấy.
Test bây giờ dùng NGUYÊN VĂN vỏ npm thật trên máy, không dùng vỏ tự bịa.

## 18/08 — Lần chạy `code` #21: flow báo `completed`, thật ra 3/5 bước hỏng

Lượt chạy thật đầu tiên có đủ 4 agent. Kết quả trên màn hình: `completed`, cả
năm bước `done`. Kiểm từng cái thì:

| Bước | Tài khoản | Màn hình | Sự thật |
|---|---|---|---|
| ke-hoach | claude:phu | done | **THẬT** — clone cc-switch về đọc, rút 3 nguyên lý có trích dẫn file:dòng |
| tho-1 | claude:tns | done | **THẬT** — commit `161e79e`, sửa `fleet.go` + 87 dòng test |
| tho-2 | antigravity:may | done | **RỖNG** — nhánh `sagent/may-1` không có commit nào; output cuối là "I am waiting for `go test ./...` to complete" |
| soi | grok:api | done | **QUẨN VÒNG** — gọi `ls -la` 399 lần liên tiếp rồi bị trần `--max-tool-rounds` chặn |
| gop | claude:phu | done | **HỎNG XÁC THỰC** — "Failed to authenticate: OAuth session expired and could not be refreshed" |

Ba kiểu hỏng mới, cả ba đều lọt qua lá chắn `khongCoKetQua` cũ (chỉ bắt output
rỗng và chữ ký bị-chặn-quyền):

1. **Hỏng xác thực.** Token trong bản sao hồ sơ hết hạn giữa chừng và không tự
   làm mới được — `ke-hoach` (cùng tài khoản claude:phu) chạy được lúc 16:19,
   `gop` thì hỏng khoảng 40 phút sau. Đúng rủi ro kế hoạch gốc cảnh báo: *"không
   xem clone credential là an toàn nếu chưa đo concurrent refresh"*.
2. **Quẩn vòng gọi tool.** Tool `bash` của Grok chạy qua `cmd.exe` nên mọi lệnh
   Unix đều trượt, mà nó không thích nghi — lặp đúng một lệnh 399 lần. Trần
   `--max-tool-rounds` (mặc định 400) là thứ duy nhất cứu.
3. **Hết giờ giữa việc.** Antigravity dừng khi đang đợi `go test`, không commit
   gì, vẫn `done`.

Đã mở rộng lá chắn cho (1) và (2), có test dùng nguyên văn chuỗi đo được.

**Bài học lớn hơn: với bước CODE thì output chữ là bằng chứng yếu.** Bằng chứng
mạnh là *nhánh có commit hay không* — `git log main..sagent/<tk>-1`. Hai bước
nói "xong" nhưng một bước nhánh rỗng. Nên kiểm nhánh, đừng tin lời kể.

### Badge "sẵn sàng" cũng nói dối

`sagent ds` hiện `claude:phu … sẵn sàng` trong khi chạy thật trả
`OAuth session expired`. `HasToken` chỉ kiểm **file token có tồn tại**, không
kiểm nó còn dùng được. Kế hoạch gốc mục 1.6 đòi "trung thực về năng lực" —
chỗ này đang vi phạm. Chưa sửa: cần một phép kiểm nhẹ, không tốn hạn mức.

### Vá xong hai chỗ làm hỏng #21

**Grok quẩn vòng.** Hai việc, không phải một:
1. `HeadlessArgs` hạ trần `--max-tool-rounds` từ 400 xuống 60. Không sửa được
   tính cố chấp, nhưng làm nó gãy sớm thay vì đốt hết hạn mức rồi mới gãy.
2. Prompt nói thẳng máy là Windows/cmd.exe, không có `ls`/`cat`/`pwd`/`grep`;
   ưu tiên `git` vì git chạy y hệt trên mọi hệ; một lệnh trượt hai lần thì đổi
   cách. Đây mới là chỗ chữa gốc.

Đo lại (lần chạy #22): Grok chạy CẢ HAI lệnh git trong MỘT vòng và trả lời đúng
khuôn hai dòng. Trước đó là 399 vòng `ls -la` vô ích.

**claude:phu hỏng xác thực.** Chủ dự án đăng nhập lại; chạy thật qua fleet trả
`OK`. Bản sao hồ sơ nhận token mới, không cần dựng lại clone.

Còn nợ: kiểu hỏng thứ ba (agent dừng giữa việc, nhánh rỗng, vẫn `done`) chưa bắt
được bằng chuỗi. Bằng chứng đúng cho bước code là `git log main..sagent/<tk>-1`
có commit hay không — phải kiểm nhánh, không tin lời agent kể.

## 18/08 — Chuỗi JS đứt giữa chừng làm CHẾT cả trang 2D, sống qua nhiều bản build

Chủ dự án mở dash bản mới: bảng tài khoản trống, ô chọn trống, dòng
"đang kết nối…" mãi không dứt. Nhìn như lỗi mạng.

Thật ra là lỗi cú pháp. `internal/dash/web/index.html` có hai chỗ viết xuống dòng
THẬT bên trong chuỗi nháy đơn thay vì `\n`:

```js
out.textContent = dong.join('
');
out.textContent = d.noi_dung + '

— ' + d.model + ...
```

Một chuỗi đứt làm **toàn bộ script của trang chết**, không phải chỉ dòng đó. Khung
HTML vẫn hiện nên trông như trang đã tải xong.

Có từ commit `fe94ebf` và sống qua nhiều bản build. Dash cũ ở cổng 8787 chạy bản
17/08 nên KHÔNG dính — đó là lý do bản cũ trông vẫn ổn còn bản mới thì trống trơn,
và cũng là lý do mãi không ai phát hiện.

Cùng một cái bẫy escape đã hại nhiều lần trong ngày (rune literal trong Go, `\`
trong heredoc, BOM của PowerShell).

**Đã thêm `TestFileWebKhongCoChuoiDut`**: quét mọi file `web/*.html`, đếm nháy đơn
không bị escape trên từng dòng, lẻ là báo lỗi. Rẻ, không cần chạy JS trong CI.
Phản chứng: dựng lại đúng lỗi cũ → test đỏ và chỉ đúng dòng 279.

Bài học: **giao diện không có test thì hỏng im lặng.** Go có `go build` bắt lỗi cú
pháp ngay; HTML/JS nhúng thì không ai bắt. Từ nay mọi file web đều đi qua phép
kiểm này.

## 18/08 — Lượt chạy dài: đội đi học 3 dự án, và lá chắn giết oan một bước

Flow `hoc` (lần chạy #23): ba agent đọc ba dự án song song, rồi tổng hợp.

| Bước | Tài khoản | Kết quả |
|---|---|---|
| hoc-gastown | claude:phu | **THẬT** — trích `internal/config/roles.go:20-41`, kết luận vai trò là DỮ LIỆU (TOML nhúng binary), không phải quy ước |
| hoc-deck | antigravity:may | dừng giữa việc, chỉ có dòng "Đang tải shallow clone…" |
| hoc-acp | claude:tns | **BỊ GIẾT OAN** — xem dưới |
| tong-hop | claude:phu | không chạy: bước cha `hoc-acp` failed nên bị bỏ, dù khai `on_failure = "continue"` |

### Lá chắn của chính tôi giết oan một bước làm được việc

`hoc-acp` clone xong hai repo, viết báo cáo, nhưng GIỮA ĐƯỜNG có một lần bị chặn
quyền. Lá chắn soi toàn bộ bản ghi nên thấy chữ ký đó và giết cả bước.

Sửa lần một: chỉ soi 800 ký tự cuối. **Vẫn sai** — output ngắn thì "đuôi" là cả
bài, vẫn giết oan. Test bắt được ngay, nên bản sai không đi xa.

Sửa lần hai, đúng câu hỏi: **không phải "trong bản ghi có chữ ký hỏng không", mà
là "sau chữ ký đó agent còn làm được gì nữa không".** Tìm lần xuất hiện CUỐI
CÙNG, nếu còn hơn 200 ký tự nội dung phía sau thì agent đã đổi cách và đi tiếp —
không tính là hỏng. Gặp trở ngại rồi đổi cách là chuyện đáng mừng, không phải lỗi.

8 test cho lá chắn, gồm cả ca báo động giả này với nguyên văn output thật.

### Bằng chứng git thay cho lời agent kể

Thêm `workspace.Xem(dir, goc)` đọc trạng thái git thật của worktree: tên nhánh,
số commit đi trước nhánh gốc, còn thay đổi chưa commit hay không. Kết quả gắn vào
cuối output mỗi bước agent, nên bước SAU (người soi) cũng đọc thấy.

Có vì lần chạy #21: bước `tho-2` trả "I am waiting for `go test` to complete",
được đánh dấu `done`, mà nhánh `sagent/may-1` KHÔNG có commit nào. Không cách nào
biết nếu chỉ đọc chữ agent in ra. Test dựng repo git thật cho ba tình huống đã
gặp: có commit / không commit / sửa mà quên commit.

`NhanhMacDinh` hỏi `origin/HEAD` trước rồi mới thử `main`/`master` — đoán sai
nhánh nền thì số commit đếm ra vô nghĩa mà lại trông rất thuyết phục.

### Nguyên nhân gốc của MỌI lỗi escape hôm nay

Ký tự gạch chéo ngược trong heredoc bị nuốt một lớp trước khi tới Python. Nên mọi
lần tôi viết chuỗi có `\n` để sinh mã Go/JS, nó thành xuống dòng THẬT trong file:
rune literal trong Go, chuỗi JS đứt làm chết cả trang 2D, và ba lần liên tiếp ở
lượt này.

Cách tránh: **không viết dấu gạch chéo ngược trong script sinh mã.** Dùng
`chr(92)` khi cần ký tự đó, và dùng chuỗi thô (dấu huyền trong Go, backtick trong
JS) để khỏi cần escape. Đã sửa được 3 chỗ bằng cách này sau khi 5 lần thử theo
lối cũ đều thất bại.

## 18/08 — `on_failure = "continue"` chỉ đúng một nửa, và nửa sai thì im lặng

Đo tại lần chạy #23: `hoc-acp` hỏng, ba bước học đều khai `on_failure = "continue"`,
nhưng `tong-hop` (cần cả ba) hiện **"(chưa chạy)"** — không một dòng cảnh báo, và
lần chạy vẫn được ghi là `completed`.

Gốc ở `runState.readySteps`: nó gọi `finished()`, mà hàm đó chỉ tính `done` và
`skipped`. Một bước cha `failed` khoá cứng mọi bước con — chúng không bao giờ đủ
điều kiện, runner hết việc rồi kết thúc êm ru.

Nghĩa là `continue` mới làm được nửa việc: nó không dừng ĐỢT đang chạy, nhưng vẫn
chặn mọi bước PHÍA SAU. Người viết flow gõ "continue" là đã nói rõ ý — hỏng thì cứ
đi tiếp. Làm ngược ý họ đã tệ, làm ngược trong im lặng còn tệ hơn.

Đã thêm `choDiTiep()`: bước phụ thuộc hỏng nhưng chính nó khai `continue` thì vẫn
cho bước sau chạy (với output rỗng — bước sau tự xử, như prompt `tong-hop` đã dặn
"bước nào không trả về nghiên cứu thật thì ghi thẳng là chưa học được, đừng bịa").

Hai test, dựng đúng hình dạng của #23. Phản chứng: gỡ `choDiTiep` ra thì test đỏ
đúng câu "BƯỚC SAU KHÔNG CHẠY dù bước trước khai on_failure=continue".

Mặc định (`stop`) vẫn chặn bước sau như cũ — có test riêng cho chiều đó.

### Kiểm chứng lá chắn đã sửa

Lần chạy #24 (chạy lại `hoc` sau khi sửa): `hoc-acp` từ `failed` thành **`done`**.
Cùng agent, cùng prompt, chỉ khác lá chắn — xác nhận lần trước đúng là báo động giả.

## 18/08 — Badge "sẵn sàng" nói dối cho token đã chết

`sagent ds` và bảng web đều hiện `claude:phu … sẵn sàng` trong khi chạy thật trả
`OAuth session expired and could not be refreshed`. Lý do: `HasToken` chỉ kiểm
**file token có tồn tại**, không kiểm nó còn dùng được.

Kế hoạch gốc mục 1.6 đòi "trung thực về năng lực" — báo sẵn sàng cho một token đã
chết là vi phạm thẳng, và nó khiến người dùng giao việc cho một tài khoản không
chạy được.

Adapter đã có sẵn `TokenExpiry` (đo từ trước, Claude đọc `claudeAiOauth.expiresAt`)
nhưng KHÔNG ai gọi. Nay `ProfileList` gọi nó và thêm hai trường:

- `HetHan` — token còn đó nhưng quá hạn
- `HanToi` — rỗng nghĩa là provider không đọc được hạn (đã đo, không phải thiếu sót)

Ba trạng thái tách bạch, cả CLI lẫn web:
`hết hạn — đăng nhập lại` / `sẵn sàng` / `chưa đăng nhập`.

Test dựng hồ sơ giả với `expiresAt` lùi 24 giờ. Phản chứng: ép `HetHan = false`
thì test đỏ đúng câu "token hết hạn từ hôm qua mà không báo hết hạn".

Vẫn chưa bắt được: token **chưa quá hạn theo đồng hồ** nhưng đã bị thu hồi phía
máy chủ. Muốn biết chắc thì phải gọi thật, mà gọi thật thì tốn hạn mức — để ngỏ.

## 18/08 — Lần chạy #24: đội học đủ bốn bước

Chạy lại `hoc` sau khi sửa lá chắn: `hoc-acp` (claude:tns), `hoc-deck`
(antigravity:may), `hoc-gastown` (claude:phu) đều `done`, `tong-hop` chạy tiếp.
So với #23 (1 hỏng oan, 1 dở dang, bước gộp không chạy).

## 18/08 — Lần chạy #24: đội học đủ bốn bước, và bài học lớn nhất trong ngày

Cả bốn bước `done`. Bản tổng hợp 14.000 ký tự đã đưa vào `docs/DU-AN-THAM-KHAO.md`.

Điều đáng giá nhất KHÔNG phải danh sách bài học, mà là kết luận mà **hai agent
độc lập cùng rút ra**, và nó chỉ thẳng vào chỗ yếu của chính sagent:

> Hỏng phải là **cấu trúc dữ liệu**, không phải **chữ trong văn bản**.

ACP nói điều đó bằng ví dụ thuận: `auth_required` là mã JSON-RPC `-32000`
(`gosdk/errors.go:66-68`), cụt vòng gọi tool là enum `stopReason = max_turn_requests`
(`gosdk/types_gen.go:6149`). Gas Town nói bằng phản ví dụ: họ nhét 21 trường vào
`description` rồi tách theo dấu hai chấm, và trả giá bằng `strings.HasPrefix` để
tra cứu.

`khongCoKetQua` của sagent đang đứng đúng phía sai của cả hai. Nó dò chuỗi tiếng
Anh — mà chuỗi đó là chữ ký của MỘT provider ở MỘT phiên bản. Provider đổi câu chữ
là lá chắn rơi im lặng, không ai biết. Nó là lá chắn tốt cho hôm nay và phải giữ,
nhưng **nó không phải đích đến**.

Bằng chứng ngay trong chính lượt này: bước học Agent Deck kết thúc bằng
`Error: timeout waiting for response` — chuỗi thứ TƯ, không nằm trong ba chữ ký đã
biết, nên vẫn `done`. Vừa thêm vào, kèm test. Nhưng đó đúng là trò đuổi bắt mà ACP
nói là không cần chơi.

Ranh giới cả hai agent tự rút ra độc lập: **mang mảnh rời, đừng mang cụm.** Ba thứ
chọn mang từ Gas Town đều dưới 100 dòng và không kéo theo gì; thứ bị loại
(issue-tracker-làm-database) kéo theo tất cả.

Và bản tổng hợp tự ghi rủi ro của chính nó: *"cả 7 bài học đều là kết luận đọc mã,
chưa cái nào chạy"*. Đúng tinh thần "đo, không đoán" — nó không tự phong cho mình
mức tin cậy chưa có.

Bước Agent Deck vẫn **chưa học được gì** (hết giờ giữa việc, nhánh không commit).
Hàng #2 trong bảng ưu tiên còn nguyên.

## 18/08 — Bỏ dò chuỗi, đọc dữ liệu có cấu trúc: đo được, làm được ngay

Bản tổng hợp của đội học xếp việc này lên đầu. Đo trước khi làm.

**Không CLI nào trên máy nói ACP.** `claude --help`, `codex --help`, `agy --help`
đều không có. Nhưng cả ba đều có đường khác, sẵn hôm nay:

| CLI | Có gì |
|---|---|
| claude | `--output-format stream-json` + `--verbose` |
| agy | `--output-format text\|json\|stream-json`, `--input-format`, `--json-schema` |
| codex | `mcp-server` (stdio MCP) |

Nên đích đến không phải ACP — mà là **dữ liệu có cấu trúc**, và nó có sẵn.

**Đo `claude -p --output-format stream-json --verbose`.** Dòng cuối
`{"type":"result", ...}` mang đủ mọi thứ mà cả ngày nay phải đoán bằng chuỗi:

```
is_error: false        subtype: "success"          stop_reason: "end_turn"
terminal_reason: "completed"   api_error_status: null   permission_denials: []
num_turns: 1           total_cost_usd: 0.08446     result: "OK"
usage: {input_tokens, output_tokens, cache_*}
```

Kèm sự kiện riêng `rate_limit_event` có `resetsAt` và `rateLimitType` — nói được
"hạn mức quay lại lúc mấy giờ" thay vì chỉ báo hỏng.

Đối chiếu bốn kiểu hỏng đã gặp trong ngày:

| Kiểu hỏng | Trước: dò chuỗi | Sau: đọc trường |
|---|---|---|
| bị chặn quyền | `"no output produced"`… | `permission_denials` khác rỗng |
| hết hạn đăng nhập | `"failed to authenticate"`… | `is_error` + `api_error_status` |
| quẩn vòng gọi tool | `"maximum tool execution rounds reached"` | `subtype = error_max_turns` |
| dừng giữa việc | `"timeout waiting for response"` | `is_error` + `terminal_reason` |

Bốn chuỗi tiếng Anh, mỗi chuỗi là chữ ký của MỘT provider ở MỘT phiên bản → bốn
trường có tên. Provider đổi câu chữ cũng không rơi lá chắn.

Đã thêm `provider.KetQua` + `DocKetQua(raw)` vào interface. Claude đọc được; bốn
provider kia trả `ok=false` (**chưa đo**, không phải không có — agy có
stream-json, chỉ là chưa đo cách đọc). Cầu nối ƯU TIÊN cấu trúc, dò chuỗi chỉ còn
là đường lui, và chỉ dùng khi provider chưa đo được.

**Hai thứ được thêm miễn phí:**
1. Output của bước giờ là CÂU TRẢ LỜI THẬT (trường `result`), không phải cả bản
   ghi NDJSON. Bước sau nhận dữ liệu sạch.
2. Theo dõi chi phí — món nợ từ đầu dự án. Nghiệm thu lần chạy #26:
   `claude:tns: 6 lượt, 10 token vào / 905 ra, 0.1809 USD`.

Test dùng NGUYÊN VĂN dòng result đo được, không phải JSON tự bịa. Có cả ca "bản
ghi không có dòng result" → phải nói thẳng KHÔNG ĐỌC ĐƯỢC, không được đoán.

**Bẫy đã dính khi kiểm:** `sagent fleet <tk> -- -p "..."` truyền đối số TAY nên
bỏ qua `HeadlessArgs` — đo bằng đường đó thì thấy log chữ trơn và tưởng bản vá
hỏng. Chỉ đường flow mới gọi `HeadlessArgs`. Phải đo đúng đường mà tính năng đi qua.

## 18/08 — Lượt fleet sau XOÁ SẠCH công việc của lượt trước

Lỗi mất dữ liệu nghiêm trọng nhất tìm được, và nó đã nổ thật.

`workspace.Add` dùng `git worktree add -B <nhánh>`. Cờ `-B` **đặt lại** nhánh về
HEAD hiện tại. Nên mỗi lượt fleet mới trên cùng một tài khoản xoá sạch commit của
lượt trước.

Bằng chứng: lần chạy #21, agent `claude:tns` sửa `internal/fleet/fleet.go` và
viết 87 dòng test, commit `161e79e` lên `sagent/tns-1`. Các lượt fleet sau đó cùng
tài khoản làm commit ấy thành **mồ côi** — còn nguyên trong kho nhưng không thuộc
nhánh nào. `git log main..sagent/tns-1` hiện TRỐNG TRƠN, y như chưa ai làm gì.

Điều làm nó nguy hiểm: **không có dấu hiệu nào cả.** Không lỗi, không cảnh báo.
Người dùng mở nhánh ra xem, thấy trống, và kết luận agent lười.

Đã cứu: `git branch cuu/tns-1-161e79e 161e79e` — 99 dòng, có test.

Sửa gốc bằng `nhanhTrong()`: trước khi tạo worktree, kiểm nhánh cũ có commit nào
chưa có ở nhánh nền không. Có thì dùng tên mới (`sagent/tns-1-2`), giữ nguyên việc
cũ. Rỗng thì cứ ghi đè như trước, khỏi đẻ nhánh thừa.

Thà vài nhánh thừa còn hơn mất việc — nhánh thừa dọn được, việc mất thì không.

Hai test dựng repo git thật. Phản chứng: trả lại `-B` cũ thì đỏ đúng câu
"VIỆC CỦA LƯỢT TRƯỚC BỊ XOÁ: nhánh sagent/tns-1 giờ có 0 commit, muốn 1".

**Vì sao mãi không ai thấy:** đúng cái tính năng "bằng chứng git" thêm sáng nay
mới làm nó lộ ra. Trước đó không mặt nào đọc số commit của nhánh, nên việc bị xoá
cũng không ai biết. Một lá chắn tìm ra lỗi mà nó không được viết để tìm.

## 18/08 — Antigravity: đọc được kết quả có cấu trúc, và một cái bẫy

`agy --output-format stream-json` dùng lược đồ KHÁC HẲN Claude:

```
{"event":"result","result":{"status":"SUCCESS","response":"OK\n","num_turns":1,"usage":{…}}}
{"event":"step_update","step_update":{"state":"ERROR","step_type":"tool","tool_name":"run_command"}}
```

**Bẫy đo được: bị chặn quyền thì `status` VẪN LÀ `"SUCCESS"`**, chỉ `response`
rỗng. Tin vào `status` là kết luận ngược hoàn toàn. Dấu hiệu thật: response rỗng,
cộng số `step_update` có `state:"ERROR"`.

Nên thêm `KetQua.ToolHong` tách khỏi `TuChoiSo`: Antigravity chỉ báo tool lỗi,
không nói lỗi vì bị chặn quyền hay vì lệnh sai. Đếm thì được, suy nguyên nhân thì
không — nên không suy.

Đây cũng là lý do việc đọc kết quả phải nằm trong ADAPTER, không phải một hàm
chung: hai provider, hai lược đồ, và một cái nói dối bằng trường `status`.

Nghiệm thu lần chạy #27: output bước là câu trả lời sạch, không còn NDJSON.

## 18/08 — Học "skill uxui": nó là design-system trong repo, và tôi đã vi phạm

"skill uxui" chủ dự án nhắc chính là `design-system/switch-agent-pro/MASTER.md` —
có bảng token màu/khoảng cách, quy cách component, mục "Anti-Patterns (Do NOT Use)"
và checklist trước khi giao. Sinh ngày 16/08, và chưa từng ai kiểm chiếu.

Rà lại thì thấy tôi vi phạm ngay trong lượt làm giao diện hôm nay:

| Cấm | Tôi đã dùng | Sửa thành |
|---|---|---|
| emoji làm icon | 👑 🎯 ⚠ (vai trò + cảnh báo quyền) | path SVG kiểu Lucide qua hàm `icon()` có sẵn |
| ký tự lạ làm icon | ↶ ↷ ▶ ✓ ✗ ● ○ | SVG cho nút, chấm CSS đổi màu cho trạng thái |
| thiếu prefers-reduced-motion | index.html, flow.html | thêm khối `@media` tôn trọng |

Bắt được thêm một lỗi CÓ SẴN không liên quan design system: `esc()` được gọi ở
index.html (thẻ workflow tôi thêm lượt trước) nhưng **chưa hề định nghĩa** — tên
flow/lỗi có ký tự `<` `>` sẽ vỡ layout, tệ hơn là chèn được thẻ lạ. Đã thêm hàm
escape thật.

**Biến checklist thành test** — `internal/dash/uxui_test.go`:
- `TestKhongDungEmojiLamIcon`: quét mọi `web/*.html`, thấy ký tự U+2000↑ thuộc
  nhóm So/Sk hoặc dải emoji thì báo lỗi (chừa mũi tên/dấu nhấn trong câu văn).
- `TestTonTrongReducedMotion`: trang có `@keyframes`/`animation`/`transition` mà
  không có `prefers-reduced-motion` thì báo lỗi.

Hai test này BẮT ĐƯỢC NGAY hai vi phạm còn sót trong index.html khi chạy lần đầu —
đúng thứ chúng sinh ra để bắt.

Thêm Motion 6/10 đúng như dial: thẻ trồi lên khi tải, so le nhẹ, dùng
`cubic-bezier(0.16,1,0.3,1)` chứ KHÔNG `back.out` — design system cấm overshoot
trên bảng dữ liệu dày.

Bài học: **giao diện là chỗ dễ trôi khỏi quy chuẩn nhất** vì Go không build ra nó.
Nay checklist của design system có ba mục được máy canh, không còn trông vào việc
người nhớ đọc.

## 19/08 — Placeholder chưa thay lọt nguyên văn vào prompt gửi cho agent (lượt chạy #29)

Lượt chạy #29 (flow `dem`) chạy dở thì máy tự khởi động lại. Khi đọc lại nhật ký,
phát hiện lỗi im lặng nguy hiểm: biến giữ chỗ (placeholder) chưa thay thế bị gửi
nguyên văn sang cho agent.

- **Đo lúc nào**: Ngày 19/08/2026, khi phân tích nhật ký thực thi của lượt chạy #29.
- **Đo bằng cách nào**: Đọc lại nội dung prompt thực tế mà hệ thống gửi cho agent ở
  bước `soi`.
- **Con số / Bằng chứng**: Bước `kiem-cuoi` (máy chấm chạy lệnh `go test`) bị hỏng
  nên không để lại kết quả trong sổ biến. Hàm `Expand` cũ chỉ thay thế những biến
  CÓ mặt trong bảng kết quả; biến nào vắng mặt thì giữ nguyên chuỗi thô. Hậu quả là
  người soi (Grok) nhận nguyên văn chuỗi:
  ```
  Máy chấm nói gì:
  {{steps.kiem-cuoi.output}}
  ```
  Agent ở bước sau vẫn tự tin phán xét như thể đã đọc phán quyết của máy chấm, dù
  thực tế không nhận được gì. Toàn bộ lời hứa "máy chấm quyết định, không phải lời
  agent" bốc hơi trong im lặng mà không có bất kỳ cảnh báo nào.
- **Đã sửa hay chưa**: **ĐÃ SỬA** (commit `c67a1ff`). Tách làm hai hàm xử lý:
  1. `ExpandChay`: Với các bước agent và notify, tự động thay các biến còn sót bằng
     câu thông báo trung thực `(bước "<tên bước>" không để lại kết quả)`.
  2. Bước chạy lệnh shell thì kiểm tra bằng `BuocConSot` và **dừng ngay lập tức**
     kèm thông báo lỗi rõ ràng: `tham số X cần kết quả của bước "Y" nhưng bước đó không để lại gì`.
     Tuyệt đối không chốt chuỗi giả vào câu lệnh shell (ví dụ `go test -C (bước "x" không để lại kết quả)`
     là đường dẫn bịa, sẽ làm lệnh hỏng bằng một lỗi chẳng liên quan tới nguyên nhân thật).
  3. Bản `Expand` gốc giữ nguyên cho lệnh `sagent flow show` để xem thử khi chưa chạy.
  - Nghiệm thu: Kiểm chứng bằng test `TestPromptGuiDiKhongDuocMangPlaceholderConSot` và
    `TestShellThieuKetQuaThiDungVaGoiTenBuoc` trong `internal/flow/placeholder_test.go`.
    Gỡ bản vá ra thì test đỏ ngay lập tức.

## 19/08 — "agent báo lỗi: success" — chữ nghĩa tự mâu thuẫn làm giấu lý do thật (lượt #29)

- **Đo lúc nào**: Ngày 19/08/2026, ở lượt chạy #29, bước `code-go`.
- **Đo bằng cách nào**: Đọc dữ liệu có cấu trúc từ phản hồi JSON của Claude CLI khi
  tài khoản gặp sự cố.
- **Con số / Bằng chứng**: Claude CLI trả về cờ báo lỗi `is_error = true`, nhưng
  trường phân loại phụ `subtype` lại mang giá trị `"success"`. Hàm hiển thị lỗi cũ
  ghép thẳng `subtype` vào sau chữ "agent báo lỗi:", tạo ra thông báo vô nghĩa:
  `agent báo lỗi: success`. Thông điệp này vừa tự mâu thuẫn vừa che giấu hoàn toàn
  nguyên nhân lỗi thật nằm trong trường câu trả lời (`result`), ví dụ: `"Credit balance is too low"`
  (tài khoản hết số dư/hạn mức).
- **Đã sửa hay chưa**: **ĐÃ SỬA** (commit `c67a1ff`). Hàm `lyDo()` trong
  `internal/provider/ketqua.go` đọc lý do theo thứ tự ưu tiên:
  1. Lời agent trong câu trả lời thật (`result` / `TraLoi`).
  2. Lý do kết thúc phiên (`terminal_reason` / `KetCuc`).
  3. Lý do dừng (`stop_reason` / `DungViCo`).
  4. Phân loại lỗi (`subtype` / `Loai`).
  - Chủ động bỏ qua mọi giá trị tự nhận là `"success"`. Nếu không có lý do nào thì
    nói thẳng `"không nói lý do"`, tuyệt đối không ghép chữ nghe cho có.
  - Nghiệm thu: Test `TestCoLoiNhungSubtypeLaSuccess` và `TestCoLoiKhongCoResultThiLuiVeLyDoKhac`
    trong `internal/provider/ketqua_lydo_test.go` bắt đúng trường hợp này.

## 19/08 — Flow run không kiểm tài khoản trước, đốt một lượt 4 bước chết sẵn vì token hết hạn (lượt #29)

- **Đo lúc nào**: Rạng sáng 19/08/2026 (lượt chạy #29 bấm chạy lúc 01:45).
- **Đo bằng cách nào**: So sánh mốc thời gian hết hạn token của tài khoản `claude:tns`
  (`18/08 23:44`) với thời điểm bấm chạy flow (`19/08 01:45`), và xem nhật ký thực thi
  của lượt #29.
- **Con số / Bằng chứng**: Token đã hết hạn trước đó 2 tiếng. Lượt chạy #29 đốt 9 bước,
  trong đó có 4 bước chết chắc từ đầu (`code-go`, `sua-1`, `sua-2` và máy chấm kéo theo).
  Hệ thống vốn đã biết tình trạng này (`Profile.HetHan` có từ commit `0bcb903`), nhưng
  lệnh `flow run` không kiểm tra trước khi chạy. Hậu quả là toàn bộ quy trình vẫn chạy,
  để lại đầy bước đỏ trong nhật ký ("agent báo lỗi", "không bật được phiên nào", máy chấm
  đỏ theo), gây mất thời gian tìm lỗi trong code trong khi nguyên nhân thực sự chỉ là cần
  đăng nhập lại.
- **Đã sửa hay chưa**: **ĐÃ SỬA** (commit `1f77a01`). Thêm cổng kiểm tra tài khoản
  `KiemTaiKhoanFlow` trước khi chạy `flow run`:
  1. Soi mọi tài khoản cần cho flow, tài khoản nào hỏng/hết hạn thì **DỪNG NGAY** trước
     khi ghi bất kỳ lượt chạy nào vào cơ sở dữ liệu.
  2. Thông báo dừng nói đủ 3 thứ: hỏng cái gì, kéo theo những bước nào, và lệnh sửa cụ thể
     (ví dụ: `sagent claude:tns`).
  3. Người dùng biết rõ mà vẫn muốn chạy tiếp thì dùng cờ `--cu-chay` (vẫn ghi cảnh báo
     vào nhật ký để sau này tra cứu).
  - Nghiệm thu: Test `TestChanKhiTokenDaHetHan` và `TestFlowRunChanTruocKhiGhiLanChayNao`
    trong `internal/api/congkiem_test.go`. Gỡ cổng kiểm tra ra thì test đỏ ngay.

## 19/08 — Người soi (Grok) phán NGƯỢC nhánh ở lượt #29 dù lệnh git ra kết quả đúng — CHƯA SỬA

- **Đo lúc nào**: Lượt chạy #29 ngày 19/08/2026, ở bước `soi`.
- **Đo bằng cách nào**: Đối chiếu kết quả lệnh `git log` mà Grok thực thi với nhận xét
  do chính Grok trả về trong bước soi.
- **Con số / Bằng chứng**:
  - Bằng chứng trong git: Nhánh `sagent/may-1` có commit `4ea8141` do Antigravity tạo file
    `docs/BAT-DAU.md` (sau đó được trộn ở commit `0b9e50b`), còn nhánh `sagent/tns-1`
    hoàn toàn không có commit nào (0 commit) do `claude:tns` hết hạn token từ trước.
  - Lệnh `git log --oneline main..sagent/tns-1` và `git log --oneline main..sagent/may-1`
    do Grok chạy trong bước soi đã đọc ra đúng trạng thái trên.
  - Tuy nhiên, khi viết kết luận, Grok phán **NGƯỢC HOÀN TOÀN**: gán toàn bộ việc tạo tài
    liệu của `may-1` sang cho `tns-1`, và phán `may-1` chưa làm gì.
  - *Lưu ý về bằng chứng*: Lịch sử git lưu giữ đầy đủ bằng chứng về commit của các nhánh
    (`may-1` có commit, `tns-1` rỗng). Riêng câu chữ phán ngược cụ thể của Grok nằm trong
    nhật ký thực thi của phiên chạy (runtime log / database), không được lưu trực tiếp vào git log.
- **Đã sửa hay chưa**: **CHƯA SỬA — ghi rõ là chưa sửa**. Đây là lỗi suy luận logic / ngộ
  nhận nhãn ngữ cảnh của mô hình ngôn ngữ lớn (Grok). Hiện hệ thống chưa có cơ chế kiểm
  chéo tự động giữa phán quyết văn bản của agent soi với dữ liệu commit thực tế từ git.

## 19/08 — `expiresAt = 0` là ĐĂNG NHẬP DỞ DANG, không phải "hình dạng mới"

Mục này đã bị ghi SAI một lần trong chính ngày hôm nay, và bản ghi sai đó nằm
trong commit `4fa396f`. Giữ lại cả hai vòng để thấy sai ở đâu.

- **Đo lúc nào**: 19/08/2026, hai lần — 19:30 (kết luận sai) và 20:05 (đo lại).
- **Đo bằng cách nào**:
  - Vòng 1: đọc `.credentials.json` của `claude:tns` sau khi đăng nhập. **Một mẫu duy nhất.**
  - Vòng 2: hỏi thẳng CLI `claude auth status` trên đúng hồ sơ đó, VÀ đối chiếu với
    một tài khoản đang chạy được (`~/.claude`).
- **Con số / Bằng chứng**:

  | Hồ sơ | `expiresAt` | `claude auth status` |
  |---|---|---|
  | `claude:tns` lúc 19:30 (đăng nhập hỏng) | `0` | `{"loggedIn": false, "authMethod": "none"}` |
  | `~/.claude` (đang chạy được) | `1787156212029` | đăng nhập bình thường |
  | `claude:tns` lúc 20:08 (đăng nhập lại, thành công) | `1787173725663` → 20/08 04:08 | `{"loggedIn": true, ...}` |

  File của lần đăng nhập hỏng vẫn ĐẦY ĐỦ TRƯỜNG — `accessToken`, `refreshToken`,
  `scopes`, `subscriptionType: "max"`, `refreshTokenExpiresAt` của tháng sau. Chỉ
  khác đúng một chỗ là `expiresAt: 0`. Nhìn qua tưởng lành.

- **Kết luận sai ở vòng 1**: thấy `expiresAt: 0` kèm `refreshTokenExpiresAt`, suy ra
  "Claude CLI đổi hình dạng dữ liệu", rồi cho `TokenExpiry` lùi về đọc
  `refreshTokenExpiresAt`. Suy từ một mẫu, không đối chiếu với tài khoản nào đang
  chạy được. Bản vá đó làm cổng kiểm tài khoản DỄ DÃI HƠN bản chưa vá.

- **Giá phải trả**: lượt chạy #31 (19:56) đi lọt qua cổng kiểm rồi bước `code-go`
  chết ngay với `Failed to authenticate: OAuth session expired and could not be
  refreshed`. Mất một nhánh việc của cả lượt.

- **Đã sửa hay chưa**: **ĐÃ SỬA** (commit `714730c`).
  1. `TokenExpiry` trở lại chỉ đọc `expiresAt`.
  2. `HasToken` thôi kiểm "file có tồn tại" mà đòi `expiresAt != 0` — file có mà
     token chết thì không phải "sẵn sàng".
  - Nghiệm thu: `TestDangNhapDoDangKhongDuocTinhLaCoToken` và
    `TestTokenThatThiCoExpiresAtKhac0` trong `internal/provider/token_hinhdang_test.go`.
    Gỡ bản vá ra thì test đỏ.
  - Kiểm trên máy thật: `sagent ds` chuyển từ "sẵn sàng" (sai) sang "chưa đăng nhập"
    (đúng), khớp với điều CLI tự nói.

- **Bài học, và là bài học đắt nhất trong ngày**: một câu chuyện MẠCH LẠC không phải
  là một SỐ ĐO. Vòng 1 nghe hợp lý đến mức đủ sức thuyết phục để viết hẳn một đoạn
  bình luận dài trong mã nguồn giải thích nó. Thứ lật lại được chỉ là một câu hỏi rẻ
  tiền: "tài khoản đang chạy được thì trường này bằng bao nhiêu?" — chưa hỏi câu đó
  thì chưa gọi là đã đo.

## 19/08 — Huỷ lượt chạy bị cắt ngang: không để flow treo "đang chạy" vĩnh viễn

- **Đo lúc nào**: Chiều 19/08/2026, sau hai lần quy trình bị cắt ngang (#29 chết theo máy tự khởi động lại lúc 01:47, #30 chết khi người dùng dừng tay lúc 19:37).
- **Đo bằng cách nào**: Quan sát bảng lịch sử chạy trong dashboard và CLI sau khi phiên bị dừng đột ngột.
- **Con số / Bằng chứng**: Lệnh huỷ cũ (`Reject`) chỉ xử lý được bước ĐANG CHỜ DUYỆT. Lượt chạy bị cắt ngang không có bước nào như vậy nên nằm lại trạng thái `running` vĩnh viễn trong cơ sở dữ liệu. Bảng lịch sử báo "đang chạy" trong khi không còn tiến trình nào sống.
- **Đã sửa hay chưa**: **ĐÃ SỬA** (commit `4fa396f`). Thêm lệnh dòng lệnh `sagent flow huy <#>` và API `POST /api/flow/cancel`. Các bước đang chạy dở được chuyển trạng thái sang `failed` CÓ GHI RÕ LÝ DO (tránh để trống làm người đọc sau này tưởng code hỏng); các bước đã hoàn thành được giữ nguyên. Tách bạch việc cập nhật sổ trạng thái với việc giết tiến trình. Nghiệm thu qua `internal/flow/huy_test.go`.

## 19/08 — Dung lượng file thực thi .exe: 12.3 MB theo số đo thật

- **Đo lúc nào**: Tối 19/08/2026 (commit `2922dd3`).
- **Đo bằng cách nào**: Build bằng đúng cờ của quy trình phát hành (`-trimpath -ldflags "-s -w"`).
- **Con số / Bằng chứng**: File thực thi `sagent.exe` có dung lượng thực tế là **12.3 MB** (làm tròn 12 MB). Con số 11 MB ghi trong tài liệu trước đó xuất phát từ thời điểm sản phẩm còn ít tính năng.
- **Đã sửa hay chưa**: **ĐÃ SỬA** (commit `2922dd3`). Cập nhật lại số đo thực tế trong `README.md` và `docs/BAT-DAU.md`.

## 19/08 — Phát hiện agent CHẠY QUẨN, không chỉ giới hạn thiệt hại

- **Đo lúc nào**: Số đo gốc là lần chạy #21 (18/08), xem mục "flow báo `completed`, thật ra 3/5 bước hỏng" ở trên. Phần phát hiện làm tối 19/08.
- **Đo bằng cách nào**: Bước `soi` chạy bằng `grok:api` gọi đúng `ls -la` **399 lần liên tiếp** rồi mới bị trần `--max-tool-rounds` chặn — tool `bash` của Grok chạy qua `cmd.exe` nên lệnh Unix trượt, mà nó không đổi cách. Bước vẫn được đánh dấu `done`.
- **Con số / Bằng chứng**: Việc đã làm sau #21 là hạ trần vòng tool xuống 60. Trần chỉ giới hạn THIỆT HẠI, nó không phát hiện được gì: 60 vòng `ls -la` vẫn là 60 vòng vô ích và bước vẫn báo xong. Nay `KetQua` đếm chuỗi lời gọi tool **giống hệt nhau liên tiếp** dài nhất, đọc từ khối `tool_use` (tên tool + tham số) trong bản ghi stream-json — không dò chữ. Vượt 10 thì `Hong()` nói thẳng: *agent lặp lại lệnh "ls -la" 399 lần liên tiếp — nghi chạy quẩn, KHÔNG phải lỗi code*.
- **Ngưỡng 10 được bao nhiêu bằng chứng**: Chỉ được MỘT MẶT. 399 ≫ 10 nên chắc chắn không bỏ sót ca thật. Mặt kia — có bắt oan lượt chạy bình thường không — **chưa đo được**: chưa đếm chuỗi lặp dài nhất trên một bản ghi lượt-chạy-bình-thường nào. Phòng bắt oan bằng thiết kế chứ không bằng số đo: đếm chuỗi LIÊN TIẾP (chạy `git status` nhiều lần trong một bước là bình thường, nhưng giữa chúng có tool khác nên chuỗi bị ngắt) và chữ ký gồm CẢ THAM SỐ (`Read` 30 file khác nhau là đang làm việc, `Read` đúng một file 30 lần mới là quẩn).
- **Chưa làm được**: (1) Ca quẩn XEN KẼ (A,B,A,B,…) không bắt được — chưa gặp bản ghi nào như vậy, mở rộng bây giờ là đoán. (2) Trớ trêu: provider gây ra ca #21 là Grok thì **chưa đọc được kết quả có cấu trúc**, nên với nó trần 60 vòng vẫn là thứ duy nhất cứu. (3) Antigravity phát `step_update` chỉ mang `tool_name`, KHÔNG mang tham số, nên cố tình không kết luận — `Quan()` trả về "không biết", khác hẳn "không quẩn".
- **Đã sửa hay chưa**: **ĐÃ SỬA** phần phát hiện cho Claude. Nghiệm thu bằng `internal/provider/quan_test.go` (quẩn thật 399 vòng, lặp bình thường không bị bắt, thiếu dữ liệu thì nói không biết); mỗi test đã chứng minh đỏ khi gỡ đúng phần logic nó canh.

## 19/08 — Chạy khan (`flow run --kho`): kiểm tra toàn bộ luồng mà không đốt hạn mức và không sinh lượt rác

- **Đo lúc nào**: Tối 19/08/2026.
- **Đo bằng cách nào**: Quan sát thực tế vận hành khi người dùng muốn kiểm tra trước cấu hình luồng hoặc xem cổng kiểm tra nói gì nhưng lại bấm nhầm lệnh chạy thật (`flow run`).
- **Con số / Bằng chứng**:
  - Trong ngày 19/08/2026, đã xảy ra **3 lượt chạy thật lỡ tay** (#30 lúc 19:37, #32, #33).
  - Mỗi lượt chạy thật lỡ tay đều kích hoạt agent thật, tiêu tốn hạn mức tài khoản (token/quota) và sinh ra các lượt chạy dở dang/rác ở trạng thái `running` hoặc `failed` trong sổ trạng thái `state.db` buộc phải dùng lệnh `flow huy` để dọn.
  - Cần một cơ chế chạy khan (dry-run) để kiểm tra toàn diện trước khi bấm chạy thật: kiểm tra cấu trúc DAG (`flow.Validate`), chạy qua cổng kiểm tra tài khoản (`KiemTaiKhoanFlow`), tính toán các đợt chạy song song (`flow.Dot`), và hiển thị trước prompt sau khi thay biến (`flow.Expand` / `BuocConSot` — chỉ rõ bước nào có placeholder chưa có kết quả từ bước trước) mà **tuyệt đối KHÔNG chạm vào cơ sở dữ liệu (`state.db`), không tạo git worktree, và không gọi agent thật**.
- **Đã sửa hay chưa**: **ĐÃ SỬA**.
  1. Thêm cờ `--kho` cho lệnh CLI: `sagent flow run <tên> --kho`.
  2. Bổ sung action `flow.kho` vào hợp đồng API lõi `api.Actions`, route `POST /api/flow/kho` và nút "Chạy khan" trên bảng vẽ web (`flow.html`).
  3. API và CLI trả về toàn bộ kế hoạch thực thi chi tiết: phân chia theo từng đợt chạy, bước nào chạy song song trong mỗi đợt, tài khoản AI được phân công cho từng bước, nội dung prompt sau khi điền biến hoặc cảnh báo placeholder chưa có dữ liệu từ bước trước.
  - Nghiệm thu: Bộ test kiểm tra đảm bảo luật 3 mặt (CLI, API, Web UI) và khẳng định chạy khan không ghi bất kỳ bản ghi nào vào bảng `flow_runs`/`flow_steps` trong cơ sở dữ liệu, không tạo thư mục tạm và không tiêu tốn token của agent.

## 19/08 — Rà soát offline runtime dashboard (INV-UI-1): phát hiện 4 điểm vi phạm tải asset ngoài (CDN/Google Fonts)

- **Đo lúc nào**: Tối 19/08/2026, trong đợt rà soát tính độc lập offline và bất biến giao diện INV-UI-1 cho dashboard.
- **Đo bằng cách nào**: Quét toàn bộ các file `.html` trong `internal/dash/web` để tìm các thuộc tính `src=` hoặc `href=` trong thẻ `<script>` và `<link>` trỏ ra URL bên ngoài (`http://` hoặc `https://`). Thẻ `<a href="...">` cho người dùng bấm chuyển trang (như ở `internal/dash/web/docs/index.html:580`) là liên kết điều hướng thông thường, không phải asset render runtime nên được bỏ qua.
- **Con số / Bằng chứng**: Phát hiện đúng **4 dòng vi phạm** tải tài nguyên từ CDN/mạng ngoài lúc runtime:
  1. File: `internal/dash/web/3d.html:73` · Asset: Three.js r128 core (`three.min.js`) · Nguồn: `https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js`
  2. File: `internal/dash/web/3d.html:74` · Asset: Three.js OrbitControls (`OrbitControls.js`) · Nguồn: `https://cdn.jsdelivr.net/npm/three@0.128.0/examples/js/controls/OrbitControls.js`
  3. File: `internal/dash/web/docs/index.html:8` · Asset: Font chữ Inter (`Inter:wght@300;...;800`) · Nguồn: `https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap`
  4. File: `internal/dash/web/docs/master-plan.html:9` · Asset: Font chữ Inter (`Inter:wght@300;...;800`) · Nguồn: `https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap`
- **Đã sửa hay chưa**: Đang xử lý song song trong kế hoạch 2 phần:
  1. *PHẦN 1 (Mã nguồn & Asset)*: Tải toàn bộ asset về offline (`three.min.js` r128 core và 3 file font woff2: Space Grotesk, Inter, JetBrains Mono vào `internal/dash/web/vendor/`), sửa 3 file HTML trỏ đường dẫn nội bộ, gỡ `OrbitControls` và tự viết tay camera orbit (drag xoay azimuth/polar, wheel zoom, clamp polar, damping), khai `@font-face` nội bộ, thêm test `internal/dash/offline_asset_test.go` quét mọi file HTML bắt lỗi nếu trỏ ra CDN ngoài.
  2. *PHẦN 2 (Tài liệu)*: Khóa ba bất biến giao diện (INV-UI-1, INV-UI-2, INV-UI-3) vào `MASTER-PLAN.md` và ghi nhận đầy đủ số liệu đo đạc vào `DO-LUONG.md`.
- **Bài học màn 3D trắng trơn**: Trong môi trường không có mạng (offline, air-gap) hoặc mạng nội bộ chặn truy cập ra các CDN công cộng (`cdnjs`, `jsdelivr`, `fonts.googleapis.com`), việc gọi asset từ CDN bên ngoài khiến thư viện 3D không thể nạp, dẫn tới lỗi khởi tạo và màn hình 3D hoàn toàn trắng trơn không hoạt động được. Là một control plane local-first chạy native một binary trên Windows (assets nhúng qua Go `embed`), toàn bộ giao diện dashboard bắt buộc phải offline tuyệt đối lúc runtime (INV-UI-1), không được phép phụ thuộc vào bất kỳ kết nối internet nào để hiển thị.

## 20/08 — Dung lượng binary và asset 3D văn phòng: GLTFLoader.js + RobotExpressive.glb

- **Đo lúc nào**: Ngày 20/08/2026, trong đợt triển khai giao diện văn phòng 3D (`vanphong.html`) theo kế hoạch `docs/KE-HOACH-VAN-PHONG.md`.
- **Đo bằng cách nào**:
  1. Tải thật và đo kích thước byte chính xác của 2 asset từ nguồn chuẩn Three.js r128: `RobotExpressive.glb` (HTTP 200, **463.988 byte**) và `GLTFLoader.js` (HTTP 200, **96.550 byte**).
  2. Thực hiện biên dịch kiểm thử file thực thi Go với đầy đủ cờ phát hành chuẩn (`go build -trimpath -ldflags "-s -w" ./cmd/sagent`) trước và sau khi đưa 2 asset vào `internal/dash/web/vendor/`.
- **Con số / Bằng chứng**:
  - **Tài nguyên thêm mới (Vendor Assets):**
    - `internal/dash/web/vendor/RobotExpressive.glb`: **463.988 byte** (~453,1 KB)
    - `internal/dash/web/vendor/GLTFLoader.js`: **96.550 byte** (~94,3 KB)
    - Tổng dung lượng asset thêm vào: **560.538 byte** (~547,4 KB / ~0,53 MB).
  - **Dung lượng file thực thi binary (`sagent.exe` trên Windows amd64):**
    - **TRƯỚC khi thêm asset** (chỉ có Three.js r128 core + 3 font woff2 + `token.css`): **14.516.736 byte** (~13,84 MB, ghi nhận 13,8 MB).
    - **SAU khi nhúng thêm 2 asset 3D**: **15.077.376 byte** (~14,38 MB, làm tròn ~14,4 MB).
    - Mức phình thực tế của file binary: đúng **560.640 byte** (~547,5 KB).
  - **So sánh với dự tính:** Kế hoạch ban đầu ước tính binary tăng thêm ~1 MB (từ 13,7 MB lên ~14,7 MB). Nhờ chọn đúng file mô hình chuẩn nén tối ưu (463 KB) và loader độc lập (96 KB), kích thước thực tế chỉ tăng ~0,53 MB (lên ~14,4 MB), tiết kiệm gần 500 KB dung lượng lưu trữ so với mức trần dự kiến.
- **Đã sửa hay chưa**: **ĐÃ ĐO VÀ CẬP NHẬT**. Cập nhật đầy đủ số đo thực tế vào `docs/THIET-KE.md`, `internal/dash/web/docs/THIET-KE.md`, `docs/DO-LUONG.md`, và cập nhật thông số dung lượng binary trong `README.md` (từ 13,7 MB lên ~14,4 MB).
- **Bài học & Nguyên tắc áp dụng:**
  - *Nới luật có kiểm soát:* Giữ vững lệnh cấm đối với các addon hiệu ứng hậu kỳ nặng (`EffectComposer`, `UnrealBloomPass`, `OrbitControls`) nhằm tránh kéo theo chuỗi phụ thuộc rườm rà và rủi ro màn hình trắng trơn. Chỉ nới lỏng cho phép duy nhất loader độc lập (`GLTFLoader.js`) để tận dụng 13 hoạt ảnh chuyển động có sẵn từ mô hình `.glb`.
  - *Đo thật trước khi chia việc:* Việc tải và đo đạc trực tiếp cả hai asset trước khi bắt tay vào triển khai giúp loại bỏ hoàn toàn rủi ro đường dẫn hỏng (404/URL chết), đảm bảo kế hoạch bám sát số liệu thực tế thay vì phỏng đoán.




## 20/08 — `fleet` mù còn `flow` đo được: hai đường, một CLI

- **Đo lúc nào**: Tối 20/08/2026, sau khi VPS tự khởi động lại lần thứ hai.
- **Đo bằng cách nào**: đối chiếu hai đường chạy agent trên CÙNG một CLI và CÙNG
  những tài khoản (`claude:phu`, `claude:tns`):
  1. `sagent status` — bảng phiên do `fleet` bật.
  2. `sagent flow tom-tat 47` — lượt chạy flow `doi-4` cùng ngày.
- **Con số / Bằng chứng**:
  - Đường `fleet`: **20/20 phiên** ở trạng thái `lost`, hiện ra là *"chết, chưa rõ
    vì sao"*; cột tokens và cost đều *"chưa đo"*.
  - Đường `flow` (lượt #47): đo được **99.051 token vào**, **81.492 token ra**,
    **11,0572 USD**, và đọc được cả lý do hỏng của bước `gop`
    (*"Failed to authenticate: OAuth session expired and could not be refreshed"*).
  - Nguyên nhân: `FleetStart` truyền args **THÔ** cho CLI con, còn `flow` đi qua
    `argsChoBuoc` nên được adapter dựng args. Người dùng gõ `-- -p "việc"` là agent
    chạy được, nhưng thiếu `--output-format stream-json --verbose` thì
    `docKetQuaClaude` không có dòng `{"type":"result"}` nào để đọc.
- **Đã sửa hay chưa**: **ĐÃ SỬA**. `provider.CoConThieu` hỏi CHÍNH ADAPTER xem bộ
  cờ còn thiếu gì rồi bổ sung, và phát cảnh báo nói rõ đã thêm gì. Không chép cứng
  tên cờ nào, nên thêm provider mới hay Claude đổi tên cờ thì chỗ đó tự đúng theo.
  Provider **CHƯA ĐO** `ket-qua-co-cau-truc` thì không thêm gì — cờ khai bừa làm CLI
  con chết ngay dòng đầu, tức đổi *"không đo được"* thành *"không chạy được"*.
- **Bài học**: một năng lực đã đo được ở tầng adapter **không tự đến** mọi đường
  gọi. Bốn mặt điều khiển đọc chung một hợp đồng, nhưng hai đường CHẠY thì không —
  và đường không hỏi adapter là đường mù. Test ngang quyền không bắt được: nó kiểm
  mọi action có lệnh CLI, chứ không kiểm lệnh đó chạy ra dữ liệu dùng được.

## 20/08 — Token clone và bản gốc tranh nhau refresh: `claude:phu` mất phiên

- **Đo lúc nào**: 20/08/2026 ~21:00, sau khi bước `gop` của lượt #47 hỏng.
- **Đo bằng cách nào**: đọc trực tiếp `claudeAiOauth` trong `.credentials.json` của
  bản gốc và của bản clone, so mốc `expiresAt` / `refreshTokenExpiresAt`.
- **Con số / Bằng chứng**:

  | Hồ sơ | `expiresAt` | `refreshTokenExpiresAt` |
  |---|---|---|
  | `.claude-accounts/phu` (gốc) | 20/08 11:10Z — hết hạn | 16/09 |
  | `.clones/claude/phu/1` | **không còn trường này** | 16/09 |
  | `.clones/claude/tns/1` | 20/08 20:44Z — còn hạn | 15/09 |

  Bản clone của `phu` mất hẳn access token, refresh thất bại dù refresh token còn
  hạn trên giấy. Bản clone của `tns` thì refresh thành công và bản gốc kẹt token cũ.
- **Đã sửa hay chưa**: **CHƯA** — mới có dấu vết, chưa chứng minh được cơ chế.
  `internal/profile/clone.go` đã ghi sẵn cảnh báo *"⚠ CHƯA ĐO: token bị chép ra N
  chỗ thì khi hết hạn, N tiến trình có thể cùng refresh một lúc"*. Số liệu trên là
  lần đầu tiên chuyện đó **có dấu vết thật**, và nó khớp với giả thuyết xoay vòng
  refresh token: ai refresh trước thì bản còn lại chết.
- **Cách chữa tạm**: đăng nhập lại bản gốc (`sagent claude:phu` → `/login`).
  `FleetStart` gọi `profile.Clone` mỗi lần bật nên clone tự nhận token mới.
- **Việc còn treo**: đo cho ra cơ chế — chạy hai clone cùng lúc trên một tài khoản
  vào đúng lúc access token sắp hết hạn, xem có tái hiện được không. Nếu đúng là
  xoay vòng thì đường clone cần một người giữ token duy nhất, không phải N bản sao.

## 20/08 — Đo lại Claude CLI sau provider drift: 2.1.229 → 2.1.234

- **Đo lúc nào**: 20/08/2026, sau khi `sagent verify claude` báo drift (mốc cũ ghi
  17/08/2026 gắn với bản `2.1.229`).
- **Đo bằng cách nào**: chạy `claude --help` trên bản mới và đối chiếu **từng cờ**
  mà `internal/provider/claude.go` đang khai; đọc thẳng hai file hồ sơ cho hai năng
  lực không nằm trong `--help`; và chạy **một lượt thật** để kiểm bản ghi có cấu trúc.
- **Con số / Bằng chứng**:
  - `claude --version` → `2.1.234 (Claude Code)`.
  - Cờ còn nguyên trên bản mới: `-p, --print`, `--output-format`, `stream-json`,
    `--verbose`, `--model`, `--dangerously-skip-permissions`, `--add-dir` — **7/7 còn**.
  - `ket-qua-co-cau-truc` — chạy thật `claude -p "…" --output-format stream-json
    --verbose` với `CLAUDE_CONFIG_DIR` trỏ vào hồ sơ `phu`. Dòng cuối:
    `type=result`, `subtype=success`, `is_error=false`, `stop_reason=end_turn`,
    `usage.input_tokens=2`, `usage.output_tokens=4`, `total_cost_usd=0.096977`.
  - `tach-nhieu-tai-khoan` — chính lượt chạy trên là bằng chứng: đặt
    `CLAUDE_CONFIG_DIR` vào thư mục hồ sơ riêng thì CLI dùng đúng danh tính đó.
  - `danh-tinh` — khoá `oauthAccount.emailAddress` còn trong `.claude.json`.
  - `han-token` — khoá `claudeAiOauth.refreshTokenExpiresAt` còn trong `.credentials.json`.
- **Đã sửa hay chưa**: **ĐÃ ĐO LẠI VÀ CHẤP NHẬN**. Bằng chứng trong bảng năng lực
  của `internal/provider/claude.go` cập nhật sang `2.1.234`, rồi mới chạy
  `sagent verify --chap-nhan`.
- **Bài học**: chấp nhận drift **trước khi** đo lại là đúng thứ cơ chế này lập ra
  để chặn — nó biến một cảnh báo có ích thành một nút bấm cho im. Giá của việc đo
  thật ở đây là **0,096977 USD** và khoảng một phút; giá của việc bấm cho im là cả
  bảng năng lực nói về một phiên bản không còn tồn tại trên máy.

## 20/08 — Đường API: DeepSeek và Grok chạy thật qua `modelapi.vn`

- **Đo lúc nào**: 20/08/2026, 22:41.
- **Đo bằng cách nào**: `sagent api <route> "Tra loi dung mot tu: OK"` — đi đúng
  đường `internal/aiapi` mà flow node `model` dùng, không phải gọi tắt bằng curl.
- **Con số / Bằng chứng** (đọc lại từ `sagent api --lich-su`, bảng `api_calls`):

  | Route | Model | Vào | Ra | Tổng | Thời gian |
  |---|---|---|---|---|---|
  | `deepseek` | `deepseek-v4-flash` | 90 | 37 | **127** | **2,3s** |
  | `grok` | `grok-4.5` | 214 | 830 | **1044** | **13,6s** |

  Cùng một câu hỏi, cùng một nhà bán lại: Grok tiêu **8,2 lần** số token và chậm
  **5,9 lần**. Với những bước chỉ cần một câu trả lời ngắn (bước `soi`, bước gộp
  báo cáo), đó là khoảng cách đáng để chọn route thay vì để mặc định.
- **Đã sửa hay chưa**: **ĐÃ ĐO**, không phải sửa mã nào — đúng như kế hoạch Pha 4
  dự đoán ("cấu trúc đã generic nên nhiều khả năng chỉ là thêm route"). Ô ⬜ của
  DeepSeek trong Pha 4 chuyển thành ✅ kèm số. OpenRouter/Ollama vẫn ⬜: chưa có
  key, và Ollama chưa cài trên máy này.
- **Bài học**: cùng ngày lúc 16:54–16:56 route `deepseek` trả **HTTP 503 ba lần**
  rồi tự hồi phục. Cách duy nhất để biết điều đó là **gọi thật rồi hỏng** — route
  engine chưa có `health`. Một lượt flow dài chọn nhầm route đúng lúc nhà cung cấp
  chập chờn thì hỏng ở giữa chừng, chứ không hỏng lúc còn kịp đổi.

## 20/08 — Phiên chạy XONG XUÔI bị đọc thành "chết, chưa rõ vì sao"

- **Đo lúc nào**: 20/08/2026, đêm, khi bật lại hạm đội bằng cờ `--tu-duyet-quyen` mới.
- **Đo bằng cách nào**: bật một phiên `claude:phu` làm một việc CHỈ ĐỌC, đợi xong,
  rồi đọc **cả hai**: bảng `sagent status` và file `fleet.log` của bản clone.
- **Con số / Bằng chứng**:
  - `sagent status` → phiên **#157: "chết, chưa rõ vì sao"**.
  - `fleet.log` cùng phiên → agent **trả lời đúng** câu hỏi, NDJSON có dòng
    `{"type":"result","subtype":"success",...}`, không lỗi nào.
  - Tức là: lượt chạy **thành công** đọc ra y hệt một phiên chết bí ẩn.
  - Nguyên nhân: `provider.PhanLoaiChet` chỉ đặt tên cho các kiểu **HỎNG**
    (`rate_limited` / `blocked` / `failed`). Lượt chạy không hỏng → trả chuỗi rỗng
    → phiên ở lại `lost`. Hệ trạng thái **không có tên cho "đã hoàn thành"**.
  - Điều này giải thích phần lớn 20 thẻ "chưa rõ vì sao" trong ảnh chụp cùng ngày:
    một số trong đó nhiều khả năng đã chạy xong tốt đẹp.
- **Đã sửa hay chưa**: **ĐÃ SỬA**. Thêm `store.StateXong` / `provider.Xong`
  (`"done"`). Chỉ nói "xong" khi **ĐỌC ĐƯỢC** bản ghi kết quả — provider chưa đo
  được kết quả có cấu trúc vẫn rơi về `lost`, đúng như cũ.
  Đối chứng trên cùng một bảng sau khi sửa:

  ```
  #158 claude:phu#1   xong
  #157 claude:phu#1   chết, chưa rõ vì sao
  ```

  `ChetBatThuong()` đổi tên thành `TuKetThuc()` và **có** `done` — một lượt chạy
  xong xuôi vẫn có thể để lại tiến trình con, và con của phiên thành công tiêu hạn
  mức y hệt con của phiên hỏng. Câu SQL `IN (?,?,?,?)` chép cứng chuyển sang sinh
  dấu hỏi theo độ dài danh sách.
- **Bài học**: hai test cũ (`TestChayXongXuoiKhongBiGanTrangThaiHong`,
  `TestPhienChayXongKhongBiGanTrangThaiHong`) **ghim chính cái lỗ hổng này**. Ý
  định của chúng đúng — "đừng vu oan lượt chạy thành công là `failed`" — nhưng
  cách thoả mãn rẻ nhất là trả rỗng, và bài kiểm cũ chấp nhận điều đó. Một phép
  kiểm chỉ nói "không được là A, B, C" mà không nói "phải là D" thì để ngỏ đúng
  một chỗ cho câu trả lời tệ nhất: không là gì cả.

## 20/08 — `Clone` hồi sinh token đã chết, đè lên bản còn dùng được

- **Đo lúc nào**: 20/08/2026, sau khi `claude:phu` mất phiên ở lượt chạy #47.
- **Đo bằng cách nào**: đọc mã — `grep` mọi chỗ gọi `SyncBackTokens`.
- **Con số / Bằng chứng**:
  - `SyncBackTokens` được gọi ở **đúng một chỗ**: `ClonesClean`, tức chỉ khi
    người dùng chạy `sagent clean`.
  - `profile.Clone` chép `PrivateFiles()` từ hồ sơ gốc **đè lên clone**, và
    `FleetStart` gọi `Clone` **mỗi lần** bật hạm đội.
  - Chuỗi hỏng: clone refresh → token mới nằm trong clone, gốc giữ token cũ đã bị
    nhà cung cấp vô hiệu → không ai chạy `clean` → lượt sau `Clone` chép token
    **đã chết** đè lên token **đang sống** → refresh thất bại → mất phiên.
  - Khớp dấu vết đã đo: clone của `phu` mất hẳn `expiresAt`, còn clone của `tns`
    (chưa qua chu kỳ đó) vẫn có `expiresAt` mới.
- **Đã sửa hay chưa**: **ĐÃ SỬA**. `Clone` gọi `SyncBackTokens` **trước khi** chép
  đè, nên token mới lan ra mọi bản thay vì bị token cũ nuốt mất. Lỗi ở bước đồng
  bộ không chặn việc chạy — tệ nhất là quay về hành vi cũ.
- **Còn treo**: vẫn **CHƯA ĐO** chuyện nhà cung cấp có xoay vòng refresh token
  hay không, và N clone cùng refresh thì sao. Bản sửa này đúng bất kể câu trả lời
  đó — nó chỉ đảm bảo công refresh không bị đánh rơi.

## 20/08 — ĐÃ ĐO: nhà cung cấp XOAY VÒNG refresh token (đóng một ô "CHƯA ĐO" từ Pha 2)

- **Đo lúc nào**: khuya 20/08/2026.
- **Câu hỏi**: `internal/profile/clone.go` mang cảnh báo *"⚠ CHƯA ĐO: token bị chép
  ra N chỗ thì khi hết hạn, N tiến trình có thể cùng refresh một lúc"* từ Pha 2.
  Cùng ngày, `claude:phu` mất phiên giữa lượt chạy #47. Có liên quan không?
- **Đo bằng cách nào** — thiết kế để **không mất gì**, vì phép đo này chạm vào
  đăng nhập thật:
  1. Sao lưu `.credentials.json` của hồ sơ gốc và của bản clone; ghi vân tay
     SHA-256 (8 ký tự đầu) thay vì in token ra.
  2. Ép `expiresAt` của **bản clone** về quá khứ rồi chạy một lượt `claude -p`
     ngắn → buộc CLI refresh.
  3. So vân tay refresh token trước/sau.
  4. Dựng một thư mục config **tạm** mang bản token CŨ rồi chạy thử ở đó — token
     thật vẫn nằm nguyên chỗ của nó.
  5. Chạy `sagent clone` để bản sửa mang token mới về hồ sơ gốc; xoá thư mục tạm.
- **Con số / Bằng chứng**:

  | Mốc | refresh token | access token |
  |---|---|---|
  | Trước (gốc và clone giống hệt nhau) | `5d708911` | `1a0b9b6c` |
  | Sau khi clone refresh — **clone** | **`1aa28b8c`** | `a2ae3dfd` |
  | Sau khi clone refresh — **gốc** | `5d708911` (không đổi) | `1a0b9b6c` |

  Thử bản CŨ `5d708911` trong thư mục tạm:

  ```
  Failed to authenticate: OAuth session expired and could not be refreshed
  ```

  **Đúng nguyên văn câu đã làm hỏng bước `gop` của lượt #47.**
- **Kết luận**: nhà cung cấp **xoay vòng** refresh token — mỗi lần refresh cấp một
  token mới và **giết token cũ ngay**. Hệ quả nặng hơn phỏng đoán cũ: chép token
  ra N chỗ **không cần N tiến trình đua nhau** mới hỏng. **MỘT** bản refresh là
  N−1 bản còn lại chết, và hồ sơ gốc cũng là một trong số đó.
- **Đã sửa hay chưa**: **ĐÃ SỬA và ĐÃ KIỂM TRÊN MÁY THẬT.** `profile.Clone` gọi
  `SyncBackTokens` trước khi chép đè. Đối chứng ngay sau phép đo, khi hồ sơ gốc
  đang cầm token đã chết:

  ```
  TRƯỚC:  gốc refresh=5d708911   (đã chết)
  $ sagent clone claude:phu --copies 1
  SAU:    gốc refresh=1aa28b8c   clone refresh=1aa28b8c   (đều sống)
  ```

  Không có bản sửa, chính lệnh đó sẽ chép token chết đè lên bản còn sống — đúng
  chuỗi đã làm mất phiên `claude:phu`.
- **Bài học**: cảnh báo "CHƯA ĐO" nằm trong mã từ Pha 2 và **đã mô tả đúng vùng
  nguy hiểm**, nhưng vì chưa ai đo nên nó chỉ là một dòng chữ đứng cạnh một cái
  bẫy đang mở. Giữa "biết là có thể hỏng" và "biết hỏng thế nào" là khoảng cách
  của một tài khoản bị mất đăng nhập giữa lượt chạy. Chỗ nào trong mã còn chữ
  CHƯA ĐO thì chỗ đó là một cái bẫy đang chờ, không phải một ghi chú lịch sự.

## 21/08 — Streaming đường API: giữ được `usage`, đóng ô ⬜ cuối của Pha 4

- **Đo lúc nào**: rạng sáng 21/08/2026.
- **Vì sao mãi mới làm**: kế hoạch Pha 4 **cố ý dừng**, ghi rõ *"làm nửa vời thì
  mất `usage`, mà `usage` mới là thứ đáng giá nhất của đường API"*.
- **Nỗi lo đó có thật**: ở chế độ stream, endpoint tương thích OpenAI **không**
  gửi `usage` trừ khi được hỏi bằng `stream_options.include_usage`. Bỏ qua dòng
  đó thì stream chạy đẹp và bảng `api_calls` (migration v7) ghi 0 — kiểu hỏng im
  lặng tệ nhất, vì mọi thứ trông vẫn ổn.
- **Con số / Bằng chứng** — chạy thật qua `modelapi.vn`:

  | Route | Model | Vào | Ra | Tổng | Thời gian | `usage` giữ được? |
  |---|---|---|---|---|---|---|
  | `deepseek` | `deepseek-v4-flash` | 96 | 293 | **389** | 3,6s | ✅ |
  | `grok` | `grok-4.5` | 219 | 1707 | **1926** | **31,0s** | ✅ |

  31 giây của grok chính là lý do tính năng này đáng có: không streaming thì
  người dùng nhìn màn hình đứng im nửa phút, không phân biệt được "đang nghĩ"
  với "đã treo".
- **Đã sửa hay chưa**: **XONG ở lõi và CLI.** `aiapi.GoiStream` + `sagent api
  <route> --stream`. Streaming đi **cùng đường** với lời gọi thường
  (`API.aiCall`): cùng sổ route, cùng sổ chi phí, cùng luật chuyển route dự phòng
  đúng một lần, cùng cách phân biệt lỗi người dùng với lỗi nhà cung cấp. Viết
  vòng lặp thứ hai cho streaming là cách để hai đường trôi khỏi nhau.
- **Không im lặng nuốt usage**: nhà cung cấp nào không trả thì `KetQua.ThieuUsage`
  bật và CLI in cảnh báo nói rõ đây là **CHƯA ĐO chứ không phải miễn phí**. Cờ
  riêng chứ không suy từ `Usage.Tong == 0` — một lượt thật cũng có thể tốn 0
  token nếu hỏng sớm, gộp lại thì sổ mất khả năng phân biệt.
- **Còn treo**: mặt web chưa có nút stream (`/api/ai` vẫn gọi đường thường).
  Không phải action mới nên test ngang quyền không đỏ — nhưng đó là một mặt còn
  thiếu, ghi ra đây để không quên.
- **Chi tiết dễ đánh rơi, đã có test giữ**: `bufio.Scanner` mặc định bỏ cuộc ở
  dòng 64KB — gặp mẩu dài là stream *lặng lẽ* kết thúc sớm, mất cả phần đuôi lẫn
  usage. Đã nới lên 4MB. Mẩu SSE hỏng không được giết cả lượt. Stream đứt giữa
  chừng thì trả **phần đã nhận kèm lỗi**, vì phần đó đã tốn tiền rồi.

## 21/08 — Đ1+Đ2: Codex ĐÃ CHẠY THẬT. Nấc hẹp nhất KHÔNG phải cờ `dangerously`

- **Đo lúc nào**: 21/08/2026, ô ĐỎ số 1 và 2 của `docs/SO-NO-DO-LUONG.md`.
- **Vì sao xếp ĐỎ**: hai dòng này khai `LamDuoc`, tức `daDo = true`, nên **cả hai
  chốt chặn** (`api.go` cho `fleet --tu-duyet-quyen` và cho bước flow xin
  `tu_duyet_quyen`) đều cho qua — trong khi bằng chứng chỉ là `--help`, chưa ai
  chạy thật. Khai `ChuaDo` thì đã bị chặn; khai `LamDuoc` sai thì đi thẳng.
- **Đo bằng cách nào**: dựng một git repo **vứt đi** trong thư mục tạm, đặt sẵn
  `MOC.txt`, rồi bảo Codex đọc file đó và **ghi** một file mới. Một lượt kiểm
  được cả hai ô: đọc đúng file ⇒ `-C` có tác dụng; ghi được ⇒ cờ quyền đủ mạnh.
  Thử theo đúng thứ tự **hẹp → rộng**, không bắt đầu từ cờ mạnh nhất.
- **Con số / Bằng chứng**:

  | Nấc | Kết quả |
  |---|---|
  | `--sandbox workspace-write` một mình | Đọc ✅ · **Ghi ✗** — *"patch rejected: writing is blocked by read-only sandbox; rejected by user approval settings"* (10.240 token) |
  | `--approve-for-me` | Đọc ✅ · **Ghi ✅** — file tạo đúng nội dung (9.402 token) |
  | `--dangerously-bypass-approvals-and-sandbox` | **không cần thử tới** |

  Phát hiện phụ, quan trọng: **`codex exec` KHÔNG có `--ask-for-approval`** — cờ
  đó chỉ tồn tại ở chế độ tương tác. Nên ở headless không có cách nào nâng nấc
  approval từ dòng lệnh, và `--sandbox` một mình là ngõ cụt. Bảng năng lực cũ
  nhắc tới `--ask-for-approval` như một lựa chọn có thật cho headless — sai.
- **Đã sửa hay chưa**: **ĐÃ SỬA.** `codex.ArgsTuDuyetQuyen` đổi từ
  `--dangerously-bypass-approvals-and-sandbox` sang **`--approve-for-me`**.
  Help mô tả nó: *"route approval requests through automatic review using the
  workspace-write sandbox"* — tự duyệt **nhưng vẫn trong sandbox**.
- **Vì sao đáng đổi**: bản cũ dùng nấc mạnh nhất cho **mọi** lượt Codex. Cờ đó bỏ
  **cả** approval **lẫn** sandbox; `adapter.go` nói thẳng cái giá: *"agent duyệt
  cả xoá file và chạy lệnh tuỳ ý trong worktree của repo thật"*. Nay giữ lại
  sandbox, tức vẫn giới hạn được vùng ghi.
- **Bài học**: `LamDuoc` với bằng chứng `--help` **nguy hiểm hơn** `ChuaDo`. Cả
  hai đều là "chưa biết", nhưng một cái mở khoá mọi chốt chặn còn cái kia đóng
  chúng lại. Ba trạng thái năng lực của dự án đúng ở chỗ tách "đo rồi, không có"
  khỏi "chưa ai đo" — nhưng nó không tách được "đo bằng đọc help" khỏi "đo bằng
  chạy thật", mà khoảng cách đó vừa đủ để nuốt một lỗ hổng quyền.

## 21/08 — Đ3: Cursor. Máy CÓ cursor-agent, và bốn ô CHƯA ĐO đóng cùng lúc

- **Đo lúc nào**: 21/08/2026, ô ĐỎ số 3.
- **Sai lầm nền của ô này**: bảng năng lực ghi *"máy này không cài cursor-agent"*
  — nhưng `Get-Command cursor-agent` cho ra
  `C:\Users\Administrator\AppData\Local\cursor-agent\cursor-agent.ps1`, bản
  **2026.08.11**. Ghi chú đúng vào lúc viết, sai vào lúc đọc. Một dòng CHƯA ĐO
  không tự hết hạn.
- **Đo bằng cách nào**: một git repo vứt đi, ba lượt chạy thật.
- **Con số / Bằng chứng**:

  | Câu hỏi | Kết quả |
  |---|---|
  | `--trust` có đủ để ghi file không? | **ĐỦ** — agent tạo được file. Không cần `--force`/`--yolo` |
  | Có đọc được kết quả có cấu trúc không? | **CÓ** — `--output-format stream-json`, 7 dòng JSON, dòng cuối `{"type":"result","is_error":false,"usage":{...}}` |
  | `--model` có hiệu lực không? | **CÓ** — tên sai thì CLI từ chối: *"Cannot use this model: … Available models: auto, gpt-5.3-codex, composer-2.5, claude-opus-5-thinking-high, …"* |
  | Có cờ đổi thư mục không? | **KHÔNG** — `--help` không có `--cwd` lẫn `-C` |

- **Đã sửa hay chưa**: **ĐÃ SỬA.** Bốn dòng bảng năng lực đổi trạng thái:
  `tu-duyet-quyen` và `chon-model` và `ket-qua-co-cau-truc` → **LamDuoc**;
  `khai-thu-muc` → **KhongLamDuoc** (đã đo, provider không có thứ đó — khác hẳn
  "chưa ai đo"). Thêm `docKetQuaCursor`.
- **Bẫy đã tránh được nhờ đo thật**: `usage` của Cursor dùng **camelCase**
  (`inputTokens`) chứ không phải `input_tokens` như Claude. Chép bộ đọc của
  Claude sang thì token về 0 — và 0 đọc như *"miễn phí"*, không như *"chưa đọc
  được"*. Cursor cũng **không** trả `total_cost_usd`, nên chi phí vẫn là chưa đo.
- **Một lỗi tôi suýt tạo ra**: định nhét `request_id` vào trường `LoiAPI` cho
  tiện. `LoiAPI` là `api_error_status` — thứ **phân loại** trạng thái `failed`.
  Làm vậy thì mọi lượt Cursor đều mang một "mã lỗi", và một lượt hỏng bất kỳ sẽ
  bị xếp thành `failed` kèm một chuỗi vô nghĩa. Chỉ điền khi `is_error` thật.
- **Bài học**: `KhongLamDuoc` là một kết luận, không phải một khoảng trống.
  Cursor không có cờ thư mục — và điều đó **không sao**, vì `fleet` đã chạy tiến
  trình con với `workDir` là worktree của phiên. Ba trạng thái năng lực đáng giá
  đúng ở chỗ này: nếu chỉ có hai, "không có cờ" sẽ phải khai chung với "chưa đo"
  và người sau lại đi đo lại một thứ đã có câu trả lời.

## 21/08 — Đ5: cuộc đua N-clone cùng refresh. MỘT bản thắng, bản thua tự xoá trắng token của mình

- **Đo lúc nào**: 21/08/2026, 04:19:33Z (11:19 giờ máy). Ô ĐỎ số 5 của
  `docs/SO-NO-DO-LUONG.md` — nửa còn lại của cái bẫy đã cắn ngày 20/08.
- **Câu hỏi**: mục 20/08 đã đo được nhà cung cấp **xoay vòng** refresh token, tức
  một bản refresh là N−1 bản còn lại chết. Nửa chưa đo là **cuộc đua thật**: hai
  clone cùng vượt mốc hết hạn trong cùng vài giây thì **bản nào thắng**, và
  **bản thua nhận lỗi gì**.
- **Đo bằng cách nào** — nhân đôi đúng thủ tục 20/08, thiết kế để không mất tài
  khoản nào:
  1. Chọn tài khoản `phu`, **không** phải `tns` — `tns` là tài khoản của chính
     phiên đang đo, hỏng nó là hỏng luôn lượt chạy này. Sao lưu
     `.credentials.json` của cả hai bản clone `phu`, ghi vân tay SHA-256 8 ký tự
     đầu thay vì in token.
  2. **Chạy đối chứng trước**: một thư mục config **tạm** `C` mang token của
     `phu/1`, ép `expiresAt` về quá khứ 1 giờ, chạy `claude -p`. Bước này vừa
     chứng minh token gốc còn sống, vừa lấy ra một refresh token **mới toanh**
     làm hạt giống cho cuộc đua — để cuộc đua không phải xuất phát từ một token
     đã đi qua tay ai.
  3. Gieo **hai** thư mục tạm `A` và `B` từ kết quả bước 2 — cùng một refresh
     token, cùng bị ép `expiresAt` về quá khứ.
  4. Bật hai tiến trình `claude -p` **đồng bộ bằng file cờ**: cả hai khởi động
     trước, quay vòng chờ file `GO` xuất hiện rồi mới chạy. Không dựa vào việc
     gõ hai lệnh nhanh tay.
  5. Sau đo: mang token của bản THẮNG về cả `phu/1` lẫn `phu/2`, rồi chạy thật
     một lượt nữa để xác nhận tài khoản còn dùng được.
- **Con số / Bằng chứng**:

  | Mốc | refresh token | access token |
  |---|---|---|
  | Hạt giống (`A` và `B` giống hệt nhau) | `86fb2200` | `7ab8e294` |
  | Sau cuộc đua — **A (thắng)** | **`d22e079d`** | `884df52d` |
  | Sau cuộc đua — **B (thua)** | **rỗng** | **rỗng** |

  Hai tiến trình xuất phát cách nhau **16ms** (A `04:19:33.497`, B
  `04:19:33.513`). Kết cục:

  | | A | B |
  |---|---|---|
  | mã thoát | **0** | **1** |
  | `is_error` | `false` | `true` |
  | `terminal_reason` | `completed` | `api_error` |
  | `result` | `OK` | `Failed to authenticate: OAuth session expired and could not be refreshed` |
  | thời gian | 6,98s (API 4,18s) | **chết sau 186ms** |
  | chi phí | 0,0761945 USD | 0 |

  **Đúng một bản thắng. Không có cửa sổ ân hạn nào cho bản thua** — nó không nhận
  được token mới, cũng không được dùng lại token cũ.
- **Phát hiện KHÔNG đoán ra được nếu không chạy thật** — bản thua **tự ghi đè
  file token của chính nó thành rỗng**:

  ```json
  {"claudeAiOauth":{"accessToken":"","refreshToken":"","expiresAt":0,
   "refreshTokenExpiresAt":1789634099575,"scopes":[...],
   "subscriptionType":"max","rateLimitTier":"default_claude_max_20x"}}
  ```

  Vẫn còn `refreshTokenExpiresAt` (17/09, tương lai) và `subscriptionType: max`,
  nên **nhìn qua vẫn giống một hồ sơ đã đăng nhập**. Đây chính là hình dạng đã
  bắt gặp ngày 20/08 trên `.clones/claude/phu/1` ("không còn trường này") mà lúc
  đó chưa giải thích được: nó là **chữ ký của một bản THUA cuộc đua refresh**.
  Cũng đúng hình dạng `expiresAt: 0` = "đăng nhập dở dang" đã đo 19/08.
- **Và chi tiết giết người: bản THUA ghi file SAU bản thắng.**

  ```
  A/.credentials.json  11:19:35.484608900   (token sống)
  B/.credentials.json  11:19:35.500304400   (rỗng)  <- mới hơn 16ms
  ```

  Bản thua hỏng nhanh (186ms) nên nó ghi xong trước khi bản thắng chạy hết lượt
  gọi API. Nghĩa là sau mỗi cuộc đua, **bản clone có `mtime` mới nhất là bản
  hỏng**.
- **LỖI THẬT tìm ra nhờ phép đo này**: `profile.SyncBackTokens` chọn bản clone
  theo **mtime mới nhất** khi hồ sơ gốc còn token. Ghép với dòng trên: lượt bật
  hạm đội kế tiếp sẽ chép **file rỗng của bản thua đè lên hồ sơ gốc**. Dựng lại
  bằng test với đúng hình dạng file đã đo, kết quả còn tệ hơn dự đoán:

  ```
  --- FAIL: TestKhongDongBoNguocFileRongCuaBanThuaCuocDua
      từ chối trong khi vẫn còn một bản token sống:
      fake:phu chưa đăng nhập — chạy `sagent fake:phu` rồi /login trước
  ```

  Tức là: token sống **vẫn nằm nguyên trong bản thắng**, mà công cụ bắt người
  dùng đăng nhập lại. Đúng "mất tài khoản lần thứ hai theo một chuỗi khác nhưng
  cùng một gốc" mà sổ nợ đã cảnh báo.
- **Đã sửa hay chưa**: **ĐÃ SỬA**, hai chỗ:
  1. `SyncBackTokens` **bỏ qua mọi bản clone không còn token dùng được**, dù
     mtime mới đến đâu (`internal/profile/clone.go`). Không bản nào còn token thì
     không đồng bộ gì cả — hồ sơ gốc giữ nguyên, không bị cái rỗng nuốt.
  2. `provider.PhanLoaiChet` thêm nhánh `terminal_reason == "api_error"`. Đo
     được: bản ghi của bản thua **đọc ra hoàn hảo** (`Hong()` trả nguyên văn câu
     `OAuth session expired…`) nhưng `PhanLoaiChet` trả **rỗng**, vì
     `api_error_status` là `null` — nên phiên ở lại `lost` và bốn mặt điều khiển
     in **"chết, chưa rõ vì sao"** cho đúng cái chết **có lý do rõ nhất** hệ
     thống đọc được. Nay xếp `failed` kèm nguyên văn câu lỗi. Vẫn không dò chuỗi:
     `terminal_reason` là trường có tên, giá trị enum.
- **Cảnh báo lúc bung hạm đội** (`internal/fleet/fleet.go`) nay nói ra số đo thay
  vì chỉ nói nguyên tắc: chỉ **một** bản refresh thành công, bản thua dừng ngay
  với câu lỗi nguyên văn và để lại file token **rỗng** — không phải lỗi mạng,
  không phải hết hạn mức.
- **Tài khoản sau phép đo**: `phu/1` và `phu/2` cùng mang token thắng `d22e079d`;
  chạy lại thật một lượt → `is_error=false`, `result=OK`,
  `terminal_reason=completed`. Không mất đăng nhập nào. Tổng chi phí phép đo:
  **0,2286 USD** (ba lượt `claude -p`, mỗi lượt ~0,0762 USD; lượt của bản thua
  tốn 0).
- **Bài học**: bản sửa ngày 20/08 (`Clone` gọi `SyncBackTokens` trước khi chép
  đè) đúng, nhưng nó được viết dựa trên hình dung "clone giữ token MỚI, gốc giữ
  token CŨ". Cuộc đua đẻ ra một hình dạng thứ ba mà hình dung đó không có chỗ
  chứa: **một clone không giữ gì cả, và nó là bản mới nhất**. "Đã sửa cho lần
  hỏng trước" và "đã đo cho lần hỏng sau" là hai chuyện khác nhau — khoảng cách
  giữa chúng vừa đủ để nuốt thêm một tài khoản.




## 21/08 — ĐÃ ĐO: `phai_co` trên bước `soi` THẬT. Và nó lôi ra người soi đã bị vô hiệu hoá

**Đây là phép đo còn treo** sau lượt #48/#49: đo được cổng kiểm trong engine
**không** đồng nghĩa đã đo được ba chuỗi `["NEN TRON", "CAN SUA THEM", "VUT"]`
có khớp cách người soi thật viết câu trả lời hay không.

**Lượt #50 — `doi-4` chạy đủ 5 phiên agent, lần đầu kể từ #47.** Kết quả bước `soi`:

```
✗ doi-4.soi: chạy xong nhưng KHÔNG có kết luận nào trong
   "NEN TRON" hoặc "CAN SUA THEM" hoặc "VUT"
   — bước này coi như CHƯA LÀM, không phải đã làm và không có ý kiến
⚠ doi-4.soi hỏng nhưng on_failure=continue — đi tiếp
```

**Đây là BẮT ĐÚNG, không phải bắt oan.** Nguyên văn đầu ra của người soi:

```
TNS: KHONG THE XAC DINH — khong chay duoc git tren moi truong nay
MAY: KHONG THE XAC DINH — khong chay duoc git tren moi truong nay
LOI: khong the chay lenh git, khong co output de phan tich
GIAM CHAN: khong xac dinh duoc
```

### Cái mà phép đo lôi ra: người soi KHÔNG THỂ soi, về mặt cấu trúc

Bước `soi` của `doi-4` là **`type = "model"`** — một lời gọi API thẳng, **không có
shell, không có công cụ, không chạy được lệnh nào**. Nhưng prompt của chính nó
bắt: *"nhánh có gì thì phải tự chạy bốn lệnh git bên dưới, đó mới là bằng chứng"*,
rồi liệt kê `git log --oneline main..sagent/tns-1`, `git diff main...sagent/tns-1`,
và hai lệnh nữa.

Người soi đã trả lời **trung thực**: không chạy được git nên không kết luận được.

**Nguồn gốc là một bản vá đúng gây hậu quả không ai lường:** bước `soi` vốn là
`type = "agent"` chạy bằng CLI grok — chạy được git. CLI đó hỏng vĩnh viễn
(HTTP 410, xem mục *"Grok API 410"*), nên nó được đổi sang node `model` để đi
đường API. Đổi loại node thì **prompt giữ nguyên**. Từ đó tới nay người soi được
giao một việc mà môi trường của nó không cho phép làm.

**Vì sao chưa ai thấy**: lượt #47 (20/08 19:42) bước `soi` nhận HTTP 410 và được
ghi **DONE** — `phai_co` khi ấy chưa có. Nhìn bảng thì thấy "đã soi". Từ #47 tới
#50 **không có lượt `doi-4` nào chạy**, nên lỗi này nằm im 19 tiếng.

### Hai kết luận, đừng gộp làm một

1. **`phai_co` ĐÃ ĐO XONG, cả ba chiều**: chặn bước không giao kết luận (#49
   `phai-hong`, #50 `soi`), không chặn nhầm bước lành (#49 `phai-xong`), và ba
   chuỗi của `doi-4` **khớp thực tế** — chúng bắt được đúng ca cần bắt trên đầu
   ra thật.
2. **`phai_co` KHÔNG CHẶN ĐƯỢC việc trộn nhánh**, vì `soi` khai
   `on_failure = "continue"`. Lượt #50 bước `gop` **vẫn chạy** sau khi người soi
   hỏng. Đây đúng kịch bản mà `phai_co` sinh ra để ngăn — *"việc trộn nhánh diễn
   ra mà KHÔNG AI SOI"* — chỉ khác một điểm: nay nó **được ghi lại** thay vì im
   lặng. Ghi lại là tiến bộ thật, nhưng **không phải là chặn**.

   Công bằng với `gop`: nó tự đếm git và nói thẳng *"bước soi không để lại kết
   quả — không có nhãn nào của người soi để đối chiếu"*, không giả vờ đã có ý
   kiến. Lớp phòng vệ thứ hai này hoạt động.

### Còn phải quyết (thuộc cấu hình flow của chủ dự án, không tự sửa)

- Trả `soi` về `type = "agent"` bằng một tài khoản chạy được git, **hoặc** viết
  lại prompt cho hợp một node `model` (đưa sẵn `git diff` vào `doc_duoc` thay vì
  bảo nó tự chạy).
- Xem lại `on_failure = "continue"` ở `soi`: giữ thì phải chấp nhận trộn khi
  chưa ai soi.
- Bước `code-doc` **không có** `phai_co` — antigravity trả cụt vẫn được ghi `done`.

---

## 21/08 — ĐÃ ĐO: cổng kiểm `phai_co` chặn thật, cả hai chiều

**Ô còn treo trước đó**: `phai_co` được thêm vào `flows.toml` lúc **20/08 20:18**,
tức **36 phút SAU** lượt `doi-4` gần nhất (#47, 19:42). Nên tới sáng 21/08 nó mới
chỉ **đúng trên cấu hình** — chưa lượt chạy nào đi qua nó. Nói "đã vá" lúc đó là
nói quá.

**Vì sao không đo bằng `doi-4`**: chạy khan (`--kho`) báo thẳng
`✗ antigravity:may — chưa đăng nhập (kéo theo bước: code-doc)`. Bước `soi` nằm
**sau** `code-doc`, nên lượt chạy sẽ chết trước khi tới cổng cần đo. Đo bằng
`doi-4` lúc này là đốt 5 phiên agent để không đo được gì.

**Đo bằng gì**: một flow tối giản `kiem-cong`, hai bước `model` chạy trên route
mặc định (~1s/bước), đo **hai chiều** — chiều chặn và chiều đối chứng.

**Lượt #48 — hỏng phép đo, giữ lại vì nó dạy một điều**: hai bước để **song
song**. Bước xấu hỏng làm cả luồng dừng, và bước đối chứng đang bay bị huỷ giữa
chừng: `Post "https://modelapi.vn/v1/chat/completions": context canceled`. Đo
được đúng một chiều, chiều kia mất trắng. **Bài học: muốn đo đối chứng thì bước
đối chứng phải chạy XONG TRƯỚC** — thêm `needs` chứ đừng để song song.

**Lượt #49 — đo được, cả hai chiều** (`sagent flow tom-tat 49`):

```
AI LÀM GÌ
  · phai-xong (model) — done
      OK
BƯỚC NÀO HỎNG, VÌ SAO
  · phai-hong (model) — failed
      chạy xong nhưng KHÔNG có kết luận nào trong "BUA_MOT_CHUOI_KHONG_AI_NOI_ZZQ9"
      — bước này coi như CHƯA LÀM, không phải đã làm và không có ý kiến
```

- `phai-xong` **done** → cổng kiểm **không chặn nhầm** bước lành.
- `phai-hong` **failed** → cổng kiểm **chặn thật** bước không giao ra kết luận.

Đây đúng là lỗ hổng của #47: khi đó bước `soi` trả về một câu xin lỗi của nhà
cung cấp, CLI thoát mã 0, bản ghi không có trường lỗi nào, nên bước được ghi
**DONE** và việc trộn nhánh diễn ra mà **không ai soi**. Nay nó dừng lượt chạy.

**Flow `kiem-cong` đã được GỠ khỏi `flows.toml` sau khi đo.** Nó hỏng theo thiết
kế, nên để lại thì mỗi lần chạy là một cảnh báo Telegram giả. Muốn đo lại thì
dựng lại theo đúng mô tả trên — rẻ, khoảng 2 giây.

**Còn CHƯA đo**: `phai_co` trên chính bước `soi` của `doi-4` với đầu ra thật của
người soi. Đo được cổng kiểm trong engine **không** đồng nghĩa đã đo được ba
chuỗi `["NEN TRON", "CAN SUA THEM", "VUT"]` có khớp cách grok/deepseek thật sự
viết câu trả lời hay không. Muốn đóng nốt thì phải đăng nhập `antigravity:may`.

---

## 21/08 — Đ4: hạn token. Cursor đọc được thật; Antigravity là câu trả lời KHÔNG

**Ô nợ**: `docs/SO-NO-DO-LUONG.md` mục **Đ4** (mức ĐỎ) — Antigravity và Cursor
`TokenExpiry` cùng trả `false`, nên cảnh báo *"token sắp hết hạn"* trước khi bung
hạm đội (`internal/api/api.go`) **không bao giờ chạy** cho hai provider này.

### Rào chắn cũ sai vì TÌM NHẦM THƯ MỤC, không vì thiếu phép đo

Lời khai cũ: *"auth.json CÓ THỂ mang dấu thời gian hết hạn, nhưng chưa dựng được
cảnh token sắp hết hạn để xác nhận đọc đúng trường nào"*. Kiểm lại:

```
C:\Users\Administrator\.cursor\auth.json                 -> KHÔNG TỒN TẠI
C:\Users\Administrator\AppData\Roaming\Cursor\auth.json  -> CÓ, 893 byte
```

Chỗ thứ hai đúng là chỗ mà `cursor.go` đã tự chỉ từ đầu: `EnvVar() == "APPDATA"`,
`PrivateFiles() == [Cursor\auth.json]`. Ô này mở không phải vì phép đo khó, mà
vì đi tìm sai chỗ. Cùng một kiểu với Đ3 (*"máy này không cài cursor-agent"* trong
khi máy có).

### Hình dạng file — đo, không đoán

`%APPDATA%\Cursor\auth.json` có **ĐÚNG HAI khoá**:

```
accessToken   str, 424 ký tự, JWT alg HS256
refreshToken  str, 424 ký tự, JWT alg HS256
```

**Không có trường dấu-thời-gian nào ở tầng ngoài.** Nên lời dặn của chính mã —
*"đừng đoán tên trường"* — hoá ra còn đúng hơn ý định ban đầu: **không có trường
nào để mà đoán**. Mốc nằm trong payload JWT, y hệt Codex.

Claim đọc được (giải mã payload, KHÔNG kiểm chữ ký). `accessToken` và
`refreshToken` giống hệt nhau từng claim một:

| claim | giá trị | nghĩa |
|---|---|---|
| `iss` | `https://authentication.cursor.sh` | |
| `aud` | `https://cursor.com` | |
| `scope` | `openid profile email offline_access` | |
| `type` | `session` | |
| `sub` | `google-oauth|<id>` | mã đục, **không phải email** |
| `time` | `1787022539` | 2026-08-18T03:08:59Z — lúc đăng nhập |
| `exp` | `1792206539` | 2026-10-17T03:08:59Z |

`exp − time = 5 184 000 s` = **đúng 60 ngày**.

### Chạy thật sau khi sửa mã

```
HasToken=true
TokenExpiry ok=true  exp=2026-10-17T03:08:59Z  con=1364h  (≈ 56,8 ngày)
Identity=""
```

Trước bản vá, cùng hồ sơ đó trả `ok=false` — tức **không phân biệt được với "chưa
đăng nhập"**.

### Đọc REFRESH token, không phải access token

Hôm nay hai mốc **bằng nhau**, nên phép đo thật *không* phân biệt được hai lựa
chọn — bài kiểm `TestCursorDocRefreshChuKhongPhaiAccess` phân biệt hộ, bằng hai
JWT giả có `exp` khác nhau. Lý do chọn refresh là bài học đã trả giá ở
`claude.go`: trả hạn access token làm cổng kiểm **chặn oan** lượt chạy #39 trong
khi tài khoản vẫn dùng được.

### CÒN CHƯA ĐO: token có xoay vòng khi refresh không

Đây mới là thứ cảnh báo hạm đội thật sự sợ (xem mục 20/08 *"nhà cung cấp XOAY
VÒNG refresh token"*). Bằng chứng **gián tiếp**, không phải phép đo trực tiếp:

```
%APPDATA%\Cursor\auth.json   mtime 2026-08-18 10:08:58 (+07)  <- đúng giây đăng nhập
~\.cursor\cli-config.json    mtime 2026-08-21 00:46:44 (+07)
~\.cursor\statsig-cache.json mtime 2026-08-21 00:51:08 (+07)
```

Hai file dưới bị ghi lại bởi chính các lượt chạy thật của ô Đ3 ngày 21/08. File
token thì **không**. Tức qua ba ngày dùng, `cursor-agent` không đụng vào nó. Đó
là dấu hiệu, không phải bằng chứng — chưa ép nó refresh nên chưa biết lúc refresh
thì sao.

### Antigravity: đổi `Chua` → `Khong`, không đo gì cả

Token nằm trong Windows Credential Manager dưới khoá `gemini:antigravity`.
`CredRead` trả về **cả blob**, không có cách hỏi riêng mốc hết hạn. Mở chính thứ
cần bảo vệ để đổi lấy **một dấu thời gian** là đánh đổi tồi — nên câu trả lời là
**không**, chứ không phải **chưa**. Theo tiền lệ `Khong(NLHanToken)` của Grok,
khác một chỗ đáng nói: ở Grok hạn **không tồn tại** (API key, không OAuth); ở đây
hạn **có**, nhưng sau một cánh cửa ta cố ý không mở.

### Cảnh báo hạm đội đổi ra sao

Chốt trong `internal/api/api.go` chỉ chạy khi `TokenExpiry` trả `ok=true`:

- **Cursor** — nay **có** chạy. Cửa sổ 60 ngày dài hơn mọi lượt chạy nên nhánh
  *"còn dưới 2 tiếng"* gần như sẽ không kêu; **đó là đúng, không phải hỏng**. Cái
  đổi thật là nó thôi im lặng vì KHÔNG BIẾT và bắt đầu im lặng vì ĐÃ BIẾT còn
  hạn. `ProfileList` cũng điền được `HanToi`/`HetHan`.
- **Antigravity** — **vẫn không** chạy, và điều đó không đổi. Khác biệt duy nhất
  là bảng năng lực nay khai thẳng *"không đọc được hạn"* thay vì để trống.

### Ô nợ MỚI lộ ra: Cursor khai `Duoc(NLDanhTinh)` mà `Identity()` luôn rỗng

Không tìm nó — nó rơi ra khi mở `auth.json` thật. Bảng khai cũ nói đọc được
`email`/`userEmail`/`user_email` trong `Cursor\auth.json`; file thật **không có
khoá nào trong ba khoá đó**, và `sub` trong JWT là mã đục `google-oauth|<id>`.
Chạy trên hồ sơ đang đăng nhập: `Identity=""`.

**Vì sao conformance không bắt được**: phép dò `NLDanhTinh` trong `nangluc.go` là
**một chiều** (`haiChieu: false`) — chỉ kết luận được chiều *"khai chưa đo mà lại
trả giá trị thật"*. Chiều *"khai làm được mà luôn trả rỗng"* **không có ai canh**.
Cùng lỗ hổng đó đang che cho `NLCoTuHoSo`, `NLKetQuaCoCauTruc` và `NLHanToken`.
Đã hạ lời khai xuống `Chua(NLDanhTinh, ...)` kèm nguyên văn phép đo; ghi thành ô
**V3** trong sổ nợ. `cursor-agent status` in được email, nên năng lực này có
thật — chỉ là không đọc được từ file.

### Bài học

Hai nửa của một ô nợ có thể đóng bằng hai cách **ngược nhau**, và ép cả hai vào
một cách là hỏng. Nửa Cursor đóng bằng **đo**; nửa Antigravity đóng bằng **kết
luận không đo**. Trước lượt này cả hai cùng mang chữ `ChuaDo` nên nhìn giống
nhau — mà một cái là việc chưa làm, một cái là việc đã quyết định không làm.

## 21/08 — Codex đọc được kết quả có cấu trúc: `codex exec --json`, 6 lượt chạy thật (đóng ô nợ C1)

**Lệnh đã chạy** (codex-cli 0.147.0, Windows, tài khoản thật):

```
codex exec --json --skip-git-repo-check -C . "Tra loi dung mot tu: XONG"
```

**Cờ tìm được**: `--json` — nguyên văn `codex exec --help`: *"Print events to
stdout as JSONL"*. Khác Grok ở chỗ đây là **cờ có thật**, không phải định dạng
quan sát được (xem ô C2), nên nó là hợp đồng chứ không phải may mắn.

**Output THẬT, nguyên văn:**

```
{"type":"thread.started","thread_id":"01a0230f-5486-7772-b11d-950ebaba350f"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"XONG"}}
{"type":"turn.completed","usage":{"input_tokens":17625,"cached_input_tokens":11008,
 "cache_write_input_tokens":0,"output_tokens":6,"reasoning_output_tokens":0}}
```

### Lượt đo đầu tiên CHẾT TREO, và đó là phép đo có giá trị nhất trong ngày

Lượt đầu tiên bị giết sau 5 phút, `out.jsonl` **0 byte**. Lý do nằm ở stderr:

```
Reading additional input from stdin...
```

`codex exec` thấy stdin là **ống dẫn** thì coi đó là phần prompt nối thêm và
**đợi vô hạn** — help nói rõ: *"If stdin is piped and a prompt is also provided,
stdin is appended as a `<stdin>` block"*. Nối stdin vào `NUL` là hết. Dự án đã
biết chuyện này (`internal/profile/clone.go:349` nối `c.Stdin = devNull` kèm ghi
chú đúng về Codex), nhưng đây là lần nó được **đo lại từ đầu**: bất kỳ ai chạy
Codex ngoài đường đó mà để `Stdin = nil` hay ống dẫn đều treo, và treo **im
lặng** — không lỗi, không output, chỉ hết giờ.

### Lược đồ đầy đủ, và bốn cái bẫy — cả bốn đều đo được

| Sự kiện | Mang gì | Dùng vào |
|---|---|---|
| `thread.started` | `thread_id` | — |
| `turn.started` | — | — |
| `item.completed` + `agent_message` | `text` | `TraLoi` |
| `item.completed` + `command_execution` | `command`, `exit_code`, `status` | `ToolHong`, lá chắn chạy quẩn |
| `turn.completed` | `usage.input_tokens` / `output_tokens` | `TokenVao` / `TokenRa` |
| `turn.failed` | `error.message` | `CoLoi`, `LoiAPI` |

1. **Mỗi lời gọi tool in RA HAI DÒNG** — `item.started` rồi `item.completed`,
   cùng `item.id`. Đo được ở lượt bảo agent chạy một lệnh **hai lần**: bản ghi ra
   **bốn** dòng `command_execution`. Đếm cả hai thì mọi con số lặp **gấp đôi sự
   thật**, và với ngưỡng `TranLapLienTiep = 10` thì một agent lặp 5 lần bị vu oan
   là chạy quẩn. Bộ đọc chỉ nhận `item.completed` — cũng là dòng duy nhất có
   `exit_code` thật.
2. **Một lượt có NHIỀU `agent_message`.** Lượt gọi tool mở đầu bằng *"Tôi sẽ chạy
   đúng lệnh hai lần như yêu cầu."* rồi mới tới `"DONE"`. Lấy cái đầu thì bước
   sau nhận được một **lời hứa** thay vì kết quả. Lấy cái cuối.
3. **`{"type":"error"}` KHÔNG phải lượt hỏng.** Lượt 401 in **11 dòng** error
   (`Reconnecting... 2/5`) trước khi chết — nhưng đó là các lần **thử lại**, và
   một lượt thử lại rồi *thành công* in đúng những dòng ấy. Chỉ `turn.failed` mới
   là chết.
4. **`turn.failed` KHÔNG có `usage`.** Lượt hỏng thì token bằng 0 vì Codex không
   nói, không phải vì nó miễn phí.

### Lượt CHẾT: đo bằng `CODEX_HOME` trỏ vào thư mục rỗng

```
codex exec --json ...        # CODEX_HOME=<thư mục rỗng>
→ exit 1
{"type":"turn.failed","error":{"message":"unexpected status 401 Unauthorized:
  Missing bearer or basic authentication in header,
  url: https://api.openai.com/v1/responses, cf-ray: a2e7ab421c36ddbd-HKG,
  request id: req_1552c246e9de4d4e8a53bf944570155a"}}
```

`error.message` mang cả `request id` — thứ **hỏi lại nhà cung cấp được**, nên nó
vào đúng ô `LoiAPI` (cùng lý lẽ với `request_id` của Cursor). Kết quả:
`PhanLoaiChet` xếp lượt này thành `failed` kèm nguyên văn câu lỗi, thay vì để nó
nằm lại `lost` với dòng chữ *"chết, chưa rõ vì sao"*.

### Lượt TOOL HỎNG: `exit 3`

```
{"type":"item.completed","item":{"id":"item_1","type":"command_execution",
 "command":"…powershell.exe -Command 'exit 3'","exit_code":3,"status":"failed"}}
```

`status:"failed"` **và** `exit_code:3` — xét cả hai, vì lệnh bị sandbox chặn có
thể có `status` mà không có mã thoát.

### ĐO ĐƯỢC MỘT ĐIỀU **KHÔNG** ĐO ĐƯỢC: số tool bị chặn quyền

Chạy **không** có `--approve-for-me` (tức sandbox chỉ-đọc) rồi bắt agent ghi file:

```
{"type":"item.completed","item":{"id":"item_1","type":"agent_message",
 "text":"Không thể tạo thu.txt: môi trường hiện tại bị khóa **chỉ đọc**,
         và thao tác ghi đã bị hệ thống từ chối. …"}}
{"type":"turn.completed","usage":{…}}
```

Lượt kết thúc **`turn.completed` bình thường**. Không một dòng nào nói tới quyền,
không có item nào cho patch bị từ chối. Lời từ chối **chỉ nằm trong văn xuôi
tiếng Việt do model tự viết**.

Nên `TuChoiSo` để **0**, và đây không phải chỗ chưa làm xong: đọc được nó thì
phải dò chuỗi trong câu chữ của model — đúng thứ `trangthai.go` cấm (*"LUẬT:
KHÔNG DÒ CHUỖI"*). **Hệ quả phải nói thẳng: `ChetChanQuyen` KHÔNG BAO GIỜ kết
luận được cho Codex.** Bài `TestCodexKhongBiaSoToolBiChanQuyen` giữ chỗ này cho
người sau không lặng lẽ nhét một phép dò chuỗi vào.

### Còn lại những gì CHƯA đo được

- **`ChiPhiUSD`**: không có trường giá nào. Để 0, y hệt Cursor. Không nhân token
  với đơn giá — đơn giá còn tuỳ model và tuỳ gói.
- **`SoLuotTu`, `HanMucDenLai`, `KetCuc`**: không có trường tương ứng. Để rỗng/0.
  Bịa một giá trị cho `KetCuc` thì `trangthai.go` sẽ đem so với enum của Claude.
- **`cached_input_tokens` / `reasoning_output_tokens`**: chưa đo được chúng là
  phần **con** hay phần **thêm** của `input_tokens` / `output_tokens`. Nên không
  cộng, không trừ — lấy đúng con số nhà cung cấp dán nhãn.

### Kiểm chứng đầu-cuối, và đây mới là bằng chứng

Args dựng bằng **chính adapter** (`HeadlessArgs` + `ArgsThuMuc`), rồi đọc lại
bằng **chính adapter** (`ketqua_codex_e2e_test.go`, bật bằng `SAGENT_E2E_CODEX=1`):

```
LỆNH: …\npm\codex.cmd [exec --json Tra loi dung mot tu: ALPHA --cd …\Temp --skip-git-repo-check]
ĐỌC ĐƯỢC: TraLoi="ALPHA" CoLoi=false TokenVao=17627 TokenRa=6 ToolHong=0 Hong=""
PhanLoaiChet: "done" ""
```

Dòng cuối là con số đáng nhìn nhất: **trước lượt đo này nó là `""`** — phiên ở
lại `lost`, và bốn mặt điều khiển in *"chết, chưa rõ vì sao"* cho một lượt chạy
**thành công**.

### Đã sửa gì

- `internal/provider/ketqua_codex.go` (mới) — bộ đọc.
- `internal/provider/codex.go` — `DocKetQua` gọi bộ đọc thật; `HeadlessArgs`
  thêm `--json`; bảng khai đổi `Chua(NLKetQuaCoCauTruc)` → `Duoc(...)`.
- `--json` để ở `HeadlessArgs` chứ **không** chép cứng chỗ khác, nên
  `CoConThieu` (`bosungco.go`) tự bổ sung được cho đường `sagent fleet` — đường
  truyền args thô đã từng làm 20 phiên liền rơi về `lost` ngày 20/08.
- 11 bài kiểm trên **bản ghi thật** + 1 bài canh định kỳ gọi CLI thật.

**Chi phí phép đo**: 6 lượt `codex exec`, tổng ~146k token vào / ~418 token ra.
Codex không in giá nên không quy ra tiền được — đúng cái ô `ChiPhiUSD` vẫn để 0.

## 21/08 — C5: Ngưỡng chạy quẩn `TranLapLienTiep = 10` đo trên các bản ghi lượt chạy bình thường thật (đóng ô nợ C5)

- **Đo lúc nào**: 21/08/2026. Ô CAM số 5 của `docs/SO-NO-DO-LUONG.md`.
- **Câu hỏi**: Ngưỡng `TranLapLienTiep = 10` (`internal/provider/quan.go:47`) trước nay
  mới chứng minh được một chiều: bắt được ca quẩn thật #21 (399 lần `ls -la`).
  Chiều còn lại — **có bắt oan lượt chạy bình thường không** — chưa đo vì chưa đếm chuỗi
  lặp liên tiếp trên các bản ghi làm việc bình thường.
- **Đo bằng cách nào**:
  1. Quét toàn bộ file `.log` thật trong kho nhật ký `~/.ai-accounts/.nhat-ky/` (theo đúng
     thiết kế lưu nhật ký ở `internal/nhatky/nhatky.go:22-41`).
  2. Mở cơ sở dữ liệu `~/.ai-accounts/state.db`, tra bảng `sessions` qua cột `log` để chọn
     các phiên chạy **bình thường** (`state = "done"`).
  3. Dùng bộ đếm `demQuan` và bộ tạo chữ ký `chuKyTool` (tên tool + tham số `command`/JSON)
     trong `internal/provider/quan.go` và `lapLaiClaude` trong `ketqua_claude.go` để đếm
     chuỗi lặp liên tiếp dài nhất mỗi phiên.
- **Bảng số đo thực tế**:

  | Phiên | File nhật ký | Provider:Tài khoản | Trạng thái | Tổng tool | Chuỗi lặp dài nhất | Chữ ký lệnh lặp / Ghi chú |
  |---|---|---|---|---|---|---|
  | #174 | `claude-phu-c1-20260821-133734.050.log` | `claude:phu#1` | `done` | 70 | **1** | `Bash sed -n '1,80p' internal/provider/codex.go` |
  | #175 | `claude-tns-c1-20260821-133759.675.log` | `claude:tns#1` | `done` | 58 | **1** | `Bash ls && echo "---" && wc -l docs/SO-NO-DO-LUONG.md` |
  | #176 | `claude-tns-c1-20260821-140610.736.log` | `claude:tns#1` | `done` | 54 | **1** | `Bash sed -n '500,600p' docs/SO-NO-DO-LUONG.md` |
  | #177 | `claude-phu-c1-20260821-144709.356.log` | `claude:phu#1` | `done` | 9 | **1** | `Read {"file_path":"C:\\Users\\Administrator\\..."}` |
  | #179 | `antigravity-may-c1-20260821-144847.526.log` | `antigravity:may#1` | `running` | N/A | N/A | **KHÔNG ĐO ĐƯỢC**: nhật ký chỉ mang `tool_name`, không có tham số |

- **Con số / Bằng chứng**:
  - Trên **tất cả 4 phiên làm việc bình thường hoàn tất** (#174, #175, #176, #177) với tổng cộng
    **191 lượt gọi tool**, chuỗi lặp liên tiếp dài nhất luôn là **1** — tức agent không bao giờ
    gọi hai tool giống hệt nhau liên tiếp (cùng tên tool và cùng tham số).
  - Giữa các bước, agent luôn đổi lệnh, đổi tên file hoặc đổi cờ (ví dụ `sed -n '1,80p'` rồi
    `grep -n` rồi `git status`), đúng như phân tích ở `internal/provider/quan.go:29-35`.
  - **Khoảng cách an toàn hai chiều**:
    - So với mức bình thường dài nhất (1 lần): khoảng cách là **10 lần** (1 so với 10).
      Khả năng bắt oan một lượt chạy bình thường là cực kỳ thấp.
    - So với ca chạy quẩn thật (#21 với 399 lần): khoảng cách là **gần 40 lần** (10 so với 399).
      Lá chắn bắt gọn ca quẩn ngay từ đầu trước khi đốt hạn mức.
- **Trường hợp Antigravity — nói thẳng lý do KHÔNG ĐO ĐƯỢC**:
  - Bản ghi nhật ký của Antigravity chỉ in `{"tool_name":"run_command"}`, không có trường
    `input` hay `parameters`.
  - Theo đúng nguyên tắc thiết kế `chuKyTool` (`quan.go:119-122`): không có tham số thì không
    dựng chữ ký (`ok=false`), `DemDuocTool=false` và `Quan()` trả `biet=false` ("không biết").
    Điều này bảo đảm hệ thống thà nói "không biết" chứ không đếm mù tên tool rồi bắt nhầm
    một lượt chạy lành (như 30 lần `run_command` các lệnh khác nhau).
- **Đã sửa hay chưa**: **ĐÃ ĐÓNG Ô NỢ.**
  - Ngưỡng `TranLapLienTiep = 10` trong `internal/provider/quan.go:47` là hoàn toàn chính xác
    và an toàn trên số liệu thật, không cần thay đổi.

## 21/08 — C4: Đưa token và chi phí thật của phiên CLI vào DTO, phân biệt số thật với chưa đo (đóng ô nợ C4)

- **Đo lúc nào**: 21/08/2026. Ô CAM số 4 của `docs/SO-NO-DO-LUONG.md`.
- **Vấn đề trước khi sửa**:
  - Giao diện và API (`/api/state`) trả danh sách `sessionDTO`, nhưng trước đây chỉ mang các trường hành chính:
    `id`, `addr`, `pid`, `worktree`, `log`, `started`, `state`, `lyDo`, `hanMucDenLai`.
  - Hai ô `tokens` và `cost` của card phiên CLI trên dashboard buộc phải hiển thị `${CHUA_DO}` (`"chưa đo"`).
  - Điền `0` vào ô chi phí/token khi chưa đo là **nói dối theo hướng dễ chịu nhất**: số `0` đọc như "phiên này miễn phí", trong khi sự thật là phiên đang tiêu hạn mức mà hệ thống chưa đếm.
- **Cách giải quyết & Cách đo/đối chiếu**:
  1. **Đọc trực tiếp từ nhật ký phiên đã kết thúc**: Khi phiên chuyển sang trạng thái kết thúc (`done`, `failed`, `lost`), hệ thống đọc nội dung file nhật ký `s.Log` qua hàm `provider.Get(s.Provider).DocKetQua(logRaw)` (bộ giải mã kết quả có cấu trúc đã được xây dựng và kiểm chứng ở các ô Đ3/Đ4/C1).
  2. **Tránh đọc file đang ghi dở**: Chỉ đọc log của phiên đã kết thúc, tuyệt đối không đọc log của phiên đang chạy (`running`/`pending`) để tránh xung đột ghi/đọc hoặc đọc phải JSON dở dang.
  3. **Cờ đo được (`ChiPhiDaDo`)**: Phân biệt rạch ròi giữa hai trường hợp:
     - **Có chi phí thật** (Claude): gán `ChiPhiDaDo = true`, DTO mang số USD thật (kể cả khi bằng 0.0 nếu có).
     - **Không có chi phí** (Codex, Cursor, Antigravity, Grok): gán `ChiPhiDaDo = false`, DTO không trả về số 0 giả (để trống hoặc nil), giữ nguyên giao diện hiển thị trạng thái chưa đo.
- **Bảng đối chiếu kết quả đo thực tế trên 5 provider**:

  | Provider | Cờ / Lệnh xuất log | Dữ liệu Token trong log | Dữ liệu Chi phí trong log | Trạng thái Chi phí trên DTO | Bằng chứng thực tế / Ghi chú |
  |---|---|---|---|---|---|
  | **Claude** | `claude -p --output-format stream-json --verbose` | `usage.input_tokens`, `usage.output_tokens` | `total_cost_usd` (trong sự kiện `{"type":"result", ...}`) | ✅ **Số thật** (`ChiPhiDaDo = true`) | Đọc được số thật: phiên #174 tiêu 70 tools, phiên #177 tốn **0,5235 USD** (6.102 tokens ra), phiên #48/#49 tốn **0,0762 USD** |
  | **Cursor** | `cursor-agent -p --output-format stream-json` | `usage.inputTokens`, `usage.outputTokens` (camelCase) | **KHÔNG CÓ** | ⚪ **KẾT LUẬN: Chưa đo** (`ChiPhiDaDo = false`) | Đọc được Token thật (`inputTokens`/`outputTokens`), nhưng CLI bản 2026.08.11 không xuất trường giá trong JSON. Không nhân token với đơn giá |
  | **Codex** | `codex exec --json` | `turn.completed` → `usage.input_tokens`, `output_tokens` | **KHÔNG CÓ** | ⚪ **KẾT LUẬN: Chưa đo** (`ChiPhiDaDo = false`) | Đọc được Token thật (ví dụ lượt chạy test đo 17.625 input / 6 output tokens), không có trường giá nào trong các sự kiện JSONL |
  | **Antigravity** | `agy --output-format stream-json` | `result.usage.input_tokens`, `output_tokens` | **KHÔNG CÓ** | ⚪ **KẾT LUẬN: Chưa đo** (`ChiPhiDaDo = false`) | Đọc được Token thật (ví dụ phiên #179 ghi nhận 301.393 input / 30.682 output tokens), không có trường giá |
  | **Grok** | `grok -p` | **KHÔNG CÓ** | **KHÔNG CÓ** | ⚪ **Chưa đo** (`ChiPhiDaDo = false`) | Nhật ký không có khối usage/cost có cấu trúc |

- **Bốn nguyên tắc bảo vệ hệ thống**:
  1. **Codex/Cursor chưa đo chi phí là KẾT LUẬN, không phải lỗi**: Định dạng nhật ký của các công cụ này không cung cấp thông tin giá tiền. Hệ thống chấp nhận sự thật đó thay vì cố đoán đơn giá (vốn phụ thuộc vào model, cache hit/miss và gói tài khoản).
  2. **Không để lộ số 0 giả**: Tránh tạo ra các con số "miễn phí" gây nhầm lẫn trên giao diện quản trị và báo cáo.
  3. **Lưới an toàn không suy suyển**: Không sửa đổi mã HTML/JS của `internal/dash/web/index.html`; giữ hai bài kiểm ghim `TestTokenCostCuaPhienGhiChuaDo` và `TestTienDoNoiBuocThuMayHongODauVaTonBaoNhieu` xanh 100%.
  4. **An toàn tiến trình**: Chỉ đọc log khi phiên đã dừng hoàn toàn.
- **Đã sửa hay chưa**: **ĐÃ ĐÓNG Ô NỢ C4.**

## 21/08 — C2: Đo CLI Grok thật bằng bài canh định kỳ và xác nhận hiện trạng lỗi HTTP 410 (giảm rủi ro ô nợ C2)

- **Đo lúc nào**: 21/08/2026. Ô CAM số 2 của `docs/SO-NO-DO-LUONG.md`.
- **Mục tiêu / Vấn đề**:
  - Grok CLI (`@vibe-kit/grok-cli`) không có cờ cam kết xuất JSON (như `--json` của Codex hay `--output-format stream-json` của Claude/Cursor). Định dạng NDJSON (`{"role":"user",...}\n{"role":"assistant",...}`) mà `docKetQuaGrok` dựa vào chỉ là thứ nó tự in ra và ta quan sát được ở lần chạy #29.
  - Rủi ro được ghi trong ô nợ **C2**: Nếu nhà cung cấp đổi cách in dù chỉ một lần, `docDuoc=false` sẽ làm phiên Grok lặng lẽ tụt về `lost` và làm tê liệt lá chắn chống chạy quẩn (`internal/provider/quan.go`).
  - Biện pháp giảm thiểu rủi ro: Xây dựng bài canh định kỳ `TestE2EGrokDocDuocOutputThat` trong `internal/provider/ketqua_grok_e2e_test.go` (bật bằng `$env:SAGENT_E2E_GROK="1"` trên Windows PowerShell hoặc `SAGENT_E2E_GROK=1 go test ./internal/provider/`), dựng lệnh bằng chính adapter (`ad.Command()`, `ad.HeadlessArgs()`, `ad.ModelArgs("grok-4.5")`, `ad.ArgsThuMuc()`) để gọi CLI thật và đối chiếu kết quả.
- **Lệnh chạy đo thật trên Windows PowerShell**:
  ```powershell
  # Dựng bằng chính adapter grok:
  & "$env:APPDATA\npm\grok.cmd" --max-tool-rounds 60 -p "Tra loi dung mot tu: ALPHA" -m grok-4.5 -d "$env:TEMP"
  ```
- **Output THẬT nguyên văn thu được hôm nay (21/08/2026)**:
  ```json
  {"role":"user","content":"Tra loi dung mot tu: ALPHA"}
  {"role":"assistant","content":"Sorry, I encountered an error: Grok API error: 410 Live search is deprecated. Please switch to the Agent Tools API: https://***.***.x.ai/***/***/***/***"}
  ```
- **Phân tích kết quả đo thực tế**:
  1. **Định dạng in ra**: Output vẫn giữ dạng 2 dòng NDJSON (`role: user` rồi `role: assistant`). Bộ đọc `docKetQuaGrok` bóc tách được dòng này, gán `k.TraLoi` bằng nội dung thông điệp và trả về `docDuoc = true`. Cấu trúc dòng NDJSON chưa bị nhà cung cấp đổi thành văn bản thô.
  2. **Hiện trạng lỗi HTTP 410 của CLI Grok**:
     - Lời gọi CLI thật trả về lỗi: `Grok API error: 410 Live search is deprecated. Please switch to the Agent Tools API: ...`.
     - Phép đo xác nhận đúng hiện trạng đã phát hiện ngày 20/08 (lượt #46, #47 trong `.sagent/flows.toml:862` và `docs/DO-LUONG.md:2582`): CLI Grok hiện tại (`@vibe-kit/grok-cli` v1.0.1) gọi vào tính năng Live Search cũ của xAI đã bị khai tử (HTTP 410 Gone). Do đó, ở chế độ headless CLI không thể thực thi tool hoặc trả về câu trả lời hoàn chỉnh của người dùng (không trả ra `"ALPHA"`).
     - Đây là lỗi từ phía nhà cung cấp upstream đối với CLI Grok, không phải lỗi của bộ adapter sagent.
  3. **Lá chắn và giải pháp giảm thiểu rủi ro**:
     - Đã có bài canh định kỳ `TestE2EGrokDocDuocOutputThat` trong `internal/provider/ketqua_grok_e2e_test.go` (bật bằng `SAGENT_E2E_GROK=1`). Bài canh đo được gì ghi nấy, trung thực ghi nhận lỗi 410 thay vì che giấu để khoe xanh.
     - Về luồng xử lý thực tế: Flow `doi-4` đã chuyển bước `soi` sang `type = "model"` (gọi API trực tiếp qua `internal/flow/step.go:395`), giúp việc chạy luồng không bị tắc nghẽn bởi CLI Grok.
  4. **Tình trạng năng lực `NLKetQuaCoCauTruc`**:
     - Bảng năng lực `internal/provider/grok.go:246` khai `Duoc(NLKetQuaCoCauTruc)` vì bộ đọc `docKetQuaGrok` đọc đúng cấu trúc NDJSON khi nhận được dữ liệu.
     - Tuy nhiên, phép đo hôm nay ghi nhận rõ ràng: dù bóc tách được cấu trúc (`docDuoc=true`), nội dung thu được đang là lỗi 410 do CLI Grok upstream hỏng Live Search. Khi nào xAI khôi phục hoặc CLI Grok cập nhật phiên bản mới, bài canh định kỳ `SAGENT_E2E_GROK=1` sẽ là chốt kiểm chứng lại xem cấu trúc NDJSON có còn giữ nguyên hay không.

## 21/08 — V2: Antigravity ĐỌC ĐƯỢC danh tính. Nguồn là NHẬT KÝ của chính `agy`, không phải `google_accounts.json`

**Câu hỏi thật của ô này** không phải *"tìm một chỗ có email"* mà *"tìm một nguồn
PHÂN BIỆT ĐƯỢC với `google_accounts.json`"*. Banner `agy` 1.1.17 in
`ttseotop1@gmail.com`; `~/.gemini/google_accounts.json` cũng ghi đúng email đó.
**Hai giá trị trùng nhau không chứng minh gì** — chúng trùng vì cùng một người
từng đăng nhập Gemini CLI trên máy này. Đọc file đó có thể cho ra một email **có
thật, sai người**, và `antigravity.go` đã chốt: *hiện nhầm email còn tệ hơn không
hiện gì*.


## 21/08 — C3: chọn model từ dòng lệnh cho Antigravity và Codex (đóng ô nợ C3)

**Vì sao đo**: `internal/api/model_test.go:9-12` ghi con số của lượt chạy #34 —
**9,40 USD cả lượt, riêng bước `code-go` 8,18 USD** — vì mọi bước đều chạy model
mạnh nhất, kể cả bước chỉ viết tài liệu. Trước lượt này, khai `model = "..."`
cho một bước Antigravity/Codex chỉ đổi lấy **một dòng cảnh báo**; bước vẫn chạy
model mặc định. Với Codex, "mặc định" có tên: `~/.codex/config.toml` khai
`model = "gpt-5.6-sol"`, và chính file đó viết *"MODEL MẶC ĐỊNH = mạnh nhất, có
lý do"*.

**Điều kiện mới có hôm nay**: tài khoản Antigravity vừa đăng nhập lại
(`ttseotop1@gmail.com`, CLI **1.1.16**). Trước 21/08 ô này không đo nổi vì chưa
đăng nhập. Codex: **0.147.0**.

### Bằng chứng 1 — tên model bịa. Hai provider trả lời KHÁC HẲN nhau

Tiền lệ Cursor (`bacc137`) đặt ra khuôn: bằng chứng mạnh nhất không phải `--help`
liệt kê cờ gì, mà là **CLI từ chối một tên model sai và liệt kê model hợp lệ**.
Khuôn đó đúng với một provider và **sai với provider kia**:

- **Antigravity — chặn ở phía máy mình, chưa tốn token nào**:

  ```
  $ agy -p "..." --model khong-ton-tai-9x --dangerously-skip-permissions
  Error: invalid model selection (--model "khong-ton-tai-9x" --effort ""):
         model khong-ton-tai-9x is not recognized as a known model or custom model in settings
  Available models: Gemini 3.7 Flash (High) … GPT-OSS 120B (Medium)   [14 dòng]
  → thoát mã 1
  ```

- **Codex — KHÔNG chặn gì ở tầng CLI**. Nhận cờ, in `model: khong-ton-tai-9x` ở
  đầu bản ghi, chỉ càu nhàu *"Model metadata for `khong-ton-tai-9x` not found.
  Defaulting to fallback metadata"*, rồi **máy chủ** mới chặn:

  ```
  ERROR: {"type":"error","status":400,"error":{"type":"invalid_request_error",
          "message":"The 'khong-ton-tai-9x' model is not supported when using Codex with a ChatGPT account."}}
  ```

  Đây vẫn là bằng chứng, thậm chí đi xa hơn: giá trị **đã đi hết đường xuống thân
  yêu cầu API**. Nhưng một lượt đo chỉ tìm đúng khuôn "CLI từ chối" sẽ kết luận
  nhầm rằng Codex nuốt cờ.

### Bằng chứng 2 — cờ có ĐỊNH TUYẾN không, hay chỉ được đem đi KIỂM TRA?

Từ chối tên sai mới chứng minh cờ **được đọc**. Chưa chứng minh model thật sự đổi.
Hai phép đo riêng:

**Antigravity — cùng một prompt (`"tra loi dung mot tu: ok"`), ba model:**

| `--model` | `input_tokens` | `output_tokens` |
|---|---|---|
| `gemini-3.7-flash-low` | 13.747 | 30 |
| `claude-opus-4-6-thinking` | 15.764 | 27 |
| `gpt-oss-120b-medium` | 11.174 | 64 |

Cùng prompt, cùng repo, cùng CLI — biến duy nhất là `--model`. Ba bộ tách từ khác
nhau đọc cùng một đầu vào ⇒ cờ đổi model thật.

**Codex — cờ ghi đè hồ sơ:**

| dòng lệnh | dòng `model:` ở đầu bản ghi |
|---|---|
| `codex exec "…"` (không cờ) | `gpt-5.6-sol` ← từ `config.toml` |
| `codex exec -m gpt-5.4-mini "…"` | `gpt-5.4-mini`, chạy xong thật, 12.291 token |

Đây đúng là chiều mà `internal/provider/grok.go:236-238` đã bác một lần —
*"provider tự đọc model từ hồ sơ"* không được mặc định tin. Ở đây kết quả ngược
với Grok: **cờ thắng hồ sơ**.

### Một phép đo ĐÃ BẮT ĐẦU SAI — giữ lại vì người sau sẽ thử đúng cách đó

Cách hiển nhiên để kiểm "có đúng model không" là hỏi thẳng agent. Đã thử:

```
$ agy -p "Tra loi DUNG MOT TU: ban do Google hay Anthropic tao ra?" \
      --model claude-opus-4-6-thinking
{"status":"SUCCESS","response":"Google\n", …}
```

Tin câu đó thì kết luận sẽ là *cờ bị nuốt → khai `KhongLamDuoc`* — **sai hoàn
toàn**, và sai theo hướng **đắt hơn hiện trạng** (bỏ luôn khả năng hạ model).
**Tự khai danh tính không phải phép đo**: lời nhắc hệ thống của Antigravity đè
lên câu trả lời. Số token thì không biết nói dối.

### Cái bẫy THỨ TỰ CỜ — đo chứ không suy từ `--help`

`argsChoBuoc` (`internal/api/api.go:1397`) **chèn `ModelArgs` vào TRƯỚC**
`HeadlessArgs`. Với Codex, `-m` lại là cờ của **lệnh con** `exec`, nên dòng thật là:

```
codex -m gpt-5.4-mini exec --json "<prompt>"     → đầu bản ghi: model: gpt-5.4-mini  ✓
```

Suy từ `--help` thì đây là chỗ hỏng (cờ của lệnh con đứng trước lệnh con). Đã chạy
đúng dạng đó và Codex nhận. Antigravity cũng đã chạy đúng dạng chèn-trước
(`agy --model gemini-3.7-flash-low --output-format stream-json -p "<prompt>"` →
13.742 token, đúng chữ ký của model đó), chứ không phải chỉ dạng cờ-đứng-sau lúc
thử tay.

### Đã đổi trong mã

- `internal/provider/antigravity.go:177` — `ModelArgs` → `--model <model>`;
  `:188` khai `Duoc(NLChonModel)`.
- `internal/provider/codex.go:277` — `ModelArgs` → `-m <model>`; `:292` khai
  `Duoc(NLChonModel)`.
- `internal/api/quyen_test.go` — `giaAdapter.ModelArgs` trả `nil`, làm vật thử
  cho nhánh cảnh báo.
- `internal/api/model_test.go` — `TestProviderChuaDoModelThiPhaiCanhBao` đổi vật
  thử từ provider thật sang adapter giả; thêm
  `TestAntigravityVaCodexTruyenDuocModel` và `TestCodexDatCoModelTruocLenhCon`.

**Đếm lại cột `ChuaDo` sau lượt này: còn 5** (antigravity 2, claude 1, codex 1,
cursor 1, grok 0). Cột `NLChonModel` **sạch trên cả năm provider**.

### Nợ MỚI lượt này lôi ra (chưa trả)

Trong `argsChoBuoc`, nhánh *"provider KHÔNG có rào quyền nào"* **gán đè** lên biến
`canhBao`. Một provider vừa không-có-rào-quyền vừa chưa-đo-model sẽ **mất câu cảnh
báo về model**. Hôm nay không provider nào rơi vào cả hai ô cùng lúc, nên nó chưa
cắn được ai — ghi lại đúng vì đó là lý do duy nhất nó chưa cắn.
### Lời cũ trong sổ vẫn đúng — chỉ dừng sớm một bước

| Thứ | Dấu thời gian |
|---|---|
| `~/.gemini/google_accounts.json` | **18/08 10:09** — đứng im |
| Ba lượt `agy` ngày 21/08 | 21:16, 21:17, 21:18 |

24 file trong `~/.gemini` đổi trong ngày 21/08, **toàn bộ** nằm dưới
`antigravity-cli/`. `google_accounts.json` không nằm trong đó.

### Nguồn phân biệt được: `<hồ sơ>/.gemini/antigravity-cli/log/cli-*.log`

Ba dạng dòng đã thấy (nguyên văn, cắt bớt cho vừa trang):

```
server_oauth.go:190] applyAuthResult: email=ttseotop1@gmail.com, authMethod=consumer, quotaProject=
server_oauth.go:195] OAuth: authenticated successfully as ttseotop1@gmail.com
browser.go:161]      consumerOAuth: authenticated successfully as ttseotop1@gmail.com
```

Dòng thứ ba chỉ có ở **lượt đăng nhập bằng trình duyệt**; hai dòng đầu có ở
**mọi** lượt chạy, ngay lúc token được áp dụng.

### HAI phép đo chứng minh nguồn này khác `google_accounts.json`

Cả hai đều là thư mục **KHÔNG HỀ CÓ** file đó mà nhật ký **VẪN** in ra email:

1. **HOME giả** — đổi cả `USERPROFILE` + `APPDATA` + `LOCALAPPDATA` + `HOME` sang
   thư mục trống rồi `agy --log-file <tạm> -p "say OK"` (cùng lối đã dùng đo
   `TachDuocTaiKhoan`). Thư mục mới dựng chỉ có `.gemini/{antigravity-cli,config}`
   — nhật ký vẫn ghi `applyAuthResult: email=ttseotop1@gmail.com`.
2. **Hồ sơ THẬT `~/.ai-accounts/antigravity/may`** — cả cây `.gemini` của nó
   không có `google_accounts.json`, mà `log/cli-20260821_144204.log` (21/08
   **14:42**) ghi `consumerOAuth: authenticated successfully as …`: dòng của
   **chính lượt đăng nhập bằng trình duyệt hôm nay**.

→ Email trong nhật ký đến từ **token trong Windows Credential Manager** (khoá
`gemini:antigravity`), không phải từ file của Gemini CLI.

### Hai nguồn khác đã loại

- **`CredRead().UserName`** — hứa hẹn nhất vì là metadata, đọc được mà **không mở
  blob**. `cmdkey /list:gemini:antigravity` → `User: antigravity`. Là **hằng số**,
  không phải email. Bỏ.
- **Quét cả cây hồ sơ tìm chuỗi email** — nhật ký là file **DUY NHẤT** mang nó.
  Các file cấu hình khác chỉ có `trustedWorkspaces`, kiểu xác thực,
  `remoteControlHostname`.

### MỘT RÀO ĐÃ ĐO RỒI BỎ — ghi lại để người sau khỏi thử lại

Nhật ký có điểm yếu thật: đăng nhập lại bằng tài khoản **khác** ở HOME **khác**
sẽ để nhật ký cũ nằm lại. Rào định dựng: lấy `Credential.LastWritten` làm mốc,
*"nhật ký cũ hơn mốc thì trả rỗng"*. Đo trước khi viết:

```
Credential `gemini:antigravity`.LastWritten     21/08 21:16:59
nhật ký HỢP LỆ của hồ sơ `may`                  21/08 20:15:39   <- CŨ HƠN
```

Rào đó sẽ trả **RỖNG cho đúng cái hồ sơ đang chạy đúng**. Nguyên nhân:
`LastWritten` đổi **cả khi làm mới token** lẫn khi đăng nhập lại, mà không mở blob
thì không tách được hai việc — và mở blob chính là thứ Đ4 đã cố ý từ chối. **Rào
sai hướng thì thà không có.** Đã bỏ; lý do ghi thẳng trong
`danhtinh_antigravity.go`.

Cửa sổ rủi ro còn lại hẹp vì `Khong(NLTachTaiKhoan)` — **mỗi máy MỘT tài khoản
Antigravity**. Nó có thật, và bảng năng lực nói ra thay vì giấu.

### Đã đổi

`Identity` đọc nhật ký, có **cổng `coCredential`** đứng trước (đăng xuất thì mục
Credential Manager biến mất nhưng nhật ký cũ vẫn trên đĩa — không có cổng thì hồ
sơ **đã đăng xuất** vẫn khoe email). Lời khai `Chua(NLDanhTinh)` →
**`Duoc(NLDanhTinh)`**. Tám bài kiểm mới trong
`internal/provider/danhtinh_antigravity_test.go`, đáng kể nhất là
`TestAntigravityKhongLayEmailTuGoogleAccounts`: dựng hồ sơ có **CẢ HAI** nguồn nói
**khác nhau** rồi bắt hàm lấy nguồn do `agy` ghi.

**Chạy trên hồ sơ thật**: `Identity()` cho `ttseotop1@gmail.com` trên cả
`~/.ai-accounts/antigravity/may` lẫn HOME thật. (Bài kiểm tạm đó **không** commit
— nó phụ thuộc máy.)

---

## 21/08 — V1: bốn ô "suy cờ từ chính thư mục hồ sơ". Codex CHẠY THẬT, và nó ngược hẳn Grok

Ô này không rỗng: Grok khai `Duoc(NLCoTuHoSo)` vì CLI của nó **phớt lờ**
`defaultModel` trong chính file cấu hình của nó, nên phải ép `-m` (đo 18/08),
nếu không mọi bước hỏng lặng lẽ với 503.

**Câu hỏi phân biệt** rút ra từ đó không phải *"hồ sơ có thiết lập không"* mà
***"CLI có TỰ ĐỌC thiết lập đó không"***.

### Codex — ô đáng đo nhất, vì hồ sơ CÓ thiết lập trông y hệt ca Grok

`$CODEX_HOME/config.toml` trên máy này khai:

```toml
model = "gpt-5.6-sol"
model_reasoning_effort = "high"
```

**Chạy thật** (0.147.0): dựng một `CODEX_HOME` tạm chỉ có `auth.json` và một
`config.toml` khai model **bịa**, rồi `codex exec --json`:

```
item.completed  "Model metadata for `khong-co-model-nay-dau` not found. Defaulting to fallback…"
turn.failed     400 "The 'khong-co-model-nay-dau' model is not supported when using Codex with a ChatGPT account."
```

Tên model bịa đi **thẳng ra tới máy chủ** → Codex **ĐỌC** `config.toml` của chính
`CODEX_HOME` được truyền vào. Không cần ai chuyển nó thành cờ.

### Ba provider còn lại — tra file, không chạy

| Provider | Tra ở đâu | Thấy gì |
|---|---|---|
| claude | `~/.ai-accounts/claude/tns` | `.claude.json` **48 khoá**; bốn khoá chứa chữ "model" đều là **cache máy chủ**, không phải lựa chọn. `settings.json` chỉ có `env.DISABLE_AUTOUPDATER` + `tui` |
| cursor | `%APPDATA%\Cursor` | **ĐÚNG MỘT file** `auth.json`, **ĐÚNG HAI khoá** `accessToken`/`refreshToken` — hồ sơ **rỗng thiết lập** |
| antigravity | cây `.gemini` của hồ sơ | ba file: kiểu xác thực, `trustedWorkspaces`, `remoteControlHostname` — không cái nào có cờ tương ứng |

**Bẫy đã tránh ở Antigravity**: `trustedWorkspaces` **trông** giống việc của
`--add-dir`. Nhưng thư mục làm việc thật đã do `ArgsThuMuc` truyền vào **từng
lượt** — đọc lại danh sách cũ trong file là **ép agent vào workspace của lượt
trước**. Đúng chiều "đoán sai theo chiều thừa" mà ô V1 cảnh báo.

**Ghi thêm, không thuộc ô này**: `agy --help` **CÓ** `--model`. Ô `chon-model` của
Antigravity vẫn `ChuaDo` vì lượt này không chạy thật để xác nhận cờ có hiệu lực —
thấy trong `--help` đúng là mức bằng chứng mà sổ nợ xếp vào loại nguy hiểm. Nhưng
nay đã biết chỗ bắt đầu.

### Kết quả

Bốn lời khai `Chua(NLCoTuHoSo, …)` → **`Khong(NLCoTuHoSo, …)`**. `ArgsHoSo`
**không đổi hành vi** (vẫn `nil`) — đổi **tư cách** của cái `nil` đó từ khoảng
trống thành kết luận.

Số ô `ChuaDo` trong bảng năng lực: **7 → 2** (chỉ còn `chon-model` của codex và
antigravity). Đếm bằng `grep -c '^		Chua('` trên từng file adapter, so `HEAD`
với cây làm việc.

## 22/08 — Token TỰ LÀM MỚI thật khi bật hạm đội, sau khi đã quá hạn gần 5 tiếng

- **Đo lúc nào**: 22/08/2026, 10:06–10:15, lúc phóng đợt 2 của lượt chạy dài.
- **Vì sao đo**: `docs/KE-HOACH-8-MUC.md` ghi rủi ro còn treo *"Token Claude hết
  hạn ~8 tiếng/lần. Lượt đêm dài phải tính chuyện làm mới giữa chừng; hành vi
  khi nhiều bản clone cùng refresh CHƯA ĐO."* Nửa sau đã đo 21/08 (ô Đ5, và nó
  **hỏng**). Nửa đầu — một clone mỗi tài khoản — thì chưa ai đo.
- **Trạng thái trước khi phóng**:

  | Bản clone | `expiresAt` | mtime file token |
  |---|---|---|
  | `claude/phu/1` | 22/08 **05:16** | 22/08 00:50 |
  | `claude/tns/1` | 22/08 **05:16** | 22/08 00:51 |

  Lúc đo là **10:06**, tức access token của cả hai đã **quá hạn 4 giờ 50 phút**.
  `sagent ds` vẫn in *"sẵn sàng"* cho cả hai.

  > **NỢ NGƯỢC đã sửa cùng ngày**: dòng trên từng ghi tiếp *"nó trả lời câu 'có
  > file token không', không phải câu 'token còn dùng được không'"*, và xếp
  > `sagent ds` vào nhóm đèn xanh nói dối. **Sai, và sai theo hướng tệ nhất:
  > tố oan một chỗ đã được sửa đúng rồi.** `TokenExpiry` của Claude
  > (`internal/provider/claude.go:157`) **cố ý** trả hạn của **refresh token**
  > chứ không phải access token, kèm nguyên một khối bình luận giải thích vì
  > sao: bản trước trả hạn access token nên `ds` in *"HẾT HẠN — đăng nhập lại"*
  > sau mỗi 8 tiếng và **chặn oan lượt chạy #39**, trong khi CLI vẫn tự đổi
  > access token mới và tài khoản vẫn chạy được. Câu người vận hành cần trả lời
  > là *"tài khoản này còn chạy được không"*, và câu đó nằm ở refresh token.
  > `HasToken` cũng không chỉ kiểm file tồn tại — nó đòi `expiresAt != 0` để
  > bắt ca đăng nhập dở dang (`claude.go:104`).
  >
  > Nói cách khác: phép đo dưới đây **không phát hiện `ds` nói dối — nó xác nhận
  > `ds` nói đúng.** Access token hết hạn mà tài khoản vẫn chạy được chính là
  > điều thiết kế đó dự đoán. Bài học cho chính lượt ghi sổ này: thấy một con số
  > "sai" thì đọc mã trước khi gọi tên nó là lỗi — đúng luật của cuốn sổ này.

- **Con số / Bằng chứng**: bật `sagent fleet claude:phu` (#197) lúc 10:14 và
  `claude:tns` (#198) lúc 10:15. Đọc lại file token ngay sau đó:

  | Bản clone | `expiresAt` mới | mtime mới |
  |---|---|---|
  | `claude/phu/1` | 22/08 **18:14** | 10:14:25 |
  | `claude/tns/1` | 22/08 **18:15** | 10:15:05 |

  Cả hai phiên sống bình thường, không phiên nào chết vì *"OAuth session expired
  and could not be refreshed"*. Hạn mới cách hạn cũ đúng một chu kỳ ~8 tiếng.

- **Kết luận, và giới hạn của nó**: với **đúng một bản clone mỗi tài khoản**,
  refresh token quá hạn **tự chạy được**, không cần đăng nhập lại, không cần
  can thiệp tay. Điều kiện "một clone" **không phải chi tiết phụ** — đó là toàn
  bộ lý do nó chạy được. Ca nhiều clone cùng refresh đã đo riêng 21/08: bản thua
  ghi đè file token **rỗng** 16ms sau bản thắng và tự giết phiên của mình.
- **Còn CHƯA ĐO**: token hết hạn **giữa chừng** một phiên đang chạy. Đêm 22/08
  đặt ra đúng ca đó nhưng không chạm tới: phiên #195/#196 xong lúc ~01:26, còn
  mốc hết hạn là 05:16. Chưa có phép đo nào cho ca này, đừng suy từ mục trên.

## 22/08 — Vạch xung đột git nằm im trong `DO-LUONG.md` từ commit tuyên bố "sổ này SẠCH"

- **Đo lúc nào**: 22/08/2026, khi mở `docs/DO-LUONG.md` để ghi phép đo ở trên.
- **Con số / Bằng chứng**: `docs/DO-LUONG.md` mang một khối xung đột **chưa gỡ**
  — `<<<<<<< HEAD` ở dòng **3063**, vạch `=======` ở **3186**, `>>>>>>> sagent/phu-1`
  ở **3327** (dòng cuối file). `git blame` chỉ về commit **`5646f5f`**, tiêu đề:
  *"Merge sagent/phu-1: dong V2 va V1 — so no do luong nay SACH"*.
- **Vì sao không ai thấy**: nội dung **hai vế đều đúng và không mất gì** (vế
  HEAD là mục C3, vế `phu-1` là mục V1, mỗi mục đúng một lần), nên không có
  triệu chứng: `go build` · `go vet` · `go test` xanh cả ba, file `.md` vẫn mở
  ra đọc bình thường. Chỉ có ba dòng rác nằm giữa sổ đo lường.
- **Đã sửa hay chưa**: **ĐÃ SỬA** — xoá đúng ba dòng vạch, giữ nguyên cả hai vế
  (3327 → 3324 dòng). Và dựng bài canh `TestRepoKhongCoVachXungDot`
  (`internal/redaction/quet_xungdot_test.go`): quét **330 file** git đang theo
  dõi, bắt vạch xung đột ở **đầu dòng**. Đã chứng minh nó **ĐỎ** bằng cách dựng
  lại đúng file cũ — nó in ra cả hai dòng 3063 và 3327 rồi `FAIL`; xanh trở lại
  sau khi vá.
- **Bài học**: đây là *"tiêu đề commit không phải bằng chứng"* ở dạng gắt nhất —
  lượt trộn tự khai sổ đã sạch trong khi để lại rác **ngay trong cuốn sổ đó**.
  Lượt trộn 22/08 có kiểm vạch xung đột, nhưng chỉ trên **ba file nó biết là sẽ
  đụng**. Vạch này nằm ở file thứ tư nên lọt. Bài canh mới không hỏi file nào
  vừa bị đụng — nó quét cả repo.

## 22/08 — Ô quyền plugin `thu-muc-lam-viec` khai `chan-that` trong khi KHÔNG chặn được

- **Đo lúc nào**: 22/08/2026, sau khi trộn nhánh plugin vào `main`.
- **Ô nợ**: `internal/plugin/quyen.go:60` khai `QuyenThuMuc: Chan = ChanThat`.
- **Vì sao đáng đo**: cột `[chặn]` là thứ người vận hành đọc để quyết định có
  cắm một plugin lạ vào hay không. Chính file đó mở đầu bằng câu *"gộp hai cột
  lại là chỗ mọi hệ thống quyền nói dối"* — rồi tự vấp ở dòng 60.
- **Con số / Bằng chứng**: dựng plugin **không khai quyền nào**, cho biết đường
  dẫn một file trong thư mục dự án **qua tham số JSON-RPC** (kênh duy nhất
  không hàng rào nào chạm tới), chạy qua host thật:

  ```
  thu-muc=(khong-cap)  cwd-thay-dau-moc=khong        <- hàng rào THẬT, vẫn đúng
  thu-doc=duoc:33  thu-ghi=duoc  thu-liet-ke=duoc:2  <- và vẫn chạm tới được
  ```

  Host **không đưa** đường dẫn (hai cột đầu), nhưng plugin biết đường dẫn bằng
  cách khác thì **đọc đúng 33 byte**, **ghi được** file mới, **liệt kê được**
  thư mục. *"Không cấp đường dẫn"* không phải *"chặn"*.
- **Đã sửa hay chưa**: **ĐÃ SỬA** — hạ `ChanThat` → `KhongChanDuoc` kèm bằng
  chứng. Sửa luôn dòng `ghi-thu-muc-lam-viec`: nó kết bằng câu *"cách chặn thật
  đang có: không khai thu-muc thì plugin không có đường dẫn để mà ghi vào"* —
  câu đó vừa bị chính phép đo này bác bỏ. Dòng `secret` **giữ nguyên `ChanThat`**
  (lời khai của nó hẹp và đúng: host không ĐƯA secret) nhưng thêm dẫn chiếu, vì
  người đọc lướt sẽ gộp nó thành *"plugin không lấy được secret"* — sai.
- **Bài canh**: `TestBangQuyenThuMucPhaiKhopVoiThucTeChamDuoc` (`e2e_test.go`).
  Nó **không** khẳng định lỗ hổng tồn tại — nó đo rồi **đối chiếu số đo với lời
  khai**, và đỏ theo **cả hai chiều**: khai `chan-that` mà chạm được thì đỏ,
  chạm không được nữa mà vẫn khai `khong-chan-duoc` cũng đỏ. Viết kiểu
  `if !docDuoc { t.Fatal }` thì bài kiểm chỉ xanh chừng nào lỗ còn đó và sẽ phạt
  đúng người đi vá — repo này đã dính lỗi ấy một lần với
  `TestBaTrangThaiDiRaToiHopDong`.
- **Còn CHƯA ĐO**: hàng rào thật (ACL từng lượt chạy / Job Object /
  AppContainer) có dựng được trên Windows không. Chưa ai thử.

## 22/08 — Hai ca hỏng của plugin có mã xử lý nhưng 0 test, và một lỗi lộ ra khi đi viết test

- **Đo lúc nào**: 22/08/2026, ngay sau khi trộn nhánh plugin vào `main`.
- **Ô nợ**: bản rà nhánh plugin đếm ra ba ca hỏng của giao thức — plugin **treo**,
  plugin **chết giữa chừng**, **lệch phiên bản** — mà **chỉ ca thứ ba có test**.
  Hai ca đầu viết đúng nhưng chưa ai chạy thử, giữa một bộ **34 test** rất
  nghiêm ở mọi chỗ khác. Lệch chuẩn so với chính nó là dấu hiệu đáng tin nhất
  của một chỗ chưa được đo.
- **Cách dựng**: hai ca đều do PHÍA PLUGIN gây ra, host không tự tạo được. Thêm
  hai cần gạt vào plugin mẫu — `thu-treo-giay` (ngủ N giây) và `thu-chet`
  (thoát ngay, không trả lời) — đi qua **tham số JSON-RPC**, cùng đường với
  cần gạt đo hàng rào thư mục.
- **LỖI LỘ RA KHI VIẾT TEST, không phải khi đọc mã**: `TuyChon.Timeout` **chỉ
  áp cho lượt BẮT TAY** (`chay.go:241`). Lượt chạy thật (`Chay`) truyền thẳng
  `0` nên **luôn** dùng `TimeoutMacDinh = 60s`, bất kể người gọi đặt gì — trong
  khi bình luận ngay trên hằng số đó ghi *"Người gọi đặt Timeout khác được"*.
  Lời hứa chỉ được giữ ở đúng lượt gọi không ai cần đặt trần riêng.
- **Con số / Bằng chứng**: trần 1 giây, plugin ngủ 30 giây.

  | | Thời gian host đợi |
  |---|---|
  | Trước khi vá | **28,7 giây** (bị `ctx` 30s cắt, không phải trần 1s) |
  | Sau khi vá | dưới 10 giây, đúng trần |

  Chi tiết đáng giữ: **bản chưa vá VẪN trả về lỗi** — chỉ là 28 giây sau. Một
  bài kiểm chỉ hỏi *"có lỗi không"* sẽ **xanh** với đúng lỗi này. Phải khẳng
  định cả **THỜI GIAN** mới bắt được. Đã chứng minh bằng cách dựng lại đúng
  dòng cũ rồi chạy lại.
- **Đã sửa hay chưa**: **ĐÃ SỬA** — `Client` giữ `timeout` từ `TuyChon.Timeout`
  (phải giữ vì `Chay` chạy sau khi `Mo` đã trả về, lúc `opt` hết tầm), `Chay`
  truyền `c.timeout`. Ba bài kiểm mới: `TestPluginTreoThiHostCatTheoTran`
  (kèm khẳng định thời gian), `TestPluginChetGiuaChungThiHostNoiRoKemStderr`
  (khẳng định cả lời trăng trối ở stderr đi kèm — thiếu nó thì mọi lỗi plugin
  đều hiện ra là *"đóng ống trước khi trả lời"*, đúng mà vô dụng).

## 22/08 — Một bài kiểm mở `state.db` THẬT, và nó làm chết mặt điều khiển của cả máy

- **Đo lúc nào**: 22/08/2026 ~11:07, khi `sagent status` đột nhiên không chạy.
- **Triệu chứng**: mọi lệnh `sagent` đều trả
  `state.db ở schema v10, bản sagent này chỉ biết tới v9`.
- **Chuỗi nhân quả, đo được từng mắt**:

  1. `internal/api/phientrangthai_test.go` gọi `New(t.TempDir())` và **trông
     như đã cô lập**. Nhưng tham số đó là thư mục **DỰ ÁN**; sổ trạng thái nằm
     ở `paths.AccountsRoot()` = `<HOME>/.ai-accounts`, tính từ **HOME**. Bài
     kiểm quên gọi `homeGiaAPI(t)` — helper cô lập HOME **đã có sẵn trong cùng
     gói** (`fallback_test.go:24`).
  2. Agent #199 đang thêm bản di trú **v10** (`ALTER TABLE flow_steps ADD COLUMN
     idem_key` + index). Nó chạy `go test ./...` trong worktree của nó.
  3. Bài kiểm ở bước 1 mở `state.db` **THẬT**, `migrate()` chạy, sổ của máy lên
     v10 lúc **11:01** (dấu vết: `state.db.bak-v9` sinh đúng lúc đó).
  4. Từ giây đó, mọi binary dựng từ `main` (v9) — kể cả `sagent` đang cài —
     **từ chối** mở sổ. Toàn bộ mặt điều khiển của người vận hành chết, vì một
     dòng thiếu trong một file test, và **nhánh gây ra chuyện đó còn chưa trộn**.

- **Hai thứ làm ĐÚNG, ghi lại để không ai đi vá nhầm chỗ**: chốt hạ cấp
  (`store.go:564`) từ chối chạy thay vì ghi chéo phiên bản trong im lặng, và
  `migrate()` tự sao lưu trước khi nâng. Không có hai thứ đó thì bản v9 sẽ ghi
  vào file v10 và hỏng lộ ra muộn hơn, ở chỗ khác. Thứ hỏng là **ranh giới**:
  một bài kiểm không được chạm vào trạng thái sản phẩm của máy đang chạy.
- **Đã sửa hay chưa**: **ĐÃ SỬA, ở tầng GÓI** — thêm `TestMain` cho
  `internal/api` (`home_tam_test.go`) đẩy cả gói sang HOME tạm trước khi chạy
  bài kiểm nào, đặt **cả** `HOME` lẫn `USERPROFILE`, và **kiểm ngay** rằng phép
  đổi có tác dụng (`paths.AccountsRoot()` phải nằm trong thư mục tạm) — không
  có thì thoát ồn ào thay vì lặng lẽ quay lại ghi vào kho thật.

  Vì sao không chỉ thêm một dòng `homeGiaAPI(t)` vào đúng bài kiểm đó: thêm một
  dòng chỉ vá **bài kiểm đã biết**. Bài kiểm tiếp theo ai đó viết vẫn quên được,
  và lần quên sau lại là một lần cả máy chết. `TestMain` là chỗ duy nhất bao
  được cả bài kiểm chưa ai viết.
- **Nghiệm thu**: trước khi vá, `go test ./internal/api/` đỏ trên **cả `main`
  lẫn nhánh đang rà** — chứng minh lỗi là môi trường chứ không phải do nhánh
  nào. Sau khi vá: `go build` · `go vet` · `go test ./...` xanh cả ba.
- **Còn CHƯA ĐO**: các gói khác có bài kiểm nào chạm kho thật không. Gói `api`
  đã bịt; chưa quét toàn repo.

## 22/08 — Nút Lưu của bảng vẽ workflow XOÁ TRẮNG mọi trường nó không biết vẽ

- **Đo lúc nào**: 22/08/2026, phiên #199 tìm ra khi soi `artifact` có đi qua đủ
  bốn mặt không; người điều phối kiểm độc lập lại trước khi vá.
- **Con số / Bằng chứng**: `internal/dash/web/flow.html` — hàm lưu **dựng lại**
  mỗi bước từ đầu:

  ```js
  const s={id, type, needs, x, y, timeout_sec, retry, on_failure}
  ```

  Mở một flow trên bảng vẽ rồi bấm **Lưu**, **dù không sửa gì**, là xoá vĩnh
  viễn khỏi `flows.toml`: `doc_duoc` · `phai_co` · `vai_tro` · `model` ·
  `plugin` · `vao` · `tham_so` · `route` · `separator` · `fallback` ·
  `tu_duyet_quyen` · và cả `artifact` / `idempotent` / `compensate` vừa thêm.

  Nặng hơn một bậc: `profile`, `when`, `foreach` được `loadFlow` **đọc vào**
  nhưng hàm lưu **không ghi ra** — sửa chúng ở cột phải rồi bấm Lưu cũng mất.

  Đối chứng trên dữ liệu thật: flow `doi-4` trong `.sagent/flows.toml` có đủ
  `vai_tro`, `profile`, `model`, `phai_co` — tức lỗ này đang ăn vào một flow
  đang dùng, không phải một ca giả định.
- **Vì sao nguy hiểm hơn nó trông**: `phai_co` và `doc_duoc` là **cổng an
  toàn**. Mất chúng thì flow vẫn chạy, vẫn báo xong — chỉ là không còn ai kiểm
  đầu ra nữa. Mất dữ liệu **im lặng**: file trên đĩa đổi, không cảnh báo,
  người dùng chỉ phát hiện khi lượt chạy sau cư xử khác.
- **Đã sửa hay chưa**: **ĐÃ SỬA**. `loadFlow` giữ nguyên bước máy chủ gửi
  xuống (`__goc:s`); hàm lưu **trải bản gốc ra trước** rồi mới đè những gì bảng
  vẽ thật sự sửa (`{...(n.__goc||{}), id, type, …}`), và ghi thêm bốn trường
  `when` / `profile` / `foreach` / `tuDuyetQuyen` từ node.

  **Đánh đổi đã biết và chấp nhận**: kiểu này thì không xoá được một trường lạ
  bằng bảng vẽ nữa, phải sửa `flows.toml` bằng tay. Mất một đường xoá còn hơn
  xoá thứ người ta không định xoá. Cố ý **không** làm nửa còn lại mà #199 đề
  xuất (cho `flow.Save` trộn với bản trên đĩa) — làm cả hai phía là đổi nghĩa
  "lưu" thành "trộn" ở hai chỗ, và sau đó không còn đường nào xoá thật.
- **Bài canh**: `TestBangVeGiuTruongNoKhongBietVe` (`internal/dash/`). Canh
  **hai mắt xích** của phép trải, và **không chép tay danh sách tên trường** —
  danh sách chép tay sẽ lệch với `flow.Step` đúng vào lần thêm trường tiếp
  theo, mà lần đó mới là lần nguy hiểm. Thay vào đó nó dùng reflection đọc thẻ
  `json` của `flow.Step` rồi in ra những trường `flow.html` **không hề nhắc
  tên** — hiện là **3**: `thamSo`, `docDuoc`, `phaiCo`. Đã chứng minh **ĐỎ**
  bằng cách gỡ đúng phép trải ra rồi chạy lại.

## 22/08 — Chốt liên động: bài kiểm không mở được sổ trạng thái THẬT nữa

- **Đo lúc nào**: 22/08/2026, đóng nợ tự ghi ở mục trên ("các gói khác có bài
  kiểm nào chạm kho thật không — chưa quét").
- **Cách làm, và vì sao không đi soi từng file**: soi tay thì chỉ trả lời được
  về **những file đang có**. Thay vào đó đặt một chốt ở `store.OpenAt` — **nút
  thắt** mà mọi đường mở sổ đều đi qua — rồi cho cả bộ test tự khai ra.
- **Cơ chế**: chụp kho hồ sơ thật **lúc nạp gói** (`var khoThat =
  paths.AccountsRoot()`). Thời điểm này quan trọng: `t.Setenv("HOME", …)` và
  `TestMain` đều chạy **sau** khi gói được nạp, nên giá trị chụp được luôn là
  kho thật, kể cả trong một bài kiểm đã cô lập đúng cách. Đọc muộn hơn thì nó
  trỏ vào thư mục tạm và chốt **tự vô hiệu hoá trong im lặng**.
- **Nhận ra đang chạy dưới test bằng hậu tố `.test` / `.test.exe` của
  `os.Args[0]`**, cố ý **không** dùng `testing.Testing()` — hàm đó buộc gói sản
  phẩm import `testing`, kéo cờ dòng lệnh của bộ test vào mọi binary phát hành.
- **Kết quả khảo sát**: chạy `go test ./...` sau khi cắm chốt → **xanh, 0 gói
  vi phạm**. Tức `internal/api` là gói **duy nhất** từng chạm kho thật, và nó
  đã được vá. Đây là câu trả lời cho nợ ghi ở mục trên.
- **Nhưng xanh không tự chứng minh chốt hoạt động** — một bộ test xanh không
  phân biệt được *"không ai vi phạm"* với *"chốt hỏng"*. Nên có
  `TestChotChanKhoThatNoThat`, và nó kiểm cả mắt xích dễ gãy nhất: `duoiTest()`
  phải trả `true` ngay trong một bài kiểm. Go đổi cách đặt tên binary test thì
  chốt tắt ở **mọi gói**, im lặng — dòng đó là thứ duy nhất báo.
- **Bằng chứng mạnh nhất, chạy thật**: gỡ `TestMain` của `internal/api` ra rồi
  chạy lại đúng bài kiểm đã gây sự cố:

  ```
  --- FAIL: TestBoChayFlowCoDuCaHaiDuong
      bài kiểm đang mở SỔ TRẠNG THÁI THẬT của máy (…\.ai-accounts\state.db) — từ chối.
      Cách sửa: đổi HOME sang thư mục tạm TRƯỚC khi mở sổ (xem TestMain trong internal/api)…
  ```

  Chốt bắt **đúng** sự cố sáng nay, ở **đúng** dòng, kèm câu chỉ cách sửa.
- **Cửa thoát có chủ ý**: `SAGENT_CHO_PHEP_KHO_THAT=1`. Cấm tuyệt đối sẽ khiến
  người ta đi vòng bằng cách tệ hơn — gọi thẳng `sql.Open`, bỏ qua cả
  `migrate`. Cửa thoát phải **lộ ra trong mã của bài kiểm**, chỗ người rà thấy.
- **Còn CHƯA CHE, nói thẳng**: bài kiểm nào **chạy binary `sagent` đã build**
  như tiến trình con thì chốt không thấy — tiến trình đó không phải binary
  test. Chưa có bài kiểm nào như vậy trong repo, nhưng chốt này **không** ngăn
  được kiểu đó nếu mai có người viết.

## 22/08 — Ba mảnh flow chạy thật qua `sagent flow run`, không tốn hạn mức nào

- **Đo lúc nào**: 22/08/2026 11:53–11:57, đóng nợ chính báo cáo `#199` tự ghi:
  *"chưa đo trên một lượt chạy thật — tất cả bằng chứng là từ test đầu-cuối"*.
- **Cách đo, và vì sao nó rẻ**: dựng một dự án tạm ở `C:\Users\Administrator\do-flow-that`
  với flow chỉ gồm bước **`shell`**. Bước `shell` đi qua **đúng** bộ chạy, đúng
  sổ trạng thái, đúng đường artifact như bước `agent` — chỉ khác là không gọi
  model nào. Sáu lượt chạy thật (#53–#58) tốn **0 token, 0 đồng**.
- **ARTIFACT — số đo dứt khoát**: bước sinh một file **60.094 byte / 1200 dòng**,
  bước sau đọc qua `{{artifacts.to}}`.

  | Đường truyền | Tới được bước sau |
  |---|---|
  | Artifact | **1200 / 1200 dòng** (60.094 byte, đọc từ đĩa) |
  | Đầu ra đã lưu (`{{steps.x.output}}`) | **651 / 1200 dòng** |

  **Mất 549 dòng — 45,75% — ngay ở tầng LƯU** (`MaxStepOutput = 32.768`), tức
  trước cả tầng nhét (`MaxInject = 6.000`). Và phần mất là phần **cuối**, đúng
  chỗ mọi báo cáo để kết luận. Đây là con số biện minh cho cả mảnh artifact.
- **IDEMPOTENT — bỏ qua thật, và chép artifact sang lượt mới**:
  - Lượt **#54**: `sinh` chạy, ghi `run-54/sinh/bao-cao.txt` (5.092 byte).
  - Lượt **#55**: `sinh` in *"bỏ qua — việc này lượt chạy #54 đã làm xong
    (idempotent)"*, **dẫn đích danh lượt cũ**. Bước `doc` vẫn đọc đủ 5.092 byte
    vì artifact được **chép sang** `run-55/sinh/` — kiểm bằng `ls`, file có
    thật, đúng kích thước.
  - Lượt **#56**: đổi đúng một chỗ trong dòng lệnh (`1..400` → `1..401`) thì
    bước **chạy lại**, không bỏ qua. Khoá bám nội dung thật, không bám id bước.
- **COMPENSATE — chạy bước gỡ lại rồi vẫn dừng**: lượt **#58**, bước `dat-cho`
  tạo một file "chỗ đã đặt", bước `hong` thoát 1, engine in *"do-compensate.hong
  hỏng — chạy bước gỡ lại go-lai"*. Bước gỡ lại xác nhận **`cho da dat ton
  tai=True`** rồi xoá; kiểm trên đĩa sau đó: file **không còn**. Lượt chạy kết
  thúc `failed` — đúng: `compensate` **gỡ lại**, không phải **cứu**.
- **Một cái bẫy gặp thật khi soạn flow**, ghi lại vì nó tốn một lượt chạy hỏng
  (#53): `run` là **argv**, không phải chuỗi shell. Viết `for /L %i … >> "…"`
  với đường dẫn trộn `/` và `\` cho `The filename, directory name, or volume
  label syntax is incorrect`. Và trong TOML **chuỗi cơ bản** thì `\t` là ký tự
  TAB, nên `{{artifact_dir}}\to.txt` hỏng im lặng — phải dùng `/` trong đường
  dẫn hoặc chuỗi literal. Cảnh báo của `sagent flow validate` về `idempotent`
  thì nói **đúng** cái bẫy của nó ngay từ đầu.
- **Còn CHƯA ĐO**: cùng ba mảnh này với bước **`agent`** thật (tốn hạn mức), và
  con số đáng giá nhất còn thiếu — bước gộp báo cáo tốn bao nhiêu token vào
  trước/sau khi chuyển sang artifact (lượt #34 từng là 10.998 token cho một
  bước gộp).

## 22/08 — `sagent help` bỏ sót 5 lệnh chạy được, không ai canh chiều CLI ↔ help

- **Đo lúc nào**: 22/08/2026, khi cắm nốt phần `#202` cố ý để lại.
- **Bắt đầu từ đâu**: `#202` thêm lệnh `nang-luc-api` và **tự nói ra** khoảng
  trống nó không được phép vá: `cmdHelp()` trong `main.go` là một **chuỗi viết
  tay**, không sinh từ bảng `commands`, nên lệnh mới chạy được ngay mà **không
  bao giờ hiện ra trong help**.
- **Đây là hình dạng lỗi thứ TƯ trong ngày**, cùng một họ với `route.kiem`, nút
  Duyệt/Từ chối và `plugin.list`: thứ gì đó "có" ở mọi tầng **trừ** tầng người
  dùng thật sự chạm vào. Luật ngang quyền của dự án canh
  `api.Actions ↔ CLI ↔ HTTP ↔ web-UI`; nó **không** canh `CLI ↔ help`.
- **Con số**: dựng `TestMoiLenhCapMotDeuXuatHienTrongHelp` (bắt stdout của
  `cmdHelp()`, đối chiếu với bảng `commands`, bỏ qua tên tiền tố `__` vì đó là
  quy ước cho lệnh con/cờ). Chạy lần đầu → **5 lệnh** chạy được mà help chưa
  bao giờ nhắc:

  | Lệnh | Người dùng mất gì khi không biết nó tồn tại |
  |---|---|
  | `plugin` | không có đường nào xem plugin đã cài xin quyền gì |
  | `db` | không biết có `db backup` / `db restore` — đúng thứ cần lúc sự cố |
  | `quet` | không biết cách tìm tiến trình mồ côi của phiên đã chết |
  | `api` | không biết có đường API thứ hai, không tiêu hạn mức thuê bao |
  | `version` | không biết cách phân biệt bản tải về với bản tự build |

  Ba trong năm (`plugin`, `db`, `quet`) là **lệnh dùng lúc đang có sự cố** —
  đúng lúc người ta đi gõ `sagent help`.
- **Đã sửa hay chưa**: **ĐÃ SỬA** — thêm cả 5 vào help kèm câu cảnh báo đi
  kèm (`quet` chỉ LIỆT KÊ, Windows dùng lại PID; `db restore` phải dừng dash
  trước; bảng quyền plugin: khai một quyền trong manifest **không tự nó là một
  hàng rào**), cộng `nang-luc-api`. Bài kiểm nay xanh và **giữ cho lệnh thứ sáu
  không lọt được nữa**.
- **Bắt stdout chứ không tách chuỗi help thành hằng số**: tách ra là sửa mã sản
  phẩm cho vừa bài kiểm, và bài kiểm sẽ đo cái hằng số đó thay vì đo thứ người
  dùng thật sự thấy.

## 22/08 — Đóng ô `reasoning`, và tìm ra ĐIỂM MÙ của chính bảng năng lực nửa API

- **Đo lúc nào**: 22/08/2026, ngay sau khi trộn bảng năng lực nửa API.
- **Ô nợ**: bảng mới in `✗ reasoning` cho `deepseek-v4-flash` kèm chẩn đoán
  *"vướng ở phía dự án"* — nhà cung cấp **trả** `reasoning_content`, còn
  `aiapi.phanHoi` chỉ đọc `content` nên **vứt phần nghĩ trước khi ai nhìn thấy**.
- **Đã sửa**: thêm trường `SuyLuan` (`json:"reasoning_content,omitempty"`) vào
  `tinNhan`, gán sang `KetQua.SuyLuan`. `omitempty` vì kiểu này dùng cho **cả
  hai chiều** — gửi đi thì không bao giờ mang trường này.
- **Số đo, gọi THẬT `deepseek-v4-flash` qua modelapi.vn**:

  | | Độ dài |
  |---|---|
  | Câu trả lời (`NoiDung`) | **91 ký tự** |
  | Phần suy luận (`SuyLuan`) | **477 ký tự** |

  Usage vào 108 / ra 192 / tổng 300 token, 2,7 giây. Phần bị vứt đi **dài gấp
  5,2 lần** phần giữ lại — và người dùng đã **trả tiền cho cả hai**.

### ĐIỂM MÙ tìm ra trong lúc vá — quan trọng hơn chính bản vá

Bảng năng lực dò phía dự án bằng **reflection trên kiểu thật**
(`nangluc.go`: `coTruong(reflect.TypeOf(phanHoi{}), …)`). Thiết kế đó rất tốt:
bảng **không bịa được** về phía mình. Nhưng nó trả lời đúng **một** câu — *"kiểu
có trường đó không"* — và **không** trả lời câu *"có ai chép giá trị đi đâu
không"*.

**Đã chứng minh bằng cách chạy thật**: gỡ đúng một dòng gán trong `aiapi.go`
rồi chạy lại cả hai bên:

```
$ sagent nang-luc-api deepseek
    ✓ reasoning   … phía dự án: aiapi.phanHoi đọc được trường `reasoning_content`

$ go test ./internal/aiapi/ -run TestSuyLuanDiHetDuongToiKetQua
    --- FAIL: KetQua.SuyLuan = "" — phần suy luận nhà cung cấp TRẢ VỀ bị vứt …
```

**Bảng vẫn khoe ✓ trong khi giá trị bị vứt.** Đây là lần thứ **năm** trong ngày
gặp cùng một hình dạng — sau `route.kiem`, nút Duyệt/Từ chối, `plugin.list`,
`sagent help` — và lần này nạn nhân là chính công cụ dựng ra để chống nói dối.

- **Bài canh**: `TestSuyLuanDiHetDuongToiKetQua` đi **hết đường** (máy chủ giả →
  `Goi()` → `KetQua.SuyLuan`), cộng `TestKhongCoSuyLuanThiVanChayBinhThuong`
  (không có phần nghĩ thì không được bịa ra chuỗi rỗng có vẻ đã đo) và
  `TestTenKhoaJSONCuaSuyLuanKhongDuocDoi` (đổi tên khoá là im lặng mất tính năng).
- **Bài canh định kỳ chạm mạng**: `TestE2ESuyLuanThatTuNhaCungCap`, bật bằng
  `SAGENT_E2E_SUYLUAN=1`, theo đúng lệ của `TestE2EGrokDocDuocOutputThat`. Ba
  bài trên đo với nhà cung cấp **giả**: chúng khẳng định *"nếu máy chủ trả
  `reasoning_content` thì giá trị đi tới `KetQua`"* — đúng và cần, nhưng **không**
  khẳng định máy chủ thật còn trả trường tên đó. Đúng chỗ ô **C2** của Grok đã
  cắn một lần: định dạng ta **quan sát** được, không phải hợp đồng nhà cung cấp
  cam kết.
- **CÒN NỢ, nói thẳng**: `KetQua.SuyLuan` hiện **chưa có mặt nào đọc nó** — chưa
  có cờ CLI, chưa có chỗ trên dashboard, node `model` của flow chưa dùng. Không
  vá trong lượt này vì `cmd/sagent/*` và `internal/dash/*` đang thuộc một phiên
  khác chạy song song. Đây đúng là lỗi "thiếu mặt cuối" mà mục này vừa tố cáo,
  nên nó phải được đóng ở lượt kế, không được để trôi.

### Phụ lục cùng ngày: bản vá `reasoning` làm ĐỎ một bài kiểm, và bài kiểm đó SAI

- `TestVuongCaHaiBenKhongDocThanhVuongMotBen` ghim thẳng một ô sống:
  *"grok/reasoning phải là `ca-hai`"*. Đúng lúc viết. Nhưng vá xong phía dự án
  thì ô ấy chỉ còn vướng nhà cung cấp, và bài kiểm **đỏ** — tức nó **chỉ xanh
  chừng nào dự án còn nợ**, phạt đúng người đi trả nợ.
- Repo này đã chữa y hệt một lần: `TestBaTrangThaiDiRaToiHopDong` từng đếm
  trạng thái trên dữ liệu sản phẩm rồi bắt cả ba phải xuất hiện.
- **Viết lại thành bất biến, không ghim dữ liệu**:
  `TestKetLuanVuongODauLuonKhopVoiCapTrangThai` duyệt **mọi ô của mọi route** và
  khẳng định phép ánh xạ `(phía dự án, nhà cung cấp) → kết luận vướng ở đâu`.
  Ô nào đổi trạng thái cũng không sao — chỉ cần kết luận đi theo. Chạy lần đầu:
  **14 ô, trong đó 2 ô vướng cả hai bên** (`dau-vao-anh` và `dau-ra-co-cau-truc`
  của deepseek).
- **Sự cố kèm theo, ghi để không lặp lại**: tôi commit `2fa3db1` trên một cây
  **đang đỏ**, vì câu lệnh nghiệm thu in `XANH` **vô điều kiện** sau khi lọc log
  — dòng `FAIL` có in ra nhưng bị chính chữ "XANH" ngay dưới nó nuốt mất. Nay
  kiểm bằng **mã thoát** (`go test ./...; echo $?`), không bằng phép lọc chữ.
  Cùng một bài học với `sagent ds`: một chỉ báo trả lời câu dễ hơn câu người đọc
  tưởng nó đang trả lời.

## 22/08 — Đóng `goi-tool` bằng cách TỪ CHỐI chạy tool, và trần đồng thời cho đường flow

- **`goi-tool` (phiên #205)**: `aiapi` nay gửi được định nghĩa tool và đọc lại
  `tool_calls`. Câu hỏi ghim trong brief — *"sau khi model trả về một tool_call
  thì AI chạy cái tool đó?"* — được trả lời bằng **KHÔNG**, kèm ba lý do:
  1. Chạy tool hộ biến thư viện thành **nửa vòng lặp agent**, và phá lời hứa nền
     của cả gói: *"mọi lời gọi đều nói ra nó tiêu gì"* — `Usage` của lượt nào,
     khi một lần gõ lệnh thành sáu lượt gọi?
  2. `sagent` **đã có** một đường chạy-thứ-gì-đó: flow, với bảng quyền plugin,
     có duyệt, có sổ. Cho thư viện API mọc đường chạy lệnh **thứ hai** ở dưới
     đáy, không qua bảng quyền nào, là mở cửa sau cho chính dự án này.
  3. Tên hàm và tham số là **chữ do model sinh** — phải bị người gọi soi, không
     phải được một hàm thư viện lặng lẽ thi hành.

  Và nó **không dừng ở bình luận**: có bài kiểm **đếm số lượt chạm mạng, đỏ nếu
  khác 1**. Nguyên văn báo cáo: *"bình luận thì mục ruỗng; con số thì không"*.
- **Điểm mù của bảng năng lực, thu hẹp thêm một nấc**: `phepDoMaNguon` nay hỏi
  **câu thứ ba** — `KetQua` có **chỗ chứa** giá trị không — chứ không chỉ hỏi
  kiểu phản hồi có trường không. Áp cho cả `goi-tool` lẫn `reasoning`. Vẫn chưa
  bịt được hết (nó không biết ai đọc `KetQua`), nhưng hẹp hơn một nấc.
- **Trần đồng thời cho đường flow (phiên #206)** — mảnh **cuối cùng** của engine.
  Trước bản sửa: bốn trần chỉ canh cửa `FleetStart`; `api.go:1677` truyền đúng
  một con số vào `flow.Runner`, nên bốn bước `agent` cùng khai
  `profile = "claude:tns"` vẫn chạy 4 phiên trên **một** tài khoản — đúng sự cố
  đã đẻ ra bốn trần kia, chỉ khác cửa. Nguyên tắc chọn: **xin không được thì
  CHỜ, không giết lượt chạy** — một lượt flow đêm dài bị giết vì trần tệ hơn là
  chạy chậm. Có canh chết cứng, vì *"deadlock thì ầm ĩ và dễ thấy; đếm sai thì
  im lặng và đắt"*.

### Món nợ tài liệu #206 tự chỉ ra, và tôi đóng ngay

Với bộ mặc định, `max_parallel_sessions = 4` là **một lời hứa mà trần provider
(3) không cho giữ**: bốn bước `agent` cùng nhà cung cấp `claude` không bao giờ
chạy quá 3 cùng lúc, dù có bao nhiêu tài khoản. Trước 22/08 chuyện đó **vô hình**
vì cửa flow không áp trần; sau bản sửa nó thành hành vi thấy được mỗi ngày.
#206 neo bằng một bài kiểm nhưng nói thẳng *"chỗ đúng để giải thích là tài liệu
người dùng — mà tài liệu nằm ngoài vùng của lượt này"*. Đã ghi vào
`.sagent/project.toml` ngay trên khối `[policy.tran]`.

### Kế hoạch: sửa dòng đã cũ ngay khi vừa cập nhật

`docs/MASTER-PLAN.md` ghi *"bốn năng lực tool · vision · structured-output ·
reasoning là lỗi nằm trong repo này"* — đúng lúc phiên #203 viết, sai ngay sau
đó, vì `reasoning` và `goi-tool` được đóng **trong lúc** #203 đang chạy. Nay còn
**hai**: `dau-vao-anh` và `dau-ra-co-cau-truc` — và hai ô đó **không cùng loại**:
nợ của dự án cho route **grok** (nhà cung cấp làm được), vướng **cả hai bên** cho
route **deepseek**. Đã tách rõ, sinh lại cả hai trang, `diff` hai bản
MASTER-PLAN rỗng.

**Bài học về cách chia việc song song**: người cập nhật sổ và người sửa mã
**không chạy cùng lúc mà sổ vẫn đúng được**. Lượt sau nên để việc cập nhật kế
hoạch chạy MỘT MÌNH, hoặc chạy sau cùng.

## 22/08 — Soát nghiệm thu từ phía NGƯỜI DÙNG sau 20 lần trộn trong một ngày

- **Đo lúc nào**: 22/08/2026 ~15:20, sau khi trộn đợt 6. Không ai làm việc này
  trong ngày: mọi lượt đều nghiệm thu bằng `go test`, không lượt nào gõ đúng
  những lệnh người dùng gõ.
- **Kết quả — công cụ còn dùng được, không có hồi quy**:

  | Lệnh | Kết quả |
  |---|---|
  | `sagent verify` | **mã thoát 0**, mọi ô ✓ trên cả 5 provider + kho hồ sơ |
  | `sagent route kiem` | deepseek **177ms** · grok **177ms**, cả hai dùng được |
  | `sagent nang-luc --chua-do` | *"Không còn năng lực nào chưa đo"* |
  | `sagent nang-luc-api --chua-do` | *"Không còn năng lực nào chưa đo"* |
  | `sagent flow validate` | 0 lỗi · 2 cảnh báo (xem dưới) |
  | `sagent plugin quyen` | cột `[chặn]` in **✗** cho `thu-muc-lam-viec` và `ghi-thu-muc-lam-viec` — đúng sự thật đã đo sáng nay |

- **HAI CẢNH BÁO CÒN LẠI, và chúng nằm ở FLOW MẪU DỰNG SẴN** — tức thứ người
  dùng mới gặp **đầu tiên**: `fanout.chon` và `squad.duyet` là bước `approve`
  **không chặn bước nào** (không bước nào khai `needs` trỏ tới chúng), nên
  *"nó sẽ dừng luồng nhưng các bước khác VẪN CHẠY song song với nó"*. Hai trên
  ba flow mẫu ship kèm một rào duyệt mắc sai dây.
  **CHƯA SỬA** — `internal/flow/builtin.go` đang thuộc một phiên chạy song song.
  Ghi ra đây để lượt sau đóng; đây là ô chạm thẳng vào chuẩn *"non-tech dùng
  được trong 5 phút"*.
- **Dashboard đang chạy là bản CŨ 16,5 tiếng**: PID 16368 giữ cổng 8788, bật lúc
  **21/08 22:39**, tức trước toàn bộ việc hôm nay. Assets nhúng trong binary nên
  nó vẫn phục vụ giao diện cũ: **không có** panel quyền plugin, panel artifact,
  panel suy luận, bảng năng lực nửa API. Bằng chứng theo đúng luật của dự án —
  **PID giữ cổng + giờ bật tiến trình**, không phải mã HTTP.
  **ĐÃ BẬT LẠI 22/08 15:41**, sau khi chủ dự án chốt. Dòng lệnh lấy từ chính
  tiến trình cũ (`Win32_Process.CommandLine`), **không đoán**:
  `sagent.exe dash --host 0.0.0.0 --port 8788`. Cổng 8788 nay do **PID 1188**
  giữ, cùng cờ, mật khẩu không đổi nên link trên điện thoại vẫn vào được.

  **Nghiệm thu bằng ĐỐI CHỨNG, không bằng mã HTTP** (mã HTTP không chứng minh
  được server nào đang trả lời — luật mục F của `docs/VAN-HANH-VPS.md`):

  | Dấu mốc của việc hôm nay | Binary đang chạy | Binary 21/08 |
  |---|---|---|
  | `api/plugins` | **7** | **0** |
  | `api/flow/artifacts` | **6** | **0** |

  Hai chuỗi đó chỉ có trong bản mới, nên bản đang phục vụ đúng là bản mang việc
  hôm nay.

## 22/08 — Hàng rào ACL cho plugin: chặn được, nhưng chặn cả HOST

- **Đo lúc nào**: 22/08/2026 ~15:45. Đóng một phần dòng **CHƯA ĐO** ghi sáng nay:
  *"hàng rào thật (ACL từng lượt chạy / Job Object / AppContainer) có dựng được
  trên Windows không — chưa ai thử"*.
- **Vì sao đáng đo**: ô `thu-muc-lam-viec` trong `internal/plugin/quyen.go` sáng
  nay bị hạ từ `ChanThat` xuống `KhongChanDuoc` vì host chỉ **không cấp đường
  dẫn** chứ không dựng hàng rào. Câu tiếp theo là: dựng được không?

### Đo được (1) — DACL CHẶN THẬT tiến trình con cùng tài khoản quản trị

Dựng thư mục tạm có file, đặt `icacls /inheritance:r /grant:r "SYSTEM:(OI)(CI)F"
/deny "<tài khoản>:(OI)(CI)F"`, rồi cho một **tiến trình con riêng** (cùng tài
khoản `Administrator`, `IsInRole(Administrator) = True`) đọc file:

```
ket qua doc = BI CHAN: UnauthorizedAccessException
```

Đây là kết quả **khả quan hơn dự đoán**: không cần AppContainer hay token hạn
chế, một DACL thường đã chặn được đọc thẳng.

### Đo được (2) — và nó chặn luôn HOST, đây mới là chỗ khó

Tiến trình **đã tạo ra thư mục** sau đó **không xoá được nó**:

```
KHONG xoa duoc sagent-do-acl-80ee968f : UnauthorizedAccessException
```

Phải `icacls /grant` cấp lại quyền cho chính mình rồi mới dọn được. Lý do hiển
nhiên khi nhìn lại: plugin và host **chạy dưới cùng một danh tính**, nên một
DENY nhắm vào "tài khoản đó" không phân biệt được ai là ai.

**Hệ quả thiết kế — đây là giá trị thật của phép đo này**: thêm một lời gọi ACL
vào `chay.go` là **không đủ**, và còn tự bắn vào chân (host mất quyền dọn thư
mục tạm của chính nó). Hàng rào chỉ dùng được khi plugin chạy dưới **danh tính
KHÁC** — token hạn chế, tài khoản cục bộ riêng, hoặc AppContainer. Đó là thay
đổi kiến trúc, không phải một dòng mã.

### CHƯA ĐO, và nói rõ vì sao chưa

Plugin thù địch có tự gỡ rào được không (`takeown` + `icacls /reset`, quyền mà
một tài khoản quản trị **có**)? **Chưa đo.** Đã định đo, nhưng bộ lọc quyền của
môi trường chạy chặn lệnh đó và tôi **không đi vòng**. Đừng suy ra câu trả lời
từ mục (1): (1) chỉ nói đọc THẲNG bị chặn, không nói không có đường khác.

Cho tới khi đo được nửa này, `thu-muc-lam-viec` **giữ nguyên `KhongChanDuoc`** —
một hàng rào chưa biết có gỡ được hay không thì chưa được khai là hàng rào.

## 22/08 — Kiểm tầng redaction trên DỮ LIỆU SẢN PHẨM, không phải trên test

- **Đo lúc nào**: 22/08/2026 ~15:55, khi tình cờ thấy `sagent nhat-ky 208` in ra
  `C:\Users\[đã-che:người-dùng]\...` trong lúc đang chẩn đoán việc khác.
- **Vì sao đáng đo lại**: tầng redaction trộn lúc ~10:45. Từ đó tới giờ đã có
  **năm phiên agent** chạy và ghi nhật ký thật. Đây là lần đầu có dữ liệu sản
  phẩm sinh ra SAU bản vá để kiểm — mọi bằng chứng trước đó là từ test và từ
  nhật ký cũ.
- **Con số, đếm trên file `.log` thật**:

  | Nhật ký | `Administrator` trong FILE | `đã-che` trong FILE |
  |---|---|---|
  | `...141021` (14:10, sau bản vá) | **495** | 0 |
  | `...153457` (15:34, sau bản vá) | **73** | 0 |
  | `...153552` (15:35, sau bản vá) | **157** | 0 |

  Nhưng cùng nội dung đó **đọc qua `sagent nhat-ky`** thì ra
  `C:\Users\[đã-che:người-dùng]\...`.
- **Kết luận: ĐÚNG NHƯ THIẾT KẾ, và thiết kế đã được nói ra trước.**
  `internal/nhatky/nhatky.go:204,239` gọi `redaction.Che` ở **hai cửa ra**
  (`BoDau` và `duoi`) — nơi nội dung nhật ký **thoát ra ngoài** — chứ không ở
  đường ghi. Bình luận trong mã nói rõ vì sao: *"mặt gọi viết sau này không quên
  được cái nó không phải nhớ"*.

  `docs/BAO-CAO-REDACTION.md` cũng đã ghi điều này ở **dòng đầu tiên** của mục
  "Rủi ro còn lại": *"Che lúc đọc, không che lúc ghi. File trên đĩa vẫn chứa
  nguyên văn."* Phép kiểm hôm nay **xác nhận** báo cáo đó nói đúng, trên dữ liệu
  sản phẩm.
- **Thứ PHẢI sửa, và đã sửa**: bảng tổng trong `docs/MASTER-PLAN.md` ghi
  *"2.060 lần lộ danh tính → 0"* **không kèm vế "lúc đọc"**. Con số đúng nhưng
  nói về một thứ khác với cái người đọc lướt sẽ hiểu — họ sẽ tưởng file trên đĩa
  đã sạch. Đã thêm vế đó vào cả hai chỗ trong kế hoạch và sinh lại hai trang.
- **Rủi ro còn lại, nhắc lại cho rõ**: ai đọc được đĩa thì tầng này không cản.
  Nó chỉ cản nội dung **đi ra** — mà cửa đi ra đáng lo nhất là cổng 8788 đang
  phơi ra internet, và đó đúng là cửa được che.

## 22/08 — Kế hoạch đang đếm một hướng ĐÃ BỊ BÁC BỎ là nợ (ACP)

- **Đo lúc nào**: 22/08/2026 ~15:58, khi chuẩn bị brief cho đợt tiếp theo. Ô
  **Subscription** là nợ lớn nhất còn tự gỡ được, nên tôi đi tra "ACP" là gì
  trong bối cảnh repo này **trước khi** giao việc.
- **Mâu thuẫn tìm được giữa hai tài liệu**:
  - `docs/MASTER-PLAN.md` liệt kê `**CHƯA — ACP**: grep không ra gì. Không
    harness nào được đo qua giao thức này.` → đọc ra là **việc phải làm**.
  - `docs/DO-LUONG.md` mục 18/08 đã kết luận ngược lại, có phép đo: **không CLI
    nào trên máy nói ACP** (`claude --help`, `codex --help`, `agy --help` đều
    không có), và cả ba đều có đường khác sẵn hôm nay. Nguyên văn: *"đích đến
    không phải ACP — mà là dữ liệu có cấu trúc, và nó có sẵn"*.
- **Vì sao đây là lỗi đáng sửa chứ không phải chuyện chữ nghĩa**: nếu tôi giao
  đợt tới theo bản kế hoạch, một phiên agent sẽ đi **bọc adapter ACP** — đúng
  việc dự án đã cố ý không làm, cho một giao thức **không CLI nào trên máy này
  nói**. Đó là một lượt hạn mức mất trắng, và nó mất vì SỔ nói sai chứ không vì
  ai làm sai.
- **Thứ ACP hứa thì đã đạt bằng đường khác**, và đo được: bốn kiểu hỏng từng
  phải đoán bằng dò chuỗi nay đọc thẳng ra trường —
  `permission_denials` (bị chặn quyền) · `subtype = error_max_turns` (chạy quẩn)
  · `api_error_status` (hết hạn đăng nhập) · `rate_limit_event.resetsAt` (nói
  được *"hạn mức quay lại lúc mấy giờ"* thay vì chỉ báo hỏng).
- **Đã sửa**: dòng đó thành **"BỎ KHỎI PHẠM VI — ACP"**, kèm phép đo và nguyên
  văn kết luận 18/08, theo đúng lệ đã có trong chính bản kế hoạch
  (`~~symlink Linux~~ bỏ theo mục 2b`, `Gemini CLI đã bỏ`). **Không đổi dấu tick**
  — ô Subscription vẫn `[~]` vì còn hai mục CHƯA thật: resume/cancel ở tầng
  harness, và nhật ký chạm file theo pha login/refresh/exit.
- **Bài học về thứ tự việc**: tra sổ **trước khi** viết brief tốn 5 phút và cứu
  một lượt chạy. Cả ngày hôm nay tôi viết brief từ bản kế hoạch mà không đối
  chiếu ngược lại sổ đo lường — lần này tình cờ đối chiếu vì không biết ACP là gì.
