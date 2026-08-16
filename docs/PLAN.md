# ccswitch v2 — Kế hoạch thực thi chi tiết

> ⚠ **Bị thay thế bởi [`docs/MASTER-PLAN.md`](MASTER-PLAN.md)** (2026-08-17) — bản
> hợp nhất với kế hoạch control-plane v1.1 và đổi tên dự án thành **Switch-Agent-Pro**.
> File này giữ lại làm nhật ký thực thi Pha 1 (đã chạy trên Windows).

> Tài liệu này là **bản đồ thi công**, đi kèm bản thiết kế `docs/THIET-KE.md`.
> Thiết kế trả lời "làm cái gì và vì sao"; plan này trả lời "làm theo bước nào,
> xong thì kiểm ra sao". Mỗi bước có **việc cụ thể**, **sản phẩm**, và **tiêu chí
> nghiệm thu (Definition of Done)** để không có bước nào "coi như xong".

## Cách đọc plan

- Đơn vị nhỏ nhất là **bước** (checkbox). Mỗi bước làm được trong ~nửa ngày.
- Mỗi pha kết thúc bằng **DoD** — chưa đạt hết thì pha chưa đóng.
- Ký hiệu: 🎯 mục tiêu · 📦 sản phẩm · ✅ nghiệm thu · ⚠ bẫy đã biết.
- Thứ tự pha là thứ tự làm. Pha sau dựa trên pha trước; **không nhảy cóc** trừ
  Pha 4–6 (các provider) có thể làm song song sau khi Pha 1–3 xong.

## Tổng quan cột mốc

| Pha | Tên | Kết quả bàn giao | Phụ thuộc |
|----|-----|------------------|-----------|
| 0 | Đo giả định | Báo cáo đo + bộ `verify` chạy được | — |
| 1 | Lõi Go + adapter Claude | Binary thay thế `tk` v1 (Win+Linux) | 0 |
| 2 | Chạy song song | `clone` + `fleet` + registry | 1 |
| 3 | Flow ghép được | `flows.toml` + adapter plugin | 1 |
| 3b | UI Dashboard 3D | `tk dash` + web cục bộ | 2 |
| 4 | Adapter Codex | Codex chạy trong hệ | 1, (0) |
| 5 | Adapter Gemini | Gemini chạy trong hệ | 1, (0) |
| 6 | Adapter Cursor | Cursor chạy trong hệ | 1, (0) |

---

## Pha 0 — Đo giả định (nền tảng, không code tính năng)

🎯 Biết primitive "tách thư mục = tách tài khoản" có đứng vững trên từng OS và
từng provider **trước khi** viết Go. Đây là phương châm "đã đo — không suy luận".

- [ ] **0.1** Đo Claude trên **Windows** (xác nhận lại v1): đặt `CLAUDE_CONFIG_DIR=X`
  → Claude ghi `X\.claude.json`, `X\.credentials.json`; `claude mcp list` ở X không
  thấy MCP tài khoản khác. Ghi lại đường dẫn token thật.
- [ ] **0.2** Đo Claude trên **Linux**: token nằm ở **file** trong config dir, hay ở
  `libsecret`/`gnome-keyring`? ⚠ Nếu ở keyring, primitive gãy trên Linux → phải thiết
  kế đường token riêng cho Linux. Đây là rủi ro số 1 của cả pha.
- [ ] **0.3** Đo **Codex CLI**: biến config dir có đúng là `CODEX_HOME`? File token/danh
  tính tên gì? Đặt biến rồi kiểm hai profile không thấy nhau.
- [ ] **0.4** Đo **Gemini CLI**: cơ chế thư mục config (`~/.gemini/`? có biến override
  không?), token ở đâu.
- [ ] **0.5** Đo **Cursor**: có CLI headless + cơ chế config dir không; nếu không thì
  ghi rõ "chưa hỗ trợ được" thay vì cố.
- [ ] **0.6** Đo cách tạo **junction trên Windows từ Go** không cần admin (shell
  `cmd /c mklink /J` hay syscall `DeviceIoControl`), và **symlink trên Linux** (`os.Symlink`).
- [ ] **0.7** Viết các phép đo trên thành script chạy được (mầm của gói `verify/`),
  thoát mã ≠ 0 nếu có mục đỏ.

📦 `docs/DO-LUONG.md` (bảng kết quả từng ô, mỗi ô ghi "đo thế nào") + thư mục
`measure/` chứa script.
✅ **DoD:** mọi ô 0.1–0.6 có kết luận ĐO ĐƯỢC (không phải "chắc là"), và mỗi
provider được đánh dấu `stable` / `experimental` / `unsupported` dựa trên kết quả.

---

## Pha 1 — Lõi Go + adapter Claude

🎯 Một binary Go thay thế trọn vẹn `tk` v1, chạy **Windows + Linux**, bỏ Python.

### 1A. Dựng khung dự án
- [ ] **1.1** `go mod init github.com/trantiendevweb/ccswitch`; Go ≥ 1.22; đặt cây thư
  mục theo `docs/THIET-KE.md` mục 9.
- [ ] **1.2** CLI framework: chọn stdlib `flag` + bộ điều phối subcommand tự viết (giữ
  nhẹ, không kéo thư viện lớn). Địa chỉ hoá `provider:account`, mặc định provider `claude`.
- [ ] **1.3** Thiết lập CI GitHub Actions: build matrix `windows-latest` + `ubuntu-latest`,
  chạy `go vet`, `go test`, và `verify` (phần chạy được không cần đăng nhập).

### 1B. Trừu tượng nền tảng
- [ ] **1.4** Gói `link/`: hàm `LinkDir`/`LinkFile` với build tag `link_windows.go`
  (junction) và `link_linux.go` (symlink). Có hàm `IsLink` và `Unlink` an toàn.
- [ ] **1.5** Gói `jsonutil/`: đọc/ghi `.claude.json` — xử lý **khoá trùng hoa/thường**
  (Go lấy khoá cuối, xác nhận bằng test), **ghi nguyên tử** (tmp + rename), sao lưu `.bak-<ts>`.

### 1C. Adapter interface + Claude
- [ ] **1.6** Định nghĩa `provider.Adapter` (mục 4 thiết kế). Đăng ký adapter vào registry.
- [ ] **1.7** `provider/claude.go`: `EnvVar=CLAUDE_CONFIG_DIR`, `PrivateFiles`, whitelist
  19 khoá (port nguyên `CHIA_SE` từ `cfg.py`), `Identity`, `HasToken`, `Verify`.
- [ ] **1.8** Port whitelist + lý do KHÔNG copy (oauthAccount, cache gói cước…) thành
  hằng có chú thích — giữ **danh sách trắng**, không đổi sang danh sách đen.

### 1D. Verb lõi
- [ ] **1.9** `create <provider:account>`: tạo config dir, `link` phần dùng chung, `seed` whitelist.
- [ ] **1.10** `run <provider:account> [-- args]`: đặt env của **tiến trình con**, chạy CLI.
  ⚠ Tài khoản gốc phải **xoá** biến, không trỏ vào `~/.claude` (bẫy file lạc của v1).
- [ ] **1.11** `list` / bảng tương tác `tk` (đánh số, `*` = đang dùng). ⚠ Không có bàn
  phím (CI/redirect) thì in trợ giúp, không treo.
- [ ] **1.12** `sync` (dong-bo): đẩy whitelist sang mọi profile, có `--dry-run`, có `.bak`.
- [ ] **1.13** `remove`: **xoá an toàn** — gỡ từng link, kiểm không còn reparse point,
  rồi mới `RemoveAll`. ⚠ Bẫy nguy hiểm nhất: `RemoveAll` xuyên junction xoá dữ liệu thật.
- [ ] **1.14** Di trú v1: tự nhận `~/.claude-accounts/*` cũ thành `claude/*`.

📦 Binary `ccswitch` + alias `tk`; script cài cho Win (`.ps1`) và Linux (`.sh`).
✅ **DoD:** trên **cả Win và Linux**: tạo 2 tài khoản, đăng nhập, đổi qua lại
không phải đăng nhập lại, `sync` hoạt động, `remove` không đụng dữ liệu gốc (có
test đặt file mồi trong base rồi đếm lại), `verify` xanh. Không còn phụ thuộc Python.

> **Trạng thái 2026-08-17 — Windows XONG, Linux HOÃN (chưa có VM):**
> - [x] Khung Go + gói `link/jsonutil/provider/profile` + dispatcher (1.1–1.13)
> - [x] Bỏ Python (test khoá JSON trùng), ghi nguyên tử, whitelist 19 khoá
> - [x] `create/run/ds/dong-bo/xoa/verify/goc` chạy thật trên Windows
> - [x] Xoá an toàn (unit test + chạy thật với file mồi), junction không cần admin
> - [x] Di trú v1 (`~/.claude-accounts/*` → `claude:*`) — `ds` nhận đúng email/token
> - [x] `run` end-to-end (`claude --version` qua env biệt lập)
> - [x] Script cài `install/cai-dat.ps1` (Win) + `install/cai-dat.sh` (Linux, chưa đo) + CI 2 OS
> - [ ] **Linux**: build + đo — hoãn tới khi có VM; nhãn Linux giữ `experimental`
> - [ ] Bảng chọn tương tác đánh số (v1 có, v2 mới chỉ liệt kê) — để sau

---

## Pha 2 — Chạy song song

🎯 Mục tiêu 2 (nhiều phiên / 1 tài khoản) và 3 (nhiều tài khoản song song).

- [ ] **2.1** `clone <provider:account> --into N`: chép `.credentials.json` ra N config
  dir tạm, mỗi cái `.claude.json` **riêng** (né đua ghi), nối phần dùng chung.
- [ ] **2.2** Registry tiến trình: `~/.ai-accounts/running.json` — ghi PID, profile,
  thời điểm chạy; dọn mục chết khi khởi động.
- [ ] **2.3** `fleet <flow|adhoc>`: khởi chạy nhiều `run` song song, mỗi tiến trình một
  profile; gom stdout/stderr theo nhãn. Hỗ trợ headless (`run "claude -p …"`).
- [ ] **2.4** Cảnh báo hạn mức: khi N phiên cùng một account, in cảnh báo "tiêu hạn mức
  gấp N" rõ ràng, không im lặng.
- [ ] **2.5** `status`: bảng cho thấy profile nào đang chạy (đọc registry).

📦 Verb `clone`, `fleet`, `status` + registry.
✅ **DoD:** chạy được 4 phiên song song trên 1 tài khoản (worktree riêng nếu bật)
và 3 tài khoản song song; `status` phản ánh đúng; đóng phiên thì registry cập nhật;
không có hỏng `.claude.json` do đua ghi (test chạy 10 phiên đồng thời rồi kiểm JSON hợp lệ).

---

## Pha 3 — Flow ghép được (mục tiêu 5)

🎯 Người dùng tự dựng luồng từ verb, ba tầng.

- [ ] **3.1** Tầng 2 — `flows.toml`: schema (`clone`, `copies`, `parallel`, `worktree`,
  `run`, `desc`); loader + validate; `fleet <tên>` chạy theo định nghĩa.
- [ ] **3.2** Tầng 3 — adapter plugin TOML: người dùng thả file khai báo `env_var`,
  `private_files`, `shared_keys`, lệnh đọc identity → thêm provider "dễ" không cần build lại.
- [ ] **3.3** Ba flow mẫu kèm sẵn: `fanout` (N phiên/1 account), `squad` (đa provider
  cùng task), `agents` (đội headless).
- [ ] **3.4** Tài liệu `docs/FLOW.md` + ví dụ chạy được.

📦 Loader flow + plugin + 3 flow mẫu + docs.
✅ **DoD:** người dùng viết một flow mới trong `flows.toml` và chạy được mà không
sửa mã Go; thêm một provider giả bằng plugin TOML và `list` thấy nó.

---

## Pha 3b — UI Dashboard 3D (mục tiêu 6)

🎯 `tk dash` mở dashboard "phòng điều khiển đội agent" theo thời gian thực.

- [ ] **3b.1** `dash/server.go`: HTTP localhost + WebSocket, đọc `running.json` + hạn
  mức, đẩy cập nhật; assets nhúng bằng Go `embed` (vẫn một binary).
- [ ] **3b.2** Front-end React Three Fiber: cụm provider + **mascot 3D biết đi/biểu
  diễn hoạt động** + orb phiên màu-theo-trạng-thái; panel kính mờ.
- [ ] **3b.3** Áp `design-system/switch-agent-pro/MASTER.md`: token màu/Inter; quy tắc
  threejs (InstancedMesh, FogExp2, ACES, antialias lúc khởi tạo); tôn trọng `prefers-reduced-motion`.
- [ ] **3b.4** **Responsive điện thoại**: panel thành bottom-sheet, điều khiển cảm ứng,
  camera lùi xa hơn trên màn nhỏ.
- [ ] **3b.5** Fallback 2D: bảng phiên khi máy yếu / reduced-motion.

📦 `tk dash` + web nhúng. (Hai file `plan.html` / `index.html` hiện tại
là **nguyên mẫu tĩnh** để duyệt hình khối trước; pha này biến chúng thành app thật đọc dữ liệu sống.)
✅ **DoD:** `tk dash` mở dashboard, phản ánh đúng trạng thái `fleet` đang chạy, mượt
trên điện thoại, tắt UI không ảnh hưởng lõi.

---

## Pha 4–6 — Thêm nhà cung cấp (Codex → Gemini → Cursor)

🎯 Mở rộng đa provider. Mỗi pha lặp cùng một quy trình, dựa trên kết quả Pha 0.

Cho **mỗi** provider (Codex, Gemini, Cursor):
- [ ] **X.1** Viết `provider/<tên>.go` hiện thực `Adapter` theo số liệu đo ở Pha 0.
- [ ] **X.2** Viết `Verify()` chứng minh "tách thật" cho provider đó trên Win+Linux.
- [ ] **X.3** Bổ sung `Identity`/`HasToken` đúng định dạng file token của nó.
- [ ] **X.4** Mascot + màu thương hiệu trong dashboard.
- [ ] **X.5** Tài liệu: cập nhật bảng provider + trạng thái `stable/experimental`.

✅ **DoD từng provider:** `create`, `run`, `list`, `remove`, `verify` chạy đúng;
đổi qua lại không lẫn token; đánh dấu trạng thái trung thực (không gắn `stable`
nếu chưa đo đủ).

---

## Xuyên suốt (làm liên tục, không phải một pha)

- **Kiểm thử:** mỗi verb có test; phần nguy hiểm (`remove`, ghi JSON, link) có test
  hồi quy. Test chạy trên cả hai OS trong CI.
- **An toàn:** không log token; thao tác xoá luôn qua đường "gỡ link trước".
- **Trung thực trạng thái:** provider/OS chưa đo xong → `experimental` + cảnh báo khi dùng.
- **Tài liệu song hành:** đụng hành vi nào thì cập nhật `HDSD` + `SKILL.md` ngay.

## Việc còn treo cần bạn quyết

- Pha 0 chạy Linux cần một máy/VM Linux để đo — bạn có sẵn môi trường đó không?
- `fleet` gom log: ghi ra file theo phiên, hay stream về dashboard, hay cả hai?
- Ngưỡng cảnh báo hạn mức lấy từ đâu (API provider có trả % còn lại không) — cần đo.
