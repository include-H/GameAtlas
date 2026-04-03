<template>
  <router-view v-if="isAuthPage" v-slot="{ Component, route }">
    <transition name="route-fade" mode="out-in">
      <div :key="String(route.name || route.path)" class="auth-route-shell">
        <shared-ambient-background />
        <component :is="Component" />
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
          <a-button class="app-text-action-btn" type="text" @click="handleAuthAction">
            {{ isAdmin ? '退出' : '登录' }}
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
            <div
              :key="String(route.name || route.path)"
              class="route-fade-shell"
            >
              <component :is="Component" />
            </div>
          </router-view>

          <alert-banner />
        </a-layout-content>

        <a-layout-footer class="footer">
          <span>&copy; 这份作品来自不知名网友Hao和他的星期五</span>
        </a-layout-footer>
      </a-layout>
    </a-layout>
  </a-layout>

</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import useMenu from '@/hooks/useMenu'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import AlertBanner from '@/components/AlertBanner.vue'
import AppNavigationMenu from '@/components/AppNavigationMenu.vue'
import SharedAmbientBackground from '@/components/SharedAmbientBackground.vue'
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
const { isAdmin, adminDisplayName } = storeToRefs(authStore)

const appName = 'GameAtlas'
const sideWidth = 240
const collapsedSideWidth = 48
const compactNavigationBreakpoint = 992
const isAuthPage = computed(() => route.name === 'login')

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

const syncOpenKeysWithRoute = () => {
  desktopOpenKeys.value = [...routeOpenKeys.value]
  mobileOpenKeys.value = [...routeOpenKeys.value]
}

const handleAuthAction = async () => {
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

.pro-header .logo :deep(.arco-icon) {
  color: color-mix(in srgb, var(--color-primary-light-2) 14%, var(--color-primary-6));
  filter: none;
}

.pro-header .logo-text {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: 0.5px;
  color: color-mix(in srgb, var(--color-text-1) 74%, #c6d7e7 26%);
  text-shadow: none;
}

.welcome-text {
  color: color-mix(in srgb, var(--color-text-1) 68%, var(--color-primary-light-2) 32%);
  font-size: 14px;
  white-space: nowrap;
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
  padding: 14px 0;
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
  box-shadow: 0 10px 24px rgba(3, 8, 20, 0.28);
  transition:
    background-color var(--transition-fast),
    color var(--transition-fast),
    border-color var(--transition-fast),
    box-shadow var(--transition-fast);
}

.mobile-menu-btn:hover {
  background: var(--app-sider-hover) !important;
  color: var(--color-text-1) !important;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.035), 0 10px 24px rgba(3, 8, 20, 0.28);
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
  min-height: 100vh;
  position: relative;
  isolation: isolate;
  overflow: hidden;
  background: transparent;
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
  background-color: rgba(122, 162, 199, 0.14) !important;
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
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.035);
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
  background: rgba(6, 10, 18, 0.46);
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
