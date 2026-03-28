import { describe, expect, it } from 'vitest'

import { formatDisplayDate, getDisplayYear, parseDisplayDate } from './date'

describe('date utils', () => {
  it('parses date-only values without timezone drift', () => {
    const parsed = parseDisplayDate('2024-01-01')

    expect(parsed).not.toBeNull()
    expect(parsed?.getFullYear()).toBe(2024)
    expect(parsed?.getMonth()).toBe(0)
    expect(parsed?.getDate()).toBe(1)
  })

  it('formats date-only values consistently', () => {
    expect(formatDisplayDate('2024-01-01')).toBe('2024-01-01')
    expect(getDisplayYear('2024-01-01')).toBe('2024')
  })

  it('keeps timestamp values usable and returns fallback for invalid values', () => {
    expect(formatDisplayDate('2024-01-01T12:34:56Z')).toBeTruthy()
    expect(formatDisplayDate('not-a-date')).toBe('not-a-date')
    expect(getDisplayYear('')).toBe('')
  })
})
