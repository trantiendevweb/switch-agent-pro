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

---

## Việc cần bạn hỗ trợ

- **Máy/VM Linux** để chạy các ô Linux ở trên. Không có thì phần Linux của
  Pha 0/1 phải hoãn, và nhãn Linux giữ nguyên `experimental`.
