import { computed, type Ref } from 'vue'
import { buildAssetUploadUrl } from '@/services/api-url'

interface UseEditGameUploadUrlsOptions {
  gameId: Ref<number | undefined>
}

export const useEditGameUploadUrls = (options: UseEditGameUploadUrlsOptions) => {
  const uploadAction = computed(() => {
    return buildAssetUploadUrl('cover')
  })

  const uploadData = computed(() => ({
    game_id: String(options.gameId.value || ''),
  }))

  const bannerUploadAction = computed(() => {
    return buildAssetUploadUrl('banner')
  })

  const bannerUploadData = computed(() => ({
    game_id: String(options.gameId.value || ''),
  }))

  const screenshotUploadAction = computed(() => {
    return buildAssetUploadUrl('screenshot')
  })

  const screenshotUploadData = computed(() => ({
    game_id: String(options.gameId.value || ''),
  }))

  const logoUploadAction = computed(() => {
    return buildAssetUploadUrl('logo')
  })

  const logoUploadData = computed(() => ({
    game_id: String(options.gameId.value || ''),
  }))

  const uploadHeaders = computed(() => ({}))

  return {
    uploadAction,
    uploadData,
    bannerUploadAction,
    bannerUploadData,
    screenshotUploadAction,
    screenshotUploadData,
    logoUploadAction,
    logoUploadData,
    uploadHeaders,
  }
}
