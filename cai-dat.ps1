<#
  cai-dat.ps1 — cài ccswitch cho người dùng hiện tại.

  Không cần quyền quản trị: chỉ ghi vào %USERPROFILE%, và nối thư mục bằng
  junction chứ không phải symlink (junction không đòi quyền quản trị).

  Chạy:  .\cai-dat.ps1
#>

[CmdletBinding()]
param([switch]$KhongHoi)

$ErrorActionPreference = 'Stop'
try { [Console]::OutputEncoding = [Text.Encoding]::UTF8 } catch { }

$Nguon   = $PSScriptRoot
$Skill   = Join-Path $env:USERPROFILE '.claude\skills\ccswitch'
$ThuMucBin = Join-Path $env:USERPROFILE 'bin'
$Shim    = Join-Path $ThuMucBin 'tk.cmd'

function Ok($m)   { Write-Host "  ✓ $m" -ForegroundColor Green }
function Nhac($m) { Write-Host "  ! $m" -ForegroundColor Yellow }
function Loi($m)  { Write-Host "  ✗ $m" -ForegroundColor Red; exit 1 }
function Mo($m)   { Write-Host "  $m" -ForegroundColor DarkGray }

Write-Host ''
Write-Host '  Cài ccswitch' -ForegroundColor Cyan
Write-Host ''

# --- 1. Kiểm thứ cần có trước ------------------------------------------------
$claude = Get-Command claude -ErrorAction SilentlyContinue
if (-not $claude) {
  $n = Join-Path $env:USERPROFILE '.local\bin\claude.exe'
  if (Test-Path $n) { $claude = $n } else { Loi 'Chưa thấy Claude Code. Cài Claude Code rồi chạy lại.' }
}
Ok 'Claude Code: có'

$py = $null
foreach ($t in @('python', 'py', 'python3')) {
  $c = Get-Command $t -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
  if ($c -and $c.Source) { $py = $c.Source; break }
}
if (-not $py) { Loi 'Chưa thấy Python 3. Cần Python để đọc .claude.json (PowerShell 5.1 chết khi JSON có khoá trùng hoa/thường).' }
Ok "Python: $py"

# --- 2. Chép script ----------------------------------------------------------
New-Item -ItemType Directory -Path (Join-Path $Skill 'scripts') -Force | Out-Null
Copy-Item (Join-Path $Nguon 'src\*') (Join-Path $Skill 'scripts') -Force
Copy-Item (Join-Path $Nguon 'docs\SKILL.md') $Skill -Force
Copy-Item (Join-Path $Nguon 'docs\HDSD.md')  $Skill -Force
Ok "Đã chép script vào $Skill"

# --- 3. Tạo lệnh tk ----------------------------------------------------------
New-Item -ItemType Directory -Path $ThuMucBin -Force | Out-Null
# File .cmd chay bang bang ma he thong, khong phai UTF-8 — nen chu thich trong
# day co y viet KHONG DAU, tranh bien thanh dau hoi luc chay.
@"
@echo off
rem tk - doi tai khoan Claude Code (ccswitch).
rem Ban that o: %USERPROFILE%\.claude\skills\ccswitch\scripts\tk.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File "%USERPROFILE%\.claude\skills\ccswitch\scripts\tk.ps1" %*
"@ | Set-Content -LiteralPath $Shim -Encoding ASCII
Ok "Đã tạo lệnh: $Shim"

# --- 4. PATH -----------------------------------------------------------------
$duongNguoiDung = [Environment]::GetEnvironmentVariable('Path', 'User')
$coTrongPath = ($duongNguoiDung -split ';' | Where-Object { $_.TrimEnd('\') -eq $ThuMucBin.TrimEnd('\') }).Count -gt 0
if ($coTrongPath) {
  Ok 'Thư mục bin đã nằm trong PATH'
} else {
  Nhac "Thư mục $ThuMucBin chưa nằm trong PATH của bạn."
  $tra = 'k'
  if ($KhongHoi) { $tra = 'c' }
  elseif (-not [Console]::IsInputRedirected) { $tra = (Read-Host '  Thêm vào PATH bây giờ? (c/k)').Trim().ToLower() }
  if ($tra -eq 'c') {
    [Environment]::SetEnvironmentVariable('Path', ($duongNguoiDung.TrimEnd(';') + ';' + $ThuMucBin), 'User')
    $env:Path = $env:Path + ';' + $ThuMucBin
    Ok 'Đã thêm vào PATH (cửa sổ mở sẵn cần mở lại mới thấy)'
  } else {
    Mo "Chưa thêm. Gõ đường dẫn đầy đủ cũng chạy: $Shim"
  }
}

# --- 5. Xong -----------------------------------------------------------------
Write-Host ''
Write-Host '  Xong. Làm tiếp:' -ForegroundColor Cyan
Write-Host '    tk               xem bảng tài khoản'
Write-Host '    tk them phu1     thêm tài khoản thứ hai rồi đăng nhập'
Write-Host ''
Mo 'Muốn tự đo lại các giả định của công cụ: .\kiem-tra.ps1'
Write-Host ''
