# Sổ tay gỡ rối & khắc phục sự cố

Tài liệu này tổng hợp các sự cố **thực tế đã xảy ra** trong quá trình vận hành `sagent` và cách xử lý từng bước dành cho người **không rành kỹ thuật**.

Mọi câu lệnh trong tài liệu đều được viết và kiểm chứng trên **Windows PowerShell**.

---

## 1. Báo lỗi: "không mở được sổ trạng thái ... ở schema v5, bản sagent này chỉ biết tới v4"

### 🔴 Hiện tượng bạn nhìn thấy
Khi gõ bất kỳ lệnh `sagent` nào trong cửa sổ dòng lệnh, chương trình không chạy mà hiện thông báo lỗi:
```text
không mở được sổ trạng thái (C:\Users\...\.ai-accounts\state.db): ... ở schema v5, bản sagent này chỉ biết tới v4 — nâng cấp sagent, hoặc khôi phục bản sao lưu cũ: sagent db restore <file>
```

### 🔍 Nguyên nhân
- Máy tính của bạn đang chạy một file chương trình `sagent.exe` phiên bản cũ (chỉ hiểu cấu trúc dữ liệu v4), trong khi file dữ liệu `state.db` trên máy đã được tạo hoặc nâng cấp bởi một phiên bản `sagent` mới hơn (cấu trúc v5).
- `sagent` chủ động **từ chối chạy để bảo vệ dữ liệu của bạn**: nếu cho phép bản cũ ghi vào dữ liệu của bản mới, các thông tin mới sẽ bị làm hỏng hoặc xóa mất trong im lặng.

### 🛠️ Lệnh gõ để sửa

**Cách 1: Nâng cấp sagent lên bản mới nhất (Khuyên dùng)**
Mở PowerShell và chạy lệnh cài đặt tự động một dòng:
```powershell
iex (irm https://raw.githubusercontent.com/trantiendevweb/switch-agent-pro/main/install/get.ps1)
```

**Cách 2: Nếu muốn dùng tiếp bản cũ và quay về dữ liệu cũ tương thích**
1. Đảm bảo đã dừng toàn bộ tiến trình và trang quản lý:
   ```powershell
   sagent stop all
   ```
2. Khôi phục lại bản sao lưu tự động của schema cũ (chương trình luôn tự động sao lưu trước khi nâng cấp):
   ```powershell
   sagent db restore $HOME\.ai-accounts\state.db.bak-v4
   ```

### ✅ Cách kiểm đã sửa xong
Gõ lệnh:
```powershell
sagent
```
hoặc:
```powershell
sagent db
```
Nếu màn hình hiển thị danh sách tài khoản hoặc thông tin cơ sở dữ liệu bình thường, không còn dòng báo lỗi từ chối, nghĩa là sự cố đã được xử lý xong.

---

## 2. Lượt chạy kẹt "đang chạy" (running) mãi sau khi khởi động lại máy hoặc tắt tay

### 🔴 Hiện tượng bạn nhìn thấy
- Trên trang Dashboard hoặc khi gõ `sagent flow runs`, bạn thấy một lượt chạy (ví dụ lượt `#29` hoặc `#30`) cứ hiển thị trạng thái `running` ("đang chạy") mãi không dừng.
- Dù thực tế bạn đã tắt cửa sổ dòng lệnh, tắt máy tính hoặc máy tính vừa tự khởi động lại và không còn chương trình AI nào đang chạy.

### 🔍 Nguyên nhân
- Khi máy tính bị tắt đột ngột hoặc bạn đóng cửa sổ dòng lệnh ngang lúc flow đang làm việc, tiến trình bên ngoài đã dừng nhưng sổ trạng thái `state.db` chưa kịp nhận tín hiệu để đánh dấu kết thúc.
- Lượt chạy đó bị kẹt lại ở trạng thái `running` vĩnh viễn trong cơ sở dữ liệu nếu không được huỷ chủ động.

### 🛠️ Lệnh gõ để sửa
Dùng lệnh huỷ lượt chạy kèm số thứ tự (ID) của lượt chạy bị kẹt:
1. Xem danh sách các lượt chạy và tìm số ID bị kẹt:
   ```powershell
   sagent flow runs
   ```
2. Gõ lệnh huỷ lượt chạy đó (ví dụ lượt `#29`):
   ```powershell
   sagent flow huy 29
   ```
*(Thay `29` bằng số thứ tự lượt chạy bị kẹt trên máy của bạn).*

### ✅ Cách kiểm đã sửa xong
Gõ lại lệnh xem danh sách:
```powershell
sagent flow runs
```
Lượt chạy đó sẽ chuyển trạng thái sang `failed` kèm lý do rõ ràng (đã huỷ bởi người dùng), không còn kẹt ở trạng thái `running`.

---

## 3. Tài khoản Claude hết hạn hoặc đăng nhập dở dang (bảng báo sẵn sàng nhưng chạy thì hỏng)

### 🔴 Hiện tượng bạn nhìn thấy
- **Trường hợp A (Cổng kiểm tra chặn trước)**: Khi bạn gõ lệnh chạy quy trình (`sagent flow run`), hệ thống dừng ngay lập tức và in thông báo màu đỏ:
  ```text
  tài khoản claude:tns đã hết hạn token (từ ...), cần đăng nhập lại trước khi chạy flow
  ```
- **Trường hợp B (Đăng nhập dở dang)**: Bảng `sagent ds` có thể thấy tài khoản, nhưng khi bước làm việc của agent (`code-go`) khởi động thì văng lỗi xác thực ngay lập tức:
  ```text
  Failed to authenticate: OAuth session expired and could not be refreshed
  ```

### 🔍 Nguyên nhân
- **Hết hạn**: Mã đăng nhập (token) của Claude có thời hạn ngắn (khoảng 7,5 giờ). Nếu bạn để qua đêm hoặc lâu không dùng, token sẽ tự hết hạn.
- **Đăng nhập dở dang (`expiresAt = 0`)**: Bạn vừa thêm tài khoản mới hoặc bấm đăng nhập nhưng chưa hoàn tất xác thực trên trình duyệt web, file lưu trữ đã được tạo ra nhưng thời hạn sử dụng bằng 0 (chưa kích hoạt thành công).

### 🛠️ Lệnh gõ để sửa
Đăng nhập lại trực tiếp cho tài khoản bị hết hạn (ví dụ tài khoản `claude:tns`):
```powershell
sagent claude:tns
```
1. Trình duyệt web sẽ tự động mở trang đăng nhập của Claude.
2. Hoàn tất đăng nhập và bấm **Authorize** (Cho phép).
3. Quay lại cửa sổ PowerShell cho đến khi thấy thông báo đăng nhập thành công.

*(Mẹo: Nếu bạn biết rõ tài khoản hết hạn nhưng vẫn muốn chạy thử các bước không dùng tài khoản đó, bạn có thể thêm cờ `--cu-chay`: `sagent flow run dem --cu-chay`)*.

### ✅ Cách kiểm đã sửa xong
Gõ lệnh kiểm tra danh sách tài khoản:
```powershell
sagent ds
```
Tài khoản của bạn sẽ hiển thị trạng thái `sẵn sàng` (màu xanh), không còn nhãn `hết hạn` hay `chưa đăng nhập`.

---

## 4. Worktree chưa tạo hoặc bị dở dang khiến máy chấm báo FAIL mà không phải do code

### 🔴 Hiện tượng bạn nhìn thấy
- Trong nhật ký chạy flow, bước máy chấm (ví dụ `kiem-1` hoặc `kiem-cuoi` chạy lệnh `go test`) báo lỗi đỏ `FAIL` ngay lập tức.
- Máy chấm báo lỗi không tìm thấy thư mục làm việc hoặc không đọc được file, làm toàn bộ quy trình bị gián đoạn dù mã nguồn không hề có lỗi logic.

### 🔍 Nguyên nhân
- Mỗi agent khi làm việc song song sẽ được cấp một thư mục làm việc riêng biệt gọi là **git worktree** (tránh sửa đè lên thư mục chính).
- Nếu bước làm việc trước đó (`code-go`) bị chết sớm (ví dụ do tài khoản hết hạn như sự cố ở mục 3), thư mục worktree chưa kịp được tạo ra trên đĩa.
- Khi máy chấm nhảy vào kiểm tra thư mục đó, lệnh kiểm tra bị văng lỗi vì đường dẫn không tồn tại.
- Ngoài ra, nếu các lần chạy trước bị huỷ ngang để lại thư mục worktree dở dang chưa dọn sạch, các lệnh kiểm tra tiếp theo cũng có thể bị ảnh hưởng.

### 🛠️ Lệnh gõ để sửa
1. **Bước 1**: Đảm bảo tài khoản AI đã đăng nhập sẵn sàng (xem hướng dẫn ở Mục 3).
2. **Bước 2**: Dọn dẹp các thư mục worktree dở dang của tài khoản:
   ```powershell
   sagent clean claude:tns --force
   ```
   *(Lệnh này dọn sạch các thư mục tạm đang kẹt mà vẫn giữ nguyên toàn bộ các commit và nhánh git an toàn).*
3. **Bước 3**: Chạy lại quy trình:
   ```powershell
   sagent flow run dem
   ```

### ✅ Cách kiểm đã sửa xong
1. Gõ lệnh kiểm tra trạng thái các phiên:
   ```powershell
   sagent status
   ```
2. Quan sát nhật ký: agent tạo worktree thành công, tiến hành viết mã, và bước máy chấm thực thi `go test` trên đúng thư mục đó với kết quả xanh.
