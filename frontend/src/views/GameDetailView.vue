<template>
  <div v-if="game" class="game-detail">
    <div class="game-detail__container">
      <!-- Game Header Navigation & Title -->
      <div class="game-detail__header">
        <div class="header-content">
        <a-button class="header-back-btn" type="text" @click="handleGoBack">
          <template #icon>
            <icon-left />
          </template>
          返回
        </a-button>
        
        <div class="header-info">
          <h1 class="header-title">{{ game.title }}</h1>
          <div class="header-actions">
            <a-button 
              type="text" 
              class="header-favorite-btn"
              :class="{ 'is-favorite': game.isFavorite }"
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
              type="text" 
              class="header-edit-btn"
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

	      <!-- Main Content -->
	      <div class="game-detail__content">
	        <div class="game-detail__main">
	        <!-- Screenshot Carousel -->
	        <screenshot-carousel
	          :preview-video="game.preview_video?.path || null"
	          :preview-videos="game.preview_videos?.map((item) => item.path) || []"
	          :video-poster="game.banner_image || game.cover_image || null"
	          :screenshots="game.screenshots || []"
	          :alt="game.title"
	          :height="desktopTopSectionHeight"
        />
      </div>

      <!-- Sidebar - Steam Style -->
      <div class="game-detail__sidebar">
        <div class="sidebar-card">
          <div class="sidebar-card__inner" ref="sidebarCardInnerRef">
          <!-- Header Banner (Steam Style) -->
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
          
          <!-- Game Summary -->
          <div v-if="game.summary" class="sidebar-summary">
            {{ game.summary }}
          </div>
          
          <!-- Download Button -->
          <div class="sidebar-actions">
            <a-button
              type="primary"
              size="large"
              long
              @click="showDownloadModal = true"
            >
              <template #icon>
                <icon-download />
              </template>
              下载游戏
            </a-button>
          </div>
          
          <!-- Game Info -->
          <div class="sidebar-info">
            <div v-if="game.series && game.series.length > 0" class="sidebar-info__item">
              <span class="sidebar-info__label">系列</span>
              <span class="sidebar-info__value">{{ game.series[0].name }}</span>
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
            <div v-if="game.platforms && game.platforms.length > 0" class="sidebar-info__item">
              <span class="sidebar-info__label">平台</span>
              <div class="sidebar-info__value">
                <a-space wrap>
                  <a-tag v-for="platform in game.platforms" :key="platform">
                    {{ platform }}
                  </a-tag>
                </a-space>
              </div>
            </div>
            <div v-else-if="game.platform" class="sidebar-info__item">
              <span class="sidebar-info__label">平台</span>
              <span class="sidebar-info__value">{{ game.platform }}</span>
            </div>
          </div>
          </div>
        </div>
      </div>

      </div>

      <!-- Wiki Section - Full Width -->
      <div class="game-detail__wiki-section">
        <!-- Wiki Content with TOC -->
        <div v-if="wiki" class="game-detail__wiki-wrapper">
          <!-- Wiki TOC Sidebar -->
          <wiki-toc />

          <!-- Wiki Main Content -->
          <a-card class="game-detail__card game-detail__wiki-card">
            <template #title>
              <div class="game-detail__wiki-title">
                <span>关于这款游戏</span>
                <a-button
                  v-if="canEdit"
                  type="text"
                  size="small"
                  @click="router.push({ name: 'wiki-edit', params: { gameId: String(game.id) } })"
                >
                  <template #icon>
                    <icon-edit />
                  </template>
                  编辑Wiki
                </a-button>
              </div>
            </template>
            <markdown-renderer :content="wiki.content || ''" />
          </a-card>
        </div>

        <!-- No Wiki Placeholder -->
        <a-card v-else class="game-detail__card game-detail__wiki-placeholder">
          <div class="game-detail__no-wiki">
            <icon-file class="game-detail__no-wiki-icon" />
            <p class="game-detail__no-wiki-text">暂无游戏介绍</p>
            <a-button
              v-if="canEdit"
              type="primary"
              @click="router.push({ name: 'wiki-edit', params: { gameId: String(game.id) } })"
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

  <!-- Download Version Modal -->
  <a-modal
    v-model:visible="showDownloadModal"
    title="选择游戏版本"
    :footer="false"
    :width="500"
  >
    <div v-if="versions.length > 0" class="download-version-list">
      <div
        v-for="version in versions"
        :key="version.id"
        class="download-version-item"
        @click="handleDownloadVersion(version)"
      >
        <div class="version-info">
          <div class="version-name">
            {{ version.version }}
            <a-tag v-if="version.isLatest" size="small" color="arcoblue">最新</a-tag>
          </div>
          <div class="version-meta">
            <span class="version-size">{{ formatSize(version.size) }}</span>
            <span v-if="version.releaseDate" class="version-date">{{ formatDate(version.releaseDate) }}</span>
          </div>
        </div>
        <a-button type="primary" size="small">
          <template #icon>
            <icon-download />
          </template>
          下载
        </a-button>
      </div>
    </div>
    <div v-else class="download-empty">
      <icon-file class="download-empty-icon" />
      <p class="download-empty-text">暂无可下载版本</p>
    </div>
  </a-modal>

  <!-- Ambient Background Effect (Subtle, Bottom) -->
  <div 
    class="game-detail__ambient-bg"
    :style="heroBackgroundStyle"
  >
    <div class="ambient-overlay"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import wikiService, { type WikiContent } from '@/services/wiki.service'
import downloadService from '@/services/download.service'
import type { GameVersion } from '@/services/types'
import ScreenshotCarousel from '@/components/ScreenshotCarousel.vue'
import MarkdownRenderer from '@/components/MarkdownRenderer.vue'
import EditGameModal from '@/components/EditGameModal.vue'
import WikiToc from '@/components/WikiToc.vue'
import {
  IconEdit,
  IconLeft,
  IconFile,
  IconDownload,
  IconHeart,
  IconHeartFill
} from '@arco-design/web-vue/es/icon'

const route = useRoute()
const router = useRouter()
const gamesStore = useGamesStore()
const uiStore = useUiStore()

const game = computed(() => gamesStore.currentGame)
const versions = computed(() => gamesStore.currentVersions)
const wiki = ref<WikiContent | null>(null)
const showEditModal = ref(false)
const showDownloadModal = ref(false)
const ambientBackgroundUrl = ref('')

const pickAmbientBackground = (currentGame: typeof game.value) => {
  if (!currentGame) return ''

  const screenshots = (currentGame.screenshots || []).filter(Boolean)
  if (screenshots.length > 0) {
    const index = Math.floor(Math.random() * screenshots.length)
    return screenshots[index]
  }

  return currentGame.banner_image || currentGame.cover_image || ''
}

// Hero background style computed property
const heroBackgroundStyle = computed(() => {
  if (ambientBackgroundUrl.value) {
    return {
      backgroundImage: `url(${ambientBackgroundUrl.value})`,
      backgroundSize: 'cover',
      backgroundPosition: 'center',
    }
  }

  return {
    background: 'linear-gradient(to bottom, #1b2838, #2a475e)'
  }
})

const developerNames = computed(() => (game.value?.developers || []).map((item) => item.name).join(' / '))
const publisherNames = computed(() => (game.value?.publishers || []).map((item) => item.name).join(' / '))

// Sidebar 高度计算

const canEdit = computed(() => true)

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

const handleDownloadVersion = async (version: GameVersion) => {
  if (!game.value) return

  try {
    await downloadService.startDownload(String(game.value.id), version.id)
    uiStore.addAlert(`已开始下载 ${version.version}`, 'success')
    showDownloadModal.value = false
  } catch (error) {
    uiStore.addAlert('下载启动失败', 'error')
  }
}

const handleEditSuccess = async () => {
  // Reload game data after edit
  if (game.value) {
    await gamesStore.fetchGame(String(game.value.id))
  }
}

const handleGoBack = () => {
  router.push({ name: 'games' })
}

const handleToggleFavorite = async () => {
  if (!game.value) return
  try {
    await gamesStore.toggleFavorite(String(game.value.id))
    uiStore.addAlert('收藏已更新', 'success')
  } catch (error) {
    uiStore.addAlert('更新收藏失败', 'error')
  }
}

const sidebarCardInnerRef = ref<HTMLElement | null>(null)
const sidebarContentHeight = ref(0)
const isDesktop = ref(true)
let sidebarResizeObserver: ResizeObserver | null = null

const disconnectSidebarObserver = () => {
  if (sidebarResizeObserver) {
    sidebarResizeObserver.disconnect()
    sidebarResizeObserver = null
  }
}

const syncSidebarHeight = () => {
  const element = sidebarCardInnerRef.value
  if (!element) {
    sidebarContentHeight.value = 0
    return
  }
  sidebarContentHeight.value = Math.round(element.getBoundingClientRect().height)
}

const setupSidebarObserver = async () => {
  await nextTick()

  if (!isDesktop.value || typeof ResizeObserver === 'undefined') {
    disconnectSidebarObserver()
    syncSidebarHeight()
    return
  }

  const element = sidebarCardInnerRef.value
  if (!element) {
    sidebarContentHeight.value = 0
    return
  }

  disconnectSidebarObserver()
  syncSidebarHeight()

  sidebarResizeObserver = new ResizeObserver((entries) => {
    const entry = entries[0]
    if (!entry) return
    sidebarContentHeight.value = Math.round(entry.contentRect.height)
  })
  sidebarResizeObserver.observe(element)
}

const updateBreakpoint = () => {
  if (typeof window === 'undefined') return
  isDesktop.value = window.innerWidth >= 1024
}

const desktopTopSectionHeight = computed(() => {
  if (!isDesktop.value) return undefined
  if (sidebarContentHeight.value <= 520) return undefined
  return Math.round(sidebarContentHeight.value)
})

const loadGameDetail = async (gameId: string) => {
  try {
    const [loadedGame] = await Promise.all([
      gamesStore.fetchGame(gameId),
      gamesStore.fetchGameVersions(gameId),
    ])
    ambientBackgroundUrl.value = pickAmbientBackground(loadedGame)

    wiki.value = null
    try {
      wiki.value = await wikiService.getWikiPage(gameId)
    } catch {
      // Wiki doesn't exist
    }
  } catch (error) {
    uiStore.addAlert('加载游戏详情失败', 'error')
  }
}

watch(
  () => route.params.id,
  async (gameId) => {
    if (!gameId || typeof gameId !== 'string') return
    showEditModal.value = false
    showDownloadModal.value = false
    await loadGameDetail(gameId)
  },
  { immediate: true },
)

onMounted(async () => {
  updateBreakpoint()
  window.addEventListener('resize', updateBreakpoint)
  await setupSidebarObserver()
})

onUnmounted(() => {
  window.removeEventListener('resize', updateBreakpoint)
  disconnectSidebarObserver()
})

watch(
  [game, isDesktop, wiki],
  async () => {
    await setupSidebarObserver()
  },
  { flush: 'post' },
)

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

/* Ambient Background - Subtle, Fixed Bottom */
.game-detail__ambient-bg {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 60vh;
  z-index: -1;
  overflow: hidden;
  opacity: 0.24;
  pointer-events: none;
  /* Use mask to fade out the top part */
  -webkit-mask-image: linear-gradient(to top, rgba(0,0,0,1) 0%, rgba(0,0,0,0) 100%);
  mask-image: linear-gradient(to top, rgba(0,0,0,1) 0%, rgba(0,0,0,0) 100%);
}

.ambient-overlay {
  position: absolute;
  inset: 0;
  background: radial-gradient(
    circle at bottom right,
    rgba(0, 0, 0, 0.06) 0%,
    rgba(0, 0, 0, 0.18) 52%,
    rgba(0, 0, 0, 0.34) 100%
  );
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

.header-back-btn {
  align-self: flex-start;
  padding-left: 0;
  color: var(--color-text-3);
}

.header-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.header-title {
  font-size: 2.25rem;
  font-weight: 700;
  color: #fff;
  margin: 0;
  letter-spacing: -0.5px;
}

.header-actions {
  display: flex; gap: 8px; } .header-favorite-btn, .header-edit-btn { color: var(--color-text-2); } .header-favorite-btn.is-favorite { color: #ff4d4f !important; } /* Main Content Layout - 固定比例: 左侧70%, 右侧30% */
.game-detail__content {
  display: flex;
  padding: 16px 0;
  align-items: flex-start;
}

/* Main Column - 固定占68%宽度 */
.game-detail__main {
  flex: 0 0 68%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.game-detail__main > :deep(.screenshot-carousel) {
  margin-bottom: 0;
  width: 100%;
}

.game-detail__main > :deep(.screenshot-carousel__viewport) {
  border-radius: var(--radius-lg);
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
  padding: 0 0 16px;
}

.game-detail__wiki-wrapper {
  display: flex;
  align-items: start;
  align-items: start;
  max-width: 100%;
  margin: 0;
  padding: 0;
  width: 100%;
  box-sizing: border-box;
}

.game-detail__wiki-card {
  flex: 0 0 70%;
  border-radius: var(--radius-lg);
  width: 100%;
  margin-left: 16px;
}

.game-detail__wiki-placeholder {
  max-width: 100%;
  margin: 0;
  border-radius: var(--radius-lg);
  width: 100%;
  box-sizing: border-box;
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
  padding: 24px 16px;
}

.game-detail__no-wiki-icon {
  font-size: 40px;
  color: var(--color-text-3);
}

.game-detail__no-wiki-text {
  color: var(--color-text-3);
  margin: 12px 0 16px;
}

/* Sidebar - Steam Style - 固定占30%宽度 */
.game-detail__sidebar {
  flex: 0 0 30%;
  margin-left: 16px;
  display: flex;
  flex-direction: column;
  position: sticky;
  top: 16px;
  min-width: 0;
  margin-left: 16px;
}

.sidebar-card {
  background: var(--color-bg-2);
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-lg);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-soft);
}

.sidebar-card__inner {
  display: flex;
  flex-direction: column;
  min-height: 100%;
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
}

/* Sidebar Summary */
.sidebar-summary {
  padding: 12px 12px 0;
  font-size: 13px;
  color: var(--color-text-2);
  line-height: 1.5;
}

/* Sidebar Actions */
.sidebar-actions {
  padding: 12px;
  border-bottom: 1px solid var(--color-border-1);
}

.sidebar-actions :deep(.arco-btn) {
  background: linear-gradient(135deg, var(--color-primary-6), #007aff);
  border: none;
  font-weight: 600;
  border-radius: var(--radius-md);
  box-shadow: 0 4px 12px rgba(26, 159, 255, 0.3);
  transition: all var(--transition-fast);
}

.sidebar-actions :deep(.arco-btn:hover) {
  background: linear-gradient(135deg, var(--color-primary-5), #3395ff);
  box-shadow: 0 6px 16px rgba(26, 159, 255, 0.4);
  transform: translateY(-1px);
}

/* Sidebar Info */
.sidebar-info {
  padding: 12px;
  flex: 1;
}

.sidebar-info__item {
  display: flex;
  flex-direction: column;
  margin-bottom: 12px;
}

.sidebar-info__item:last-child {
  margin-bottom: 0;
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

/* Download Version Modal */
.download-version-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.download-version-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  background: var(--color-fill-2);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.download-version-item:hover {
  background: var(--color-fill-3);
}

.version-info {
  flex: 1;
}

.version-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-1);
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.version-meta {
  font-size: 12px;
  color: var(--color-text-3);
  display: flex;
  gap: 16px;
}

.version-size {
  color: var(--color-text-2);
}

/* Download Empty State */
.download-empty {
  text-align: center;
  padding: 24px 16px;
}

.download-empty-icon {
  font-size: 40px;
  color: var(--color-text-3);
}

.download-empty-text {
  color: var(--color-text-3);
  margin-top: 12px;
}

/* Responsive - Arco Design Breakpoints */
/* lg: 992px */
@media (max-width: 992px) {
  .game-detail__content {
    grid-template-columns: 1fr;
  }

  .game-detail__sidebar {
    order: -1;
    position: static;
  }

  .sidebar-header-image {
    aspect-ratio: 21/9;
    max-height: 200px;
  }
}
</style>
