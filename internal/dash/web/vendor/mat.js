/* mat.js — phần `[ui]` của cấu hình, áp cho MỌI mặt web.
 *
 * Vì sao một file chung thay vì chép vào bốn trang: bố cục do project.toml
 * quyết định, nên bốn trang phải hiểu cùng một luật. Chép bốn bản thì sửa một
 * chỗ là ba chỗ kia trôi đi — đúng lỗi đã xảy ra với bảng màu trước khi có
 * token.css, và cũng đúng lỗi đã xảy ra khi index.html tự đoán trạng thái phiên
 * còn màn 3D đoán kiểu khác.
 *
 * File nằm trong vendor/ vì server mở công khai thư mục đó. Không có bí mật nào
 * ở đây: nó chỉ đọc lại thứ /api/state đã trả về.
 */

(function (window) {
  'use strict';

  /* Khoá localStorage. Đây là BỘ NHỚ ĐỆM, không phải nguồn sự thật: nguồn là
     `[ui]` trong project.toml, do server trả qua /api/state. Đệm chỉ tồn tại để
     lần tải sau không nháy từ tối sang sáng trước mắt người dùng. */
  const KHOA_THEME = 'sagent.theme';

  /* themeSom áp chủ đề NGAY khi trang bắt đầu tải, trước cả khi có mạng.
   *
   * Phải chạy sớm nhất có thể — nếu đợi /api/state về mới đặt thì người dùng chủ
   * đề sáng sẽ thấy một nhịp màn hình tối rồi mới lật, mỗi lần mở trang. Lần đầu
   * tiên trong đời máy thì chưa có đệm nên vẫn nháy đúng một lần; từ lần hai trở
   * đi thì không. */
  function themeSom() {
    try {
      const t = localStorage.getItem(KHOA_THEME);
      if (t === 'light' || t === 'dark') document.documentElement.dataset.theme = t;
    } catch (_) {
      /* localStorage bị chặn (chế độ riêng tư, cookie tắt) thì bỏ qua: mất phần
         chống nháy, không mất tính năng. Ném lỗi ở đây sẽ chặn cả trang. */
    }
  }

  /* apDungUI áp toàn bộ `[ui]` sau khi /api/state về. Đây là lời cuối cùng —
     server nói gì thì theo, kể cả khi đệm nói khác. */
  function apDungUI(ui) {
    if (!ui) return;
    datTheme(ui.theme);
    anLoiVao3D(ui.enable3d);
  }

  function datTheme(theme) {
    const t = theme === 'light' ? 'light' : 'dark';
    document.documentElement.dataset.theme = t;
    try { localStorage.setItem(KHOA_THEME, t); } catch (_) {}
  }

  /* anLoiVao3D gỡ hẳn link tới mặt Trung tâm khi `ui.enable_3d = false`.
   *
   * GỠ chứ không chỉ `display:none`: link ẩn vẫn nằm trong thứ tự Tab và vẫn đọc
   * được bằng trình đọc màn hình, nên người dùng bàn phím vẫn lạc vào một mặt mà
   * dự án đã tắt. Tắt thì phải tắt thật. */
  function anLoiVao3D(bat) {
    if (bat !== false) return;
    document.querySelectorAll('a[href$="trung-tam.html"]').forEach(a => a.remove());
  }

  /* NHAN_COT là nhãn hiển thị của từng cột trong `ui.columns`.
   *
   * Nhãn nằm ở ĐÂY chứ không trong HTML tĩnh, vì HTML không biết trước cột nào
   * được chọn — bảng phiên phải dựng đầu bảng lúc chạy. Tên khoá phải khớp
   * `config.CotPhien` bên Go; có test giữ hai danh sách không trôi khỏi nhau. */
  const NHAN_COT = {
    provider:   'Provider',
    tai_khoan:  'Tài khoản',
    danh_tinh:  'Danh tính',
    trang_thai: 'Trạng thái',
    pid:        'PID',
    nhanh:      'Nhánh',
    bat_dau:    'Bắt đầu',
  };

  function nhanCot(khoa) { return NHAN_COT[khoa] || khoa; }

  /* cotHopLe lọc bỏ tên cột lạ. Server đã chặn từ lúc đọc file nên đây là lớp
     thứ hai; giữ vì mặt web còn nhận cấu hình cũ nằm trong đệm của trình duyệt. */
  function cotHopLe(cot) {
    return (cot || []).filter(c => Object.prototype.hasOwnProperty.call(NHAN_COT, c));
  }

  /* ghimLenDau đưa những mục có tên trong `ui.pinned_flows` lên đầu danh sách,
     giữ nguyên thứ tự người dùng khai, phần còn lại giữ nguyên thứ tự cũ.
   *
   * Tên KHÔNG có trong danh sách thật thì bỏ qua — không dựng mục ma. Ghim một
   * flow đã xoá khỏi flows.toml mà bảng vẫn hiện tên nó thì người dùng bấm vào
   * sẽ nhận lỗi từ server, và họ sẽ tưởng flow hỏng chứ không nghĩ là nó không
   * còn tồn tại. */
  function ghimLenDau(dsach, ghim, tenCua) {
    const ds = dsach || [];
    if (!ghim || !ghim.length) return ds.slice();
    const ten = tenCua || (x => x);
    const daGhim = [];
    ghim.forEach(g => {
      const m = ds.find(x => ten(x) === g);
      if (m && daGhim.indexOf(m) === -1) daGhim.push(m);
    });
    return daGhim.concat(ds.filter(x => daGhim.indexOf(x) === -1));
  }

  /* Phơi ra đúng những thứ các trang cần. Không dùng ES module vì bốn trang đang
     là <script> cổ điển — đổi kiểu script chỉ để nạp một file phụ là thay đổi
     rộng hơn việc nó phục vụ. */
  window.Mat = { themeSom, apDungUI, nhanCot, cotHopLe, ghimLenDau };
})(window);
