# Bảng nghiên cứu — dashboard agent orchestration (cho sagent)

Mục tiêu: học cách làm 2D + 3D **hiện đại, trực quan, custom sâu**, ưu tiên cái **bê thẳng được vào stack nhúng-binary** của sagent (vanilla JS, không Node build, offline).

**Cột "Hợp stack":**
- ✅ **Bê thẳng** — vanilla / zero-dependency (không phụ thuộc thư viện), đọc là port được vào Go embed.
- 🟡 **Học ý tưởng** — React/Next → phải `build` ra static assets rồi mới nhúng; học pattern là chính.
- 🔵 **Khác nền tảng** — Rust / desktop / TUI (terminal UI) — học kiến trúc & tư duy, không port trực tiếp.

---

## Nhóm A — Cùng thể loại "agent = nhân vật/orb trong một thế giới" (đối chuẩn trực tiếp view 3D)

| Dự án | Học được gì | Hợp stack | Link |
|---|---|---|---|
| **rafapetter/agent-town** | Pixel-art agent viz **zero-dependency, framework-agnostic**. Một file `live-dashboard.html` mở thẳng browser, tự nối WebSocket, agent thành nhân vật. Bridge server là script Node zero-dep. Đọc **một** file là hiểu trọn kỹ thuật. | ✅ | github.com/rafapetter/agent-town |
| **pixel-agents-hq/pixel-agents** (pablodelucca) | Repo "đầu đàn" của genre. Điểm học là **kiến trúc mở rộng**: interface `HookProvider` có kiểu (typed) làm ranh giới tích hợp — thêm agent CLI mới = thêm một thư mục con, không viết lại. Đúng bài toán 5-provider của sagent. | 🟡 | github.com/pixel-agents-hq/pixel-agents |
| **IvanWng97/pixtuoid** | "Pixel-art office" viết **Rust, single binary, zero terminal deps** — cùng triết lý một-file như Go của anh, hỗ trợ đúng bộ provider gồm Antigravity. Có themes, sound, minimap. Bản "anh em Rust" của sagent. | 🔵 | github.com/IvanWng97/pixtuoid |
| **askmojo/moltcraft** | Isometric (góc chéo) pixel dashboard, agent đi lại + "đào token", **bấm vào toà nhà để xem data thật**. Học tương tác click-để-xem-chi-tiết trong scene. | 🟡 | github.com/askmojo/moltcraft |
| **liuyixin-louis/agentroom** | Canvas 2D game engine có **BFS pathfinding** (nhân vật tự tìm đường), office lưu layout theo từng project, token dashboard realtime. Học cho orb có *hành vi* thay vì đứng yên. | 🟡 | github.com/liuyixin-louis/agentroom |

---

## Nhóm B — Học "LÀM CHO ĐẸP": design system + hiệu ứng (Canvas 2D thuần)

| Dự án | Học được gì | Hợp stack | Link |
|---|---|---|---|
| **maharshi-coding/agent-pixel-visualizer** | **Mỏ vàng cho câu hỏi "sao cho đẹp".** README liệt kê thẳng design system: glassmorphism (kính mờ), CRT scanlines (vệt quét màn hình cũ), aurora ambient layer (lớp sáng nền), bento grid, holographic status rings (vòng trạng thái ánh kim), data pulses, micro-interactions. Tất cả trong `webview.js` + `webview.css` **Canvas 2D thuần, không framework**. Bê thẳng hiệu ứng được. | ✅ | github.com/maharshi-coding/agent-pixel-visualizer |

---

## Nhóm C — Học "TRỰC QUAN + CUSTOM SÂU": flow / DAG / dashboard 2D giàu thông tin

| Dự án | Học được gì | Hợp stack | Link |
|---|---|---|---|
| **hoangsonww/Claude-Code-Agent-Monitor** | Nặng đô nhất về data-viz: **D3.js** với 11 section tương tác (orchestration DAG, Sankey diagram, collaboration network, error propagation map), Kanban board, analytics. Hình mẫu cho dashboard 2D *nhồi nhiều thông tin mà vẫn đọc được* — đúng cái view 3D đang thiếu. | 🟡 | github.com/hoangsonww/Claude-Code-Agent-Monitor |
| **patoles/agent-flow** | Realtime viz agent "nghĩ / rẽ nhánh / phối hợp". Đúng khái niệm `flow: doi-4` trong sagent, làm ra hồn: biến execution từ hộp đen thành luồng nhìn được. | 🟡 | github.com/patoles/agent-flow |
| **simonstaton/ClaudeSwarm** (AgentManager) | **Interactive SVG tree** topology cha-con của agent, tô màu theo status, token usage trên mỗi node + SSE streaming. SVG nhẹ hơn 3D nhiều, có thể là bản thay thế thực dụng cho view 3D. | 🟡 | github.com/simonstaton/ClaudeSwarm |
| **disler/claude-code-hooks-multi-agent-observability** | Mẫu **observability qua hook event**: swim lane theo agent, pulse chart mật độ hoạt động, task lifecycle. Học cách *cấu trúc luồng sự kiện* để đổ vào cả 2D log lẫn 3D. | 🟡 | github.com/disler/claude-code-hooks-multi-agent-observability |

---

## Nhóm D — Thư viện nền để "custom được nhiều" (không phải app, là công cụ)

| Thư viện | Dùng khi | Hợp stack | Ghi chú |
|---|---|---|---|
| **three.js** (examples chính thống) | Toàn bộ view 3D. Học từ examples chính thống thay vì tự mò. | ✅ (vendor 1 file) | threejs.org/examples — chú ý phần sprites/points, fog, postprocessing |
| **React Flow / xyflow** | Chuẩn công nghiệp cho **node/edge editor custom sâu**. Gallery ví dụ là nơi học tốt nhất về flow view. | 🟡 (React) | reactflow.dev — học ý tưởng nếu không muốn kéo React vào binary |
| **LiteGraph.js** | Node graph **vanilla, canvas**, không React. Bản "React Flow cho người nhúng-binary". | ✅ | github.com/jagenjo/litegraph.js |
| **Drawflow** | Flow/node editor **vanilla, siêu nhẹ**, dễ nhúng. | ✅ | github.com/jerosoler/Drawflow |
| **Rete.js** | Node editor **framework-agnostic**, engine xử lý luồng mạnh. | 🟡 | retejs.org |

---

## Chưa xác minh (thấy qua bảng tổng hợp, anh tự tra tên trên GitHub)

| Dự án | Vì sao đáng xem |
|---|---|
| **mindwalk** (~871 sao) | Replay session của coding-agent trên **bản đồ 3D của chính codebase** — đúng hướng 3D anh muốn giữ. Tôi thấy qua roundup, chưa mở repo trực tiếp. |

---

## Đọc theo thứ tự nào (nếu ít thời gian)

1. **agent-town** — nắm khung zero-dep + luồng WebSocket → nhân vật. (nền tảng)
2. **agent-pixel-visualizer** — lấy design system để nâng phần "đẹp". (thẩm mỹ)
3. **Claude-Code-Agent-Monitor** *hoặc* **ClaudeSwarm** — học cách 2D/SVG nhồi thông tin trực quan. (mật độ)
4. **LiteGraph.js** / **Drawflow** — nếu muốn view "workflow" (nút `Workflow` trên topbar sagent) custom sâu mà vẫn vanilla.
