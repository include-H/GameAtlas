import type { StartScreenTile, StartScreenTileSize } from '@/services/types'

export const START_SCREEN_FREE_COLS = 12

// 组高上限：12 行（3 个 4x4 竖排），放不下溢出到右侧组顶部（复刻 Win8 组高度）。
export const START_SCREEN_GROUP_MAX_ROWS = 12

// Win10 磁贴比例体系：没有 1x1 那么小的资源，最小单元从 2x2 起，
// 2x2（小）→ 2x4（宽）→ 4x4（大）。
const START_SCREEN_TILE_SPANS: Record<StartScreenTileSize, { rows: number; cols: number }> = {
  small: { rows: 2, cols: 2 },
  wide: { rows: 2, cols: 4 },
  large: { rows: 4, cols: 4 },
}

export interface PackedStartScreenSlot {
  tile: StartScreenTile
  globalIndex: number
  row: number
  col: number
}

export interface PackedStartScreenGroup {
  columnIndex: number
  slots: PackedStartScreenSlot[]
}

export interface StartScreenDropTarget {
  columnIndex: number
  row: number
  col: number
}

// 组内 12 列 × 行数不限的稀疏占用表：row -> 已占用的列位
type Occupancy = Map<number, boolean[]>

const createOccupancy = (): Occupancy => new Map()

const fits = (occupied: Occupancy, row: number, col: number, rows: number, cols: number): boolean => {
  for (let r = row; r < row + rows; r += 1) {
    const line = occupied.get(r)
    if (!line) continue
    for (let c = col; c < col + cols; c += 1) {
      if (line[c]) return false
    }
  }
  return true
}

// 行优先找空位；超出 maxRow（磁贴底部越界）返回 null，由调用方决定溢出迁移。
const findFirstFit = (
  occupied: Occupancy,
  rows: number,
  cols: number,
  fromRow = 0,
  maxRow = Number.POSITIVE_INFINITY,
): { row: number; col: number } | null => {
  let row = fromRow
  for (;;) {
    if (row + rows > maxRow) return null
    for (let col = 0; col <= START_SCREEN_FREE_COLS - cols; col += 1) {
      if (fits(occupied, row, col, rows, cols)) {
        return { row, col }
      }
    }
    row += 1
  }
}

const occupy = (occupied: Occupancy, row: number, col: number, rows: number, cols: number) => {
  for (let r = row; r < row + rows; r += 1) {
    const line = occupied.get(r) ?? []
    occupied.set(r, line)
    for (let c = col; c < col + cols; c += 1) {
      line[c] = true
    }
  }
}

const normalizedPosition = (value: number, fallback: number): number => {
  if (!Number.isFinite(value)) return fallback
  return Math.max(0, Math.trunc(value))
}

// 按列构建占用表（跳过 excludedGameId，即拖拽中的磁贴）
const buildOccupancy = (tiles: StartScreenTile[], excludedGameId: number): Map<number, Occupancy> => {
  const occupiedByColumn = new Map<number, Occupancy>()
  const columnOf = (index: number): Occupancy => {
    const existing = occupiedByColumn.get(index)
    if (existing) return existing
    const next = createOccupancy()
    occupiedByColumn.set(index, next)
    return next
  }
  for (const tile of tiles) {
    if (tile.game_id === excludedGameId) continue
    const span = START_SCREEN_TILE_SPANS[tile.tile_size]
    const tileColumn = columnOf(normalizedPosition(tile.column_index, 0))
    const tileRow = normalizedPosition(tile.grid_row, 0)
    const tileCol = Math.min(
      START_SCREEN_FREE_COLS - span.cols,
      normalizedPosition(tile.grid_col, 0),
    )
    if (fits(tileColumn, tileRow, tileCol, span.rows, span.cols)) {
      occupy(tileColumn, tileRow, tileCol, span.rows, span.cols)
    }
  }
  return occupiedByColumn
}

/**
 * 全屏自定义网格：组（列）宽 12 列、高 12 行，行优先 first-fit 自动摆放，
 * 组内放满后开新组（column_index + 1），任意尺寸组合自然铺开（复刻 Win8 组高）。
 */
export function packStartScreenTiles(tiles: StartScreenTile[]): PackedStartScreenGroup[] {
  if (tiles.length === 0) return []
  const groups: PackedStartScreenGroup[] = []
  let columnIndex = 0
  let occupied = createOccupancy()
  let slots: PackedStartScreenSlot[] = []
  const flush = () => {
    if (slots.length > 0) {
      groups.push({ columnIndex, slots })
      slots = []
    }
  }

  tiles.forEach((tile, globalIndex) => {
    const span = START_SCREEN_TILE_SPANS[tile.tile_size]
    const position = findFirstFit(occupied, span.rows, span.cols, 0, START_SCREEN_GROUP_MAX_ROWS)
    if (!position) {
      flush()
      columnIndex += 1
      occupied = createOccupancy()
      const next = findFirstFit(occupied, span.rows, span.cols, 0, START_SCREEN_GROUP_MAX_ROWS)
      if (!next) throw new Error('unreachable: empty group always fits')
      slots.push({
        tile: { ...tile, column_index: columnIndex, grid_row: next.row, grid_col: next.col },
        globalIndex,
        row: next.row,
        col: next.col,
      })
      occupy(occupied, next.row, next.col, span.rows, span.cols)
      return
    }
    slots.push({
      tile: { ...tile, column_index: columnIndex, grid_row: position.row, grid_col: position.col },
      globalIndex,
      row: position.row,
      col: position.col,
    })
    occupy(occupied, position.row, position.col, span.rows, span.cols)
  })
  flush()
  return groups
}

/**
 * 显式坐标纠正为不重叠、不越界（组高 12 行）的布局：组内就近找空位；
 * 组内放不下（含被顶出底部）的磁贴迁移到右侧组顶部继续放置（链式）。
 * 同时压缩空行：组内磁贴按行号重映射（跳过完全空的行），布局无洞。
 */
export function normalizeStartScreenTiles(tiles: StartScreenTile[]): StartScreenTile[] {
  const normalized = tiles.map((tile) => ({
    ...tile,
    column_index: normalizedPosition(tile.column_index, 0),
    grid_row: normalizedPosition(tile.grid_row, 0),
    grid_col: normalizedPosition(tile.grid_col, 0),
  }))

  const byColumn = new Map<number, StartScreenTile[]>()
  for (const tile of normalized) {
    const list = byColumn.get(tile.column_index) ?? []
    list.push(tile)
    byColumn.set(tile.column_index, list)
  }

  const result: StartScreenTile[] = []
  let guard = 0
  while (byColumn.size > 0 && guard < 10000) {
    guard += 1
    const columnIndex = Math.min(...byColumn.keys())
    const groupTiles = byColumn.get(columnIndex) as StartScreenTile[]
    byColumn.delete(columnIndex)
    groupTiles.sort(
      (a, b) =>
        a.grid_row - b.grid_row ||
        a.grid_col - b.grid_col ||
        a.sort_order - b.sort_order ||
        a.game_id - b.game_id,
    )
    // 空行压缩：按"占用行集合"（含磁贴内部行）重映射为连续行带——
    // 2x2 磁贴占 2 行，按起始行压缩会让相邻磁贴重叠；按占用行压缩
    // 保留磁贴间的最小行距，只去掉完全空的行。
    const occupiedRows = new Set<number>()
    for (const tile of groupTiles) {
      const span = START_SCREEN_TILE_SPANS[tile.tile_size]
      for (let r = tile.grid_row; r < tile.grid_row + span.rows; r += 1) {
        occupiedRows.add(r)
      }
    }
    const sortedRows = [...occupiedRows].sort((a, b) => a - b)
    const rowMap = new Map(sortedRows.map((row, index) => [row, index]))
    for (const tile of groupTiles) {
      tile.grid_row = rowMap.get(tile.grid_row) ?? 0
    }
    const occupied = createOccupancy()
    const overflow: StartScreenTile[] = []
    for (const tile of groupTiles) {
      const span = START_SCREEN_TILE_SPANS[tile.tile_size]
      const row = tile.grid_row
      const col = Math.min(START_SCREEN_FREE_COLS - span.cols, tile.grid_col)
      if (row + span.rows <= START_SCREEN_GROUP_MAX_ROWS && fits(occupied, row, col, span.rows, span.cols)) {
        occupy(occupied, row, col, span.rows, span.cols)
        result.push({ ...tile, grid_row: row, grid_col: col })
        continue
      }
      const position = findFirstFit(
        occupied,
        span.rows,
        span.cols,
        Math.min(row, START_SCREEN_GROUP_MAX_ROWS),
        START_SCREEN_GROUP_MAX_ROWS,
      )
      if (position) {
        occupy(occupied, position.row, position.col, span.rows, span.cols)
        result.push({ ...tile, grid_row: position.row, grid_col: position.col })
        continue
      }
      // 组内放不下 → 迁移右侧组顶部
      overflow.push({ ...tile, column_index: columnIndex + 1, grid_row: 0, grid_col: 0 })
    }
    if (overflow.length > 0) {
      const target = byColumn.get(columnIndex + 1) ?? []
      byColumn.set(columnIndex + 1, [...target, ...overflow])
    }
  }
  return result
}

export function layoutStartScreenTiles(
  tiles: StartScreenTile[],
  columnCount = 0,
): PackedStartScreenGroup[] {
  const normalized = normalizeStartScreenTiles(tiles)
  const originalIndices = new Map<number, number>()
  tiles.forEach((tile, index) => originalIndices.set(tile.game_id, index))

  const maxTileColumn = normalized.reduce((max, tile) => Math.max(max, tile.column_index), -1)
  const count = Math.max(1, columnCount, maxTileColumn + 1)

  const groups: PackedStartScreenGroup[] = []
  for (let columnIndex = 0; columnIndex < count; columnIndex += 1) {
    const groupTiles = normalized.filter((tile) => tile.column_index === columnIndex)
    const slots: PackedStartScreenSlot[] = groupTiles.map((tile) => ({
      tile,
      globalIndex: originalIndices.get(tile.game_id) ?? tile.sort_order,
      row: tile.grid_row,
      col: tile.grid_col,
    }))
    slots.sort((a, b) => a.row - b.row || a.col - b.col || a.tile.sort_order - b.tile.sort_order)
    groups.push({ columnIndex, slots })
  }
  return groups
}

// 从 (row, col) 起向右下的最大连续空闲矩形宽度（每行连续空闲列的最小宽度）
const freeRectExtent = (
  occupied: Occupancy,
  row: number,
  col: number,
  minCols: number,
): { rows: number; cols: number } => {
  let width = START_SCREEN_FREE_COLS - col
  let rows = 0
  for (;;) {
    const line = occupied.get(row + rows)
    if (!line) break
    let w = 0
    for (let c = col; c < START_SCREEN_FREE_COLS && !line[c]; c += 1) w += 1
    if (w < minCols) break
    width = Math.min(width, w)
    rows += 1
  }
  return { rows, cols: width }
}

// 指针格子到矩形（含内部）的切比雪夫距离
const rectChebyshevDistance = (
  pointerRow: number,
  pointerCol: number,
  row: number,
  col: number,
  rows: number,
  cols: number,
): number => {
  const dx = Math.max(0, col - pointerCol, pointerCol - (col + cols - 1))
  const dy = Math.max(0, row - pointerRow, pointerRow - (row + rows - 1))
  return Math.max(dx, dy)
}

// 吸附：指针附近"空闲矩形恰好等于磁贴尺寸"的精确空位，指针落在空位内或
// 边缘 1 格即吸附到其左上角——宽磁贴不必精确拖到第一个格子。
const findSnapFit = (
  occupied: Occupancy,
  rows: number,
  cols: number,
  pointerRow: number,
  pointerCol: number,
): { row: number; col: number } | null => {
  let best: { row: number; col: number; distance: number } | null = null
  for (let r = Math.max(0, pointerRow - 1); r <= pointerRow + 1; r += 1) {
    for (let c = 0; c <= START_SCREEN_FREE_COLS - cols; c += 1) {
      if (!fits(occupied, r, c, rows, cols)) continue
      const extent = freeRectExtent(occupied, r, c, cols)
      if (extent.rows !== rows || extent.cols !== cols) continue
      const distance = rectChebyshevDistance(pointerRow, pointerCol, r, c, rows, cols)
      if (distance <= 1 && (!best || distance < best.distance)) {
        best = { row: r, col: c, distance }
      }
    }
  }
  return best ? { row: best.row, col: best.col } : null
}

// 落点解析：请求坐标可放 → 直落；否则精确空位吸附；否则返回请求坐标本身
// （冲突由应用层避让插入解决，不再 first-fit 弹走）。
const resolveDropTarget = (
  occupied: Occupancy,
  span: { rows: number; cols: number },
  requestedRow: number,
  requestedCol: number,
): StartScreenDropTarget => {
  if (fits(occupied, requestedRow, requestedCol, span.rows, span.cols)) {
    return { columnIndex: 0, row: requestedRow, col: requestedCol }
  }
  const snap = findSnapFit(occupied, span.rows, span.cols, requestedRow, requestedCol)
  if (snap) {
    return { columnIndex: 0, row: snap.row, col: snap.col }
  }
  return { columnIndex: 0, row: requestedRow, col: requestedCol }
}

export function findStartScreenDropTarget(
  tiles: StartScreenTile[],
  excludedGameId: number,
  columnIndex: number,
  row: number,
  col: number,
  tileSize: StartScreenTileSize,
): StartScreenDropTarget {
  const span = START_SCREEN_TILE_SPANS[tileSize]
  const targetColumn = buildOccupancy(tiles, excludedGameId).get(
    Math.max(0, Math.trunc(columnIndex || 0)),
  )
  const requestedRow = Math.max(0, Math.trunc(row || 0))
  const requestedCol = Math.min(
    START_SCREEN_FREE_COLS - span.cols,
    Math.max(0, Math.trunc(col || 0)),
  )
  const target = resolveDropTarget(targetColumn ?? createOccupancy(), span, requestedRow, requestedCol)
  return { columnIndex: Math.max(0, Math.trunc(columnIndex || 0)), row: target.row, col: target.col }
}

export interface StartScreenInsertionMove {
  gameId: number
  columnIndex: number
  row: number
  col: number
}

export interface StartScreenInsertionPlan {
  target: StartScreenDropTarget
  moves: StartScreenInsertionMove[]
}

// 与 (row, col, rows, cols) 矩形相交的其他磁贴（同列、排除 excludedGameId），行优先排序
// 与 (row, col, rows, cols) 矩形相交的磁贴：同列、非 excluded，
// 位置取"已放置位置（placed）或原坐标"，行优先排序。
const findBlockers = (
  tiles: StartScreenTile[],
  excludedGameId: number,
  columnIndex: number,
  row: number,
  col: number,
  rows: number,
  cols: number,
  placed: Map<number, { row: number; col: number }>,
): StartScreenTile[] => {
  const blockers: StartScreenTile[] = []
  for (const tile of tiles) {
    if (tile.game_id === excludedGameId) continue
    if (normalizedPosition(tile.column_index, 0) !== columnIndex) continue
    const span = START_SCREEN_TILE_SPANS[tile.tile_size]
    const position = placed.get(tile.game_id) ?? {
      row: normalizedPosition(tile.grid_row, 0),
      col: normalizedPosition(tile.grid_col, 0),
    }
    const overlaps =
      position.row < row + rows &&
      position.row + span.rows > row &&
      position.col < col + cols &&
      position.col + span.cols > col
    if (overlaps) blockers.push(tile)
  }
  return blockers.sort(
    (a, b) =>
      (placed.get(a.game_id)?.row ?? normalizedPosition(a.grid_row, 0)) -
        (placed.get(b.game_id)?.row ?? normalizedPosition(b.grid_row, 0)) ||
      (placed.get(a.game_id)?.col ?? normalizedPosition(a.grid_col, 0)) -
        (placed.get(b.game_id)?.col ?? normalizedPosition(b.grid_col, 0)),
  )
}

const unoccupy = (occupied: Occupancy, row: number, col: number, rows: number, cols: number) => {
  for (let r = row; r < row + rows; r += 1) {
    const line = occupied.get(r)
    if (!line) continue
    for (let c = col; c < col + cols; c += 1) {
      line[c] = false
    }
  }
}

// 避让规划：推土机语义——落点矩形内被覆盖的磁贴"顶着"向下顺延一格
// （保持列），顺延目标被占则继续顶（递归）；越过组底（12 行）溢到
// 右侧组顶部继续顶（链式跨组）。被顶磁贴离开原位的格子立即释放，
// 供上层的磁贴回填。
const planDisplacement = (
  tiles: StartScreenTile[],
  excludedGameId: number,
  columns: Map<number, Occupancy>,
  columnIndex: number,
  row: number,
  col: number,
  rows: number,
  cols: number,
): StartScreenInsertionMove[] => {
  const moves: StartScreenInsertionMove[] = []
  const placed = new Map<number, { row: number; col: number }>()

  const placeWithShift = (
    gameId: number,
    span: { rows: number; cols: number },
    targetColumn: number,
    targetRow: number,
    targetCol: number,
  ) => {
    const blockers = findBlockers(
      tiles,
      excludedGameId,
      targetColumn,
      targetRow,
      targetCol,
      span.rows,
      span.cols,
      placed,
    )
    for (const blocker of blockers) {
      const blockerSpan = START_SCREEN_TILE_SPANS[blocker.tile_size]
      const blockerPos = placed.get(blocker.game_id) ?? {
        row: normalizedPosition(blocker.grid_row, 0),
        col: normalizedPosition(blocker.grid_col, 0),
      }
      const blockerColumn = normalizedPosition(blocker.column_index, 0)
      const blockerOcc = columns.get(blockerColumn)
      if (blockerOcc) {
        unoccupy(blockerOcc, blockerPos.row, blockerPos.col, blockerSpan.rows, blockerSpan.cols)
      }
      const nextRow = blockerPos.row + blockerSpan.rows
      if (nextRow + blockerSpan.rows > START_SCREEN_GROUP_MAX_ROWS) {
        placeWithShift(blocker.game_id, blockerSpan, blockerColumn + 1, 0, blockerPos.col)
      } else {
        placeWithShift(blocker.game_id, blockerSpan, blockerColumn, nextRow, blockerPos.col)
      }
    }
    const occupied = columns.get(targetColumn) ?? createOccupancy()
    columns.set(targetColumn, occupied)
    occupy(occupied, targetRow, targetCol, span.rows, span.cols)
    placed.set(gameId, { row: targetRow, col: targetCol })
    if (gameId !== excludedGameId) {
      moves.push({ gameId, columnIndex: targetColumn, row: targetRow, col: targetCol })
    }
  }

  placeWithShift(excludedGameId, { rows, cols }, columnIndex, row, col)
  return moves
}

/**
 * 避让插入落位方案：目标落点 + 被覆盖磁贴的避让移动列表。
 * 落点 = 请求坐标（可放则直落，否则精确空位吸附，冲突也返回请求坐标，
 * 不再 first-fit 弹走）；冲突时被覆盖磁贴链式让位。
 */
export function planStartScreenInsertion(
  tiles: StartScreenTile[],
  excludedGameId: number,
  columnIndex: number,
  row: number,
  col: number,
  tileSize: StartScreenTileSize,
): StartScreenInsertionPlan | null {
  const span = START_SCREEN_TILE_SPANS[tileSize]
  const requestedColumnIndex = Math.max(0, Math.trunc(columnIndex || 0))
  const columns = buildOccupancy(tiles, excludedGameId)
  const targetColumn = columns.get(requestedColumnIndex)
  const requestedRow = Math.max(0, Math.trunc(row || 0))
  const requestedCol = Math.min(
    START_SCREEN_FREE_COLS - span.cols,
    Math.max(0, Math.trunc(col || 0)),
  )
  // 目标列为空（无磁贴）→ 请求坐标直接可用
  const target = targetColumn
    ? resolveDropTarget(targetColumn, span, requestedRow, requestedCol)
    : { columnIndex: 0, row: requestedRow, col: requestedCol }
  if (!targetColumn || fits(targetColumn, target.row, target.col, span.rows, span.cols)) {
    return { target: { columnIndex: requestedColumnIndex, row: target.row, col: target.col }, moves: [] }
  }
  const moves = planDisplacement(
    tiles,
    excludedGameId,
    columns,
    requestedColumnIndex,
    target.row,
    target.col,
    span.rows,
    span.cols,
  )
  return { target: { columnIndex: requestedColumnIndex, row: target.row, col: target.col }, moves }
}
