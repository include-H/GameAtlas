# GameVHD Runtime 实施计划（v2：卡带式启动器）

> 状态：定案待实施（2026-08-07 架构重设计定案；v1「BAT + diskpart」方案被本版取代，历史见 git）
> 定位基准：`docs/GameAtlas_GameVHD_定位与意义.md`（卡带模型：**VHD = 卡带，启动器 = 卡带槽**）
> 关联：GameAtlas 下载端点（现 `backend/internal/services/windows_launch.go` 的 BAT 渲染将被替换）

## 0. v1 → v2 重设计摘要

| 决策点 | v1 | v2 | 理由 |
|---|---|---|---|
| 分发形态 | 后端渲染 BAT | 通用启动器 EXE + 尾部配置（后端拼装，不编译） | 沙箱/磁盘/生命周期全是 API 级能力，脚本装不下 |
| 磁盘操作 | diskpart（临时脚本 + 文本解析） | virtdisk.dll API（Rust） | diskpart 脆弱在脚本/解析层；API 结构化错误码 |
| 盘符发现 | PowerShell `Get-Disk SimpleMatch` 启发式 | 原生 API 卷枚举 + 显式分配盘符 | 启发式不可靠；启动器是管理员，可直接 `DefineDosDevice` |
| 差分盘位置 | `C:\<文件名>.vhdx` | `%LOCALAPPDATA%\GameAtlas\diff\` | 消掉一个提权源，不污染 C 根 |
| 游戏启动账号 | 用户手动启动（非提权） | 启动器提权上下文内启动（游戏 = 提权） | 自提权子进程不再穿透沙箱与 Job Object |
| exe 发现 | 用户手动找 | 卷内 .exe 扫描 + 首次选择记忆（box.json） | 自动确定启动文件，无需改 DB、无需写 VHD 标记 |
| 生命周期 | BAT `pause` 等按键 | Job Object 自动等待 + 托盘 + 自愈链 | 无人值守闭环 |
| 重写规则 | Documents\*\ 等硬编码 | `%USERPROFILE%` + `%TEMP%` 路径模式 | 简单可扩展，覆盖更全 |
| UI | 控制台文本菜单 | 原生 Win32 状态窗 + 托盘 | 卡带式体验 |
| 杀软 | 无 | 不对抗；README 注明白名单 | 个人场景，对抗成本不划算 |

## 1. 背景与目标

### 1.1 现状（v2 基线）

- VHD 差分盘远程启动已跑通：SMB 连接 → 本地创建差分盘 → diskpart 挂载 → 用户手动从盘符启动游戏 → 卸载清理。
- **缺口**：游戏运行期写宿主 C 盘的状态（Documents、AppData、Saved Games、HKCU）不进 VHD，换机丢失、宿主被污染。
- 已交付资产（W1-W3，部分已提交）：注入协议、首进程/子进程注入、Rust 编排骨架（71 测试）、`reg.c` 注册表重定向、hive 方案；**`file.c` 文件重定向未实现**。

### 1.2 目标：卡带式体验

- 在 GameAtlas 点"开始游戏" → 下载 `《地平线》.exe` → 双击：
  自动挂载只读 SMB → 创建差分盘 → 挂载 → 自动确定盘符 → 自动找到游戏 exe → 沙箱内启动 → 游玩 → 退出自动卸载 → 宿主零痕迹。
- 状态唯一归属 = 差分盘（一个文件 = 完整卡带：本体写覆盖 + 存档 + 配置 + 注册表）。
- 卸载 VHD = 整个游戏没来过；迁移 = 拷走差分盘。

### 1.3 与 Sandboxie 的取舍（保留 v1 定案）

| Sandboxie 能力 | 游戏场景需要？ | 结论 |
|---|---|---|
| 用户态 NT API hook + 路径重写 | ✅ 核心 | 保留（复制其思想，不抄代码） |
| 读穿透 / 写重定向（write-copy） | ✅ 核心 | 保留 |
| 子进程自动注入 | ✅ 需要（游戏会拉起子进程） | 保留 |
| 内核驱动防绕过 | ❌ 游戏不防恶意，不会主动绕过 hook | 裁剪 |
| 服务架构（SbieSvc） | ❌ | 裁剪（hook DLL 内联决策） |
| 跨进程句柄强制隔离 | ❌ | 裁剪 |
| WOW64 双视图 | ⚠️ 32 位游戏存在 | 保留（双位数 hook DLL） |

## 2. 总体架构

```
┌─ GameAtlas 后端（Linux）─────────────────────────────────────┐
│  下载端点：通用启动器 launcher.exe + 游戏配置块（尾部追加）   │
└───────────────────────────┬──────────────────────────────────┘
                            │ 下载《游戏名》.exe，双击
┌───────────────────────────▼──────────────────────────────────┐
│ 启动器（Rust x64，零依赖 + winffi；内嵌 gvhook x64/x86 + 模板 hive）│
│  提权自举（一次 UAC）→ SMB → 建差分 → attach → 定盘符         │
│  → 扫描 exe → 提权启动游戏 + 注入 → Job 等待 → 清理链 → 自愈  │
└──────┬────────────────────────────┬──────────────────────────┘
       │ CreateProcess(CREATE_SUSPENDED) + 远程线程注入
┌──────▼────────────────────────────▼──────────────────────────┐
│ gvhook.dll（C + MinHook，x64/x86 两版）——注入每个游戏进程       │
│  hook NTDLL：文件 API → <盘符>:\GameData 重写 + write-copy      │
│               注册表 API → HKU\GameVHD_<id> 重写 + write-copy   │
│               进程 API → 子进程自动注入                        │
└───────────────────────────────────────────────────────────────┘
```

**VHD 内布局**（随卡带走）：

```
<盘符>:\                    ← 差分 VHD（盘符由启动器显式分配）
├── Game\                   ← 基础盘内容（游戏本体，只读）
├── GameData\               ← 沙箱重定向根（差分盘内）
│   ├── Users\<u>\Documents|AppData|Saved Games|Temp\... ← 文件重定向镜像
│   ├── Registry\user.dat   ← HKCU 隔离 hive
│   └── box.json            ← 沙箱状态 + 记住的游戏 exe（路径一律相对卷根）
```

**宿主侧布局**（启动器自身的状态，非游戏状态）：

```
%LOCALAPPDATA%\GameAtlas\
├── diff\<文件名>.vhdx      ← 差分盘（卡带本体）
├── logs\session-*.log      ← 会话日志（成功清、失败留 3）
└── state.json              ← 会话状态（上次挂载/游戏进程，供自愈对账）
```

## 3. 关键技术定案

### 3.1 启动器形态与配置尾巴

- 启动器是**通用二进制**：与具体游戏无关；游戏参数全部来自尾部配置块。
- 下载端点行为：`launcher.exe` + config JSON + footer（magic + config 长度），启动器启动时从文件尾部自定位解析。单文件、离线可用。
- 配置块内容：`game_id`、`title`、`base_vhd`（UNC）、`diff_name`、SMB 凭据（明文 = 项目公认边界，README 声明）、可选 `exe_hint`、可选规则覆盖（`skip_cache_dirs` 等）。
- 后端不编译任何东西：`runtime_bundle.go` 内嵌通用启动器 + 双架构 hook + 模板 hive，请求时拼装。

### 3.2 磁盘层（替代 diskpart）

- SMB：`WNetAddConnection2`（只读共享，失败重试 ×3，覆盖 NAS 休眠唤醒场景）。
- 差分创建：`CreateVirtualDisk`（参数含 parent = UNC 基础盘路径）。
- 挂载/卸载：`AttachVirtualDisk` / `DetachVirtualDisk`（卸载前先尝试清残留）。
- 盘符：attach 后 `GetVirtualDiskPhysicalPath` → 卷枚举（`FindFirstVolume` + `IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS` 匹配物理盘）→ 取卷 GUID → **`DefineDosDevice` 显式分配盘符**（优先配置指定字母，否则取第一个空闲；记入 `state.json`）。
- 全部结构化错误码：无临时脚本、无文本解析、无 PowerShell。

### 3.3 注入机制（保留 v1 定案）

- 首进程：`CreateProcessW(CREATE_SUSPENDED)` → `VirtualAllocEx` 写参数块（hook DLL 路径 + 重写规则表 + 日志路径）→ `CreateRemoteThread(LoadLibraryW)` → `ResumeThread`。
- 子进程：hook `NtCreateUserProcess` / `NtCreateProcessEx`，同手法递归注入。
- 跨位数：启动器 x64；32 位游戏走 WOW64 re-exec（x86 版 hook 注入）；注入协议参数块带 `_Static_assert` 双架构一致性 + 相对偏移设计（已提交）。
- 参数块含 NT 层 suspend 标志（0x1），保护应用自请求挂起语义（已提交修正）。

### 3.4 文件重定向（v1 定案保留；`file.c` 待实现）

- hook 集：`NtCreateFile` / `NtOpenFile`（含 CreateDisposition 语义）、`NtQueryAttributesFile` / `NtQueryFullAttributesFile`、`NtQueryDirectoryFile`（枚举合并）、`NtSetInformationFile`（改名/删除）、`NtDeleteFile`。
- write-copy 语义：
  - 写 → 一律沙箱路径，父目录自动创建
  - 读 → 沙箱优先，否则宿主穿透
  - 枚举 → 宿主 + 沙箱合并（去重，沙箱优先）
  - 删 → 沙箱内生效；沙箱无 → `STATUS_OBJECT_NAME_NOT_FOUND`（宿主不动，防误删）
  - 打开即重定向：按句柄的后续操作天然落沙箱
- 重写规则（v2 定案）：以 `%USERPROFILE%`（Documents / AppData / Saved Games 等全含）与 `%TEMP%` 为锚的路径模式；直通区 = `C:\Windows`、`C:\Program Files*`、其他盘符、VHD 盘符本身。`skip_cache_dirs` 可选直通 + 退出清理。
- `\\?\` 前缀规范化后按同规则处理。

### 3.5 注册表 hive（v1 定案保留）

- 存放 `GameData\Registry\user.dat`（VHD 内）；模板 hive 随包分发（`RegCreateKeyEx` + `RegSaveKey` 生成一次，首次复制，轮换 `.bak`）。
- 挂载 `RegLoadKey(HKUS, "GameVHD_<game_id>", ...)`；键名含 game_id 防并发冲突。
- 重写：`\REGISTRY\USER\<SID>\Software\X` → `\REGISTRY\USER\GameVHD_<id>\Software\X`；仅匹配当前用户 SID；Wow6432Node 随 Software 子树自然重写；读穿透宿主。
- `NtRenameKey` / `NtNotifyChangeKey` 先直通，标注已知局限（游戏监听注册表变更可能漏）。
- 卸载顺序（关键）：进程树全退 → 重试等句柄释放 → `RegUnLoadKey` → `DetachVirtualDisk`。
- 崩溃恢复：启动先幂等 `RegUnLoadKey`；hive 损坏 → `.bak` 恢复 + 警告。
- **HKLM 直通宿主**（记录警告）。v2 注意：游戏提权后 HKLM 写入从"不可能"变为"可能"——属已知局限，见 §7。

### 3.6 游戏启动账号（v2 定案：提权运行）

- 启动器本身提权（磁盘操作必需）；游戏直接在该上下文内 `CreateProcess` 启动（**不做降权**，推翻 v1 的降权设想）。
- 理由：游戏若自提权（requireAdministrator manifest / 启动器触发 UAC），降权父进程既无法注入也无法挂 Job；提权后整棵进程树都在沙箱与 Job 控制内，无静默穿透。
- 影响：游戏可写 HKLM / ProgramData → 零痕迹断言范围收窄为 `%USERPROFILE%` + `%TEMP%` + HKCU（§7 风险表）。
- 不再需要 `CreateProcessWithTokenW`。

### 3.7 exe 发现（v2 定案：扫描 + 记忆）

- 挂载后枚举卷内全部 `.exe` → 噪声过滤（`unins*`、`setup*`、`install*`、`vcredist*`、`dxsetup`、`redist` 等）→ 排序（深度浅优先、体积大优先）。
- 候选唯一 → 直接启动；多个 → 启动器 UI 候选列表，玩家选一次。
- 选择记入 `box.json`（差分盘内，随卡带走，换机也认）；记忆失效（exe 不存在）→ 回退重扫。
- 不写 VHD 标记、不改数据库；基础盘重做后布局变化由重扫 + 重选兜底。

### 3.8 生命周期与自愈（v2 定案）

- 运行期：Job Object 等待整棵进程树退出（kill-on-close：启动器被强杀时游戏进程树随之终止）。
- 窗口关闭 = 缩托盘；托盘菜单："结束游戏并卸载"、"查看日志"。
- 清理链：`RegUnLoadKey`（重试）→ `DetachVirtualDisk` → 断 SMB → 清凭据 → 日志归档。
- 宿主 `state.json` 对账：上次残留挂载 → 复用或清理；残留 hive → 幂等卸载；box.json 状态回滚 `cleaning → clean`。
- 基础盘换新（parent 失配）：挂载/attach 失败检测 → 提示"旧差分盘与当前基础盘不匹配" → 确认后**删除旧差分盘并重建**（不做孤儿保留；存档随之丢失，提示明确）。
- 日志：`%LOCALAPPDATA%\GameAtlas\logs\`，成功退出清空、失败保留最近 3 份供排错。

### 3.9 状态文件 box.json（v2 更新）

```json
{
  "game_id": "horizon-zero-dawn",
  "exe_relative": "Game\\HorizonZeroDawn.exe",
  "game_data_root": "GameData",
  "registry_hive": "GameData\\Registry\\user.dat",
  "user_profile": "C:\\Users\\Hao",
  "state": "clean"
}
```

- 状态机：`clean → running → cleaning → clean`（启动器视角）。
- 路径一律**相对卷根**（盘符由启动器每会话分配，不持久化绝对盘符）；跨机器按新 `USERPROFILE` 重建映射。

### 3.10 与 GameAtlas 集成

- 构建链：`scripts/build-runtime.sh` 交叉编译（启动器 x64 + hook x64/x86 + 模板 hive）→ 生成 `runtime_bundle.go`（base64）→ 编译主后端。
- 后端：新下载端点（GameAtlas 点"开始游戏"）→ 拼装 `《title》.exe` 返回；**删除** `windows_launch.go` 的 BAT 渲染与旧端点。
- 无 DB schema 变更；游戏记录不加"启动文件"字段（扫描 + 记忆替代）。
- README（仓库根 + `gamevhd-runtime/`）注明：杀软/Defender 需将 GameAtlas 启动器加入白名单（不对抗）。

## 4. 阶段划分（v2 顺序）

> 阶段 0-1 已完成（已提交）；阶段 2-3 部分完成（`file.c` 未实现、reg 验收未跑）；阶段 4-6 为 v2 新增。

### 阶段 0：工程骨架 ✓（已提交）

Rust workspace（x64/x86 target）+ C hook 工程（MinHook vendored）+ CLI + 断言框架 + 交叉编译验证。

### 阶段 1：注入与 hook 骨架 ✓（已提交）

注入协议（参数块/规则表/标记串，双架构 `_Static_assert`）；首进程三件套注入 + 子进程自动注入（`gvhook_init` 导出验证）；test-app 注入自检契约。

### 阶段 2：文件系统重定向（进行中，`file.c` 未实现）

hook 全集 + write-copy + 父目录自动创建 + 枚举合并 + 删除/改名语义；规则表生成（`%USERPROFILE%` / `%TEMP%` 模式）；test-app file_mode + `assert_fs.ps1`。验收：7 操作 × 3 目录 × 双位数 + 边界用例 + 宿主零残留。

### 阶段 3：注册表重定向（`reg.c`/hive 已实现，验收未跑）

模板 hive + 挂载/卸载 + HKCU 重写/读穿透/枚举合并；test-app reg_mode + `assert_reg.ps1`。验收：6 操作 × 双位数 + 持久化 + 宿主零痕迹 + 读穿透。

### 阶段 4：磁盘层与盘符发现（新）

`runtime/disk.rs`：`WNetAddConnection2` / `CreateVirtualDisk` / `AttachVirtualDisk` / `DetachVirtualDisk` / 卷枚举 / `DefineDosDevice` 显式盘符。
验收：`assert_mount.ps1` 改走 API 全流程；错误路径单测（parent 失配、重复挂载、SMB 重试）。

### 阶段 5：启动器闭环（新）

- `manifest.rs`：尾部配置自定位/解析；后端拼装端点 + 移除 BAT。
- `scan.rs`：exe 枚举/噪声过滤/排序 + box.json 记忆读写。
- `run` 主流程：磁盘 → 扫描/选择 → 注入启动 → Job 等待 → 清理链 → 自愈对账（state.json / hive .bak / parent 失配删除重建）。
- UI：原生 Win32 深色状态窗（引导步骤 + 候选选择）+ 托盘（游玩中："结束游戏并卸载"）。

### 阶段 6：真机验收（地平线：零之曙光 GOG 版）

首启（建差分 + 选 exe）→ 二启零点击（存档延续）→ 迁移模拟（拷贝差分盘到干净机器）→ 强杀/断电自愈 → 基础盘换新（parent 失配处理）→ 10 次循环无污染 → 宿主零痕迹双查（`%USERPROFILE%` + `%TEMP%` + HKCU）。

## 5. 测试总纲

### 5.1 测试工具（更新）

| 工具 | 说明 |
|---|---|
| launcher.exe（x64） | 自检模式 `--selftest`（footer 解析 / 规则生成 / PE 探测）+ 正常引导流程 |
| test-app.exe（x64/x86） | 注入 / 文件 / 注册表 / 子进程四模式（已有） |
| test/assert_*.ps1 | 每阶段断言（`assert_mount` 改 virtdisk API 版；新增 `assert_scan` / `assert_launcher`） |

### 5.2 阶段测试矩阵

| 阶段 | 测试内容 | 通过标准 |
|---|---|---|
| 0-1 | 挂载 / 注入自检 | 已过（双位数标记齐全） |
| 2 | 文件 7 操作 × 3 目录 × 双位数 + 边界 | 沙箱落点 ✓ 宿主零残留 ✓ 内容一致 ✓ |
| 3 | 注册表 6 操作 × 双位数 + 读穿透 + 持久化 | hive 落点 ✓ 宿主零残留 ✓ 重启可读 ✓ |
| 4 | SMB / 差分 / attach / 盘符 / 错误路径 | API 全流程 ✓ 盘符显式可预期 ✓ 错误码路径 ✓ |
| 5 | 扫描 / 记忆 / 生命周期 / 自愈 | 首启选择 ✓ 二启零点击 ✓ 强杀自愈 ✓ 10 循环 ✓ |
| 6 | 真机全流程 + 迁移 | 存档落 VHD ✓ 零痕迹 ✓ 迁移完整 ✓ |

### 5.3 环境要求（不变）

Linux 交叉编译 + Windows 11 x64 真机；管理员权限；GOG 版地平线；测试 VHD（x64/x86 双 exe）。

## 6. 目录结构（v2）

```
gamevhd-runtime/
├── runtime/                    ← Rust（启动器 = 单二进制，x64）
│   └── src/
│       ├── main.rs             ← 引导/提权/命令分发 + --selftest
│       ├── manifest.rs         ← 尾部配置解析（新）
│       ├── disk.rs             ← SMB/virtdisk/盘符（新，替代 diskpart）
│       ├── scan.rs             ← exe 扫描/过滤/排序 + box.json 记忆（新）
│       ├── inject.rs           ← CREATE_SUSPENDED 三件套注入（已有）
│       ├── rules.rs            ← 重写规则生成/持久化（已有）
│       ├── hive.rs             ← RegLoadKey/RegUnLoadKey/模板（已有）
│       ├── process.rs          ← Job Object 进程树等待（已有）
│       ├── state.rs            ← 宿主 state.json 自愈对账（新）
│       └── ui.rs               ← Win32 状态窗 + 托盘（新）
├── hook/                       ← C gvhook（x64/x86；init/proc/reg 已有，file 待实现）
│   ├── src/{file.c, reg.c, proc.c, init.c, rules.h, hook_common.h}
│   └── deps/minhook/
├── scripts/
│   ├── build-runtime.sh        ← 交叉编译 + 打包
│   └── embed-bundle.go.tmpl    ← 生成 runtime_bundle.go（base64）
└── test/
    ├── test-app/               ← C 测试程序（x64/x86）
    └── assert_*.ps1

backend/
├── internal/services/launcher.go       ← 新：拼装《title》.exe（替换 windows_launch.go）
├── internal/services/windows_launch.go ← 删除（BAT 渲染退役）
└── internal/.../runtime_bundle.go      ← 构建产物（构建时生成）
```

## 7. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 游戏提权后写 HKLM / ProgramData | 宿主残留，违反零痕迹 | 已知局限：断言范围 = %USERPROFILE% + %TEMP% + HKCU；hook 记录警告；遇真写再补规则 |
| 反作弊/反调试游戏检测 hook | 无法沙箱化 | GOG 范围无反作弊；异常时降级提示（不承诺解决） |
| 32 位游戏注入 | 兼容性 | 双位数 hook + WOW64 re-exec 全程回归 |
| 目录枚举合并 | 游戏读不到存档 | 测试矩阵重点覆盖 + 去重规则单测 |
| `\\?\` / 设备路径 / 共享内存映射 | 绕过重写 | 规范化处理 + 日志留证 |
| 杀软 / SmartScreen 拦截启动器与 hook DLL | 无法启动 | 不对抗；README 注明白名单；签名留作后续 |
| hive 损坏（断电时写入） | 设置丢失 | `.bak` 轮换 + 恢复警告 |
| 基础盘换新（parent 失配） | 差分盘失效 | 检测 → 确认删除重建（用户已确认不留孤儿） |
| 强杀 / 断电残留挂载 | 下次启动异常 | state.json 对账 + 幂等清理（自愈链） |
| SMB 断连（NAS 休眠） | 挂载失败 | 重试 ×3 + 明确错误文案 |
| 多沙箱并发（同机双游戏） | hive 键名冲突 | 键名含 game_id；状态文件独立；单写者约束保持 |
| UWP / 微软商店游戏 | 不支持 | 文档明示范围 |

## 8. 实施顺序与依赖

```
阶段 2/3（遗留，可并行）──┐
                          ├──→ 阶段 5（启动器闭环）──→ 阶段 6（真机）
阶段 4（磁盘层，新）──────┘
```

- 阶段 4 不依赖 2/3，先行（启动器骨架的地基）。
- 每阶段结束跑对应 `assert_*.ps1` + 真机抽查才算完成。
- 构建链从阶段 4 起维护交叉编译可用性（CI 可先加编译冒烟）。

## 9. 明确不做（本迭代范围外）

- 内核驱动 / minifilter、内核注册表回调（不变）
- HKLM 注册表重定向（v1 不做；游戏提权后按需补规则，机制留 hook 层）
- Registry Overlay 读合并精细语义（hive 内删除标记等，MVP 简化）
- VHD 内标记文件 / DB 启动文件字段（exe 发现走扫描 + 记忆）
- 孤儿差分盘保留（parent 失配直接确认删除重建）
- BAT 兼容（旧端点直接替换）
- 杀软对抗 / 代码签名（README 白名单说明）
- 快照 / 版本管理 / Mod 管理 GUI
- ETW 监控（如将来做"未知路径发现"再引入）
