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

      <div
        class="character-scene"
        :class="{
          'character-scene--entering': isSceneEntering,
          'character-scene--typing': isProtectingPassword,
          'character-scene--revealed': shouldPeek,
          'character-scene--hiding': isHidingPassword,
          'character-scene--error': isErrorAnimating,
          'character-scene--success': showSuccessTransition,
        }"
      >
        <div
          class="character character--purple"
          :class="{ 'character--blink': purpleBlinking }"
          :style="getCharacterStyle('purple')"
        >
          <div class="character__hat character__hat--mage"></div>
          <div class="character__cape"></div>
          <div class="character__trim character__trim--mage"></div>
          <div class="character__belt character__belt--mage"></div>
          <div class="character__staff"></div>
          <div class="character__hand character__hand--mage"></div>
          <div class="character__rune"></div>
          <div class="character__eyes" :style="getEyesStyle('purple')">
            <div class="character__eye">
              <div class="character__pupil" :style="getPupilStyle('purple')"></div>
            </div>
            <div class="character__eye">
              <div class="character__pupil" :style="getPupilStyle('purple')"></div>
            </div>
          </div>
        </div>

        <div
          class="character character--black"
          :class="{ 'character--blink': blackBlinking }"
          :style="getCharacterStyle('black')"
        >
          <div class="character__visor"></div>
          <div class="character__trim character__trim--robot"></div>
          <div class="character__panel"></div>
          <div class="character__core"></div>
          <div class="character__antenna"></div>
          <div class="character__eyes" :style="getEyesStyle('black')">
            <div class="character__eye character__eye--small">
              <div class="character__pupil character__pupil--small" :style="getPupilStyle('black')"></div>
            </div>
            <div class="character__eye character__eye--small">
              <div class="character__pupil character__pupil--small" :style="getPupilStyle('black')"></div>
            </div>
          </div>
        </div>

        <div class="character character--orange" :style="getCharacterStyle('orange')">
          <div class="character__shine"></div>
          <div class="character__crown"></div>
          <div class="character__grin" :class="{ 'character__grin--sad': isErrorAnimating }"></div>
          <div class="character__eyes character__eyes--pupil-only" :style="getEyesStyle('orange')">
            <div class="character__pupil character__pupil--dark" :style="getPupilStyle('orange')"></div>
            <div class="character__pupil character__pupil--dark" :style="getPupilStyle('orange')"></div>
          </div>
        </div>

        <div class="character character--yellow" :style="getCharacterStyle('yellow')">
          <div class="character__helmet"></div>
          <div class="character__visor character__visor--knight"></div>
          <div class="character__trim character__trim--knight"></div>
          <div class="character__shield"></div>
          <div class="character__tabard"></div>
          <div class="character__eyes character__eyes--pupil-only" :style="getEyesStyle('yellow')">
            <div class="character__pupil character__pupil--dark" :style="getPupilStyle('yellow')"></div>
            <div class="character__pupil character__pupil--dark" :style="getPupilStyle('yellow')"></div>
          </div>
          <div class="character__mouth" :class="{ 'character__mouth--sad': isErrorAnimating }" :style="getMouthStyle()"></div>
        </div>
      </div>
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
                @focus="isTyping = true"
                @blur="isTyping = false"
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
import { computed, onBeforeUnmount, onMounted, ref, watchEffect } from 'vue'
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

type CharacterName = 'purple' | 'black' | 'orange' | 'yellow'

interface CharacterMotion {
  bodySkew: number
  faceX: number
  faceY: number
  pupilX: number
  pupilY: number
}

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const adminDisplayName = (import.meta.env.VITE_USERNAME || 'Admin').trim() || 'Admin'

const password = ref('')
const showPassword = ref(false)
const isSubmitting = ref(false)
const isTyping = ref(false)
const showSuccessTransition = ref(false)
const remainingAttempts = ref<number | null>(null)
const cooldownLeftSeconds = ref(0)

const mouseX = ref(0)
const mouseY = ref(0)
const isSceneEntering = ref(true)
const purpleBlinking = ref(false)
const blackBlinking = ref(false)
const purplePeeking = ref(false)
const isErrorAnimating = ref(false)

let enterAnimationTimer: number | null = null
let purpleBlinkTimer: number | null = null
let blackBlinkTimer: number | null = null
let purplePeekTimer: number | null = null
let errorAnimationTimer: number | null = null
let successTransitionTimer: number | null = null
let cooldownTimer: number | null = null

const shouldPeek = computed(() => password.value.length > 0 && showPassword.value)
const isHidingPassword = computed(() => password.value.length > 0 && !showPassword.value)
const isProtectingPassword = computed(() => isTyping.value && !showPassword.value)
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

const clamp = (value: number, min: number, max: number) => Math.max(min, Math.min(max, value))
const amplifyMotion = (value: number) => Math.sign(value) * Math.pow(Math.abs(value), 0.8)

const getMotion = (name: CharacterName): CharacterMotion => {
  const normalizedX = window.innerWidth ? (mouseX.value / window.innerWidth) * 2 - 1 : 0
  const normalizedY = window.innerHeight ? (mouseY.value / window.innerHeight) * 2 - 1 : 0
  const motionX = amplifyMotion(normalizedX)
  const motionY = amplifyMotion(normalizedY)

  const factors: Record<CharacterName, { skew: number; face: number; pupil: number }> = {
    purple: { skew: -8, face: 18, pupil: 7 },
    black: { skew: -6, face: 14, pupil: 6 },
    orange: { skew: -6, face: 17, pupil: 7 },
    yellow: { skew: -5, face: 12, pupil: 6 },
  }

  const factor = factors[name]

  return {
    bodySkew: clamp(motionX * factor.skew, -13, 13),
    faceX: clamp(motionX * factor.face, -20, 20),
    faceY: clamp(motionY * factor.face * 0.74, -14, 14),
    pupilX: clamp(motionX * factor.pupil, -7, 7),
    pupilY: clamp(motionY * factor.pupil, -7, 7),
  }
}

const getCharacterStyle = (name: CharacterName) => {
  const motion = getMotion(name)

  if (name === 'purple') {
    return {
      transform: shouldPeek.value
        ? 'skewX(0deg) translateX(0)'
        : (isProtectingPassword.value || isHidingPassword.value)
          ? `skewX(${motion.bodySkew - 8}deg) translateX(34px)`
          : `skewX(${motion.bodySkew}deg)`,
    }
  }

  if (name === 'black') {
    return {
      transform: shouldPeek.value
        ? 'skewX(0deg)'
        : isProtectingPassword.value
          ? `skewX(${motion.bodySkew + 8}deg) translateX(12px)`
          : `skewX(${motion.bodySkew}deg)`,
    }
  }

  return {
    transform: shouldPeek.value ? 'skewX(0deg)' : `skewX(${motion.bodySkew}deg)`,
  }
}

const getEyesStyle = (name: CharacterName) => {
  const motion = getMotion(name)

  if (name === 'purple') {
    return shouldPeek.value
      ? { left: '18px', top: purplePeeking.value ? '42px' : '34px' }
      : { left: `${38 + motion.faceX}px`, top: `${64 + motion.faceY}px` }
  }

  if (name === 'black') {
    return shouldPeek.value
      ? { left: '14px', top: '26px' }
      : { left: `${28 + motion.faceX}px`, top: `${34 + motion.faceY}px` }
  }

  if (name === 'orange') {
    return shouldPeek.value
      ? { left: '44px', top: '74px' }
      : { left: `${76 + motion.faceX}px`, top: `${78 + motion.faceY}px` }
  }

  if (name === 'yellow') {
    return shouldPeek.value
      ? { left: '20px', top: '36px' }
      : { left: `${48 + motion.faceX * 0.92}px`, top: `${48 + motion.faceY}px` }
  }

  return { left: '0', top: '0' }
}

const getPupilStyle = (name: CharacterName) => {
  if (shouldPeek.value) {
    if (name === 'purple') {
      return {
        transform: `translate(${purplePeeking.value ? 5 : -5}px, ${purplePeeking.value ? 5 : -4}px)`,
      }
    }

    return { transform: 'translate(-5px, -4px)' }
  }

  const motion = getMotion(name)
  return {
    transform: `translate(${motion.pupilX}px, ${motion.pupilY}px)`,
  }
}

const getMouthStyle = () => {
  const motion = getMotion('yellow')
  return shouldPeek.value
    ? { left: '12px', top: '92px' }
    : { left: `${42 + motion.faceX}px`, top: `${92 + motion.faceY}px` }
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

const handleMouseMove = (event: MouseEvent) => {
  mouseX.value = event.clientX
  mouseY.value = event.clientY
}

const handleClose = () => {
  if (window.history.length > 1) {
    router.back()
    return
  }

  router.push({ name: 'dashboard' })
}

const scheduleBlink = (target: typeof purpleBlinking, setter: (value: number | null) => void) => {
  const run = () => {
    const nextTimer = window.setTimeout(() => {
      target.value = true
      window.setTimeout(() => {
        target.value = false
        run()
      }, 160)
    }, 2600 + Math.random() * 2800)

    setter(nextTimer)
  }

  run()
}

const schedulePurplePeek = () => {
  if (!shouldPeek.value) {
    purplePeeking.value = false
    if (purplePeekTimer) {
      window.clearTimeout(purplePeekTimer)
      purplePeekTimer = null
    }
    return
  }

  purplePeekTimer = window.setTimeout(() => {
    purplePeeking.value = true
    window.setTimeout(() => {
      purplePeeking.value = false
      schedulePurplePeek()
    }, 850)
  }, 1800 + Math.random() * 2600)
}

const clearTimers = () => {
  if (enterAnimationTimer) {
    window.clearTimeout(enterAnimationTimer)
    enterAnimationTimer = null
  }

  if (purpleBlinkTimer) {
    window.clearTimeout(purpleBlinkTimer)
    purpleBlinkTimer = null
  }

  if (blackBlinkTimer) {
    window.clearTimeout(blackBlinkTimer)
    blackBlinkTimer = null
  }

  if (purplePeekTimer) {
    window.clearTimeout(purplePeekTimer)
    purplePeekTimer = null
  }

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
  } catch (error: any) {
    triggerErrorAnimation()
    const status = Number(error?.response?.status || 0)
    const data = error?.response?.data?.data || {}

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

    Message.error(error?.response?.data?.error || '登录失败')
  } finally {
    isSubmitting.value = false
  }
}

onMounted(() => {
  mouseX.value = window.innerWidth * 0.6
  mouseY.value = window.innerHeight * 0.4
  enterAnimationTimer = window.setTimeout(() => {
    isSceneEntering.value = false
    enterAnimationTimer = null
  }, 1500)
  window.addEventListener('mousemove', handleMouseMove)
  scheduleBlink(purpleBlinking, (value) => {
    purpleBlinkTimer = value
  })
  scheduleBlink(blackBlinking, (value) => {
    blackBlinkTimer = value
  })
})

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', handleMouseMove)
  clearCooldownTimer()
  clearTimers()
})

watchEffect(() => {
  schedulePurplePeek()
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

.character-scene {
  position: relative;
  width: min(100%, 500px);
  height: 430px;
  margin: auto auto;
  transition: transform 0.55s ease, opacity 0.55s ease, filter 0.55s ease;
}

.character-scene--success {
  opacity: 0.18;
  filter: blur(4px);
  transform: scale(0.96) translateY(18px);
}

.login-card {
  transition: transform 0.45s ease, opacity 0.45s ease, filter 0.45s ease;
}

.login-card--success {
  opacity: 0.2;
  filter: blur(6px);
  transform: scale(0.98) translateY(16px);
}

.character-scene--error .character--purple .character__eyes {
  animation: face-nope-left 0.72s ease-in-out;
}

.character-scene--error .character--black .character__eyes {
  animation: face-nope-right 0.72s ease-in-out;
}

.character-scene--error .character--orange .character__eyes {
  animation: face-nope-left 0.72s ease-in-out 0.04s;
}

.character-scene--error .character--orange .character__grin {
  animation: face-nope-left 0.72s ease-in-out 0.04s;
}

.character-scene--error .character--yellow .character__eyes,
.character-scene--error .character--yellow .character__mouth {
  animation: face-nope-right 0.72s ease-in-out 0.04s;
}

.character {
  position: absolute;
  bottom: 0;
  transform-origin: bottom center;
  transition: transform 0.7s cubic-bezier(0.2, 0.8, 0.2, 1);
  image-rendering: pixelated;
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

.character--purple {
  left: 62px;
  width: 156px;
  height: 408px;
  background: linear-gradient(180deg, #8357ff 0%, #6c3ff5 40%, #4f2ec8 100%);
  border-radius: 8px 8px 0 0;
  box-shadow:
    0 0 0 4px rgba(43, 26, 99, 0.82),
    0 36px 48px rgba(108, 63, 245, 0.18);
  transition:
    transform 0.9s cubic-bezier(0.2, 0.8, 0.2, 1),
    height 1.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.character-scene--typing .character--purple {
  height: 432px;
}

.character-scene--hiding .character--purple {
  height: 432px;
}

.character--black {
  left: 212px;
  width: 122px;
  height: 314px;
  background: linear-gradient(180deg, #3c414d 0%, #1d2028 100%);
  border-radius: 6px 6px 0 0;
  box-shadow: 0 0 0 4px rgba(12, 15, 22, 0.86);
  z-index: 2;
}

.character--orange {
  left: 20px;
  width: 228px;
  height: 184px;
  background: radial-gradient(circle at 50% 18%, #ffc58a 0%, #ffad6d 22%, #ff8f5a 60%, #e46c39 100%);
  border-radius: 64px 64px 12px 12px;
  box-shadow: 0 0 0 4px rgba(126, 58, 28, 0.72);
  z-index: 3;
}

.character--yellow {
  left: 286px;
  width: 136px;
  height: 242px;
  background: linear-gradient(180deg, #ffec7a 0%, #e4d34d 46%, #b79d2a 100%);
  border-radius: 18px 18px 0 0;
  box-shadow: 0 0 0 4px rgba(123, 96, 27, 0.78);
  z-index: 4;
}

.character--yellow::before,
.character--yellow::after {
  content: '';
  position: absolute;
  top: 18px;
  width: 16px;
  height: 74px;
  border-radius: 4px;
  background: linear-gradient(180deg, #f2de7b 0%, #d5bc52 100%);
  box-shadow:
    0 0 0 4px rgba(123, 96, 27, 0.6),
    inset 0 -6px 0 rgba(118, 89, 16, 0.16);
}

.character--yellow::before {
  left: -8px;
}

.character--yellow::after {
  right: -8px;
}

.character-scene--entering .character--purple {
  animation: character-enter-purple 1.2s cubic-bezier(0.2, 0.9, 0.2, 1) 0.08s both;
}

.character-scene--entering .character--black {
  animation: character-enter-black 1.05s cubic-bezier(0.2, 0.9, 0.2, 1) 0.2s both;
}

.character-scene--entering .character--orange {
  animation: character-enter-orange 1s cubic-bezier(0.18, 0.88, 0.22, 1) both;
}

.character-scene--entering .character--yellow {
  animation: character-enter-yellow 1.1s cubic-bezier(0.2, 0.9, 0.2, 1) 0.14s both;
}

.character__eyes {
  position: absolute;
  display: flex;
  gap: 24px;
  transition: left 0.25s ease, top 0.25s ease;
  z-index: 2;
}

.character__eyes--pupil-only {
  gap: 18px;
}

.character__eye {
  width: 18px;
  height: 18px;
  border-radius: 4px;
  display: grid;
  place-items: center;
  background: #fff;
  transition: height 0.16s ease, transform 0.16s ease;
  overflow: hidden;
}

.character__eye--small {
  width: 16px;
  height: 16px;
}

.character--blink .character__eye {
  height: 2px;
}

.character__pupil {
  width: 7px;
  height: 7px;
  border-radius: 2px;
  background: #2d2d2d;
  transition: transform 0.12s ease-out;
}

.character__pupil--small {
  width: 6px;
  height: 6px;
}

.character__pupil--dark {
  width: 10px;
  height: 10px;
  border-radius: 2px;
}

.character__mouth {
  position: absolute;
  width: 80px;
  height: 4px;
  border-radius: 999px;
  background: #2d2d2d;
  transition: left 0.25s ease, top 0.25s ease;
  z-index: 2;
}

.character__mouth--sad {
  width: 66px;
  height: 18px;
  border-radius: 999px 999px 0 0;
  border-top: 4px solid #2d2d2d;
  background: transparent;
}

.character__hat,
.character__cape,
.character__trim,
.character__belt,
.character__hand,
.character__visor,
.character__panel,
.character__shine,
.character__crown,
.character__helmet,
.character__staff,
.character__rune,
.character__core,
.character__antenna,
.character__gem,
.character__grin,
.character__shield,
.character__tabard {
  position: absolute;
  pointer-events: none;
}

.character__hat--mage {
  top: -50px;
  left: -6px;
  width: 132px;
  height: 102px;
  clip-path: polygon(12% 100%, 20% 80%, 25% 58%, 38% 24%, 60% 0, 75% 10%, 69% 42%, 80% 62%, 94% 100%);
  background: linear-gradient(180deg, #6c45d8 0%, #5731bd 56%, #41238f 100%);
  box-shadow:
    0 0 0 4px rgba(48, 27, 100, 0.82),
    inset -6px -8px 0 rgba(31, 16, 77, 0.24);
  transform: rotate(-2deg);
  transform-origin: bottom center;
  z-index: 2;
}

.character__hat--mage::after {
  content: '';
  position: absolute;
  left: -22px;
  top: 82px;
  width: 176px;
  height: 16px;
  border-radius: 2px;
  background: linear-gradient(180deg, #f0df8f 0%, #d6b95c 100%);
  box-shadow:
    0 0 0 4px rgba(76, 45, 150, 0.82),
    inset 0 -4px 0 rgba(130, 94, 17, 0.18);
}

.character__hat--mage::before {
  content: '';
  position: absolute;
  left: 58px;
  top: 56px;
  width: 16px;
  height: 16px;
  border-radius: 3px;
  background: linear-gradient(180deg, #8ceaff 0%, #2da4e8 100%);
  box-shadow: 0 0 0 4px rgba(31, 83, 128, 0.72);
}

.character__cape {
  left: -18px;
  right: -10px;
  top: 88px;
  height: 210px;
  clip-path: polygon(0 0, 100% 0, 86% 100%, 14% 100%);
  background: linear-gradient(180deg, #342170 0%, #23144a 100%);
  box-shadow: 0 0 0 4px rgba(44, 25, 96, 0.74);
  z-index: -1;
}

.character__trim--mage {
  left: 34px;
  right: 34px;
  top: 102px;
  height: 30px;
  border-radius: 2px;
  clip-path: polygon(0 0, 100% 0, 70% 100%, 30% 100%);
  background: linear-gradient(180deg, rgba(244, 225, 134, 0.96) 0%, rgba(218, 190, 88, 0.96) 100%);
  z-index: 2;
}

.character__belt--mage {
  left: 18px;
  right: 18px;
  bottom: 98px;
  height: 18px;
  border-radius: 2px;
  background: linear-gradient(180deg, #f6d36e 0%, #b47c25 100%);
  box-shadow: inset 0 0 0 4px rgba(92, 48, 11, 0.28);
}

.character__belt--mage::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 28px;
  height: 22px;
  transform: translate(-50%, -50%);
  border-radius: 2px;
  background: #6b4517;
  box-shadow: inset 0 0 0 4px #e8d47f;
}

.character__staff {
  left: 116px;
  top: 46px;
  width: 12px;
  height: 238px;
  border-radius: 2px;
  background: linear-gradient(180deg, #8d6738 0%, #5d3c1b 100%);
  box-shadow: 0 0 0 4px rgba(64, 35, 12, 0.7);
  z-index: 3;
}

.character__staff::before {
  content: '';
  position: absolute;
  left: -10px;
  top: -18px;
  width: 28px;
  height: 28px;
  border-radius: 4px;
  background: linear-gradient(180deg, #6ff0ff 0%, #2798df 100%);
  box-shadow: 0 0 0 4px rgba(28, 89, 142, 0.78);
}

.character__hand--mage {
  left: 112px;
  top: 148px;
  width: 24px;
  height: 20px;
  border-radius: 4px;
  background: linear-gradient(180deg, #f1dc84 0%, #cfb354 100%);
  box-shadow:
    0 0 0 4px rgba(79, 46, 136, 0.72),
    inset 0 -4px 0 rgba(133, 97, 19, 0.18);
  z-index: 3;
}

.character__hand--mage::before {
  content: '';
  position: absolute;
  right: -12px;
  top: -2px;
  width: 16px;
  height: 24px;
  border-radius: 0 4px 4px 0;
  background: linear-gradient(180deg, #5d35bd 0%, #45268e 100%);
  box-shadow: 0 0 0 4px rgba(48, 27, 100, 0.56);
}

.character__hand--mage::after {
  content: '';
  position: absolute;
  right: 4px;
  top: 4px;
  width: 6px;
  height: 12px;
  border-radius: 2px;
  background: rgba(120, 84, 28, 0.42);
}

.character__rune {
  left: 26px;
  bottom: 54px;
  width: 34px;
  height: 34px;
  border-radius: 4px;
  background:
    linear-gradient(90deg, transparent 42%, rgba(120, 244, 255, 0.9) 42%, rgba(120, 244, 255, 0.9) 58%, transparent 58%),
    linear-gradient(transparent 42%, rgba(120, 244, 255, 0.9) 42%, rgba(120, 244, 255, 0.9) 58%, transparent 58%);
  box-shadow: 0 0 0 4px rgba(51, 43, 118, 0.72);
  opacity: 0.88;
}

.character__visor {
  left: 14px;
  right: 14px;
  top: 22px;
  height: 46px;
  border-radius: 4px;
  background: linear-gradient(180deg, #6ef3ff 0%, #2789c8 100%);
  opacity: 0.26;
  box-shadow: inset 0 0 0 4px rgba(255, 255, 255, 0.12);
}

.character__trim--robot {
  left: 18px;
  right: 18px;
  top: 106px;
  height: 12px;
  border-radius: 2px;
  background: linear-gradient(90deg, #2fe3ff 0%, #7af7ff 50%, #2fe3ff 100%);
  box-shadow: 0 0 18px rgba(47, 227, 255, 0.32);
}

.character__panel {
  left: 34px;
  right: 34px;
  bottom: 52px;
  height: 84px;
  border-radius: 4px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.08), transparent 36%),
    linear-gradient(180deg, #313845 0%, #1f2430 100%);
  box-shadow: inset 0 0 0 4px rgba(255, 255, 255, 0.08);
}

.character__panel::before,
.character__panel::after {
  content: '';
  position: absolute;
  top: 22px;
  width: 14px;
  height: 14px;
  border-radius: 999px;
  background: #72f2ff;
  box-shadow: 0 0 12px rgba(114, 242, 255, 0.5);
}

.character__panel::before {
  left: 18px;
}

.character__panel::after {
  right: 18px;
}

.character__core {
  left: 42px;
  top: 146px;
  width: 38px;
  height: 38px;
  border-radius: 4px;
  background: linear-gradient(180deg, #92fbff 0%, #29d7ff 100%);
  box-shadow:
    0 0 0 4px rgba(19, 78, 116, 0.82),
    0 0 18px rgba(52, 228, 255, 0.28);
}

.character__core::before,
.character__core::after {
  content: '';
  position: absolute;
  background: rgba(17, 52, 86, 0.86);
}

.character__core::before {
  left: 14px;
  top: 5px;
  width: 10px;
  height: 28px;
}

.character__core::after {
  left: 5px;
  top: 14px;
  width: 28px;
  height: 10px;
}

.character__antenna {
  left: 52px;
  top: -26px;
  width: 8px;
  height: 30px;
  background: #667285;
  box-shadow: 0 0 0 3px rgba(27, 30, 37, 0.72);
}

.character__antenna::before {
  content: '';
  position: absolute;
  left: -6px;
  top: -10px;
  width: 20px;
  height: 20px;
  border-radius: 4px;
  background: #7df5ff;
  box-shadow: 0 0 0 4px rgba(18, 78, 98, 0.78);
}

.character__shine {
  top: 30px;
  left: 46px;
  width: 54px;
  height: 18px;
  border-radius: 2px;
  background: rgba(255, 245, 214, 0.34);
  transform: skewX(-24deg);
}

.character__crown {
  top: -8px;
  left: 88px;
  width: 46px;
  height: 18px;
  background: linear-gradient(180deg, #ffe38b 0%, #f1bd38 100%);
  clip-path: polygon(0 100%, 8% 44%, 24% 66%, 38% 18%, 52% 66%, 68% 10%, 82% 66%, 92% 38%, 100% 100%);
  filter: drop-shadow(0 0 0 rgba(0, 0, 0, 0));
  box-shadow: 0 0 0 4px rgba(160, 105, 18, 0.55);
}

.character__grin {
  left: 100px;
  top: 110px;
  width: 24px;
  height: 8px;
  border-radius: 0 0 8px 8px;
  border-bottom: 4px solid rgba(115, 44, 25, 0.9);
  background: transparent;
  transition: border-color 0.2s ease, transform 0.2s ease;
}

.character__grin--sad {
  height: 9px;
  border-radius: 8px 8px 0 0;
  border-bottom: 0;
  border-top: 4px solid rgba(115, 44, 25, 0.9);
}

.character__helmet {
  top: -12px;
  left: 12px;
  width: 112px;
  height: 70px;
  border-radius: 10px 10px 6px 6px;
  background: linear-gradient(180deg, #fff0a4 0%, #e8d26a 54%, #b9972c 100%);
  box-shadow:
    0 0 0 4px rgba(123, 96, 27, 0.78),
    inset 0 -8px 0 rgba(118, 89, 16, 0.24);
  z-index: 1;
}

.character__helmet::before {
  content: '';
  position: absolute;
  left: 46px;
  top: -18px;
  width: 16px;
  height: 30px;
  border-radius: 8px 8px 4px 4px;
  background: linear-gradient(180deg, #e06a47 0%, #be4429 100%);
  box-shadow: 0 0 0 2px rgba(120, 42, 28, 0.28);
}

.character__helmet::after {
  content: '';
  position: absolute;
  left: 4px;
  right: 4px;
  bottom: -4px;
  height: 18px;
  border-radius: 0 0 6px 6px;
  background: linear-gradient(180deg, #b89a2d 0%, #8c6f16 100%);
  box-shadow: 0 4px 0 rgba(88, 65, 10, 0.18);
}

.character__visor--knight {
  left: 36px;
  right: 36px;
  top: 54px;
  height: 6px;
  border-radius: 2px;
  background: linear-gradient(180deg, rgba(120, 92, 18, 0.96) 0%, rgba(78, 58, 9, 0.96) 100%);
  box-shadow:
    0 0 0 2px rgba(173, 148, 64, 0.18),
    inset 0 1px 0 rgba(255, 234, 153, 0.1);
  opacity: 1;
  z-index: 3;
}

.character__shield {
  left: -24px;
  bottom: 32px;
  width: 52px;
  height: 72px;
  border-radius: 6px 6px 12px 12px;
  background: linear-gradient(180deg, #7bc8ff 0%, #447cb6 100%);
  box-shadow: 0 0 0 4px rgba(41, 73, 113, 0.82);
}

.character__shield::before {
  content: '';
  position: absolute;
  left: 20px;
  top: 12px;
  width: 12px;
  height: 42px;
  background: rgba(233, 245, 255, 0.82);
}

.character__shield::after {
  content: '';
  position: absolute;
  left: 10px;
  top: 27px;
  width: 32px;
  height: 12px;
  background: rgba(233, 245, 255, 0.82);
}

.character__tabard {
  left: 40px;
  bottom: 28px;
  width: 40px;
  height: 78px;
  clip-path: polygon(0 0, 100% 0, 100% 84%, 50% 100%, 0 84%);
  background: linear-gradient(180deg, #f56f4c 0%, #b83724 100%);
  box-shadow: 0 0 0 4px rgba(120, 42, 28, 0.66);
}

.character--yellow .character__eyes {
  top: 58px !important;
  z-index: 3;
}

.character--yellow .character__mouth {
  width: 58px;
  z-index: 3;
}

.character__trim--knight {
  left: 18px;
  right: 18px;
  top: 118px;
  height: 12px;
  border-radius: 2px;
  background: linear-gradient(90deg, #c63c29 0%, #ef6c4c 50%, #c63c29 100%);
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

@keyframes face-nope-left {
  0%,
  100% {
    transform: translateX(0);
  }

  20% {
    transform: translateX(-8px);
  }

  40% {
    transform: translateX(8px);
  }

  60% {
    transform: translateX(-6px);
  }

  80% {
    transform: translateX(5px);
  }
}

@keyframes face-nope-right {
  0%,
  100% {
    transform: translateX(0);
  }

  20% {
    transform: translateX(8px);
  }

  40% {
    transform: translateX(-8px);
  }

  60% {
    transform: translateX(6px);
  }

  80% {
    transform: translateX(-5px);
  }
}

@keyframes character-enter-purple {
  0% {
    opacity: 0;
    transform: translate(-90px, 12px) scale(0.94);
  }

  65% {
    opacity: 1;
    transform: translate(10px, -4px) scale(1.02);
  }

  100% {
    opacity: 1;
    transform: translate(0, 0) scale(1);
  }
}

@keyframes character-enter-black {
  0% {
    opacity: 0;
    transform: translateY(72px) scale(0.92);
  }

  60% {
    opacity: 1;
    transform: translateY(-6px) scale(1.02);
  }

  100% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes character-enter-orange {
  0% {
    opacity: 0;
    transform: translate(-72px, 40px) scale(0.9);
  }

  60% {
    opacity: 1;
    transform: translate(8px, -4px) scale(1.02);
  }

  100% {
    opacity: 1;
    transform: translate(0, 0) scale(1);
  }
}

@keyframes character-enter-yellow {
  0% {
    opacity: 0;
    transform: translate(74px, -32px) scale(0.9);
  }

  62% {
    opacity: 1;
    transform: translate(-6px, 4px) scale(1.03);
  }

  100% {
    opacity: 1;
    transform: translate(0, 0) scale(1);
  }
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

  .character-scene {
    height: 320px;
    transform: scale(0.74);
    transform-origin: center bottom;
    margin-top: 18px;
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

  .character-scene {
    height: 280px;
    transform: scale(0.64);
    margin-top: 8px;
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
