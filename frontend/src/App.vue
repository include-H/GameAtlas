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
        <a-button
          class="app-text-action-btn start-button"
          type="text"
          shape="circle"
          :aria-label="startScreenVisible ? '关闭开始屏幕' : '打开开始屏幕'"
          @click="toggleStartScreen"
        >
          <template #icon>
            <icon-apps />
          </template>
        </a-button>
        <div class="logo">
          <icon-trophy :size="28" />
          <span class="logo-text">{{ appName }}</span>
        </div>
      </div>

      <div class="header-right">
        <a-space :size="20">
          <span v-if="isAdmin" class="welcome-text">欢迎您，{{ adminDisplayName }}</span>
          <span v-else-if="authLoadFailed" class="welcome-text welcome-text--warning">认证状态加载失败</span>
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
          <router-view v-slot="{ Component, route }">
            <!-- Vue 3.5's out-in transition clones KeepAlive with null slots during route changes. -->
            <transition name="route-fade">
              <keep-alive include="GamesView">
                <component
                  :is="Component"
                  :key="String(route.name || route.path)"
                  class="route-fade-shell"
                />
              </keep-alive>
            </transition>
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
    :favorite-pool="startScreenFavoritePool"
    :can-edit="isAdmin"
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
    @move="moveStartScreenTile"
    @add="addStartScreenTile"
    @apply-crop="applyStartScreenCrop"
    @rename-column="renameStartScreenColumn"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
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
import {
  IconTrophy,
  IconMenuFold,
  IconMenuUnfold,
  IconUp,
  IconApps,
} from '@arco-design/web-vue/es/icon'

const router = useRouter()
const route = useRoute()
const uiStore = useUiStore()
const authStore = useAuthStore()
const { menuList, activeKey, openKeys: routeOpenKeys } = useMenu()
const { sidebarCollapsed } = storeToRefs(uiStore)
const { isAdmin, adminDisplayName, authLoadFailed } = storeToRefs(authStore)
const {
  visible: startScreenVisible,
  tiles: startScreenTiles,
  columns: startScreenColumns,
  favoritePool: startScreenFavoritePool,
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
  moveTile: moveStartScreenTile,
  addTile: addStartScreenTile,
  applyTileCrop: applyStartScreenCrop,
  renameColumn: renameStartScreenColumn,
} = useStartScreen({
  fetchTiles: () => startScreenService.getTiles(),
  uploadTileImage: (file, size) => startScreenService.uploadTileImage(file, size),
  fetchFavorites: async () => {
    const favorites: GameListItem[] = []
    let page = 1
    while (true) {
      const result = await gamesService.getGames({
        query: { favorite: true, page, limit: 100 },
      })
      favorites.push(...result.data)
      if (result.data.length === 0 || favorites.length >= result.pagination.total || favorites.length >= 500) {
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
const compactNavigationBreakpoint = 992
const isFullscreenPage = computed(() => route.name === 'login' || route.meta?.fullscreen === true)

const collapsed = computed({
  get: () => sidebarCollapsed.value,
  set: (value: boolean) => {
    uiStore.setSidebarCollapsed(value)
  },
})

const isCompactNavigation = ref(false)
const showMobileMenu = ref(false)
const desktopOpenKeys = ref<string[]>([])
const mobileOpenKeys = ref<string[]>([])
const authActionLabel = computed(() => {
  if (authLoadFailed.value) {
    return '重试认证'
  }

  return isAdmin.value ? '退出' : '登录'
})

const syncOpenKeysWithRoute = () => {
  desktopOpenKeys.value = [...routeOpenKeys.value]
  mobileOpenKeys.value = [...routeOpenKeys.value]
}

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

const handleMenuClick = (key: string) => {
  router.push({ name: key })
}

const handleMobileMenuClick = (key: string) => {
  handleMenuClick(key)
  showMobileMenu.value = false
}

const handleDesktopOpenKeysChange = (keys: string[]) => {
  desktopOpenKeys.value = keys
}

const handleMobileOpenKeysChange = (keys: string[]) => {
  mobileOpenKeys.value = keys
}

const handleResize = () => {
  const compact = window.innerWidth < compactNavigationBreakpoint
  isCompactNavigation.value = compact

  if (compact) {
    showMobileMenu.value = false
  }
}

watch([activeKey, routeOpenKeys], () => {
  syncOpenKeysWithRoute()
}, { immediate: true })

onMounted(() => {
  handleResize()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.app-layout {
  height: 100vh;
  height: 100dvh;
  width: 100%;
  min-width: 0;
  overflow: hidden;
}

.main-layout {
  padding-top: 56px;
  height: 100vh;
  height: 100dvh;
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
  overflow: hidden;
}

.content-layout {
  height: 100%;
  min-width: 0;
  flex: 1;
  overflow: hidden;
  position: relative;
  z-index: 1;
}

.pro-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 24px;
  background: var(--app-header-surface);
  border-bottom: 1px solid var(--app-header-border);
  box-shadow: var(--app-header-shadow);
  height: 56px;
  line-height: 56px;
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  z-index: 100;
  box-sizing: border-box;
}

.pro-header .header-left,
.pro-header .header-right {
  display: flex;
  align-items: center;
  position: relative;
  z-index: 1;
}

.pro-header .logo {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--color-primary-6);
  padding: 4px 12px;
  border-radius: var(--radius-md);
  background: transparent;
  border: none;
}

.start-button {
  margin-right: 6px;
  color: var(--color-text-2);
  background: transparent;
  transition: color var(--transition-fast), background var(--transition-fast);
}

.start-button:hover {
  color: var(--color-primary-6);
  background: var(--app-header-hover);
}

.pro-header .logo :deep(.arco-icon) {
  color: color-mix(in srgb, var(--color-primary-light-2) 14%, var(--color-primary-6));
  filter: none;
}

.pro-header .logo-text {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: 0.5px;
  color: color-mix(in srgb, var(--color-text-1) 74%, var(--color-logo-text-mix) 26%);
  text-shadow: none;
}

.welcome-text {
  color: color-mix(in srgb, var(--color-text-1) 68%, var(--color-primary-light-2) 32%);
  font-size: 14px;
  white-space: nowrap;
}

.welcome-text--warning {
  color: color-mix(in srgb, var(--color-welcome-warning-mix) 74%, var(--color-text-1) 26%);
}

.pro-header :deep(.arco-btn-text) {
  border-radius: 0;
  color: var(--color-text-2);
  background-color: transparent;
  transition:
    color var(--transition-fast),
    background-color var(--transition-fast);
}

.pro-header :deep(.arco-btn-text:hover) {
  color: var(--color-text-1);
  background-color: var(--app-header-hover) !important;
}

.pro-header :deep(.arco-btn-text .arco-icon) {
  color: inherit;
}

.app-sider__inner {
  height: 100%;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--app-sider-surface);
}

.content {
  padding: 24px;
  background: transparent;
  height: calc(100vh - 56px - 48px);
  height: calc(100dvh - 56px - 48px);
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
  overflow-y: auto;
  overflow-x: hidden;
  position: relative;
  z-index: 1;
}

.footer {
  /* Footer intentionally shares the same blue-gray family, but stays lighter than header/sider so the page frame tapers off instead of closing too hard. */
  text-align: center;
  color: color-mix(in srgb, var(--color-text-2) 72%, var(--color-text-3));
  font-size: 13px;
  background: var(--app-footer-surface);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-top: 1px solid var(--app-footer-border);
  box-shadow: var(--app-footer-shadow);
  position: relative;
  z-index: 1;
}

.mobile-drawer-header {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--color-primary-6);
  font-size: 18px;
  font-weight: 600;
}

.mobile-drawer :deep(.arco-drawer-body) {
  padding: 0;
}

.mobile-menu-btn {
  position: fixed;
  right: 32px;
  bottom: calc(36px + env(safe-area-inset-bottom, 0px));
  z-index: 130;
  width: 48px;
  height: 48px;
  border: 1px solid var(--app-sider-border);
  border-radius: 16px;
  background: color-mix(in srgb, var(--app-sider-surface) 72%, transparent) !important;
  color: color-mix(in srgb, var(--color-text-1) 74%, var(--color-primary-light-2) 26%) !important;
  backdrop-filter: blur(10px) saturate(120%);
  -webkit-backdrop-filter: blur(10px) saturate(120%);
  box-shadow: var(--shadow-hover);
  transition:
    background-color var(--transition-fast),
    color var(--transition-fast),
    border-color var(--transition-fast),
    box-shadow var(--transition-fast);
}

.mobile-menu-btn:hover {
  background: var(--app-sider-hover) !important;
  color: var(--color-text-1) !important;
  box-shadow: inset 0 1px 0 var(--color-border-1), var(--shadow-hover);
}

.mobile-menu-btn.mobile-menu-btn--active {
  background: color-mix(in srgb, var(--app-sider-surface) 56%, transparent) !important;
  border-color: color-mix(in srgb, var(--app-sider-border) 78%, transparent);
}

.mobile-menu-btn :deep(.arco-icon) {
  color: inherit;
}

@media (max-width: 767px) {
  .pro-header {
    padding: 0 16px;
  }

  .pro-header .logo-text {
    font-size: 18px;
  }

  .content {
    padding: 16px;
  }
}

@media (max-width: 576px) {
  .pro-header {
    padding: 0 12px;
  }

  .pro-header .logo {
    gap: 8px;
    padding: 4px 8px;
  }

  .pro-header .logo-text {
    font-size: 16px;
  }

  .welcome-text {
    display: none;
  }

  .content {
    padding: 12px;
  }

  .mobile-menu-btn {
    right: 28px;
    bottom: calc(32px + env(safe-area-inset-bottom, 0px));
  }
}
</style>

<style>
html {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
  overflow-x: hidden;
}

body {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
  overflow: hidden;
}

#app {
  min-height: 100vh;
  width: 100%;
  min-width: 0;
  overflow-x: hidden;
}

.auth-route-shell {
  height: 100vh;
  height: 100dvh;
  min-height: 100vh;
  min-height: 100dvh;
  position: relative;
  isolation: isolate;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
  background: transparent;
}

.fullscreen-route-shell {
  /* 全屏沉浸式页面（登录/游戏店）的深色底是品牌特例，仅用于脱离后台框架的页面壳。 */
  height: 100vh;
  height: 100dvh;
  width: 100%;
  min-height: 100vh;
  min-height: 100dvh;
  position: relative;
  isolation: isolate;
  overflow: hidden;
  overscroll-behavior: contain;
  background: #17110d;
}

.arco-layout {
  height: 100%;
  min-width: 0;
  background: transparent;
}

.arco-layout-content {
  width: auto;
  min-width: 0;
}

.app-sider.arco-layout-sider {
  background: var(--app-sider-surface);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-right: 1px solid var(--app-sider-border);
  z-index: 99;
  position: relative;
  height: 100%;
  overflow-x: hidden;
}

.app-sider.arco-layout-sider-has-trigger {
  padding-bottom: 0;
}

.app-sider .arco-layout-sider-children {
  height: calc(100% - 48px);
  overflow: hidden;
}

.app-navigation-menu.arco-menu {
  background: transparent;
  border-right: none;
}

.app-navigation-menu .arco-menu-selected {
  background-color: var(--color-sidebar-selected-bg) !important;
  color: var(--color-primary-6) !important;
  font-weight: 600;
}

.app-sider :is(.arco-layout-sider-trigger, .arco-layout-sider-trigger-light) {
  height: 48px;
  background: color-mix(in srgb, var(--app-sider-surface) 72%, transparent);
  border-top: 1px solid var(--app-sider-border);
  color: color-mix(in srgb, var(--color-text-1) 74%, var(--color-primary-light-2) 26%);
  backdrop-filter: blur(10px) saturate(120%);
  -webkit-backdrop-filter: blur(10px) saturate(120%);
  box-shadow: none;
  transition:
    background-color var(--transition-fast),
    color var(--transition-fast),
    box-shadow var(--transition-fast);
}

.app-sider :is(.arco-layout-sider-trigger, .arco-layout-sider-trigger-light):hover {
  background: var(--app-sider-hover);
  color: var(--color-text-1);
  box-shadow: inset 0 1px 0 var(--color-border-1);
}

.app-sider.arco-layout-sider-collapsed :is(.arco-layout-sider-trigger, .arco-layout-sider-trigger-light) {
  background: color-mix(in srgb, var(--app-sider-surface) 56%, transparent);
  border-top-color: color-mix(in srgb, var(--app-sider-border) 78%, transparent);
}

.app-sider.arco-layout-sider-collapsed :is(.arco-layout-sider-trigger, .arco-layout-sider-trigger-light):hover {
  background: color-mix(in srgb, var(--app-sider-hover) 82%, transparent);
}

.app-sider :is(.arco-layout-sider-trigger, .arco-layout-sider-trigger-light) .arco-icon {
  color: inherit;
}

.pro-header.arco-layout-header {
  background: var(--app-header-surface) !important;
  border-bottom: 1px solid var(--app-header-border) !important;
  box-shadow: var(--app-header-shadow) !important;
}

.footer.arco-layout-footer {
  background: var(--app-footer-surface) !important;
  border-top: 1px solid var(--app-footer-border) !important;
  box-shadow: var(--app-footer-shadow) !important;
}

.mobile-drawer .arco-drawer-mask {
  background: var(--app-scrim);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
}

.mobile-drawer .arco-drawer-content {
  background: var(--app-sider-surface) !important;
  border-right: 1px solid var(--app-sider-border);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  box-shadow: var(--app-header-shadow);
}

.mobile-drawer .arco-drawer {
  background: var(--app-sider-surface) !important;
  border-right: 1px solid var(--app-sider-border);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  box-shadow: var(--app-header-shadow);
}

.mobile-drawer .arco-drawer-header {
  background: color-mix(in srgb, var(--app-sider-surface) 82%, transparent) !important;
  border-bottom: 1px solid var(--app-sider-border);
}

.mobile-drawer .arco-drawer-title,
.mobile-drawer .arco-drawer-close-btn {
  color: color-mix(in srgb, var(--color-text-1) 74%, var(--color-primary-light-2) 26%) !important;
}

.mobile-drawer .arco-drawer-body {
  background: var(--app-sider-surface) !important;
}
</style>
