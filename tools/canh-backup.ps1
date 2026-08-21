# canh-backup.ps1 - lam cho "backup khong roi khoi may" DO len thay vi im lang.
#
# VI SAO CO (do 21/08/2026): backup .enc duoc ghi vao C:\Users\<user>\OneDrive,
# nhung OneDrive.exe KHONG CON TREN MAY - khong o Program Files, khong o
# %LOCALAPPDATA%, khong phai goi Appx. Registry thi VAN giu lien ket tai khoan,
# nen thu muc do TRONG y het mot thu muc dong bo. Ket qua: 19 file / 782.8 MB
# nam cung o dia voi du lieu goc, tren MOT o vat ly duy nhat.
#
# Cai lam no nguy hiem khong phai loi, ma la SU IM LANG: LastTaskResult = 0,
# file van sinh deu, moi den deu xanh. Day la kieu hong te nhat - hong ma khong
# co gi bao.
#
# Bai canh nay khong sua duoc goc (can cai lai client + dang nhap). No chi bao
# dam mot dieu: neu backup khong roi khoi may thi CO NGUOI BIET.
#
# LUU Y CU PHAP: file nay chi dung ASCII. PowerShell 5.1 doc .ps1 UTF-8 khong
# BOM bang CP1252, nen dau gach ngang dai nam trong CHUOI se bien thanh nhay
# cong dong va lam hong cu phap IM LANG. Xem docs/VAN-HANH-VPS.md muc D1.

[CmdletBinding()]
param(
    # Bao nhieu gio khong co file .enc moi thi coi la task backup da chet.
    # Mac dinh 30: cac task chay hang ngay, nen 24 gio la binh thuong, 30 cho
    # du bien de mot lan chay tre khong gay bao dong gia.
    [int]$NguongGio = 30,

    # In ra man hinh, KHONG gui Telegram. Dung khi kiem thu.
    [switch]$Kho
)

$ErrorActionPreference = 'Stop'
$thuMuc = Join-Path $env:USERPROFILE 'OneDrive'
$loi = New-Object System.Collections.Generic.List[string]

# --- 1. Client dong bo con ton tai va co chay khong -------------------------
$duong = @(
    (Join-Path $env:LOCALAPPDATA 'Microsoft\OneDrive\OneDrive.exe'),
    (Join-Path $env:ProgramFiles 'Microsoft OneDrive\OneDrive.exe'),
    (Join-Path ${env:ProgramFiles(x86)} 'Microsoft OneDrive\OneDrive.exe')
)
$coExe = $false
foreach ($d in $duong) { if ($d -and (Test-Path $d)) { $coExe = $true } }

if (-not $coExe) {
    $loi.Add('OneDrive.exe KHONG CO tren may. Backup khong the roi khoi o dia.')
} elseif (-not (Get-Process -Name OneDrive -ErrorAction SilentlyContinue)) {
    $loi.Add('OneDrive.exe co tren may nhung KHONG CHAY.')
}

# --- 2. Dang nhap con hieu luc khong ----------------------------------------
# CO EXE + CO CHAY VAN CHUA DU. Do 21/08 20:17: client vua cai lai, tien trinh
# chay binh thuong, nhung dang dung doi mat khau va KHONG day gi len ca. Ban dau
# bai canh nay chi kiem exe + tien trinh nen no bao "OK" trong dung tinh huong
# do - tu no mac lai dung cai benh no sinh ra de chua.
$khoa = 'HKCU:\Software\Microsoft\OneDrive\Accounts\Personal'
if (Test-Path $khoa) {
    $tk = Get-ItemProperty -Path $khoa -ErrorAction SilentlyContinue
    if ($tk.LastSignInResult -and $tk.LastSignInResult -ne 0) {
        $loi.Add("OneDrive dang LOI DANG NHAP (LastSignInResult=$($tk.LastSignInResult), tai khoan $($tk.UserEmail)). Bam 'Re-enter credentials' va dang nhap lai.")
    } elseif ($tk.SignInErrorStateStartTimestamp) {
        $loi.Add("OneDrive dang o trang thai loi dang nhap (tai khoan $($tk.UserEmail)).")
    }
}

# --- 3. File .enc da THAT SU len may chua -----------------------------------
# Day moi la cau hoi that: "du lieu da roi khoi may chua", chu khong phai "co
# tien trinh nao dang chay khong".
#
# File nam trong thu muc OneDrive va DA duoc dam may biet den thi mang thuoc
# tinh ReparsePoint (placeholder cloud). File chi co Archive tron la file CHUA
# he duoc day len - no chi dang nam tren o dia nay.
if (-not (Test-Path $thuMuc)) {
    $loi.Add("Khong thay thu muc backup: $thuMuc")
    $tapTin = @()
} else {
    $tapTin = @(Get-ChildItem -LiteralPath $thuMuc -Recurse -Filter '*.enc' -ErrorAction SilentlyContinue)
}

$chuaLen = @($tapTin | Where-Object { $_.Attributes -notmatch 'ReparsePoint' })
$chuaLenMB = 0
if ($chuaLen.Count -gt 0) {
    $chuaLenMB = [math]::Round((($chuaLen | Measure-Object -Property Length -Sum).Sum / 1MB), 1)
    $loi.Add("$($chuaLen.Count)/$($tapTin.Count) file .enc CHUA len may ($chuaLenMB MB van nam tren o dia nay).")
}

if ($tapTin.Count -eq 0) {
    $loi.Add("Khong tim thay file .enc nao trong $thuMuc")
} else {
    $moiNhat = $tapTin | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    $gioTuoi = [math]::Round(((Get-Date) - $moiNhat.LastWriteTime).TotalHours, 1)
    if ($gioTuoi -gt $NguongGio) {
        $loi.Add("File .enc moi nhat da $gioTuoi gio tuoi (nguong $NguongGio). Task backup co the da chet.")
    }
}

$tongMB = 0
if ($tapTin.Count -gt 0) {
    $tongMB = [math]::Round((($tapTin | Measure-Object -Property Length -Sum).Sum / 1MB), 1)
}

# --- 4. Ket luan ------------------------------------------------------------
if ($loi.Count -eq 0) {
    Write-Output "OK: ca $($tapTin.Count) file .enc ($tongMB MB) deu da len may."
    exit 0
}

$dong = New-Object System.Collections.Generic.List[string]
$dong.Add('CANH BAO BACKUP - du lieu KHONG roi khoi may')
$dong.Add('')
foreach ($l in $loi) { $dong.Add("- $l") }
$dong.Add('')
$dong.Add("Tong: $($tapTin.Count) file .enc, $tongMB MB. Chua len may: $($chuaLen.Count) file, $chuaLenMB MB.")
$dong.Add("May chi co MOT o vat ly, nen o chet la mat ca hai ban.")
$dong.Add('')
$dong.Add('Tra cuu: docs/VAN-HANH-VPS.md muc E.')
$tin = $dong -join "`n"

Write-Output $tin

if ($Kho) {
    Write-Output ''
    Write-Output '(che do --Kho: khong gui Telegram)'
    exit 1
}

# Dung DUNG duong bao tin sagent da cau hinh, khong dung thong tin dang nhap moi.
$cauHinh = Join-Path $env:USERPROFILE '.ai-accounts\telegram.json'
if (-not (Test-Path $cauHinh)) {
    Write-Output "Khong gui duoc: chua co $cauHinh. Dat bang: sagent tele --set-token <token> --chat <id>"
    exit 1
}

try {
    $c = Get-Content -LiteralPath $cauHinh -Raw | ConvertFrom-Json
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $than = @{ chat_id = $c.chat_id; text = $tin } | ConvertTo-Json -Compress
    $null = Invoke-RestMethod -Method Post -TimeoutSec 30 `
        -Uri "https://api.telegram.org/bot$($c.token)/sendMessage" `
        -ContentType 'application/json; charset=utf-8' `
        -Body ([Text.Encoding]::UTF8.GetBytes($than))
    Write-Output 'Da gui canh bao qua Telegram.'
} catch {
    # Gui hong thi VAN phai thoat khac 0: mat duong bao tin ma task bao xanh la
    # quay lai dung cai benh ban dau.
    Write-Output "Gui Telegram hong: $($_.Exception.Message)"
}

exit 1
