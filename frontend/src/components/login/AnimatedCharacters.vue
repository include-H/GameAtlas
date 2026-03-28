<template>
  <div
    class="animated-characters"
    :class="{
      'animated-characters--success': isSuccess,
      'animated-characters--error': props.isError,
    }"
  >
    <div class="animated-characters__scene">
      <div ref="purpleRef" class="character character--purple" :style="purpleStyle">
        <div class="character__eyes character__eyes--purple" :style="purpleEyesStyle">
          <EyeBall
            :size="18"
            :pupil-size="7"
            :max-distance="5"
            eye-color="white"
            pupil-color="#2D2D2D"
            :is-blinking="isPurpleBlinking"
            :force-look-x="purpleForceLook.x"
            :force-look-y="purpleForceLook.y"
          />
          <EyeBall
            :size="18"
            :pupil-size="7"
            :max-distance="5"
            eye-color="white"
            pupil-color="#2D2D2D"
            :is-blinking="isPurpleBlinking"
            :force-look-x="purpleForceLook.x"
            :force-look-y="purpleForceLook.y"
          />
        </div>
      </div>

      <div ref="blackRef" class="character character--black" :style="blackStyle">
        <div class="character__eyes character__eyes--black" :style="blackEyesStyle">
          <EyeBall
            :size="16"
            :pupil-size="6"
            :max-distance="4"
            eye-color="white"
            pupil-color="#2D2D2D"
            :is-blinking="isBlackBlinking"
            :force-look-x="blackForceLook.x"
            :force-look-y="blackForceLook.y"
          />
          <EyeBall
            :size="16"
            :pupil-size="6"
            :max-distance="4"
            eye-color="white"
            pupil-color="#2D2D2D"
            :is-blinking="isBlackBlinking"
            :force-look-x="blackForceLook.x"
            :force-look-y="blackForceLook.y"
          />
        </div>
      </div>

      <div ref="orangeRef" class="character character--orange" :style="orangeStyle">
        <div class="character__eyes character__eyes--orange" :style="orangeEyesStyle">
          <Pupil
            :size="12"
            :max-distance="5"
            pupil-color="#2D2D2D"
            :force-look-x="sharedPupilForceLook.x"
            :force-look-y="sharedPupilForceLook.y"
          />
          <Pupil
            :size="12"
            :max-distance="5"
            pupil-color="#2D2D2D"
            :force-look-x="sharedPupilForceLook.x"
            :force-look-y="sharedPupilForceLook.y"
          />
        </div>
      </div>

      <div ref="yellowRef" class="character character--yellow" :style="yellowStyle">
        <div class="character__eyes character__eyes--yellow" :style="yellowEyesStyle">
          <Pupil
            :size="12"
            :max-distance="5"
            pupil-color="#2D2D2D"
            :force-look-x="sharedPupilForceLook.x"
            :force-look-y="sharedPupilForceLook.y"
          />
          <Pupil
            :size="12"
            :max-distance="5"
            pupil-color="#2D2D2D"
            :force-look-x="sharedPupilForceLook.x"
            :force-look-y="sharedPupilForceLook.y"
          />
        </div>
        <div class="character__mouth character__mouth--yellow" :style="yellowMouthStyle"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import EyeBall from '@/components/login/EyeBall.vue'
import Pupil from '@/components/login/Pupil.vue'

interface FacePosition {
  faceX: number
  faceY: number
  bodySkew: number
}

const props = withDefaults(
  defineProps<{
    isTyping: boolean
    showPassword: boolean
    passwordLength: number
    isSuccess?: boolean
    isError?: boolean
  }>(),
  {
    isSuccess: false,
    isError: false,
  },
)

const mouseX = ref(0)
const mouseY = ref(0)
const isPurpleBlinking = ref(false)
const isBlackBlinking = ref(false)
const isLookingAtEachOther = ref(false)
const isPurplePeeking = ref(false)

const purpleRef = ref<HTMLElement | null>(null)
const blackRef = ref<HTMLElement | null>(null)
const yellowRef = ref<HTMLElement | null>(null)
const orangeRef = ref<HTMLElement | null>(null)

const timers = new Set<number>()
let purplePeekTimer: number | null = null

const isSuccess = computed(() => props.isSuccess)
const isHidingPassword = computed(() => props.passwordLength > 0 && !props.showPassword)
const isRevealed = computed(() => props.passwordLength > 0 && props.showPassword)

const addTimer = (callback: () => void, delay: number) => {
  const id = window.setTimeout(() => {
    timers.delete(id)
    callback()
  }, delay)
  timers.add(id)
  return id
}

const clearAllTimers = () => {
  timers.forEach((id) => window.clearTimeout(id))
  timers.clear()
  purplePeekTimer = null
}

const randomBlinkDelay = () => Math.random() * 4000 + 3000

const schedulePurpleBlink = () => {
  addTimer(() => {
    isPurpleBlinking.value = true
    addTimer(() => {
      isPurpleBlinking.value = false
      schedulePurpleBlink()
    }, 150)
  }, randomBlinkDelay())
}

const scheduleBlackBlink = () => {
  addTimer(() => {
    isBlackBlinking.value = true
    addTimer(() => {
      isBlackBlinking.value = false
      scheduleBlackBlink()
    }, 150)
  }, randomBlinkDelay())
}

const schedulePurplePeek = () => {
  if (purplePeekTimer) {
    window.clearTimeout(purplePeekTimer)
    timers.delete(purplePeekTimer)
    purplePeekTimer = null
  }

  if (!isRevealed.value) {
    isPurplePeeking.value = false
    return
  }

  purplePeekTimer = addTimer(() => {
    isPurplePeeking.value = true
    addTimer(() => {
      isPurplePeeking.value = false
      schedulePurplePeek()
    }, 800)
  }, Math.random() * 3000 + 2000)
}

const handleMouseMove = (event: MouseEvent) => {
  mouseX.value = event.clientX
  mouseY.value = event.clientY
}

const calculatePosition = (element: HTMLElement | null): FacePosition => {
  if (!element) {
    return { faceX: 0, faceY: 0, bodySkew: 0 }
  }

  const rect = element.getBoundingClientRect()
  const centerX = rect.left + rect.width / 2
  const centerY = rect.top + rect.height / 3
  const deltaX = mouseX.value - centerX
  const deltaY = mouseY.value - centerY

  return {
    faceX: Math.max(-15, Math.min(15, deltaX / 20)),
    faceY: Math.max(-10, Math.min(10, deltaY / 30)),
    bodySkew: Math.max(-6, Math.min(6, -deltaX / 120)),
  }
}

const purplePos = computed(() => calculatePosition(purpleRef.value))
const blackPos = computed(() => calculatePosition(blackRef.value))
const yellowPos = computed(() => calculatePosition(yellowRef.value))
const orangePos = computed(() => calculatePosition(orangeRef.value))

const purpleForceLook = computed(() => {
  if (isRevealed.value) {
    return {
      x: isPurplePeeking.value ? 4 : -4,
      y: isPurplePeeking.value ? 5 : -4,
    }
  }
  if (isLookingAtEachOther.value) {
    return { x: 3, y: 4 }
  }
  return { x: undefined, y: undefined }
})

const blackForceLook = computed(() => {
  if (isRevealed.value) {
    return { x: -4, y: -4 }
  }
  if (isLookingAtEachOther.value) {
    return { x: 0, y: -4 }
  }
  return { x: undefined, y: undefined }
})

const sharedPupilForceLook = computed(() => {
  if (isRevealed.value) {
    return { x: -5, y: -4 }
  }
  return { x: undefined, y: undefined }
})

const purpleStyle = computed(() => ({
  left: '70px',
  width: '180px',
  height: props.isTyping || isHidingPassword.value ? '440px' : '400px',
  backgroundColor: '#6C3FF5',
  borderRadius: '10px 10px 0 0',
  zIndex: 1,
  transform: isRevealed.value
    ? 'skewX(0deg)'
    : props.isTyping || isHidingPassword.value
      ? `skewX(${purplePos.value.bodySkew - 12}deg) translateX(40px)`
      : `skewX(${purplePos.value.bodySkew}deg)`,
  transformOrigin: 'bottom center',
}))

const purpleEyesStyle = computed(() => ({
  left: isRevealed.value ? '20px' : isLookingAtEachOther.value ? '55px' : `${45 + purplePos.value.faceX}px`,
  top: isRevealed.value ? '35px' : isLookingAtEachOther.value ? '65px' : `${40 + purplePos.value.faceY}px`,
}))

const blackStyle = computed(() => ({
  left: '240px',
  width: '120px',
  height: '310px',
  backgroundColor: '#2D2D2D',
  borderRadius: '8px 8px 0 0',
  zIndex: 2,
  transform: isRevealed.value
    ? 'skewX(0deg)'
    : isLookingAtEachOther.value
      ? `skewX(${blackPos.value.bodySkew * 1.5 + 10}deg) translateX(20px)`
      : props.isTyping || isHidingPassword.value
        ? `skewX(${blackPos.value.bodySkew * 1.5}deg)`
        : `skewX(${blackPos.value.bodySkew}deg)`,
  transformOrigin: 'bottom center',
}))

const blackEyesStyle = computed(() => ({
  left: isRevealed.value ? '10px' : isLookingAtEachOther.value ? '32px' : `${26 + blackPos.value.faceX}px`,
  top: isRevealed.value ? '28px' : isLookingAtEachOther.value ? '12px' : `${32 + blackPos.value.faceY}px`,
}))

const orangeStyle = computed(() => ({
  left: '0px',
  width: '240px',
  height: '200px',
  zIndex: 3,
  backgroundColor: '#FF9B6B',
  borderRadius: '120px 120px 0 0',
  transform: isRevealed.value ? 'skewX(0deg)' : `skewX(${orangePos.value.bodySkew}deg)`,
  transformOrigin: 'bottom center',
}))

const orangeEyesStyle = computed(() => ({
  left: isRevealed.value ? '50px' : `${82 + orangePos.value.faceX}px`,
  top: isRevealed.value ? '85px' : `${90 + orangePos.value.faceY}px`,
}))

const yellowStyle = computed(() => ({
  left: '310px',
  width: '140px',
  height: '230px',
  backgroundColor: '#E8D754',
  borderRadius: '70px 70px 0 0',
  zIndex: 4,
  transform: isRevealed.value ? 'skewX(0deg)' : `skewX(${yellowPos.value.bodySkew}deg)`,
  transformOrigin: 'bottom center',
}))

const yellowEyesStyle = computed(() => ({
  left: isRevealed.value ? '20px' : `${52 + yellowPos.value.faceX}px`,
  top: isRevealed.value ? '35px' : `${40 + yellowPos.value.faceY}px`,
}))

const yellowMouthStyle = computed(() => ({
  left: isRevealed.value ? '10px' : `${40 + yellowPos.value.faceX}px`,
  top: isRevealed.value ? '88px' : `${88 + yellowPos.value.faceY}px`,
}))

watch(
  () => props.isTyping,
  (isTyping) => {
    if (!isTyping) {
      isLookingAtEachOther.value = false
      return
    }

    isLookingAtEachOther.value = true
    addTimer(() => {
      isLookingAtEachOther.value = false
    }, 800)
  },
)

watch(
  [() => props.passwordLength, () => props.showPassword],
  () => {
    if (!isRevealed.value) {
      isPurplePeeking.value = false
      return
    }
    schedulePurplePeek()
  },
)

onMounted(() => {
  window.addEventListener('mousemove', handleMouseMove)
  schedulePurpleBlink()
  scheduleBlackBlink()
})

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', handleMouseMove)
  clearAllTimers()
})
</script>

<style scoped>
.animated-characters {
  position: relative;
  width: min(770px, 100%);
  height: 560px;
  margin: auto auto;
  overflow: hidden;
  transition: transform 0.7s ease, opacity 0.7s ease, filter 0.7s ease;
}

.animated-characters__scene {
  --scene-scale: 1.1;
  --scene-shift-x: 0px;
  --scene-bottom: 0px;
  position: absolute;
  left: 50%;
  bottom: var(--scene-bottom);
  width: 550px;
  height: 400px;
  transform: translateX(calc(-50% + var(--scene-shift-x))) scale(var(--scene-scale));
  transform-origin: center bottom;
}

.animated-characters--success {
  opacity: 0.18;
  filter: blur(4px);
  transform: scale(0.96) translateY(18px);
}

.animated-characters--error .character__eyes--purple {
  animation: face-nope-left 0.72s ease-in-out;
}

.animated-characters--error .character__eyes--black {
  animation: face-nope-right 0.72s ease-in-out;
}

.animated-characters--error .character__eyes--orange {
  animation: face-nope-left 0.72s ease-in-out 0.04s;
}

.animated-characters--error .character__eyes--yellow,
.animated-characters--error .character__mouth--yellow {
  animation: face-nope-right 0.72s ease-in-out 0.04s;
}

@media (max-width: 1080px) {
  .animated-characters__scene {
    --scene-shift-x: 35px;
  }
}

.character {
  position: absolute;
  bottom: 0;
  transition: all 0.7s ease-in-out;
}

.character__eyes {
  position: absolute;
  display: flex;
  gap: 32px;
  transition: all 0.7s ease-in-out;
}

.character__eyes--black {
  gap: 24px;
}

.character__eyes--orange {
  gap: 32px;
  transition-duration: 0.2s;
  transition-timing-function: ease-out;
}

.character__eyes--yellow {
  gap: 24px;
  transition-duration: 0.2s;
  transition-timing-function: ease-out;
}

.character__mouth {
  position: absolute;
  width: 80px;
  height: 4px;
  background: #2d2d2d;
  border-radius: 999px;
  transition: all 0.2s ease-out;
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

@media (max-width: 768px) {
  .animated-characters {
    height: 420px;
    margin-top: 18px;
  }

  .animated-characters__scene {
    --scene-scale: 0.98;
    --scene-shift-x: 22px;
  }
}

@media (max-width: 576px) {
  .animated-characters {
    height: 320px;
    margin-top: 8px;
  }

  .animated-characters__scene {
    --scene-scale: 0.76;
    --scene-shift-x: 34px;
    --scene-bottom: -60px;
  }
}
</style>
