<template>
  <a-modal
    v-model:visible="visible"
    title="添加游戏"
    :width="600"
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
    </a-form>

    <div class="add-game-modal__actions">
      <a-button class="app-secondary-cta" type="secondary" @click="handleCancel">
        取消
      </a-button>
      <a-button class="app-primary-cta" type="primary" :loading="isSubmitting" @click="handleSubmit">
        添加
      </a-button>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { FieldRule } from '@arco-design/web-vue'

interface FormState {
  title: string
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
const form = ref<FormState>({
  title: ''
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

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  isSubmitting.value = true

  emit('submit', {
    title: form.value.title
  })

  // Reset form
  visible.value = false
  form.value.title = ''
  isSubmitting.value = false
}

const handleCancel = () => {
  visible.value = false
  form.value.title = ''
}
</script>

<style scoped>
:deep(.arco-input-group) {
  display: flex;
  gap: 8px;
}

.add-game-modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
