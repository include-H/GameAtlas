const DATE_ONLY_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/

const MAINLAND_CHINA_TIME_ZONE = 'Asia/Shanghai'

const displayDateFormatter = new Intl.DateTimeFormat('zh-CN', {
  timeZone: MAINLAND_CHINA_TIME_ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
})

const displayDateTimeFormatter = new Intl.DateTimeFormat('zh-CN', {
  timeZone: MAINLAND_CHINA_TIME_ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
})

function getFormatterPart(parts: Intl.DateTimeFormatPart[], type: Intl.DateTimeFormatPartTypes): string {
  return parts.find((part) => part.type === type)?.value || ''
}

function formatDateParts(parts: Intl.DateTimeFormatPart[]): string {
  return `${getFormatterPart(parts, 'year')}-${getFormatterPart(parts, 'month')}-${getFormatterPart(parts, 'day')}`
}

function normalizeDateTimeInput(value: string): string {
  const normalizedValue = value.includes('T') ? value : value.replace(' ', 'T')
  return /(?:Z|[+-]\d{2}:\d{2})$/.test(normalizedValue) ? normalizedValue : `${normalizedValue}Z`
}

function parseDisplayDateTime(value?: string | null): Date | null {
  const normalized = (value || '').trim()
  if (!normalized) return null

  const parsed = new Date(normalizeDateTimeInput(normalized))
  if (Number.isNaN(parsed.getTime())) {
    return null
  }

  return parsed
}

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
  const normalized = (value || '').trim()
  if (!normalized) return ''

  if (DATE_ONLY_PATTERN.test(normalized)) {
    return normalized
  }

  const date = parseDisplayDateTime(normalized)
  if (!date) return normalized
  return formatDateParts(displayDateFormatter.formatToParts(date))
}

export function getDisplayYear(value?: string | null): string {
  const date = parseDisplayDate(value)
  if (!date) return ''
  return String(date.getFullYear())
}

export function formatDisplayDateTime(value?: string | null): string {
  const normalized = (value || '').trim()
  if (!normalized) return ''

  const date = parseDisplayDateTime(normalized)
  if (!date) return normalized

  const parts = displayDateTimeFormatter.formatToParts(date)
  return `${formatDateParts(parts)} ${getFormatterPart(parts, 'hour')}:${getFormatterPart(parts, 'minute')}`
}
