<template>
  <a-modal
    v-model:visible="visible"
    class="add-game-modal"
    title="添加游戏"
    :width="modalWidth"
    :footer="false"
    @cancel="handleCancel"
    :mask-closable="false"
  >
    <a-form ref="formRef" :model="form" :rules="rules" layout="vertical">
      <!-- 游戏标题 -->
      <a-form-item field="title" label="游戏标题">
        <a-input
          v-model="form.title"
          placeholder="请输入游戏标题"
          allow-clear
          @press-enter="handleSubmit"
        />
      </a-form-item>

      <a-form-item field="visibility" label="可见性">
        <a-radio-group v-model="form.visibility" type="button">
          <a-radio value="public">公开</a-radio>
          <a-radio value="private">私有</a-radio>
        </a-radio-group>
      </a-form-item>
    </a-form>

    <div class="add-game-modal__actions">
      <a-button class="app-text-action-btn" type="text" @click="handleCancel">
        取消
      </a-button>
      <a-button type="primary" :loading="isSubmitting" @click="handleSubmit">
        添加
      </a-button>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import type { FieldRule } from '@arco-design/web-vue'

interface FormState {
  title: string
  visibility: 'public' | 'private'
}

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'submit': [data: FormState]
}>()

const formRef = ref()
const isSubmitting = ref(false)
const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1280)
const form = ref<FormState>({
  title: '',
  visibility: 'public',
})

const rules: Record<string, FieldRule[]> = {
  title: [
    { required: true, message: '请输入游戏标题' }
  ]
}

const visible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value)
})

const modalWidth = computed(() => {
  if (viewportWidth.value <= 576) return 'calc(100vw - 24px)'
  if (viewportWidth.value <= 912) return 'min(600px, calc(100vw - 48px))'
  return 600
})

const syncViewportWidth = () => {
  viewportWidth.value = window.innerWidth
}

onMounted(() => {
  syncViewportWidth()
  window.addEventListener('resize', syncViewportWidth)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', syncViewportWidth)
})

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  isSubmitting.value = true

  emit('submit', {
    title: form.value.title,
    visibility: form.value.visibility,
  })

  // Reset form
  visible.value = false
  form.value.title = ''
  form.value.visibility = 'public'
  isSubmitting.value = false
}

const handleCancel = () => {
  visible.value = false
  form.value.title = ''
  form.value.visibility = 'public'
}
</script>

<style scoped>
:deep(.add-game-modal .arco-modal-body) {
  padding: 20px 24px 24px;
}

:deep(.arco-input-group) {
  display: flex;
  gap: 8px;
}

.add-game-modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

@media (max-width: 576px) {
  :deep(.add-game-modal .arco-modal-body) {
    padding: 16px;
  }

  .add-game-modal__actions {
    justify-content: stretch;
  }

  .add-game-modal__actions :deep(.arco-btn) {
    flex: 1;
  }
}
</style>
