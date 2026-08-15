<template>
  <div class="game-detail-view">
  <a-empty v-if="hasLoadFailure" class="game-detail__load-failure">
    <template #description>
      <div class="empty-description">
        <h3>加载游戏详情失败</h3>
        <p>当前页面未成功获取这条游戏数据，请稍后重试。</p>
      </div>
    </template>
  </a-empty>

  <div v-else-if="game" class="game-detail">
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
              v-if="canEdit"
              class="app-text-action-btn"
              type="text"
              @click="openEditModal"
            >
              <template #icon>
                <icon-edit />
              </template>
              编辑
            </a-button>

            <a-button
              v-if="canEdit"
              class="app-text-action-btn"
              type="text"
              @click="router.push({ name: 'game-media', params: { publicId: game.public_id } })"
            >
              <template #icon>
                <icon-image />
              </template>
              素材管理
            </a-button>
          </div>
        </div>
      </div>
      </div>

      <div ref="topSectionRef" class="game-detail__top">
        <div class="game-detail__content">
          <div class="game-detail__main">
            <screenshot-carousel
              :preview-videos="game.preview_videos || []"
              :video-poster="game.banner_image || game.cover_image || null"
              :screenshots="game.screenshots.map((item) => item.path)"
              :alt="game.title"
            />
          </div>
        </div>

        <div class="game-detail__sidebar">
          <div class="sidebar-card sidebar-card--hero app-glass-surface">
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
              <img
                v-if="primaryLogo && game?.logo_visible !== false"
                :src="primaryLogo.path"
                :alt="game.title"
                class="sidebar-logo-overlay"
                :style="logoOverlayStyle"
              />
            </div>

            <div v-if="game.summary" class="sidebar-summary">
              {{ game.summary }}
            </div>
          </div>

          <div class="sidebar-card sidebar-card--meta app-glass-surface">
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
            </div>
          </div>
        </div>
      </div>

      <div class="game-detail__download-section">
        <div v-if="game.save_path_template" class="save-path-panel app-glass-surface">
          <div class="save-path-row">
            <span class="save-path-label">存档目录</span>
            <code class="save-path-value">{{ game.save_path_template }}</code>
            <span class="save-path-hint">启动脚本中"打开存档目录"按此模板定位</span>
          </div>
        </div>

        <div v-if="versions.length > 0" class="download-version-panel app-glass-surface">
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
                  v-if="version.canLaunch"
                  class="app-text-action-btn"
                  type="text"
                  @click.stop="handleDownloadLaunchScript(version)"
                >
                  开始游玩
                </a-button>
                <a-button
                  type="primary"
                  @click="handleDownloadVersion(version)"
                >
                  <template #icon>
                    <icon-download />
                  </template>
                  下载
                </a-button>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="download-empty app-glass-surface">
          <p class="download-empty-title">暂无可下载版本</p>
        </div>
      </div>

      <div class="game-detail__wiki-section">
        <!-- Wiki Content with TOC -->
        <div v-if="hasWikiContent" class="game-detail__wiki-wrapper">
          <!-- Wiki TOC Sidebar -->
          <wiki-toc :content="game.wiki_content || ''" />

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
            <markdown-renderer :content="game.wiki_content || ''" />
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
    :game="editableGame"
    @success="handleEditSuccess"
    @sync="handleEditSync"
  />
  </div>

</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useGamesStore } from '@/stores/games'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import ScreenshotCarousel from '@/components/ScreenshotCarousel.vue'
const EditGameModal = defineAsyncComponent(() => import('@/components/EditGameModal.vue'))
import WikiToc from '@/components/WikiToc.vue'
import { useGameDetailView } from '@/composables/useGameDetailView'
import {
  IconEdit,
  IconLeft,
  IconImage,
  IconDownload
} from '@arco-design/web-vue/es/icon'

const route = useRoute()
const router = useRouter()
const gamesStore = useGamesStore()
const authStore = useAuthStore()
const uiStore = useUiStore()
const { isAdmin } = storeToRefs(authStore)
const MarkdownRenderer = defineAsyncComponent(() => import('@/components/MarkdownRenderer.vue'))

const {
  canEdit,
  developerNames,
  editableGame,
  formatDate,
  formatSize,
  game,
  hasLoadFailure,
  handleDownloadLaunchScript,
  handleDownloadVersion,
  handleEditSuccess,
  handleEditSync,
  handleGoBack,
  hasWikiContent,
  openEditModal,
  openWikiEditor,
  publisherNames,
  showEditModal,
  topSectionRef,
  versions,
} = useGameDetailView({
  route,
  router,
  gamesStore,
  uiStore,
  isAdmin,
})

const primaryLogo = computed(() => game.value?.logos?.[0] || null)

const logoOverlayStyle = computed(() => {
  const logo = game.value?.logos?.[0]
  if (!logo) return {}
  return {
    left: `${logo.position_x ?? 50}%`,
    top: `${logo.position_y ?? 50}%`,
    width: `${logo.width_pct ?? 30}%`,
    transform: 'translate(-50%, -50%)',
  }
})
</script>
<style scoped src="./GameDetailView.css"></style>
