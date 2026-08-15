# GameAtlas Moonlight Web 浏览器串流可行性设计

> 状态：**初稿**（2026-08-15，DeepSeek 产出，待评审）
> 定位：GameManager 开始游戏双选项（本地 bat / Moonlight 远程）之远程游玩的可行性论证与实施路线
> 关联：`docs/GameVHD_DXVK集成设计.md`（同类可行性设计文风）、本地游玩 bat 生成链路（现状）
> 动机：不想为远程游玩装 Moonlight 客户端，希望"开始游戏 → 选 Moonlight 远程 → 浏览器直接玩"

## 0. TL;DR（先给结论）

- **目标**：开始游戏页提供两个选项——① 下载 bat 本地游玩（现有）；② Moonlight 远程（浏览器直接玩，零客户端）。
- **可行性：成立**，但它是三方协作的链路，不是自研播放器：
  - **宿主面**：Sunshine（游戏 PC）——app 启动（可指向 GameManager 生成的 bat）、NVENC 编码、客户端配对。
  - **流面**：`moonlight-web-stream`（第三方，Rust + TS）——浏览器 WebRTC 播放器，配对持久化，有 Docker 镜像，WebSocket 传输兜底。
  - **控制面**：GameManager 后端——"开始游戏 → 远程"编排、会话/时长记录、iframe 全屏集成。
- **关键结论与坑**（详见 §4 决策表）：
  - 🔴 浏览器无法直连 GameStream 协议（需裸 TCP/UDP），**必须有本地代理**——moonlight-web 就是这层代理，不可省略。
  - 🟠 自动启动需 **fork moonlight-web 前端** 加 `?app=&autostart=1` 参数（TS 前端，改动小；源码未核）。
  - 🟠 时长统计的权威来源用 **Sunshine app 的"运行前/后"命令回调 GameManager API**，而非前端 postMessage（app 退出必然触发，postMessage 只做 UI 回跳）。
  - 🟡 延迟比原生 Moonlight 略高，局域网内可接受；动作竞技类不保证。
- **实施分三阶段**：P0 真机手工验证（零代码）→ P1 自动启动（fork 前端 + 按钮）→ P2 完整闭环（会话记录 + 脚本复用）。P0 未过，不进入 P1。

## 1. 背景与动机

### 1.1 为什么做

- 现有"开始游戏"只有下载 bat 本地游玩：客户端机器要能访问 NAS 游戏库（或 GameVHD 沙箱），且只在装了配套环境的机器上可用。
- 场景诉求：NAS 局域网内任何设备（笔记本、平板、电视浏览器）不装任何客户端，浏览器打开即玩。
- 配套生态：Sunshine 在游戏 PC 上运行、浏览器有成熟的 WebRTC 串流客户端（moonlight-web-stream），GameManager 缺的是把三者编排起来的"总指挥"。

### 1.2 为什么不自研播放器

- GameStream 协议要求裸 TCP/UDP socket + NVENC 流处理 + 低延迟编解码，浏览器不具备，自研 WebRTC 桥是巨型工程。
- moonlight-web-stream 已实现：WebRTC/WebSocket 双传输、H.264/HEVC/AV1、配对流程、流统计、i18n（含中文），454★ 且持续更新。性价比远超自研。

### 1.3 为什么把"启动"交给 Sunshine 而不是自管

- Sunshine 的 app 天然就是"任意命令/bat"，与 GameManager 已生成的本地游玩脚本同构——远程游玩复用同一脚本，只是执行者在游戏 PC 侧。
- Sunshine 的 `/launch`、`/resume`、`/cancel` 需客户端配对证书认证，而 moonlight-web 本身已是持久化配对的客户端，**GameManager 不需要持有 Sunshine 凭据**，也不实现配对协议。

## 2. 架构总览

```text
浏览器（客户端，零安装）
  │  打开 GameManager → "开始游戏 → Moonlight 远程" → 全屏 iframe
  │  WebRTC UDP（媒体流，浏览器直连 NAS，不经反代）   ←┐
  ▼                                                    │
GameManager 后端（NAS，控制面）                         │
  ├─ 编排：保活/拉起 moonlight-web、记录会话、API       │
  └─ 反向代理 moonlight-web 的 HTTP(S)（同源出 /stream）│
       ▼                                                │
moonlight-web（NAS，流面）—— web-server + streamer 子进程
  │  RTSP/Enet/RTP（GameStream 协议，局域网）            │
  ▼                                                    │
Sunshine（游戏 PC，宿主面）→ 启动 app（GameManager bat 脚本）→ NVENC 编码
```

三个面各自独立、接口最小：

| 面 | 组件 | 职责 | 状态 |
|---|---|---|---|
| 控制面 | GameManager 后端 | 按钮编排、会话记录、iframe 宿主、反向代理 | 本项目（P1/P2） |
| 流面 | moonlight-web-stream | 播放器 + GameStream 协议桥 | 第三方（fork 前端） |
| 宿主面 | Sunshine + GameManager 脚本 | app 启动、编码、配对 | 配置（P0 验证） |

## 3. 证据边界（可行性判断依据）

| # | 事实 | 证据来源 | 影响 |
|---|---|---|---|
| 1 | GameStream 需要裸 TCP/UDP socket，浏览器不支持；官方无纯 Web 客户端 | moonlight-docs FAQ | 必须有本地代理（moonlight-web），不可省略 |
| 2 | moonlight-web-stream v2：Rust web-server + streamer 子进程，WebRTC 传输，WebSocket 兜底，H.264/HEVC/AV1，Docker 镜像 | 仓库 README、Docker Hub | 可行；锁 v2.6+（含信令竞态修复、i18n） |
| 3 | Sunshine app 可指向任意命令/批处理 | Sunshine 文档 | 与 GameManager 本地 bat 复用，双选项同源 |
| 4 | Sunshine `/launch` `/resume` `/cancel` 需客户端配对证书；moonlight-web 自身为持久化配对客户端（data.json） | DeepWiki NVHTTP 章节 | GameManager 不碰配对/凭据 |
| 5 | `/resume` 支持加入已运行会话 | DeepWiki NVHTTP 章节 | 可作为"断线重连"体验的备选路径 |
| 6 | moonlight-web 前端为 TS；`?app=&autostart=1` 与 `parent.postMessage` 支持**未核实** | 仓库结构（待 P0 后核源码） | fork 改造点，需 P0 后代码验证 |
| 7 | moonlight-web 需暴露 UDP 端口范围（Docker 默认 40000-40100/udp） | Docker Hub | iframe 反代只管 HTTP(S)；UDP 由浏览器直连 NAS |
| 8 | Docker 部署需 `WEBRTC_NAT_1TO1_HOST=LAN_IP` 供浏览器协商 | Docker Hub | NAS 部署时必配；P0 即验证 |

## 4. 关键设计决策

| 决策点 | 结论 | 理由 | 风险 |
|---|---|---|---|
| D1 部署位 | moonlight-web 作为 **NAS 独立服务（Docker 优先）**；GameManager 不内嵌二进制、不管理其子进程 | 进程/升级/数据（data.json）归 Docker 管，GameManager 只做入口 | NAS 需开 8080 + UDP 端口段；版本升级由用户 Docker 侧执行 |
| D2 配对/账号 | 一次性人工配对；GameManager 设置页只配置"主机地址 + 游戏 app 名映射"，不实现 NvHTTP 配对 | 配对协议复杂且 moonlight-web 已内置；持久化在它 data.json | 换主机/重装 moonlight-web 需重新配对（罕见，文档化） |
| D3 自动启动 | **fork moonlight-web 前端**加 `?app=&autostart=1`（加载后自动定位主机并点击 launch） | moonlight-web 本身就是已配对客户端，自己 launch 最干净；GameManager 不碰 Sunshine 凭据 | fork 上游合并成本；改动点需 P0 后核源码确认 |
| D4 时长统计 | **Sunshine app "运行前/后"命令回调 GameManager API**（POST /api/stream-session），作为权威来源；前端 postMessage 仅做 UI 回跳 | app 退出必然触发（进程结束钩子），比前端事件可靠；前端关闭页面也不丢数据 | 需给每个游戏 app 配钩子（可脚本生成，见 D7） |
| D5 结束回跳 | iframe 全屏 + `parent.postMessage('stream-end')` → GameManager 收事件关全屏回游戏详情 | 同源反代下 postMessage 安全可控 | 浏览器关页/断网时收不到——兜底由 D4 的权威时长补 |
| D6 集成形态 | GameManager 后端反代 moonlight-web HTTP(S) 到 `/stream/` 同源路径，iframe 嵌入；WebRTC UDP 不经反代 | 同源免跨域、免 TLS 自签证书警告；UDP 媒体流浏览器直连 NAS 局域网可达 | 反代需正确透传 WebSocket（升级头）；Nginx/Go 均可，Go 自带 revproxy 即可 |
| D7 脚本复用 | 生成本地游玩 bat 的同时，产出该游戏 Sunshine app 配置片段（命令指向同一 bat 的远程路径），并提供批量导入指引 | 双选项同脚本，行为一致；避免维护两套启动逻辑 | Sunshine app 是手导配置（无 API 写配置）——P2 提供"复制片段"而非自动配置 |
| D8 并发互斥 | 同一游戏"本地 bat"与"Moonlight 远程"**禁止同时启动**（GameVHD 差分盘锁/运行中标记） | 防止双写冲突（沙箱盘、存档） | GameManager 需统一"运行中"状态位，两选项共享 |

## 5. 分阶段实施（P0 未过不进 P1）

### P0 真机手工验证（零代码，1-2 小时）

1. NAS 起 moonlight-web-stream Docker（配 `WEBRTC_NAT_1TO1_HOST` + UDP 端口段）
2. 浏览器首次访问：建 admin 账号 → 添加主机（游戏 PC 地址）→ 配对（Sunshine PIN）
3. 手工启动一个 app，实测：延迟观感、WebSocket 兜底切换、音频/手柄（浏览器 Gamepad API）、退出流程
4. **核源码**：前端是否易加 `?app=` 自动启动与退出 postMessage（决定 D3 改动量）
5. 验收标准：局域网内 1080p60 基本可玩（延迟主观可接受）、配对持久化（重启后免配）、WebSocket 模式可通

### P1 自动启动（fork 前端 + GameManager 按钮）

1. fork moonlight-web-stream：加 URL 参数 `?host=&app=&autostart=1`、流结束 `parent.postMessage('stream-end')`、可选 `?no-auth-session=1` 直通（若实现成本低）
2. GameManager 设置页：Moonlight 主机地址 + 游戏→app 名映射 + 手动配对入口指引
3. 游戏详情"开始游戏"加第二选项"Moonlight 远程游玩"→ 后端校验配置 → 打开全屏 iframe（`/stream/?host=..&app=..&autostart=1`）
4. 监听 `stream-end` → 关 iframe 回详情；丢失兜底：轮询 Sunshine 会话状态（可选）

### P2 完整闭环（会话与脚本打通）

1. bat 生成时同时产出 Sunshine app 配置片段（app name = GameManager 游戏 ID，命令 = 远程路径 bat），设置页展示导入步骤
2. Sunshine "运行前/后"命令：`curl POST NAS/api/stream-session/start|end`（携带 app 名）→ GameManager 记录开始/结束/时长，游戏详情展示远程游玩历史
3. "运行中"状态位统一：本地 bat 与远程互斥（D8）
4. 测试矩阵落地（§7）

### P3 加分项（可选）

- GameManager 后端按需拉起/保活 moonlight-web（Docker API 或 systemd），免用户手动起服务
- 登录态透传（moonlight-web 会话复用）；多主机选择；断线自动 resume（利用 `/resume`）

## 6. 未闭合的门（诚实标注）

1. **moonlight-web 前端源码未核**：`?app=` 自动启动、退出 postMessage 的改动成本是 P1 的前提，P0 第 4 步必须确认；若改动成本高，退路是 P1 改为"打开新标签页 + 手动点击"（体验降级但可用）
2. 浏览器端 UDP 直连 NAS 的 NAT/防火墙未知数：局域网默认可通；若不通，退 WebSocket 兜底（同端口 80/443，WebSocket 传输）
3. moonlight-web 与最新版 Sunshine 的兼容漂移（历史 issue：`sessionUrl0`、H265 启动失败）——锁 moonlight-web v2.6+ 并实测；上游修复节奏需持续跟进
4. 延迟实测数据缺失：P0 建立主观基线（局域网 1080p60）；若不可接受，只能回归原生 Moonlight 客户端（功能保留但选项无意义）
5. moonlight-web 自带账号体系（admin 用户 + 主机列表在 data.json）与 GameManager 用户体系隔离——多用户场景需 P3 处理或接受"家庭共享一个入口"
6. fork 的上游合并成本：fork 分支需定期 rebase 上游 release（或锁版本放弃更新）
7. 手柄支持依赖浏览器 Gamepad API 与 moonlight-web 的映射质量——P0 实测，不达标则标注"仅键鼠游玩"
8. Sunshine app 配置是手动导入（无 API），批量游戏迁移脚本的成本未评估（P2 提供片段复制已是最小成本路径）

## 7. 测试矩阵（P2 终点证据）

| 类别 | 覆盖项 | 终点证据 |
|---|---|---|
| 启动链 | 按钮 → iframe → autostart → 画面出现 | 全流程无人工点击；失败路径（未配对/主机离线/app 名错）有明确错误提示 |
| 会话 | 游戏退出 → Sunshine 后钩子 → GameManager 时长入库 | app 退出必然产生一条 session 记录；前端关闭/断网不丢（钩子兜底） |
| 回跳 | `stream-end` → iframe 关闭回详情 | 正常退出回跳成功；异常（浏览器被杀）由会话钩子补 |
| 并发 | 本地 bat + 远程互斥 | 运行中状态位阻止双开；释放后恢复 |
| 传输 | WebRTC 直连 / WebSocket 兜底 | 两种模式均可串流；WebSocket 下延迟标为"可用"非"最佳" |
| 配置 | 主机映射缺失 / app 名不存在 / 未配对 | 每个错误有可执行的修复指引 |
| 兼容 | 最新 Sunshine 版本、HEVC、AV1、1080p60 | P0 实测基线逐项过 |

## 8. 参考

- moonlight-web-stream：`MrCreativ3001/moonlight-web-stream`（v2.6+，Docker: `mrcreativ3001/moonlight-web-stream`）
- 官方 FAQ 无纯 Web 客户端：`moonlight-stream/moonlight-docs` wiki
- Sunshine：`LizardByte/Sunshine`；NVHTTP `/launch /resume /cancel` 认证语义见 DeepWiki 章节
- 相关 fork 佐证可改造性：`royka1/moonlight-webclient`（WASM 方案，成本更高，备选）、`Argon2000/moonlight-web-stream-tsla`（特斯拉浏览器适配 fork，佐证前端可深度改）
- 本地游玩 bat 生成链路：GameManager 现有"开始游戏 → 下载 bat"逻辑（复用脚本，见 D7）
