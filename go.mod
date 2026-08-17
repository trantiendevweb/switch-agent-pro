module github.com/trantiendevweb/switch-agent-pro

go 1.25.0

// Toolchain ghim tường minh: go1.25.0 dính 23 lỗ hổng stdlib mà govulncheck
// xác nhận là CÓ ĐƯỜNG GỌI TỚI (crypto/tls, crypto/x509, net/http, net/url,
// net/textproto, encoding/asn1) — hầu hết qua dash.Server.Run → http.Serve.
// Nâng toolchain là cách sửa duy nhất; không dependency nào phải đổi.
toolchain go1.25.13

require (
	github.com/BurntSushi/toml v1.6.0
	golang.org/x/sys v0.47.0
	modernc.org/sqlite v1.56.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	modernc.org/libc v1.74.4 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)
