import { describe, expect, it } from 'vitest'

import { extractWikiTagCandidates } from './wiki-tag-parser'

describe('wiki-tag-parser', () => {
  it('extracts type tags and grouped tags from wiki content', () => {
    const result = extractWikiTagCandidates(`
- 类型: 动作冒险、Roguelike / 卡牌
- 题材: 科幻
- 子类型: 牌组构筑
- 视角: 第三人称
- 内容属性: 黑暗奇幻
`)

    expect(result).toEqual([
      { value: '动作冒险', sourceLabel: '类型' },
      { value: 'Roguelike', sourceLabel: '类型' },
      { value: '卡牌', sourceLabel: '类型' },
      { value: '科幻', sourceLabel: '题材', groupKey: 'genre' },
      { value: '牌组构筑', sourceLabel: '子类型', groupKey: 'subgenre' },
      { value: '第三人称', sourceLabel: '视角', groupKey: 'perspective' },
      { value: '黑暗奇幻', sourceLabel: '内容属性', groupKey: 'theme' },
    ])
  })

  it('normalizes tag text and deduplicates repeated values', () => {
    const result = extractWikiTagCandidates(`
类型: 合作 / 生存
类型: 合作、生存
题材: 科幻（太空）
题材: 科幻(太空)
`)

    expect(result).toEqual([
      { value: '合作', sourceLabel: '类型' },
      { value: '生存', sourceLabel: '类型' },
      { value: '科幻(太空)', sourceLabel: '题材', groupKey: 'genre' },
    ])
  })

  it('returns an empty array for empty content', () => {
    expect(extractWikiTagCandidates('')).toEqual([])
    expect(extractWikiTagCandidates(null)).toEqual([])
    expect(extractWikiTagCandidates(undefined)).toEqual([])
  })
})
