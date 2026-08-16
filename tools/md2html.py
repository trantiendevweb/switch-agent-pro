# -*- coding: utf-8 -*-
"""md2html.py — dung docs/*.md thanh trang HTML TINH, tu chua.

Vi sao can: ban truoc dung marked.js tu CDN + fetch() de render markdown. Tren
mang di dong CDN co the khong tai duoc -> script inline khong bao gio chay ->
trang ket o "Dang tai...". Ngoai ra `python -m http.server` chay DON LUONG nen
request thu hai (fetch file .md) de bi nghen.

Cach sua: render san o day. Trang ra doi KHONG can CDN, KHONG fetch, chi mot
request. Font tai kieu khong chan (media=print onload), thieu font van doc duoc.

Dung:
    python tools/md2html.py docs/MASTER-PLAN.md master-plan.html "Master Plan"
"""
import html
import re
import sys

import markdown

CSS = """
:root{--bg:#0F172A;--primary:#1E293B;--muted:#272F42;--fg:#F8FAFC;--sub:#94A3B8;
--border:#475569;--run:#22C55E;--warn:#F59E0B;--limit:#EF4444;
--panel:rgba(30,41,59,.55);--ease:cubic-bezier(0.16,1,0.3,1);
--safe-b:env(safe-area-inset-bottom,0px)}
*{box-sizing:border-box}
html{scroll-behavior:smooth;-webkit-text-size-adjust:100%}
body{margin:0;background:
radial-gradient(1100px 520px at 50% -8%,rgba(34,197,94,.10),transparent 60%),
radial-gradient(800px 460px at 100% 6%,rgba(66,133,244,.07),transparent 55%),var(--bg);
color:var(--fg);font-family:Inter,system-ui,-apple-system,"Segoe UI",sans-serif;
line-height:1.65;-webkit-font-smoothing:antialiased;overflow-wrap:break-word}
.nav{position:sticky;top:0;z-index:20;display:flex;align-items:center;gap:10px;padding:12px 16px;
backdrop-filter:blur(18px);-webkit-backdrop-filter:blur(18px);background:rgba(15,23,42,.85);
border-bottom:1px solid rgba(71,85,105,.45)}
.nav .b{display:flex;align-items:center;gap:8px;font-weight:700;font-size:15px}
.nav .sp{margin-left:auto;display:flex;gap:8px}
.nav a{text-decoration:none;color:var(--fg);border:1px solid var(--border);border-radius:8px;
padding:7px 11px;font-size:12px;font-weight:600;transition:all .2s var(--ease);white-space:nowrap}
.nav a:hover{border-color:var(--run);background:rgba(34,197,94,.1)}
.nav a.p{border-color:var(--run);background:rgba(34,197,94,.14);color:var(--run)}
main{max-width:860px;margin:0 auto;padding:22px 18px calc(80px + var(--safe-b))}
h1{font-size:clamp(26px,6vw,40px);line-height:1.14;font-weight:800;letter-spacing:-.02em;margin:6px 0 14px;
background:linear-gradient(120deg,#22C55E,#4285F4);-webkit-background-clip:text;background-clip:text;color:transparent}
h2{font-size:clamp(19px,4vw,26px);font-weight:700;margin:38px 0 10px;padding-top:14px;
border-top:1px solid rgba(71,85,105,.4);letter-spacing:-.01em;scroll-margin-top:64px}
h3{font-size:clamp(16px,3vw,19px);font-weight:600;margin:24px 0 8px;color:var(--run);scroll-margin-top:64px}
h4{font-size:15px;font-weight:600;margin:18px 0 6px;color:var(--sub)}
p{margin:10px 0;color:#dbe3ee}
a{color:#7dd3fc}
ul,ol{padding-left:22px;margin:10px 0}
li{margin:5px 0;color:#dbe3ee}
li::marker{color:var(--run)}
strong{color:var(--fg);font-weight:600}
em{color:var(--sub)}
code{background:var(--muted);padding:2px 6px;border-radius:5px;font-size:.87em;color:#e2e8f0;
font-family:ui-monospace,Consolas,monospace}
pre{background:rgba(15,23,42,.8);border:1px solid var(--border);border-radius:12px;padding:14px;
overflow-x:auto;margin:14px 0}
pre code{background:none;padding:0;font-size:12.5px;line-height:1.6}
blockquote{margin:14px 0;padding:12px 16px;border-left:3px solid var(--run);
background:rgba(34,197,94,.07);border-radius:0 10px 10px 0;color:#cbd5e1}
blockquote strong{color:var(--run)}
.tw{overflow-x:auto;margin:14px 0;-webkit-overflow-scrolling:touch}
table{width:100%;border-collapse:collapse;font-size:13.5px;min-width:min(100%,520px)}
th,td{text-align:left;padding:9px 11px;border-bottom:1px solid rgba(71,85,105,.4);vertical-align:top}
th{color:var(--sub);font-weight:600;text-transform:uppercase;font-size:11px;letter-spacing:.07em;white-space:nowrap}
hr{border:0;border-top:1px solid rgba(71,85,105,.35);margin:26px 0}
.task{list-style:none;margin-left:-16px}
.task .box{display:inline-block;width:15px;height:15px;border:1.5px solid var(--border);border-radius:4px;
margin-right:8px;vertical-align:-2px}
.task.done{color:var(--sub)}
.task.done .box{background:var(--run);border-color:var(--run);position:relative}
.task.done .box::after{content:"";position:absolute;left:4px;top:1px;width:4px;height:8px;
border:solid #0F172A;border-width:0 2px 2px 0;transform:rotate(45deg)}
.top{position:fixed;right:16px;bottom:calc(16px + var(--safe-b));z-index:15;background:var(--panel);
border:1px solid var(--border);color:var(--fg);border-radius:999px;width:44px;height:44px;display:grid;
place-items:center;cursor:pointer;backdrop-filter:blur(14px);text-decoration:none}
@media(prefers-reduced-motion:reduce){html{scroll-behavior:auto}}
"""

PAGE = """<!DOCTYPE html>
<html lang="vi">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
<title>{title}</title>
<!-- font tai kieu KHONG CHAN: hong font van doc duoc -->
<link rel="stylesheet" media="print" onload="this.media='all'"
 href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap" />
<style>{css}</style>
</head>
<body id="top">
<nav class="nav">
  <div class="b">
    <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="#22C55E" stroke-width="1.8"><path d="M12 2l8.5 5v10L12 22 3.5 17V7z"/><circle cx="12" cy="12" r="3" fill="#22C55E" stroke="none"/></svg>
    Switch-Agent-Pro
  </div>
  <div class="sp">{nav}</div>
</nav>
<main>{body}</main>
<a class="top" href="#top" aria-label="Len dau trang">
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 19V5M5 12l7-7 7 7"/></svg>
</a>
</body>
</html>
"""

NAV = ('<a class="p" href="MASTER-PLAN.md">.md</a>'
       '<a href="plan.html">Plan</a>'
       '<a href="index.html">3D</a>')


ITEM = re.compile(r'^\s*([-*+]|\d+\.)\s')


def normalize_lists(text):
    """Chen dong trong truoc mot danh sach di ngay sau doan van.

    Python-Markdown (khac GitHub) doi PHAI co dong trong truoc list; khong co thi
    no nuot cac dong '- [ ]' vao doan van va checkbox khong bao gio thanh hinh.
    Bo qua ben trong khoi code ```.
    """
    out, prev, in_fence = [], "", False
    for line in text.split("\n"):
        if line.lstrip().startswith("```"):
            in_fence = not in_fence
        if not in_fence and ITEM.match(line):
            prev_blank = prev.strip() == ""
            prev_item = bool(ITEM.match(prev))
            # dong thut le = dong tiep cua muc truoc -> van cung mot list
            prev_cont = prev.startswith(("  ", "\t")) and prev.strip() != ""
            if not (prev_blank or prev_item or prev_cont):
                out.append("")
        out.append(line)
        prev = line
    return "\n".join(out)


def task_lists(h):
    """Bien '- [ ]' / '- [x]' thanh o tick tinh (khong can JS)."""
    h = re.sub(r'<li>\s*\[[xX]\]\s*', '<li class="task done"><span class="box"></span>', h)
    h = re.sub(r'<li>\s*\[ \]\s*', '<li class="task"><span class="box"></span>', h)
    return h


def main(argv):
    if len(argv) < 3:
        print(__doc__)
        return 1
    src, dst = argv[1], argv[2]
    title = argv[3] if len(argv) > 3 else "Plan"

    with open(src, encoding="utf-8") as f:
        text = f.read()

    body = markdown.markdown(
        normalize_lists(text),
        # KHONG dung nl2br: file .md xuong dong cung ~80 ky tu, nl2br se lam
        # cau bi gay giua chung tren dien thoai.
        extensions=["tables", "fenced_code", "sane_lists"],
        output_format="html5",
    )
    body = task_lists(body)
    # bang cuon ngang duoc tren dien thoai
    body = body.replace("<table>", '<div class="tw"><table>').replace("</table>", "</table></div>")

    out = PAGE.format(title=html.escape(title), css=CSS, nav=NAV, body=body)
    with open(dst, "w", encoding="utf-8") as f:
        f.write(out)
    print("  da render %s -> %s (%d KB)" % (src, dst, len(out) // 1024))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
