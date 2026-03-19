<template>
  <div class="series-library">
    <div class="series-library__header">
      <div>
        <h1 class="series-library__title text-gradient">系列库</h1>
        <p class="series-library__subtitle">按系列浏览，再进入对应作品集合。</p>
      </div>
      <a-input-search
        v-model="searchQuery"
        class="series-library__search"
        placeholder="搜索系列"
        allow-clear
      />
    </div>

    <div v-if="isLoading" class="series-library__loading">
      <a-spin :size="24" />
      <p>加载系列中...</p>
    </div>

    <template v-else>
      <div class="series-library__meta">
        共 {{ filteredSeries.length }} 个系列
      </div>

      <div v-if="filteredSeries.length > 0" class="series-library__grid">
        <button
          v-for="series in filteredSeries"
          :key="series.id"
          type="button"
          class="series-card hover-lift"
          @click="openSeries(series.id)"
        >
          <div class="series-card__cover">
            <img
              v-if="series.cover_image"
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
        </button>
      </div>

      <a-empty v-else description="暂无系列数据" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { seriesService } from '@/services/series.service'
import type { Series } from '@/services/types'

defineOptions({
  name: 'SeriesLibraryView',
})

interface SeriesCardItem extends Series {
  game_count: number
  cover_image?: string | null
  latest_updated_at?: string | null
}

const router = useRouter()
const isLoading = ref(false)
const searchQuery = ref('')
const seriesCards = ref<SeriesCardItem[]>([])

const filteredSeries = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  if (!keyword) return seriesCards.value
  return seriesCards.value.filter((item) => item.name.toLowerCase().includes(keyword))
})

const loadSeries = async () => {
  isLoading.value = true
  try {
    const allSeries = await seriesService.getAllSeries()
    seriesCards.value = allSeries
      .filter((item) => (item.game_count || 0) > 0)
      .map((item) => ({
        ...item,
        game_count: item.game_count || 0,
        cover_image: item.cover_image ?? null,
        latest_updated_at: item.latest_updated_at ?? null,
      }))
      .sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'))
  } finally {
    isLoading.value = false
  }
}

const openSeries = (id: number) => {
  router.push({ name: 'series-detail', params: { id: String(id) } })
}

const formatDate = (value: string) => {
  if (!value) return ''
  const date = new Date(value)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

onMounted(() => {
  loadSeries()
})
</script>

<style scoped>
.series-library {
  animation: fadeInUp 0.4s cubic-bezier(0.2, 0.8, 0.2, 1) forwards;
}

.series-library__header {
  display: flex;
  justify-content: space-between;
  align-items: end;
  gap: 16px;
  margin-bottom: 20px;
}

.series-library__title {
  margin: 0;
  font-size: 32px;
  font-weight: 800;
}

.series-library__subtitle {
  margin: 6px 0 0;
  color: var(--color-text-3);
}

.series-library__search {
  width: min(320px, 100%);
}

.series-library__meta {
  margin-bottom: 16px;
  color: var(--color-text-3);
  font-size: 14px;
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

.series-card {
  position: relative;
  padding: 0;
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-lg);
  background: var(--color-bg-2);
  overflow: hidden;
  cursor: pointer;
  text-align: left;
  display: flex;
  flex-direction: column;
  height: 100%;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  transition: all var(--transition-fast);
}

.series-card:hover {
  border-color: rgba(26, 159, 255, 0.3);
  box-shadow: var(--shadow-hover);
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
  color: rgba(255, 255, 255, 0.92);
}

.series-card__overlay {
  position: absolute;
  inset: 0;
  background: transparent;
}

.series-card__body {
  padding: 12px 16px;
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  gap: 4px;
}

.series-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1);
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.series-card__meta-row {
  margin-top: auto;
  display: flex;
  justify-content: space-between;
  gap: 8px;
  color: var(--color-text-3);
  font-size: 12px;
  line-height: 1.35;
}

.text-gradient {
  background: linear-gradient(135deg, var(--color-primary-light-3), var(--color-primary-6));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(15px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
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

  .series-library__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .series-library__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
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
