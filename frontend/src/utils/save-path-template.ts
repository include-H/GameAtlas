// 存档目录（Windows 模板）的拆分 / 拼接 / 规范化工具。
// 存库值始终是模板字符串（如 `%USERPROFILE%\Documents\My Games\GT4\SaveGame`），
// 由后端启动脚本在运行时替换 %VAR% 占位符，因此这里只做字符串层面处理。

export interface SavePathBaseOption {
  /** 稳定标识，用于下拉框内部状态（v-model 用），不随文案变化。 */
  key: string
  label: string
  /** 模板变量本身；custom 为空字符串。 */
  value: string
}

export const SAVE_PATH_BASE_OPTIONS: SavePathBaseOption[] = [
  { key: 'mydocuments', label: '我的文档 (C:\\Users\\你的用户名\\Documents)', value: '%USERPROFILE%\\Documents' },
  { key: 'userprofile', label: '用户家目录 (C:\\Users\\你的用户名)', value: '%USERPROFILE%' },
  { key: 'appdata', label: '应用数据-漫游 (C:\\Users\\你的用户名\\AppData\\Roaming)', value: '%APPDATA%' },
  { key: 'localappdata', label: '应用数据-本地 (C:\\Users\\你的用户名\\AppData\\Local)', value: '%LOCALAPPDATA%' },
  { key: 'game_drive', label: '游戏盘符 (挂载后的游戏盘)', value: '%GAME_DRIVE%' },
  { key: 'custom', label: '自定义路径', value: '' },
]

export type SavePathTemplateMode = 'base' | 'custom'

export interface SavePathTemplateParts {
  mode: SavePathTemplateMode
  baseKey: string
  subpath: string
}

/**
 * 把 Windows 路径统一成反斜杠风格：`/` 转 `\`、去首尾空白、合并连续反斜杠。
 * `%VAR%` 占位符保持不变。
 */
export const normalizeWindowsPath = (path: string): string =>
  path.replace(/\//g, '\\').trim().replace(/\\{2,}/g, '\\')

/**
 * 把模板字符串拆成「基础目录 + 子路径」。
 * - 以某个基础变量的 `变量\` 前缀开头（大小写不敏感）→ base 模式；
 * - 恰好等于某个基础变量（无子路径）→ base 模式，subpath 为空；
 * - 其余一律视为 custom；
 * - 空模板默认选中用户目录。
 */
export const splitSavePathTemplate = (template: string): SavePathTemplateParts => {
  const trimmed = template.trim()
  if (!trimmed) {
    return { mode: 'base', baseKey: 'userprofile', subpath: '' }
  }

  const lowerTemplate = trimmed.toLowerCase()
  // 长 value 优先匹配：%USERPROFILE%\Documents 必须先于 %USERPROFILE% 命中
  const baseOptions = SAVE_PATH_BASE_OPTIONS.filter((option) => option.value !== '')
    .sort((a, b) => b.value.length - a.value.length)

  for (const option of baseOptions) {
    const lowerValue = option.value.toLowerCase()
    if (lowerTemplate === lowerValue) {
      return { mode: 'base', baseKey: option.key, subpath: '' }
    }
    const prefix = `${lowerValue}\\`
    if (lowerTemplate.startsWith(prefix)) {
      return { mode: 'base', baseKey: option.key, subpath: trimmed.slice(prefix.length) }
    }
  }

  return { mode: 'custom', baseKey: 'custom', subpath: trimmed }
}

/**
 * 把「基础目录 + 子路径」拼回模板字符串。
 * - custom：直接返回规范化后的 subpath（即完整模板）；
 * - base：子路径为空只返回基础变量，否则 `变量\子路径`。
 */
export const joinSavePathTemplate = (parts: SavePathTemplateParts): string => {
  if (parts.mode === 'custom') {
    return normalizeWindowsPath(parts.subpath)
  }

  const option = SAVE_PATH_BASE_OPTIONS.find(
    (candidate) => candidate.key === parts.baseKey && candidate.value !== '',
  )
  if (!option) {
    return normalizeWindowsPath(parts.subpath)
  }

  const subpath = normalizeWindowsPath(parts.subpath)
  if (!subpath) {
    return option.value
  }
  return `${option.value}\\${subpath}`
}
