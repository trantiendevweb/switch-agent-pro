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

---

## Việc cần bạn hỗ trợ

- **Máy/VM Linux** để chạy các ô Linux ở trên. Không có thì phần Linux của
  Pha 0/1 phải hoãn, và nhãn Linux giữ nguyên `experimental`.
