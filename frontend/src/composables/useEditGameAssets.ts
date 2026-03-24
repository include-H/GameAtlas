import { type Ref } from 'vue'
import { uploadAsset, type UploadedAssetResult } from '@/services/assets'

type AlertType = 'success' | 'warning' | 'error'
type AssetType = 'cover' | 'banner' | 'screenshot' | 'video'

interface AssetEditableScreenshot {
  id?: number
  asset_uid?: string
  path: string
  client_key: string
}

interface AssetEditableVideo {
  id?: number
  asset_uid?: string
  path: string
}

interface AssetFormBridge {
  cover_image: string
  banner_image: string
  screenshots: AssetEditableScreenshot[]
  preview_videos: AssetEditableVideo[]
  primary_preview_video_uid: string
}

interface UploadResponseLike {
  success?: boolean
  data?: UploadedAssetResult
  error?: string
}

interface UploadSuccessFileItem {
  response?: UploadResponseLike
}

interface UseEditGameAssetsOptions {
  form: Ref<AssetFormBridge>
  gameId: Ref<number | undefined>
  showCoverSelector: Ref<boolean>
  showBannerSelector: Ref<boolean>
  showScreenshotSelector: Ref<boolean>
  showVideoSelector: Ref<boolean>
  isUploadingVideo: Ref<boolean>
  videoUploadProgress: Ref<number>
  videoUploadFileName: Ref<string>
  queueAssetDeletion: (type: AssetType, path: string, assetId?: number, assetUid?: string) => void
  createEditableScreenshot: (asset: UploadedAssetResult, index: number) => AssetEditableScreenshot
  createEditableVideo: (asset: UploadedAssetResult) => AssetEditableVideo
  addAlert: (message: string, type: AlertType) => void
}

const readUploadError = (response?: UploadResponseLike) => {
  return response?.error || '未知错误'
}

const readErrorMessage = (error: unknown, fallback = '未知错误') => {
  if (error instanceof Error && error.message) return error.message
  return fallback
}

const parseUploadSuccessFileItem = (value: unknown): UploadSuccessFileItem => {
  if (typeof value !== 'object' || value === null) {
    return {}
  }
  return value as UploadSuccessFileItem
}

export const useEditGameAssets = (options: UseEditGameAssetsOptions) => {
  const appendPreviewVideo = (video: AssetEditableVideo) => {
    options.form.value.preview_videos.push(video)
    if (!options.form.value.primary_preview_video_uid && video.asset_uid) {
      options.form.value.primary_preview_video_uid = video.asset_uid
    }
  }

  const handleCoverUploadSuccess = (fileItem: unknown) => {
    const parsedFileItem = parseUploadSuccessFileItem(fileItem)
    const response = parsedFileItem.response
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

  const handleBannerUploadSuccess = (fileItem: unknown) => {
    const parsedFileItem = parseUploadSuccessFileItem(fileItem)
    const response = parsedFileItem.response
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

  const handleScreenshotUploadSuccess = (fileItem: unknown) => {
    const parsedFileItem = parseUploadSuccessFileItem(fileItem)
    const response = parsedFileItem.response
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
      options.addAlert('预告片上传失败：' + readErrorMessage(error), 'error')
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
    if (options.form.value.primary_preview_video_uid === assetUid) {
      options.form.value.primary_preview_video_uid = options.form.value.preview_videos[0]?.asset_uid || ''
    }
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
