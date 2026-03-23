export type WikiTagGroupKey = 'genre' | 'subgenre' | 'perspective' | 'theme'

interface ExtractedWikiTagCandidate {
  value: string
  sourceLabel: string
}

const TYPE_LINE_PATTERN = /^(?:[-*+]\s*)?(?:\*\*)?\s*类型(?:标签)?\s*(?:\*\*)?\s*[:：]\s*(.+)$/i

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
