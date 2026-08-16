#!/usr/bin/env bash
# cai-dat.sh (v2) — build & cài `ccswitch` trên Linux.
#
# ⚠ CHƯA ĐO trên Linux: Pha 0 Linux chưa chạy (token nằm ở file hay keyring
#   chưa xác minh). Dùng script này khi bạn có môi trường Linux để thử; phần
#   Linux còn ở trạng thái experimental.
#
# Cần Go: ưu tiên ~/go-sdk, hoặc `go` trong PATH.
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
bin="$HOME/.local/bin"

go="$(command -v go || true)"
[ -x "$HOME/go-sdk/go/bin/go" ] && go="$HOME/go-sdk/go/bin/go"
[ -z "$go" ] && { echo "  ✗ Không thấy Go. Cài Go (https://go.dev/dl) rồi chạy lại."; exit 1; }
echo "  ✓ Go: $go"

mkdir -p "$bin"
echo "  Đang build..."
( cd "$repo" && "$go" build -o "$bin/ccswitch" ./cmd/ccswitch )
echo "  ✓ Đã cài: $bin/ccswitch"

case ":$PATH:" in
  *":$bin:"*) : ;;
  *) echo "  ! $bin chưa nằm trong PATH — thêm 'export PATH=\$HOME/.local/bin:\$PATH' vào ~/.profile" ;;
esac

echo "  Thử: ccswitch  ·  ccswitch them claude:phu1"
