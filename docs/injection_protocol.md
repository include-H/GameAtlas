# GameVHD Runtime 注入协议（Injection Protocol）——单一事实源

> 状态：定案（single source of truth）
> 协议版本：`GVHD_PROTOCOL_VERSION = 1`
> 日期：2026-08-07
> 权威性：**本文档是 Rust 编排（`gamevhd-runtime.exe`）与 C 钩子（`gvhook.dll`）之间的唯一协议权威**。任何一侧的实现不符本文档即视为缺陷。
> C 参考实现：`gamevhd-runtime/hook/src/hook_common.h`、`hook/src/rules.h`（与本文档逐字节同步，含编译期 `_Static_assert` 布局校验）。
> 关联：`docs/GameVHD_Runtime_实施计划.md` §3.1（注入机制）、§3.2（hook 清单）、§3.3（路径重写规则）、§3.4（write-copy）、§6（目录结构）。

---

## 0. 本文件的地位与变更规则

- **单一事实源**：编排与钩子各自独立实现本协议（Rust 用 `#[repr(C)]` + `u32`/`u16`，C 用本仓库头文件）。协议 ABI 不匹配是两仓库之间最昂贵的故障，因此：
  1. 协议任何字段、偏移、大小、语义的变更，**必须先改本文档，再同步 C 头文件**，并 bump `GVHD_PROTOCOL_VERSION`。
  2. 头文件末尾的 `_Static_assert` 在 x86/x64 双架构下于**编译期**强制字段偏移与结构大小与本文档一致——布局漂移会直接编译失败。
  3. 新增字段只允许追加在 `reserved` 区或结构末尾，不得改动既有字段偏移（前向兼容策略）。
- 本文件不约束 hook 内部实现（各 `.c` 文件归属波次见 §9）。

---

## 1. 设计决策记录（ADR）

| # | 决策 | 理由 | 备选（被否决） |
|---|---|---|---|
| 1 | 参数块内**不含任何指针**；规则表用 **u32 相对字节偏移**（`rule_table_offset`）而非指针 | ① 结构体零指针成员 → x86/x64 布局**完全一致**，一套偏移表通吃两种位数，从根上消灭 ABI 错位；② 子进程注入时可把「参数块 + 规则表」**整块原样 memcpy** 进子进程，无需翻译指针（父进程指针在子进程地址空间无效）；③ 编排侧 `WriteProcessMemory` 单次连续写入 | 裸指针 `rule_table_ptr`：x86/x64 偏移不同、子进程注入需逐字段重写指针地址、易错 |
| 2 | 路径类字段（hook DLL 路径、GameDataRoot、USERPROFILE、日志路径、hive 路径、game_id）用**固定宽度 WCHAR 数组** | 注入保持简单：单次 `VirtualAllocEx` + 单次 `WriteProcessMemory` 即完成全部参数传递，无需为每个字符串单独分配/写入/记录指针 | 字符串指针 + 远程字符串分配：多块分配、多次写入、指针失效风险，收益（省几十字节）不值得 |
| 3 | **自然对齐，不用 `#pragma pack(1)`** | 成员对齐不超过 4（`uint32_t`=4、`wchar_t`=2），所有偏移天然对齐、编译器零 padding，自然对齐与 packed 结果完全相同；保留编译器最佳实践 | `#pragma pack(1)`：无任何收益，且掩盖对齐问题 |
| 4 | **双远程线程注入**：①`LoadLibraryW(hook_dll_path)` → ②`gvhook_init(param_block)` | ① `DllMain` 在 loader lock 下执行，安装 MinHook / 开文件 / 初始化存在死锁与重入风险，且**拿不到 `lpParameter`**；② 拆成独立线程后 `gvhook_init` 在正常线程上下文运行，可安全调用全部 API；③ 编排可**等待注入线程结束**再 `ResumeThread`，保证游戏主线程运行前钩子已装好 | 单线程 `CreateRemoteThread(LoadLibraryW, param)` 加 magic 扫描：`DllMain` 无法感知 `lpParameter`，只能脆弱地扫描内存找 magic；且 `DllMain` 内干重活违反 loader lock 纪律 |
| 5 | 注入入口导出名 **`gvhook_init`**（实施计划中的 `hook_init`） | 导出符号进入目标进程全局命名空间，通用名 `hook_init` 可能与游戏自身符号冲突；前缀 `gvhook_` 命名空间安全 | 裸 `hook_init` |
| 6 | 首进程的 `LoadLibraryW` 参数 = `param_block + offsetof(hook_dll_path)`（即参数块内偏移 32 处的 WCHAR 数组） | 参数块已含 DLL 绝对路径，无需单独分配路径缓冲 | 单独写一份路径字符串：浪费一块分配 |
| 7 | 同位数注入不变式（子进程位宽 == 钩子位宽），见 §6.4 | 跨位数无法 `CreateRemoteThread` 注入；WoW64 创建异位数子进程走另一条代码路径，本钩子不可见 | — |
| 8 | 日志文件打不开视为 `gvhook_init` 失败（`GVHD_INIT_ERR_LOG`） | 整套测试/断言体系依赖日志 marker；静默降级会让注入问题不可诊断 | 继续安装钩子但无日志 |

---

## 2. 类型、字节序与对齐规则

- **字节序**：Windows（x86/x64 均为）小端。`GVHD_PARAM_MAGIC` 按小端读出的 u32 值 = `0x44485647`，对应内存字节 `47 56 48 44`（即 ASCII `G V H D`）。Rust 侧等价写法：`u32::from_le_bytes(*b"GVHD")`。
- **`WCHAR`**：Windows 下恒为 16 位（UTF-16LE），x86/x64 均 `sizeof == 2`（头文件已 `_Static_assert`）。
- **`uint32_t`**：4 字节，对齐 4。
- **对齐规则**：结构体对齐 = 最大成员对齐 = 4。所有成员偏移天然满足对齐，编译器不插入 padding。**参数块与规则条目在 x86/x64 下字节布局完全一致**（无指针成员 → 无 8 字节对齐跳变）。
- 所有字符串字段要求 **NUL 结尾**；编排必须**先将整个参数块清零再逐字段写入**（保证字符串与保留字段就绪）。超出缓冲的内容被截断（保留结尾 NUL）。

---

## 3. 参数块 `gvhd_param_block`（5280 字节，x86/x64 一致）

```c
struct gvhd_param_block {
    uint32_t magic;              /* @0    */
    uint32_t version;            /* @4    */
    uint32_t flags;              /* @8    */
    uint32_t rule_count;         /* @12   */
    uint32_t rule_table_offset;  /* @16   */
    uint32_t game_id_len;        /* @20   */
    uint32_t reserved[2];        /* @24   */
    wchar_t  hook_dll_path[512]; /* @32   */
    wchar_t  game_data_root[512];/* @1056 */
    wchar_t  user_profile[512];  /* @2080 */
    wchar_t  log_path[512];      /* @3104 */
    wchar_t  registry_hive[512]; /* @4128 */
    wchar_t  game_id[64];        /* @5152 */
};
```

| 偏移 | 大小 | 字段 | 类型 | 对齐 | 说明 |
|---|---|---|---|---|---|
| 0 | 4 | `magic` | `uint32_t` | 4 | 恒为 `GVHD_PARAM_MAGIC` = `0x44485647`（`'G' 'V' 'H' 'D'`）。hook 用它验证「这是本协议的参数块」 |
| 4 | 4 | `version` | `uint32_t` | 4 | 恒为 `GVHD_PROTOCOL_VERSION` = 1。hook 拒绝不支持的版本 |
| 8 | 4 | `flags` | `uint32_t` | 4 | 参数块标志，见 §3.1 |
| 12 | 4 | `rule_count` | `uint32_t` | 4 | 规则条数，`0 ≤ n ≤ GVHD_RULE_MAX(32)` |
| 16 | 4 | `rule_table_offset` | `uint32_t` | 4 | **参数块基址到首条规则的字节偏移**（相对偏移，非指针）。须为 4 的倍数；推荐值 = `sizeof(gvhd_param_block)` = 5280（规则表紧跟在参数块后） |
| 20 | 4 | `game_id_len` | `uint32_t` | 4 | `game_id` 的 wchar 长度（不含 NUL）；0 = 无 game_id |
| 24 | 8 | `reserved[2]` | `uint32_t[2]` | 4 | 必须为 0。扩展预留，前向兼容（只可在此追加语义，不可缩改） |
| 32 | 1024 | `hook_dll_path` | `wchar_t[512]` | 2 | gvhook DLL **绝对路径**。既是子进程注入要复用的 DLL 路径，也是首进程 `LoadLibraryW` 的远程线程参数（§5 步骤 4） |
| 1056 | 1024 | `game_data_root` | `wchar_t[512]` | 2 | 沙箱重定向根，如 `E:\GameData`（VHD 内） |
| 2080 | 1024 | `user_profile` | `wchar_t[512]` | 2 | 宿主 `USERPROFILE`，如 `C:\Users\Hao`（编排解析得出；跨机器按新 USERPROFILE 重建） |
| 3104 | 1024 | `log_path` | `wchar_t[512]` | 2 | 日志文件**绝对路径**（格式见 §7）。hook 以追加方式打开 |
| 4128 | 1024 | `registry_hive` | `wchar_t[512]` | 2 | hive 文件绝对路径，如 `E:\GameData\Registry\user.dat`（供日志/诊断；挂载本身由编排完成） |
| 5152 | 128 | `game_id` | `wchar_t[64]` | 2 | 游戏 id，注册表隔离键名后缀 `GameVHD_<game_id>`；无则空串 |
| **5280** | | **总计** | | 4 | `sizeof(struct gvhd_param_block)` |

### 3.1 参数块 flags

| 位 | 掩码 | 名称 | 含义 |
|---|---|---|---|
| 0 | `0x00000001` | `GVHD_PARAM_FLAG_CHILD_INJECT` | 启用子进程自动注入（proc.c 安装 `NtCreateUserProcess`/`NtCreateProcessEx` 钩子）。编排应默认置位 |
| 1 | `0x00000002` | `GVHD_PARAM_FLAG_LOG_VERBOSE` | 详细日志（记录 NtQueryAttributesFile 等调用，阶段 1 验证链路用） |
| 2–31 | — | 预留 | 必须为 0 |

---

## 4. 规则表 `gvhd_rule_entry`（每条 4104 字节，x86/x64 一致）

```c
struct gvhd_rule_entry {
    wchar_t  prefix[1024];      /* @0    */
    wchar_t  replacement[1024]; /* @2048 */
    uint32_t flags;             /* @4096 */
    uint32_t reserved;          /* @4100 */
};
```

| 偏移 | 大小 | 字段 | 类型 | 对齐 | 说明 |
|---|---|---|---|---|---|
| 0 | 2048 | `prefix` | `wchar_t[1024]` | 2 | 匹配前缀：**绝对路径**，大小写不敏感。如 `C:\Users\Hao\Documents` |
| 2048 | 2048 | `replacement` | `wchar_t[1024]` | 2 | REWRITE 时的替换前缀。如 `E:\GameData\Users\Hao\Documents` |
| 4096 | 4 | `flags` | `uint32_t` | 4 | 动作标志，见 §4.1 |
| 4100 | 4 | `reserved` | `uint32_t` | 4 | 必须为 0 |
| **4104** | | **总计** | | 4 | `sizeof(struct gvhd_rule_entry)` |

### 4.1 规则动作标志

| 掩码 | 名称 | 语义 |
|---|---|---|
| `0x00000001` | `GVHD_RULE_FLAG_REWRITE` | 命中 → 重写到 `replacement` |
| `0x00000002` | `GVHD_RULE_FLAG_PASSTHROUGH` | 命中 → 不重写（直通宿主/原路径） |
| 其他 | — | 无效；两动作位同时置位时 **REWRITE 优先** |

### 4.2 匹配语义（表序优先）

- 对目标绝对路径，**按表序**逐条 `gvhd_rule_match_prefix(path, entry.prefix)`（大小写不敏感前缀匹配，见 `rules.h`）。**第一条命中的条目决定动作**，不再继续。
- 空前缀永不匹配。
- 命中长度 = 匹配到的前缀的 wchar 数（0 = 未命中）。
- 表序由编排负责（`runtime/rules.rs` 生成），推荐顺序（实施计划 §3.3）：
  1. **沙箱根/VHD 盘符直通**（游戏本体已在差分盘上，如 `E:\` → PASSTHROUGH）；
  2. **USERPROFILE 重写区**（`Documents`、`AppData`（Local/LocalLow/Roaming）、`Saved Games` → REWRITE）；
  3. **系统/其他直通区**（`C:\Windows`、`C:\Program Files*`、其他盘符 → PASSTHROUGH）。
- 匹配基于**含盘符的完整绝对路径**；`\\?\` 长路径前缀由实现层（W3T9）规范化后按同一规则处理。

### 4.3 重写公式

```
命中条目 action == REWRITE，matched = 前缀命中长度：
    rewritten_path = entry.replacement + path[matched ..]
例如：C:\Users\Hao\Documents\Save\a.dat
      prefix=C:\Users\Hao\Documents, replacement=E:\GameData\Users\Hao\Documents
  →   E:\GameData\Users\Hao\Documents\Save\a.dat
path 恰好等于 prefix 时 → replacement（目录本身）。
action == PASSTHROUGH → path 原样使用。
```

---

## 5. 顶层注入序列（编排 → 首进程）

前提：编排已探测游戏 exe 位数（PE header），**用同位数编排启动**（实施计划 §3.1）。以下句柄均为跨进程操作。

### 5.1 远程内存布局（单次 `VirtualAllocEx`）

```
目标进程远程内存（单块连续，MEM_COMMIT|MEM_RESERVE, PAGE_READWRITE）：
  +0x0000  gvhd_param_block                      (5280 B)
  +0x14A0  rule_entry[0]                          (4104 B)   ← rule_table_offset = 5280
  +0x2A38  rule_entry[1]                          (4104 B)
  +...     共 rule_count 条
总大小 = 5280 + rule_count * 4104   （rule_count=10 时约 46 KB）
```

### 5.2 步骤

1. `CreateProcessW(..., CREATE_SUSPENDED)` → `hProcess`, `hThread`。
   - 主线程保持挂起，注入完成前**游戏一行代码都不运行**。
2. `VirtualAllocEx(hProcess, NULL, region_size, MEM_COMMIT|MEM_RESERVE, PAGE_READWRITE)` → `remote_base`。
3. 构造参数块：**先整体清零**，再填 `magic`、`version`、`flags`、`rule_count`、`rule_table_offset = 5280`、各字符串（UTF-16LE + NUL 结尾）。`WriteProcessMemory` 一次性写入 `[参数块 | 规则表]` 整块。
4. 线程①：`CreateRemoteThread(hProcess, LoadLibraryW, (LPVOID)(remote_base + 32))`。
   - `+32` 即 `offsetof(hook_dll_path)`：`LoadLibraryW` 直接以参数块内的 DLL 路径为参数加载 gvhook。
   - `LoadLibraryW` 地址从本进程 `GetProcAddress(kernel32, "LoadLibraryW")` 取得（kernel32 为 known-DLL，跨进程同基址，注入器标准假设）。
5. `WaitForSingleObject` 线程①结束；`GetExitCodeThread` 得 `remote_hmod`（目标进程中 gvhook 的 HMODULE）。为 0 → DLL 加载失败，走 §5.3 失败路径。
6. 解析 `gvhook_init` 在目标进程的地址（**RVA 换算**，同文件保证导出 RVA 相同）：
   - 编排在本进程加载**同位数** gvhook：`local_hmod = LoadLibraryW(hook_dll_path)`；
   - `local_fn = GetProcAddress(local_hmod, "gvhook_init")`；
   - `rva = (BYTE*)local_fn - (BYTE*)local_hmod`；
   - `remote_fn = (BYTE*)remote_hmod + rva`。
7. 线程②：`CreateRemoteThread(hProcess, remote_fn, (LPVOID)remote_base)`（即 `gvhook_init(param_block)`）。
8. `WaitForSingleObject` 线程②结束；`GetExitCodeThread` 读返回码：`0`（`GVHD_INIT_OK`）为成功，非 0 见 §10。
9. `ResumeThread(hThread)` —— 此刻钩子已全部安装。
10. 编排可 `VirtualFreeEx(hProcess, remote_base, ...)` 释放远程区域（hook 已在 `gvhook_init` 内保存私有副本，见 §6.1）。

### 5.3 失败路径

任一步失败：释放远程区域（`VirtualFreeEx`）→ 关闭句柄 → **`TerminateProcess`** → 编排以错误码退出并记录诊断。不得以「未完全注入」的状态放行游戏运行。

---

## 6. 子进程注入序列（hook → 子进程，proc.c）

### 6.1 hook 的私有副本（前置）

`gvhook_init` 成功时，hook 必须将参数块 + 规则表**原样复制到自己的堆**（`gvhd_get_param()` / `gvhd_get_rules()` / `gvhd_get_rule_count()` 返回该副本）。原因：
- 编排可能在 `gvhook_init` 返回后释放远程区域（§5.2 步骤 10）；
- 子进程注入需要**整块逐字节复制**进子进程（偏移式规则表无需任何地址翻译）。

### 6.2 钩子行为

1. proc.c 钩住 `NtCreateUserProcess` / `NtCreateProcessEx`。
2. 钩子调用原函数（trampoline）时**强制 `CreateFlags |= CREATE_SUSPENDED (0x4)`**，使子进程以挂起状态创建。
3. 对返回的 `hProcess` / `hThread` 调用 `gvhd_inject_child(hProcess, hThread)`：
   - 在子进程 `VirtualAllocEx` 分配 `5280 + rule_count*4104` 区域；
   - `WriteProcessMemory` 写入父进程私有副本的**参数块 + 规则表整块**（`hook_dll_path` 为绝对路径，本机父子进程同一路径，无需修改；`rule_table_offset` 相对偏移，无需翻译）；
   - 线程①：`CreateRemoteThread(child, LoadLibraryW, child_base + 32)`，等待结束得 `child_hmod`；
   - 计算 `gvhook_init` 在子进程的地址：proc.c 用**自身镜像**算 RVA——`rva = (BYTE*)&gvhook_init - (BYTE*)GetModuleHandleW(NULL)`（同 DLL 文件 → 同导出 RVA），再 `remote_fn = (BYTE*)child_hmod + rva`；
   - 线程②：`CreateRemoteThread(child, remote_fn, child_base)`，等待结束；
   - `ResumeThread(child hThread)`。
4. 成功 → 日志 `CHILD_INJECTED pid=<child_pid>`；失败 → 日志告警 + `ResumeThread` 放行（不阻塞游戏）→ 返回原调用链。
5. 释放子进程远程区域（可选，子进程 hook 已存副本）。

### 6.3 不变式：子进程位宽 == 父进程位宽

- **为什么**：`CreateRemoteThread` + `LoadLibraryW` 无法跨位数注入（异位数进程的加载器路径不同，远程线程地址语义也不兼容）。本协议的所有传递都发生在**同位数地址空间对**之间。
- **为什么成立**：顶层由编排先探测 exe 位数、用同位数编排启动（§5 前提）。WoW64 下 32 位进程创建 64 位原生子进程会走 64 位 ntdll 的创建路径，32 位 `NtCreateUserProcess` 钩子不是实际创建者——即**钩子只见同位数子进程**。
- **防御**：若实现层观察到异位数子进程（注入失败可探测），**跳过注入 + 记警告日志**，绝不半注入。
- **跨位数覆盖由编排负责**：顶层跨位数场景（如 32 位启动器拉起 64 位主程序）由编排「先探测、再选择对应位数 runtime 重新执行」解决；协议本身位数中立，**同一份参数块布局、同一套注入代码**被 32/64 位两个构建复用，只是 gvhook 二进制不同（`gvhook-x86.dll` / `gvhook-x64.dll`）。

### 6.4 安全时刻

子进程在 `CREATE_SUSPENDED` 下创建，注入与钩子安装先于其任何用户代码；恢复运行后，其所有文件/注册表操作都已处于沙箱决策之下。进程树全覆盖（游戏 → ModManager → CrashReporter）。

### 6.5 注册表重写目标（reg.c 依据）

hive 由编排 `RegLoadKey(HKUS, "GameVHD_<game_id>", registry_hive)` 挂载（实施计划 §3.5）；reg.c 把 `\REGISTRY\USER\<当前 SID>\Software\X` 重写为 `\REGISTRY\USER\GameVHD_<game_id>\Software\X`，键名中的 `<game_id>` 即参数块 `game_id` 字段。`game_id` 为空 → 注册表钩子不安装/不重写（直通）。

---

## 7. 日志约定

- **文件**：参数块 `log_path` 指定的绝对路径。hook 以**追加**方式打开（`FILE_APPEND_DATA`），**从不截断**；多进程（父子）追加同一文件。编排负责在会话开始时清空（其自身职责，hook 不碰）。
- **格式**：每行一条 `WriteFile`，格式为

  ```
  [gvhook] YYYY-MM-DD HH:MM:SS.mmm <message>
  ```

  时间戳为本地时间。**断言脚本只按子串匹配 marker 片段**（时间戳前缀可变），因此 marker 宏只约定消息片段（见下）。
- **marker 行（断言脚本依赖的精确片段）**：

  | 片段 | 触发点 | 示例完整行 |
  |---|---|---|
  | `HOOK_DLL_PRESENT` | `gvhook_init` 全部成功（MinHook 就绪 + 钩子已装） | `[gvhook] 2026-08-07 12:00:00.123 HOOK_DLL_PRESENT` |
  | `RULES_LOADED <n>` | 规则表解析完成，n = 实际条数 | `[gvhook] 2026-08-07 12:00:00.124 RULES_LOADED 3` |
  | `CHILD_INJECTED pid=<pid>` | 每次子进程注入成功 | `[gvhook] 2026-08-07 12:00:00.130 CHILD_INJECTED pid=1234` |

- marker 宏定义（`hook_common.h`）：`GVHD_MARKER_PRESENT`、`GVHD_MARKER_RULES_LOADED`、`GVHD_MARKER_CHILD_INJECTED`。带参的两个由实现层拼装（`RULES_LOADED %u`、`CHILD_INJECTED pid=%lu`）。

---

## 8. 常量总表（C ↔ Rust 对照）

| 常量 | 值 | C（hook_common.h） | Rust（runtime/） |
|---|---|---|---|
| magic | `0x44485647` | `GVHD_PARAM_MAGIC` | `u32::from_le_bytes(*b"GVHD")` |
| 协议版本 | `1` | `GVHD_PROTOCOL_VERSION` | 同名常量 |
| 路径缓冲（WCHAR） | `512` | `GVHD_PATH_MAX` | `512` |
| 规则串缓冲（WCHAR） | `1024` | `GVHD_RULE_STRING_MAX` | `1024` |
| game_id 缓冲（WCHAR） | `64` | `GVHD_GAME_ID_MAX` | `64` |
| 最大规则条数 | `32` | `GVHD_RULE_MAX` | `32` |
| 参数块大小（字节） | `5280` | `sizeof(struct gvhd_param_block)` | `size_of::<GvhdParamBlock>()`（应 `debug_assert_eq!(_, 5280)`） |
| 规则条目大小（字节） | `4104` | `sizeof(struct gvhd_rule_entry)` | `size_of::<GvhdRuleEntry>()` |
| 规则表偏移推荐值 | `5280` | — | `size_of::<GvhdParamBlock>() as u32` |
| `LoadLibraryW` 参数偏移 | `32` | `offsetof(hook_dll_path)` | `32` |
| 参数标志：子进程注入 | `0x00000001` | `GVHD_PARAM_FLAG_CHILD_INJECT` | 同名 |
| 参数标志：详细日志 | `0x00000002` | `GVHD_PARAM_FLAG_LOG_VERBOSE` | 同名 |
| 规则标志：REWRITE | `0x00000001` | `GVHD_RULE_FLAG_REWRITE` | 同名 |
| 规则标志：PASSTHROUGH | `0x00000002` | `GVHD_RULE_FLAG_PASSTHROUGH` | 同名 |

Rust 参考定义（布局由 `#[repr(C)]` + 自然对齐保证与 C 一致；无指针字段 → x86/x64 同一结构）：

```rust
#[repr(C)]
pub struct GvhdParamBlock {
    pub magic: u32,               // u32::from_le_bytes(*b"GVHD")
    pub version: u32,
    pub flags: u32,
    pub rule_count: u32,
    pub rule_table_offset: u32,
    pub game_id_len: u32,
    pub reserved: [u32; 2],
    pub hook_dll_path: [u16; 512],
    pub game_data_root: [u16; 512],
    pub user_profile: [u16; 512],
    pub log_path: [u16; 512],
    pub registry_hive: [u16; 512],
    pub game_id: [u16; 64],
}

#[repr(C)]
pub struct GvhdRuleEntry {
    pub prefix: [u16; 1024],
    pub replacement: [u16; 1024],
    pub flags: u32,
    pub reserved: u32,
}
```

---

## 9. 对齐与大小验证（_Static_assert 对照表）

C 头文件内置的编译期断言在 **x86 与 x64 双架构**下都执行；任一偏移/大小不符即编译失败。这与本文档各表一一对应：

| 断言对象 | 期望值 |
|---|---|
| `sizeof(wchar_t)` | 2 |
| `sizeof(struct gvhd_param_block)` | 5280 |
| `offsetof(param, magic/version/flags/rule_count/rule_table_offset/game_id_len)` | 0 / 4 / 8 / 12 / 16 / 20 |
| `offsetof(param, reserved)` | 24 |
| `offsetof(param, hook_dll_path)` | 32 |
| `offsetof(param, game_data_root)` | 1056 |
| `offsetof(param, user_profile)` | 2080 |
| `offsetof(param, log_path)` | 3104 |
| `offsetof(param, registry_hive)` | 4128 |
| `offsetof(param, game_id)` | 5152 |
| `sizeof(struct gvhd_rule_entry)` | 4104 |
| `offsetof(rule, replacement)` | 2048 |
| `offsetof(rule, flags)` | 4096 |
| `offsetof(rule, reserved)` | 4100 |

**布局推理**（为何 x86/x64 一致）：参数块与规则条目均无指针成员，最大成员对齐为 4（`uint32_t`），`wchar_t` 对齐为 2；所有成员偏移天然是 2 的倍数、数组边界是 4 的倍数，编译器在两种架构下都不插入 padding；结构体对齐 = 4。因此在两种架构下字节布局逐位相同——这正是选择「相对偏移而非指针」的直接收益。

---

## 10. 错误处理与自检

- `gvhook_init` 返回码（编排经线程② `GetExitCodeThread` 读取并记日志）：

  | 码 | 宏 | 含义 |
  |---|---|---|
  | 0 | `GVHD_INIT_OK` | 成功，日志含 `HOOK_DLL_PRESENT` / `RULES_LOADED <n>` |
  | 1 | `GVHD_INIT_ERR_MAGIC` | 参数块 magic 不匹配 |
  | 2 | `GVHD_INIT_ERR_VERSION` | version 不受支持 |
  | 3 | `GVHD_INIT_ERR_MINHOOK` | MinHook 初始化失败 |
  | 4 | `GVHD_INIT_ERR_HOOK` | 钩子安装失败 |
  | 5 | `GVHD_INIT_ERR_LOG` | 日志文件无法创建/打开 |

- 自检顺序（`gvhook_init` 内部，W2T6 实现）：magic → version → 复制私有副本 → 打开日志 → MinHook 初始化 → 安装钩子 → `RULES_LOADED <n>` → `HOOK_DLL_PRESENT`。任一步失败按上表返回，**不写**成功 marker。

---

## 11. Rust 侧实现要点（runtime/，零依赖 FFI）

- 不要引入任何 Windows FFI crate；直接 `#[link(name = "kernel32")] extern "system"` 声明 `CreateProcessW` / `VirtualAllocEx` / `WriteProcessMemory` / `CreateRemoteThread` / `WaitForSingleObject` / `GetExitCodeThread` / `GetProcAddress` / `ResumeThread` / `VirtualFreeEx` / `TerminateProcess`。
- 字符串写入：`encode_utf16` 后截断到 `N - 1` 个码元并补 NUL（**先整体清零**再写）。
- `LoadLibraryW` 与 `GetProcAddress` 用本进程解析（known-DLL 同基址假设，§5.2 步骤 4）；`gvhook_init` 远程地址用 RVA 换算（§5.2 步骤 6）。
- 注入线程句柄全部 `CloseHandle`；失败路径按 §5.3 处理。
- 编排自身结构体加 `#[repr(C)]` 并在 `#[cfg(test)]` 断言 `size_of` == 5280 / 4104，与 C 头文件 `_Static_assert` 互为镜像。

---

## 12. 波次归属与变更记录

| 变更 | 日期 | 说明 |
|---|---|---|
| 定案 v1 | 2026-08-07 | 注入协议全字段规格、双序列、日志约定 |

- `hook/src/init.c`、`hook/src/proc.c` → **W2T6**（注入与 hook 骨架：参数解析、自检日志、子进程注入、NtQueryAttributesFile 打日志验证链路）。
- `hook/src/file.c` → **W3T9**（文件系统重定向）。
- `hook/src/reg.c` → **W3T12**（注册表重定向）。
- `hook/src/Makefile` → **W2T6** 创建（本协议只负责源文件结构：`init.c proc.c file.c reg.c` + `hook_common.h rules.h` + `deps/minhook`，Makefile 需定义 `GVHD_BUILD_DLL` 并以 `-std=c11 -Wall -Wextra` 编译，x64/x86 双目标）。
