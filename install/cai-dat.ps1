<#
  cai-dat.ps1 — cài `sagent` cho Windows.

  Một dòng, không cần Go, không cần quyền quản trị:

      irm https://raw.githubusercontent.com/trantiendevweb/switch-agent-pro/main/install/cai-dat.ps1 | iex

  Nó tải binary dựng sẵn từ GitHub Releases, đối chiếu SHA256, đặt vào
  %USERPROFILE%\bin và thêm vào PATH của NGƯỜI DÙNG (không đụng PATH hệ thống,
  nên không cần admin).

  Cờ:
    -TuNguon        build từ mã nguồn (cần Go + đang đứng trong repo đã clone)
    -Phien v0.2.0   cài đúng một phiên bản thay vì bản mới nhất
    -KhongHoi       không hỏi gì, tự thêm PATH

  Yêu cầu: Windows + PowerShell 5.1 (có sẵn từ Windows 10). Không cần gì khác.
#>
[CmdletBinding()]
param(
  [switch]$TuNguon,
  [string]$Phien = '',
  [switch]$KhongHoi
)

$ErrorActionPreference = 'Stop'
try { [Console]::OutputEncoding = [Text.Encoding]::UTF8 } catch { }

# Thanh tiến trình của PowerShell 5.1 làm Invoke-WebRequest chậm đi NHIỀU LẦN vì
# nó vẽ lại sau mỗi khối dữ liệu. Tắt đi là mẹo tăng tốc lớn nhất ở đây.
$ProgressPreference = 'SilentlyContinue'

$Repo = 'trantiendevweb/switch-agent-pro'
$Bin  = Join-Path $env:USERPROFILE 'bin'
$Exe  = Join-Path $Bin 'sagent.exe'

function Ok($m)  { Write-Host "  ✓ $m" -ForegroundColor Green }
function Nhac($m){ Write-Host "  ! $m" -ForegroundColor Yellow }
function Loi($m) { Write-Host "  ✗ $m" -ForegroundColor Red; exit 1 }

Write-Host ''
Write-Host '  Cài sagent' -ForegroundColor Cyan
Write-Host ''

if ($env:OS -ne 'Windows_NT') { Loi 'sagent chỉ hỗ trợ Windows.' }

# Windows cũ mặc định TLS 1.0/1.1 — GitHub đã từ chối cả hai. Không đặt dòng này
# thì lỗi hiện ra là "kết nối bị đóng", chẳng chỉ được gì cho ai.
try {
  [Net.ServicePointManager]::SecurityProtocol =
    [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
} catch { }

# --- kiến trúc ---
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  'AMD64' { 'amd64' }
  'ARM64' { 'arm64' }
  'x86'   { Loi 'Windows 32-bit không có bản dựng sẵn. Dùng -TuNguon nếu bạn có Go.' }
  default { 'amd64' }
}

New-Item -ItemType Directory -Force -Path $Bin | Out-Null

function Cai-TuNguon {
  $goExe = $null
  $sdk = Join-Path $env:USERPROFILE 'go-sdk\go\bin\go.exe'
  if (Test-Path $sdk) { $goExe = $sdk }
  else { $c = Get-Command go -ErrorAction SilentlyContinue; if ($c) { $goExe = $c.Source } }
  if (-not $goExe) { Loi 'Không thấy Go. Bỏ cờ -TuNguon để tải bản dựng sẵn (không cần Go).' }

  $root = if ($PSScriptRoot) { Split-Path $PSScriptRoot -Parent } else { (Get-Location).Path }
  if (-not (Test-Path (Join-Path $root 'go.mod'))) {
    Loi "Không thấy go.mod ở $root — build từ nguồn phải chạy trong repo đã clone."
  }
  Ok "Go: $goExe"
  Write-Host '  Đang build...' -ForegroundColor Cyan
  Push-Location $root
  try {
    & $goExe build -trimpath -ldflags '-s -w' -o $Exe ./cmd/sagent
  } finally { Pop-Location }
  if ($LASTEXITCODE -ne 0) { Loi 'build thất bại' }
}

function Cai-BanDungSan {
  # Hỏi GitHub bản mới nhất. Không có release nào thì nói thẳng chứ đừng để
  # người dùng nhìn một lỗi JSON.
  if ($Phien) {
    $tag = $Phien
  } else {
    try {
      $r = Invoke-RestMethod -UseBasicParsing -Uri "https://api.github.com/repos/$Repo/releases/latest"
      $tag = $r.tag_name
    } catch {
      Nhac 'Chưa có bản phát hành nào trên GitHub.'
      Write-Host '    Cài từ nguồn (cần Go, phải đứng trong repo đã clone):' -ForegroundColor Yellow
      Write-Host '      .\install\cai-dat.ps1 -TuNguon' -ForegroundColor Yellow
      exit 1
    }
  }
  Ok "Bản: $tag ($arch)"

  $ten  = "sagent-windows-$arch.exe"
  $goc  = "https://github.com/$Repo/releases/download/$tag"
  $tam  = Join-Path ([IO.Path]::GetTempPath()) ("sagent-" + [Guid]::NewGuid().ToString('N'))
  New-Item -ItemType Directory -Force -Path $tam | Out-Null
  $tai  = Join-Path $tam $ten

  try {
    Invoke-WebRequest -UseBasicParsing -Uri "$goc/$ten" -OutFile $tai
    Ok ("Đã tải {0:N1} MB" -f ((Get-Item $tai).Length / 1MB))

    # Đối chiếu băm. Tải một file .exe rồi chạy ngay mà không kiểm thì trình cài
    # này chính là lỗ hổng nó lẽ ra phải tránh.
    $sumFile = Join-Path $tam 'SHA256SUMS.txt'
    Invoke-WebRequest -UseBasicParsing -Uri "$goc/SHA256SUMS.txt" -OutFile $sumFile
    $muon = $null
    foreach ($d in Get-Content $sumFile) {
      $p = $d -split '\s+'
      if ($p.Length -ge 2 -and $p[1].TrimStart('*') -eq $ten) { $muon = $p[0].ToLower() }
    }
    if (-not $muon) { Loi "SHA256SUMS.txt không có dòng cho $ten — không cài." }
    $that = (Get-FileHash -Algorithm SHA256 $tai).Hash.ToLower()
    if ($that -ne $muon) { Loi "BĂM KHÔNG KHỚP. muốn $muon, được $that — KHÔNG cài." }
    Ok 'SHA256 khớp'

    # Binary cũ đang chạy (dash chẳng hạn) thì Windows khoá file. Đổi tên rồi
    # ghi đè: Windows cho đổi tên file đang mở, không cho ghi đè.
    if (Test-Path $Exe) {
      $cu = "$Exe.cu-$(Get-Date -Format yyyyMMdd-HHmmss)"
      try { Move-Item -Force $Exe $cu } catch { Loi "sagent.exe đang chạy và không đổi tên được. Dừng dash/phiên rồi cài lại." }
      Nhac "Bản cũ đổi tên thành $(Split-Path $cu -Leaf) — xoá được sau khi đóng tiến trình cũ."
    }
    Move-Item -Force $tai $Exe
  } finally {
    Remove-Item -Recurse -Force $tam -ErrorAction SilentlyContinue
  }
}

if ($TuNguon) { Cai-TuNguon } else { Cai-BanDungSan }
Ok "Đã cài: $Exe"

# --- PATH của người dùng (không cần admin) ---
$u = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not $u) { $u = '' }
$daCo = ($u -split ';' | Where-Object { $_ -and $_.TrimEnd('\') -eq $Bin.TrimEnd('\') }).Count -gt 0
if (-not $daCo) {
  $tra = 'c'
  if (-not $KhongHoi -and -not [Console]::IsInputRedirected) {
    $tra = (Read-Host "  Thêm $Bin vào PATH? (c/k)").Trim().ToLower()
  }
  if ($tra -eq 'c') {
    [Environment]::SetEnvironmentVariable('Path', ($u.TrimEnd(';') + ';' + $Bin), 'User')
    $env:Path = $env:Path + ';' + $Bin   # dùng được ngay trong phiên này
    Ok 'Đã thêm vào PATH (cửa sổ khác cần mở lại)'
  } else {
    Nhac "Chưa thêm PATH. Gọi bằng đường dẫn đầy đủ: $Exe"
  }
} else {
  Ok 'bin đã nằm trong PATH'
}

Write-Host ''
& $Exe version
Write-Host '  Thử ngay:' -ForegroundColor Cyan
Write-Host '    sagent                       bảng tài khoản'
Write-Host '    sagent them claude:phu1      thêm tài khoản rồi đăng nhập'
Write-Host '    sagent verify                chạy bộ "đã đo"'
Write-Host '    sagent dash --set-password   đặt mật khẩu dashboard'
Write-Host ''
