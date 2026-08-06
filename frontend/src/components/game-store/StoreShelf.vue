<template>
  <!-- 后方主货架 -->
  <div class="store-shelf">
    <div class="store-shelf__crown" />
    <div class="store-shelf__side store-shelf__side--left" />
    <div class="store-shelf__side store-shelf__side--right" />

    <div class="store-shelf__rows">
      <div v-for="(row, rowIndex) in rows" :key="rowIndex" class="store-shelf__row">
        <div class="store-shelf__row-boxes">
          <GameBox
            v-for="cell in row"
            :key="cell.game.publicId"
            :cell="cell"
            :hovered="hoveredId === cell.game.publicId"
            :picked="pickedId === cell.game.publicId"
            @hover="hoveredId = cell.game.publicId"
            @leave="hoveredId = null"
            @pick="onBoxPick"
          />
        </div>
        <div class="store-shelf__plank">
          <div class="store-shelf__plank-shadow" />
        </div>
      </div>
    </div>

    <div class="store-shelf__base" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { ShelfCell } from '@/composables/useShelfLayout'
import type { StoreShelfGame } from '@/composables/useStoreSession'

defineProps<{
  rows: ShelfCell[][]
  pickedId: string | null
}>()

const emit = defineEmits<{
  (e: 'pick', game: StoreShelfGame, event: MouseEvent): void
}>()

// hover 状态只在货架内部流转，不进主视图
const hoveredId = ref<string | null>(null)

const onBoxPick = (game: StoreShelfGame, event: MouseEvent) => {
  // 点击即离手：清掉 hover 态，避免货架盒子一直停留在“高亮”
  hoveredId.value = null
  emit('pick', game, event)
}
</script>

<style scoped>
/* ---------- 主货架 ---------- */
.store-shelf {
  position: absolute;
  top: 50.67px;
  left: 50%;
  transform: translateX(-50%);
  width: 600px;
  height: 480px;
  z-index: 2;
  background: linear-gradient(180deg, #6d5138, #543c28 18%, #462f1f 100%);
  border: 6.67px solid #3d2b1d;
  border-radius: 4px;
  box-shadow:
    0 16px 29.33px rgba(0, 0, 0, 0.6),
    inset 0 1.33px 0 rgba(255, 230, 190, 0.18),
    inset 0 -1.33px 0 rgba(0, 0, 0, 0.5);
}

.store-shelf__crown {
  position: absolute;
  top: -6.67px;
  left: -6.67px;
  right: -6.67px;
  height: 17.33px;
  background: linear-gradient(180deg, #7d5d3e, #543c28);
  border-radius: 4px 4px 0 0;
  box-shadow: 0 2px 5.33px rgba(0, 0, 0, 0.35);
}

.store-shelf__side {
  position: absolute;
  top: 10.67px;
  bottom: 10.67px;
  width: 12px;
  background: linear-gradient(90deg, #4a3423, #2c1f14);
  box-shadow: inset 0 0 5.33px rgba(0, 0, 0, 0.55);
}

.store-shelf__side--left {
  left: -6.67px;
  border-radius: 2.67px 0 0 2.67px;
}

.store-shelf__side--right {
  right: -6.67px;
  border-radius: 0 2.67px 2.67px 0;
}

.store-shelf__rows {
  position: absolute;
  inset: 10.67px 12px 6.67px;
  display: flex;
  flex-direction: column;
}

.store-shelf__row {
  position: relative;
  flex: 1;
  min-height: 0;
}

.store-shelf__row-boxes {
  position: absolute;
  inset: 0 0 9.33px;
  display: flex;
  justify-content: center;
  align-items: flex-end;
}

.store-shelf__plank {
  position: absolute;
  left: -1.33px;
  right: -1.33px;
  bottom: 0;
  height: 9.33px;
  background:
    linear-gradient(180deg, #8a6a47 0%, #6b4f33 55%, #4e3824 100%);
  border-radius: 1.33px;
  box-shadow:
    0 1.33px 3.33px rgba(0, 0, 0, 0.45),
    inset 0 0.67px 0 rgba(255, 230, 190, 0.22);
  z-index: 5;
}

.store-shelf__plank-shadow {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  height: 12px;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.5), transparent);
}

.store-shelf__base {
  position: absolute;
  left: -6.67px;
  right: -6.67px;
  bottom: -12px;
  height: 14.67px;
  background: linear-gradient(180deg, #543c28, #352516 70%);
  border-radius: 0 0 3.33px 3.33px;
  box-shadow: 0 6.67px 12px rgba(0, 0, 0, 0.55);
}
</style>
