<template>
  <div class="not-found">
    <a-result status="404" title="404" sub-title="页面不存在">
      <template #extra>
        <div class="not-found-content">
          <p class="not-found-message">
            你访问的页面不存在，或者已经被移动。
          </p>
          <a-button type="primary" size="large" @click="goHome">
            返回首页
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

@media (max-width: 576px) {
  .not-found {
    min-height: 50vh;
    padding: 12px 0;
  }

  .not-found-message {
    margin-bottom: 16px;
    font-size: 13px;
  }
}
</style>
