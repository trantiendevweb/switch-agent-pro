# Switch-Agent-Pro

**Local-first control plane điều phối nhiều coding agent và nhiều AI API.**
Một binary, chạy native trên Windows và Linux, có dashboard quan sát realtime.

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
| Chạy cái gì | Claude Code, Codex CLI, Gemini CLI, Cursor | Anthropic, OpenAI, Gemini, Grok, DeepSeek, OpenAI-compatible |
| Xác thực bằng | credential của chính CLI đó | API key riêng |

Hai đường **dùng chung** Project · Task · Workspace · Flow · Scheduler · Event · Dashboard —
chỉ khác ở auth, protocol và cách agent/model được thực thi.

## Cài

Cần [Go](https://go.dev/dl) ≥ 1.23. Không cần quyền quản trị.

```powershell
git clone https://github.com/trantiendevweb/switch-agent-pro
cd switch-agent-pro
.\install\cai-dat.ps1        # Windows
```

```bash
./install/cai-dat.sh         # Linux (đang experimental — xem docs/DO-LUONG.md)
```

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

Địa chỉ hoá `provider:account`, nên `sagent phu` == `sagent claude:phu`.
Claude Code lấy **thư mục hiện tại** làm nơi làm việc — `cd` vào dự án rồi mới gọi.

### Chạy nhiều agent song song

```bash
sagent fleet claude:phu --copies 4 --worktree -- -p "sửa lỗi trong repo"
sagent status          # phiên nào đang chạy, PID, worktree/log ở đâu
sagent stop all        # dừng hết (giết cả cây tiến trình con)
sagent clean claude:phu   # gỡ worktree + xoá clone — an toàn, không xuyên junction
```

Hai lớp tách biệt, mỗi lớp giải một bài:

| Tách cái gì | Bằng cách nào | Nếu không có thì sao |
|---|---|---|
| **Cấu hình/token** | mỗi phiên một config dir riêng (`clone`) | N tiến trình đua ghi `.claude.json`, hỏng trust dialog |
| **File đang sửa** | mỗi phiên một **git worktree** + nhánh `sagent/<tên>-<n>` (`--worktree`) | 4 agent sửa đè file của nhau, kết quả không đoán được |

Worktree đặt **ngoài repo** (`~/.ai-accounts/.worktrees/…`) nên `git status` của bạn
không bị rác. `clean` gỡ worktree nhưng **giữ nhánh** — việc agent làm nằm trong đó.

Trạng thái lưu ở `~/.ai-accounts/state.db` (SQLite, migration có version), và
`status` luôn đối chiếu PID thật nên không bao giờ báo sống một phiên đã chết.

> Hai điều công cụ nói thẳng mỗi lần chạy `fleet`: N phiên trên một tài khoản
> **tiêu hạn mức gấp N**, và hành vi khi nhiều phiên **cùng refresh token thì chưa
> đo** — xem [`docs/DO-LUONG.md`](docs/DO-LUONG.md).

## Trạng thái thật

Không tô hồng:

| Hạng mục | Windows | Linux |
|---|---|---|
| Đổi/chạy tài khoản Claude, không đăng nhập lại | ✅ chạy thật | ⬜ chưa đo |
| Junction/symlink phần dùng chung, không cần admin | ✅ | ⬜ chưa đo |
| Xoá an toàn (không đụng dữ liệu gốc) | ✅ test + thật | ⬜ chưa đo |
| Bỏ phụ thuộc Python | ✅ | ✅ CI |
| Domain layer · SQLite · daemon · flow · đường API | ⬜ đang làm (Pha 1–2) | ⬜ |

**Đang bị chặn:** chưa có máy Linux để đo token nằm ở file hay keyring; chưa có API key
để verify đường API. Nhãn Linux và mọi provider ngoài Claude/Windows giữ `experimental`.

## Tài liệu

| File | Nội dung |
|---|---|
| [`docs/MASTER-PLAN.md`](docs/MASTER-PLAN.md) | **Lộ trình chính** — kiến trúc, 8 pha, DoD |
| [`docs/THIET-KE.md`](docs/THIET-KE.md) | Vì sao thiết kế như vậy |
| [`docs/DO-LUONG.md`](docs/DO-LUONG.md) | Báo cáo đo — cái gì đã chứng minh, cái gì chưa |
| [`docs/PLAN.md`](docs/PLAN.md) | Nhật ký thực thi Pha 1 (đã bị MASTER-PLAN thay thế) |
| `plan.html` · `master-plan.html` · `index.html` | Trang xem kế hoạch + nguyên mẫu dashboard 3D |

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
