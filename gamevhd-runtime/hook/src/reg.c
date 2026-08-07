/*
 * gvhook — reg.c
 *
 * 归属波次：W3T12（阶段 3 任务 12：注册表重定向）。
 * 依据 docs/GameVHD_Runtime_实施计划.md §3.2（注册表 hook 清单）/ §3.5（hive 设计）
 * 与 docs/injection_protocol.md §6.5（重写目标键名）实现。
 *
 * ── 职责 ─────────────────────────────────────────────────────────────
 *   hive 由编排 RegLoadKey 挂载于 HKU\GameVHD_<game_id>（键名 = 参数块 game_id），
 *   hook 只做「路径重写」：
 *
 *   1. 重写核心 gvhd_reg_rewrite()：
 *        \REGISTRY\USER\<当前SID>\Software\X
 *          → \REGISTRY\USER\GameVHD_<game_id>\Software\X
 *      仅 Software 子树重写；Environment / Control Panel / AppEvents 等其它
 *      HKCU 根、.DEFAULT 与其它用户 SID → 直通宿主。\REGISTRY\MACHINE（HKLM）
 *      直通宿主 + 节流警告 REG_HKLM_ACCESS（每 5000 次调用至多 1 条）。
 *   2. 写操作（NtCreateKey*）：一律沙箱路径 + hive 内父链自动创建
 *      （打开即创建中间键，KEY_CREATE_SUB_KEY|KEY_WOW64_64KEY，走原函数跳板）。
 *   3. 读操作（NtOpenKey*）：读穿透——hive 无键（NOT_FOUND/PATH_NOT_FOUND/
 *      ACCESS_DENIED）→ 回退宿主路径重试，节流日志 REG_READTHROUGH。
 *   4. 删除（NtDeleteKey/Ex、NtDeleteValueKey）：句柄级保护——先查句柄 native
 *      路径并重写为 hive 路径探测（KEY_QUERY_VALUE 打开）；hive 无对应键 →
 *      直接返回 STATUS_OBJECT_NAME_NOT_FOUND，原函数不被调用，宿主永不被删。
 *   5. 句柄级读写（NtSetValueKey/NtQueryValueKey/NtEnumerate* / NtQueryKey/
 *      NtRenameKey/NtNotifyChangeKey）：由「打开即重定向」保证——句柄已在
 *      打开/创建时指向 hive 或宿主的正确键，直接透传原函数。
 *
 * ── 已知局限（MVP，实施计划 §3.4 / §9 / §7 风险表）──────────────────────
 *   1. 枚举不做真正双亲合并：键仅存在于宿主（读穿透场景）→ 枚举宿主；键在
 *      hive → 枚举 hive。键同时存在于宿主与 hive 且内容不同的双合并语义不
 *      实现（游戏极少枚举这种键）。
 *   2. NtNotifyChangeKey 直通宿主：对宿主键的变更通知收不到 hive 变更，反之
 *      亦然（实施计划 §7 风险表「先直通，标注已知局限」）。
 *   3. NtRenameKey 直通宿主：经读穿透得到的宿主句柄上重命名会改宿主键名
 *      （罕见路径，MVP 不拦截）。
 *   4. 经读穿透打开的宿主键句柄上做 NtSetValueKey 会写宿主（「打开即重定向，
 *      句柄后续操作免重写」的固有限制，实施计划 §3.4；测试流建键走 NtCreateKey
 *      沙箱，句柄天然指向 hive）。
 *   5. 32 位游戏的 Wow6432Node：随 Software 子树自然重写进 hive，hive 内有
 *      自己的 Wow6432Node 空间按需创建；不做任何特殊合并（实施计划 §3.5）。
 *   6. 路径超缓冲（GVHD_REG_PATH_MAX）时宁可直通宿主也不截断，避免写出错路径。
 */
// allow: SIZE_OK — 实施计划 §6/§12 将注册表重定向固定归属本文件（reg.c），
// 结构与波次由单一事实源 docs/injection_protocol.md §12 规定。

#include <stdint.h>
#include <string.h>
#include <wchar.h>

#include <windows.h>
#include <winternl.h>

#include "hook_common.h"
#include "rules.h"
#include "internal.h"
#include "MinHook.h"

/* ================================================================ */
/* 常量                                                               */
/* ================================================================ */

#define GVHD_REG_PREFIX_USER      L"\\REGISTRY\\USER\\"
#define GVHD_REG_PREFIX_MACHINE   L"\\REGISTRY\\MACHINE"
#define GVHD_REG_HIVE_ROOT        L"\\REGISTRY\\USER\\GameVHD_"
#define GVHD_REG_PATH_MAX         1024u   /* 重写缓冲（wchar 数） */

/* 节流：每 N 次同类调用至多记录 1 条日志，避免刷屏 */
#define GVHD_REG_HKLM_WARN_EVERY   5000u
#define GVHD_REG_READTHRU_EVERY    1000u

/* NTSTATUS 等价常量（ntstatus.h 在 mingw 未随 winternl 提供；u 后缀避免
 * long 溢出告警，与 proc.c 同风格） */
#define GVHD_STATUS_SUCCESS                 ((NTSTATUS)0)
#define GVHD_STATUS_OBJECT_NAME_NOT_FOUND   ((NTSTATUS)0xC0000034u)
#define GVHD_STATUS_OBJECT_PATH_NOT_FOUND   ((NTSTATUS)0xC000003Au)
#define GVHD_STATUS_ACCESS_DENIED           ((NTSTATUS)0xC0000022u)
#define GVHD_STATUS_BUFFER_OVERFLOW         ((NTSTATUS)0x80000005u)
#define GVHD_STATUS_INSUFFICIENT_RESOURCES  ((NTSTATUS)0xC000009Au)

/* ================================================================ */
/* 注册表枚举/结构（mingw-w64 winternl.h 未提供，本地定义；ddk/wdm.h 也
 *   有同名枚举但我们不引入 ddk，用 #ifndef 防御未来头文件补全）            */
/* ================================================================ */

#ifndef KEY_INFORMATION_CLASS
typedef enum _KEY_INFORMATION_CLASS {
    KeyBasicInformation = 0,
    KeyNodeInformation = 1,
    KeyFullInformation = 2,
    KeyNameInformation = 3
} KEY_INFORMATION_CLASS;
#endif

#ifndef KEY_VALUE_INFORMATION_CLASS
typedef enum _KEY_VALUE_INFORMATION_CLASS {
    KeyValueBasicInformation = 0,
    KeyValueFullInformation = 1,
    KeyValuePartialInformation = 2,
    KeyValueFullInformationAlign64 = 3,
    KeyValuePartialInformationAlign64 = 4
} KEY_VALUE_INFORMATION_CLASS;
#endif

#ifndef KEY_NAME_INFORMATION
typedef struct _KEY_NAME_INFORMATION {
    ULONG NameLength;
    WCHAR Name[1];
} KEY_NAME_INFORMATION, *PKEY_NAME_INFORMATION;
#endif

/* ================================================================ */
/* NT 函数签名（GetProcAddress 解析，不静态链接 ntdll 除 NtClose 外）     */
/* ================================================================ */

typedef NTSTATUS(NTAPI *P_NtOpenKey)(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes);
typedef NTSTATUS(NTAPI *P_NtOpenKeyEx)(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG OpenOptions);
typedef NTSTATUS(NTAPI *P_NtOpenKeyTransacted)(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    HANDLE TransactionHandle);
typedef NTSTATUS(NTAPI *P_NtOpenKeyTransactedEx)(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG OpenOptions, HANDLE TransactionHandle);
typedef NTSTATUS(NTAPI *P_NtCreateKey)(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG TitleIndex, PUNICODE_STRING Class, ULONG CreateOptions, PULONG Disposition);
typedef NTSTATUS(NTAPI *P_NtCreateKeyEx)(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG TitleIndex, PUNICODE_STRING Class, ULONG CreateOptions,
    PULONG Disposition, PULONG ExtendedDisposition);
typedef NTSTATUS(NTAPI *P_NtCreateKeyTransacted)(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG TitleIndex, PUNICODE_STRING Class, ULONG CreateOptions,
    HANDLE TransactionHandle, PULONG Disposition);
typedef NTSTATUS(NTAPI *P_NtCreateKeyTransactedEx)(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG TitleIndex, PUNICODE_STRING Class, ULONG CreateOptions,
    HANDLE TransactionHandle, PULONG Disposition, PULONG ExtendedDisposition);
typedef NTSTATUS(NTAPI *P_NtSetValueKey)(
    HANDLE KeyHandle, PUNICODE_STRING ValueName, ULONG TitleIndex, ULONG Type,
    PVOID Data, ULONG DataSize);
typedef NTSTATUS(NTAPI *P_NtDeleteValueKey)(
    HANDLE KeyHandle, PUNICODE_STRING ValueName);
typedef NTSTATUS(NTAPI *P_NtDeleteKey)(
    HANDLE KeyHandle);
typedef NTSTATUS(NTAPI *P_NtDeleteKeyEx)(
    HANDLE KeyHandle, ULONG Flags);
typedef NTSTATUS(NTAPI *P_NtQueryValueKey)(
    HANDLE KeyHandle, PUNICODE_STRING ValueName, KEY_VALUE_INFORMATION_CLASS KeyValueInformationClass,
    PVOID KeyValueInformation, ULONG Length, PULONG ResultLength);
typedef NTSTATUS(NTAPI *P_NtQueryMultipleValueKey)(
    HANDLE KeyHandle, PKEY_VALUE_ENTRY ValueEntries, ULONG EntryCount,
    PVOID ValueBuffer, PULONG BufferLength, PULONG RequiredBufferLength);
typedef NTSTATUS(NTAPI *P_NtEnumerateValueKey)(
    HANDLE KeyHandle, ULONG Index, KEY_VALUE_INFORMATION_CLASS KeyValueInformationClass,
    PVOID KeyValueInformation, ULONG Length, PULONG ResultLength);
typedef NTSTATUS(NTAPI *P_NtQueryKey)(
    HANDLE KeyHandle, KEY_INFORMATION_CLASS KeyInformationClass,
    PVOID KeyInformation, ULONG Length, PULONG ResultLength);
typedef NTSTATUS(NTAPI *P_NtEnumerateKey)(
    HANDLE KeyHandle, ULONG Index, KEY_INFORMATION_CLASS KeyInformationClass,
    PVOID KeyInformation, ULONG Length, PULONG ResultLength);
typedef NTSTATUS(NTAPI *P_NtRenameKey)(
    HANDLE KeyHandle, PUNICODE_STRING NewName);
typedef NTSTATUS(NTAPI *P_NtNotifyChangeKey)(
    HANDLE KeyHandle, HANDLE Event, PIO_APC_ROUTINE ApcRoutine, PVOID ApcContext,
    PIO_STATUS_BLOCK IoStatusBlock, ULONG CompletionFilter, BOOLEAN WatchTree,
    PVOID Buffer, ULONG BufferSize, BOOLEAN Asynchronous);
typedef NTSTATUS(NTAPI *P_NtClose)(
    HANDLE Handle);

/* ================================================================ */
/* 全局：原函数地址（MinHook target）与跳板（original）                  */
/* ================================================================ */

static LPVOID __sys_NtOpenKey = NULL;
static LPVOID __sys_NtOpenKeyEx = NULL;
static LPVOID __sys_NtOpenKeyTransacted = NULL;
static LPVOID __sys_NtOpenKeyTransactedEx = NULL;
static LPVOID __sys_NtCreateKey = NULL;
static LPVOID __sys_NtCreateKeyEx = NULL;
static LPVOID __sys_NtCreateKeyTransacted = NULL;
static LPVOID __sys_NtCreateKeyTransactedEx = NULL;
static LPVOID __sys_NtSetValueKey = NULL;
static LPVOID __sys_NtDeleteValueKey = NULL;
static LPVOID __sys_NtDeleteKey = NULL;
static LPVOID __sys_NtDeleteKeyEx = NULL;
static LPVOID __sys_NtQueryValueKey = NULL;
static LPVOID __sys_NtQueryMultipleValueKey = NULL;
static LPVOID __sys_NtEnumerateValueKey = NULL;
static LPVOID __sys_NtQueryKey = NULL;
static LPVOID __sys_NtEnumerateKey = NULL;
static LPVOID __sys_NtRenameKey = NULL;
static LPVOID __sys_NtNotifyChangeKey = NULL;
static LPVOID __sys_NtClose = NULL;

static P_NtOpenKey               pfnOrig_NtOpenKey = NULL;
static P_NtOpenKeyEx             pfnOrig_NtOpenKeyEx = NULL;
static P_NtOpenKeyTransacted     pfnOrig_NtOpenKeyTransacted = NULL;
static P_NtOpenKeyTransactedEx   pfnOrig_NtOpenKeyTransactedEx = NULL;
static P_NtCreateKey             pfnOrig_NtCreateKey = NULL;
static P_NtCreateKeyEx           pfnOrig_NtCreateKeyEx = NULL;
static P_NtCreateKeyTransacted   pfnOrig_NtCreateKeyTransacted = NULL;
static P_NtCreateKeyTransactedEx pfnOrig_NtCreateKeyTransactedEx = NULL;
static P_NtSetValueKey           pfnOrig_NtSetValueKey = NULL;
static P_NtDeleteValueKey        pfnOrig_NtDeleteValueKey = NULL;
static P_NtDeleteKey             pfnOrig_NtDeleteKey = NULL;
static P_NtDeleteKeyEx           pfnOrig_NtDeleteKeyEx = NULL;
static P_NtQueryValueKey         pfnOrig_NtQueryValueKey = NULL;
static P_NtQueryMultipleValueKey pfnOrig_NtQueryMultipleValueKey = NULL;
static P_NtEnumerateValueKey     pfnOrig_NtEnumerateValueKey = NULL;
static P_NtQueryKey              pfnOrig_NtQueryKey = NULL;
static P_NtEnumerateKey          pfnOrig_NtEnumerateKey = NULL;
static P_NtRenameKey             pfnOrig_NtRenameKey = NULL;
static P_NtNotifyChangeKey       pfnOrig_NtNotifyChangeKey = NULL;

/* ================================================================ */
/* 节流计数器（Interlocked，多线程安全）                                 */
/* ================================================================ */

static volatile LONG gvhd_reg_hklm_count = 0;
static volatile LONG gvhd_reg_readthru_count = 0;

static void gvhd_reg_warn_hklm(const wchar_t *path)
{
    LONG n = InterlockedIncrement(&gvhd_reg_hklm_count);

    if (n % (LONG)GVHD_REG_HKLM_WARN_EVERY == 1) {
        gvhd_log_write(L"REG_HKLM_ACCESS %ls", path);
    }
}

static void gvhd_reg_log_readthrough(const wchar_t *path)
{
    LONG n = InterlockedIncrement(&gvhd_reg_readthru_count);

    if (n % (LONG)GVHD_REG_READTHRU_EVERY == 1) {
        gvhd_log_write(L"REG_READTHROUGH %ls", path);
    }
}

/* ================================================================ */
/* 大小写不敏感字符串比较（ASCII 区，规则同 rules.h 的 gvhd_ascii_lower）   */
/* ================================================================ */

static BOOLEAN gvhd_reg_ieq(const wchar_t *a, const wchar_t *b)
{
    for (;;) {
        wchar_t ca = gvhd_ascii_lower(*a);
        wchar_t cb = gvhd_ascii_lower(*b);

        if (ca != cb) {
            return FALSE;
        }
        if (ca == L'\0') {
            return TRUE;
        }
        ++a;
        ++b;
    }
}

static BOOLEAN gvhd_reg_ieq_prefix(const wchar_t *path, const wchar_t *prefix)
{
    size_t i;

    if (prefix[0] == L'\0') {
        return FALSE;
    }
    for (i = 0; prefix[i] != L'\0'; ++i) {
        if (gvhd_ascii_lower(path[i]) != gvhd_ascii_lower(prefix[i])) {
            return FALSE;
        }
    }
    return TRUE;
}

/* ================================================================ */
/* 重写核心（§3.5 / §6.5）                                             */
/* ================================================================ */

/* native 是可重写的 HKCU 绝对路径时，把重写结果写入 out 并返回 out；
 * 否则（宿主直通场景）返回 NULL，out 内容未定义。 */
static wchar_t *gvhd_reg_rewrite(const wchar_t *native, wchar_t *out, size_t out_cap)
{
    const struct gvhd_param_block *param = gvhd_get_param();
    const wchar_t *p;
    const wchar_t *rest;
    size_t prefix_len;
    size_t game_len;
    size_t rest_len;
    size_t total;

    if (native == NULL || out == NULL || out_cap == 0) {
        return NULL;
    }
    if (param->game_id[0] == L'\0') {
        return NULL;   /* 无 game_id → 无隔离，全直通（协议 §6.5） */
    }

    if (gvhd_reg_ieq_prefix(native, GVHD_REG_PREFIX_USER)) {
        p = native + wcslen(GVHD_REG_PREFIX_USER);   /* 指向 <SID> 段 */
        /* .DEFAULT 与其它用户 SID → 直通宿主（只重写当前用户自己的 hive） */
        if (gvhd_reg_ieq_prefix(p, L".DEFAULT\\")) {
            return NULL;
        }
        rest = wcschr(p, L'\\');                     /* <SID> 后的第一个 '\' */
        if (rest == NULL) {
            return NULL;   /* \REGISTRY\USER\<SID> 根本身 → 宿主 */
        }
        rest += 1;                                    /* 指向 e.g. Software\... */
        /* 仅 Software 子树重写；Environment / Control Panel / AppEvents 等直通 */
        if (!gvhd_reg_ieq(rest, L"Software") &&
            !gvhd_reg_ieq_prefix(rest, L"Software\\")) {
            return NULL;
        }
        /* 组装 out = \REGISTRY\USER\GameVHD_<game_id>\ + rest */
        prefix_len = wcslen(GVHD_REG_HIVE_ROOT);
        game_len = wcslen(param->game_id);
        rest_len = wcslen(rest);
        total = prefix_len + game_len + rest_len;
        if (total + 1 > out_cap) {
            return NULL;   /* 超缓冲：宁可直通也不截断 */
        }
        memcpy(out, GVHD_REG_HIVE_ROOT, prefix_len * sizeof(wchar_t));
        memcpy(out + prefix_len, param->game_id, game_len * sizeof(wchar_t));
        memcpy(out + prefix_len + game_len, rest, (rest_len + 1) * sizeof(wchar_t));
        return out;
    }

    if (gvhd_reg_ieq_prefix(native, GVHD_REG_PREFIX_MACHINE)) {
        gvhd_reg_warn_hklm(native);
        return NULL;   /* HKLM 直通宿主 + 节流警告 */
    }
    return NULL;
}

/* ================================================================ */
/* OBJECT_ATTRIBUTES 重写评估（每 hook 调用一次，栈上局部数据）            */
/* ================================================================ */

struct gvhd_reg_rewritten {
    BOOLEAN active;                              /* TRUE → 用 &oa 调原函数 */
    OBJECT_ATTRIBUTES oa;                        /* 重写后的 OA（RootDirectory=NULL） */
    UNICODE_STRING name;                         /* 重写后的名字 */
    wchar_t name_buf[GVHD_REG_PATH_MAX];         /* 重写结果缓冲 */
    wchar_t orig_buf[GVHD_REG_PATH_MAX];         /* NUL 结尾原始路径（日志/回退用） */
};

static void gvhd_reg_eval_oa(const OBJECT_ATTRIBUTES *oa_in,
                             struct gvhd_reg_rewritten *rw)
{
    const wchar_t *native;
    wchar_t *res;
    size_t chars;
    size_t cap = GVHD_REG_PATH_MAX;

    memset(rw, 0, sizeof(*rw));
    if (oa_in == NULL || oa_in->ObjectName == NULL ||
        oa_in->ObjectName->Buffer == NULL || oa_in->ObjectName->Length == 0) {
        return;
    }
    /* 相对打开（RootDirectory != NULL）：根句柄已在打开时解析，不重写 */
    if (oa_in->RootDirectory != NULL) {
        return;
    }
    chars = (size_t)(oa_in->ObjectName->Length / sizeof(wchar_t));
    if (chars >= cap) {
        return;   /* 超长路径：直通 */
    }
    memcpy(rw->orig_buf, oa_in->ObjectName->Buffer, chars * sizeof(wchar_t));
    rw->orig_buf[chars] = L'\0';
    native = rw->orig_buf;

    res = gvhd_reg_rewrite(native, rw->name_buf, cap);
    if (res == NULL) {
        return;
    }

    gvhd_log_write(L"REG_REWRITE src=%ls dst=%ls", native, res);

    rw->name.Buffer = rw->name_buf;
    rw->name.Length = (USHORT)(wcslen(res) * sizeof(wchar_t));
    rw->name.MaximumLength = (USHORT)(cap * sizeof(wchar_t));

    rw->oa = *oa_in;
    rw->oa.RootDirectory = NULL;
    rw->oa.ObjectName = &rw->name;
    rw->active = TRUE;
}

/* ================================================================ */
/* hive 内父链自动创建（NtCreateKey 的中间键缺失处理）                     */
/* ================================================================ */

static NTSTATUS gvhd_reg_close(HANDLE h)
{
    P_NtClose pNtClose = (P_NtClose)(LPVOID)__sys_NtClose;

    if (pNtClose == NULL) {
        return GVHD_STATUS_SUCCESS;
    }
    return pNtClose(h);
}

/* 用原函数以「打开即创建」语义确保单个键存在（只创建缺失的父键本身）。 */
static NTSTATUS gvhd_reg_open_or_create(const wchar_t *key_path)
{
    UNICODE_STRING name;
    OBJECT_ATTRIBUTES oa;
    HANDLE hKey = NULL;
    NTSTATUS status;

    name.Buffer = (PWSTR)key_path;   /* 只读用途；NtCreateKey 不修改名字 */
    name.Length = (USHORT)(wcslen(key_path) * sizeof(wchar_t));
    name.MaximumLength = (USHORT)(name.Length + sizeof(wchar_t));

    memset(&oa, 0, sizeof(oa));
    oa.Length = sizeof(OBJECT_ATTRIBUTES);
    oa.RootDirectory = NULL;
    oa.ObjectName = &name;
    oa.Attributes = OBJ_CASE_INSENSITIVE;

    status = pfnOrig_NtCreateKeyEx(&hKey, KEY_CREATE_SUB_KEY | KEY_WOW64_64KEY,
                                   &oa, 0, NULL, 0, NULL, NULL);
    if (hKey != NULL) {
        gvhd_reg_close(hKey);
    }
    return status;
}

/* 在 hive 内创建 path 的全部缺失父链（自 hive 根向下逐级）。
 * path 必须以 \REGISTRY\USER\GameVHD_<game_id> 开头（gvhd_reg_rewrite 产物）。 */
static NTSTATUS gvhd_reg_create_parents(const wchar_t *path)
{
    const struct gvhd_param_block *param = gvhd_get_param();
    size_t root_len;
    const wchar_t *cursor;
    NTSTATUS status;

    root_len = wcslen(GVHD_REG_HIVE_ROOT) + wcslen(param->game_id);
    if (path[root_len] == L'\0') {
        return GVHD_STATUS_SUCCESS;   /* 仅 hive 根：已挂载，无需创建 */
    }
    cursor = path + root_len;   /* 指向 '\' */
    while (*cursor == L'\\' && cursor[1] != L'\0') {
        const wchar_t *next = wcschr(cursor + 1, L'\\');

        if (next == NULL) {
            break;   /* 最后一段 = 目标键本身；其父已由上一轮创建 */
        }
        {
            wchar_t ancestor[GVHD_REG_PATH_MAX];
            size_t n = (size_t)(next - path);

            if (n >= GVHD_REG_PATH_MAX) {
                return GVHD_STATUS_OBJECT_PATH_NOT_FOUND;
            }
            memcpy(ancestor, path, n * sizeof(wchar_t));
            ancestor[n] = L'\0';
            status = gvhd_reg_open_or_create(ancestor);
            if (!NT_SUCCESS(status)) {
                return status;
            }
        }
        cursor = next;
    }
    return GVHD_STATUS_SUCCESS;
}

/* ================================================================ */
/* 句柄路径查询与删除保护（宿主零触碰）                                    */
/* ================================================================ */

/* 查询键句柄的完整 native 路径（KeyNameInformation）。
 * 成功 → STATUS_SUCCESS 且 buf 为 NUL 结尾路径。 */
static NTSTATUS gvhd_reg_handle_path(HANDLE hKey, wchar_t *buf, size_t cap)
{
    PKEY_NAME_INFORMATION kinf;
    ULONG need = 0;
    NTSTATUS status;
    size_t nchars;

    status = pfnOrig_NtQueryKey(hKey, KeyNameInformation, NULL, 0, &need);
    if (status != GVHD_STATUS_BUFFER_OVERFLOW) {
        return status;
    }
    kinf = (PKEY_NAME_INFORMATION)HeapAlloc(GetProcessHeap(), 0, need);
    if (kinf == NULL) {
        return GVHD_STATUS_INSUFFICIENT_RESOURCES;
    }
    status = pfnOrig_NtQueryKey(hKey, KeyNameInformation, kinf, need, &need);
    if (status == GVHD_STATUS_SUCCESS || status == GVHD_STATUS_BUFFER_OVERFLOW) {
        nchars = (size_t)(kinf->NameLength / sizeof(wchar_t));
        if (nchars >= cap) {
            nchars = cap - 1;
        }
        if (nchars > 0) {
            memcpy(buf, kinf->Name, nchars * sizeof(wchar_t));
        }
        buf[nchars] = L'\0';
        status = GVHD_STATUS_SUCCESS;
    }
    HeapFree(GetProcessHeap(), 0, kinf);
    return status;
}

/* 句柄路径是否已是本沙箱 hive 之下（打开/创建时已重写 → 直接原函数即可）。 */
static BOOLEAN gvhd_reg_is_hive_path(const wchar_t *path)
{
    const struct gvhd_param_block *param = gvhd_get_param();
    size_t prefix_len = wcslen(GVHD_REG_HIVE_ROOT);
    size_t game_len = wcslen(param->game_id);
    size_t root_len = prefix_len + game_len;

    if (wcsncmp(path, GVHD_REG_HIVE_ROOT, prefix_len) != 0) {
        return FALSE;
    }
    if (wcsncmp(path + prefix_len, param->game_id, game_len) != 0) {
        return FALSE;
    }
    return path[root_len] == L'\\';   /* 必须是 hive 根之下，而非根本身 */
}

/* 打开 hive 键并返回句柄（删除探测 / 删除重定向用），键不存在返回 FALSE。 */
static BOOLEAN gvhd_reg_probe_key(const wchar_t *hive_path)
{
    UNICODE_STRING name;
    OBJECT_ATTRIBUTES oa;
    HANDLE hKey = NULL;
    NTSTATUS status;

    name.Buffer = (PWSTR)hive_path;
    name.Length = (USHORT)(wcslen(hive_path) * sizeof(wchar_t));
    name.MaximumLength = (USHORT)(name.Length + sizeof(wchar_t));

    memset(&oa, 0, sizeof(oa));
    oa.Length = sizeof(OBJECT_ATTRIBUTES);
    oa.RootDirectory = NULL;
    oa.ObjectName = &name;
    oa.Attributes = OBJ_CASE_INSENSITIVE;

    status = pfnOrig_NtOpenKeyEx(&hKey, KEY_QUERY_VALUE, &oa, 0);
    if (hKey != NULL) {
        gvhd_reg_close(hKey);
    }
    return NT_SUCCESS(status);
}

/* 在 hive 副本上执行删除（替代宿主句柄上的删除，保证宿主零触碰）。 */
static NTSTATUS gvhd_reg_delete_hive_key(const wchar_t *hive_path, ULONG flags)
{
    UNICODE_STRING name;
    OBJECT_ATTRIBUTES oa;
    HANDLE hKey = NULL;
    NTSTATUS status;

    name.Buffer = (PWSTR)hive_path;
    name.Length = (USHORT)(wcslen(hive_path) * sizeof(wchar_t));
    name.MaximumLength = (USHORT)(name.Length + sizeof(wchar_t));

    memset(&oa, 0, sizeof(oa));
    oa.Length = sizeof(OBJECT_ATTRIBUTES);
    oa.RootDirectory = NULL;
    oa.ObjectName = &name;
    oa.Attributes = OBJ_CASE_INSENSITIVE;

    status = pfnOrig_NtOpenKeyEx(&hKey, DELETE, &oa, 0);
    if (NT_SUCCESS(status)) {
        if (flags != 0) {
            status = pfnOrig_NtDeleteKeyEx(hKey, flags);
        } else {
            status = pfnOrig_NtDeleteKey(hKey);
        }
    }
    if (hKey != NULL) {
        gvhd_reg_close(hKey);
    }
    return status;
}

/* 在 hive 副本上删除值（替代宿主句柄上的删值）。 */
static NTSTATUS gvhd_reg_delete_hive_value(const wchar_t *hive_path,
                                           PUNICODE_STRING ValueName)
{
    UNICODE_STRING name;
    OBJECT_ATTRIBUTES oa;
    HANDLE hKey = NULL;
    NTSTATUS status;

    name.Buffer = (PWSTR)hive_path;
    name.Length = (USHORT)(wcslen(hive_path) * sizeof(wchar_t));
    name.MaximumLength = (USHORT)(name.Length + sizeof(wchar_t));

    memset(&oa, 0, sizeof(oa));
    oa.Length = sizeof(OBJECT_ATTRIBUTES);
    oa.RootDirectory = NULL;
    oa.ObjectName = &name;
    oa.Attributes = OBJ_CASE_INSENSITIVE;

    status = pfnOrig_NtOpenKeyEx(&hKey, KEY_SET_VALUE, &oa, 0);
    if (NT_SUCCESS(status)) {
        status = pfnOrig_NtDeleteValueKey(hKey, ValueName);
    }
    if (hKey != NULL) {
        gvhd_reg_close(hKey);
    }
    return status;
}

/* ================================================================ */
/* 钩子：路径类打开（重写 + 读穿透）                                      */
/* ================================================================ */

static NTSTATUS NTAPI Hook_NtOpenKey(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes)
{
    struct gvhd_reg_rewritten rw;
    NTSTATUS status;

    gvhd_reg_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtOpenKey(KeyHandle, DesiredAccess, ObjectAttributes);
    }
    status = pfnOrig_NtOpenKey(KeyHandle, DesiredAccess, &rw.oa);
    if (status == GVHD_STATUS_OBJECT_NAME_NOT_FOUND ||
        status == GVHD_STATUS_OBJECT_PATH_NOT_FOUND ||
        status == GVHD_STATUS_ACCESS_DENIED) {
        gvhd_reg_log_readthrough(rw.orig_buf);
        return pfnOrig_NtOpenKey(KeyHandle, DesiredAccess, ObjectAttributes);
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtOpenKeyEx(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG OpenOptions)
{
    struct gvhd_reg_rewritten rw;
    NTSTATUS status;

    gvhd_reg_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtOpenKeyEx(KeyHandle, DesiredAccess, ObjectAttributes, OpenOptions);
    }
    status = pfnOrig_NtOpenKeyEx(KeyHandle, DesiredAccess, &rw.oa, OpenOptions);
    if (status == GVHD_STATUS_OBJECT_NAME_NOT_FOUND ||
        status == GVHD_STATUS_OBJECT_PATH_NOT_FOUND ||
        status == GVHD_STATUS_ACCESS_DENIED) {
        gvhd_reg_log_readthrough(rw.orig_buf);
        return pfnOrig_NtOpenKeyEx(KeyHandle, DesiredAccess, ObjectAttributes, OpenOptions);
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtOpenKeyTransacted(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    HANDLE TransactionHandle)
{
    struct gvhd_reg_rewritten rw;
    NTSTATUS status;

    gvhd_reg_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtOpenKeyTransacted(KeyHandle, DesiredAccess, ObjectAttributes,
                                           TransactionHandle);
    }
    status = pfnOrig_NtOpenKeyTransacted(KeyHandle, DesiredAccess, &rw.oa, TransactionHandle);
    if (status == GVHD_STATUS_OBJECT_NAME_NOT_FOUND ||
        status == GVHD_STATUS_OBJECT_PATH_NOT_FOUND ||
        status == GVHD_STATUS_ACCESS_DENIED) {
        gvhd_reg_log_readthrough(rw.orig_buf);
        return pfnOrig_NtOpenKeyTransacted(KeyHandle, DesiredAccess, ObjectAttributes,
                                           TransactionHandle);
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtOpenKeyTransactedEx(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG OpenOptions, HANDLE TransactionHandle)
{
    struct gvhd_reg_rewritten rw;
    NTSTATUS status;

    gvhd_reg_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtOpenKeyTransactedEx(KeyHandle, DesiredAccess, ObjectAttributes,
                                             OpenOptions, TransactionHandle);
    }
    status = pfnOrig_NtOpenKeyTransactedEx(KeyHandle, DesiredAccess, &rw.oa,
                                           OpenOptions, TransactionHandle);
    if (status == GVHD_STATUS_OBJECT_NAME_NOT_FOUND ||
        status == GVHD_STATUS_OBJECT_PATH_NOT_FOUND ||
        status == GVHD_STATUS_ACCESS_DENIED) {
        gvhd_reg_log_readthrough(rw.orig_buf);
        return pfnOrig_NtOpenKeyTransactedEx(KeyHandle, DesiredAccess, ObjectAttributes,
                                             OpenOptions, TransactionHandle);
    }
    return status;
}

/* ================================================================ */
/* 钩子：路径类创建（一律沙箱 + 父链自动创建）                              */
/* ================================================================ */

static NTSTATUS NTAPI Hook_NtCreateKey(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG TitleIndex, PUNICODE_STRING Class, ULONG CreateOptions, PULONG Disposition)
{
    struct gvhd_reg_rewritten rw;
    NTSTATUS status;

    gvhd_reg_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtCreateKey(KeyHandle, DesiredAccess, ObjectAttributes,
                                   TitleIndex, Class, CreateOptions, Disposition);
    }
    status = pfnOrig_NtCreateKey(KeyHandle, DesiredAccess, &rw.oa,
                                 TitleIndex, Class, CreateOptions, Disposition);
    if (status == GVHD_STATUS_OBJECT_PATH_NOT_FOUND ||
        status == GVHD_STATUS_OBJECT_NAME_NOT_FOUND) {
        gvhd_reg_create_parents(rw.name_buf);
        status = pfnOrig_NtCreateKey(KeyHandle, DesiredAccess, &rw.oa,
                                     TitleIndex, Class, CreateOptions, Disposition);
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtCreateKeyEx(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG TitleIndex, PUNICODE_STRING Class, ULONG CreateOptions,
    PULONG Disposition, PULONG ExtendedDisposition)
{
    struct gvhd_reg_rewritten rw;
    NTSTATUS status;

    gvhd_reg_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtCreateKeyEx(KeyHandle, DesiredAccess, ObjectAttributes,
                                     TitleIndex, Class, CreateOptions,
                                     Disposition, ExtendedDisposition);
    }
    status = pfnOrig_NtCreateKeyEx(KeyHandle, DesiredAccess, &rw.oa,
                                   TitleIndex, Class, CreateOptions,
                                   Disposition, ExtendedDisposition);
    if (status == GVHD_STATUS_OBJECT_PATH_NOT_FOUND ||
        status == GVHD_STATUS_OBJECT_NAME_NOT_FOUND) {
        gvhd_reg_create_parents(rw.name_buf);
        status = pfnOrig_NtCreateKeyEx(KeyHandle, DesiredAccess, &rw.oa,
                                       TitleIndex, Class, CreateOptions,
                                       Disposition, ExtendedDisposition);
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtCreateKeyTransacted(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG TitleIndex, PUNICODE_STRING Class, ULONG CreateOptions,
    HANDLE TransactionHandle, PULONG Disposition)
{
    struct gvhd_reg_rewritten rw;
    NTSTATUS status;

    gvhd_reg_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtCreateKeyTransacted(KeyHandle, DesiredAccess, ObjectAttributes,
                                             TitleIndex, Class, CreateOptions,
                                             TransactionHandle, Disposition);
    }
    status = pfnOrig_NtCreateKeyTransacted(KeyHandle, DesiredAccess, &rw.oa,
                                           TitleIndex, Class, CreateOptions,
                                           TransactionHandle, Disposition);
    if (status == GVHD_STATUS_OBJECT_PATH_NOT_FOUND ||
        status == GVHD_STATUS_OBJECT_NAME_NOT_FOUND) {
        gvhd_reg_create_parents(rw.name_buf);
        status = pfnOrig_NtCreateKeyTransacted(KeyHandle, DesiredAccess, &rw.oa,
                                               TitleIndex, Class, CreateOptions,
                                               TransactionHandle, Disposition);
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtCreateKeyTransactedEx(
    PHANDLE KeyHandle, ACCESS_MASK DesiredAccess, POBJECT_ATTRIBUTES ObjectAttributes,
    ULONG TitleIndex, PUNICODE_STRING Class, ULONG CreateOptions,
    HANDLE TransactionHandle, PULONG Disposition, PULONG ExtendedDisposition)
{
    struct gvhd_reg_rewritten rw;
    NTSTATUS status;

    gvhd_reg_eval_oa(ObjectAttributes, &rw);
    if (!rw.active) {
        return pfnOrig_NtCreateKeyTransactedEx(KeyHandle, DesiredAccess, ObjectAttributes,
                                               TitleIndex, Class, CreateOptions,
                                               TransactionHandle, Disposition,
                                               ExtendedDisposition);
    }
    status = pfnOrig_NtCreateKeyTransactedEx(KeyHandle, DesiredAccess, &rw.oa,
                                             TitleIndex, Class, CreateOptions,
                                             TransactionHandle, Disposition,
                                             ExtendedDisposition);
    if (status == GVHD_STATUS_OBJECT_PATH_NOT_FOUND ||
        status == GVHD_STATUS_OBJECT_NAME_NOT_FOUND) {
        gvhd_reg_create_parents(rw.name_buf);
        status = pfnOrig_NtCreateKeyTransactedEx(KeyHandle, DesiredAccess, &rw.oa,
                                                 TitleIndex, Class, CreateOptions,
                                                 TransactionHandle, Disposition,
                                                 ExtendedDisposition);
    }
    return status;
}

/* ================================================================ */
/* 钩子：删除（句柄级保护，宿主零触碰）                                    */
/* ================================================================ */

static NTSTATUS NTAPI Hook_NtDeleteKey(HANDLE KeyHandle)
{
    wchar_t path[GVHD_REG_PATH_MAX];
    wchar_t hive_path[GVHD_REG_PATH_MAX];
    NTSTATUS status;

    /* 常见路径：句柄来自重写后的打开/创建 → 已指向 hive → 直接原函数 */
    status = gvhd_reg_handle_path(KeyHandle, path, GVHD_REG_PATH_MAX);
    if (status != GVHD_STATUS_SUCCESS) {
        return pfnOrig_NtDeleteKey(KeyHandle);   /* 无法识别：放行原行为 */
    }
    if (gvhd_reg_is_hive_path(path)) {
        return pfnOrig_NtDeleteKey(KeyHandle);
    }
    /* 宿主句柄（读穿透场景）：保护宿主 */
    if (gvhd_reg_rewrite(path, hive_path, GVHD_REG_PATH_MAX) == NULL) {
        return pfnOrig_NtDeleteKey(KeyHandle);   /* 非 Software 根：宿主语义 */
    }
    if (!gvhd_reg_probe_key(hive_path)) {
        return GVHD_STATUS_OBJECT_NAME_NOT_FOUND;   /* hive 无此键 → 不动宿主 */
    }
    /* hive 有副本：删除 hive 副本而非宿主键 */
    return gvhd_reg_delete_hive_key(hive_path, 0);
}

static NTSTATUS NTAPI Hook_NtDeleteKeyEx(HANDLE KeyHandle, ULONG Flags)
{
    wchar_t path[GVHD_REG_PATH_MAX];
    wchar_t hive_path[GVHD_REG_PATH_MAX];
    NTSTATUS status;

    status = gvhd_reg_handle_path(KeyHandle, path, GVHD_REG_PATH_MAX);
    if (status != GVHD_STATUS_SUCCESS) {
        return pfnOrig_NtDeleteKeyEx(KeyHandle, Flags);
    }
    if (gvhd_reg_is_hive_path(path)) {
        return pfnOrig_NtDeleteKeyEx(KeyHandle, Flags);
    }
    if (gvhd_reg_rewrite(path, hive_path, GVHD_REG_PATH_MAX) == NULL) {
        return pfnOrig_NtDeleteKeyEx(KeyHandle, Flags);
    }
    if (!gvhd_reg_probe_key(hive_path)) {
        return GVHD_STATUS_OBJECT_NAME_NOT_FOUND;
    }
    return gvhd_reg_delete_hive_key(hive_path, Flags);
}

static NTSTATUS NTAPI Hook_NtDeleteValueKey(HANDLE KeyHandle, PUNICODE_STRING ValueName)
{
    wchar_t path[GVHD_REG_PATH_MAX];
    wchar_t hive_path[GVHD_REG_PATH_MAX];
    NTSTATUS status;

    status = gvhd_reg_handle_path(KeyHandle, path, GVHD_REG_PATH_MAX);
    if (status != GVHD_STATUS_SUCCESS) {
        return pfnOrig_NtDeleteValueKey(KeyHandle, ValueName);
    }
    if (gvhd_reg_is_hive_path(path)) {
        return pfnOrig_NtDeleteValueKey(KeyHandle, ValueName);
    }
    if (gvhd_reg_rewrite(path, hive_path, GVHD_REG_PATH_MAX) == NULL) {
        return pfnOrig_NtDeleteValueKey(KeyHandle, ValueName);
    }
    if (!gvhd_reg_probe_key(hive_path)) {
        return GVHD_STATUS_OBJECT_NAME_NOT_FOUND;   /* hive 无此键 → 不动宿主 */
    }
    /* hive 有副本：在 hive 副本上删值 */
    return gvhd_reg_delete_hive_value(hive_path, ValueName);
}

/* ================================================================ */
/* 钩子：句柄级操作（打开即重定向已保证句柄正确，直接透传）                   */
/* ================================================================ */

static NTSTATUS NTAPI Hook_NtSetValueKey(
    HANDLE KeyHandle, PUNICODE_STRING ValueName, ULONG TitleIndex, ULONG Type,
    PVOID Data, ULONG DataSize)
{
    return pfnOrig_NtSetValueKey(KeyHandle, ValueName, TitleIndex, Type, Data, DataSize);
}

static NTSTATUS NTAPI Hook_NtQueryValueKey(
    HANDLE KeyHandle, PUNICODE_STRING ValueName, KEY_VALUE_INFORMATION_CLASS Kvic,
    PVOID KeyValueInformation, ULONG Length, PULONG ResultLength)
{
    return pfnOrig_NtQueryValueKey(KeyHandle, ValueName, Kvic,
                                   KeyValueInformation, Length, ResultLength);
}

static NTSTATUS NTAPI Hook_NtQueryMultipleValueKey(
    HANDLE KeyHandle, PKEY_VALUE_ENTRY ValueEntries, ULONG EntryCount,
    PVOID ValueBuffer, PULONG BufferLength, PULONG RequiredBufferLength)
{
    return pfnOrig_NtQueryMultipleValueKey(KeyHandle, ValueEntries, EntryCount,
                                           ValueBuffer, BufferLength, RequiredBufferLength);
}

static NTSTATUS NTAPI Hook_NtEnumerateValueKey(
    HANDLE KeyHandle, ULONG Index, KEY_VALUE_INFORMATION_CLASS Kvic,
    PVOID KeyValueInformation, ULONG Length, PULONG ResultLength)
{
    return pfnOrig_NtEnumerateValueKey(KeyHandle, Index, Kvic,
                                       KeyValueInformation, Length, ResultLength);
}

static NTSTATUS NTAPI Hook_NtQueryKey(
    HANDLE KeyHandle, KEY_INFORMATION_CLASS Kic,
    PVOID KeyInformation, ULONG Length, PULONG ResultLength)
{
    return pfnOrig_NtQueryKey(KeyHandle, Kic, KeyInformation, Length, ResultLength);
}

static NTSTATUS NTAPI Hook_NtEnumerateKey(
    HANDLE KeyHandle, ULONG Index, KEY_INFORMATION_CLASS Kic,
    PVOID KeyInformation, ULONG Length, PULONG ResultLength)
{
    return pfnOrig_NtEnumerateKey(KeyHandle, Index, Kic,
                                  KeyInformation, Length, ResultLength);
}

static NTSTATUS NTAPI Hook_NtRenameKey(HANDLE KeyHandle, PUNICODE_STRING NewName)
{
    /* 已知局限：读穿透得到的宿主句柄上重命名会改宿主键名（MVP 不拦截） */
    return pfnOrig_NtRenameKey(KeyHandle, NewName);
}

static NTSTATUS NTAPI Hook_NtNotifyChangeKey(
    HANDLE KeyHandle, HANDLE Event, PIO_APC_ROUTINE ApcRoutine, PVOID ApcContext,
    PIO_STATUS_BLOCK IoStatusBlock, ULONG CompletionFilter, BOOLEAN WatchTree,
    PVOID Buffer, ULONG BufferSize, BOOLEAN Asynchronous)
{
    /* 已知局限：变更通知直通宿主，hive 变更与宿主通知互相不可见 */
    return pfnOrig_NtNotifyChangeKey(KeyHandle, Event, ApcRoutine, ApcContext,
                                     IoStatusBlock, CompletionFilter, WatchTree,
                                     Buffer, BufferSize, Asynchronous);
}

/* ================================================================ */
/* 安装入口（init.c 在 MH_Initialize 成功后调用）                         */
/* ================================================================ */

#define GVHD_REG_RESOLVE(FN)                                                    \
    do {                                                                        \
        __sys_##FN = (LPVOID)GetProcAddress(hNtdll, #FN);                       \
        if (__sys_##FN == NULL) {                                               \
            gvhd_log_write(L"REG_HOOK_RESOLVE_FAILED fn=%S", #FN);              \
            return GVHD_INIT_ERR_HOOK;                                          \
        }                                                                       \
    } while (0)

#define GVHD_REG_CREATE(FN)                                                     \
    do {                                                                        \
        mh = MH_CreateHook(__sys_##FN, (LPVOID)&Hook_##FN,                      \
                           (LPVOID *)&pfnOrig_##FN);                            \
        if (mh != MH_OK) {                                                      \
            gvhd_log_write(L"REG_HOOK_CREATE_FAILED fn=%S mh=%d", #FN, (int)mh);\
            return GVHD_INIT_ERR_HOOK;                                          \
        }                                                                       \
    } while (0)

#define GVHD_REG_ENABLE(FN)                                                     \
    do {                                                                        \
        mh = MH_EnableHook(__sys_##FN);                                         \
        if (mh != MH_OK) {                                                      \
            gvhd_log_write(L"REG_HOOK_ENABLE_FAILED fn=%S mh=%d", #FN, (int)mh);\
            return GVHD_INIT_ERR_HOOK;                                          \
        }                                                                       \
    } while (0)

uint32_t gvhd_install_registry_hooks(void)
{
    HMODULE hNtdll;
    MH_STATUS mh;

    /* 协议 §6.5：game_id 为空 → 不安装注册表钩子（直通宿主） */
    if (gvhd_get_param()->game_id[0] == L'\0') {
        gvhd_log_write(L"REG_HOOKS_SKIPPED reason=game_id_empty");
        return 0;
    }

    hNtdll = GetModuleHandleW(L"ntdll.dll");
    if (hNtdll == NULL) {
        return GVHD_INIT_ERR_HOOK;
    }

    /* 1. 解析全部符号（含 NtClose，句柄释放辅助） */
    GVHD_REG_RESOLVE(NtOpenKey);
    GVHD_REG_RESOLVE(NtOpenKeyEx);
    GVHD_REG_RESOLVE(NtOpenKeyTransacted);
    GVHD_REG_RESOLVE(NtOpenKeyTransactedEx);
    GVHD_REG_RESOLVE(NtCreateKey);
    GVHD_REG_RESOLVE(NtCreateKeyEx);
    GVHD_REG_RESOLVE(NtCreateKeyTransacted);
    GVHD_REG_RESOLVE(NtCreateKeyTransactedEx);
    GVHD_REG_RESOLVE(NtSetValueKey);
    GVHD_REG_RESOLVE(NtDeleteValueKey);
    GVHD_REG_RESOLVE(NtDeleteKey);
    GVHD_REG_RESOLVE(NtDeleteKeyEx);
    GVHD_REG_RESOLVE(NtQueryValueKey);
    GVHD_REG_RESOLVE(NtQueryMultipleValueKey);
    GVHD_REG_RESOLVE(NtEnumerateValueKey);
    GVHD_REG_RESOLVE(NtQueryKey);
    GVHD_REG_RESOLVE(NtEnumerateKey);
    GVHD_REG_RESOLVE(NtRenameKey);
    GVHD_REG_RESOLVE(NtNotifyChangeKey);
    __sys_NtClose = (LPVOID)GetProcAddress(hNtdll, "NtClose");
    if (__sys_NtClose == NULL) {
        gvhd_log_write(L"REG_HOOK_RESOLVE_FAILED fn=NtClose");
        return GVHD_INIT_ERR_HOOK;
    }

    /* 2. 创建全部钩子（先建全再启用，保证任何钩子触发时全部 pfnOrig 已就绪，
     *    删除钩子内部要用 pfnOrig_NtQueryKey / NtOpenKeyEx 做探测） */
    GVHD_REG_CREATE(NtOpenKey);
    GVHD_REG_CREATE(NtOpenKeyEx);
    GVHD_REG_CREATE(NtOpenKeyTransacted);
    GVHD_REG_CREATE(NtOpenKeyTransactedEx);
    GVHD_REG_CREATE(NtCreateKey);
    GVHD_REG_CREATE(NtCreateKeyEx);
    GVHD_REG_CREATE(NtCreateKeyTransacted);
    GVHD_REG_CREATE(NtCreateKeyTransactedEx);
    GVHD_REG_CREATE(NtSetValueKey);
    GVHD_REG_CREATE(NtDeleteValueKey);
    GVHD_REG_CREATE(NtDeleteKey);
    GVHD_REG_CREATE(NtDeleteKeyEx);
    GVHD_REG_CREATE(NtQueryValueKey);
    GVHD_REG_CREATE(NtQueryMultipleValueKey);
    GVHD_REG_CREATE(NtEnumerateValueKey);
    GVHD_REG_CREATE(NtQueryKey);
    GVHD_REG_CREATE(NtEnumerateKey);
    GVHD_REG_CREATE(NtRenameKey);
    GVHD_REG_CREATE(NtNotifyChangeKey);

    /* 3. 启用 */
    GVHD_REG_ENABLE(NtOpenKey);
    GVHD_REG_ENABLE(NtOpenKeyEx);
    GVHD_REG_ENABLE(NtOpenKeyTransacted);
    GVHD_REG_ENABLE(NtOpenKeyTransactedEx);
    GVHD_REG_ENABLE(NtCreateKey);
    GVHD_REG_ENABLE(NtCreateKeyEx);
    GVHD_REG_ENABLE(NtCreateKeyTransacted);
    GVHD_REG_ENABLE(NtCreateKeyTransactedEx);
    GVHD_REG_ENABLE(NtSetValueKey);
    GVHD_REG_ENABLE(NtDeleteValueKey);
    GVHD_REG_ENABLE(NtDeleteKey);
    GVHD_REG_ENABLE(NtDeleteKeyEx);
    GVHD_REG_ENABLE(NtQueryValueKey);
    GVHD_REG_ENABLE(NtQueryMultipleValueKey);
    GVHD_REG_ENABLE(NtEnumerateValueKey);
    GVHD_REG_ENABLE(NtQueryKey);
    GVHD_REG_ENABLE(NtEnumerateKey);
    GVHD_REG_ENABLE(NtRenameKey);
    GVHD_REG_ENABLE(NtNotifyChangeKey);

    return 0;
}
