import { describe, expect, it } from 'vitest'
import {
  SAVE_PATH_BASE_OPTIONS,
  joinSavePathTemplate,
  normalizeWindowsPath,
  splitSavePathTemplate,
} from './save-path-template'

describe('SAVE_PATH_BASE_OPTIONS', () => {
  it('包含五个基础变量与 custom 项，key 稳定', () => {
    expect(SAVE_PATH_BASE_OPTIONS).toEqual([
      { key: 'mydocuments', label: '我的文档 (C:\\Users\\你的用户名\\Documents)', value: '%USERPROFILE%\\Documents' },
      { key: 'userprofile', label: '用户家目录 (C:\\Users\\你的用户名)', value: '%USERPROFILE%' },
      { key: 'appdata', label: '应用数据-漫游 (C:\\Users\\你的用户名\\AppData\\Roaming)', value: '%APPDATA%' },
      { key: 'localappdata', label: '应用数据-本地 (C:\\Users\\你的用户名\\AppData\\Local)', value: '%LOCALAPPDATA%' },
      { key: 'game_drive', label: '游戏盘符 (挂载后的游戏盘)', value: '%GAME_DRIVE%' },
      { key: 'custom', label: '自定义路径', value: '' },
    ])
  })
})

describe('normalizeWindowsPath', () => {
  it('把 / 转成 \\ 并去除首尾空白', () => {
    expect(normalizeWindowsPath('  c:/xboxgame  ')).toBe('c:\\xboxgame')
  })

  it('合并重复反斜杠', () => {
    expect(normalizeWindowsPath('C:\\\\Users\\\\foo')).toBe('C:\\Users\\foo')
    expect(normalizeWindowsPath('C:/Users//foo')).toBe('C:\\Users\\foo')
  })

  it('保留 %VAR% 占位符', () => {
    expect(normalizeWindowsPath(' %USERPROFILE%\\Documents//My Games ')).toBe(
      '%USERPROFILE%\\Documents\\My Games',
    )
  })
})

describe('joinSavePathTemplate', () => {
  it('基础变量 + 子路径拼接', () => {
    expect(
      joinSavePathTemplate({
        mode: 'base',
        baseKey: 'userprofile',
        subpath: 'Documents\\My Games\\GT4\\SaveGame',
      }),
    ).toBe('%USERPROFILE%\\Documents\\My Games\\GT4\\SaveGame')
  })

  it('子路径为空只存基础变量', () => {
    expect(joinSavePathTemplate({ mode: 'base', baseKey: 'appdata', subpath: '' })).toBe('%APPDATA%')
    expect(joinSavePathTemplate({ mode: 'base', baseKey: 'appdata', subpath: '   ' })).toBe('%APPDATA%')
  })

  it('custom 模式原样存', () => {
    expect(joinSavePathTemplate({ mode: 'custom', baseKey: 'custom', subpath: 'D:\\Games\\xxx' })).toBe(
      'D:\\Games\\xxx',
    )
  })

  it('custom 模式同样做斜杠规范化', () => {
    expect(joinSavePathTemplate({ mode: 'custom', baseKey: 'custom', subpath: 'c:/xboxgame' })).toBe(
      'c:\\xboxgame',
    )
  })

  it('base 子路径含 / 时规范化为 \\', () => {
    expect(
      joinSavePathTemplate({ mode: 'base', baseKey: 'userprofile', subpath: 'Documents/My Games' }),
    ).toBe('%USERPROFILE%\\Documents\\My Games')
  })

  it('我的文档选项拼接完整模板', () => {
    expect(
      joinSavePathTemplate({ mode: 'base', baseKey: 'mydocuments', subpath: 'My Games\\GT4\\SaveGame' }),
    ).toBe('%USERPROFILE%\\Documents\\My Games\\GT4\\SaveGame')
  })
})

describe('splitSavePathTemplate', () => {
  it('识别基础变量前缀并拆分出子路径', () => {
    expect(splitSavePathTemplate('%APPDATA%\\My Games\\GT4')).toEqual({
      mode: 'base',
      baseKey: 'appdata',
      subpath: 'My Games\\GT4',
    })
  })

  it('恰好等于基础变量时子路径为空', () => {
    expect(splitSavePathTemplate('%GAME_DRIVE%')).toEqual({
      mode: 'base',
      baseKey: 'game_drive',
      subpath: '',
    })
  })

  it('前缀比较大小写不敏感', () => {
    expect(splitSavePathTemplate('%userprofile%\\save')).toEqual({
      mode: 'base',
      baseKey: 'userprofile',
      subpath: 'save',
    })
  })

  it('我的文档长前缀优先于用户目录匹配', () => {
    expect(splitSavePathTemplate('%USERPROFILE%\\Documents\\My Games\\GT4')).toEqual({
      mode: 'base',
      baseKey: 'mydocuments',
      subpath: 'My Games\\GT4',
    })
  })

  it('非 Documents 前缀仍归属用户目录', () => {
    expect(splitSavePathTemplate("%USERPROFILE%\\Sid Meier's Civilization 5")).toEqual({
      mode: 'base',
      baseKey: 'userprofile',
      subpath: "Sid Meier's Civilization 5",
    })
  })

  it('我的文档无子路径时仅存变量前缀', () => {
    expect(splitSavePathTemplate('%USERPROFILE%\\Documents')).toEqual({
      mode: 'base',
      baseKey: 'mydocuments',
      subpath: '',
    })
  })

  it('未知前缀视为 custom', () => {
    expect(splitSavePathTemplate('D:\\Games\\xxx')).toEqual({
      mode: 'custom',
      baseKey: 'custom',
      subpath: 'D:\\Games\\xxx',
    })
  })

  it('空模板默认选中用户目录', () => {
    expect(splitSavePathTemplate('')).toEqual({ mode: 'base', baseKey: 'userprofile', subpath: '' })
    expect(splitSavePathTemplate('   ')).toEqual({ mode: 'base', baseKey: 'userprofile', subpath: '' })
  })

  it('split 与 join 往返一致', () => {
    const parts = splitSavePathTemplate('%LOCALAPPDATA%\\My Games\\GT4\\SaveGame')
    expect(joinSavePathTemplate(parts)).toBe('%LOCALAPPDATA%\\My Games\\GT4\\SaveGame')
  })
})
