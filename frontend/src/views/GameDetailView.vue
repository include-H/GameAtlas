<template>
  <div v-if="game" class="game-detail">
    <div class="game-detail__container">
      <!-- Game Header Navigation & Title -->
      <div class="game-detail__header">
        <div class="header-content">
        <a-button class="app-text-action-btn back-button" type="text" @click="handleGoBack">
          <template #icon>
            <icon-left />
          </template>
          返回
        </a-button>
        
        <div class="header-info">
          <h1 class="header-title">{{ game.title }}</h1>
          <div class="header-actions">
            <a-button
              class="app-text-action-btn"
              type="text"
              :status="game.isFavorite ? 'danger' : undefined"
              @click="handleToggleFavorite"
            >
              <template #icon>
                <icon-heart-fill v-if="game.isFavorite" />
                <icon-heart v-else />
              </template>
              {{ game.isFavorite ? '已收藏' : '收藏' }}
            </a-button>

            <a-button
              v-if="canEdit"
              class="app-text-action-btn"
              type="text"
              @click="showEditModal = true"
            >
              <template #icon>
                <icon-edit />
              </template>
              编辑
            </a-button>
          </div>
        </div>
      </div>
      </div>

      <div ref="topSectionRef" class="game-detail__top">
        <div class="game-detail__content">
          <div class="game-detail__main">
            <screenshot-carousel
              :preview-videos="game.preview_videos?.map((item) => item.path) || []"
              :video-poster="game.banner_image || game.cover_image || null"
              :screenshots="game.screenshots.map((item) => item.path)"
              :alt="game.title"
              :height="carouselHeight"
            />
          </div>
        </div>

        <div class="game-detail__sidebar">
          <div class="sidebar-card sidebar-card--hero">
            <div class="sidebar-header-image">
              <img
                v-if="game.banner_image"
                :src="game.banner_image"
                :alt="game.title"
                class="sidebar-header-image__img"
              />
              <img
                v-else-if="game.cover_image"
                :src="game.cover_image"
                :alt="game.title"
                class="sidebar-header-image__img sidebar-header-image__img--contain"
              />
              <div v-else class="sidebar-header-image__placeholder">
                {{ game.title?.charAt(0) || '?' }}
              </div>
            </div>

            <div v-if="game.summary" class="sidebar-summary">
              {{ game.summary }}
            </div>
          </div>

          <div class="sidebar-card sidebar-card--meta">
            <div class="sidebar-info">
              <div v-if="game.series" class="sidebar-info__item">
                <span class="sidebar-info__label">系列</span>
                <span class="sidebar-info__value">{{ game.series.name }}</span>
              </div>
              <div v-if="game.developers && game.developers.length > 0" class="sidebar-info__item">
                <span class="sidebar-info__label">开发商</span>
                <span class="sidebar-info__value">{{ developerNames }}</span>
              </div>
              <div v-if="game.publishers && game.publishers.length > 0" class="sidebar-info__item">
                <span class="sidebar-info__label">发行商</span>
                <span class="sidebar-info__value">{{ publisherNames }}</span>
              </div>
              <div v-if="game.release_date" class="sidebar-info__item">
                <span class="sidebar-info__label">发行日期</span>
                <span class="sidebar-info__value">{{ formatDate(game.release_date) }}</span>
              </div>
              <div v-if="game.engine" class="sidebar-info__item">
                <span class="sidebar-info__label">游戏引擎</span>
                <span class="sidebar-info__value">{{ game.engine }}</span>
              </div>
              <div
                v-if="game.platforms && game.platforms.length > 0"
                :class="[
                  'sidebar-info__item',
                  { 'sidebar-info__item--wide': shouldSpanMetadataRow(game.platforms) }
                ]"
              >
                <span class="sidebar-info__label">平台</span>
                <div class="sidebar-info__value">
                  <a-space wrap>
                    <a-tag v-for="platform in game.platforms" :key="platform.id">
                      {{ platform.name }}
                    </a-tag>
                  </a-space>
                </div>
              </div>
              <div
                v-for="group in game.tag_groups || []"
                :key="group.id"
                :class="[
                  'sidebar-info__item',
                  { 'sidebar-info__item--wide': shouldSpanMetadataRow(group.tags) }
                ]"
              >
                <span class="sidebar-info__label">{{ group.name }}</span>
                <div class="sidebar-info__value">
                  <a-space wrap>
                    <a-tag v-for="tag in group.tags" :key="tag.id">
                      {{ tag.name }}
                    </a-tag>
                  </a-space>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="game-detail__download-section">
        <div v-if="versions.length > 0" class="download-version-panel">
          <div class="download-version-list">
            <div
              v-for="version in versions"
              :key="version.id"
              class="download-version-item"
            >
              <div class="version-info">
                <div class="version-name">
                  <span class="version-name__text">{{ version.version }}</span>
                  <span v-if="version.isLatest" class="version-badge version-badge--latest">最新版本</span>
                </div>
                <div class="version-meta">
                  <span class="version-meta-pill">
                    <span class="version-meta-pill__label">大小</span>
                    <span class="version-size">{{ formatSize(version.size) }}</span>
                  </span>
                  <span v-if="version.releaseDate" class="version-meta-pill">
                    <span class="version-meta-pill__label">日期</span>
                    <span class="version-date">{{ formatDate(version.releaseDate) }}</span>
                  </span>
                </div>
              </div>
              <div class="version-actions">
                <a-button
                  type="primary"
                  @click="handleDownloadVersion(version)"
                >
                  <template #icon>
                    <icon-download />
                  </template>
                  下载
                </a-button>
                <a-button
                  v-if="version.canLaunch"
                  class="app-text-action-btn"
                  type="text"
                  @click.stop="handleDownloadLaunchScript(version)"
                >
                  开始游玩
                </a-button>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="download-empty">
          <p class="download-empty-title">暂无可下载版本</p>
        </div>
      </div>

      <div class="game-detail__wiki-section">
        <!-- Wiki Content with TOC -->
        <div v-if="hasWikiContent" class="game-detail__wiki-wrapper">
          <!-- Wiki TOC Sidebar -->
          <wiki-toc />

          <!-- Wiki Main Content -->
          <a-card class="game-detail__card game-detail__wiki-card">
            <template #title>
              <div class="game-detail__wiki-title">
                <span>关于这款游戏</span>
                <a-button
                  v-if="canEdit"
                  class="app-text-action-btn"
                  type="text"
                  size="small"
                  @click="openWikiEditor"
                >
                  <template #icon>
                    <icon-edit />
                  </template>
                  编辑Wiki
                </a-button>
              </div>
            </template>
            <markdown-renderer :content="wiki?.content || ''" />
          </a-card>
        </div>

        <!-- No Wiki Placeholder -->
        <a-card v-else class="game-detail__card game-detail__wiki-placeholder">
          <div class="game-detail__no-wiki">
            <p class="game-detail__no-wiki-text">暂无 Wiki</p>
            <a-button
              v-if="canEdit"
              class="app-text-action-btn"
              type="text"
              size="small"
              @click="openWikiEditor"
            >
              创建Wiki页面
            </a-button>
          </div>
        </a-card>
      </div>
      </div>
    </div>

  <!-- Loading State -->
  <div v-else class="game-detail__loading">
    <a-spin :size="24" />
    <p class="game-detail__loading-text">加载中...</p>
  </div>

  <!-- Edit Game Modal -->
  <edit-game-modal
    v-model:visible="showEditModal"
    :game="game"
    @success="handleEditSuccess"
  />

</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick, defineAsyncComponent } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useGamesStore } from '@/stores/games'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import wikiService, { type WikiDocumentResponse } from '@/services/wiki.service'
import downloadService from '@/services/download.service'
import type { GameVersion } from '@/services/types'
import { getHttpStatus } from '@/utils/http-error'
import ScreenshotCarousel from '@/components/ScreenshotCarousel.vue'
import EditGameModal from '@/components/EditGameModal.vue'
import WikiToc from '@/components/WikiToc.vue'
import { useNamedRouteGuard, watchRouteParamWhenActive } from '@/composables/useNamedRouteGuard'
import { createDetailRouteQuery, resolveReturnRoute } from '@/utils/navigation'
import {
  IconEdit,
  IconLeft,
  IconDownload,
  IconHeart,
  IconHeartFill
} from '@arco-design/web-vue/es/icon'

const route = useRoute()
const router = useRouter()
const gamesStore = useGamesStore()
const authStore = useAuthStore()
const uiStore = useUiStore()
const { runWhenActive } = useNamedRouteGuard(route, 'game-detail')
const { isAdmin } = storeToRefs(authStore)

const game = computed(() => gamesStore.currentGame)
const versions = computed(() => gamesStore.currentVersions)
const wiki = ref<WikiDocumentResponse | null>(null)
const showEditModal = ref(false)
const topSectionRef = ref<HTMLElement | null>(null)
const topSectionHeight = ref<number | undefined>(undefined)
const isDesktopTopLayout = ref(false)
let topSectionObserver: ResizeObserver | null = null
const MarkdownRenderer = defineAsyncComponent(() => import('@/components/MarkdownRenderer.vue'))

const developerNames = computed(() => (game.value?.developers || []).map((item) => item.name).join(' / '))
const publisherNames = computed(() => (game.value?.publishers || []).map((item) => item.name).join(' / '))
const hasWikiContent = computed(() => Boolean(wiki.value?.content?.trim()))

const canEdit = computed(() => isAdmin.value)

const formatDate = (dateStr: string) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const formatSize = (bytes: number) => {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = bytes
  let unitIndex = 0

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex++
  }

  return `${size.toFixed(1)} ${units[unitIndex]}`
}

const shouldSpanMetadataRow = (items?: { length: number } | null) => {
  return (items?.length || 0) > 2
}

const handleDownloadVersion = async (version: GameVersion) => {
  if (!game.value?.public_id) return

  try {
    await downloadService.startDownload(game.value.public_id, version.id)
    uiStore.addAlert(`已开始下载 ${version.version}`, 'success')
  } catch {
    uiStore.addAlert('下载启动失败', 'error')
  }
}

const handleDownloadLaunchScript = (version: GameVersion) => {
  if (!game.value?.public_id) return

  try {
    downloadService.downloadLaunchScript(game.value.public_id, version.id)
    uiStore.addAlert(`已为 ${version.version} 生成启动脚本`, 'success')
  } catch {
    uiStore.addAlert('开始游玩失败', 'error')
  }
}

const handleEditSuccess = async () => {
  // Reload game data after edit
  if (game.value?.public_id) {
    await Promise.all([
      gamesStore.fetchGame(game.value.public_id),
      gamesStore.fetchGameVersions(game.value.public_id),
    ])
  }
}

const handleGoBack = () => {
  router.push(resolveReturnRoute(route, { name: 'games' }))
}

const openWikiEditor = () => {
  if (!game.value?.public_id) return
  router.push({
    name: 'wiki-edit',
    params: { publicId: game.value.public_id },
    query: createDetailRouteQuery(route),
  })
}

const handleToggleFavorite = async () => {
  if (!game.value?.public_id) return
  try {
    await gamesStore.toggleFavorite(game.value.public_id)
    uiStore.addAlert('收藏已更新', 'success')
  } catch {
    uiStore.addAlert('更新收藏失败', 'error')
  }
}

const carouselHeight = computed(() => {
  if (!isDesktopTopLayout.value) return undefined
  if (!topSectionHeight.value) return undefined
  return Math.max(Math.round(topSectionHeight.value), 420)
})

const disconnectTopSectionObserver = () => {
  if (topSectionObserver) {
    topSectionObserver.disconnect()
    topSectionObserver = null
  }
}

const loadGameDetail = async (gameId: string) => {
  await runWhenActive(async () => {
    try {
      await Promise.all([
        gamesStore.fetchGame(gameId),
        gamesStore.fetchGameVersions(gameId),
      ])

      wiki.value = null
      try {
        wiki.value = await wikiService.getWikiPage(gameId)
      } catch {
        // Wiki doesn't exist
      }
    } catch (error) {
      const status = getHttpStatus(error)
      if (status === 404) {
        router.replace({ name: 'not-found' })
        return
      }
      uiStore.addAlert('加载游戏详情失败', 'error')
    }
  })
}

watchRouteParamWhenActive(
  route,
  'game-detail',
  'publicId',
  async (gameId) => {
    showEditModal.value = false
    await loadGameDetail(gameId)
  },
)

const syncTopSectionHeight = () => {
  const element = topSectionRef.value
  if (!element) {
    topSectionHeight.value = undefined
    return
  }

  if (typeof window !== 'undefined') {
    isDesktopTopLayout.value = window.innerWidth > 992
  }
  if (!isDesktopTopLayout.value) {
    topSectionHeight.value = undefined
    return
  }

  const nextHeight = Math.round(element.getBoundingClientRect().height)
  topSectionHeight.value = nextHeight > 0 ? nextHeight : undefined
}

const setupTopSectionObserver = async () => {
  await nextTick()
  disconnectTopSectionObserver()
  syncTopSectionHeight()

  if (!topSectionRef.value || typeof ResizeObserver === 'undefined') return

  topSectionObserver = new ResizeObserver(() => {
    syncTopSectionHeight()
  })
  topSectionObserver.observe(topSectionRef.value)
}

onMounted(() => {
  if (typeof window !== 'undefined') {
    isDesktopTopLayout.value = window.innerWidth > 992
    window.addEventListener('resize', syncTopSectionHeight, { passive: true })
  }
  void setupTopSectionObserver()
})

watch(
  game,
  () => {
    void setupTopSectionObserver()
  },
  { flush: 'post' },
)

onUnmounted(() => {
  disconnectTopSectionObserver()
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', syncTopSectionHeight)
  }
})

</script>

<style scoped>
.game-detail {
  animation: fadeIn 0.3s ease;
  max-width: 100%;
  margin: 0;
  padding: 0;
  position: relative;
  z-index: 1;
}

.game-detail__container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 16px;
  width: 100%;
  box-sizing: border-box;
}

.game-detail__top {
  display: grid;
  grid-template-columns: minmax(0, 68fr) minmax(280px, 30fr);
  column-gap: 10px;
  align-items: stretch;
  margin-top: 10px;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Game Header (Title & Nav) */
.game-detail__header {
  position: relative;
  z-index: 10;
  padding: 24px 0 12px;
}

.header-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.header-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.header-title {
  font-size: clamp(28px, 3.2vw, 36px);
  font-weight: 700;
  color: #fff;
  margin: 0;
  letter-spacing: -0.5px;
  line-height: 1.15;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.back-button {
  align-self: flex-start;
}

/* Main Content Layout - 固定比例: 左侧70%, 右侧30% */
.game-detail__content {
  width: 100%;
  display: flex;
  min-width: 0;
}

.game-detail__main {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  min-height: 420px;
  height: 100%;
}

.game-detail__main > :deep(.screenshot-carousel) {
  min-width: 0;
  margin-bottom: 0;
  width: 100%;
  height: 100%;
}

.game-detail__main > :deep(.screenshot-carousel__viewport) {
  border-radius: var(--radius-lg);
  min-height: 420px;
}

.game-detail__main > :deep(.screenshot-carousel__main) {
  min-height: 420px;
}

.game-detail__main > :deep(.screenshot-carousel__filmstrip) {
  border-radius: var(--radius-lg);
}

.game-detail__main > :deep(.screenshot-carousel__filmstrip-inner) {
  padding-left: 12px;
  padding-right: 12px;
}

.game-detail__card {
  margin-bottom: 0;
}

.game-detail__card:last-child {
  margin-bottom: 0;
}

/* Wiki Section - Full Width */
.game-detail__wiki-section {
  padding: 12px 0;
  min-width: 0;
}

.game-detail__download-section {
  padding: 10px 0 0;
  min-width: 0;
}

.game-detail__wiki-wrapper {
  display: flex;
  align-items: start;
  gap: 10px;
  max-width: 100%;
  margin: 0;
  padding: 0;
  width: 100%;
  box-sizing: border-box;
}

.game-detail__wiki-card {
  flex: 1 1 auto;
  min-width: 0;
  border-radius: var(--radius-lg);
  width: auto;
}

.game-detail__wiki-placeholder {
  max-width: 100%;
  margin: 0;
  border-radius: var(--radius-lg);
  width: 100%;
  box-sizing: border-box;
}

.game-detail__wiki-placeholder :deep(.arco-card-body) {
  padding: 0;
}

/* Wiki Card Styling */
.game-detail__card :deep(.arco-card-header) {
  padding: 12px 20px;
  border-bottom: 1px solid var(--color-border-2);
}

.game-detail__card :deep(.arco-card-header-title) {
  font-size: 16px;
  font-weight: 600;
}

.game-detail__card :deep(.arco-card-body) {
  padding: 20px;
}

/* Wiki Title */
.game-detail__wiki-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

/* No Wiki */
.game-detail__no-wiki {
  text-align: center;
  padding: 10px 12px;
  min-height: 46px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.game-detail__no-wiki-text {
  color: var(--color-text-3);
  margin: 0;
  font-size: 12px;
  line-height: 1;
}

/* Sidebar - Steam Style - 固定占30%宽度 */
.game-detail__sidebar {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  height: 100%;
}

.sidebar-card {
  background: var(--app-card-surface);
  border: 1px solid var(--app-card-border);
  border-radius: var(--radius-lg);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-soft);
}

.sidebar-card--meta {
  padding-top: 4px;
  flex: 1;
}

/* 侧边栏横幅区域 (Steam Header Image) */
.sidebar-header-image {
  width: 100%;
  aspect-ratio: 460 / 215; /* Standard Steam header ratio */
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: transparent;
}

.sidebar-header-image__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.sidebar-header-image__img--contain {
  object-fit: contain;
}

.sidebar-header-image__placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 3rem;
  color: rgba(255, 255, 255, 0.1);
  background: transparent;
}

/* Sidebar Summary */
.sidebar-summary {
  padding: 12px;
  font-size: 13px;
  color: var(--color-text-2);
  line-height: 1.5;
  background: transparent;
}

/* Sidebar Info */
.sidebar-info {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  grid-auto-rows: min-content;
  align-content: start;
  gap: 12px 16px;
  padding: 12px;
}

.sidebar-info__item {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.sidebar-info__item--wide {
  grid-column: 1 / -1;
}

.sidebar-info__item:last-child:nth-child(odd) {
  grid-column: 1 / -1;
}

.sidebar-info__label {
  font-size: 11px;
  color: var(--color-text-3);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 4px;
}

.sidebar-info__value {
  font-size: 13px;
  color: var(--color-text-1);
  line-height: 1.4;
  min-width: 0;
  word-break: break-word;
}

.sidebar-info__value :deep(.arco-space) {
  display: flex;
  flex-wrap: wrap;
  row-gap: 6px;
  column-gap: 6px;
}

.sidebar-info__value :deep(.arco-space-item) {
  display: inline-flex;
  max-width: 100%;
}

.sidebar-info__value :deep(.arco-tag) {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  height: auto;
  min-height: 24px;
  padding: 4px 8px;
  line-height: 1.35;
  white-space: normal;
  word-break: break-word;
}

/* Loading */
.game-detail__loading {
  text-align: center;
  padding: 32px 24px;
}

.game-detail__loading-text {
  margin-top: 12px;
  color: var(--color-text-3);
}

.download-version-panel {
  padding: 14px;
  border-radius: var(--radius-lg);
  background: var(--app-card-surface);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  box-shadow: var(--shadow-soft);
}

.download-version-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 156px;
  overflow-y: auto;
  padding-right: 2px;
  background: transparent;
}

.download-version-item {
  margin-bottom: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 24px;
  background: transparent;
  border: none;
  box-shadow: none;
  border-radius: 10px;
}

.download-version-item + .download-version-item {
  border-top: 1px solid var(--color-border-1);
  padding-top: 8px;
}

.version-info {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  padding-left: 6px;
}

.version-name {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 0;
}

.version-name__text {
  font-size: 12px;
  font-weight: 700;
  color: var(--color-text-1);
  line-height: 1;
}

.version-badge {
  display: inline-flex;
  align-items: center;
  min-height: 14px;
  padding: 0 5px;
  border-radius: var(--radius-pill);
  font-size: 10px;
  line-height: 1;
  font-weight: 700;
}

.version-badge--latest {
  background: var(--color-fill-2);
  color: var(--color-text-2);
}

.version-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.version-meta-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 14px;
  padding: 0 5px;
  border-radius: var(--radius-pill);
  background: var(--color-fill-2);
  color: var(--color-text-2);
  font-size: 10px;
  line-height: 1;
}

.version-meta-pill__label {
  color: var(--color-text-3);
}

.version-size {
  color: inherit;
}

.version-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

/* Download Empty State */
.download-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  min-height: 46px;
  padding: 8px 12px;
  border-radius: var(--radius-lg);
  background: var(--app-card-surface);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  box-shadow: var(--shadow-soft);
  gap: 4px;
}

.download-empty-title {
  margin: 0;
  font-size: 12px;
  font-weight: 700;
  color: var(--color-text-1);
  line-height: 1;
}

/* Responsive - Arco Design Breakpoints */
/* lg: 992px */
@media (max-width: 992px) {
  .header-info {
    align-items: flex-start;
    flex-direction: column;
    gap: 12px;
  }

  .header-title {
    font-size: 28px;
  }

  .game-detail__top {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .game-detail__sidebar {
    width: 100%;
    height: auto;
    gap: 10px;
  }

  .game-detail__content,
  .game-detail__main,
  .game-detail__download-section,
  .game-detail__wiki-section {
    width: 100%;
  }

  .game-detail__content,
  .game-detail__main {
    min-height: auto;
    height: auto;
  }

  .game-detail__main > :deep(.screenshot-carousel) {
    height: auto;
  }

  .game-detail__wiki-wrapper {
    flex-direction: column;
  }

  .game-detail__download-section {
    padding-top: 10px;
  }

  .game-detail__wiki-section {
    padding-top: 10px;
    padding-bottom: 10px;
  }

  .game-detail__main > :deep(.screenshot-carousel__viewport) {
    min-height: 320px;
  }

  .game-detail__main > :deep(.screenshot-carousel__main) {
    min-height: 320px;
  }

  .sidebar-header-image {
    aspect-ratio: 21/9;
    max-height: 200px;
  }

  .sidebar-info {
    grid-template-columns: 1fr;
  }

  .download-version-panel {
    padding: 12px;
    border-radius: 18px;
  }

  .download-empty {
    min-height: 46px;
    padding: 8px 10px;
  }

  .download-version-item {
    min-height: auto;
    align-items: flex-start;
    flex-direction: column;
    gap: 6px;
    padding: 8px;
  }

  .download-version-item + .download-version-item {
    padding-top: 10px;
  }

  .version-actions {
    width: 100%;
  }
  .download-version-list {
    max-height: 220px;
  }

  .version-info {
    width: 100%;
    align-items: flex-start;
    flex-direction: column;
    gap: 6px;
  }
}
</style>
