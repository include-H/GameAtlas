<template>
  <div class="login-page">
    <transition name="login-success-overlay">
      <div v-if="showSuccessTransition" class="login-success">
        <div class="login-success__backdrop"></div>
        <div class="login-success__halo"></div>
        <div class="login-success__content">
          <p class="login-success__eyebrow">ACCESS GRANTED</p>
          <h2 class="login-success__title">正在返回首页</h2>
          <p class="login-success__text">GameAtlas 已解锁，正在载入你的管理空间。</p>
          <div class="login-success__track">
            <span class="login-success__bar"></span>
          </div>
        </div>
      </div>
    </transition>

    <section class="login-stage">
      <div class="login-stage__backdrop"></div>
      <div class="login-stage__grid"></div>
      <div class="login-stage__brand">
        <div class="login-stage__brand-mark">
          <icon-trophy />
        </div>
        <div>
          <h1 class="login-stage__title text-gradient">进入管理模式</h1>
          <p class="login-stage__eyebrow">GameAtlas Admin Access</p>
        </div>
      </div>

      <AnimatedCharacters
        :is-typing="isTyping"
        :show-password="showPassword"
        :password-length="password.length"
        :is-error="isErrorAnimating"
        :is-success="showSuccessTransition"
      />
    </section>

    <section class="login-panel">
      <div class="login-card" :class="{ 'login-card--success': showSuccessTransition }">
        <a-button
          class="login-stage__close"
          type="text"
          shape="circle"
          aria-label="关闭登录页"
          :disabled="showSuccessTransition"
          @click="handleClose"
        >
          <icon-close />
        </a-button>
        <div class="login-card__header">
          <h2>欢迎您，{{ adminDisplayName }}</h2>
          <p>请输入访问密码，继续进入你的游戏库与私有内容。</p>
        </div>

        <form class="login-form" @submit.prevent="handleLogin">
          <label class="login-field">
            <span class="login-field__label">访问密码</span>
            <div class="login-field__shell" :class="{ 'login-field__shell--focus': isTyping }">
              <div class="login-field__icon">
                <icon-lock />
              </div>
              <a-input
                v-model="password"
                class="login-field__input"
                :type="showPassword ? 'text' : 'password'"
                placeholder="请输入访问密码"
                allow-clear
                :disabled="showSuccessTransition"
                @focus="handlePasswordFocus"
                @blur="handlePasswordBlur"
              />
              <a-button
                class="login-field__toggle"
                type="text"
                shape="circle"
                :aria-label="showPassword ? '隐藏密码' : '显示密码'"
                :disabled="showSuccessTransition"
                @click="togglePasswordVisibility"
              >
                <icon-eye-invisible v-if="showPassword" />
                <icon-eye v-else />
              </a-button>
            </div>
          </label>
          <p v-if="remainingAttempts !== null && !isCooldownActive" class="login-feedback">
            剩余尝试次数：{{ remainingAttempts }}
          </p>

          <a-button
            class="login-submit"
            type="primary"
            html-type="submit"
            size="large"
            long
            :loading="isSubmitting && !showSuccessTransition"
            :disabled="isLoginDisabled"
          >
            {{ submitButtonText }}
          </a-button>
        </form>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import {
  IconClose,
  IconEye,
  IconEyeInvisible,
  IconLock,
  IconTrophy,
} from '@arco-design/web-vue/es/icon'
import { useAuthStore } from '@/stores/auth'
import { getHttpErrorData, getHttpErrorMessage, getHttpStatus } from '@/utils/http-error'
import AnimatedCharacters from '@/components/login/AnimatedCharacters.vue'

interface LoginErrorData {
  remaining_attempts?: number
  retry_after_seconds?: number
}

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const adminDisplayName = computed(() => authStore.adminDisplayName || 'Admin')

const password = ref('')
const showPassword = ref(false)
const isSubmitting = ref(false)
const isTyping = ref(false)
const showSuccessTransition = ref(false)
const remainingAttempts = ref<number | null>(null)
const cooldownLeftSeconds = ref(0)
const isErrorAnimating = ref(false)

let errorAnimationTimer: number | null = null
let successTransitionTimer: number | null = null
let cooldownTimer: number | null = null

const isCooldownActive = computed(() => cooldownLeftSeconds.value > 0)
const isLoginDisabled = computed(() => showSuccessTransition.value || isCooldownActive.value)
const cooldownLabel = computed(() => {
  const total = Math.max(0, cooldownLeftSeconds.value)
  const minutes = Math.floor(total / 60)
  const seconds = total % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
})
const submitButtonText = computed(() => {
  if (isCooldownActive.value) {
    return `冷静中 ${cooldownLabel.value}`
  }
  return '进入管理模式'
})

const togglePasswordVisibility = () => {
  showPassword.value = !showPassword.value
}

const resolveRedirect = () => {
  const rawRedirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'

  if (!rawRedirect.startsWith('/')) {
    return '/'
  }

  if (rawRedirect === '/login' || rawRedirect.startsWith('/login?')) {
    return '/'
  }

  return rawRedirect
}

const handlePasswordFocus = () => {
  isTyping.value = true
}

const handlePasswordBlur = () => {
  isTyping.value = false
}

const handleClose = () => {
  if (window.history.length > 1) {
    router.back()
    return
  }

  router.push({ name: 'dashboard' })
}

const clearSceneStatusTimers = () => {
  if (errorAnimationTimer) {
    window.clearTimeout(errorAnimationTimer)
    errorAnimationTimer = null
  }

  if (successTransitionTimer) {
    window.clearTimeout(successTransitionTimer)
    successTransitionTimer = null
  }
}

const triggerErrorAnimation = () => {
  isErrorAnimating.value = false

  if (errorAnimationTimer) {
    window.clearTimeout(errorAnimationTimer)
  }

  window.setTimeout(() => {
    isErrorAnimating.value = true
    errorAnimationTimer = window.setTimeout(() => {
      isErrorAnimating.value = false
      errorAnimationTimer = null
    }, 720)
  }, 0)
}

const clearCooldownTimer = () => {
  if (cooldownTimer) {
    window.clearInterval(cooldownTimer)
    cooldownTimer = null
  }
}

const startCooldown = (seconds: number) => {
  clearCooldownTimer()
  cooldownLeftSeconds.value = Math.max(0, Math.floor(seconds))
  if (cooldownLeftSeconds.value <= 0) {
    return
  }

  cooldownTimer = window.setInterval(() => {
    if (cooldownLeftSeconds.value <= 1) {
      cooldownLeftSeconds.value = 0
      clearCooldownTimer()
      return
    }
    cooldownLeftSeconds.value -= 1
  }, 1000)
}

const handleLogin = async () => {
  if (isSubmitting.value) {
    return
  }
  if (!password.value.trim()) {
    Message.warning('请输入访问密码')
    return
  }
  if (isCooldownActive.value) {
    Message.warning(`请在冷静期结束后再试（剩余 ${cooldownLabel.value}）`)
    return
  }

  isSubmitting.value = true

  try {
    await authStore.login(password.value)
    remainingAttempts.value = null
    cooldownLeftSeconds.value = 0
    clearCooldownTimer()
    const redirect = resolveRedirect()
    showSuccessTransition.value = true
    successTransitionTimer = window.setTimeout(() => {
      router.push(redirect)
    }, 980)
  } catch (error) {
    triggerErrorAnimation()
    const status = getHttpStatus(error)
    const data = getHttpErrorData<LoginErrorData>(error) || {}

    if (status === 401) {
      const attempts = Number(data?.remaining_attempts)
      if (Number.isFinite(attempts) && attempts >= 0) {
        remainingAttempts.value = attempts
      }
    }

    if (status === 429) {
      const retryAfter = Number(data?.retry_after_seconds)
      if (Number.isFinite(retryAfter) && retryAfter > 0) {
        startCooldown(retryAfter)
      }
    }

    Message.error(getHttpErrorMessage(error, '登录失败'))
  } finally {
    isSubmitting.value = false
  }
}

onBeforeUnmount(() => {
  clearCooldownTimer()
  clearSceneStatusTimers()
})
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(360px, 480px);
  background:
    radial-gradient(circle at top right, rgba(26, 159, 255, 0.18), transparent 30%),
    radial-gradient(circle at bottom left, rgba(98, 0, 238, 0.12), transparent 34%),
    linear-gradient(180deg, rgba(15, 18, 25, 0.98), rgba(15, 18, 25, 0.9));
  overflow: hidden;
}

.login-success {
  position: fixed;
  inset: 0;
  z-index: 20;
  display: grid;
  place-items: center;
  pointer-events: auto;
}

.login-success__backdrop,
.login-success__halo {
  position: absolute;
  inset: 0;
}

.login-success__backdrop {
  background:
    radial-gradient(circle at 50% 18%, rgba(124, 231, 212, 0.16), transparent 28%),
    radial-gradient(circle at 50% 80%, rgba(75, 141, 255, 0.18), transparent 36%),
    linear-gradient(180deg, rgba(3, 8, 18, 0.18), rgba(3, 8, 18, 0.92));
  backdrop-filter: blur(14px);
}

.login-success__halo {
  inset: 18%;
  border-radius: 999px;
  background: radial-gradient(circle, rgba(110, 247, 214, 0.22), transparent 62%);
  filter: blur(22px);
  animation: login-success-halo 0.95s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.login-success__content {
  position: relative;
  z-index: 1;
  width: min(460px, calc(100vw - 48px));
  padding: 36px 32px;
  border-radius: 28px;
  text-align: center;
  color: rgba(255, 255, 255, 0.96);
  background: rgba(9, 16, 29, 0.72);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.42);
  animation: login-success-card 0.95s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.login-success__eyebrow {
  margin: 0 0 12px;
  color: rgba(255, 255, 255, 0.6);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.36em;
}

.login-success__title {
  margin: 0;
  font-size: clamp(30px, 5vw, 42px);
  line-height: 1.04;
  letter-spacing: -0.04em;
}

.login-success__text {
  margin: 12px 0 20px;
  color: rgba(255, 255, 255, 0.72);
  font-size: 14px;
}

.login-success__track {
  position: relative;
  width: 100%;
  height: 4px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
}

.login-success__bar {
  display: block;
  width: 100%;
  height: 100%;
  transform-origin: left center;
  background: linear-gradient(90deg, #74f6d6 0%, #55a7ff 56%, #ffffff 100%);
  animation: login-success-load 0.82s cubic-bezier(0.22, 1, 0.36, 1) forwards;
}

.login-stage,
.login-panel {
  position: relative;
}

.login-stage {
  min-height: 100vh;
  padding: 48px 56px 40px;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  color: var(--color-text-1);
  isolation: isolate;
}

.login-stage__close {
  position: absolute;
  top: 28px;
  right: 28px;
  z-index: 3;
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 999px;
  background: rgba(22, 26, 37, 0.72);
  color: var(--color-text-2);
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.18);
  cursor: pointer;
  transition: transform 0.2s ease, border-color 0.2s ease, color 0.2s ease, background 0.2s ease;
  padding: 0;
  min-width: 42px;
}

.login-stage__close:hover {
  transform: translateY(-1px);
  color: var(--color-text-1);
  border-color: rgba(26, 159, 255, 0.3);
  background: rgba(28, 34, 48, 0.9);
}

.login-stage__backdrop,
.login-stage__grid {
  position: absolute;
  inset: 0;
}

.login-stage__backdrop {
  background:
    radial-gradient(circle at 18% 18%, rgba(26, 159, 255, 0.14), transparent 26%),
    radial-gradient(circle at 78% 82%, rgba(255, 255, 255, 0.08), transparent 24%);
  z-index: -2;
}

.login-stage__grid {
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.05) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.05) 1px, transparent 1px);
  background-size: 22px 22px;
  mask-image: linear-gradient(to bottom, rgba(0, 0, 0, 0.4), transparent 75%);
  z-index: -1;
}

.login-stage__brand {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  max-width: 640px;
  padding-top: 6px;
}

.login-stage__brand-mark {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border-radius: 14px;
  background: rgba(22, 26, 37, 0.72);
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 18px 36px rgba(0, 0, 0, 0.28);
  font-size: 24px;
  color: rgb(var(--primary-6));
}

.login-stage__eyebrow {
  margin: 10px 0 0;
  font-size: 12px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--color-text-3);
}

.login-stage__title {
  margin: 0;
  font-size: clamp(34px, 4vw, 58px);
  line-height: 1.04;
  letter-spacing: -0.04em;
}

.login-card {
  transition: transform 0.45s ease, opacity 0.45s ease, filter 0.45s ease;
}

.login-card--success {
  opacity: 0.2;
  filter: blur(6px);
  transform: scale(0.98) translateY(16px);
}

.login-success-overlay-enter-active,
.login-success-overlay-leave-active {
  transition: opacity 0.28s ease;
}

.login-success-overlay-enter-from,
.login-success-overlay-leave-to {
  opacity: 0;
}

@keyframes login-success-card {
  0% {
    opacity: 0;
    transform: translateY(24px) scale(0.94);
  }
  38% {
    opacity: 1;
  }
  100% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes login-success-halo {
  0% {
    opacity: 0;
    transform: scale(0.76);
  }
  100% {
    opacity: 1;
    transform: scale(1.04);
  }
}

@keyframes login-success-load {
  0% {
    transform: scaleX(0.04);
    opacity: 0.7;
  }
  100% {
    transform: scaleX(1);
    opacity: 1;
  }
}

.login-panel {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  background: linear-gradient(180deg, rgba(15, 18, 25, 0.44), rgba(15, 18, 25, 0.62));
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border-left: 1px solid rgba(255, 255, 255, 0.05);
}

.login-card {
  position: relative;
  width: min(100%, 420px);
  padding: 36px 30px 30px;
  border-radius: 28px;
  background: rgba(22, 26, 37, 0.76);
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
}

.login-card__header {
  padding-right: 52px;
  margin-bottom: 28px;
}

.login-card__header h2 {
  margin: 0 0 10px;
  font-size: 34px;
  line-height: 1;
  letter-spacing: -0.04em;
  color: var(--color-text-1);
}

.login-card__header p {
  margin: 0;
  color: var(--color-text-2);
  line-height: 1.6;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.login-feedback {
  margin: -2px 2px 0;
  font-size: 13px;
  line-height: 1.45;
  color: rgba(255, 255, 255, 0.72);
}

.login-feedback--warning {
  color: #ffcc80;
}

.login-field {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.login-field__label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-2);
}

.login-field__shell {
  display: grid;
  grid-template-columns: 44px 1fr 44px;
  align-items: center;
  min-height: 58px;
  border-radius: 18px;
  background: rgba(28, 34, 48, 0.84);
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.login-field__shell--focus {
  border-color: rgba(26, 159, 255, 0.42);
  box-shadow: 0 0 0 4px rgba(26, 159, 255, 0.12);
}

.login-field__icon,
.login-field__toggle {
  width: 44px;
  height: 44px;
  display: grid;
  place-items: center;
  color: var(--color-text-3);
  font-size: 18px;
}

.login-field__toggle {
  transition: color 0.2s ease, transform 0.2s ease;
  padding: 0;
  min-width: 44px;
}

.login-field__toggle:hover {
  color: var(--color-text-1);
  transform: scale(1.04);
}

:deep(.login-field__input.arco-input-wrapper) {
  border: 0;
  background: transparent;
  box-shadow: none;
}

:deep(.login-field__input .arco-input) {
  background: transparent;
  padding-left: 0;
  padding-right: 0;
  color: var(--color-text-1);
  font-size: 15px;
}

:deep(.login-field__input .arco-input::placeholder) {
  color: var(--color-text-4);
}

.login-submit {
  height: 56px;
  border-radius: 18px;
  font-size: 15px;
  font-weight: 600;
  background: linear-gradient(135deg, var(--color-primary-6) 0%, #007aff 100%);
  border: 0;
  box-shadow: 0 16px 32px rgba(26, 159, 255, 0.24);
}

.login-submit:hover {
  background: linear-gradient(135deg, var(--color-primary-7) 0%, #3395ff 100%);
}

@media (max-width: 1080px) {
  .login-page {
    grid-template-columns: 1fr;
  }

  .login-stage {
    min-height: auto;
    padding-bottom: 16px;
  }

  .login-panel {
    min-height: auto;
    padding-top: 0;
  }
}

@media (max-width: 768px) {
  .login-stage {
    padding: 28px 20px 10px;
  }

  .login-stage__close {
    top: 18px;
    right: 18px;
  }

  .login-stage__brand {
    gap: 12px;
    padding-top: 2px;
  }

  .login-stage__brand-mark {
    width: 42px;
    height: 42px;
  }

  .login-panel {
    padding: 18px 16px 24px;
    background: transparent;
    backdrop-filter: none;
    -webkit-backdrop-filter: none;
    border-left: 0;
  }

  .login-card {
    padding: 28px 20px 20px;
    border-radius: 22px;
  }

  .login-card__header h2 {
    font-size: 28px;
  }
}

@media (max-width: 576px) {
  .login-stage {
    padding: 20px 12px 8px;
  }

  .login-stage__close {
    top: 12px;
    right: 12px;
  }

  .login-panel {
    padding: 12px 0 20px;
  }

  .login-card {
    padding: 22px 16px 18px;
    border-radius: 18px;
  }

  .login-card__header h2 {
    font-size: 24px;
  }
}
</style>
