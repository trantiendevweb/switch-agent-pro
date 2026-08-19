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

# Bản nhúng khác bản gốc hai chỗ, và đều bắt buộc:
#   1. nav trỏ về dashboard thay vì file rời;
#   2. font lấy từ web/vendor chứ KHÔNG từ Google Fonts — asset nhúng phải vẽ
#      được khi máy rời mạng (internal/dash/offline_asset_test.go giữ điều đó).
#      Chép đè xong mà quên bước này là link CDN quay lại, và test sẽ đỏ.
python - <<'PY'
import io, re

FONT = u'''/* Font nằm sẵn trong binary (web/vendor) — trang này đọc được cả khi máy rời mạng.
   Trước đây font tải từ Google Fonts: một URL ngoài trong trang offline. */
@font-face{font-family:Inter;src:url("../vendor/inter-variable.woff2") format("woff2");
  font-weight:100 900;font-style:normal;font-display:swap}
@font-face{font-family:"Space Grotesk";src:url("../vendor/space-grotesk-variable.woff2") format("woff2");
  font-weight:300 700;font-style:normal;font-display:swap}
@font-face{font-family:"JetBrains Mono";src:url("../vendor/jetbrains-mono-variable.woff2") format("woff2");
  font-weight:100 800;font-style:normal;font-display:swap}
'''

# <link> Google Fonts, kèm dòng comment đứng trước nếu có
LINK = re.compile(u'(<!--[^\n]*font[^\n]*-->\n)?<link rel="stylesheet"[^>]*?fonts\.googleapis[^>]*?/>\n')

def thut(khoi, dem):
    if not dem:
        return khoi
    return u''.join((dem + d if d.strip() else d) + u'\n' for d in khoi.rstrip(u'\n').split(u'\n'))

for p, dem, olds in [
  ('internal/dash/web/docs/index.html', u'  ',
   [('<a href="index.html">3D</a>', '<a href="/">Dashboard ↗</a>')]),
  ('internal/dash/web/docs/master-plan.html', u'',
   [('<a href="plan.html">Plan</a><a href="index.html">3D</a>',
     '<a href="./">Plan</a><a href="/">Dashboard ↗</a>')]),
]:
    s = io.open(p, encoding='utf-8').read()
    for a, b in olds:
        s = s.replace(a, b)
    if not LINK.search(s):
        raise SystemExit(p + ': khong thay <link> Google Fonts — sua tay roi thi bo buoc nay di')
    s = LINK.sub(u'', s, count=1)
    s = s.replace(u'<style>\n', u'<style>\n' + thut(FONT, dem), 1)
    io.open(p, 'w', encoding='utf-8', newline='').write(s)
PY

echo "  ✓ đã đồng bộ docs vào bản nhúng — nhớ build lại: go build -o sagent ./cmd/sagent"
