# Sổ mã nguồn mở

> Cam kết ở `docs/MASTER-PLAN.md` mục 0: mọi phụ thuộc phải là mã nguồn mở, giấy
> phép tương thích MIT, và ghi lại ở đây. Ưu tiên thư viện chuẩn — thêm dependency
> phải có lý do không né được.
>
> Cập nhật: 2026-08-17.

## Phụ thuộc trực tiếp

| Gói | Phiên bản | Giấy phép | Vì sao cần | Vì sao không dùng stdlib |
|---|---|---|---|---|
| [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) | v1.56.0 | BSD-3-Clause | Nơi giữ trạng thái phiên (`~/.ai-accounts/state.db`) | Go không có SQLite trong stdlib. Chọn bản **thuần Go** thay vì `mattn/go-sqlite3` để **không cần cgo** — giữ được "một binary, build thẳng cho Windows và Linux" |
| [`github.com/BurntSushi/toml`](https://github.com/BurntSushi/toml) | v1.6.0 | MIT | Đọc `.sagent/project.toml`, `config.toml`, sau này là `flows.toml` | Go không có TOML trong stdlib. Chọn TOML thay JSON vì file này người dùng **sửa tay** và cần chú thích |

## Phụ thuộc gián tiếp

Kéo theo bởi `modernc.org/sqlite`, không gọi trực tiếp:

| Gói | Giấy phép |
|---|---|
| `modernc.org/libc`, `modernc.org/mathutil`, `modernc.org/memory` | BSD-3-Clause |
| `github.com/dustin/go-humanize` | MIT |
| `github.com/google/uuid` | BSD-3-Clause |
| `github.com/mattn/go-isatty` | MIT |
| `github.com/ncruces/go-strftime` | BSD-3-Clause |
| `github.com/remyoudompheng/bigfft` | BSD-3-Clause |
| `golang.org/x/sys` | BSD-3-Clause |

Toàn bộ là BSD-3-Clause hoặc MIT — **tương thích với giấy phép MIT** của dự án,
chỉ yêu cầu giữ thông báo bản quyền khi phát hành bản build.

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
