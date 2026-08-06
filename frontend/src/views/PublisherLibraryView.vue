<template>
  <div class="publisher-library">
    <div class="publisher-library__header page-hero">
      <div class="page-hero__content">
        <h1 class="publisher-library__title page-hero__title text-gradient">发行商库</h1>
        <p class="publisher-library__subtitle page-hero__subtitle">按发行商，整理每一份游戏收藏。</p>
      </div>
      <div class="publisher-library__search app-glass-surface">
        <div class="publisher-library__search-body app-input-action-row">
          <a-input
            v-model="searchQuery"
            class="app-input-action-row__field"
            placeholder="搜索发行商"
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

    <div v-if="isLoading" class="publisher-library__loading">
      <a-spin :size="24" />
      <p>加载发行商中...</p>
    </div>

    <template v-else>
      <a-empty v-if="hasLoadFailure" description="发行商列表加载失败，请稍后重试。" />

      <template v-if="!hasLoadFailure && publisherCards.length > 0">
        <div class="publisher-library__grid">
          <div
            v-for="publisher in publisherCards"
            :key="publisher.id"
            class="publisher-card hover-lift app-glass-surface app-glass-surface--interactive"
            role="button"
            tabindex="0"
            @click="openPublisher(publisher.id)"
            @keydown.enter="openPublisher(publisher.id)"
            @keydown.space.prevent="openPublisher(publisher.id)"
          >
            <div class="publisher-card__cover">
              <div
                v-if="(publisher.game_count || 0) >= 4 && publisher.cover_candidates && publisher.cover_candidates.length >= 4"
                class="publisher-card__collage"
              >
                <div
                  v-for="(cover, index) in publisher.cover_candidates.slice(0, 4)"
                  :key="`${publisher.id}-${index}`"
                  class="publisher-card__collage-tile"
                >
                  <img
                    :src="cover"
                    :alt="`${publisher.name}-${index + 1}`"
                    class="publisher-card__collage-image"
                    loading="lazy"
                    decoding="async"
                  />
                </div>
              </div>
              <img
                v-else-if="publisher.cover_image"
                :src="publisher.cover_image"
                :alt="publisher.name"
                class="publisher-card__image"
                loading="lazy"
                decoding="async"
              />
              <div v-else class="publisher-card__placeholder">
                {{ publisher.name.charAt(0) || '?' }}
              </div>
              <div class="publisher-card__overlay" />
            </div>
            <div class="publisher-card__body">
              <div class="publisher-card__title">{{ publisher.name }}</div>
              <div class="publisher-card__meta-row">
                <span>{{ publisher.game_count }} 部作品</span>
                <span v-if="publisher.latest_updated_at">{{ formatDate(publisher.latest_updated_at) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div
          ref="loadMoreSentinel"
          class="publisher-library__infinite-scroll"
        >
          <a-spin v-if="isLoadingMore" :size="20" />
        </div>
      </template>

      <a-empty v-else description="暂无发行商数据" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { IconSearch } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'
import { publishersService } from '@/services/publishers.service'
import type { Publisher } from '@/services/types'
import { formatDisplayDate } from '@/utils/date'
import { useUiStore } from '@/stores/ui'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'

defineOptions({
  name: 'PublisherLibraryView',
})

const PUBLISHER_PAGE_SIZE = 24

interface PublisherCardItem extends Publisher {
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
  items: publisherCards,
  isLoading,
  isLoadingMore,
  hasLoadFailure,
  loadFirstPage,
} = useInfiniteScroll<PublisherCardItem>({
  pageSize: PUBLISHER_PAGE_SIZE,
  sentinel: loadMoreSentinel,
  searchQuery,
  loadPage: async (params) => {
    const response = await publishersService.getPublishersPage({
      ...params,
      sort: 'name',
    })
    return {
      data: response.data as PublisherCardItem[],
      pagination: response.pagination,
    }
  },
  normalizeItems: (items) => items.map((item): PublisherCardItem => ({
        ...item,
        game_count: item.game_count || 0,
        cover_image: item.cover_image ?? null,
        cover_candidates: (item.cover_candidates || []).filter((value) => value.trim().length > 0).slice(0, 4),
        latest_updated_at: item.latest_updated_at ?? null,
      })),
  onError: (message) => uiStore.addAlert(message === '加载失败' ? '加载发行商列表失败' : '加载更多发行商失败', 'error'),
})

const openPublisher = (id: number) => {
  router.push({ name: 'publisher-detail', params: { id: String(id) } })
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
.publisher-library {
  --publisher-collage-bg: linear-gradient(135deg, rgba(12, 18, 30, 0.96), rgba(16, 20, 30, 0.82));
  --publisher-cover-overlay: linear-gradient(180deg, rgba(0, 0, 0, 0.02), rgba(0, 0, 0, 0.14));
  --publisher-placeholder-text: var(--color-text-on-dark);

}

.publisher-library__header {
  margin-bottom: 10px;
}

.publisher-library__title,
.publisher-library__subtitle {
  margin: 0;
}

.publisher-library__search {
  width: min(320px, 100%);
  border-radius: var(--radius-lg);
}

.publisher-library__search-body {
  width: 100%;
  padding: 12px;
  box-sizing: border-box;
}

.publisher-library__meta {
  margin-bottom: 10px;
  color: var(--color-text-3);
  font-size: 14px;
}

.publisher-library__loading {
  padding: 64px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.publisher-library__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.publisher-library__infinite-scroll {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 48px;
  margin-top: 24px;
}

.publisher-card {
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
}

.publisher-card:hover {
  border-color: var(--app-glass-border-hover);
  box-shadow: var(--app-glass-shadow-hover);
}

.publisher-card__cover {
  position: relative;
  aspect-ratio: 2 / 3;
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
}

.publisher-card__image,
.publisher-card__placeholder {
  width: 100%;
  height: 100%;
}

.publisher-card__collage {
  width: 100%;
  height: 100%;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  grid-template-rows: repeat(2, minmax(0, 1fr));
  gap: 2px;
  background: var(--publisher-collage-bg);
}

.publisher-card__collage-tile {
  position: relative;
  overflow: hidden;
}

.publisher-card__collage-image {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
  object-position: center 22%;
  transform: scale(1.02);
}

.publisher-card__image {
  object-fit: cover;
  object-position: center 22%;
  display: block;
}

.publisher-card__placeholder {
  display: grid;
  place-items: center;
  font-size: 48px;
  font-weight: 800;
  color: var(--publisher-placeholder-text);
}

.publisher-card__overlay {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: var(--publisher-cover-overlay);
}

.publisher-card__body {
  padding: 12px 16px;
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
  text-align: left;
}

.publisher-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1);
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.publisher-card__meta-row {
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
  .publisher-library__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 991px) {
  .publisher-library__header {
    flex-direction: column;
    align-items: stretch;
  }

  .publisher-library__search {
    width: 100%;
  }

  .publisher-library__search-body {
    padding: 10px;
  }

  .publisher-library__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .publisher-library__meta {
    font-size: 13px;
  }

  .publisher-library__search-body {
    flex-direction: row;
    align-items: stretch;
  }

  .publisher-library__search-body .app-input-action-row__action {
    width: auto;
  }

  .publisher-library__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .publisher-card__body {
    padding: 10px 12px;
  }

  .publisher-card__meta-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}

@media (min-width: 1200px) {
  .publisher-library__grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}

@media (min-width: 1600px) {
  .publisher-library__grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}

@media (min-width: 2200px) {
  .publisher-library__grid {
    grid-template-columns: repeat(12, minmax(0, 1fr));
  }
}
</style>
