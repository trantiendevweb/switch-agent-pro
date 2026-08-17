# get.ps1 - moi cai dat, danh cho:  iex (irm <url cua file nay>)
#
# TAI SAO CAN FILE NAY (da do, khong doan):
#
#   PowerShell 5.1 doc file .ps1 KHONG co BOM theo bang ma ANSI, nen chu tieng
#   Viet UTF-8 bien thanh dau nhay cong va script VO CU PHAP. Vi vay cai-dat.ps1
#   bat buoc phai co BOM.
#
#   Nhung `iex (irm <url>)` thi NGUOC LAI: irm tra ve chuoi bat dau bang U+FEFF
#   va Invoke-Expression khong parse duoc ky tu do. Da do:
#       iex (script khong BOM)  -> chay
#       iex (BOM + script)      -> hong
#
#   Mot file khong the thoa ca hai. File nay la loi thoat: ASCII THUAN, KHONG
#   BOM, KHONG param() - nen qua iex duoc - roi tai cai-dat.ps1 ve file tam va
#   goi bang `&`, luc do doc tu file nen BOM lai la thu can.
#
# DUNG THEM DAU TIENG VIET VAO FILE NAY. Do la ca ly do no ton tai.

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
try {
  [Net.ServicePointManager]::SecurityProtocol =
    [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
} catch { }

$url = 'https://raw.githubusercontent.com/trantiendevweb/switch-agent-pro/main/install/cai-dat.ps1'
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("cai-dat-" + [Guid]::NewGuid().ToString('N') + ".ps1")
try {
  Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $tmp

  # Qua iex thi khong truyen duoc tham so dong lenh, nen dung bien moi truong.
  $a = @{}
  if ($env:SAGENT_PHIEN)     { $a['Phien']    = $env:SAGENT_PHIEN }
  if ($env:SAGENT_TU_NGUON)  { $a['TuNguon']  = $true }
  if ($env:SAGENT_KHONG_HOI) { $a['KhongHoi'] = $true }

  & $tmp @a
} finally {
  Remove-Item -Force $tmp -ErrorAction SilentlyContinue
}

# LUU Y ve cach goi: PHAI dung `iex (irm <url>)`, KHONG dung `irm <url> | iex`.
# Da do: dang pipe hong voi "Cannot bind argument to parameter 'Command' because
# it is an empty string" roi parse loi tu giua file, trong khi dang ngoac don
# chay dung. Cung mot noi dung, khac moi cach dua vao iex.
