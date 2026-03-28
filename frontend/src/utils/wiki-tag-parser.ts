export type WikiTagGroupKey = 'genre' | 'subgenre' | 'perspective' | 'theme'

interface ExtractedWikiTagCandidate {
  value: string
  sourceLabel: string
  groupKey?: WikiTagGroupKey
}

// Wiki 标签字段遵循写作指南中的四分类：
// - 题材：作品大类
// - 子类型：玩法结构或产品形态
// - 视角：镜头 / 观察方式
// - 内容属性：叙事题材与气质
//
// 这里刻意按“可复用短标签”解析，而不是把整句自然语言描述直接当成一个标签。
// 例如“单人 Boss 战导向动作冒险”不应作为单一标签长期沉淀，
// 更适合在文档层拆成“动作冒险”“Boss 战导向”等可复用项。
const TYPE_LINE_PATTERN = /^(?:[-*+]\s*)?(?:\*\*)?\s*类型(?:标签)?\s*(?:\*\*)?\s*[:：]\s*(.+)$/i
const GROUP_LINE_PATTERNS: Array<{
  pattern: RegExp
  sourceLabel: string
  groupKey: WikiTagGroupKey
}> = [
  {
    pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*题材\s*(?:\*\*)?\s*[:：]\s*(.+)$/i,
    sourceLabel: '题材',
    groupKey: 'genre',
  },
  {
    pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*子类型\s*(?:\*\*)?\s*[:：]\s*(.+)$/i,
    sourceLabel: '子类型',
    groupKey: 'subgenre',
  },
  {
    pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*视角\s*(?:\*\*)?\s*[:：]\s*(.+)$/i,
    sourceLabel: '视角',
    groupKey: 'perspective',
  },
  {
    pattern: /^(?:[-*+]\s*)?(?:\*\*)?\s*内容属性\s*(?:\*\*)?\s*[:：]\s*(.+)$/i,
    sourceLabel: '内容属性',
    groupKey: 'theme',
  },
]

const normalizeTagText = (value: string) => {
  return value
    .replace(/[（）]/g, (char) => (char === '（' ? '(' : ')'))
    .replace(/\s+/g, ' ')
    .replace(/\s*\/\s*/g, ' / ')
    .trim()
}

const splitTypeValues = (value: string) => {
  return value
    .split(/\s*[、，,]\s*|\s*\/\s*/g)
    .map((item) => normalizeTagText(item))
    .filter((item) => item.length > 0)
}

export const extractWikiTagCandidates = (content?: string | null): ExtractedWikiTagCandidate[] => {
  if (!content) return []

  const items: ExtractedWikiTagCandidate[] = []
  const seen = new Set<string>()
  const lines = content.split(/\r?\n/)

  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line) continue

    let matched = line.match(TYPE_LINE_PATTERN)
    if (matched) {
      const tokens = splitTypeValues(matched[1] || '')
      for (const token of tokens) {
        const key = `type:${token.toLowerCase()}`
        if (!token || seen.has(key)) continue
        seen.add(key)
        items.push({
          value: token,
          sourceLabel: '类型',
        })
      }
      continue
    }

    for (const rule of GROUP_LINE_PATTERNS) {
      matched = line.match(rule.pattern)
      if (!matched) continue

      const tokens = splitTypeValues(matched[1] || '')
      for (const token of tokens) {
        const key = `${rule.groupKey}:${token.toLowerCase()}`
        if (!token || seen.has(key)) continue
        seen.add(key)
        items.push({
          value: token,
          sourceLabel: rule.sourceLabel,
          groupKey: rule.groupKey,
        })
      }
      break
    }
  }

  return items
}
