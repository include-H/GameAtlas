<template>
  <card-row
    :title="title"
    :icon="icon"
    :items="items"
    :count="items.length"
    :show-view-all="true"
    :view-all-route="viewAllRoute"
  >
    <template #item="{ item }">
      <game-card
        :game="item"
        @view="handleView"
        @view-series="handleViewSeries"
        @toggle-favorite="handleToggleFavorite"
      />
    </template>
  </card-row>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import CardRow from '@/components/CardRow.vue'
import GameCard from '@/components/GameCard.vue'
import type { GameListItem } from '@/services/types'

defineProps<{
  title: string
  icon: string
  items: GameListItem[]
  viewAllRoute?: RouteLocationRaw
}>()

const emit = defineEmits<{
  view: [publicId: string]
  'view-series': [id: number]
  'toggle-favorite': [gameRef: string]
}>()

const handleView = (publicId: string) => emit('view', publicId)
const handleViewSeries = (id: number) => emit('view-series', id)
const handleToggleFavorite = (gameRef: string) => emit('toggle-favorite', gameRef)
</script>
