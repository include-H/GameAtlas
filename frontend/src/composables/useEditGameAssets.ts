import { type Ref } from 'vue'
import type {
  EditGameEditableScreenshot,
  EditGameEditableVideo,
  EditGameForm,
} from '@/composables/edit-game-form'
import { uploadAsset, type UploadedAssetResult } from '@/services/assets'
import type { FileItem } from '@arco-design/web-vue/es/upload/interfaces'
import { getHttpErrorMessage } from '@/utils/http-error'

type AlertType = 'success' | 'warning' | 'error'
type AssetType = 'cover' | 'banner' | 'screenshot' | 'video'

interface UploadResponseLike {
  success?: boolean
  data?: UploadedAssetResult
  error?: string
}

interface UseEditGameAssetsOptions {
  form: Ref<Pick<EditGameForm, 'cover_image' | 'banner_image' | 'screenshots' | 'preview_videos'>>
  gameId: Ref<number | undefined>
  showCoverSelector: Ref<boolean>
  showBannerSelector: Ref<boolean>
  showScreenshotSelector: Ref<boolean>
  showVideoSelector: Ref<boolean>
  isUploadingVideo: Ref<boolean>
  videoUploadProgress: Ref<number>
  videoUploadFileName: Ref<string>
  queueAssetDeletion: (type: AssetType, path: string, assetId?: number, assetUid?: string) => void
  createEditableScreenshot: (asset: UploadedAssetResult, index: number) => EditGameEditableScreenshot
  createEditableVideo: (asset: UploadedAssetResult) => EditGameEditableVideo
  addAlert: (message: string, type: AlertType) => void
}

const readUploadError = (response?: UploadResponseLike) => {
  return response?.error || '未知错误'
}

export const useEditGameAssets = (options: UseEditGameAssetsOptions) => {
  const appendPreviewVideo = (video: EditGameEditableVideo) => {
    options.form.value.preview_videos.push(video)
  }

  const handleCoverUploadSuccess = (fileItem: FileItem) => {
    const response = fileItem.response as UploadResponseLike | undefined
    if (response?.success && response.data?.path) {
      if (options.form.value.cover_image) {
        options.queueAssetDeletion('cover', options.form.value.cover_image)
      }
      options.form.value.cover_image = response.data.path
      options.showCoverSelector.value = false
      options.addAlert('封面上传成功', 'success')
      return
    }

    options.addAlert('上传失败：' + readUploadError(response), 'error')
  }

  const handleCoverUploadError = () => {
    options.addAlert('封面上传失败', 'error')
  }

  const handleBannerUploadSuccess = (fileItem: FileItem) => {
    const response = fileItem.response as UploadResponseLike | undefined
    if (response?.success && response.data?.path) {
      if (options.form.value.banner_image) {
        options.queueAssetDeletion('banner', options.form.value.banner_image)
      }
      options.form.value.banner_image = response.data.path
      options.showBannerSelector.value = false
      options.addAlert('横幅上传成功', 'success')
      return
    }

    options.addAlert('上传失败：' + readUploadError(response), 'error')
  }

  const handleBannerUploadError = () => {
    options.addAlert('横幅上传失败', 'error')
  }

  const handleScreenshotUploadSuccess = (fileItem: FileItem) => {
    const response = fileItem.response as UploadResponseLike | undefined
    if (response?.success && response.data?.path) {
      options.form.value.screenshots.push(
        options.createEditableScreenshot(response.data, options.form.value.screenshots.length),
      )
      options.showScreenshotSelector.value = false
      options.addAlert('截图上传成功', 'success')
      return
    }

    options.addAlert('上传失败：' + readUploadError(response), 'error')
  }

  const handleScreenshotUploadError = () => {
    options.addAlert('截图上传失败', 'error')
  }

  const openVideoSelector = () => {
    options.showVideoSelector.value = true
  }

  const handleVideoFileChange = async (event: Event) => {
    const input = event.target as HTMLInputElement
    const file = input.files?.[0]
    const gameId = options.gameId.value
    if (!file || !gameId) return

    options.isUploadingVideo.value = true
    options.videoUploadProgress.value = 0
    options.videoUploadFileName.value = file.name

    try {
      const uploaded = await uploadAsset('video', gameId, file, options.form.value.preview_videos.length, (percent) => {
        options.videoUploadProgress.value = percent
      })
      appendPreviewVideo(options.createEditableVideo(uploaded))
      options.videoUploadProgress.value = 100
      options.addAlert('预告片上传成功', 'success')
    } catch (error) {
      options.videoUploadProgress.value = 0
      options.addAlert('预告片上传失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      options.isUploadingVideo.value = false
      input.value = ''
    }
  }

  const removeCover = () => {
    const coverUrl = options.form.value.cover_image
    if (!coverUrl) return
    options.queueAssetDeletion('cover', coverUrl)
    options.form.value.cover_image = ''
  }

  const removeBanner = () => {
    const bannerUrl = options.form.value.banner_image
    if (!bannerUrl) return
    options.queueAssetDeletion('banner', bannerUrl)
    options.form.value.banner_image = ''
  }

  const removeScreenshot = (clientKey: string) => {
    const screenshot = options.form.value.screenshots.find((item) => item.client_key === clientKey)
    if (!screenshot) return
    options.queueAssetDeletion('screenshot', screenshot.path, screenshot.id, screenshot.asset_uid)
    options.form.value.screenshots = options.form.value.screenshots.filter((item) => item.client_key !== clientKey)
  }

  const removePreviewVideo = (assetUid?: string) => {
    const target = options.form.value.preview_videos.find((item) => item.asset_uid === assetUid)
    if (!target) return
    options.queueAssetDeletion('video', target.path, target.id, target.asset_uid)
    options.form.value.preview_videos = options.form.value.preview_videos.filter((item) => item.asset_uid !== assetUid)
  }

  const resetVideoUploadState = () => {
    options.showVideoSelector.value = false
    options.videoUploadProgress.value = 0
    options.videoUploadFileName.value = ''
    options.isUploadingVideo.value = false
  }

  return {
    handleCoverUploadSuccess,
    handleCoverUploadError,
    handleBannerUploadSuccess,
    handleBannerUploadError,
    handleScreenshotUploadSuccess,
    handleScreenshotUploadError,
    openVideoSelector,
    handleVideoFileChange,
    removeCover,
    removeBanner,
    removeScreenshot,
    removePreviewVideo,
    resetVideoUploadState,
  }
}
