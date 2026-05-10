import { type Ref } from 'vue'
import type {
  EditGameEditableCover,
  EditGameEditableLogo,
  EditGameEditableScreenshot,
  EditGameEditableVideo,
  EditGameForm,
} from '@/composables/edit-game-form'
import { uploadAsset, type UploadedAssetResult } from '@/services/assets'
import type { FileItem } from '@arco-design/web-vue/es/upload/interfaces'
import { getHttpErrorMessage } from '@/utils/http-error'

type AlertType = 'success' | 'warning' | 'error'
type AssetType = 'cover' | 'banner' | 'screenshot' | 'video' | 'logo'

interface UploadResponseLike {
  success?: boolean
  data?: UploadedAssetResult
  error?: string
}

interface UseEditGameAssetsOptions {
  form: Ref<Pick<EditGameForm, 'covers' | 'logo' | 'banner_image' | 'screenshots' | 'preview_videos'>>
  gameId: Ref<number | undefined>
  showCoverSelector: Ref<boolean>
  showBannerSelector: Ref<boolean>
  showScreenshotSelector: Ref<boolean>
  showVideoSelector: Ref<boolean>
  showLogoSelector: Ref<boolean>
  isUploadingVideo: Ref<boolean>
  videoUploadProgress: Ref<number>
  videoUploadFileName: Ref<string>
  queueAssetDeletion: (type: AssetType, path: string, assetId?: number, assetUid?: string) => void
  createEditableCover: (asset: UploadedAssetResult) => EditGameEditableCover
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
      const oldLogo = options.form.value.logo
      if (oldLogo) {
        options.queueAssetDeletion('logo', oldLogo.path, oldLogo.id, oldLogo.asset_uid)
      }
      options.form.value.logo = options.createEditableLogo(response.data)
      void options.onAssetPersisted?.()
      options.showLogoSelector.value = false
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
    if (response?.success && response.data?.path) {
      if (options.form.value.banner_image) {
        options.queueAssetDeletion('banner', options.form.value.banner_image)
      }
      options.form.value.banner_image = response.data.path
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
      await options.onAssetPersisted?.()
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

  const removeCover = (index: number) => {
    const cover = options.form.value.covers[index]
    if (!cover) return
    options.queueAssetDeletion('cover', cover.path, cover.id, cover.asset_uid)
    options.form.value.covers = options.form.value.covers.filter((_, i) => i !== index)
  }

  const setPrimaryCover = (index: number) => {
    if (index <= 0 || index >= options.form.value.covers.length) return
    const covers = [...options.form.value.covers]
    const [moved] = covers.splice(index, 1)
    covers.splice(0, 0, moved)
    options.form.value.covers = covers
  }

  const removeLogo = () => {
    const logo = options.form.value.logo
    if (!logo) return
    options.queueAssetDeletion('logo', logo.path, logo.id, logo.asset_uid)
    options.form.value.logo = null
  }

  const handleLogoPositionConfirm = (payload: { position_x: number; position_y: number; width_pct: number }) => {
    const logo = options.form.value.logo
    if (!logo) return
    logo.position_x = payload.position_x
    logo.position_y = payload.position_y
    logo.width_pct = payload.width_pct
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
    handleLogoUploadSuccess,
    handleLogoUploadError,
    handleBannerUploadSuccess,
    handleBannerUploadError,
    handleScreenshotUploadSuccess,
    handleScreenshotUploadError,
    openVideoSelector,
    handleVideoFileChange,
    removeCover,
    removeLogo,
    removeBanner,
    removeScreenshot,
    removePreviewVideo,
    setPrimaryCover,
    handleLogoPositionConfirm,
    resetVideoUploadState,
  }
}
