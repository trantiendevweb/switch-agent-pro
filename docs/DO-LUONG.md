# Pha 0 — Báo cáo đo giả định

> "Đã đo — không suy luận." Mỗi ô ghi **đo thế nào** và **kết quả**. Chưa đo xong
> thì provider/OS đó còn nhãn `experimental`, công cụ cảnh báo khi dùng.
>
> Cập nhật: 2026-08-17.

## Bảng trạng thái

| Provider | Windows | Linux | Nhãn |
|---|---|---|---|
| Claude | ✅ đo | ⬜ chưa đo | Windows `stable`, Linux `experimental` |
| Codex  | ✅ đo | ⬜ chưa đo | Windows `stable`, Linux `experimental` |
| Gemini | ⬜ chưa đo | ⬜ chưa đo | `unknown` — CLI chưa có trên máy đo |
| Cursor | ⬜ chưa đo | ⬜ chưa đo | `unknown` — CLI chưa có trên máy đo |

> Đã kiểm ngày 2026-08-17: máy đo chỉ có `claude` và `codex`; `gemini`, `cursor`,
> `opencode`, `aider` đều không cài. Không đo được thì để `unknown`, không đoán.

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

---

## Việc cần bạn hỗ trợ

- **Máy/VM Linux** để chạy các ô Linux ở trên. Không có thì phần Linux của
  Pha 0/1 phải hoãn, và nhãn Linux giữ nguyên `experimental`.
