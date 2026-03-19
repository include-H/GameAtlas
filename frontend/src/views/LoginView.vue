<template>
  <div class="login-page">
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
          'character-scene--typing': isTyping,
          'character-scene--revealed': shouldPeek,
          'character-scene--hiding': isHidingPassword,
          'character-scene--error': isErrorAnimating,
        }"
      >
        <div
          class="character character--purple"
          :class="{ 'character--blink': purpleBlinking }"
          :style="getCharacterStyle('purple')"
        >
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
          <div class="character__eyes character__eyes--pupil-only" :style="getEyesStyle('orange')">
            <div class="character__pupil character__pupil--dark" :style="getPupilStyle('orange')"></div>
            <div class="character__pupil character__pupil--dark" :style="getPupilStyle('orange')"></div>
          </div>
        </div>

        <div class="character character--yellow" :style="getCharacterStyle('yellow')">
          <div class="character__eyes character__eyes--pupil-only" :style="getEyesStyle('yellow')">
            <div class="character__pupil character__pupil--dark" :style="getPupilStyle('yellow')"></div>
            <div class="character__pupil character__pupil--dark" :style="getPupilStyle('yellow')"></div>
          </div>
          <div class="character__mouth" :class="{ 'character__mouth--sad': isErrorAnimating }" :style="getMouthStyle()"></div>
        </div>
      </div>
    </section>

    <section class="login-panel">
      <div class="login-card">
        <button
          type="button"
          class="login-stage__close"
          aria-label="关闭登录页"
          @click="handleClose"
        >
          <icon-close />
        </button>
        <div class="login-card__header">
          <h2>欢迎回来</h2>
          <p>输入管理员密码，继续管理你的游戏库和私有内容。</p>
        </div>

        <form class="login-form" @submit.prevent="handleLogin">
          <label class="login-field">
            <span class="login-field__label">管理员密码</span>
            <div class="login-field__shell" :class="{ 'login-field__shell--focus': isTyping }">
              <div class="login-field__icon">
                <icon-lock />
              </div>
              <a-input
                v-model="password"
                class="login-field__input"
                :type="showPassword ? 'text' : 'password'"
                placeholder="请输入管理员密码"
                allow-clear
                @focus="isTyping = true"
                @blur="isTyping = false"
                @press-enter="handleLogin"
              />
              <button
                type="button"
                class="login-field__toggle"
                :aria-label="showPassword ? '隐藏密码' : '显示密码'"
                @click="togglePasswordVisibility"
              >
                <icon-eye-invisible v-if="showPassword" />
                <icon-eye v-else />
              </button>
            </div>
          </label>

          <a-button
            class="login-submit"
            type="primary"
            size="large"
            long
            :loading="isSubmitting"
            @click="handleLogin"
          >
            进入管理模式
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

const password = ref('')
const showPassword = ref(false)
const isSubmitting = ref(false)
const isTyping = ref(false)

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

const shouldPeek = computed(() => password.value.length > 0 && showPassword.value)
const isHidingPassword = computed(() => password.value.length > 0 && !showPassword.value)

const clamp = (value: number, min: number, max: number) => Math.max(min, Math.min(max, value))

const getMotion = (name: CharacterName): CharacterMotion => {
  const normalizedX = window.innerWidth ? (mouseX.value / window.innerWidth) * 2 - 1 : 0
  const normalizedY = window.innerHeight ? (mouseY.value / window.innerHeight) * 2 - 1 : 0

  const factors: Record<CharacterName, { skew: number; face: number; pupil: number }> = {
    purple: { skew: -7, face: 16, pupil: 6 },
    black: { skew: -5, face: 12, pupil: 5 },
    orange: { skew: -3, face: 10, pupil: 5 },
    yellow: { skew: -4, face: 10, pupil: 5 },
  }

  const factor = factors[name]

  return {
    bodySkew: clamp(normalizedX * factor.skew, -12, 12),
    faceX: clamp(normalizedX * factor.face, -18, 18),
    faceY: clamp(normalizedY * factor.face * 0.7, -12, 12),
    pupilX: clamp(normalizedX * factor.pupil, -6, 6),
    pupilY: clamp(normalizedY * factor.pupil, -6, 6),
  }
}

const getCharacterStyle = (name: CharacterName) => {
  const motion = getMotion(name)

  if (name === 'purple') {
    return {
      transform: shouldPeek.value
        ? 'skewX(0deg) translateX(0)'
        : (isTyping.value || isHidingPassword.value)
          ? `skewX(${motion.bodySkew - 8}deg) translateX(34px)`
          : `skewX(${motion.bodySkew}deg)`,
    }
  }

  if (name === 'black') {
    return {
      transform: shouldPeek.value
        ? 'skewX(0deg)'
        : isTyping.value
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
      ? { left: '24px', top: purplePeeking.value ? '42px' : '34px' }
      : { left: `${52 + motion.faceX}px`, top: `${48 + motion.faceY}px` }
  }

  if (name === 'black') {
    return shouldPeek.value
      ? { left: '14px', top: '26px' }
      : { left: `${28 + motion.faceX}px`, top: `${34 + motion.faceY}px` }
  }

  if (name === 'orange') {
    return shouldPeek.value
      ? { left: '48px', top: '86px' }
      : { left: `${84 + motion.faceX}px`, top: `${92 + motion.faceY}px` }
  }

  return shouldPeek.value
    ? { left: '20px', top: '36px' }
    : { left: `${54 + motion.faceX}px`, top: `${42 + motion.faceY}px` }
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

const handleLogin = async () => {
  if (!password.value.trim()) {
    Message.warning('请输入管理员密码')
    return
  }

  isSubmitting.value = true

  try {
    await authStore.login(password.value)
    const redirect = resolveRedirect()
    router.push(redirect)
  } catch (error: any) {
    triggerErrorAnimation()
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

.character-scene--error .character--yellow .character__eyes,
.character-scene--error .character--yellow .character__mouth {
  animation: face-nope-right 0.72s ease-in-out 0.04s;
}

.character {
  position: absolute;
  bottom: 0;
  transform-origin: bottom center;
  transition: transform 0.7s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.character--purple {
  left: 58px;
  width: 184px;
  height: 400px;
  background: #6c3ff5;
  border-radius: 14px 14px 0 0;
  box-shadow: 0 36px 48px rgba(108, 63, 245, 0.18);
  transition:
    transform 0.9s cubic-bezier(0.2, 0.8, 0.2, 1),
    height 1.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.character-scene--typing .character--purple {
  height: 426px;
}

.character-scene--hiding .character--purple {
  height: 426px;
}

.character--black {
  left: 212px;
  width: 122px;
  height: 314px;
  background: #25262b;
  border-radius: 10px 10px 0 0;
  z-index: 2;
}

.character--orange {
  left: 18px;
  width: 244px;
  height: 206px;
  background: #ff9b6b;
  border-radius: 122px 122px 0 0;
  z-index: 3;
}

.character--yellow {
  left: 284px;
  width: 142px;
  height: 236px;
  background: #e8d754;
  border-radius: 74px 74px 0 0;
  z-index: 4;
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
}

.character__eyes--pupil-only {
  gap: 28px;
}

.character__eye {
  width: 18px;
  height: 18px;
  border-radius: 999px;
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
  border-radius: 999px;
  background: #2d2d2d;
  transition: transform 0.12s ease-out;
}

.character__pupil--small {
  width: 6px;
  height: 6px;
}

.character__pupil--dark {
  width: 12px;
  height: 12px;
}

.character__mouth {
  position: absolute;
  width: 80px;
  height: 4px;
  border-radius: 999px;
  background: #2d2d2d;
  transition: left 0.25s ease, top 0.25s ease;
}

.character__mouth--sad {
  width: 66px;
  height: 18px;
  border-radius: 999px 999px 0 0;
  border-top: 4px solid #2d2d2d;
  background: transparent;
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
  border: 0;
  background: transparent;
  cursor: pointer;
  transition: color 0.2s ease, transform 0.2s ease;
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
</style>
