<template>
  <section class="media-card" :class="cardClass">
    <header class="media-card__header">
      <div class="media-card__heading">
        <span class="media-card__title">{{ title }}</span>
        <span v-if="count" class="media-card__count">{{ count }}</span>
      </div>
      <div class="media-card__toolbar">
        <slot name="toolbar" />
      </div>
    </header>
    <div class="media-card__body">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  count?: string
  cardClass?: string
}>()
</script>

<style scoped>
.media-card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--app-card-border);
  border-radius: 14px;
  background: color-mix(in srgb, var(--app-card-surface) 92%, transparent);
  overflow: hidden;
}

.media-card--cover {
  grid-column: span 4;
}

.media-card--banner {
  grid-column: span 4;
}

.media-card--screenshots {
  grid-column: span 4;
}

.media-card--logo {
  grid-column: span 7;
}

.media-card--video {
  grid-column: span 5;
}

.media-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px 0;
}

.media-card__heading {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.media-card__title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
}

.media-card__count {
  font-size: 12px;
  color: var(--color-text-3);
}

.media-card__toolbar {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.media-card__action {
  color: var(--color-text-2);
}

.media-card__body {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  gap: 8px;
  padding: 8px 12px 12px;
  min-width: 0;
  min-height: 0;
}

@media (max-width: 768px) {
  .media-card--cover,
  .media-card--banner,
  .media-card--screenshots,
  .media-card--logo,
  .media-card--video {
    grid-column: auto;
  }
}
</style>
