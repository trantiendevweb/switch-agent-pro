<#
  kiem-tra.ps1 — đo lại trên máy bạn những giả định mà ccswitch đang đứng lên.

  Không phải test cho vui: mỗi mục ở đây từng là một chỗ có thể sai lặng lẽ.
  Bộ kiểm thoát với mã KHÁC 0 nếu có mục đỏ — để dùng được trong CI, và để
  không bao giờ có chuyện "chạy xong không thấy lỗi" trong khi nó chết giữa chừng.

  Chạy:  .\kiem-tra.ps1
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Continue'
try { [Console]::OutputEncoding = [Text.Encoding]::UTF8 } catch { }

$Goc  = Join-Path $env:USERPROFILE '.claude'
$Kho  = Join-Path $env:USERPROFILE '.claude-accounts'
$Tk   = Join-Path $env:USERPROFILE '.claude\skills\ccswitch\scripts\tk.ps1'
$TenThu = 'kiem-tra-tam'

$script:so_dat = 0
$script:so_hong = 0
$script:so_bo_qua = 0

function Dat($ten, $chiTiet)   { Write-Host ("  ✓ {0,-52} {1}" -f $ten, $chiTiet) -ForegroundColor Green;  $script:so_dat++ }
function Hong($ten, $chiTiet)  { Write-Host ("  ✗ {0,-52} {1}" -f $ten, $chiTiet) -ForegroundColor Red;    $script:so_hong++ }
function BoQua($ten, $chiTiet) { Write-Host ("  – {0,-52} {1}" -f $ten, $chiTiet) -ForegroundColor DarkGray; $script:so_bo_qua++ }

function Duong-Python() {
  foreach ($t in @('python', 'py', 'python3')) {
    $c = Get-Command $t -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($c -and $c.Source) { return $c.Source }
  }
  return $null
}

function Duong-Claude() {
  $c = Get-Command claude -ErrorAction SilentlyContinue
  if ($c) { return $c.Source }
  $n = Join-Path $env:USERPROFILE '.local\bin\claude.exe'
  if (Test-Path $n) { return $n }
  return $null
}

Write-Host ''
Write-Host '  Kiểm tra ccswitch' -ForegroundColor Cyan
Write-Host ''

# --- 0. Có đủ đồ nghề chưa ---------------------------------------------------
$py = Duong-Python
if ($py) { Dat 'Có Python' (Split-Path $py -Leaf) } else { Hong 'Có Python' 'không thấy — cfg.py sẽ không chạy' }

$cl = Duong-Claude
if ($cl) { Dat 'Có Claude Code' (Split-Path $cl -Leaf) } else { Hong 'Có Claude Code' 'không thấy lệnh claude' }

if (Test-Path $Tk) { Dat 'Đã cài tk.ps1' '' } else { Hong 'Đã cài tk.ps1' 'chạy .\cai-dat.ps1 trước' }

# --- 1. claude.exe có đọc CLAUDE_CONFIG_DIR không -----------------------------
$exe = Join-Path $env:USERPROFILE '.local\bin\claude.exe'
if (Test-Path $exe) {
  findstr /m /c:"CLAUDE_CONFIG_DIR" $exe > $null 2>&1
  if ($LASTEXITCODE -eq 0) { Dat 'claude.exe đọc CLAUDE_CONFIG_DIR' 'tìm thấy chuỗi trong binary' }
  else { Hong 'claude.exe đọc CLAUDE_CONFIG_DIR' 'KHÔNG thấy chuỗi — cả công cụ dựa vào điều này' }
} else {
  BoQua 'claude.exe đọc CLAUDE_CONFIG_DIR' 'không tìm thấy file exe để soi'
}

# --- 2. Đặt biến thì Claude ghi cấu hình vào đúng thư mục đó ------------------
if ($cl) {
  $tam = Join-Path $env:TEMP ('ccswitch-kiem-' + [Guid]::NewGuid().ToString('N').Substring(0, 8))
  New-Item -ItemType Directory $tam -Force | Out-Null
  $cu = $env:CLAUDE_CONFIG_DIR
  try {
    $env:CLAUDE_CONFIG_DIR = $tam
    $ra = & $cl mcp list 2>&1 | Out-String
    if (Test-Path (Join-Path $tam '.claude.json')) { Dat 'CLAUDE_CONFIG_DIR=X thì Claude ghi X\.claude.json' '' }
    else { Hong 'CLAUDE_CONFIG_DIR=X thì Claude ghi X\.claude.json' 'không sinh file' }

    # Thư mục trắng thì phải KHÔNG thấy MCP của tài khoản gốc.
    if ($ra -match 'No MCP servers configured') { Dat 'Thư mục mới không thấy MCP của tài khoản khác' 'tách thật' }
    else { Hong 'Thư mục mới không thấy MCP của tài khoản khác' 'thấy MCP — KHÔNG tách' }
  } finally {
    if ($cu) { $env:CLAUDE_CONFIG_DIR = $cu } else { Remove-Item Env:\CLAUDE_CONFIG_DIR -ErrorAction SilentlyContinue }
    Remove-Item $tam -Recurse -Force -ErrorAction SilentlyContinue
  }
} else {
  BoQua 'CLAUDE_CONFIG_DIR=X thì Claude ghi X\.claude.json' 'không có claude'
}

# --- 3. Token nằm ở file trong thư mục cấu hình ------------------------------
if (Test-Path (Join-Path $Goc '.credentials.json')) {
  Dat 'Token nằm ở .credentials.json trong thư mục cấu hình' 'không phải Credential Manager'
} else {
  BoQua 'Token nằm ở .credentials.json trong thư mục cấu hình' 'tài khoản gốc chưa đăng nhập?'
}

# --- 4. Nguồn cấu hình là ~\.claude.json, không phải file trong .claude -------
$nhaJson  = Join-Path $env:USERPROFILE '.claude.json'
$trongJson = Join-Path $Goc '.claude.json'
if ((Test-Path $nhaJson) -and (Test-Path $trongJson) -and $py) {
  $dem = @"
import json, sys
def n(p):
    try:
        d = json.load(open(p, encoding='utf-8'))
        pr = d.get('projects') or {}
        return sum(1 for v in pr.values() if isinstance(v, dict) and v.get('hasTrustDialogAccepted'))
    except Exception:
        return -1
print(n(sys.argv[1]), n(sys.argv[2]))
"@
  $f = Join-Path $env:TEMP 'ccswitch-dem.py'
  $dem | Set-Content -LiteralPath $f -Encoding UTF8
  $kq = (& $py $f $nhaJson $trongJson) -split '\s+'
  Remove-Item $f -Force -ErrorAction SilentlyContinue
  if ([int]$kq[0] -ge [int]$kq[1]) {
    Dat 'Nguồn cấu hình đúng là ~\.claude.json' ("trust: nhà {0} / trong .claude {1}" -f $kq[0], $kq[1])
  } else {
    Hong 'Nguồn cấu hình đúng là ~\.claude.json' ("file trong .claude lại nhiều trust hơn ({0} < {1}) — kiểm tay" -f $kq[0], $kq[1])
  }
} else {
  BoQua 'Nguồn cấu hình đúng là ~\.claude.json' 'chỉ có một file, không phải so'
}

# --- 5. Vòng đời tài khoản + xoá không xuyên qua junction ---------------------
# Đây là phép đo quan trọng nhất: thư mục tài khoản toàn junction trỏ về .claude
# gốc, nên một lệnh xoá sai là mất sạch skill, plugin, lịch sử phiên.
if ((Test-Path $Tk) -and $py) {
  $acc = Join-Path $Kho $TenThu
  if (Test-Path $acc) { & $Tk xoa $TenThu *> $null }

  $moi = Join-Path $Goc 'plugins\CCSWITCH-KIEM-TRA.txt'
  $coThuMucPlugins = Test-Path (Join-Path $Goc 'plugins')
  if ($coThuMucPlugins) { 'file mồi của bộ kiểm' | Set-Content -LiteralPath $moi -Encoding UTF8 }
  $truoc = @(Get-ChildItem $Goc -Force -Recurse -ErrorAction SilentlyContinue).Count

  & $Tk them $TenThu -KhongMo *> $null

  if (Test-Path (Join-Path $acc '.claude.json')) {
    Dat 'Tạo được tài khoản mới' $TenThu

    # Gieo cấu hình có sạch không: không được lọt danh tính hay khoá gói cước.
    $ktr = @"
import json, sys
d = json.load(open(sys.argv[1], encoding='utf-8'))
xau = [k for k in d if k in ('oauthAccount', 'userID')
       or 'odelAccess' in k or 'asses' in k or 'enguin' in k]
pr = d.get('projects') or {}
print(len(d), len(pr), ','.join(xau) if xau else 'sach')
"@
    $f2 = Join-Path $env:TEMP 'ccswitch-ktr.py'
    $ktr | Set-Content -LiteralPath $f2 -Encoding UTF8
    $r = (& $py $f2 (Join-Path $acc '.claude.json')) -split '\s+'
    Remove-Item $f2 -Force -ErrorAction SilentlyContinue
    if ($r[2] -eq 'sach') { Dat 'Cấu hình gieo sang không lọt danh tính / gói cước' ("{0} khoá, {1} project" -f $r[0], $r[1]) }
    else { Hong 'Cấu hình gieo sang không lọt danh tính / gói cước' ("lọt: {0}" -f $r[2]) }

    if (-not (Test-Path (Join-Path $acc '.credentials.json'))) { Dat 'Tài khoản mới KHÔNG mang theo token của tài khoản cũ' '' }
    else { Hong 'Tài khoản mới KHÔNG mang theo token của tài khoản cũ' 'có .credentials.json — sai' }

    if ($coThuMucPlugins -and (Test-Path (Join-Path $acc 'plugins\CCSWITCH-KIEM-TRA.txt'))) {
      Dat 'Nối được thư mục dùng chung' 'thấy file mồi qua junction'
    }
  } else {
    Hong 'Tạo được tài khoản mới' 'không sinh .claude.json'
  }

  & $Tk xoa $TenThu *> $null
  $sau = @(Get-ChildItem $Goc -Force -Recurse -ErrorAction SilentlyContinue).Count

  if (-not (Test-Path $acc)) { Dat 'Xoá được tài khoản' '' } else { Hong 'Xoá được tài khoản' 'thư mục vẫn còn' }

  if ($coThuMucPlugins) {
    if (Test-Path $moi) { Dat 'Xoá tài khoản KHÔNG xuyên qua junction' 'file mồi trong .claude còn nguyên' }
    else { Hong 'Xoá tài khoản KHÔNG xuyên qua junction' 'FILE MỒI ĐÃ BỊ XOÁ — dữ liệu thật bị đụng' }
    Remove-Item $moi -Force -ErrorAction SilentlyContinue
  }
  if ($truoc -eq $sau) { Dat 'Số file trong .claude gốc không đổi' "$sau file" }
  else { Hong 'Số file trong .claude gốc không đổi' "trước $truoc, sau $sau" }
} else {
  BoQua 'Vòng đời tài khoản (tạo → kiểm → xoá)' 'chưa cài tk hoặc thiếu python'
}

# --- Tổng kết ----------------------------------------------------------------
Write-Host ''
Write-Host ("  Đạt {0}  ·  Hỏng {1}  ·  Bỏ qua {2}" -f $script:so_dat, $script:so_hong, $script:so_bo_qua)
Write-Host ''
if ($script:so_hong -gt 0) {
  Write-Host '  Có mục đỏ. Đừng dùng tiếp tới khi hiểu vì sao.' -ForegroundColor Red
  Write-Host ''
  exit 1
}
Write-Host '  Tất cả đều đạt.' -ForegroundColor Green
Write-Host ''
exit 0
