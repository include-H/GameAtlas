<template>
  <router-view v-if="isFullscreenPage" v-slot="{ Component, route }">
    <transition name="route-fade" mode="out-in">
      <div
        :key="String(route.name || route.path)"
        :class="route.name === 'login' ? 'auth-route-shell' : 'fullscreen-route-shell'"
      >
        <shared-ambient-background v-if="route.name === 'login'" />
        <component :is="Component" />
        <alert-banner v-if="route.name === 'login'" />
      </div>
    </transition>
  </router-view>

  <a-layout v-else class="app-layout">
    <a-layout-header class="pro-header glass-header">
      <div class="header-left">
        <div class="logo">
          <icon-trophy :size="28" />
          <span class="logo-text">{{ appName }}</span>
        </div>
      </div>

      <div class="header-right">
        <a-space :size="20">
          <span v-if="isAdmin" class="welcome-text">欢迎您，{{ adminDisplayName }}</span>
          <span v-else-if="authLoadFailed" class="welcome-text welcome-text--warning">认证状态加载失败</span>
          <a-button
            class="app-text-action-btn"
            type="text"
            :aria-label="startScreenVisible ? '关闭开始屏幕' : '打开开始屏幕'"
            @click="toggleStartScreen"
          >
            开始
          </a-button>
          <a-button class="app-text-action-btn" type="text" @click="handleAuthAction">
            {{ authActionLabel }}
          </a-button>
          <a-button class="app-text-action-btn" type="text" shape="circle" @click="scrollToTop">
            <template #icon>
              <icon-up />
            </template>
          </a-button>
        </a-space>
      </div>
    </a-layout-header>

    <a-button
      v-if="isCompactNavigation"
      :class="['app-text-action-btn', 'mobile-menu-btn', { 'mobile-menu-btn--active': showMobileMenu }]"
      type="text"
      shape="circle"
      @click="showMobileMenu = !showMobileMenu"
    >
      <template #icon>
        <icon-menu-fold v-if="showMobileMenu" />
        <icon-menu-unfold v-else />
      </template>
    </a-button>

    <a-layout class="main-layout">
      <a-layout-sider
        v-if="!isCompactNavigation"
        v-model:collapsed="collapsed"
        :width="sideWidth"
        :collapsed-width="collapsedSideWidth"
        collapsible
        class="app-sider"
      >
        <div class="app-sider__inner">
          <app-navigation-menu
            :items="menuList"
            :active-key="activeKey"
            :open-keys="desktopOpenKeys"
            @navigate="handleMenuClick"
            @update:open-keys="handleDesktopOpenKeysChange"
          />
        </div>

        <template #trigger="{ collapsed: isCollapsed }">
          <icon-menu-unfold v-if="isCollapsed" />
          <icon-menu-fold v-else />
        </template>
      </a-layout-sider>

      <a-drawer
        v-model:visible="showMobileMenu"
        placement="left"
        :width="sideWidth"
        :footer="false"
        class="mobile-drawer"
        @cancel="showMobileMenu = false"
      >
        <template #title>
          <div class="mobile-drawer-header">
            <icon-trophy :size="24" />
            <span>{{ appName }}</span>
          </div>
        </template>
        <app-navigation-menu
          :items="menuList"
          :active-key="activeKey"
          :open-keys="mobileOpenKeys"
          @navigate="handleMobileMenuClick"
          @update:open-keys="handleMobileOpenKeysChange"
        />
      </a-drawer>

      <a-layout class="content-layout">
        <shared-ambient-background />

        <a-layout-content class="content">
          <router-view v-slot="{ Component }">
            <!-- Vue 官方模式：v-if 包裹 + out-in，避免 keep-alive 在过渡期间
                 残留空槽/旧组件 DOM（曾导致库页面残留详情页、返回时被重挂载）。 -->
            <template v-if="Component">
              <transition name="route-fade" mode="out-in">
                <keep-alive include="GamesView">
                  <component :is="Component" class="route-fade-shell" />
                </keep-alive>
              </transition>
            </template>
          </router-view>

          <alert-banner />
        </a-layout-content>

        <a-layout-footer class="footer">
          <span>&copy; 这份作品来自不知名网友Hao和他的星期五</span>
        </a-layout-footer>
      </a-layout>
    </a-layout>
  </a-layout>

  <start-screen
    :visible="startScreenVisible"
    :tiles="startScreenTiles"
    :columns="startScreenColumns"
    :can-edit="isAdmin"
    :admin-display-name="adminDisplayName"
    :is-loading="startScreenLoading"
    :has-load-failure="startScreenLoadFailed"
    :is-editing="startScreenEditing"
    :is-saving="startScreenSaving"
    :save-error="startScreenSaveError"
    @close="closeStartScreen"
    @retry="retryStartScreen"
    @start-edit="startStartScreenEdit"
    @cancel-edit="cancelStartScreenEdit"
    @save-edit="saveStartScreenEdit"
    @resize="resizeStartScreenTile"
    @remove="removeStartScreenTile"
    @apply-placement="applyStartScreenPlacement"
    @add-column="addStartScreenColumn"
    @remove-column="removeStartScreenColumn"
    @apply-crop="applyStartScreenCrop"
    @rename-column="renameStartScreenColumn"
  />
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import useMenu from '@/hooks/useMenu'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import gamesService from '@/services/games.service'
import startScreenService from '@/services/start-screen.service'
import type { GameListItem, StartScreenLayoutInput } from '@/services/types'
import AlertBanner from '@/components/AlertBanner.vue'
import AppNavigationMenu from '@/components/AppNavigationMenu.vue'
import SharedAmbientBackground from '@/components/SharedAmbientBackground.vue'
import StartScreen from '@/components/start-screen/StartScreen.vue'
import { useStartScreen } from '@/composables/useStartScreen'
import { useCompactNavigation } from '@/composables/useCompactNavigation'
import { useAppNavigation } from '@/composables/useAppNavigation'
import {
  IconTrophy,
  IconMenuFold,
  IconMenuUnfold,
  IconUp,
} from '@arco-design/web-vue/es/icon'

const router = useRouter()
const route = useRoute()
const uiStore = useUiStore()
const authStore = useAuthStore()
const { menuList, activeKey, openKeys: routeOpenKeys } = useMenu()
const { sidebarCollapsed } = storeToRefs(uiStore)
const { isAdmin, adminDisplayName, authLoadFailed } = storeToRefs(authStore)
const { isCompactNavigation, showMobileMenu } = useCompactNavigation()
const {
  desktopOpenKeys,
  mobileOpenKeys,
  handleMenuClick,
  handleMobileMenuClick,
  handleDesktopOpenKeysChange,
  handleMobileOpenKeysChange,
} = useAppNavigation({
  router,
  routeOpenKeys,
  activeKey,
  closeMobileMenu: () => {
    showMobileMenu.value = false
  },
})
const {
  visible: startScreenVisible,
  tiles: startScreenTiles,
  columns: startScreenColumns,
  isLoading: startScreenLoading,
  hasLoadFailure: startScreenLoadFailed,
  isEditing: startScreenEditing,
  isSaving: startScreenSaving,
  saveError: startScreenSaveError,
  close: closeStartScreen,
  toggle: toggleStartScreen,
  retry: retryStartScreen,
  startEdit: startStartScreenEdit,
  cancelEdit: cancelStartScreenEdit,
  saveEdit: saveStartScreenEdit,
  resizeTile: resizeStartScreenTile,
  removeTile: removeStartScreenTile,
  applyTilePlacement: applyStartScreenPlacement,
  addColumn: addStartScreenColumn,
  removeColumn: removeStartScreenColumn,
  applyTileCrop: applyStartScreenCrop,
  renameColumn: renameStartScreenColumn,
} = useStartScreen({
  fetchTiles: () => startScreenService.getTiles(),
  uploadTileImage: (file) => startScreenService.uploadTileImage(file),
  fetchFavorites: async () => {
    const favorites: GameListItem[] = []
    let page = 1
    while (true) {
      const result = await gamesService.getGames({
        query: { favorite: true, page, limit: 100 },
      })
      favorites.push(...result.data)
      if (result.data.length === 0 || page >= result.pagination.totalPages || favorites.length >= 500) {
        break
      }
      page += 1
    }
    return favorites
  },
  saveTiles: (input: StartScreenLayoutInput) => startScreenService.updateTiles(input),
  addAlert: (message, type) => {
    uiStore.addAlert(message, type)
  },
})

const appName = 'GameAtlas'
const sideWidth = 240
const collapsedSideWidth = 48
const isFullscreenPage = computed(() => route.name === 'login' || route.meta?.fullscreen === true)

const collapsed = computed({
  get: () => sidebarCollapsed.value,
  set: (value: boolean) => {
    uiStore.setSidebarCollapsed(value)
  },
})

const authActionLabel = computed(() => {
  if (authLoadFailed.value) {
    return '重试认证'
  }

  return isAdmin.value ? '退出' : '登录'
})

const handleAuthAction = async () => {
  if (authLoadFailed.value) {
    await authStore.fetchMe()
    if (!authStore.authLoadFailed) {
      uiStore.addAlert('认证状态已刷新', 'success')
      return
    }
    uiStore.addAlert('认证状态刷新失败', 'error')
    return
  }

  if (!isAdmin.value) {
    if (router.currentRoute.value.name === 'login') {
      return
    }

    router.push({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
    return
  }

  try {
    await authStore.logout()
    uiStore.addAlert('已退出管理模式', 'success')
    router.push({ name: 'dashboard' })
  } catch {
    uiStore.addAlert('退出失败', 'error')
  }
}

const scrollToTop = () => {
  const content = document.querySelector('.content')
  if (content) {
    content.scrollTo({ top: 0, behavior: 'smooth' })
    return
  }

  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 共享滚动容器 .content 跨路由保留 scrollTop，切换页面必须回顶，
// 否则从库中部点进详情会直接落在中间（如 wiki 区）。
// 游戏库路由除外：keep-alive 恢复时由 GamesView 自己恢复离开位置。
watch(
  () => route.fullPath,
  () => {
    if (route.name === 'games') return
    document.querySelector('.content')?.scrollTo({ top: 0 })
  },
)

// 看板娘资源预热：应用启动空闲时低优先级预取，进游戏店时命中缓存秒现，
// 避免 Live2D 三件套 + 模型贴图（1.1M）在进店瞬间才开始下载。
const PREFETCH_WAIFU_URLS = [
  '/live2d-widget/waifu.css',
  '/live2d-widget/waifu-tips.js',
  '/live2d-widget/live2d.min.js',
  '/live2d-models/model/kobayaxi/index.json',
  '/live2d-models/model/kobayaxi/Kobayaxi.moc',
  '/live2d-models/model/kobayaxi/Kobayaxi.2048/texture_00.png',
]

const prefetchWaifuResources = () => {
  for (const url of PREFETCH_WAIFU_URLS) {
    const link = document.createElement('link')
    link.rel = 'prefetch'
    link.href = url
    document.head.appendChild(link)
  }
}

if (typeof requestIdleCallback === 'function') {
  requestIdleCallback(() => prefetchWaifuResources(), { timeout: 3000 })
} else {
  window.setTimeout(prefetchWaifuResources, 1000)
}
</script>

<style scoped src="./App.css"></style>

<style src="./App.global.css"></style>
