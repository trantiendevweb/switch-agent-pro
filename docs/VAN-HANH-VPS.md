# Vận hành VPS — sổ tra lúc đang có sự cố

> Chốt ngày **21/08/2026**. Đây là kết quả điều tra của **đúng một ngày**, gom lại để
> lần sau **không phải điều tra lại từ đầu**.
>
> Hôm 21/08 mất khoảng **20 phút** chỉ để tìm ra "IP public không vô được" là do
> **một tiến trình chết**, và mất thêm rất nhiều lượt nữa để **bác bỏ bốn giả thuyết sai**.
> Bốn giả thuyết đó nằm ở mục C — đọc trước khi bạn định đi lại đường cũ.

**Tài liệu này KHÔNG nói về lỗi của `sagent` khi chạy lệnh.** Những thứ đó đã có chỗ riêng,
đừng tra ở đây:

| Bạn đang gặp | Tra ở đâu |
|---|---|
| `sagent` từ chối chạy vì `state.db` ở schema mới hơn | [`KHAC-PHUC-SU-CO.md`](KHAC-PHUC-SU-CO.md) §1 |
| Lượt chạy kẹt `running` mãi sau reboot | [`KHAC-PHUC-SU-CO.md`](KHAC-PHUC-SU-CO.md) §2 |
| Tài khoản Claude hết hạn token / `expiresAt = 0` | [`KHAC-PHUC-SU-CO.md`](KHAC-PHUC-SU-CO.md) §3 |
| Worktree dở dang làm máy chấm báo FAIL, cần dọn | [`KHAC-PHUC-SU-CO.md`](KHAC-PHUC-SU-CO.md) §4 |
| `sagent dash` từ chối chạy vì chưa đặt mật khẩu | [`BAT-DAU.md`](BAT-DAU.md) §4 |

Ở đây chỉ có thứ **KHAC-PHUC-SU-CO.md không phủ**: máy chết, cổng chết, tiến trình chết,
watchdog, và những cái bẫy ở tầng máy chủ.

---

## A. Ba triệu chứng — checklist chạy được ngay

### A1. "IP public không vô được nữa"

**Hầu như luôn là dashboard `sagent dash` cổng 8788 chết.** Không phải mạng hỏng,
không phải firewall hỏng.

> 🪤 **Cái bẫy lớn nhất của mục này: ĐỪNG dùng `Get-Process sagent` để kết luận.**
> Máy này **luôn** có nhiều `sagent.exe` khác đang chạy — đó là các **phiên agent**
> (ví dụ `sagent.exe claude:phu`). Thấy có `sagent` trong danh sách tiến trình
> **KHÔNG** có nghĩa là dashboard đang chạy. Đã mất thời gian vì đúng cái bẫy này.

**Cách đúng — hỏi ai đang GIỮ CỔNG, không hỏi ai đang chạy:**

```powershell
Get-NetTCPConnection -LocalPort 8788 -State Listen
```

- Có dòng trả về → dash sống. Không có dòng nào → dash chết, đây chính là nguyên nhân.
- Dash bind **dual-stack**, nên cột `LocalAddress` hiện `::`. **Đừng lọc theo `LocalAddress`**,
  lọc là mất kết quả và bạn sẽ kết luận sai chiều.
- Cột `OwningProcess` cho PID thật — đó mới là bằng chứng.

**Hai cái bẫy phụ, cùng gây kết luận sai:**

1. **Dash chạy HTTPS, không phải HTTP.** Gõ `http://...:8788` sẽ nhận **lỗi 400**.
   Lỗi 400 ở đây **không** có nghĩa server hỏng — nó có nghĩa bạn gõ sai giao thức.
   Phải là `https://`.
2. **"Vào được web khác" KHÔNG loại trừ được triệu chứng này.** Ngày 21/08 các cổng
   **80 / 443 / 3000 / 8080 / 8090 vẫn mở bình thường** trong khi 8788 đã chết.
   Cổng khác sống chỉ chứng minh máy còn sống và mạng còn thông — không chứng minh gì
   về 8788.

**Trình tự tra, đúng thứ tự:**

```powershell
# 1. Cổng 8788 có ai giữ không? (câu hỏi duy nhất cần trả lời trước)
Get-NetTCPConnection -LocalPort 8788 -State Listen

# 2. Nếu rỗng: các cổng khác thế nào — để biết là "chỉ dash chết" hay "máy chết"
Get-NetTCPConnection -State Listen | Where-Object LocalPort -in 80,443,3000,8080,8090,8788

# 3. Đối chiếu PID giữ cổng với tiến trình (KHÔNG làm ngược lại)
Get-Process -Id (Get-NetTCPConnection -LocalPort 8788 -State Listen).OwningProcess
```

Nếu bước 1 rỗng mà bước 2 vẫn đủ cổng khác → **chỉ dash chết**, watchdog (mục B) đáng lẽ
phải dựng lại nó; nếu nó không dựng lại thì lỗi nằm ở watchdog, xem bẫy em-dash ở mục D.

---

### A2. "Rớt Claude remote"

**Hỏi "máy có reboot không?" TRƯỚC.** Đây là câu hỏi rẻ nhất và loại được nhiều thứ nhất —
xem A3 để tra reboot.

**Nhật ký cần đọc:**

```text
C:\Users\Administrator\SEO Project\projects\tainguyenseo\.rc-script.log
```

Đó là nhật ký của watchdog, có **PID và mốc giờ của từng dự án**. Đây là nguồn duy nhất
nói được "server chết lúc mấy giờ" và "ai dựng nó lại".

**Env ID hiện tại của cả 4 dự án** nằm ở `.rc-env-state.json`, cùng thư mục với script watchdog.

> ✅ **ĐÃ ĐO VÀ BÁC BỎ: restart tiến trình KHÔNG làm đổi `environment` ID.**
>
> Thí nghiệm 21/08 — giết KNOWLEDGE-OS lúc **10:43:55**, nó sống lại lúc **10:44:03**
> với **PID mới 14120**, `environment` vẫn nguyên `env_01VzrLwcnwKuD4cJ7C1BqkAC`.
>
> **Hệ quả thực dụng:** khi watchdog cứu server, **link trên điện thoại KHÔNG chết.**
> Đừng đi tạo lại link, đừng đi báo người dùng đổi link. Chỉ cần chờ nhịp watchdog
> (~60 giây, xem mục B).

---

### A3. Máy tự khởi động lại

```powershell
Get-WinEvent -FilterHashtable @{LogName='Application'; Id=1015}
```

**Event 1015 của Wininit nói thẳng:** `lsass.exe failed with status code 1`.
Không cần suy diễn — Windows đã tự khai nguyên nhân reboot.

| Lần | Thời điểm |
|---|---|
| 1 | **19/08 01:47** |
| 2 | **20/08 20:24:03** |

**Nghi phạm: brute-force** — khoảng **3.885 lần thử/giờ** vào tài khoản `Administrator`
(dấu chấm là phân cách nghìn).

**CHƯA chứng minh được, vì chưa có dump của lsass.** Đây là "nghi phạm", không phải
"nguyên nhân". Đừng ghi vào báo cáo như thể đã kết luận.

---

## B. Kiến trúc watchdog — cái gì canh cái gì

| Thành phần | Chi tiết |
|---|---|
| Script | `start-remote-control.ps1`, ở thư mục `tainguyenseo` |
| Gọi bởi | **HAI** Scheduled Task |
| Task 1 | **"TNS Remote Control"** — kích hoạt theo **logon** |
| Task 2 | **"TNS Remote Control Boot"** — kích hoạt theo **boot**, chạy **S4U** |
| Chu kỳ lặp | Cả hai đều **PT2M** |
| **Nhịp hiệu dụng** | **~60 giây** — vì hai task **lệch pha** nhau |
| Chống giẫm chân | Mutex `Global\TNS-Remote-Control` — chặn hai task cùng chạy một lúc |
| Nó canh | **4 server `claude remote-control`** + **dashboard sagent cổng 8788** |

Hai điểm dễ hiểu nhầm:

- **PT2M không phải nhịp thật.** Đọc task thấy 2 phút rồi kết luận "tối đa 2 phút mới cứu"
  là sai — hai task lệch pha nên thực tế **~60 giây**.
- **Mutex tồn tại là có lý do.** Nếu bạn thêm task thứ ba hoặc chạy tay script này,
  nó sẽ bị mutex chặn chứ không chạy song song. Đó là hành vi đúng, không phải lỗi.

---

## C. Bốn giả thuyết ĐÃ BỊ BÁC — đừng đi lại

Mỗi dòng dưới đây đã tốn thời gian thật. Bác bỏ rồi thì thôi.

| # | Giả thuyết | Phản chứng cụ thể |
|---|---|---|
| 1 | "Server bật từ **session 0** thì chết yểu" | KNOWLEDGE-OS **PID 13700** bật từ session 0, **sống 29 tiếng**. |
| 2 | "**ClipSVC** dừng thì giết tiến trình MSIX" | ClipSVC **tắt mỗi ~5 phút, cả ngày**. Server sống xuyên qua **hàng trăm lần** tắt như thế. |
| 3 | "Gói **MSIX Claude servicing** thì giết sạch tiến trình" | Chữ ký service **19/08 15:49:55** **giống hệt** **21/08 06:42:30**. Lần đầu **4 server sống**, lần sau **cả 4 chết**. Cùng một nguyên nhân không thể cho hai kết quả ngược nhau. |
| 4 | "**Watchdog báo nhầm** do WMI trả rỗng" | Lấy mẫu **200 lượt** → **200/200 thấy đủ 4**. Watchdog không báo nhầm. |

### Kết luận trung thực

**Hai lần chết — 20/08 ~20:32 và 21/08 ~06:40 — tới nay VẪN CHƯA CÓ NGUYÊN NHÂN NÀO
ĐỨNG VỮNG.**

Không có giả thuyết thứ năm. **Đừng bịa một cái cho đỡ trống.** Nếu bạn có giả thuyết mới,
nó chỉ được ghi vào bảng trên khi kèm **phản chứng cụ thể** hoặc **bằng chứng cụ thể** —
đúng chuẩn của bốn dòng trên, không phải "có vẻ hợp lý".

---

## D. Bẫy đã biết

### D1. Em-dash trong file `.ps1` — bẫy nguy hiểm nhất, vì im lặng

`start-remote-control.ps1` là **UTF-8 KHÔNG BOM**. PowerShell 5.1 đọc nó bằng **CP1252**.
Hậu quả: dấu gạch ngang dài `—` nằm trong một **CHUỖI** biến thành **nháy cong đóng**
và **kết thúc chuỗi sớm** → **hỏng cú pháp**.

Khi đó **mất cả 4 server lẫn dashboard**. Và vì script **chạy ẩn** nên **không ai thấy lỗi** —
nhìn từ ngoài y hệt triệu chứng A1 + A2 xảy ra cùng lúc.

**Luôn kiểm trước khi lưu:**

```powershell
$err = $null
[System.Management.Automation.Language.Parser]::ParseFile(
    'C:\...\start-remote-control.ps1', [ref]$null, [ref]$err)
$err   # rỗng là sạch; có dòng nào là ĐỪNG lưu
```

Chuẩn an toàn cho script này: **không đặt ký tự Unicode "đẹp" (`—`, `"`, `"`, `'`, `'`)
vào trong chuỗi.** Dùng `-` thường.

### D2. `sagent fleet --copies 1` chạy hai lần KHÔNG cho hai worktree

Chạy `sagent fleet --copies 1` hai lần **không** tạo ra hai worktree độc lập —
**cả hai rơi vào bản clone số 1 và giẫm lên nhau.**

- **Muốn hai việc khác nhau chạy song song → dùng HAI TÀI KHOẢN khác nhau.**
- **Kiểm ngay** bằng `sagent status`, **nhìn cột worktree**. Hai dòng cùng worktree
  nghĩa là chúng đang ghi đè nhau.

(Đây là bẫy khác với worktree dở dang do bước trước chết — cái đó và lệnh dọn
`sagent clean` nằm ở [`KHAC-PHUC-SU-CO.md`](KHAC-PHUC-SU-CO.md) §4.)

### D3. ~~Nhật ký phiên fleet KHÔNG truy được sau khi phiên kết thúc~~ — ✅ ĐÃ VÁ 21/08

> **Bẫy này đã được gỡ**, bản vá vào `main` lúc 21/08 chiều (merge `417c133`, gốc
> `a8bd2e6`). Mô tả cũ giữ lại bên dưới vì nó vẫn đúng với **mọi phiên chạy bằng
> bản `sagent` trước bản vá** — nhật ký của các phiên đó đã mất thật, không lấy lại được.

**Nay nhật ký nằm ở `~/.ai-accounts/.nhat-ky/`**, ngang hàng `state.db` — tức là **sống lâu
hơn cả worktree lẫn thư mục clone**, và nằm ngoài repo của người dùng.

```powershell
sagent nhat-ky            # các phiên gần đây, còn đọc lại được không
sagent nhat-ky <số phiên> # đọc lại một phiên ĐÃ KẾT THÚC
```

Ba chỗ hỏng cũ đã được đóng, ghi ra để không ai dựng lại:

1. Đường dẫn log cũ là `<thư mục clone>/fleet.log` — **chỉ phụ thuộc số bản clone**. Đọc
   `state.db` thật: **20 phiên gần nhất chỉ ứng với 6 đường dẫn log**. Sáu phiên
   `claude:tns#1` cùng trỏ vào **một** file; `sagent nhat-ky 167` trả về nội dung của
   phiên **#173** đang chạy. Bằng chứng của #167 **không còn tồn tại**.
2. `os.Create` **cắt trắng** file mỗi lần bật phiên mới.
3. `sagent clean` **xoá nguyên thư mục clone** — lệnh dọn dẹp sau một lượt hỏng chính là
   lệnh phá tang chứng.

Tên file nay mang **địa chỉ + mốc thời gian tới mili giây**, không mang số phiên — vì số
phiên chỉ được cấp **sau** khi tiến trình đã bật, và đổi tên một file đang mở thì hỏng trên
Windows. Ánh xạ "số phiên → đường dẫn" nằm ở cột `log` của sổ.

**Dù vậy, thói quen cũ vẫn giữ nguyên: mọi phiên fleet vẫn phải được YÊU CẦU ghi báo cáo
thành FILE trong repo rồi COMMIT.** Nhật ký nói *agent đã nói gì*; `git` nói *mã đã đổi gì*.
Hai thứ đó khác nhau, và khi lệch nhau thì **tin `git`** (mục F). Nhật ký là để truy nguyên
một phiên hỏng, **không phải** để thay bằng chứng.

---

## E. Rủi ro còn treo — **ĐÃ ĐO LẠI 21/08 lúc ~14:20**

> ⚠ **Bảng cũ có BỐN dòng đều ghi "chưa xử lý". Đo lại thì HAI trong bốn dòng đó
> đã sai** — không phải vì ai vá lén, mà vì sổ chép lại một trạng thái cũ rồi
> không ai đo lại. Sổ ghi **khẳng định** về một thứ nó đã thôi không đo nữa, và
> đó là kiểu sai nguy hiểm hơn cả bỏ trống.

| Rủi ro | Trạng thái (đo 21/08 ~14:20) | Bằng chứng |
|---|---|---|
| ~~Backup không thật sự rời khỏi máy~~ | ✅ **ĐÃ XỬ LÝ 21/08 20:47 — có số đo** | **Nguyên nhân gốc**: `OneDrive.exe` đã bị gỡ khỏi máy, mà thư mục `OneDrive` vẫn giữ **reparse point mồ côi** (thẻ `0x9000701a`, cloud placeholder) của lần cài cũ — trong khi `SyncRootManager` trong registry đã bị xoá. Client mới thấy chỗ đã có chủ nên báo *'We can't add your OneDrive folder'*; `OneDrive.exe /reset` **không** gỡ được dấu này. **Cách sửa**: đổi tên thư mục cũ sang `OneDrive.cu-20260821` cho client dựng gốc đồng bộ sạch, đăng nhập lại, rồi chép 17 file `.enc` cục bộ sang. **Nghiệm thu bằng ĐỐI CHIẾU, không bằng biểu tượng**: lấy 17 file vốn chỉ nằm trên đĩa, đối chiếu với danh sách file đã có `ReparsePoint` trong thư mục mới → **thiếu 0**. 782.4 MB nay đã ở trên mây. |
| **`RunAsPPL` chưa bật** | ❌ **CÓ THẬT** | `HKLM:\SYSTEM\CurrentControlSet\Control\Lsa` **không có** giá trị `RunAsPPL`. `lsass` không được bảo vệ trước công cụ trộm credential. |
| ~~Hai task backup dùng `LogonType=Interactive`~~ | ✅ **SỔ SAI — thực tế là `S4U`** | Cả ba task đều `LogonType=S4U`, `RunLevel=Highest`: *KNOWLEDGE OS daily backup*, *Tainguyenseo website off-host backup*, *TNS OS off-host backup*. `S4U` **chạy được khi không ai đăng nhập**, nên lo ngại "reboot xong không ai RDP thì task không chạy" **không còn đúng**. Đối chứng: `knowledge-os-20260821-140001.enc` sinh lúc **14:00**, tức **sau** lần reboot 12:06. |
| ~~Brute-force chưa chặn~~ | ✅ **SỔ SAI — đang chặn thật** | Task **`TNS Chan Bruteforce`** chạy **mỗi 5 phút** (`chan-bruteforce-rdp.ps1 -PhutNhinLai 30 -NguongSai 20`), lần cuối **14:20:20**, `Result=0`. Rule tường lửa cùng tên **`Enabled=True, Action=Block`**, đang chặn **53 IP**. |

**Nhưng đừng đọc dòng cuối thành "đã yên".** Trong **một giờ** gần nhất vẫn có
**1.867 lượt đăng nhập thất bại** (event `4625`). So với mốc **~3.900 lượt/giờ**
đo sáng 21/08 thì việc chặn có tác dụng thật — nhưng cuộc tấn công **vẫn đang
diễn ra**, và kết luận ở mục A3 (**máy sẽ còn reboot**) **không đổi**.

**Việc còn phải làm, xếp theo hậu quả:**

1. ✅ **XONG 21/08.** Xem dòng đầu bảng trên. Ba điều đáng giữ lại từ lượt này:

   - **`/reset` không gỡ được reparse point mồ côi.** Đừng mất thời gian lặp lại nó.
     Dấu nằm ở tầng hệ thống tệp, không nằm ở cấu hình client.
   - **Phải kiểm Known Folder Move TRƯỚC khi đổi tên thư mục `OneDrive`.** Lần này
     `Desktop`/`Documents`/`Pictures`/`Downloads` đều trỏ về `C:\Users\<user>\...` nên
     đổi tên vô hại. Nếu chúng trỏ vào trong OneDrive thì đổi tên là mất màn hình nền.
   - **Giữ thư mục cũ cho tới khi đếm đủ.** `OneDrive.cu-20260821` vẫn còn; chỉ xoá
     sau khi đối chiếu xong. Trong một việc mà mọi đèn xanh đều từng nói dối, bản gốc
     là thứ rẻ nhất mua được.

2. ✅ **ĐÃ LÀM 21/08 — nó đã ĐỎ.** `tools/canh-backup.ps1` kiểm hai thứ:
   `OneDrive.exe` có tồn tại/có chạy không, và file `.enc` mới nhất có quá
   `-NguongGio` (mặc định 30) không. Có vấn đề thì **báo Telegram qua đúng đường
   `sagent tele` đã cấu hình** (không dựng thông tin đăng nhập mới) và **thoát
   khác 0**.

   Chạy định kỳ bằng task **`TNS canh backup roi may`** — hằng ngày 05:00, sau
   nhịp backup 04:00, `LogonType=S4U` nên chạy cả khi không ai đăng nhập.

   Xem thử ngay, không gửi tin: `.\tools\canh-backup.ps1 -Kho`

   Chạy thật lúc 21/08 16:16 → `LastTaskResult = 1`, tin đã gửi. **Đây là lần
   đầu tình trạng này phát ra tín hiệu** — trước đó nó im lặng suốt.

   Bài canh này **không sửa được gốc**. Nó chỉ bảo đảm lần sau không ai phải
   tình cờ phát hiện.
3. **Bật `RunAsPPL`** (cần khởi động lại máy).

## F. Nguyên tắc chẩn đoán của dự án này

- **Bằng chứng là PID GIỮ CỔNG, không phải TÊN TIẾN TRÌNH.** `Get-Process sagent` không
  chứng minh được gì (xem A1).
- **Mã HTTP không chứng minh được server nào đang trả lời.** 200 có thể từ server khác;
  400 có thể chỉ vì bạn gõ `http://` thay vì `https://`.
- **Tin `git` hơn tin lời khai của agent.** Phiên kết thúc là lời khai bay mất; commit thì ở lại.
- **Giả thuyết chỉ được coi là BÁC BỎ khi có phản chứng CỤ THỂ** — số đo, mốc giờ, PID.
  "Nghe không hợp lý" không phải phản chứng.
- **Không đo được thì ghi thẳng "không đo được" kèm lý do. Đừng đoán.**
  Ví dụ mẫu: brute-force là *nghi phạm*, chưa phải nguyên nhân, **vì chưa có dump**.
