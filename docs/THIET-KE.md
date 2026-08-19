# Switch-Agent-Pro — Thiết kế (v2)

> Trạng thái: **bản thiết kế, chưa code.** Mục tiêu của tài liệu này là chốt
> kiến trúc trước khi viết dòng Go đầu tiên. Nó giữ nguyên linh hồn của v1
> (whitelist, "đã đo — không suy luận", xoá an toàn) và tổng quát hoá đúng
> *một* nguyên lý cốt lõi để phục vụ 4 mục tiêu mới.

---

## 1. Mục tiêu & phi mục tiêu

**Mục tiêu**

1. Quản lý nhiều **nhà cung cấp AI**: Claude Code, OpenAI Codex CLI, Gemini CLI, Cursor… (thứ tự thêm: Codex → Gemini → Cursor).
2. Chạy **nhiều phiên song song trên một tài khoản** (để phát triển agent).
3. Chạy **nhiều tài khoản song song** (nhiều nhà cung cấp cùng lúc, để phát triển agent).
4. Chạy trên **Windows + Linux**.
5. **★ Điểm mới quan trọng nhất:** biến công cụ thành một *primitive ghép được*.
   Người dùng tự dựng nhiều **flow** theo ý mình từ các "động từ" nguyên thuỷ,
   thay vì bị khoá cứng vào vài lệnh có sẵn.
6. **Giao diện theo dõi trực quan (UI/UX), ưu tiên 3D** — người dùng *nhìn thấy*
   đội tài khoản/phiên/agent đang chạy theo thời gian thực. Xem mục 8.

**Phi mục tiêu (cố ý bỏ)**

- **macOS.** Bỏ hẳn ở bản này. Hệ quả tốt: rủi ro lớn nhất — token nằm trong
  Keychain thay vì file — biến mất khỏi bàn (đó là chuyện của macOS).
- **Vượt hạn mức.** Công cụ không khuyến khích lách ToS; xem mục 12.
- **App native hoặc dịch vụ đám mây.** UI là web dashboard *cục bộ* do chính
  binary phục vụ — không phải app phải cài, không phải service online. Lõi vẫn
  là CLI; UI là lớp tuỳ chọn đọc cùng trạng thái mà CLI ghi.

---

## 2. Nguyên lý cốt lõi (thứ mọi mục tiêu đều dựa vào)

> Trỏ một CLI vào **một thư mục cấu hình biệt lập** qua **một biến môi trường**,
> và biết trong thư mục đó file nào là *token/danh tính* (riêng) còn file nào là
> *thói quen máy* (chung).

Với Claude Code (đã đo trên Windows, v1):

| Thành phần | Giá trị |
|---|---|
| Biến môi trường | `CLAUDE_CONFIG_DIR` |
| File riêng (token/danh tính) | `.credentials.json`, `.claude.json` |
| Phần dùng chung | mọi thứ còn lại → nối link về base |
| Khoá "thói quen máy" copy sang | 19 khoá whitelist trong `.claude.json` |

Cả 4 mục tiêu chỉ là **tổ hợp lại** primitive này. Đó là lý do kiến trúc v1 đã
có sẵn xương sống đúng — v2 chỉ tổng quát hoá và làm nó *ghép được*.

---

## 3. Mô hình khái niệm

Ba danh từ + một danh từ ghép:

- **Provider (nhà cung cấp)** — Claude / Codex / Gemini / Cursor. Mỗi provider
  là một *adapter* khai báo primitive ở mục 2 cho riêng nó.
- **Account (tài khoản)** — một danh tính đăng nhập của một provider. Có token.
- **Profile (hồ sơ)** = `(provider, account)` + một thư mục config biệt lập.
  Đây là đơn vị mà công cụ thao tác. Địa chỉ hoá: **`provider:account`**
  (ví dụ `claude:phu`, `codex:main`, `gemini:g1`). Thiếu provider thì mặc định
  `claude` — nên `tk phu` == `tk claude:phu` (tương thích ngược v1).
- **Flow (luồng)** — một *công thức* ghép các động từ nguyên thuỷ lại. Đây là
  mục tiêu 5. Xem mục 6.

Thư mục kho (tổng quát hoá từ `~/.claude-accounts/<name>`):

```
~/.ai-accounts/
  claude/
    phu/        ← 1 profile: config dir biệt lập của claude:phu
    goc-ref     ← (tuỳ chọn) con trỏ tới base để nối link
  codex/
    main/
  gemini/
    g1/
```

> Di trú v1: nếu thấy `~/.claude-accounts/*` cũ thì tự nhận là `claude/*`.

---

## 4. Giao diện Provider Adapter

Trái tim của tính đa-nhà-cung-cấp. Mỗi provider hiện thực đúng interface này.
Không có nhánh `if provider == "claude"` rải rác trong lõi.

```go
// Adapter khai báo primitive ở mục 2 cho một nhà cung cấp.
type Adapter interface {
    Name() string                 // "claude", "codex", "gemini", "cursor"

    // EnvVar là biến trỏ CLI vào thư mục config biệt lập.
    // claude → "CLAUDE_CONFIG_DIR", codex → "CODEX_HOME" (CẦN ĐO), ...
    EnvVar() string

    // Command là lệnh CLI để chạy ("claude", "codex", "gemini").
    Command() (string, error)

    // PrivateFiles là các file KHÔNG dùng chung (token + danh tính).
    // Mọi file/thư mục khác trong config dir sẽ được nối link về base.
    PrivateFiles() []string

    // SharedKeys: nếu provider dùng MỘT file config gộp (như .claude.json),
    // đây là whitelist khoá "thói quen máy" được copy sang. Trả nil nếu
    // provider không có kiểu file gộp đó.
    SharedKeys() []string

    // Identity đọc email/định danh để hiển thị trong bảng. Trả "" nếu
    // chưa đăng nhập.
    Identity(configDir string) string

    // HasToken cho biết profile đã đăng nhập chưa.
    HasToken(configDir string) bool

    // Verify là bộ "đã đo": chứng minh trên MÁY NÀY việc tách thư mục là
    // tách thật (CLI có đọc EnvVar, và một profile không thấy dữ liệu của
    // profile khác). Thoát != 0 nếu đỏ. Đây là điều kiện để bật một adapter.
    Verify() []Check
}
```

Ba thứ **mỗi adapter phải ĐO** trước khi được coi là dùng được (đúng tinh thần
"Bốn điều đã đo" của v1 — không suy luận):

1. CLI **có** đọc `EnvVar` không?
2. Đặt `EnvVar=X` thì CLI ghi token/config vào `X`, và ở `X` **không** thấy dữ
   liệu của tài khoản khác — *tách thật*, không phải tách trên giấy.
3. Token nằm trong **file** ở config dir (không phải keyring/Credential
   Manager). Trên Windows đã đo cho Claude. Trên **Linux cần đo lại** — một số
   CLI dùng libsecret/gnome-keyring.

> **Chưa đo xong = adapter ở trạng thái "thử nghiệm", in cảnh báo khi dùng.**

---

## 5. Các động từ nguyên thuỷ (verbs)

Đây là API. Cố ý nhỏ và trực giao để ghép được thành flow (mục 6).

| Verb | Ý nghĩa |
|---|---|
| `create <provider:account>` | Tạo config dir biệt lập + nối phần dùng chung + gieo whitelist |
| `link <profile>` | (Chạy lại) nối các mục "thói quen máy" từ base — junction(Win)/symlink(Linux) |
| `seed <profile>` | Gieo whitelist khoá vào file config gộp (chỉ provider có kiểu đó) |
| `run <profile> [-- args…]` | Đặt env → chạy CLI của provider với config dir của profile |
| `clone <profile> --into N` | **Chép credential** của 1 account ra N config dir tạm → nền cho mục tiêu 2 |
| `sync [provider]` | Đẩy whitelist "thói quen máy" sang mọi profile (v1: `dong-bo`) |
| `list` / `status` | Bảng: profile nào, email, có token chưa, cái nào đang chạy |
| `remove <profile>` | Xoá an toàn: gỡ từng link trước, kiểm sạch, rồi mới xoá phần còn lại |
| `verify [provider]` | Chạy bộ "đã đo" của adapter |
| `fleet <flow>` | Điều phối: chạy nhiều `run` song song theo một flow, theo dõi trạng thái |

Địa chỉ hoá thống nhất `provider:account` cho mọi verb. `tk` (không tham số)
vẫn mở bảng chọn tương tác như v1.

---

## 6. ★ Flow — cách người dùng tự ghép (mục tiêu 5)

Nguyên tắc thiết kế: **verb là API, flow là công thức.** Ba tầng, người dùng
chọn tầng hợp với mình:

**Tầng 1 — ghép bằng chính CLI (không cần học gì mới).** Verb trực giao nên
ghép được trong shell/script bất kỳ:

```bash
# fan-out 4 phiên trên 1 tài khoản, mỗi phiên 1 git worktree
tk clone claude:phu --into 4 | while read p; do tk run "$p" & done
```

**Tầng 2 — flow khai báo trong file (không cần viết script).** File
`~/.ai-accounts/flows.toml`. Người dùng đặt tên flow riêng, `tk fleet <tên>`:

```toml
# Chạy N phiên song song trên MỘT tài khoản Claude để phát triển agent.
[flow.fanout]
desc     = "4 phiên Claude song song, mỗi phiên 1 worktree"
clone    = "claude:phu"     # chép credential từ đây...
copies   = 4                # ...ra 4 config dir tạm
worktree = true             # mỗi phiên 1 git worktree riêng (tránh giẫm file)
run      = "claude"

# Nhiều nhà cung cấp cùng làm một việc (mục tiêu 3).
[flow.squad]
desc     = "3 nhà cung cấp song song trên cùng một task"
parallel = ["claude:phu", "codex:main", "gemini:g1"]

# Đội agent headless (không tương tác), gom log.
[flow.agents]
desc     = "6 agent headless đọc task từ hàng đợi"
clone    = "claude:phu"
copies   = 6
run      = "claude -p"      # chế độ print/headless
```

**Tầng 3 — plugin adapter.** Người dùng thêm provider mới bằng cách thả một
file mô tả (TOML) khai báo `env_var`, `private_files`, `shared_keys`, lệnh đọc
identity — không cần build lại binary cho provider "dễ". Provider "khó" (cần
logic riêng) thì viết adapter Go.

> Đây là chỗ trả lời trực tiếp yêu cầu của bạn: *"từ cái này có thể build thành
> nhiều flow theo ý muốn của người sử dụng."* Primitive nhỏ + 3 tầng ghép =
> người dùng tự dựng luồng, công cụ không đoán trước hộ họ.

---

## 7. Hiện thực 3 mục tiêu song song

**Mục tiêu 3 — nhiều tài khoản song song.** Gần như đã có ở v1: env đặt trong
env của *chính process con*, nên mỗi terminal chạy `tk run <profile>` là một
phiên độc lập. v2 thêm `fleet` để *khởi chạy + theo dõi* N phiên từ một chỗ,
và bảng `status` cho thấy profile nào đang chạy (PID registry ở
`~/.ai-accounts/running.json`).

**Mục tiêu 2 — nhiều phiên trên một tài khoản.** Bẫy phải né: hai process cùng
trỏ *một* config dir sẽ **đua ghi `.claude.json`** → hỏng. Nên cơ chế đúng là
`clone`:

- Chép `.credentials.json` (token, dùng chung được — cùng một account) ra N
  config dir tạm.
- Mỗi config dir có `.claude.json` **riêng** → không giành nhau ghi.
- Nối phần dùng chung như thường; `sync` để trust dialog đồng nhất.
- **Cảnh báo hạn mức:** N phiên cùng một account = tiêu hạn mức gấp N. `fleet`
  in cảnh báo này rõ ràng, không im lặng.

**Mục tiêu 2+3 kết hợp — đội agent.** `run "<cli> -p …"` chạy chế độ headless;
`fleet` cấp cho mỗi agent một profile + gom log. Đây là nền để "phát triển
agent".

---

## 8. Giao diện theo dõi trực quan (UI/UX + 3D) — mục tiêu 6

Yêu cầu: người dùng phải *nhìn thấy* đội tài khoản/phiên/agent đang chạy, trực
quan, thời gian thực, có 3D càng tốt.

**Hình thức: web dashboard cục bộ do chính binary Go phục vụ.** Không phá vỡ
triết lý "một binary":

- Assets nhúng bằng **Go `embed`** → vẫn đúng một file, không cần cài Node để chạy.
- `tk dash` bật server `localhost` rồi mở trình duyệt.
- Cập nhật **thời gian thực** qua WebSocket: đọc registry PID + hạn mức/hoạt động
  từ đúng nơi `fleet` ghi (`~/.ai-accounts/running.json`).
- **3D bằng Three.js / React Three Fiber** trong trình duyệt — full sức mạnh web,
  không cần native. Chạy Windows+Linux như nhau vì chỉ cần trình duyệt.

**Vì sao không TUI / không app native:** TUI (Bubble Tea) nhanh nhưng không có
3D; app native phá "một binary" và nặng. Web cục bộ được cả 3D lẫn gọn.

**Hệ thiết kế (Design System):**

Toàn bộ giao diện dashboard của Switch-Agent-Pro (cả màn 2D lẫn không gian 3D)
tuân thủ triết lý tối giản, kỹ thuật, rõ ràng và hỗ trợ người vận hành nắm bắt
trạng thái tức thì (*glanceable* — nhìn phát hiểu ngay). Mọi trang đều nạp chung
một file token duy nhất tại `internal/dash/web/vendor/token.css`.

#### 1. Bảng Design Tokens chuẩn

| Nhóm | Biến CSS | Giá trị màu | Ý nghĩa & Vùng áp dụng |
|---|---|---|---|
| **Nền (Background)** | `--void` | `#070810` | Nền sâu nhất, canvas 3D, nền viewport |
| | `--panel` | `#0C0E17` | Nền thẻ (card), thanh điều khiển, panel chính |
| | `--panel-2` | `#11141F` | Nền panel nổi cấp 2, dropdown, modal, hover |
| **Hairline (Viền mờ)** | `--line` | `rgba(255,255,255,.06)` | Đường kẻ phân tách nhẹ, viền panel mặc định |
| | `--line-2` | `rgba(255,255,255,.12)` | Viền khi hover, viền phân cách nổi bật |
| **Chữ (Typography)** | `--hi` | `#EAECF4` | Chữ chính (độ tương phản cao, tiêu đề, nội dung quan trọng) |
| | `--mid` | `#969DB0` | Chữ phụ (mô tả bổ trợ, nhãn phụ) |
| | `--lo` | `#565D71` | Nhãn mờ, chú thích, thông tin nền ít quan trọng |
| **Trạng thái agent (Status)** | `--run` | `#2FE0A0` | Đang chạy (Active / Running) — orb phát sáng, viền pulse xanh ngọc |
| | `--pending` | `#F6B23D` | Đang chờ / xử lý (Pending / Queued) — chấm cam hổ phách |
| | `--done` | `#6E8CFF` | Hoàn thành (Done / Success) — xanh dương dịu |
| | `--idle` | `#5A6273` | Nghỉ / rảnh rỗi (Idle / Standby) — xám trầm |
| | `--error` | `#FF5D79` | Lỗi / cạn hạn mức (Error / Failed / Limit) — đỏ san hô cảnh báo |
| **Hạ tầng kết nối (Core & Link)** | `--core` | `#8FB2FF` | Lõi điều phối trung tâm (Orchestrator core wireframe / halo) |
| | `--link` | `#39D9E0` | Đường nối kết nối (Beam / link dữ liệu giữa Core và Agent) |

#### 2. Ba vai chữ (Typography Hierarchy)

Ba font chữ được vendor offline trong `internal/dash/web/vendor/` dưới định dạng `.woff2`, phục vụ 3 vai trò tách bạch:

- **Display / Heading (`Space Grotesk`):** Dùng cho tiêu đề lớn, tên màn hình và nhãn thương hiệu. Có cá tính công nghệ cao nhưng được sử dụng tiết chế, không lạm dụng ở các đoạn văn dài.
- **Body / UI (`Inter`):** Dùng cho toàn bộ văn bản giao diện thông thường, nhãn nút bấm, câu hướng dẫn và các khối nội dung người đọc. Dễ đọc, sạch và trung tính.
- **Data / Machine (`JetBrains Mono`):** Bắt buộc áp dụng cho **MỌI dữ liệu do máy sinh ra**:
  - Định danh tài khoản `provider:account` (ví dụ `claude:phu`, `codex:main`)
  - Số lượng token tiêu thụ
  - Chi phí (`cost`, ví dụ `$0.042`)
  - Mã tiến trình hệ thống (`PID: 12480`)
  - Mốc thời gian & thời lượng (`timestamp`, `elapsed`: `12s`, `02:45`)
  - Dòng lệnh và nhật ký sự kiện (`event log`)
  *(Lý do: Monospace căn thẳng hàng các cột số liệu, phân biệt rõ từng ký tự dễ nhầm lẫn như 0/O, 1/l, giúp người vận hành quét thông số tức thì mà không nhầm.)*

#### 3. Luật màu-chỉ-cho-trạng-thái (Color Only for Status)

- **Quy tắc:** Màu sắc rực rỡ chỉ dành riêng cho **trạng thái của agent** (`--run`, `--pending`, `--done`, `--idle`, `--error`).
- **Khung giao diện (Chrome):** Topbar, panel, lưới bảng, viền khung và nút bấm đều phải giữ **đơn sắc** trung tính (`--line`, `--line-2`, `--hi`, `--mid`, `--lo`, `--panel`).
- **Lý do thiết kế:**
  - *Dồn điểm nhấn vào một chỗ (Glanceable):* Mục tiêu số một của dashboard là giúp người vận hành chỉ cần liếc mắt trong 1 giây là biết ngay hạm đội agent đang làm gì (ai đang chạy, ai lỗi, ai cạn quota).
  - *Chống Anti-pattern số 1:* Rải màu ra khung, viền và các thành phần tĩnh là lỗi phổ biến nhất làm loãng giao diện, biến màn hình thành "cây thông Noel" gây nhiễu thị giác và làm mất khả năng cảnh báo sớm khi có sự cố.

#### 4. Lý do một file token dùng chung cho cả 2D lẫn 3D

- **Triết lý "Agent = Ánh sáng sống":**
  - Trong **3D (Không gian)**: Mỗi agent là một quả cầu ánh sáng (orb) phát sáng và xung nhịp theo màu status (`--run`, `--pending`, `--error`), nối với lõi điều phối (`--core`) qua luồng hạt (`--link`).
  - Trong **2D (Bảng điều khiển bento)**: Đúng ánh sáng đó được chuyển thể thành thanh ray chỉ báo (rail glow), quầng aura quanh thẻ và chấm nhịp tim (pulse) trên card agent.
- **Lợi ích kiến trúc:**
  1. *Đồng nhất tuyệt đối:* Trải nghiệm chuyển đổi qua lại giữa 2D và 3D liền mạch, cùng một ngôn ngữ thị giác của MỘT sản phẩm duy nhất.
  2. *Một nguồn sự thật (Single Source of Truth):* Toàn bộ token màu sắc, typography và `@font-face` nằm gọn trong `vendor/token.css`. Mọi trang (`index.html`, `flow.html`, `hoi-thoai.html`, `3d.html`) chỉ cần `<link>` vào là dùng được ngay.
  3. *Tương thích nhúng offline:* Không cần build tool hay preprocessor, phục vụ trực tiếp từ Go `embed` tĩnh.

#### 5. Mặt 3D (Kiến trúc không gian & Ba quyết định cốt lõi)

Không gian theo dõi 3D (`3d.html`) đóng vai trò là "phòng điều khiển trực quan" cho toàn bộ hạm đội agent. Người vận hành chỉ cần mở màn hình là nắm bắt được ngay trạng thái vận hành của từng phiên làm việc, dòng chảy dữ liệu giữa các agent và tiến độ thực thi theo thời gian thực.

Để đảm bảo hiệu năng cao nhất, giao diện sắc nét và khả năng hoạt động độc lập 100% offline trong một binary duy nhất, kiến trúc 3D được xây dựng dựa trên ba quyết định thiết kế cốt lõi:

- **Quyết định 1: Xếp agent trên một Ring (vòng tròn) quanh Core điều phối thay vì thả trôi tự do**
  - *Vấn đề của bản cũ (Lỗi trực quan số 1):* Trong các bản phác thảo đầu tiên, các agent và quả cầu phiên (orb) được thả trôi lơ lửng ngẫu nhiên quanh không gian. Điều này tạo ra trải nghiệm thị giác rất rối rắm: mắt người dùng không thể xác định được trật tự trước - sau, không đếm nhanh được số lượng phiên đang chạy, và khi xoay góc nhìn camera thì các đối tượng che khuất lẫn nhau khiến việc theo dõi bị gián đoạn.
  - *Giải pháp & Lợi ích:* Tất cả các agent được sắp xếp cách đều nhau trên **một vòng tròn cố định (Ring)** bao quanh khối đa diện lõi điều phối (`Core` icosahedron nằm ở trung tâm $y = 1.1$). Bố cục hình học tròn mang lại trật tự trực quan rõ ràng:
    - *Quét nhanh trong một cái nhìn:* Mắt người vận hành chỉ cần quét một vòng cung là nhận diện được toàn bộ các agent, màu trạng thái chân đế (`--run` xanh ngọc, `--pending` vàng cam, `--done` xanh dịu, `--error` đỏ) và vai trò của từng thành viên.
    - *Đường truyền dữ liệu mạch lạc:* Tia kết nối (beam) và các luồng hạt tín hiệu (particle) di chuyển giữa Core và các agent trên Ring hoặc giữa các bước phụ thuộc (`needs`) theo những đường cong xác định, không bị chồng chéo hay rối mắt.

- **Quyết định 2: Quầng sáng phát quang bằng Glow Sprite Additive thay vì Post-Processing (Bloom / EffectComposer)**
  - *Ràng buộc kiến trúc sống còn (Offline & Một file core duy nhất):*
    - Switch-Agent-Pro tuân thủ tiêu chuẩn đóng gói nhúng tĩnh trong Go (`go:embed`), chạy hoàn toàn cục bộ (offline 100%) mà không phụ thuộc vào bất kỳ kết nối mạng hay CDN bên ngoài nào.
    - Thư viện Three.js được nhúng duy nhất **một file lõi** (`three.min.js` r128). Các giải pháp hậu kỳ hình ảnh như `EffectComposer`, `UnrealBloomPass` hay `ShaderPass` là các addon mở rộng rời rạc. Sử dụng chúng sẽ kéo theo nhiều file JavaScript phụ thuộc, làm tăng độ phức tạp và tiềm ẩn nguy cơ "màn hình trắng trơn" khi chạy trên môi trường máy chủ nội bộ bị chặn Internet.
    - Ngoài ra, kỹ thuật hậu kỳ Bloom (post-processing) yêu cầu GPU phải xử lý render nhiều lượt (multi-pass), gây nóng máy và tụt khung hình trên các thiết bị hoặc máy ảo không có card đồ hoạ rời.
  - *Giải pháp & Lợi ích:* Sử dụng `THREE.Sprite` kết hợp với chế độ trộn màu cộng hưởng (`THREE.AdditiveBlending`) và ảnh quầng sáng (radial gradient) sinh trực tiếp trong bộ nhớ lúc khởi chạy. Kỹ thuật này tạo ra quầng sáng aura rực rỡ, mềm mại bao quanh các orb khi agent hoạt động mà hoàn toàn:
    - *Chỉ dùng Three.js core:* Không cần thêm bất kỳ file addon nào, an toàn tuyệt đối khi phân phối offline.
    - *Siêu nhẹ và mượt mà:* Chi phí dựng hình cực thấp, đảm bảo tốc độ khung hình 60 FPS ổn định trên mọi cấu hình máy tính.

- **Quyết định 3: Nhãn thông tin (Label) bằng HTML Overlay chiếu World → Screen thay vì 3D Text**
  - *Nhược điểm của 3D Text / Texture Canvas trong 3D:*
    - Văn bản tạo bằng 3D mesh (`TextGeometry`) hoặc vẽ lên Canvas texture rồi dán lên mặt phẳng 3D (billboard) thường xuyên bị hiện tượng nhoè chữ, răng cưa vỡ hạt khi phóng to, thu nhỏ hoặc nhìn ở góc nghiêng.
    - Dựng chữ 3D đòi hỏi phải tải thêm file font 3D vector JSON nặng nề, tốn bộ nhớ hình học, và không hỗ trợ các tính năng định dạng giao diện hiện đại (như đổ bóng viền chữ, căn chỉnh linh hoạt nhiều dòng).
  - *Giải pháp & Lợi ích:* Nhãn thông tin được xây dựng bằng các thẻ `<div>` HTML phẳng nằm đè (overlay) phía trên canvas WebGL. Mỗi khung hình, toạ độ 3D của agent được chiếu thành toạ độ 2D trên màn hình thông qua hàm chiếu `vector.project(camera)`. Nhãn sẽ tự động ẩn đi nếu agent nằm khuất phía sau góc nhìn camera ($z \ge 1$).
    - *Chữ sắc nét 100%:* Được render bằng bộ engine font chữ tự nhiên của trình duyệt, nét căng tuyệt đối trên mọi độ phân giải và màn hình mật độ điểm ảnh cao (Retina / 4K).
    - *Đồng bộ Design Tokens:* Nhãn sử dụng chuẩn font kỹ thuật `JetBrains Mono` (`--font-mono`), cỡ chữ, màu tương phản (`--hi`, `--mid`) và màu trạng thái thừa hưởng trực tiếp từ file `vendor/token.css`, đồng nhất hoàn toàn với giao diện 2D.
    - *Hiển thị chi tiết đa tầng:* Dễ dàng định dạng cấu trúc nhãn đa thông tin gồm: định danh tài khoản (`provider:account`), vai trò điều phối (`giao việc`, `chốt`, `nhận rồi giao`) và tên bước công việc đang thực thi.

---

**Concept 3D — "phòng điều khiển đội agent":**

- Mỗi **provider** = một cụm (Claude/Codex/Gemini/Cursor), tách vùng trong không
  gian, có **linh vật (mascot) riêng** ở tâm cụm — robot tô màu thương hiệu + dấu
  nhận diện (Claude spark, Codex lục giác, Gemini sao 4 cánh, Cursor con trỏ).
  Mascot cuối cùng có thể thay bằng nhân vật minh hoạ/3D riêng cho từng AI.
- Mỗi **profile/phiên** = một orb xếp đều trên một ring quanh core; đập/sáng theo
  hoạt động; màu theo trạng thái (`--run`, `--pending`, `--done`, `--idle`, `--error`).
- **Hạn mức/token** = kích thước·độ cao orb hoặc gauge — thấy ngay phiên nào sắp cạn quota.
- **Kết nối** = đường nối orchestrator ↔ các phiên (`--link`); sáng lên và có hạt dữ liệu khi có luồng.
- **Fallback 2D** cho máy yếu / prefers-reduced-motion: bảng + sparkline hạn mức,
  cùng token màu trên.

**Ba quy tắc `threejs` (từ skill) định hình kiến trúc render — không phải trang trí:**

- **`InstancedMesh` cho đội orb:** 50+ agent cùng geometry/material → **một** draw
  call thay vì N. Đây là điều kiện để dashboard không khựng khi fleet đông.
- **`FogExp2`** (`0x070810`, density ~0.02): chiều sâu khí quyển + cull ngầm orb ở xa.
- **`ACESFilmicToneMapping` + `sRGBEncoding`**, và **`antialias:true` đặt LÚC KHỞI
  TẠO** `WebGLRenderer` (gán sau vô tác dụng — bẫy đã ghi trong skill).
- Tôn trọng **`prefers-reduced-motion`**: tắt đập/oscillation, dữ liệu vẫn đọc được ngay.

**Tách lớp:** lõi CLI **không** phụ thuộc UI. UI chỉ *đọc* trạng thái từ cùng
registry mà `fleet` ghi → tắt UI thì mọi thứ vẫn chạy; UI hỏng không kéo theo
lõi. Đây là điều kiện để giữ tinh thần CLI-first.

**Về skill:** "UXUI" chính là skill **`ui-ux-pro-max`** và **`sagent-dashboard`**
(bộ token chuẩn đã lưu trong `internal/dash/web/vendor/token.css` và `docs/THIET-KE.md`).
Quy trình dùng khi dựng UI: đọc `docs/THIET-KE.md` và `token.css` trước khi chỉnh sửa bất kỳ trang nào.

---

## 9. Kiến trúc Go & chuyện đa nền tảng

**Một binary tĩnh mỗi OS, không cần Python/Node/PowerShell.** Bỏ luôn lý do
tồn tại của `cfg.py`: Go `encoding/json` xử lý **khoá JSON trùng hoa/thường**
gọn (lấy khoá cuối, không ném lỗi) — đúng cái làm PowerShell 5.1 chết. Một
trong ba phụ thuộc của v1 tự tan.

```
switch-agent-pro/             (binary tên: sagent)
  cmd/sagent/main.go          điểm vào, phân giải verb
  internal/
    profile/                  create / link / seed / remove (xoá an toàn)
    provider/
      adapter.go              interface ở mục 4
      claude.go               adapter Claude (port từ v1)
      codex.go                adapter Codex   (sau khi ĐO CODEX_HOME)
      gemini.go               adapter Gemini  (sau khi ĐO)
      cursor.go               adapter Cursor  (sau khi ĐO)
      plugin.go               nạp adapter khai báo từ TOML (tầng 3)
    flow/                     đọc flows.toml, fleet, registry PID
    dash/
      server.go               server localhost + WebSocket, đọc running.json
      web/                     assets nhúng (Go embed): React Three Fiber + fallback 2D
    link/
      link_windows.go         junction qua `cmd /c mklink /J` (không cần admin)
      link_linux.go           os.Symlink (native, không cần quyền)
    jsonutil/                 đọc/ghi .claude.json, whitelist, ghi nguyên tử
    verify/                   bộ "đã đo" per-OS + per-provider
```

**Trừu tượng `link`** là điểm khác biệt OS duy nhất đáng kể, cô lập bằng build
tags:

| | Windows | Linux |
|---|---|---|
| Nối thư mục dùng chung | junction `mklink /J` (không đòi admin) | `os.Symlink` (native) |
| Token ở đâu | file `.credentials.json` (đã đo) | **cần đo** — có thể là keyring |
| Khoá JSON trùng hoa/thường | Go xử lý gọn | Go xử lý gọn |

**Giữ nguyên linh hồn v1 trong Go:**

- **Whitelist, không blacklist** — mai sau provider thêm khoá gói cước mới thì
  nó *không* tự lọt sang tài khoản khác.
- **Xoá an toàn** — `remove` gỡ từng link trước, kiểm không còn reparse point
  nào, rồi mới xoá đệ quy. (Bẫy "Remove-Item xuyên junction" của v1 vẫn đúng
  trên Windows; Go `os.RemoveAll` cũng cần cùng sự cẩn trọng với junction.)
- **Ghi nguyên tử** — ghi file tạm rồi `os.Rename`, kèm sao lưu `.bak-<ts>`.
- **Không bàn phím thì in trợ giúp**, không treo.

---

## 10. Việc phải ĐO trước khi cam kết (checklist)

Đúng phương châm repo. Không tick xong ô nào thì adapter/OS đó còn là "thử
nghiệm".

- [ ] **Linux / Claude:** token nằm ở file trong config dir hay ở
      libsecret/gnome-keyring? (quyết định primitive có đứng vững trên Linux)
- [ ] **Codex CLI:** biến config dir đúng là `CODEX_HOME`? File token/identity
      tên gì? Tách thật không?
- [ ] **Gemini CLI:** cơ chế thư mục config? (`~/.gemini/`? có biến override
      không?) Token ở đâu?
- [ ] **Cursor:** có CLI + cơ chế config dir không, hay phải cách khác?
- [ ] **Windows junction từ Go:** tạo được không cần admin bằng cách nào (shell
      `mklink /J` hay syscall trực tiếp)?

Mỗi ô = một mục trong `verify/`, chạy được, thoát != 0 nếu đỏ — để người dùng
tự đo lại trên máy họ như `kiem-tra.ps1` hiện nay.

---

## 11. Lộ trình theo pha

| Pha | Nội dung | Ra được gì |
|---|---|---|
| **0. Đo** | Chạy checklist mục 10 cho Windows+Linux, Claude trước | Biết primitive có đứng vững không |
| **1. Lõi Go + Claude** | Port v1 sang Go: create/run/sync/remove/list + adapter Claude + `link` 2-OS | Thay `tk` cũ, chạy Windows+Linux, bỏ Python |
| **2. Song song** | `clone`, `fleet`, registry PID, cảnh báo hạn mức | Mục tiêu 2 & 3 |
| **3. Flow** | `flows.toml` (tầng 2) + adapter plugin TOML (tầng 3) | Mục tiêu 5 |
| **3b. UI dashboard** | `tk dash`: server localhost + WebSocket + R3F, đọc registry của `fleet`; màu/motion theo `MASTER.md`; fallback 2D | Mục tiêu 6 |
| **4. Codex** | Adapter + verify Codex | Nhà cung cấp thứ 2 |
| **5. Gemini** | Adapter + verify Gemini | Nhà cung cấp thứ 3 |
| **6. Cursor** | Adapter + verify Cursor | Nhà cung cấp thứ 4 |

> Mỗi pha ra một binary dùng được. Pha 1 đã là bản thay thế hoàn chỉnh cho v1.

---

## 12. Rủi ro & một câu sòng phẳng

- **Hạn mức khi song song:** N phiên trên 1 account tiêu hạn mức gấp N. Công cụ
  cảnh báo, không giấu.
- **ToS:** dùng nhiều tài khoản để vượt hạn mức nhiều khả năng đi ngược điều
  khoản của nhà cung cấp; cái mất nếu bị phát hiện là tài khoản. Có lý do dùng
  hoàn toàn bình thường (tài khoản cá nhân + công ty trên một máy; nhiều
  provider để so sánh). Bạn tự cân — ghi lại cho rõ, như v1.
- **Cơ chế provider đổi:** các CLI này còn thay đổi nhanh. Vì vậy adapter tách
  rời + `verify` chạy được là bắt buộc: đo lại rẻ hơn đoán sai.
- **Bỏ macOS** là quyết định có ý thức của bản này; mở lại sau sẽ phải giải bài
  Keychain riêng.

---

## 13. Quyết định đã chốt

- Ngôn ngữ lõi: **Go, một binary**, bỏ phụ thuộc Python/PowerShell.
- Nền tảng: **Windows + Linux**. Bỏ macOS.
- Provider theo thứ tự: **Claude (có) → Codex → Gemini → Cursor**.
- Primitive nhỏ + 3 tầng ghép flow, để **người dùng tự dựng luồng**.
- Giữ nguyên: whitelist, "đã đo — không suy luận", xoá an toàn, ghi nguyên tử.
- **UI: web dashboard cục bộ** do binary phục vụ (Go embed), 3D bằng React Three
  Fiber, fallback 2D. Không app native, không đám mây.
- **Skill UI = `ui-ux-pro-max`** (bạn đã gửi). Hệ thiết kế đã sinh & lưu ở
  `design-system/switch-agent-pro/MASTER.md`. Đề xuất cài thành skill dùng
  chung ở `~/.claude/skills/` (chờ bạn đồng ý — xem câu hỏi cuối).
