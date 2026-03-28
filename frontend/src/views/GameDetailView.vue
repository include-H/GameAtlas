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
import { defineAsyncComponent } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useGamesStore } from '@/stores/games'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import ScreenshotCarousel from '@/components/ScreenshotCarousel.vue'
import EditGameModal from '@/components/EditGameModal.vue'
import WikiToc from '@/components/WikiToc.vue'
import { useGameDetailView } from '@/composables/useGameDetailView'
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
const { isAdmin } = storeToRefs(authStore)
const MarkdownRenderer = defineAsyncComponent(() => import('@/components/MarkdownRenderer.vue'))

const {
  canEdit,
  carouselHeight,
  developerNames,
  formatDate,
  formatSize,
  game,
  handleDownloadLaunchScript,
  handleDownloadVersion,
  handleEditSuccess,
  handleGoBack,
  handleToggleFavorite,
  hasWikiContent,
  openWikiEditor,
  publisherNames,
  shouldSpanMetadataRow,
  showEditModal,
  topSectionRef,
  versions,
  wiki,
} = useGameDetailView({
  route,
  router,
  gamesStore,
  uiStore,
  isAdmin,
})
</script>
<style scoped src="./GameDetailView.css"></style>
