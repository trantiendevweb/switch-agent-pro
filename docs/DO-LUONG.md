# Pha 0 — Báo cáo đo giả định

> "Đã đo — không suy luận." Mỗi ô ghi **đo thế nào** và **kết quả**. Chưa đo xong
> thì provider/OS đó còn nhãn `experimental`, công cụ cảnh báo khi dùng.
>
> Cập nhật: 2026-08-17.

## Bảng trạng thái

| Provider | Windows | Linux | Nhãn |
|---|---|---|---|
| Claude | ✅ đo (v1) | ⬜ chưa đo | Windows `stable`, Linux `experimental` |
| Codex  | ⬜ chưa đo | ⬜ chưa đo | `experimental` |
| Gemini | ⬜ chưa đo | ⬜ chưa đo | `experimental` |
| Cursor | ⬜ chưa đo | ⬜ chưa đo | `unsupported` cho tới khi đo |

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
- [ ] ⚠ **Concurrent refresh — CHƯA ĐO.** Token bị chép ra N chỗ; khi hết hạn,
  N tiến trình có thể cùng gọi refresh. Chưa biết nhà cung cấp xử lý thế nào
  (có thể thu hồi refresh token cũ → các phiên khác văng). `fleet` **in cảnh báo
  này mỗi lần chạy** thay vì hứa an toàn. Cần một phiên chạy đủ dài qua mốc hết
  hạn token để đo.

---

## Việc cần bạn hỗ trợ

- **Máy/VM Linux** để chạy các ô Linux ở trên. Không có thì phần Linux của
  Pha 0/1 phải hoãn, và nhãn Linux giữ nguyên `experimental`.
