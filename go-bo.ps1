<#
  go-bo.ps1 — gỡ ccswitch.

  Xoá lệnh tk và thư mục skill. KHÔNG đụng vào %USERPROFILE%\.claude.
  Thư mục tài khoản (%USERPROFILE%\.claude-accounts) chỉ xoá khi bạn đồng ý,
  vì trong đó có token — xoá là phải đăng nhập lại từ đầu.
#>

[CmdletBinding()]
param([switch]$XoaLuonTaiKhoan)

$ErrorActionPreference = 'Stop'
try { [Console]::OutputEncoding = [Text.Encoding]::UTF8 } catch { }

$Skill = Join-Path $env:USERPROFILE '.claude\skills\ccswitch'
$Shim  = Join-Path $env:USERPROFILE 'bin\tk.cmd'
$Kho   = Join-Path $env:USERPROFILE '.claude-accounts'

function Ok($m)   { Write-Host "  ✓ $m" -ForegroundColor Green }
function Mo($m)   { Write-Host "  $m" -ForegroundColor DarkGray }

function La-Link($duong) {
  $it = Get-Item -LiteralPath $duong -Force -ErrorAction SilentlyContinue
  if (-not $it) { return $false }
  return [bool]($it.Attributes -band [IO.FileAttributes]::ReparsePoint)
}

Write-Host ''
Write-Host '  Gỡ ccswitch' -ForegroundColor Cyan
Write-Host ''

if (Test-Path $Shim)  { Remove-Item $Shim -Force; Ok 'Đã xoá lệnh tk' } else { Mo 'Không thấy lệnh tk' }
if (Test-Path $Skill) { Remove-Item $Skill -Recurse -Force; Ok 'Đã xoá thư mục skill' } else { Mo 'Không thấy thư mục skill' }

if (Test-Path $Kho) {
  $ds = @(Get-ChildItem $Kho -Directory -ErrorAction SilentlyContinue)
  Write-Host ''
  Write-Host "  Còn $($ds.Count) tài khoản trong $Kho (có token bên trong)." -ForegroundColor Yellow
  $tra = 'k'
  if ($XoaLuonTaiKhoan) { $tra = 'c' }
  elseif (-not [Console]::IsInputRedirected) { $tra = (Read-Host '  Xoá luôn? Xoá là phải đăng nhập lại (c/k)').Trim().ToLower() }

  if ($tra -eq 'c') {
    # Gỡ link trước rồi mới xoá — Remove-Item -Recurse có thể xuyên qua junction
    # xoá luôn dữ liệu thật trong .claude gốc.
    foreach ($d in $ds) {
      foreach ($m in Get-ChildItem -LiteralPath $d.FullName -Force) {
        if (La-Link $m.FullName) {
          if ($m.PSIsContainer) { cmd /c rmdir "$($m.FullName)" > $null 2>&1 }
          else { [IO.File]::Delete($m.FullName) }
        }
      }
    }
    $con = @(Get-ChildItem -LiteralPath $Kho -Force -Recurse -ErrorAction SilentlyContinue | Where-Object { La-Link $_.FullName })
    if ($con.Count -gt 0) {
      Write-Host "  ✗ Còn $($con.Count) link chưa gỡ, không dám xoá đệ quy. Xem tay: $Kho" -ForegroundColor Red
      exit 1
    }
    Remove-Item $Kho -Recurse -Force
    Ok 'Đã xoá toàn bộ tài khoản và token'
  } else {
    Mo "Giữ nguyên $Kho. Cài lại ccswitch là dùng tiếp được, không phải đăng nhập lại."
  }
}

Write-Host ''
Mo 'Thư mục .claude gốc không bị đụng tới.'
Write-Host ''
