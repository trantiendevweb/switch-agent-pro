# Hướng dẫn 5 phút đầu tiên cho người mới bắt đầu

Tài liệu này dành cho người **không rành kỹ thuật** muốn dùng nhiều tài khoản AI (như Claude, Cursor, Codex...) trên cùng một máy tính Windows mà không phải đăng xuất hay đăng nhập lại liên tục.

---

## 1. Cài đặt trong 1 phút

Bạn chỉ cần một máy tính chạy Windows 10 hoặc Windows 11. Không cần cài thêm phần mềm hỗ trợ nào khác và không cần quyền quản trị viên (Admin).

**Cách làm:**
1. Bấm phím **Start** (phím logo Windows trên bàn phím), gõ chữ `powershell`, rồi bấm **Enter** để mở cửa sổ dòng lệnh Windows PowerShell.
2. Sao chép đúng một dòng lệnh dưới đây, dán vào cửa sổ PowerShell rồi bấm **Enter**:

```powershell
iex (irm https://raw.githubusercontent.com/trantiendevweb/switch-agent-pro/main/install/get.ps1)
```

Chờ vài giây để chương trình tự động tải file chạy `sagent.exe` (khoảng 11 MB) và thiết lập sẵn sàng trên máy tính của bạn.

---

## 2. Xử lý cảnh báo "Windows protected your PC"

Khi bạn chạy chương trình lần đầu, hệ thống bảo vệ của Windows (SmartScreen) có thể hiện một khung cảnh báo màu xanh với dòng chữ **"Windows protected your PC"** (hoặc *Windows đã bảo vệ PC của bạn*).

### Cách xử lý:
1. Bấm vào dòng chữ **"More info"** (hoặc *Thông tin khác* / *Xem thêm*).
2. Bấm tiếp nút **"Run anyway"** (hoặc *Vẫn chạy*).

### Vì sao có cảnh báo này?
Bạn hoàn toàn có thể yên tâm:
- `sagent` là phần mềm **mã nguồn mở hoàn toàn miễn phí**. Để không bị cảnh báo này, phần mềm phải mua một chứng chỉ số thương mại rất đắt tiền từ Microsoft. Các dự án miễn phí thường không mua chứng chỉ này.
- Windows chưa thấy file này xuất hiện phổ biến trên hàng triệu máy nên hiển thị hỏi để bạn xác nhận.
- Trình cài đặt đã tự động kiểm tra mã băm an toàn (SHA-256) trước khi lưu vào máy, đảm bảo file nguyên bản từ nhà phát triển và không bị can thiệp.

---

## 3. Ba bước đầu tiên: Gõ gì và Thấy gì

Sau khi cài đặt xong, bạn thao tác trực tiếp trên cửa sổ PowerShell:

### Bước 1: Xem các tài khoản đang có trên máy
**Gõ lệnh:**
```powershell
sagent
```
*(hoặc gõ `sagent ds`)*

**Bạn sẽ thấy gì:**
Một bảng danh sách các tài khoản AI trên máy. Nếu trước đó bạn đã từng đăng nhập Claude trên máy tính này, chương trình sẽ tự động nhận diện tài khoản gốc và hiển thị trạng thái `sẵn sàng`.

---

### Bước 2: Thêm tài khoản mới (để dùng song song)
Ví dụ bạn muốn thêm một tài khoản phụ (hoặc tài khoản công ty) tên là `phu2`:

**Gõ lệnh:**
```powershell
sagent them claude:phu2
```

**Bạn sẽ thấy gì:**
Trình duyệt web sẽ mở ra trang đăng nhập của nhà cung cấp AI (hoặc hiển thị đường link để bạn đăng nhập). Bạn chỉ cần hoàn tất đăng nhập. Tài khoản mới sẽ được lưu trữ riêng biệt, hoàn toàn không ảnh hưởng hay đè lên tài khoản đầu tiên.

---

### Bước 3: Mở AI lên làm việc bằng tài khoản bạn muốn
Vào thư mục dự án bạn cần làm việc, rồi gõ lệnh:

**Gõ lệnh:**
- Để làm việc bằng tài khoản mới vừa tạo:
  ```powershell
  sagent claude:phu2
  ```
- Hoặc để quay lại dùng tài khoản gốc ban đầu:
  ```powershell
  sagent goc
  ```

**Bạn sẽ thấy gì:**
Giao diện làm việc của AI sẽ khởi động ngay lập tức với đúng tài khoản bạn đã chọn mà **không cần phải đăng xuất hay nhập lại mã đăng nhập**.

---

## 4. Gặp lỗi này thì làm gì? (Bảng xử lý nhanh)

Dưới đây là các trường hợp thường gặp nhất và cách xử lý đơn giản:

| Hiện tượng / Thông báo lỗi | Nguyên nhân | Cách xử lý trong 5 giây |
|---|---|---|
| Báo lỗi lệnh không nhận: `'sagent' is not recognized...` | Cửa sổ PowerShell mở từ trước khi cài nên chưa nhận diện được đường dẫn mới. | **Đóng cửa sổ PowerShell hiện tại và mở lại cửa sổ mới.** Hoặc chạy bằng đường dẫn đầy đủ: `& "$env:USERPROFILE\bin\sagent.exe"`. |
| Bật trang quản lý (`sagent dash`) bị từ chối chạy | Bạn chưa đặt mật khẩu bảo vệ cho trang quản lý. | Gõ lệnh: `sagent dash --set-password`, sau đó nhập tên đăng nhập và mật khẩu bạn muốn. Đặt xong gõ lại `sagent dash`. |
| Cài đặt / cập nhật báo: `sagent.exe đang chạy và không đổi tên được` | Có một tiến trình hoặc trang quản lý cũ đang chạy nền nên Windows khoá file. | Gõ lệnh: `sagent stop all` để dừng toàn bộ phiên đang chạy, sau đó chạy lại lệnh cài đặt. |
| Dùng Grok bị báo lỗi `503 No available channel` | Công cụ Grok yêu cầu chỉ định rõ tên mô hình AI muốn dùng. | Thêm `-m <tên-model>` vào câu lệnh, ví dụ: `sagent goc grok -m grok-4.5 -p "việc cần làm"`. |
| Dùng Antigravity không tạo được nhiều tài khoản | Phần mềm Antigravity lưu thông tin đăng nhập trong phần quản lý mật khẩu chung của Windows. | Mỗi máy tính hiện chỉ dùng được **1 tài khoản Antigravity** tại một thời điểm (đây là giới hạn của chính Antigravity, không phải lỗi của `sagent`). |
