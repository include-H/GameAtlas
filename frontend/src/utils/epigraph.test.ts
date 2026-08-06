import { describe, expect, it } from 'vitest'

import { classifyEpigraphLine } from './epigraph'

describe('classifyEpigraphLine', () => {
  it('classifies pure English lines as en (small text)', () => {
    expect(classifyEpigraphLine('In my restless dreams, I see that town...')).toBe('en')
    expect(classifyEpigraphLine("Tom Clancy's Splinter Cell")).toBe('en')
    expect(classifyEpigraphLine('WHO THREATENS THE NATION')).toBe('en')
  })

  it('classifies pure Chinese lines as cn (large text)', () => {
    expect(classifyEpigraphLine('名单上的名字越来越少，')).toBe('cn')
    expect(classifyEpigraphLine('谁威胁国家，谁进名单。')).toBe('cn')
  })

  it('regression: mixed line starting with English is cn, not en', () => {
    expect(classifyEpigraphLine('Sam Fisher 依然更激进了，')).toBe('cn')
  })

  it('regression: mixed line with more ASCII letters than CJK chars is cn', () => {
    expect(classifyEpigraphLine('Fourth Echelon 的规则只有一条：')).toBe('cn')
  })

  it('classifies lines without letters as cn', () => {
    expect(classifyEpigraphLine('……')).toBe('cn')
    expect(classifyEpigraphLine('42')).toBe('cn')
    expect(classifyEpigraphLine('')).toBe('cn')
  })
})
