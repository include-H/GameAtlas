# GameVHD Runtime 实施计划（B 方案：纯用户态轻量沙箱）

> 状态：已定案，待实施
> 日期：2026-08-07
> 关联：GameAtlas VHD 远程启动（`backend/internal/services/windows_launch.go`）

## 1. 背景与目标

### 1.1 现状

GameAtlas 已有 VHD 差分盘远程启动：BAT 脚本连接 SMB → 本地创建差分盘 → `diskpart` 挂载 → 用户手动从盘符启动游戏 → 卸载清理。**缺口**：游戏运行时会往宿主 C 盘写配置/存档/注册表（Documents、AppData、Saved Games），这些状态**不进 VHD**，换机器丢失、宿主被污染。

### 1.2 目标

为游戏进程提供**纯用户态轻量沙箱**：

- 游戏所有文件写入 → 重定向到 VHD 内目录（`E:\GameData\...`）
- 游戏所有 HKCU 注册表写入 → 重定向到 VHD 内的 hive 文件
- **无驱动、无服务、无安装、单目录部署**（对比 Sandboxie：裁剪其安全对抗模型，仅保留重定向能力）
- 宿主在游戏不运行时零痕迹；状态全部收敛在 VHD 差分盘里

### 1.3 与 Sandboxie 的取舍（为什么可以裁剪）

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
┌─ GameAtlas 后端（Linux）────────────────────────────────┐
│  windows_launch.go 渲染 BAT（内嵌 base64 产物包）        │
└──────────────────────────┬───────────────────────────────┘
                           │ 下载 .bat，Windows 客户端双击
┌──────────────────────────▼───────────────────────────────┐
│ BAT（提权 → SMB 连接 → diskpart 挂载差分 VHD → 调用 runtime）│
└──────────────────────────┬───────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────┐
│ gamevhd-runtime.exe（编排，Rust，x64/x86 两版）            │
│  run：扫描 exe → 确认 → 挂载 hive → 注入启动游戏          │
│        → 等进程树退出 → 卸载 hive → 状态落盘              │
└──────┬───────────────────────────┬───────────────────────┘
       │ CreateProcess(CREATE_SUSPENDED) + 远程线程注入
┌──────▼───────────────────────────▼───────────────────────┐
│ gvhook.dll（C + MinHook，x64/x86 两版）——注入每个游戏进程   │
│  hook NTDLL：文件 API → E:\GameData 重写 + write-copy      │
│               注册表 API → HKU\GameVHD 重写 + write-copy   │
│               进程 API → 子进程自动注入                    │
└───────────────────────────────────────────────────────────┘
```

**运行数据布局（VHD 内）**：

```
E:\                        ← 差分 VHD 挂载盘符
├── Game\                  ← 游戏本体（基础盘只读内容）
├── GameData\              ← 沙箱重定向根（差分盘内，随 VHD 走）
│   ├── Users\Hao\Documents\...    ← 文件重定向镜像（相对 USERPROFILE 结构）
│   ├── Users\Hao\AppData\...      ← 同上
│   ├── Users\Hao\Saved Games\...  ← 同上
│   ├── Registry\user.dat          ← HKCU 隔离 hive
│   └── box.json                   ← 沙箱状态文件
```

## 3. 关键技术定案

### 3.1 注入机制

- **首进程**：编排进程 `CreateProcessW(CREATE_SUSPENDED)` → `VirtualAllocEx` 写入参数块（hook DLL 路径 + 重写规则表）→ `CreateRemoteThread(LoadLibraryW)` → `ResumeThread`。
- **子进程**：gvhook 内 hook `NtCreateUserProcess` / `NtCreateProcessEx`，子进程创建时以同样手法注入 → **进程树全覆盖**（游戏 → ModManager → CrashReporter 全部进沙箱）。
- **32/64 位**：跨位数进程无法 `CreateRemoteThread` 注入。编排发布 x64/x86 两版，**先探测游戏 exe 位数（PE header），用同位数编排启动**；子进程由同位数 hook 自然继承（32 位游戏创建的必然是 32 位子进程）。
- **重写规则传递**：参数块含 `GameDataRoot`（如 `E:\GameData`）、`UserProfile` 映射表，hook DLL 初始化时读取。
- hook DLL 随进程退出自动卸载，无系统残留。

### 3.2 hook 函数清单（NTDLL）

**文件**（`hook/file.c`）：

| 函数 | 处理 |
|---|---|
| `NtCreateFile` / `NtOpenFile` | 路径重写 + write-copy 决策 + 父目录自动创建 |
| `NtQueryAttributesFile` / `NtQueryFullAttributesFile` | 读穿透：沙箱无 → 查宿主 |
| `NtQueryDirectoryFile` | **目录枚举合并**（宿主 + 沙箱合并返回，去重） |
| `NtDeleteFile` | 删除重定向到沙箱路径 |
| `NtSetInformationFile` | 按句柄操作（句柄已指向沙箱文件）；`FileRenameInformation` 新路径需重写 |

**注册表**（`hook/reg.c`）：

| 函数 | 处理 |
|---|---|
| `NtOpenKey(Ex)` / `NtCreateKey(Ex)`（含 Transacted 变体） | HKCU → `HKU\GameVHD_<id>` 路径重写 + 读穿透 |
| `NtSetValueKey` / `NtDeleteValueKey` | 写入重定向 |
| `NtDeleteKey(Ex)` | 删除重定向（沙箱无 → 返回 NOT_FOUND，不动宿主） |
| `NtQueryValueKey` / `NtQueryMultipleValueKey` / `NtEnumerateValueKey` | 读穿透 + 枚举合并 |
| `NtQueryKey` / `NtEnumerateKey` | 读穿透 + 枚举合并 |
| `NtRenameKey` / `NtNotifyChangeKey` | 边界：先直通宿主，标注已知局限 |

**进程**（`hook/proc.c`）：`NtCreateUserProcess` / `NtCreateProcessEx` → 子进程注入。

### 3.3 路径重写规则

- **重写区**（前缀匹配，相对 `%USERPROFILE%` 保存相对结构，跨机器可重建）：
  - `%USERPROFILE%\Documents`
  - `%USERPROFILE%\AppData`（Local / LocalLow / Roaming）
  - `%USERPROFILE%\Saved Games`
- **映射**：`C:\Users\Hao\Documents\X` → `E:\GameData\Users\Hao\Documents\X`
- **直通区**：`C:\Windows`、`C:\Program Files*`、其他盘符、VHD 盘符本身（游戏本体在 VHD 里，本来就在差分盘上）——不重写。
- `\\?\` 长路径前缀：规范化后按上述规则处理。
- 规则表由编排生成（解析当前 `USERPROFILE`），经注入参数块传给 hook DLL；`box.json` 持久化，跨机器按新 `USERPROFILE` 重建。

### 3.4 write-copy 语义（MVP 定案）

- **写**（`DesiredAccess` 含 `FILE_WRITE_DATA`/`GENERIC_WRITE`，或 `CreateDisposition` 非 `FILE_OPEN`）→ 一律沙箱路径；**父目录不存在则自动创建**。
- **读**（只读打开）→ 沙箱路径存在则沙箱，否则宿主（穿透）。
- **目录枚举** → 宿主目录句柄 + 沙箱对应目录合并返回（去重；沙箱优先覆盖同名条目）。
- **删除** → 重定向到沙箱；沙箱无此对象 → `STATUS_OBJECT_NAME_NOT_FOUND`（宿主文件不动，防误删）。升级选项（Sandboxie DELETE_MARK 语义）留待后续。
- **打开即重定向，句柄后续操作免重写**：重写只发生在"按路径打开/查询"层，打开的句柄已指向沙箱文件，`NtSetInformationFile` 等按句柄操作天然落在沙箱。

### 3.5 注册表 hive

- **存放**：`E:\GameData\Registry\user.dat`（VHD 内，随 VHD 走）。
- **模板 hive**：`RegLoadKey` 要求目标文件已是有效 hive。随包分发空模板（工具用 `RegCreateKeyEx` + `RegSaveKey` 生成一次，几 KB），首次运行复制为 `user.dat`，后续轮换 `.bak`。
- **挂载**：`RegLoadKey(HKUS, "GameVHD_<game_id>", E:\GameData\Registry\user.dat)`（键名含 game_id 防多沙箱并发冲突）。
- **重写**：native 路径 `\REGISTRY\USER\<当前 SID>\Software\X` → `\REGISTRY\USER\GameVHD_<game_id>\Software\X`；仅匹配当前用户 SID；`\REGISTRY\MACHINE`（HKLM）**直通宿主**（GOG 绿色版不写 HKLM；真写了记录警告不隔离）。32 位游戏的 `Wow6432Node` 随 `Software` 子树自然重写。
- **读穿透**：hive 无键 → 读宿主 HKCU（hook 内 fallback）。
- **卸载顺序（关键）**：进程树全退 → 重试等待句柄释放 → `RegUnLoadKey` → 卸载 VHD。hive 未卸载前 `diskpart detach` 会失败。
- **崩溃恢复**：启动时先尝试 `RegUnLoadKey`（幂等，清上次残留挂载）；`RegLoadKey` 失败（hive 损坏）→ 从 `.bak` 恢复并警告。

### 3.6 状态文件 `box.json`（存 VHD：`E:\GameData\box.json`）

```json
{
  "game_id": "horizon-zero-dawn",
  "exe_relative": "Game\\HorizonZeroDawn.exe",
  "game_data_root": "E:\\GameData",
  "user_profile": "C:\\Users\\Hao",
  "registry_hive": "GameData\\Registry\\user.dat",
  "state": "clean"
}
```

- 状态机：`clean → running → cleaning → clean`。
- 用途：二启免确认 exe；跨机器迁移按新 `USERPROFILE` 重建映射；崩溃残留自检依据。

### 3.7 与 GameAtlas 集成

- **构建链**：`scripts/build-runtime.sh` 交叉编译（Rust x64/x86 + C hook x64/x86 + 模板 hive）→ 打包 → 生成 `runtime_bundle.go`（base64 常量，`//go:embed` 或 const 注入）→ 编译主后端。
- **BAT 渲染**（`windows_launch.go` 改造）：现有挂载逻辑后追加——解码 base64 产物到 `%TEMP%\gamevhd-runtime\` → 探测 exe 位数 → 调对应 `gamevhd-runtime.exe run --box ...` → 结束后 `cleanup` → 现有卸载逻辑。
- **产物清单**：`gamevhd-runtime-x64.exe`、`gamevhd-runtime-x86.exe`、`gvhook-x64.dll`、`gvhook-x86.dll`、`hive-template.dat`（合计预计 < 2MB，base64 后 BAT 约 2.7MB）。

## 4. 阶段划分

> 每阶段独立可验收；验收标准即"完成"定义。阶段 0-1 完成后即可并行开发 2/3。

### 阶段 0：工程骨架与编排 CLI

**目标**：工程立起来，挂载/卸载/扫描链路跑通（不动现有 BAT 主流程，仅验证编排能力）。

**工作内容**：
1. 目录结构初始化（见 §6）；Rust workspace（x64/x86 target）+ C hook 工程（MinHook 依赖）
2. 编排 CLI：`mount <vhd>`（调用 diskpart 逻辑）、`unmount <vhd>`、`scan <drive>`（扫描 exe、过滤 `unins*.exe`/`install*.exe`、显示候选清单）、`probe <exe>`（PE 位数检测）
3. 日志框架（结构化输出到控制台 + 文件）

**交付物**：可交叉编译的三平台产物；CLI 四命令可用。

**验收标准**：
- Linux 交叉编译产物在 Windows 11 x64 运行正常
- `mount` 挂载测试 VHD 成功（`Get-Volume` 可见盘符）
- `scan` 正确列出盘符下 exe 并标注位数
- `unmount` 干净卸载（无残留挂载）

**测试方法**：
- 真机手动：造一个含 2 个 exe（x64/x86 各一）的测试 VHD，跑通 mount → scan → unmount
- PowerShell 断言脚本 `test/assert_mount.ps1`：断言卷存在/消失、盘符正确

### 阶段 1：注入与 hook 骨架

**目标**：游戏进程及其全部子进程被注入，hook 生效（只验证拦截，不做重定向）。

**工作内容**：
1. gvhook.dll（x64/x86）：MinHook 初始化、`hook_init(hook_dll_path, rules, log_path)` 参数解析、注入自检日志
2. 编排注入：`CreateProcessW(CREATE_SUSPENDED)` 三件套注入；位数探测后选择对应编排
3. hook `NtCreateUserProcess`：子进程自动注入（递归）
4. hook `NtQueryAttributesFile`：仅打日志（验证拦截链路，不重写）

**交付物**：gvhook x64/x86 + test-app（x64/x86 测试程序，见 §5）。

**验收标准**：
- test-app 启动后日志显示"注入成功 + 规则表正确"
- test-app 创建子进程 → 子进程日志同样显示注入成功（进程树全覆盖）
- x86 test-app 走 x86 编排 + x86 hook，无位数错乱
- 日志能记录 `NtQueryAttributesFile` 调用（证明 hook 生效）

**测试方法**：
- test-app 内置自检：启动即输出 `HOOK_DLL_PRESENT` 标记 + 创建子进程
- 自动化：`test/assert_inject.ps1` 启动 test-app，捕获输出断言标记与子进程标记

### 阶段 2：文件系统重定向

**目标**：游戏文件写入全部落 `E:\GameData`，读回一致，宿主零残留。

**工作内容**：
1. §3.2 文件 API hook 全集实现：路径重写 + write-copy + 父目录自动创建 + 目录枚举合并 + 删除/改名语义
2. 重写规则表生成（编排解析 USERPROFILE）+ 注入传递
3. test-app 文件测试模式扩展（见 §5 测试矩阵）

**交付物**：文件重定向完整实现 + test-app 文件模式 + 断言脚本。

**验收标准**（test-app 全套操作通过 + PowerShell 断言）：
- 新建/覆盖/追加写 `Documents\Test\*`、`AppData\Local\Test\*`、`Saved Games\Test\*` → 文件落在 `E:\GameData\Users\<u>\...` 对应位置
- 同路径读回内容一致；列目录能看到沙箱内文件（枚举合并）
- 删除/改名行为正确（沙箱内生效；宿主文件不受影响）
- 只读穿透：读 `C:\Windows\...`、`C:\Program Files\...` 不受影响
- **宿主对应目录零新增文件**（断言脚本对比操作前后目录树）
- x86 test-app 同样全过

**测试方法**：
- test-app 文件模式：`test-app file --action <新建|覆盖|追加|读|列目录|删除|改名> --target <路径>`
- `test/assert_fs.ps1`：操作后断言（a）文件在沙箱内存在且内容一致（b）宿主无新增（c）目录枚举结果正确
- 边界用例：深层目录自动创建、`\\?\` 长路径、只读打开宿主文件、不存在的父目录写

### 阶段 3：注册表重定向

**目标**：HKCU 写入落 VHD hive，重启可读回，宿主注册表零痕迹。

**工作内容**：
1. 模板 hive 生成工具 + 首次复制/轮换逻辑
2. `RegLoadKey` 挂载/卸载（含句柄等待重试、崩溃残留清理）
3. §3.2 注册表 API hook 全集：HKCU → `HKU\GameVHD_<id>` 重写 + 读穿透 + 枚举合并
4. test-app 注册表测试模式

**验收标准**：
- 写 `HKCU\Software\TestGame` 全套（建键/写值/覆盖值/枚举/删值/删键）→ 数据落在 hive
- 重启 runtime（卸载 hive → 重新挂载）后读回一致（游戏设置持久化）
- **宿主 `HKCU\Software\TestGame` 不存在**（reg query 断言）
- 读穿透：读宿主已有键（如 `HKCU\Software\Microsoft\...`）正常
- 32 位 test-app 写入 `Wow6432Node` 路径随重写正确落 hive
- HKLM 写入直通宿主并产生警告日志

**测试方法**：
- test-app 注册表模式：`test-app reg --action <建键|写值|读值|枚举|删值|删键> --path <HKCU\...>`
- `test/assert_reg.ps1`：断言（a）hive 文件存在且 `RegLoadKey` 后能查到数据（b）宿主键不存在（c）读穿透键可读
- 重启持久化测试：run 两次，第二次读到第一次写入

### 阶段 4：生命周期与崩溃恢复

**目标**：完整闭环 + 异常场景自愈。

**工作内容**：
1. `run` 命令完整流程：挂载 hive → 注入启动 → 进程树等待（root + 全部子进程退出）→ 清理（hive 卸载重试）→ 状态落盘
2. `cleanup` 命令：崩溃残留自检（残留 hive 挂载 → 卸载；残留 VHD 挂载 → 卸载；box.json 状态回滚）
3. 退出码与错误路径文档化

**验收标准**：
- 正常流程：启动 → 游戏运行 → 退出 → 全部清理（hive 卸载、无残留注入、VHD 可正常 detach）
- 强杀游戏进程 → 再次运行 `run`：自动清残留、正常启动
- 脚本被强杀（断电模拟）→ 再次运行：自愈
- 连续 10 次运行-退出循环：状态无累积污染（box.json 回到 clean、宿主零痕迹）

**测试方法**：
- `test/assert_lifecycle.ps1`：完整生命周期断言（挂载点、hive 状态、宿主痕迹）
- 强杀：`Stop-Process` 游戏进程后重跑；断电：VM 快照回滚模拟（可选）
- 循环测试：10 次循环断言

### 阶段 5：GameAtlas 集成与真机验证

**目标**：从 GameAtlas 网页下载 BAT 全自动跑通真实游戏，状态全在 VHD。

**工作内容**：
1. `scripts/build-runtime.sh`：完整构建链（交叉编译 → 打包 → 生成 `runtime_bundle.go`）+ CI 步骤
2. `windows_launch.go` 渲染改造：内嵌 base64 产物解码、位数探测、调 runtime、清理
3. 真机验证（地平线：零之曙光 GOG 版）

**验收标准**：
- 下载 BAT → 双击 → 自动：挂载 → 扫描确认 exe（首次）→ 沙箱启动 → 游戏可玩
- 游戏存档/画质设置写 Documents/AppData → 全部落 `E:\GameData`（差分盘）
- 游戏退出 → 宿主文件/注册表零痕迹（双查）
- 二次启动：存档延续（读回 VHD 内数据）
- 迁移模拟：复制差分 VHD + 基础盘到干净机器 → 存档/设置完整

**测试方法**：
- 真机全流程手动 + 断言脚本
- 迁移模拟：注册表无法复制（hive 在 VHD 内，随 VHD 走——验证新机器 `USERPROFILE` 重建映射正确）

## 5. 测试总纲

### 5.1 测试工具

| 工具 | 说明 |
|---|---|
| `test-app.exe`（x64/x86） | 测试程序：注入自检模式 + 文件模式 + 注册表模式 + 子进程模式；操作日志输出到 stdout 与文件 |
| `test/assert_*.ps1` | 每阶段断言脚本（断言沙箱落点、宿主零痕迹、数据一致性） |
| 测试 VHD | 含 x64/x86 测试 exe + 一份测试游戏的迷你 VHD |

### 5.2 阶段测试矩阵

| 阶段 | 测试内容 | 通过标准 |
|---|---|---|
| 0 | 挂载/卸载/扫描 | 卷出现/消失、exe 清单正确 |
| 1 | 注入自检（x64/x86、父子进程） | 双位数标记齐全 |
| 2 | 文件 7 操作 × 3 目录 × x64/x86 + 边界用例 | 沙箱落点 ✓ 宿主零残留 ✓ 内容一致 ✓ |
| 3 | 注册表 6 操作 × x64/x86 + 读穿透 + 持久化 | hive 落点 ✓ 宿主零残留 ✓ 重启可读 ✓ |
| 4 | 生命周期 + 强杀 + 断电 + 10 次循环 | 全清理 ✓ 自愈 ✓ 无污染 ✓ |
| 5 | 真实游戏全流程 + 迁移模拟 | 存档/设置落 VHD ✓ 零痕迹 ✓ 迁移完整 ✓ |

### 5.3 环境要求

- 开发：Linux（交叉编译）+ Windows 11 x64（真机验证）
- 真机验证需要：管理员权限、一份 GOG 版地平线、测试 VHD

## 6. 目录结构

```
gamevhd-runtime/                  ← 新增（仓库根下）
├── runtime/                      ← Rust 编排（x64/x86）
│   ├── Cargo.toml（x86_64-pc-windows-gnu / i686-pc-windows-gnu）
│   └── src/
│       ├── main.rs               ← CLI：mount/unmount/scan/probe/run/cleanup
│       ├── inject.rs             ← CREATE_SUSPENDED 三件套注入
│       ├── rules.rs              ← 重写规则表生成/持久化
│       ├── hive.rs               ← RegLoadKey/RegUnLoadKey/模板复制
│       └── process.rs            ← 进程树等待/枚举
├── hook/                         ← C gvhook（x64/x86）
│   ├── src/{file.c, reg.c, proc.c, init.c, rules.h, hook_common.h}
│   └── deps/minhook/
├── scripts/
│   ├── build-runtime.sh          ← 交叉编译 + 打包
│   └── embed-bundle.go.tmpl      ← 生成 runtime_bundle.go（base64）
└── test/
    ├── test-app/                 ← C 测试程序（x64/x86）
    └── assert_*.ps1

backend/                          ← 改动
├── internal/services/windows_launch.go   ← BAT 渲染追加 runtime 段
└── internal/.../runtime_bundle.go        ← 构建产物（gitignore 或提交？——随发行构建生成）
```

## 7. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 反作弊/反调试游戏检测到 hook | 无法沙箱化 | GOG 场景无反作弊；检测到异常时降级提示（记录，不承诺解决） |
| 32 位游戏注入复杂度 | 兼容性 | 阶段 1 即双位数覆盖，test-app x86 全程回归 |
| 目录枚举合并正确性 | 游戏读不到自己的存档 | 测试矩阵重点覆盖；枚举合并去重规则单测 |
| 游戏使用 `\\?\`/设备路径/共享内存映射 | 绕过重写 | 路径规范化处理；异常场景记录日志便于排查 |
| UWP/微软商店游戏 | 不支持 | 文档明示范围 |
| `NtNotifyChangeKey` 直通局限 | 游戏监听注册表变更可能漏 | 先直通，标注已知局限，实际游戏遇到再补 |
| 杀软误报 hook DLL | 用户侧问题 | 文档说明；签名可选后续 |
| hive 损坏（断电时写入） | 设置丢失 | `.bak` 轮换 + 恢复警告 |
| 多沙箱并发（同机跑两个游戏） | hive 键名冲突 | 键名含 game_id；状态文件独立 |

## 8. 实施顺序与依赖

```
阶段 0 ──→ 阶段 1 ──┬──→ 阶段 2 ──→ 阶段 4 ──→ 阶段 5
                    └──→ 阶段 3 ──↗
```

- 阶段 2 与阶段 3 可并行（hook 框架已就绪，文件/注册表互不依赖）
- 每阶段结束跑对应 `assert_*.ps1` + 真机抽查后才算完成
- 阶段 5 的构建链需要阶段 0 起就维护交叉编译可用性（CI 可先加编译冒烟）

## 9. 明确不做（本迭代范围外）

- 内核驱动/minifilter、内核注册表回调
- Registry Overlay 读合并的精细语义（hive 内删除标记等）——当前为 MVP 简化语义
- 快照/版本管理/Mod 管理 GUI
- ETW 监控（B 方案不再需要学习流程；如将来做"未知路径发现"再引入）
