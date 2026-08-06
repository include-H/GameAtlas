<template>
  <div class="series-library">
    <div class="series-library__header page-hero">
      <div class="page-hero__content">
        <h1 class="series-library__title page-hero__title text-gradient">系列库</h1>
        <p class="series-library__subtitle page-hero__subtitle">每个系列，都值得被好好归档。</p>
      </div>
      <div class="series-library__search app-glass-surface">
        <div class="series-library__search-body app-input-action-row">
          <a-input
            v-model="searchQuery"
            class="app-input-action-row__field"
            placeholder="搜索系列"
            allow-clear
            @press-enter="handleSearchSubmit"
          >
            <template #prefix>
              <icon-search />
            </template>
          </a-input>
          <a-button class="app-text-action-btn app-input-action-row__action" type="text" @click="handleSearchSubmit">
            搜索
          </a-button>
        </div>
      </div>
    </div>

    <div v-if="isLoading" class="series-library__loading">
      <a-spin :size="24" />
      <p>加载系列中...</p>
    </div>

    <template v-else>
      <a-empty v-if="hasLoadFailure" description="系列列表加载失败，请稍后重试。" />

      <template v-if="!hasLoadFailure && seriesCards.length > 0">
        <div class="series-library__grid">
          <div
            v-for="series in seriesCards"
            :key="series.id"
            class="series-card hover-lift app-glass-surface app-glass-surface--interactive"
            role="button"
            tabindex="0"
            @click="openSeries(series.id)"
            @keydown.enter="openSeries(series.id)"
            @keydown.space.prevent="openSeries(series.id)"
          >
            <div class="series-card__cover">
              <div
                v-if="(series.game_count || 0) >= 4 && series.cover_candidates && series.cover_candidates.length >= 4"
                class="series-card__collage"
              >
                <div
                  v-for="(cover, index) in series.cover_candidates.slice(0, 4)"
                  :key="`${series.id}-${index}`"
                  class="series-card__collage-tile"
                >
                  <img
                    :src="cover"
                    :alt="`${series.name}-${index + 1}`"
                    class="series-card__collage-image"
                  />
                </div>
              </div>
              <img
                v-else-if="series.cover_image"
                :src="series.cover_image"
                :alt="series.name"
                class="series-card__image"
              />
              <div v-else class="series-card__placeholder">
                {{ series.name.charAt(0) || '?' }}
              </div>
              <div class="series-card__overlay" />
            </div>
            <div class="series-card__body">
              <div class="series-card__title">{{ series.name }}</div>
              <div class="series-card__meta-row">
                <span>{{ series.game_count }} 部作品</span>
                <span v-if="series.latest_updated_at">{{ formatDate(series.latest_updated_at) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div
          ref="loadMoreSentinel"
          class="series-library__infinite-scroll"
        >
          <a-spin v-if="isLoadingMore" :size="20" />
        </div>
      </template>

      <a-empty v-else description="暂无系列数据" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { IconSearch } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'
import { seriesService } from '@/services/series.service'
import type { Series } from '@/services/types'
import { formatDisplayDate } from '@/utils/date'
import { useUiStore } from '@/stores/ui'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'

defineOptions({
  name: 'SeriesLibraryView',
})

const SERIES_PAGE_SIZE = 24

interface SeriesCardItem extends Series {
  game_count: number
  cover_image?: string | null
  cover_candidates?: string[]
  background_candidates?: string[]
  latest_updated_at?: string | null
}

const router = useRouter()
const uiStore = useUiStore()
const searchQuery = ref('')
const loadMoreSentinel = ref<HTMLElement | null>(null)

const {
  items: seriesCards,
  isLoading,
  isLoadingMore,
  hasLoadFailure,
  loadFirstPage,
} = useInfiniteScroll<SeriesCardItem>({
  pageSize: SERIES_PAGE_SIZE,
  sentinel: loadMoreSentinel,
  searchQuery,
  loadPage: async (params) => {
    const response = await seriesService.getSeriesPage({
      ...params,
      sort: 'name',
    })
    return {
      data: response.data as SeriesCardItem[],
      pagination: response.pagination,
    }
  },
  normalizeItems: (items) => items.map((item): SeriesCardItem => ({
        ...item,
        game_count: item.game_count || 0,
        cover_image: item.cover_image ?? null,
        cover_candidates: (item.cover_candidates || []).filter((value) => value.trim().length > 0).slice(0, 4),
        latest_updated_at: item.latest_updated_at ?? null,
      })),
  onError: (message) => uiStore.addAlert(message === '加载失败' ? '加载系列列表失败' : '加载更多系列失败', 'error'),
})

const openSeries = (id: number) => {
  router.push({ name: 'series-detail', params: { id: String(id) } })
}

const handleSearchSubmit = () => {
  void loadFirstPage()
}

const formatDate = (value: string) => formatDisplayDate(value)

onMounted(() => {
  void loadFirstPage()
})
</script>

<style scoped>
.series-library {
  --series-collage-bg: linear-gradient(135deg, rgba(12, 18, 30, 0.96), rgba(16, 20, 30, 0.82));
  --series-cover-overlay: linear-gradient(180deg, rgba(0, 0, 0, 0.02), rgba(0, 0, 0, 0.14));
  --series-placeholder-text: var(--color-text-on-dark);

}

.series-library__header {
  margin-bottom: 10px;
}

.series-library__title {
  margin: 0;
}

.series-library__subtitle {
  margin: 0;
}

.series-library__search {
  width: min(320px, 100%);
  border-radius: var(--radius-lg);
}

.series-library__search-body {
  width: 100%;
  padding: 12px;
  box-sizing: border-box;
}

.series-library__loading {
  padding: 64px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.series-library__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.series-library__infinite-scroll {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 48px;
  margin-top: 24px;
}

.series-card {
  position: relative;
  padding: 0;
  border-radius: var(--radius-lg);
  overflow: hidden;
  cursor: pointer;
  text-align: left;
  display: flex;
  flex-direction: column;
  height: 100%;
  content-visibility: auto;
  contain-intrinsic-size: auto 420px;
  transition: transform var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast);
  text-align: left;
}

.series-card:hover {
  border-color: var(--app-glass-border-hover);
  box-shadow: var(--app-glass-shadow-hover);
}

.series-card__cover {
  position: relative;
  aspect-ratio: 2 / 3;
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
}

.series-card__image,
.series-card__placeholder {
  width: 100%;
  height: 100%;
}

.series-card__collage {
  width: 100%;
  height: 100%;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  grid-template-rows: repeat(2, minmax(0, 1fr));
  gap: 2px;
  background: var(--series-collage-bg);
}

.series-card__collage-tile {
  position: relative;
  overflow: hidden;
}

.series-card__collage-image {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
  object-position: center 22%;
  transform: scale(1.02);
}

.series-card__image {
  object-fit: cover;
  object-position: center 22%;
  display: block;
}

.series-card__placeholder {
  display: grid;
  place-items: center;
  font-size: 48px;
  font-weight: 800;
  color: var(--series-placeholder-text);
}

.series-card__overlay {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: var(--series-cover-overlay);
}

.series-card__body {
  padding: 12px 16px;
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
  text-align: left;
}

.series-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1);
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.series-card__meta-row {
  margin-top: auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--color-text-3);
  font-size: 12px;
  line-height: 1.35;
}

@media (max-width: 1199px) {
  .series-library__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 991px) {
  .series-library__header {
    flex-direction: column;
    align-items: stretch;
  }

  .series-library__search {
    width: 100%;
  }

  .series-library__search-body {
    padding: 10px;
  }

  .series-library__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .series-library__meta {
    font-size: 13px;
  }

  .series-library__search-body {
    flex-direction: row;
    align-items: stretch;
  }

  .series-library__search-body .app-input-action-row__action {
    width: auto;
  }

  .series-library__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .series-card__body {
    padding: 10px 12px;
  }

  .series-card__meta-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}

@media (min-width: 1200px) {
  .series-library__grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}

@media (min-width: 1600px) {
  .series-library__grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}

@media (min-width: 2200px) {
  .series-library__grid {
    grid-template-columns: repeat(12, minmax(0, 1fr));
  }
}
</style>
