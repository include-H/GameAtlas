# GameAtlas Moonlight Web 浏览器串流 + 统一执行代理可行性设计

> 状态：**v2 修订**（2026-08-15；初稿仅覆盖串流链路，本次并入"统一执行代理"设计，形成一条链两种执行形态）
> 定位：GameManager 开始游戏双选项（本地 bat / Moonlight 远程）之远程游玩的完整设计；统一执行代理（改造 exe）为推荐形态
> 关联：`backend/internal/services/windows_launch.go`（现状 bat 生成）、feature/gamevhd-runtime-sandbox 分支（GameVHD launcher、差分盘、gvhook）、`docs/GameVHD_DXVK集成设计.md`
> 蓝本：`royka1/moonlight-webclient`（moonlight-common-c 编 WASM + WebCodecs 解码 + 键鼠捕获，自研浏览器客户端参照）

## 0. TL;DR（先给结论）

- **目标**：开始游戏提供两个选项——① 下载 bat 本地游玩（现有）；② Moonlight 远程（浏览器直接玩）。两条选项共享同一条"游戏执行链"，只是交付方式不同。
- **一条链**：`点击远程启动 → WOL 唤醒（未开机）→ 等待主机就绪 → 串流连接 → 触发游戏执行 → 画面游玩 → 退出 → 生命周期清理与上报`。
- **两种执行形态**：
  - **形态 1（不启用 feature）**：现状 bat 链路——目标机下载 bat 手动运行，bat 自管挂载/启动/清理。**自动化有缺口**：串流连上后没有"监听者"等请求，无法远程自动拉起游戏。
  - **形态 2（推荐，启用 feature）**：改造 GameVHD launcher exe 为**统一执行代理**，常驻监听 Atlas 请求，吸收 bat 全部职责（挂载差分盘/启动/清理）+ 新增监听/上报能力。它补齐了形态 1 的缺口——**bat 是"人肉 agent"，改造 exe 是"机器 agent"**。
- **浏览器客户端**：不依赖原生 Moonlight，以 `moonlight-webclient` 为蓝本（moonlight-common-c → WASM + WebCodecs 硬解 + Pointer/Keyboard Lock 键鼠），或最小改动借壳 moonlight-web-stream。自研程度与工作量成反比，见 D6。
- **实施分四阶段**：P0 真机验证 → P1 自动启动 → P2 统一 Agent → P3 深集成。P0 未过不进 P1。

## 1. 背景与动机

### 1.1 现状与缺口

- 现有"开始游戏"只有下载 bat：`windows_launch.go` 生成 bat 壳（内嵌 base64 PS 主脚本），由目标机用户**手动**运行；bat 自管 SMB VHD 差分盘挂载、游戏进程、退出清理。
- 想加"Moonlight 远程"选项，但沿用手动 bat 无法闭环：串流画面连上后，**没有实体接收"启动游戏"请求**——人不在目标机前，bat 无人执行。
- 于是想到形态 2：把 feature 分支的 launcher exe 改造成服务——**既当远程 agent，也当本地执行器**，由 Atlas 统一编排。

### 1.2 为什么浏览器直接玩

- 免装 Moonlight 客户端：NAS 局域网内任何带浏览器的设备（笔记本/平板/电视）即开即玩。
- GameStream 协议需裸 TCP/UDP，浏览器无法直连，必须经本地代理（moonlight-web-stream 的 Rust streamer，或自研 Go 桥 + WASM 客户端）。

## 2. 统一链路全景

```text
┌─────────────────────── 控制面：GameManager（NAS 后端 + 前端）──────────────────────┐
│ 开始远程游玩 → WOL → 轮询探活 → 下发 launch → 会话记录 → 收到结束 → 时长入库          │
└──────────────┬───────────────────────────────────────────────────────┬─────────────┘
               │ HTTP + token                              HTTP + token │
   ┌───────────▼──────────┐                              ┌─────────────▼─────────────┐
   │ 流面：浏览器客户端      │  WebRTC(RTP UDP)  ┌─────────┴────────┐  │ 执行面：统一 Agent    │
   │ (WebCodecs+键鼠)      │◄─────────────────►│ Sunshine(游戏PC) │◄─┤ (改造 exe, 常驻监听)  │
   └───────────────────────┘   WebSocket 兜底   └──────────────────┘  │ 挂载差分→启动→清理→上报│
                                                                     └─────────────────────┘
```

- **控制面**（本项目）：编排、WOL、探活、会话、配置下发、反代（若借壳）/接口（若自研）。
- **流面**：浏览器播放器 + 协议桥（两选一，见 D6）。
- **执行面**：统一 Agent（形态 2）或手动 bat（形态 1 降级）。

### 2.1 全生命周期时序（形态 2）

```text
前端                 Atlas 后端(NAS)        统一 Agent(目标机)       Sunshine(游戏 PC)
 │  点击"远程游玩"       │                       │                        │
 │────────────────────► │                       │                        │
 │                      │── WOL magic packet ──►│ (UDP 9 广播, 未开机时)  │
 │                      │    ┌──────────────────┴────── 目标机开机 ──────┐
 │                      │◄── 轮询 /health 直至就绪 ──│ (agent 开机自启)      │
 │                      │                       │                        │
 │   ┌── 角色 B（有 Sunshine）：Atlas 下发游戏配置 ──►│ 监听并自动生成        │
 │   │                    │                       │ Sunshine app 配置→热重载 │
 │   │                    │ 串流照旧（浏览器点开即玩） │                        │
 │   └── 角色 A（无 Sunshine）：Atlas 下发启动任务 ──►│ 下载 exe → 挂载差分盘   │
 │                      │                       │ 启动游戏（原 bat 行为）     │
 │                      │                       │ 状态上报 running          │
 │◄── 串流地址/会话 id ──┤                       │                        │
 │ 打开串流画面          │◄──────────────────────│──── RTP 编码流 ──────────│
 │   游戏退出            │                       │◄── 进程退出检测 ──────────│
 │◄── stream-end ────────┤                       │                        │
 │                      │                       │ 清理差分层 → 上报时长     │
 │                      │── 会话入库(开始/结束/时长) ────────────────────────►│
```

## 3. 统一执行代理（改造 exe，形态 2 核心）

### 3.1 角色矩阵（同一 exe，按目标机是否装 Sunshine 区分角色）

两种角色的**行为模式不变**：挂载差分盘 → 启动 → 清理（GameVHD 既有逻辑），区别在"配置从哪来、是否走串流"。

| 角色 | 部署位置 | 配置来源 | 职责 |
|---|---|---|---|
| **角色 A：本地执行** | 目标机**未装 Sunshine** | **exe 自携带**（内嵌该游戏的 SMB 路径、账号密码、差分盘名、SavePath 等，同现有 bat 注入方式） | 获取 exe 即运行：自动挂载差分盘 → 启动游戏 → 清理（原 bat 行为模式的 exe 化，行为原样） |
| **角色 B：串流执行** | 目标机**装了 Sunshine** | **Atlas 动态推送**（游戏路径、账号密码、其他参数） | agent 常驻监听 → 收到配置即执行（生成 Sunshine app 配置/挂载启动）→ 串流玩法照旧 |

> **exe 就是 agent 本身，同时是配置载体**：角色 A 下配置内嵌（与 `windows_launch.go` 渲染 bat 时注入 SMB 凭据/VHD 路径同源），角色 B 下配置由 Atlas 推送（agent 只负责监听与执行）。
> 角色 B 的配置自动化：agent 把 Atlas 下发的游戏信息写成 Sunshine `apps.json` 条目（name = 游戏、cmd = 启动命令、imagePath = 封面），热重载后浏览器端 app 列表自动出现——用户**不需要手工进 Sunshine Web UI 配 app**。

### 3.2 与现有 bat 的关系

- bat 的职责全部被 Agent 吸收（挂载/启动/清理是同一套 GameVHD 逻辑），Agent 新增：常驻监听（角色 B）、动态配置（角色 B）。
- **bat 退化为降级通道**：Agent 不可用（无 feature 环境/未部署）时保持现状能力。
- **配置注入同源**：角色 A 的 exe 内嵌 SMB 凭据/VHD 路径等，与 `windows_launch.go` 的 `renderBatLauncher` 注入逻辑同一来源，生成侧复用。

### 3.3 协议与接口（草案）

- 传输：HTTP/JSON，局域网；鉴权 Bearer token（Atlas 设置页生成、可轮换）。
- 接口（Agent 侧）：
  - `GET /health` → 就绪探活（Atlas 轮询，含开机等待）
  - `POST /launch {game_id}` → 挂载差分 + 启动，返回会话 id
  - `POST /stop {session_id}` → 主动停止（保底，正常以进程退出为准）
  - `GET /status` → 当前会话状态
- 接口（Atlas 侧，Agent 回调）：`POST /api/agent/events`（started/stopped + 时长）——权威时长来源。
- 单例/防重入：差分盘锁复用 GameVHD 既有设计；同一游戏运行中禁止二次 launch。

### 3.4 生命周期状态机

```text
idle → launching(挂载差分盘) → running(游戏进程) → cleaning(回收差分盘) → idle
         │ 失败                    │ 进程退出/stop       │
         └─→ error(上报, 清理残留)  └───────────────────┘
```

### 3.5 服务形态与安全

- **常驻 exe + 开机自启注册**（非 Windows SCM Service）：权限要求低、调试/升级简单、卸载干净；启动失败自动拉起（计划任务/启动目录）。
- 仅监听局域网网卡；token 由 Atlas 下发并支持轮换；不做公网暴露（远程场景走内网穿透/Tailscale 另行评估）。

## 4. 串流接入链

| 环节 | 方案 | 备注 |
|---|---|---|
| WOL 唤醒 | Atlas 后端发 UDP 9 magic packet（需记录目标机 MAC） | Go stdlib 即可；目标机主板/网卡需开启 WOL |
| 等待就绪 | 轮询 Agent `GET /health`（含 Windows 登录延迟、agent 自启竞态，超时可调） | 就绪后可先做串流预连接 |
| 串流连接 | Sunshine Desktop app（桌面先行）或 Sunshine app 直启游戏 | 见 D9 两种模式 |
| 触发游戏 | Agent `POST /launch`（形态 2）/ 手动 bat（形态 1 降级） | 形态 1 自动化缺口即在此 |
| 画面 | 浏览器客户端（WebRTC/WebSocket） | 见 §5 |
| 退出 | 游戏进程退出 → Agent 清理上报 → Atlas 会话入库 → 前端关闭串流 | stream-end 仅做 UI 回跳 |

## 5. 浏览器客户端（流面）——两选一

| 选项 | 做法 | 工作量 | 适用 |
|---|---|---|---|
| **B. 自研（蓝本）** | 以 `royka1/moonlight-webclient` 为蓝本：moonlight-common-c 编 WASM（协议/配对/RTSP/Enet/RTP 全在 WASM 跑），浏览器只写 WebCodecs 解码（H.264/HEVC/AV1）+ Pointer/Keyboard Lock 键鼠 + UI；配对凭据存 Atlas DB，前端连 Atlas 的桥 | 1-2 周 | 深集成诉求（配对进 DB、自动 launch、悬浮窗、时长天然闭环）——与 Agent 形态契合 |
| **C. 借壳** | fork moonlight-web-stream 加 `?app=&autostart=1` + `parent.postMessage('stream-end')` | 1-2 天 | 快速验证、先跑通链路 |

- 视频：WebCodecs `VideoDecoder` 硬解 → VideoFrame → canvas/`MediaStreamTrackGenerator`；需自写 RTP depacketizer（AVCC 拼接，蓝本已有思路）。
- 音频：OPUS → WebCodecs `AudioDecoder`。
- 键鼠：Pointer Lock + Keyboard Lock（Chrome 全屏）+ Gamepad API。
- 浏览器不能发裸 UDP → 协议桥（WASM 内联的 WebSocket 中继，或借壳方案自带 streamer）省不掉；Direct Sockets API（Chrome 129+）可作 P3 试点，受 PNA/局域网限制。

## 6. 证据边界（可行性判断依据）

| # | 事实 | 证据来源 | 影响 |
|---|---|---|---|
| 1 | GameStream 需裸 TCP/UDP，浏览器不支持；官方无纯 Web 客户端 | moonlight-docs FAQ | 必须有本地代理/桥 |
| 2 | `royka1/moonlight-webclient`：moonlight-common-c → WASM + WebCodecs + 键鼠端到端可用（配对/launch/解码/输入全通） | 仓库 README（实测说明） | B 路线（自研）成立，非理论 |
| 3 | moonlight-web-stream v2：Rust server + streamer，WebRTC/WebSocket 双传输，H.264/HEVC/AV1，Docker | 仓库/Docker Hub | C 路线（借壳）成立 |
| 4 | Sunshine app 可指向任意命令/bat | Sunshine 文档 | 执行面可复用同一启动脚本 |
| 5 | Sunshine `/launch /resume /cancel` 需客户端配对证书；moonlight-webclient 与 moonlight-web 均已是配对客户端 | DeepWiki NVHTTP | 配对一次性，持久化即可 |
| 6 | 现有 bat 生成链路在 `windows_launch.go`（bat 壳 + base64 PS + SMB VHD 差分盘） | 代码 | Agent 复用同一套挂载/清理逻辑，成本低 |
| 7 | GameVHD launcher 为 Rust 编排器，零依赖（`pe.rs`、`hook_dll_for`、差分盘逻辑） | feature 分支、DXVK 设计文档 | 加 HTTP 监听 + token 鉴权即得 Agent |
| 8 | WOL：UDP 9 广播 magic packet，Go stdlib 可实现 | 通用事实 | 控制面成本极低 |
| 9 | WebCodecs/Keyboard Lock/Pointer Lock 在 Chromium 系浏览器成熟 | 平台事实 | 流面能力具备 |
| 10 | 目标机需 Agent 自启 + Windows 登录时序 | 平台事实 | 探活轮询需容忍登录延迟 |

## 7. 关键设计决策

| 决策点 | 结论 | 理由 | 风险 |
|---|---|---|---|
| D1 执行形态 | **形态 2（统一 Agent）为主**，形态 1 作降级保留 | Agent 补齐自动化缺口、共享 GameVHD 逻辑；角色 A（本地）/角色 B（串流）同一 exe 两种角色 | feature 分支需合入主线并维护 |
| D2 Agent 服务形态 | 常驻 exe + 开机自启，非 SCM Service | 权限/调试/升级简单 | 开机自启竞态需探活兜底 |
| D3 Agent 鉴权 | 局域网 + Bearer token（Atlas 生成/轮换） | 最小够用；不开放公网 | token 泄露面=局域网 |
| D4 会话权威来源 | Agent 上报（进程退出必然触发），前端 postMessage 仅 UI 回跳 | 可靠性优先；浏览器关页不丢数据 | 需 Atlas 事件接口幂等 |
| D5 WOL | Atlas 后端发 UDP 9；设置页维护 目标机 MAC + 关机状态 | Go stdlib，成本低 | 目标机 BIOS/网卡需开 WOL |
| D6 浏览器客户端 | **B（自研，蓝本）与 Agent 形态配套**；P0 阶段先 C 借壳跑通链路 | 深集成价值高；借壳快速验证 | B 有 1-2 周成本；C 是 fork，深集成受限 |
| D7 配置自动化（角色 B） | agent 写 Sunshine `apps.json`（name/cmd/imagePath）→ 热重载；不触碰 Sunshine 其他配置 | 免人工 Web UI 配 app；配置随 Atlas 游戏库增删自动同步 | apps.json 格式与热重载机制需 P2 实测（版本差异） |
| D8 配置来源 | 角色 A：exe 内嵌（同 bat 注入）；角色 B：Atlas 推送 | 角色 A 即"bat exe 化"，行为原样；角色 B 动态下发 | 内嵌凭据随 exe 分发（与现有 bat 同风险面，不新增）；推送链路需 token 保护 |
| D9 双开互斥 | 统一"运行中"状态位：本地与远程共享 | 差分盘锁防双写 | 状态位一致性（异常退出需超时回收） |
| D10 配对 | 一次性人工配对（借壳）/ WASM 内配对凭据存 Atlas DB（自研） | 不实现 NvHTTP 配对协议 | 换主机需重配（文档化） |
| D11 串流模式 | 双模式：桌面先行（登录/操作）与 Sunshine app 直启游戏（纯游戏） | 覆盖"先看桌面"与"直接玩"两种诉求 | 直启模式需 Agent 与 Sunshine 同一进程环境（同机） |

## 8. 实施阶段（P0 未过不进 P1）

### P0 真机手工验证（零代码）
1. NAS 起 moonlight-web-stream Docker（`WEBRTC_NAT_1TO1_HOST` + UDP 端口段）→ 浏览器串流手工跑通
2. 核 moonlight-web-stream 前端源码：`?app=` 自动启动与退出 postMessage 的改动成本（决定 P1 是借壳还是直接上自研）
3. 实测：延迟观感（1080p60）、WebSocket 兜底、手柄（Gamepad API）、退出流程
4. 验收：局域网 1080p60 可玩、配对持久化、WebSocket 可通

### P1 自动启动（串流链闭环）
1. 借壳方案：fork 前端加 `?host=&app=&autostart=1` + `parent.postMessage('stream-end')`
2. Atlas 设置页：目标机配置（主机地址、MAC、app 映射）+ 后端 WOL + 探活轮询
3. 游戏详情"开始游戏"加"Moonlight 远程游玩"→ WOL → 探活 → 全屏 iframe（带 autostart）→ stream-end 回跳
4. 会话兜底：轮询 Sunshine 会话状态

### P2 统一 Agent（执行面落地）
1. feature 分支 launcher 加 HTTP 监听 + token 鉴权 + `/health /launch /stop /status`（复用差分盘挂载/清理逻辑）
2. **角色 A（本地执行）**：exe 内嵌游戏配置（SMB 凭据/路径，同 `renderBatLauncher` 注入）→ 获取即运行，挂载差分盘 → 启动 → 清理
3. **角色 B（串流执行）**：Atlas 动态推送配置 → agent 监听执行（生成 Sunshine `apps.json` 条目 → 热重载 → 浏览器 app 列表自动出现，D7）
4. Agent 开机自启注册；Atlas 事件接口（started/stopped 幂等入库）
5. "运行中"状态位统一（D9）
6. 串流双模式验证（D11）：桌面先行 / app 直启（app cmd = agent client 模式）

### P3 深集成（可选）
- 自研浏览器客户端（B 路线，蓝本 royka1）替换借壳，配对凭据入 Atlas DB
- Direct Sockets 试点（去桥直连）；登录态透传；断线 `/resume` 自动重连
- Agent 自更新、多主机、公网（Tailscale）评估

## 9. 测试矩阵（P2 终点证据）

| 类别 | 覆盖项 | 终点证据 |
|---|---|---|
| 启动链 | 按钮 → WOL(未开机) → 探活 → Agent launch → 画面 | 全流程无人为点击；各失败点（未配 MAC/主机离线/agent 未装）有明确提示与修复指引 |
| Agent | launch/stop/status、差分盘挂载与清理、进程退出检测、异常退出残留回收 | 状态机各态迁移正确；清理幂等；残留可诊断 |
| 会话 | Agent 上报 started/stopped → Atlas 入库 | 幂等；前端关页/断网不丢时长 |
| 并发 | 本地与远程互斥、重复 launch 拒绝 | 差分盘锁生效；状态位统一 |
| 传输 | WebRTC / WebSocket 兜底 | 双模式可串流；WS 延迟标注"可用"非"最佳" |
| WOL | 关机→唤醒→探活时序、开机慢机器 | 轮询超时可配；失败路径清晰 |
| 降级 | 无 feature 环境走 bat | 现状回归不受影响 |

## 10. 未闭合的门（诚实标注）

1. **moonlight-web-stream 前端源码未核**（`?app=`/postMessage 改动成本）——P0 第 2 步必须确认；成本高则 P1 改"新标签页 + 手动点击"降级
2. **royka1/moonlight-webclient 自研蓝本的实测细节未验证**（AVCC 拼接、MSTG、键鼠手感）——需 P0 后单独 spike 评估 B 路线
3. Agent 与 Sunshine 同机的进程环境耦合（桌面会话/服务会话的显示环境差异）——Windows 服务与交互式会话问题，P2 实测
4. 浏览器 UDP 直连 NAS 的 NAT/防火墙未知数——局域网默认可通，兜底 WebSocket
5. moonlight-web/Sunshine 上游兼容漂移（`sessionUrl0` 等历史 issue）——锁版本实测，持续跟进
6. WOL 依赖目标机主板/网卡设置（BIOS + 网卡属性），非软件可控——文档化前置条件
7. Agent 开机自启与 Windows 登录/锁屏时序竞态（登录前无桌面会话）——探活超时 + 状态机兜底
8. 自研 B 路线的手柄/多设备兼容矩阵未建立（Gamepad API 差异）
9. feature 分支与主线合入策略未定（Agent 是 feature 的收割点之一）
10. 角色 A 的 exe 内嵌 SMB 凭据随文件分发（与现有 bat 风险面一致，不新增暴露；未来可换 per-user 令牌）

## 11. 参考

- 自研蓝本：`royka1/moonlight-webclient`（moonlight-common-c → WASM + WebCodecs + 键鼠）
- 借壳对象：`MrCreativ3001/moonlight-web-stream`（v2.6+，Docker: `mrcreativ3001/moonlight-web-stream`）
- 主机端：`LizardByte/Sunshine`；NVHTTP `/launch /resume /cancel` 认证语义见 DeepWiki
- 执行面现状：`backend/internal/services/windows_launch.go`；feature/gamevhd-runtime-sandbox（launcher、差分盘、gvhook）；`docs/GameVHD_DXVK集成设计.md`
- 官方 FAQ（无纯 Web 客户端原因）：`moonlight-stream/moonlight-docs` wiki
