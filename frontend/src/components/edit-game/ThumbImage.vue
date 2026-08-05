<template>
  <a-image
    :src="thumbSrc"
    width="100%"
    height="100%"
    fit="contain"
    hide-footer
    :preview="preview"
  >
    <template #error>
      <img
        :src="src"
        :alt="alt"
        class="thumb-image__fallback"
        loading="lazy"
        decoding="async"
      />
    </template>
  </a-image>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getAssetThumbUrl } from '@/utils/asset-url'

const props = withDefaults(defineProps<{
  src: string
  alt?: string
  preview?: boolean
}>(), {
  alt: '',
  preview: false,
})

const thumbSrc = computed(() => getAssetThumbUrl(props.src))
</script>

<style scoped>
.thumb-image__fallback {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
}
</style>
