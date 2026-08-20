# Switch-Agent-Pro

> ⚡ **Mới bắt đầu?** Xem ngay [**Hướng dẫn 5 phút đầu tiên (docs/BAT-DAU.md)**](docs/BAT-DAU.md) — dành riêng cho người không rành kỹ thuật (cách cài đặt 1 dòng, vượt cảnh báo Windows và 3 bước dùng ngay).

**Local-first control plane điều phối nhiều coding agent và nhiều AI API.**
Một file `.exe` **14,4 MB**, không phụ thuộc gì, chạy native trên **Windows**,
có dashboard quan sát realtime.

```
$ sagent ds

  Tài khoản AI trên máy này

  *  1  claude  phu          ban@gmail.com                      sẵn sàng
     2  claude  cong-ty      cty@congty.com                     sẵn sàng
```

> Trước đây dự án tên `ccswitch` — chỉ để đổi tài khoản Claude Code trên Windows.
> Nay mở rộng thành control plane cho cả đội AI, và đổi tên để không trùng
> [`farion1231/cc-switch`](https://github.com/farion1231/cc-switch).
> Bộ công cụ PowerShell v1 vẫn còn ở [`legacy/v1-powershell/`](legacy/v1-powershell/).

## Nó giải bài gì

Một máy, nhiều tài khoản AI, nhiều dự án. Bạn muốn:

- đổi qua lại giữa các tài khoản **không phải đăng nhập lại**;
- chạy **nhiều phiên song song** — nhiều tài khoản, hoặc nhiều phiên trên một tài khoản;
- dùng cả **CLI đăng nhập bằng gói cước** lẫn **API key gọi thẳng model**;
- và **nhìn thấy** cả đội đang làm gì.

## Nguyên lý cốt lõi

> Trỏ một CLI vào **một thư mục config biệt lập** qua **một biến môi trường**, và biết
> trong đó file nào là *token/danh tính* (riêng) còn file nào là *thói quen máy* (chung).

Với Claude Code: biến `CLAUDE_CONFIG_DIR`; riêng `.credentials.json` + `.claude.json`;
phần còn lại nối link về `~/.claude` nên sửa một chỗ mọi tài khoản thấy.

## Hai đường sử dụng

| | Subscription | API trực tiếp |
|---|---|---|
| Chạy cái gì | Claude Code, Codex CLI, Cursor CLI, Antigravity CLI, Grok CLI | Anthropic, OpenAI, Gemini, Grok, DeepSeek, OpenAI-compatible |
| Xác thực bằng | credential của chính CLI đó | API key riêng |

Hai đường **dùng chung** Project · Task · Workspace · Flow · Scheduler · Event · Dashboard —
chỉ khác ở auth, protocol và cách agent/model được thực thi.

## Cài

Một dòng. **Không cần Go, không cần quyền quản trị, không cần cài gì trước.**

```powershell
iex (irm https://raw.githubusercontent.com/trantiendevweb/switch-agent-pro/main/install/get.ps1)
```

> **Windows sẽ cảnh báo lần chạy đầu.** Binary **không ký số** — dự án mã nguồn mở, không
> mua chứng chỉ code-signing. SmartScreen xét chữ ký và độ phổ biến của file chứ không xét
> giấy phép, nên mở mã nguồn không gỡ được cảnh báo này. Thay vào đó mỗi bản phát hành có
> `SHA256SUMS.txt`: trình cài **tự đối chiếu băm**, và bạn kiểm lại tay được bất cứ lúc nào.

Nó tải một file `.exe` (~14,4 MB) từ GitHub Releases, **đối chiếu SHA256**, đặt vào
`%USERPROFILE%\bin` rồi thêm vào PATH của người dùng. Chỉ cần Windows 10 trở lên —
PowerShell 5.1 có sẵn là đủ. Có bản `amd64` và `arm64`, trình cài tự chọn.

Cài đúng một phiên bản, hoặc build từ nguồn (cần Go, phải đứng trong repo đã clone):

```powershell
.\install\cai-dat.ps1 -Phien v0.2.0
.\install\cai-dat.ps1 -TuNguon
```

Qua lệnh một dòng thì không truyền được tham số, nên dùng biến môi trường:

```powershell
$env:SAGENT_PHIEN = 'v0.2.0'
iex (irm https://raw.githubusercontent.com/trantiendevweb/switch-agent-pro/main/install/get.ps1)
```

Gỡ: xoá `%USERPROFILE%\bin\sagent.exe`. Dữ liệu nằm ở `~/.ai-accounts` — xoá riêng nếu muốn.

> **Chỉ Windows.** Nhánh Linux đã bị bỏ (2026-08-17). Mọi thứ khiến công cụ này đáng
> dùng đều là chi tiết Windows — junction thay symlink, ACL thay bit quyền, `taskkill`
> thay process group — và mọi phép đo trong [`docs/DO-LUONG.md`](docs/DO-LUONG.md) đều
> là phép đo Windows. Giữ nhánh Linux mà không có máy Linux để chạy thì đó không phải
> hỗ trợ, đó là lời hứa suông.

## Dùng

| Lệnh | Làm gì |
|---|---|
| `sagent` | Bảng tài khoản |
| `sagent <provider:tên>` | Chạy CLI bằng tài khoản đó (mặc định provider `claude`) |
| `sagent goc` | Chạy bằng tài khoản gốc |
| `sagent them <provider:tên>` | Tạo tài khoản mới rồi đăng nhập |
| `sagent dong-bo [--dry-run]` | Đồng bộ cấu hình dùng chung sang mọi tài khoản |
| `sagent xoa <provider:tên>` | Xoá tài khoản (an toàn) |
| `sagent verify [provider]` | Chạy bộ "đã đo" trên máy bạn |
| `sagent init` · `sagent config` | Tạo/xem cấu hình dự án |

Địa chỉ hoá `provider:account`, nên `sagent phu` == `sagent claude:phu`.
Claude Code lấy **thư mục hiện tại** làm nơi làm việc — `cd` vào dự án rồi mới gọi.

### Năm provider — dùng cái nào khi cái kia hết hạn mức

```powershell
sagent goc                                   # claude (tài khoản gốc)
sagent goc codex exec "việc cần làm"
sagent goc cursor --trust -p "việc cần làm"
sagent goc antigravity -p "việc cần làm"
sagent goc grok -m grok-4.5 -p "việc cần làm"
```

`sagent goc <provider>` chạy **tài khoản gốc** — tức tài khoản mà chính CLI đó dùng khi
không có `sagent`. Không cần tạo hồ sơ gì.

Tạo hồ sơ chỉ cần khi bạn có **nhiều tài khoản cùng một provider**:

```powershell
sagent them cursor:acc2      # tạo hồ sơ rỗng, rồi đăng nhập vào nó
sagent cursor:acc2           # chạy bằng acc2 — token riêng, không đè acc1
sagent ds                    # xem hết
```

Vài chỗ **phải biết trước**, đều là giới hạn của CLI bên dưới chứ không phải của `sagent`:

| Provider | Lưu ý |
|---|---|
| **antigravity** | **Một máy một tài khoản.** Token nằm trong Windows Credential Manager theo khoá cố định, không theo thư mục hồ sơ. `fleet --copies 2` sẽ từ chối. |
| **grok** | Phải truyền `-m <model>`. CLI **bỏ qua** `defaultModel` trong chính file cấu hình của nó và dùng `grok-code-fast-1` dựng sẵn; endpoint nào không bán model đó sẽ trả `503 No available channel`. |
| **cursor** | Cần `--trust` để chạy headless. Cố ý **không** dùng `--yolo`/`-f` — hai cờ đó nghĩa là *chạy mọi lệnh không hỏi*. |

`sagent verify` in ra đúng những giới hạn này cho từng provider, nên không phải nhớ.

### Chạy nhiều agent song song

```bash
sagent fleet claude:phu --copies 4 --worktree -- -p "sửa lỗi trong repo"
sagent status          # phiên nào đang chạy, PID, worktree/log ở đâu
sagent stop all        # dừng hết (giết cả cây tiến trình con)
sagent quet            # tìm tiến trình còn sống của phiên đã tự chết
sagent quet --giet     # ...và dừng chúng
sagent clean claude:phu   # gỡ worktree + xoá clone — an toàn, không xuyên junction
```

Hai lớp tách biệt, mỗi lớp giải một bài:

| Tách cái gì | Bằng cách nào | Nếu không có thì sao |
|---|---|---|
| **Cấu hình/token** | mỗi phiên một config dir riêng (`clone`) | N tiến trình đua ghi `.claude.json`, hỏng trust dialog |
| **File đang sửa** | mỗi phiên một **git worktree** + nhánh `sagent/<tên>-<n>` (`--worktree`) | 4 agent sửa đè file của nhau, kết quả không đoán được |

Worktree đặt **ngoài repo** (`~/.ai-accounts/.worktrees/…`) nên `git status` của bạn
không bị rác. `clean` gỡ worktree nhưng **giữ nhánh** — việc agent làm nằm trong đó —
và **từ chối gỡ** worktree còn thay đổi chưa commit (muốn bỏ thật thì thêm `--force`).

### Cấu hình theo từng dự án

```bash
sagent init      # tạo .sagent/project.toml cho repo hiện tại
sagent config    # xem cấu hình đã gộp + đọc từ file nào
```

Tầng cấu hình, dưới đè lên trên: **mặc định → `~/.ai-accounts/config.toml` →
`.sagent/project.toml` → cờ dòng lệnh**. Khoá không khai báo thì giữ giá trị tầng trên.

```toml
[project]
workspace = "worktree"        # fleet tự bật worktree, khỏi gõ --worktree
[policy]
max_parallel_sessions = 4     # trần cứng, gõ --copies 9 cũng bị hạ xuống 4
require_approval_for  = ["merge", "deploy"]
[ui]
default_surface = "tui"       # tui | dashboard | workflow | 3d
```

### Dashboard

Cửa vào **duy nhất** là form đăng nhập. Phải đặt mật khẩu trước, không thì server
từ chối chạy:

```bash
sagent dash --set-password        # hỏi tên đăng nhập + mật khẩu, lưu ĐÃ BĂM
                                  # ở ~/.ai-accounts/dash-auth.json (ngoài repo)
sagent dash                       # in ra URL, mở trên trình duyệt (cùng máy)
```

Dashboard 2D xem phiên đang chạy, bật/dừng hạm đội, nhật ký sự kiện realtime; bấm
nút **3D** để xem dạng sơ đồ không gian (orb = phiên thật, bấm orb để dừng đúng phiên đó)
hoặc bấm **Văn phòng** để xem mô phỏng văn phòng 3D theo phòng ban (robot đi lại, vẫy tay giao việc, bóng thoại realtime).
Mặc định chỉ nghe `127.0.0.1`, chặn Host/Origin lạ, DTO allowlist nên không gửi
token/API key ra trình duyệt. Assets nhúng trong binary (không cần Node).

Mật khẩu băm bằng PBKDF2-HMAC-SHA256 210k vòng; phiên giữ bằng cookie HttpOnly
SameSite=Lax, hết hạn sau 12 giờ và mất khi tắt server; sai nhiều lần thì bị bắt chờ.

> **Không còn token trong URL.** Trước đây `/?t=<token>` và header `X-Sagent-Token`
> cũng mở được dashboard — đã bỏ hẳn. Một secret nằm trong địa chỉ sẽ rơi vào log
> proxy, lịch sử trình duyệt và ảnh chụp màn hình. Script/curl bây giờ đăng nhập
> qua `POST /login` rồi giữ cookie (`curl -c/-b`).

**Xem từ máy khác / điện thoại** — khi bạn làm việc trên server không có màn hình:

```bash
sagent dash --host 0.0.0.0 --port 8787
```

Phơi ra mạng thì **HTTPS là mặc định** — chứng chỉ tự ký sinh tự động, phủ mọi IP của
máy. Vì tự ký nên trình duyệt sẽ cảnh báo; công cụ in **vân tay SHA-256** ra terminal để
bạn đối chiếu trong hộp thoại xem chứng chỉ trước khi bấm "vẫn tiếp tục":

```
    17:BD:F4:E5:10:1F:43:18:F5:63:63:7A:8C:53:46:00:AA:BE:A7:4D:...
```

> ⚠ **Không đối chiếu vân tay thì TLS chỉ chống nghe lén, không chống kẻ đứng giữa.**
> Và mật khẩu vẫn là hàng rào duy nhất: ai đoán được đều bật/dừng được agent của bạn và
> tiêu hạn mức. Đặt mật khẩu dài, đóng cổng khi xong.

Muốn HTTP trần (đã có SSH tunnel/VPN bọc ngoài) thì phải gõ thêm `--http-tran`; không gõ
thì server **từ chối chạy** chứ không lặng lẽ gửi mật khẩu dạng chữ thường. Kín hơn cả
là đừng phơi cổng: `ssh -L 4600:127.0.0.1:4600 user@server` rồi mở `http://127.0.0.1:4600`
ở máy bạn.

> File cấu hình **không bao giờ chứa secret** — API key/token chỉ được tham chiếu bằng ID.

Trạng thái lưu ở `~/.ai-accounts/state.db` (SQLite, migration có version), và
`status` luôn đối chiếu PID thật nên không bao giờ báo sống một phiên đã chết.

Phiên tự chết thì biến khỏi `status` — nhưng tiến trình con nó đẻ ra có thể **vẫn chạy
và vẫn tiêu hạn mức**. `sagent quet` là chỗ nhìn ra chúng. Mặc định chỉ liệt kê kèm tên
và thời điểm, không tự giết: Windows dùng lại PID nên danh sách có thể lẫn tiến trình
không liên quan, và bạn phải là người quyết.

> Hai điều công cụ nói thẳng mỗi lần chạy `fleet`: N phiên trên một tài khoản
> **tiêu hạn mức gấp N**, và hành vi khi nhiều phiên **cùng refresh token thì chưa
> đo** — xem [`docs/DO-LUONG.md`](docs/DO-LUONG.md).

### Sao lưu cơ sở dữ liệu

```bash
sagent db                      # đường dẫn, kích thước, schema, các bản sao lưu đang có
sagent db backup [file]        # chụp ảnh nhất quán (mặc định: state.db.bak-<ngày-giờ>)
sagent db restore <file>       # ghi đè — dừng dash và mọi phiên trước
```

Sao lưu bằng `VACUUM INTO` chứ **không chép file**: DB chạy ở chế độ WAL nên dữ liệu mới
nhất còn nằm trong `state.db-wal`; chép mình file chính ra được một bản thiếu mà trông
như đủ.

Ba thứ tự động, không phải bấm:

- **Nâng schema thì sao lưu trước** → `state.db.bak-v<cũ>`. Transaction chống được
  migration hỏng giữa chừng, không chống được migration chạy trót lọt mà sai ý.
- **Khôi phục thì cứu bản hiện tại trước** → `state.db.truoc-khi-khoi-phuc`. Một lệnh
  khôi phục không hoàn tác được thì chỉ là một lệnh xoá có thêm bước.
- **Binary cũ không mở được DB của binary mới.** Trước đây nó mở được, đọc được và **ghi
  được** trong im lặng; giờ nó từ chối và chỉ đường.

### Quyền truy cập kho hồ sơ

Kho `~/.ai-accounts` chứa token của từng hồ sơ và mật khẩu dashboard đã băm. Trên Windows,
`0o600` trong code Go **không bảo vệ gì** — quyền thật đến từ ACL kế thừa của thư mục cha
(đã đo: file `0o600` và `0o644` có ACL y hệt). `sagent` siết ACL tường minh và cắt kế thừa
khi tạo kho, và `sagent verify` nói cho bạn biết trạng thái thật:

```
  [kho hồ sơ]
    ✓ quyền truy cập ~/.ai-accounts    chỉ chủ sở hữu, SYSTEM và nhóm quản trị
```

Ô này ✗ nghĩa là token của bạn đang đọc được bởi người dùng khác trên máy.

`verify` cũng ghi lại **phiên bản CLI** của từng provider và báo động khi nó đổi:

```
  [codex]
    ✗ phiên bản CLI (provider drift)   CLI ĐÃ ĐỔI: codex-cli 0.147.0 → 0.200.0 …
```

Vì sao quan trọng: mọi khẳng định "đã đo" trong [`docs/DO-LUONG.md`](docs/DO-LUONG.md)
đều gắn với **một** phiên bản CLI. Nâng cấp CLI là số đo cũ hết hiệu lực. Cảnh báo **không
tự tắt** — đo lại xong thì `sagent verify --chap-nhan`.

## Trạng thái thật

Không tô hồng:

| Hạng mục | Trạng thái |
|---|---|
| Đổi/chạy tài khoản Claude, không đăng nhập lại | ✅ chạy thật |
| Junction phần dùng chung, không cần admin | ✅ |
| Xoá an toàn (không đụng dữ liệu gốc, không xuyên junction) | ✅ test + đã nổ thật một lần |
| Quyền truy cập kho token (ACL, không phải `0o600` giả) | ✅ |
| Dừng phiên kèm cả cây tiến trình con | ✅ |
| Sao lưu / khôi phục `state.db`, chặn hạ cấp binary | ✅ |
| Provider | ✅ **5**: claude · codex · cursor · antigravity · grok |
| Chạy nhiều tài khoản một provider | ✅ 4/5 — Antigravity **không** (token ở Credential Manager) |
| Đường API (nhiều AI API) | ⬜ chưa có API key để verify |
| TLS cho dashboard | ✅ HTTPS mặc định khi phơi ra mạng (chứng chỉ tự ký, in vân tay) |

**Đang bị chặn:** đường AI API (Pha 4) chưa làm — nhưng key Grok đã đo được nên hết
chặn về mặt dữ liệu.

> **Năm provider giấu danh tính ở năm chỗ khác nhau**, và ba trong số đó khác với thứ
> tài liệu của họ gợi ý. Bảng đo đầy đủ ở [`docs/DO-LUONG.md`](docs/DO-LUONG.md).

## Tài liệu

| File | Nội dung |
|---|---|
| [`docs/BAT-DAU.md`](docs/BAT-DAU.md) | **5 phút đầu tiên** — hướng dẫn bắt đầu nhanh cho người không rành kỹ thuật |
| [`docs/KHAC-PHUC-SU-CO.md`](docs/KHAC-PHUC-SU-CO.md) | **Sổ tay gỡ rối & khắc phục sự cố** — hướng dẫn xử lý các sự cố thực tế cho người vận hành |
| [`docs/MASTER-PLAN.md`](docs/MASTER-PLAN.md) | **Lộ trình chính** — kiến trúc, 8 pha, DoD |
| [`docs/THIET-KE.md`](docs/THIET-KE.md) | Vì sao thiết kế như vậy |
| [`docs/DO-LUONG.md`](docs/DO-LUONG.md) | Báo cáo đo — cái gì đã chứng minh, cái gì chưa |
| [`docs/PLAN.md`](docs/PLAN.md) | Nhật ký thực thi Pha 1 (đã bị MASTER-PLAN thay thế) |
| `plan.html` · `master-plan.html` · `index.html` | Trang xem kế hoạch + nguyên mẫu dashboard 3D |
| [`docs/DI-TRU-TU-V1.md`](docs/DI-TRU-TU-V1.md) | Di trú từ `tk` v1 — câu trả lời ngắn: không phải làm gì |

## Bốn nguyên tắc đã trả giá để có

1. **Whitelist, không blacklist.** Mai sau provider thêm khoá gói cước mới, blacklist sẽ
   lặng lẽ để nó lọt sang tài khoản khác.
2. **Xoá an toàn.** `RemoveAll` có thể xuyên junction xoá luôn dữ liệu thật — nên gỡ từng
   link, kiểm sạch, rồi mới xoá.
3. **Ghi nguyên tử.** File `.claude.json` chứa trust dialog; hỏng là bấm lại từ đầu.
4. **Đã đo — không suy luận.** `sagent verify` chạy lại các phép đo trên máy bạn.

## Một câu sòng phẳng

Công cụ chạy cục bộ, không gửi gì đi đâu, không đụng token ngoài việc để chúng ở các thư
mục khác nhau. Nhưng dùng nhiều tài khoản để vượt hạn mức nhiều khả năng đi ngược điều
khoản dịch vụ, và cái mất nếu bị phát hiện là tài khoản. Có những lý do dùng hoàn toàn
bình thường — tài khoản cá nhân và tài khoản công ty trên cùng một máy, nhiều nhà cung
cấp để so sánh. Bạn tự cân.

## Giấy phép

MIT. Xem [`LICENSE`](LICENSE).
