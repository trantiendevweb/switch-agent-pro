// Package sinhkehoach dựng trang kế hoạch HTML từ file Markdown nguồn.
//
// Vì sao có gói này: `internal/dash/web/docs/master-plan.html` trước đây được
// dọn TAY sau khi chép từ bản render Python. Không có khâu sinh lại thì mỗi lần
// dọn tay là một lần trang lệch với bản .md mà không ai báo. Đo ngày 21/08/2026:
// trang .html nhúng và `MASTER-PLAN.md` nhúng đã lệch tới mức `uxui_test.go`
// phải MIỄN TRỪ hẳn trang này khỏi mọi luật giao diện, kèm ghi chú "chưa có khâu
// sinh lại — dọn tay sẽ lệch với bản .md". Miễn trừ đó là nợ, không phải là ổn.
//
// Nay trang là ĐẦU RA chứ không phải file nguồn: sửa .md rồi chạy
//
//	go run ./tools/sinhkehoach/cmd/sinhkehoach
//
// (hoặc `go generate ./tools/...`), và TestTrangKeHoachKhopBanSinh trong
// internal/dash giữ cho hai bên không trôi khỏi nhau — lệch là đỏ.
//
// KHÔNG kéo thư viện markdown ngoài: dự án cam kết một binary không phụ thuộc
// thừa, và tập cú pháp mà MASTER-PLAN.md thật sự dùng thì đếm được — 725 dòng,
// 6 loại khối (tiêu đề, danh sách, bảng, khối mã, trích dẫn, đường kẻ) và 5 lối
// nội tuyến (mã, đậm, nghiêng, gạch, liên kết). Viết tay rẻ hơn một dependency.
package sinhkehoach

//go:generate go run ./cmd/sinhkehoach

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Tên file cố định của cặp nguồn/đích trong thư mục nhúng. Để ở đây làm MỘT
// nguồn: bộ sinh, lệnh chạy và test cùng đọc, không ai gõ lại chuỗi.
const (
	TenNguon = "MASTER-PLAN.md"
	TenDich  = "master-plan.html"
	TieuDe   = "Switch-Agent-Pro — Master Plan"
)

// ThuMucNhung là đường (theo gốc repo) tới thư mục dash nhúng bằng go:embed.
const ThuMucNhung = "internal/dash/web/docs"

// Trang sinh toàn bộ trang HTML từ nội dung Markdown.
//
// Đầu ra ổn định: cùng đầu vào ra cùng byte, không mốc thời gian, không số ngẫu
// nhiên — nếu không thì test so khớp sẽ đỏ ngẫu hứng và người ta sẽ tắt nó đi.
func Trang(md []byte, tieuDe string) []byte {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"vi\">\n<head>\n" +
		"<meta charset=\"UTF-8\" />\n" +
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1, viewport-fit=cover\" />\n" +
		"<title>")
	b.WriteString(html.EscapeString(tieuDe))
	b.WriteString("</title>\n<style>\n")
	b.WriteString(css)
	b.WriteString("</style>\n</head>\n")
	b.WriteString(khungTren)
	b.WriteString(than(string(md)))
	b.WriteString(khungDuoi)
	return []byte(b.String())
}

// Font nằm sẵn trong binary (web/vendor) nên trang vẽ được khi máy rời mạng.
// Một <link> tới fonts.googleapis.com ở đây là lỗi im lặng: chữ rơi về font hệ
// điều hành và không ai báo — internal/dash/offline_asset_test.go quét chỗ này.
const css = `@font-face{font-family:Inter;src:url("../vendor/inter-variable.woff2") format("woff2");
  font-weight:100 900;font-style:normal;font-display:swap}
@font-face{font-family:"Space Grotesk";src:url("../vendor/space-grotesk-variable.woff2") format("woff2");
  font-weight:300 700;font-style:normal;font-display:swap}
@font-face{font-family:"JetBrains Mono";src:url("../vendor/jetbrains-mono-variable.woff2") format("woff2");
  font-weight:100 800;font-style:normal;font-display:swap}
:root{--mono:"JetBrains Mono",ui-monospace,Consolas,monospace;
--bg:#0F172A;--primary:#1E293B;--muted:#272F42;--fg:#F8FAFC;--sub:#94A3B8;
--border:#475569;--run:#22C55E;--warn:#F59E0B;--limit:#EF4444;
--panel:rgba(30,41,59,.55);--ease:cubic-bezier(0.16,1,0.3,1);
--safe-b:env(safe-area-inset-bottom,0px)}
*{box-sizing:border-box}
html{scroll-behavior:smooth;-webkit-text-size-adjust:100%}
body{margin:0;background:
radial-gradient(1100px 520px at 50% -8%,rgba(34,197,94,.10),transparent 60%),
radial-gradient(800px 460px at 100% 6%,rgba(66,133,244,.07),transparent 55%),var(--bg);
color:var(--fg);font-family:Inter,system-ui,-apple-system,"Segoe UI",sans-serif;
line-height:1.65;-webkit-font-smoothing:antialiased;overflow-wrap:break-word}
.nav{position:sticky;top:0;z-index:20;display:flex;align-items:center;gap:10px;padding:12px 16px;
backdrop-filter:blur(18px);-webkit-backdrop-filter:blur(18px);background:rgba(15,23,42,.85);
border-bottom:1px solid rgba(71,85,105,.45)}
.nav .b{display:flex;align-items:center;gap:8px;font-weight:700;font-size:15px}
.nav .sp{margin-left:auto;display:flex;gap:8px}
.nav a{text-decoration:none;color:var(--fg);border:1px solid var(--border);border-radius:8px;
padding:7px 11px;font-size:12px;font-weight:600;transition:all .2s var(--ease);white-space:nowrap}
.nav a:hover{border-color:var(--run);background:rgba(34,197,94,.1)}
.nav a.p{border-color:var(--run);background:rgba(34,197,94,.14);color:var(--run)}
main{max-width:860px;margin:0 auto;padding:22px 18px calc(80px + var(--safe-b))}
h1,h2,h3{font-family:"Space Grotesk",Inter,system-ui,sans-serif}
h1{font-size:clamp(26px,6vw,40px);line-height:1.14;font-weight:800;letter-spacing:-.02em;margin:6px 0 14px;
background:linear-gradient(120deg,#22C55E,#4285F4);-webkit-background-clip:text;background-clip:text;color:transparent}
h2{font-size:clamp(19px,4vw,26px);font-weight:700;margin:38px 0 10px;padding-top:14px;
border-top:1px solid rgba(71,85,105,.4);letter-spacing:-.01em;scroll-margin-top:64px}
h3{font-size:clamp(16px,3vw,19px);font-weight:600;margin:24px 0 8px;color:var(--run);scroll-margin-top:64px}
h4{font-size:15px;font-weight:600;margin:18px 0 6px;color:var(--sub)}
p{margin:10px 0;color:#dbe3ee}
a{color:#7dd3fc}
ul,ol{padding-left:22px;margin:10px 0}
li{margin:5px 0;color:#dbe3ee}
li::marker{color:var(--run)}
strong{color:var(--fg);font-weight:600}
em{color:var(--sub)}
del{color:var(--sub);text-decoration-color:var(--limit)}
code{background:var(--muted);padding:2px 6px;border-radius:5px;font-size:.87em;color:#e2e8f0;
font-family:var(--mono)}
pre{background:rgba(15,23,42,.8);border:1px solid var(--border);border-radius:12px;padding:14px;
overflow-x:auto;margin:14px 0}
pre code{background:none;padding:0;font-size:12.5px;line-height:1.6}
blockquote{margin:14px 0;padding:12px 16px;border-left:3px solid var(--run);
background:rgba(34,197,94,.07);border-radius:0 10px 10px 0;color:#cbd5e1}
blockquote strong{color:var(--run)}
.tw{overflow-x:auto;margin:14px 0;-webkit-overflow-scrolling:touch}
table{width:100%;border-collapse:collapse;font-size:13.5px;min-width:min(100%,520px)}
th,td{text-align:left;padding:9px 11px;border-bottom:1px solid rgba(71,85,105,.4);vertical-align:top}
th{color:var(--sub);font-weight:600;text-transform:uppercase;font-size:11px;letter-spacing:.07em;white-space:nowrap}
hr{border:0;border-top:1px solid rgba(71,85,105,.35);margin:26px 0}
.task{list-style:none;margin-left:-16px}
.task .box{display:inline-block;width:15px;height:15px;border:1.5px solid var(--border);border-radius:4px;
margin-right:8px;vertical-align:-2px}
.task.done{color:var(--sub)}
.task.done .box{background:var(--run);border-color:var(--run);position:relative}
.task.done .box::after{content:"";position:absolute;left:4px;top:1px;width:4px;height:8px;
border:solid #0F172A;border-width:0 2px 2px 0;transform:rotate(45deg)}
/* [~] xong mot phan: o day NUA. Co y khong dung dau tick mo dan — mo dan doc ra
   la "xong, hoi nhat", con nua o doc ra la "moi duoc mot nua", va do moi la nghia. */
.task.part .box{border-color:var(--warn);position:relative}
.task.part .box::after{content:"";position:absolute;left:1px;top:1px;bottom:1px;width:5px;
background:var(--warn);border-radius:2px 0 0 2px}
/* [!] bi chan: dau cham than, khong phai o trong. Bi chan KHAC chua lam. */
.task.block .box{border-color:var(--limit);position:relative}
.task.block .box::after{content:"!";position:absolute;inset:0;display:grid;place-items:center;
color:var(--limit);font-size:11px;font-weight:800;line-height:1}
.top{position:fixed;right:16px;bottom:calc(16px + var(--safe-b));z-index:15;background:var(--panel);
border:1px solid var(--border);color:var(--fg);border-radius:999px;width:44px;height:44px;display:grid;
place-items:center;cursor:pointer;backdrop-filter:blur(14px);text-decoration:none}
@media(prefers-reduced-motion:reduce){html{scroll-behavior:auto}
.nav a{transition:none}}
`

// Thanh điều hướng của bản NHÚNG trỏ về dashboard đang chạy ("/") chứ không trỏ
// tới file rời như bản ở gốc repo — trang này do server phục vụ, không mở bằng
// file:// nên "plan.html" cạnh bên sẽ 404.
const khungTren = `<body id="top">
<nav class="nav">
  <div class="b">
    <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="#22C55E" stroke-width="1.8"><path d="M12 2l8.5 5v10L12 22 3.5 17V7z"/><circle cx="12" cy="12" r="3" fill="#22C55E" stroke="none"/></svg>
    Switch-Agent-Pro
  </div>
  <div class="sp"><a class="p" href="MASTER-PLAN.md">.md</a><a href="./">Plan</a><a href="/">Dashboard</a></div>
</nav>
<main>
`

const khungDuoi = `</main>
<a class="top" href="#top" aria-label="Len dau trang">
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 19V5M5 12l7-7 7 7"/></svg>
</a>
</body>
</html>
`

var (
	reTieuDe  = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*$`)
	reMuc     = regexp.MustCompile(`^([ \t]*)([-*+]|\d+\.)[ \t]+(.*)$`)
	reViec    = regexp.MustCompile(`^\[([ xX~!])\][ \t]+`)
	reLien    = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	reDam     = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reGach    = regexp.MustCompile(`~~([^~]+)~~`)
	reNghieng = regexp.MustCompile(`\*([^\s*][^*]*)\*`)
	reOMa     = regexp.MustCompile("\x00(\\d+)\x00")
)

// than dịch thân Markdown thành HTML. Vòng lặp theo DÒNG chứ không theo ký tự:
// mọi khối trong MASTER-PLAN.md đều nhận ra được từ đầu dòng.
func than(src string) string {
	dong := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	var b strings.Builder
	for i := 0; i < len(dong); {
		d := dong[i]
		t := strings.TrimSpace(d)
		switch {
		case t == "":
			i++
		case strings.HasPrefix(t, "```"):
			i = khoiMa(&b, dong, i)
		case reTieuDe.MatchString(t):
			m := reTieuDe.FindStringSubmatch(t)
			fmt.Fprintf(&b, "<h%d>%s</h%d>\n", len(m[1]), noiTuyen(m[2]), len(m[1]))
			i++
		case laDuongKe(t):
			b.WriteString("<hr />\n")
			i++
		case strings.HasPrefix(t, ">"):
			i = khoiTrich(&b, dong, i)
		case laDauBang(dong, i):
			i = khoiBang(&b, dong, i)
		case reMuc.MatchString(d):
			i = danhSach(&b, dong, i, 0)
		default:
			i = khoiDoan(&b, dong, i)
		}
	}
	return b.String()
}

func laDuongKe(t string) bool {
	if len(t) < 3 {
		return false
	}
	for _, k := range []string{"-", "*", "_"} {
		if strings.Trim(t, k) == "" {
			return true
		}
	}
	return false
}

func khoiMa(b *strings.Builder, dong []string, i int) int {
	t := strings.TrimSpace(dong[i])
	n := len(t) - len(strings.TrimLeft(t, "`"))
	i++
	var noi []string
	for i < len(dong) {
		c := strings.TrimSpace(dong[i])
		if len(c) >= n && strings.Trim(c, "`") == "" {
			i++
			break
		}
		noi = append(noi, dong[i])
		i++
	}
	b.WriteString("<pre><code>" + html.EscapeString(strings.Join(noi, "\n")) + "\n</code></pre>\n")
	return i
}

// khoiTrich gỡ đúng một dấu ">" và một dấu cách sau nó, rồi dịch phần còn lại
// như một tài liệu con — nhờ vậy tiêu đề/danh sách nằm trong khối trích vẫn ra
// hình, và mức thụt bên trong không bị mất.
func khoiTrich(b *strings.Builder, dong []string, i int) int {
	var noi []string
	for i < len(dong) {
		d := strings.TrimLeft(dong[i], " \t")
		if !strings.HasPrefix(d, ">") {
			break
		}
		noi = append(noi, strings.TrimPrefix(d[1:], " "))
		i++
	}
	b.WriteString("<blockquote>\n" + than(strings.Join(noi, "\n")) + "</blockquote>\n")
	return i
}

// laDauBang: một dòng bảng chỉ tính là bảng khi dòng NGAY SAU là dòng ngăn
// (|---|---|). Không có luật đó thì mọi câu văn mở đầu bằng "|" thành bảng một ô.
func laDauBang(dong []string, i int) bool {
	if !strings.HasPrefix(strings.TrimSpace(dong[i]), "|") || i+1 >= len(dong) {
		return false
	}
	o := oBang(dong[i+1])
	if len(o) < 2 {
		return false
	}
	for _, c := range o {
		c = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(c), ":"), ":")
		if c == "" || strings.Trim(c, "-") != "" {
			return false
		}
	}
	return true
}

// oBang cắt một dòng bảng thành các ô, tôn trọng "\|" là ký tự sổ thật.
func oBang(d string) []string {
	t := strings.TrimSpace(d)
	t = strings.TrimSuffix(strings.TrimPrefix(t, "|"), "|")
	var o []string
	var cur strings.Builder
	for i := 0; i < len(t); i++ {
		if t[i] == '\\' && i+1 < len(t) && t[i+1] == '|' {
			cur.WriteByte('|')
			i++
			continue
		}
		if t[i] == '|' {
			o = append(o, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(t[i])
	}
	return append(o, strings.TrimSpace(cur.String()))
}

// Bảng bọc trong .tw để cuộn ngang được trên điện thoại — bảng "Món/Quyết định/
// Lý do" rộng hơn màn 360px, không bọc thì nó đẩy cả trang trượt ngang.
func khoiBang(b *strings.Builder, dong []string, i int) int {
	b.WriteString("<div class=\"tw\"><table>\n<thead>\n<tr>\n")
	for _, o := range oBang(dong[i]) {
		b.WriteString("<th>" + noiTuyen(o) + "</th>\n")
	}
	b.WriteString("</tr>\n</thead>\n<tbody>\n")
	i += 2
	for i < len(dong) && strings.HasPrefix(strings.TrimSpace(dong[i]), "|") {
		b.WriteString("<tr>\n")
		for _, o := range oBang(dong[i]) {
			b.WriteString("<td>" + noiTuyen(o) + "</td>\n")
		}
		b.WriteString("</tr>\n")
		i++
	}
	b.WriteString("</tbody>\n</table></div>\n")
	return i
}

// danhSach dựng một danh sách bắt đầu ở dòng i, chỉ nhận mục thụt đúng `thut`
// cột; mục thụt sâu hơn thành danh sách con NẰM TRONG <li> đang mở — không đóng
// <li> ngay sau nội dung, vì đóng sớm thì danh sách con văng ra ngoài mục cha.
//
// MASTER-PLAN.md chỉ dùng một cấp lồng (2 dấu cách, 24 dòng), nhưng viết đệ quy
// thì không phải sửa lại khi tài liệu sâu thêm.
func danhSach(b *strings.Builder, dong []string, i, thut int) int {
	the := "ul"
	if m := reMuc.FindStringSubmatch(dong[i]); m != nil && !strings.ContainsAny(m[2], "-*+") {
		the = "ol"
	}
	b.WriteString("<" + the + ">\n")
	moLi := false
	dongLi := func() {
		if moLi {
			b.WriteString("</li>\n")
			moLi = false
		}
	}
	for i < len(dong) {
		d := dong[i]
		if strings.TrimSpace(d) == "" {
			// Dòng trống chỉ cắt danh sách khi thứ theo sau KHÔNG còn là mục
			// cùng cấp; nếu còn thì đây là danh sách thưa, vẫn một danh sách.
			j := i
			for j < len(dong) && strings.TrimSpace(dong[j]) == "" {
				j++
			}
			if j < len(dong) {
				if m := reMuc.FindStringSubmatch(dong[j]); m != nil && len(m[1]) >= thut {
					i = j
					continue
				}
			}
			break
		}
		m := reMuc.FindStringSubmatch(d)
		if m == nil {
			// Dòng nối tiếp đứng sau một danh sách con: thụt vào, và đang có mục mở.
			if moLi && strings.HasPrefix(d, " ") {
				b.WriteString("\n" + noiTuyen(strings.TrimSpace(d)))
				i++
				continue
			}
			break
		}
		if len(m[1]) < thut {
			break
		}
		if len(m[1]) > thut {
			if !moLi {
				b.WriteString("<li>")
				moLi = true
			}
			b.WriteString("\n")
			i = danhSach(b, dong, i, len(m[1]))
			continue
		}
		dongLi()
		// Gom cả dòng nối tiếp TRƯỚC khi dịch nội tuyến: chữ **đậm** trong
		// MASTER-PLAN.md thường vắt qua hai dòng (file gói ~80 cột), dịch từng
		// dòng một sẽ để lộ dấu sao ra mặt trang.
		noi := []string{m[3]}
		i++
		for i < len(dong) {
			c := dong[i]
			if strings.TrimSpace(c) == "" || reMuc.MatchString(c) || !strings.HasPrefix(c, " ") {
				break
			}
			noi = append(noi, strings.TrimSpace(c))
			i++
		}
		vb := strings.Join(noi, "\n")
		if v := reViec.FindStringSubmatch(vb); v != nil {
			b.WriteString(`<li class="` + lopViec(v[1]) + `"><span class="box"></span>` + noiTuyen(vb[len(v[0]):]))
		} else {
			b.WriteString("<li>" + noiTuyen(vb))
		}
		moLi = true
	}
	dongLi()
	b.WriteString("</" + the + ">\n")
	return i
}

// lopViec dịch dấu trong ô việc sang lớp CSS.
//
// Vì sao BỐN dấu chứ không hai: bản trước chỉ phân biệt "rỗng" với "khác rỗng",
// nên mọi thứ không phải `[ ]` đều vẽ thành dấu tick xanh. Từ 21/08 bản .md dùng
// thêm `[~]` (xong một phần) và `[!]` (bị chặn vì thiếu thứ bên ngoài); để nguyên
// luật cũ thì một mục mới làm được nửa sẽ hiện trên trang y hệt một mục đã xong —
// đúng kiểu lệch im lặng mà cả bộ sinh này sinh ra để chặn.
//
// Dấu lạ rơi về "task" (ô trống). Không thể xảy ra vì reViec đã lọc, nhưng viết
// nhánh mặc định là chưa-xong chứ không phải đã-xong: đoán sai theo chiều tô hồng
// tốn hơn đoán sai theo chiều dè dặt.
func lopViec(dau string) string {
	switch dau {
	case "x", "X":
		return "task done"
	case "~":
		return "task part"
	case "!":
		return "task block"
	default:
		return "task"
	}
}

func khoiDoan(b *strings.Builder, dong []string, i int) int {
	var noi []string
	for i < len(dong) {
		d := dong[i]
		t := strings.TrimSpace(d)
		if t == "" || laDuongKe(t) || reTieuDe.MatchString(t) ||
			strings.HasPrefix(t, ">") || strings.HasPrefix(t, "```") ||
			reMuc.MatchString(d) || laDauBang(dong, i) {
			break
		}
		noi = append(noi, t)
		i++
	}
	if len(noi) > 0 {
		b.WriteString("<p>" + noiTuyen(strings.Join(noi, "\n")) + "</p>\n")
	}
	return i
}

// noiTuyen dịch phần nội tuyến. Thứ tự BẮT BUỘC: rút `mã` ra trước rồi mới thoát
// HTML và bắt cặp sao/ngã — nếu không, dấu sao trong một tên biến sẽ hoá chữ
// đậm, và dấu < trong khối mã sẽ bị trình duyệt coi là thẻ.
func noiTuyen(s string) string {
	var ma []string
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '`' {
			b.WriteByte(s[i])
			i++
			continue
		}
		n := 0
		for i+n < len(s) && s[i+n] == '`' {
			n++
		}
		rao := strings.Repeat("`", n)
		j := strings.Index(s[i+n:], rao)
		if j < 0 {
			// Dấu huyền lẻ (không có dấu đóng) là chữ thật, không phải khối mã.
			b.WriteString(rao)
			i += n
			continue
		}
		ma = append(ma, s[i+n:i+n+j])
		fmt.Fprintf(&b, "\x00%d\x00", len(ma)-1)
		i += n + j + n
	}
	t := html.EscapeString(b.String())
	t = reLien.ReplaceAllString(t, `<a href="$2">$1</a>`)
	t = reDam.ReplaceAllString(t, "<strong>$1</strong>")
	t = reGach.ReplaceAllString(t, "<del>$1</del>")
	t = reNghieng.ReplaceAllString(t, "<em>$1</em>")
	return reOMa.ReplaceAllStringFunc(t, func(o string) string {
		var k int
		fmt.Sscanf(o, "\x00%d\x00", &k)
		return "<code>" + html.EscapeString(ma[k]) + "</code>"
	})
}
