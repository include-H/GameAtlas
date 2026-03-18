<template>
  <a-layout class="app-layout">
    <!-- Pro-style Header (full-width at top) -->
    <a-layout-header class="pro-header glass-header">
      <div class="header-left">
        <!-- Logo in header (Pro style) -->
        <div class="logo hover-glow" @click="handleLogoClick">
          <icon-trophy :size="28" />
          <span class="logo-text">{{ appName }}</span>
        </div>
      </div>

      <div class="header-right">
        <a-space :size="20">
          <!-- Back to Top -->
          <a-button
            type="text"
            shape="circle"
            @click="scrollToTop"
          >
            <template #icon>
              <icon-up />
            </template>
          </a-button>
        </a-space>
      </div>
    </a-layout-header>

    <!-- Mobile Menu Button -->
    <a-button
      v-if="isMobile"
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
      <!-- Sidebar (Pro-style - menu only, no logo) -->
      <!-- Desktop: always show, Mobile: hidden (use drawer instead) -->
      <a-layout-sider
        v-if="!isMobile"
        v-model:collapsed="collapsed"
        :width="sideWidth"
        breakpoint="xl"
        collapsible
        class="app-sider"
        @collapse="handleCollapse"
      >
        <!-- Menu generated from routes -->
        <a-menu
          :selected-keys="[activeKey]"
          :auto-open-selected="true"
          :collapsed="collapsed"
          @menu-item-click="handleMenuClick"
        >
          <template v-for="item in menuList" :key="item.name">
            <!-- Menu Item without children -->
            <a-menu-item v-if="!item.children || item.children.length === 0" :key="item.name">
              <template #icon>
                <component :is="item.icon" />
              </template>
              {{ t(item.locale) }}
            </a-menu-item>

            <!-- Sub Menu with children -->
            <a-sub-menu v-else :key="item.name">
              <template #icon>
                <component :is="item.icon" />
              </template>
              <template #title>
                {{ t(item.locale) }}
              </template>
              <a-menu-item
                v-for="child in item.children"
                :key="child.name"
              >
                {{ t(child.locale) }}
              </a-menu-item>
            </a-sub-menu>
          </template>
        </a-menu>

        <!-- Collapse trigger (Arco standard position at bottom) -->
        <template #trigger="{ collapsed }">
          <icon-menu-unfold v-if="collapsed" />
          <icon-menu-fold v-else />
        </template>
      </a-layout-sider>

      <!-- Mobile Drawer Menu -->
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
        <a-menu
          :selected-keys="[activeKey]"
          :auto-open-selected="true"
          @menu-item-click="(key: string) => { handleMenuClick(key); showMobileMenu = false; }"
        >
          <template v-for="item in menuList" :key="item.name">
            <a-menu-item v-if="!item.children || item.children.length === 0" :key="item.name">
              <template #icon>
                <component :is="item.icon" />
              </template>
              {{ t(item.locale) }}
            </a-menu-item>
            <a-sub-menu v-else :key="item.name">
              <template #icon>
                <component :is="item.icon" />
              </template>
              <template #title>
                {{ t(item.locale) }}
              </template>
              <a-menu-item
                v-for="child in item.children"
                :key="child.name"
              >
                {{ t(child.locale) }}
              </a-menu-item>
            </a-sub-menu>
          </template>
        </a-menu>
      </a-drawer>

      <!-- Main Content -->
      <a-layout class="content-layout">
        <a-layout-content class="content">
          <router-view v-slot="{ Component }">
            <keep-alive :include="['GamesView', 'DashboardView']">
              <component :is="Component" />
            </keep-alive>
          </router-view>

          <!-- Global Alert Banner -->
          <alert-banner />
        </a-layout-content>

        <!-- Footer -->
        <a-layout-footer class="footer">
          <span>&copy; 这份作品来自不知名网友Hao和他的星期五</span>
        </a-layout-footer>
      </a-layout>
    </a-layout>
  </a-layout>

  <!-- Global Message -->
  <a-message
    v-model:visible="message.show"
    :type="message.type"
  >
    {{ message.content }}
  </a-message>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, provide } from 'vue'
import { useRouter } from 'vue-router'
import useMenu from '@/hooks/useMenu'
import useLocale from '@/hooks/useLocale'
import AlertBanner from '@/components/AlertBanner.vue'
import {
  IconTrophy,
  IconMenuFold,
  IconMenuUnfold,
  IconUp,
} from '@arco-design/web-vue/es/icon'

const router = useRouter()
const { menuList, activeKey } = useMenu()
const { t } = useLocale()

// App configuration
const appName = 'GameAtlas'
const sideWidth = 240

// State
const collapsed = ref(false)

// Message state
const message = ref({
  show: false,
  content: '',
  type: 'info',
})

// Event handlers
const handleLogoClick = () => {
  router.push({ name: 'dashboard' })
}

const scrollToTop = () => {
  // Find the main content scroll container
  const content = document.querySelector('.content')
  if (content) {
    content.scrollTo({ top: 0, behavior: 'smooth' })
  } else {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

const handleMenuClick = (key: string) => {
  router.push({ name: key })
}

const handleCollapse = (val: boolean) => {
  collapsed.value = val
}

const showMessage = (content: string, type = 'info') => {
  message.value = { show: true, content, type }
  setTimeout(() => {
    message.value.show = false
  }, 3000)
}

// Mobile detection - reactive
const isMobile = ref(false)
const showMobileMenu = ref(false)

// Handle responsive sidebar
// Arco Design breakpoints: xs < 576, sm >= 576, md >= 768, lg >= 992, xl >= 1200
const handleResize = () => {
  const width = window.innerWidth
  // Mobile: < 768px (md breakpoint)
  isMobile.value = width < 768
  // Collapse sidebar on mobile and tablet (< 992px, lg breakpoint)
  if (width < 992) {
    collapsed.value = true
  } else {
    collapsed.value = false
  }
}

onMounted(() => {
  handleResize()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})

// Expose message globally
provide('showMessage', showMessage)
</script>

<style scoped>
.app-layout {
  height: 100vh;
  width: 100%;
  overflow: hidden;
}

.main-layout {
  padding-top: 56px;
  height: 100vh;
  width: 100%;
  box-sizing: border-box;
}

.content-layout {
  height: 100%;
  overflow-y: auto;
}

/* Pro-style Header */
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

.pro-header .header-left {
  display: flex;
  align-items: center;
}

.pro-header .header-right {
  display: flex;
  align-items: center;
}

/* Logo in header (Pro style) */
.pro-header .logo {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--color-primary-6); /* Use primary color for the logo */
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

.content {
  padding: 24px;
  background: transparent; /* Use transparent to let body background through */
  height: calc(100vh - 56px - 48px); /* viewport height - header - footer */
  width: 100%;
  box-sizing: border-box;
  overflow-y: auto;
}

.footer {
  text-align: center;
  color: var(--color-text-3);
  font-size: 13px;
  background: transparent;
  padding: 16px 0;
  border-top: 1px solid var(--color-border-1);
}

/* Mobile Menu Button */
.mobile-menu-btn {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 99;
  width: 56px;
  height: 56px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.mobile-menu-btn :deep(.arco-btn-icon) {
  font-size: 24px;
}

/* Mobile Drawer Header */
.mobile-drawer-header {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--color-primary-6);
  font-size: 18px;
  font-weight: 600;
}

/* Mobile Drawer Menu Styling */
.mobile-drawer :deep(.arco-drawer-body) {
  padding: 0;
}

.mobile-drawer :deep(.arco-menu) {
  width: 100%;
  border-right: none;
}

.mobile-drawer :deep(.arco-menu-item),
.mobile-drawer :deep(.arco-menu-inline-header) {
  min-height: 48px;
  line-height: 48px;
}

</style>

<style>
/* Global styles */
html {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
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
}

/* Arco Layout customization for Pro layout */
.arco-layout {
  width: 100%;
  height: 100%;
  background: transparent; /* Make layout transparent for global background */
}

.arco-layout-sider {
  background: rgba(22, 26, 37, 0.4); 
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-right: 1px solid var(--color-border-1);
  z-index: 99;
  position: relative;
  height: 100%;
}

.arco-layout-content {
  width: 100%;
}

.arco-menu {
  background: transparent;
}

.arco-menu:not(.arco-menu-collapsed) .arco-menu-item,
.arco-menu:not(.arco-menu-collapsed) .arco-menu-inline-header {
  margin: 4px 8px !important;
  border-radius: var(--radius-md) !important;
}

.arco-menu-collapsed .arco-menu-item {
  border-radius: var(--radius-lg) !important;
}

.arco-menu-item {
  transition: all var(--transition-fast) !important;
}

.arco-menu-selected {
  background-color: rgba(26, 159, 255, 0.15) !important;
  color: var(--color-primary-6) !important;
  font-weight: 600;
}

/* Collapse trigger styling */
.arco-layout-sider-trigger {
  background: var(--color-fill-1);
  border-top: 1px solid var(--color-border-1);
  transition: background-color 0.2s;
}

.arco-layout-sider-trigger:hover {
  background: var(--color-fill-2);
}

/* Menu item alignment when collapsed - Arco handles this automatically */
</style>
