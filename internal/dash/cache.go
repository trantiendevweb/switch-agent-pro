// Van xác thực cho asset nhúng — thứ mà bản trước KHÔNG có, và cái giá phải trả
// là cả một ngày sửa giao diện không tới được người dùng.
//
// CHUYỆN ĐÃ XẢY RA (20/08): tôi sửa mặt văn phòng bốn lần, mỗi lần đều build lại
// binary, cài lại, bật lại dash, và mỗi lần đều báo "xong". Người dùng gửi ảnh
// chụp về: vẫn y nguyên bản cũ. Không phải mã sai — mã đúng và nằm trong binary.
// Chỉ là trình duyệt không thèm hỏi lại lần nào.
//
// ĐO ĐƯỢC, không suy luận:
//
//	$ curl -skI https://.../docs/
//	HTTP/1.1 200 OK
//	Accept-Ranges: bytes
//	Content-Length: 51266
//	Content-Type: text/html; charset=utf-8
//	Date: ...
//
// Không Cache-Control, không ETag, không Last-Modified. KHÔNG MỘT VAN NÀO.
//
// Vì sao thiếu cả Last-Modified: `embed.FS` trả mốc thời gian RỖNG cho mọi file
// (giờ build không được nhúng vào), mà `http.ServeContent` thì bỏ qua mốc rỗng.
// Nên `http.FileServer` bọc `http.FS(embed.FS)` không phát ra van nào cả. Đây là
// cái bẫy riêng của embed — cùng đoạn mã đó chạy trên thư mục thật thì có
// Last-Modified và không ai gặp chuyện này.
//
// Không có van thì RFC 9111 cho phép trình duyệt tự đoán hạn dùng, và nó đoán
// rất rộng rãi. Kết quả: sửa giao diện xong, người dùng phải tự nghĩ ra chuyện
// bấm Ctrl+F5 — mà không có lý do gì để nghĩ ra.
package dash

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// bangETag băm từng file nhúng MỘT LẦN lúc dựng server.
//
// Băm nội dung chứ không lấy giờ build: hai lần build cùng một mã phải ra cùng
// một ETag, nếu không thì mỗi lần khởi động lại dash là mọi trình duyệt tải lại
// 1 MB vendor mà chẳng có gì đổi.
func bangETag(f fs.FS) map[string]string {
	b := map[string]string{}
	fs.WalkDir(f, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		noi, err := fs.ReadFile(f, p)
		if err != nil {
			return nil
		}
		s := sha256.Sum256(noi)
		b[p] = `"` + hex.EncodeToString(s[:8]) + `"`
		return nil
	})
	return b
}

// tepNhung bọc http.FileServer để gắn ETag + Cache-Control.
//
// `no-cache` KHÔNG có nghĩa là "đừng lưu" — nó có nghĩa "lưu thì lưu, nhưng lần
// nào cũng phải hỏi lại". Kèm ETag thì lần hỏi lại đó là một cái 304 rỗng, vài
// trăm byte. Đúng thứ cần cho công cụ chạy trên máy nhà: không tốn gì, mà không
// bao giờ còn cảnh sửa xong người dùng không thấy.
//
// http.ServeContent tự đọc ETag đã đặt sẵn trong header rồi so với
// If-None-Match, nên chỉ cần gắn vào trước khi chuyển tiếp.
func tepNhung(f fs.FS, trong http.Handler) http.Handler {
	et := bangETag(f)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		// FileServer tự trả index.html cho đường kết thúc bằng "/", nên phải tra
		// đúng file ấy — không thì trang chủ và /docs/ vẫn không có van.
		if p == "" || p == "." {
			p = "index.html"
		} else if strings.HasSuffix(r.URL.Path, "/") {
			p = p + "/index.html"
		}
		if v, ok := et[p]; ok {
			w.Header().Set("ETag", v)
			w.Header().Set("Cache-Control", "no-cache")
		}
		trong.ServeHTTP(w, r)
	})
}
