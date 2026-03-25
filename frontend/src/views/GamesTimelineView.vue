<template>
  <div class="games-timeline">
    <div class="games-timeline__hero page-hero">
      <div class="page-hero__content">
        <h1 class="page-hero__title text-gradient">发售时间轴</h1>
        <p class="page-hero__subtitle">默认展示近两年，继续下滑可追溯更早年份，按年与月份查看游戏发售记录。</p>
      </div>
    </div>

    <div v-if="isLoading" class="games-timeline__loading">
      <a-spin :size="24" />
      <p>正在整理时间线...</p>
    </div>

    <a-empty v-else-if="datedGames.length === 0" description="暂无带发售日期的游戏" />

    <template v-else>
      <a-timeline
        class="timeline-shell"
        label-position="relative"
      >
        <a-timeline-item
          v-for="row in timelineRows"
          :key="row.key"
          class="timeline-item"
        >
          <template v-if="row.showYearBadge" #label>
            <div class="timeline-item__year">{{ row.year }}</div>
          </template>

          <div class="timeline-item__content">
            <a-card class="timeline-month">
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
            </a-card>
          </div>
        </a-timeline-item>
      </a-timeline>

      <div v-if="hasMore" class="timeline-shell__loading-more">
        <template v-if="isLoadingMore">
          <a-spin :size="18" />
          <span>正在加载更多月份...</span>
        </template>
        <template v-else>
          <span>继续下滑加载更多月份</span>
          <a-button type="text" size="small" @click="handleManualLoadMore">
            加载更多
          </a-button>
        </template>
      </div>
      <div v-else class="timeline-shell__end">
        时间线已经到底了
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import GameCard from '@/components/GameCard.vue'
import gamesService from '@/services/games.service'
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

const route = useRoute()
const router = useRouter()

const isLoading = ref(false)
const isLoadingMore = ref(false)
const allGames = ref<Game[]>([])
const hasMore = ref(false)
const nextCursor = ref<string | null>(null)
const currentWindowFrom = ref<string | null>(null)
const currentWindowTo = ref<string | null>(null)
const scrollRootRef = ref<HTMLElement | null>(null)
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

const appendTimelineChunk = (games: Game[]) => {
  if (games.length === 0) return
  const existingIDs = new Set(allGames.value.map((game) => game.public_id).filter(Boolean) as string[])
  const incoming = games.filter((game) => Boolean(game.public_id) && !existingIDs.has(String(game.public_id)))
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
    appendTimelineChunk(response.data)
    hasMore.value = response.hasMore
    nextCursor.value = response.nextCursor
    currentWindowFrom.value = response.from
    currentWindowTo.value = response.to
    hasLoadedTimeline.value = true
  } finally {
    isLoading.value = false
  }
}

const openGame = (publicId: string) => {
  if (!publicId) return
  router.push({
    name: 'game-detail',
    params: { publicId },
    query: createDetailRouteQuery(route),
  })
}

onMounted(() => {
  scrollRootRef.value = resolveScrollRoot()
  scrollRootRef.value?.addEventListener('scroll', handleScrollLoadMore, { passive: true })
  loadTimeline()
})

onBeforeUnmount(() => {
  scrollRootRef.value?.removeEventListener('scroll', handleScrollLoadMore)
})
</script>

<style scoped>
.games-timeline {
  animation: fadeInUp 0.45s cubic-bezier(0.2, 0.8, 0.2, 1) forwards;
}

.games-timeline__hero {
  margin-bottom: 10px;
}

.games-timeline__intro {
  margin-bottom: 10px;
  border-radius: var(--radius-xl);
}

.games-timeline__intro-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
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
  margin-top: 4px;
}

.timeline-shell :deep(.arco-timeline-item) {
  padding-bottom: 28px;
}

.timeline-shell :deep(.arco-timeline-item-content) {
  min-width: 0;
}

.timeline-item__year {
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-1);
}

.timeline-item__content {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.timeline-month {
  background: var(--app-card-surface);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
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
  .timeline-shell :deep(.arco-timeline-item-label) {
    width: 72px;
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

  .timeline-shell :deep(.arco-timeline-item-label) {
    width: auto;
    margin-bottom: 10px;
  }

  .timeline-item__year {
    font-size: 14px;
  }

  .timeline-month__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px;
  }

}

@media (max-width: 520px) {
  .timeline-month__header {
    gap: 8px;
  }

  .timeline-month__header :deep(.arco-tag) {
    align-self: flex-start;
  }

  .timeline-month__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
