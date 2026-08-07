import { describe, expect, it } from 'vitest'

import {
  isAdminGameDetail,
  type AdminGameFileEntry,
  type GameDetail,
  type PublicGameFileEntry,
} from './types'

const makePublicFile = (id: number): PublicGameFileEntry => ({
  id,
  game_id: 1,
  file_name: `game-${id}.vhdx`,
  label: null,
  notes: null,
  size_bytes: 1024,
  sort_order: id,
  source_created_at: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
})

const makeAdminFile = (id: number): AdminGameFileEntry => ({
  ...makePublicFile(id),
  file_path: `/roms/game-${id}.vhdx`,
})

const gameWithFiles = (files: PublicGameFileEntry[]): GameDetail => {
  return { files } as unknown as GameDetail
}

describe('isAdminGameDetail', () => {
  it('rejects null and undefined details', () => {
    expect(isAdminGameDetail(null)).toBe(false)
    expect(isAdminGameDetail(undefined)).toBe(false)
  })

  it('accepts a detail without files as an editable empty admin state', () => {
    // 待处理工作台专治"缺文件"游戏：无文件详情必须能打开编辑，否则永远无法补文件。
    expect(isAdminGameDetail(gameWithFiles([]))).toBe(true)
  })

  it('requires every file to have a resolved file path', () => {
    expect(isAdminGameDetail(gameWithFiles([makeAdminFile(1)]))).toBe(true)
    expect(isAdminGameDetail(gameWithFiles([makeAdminFile(1), makePublicFile(2)]))).toBe(false)
    expect(isAdminGameDetail(gameWithFiles([makePublicFile(1)]))).toBe(false)
  })

  it('rejects blank file paths', () => {
    const file = makeAdminFile(1)
    file.file_path = '   '
    expect(isAdminGameDetail(gameWithFiles([file]))).toBe(false)
  })
})
