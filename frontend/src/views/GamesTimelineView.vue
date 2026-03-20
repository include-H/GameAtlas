<template>
  <div class="games-timeline">
    <div class="games-timeline__hero page-hero">
      <div class="page-hero__content">
        <h1 class="page-hero__title text-gradient">发售时间轴</h1>
        <p class="page-hero__subtitle">默认展示近两年，继续下滑可追溯更早年份，按年与月份查看游戏发售记录。</p>
      </div>

      <div class="games-timeline__hero-side">
        <div class="games-timeline__stat glass-panel">
          <span class="games-timeline__stat-value">{{ datedGames.length }}</span>
          <span class="games-timeline__stat-label">已归档游戏</span>
        </div>
        <div class="games-timeline__stat glass-panel">
          <span class="games-timeline__stat-value">{{ monthGroups.length }}</span>
          <span class="games-timeline__stat-label">时间分组</span>
        </div>
      </div>
    </div>

    <div v-if="isLoading" class="games-timeline__loading">
      <a-spin :size="24" />
      <p>正在整理时间线...</p>
    </div>

    <a-empty v-else-if="datedGames.length === 0" description="暂无带发售日期的游戏" />

    <div
      v-else
      ref="timelineShellRef"
      class="timeline-shell"
    >
      <div
        v-if="virtualPaddingTop > 0"
        class="timeline-shell__spacer"
        :style="{ height: `${virtualPaddingTop}px` }"
      />
      <section
        v-for="row in visibleTimelineRows"
        :key="row.key"
        class="timeline-year"
        :ref="(el) => setRowRef(row.key, el)"
      >
        <div class="timeline-year__rail">
          <div v-if="row.showYearBadge" class="timeline-year__badge">{{ row.year }}</div>
        </div>

        <div class="timeline-year__content">
          <section class="timeline-month glass-panel">
            <div class="timeline-month__header">
              <div>
                <h2 class="timeline-month__title">{{ row.month.monthLabel }}</h2>
                <p class="timeline-month__subtitle">{{ row.month.games.length }} 个游戏</p>
              </div>
              <a-tag color="orangered">{{ row.month.fullLabel }}</a-tag>
            </div>

            <div class="timeline-month__grid">
              <div
                v-for="game in row.month.games"
                :key="game.id"
                class="timeline-month__grid-item"
              >
                <game-card
                  :game="game"
                  cover-only
                  @view="openGame"
                />
              </div>
            </div>
          </section>
        </div>
      </section>
      <div
        v-if="virtualPaddingBottom > 0"
        class="timeline-shell__spacer"
        :style="{ height: `${virtualPaddingBottom}px` }"
      />

      <div v-if="hasMore" class="timeline-shell__loading-more">
        <template v-if="isLoadingMore">
          <a-spin :size="18" />
          <span>正在加载更多月份...</span>
        </template>
        <template v-else>
          <span>继续下滑加载更多月份</span>
          <a-button type="outline" size="small" @click="handleManualLoadMore">
            加载更多
          </a-button>
        </template>
      </div>
      <div v-else class="timeline-shell__end">
        时间线已经到底了
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import GameCard from '@/components/GameCard.vue'
import { gamesService } from '@/services/games.service'
import type { Game } from '@/services/types'
import { createDetailRouteQuery } from '@/utils/navigation'

defineOptions({
  name: 'GamesTimelineView',
})

interface TimelineMonthGroup {
  key: string
  year: string
  month: number
  monthLabel: string
  fullLabel: string
  sortValue: number
  games: Game[]
}

interface TimelineRow {
  key: string
  year: string
  month: TimelineMonthGroup
  showYearBadge: boolean
}

const TIMELINE_YEARS = 2
const TIMELINE_PAGE_SIZE = 60
const VIRTUAL_ENABLE_THRESHOLD = 24
const VIRTUAL_OVERSCAN_PX = 2000
const DEFAULT_ROW_HEIGHT = 320
const ROW_VERTICAL_GAP = 28

const route = useRoute()
const router = useRouter()

const isLoading = ref(false)
const isLoadingMore = ref(false)
const allGames = ref<Game[]>([])
const hasMore = ref(false)
const nextCursor = ref<string | null>(null)
const currentWindowFrom = ref<string | null>(null)
const currentWindowTo = ref<string | null>(null)
const timelineShellRef = ref<HTMLElement | null>(null)
const scrollRootRef = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewportHeight = ref(720)
const shellWidth = ref(1200)
const rowHeightMap = ref<Record<string, number>>({})
const isScrollingUp = ref(false)
const lastKnownScrollTop = ref(0)
const hasLoadedTimeline = ref(false)

const parseDateParts = (value?: string | null) => {
  const raw = (value || '').trim()
  const match = raw.match(/^(\d{4})-(\d{1,2})(?:-(\d{1,2}))?/)
  if (!match) return null

  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3] || '1')

  if (Number.isNaN(year) || Number.isNaN(month) || Number.isNaN(day)) {
    return null
  }

  return {
    year: String(year),
    month,
    day,
    monthKey: `${year}-${String(month).padStart(2, '0')}`,
    timestamp: Date.UTC(year, Math.max(0, month - 1), day),
    monthSortValue: year * 100 + month,
  }
}

const monthFormatter = new Intl.DateTimeFormat('zh-CN', { month: 'long' })

const datedGames = computed(() => {
  return [...allGames.value]
    .filter((game) => parseDateParts(game.release_date))
    .sort((left, right) => {
      const leftDate = parseDateParts(left.release_date)
      const rightDate = parseDateParts(right.release_date)
      return (rightDate?.timestamp || 0) - (leftDate?.timestamp || 0)
    })
})

const monthGroups = computed<TimelineMonthGroup[]>(() => {
  const map = new Map<string, TimelineMonthGroup>()

  for (const game of datedGames.value) {
    const parts = parseDateParts(game.release_date)
    if (!parts) continue

    if (!map.has(parts.monthKey)) {
      const monthLabel = monthFormatter.format(new Date(Date.UTC(Number(parts.year), parts.month - 1, 1)))
      map.set(parts.monthKey, {
        key: parts.monthKey,
        year: parts.year,
        month: parts.month,
        monthLabel,
        fullLabel: `${parts.year} / ${String(parts.month).padStart(2, '0')}`,
        sortValue: parts.monthSortValue,
        games: [],
      })
    }

    map.get(parts.monthKey)?.games.push(game)
  }

  return [...map.values()].sort((left, right) => right.sortValue - left.sortValue)
})

const timelineRows = computed<TimelineRow[]>(() => {
  const rows: TimelineRow[] = []
  let previousYear = ''
  for (const monthGroup of monthGroups.value) {
    const showYearBadge = previousYear !== monthGroup.year
    rows.push({
      key: monthGroup.key,
      year: monthGroup.year,
      month: monthGroup,
      showYearBadge,
    })
    previousYear = monthGroup.year
  }
  return rows
})

const timelineColumns = computed(() => {
  const width = shellWidth.value
  if (width <= 520) return 3
  if (width <= 768) return 4
  if (width <= 900) return 6
  if (width <= 1200) return 7
  if (width <= 1400) return 8
  return 10
})

const estimatedCardHeight = computed(() => {
  const columns = Math.max(1, timelineColumns.value)
  const effectiveWidth = Math.max(shellWidth.value - 140, 320)
  const cardWidth = effectiveWidth / columns
  return Math.max(96, Math.min(190, Math.round(cardWidth * (4 / 3))))
})

const estimateRowHeight = (gameCount: number) => {
  const columns = Math.max(1, timelineColumns.value)
  const rows = Math.max(1, Math.ceil(gameCount / columns))
  const gridHeight = rows * estimatedCardHeight.value + Math.max(0, rows - 1) * 6
  return 72 + gridHeight + ROW_VERTICAL_GAP
}

const getRowHeight = (row: TimelineRow) => {
  return rowHeightMap.value[row.key] || Math.max(DEFAULT_ROW_HEIGHT, estimateRowHeight(row.month.games.length))
}

const getRowHeightFromMap = (row: TimelineRow, map: Record<string, number>) => {
  return map[row.key] || Math.max(DEFAULT_ROW_HEIGHT, estimateRowHeight(row.month.games.length))
}

const shouldVirtualize = computed(() => timelineRows.value.length >= VIRTUAL_ENABLE_THRESHOLD)

const virtualRowsState = computed(() => {
  const rows = timelineRows.value
  if (!shouldVirtualize.value || rows.length === 0) {
    return {
      rows,
      paddingTop: 0,
      paddingBottom: 0,
    }
  }

  const topBoundary = Math.max(0, scrollTop.value - VIRTUAL_OVERSCAN_PX)
  const bottomBoundary = scrollTop.value + Math.max(viewportHeight.value, 600) + VIRTUAL_OVERSCAN_PX

  let startIndex = 0
  let topOffset = 0

  for (let i = 0; i < rows.length; i += 1) {
    const rowHeight = getRowHeight(rows[i])
    if (topOffset + rowHeight >= topBoundary) {
      startIndex = i
      break
    }
    topOffset += rowHeight
  }

  let endIndex = startIndex
  let consumedHeight = topOffset
  while (endIndex < rows.length && consumedHeight < bottomBoundary) {
    consumedHeight += getRowHeight(rows[endIndex])
    endIndex += 1
  }

  endIndex = Math.max(endIndex, Math.min(startIndex + 1, rows.length))

  let totalHeight = 0
  for (const row of rows) {
    totalHeight += getRowHeight(row)
  }

  return {
    rows: rows.slice(startIndex, endIndex),
    paddingTop: topOffset,
    paddingBottom: Math.max(0, totalHeight - consumedHeight),
  }
})

const visibleTimelineRows = computed(() => virtualRowsState.value.rows)
const virtualPaddingTop = computed(() => virtualRowsState.value.paddingTop)
const virtualPaddingBottom = computed(() => virtualRowsState.value.paddingBottom)

const syncViewportMetrics = () => {
  const root = scrollRootRef.value
  if (root) {
    const nextTop = root.scrollTop
    isScrollingUp.value = nextTop < lastKnownScrollTop.value - 0.5
    lastKnownScrollTop.value = nextTop
    scrollTop.value = root.scrollTop
    viewportHeight.value = root.clientHeight
  }

  const shell = timelineShellRef.value
  if (shell) {
    shellWidth.value = shell.clientWidth
  } else if (root) {
    shellWidth.value = root.clientWidth
  }
}

const getPrefixHeight = (rows: TimelineRow[], map: Record<string, number>, endIndexExclusive: number) => {
  let total = 0
  for (let index = 0; index < endIndexExclusive; index += 1) {
    total += getRowHeightFromMap(rows[index], map)
  }
  return total
}

const locateAnchorByScrollTop = (rows: TimelineRow[], map: Record<string, number>, currentTop: number) => {
  let accumulated = 0
  for (let index = 0; index < rows.length; index += 1) {
    const rowHeight = getRowHeightFromMap(rows[index], map)
    if (accumulated + rowHeight > currentTop) {
      return {
        index,
        offsetWithin: currentTop - accumulated,
      }
    }
    accumulated += rowHeight
  }

  if (rows.length === 0) return null
  return {
    index: rows.length - 1,
    offsetWithin: 0,
  }
}

const resolveElementRef = (el: unknown): HTMLElement | null => {
  if (el instanceof HTMLElement) return el
  if (el && typeof el === 'object' && '$el' in el) {
    const candidate = (el as { $el?: unknown }).$el
    return candidate instanceof HTMLElement ? candidate : null
  }
  return null
}

const setRowRef = (key: string, el: unknown) => {
  const element = resolveElementRef(el)
  if (!element) return

  window.requestAnimationFrame(() => {
    const measured = Math.max(DEFAULT_ROW_HEIGHT, element.offsetHeight + ROW_VERTICAL_GAP)
    const current = rowHeightMap.value[key]
    if (current && Math.abs(current - measured) < 4) return

    const root = scrollRootRef.value
    const rows = timelineRows.value
    const previousMap = rowHeightMap.value
    const nextMap = {
      ...previousMap,
      [key]: measured,
    }

    let compensatedTop: number | null = null
    if (root && shouldVirtualize.value && isScrollingUp.value && rows.length > 0) {
      const anchor = locateAnchorByScrollTop(rows, previousMap, root.scrollTop)
      if (anchor) {
        const newPrefix = getPrefixHeight(rows, nextMap, anchor.index)
        compensatedTop = Math.max(0, newPrefix + anchor.offsetWithin)
      }
    }

    rowHeightMap.value = nextMap

    if (root && compensatedTop !== null) {
      const targetTop = compensatedTop
      void nextTick(() => {
        root.scrollTop = targetTop
        syncViewportMetrics()
      })
    }
  })
}

const handleWindowResize = () => {
  rowHeightMap.value = {}
  syncViewportMetrics()
}

const appendTimelineChunk = (games: Game[]) => {
  if (games.length === 0) return
  const existingIDs = new Set(allGames.value.map((game) => String(game.id)))
  const incoming = games.filter((game) => !existingIDs.has(String(game.id)))
  if (incoming.length > 0) {
    allGames.value.push(...incoming)
  }
}

const getPreviousWindowTo = (fromDate?: string | null) => {
  const raw = (fromDate || '').trim()
  if (!raw) return null

  const baseDate = new Date(`${raw}T00:00:00Z`)
  if (Number.isNaN(baseDate.getTime())) return null

  baseDate.setUTCDate(baseDate.getUTCDate() - 1)
  return baseDate.toISOString().slice(0, 10)
}

const loadMoreTimeline = async () => {
  if (!hasMore.value || isLoadingMore.value) return
  isLoadingMore.value = true
  try {
    const response = nextCursor.value
      ? await gamesService.getTimelineGames({
        years: TIMELINE_YEARS,
        limit: TIMELINE_PAGE_SIZE,
        cursor: nextCursor.value,
        from: currentWindowFrom.value,
        to: currentWindowTo.value,
      })
      : await gamesService.getTimelineGames({
        years: TIMELINE_YEARS,
        limit: TIMELINE_PAGE_SIZE,
        to: getPreviousWindowTo(currentWindowFrom.value),
      })
    appendTimelineChunk(response.data)
    hasMore.value = response.hasMore
    nextCursor.value = response.nextCursor
    currentWindowFrom.value = response.from
    currentWindowTo.value = response.to
  } finally {
    isLoadingMore.value = false
  }
}

const handleManualLoadMore = () => {
  void loadMoreTimeline()
}

const resolveScrollRoot = () => {
  if (typeof document === 'undefined') return null
  return document.querySelector('.content') as HTMLElement | null
}

const handleScrollLoadMore = () => {
  const root = scrollRootRef.value
  if (!root) return

  syncViewportMetrics()
  if (!hasMore.value || isLoadingMore.value) return
  if (root.scrollTop <= 0) return

  const remaining = root.scrollHeight - root.scrollTop - root.clientHeight
  if (remaining <= 320) {
    void loadMoreTimeline()
  }
}

const loadTimeline = async () => {
  if (isLoading.value || hasLoadedTimeline.value) return
  isLoading.value = true

  try {
    const response = await gamesService.getTimelineGames({
      years: TIMELINE_YEARS,
      limit: TIMELINE_PAGE_SIZE,
    })
    allGames.value = []
    rowHeightMap.value = {}
    appendTimelineChunk(response.data)
    hasMore.value = response.hasMore
    nextCursor.value = response.nextCursor
    currentWindowFrom.value = response.from
    currentWindowTo.value = response.to
    await nextTick()
    syncViewportMetrics()
    hasLoadedTimeline.value = true
  } finally {
    isLoading.value = false
  }
}

const openGame = (id: string | number) => {
  router.push({
    name: 'game-detail',
    params: { id: String(id) },
    query: createDetailRouteQuery(route),
  })
}

onMounted(() => {
  scrollRootRef.value = resolveScrollRoot()
  scrollRootRef.value?.addEventListener('scroll', handleScrollLoadMore, { passive: true })
  window.addEventListener('resize', handleWindowResize, { passive: true })
  syncViewportMetrics()
  loadTimeline()
})

onBeforeUnmount(() => {
  scrollRootRef.value?.removeEventListener('scroll', handleScrollLoadMore)
  window.removeEventListener('resize', handleWindowResize)
})
</script>

<style scoped>
.games-timeline {
  animation: fadeInUp 0.45s cubic-bezier(0.2, 0.8, 0.2, 1) forwards;
}

.games-timeline__hero {
  margin-bottom: 20px;
}

.games-timeline__hero-side {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.games-timeline__stat {
  min-width: 132px;
  padding: 16px 18px;
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  gap: 6px;
  box-shadow: var(--shadow-soft);
}

.games-timeline__stat-value {
  font-size: 28px;
  line-height: 1;
  font-weight: 800;
  color: var(--color-text-1);
}

.games-timeline__stat-label {
  font-size: 12px;
  color: var(--color-text-3);
}

.games-timeline__intro {
  margin-bottom: 20px;
  border-radius: var(--radius-xl);
}

.games-timeline__intro-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.games-timeline__intro-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-1);
}

.games-timeline__intro-text {
  margin-top: 4px;
  color: var(--color-text-3);
  font-size: 13px;
}

.games-timeline__loading {
  min-height: 280px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.timeline-shell {
  position: relative;
  padding-left: 24px;
}

.timeline-shell::before {
  content: '';
  position: absolute;
  left: 10px;
  top: 0;
  bottom: 0;
  width: 2px;
  background: linear-gradient(180deg, rgba(26, 159, 255, 0.55), rgba(26, 159, 255, 0.08));
}

.timeline-year {
  position: relative;
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  gap: 20px;
  margin-bottom: 28px;
}

.timeline-year__badge {
  position: sticky;
  top: 84px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 72px;
  padding: 10px 14px;
  border-radius: var(--radius-pill);
  background: linear-gradient(135deg, rgba(26, 159, 255, 0.95), rgba(0, 132, 240, 0.85));
  color: white;
  font-size: 16px;
  font-weight: 800;
  box-shadow: 0 12px 30px rgba(26, 159, 255, 0.28);
}

.timeline-year__content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.timeline-month {
  padding: 10px;
  border-radius: 14px;
  border: 1px solid var(--color-border-2);
}

.timeline-month__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
}

.timeline-month__title {
  margin: 0;
  font-size: 16px;
  color: var(--color-text-1);
}

.timeline-month__subtitle {
  margin: 4px 0 0;
  color: var(--color-text-3);
  font-size: 11px;
}

.timeline-month__grid {
  display: grid;
  grid-template-columns: repeat(10, minmax(0, 1fr));
  gap: 6px;
}

.timeline-month__grid-item {
  min-width: 0;
}

.timeline-shell__sentinel {
  height: 1px;
}

.timeline-shell__spacer {
  width: 100%;
  pointer-events: none;
}

.timeline-shell__loading-more,
.timeline-shell__end {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 20px 0 8px;
  color: var(--color-text-3);
  font-size: 13px;
}

@media (max-width: 1400px) {
  .timeline-month__grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}

@media (max-width: 1200px) {
  .timeline-month__grid {
    grid-template-columns: repeat(7, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .timeline-shell {
    padding-left: 0;
  }

  .timeline-shell::before {
    display: none;
  }

  .timeline-year {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .timeline-year__badge {
    position: relative;
    top: 0;
  }

  .timeline-month__grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .games-timeline__hero,
  .games-timeline__intro-row,
  .timeline-month__header {
    flex-direction: column;
    align-items: flex-start;
  }

  .games-timeline__hero-side {
    width: 100%;
    justify-content: flex-start;
  }

  .timeline-month__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px;
  }

  .timeline-month {
    padding: 8px;
  }
}

@media (max-width: 520px) {
  .timeline-month__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
