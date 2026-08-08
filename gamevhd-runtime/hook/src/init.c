/*
 * gvhook — init.c
 *
 * 归属波次：W2T6（阶段 2 任务 6：注入与 hook 骨架）。
 *
 * 依据 docs/injection_protocol.md（单一事实源）§5 / §6 / §7 / §10 实现：
 *   - DllMain：PROCESS_ATTACH 仅返回 TRUE。一切初始化都在 gvhook_init() 中
 *     进行（协议 ADR 4：DllMain 在 loader lock 下执行，禁止干重活）。
 *   - gvhook_init()：注入入口。自检顺序（§10）：
 *       magic → version → 复制私有副本 → 打开日志 → MinHook 初始化 →
 *       安装钩子 → RULES_LOADED <n> → HOOK_DLL_PRESENT。
 *   - gvhd_log_write()：向参数块 log_path 指定的文件追加一行
 *       "[gvhook] YYYY-MM-DD HH:MM:SS.mmm <message>\r\n"（§7，UTF-8 编码）。
 *   - 参数块 + 规则表的进程内私有副本与访问器（§6.1）。
 */

#include <stdarg.h>
#include <stdio.h>
#include <stdint.h>

#include <windows.h>

#include "hook_common.h"
#include "rules.h"
#include "internal.h"
#include "MinHook.h"

/* ================================================================ */
/* 进程内私有副本（§6.1：编排可能在 gvhook_init 返回后释放远程区域）     */
/* ================================================================ */

static struct gvhd_param_block g_param;
static struct gvhd_rule_entry  g_rules[GVHD_RULE_MAX];
static uint32_t                g_rule_count = 0;

const struct gvhd_param_block *gvhd_get_param(void)
{
    return &g_param;
}

const struct gvhd_rule_entry *gvhd_get_rules(void)
{
    return g_rules;
}

uint32_t gvhd_get_rule_count(void)
{
    return g_rule_count;
}

DWORD gvhd_current_pid(void)
{
    return GetCurrentProcessId();
}

/* ================================================================ */
/* 日志（§7）                                                         */
/* ================================================================ */

#define GVHD_LOG_LINE_MAX 4096u   /* 单行消息上限（wchar 数） */
#define GVHD_TS_MAX       32u     /* 时间戳缓冲区 */

static HANDLE            g_log_file = INVALID_HANDLE_VALUE;
static CRITICAL_SECTION  g_log_lock;

/* 时间戳 "YYYY-MM-DD HH:MM:SS.mmm"（本地时间）。 */
static void gvhd_format_timestamp(wchar_t *buf, size_t buflen)
{
    SYSTEMTIME st;

    GetLocalTime(&st);
    _snwprintf(buf, buflen,
               L"%04u-%02u-%02u %02u:%02u:%02u.%03u",
               (unsigned)st.wYear, (unsigned)st.wMonth, (unsigned)st.wDay,
               (unsigned)st.wHour, (unsigned)st.wMinute, (unsigned)st.wSecond,
               (unsigned)st.wMilliseconds);
}

/* 以追加方式打开日志文件（FILE_APPEND_DATA：从不截断；FILE_SHARE_WRITE
 * 允许父子进程并发追加同一文件——每次 WriteFile 追加写是原子的）。 */
static uint32_t gvhd_log_open(const wchar_t *path)
{
    HANDLE h;

    if (path[0] == L'\0') {
        return GVHD_INIT_ERR_LOG;
    }
    h = CreateFileW(path, FILE_APPEND_DATA,
                    FILE_SHARE_READ | FILE_SHARE_WRITE,
                    NULL, OPEN_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    if (h == INVALID_HANDLE_VALUE) {
        return GVHD_INIT_ERR_LOG;
    }
    g_log_file = h;
    return GVHD_INIT_OK;
}

/* 追加一行日志；时间戳与换行在内部拼装。消息按 UTF-8 写入，保证
 * 断言脚本子串匹配的 marker 片段在文件中为字节原样（§7）。 */
void gvhd_log_write(const wchar_t *fmt, ...)
{
    wchar_t msg[GVHD_LOG_LINE_MAX];
    wchar_t line[GVHD_LOG_LINE_MAX + 96];
    char    utf8[GVHD_LOG_LINE_MAX * 3 + 256];
    va_list ap;
    int     n;
    int     u8len;
    DWORD   written = 0;

    if (g_log_file == INVALID_HANDLE_VALUE) {
        return;
    }

    va_start(ap, fmt);
    n = _vsnwprintf(msg, GVHD_LOG_LINE_MAX - 1, fmt, ap);
    va_end(ap);
    if (n < 0) {
        n = (int)(GVHD_LOG_LINE_MAX - 1);
    }
    msg[n] = L'\0';

    {
        wchar_t ts[GVHD_TS_MAX];
        gvhd_format_timestamp(ts, GVHD_TS_MAX);
        _snwprintf(line, GVHD_LOG_LINE_MAX + 95, L"[gvhook] %ls %ls\r\n", ts, msg);
    }

    u8len = WideCharToMultiByte(CP_UTF8, 0, line, -1,
                                utf8, (int)sizeof(utf8), NULL, NULL);
    if (u8len > 1) {
        EnterCriticalSection(&g_log_lock);
        WriteFile(g_log_file, utf8, (DWORD)(u8len - 1), &written, NULL);
        LeaveCriticalSection(&g_log_lock);
    }
}

/* ================================================================ */
/* DllMain：仅返回 TRUE（一切初始化在 gvhook_init，协议 ADR 4）          */
/* ================================================================ */

BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpvReserved)
{
    (void)hinstDLL;
    (void)fdwReason;
    (void)lpvReserved;
    return TRUE;
}

/* ================================================================ */
/* 注入入口 gvhook_init（协议 §5 步骤 7-8 / §10 自检顺序）               */
/* ================================================================ */

GVHD_API uint32_t GVHD_CALL gvhook_init(void *param_block)
{
    const struct gvhd_param_block *src =
        (const struct gvhd_param_block *)param_block;
    uint32_t rc;

    /* 1. magic */
    if (src == NULL || src->magic != GVHD_PARAM_MAGIC) {
        return GVHD_INIT_ERR_MAGIC;
    }
    /* 2. version */
    if (src->version != GVHD_PROTOCOL_VERSION) {
        return GVHD_INIT_ERR_VERSION;
    }
    /* 3. 复制参数块私有副本 */
    g_param = *src;
    g_rule_count = 0;

    /* 4. 打开日志（失败即返回，协议 ADR 8：日志打不开视为初始化失败） */
    rc = gvhd_log_open(g_param.log_path);
    if (rc != GVHD_INIT_OK) {
        return rc;
    }
    InitializeCriticalSection(&g_log_lock);

    /* 5. 复制规则表私有副本（偏移式规则表，无需地址翻译；§6.1） */
    if (g_param.rule_count > GVHD_RULE_MAX) {
        gvhd_log_write(L"INIT_WARN rule_count=%lu exceeds max %u; clamping to 0 rules",
                       (unsigned long)g_param.rule_count,
                       (unsigned)GVHD_RULE_MAX);
    } else if (g_param.rule_table_offset < sizeof(struct gvhd_param_block) ||
               (g_param.rule_table_offset & 3u) != 0) {
        gvhd_log_write(L"INIT_WARN invalid rule_table_offset=%lu; clamping to 0 rules",
                       (unsigned long)g_param.rule_table_offset);
    } else {
        const uint8_t *table =
            (const uint8_t *)param_block + g_param.rule_table_offset;
        for (uint32_t i = 0; i < g_param.rule_count; ++i) {
            g_rules[i] = ((const struct gvhd_rule_entry *)table)[i];
        }
        g_rule_count = g_param.rule_count;
    }

    /* 6. MinHook 初始化 */
    if (MH_Initialize() != MH_OK) {
        gvhd_log_write(L"INIT_FAILED code=%u reason=MH_Initialize",
                       (unsigned)GVHD_INIT_ERR_MINHOOK);
        return GVHD_INIT_ERR_MINHOOK;
    }

    /* 7. 安装钩子（proc.c：子进程注入；file.c：文件路径虚拟化；
     *    reg.c：注册表重定向） */
    if (gvhd_install_process_hooks() != 0) {
        gvhd_log_write(L"INIT_FAILED code=%u reason=gvhd_install_process_hooks",
                       (unsigned)GVHD_INIT_ERR_HOOK);
        return GVHD_INIT_ERR_HOOK;
    }
    if (gvhd_install_file_hooks() != 0) {
        gvhd_log_write(L"INIT_FAILED code=%u reason=gvhd_install_file_hooks",
                       (unsigned)GVHD_INIT_ERR_HOOK);
        return GVHD_INIT_ERR_HOOK;
    }
    if (gvhd_install_registry_hooks() != 0) {
        gvhd_log_write(L"INIT_FAILED code=%u reason=gvhd_install_registry_hooks",
                       (unsigned)GVHD_INIT_ERR_HOOK);
        return GVHD_INIT_ERR_HOOK;
    }

    /* 8. 自检 marker（§7 断言脚本依赖的精确片段） */
    gvhd_log_write(L"%S %u", GVHD_MARKER_RULES_LOADED,
                   (unsigned)g_rule_count);
    gvhd_log_write(L"%S", GVHD_MARKER_PRESENT);

    return GVHD_INIT_OK;
}
