<template>
  <a-modal
    class="video-poster-picker-modal"
    :visible="visible"
    title="重选封面帧"
    :width="640"
    :mask-closable="false"
    :footer="false"
    @cancel="emit('cancel')"
  >
    <div class="video-poster-picker">
      <video
        :key="video?.path"
        ref="videoRef"
        class="video-poster-picker__player"
        :src="video?.path"
        controls
        preload="metadata"
      />
      <p class="video-poster-picker__hint">播放并暂停到想要的画面，或拖动进度条定位后点击截取</p>
      <a-button type="primary" long html-type="button" :disabled="!video" @click="handleCapture">
        截取当前帧
      </a-button>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { extractVideoFrameFromElement } from '@/utils/video-poster'

export interface VideoPosterTarget {
  path: string
  poster_path?: string | null
}

defineProps<{
  visible: boolean
  video: VideoPosterTarget | null
}>()

const emit = defineEmits<{
  confirm: [blob: Blob]
  cancel: []
}>()

const videoRef = ref<HTMLVideoElement | null>(null)

const handleCapture = async () => {
  const video = videoRef.value
  if (!video) return
  try {
    const blob = await extractVideoFrameFromElement(video)
    emit('confirm', blob)
  } catch {
    // 取帧失败时保持弹窗打开，提示用户重试。
    emit('cancel')
  }
}
</script>

<style scoped>
.video-poster-picker {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.video-poster-picker__player {
  width: 100%;
  max-height: 360px;
  border-radius: 8px;
  background: #000;
}

.video-poster-picker__hint {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-3);
}
</style>
