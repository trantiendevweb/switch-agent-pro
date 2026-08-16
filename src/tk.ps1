<#
  tk.ps1 — đổi qua lại nhiều tài khoản Claude Code trên Windows.

  NGUYÊN LÝ (không có phép thuật gì)
  Claude Code đọc biến môi trường CLAUDE_CONFIG_DIR để biết lấy cấu hình và token
  ở thư mục nào. Công cụ này tạo cho mỗi tài khoản một thư mục riêng trong
  %USERPROFILE%\.claude-accounts\<tên>, rồi trỏ biến đó vào trước khi chạy claude.

  Hệ quả: mỗi tài khoản đăng nhập MỘT LẦN, đổi qua lại không phải đăng nhập lại,
  không tài khoản nào thấy token của tài khoản khác.

  BA ĐIỀU ĐÃ ĐO 16/08/2026, đừng dựa vào trí nhớ:
  1. claude.exe có đọc CLAUDE_CONFIG_DIR (tìm thấy chuỗi trong binary).
  2. Đặt CLAUDE_CONFIG_DIR=X thì Claude ghi X\.claude.json và X không thấy MCP
     của tài khoản khác -> tách thật.
  3. Khi KHÔNG đặt biến, Claude dùng %USERPROFILE%\.claude.json, KHÔNG phải
     %USERPROFILE%\.claude\.claude.json. Máy có thể tồn tại cả hai, và file
     trong .claude là file lạc chưa trust project nào — gieo nhầm nó là mất
     sạch trust dialog mà không có gì báo.
#>

[CmdletBinding()]
param(
  [Parameter(Position=0)][string]$Lenh,
  [Parameter(Position=1)][string]$Ten,
  [switch]$XemTruoc,
  [switch]$KhongMo,
  [Parameter(ValueFromRemainingArguments=$true)][string[]]$ThamSo
)

$ErrorActionPreference = 'Stop'
# Cho tiếng Việt hiện đúng trong cửa sổ lệnh.
try { [Console]::OutputEncoding = [Text.Encoding]::UTF8 } catch { }

$Goc   = Join-Path $env:USERPROFILE '.claude'
$Kho   = Join-Path $env:USERPROFILE '.claude-accounts'
$CfgPy = Join-Path $PSScriptRoot 'cfg.py'

# Những thứ KHÔNG dùng chung: token và danh tính. Mọi thứ khác đều nối link.
$RIENG = @('.credentials.json', '.claude.json')

function Loi($m)   { Write-Host "  ✗ $m" -ForegroundColor Red; exit 1 }
function Nhac($m)  { Write-Host "  $m" -ForegroundColor Yellow }
function Ok($m)    { Write-Host "  ✓ $m" -ForegroundColor Green }
function Mo($m)    { Write-Host "  $m" -ForegroundColor DarkGray }

function Duong-Tai-Khoan($ten) { return (Join-Path $Kho $ten) }

# File danh tính của tài khoản gốc. Xem ghi chú (3) ở đầu file.
function Json-Goc() {
  $a = Join-Path $env:USERPROFILE '.claude.json'
  if (Test-Path -LiteralPath $a) { return $a }
  return (Join-Path $Goc '.claude.json')
}

# !! Bẫy đã trả giá: đừng đặt tên hàm trùng tên lệnh cần gọi !!
# PowerShell ưu tiên Function hơn Application, nên hàm tên "Python" mà gọi
# Get-Command python thì nó trả về CHÍNH NÓ, .Source rỗng, và lỗi báo ra là
# "expression after '&' ... not valid" — không hề nhắc gì tới đệ quy.
function Duong-Python() {
  foreach ($t in @('python', 'py', 'python3')) {
    $p = Get-Command $t -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($p -and $p.Source) { return $p.Source }
  }
  Loi 'Không tìm thấy python. cfg.py cần python để đọc .claude.json (PowerShell 5.1 chết vì khoá trùng hoa/thường).'
}

function La-Link($duong) {
  $it = Get-Item -LiteralPath $duong -Force -ErrorAction SilentlyContinue
  if (-not $it) { return $false }
  return [bool]($it.Attributes -band [IO.FileAttributes]::ReparsePoint)
}

# Nối phần dùng chung từ .claude gốc sang thư mục tài khoản.
# Thư mục dùng junction (mklink /J) vì junction KHÔNG đòi quyền quản trị.
# File thử symlink trước, không được thì hardlink, không được nữa thì chép.
function Noi-Phan-Dung-Chung($acc) {
  $them = 0
  foreach ($m in Get-ChildItem -LiteralPath $Goc -Force) {
    if ($RIENG -contains $m.Name) { continue }
    $dich = Join-Path $acc $m.Name
    if (Test-Path -LiteralPath $dich) { continue }
    if ($m.PSIsContainer) {
      cmd /c mklink /J "$dich" "$($m.FullName)" > $null 2>&1
    } else {
      cmd /c mklink "$dich" "$($m.FullName)" > $null 2>&1
      if (-not (Test-Path -LiteralPath $dich)) { cmd /c mklink /H "$dich" "$($m.FullName)" > $null 2>&1 }
      if (-not (Test-Path -LiteralPath $dich)) { Copy-Item -LiteralPath $m.FullName -Destination $dich }
    }
    if (Test-Path -LiteralPath $dich) { $them++ }
  }
  return $them
}

function Email-Cua($acc) {
  $f = Join-Path $acc '.claude.json'
  if (-not (Test-Path -LiteralPath $f)) { return '(chưa đăng nhập)' }
  $e = (& (Duong-Python) $CfgPy email $f)
  if (-not $e -or $e -eq '-') { return '(chưa đăng nhập)' }
  return $e
}

function Email-Goc() {
  $e = (& (Duong-Python) $CfgPy email (Json-Goc))
  if (-not $e -or $e -eq '-') { return '(không đọc được)' }
  return $e
}

function Co-Token($acc) { return (Test-Path -LiteralPath (Join-Path $acc '.credentials.json')) }

function Danh-Sach() {
  if (-not (Test-Path $Kho)) { return @() }
  return @(Get-ChildItem -LiteralPath $Kho -Directory -ErrorAction SilentlyContinue | Sort-Object Name)
}

function Lenh-Claude() {
  $c = Get-Command claude -ErrorAction SilentlyContinue
  if ($c) { return $c.Source }
  $n = Join-Path $env:USERPROFILE '.local\bin\claude.exe'
  if (Test-Path $n) { return $n }
  Loi 'Không tìm thấy lệnh claude. Cài Claude Code trước đã.'
}

# Tham số thứ hai KHÔNG được đặt tên $args: đó là biến tự động của PowerShell
# (chứa mọi đối số truyền vào hàm), đặt trùng thì tham số bị ghi đè và lệnh
# claude nhận được một mớ đối số méo mó. Đã dính 16/08/2026.
function Chay-Bang($acc, $doiSo) {
  if ($acc) {
    $env:CLAUDE_CONFIG_DIR = $acc
    Mo "tài khoản: $(Split-Path $acc -Leaf)   ($acc)"
  } else {
    # Tài khoản gốc: phải XOÁ biến, không được trỏ vào %USERPROFILE%\.claude
    # (trỏ vào là Claude đọc file lạc trong đó, mất 14 project đã trust).
    Remove-Item Env:\CLAUDE_CONFIG_DIR -ErrorAction SilentlyContinue
    Mo 'tài khoản: gốc'
  }
  $exe = Lenh-Claude
  if ($doiSo -and $doiSo.Count -gt 0) { & $exe @doiSo } else { & $exe }
}

# ============================== các lệnh =====================================

function In-Bang() {
  $ds = Danh-Sach
  $dangDung = $env:CLAUDE_CONFIG_DIR
  Write-Host ''
  Write-Host '  Tài khoản Claude Code trên máy này' -ForegroundColor Cyan
  Write-Host ''
  $i = 0
  foreach ($d in $ds) {
    $i++
    $tok = 'chưa đăng nhập'
    $mau = 'Yellow'
    if (Co-Token $d.FullName) { $tok = 'sẵn sàng'; $mau = 'Green' }
    $dau = ' '
    if ($dangDung -and ($dangDung.TrimEnd('\') -eq $d.FullName.TrimEnd('\'))) { $dau = '*' }
    Write-Host ('   {0}{1,2}  {2,-12} {3,-34} ' -f $dau, $i, $d.Name, (Email-Cua $d.FullName)) -NoNewline
    Write-Host $tok -ForegroundColor $mau
  }
  $dauGoc = ' '
  if (-not $dangDung) { $dauGoc = '*' }
  Write-Host ('   {0} 0  {1,-12} {2,-34} ' -f $dauGoc, 'gốc', (Email-Goc)) -NoNewline
  Write-Host 'sẵn sàng' -ForegroundColor Green
  if ($ds.Count -eq 0) {
    Write-Host ''
    Nhac 'Chưa có tài khoản phụ nào. Bấm t để thêm.'
  }
  return $ds
}

function Bang-Chon() {
  # Không có bàn phím (chạy trong script, CI, hay bị chuyển hướng đầu vào)
  # thì in trợ giúp thay vì treo chờ gõ.
  if ([Console]::IsInputRedirected) { Lenh-Giup; return }

  while ($true) {
    $ds = In-Bang
    Write-Host ''
    Write-Host '   Gõ số để mở  ·  t thêm  ·  d đồng bộ  ·  x xoá  ·  ? trợ giúp  ·  Enter thoát' -ForegroundColor DarkGray
    Write-Host ''
    $chon = Read-Host '   Chọn'
    $chon = $chon.Trim()

    if ($chon -eq '')  { return }
    if ($chon -eq '?') { Lenh-Giup; return }
    if ($chon -eq 'd') { Lenh-Dong-Bo $false; Write-Host ''; continue }
    if ($chon -eq 't') {
      $ten = (Read-Host '   Tên tài khoản mới (chữ thường, không dấu)').Trim()
      if ($ten) { Lenh-Them $ten }
      return
    }
    if ($chon -eq 'x') {
      $ten = (Read-Host '   Xoá tài khoản tên gì').Trim()
      if ($ten) { Lenh-Xoa $ten }
      Write-Host ''
      continue
    }
    if ($chon -match '^\d+$') {
      $so = [int]$chon
      if ($so -eq 0) { Chay-Bang $null @(); return }
      if ($so -ge 1 -and $so -le $ds.Count) { Chay-Bang $ds[$so - 1].FullName @(); return }
      Nhac "Không có số $so trong bảng."
      continue
    }
    # gõ thẳng tên cũng được
    $acc = Duong-Tai-Khoan $chon
    if (Test-Path $acc) { Chay-Bang $acc @(); return }
    Nhac "Không hiểu '$chon'."
  }
}

function Lenh-Ds() { In-Bang | Out-Null; Write-Host '' }

function Lenh-Them($ten) {
  if (-not $ten) { Loi 'Thiếu tên. Ví dụ: tk them chinh' }
  if ($ten -notmatch '^[a-z0-9][a-z0-9-]*$') { Loi 'Tên chỉ dùng chữ thường, số và dấu gạch ngang. Ví dụ: phu1' }
  $acc = Duong-Tai-Khoan $ten
  if (Test-Path $acc) { Loi "Đã có tài khoản '$ten'. Chạy: tk $ten" }
  if (-not (Test-Path $Goc)) { Loi "Không thấy $Goc" }

  New-Item -ItemType Directory -Path $acc -Force | Out-Null
  $sl = Noi-Phan-Dung-Chung $acc
  & (Duong-Python) $CfgPy gieo (Json-Goc) (Join-Path $acc '.claude.json')
  if ($LASTEXITCODE -ne 0) { Loi 'Gieo cấu hình thất bại.' }
  Ok "Đã tạo tài khoản '$ten'"
  Mo "nối $sl mục dùng chung, giữ nguyên trust của các project đã mở"
  Write-Host ''
  if ($KhongMo) { Nhac "Chưa mở Claude Code (-KhongMo). Khi nào muốn đăng nhập thì gõ: tk $ten"; return }

  Write-Host '  Đang mở Claude Code. Làm một lần rồi thôi:' -ForegroundColor Cyan
  Write-Host '    1. Nó sẽ hỏi đăng nhập'
  Write-Host '    2. Đăng nhập bằng tài khoản Claude MỚI (không phải tài khoản cũ)'
  Write-Host '    3. Xong thì gõ  /exit  để quay lại đây'
  Write-Host ''
  Start-Sleep -Milliseconds 800
  Chay-Bang $acc @()

  Write-Host ''
  if (Co-Token $acc) { Ok "Xong. Từ giờ gõ  tk $ten  là vào thẳng, không phải đăng nhập lại." }
  else { Nhac "Chưa thấy token trong tài khoản '$ten' — có vẻ chưa đăng nhập xong. Gõ  tk $ten  để làm tiếp." }
}

function Lenh-Dong-Bo($xemTruoc) {
  # Vì sao cần: gieo chỉ chạy MỘT LẦN lúc tạo tài khoản. Sau này mở project mới,
  # bấm trust dialog, bật MCP cho project đó ở tài khoản A thì tài khoản B không
  # hề biết — đổi sang lại phải bấm lại từ đầu.
  if ((Danh-Sach).Count -eq 0) { Nhac 'Chưa có tài khoản phụ nào để đồng bộ.'; return }
  Write-Host ''
  if ($xemTruoc) { & (Duong-Python) $CfgPy dong-bo (Json-Goc) $Kho --xem-truoc; return }

  & (Duong-Python) $CfgPy dong-bo (Json-Goc) $Kho
  foreach ($d in Danh-Sach) {
    $n = Noi-Phan-Dung-Chung $d.FullName
    if ($n -gt 0) { Mo ("{0,-14} nối thêm {1} mục dùng chung" -f $d.Name, $n) }
  }
}

function Lenh-Xoa($ten) {
  if (-not $ten) { Loi 'Thiếu tên. Ví dụ: tk xoa phu1' }
  $acc = Duong-Tai-Khoan $ten
  if (-not (Test-Path $acc)) { Loi "Không có tài khoản '$ten'" }

  # !! CHỖ NGUY HIỂM NHẤT CỦA CẢ CÔNG CỤ !!
  # Thư mục này chứa junction trỏ về .claude gốc. Remove-Item -Recurse có thể
  # xuyên qua junction xoá luôn dữ liệu THẬT. Nên: gỡ từng link trước, kiểm lại
  # không còn link nào, rồi mới xoá phần còn lại.
  foreach ($m in Get-ChildItem -LiteralPath $acc -Force) {
    if (La-Link $m.FullName) {
      if ($m.PSIsContainer) { cmd /c rmdir "$($m.FullName)" > $null 2>&1 }
      else { [IO.File]::Delete($m.FullName) }
    }
  }
  $con = @(Get-ChildItem -LiteralPath $acc -Force -Recurse -ErrorAction SilentlyContinue | Where-Object { La-Link $_.FullName })
  if ($con.Count -gt 0) { Loi ("Còn {0} link chưa gỡ, không dám xoá đệ quy. Xem tay: {1}" -f $con.Count, $acc) }
  Remove-Item -LiteralPath $acc -Recurse -Force
  Ok "Đã xoá tài khoản '$ten' và token của nó."
}

function Lenh-Giup() {
  Write-Host ''
  Write-Host '  tk — đổi qua lại nhiều tài khoản Claude Code' -ForegroundColor Cyan
  Write-Host ''
  Write-Host '    tk                      bảng chọn (gõ số là vào)'
  Write-Host '    tk <tên>                chạy Claude Code bằng tài khoản đó'
  Write-Host '    tk goc                  chạy bằng tài khoản gốc'
  Write-Host '    tk them <tên>           tạo tài khoản mới rồi đăng nhập'
  Write-Host '    tk ds                   liệt kê'
  Write-Host '    tk dong-bo [-XemTruoc]  chép cấu hình dùng chung sang mọi tài khoản'
  Write-Host '    tk xoa <tên>            xoá tài khoản và token'
  Write-Host ''
  Write-Host '  Chạy lần đầu:' -ForegroundColor Cyan
  Write-Host '    tk them chinh    rồi    tk them phu1'
  Write-Host ''
  Mo "kho tài khoản : $Kho"
  Mo "nguồn cấu hình: $(Json-Goc)"
  Write-Host ''
}

$conLai = @()
foreach ($x in @($Ten) + @($ThamSo)) { if ($x) { $conLai += $x } }

switch -Regex ($Lenh) {
  '^$'                      { Bang-Chon; break }
  '^(giup|help|-h|--help|\?)$' { Lenh-Giup; break }
  '^(ds|list)$'             { Lenh-Ds; break }
  '^(them|add)$'            { Lenh-Them $Ten; break }
  '^(dong-bo|sync)$'        { Lenh-Dong-Bo $XemTruoc.IsPresent; break }
  '^(xoa|remove)$'          { Lenh-Xoa $Ten; break }
  '^goc$'                   { Chay-Bang $null $conLai; break }
  default {
    $acc = Duong-Tai-Khoan $Lenh
    if (-not (Test-Path $acc)) { Loi "Không có tài khoản '$Lenh'. Gõ  tk  để xem bảng, hoặc  tk them $Lenh  để tạo." }
    Chay-Bang $acc $conLai
  }
}
