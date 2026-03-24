<template>
  <div class="card-row">
    <!-- Row Header -->
    <div class="card-row__header">
      <div class="card-row__header-left">
        <component :is="iconComponent" class="card-row__icon" />
        <h3 class="card-row__title">{{ title }}</h3>
        <a-tag
          v-if="count !== undefined"
          size="small"
          class="card-row__count"
        >
          {{ count }}
        </a-tag>
      </div>

      <div class="card-row__header-right">
        <!-- Scroll Buttons -->
        <div v-if="showScrollButtons && items.length > 0" class="card-row__controls">
          <a-button
            class="app-secondary-compact"
            size="small"
            type="text"
            @click="scrollLeft"
          >
            <template #icon>
              <icon-left />
            </template>
          </a-button>
          <a-button
            class="app-secondary-compact"
            size="small"
            type="text"
            @click="scrollRight"
          >
            <template #icon>
              <icon-right />
            </template>
          </a-button>
        </div>

        <a-button
          v-if="showViewAll"
          type="text"
          size="small"
          @click="router.push(viewAllRoute)"
        >
          查看全部
          <template #icon>
            <icon-right />
          </template>
        </a-button>
      </div>
    </div>

    <!-- Horizontal Scroll Container -->
    <div class="card-row__scroll-container">
      <div
        ref="scrollContainer"
        class="card-row__scroll"
      >
        <div
          v-for="(item, index) in items"
          :key="item.id"
          class="card-row__item"
          :style="{ width: cardWidth }"
        >
          <slot name="item" :item="item" :index="index" />
        </div>

        <!-- Empty state -->
        <div v-if="items.length === 0" class="card-row__empty">
          <icon-folder class="card-row__empty-icon" />
          <p class="card-row__empty-text">{{ emptyMessage }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import type { Component } from 'vue'
import { useRouter } from 'vue-router'
import type { RouteLocationRaw } from 'vue-router'
import type { Game } from '@/services/types'
import {
  IconRight,
  IconLeft,
  IconFolder,
  IconPlus,
  IconDownload,
  IconHeart,
  IconEye,
  IconStar,
  IconFire,
  IconThunderbolt,
  IconTrophy
} from '@arco-design/web-vue/es/icon'

interface Props {
  title: string
  icon: string
  items: Game[]
  cardWidth?: string
  showViewAll?: boolean
  viewAllRoute?: RouteLocationRaw
  showScrollButtons?: boolean
  emptyMessage?: string
  count?: number
}

const props = withDefaults(defineProps<Props>(), {
  cardWidth: '200px',
  showViewAll: true,
  viewAllRoute: '/',
  showScrollButtons: true,
  emptyMessage: 'No items found',
})

const router = useRouter()
const scrollContainer = ref<HTMLElement | null>(null)
const canScrollLeft = ref(false)
const canScrollRight = ref(false)

// Map icon names to components
const iconMap: Record<string, Component> = {
  'mdi-new-box': IconPlus,
  'mdi-download': IconDownload,
  'mdi-heart': IconHeart,
  'mdi-eye': IconEye,
  'mdi-star': IconStar,
  'mdi-fire': IconFire,
  'mdi-flash': IconThunderbolt,
  'mdi-gamepad-variant': IconTrophy,
  'mdi-crown': IconStar,
  'mdi-trophy': IconTrophy,
  'new-box': IconPlus,
  'download': IconDownload,
  'heart': IconHeart,
  'eye': IconEye,
  'star': IconStar,
  'fire': IconFire,
  'bolt': IconThunderbolt,
  'gamepad': IconTrophy,
  'crown': IconStar,
  'trophy': IconTrophy,
}

const iconComponent = computed(() => {
  return iconMap[props.icon] || IconStar
})

const checkScroll = () => {
  if (!scrollContainer.value) return

  canScrollLeft.value = scrollContainer.value.scrollLeft > 0
  canScrollRight.value = scrollContainer.value.scrollLeft <
    scrollContainer.value.scrollWidth - scrollContainer.value.clientWidth - 10
}

const scrollLeft = () => {
  if (!scrollContainer.value) return

  const scrollAmount = scrollContainer.value.clientWidth * 0.8
  scrollContainer.value.scrollBy({
    left: -scrollAmount,
    behavior: 'smooth',
  })
}

const scrollRight = () => {
  if (!scrollContainer.value) return

  const scrollAmount = scrollContainer.value.clientWidth * 0.8
  scrollContainer.value.scrollBy({
    left: scrollAmount,
    behavior: 'smooth',
  })
}

onMounted(() => {
  if (scrollContainer.value) {
    scrollContainer.value.addEventListener('scroll', checkScroll)
    checkScroll()
  }
})

watch(
  () => props.items.length,
  async () => {
    await nextTick()
    checkScroll()
  },
)

onUnmounted(() => {
  if (scrollContainer.value) {
    scrollContainer.value.removeEventListener('scroll', checkScroll)
  }
})
</script>

<style scoped>
.card-row {
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--color-border-2);
}

.card-row:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.card-row__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  position: relative;
  z-index: 1;
}

.card-row__header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-row__icon {
  font-size: 16px;
  color: var(--color-text-2);
}

.card-row__title {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
  color: var(--color-text-1);
}

.card-row__count {
  margin-left: 8px;
}

.card-row__scroll-container {
  position: relative;
}

.card-row__scroll {
  display: flex;
  gap: 16px;
  overflow-x: auto;
  overflow-y: hidden;
  padding: 4px 0;
  scroll-snap-type: x mandatory;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
  scrollbar-color: transparent transparent;
}

.card-row__scroll::-webkit-scrollbar {
  width: 0 !important;
  height: 0 !important;
  display: none !important;
}

.card-row__item {
  flex-shrink: 0;
  scroll-snap-align: start;
}

.card-row__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-width: 200px;
  padding: 32px;
}

.card-row__empty-icon {
  font-size: 48px;
  color: var(--color-text-3);
}

.card-row__empty-text {
  margin: 8px 0 0;
  color: var(--color-text-3);
}

.card-row__header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-row__controls {
  display: flex;
  gap: 8px;
}

@media (max-width: 768px) {
  .card-row {
    margin-bottom: 20px;
    padding-bottom: 20px;
  }

  .card-row__header {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
  }

  .card-row__header-right {
    width: 100%;
    justify-content: space-between;
  }
}

@media (max-width: 576px) {
  .card-row__header-right {
    flex-wrap: wrap;
    gap: 8px;
  }

  .card-row__controls {
    display: none;
  }
}
</style>
