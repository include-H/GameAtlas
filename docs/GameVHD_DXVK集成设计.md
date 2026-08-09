# GameVHD DXVK 集成设计（启动器"使用 DXVK 运行"选项）

> 状态：**SOL MAX 评审后修订版**（2026-08-08；初稿由 DeepSeek 产出，评审由 GPT-5.6-SOL MAX 完成，本文已合并评审结论）
> 定位基准：`docs/GameVHD_Runtime_实施计划.md`（v2 卡带式启动器）、`docs/GameAtlas_GameVHD_定位与意义.md`
> 关联：`gamevhd-runtime/`（feature/gamevhd-runtime-sandbox 分支，a9ac2e4 后）；评审会话 `019fe184-8679-75a0-b835-2f8d9c26278c`（gpt-5.6-sol）
> 动机：RDNA 架构 Windows 驱动对 DX11 以前游戏支持差，用 DXVK 将 D3D8-11 转译 Vulkan 绕开慢速 legacy 路径
> 修订说明：SOL 评审抓出初稿 12 处设计错误/遗漏（含 1 处致命：D3D12"天然回退"不成立），D1-D10 全部给出明确结论；本文以评审结论为权威。

## 0. TL;DR（先给结论）

- **目标**：启动器（`launcher.exe`）加"使用 DXVK 运行此游戏"选项，把 D3D8/9/10/11 游戏转译 Vulkan，绕开 RDNA 对旧 DX 的慢速驱动路径。
- **可行性：高，但约束比初稿判断的更严**：
  - 🔴 **D3D12 必须硬拒绝**（不是"天然回退"）——DXVK 虽无 `d3d12.dll`，但有 `dxgi.dll`，D3D12 游戏也会加载 dxgi，复制 DXVK dxgi 会污染 D3D12 行为。
  - 🔴 **PE 导入表不是权威**——delay-import / LoadLibrary / 子进程可绕过静态扫描；权威 API 集由 manifest 指定，PE 检测仅作校验提示。
  - 🟠 **关闭清理必须用安装日志 + 文件哈希 + 进程退出确认**——禁止"扫描文件名删除"（会误删游戏本体和第三方 wrapper）。
- **版本锁定**：v1 锁 **DXVK v2.7**（RDNA1/2 兼容优先），不追 v3.0。
- **实现成本：中**（比初稿估计高）：新增版本化包目录 + 安装日志 + 事务回滚 + `cleanup_pending` 状态机 + Vulkan 能力预检。
- **渲染验收只能真机**：Wine/virgl 无 Vulkan 物理设备，Linux 侧只验证控制面和失败路径。

## 1. 背景与动机

### 1.1 为什么做

- RDNA1/2 Windows 驱动不再收 feature 更新，DX9-11 只能走"slow legacy binding model"（DXVK 官方原话：*"suffers from severe performance issues"*）。
- DXVK 把 D3D 调用转译 Vulkan，而 Vulkan 是 AMD Windows 驱动的强项路径。
- 目标游戏：**D3D8/9/10/11**（老游戏、模拟器、经典单机）。D3D12 游戏（如 HZD）**明确不支持 DXVK 模式**。

### 1.2 为什么在启动器里做

- 沙箱游戏跑在 VHD 差分盘上，游戏目录默认不可由用户直接改。
- 启动器是唯一能"按游戏配置 + 按位数 + 按 API + 按 GPU 能力自动决策"的入口。
- 与 gvhook 分发模式一致。

## 2. 证据边界（可行性判断依据）

| # | 事实 | 证据来源 | 影响 |
|---|---|---|---|
| 1 | DXVK v3.0 支持 D3D8/9/10/11（含 SM1-3、fixed-function）；v2.7 支持 D3D9/10/11 | DXVK release notes | 若锁 v2.7，**D3D8 需真机验收，未过则该 API 标不可用** |
| 2 | DXVK v2.7+ 要求 Vulkan 1.4 + `VK_KHR_maintenance5` | DXVK release notes | 需 Vulkan 能力预检；RDNA1/2 老驱动可能不满足 |
| 3 | RDNA1/2 Windows 驱动停止 feature 更新 | DXVK 官方 | 动机成立且长期成立 |
| 4 | DXVK 是纯文件，无注册表/服务依赖 | DXVK Wiki | 与沙箱架构兼容 |
| 5 | DXVK dll 清单含 `dxgi.dll`，无 `d3d12.dll` | 发布包 | 🔴 **不能据此宣称 D3D12 自动回退**（dxgi 会污染 D3D12） |
| 6 | Windows 上跑 DXVK 社区验证可行，官方不正式支持 | DXVK Wiki "Windows" | 可行；README 白名单说明 |
| 7 | Windows DLL 搜索顺序：exe 目录优先 | Windows 加载器 | 复制到渲染 EXE 目录即可命中 |
| 8 | 现有 Rust 编排器零依赖；已有 `pe.rs`（位数探测）、`hook_dll_for(bits)` | 代码 | 检测与选择逻辑可复用 |

## 3. SOL 评审发现：初稿的 12 处设计错误/遗漏

| # | 问题 | 严重度 | 修正方向 |
|---|---|---|---|
| 1 | **"D3D12 天然回退"不成立**——DXVK 有 dxgi.dll，D3D12 游戏也会加载它 | 🔴 致命 | D3D12 硬拒绝；不复制任何 DXVK 文件，清空 DXVK 环境变量 |
| 2 | **PE 导入表不能作为唯一 API 判断**——delay-import/LoadLibrary/子进程可绕过 | 🟠 | PE 检测 = 校验提示；权威 API 集由 manifest 指定 |
| 3 | **复制缺所有权与冲突保护**——可能覆盖 ReShade/Special K/游戏自带 wrapper | 🟠 | 目标已存在则 v1 拒绝覆盖；原子写入 + 哈希校验 |
| 4 | **关闭语义不安全**——"扫描删除 dll"会误删用户文件 | 🟠 | 安装日志 + 文件哈希 + 进程退出确认；只删自有且未变文件 |
| 5 | **配置项不应作全局默认**——descriptorBuffer/relaxedBarriers 会改渲染正确性 | 🟡 | 仅白名单逐游戏 manifest 覆盖；默认不强制 |
| 6 | **dxvk.conf 位置与环境继承未定义** | 🟡 | `GameData/LauncherState/DXVK/config/`，绝对路径 `DXVK_CONFIG` 指定 |
| 7 | **未处理 launcher ≠ 渲染进程**——dll 必须放实际加载 D3D 的进程 EXE 目录 | 🟠 | manifest 指定实际渲染 EXE（`dxvk_targets`） |
| 8 | **无 Vulkan 能力预检**——32 位还需 32 位 Vulkan loader/ICD | 🟠 | `vulkan_probe::check(bits, 1.4, [maintenance5])`；不满足则拒绝 |
| 9 | **无事务回滚**——hook 失败/复制失败/提前退出时状态未定义 | 🟠 | 注入失败不 Resume，回滚安装事务 |
| 10 | **box.json 不当唯一权威**——在差分盘可被游戏改/重置 | 🟡 | 用户设置在主机侧 profile（game_id + base_vhd_identity）；box.json 只存快照 |
| 11 | **无包完整性/版本/许可证定义** | 🟡 | 版本化包目录 + package.json（版本、API、位数、哈希、许可证） |
| 12 | **无并发/残留/锁定/清理失败状态** | 🟡 | 新增 `cleanup_pending` 状态；清理失败阻止声称"原生运行" |

## 4. D1-D10 决策结论（SOL 权威）

| 决策点 | 结论 | 理由 | 风险 |
|---|---|---|---|
| D1 分发 | **启动器安装目录下的版本化本地包**（`launcher/dxvk/2.7/`）；不下载、不内嵌 | 离线可用、版本可复现、符合 gvhook 模式 | 包缺失/哈希错必须明确报错；需许可证 |
| D2 版本 | v1 锁 **DXVK v2.7** | RDNA1/2 兼容优先；v3.0 驱动要求更高 | v2.7 也要 Vulkan 1.4；D3D8 未真机验收前标不可用 |
| D3 DLL 目标 | **实际渲染 EXE 所在目录的差分层覆盖路径**；不改 PATH、不用 AddDllDirectory | EXE 目录优先级最高；后两者不可靠且影响其他 DLL | 已有 wrapper 时拒绝启用 |
| D4 Shader cache | `GameData/LauncherState/DXVK/cache/<exe-id>/<package-id>/` + `DXVK_STATE_CACHE_PATH`；**跨重置保留** | 避免重复编译；按 EXE/包版本隔离 | 缓存膨胀；需大小上限 |
| D5 关闭清理 | 安装日志 + 文件哈希；只删本次创建且未变的文件；**已有文件绝不覆盖/删除** | 防误删游戏本体和 wrapper | 文件被改/锁定 → 留 orphan 进 `cleanup_pending` |
| D6 配置 | `GameData/LauncherState/DXVK/config/`，绝对路径 `DXVK_CONFIG`；同时设日志/缓存路径 | 不污染游戏目录、不碰撞已有 dxvk.conf | 游戏可能清环境变量；v2.7 实机确认优先级 |
| D7 Hook 交互 | 文件准备在 `CreateProcess(CREATE_SUSPENDED)` **前**完成；默认 **Early-Bird APC**，CRT 作受控回退；注入成功才 Resume | 游戏用户代码/D3D 设备创建前已有隔离规则 | hook 重定向可能误伤 DXVK 配置/缓存/Vulkan loader——需系统路径豁免 |
| D8 32 位 | x86 一等目标：x86 DXVK + x86 Vulkan probe；无 32 位 loader/ICD 则拒绝 | DXVK 无必然 x86 禁止项，真问题是位数匹配 | 32 位 wrapper/子进程失败率高；需真机证据 |
| D9 测试 | Linux/Wine 只验证**控制面和失败路径**；渲染/全屏/驱动行为必须真机 | virgl 无物理 Vulkan 设备，不能证明渲染正确 | CI 无法覆盖 AMD 老驱动/HDR/VRR |
| D10 API 检测 | PE 检测（普通导入 + delay-import）为**校验提示**；**权威 API 集由 manifest 指定**；D3D12 硬拒绝；D3D11/D3D12 混合或未知动态 API 拒绝 | 解决动态加载与 launcher/child；防 dxgi 污染 | manifest 需维护；未知不猜测、不复制全套 |

## 5. 修改后设计（SOL 评审版）

### 5.1 配置模型

manifest 新增：

- `dxvk`: 后端给出的默认开关
- `dxvk_package`: 固定包 ID（如 `dxvk-2.7`）
- `dxvk_targets`: 实际渲染 EXE、位数、API 集（权威来源）
- `dxvk_config`: 仅白名单配置项（禁止任意路径/DLL 名）

用户开关写入**启动器主机侧 profile**（key = `game_id + base_vhd_identity`）。`box.json.use_dxvk` 仅作本次有效值快照（崩溃恢复/诊断），非唯一权威。

```text
launcher/
  dxvk/
    2.7/
      package.json      # 版本、支持的 API、位数、文件哈希、配置 schema、许可证
      x86/
      x64/
```

### 5.2 模块接口

```text
pe::inspect(exe) -> PeInfo            # bits, normal_imports, delay_imports,
                                      # d3d_api_set, has_d3d12, confidence
vulkan_probe::check(bits, min_version, required_extensions) -> Capability
dxvk::resolve_plan(package, target_exe, api_set, bits) -> DxvkPlan   # 只返回所需 DLL 闭包
dxvk::install(plan, overlay_root) -> InstallJournal                  # 原子写入 + 哈希 + 日志
dxvk::cleanup(journal) -> Cleaned | CleanupPending | Conflict
dxvk::env(plan) -> EnvPatch
```

API 闭包（resolve_plan 只复制所需，不复制全套）：

- D3D8：`d3d8.dll`
- D3D9：`d3d9.dll`
- D3D10：`d3d10*.dll` + `d3d10core.dll` + `dxgi.dll`
- D3D11：`d3d11.dll` + `dxgi.dll`
- D3D12：**空集，强制原生**

### 5.3 安装与所有权

- 目标路径已存在（可见合并文件系统）→ **v1 拒绝覆盖**
- 每个文件：写同目录临时文件 → 校验哈希 → 原子改名
- 安装日志：目标路径、包哈希、安装后哈希、是否由 GameVHD 创建
- 不依赖文件名扫描删除
- dxvk.conf/日志/缓存用独立 GameData 路径，不覆盖游戏已有文件
- 清理前确认 Job 内所有进程/子进程已退出
- 文件被改/锁定/所有权不明 → 保留并进 `cleanup_pending`，下次启动先重试；**不得静默继续原生运行**

### 5.4 配置默认值

生成配置只含必要、稳定、v2.7 确认项。**默认不强制**：descriptorBuffer / relaxedBarriers / maxTessFactor / fake FSE / 固定编译线程数。以上仅作逐游戏 manifest 白名单覆盖项。日志写 GameData 并限大小。

### 5.5 启动流程

1. 读取并验证 manifest、主机 profile、box 状态
2. 获取 box 锁；恢复 `preparing/running/cleaning/cleanup_pending` 遗留事务
3. 选择**实际渲染 EXE**（非盲目用 launcher EXE）
4. 解析位数 + 导入表，合并 manifest API
5. **D3D12-only → 不安装任何 DXVK 文件，清空 DXVK 环境变量，强制原生**
6. **D3D11/D3D12 混合或未知动态 API → 拒绝 DXVK，提供明确原生启动操作**
7. 目标位数执行 Vulkan 1.4 + `VK_KHR_maintenance5` 能力检查
8. 生成 `DxvkPlan`，检查 DLL 冲突并安装差分层文件
9. 创建挂起进程，传 `DXVK_CONFIG` / `DXVK_STATE_CACHE_PATH` / `DXVK_LOG_PATH`（不修改全局环境）
10. 注入 gvhook（默认 Early-Bird APC）；**失败则不 Resume，回滚安装事务**
11. 注入成功 → Resume → 写 `running`
12. Job 完整退出 → `cleaning` → 哈希保护清理（保留 shader cache）→ `clean`

用户关闭 DXVK：先清理自有 DLL/配置，再创建原生进程；清理失败**阻止启动**并展示具体冲突文件。

## 6. 实施顺序（最小闭环）

1. manifest schema + 主机 profile + box 状态迁移
2. 版本化 DXVK 包目录 + 哈希校验 + 许可证
3. `pe.rs` 扩展：普通导入、delay-import、D3D12 硬拒绝
4. `dxvk::resolve_plan` + 安装日志 + 原子复制 + 哈希清理
5. GameData 配置/缓存/日志路径 + 环境补丁
6. 挂起创建 + 注入成功判定 + 失败回滚 + `cleanup_pending`
7. gvhook 确认覆盖层路径优先级、系统/Vulkan 路径豁免
8. x86/x64 capability probe
9. Linux 控制面测试 → Windows 真机验收
10. 最后开放 UI 默认开关；未通过的 API/GPU/游戏保持明确不可用

## 7. 测试矩阵（终点证据）

| 类别 | 覆盖项 | 终点证据 |
|---|---|---|
| Manifest/profile | 默认值、用户覆盖、版本迁移、损坏 JSON | 优先级固定；非法配置不启动 DXVK |
| PE 检测 | x86/x64、普通/delay 导入、大小写、损坏 PE | API 集与 D3D12 判定可重复 |
| DLL 安装 | 空目录、已有原生 DLL、已有 wrapper、包哈希错误 | 不覆盖既有文件；错误原因明确 |
| 清理 | 正常退出、强杀、启动失败、文件被改/锁定 | 只删自有且未变文件；失败进 pending |
| Linux/Wine | 无 Vulkan 设备、无 loader、模拟 probe 失败 | 不复制 DLL；返回可诊断 unsupported |
| AMD RDNA1/2 | 老驱动 / 满足 Vulkan 1.4 驱动 | 明确 native fallback 或 DXVK 日志 |
| 其他 GPU | RDNA3、NVIDIA、Intel | DXVK 日志显示正确 Vulkan 设备 |
| API | D3D8/9/10/10.1/11/12/混合 | 目标 DLL 从差分层加载；D3D12 不加载 DXVK dxgi |
| 位数 | x86/x64、x86 loader/ICD | DLL 位数与 EXE 一致；hook 注入成功 |
| 进程 | Early-Bird、CRT、launcher+child、Job 退出 | 所有实际渲染进程被覆盖或明确拒绝 |
| 缓存/重置 | 首次/二次编译、游戏数据重置 | cache 保留且按 EXE/包隔离；基础盘哈希不变 |
| 图形行为 | 窗口、无边框、独占全屏、HDR、VRR、覆盖层 | 记录 DXVK 日志与用户可见结果；不支持行为不标通过 |

## 8. 未闭合的门（诚实标注）

1. 真实 RDNA1/2 驱动是否提供 Vulkan 1.4 + `VK_KHR_maintenance5`——需建立目标用户驱动版本清单
2. DXVK v2.7 对目标 D3D8 游戏（固定管线）兼容性——真机验收前标不可用
3. 32 位 Vulkan loader/ICD 与第三方覆盖层实际安装率未知
4. v2.7 对 `DXVK_CONFIG` / `DXVK_STATE_CACHE_PATH` / `DXVK_LOG_PATH` 的确切优先级与行为需实机确认
5. 已有 dxgi wrapper / ReShade / Special K / 反作弊的产品策略：v1 默认冲突即拒绝
6. gvhook 对差分层 DLL、GameData 配置、Vulkan 系统路径的 NT 路径归一化不误重定向
7. "游戏重置不删除 `GameData/LauncherState/DXVK`" 持久化契约必须落实
8. DXVK 包许可证、来源哈希、更新签名、回滚流程待发布流程确认

## 9. 协作与评审记录

- 初稿：DeepSeek（Sisyphus），2026-08-08
- 评审：GPT-5.6-SOL MAX（`codex exec --model gpt-5.6-sol`，会话 `019fe184-8679-75a0-b835-2f8d9c26278c`，38K tokens）
- 评审价值：抓出 1 致命 + 4 高危设计错误；D1-D10 全部明确决策；测试矩阵从 5 层扩到 12 类
- 分工模式：用户提需求 → SOL 出决策完备设计 → DeepSeek 执行（见 `docs/2026.8.8_三方模型思维对比_SOL_Luna_DeepSeek.md` §5）

## 10. 参考

- DXVK：`doitsujin/dxvk`（v2.7 锁定目标；v3.0 支持 D3D8）
- DXVK Wiki "Windows"：Windows 使用边界
- 现有代码：`gamevhd-runtime/runtime/src/{pe,run,boxfile,manifest}.rs`（feature/gamevhd-runtime-sandbox @ a9ac2e4）
- 三方对比：`docs/2026.8.8_三方模型思维对比_SOL_Luna_DeepSeek.md`
