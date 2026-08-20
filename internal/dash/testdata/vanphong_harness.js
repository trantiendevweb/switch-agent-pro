// Chay that script vanphong voi DOM gia + THREE gia. Muc tieu: bat loi LUC CHAY
// (goi ham khong ton tai, doc thuoc tinh cua undefined) — thu ma `node --check`
// khong bao gio thay duoc. Day dung la cach da bat ra loi script chet o mat 2D.
const fs = require('fs');
const vm = require('vm');

class V3 {
  constructor(x = 0, y = 0, z = 0) { this.x = x; this.y = y; this.z = z; }
  set(x, y, z) { this.x = x; this.y = y; this.z = z; return this; }
  copy(o) { this.x = o.x; this.y = o.y; this.z = o.z; return this; }
  clone() { return new V3(this.x, this.y, this.z); }
  add(o) { this.x += o.x; this.y += o.y; this.z += o.z; return this; }
  sub(o) { this.x -= o.x; this.y -= o.y; this.z -= o.z; return this; }
  multiplyScalar(s) { this.x *= s; this.y *= s; this.z *= s; return this; }
  length() { return Math.hypot(this.x, this.y, this.z); }
  normalize() { const l = this.length() || 1; return this.multiplyScalar(1 / l); }
  distanceTo(o) { return Math.hypot(this.x - o.x, this.y - o.y, this.z - o.z); }
  project() { this.x = this.x / 40; this.y = this.y / 40; this.z = 0.5; return this; }
}
class Col {
  constructor(h) { this.h = typeof h === 'number' ? h : 0; }
  setHex(h) { this.h = h; return this; }
  lerp() { return this; }
  getHex() { return this.h; }
}
class Obj {
  constructor() {
    this.position = new V3(); this.rotation = { x: 0, y: 0, z: 0 };
    this.scale = { setScalar() {} }; this.children = []; this.userData = {};
    this.visible = true; this.renderOrder = 0;
  }
  add(o) { this.children.push(o); return this; }
  remove(o) { const i = this.children.indexOf(o); if (i >= 0) this.children.splice(i, 1); return this; }
  traverse(f) { f(this); this.children.forEach(c => c.traverse && c.traverse(f)); }
  lookAt() {}
  updateProjectionMatrix() {}
  updateMatrixWorld() {}
  // Noi that la luoi TINH nen ma that clone thang. Ban gia phai clone THAT su,
  // khong duoc tra ve chinh no — tra ve chinh no thi 95 mon do noi that deu la
  // MOT doi tuong, va bai kiem se xanh trong khi canh chi co mot cai ban.
  clone() { const o = new this.constructor(); demClone++; return o; }
}
class Geo {
  constructor() { this.attributes = {}; }
  dispose() {}
  setAttribute(k, v) { this.attributes[k] = v; }
}
class Mat {
  constructor(o) { Object.assign(this, o || {}); this.color = new Col(o && o.color); }
  dispose() {}
  clone() { return new Mat(this); }
}
function attr(n) {
  return {
    array: new Float32Array(n * 3), needsUpdate: false,
    setXYZ(i, x, y, z) { this.array[i * 3] = x; this.array[i * 3 + 1] = y; this.array[i * 3 + 2] = z; }
  };
}

const veLai = [];
const nhomTao = [];   // moi Group duoc tao, de do lai vi tri noi that
const lineTao = [];   // moi duong giao viec, de kiem no co duoc dat toa do khong
let demClone = 0;     // so mon noi that that su duoc lap vao canh

// Nguoi dung bat "giam chuyen dong" thi ca vong lap dong hinh bi cat bot. Chay
// them mot luot o che do do de biet duong giao viec CO duoc cap nhat khong —
// neu khong, no nam nguyen o goc toa do, tuc la mot vach cheo vo nghia giua san.
const RM_GIA = process.argv[3] === 'rm';
let demLine = 0, demHat = 0;
// Hinh cau la lop RIENG de dem duoc: hat chay doc duong giao viec la Mesh duy
// nhat dung hinh cau trong ca canh.
class GeoCau extends Geo {}
// Ba lop nay NHO lai kich thuoc that. Khong nho thi khong do duoc hai mon noi
// that co cam vao nhau khong — ma do la dung loai loi da xay ra: tu ho so dua
// vach ben cam vao goc mat ban hang sau.
class GeoHop extends Geo { constructor(w, h, d) { super(); this.fw = w; this.fh = h; this.fd = d; } }
class GeoTru extends Geo { constructor(rt, rb, h) { super(); const r = Math.max(rt, rb); this.fw = r * 2; this.fh = h; this.fd = r * 2; } }
class GeoPhang extends Geo { constructor(w, h) { super(); this.fw = w; this.fh = h; this.fd = 0.02; } }
const THREE = {
  WebGLRenderer: class { setPixelRatio() {} setSize() {} render() { veLai.push(1); } },
  Scene: class extends Obj {},
  PerspectiveCamera: class extends Obj {},
  Color: Col, Vector3: V3,
  FogExp2: class {},
  Box3: class {
    setFromObject() {
      this.min = { x: -0.5, y: 0, z: -0.5 };
      this.max = { x: 0.5, y: 1.8, z: 0.5 };
      return this;
    }
  },
  AmbientLight: class extends Obj {}, DirectionalLight: class extends Obj {},
  MeshStandardMaterial: Mat, MeshBasicMaterial: Mat, LineBasicMaterial: Mat,
  BoxGeometry: GeoHop, CylinderGeometry: GeoTru, PlaneGeometry: GeoPhang, CircleGeometry: Geo,
  TorusGeometry: Geo, SphereGeometry: GeoCau, EdgesGeometry: Geo, BufferGeometry: Geo,
  BufferAttribute: function (a) { return attr(a.length / 3); },
  Mesh: class extends Obj { constructor(g, m) { super(); this.geometry = g; this.material = m; this.isMesh = true;
      if (g instanceof GeoCau) demHat++; } },
  Group: class extends Obj { constructor() { super(); nhomTao.push(this); } },
  Line: class extends Obj { constructor(g, m) { super(); this.geometry = g; this.material = m; demLine++; lineTao.push(this); } },
  LineSegments: class extends Obj { constructor(g, m) { super(); this.geometry = g; this.material = m; } },
  GridHelper: class extends Obj { constructor() { super(); this.material = { transparent: false, opacity: 1 }; } },
  AnimationMixer: class {
    clipAction(c) {
      return {
        reset() { return this; }, setLoop() { return this; }, play() { return this; },
        stop() {}, fadeIn() {}, fadeOut() {}, setEffectiveWeight() {},
        clampWhenFinished: false, name: c.name
      };
    }
    update() {}
    addEventListener() {}
  },
  Clock: class { constructor() { this.elapsedTime = 0; } getDelta() { this.elapsedTime += 0.016; return 0.016; } },
  GLTFLoader: class {
    parse(buf, path, ok) {
      const s = new Obj();
      const m = new Obj(); m.isMesh = true; m.material = new Mat({}); m.material.name = 'Main';
      s.add(m);
      ok({ scene: s, animations: ['Idle', 'Walking', 'Running', 'Wave', 'ThumbsUp', 'No'].map(n => ({ name: n })) });
    }
  },
  Vector2: class { constructor(x = 0, y = 0) { this.x = x; this.y = y; } },
  Raycaster: class {
    setFromCamera() {}
    intersectObjects(ds) {
      const ra = [];
      const di = o => { ra.push({ object: o }); (o.children || []).forEach(di); };
      (ds || []).forEach(di);
      return ra;
    }
  },
  sRGBEncoding: 1, ACESFilmicToneMapping: 2, LoopOnce: 3, LoopRepeat: 4
};

// Su kien gan len chinh cua so (keydown chang han) — de mo phong go phim that.
const phimCua = {};
function banRaCuaSo(loai, ev) { (phimCua[loai] || []).forEach(f => f(ev)); }

// Ban mot su kien vao phan tu — de mo phong cu bam that.
function banRa(el, loai, ev) {
  (el._h[loai] || []).forEach(f => f(ev));
}

// Moi phan tu giu MOT co dinh suot doi. Bien doi moi lan doc thi bai kiem tra
// de nhau ben duoi khong con nghia gi: script cache mot con so, con bai kiem lai
// doc mot con so khac.
let dem = 0;
const soDo = [];   // moi phan tu duoc tao, de kiem tra de nhau sau khi ve
function elMoi(tag) {
  const rong = 120 + (dem++ % 4) * 45;
  const e = {
    tagName: tag, style: {}, children: [], className: '', textContent: '',
    classList: { add() {}, remove() {}, contains() { return false; } },
    appendChild(c) { this.children.push(c); return c; },
    remove() { e.goBo = true; }, setPointerCapture() {},
    _h: {},
    addEventListener(t, f) { (this._h[t] = this._h[t] || []).push(f); },
    getBoundingClientRect() { return { left: 0, top: 0, width: 1600, height: 900 }; },
    goBo: false, hidden: false, disabled: false, title: '', href: '',
    get offsetWidth() { return rong; },
    get offsetHeight() { return this.className === 'thoai' ? 54 : 26; }
  };
  soDo.push(e);
  return e;
}

// O tren man hinh cua mot nhan, tinh nguoc tu style + neo CSS cua tung loai.
function oCuaNhan(e) {
  if (e.goBo || e.style.display === 'none') return null;
  if (['lbl', 'thoai', 'phong-lbl'].indexOf(e.className) < 0) return null;
  const x = parseFloat(e.style.left), y = parseFloat(e.style.top);
  if (!isFinite(x) || !isFinite(y)) return null;
  const w = e.offsetWidth, h = e.offsetHeight;
  // .thoai neo o DAY (translate -50%,-100%); hai loai kia neo o TAM.
  const t = e.className === 'thoai' ? y - h : y - h / 2;
  return { l: x - w / 2, r: x + w / 2, t, d: t + h, ten: e.className };
}
function deNhau(a, b) {
  return a.l < b.r - 2 && a.r > b.l + 2 && a.t < b.d - 2 && a.d > b.t + 2;
}
const kho = {};
['scene', 'conn', 'conntext', 'flowname', 'scount', 'thieu', 'thieuten', 'thieuly',
  'chitiet', 'ct-dong', 'ct-ten', 'ct-phu', 'ct-bang', 'ct-out', 'ct-nut',
  'ct-hoithoai', 'ct-huy', 'ct-bao']
  .forEach(id => { kho[id] = elMoi('div'); });

const buocGia = [
  { id: 'gop', type: 'agent', profile: 'claude:phu', vaiTro: 'ceo', state: 'done', needs: [], output: '# Bao cao luot #46' },
  { id: 'code-go', type: 'agent', profile: 'antigravity:may', vaiTro: 'coder', state: 'running', needs: ['gop'], output: 'dang sua internal/aiapi' },
  { id: 'kiem-1', type: 'test', profile: '', vaiTro: 'tester', state: 'done', needs: ['code-go'], output: 'ok github.com/trantiendevweb/switch-agent-pro/cmd/sagent (cached)' },
  { id: 'kiem-2', type: 'test', profile: '', vaiTro: 'tester', state: 'pending', needs: ['code-go'], output: 'ok github.com/trantiendevweb/switch-agent-pro/internal/api' },
  { id: 'soi', type: 'review', profile: 'grok:api', vaiTro: 'soi', state: 'failed', needs: ['kiem-1', 'kiem-2'], output: 'soi failed' },
  { id: 'chualam', type: 'agent', profile: 'codex:tns', vaiTro: '', state: '', needs: ['soi'], output: '' }
];

let khung = null;
const ctx = {
  THREE, console, Math, JSON, Promise, Error, Float32Array, Array, Object,
  String, Number, Boolean, Date, Set, Map,
  document: {
    getElementById: id => kho[id] || null, createElement: elMoi,
    body: { appendChild(c) { return c; } },
    // Ma that hoi activeElement de biet nguoi dung co dang go trong o nhap khong.
    activeElement: null
  },
  matchMedia: () => ({ matches: RM_GIA }),
  getComputedStyle: () => ({ getPropertyValue: n => (n === '--link' ? '#39D9E0' : '#123456') }),
  innerWidth: 1600, innerHeight: 900, devicePixelRatio: 1,
  addEventListener(t, f) { (phimCua[t] = phimCua[t] || []).push(f); },
  setInterval() {}, setTimeout() {},
  requestAnimationFrame(f) { khung = f; },
  EventSource: class {},
  fetch: async (u) => {
    const s = String(u);
    if (s.endsWith('.glb')) return { ok: true, arrayBuffer: async () => new ArrayBuffer(8) };
    if (s.indexOf('/api/flows') >= 0) return { ok: true, json: async () => ({ runs: [{ id: 46, flow: 'doi-4', state: 'running' }] }) };
    if (s.indexOf('/api/flow/detail') >= 0) return { ok: true, json: async () => ({ steps: buocGia }) };
    return { ok: false };
  }
};
ctx.window = ctx;
vm.createContext(ctx);

try {
  vm.runInContext(fs.readFileSync(process.argv[2], 'utf8'), ctx, { filename: 'vanphong.js' });
} catch (e) {
  console.error('NO LUC NAP:', e.message);
  console.error(e.stack.split('\n').slice(0, 5).join('\n'));
  process.exit(1);
}

// Cho refresh() (async) chay xong, roi quay 60 khung hinh.
setTimeout(() => {
  if (!khung) { console.error('NO: khong ai goi requestAnimationFrame'); process.exit(1); }
  try {
    for (let i = 0; i < 60; i++) khung();
  } catch (e) {
    console.error('NO LUC QUAY:', e.message);
    console.error(e.stack.split('\n').slice(0, 6).join('\n'));
    process.exit(1);
  }
  console.log('OK — nap va quay 60 khung, khong nem loi');
  console.log('  scount   :', kho.scount.textContent);
  console.log('  flowname :', kho.flowname.textContent);
  console.log('  so lan ve:', veLai.length);

  // Dem lai TU DAU tren mot luot capNhat duy nhat: luc nap, capNhat co the chay
  // hai lan (mot lan tu refresh, mot lan nua khi .glb ve) nen con so cong don
  // khong noi len dieu gi.
  let hong = 0;
  demLine = 0; demHat = 0;
  ctx.capNhat();
  for (let i = 0; i < 10; i++) khung();

  // 1) KHONG NHAN NAO DUOC DE NHAN NAO. Day la loi thay ngay tren anh chup.
  const o = soDo.map(oCuaNhan).filter(Boolean);
  console.log('  nhan dang hien:', o.length);
  for (let i = 0; i < o.length; i++) {
    for (let j = i + 1; j < o.length; j++) {
      if (deNhau(o[i], o[j])) {
        console.error('  HONG: "' + o[i].ten + '" de len "' + o[j].ten + '"',
          JSON.stringify(o[i]), JSON.stringify(o[j]));
        hong++;
      }
    }
  }

  // 2) DUONG GIAO VIEC dung bang so canh `needs` giua HAI nhan vat khac nhau.
  //    Sau buoc gia o tren cho ra dung 6 canh, trong do 1 canh dang hoat dong
  //    (gop xong -> code-go dang chay).
  const hatMong = RM_GIA ? 0 : 1;   // giam chuyen dong thi KHONG dung hat chay
  console.log('  duong noi   :', demLine, '(cho doi 6)');
  console.log('  hat chay    :', demHat, '(cho doi ' + hatMong + ')');
  if (demLine !== 6) { console.error('  HONG: so duong noi sai'); hong++; }
  if (demHat !== hatMong) { console.error('  HONG: so hat sai'); hong++; }

  // Duong noi phai duoc DAT TOA DO moi khung hinh. Chua dat thi ca sau diem con
  // nam o goc — sau vach cheo qua giua san, khong noi len dieu gi.
  const dat = lineTao.slice(-6).filter(l => {
    const a2 = l.geometry.attributes.position;
    return a2 && a2.array[1] > 1.1 && a2.array[4] > 1.1;
  });
  console.log('  duong da dat toa do:', dat.length + '/6');
  if (dat.length !== 6) {
    console.error('  HONG: duong giao viec khong duoc cap nhat toa do' +
      (RM_GIA ? ' o che do giam chuyen dong' : ''));
    hong++;
  }

  // 3) HINH HOC PHONG. Toi khong co mat de nhin canh 3D, nen phai DO. Ba dieu:
  //    (a) moi mon noi that nam trong long phong, khong xuyen qua vach;
  //    (b) moi cho dung cung nam trong long phong;
  //    (c) khong mon noi that nao cam vao cho dung cua nguoi.
  //
  //    (c) da bat duoc mot loi that: tu ho so dua vach ben o z=-1.2 cam vao goc
  //    ban hang sau 0.35 don vi.
  const NUA = 4.6, CACH = 0.55;
  [['ban', 15, 0], ['may', 0, 15], ['hop', 0, -15]].forEach(([kieu, px, pz]) => {
    const p = { khoa: 't', x: px, z: pz, kieu };
    const truoc = nhomTao.length;
    ctx.dungPhong(p);
    const nhom = nhomTao.slice(truoc);
    if (!nhom.length) { console.error('  HONG: dungPhong khong tao Group nao'); hong++; return; }
    const phong = nhom[0];

    // Gom toa do RIENG cua tung mon: ghe la Group long trong Group phong.
    const mon = [];
    let soNhom = 0;
    const di = (o, ox, oz, cha) => o.children.forEach(c => {
      const x = ox + c.position.x, z = oz + c.position.z;
      if (c.children.length) { di(c, x, z, ++soNhom); return; }
      const g = c.geometry;
      mon.push({ x, z, y: c.position.y, cha,
        fw: g && g.fw, fh: g && g.fh, fd: g && g.fd });
    });
    di(phong, 0, 0, 0);

    mon.forEach(m => {
      if (Math.abs(m.x) > NUA || Math.abs(m.z) > NUA) {
        console.error('  HONG[' + kieu + ']: noi that loi ra ngoai vach', m.x.toFixed(2), m.z.toFixed(2));
        hong++;
      }
    });

    // Doi cho dung tu toa do san ve toa do rieng cua phong de so voi noi that.
    const c0 = Math.cos(p.goc), s0 = Math.sin(p.goc);
    for (let i = 0; i < 8; i++) {
      const c = ctx.oTrongPhong(p, i, 8);
      const dx = c.v.x - p.x, dz = c.v.z - p.z;
      const lx = dx * c0 - dz * s0, lz = dx * s0 + dz * c0;
      if (Math.abs(lx) > NUA || Math.abs(lz) > NUA) {
        console.error('  HONG[' + kieu + ']: cho thu ' + i + ' loi ra ngoai phong',
          lx.toFixed(2), lz.toFixed(2));
        hong++;
        continue;
      }
      // Chi xet mon o TAM THAN NGUOI (y trong 0.3..1.5). Mat ban cao 0.74 nam
      // ngay truoc mat nguoi la dung; van de la mon nao chiem CHO DUNG.
      mon.forEach(m => {
        if (m.y < 0.3 || m.y > 1.5) return;
        const d = Math.hypot(m.x - lx, m.z - lz);
        if (d < CACH) {
          console.error('  HONG[' + kieu + ']: mon noi that cam vao cho thu ' + i +
            ' (cach ' + d.toFixed(2) + ')', m.x.toFixed(2), m.z.toFixed(2));
          hong++;
        }
      });
    }
    // (d) HAI MON NOI THAT KHONG DUOC CAM VAO NHAU.
    //
    // Chi xet mon DAY (ca hai chieu day >= 0.4) va o TAM NGUOI NHIN (y cham vao
    // khoang 0.3..1.6). Bo qua mon mong nhu chan ban, vien man hinh, khe tu:
    // chung VON PHAI cham vao mat ban, dua chung vao la bao dong gia lien tuc.
    // Bo qua ca hai mon trong CUNG mot nhom con (bon phan cua mot cai ghe).
    const day = mon.filter(m => m.fw >= 0.4 && m.fd >= 0.4 &&
      (m.y + m.fh / 2) > 0.3 && (m.y - m.fh / 2) < 1.6);
    for (let i = 0; i < day.length; i++) {
      for (let j = i + 1; j < day.length; j++) {
        const a = day[i], b = day[j];
        if (a.cha === b.cha && a.cha !== 0) continue;
        const gx = Math.min(a.x + a.fw / 2, b.x + b.fw / 2) - Math.max(a.x - a.fw / 2, b.x - b.fw / 2);
        const gz = Math.min(a.z + a.fd / 2, b.z + b.fd / 2) - Math.max(a.z - a.fd / 2, b.z - b.fd / 2);
        const gy = Math.min(a.y + a.fh / 2, b.y + b.fh / 2) - Math.max(a.y - a.fh / 2, b.y - b.fh / 2);
        if (gx > 0.02 && gz > 0.02 && gy > 0.02) {
          console.error('  HONG[' + kieu + ']: hai mon noi that cam vao nhau — chong ' +
            gx.toFixed(2) + ' x ' + gz.toFixed(2) + ' x ' + gy.toFixed(2),
            '(' + a.x.toFixed(2) + ',' + a.z.toFixed(2) + ')',
            '(' + b.x.toFixed(2) + ',' + b.z.toFixed(2) + ')');
          hong++;
        }
      }
    }
    console.log('  phong ' + kieu + ': ' + mon.length + ' mon (' + day.length + ' mon day), do xong');
  });
  // 4) BAM VAO NHAN VAT phai mo bang chi tiet, KEO XOAY thi khong duoc mo.
  //    Luat ngang quyen: mat nao cung phai dieu khien duoc, khong chi de ngam.
  const canvas = kho['scene'], bang = kho['chitiet'];
  bang.hidden = true;

  // (a) Keo xoay camera: nha tay o cho khac -> KHONG duoc mo bang.
  banRa(canvas, 'pointerdown', { clientX: 800, clientY: 450 });
  banRa(canvas, 'pointerup', { clientX: 860, clientY: 470 });
  if (!bang.hidden) { console.error('  HONG: keo xoay camera lai mo bang chi tiet'); hong++; }

  // (b) Bam that: xuong va len cung mot cho -> mo bang, ten phai la nhan vat that.
  banRa(canvas, 'pointerdown', { clientX: 800, clientY: 450 });
  banRa(canvas, 'pointerup', { clientX: 801, clientY: 451 });
  if (bang.hidden) {
    console.error('  HONG: bam vao nhan vat khong mo duoc bang chi tiet'); hong++;
  } else {
    const ten = kho['ct-ten'].textContent;
    const co = buocGia.some(b => ten === b.profile || ten === 'máy chấm · ' + b.id);
    console.log('  bang chi tiet:', JSON.stringify(ten),
      '| so dong:', kho['ct-bang'].children.length,
      '| nut Huy khoa:', kho['ct-huy'].disabled);
    if (!co) { console.error('  HONG: ten trong bang khong khop nhan vat that nao'); hong++; }
    if (!kho['ct-bang'].children.length) { console.error('  HONG: bang chi tiet rong'); hong++; }
    // Luot chay gia dang `running` -> nut Huy PHAI bam duoc.
    if (kho['ct-huy'].disabled) { console.error('  HONG: luot dang chay ma nut Huy bi khoa'); hong++; }
  }

  // (c) Bam nut dong -> bang phai an.
  banRa(kho['ct-dong'], 'click', {});
  if (!bang.hidden) { console.error('  HONG: bam dong ma bang van hien'); hong++; }

  // (d) BAN PHIM. San chat luong doi thao tac duoc bang ban phim, ma raycaster
  //     thi chi co chuot. Mui ten phai di duoc vong qua ca dan, Esc phai dong.
  const ngan = () => {};
  banRaCuaSo('keydown', { key: 'ArrowRight', preventDefault: ngan });
  if (bang.hidden) { console.error('  HONG: mui ten phai khong chon duoc ai'); hong++; }
  const dau = kho['ct-ten'].textContent;
  const tham = new Set([dau]);
  for (let i = 0; i < 8; i++) {
    banRaCuaSo('keydown', { key: 'ArrowRight', preventDefault: ngan });
    tham.add(kho['ct-ten'].textContent);
  }
  console.log('  ban phim: di qua ' + tham.size + '/6 nhan vat, bat dau tu ' + JSON.stringify(dau));
  if (tham.size !== 6) {
    console.error('  HONG: mui ten khong di het dan — chi toi duoc ' + tham.size + '/6'); hong++;
  }
  // Mui ten trai phai quay NGUOC lai, khong phai cung mot huong.
  const truoc = kho['ct-ten'].textContent;
  banRaCuaSo('keydown', { key: 'ArrowLeft', preventDefault: ngan });
  banRaCuaSo('keydown', { key: 'ArrowRight', preventDefault: ngan });
  if (kho['ct-ten'].textContent !== truoc) {
    console.error('  HONG: trai roi phai khong quay ve cho cu — hai mui ten cung mot huong'); hong++;
  }
  // Dang go trong o nhap thi mui ten la de di con tro, khong phai doi nguoi.
  ctx.document.activeElement = { tagName: 'INPUT' };
  const giu = kho['ct-ten'].textContent;
  banRaCuaSo('keydown', { key: 'ArrowRight', preventDefault: ngan });
  if (kho['ct-ten'].textContent !== giu) {
    console.error('  HONG: dang go trong o nhap ma mui ten van doi nguoi'); hong++;
  }
  ctx.document.activeElement = null;
  banRaCuaSo('keydown', { key: 'Escape', preventDefault: ngan });
  if (!bang.hidden) { console.error('  HONG: Esc khong dong duoc bang'); hong++; }

  // 5) NOI THAT THAT (Kenney CC0) phai duoc LAP VAO, va ban khoi hop phai TAT.
  //
  //    So mon dem duoc tu chinh so do phong, khong phai con so lay dai:
  //      phong hop  : 3 ban + 6 ghe + 2 cay                      = 11
  //      phong ban x2: (6 cho x 4 mon) + 1 tu + 1 cay = 26 moi phong = 52
  //      phong may  : (6 cho x 4 mon) + 2 tu may + 1 cay          = 27
  //      sanh chung : 1 tham + 1 ban tra + 3 ghe                  =  5
  //                                                          tong = 95
  //    Lech con so nay nghia la mot phong bi bo sot, hoac mot mon bi lap hai lan.
  console.log('  noi that lap vao:', demClone + '/95 mon');
  if (demClone !== 95) {
    console.error('  HONG: so mon noi that sai — cho doi 95, dem duoc ' + demClone); hong++;
  }
  //    Nam nhom khoi hop (4 phong + sanh) phai bi TAT sau khi mau ve. Con hien
  //    la hai lop ban ghe chong len nhau, nhin ra ngay la loi.
  const anDi = nhomTao.filter(g => g.visible === false).length;
  console.log('  nhom khoi hop da tat:', anDi + '/5');
  if (anDi !== 5) {
    console.error('  HONG: cho doi 5 nhom khoi hop bi tat, dem duoc ' + anDi); hong++;
  }

  // 6) KHO DIEN THOAI. Man hep thi nhan chen nhau nang hon han, va luot tach de
  //    chi biet day LEN — day mai thi nhan bay len khoi mep tren, tuc la mat
  //    han. O 1600px chuyen do khong xay ra nen khong the thay bang mat thuong.
  ctx.innerWidth = 390; ctx.innerHeight = 844;
  soDo.forEach(e => { e.style.left = undefined; e.style.top = undefined; });
  for (let i = 0; i < 5; i++) khung();
  const om = soDo.map(oCuaNhan).filter(Boolean);
  let deM = 0, bayM = 0;
  for (let i = 0; i < om.length; i++) {
    // Thanh tren cao ~50px; nhan bi day len tren no la coi nhu mat.
    if (om[i].t < 4) bayM++;
    for (let j = i + 1; j < om.length; j++) if (deNhau(om[i], om[j])) deM++;
  }
  console.log('  kho 390x844: ' + om.length + ' nhan, ' + deM + ' cho de, ' + bayM + ' nhan bay khoi man');
  if (deM) { console.error('  HONG: o kho dien thoai con ' + deM + ' cho de nhau'); hong++; }
  if (bayM) { console.error('  HONG: ' + bayM + ' nhan bi day len khoi mep tren man hinh'); hong++; }

  console.log(hong ? '\nCO ' + hong + ' CHO HONG' : '\nTAT CA KIEM TRA XANH');
  process.exit(hong ? 1 : 0);
}, 400);
