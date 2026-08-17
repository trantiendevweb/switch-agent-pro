// Package acl siết quyền truy cập cho những thư mục/file chứa bí mật.
//
// Vì sao cần cả một package: `os.WriteFile(path, data, 0o600)` **không làm gì**
// trên Windows. Đã đo — dựng một thư mục có ACL rộng rồi ghi hai file từ Go:
//
//	secret-0600.json   ->  BUILTIN\Users:(I)(F)     ← Users TOÀN QUYỀN
//	public-0644.json   ->  BUILTIN\Users:(I)(F)     ← y hệt
//	dir-0700           ->  BUILTIN\Users:(I)(OI)(CI)(F)
//
// Bit quyền Unix chỉ là trang trí; quyền thật đến từ ACL kế thừa của thư mục cha.
// Trên máy dev nó tình cờ an toàn vì `C:\Users\<tên>` vốn đã kín — nhưng đó là
// MAY, không phải BẢO ĐẢM: đổi `AccountsRoot`, để home trên ổ chia sẻ, hay một
// máy có profile lỏng là token nằm phơi.
//
// Mà thứ nằm trong đó là `.credentials.json` và `dash-auth.json`.
package acl

// Restrict siết một thư mục hoặc file về CHỈ chủ sở hữu (cùng SYSTEM và nhóm
// quản trị, vì không có hai cái đó thì sao lưu và khôi phục hệ thống sẽ gãy).
//
// Trên Windows: dựng DACL tường minh và CẮT KẾ THỪA — không cắt thì ACE rộng từ
// thư mục cha vẫn chảy xuống. Trên Linux: chmod, vì ở đó bit quyền là thật.
//
// Gọi được nhiều lần, không hại gì.

// Check nói xem đường dẫn có đang phơi ra cho người khác không.
// ok=false kèm detail giải thích ai đang có quyền.
