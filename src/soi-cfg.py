# Soi hai file .claude.json xem file nao la file THAT dang duoc dung.
# Viet rieng vi PowerShell 5.1 chet khi JSON co khoa trung hoa/thuong.
import json, os, sys

home = os.path.expanduser("~")
for p in [os.path.join(home, ".claude.json"), os.path.join(home, ".claude", ".claude.json")]:
    print("=" * 70)
    print(p)
    if not os.path.exists(p):
        print("  KHONG CO")
        continue
    with open(p, encoding="utf-8") as f:
        d = json.load(f)          # json cua Python: khoa trung thi lay cai cuoi
    oauth = d.get("oauthAccount") or {}
    prj = d.get("projects") or {}
    tin_cay = [k for k, v in prj.items() if isinstance(v, dict) and v.get("hasTrustDialogAccepted")]
    print("  kich thuoc     : %d byte" % os.path.getsize(p))
    print("  so khoa        : %d" % len(d))
    print("  email          : %s" % (oauth.get("emailAddress") or "(khong co)"))
    print("  so project     : %d" % len(prj))
    print("  da trust       : %d" % len(tin_cay))
    print("  numStartups    : %s" % d.get("numStartups"))
    print("  installMethod  : %s" % d.get("installMethod"))
    print("  khoa goi cuoc  : %s" % ", ".join(k for k in d if "odelAccess" in k or "asses" in k or "enguin" in k) or "(khong co)")
