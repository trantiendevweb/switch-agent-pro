# Báo cáo — đóng hai câu cuối của ô `Subscription`

Ngày đo: **22/08/2026**. Nhánh: `sagent/subscription-22-08` (tách từ `main`, 0 commit lệch lúc bắt đầu).
Máy: Windows Server 2022, PowerShell. Không sửa một dòng mã nào — lượt này là **đo**, không phải viết.

Ô `Subscription` trong kế hoạch còn thiếu ba mảnh: ACP, **resume/cancel ở tầng harness**,
và **trình tự chạm file theo pha**. Báo cáo này đóng hai mảnh sau. ACP không đụng tới.

---

## 1. Đã làm (kèm số đo)

### 1.0. Việc đầu tiên: xác định tài khoản nào được phép chạm

`sagent status` lúc bắt đầu:

```
#209 claude:phu#1       PID 11052    1m18s
#210 claude:tns#1       PID 1736       21s
```

`#210` chính là phiên đang viết báo cáo này. **Cả hai tài khoản claude đều có phiên sống**,
mà nhà cung cấp xoay vòng refresh token: một lần refresh là mọi bản sao cũ chết. Nên mọi
phép đo cần chạy `claude` bằng danh tính thật đều **không làm**. `codex`, `cursor`,
`antigravity:may`, `grok:api` rảnh (`sagent ds`) nên đo thoải mái.

Thay vào đó, với claude dùng hai đường vòng **không tốn token và không chạm token thật**:

- **Hồ sơ sandbox**: `CLAUDE_CONFIG_DIR` trỏ vào thư mục tạm rỗng, cộng biến `ANTHROPIC_API_KEY`
  đặt bằng một chuỗi vô nghĩa. CLI đi hết đường thật rồi chết ở `401 authentication_error`
  (`"result":"Failed to authenticate. API Error: 401 API key is invalid."`, `total_cost_usd: 0`).
  **Đã kiểm chứng là kín**: chụp `~/.claude` mức 1 cộng `~/.claude.json` trước và sau lượt chạy,
  `diff` ra **rỗng** — `CLAUDE_CONFIG_DIR` bao trọn cả auth lẫn config, không rò sang cây dùng chung.
- **Quan sát chính phiên đang chạy**: chụp mtime cây cấu hình trước và sau một lượt hỏi của
  chính `#210` và `#209`. Đây là pha `prompt` **trên harness thật, tài khoản thật**, chi phí thêm bằng 0.

### 1.1. CÂU 1 — `--resume` / huỷ phiên ở tầng harness

Nhắc lại cho khỏi lẫn: `flow resume` / `flow huy` là của **bộ chạy flow**
(`internal/flow/approve.go`). Phần dưới đây là của **từng CLI**, chuyện khác hẳn.

#### a) Có cờ nối lại không — nguyên văn `--help`

**claude 2.1.235** — `LamDuoc`:

```
  -c, --continue                        Continue the most recent conversation in
                                        the current directory
  -r, --resume [value]                  Resume a conversation by session ID, or
                                        open interactive picker with optional
                                        search term
  --fork-session                        When resuming, create a new session ID
                                        instead of reusing the original (use
                                        with --resume or --continue)
  --session-id <uuid>                   Use a specific session ID for the
                                        conversation (must be a valid UUID)
  --no-session-persistence              Disable session persistence - sessions
                                        will not be saved to disk and cannot be
                                        resumed (only works with --print)
```

**codex-cli 0.147.0** — `LamDuoc`. Quan trọng: có ở **cả hai tầng**. Ở tầng gốc:

```
  resume          Resume a previous interactive session (picker by default; use --last to continue
                  the most recent)
  fork            Fork a previous interactive session (picker by default; use --last to fork the
                  most recent)
  archive         Archive a saved session by id or session name
  delete          Permanently delete a saved session by id or session name
```

và ở tầng `codex exec` — tức đúng đường headless mà dự án đang dùng:

```
Usage: codex exec [OPTIONS] [PROMPT]
Commands:
  resume  Resume a previous session by id or pick the most recent with --last
```

**cursor-agent 2026.08.11-e8db854** — `LamDuoc`:

```
  --resume [chatId]           Select a session to resume (default: false)
  --continue                  Continue previous session (default: false)
Commands:
  ls                          Resume a chat session
  resume                      Resume the latest chat session
  create-chat                 Create a new empty chat and return its ID
```

**antigravity / `agy` 1.1.18** — `LamDuoc`:

```
  -c                              Short alias for --continue
  --continue                      Continue the most recent conversation
  --conversation                  Resume a previous conversation by ID
```

**grok (@vibe-kit/grok-cli)** — `KhongLamDuoc`. Đây là **toàn bộ** `grok --help`, không cắt
dòng nào, để thấy là không có chỗ nào giấu cờ nối lại:

```
Usage: grok [options] [command] [message...]

Options:
  -V, --version               output the version number
  -d, --directory <dir>       set working directory
  -k, --api-key <key>         Grok API key (or set GROK_API_KEY env var)
  -u, --base-url <url>        Grok API base URL (or set GROK_BASE_URL env var)
  -m, --model <model>         AI model to use
  -p, --prompt <prompt>       process a single prompt and exit (headless mode)
  --max-tool-rounds <rounds>  maximum number of tool execution rounds (default: 400)
  -h, --help                  display help for command

Commands:
  git                         Git operations with AI assistance
  mcp                         Manage MCP (Model Context Protocol) servers
```

Hai lệnh con cũng đã mở ra xem: `grok git` chỉ có `commit-and-push`, `grok mcp` chỉ quản MCP.
Không phải "chưa kiểm tra" — là **đã đo và không có**.

#### b) Nối lại thì nối lại CÁI GÌ

Phép đo: chạy một lượt ngắn nhét một mã bí mật, rồi nối lại và hỏi lại mã đó.

| Harness | Lệnh nối lại | Hỏi lại mã | ID phiên | input_tokens qua các lượt |
|---|---|---|---|---|
| codex | `codex exec resume --json --last` | **ZULU-7391** đúng | `01a02907-...2befe` giữ nguyên | 17.571 → 35.182 |
| cursor | `-p --continue` | **YANKEE-4482** đúng | `2c93816e-...1ea50` giữ nguyên | 4.909 → 4.724 |
| cursor | `-p --resume <chatId>` | **YANKEE-4482** đúng | cùng ID trên | 4.885 |
| agy | `--continue` | **XRAY-1157** đúng | `6d0bfa69-...6052d` giữ nguyên | 13.720 → 27.784 |
| agy | `--conversation <ID>` | **XRAY-1157** đúng | cùng ID trên | 42.171 (`num_turns: 3`) |

Kết luận: ba harness này nối lại **cả hội thoại**, không phải chỉ context. Bằng chứng không chỉ
là câu trả lời đúng mà là **`input_tokens` phình lên theo số lượt** (codex 17k lên 35k, agy
13,7k lên 27,8k lên 42,2k) — tức lịch sử được gửi lại nguyên, chứ không phải một con trỏ.
`agy` còn đếm cộng dồn `num_turns: 1, 2, 3` trong cùng `conversation_id`.

Cursor là ngoại lệ đáng chú ý: `input_tokens` **không** phình (4.909, 4.724, 4.885) trong khi
`cacheReadTokens` đứng ở 8.352. Lịch sử được giữ ở phía máy chủ theo `chatId` chứ không nhét lại
vào từng yêu cầu. Cùng một kết quả với người dùng, khác hẳn nhau về giá.

**claude — không đo được bằng cách chạy thật, và đây là lý do:** cả `claude:tns` lẫn `claude:phu`
đều đang có phiên sống. Chạy một lượt `claude` bằng một trong hai danh tính đó là mở đường cho
một lần refresh xoay vòng, tức tự giết phiên đang chạy. Không làm. **Đo bằng hiện vật trên đĩa
thay thế** — đọc chính bản ghi phiên của `#210` tại
`~/.claude/projects/<slug>/8c598c73-....jsonl`, 251 bản ghi:

```
TYPE: {'user': 54, 'assistant': 99, 'attachment': 59, 'ai-title': 14,
       'atis-latch': 14, 'last-prompt': 13, 'queue-operation': 2}
KEYS: aiTitle, atis, attachment, content, cwd, effort, entrypoint, gitBranch,
      isSidechain, lastPrompt, leafUuid, message, operation, parentUuid,
      permissionMode, promptId, promptSource, requestId, sessionId,
      sourceToolAssistantUUID, timestamp, toolUseResult, type, userType, uuid, version
```

Trên đĩa có **nguyên văn cả `message`/`content` của hai chiều lẫn `toolUseResult`**, kèm `cwd`,
`gitBranch`, `permissionMode`, `effort`, `version`, và chuỗi `parentUuid` (chính là thứ
`--fork-session` cần để tách nhánh). Nên **hạ tầng cho "nối lại cả hội thoại" là có đủ trên đĩa**.
Nói rõ ranh giới: đây là kết luận về **cái được lưu**, không phải một lượt resume chạy thật.
Muốn chốt hẳn thì cần một tài khoản claude thứ ba đang rảnh.

**grok — không chạy được lượt nào**, nhưng vì lý do khác và không ảnh hưởng kết luận: nhà bán lại
`modelapi.vn` trả `410 Live search is deprecated. Please switch to the Agent Tools API` cho
`grok-4.5`, và `503 No available channel for model grok-code-fast-1` cho model kia. Dù có chạy
được thì cũng không có gì để nối lại — xem mục 1.2, grok **không ghi một file nào**.

#### c) Một chỗ hỏng của codex đáng ghi vào bảng năng lực

`codex exec resume --help` liệt kê hẳn `-s, --sandbox <SANDBOX_MODE>`, nhưng bộ phân tích đối số
**từ chối chính cờ đó**. Đã thử cả ba thứ tự:

```
$ codex exec resume --last --sandbox read-only --json "..."
error: unexpected argument '--sandbox' found
Usage: codex exec resume --last [SESSION_ID] [PROMPT]

$ codex exec resume --sandbox read-only --json --last "..."
error: unexpected argument '--sandbox' found
Usage: codex exec resume [OPTIONS] [SESSION_ID] [PROMPT]
```

Chỉ `codex exec resume --json --last "<prompt>"` chạy được. Hệ quả cho dự án: đường headless
của codex trong `internal/provider/codex.go` đi kèm cờ sandbox; nếu sau này nối `resume` vào
mà bê nguyên bộ cờ đó sang thì **lệnh vỡ ngay ở khâu phân tích đối số**, chưa kịp gọi API.
Ghi chú thêm: sau `--last` thì dòng `Usage` tự thu lại thành `resume --last [SESSION_ID] [PROMPT]`
— tức không phải "sai thứ tự", mà là cờ đó không tồn tại ở nhánh này dù help nói có.

#### d) Huỷ giữa chừng — giết tiến trình để lại rác gì

Phép đo: bật một lượt dài, chụp cây tiến trình, `taskkill /PID <cha> /F` **không kèm `/T`**
(đúng cảnh huống thật khi một phiên chết), rồi theo dõi con và soi file còn lại.

| Harness | Cây tiến trình | Mồ côi sau khi giết cha | Rác file để lại |
|---|---|---|---|
| codex | `cmd`, `node` (vỏ npm), `codex.exe`, `node` | **4**, chết hết sau **10 s** | rollout jsonl (xem dưới) |
| claude | `claude.exe`, **không đẻ con** | **0** | `sessions/<PID>.json` và `.key` **ở lại** |
| cursor | `cmd` (vỏ .cmd), `conhost`, `powershell`, `node` | **3**, chết hết sau **10 s** | thư mục chat ở lại, **không có khoá** |
| agy | `agy.exe`, chỉ `conhost` | **0** | `presence/<id>.lock`, `crashes/crash_<PID>_*.log` |
| grok | `cmd`, `conhost`, `node` | **2**, chết trong **dưới 6 s** | **không ghi file nào** |

**Phát hiện nặng nhất — giết tiến trình cha KHÔNG dừng được lượt gọi API của codex.**
Đo có dấu thời gian, không suy đoán:

```
t_bat             = 1787395734
turn started_at   = 1787395735
t_giet_cha        = 1787395747     <- taskkill /F tien trinh cha
turn completed_at = 1787395755     <- SAU khi cha chet 8 giay, duration_ms = 19837
t_con_chet_het    = 1787395757
```

Tiến trình mồ côi `codex.exe` **chạy nốt trọn lượt, sinh đủ 400 dòng, tiêu hết hạn mức**, ghi
rollout hoàn chỉnh rồi mới tự thoát 2 giây sau đó. Đúng cái mà bình luận đầu `cmd/sagent/quet.go`
cảnh báo — nay có số. Cây bốn tầng của codex (`cmd`, vỏ npm, `codex.exe`, `node`) là lý do
`taskkill` không kèm `/T` không với tới được.

**claude — chỗ đẻ ra rác, và bằng chứng nó đã đẻ thật.** Mỗi lượt claude tạo hai file đăng ký
phiên khoá theo PID:

```
sessions/12120.json                       (461 B)
sessions/12120.<sha256>.key               (83 B)
```

Nội dung `sessions/<PID>.json` (đã rút gọn):

```json
{"pid":12120,"sessionId":"b9f08816-...","cwd":"...","startedAt":1787395416094,
 "procStart":"134318690142860054","version":"2.1.235","peerProtocol":1,
 "kind":"interactive","entrypoint":"sdk-cli","name":"work-claude-08",
 "messagingSocketPath":"<ong ten cc-msg-491cbb10c7d5e81b2dcaf55538ab1fe9>",
 "updatedAt":1787395416207}
```

Có `procStart` bên cạnh `pid` — tức claude tự phòng chuyện Windows dùng lại PID. Đáng học,
vì `sagent quet` đang vấp đúng chỗ đó (xem mục e).

Hai phép đo đối nhau, chạy trên hồ sơ sandbox nên không chạm tài khoản thật:

- **Thoát sạch** thì cả `<PID>.json` lẫn `<PID>.*.key` bị **XOÁ** (diff pha exit ghi rõ `- XOA`).
- **Giết bằng `taskkill /F`** thì cả hai **CÒN LẠI** nguyên (`CON LAI: 17844.json 461B`).

Nên đống tồn đọng trong `~/.claude/sessions/` là dấu vết của những lần chết **không sạch**:
**13 trên 18** bản ghi là mồ côi (đối chiếu PID trong sổ với `tasklist`), cũ nhất từ
**20/08 11:41**, cộng thêm một file `.key` lẻ (PID 6692) không có `.json` đi kèm — tức một lần
dọn dẹp đứt nửa chừng. Còn một điểm nữa: `sessions/` trong hồ sơ clone là **junction trỏ về
`~/.claude/sessions` dùng chung**, nên đống này **trộn lẫn cả `tns`, `phu` lẫn `goc`** — xem 1.2b.

**agy — khoá không bao giờ được dọn.** Một lượt để lại `presence/<conversation-id>.lock`,
`conversations/<id>.db` và cặp `-wal`/`-shm`. Đếm được **14 khoá presence, 14 db, 19 file log
xoay vòng**, cũ nhất từ **18/08 10:30**. Quan trọng: khoá **không phải rác của việc giết** —
lượt `6d0bfa69` của tôi thoát sạch lúc 17:35 mà khoá của nó vẫn còn nguyên. Riêng
`crashes/crash_<PID>_<uuid>.log` thì đúng là dấu vết của lần chết bẩn: sau 14 hội thoại chỉ có
**đúng 1** file, và nó mang PID 3668 — chính tiến trình tôi vừa giết.

**codex — một suýt-nữa-kết-luận-sai, cứu bằng phép đối chứng.** Sau khi giết, `.codex` còn
`goals_1.sqlite-shm` (32 KB) và `goals_1.sqlite-wal` (16,5 KB). Rất dễ ghi thành "giết để lại
khoá SQLite". Chạy thêm một lượt **thoát hoàn toàn bình thường** rồi đo lại: không những vẫn còn,
mà còn **sinh thêm** `logs_2.sqlite-shm`, `state_5.sqlite-shm`, `state_5.sqlite-wal` (334 KB).
Vậy đó là nếp thường ngày của codex (WAL không checkpoint lúc đóng), **không phải rác của việc huỷ**.

#### e) `sagent quet` với đống rác này

Chạy thật:

```
Phiên #94 grok:api#1 (chết, PID cũ 16204, bật lúc 23:26 19/08)
  · PID 6640    conhost.exe              bắt đầu 17:28:59 22/08
```

Hai điều rút ra. Một, đây là **dương tính giả** đúng như chính lệnh tự cảnh báo: phiên chết từ
19/08 mà tiến trình được gán cho nó là một `conhost.exe` bật **17:28:59 hôm nay** — Windows dùng
lại PID. Câu cảnh báo trong `quet.go` không phải văn vẻ, nó vừa cứu một kết luận sai.
Hai, và đây mới là khoảng trống: `quet` chỉ nhìn **tiến trình**. Nó **không thấy** 13 bản ghi
phiên mồ côi của claude, cũng không thấy 14 khoá `presence` của agy — vì lúc nó chạy thì tiến
trình đã chết cả rồi, chỉ còn file nằm đó.

### 1.2. CÂU 2 — nhật ký chạm file theo pha

#### a) Kiểm atime TRƯỚC, vì nó quyết định cả phương pháp

```
$ fsutil behavior query DisableLastAccess
DisableLastAccess = 3  (System Managed, Last Access Time Updates DISABLED)
```

**atime TẮT trên máy này.** Nghĩa là cách đo mà đề bài gợi ý — so atime trước/sau — **không đo
được chiều ĐỌC**, chỉ còn chiều GHI qua mtime. Không im lặng báo mtime rồi coi như xong. Bù bằng
hai đường khác, cả hai đều không cần công cụ ngoài:

- **GHI**: chụp `mtime + size` toàn cây trước/sau từng pha rồi `diff`.
- **ĐỌC**: claude có sẵn hạng mục debug `file` — `claude -p -d file --debug-file <log>`. Log ghi
  thẳng ra đường dẫn nó dò, kèm dấu thời gian tới mili-giây. Đây là đường **trong chính công cụ**,
  không phải Process Monitor.

#### b) Phát hiện nền: hồ sơ clone gần như toàn junction

`ls -la` hồ sơ `claude:tns#1` — chỉ **2 file thật**, phần còn lại trỏ về `~/.claude` dùng chung:

| Riêng từng tài khoản (file thật) | Dùng chung cả máy (junction về `~/.claude/...`) |
|---|---|
| `.claude.json`, `.credentials.json` | `sessions`, `projects`, `history.jsonl`, `shell-snapshots`, `tasks`, `telemetry`, `plans`, `plugins`, `skills`, `uploads`, `cache`, `backups`, `chrome`, `settings.json`, `mcp-needs-auth-cache.json`, `.last-cleanup`, `.last-update-result.json` |

Khớp đúng `PrivateFiles()` trong `internal/provider/claude.go:74`. Nhưng hệ quả thì đáng nói:
**bản ghi hội thoại, sổ phiên và lịch sử lệnh của cả ba tài khoản nằm chung một chỗ.** Ô kế
hoạch hỏi "config root override có bao trùm config/session/auth?" — câu trả lời đo được là
**auth và config: có; session: không**.

#### c) Nhật ký claude theo pha

**Pha LOGIN / khởi động** — từ log `-d file`, mốc tương đối tính từ `T = 10:47:25.111Z`:

| +ms | Việc | Đường dẫn |
|---|---|---|
| 0 | ĐỌC | chính sách MDM (`MDM settings load completed in 70ms`) |
| +13 | ĐỌC | `<CONFIG_DIR>/settings.json` |
| +17 | ĐỌC | `C:\Program Files\ClaudeCode\managed-settings.json` |
| +19 | ĐỌC | `<cwd>/.claude/settings.json` |
| +22 | ĐỌC | `<cwd>/.claude/settings.local.json` |
| +89 | GHI hụt | `Failed to save config with lock: ENOENT ... .claude.json` |
| +103 tới +622 | **GHI 9 lần** | `.claude.json` (mỗi lần: `.claude.json.tmp.<PID>.<rand>` rồi rename) |
| +322 | ĐỌC | `installed_plugins.json` (không có), `plugins/cache` |
| +845 | ĐỌC mạng | `Remote settings: Fetch failed (http_401) and no cached settings` |
| +868 tới +873 | ĐỌC lần 2 | trọn bộ 4 `settings.json` ở trên, lặp lại |
| +880 | ĐỌC | `<CONFIG_DIR>/skills`, `C:\Program Files\ClaudeCode\.claude\skills` |
| +885 tới +888 | ĐỌC | `<CONFIG_DIR>/agents`, `<CONFIG_DIR>/commands`, `output-styles` |
| — | TẠO | `sessions/<PID>.json`, `sessions/<PID>.<sha256>.key`, một ống tên `cc-msg-<hash>` |
| — | TẠO | `projects/<slug>/`, `projects/<slug>/memory/`, `projects/<slug>/<sessionId>.jsonl` |

Thứ tự đọc settings là **4 tầng, và nó lặp lại nguyên bộ hai lần** trong 900 ms.
Con số đáng nhớ nhất: **`.claude.json` bị ghi lại nguyên tử 9 lần trong 620 ms khởi động**, mỗi
lần một file tạm `.claude.json.tmp.<PID>.<random>` rồi rename. Đó chính là **mặt va chạm của câu
concurrent-refresh** đã đo hôm 21/08: hai tiến trình chung một hồ sơ thì mỗi lần bật là 9 nhịp
đọc-sửa-ghi chồng lên nhau, và dòng `Failed to save config with lock` cho thấy có cơ chế khoá
nhưng nó **hụt ngay nhịp đầu**.

**Pha PROMPT** — đo trên phiên thật `#210` và `#209`, chụp 1.573 file trước/sau một lượt hỏi
(34 giây). Toàn bộ thay đổi, không cắt gì:

```
MOI (0):
SUA (5):
  ~ ~/.claude/.last-cleanup                                     24 -> 24 bytes
  ~ ~/.claude/projects/<slug-phu-1>/13de4405-....jsonl      632407 -> 657230
  ~ ~/.claude/projects/<slug-tns-1>/8c598c73-....jsonl      457573 -> 487733
  ~ ~/.claude/projects/<slug-seo-center>/1b632b5a-....jsonl 17562789 -> 17579164
  ~ ~/.claude/projects/<slug-seo-center>/.../ccr-tip.json        89 -> 89
MAT (0):
```

Ba điều đọc ra được:

1. Một lượt hỏi chỉ ghi **đúng một file transcript nối thêm**, cộng `.last-cleanup`. Không hơn.
2. **`.credentials.json` KHÔNG hề bị chạm.** Refresh token không xảy ra theo từng lượt — đây là
   kết quả **phủ định** nhưng là thứ trực tiếp trả lời "pha nào chạm token".
3. Ba phiên claude chạy song song trong cùng cửa sổ đó, mỗi phiên chỉ ghi transcript **của
   riêng nó**, không giẫm chân nhau — dù cùng nằm trong cây `projects/` dùng chung.

Đối chiếu mtime của các file token: `~/.codex/auth.json` ghi lần cuối **14/08 02:00**, không hề
bị chạm bởi cả bốn lượt codex hôm nay. `~/.cursor/cli-config.json` ghi **17:33:58**, trùng lượt
`cursor-agent status` chứ không trùng hai lượt prompt (17:38 và 17:51).
`.clones/claude/tns/1/.credentials.json` ghi **17:29:01**, trùng lúc phiên `#210`
khởi động. Nghĩa là: **file token bị ghi lúc bật phiên, không phải lúc hỏi.** Việc ghi lúc bật
phiên đó là một lần refresh hay chỉ là ghi lại nguyên trạng thì **không xác định được** — muốn
biết phải đọc nội dung token, đúng thứ không được đụng.

**Pha REFRESH — không đo được, và đây là lý do chứ không phải chỗ bỏ trống.** Ép hết hạn, buộc
refresh hay đua hai tiến trình vào cùng credential trên `claude:tns` là tự giết phiên đang viết
báo cáo này; `claude:phu` thì có phiên khác đang chạy. Không có tài khoản claude thứ ba. Với hồ
sơ sandbox thì không có refresh token để mà refresh. Đây là kết luận hợp lệ, không phải "chưa
kiểm tra".

**Pha EXIT** — diff giữa ảnh chụp **giữa lượt** và ảnh chụp **sau khi tiến trình thoát hẳn**:

```
  ~ GHI  ./.claude.json                                33266B -> 33328B
  + TAO  ./backups/.claude.json.backup.1787395596569   33266B
  + TAO  ./.last-cleanup                               24B
  ~ GHI  ./projects/<slug>/<sessionId>.jsonl            9833B -> 11235B
  - XOA  ./sessions/12120.json
  - XOA  ./sessions/12120.<sha256>.key
```

Thoát sạch = ghi nốt transcript, ghi `.claude.json` kèm **một bản sao lưu mới trong `backups/`**,
rồi **xoá sổ đăng ký phiên**. Đúng hai file bị xoá này là thứ ở lại khi bị giết (mục 1.1d).

Ghi chú cách tách pha: lượt sandbox chết ở 401 nhưng CLI **thử lại 11 lần trong gần 3 phút**.
Khoảng lặng dài đó chính là cái vạch chia — mọi thứ ghi trước nó là khởi động, mọi thứ ghi sau
nó là thoát. Không cần công cụ nào để tách, chỉ cần chụp thêm một ảnh **giữa lượt**.

#### d) Nhật ký ba harness còn lại

Làm thêm ngoài mức tối thiểu. Tách pha bằng dấu thời gian của chính các lần ghi.

**codex** (`~/.codex`):

| Pha | File |
|---|---|
| khởi động 17:45:04 tới :08 | `goals_1.sqlite-shm`, `logs_2.sqlite`, `models_cache.json`, `plugins/cache/openai-curated-remote/*/.codex-remote-plugin-install.json` (5 file), `cache/codex_apps_server_info/<sha1>.json`, `cache/codex_apps_tools/<sha1>.json` (1,4 MB) |
| prompt 17:45:06 tới :07 | `sessions/2026/08/22/rollout-<ISO>-<uuid>.jsonl`, `goals_1.sqlite-wal` |
| kết thúc 17:45:23 | `memories_1.sqlite`, `queue_1.sqlite`, `state_5.sqlite` |
| **không bao giờ** | `auth.json` (mtime vẫn là 14/08) |

Bản ghi để `resume` đọc là `rollout-*.jsonl`, xếp theo **năm/tháng/ngày**. Ghi chú vận hành:
lượt bị giết cũng để lại một rollout **hợp lệ và mới nhất** (dòng cuối vẫn parse được thành JSON),
nên `codex exec resume --last` sau một lần huỷ sẽ vớ đúng cái phiên vừa bị cắt.

**cursor** (`~/.cursor`):

| Pha | File |
|---|---|
| khởi động 17:51:20 tới :23 | `skills-cursor/.sync-manifest.json`, `statsig-cache.json` (**726 KB**) |
| prompt 17:51:25 tới :27 | `projects/<slug>/worker.log`, `chats/<hash-workspace>/<chatId>/meta.json`, `chats/<hash-workspace>/<chatId>/store.db` (SQLite 64 KB), `projects/<slug>/agent-transcripts/<chatId>/<chatId>.jsonl` |
| **không** | không một file `lock`/`.tmp`/`-wal`/`-shm` nào sót lại sau khi giết |

Chi tiết đáng lưu: chat được đánh chỉ mục theo **băm của workspace** (`7359a1ec...`), nên
`--continue` là "lượt gần nhất **trong workspace này**", không phải trên toàn máy.
Cursor là harness duy nhất **không để lại khoá nào** sau khi bị giết.

**agy** (`~/.gemini/antigravity-cli`):

| Pha | File |
|---|---|
| khởi động 17:52:05 tới :07 | `cli.log`, `crashes/crash_<PID>_<uuid>.log`, `last_check.timestamp`, `conversation_summaries.db-wal` và `-shm`, `cache/onboarding.json`, `updater/update_status.json` |
| prompt 17:52:11 | `conversations/<id>.db` và `-wal` và `-shm`, **`presence/<id>.lock`** |
| trong lượt 17:52:12 | `log/cli-<YYYYMMDD_HHMMSS>.log` |
| **không có trên đĩa** | token — nằm ở Windows Credential Manager, khoá cố định `gemini:antigravity` |

**grok**: chụp toàn bộ hồ sơ `grok/api/1` trước và sau một lượt — **giống hệt từng byte**;
`fleet.log`, `user-settings.json.bak`, `.grok/user-settings.json` đều nguyên mtime cũ (20/08, 18/08).
grok **chỉ đọc, không ghi gì**. Đây chính là lời giải thích cấu trúc cho mục 1.1a: nó không có
`--resume` vì **không có chỗ nào để nối lại từ đó**. Hai chuyện khớp nhau, không phải hai chuyện rời.

### 1.3. Nghiệm thu

Chạy từng lệnh riêng, kiểm bằng mã thoát:

```
go build ./...   -> BUILD_EXIT=0
go vet ./...     -> VET_EXIT=0
go test ./...    -> TEST_EXIT=0   (26 gói ok, 0 FAIL)
```

Không sửa bảng năng lực provider nên không có test nào cần phải đỏ khi gỡ phần sửa ra.

---

## 2. Sự cố

1. **atime bị tắt toàn máy** (`DisableLastAccess = 3`). Đây là sự cố về **phương pháp**: cách đo
   mà đề bài đưa ra chỉ chạy được một nửa. Đã đổi sang `-d file` của claude cho chiều đọc, và
   nói thẳng ở 1.2a thay vì lặng lẽ báo mỗi mtime.
2. **Cả hai tài khoản claude đều bận.** Chặn hẳn hai phép đo: resume chạy thật, và cả pha refresh.
   Ghi là "không đo được" kèm lý do, không đoán.
3. **grok không gọi được model.** `410 Live search is deprecated` với `grok-4.5`,
   `503 No available channel` với `grok-code-fast-1`, cả hai từ nhà bán lại `modelapi.vn`.
   Không cản kết luận vì grok không ghi file phiên nào.
4. **`agy --add-dir` chết trong chế độ headless.** `agy -p --add-dir <dir> ...` trả
   `Error: permission check failed for command "agy --help": user denied permission to run command`.
   Bỏ `--add-dir` thì chạy bình thường. Đáng chú ý vì `internal/provider/antigravity.go:156`
   khai `ArgsThuMuc` trả về đúng `--add-dir`, mà cờ đó lại có bằng chứng gãy ở chế độ `-p`.
   **Mới thấy đúng một lần, chưa dựng lại có hệ thống** nên chưa đủ để sửa bảng năng lực — xem mục 3.
5. **Thứ tự cờ của `agy` là bắt buộc.** `agy -p --output-format json "<prompt>"` báo
   `-p took "--output-format" as its prompt`. Dự án đã đặt đúng
   (`--output-format stream-json -p <prompt>`, `antigravity.go:53`) nên không có lỗi. Ghi lại để
   ai sửa file đó sau này biết thứ tự ở đây không phải sở thích.
6. **Suýt kết luận sai về `-wal`/`-shm` của codex**, cứu bằng phép đối chứng thoát sạch (1.1d).
7. **Khoá API của grok in thẳng ra màn hình** khi `cat` file `user-settings.json` lúc đo. Không
   đưa giá trị nào vào báo cáo này. Nhắc lại rằng `internal/redaction` che **lúc đọc**, nên `cat`
   thẳng một file cấu hình trong lúc đo là đi vòng qua tầng đó.

---

## 3. Bước tiếp theo

1. **Dựng lại cho ra ngô khoai lỗi `agy --add-dir` ở chế độ `-p`** (sự cố 4). Nếu đúng là gãy
   thì `NLThuMuc` của antigravity đang khai `LamDuoc` mà thực tế hỏng ở đúng chế độ dự án dùng —
   phải hạ khai báo và sửa `ArgsThuMuc`. Chạy 5 lượt có cờ và 5 lượt không cờ rồi đếm.
2. **Ghi việc `-s/--sandbox` không dùng được với `codex exec resume` vào chỗ nào mã đọc được**,
   trước khi có ai nối resume vào `internal/provider/codex.go`. Hiện chỉ nằm trong báo cáo này.
3. **Cho `sagent quet` nhìn được rác FILE, không chỉ tiến trình** (1.1e). Ba việc cụ thể, đều
   đối chiếu được bằng PID và tiến trình còn sống:
   `~/.claude/sessions/<PID>.json` và `.key` mồ côi · `~/.gemini/antigravity-cli/presence/*.lock`
   · `crashes/crash_<PID>_*.log` của agy. Hiện có **13 + 14 + 1** món đang nằm đó.
   Mượn luôn mẹo của claude: đối chiếu cả `procStart`, không chỉ `pid`, để khỏi dương tính giả.
4. **Đo pha refresh khi có một tài khoản claude rảnh.** Cần đúng một tài khoản thứ ba không phiên.
   Đây là mảnh duy nhất của Câu 2 còn trống.
5. **Chạy một lượt resume thật của claude** để chốt 1.1b, cùng điều kiện như mục 4.
6. **Cân nhắc tách `sessions/` và `projects/` ra khỏi junction dùng chung** (1.2b), hoặc ít nhất
   nói ra trong tài liệu rằng chúng dùng chung. Hiện `sagent ds` trình bày ba tài khoản như ba
   thứ tách rời, mà bản ghi hội thoại thì nằm chung một rổ.
7. **Chưa đụng tới ACP** — mảnh thứ ba của ô `Subscription`, ngoài phạm vi lượt này.

---

## 4. Bảng: Việc | Model | Effort

| Việc | Model | Effort |
|---|---|---|
| Đọc kế hoạch, khoanh phạm vi hai câu, dựng nhánh sạch | Opus 5 (1M) | thấp |
| Kiểm `sagent status`/`ds`, quyết tài khoản nào được chạm | Opus 5 (1M) | thấp |
| Thu `--help` nguyên văn của 5 CLI, đọc cả lệnh con | Opus 5 (1M) | thấp |
| Thiết kế và chạy phép đo "nối lại nhớ gì" (mã bí mật, 3 harness, 5 lượt) | Opus 5 (1M) | vừa |
| Dò lỗi cờ `-s/--sandbox` của `codex exec resume` (3 thứ tự) | Opus 5 (1M) | vừa |
| Phân tích hiện vật transcript claude thay cho resume chạy thật | Opus 5 (1M) | vừa |
| Đo huỷ giữa chừng: cây tiến trình và mồ côi có dấu thời gian, 5 harness | Opus 5 (1M) | **cao** |
| Phép đối chứng `-wal`/`-shm` của codex | Opus 5 (1M) | vừa |
| Đối chiếu 18 bản ghi `~/.claude/sessions` với `tasklist` | Opus 5 (1M) | vừa |
| Kiểm atime, đổi phương pháp sang `-d file` | Opus 5 (1M) | vừa |
| Dựng hồ sơ sandbox 401 và chứng minh nó kín với cây dùng chung | Opus 5 (1M) | **cao** |
| Nhật ký 4 pha của claude (login/prompt/refresh/exit) | Opus 5 (1M) | **cao** |
| Nhật ký theo pha của codex, cursor, agy, grok | Opus 5 (1M) | vừa |
| Nghiệm thu build/vet/test, viết báo cáo | Opus 5 (1M) | vừa |

Cả lượt chạy trên một model duy nhất — Opus 5 bản 1M context. Không gọi subagent, không dùng
workflow: việc này là một chuỗi phép đo có thứ tự, mỗi phép quyết định phép sau
(atime tắt thì phải đổi cách đo; `-wal` còn lại thì phải có đối chứng), nên chia ra song song là hỏng.

---

## 5. Nhận xét tự do

**Thứ đáng giá nhất lượt này không phải một con số mà là một phép đối chứng.** Sau khi giết codex,
`.codex` còn lại `goals_1.sqlite-wal` và `-shm`. Viết "giết tiến trình để lại khoá SQLite" thì
gọn, nghe có lý, và **sai**. Chạy thêm một lượt thoát sạch mới lòi ra là codex luôn để lại như
thế, thậm chí còn nhiều hơn. Bài học ứng ngay vào cách đọc bảng năng lực: một quan sát sau khi
làm X **chưa phải** hệ quả của X, chừng nào chưa có lượt không-X để so. Ba phép đo lớn nhất trong
báo cáo này đều có cặp đối chứng — claude thoát sạch với bị giết, agy khoá presence với crash log,
codex WAL sạch với bẩn — và cả ba lần cái đối chứng đều đổi kết luận.

**Chỗ thứ hai: "không đo được" hoá ra là một kết quả, không phải một lỗ hổng.** Hai câu đứng
trước một hàng rào thật — cả hai tài khoản claude đều đang chạy, mà nhà cung cấp thì xoay vòng
refresh token. Cái bẫy ở đây không phải là bỏ cuộc, mà là **thoả hiệp cho xong**: chạy một lượt
`claude` bằng danh tính thật, lấy được số, rồi giết mất phiên và mất trắng cả lượt. Đường thứ ba
tốt hơn hẳn — quan sát **chính phiên đang chạy**. Nó cho ra phép đo pha `prompt` trên harness
thật, tài khoản thật, chi phí thêm đúng bằng không, và cái đọc ra được lại là một kết quả phủ
định có sức nặng: **một lượt hỏi không hề chạm `.credentials.json`.** Câu "file nào bị chạm lúc
refresh" vẫn còn trống, nhưng giờ nó trống một cách có hình dạng — biết chắc là **không phải** ở
pha prompt.

**Chỗ thứ ba, và tôi nghĩ đây là chỗ ô `Subscription` đang giấu nợ.** Bốn câu đã đóng trước đây
đều là câu về **một tiến trình đang sống**: biến tách ở đâu, token nằm đâu, headless ra sao, kết
quả có cấu trúc không. Hai câu lượt này là câu về **cái còn lại sau khi tiến trình chết** — và
đó là chỗ mọi harness đều bẩn theo kiểu riêng. claude bỏ lại sổ đăng ký phiên (13 món, có món từ
20/08). agy bỏ lại khoá presence **kể cả khi thoát tử tế** (14 món, có món từ 18/08). codex thì
tệ hơn cả hai: giết tiến trình cha **không dừng được lượt gọi API** — mồ côi chạy thêm 8 giây,
sinh đủ 400 dòng, tiêu hết hạn mức, ghi rollout hoàn chỉnh rồi mới thoát. Một hạm đội `--copies N`
bị dừng nửa chừng thì tất cả những thứ đó nhân lên N lần. `sagent quet` được sinh ra đúng cho
việc này và câu cảnh báo PID-dùng-lại của nó vừa cứu tôi khỏi một kết luận sai — nhưng nó nhìn
**tiến trình**, mà đến lúc ai đó gõ `quet` thì tiến trình chết cả rồi, chỉ còn file. Đó là mảnh
tự nhiên tiếp theo, và nó không cần thêm phép đo nào nữa — số liệu để dựng đã nằm hết trong mục 1.1d.

**Cuối cùng, một điều nhỏ mà tôi thấy vui.** grok không có `--resume`, và grok cũng không ghi
một byte nào xuống đĩa. Hai sự thật ấy đo bằng hai cách hoàn toàn tách rời — một cái đọc `--help`,
một cái so ảnh chụp thư mục — rồi gặp nhau đúng ở một chỗ. Không phải hai kết quả rời, mà là
**cùng một sự thật nhìn từ hai phía**: không có gì trên đĩa thì không có gì để nối lại. Khi hai
phép đo độc lập ăn khớp như thế, độ tin của cả hai đều tăng lên. Đó cũng là lý do đáng bỏ công
đo cả những harness đã biết trước câu trả lời.
