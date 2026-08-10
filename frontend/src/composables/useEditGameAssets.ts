import { type Ref } from 'vue'
import type {
  EditGameEditableBanner,
  EditGameEditableCover,
  EditGameEditableLogo,
  EditGameEditableScreenshot,
  EditGameEditableVideo,
  EditGameForm,
  LogoPositionChange,
} from '@/utils/edit-game-form'
import { uploadAsset, type UploadedAssetResult } from '@/services/assets'
import type { FileItem } from '@arco-design/web-vue/es/upload/interfaces'
import { getHttpErrorMessage } from '@/utils/http-error'
import { extractVideoPoster } from '@/utils/video-poster'
import { createRequestGeneration } from '@/utils/request-generation'

type AlertType = 'success' | 'warning' | 'error'

interface UploadResponseLike {
  success?: boolean
  data?: UploadedAssetResult
  error?: string
}

interface UseEditGameAssetsOptions {
  form: Ref<Pick<EditGameForm, 'covers' | 'logos' | 'logo_visible' | 'banners' | 'screenshots' | 'preview_videos'>>
  gameId: Ref<number | undefined>
  showCoverSelector: Ref<boolean>
  showBannerSelector: Ref<boolean>
  showScreenshotSelector: Ref<boolean>
  isUploadingVideo: Ref<boolean>
  videoUploadProgress: Ref<number>
  videoUploadFileName: Ref<string>
  createEditableCover: (asset: UploadedAssetResult) => EditGameEditableCover
  createEditableBanner: (asset: UploadedAssetResult) => EditGameEditableBanner
  createEditableLogo: (asset: UploadedAssetResult) => EditGameEditableLogo
  createEditableScreenshot: (asset: UploadedAssetResult, index: number) => EditGameEditableScreenshot
  createEditableVideo: (asset: UploadedAssetResult) => EditGameEditableVideo
  addAlert: (message: string, type: AlertType) => void
  onAssetPersisted?: () => Promise<void> | void
}

const readUploadError = (response?: UploadResponseLike) => {
  return response?.error || '未知错误'
}

export const useEditGameAssets = (options: UseEditGameAssetsOptions) => {
  const videoUploadRequests = createRequestGeneration()
  const appendPreviewVideo = (video: EditGameEditableVideo) => {
    options.form.value.preview_videos.push(video)
  }

  const handleCoverUploadSuccess = (fileItem: FileItem) => {
    const response = fileItem.response as UploadResponseLike | undefined
    if (response?.success && response.data) {
      options.form.value.covers.push(options.createEditableCover(response.data))
      void options.onAssetPersisted?.()
      options.showCoverSelector.value = false
      options.addAlert('封面上传成功', 'success')
      return
    }

    options.addAlert('上传失败：' + readUploadError(response), 'error')
  }

  const handleCoverUploadError = () => {
    options.addAlert('封面上传失败', 'error')
  }

  const handleLogoUploadSuccess = (fileItem: FileItem) => {
    const response = fileItem.response as UploadResponseLike | undefined
    if (response?.success && response.data) {
      options.form.value.logos.push(options.createEditableLogo(response.data))
      void options.onAssetPersisted?.()
      options.addAlert('Logo 上传成功', 'success')
      return
    }

    options.addAlert('上传失败：' + readUploadError(response), 'error')
  }

  const handleLogoUploadError = () => {
    options.addAlert('Logo 上传失败', 'error')
  }

  const handleBannerUploadSuccess = (fileItem: FileItem) => {
    const response = fileItem.response as UploadResponseLike | undefined
    if (response?.success && response.data) {
      options.form.value.banners.push(options.createEditableBanner(response.data))
      void options.onAssetPersisted?.()
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
      void options.onAssetPersisted?.()
      options.showScreenshotSelector.value = false
      options.addAlert('截图上传成功', 'success')
      return
    }

    options.addAlert('上传失败：' + readUploadError(response), 'error')
  }

  const handleScreenshotUploadError = () => {
    options.addAlert('截图上传失败', 'error')
  }

  const handleVideoFileChange = async (event: Event) => {
    const input = event.target as HTMLInputElement
    const files = input.files
    const gameId = options.gameId.value
    if (!files || files.length === 0 || !gameId || options.isUploadingVideo.value) return

    const request = videoUploadRequests.begin()
    options.isUploadingVideo.value = true

    try {
      for (let i = 0; i < files.length; i++) {
        const file = files[i]
        options.videoUploadProgress.value = 0
        options.videoUploadFileName.value = file.name

        const uploaded = await uploadAsset('video', gameId, file, (percent) => {
          if (request.isCurrent() && options.gameId.value === gameId) {
            options.videoUploadProgress.value = percent
          }
        })
        if (!request.isCurrent() || options.gameId.value !== gameId) return
        const video = options.createEditableVideo(uploaded)
        if (file.type.startsWith('video/')) {
          try {
            const posterBlob = await extractVideoPoster(file)
            const posterFile = new File([posterBlob], `poster-${Date.now()}.jpg`, {
              type: 'image/jpeg',
            })
            const poster = await uploadAsset('poster', gameId, posterFile)
            if (!request.isCurrent() || options.gameId.value !== gameId) return
            video.poster_path = poster.path
          } catch {
            // 封面帧抽取是增强能力，失败时预告片仍可正常使用。
          }
        }
        if (!request.isCurrent() || options.gameId.value !== gameId) return
        appendPreviewVideo(video)
        await options.onAssetPersisted?.()
      }
      options.videoUploadProgress.value = 100
      options.addAlert(files.length > 1 ? `${files.length} 个预告片上传成功` : '预告片上传成功', 'success')
    } catch (error) {
      options.videoUploadProgress.value = 0
      options.addAlert('预告片上传失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      if (request.isCurrent()) {
        options.isUploadingVideo.value = false
      }
      input.value = ''
    }
  }

  const removeCover = (index: number) => {
    const cover = options.form.value.covers[index]
    if (!cover) return
    options.form.value.covers = options.form.value.covers.filter((_, i) => i !== index)
  }

  const removeLogo = (index: number) => {
    const logo = options.form.value.logos[index]
    if (!logo) return
    options.form.value.logos = options.form.value.logos.filter((_, i) => i !== index)
  }

  const applyLogoPositionChange = (key: string, payload: Omit<LogoPositionChange, 'key'>) => {
    const logo = options.form.value.logos.find(
      (item) => (item.asset_uid || item.path) === key,
    )
    if (!logo) return
    logo.position_x = payload.position_x
    logo.position_y = payload.position_y
    logo.width_pct = payload.width_pct
    options.form.value.logo_visible = payload.logo_visible
  }

  const handleLogoPositionChange = (change: LogoPositionChange) => {
    applyLogoPositionChange(change.key, change)
  }

  const removeBanner = (index: number) => {
    const banner = options.form.value.banners[index]
    if (!banner) return
    options.form.value.banners = options.form.value.banners.filter((_, i) => i !== index)
  }

  const removeScreenshot = (clientKey: string) => {
    const screenshot = options.form.value.screenshots.find((item) => item.client_key === clientKey)
    if (!screenshot) return
    options.form.value.screenshots = options.form.value.screenshots.filter((item) => item.client_key !== clientKey)
  }

  const removePreviewVideo = (assetUid?: string) => {
    const target = options.form.value.preview_videos.find((item) => item.asset_uid === assetUid)
    if (!target) return
    options.form.value.preview_videos = options.form.value.preview_videos.filter((item) => item.asset_uid !== assetUid)
  }

  const resetVideoUploadState = () => {
    videoUploadRequests.invalidate()
    options.videoUploadProgress.value = 0
    options.videoUploadFileName.value = ''
    options.isUploadingVideo.value = false
  }

  return {
    handleCoverUploadSuccess,
    handleCoverUploadError,
    handleLogoUploadSuccess,
    handleLogoUploadError,
    handleBannerUploadSuccess,
    handleBannerUploadError,
    handleScreenshotUploadSuccess,
    handleScreenshotUploadError,
    handleVideoFileChange,
    removeCover,
    removeLogo,
    removeBanner,
    removeScreenshot,
    removePreviewVideo,
    handleLogoPositionChange,
    resetVideoUploadState,
  }
}
