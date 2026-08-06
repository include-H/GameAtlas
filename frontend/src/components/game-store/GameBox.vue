<template>
  <button
    type="button"
    class="game-box"
    :class="{
      'game-box--hovered': hovered,
      'game-box--picked': picked,
    }"
    :style="style"
    :title="cell.game.title"
    @mouseenter="emit('hover')"
    @mouseleave="emit('leave')"
    @click="onClick"
  >
    <img
      class="game-box__cover"
      :src="cell.game.coverUrl"
      :alt="cell.game.title"
      draggable="false"
    >
    <span class="game-box__sheen" />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { boxStyle, type ShelfCell } from '@/composables/useShelfLayout'

const props = defineProps<{
  cell: ShelfCell
  hovered: boolean
  picked: boolean
}>()

const emit = defineEmits<{
  (e: 'hover'): void
  (e: 'leave'): void
  (e: 'pick', game: ShelfCell['game'], event: MouseEvent): void
}>()

const style = computed(() => boxStyle(props.cell))

const onClick = (event: MouseEvent) => {
  emit('pick', props.cell.game, event)
}
</script>

<style scoped>
/* ---------- 游戏盒 ---------- */
.game-box {
  appearance: none;
  border: 0;
  padding: 0;
  margin: 0 14.67px;
  position: relative;
  /* 盒身比例统一 0.72（宽:高 = 18:25），与检查盒一致；
     封面源图 600×900 用 object-fit: cover 轻微裁左右出血 */
  width: 72px;
  height: 100px;
  aspect-ratio: 0.72;
  background: transparent;
  cursor: pointer;
  outline: none;
  overflow: hidden;
  transform:
    translate(var(--dx), var(--dy))
    rotate(var(--rot));
  transform-style: preserve-3d;
  z-index: var(--box-z, 10);
  transition:
    transform 0.3s cubic-bezier(0.22, 0.61, 0.36, 1),
    box-shadow 0.3s ease,
    filter 0.3s ease;
  filter: brightness(0.94);
}

.game-box__cover {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  background: #241a12;
  box-shadow:
    inset 0 0 0 0.67px rgba(255, 255, 255, 0.08),
    0 3.33px 6px rgba(0, 0, 0, 0.42);
}

/* 盒脊：右侧厚度 */
.game-box::after {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  width: 2.67px;
  height: 100%;
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.24), rgba(0, 0, 0, 0.5));
  border-radius: 0 1.33px 1.33px 0;
  z-index: 2;
}

/* 盒顶厚度 */
.game-box::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.3), rgba(0, 0, 0, 0.2));
  border-radius: 1.33px 1.33px 0 0;
  z-index: 1;
}

.game-box__sheen {
  position: absolute;
  inset: 0;
  z-index: 2;
  pointer-events: none;
  background:
    linear-gradient(115deg, rgba(255, 255, 255, 0.16) 0%, transparent 26%),
    linear-gradient(0deg, rgba(0, 0, 0, 0.28), transparent 34%);
}

.game-box--hovered,
.game-box:hover {
  z-index: 60 !important;
  transform:
    translate(calc(var(--dx) - 5.33px), calc(var(--dy) - 4.67px))
    rotate(var(--rot))
    scale(1.08);
  box-shadow: 0 14.67px 22.67px rgba(0, 0, 0, 0.58);
  filter: brightness(1.08);
}

.game-box--hovered .game-box__cover,
.game-box:hover .game-box__cover {
  box-shadow:
    inset 0 0 0 0.67px rgba(255, 255, 255, 0.14),
    0 9.33px 16px rgba(0, 0, 0, 0.55);
}

/* 已拿在手里的那盒：货架上原位置留空，模拟“盒子在我手上” */
.game-box--picked {
  visibility: hidden;
  pointer-events: none;
}
</style>
