#!/usr/bin/env bash
# Đồng bộ trang kế hoạch vào thư mục nhúng của dash, rồi build lại binary.
#
# Vì sao cần: plan.html / master-plan.html được nhúng vào binary bằng go:embed,
# nên sửa file ở gốc repo thôi CHƯA đủ — phải chép vào internal/dash/web/docs/
# rồi build lại thì server mới phục vụ bản mới.
set -euo pipefail
cd "$(dirname "$0")/.."

python tools/md2html.py docs/MASTER-PLAN.md master-plan.html "Switch-Agent-Pro — Master Plan"

mkdir -p internal/dash/web/docs
cp plan.html        internal/dash/web/docs/index.html
cp master-plan.html internal/dash/web/docs/master-plan.html
cp docs/MASTER-PLAN.md internal/dash/web/docs/MASTER-PLAN.md

# nav trong bản nhúng trỏ về dashboard thay vì file rời
python - <<'PY'
import io
for p, olds in [
  ('internal/dash/web/docs/index.html', [('<a href="index.html">3D</a>', '<a href="/">Dashboard ↗</a>')]),
  ('internal/dash/web/docs/master-plan.html', [('<a href="plan.html">Plan</a><a href="index.html">3D</a>',
                                                '<a href="./">Plan</a><a href="/">Dashboard ↗</a>')]),
]:
    s = io.open(p, encoding='utf-8').read()
    for a, b in olds:
        s = s.replace(a, b)
    io.open(p, 'w', encoding='utf-8', newline='').write(s)
PY

echo "  ✓ đã đồng bộ docs vào bản nhúng — nhớ build lại: go build -o sagent ./cmd/sagent"
