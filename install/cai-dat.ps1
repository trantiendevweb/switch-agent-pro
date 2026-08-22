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
    -Go             GỠ: xoá binary, xoá alias, hỏi gỡ PATH

  Nó cài luôn hai tên gọi khác: `tk` và `ccswitch`. Gõ tên nào cũng chạy đúng
  một chương trình đó. Xem khối "ALIAS" bên dưới để biết vì sao làm bằng file
  .cmd chứ không phải bản sao binary.

  Yêu cầu: Windows + PowerShell 5.1 (có sẵn từ Windows 10). Không cần gì khác.
#>
[CmdletBinding()]
param(
  [switch]$TuNguon,
  [string]$Phien = '',
  [switch]$KhongHoi,
  [switch]$Go
)

$ErrorActionPreference = 'Stop'
try { [Console]::OutputEncoding = [Text.Encoding]::UTF8 } catch { }

# Thanh tiến trình của PowerShell 5.1 làm Invoke-WebRequest chậm đi NHIỀU LẦN vì
# nó vẽ lại sau mỗi khối dữ liệu. Tắt đi là mẹo tăng tốc lớn nhất ở đây.
$ProgressPreference = 'SilentlyContinue'

$Repo = 'trantiendevweb/switch-agent-pro'
$Bin  = Join-Path $env:USERPROFILE 'bin'
$Exe  = Join-Path $Bin 'sagent.exe'

# --------------------------------------------------------------------------
# ALIAS: `tk` va `ccswitch` chay dung nhu `sagent`.
#
# BA CACH LAM DUOC TREN WINDOWS, va vi sao chon cach nay:
#
#   (a) File .cmd nho ben canh binary       <- CHON
#   (b) Ban sao cua sagent.exe              (tk.exe, ccswitch.exe)
#   (c) Hard link toi sagent.exe
#
# (b) va (c) deu HONG THEO KIEU IM LANG khi nang cap. Ban cai dat thay
# sagent.exe bang mot FILE MOI (Move-Item), nen ban sao van la ban cu va hard
# link van tro toi noi dung cu. Nguoi dung go `tk` sau khi nang cap se chay
# binary phien ban cu ma khong mot dong nao noi ra. Do dung la lop loi du an
# nay so nhat: hai thu cung ten, lang le lech nhau.
#
# (a) khong co van de do: shim tro toi sagent.exe THEO DUONG DAN, nen no luon
# chay dung file dang nam do - ke ca khi ban tu `go build -o $Exe`.
#
# Gia phai tra cua (a), noi thang:
#   - moi lan goi qua alias ton them mot tien trinh cmd.exe;
#   - bam Ctrl+C vao mot lenh chay lau (`tk dash`) thi cmd hoi
#     "Terminate batch job (Y/N)?".
# Ca hai deu NHIN THAY DUOC. Chay nham phien ban thi khong.
#
# Vi sao KHONG doc os.Args[0] trong Go: doc ten binary chi can khi chuong
# trinh muon CU XU KHAC theo ten goi. O day ta muon nguoc lai - giong het.
# Them mot nhanh re theo ten la them mot duong cho hai loi goi lech nhau.
#
# Hai bien duoi la NGUON SU THAT DUY NHAT cho phan alias, va bai test
# TestShimAliasChayThat (cmd/sagent/alias_test.go) DOC THANG hai dong nay roi
# chay thu shim. Doi dinh dang hai dong nay thi sua ca test.
# --------------------------------------------------------------------------
$TenAlias = @('tk','ccswitch')
$ShimNoiDung = '@"%~dp0sagent.exe" %*'

function Cai-Alias {
  foreach ($ten in $TenAlias) {
    $p = Join-Path $Bin ($ten + '.cmd')
    # ASCII: shim khong co ky tu ngoai ASCII, va .cmd doc theo bang ma cua
    # console chu khong theo BOM. Ghi UTF-8 co BOM vao day thi dong dau tien
    # cua file bat dau bang rac va cmd bao "'ï»¿@' is not recognized".
    Set-Content -LiteralPath $p -Value $ShimNoiDung -Encoding ASCII
    Ok "alias: $ten"
  }
}

function Go-CaiDat {
  # Duong GO. `sagent xoa` KHONG lam viec nay: no la profile.remove, tuc xoa
  # mot TAI KHOAN trong so ho so, khong dinh gi toi binary tren dia.
  $daXoa = 0
  foreach ($ten in $TenAlias) {
    $p = Join-Path $Bin ($ten + '.cmd')
    if (Test-Path -LiteralPath $p) { Remove-Item -Force -LiteralPath $p; Ok "da xoa alias: $ten"; $daXoa++ }
  }
  if (Test-Path -LiteralPath $Exe) {
    try { Remove-Item -Force -LiteralPath $Exe; Ok "da xoa: $Exe"; $daXoa++ }
    catch { Nhac "khong xoa duoc $Exe (dang chay?). Dung dash/phien roi go lai." }
  }
  # Ban cu doi ten sau moi lan nang cap. Khong don thi chung nam lai mai mai.
  Get-ChildItem -LiteralPath $Bin -Filter 'sagent.exe.cu-*' -ErrorAction SilentlyContinue | ForEach-Object {
    try { Remove-Item -Force -LiteralPath $_.FullName; Ok ("da xoa ban cu: " + $_.Name); $daXoa++ } catch { }
  }
  if ($daXoa -eq 0) { Nhac "khong thay gi de go trong $Bin" }

  $u = [Environment]::GetEnvironmentVariable('Path', 'User')
  if ($u -and (($u -split ';' | Where-Object { $_ -and $_.TrimEnd('\') -eq $Bin.TrimEnd('\') }).Count -gt 0)) {
    $tra = 'k'
    if (-not $KhongHoi -and -not [Console]::IsInputRedirected) {
      $tra = (Read-Host "  Go $Bin khoi PATH? (c/k)").Trim().ToLower()
    }
    if ($tra -eq 'c') {
      $moi = ($u -split ';' | Where-Object { $_ -and $_.TrimEnd('\') -ne $Bin.TrimEnd('\') }) -join ';'
      [Environment]::SetEnvironmentVariable('Path', $moi, 'User')
      Ok 'da go khoi PATH (cua so khac can mo lai)'
    } else {
      Nhac "van giu $Bin trong PATH"
    }
  }
  Write-Host ''
  exit 0
}

function Ok($m)  { Write-Host "  ✓ $m" -ForegroundColor Green }
function Nhac($m){ Write-Host "  ! $m" -ForegroundColor Yellow }
function Loi($m) { Write-Host "  ✗ $m" -ForegroundColor Red; exit 1 }

Write-Host ''
Write-Host '  Cài sagent' -ForegroundColor Cyan
Write-Host ''

if ($env:OS -ne 'Windows_NT') { Loi 'sagent chỉ hỗ trợ Windows.' }

# Go truoc moi thu khac: khong tai gi, khong hoi GitHub, khong doi kien truc.
if ($Go) { Go-CaiDat }

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

# Ghi lai shim SAU MOI LAN CAI, ke ca khi file da co. Re (hai file vai chuc
# byte) va no dong luon lo hong duy nhat cua cach nay: ban cu co shim tro toi
# mot cai ten khac thi lan cai nay chua lai cho dung.
Cai-Alias

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
Write-Host '  Ba ten goi, mot chuong trinh:' -ForegroundColor Cyan
Write-Host ('    sagent  =  ' + ($TenAlias -join '  =  '))
Write-Host '  Go het:  .\install\cai-dat.ps1 -Go'
Write-Host ''
