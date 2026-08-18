# Bản đồ dự án tham khảo

> Nguồn: [`KE-HOACH-GOC.md`](KE-HOACH-GOC.md) mục 3 — kế hoạch phát triển do chủ
> dự án giao. Tách ra file riêng vì **đã mất một lần**: ngày 18/08 cần danh sách
> này để giao đội agent đi học, lục toàn bộ repo và 8 thư mục phiên Claude trên
> máy đều không thấy — nó chỉ nằm trong file tải lên của một lượt chat cũ.
> Từ nay ai cần cũng tìm được ở đây.

## Luật dùng (rút từ kế hoạch gốc, mục 2)

- Học và **tái triển khai nguyên lý** thì thoải mái với mọi dự án.
- **Chép mã trực tiếp** thì phải xác nhận giấy phép tương thích, giữ attribution,
  và ghi vào [`OPEN_SOURCE_LEDGER.md`](OPEN_SOURCE_LEDGER.md): dự án, URL, license,
  commit, phần đã dùng.
- **AGPL/GPL chỉ đọc để hiểu hành vi**, không chép mã, trừ khi chủ dự án chủ động
  chấp nhận nghĩa vụ giấy phép.
- Không chọn dự án theo số sao. Chọn theo mức khớp với **một boundary cụ thể**.

## Xếp ưu tiên cho sagent — theo lỗ hổng ĐÃ ĐO, không theo thứ tự trong kế hoạch

| # | Dự án | Giấy phép | Vá lỗ hổng nào của sagent | Bằng chứng lỗ hổng |
|---|---|---|---|---|
| 1 | [Agent Client Protocol](https://github.com/agentclientprotocol/agent-client-protocol) + [ACP Go SDK](https://github.com/coder/acp-go-sdk) | Apache-2.0 | **Sự kiện có cấu trúc.** Hiện chỉ biết bước `running/done`; output agent là một khối chữ (Grok trả JSONL, Claude trả văn xuôi). Nên 3D không thể nói "đang làm gì" chi tiết, và không phát hiện được agent chạy suông. | 18/08: bước `done` mà agent chỉ trả câu từ chối quyền; phải viết `khongCoKetQua()` đoán theo chữ ký chuỗi — cách chữa cháy, không phải giao thức |
| 2 | [Agent Deck](https://github.com/asheshgoplani/agent-deck) | MIT | Anh em gần nhất: Go, fleet, worktree, fork/resume, project grouping, web command center | sagent có fleet + worktree nhưng chưa có fork/resume phiên, chưa gom theo project |
| 3 | [Gas Town](https://github.com/gastownhall/gastown) | MIT | **Agent identity, handoff, merge queue, watchdog.** Đúng câu hỏi của chủ dự án: "ai là leader, ai là nhân viên, nhiệm vụ ra sao" | 18/08: bảng vẽ và 3D không hiện được vai trò; thợ commit lên nhánh riêng nhưng không có hàng đợi trộn |
| 4 | [multiclaude](https://github.com/dlorenc/multiclaude) | MIT | Daemon, IPC, atomic state, health loop, crash recovery, vòng đời worktree | sagent không có daemon; PID nằm trong SQLite, chết máy là mồ côi — đã phải viết `session.sweep` để quét |
| 5 | [CCManager](https://github.com/kbwo/ccmanager) | MIT | **Nhận biết trạng thái theo từng CLI**, PTY không cần tmux | 18/08: vỏ `.cmd` cắt prompt, Grok trả 503 in ra như câu trả lời — không mặt nào nhận ra |
| 6 | [Agent of Empires](https://github.com/agent-of-empires/agent-of-empires) | MIT | Dashboard: ACP structured view, duyệt trên điện thoại, multi-repo | 3D hiện chỉ vẽ được lượt chạy mới nhất |
| 7 | [Beads](https://github.com/gastownhall/beads) | theo repo | Đồ thị phụ thuộc, claim/close task, ready queue, bộ nhớ bền | flow DAG có rồi, nhưng chưa có hàng đợi task và bộ nhớ giữa các lượt |
| 8 | [LiteLLM](https://github.com/BerriAI/litellm) | kiểm tại commit | Chuẩn hoá request nhiều provider, model registry, fallback, theo dõi chi phí | `internal/aiapi` mới có fallback thô, chưa ghi usage/cost |
| 9 | [farion1231/cc-switch](https://github.com/farion1231/cc-switch) | MIT | Quản lý provider/config, SQLite SSOT, ghi nguyên tử, local proxy, circuit breaker | **ĐÃ HỌC** 18/08 — xem dưới |
| 10 | [CCPM](https://github.com/automazeio/ccpm) | MIT | PRD → epic → task, chạy song song | flow `code` mới chia được 2 phần bằng prompt |
| 11 | [Compound Engineering](https://github.com/EveryInc/compound-engineering-plugin) | theo repo | Tích luỹ tri thức sau mỗi task | `docs/DO-LUONG.md` đang làm việc này thủ công |
| 12 | [LocalAI](https://github.com/mudler/LocalAI) | MIT | Endpoint tương thích OpenAI/Anthropic tại máy, registry capability | chưa đo Ollama/LocalAI |
| 13 | [OpenHands](https://github.com/OpenHands/OpenHands) | MIT | Control plane, conversation event, siết bảo mật | tham khảo sau |
| 14 | [New API](https://github.com/QuantumNous/new-api) | **AGPL-3.0** | UX quản lý channel/model, chuyển đổi format | **chỉ đọc hành vi, không chép mã** |
| 15 | [Claude Squad](https://github.com/smtg-ai/claude-squad) | **AGPL-3.0** | UX quản lý session/worktree | **chỉ đọc hành vi, không chép mã** |

## Đã học được gì

### cc-switch — 18/08, lần chạy flow `code` #21

Bước `ke-hoach` (claude:phu) tự clone repo về đọc, rút ba nguyên lý kèm trích dẫn:

1. **Năng lực là method trên type, không phải bảng tra tên.**
   `AppType::supports_local_proxy()` và `is_additive_mode()` (`src-tauri/src/app_config.rs:417,424`)
   — thông điệp và hành vi rẽ theo method. Khớp đúng luật của sagent tại
   `internal/provider/adapter.go`: không được `if provider == "antigravity"` trong lõi.
2. **Dọn dẹp chỉ động vào thứ do chính công cụ tạo ra.**
   `codex_config.rs:2540` ghi rõ *"Clean only CC Switch's placeholder … Real user
   bearer tokens are preserved"* → lệnh dọn nhánh của sagent phải bám tiền tố
   `sagent/`, không bao giờ đụng nhánh người dùng.
3. **Việc phá huỷ phải có dry-run kiểm trước rồi mới ghi.**
   `Database::migrate_from_json_dry_run` (`src-tauri/src/database/migration.rs:28`)
   có test riêng `schema_dry_run_does_not_write_to_disk` → lệnh dọn nhánh mặc định
   chỉ liệt kê, phải thêm `--xoa` mới thật xoá.

Và chỗ **không** học được, agent tự nói ra: *"cc-switch là app Tauri quản config
tĩnh, KHÔNG có worktree/fleet nên không học được gì về vòng đời nhánh git."*

Chưa chép dòng mã nào nên chưa phải ghi vào `OPEN_SOURCE_LEDGER.md`.


### ACP + Gas Town + Agent Deck — 18/08, lượt học ba dự án song song

Ba agent chạy song song, mỗi agent một dự án trong bảng trên. Kết quả **không đều** — ghi thẳng cả chỗ trắng.

| Bước | Dự án | Kết quả |
|---|---|---|
| `hoc-acp` (claude:tns) | #1 ACP + ACP Go SDK | Có nghiên cứu thật, trích dẫn `file:dòng` từ hai repo upstream |
| `hoc-gastown` (claude:phu) | #3 Gas Town | Có nghiên cứu thật, 31 trích dẫn `file:dòng` |
| `hoc-agentdeck` (antigravity:may) | #2 Agent Deck | **Chưa học được gì** — xem dưới |

#### Bước Agent Deck: chưa học được gì

Bước này **không trả về nghiên cứu**. Toàn bộ output là ba dòng tường thuật ý định
("đang clone", "đang tìm định nghĩa hàm quản lý profile", "đang tìm điểm xử lý
Fork/Resume") rồi `Error: timeout waiting for response`. Không một trích dẫn
`file:dòng`, không một kết luận. Nhánh `sagent/may-1` không có commit nào.

Nên **hàng #2 trong bảng ưu tiên vẫn nguyên trạng chưa nghiên cứu**. Fork/resume
phiên và gom theo project vẫn là lỗ hổng chưa có lời giải tham khảo. Không suy
đoán thay agent: nó chưa kịp mở file nào thì ở đây không có gì để ghi.

Đáng chú ý: bước này **kết thúc bằng lỗi rõ ràng nên bị bắt**. Đây là ca dễ —
`khongCoKetQua` (`internal/api/api.go:841`) không phải làm gì cả. Ba ca khó là ba
ca dưới đây, nơi agent thoát mã 0 và vẫn báo `done`.

---

#### ACP — bốn bài học

**1. Hỏng phải là lỗi giao thức, không phải chữ tiếng Anh trong output.**

- `auth_required` là lỗi JSON-RPC mã `-32000` (`gosdk/errors.go:66-68`), trả về từ
  `session/new` hoặc mọi hoạt động phiên sau `logout`
  (`acp/docs/protocol/v1/authentication.mdx:111,140`).
- Cụt vòng gọi tool là `stopReason = max_turn_requests` — một **giá trị enum**, không
  phải câu chữ (`gosdk/types_gen.go:6149`,
  `acp/docs/protocol/v1/prompt-turn.mdx:304`).

**Vá lỗ hổng đo được nào:** `khongCoKetQua` hiện đang dò chuỗi tiếng Anh cho đúng hai
kiểu hỏng này. `"failed to authenticate"`, `"session expired"`, `"please run /login"`,
`"not logged in"` ở `api.go:882` là để bắt lần chạy #21 — bước `gop` trả nguyên câu
`"Failed to authenticate: OAuth session expired and could not be refreshed"` mà vẫn
`done`, cả flow vẫn `completed`. `"maximum tool execution rounds reached"` ở
`api.go:889` là để bắt bước `soi` (grok) gọi `ls -la` 399 lần trên Windows.
Cả hai chuỗi đều là **chữ ký của một provider cụ thể ở một phiên bản cụ thể** —
provider đổi câu chữ là lá chắn rơi im lặng. Enum thì không đổi.

**2. Bên quyết định quyền phải là sagent, không phải agent.**

Trong ACP agent không tự từ chối rồi in ra chữ; nó gọi `session/request_permission`
và client trả về một trong `allow_once` / `allow_always` / `reject_once` /
`reject_always` (`acp/docs/protocol/v1/tool-calls.mdx:110-148`,
`gosdk/types_gen.go:3316-3319`).

**Vá lỗ hổng đo được nào:** lần chạy #8 (flow `doi-hinh-khong-claude`), cả hai bước
antigravity thoát mã 0, runner đánh dấu `done`, flow báo `completed`, log chỉ chứa
câu từ chối quyền — và **bước sau nuốt luôn câu đó làm dữ liệu rồi "hoàn thành"
trên rác**. Lá chắn hiện tại là dò `"auto-denied"` / `"no output produced"` /
`"headless mode cannot prompt"` (`api.go:873`). Với ACP, sagent chính là bên đã
reject nên nó có sổ riêng: "tôi từ chối N lần" — không cần nhìn output.

**3. Bắt được TRONG LÚC CHẠY, không phải sau khi tiến trình chết.**

Sự kiện `tool_call` tới theo luồng. sagent đếm khi chúng tới; thấy cái thứ 5 cùng
`kind: "execute"`, cùng `rawInput` `ls -la`, cùng `status: "failed"` thì hủy bằng
`session/cancel`.

**Vá lỗ hổng đo được nào:** ở #21, 399 vòng `ls -la` chạy hết trần rồi sagent mới
biết. Chi phí là toàn bộ thời gian và token của 399 vòng đó.

**4. Chỗ CHƯA đo, agent tự khai:** không có bằng chứng nào cho thấy provider nào của
sagent thực sự nói ACP. Ví dụ duy nhất trong SDK là `gemini --experimental-acp`
(`gosdk/example_gemini_test.go:62-87`) — không nằm trong 5 adapter tại
`internal/provider/` (claude, codex, cursor, grok, antigravity). Agent nói rõ:
*"tôi chỉ đọc hai repo, không chạy CLI nào."*

Nên toàn bộ ba bài trên **đúng về mặt giao thức, chưa biết đúng về mặt thực tế**.
Việc nhỏ nhất để biết: chương trình dò đứng ngoài `RunAgents`, bật từng CLI ở chế độ
ACP qua stdio, gửi đúng một request `initialize` với `protocolVersion: 1`
(`gosdk/constants_gen.go:6`), ra bảng 5 dòng CÓ/KHÔNG. 0/5 thì hướng ACP chết tại đó
và không mất gì.

**Hai thứ KHÔNG được bỏ khi bật ACP** (agent nói trước, ghi lại):
- `khongCoKetQua` vẫn là lưới cho provider không nói ACP. Thêm vào đó,
  `stopReason: end_turn` với **0 sự kiện `tool_call`** là một dấu hiệu hỏng mới,
  sạch hơn dò chuỗi — nhưng phải viết ra, không tự có.
- Bằng chứng git (`api.go:775-779`) độc lập với ACP và vẫn đứng vững. `tool_call`
  chỉ nói agent ĐÃ GỌI `edit`; chỉ git mới nói file có đổi thật. Giữ cả hai.

---

#### Gas Town — hai bài học và một chống mẫu

**1. Định nghĩa vai trò là cấu hình tĩnh; trạng thái agent mới là dữ liệu chạy.**

Bằng chứng mạnh nhất không phải cái Gas Town có, mà cái nó **đã bỏ**:
`// Note: RoleBead field removed - role definitions are now config-based`
(`internal/beads/beads_agent.go:49`). Họ từng lưu định nghĩa vai trò trong database
rồi rút về file TOML nhúng binary (`internal/config/roles.go` + các file TOML).

**Vá lỗ hổng đo được nào:** hàng #3 bảng trên — 18/08, bảng vẽ 2D và 3D không hiện
được vai trò. sagent nên tách đúng ranh giới này **ngay từ đầu**, đừng đi vòng như họ.

**2. Ba mảnh rời đáng mang về, mỗi cái dưới 100 dòng và không kéo theo cái nào khác:**
công thức điểm `ScoreMR` (`internal/refinery/score.go`, thuần, có test bảng
`score_test.go`); merge slot — một khoá độc quyền cho nhánh chính, backoff mũ, giải
phóng bằng `defer` (`internal/refinery/engineer.go`, vùng `acquireMainPushSlot`); và
đặt `GIT_AUTHOR_NAME`/`GIT_AUTHOR_EMAIL` theo vai trò lúc spawn thợ
(`internal/config/env.go`).

**Vá lỗ hổng đo được nào:** hàng #3 — thợ commit lên nhánh riêng nhưng **không có
hàng đợi trộn**. Hiện `bangChungWorktree` (`api.go:782`) đọc được nhánh nào có commit,
nhưng không có gì quyết định trộn cái nào trước.

**3. Chống mẫu — issue tracker làm database. ĐỪNG.**

Toàn bộ merge queue của Gas Town là bead có label `gt:merge-request`, và **21 trường
có cấu trúc được serialize thành các dòng `key: value` nhét vào description rồi parse
ngược ra bằng cách tách dấu hai chấm** (`internal/beads/fields.go:761-790` ghi,
`653-680` đọc). Hậu quả lộ ngay trong code của chính họ:

- Tra cứu bằng so khớp tiền tố chuỗi:
  `strings.HasPrefix(issue.Description, "branch: "+branch+"\n")`
  (`internal/beads/beads_mr.go:38,43`) — đổi thứ tự trường là hỏng.
- Phải lọc lại phía client vì bộ lọc của store không tin được:
  `// Skip closed MRs (workaround for bd list not respecting --status filter)`
  (`engineer.go:2091`).
- Trường `MRPhase` không lưu được vào status của store, phải nhét chỗ khác
  (`types.go:74-76`).
- Description trống thì `ParseMRFields` trả `nil` và MR bị **bỏ qua âm thầm**
  (`fields.go:654-656`, `engineer.go:2109-2112`).

Kéo theo cả một Dolt SQL server, một CLI ngoài (`bd`) gọi bằng `exec.Command`, và một
routing table prefix→rig. **Mang công thức điểm và merge slot về; đừng mang cái kho.**
Hàng đợi trộn của sagent cần bảng có cột thật (`branch`, `commit_sha`, `phase`,
`retry_count`, `claimed_by`, `claimed_at`) — sqlite làm tốt hơn hẳn.

**Chỗ agent tự khai là chưa chắc:**
- `internal/refinery/engineer.go` dài 2657 dòng, chỉ đọc ~700 dòng có chọn lọc. Nhánh
  `doMergePR` (merge qua API GitHub/Bitbucket, `engineer.go:775-890`) mới đọc phần
  đầu — nếu sagent chọn đường PR thay vì merge local thì phải đọc lại vùng đó.
- Doc tự mâu thuẫn: `docs/design/architecture.md:66-80` nói Witness "quản lý vòng đời
  polecat", `architecture.md:241-242` nói polecat tự quản còn Witness chỉ quan sát.
  Code (`internal/cmd/done.go:34-42`) đứng về phía "polecat tự quản". Nếu sagent bắt
  chước quan hệ leader/nhân viên này thì **tin code, đừng tin bảng trong doc**.

---

#### Việc nên làm tiếp — xếp theo (giá trị / công sức)

| # | Việc | Giá trị | Công sức | Vì sao xếp ở đây |
|---|---|---|---|---|
| 1 | Set `GIT_AUTHOR_NAME`/`GIT_AUTHOR_EMAIL` theo vai trò lúc spawn thợ | Trung bình | Rất thấp | Vài dòng ở chỗ dựng env tiến trình con. Làm được ngay, không phụ thuộc quyết định nào |
| 2 | Chương trình dò ACP `initialize` cho 5 provider, đứng ngoài `RunAgents` | Rất cao | Thấp | Một bảng 5 dòng CÓ/KHÔNG quyết định số phận cả hàng #1. 0/5 thì đóng hướng, không mất gì. **Chưa có bảng này thì mọi việc ACP khác đều là hứa suông** |
| 3 | Port `ScoreMR` + test bảng | Trung bình | Thấp | Công thức thuần, không phụ thuộc gì. Nhưng vô dụng nếu chưa có #5 |
| 4 | Tách định nghĩa vai trò ra cấu hình tĩnh, tách khỏi trạng thái chạy | Cao | Trung bình | Vá trực tiếp "2D/3D không hiện được vai trò". Làm sai chỗ này về sau rất đắt để sửa — Gas Town đã phải rút lui một lần |
| 5 | Schema bảng `merge_request` trong sqlite (cột thật, không description) | Cao | Trung bình | Quyết định kiến trúc, phải chốt trước #3 và #6 |
| 6 | Merge slot: khoá độc quyền + backoff mũ + giải phóng bằng `defer` | Cao | Cao | Việc đồng thời, dễ sai. Chỉ làm sau khi có #5 |
| 7 | Chạy lại nghiên cứu Agent Deck (bước 18/08 timeout, chưa có gì) | Chưa biết | Thấp | Hàng #2 bảng ưu tiên vẫn trắng. Rẻ để thử lại, nhưng giá trị chưa đo được vì lần trước không ra chữ nào |
| 8 | Thêm dấu hiệu hỏng `stopReason: end_turn` + 0 `tool_call` | Cao | Cao | Sạch hơn dò chuỗi, nhưng **chặn sau #2** — không có provider nói ACP thì không có `tool_call` để đếm |
| 9 | Nối `tool_call`/`tool_call_update` vào runner, thay dần dò chuỗi | Rất cao | Rất cao | Đích đến thật của hướng ACP. Chặn sau #2 |
| 10 | Đọc nốt nhánh `doMergePR` (`engineer.go:775-890`) | Thấp | Trung bình | Chỉ cần nếu chọn đường merge qua PR API. Chưa chọn thì chưa đọc |

Ba việc đầu bảng (#1, #2, #3) đều **không phụ thuộc nhau** và không phụ thuộc quyết
định kiến trúc nào — chạy song song được.

**Chưa chép dòng mã nào** từ ACP, ACP Go SDK hay Gas Town — cả hai agent đều chỉ clone
nông ra thư mục tạm để đọc, không sửa file nào trong repo này. Nên **chưa phải ghi vào**
[`OPEN_SOURCE_LEDGER.md`](OPEN_SOURCE_LEDGER.md). Ba repo đều Apache-2.0 hoặc MIT nên
khi nào thật sự chép thì ghi được, không vướng giấy phép.

---

## Báo cáo

**1. Đã làm**
- Đọc `docs/DU-AN-THAM-KHAO.md` (61 dòng) để khớp giọng và cấu trúc mục "Đã học được gì".
- Đối chiếu mọi khẳng định về lỗ hổng sagent với mã thật, không tin lời agent kể:
  `internal/api/api.go:769` (điểm gọi lá chắn), `:775-779` (bằng chứng git), `:841-891`
  (`khongCoKetQua` — đã đọc cả 3 nhóm chữ ký chuỗi và 2 lần sửa sai trước đó ghi trong
  comment), `internal/api/ketqua_test.go`, `internal/provider/` (đếm được đúng 5 adapter).
- Soạn mục bổ sung: 3 bước → 1 ghi thẳng là chưa học được, 2 có bài học; 7 bài học,
  mỗi bài đủ ba phần (nguyên lý / trích dẫn upstream / lỗ hổng đo được được vá);
  bảng 10 việc xếp theo giá trị-trên-công-sức.
- **Không sửa file nào.** Yêu cầu là trả nguyên văn markdown để chủ dự án tự dán.

**2. Sự cố**
- Không có sự cố kỹ thuật.
- Một hạn chế về dữ liệu vào, phải nói thẳng: **hai bản ACP và Gas Town đều đã bị cắt
  đầu, chỉ còn 6000 ký tự cuối.** Phần bị cắt là mục 1 và 2 của cả hai bản. Nên mục bổ
  sung này chỉ chép lại được những trích dẫn `file:dòng` còn sót trong phần đuôi —
  ACP nói có 13 sự kiện `sessionUpdate` liệt kê đủ tên thật, Gas Town nói có 31 trích
  dẫn; tôi chỉ thấy một phần. Nếu còn bản đầy đủ ở đâu đó thì mục này nên được bổ sung
  lại, không phải viết lại.
- Bước Agent Deck mất trắng: 0 trích dẫn, 0 commit, chỉ có `Error: timeout waiting for
  response`. Hàng #2 bảng ưu tiên vẫn chưa được nghiên cứu.

**3. Bước tiếp theo**
- Chủ dự án dán mục trên vào cuối `docs/DU-AN-THAM-KHAO.md`, ngay sau mục cc-switch.
- Chạy việc #2 trong bảng — chương trình dò `initialize` cho 5 provider. Nó chặn 3
  việc khác (#8, #9) và có thể đóng hẳn hàng ưu tiên #1.
- Song song, làm #1 (git author) và #3 (`ScoreMR`) — cả hai không chặn ai.

**4. Nên xài model gì, effort nào**

| Việc | Model | Effort |
|---|---|---|
| Dán mục vào doc | — | không cần agent |
| Chương trình dò ACP `initialize` cho 5 provider (#2) | Sonnet 5 | medium |
| Set git author theo vai trò lúc spawn (#1) | Sonnet 5 | low |
| Port `ScoreMR` + test bảng (#3) | Sonnet 5 | medium |
| Chạy lại nghiên cứu Agent Deck (#7) | Sonnet 5 | high |
| Tách định nghĩa vai trò khỏi trạng thái chạy (#4) | Opus 5 | high |
| Thiết kế schema hàng đợi trộn (#5) | Opus 5 | high |
| Merge slot — khoá + backoff + defer (#6) | Opus 5 | high |
| Nối `tool_call` vào runner (#9, chỉ sau khi #2 ra kết quả dương) | Opus 5 | high |

**5. Muốn nói gì thì nói ở dưới**

Ba bản nghiên cứu này hội tụ về **cùng một câu**, dù hai agent không nói chuyện với
nhau: hỏng phải là **cấu trúc dữ liệu**, không phải **chữ trong văn bản**. ACP nói
điều đó về giao tiếp agent↔sagent (`auth_required` là mã `-32000`, không phải câu
tiếng Anh). Gas Town nói điều đó về lưu trữ, và nói bằng phản ví dụ — họ nhét 21
trường vào description rồi tách dấu hai chấm, và trả giá bằng `strings.HasPrefix` để
tra cứu. `khongCoKetQua` của sagent hiện đang ở đúng phía sai của cả hai câu: nó dò
chuỗi. Nó là lá chắn tốt cho hôm nay và phải giữ, nhưng nó không phải đích đến.

Ranh giới đáng giữ nhất, cả hai agent tự rút ra độc lập: **mang mảnh rời, đừng mang
cụm.** Gas Town có bảy vai trò, daemon, tmux, Dolt server, mail, convoy, molecule,
formula, dog, wisp. Ba thứ chọn mang về đều dưới 100 dòng và không kéo theo gì. Thứ bị
loại — issue-tracker-làm-database — thì kéo theo tất cả. Áp cùng thước đó cho ACP:
chương trình dò `initialize` là mảnh rời (đứng ngoài `RunAgents`, hỏng thì vứt);
"chuyển toàn bộ runner sang ACP" là cả cụm, và chưa có bảng 5 dòng thì chưa được bàn.

Rủi ro còn lại: mục này ghi 7 bài học từ 2 bước, và cả 7 đều là **kết luận đọc mã, chưa
cái nào chạy**. Bảng ở trên xếp #2 lên gần đầu chính vì thế — nó là việc rẻ nhất biến
một kết luận thành một số đo. Đừng làm #9 trước #2.
