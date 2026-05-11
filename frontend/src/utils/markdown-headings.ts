export interface HeadingEntry {
  id: string
  text: string
  level: number
}

export function generateHeadingId(text: string, seen: Map<string, number>): string {
  const baseId = text
    .toLowerCase()
    .replace(/[^\w一-龥]+/g, '-')
    .replace(/^-+|-+$/g, '')

  const count = seen.get(baseId) || 0
  seen.set(baseId, count + 1)

  return count > 0 ? `${baseId}-${count}` : baseId
}

const CODE_FENCE_PATTERN = /^```/

export function extractHeadings(content: string): HeadingEntry[] {
  if (!content) return []

  const lines = content.split('\n')
  const result: HeadingEntry[] = []
  const seen = new Map<string, number>()
  let inCodeBlock = false

  for (const line of lines) {
    if (CODE_FENCE_PATTERN.test(line)) {
      inCodeBlock = !inCodeBlock
      continue
    }

    if (inCodeBlock) continue

    const match = line.match(/^(#{1,3})\s+(.+)$/)
    if (!match) continue

    const level = match[1].length
    const text = match[2].trim()
    if (!text) continue

    const id = generateHeadingId(text, seen)
    result.push({ id, text, level })
  }

  return result
}
