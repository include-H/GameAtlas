import { ref, type Ref } from 'vue'
import type {
  EditGameForm,
  EditGameEditableBanner,
  EditGameEditableCover,
  EditGameEditableLogo,
  EditGameEditableScreenshot,
  EditGameEditableVideo,
} from '@/utils/edit-game-form'

interface UseEditGameMediaStateOptions {
  form: Ref<Pick<EditGameForm, 'screenshots' | 'preview_videos' | 'covers' | 'banners' | 'logos'>>
}

const getEditableCoverKey = (cover: EditGameEditableCover) => {
  return cover.asset_uid || cover.path
}

const getEditableBannerKey = (banner: EditGameEditableBanner) => {
  return banner.asset_uid || banner.path
}

const getEditableVideoKey = (video: EditGameEditableVideo) => {
  return video.asset_uid || video.path
}

const getEditableLogoKey = (logo: EditGameEditableLogo) => {
  return logo.asset_uid || logo.path
}

const getEditableScreenshotKey = (screenshot: EditGameEditableScreenshot) => {
  return screenshot.client_key
}

const reorderByKey = <T>(
  items: T[],
  fromKey: string,
  toKey: string,
  getKey: (item: T) => string,
): T[] => {
  const fromIndex = items.findIndex((item) => getKey(item) === fromKey)
  const toIndex = items.findIndex((item) => getKey(item) === toKey)
  if (fromIndex === -1 || toIndex === -1 || fromIndex === toIndex) return items

  const next = [...items]
  const [moved] = next.splice(fromIndex, 1)
  next.splice(toIndex, 0, moved)
  return next
}

export const useEditGameMediaState = (options: UseEditGameMediaStateOptions) => {
  const draggedScreenshotKey = ref<string | null>(null)
  const dragOverScreenshotKey = ref<string | null>(null)
  const draggedCoverKey = ref<string | null>(null)
  const dragOverCoverKey = ref<string | null>(null)
  const draggedBannerKey = ref<string | null>(null)
  const dragOverBannerKey = ref<string | null>(null)
  const draggedLogoKey = ref<string | null>(null)
  const dragOverLogoKey = ref<string | null>(null)

  const reorderEditableVideos = (targetKey: string, direction: -1 | 1) => {
    const videos = [...options.form.value.preview_videos]
    const index = videos.findIndex((item) => getEditableVideoKey(item) === targetKey)
    if (index === -1) return

    const nextIndex = index + direction
    if (nextIndex < 0 || nextIndex >= videos.length) return

    const [moved] = videos.splice(index, 1)
    videos.splice(nextIndex, 0, moved)
    options.form.value.preview_videos = videos
  }

  const handleScreenshotDragStart = (clientKey: string) => {
    draggedScreenshotKey.value = clientKey
    dragOverScreenshotKey.value = clientKey
  }

  const handleScreenshotDragEnter = (clientKey: string) => {
    if (!draggedScreenshotKey.value || draggedScreenshotKey.value === clientKey) return
    dragOverScreenshotKey.value = clientKey
  }

  const handleScreenshotDrop = (clientKey: string) => {
    if (!draggedScreenshotKey.value) return
    options.form.value.screenshots = reorderByKey(
      options.form.value.screenshots,
      draggedScreenshotKey.value,
      clientKey,
      getEditableScreenshotKey,
    )
    draggedScreenshotKey.value = null
    dragOverScreenshotKey.value = null
  }

  const handleScreenshotDragEnd = () => {
    draggedScreenshotKey.value = null
    dragOverScreenshotKey.value = null
  }

  const handleCoverDragStart = (key: string) => {
    draggedCoverKey.value = key
    dragOverCoverKey.value = key
  }

  const handleCoverDragEnter = (key: string) => {
    if (!draggedCoverKey.value || draggedCoverKey.value === key) return
    dragOverCoverKey.value = key
  }

  const handleCoverDrop = (key: string) => {
    if (!draggedCoverKey.value) return
    options.form.value.covers = reorderByKey(
      options.form.value.covers,
      draggedCoverKey.value,
      key,
      getEditableCoverKey,
    )
    draggedCoverKey.value = null
    dragOverCoverKey.value = null
  }

  const handleCoverDragEnd = () => {
    draggedCoverKey.value = null
    dragOverCoverKey.value = null
  }

  const handleBannerDragStart = (key: string) => {
    draggedBannerKey.value = key
    dragOverBannerKey.value = key
  }

  const handleBannerDragEnter = (key: string) => {
    if (!draggedBannerKey.value || draggedBannerKey.value === key) return
    dragOverBannerKey.value = key
  }

  const handleBannerDrop = (key: string) => {
    if (!draggedBannerKey.value) return
    options.form.value.banners = reorderByKey(
      options.form.value.banners,
      draggedBannerKey.value,
      key,
      getEditableBannerKey,
    )
    draggedBannerKey.value = null
    dragOverBannerKey.value = null
  }

  const handleBannerDragEnd = () => {
    draggedBannerKey.value = null
    dragOverBannerKey.value = null
  }

  const handleLogoDragStart = (key: string) => {
    draggedLogoKey.value = key
    dragOverLogoKey.value = key
  }

  const handleLogoDragEnter = (key: string) => {
    if (!draggedLogoKey.value || draggedLogoKey.value === key) return
    dragOverLogoKey.value = key
  }

  const handleLogoDrop = (key: string) => {
    if (!draggedLogoKey.value) return
    options.form.value.logos = reorderByKey(
      options.form.value.logos,
      draggedLogoKey.value,
      key,
      getEditableLogoKey,
    )
    draggedLogoKey.value = null
    dragOverLogoKey.value = null
  }

  const handleLogoDragEnd = () => {
    draggedLogoKey.value = null
    dragOverLogoKey.value = null
  }

  return {
    draggedScreenshotKey,
    dragOverScreenshotKey,
    draggedCoverKey,
    dragOverCoverKey,
    draggedBannerKey,
    dragOverBannerKey,
    draggedLogoKey,
    dragOverLogoKey,
    reorderEditableVideos,
    handleScreenshotDragStart,
    handleScreenshotDragEnter,
    handleScreenshotDrop,
    handleScreenshotDragEnd,
    handleCoverDragStart,
    handleCoverDragEnter,
    handleCoverDrop,
    handleCoverDragEnd,
    handleBannerDragStart,
    handleBannerDragEnter,
    handleBannerDrop,
    handleBannerDragEnd,
    handleLogoDragStart,
    handleLogoDragEnter,
    handleLogoDrop,
    handleLogoDragEnd,
  }
}
