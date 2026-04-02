import { ref, type Ref } from 'vue'
import type { EditGameForm, EditGameEditableVideo } from '@/composables/edit-game-form'

interface UseEditGameMediaStateOptions {
  form: Ref<Pick<EditGameForm, 'screenshots' | 'preview_videos'>>
}

const getEditableVideoKey = (video: EditGameEditableVideo) => {
  return video.asset_uid || video.path
}

export const useEditGameMediaState = (options: UseEditGameMediaStateOptions) => {
  const draggedScreenshotKey = ref<string | null>(null)
  const dragOverScreenshotKey = ref<string | null>(null)

  const reorderEditableVideos = (targetKey: string, direction: -1 | 1) => {
    const videos = [...options.form.value.preview_videos]
    const index = videos.findIndex((item) => getEditableVideoKey(item) === targetKey)
    if (index === -1) return

    const nextIndex = index + direction
    if (nextIndex < 0 || nextIndex >= videos.length) return

    const [moved] = videos.splice(index, 1)
    videos.splice(nextIndex, 0, moved)
    options.form.value.preview_videos = videos.map((item, order) => ({
      ...item,
      sort_order: order,
    }))
  }

  const reorderEditableScreenshots = (fromKey: string, toKey: string) => {
    const screenshots = [...options.form.value.screenshots]
    const fromIndex = screenshots.findIndex((item) => item.client_key === fromKey)
    const toIndex = screenshots.findIndex((item) => item.client_key === toKey)
    if (fromIndex === -1 || toIndex === -1 || fromIndex === toIndex) return

    const [moved] = screenshots.splice(fromIndex, 1)
    screenshots.splice(toIndex, 0, moved)
    options.form.value.screenshots = screenshots.map((item, index) => ({
      ...item,
      sort_order: index,
    }))
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
    reorderEditableScreenshots(draggedScreenshotKey.value, clientKey)
    draggedScreenshotKey.value = null
    dragOverScreenshotKey.value = null
  }

  const handleScreenshotDragEnd = () => {
    draggedScreenshotKey.value = null
    dragOverScreenshotKey.value = null
  }

  return {
    draggedScreenshotKey,
    dragOverScreenshotKey,
    reorderEditableVideos,
    handleScreenshotDragStart,
    handleScreenshotDragEnter,
    handleScreenshotDrop,
    handleScreenshotDragEnd,
  }
}
