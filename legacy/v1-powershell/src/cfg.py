# -*- coding: utf-8 -*-
"""
cfg.py — phan lam viec voi .claude.json cua cong cu tk.

Vi sao tach ra Python thay vi lam thang trong PowerShell:
PowerShell 5.1 (ban co san tren Windows) NEM LOI khi JSON co hai khoa chi khac
nhau hoa/thuong. File .claude.json that hay dinh dung loi do, vi duong dan
project duoc ghi ca hai kieu:
    "C:\\Users\\ban\\Du An"  va  "c:\\users\\ban\\du an"
Neu de PowerShell doc, cong cu chet ngay tren nhung may nhu vay. Python thi doc
duoc (khoa trung thi lay cai cuoi).

Lenh:
    python cfg.py gieo    <nguon.json> <dich.json>
    python cfg.py dong-bo <nguon.json> <thu-muc-kho> [--xem-truoc]
    python cfg.py email   <file.json>
"""
import json
import os
import shutil
import sys
import time

# Khoa DUOC chep sang moi tai khoan — thu thuoc ve CAI MAY va THOI QUEN LAM VIEC.
#
# Danh sach TRANG, khong phai danh sach den. Lam nguoc lai (chep tat tru vai khoa)
# la kieu de ro: mai sau Claude Code them mot khoa gan voi goi cuoc, no lang le do
# sang tai khoan khac ma khong ai biet.
CHIA_SE = [
    "projects",                      # trust dialog, allowedTools, MCP theo tung project
    "hasCompletedOnboarding",
    "lastOnboardingVersion",
    "remoteDialogSeen",
    "remoteControlSurfacesSeen",
    "claudeAiMcpEverConnected",
    "githubRepoPaths",
    "tipsHistory",
    "tipLifetimeShownCounts",
    "skillUsage",
    "pluginUsage",
    "autoUpdates",
    "installMethod",
    "seenNotifications",
    "announcementImpressions",
    "lastReleaseNotesSeen",
    "hasCompletedClaudeInChromeOnboarding",
    "claudeInChromeDefaultEnabled",
    "cachedChromeExtensionInstalled",
]

# KHONG chep, va vi sao:
#   oauthAccount, userID
#       -> danh tinh tai khoan. Chep sang la hai tai khoan cung mot danh tinh.
#   modelAccessCache, orgModelDefaultCache, penguinModeOrgEnabled,
#   passesEligibilityCache, passesLastSeenRemaining, passesUpsellSeenCount,
#   hasVisitedPasses, additionalModelCostsCache, additionalModelOptionsCache,
#   cachedExtraUsageDisabledReason
#       -> gan voi GOI CUOC / TO CHUC cua tung tai khoan. Chep sang la tai khoan B
#          tuong minh co quyen cua A, roi bao loi kho hieu luc dung.
#   firstStartTime, claudeCodeFirstTokenDate, machineID, numStartups
#       -> so lieu rieng, chep sang chi lam sai thong ke.


def doc(duong):
    with open(duong, encoding="utf-8") as f:
        return json.load(f)


def ghi_nguyen_tu(duong, obj):
    """Ghi ra file tam roi doi ten. Khong bao gio de lai file nua voi."""
    tmp = duong + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(obj, f, ensure_ascii=False)
    os.replace(tmp, duong)


def phan_dung_chung(goc):
    return {k: goc[k] for k in CHIA_SE if k in goc}


def email_cua(d):
    # Tra ve dau gach khi chua dang nhap. Co y KHONG in tieng Viet co dau o day:
    # stdout cua Python tren Windows doi bang ma theo console, bi chuyen huong la
    # nat chu. Phia PowerShell se doi "-" thanh "(chua dang nhap)" co dau.
    return (d.get("oauthAccount") or {}).get("emailAddress") or "-"


def lenh_gieo(nguon, dich):
    """Tao .claude.json moi CHI tu danh sach trang.

    Co y KHONG chep ca file roi xoa vai khoa: lam vay la mang theo ca cache goi
    cuoc cua tai khoan cu sang tai khoan moi.
    """
    goc = doc(nguon)
    co = phan_dung_chung(goc)
    ghi_nguyen_tu(dich, co)
    prj = co.get("projects") or {}
    tin = sum(1 for v in prj.values() if isinstance(v, dict) and v.get("hasTrustDialogAccepted"))
    print("  gieo %d khoa, %d project (%d da trust)" % (len(co), len(prj), tin))
    return 0


def lenh_dong_bo(nguon, kho, xem_truoc):
    goc = doc(nguon)
    co = phan_dung_chung(goc)
    print("  Nguon : %s" % nguon)
    print("  Se dong bo %d khoa" % len(co))
    print("")

    if not os.path.isdir(kho):
        print("  Chua co tai khoan nao.")
        return 0

    accs = sorted(d for d in os.listdir(kho) if os.path.isdir(os.path.join(kho, d)))
    if not accs:
        print("  Chua co tai khoan nao.")
        return 0

    for acc in accs:
        dich = os.path.join(kho, acc, ".claude.json")
        if not os.path.exists(dich):
            print("  %-14s bo qua (chua co .claude.json)" % acc)
            continue
        try:
            d = doc(dich)
        except Exception as e:
            print("  %-14s HONG, bo qua: %s" % (acc, e))
            continue

        doi = [k for k, v in co.items() if d.get(k) != v]
        em = email_cua(d)
        if not doi:
            print("  %-14s %-30s da khop" % (acc, em))
            continue
        print("  %-14s %-30s doi %d khoa: %s" % (acc, em, len(doi), ", ".join(sorted(doi)[:4])))
        if xem_truoc:
            continue

        # sao luu truoc khi ghi — file nay chua trust dialog, hong thi phai bam lai het
        shutil.copy2(dich, dich + ".bak-" + time.strftime("%Y%m%d-%H%M%S"))
        d.update(co)
        ghi_nguyen_tu(dich, d)

    print("")
    if xem_truoc:
        print("  CHE DO XEM TRUOC — chua ghi gi. Bo -XemTruoc de chay that.")
    else:
        print("  Xong. Ban cu luu kem duoi .bak-<ngay gio> canh moi file.")
    return 0


def lenh_email(duong):
    try:
        print(email_cua(doc(duong)))
    except Exception:
        print("(khong doc duoc)")
    return 0


def main(argv):
    if len(argv) < 2:
        print(__doc__)
        return 1
    lenh = argv[1]
    if lenh == "gieo" and len(argv) == 4:
        return lenh_gieo(argv[2], argv[3])
    if lenh == "dong-bo" and len(argv) >= 4:
        return lenh_dong_bo(argv[2], argv[3], "--xem-truoc" in argv)
    if lenh == "email" and len(argv) == 3:
        return lenh_email(argv[2])
    print(__doc__)
    return 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
