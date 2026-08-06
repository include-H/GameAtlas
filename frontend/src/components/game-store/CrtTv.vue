<template>
  <div class="store-crt" :class="{ 'store-crt--off': !crtPowered }">
    <div class="store-crt__cabinet">
      <div class="store-crt__screen">
        <video
          ref="crtVideoRef"
          class="store-crt__video"
          :src="crtVideoUrl || undefined"
          autoplay
          muted
          playsinline
          preload="auto"
          @ended="handleCrtVideoEnded"
        />
        <div class="store-crt__glass" />
      </div>
      <div class="store-crt__brand">GAMEATRON</div>
      <div class="store-crt__controls">
        <button
          type="button"
          class="store-crt__knob"
          :class="{ 'is-off': !crtPowered }"
          :title="crtPowered ? '关闭电视' : '打开电视'"
          @click="toggleCrtPower"
        />
        <button
          type="button"
          class="store-crt__knob store-crt__knob--small"
          :class="{ 'is-paused': crtPaused }"
          :title="crtPaused ? '播放' : '暂停'"
          @click="toggleCrtPause"
        />
        <span class="store-crt__led" :class="{ 'is-off': !crtPowered }" />
      </div>
      <div class="store-crt__vents" />
    </div>
    <div class="store-crt__stand" />
    <div class="store-crt__cable" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  playlist: string[]
}>()

const crtVideoUrl = ref('')
const crtPlaylistIndex = ref(0)
const crtPowered = ref(true)
const crtPaused = ref(false)
const crtVideoRef = ref<HTMLVideoElement | null>(null)

// CRT 播放真实预告：Session 加载完成后把整个预告片列表拍平成轮播，播完自动换下一个
watch(
  () => props.playlist,
  (list) => {
    if (list.length === 0) {
      crtPlaylistIndex.value = 0
      crtVideoUrl.value = ''
      return
    }
    crtPlaylistIndex.value = 0
    crtVideoUrl.value = list[0]
  },
)

const handleCrtVideoEnded = () => {
  if (props.playlist.length === 0) return
  crtPlaylistIndex.value = (crtPlaylistIndex.value + 1) % props.playlist.length
  crtVideoUrl.value = props.playlist[crtPlaylistIndex.value]
}

const toggleCrtPower = () => {
  crtPowered.value = !crtPowered.value
  const video = crtVideoRef.value
  if (!video) return
  if (crtPowered.value) {
    void video.play().catch(() => {})
  } else {
    video.pause()
  }
}

const toggleCrtPause = () => {
  if (!crtPowered.value) return
  crtPaused.value = !crtPaused.value
  const video = crtVideoRef.value
  if (!video) return
  if (crtPaused.value) {
    video.pause()
  } else {
    void video.play().catch(() => {})
  }
}
</script>

<style scoped>
/* ---------- CRT 电视 ---------- */
.store-crt {
  position: absolute;
  right: 120px;
  bottom: 200px;
  width: 280px;
  z-index: 3;
  transform: none;
}

.store-crt__cabinet {
  position: relative;
  padding: 10.67px 10.67px 8px;
  border-radius: 12px 12px 8px 8px;
  background:
    linear-gradient(180deg, #d8c9a8 0%, #b8a37e 34%, #8f7a5b 100%);
  border: 1.33px solid #5d4a32;
  box-shadow:
    0 12px 20px rgba(0, 0, 0, 0.55),
    inset 0 1.33px 0 rgba(255, 245, 220, 0.5),
    inset 0 -4px 9.33px rgba(0, 0, 0, 0.28);
}

.store-crt__screen {
  position: relative;
  aspect-ratio: 16 / 9;
  border-radius: 6.67px;
  overflow: hidden;
  background:
    radial-gradient(ellipse at 42% 36%, rgba(90, 120, 96, 0.5), rgba(12, 22, 16, 0.95) 72%),
    #08140c;
  border: 2.67px solid #2c2116;
  box-shadow:
    inset 0 0 17.33px rgba(0, 0, 0, 0.9),
    0 0 14.67px rgba(190, 235, 190, 0.16),
    0 0 40px rgba(160, 220, 160, 0.08);
  transition: background 0.4s ease, box-shadow 0.4s ease;
}

.store-crt__video {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  background: #08140c;
  transition: opacity 0.4s ease;
}

.store-crt__glass {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(118deg, rgba(255, 255, 255, 0.13) 0%, transparent 24%),
    linear-gradient(0deg, rgba(255, 255, 255, 0.05), transparent 40%);
  box-shadow: inset 0 0 26.67px rgba(255, 255, 255, 0.05);
  animation: crt-flicker 7s ease-in-out infinite;
}

.store-crt--off .store-crt__screen {
  background: radial-gradient(ellipse at 42% 36%, rgba(86, 86, 86, 0.26), #050505 78%), #050505;
  box-shadow: inset 0 0 20px rgba(0, 0, 0, 0.95);
}

.store-crt--off .store-crt__video {
  opacity: 0;
}

.store-crt--off .store-crt__glass {
  opacity: 0.25;
  animation: none;
}

.store-crt__brand {
  margin-top: 5.33px;
  text-align: center;
  font-size: 6px;
  letter-spacing: 2.67px;
  color: #4a3824;
  font-weight: 700;
}

.store-crt__controls {
  position: absolute;
  right: 12px;
  bottom: 8.33px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.store-crt__knob {
  appearance: none;
  padding: 0;
  cursor: pointer;
  width: 9.33px;
  height: 9.33px;
  border-radius: 50%;
  background: radial-gradient(circle at 35% 30%, #efe3c8, #8a7556 70%);
  border: 0.67px solid #4a3824;
  box-shadow: 0 1.33px 2px rgba(0, 0, 0, 0.4);
}

.store-crt__knob:hover {
  filter: brightness(1.12);
}

.store-crt__knob.is-off {
  background: radial-gradient(circle at 35% 30%, #6f6a5e, #4a4238 70%);
}

.store-crt__knob--small {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 6.67px;
  height: 6.67px;
}

.store-crt__knob--small::after {
  content: '❚❚';
  font-size: 4.67px;
  line-height: 1;
  color: rgba(30, 20, 10, 0.85);
}

.store-crt__knob--small.is-paused::after {
  content: '▶';
  font-size: 4px;
}

.store-crt__led {
  width: 3.33px;
  height: 3.33px;
  border-radius: 50%;
  background: #7be37b;
  box-shadow: 0 0 4px #7be37b;
  animation: crt-breath 4.8s ease-in-out infinite;
}

.store-crt__led.is-off {
  background: #5d2222;
  box-shadow: 0 0 2.67px rgba(140, 40, 40, 0.5);
  animation: none;
}

.store-crt__vents {
  position: absolute;
  left: 12px;
  bottom: 12px;
  width: 56px;
  height: 6.67px;
  background: repeating-linear-gradient(90deg, #6d5a3e 0 1.33px, transparent 1.33px 3.33px);
  border-radius: 1.33px;
  opacity: 0.75;
}

.store-crt__stand {
  width: 72%;
  height: 14.67px;
  margin: 0 auto;
  background: linear-gradient(180deg, #7d6240, #4e3824);
  border-radius: 0 0 5.33px 5.33px;
  box-shadow: 0 8px 12px rgba(0, 0, 0, 0.5);
}

.store-crt__cable {
  position: absolute;
  right: -10%;
  bottom: -25.33px;
  width: 60%;
  height: 34.67px;
  border: 2px solid #241a12;
  border-top: 0;
  border-left: 0;
  border-radius: 0 0 40px 0;
  opacity: 0.85;
}

/* ---------- 动画 ---------- */
@keyframes crt-noise {
  0% { background-position: 0 0; }
  25% { background-position: -20px 8px; }
  50% { background-position: 12px -14.67px; }
  75% { background-position: -8px -20px; }
  100% { background-position: 16px 12px; }
}

@keyframes crt-flicker {
  0%, 100% { opacity: 1; }
  92% { opacity: 1; }
  93% { opacity: 0.78; }
  94% { opacity: 1; }
  97% { opacity: 0.9; }
  98% { opacity: 1; }
}

@keyframes crt-breath {
  0%, 100% { opacity: 0.55; }
  50% { opacity: 1; }
}
</style>
