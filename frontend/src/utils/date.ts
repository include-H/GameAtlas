const DATE_ONLY_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/

export function parseDisplayDate(value?: string | null): Date | null {
  const normalized = (value || '').trim()
  if (!normalized) return null

  const dateOnlyMatch = DATE_ONLY_PATTERN.exec(normalized)
  if (dateOnlyMatch) {
    const year = Number.parseInt(dateOnlyMatch[1], 10)
    const month = Number.parseInt(dateOnlyMatch[2], 10)
    const day = Number.parseInt(dateOnlyMatch[3], 10)
    return new Date(year, month - 1, day)
  }

  const parsed = new Date(normalized)
  if (Number.isNaN(parsed.getTime())) {
    return null
  }

  return parsed
}

export function formatDisplayDate(value?: string | null): string {
  const date = parseDisplayDate(value)
  if (!date) return value?.trim() || ''
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

export function getDisplayYear(value?: string | null): string {
  const date = parseDisplayDate(value)
  if (!date) return ''
  return String(date.getFullYear())
}
