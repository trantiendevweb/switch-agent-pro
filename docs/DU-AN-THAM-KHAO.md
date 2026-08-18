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
