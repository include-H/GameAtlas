export type WikiTagGroupKey = 'genre' | 'subgenre' | 'perspective' | 'theme'

export interface ParsedWikiTags {
  genre: string[]
  subgenre: string[]
  perspective: string[]
  theme: string[]
}

export interface ExtractedWikiTagCandidate {
  value: string
  sourceLabel: string
}

const emptyParsedWikiTags = (): ParsedWikiTags => ({
  genre: [],
  subgenre: [],
  perspective: [],
  theme: [],
})

const TYPE_LINE_PATTERN = /^(?:[-*+]\s*)?(?:\*\*)?\s*类型(?:标签)?\s*(?:\*\*)?\s*[:：]\s*(.+)$/i

const PERSPECTIVE_KEYWORDS = [
  '第一人称',
  '第三人称',
  '俯视',
  '上帝视角',
  '横版',
  '侧视',
  '卷轴',
  '等角',
  '2.5d',
  '固定视角',
]

const THEME_KEYWORDS = [
  '恋爱',
  '喜剧',
  '校园',
  '悬疑',
  '恐怖',
  '科幻',
  '奇幻',
  '百合',
  '女性向',
  '乙女',
]

const SUBGENRE_RULES: Array<{ pattern: RegExp; tags: Partial<ParsedWikiTags> }> = [
  {
    pattern: /^动作角色扮演$/i,
    tags: {
      genre: ['角色扮演'],
      subgenre: ['动作角色扮演'],
    },
  },
  {
    pattern: /^恋爱冒险$/i,
    tags: {
      genre: ['冒险'],
      subgenre: ['恋爱冒险'],
      theme: ['恋爱'],
    },
  },
  {
    pattern: /^视觉小说$/i,
    tags: {
      genre: ['视觉小说'],
    },
  },
  {
    pattern: /^(?:主线推进型\s*)?adv$/i,
    tags: {
      subgenre: ['主线推进型 ADV'],
    },
  },
  {
    pattern: /^即时战略$/i,
    tags: {
      genre: ['策略'],
      subgenre: ['即时战略'],
    },
  },
  {
    pattern: /^即时战术$/i,
    tags: {
      genre: ['策略'],
      subgenre: ['即时战术'],
    },
  },
  {
    pattern: /^空战$/i,
    tags: {
      subgenre: ['空战'],
    },
  },
  {
    pattern: /^地面战混合$/i,
    tags: {
      subgenre: ['地面战混合'],
    },
  },
]

const normalizeTagText = (value: string) => {
  return value
    .replace(/[（）]/g, (char) => (char === '（' ? '(' : ')'))
    .replace(/\s+/g, ' ')
    .replace(/\s*\/\s*/g, ' / ')
    .trim()
}

const pushUnique = (target: string[], value: string) => {
  if (!value) return
  if (target.some((item) => item.toLowerCase() === value.toLowerCase())) return
  target.push(value)
}

const appendTags = (target: ParsedWikiTags, next: Partial<ParsedWikiTags>) => {
  for (const value of next.genre || []) pushUnique(target.genre, value)
  for (const value of next.subgenre || []) pushUnique(target.subgenre, value)
  for (const value of next.perspective || []) pushUnique(target.perspective, value)
  for (const value of next.theme || []) pushUnique(target.theme, value)
}

const classifySingleToken = (token: string): Partial<ParsedWikiTags> => {
  const normalized = normalizeTagText(token)
  if (!normalized) return {}

  for (const rule of SUBGENRE_RULES) {
    if (rule.pattern.test(normalized)) {
      return rule.tags
    }
  }

  const lower = normalized.toLowerCase()

  if (PERSPECTIVE_KEYWORDS.some((keyword) => lower.includes(keyword))) {
    return { perspective: [normalized] }
  }

  if (THEME_KEYWORDS.some((keyword) => normalized.includes(keyword))) {
    return { theme: [normalized] }
  }

  return { subgenre: [normalized] }
}

const splitTypeValues = (value: string) => {
  return value
    .split(/\s*[、，,]\s*|\s*\/\s*/g)
    .map((item) => normalizeTagText(item))
    .filter((item) => item.length > 0)
}

export const parseWikiTypeTags = (content?: string | null): ParsedWikiTags => {
  const result = emptyParsedWikiTags()
  if (!content) return result

  const lines = content.split(/\r?\n/)

  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line) continue

    const matched = line.match(TYPE_LINE_PATTERN)
    if (!matched) continue

    const tokens = splitTypeValues(matched[1] || '')
    for (const token of tokens) {
      appendTags(result, classifySingleToken(token))
    }
  }

  return result
}

export const extractWikiTagCandidates = (content?: string | null): ExtractedWikiTagCandidate[] => {
  if (!content) return []

  const items: ExtractedWikiTagCandidate[] = []
  const seen = new Set<string>()
  const lines = content.split(/\r?\n/)

  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line) continue

    const matched = line.match(TYPE_LINE_PATTERN)
    if (!matched) continue

    const tokens = splitTypeValues(matched[1] || '')
    for (const token of tokens) {
      const key = token.toLowerCase()
      if (!token || seen.has(key)) continue
      seen.add(key)
      items.push({
        value: token,
        sourceLabel: '类型',
      })
    }
  }

  return items
}
