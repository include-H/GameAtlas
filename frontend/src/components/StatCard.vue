<template>
  <a-card
    :style="{
      borderLeft: `4px solid ${color}`,
      height: typeof height === 'number' ? `${height}px` : height 
    }"
    class="stat-card app-glass-surface app-glass-surface--interactive"
    :hoverable="true"
    @click="$emit('click', $event)"
  >
    <div class="stat-card-content">
      <div class="stat-card-main">
        <div>
          <div class="stat-card-title">
            {{ title }}
          </div>
          <div class="stat-card-value">
            {{ value }}
          </div>
        </div>
        <div class="stat-icon-wrapper" :style="{ background: `color-mix(in srgb, ${color} 15%, transparent)` }">
          <component
            :is="iconComponent"
            class="stat-card-icon"
            :style="{ fontSize: '32px', color: color }"
          />
        </div>
      </div>

    </div>
  </a-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'
import {
  IconTrophy,
  IconPlayCircle,
  IconDownload,
  IconHeart,
  IconStar,
  IconUser,
  IconEye,
  IconThunderbolt,
  IconFire,
  IconCheckCircle,
  IconClockCircle
} from '@arco-design/web-vue/es/icon'

interface Props {
  title: string
  value: string | number
  icon: string
  color?: string
  height?: string | number
}

const props = withDefaults(defineProps<Props>(), {
  color: 'rgb(var(--primary-6))',
  height: 120,
})

defineEmits<{
  click: [event: MouseEvent]
}>()

// Map icon names to components
const iconMap: Record<string, Component> = {
  'mdi-gamepad-variant': IconTrophy,
  'mdi-play-circle': IconPlayCircle,
  'mdi-download': IconDownload,
  'mdi-heart': IconHeart,
  'mdi-star': IconStar,
  'mdi-account': IconUser,
  'mdi-eye': IconEye,
  'mdi-flash': IconThunderbolt,
  'mdi-fire': IconFire,
  'mdi-trophy': IconTrophy,
  'mdi-shield': IconCheckCircle,
  'mdi-check-circle': IconCheckCircle,
  'mdi-clock': IconClockCircle,
  'gamepad': IconTrophy,
  'play': IconPlayCircle,
  'download': IconDownload,
  'heart': IconHeart,
  'star': IconStar,
  'user': IconUser,
  'eye': IconEye,
  'bolt': IconThunderbolt,
}

const iconComponent = computed(() => {
  return iconMap[props.icon] || IconStar
})
</script>

<style scoped>
.stat-card {
  cursor: pointer;
  transition: transform var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast);
  overflow: hidden;
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: var(--app-glass-shadow-hover);
}

.stat-card-content {
  color: var(--color-text-1);
}

.stat-card-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.stat-icon-wrapper {
  padding: 12px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-card-value {
  font-size: 32px;
  font-weight: 800;
  color: var(--color-text-1);
  letter-spacing: -0.5px;
}

.stat-card-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-3);
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.stat-card-icon {
  opacity: 0.9;
}

.stat-card-divider {
  background-color: var(--color-border-1);
}
</style>
