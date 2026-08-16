<#
  cai-dat.ps1 (v2) — build & cài binary Go `sagent` cho Windows.

  Không cần quyền quản trị. Cần Go: ưu tiên ~/go-sdk (bản giải nén), hoặc `go`
  trong PATH. Cài binary vào %USERPROFILE%\bin (nơi v1 cũng dùng cho `tk`).

  Chạy:  .\install\cai-dat.ps1
#>
[CmdletBinding()]
param([switch]$KhongHoi)

$ErrorActionPreference = 'Stop'
try { [Console]::OutputEncoding = [Text.Encoding]::UTF8 } catch { }

$Repo = Split-Path $PSScriptRoot -Parent
$Bin  = Join-Path $env:USERPROFILE 'bin'

function Ok($m)  { Write-Host "  ✓ $m" -ForegroundColor Green }
function Loi($m) { Write-Host "  ✗ $m" -ForegroundColor Red; exit 1 }

Write-Host ''
Write-Host '  Cài sagent (v2, Go)' -ForegroundColor Cyan
Write-Host ''

# --- tìm Go ---
$go = $null
$sdk = Join-Path $env:USERPROFILE 'go-sdk\go\bin\go.exe'
if (Test-Path $sdk) { $go = $sdk }
else { $c = Get-Command go -ErrorAction SilentlyContinue; if ($c) { $go = $c.Source } }
if (-not $go) { Loi 'Không thấy Go. Cài Go (https://go.dev/dl) rồi chạy lại.' }
Ok "Go: $go"

# --- build ---
New-Item -ItemType Directory -Force -Path $Bin | Out-Null
$exe = Join-Path $Bin 'sagent.exe'
Write-Host '  Đang build...' -ForegroundColor Cyan
Push-Location $Repo
try { & $go build -o $exe ./cmd/sagent } finally { Pop-Location }
if ($LASTEXITCODE -ne 0) { Loi 'build thất bại' }
Ok "Đã cài: $exe"

# --- PATH ---
$u = [Environment]::GetEnvironmentVariable('Path', 'User')
if (($u -split ';' | Where-Object { $_.TrimEnd('\') -eq $Bin.TrimEnd('\') }).Count -eq 0) {
  Write-Host "  ! $Bin chưa nằm trong PATH." -ForegroundColor Yellow
  $tra = 'k'
  if ($KhongHoi) { $tra = 'c' }
  elseif (-not [Console]::IsInputRedirected) { $tra = (Read-Host '  Thêm vào PATH bây giờ? (c/k)').Trim().ToLower() }
  if ($tra -eq 'c') {
    [Environment]::SetEnvironmentVariable('Path', ($u.TrimEnd(';') + ';' + $Bin), 'User')
    Ok 'Đã thêm PATH (cửa sổ đang mở cần mở lại mới thấy)'
  }
} else { Ok 'bin đã trong PATH' }

Write-Host ''
Write-Host '  Xong. Thử:' -ForegroundColor Cyan
Write-Host '    sagent                    bảng tài khoản'
Write-Host '    sagent them claude:phu1   thêm tài khoản rồi đăng nhập'
Write-Host '    sagent verify claude      chạy bộ "đã đo"'
Write-Host ''
