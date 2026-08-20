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
let demLine = 0, demHat = 0;
// Hinh cau la lop RIENG de dem duoc: hat chay doc duong giao viec la Mesh duy
// nhat dung hinh cau trong ca canh.
class GeoCau extends Geo {}
const THREE = {
  WebGLRenderer: class { setPixelRatio() {} setSize() {} render() { veLai.push(1); } },
  Scene: class extends Obj {},
  PerspectiveCamera: class extends Obj {},
  Color: Col, Vector3: V3,
  FogExp2: class {},
  Box3: class { setFromObject() { this.max = { y: 1.8 }; this.min = { y: 0 }; return this; } },
  AmbientLight: class extends Obj {}, DirectionalLight: class extends Obj {},
  MeshStandardMaterial: Mat, MeshBasicMaterial: Mat, LineBasicMaterial: Mat,
  BoxGeometry: Geo, CylinderGeometry: Geo, PlaneGeometry: Geo, CircleGeometry: Geo,
  TorusGeometry: Geo, SphereGeometry: GeoCau, EdgesGeometry: Geo, BufferGeometry: Geo,
  BufferAttribute: function (a) { return attr(a.length / 3); },
  Mesh: class extends Obj { constructor(g, m) { super(); this.geometry = g; this.material = m; this.isMesh = true;
      if (g instanceof GeoCau) demHat++; } },
  Group: class extends Obj {},
  Line: class extends Obj { constructor(g, m) { super(); this.geometry = g; this.material = m; demLine++; } },
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
  sRGBEncoding: 1, ACESFilmicToneMapping: 2, LoopOnce: 3, LoopRepeat: 4
};

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
    remove() { e.goBo = true; }, addEventListener() {}, setPointerCapture() {},
    goBo: false,
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
['scene', 'conn', 'conntext', 'flowname', 'scount', 'thieu', 'thieuten', 'thieuly']
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
  document: { getElementById: id => kho[id] || null, createElement: elMoi, body: { appendChild(c) { return c; } } },
  matchMedia: () => ({ matches: false }),
  getComputedStyle: () => ({ getPropertyValue: n => (n === '--link' ? '#39D9E0' : '#123456') }),
  innerWidth: 1600, innerHeight: 900, devicePixelRatio: 1,
  addEventListener() {}, setInterval() {}, setTimeout() {},
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
  console.log('  duong noi   :', demLine, '(cho doi 6)');
  console.log('  hat chay    :', demHat, '(cho doi 1)');
  if (demLine !== 6) { console.error('  HONG: so duong noi sai'); hong++; }
  if (demHat !== 1) { console.error('  HONG: so hat sai'); hong++; }

  // 3) NGUOI DUNG DUNG CHO CUA BAN MINH: chỗ thứ i trong phòng phải nằm trong
  //    long phong, khong loi ra ngoai vach.
  const phongThu = { khoa: 'code', x: 15, z: 0, kieu: 'ban', goc: Math.atan2(-15, 0) };
  ctx.dungPhong(phongThu);
  for (let i = 0; i < 8; i++) {
    const c = ctx.oTrongPhong(phongThu, i, 8);
    const dx = Math.abs(c.v.x - phongThu.x), dz = Math.abs(c.v.z - phongThu.z);
    if (Math.max(dx, dz) > 4.6) {
      console.error('  HONG: cho thu ' + i + ' loi ra ngoai phong', c.v.x, c.v.z);
      hong++;
    }
  }
  console.log(hong ? '\nCO ' + hong + ' CHO HONG' : '\nTAT CA KIEM TRA XANH');
  process.exit(hong ? 1 : 0);
}, 400);
