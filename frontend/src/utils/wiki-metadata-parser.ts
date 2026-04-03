export interface WikiMetadataExtraction {
  summary: string
  releaseDate: string
  englishTitleAlt: string
  chineseTitleAlt: string
  engine: string
  developers: string[]
  publishers: string[]
  platforms: string[]
}

const INLINE_FIELD_RULES: Array<{
  key: 'summary' | 'releaseDate' | 'englishTitleAlt' | 'chineseTitleAlt' | 'engine' | 'developers' | 'publishers' | 'platforms'
  pattern: RegExp
  multiValue?: boolean
}> = [
  { key: 'summary', pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*(?:简介|概述|摘要|游戏简介|作品简介|内容简介)\s*(?:\*\*)?\s*[:：]\s*(.+)$/i },
  { key: 'releaseDate', pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*(?:首发日期|发售日期|发行日期|上市日期|发售时间|发行时间)\s*(?:\*\*)?\s*[:：]\s*(.+)$/i },
  { key: 'englishTitleAlt', pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*(?:英文常见译名|英文名|英文译名|外文名|原名)\s*(?:\*\*)?\s*[:：]\s*(.+)$/i, multiValue: true },
  { key: 'chineseTitleAlt', pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*(?:中文常见译名|常见译名|别名)\s*(?:\*\*)?\s*[:：]\s*(.+)$/i, multiValue: true },
  { key: 'engine', pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*(?:游戏引擎|开发引擎|引擎)\s*(?:\*\*)?\s*[:：]\s*(.+)$/i },
  { key: 'developers', pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*(?:开发商|开发者|研发商|开发公司|制作公司|开发团队)\s*(?:\*\*)?\s*[:：]\s*(.+)$/i, multiValue: true },
  { key: 'publishers', pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*(?:发行商|发行者|发行公司)\s*(?:\*\*)?\s*[:：]\s*(.+)$/i, multiValue: true },
  { key: 'platforms', pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*(?:平台|首发平台|已确认平台|登陆平台|登录平台|发售平台)\s*(?:\*\*)?\s*[:：]\s*(.+)$/i, multiValue: true },
]

const BLOCK_FIELD_RULES: Array<{
  key: 'summary'
  headingPattern: RegExp
}> = [
  { key: 'summary', headingPattern: /^(?:#{1,6}\s*|\*\*)\s*(?:简介|概述|摘要|游戏简介|作品简介|内容简介)\s*(?:\*\*)?\s*$/i },
]

const NEXT_FIELD_PATTERN = /^(?:[-*+]\s*)?(?:\*\*)?\s*(?:简介|概述|摘要|游戏简介|作品简介|内容简介|发售日期|发行日期|上市日期|发售时间|发行时间|开发商|开发者|研发商|开发公司|制作公司|开发团队|发行商|发行者|发行公司|平台|登陆平台|登录平台|发售平台|类型|题材|子类型|视角|内容属性)\s*(?:\*\*)?\s*[:：]?/i

const normalizeInlineMarkdown = (value: string) => {
  return value
    .replace(/!\[[^\]]*]\([^)]+\)/g, ' ')
    .replace(/\[([^\]]+)]\([^)]+\)/g, '$1')
    .replace(/[*_~`>#]/g, ' ')
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/p>/gi, '\n\n')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/gi, ' ')
    .replace(/\u00a0/g, ' ')
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n{3,}/g, '\n\n')
    .replace(/[ \t]{2,}/g, ' ')
    .trim()
}

const normalizeToken = (value: string) => {
  return normalizeInlineMarkdown(value)
    .replace(/[（）]/g, (char) => (char === '（' ? '(' : ')'))
    .replace(/[。；，,]+$/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

const stripPlatformNarrativePrefixes = (value: string) => {
  let normalized = normalizeToken(value)

  const prefixPatterns = [
    /^截至目前公开主流资料显示[，,、:\s]*/i,
    /^根据目前公开资料[，,、:\s]*/i,
    /^根据公开资料[，,、:\s]*/i,
    /^公开资料显示[，,、:\s]*/i,
  ]

  for (const pattern of prefixPatterns) {
    normalized = normalized.replace(pattern, '')
  }

  const leadInPatterns = [
    /^本作已确认的?(?:主要)?平台(?:为|有)\s*/i,
    /^已确认的?(?:主要)?平台(?:为|有)\s*/i,
    /^本作(?:主要)?平台(?:为|有)\s*/i,
    /^主要平台(?:为|有)\s*/i,
    /^平台(?:为|有)\s*/i,
    /^登陆平台(?:为|有)\s*/i,
    /^登录平台(?:为|有)\s*/i,
    /^发售平台(?:为|有)\s*/i,
  ]

  for (const pattern of leadInPatterns) {
    normalized = normalized.replace(pattern, '')
  }

  return normalized
}

const splitMultiValue = (value: string) => {
  return value
    .split(/\s*[、，,;/／|]\s*|\s+\+\s+/g)
    .map((item) => normalizeToken(item))
    .filter(Boolean)
}

const splitPlatformValues = (value: string) => {
  const normalized = stripPlatformNarrativePrefixes(value)
  return normalized
    .split(/\s*[、，,;/／|]\s*|\s+\+\s+|\s+\/\s+/g)
    .map((item) => normalizeToken(item))
    .filter(Boolean)
}

const normalizeDate = (value: string) => {
  const raw = normalizeToken(value)
  if (!raw) return ''

  const chinese = raw.match(/(\d{4})\s*年\s*(\d{1,2})\s*月\s*(\d{1,2})\s*日/)
  if (chinese) {
    return `${chinese[1]}-${chinese[2].padStart(2, '0')}-${chinese[3].padStart(2, '0')}`
  }

  const iso = raw.match(/(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})/)
  if (iso) {
    return `${iso[1]}-${iso[2].padStart(2, '0')}-${iso[3].padStart(2, '0')}`
  }

  // Some Wiki entries only provide year + month, for example "2007 年 8 月".
  // We intentionally coerce those partial dates to the first day of the month
  // so downstream release_date consumers can still sort and render them.
  // This is a precision-losing fallback, not a claim that the real release day is known.
  const chineseYearMonth = raw.match(/(\d{4})\s*年\s*(\d{1,2})\s*月/)
  if (chineseYearMonth) {
    return `${chineseYearMonth[1]}-${chineseYearMonth[2].padStart(2, '0')}-01`
  }

  const isoYearMonth = raw.match(/(\d{4})[-/.](\d{1,2})(?:$|[^\d])/)
  if (isoYearMonth) {
    return `${isoYearMonth[1]}-${isoYearMonth[2].padStart(2, '0')}-01`
  }

  return ''
}

const inferDateFromNarrative = (value: string) => {
  const raw = normalizeInlineMarkdown(value)
  if (!raw) return ''

  const patterns = [
    /于\s*(\d{4})\s*年\s*(\d{1,2})\s*月\s*(\d{1,2})\s*日(?:[^，。；\n]*)?(?:发售|推出|上线|登陆|登录)/,
    /(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})(?:[^，。；\n]*)?(?:发售|推出|上线|登陆|登录)/,
  ]

  for (const pattern of patterns) {
    const matched = raw.match(pattern)
    if (!matched) continue
    return `${matched[1]}-${matched[2].padStart(2, '0')}-${matched[3].padStart(2, '0')}`
  }

  return ''
}

const pushUnique = (target: string[], values: string[]) => {
  const seen = new Set(target.map((item) => item.toLowerCase()))
  for (const value of values) {
    const normalized = normalizeToken(value)
    if (!normalized) continue
    const key = normalized.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    target.push(normalized)
  }
}

const mergeTitleAlt = (current: string, incoming: string[]) => {
  const existing = current
    .split(/\s*\/\s*/g)
    .map((item) => normalizeToken(item))
    .filter(Boolean)
  const merged = [...existing]
  pushUnique(merged, incoming)
  return merged.join(' / ')
}

const extractBlockValue = (lines: string[], startIndex: number) => {
  const buffer: string[] = []

  for (let index = startIndex + 1; index < lines.length; index += 1) {
    const trimmed = lines[index].trim()
    if (!trimmed) {
      if (buffer.length > 0) break
      continue
    }
    if (NEXT_FIELD_PATTERN.test(trimmed) || /^(?:#{1,6}\s+)/.test(trimmed)) {
      break
    }
    buffer.push(trimmed.replace(/^[-*+]\s*/, ''))
  }

  return normalizeInlineMarkdown(buffer.join('\n'))
}

export const extractWikiMetadata = (content?: string | null): WikiMetadataExtraction => {
  const result: WikiMetadataExtraction = {
    summary: '',
    releaseDate: '',
    englishTitleAlt: '',
    chineseTitleAlt: '',
    engine: '',
    developers: [],
    publishers: [],
    platforms: [],
  }

  if (!content) return result

  result.releaseDate = inferDateFromNarrative(content)

  const lines = content.split(/\r?\n/)

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index].trim()
    if (!line) continue

    for (const rule of INLINE_FIELD_RULES) {
      const matched = line.match(rule.pattern)
      if (!matched) continue

      const value = matched[1] || ''
      switch (rule.key) {
        case 'summary':
          if (!result.summary) {
            result.summary = normalizeInlineMarkdown(value)
          }
          break
        case 'releaseDate':
          {
            const normalizedDate = normalizeDate(value)
            if (normalizedDate) {
              result.releaseDate = normalizedDate
            }
          }
          break
        case 'developers':
          pushUnique(result.developers, splitMultiValue(value))
          break
        case 'englishTitleAlt':
          result.englishTitleAlt = mergeTitleAlt(result.englishTitleAlt, splitMultiValue(value))
          break
        case 'chineseTitleAlt':
          result.chineseTitleAlt = mergeTitleAlt(result.chineseTitleAlt, splitMultiValue(value))
          break
        case 'engine':
          if (!result.engine) {
            result.engine = normalizeToken(value)
          }
          break
        case 'publishers':
          pushUnique(result.publishers, splitMultiValue(value))
          break
        case 'platforms':
          pushUnique(result.platforms, splitPlatformValues(value))
          break
      }
      break
    }

    for (const rule of BLOCK_FIELD_RULES) {
      if (result.summary) continue
      if (!rule.headingPattern.test(line)) continue

      const value = extractBlockValue(lines, index)
      if (!value) continue

      result.summary = value
      break
    }
  }

  return result
}
