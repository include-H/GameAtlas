<template>
  <!-- 后墙 -->
  <div class="store-backwall">
    <div class="store-wall__paper" />
    <div class="store-wall__trim" />
    <div class="store-poster store-poster--left">
      <img
        v-if="posters[0]"
        class="store-poster__img"
        :src="posters[0]"
        alt="新到货"
        decoding="async"
      >
    </div>
    <div class="store-poster store-poster--right">
      <img
        v-if="posters[1]"
        class="store-poster__img"
        :src="posters[1]"
        alt="畅销榜"
        decoding="async"
      >
    </div>
    <div class="store-sign">
      ATLAS&nbsp;GAMES
      <span class="store-sign__sub">阿特拉斯电玩</span>
    </div>
    <div class="store-sign__cord" />
  </div>

  <!-- 地面 -->
  <div class="store-floor">
    <div class="store-floor__rug" />
  </div>

  <!-- 前台 / 游戏吧 -->
  <div class="store-counter">
    <div class="store-counter__top">
      <div class="store-counter__top-glow" />
    </div>
    <div class="store-counter__front">
      <div class="store-counter__trim" />
    </div>
  </div>

  <!-- 灯光 / 氛围 -->
  <div class="store-light" />
  <div class="store-vignette" />
</template>

<script setup lang="ts">
defineProps<{
  posters: string[]
}>()
</script>

<style scoped>
/* ---------- 后墙 ---------- */
.store-backwall {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 480px;
  z-index: 1;
  background:
    linear-gradient(180deg, rgba(58, 44, 34, 0.55), rgba(46, 35, 27, 0) 30%),
    repeating-linear-gradient(90deg, rgba(24, 18, 14, 0.12) 0 0.67px, transparent 0.67px 80px),
    linear-gradient(180deg, #b3986f 0%, #9a7f5b 55%, #846b4b 100%);
  box-shadow: inset 0 -12px 20px rgba(0, 0, 0, 0.35);
}

.store-wall__paper {
  position: absolute;
  inset: 0;
  opacity: 0.18;
  background:
    radial-gradient(ellipse at 50% 42%, rgba(255, 226, 170, 0.28), transparent 58%),
    repeating-linear-gradient(0deg, rgba(255, 240, 210, 0.05) 0 1.33px, transparent 1.33px 6px);
}

.store-wall__trim {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 9.33px;
  background: linear-gradient(180deg, #5d4a36, #3f3024);
  box-shadow: 0 2.67px 6.67px rgba(0, 0, 0, 0.45);
}

/* ---------- 海报 ---------- */
.store-poster {
  position: absolute;
  top: 64px;
  width: 200px;
  aspect-ratio: 16 / 9;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3.33px;
  border-radius: 2.67px;
  overflow: hidden;
  border: 2px solid #e8d5ad;
  box-shadow: 0 6.67px 16px rgba(0, 0, 0, 0.42);
  background:
    linear-gradient(180deg, rgba(30, 22, 16, 0.72), rgba(18, 13, 9, 0.82)),
    #3a2a1c;
}

.store-poster--left {
  left: 60px;
  transform: rotate(-0.8deg) scale(1.33);
}

.store-poster--right {
  right: 60px;
  transform: rotate(0.8deg) scale(1.33);
}

.store-poster__img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

/* ---------- 霓虹招牌 ---------- */
.store-sign {
  position: absolute;
  top: 4px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.67px;
  font-size: 21px;
  font-weight: 700;
  letter-spacing: 4.67px;
  color: #ffd9a0;
  text-shadow:
    0 0 4px rgba(255, 190, 110, 0.95),
    0 0 12px rgba(255, 160, 70, 0.75),
    0 0 28px rgba(255, 130, 40, 0.55);
  background: rgba(30, 20, 14, 0.35);
  padding: 2.67px 12px 4px;
  border-radius: 2.67px;
}

.store-sign__sub {
  font-size: 9px;
  font-weight: 400;
  letter-spacing: 2.67px;
  color: rgba(255, 217, 160, 0.82);
  text-shadow:
    0 0 3.33px rgba(255, 190, 110, 0.8),
    0 0 8px rgba(255, 160, 70, 0.5);
}

.store-sign__cord {
  position: absolute;
  top: 100%;
  left: 50%;
  width: 1.33px;
  height: 9.33px;
  background: rgba(20, 14, 10, 0.8);
}

/* ---------- 地面 ---------- */
.store-floor {
  position: absolute;
  left: -4%;
  right: -4%;
  top: 413.33px;
  bottom: 200px;
  z-index: 1;
  transform: perspective(466.67px) rotateX(32deg);
  transform-origin: top center;
  background:
    repeating-linear-gradient(0deg, rgba(0, 0, 0, 0.22) 0 1.33px, transparent 1.33px 28px),
    linear-gradient(180deg, #4a3525, #2b1d13 90%);
  box-shadow: inset 0 12px 22.67px rgba(0, 0, 0, 0.5);
}

.store-floor__rug {
  position: absolute;
  left: 50%;
  bottom: 6%;
  width: 46%;
  height: 46%;
  transform: translateX(-50%);
  background: radial-gradient(ellipse at 50% 30%, rgba(130, 52, 42, 0.55), rgba(70, 28, 24, 0.18) 70%, transparent 72%);
}

/* ---------- 前台 / 游戏吧 ---------- */
.store-counter {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 200px;
  z-index: 4;
}

.store-counter__top {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 10.67px;
  background:
    linear-gradient(180deg, #b08a5c 0%, #8d6a42 72%, #6f4f2e 100%);
  box-shadow:
    0 4px 8px rgba(0, 0, 0, 0.55),
    inset 0 0.67px 0 rgba(255, 235, 200, 0.4);
  transform: perspective(266.67px) rotateX(24deg);
  transform-origin: bottom center;
  border-radius: 2px 2px 0 0;
}

.store-counter__top-glow {
  position: absolute;
  inset: 6.67px 0 0 0;
  background: linear-gradient(90deg, transparent 20%, rgba(255, 230, 170, 0.25) 50%, transparent 80%);
}

.store-counter__front {
  position: absolute;
  top: 10.67px;
  left: 0;
  right: 0;
  bottom: 0;
  background:
    repeating-linear-gradient(0deg, rgba(0, 0, 0, 0.12) 0 0.67px, transparent 0.67px 6px),
    linear-gradient(180deg, #6f4f2e, #4a3320 78%);
  box-shadow:
    inset 0 0.67px 0 rgba(255, 235, 200, 0.16),
    inset 0 -16px 26.67px rgba(0, 0, 0, 0.5);
}

.store-counter__trim {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: rgba(255, 235, 200, 0.14);
}

/* ---------- 灯光 / 氛围 ---------- */
.store-light {
  position: absolute;
  inset: 0;
  z-index: 5;
  pointer-events: none;
  background:
    radial-gradient(ellipse at 50% -8%, rgba(255, 196, 120, 0.28), transparent 46%),
    radial-gradient(ellipse at 50% 38%, rgba(255, 180, 100, 0.07), transparent 62%);
  mix-blend-mode: screen;
}

.store-vignette {
  position: absolute;
  inset: 0;
  z-index: 6;
  pointer-events: none;
  background:
    linear-gradient(180deg, rgba(10, 7, 5, 0.42), transparent 26%),
    radial-gradient(ellipse at 50% 46%, transparent 52%, rgba(8, 5, 3, 0.62) 100%);
}
</style>
