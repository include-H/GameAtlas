<template>
  <div class="login-container">
    <div class="login-wrapper">
      <a-card class="login-card">
        <!-- Logo/Title -->
        <div class="login-header">
          <icon-trophy :size="64" :style="{ color: 'rgb(var(--primary-6))' }" />
          <h1 class="login-title">GameAtlas</h1>
          <p class="login-subtitle">
            输入管理员密码以查看私有内容和管理游戏库
          </p>
        </div>

        <a-divider />

        <a-space direction="vertical" fill :size="16">
          <a-input-password
            v-model="password"
            size="large"
            placeholder="输入管理员密码"
            allow-clear
            @press-enter="handleLogin"
          />

          <a-button
            class="app-primary-cta app-primary-cta--large"
            type="primary"
            long
            size="large"
            :loading="isSubmitting"
            @click="handleLogin"
          >
            <template #icon>
              <icon-user />
            </template>
            登录管理模式
          </a-button>
        </a-space>
      </a-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Message } from '@arco-design/web-vue'
import { IconTrophy, IconUser } from '@arco-design/web-vue/es/icon'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const password = ref('')
const isSubmitting = ref(false)

const handleLogin = async () => {
  if (!password.value.trim()) {
    Message.warning('请输入管理员密码')
    return
  }

  isSubmitting.value = true
  try {
    await authStore.login(password.value)
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (error: any) {
    Message.error(error?.response?.data?.error || '登录失败')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--color-bg-1) 0%, var(--color-bg-2) 100%);
  padding: 20px;
}

.login-wrapper {
  width: 100%;
  max-width: 420px;
}

.login-card {
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
}

.login-header {
  text-align: center;
  padding: 24px 0;
}

.login-title {
  font-size: 28px;
  font-weight: 600;
  margin: 16px 0 8px;
  color: var(--color-text-1);
}

.login-subtitle {
  color: var(--color-text-3);
  margin: 0;
}

.login-footer {
  text-align: center;
  color: var(--color-text-3);
}

.text-grey {
  color: var(--color-text-3);
}
</style>
