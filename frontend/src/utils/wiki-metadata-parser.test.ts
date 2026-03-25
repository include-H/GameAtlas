import { describe, expect, it } from 'vitest'

import { extractWikiMetadata } from './wiki-metadata-parser'

describe('wiki-metadata-parser', () => {
  it('extracts inline metadata fields', () => {
    const result = extractWikiMetadata(`
- 简介: 经典回合制仙侠 RPG，围绕宿命与轮回展开。
- 发售日期: 2007年8月1日
- 开发商: 上海软星
- 发行商: 寰宇之星、方块游戏
- 平台: Windows / Steam
`)

    expect(result).toEqual({
      summary: '经典回合制仙侠 RPG，围绕宿命与轮回展开。',
      releaseDate: '2007-08-01',
      englishTitleAlt: '',
      chineseTitleAlt: '',
      engine: '',
      developers: ['上海软星'],
      publishers: ['寰宇之星', '方块游戏'],
      platforms: ['Windows', 'Steam'],
    })
  })

  it('recognizes real wiki aliases like 首发日期 and 已确认平台', () => {
    const result = extractWikiMetadata(`
- 开发商：上海软星
- 发行商：大宇资讯股份有限公司
- 首发日期：2007 年 8 月
- 已确认平台：Windows
`)

    expect(result).toEqual({
      summary: '',
      releaseDate: '2007-08-01',
      englishTitleAlt: '',
      chineseTitleAlt: '',
      engine: '',
      developers: ['上海软星'],
      publishers: ['大宇资讯股份有限公司'],
      platforms: ['Windows'],
    })
  })

  it('extracts summary from a heading block and deduplicates names', () => {
    const result = extractWikiMetadata(`
## 简介

《仙剑奇侠传4》讲述少年云天河踏入江湖后，
逐步揭开门派与宿命真相的故事。

开发商: 上海软星、上海软星
发行公司: 寰宇之星
平台: Windows、Windows、Steam
`)

    expect(result).toEqual({
      summary: '《仙剑奇侠传4》讲述少年云天河踏入江湖后，\n逐步揭开门派与宿命真相的故事。',
      releaseDate: '',
      englishTitleAlt: '',
      chineseTitleAlt: '',
      engine: '',
      developers: ['上海软星'],
      publishers: ['寰宇之星'],
      platforms: ['Windows', 'Steam'],
    })
  })

  it('extracts English and Chinese aliases separately', () => {
    const result = extractWikiMetadata(`
- 英文常见译名：The Legend of Sword and Fairy 4、Chinese Paladin 4
- 中文常见译名：仙剑奇侠传四、仙剑四
`)

    expect(result).toEqual({
      summary: '',
      releaseDate: '',
      englishTitleAlt: 'The Legend of Sword and Fairy 4 / Chinese Paladin 4',
      chineseTitleAlt: '仙剑奇侠传四 / 仙剑四',
      engine: '',
      developers: [],
      publishers: [],
      platforms: [],
    })
  })

  it('extracts engine fields', () => {
    const result = extractWikiMetadata(`
- 游戏引擎：Unreal Engine 5
`)

    expect(result).toEqual({
      summary: '',
      releaseDate: '',
      englishTitleAlt: '',
      chineseTitleAlt: '',
      engine: 'Unreal Engine 5',
      developers: [],
      publishers: [],
      platforms: [],
    })
  })

  it('strips narrative prefixes from platform fields', () => {
    const result = extractWikiMetadata(`
- 已确认平台：截至目前公开主流资料显示，本作已确认的主要平台为 Windows / Steam。
`)

    expect(result).toEqual({
      summary: '',
      releaseDate: '',
      englishTitleAlt: '',
      chineseTitleAlt: '',
      engine: '',
      developers: [],
      publishers: [],
      platforms: ['Windows', 'Steam'],
    })
  })

  it('extracts exact release date from narrative overview sentences', () => {
    const result = extractWikiMetadata(`
## 作品概览

本作由 NEKO WORKs 开发、Sekai Project 发行，于 2025 年 6 月 30 日在 Windows / Steam 发售。
`)

    expect(result).toEqual({
      summary: '',
      releaseDate: '2025-06-30',
      englishTitleAlt: '',
      chineseTitleAlt: '',
      engine: '',
      developers: [],
      publishers: [],
      platforms: [],
    })
  })

  it('returns empty metadata for empty wiki content', () => {
    expect(extractWikiMetadata('')).toEqual({
      summary: '',
      releaseDate: '',
      englishTitleAlt: '',
      chineseTitleAlt: '',
      engine: '',
      developers: [],
      publishers: [],
      platforms: [],
    })
  })
})
