<template>
  <a-row :gutter="[16, 16]" class="stat-overview">
    <a-col :xs="12" :lg="6">
      <stat-card
        title="游戏总数"
        :value="totalGames"
        icon="mdi-gamepad-variant"
        color="var(--color-primary-6)"
        :height="104"
        @click="router.push('/games')"
      />
    </a-col>
    <a-col :xs="12" :lg="6">
      <stat-card
        title="总下载"
        :value="totalDownloads"
        icon="mdi-download"
        color="var(--color-success-6)"
        :height="104"
        @click="router.push('/games?sort=downloads&order=desc')"
      />
    </a-col>
    <a-col :xs="12" :lg="6">
      <stat-card
        title="收藏"
        :value="favoriteCount"
        icon="mdi-heart"
        color="var(--color-danger-6)"
        :height="104"
        @click="router.push('/games?favorite=true')"
      />
    </a-col>
    <a-col :xs="12" :lg="6">
      <stat-card
        title="待处理"
        :value="pendingReviews"
        icon="mdi-clock"
        color="var(--color-warning-6)"
        :height="104"
        @click="router.push('/games/pending')"
      />
    </a-col>
  </a-row>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import StatCard from '@/components/StatCard.vue'

defineProps<{
  totalGames: number
  totalDownloads: number
  favoriteCount: number
  pendingReviews: number
}>()

const router = useRouter()
</script>

<style scoped>
.stat-overview {
  margin-bottom: 24px;
}
</style>
