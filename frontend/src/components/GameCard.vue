<template>
  <div
    :class="['game-card hover-lift', { 'game-card--list': isList }]"
    @click="$emit('view', String(game.id))"
  >
    <!-- Cover Image -->
    <div class="game-card__image-wrapper">
      <img
        :src="displayImage"
        :alt="game.title"
        class="game-card__image"
      />

      <!-- Overlay with gradient -->
      <div class="game-card__overlay" />

      <!-- Favorite badge -->
      <a-tag
        v-if="game.isFavorite"
        color="red"
        class="game-card__favorite"
      >
        <template #icon>
          <icon-heart-fill />
        </template>
      </a-tag>
    </div>

    <!-- Card Content -->
    <div class="game-card__content">
      <!-- Row 1: Title and Year -->
      <div class="game-card__row game-card__row--title">
        <div class="game-card__title" :title="game.title">
          {{ game.title }}
        </div>
        <span v-if="game.release_date" class="game-card__year">
          {{ game.release_date.split('-')[0] }}
        </span>
      </div>

      <!-- Row 2: Developer and Actions -->
      <div class="game-card__row game-card__row--metadata">
        <span class="game-card__developer" :title="game.developers && game.developers.length > 0 ? game.developers[0].name : ''">
          {{ game.developers && game.developers.length > 0 ? game.developers[0].name : '' }}
        </span>
        
        <!-- Card Actions moved inside metadata row -->
        <div v-if="!isList" class="game-card__actions">
          <a-button
            class="app-text-compact"
            type="text"
            size="small"
            :class="{ 'is-favorite': game.isFavorite }"
            @click.stop="$emit('toggle-favorite', String(game.id))"
          >
            <template #icon>
              <icon-heart v-if="!game.isFavorite" />
              <icon-heart-fill v-else />
            </template>
          </a-button>
          <a-dropdown v-if="isAdmin">
            <a-button
              class="app-text-compact"
              type="text"
              size="small"
              @click.stop
            >
              <template #icon>
                <icon-more />
              </template>
            </a-button>
            <template #content>
              <a-doption @click="$emit('delete', String(game.id))" style="color: rgb(var(--danger-6));">
                <template #icon>
                  <icon-delete />
                </template>
                删除
              </a-doption>
            </template>
          </a-dropdown>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import type { Game } from '@/services/types'
import { useAuthStore } from '@/stores/auth'
import {
  IconHeartFill,
  IconHeart,
  IconMore,
  IconDelete
} from '@arco-design/web-vue/es/icon'

interface Props {
  game: Game
  isList?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isList: false,
})

const authStore = useAuthStore()
const { isAdmin } = storeToRefs(authStore)

defineEmits<{
  view: [id: string]
  'toggle-favorite': [id: string]
  delete: [id: string]
}>()

const placeholderImage = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"%3E%3Cpath fill="%23424242" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/%3E%3C/svg%3E'

const displayImage = computed(() => {
  if (props.isList) {
    // List mode prefers horizontal banner
    return props.game.banner_image || props.game.cover_image || placeholderImage
  }
  // Grid mode prefers vertical cover
  return props.game.cover_image || props.game.banner_image || placeholderImage
})
</script>

<style scoped>
.game-card {
  position: relative;
  cursor: pointer;
  background: var(--color-bg-2);
  border-radius: var(--radius-lg);
  overflow: hidden;
  margin-bottom: 5px;
  border: 1px solid var(--color-border-1);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  transition: all var(--transition-fast);
  display: flex;
  flex-direction: column;
  height: 100%;
}

.game-card:hover {
  border-color: rgba(26, 159, 255, 0.3);
  box-shadow: var(--shadow-hover);
}

.game-card--list {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 5px;
}

.game-card--list .game-card__image-wrapper {
  width: 160px;
  height: 90px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.game-card--list .game-card__image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.game-card--list .game-card__content {
  flex: 1;
  padding: 12px 16px;
}

.game-card__image-wrapper {
  position: relative;
  width: 100%;
  aspect-ratio: 3/4;
  overflow: hidden;
}

.game-card__image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.game-card__overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, rgba(0, 0, 0, 0.7) 0%, rgba(0, 0, 0, 0) 60%);
  opacity: 0.8;
  transition: opacity var(--transition-fast);
}

.game-card:hover .game-card__overlay {
  opacity: 1;
}

.game-card__favorite {
  position: absolute;
  top: 8px;
  left: 8px;
}

.game-card__content {
  padding: 12px 16px;
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
}

.game-card__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
}

.game-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.game-card__year {
  font-size: 12px;
  color: var(--color-text-3);
  flex-shrink: 0;
}

.game-card__developer {
  font-size: 12px;
  color: var(--color-text-3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.game-card__actions {
  display: flex;
  gap: 0;
  flex-shrink: 0;
  margin-right: -8px; /* Offset Arco button default padding */
}

.game-card__actions .arco-btn {
  padding: 0 4px;
}

.game-card__actions .is-favorite {
  color: rgb(var(--danger-6));
}
</style>
