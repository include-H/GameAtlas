<template>
  <div class="register-container">
    <div class="register-wrapper">
      <a-card class="register-card">
        <!-- Logo/Title -->
        <div class="register-header">
          <icon-trophy class="register-icon" />
          <h1 class="register-title">GameAtlas</h1>
          <p class="register-subtitle">
            点击按钮进入游戏库
          </p>
        </div>

        <a-divider />

        <!-- Enter Button -->
        <a-button
          class="app-primary-cta app-primary-cta--large"
          type="primary"
          size="large"
          long
          @click="handleEnter"
        >
          <template #icon>
            <icon-user />
          </template>
          进入系统
        </a-button>

        <a-divider />

        <div class="register-footer">
          <span class="text-grey">当前版本无需单独登录，点击上方按钮即可进入。</span>
        </div>
      </a-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { IconTrophy, IconUser } from '@arco-design/web-vue/es/icon'

const authStore = useAuthStore()

const handleEnter = async () => {
  await authStore.register()
  window.location.href = '/'
}
</script>

<style scoped>
.register-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #121212 0%, #1E1E1E 100%);
  padding: 20px;
}

.register-wrapper {
  width: 100%;
  max-width: 450px;
}

.register-card {
  border-radius: 16px;
}

.register-header {
  text-align: center;
  padding: 24px 0 8px;
}

.register-icon {
  font-size: 48px;
  color: rgb(var(--primary-6));
}

.register-title {
  font-size: 24px;
  font-weight: 600;
  margin: 16px 0 4px;
  color: var(--color-text-1);
}

.register-subtitle {
  color: var(--color-text-3);
  margin: 0;
}

.register-footer {
  text-align: center;
  padding-bottom: 8px;
}

.text-grey {
  color: var(--color-text-3);
}
</style>
