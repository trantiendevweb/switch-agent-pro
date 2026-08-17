# Sổ mã nguồn mở

> Cam kết ở `docs/MASTER-PLAN.md` mục 0: mọi phụ thuộc phải là mã nguồn mở, giấy
> phép tương thích MIT, và ghi lại ở đây. Ưu tiên thư viện chuẩn — thêm dependency
> phải có lý do không né được.
>
> Cập nhật: 2026-08-17.
>
> **Bảng dưới là phần "vì sao", viết tay. Phần "cái gì" thì KHÔNG:** danh sách
> phụ thuộc và toàn văn giấy phép nằm ở [`THONG-BAO-GIAY-PHEP.txt`](../THONG-BAO-GIAY-PHEP.txt),
> sinh tự động bằng `go run ./tools/giayphep` từ `go list -deps ./cmd/sagent` —
> tức những module THẬT SỰ đi vào binary. CI chạy `-kiem` nên file đó không trôi được.
>
> Vì sao tách: bản viết tay của tài liệu này **đã trôi thật** — nó từng kê
> `github.com/google/uuid` (có trong `go.mod` nhưng KHÔNG được liên kết) và vẫn
> xếp `golang.org/x/sys` vào bảng gián tiếp sau khi nó đã thành trực tiếp.

## Phụ thuộc trực tiếp

| Gói | Phiên bản | Giấy phép | Vì sao cần | Vì sao không dùng stdlib |
|---|---|---|---|---|
| [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) | v1.56.0 | BSD-3-Clause | Nơi giữ trạng thái phiên (`~/.ai-accounts/state.db`) | Go không có SQLite trong stdlib. Chọn bản **thuần Go** thay vì `mattn/go-sqlite3` để **không cần cgo** — nhờ vậy `.exe` không phụ thuộc DLL nào, chép sang máy khác là chạy |
| [`github.com/BurntSushi/toml`](https://github.com/BurntSushi/toml) | v1.6.0 | MIT | Đọc `.sagent/project.toml`, `config.toml`, `flows.toml` | Go không có TOML trong stdlib. Chọn TOML thay JSON vì file này người dùng **sửa tay** và cần chú thích |
| [`golang.org/x/sys`](https://cs.opensource.google/go/x/sys) | v0.47.0 | BSD-3-Clause | Gọi thẳng Win32: `SetNamedSecurityInfo` (siết ACL kho token) và `GetProcessTimes` (chống PID dùng lại khi quét mồ côi) | `syscall` trong stdlib không có hai API này. Cách còn lại là shell ra `icacls`/`wmic` — mà `wmic` **đã bị gỡ khỏi Windows Server 2022** và tên nhóm ACL thì đổi theo ngôn ngữ hệ thống |

## Phụ thuộc gián tiếp

Kéo theo bởi `modernc.org/sqlite`, không gọi trực tiếp:

| Gói | Giấy phép |
|---|---|
| `modernc.org/libc`, `modernc.org/mathutil`, `modernc.org/memory` | BSD-3-Clause |
| `github.com/dustin/go-humanize` | MIT |
| `github.com/mattn/go-isatty` | MIT |
| `github.com/ncruces/go-strftime` | BSD-3-Clause |
| `github.com/remyoudompheng/bigfft` | BSD-3-Clause |

Toàn bộ là BSD-3-Clause hoặc MIT — **tương thích với giấy phép MIT** của dự án,
chỉ yêu cầu giữ thông báo bản quyền khi phát hành bản build. Đó là việc mà
`THONG-BAO-GIAY-PHEP.txt` làm, và nó được đính kèm mọi bản phát hành.

> `github.com/google/uuid` từng nằm trong bảng này. Nó có trong `go.mod` nhưng
> **không** được liên kết vào binary — đúng kiểu sai mà một sổ viết tay hay mắc,
> và là lý do phần "cái gì" giờ do máy sinh.

## SBOM

Mỗi bản phát hành kèm `sbom.cdx.json` (CycloneDX 1.6, 10 thành phần), sinh bằng
`cyclonedx-gomod`. **SBOM không thay được thông báo giấy phép:** đã đo, cờ
`-licenses` của công cụ đó trả về **0/10** trường giấy phép trên chính repo này —
im lặng, không báo lỗi. SBOM trả lời "có những gì bên trong"; `THONG-BAO-GIAY-PHEP.txt`
trả lời "và đây là văn bản giấy phép của chúng", thứ MIT/BSD thật sự đòi.

## Mã port trực tiếp từ dự án khác

**Chưa có.** Tới thời điểm này mọi dòng mã trong `cmd/` và `internal/` đều tự
viết. Các dự án trong "bản đồ tham khảo" (`docs/MASTER-PLAN.md` mục 5) mới ở mức
**đọc để học nguyên lý**, chưa sao chép mã.

Khi nào có: mỗi mục phải ghi **dự án · URL · giấy phép · commit · phần dùng ·
attribution đặt ở đâu**, và giữ nguyên thông báo bản quyền của thượng nguồn.

## Việc phải làm trước khi phát hành (Pha 7)

- [ ] Sinh SBOM (`go version -m` hoặc CycloneDX) kèm mỗi bản phát hành.
- [ ] Gộp file NOTICE chứa thông báo bản quyền của mọi phụ thuộc.
- [ ] Quét phụ thuộc (`govulncheck`) trong CI.
