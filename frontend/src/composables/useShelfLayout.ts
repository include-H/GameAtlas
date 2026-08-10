import { computed, type Ref } from 'vue'
import type { StoreShelfGame } from '@/composables/useStoreSession'

export interface ShelfCell {
  game: StoreShelfGame
  dx: number
  dy: number
  rot: number
  z: number
}

export const boxStyle = (cell: ShelfCell) => ({
  '--dx': `${cell.dx}px`,
  '--dy': `${cell.dy}px`,
  '--rot': `${cell.rot}deg`,
  '--box-z': String(cell.z),
})

/**
 * 确定性伪随机摆放：每次进入页面结果一致，
 * 但每盒都有少量位置偏移与角度变化，避免“像素级整齐”。
 */
export const useShelfLayout = (gameStoreSessionGames: Ref<StoreShelfGame[]>) => {
  const shelfRows = computed<ShelfCell[][]>(() => {
    const rows: ShelfCell[][] = []
    gameStoreSessionGames.value.forEach((game, index) => {
      const rowIndex = Math.floor(index / 5)
      if (!rows[rowIndex]) rows[rowIndex] = []
      // 上层货架在前：row0 最前，row3 最后，避免下层盒子顶部穿到上层时遮挡关系错误
      const rowZ = [40, 30, 20, 10][rowIndex] ?? 10
      rows[rowIndex].push({
        game,
        dx: ((index * 37 + 11) % 5) - 2,
        dy: 0,
        rot: (((index * 29 + 7) % 7) - 3) / 12,
        z: rowZ + ((index * 7) % 5),
      })
    })
    return rows
  })

  return {
    shelfRows,
    boxStyle,
  }
}
