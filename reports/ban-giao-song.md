<!-- FILE BÀN GIAO SỐNG — Switch-Agent-Pro. Luật: TNS-AI-HUB/AGENTS.md mục 4.
     Khối mới ghi lên ĐẦU (ngay dưới khung này). Người nhận chỉ đọc khối trên cùng.
     Bắt tay vào thì ghi dòng «🔒 ĐANG CẦM: <AI> từ <giờ>» lên đầu; xong thì thay bằng khối bàn giao.
     Không có ghi chú riêng.

# <giờ từ lệnh date> — <AI> (Claude/Codex/Antigravity): <việc, mã phiếu>
1. Đang làm:
2. Đã xong + lệnh kiểm + kết quả thật (ĐO RỒI / SUY TỪ MÃ / NHỚ LÀ CHẠY):
3. Dở ở (file:dòng) → bước kế tiếp (đúng một):
4. Không được đụng:
5. Chờ chủ dự án:
6. git status / commit / push:
-->

# 02/11/2026 16:43 — Claude (quan-ly-site-91, ca đêm toàn máy): vá 3 test chập chờn
1. Đang làm: test `go test ./...` chập chờn (bàn giao cũ dòng 17). Không mã phiếu.
2. Đã xong (ĐO RỒI): GPT điều tra d285 → 3 thất bại thật (2 test flow đỏ oan vì giàn dựa giờ thật; 1 race stderr sản phẩm ở internal/plugin/chay.go). Vá `945439a` (main, đã push): test flow dùng chốt thật; chay.go chỉ khởi cmd.Wait (sync.Once) khi stdout EOF hoặc trong Dong(). Kiểm: cấy lỗi đỏ đúng chỗ; đích -count=100/50 xanh; `go test ./... -count=2` exit 0; `go vet ./internal/...` sạch.
3. Dở: không. Bước kế (một): theo dõi CI/lượt chạy tới — nếu còn flake khác thì giao GPT dieu-tra với output thật.
4. Không đụng: `.rc-test-c.log` (rác log 26/08, chưa rõ chủ).
5. Chờ chủ dự án: không.
6. git: main 945439a đã push; worktree GPT đã gỡ.

# 08/10/2026 17:19 — Codex: cập nhật SỔ theo ba merge 22/08
1. Đang làm: đã hoàn tất cập nhật tài liệu, không đổi mã.
2. Đã xong + lệnh kiểm + kết quả thật (ĐO RỒI / SUY TỪ MÃ / NHỚ LÀ CHẠY): ĐO RỒI — `git show` xác nhận 3 commit; kiểm đếm `97/2/0/1`; `git diff --check` = 0; `go test ./...` đỏ 1 bài deadline, chạy riêng 3 lần = `ok`.
3. Dở ở (file:dòng) → bước kế tiếp (đúng một): không có → Claude đọc `git diff --check` và diff hai tệp.
4. Không được đụng: mã nguồn, secrets, `.env*`, production và mọi tệp ngoài hai tài liệu của phiếu.
5. Chờ chủ dự án: không.
6. git status / commit / push: 2 tệp sửa · không commit · không push.

# 03/10/2026 00:01 — khởi tạo file bàn giao
- File bàn giao duy nhất của dự án, tạo theo TNS-AI-HUB/AGENTS.md mục 4. Chưa có khối bàn giao nào.
