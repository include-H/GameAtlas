<template>
  <a-layout class="app-layout">
    <a-layout-header class="pro-header glass-header">
      <div class="header-left">
        <div class="logo hover-glow" @click="handleLogoClick">
          <icon-trophy :size="28" />
          <span class="logo-text">{{ appName }}</span>
        </div>
      </div>

      <div class="header-right">
        <a-space :size="20">
          <a-button type="text" @click="handleAuthAction">
            {{ isAdmin ? '退出' : '登录' }}
          </a-button>
          <a-button type="text" shape="circle" @click="scrollToTop">
            <template #icon>
              <icon-up />
            </template>
          </a-button>
        </a-space>
      </div>
    </a-layout-header>

    <a-button
      v-if="isCompactNavigation"
      class="mobile-menu-btn"
      type="primary"
      shape="circle"
      @click="showMobileMenu = true"
    >
      <template #icon>
        <icon-menu-unfold />
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
            <keep-alive :include="['GamesView', 'DashboardView']">
              <component :is="Component" />
            </keep-alive>
          </router-view>

          <alert-banner />
        </a-layout-content>

        <a-layout-footer class="footer">
          <span>&copy; 这份作品来自不知名网友Hao和他的星期五</span>
        </a-layout-footer>
      </a-layout>
    </a-layout>
  </a-layout>

  <a-message v-model:visible="message.show" :type="message.type">
    {{ message.content }}
  </a-message>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, provide, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
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
const uiStore = useUiStore()
const authStore = useAuthStore()
const { menuList, activeKey, openKeys: routeOpenKeys } = useMenu()
const { sidebarCollapsed } = storeToRefs(uiStore)
const { isAdmin } = storeToRefs(authStore)

const appName = 'GameAtlas'
const sideWidth = 240
const collapsedSideWidth = 48
const compactNavigationBreakpoint = 992

const collapsed = computed({
  get: () => sidebarCollapsed.value,
  set: (value: boolean) => {
    uiStore.setSidebarCollapsed(value)
  },
})

const message = ref({
  show: false,
  content: '',
  type: 'info',
})

const isCompactNavigation = ref(false)
const showMobileMenu = ref(false)
const desktopOpenKeys = ref<string[]>([])
const mobileOpenKeys = ref<string[]>([])

const syncOpenKeysWithRoute = () => {
  desktopOpenKeys.value = [...routeOpenKeys.value]
  mobileOpenKeys.value = [...routeOpenKeys.value]
}

const handleLogoClick = () => {
  router.push({ name: 'dashboard' })
}

const handleAuthAction = async () => {
  if (!isAdmin.value) {
    router.push({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
    return
  }

  try {
    await authStore.logout()
    showMessage('已退出管理模式', 'success')
    router.push({ name: 'dashboard' })
  } catch {
    showMessage('退出失败', 'error')
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

const showMessage = (content: string, type = 'info') => {
  message.value = { show: true, content, type }
  setTimeout(() => {
    message.value.show = false
  }, 3000)
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

provide('showMessage', showMessage)
</script>

<style scoped>
.app-layout {
  height: 100vh;
  width: 100%;
  min-width: 0;
  overflow: hidden;
}

.main-layout {
  padding-top: 56px;
  height: 100vh;
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
  background: var(--color-bg-2);
  border-bottom: 1px solid var(--color-border-2);
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
}

.pro-header .logo {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--color-primary-6);
  cursor: pointer;
  padding: 4px 12px;
  border-radius: var(--radius-md);
  border: none;
  outline: none;
  transition: all var(--transition-fast);
}

.pro-header .logo:hover {
  background: transparent;
  box-shadow: none !important;
}

.pro-header .logo-text {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: 0.5px;
  background: linear-gradient(135deg, var(--color-primary-light-3), var(--color-primary-6));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.app-sider__inner {
  height: 100%;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
}

.content {
  padding: 24px;
  background: transparent;
  height: calc(100vh - 56px - 48px);
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
  overflow-y: auto;
  overflow-x: hidden;
  position: relative;
  z-index: 1;
}

.footer {
  text-align: center;
  color: var(--color-text-3);
  font-size: 13px;
  background: transparent;
  padding: 16px 0;
  border-top: 1px solid var(--color-border-1);
  position: relative;
  z-index: 1;
}

.mobile-menu-btn {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 99;
  width: 56px;
  height: 56px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.mobile-menu-btn :deep(.arco-btn-icon) {
  font-size: 24px;
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
  background: rgba(22, 26, 37, 0.4);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-right: 1px solid var(--color-border-1);
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
  background-color: rgba(22, 93, 255, 0.16) !important;
  color: var(--color-primary-6) !important;
  font-weight: 600;
}

.app-sider .arco-layout-sider-trigger {
  background: var(--color-fill-1);
  border-top: 1px solid var(--color-border-1);
  transition: background-color 0.2s;
}

.app-sider .arco-layout-sider-trigger:hover {
  background: var(--color-fill-2);
}
</style>
