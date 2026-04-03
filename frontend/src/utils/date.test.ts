import { describe, expect, it } from 'vitest'

import { formatDisplayDate, formatDisplayDateTime, getDisplayYear, parseDisplayDate } from './date'

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

  it('formats timestamps in mainland China timezone for date display', () => {
    expect(formatDisplayDate('2024-01-01T16:30:00Z')).toBe('2024-01-02')
    expect(formatDisplayDate('2024-01-01 16:30:00')).toBe('2024-01-02')
  })

  it('formats date time values in mainland China timezone', () => {
    expect(formatDisplayDateTime('2024-01-01T16:30:00Z')).toBe('2024-01-02 00:30')
    expect(formatDisplayDateTime('2024-01-01 16:30:00')).toBe('2024-01-02 00:30')
    expect(formatDisplayDateTime('invalid')).toBe('invalid')
  })
})
