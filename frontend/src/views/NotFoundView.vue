<template>
  <div class="not-found">
    <a-result status="404" title="404" sub-title="Page Not Found">
      <template #extra>
        <div class="not-found-content">
          <p class="not-found-message">
            The page you're looking for doesn't exist or has been moved.
          </p>
          <a-button type="primary" size="large" @click="goHome">
            Go Home
          </a-button>
        </div>
      </template>
    </a-result>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const goHome = () => {
  router.push('/')
}
</script>

<style scoped>
.not-found {
  min-height: 60vh;
  display: flex;
  align-items: center;
  justify-content: center;
}

.not-found-content {
  text-align: center;
}

.not-found-message {
  margin-bottom: 24px;
  color: var(--color-text-3);
}
</style>
