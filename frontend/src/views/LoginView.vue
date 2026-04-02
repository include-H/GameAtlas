<template>
  <div class="login-page">
    <section class="login-stage">
      <div class="login-stage__backdrop"></div>
      <div class="login-stage__grid"></div>
      <div class="login-stage__brand">
        <div class="login-stage__brand-mark" aria-hidden="true">
          <span class="login-stage__brand-mark-core">
            <icon-trophy class="login-stage__brand-icon" />
          </span>
        </div>
        <div class="login-stage__brand-copy">
          <h1 class="login-stage__title text-gradient">进入管理模式</h1>
          <p class="login-stage__eyebrow">GameAtlas Admin Access</p>
        </div>
      </div>

      <AnimatedCharacters
        :is-typing="isTyping"
        :show-password="showPassword"
        :password-length="password.length"
        :is-error="isErrorAnimating"
        :is-success="false"
      />
    </section>

    <section class="login-panel">
      <div class="login-card">
        <a-button
          class="app-text-action-btn login-stage__close"
          type="text"
          shape="circle"
          aria-label="关闭登录页"
          @click="handleClose"
        >
          <icon-close />
        </a-button>
        <div class="login-card__header">
          <h2>欢迎您</h2>
          <p>{{ loginCardDescription }}</p>
          <span v-if="loginQuoteSource" class="login-card__quote-source">{{ loginQuoteSource }}</span>
        </div>

        <form class="login-form" @submit.prevent="handleLogin">
          <label class="login-field">
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
                @focus="handlePasswordFocus"
                @blur="handlePasswordBlur"
              />
              <a-button
                class="app-text-action-btn login-field__toggle"
                type="text"
                shape="circle"
                :aria-label="showPassword ? '隐藏密码' : '显示密码'"
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
            :loading="isSubmitting"
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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
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
import hitokotoService from '@/services/hitokoto.service'
import { getHttpErrorData, getHttpErrorMessage, getHttpStatus } from '@/utils/http-error'
import AnimatedCharacters from '@/components/login/AnimatedCharacters.vue'

interface LoginErrorData {
  remaining_attempts?: number
  retry_after_seconds?: number
}

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const fallbackLoginQuote = '输入访问密码，继续进入你的游戏库。'

const password = ref('')
const showPassword = ref(false)
const isSubmitting = ref(false)
const isTyping = ref(false)
const remainingAttempts = ref<number | null>(null)
const cooldownLeftSeconds = ref(0)
const isErrorAnimating = ref(false)
const loginQuoteText = ref(fallbackLoginQuote)
const loginQuoteSource = ref('')

let errorAnimationTimer: number | null = null
let cooldownTimer: number | null = null

const isCooldownActive = computed(() => cooldownLeftSeconds.value > 0)
const isLoginDisabled = computed(() => isCooldownActive.value)
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
const loginCardDescription = computed(() => loginQuoteText.value || fallbackLoginQuote)

const loadLoginQuote = async () => {
  try {
    const sentence = await hitokotoService.getGameSentence({
      min_length: 10,
      max_length: 34,
    })
    loginQuoteText.value = sentence.hitokoto || fallbackLoginQuote
    loginQuoteSource.value = sentence.from ? `《${sentence.from}》` : ''
  } catch {
    loginQuoteText.value = fallbackLoginQuote
    loginQuoteSource.value = ''
  }
}

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
    router.push(redirect)
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

onMounted(() => {
  void loadLoginQuote()
})

onBeforeUnmount(() => {
  clearCooldownTimer()
  clearSceneStatusTimers()
})
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(400px, 560px);
  background:
    radial-gradient(circle at top right, rgba(122, 162, 199, 0.14), transparent 32%),
    radial-gradient(circle at bottom left, rgba(67, 87, 110, 0.16), transparent 36%),
    linear-gradient(180deg, rgba(6, 10, 16, 0.42), rgba(8, 12, 18, 0.58));
  overflow: hidden;
  position: relative;
  z-index: 1;
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
  background: transparent;
  z-index: -2;
}

.login-stage__grid {
  background: transparent;
  mask-image: none;
  z-index: -1;
}

.login-stage__brand {
  display: flex;
  align-items: center;
  gap: 20px;
  max-width: 720px;
  padding-top: 6px;
}

.login-stage__brand-copy {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.login-stage__brand-mark {
  position: relative;
  width: 72px;
  height: 72px;
  flex: none;
  display: grid;
  place-items: center;
  overflow: hidden;
  border-radius: 24px;
  background: linear-gradient(180deg, rgba(19, 26, 38, 0.94), rgba(10, 14, 22, 0.88));
  border: 1px solid rgba(176, 196, 216, 0.18);
  box-shadow:
    0 22px 44px rgba(0, 0, 0, 0.3),
    inset 0 1px 0 rgba(255, 255, 255, 0.08);
}

.login-stage__brand-mark::before,
.login-stage__brand-mark::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.login-stage__brand-mark::before {
  background:
    linear-gradient(135deg, rgba(220, 231, 241, 0.24), rgba(26, 159, 255, 0.08) 42%, transparent 76%),
    radial-gradient(circle at 28% 24%, rgba(95, 197, 255, 0.22), transparent 42%);
}

.login-stage__brand-mark::after {
  inset: 18px;
  border-radius: 20px;
  background: radial-gradient(circle, rgba(95, 197, 255, 0.24), transparent 72%);
  filter: blur(8px);
}

.login-stage__brand-mark-core {
  position: relative;
  z-index: 1;
  width: 52px;
  height: 52px;
  display: grid;
  place-items: center;
  border-radius: 18px;
  background: linear-gradient(160deg, rgba(103, 148, 196, 0.22), rgba(26, 159, 255, 0.08));
  border: 1px solid rgba(196, 214, 230, 0.14);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.16),
    0 10px 24px rgba(7, 17, 30, 0.28);
}

.login-stage__brand-icon {
  font-size: 30px;
  color: #93dbff;
  transform: translateY(-1px);
  filter: drop-shadow(0 6px 14px rgba(26, 159, 255, 0.3));
}

.login-stage__eyebrow {
  margin: 0;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(219, 229, 238, 0.62);
}

.login-stage__title {
  margin: 0;
  font-size: clamp(34px, 4vw, 58px);
  line-height: 1;
  letter-spacing: -0.04em;
  text-shadow: 0 12px 28px rgba(26, 159, 255, 0.16);
}

.login-panel {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  background: transparent;
  backdrop-filter: none;
  -webkit-backdrop-filter: none;
  border-left: 0;
}

.login-card {
  position: relative;
  width: min(100%, 468px);
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

.login-card__quote-source {
  display: inline-block;
  margin-top: 10px;
  font-size: 12px;
  letter-spacing: 0.08em;
  color: var(--color-text-4);
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
    padding: 28px 20px 0;
  }

  .login-stage__close {
    top: 18px;
    right: 18px;
  }

  .login-stage__brand {
    gap: 14px;
    padding-top: 2px;
  }

  .login-stage__brand-mark {
    width: 60px;
    height: 60px;
    border-radius: 20px;
  }

  .login-stage__brand-mark::after {
    inset: 14px;
  }

  .login-stage__brand-mark-core {
    width: 44px;
    height: 44px;
    border-radius: 15px;
  }

  .login-stage__brand-icon {
    font-size: 25px;
  }

  .login-stage__brand-copy {
    gap: 8px;
  }

  .login-panel {
    margin-top: -18px;
    padding: 18px 16px 24px;
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
    padding: 20px 12px 0;
  }

  .login-stage__brand {
    gap: 12px;
  }

  .login-stage__brand-mark {
    width: 54px;
    height: 54px;
    border-radius: 18px;
  }

  .login-stage__brand-mark::after {
    inset: 12px;
  }

  .login-stage__brand-mark-core {
    width: 40px;
    height: 40px;
    border-radius: 14px;
  }

  .login-stage__brand-icon {
    font-size: 23px;
  }

  .login-stage__close {
    top: 12px;
    right: 12px;
  }

  .login-panel {
    margin-top: -22px;
    padding: 12px 10px 20px;
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
